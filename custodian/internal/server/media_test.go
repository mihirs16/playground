package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestReserveMediaReturnsPresignedUpload covers the reserve contract: a pending
// record lands, and broom gets an upload URL, an extension-free public URL, and
// an expiry.
func TestReserveMediaReturnsPresignedUpload(t *testing.T) {
	h := newHarness(t)

	resp := h.requestJSON(t, http.MethodPost, "/admin/v1/media", adminAuth(), map[string]any{
		"key":          "sunset-over-the-bay",
		"content_type": "image/png",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var reservation struct {
		Key       string `json:"key"`
		UploadUrl string `json:"upload_url"`
		Url       string `json:"url"`
		ExpiresAt string `json:"expires_at"`
	}
	decode(t, resp, &reservation)

	if reservation.Key != "sunset-over-the-bay" {
		t.Fatalf("key = %q, want the author-chosen key", reservation.Key)
	}
	if reservation.UploadUrl == "" {
		t.Fatal("upload_url is empty; broom has nowhere to PUT")
	}
	if want := testCDNBase + "/sunset-over-the-bay"; reservation.Url != want {
		t.Fatalf("url = %q, want %q", reservation.Url, want)
	}
	if strings.Contains(reservation.Url, ".png") {
		t.Fatalf("url = %q, want extension-free", reservation.Url)
	}
	if reservation.ExpiresAt == "" {
		t.Fatal("expires_at is empty; the upload window is unbounded")
	}
	if state, ok := h.mediaState(t, "sunset-over-the-bay"); !ok || state != "pending" {
		t.Fatalf("stored state = %q (present=%v), want pending", state, ok)
	}
}

// TestReserveMediaGeneratesKeyWhenOmitted proves an omitted key gets a random
// kebab-case one custodian generates.
func TestReserveMediaGeneratesKeyWhenOmitted(t *testing.T) {
	h := newHarness(t)

	resp := h.requestJSON(t, http.MethodPost, "/admin/v1/media", adminAuth(), map[string]any{
		"content_type": "image/jpeg",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var reservation struct {
		Key string `json:"key"`
		Url string `json:"url"`
	}
	decode(t, resp, &reservation)

	if reservation.Key == "" {
		t.Fatal("generated key is empty")
	}
	if !kebabCase(reservation.Key) {
		t.Fatalf("generated key = %q, want kebab-case", reservation.Key)
	}
	if reservation.Url != testCDNBase+"/"+reservation.Key {
		t.Fatalf("url = %q, want it built from the generated key", reservation.Url)
	}
}

// TestReserveMediaDuplicateKeyIsConflict proves a second reserve of the same key
// is refused, never a silent overwrite.
func TestReserveMediaDuplicateKeyIsConflict(t *testing.T) {
	h := newHarness(t)

	first := h.requestJSON(t, http.MethodPost, "/admin/v1/media", adminAuth(), map[string]any{
		"key":          "taken-key",
		"content_type": "image/png",
	})
	first.Body.Close()
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first reserve status = %d, want 201", first.StatusCode)
	}

	second := h.requestJSON(t, http.MethodPost, "/admin/v1/media", adminAuth(), map[string]any{
		"key":          "taken-key",
		"content_type": "image/gif",
	})
	defer second.Body.Close()
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("second reserve status = %d, want 409", second.StatusCode)
	}
	assertProblemCode(t, second, "media_key_taken")
}

func TestReserveMediaRejectsBadKey(t *testing.T) {
	h := newHarness(t)

	resp := h.requestJSON(t, http.MethodPost, "/admin/v1/media", adminAuth(), map[string]any{
		"key":          "Not A Key",
		"content_type": "image/png",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

// TestMediaFlowEndToEnd exercises reserve → simulated upload → confirm → HEAD →
// available against the fake S3.
func TestMediaFlowEndToEnd(t *testing.T) {
	h := newHarness(t)

	reserve := h.requestJSON(t, http.MethodPost, "/admin/v1/media", adminAuth(), map[string]any{
		"key":          "hero-shot",
		"content_type": "image/webp",
	})
	reserve.Body.Close()
	if reserve.StatusCode != http.StatusCreated {
		t.Fatalf("reserve status = %d, want 201", reserve.StatusCode)
	}

	// broom completes the presigned PUT; the bytes now exist in the bucket.
	h.objectStore(t).PutBytes("hero-shot")

	confirm := h.request(t, http.MethodPost, "/admin/v1/media/hero-shot/confirm", adminAuth())
	defer confirm.Body.Close()
	if confirm.StatusCode != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200", confirm.StatusCode)
	}
	var record struct {
		State string `json:"state"`
	}
	decode(t, confirm, &record)
	if record.State != "available" {
		t.Fatalf("state = %q, want available after confirm", record.State)
	}
	if state, _ := h.mediaState(t, "hero-shot"); state != "available" {
		t.Fatalf("stored state = %q, want available", state)
	}
}

// TestConfirmWithoutBytesDoesNotFlip proves the invariant: confirm HEADs S3
// first and refuses to flip a record whose bytes never landed.
func TestConfirmWithoutBytesDoesNotFlip(t *testing.T) {
	h := newHarness(t)

	reserve := h.requestJSON(t, http.MethodPost, "/admin/v1/media", adminAuth(), map[string]any{
		"key":          "never-uploaded",
		"content_type": "image/png",
	})
	reserve.Body.Close()

	confirm := h.request(t, http.MethodPost, "/admin/v1/media/never-uploaded/confirm", adminAuth())
	defer confirm.Body.Close()

	if confirm.StatusCode != http.StatusConflict {
		t.Fatalf("confirm status = %d, want 409 (no bytes)", confirm.StatusCode)
	}
	assertProblemCode(t, confirm, "media_bytes_missing")
	if state, _ := h.mediaState(t, "never-uploaded"); state != "pending" {
		t.Fatalf("state = %q, want pending (confirm must not flip)", state)
	}
}

func TestConfirmUnknownKeyIs404(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodPost, "/admin/v1/media/ghost/confirm", adminAuth())
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// TestListAndSearchMedia proves the index lists records and the q term narrows
// it so an existing asset can be found.
func TestListAndSearchMedia(t *testing.T) {
	h := newHarness(t)

	for _, key := range []string{"sunset-beach", "sunset-hills", "city-lights"} {
		resp := h.requestJSON(t, http.MethodPost, "/admin/v1/media", adminAuth(), map[string]any{
			"key":          key,
			"content_type": "image/png",
		})
		resp.Body.Close()
	}

	all := h.request(t, http.MethodGet, "/admin/v1/media", adminAuth())
	defer all.Body.Close()
	var index struct {
		Total int `json:"total"`
	}
	decode(t, all, &index)
	if index.Total != 3 {
		t.Fatalf("total = %d, want 3", index.Total)
	}

	search := h.request(t, http.MethodGet, "/admin/v1/media?q=sunset", adminAuth())
	defer search.Body.Close()
	var found struct {
		Total int `json:"total"`
		Items []struct {
			Key string `json:"key"`
		} `json:"items"`
	}
	decode(t, search, &found)
	if found.Total != 2 || len(found.Items) != 2 {
		t.Fatalf("q=sunset total = %d, items = %d, want 2 each", found.Total, len(found.Items))
	}
}

func TestDeleteMediaRemovesRecord(t *testing.T) {
	h := newHarness(t)

	reserve := h.requestJSON(t, http.MethodPost, "/admin/v1/media", adminAuth(), map[string]any{
		"key":          "doomed-asset",
		"content_type": "image/png",
	})
	reserve.Body.Close()

	del := h.request(t, http.MethodDelete, "/admin/v1/media/doomed-asset", adminAuth())
	defer del.Body.Close()
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", del.StatusCode)
	}
	if _, ok := h.mediaState(t, "doomed-asset"); ok {
		t.Fatal("media record still present after delete")
	}
}

func TestDeleteUnknownMediaIs404(t *testing.T) {
	h := newHarness(t)

	resp := h.request(t, http.MethodDelete, "/admin/v1/media/ghost", adminAuth())
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// kebabCase reports whether s is lowercase alphanumeric words joined by single
// hyphens — the shape every media key, generated or author-chosen, must take.
func kebabCase(s string) bool {
	if s == "" {
		return false
	}
	for _, group := range strings.Split(s, "-") {
		if group == "" {
			return false
		}
		for _, r := range group {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
				return false
			}
		}
	}
	return true
}
