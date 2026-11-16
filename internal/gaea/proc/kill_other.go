//go:build !windows

package proc

import (
	"os"
	"os/exec"
	"syscall"
)

// KillTree kills cmd's whole process group. StartTracked (and SetProcessGroupKill
// for children started outside it) put the child in its own group via Setpgid, so
// the negative-pid signal reaches every descendant, including a launcher whose
// sub-daemon survives the parent, where a plain Process.Kill would only hit the
// direct child and orphan the grandchild.
func KillTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	KillTreePID(cmd.Process.Pid)
	_ = cmd.Process.Kill()
}

// KillTreePID terminates a persistent process group by PID.
func KillTreePID(pid int) {
	if pid <= 0 {
		return
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		if p, findErr := os.FindProcess(pid); findErr == nil {
			_ = p.Kill()
		}
	}
}

// ReleaseTracked is a no-op off Windows: independent process groups already
// survive their parent. KillTreePID remains available for re-adopted sessions.
func ReleaseTracked(*exec.Cmd, uintptr) error {
	return nil
}

// SetProcessGroupKill makes cmd start in its own process group so KillTree can
// reap its whole tree. Use it for children started outside StartTracked (e.g. a
// one-shot CombinedOutput). It is a no-op on Windows, where the Job Object that
// TrackTree/StartTracked assigns handles the tree instead.
func SetProcessGroupKill(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// StartTracked starts cmd in its own process group so KillTracked / KillTree can
// reap its whole tree. Off Windows the process group is the equivalent of the
// Windows Job Object; it returns a 0 handle, and KillTracked falls back to
// KillTree.
func StartTracked(cmd *exec.Cmd) (uintptr, error) {
	SetProcessGroupKill(cmd)
	return 0, cmd.Start()
}

// KillTracked terminates cmd's process tree; the handle is unused off Windows.
func KillTracked(cmd *exec.Cmd, _ uintptr) { KillTree(cmd) }
