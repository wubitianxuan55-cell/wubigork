//go:build !windows

package wspath

import "path/filepath"

func canonicalExisting(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
