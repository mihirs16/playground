package server_test

import (
	"bytes"
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
	testCDNBase    = "https://cdn.example.com"
)

// harness stands the real chi router up against a real temp-file SQLite (with
// migrations applied) and the in-memory fake edges — a black-box client of the
// published HTTP contract.
type harness struct {
	server *httptest.Server
	srv    *server.Server
	edges  edges.Set
	db     *storage.DB
}

// sourceClient exposes the fake Steam/GitHub client the harness is wired to, so
// an integration test can script changed / unchanged / unreachable results.
func (h *harness) sourceClient(t *testing.T) *edges.FakeSourceClient {
	t.Helper()
	client, ok := h.edges.SourceClient.(*edges.FakeSourceClient)
	if !ok {
		t.Fatalf("source client is %T, want *edges.FakeSourceClient", h.edges.SourceClient)
	}
	return client
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
		AdminTokenHash:  hex.EncodeToString(sum[:]),
		CORSAllowlist:   []string{testOrigin},
		MediaCDNBase:    testCDNBase,
		IntegrationKeys: map[string]string{"steam": "steam-key", "github": "github-pat"},
		PollIntervals:   map[string]time.Duration{"steam": time.Minute, "github": time.Minute},
	}

	fakes := edges.NewFakes()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := server.New(cfg, db, fakes, logger)

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)

	return &harness{server: ts, srv: srv, edges: fakes, db: db}
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

// objectStore exposes the fake S3 the harness is wired to, so a media test can
// simulate broom's upload and assert what custodian presigned.
func (h *harness) objectStore(t *testing.T) *edges.FakeObjectStore {
	t.Helper()
	store, ok := h.edges.ObjectStore.(*edges.FakeObjectStore)
	if !ok {
		t.Fatalf("object store is %T, want *edges.FakeObjectStore", h.edges.ObjectStore)
	}
	return store
}

// mediaState reads a media record's state straight from the database, for tests
// that assert confirm did (or did not) flip it.
func (h *harness) mediaState(t *testing.T, key string) (string, bool) {
	t.Helper()
	var state string
	err := h.db.QueryRow(`SELECT state FROM media WHERE key = ?`, key).Scan(&state)
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

// requestJSON sends a request carrying a JSON body, marshalling body and setting
// Content-Type. A nil body sends no payload.
func (h *harness) requestJSON(t *testing.T, method, path string, headers map[string]string, body any) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, h.server.URL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
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

// seedMedia inserts a media row directly so a reap test has reservations to act
// on without walking the reserve path. expiresAt is stored as RFC3339Nano, the
// format the reserve path writes.
func (h *harness) seedMedia(t *testing.T, key, state string, expiresAt time.Time) {
	t.Helper()
	_, err := h.db.Exec(
		`INSERT INTO media (key, state, content_type, url, created_at, expires_at)
		 VALUES (?, ?, 'image/png', ?, ?, ?)`,
		key, state, testCDNBase+"/"+key,
		time.Now().UTC().Format(time.RFC3339Nano), expiresAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatalf("seed media %q: %v", key, err)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
