//go:build windows

package app

import (
	"testing"
)

// 回归钉（2026-11-08 用户实弹）：编程板块启动后每 3.5s 状态轮询都会跑一次
// tasklist 探针；探针若不带 CREATE_NO_WINDOW，GUI 进程每次 spawn 都会分配
// 可见控制台 = cmd 窗口不停闪。此钉保证探针永远走 proc.HideWindow。
func TestProgrammingWebPIDProbeHidesWindow(t *testing.T) {
	const createNoWindow = 0x08000000
	probe := programmingWebPIDProbe(123)
	if probe.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil; probe skipped proc.HideWindow")
	}
	if probe.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("CREATE_NO_WINDOW not set on tasklist probe; CreationFlags=%#x", probe.SysProcAttr.CreationFlags)
	}
}
