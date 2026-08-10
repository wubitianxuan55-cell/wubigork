package app

import (
	"os"
	"strings"
	"testing"
)

func TestGaeaWorkspaceSearch(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/成本测算.md", []byte("本项目成本测算总金额为 100 万元。"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/说明.md", []byte("无关内容。"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("node_modules", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("node_modules/藏.md", []byte("成本机密。"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	hits := a.GaeaWorkspaceSearch("成本", 10)
	if len(hits) == 0 {
		t.Fatal("应命中成本相关文件")
	}
	if hits[0].Path != "docs/成本测算.md" {
		t.Fatalf("应命中 docs/成本测算.md: %+v", hits)
	}
	if !strings.Contains(hits[0].Snippet, "成本") {
		t.Fatalf("片段应含关键词: %q", hits[0].Snippet)
	}
	for _, h := range hits {
		if strings.HasPrefix(h.Path, "node_modules") {
			t.Fatalf("噪音目录泄漏: %s", h.Path)
		}
	}

	if hits := a.GaeaWorkspaceSearch("不存在xyz", 10); len(hits) != 0 {
		t.Fatalf("无命中应返回空: %+v", hits)
	}
}
