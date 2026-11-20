package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// 普通递归删除必须直接执行，不能生成 .trash 副本或把目标“移动”过去。
func TestBashDeletesChildDirectoryDirectly(t *testing.T) {
	root := t.TempDir()
	victim := filepath.Join(root, "build")
	if err := os.MkdirAll(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(victim, "artifact.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := bash{workDir: root}
	if out, err := b.Execute(context.Background(), []byte(`{"command":"rm -rf build"}`)); err != nil {
		t.Fatalf("direct rm failed: %v (out=%q)", err, out)
	}
	if _, err := os.Stat(victim); !os.IsNotExist(err) {
		t.Fatalf("target still exists after rm: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".trash")); !os.IsNotExist(err) {
		t.Fatalf("rm created a .trash fallback: %v", err)
	}
}
