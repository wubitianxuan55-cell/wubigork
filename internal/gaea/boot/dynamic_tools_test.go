package boot

// A2 动态工具目录装配测试（dsh-ops 蒸馏 2026-11-10）。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// stubDyn 假工具（big=撑 auto 阈值的大 schema MCP 工具）。
type stubDyn struct {
	name string
	big  bool
}

func (s stubDyn) Name() string { return s.name }
func (s stubDyn) Description() string {
	if s.big {
		return strings.Repeat("d", 40*1024)
	}
	return "stub " + s.name
}
func (s stubDyn) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (s stubDyn) ReadOnly() bool          { return true }
func (s stubDyn) Execute(context.Context, json.RawMessage) (string, error) {
	return "ok", nil
}

func TestResidentToolsForModes(t *testing.T) {
	reg := tool.NewRegistry()
	// 标准面小型 + 两个大 MCP 工具（超 32KB 阈值）。
	for _, n := range []string{"read_file", "write_file", "task"} {
		reg.Add(stubDyn{name: n})
	}
	reg.Add(stubDyn{name: "mcp__rea__scan", big: true})
	reg.Add(stubDyn{name: "mcp__rea__probe", big: true})

	// off：nil。
	if r := residentToolsFor("off", reg); r != nil {
		t.Fatalf("off 应 nil: %v", r)
	}
	// auto：MCP 重载 → 非 mcp 全常驻。
	r := residentToolsFor("auto", reg)
	got := map[string]bool{}
	for _, n := range r {
		got[n] = true
	}
	if !got["read_file"] || !got["task"] || got["mcp__rea__scan"] || got["mcp__rea__probe"] {
		t.Fatalf("auto 应非 MCP 全常驻: %v", r)
	}
	// on：核心+build-internal 常驻（本注册表只命中核心 read_file/write_file；
	// task 经 RegistryMap build-internal 判定常驻；MCP 可发现）。
	r2 := residentToolsFor("on", reg)
	got2 := map[string]bool{}
	for _, n := range r2 {
		got2[n] = true
	}
	if !got2["read_file"] || !got2["write_file"] || !got2["task"] || got2["mcp__rea__scan"] {
		t.Fatalf("on 核心清单错: %v", r2)
	}
	// 小型注册表（无 MCP/未超阈值）：auto=nil（标准会话零变化）。
	small := tool.NewRegistry()
	small.Add(stubDyn{name: "read_file"})
	small.Add(stubDyn{name: "mcp__tiny__one"})
	if r := residentToolsFor("auto", small); r != nil {
		t.Fatalf("auto 小面不应开启: %v", r)
	}
	if r := residentToolsFor("", small); r != nil {
		t.Fatalf("空模式回落 auto: %v", r)
	}
}

func TestWireDynamicToolsRegistersDiscover(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(stubDyn{name: "read_file"})
	reg.Add(stubDyn{name: "write_file"})
	reg.Add(stubDyn{name: "task"})
	reg.Add(stubDyn{name: "mcp__rea__scan", big: true})
	runner := agent.New(nil, reg, agent.NewSession(""), agent.Options{}, nil)
	if !wireDynamicTools("on", reg, runner) {
		t.Fatal("on 应开启")
	}
	if _, ok := reg.Get(agent.DiscoverToolName); !ok {
		t.Fatal("开启时应注册 discover_tools")
	}
	if !runner.DynamicToolsOn() {
		t.Fatal("runner 应装配目录")
	}
	names := map[string]bool{}
	for _, s := range runner.CurrentSchemas() {
		names[s.Name] = true
	}
	if !names[agent.DiscoverToolName] || !names["read_file"] {
		t.Fatalf("常驻面缺件: %v", names)
	}
	if names["mcp__rea__scan"] {
		t.Fatal("MCP 应可发现不进面")
	}
	// off：不注册、不装配。
	reg2 := tool.NewRegistry()
	reg2.Add(stubDyn{name: "read_file"})
	runner2 := agent.New(nil, reg2, agent.NewSession(""), agent.Options{}, nil)
	if wireDynamicTools("off", reg2, runner2) {
		t.Fatal("off 不应开启")
	}
	if _, ok := reg2.Get(agent.DiscoverToolName); ok {
		t.Fatal("关闭时不得注册 discover_tools（零 schema 增量）")
	}
}
