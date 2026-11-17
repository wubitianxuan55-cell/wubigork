package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// T9（codex 蒸馏）验收：无摘要占位 + 起止事件 / [[keep]] 用户目标保留 /
// 窗口序号单调 / 开关关时 no-op / 保留项预算截断走既有机制。

// 事件采集 sink。
type captureSink struct{ events []event.Event }

func (c *captureSink) Emit(e event.Event) { c.events = append(c.events, e) }

func newWindowTestAgent(t *testing.T, sink event.Sink) *AgentRunner {
	t.Helper()
	a := &AgentRunner{
		compaction: CompactionConfig{Window: 2000, Ratio: 0.8, RecentKeep: 2, NewWindow: true},
		sink:       sink,
	}
	return a
}

func bigHistory() *Session {
	s := NewSession("system prompt here")
	for i := 0; i < 30; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "[[keep]] 目标：交付导出功能 " + strings.Repeat("x", 150)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "ok code " + strings.Repeat("y", 400)})
		s.Add(provider.Message{Role: provider.RoleTool, Name: "write_file", Content: "wrote " + strings.Repeat("z", 500)})
	}
	return s
}

// ① 换窗口压缩后历史里没有摘要占位消息（summaryTagOpen 不出现），但有
// CompactionStarted/Done 事件对，Done 的 Summary 为空（空摘要标记）。
func TestNewWindowCompactNoSummaryPlaceholder(t *testing.T) {
	sink := &captureSink{}
	a := newWindowTestAgent(t, sink)
	a.session = bigHistory()
	did, err := a.NewWindowCompact(context.Background())
	if err != nil || !did {
		t.Fatalf("NewWindowCompact = (%v,%v), want 压缩发生", did, err)
	}
	for _, m := range a.session.Messages {
		if strings.Contains(m.Content, summaryTagOpen) {
			t.Fatalf("换窗口形态不得有摘要占位消息: %q", m.Content[:min(60, len(m.Content))])
		}
	}
	var started, done *event.Event
	for i := range sink.events {
		switch sink.events[i].Kind {
		case event.CompactionStarted:
			started = &sink.events[i]
		case event.CompactionDone:
			done = &sink.events[i]
		}
	}
	if started == nil || done == nil {
		t.Fatalf("缺起止事件: %+v", sink.events)
	}
	if done.Compaction.Summary != "" {
		t.Fatalf("空摘要标记缺失: %q", done.Compaction.Summary)
	}
	if done.Compaction.Trigger != "new_window" {
		t.Fatalf("触发标签不符: %q", done.Compaction.Trigger)
	}
}

// ② [[keep]] 用户目标不丢（input_goal_ids 继承的 gaea 折算）。
func TestNewWindowCompactKeepsUserGoals(t *testing.T) {
	a := newWindowTestAgent(t, event.Discard)
	a.session = bigHistory()
	if _, err := a.NewWindowCompact(context.Background()); err != nil {
		t.Fatalf("NewWindowCompact: %v", err)
	}
	found := false
	for _, m := range a.session.Messages {
		if m.Role == provider.RoleUser && strings.Contains(m.Content, "[[keep]] 目标：交付导出功能") {
			found = true
		}
	}
	if !found {
		t.Fatalf("[[keep]] 用户目标应保留在白名单保留项里")
	}
}

// ③ 窗口序号单调递增；二次压缩序号 +1。
func TestNewWindowCompactWindowNumberMonotonic(t *testing.T) {
	sink := &captureSink{}
	a := newWindowTestAgent(t, sink)
	a.session = bigHistory()
	if _, err := a.NewWindowCompact(context.Background()); err != nil {
		t.Fatalf("第一次: %v", err)
	}
	first := 0
	for i := range sink.events {
		if sink.events[i].Kind == event.CompactionDone {
			first = sink.events[i].Compaction.Window
		}
	}
	if first != 1 {
		t.Fatalf("首个窗口序号 = %d, want 1", first)
	}
	// 再灌一段历史压第二次。
	for i := 0; i < 30; i++ {
		a.session.Add(provider.Message{Role: provider.RoleUser, Content: "more " + strings.Repeat("q", 300)})
		a.session.Add(provider.Message{Role: provider.RoleAssistant, Content: "ans " + strings.Repeat("w", 300)})
	}
	if _, err := a.NewWindowCompact(context.Background()); err != nil {
		t.Fatalf("第二次: %v", err)
	}
	second := 0
	for i := range sink.events {
		if sink.events[i].Kind == event.CompactionDone {
			second = sink.events[i].Compaction.Window
		}
	}
	if second != first+1 {
		t.Fatalf("窗口序号应单调: %d → %d", first, second)
	}
}

