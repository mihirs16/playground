package custodian

import (
	"context"
	"errors"

	"github.com/mihirs16/playground/broom/internal/apiclient"
)

// CreateLog creates a log as an unlisted draft and returns the created record.
// A slug already in use comes back as an APIError carrying custodian's
// slug_conflict code, which the caller surfaces as a re-prompt.
func (c *Client) CreateLog(ctx context.Context, body apiclient.LogCreate) (*apiclient.Log, error) {
	resp, err := c.api.CreateLogWithResponse(ctx, body)
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON201 != nil {
		return resp.JSON201, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// PatchLog partially updates a log — a body edit, a state toggle, or a slug
// rename are all this one call — and returns the updated record (which echoes
// the current slug).
func (c *Client) PatchLog(ctx context.Context, slug string, body apiclient.LogPatch) (*apiclient.Log, error) {
	resp, err := c.api.PatchLogWithResponse(ctx, slug, body)
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// ListLogs fetches the author's logs from the admin surface, where every state
// is visible — including the unlisted drafts the public index hides. A nil state
// lists all states; a non-nil state narrows to just that one.
func (c *Client) ListLogs(ctx context.Context, state *apiclient.LogState) (*apiclient.LogIndex, error) {
	resp, err := c.api.ListAdminLogsWithResponse(ctx, &apiclient.ListAdminLogsParams{State: state})
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// DeleteLog removes a log entirely, so a draft or a retired post can be taken
// down. Custodian answers a successful delete with no content.
func (c *Client) DeleteLog(ctx context.Context, slug string) error {
	resp, err := c.api.DeleteLogWithResponse(ctx, slug)
	if err != nil {
		return &TransportError{URL: c.baseURL, Err: err}
	}
	if status := resp.StatusCode(); status >= 200 && status < 300 {
		return nil
	}
	return c.errorFrom(resp.StatusCode(), resp.Body)
}

// GetLog fetches a single log by slug, body included, in any state. It reads the
// public endpoint because an unlisted draft is reachable at its real URL, so
// there is nothing admin-only to fetch — this is how `logs edit` pulls the
// current body.
func (c *Client) GetLog(ctx context.Context, slug string) (*apiclient.Log, error) {
	resp, err := c.api.GetPublicLogWithResponse(ctx, slug)
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// IsSlugConflict reports whether err is custodian rejecting a create or rename
// because the slug is already taken, so a command can re-prompt in-flow rather
// than surfacing a raw conflict.
func IsSlugConflict(err error) bool {
	return codeOf(err) == "slug_conflict"
}

// codeOf extracts the stable problem code from an APIError, or "" for anything
// else (a transport failure, a nil error).
func codeOf(err error) string {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return ""
	}
	return apiErr.Code()
}
