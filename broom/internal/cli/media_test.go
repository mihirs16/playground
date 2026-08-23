package cli_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// mediaFake is a stateful in-process custodian covering the media endpoints plus
// the presigned-PUT target, so broom's real reserve → upload → confirm flow runs
// end to end against it. It fakes the two things that leave the machine: the S3
// PUT (served at /upload/<key>, recording the bytes) and, through those bytes,
// the HEAD custodian does at confirm. It also serves the log endpoints the rm
// reference scan reads. Every request is recorded.
type mediaFake struct {
	server     *httptest.Server
	requests   []recordedRequest
	media      map[string]map[string]any // key -> record
	uploaded   map[string][]byte         // key -> bytes PUT to the presigned URL
	logs       map[string]map[string]any // slug -> log (with body)
	taken      map[string]bool
	cdnBase    string
	uploadAuth string // Authorization header seen on the presigned PUT, if any
}

func newMediaFake(t *testing.T) *mediaFake {
	t.Helper()
	f := &mediaFake{
		media:    map[string]map[string]any{},
		uploaded: map[string][]byte{},
		logs:     map[string]map[string]any{},
		taken:    map[string]bool{},
		cdnBase:  "https://cdn.mihirsingh.dev/media",
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *mediaFake) handle(w http.ResponseWriter, r *http.Request) {
	// The presigned upload PUT is not JSON; capture its raw bytes instead.
	if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/upload/") {
		key := strings.TrimPrefix(r.URL.Path, "/upload/")
		raw, _ := io.ReadAll(r.Body)
		f.uploaded[key] = raw
		f.uploadAuth = r.Header.Get("Authorization")
		f.requests = append(f.requests, recordedRequest{Method: r.Method, Path: r.URL.Path})
		w.WriteHeader(http.StatusOK)
		return
	}

	body := decodeBody(r)
	f.requests = append(f.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: body})

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/admin/v1/media":
		f.reserve(w, body)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/confirm"):
		key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/admin/v1/media/"), "/confirm")
		f.confirm(w, key)
	case r.Method == http.MethodGet && r.URL.Path == "/admin/v1/media":
		f.list(w, r.URL.Query().Get("q"))
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/admin/v1/media/"):
		f.get(w, strings.TrimPrefix(r.URL.Path, "/admin/v1/media/"))
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/admin/v1/media/"):
		f.delete(w, strings.TrimPrefix(r.URL.Path, "/admin/v1/media/"))
	case r.Method == http.MethodGet && r.URL.Path == "/admin/v1/logs":
		f.listLogs(w)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/logs/"):
		f.getLog(w, strings.TrimPrefix(r.URL.Path, "/v1/logs/"))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// reserve mints (or accepts) a key, records a pending record, and hands back a
// presigned upload URL that points back at this same server's PUT handler.
func (f *mediaFake) reserve(w http.ResponseWriter, body map[string]any) {
	key, _ := body["key"].(string)
	if key == "" {
		key = "minted-random-key"
	}
	if f.taken[key] || f.media[key] != nil {
		writeProblem(w, http.StatusConflict, "media_key_taken", "a media record with that key already exists")
		return
	}
	ct, _ := body["content_type"].(string)
	url := f.cdnBase + "/" + key
	f.media[key] = map[string]any{
		"key": key, "state": "pending", "content_type": ct, "url": url,
		"created_at": "2026-08-21T00:00:00Z", "expires_at": "2026-08-21T00:15:00Z",
	}
	writeJSONResp(w, http.StatusCreated, map[string]any{
		"key": key, "url": url, "upload_url": f.server.URL + "/upload/" + key,
		"expires_at": "2026-08-21T00:15:00Z",
	})
}

// confirm flips the record to available, but only if bytes were actually PUT —
// standing in for custodian's HEAD against S3.
func (f *mediaFake) confirm(w http.ResponseWriter, key string) {
	record := f.media[key]
	if record == nil {
		writeProblem(w, http.StatusNotFound, "not_found", "no media with that key")
		return
	}
	if _, ok := f.uploaded[key]; !ok {
		writeProblem(w, http.StatusConflict, "media_bytes_missing", "no uploaded bytes found for that key")
		return
	}
	record["state"] = "available"
	writeJSONResp(w, http.StatusOK, record)
}

