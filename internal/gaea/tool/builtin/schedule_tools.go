package builtin

// schedule_tools.go — 进度计划 agent 工具三件套（v4.113.0 刀4：AI 原生通道地基）。
//
// 计划文件（.gsched.json）是「进度计划」板块与 agent 的共享资产：
//   - schedule_get     读取当前计划 + CPM 结果（排程全景）；
//   - schedule_apply   全量（project）或增量（ops）写入；快照+证据卡+CPM fail-closed；
//   - schedule_analyze 质检（错漏/悬空/孤立）+ 叙事分析（关键链/近关键/里程碑）。
//
// 口径纪律（与确定性引擎的分工，市场调研结论）：AI 只产结构化字段（任务/工期/
// 搭接/日历），可行性与环检测一律由 gaea CPM 引擎裁决——拒绝即把错误原文回传，
// 模型修复后重试（「成功≠正确」：apply 回执带新总工期，模型须自行核对）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/schedule"
)

func init() {
	tool.RegisterBuiltin(scheduleGet{})
	tool.RegisterBuiltin(scheduleApply{})
	tool.RegisterBuiltin(scheduleAnalyze{})
}

// resolveSchedulePath 缺省路径回落当前工程指针（索引 .gaea/schedule/index.json
// 的 current，与板块打开的是同一个；索引不可用回落默认计划文件）。
// 显式 path 永远优先（多工程：docs/gaea-schedule-multi-project-design-2026-09.md §3.3）。
func resolveSchedulePath(workDir, path string) string {
	if strings.TrimSpace(path) == "" {
		path = schedule.DefaultRelPath
		if workDir != "" {
			if idx, err := schedule.LoadScheduleIndex(workDir); err == nil && strings.TrimSpace(idx.Current) != "" {
				path = idx.Current
			}
		}
	}
	return resolveIn(workDir, path)
}

// scheduleProject 汇总读取：装载 + CPM + 叙事分析。
func scheduleProject(path string) (schedule.Project, schedule.CpmResult, schedule.Analysis, error) {
	p, err := schedule.Load(path)
	if err != nil {
		return p, schedule.CpmResult{}, schedule.Analysis{}, err
	}
	cpm, a := p.Analyze()
	return p, cpm, a, nil
}

// ── schedule_get ─────────────────────────────────────────────

type scheduleGet struct{ workDir string }

func (scheduleGet) Name() string { return "schedule_get" }

func (scheduleGet) Description() string {
	return "读取工程进度计划文件（.gsched.json，进度计划板块同款数据）并返回 CPM 计算结果：任务表（ES/EF/LS/LF/总时差/关键标记）、搭接关系、工作日历与总工期；含资源维度时一并返回资源表（resources）、任务↔资源分配（assignments）与成本汇总（costs：total 总成本/byTask 各任务行成本/byResource 按资源汇总，单位元；费率口径=工时资源元/工日、材料资源元/单位）。工期口径为工作日（按日历扣除周末/节假日）。传 path 缺省读当前工程（可在进度计划板块切换）。用于编辑前了解现状、或核对修改后的计划与成本。"
}

func (scheduleGet) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"计划文件路径（.gsched.json）；缺省=当前工程（可在进度计划板块切换）"}
}
}`)
}

func (scheduleGet) ReadOnly() bool { return true }

func (scheduleGet) CompactDescription() string     { return compactDesc["schedule_get"] }
func (scheduleGet) CompactSchema() json.RawMessage { return compactSchema["schedule_get"] }

func (s scheduleGet) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path string `json:"path,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	path := resolveSchedulePath(s.workDir, p.Path)
	proj, cpm, a, err := scheduleProject(path)
	if err != nil {
		return "", err
	}
	if !cpm.OK {
		return mustJSON(map[string]any{"path": path, "ok": false, "error": cpm.Error, "cycle": cpm.Cycle})
	}
	out := map[string]any{
		"path":      path,
		"ok":        true,
		"name":      proj.Name,
		"startDate": proj.StartDate,
		"calendar": map[string]any{
			"workweek": proj.Calendar.Workweek,
			"holidays": proj.Calendar.Holidays,
		},
		"duration":      cpm.Duration,
		"criticalCount": len(a.Critical),
		"leafCount":     a.LeafCount,
		"tasks":         taskTable(proj, cpm),
		"links":         proj.Links,
	}
	// 资源成本刀2（v4.122）：资源表/分配集/成本汇总（ComputeCosts；CPM 不过时
	// costs.ok=false 成本不出——本工具在 CPM 不过时已提前整单返回，此为兜底口径）。
	resources := proj.Resources
	if resources == nil {
		resources = []schedule.Resource{}
	}
	assignments := proj.Assignments
	if assignments == nil {
		assignments = []schedule.Assignment{}
	}
	costs := schedule.ComputeCosts(proj, cpm)
	out["resources"] = resources
	out["assignments"] = assignments
	out["costs"] = map[string]any{
		"ok":         costs.OK,
		"total":      costs.Total,
		"byTask":     costs.Rows,
		"byResource": costs.ByResource,
	}
	if b := proj.Baseline; b != nil {
		out["baseline"] = map[string]any{"name": b.Name, "savedAt": b.SavedAt, "duration": b.Duration, "rowCount": len(b.Rows)}
	}
	if proj.Deadline != "" {
		out["deadline"] = proj.Deadline
	}
	return mustJSON(out)
}

