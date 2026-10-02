//go:build windows

package backup

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// lockExclusive 以 dwShareMode=0 独占打开文件，返回释放函数。独占句柄让
// os.Rename / os.Open 在该文件上失败（ERROR_SHARING_VIOLATION），是 Windows
// 上可稳定复现的 IO 失败注入点（审计 GA4-11 回滚部分：部分失败不得报成功）。
func lockExclusive(t *testing.T, path string) func() {
	t.Helper()
	if err := os.WriteFile(path, []byte("旧数据"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("独占打开 %s 失败: %v", path, err)
	}
	return func() { _ = syscall.CloseHandle(h) }
}

// TestRollbackBeforePartialFailureIsNotSuccess 审计 GA4-11：回滚中一个条目
// 失败（被独占占用 → rename 与复制回退都失败）、另一个成功后，旧实现按
// moved>0 返回 (true, nil) 谎报成功；修复后必须 (false, error) 且点名失败项，
// 同时保留 .restore-before 供重试。
func TestRollbackBeforePartialFailureIsNotSuccess(t *testing.T) {
	root := t.TempDir()
	before := filepath.Join(root, ".restore-before")
	if err := os.MkdirAll(before, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(before, "good.txt"), []byte("旧数据"), 0o644); err != nil {
		t.Fatal(err)
	}
	unlock := lockExclusive(t, filepath.Join(before, "locked.txt"))
	defer unlock()

	ok, err := RollbackBefore(root)
	if err == nil {
		t.Fatalf("存在回滚失败项时必须返回 error（旧实现 moved>0 即报 (true,nil)）: ok=%v", ok)
	}
	if ok {
		t.Error("部分失败不得报告回滚成功")
	}
	if !strings.Contains(err.Error(), "locked.txt") {
		t.Errorf("聚合错误须点名失败条目，实际: %v", err)
	}
	if data, rerr := os.ReadFile(filepath.Join(root, "good.txt")); rerr != nil || string(data) != "旧数据" {
		t.Errorf("可回滚条目应已回滚: %q err=%v", data, rerr)
	}
	if _, serr := os.Stat(before); serr != nil {
		t.Errorf("失败时应保留 .restore-before 供重试: %v", serr)
	}
}
