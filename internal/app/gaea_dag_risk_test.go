package app

// 危险操作分级审批端到端测试（v4.243）：高风险节点执行器前置闸（hold+链停，
// 下游保持待跑不级联跳过）→ 人批准（approve→pending+Approved）→ 续跑放行；
// 高风险节点改图变更后重新批；单跑路径同样受闸。

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/dag"
)

// planRiskChain 三节点链：read(normal) → pivot(high) → report(normal)。
// 与 planChain 同法落盘，取唯一 run id。
func planRiskChain(t *testing.T) string {
	t.Helper()
	tool := dagPlanTool{dir: filepath.Join(".", ".gaea", "work", "dag")}
	args := map[string]any{
		"goal": "出带风险闸的月度报告",
		"nodes": []map[string]any{
			{"id": "read", "title": "读报表", "prompt": "读取三份月度 xlsx"},
			{"id": "pivot", "title": "覆盖汇总", "prompt": "透视汇总出 summary.xlsx", "depends_on": []string{"read"}, "risk": "high"},
			{"id": "report", "title": "出报告", "prompt": "嵌图表出报告 docx", "depends_on": []string{"pivot"}},
		},
	}
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	if _, err := tool.Execute(context.Background(), raw); err != nil {
		t.Fatalf("dag_plan: %v", err)
	}
	runs, err := a0().dagStore().List()
	if err != nil || len(runs) != 1 {
		t.Fatalf("planRiskChain 落盘: err=%v runs=%d", err, len(runs))
	}
	return runs[0].ID
}

// a0 测试用裸 App（dagStore 只吃 cwd，无需 engine 接线）。
func a0() *App { return &App{} }

func TestDagRiskApprovalGate(t *testing.T) {
	injectDagEnv(t)
	id := planRiskChain(t)

	SetDagRunnerForTest(func(ctx context.Context, prompt string, emit func(ref, text string)) (string, string, error) {
		return "ok", "sa_x", nil
	})
	a := &App{}
	if _, err := a.GaeaDagRun(id); err != nil {
		t.Fatalf("GaeaDagRun: %v", err)
	}
	r := waitDagDone(t, id)

	// 高风险节点挂起：hold + 待审批文案；链停在闸上——下游保持待跑不被级联跳过。
	if n := dagNode(t, r, "pivot"); n.Status != dag.StatusHold || !strings.Contains(n.Error, "待审批") {
		t.Fatalf("高风险节点应 hold 待审批: %+v", n)
	}
	if n := dagNode(t, r, "report"); n.Status != dag.StatusPending {
		t.Fatalf("hold 下游应保持待跑（不级联跳过）: %+v", n)
	}
	// hold 不可验收、不可改向续跑（尚无完成运行）。
	if _, err := a.GaeaDagNodeAccept(id, "pivot"); err == nil {
		t.Fatal("hold 节点验收应拒绝")
	}

	// 批准：hold→pending + Approved；续跑放行，链跑完。
	if out, err := a.GaeaDagNodeApprove(id, "pivot"); err != nil || !strings.Contains(out, "已批准") {
		t.Fatalf("approve: %q %v", out, err)
	}
	r, _ = a.dagStore().Get(id)
	if n := dagNode(t, r, "pivot"); n.Status != dag.StatusPending || !n.Approved {
		t.Fatalf("批准后应 pending+Approved: %+v", n)
	}
	if _, err := a.GaeaDagRun(id); err != nil {
		t.Fatalf("批准后续跑: %v", err)
	}
	r = waitDagDone(t, id)
	if n := dagNode(t, r, "pivot"); n.Status != dag.StatusDone {
		t.Fatalf("批准后应可执行: %+v", n)
	}
	if n := dagNode(t, r, "report"); n.Status != dag.StatusDone {
		t.Fatalf("下游应续跑至完成: %+v", n)
	}
	// 已批准的高风险节点单跑（重跑）不再重新挂闸。
	if _, err := a.GaeaDagNodeRun(id, "pivot"); err != nil {
		t.Fatalf("已批准单跑: %v", err)
	}
	waitDagDone(t, id)
	r, _ = a.dagStore().Get(id)
	if n := dagNode(t, r, "pivot"); n.Status != dag.StatusDone {
		t.Fatalf("已批准重跑不应再挂闸: %+v", n)
	}
	// approve 幂等守卫：非 hold 节点拒绝。
	if _, err := a.GaeaDagNodeApprove(id, "pivot"); err == nil || !strings.Contains(err.Error(), "只审批待审批") {
		t.Fatalf("非 hold 节点 approve 应拒绝: %v", err)
	}
}
