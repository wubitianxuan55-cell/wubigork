package app

// 6.3 文件流水线 App 层测试（设计 docs/gaea-office-dag-63-design-2026-09.md §5）：
// fake runner 锁 3 节点链全生命周期——起跑→产物归因（证据卡窗口增量）→重跑
// 降验收→验收回流记忆；Cancel 级联 skipped；上游失败下游跳过；runner 未接线
// fail-closed；重启中断懒清扫。全程临时工作区 + 隔离记忆库，不触碰真实用户
// 数据（injectWorkSpace / SetOfficeStoreForTest 先例）。

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/dag"
	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// injectDagEnv 隔离三件套：t.Chdir 临时工作区（gaeaCwd 回退 os.Getwd，空 cfg
// = injectWorkSpace 先例）+ 隔离记忆库 + 清空 nodeRunner/cancels。返回证据卡
// 目录（fake runner 往里落卡模拟节点产物）。
func injectDagEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)
	origCfg := ga.cfg
	ga.cfg = &config.Config{}
	t.Cleanup(func() { ga.cfg = origCfg })

	ga.followUpMu.Lock()
	origFollowUp, origNode := ga.followUp, ga.nodeRunner
	ga.followUp, ga.nodeRunner = nil, nil
	ga.followUpMu.Unlock()
	t.Cleanup(func() {
		ga.followUpMu.Lock()
		ga.followUp, ga.nodeRunner = origFollowUp, origNode
		ga.followUpMu.Unlock()
	})

	SetOfficeStoreForTest(memory.Store{Dir: t.TempDir(), GlobalDir: t.TempDir()})
	t.Cleanup(ResetOfficeStoreForTest)

	journalDir := filepath.Join(tmp, ".gaea", "work", "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return journalDir
}

// planChain 用 dag_plan 工具落一条 3 节点链（读报表→透视→出报告），返回 run id。
func planChain(t *testing.T) string {
	t.Helper()
	tool := dagPlanTool{dir: filepath.Join(".", ".gaea", "work", "dag")}
	args := map[string]any{
		"goal": "出月度报告",
		"nodes": []map[string]any{
			{"id": "read", "title": "读报表", "prompt": "读取三份月度 xlsx"},
			{"id": "pivot", "title": "透视汇总", "prompt": "透视汇总出 summary.xlsx", "depends_on": []string{"read"}},
			{"id": "report", "title": "出报告", "prompt": "嵌图表出报告 docx 并导 PDF", "depends_on": []string{"pivot"}},
		},
	}
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	out, err := tool.Execute(context.Background(), b)
	if err != nil {
		t.Fatalf("dag_plan: %v", err)
	}
	runs, err := (&App{}).dagStore().List()
	if err != nil || len(runs) != 1 {
		t.Fatalf("planChain 落盘: err=%v runs=%d", err, len(runs))
	}
	if runs[0].Goal != "出月度报告" || len(runs[0].Nodes) != 3 {
		t.Fatalf("planChain 内容不符: %+v", runs[0])
	}
	if out == "" {
		t.Fatal("dag_plan 应返回受理文案")
	}
	return runs[0].ID
}

