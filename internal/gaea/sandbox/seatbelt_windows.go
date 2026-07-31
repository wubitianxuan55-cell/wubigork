//go:build windows

package sandbox

// Command wraps a shell command to enforce the sandbox spec. On Windows when
// WSL2 is available and spec.Mode is "enforce", the command runs inside a WSL2
// Linux distribution for stronger isolation. When WSL2 is unavailable, the
// command runs unconfined (the permission layer still gates the call).
func Command(spec Spec, sh Shell, command string) ([]string, bool) {
	if !spec.enforce() {
		return sh.argv(command), false
	}

	// Try WSL2 isolation
	if distro := DetectWSL2(); distro != nil {
		argv, ok := distro.WrapCommand(command)
		if ok {
			return argv, true
		}
	}

	// WSL2 unavailable — fall back to unconfined (boot/acp already warned
	// at startup; the false result signals "not sandboxed").
	return sh.argv(command), false
}

// Available reports whether an OS sandbox is available on this platform.
// On Windows, this checks for WSL2 with at least one available distribution.
func Available() bool {
	return AvailableWSL2()
}
