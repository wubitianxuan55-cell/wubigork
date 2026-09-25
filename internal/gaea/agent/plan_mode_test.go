package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// ── 计划模式审批门（v4.414 plan-mode 蒸馏）──────────────────────────

// stubAsker 固定回答的假问询通道（plan-review 问「批准」或「继续计划」）。
type stubAsker struct{ label string }

func (s stubAsker) Ask(_ context.Context, qs []event.AskQuestion) ([]event.AskAnswer, error) {
	for _, q := range qs {
		if q.ID == "plan-review" {
			return []event.AskAnswer{{QuestionID: q.ID, Selected: []string{s.label}}}, nil
		}
	}
	return nil, nil
}

// stubPlanGate 独立状态闸（工具层测试用；runner 闸另有直测）。
type stubPlanGate struct{ on bool }

func (g *stubPlanGate) PlanMode() bool { return g.on }
func (g *stubPlanGate) ApprovePlan()   { g.on = false }

func planCtx(g *stubPlanGate, asker Asker) context.Context {
	ctx := WithPlanGate(context.Background(), g)
	return withCallContext(ctx, "call-1", event.Discard, asker)
}

const goodPlan = "# 走查计划\\n\\n第一步：读代码。" // Go 层含字面 \n 转义——JSON 串内合法

func TestExitPlanMode_InactiveErrors(t *testing.T) {
	g := &stubPlanGate{on: false}
	_, err := NewExitPlanModeTool().Execute(planCtx(g, stubAsker{label: "批准"}), []byte(`{"plan":"`+goodPlan+`"}`))
	if err == nil || !strings.Contains(err.Error(), "仅在计划模式可用") {
		t.Fatalf("非计划模式应报「仅在计划模式可用」, got %v", err)
	}
}

func TestExitPlanMode_ApproveFlipsAndNarrates(t *testing.T) {
	g := &stubPlanGate{on: true}
	out, err := NewExitPlanModeTool().Execute(planCtx(g, stubAsker{label: "批准"}), []byte(`{"plan":"`+goodPlan+`"}`))
	if err != nil {
		t.Fatalf("批准路径不应报错: %v", err)
	}
	if !strings.Contains(out, "已退出计划模式") {
		t.Fatalf("批准结果应告知退出, got %q", out)
	}
	if g.on {
		t.Fatal("批准后闸应翻转为 off")
	}
}

func TestExitPlanMode_KeepPlanning(t *testing.T) {
	g := &stubPlanGate{on: true}
	_, err := NewExitPlanModeTool().Execute(planCtx(g, stubAsker{label: "继续计划"}), []byte(`{"plan":"`+goodPlan+`"}`))
	if err == nil || !strings.Contains(err.Error(), "继续计划") {
		t.Fatalf("继续计划应以错误带回反馈语义, got %v", err)
	}
	if !g.on {
		t.Fatal("继续计划后闸应保持 on")
	}
}

func TestExitPlanMode_HeadlessRefuses(t *testing.T) {
	g := &stubPlanGate{on: true}
	// 无 asker（headless）：执行闸必须人来开——拒绝而非自批（与 ask 的
	// Never-Ask 自选降级刻意不同）。
	ctx := WithPlanGate(context.Background(), g)
	ctx = withCallContext(ctx, "call-1", event.Discard, nil)
	_, err := NewExitPlanModeTool().Execute(ctx, []byte(`{"plan":"`+goodPlan+`"}`))
	if err == nil || !strings.Contains(err.Error(), "无交互用户可审批") {
		t.Fatalf("headless 应诚实拒绝, got %v", err)
	}
	if !g.on {
		t.Fatal("headless 拒绝后闸应保持 on")
	}
}

func TestExitPlanMode_RequiresHeading(t *testing.T) {
	g := &stubPlanGate{on: true}
	for _, bad := range []string{`{"plan":""}`, `{"plan":"没有标题的计划"}`, `{"plan":"####### 七级标题"}`} {
		if _, err := NewExitPlanModeTool().Execute(planCtx(g, stubAsker{label: "批准"}), []byte(bad)); err == nil {
			t.Fatalf("计划 %q 应因标题校验被拒", bad)
		}
	}
	// 六级标题合法（任意级标题口径）。
	if _, err := NewExitPlanModeTool().Execute(planCtx(g, stubAsker{label: "批准"}), []byte(`{"plan":"###### 深标题\n正文"}`)); err != nil {
		t.Fatalf("六级标题应合法: %v", err)
	}
}

// TestRunnerPlanGate 真闸直测：翻转/批准叙事落历史（role=user、前缀不动）。
func TestRunnerPlanGate(t *testing.T) {
	r := New(nil, nil, NewSession("系统提示词"), Options{}, event.Discard)
	if r.PlanMode() {
		t.Fatal("初始应为 off")
	}
	r.SetPlanMode(true)
	r.AppendUserMessage(PlanPolicyNotice)
	if !r.PlanMode() {
		t.Fatal("on 后应为 true")
	}
	r.ApprovePlan()
	if r.PlanMode() {
		t.Fatal("ApprovePlan 后应为 off")
	}
	msgs := r.session.Snapshot()
	if msgs[0].Role != "system" || msgs[0].Content != "系统提示词" {
		t.Fatalf("系统前缀不得被改写, got %q", msgs[0].Content)
	}
	if len(msgs) != 3 {
		t.Fatalf("应有 政策+批准叙事 两条注入, got %d", len(msgs)-1)
	}
	if msgs[1].Role != "user" || !strings.Contains(msgs[1].Content, "计划模式") {
		t.Fatalf("第 1 条注入应为计划政策 user 消息, got %q", msgs[1].Content)
	}
	if msgs[2].Role != "user" || !strings.Contains(msgs[2].Content, "已批准计划") {
		t.Fatalf("第 2 条注入应为批准叙事 user 消息, got %q", msgs[2].Content)
	}
}
