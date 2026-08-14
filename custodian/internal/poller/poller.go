// Package poller drives custodian's one background loop: it polls each
// third-party source on its own cadence and, on the same schedule, reaps stale
// media reservations. Every write it makes lands in storage; persona reads only
// ever touch the stored rows, never a live third party.
package poller

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/mihirs16/playground/custodian/internal/config"
	"github.com/mihirs16/playground/custodian/internal/edges"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// Poller owns the poll-and-reap loop. Credentials are resolved from config at
// construction (env-at-startup, like every other secret) and passed per-fetch;
// the poller never reads or writes them anywhere else.
type Poller struct {
	db        *storage.DB
	client    edges.SourceClient
	keys      map[string]string
	intervals map[string]time.Duration
}

// New builds a poller over the injected source client. keys maps a source to its
// third-party credential; intervals maps a source to its poll cadence, each
// already resolved against the default by config.
func New(db *storage.DB, client edges.SourceClient, keys map[string]string, intervals map[string]time.Duration) *Poller {
	return &Poller{db: db, client: client, keys: keys, intervals: intervals}
}

// Poll fetches a source once and appends a new row only when the state changed.
// A 304 (NotModified) and an unchanged body both leave the timeseries alone and
// return the last stored row, so an idle poll and a briefly-unreachable source
// are indistinguishable downstream. It returns the row now current for the
// source. A fetch error is returned as-is: an unreachable source is never
// written as a state change.
func (p *Poller) Poll(ctx context.Context, source string) (storage.Integration, error) {
	latest, err := p.db.LatestIntegration(ctx, source)
	hasLatest := err == nil
	if err != nil && !errors.Is(err, storage.ErrIntegrationNotFound) {
		return storage.Integration{}, err
	}

	priorETag := ""
	if hasLatest {
		priorETag = latest.ETag
	}

	result, err := p.client.Fetch(ctx, source, p.keys[source], priorETag)
	if err != nil {
		return storage.Integration{}, err
	}
	if result.NotModified && hasLatest {
		return latest, nil
	}

	data, err := json.Marshal(result.Data)
	if err != nil {
		return storage.Integration{}, err
	}
	body := string(data)
	if string(data) == "null" {
		body = ""
	}

	if hasLatest && body == latest.Data {
		return latest, nil
	}

	return p.db.AppendIntegration(ctx, storage.Integration{
		Source: source,
		Data:   body,
		ETag:   result.ETag,
	})
}

// Reap deletes every pending media reservation whose upload window has closed,
// returning how many were removed.
func (p *Poller) Reap(ctx context.Context, now time.Time) (int64, error) {
	return p.db.ReapExpiredPendingMedia(ctx, now)
}

// Startup polls every known source once, so a row exists before persona's first
// read. A source that is unreachable at startup is logged and skipped — the read
// surface serves its empty-but-present shape until a later poll succeeds.
func (p *Poller) Startup(ctx context.Context, logger *slog.Logger) {
	for _, source := range p.sources() {
		if _, err := p.Poll(ctx, source); err != nil {
			logger.Warn("startup poll failed", "source", source, "error", err)
		}
	}
}

// Run polls each source once at startup, then loops each source on its own
// ticker until ctx is cancelled. Every tick also reaps stale media, so the poll
// loop is the single place stale reservations are cleaned up.
func (p *Poller) Run(ctx context.Context, logger *slog.Logger) {
	p.Startup(ctx, logger)

	done := make(chan struct{})
	sources := p.sources()
	for _, source := range sources {
		go p.loop(ctx, source, logger, done)
	}
	for range sources {
		<-done
	}
}

func (p *Poller) loop(ctx context.Context, source string, logger *slog.Logger, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()

	ticker := time.NewTicker(p.interval(source))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := p.Poll(ctx, source); err != nil {
				logger.Warn("poll failed", "source", source, "error", err)
			}
			if _, err := p.Reap(ctx, time.Now().UTC()); err != nil {
				logger.Warn("reap failed", "error", err)
			}
		}
	}
}

func (p *Poller) sources() []string {
	return config.Sources
}

func (p *Poller) interval(source string) time.Duration {
	if d, ok := p.intervals[source]; ok && d > 0 {
		return d
	}
	return 5 * time.Minute
}
