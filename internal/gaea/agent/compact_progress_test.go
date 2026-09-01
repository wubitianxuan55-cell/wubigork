package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadProgressFilePrefersTodosAndFallsBack v4.27.4：todo 持久化改名
// todos.md 后，读取端优先新名、回退旧名（存量工作区兼容）。
func TestReadProgressFilePrefersTodosAndFallsBack(t *testing.T) {
	root := t.TempDir()
	gaeaDir := filepath.Join(root, ".gaea")
	if err := os.MkdirAll(gaeaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, "sub")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}

	// 两份并存：优先 todos.md
	if err := os.WriteFile(filepath.Join(gaeaDir, "todos.md"), []byte("新"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gaeaDir, "progress.md"), []byte("旧"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readProgressFile(); got != "新" {
		t.Fatalf("并存时应优先 todos.md, got %q", got)
	}

	// 只有旧名：回退兼容（存量工作区）
	if err := os.Remove(filepath.Join(gaeaDir, "todos.md")); err != nil {
		t.Fatal(err)
	}
	if got := readProgressFile(); got != "旧" {
		t.Fatalf("仅旧名时应回退 progress.md, got %q", got)
	}

	// 注：不测「均缺失→空串」——readProgressFile 会向上遍历祖先目录，真实
	// 机器上（用户主目录/仓库根）常存在其他 .gaea/progress.md，属于设计内行为。
}
