// Package problem turns a custodian response into the message broom prints. It
// draws the line the author cares about most: a network / custodian-down
// failure (broom never reached a custodian, or reached something that did not
// answer in the RFC 9457 problem+json contract) is not the same as a request
// custodian understood and rejected. The first is an infrastructure problem;
// the second is a content problem, and only the second carries a stable `code`
// and field-errors worth reading.
package problem

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mihirs16/playground/broom/internal/apiclient"
)

// ContentType is the media type custodian stamps on every problem document.
const ContentType = "application/problem+json"

// Rejected is a request custodian received and refused. It wraps the parsed
// problem document so callers can branch on the stable Code.
type Rejected struct {
	Status  int
	Problem apiclient.Problem
}

// Code is the stable machine-readable error code custodian assigned. Callers
// branch on this, never on Status or the human detail.
func (r *Rejected) Code() string { return r.Problem.Code }

func (r *Rejected) Error() string { return Render(r) }

// Down is a failure to get a usable answer out of custodian at all: the
// transport errored, or the response was not a problem document broom could
// parse. It is deliberately distinct from Rejected so broom never blames the
// author's content for what is an infrastructure fault.
type Down struct {
	// Cause is the underlying transport or decode failure, if any.
	Cause error
}

func (d *Down) Error() string { return Render(d) }
func (d *Down) Unwrap() error { return d.Cause }

// FromResponse classifies a non-success custodian response. The caller has
// already read the body and confirmed the status is an error; FromResponse
// decides whether it is a well-formed rejection (Rejected) or an unparseable
// answer that means custodian is effectively down (Down).
func FromResponse(resp *http.Response, body []byte) error {
	if !isProblemContentType(resp.Header.Get("Content-Type")) {
		return &Down{Cause: fmt.Errorf("custodian answered %s without a problem document", resp.Status)}
	}
	var doc apiclient.Problem
	if err := json.Unmarshal(body, &doc); err != nil {
		return &Down{Cause: fmt.Errorf("custodian's error response was not valid problem+json: %w", err)}
	}
	if doc.Code == "" {
		return &Down{Cause: fmt.Errorf("custodian answered %s with no error code", resp.Status)}
	}
	return &Rejected{Status: resp.StatusCode, Problem: doc}
}

// Transport wraps a failure to reach custodian at all — a dial timeout, DNS
// failure, or connection refused — as a Down.
func Transport(err error) error {
	return &Down{Cause: err}
}

// Render is the single place broom turns any of these failures into text. It
// leads with the human detail, then — for a rejection — surfaces the stable
// code and any per-field errors so the author can fix the offending fields.
func Render(err error) string {
	switch e := err.(type) {
	case *Down:
		if e.Cause != nil {
			return fmt.Sprintf("cannot reach custodian: %v", e.Cause)
		}
		return "cannot reach custodian"
	case *Rejected:
		return renderRejected(e)
	default:
		return err.Error()
	}
}

func renderRejected(r *Rejected) string {
	var b strings.Builder

	detail := r.Problem.Title
	if r.Problem.Detail != nil && *r.Problem.Detail != "" {
		detail = *r.Problem.Detail
	}
	fmt.Fprintf(&b, "custodian rejected the request: %s", detail)
	if r.Problem.Code != "" {
		fmt.Fprintf(&b, " (%s)", r.Problem.Code)
	}

	if r.Problem.Errors != nil {
		for _, fieldErr := range *r.Problem.Errors {
			fmt.Fprintf(&b, "\n  - %s: %s", fieldErr.Field, fieldErr.Message)
		}
	}
	return b.String()
}

func isProblemContentType(header string) bool {
	if header == "" {
		return false
	}
	mediaType := header
	if semi := strings.IndexByte(header, ';'); semi >= 0 {
		mediaType = header[:semi]
	}
	return strings.EqualFold(strings.TrimSpace(mediaType), ContentType)
}
