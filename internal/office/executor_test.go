// Package office — executor_test.go
package office

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExecWriteFileAtomicReadableBack 覆盖 write_file 的原子写路径（fileutil.AtomicWrite：
// 临时文件 + rename）。崩溃中断留原文件的语义难以直接测，这里测"写成功后可读回一致"，
// 同时验证父目录自动创建与返回 Path 不变。
func TestExecWriteFileAtomicReadableBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reports", "造价汇总.txt")
	content := "项目A: 1000000 元\n项目B: 2000000 元\n"

	res := execWriteFile(path, content)
	if !res.Success {
		t.Fatalf("execWriteFile: %s", res.Error)
	}
	if res.Path != path {
		t.Fatalf("Path = %q, want %q", res.Path, path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != content {
		t.Fatalf("content mismatch:\n got  %q\n want %q", got, content)
	}
}

// TestExecWriteFileOverwrite 原子写覆盖已有文件：rename 直接替换旧内容，
// 且目标目录只留目标文件（无 *.tmp 残留）。
func TestExecWriteFileOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	if res := execWriteFile(path, "old"); !res.Success {
		t.Fatalf("first write: %s", res.Error)
	}
	if res := execWriteFile(path, "new content"); !res.Success {
		t.Fatalf("overwrite: %s", res.Error)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "new content" {
		t.Fatalf("content = %q, want %q", got, "new content")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "out.txt" {
		t.Fatalf("dir should contain only out.txt, got %d entries", len(entries))
	}
}