func (f *mediaFake) list(w http.ResponseWriter, q string) {
	items := []map[string]any{}
	for _, record := range f.media {
		if q == "" || strings.Contains(record["key"].(string), q) {
			items = append(items, record)
		}
	}
	writeJSONResp(w, http.StatusOK, map[string]any{"total": len(items), "items": items})
}

func (f *mediaFake) get(w http.ResponseWriter, key string) {
	record := f.media[key]
	if record == nil {
		writeProblem(w, http.StatusNotFound, "not_found", "no media with that key")
		return
	}
	writeJSONResp(w, http.StatusOK, record)
}

func (f *mediaFake) delete(w http.ResponseWriter, key string) {
	if f.media[key] == nil {
		writeProblem(w, http.StatusNotFound, "not_found", "no media with that key")
		return
	}
	delete(f.media, key)
	w.WriteHeader(http.StatusNoContent)
}

func (f *mediaFake) listLogs(w http.ResponseWriter) {
	items := []map[string]any{}
	for _, log := range f.logs {
		items = append(items, log)
	}
	writeJSONResp(w, http.StatusOK, map[string]any{"total": len(items), "items": items})
}

func (f *mediaFake) getLog(w http.ResponseWriter, slug string) {
	log := f.logs[slug]
	if log == nil {
		writeProblem(w, http.StatusNotFound, "not_found", "no log with that slug")
		return
	}
	writeJSONResp(w, http.StatusOK, log)
}

func (f *mediaFake) seedMedia(key, state string) {
	url := f.cdnBase + "/" + key
	f.media[key] = map[string]any{
		"key": key, "state": state, "content_type": "image/png", "url": url,
		"created_at": "2026-08-21T00:00:00Z",
	}
}

func (f *mediaFake) seedLog(slug, body string) {
	f.logs[slug] = map[string]any{
		"slug": slug, "title": slug, "state": "listed", "body": body,
		"created_at": "2026-08-21T00:00:00Z", "updated_at": "2026-08-21T00:00:00Z",
	}
}

