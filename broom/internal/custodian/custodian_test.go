package custodian

import (
	"strings"
	"testing"

	"github.com/mihirs16/playground/broom/internal/apiclient"
)

func strptr(s string) *string { return &s }

func TestAPIErrorRendersDetail(t *testing.T) {
	err := &APIError{Status: 409, Problem: &apiclient.Problem{
		Code:   "slug_conflict",
		Title:  "Conflict",
		Detail: strptr("a log with that slug already exists"),
	}}
	if got := err.Error(); got != "a log with that slug already exists" {
		t.Errorf("Error() = %q, want the detail string", got)
	}
}

func TestAPIErrorBranchesOnCode(t *testing.T) {
	err := &APIError{Status: 401, Problem: &apiclient.Problem{
		Code:   "unauthorized",
		Title:  "Unauthorized",
		Detail: strptr("a valid admin bearer token is required"),
	}}
	got := err.Error()
	if !strings.HasPrefix(got, "not logged in or token rejected:") {
		t.Errorf("Error() = %q, want the unauthorized headline", got)
	}
	if !strings.Contains(got, "a valid admin bearer token is required") {
		t.Errorf("Error() = %q, want the detail retained", got)
	}
}

func TestAPIErrorFallsBackToTitleWithoutDetail(t *testing.T) {
	err := &APIError{Status: 500, Problem: &apiclient.Problem{Code: "internal", Title: "Internal Server Error"}}
	if got := err.Error(); got != "Internal Server Error" {
		t.Errorf("Error() = %q, want the title", got)
	}
}

func TestAPIErrorListsFieldErrors(t *testing.T) {
	err := &APIError{Status: 422, Problem: &apiclient.Problem{
		Code:   "validation_failed",
		Detail: strptr("the request body failed validation"),
		Errors: &[]apiclient.FieldError{
			{Field: "title", Message: "is required"},
			{Field: "slug", Message: "must be kebab-case"},
		},
	}}
	got := err.Error()
	for _, want := range []string{
		"the request body failed validation",
		"- title: is required",
		"- slug: must be kebab-case",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Error() = %q, missing %q", got, want)
		}
	}
}

func TestAPIErrorFrozenSlugReadsAsGuidance(t *testing.T) {
	err := &APIError{Status: 409, Problem: &apiclient.Problem{
		Code:   "slug_frozen_while_listed",
		Title:  "Conflict",
		Detail: strptr("a listed log's slug cannot change"),
	}}
	got := err.Error()
	if !strings.Contains(got, "published links are frozen") {
		t.Errorf("Error() = %q, want the frozen-links guidance", got)
	}
	if !strings.Contains(got, "a listed log's slug cannot change") {
		t.Errorf("Error() = %q, want the detail retained", got)
	}
}

func TestIsUnauthorized(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"401 status", &APIError{Status: 401, Problem: &apiclient.Problem{Code: "unauthorized"}}, true},
		{"unauthorized code only", &APIError{Status: 403, Problem: &apiclient.Problem{Code: "unauthorized"}}, true},
		{"other api error", &APIError{Status: 409, Problem: &apiclient.Problem{Code: "slug_conflict"}}, false},
		{"transport error", &TransportError{URL: "x", Err: errString("boom")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsUnauthorized(tc.err); got != tc.want {
				t.Errorf("IsUnauthorized = %v, want %v", got, tc.want)
			}
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }
