package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mihirs16/playground/broom/internal/config"
)

// noEnv is the getenv for tests that assert no environment override is applied.
func noEnv(string) string { return "" }

func TestResolveMissingFileUsesDefaultURLAndNoToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	cfg, err := config.Resolve(path, noEnv)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.URL != config.DefaultURL {
		t.Errorf("URL = %q, want default %q", cfg.URL, config.DefaultURL)
	}
	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
}

func TestResolveReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{URL: "https://file.example", Token: "file-token"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	cfg, err := config.Resolve(path, noEnv)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.URL != "https://file.example" || cfg.Token != "file-token" {
		t.Errorf("resolved %+v, want file values", cfg)
	}
}

func TestResolveEnvOverridesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{URL: "https://file.example", Token: "file-token"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	env := map[string]string{
		"BROOM_URL":   "https://env.example",
		"BROOM_TOKEN": "env-token",
	}

	cfg, err := config.Resolve(path, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.URL != "https://env.example" {
		t.Errorf("URL = %q, want env override", cfg.URL)
	}
	if cfg.Token != "env-token" {
		t.Errorf("Token = %q, want env override", cfg.Token)
	}
}

func TestResolveEnvOverridesDefaultURLWithoutFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	env := map[string]string{"BROOM_TOKEN": "env-token"}

	cfg, err := config.Resolve(path, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.URL != config.DefaultURL {
		t.Errorf("URL = %q, want default", cfg.URL)
	}
	if cfg.Token != "env-token" {
		t.Errorf("Token = %q, want env token", cfg.Token)
	}
}

func TestSaveWritesOwnerOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions are not represented on Windows")
	}
	path := filepath.Join(t.TempDir(), "nested", "config.json")

	if err := config.Save(path, config.Config{URL: "https://x", Token: "t"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != config.FileMode {
		t.Errorf("perm = %o, want %o", perm, config.FileMode)
	}
}

func TestClearTokenKeepsURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{URL: "https://keep.example", Token: "secret"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := config.ClearToken(path); err != nil {
		t.Fatalf("clear: %v", err)
	}

	cfg, err := config.Resolve(path, noEnv)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.Token != "" {
		t.Errorf("Token = %q, want cleared", cfg.Token)
	}
	if cfg.URL != "https://keep.example" {
		t.Errorf("URL = %q, want preserved", cfg.URL)
	}
}

func TestClearTokenMissingFileIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.ClearToken(path); err != nil {
		t.Errorf("clear on missing file: %v", err)
	}
}
