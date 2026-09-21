package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// ─── 刀1: 上下文溢出自愈（Reasonix/dsh compaction-basic 蒸馏）─────────────────

// fatSession builds a session long enough that compaction has a fold region.
func fatSession(t *testing.T, pairs int) *Session {
	t.Helper()
	s := NewSession("system prompt here")
	for i := 0; i < pairs; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "do task " + strings.Repeat("u", 20)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "working " + strings.Repeat("w", 200)})
	}
	return s
}

func TestTryOverflowRecoveryRecovers(t *testing.T) {
	s := fatSession(t, 30)
	a := testAgentC(4000, 0.8, 5)
	a.session = s
	version := s.RewriteVersion()

	if !a.tryOverflowRecovery(context.Background(), errors.New("400 context_length_exceeded: requested 5000 tokens")) {
		t.Fatal("overflow error should recover")
	}
	if a.overflowRecoveries != 1 {
		t.Fatalf("recovery counter = %d, want 1", a.overflowRecoveries)
	}
	if s.RewriteVersion() == version {
		t.Fatal("recovery must leave a durable shrink (rewrite version must advance)")
	}
	// cap：同一请求只自愈一次
	if a.tryOverflowRecovery(context.Background(), errors.New("context window exceeded")) {
		t.Fatal("second consecutive recovery should hit the cap")
	}
}

func TestTryOverflowRecoveryRejectsNonOverflow(t *testing.T) {
	s := fatSession(t, 5)
	a := testAgentC(4000, 0.8, 5)
	a.session = s
	version := s.RewriteVersion()

	if a.tryOverflowRecovery(context.Background(), errors.New("connection reset by peer")) {
		t.Fatal("non-overflow error must not trigger recovery")
	}
	if a.overflowRecoveries != 0 || s.RewriteVersion() != version {
		t.Fatal("non-overflow path must not consume budget or rewrite the session")
	}
}

func TestTryOverflowRecoveryNoWindowNoProgress(t *testing.T) {
	// 无窗口：压缩机制不可用，无从自愈
	a := testAgentC(0, 0.8, 5)
	a.session = fatSession(t, 5)
	if a.tryOverflowRecovery(context.Background(), errors.New("context limit reached")) {
		t.Fatal("no window means no self-heal")
	}

	// 有窗口但没有可折叠区（只有 system+user，head 钉住全部）：
	// 剪枝与压缩都无进展时必须如实放弃
	b := testAgentC(4000, 0.8, 5)
	s := NewSession("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	b.session = s
	version := s.RewriteVersion()
	if b.tryOverflowRecovery(context.Background(), errors.New("context window exceeded")) {
		t.Fatal("no-progress recovery must report failure")
	}
	if s.RewriteVersion() != version {
		t.Fatal("failed recovery must not rewrite the session")
	}
}

// ─── 刀2: 缓存对齐的摘要调用 ────────────────────────────────────────────────

func TestSummarizeRequestCacheAlignedShape(t *testing.T) {
	s := NewSession("L1 system line")
	s.Add(provider.Message{Role: provider.RoleSystem, Content: "L2 extra rules"})
	s.Add(provider.Message{Role: provider.RoleUser, Content: "first user turn"})
	fold := []provider.Message{
		{Role: provider.RoleUser, Content: "do the thing"},
		{Role: provider.RoleAssistant, Content: "working",
			ToolCalls: []provider.ToolCall{{ID: "t1", Name: "bash", Arguments: `{"command":"ls"}`}}},
		{Role: provider.RoleTool, Name: "bash", ToolCallID: "t1", Content: "file.txt"},
	}
	prov := &mockProvider{name: "sum", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: "brief"}}}
	a := &AgentRunner{session: s, sink: event.Discard, prov: prov}

	out, err := a.summarize(context.Background(), fold, "", false)
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if out != "brief" {
		t.Fatalf("summary = %q", out)
	}

	req := prov.lastReq
	got := req.Messages
	// 前缀 = 会话全部 system 消息（原样）+ fold 区间（逐字原样）+ 末尾指令
	if len(got) != 2+len(fold)+1 {
		t.Fatalf("request has %d messages, want %d (2 system + fold + instruction)", len(got), 2+len(fold)+1)
	}
	if got[0].Content != "L1 system line" || got[1].Content != "L2 extra rules" {
		t.Fatalf("system messages not replayed verbatim: %q / %q", got[0].Content, got[1].Content)
	}
	for i, m := range fold {
		if got[2+i].Role != m.Role || got[2+i].Content != m.Content || got[2+i].ToolCallID != m.ToolCallID {
			t.Fatalf("fold message %d not replayed verbatim: %+v", i, got[2+i])
		}
	}
	last := got[len(got)-1]
	if last.Role != provider.RoleUser {
		t.Fatalf("instruction must be the last user message, got %s", last.Role)
	}
	if !strings.Contains(last.Content, "Standing facts") || !strings.Contains(last.Content, summaryInstruction) {
		t.Fatal("instruction message must carry summaryInstruction")
	}
	if req.Temperature != 0 {
		t.Fatalf("summarizer temperature = %v, want 0", req.Temperature)
	}

	// 动态注入只允许进最后一条指令消息
	prov2 := &mockProvider{name: "sum2", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: "ok"}}}
	b := &AgentRunner{session: s, sink: event.Discard, prov: prov2}
	if _, err := b.summarize(context.Background(), fold, "EXTRA-CONTEXT-123", false); err != nil {
		t.Fatalf("summarize with instructions: %v", err)
	}
	m2 := prov2.lastReq.Messages
	if len(m2) != len(got) {
		t.Fatalf("instructions must not change the message count: %d vs %d", len(m2), len(got))
	}
	if !strings.Contains(m2[len(m2)-1].Content, "EXTRA-CONTEXT-123") {
		t.Fatal("dynamic instructions must ride in the final user message")
	}
	for i := 0; i < len(m2)-1; i++ {
		if m2[i].Content != got[i].Content {
			t.Fatalf("prefix message %d must be byte-stable across instructions", i)
		}
	}
}

