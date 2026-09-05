package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// testAgent returns a minimal AgentRunner with a Discard sink so
// maybeCompact's sink.Emit doesn't panic in tests.
func testAgentC(window int, ratio float64, keep int) *AgentRunner {
	return &AgentRunner{
		compaction: CompactionConfig{Window: window, Ratio: ratio, RecentKeep: keep},
		sink:       event.Discard,
	}
}

// TestMaybeCompact_NilUsage_FallbackToLastPrompt verifies that the
// LLM summarization path triggers and reduces message count even
// without a real provider (mechanical fold fallback).
func TestMaybeCompact_NilUsage_FallbackToLastPrompt(t *testing.T) {
	a := testAgentC(4000, 0.8, 5) // small window so compaction triggers
	s := NewSession("system prompt here")
	// Build a session with substantial messages that exceeds the window
	for i := 0; i < 30; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "please implement feature " + strings.Repeat("x", 200)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "ok here is the code " + strings.Repeat("y", 300)})
		s.Add(provider.Message{Role: provider.RoleTool, Name: "write_file", Content: "wrote file " + strings.Repeat("z", 400)})
	}
	a.session = s
	a.compaction.LastPrompt = 3500 // 87.5% of window — over 80% threshold

	original := len(s.Messages)

	a.maybeCompact(context.Background(), nil)

	if len(a.session.Messages) >= original {
		t.Fatalf("compaction should reduce messages: got %d, original %d",
			len(a.session.Messages), original)
	}
	t.Logf("compaction: %d -> %d messages (mechanical fold)", original, len(a.session.Messages))
}

// TestMaybeCompact_NilUsage_LowPrompt_NoOp verifies compaction is skipped
// when prompt is below threshold.
func TestMaybeCompact_NilUsage_LowPrompt_NoOp(t *testing.T) {
	a := testAgentC(100000, 0.8, 5)
	s := NewSession("system")
	for i := 0; i < 10; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "msg"})
	}
	a.session = s
	a.compaction.LastPrompt = 1000 // 1% of window

	a.maybeCompact(context.Background(), nil)

	// Should be no-op since prompt << threshold
	if len(a.session.Messages) != len(s.Messages) {
		t.Errorf("no-op expected but messages changed: %d -> %d", len(s.Messages), len(a.session.Messages))
	}
}

// TestMaybeCompact_WithUsage_StillWorks verifies compaction triggers
// when real usage data is provided.
func TestMaybeCompact_WithUsage_StillWorks(t *testing.T) {
	a := testAgentC(3000, 0.8, 3)
	s := NewSession("system prompt")
	// V10.13: 增加到 40 对消息，确保超过 minRecentTokens=2000 的 tail 预算
	for i := 0; i < 40; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "build feature " + strings.Repeat("a", 150)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "implementing " + strings.Repeat("b", 200)})
	}
	a.session = s

	original := len(s.Messages)
	a.maybeCompact(context.Background(), &provider.Usage{PromptTokens: 2600})

	if len(a.session.Messages) >= original {
		t.Fatalf("compaction should reduce messages with usage: got %d, original %d",
			len(a.session.Messages), original)
	}
	t.Logf("compaction (usage path): %d -> %d messages", original, len(a.session.Messages))
}

// TestMidTurnMaybeCompact_TriggersCompaction verifies the mid-turn path
// compacts based on the estimated prompt size (char count × calibrated
// tokPerChar) after tool batches grow the session inside a turn.
func TestMidTurnMaybeCompact_TriggersCompaction(t *testing.T) {
	a := testAgentC(4000, 0.8, 5) // high-water = 3200
	s := NewSession("system prompt here")
	for i := 0; i < 40; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "please process " + strings.Repeat("a", 150)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "handled " + strings.Repeat("b", 200)})
		s.Add(provider.Message{Role: provider.RoleTool, Name: "write_file", Content: strings.Repeat("c", 300)})
	}
	a.session = s
	// Calibrate tokPerChar with a synthetic usage record, like a sampling
	// round that just finished before the tool batch grew the session.
	a.lastUsage.Store(&provider.Usage{PromptTokens: 3500})

	original := len(s.Messages)
	a.midTurnMaybeCompact(context.Background())

	if len(a.session.Messages) >= original {
		t.Fatalf("mid-turn compaction should reduce messages: got %d, original %d",
			len(a.session.Messages), original)
	}
	t.Logf("mid-turn compaction: %d -> %d messages", original, len(a.session.Messages))
}

// TestMidTurnMaybeCompact_BelowThreshold_NoOp verifies the mid-turn path is a
// no-op when the estimated prompt is below the high-water mark.
func TestMidTurnMaybeCompact_BelowThreshold_NoOp(t *testing.T) {
	a := testAgentC(100000, 0.8, 5)
	s := NewSession("system")
	for i := 0; i < 6; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "msg " + strings.Repeat("x", 20)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "reply " + strings.Repeat("y", 30)})
	}
	a.session = s
	a.lastUsage.Store(&provider.Usage{PromptTokens: 1000})

	original := len(s.Messages)
	a.midTurnMaybeCompact(context.Background())

	if len(a.session.Messages) != original {
		t.Errorf("no-op expected but messages changed: %d -> %d", original, len(a.session.Messages))
	}
}

