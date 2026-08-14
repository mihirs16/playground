package server_test

import (
	"net/http"
	"testing"
)

// TestCreateLogMakesUnlistedDraft covers the create contract: an author-chosen
// slug lands as an unlisted draft, echoed back with a 201.
func TestCreateLogMakesUnlistedDraft(t *testing.T) {
	h := newHarness(t)

	resp := h.requestJSON(t, http.MethodPost, "/admin/v1/logs", adminAuth(), map[string]any{
		"slug":  "first-post",
		"title": "First Post",
		"body":  "# hello",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var log struct {
		Slug  string `json:"slug"`
		State string `json:"state"`
		Body  string `json:"body"`
	}
	decode(t, resp, &log)
	if log.Slug != "first-post" {
		t.Fatalf("slug = %q, want first-post", log.Slug)
	}
	if log.State != "unlisted" {
		t.Fatalf("state = %q, want unlisted (new logs are drafts)", log.State)
	}
	if log.Body != "# hello" {
		t.Fatalf("body = %q, want the submitted body", log.Body)
	}
}

func TestCreateLogSlugCollisionIs409(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "taken", title: "Taken", state: "listed", createdAt: day(1), updatedAt: day(1)})

	resp := h.requestJSON(t, http.MethodPost, "/admin/v1/logs", adminAuth(), map[string]any{
		"slug":  "taken",
		"title": "Second",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
	assertProblemCode(t, resp, "slug_conflict")
}

func TestCreateLogValidationIs422WithFieldErrors(t *testing.T) {
	h := newHarness(t)

	resp := h.requestJSON(t, http.MethodPost, "/admin/v1/logs", adminAuth(), map[string]any{
		"slug":  "Not A Slug",
		"title": "",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
	var problem struct {
		Code   string `json:"code"`
		Errors []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	decode(t, resp, &problem)
	if problem.Code != "validation_failed" {
		t.Fatalf("code = %q, want validation_failed", problem.Code)
	}
	if len(problem.Errors) < 2 {
		t.Fatalf("errors = %+v, want a field error for each of slug and title", problem.Errors)
	}
}

// TestPatchPublishesAndUnpublishes covers state transitions with no bespoke
// endpoint: PATCH state:listed publishes, PATCH state:unlisted takes down.
func TestPatchPublishesAndUnpublishes(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "draft", title: "Draft", state: "unlisted", createdAt: day(1), updatedAt: day(1)})

	publish := h.requestJSON(t, http.MethodPatch, "/admin/v1/logs/draft", adminAuth(), map[string]any{"state": "listed"})
	defer publish.Body.Close()
	if publish.StatusCode != http.StatusOK {
		t.Fatalf("publish status = %d, want 200", publish.StatusCode)
	}
	if state, _ := h.logState(t, "draft"); state != "listed" {
		t.Fatalf("after publish, state = %q, want listed", state)
	}

	unpublish := h.requestJSON(t, http.MethodPatch, "/admin/v1/logs/draft", adminAuth(), map[string]any{"state": "unlisted"})
	defer unpublish.Body.Close()
	if state, _ := h.logState(t, "draft"); state != "unlisted" {
		t.Fatalf("after unpublish, state = %q, want unlisted", state)
	}
}

// TestPatchOnlyTouchesPresentFields proves the partial-update contract: a patch
// of one field leaves the others as they were.
func TestPatchOnlyTouchesPresentFields(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "post", title: "Original", body: "original body", state: "listed", createdAt: day(1), updatedAt: day(1)})

	resp := h.requestJSON(t, http.MethodPatch, "/admin/v1/logs/post", adminAuth(), map[string]any{"title": "Renamed Title"})
	defer resp.Body.Close()

	var log struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		State string `json:"state"`
	}
	decode(t, resp, &log)
	if log.Title != "Renamed Title" {
		t.Fatalf("title = %q, want the patched value", log.Title)
	}
	if log.Body != "original body" {
		t.Fatalf("body = %q, want the untouched original", log.Body)
	}
	if log.State != "listed" {
		t.Fatalf("state = %q, want the untouched listed", log.State)
	}
}

