package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ─── 刀5: 收尾判定（Reasonix v1.37 Deterministic natural-turn completion）──

// reasoning-only 的干净 stop 即最终回答——不再注入合成 user 重试消息。
func TestReasoningOnlyStopIsFinal(t *testing.T) {
	prov := &mockProvider{name: "p", chunks: []provider.Chunk{
		{Type: provider.ChunkReasoning, Text: "thinking hard about the answer..."},
		{Type: provider.ChunkDone},
	}}
	s := NewSession("")
	a := New(prov, tool.NewRegistry(), s, Options{DisableVerify: true}, event.Discard)

	res, err := a.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("reasoning-only stop should complete the turn: %v", err)
	}
	_ = res
	for _, m := range s.Messages {
		if m.Role == provider.RoleUser && strings.Contains(m.Content, "visible answer") {
			t.Fatal("reasoning-only stop must not trigger a synthetic retry prompt")
		}
	}
}

// 真零内容响应 = 冻结请求原样重放：不落库合成 user 消息，3 次后如实报错。
func TestZeroContentRetriesFrozenRequest(t *testing.T) {
	prov := &mockProvider{name: "p", chunks: []provider.Chunk{
		{Type: provider.ChunkDone},
	}}
	s := NewSession("")
	a := New(prov, tool.NewRegistry(), s, Options{DisableVerify: true}, event.Discard)

	_, err := a.Run(context.Background(), "hello")
	if err == nil || !strings.Contains(err.Error(), "without a visible final answer") {
		t.Fatalf("zero-content turn should fail honestly, got %v", err)
	}
	userMsgs := 0
	for _, m := range s.Messages {
		if m.Role == provider.RoleUser {
			userMsgs++
			if strings.Contains(m.Content, "visible answer") {
				t.Fatal("frozen replay must not inject a synthetic retry prompt")
			}
		}
	}
	if userMsgs != 1 {
		t.Fatalf("session should contain only the original user input, got %d user messages", userMsgs)
	}
}

// ─── 刀7: compactStuck 新消息解锁（Reasonix stuckInputHash 语义）────────────

func TestCompactStuckUnlocksOnNewMessages(t *testing.T) {
	s := NewSession("system prompt here")
	for i := 0; i < 20; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "task " + strings.Repeat("u", 30)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "work " + strings.Repeat("w", 120)})
	}
	a := testAgentC(2000, 0.8, 5)
	a.session = s
	// 模拟已卡死：计数与形状对齐当前会话
	a.compactStuck = true
	a.consecutiveCompacts = 2
	a.stuckAtMessages = len(s.Messages)

	// 新消息=新折叠边界：同水位重入应解锁并实际尝试（consecutiveCompacts 前进）
	s.Add(provider.Message{Role: provider.RoleUser, Content: "one more round " + strings.Repeat("n", 50)})
	a.compactIfOver(context.Background(), 1800, "mid-turn")

	if a.compactStuck && a.consecutiveCompacts == 2 && len(s.Messages) != a.stuckAtMessages {
		t.Fatal("new messages should unlock the stuck circuit breaker")
	}
}

// 会话未变时熔断维持。
func TestCompactStuckHoldsWhenSessionUnchanged(t *testing.T) {
	s := NewSession("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi " + strings.Repeat("u", 100)})
	a := testAgentC(2000, 0.8, 5)
	a.session = s
	a.compactStuck = true
	a.consecutiveCompacts = 2
	a.stuckAtMessages = len(s.Messages)

	a.compactIfOver(context.Background(), 1800, "auto")

	if !a.compactStuck {
		t.Fatal("unchanged session must keep the stuck breaker closed")
	}
}

// TestEmptyFinalRetryBackoffAllowsRecovery 空终答退避（2026-11-10 实弹根修）：
// 上游瞬时空响应——前两次真零内容、第三次正常回答；退避后应恢复成功而非
// 三连发打光。退避经 Options 注入小值（50ms）计时验证。
func TestEmptyFinalRetryBackoffAllowsRecovery(t *testing.T) {
	prov := &mockProvider{name: "p", streamFn: func(call int) []provider.Chunk {
		if call <= 2 {
			return []provider.Chunk{{Type: provider.ChunkDone}} // 真零内容
		}
		return []provider.Chunk{{Type: provider.ChunkText, Text: "recovered answer"}, {Type: provider.ChunkDone}}
	}}
	s := NewSession("")
	a := New(prov, tool.NewRegistry(), s, Options{
		DisableVerify:          true,
		EmptyFinalRetryBackoff: 50 * time.Millisecond,
	}, event.Discard)

	start := time.Now()
	res, err := a.Run(context.Background(), "hello")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("瞬态空响应后退避应恢复: %v", err)
	}
	if res.Summary != "recovered answer" {
		t.Fatalf("恢复后应取到最终回答: %q", res.Summary)
	}
	// 两次退避（1×50ms + 2×50ms）应真实发生
	if elapsed < 150*time.Millisecond {
		t.Fatalf("退避未生效（elapsed=%v < 150ms）", elapsed)
	}
}