// ④ 开关关时 no-op（历史逐字节不变）。
func TestNewWindowCompactDisabledNoOp(t *testing.T) {
	a := newWindowTestAgent(t, event.Discard)
	a.compaction.NewWindow = false
	a.session = bigHistory()
	before := len(a.session.Messages)
	did, err := a.NewWindowCompact(context.Background())
	if err != nil || did {
		t.Fatalf("开关关应 no-op: (%v,%v)", did, err)
	}
	if len(a.session.Messages) != before {
		t.Fatalf("no-op 不得动历史")
	}
}

// ⑤ 无物可压（历史太小，规划不出折叠区）→ no-op 且不动历史。
func TestNewWindowCompactNothingToFold(t *testing.T) {
	sink := &captureSink{}
	a := newWindowTestAgent(t, sink)
	a.session = NewSession("system")
	a.session.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	a.session.Add(provider.Message{Role: provider.RoleAssistant, Content: "hello"})
	did, err := a.NewWindowCompact(context.Background())
	if err != nil || did {
		t.Fatalf("小历史应 no-op: (%v,%v)", did, err)
	}
	if len(a.session.Messages) != 3 {
		t.Fatalf("历史不得变动")
	}
	for _, e := range sink.events {
		if e.Kind == event.CompactionDone {
			t.Fatalf("no-op 不应发 Done 事件")
		}
	}
}

// ⑥ 审计补（2026-11-17）：自动压缩链真实接线——compaction_new_window=true
// 且超阈值时走换窗口形态（无摘要占位），关时走摘要式（现状）。
// 用无 [[keep]] 的普通历史（[[keep]] 钉前缀会触发 compact 的经济学拒绝，
// 那是摘要式自身的闸，与本接线断言无关）。
func TestNewWindowWiredIntoAutoCompact(t *testing.T) {
	run := func(enabled bool) (msgs int, hasSummary bool) {
		sink := &captureSink{}
		a := &AgentRunner{
			compaction: CompactionConfig{Window: 4000, Ratio: 0.8, RecentKeep: 5, NewWindow: enabled},
			sink:       sink,
		}
		s := NewSession("system prompt here")
		for i := 0; i < 30; i++ {
			s.Add(provider.Message{Role: provider.RoleUser, Content: "please implement feature " + strings.Repeat("x", 200)})
			s.Add(provider.Message{Role: provider.RoleAssistant, Content: "ok here is the code " + strings.Repeat("y", 300)})
			s.Add(provider.Message{Role: provider.RoleTool, Name: "write_file", Content: "wrote file " + strings.Repeat("z", 400)})
		}
		a.session = s
		a.compaction.LastPrompt = 3500 // 4000×0.8=3200 触发线之上
		a.maybeCompact(context.Background(), nil)
		for _, m := range a.session.Messages {
			if strings.Contains(m.Content, summaryTagOpen) {
				hasSummary = true
			}
		}
		return len(a.session.Messages), hasSummary
	}
	onMsgs, onSummary := run(true)
	if onSummary {
		t.Fatalf("开关开应走换窗口形态（无摘要占位）")
	}
	if onMsgs >= 90 {
		t.Fatalf("换窗口应折叠历史: %d", onMsgs)
	}
	offMsgs, offSummary := run(false)
	if !offSummary || offMsgs >= 90 {
		t.Fatalf("开关关应走摘要式压缩（现状）: msgs=%d summary=%v", offMsgs, offSummary)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