// appendCard fake runner 的产物模拟：往 Journal 落一张证据卡（Append 自动补
// ID/At），Target=工作区相对路径。
func appendCard(t *testing.T, dir, session, target string) {
	t.Helper()
	st, err := evidence.OpenJournal(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Append(evidence.ChangeRecord{
		SessionID: session, Space: "work", Tool: "write_file", Target: target,
	}); err != nil {
		t.Fatal(err)
	}
}

func waitDerived(t *testing.T, id, want string) dag.Run {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r, err := (&App{}).dagStore().Get(id)
		if err != nil {
			t.Fatalf("Get %s: %v", id, err)
		}
		if dag.Derived(r) == want {
			return r
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待派生态 %s 超时", want)
	return dag.Run{}
}

// waitDagDone 等执行器 goroutine 收尾（dagCancels 在途登记消失，defer 删除时
// 级联 marking 已完成），返回最终 run——终态断言用它，避免读到中间态。
func waitDagDone(t *testing.T, id string) dag.Run {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, active := ga.dagCancels.Load(id); !active {
			r, err := (&App{}).dagStore().Get(id)
			if err != nil {
				t.Fatalf("Get %s: %v", id, err)
			}
			return r
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("等待执行器收尾超时")
	return dag.Run{}
}

func dagNode(t *testing.T, r dag.Run, id string) dag.Node {
	t.Helper()
	if idx := dagNodeIndex(r, id); idx >= 0 {
		return r.Nodes[idx]
	}
	t.Fatalf("节点 %s 不存在", id)
	return dag.Node{}
}

// TestDagLifecycle 全生命周期：3 节点链起跑→各节点产物归因正确→验收降级重跑
// →验收回流记忆。
func TestDagLifecycle(t *testing.T) {
	journalDir := injectDagEnv(t)
	id := planChain(t)

	var calls []string
	SetDagRunnerForTest(func(ctx context.Context, prompt string, emit func(ref, text string)) (string, string, error) {
		var node string
		switch {
		case strings.Contains(prompt, "读取三份月度 xlsx"):
			node = "read"
		case strings.Contains(prompt, "透视汇总出 summary.xlsx"):
			node = "pivot"
		case strings.Contains(prompt, "嵌图表出报告 docx"):
			node = "report"
		}
		calls = append(calls, node)
		// 每节点落一张证据卡=其产物；上游产物对下游可见（prompt 携带上游清单）。
		appendCard(t, journalDir, "sess-"+node, "docs/"+node+".xlsx")
		if node == "report" && !strings.Contains(prompt, "docs/pivot.xlsx") {
			return "", "sa_report", errors.New("report 节点 prompt 未携带上游产物")
		}
		return "ok: " + node, "sa_" + node, nil
	})

	a := &App{}
	if _, err := a.GaeaDagRun(id); err != nil {
		t.Fatalf("GaeaDagRun: %v", err)
	}
	r := waitDagDone(t, id)
	if len(calls) != 3 || calls[0] != "read" || calls[1] != "pivot" || calls[2] != "report" {
		t.Fatalf("执行顺序错误: %v", calls)
	}
	if n := dagNode(t, r, "read"); n.Status != dag.StatusDone || len(n.Outputs) != 1 || n.Outputs[0] != "docs/read.xlsx" {
		t.Fatalf("read 节点产物归因错误: %+v", n)
	}
	if n := dagNode(t, r, "report"); n.Ref != "sa_report" {
		t.Fatalf("report 节点 ref 未回写: %+v", n)
	}

	// 验收：节点 accepted + 产物回流记忆。
	if out, err := a.GaeaDagNodeAccept(id, "read"); err != nil {
		t.Fatalf("GaeaDagNodeAccept: %v", err)
	} else if !strings.Contains(out, "已验收") {
		t.Fatalf("验收文案: %q", out)
	}
	r, _ = (&App{}).dagStore().Get(id)
	if n := dagNode(t, r, "read"); n.Status != dag.StatusAccepted || n.AcceptedAt == "" {
		t.Fatalf("验收状态错误: %+v", n)
	}
	store := officeStoreOverride
	if m, ok := store.Get("read"); !ok || m.Title != "读报表" {
		t.Fatalf("记忆回流缺失: %+v ok=%v", m, ok)
	}

	// 重跑已验收节点：验收失效（退回 done）→ 重新执行 → 重新验收。
	if _, err := a.GaeaDagNodeRun(id, "read"); err != nil {
		t.Fatalf("重跑已验收节点: %v", err)
	}
	waitDagDone(t, id)
	r, _ = (&App{}).dagStore().Get(id)
	if n := dagNode(t, r, "read"); n.Status != dag.StatusDone || n.RunCount != 2 {
		t.Fatalf("重跑后状态错误: %+v", n)
	}
}

// TestDagUpstreamFailureSkipsDownstream 上游失败→下游 skipped，不空跑。
func TestDagUpstreamFailureSkipsDownstream(t *testing.T) {
	injectDagEnv(t)
	id := planChain(t)

	SetDagRunnerForTest(func(ctx context.Context, prompt string, emit func(ref, text string)) (string, string, error) {
		if strings.Contains(prompt, "读取三份月度 xlsx") {
			return "", "", errors.New("报表文件不存在")
		}
		t.Fatal("上游失败后不应继续执行下游节点")
		return "", "", nil
	})

	a := &App{}
	if _, err := a.GaeaDagRun(id); err != nil {
		t.Fatalf("GaeaDagRun: %v", err)
	}
	r := waitDagDone(t, id)
	if n := dagNode(t, r, "read"); n.Status != dag.StatusFailed || n.Error == "" {
		t.Fatalf("上游失败状态错误: %+v", n)
	}
	for _, want := range []string{"pivot", "report"} {
		if n := dagNode(t, r, want); n.Status != dag.StatusSkipped {
			t.Fatalf("下游 %s 应 skipped: %+v", want, n)
		}
	}
}

// TestDagCancelCascade 终止级联：在跑节点随 ctx 取消，未起跑节点 skipped。
func TestDagCancelCascade(t *testing.T) {
	injectDagEnv(t)
	id := planChain(t)

	release := make(chan struct{})
	SetDagRunnerForTest(func(ctx context.Context, prompt string, emit func(ref, text string)) (string, string, error) {
		if strings.Contains(prompt, "读取三份月度 xlsx") {
			<-ctx.Done() // 挂住直到取消
			return "", "", ctx.Err()
		}
		<-release
		return "ok", "sa_x", nil
	})
	t.Cleanup(func() { close(release) })

	a := &App{}
	if _, err := a.GaeaDagRun(id); err != nil {
		t.Fatalf("GaeaDagRun: %v", err)
	}
	// 等 read 真正 in-flight（running）再取消——取消落在起跑前会按 skipped
	// 语义处理（节点从未跑过），那也是对的，但不是本用例要锁的行为。
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if r, err := (&App{}).dagStore().Get(id); err == nil && dagNode(t, r, "read").RunCount > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := a.GaeaDagCancel(id); err != nil {
		t.Fatalf("GaeaDagCancel: %v", err)
	}
	r := waitDagDone(t, id)
	if n := dagNode(t, r, "read"); n.Status != dag.StatusFailed {
		t.Fatalf("被取消的在跑节点应 failed: %+v", n)
	}
	for _, want := range []string{"pivot", "report"} {
		if n := dagNode(t, r, want); n.Status != dag.StatusSkipped {
			t.Fatalf("级联 %s 应 skipped: %+v", want, n)
		}
	}
}

// TestDagFailClosedNoRunner runner 未接线（引擎未构建）→ 全链显式失败不静默。
func TestDagFailClosedNoRunner(t *testing.T) {
	injectDagEnv(t)
	id := planChain(t)

	a := &App{}
	out, err := a.GaeaDagRun(id)
	if err != nil {
		t.Fatalf("受理不应报错: %v", err)
	}
	if out == "" {
		t.Fatal("受理文案为空")
	}
	r := waitDagDone(t, id)
	for _, n := range r.Nodes {
		if n.Status != dag.StatusFailed || !strings.Contains(n.Error, "执行器未接线") {
			t.Fatalf("fail-closed 状态错误: %+v", n)
		}
	}
}

// TestDagSweepInterrupted 应用重启懒清扫：running 无在途执行→诚实置 failed。
func TestDagSweepInterrupted(t *testing.T) {
	injectDagEnv(t)
	id := planChain(t)
	store := (&App{}).dagStore()

	r, _ := store.Get(id)
	r.Nodes[1].Status = dag.StatusRunning
	if err := store.Save(r); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	view, err := a.GaeaDagGet(id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Derived != dag.DerivedFailed {
		t.Fatalf("清扫后应 failed，得 %s", view.Derived)
	}
	if n := dagNode(t, view.Run, "pivot"); !strings.Contains(n.Error, "中断") {
		t.Fatalf("中断原因未注明: %+v", n)
	}
}

// TestDagPlanValidation dag_plan 对环/悬空依赖 fail-closed。
func TestDagPlanValidation(t *testing.T) {
	injectDagEnv(t)
	tool := dagPlanTool{dir: filepath.Join(".", ".gaea", "work", "dag")}
	cases := []struct {
		name string
		args string
	}{
		{"成环", `{"goal":"g","nodes":[
			{"id":"a","title":"t","prompt":"p","depends_on":["b"]},
			{"id":"b","title":"t","prompt":"p","depends_on":["a"]}]}`},
		{"悬空依赖", `{"goal":"g","nodes":[
			{"id":"a","title":"t","prompt":"p","depends_on":["ghost"]}]}`},
		{"空链", `{"goal":"g","nodes":[]}`},
	}
	for _, tc := range cases {
		if _, err := tool.Execute(context.Background(), json.RawMessage(tc.args)); err == nil {
			t.Errorf("%s: 应拒绝", tc.name)
		}
	}
}

// TestDagSteerGuard steer 守卫：无可改向运行/空指令/执行器未接线均显式报错。
func TestDagSteerGuard(t *testing.T) {
	injectDagEnv(t)
	id := planChain(t)
	a := &App{}
	if _, err := a.GaeaDagNodeSteer(id, "read", "改成季度口径"); err == nil {
		t.Fatal("未跑过的节点不应可改向")
	}
	SetDagRunnerForTest(func(ctx context.Context, prompt string, emit func(ref, text string)) (string, string, error) {
		appendCard(t, filepath.Join(".", ".gaea", "work", "journal"), "s", "docs/read.xlsx")
		return "ok", "sa_read", nil
	})
	if _, err := a.GaeaDagNodeRun(id, "read"); err != nil {
		t.Fatal(err)
	}
	waitDagDone(t, id)
	if _, err := a.GaeaDagNodeSteer(id, "read", "  "); err == nil {
		t.Fatal("空指令应拒绝")
	}
	// ga.followUp 未接线（injectDagEnv 清空）→ fail-closed。
	if _, err := a.GaeaDagNodeSteer(id, "read", "追加同比列"); err == nil {
		t.Fatal("改向执行器未接线应报错")
	}
}

// TestDagTemplateFlow 模板库全流程（6.3 余项）：存模板（剥运行痕迹/空名回退
// goal）→ 列表 → 一键重建（全新草稿不自动起跑）→ 重建链可跑通；守卫=存不存在
// 的 run/重建不存在的模板/删除后显式报错。
func TestDagTemplateFlow(t *testing.T) {
	injectDagEnv(t)
	id := planChain(t)
	a := &App{}

	// 先把链跑出一轮产物/验收痕迹，验证存模板剥得净。
	SetDagRunnerForTest(func(ctx context.Context, prompt string, emit func(ref, text string)) (string, string, error) {
		return "ok", "sa_x", nil
	})
	if _, err := a.GaeaDagRun(id); err != nil {
		t.Fatalf("GaeaDagRun: %v", err)
	}
	waitDagDone(t, id)

	// 守卫：run 不在场报错。
	if _, err := a.GaeaDagTemplateSave("dag_ghost", ""); err == nil {
		t.Fatal("存不存在的 run 应报错")
	}

	// 空名存 → 回退 goal；运行痕迹剥净。
	if out, err := a.GaeaDagTemplateSave(id, ""); err != nil {
		t.Fatalf("GaeaDagTemplateSave: %v", err)
	} else if !strings.Contains(out, "已存为模板") {
		t.Fatalf("存模板文案: %q", out)
	}
	tpls, err := a.GaeaDagTemplateList()
	if err != nil || len(tpls) != 1 {
		t.Fatalf("TemplateList: err=%v n=%d", err, len(tpls))
	}
	tpl := tpls[0]
	if tpl.Name != "出月度报告" || tpl.SourceRunID != id || len(tpl.Nodes) != 3 {
		t.Fatalf("模板内容错误: %+v", tpl)
	}
	for _, n := range tpl.Nodes {
		if n.Status != dag.StatusPending || n.Ref != "" || n.RunCount != 0 || len(n.Outputs) != 0 {
			t.Fatalf("模板节点未剥运行痕迹: %+v", n)
		}
	}

	// 一键重建：全新草稿 run（不自动起跑），图形状与依赖保持。
	if _, err := a.GaeaDagTemplateNew("dagtpl_ghost"); err == nil {
		t.Fatal("重建不存在的模板应报错")
	}
	out, err := a.GaeaDagTemplateNew(tpl.ID)
	if err != nil {
		t.Fatalf("GaeaDagTemplateNew: %v", err)
	}
	if !strings.Contains(out, "未起跑") {
		t.Fatalf("重建文案应注明未起跑: %q", out)
	}
	runs, err := a.dagStore().List()
	if err != nil || len(runs) != 2 {
		t.Fatalf("重建后应有 2 条 run: err=%v n=%d", err, len(runs))
	}
	var fresh dag.Run
	for _, r := range runs {
		if r.ID != id {
			fresh = r
		}
	}
	if fresh.ID == "" || dag.Derived(fresh) != dag.DerivedDraft || len(fresh.Nodes) != 3 {
		t.Fatalf("重建 run 应为全新草稿: %+v", fresh)
	}
	if n := dagNode(t, fresh, "report"); len(n.DependsOn) != 1 || n.DependsOn[0] != "pivot" {
		t.Fatalf("重建依赖形状丢失: %+v", n)
	}

	// 重建链可跑通：fake runner 按指令路由，全链 done。
	SetDagRunnerForTest(func(ctx context.Context, prompt string, emit func(ref, text string)) (string, string, error) {
		switch {
		case strings.Contains(prompt, "读取三份月度 xlsx"):
			return "ok", "sa_read", nil
		case strings.Contains(prompt, "透视汇总出 summary.xlsx"):
			return "ok", "sa_pivot", nil
		case strings.Contains(prompt, "嵌图表出报告 docx"):
			return "ok", "sa_report", nil
		}
		return "", "", errors.New("未知节点指令")
	})
	if _, err := a.GaeaDagRun(fresh.ID); err != nil {
		t.Fatalf("重建链起跑: %v", err)
	}
	r := waitDagDone(t, fresh.ID)
	for _, n := range r.Nodes {
		if n.Status != dag.StatusDone {
			t.Fatalf("重建链应全 done: %+v", n)
		}
	}

	// 删除闭环；再删/再用显式报错。
	if _, err := a.GaeaDagTemplateDelete(tpl.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if tpls, _ := a.GaeaDagTemplateList(); len(tpls) != 0 {
		t.Fatalf("删除后应为空: %d", len(tpls))
	}
	if _, err := a.GaeaDagTemplateDelete(tpl.ID); err == nil {
		t.Fatal("删除不存在模板应报错")
	}
}
