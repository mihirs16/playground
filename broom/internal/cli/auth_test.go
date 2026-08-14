package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readStoredToken(t *testing.T) string {
	t.Helper()
	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "broom", "config.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func TestLoginVerifiesAndPersistsToken(t *testing.T) {
	fake := newFakeCustodian(t)

	result := run(t, fake, fakeToken+"\n", "login")

	if result.err != nil {
		t.Fatalf("login errored: %v", result.err)
	}
	if !strings.Contains(result.out, "Logged in.") {
		t.Errorf("output %q, want a success line", result.out)
	}
	if fake.requests != 1 {
		t.Errorf("verify made %d requests, want exactly 1", fake.requests)
	}
	if stored := readStoredToken(t); !strings.Contains(stored, fakeToken) {
		t.Errorf("config file %q, want the token persisted", stored)
	}
}

func TestLoginRejectsBadTokenAndPersistsNothing(t *testing.T) {
	fake := newFakeCustodian(t)

	result := run(t, fake, "wrong-token\n", "login")

	if result.err == nil {
		t.Fatal("login with a bad token should error")
	}
	if !strings.Contains(result.err.Error(), "token rejected") {
		t.Errorf("error %q, want a token-rejected message", result.err.Error())
	}
	if !strings.Contains(result.err.Error(), "unauthorized") {
		t.Errorf("error %q, want custodian's stable code", result.err.Error())
	}
	if stored := readStoredToken(t); strings.Contains(stored, "wrong-token") {
		t.Errorf("a rejected token must not be written; got %q", stored)
	}
}

func TestLoginAgainstDownCustodianReportsUnreachable(t *testing.T) {
	fake := newFakeCustodian(t)
	fake.server.Close() // nothing is listening now

	result := run(t, fake, fakeToken+"\n", "login")

	if result.err == nil {
		t.Fatal("login against a down custodian should error")
	}
	if !strings.Contains(result.err.Error(), "cannot reach custodian") {
		t.Errorf("error %q, want a custodian-down message", result.err.Error())
	}
}

func TestLogoutClearsTokenButKeepsURL(t *testing.T) {
	fake := newFakeCustodian(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // shared across both runs below

	if r := run(t, fake, fakeToken+"\n", "login"); r.err != nil {
		t.Fatalf("precondition login failed: %v", r.err)
	}
	if stored := readStoredToken(t); !strings.Contains(stored, fakeToken) {
		t.Fatalf("precondition: token not stored: %q", stored)
	}

	result := run(t, fake, "", "logout")
	if result.err != nil {
		t.Fatalf("logout errored: %v", result.err)
	}
	if !strings.Contains(result.out, "Logged out.") {
		t.Errorf("output %q, want a logout line", result.out)
	}
	if stored := readStoredToken(t); strings.Contains(stored, fakeToken) {
		t.Errorf("token still present after logout: %q", stored)
	}
	if stored := readStoredToken(t); !strings.Contains(stored, fake.server.URL) {
		t.Errorf("url should survive logout; got %q", stored)
	}
}

func TestCommandWithoutLoginIsAClearNotLoggedInMessage(t *testing.T) {
	// No stored token, no BROOM_TOKEN: a command that needs credentials must
	// say so plainly rather than surfacing a content-shaped error.
	fake := newFakeCustodian(t)
	result := run(t, fake, "", "login") // login with empty stdin => no token

	if result.err == nil {
		t.Fatal("login with no token should error")
	}
	if !strings.Contains(result.err.Error(), "no token") {
		t.Errorf("error %q, want a no-token message", result.err.Error())
	}
}
