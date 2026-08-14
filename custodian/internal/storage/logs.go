package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrLogNotFound is returned by GetLog, UpdateLog, and DeleteLog when no log
// carries the requested slug.
var ErrLogNotFound = errors.New("log not found")

// ErrSlugConflict is returned by CreateLog and UpdateLog when the target slug is
// already taken by another log.
var ErrSlugConflict = errors.New("slug already exists")

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

const logColumns = "slug, title, subtitle, description, cover_image, reading_time, tags, body, state, created_at, updated_at"

// CreateLog inserts a new log and returns it with its timestamps filled in. Both
// created_at and updated_at are stamped now; the caller owns every other field,
// including state. A slug already in use reports ErrSlugConflict.
func (db *DB) CreateLog(ctx context.Context, log Log) (Log, error) {
	now := time.Now().UTC()
	log.CreatedAt, log.UpdatedAt = now, now

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Log{}, fmt.Errorf("create log %q: %w", log.Slug, err)
	}
	defer tx.Rollback()

	taken, err := slugTaken(ctx, tx, log.Slug)
	if err != nil {
		return Log{}, err
	}
	if taken {
		return Log{}, ErrSlugConflict
	}

	if err := insertLog(ctx, tx, log); err != nil {
		return Log{}, err
	}
	if err := tx.Commit(); err != nil {
		return Log{}, fmt.Errorf("create log %q: %w", log.Slug, err)
	}
	return log, nil
}

// UpdateLog overwrites the log currently at oldSlug with the fields of log,
// stamping updated_at now and leaving created_at untouched. When log.Slug
// differs from oldSlug this performs the rename move; a rename onto an existing
// slug reports ErrSlugConflict. A missing oldSlug reports ErrLogNotFound.
func (db *DB) UpdateLog(ctx context.Context, oldSlug string, log Log) (Log, error) {
	log.UpdatedAt = time.Now().UTC()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Log{}, fmt.Errorf("update log %q: %w", oldSlug, err)
	}
	defer tx.Rollback()

	if log.Slug != oldSlug {
		taken, err := slugTaken(ctx, tx, log.Slug)
		if err != nil {
			return Log{}, err
		}
		if taken {
			return Log{}, ErrSlugConflict
		}
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE log SET slug = ?, title = ?, subtitle = ?, description = ?, cover_image = ?,
		 reading_time = ?, tags = ?, body = ?, state = ?, updated_at = ? WHERE slug = ?`,
		log.Slug, log.Title, log.Subtitle, log.Description, log.CoverImage,
		log.ReadingTime, encodeTags(log.Tags), log.Body, log.State,
		log.UpdatedAt.Format(time.RFC3339Nano), oldSlug,
	)
	if err != nil {
		return Log{}, fmt.Errorf("update log %q: %w", oldSlug, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Log{}, fmt.Errorf("update log %q: %w", oldSlug, err)
	}
	if affected == 0 {
		return Log{}, ErrLogNotFound
	}
	if err := tx.Commit(); err != nil {
		return Log{}, fmt.Errorf("update log %q: %w", oldSlug, err)
	}
	return log, nil
}

// DeleteLog removes the log at slug, reporting ErrLogNotFound when there is none.
func (db *DB) DeleteLog(ctx context.Context, slug string) error {
	result, err := db.ExecContext(ctx, "DELETE FROM log WHERE slug = ?", slug)
	if err != nil {
		return fmt.Errorf("delete log %q: %w", slug, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete log %q: %w", slug, err)
	}
	if affected == 0 {
		return ErrLogNotFound
	}
	return nil
}

func insertLog(ctx context.Context, tx *sql.Tx, log Log) error {
	_, err := tx.ExecContext(ctx,
		"INSERT INTO log ("+logColumns+") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		log.Slug, log.Title, log.Subtitle, log.Description, log.CoverImage,
		log.ReadingTime, encodeTags(log.Tags), log.Body, log.State,
		log.CreatedAt.Format(time.RFC3339Nano), log.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert log %q: %w", log.Slug, err)
	}
	return nil
}

func slugTaken(ctx context.Context, tx *sql.Tx, slug string) (bool, error) {
	var one int
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM log WHERE slug = ?", slug).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check slug %q: %w", slug, err)
	}
	return true, nil
}

// encodeTags renders a tags slice as the JSON array the tags column stores,
// mapping a nil or empty slice to "[]" rather than "null".
func encodeTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	encoded, err := json.Marshal(tags)
	if err != nil {
		return "[]"
	}
	return string(encoded)
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