// taskView 单任务 + CPM 行（JSON 输出形态）。
type taskView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Level       int    `json:"level"`
	Duration    int    `json:"duration"`
	Progress    int    `json:"progress"`
	IsMilestone bool   `json:"isMilestone,omitempty"`
	Mode        string `json:"mode,omitempty"`
	ManualStart int    `json:"manualStart,omitempty"`
	ES          int    `json:"es"`
	EF          int    `json:"ef"`
	LS          int    `json:"ls"`
	LF          int    `json:"lf"`
	TF          int    `json:"tf"`
	FF          int    `json:"ff"`
	Critical    bool   `json:"critical"`
}

func taskTable(p schedule.Project, cpm schedule.CpmResult) []taskView {
	out := make([]taskView, 0, len(p.Tasks))
	for _, t := range p.Tasks {
		row := cpm.Rows[t.ID]
		mode := string(t.Mode)
		if mode == "" {
			mode = string(schedule.ModeAuto)
		}
		out = append(out, taskView{
			ID: t.ID, Name: t.Name, Level: t.Level, Duration: t.Duration,
			Progress: t.Progress, IsMilestone: t.IsMilestone, Mode: mode,
			ManualStart: t.ManualStart,
			ES:          row.ES, EF: row.EF, LS: row.LS, LF: row.LF, TF: row.TF, FF: row.FF,
			Critical: row.Critical,
		})
	}
	return out
}

// ── schedule_apply ───────────────────────────────────────────

type scheduleApply struct {
	workDir string
	roots   []string
}

func (scheduleApply) Name() string { return "schedule_apply" }

func (scheduleApply) Description() string {
	return "写入工程进度计划（进度计划板块同款数据）。两种通道二选一：① project=完整计划 JSON（整计划生成/重排，任务含 id/name/level(0分组,1子任务)/duration(工作日)/isMilestone/fixedCost(固定成本,元,叶任务专属)，搭接 links 含 from/to/type(FS|SS|FF|SF)/lag，resources 含 id/name/type(work|material|cost)/standardRate(元/工日或元/单位)/costPerUse(每次使用,元)/unit(材料计量单位)/maxUnits，assignments 含 taskId/resourceId/units|quantity|amount，calendar 含 workweek(getDay 口径 0=周日..6=周六)/holidays）；② ops=增量操作数组（upsert_task/patch_task/remove_task/set_links/set_meta/upsert_resource/patch_resource/remove_resource/set_assignments，用于局部调整如压缩某任务工期、改搭接、挂资源、调费率）。计划编制纪律：工期为工作日整数；里程碑 duration=0；搭接缺省 FS lag=0；分组行 duration=0、不参与搭接、禁挂分配与固定成本；费率一律元/工日（工时）与元/单位（材料）；先建资源再挂分配。写入前引擎自动校验并做 CPM 计算，存在循环依赖/悬空引用/非法字段则整批拒绝（返回错误原文，修复后重试）。回执含写入后总工期、关键工作数与总成本（totalCost 及涉及任务行成本），必须核对该结果是否与预期一致。确认机制：project 整量通道须经用户在 diff 确认卡批准后才执行（任何权限级别逐条确认，不存在会话放行）；被拒即未写入，不要原样重发——按用户在对话里给出的意见修改参数后重发是新一次确认，连续被拒两次应停止下发并要明确口径。ops 通道在 ask 权限级别同样弹确认卡。回执与预览数字不一致时以回执为准并向用户点破。"
}