// TestSlugRenameWhileUnlistedMoves covers a rename move: the log answers at the
// new slug and no longer at the old one.
func TestSlugRenameWhileUnlistedMoves(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "old-slug", title: "Draft", state: "unlisted", createdAt: day(1), updatedAt: day(1)})

	resp := h.requestJSON(t, http.MethodPatch, "/admin/v1/logs/old-slug", adminAuth(), map[string]any{"slug": "new-slug"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var log struct {
		Slug string `json:"slug"`
	}
	decode(t, resp, &log)
	if log.Slug != "new-slug" {
		t.Fatalf("echoed slug = %q, want new-slug", log.Slug)
	}
	if _, ok := h.logState(t, "old-slug"); ok {
		t.Fatal("old slug still resolves; the rename did not move the row")
	}
	if _, ok := h.logState(t, "new-slug"); !ok {
		t.Fatal("new slug does not resolve; the rename lost the row")
	}
}

func TestSlugRenameWhileListedIsFrozen(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "published", title: "Published", state: "listed", createdAt: day(1), updatedAt: day(1)})

	resp := h.requestJSON(t, http.MethodPatch, "/admin/v1/logs/published", adminAuth(), map[string]any{"slug": "renamed"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
	assertProblemCode(t, resp, "slug_frozen_while_listed")
	if _, ok := h.logState(t, "published"); !ok {
		t.Fatal("a frozen rename must leave the log at its original slug")
	}
}

func TestPatchUnknownSlugIs404(t *testing.T) {
	h := newHarness(t)

	resp := h.requestJSON(t, http.MethodPatch, "/admin/v1/logs/nope", adminAuth(), map[string]any{"title": "x"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestDeleteRemovesLog(t *testing.T) {
	h := newHarness(t)
	h.seed(t, seedLog{slug: "doomed", title: "Doomed", state: "listed", createdAt: day(1), updatedAt: day(1)})

	resp := h.request(t, http.MethodDelete, "/admin/v1/logs/doomed", adminAuth())
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if _, ok := h.logState(t, "doomed"); ok {
		t.Fatal("log still present after delete")
	}
}

func TestDeleteUnknownSlugIs404(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodDelete, "/admin/v1/logs/ghost", adminAuth())
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// TestAdminListShowsEveryState proves the admin index returns drafts the public
// index would hide, and that the state filter narrows it.
func TestAdminListShowsEveryState(t *testing.T) {
	h := newHarness(t)
	h.seed(t,
		seedLog{slug: "shipped", title: "Shipped", state: "listed", createdAt: day(2), updatedAt: day(2)},
		seedLog{slug: "draft", title: "Draft", state: "unlisted", createdAt: day(3), updatedAt: day(3)},
	)

	all := h.request(t, http.MethodGet, "/admin/v1/logs", adminAuth())
	defer all.Body.Close()
	var index struct {
		Total int `json:"total"`
	}
	decode(t, all, &index)
	if index.Total != 2 {
		t.Fatalf("admin total = %d, want 2 (drafts included)", index.Total)
	}

	drafts := h.request(t, http.MethodGet, "/admin/v1/logs?state=unlisted", adminAuth())
	defer drafts.Body.Close()
	var filtered struct {
		Total int `json:"total"`
		Items []struct {
			Slug string `json:"slug"`
		} `json:"items"`
	}
	decode(t, drafts, &filtered)
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Slug != "draft" {
		t.Fatalf("state=unlisted filter = %+v, want only draft", filtered)
	}
}

// TestAdminResponsesAreNoStore covers the caching guard: no admin response —
// success or rejection — may be cached.
func TestAdminResponsesAreNoStore(t *testing.T) {
	h := newHarness(t)

	authed := h.request(t, http.MethodGet, "/admin/v1/logs", adminAuth())
	authed.Body.Close()
	if cc := authed.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("authed Cache-Control = %q, want no-store", cc)
	}

	rejected := h.request(t, http.MethodGet, "/admin/v1/logs", nil)
	rejected.Body.Close()
	if cc := rejected.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("rejected Cache-Control = %q, want no-store", cc)
	}
}

// TestAdminIgnoresCookieCredential proves the token is honoured only on the
// Authorization header: the same token in a cookie is not a credential.
func TestAdminIgnoresCookieCredential(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodGet, "/admin/v1/logs", map[string]string{
		"Cookie": "Authorization=Bearer " + testAdminToken,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (a cookie is never a credential)", resp.StatusCode)
	}
}

func assertProblemCode(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	if ct := resp.Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type = %q, want application/problem+json", ct)
	}
	var problem struct {
		Code string `json:"code"`
	}
	decode(t, resp, &problem)
	if problem.Code != want {
		t.Fatalf("code = %q, want %q", problem.Code, want)
	}
}
