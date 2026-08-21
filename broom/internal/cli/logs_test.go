package cli_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mihirs16/playground/broom/internal/cli"
)

// recordedRequest is one call the fake custodian received, captured so a test can
// assert on the sequence of requests a workflow makes — broom's real contract.
type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   map[string]any
}

// logsFake is a stateful in-process custodian covering the authoring endpoints:
// it stores created logs, serves them back with their bodies, applies patches,
// and can be told a slug is already taken so the create path exercises the
// slug_conflict recovery. It records every request it receives.
type logsFake struct {
	server   *httptest.Server
	requests []recordedRequest
	logs     map[string]map[string]any
	taken    map[string]bool
}

func newLogsFake(t *testing.T) *logsFake {
	t.Helper()
	f := &logsFake{
		logs:  map[string]map[string]any{},
		taken: map[string]bool{},
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *logsFake) handle(w http.ResponseWriter, r *http.Request) {
	body := decodeBody(r)
	f.requests = append(f.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: body})

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/admin/v1/logs":
		f.listLogs(w, r.URL.Query().Get("state"))

	case r.Method == http.MethodPost && r.URL.Path == "/admin/v1/logs":
		f.createLog(w, body)

	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/admin/v1/logs/"):
		slug := strings.TrimPrefix(r.URL.Path, "/admin/v1/logs/")
		f.deleteLog(w, slug)

	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/admin/v1/logs/"):
		slug := strings.TrimPrefix(r.URL.Path, "/admin/v1/logs/")
		f.patchLog(w, slug, body)

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/logs/"):
		slug := strings.TrimPrefix(r.URL.Path, "/v1/logs/")
		f.getLog(w, slug)

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *logsFake) createLog(w http.ResponseWriter, body map[string]any) {
	slug, _ := body["slug"].(string)
	if f.taken[slug] || f.logs[slug] != nil {
		writeProblem(w, http.StatusConflict, "slug_conflict", "a log with that slug already exists")
		return
	}
	log := map[string]any{
		"slug":       slug,
		"title":      body["title"],
		"state":      "unlisted",
		"body":       "",
		"created_at": "2026-08-21T00:00:00Z",
		"updated_at": "2026-08-21T00:00:00Z",
	}
	f.logs[slug] = log
	writeJSONResp(w, http.StatusCreated, log)
}

func (f *logsFake) patchLog(w http.ResponseWriter, slug string, body map[string]any) {
	log := f.logs[slug]
	if log == nil {
		writeProblem(w, http.StatusNotFound, "not_found", "no log with that slug")
		return
	}
	for k, v := range body {
		log[k] = v
	}
	writeJSONResp(w, http.StatusOK, log)
}

func (f *logsFake) getLog(w http.ResponseWriter, slug string) {
	log := f.logs[slug]
	if log == nil {
		writeProblem(w, http.StatusNotFound, "not_found", "no log with that slug")
		return
	}
	writeJSONResp(w, http.StatusOK, log)
}

// listLogs serves the stored logs on the admin surface, narrowing to a single
// state when one is asked for, so a test can assert broom sees drafts of any
// state and that the state filter reaches the wire.
func (f *logsFake) listLogs(w http.ResponseWriter, state string) {
	items := []map[string]any{}
	for _, log := range f.logs {
		if state == "" || log["state"] == state {
			items = append(items, log)
		}
	}
	writeJSONResp(w, http.StatusOK, map[string]any{"total": len(items), "items": items})
}

func (f *logsFake) deleteLog(w http.ResponseWriter, slug string) {
	if f.logs[slug] == nil {
		writeProblem(w, http.StatusNotFound, "not_found", "no log with that slug")
		return
	}
	delete(f.logs, slug)
	w.WriteHeader(http.StatusNoContent)
}

