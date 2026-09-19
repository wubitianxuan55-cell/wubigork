package app

// 办公多文件流水线（阶段六 6.3，设计 docs/gaea-office-dag-63-design-2026-09.md）：
// 规划（dag_plan 工具）→ 存储（<cwd>/.gaea/work/dag/<id>.json）→ 执行（波次推进，
// 每节点=一次 TaskTool.RunNew 全新子代理会话，v4.221 起波内并行）→ 控制（绑定面）→
// 验收回流记忆（hubOfficeStore.Save，人拍板即写入确认面）。
// 关键口径见设计 §3：产物=按会话归因（SessionID=sa_ ref；ref 空回退窗口增量）、
// 终止级联、fail-closed。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/dag"
	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// dagPlanTool 让主代理把委托式目标规划成可执行的任务图（节点=子代理可完成的
// 文件加工步骤）。PersistWrite 标记：子代理注册表自动剔除（FilterRegistry），
// 流水线不能嵌套派生流水线。
type dagPlanTool struct{ dir string }

func (d dagPlanTool) Name() string       { return "dag_plan" }
func (d dagPlanTool) ReadOnly() bool     { return false }
func (d dagPlanTool) SpaceTag() string   { return spaces.SpaceWork }
func (d dagPlanTool) PersistWrite() bool { return true }

func (d dagPlanTool) Description() string {
	return "把多文件跨格式任务（如「读 N 份报表→透视→嵌图表→出 Word→导 PDF」）规划成一条可执行的文件流水线：逐节点写清委托指令与依赖，落盘后用户在 任务中心→文件流水线 起跑、逐节点验收。只做规划不执行。带 run_id 时为增量改图：对既有流水线原地调和（形状未变的节点保留状态与产物，变更/新增节点回待跑，撤销已验收节点会如实注明），不必整链重建。单文件小改动不要用。"
}

func (d dagPlanTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "goal":{"type":"string","description":"整条流水线的最终目标与验收口径（一句话）"},
  "run_id":{"type":"string","description":"可选。传入既有流水线 id 时为增量改图模式（运行中的流水线会被拒绝）；省略时新建流水线"},
  "nodes":{"type":"array","minItems":2,"items":{"type":"object","properties":{
    "id":{"type":"string","description":"节点短 id（ASCII kebab，如 read-reports）"},
    "title":{"type":"string","description":"节点名（3-8 字，验收后将作为记忆条目名）"},
    "prompt":{"type":"string","description":"该节点的完整委托指令：输入文件（工作区相对路径）、加工动作、期望产物路径。子代理看不到对话，prompt 必须自包含"},
    "depends_on":{"type":"array","items":{"type":"string"},"description":"依赖的节点 id 列表（可空=无依赖）"},
    "risk":{"type":"string","enum":["normal","high"],"description":"风险分级：覆盖/删除工作区既有文件、批量移动、全局性改动的节点标 high（起跑前会暂停等用户审批）；常规新增/编辑用 normal（缺省）"}}},
    "required":["id","title","prompt"]}}
},
"required":["goal","nodes"]
}`)
}

func (d dagPlanTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Goal  string `json:"goal"`
		RunID string `json:"run_id"`
		Nodes []struct {
			ID        string   `json:"id"`
			Title     string   `json:"title"`
			Prompt    string   `json:"prompt"`
			DependsOn []string `json:"depends_on"`
			Risk      string   `json:"risk"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	nodes := make([]dag.Node, 0, len(p.Nodes))
	for _, n := range p.Nodes {
		nodes = append(nodes, dag.Node{
			ID:        strings.TrimSpace(n.ID),
			Title:     strings.TrimSpace(n.Title),
			Prompt:    strings.TrimSpace(n.Prompt),
			DependsOn: n.DependsOn,
			Risk:      n.Risk,
			Status:    dag.StatusPending,
		})
	}
	goal := strings.TrimSpace(p.Goal)

	// 增量改图模式（run_id 在场）：原地调和，不换 run id，不重排既有验收。
	if runID := strings.TrimSpace(p.RunID); runID != "" {
		store := dag.NewStore(d.dir)
		r, err := store.Get(runID)
		if err != nil {
			return "", err
		}
		r, rep, err := dag.ApplyEdit(r, goal, nodes)
		if err != nil {
			return "", err
		}
		if err := store.Save(r); err != nil {
			return "", fmt.Errorf("流水线改图落盘: %w", err)
		}
		slog.Info("文件流水线已改图", "id", r.ID, "kept", rep.Kept, "updated", rep.Updated, "removed", rep.Removed, "revoked", rep.RevokedAcc)
		msg := fmt.Sprintf("已更新流水线 %s：保留 %d 节点、改/增 %d 节点、移除 %d 节点。", r.ID, rep.Kept, rep.Updated, rep.Removed)
		if rep.RevokedAcc > 0 {
			msg += fmt.Sprintf("注意：%d 个已验收节点因指令变更撤销验收，重跑后需重新验收。", rep.RevokedAcc)
		}
		return msg, nil
	}

	if err := dag.Validate(goal, nodes); err != nil {
		return "", err
	}
	run := dag.Run{
		ID:        dag.NewID(),
		Goal:      goal,
		CreatedAt: time.Now().Format(time.RFC3339),
		Nodes:     nodes,
	}
	if err := dag.NewStore(d.dir).SaveNew(&run); err != nil {
		return "", fmt.Errorf("流水线落盘: %w", err)
	}
	slog.Info("文件流水线已规划", "id", run.ID, "nodes", len(run.Nodes))
	return fmt.Sprintf("已规划文件流水线 %s：%d 个节点。用户可在 任务中心→文件流水线 中查看并起跑；也可以直接说「跑这条流水线」。", run.ID, len(run.Nodes)), nil
}

