// Package client is broom's single transport to custodian. It holds no
// hand-rolled HTTP: every call goes through the generated OpenAPI client, with
// the admin bearer token attached to each request. Non-success responses are
// classified once, here, into the problem package's rejected-vs-down split so
// no caller re-implements that judgement.
package client

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/mihirs16/playground/broom/internal/apiclient"
	"github.com/mihirs16/playground/broom/internal/config"
	"github.com/mihirs16/playground/broom/internal/problem"
)

// ErrNotLoggedIn is returned before any request is made when no token is
// configured. It is deliberately distinct from a custodian rejection: broom
// never sends an unauthenticated admin call just to be told no.
var ErrNotLoggedIn = errors.New("not logged in — run `broom login` (or set BROOM_TOKEN)")

// Client is broom's authenticated handle to one custodian.
type Client struct {
	api   *apiclient.Client
	token string
}

// New builds a client for the resolved config. An empty token is allowed here
// — the caller decides whether a given command tolerates it — but a request
// made without one fails fast with ErrNotLoggedIn rather than a 401 round-trip.
func New(cfg config.Config, httpClient apiclient.HttpRequestDoer) (*Client, error) {
	opts := []apiclient.ClientOption{
		apiclient.WithRequestEditorFn(bearer(cfg.Token)),
	}
	if httpClient != nil {
		opts = append(opts, apiclient.WithHTTPClient(httpClient))
	}
	api, err := apiclient.NewClient(cfg.URL, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{api: api, token: cfg.Token}, nil
}

// Verify makes the single authenticated call login relies on: it lists admin
// logs with the smallest possible page and reports whether the token was
// accepted. A transport failure surfaces as custodian-down; a rejection
// surfaces with custodian's stable code so a bad token reads as "token
// rejected", never as a content bug.
func (c *Client) Verify(ctx context.Context) error {
	if c.token == "" {
		return ErrNotLoggedIn
	}
	limit := 1
	resp, err := c.api.ListAdminLogs(ctx, &apiclient.ListAdminLogsParams{Limit: &limit})
	if err != nil {
		return problem.Transport(err)
	}
	return classify(resp)
}

// classify reads a response to completion and turns a non-2xx status into the
// appropriate problem error, closing the body either way.
func classify(resp *http.Response) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return problem.Transport(err)
	}
	return problem.FromResponse(resp, body)
}

func bearer(token string) apiclient.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return nil
	}
}
