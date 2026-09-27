//go:build !windows

package builtin

import (
	"os/exec"
)

// hideBashWindow 在非 Windows 平台为空操作
func hideBashWindow(cmd *exec.Cmd) {}

// assignJobObjectCleanup 非 Windows 平台不支持 Job Object，恒返回 ok=false，
// 调用方应 fallback 到 killProcessTree。签名与 windows 变体逐字一致
//（不得引用 syscall.Handle 等 Windows-only 类型，否则 Linux 交叉编译失败）。
func assignJobObjectCleanup(cmd *exec.Cmd) (cleanup func(), ok bool) {
	return nil, false
}