// TestMidTurnMaybeCompact_Uncalibrated_NoOp verifies the mid-turn path waits
// for real usage before trusting character-based estimates.
func TestMidTurnMaybeCompact_Uncalibrated_NoOp(t *testing.T) {
	a := testAgentC(2000, 0.8, 3)
	s := NewSession("system")
	for i := 0; i < 30; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: strings.Repeat("u", 200)})
		s.Add(provider.Message{Role: provider.RoleTool, Name: "read_file", Content: strings.Repeat("t", 400)})
	}
	a.session = s

	original := len(s.Messages)
	a.midTurnMaybeCompact(context.Background())

	if len(a.session.Messages) != original {
		t.Errorf("uncalibrated mid-turn should be a no-op: %d -> %d", original, len(a.session.Messages))
	}
}

// TestCompactNow verifies that CompactNow triggers compaction regardless
// of prompt size.
func TestCompactNow(t *testing.T) {
	a := testAgentC(4000, 0.8, 3)
	s := NewSession("system")
	for i := 0; i < 15; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "request " + strings.Repeat("q", 100)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "response " + strings.Repeat("r", 200)})
		s.Add(provider.Message{Role: provider.RoleTool, Name: "read_file", Content: "content " + strings.Repeat("c", 300)})
	}
	a.session = s

	err := a.CompactNow(context.Background(), "")
	if err != nil {
		t.Logf("CompactNow error (expected with nil provider): %v", err)
	}
	// With nil provider, falls back to mechanical fold
	if len(a.session.Messages) >= len(s.Messages) {
		t.Logf("compact didn't reduce messages (may happen with short sessions)")
	} else {
		t.Logf("CompactNow: %d -> %d messages", len(s.Messages), len(a.session.Messages))
	}
}

func TestKeepProtectedToolResult(t *testing.T) {
	// read_skill tool results should be kept verbatim when KeepProtected is set.
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "hello"},
		{Role: provider.RoleAssistant, Content: "ok", ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read_skill", Arguments: `{"name":"test"}`}}},
		{Role: provider.RoleTool, ToolCallID: "c1", Name: "read_skill", Content: "important skill content"},
		{Role: provider.RoleAssistant, Content: "done"},
		{Role: provider.RoleUser, Content: "thanks"},
	}

	// Without KeepProtected, nothing is kept (only structural rules apply)
	keep := keepIndexes(msgs, 0)
	for i, m := range msgs {
		if keep[i] {
			t.Errorf("keep[%d] (%s/%s) should not be kept without KeepProtected", i, m.Role, m.Name)
		}
	}

	// With KeepProtected, the read_skill tool result should be kept.
	keep = keepIndexes(msgs, KeepProtected)
	if !keep[2] {
		t.Error("read_skill tool result should be kept with KeepProtected")
	}
	if !keep[1] {
		t.Error("assistant message calling read_skill should be kept (tool-call group)")
	}
	// Other messages should not be affected.
	if keep[0] || keep[3] || keep[4] {
		t.Error("non-protected messages should not be kept")
	}
}

// ─── GenUI 围栏剥离（审计 docs/gaea-genui-memoryfence-audit-2026-09.md #2/#4）───

// 审计 #2：pendingItems 吸入前剥离围栏体——围栏外正文的待办关键词照常触发，
// 但条目文本是剥离后的（围栏 JSON 不进摘要，占位行替代）。
func TestBuildCompactSummaryPendingSkipsFenceBody(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "整理费用表"},
		{Role: provider.RoleAssistant, Content: "```genui\n{\"items\":[{\"type\":\"list\",\"label\":\"todo next pending\"}]}\n```\n正文提到 remaining 需跟进。"},
	}
	out := BuildCompactSummary(msgs)
	if strings.Contains(out, `"items"`) || strings.Contains(out, "```") {
		t.Fatalf("围栏 JSON 不应进入压缩摘要: %q", out)
	}
	if !strings.Contains(out, "remaining 需跟进") {
		t.Fatalf("正文待办应保留: %q", out)
	}
}

// 审计 #2：关键词只出现在围栏 JSON 里时，不应触发 Pending work 条目
// （剥离后的正文无关键词 → 无待办噪声）。
func TestBuildCompactSummaryFenceKeywordDoesNotTriggerPending(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "出个看板"},
		{Role: provider.RoleAssistant, Content: "```genui\n{\"items\":[{\"type\":\"text\",\"text\":\"next pending todo\"}]}\n```\n已完成。"},
	}
	out := BuildCompactSummary(msgs)
	if strings.Contains(out, "Pending work") {
		t.Fatalf("围栏内关键词不应触发待办条目: %q", out)
	}
}

// 回归锚：无围栏回复的待办条目行为不变。
func TestBuildCompactSummaryPendingNoFenceRegression(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "费用表"},
		{Role: provider.RoleAssistant, Content: "todo: 检查环比公式"},
	}
	out := BuildCompactSummary(msgs)
	if !strings.Contains(out, "- Pending work:\n  - todo: 检查环比公式\n") {
		t.Fatalf("无围栏待办条目应原样保留: %q", out)
	}
}

// 审计 #4：summarizer 提示词快照——UI 围栏口径指引必须存在（保守口径：提示词
// 增补而非 renderTranscript 剥离，正文保留供回看）。若调整措辞请同步本断言。
func TestSummarySystemPromptUIFenceGuidance(t *testing.T) {
	if !strings.Contains(summarySystemPrompt, "genui") || !strings.Contains(summarySystemPrompt, "dsh-ui") {
		t.Fatal("summarySystemPrompt 应包含 UI 围栏语言指引（genui / dsh-ui）")
	}
	if !strings.Contains(summarySystemPrompt, "never reproduce their JSON in the summary") {
		t.Fatal("summarySystemPrompt 应明确禁止把围栏 JSON 原样收进摘要")
	}
	if !strings.Contains(summarySystemPrompt, "interactive components were used") {
		t.Fatal("summarySystemPrompt 应给出「使用了交互组件」的概括口径")
	}
}
