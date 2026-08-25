package custodian

import (
	"context"

	"github.com/mihirs16/playground/broom/internal/apiclient"
)

// GetProfile fetches a single profile record by key, body included. It reads the
// public endpoint — a profile record is served at its real URL — so this is how
// `profile get` and `profile edit` pull the current value.
func (c *Client) GetProfile(ctx context.Context, key string) (*apiclient.Profile, error) {
	resp, err := c.api.GetPublicProfileWithResponse(ctx, key)
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// PutProfile upserts a profile record's body under key and returns the stored
// record. The body is opaque — custodian does not validate its shape — so broom
// round-trips whatever the author saved.
func (c *Client) PutProfile(ctx context.Context, key string, body apiclient.ProfileBody) (*apiclient.Profile, error) {
	resp, err := c.api.PutProfileWithResponse(ctx, key, body)
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// IsNotFound reports whether err is custodian saying no record exists at the
// requested key, so a command can treat a missing record as an empty starting
// point rather than surfacing a raw 404.
func IsNotFound(err error) bool {
	return codeOf(err) == "not_found"
}
