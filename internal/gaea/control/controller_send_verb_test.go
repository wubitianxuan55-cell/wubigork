package control

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
)

// captureSink 收集事件供断言（controller_send_verb_test 用）。
type verbCaptureSink struct {
	kinds []event.Kind
	texts []string
}

func (s *verbCaptureSink) Emit(e event.Event) {
	s.kinds = append(s.kinds, e.Kind)
	if e.Text != "" {
		s.texts = append(s.texts, e.Text)
	}
}

// TestSendDispatchesSlashVerbs（v4.414.1 根修回归）：桌面 Send 路径此前绕过
// 全部斜杠动词——/plan 会被当普通文本开回合发给模型（真机走查实录）。
// Send 现与 Submit 共用 dispatchSlash：动词经 Notice 反馈且不开回合。
func TestSendDispatchesSlashVerbs(t *testing.T) {
	sink := &verbCaptureSink{}
	c := &Controller{sink: sink, executor: agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard)}

	c.Send("/plan on")
	if !c.executor.PlanMode() {
		t.Fatal("Send(/plan on) 应翻转计划模式")
	}
	if !joined(sink.texts, "计划模式已开启") {
		t.Fatalf("应有 Notice 反馈, got %v", sink.texts)
	}

	sink.kinds = nil
	sink.texts = nil
	c.Send("/plan status")
	if !joined(sink.texts, "计划模式：开") {
		t.Fatalf("/plan status 应报开, got %v", sink.texts)
	}

	sink.kinds = nil
	sink.texts = nil
	c.Send("/plan off")
	if c.executor.PlanMode() {
		t.Fatal("Send(/plan off) 应关闭计划模式")
	}

	// 未知名义：unknown Notice，不开回合（无 TurnStarted/TurnDone）。
	sink.kinds = nil
	sink.texts = nil
	c.Send("/definitely-not-a-verb")
	if !joined(sink.texts, "unknown command") {
		t.Fatalf("未知斜杠应报 unknown command, got %v", sink.texts)
	}
	for _, k := range sink.kinds {
		if k == event.TurnStarted || k == event.TurnDone {
			t.Fatalf("未知斜杠不应开回合, got %v", sink.kinds)
		}
	}
}

func joined(texts []string, substr string) bool {
	for _, t := range texts {
		if strings.Contains(t, substr) {
			return true
		}
	}
	return false
}
