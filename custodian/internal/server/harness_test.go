package server_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mihirs16/playground/custodian/internal/config"
	"github.com/mihirs16/playground/custodian/internal/edges"
	"github.com/mihirs16/playground/custodian/internal/server"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

const (
	testAdminToken = "test-admin-token"
	testOrigin     = "https://persona.example.com"
)

// harness stands the real chi router up against a real temp-file SQLite (with
// migrations applied) and the in-memory fake edges — a black-box client of the
// published HTTP contract.
type harness struct {
	server *httptest.Server
	edges  edges.Set
	db     *storage.DB
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "custodian.db"))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	sum := sha256.Sum256([]byte(testAdminToken))
	cfg := config.Config{
		AdminTokenHash: hex.EncodeToString(sum[:]),
		CORSAllowlist:  []string{testOrigin},
	}

	fakes := edges.NewFakes()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := server.New(cfg, db, fakes, logger)

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	return &harness{server: ts, edges: fakes, db: db}
}

// seedLog is the test-side stand-in for the not-yet-built write path (ticket
// 04): it inserts a log row directly so the read surface has real rows to
// serve. Timestamps are stored as RFC3339, the format the write path will use.
type seedLog struct {
	slug      string
	title     string
	body      string
	state     string
	tags      []string
	createdAt time.Time
	updatedAt time.Time
}

func (h *harness) seed(t *testing.T, logs ...seedLog) {
	t.Helper()
	for _, log := range logs {
		tags, err := json.Marshal(log.tags)
		if err != nil {
			t.Fatalf("marshal tags: %v", err)
		}
		if log.tags == nil {
			tags = []byte("[]")
		}
		_, err = h.db.Exec(
			`INSERT INTO log (slug, title, body, state, tags, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			log.slug, log.title, log.body, log.state, string(tags),
			log.createdAt.Format(time.RFC3339), log.updatedAt.Format(time.RFC3339),
		)
		if err != nil {
			t.Fatalf("seed log %q: %v", log.slug, err)
		}
	}
}

// logState reads a log's state straight from the database, for tests that
// assert the read surface never mutated stored state.
func (h *harness) logState(t *testing.T, slug string) (string, bool) {
	t.Helper()
	var state string
	err := h.db.QueryRow(`SELECT state FROM log WHERE slug = ?`, slug).Scan(&state)
	if err != nil {
		return "", false
	}
	return state, true
}

func (h *harness) request(t *testing.T, method, path string, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, h.server.URL+path, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := h.server.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func adminAuth() map[string]string {
	return map[string]string{"Authorization": "Bearer " + testAdminToken}
}