func (f *mediaFake) last(t *testing.T, method, pathPrefix string) recordedRequest {
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

// lastExact returns the most recent request to an exact method and path, so a
// reserve (POST /admin/v1/media) is not confused with a confirm (POST
// /admin/v1/media/<key>/confirm), which shares its prefix.
func (f *mediaFake) lastExact(t *testing.T, method, path string) recordedRequest {
	t.Helper()
	for i := len(f.requests) - 1; i >= 0; i-- {
		req := f.requests[i]
		if req.Method == method && req.Path == path {
			return req
		}
	}
	t.Fatalf("no %s request to %s among %+v", method, path, f.requests)
	return recordedRequest{}
}

func (f *mediaFake) count(method, pathPrefix string) int {
	n := 0
	for _, req := range f.requests {
		if req.Method == method && strings.HasPrefix(req.Path, pathPrefix) {
			n++
		}
	}
	return n
}

// fakeClipboard captures what a command copies, standing in for the system
// clipboard.
type fakeClipboard struct{ copied string }

func (c *fakeClipboard) copy(text string) error { c.copied = text; return nil }

// writeTempFile writes content to a temp file with the given name and returns its
// path, so `media add` has a real file (with a real extension) to read.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := t.TempDir() + "/" + name
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestMediaAddReserveUploadConfirm(t *testing.T) {
	fake := newMediaFake(t)
	clip := &fakeClipboard{}
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Copy = clip.copy
	file := writeTempFile(t, "diagram.png", "PNGBYTES")

	if err := run(env, "media", "add", file, "--key", "my-diagram"); err != nil {
		t.Fatalf("media add: %v", err)
	}

	// The three legs happen in order: reserve, PUT the bytes, confirm.
	reserve := fake.lastExact(t, http.MethodPost, "/admin/v1/media")
	if reserve.Body["key"] != "my-diagram" {
		t.Errorf("reserve key = %v, want my-diagram", reserve.Body["key"])
	}
	if reserve.Body["content_type"] != "image/png" {
		t.Errorf("reserve content_type = %v, want image/png", reserve.Body["content_type"])
	}
	if got := string(fake.uploaded["my-diagram"]); got != "PNGBYTES" {
		t.Errorf("uploaded bytes = %q, want the file bytes", got)
	}
	if fake.count(http.MethodPost, "/admin/v1/media/my-diagram/confirm") != 1 {
		t.Errorf("want exactly one confirm, got %d", fake.count(http.MethodPost, "/admin/v1/media/my-diagram/confirm"))
	}
	if fake.media["my-diagram"]["state"] != "available" {
		t.Errorf("record state = %v, want available after confirm", fake.media["my-diagram"]["state"])
	}

	// The exact markdown reference is both printed and copied.
	want := "![](https://cdn.mihirsingh.dev/media/my-diagram)"
	if !strings.Contains(out.String(), want) {
		t.Errorf("stdout = %q, want the markdown reference %q", out.String(), want)
	}
	if clip.copied != want {
		t.Errorf("clipboard = %q, want %q", clip.copied, want)
	}
}

// broom must never PUT before it has reserved: the bytes go to the URL the
// reservation hands back, so a confirm only ever follows a real upload.
func TestMediaAddOrdersReserveBeforeUpload(t *testing.T) {
	fake := newMediaFake(t)
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Copy = (&fakeClipboard{}).copy
	file := writeTempFile(t, "pic.png", "x")

	if err := run(env, "media", "add", file, "--key", "k"); err != nil {
		t.Fatalf("media add: %v", err)
	}
	var order []string
	for _, req := range fake.requests {
		switch {
		case req.Method == http.MethodPost && req.Path == "/admin/v1/media":
			order = append(order, "reserve")
		case req.Method == http.MethodPut:
			order = append(order, "upload")
		case strings.HasSuffix(req.Path, "/confirm"):
			order = append(order, "confirm")
		}
	}
	if strings.Join(order, ",") != "reserve,upload,confirm" {
		t.Errorf("request order = %v, want reserve,upload,confirm", order)
	}
}

func TestMediaAddMintedKeyWhenOmitted(t *testing.T) {
	fake := newMediaFake(t)
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Copy = (&fakeClipboard{}).copy
	file := writeTempFile(t, "pic.png", "x")

	if err := run(env, "media", "add", file); err != nil {
		t.Fatalf("media add: %v", err)
	}
	reserve := fake.lastExact(t, http.MethodPost, "/admin/v1/media")
	if _, sent := reserve.Body["key"]; sent {
		t.Errorf("omitted --key must not send a key, body carried %v", reserve.Body)
	}
	if !strings.Contains(out.String(), "minted-random-key") {
		t.Errorf("stdout = %q, want the custodian-minted key in the reference", out.String())
	}
}

func TestMediaAddDuplicateKeyReportedLegibly(t *testing.T) {
	fake := newMediaFake(t)
	fake.taken["taken-key"] = true
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Copy = (&fakeClipboard{}).copy
	file := writeTempFile(t, "pic.png", "x")

	err := run(env, "media", "add", file, "--key", "taken-key")
	if err == nil {
		t.Fatal("a duplicate key must fail, not silently overwrite")
	}
	if !strings.Contains(err.Error(), "taken-key") || !strings.Contains(err.Error(), "already taken") {
		t.Errorf("error = %q, want a legible taken-key message", err.Error())
	}
	// No overwrite: nothing was uploaded and no confirm was attempted.
	if len(fake.uploaded) != 0 {
		t.Errorf("a rejected reserve must not upload, uploaded %v", fake.uploaded)
	}
	if fake.count(http.MethodPost, "/admin/v1/media/taken-key/confirm") != 0 {
		t.Error("a rejected reserve must not confirm")
	}
}

func TestMediaLsListsAndSearches(t *testing.T) {
	fake := newMediaFake(t)
	fake.seedMedia("hero-shot", "available")
	fake.seedMedia("footer-logo", "available")
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "media", "ls"); err != nil {
		t.Fatalf("media ls: %v", err)
	}
	if !strings.Contains(out.String(), "hero-shot") || !strings.Contains(out.String(), "footer-logo") {
		t.Errorf("stdout = %q, want both seeded keys", out.String())
	}

	// A query reaches the wire and narrows the listing.
	env2, out2, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	if err := run(env2, "media", "ls", "hero"); err != nil {
		t.Fatalf("media ls hero: %v", err)
	}
	search := fake.last(t, http.MethodGet, "/admin/v1/media")
	if !strings.Contains(search.Query, "q=hero") {
		t.Errorf("query = %q, want q=hero", search.Query)
	}
	if strings.Contains(out2.String(), "footer-logo") {
		t.Errorf("stdout = %q, search should narrow away footer-logo", out2.String())
	}
}

