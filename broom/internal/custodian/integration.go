package custodian

import (
	"context"

	"github.com/mihirs16/playground/broom/internal/apiclient"
)

// GetIntegration reads a source's latest stored record — custodian's public,
// last-known-good read. It never forces a poll: it returns whatever the last
// automatic tick stored, including the empty-but-present shape (null data, zero
// timestamp) for a source not yet polled successfully.
func (c *Client) GetIntegration(ctx context.Context, source string) (*apiclient.Integration, error) {
	resp, err := c.api.GetPublicIntegrationWithResponse(ctx, apiclient.Source(source))
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}

// RefreshIntegration forces custodian to poll a source immediately and returns
// the freshly polled record. Unlike the public read, custodian surfaces a failed
// poll as an error here — the operator wants to know a rotated key did not land,
// not last-known-good. The credential itself lives in custodian (resolved from
// its deploy-time environment); broom only asks for the poll.
func (c *Client) RefreshIntegration(ctx context.Context, source string) (*apiclient.Integration, error) {
	resp, err := c.api.RefreshIntegrationWithResponse(ctx, apiclient.Source(source))
	if err != nil {
		return nil, &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, c.errorFrom(resp.StatusCode(), resp.Body)
}
