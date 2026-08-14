package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrLogNotFound is returned by GetLog when no log carries the requested slug.
var ErrLogNotFound = errors.New("log not found")

// Log is one row of the log table as read from the database. Pointer fields are
// the nullable columns; a nil pointer is a NULL column, distinct from an empty
// string. Body is empty on rows read through the index (it is never selected
// there) and populated on rows read through GetLog.
type Log struct {
	Slug        string
	Title       string
	Subtitle    *string
	Description *string
	CoverImage  *string
	ReadingTime *int
	Tags        []string
	Body        string
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LogQuery selects and pages a slice of the log index. An empty State matches
// every state; an empty Tag applies no tag filter.
type LogQuery struct {
	State  string
	Tag    string
	Limit  int
	Offset int
}

const logSummaryColumns = "slug, title, subtitle, description, cover_image, reading_time, tags, state, created_at, updated_at"

// ListLogs returns the total number of logs matching the state and tag filters
// (ignoring the page window) alongside one page of summaries ordered newest
// first. Summaries carry no body.
func (db *DB) ListLogs(ctx context.Context, q LogQuery) (total int, logs []Log, err error) {
	where, args := logFilter(q.State, q.Tag)

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM log "+where, args...).Scan(&total); err != nil {
		return 0, nil, fmt.Errorf("count logs: %w", err)
	}

	query := "SELECT " + logSummaryColumns + " FROM log " + where +
		" ORDER BY created_at DESC, slug ASC LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, query, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return 0, nil, fmt.Errorf("list logs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			log     Log
			tagsRaw string
			created string
			updated string
		)
		err := rows.Scan(
			&log.Slug, &log.Title, &log.Subtitle, &log.Description, &log.CoverImage,
			&log.ReadingTime, &tagsRaw, &log.State, &created, &updated,
		)
		if err != nil {
			return 0, nil, fmt.Errorf("scan log summary: %w", err)
		}
		if err := decodeLog(&log, tagsRaw, created, updated); err != nil {
			return 0, nil, err
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("list logs: %w", err)
	}
	return total, logs, nil
}

// GetLog returns a single log by slug in any state, including its full body. It
// reports ErrLogNotFound when no such slug exists.
func (db *DB) GetLog(ctx context.Context, slug string) (Log, error) {
	query := "SELECT " + logSummaryColumns + ", body FROM log WHERE slug = ?"

	var (
		log     Log
		tagsRaw string
		created string
		updated string
	)
	err := db.QueryRowContext(ctx, query, slug).Scan(
		&log.Slug, &log.Title, &log.Subtitle, &log.Description, &log.CoverImage,
		&log.ReadingTime, &tagsRaw, &log.State, &created, &updated, &log.Body,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Log{}, ErrLogNotFound
	}
	if err != nil {
		return Log{}, fmt.Errorf("get log %q: %w", slug, err)
	}
	if err := decodeLog(&log, tagsRaw, created, updated); err != nil {
		return Log{}, err
	}
	return log, nil
}

// logFilter builds the shared WHERE clause and its arguments for both the count
// and the page query, so the two never drift apart.
func logFilter(state, tag string) (string, []any) {
	where := "WHERE 1 = 1"
	var args []any
	if state != "" {
		where += " AND state = ?"
		args = append(args, state)
	}
	if tag != "" {
		where += " AND EXISTS (SELECT 1 FROM json_each(log.tags) WHERE value = ?)"
		args = append(args, tag)
	}
	return where, args
}

// decodeLog fills in the fields that don't map one-to-one to a column scan: the
// JSON-encoded tags array and the two text timestamps.
func decodeLog(log *Log, tagsRaw, created, updated string) error {
	if err := json.Unmarshal([]byte(tagsRaw), &log.Tags); err != nil {
		return fmt.Errorf("decode tags for %q: %w", log.Slug, err)
	}
	var err error
	if log.CreatedAt, err = parseTimestamp(created); err != nil {
		return fmt.Errorf("parse created_at for %q: %w", log.Slug, err)
	}
	if log.UpdatedAt, err = parseTimestamp(updated); err != nil {
		return fmt.Errorf("parse updated_at for %q: %w", log.Slug, err)
	}
	return nil
}

func parseTimestamp(raw string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised timestamp %q", raw)
}
