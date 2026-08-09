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
	ok, err := c.askPlanApproval(context.Background(), "计划：\n1. 读取资料\n2. 制表")
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
	ok, err := c.askPlanApproval(context.Background(), "计划")
	if err != nil || ok {
		t.Fatalf("askPlanApproval = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestPlanGateNilExecutor(t *testing.T) {
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	ok, err := c.planGate(context.Background(), "任务")
	if err != nil || !ok {
		t.Fatalf("planGate(nil executor) = (%v, %v), want (true, nil)", ok, err)
	}
}
