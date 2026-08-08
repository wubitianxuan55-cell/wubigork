package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGaeaPreview_Mermaid 验证 .mmd 图表文件可按 markdown 预览（渲染成图）。
func TestGaeaPreview_Mermaid(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "diagram-test.mmd")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	code := "flowchart LR\nA-->B"
	if err := os.WriteFile(rel, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	got := a.GaeaPreview(filepath.ToSlash(rel))
	if got.Kind != "markdown" {
		t.Fatalf("kind = %q, want markdown", got.Kind)
	}
	if !strings.HasPrefix(got.Body, "```mermaid\n") || !strings.Contains(got.Body, "flowchart LR") {
		t.Errorf("body = %q, want mermaid 围栏包裹", got.Body)
	}
}

// TestGaeaPreview_Missing 不存在的文件返回 error。
func TestGaeaPreview_Missing(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	got := a.GaeaPreview("nope.mmd")
	if got.Kind != "error" {
		t.Fatalf("kind = %q, want error", got.Kind)
	}
}
