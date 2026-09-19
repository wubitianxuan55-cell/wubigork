package app

// 2026-09-19 审计读侧收口回归：①GaeaReadFileB64 契约强制化（仅系统对话框
// 选择过的文件可读）②GaeaReadFile 拒绝穿越/绝对路径③快照 sceneID 白名单。

import (
	"os"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/snapshot"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileB64RequiresPick(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	a := &App{core: &core{cfg: &config.Config{}}}

	dir := t.TempDir()
	p := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 未登记：拒绝（fail-closed——此前任意绝对路径 ≤64MB 可读）。
	if _, err := a.GaeaReadFileB64(p); err == nil || !strings.Contains(err.Error(), "文件对话框") {
		t.Fatalf("未登记路径应拒绝, got %v", err)
	}

	// 经 GaeaPickFiles 登记后可读——直接调登记助手模拟对话框结果。
	registerPickedFile(p)
	b64, err := a.GaeaReadFileB64(p)
	if err != nil {
		t.Fatalf("登记后应可读: %v", err)
	}
	if !strings.Contains(b64, "aGVsbG8=") { // base64("hello")
		t.Fatalf("内容不符: %s", b64)
	}
}

func TestGaeaReadFileTraversalGuard(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	a := &App{core: &core{cfg: &config.Config{}}}

	for _, rel := range []string{`..\..\secret.txt`, "../../secret.txt", `C:\Windows\win.ini`} {
		got := a.GaeaReadFile(rel)
		if !strings.Contains(got.Markdown, "非法") && !strings.Contains(got.Markdown, "不在可读范围") {
			t.Fatalf("穿越路径 %q 应被拒绝, got %+v", rel, got)
		}
	}
	// 工作区内正常文件不受影响。
	if err := os.WriteFile("note.md", []byte("# ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := a.GaeaReadFile("note.md"); !strings.Contains(got.Markdown, "# ok") {
		t.Fatalf("工作区内文件应可读, got %+v", got)
	}
}

func TestSnapshotSceneIDGuard(t *testing.T) {
	s := snapshot.NewStore(t.TempDir())
	if _, err := s.Capture(`..\evil`, "正文", "标签", "manual"); err == nil || !strings.Contains(err.Error(), "非法场景") {
		t.Fatalf("穿越 sceneID 应拒绝, got %v", err)
	}
	if _, err := s.Capture("001-evil", "正文", "标签", "manual"); err != nil {
		t.Fatalf("合法 sceneID 应通过: %v", err)
	}
}
