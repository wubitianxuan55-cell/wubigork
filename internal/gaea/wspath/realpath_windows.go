//go:build windows

package wspath

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// canonicalExisting opens the path (following reparse points) and asks Windows
// for the final DOS path. Unlike filepath.EvalSymlinks on the current Windows
// runtime, this resolves directory junctions as well as symbolic links.
func canonicalExisting(path string) (string, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	handle, err := windows.CreateFile(
		name,
		windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return "", err
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	buf := make([]uint16, windows.MAX_LONG_PATH)
	n, err := windows.GetFinalPathNameByHandle(handle, &buf[0], uint32(len(buf)), 0)
	if err != nil {
		return "", err
	}
	if n >= uint32(len(buf)) {
		buf = make([]uint16, n+1)
		n, err = windows.GetFinalPathNameByHandle(handle, &buf[0], uint32(len(buf)), 0)
		if err != nil {
			return "", err
		}
	}
	final := windows.UTF16ToString(buf[:n])
	switch {
	case strings.HasPrefix(final, `\\?\UNC\`):
		final = `\\` + strings.TrimPrefix(final, `\\?\UNC\`)
	case strings.HasPrefix(final, `\\?\`):
		final = strings.TrimPrefix(final, `\\?\`)
	}
	return filepath.Clean(final), nil
}
