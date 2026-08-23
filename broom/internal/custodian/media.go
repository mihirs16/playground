package custodian

import (
	"context"

	"github.com/mihirs16/playground/broom/internal/apiclient"
)

// ReserveMedia reserves a pending media record and returns custodian's
// reservation: the key, the presigned S3 upload URL broom PUTs bytes to, and the
// public CDN url. A key already in use comes back as an APIError carrying
// custodian's media_key_taken code, never a silent overwrite.
func (c *Client) ReserveMedia(ctx context.Context, body apiclient.MediaReserve) (*apiclient.MediaReservation, error) {
	resp, err := c.api.ReserveMediaWithResponse(ctx, body)
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON201 != nil {
		return resp.JSON201, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// ConfirmMedia asks custodian to HEAD S3 and flip the reserved record to
// available, returning the confirmed record. It is the third leg of the
// reserve → upload → confirm flow, run only after the bytes have landed.
func (c *Client) ConfirmMedia(ctx context.Context, key string) (*apiclient.Media, error) {
	resp, err := c.api.ConfirmMediaWithResponse(ctx, key)
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// GetMedia fetches a single media record by key, so a command can learn its
// public url — the reference form a post body would carry.
func (c *Client) GetMedia(ctx context.Context, key string) (*apiclient.Media, error) {
	resp, err := c.api.GetMediaWithResponse(ctx, key)
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// ListMedia lists and searches existing media so an asset can be found and
// reused instead of re-uploaded. An empty query lists everything; a non-empty
// one narrows to matches on the key.
func (c *Client) ListMedia(ctx context.Context, query string) (*apiclient.MediaList, error) {
	params := &apiclient.ListMediaParams{}
	if query != "" {
		params.Q = &query
	}
	resp, err := c.api.ListMediaWithResponse(ctx, params)
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// DeleteMedia removes a media record and its bytes — custodian holds the only S3
// credentials, so it deletes the object too. Custodian answers a successful
// delete with no content.
func (c *Client) DeleteMedia(ctx context.Context, key string) error {
	resp, err := c.api.DeleteMediaWithResponse(ctx, key)
	if err != nil {
		return &TransportError{URL: c.baseURL, Err: err}
	}
	if status := resp.StatusCode(); status >= 200 && status < 300 {
		return nil
	}
	return c.errorFrom(resp.StatusCode(), resp.Body)
}

// IsMediaKeyTaken reports whether err is custodian rejecting a reserve because
// the key is already in use, so a command can surface the conflict legibly
// rather than as a raw status.
func IsMediaKeyTaken(err error) bool {
	return codeOf(err) == "media_key_taken"
}
