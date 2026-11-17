package control

// T11 刀三验收：allow 放行入账 / deny 可追认 / 失败回落用户卡（红线）/
// 连续拒绝熔断 / hardAsk 通道不受机器评审影响。

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/guardian"
	"github.com/gaea/gaea/internal/gaea/permission"
)

// fakeGuardian 按脚本返回评审结论。
type fakeGuardian struct {
	results []guardian.Review // 逐次弹出；耗尽重复末条
	calls   int
}

func (f *fakeGuardian) Review(_ context.Context, _ guardian.Request) guardian.Review {
	f.calls++
	if len(f.results) == 0 {
		return guardian.Review{Err: guardian.ErrSession}
	}
	i := f.calls - 1
	if i >= len(f.results) {
		i = len(f.results) - 1
	}
	return f.results[i]
}

func allowReview() guardian.Review {
	return guardian.Review{Assessment: guardian.Assessment{
		Risk: guardian.RiskLow, Authorization: guardian.AuthHigh,
		Outcome: guardian.OutcomeAllow, Rationale: "与用户指令一致",
	}}
}

func denyReview(why string) guardian.Review {
	return guardian.Review{Assessment: guardian.Assessment{
		Risk: guardian.RiskCritical, Authorization: guardian.AuthUnknown,
		Outcome: guardian.OutcomeDeny, Rationale: why,
	}}
}

func newGuardianTestController(g guardian.Reviewer) (*Controller, *attentionRecorder, *permission.Gate) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec, Guardian: g, Policy: permission.New("ask", nil, nil, nil)})
	return c, rec, c.newGate()
}

const commitCmd = `{"command":"git commit -m wip"}`

// ① allow：不弹卡，放行；账本带三字段（risk/authorization/rationale）。
func TestGuardianAllowLedger(t *testing.T) {
	fg := &fakeGuardian{results: []guardian.Review{allowReview()}}
	c, rec, g := newGuardianTestController(fg)
	c.lastUserInput = "帮我提交当前修改"

	allow, _, err := g.Check(context.Background(), "bash", []byte(commitCmd), false)
	if err != nil || !allow {
		t.Fatalf("guardian allow 应放行: (%v,%v)", allow, err)
	}
	var led *event.AutoApproval
	for i := range rec.snapshot() {
		if e := rec.snapshot()[i]; e.Kind == event.ApprovalAuto {
			a := e.Auto
			led = &a
		}
	}
	if led == nil {
		t.Fatalf("应落 ApprovalAuto 账本")
	}
	if led.Source != event.ApprovalSourceGuardian || led.Decided != event.AutoDecidedGuardianAllow {
		t.Fatalf("账本来源/决策者不符: %+v", led)
	}
	if led.Risk != "low" || led.Authorization != "high" || led.Rationale != "与用户指令一致" {
		t.Fatalf("三字段账本不符: %+v", led)
	}
	// 授权证据源进评审请求。
	if fg.calls != 1 {
		t.Fatalf("评审应恰好调用一次, got %d", fg.calls)
	}
}

// ② deny：拒绝理由回喂模型；动作落待追认账本；追认后同形命令放行。
func TestGuardianDenyAndReinstate(t *testing.T) {
	fg := &fakeGuardian{results: []guardian.Review{denyReview("越权写面：与用户指令无关")}}
	c, rec, g := newGuardianTestController(fg)
	c.lastUserInput = "只读检查一下"

	allow, reason, err := g.Check(context.Background(), "bash", []byte(commitCmd), false)
	if err != nil || allow {
		t.Fatalf("guardian deny 应拒绝: (%v,%v)", allow, err)
	}
	if !strings.Contains(reason, "自动评审拒绝") || !strings.Contains(reason, "越权写面") {
		t.Fatalf("拒绝理由应回喂: %q", reason)
	}
	// 账本：guardian_deny 三字段。
	var led *event.AutoApproval
	for _, e := range rec.snapshot() {
		if e.Kind == event.ApprovalAuto && e.Auto.Decided == event.AutoDecidedGuardianDeny {
			a := e.Auto
			led = &a
		}
	}
	if led == nil || led.Risk != "critical" || led.Rationale != "越权写面：与用户指令无关" {
		t.Fatalf("deny 账本不符: %+v", led)
	}
	// 待追认账本（T3 复用）。
	acts := c.PendingDeniedActions()
	if len(acts) != 1 || acts[0].tool != "bash" || acts[0].reason != "越权写面：与用户指令无关" {
		t.Fatalf("deny 应落待追认账本: %+v", acts)
	}
	// 追认：弹用户卡（红线——由人裁决，不是机器复审），批准后同形放行。
	done := make(chan struct{})
	go func() {
		defer close(done)
		id := <-rec.approvals
		c.Approve(id, DecisionAllowOnce)
	}()
	c.ApproveDenied(acts[0].id)
	<-done
	if allow, _, _ := g.Check(context.Background(), "bash", []byte(commitCmd), false); !allow {
		t.Fatalf("追认后同形命令应放行")
	}
}

