//go:build windows

package docmd

import "golang.org/x/sys/windows"

// processAlive 报告 pid 进程是否仍在运行（Windows 实现：OpenProcess +
// GetExitCodeProcess，退出码 259 = STILL_ACTIVE 表示存活）。
func processAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}
