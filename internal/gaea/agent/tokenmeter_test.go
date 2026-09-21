package agent

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ─── 锚定式 token 计量（v4.381，dsh token-meter 蒸馏）──────────────────────

func meterTestRunner(t *testing.T) *AgentRunner {
	t.Helper()
	return New(&mockProvider{name: "p"}, tool.NewRegistry(), NewSession(""), Options{DisableVerify: true}, event.Discard)
}

// 锚定核心性质：比率源漂移（lastUsage 变化→tokPerChar 重校准）不影响已锚定
// 的未变消息——压力只随「新增/删除/改写」的消息移动，且重复读取零漂移。
func TestMeterAnchoredIgnoresRatioDrift(t *testing.T) {
	a := meterTestRunner(t)
	big := provider.Message{Role: provider.RoleSystem, Content: strings_rep(4000)}
	user := provider.Message{Role: provider.RoleUser, Content: "hello"}
	a.session.Add(big)
	a.session.Add(user)

	// 落锚：锚点时点 tokPerChar=fallback（无 lastUsage）。
	a.meter.anchor(a, 1000, a.session.Messages)
	if got := a.EstimateContextTokens(); got != 1000 {
		t.Fatalf("right after anchor, pressure = %d, want the real anchor 1000", got)
	}

	// 比率源漂移：lastUsage 换成小值 → tokPerChar 重校准。
	a.lastUsage.Store(&provider.Usage{PromptTokens: 100})

	// 新增消息：压力=锚点+新消息估价（新消息按当前比率估——正确语义），
	// 未变的老消息仍按锚点时点估价相消，不被新比率重算。
	a.session.Add(provider.Message{Role: provider.RoleUser, Content: strings_rep(400)})
	estNew := a.meter.msgEstLocked(a, provider.Message{Role: provider.RoleUser, Content: strings_rep(400)})
	if estNew >= 1000 {
		t.Fatalf("precondition: new small message should estimate far below the anchor, got %d", estNew)
	}
	if got, want := a.EstimateContextTokens(), 1000+estNew; got != want {
		t.Fatalf("after adding a message, pressure = %d, want anchor+delta = %d (ratio drift must not rescale old messages)", got, want)
	}
	// 重复读取零漂移。
	if again := a.EstimateContextTokens(); again != 1000+estNew {
		t.Fatalf("repeated read drifted: %d != %d", again, 1000+estNew)
	}

	// 移除该消息 → 精确回到锚点。
	a.session.Messages = a.session.Messages[:len(a.session.Messages)-1]
	if got := a.EstimateContextTokens(); got != 1000 {
		t.Fatalf("after removing the added message, pressure = %d, want exactly the anchor 1000", got)
	}
}

// 压缩重写场景：老消息被摘要替换，压力按「删锚点估价、加新估价」精确移动
// （上游「剪枝/压缩重写后估计不再漂移」的性质）。
func TestMeterReflectsCompactionRewrite(t *testing.T) {
	a := meterTestRunner(t)
	old := provider.Message{Role: provider.RoleTool, Content: strings_rep(8000), ToolCallID: "c1"}
	a.session.Add(provider.Message{Role: provider.RoleSystem, Content: "sys"})
	a.session.Add(old)
	a.meter.anchor(a, 5000, a.session.Messages)

	// 压缩：老消息被 200 字符摘要替换。
	summary := provider.Message{Role: provider.RoleUser, Content: strings_rep(200)}
	msgs := a.session.Messages
	a.session.Messages = append([]provider.Message{msgs[0], summary}, msgs[2:]...)

	oldEst := a.meter.msgEstLocked(a, old)
	sumEst := a.meter.msgEstLocked(a, summary)
	want := 5000 - oldEst + sumEst
	if got := a.EstimateContextTokens(); got != want {
		t.Fatalf("after rewrite, pressure = %d, want %d (anchor − old + summary)", got, want)
	}
}

// 差值为负（删得多）时钳非负。
func TestMeterClampsNegativePressure(t *testing.T) {
	a := meterTestRunner(t)
	a.session.Add(provider.Message{Role: provider.RoleSystem, Content: strings_rep(8000)})
	a.meter.anchor(a, 100, a.session.Messages)
	a.session.Messages = nil
	if got := a.EstimateContextTokens(); got != 0 {
		t.Fatalf("negative delta must clamp to 0, got %d", got)
	}
}

// 无锚（首个真实 usage 之前）回退裸字符估算——旧口径不变。
func TestMeterUnanchoredFallsBackToChars(t *testing.T) {
	a := meterTestRunner(t)
	a.session.Add(provider.Message{Role: provider.RoleUser, Content: strings_rep(400)})
	want := int(float64(charsOfMessages(a.session.Messages)) * fallbackTokPerChar)
	if got := a.EstimateContextTokens(); got != want {
		t.Fatalf("unanchored estimate = %d, want legacy chars×fallback = %d", got, want)
	}
	if a.meter.anchored {
		t.Fatal("estimate must not anchor implicitly")
	}
}

// maybeCompact 收到真实 usage 即落锚；LastPrompt 回退路径不落锚。
func TestMaybeCompactAnchorsOnRealUsage(t *testing.T) {
	a := meterTestRunner(t)
	a.compaction.Window = 1_000_000
	a.session.Add(provider.Message{Role: provider.RoleUser, Content: strings_rep(400)})

	a.maybeCompact(context.Background(), &provider.Usage{PromptTokens: 1234})
	if !a.meter.anchored || a.meter.anchorReal != 1234 {
		t.Fatalf("real usage must anchor the meter, anchored=%v real=%d", a.meter.anchored, a.meter.anchorReal)
	}

	// 无真实 usage（LastPrompt 回退路径）不重锚。
	a2 := meterTestRunner(t)
	a2.compaction.Window = 1_000_000
	a2.compaction.LastPrompt = 777
	a2.meter.anchor(a2, 5000, a2.session.Messages)
	a2.maybeCompact(context.Background(), nil)
	if a2.meter.anchorReal != 5000 {
		t.Fatalf("LastPrompt fallback must not re-anchor, real=%d", a2.meter.anchorReal)
	}
}

// strings_rep 生成 n 字符内容（避免与 strings 包名冲突的本地助手）。
func strings_rep(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}
