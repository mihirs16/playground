package problem_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/mihirs16/playground/broom/internal/problem"
)

func problemResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{problem.ContentType}},
	}
}

func TestFromResponseParsesRejectionAndCode(t *testing.T) {
	body := []byte(`{"title":"Unauthorized","status":401,"code":"unauthorized","detail":"a valid admin bearer token is required"}`)

	err := problem.FromResponse(problemResponse(http.StatusUnauthorized), body)

	var rejected *problem.Rejected
	if !errors.As(err, &rejected) {
		t.Fatalf("got %T, want *problem.Rejected", err)
	}
	if rejected.Code() != "unauthorized" {
		t.Errorf("code = %q, want unauthorized", rejected.Code())
	}
	if !strings.Contains(err.Error(), "a valid admin bearer token is required") {
		t.Errorf("render %q, want the detail", err.Error())
	}
}

func TestFromResponseListsFieldErrors(t *testing.T) {
	body := []byte(`{"title":"Unprocessable Entity","status":422,"code":"validation_failed",` +
		`"detail":"the request body failed validation",` +
		`"errors":[{"field":"slug","message":"must be kebab-case"},{"field":"title","message":"required"}]}`)

	err := problem.FromResponse(problemResponse(http.StatusUnprocessableEntity), body)

	rendered := problem.Render(err)
	for _, want := range []string{"slug: must be kebab-case", "title: required", "validation_failed"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("render missing %q; got:\n%s", want, rendered)
		}
	}
}

func TestNonProblemBodyIsTreatedAsDown(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Status:     "502 Bad Gateway",
		Header:     http.Header{"Content-Type": []string{"text/html"}},
	}

	err := problem.FromResponse(resp, []byte("<html>gateway timeout</html>"))

	var down *problem.Down
	if !errors.As(err, &down) {
		t.Fatalf("got %T, want *problem.Down for a non-problem body", err)
	}
}

func TestProblemContentTypeWithCharsetStillParses(t *testing.T) {
	resp := problemResponse(http.StatusConflict)
	resp.Header.Set("Content-Type", problem.ContentType+"; charset=utf-8")
	body := []byte(`{"title":"Conflict","status":409,"code":"slug_taken","detail":"that slug is in use"}`)

	err := problem.FromResponse(resp, body)

	var rejected *problem.Rejected
	if !errors.As(err, &rejected) {
		t.Fatalf("got %T, want *problem.Rejected", err)
	}
	if rejected.Code() != "slug_taken" {
		t.Errorf("code = %q, want slug_taken", rejected.Code())
	}
}

func TestTransportErrorRendersAsCustodianDown(t *testing.T) {
	err := problem.Transport(errors.New("dial tcp: connection refused"))

	var down *problem.Down
	if !errors.As(err, &down) {
		t.Fatalf("got %T, want *problem.Down", err)
	}
	rendered := problem.Render(err)
	if !strings.Contains(rendered, "cannot reach custodian") {
		t.Errorf("render %q, want a custodian-down message", rendered)
	}
	if !strings.Contains(rendered, "connection refused") {
		t.Errorf("render %q, want the underlying cause", rendered)
	}
}

// A well-formed problem+json that omits the stable code is unusable for
// broom's branch-on-code contract, so it is a custodian fault, not content.
func TestProblemWithoutCodeIsDown(t *testing.T) {
	body := []byte(`{"title":"Bad Gateway","status":502}`)

	err := problem.FromResponse(problemResponse(http.StatusBadGateway), body)

	var down *problem.Down
	if !errors.As(err, &down) {
		t.Fatalf("got %T, want *problem.Down for a codeless problem", err)
	}
}