// ── 绑定面 ────────────────────────────────────────────────────────────

// dagRunner 取全新子代理执行器（boot OnNodeRunnerReady 注入；未构建返回 nil，
// 执行器侧 fail-closed）。
func gaDagRunner() gaeaAgent.SubagentRunRunner {
	ga.followUpMu.Lock()
	defer ga.followUpMu.Unlock()
	return ga.nodeRunner
}

// SetDagRunnerForTest 注入假执行器（测试用，fake runner 锁全生命周期行为）。
func SetDagRunnerForTest(fn gaeaAgent.SubagentRunRunner) {
	ga.followUpMu.Lock()
	ga.nodeRunner = fn
	ga.followUpMu.Unlock()
}

func (a *App) dagStore() *dag.Store {
	return dag.NewStore(filepath.Join(gaeaCwd(), ".gaea", "work", "dag"))
}

// dagSweep 懒清扫：running 节点但无在途执行（应用重启/进程退出）→ 诚实置
// failed。返回是否发生变更（调用方负责回存）。
func dagSweep(r *dag.Run) bool {
	if _, active := ga.dagCancels.Load(r.ID); active {
		return false
	}
	return dag.MarkInterrupted(r, "应用重启中断，重跑续接")
}

// GaeaDagList 全部流水线（创建时间倒序，带派生态）。
func (a *App) GaeaDagList() ([]dag.RunView, error) {
	runs, err := a.dagStore().List()
	if err != nil {
		return nil, err
	}
	out := make([]dag.RunView, 0, len(runs))
	for _, r := range runs {
		if dagSweep(&r) {
			// sweep 幂等（下次 List/Get 会重扫重存），失败 warn 留痕即可。
			if err := a.dagStore().Save(r); err != nil {
				slog.Warn("dag 懒清扫结果落盘失败", "id", r.ID, "error", err)
			}
		}
		out = append(out, dag.View(r))
	}
	return out, nil
}

// GaeaDagGet 单条流水线（含节点明细）。
func (a *App) GaeaDagGet(id string) (*dag.RunView, error) {
	r, err := a.dagStore().Get(id)
	if err != nil {
		return nil, err
	}
	if dagSweep(&r) {
		if err := a.dagStore().Save(r); err != nil {
			slog.Warn("dag 懒清扫结果落盘失败", "id", r.ID, "error", err)
		}
	}
	v := dag.View(r)
	return &v, nil
}

