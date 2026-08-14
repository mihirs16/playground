package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrIntegrationNotFound is returned by LatestIntegration when a source has no
// stored row yet — the state before its first successful poll.
var ErrIntegrationNotFound = errors.New("integration not found")

// Integration is the latest polled state of one third-party source. Data is the
// raw JSON body verbatim (empty when the row carried none); ETag is the
// validator to send back as If-None-Match on the next poll; FetchedAt is when
// the row was written, which the read surface serves as last-known-good.
type Integration struct {
	Source    string
	Data      string
	ETag      string
	FetchedAt time.Time
}

// LatestIntegration returns the most recent row for a source, reporting
// ErrIntegrationNotFound when the source has never been polled successfully.
func (db *DB) LatestIntegration(ctx context.Context, source string) (Integration, error) {
	row := db.QueryRowContext(ctx,
		"SELECT source, data, etag, fetched_at FROM integration WHERE source = ? ORDER BY id DESC LIMIT 1",
		source,
	)

	var (
		rec     Integration
		data    sql.NullString
		etag    sql.NullString
		fetched string
	)
	err := row.Scan(&rec.Source, &data, &etag, &fetched)
	if errors.Is(err, sql.ErrNoRows) {
		return Integration{}, ErrIntegrationNotFound
	}
	if err != nil {
		return Integration{}, fmt.Errorf("latest integration %q: %w", source, err)
	}

	rec.Data = data.String
	rec.ETag = etag.String
	parsed, err := parseTimestamp(fetched)
	if err != nil {
		return Integration{}, fmt.Errorf("parse fetched_at for %q: %w", source, err)
	}
	rec.FetchedAt = parsed
	return rec, nil
}

// AppendIntegration inserts a new timestamped row for a source. It is only ever
// called when the polled state changed, so the timeseries holds one row per
// distinct state; FetchedAt is stamped now when the caller leaves it zero.
func (db *DB) AppendIntegration(ctx context.Context, rec Integration) (Integration, error) {
	if rec.FetchedAt.IsZero() {
		rec.FetchedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(ctx,
		"INSERT INTO integration (source, data, etag, fetched_at) VALUES (?, ?, ?, ?)",
		rec.Source, nullString(rec.Data), nullString(rec.ETag),
		rec.FetchedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Integration{}, fmt.Errorf("append integration %q: %w", rec.Source, err)
	}
	return rec, nil
}

// ReapExpiredPendingMedia deletes pending media reservations whose upload window
// closed before now, so an abandoned reservation never lingers holding its key.
// Available records and reservations still in their window are left untouched.
// It returns how many rows were reaped.
func (db *DB) ReapExpiredPendingMedia(ctx context.Context, now time.Time) (int64, error) {
	result, err := db.ExecContext(ctx,
		"DELETE FROM media WHERE state = 'pending' AND expires_at IS NOT NULL AND julianday(expires_at) < julianday(?)",
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("reap expired pending media: %w", err)
	}
	reaped, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("reap expired pending media: %w", err)
	}
	return reaped, nil
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
