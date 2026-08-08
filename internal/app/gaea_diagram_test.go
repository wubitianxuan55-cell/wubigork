package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiagramToolMeta 校验画图工具元信息与 Schema。
func TestDiagramToolMeta(t *testing.T) {
	tool := diagramTool{}
	if tool.Name() != "diagram" || strings.TrimSpace(tool.Description()) == "" {
		t.Fatalf("工具元信息异常: %s", tool.Name())
	}
	if !json.Valid(tool.Schema()) || !json.Valid(tool.CompactSchema()) {
		t.Fatal("Schema 非法")
	}
	if tool.ReadOnly() {
		t.Error("diagram 不应为只读工具")
	}
}

// TestExtractDiagramMermaid 兼容围栏/裸代码提取。
func TestExtractDiagramMermaid(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"```mermaid\nflowchart LR\nA-->B\n```", "flowchart LR\nA-->B"},
		{"```\nflowchart TD\nA-->B\n```", "flowchart TD\nA-->B"},
		{"flowchart LR\nA-->B", "flowchart LR\nA-->B"},
		{"请参考：\n```mermaid\npie title T\n\"A\": 1\n```\n以上", "pie title T\n\"A\": 1"},
	}
	for _, c := range cases {
		if got := extractDiagramMermaid(c.raw); got != c.want {
			t.Errorf("extractDiagramMermaid(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

// TestValidMermaidStart 首行关键字校验。
func TestValidMermaidStart(t *testing.T) {
	ok := []string{"flowchart LR\nA-->B", "sequenceDiagram\nA->>B", "%% comment\nmindmap\n root", "gantt\ntitle X"}
	for _, s := range ok {
		if !validMermaidStart(s) {
			t.Errorf("validMermaidStart(%q) = false, want true", s)
		}
	}
	bad := []string{"随便一句话", "graphql query", ""}
	for _, s := range bad {
		if validMermaidStart(s) {
			t.Errorf("validMermaidStart(%q) = true, want false", s)
		}
	}
}

// TestSaveDiagramMermaid 落盘 .mmd 文件。
func TestSaveDiagramMermaid(t *testing.T) {
	t.Chdir(t.TempDir())
	rel, err := saveDiagramMermaid(".", "flowchart LR\nA-->B")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel, ".gaea/uploads/diagram-") || !strings.HasSuffix(rel, ".mmd") {
		t.Fatalf("路径 = %q", rel)
	}
	data, err := os.ReadFile(filepath.FromSlash(rel))
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if !strings.Contains(string(data), "flowchart") {
		t.Error("文件内容不正确")
	}
}