// GaeaDagRun 起跑/续跑整链：所有未完成节点（done/accepted 不重跑）按依赖
// 分波推进——波序本身保证上游先跑，无需起跑前依赖已满足；上游失败由执行器
// 级联 skipped。
func (a *App) GaeaDagRun(id string) (string, error) {
	r, err := a.dagStore().Get(id)
	if err != nil {
		return "", err
	}
	if d := dag.Derived(r); d == dag.DerivedRunning {
		return "", fmt.Errorf("流水线已在运行中")
	}
	selected := map[string]bool{}
	for _, n := range r.Nodes {
		if n.Status == dag.StatusDone || n.Status == dag.StatusAccepted {
			continue
		}
		selected[n.ID] = true
	}
	if len(selected) == 0 {
		return "", fmt.Errorf("没有可跑节点（全部完成；重跑请用单节点重跑）")
	}
	waves, err := dag.Order(r.Nodes, selected)
	if err != nil {
		return "", err
	}
	a.dagStart(id, waves)
	return fmt.Sprintf("流水线 %s 已受理：%d 个节点待跑。", id, len(selected)), nil
}

// GaeaDagNodeRun 单节点跑/重跑（accepted 重跑先失效验收）。
func (a *App) GaeaDagNodeRun(id, nodeID string) (string, error) {
	r, err := a.dagStore().Get(id)
	if err != nil {
		return "", err
	}
	if d := dag.Derived(r); d == dag.DerivedRunning {
		return "", fmt.Errorf("流水线运行中，先终止再单跑")
	}
	idx := dagNodeIndex(r, nodeID)
	if idx < 0 {
		return "", fmt.Errorf("节点不存在: %s", nodeID)
	}
	if !dagDepsSatisfied(r, r.Nodes[idx]) {
		return "", fmt.Errorf("上游依赖未完成，先跑依赖节点")
	}
	waves, err := dag.Order(r.Nodes, map[string]bool{nodeID: true})
	if err != nil {
		return "", err
	}
	a.dagStart(id, waves)
	return fmt.Sprintf("节点 %s 已受理单跑。", nodeID), nil
}

// GaeaDagNodeSteer 对已完成/失败节点续跑改向（复用追问管道：PrepareContinue
// 拒绝在跑运行，followUpClaims 每 ref 单飞）。节点状态不动——改向产物以子代理
// transcript 为准，验收仍锁定在既有产物上，需重跑后重新验收。
func (a *App) GaeaDagNodeSteer(id, nodeID, prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("改向指令不能为空")
	}
	r, err := a.dagStore().Get(id)
	if err != nil {
		return "", err
	}
	idx := dagNodeIndex(r, nodeID)
	if idx < 0 {
		return "", fmt.Errorf("节点不存在: %s", nodeID)
	}
	node := r.Nodes[idx]
	// 运行中直穿（v4.243）：凭 ref 找到在跑子代理 runner，注入 steer 队列
	//（不打断工具执行，下一回合生效）——与主对话 GaeaSteer 同机制。节点状态
	// 不动；查无在跑登记（恰好收跑/ephemeral）如实报错，不静默转续跑。
	if node.Status == dag.StatusRunning {
		if node.Ref == "" {
			return "", fmt.Errorf("节点在跑但暂无会话引用，稍后再试或收跑后用续跑改向")
		}
		if err := gaeaAgent.SteerSubagent(node.Ref, prompt); err != nil {
			return "", err
		}
		slog.Info("流水线节点改向直穿", "run", id, "node", nodeID, "ref", node.Ref)
		return "改向指令已直穿在跑节点（不打断执行，下一回合生效）。", nil
	}
	if node.Ref == "" || (node.Status != dag.StatusDone && node.Status != dag.StatusFailed) {
		return "", fmt.Errorf("节点 %s 尚无可改向的完成运行（先跑一次）", nodeID)
	}
	ga.followUpMu.Lock()
	runner := ga.followUp
	ga.followUpMu.Unlock()
	if runner == nil {
		return "", fmt.Errorf("改向执行器未接线（引擎尚未构建完成）")
	}
	if _, loaded := followUpClaims.LoadOrStore(node.Ref, true); loaded {
		return "", fmt.Errorf("该节点已有一次改向正在运行")
	}
	slog.Info("流水线节点改向受理", "run", id, "node", nodeID, "ref", node.Ref)
	go func() {
		defer followUpClaims.Delete(node.Ref)
		// 改向执行与 GaeaSubagentFollowUp 后台 goroutine 逐行同构，防线同款
		//（子代理运行链 panic 不带崩进程）。
		defer func() {
			if r := recover(); r != nil {
				slog.Error("流水线节点改向 panic", "ref", node.Ref, "panic", r)
			}
		}()
		ctx := gaeaAgent.WithSpace(context.Background(), gaeaSessionSpace())
		if err := runner(ctx, node.Ref, prompt); err != nil {
			slog.Warn("流水线节点改向失败", "ref", node.Ref, "error", err)
		}
	}()
	return fmt.Sprintf("节点 %s 改向已受理，结果在子代理会话内。", nodeID), nil
}