// ─── 刀3: shrink 硬校验 ─────────────────────────────────────────────────────

func TestCompactShrinkCheckFallsBackToMechanical(t *testing.T) {
	// 折叠区以 assistant 长消息为主（短 user turn 会被 pin 保留）。
	s := NewSession("system prompt here")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "go"})
	for i := 0; i < 12; i++ {
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "result " + strings.Repeat("r", 200)})
	}
	a := testAgentC(2000, 0.8, 5)
	a.session = s
	// 摘要比被折叠区还长 → 退机械摘要
	huge := strings.Repeat("S", 8000)
	prov := &mockProvider{name: "fat", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: huge}}}
	a.prov = prov

	if err := a.CompactNow(context.Background(), ""); err != nil {
		t.Fatalf("CompactNow: %v", err)
	}
	foundDigest := false
	for _, m := range a.session.Messages {
		if strings.Contains(m.Content, "were folded here") {
			foundDigest = true
		}
		if strings.Contains(m.Content, huge) {
			t.Fatal("inflating LLM summary must not land in the session")
		}
	}
	if !foundDigest {
		t.Fatal("shrink check should fall back to the mechanical digest")
	}
}

func TestCompactShrinkCheckKeepsGoodSummary(t *testing.T) {
	s := NewSession("system prompt here")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "go"})
	for i := 0; i < 12; i++ {
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "result " + strings.Repeat("r", 200)})
	}
	a := testAgentC(2000, 0.8, 5)
	a.session = s
	prov := &mockProvider{name: "lean", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: "## Goal\nshort"}}}
	a.prov = prov

	if err := a.CompactNow(context.Background(), ""); err != nil {
		t.Fatalf("CompactNow: %v", err)
	}
	found := false
	for _, m := range a.session.Messages {
		if strings.Contains(m.Content, "## Goal") {
			found = true
		}
	}
	if !found {
		t.Fatal("a shrinking LLM summary should be kept verbatim")
	}
}

// ─── 刀4: force 路径多轮压缩 ────────────────────────────────────────────────

func TestCompactIfOverForceBoundedPasses(t *testing.T) {
	s := fatSession(t, 40)
	a := testAgentC(2000, 0.8, 5)
	a.session = s
	// 95% 窗口：越过 force 水位（90%），走多轮路径
	before := len(s.Messages)
	a.maybeCompact(context.Background(), &provider.Usage{PromptTokens: 1900})

	if len(a.session.Messages) >= before {
		t.Fatal("force path should shrink the session")
	}
	if a.consecutiveCompacts < 1 || a.consecutiveCompacts > 2 {
		t.Fatalf("consecutiveCompacts = %d, want 1..2", a.consecutiveCompacts)
	}
}
