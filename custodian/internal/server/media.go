package server

import (
	"crypto/rand"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/mihirs16/playground/custodian/internal/api"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

// uploadWindow is how long a presigned upload URL — and so the reservation it
// belongs to — stays valid. broom must PUT the bytes and confirm within it.
const uploadWindow = 15 * time.Minute

// mediaKeyGroups is the shape of a generated key: three random kebab-case
// groups, so an omitted key is still a legible, collision-resistant slug.
const (
	mediaKeyGroups   = 3
	mediaKeyGroupLen = 4
)

// ReserveMedia reserves a pending media record and hands broom a presigned S3
// PUT plus the public CDN url. The bytes go straight from broom to the bucket;
// custodian never touches them. A key already reserved is a media_key_taken
// conflict, never a silent overwrite.
func (h *handlers) ReserveMedia(w http.ResponseWriter, r *http.Request) {
	var body api.MediaReserve
	if !decodeJSON(w, r, &body) {
		return
	}
	if fields := validateReserve(body); len(fields) > 0 {
		writeValidationProblem(w, fields)
		return
	}

	key, err := reservedKey(body.Key)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not generate a media key")
		return
	}

	uploadURL, err := h.edges.ObjectStore.PresignPut(r.Context(), key, body.ContentType, uploadWindow)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not presign the upload")
		return
	}

	expiresAt := time.Now().UTC().Add(uploadWindow)
	record := storage.Media{
		Key:         key,
		State:       string(api.Pending),
		ContentType: body.ContentType,
		URL:         h.publicURL(key),
		ExpiresAt:   &expiresAt,
	}

	created, err := h.db.CreateMedia(r.Context(), record)
	if errors.Is(err, storage.ErrMediaKeyTaken) {
		writeProblem(w, http.StatusConflict, "media_key_taken", "a media record with that key already exists")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not reserve the media record")
		return
	}

	writeJSON(w, http.StatusCreated, api.MediaReservation{
		Key:       created.Key,
		UploadUrl: uploadURL,
		Url:       created.URL,
		ExpiresAt: *created.ExpiresAt,
	})
}

// ConfirmMedia flips a reserved record to available, but only after a HEAD
// against S3 proves the bytes landed — so no record is ever available without
// real bytes behind it.
func (h *handlers) ConfirmMedia(w http.ResponseWriter, r *http.Request, key api.MediaKey) {
	if _, err := h.db.GetMedia(r.Context(), key); err != nil {
		h.writeMediaReadError(w, err)
		return
	}

	present, err := h.edges.ObjectStore.HeadObject(r.Context(), key)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not check the uploaded object")
		return
	}
	if !present {
		writeProblem(w, http.StatusConflict, "media_bytes_missing", "no uploaded bytes found for that key; upload before confirming")
		return
	}

	confirmed, err := h.db.SetMediaAvailable(r.Context(), key)
	if err != nil {
		h.writeMediaReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMedia(confirmed))
}

// ListMedia lists and searches media so an existing asset can be found and
// reused rather than re-uploaded. The q term matches against the key.
func (h *handlers) ListMedia(w http.ResponseWriter, r *http.Request, params api.ListMediaParams) {
	query := storage.MediaQuery{
		Limit:  clampLimit(params.Limit),
		Offset: clampOffset(params.Offset),
	}
	if params.Q != nil {
		query.Search = *params.Q
	}

	total, records, err := h.db.ListMedia(r.Context(), query)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not read the media index")
		return
	}

	list := api.MediaList{Total: total, Items: make([]api.Media, 0, len(records))}
	for _, record := range records {
		list.Items = append(list.Items, toMedia(record))
	}
	writeJSON(w, http.StatusOK, list)
}

// GetMedia fetches a single media record by key.
func (h *handlers) GetMedia(w http.ResponseWriter, r *http.Request, key api.MediaKey) {
	record, err := h.db.GetMedia(r.Context(), key)
	if err != nil {
		h.writeMediaReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMedia(record))
}

// DeleteMedia removes a media record. Any pre-delete reference scan is broom's
// courtesy — custodian does not parse log bodies for the url.
func (h *handlers) DeleteMedia(w http.ResponseWriter, r *http.Request, key api.MediaKey) {
	err := h.db.DeleteMedia(r.Context(), key)
	if errors.Is(err, storage.ErrMediaNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "no media with that key")
		return
	}
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "internal", "could not delete the media record")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) writeMediaReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrMediaNotFound) {
		writeProblem(w, http.StatusNotFound, "not_found", "no media with that key")
		return
	}
	writeProblem(w, http.StatusInternalServerError, "internal", "could not read the media record")
}

// publicURL builds a record's extension-free public CDN url from the configured
// base and the key.
func (h *handlers) publicURL(key string) string {
	return strings.TrimRight(h.mediaBaseURL, "/") + "/" + key
}

func validateReserve(body api.MediaReserve) []api.FieldError {
	var fields []api.FieldError
	if body.ContentType == "" {
		fields = append(fields, api.FieldError{Field: "content_type", Message: "content_type is required"})
	}
	if body.Key != nil && !slugPattern.MatchString(*body.Key) {
		fields = append(fields, api.FieldError{Field: "key", Message: "key must be lowercase alphanumeric words joined by hyphens"})
	}
	return fields
}

// reservedKey returns the author's key when given one, or a freshly generated
// kebab-case key when it is omitted.
func reservedKey(provided *string) (string, error) {
	if provided != nil {
		return *provided, nil
	}
	return generateMediaKey()
}

// generateMediaKey builds a random kebab-case key from a lowercase-base32
// alphabet, so it drops straight into the public url with no escaping.
func generateMediaKey() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz234567"

	raw := make([]byte, mediaKeyGroups*mediaKeyGroupLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	groups := make([]string, mediaKeyGroups)
	for g := range groups {
		var b strings.Builder
		for i := 0; i < mediaKeyGroupLen; i++ {
			b.WriteByte(alphabet[int(raw[g*mediaKeyGroupLen+i])%len(alphabet)])
		}
		groups[g] = b.String()
	}
	return strings.Join(groups, "-"), nil
}

func toMedia(record storage.Media) api.Media {
	media := api.Media{
		Key:         record.Key,
		State:       api.MediaState(record.State),
		ContentType: record.ContentType,
		Url:         record.URL,
		CreatedAt:   record.CreatedAt,
	}
	if record.ExpiresAt != nil {
		media.ExpiresAt = record.ExpiresAt
	}
	return media
}
