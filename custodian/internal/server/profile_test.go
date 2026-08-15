package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// profileBody reads a profile record's stored body straight from the database,
// for tests that assert an upsert wrote what it claimed.
func (h *harness) profileBody(t *testing.T, key string) (string, bool) {
	t.Helper()
	var body string
	err := h.db.QueryRow(`SELECT body FROM profile WHERE key = ?`, key).Scan(&body)
	if err != nil {
		return "", false
	}
	return body, true
}

// TestPutProfileInsertsThenOverwrites covers upsert semantics: the first PUT
// creates the row, a second PUT to the same key overwrites it rather than
// conflicting or duplicating.
func TestPutProfileInsertsThenOverwrites(t *testing.T) {
	h := newHarness(t)

	first := h.requestJSON(t, http.MethodPut, "/admin/v1/profile/about", adminAuth(),
		map[string]any{"headline": "hello"})
	first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("insert status = %d, want 200", first.StatusCode)
	}

	stored, ok := h.profileBody(t, "about")
	if !ok {
		t.Fatal("about profile was not stored")
	}
	if !jsonEqual(t, stored, `{"headline":"hello"}`) {
		t.Fatalf("stored body = %s, want the inserted body", stored)
	}

	second := h.requestJSON(t, http.MethodPut, "/admin/v1/profile/about", adminAuth(),
		map[string]any{"headline": "goodbye", "extra": 1})
	second.Body.Close()
	if second.StatusCode != http.StatusOK {
		t.Fatalf("overwrite status = %d, want 200", second.StatusCode)
	}

	stored, _ = h.profileBody(t, "about")
	if !jsonEqual(t, stored, `{"headline":"goodbye","extra":1}`) {
		t.Fatalf("stored body = %s, want the overwritten body", stored)
	}
}

// TestPutProfileDoesNotValidateBodyShape covers that custodian stores whatever
// opaque JSON it is handed — here a JSON array, which the ProfileBody schema
// nominally describes as an object.
func TestPutProfileDoesNotValidateBodyShape(t *testing.T) {
	h := newHarness(t)

	resp := h.requestJSON(t, http.MethodPut, "/admin/v1/profile/skills", adminAuth(),
		[]string{"go", "sqlite"})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body shape is not validated)", resp.StatusCode)
	}

	stored, ok := h.profileBody(t, "skills")
	if !ok {
		t.Fatal("skills profile was not stored")
	}
	if !jsonEqual(t, stored, `["go","sqlite"]`) {
		t.Fatalf("stored body = %s, want the array verbatim", stored)
	}
}

func TestPutProfileRequiresAuth(t *testing.T) {
	h := newHarness(t)

	resp := h.requestJSON(t, http.MethodPut, "/admin/v1/profile/about", nil,
		map[string]any{"headline": "hello"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if _, ok := h.profileBody(t, "about"); ok {
		t.Fatal("unauthenticated PUT must not write a profile record")
	}
}

func TestPutProfileIsNoStore(t *testing.T) {
	h := newHarness(t)

	resp := h.requestJSON(t, http.MethodPut, "/admin/v1/profile/about", adminAuth(),
		map[string]any{"headline": "hello"})
	defer resp.Body.Close()
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}
}

// TestGetPublicProfileRoundTripsBody covers the public read: the record baked by
// a PUT reads back with its body intact.
func TestGetPublicProfileRoundTripsBody(t *testing.T) {
	h := newHarness(t)
	h.requestJSON(t, http.MethodPut, "/admin/v1/profile/about", adminAuth(),
		map[string]any{"headline": "hello", "nested": map[string]any{"x": 1}}).Body.Close()

	resp := h.request(t, http.MethodGet, "/v1/profile/about", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var record struct {
		Key  string          `json:"key"`
		Body json.RawMessage `json:"body"`
	}
	decode(t, resp, &record)
	if record.Key != "about" {
		t.Fatalf("key = %q, want about", record.Key)
	}
	if !jsonEqual(t, string(record.Body), `{"headline":"hello","nested":{"x":1}}`) {
		t.Fatalf("body = %s, want the round-tripped body", record.Body)
	}
}

func TestGetPublicProfileUnknownKeyIs404(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/v1/profile/nope", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}
}

// TestProfileReadCarriesETagAndCacheControl asserts the public read carries the
// same validator + caching headers as logs.
func TestProfileReadCarriesETagAndCacheControl(t *testing.T) {
	h := newHarness(t)
	h.requestJSON(t, http.MethodPut, "/admin/v1/profile/about", adminAuth(),
		map[string]any{"headline": "hello"}).Body.Close()

	resp := h.request(t, http.MethodGet, "/v1/profile/about", nil)
	resp.Body.Close()
	if etag := resp.Header.Get("ETag"); etag == "" {
		t.Fatal("missing ETag")
	}
	if cc := resp.Header.Get("Cache-Control"); cc == "" {
		t.Fatal("missing Cache-Control")
	}
}

// TestProfileReadHonoursIfNoneMatch asserts a matching validator yields a 304
// with an empty body that still carries the ETag.
func TestProfileReadHonoursIfNoneMatch(t *testing.T) {
	h := newHarness(t)
	h.requestJSON(t, http.MethodPut, "/admin/v1/profile/about", adminAuth(),
		map[string]any{"headline": "hello"}).Body.Close()

	first := h.request(t, http.MethodGet, "/v1/profile/about", nil)
	etag := first.Header.Get("ETag")
	first.Body.Close()

	second := h.request(t, http.MethodGet, "/v1/profile/about", map[string]string{"If-None-Match": etag})
	body, _ := io.ReadAll(second.Body)
	second.Body.Close()

	if second.StatusCode != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", second.StatusCode)
	}
	if len(body) != 0 {
		t.Fatalf("304 body = %q, want empty", body)
	}
	if second.Header.Get("ETag") != etag {
		t.Fatal("304 must still carry the ETag")
	}
}

// TestProfileReadIsOnCORSAllowlist asserts the public read echoes an allowlisted
// origin's CORS headers and exposes the ETag.
func TestProfileReadIsOnCORSAllowlist(t *testing.T) {
	h := newHarness(t)
	h.requestJSON(t, http.MethodPut, "/admin/v1/profile/about", adminAuth(),
		map[string]any{"headline": "hello"}).Body.Close()

	resp := h.request(t, http.MethodGet, "/v1/profile/about", map[string]string{"Origin": testOrigin})
	resp.Body.Close()
	if acao := resp.Header.Get("Access-Control-Allow-Origin"); acao != testOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", acao, testOrigin)
	}
	if expose := resp.Header.Get("Access-Control-Expose-Headers"); expose != "ETag" {
		t.Fatalf("Access-Control-Expose-Headers = %q, want ETag", expose)
	}
}

func TestPutProfileRejectsInvalidJSON(t *testing.T) {
	h := newHarness(t)

	req, err := http.NewRequest(http.MethodPut, h.server.URL+"/admin/v1/profile/about", strings.NewReader("not json"))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+testAdminToken)
	resp, err := h.server.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

// jsonEqual reports whether two JSON documents are semantically equal, so tests
// compare content rather than incidental key order or whitespace.
func jsonEqual(t *testing.T, a, b string) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		t.Fatalf("unmarshal %q: %v", a, err)
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		t.Fatalf("unmarshal %q: %v", b, err)
	}
	ab, _ := json.Marshal(av)
	bb, _ := json.Marshal(bv)
	return string(ab) == string(bb)
}
