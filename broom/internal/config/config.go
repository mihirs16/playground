// Package config resolves broom's credential — the custodian url and admin
// token — from a single XDG config file, with environment overrides. The file
// is the durable store login writes and logout clears; the environment is an
// ephemeral override for a one-off invocation against a different custodian.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// DefaultURL is the baked-in custodian used when neither the file nor the
// environment names one.
const DefaultURL = "https://custodian.mihirsingh.dev"

const (
	envURL   = "BROOM_URL"
	envToken = "BROOM_TOKEN"
)

// Config is broom's resolved credential for one invocation.
type Config struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// stored is the on-disk shape. It is intentionally identical to Config; the
// separate type keeps the file format from silently tracking in-memory fields
// added later for other purposes.
type stored struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// Resolve reads the config file and applies environment overrides on top. A
// missing file is not an error — it yields the baked-in default url and an
// empty token, which the caller reads as "not logged in". BROOM_URL and
// BROOM_TOKEN each override the file independently.
func Resolve() (Config, error) {
	cfg := Config{URL: DefaultURL}

	path, err := Path()
	if err != nil {
		return cfg, err
	}
	file, err := readFile(path)
	if err != nil {
		return cfg, err
	}
	if file.URL != "" {
		cfg.URL = file.URL
	}
	cfg.Token = file.Token

	if url := os.Getenv(envURL); url != "" {
		cfg.URL = url
	}
	if token := os.Getenv(envToken); token != "" {
		cfg.Token = token
	}
	return cfg, nil
}

// Save persists url and token to the config file at 0600, creating the parent
// directory if needed. It replaces the file wholesale — the credential is the
// only thing broom stores.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(stored{URL: cfg.URL, Token: cfg.Token}, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o600)
}

// ClearToken removes the stored token while leaving the url in place, so a
// later login lands on the same custodian without re-typing the url. A missing
// file is already logged-out, so it succeeds silently.
func ClearToken() error {
	path, err := Path()
	if err != nil {
		return err
	}
	file, err := readFile(path)
	if err != nil {
		return err
	}
	if file.URL == "" && file.Token == "" {
		return nil
	}
	return Save(Config{URL: file.URL})
}

// Path is the resolved location of the config file. It honours
// XDG_CONFIG_HOME, falling back to the platform's conventional config dir.
func Path() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "broom", "config.json"), nil
	}
	dir, err := configHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "broom", "config.json"), nil
}

// configHome is the conventional per-user config directory when
// XDG_CONFIG_HOME is unset. On non-Windows platforms os.UserConfigDir already
// resolves to ~/.config; on Windows it points at %AppData%, which is where a
// user expects a CLI's config to live.
func configHome() (string, error) {
	if runtime.GOOS != "windows" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config"), nil
	}
	return os.UserConfigDir()
}

func readFile(path string) (stored, error) {
	var file stored
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return file, nil
		}
		return file, err
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return file, err
	}
	return file, nil
}
