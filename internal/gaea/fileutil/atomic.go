// Package fileutil provides file system utilities shared across the kernel.
package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite writes data to path atomically: it writes to a temporary
// sibling file and renames it over the target. Readers never see a partial
// write, and a crash mid-write can't leave a truncated file that would fail
// to reload. The parent directory is created if missing.
//
// This is the single shared implementation for the many places that need
// crash-safe persistence (session files, config, memory store, …).
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %s: %w", path, err)
	}
	return nil
}
