// Package custodian wraps the generated OpenAPI client with the two things every
// broom command needs on top of raw transport: bearer authentication, and a
// single place that turns a custodian response into a legible error.
//
// It draws a hard line between two failure modes that must never be conflated
// (user story 29): custodian was unreachable (a TransportError) versus custodian
// answered and said no (an APIError rendered from its RFC 9457 problem+json).
package custodian

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mihirs16/playground/broom/internal/apiclient"
	"github.com/mihirs16/playground/broom/internal/config"
)

// Client is broom's authenticated handle on a custodian.
type Client struct {
	api     apiclient.ClientWithResponsesInterface
	baseURL string
}

// New builds a client that signs every request with the config's bearer token
// and talks to the config's URL.
func New(cfg config.Config) (*Client, error) {
	auth := apiclient.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		if cfg.Token != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.Token)
		}
		return nil
	})
	api, err := apiclient.NewClientWithResponses(cfg.URL, auth)
	if err != nil {
		return nil, err
	}
	return &Client{api: api, baseURL: cfg.URL}, nil
}

// VerifyToken confirms the configured token is accepted, using a single
// authenticated call (a one-item admin log listing). It is how `login` learns
// immediately whether a credential works rather than at first write.
func (c *Client) VerifyToken(ctx context.Context) error {
	limit := 1
	resp, err := c.api.ListAdminLogsWithResponse(ctx, &apiclient.ListAdminLogsParams{Limit: &limit})
	if err != nil {
		return &TransportError{URL: c.baseURL, Err: err}
	}
	if resp.StatusCode() == http.StatusOK {
		return nil
	}
	return c.errorFrom(resp.StatusCode(), resp.Body)
}

// errorFrom turns a non-success response into an APIError, parsing custodian's
// problem+json body when present and falling back to the bare status otherwise.
func (c *Client) errorFrom(status int, body []byte) error {
	if p := parseProblem(body); p != nil {
		return &APIError{Status: status, Problem: p}
	}
	return &APIError{Status: status, Problem: &apiclient.Problem{
		Status: status,
		Title:  http.StatusText(status),
		Code:   "unknown",
	}}
}

// parseProblem decodes an RFC 9457 problem document, returning nil if the body
// is not one (no stable code means it is not custodian's problem shape).
func parseProblem(body []byte) *apiclient.Problem {
	var p apiclient.Problem
	if err := json.Unmarshal(body, &p); err != nil {
		return nil
	}
	if p.Code == "" {
		return nil
	}
	return &p
}

// TransportError means custodian could not be reached at all — a network
// failure or a down custodian, never a rejected request.
type TransportError struct {
	URL string
	Err error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("cannot reach custodian at %s: %v", e.URL, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

// APIError is a request custodian answered and rejected, carrying its parsed
// problem document so it renders as guidance rather than a raw HTTP status.
type APIError struct {
	Status  int
	Problem *apiclient.Problem
}

// Code is the stable machine-readable error code commands branch on.
func (e *APIError) Code() string {
	if e.Problem == nil {
		return ""
	}
	return e.Problem.Code
}

func (e *APIError) Error() string {
	if e.Problem == nil {
		return http.StatusText(e.Status)
	}

	var b strings.Builder
	b.WriteString(headline(e.Problem))
	if e.Problem.Errors != nil {
		for _, fe := range *e.Problem.Errors {
			fmt.Fprintf(&b, "\n  - %s: %s", fe.Field, fe.Message)
		}
	}
	return b.String()
}

// headline turns a problem into its leading message, branching on the stable
// code so a rejection reads as guidance. Unrecognised codes fall back to the
// custodian-supplied detail (or title), which is always safe to show verbatim.
func headline(p *apiclient.Problem) string {
	detail := p.Title
	if p.Detail != nil && *p.Detail != "" {
		detail = *p.Detail
	}
	switch p.Code {
	case "unauthorized":
		return "not logged in or token rejected: " + detail
	case "slug_frozen_while_listed":
		return "published links are frozen — unpublish before renaming: " + detail
	default:
		return detail
	}
}

// IsUnauthorized reports whether err is custodian rejecting the credential, so
// callers can render the not-logged-in / token-rejected message instead of a
// generic failure.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == http.StatusUnauthorized || apiErr.Code() == "unauthorized"
}
