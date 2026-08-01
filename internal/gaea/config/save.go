package config

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// Save persists cfg to the user config file (~/.config/gaea/config.toml).
// Writes are atomic (temp file + rename) so a crash mid-write can never leave
// a truncated config behind. Secrets are never stored here — they live in the
// environment, resolved by api_key_env at load time.
func Save(cfg *Config) error {
	path := userConfigPath()
	if path == "" {
		return fmt.Errorf("config: user config dir unavailable")
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("config: encode: %w", err)
	}
	return fileutil.AtomicWrite(path, buf.Bytes(), 0o644)
}
