package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mihirs16/playground/broom/internal/cli"
)

// fakeCustodian is an in-process stand-in for custodian that speaks the same
// admin contract broom's generated client expects. It is the single test seam:
// broom's real command wiring and real generated client run against it, faking
// only the thing that leaves the machine. It authenticates the one call login
// makes (GET /admin/v1/logs) and answers everything else with 404.
func fakeCustodian(t *testing.T, wantToken string) *httptest.Server {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/v1/logs" && r.Method == http.MethodGet {
			if r.Header.Get("Authorization") != "Bearer "+wantToken {
				writeProblem(w, http.StatusUnauthorized, "unauthorized", "a valid admin bearer token is required")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"total": 0, "items": []any{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"title":  http.StatusText(status),
		"status": status,
		"code":   code,
		"detail": detail,
	})
}

// testEnv builds a fully-isolated Env: a temp config file, buffered IO, and a
// scripted environment. stdin is whatever the caller scripts (a token, usually).
func testEnv(t *testing.T, stdin string, environ map[string]string) (cli.Env, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	env := cli.Env{
		Stdin:      strings.NewReader(stdin),
		Stdout:     out,
		Stderr:     errBuf,
		Getenv:     func(k string) string { return environ[k] },
		ConfigPath: path,
	}
	return env, out, errBuf, path
}

// run executes the command tree with args and returns its error, discarding the
// prompt output cobra would otherwise duplicate onto the env buffers.
func run(env cli.Env, args ...string) error {
	root := cli.NewRootCmd(env)
	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	return root.ExecuteContext(context.Background())
}
