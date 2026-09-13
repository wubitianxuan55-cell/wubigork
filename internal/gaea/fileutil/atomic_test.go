package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// Windows：目标文件被无 FILE_SHARE_DELETE 的句柄持有时，覆盖 rename 报
// Access is denied（os.ErrPermission）——AV/索引器瞬态持锁是 ci 假红根源。
func TestRenameWithRetry_TransientLockSucceeds(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Access-denied rename 语义仅 Windows")
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "target.json")
	os.WriteFile(dst, []byte("old"), 0o644)
	src := filepath.Join(dir, "src.json")
	os.WriteFile(src, []byte("new"), 0o644)

	// 持锁 5ms 后释放；RenameWithRetry 的重试序列（5/10/20ms）应覆盖到释放后。
	h, err := os.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(5 * time.Millisecond)
		h.Close()
	}()

	if err := RenameWithRetry(src, dst); err != nil {
		t.Fatalf("瞬态持锁应经重试成功: %v", err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "new" {
		t.Fatalf("目标内容应为新数据: %q", got)
	}
}

func TestRenameWithRetry_PersistentLockFailsFastClass(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Access-denied rename 语义仅 Windows")
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "target.json")
	os.WriteFile(dst, []byte("old"), 0o644)
	src := filepath.Join(dir, "src.json")
	os.WriteFile(src, []byte("new"), 0o644)

	// 全程持锁：重试穷尽后报错（错误可继续用 errors.Is 判别权限类）
	h, err := os.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	start := time.Now()
	err = RenameWithRetry(src, dst)
	if err == nil {
		t.Fatal("持续持锁应失败")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("应保留权限类错误: %v", err)
	}
	if time.Since(start) < 30*time.Millisecond {
		t.Fatalf("应经历完整退避序列: %v", time.Since(start))
	}
	// 原数据不被破坏
	got, _ := os.ReadFile(dst)
	if string(got) != "old" {
		t.Fatalf("失败后目标应原样: %q", got)
	}
}

func TestAtomicWrite_UsesRetryPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := AtomicWrite(path, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatalf("首次写入: %v", err)
	}
	if err := AtomicWrite(path, []byte(`{"a":2}`), 0o644); err != nil {
		t.Fatalf("覆盖写入: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != `{"a":2}` {
		t.Fatalf("覆盖结果不符: %q", got)
	}
}
