package wspath

import (
	"path/filepath"
)

// RealPath resolves path to an absolute, cleaned, symlink-free form. Because a
// write target need not exist yet, it resolves the deepest existing ancestor
// with EvalSymlinks and re-appends the not-yet-existing tail. On Windows this
// also resolves directory junctions.
//
// The result is intentionally lexical for a completely non-existent path: when
// no ancestor beyond a volume root exists, the cleaned absolute path is
// returned. Callers must still apply Within against canonical roots.
func RealPath(path string) (string, error) {
	abs, err := filepath.Abs(filepath.FromSlash(path))
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	tail := ""
	cur := abs
	for {
		if real, err := canonicalExisting(cur); err == nil {
			return filepath.Join(real, tail), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return abs, nil
		}
		tail = filepath.Join(filepath.Base(cur), tail)
		cur = parent
	}
}
