package cli_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// profileFake is a stateful in-process custodian covering the profile endpoints:
// it serves records on the public GET surface and upserts them on the admin PUT
// surface, storing the opaque body verbatim. It records every request so a test
// can assert on the round-trip broom makes.
type profileFake struct {
	server   *httptest.Server
	requests []recordedRequest
	profiles map[string]map[string]any
}

func newProfileFake(t *testing.T) *profileFake {
	t.Helper()
	f := &profileFake{profiles: map[string]map[string]any{}}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *profileFake) handle(w http.ResponseWriter, r *http.Request) {
	body := decodeBody(r)
	f.requests = append(f.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: body})

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/profile/"):
		f.getProfile(w, strings.TrimPrefix(r.URL.Path, "/v1/profile/"))

	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/admin/v1/profile/"):
		f.putProfile(w, strings.TrimPrefix(r.URL.Path, "/admin/v1/profile/"), body)

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *profileFake) getProfile(w http.ResponseWriter, key string) {
	stored := f.profiles[key]
	if stored == nil {
		writeProblem(w, http.StatusNotFound, "not_found", "no profile record with that key")
		return
	}
	writeJSONResp(w, http.StatusOK, map[string]any{"key": key, "body": stored})
}

func (f *profileFake) putProfile(w http.ResponseWriter, key string, body map[string]any) {
	f.profiles[key] = body
	writeJSONResp(w, http.StatusOK, map[string]any{"key": key, "body": body})
}

func (f *profileFake) last(t *testing.T, method, pathPrefix string) recordedRequest {
	t.Helper()
	for i := len(f.requests) - 1; i >= 0; i-- {
		req := f.requests[i]
		if req.Method == method && strings.HasPrefix(req.Path, pathPrefix) {
			return req
		}
	}
	t.Fatalf("no %s request to %s among %+v", method, pathPrefix, f.requests)
	return recordedRequest{}
}

func (f *profileFake) count(method, pathPrefix string) int {
	n := 0
	for _, req := range f.requests {
		if req.Method == method && strings.HasPrefix(req.Path, pathPrefix) {
			n++
		}
	}
	return n
}

func TestProfileGetShowsRawJSON(t *testing.T) {
	fake := newProfileFake(t)
	fake.profiles["about"] = map[string]any{"headline": "Engineer", "location": "Remote"}
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "profile", "get", "about"); err != nil {
		t.Fatalf("profile get: %v", err)
	}

	if fake.count(http.MethodGet, "/v1/profile/about") != 1 {
		t.Errorf("want one fetch, got %d", fake.count(http.MethodGet, "/v1/profile/about"))
	}
	if !strings.Contains(out.String(), `"headline": "Engineer"`) {
		t.Errorf("stdout = %q, want the record's JSON", out.String())
	}
}

// The body is opaque: whatever shape the author saves is round-tripped through
// the fake custodian and the faked $EDITOR without broom imposing a schema.
func TestProfileEditUpsertsEditedJSON(t *testing.T) {
	fake := newProfileFake(t)
	fake.profiles["about"] = map[string]any{"headline": "Old"}
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	var seenInitial string
	env.Edit = func(path string) error {
		raw, _ := os.ReadFile(path)
		seenInitial = string(raw)
		return os.WriteFile(path, []byte(`{"headline":"New","nested":{"a":[1,2]}}`), 0o600)
	}

	if err := run(env, "profile", "edit", "about"); err != nil {
		t.Fatalf("profile edit: %v", err)
	}

	if !strings.Contains(seenInitial, `"headline": "Old"`) {
		t.Errorf("editor opened on %q, want the pulled body", seenInitial)
	}
	put := fake.last(t, http.MethodPut, "/admin/v1/profile/about")
	if put.Body["headline"] != "New" {
		t.Errorf("put headline = %v, want New", put.Body["headline"])
	}
	nested, ok := put.Body["nested"].(map[string]any)
	if !ok || len(nested["a"].([]any)) != 2 {
		t.Errorf("opaque nested shape not round-tripped: %v", put.Body["nested"])
	}
	if !strings.Contains(out.String(), "Saved profile about") {
		t.Errorf("stdout = %q, want the saved line", out.String())
	}
}

// A key with no record yet opens as an empty object and edit upserts it, so a
// new profile record can be authored the same way one is revised.
func TestProfileEditCreatesMissingRecord(t *testing.T) {
	fake := newProfileFake(t)
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	var seenInitial string
	env.Edit = func(path string) error {
		raw, _ := os.ReadFile(path)
		seenInitial = string(raw)
		return os.WriteFile(path, []byte(`{"created":true}`), 0o600)
	}

	if err := run(env, "profile", "edit", "resume-link"); err != nil {
		t.Fatalf("profile edit: %v", err)
	}
	if strings.TrimSpace(seenInitial) != "{}" {
		t.Errorf("missing record should open on empty object, got %q", seenInitial)
	}
	put := fake.last(t, http.MethodPut, "/admin/v1/profile/resume-link")
	if put.Body["created"] != true {
		t.Errorf("put body = %v, want created:true", put.Body)
	}
}

func TestProfileEditUnchangedSkipsPut(t *testing.T) {
	fake := newProfileFake(t)
	fake.profiles["skills"] = map[string]any{"langs": []any{"go"}}
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Edit = abortEditor() // leaves the pulled JSON untouched

	if err := run(env, "profile", "edit", "skills"); err != nil {
		t.Fatalf("profile edit: %v", err)
	}
	if n := fake.count(http.MethodPut, "/admin/v1/profile/"); n != 0 {
		t.Errorf("unchanged body must not PUT, got %d", n)
	}
	if !strings.Contains(out.String(), "No changes") {
		t.Errorf("stdout = %q, want no-changes note", out.String())
	}
}

func TestProfileEditRejectsInvalidJSON(t *testing.T) {
	fake := newProfileFake(t)
	fake.profiles["about"] = map[string]any{"headline": "Old"}
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Edit = func(path string) error {
		return os.WriteFile(path, []byte("{not json"), 0o600)
	}

	if err := run(env, "profile", "edit", "about"); err == nil {
		t.Fatal("invalid JSON should fail rather than PUT garbage")
	}
	if n := fake.count(http.MethodPut, "/admin/v1/profile/"); n != 0 {
		t.Errorf("invalid JSON must not reach the wire, got %d puts", n)
	}
}
