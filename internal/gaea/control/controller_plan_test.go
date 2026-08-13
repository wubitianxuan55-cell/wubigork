package control

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

func TestAskPlanApprovalConfirm(t *testing.T) {
	asks := make(chan event.Ask, 1)
	c := New(Options{Sink: event.FuncSink(func(e event.Event) {
		if e.Kind == event.AskRequest {
			asks <- e.Ask
		}
	})})
	go func() {
		req := <-asks
		if req.ID == "" || len(req.Questions) == 0 || req.Questions[0].Header != "开工计划" {
			t.Errorf("异常 AskRequest: %+v", req)
		}
		c.AnswerQuestion(req.ID, []event.AskAnswer{{QuestionID: "plan", Selected: []string{"确认执行"}}})
	}()
	ok, err := c.askPlanApproval(context.Background(), "计划：\n1. 读取资料\n2. 制表", nil)
	if err != nil || !ok {
		t.Fatalf("askPlanApproval = (%v, %v), want (true, nil)", ok, err)
	}
}

func TestAskPlanApprovalAdjust(t *testing.T) {
	asks := make(chan event.Ask, 1)
	c := New(Options{Sink: event.FuncSink(func(e event.Event) {
		if e.Kind == event.AskRequest {
			asks <- e.Ask
		}
	})})
	go func() {
		req := <-asks
		c.AnswerQuestion(req.ID, []event.AskAnswer{{QuestionID: "plan", Selected: []string{"先调整"}}})
	}()
	ok, err := c.askPlanApproval(context.Background(), "计划", nil)
	if err != nil || ok {
		t.Fatalf("askPlanApproval = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestAskPlanApprovalCarriesStructuredPlan(t *testing.T) {
	asks := make(chan event.Ask, 1)
	c := New(Options{Sink: event.FuncSink(func(e event.Event) {
		if e.Kind == event.AskRequest {
			asks <- e.Ask
		}
	})})
	go func() {
		req := <-asks
		if req.Plan == nil || req.Plan.Goal != "整理成本测算" || len(req.Plan.Steps) != 2 {
			t.Errorf("结构化计划未随 Ask 下发: %+v", req.Plan)
		}
		if req.Questions[0].Prompt == "" {
			t.Error("计划 Markdown 兜底为空")
		}
		c.AnswerQuestion(req.ID, []event.AskAnswer{{QuestionID: "plan", Selected: []string{"确认执行"}}})
	}()
	plan := &event.Plan{
		Goal: "整理成本测算",
		Steps: []event.PlanStep{
			{Title: "读取数据", Resources: []string{"成本.xlsx"}},
			{Title: "生成表格", Tools: []string{"xlsx_edit"}},
		},
	}
	ok, err := c.askPlanApproval(context.Background(), "**任务理解**：整理成本测算", plan)
	if err != nil || !ok {
		t.Fatalf("askPlanApproval = (%v, %v), want (true, nil)", ok, err)
	}
}

func TestPlanGateNilExecutor(t *testing.T) {
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	ok, err := c.planGate(context.Background(), "任务")
	if err != nil || !ok {
		t.Fatalf("planGate(nil executor) = (%v, %v), want (true, nil)", ok, err)
	}
}
