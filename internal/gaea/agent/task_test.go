package agent

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestFilterRegistryStripsPersistentWrites 回归：子代理注册表必须剔除
// 持久化写入工具（cost_save/remember/forget/knowledge_add/promote_session_facts/
// install_skill），防止子代理在 headless 审批通道上静默写入成本库/记忆/知识库/技能。
// T6-2.5：剔除依据是工具注册表的 PersistWrite 标记，不再是手写清单。
func TestFilterRegistryStripsPersistentWrites(t *testing.T) {
	reg := tool.NewRegistry()
	forbidden := map[string]bool{
		"cost_save": true, "remember": true, "forget": true,
		"knowledge_add": true, "promote_session_facts": true, "install_skill": true,
	}
	for _, name := range []string{
		"bash", "read_file", "memory_search",
		"cost_save", "remember", "forget", "knowledge_add", "promote_session_facts", "install_skill",
	} {
		reg.Add(fakeTool{name: name, persistWrite: forbidden[name]})
	}
	sub := FilterRegistry(reg, nil, SubagentMetaTools()...)
	names := sub.Names()
	for _, forbidden := range []string{
		"cost_save", "remember", "forget", "knowledge_add", "promote_session_facts", "install_skill",
	} {
		if _, ok := sub.Get(forbidden); ok {
			t.Fatalf("子代理注册表不应包含持久化写入工具 %q（names=%v）", forbidden, names)
		}
	}
	for _, kept := range []string{"bash", "read_file", "memory_search"} {
		if _, ok := sub.Get(kept); !ok {
			t.Fatalf("子代理注册表应保留研究/只读工具 %q", kept)
		}
	}
}

