package proposal

import (
	"context"
	"testing"
)

var errTestCheck = &ruleError{msg: "boom"}

func TestRunChecks_AllRulesExecuted(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	p, _ := s.Create("方案", "blank", "", "其他")
	rules := []CheckRule{
		ruleFunc{name: "r1", severity: "warning", fn: func(ctx context.Context, p *Proposal) ([]CheckItem, error) {
			return []CheckItem{{Rule: "r1", Severity: "warning", Status: "pass", Message: "ok"}}, nil
		}},
	}
	items := RunChecks(context.Background(), p, rules)
	if len(items) != 1 || items[0].Rule != "r1" || items[0].Status != "pass" {
		t.Fatalf("RunChecks 异常: %+v", items)
	}
}

func TestRunChecks_RuleErrorBecomesItem(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	p, _ := s.Create("方案", "blank", "", "其他")
	rules := []CheckRule{
		ruleFunc{name: "boom", severity: "critical", fn: func(ctx context.Context, p *Proposal) ([]CheckItem, error) {
			return nil, errTestCheck
		}},
	}
	items := RunChecks(context.Background(), p, rules)
	if len(items) != 1 || items[0].Status != "error" || items[0].Message == "" {
		t.Fatalf("错误规则未转 item: %+v", items)
	}
}