// GaeaDagNodeAccept 验收（人拍板）：节点置 accepted 并把产物回流记忆
// （save 路径自动落 memory_events→5.1 图谱实体，6.2 项目本体自然吃到）。
// 同名记忆（同节点 id）被后续验收覆盖=最新产物语义，诚实不堆积。
func (a *App) GaeaDagNodeAccept(id, nodeID string) (string, error) {
	ga.dagMu.Lock()
	r, err := a.dagStore().Get(id)
	if err != nil {
		ga.dagMu.Unlock()
		return "", err
	}
	idx := dagNodeIndex(r, nodeID)
	if idx < 0 {
		ga.dagMu.Unlock()
		return "", fmt.Errorf("节点不存在: %s", nodeID)
	}
	if r.Nodes[idx].Status != dag.StatusDone {
		ga.dagMu.Unlock()
		return "", fmt.Errorf("只验收已完成的节点（当前 %s）", r.Nodes[idx].Status)
	}
	r.Nodes[idx].Status = dag.StatusAccepted
	r.Nodes[idx].AcceptedAt = time.Now().Format(time.RFC3339)
	if err := a.dagStore().Save(r); err != nil {
		ga.dagMu.Unlock()
		return "", err
	}
	ga.dagMu.Unlock()

	node := r.Nodes[idx]
	if memErr := dagAcceptMemoryWrite(a, r, node); memErr != nil {
		// 验收动作已成立；记忆回写失败如实上报，不静默不回滚验收。
		return fmt.Sprintf("节点 %s 已验收，但记忆回写失败: %v", nodeID, memErr), nil
	}
	slog.Info("流水线节点已验收", "run", id, "node", nodeID, "outputs", node.Outputs)
	return fmt.Sprintf("节点 %s 已验收，产物已回流记忆。", nodeID), nil
}

// dagAcceptMemoryWrite 是节点验收的记忆回写核心（单验收/一键验收共用单一真源）：
// 产物清单落「流水线交付」记忆（Tags dag/交付物），由 Save 路径自动落
// memory_events。回写失败由调用方如实上报——验收动作已成立，不回滚。
func dagAcceptMemoryWrite(a *App, r dag.Run, node dag.Node) error {
	store := a.hubOfficeStore()
	outputs := strings.Join(node.Outputs, "、")
	if outputs == "" {
		outputs = "（无文件产物）"
	}
	_, err := store.Save(memory.Memory{
		Name:        node.ID,
		Title:       node.Title,
		Description: fmt.Sprintf("流水线「%s」节点「%s」交付：%s", r.Goal, node.Title, outputs),
		Type:        memory.TypeProject,
		Kind:        memory.KindSemantic,
		Tags:        []string{"dag", "交付物"},
		Body:        fmt.Sprintf("## 流水线交付\n\n- 目标：%s\n- 节点：%s（%s）\n- 产物：%s\n- 验收时间：%s", r.Goal, node.Title, node.ID, outputs, node.AcceptedAt),
	})
	return err
}

