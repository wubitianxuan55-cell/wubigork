//go:build windows

package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// WSL2Distro holds the name of an available WSL2 Linux distribution.
type WSL2Distro struct {
	Name string
}

// DetectWSL2 checks whether WSL2 is available and returns the first usable
// distribution. Returns nil when WSL2 is not installed or no distro is found.
func DetectWSL2() *WSL2Distro {
	// Check wsl.exe on PATH
	if _, err := exec.LookPath("wsl.exe"); err != nil {
		if _, err2 := exec.LookPath("wsl"); err2 != nil {
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wsl.exe", "--list", "--quiet")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Fields(string(out))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" && !strings.Contains(name, "msys") &&
			!strings.Contains(name, "cygwin") {
			return &WSL2Distro{Name: name}
		}
	}
	return nil
}

// WrapCommand wraps a shell command to run inside the WSL2 distribution.
// Returns the wrapped argv and a boolean indicating whether wrapping occurred.
// When spec.Network is false, the wrapping still applies (network is inherited
// from WSL2 — full network confinement requires iptables rules inside the VM).
func (d *WSL2Distro) WrapCommand(shellCmd string) ([]string, bool) {
	if d == nil || d.Name == "" {
		return nil, false
	}
	// wsl -d <distro> -- bash -c "<command>"
	argv := []string{
		"wsl.exe", "-d", d.Name, "--",
	}
	// Use bash if available, otherwise sh
	argv = append(argv, "bash", "-c", shellCmd)
	return argv, true
}

// CommandViaWSL2 runs a command inside the WSL2 distribution. Returns stdout
// content and any error. Adds a 5-minute timeout.
func (d *WSL2Distro) CommandViaWSL2(ctx context.Context, shellCmd string) (string, error) {
	argv, ok := d.WrapCommand(shellCmd)
	if !ok {
		return "", fmt.Errorf("WSL2 not available")
	}

	execCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(execCtx, argv[0], argv[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("wsl: %s: %w", errMsg, err)
		}
		return "", fmt.Errorf("wsl: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// AvailableWSL2 returns true when WSL2 is available with at least one distro.
func AvailableWSL2() bool {
	return DetectWSL2() != nil
}
