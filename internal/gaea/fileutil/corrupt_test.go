package fileutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBackupCorruptContentPreserved 损坏隔离（2026-10-06 断电事故根修落地）：
// 备份件逐字节等于原始损坏内容、名字挂在原路径后缀 .corrupt-<ts>，原文件不动。
func TestBackupCorruptContentPreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	corrupt := []byte{0, 0, 0, 0, 0}
	if err := os.WriteFile(path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	backup, err := BackupCorrupt(path, corrupt)
	if err != nil {
		t.Fatalf("BackupCorrupt() error = %v", err)
	}
	if !strings.HasPrefix(filepath.Base(backup), "config.toml.corrupt-") {
		t.Errorf("backup name = %q, want config.toml.corrupt-<ts> prefix", filepath.Base(backup))
	}
	got, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(got) != string(corrupt) {
		t.Errorf("backup content = %v, want %v", got, corrupt)
	}
	// 原文件保持原样（等下次正常写盘覆盖，而非被搬走）
	orig, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(orig) != string(corrupt) {
		t.Errorf("original file changed: %v", orig)
	}
}

// TestBackupCorruptUniqueTimestamps 连续两次隔离产生两个备份（时间戳命名不互覆）。
func TestBackupCorruptUniqueTimestamps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recent-workspaces.json")
	first, err := BackupCorrupt(path, []byte("["))
	if err != nil {
		t.Fatal(err)
	}
	second, err := BackupCorrupt(path, []byte("["))
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Errorf("two backups share the same path %q", first)
	}
}

// TestBackupCorruptWriteFailure 备份失败路径：目标目录不存在时诚实报错。
func TestBackupCorruptWriteFailure(t *testing.T) {
	if _, err := BackupCorrupt(filepath.Join(t.TempDir(), "no-such-dir", "f"), []byte("x")); err == nil {
		t.Error("BackupCorrupt() on missing dir: want error, got nil")
	}
}