// TestTaskToolReturnsSubAgentFinalAnswer runs a task against a mock provider
// that emits a single text turn, and verifies the tool returns exactly that
// text — sub-agent intermediate state isn't supposed to leak.
func TestTaskToolReturnsSubAgentFinalAnswer(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "found 3 callers of Foo"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "test-sys-prompt", nil)

	out, err := task.Execute(context.Background(), []byte(`{"prompt":"find callers of Foo"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "found 3 callers of Foo") {
		t.Errorf("got %q, want sub-agent final answer", out)
	}

	// The sub-agent must have received the prompt as its user message and
	// the configured system prompt at the top — proving the session was
	// fresh, not the parent's.
	if sys := sub.lastReq.Messages[0]; sys.Role != provider.RoleSystem || sys.Content != "test-sys-prompt" {
		t.Errorf("first message = %+v, want system 'test-sys-prompt'", sys)
	}
	if got := lastUser(sub.lastReq); got != "find callers of Foo" {
		t.Errorf("sub-agent user = %q, want the prompt verbatim", got)
	}
}

// TestTaskToolFiltersTools verifies the whitelist behaviour: when the caller
// names a subset of tools, the sub-agent's registry contains exactly that set
// with subagent/skill meta-tools stripped to prevent recursive delegation.
func TestTaskToolFiltersTools(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "read_file", readOnly: true})
	parentReg.Add(fakeTool{name: "write_file", readOnly: false})
	parentReg.Add(fakeTool{name: "bash", readOnly: false})
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task) // simulate the wiring in cli.setup
	parentReg.Add(fakeTool{name: "run_skill", readOnly: false})
	parentReg.Add(fakeTool{name: "research", readOnly: false})

	args := []byte(`{"prompt":"x","tools":["read_file","task","write_file","run_skill","research"]}`)
	if _, err := task.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// V6.0: 子代理 API 请求发送过滤后工具（排除 meta-tools），
	// 参数白名单 [read_file, task, write_file, run_skill, research]
	// 过滤 meta-tools 后 → [read_file, write_file]
	got := map[string]bool{}
	for _, s := range sub.lastReq.Tools {
		got[s.Name] = true
	}
	if !got["read_file"] || !got["write_file"] {
		t.Errorf("V6.0: API request tools = %v, want [read_file, write_file]", got)
	}
	if got["task"] || got["run_skill"] || got["research"] {
		t.Errorf("V6.0: meta-tools should be excluded, got %v", got)
	}
}

// TestTaskToolDefaultsToParentToolsWithoutMetaTools covers the no-whitelist
// path: the sub-agent inherits parent tools except subagent/skill meta-tools.
func TestTaskToolDefaultsToParentToolsWithoutMetaTools(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}
	parentReg := tool.NewRegistry()
	parentReg.Add(fakeTool{name: "read_file", readOnly: true})
	parentReg.Add(fakeTool{name: "grep", readOnly: true})
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task)
	parentReg.Add(fakeTool{name: "run_skill", readOnly: false})
	parentReg.Add(fakeTool{name: "explore", readOnly: false})
	parentReg.Add(fakeTool{name: "research", readOnly: false})
	parentReg.Add(fakeTool{name: "review", readOnly: false})
	parentReg.Add(fakeTool{name: "security_review", readOnly: false})
	parentReg.Add(fakeTool{name: "remember", readOnly: false, persistWrite: true})

	if _, err := task.Execute(context.Background(), []byte(`{"prompt":"x"}`)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// 子代理默认继承父工具但排除 meta-tools 与持久化写入工具。
	// 父工具: [read_file, grep, task, run_skill, explore, research, review, security_review, remember]
	// 排除 meta-tools + 持久化写入后 → [read_file, grep]
	got := map[string]bool{}
	for _, s := range sub.lastReq.Tools {
		got[s.Name] = true
	}
	if !got["read_file"] || !got["grep"] {
		t.Errorf("default sub-agent API request tools = %v, want [read_file, grep]", got)
	}
	if got["remember"] {
		t.Errorf("persistent-write tool remember should be stripped from sub-agent, got %v", got)
	}
	if got["task"] || got["run_skill"] || got["explore"] || got["research"] || got["review"] || got["security_review"] {
		t.Errorf("V6.0: meta-tools should be excluded, got %v", got)
	}
}

// TestTaskToolPassesPricingToSubAgent verifies the sub-agent's Usage event
// carries the parent's Pricing so cost statistics are non-zero.
func TestTaskToolPassesPricingToSubAgent(t *testing.T) {
	pricing := &provider.Pricing{CacheHit: 0.025, Input: 3, Output: 6}
	sub := &mockProvider{
		name: "sub",
		chunks: []provider.Chunk{
			{Type: provider.ChunkText, Text: "ok"},
			{Type: provider.ChunkUsage, Usage: &provider.Usage{
				PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150,
				CacheHitTokens: 80, CacheMissTokens: 20,
			}},
			{Type: provider.ChunkDone},
		},
	}
	sink := &testSink{}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(sub, pricing, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task)

	ctx := withCallContext(context.Background(), "call_1", sink, nil)
	_, err := task.Execute(ctx, []byte(`{"prompt":"test pricing flow"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Find the last Usage event (sub-agent usage tagged as "subagent")
	var lastUsage *provider.Usage
	var lastPricing *provider.Pricing
	for _, e := range sink.events {
		if e.Kind == event.Usage && e.UsageSource == event.UsageSourceSubagent {
			lastUsage = e.Usage
			lastPricing = e.Pricing
		}
	}
	if lastUsage == nil {
		t.Fatal("sub-agent did not emit a Usage event with UsageSourceSubagent")
	}
	if lastPricing == nil {
		t.Fatal("sub-agent Usage event has nil Pricing — cost will be 0")
	}
	if lastPricing != pricing {
		t.Errorf("sub-agent Pricing = %+v, want parent pricing %+v", lastPricing, pricing)
	}
	cost := pricing.Cost(lastUsage)
	if cost <= 0 {
		t.Errorf("sub-agent cost = %v, want > 0", cost)
	}
	t.Logf("sub-agent cost = %v (tokens: prompt=%d completion=%d)", cost, lastUsage.PromptTokens, lastUsage.CompletionTokens)
}

// TestTaskToolSubagentPricingFallsBackToParent verifies that when subagent_model
// pricing is nil, it falls back to the parent's pricing.
func TestTaskToolSubagentPricingFallsBackToParent(t *testing.T) {
	parentPricing := &provider.Pricing{CacheHit: 0.025, Input: 3, Output: 6}
	sub := &mockProvider{
		name: "sub",
		chunks: []provider.Chunk{
			{Type: provider.ChunkText, Text: "ok"},
			{Type: provider.ChunkUsage, Usage: &provider.Usage{
				PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150,
			}},
			{Type: provider.ChunkDone},
		},
	}
	sub2 := &mockProvider{
		name: "sub2",
		chunks: []provider.Chunk{
			{Type: provider.ChunkText, Text: "ok"},
			{Type: provider.ChunkUsage, Usage: &provider.Usage{
				PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300,
			}},
			{Type: provider.ChunkDone},
		},
	}
	sink := &testSink{}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(sub, parentPricing, parentReg, 20, 0, 0.0, "", "sys", nil)
	// Set subagent model with nil pricing — should fall back to parentPricing
	task.SetSubagentProvider(sub2, nil, 0)
	parentReg.Add(task)

	ctx := withCallContext(context.Background(), "call_1", sink, nil)
	_, err := task.Execute(ctx, []byte(`{"prompt":"test fallback"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var lastUsage *provider.Usage
	var lastPricing *provider.Pricing
	for _, e := range sink.events {
		if e.Kind == event.Usage && e.UsageSource == event.UsageSourceSubagent {
			lastUsage = e.Usage
			lastPricing = e.Pricing
		}
	}
	if lastUsage == nil {
		t.Fatal("sub-agent did not emit a Usage event")
	}
	if lastPricing == nil {
		t.Fatal("sub-agent Pricing is nil — fallback to parent pricing failed")
	}
	if lastPricing != parentPricing {
		t.Errorf("sub-agent Pricing = %+v, want parent pricing %+v", lastPricing, parentPricing)
	}
	cost := parentPricing.Cost(lastUsage)
	if cost <= 0 {
		t.Errorf("sub-agent cost = %v, want > 0", cost)
	}
	t.Logf("fallback sub-agent cost = %v", cost)
}

// TestFilterRegistryAutoExcludesNewPersistWriteTool 验证：工具注册处新增带
// PersistWrite 标记的工具后，无需改动 FilterRegistry 即自动纳入子代理禁写集合
// （T6-2.5 注册表化：禁写清单由标记推导，不再手写）。
func TestFilterRegistryAutoExcludesNewPersistWriteTool(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	reg.Add(fakeTool{name: "ordinary_writer", readOnly: false})
	// 新增的持久化写工具：仅带标记，FilterRegistry 无需感知其名字。
	reg.Add(fakeTool{name: "new_persist_writer", readOnly: false, persistWrite: true})

	sub := FilterRegistry(reg, nil, SubagentMetaTools()...)
	if _, ok := sub.Get("new_persist_writer"); ok {
		t.Fatal("新标记的持久化写工具应自动纳入子代理禁写集合")
	}
	for _, kept := range []string{"read_file", "ordinary_writer"} {
		if _, ok := sub.Get(kept); !ok {
			t.Fatalf("非持久化写工具 %q 不应被剔除", kept)
		}
	}
}

// TestPersistWriteSetMatchesLegacySix 断言：现有 6 项持久化写入工具的标记
// 与 v2.13.21 手写禁写清单完全一致（cost_save/remember/forget/knowledge_add/
// promote_session_facts/install_skill），注册表推导出的集合一个不多一个不少。
func TestPersistWriteSetMatchesLegacySix(t *testing.T) {
	reg := tool.NewRegistry()
	// cost_save / knowledge_add 是内置工具（init 注册）；其余四个由 boot 构造。
	var nilStore memory.Store // Store 是接口，需要类型化 nil
	reg.Add(memory.NewRememberTool(nilStore, nil))
	reg.Add(memory.NewForgetTool(nilStore))
	reg.Add(memory.NewPromoteSessionFactsTool())
	reg.Add(skill.NewInstallSkillTool(nil, nil))
	if ct, ok := tool.LookupBuiltin("cost_save"); ok {
		reg.Add(ct)
	} else {
		t.Fatal("cost_save 未注册为内置工具")
	}
	if kt, ok := tool.LookupBuiltin("knowledge_add"); ok {
		reg.Add(kt)
	} else {
		t.Fatal("knowledge_add 未注册为内置工具")
	}

	got := reg.PersistWriteNames()
	want := []string{"cost_save", "remember", "forget", "knowledge_add", "promote_session_facts", "install_skill"}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("持久化写工具集合 = %v, want %v（现有 6 项集合必须保持不变）", got, want)
	}
}

// TestNonPersistWritersNotMarked 反向断言：普通读写工具（bash/write_file 等）
// 不应被误标为持久化写工具，避免子代理被过度剥夺能力。
func TestNonPersistWritersNotMarked(t *testing.T) {
	for _, name := range []string{"bash", "write_file", "read_file", "grep", "memory_search"} {
		if tool.IsPersistWrite(fakeTool{name: name, readOnly: false}) {
			t.Fatalf("%s 不应被标记为持久化写工具", name)
		}
	}
}


// testSink is a simple event sink for tests.
type testSink struct {
	events []event.Event
}

func (s *testSink) Emit(e event.Event) {
	s.events = append(s.events, e)
}