func TestMediaRmWarnsWhenReferencedThenDeletesOnConfirm(t *testing.T) {
	fake := newMediaFake(t)
	fake.seedMedia("used-image", "available")
	fake.seedLog("live-post", "intro\n![](https://cdn.mihirsingh.dev/media/used-image)\noutro")
	env, out, errBuf, _ := testEnv(t, "y\n", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "media", "rm", "used-image"); err != nil {
		t.Fatalf("media rm: %v", err)
	}
	if !strings.Contains(errBuf.String(), "live-post") {
		t.Errorf("stderr = %q, want the referencing post named in the warning", errBuf.String())
	}
	if fake.count(http.MethodDelete, "/admin/v1/media/used-image") != 1 {
		t.Errorf("confirmed rm should delete, got %d deletes", fake.count(http.MethodDelete, "/admin/v1/media/used-image"))
	}
	if !strings.Contains(out.String(), "Deleted used-image") {
		t.Errorf("stdout = %q, want the deleted line", out.String())
	}
}

func TestMediaRmAbortsWhenReferencedAndDeclined(t *testing.T) {
	fake := newMediaFake(t)
	fake.seedMedia("used-image", "available")
	fake.seedLog("live-post", "![](https://cdn.mihirsingh.dev/media/used-image)")
	env, out, _, _ := testEnv(t, "n\n", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "media", "rm", "used-image"); err != nil {
		t.Fatalf("media rm: %v", err)
	}
	if fake.count(http.MethodDelete, "/admin/v1/media/used-image") != 0 {
		t.Error("a declined rm must not delete")
	}
	if !strings.Contains(out.String(), "Left used-image in place") {
		t.Errorf("stdout = %q, want the kept-in-place note", out.String())
	}
}

func TestMediaRmDeletesUnreferencedWithoutPrompt(t *testing.T) {
	fake := newMediaFake(t)
	fake.seedMedia("orphan", "available")
	fake.seedLog("live-post", "no image here")
	env, out, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})

	if err := run(env, "media", "rm", "orphan"); err != nil {
		t.Fatalf("media rm: %v", err)
	}
	if fake.count(http.MethodDelete, "/admin/v1/media/orphan") != 1 {
		t.Errorf("unreferenced rm should delete once, got %d", fake.count(http.MethodDelete, "/admin/v1/media/orphan"))
	}
	if !strings.Contains(out.String(), "Deleted orphan") {
		t.Errorf("stdout = %q, want the deleted line", out.String())
	}
}

// broom must never carry an AWS credential — the presigned URL is the whole
// authorisation. The upload PUT therefore sends no Authorization header.
func TestMediaAddUploadCarriesNoAWSCredential(t *testing.T) {
	fake := newMediaFake(t)
	env, _, _, _ := testEnv(t, "", map[string]string{"BROOM_URL": fake.server.URL, "BROOM_TOKEN": "t"})
	env.Copy = (&fakeClipboard{}).copy
	file := writeTempFile(t, "pic.png", "x")

	if err := run(env, "media", "add", file, "--key", "k"); err != nil {
		t.Fatalf("media add: %v", err)
	}
	// The presigned URL is the whole authorisation — the byte path carries no
	// bearer token or AWS credential of its own.
	if fake.uploadAuth != "" {
		t.Errorf("upload carried Authorization %q, want none", fake.uploadAuth)
	}
}
