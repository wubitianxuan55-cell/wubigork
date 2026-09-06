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

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/schedule"
)

func init() {
	tool.RegisterBuiltin(scheduleGet{})
	tool.RegisterBuiltin(scheduleApply{})
	tool.RegisterBuiltin(scheduleAnalyze{})
}

// resolveSchedulePath 缺省路径回落默认计划文件（与板块打开的是同一个）。
func resolveSchedulePath(workDir, path string) string {
	if strings.TrimSpace(path) == "" {
		path = schedule.DefaultRelPath
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
	return "读取工程进度计划文件（.gsched.json，进度计划板块同款数据）并返回 CPM 计算结果：任务表（ES/EF/LS/LF/总时差/关键标记）、搭接关系、工作日历与总工期。工期口径为工作日（按日历扣除周末/节假日）。传 path 缺省读当前计划。用于编辑前了解现状、或核对修改后的计划。"
}

func (scheduleGet) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"计划文件路径（.gsched.json）；缺省=当前计划（进度计划/当前计划.gsched.json）"}
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
	return "写入工程进度计划（进度计划板块同款数据）。两种通道二选一：① project=完整计划 JSON（整计划生成/重排，任务含 id/name/level(0分组,1子任务)/duration(工作日)/isMilestone，搭接 links 含 from/to/type(FS|SS|FF|SF)/lag，calendar 含 workweek(getDay 口径 0=周日..6=周六)/holidays）；② ops=增量操作数组（upsert_task/patch_task/remove_task/set_links/set_meta，用于局部调整如压缩某任务工期、改搭接、设里程碑）。计划编制纪律：工期为工作日整数；里程碑 duration=0；搭接缺省 FS lag=0；分组行 duration=0 且不参与搭接；先分组后子任务按 WBS 顺序排列。写入前引擎自动校验并做 CPM 计算，存在循环依赖/悬空引用/非法字段则整批拒绝（返回错误原文，修复后重试）。回执含写入后总工期与关键工作数，必须核对该结果是否与预期一致。"
}

func (scheduleApply) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"计划文件路径；缺省=当前计划（进度计划/当前计划.gsched.json）"},
  "project":{"type":"object","description":"完整计划对象（整计划生成/重排通道，与 ops 二选一）"},
  "ops":{"type":"array","description":"增量操作数组（局部调整通道，与 project 二选一）。元素 type：upsert_task{task:{id,name,level,duration,...},afterId?} | patch_task{id,patch:{name?/duration?/progress?/mode?/manualStart?/isMilestone?/level?}} | remove_task{id}(含子孙与相关搭接) | set_links{toId,links:[{from,type?,lag?}]}(整体替换该任务入边,type 缺省 FS) | set_meta{name?/startDate?/calendar?}",
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
			if oc := schedule.ComputeCpm(old.Tasks, old.Links); oc.OK {
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
	if cpm.OK {
		afterDesc = fmt.Sprintf("%s → 写入后总工期 %d 天、关键工作 %d 项", afterDesc, cpm.Duration, len(a.Critical))
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

// ── schedule_analyze ─────────────────────────────────────────

type scheduleAnalyze struct{ workDir string }

func (scheduleAnalyze) Name() string { return "schedule_analyze" }

func (scheduleAnalyze) Description() string {
	return "分析工程进度计划：确定性 CPM 引擎裁决 + 规则质检。返回总工期、关键工作链、近关键工作（总时差≤2，缓冲小）、里程碑清单，以及计划检查发现（无任何搭接的孤立任务、无出边的收尾任务、空分组、无里程碑的提醒等）。用于编写/修改后的自检（成功≠正确：先 analyze 再向用户汇报）、进度合理性解读与风险提示。"
}

func (scheduleAnalyze) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"计划文件路径；缺省=当前计划"}
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
	return mustJSON(out)
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
