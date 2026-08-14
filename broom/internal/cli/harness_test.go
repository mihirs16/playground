package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mihirs16/playground/broom/internal/cli"
)

const fakeToken = "test-admin-token"

// fakeCustodian is an in-process stand-in for custodian that speaks the same
// admin contract broom's generated client expects: bearer auth on /admin/*,
// and an RFC 9457 problem+json body on rejection. It is the single API-boundary
// seam the CLI tests run the real command wiring against.
type fakeCustodian struct {
	server   *httptest.Server
	requests int
}

func newFakeCustodian(t *testing.T) *fakeCustodian {
	t.Helper()
	fake := &fakeCustodian{}
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		fake.requests++
		if r.Header.Get("Authorization") != "Bearer "+fakeToken {
			writeProblem(w, http.StatusUnauthorized, "unauthorized", "a valid admin bearer token is required")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "total": 0})
	})
	fake.server = httptest.NewServer(mux)
	t.Cleanup(fake.server.Close)
	return fake
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"title": http.StatusText(status), "status": status, "code": code, "detail": detail,
	})
}

// run drives the real command tree with the given args and stdin, isolating the
// config file to a temp XDG dir and pointing broom at the fake custodian.
type runResult struct {
	out string
	err error
}

func run(t *testing.T, fake *fakeCustodian, stdin string, args ...string) runResult {
	t.Helper()
	if os.Getenv("XDG_CONFIG_HOME") == "" {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	}
	t.Setenv("BROOM_TOKEN", "")
	if fake != nil {
		t.Setenv("BROOM_URL", fake.server.URL)
	} else {
		t.Setenv("BROOM_URL", "")
	}

	var out bytes.Buffer
	root := cli.NewRoot(cli.IO{In: strings.NewReader(stdin), Out: &out, Err: &out})
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return runResult{out: out.String(), err: err}
}
