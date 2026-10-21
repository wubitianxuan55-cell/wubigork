package config

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// Save persists cfg to the user config file (~/.config/gaea/config.toml).
// Writes are atomic (temp file + rename) and fsynced before rename, so a
// crash or power loss can never leave a truncated config behind, nor expose
// the "rename succeeded but data is still in the page cache" window that
// zeroed config.toml on the 2026-10-06 power-loss incident. Secrets are never
// stored here — they live in the environment, resolved by api_key_env at load
// time.
func Save(cfg *Config) error {
	path := userConfigPath()
	if path == "" {
		return fmt.Errorf("config: user config dir unavailable")
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("config: encode: %w", err)
	}
	return fileutil.AtomicWriteSync(path, buf.Bytes(), 0o644)
}
