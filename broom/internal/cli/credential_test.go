package cli_test

import (
	"strings"
	"testing"

	"github.com/mihirs16/playground/broom/internal/config"
)

// An authed command with no credential anywhere must fail with the not-logged-in
// message, never something mistakable for a content bug.
func TestAuthedCommandWithoutCredentialFailsClearly(t *testing.T) {
	env, _, _, _ := testEnv(t, "", nil)

	err := run(env, "logs", "meta", "some-slug")
	if err == nil {
		t.Fatal("an authed command without a token should fail")
	}
	if !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("error = %q, want not-logged-in message", err.Error())
	}
}

// BROOM_TOKEN satisfies the credential gate without a config file, proving the
// env override reaches the command wiring. (The stub then reports not-yet-built
// rather than not-logged-in.)
func TestEnvTokenSatisfiesCredentialGate(t *testing.T) {
	env, _, _, _ := testEnv(t, "", map[string]string{
		"BROOM_URL":   "https://env.example",
		"BROOM_TOKEN": "env-token",
	})

	err := run(env, "logs", "meta", "some-slug")
	if err == nil {
		t.Fatal("stubbed command should still return a not-implemented error")
	}
	if strings.Contains(err.Error(), "not logged in") {
		t.Errorf("env token should satisfy the credential gate, got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("error = %q, want not-implemented once authed", err.Error())
	}
}

// A token persisted by login is usable by a later command reading the file, with
// no environment override in play.
func TestPersistedTokenIsUsedFromFile(t *testing.T) {
	env, _, _, path := testEnv(t, "", nil)
	if err := config.Save(path, config.Config{URL: "https://file.example", Token: "file-token"}); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	err := run(env, "logs", "meta", "some-slug")
	if err == nil || strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("file token should satisfy the gate, got: %v", err)
	}
}
