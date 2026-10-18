package app

import (
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// TestAttachKillOnCloseJob_CloseKillsTree 真进程实弹：作业句柄一关，作业内
// 进程终止——这是「gaea 退出（含崩溃）引擎树必死」的内核级保证。
func TestAttachKillOnCloseJob_CloseKillsTree(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only")
	}
	cmd := exec.Command("cmd", "/c", "ping", "-n", "60", "127.0.0.1")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	job, err := attachKillOnCloseJob(cmd)
	if err != nil {
		t.Fatalf("attachKillOnCloseJob: %v", err)
	}
	if job == 0 {
		t.Fatal("job 句柄为 0")
	}

	// 纳管后进程应活着（QUERY 权限打开成功=进程存在）
	if ph, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cmd.Process.Pid)); err != nil {
		t.Fatalf("纳管后进程应存活: %v", err)
	} else {
		_ = windows.CloseHandle(ph)
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

	// 关句柄 → kill-on-close → 进程终止
	if err := windows.CloseHandle(job); err != nil {
		t.Fatalf("CloseHandle(job): %v", err)
	}
	select {
	case <-waitDone:
		// 进程已退出（退出码非 0 正常——被内核终止）
	case <-time.After(10 * time.Second):
		t.Fatal("关闭作业句柄 10s 后进程仍未终止")
	}
}

// TestLocalEngineProc_CloseJobIdempotent closeJob 幂等：零值/重复关闭不 panic。
func TestLocalEngineProc_CloseJobIdempotent(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only")
	}
	p := &localEngineProc{}
	p.closeJob() // 零值 job=0：no-op
	p.closeJob() // 重复：仍 no-op
}
