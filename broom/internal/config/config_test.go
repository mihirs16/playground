package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mihirs16/playground/broom/internal/config"
)

// isolate points config at a throwaway XDG dir and clears the env overrides so
// each test sees a clean slate.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("BROOM_URL", "")
	t.Setenv("BROOM_TOKEN", "")
	return dir
}

func TestResolveMissingFileYieldsDefaultURLAndNoToken(t *testing.T) {
	isolate(t)

	cfg, err := config.Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.URL != config.DefaultURL {
		t.Errorf("url = %q, want default %q", cfg.URL, config.DefaultURL)
	}
	if cfg.Token != "" {
		t.Errorf("token = %q, want empty", cfg.Token)
	}
}

func TestSaveThenResolveRoundTrips(t *testing.T) {
	isolate(t)

	want := config.Config{URL: "https://custodian.example.com", Token: "sekret"}
	if err := config.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := config.Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != want {
		t.Errorf("resolved %+v, want %+v", got, want)
	}
}

func TestSaveWritesFileAt0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file mode bits are not meaningful on windows")
	}
	isolate(t)

	if err := config.Save(config.Config{URL: "https://x", Token: "t"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	path, err := config.Path()
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	isolate(t)
	if err := config.Save(config.Config{URL: "https://file", Token: "file-token"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	t.Setenv("BROOM_URL", "https://env")
	t.Setenv("BROOM_TOKEN", "env-token")

	cfg, err := config.Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.URL != "https://env" {
		t.Errorf("url = %q, want env override", cfg.URL)
	}
	if cfg.Token != "env-token" {
		t.Errorf("token = %q, want env override", cfg.Token)
	}
}

func TestEnvOverridesAreIndependent(t *testing.T) {
	isolate(t)
	if err := config.Save(config.Config{URL: "https://file", Token: "file-token"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	t.Setenv("BROOM_TOKEN", "env-token")

	cfg, err := config.Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.URL != "https://file" {
		t.Errorf("url = %q, want file value when only token overridden", cfg.URL)
	}
	if cfg.Token != "env-token" {
		t.Errorf("token = %q, want env override", cfg.Token)
	}
}

func TestClearTokenKeepsURL(t *testing.T) {
	isolate(t)
	if err := config.Save(config.Config{URL: "https://keepme", Token: "drop"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := config.ClearToken(); err != nil {
		t.Fatalf("clear: %v", err)
	}

	cfg, err := config.Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.URL != "https://keepme" {
		t.Errorf("url = %q, want retained after logout", cfg.URL)
	}
	if cfg.Token != "" {
		t.Errorf("token = %q, want cleared", cfg.Token)
	}
}

func TestClearTokenOnMissingFileSucceeds(t *testing.T) {
	isolate(t)
	if err := config.ClearToken(); err != nil {
		t.Fatalf("clear on missing file: %v", err)
	}
}

func TestPathHonoursXDGConfigHome(t *testing.T) {
	dir := isolate(t)
	path, err := config.Path()
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	want := filepath.Join(dir, "broom", "config.json")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}
