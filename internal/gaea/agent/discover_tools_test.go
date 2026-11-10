package agent

// A2 动态工具目录测试（dsh-ops 蒸馏 2026-11-10）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func newDynamicRunner(t *testing.T) *AgentRunner {
	t.Helper()
	reg := tool.NewRegistry()
	reg.Add(envelopeStub{name: "read_file", readOnly: true})
	reg.Add(envelopeStub{name: "write_file", readOnly: false})
	reg.Add(envelopeStub{name: "chart_gen", readOnly: false, result: "chart"})
	reg.Add(envelopeStub{name: "diagram_gen", readOnly: false, result: "diagram"})
	reg.Add(envelopeStub{name: "format_convert", readOnly: true, result: "converted"})
	reg.Add(NewDiscoverToolsTool())
	a := New(nil, reg, NewSession(""), Options{
		ResidentTools: []string{"read_file", "write_file", DiscoverToolName},
	}, event.Discard)
	return a
}

func dynSchemaNames(schemas []provider.ToolSchema) map[string]bool {
	out := make(map[string]bool, len(schemas))
	for _, s := range schemas {
		out[s.Name] = true
	}
	return out
}

func TestDynamicToolsetFiltersSchemas(t *testing.T) {
	a := newDynamicRunner(t)
	got := dynSchemaNames(a.currentSchemas())
	if !got["read_file"] || !got["write_file"] || !got[DiscoverToolName] {
		t.Fatalf("常驻三件缺: %v", got)
	}
	if got["chart_gen"] || got["format_convert"] {
		t.Fatalf("未激活不得进 schema 面: %v", got)
	}
	// 目录关闭（缺省 Options）：全量 schema（既有行为零变化）。
	reg := tool.NewRegistry()
	reg.Add(envelopeStub{name: "x_tool", readOnly: true})
	off := New(nil, reg, NewSession(""), Options{}, event.Discard)
	if got := dynSchemaNames(off.currentSchemas()); !got["x_tool"] {
		t.Fatalf("目录关闭应全量常驻: %v", got)
	}
}

func TestDynamicToolsetRefusesUnactivated(t *testing.T) {
	a := newDynamicRunner(t)
	o := a.executeOne(context.Background(), provider.ToolCall{Name: "chart_gen", Arguments: `{}`})
	if o.errMsg == "" {
		t.Fatal("未激活调用应软拒")
	}
	env, ok := tool.ParseEnvelope(o.output)
	if !ok || env.OK {
		t.Fatalf("应失败信封: %q", o.output)
	}
	if env.Subcode != "TOOL_NOT_ACTIVATED" {
		t.Fatalf("子码错: %+v", env)
	}
	if !strings.Contains(env.NextAction, "discover_tools") {
		t.Fatalf("应引导 discover: %q", env.NextAction)
	}
	// 常驻工具不受影响。
	if o := a.executeOne(context.Background(), provider.ToolCall{Name: "read_file", Arguments: `{}`}); o.errMsg != "" {
		t.Fatalf("常驻工具应放行: %+v", o)
	}
}

func TestDiscoverActivatesNextStep(t *testing.T) {
	a := newDynamicRunner(t)
	dt := NewDiscoverToolsTool()
	// query 词法匹配 chart。
	out, err := dt.Execute(WithRunner(context.Background(), a), json.RawMessage(`{"query":"chart 图表"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "chart_gen") || !strings.Contains(out, "下一模型步生效") {
		t.Fatalf("激活回执错: %q", out)
	}
	if !strings.Contains(out, "可见性≠授权") {
		t.Fatalf("授权口径缺失: %q", out)
	}
	// 激活后 schema 面扩展（下一模型步=currentSchemas 重读）。
	got := dynSchemaNames(a.currentSchemas())
	if !got["chart_gen"] {
		t.Fatalf("激活后应进 schema 面: %v", got)
	}
	if got["format_convert"] {
		t.Fatalf("未激活仍不得进: %v", got)
	}
	// 激活后可调用。
	if o := a.executeOne(context.Background(), provider.ToolCall{Name: "chart_gen", Arguments: `{}`}); o.errMsg != "" {
		t.Fatalf("激活后应放行: %+v", o)
	}
	// 词法确实不命中时：无匹配+目录兜底。
	out2, err := dt.Execute(WithRunner(context.Background(), a), json.RawMessage(`{"query":"zzz_no_such_thing"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "无匹配") || !strings.Contains(out2, "目录") {
		t.Fatalf("无匹配应给目录: %q", out2)
	}
}

func TestDiscoverSelectExactAndLimits(t *testing.T) {
	a := newDynamicRunner(t)
	dt := NewDiscoverToolsTool()
	// select 精确名+未知名+已常驻名混合。
	out, err := dt.Execute(WithRunner(context.Background(), a),
		json.RawMessage(`{"select":["format_convert","write_file","no_such_tool"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "format_convert") || !strings.Contains(out, "no_such_tool") {
		t.Fatalf("select 回执错: %q", out)
	}
	// 激活上限 5：6 个候选只激活 5。
	reg := a.tools
	for i := 0; i < 6; i++ {
		reg.Add(envelopeStub{name: fmt.Sprintf("extra%d_tool", i), readOnly: true})
	}
	out2, err := dt.Execute(WithRunner(context.Background(), a),
		json.RawMessage(`{"select":["extra0_tool","extra1_tool","extra2_tool","extra3_tool","extra4_tool","extra5_tool"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out2, "extra5_tool —") || !strings.Contains(out2, "extra0_tool") {
		t.Fatalf("每次≤5 激活上限失效: %q", out2)
	}
}

func TestDiscoverCatalogAndOffReply(t *testing.T) {
	a := newDynamicRunner(t)
	dt := NewDiscoverToolsTool()
	out, err := dt.Execute(WithRunner(context.Background(), a), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "chart_gen") || !strings.Contains(out, "format_convert") {
		t.Fatalf("目录应列可发现工具: %q", out)
	}
	// 目录关闭的 runner：全量常驻回复。
	reg := tool.NewRegistry()
	reg.Add(envelopeStub{name: "y_tool", readOnly: true})
	off := New(nil, reg, NewSession(""), Options{}, event.Discard)
	out2, err := dt.Execute(WithRunner(context.Background(), off), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "全量常驻") {
		t.Fatalf("关闭态回复错: %q", out2)
	}
	// 无 runner 直调：validation 拒。
	if _, err := dt.Execute(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("无 runner 应拒")
	}
}

func TestSeedActivatedFromHistory(t *testing.T) {
	a := newDynamicRunner(t)
	// 历史里模型曾调用 diagram_gen（in-history 播种：不进 TOOL_NOT_ACTIVATED）。
	a.session.Add(provider.Message{Role: provider.RoleAssistant,
		ToolCalls: []provider.ToolCall{{ID: "c1", Name: "diagram_gen", Arguments: `{}`}}})
	a.seedActivatedFromHistory()
	got := dynSchemaNames(a.currentSchemas())
	if !got["diagram_gen"] {
		t.Fatalf("历史调用应播种激活: %v", got)
	}
	// 未知名（注册表不在册）不播种。
	a.session.Add(provider.Message{Role: provider.RoleAssistant,
		ToolCalls: []provider.ToolCall{{ID: "c2", Name: "ghost_tool", Arguments: `{}`}}})
	a.seedActivatedFromHistory()
	if got := dynSchemaNames(a.currentSchemas()); got["ghost_tool"] {
		t.Fatal("未知名不得播种")
	}
}