// ③ 红线：评审失败（超时/预算/解析/调用）回落用户审批卡，不产生 guardian
// 拒绝账本，也绝不自动放行。
func TestGuardianErrorFallsBackToUserCard(t *testing.T) {
	for _, rev := range []guardian.Review{
		{Err: guardian.ErrTimeout},
		{Err: guardian.ErrInputBudget},
		{Err: guardian.ErrParse},
		{Err: guardian.ErrSession},
	} {
		c, rec, g := newGuardianTestController(&fakeGuardian{results: []guardian.Review{rev}})
		go func() {
			id := <-rec.approvals // 用户卡弹出=红线守住
			c.Approve(id, DecisionDeny)
		}()
		allow, _, err := g.Check(context.Background(), "bash", []byte(commitCmd), false)
		if err != nil || allow {
			t.Fatalf("失败路径应由用户卡裁决: (%v,%v)", allow, err)
		}
		for _, e := range rec.snapshot() {
			if e.Kind == event.ApprovalAuto && e.Auto.Source == event.ApprovalSourceGuardian {
				t.Fatalf("评审失败不得产生 guardian 账本: %+v", e.Auto)
			}
		}
	}
}

// ④ 熔断：连续 3 次 deny → 第 3 次发熔断通知并中断回合（宿主执行 Cancel）；
// 之后用户卡照常可用（熔断不锁死人工通道）。
func TestGuardianBreakerInterrupts(t *testing.T) {
	fg := &fakeGuardian{results: []guardian.Review{denyReview("连续拒绝")}}
	c, rec, g := newGuardianTestController(fg)

	// 前两次：正常拒绝（无熔断通知）。
	for i := 0; i < 2; i++ {
		if allow, _, _ := g.Check(context.Background(), "bash", []byte(commitCmd), false); allow {
			t.Fatalf("deny 不得放行")
		}
	}
	for _, e := range rec.snapshot() {
		if e.Kind == event.Notice && strings.Contains(e.Text, "连续 3 次拒绝") {
			t.Fatalf("第 2 次不应熔断")
		}
	}
	// 第三次：熔断通知发出（Cancel 由宿主执行；无在跑回合时为安全 no-op）。
	if allow, _, _ := g.Check(context.Background(), "bash", []byte(commitCmd), false); allow {
		t.Fatalf("deny 不得放行")
	}
	tripped := false
	for _, e := range rec.snapshot() {
		if e.Kind == event.Notice && strings.Contains(e.Text, "连续 3 次拒绝，已中断本轮") {
			tripped = true
		}
	}
	if !tripped {
		t.Fatalf("第 3 次拒绝应触发熔断通知")
	}
	if c.guardianBreaker.Consecutive() != 3 {
		t.Fatalf("熔断计数 = %d, want 3", c.guardianBreaker.Consecutive())
	}
}

// ⑤ hardAsk 通道不受机器评审影响：guardian 只回 allow，cost_save 仍必须
// 弹用户卡（逐条确认硬纪律）。
func TestGuardianBypassesHardAsk(t *testing.T) {
	fg := &fakeGuardian{results: []guardian.Review{allowReview()}}
	c, rec, g := newGuardianTestController(fg)
	go func() {
		id := <-rec.approvals
		c.Approve(id, DecisionDeny)
	}()
	allow, _, err := g.Check(context.Background(), "cost_save", []byte(`{"title":"x"}`), false)
	if err != nil || allow {
		t.Fatalf("hardAsk 必须由用户卡裁决: (%v,%v)", allow, err)
	}
	for _, e := range rec.snapshot() {
		if e.Kind == event.ApprovalAuto && e.Auto.Source == event.ApprovalSourceGuardian {
			t.Fatalf("hardAsk 不得产生 guardian 账本")
		}
	}
}

// 评审失败打断熔断连续性（失败不是拒绝）。
func TestGuardianErrorResetsBreaker(t *testing.T) {
	fg := &fakeGuardian{results: []guardian.Review{denyReview("a"), denyReview("b"),
		{Err: guardian.ErrTimeout}}}
	c, _, g := newGuardianTestController(fg)
	rec := newAttentionRecorder()
	c.sink = rec
	// 两次 deny 后一次失败。
	_, _, _ = g.Check(context.Background(), "bash", []byte(commitCmd), false)
	_, _, _ = g.Check(context.Background(), "bash", []byte(commitCmd), false)
	go func() { c.Approve(<-rec.approvals, DecisionDeny) }()
	_, _, _ = g.Check(context.Background(), "bash", []byte(commitCmd), false)
	if c.guardianBreaker.Consecutive() != 0 {
		t.Fatalf("失败应打断连续性: %d", c.guardianBreaker.Consecutive())
	}
}