// GaeaDagAcceptAll 一键验收（成品直出首刀，DeliverableRegistry 用户面）：
// 把 run 内全部 done 节点一次置 accepted（锁内一次翻转+save 一次），锁外逐节点
// 记忆回写（单条失败不阻断其余，汇总如实上报——验收已成立不回滚，与单验收同
// 哲学）。验收语义零变更：仍人拍板（前端两段式确认=一次拍板覆盖清单所列节点），
// 不自动验收不定时验收。无可验收节点返回明确信息非错误。
func (a *App) GaeaDagAcceptAll(id string) (string, error) {
	ga.dagMu.Lock()
	r, err := a.dagStore().Get(id)
	if err != nil {
		ga.dagMu.Unlock()
		return "", err
	}
	now := time.Now().Format(time.RFC3339)
	var flipped []dag.Node
	for i := range r.Nodes {
		if r.Nodes[i].Status == dag.StatusDone {
			r.Nodes[i].Status = dag.StatusAccepted
			r.Nodes[i].AcceptedAt = now
			flipped = append(flipped, r.Nodes[i])
		}
	}
	if len(flipped) == 0 {
		ga.dagMu.Unlock()
		return "没有可一键验收的节点（只有「完成」态节点可验收）", nil
	}
	if err := a.dagStore().Save(r); err != nil {
		ga.dagMu.Unlock()
		return "", err
	}
	ga.dagMu.Unlock()

	files := 0
	var failed []string
	for _, node := range flipped {
		files += len(node.Outputs)
		if memErr := dagAcceptMemoryWrite(a, r, node); memErr != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", node.Title, memErr))
			continue
		}
		slog.Info("流水线节点已验收", "run", id, "node", node.ID, "outputs", node.Outputs)
	}
	msg := fmt.Sprintf("已一键验收 %d 个节点、%d 件产物回流记忆。", len(flipped), files)
	if len(failed) > 0 {
		msg += fmt.Sprintf("（%d 条记忆回写失败：%s）", len(failed), strings.Join(failed, "；"))
	}
	return msg, nil
}

// GaeaDagNodeApprove 审批放行（危险操作分级审批的人拍板侧）：hold→pending
// 并置 Approved——后续起跑/续跑/单跑放行；只放行不自动跑（起跑仍人拍板，
// 与整链首跑同闸）。非 hold 节点拒绝。改图变更节点时 Approved 归零（dag 包
// ApplyEdit 新节点不携带）=新指令重新批。
func (a *App) GaeaDagNodeApprove(id, nodeID string) (string, error) {
	ga.dagMu.Lock()
	r, err := a.dagStore().Get(id)
	if err != nil {
		ga.dagMu.Unlock()
		return "", err
	}
	idx := dagNodeIndex(r, nodeID)
	if idx < 0 {
		ga.dagMu.Unlock()
		return "", fmt.Errorf("节点不存在: %s", nodeID)
	}
	if r.Nodes[idx].Status != dag.StatusHold {
		ga.dagMu.Unlock()
		return "", fmt.Errorf("只审批待审批（hold）的节点（当前 %s）", r.Nodes[idx].Status)
	}
	r.Nodes[idx].Status = dag.StatusPending
	r.Nodes[idx].Approved = true
	r.Nodes[idx].Error = ""
	if err := a.dagStore().Save(r); err != nil {
		ga.dagMu.Unlock()
		return "", err
	}
	ga.dagMu.Unlock()
	slog.Info("流水线高风险节点已批准", "run", id, "node", nodeID)
	return fmt.Sprintf("节点 %s 已批准放行（待跑），可起跑/续跑/单跑执行。", nodeID), nil
}

// GaeaDagCancel 终止级联（roadmap §16）：取消 run ctx→在跑节点子代理随之
// 取消→未起跑节点置 skipped。
func (a *App) GaeaDagCancel(id string) (string, error) {
	v, active := ga.dagCancels.Load(id)
	if !active {
		return "", fmt.Errorf("流水线 %s 当前不在运行", id)
	}
	v.(context.CancelFunc)()
	return fmt.Sprintf("流水线 %s 已终止。", id), nil
}

// ── 模板库（6.3 余项：「月度报告」存模板一键重建）────────────────────────

// dagTplStore 模板库：<cwd>/.gaea/work/dag/templates（run 档同域子目录，
// Store.List 只读顶层 *.json 互不混；删档即弃不进用户库）。
func (a *App) dagTplStore() *dag.TemplateStore {
	return dag.NewTemplateStore(filepath.Join(gaeaCwd(), ".gaea", "work", "dag", "templates"))
}

