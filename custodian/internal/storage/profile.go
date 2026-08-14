package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrProfileNotFound is returned by GetProfile when no profile record carries
// the requested key.
var ErrProfileNotFound = errors.New("profile not found")

// Profile is one row of the profile table: a key and its opaque JSON body.
// custodian stores the body verbatim and never interprets its shape — that is a
// convention shared between broom and persona.
type Profile struct {
	Key  string
	Body string
}

// GetProfile returns the profile record at key, reporting ErrProfileNotFound
// when there is none.
func (db *DB) GetProfile(ctx context.Context, key string) (Profile, error) {
	var profile Profile
	err := db.QueryRowContext(ctx, "SELECT key, body FROM profile WHERE key = ?", key).
		Scan(&profile.Key, &profile.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("get profile %q: %w", key, err)
	}
	return profile, nil
}

// UpsertProfile writes body at key, inserting a new record or overwriting the
// existing one, and returns the stored record. The body is stored verbatim.
func (db *DB) UpsertProfile(ctx context.Context, key, body string) (Profile, error) {
	_, err := db.ExecContext(ctx,
		`INSERT INTO profile (key, body) VALUES (?, ?)
		 ON CONFLICT (key) DO UPDATE SET body = excluded.body`,
		key, body,
	)
	if err != nil {
		return Profile{}, fmt.Errorf("upsert profile %q: %w", key, err)
	}
	return Profile{Key: key, Body: body}, nil
}
