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

func TestStructuredRules(t *testing.T) {
	st := newTestKnowledgeStore(t)
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	s.SetKnowledgeStoreForTest(st)
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "soil-remediation-bid", "", "环保工程", proj.ID, []ProposalSection{
		{Title: "第一章", Level: 1, Content: "修复工期为 90 日历天，采用固化稳定化工艺。GB 36600-2018 要求砷限值 60mg/kg。"},
		{Title: "第二章", Level: 1, Content: "修复工期为 90 日历天，采用固化稳定化工艺。GB 36600-2018 要求砷限值 60mg/kg。**加粗**"},
	})
	p.BidSummary = &BidSummary{
		RedLines:  []string{"投标文件未按要求签字盖章作废标处理"},
		DarkRules: []BidItem{{Name: "暗标", Content: "不得出现单位名称、不得加粗"}},
	}
	_ = s.store.SaveProjectFacts(proj.ID, map[string]string{"工期": "90 日历天", "业主单位": "某区生态环境局"})
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}
	items := RunChecks(context.Background(), p, s.structuredRules())
	byRule := map[string][]CheckItem{}
	for _, it := range items {
		byRule[it.Rule] = append(byRule[it.Rule], it)
	}
	if len(byRule["重复率检测"]) == 0 || byRule["重复率检测"][0].Status != "fail" {
		t.Fatalf("重复率规则异常: %+v", byRule["重复率检测"])
	}
	if len(byRule["暗标格式检查"]) == 0 || byRule["暗标格式检查"][0].Status != "fail" {
		t.Fatalf("暗标规则异常: %+v", byRule["暗标格式检查"])
	}
	if len(byRule["废标条款响应"]) == 0 || byRule["废标条款响应"][0].Status != "warn" {
		t.Fatalf("废标响应异常: %+v", byRule["废标条款响应"])
	}
	if len(byRule["规范引用检查"]) == 0 {
		t.Fatalf("规范引用规则未执行")
	}
}

func TestCheckAll_AggregatesRules(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	p, _ := s.Create("方案", "blank", "", "其他")
	_ = s.store.SaveProjectFacts(p.ProjectID, map[string]string{"工期": "90 日历天"})
	_, items, err := s.CheckAll(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("CheckAll: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("CheckAll 无结果")
	}
}