// GaeaDagTemplateSave 把既有流水线存为模板：只取图形状（goal+节点指令+依赖），
// 状态/产物/运行痕迹剥净。name 空则回退 goal 截断（前端给了输入框，后端仍守卫）。
func (a *App) GaeaDagTemplateSave(runID, name string) (string, error) {
	r, err := a.dagStore().Get(runID)
	if err != nil {
		return "", err
	}
	tpl, err := dag.FromRun(r, name)
	if err != nil {
		return "", err
	}
	if err := a.dagTplStore().Save(tpl); err != nil {
		return "", err
	}
	slog.Info("流水线已存为模板", "run", runID, "template", tpl.ID, "nodes", len(tpl.Nodes))
	return fmt.Sprintf("已存为模板「%s」（%d 个节点）。可在文件流水线区顶部「模板」中一键重建。", tpl.Name, len(tpl.Nodes)), nil
}

// GaeaDagTemplateList 全部模板（创建时间倒序）。
func (a *App) GaeaDagTemplateList() ([]dag.Template, error) {
	return a.dagTplStore().List()
}

// GaeaDagTemplateNew 一键重建：模板→全新草稿 run（不自动起跑——起跑仍是人
// 拍板，与整链首跑同闸）。重建即模板当前形状的快照，改模板不影响已重建的 run。
func (a *App) GaeaDagTemplateNew(templateID string) (string, error) {
	tpl, err := a.dagTplStore().Get(templateID)
	if err != nil {
		return "", err
	}
	run := dag.Instantiate(tpl)
	if err := a.dagStore().SaveNew(&run); err != nil {
		return "", fmt.Errorf("重建流水线落盘: %w", err)
	}
	slog.Info("流水线已从模板重建", "template", templateID, "run", run.ID, "nodes", len(run.Nodes))
	return fmt.Sprintf("已从模板「%s」重建流水线 %s（%d 个节点，未起跑）。可检查节点指令后起跑。", tpl.Name, run.ID, len(run.Nodes)), nil
}

// GaeaDagTemplateDelete 删模板（删档即弃；已重建的 run 不受影响）。
func (a *App) GaeaDagTemplateDelete(templateID string) (string, error) {
	if err := a.dagTplStore().Delete(templateID); err != nil {
		return "", err
	}
	return fmt.Sprintf("模板 %s 已删除。", templateID), nil
}

// ── 执行器 ────────────────────────────────────────────────────────────

// dagExecute 波次推进（v4.221 波内并行）：波间依赖分波，波内节点并发各跑各的
// 子代理会话——产物归因已升级为按会话（SessionID=sa_ ref，精确不串），不再
// 依赖「波内顺序保证窗口不重叠」。单节点失败→后续波全部 skipped（上游失败
// 不空跑）。
// dagStart 起跑执行器：cancel 登记在受理方同步完成（返回即「在途」可查可取消），
// goroutine 收尾删除——「登记存在」与「goroutine 在跑」全程一致，无窗口。
func (a *App) dagStart(id string, waves [][]dag.Node) {
	ctx, cancel := context.WithCancel(context.Background())
	ga.dagCancels.Store(id, cancel)
	go func() {
		defer ga.dagCancels.Delete(id)
		// 编排级兜底防线：节点级 recover（下方）盖住子代理执行 panic，本层
		// 盖住波次推进/收尾落盘自身的漏网 panic——不带崩进程；run 状态残留
		// running 由 dagSweep 重启兜底收尾。
		defer func() {
			if r := recover(); r != nil {
				slog.Error("流水线执行 panic", "run", id, "panic", r)
			}
		}()
		a.dagExecute(ctx, id, waves)
	}()
}

