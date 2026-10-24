package app

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPickedFileKeyMatchesWindowsCaseInsensitively(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Picked.PDF")
	if runtime.GOOS == "windows" {
		if pickedFileKey(path) != pickedFileKey(strings.ToLower(path)) {
			t.Fatal("Windows picked-file key should match case variants")
		}
		return
	}
	if pickedFileKey(path) == pickedFileKey(strings.ToLower(path)) {
		t.Fatal("case-sensitive filesystems must not collapse case variants")
	}
}