func (scheduleApply) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"计划文件路径；缺省=当前工程（可在进度计划板块切换）"},
  "project":{"type":"object","description":"完整计划对象（整计划生成/重排通道，与 ops 二选一）"},
  "ops":{"type":"array","description":"增量操作数组（局部调整通道，与 project 二选一）。元素 type：upsert_task{task:{id,name,level,duration,...},afterId?} | patch_task{id,patch:{name?/duration?/progress?/mode?/manualStart?/isMilestone?/level?/fixedCost?(元,叶任务专属)}} | remove_task{id}(含子孙与相关搭接) | set_links{toId,links:[{from,type?,lag?}]}(整体替换该任务入边,type 缺省 FS) | set_meta{name?/startDate?/calendar?/deadline?}(deadline=YYYY-MM-DD 目标竣工日期，倒排校核用；空串清除) | auto_chain{}(推荐逻辑关系缺省步：仅为无前置叶任务按 WBS 顺序补 FS 串联，已有逻辑/手动任务不动) | set_baseline{name?}(固化当前排程为基线，重大调整前建议先做；循环依赖/无叶任务会拒绝) | clear_baseline{}(清除基线，无基线时报错) | upsert_resource{resource:{id,name,type:work|material|cost,unit?,standardRate?,costPerUse?,maxUnits?}}(整量新增或按 id 替换；work 费率=元/工日，material=元/单位，cost 资源无费率) | patch_resource{id,resourcePatch:{name?/type?/unit?/standardRate?/costPerUse?/maxUnits?}}(指针三态，缺省=不动) | remove_resource{id}(级联删除其全部分配) | set_assignments{taskId,assignments:[{resourceId,units?/quantity?/amount?}]}(整体替换该任务分配集；先建资源再挂分配；分组行拒绝)",
    "items":{"type":"object","properties":{"type":{"type":"string"}},"required":["type"]}},
  "summary":{"type":"string","description":"本次修改的一句话摘要（落证据卡，供用户在轨迹中审阅）"}
},
"anyOf":[{"required":["project"]},{"required":["ops"]}]
}`)
}

func (scheduleApply) ReadOnly() bool { return false }

func (scheduleApply) CompactDescription() string     { return compactDesc["schedule_apply"] }
func (scheduleApply) CompactSchema() json.RawMessage { return compactSchema["schedule_apply"] }

func (s scheduleApply) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path    string            `json:"path,omitempty"`
		Project *schedule.Project `json:"project,omitempty"`
		Ops     []schedule.Op     `json:"ops,omitempty"`
		Summary string            `json:"summary,omitempty"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if (in.Project == nil) == (len(in.Ops) == 0) {
		return "", fmt.Errorf("project 与 ops 恰好二选一（整计划生成传 project，局部调整传 ops）")
	}
	path := resolveSchedulePath(s.workDir, in.Path)
	if err := confine(s.roots, path); err != nil {
		return "", err
	}

	// 读取现状（可不存在=新建）：Before 摘要 + 基线快照原料
	beforeDesc := "（新建计划）"
	var oldRaw []byte
	if raw, err := os.ReadFile(path); err == nil {
		oldRaw = raw
		if old, lerr := schedule.Load(path); lerr == nil {
			if oc := schedule.ComputeCpmCal(old.Tasks, old.Links, old.Calendar, old.StartDate); oc.OK {
				beforeDesc = fmt.Sprintf("「%s」%d 项工作，总工期 %d 天", old.Name, leafCount(old), oc.Duration)
			}
		}
	}

	var proj schedule.Project
	changes := []string{}
	if in.Project != nil {
		proj = *in.Project
		if proj.Tasks == nil {
			proj.Tasks = []schedule.Task{}
		}
		if proj.Links == nil {
			proj.Links = []schedule.Link{}
		}
		changes = append(changes, fmt.Sprintf("整计划写入「%s」（%d 项工作 / %d 条搭接）", proj.Name, leafCount(proj), len(proj.Links)))
	} else {
		cur, err := schedule.Load(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
			cur = schedule.Project{Tasks: []schedule.Task{}, Links: []schedule.Link{}}
		}
		// set_baseline 的 savedAt 由工具层标注（引擎是纯函数不取时钟）
		now := time.Now().Format("2006-01-02 15:04")
		for i := range in.Ops {
			if in.Ops[i].Type == "set_baseline" && strings.TrimSpace(in.Ops[i].SavedAt) == "" {
				in.Ops[i].SavedAt = now
			}
		}
		sums, err := schedule.ApplyOps(&cur, in.Ops)
		if err != nil {
			return "", err
		}
		changes = append(changes, sums...)
		proj = cur
	}

	if err := schedule.Save(path, proj); err != nil {
		return "", err // 校验/环检测失败原文回传，模型修复重试
	}

	baseline := ""
	if oldRaw != nil {
		baseline = evidence.StageBaseline(ctx, path, oldRaw)
	}
	afterDesc := strings.Join(changes, "；")
	cpm, a := proj.Analyze()
	costs := schedule.ComputeCosts(proj, cpm)
	if cpm.OK {
		afterDesc = fmt.Sprintf("%s → 写入后总工期 %d 天、关键工作 %d 项、总成本 %.2f 元", afterDesc, cpm.Duration, len(a.Critical), costs.Total)
	}
	evidence.RecordChange(ctx, evidence.ChangeRecord{
		Tool:          "schedule_apply",
		Target:        path,
		BeforeSummary: beforeDesc,
		AfterSummary:  afterDesc,
		BaselinePath:  baseline,
	})

	out := map[string]any{
		"ok":        true,
		"path":      path,
		"changes":   changes,
		"summary":   in.Summary,
		"duration":  cpm.Duration,
		"critical":  a.Critical,
		"taskCount": a.LeafCount,
	}
	// 资源成本刀2（v4.122）：写入后总成本——「成功≠正确」纪律延伸，改费率/
	// 工期/分配都必须让模型核对总成本变化；ops 通道另带涉及任务的行成本。
	out["totalCost"] = costs.Total
	if touched := costTouchedTaskIDs(in.Ops); len(touched) > 0 && costs.OK {
		rows := map[string]schedule.TaskCost{}
		for _, id := range touched {
			if row, ok := costs.Rows[id]; ok {
				rows[id] = row
			}
		}
		if len(rows) > 0 {
			out["taskCosts"] = rows
		}
	}
	// 已有基线时回执带漂移摘要——模型汇报前先讲「较基线」偏差
	if d := schedule.ComputeBaselineDrift(&proj, cpm); d != nil {
		out["baselineDrift"] = map[string]any{
			"name":           d.BaselineName,
			"baselineToNow":  fmt.Sprintf("%d→%d 天", d.BaselineDuration, d.CurrentDuration),
			"durationDrift":  d.DurationDrift,
			"shifted":        d.ShiftedCount,
			"added":          d.AddedCount,
			"removed":        d.RemovedCount,
			"criticalGained": d.CriticalGained,
			"criticalLost":   d.CriticalLost,
		}
	}
	// 已设目标竣工时回执带倒排校核（可行性+超期/富余量）
	if d := schedule.CheckDeadline(&proj, cpm); d != nil {
		out["deadlineCheck"] = d
	}
	if in.Summary != "" {
		out["declaredSummary"] = in.Summary
	}
	if !cpm.OK {
		out["warning"] = cpm.Error
	}
	return mustJSON(out)
}

