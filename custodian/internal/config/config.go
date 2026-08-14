// Package config reads custodian's runtime configuration from the process
// environment and nowhere else. The source of each variable is deliberately
// opaque: deed may inject secrets from SSM, a tmpfs env file, or a shell — this
// package only ever calls os.Getenv, so swapping the concrete store never
// touches custodian's code.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the fully-resolved configuration for one custodian process.
type Config struct {
	Addr string

	DBPath string

	// AdminTokenHash is the hex-encoded SHA-256 of the admin bearer token.
	// Only the hash is held; rotation is replace-and-restart.
	AdminTokenHash string

	OTLPEndpoint string
	OTLPToken    string

	// IntegrationKeys holds the per-source third-party secret, keyed by source.
	// Read from the environment like every other secret; the source string is
	// treated as opaque and the poller (06) consumes these at fetch time.
	IntegrationKeys map[string]string

	// CORSAllowlist is the set of origins the public surface allows explicitly.
	// Never a wildcard; an empty list means no cross-origin reads are permitted.
	CORSAllowlist []string

	MediaBucket string

	// PollIntervals holds the poll cadence per integration source, each already
	// resolved against the 5-minute default and any per-source override.
	PollIntervals map[string]time.Duration
}

const defaultPollInterval = 5 * time.Minute

// Sources custodian knows how to poll. Kept here so config can resolve a
// per-source interval for each without the poller having started.
var Sources = []string{"steam", "github"}

// Load reads configuration from the environment, applying defaults for every
// value that has a sensible one. It never fails: missing secrets surface later
// as auth or export errors, not as a boot failure, so the box always comes up.
func Load() Config {
	cfg := Config{
		Addr:           getenv("CUSTODIAN_ADDR", ":8080"),
		DBPath:         getenv("CUSTODIAN_DB_PATH", "custodian.db"),
		AdminTokenHash: os.Getenv("CUSTODIAN_ADMIN_TOKEN_HASH"),
		OTLPEndpoint:   os.Getenv("CUSTODIAN_OTLP_ENDPOINT"),
		OTLPToken:      os.Getenv("CUSTODIAN_OTLP_TOKEN"),
		CORSAllowlist:  splitList(os.Getenv("CUSTODIAN_CORS_ALLOWLIST")),
		MediaBucket:    os.Getenv("CUSTODIAN_MEDIA_BUCKET"),
		PollIntervals:  resolvePollIntervals(),
		IntegrationKeys: map[string]string{
			"steam":  os.Getenv("CUSTODIAN_STEAM_KEY"),
			"github": os.Getenv("CUSTODIAN_GITHUB_PAT"),
		},
	}
	return cfg
}

func resolvePollIntervals() map[string]time.Duration {
	base := durationEnv("CUSTODIAN_POLL_INTERVAL", defaultPollInterval)
	intervals := make(map[string]time.Duration, len(Sources))
	for _, source := range Sources {
		key := "CUSTODIAN_POLL_INTERVAL_" + strings.ToUpper(source)
		intervals[source] = durationEnv(key, base)
	}
	return intervals
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		return time.Duration(secs) * time.Second
	}
	return fallback
}
