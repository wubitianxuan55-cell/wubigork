package app

import (
	"os"
	"testing"
)

func TestGaeaPinMaterial(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/说明.md", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	if got := a.GaeaPinnedMaterials(); len(got) != 0 {
		t.Fatalf("初始固定应为空: %+v", got)
	}
	pinned := a.GaeaPinMaterial("docs/说明.md")
	if len(pinned) != 1 || pinned[0].Path != "docs/说明.md" {
		t.Fatalf("固定失败: %+v", pinned)
	}
	// 重复固定幂等
	if got := a.GaeaPinMaterial("docs/说明.md"); len(got) != 1 {
		t.Fatalf("重复固定未去重: %+v", got)
	}
	// 缺失文件不返回（但仍在清单中，pinnedView 跳过）
	_ = os.Remove("docs/说明.md")
	if got := a.GaeaPinnedMaterials(); len(got) != 0 {
		t.Fatalf("缺失文件应跳过: %+v", got)
	}
	// 取消固定
	_ = os.WriteFile("docs/说明.md", []byte("x"), 0o644)
	after := a.GaeaUnpinMaterial("docs/说明.md")
	if len(after) != 0 {
		t.Fatalf("取消固定失败: %+v", after)
	}
}

func TestGaeaPinMaterialRejectsEscape(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	if got := a.GaeaPinMaterial("../outside.md"); len(got) != 0 {
		t.Fatalf("越界路径不应固定: %+v", got)
	}
}