func leafCount(p schedule.Project) int {
	n := 0
	for _, t := range p.Tasks {
		if t.Level > 0 {
			n++
		}
	}
	return n
}

// costTouchedTaskIDs ops 通道中涉及成本的任务 id（去重保序）。set_links/
// auto_chain 不改行成本（工期不变），remove_task 的行已不存在（Rows 查不到
// 自然不出现在回执），均不入列；project 整计划通道 ops 为空 → 回执只带总成本。
func costTouchedTaskIDs(ops []schedule.Op) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ops))
	for _, op := range ops {
		id := ""
		switch op.Type {
		case "upsert_task":
			if op.Task != nil {
				id = op.Task.ID
			}
		case "patch_task", "remove_task":
			id = op.ID
		case "set_assignments":
			id = op.TaskID
		}
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// ── schedule_analyze ─────────────────────────────────────────

type scheduleAnalyze struct{ workDir string }

func (scheduleAnalyze) Name() string { return "schedule_analyze" }

func (scheduleAnalyze) Description() string {
	return "分析工程进度计划：确定性 CPM 引擎裁决 + 规则质检。返回总工期、关键工作链、近关键工作（总时差≤2，缓冲小）、里程碑清单，以及计划检查发现（无任何搭接的孤立任务、无前置的任务——可用 ops auto_chain 一键补缺省串联、空分组、无收口尾巴、无里程碑提醒等）。含资源维度时附带成本叙事（costs：total 总成本、topTasks 成本 Top5 任务、byResource 按资源汇总，单位元）与「已分配未定价」发现（工时资源未设费率或成本资源缺金额，成本按 0 计——AI 建议层提醒，非引擎拒绝）。已保存基线时附带漂移对比（baselineDrift：总工期 X→Y、推移/新增/移除、关键链进出），汇报调整效果时先讲这组偏差。已设目标竣工时附带倒排校核（deadlineCheck：可行性裁决、超期/富余量、关键工作清单——超期时压缩对象即关键链）。用于编写/修改后的自检（成功≠正确：先 analyze 再向用户汇报）、进度合理性解读与风险提示、推荐逻辑关系的依据。"
}

func (scheduleAnalyze) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"计划文件路径；缺省=当前工程（可在进度计划板块切换）"}
}
}`)
}

func (scheduleAnalyze) ReadOnly() bool { return true }

func (scheduleAnalyze) CompactDescription() string     { return compactDesc["schedule_analyze"] }
func (scheduleAnalyze) CompactSchema() json.RawMessage { return compactSchema["schedule_analyze"] }

func (s scheduleAnalyze) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path string `json:"path,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	path := resolveSchedulePath(s.workDir, p.Path)
	proj, cpm, a, err := scheduleProject(path)
	if err != nil {
		return "", err
	}
	if !cpm.OK {
		return mustJSON(map[string]any{"path": path, "ok": false, "error": cpm.Error, "cycle": cpm.Cycle})
	}
	out := map[string]any{
		"path":         path,
		"ok":           true,
		"name":         proj.Name,
		"startDate":    proj.StartDate,
		"duration":     cpm.Duration,
		"critical":     a.Critical,
		"nearCritical": a.NearCritical,
		"milestones":   a.Milestones,
		"checks":       qualityChecks(proj, cpm),
	}
	findings := out["checks"].([]string)
	// 资源成本刀2（v4.122）：成本叙事数据（总成本/Top5/按资源汇总/未定价清单）——
	// 后续刀3 板块与走查依赖这组字段名。
	narrative := proj.CostNarrative(cpm)
	out["costs"] = narrative
	if len(narrative.Unpriced) > 0 {
		findings = append(findings, unpricedFinding(narrative.Unpriced))
	}
	if d := schedule.ComputeBaselineDrift(&proj, cpm); d != nil {
		out["baselineDrift"] = d
		if d.DurationDrift != 0 || d.ShiftedCount > 0 || d.AddedCount > 0 || d.RemovedCount > 0 {
			findings = append(findings, fmt.Sprintf("较基线「%s」：总工期 %d→%d 天（%+d），推移 %d · 新增 %d · 移除 %d",
				d.BaselineName, d.BaselineDuration, d.CurrentDuration, d.DurationDrift, d.ShiftedCount, d.AddedCount, d.RemovedCount))
			if len(d.CriticalGained) > 0 {
				findings = append(findings, fmt.Sprintf("较基线新进关键线路：%s", strings.Join(d.CriticalGained, "、")))
			}
			if len(d.CriticalLost) > 0 {
				findings = append(findings, fmt.Sprintf("较基线退出关键线路：%s", strings.Join(d.CriticalLost, "、")))
			}
		}
	}
	// 已设目标竣工时输出倒排校核（引擎裁决可行性，压缩方案留给 AI 建议）
	if d := schedule.CheckDeadline(&proj, cpm); d != nil {
		out["deadlineCheck"] = d
		if !d.Feasible {
			findings = append(findings, fmt.Sprintf("倒排校核：目标竣工 %s（第 %d 工作日）不可达——总工期 %d 天，超 %d 天；需压缩关键线路", d.Deadline, d.TargetWorkdays, d.CurrentDuration, d.Overrun))
		} else if d.Overrun >= -2 {
			findings = append(findings, fmt.Sprintf("倒排校核：距目标竣工仅剩 %d 天富余，关键任务拖延即触线", -d.Overrun))
		}
	}
	out["checks"] = findings
	return mustJSON(out)
}

