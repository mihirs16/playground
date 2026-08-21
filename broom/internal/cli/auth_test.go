package cli_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/mihirs16/playground/broom/internal/config"
)

func TestLoginVerifiesAndWritesConfig(t *testing.T) {
	ts := fakeCustodian(t, "good-token")
	env, out, _, path := testEnv(t, "good-token\n", map[string]string{"BROOM_URL": ts.URL})

	if err := run(env, "login"); err != nil {
		t.Fatalf("login: %v", err)
	}

	if !strings.Contains(out.String(), "Logged in to "+ts.URL) {
		t.Errorf("stdout = %q, want success line", out.String())
	}

	cfg, err := config.Resolve(path, func(string) string { return "" })
	if err != nil {
		t.Fatalf("resolve saved config: %v", err)
	}
	if cfg.Token != "good-token" {
		t.Errorf("saved token = %q, want good-token", cfg.Token)
	}
	if cfg.URL != ts.URL {
		t.Errorf("saved url = %q, want %q", cfg.URL, ts.URL)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat config: %v", err)
		}
		if perm := info.Mode().Perm(); perm != config.FileMode {
			t.Errorf("config perm = %o, want %o", perm, config.FileMode)
		}
	}
}

func TestLoginRejectedTokenIsNotWritten(t *testing.T) {
	ts := fakeCustodian(t, "good-token")
	env, _, _, path := testEnv(t, "wrong-token\n", map[string]string{"BROOM_URL": ts.URL})

	err := run(env, "login")
	if err == nil {
		t.Fatal("login with a rejected token should fail")
	}
	if !strings.Contains(err.Error(), "token rejected") {
		t.Errorf("error = %q, want token-rejected message", err.Error())
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("config file should not be written when the token is rejected")
	}
}

// A custodian-down failure must read as unreachable, never as a rejected
// request — the two are distinct failure modes (user story 29).
func TestLoginCustodianDownIsDistinctFromRejection(t *testing.T) {
	ts := fakeCustodian(t, "good-token")
	downURL := ts.URL
	ts.Close() // now nothing is listening: connection refused, a transport failure

	env, _, _, _ := testEnv(t, "good-token\n", map[string]string{"BROOM_URL": downURL})

	err := run(env, "login")
	if err == nil {
		t.Fatal("login against a down custodian should fail")
	}
	if !strings.Contains(err.Error(), "cannot reach custodian") {
		t.Errorf("error = %q, want unreachable message", err.Error())
	}
	if strings.Contains(err.Error(), "token rejected") {
		t.Errorf("unreachable custodian must not be reported as a rejection: %q", err.Error())
	}
}

func TestLogoutClearsTokenKeepsURL(t *testing.T) {
	env, out, _, path := testEnv(t, "", nil)
	if err := config.Save(path, config.Config{URL: "https://keep.example", Token: "secret"}); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	if err := run(env, "logout"); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if !strings.Contains(out.String(), "Logged out.") {
		t.Errorf("stdout = %q, want logout line", out.String())
	}

	cfg, err := config.Resolve(path, func(string) string { return "" })
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.Token != "" {
		t.Errorf("token = %q, want cleared", cfg.Token)
	}
	if cfg.URL != "https://keep.example" {
		t.Errorf("url = %q, want preserved", cfg.URL)
	}
}
