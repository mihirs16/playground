// Package config resolves broom's single source of connection settings: the URL
// of the custodian it talks to and the bearer token it authenticates with.
//
// Resolution order, highest precedence first: the BROOM_URL / BROOM_TOKEN
// environment variables, then the on-disk config file, then a baked-in default
// URL (the token has no default — an absent token means "not logged in"). The
// file is the only thing broom writes; it lives in the user's XDG config
// directory and is always written 0600 because it holds a secret.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultURL is the custodian broom talks to when nothing else specifies one, so
// a fresh install needs only a token.
const DefaultURL = "https://custodian.mihirsingh.dev"

// FileMode is the permission broom writes the config file with. It holds a
// bearer token, so it is owner-read/write only.
const FileMode fs.FileMode = 0o600

// Config is the fully-resolved connection settings a command runs with.
type Config struct {
	URL   string
	Token string
}

// file is the on-disk shape. URL and token live together in one file (ticket 17
// added the URL beside the token ticket 10 settled).
type file struct {
	URL   string `json:"url,omitempty"`
	Token string `json:"token,omitempty"`
}

// DefaultPath returns the config file's location under the user's XDG config
// directory (honouring XDG_CONFIG_HOME on unix, %AppData% on Windows).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "broom", "config.json"), nil
}

// Resolve produces the effective config from the file at path overlaid with the
// environment. A missing file is not an error — it resolves to the default URL
// and an empty token. getenv is injected so the precedence is testable without
// touching the process environment.
func Resolve(path string, getenv func(string) string) (Config, error) {
	stored, err := load(path)
	if err != nil {
		return Config{}, err
	}

	resolved := Config{URL: stored.URL, Token: stored.Token}
	if resolved.URL == "" {
		resolved.URL = DefaultURL
	}
	if v := getenv("BROOM_URL"); v != "" {
		resolved.URL = v
	}
	if v := getenv("BROOM_TOKEN"); v != "" {
		resolved.Token = v
	}
	return resolved, nil
}

// load reads the config file, treating a missing file as an empty config.
func load(path string) (file, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return file{}, nil
	}
	if err != nil {
		return file{}, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return file{}, err
	}
	return f, nil
}

// Save writes url + token to the config file at 0600, creating the parent
// directory if needed. It always writes both fields together.
func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(file{URL: c.URL, Token: c.Token}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, FileMode)
}

// ClearToken removes the stored token while leaving the URL in place, so a
// logged-out machine keeps pointing at the same custodian. A missing file is a
// no-op — there is nothing to clear.
func ClearToken(path string) error {
	stored, err := load(path)
	if err != nil {
		return err
	}
	if stored.Token == "" {
		return nil
	}
	stored.Token = ""
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, FileMode)
}
