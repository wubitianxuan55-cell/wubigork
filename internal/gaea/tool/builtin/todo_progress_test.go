package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSaveProgressMarkdownWritesTodosNotProgress v4.27.4 回归：todo_write 的
// 计划持久化写 .gaea/todos.md，**绝不碰 .gaea/progress.md**。progress.md 是
// 宿主仓库（用 gaea 开发 gaea）的项目记忆文件，同名覆盖曾把 wubigrok 仓库
// 发布进度一天内冲掉四次。
func TestSaveProgressMarkdownWritesTodosNotProgress(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".gaea"), 0o755); err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(root, "nested", "deep")
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

	saveProgressMarkdown([]todoItem{
		{Content: "编制方案", Status: "completed"},
		{Content: "生成 Word", Status: "in_progress", ActiveForm: "正在生成 Word"},
	})

	data, err := os.ReadFile(filepath.Join(root, ".gaea", "todos.md"))
	if err != nil {
		t.Fatalf("todos.md 应被写入: %v", err)
	}
	if !strings.Contains(string(data), "编制方案") || !strings.Contains(string(data), "任务进度") {
		t.Fatalf("todos.md 内容错误: %s", data)
	}
	if _, err := os.Stat(filepath.Join(root, ".gaea", "progress.md")); !os.IsNotExist(err) {
		t.Fatalf(".gaea/progress.md 不得被 todo_write 创建/触碰 (stat err = %v)", err)
	}
}
