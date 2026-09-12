package dag

// 风险分级与审批字段测试（v4.243 危险操作分级审批）：Validate 风险值校验、
// ApplyEdit 风险变更=形状变更（回 pending+审批归零）、FromRun/Instantiate
// risk 随模板走而 approved 不带。

import (
	"strings"
	"testing"
)

func TestValidateRiskValues(t *testing.T) {
	base := func(risk string) (string, []Node) {
		return "g", []Node{{ID: "a", Title: "A", Prompt: "p", Risk: risk}}
	}
	if err := Validate(base("")); err != nil {
		t.Errorf("空 risk 应合法: %v", err)
	}
	if err := Validate(base(RiskNormal)); err != nil {
		t.Errorf("normal 应合法: %v", err)
	}
	if err := Validate(base(RiskHigh)); err != nil {
		t.Errorf("high 应合法: %v", err)
	}
	if err := Validate(base("critical")); err == nil || !strings.Contains(err.Error(), "risk 非法") {
		t.Errorf("非法 risk 应拒绝: %v", err)
	}
}

func TestApplyEditRiskChangeResetsApproval(t *testing.T) {
	r := Run{
		ID: "r1", Goal: "g",
		Nodes: []Node{{ID: "a", Title: "A", Prompt: "p", Risk: RiskHigh, Status: StatusDone, Approved: true, RunCount: 1}},
	}
	// 同形状（含 risk）：保留状态与审批
	out, _, err := ApplyEdit(r, "g", []Node{{ID: "a", Title: "A", Prompt: "p", Risk: RiskHigh}})
	if err != nil {
		t.Fatalf("ApplyEdit: %v", err)
	}
	if out.Nodes[0].Status != StatusDone || !out.Nodes[0].Approved {
		t.Fatalf("同形状应保留状态与审批: %+v", out.Nodes[0])
	}
	// 仅 risk 变更：视为形状变更 → 回 pending + 审批归零（新风险要重新批）
	out, rep, err := ApplyEdit(r, "g", []Node{{ID: "a", Title: "A", Prompt: "p", Risk: RiskNormal}})
	if err != nil {
		t.Fatalf("ApplyEdit risk 变更: %v", err)
	}
	if out.Nodes[0].Status != StatusPending || out.Nodes[0].Approved {
		t.Fatalf("risk 变更应回 pending+审批归零: %+v", out.Nodes[0])
	}
	if out.Nodes[0].Risk != RiskNormal {
		t.Fatalf("新 risk 应生效: %+v", out.Nodes[0])
	}
	if rep.Kept != 0 || rep.Updated != 1 {
		t.Fatalf("EditReport = %+v, want Updated=1", rep)
	}
}

func TestFromRunInstantiateCarryRisk(t *testing.T) {
	r := Run{
		ID: "r1", Goal: "g",
		Nodes: []Node{{ID: "a", Title: "A", Prompt: "p", Risk: RiskHigh, Status: StatusDone, Approved: true, Ref: "sa_a", RunCount: 2}},
	}
	tpl, err := FromRun(r, "模板")
	if err != nil {
		t.Fatalf("FromRun: %v", err)
	}
	if tpl.Nodes[0].Risk != RiskHigh {
		t.Fatalf("risk 是图形状应随模板: %+v", tpl.Nodes[0])
	}
	if tpl.Nodes[0].Approved {
		t.Fatalf("approved 是运行痕迹不应随模板: %+v", tpl.Nodes[0])
	}
	run := Instantiate(tpl)
	if run.Nodes[0].Risk != RiskHigh || run.Nodes[0].Approved {
		t.Fatalf("Instantiate 应带 risk 不带 approved: %+v", run.Nodes[0])
	}
}
