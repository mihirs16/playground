package cli_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// integrationFake stands in for custodian's integration surface: the authed
// refresh endpoint that forces an immediate poll, and the credential rotation is
// deliberately absent — a source's credential is a deploy-time environment
// variable in custodian, never written over the wire. Refresh answers with a
// canned fresh record per source, or a bad-gateway problem when told the source
// is unreachable, so a failed poll can be exercised. Every request is recorded.
type integrationFake struct {
	server      *httptest.Server
	requests    []recordedRequest
	unreachable map[string]bool
}

func newIntegrationFake(t *testing.T) *integrationFake {
	t.Helper()
	f := &integrationFake{unreachable: map[string]bool{}}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *integrationFake) handle(w http.ResponseWriter, r *http.Request) {
	f.requests = append(f.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: decodeBody(r)})

	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/admin/v1/integrations/") && strings.HasSuffix(r.URL.Path, "/refresh") {
		source := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/admin/v1/integrations/"), "/refresh")
		f.refresh(w, source)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// refresh answers a forced poll: a fresh record for a reachable source, or the
// bad-gateway problem custodian returns when the source could not be polled.
func (f *integrationFake) refresh(w http.ResponseWriter, source string) {
	if f.unreachable[source] {
		writeProblem(w, http.StatusBadGateway, "integration_unreachable", "could not poll the source")
		return
	}
	writeJSONResp(w, http.StatusOK, map[string]any{
		"source":     source,
		"fetched_at": "2026-08-26T12:00:00Z",
		"data":       map[string]any{"status": "online", "source": source},
	})
}

func (f *integrationFake) count(method, pathPrefix string) int {
	n := 0
	for _, req := range f.requests {
		if req.Method == method && strings.HasPrefix(req.Path, pathPrefix) {
			n++
		}
	}
	return n
}

// A named refresh POSTs to that one source's refresh endpoint and prints the
// fresh record custodian polled.
func TestIntegrationRefreshNamedSource(t *testing.T) {
	fake := newIntegrationFake(t)
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "integration", "refresh", "steam"); err != nil {
		t.Fatalf("integration refresh steam: %v", err)
	}

	if got := fake.count(http.MethodPost, "/admin/v1/integrations/steam/refresh"); got != 1 {
		t.Errorf("want exactly one refresh POST to steam, got %d among %+v", got, fake.requests)
	}
	if fake.count(http.MethodPost, "/admin/v1/integrations/github/refresh") != 0 {
		t.Error("a named steam refresh must not poll github")
	}
	if !strings.Contains(out.String(), "steam") || !strings.Contains(out.String(), "online") {
		t.Errorf("stdout = %q, want the fresh steam record", out.String())
	}
}

// With no name, refresh fans out over every known source, each an authed POST.
func TestIntegrationRefreshAllSources(t *testing.T) {
	fake := newIntegrationFake(t)
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "integration", "refresh"); err != nil {
		t.Fatalf("integration refresh: %v", err)
	}

	if got := fake.count(http.MethodPost, "/admin/v1/integrations/steam/refresh"); got != 1 {
		t.Errorf("want one steam refresh, got %d", got)
	}
	if got := fake.count(http.MethodPost, "/admin/v1/integrations/github/refresh"); got != 1 {
		t.Errorf("want one github refresh, got %d", got)
	}
	if !strings.Contains(out.String(), "steam") || !strings.Contains(out.String(), "github") {
		t.Errorf("stdout = %q, want both sources' fresh records", out.String())
	}
}

// The refresh request carries the admin bearer token — it is an authed admin
// call, not the public read.
func TestIntegrationRefreshIsAuthenticated(t *testing.T) {
	var gotAuth string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSONResp(w, http.StatusOK, map[string]any{"source": "steam", "fetched_at": "2026-08-26T12:00:00Z"})
	})
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": ts.URL, "BROOM_TOKEN": "secret-admin-token"})

	if err := run(env, "integration", "refresh", "steam"); err != nil {
		t.Fatalf("integration refresh steam: %v", err)
	}
	if gotAuth != "Bearer secret-admin-token" {
		t.Errorf("Authorization = %q, want the admin bearer token", gotAuth)
	}
}

// A source with no observed data yet prints an explicit note, not a bare null.
func TestIntegrationRefreshEmptyRecord(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSONResp(w, http.StatusOK, map[string]any{"source": "github", "fetched_at": "2026-08-26T12:00:00Z"})
	})
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": ts.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "integration", "refresh", "github"); err != nil {
		t.Fatalf("integration refresh github: %v", err)
	}
	if !strings.Contains(out.String(), "no data observed") {
		t.Errorf("stdout = %q, want the empty-but-present note", out.String())
	}
}

// A failed poll (custodian's bad-gateway) surfaces as an error — the operator
// must learn a rotated key did not land, not see a success.
func TestIntegrationRefreshUnreachableSourceErrors(t *testing.T) {
	fake := newIntegrationFake(t)
	fake.unreachable["steam"] = true
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	err := run(env, "integration", "refresh", "steam")
	if err == nil {
		t.Fatal("a failed poll must surface as an error")
	}
	if !strings.Contains(err.Error(), "poll") {
		t.Errorf("error = %q, want the poll-failure detail", err.Error())
	}
}

// An unknown source is rejected before any request reaches the wire, naming the
// sources custodian actually polls.
func TestIntegrationRefreshUnknownSourceRejectedLocally(t *testing.T) {
	fake := newIntegrationFake(t)
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	err := run(env, "integration", "refresh", "twitter")
	if err == nil {
		t.Fatal("an unknown source must fail")
	}
	if !strings.Contains(err.Error(), "twitter") || !strings.Contains(err.Error(), "known sources") {
		t.Errorf("error = %q, want a legible unknown-source message", err.Error())
	}
	if len(fake.requests) != 0 {
		t.Errorf("an unknown source must not reach the wire, got %+v", fake.requests)
	}
}

// The credential gate still applies: an unauthenticated refresh fails with the
// not-logged-in message, never a content error.
func TestIntegrationRefreshRequiresCredential(t *testing.T) {
	env, _, _, _ := testEnv(t, "", nil)

	err := run(env, "integration", "refresh", "steam")
	if err == nil {
		t.Fatal("refresh without a token should fail")
	}
	if !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("error = %q, want not-logged-in message", err.Error())
	}
}
