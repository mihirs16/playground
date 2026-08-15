// Package health computes custodian's single health signal: a 1/0 gauge it
// self-assesses on the poll loop's timer and reports through the telemetry sink.
// The assessment is deliberately narrow — SQLite reachable (SELECT 1), the
// object-store bucket reachable (HeadBucket), and local disk/memory headroom —
// and excludes every third-party source, so Steam or GitHub being down never
// turns custodian red.
package health

import (
	"context"

	"github.com/mihirs16/playground/custodian/internal/edges"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// Checker self-assesses custodian and records the result as the health gauge.
type Checker struct {
	db        *storage.DB
	store     edges.ObjectStore
	telemetry edges.Telemetry

	// headroom reports whether local disk and memory have room to spare. It is a
	// field so a test can drive the degraded branch without exhausting the host.
	headroom func() bool
}

// New builds a Checker over the database and the object-store and telemetry
// edges. The two edges are interfaces, so a test drives the whole assessment
// against fakes and never touches the network.
func New(db *storage.DB, store edges.ObjectStore, telemetry edges.Telemetry) *Checker {
	return &Checker{db: db, store: store, telemetry: telemetry, headroom: localHeadroom}
}

// Assess runs the three local checks, records the result through the telemetry
// sink as the health gauge, and returns whether custodian is healthy. Any single
// check failing is degraded; an unreachable third-party source is not a check
// here at all, by design.
func (c *Checker) Assess(ctx context.Context) bool {
	ok := true
	if err := c.db.PingContext(ctx); err != nil {
		ok = false
	}
	if err := c.store.HeadBucket(ctx); err != nil {
		ok = false
	}
	if !c.headroom() {
		ok = false
	}
	c.telemetry.RecordHealth(ctx, ok)
	return ok
}