func (a *App) dagExecute(ctx context.Context, id string, waves [][]dag.Node) {

	runner := gaDagRunner()
	if runner == nil {
		a.dagFailAll(id, waves, "执行器未接线（引擎尚未构建完成）")
		return
	}
	emit := a.dagEmit()
	for wi, wave := range waves {
		failed := false
		held := false
		var failMu sync.Mutex
		var holdMu sync.Mutex
		var wg sync.WaitGroup
		for _, node := range wave {
			node := node
			wg.Add(1)
			go func() {
				defer wg.Done()
				// 节点级 panic 防线：子代理执行链（LLM 流式/工具/事件回投）
				// panic 转节点 failed（与 err 路径同形：置 failed+波标记失败），
				// 不带崩进程也不悬挂本波其余节点。
				defer func() {
					if r := recover(); r != nil {
						slog.Error("流水线节点执行 panic", "run", id, "node", node.ID, "panic", r)
						failMu.Lock()
						failed = true
						failMu.Unlock()
						_ = a.dagMarkNode(id, node.ID, func(n *dag.Node) {
							n.Status = dag.StatusFailed
							n.Error = fmt.Sprintf("节点执行 panic: %v", r)
						})
					}
				}()
				if ctx.Err() != nil {
					a.dagMarkNode(id, node.ID, func(n *dag.Node) {
						n.Status = dag.StatusSkipped
						n.Error = "已取消"
					})
					return
				}
				// 危险操作分级审批（roadmap §12.4，v4.243）：高风险且未批准的
				// 节点置 hold 待审批，本波不执行；波收尾链停——下游保持待跑
				//（不级联 skipped），人批准（GaeaDagNodeApprove→pending）后续跑放行。
				if node.Risk == dag.RiskHigh && !node.Approved {
					holdMu.Lock()
					held = true
					holdMu.Unlock()
					_ = a.dagMarkNode(id, node.ID, func(n *dag.Node) {
						n.Status = dag.StatusHold
						n.Error = "高风险节点待审批"
					})
					slog.Info("流水线高风险节点挂起待审批", "run", id, "node", node.ID)
					return
				}
				seen := dagJournalIDs() // 窗口回退基线（ref 为空时归因用）
				startErr := a.dagMarkNode(id, node.ID, func(n *dag.Node) {
					n.Status = dag.StatusRunning
					n.RunCount++
					n.Error = ""
				})
				if startErr != nil {
					slog.Warn("流水线节点状态落盘失败", "run", id, "node", node.ID, "error", startErr)
				}
				prompt := a.dagNodePrompt(id, node.ID)
				_, ref, err := runner(gaeaAgent.WithSpace(ctx, gaeaSessionSpace()), prompt, emit)
				outputs := dagJournalTargetsAfter(seen)
				if ref != "" {
					// v4.221 按会话归因：子代理把写盘卡落在自己会话名下
					// （SessionID=sa_ ref），主对话同期写盘不再并入。
					outputs = dagJournalTargetsBySession(ref)
				}
				status, errMsg := dag.StatusDone, ""
				if err != nil {
					status, errMsg = dag.StatusFailed, err.Error()
					failMu.Lock()
					failed = true
					failMu.Unlock()
				}
				a.dagMarkNode(id, node.ID, func(n *dag.Node) {
					n.Status = status
					n.Error = errMsg
					n.Ref = ref
					if status == dag.StatusDone && len(outputs) > 0 {
						n.Outputs = outputs
					}
				})
				slog.Info("流水线节点收跑", "run", id, "node", node.ID, "status", status, "outputs", outputs)
			}()
		}
		wg.Wait()
		if held {
			// 审批闸链停：下游不动（保持 pending，不跑不跳过），等审批后续跑。
			return
		}
		if failed {
			for _, later := range waves[wi+1:] {
				for _, node := range later {
					a.dagMarkNode(id, node.ID, func(n *dag.Node) {
						if n.Status == dag.StatusPending || n.Status == dag.StatusRunning || n.Status == dag.StatusHold {
							n.Status = dag.StatusSkipped
							n.Error = "上游失败跳过"
						}
					})
				}
			}
			return
		}
	}
}

// dagNodePrompt 节点委托指令：goal+上游产物清单+节点自包含 prompt。每次现读
// run 档，保证上游产物是最新事实。
func (a *App) dagNodePrompt(id, nodeID string) string {
	var sb strings.Builder
	sb.WriteString("你在执行一条多节点文件流水线中的一环。工作区根=当前目录，产出文件写工作区相对路径，完成后用一句话收尾说明产出了什么。\n\n")
	if r, err := a.dagStore().Get(id); err == nil {
		fmt.Fprintf(&sb, "总目标：%s\n\n", r.Goal)
		if idx := dagNodeIndex(r, nodeID); idx >= 0 {
			node := r.Nodes[idx]
			var upstream []string
			for _, dep := range node.DependsOn {
				if di := dagNodeIndex(r, dep); di >= 0 && len(r.Nodes[di].Outputs) > 0 {
					upstream = append(upstream, r.Nodes[di].Outputs...)
				}
			}
			if len(upstream) > 0 {
				sb.WriteString("上游节点已产出以下文件，可直接使用：\n")
				for _, u := range upstream {
					fmt.Fprintf(&sb, "- %s\n", u)
				}
				sb.WriteString("\n")
			}
			fmt.Fprintf(&sb, "本节点「%s」指令：\n%s\n", node.Title, node.Prompt)
		}
	}
	return sb.String()
}

