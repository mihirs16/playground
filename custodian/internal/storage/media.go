package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrMediaNotFound is returned by GetMedia, SetMediaAvailable, and DeleteMedia
// when no media record carries the requested key.
var ErrMediaNotFound = errors.New("media not found")

// ErrMediaKeyTaken is returned by CreateMedia when the key is already reserved,
// so a reservation never silently overwrites existing bytes.
var ErrMediaKeyTaken = errors.New("media key already taken")

// Media is one row of the media table. A pending record has a reservation but no
// confirmed bytes behind it; an available record has been HEAD-checked against
// S3. ExpiresAt is the presigned upload window and may be NULL once irrelevant.
type Media struct {
	Key         string
	State       string
	ContentType string
	URL         string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

// MediaQuery selects and pages a slice of the media index. An empty Search
// applies no filter; otherwise it matches keys containing the term.
type MediaQuery struct {
	Search string
	Limit  int
	Offset int
}

const mediaColumns = "key, state, content_type, url, created_at, expires_at"

// CreateMedia inserts a media record and returns it with created_at stamped now.
// The caller owns every other field, including state. A key already reserved
// reports ErrMediaKeyTaken rather than overwriting the existing reservation.
func (db *DB) CreateMedia(ctx context.Context, media Media) (Media, error) {
	media.CreatedAt = time.Now().UTC()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Media{}, fmt.Errorf("create media %q: %w", media.Key, err)
	}
	defer tx.Rollback()

	taken, err := mediaKeyTaken(ctx, tx, media.Key)
	if err != nil {
		return Media{}, err
	}
	if taken {
		return Media{}, ErrMediaKeyTaken
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO media ("+mediaColumns+") VALUES (?, ?, ?, ?, ?, ?)",
		media.Key, media.State, media.ContentType, media.URL,
		media.CreatedAt.Format(time.RFC3339Nano), expiresArg(media.ExpiresAt),
	)
	if err != nil {
		return Media{}, fmt.Errorf("insert media %q: %w", media.Key, err)
	}
	if err := tx.Commit(); err != nil {
		return Media{}, fmt.Errorf("create media %q: %w", media.Key, err)
	}
	return media, nil
}

// GetMedia returns a single media record by key, reporting ErrMediaNotFound when
// no such key exists.
func (db *DB) GetMedia(ctx context.Context, key string) (Media, error) {
	row := db.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM media WHERE key = ?", key)
	media, err := scanMedia(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Media{}, ErrMediaNotFound
	}
	if err != nil {
		return Media{}, fmt.Errorf("get media %q: %w", key, err)
	}
	return media, nil
}

// ListMedia returns the total number of records matching the search (ignoring
// the page window) alongside one page ordered newest first.
func (db *DB) ListMedia(ctx context.Context, q MediaQuery) (total int, media []Media, err error) {
	where, args := mediaFilter(q.Search)

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media "+where, args...).Scan(&total); err != nil {
		return 0, nil, fmt.Errorf("count media: %w", err)
	}

	query := "SELECT " + mediaColumns + " FROM media " + where +
		" ORDER BY created_at DESC, key ASC LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, query, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return 0, nil, fmt.Errorf("list media: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		record, err := scanMedia(rows)
		if err != nil {
			return 0, nil, fmt.Errorf("scan media: %w", err)
		}
		media = append(media, record)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("list media: %w", err)
	}
	return total, media, nil
}

// SetMediaAvailable flips a record to available and returns it, reporting
// ErrMediaNotFound when no such key exists. It is the only path to available, so
// every available record has had its bytes HEAD-confirmed by the caller first.
func (db *DB) SetMediaAvailable(ctx context.Context, key string) (Media, error) {
	result, err := db.ExecContext(ctx, "UPDATE media SET state = 'available' WHERE key = ?", key)
	if err != nil {
		return Media{}, fmt.Errorf("confirm media %q: %w", key, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Media{}, fmt.Errorf("confirm media %q: %w", key, err)
	}
	if affected == 0 {
		return Media{}, ErrMediaNotFound
	}
	return db.GetMedia(ctx, key)
}

// DeleteMedia removes the record at key, reporting ErrMediaNotFound when there
// is none. It deletes only custodian's record; the handler deletes the object
// bytes separately through the object store.
func (db *DB) DeleteMedia(ctx context.Context, key string) error {
	result, err := db.ExecContext(ctx, "DELETE FROM media WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("delete media %q: %w", key, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete media %q: %w", key, err)
	}
	if affected == 0 {
		return ErrMediaNotFound
	}
	return nil
}

func mediaKeyTaken(ctx context.Context, tx *sql.Tx, key string) (bool, error) {
	var one int
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM media WHERE key = ?", key).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check media key %q: %w", key, err)
	}
	return true, nil
}

func mediaFilter(search string) (string, []any) {
	if search == "" {
		return "", nil
	}
	return "WHERE key LIKE ?", []any{"%" + search + "%"}
}

// rowScanner is the shared surface of *sql.Row and *sql.Rows, so scanMedia reads
// a record the same way whether it came from a single lookup or a page.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanMedia(row rowScanner) (Media, error) {
	var (
		media   Media
		created string
		expires sql.NullString
	)
	if err := row.Scan(&media.Key, &media.State, &media.ContentType, &media.URL, &created, &expires); err != nil {
		return Media{}, err
	}
	parsedCreated, err := parseTimestamp(created)
	if err != nil {
		return Media{}, fmt.Errorf("parse created_at for %q: %w", media.Key, err)
	}
	media.CreatedAt = parsedCreated
	if expires.Valid {
		parsedExpires, err := parseTimestamp(expires.String)
		if err != nil {
			return Media{}, fmt.Errorf("parse expires_at for %q: %w", media.Key, err)
		}
		media.ExpiresAt = &parsedExpires
	}
	return media, nil
}

func expiresArg(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}