// unpricedFinding 「已分配未定价」发现文案（AI 建议层：成本按 0 计，非引擎拒绝；
// 设计 §3.2——work 无费率且无每次使用 / cost 缺金额）。
func unpricedFinding(items []schedule.UnpricedAssignment) string {
	parts := make([]string, 0, len(items))
	for _, u := range items {
		switch u.Kind {
		case schedule.UnpricedWorkNoRate:
			parts = append(parts, fmt.Sprintf("「%s」的资源「%s」未设费率", u.TaskName, u.ResourceName))
		case schedule.UnpricedCostNoAmount:
			parts = append(parts, fmt.Sprintf("「%s」的资源「%s」未填金额", u.TaskName, u.ResourceName))
		default:
			parts = append(parts, fmt.Sprintf("「%s」↔「%s」", u.TaskName, u.ResourceName))
		}
	}
	if len(parts) > 3 {
		parts = append(parts[:3:3], fmt.Sprintf("…等 %d 处", len(parts)))
	}
	return fmt.Sprintf("已分配未定价 %d 处（成本按 0 计，建议补费率/金额）：%s", len(items), strings.Join(parts, "、"))
}

// qualityChecks 规则质检（斑马「AI 计划检查」口径的确定性子集）。
func qualityChecks(p schedule.Project, cpm schedule.CpmResult) []string {
	findings := make([]string, 0)
	inEdge := map[string]bool{}
	outEdge := map[string]bool{}
	for _, l := range p.Links {
		outEdge[l.From] = true
		inEdge[l.To] = true
	}
	groupHasChild := map[string]bool{}
	for i, t := range p.Tasks {
		if t.Level == 0 {
			if i+1 < len(p.Tasks) && p.Tasks[i+1].Level > 0 {
				groupHasChild[t.ID] = true
			}
		}
	}
	isolated := 0
	noOut := 0
	for _, t := range p.Tasks {
		if t.Level == 0 {
			if !groupHasChild[t.ID] {
				findings = append(findings, fmt.Sprintf("分组「%s」没有任何子任务", t.Name))
			}
			continue
		}
		if t.IsMilestone {
			continue
		}
		if !inEdge[t.ID] && !outEdge[t.ID] {
			isolated++
			findings = append(findings, fmt.Sprintf("任务「%s」没有任何搭接关系，像孤立项（确认是否漏了前置）", t.Name))
			continue
		}
		if !outEdge[t.ID] && cpm.Rows[t.ID].EF < cpm.Duration {
			noOut++
		}
	}
	if isolated == 0 {
		findings = append(findings, "搭接关系覆盖良好：无孤立任务")
	} else {
		findings = append(findings, fmt.Sprintf("共 %d 个孤立任务", isolated))
	}
	// 无前置（无入边）叶任务：推荐逻辑关系的主输入。首个非手动叶是计划
	// 起点、天然无前置不计；手动任务=有意定位，也不计。
	noIn := 0
	seenLeaf := false
	for _, t := range p.Tasks {
		if t.Level == 0 || t.Mode == schedule.ModeManual {
			continue
		}
		if !seenLeaf {
			seenLeaf = true
			continue
		}
		if !inEdge[t.ID] {
			noIn++
		}
	}
	if noIn > 0 {
		findings = append(findings, fmt.Sprintf("%d 个任务无前置搭接（可先 ops auto_chain 补缺省串联，再按工艺逻辑用 set_links 覆写例外）", noIn))
	}
	if noOut > 1 {
		findings = append(findings, fmt.Sprintf("%d 个任务无后续却不在总工期收尾（多条并行尾巴，确认收口里程碑是否遗漏）", noOut))
	}
	hasMilestone := false
	for _, t := range p.Tasks {
		if t.IsMilestone {
			hasMilestone = true
			break
		}
	}
	if !hasMilestone && len(p.Tasks) > 0 {
		findings = append(findings, "计划中没有里程碑（建议为关键交付节点设里程碑，duration=0）")
	}
	return findings
}

// mustJSON 序列化工具输出（单行紧凑）。
func mustJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
