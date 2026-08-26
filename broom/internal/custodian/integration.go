package custodian

import (
	"context"

	"github.com/mihirs16/playground/broom/internal/apiclient"
)

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
