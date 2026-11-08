package agent

// 2026-11-08 众神殿办公板块子代理裁决刀3：task 逐次工具级只读——
//   - readonly=true 的调用注册表交集收窄到自报 ReadOnly 的工具；
//   - 已知写工具硬名单优先于自报（误标写工具仍拦）；
//   - 模型面 ActiveSchemas 与受限注册表同源（「看得见但调用即拒」根除）；
//   - 默认 false = 既有行为零变化（写工具照常继承、父全量 schema 对齐缓存）。

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// readonlyFixture 父注册表：两只读工具 + 两写工具 + 一个误标只读的写工具
// （edit 落硬名单，构造「自报只读但实为写面」的越权候选）。
func readonlyFixture() (*TaskTool, *mockProvider) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "read_file", readOnly: true})
	parentReg.Add(fakeTool{name: "web_search", readOnly: true})
	parentReg.Add(fakeTool{name: "write_file", readOnly: false})
	parentReg.Add(fakeTool{name: "bash", readOnly: false})
	parentReg.Add(fakeTool{name: "edit", readOnly: true}) // 误标（硬名单应拦）
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task)
	return task, sub
}

func schemaNames(schemas []provider.ToolSchema) []string {
	out := make([]string, 0, len(schemas))
	for _, s := range schemas {
		out = append(out, s.Name)
	}
	return out
}

func containsName(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

func TestTaskReadOnlyRegistry(t *testing.T) {
	task, _ := readonlyFixture()
	reg, err := task.buildSubReg(nil, true)
	if err != nil {
		t.Fatalf("buildSubReg: %v", err)
	}
	for _, want := range []string{"read_file", "web_search"} {
		if _, ok := reg.Get(want); !ok {
			t.Errorf("只读子代理应保留 %q（names=%v）", want, reg.Names())
		}
	}
	for _, denied := range []string{"write_file", "bash", "edit"} {
		if _, ok := reg.Get(denied); ok {
			t.Errorf("只读子代理不应含 %q（names=%v）", denied, reg.Names())
		}
	}
	if _, ok := reg.Get(completeSubtaskName); !ok {
		t.Error("complete_subtask 应常驻子代理注册表")
	}
}

// TestTaskReadOnlyHardDenyListBeatsSelfReport 风险条：ReadOnly() 是工具自报
// 属性，误标写工具仍可能越权——硬名单不看自报，命中即拒。
func TestTaskReadOnlyHardDenyListBeatsSelfReport(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "bash", readOnly: true}) // 误标
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	reg, err := task.buildSubReg(nil, true)
	if err != nil {
		t.Fatalf("buildSubReg: %v", err)
	}
	if _, ok := reg.Get("bash"); ok {
		t.Error("误标 readOnly 的 bash 应被硬名单拒绝（名单优先于自报）")
	}
}

// TestTaskReadOnlyActiveSchemasMatchScope 端到端：readonly=true 的调用发给
// 模型的工具面=受限注册表同源（写工具从模型面消失，纠偏循环根除）。
func TestTaskReadOnlyActiveSchemasMatchScope(t *testing.T) {
	task, sub := readonlyFixture()
	args, _ := json.Marshal(map[string]any{"prompt": "explore read-only", "readonly": true})
	if _, err := task.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	names := schemaNames(sub.lastReq.Tools)
	if !containsName(names, "read_file") {
		t.Errorf("只读模型面应含 read_file（got %v）", names)
	}
	for _, denied := range []string{"write_file", "bash", "edit"} {
		if containsName(names, denied) {
			t.Errorf("只读模型面不应含 %q（got %v）", denied, names)
		}
	}
}

// TestTaskDefaultKeepsParentSchemaAlignment 默认（非只读）零行为变化：写工具
// 照常继承、模型面=父注册表全量（父→子前缀缓存对齐既有契约）。
func TestTaskDefaultKeepsParentSchemaAlignment(t *testing.T) {
	task, sub := readonlyFixture()
	args, _ := json.Marshal(map[string]any{"prompt": "work"})
	if _, err := task.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	names := schemaNames(sub.lastReq.Tools)
	for _, want := range []string{"read_file", "web_search", "write_file", "bash", "edit"} {
		if !containsName(names, want) {
			t.Errorf("默认模型面应含父全量工具 %q（got %v）", want, names)
		}
	}
}
