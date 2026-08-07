package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// E11：办公 agent 工具目录跟随工作空间（Workspace.Dir），路径逃逸一律拒绝。
func TestRegressionE11WorkspaceFollowsDirAndConfines(t *testing.T) {
	dir := t.TempDir()
	w := Workspace{Dir: dir} // WriteRoots 为空 → Dir 为唯一写根
	tools := map[string]tool.Tool{}
	for _, tl := range w.Tools() {
		tools[tl.Name()] = tl
	}
	wf, ok := tools["write_file"]
	if !ok {
		t.Fatal("write_file missing")
	}
	ctx := context.Background()
	out, err := wf.Execute(ctx, json.RawMessage(`{"path":"sub/a.txt","content":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "wrote") {
		t.Fatalf("out = %q", out)
	}
	b, err := os.ReadFile(filepath.Join(dir, "sub", "a.txt"))
	if err != nil || string(b) != "hello" {
		t.Fatalf("file=%q err=%v", b, err)
	}
	// 逃逸拒绝：不得写到处在工作区之外。
	if _, err := wf.Execute(ctx, json.RawMessage(`{"path":"../../evil.txt","content":"x"}`)); err == nil {
		t.Fatal("escape should fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "evil.txt")); !os.IsNotExist(err) {
		t.Fatal("escape file created")
	}
}
