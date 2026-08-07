package server_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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

	return &harness{server: ts, edges: fakes}
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
