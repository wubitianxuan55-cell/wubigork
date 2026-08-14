package app

import (
	"os"
	"path/filepath"
	"testing"

	wdb "github.com/gaea/gaea/internal/whisper/db"
)

// seedPinned 在 cwd 下创建固定清单 + 对应文件（供 hub 关联测试使用）。
func seedPinned(t *testing.T, cwd, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(cwd, ".gaea"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".gaea", "pinned.json"), []byte(`["`+rel+`"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(cwd, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("固定资料正文。"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHubPinnedCount(t *testing.T) {
	t.Chdir(t.TempDir())
	if got := hubPinnedCount(); got != 0 {
		t.Fatalf("空工作区固定数应为 0, got %d", got)
	}
	seedPinned(t, ".", "docs/说明.md")
	if got := hubPinnedCount(); got != 1 {
		t.Fatalf("固定数应为 1, got %d", got)
	}
}

func TestGaeaMemoryGraphPinnedNodes(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, rel := range []string{"docs/说明.md", "报价.xlsx"} {
		abs := filepath.Join(".", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("固定资料正文。"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(".gaea", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".gaea/pinned.json", []byte(`["docs/说明.md","报价.xlsx"]`), 0o644); err != nil {
		t.Fatal(err)
	}

	whisperRoot := t.TempDir()
	if _, err := wdb.GetDatabase(whisperRoot); err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	defer wdb.CloseDatabase(whisperRoot)

	a := &App{whisperState: &whisperState{whisperDataRoot: whisperRoot}}
	g := a.GaeaMemoryGraph()

	want := map[string]bool{"m:docs/说明.md": true, "m:报价.xlsx": true}
	for _, n := range g.Nodes {
		if want[n.ID] {
			if n.Type != "material" {
				t.Fatalf("固定资料节点类型应为 material: %+v", n)
			}
			delete(want, n.ID)
		}
	}
	if len(want) != 0 {
		t.Fatalf("固定资料节点缺失: %v", want)
	}
}