// dagMarkNode 读-改-存单节点（互斥锁串行化，防止与验收/清扫并发写丢更新）。
func (a *App) dagMarkNode(id, nodeID string, fn func(n *dag.Node)) error {
	ga.dagMu.Lock()
	defer ga.dagMu.Unlock()
	r, err := a.dagStore().Get(id)
	if err != nil {
		return err
	}
	idx := dagNodeIndex(r, nodeID)
	if idx < 0 {
		return fmt.Errorf("节点不存在: %s", nodeID)
	}
	fn(&r.Nodes[idx])
	return a.dagStore().Save(r)
}

func (a *App) dagFailAll(id string, waves [][]dag.Node, reason string) {
	for _, wave := range waves {
		for _, node := range wave {
			_ = a.dagMarkNode(id, node.ID, func(n *dag.Node) {
				n.Status = dag.StatusFailed
				n.Error = reason
			})
		}
	}
}

// dagEmit 节点文本增量→前端专用通道（与追问同路，wire-only 不落主对话账本）。
func (a *App) dagEmit() func(ref, text string) {
	if ga.wire == nil {
		return nil
	}
	return func(ref, text string) {
		a.emit("gaea-subagent-text", map[string]interface{}{
			"kind": "subagent_text", "text": text, "subagentRef": ref,
		})
	}
}

// ── 产物归因（证据卡窗口增量）───────────────────────────────────────────

// dagJournalIDs 快照当前全部证据卡 ID（跨会话聚合，镜像 GaeaJournalList 读取面）。
func dagJournalIDs() map[string]bool {
	seen := map[string]bool{}
	for _, rec := range dagJournalAll() {
		seen[rec.ID] = true
	}
	return seen
}

// dagJournalTargetsAfter 窗口增量：快照后新增证据卡的 Target（工作区相对路径）。
// v4.221 起只作回退口径（ref 为空的 ephemeral 运行）——有 ref 时按会话归因。
func dagJournalTargetsAfter(seen map[string]bool) []string {
	var out []string
	for _, rec := range dagJournalAll() {
		if rec.Target == "" || seen[rec.ID] || strings.HasPrefix(rec.Target, ".gaea/") {
			continue
		}
		out = append(out, rec.Target)
	}
	sort.Strings(out)
	return out
}

// dagJournalTargetsBySession 按会话归因（v4.221）：SessionID==ref（sa_…）的
// 证据卡 Target。子代理把写盘卡落在自己会话名下（Journal 按会话分文件），
// 主对话/其他节点的同期写盘天然不并入——归因从「窗口增量」升级为精确会话。
func dagJournalTargetsBySession(ref string) []string {
	var out []string
	for _, rec := range dagJournalAll() {
		if rec.Target == "" || rec.SessionID != ref || strings.HasPrefix(rec.Target, ".gaea/") {
			continue
		}
		out = append(out, rec.Target)
	}
	sort.Strings(out)
	return out
}

func dagJournalAll() []evidence.ChangeRecord {
	dir := filepath.Join(gaeaCwd(), ".gaea", "work", "journal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	st, err := evidence.OpenJournal(dir)
	if err != nil {
		return nil
	}
	var all []evidence.ChangeRecord
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		recs, err := st.List(e.Name()[:len(e.Name())-len(".jsonl")])
		if err != nil {
			continue
		}
		all = append(all, recs...)
	}
	return all
}

// ── 小工具 ────────────────────────────────────────────────────────────

func dagNodeIndex(r dag.Run, nodeID string) int {
	for i := range r.Nodes {
		if r.Nodes[i].ID == nodeID {
			return i
		}
	}
	return -1
}

func dagDepsSatisfied(r dag.Run, n dag.Node) bool {
	for _, d := range n.DependsOn {
		if di := dagNodeIndex(r, d); di < 0 ||
			(r.Nodes[di].Status != dag.StatusDone && r.Nodes[di].Status != dag.StatusAccepted) {
			return false
		}
	}
	return true
}
