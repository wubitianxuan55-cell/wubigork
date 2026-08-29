package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// stubTool 按名字构造最小 tool.Tool（仅用于空间标签测试）。
type stubTool struct{ name string }

func (s stubTool) Name() string { return s.name }
func (s stubTool) Description() string {
	return ""
}
func (s stubTool) Schema() json.RawMessage                                      { return json.RawMessage(`{"type":"object"}`) }
func (s stubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return "", nil }
func (s stubTool) ReadOnly() bool                                               { return true }

// taggedStub 额外实现 SpaceTaggedTool（自声明空间标签）。
type taggedStub struct {
	stubTool
	tag string
}

func (s taggedStub) SpaceTag() string { return s.tag }

// TestToolSpaceClassification 核对分类表：以实际注册工具名为准——
// work=办公/编辑/检索系，play=生图域，shared=通用编排/记忆（缺省 shared）。
func TestToolSpaceClassification(t *testing.T) {
	work := []string{
		"read_file", "write_file", "edit_file", "edit_lines", "multi_edit", "move_file",
		"ls", "grep", "bash", "bash_output", "kill_shell", "wait",
		"web_fetch", "web_search", "format_convert", "chart_gen", "diagram_gen", "diagram",
		"screen_capture", "vision", "ocr",
		"cost_search", "cost_save", "cost_indicators",
		"knowledge_add", "knowledge_search", "semantic_search",
		"routine_llm", "translate_text", "fact_add", "fact_list", "fact_clear",
	}
	for _, name := range work {
		if got := ToolSpace(name); got != "work" {
			t.Errorf("ToolSpace(%q) = %q, want work", name, got)
		}
	}
	play := []string{"image_gen"}
	for _, name := range play {
		if got := ToolSpace(name); got != "play" {
			t.Errorf("ToolSpace(%q) = %q, want play", name, got)
		}
	}
	shared := []string{"ask", "complete_step", "todo_write", "memory_search", "remember", "task", "run_skill"}
	for _, name := range shared {
		if got := ToolSpace(name); got != "shared" {
			t.Errorf("ToolSpace(%q) = %q, want shared", name, got)
		}
	}
	// 缺省 shared：未知名字与 MCP 动态名。
	for _, name := range []string{"definitely_not_a_tool", "mcp__github__create_issue"} {
		if got := ToolSpace(name); got != "shared" {
			t.Errorf("ToolSpace(%q) = %q, want shared（缺省）", name, got)
		}
	}
}

// TestAllowsSpaceMatrix 装配期过滤矩阵：work 不含 image_gen、play 不含 edit 系、
// shared 两空间都有；space 为空（mode=off 回退形态）全注册。
func TestAllowsSpaceMatrix(t *testing.T) {
	cases := []struct {
		name      string
		space     string
		tool      string
		wantAllow bool
	}{
		{"work 保留 edit_file", "work", "edit_file", true},
		{"work 排除 image_gen", "work", "image_gen", false},
		{"work 保留 ask", "work", "ask", true},
		{"play 排除 edit_file", "play", "edit_file", false},
		{"play 排除 bash", "play", "bash", false},
		{"play 排除 write_file", "play", "write_file", false},
		{"play 保留 image_gen", "play", "image_gen", true},
		{"play 保留 memory_search", "play", "memory_search", true},
		{"mode=off 全注册（work 工具）", "", "edit_file", true},
		{"mode=off 全注册（play 工具）", "", "image_gen", true},
		{"异常空间值 fail-open", "off", "image_gen", true},
	}
	for _, tc := range cases {
		if got := AllowsSpace(stubTool{name: tc.tool}, tc.space); got != tc.wantAllow {
			t.Errorf("%s: AllowsSpace(%q, %q) = %v, want %v", tc.name, tc.tool, tc.space, got, tc.wantAllow)
		}
	}
}

// TestSpaceTaggedToolOverridesTable SpaceTaggedTool 自声明优先于名字表
// （仿 PersistWriteTool 模式）。
func TestSpaceTaggedToolOverridesTable(t *testing.T) {
	// 名字 edit_file 在表中为 work；自声明 shared 后 play 也可注册。
	tt := taggedStub{stubTool: stubTool{name: "edit_file"}, tag: "shared"}
	if got := SpaceTagOf(tt); got != "shared" {
		t.Fatalf("SpaceTagOf = %q, want shared（自声明优先）", got)
	}
	if !AllowsSpace(tt, "play") {
		t.Fatal("自声明 shared 的 edit_file 应允许注册进 play")
	}
	// 未实现接口 → 回退名字表。
	if got := SpaceTagOf(stubTool{name: "edit_file"}); got != "work" {
		t.Fatalf("SpaceTagOf = %q, want work（名字表）", got)
	}
	// 自声明空串 → 回退名字表。
	empty := taggedStub{stubTool: stubTool{name: "edit_file"}, tag: " "}
	if got := SpaceTagOf(empty); got != "work" {
		t.Fatalf("SpaceTagOf = %q, want work（空声明回退名字表）", got)
	}
}

var _ tool.Tool = stubTool{}