// last returns the most recent request whose method and path prefix match,
// failing the test if there was none.
func (f *logsFake) last(t *testing.T, method, pathPrefix string) recordedRequest {
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

func (f *logsFake) count(method, pathPrefix string) int {
	n := 0
	for _, req := range f.requests {
		if req.Method == method && strings.HasPrefix(req.Path, pathPrefix) {
			n++
		}
	}
	return n
}

func decodeBody(r *http.Request) map[string]any {
	if r.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil || len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

func writeJSONResp(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// scriptedEditor writes content to whatever file it is handed, standing in for
// $EDITOR opening on the body and the author saving.
func scriptedEditor(content string) cli.EditorFunc {
	return func(path string) error {
		return os.WriteFile(path, []byte(content), 0o600)
	}
}

// abortEditor leaves the file untouched, standing in for the author quitting the
// editor without writing.
func abortEditor() cli.EditorFunc {
	return func(string) error { return nil }
}

func TestLogsNewCreatesUnlistedThenPatchesBody(t *testing.T) {
	fake := newLogsFake(t)
	env, out, _, _ := testEnv(t, "My First Post\n\nfor testing\ngo, cli\nA short description\n",
		map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Edit = scriptedEditor("# Hello\n\nThe body.\n")

	if err := run(env, "logs", "new"); err != nil {
		t.Fatalf("logs new: %v", err)
	}

	// The create happens before the editor opens, and it is an unlisted draft.
	create := fake.last(t, http.MethodPost, "/admin/v1/logs")
	if create.Body["slug"] != "my-first-post" {
		t.Errorf("create slug = %v, want my-first-post (derived from title)", create.Body["slug"])
	}
	if create.Body["title"] != "My First Post" {
		t.Errorf("create title = %v", create.Body["title"])
	}
	if create.Body["subtitle"] != "for testing" {
		t.Errorf("create subtitle = %v", create.Body["subtitle"])
	}
	if create.Body["description"] != "A short description" {
		t.Errorf("create description = %v", create.Body["description"])
	}
	if tags, _ := create.Body["tags"].([]any); len(tags) != 2 || tags[0] != "go" || tags[1] != "cli" {
		t.Errorf("create tags = %v, want [go cli]", create.Body["tags"])
	}

	// The saved body is PATCHed to the slug the create returned.
	patch := fake.last(t, http.MethodPatch, "/admin/v1/logs/my-first-post")
	if patch.Body["body"] != "# Hello\n\nThe body.\n" {
		t.Errorf("patched body = %q", patch.Body["body"])
	}
	if !strings.Contains(out.String(), "Created unlisted draft my-first-post") {
		t.Errorf("stdout = %q, want create line", out.String())
	}
}

func TestLogsNewAcceptsExplicitSlug(t *testing.T) {
	fake := newLogsFake(t)
	env, _, _, _ := testEnv(t, "My First Post\ncustom-slug\n\n\n\n",
		map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Edit = scriptedEditor("body")

	if err := run(env, "logs", "new"); err != nil {
		t.Fatalf("logs new: %v", err)
	}
	create := fake.last(t, http.MethodPost, "/admin/v1/logs")
	if create.Body["slug"] != "custom-slug" {
		t.Errorf("create slug = %v, want custom-slug", create.Body["slug"])
	}
}

func TestLogsNewEmptyBodyLeavesDraftWithoutPatch(t *testing.T) {
	fake := newLogsFake(t)
	env, out, _, _ := testEnv(t, "Draft\n\n\n\n\n",
		map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Edit = abortEditor()

	if err := run(env, "logs", "new"); err != nil {
		t.Fatalf("logs new: %v", err)
	}
	if fake.count(http.MethodPost, "/admin/v1/logs") != 1 {
		t.Errorf("want exactly one create, got %d", fake.count(http.MethodPost, "/admin/v1/logs"))
	}
	if n := fake.count(http.MethodPatch, "/admin/v1/logs/"); n != 0 {
		t.Errorf("aborted empty body must not PATCH, got %d patches", n)
	}
	if !strings.Contains(out.String(), "Empty draft") {
		t.Errorf("stdout = %q, want empty-draft note", out.String())
	}
}

func TestLogsNewSlugConflictReprompts(t *testing.T) {
	fake := newLogsFake(t)
	fake.taken["taken-slug"] = true
	env, out, _, _ := testEnv(t, "Post\ntaken-slug\n\n\n\nfree-slug\n",
		map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Edit = abortEditor()

	if err := run(env, "logs", "new"); err != nil {
		t.Fatalf("logs new: %v", err)
	}

	if fake.count(http.MethodPost, "/admin/v1/logs") != 2 {
		t.Errorf("want two create attempts (conflict then retry), got %d", fake.count(http.MethodPost, "/admin/v1/logs"))
	}
	create := fake.last(t, http.MethodPost, "/admin/v1/logs")
	if create.Body["slug"] != "free-slug" {
		t.Errorf("retry slug = %v, want free-slug", create.Body["slug"])
	}
	if !strings.Contains(out.String(), "Created unlisted draft free-slug") {
		t.Errorf("stdout = %q, want create on the free slug", out.String())
	}
}

// An empty or closed stdin at the required title prompt must fail, not spin
// re-prompting into a stdin that will never yield input.
func TestLogsNewEmptyStdinFailsFast(t *testing.T) {
	fake := newLogsFake(t)
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Edit = abortEditor()

	err := run(env, "logs", "new")
	if err == nil {
		t.Fatal("logs new with empty stdin should fail, not hang")
	}
	if fake.count(http.MethodPost, "/admin/v1/logs") != 0 {
		t.Errorf("no post should be created without a title, got %d", fake.count(http.MethodPost, "/admin/v1/logs"))
	}
}

func TestLogsEditRoundTripsBody(t *testing.T) {
	fake := newLogsFake(t)
	fake.logs["existing"] = map[string]any{
		"slug": "existing", "title": "Existing", "state": "unlisted",
		"body":       "old body",
		"created_at": "2026-08-21T00:00:00Z", "updated_at": "2026-08-21T00:00:00Z",
	}
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	var seenInitial string
	env.Edit = func(path string) error {
		raw, _ := os.ReadFile(path)
		seenInitial = string(raw)
		return os.WriteFile(path, []byte("new body"), 0o600)
	}

	if err := run(env, "logs", "edit", "existing"); err != nil {
		t.Fatalf("logs edit: %v", err)
	}

	if seenInitial != "old body" {
		t.Errorf("editor opened on %q, want the pulled body", seenInitial)
	}
	if fake.count(http.MethodGet, "/v1/logs/existing") != 1 {
		t.Errorf("want one body pull, got %d", fake.count(http.MethodGet, "/v1/logs/existing"))
	}
	patch := fake.last(t, http.MethodPatch, "/admin/v1/logs/existing")
	if patch.Body["body"] != "new body" {
		t.Errorf("patched body = %q, want new body", patch.Body["body"])
	}
	if !strings.Contains(out.String(), "Saved body of existing") {
		t.Errorf("stdout = %q", out.String())
	}
}

// seedLog stores a log in the fake in the given state so management commands
// have something to act on.
func (f *logsFake) seedLog(slug, title string, state string) {
	f.logs[slug] = map[string]any{
		"slug": slug, "title": title, "state": state, "body": "",
		"created_at": "2026-08-21T00:00:00Z", "updated_at": "2026-08-21T00:00:00Z",
	}
}

func TestLogsListShowsAllStates(t *testing.T) {
	fake := newLogsFake(t)
	fake.seedLog("live-post", "Live Post", "listed")
	fake.seedLog("draft-post", "Draft Post", "unlisted")
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "logs", "list"); err != nil {
		t.Fatalf("logs list: %v", err)
	}

	list := fake.last(t, http.MethodGet, "/admin/v1/logs")
	if list.Query != "" {
		t.Errorf("unfiltered list should carry no state query, got %q", list.Query)
	}
	if !strings.Contains(out.String(), "draft-post") || !strings.Contains(out.String(), "live-post") {
		t.Errorf("stdout = %q, want both the listed post and the hidden draft", out.String())
	}
}

func TestLogsListFiltersByState(t *testing.T) {
	cases := []struct {
		flag  string
		state string
	}{
		{"--unlisted", "unlisted"},
		{"--listed", "listed"},
	}
	for _, tc := range cases {
		t.Run(tc.flag, func(t *testing.T) {
			fake := newLogsFake(t)
			fake.seedLog("live-post", "Live Post", "listed")
			fake.seedLog("draft-post", "Draft Post", "unlisted")
			env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

			if err := run(env, "logs", "list", tc.flag); err != nil {
				t.Fatalf("logs list %s: %v", tc.flag, err)
			}
			list := fake.last(t, http.MethodGet, "/admin/v1/logs")
			if !strings.Contains(list.Query, "state="+tc.state) {
				t.Errorf("query = %q, want state=%s", list.Query, tc.state)
			}
		})
	}
}

func TestLogsListRejectsConflictingFilters(t *testing.T) {
	fake := newLogsFake(t)
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "logs", "list", "--listed", "--unlisted"); err == nil {
		t.Fatal("listed and unlisted together should be rejected")
	}
}

func TestLogsPublishPatchesStateListed(t *testing.T) {
	fake := newLogsFake(t)
	fake.seedLog("draft-post", "Draft Post", "unlisted")
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "logs", "publish", "draft-post"); err != nil {
		t.Fatalf("logs publish: %v", err)
	}
	patch := fake.last(t, http.MethodPatch, "/admin/v1/logs/draft-post")
	if patch.Body["state"] != "listed" {
		t.Errorf("publish patched state = %v, want listed", patch.Body["state"])
	}
	if _, hasBody := patch.Body["body"]; hasBody {
		t.Errorf("publish must patch only state, body carried %v", patch.Body)
	}
	if !strings.Contains(out.String(), "Published draft-post") {
		t.Errorf("stdout = %q, want the published line", out.String())
	}
}

func TestLogsUnpublishPatchesStateUnlisted(t *testing.T) {
	fake := newLogsFake(t)
	fake.seedLog("live-post", "Live Post", "listed")
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "logs", "unpublish", "live-post"); err != nil {
		t.Fatalf("logs unpublish: %v", err)
	}
	patch := fake.last(t, http.MethodPatch, "/admin/v1/logs/live-post")
	if patch.Body["state"] != "unlisted" {
		t.Errorf("unpublish patched state = %v, want unlisted", patch.Body["state"])
	}
	if !strings.Contains(out.String(), "Unpublished live-post") {
		t.Errorf("stdout = %q, want the unpublished line", out.String())
	}
}

func TestLogsRmDeletes(t *testing.T) {
	fake := newLogsFake(t)
	fake.seedLog("draft-post", "Draft Post", "unlisted")
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "logs", "rm", "draft-post"); err != nil {
		t.Fatalf("logs rm: %v", err)
	}
	if fake.count(http.MethodDelete, "/admin/v1/logs/draft-post") != 1 {
		t.Errorf("want one delete, got %d", fake.count(http.MethodDelete, "/admin/v1/logs/draft-post"))
	}
	if fake.logs["draft-post"] != nil {
		t.Error("the log should be gone after rm")
	}
	if !strings.Contains(out.String(), "Deleted draft-post") {
		t.Errorf("stdout = %q, want the deleted line", out.String())
	}
}

func TestLogsEditUnchangedBodySkipsPatch(t *testing.T) {
	fake := newLogsFake(t)
	fake.logs["existing"] = map[string]any{
		"slug": "existing", "title": "Existing", "state": "unlisted",
		"body":       "same body",
		"created_at": "2026-08-21T00:00:00Z", "updated_at": "2026-08-21T00:00:00Z",
	}
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Edit = abortEditor() // leaves the pulled body untouched

	if err := run(env, "logs", "edit", "existing"); err != nil {
		t.Fatalf("logs edit: %v", err)
	}
	if n := fake.count(http.MethodPatch, "/admin/v1/logs/"); n != 0 {
		t.Errorf("unchanged body must not PATCH, got %d", n)
	}
	if !strings.Contains(out.String(), "No changes") {
		t.Errorf("stdout = %q, want no-changes note", out.String())
	}
}
