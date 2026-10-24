package agent

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/tool"
	_ "github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// newSteerRunner 建一个不真正跑模型的最小 runner（直接操纵 steer 队列）。
func newSteerRunner(t *testing.T) *AgentRunner {
	t.Helper()
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "bash", readOnly: false})
	return New(&scriptedProvider{name: "p"}, reg, NewSession(""), Options{}, event.Discard)
}

// TestSteerAcceptGate 钉 B3 收尾闸：intake 开=入队成功；drain（关门+清队）
// 之后 SteerAccept 如实拒绝——controller 凭 false 降级普通 Send，不再有
// 「入队即被退出路径清掉」的丢失窗。
func TestSteerAcceptGate(t *testing.T) {
	a := newSteerRunner(t)
	if !a.SteerAccept("先跑测试再改") {
		t.Fatal("open intake must accept steer")
	}
	a.steerMu.Lock()
	if len(a.steerQueue) != 1 {
		t.Fatalf("queue = %d, want 1", len(a.steerQueue))
	}
	a.steerMu.Unlock()

	a.drainSteerOnExit() // 退出路径：关门 + 收据
	if a.SteerAccept("太晚了") {
		t.Fatal("closed intake must reject steer (controller will fall back to Send)")
	}
	a.steerMu.Lock()
	if len(a.steerQueue) != 0 {
		t.Fatalf("closed intake must not queue: %d", len(a.steerQueue))
	}
	a.steerMu.Unlock()
}

// TestDrainSteerOnExitRecordsUnapplied 钉收据语义：未消费 steer 落会话记录
// （合成轮前缀，A4 回扫跳过），重复 drain 不产生重复记录，纯文本轮
// （pendingRecovery 无工具对）不误挂交接块。
func TestDrainSteerOnExitRecordsUnapplied(t *testing.T) {
	a := newSteerRunner(t)
	a.Steer("补充指引 A")
	a.Steer("补充指引 B")
	a.drainSteerOnExit()

	var found int
	for _, m := range a.session.Messages {
		if strings.HasPrefix(m.Content, unappliedSteerPrefix) {
			found++
			if !isSyntheticUserTurn(m.Content) {
				t.Errorf("unapplied steer must be recognized as synthetic turn: %q", m.Content[:40])
			}
			if !strings.Contains(m.Content, "takes precedence") {
				t.Errorf("record must carry the not-a-new-task guard: %q", m.Content[:80])
			}
		}
	}
	if found != 2 {
		t.Fatalf("unapplied steer records = %d, want 2", found)
	}
	a.steerMu.Lock()
	if len(a.steerQueue) != 0 {
		t.Fatalf("queue must be drained: %d", len(a.steerQueue))
	}
	a.steerMu.Unlock()

	countRecords := func() int {
		n := 0
		for _, m := range a.session.Messages {
			if strings.HasPrefix(m.Content, unappliedSteerPrefix) {
				n++
			}
		}
		return n
	}
	before := countRecords()
	a.drainSteerOnExit() // 幂等：二次 drain 不追加
	if after := countRecords(); after != before {
		t.Fatalf("double drain duplicated records: %d -> %d", before, after)
	}
	if a.pendingRecovery != nil {
		t.Fatal("drain alone must not fabricate a recovery handover (no tool pairs)")
	}
}
