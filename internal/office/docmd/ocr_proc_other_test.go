//go:build !windows

package docmd

import (
	"os"
	"syscall"
)

// processAlive 报告 pid 进程是否仍在运行（非 Windows 实现：signal 0 探针）。
func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}
