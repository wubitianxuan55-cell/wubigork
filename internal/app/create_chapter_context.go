package app

// create_chapter_context.go — 章节生成的上下文编译与写前门辅助族
// （P5 破防专项拆自 create_chapter_handler.go：预算常量/前文摘要窗口/增强区段
// 组装/计划门辅助与区段渲染/世界观与文风构建/角色摘要——纯函数为主，搬移零
// 功能变更）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/novelgate"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

const (
	ctxBudgetTotal        = 4000 // 新增区段（伏笔+世界观）合计预算
	ctxForeshadowBudget   = 1600 // 伏笔区正文上限
	ctxWorldviewBudget    = 1600 // 世界观区正文上限
	ctxStyleBudget        = 1200 // 书级文风档案区上限（T6：偏好宜精不宜多）
	ctxForeshadowMaxItems = 15   // 最多注入的伏笔条数
	ctxWorldviewDimLen    = 150  // 世界观单维度截断
	ctxSceneBibleBudget   = 2200 // 场景圣经（POV 感知上下文）注入上限（rune）
	// 刀1 §7.3 意图注入：章节计划 / 大纲要点两个新区段各自的上限。它们与伏笔/
	// 文风/世界观共用一个 ctxBudgetTotal 口径（见 createChapter 的余额扣减）。
	ctxChapterPlanBudget   = 1200 // 本章计划区段上限（rune）
	ctxOutlinePointsBudget = 800  // 大纲要点（KeyPoints/Emotion）区段上限（rune）
	// 刀6：setting 槽入预算——此前前端传整篇设定完全无裁剪（预算外堆料的最大
	// 残留）；截断后与「世界观要点」区段互补（要点仍在）。
	ctxSettingBudget = 3000
)

// 角色摘要字段预算。性格截断由原 20 rune 放宽到 60，另补身份/目标/关系要点。
const (
	charPersonalityLen  = 60   // 性格截断上限（原 20）
	charFieldLen        = 60   // 身份背景/目标等字段截断
	charRelationDescLen = 30   // 单条关系描述截断
	charMaxRelations    = 3    // 每角色最多注入关系数
	charSummaryBudget   = 2400 // 角色摘要整体预算（rune）
)

// 前文摘要窗口（t3 首刀，spec docs/distill/03-long-range-consistency.md §12.1，
// 对齐 MuMu chapter_context_service.py:1375/:1408 的「最近 10 章摘要窗口」）。
const (
	ctxPrevWindowChapters = 10 // 只注入本章之前的最近 N 章
	ctxPrevChapterLen     = 180 // 窗口内单章摘要截断（rune）
	// 阶段八 8.3：回看包总上限显式化（Sudowrite Continuity 双上限口径的 gaea
	// 对应物——前文窗口合计 ≤ 窗口预算；正文原文永不压缩、全量在盘）。
	ctxPrevWindowBudget = ctxPrevWindowChapters*ctxPrevChapterLen + 80
)

// 阶段八 8.3：故事骨架/阶段合账并入组装器统一信封的额度划拨。
// spine 此前是无预算独立追加；recap 正文受 stageRecapBudgetRunes 约束但标题
// 与开篇纪律行在其外——信封按「内部预算+结构开销」划拨，超信封由裁剪次序
// （合账 Rank 2 先于骨架 Rank 1 牺牲）兜底。
const (
	ctxSpineBudget       = 1600
	ctxStageRecapSection = stageRecapBudgetRunes + 400
)

// buildPrevSummaryWindow 组装「最近 N 章」前文摘要窗口（纯函数，确定性输出）。
//
// 只取 OrderIndex ∈ [limit-N, limit) 的章节，按章号升序拼接；单章摘要经
// resolve 回退链取文并截断到 ctxPrevChapterLen；取不到文的章节跳过（不占位）。
// 整体带窗口声明头——告诉模型这是有意的部分视图，更早剧情仍然有效。
// 无可注入内容返回 ""（模板槽位按空跳过）。
func buildPrevSummaryWindow(nodes []types.OutlineNode, limitChapter int, resolve func(types.OutlineNode) string) string {
	if limitChapter <= 0 || resolve == nil {
		return ""
	}
	floor := limitChapter - ctxPrevWindowChapters
	if floor < 1 {
		floor = 1
	}
	picked := make([]types.OutlineNode, 0, ctxPrevWindowChapters)
	for _, n := range nodes {
		cn := n.OrderIndex
		if cn >= floor && cn < limitChapter && resolve(n) != "" {
			picked = append(picked, n)
		}
	}
	if len(picked) == 0 {
		return ""
	}
	sort.Slice(picked, func(i, j int) bool { return picked[i].OrderIndex < picked[j].OrderIndex })
	parts := make([]string, 0, len(picked)+1)
	parts = append(parts, fmt.Sprintf("（仅含第 %d～%d 章概要；更早章节的剧情同样有效，只是不在此重复）", floor, limitChapter-1))
	for _, n := range picked {
		parts = append(parts, fmt.Sprintf("第%d章：%s", n.OrderIndex, truncateBudget(resolve(n), ctxPrevChapterLen)))
	}
	// 阶段八 8.3：窗口合计显式封顶（双上限口径），正常情况下窗口数学已保证
	// 不超，此封顶是防上游摘要变更后的最后护栏。
	return truncateBudget(strings.Join(parts, "\n\n"), ctxPrevWindowBudget)
}

// runeLen 字符串的 rune 长度
func runeLen(s string) int {
	return len([]rune(s))
}

// truncateBudget 将 s 截断到不超过 budget 个 rune（省略号计入预算，
// util.Truncate 的 "..." 后缀占 3 rune）。
func truncateBudget(s string, budget int) string {
	if budget < 3 {
		budget = 3
	}
	if runeLen(s) <= budget {
		return s
	}
	return util.Truncate(s, budget-3)
}

// buildChapterContextSections 组装章节生成 prompt 的增强上下文区段
// （分层伏笔调度 + 世界观要点）。无数据或读取失败时返回 ""，调用方不追加
// 任何内容（prompt 中不出现空区段）。分层伏笔渲染见 foreshadow_context.go。
func buildChapterContextSections(pm *project.Manager, currentChapter int) string {
	return buildChapterContextSectionsWithin(pm, currentChapter, ctxBudgetTotal, "", "")
}

// ── 上下文组装器（阶段八 8.3，Sudowrite Chapter Continuity 同型）──────────────
//
// 章节生成的增强区段统一经 assembleViewSections 装配：合计不超预算时各区段
// 原文全量拼接（顺序=入参顺序）；超预算时按固定裁剪次序逐段牺牲——Rank 大者
// 先切（同 Rank 保入参序），每段只裁「刚好够」的量，允许部分片段注入
// （对齐 Sudowrite Continuity「允许部分片段」口径）。确定性：同输入两次装配
// 逐字节一致。
//
// 章节生成五区段的裁剪次序（先牺牲 → 最后保）：
//
//	世界观要点(3) → 阶段合账(2) → 文风档案(1) → 故事骨架(1) → 未回收伏笔(0)
//
// 伏笔是「创作约束」（不得矛盾），最后牺牲；世界观是背景板，最先切——
// 依据竞品调研 §5.1（Sudowrite 裁剪次序表：Worldbuilding 最先、Style 靠后）。

// viewSection 组装器的最小区段单元。
type viewSection struct {
	Key  string // 语义键（dump/日志可读；组装器本身不消费）
	Body string // 区段全文（含标题行）；空串=跳过不占额度
	Rank int    // 牺牲序：数值越大越先被裁（0=最后牺牲）
}

// assembleViewSections 固定裁剪次序的上下文组装器（纯函数）。
// 合计（含 "\n\n" 拼接开销）≤ budget 时全量拼接；超预算时按 Rank 从大到小
// 逐段裁剪，每段裁到「刚好够」，裁到放不下省略号（3 rune）时整段牺牲。
func assembleViewSections(sections []viewSection, budget int) string {
	type liveSec struct {
		idx, rank, length int
	}
	var live []liveSec
	total := 0
	for i, s := range sections {
		if strings.TrimSpace(s.Body) == "" {
			continue // 空区段（含纯空白）跳过不占额度，也不产生拼接符
		}
		l := runeLen(s.Body)
		live = append(live, liveSec{idx: i, rank: s.Rank, length: l})
		total += l
	}
	if len(live) == 0 {
		return ""
	}
	joiner := 2 * (len(live) - 1)
	if total+joiner <= budget {
		parts := make([]string, len(live))
		for k, ls := range live {
			parts[k] = sections[ls.idx].Body
		}
		return strings.Join(parts, "\n\n")
	}
	// 超预算：Rank 大者先牺牲，每段只裁「刚好够」的量
	order := append([]liveSec(nil), live...)
	sort.SliceStable(order, func(i, j int) bool { return order[i].rank > order[j].rank })
	remain := total + joiner - budget
	caps := make(map[int]int, len(live))
	for _, ls := range live {
		caps[ls.idx] = ls.length // 默认全量保留
	}
	for _, ls := range order {
		if remain <= 0 {
			break
		}
		switch {
		case remain >= ls.length:
			caps[ls.idx] = 0 // 整段牺牲
			remain -= ls.length
		case ls.length > 3: // 部分裁剪（省略号计入预算）
			caps[ls.idx] = ls.length - remain
			remain = 0
		default: // 剩余量连省略号都放不下且整段也不够抵：整段牺牲
			caps[ls.idx] = 0
			remain -= ls.length
		}
	}
	parts := make([]string, 0, len(live))
	for _, ls := range live {
		if c := caps[ls.idx]; c > 0 {
			parts = append(parts, truncateBudget(sections[ls.idx].Body, c))
		}
	}
	return strings.Join(parts, "\n\n")
}

// buildChapterContextSectionsWithin 同 buildChapterContextSections，但预算可调：
// 刀1 的章节计划/大纲要点区段已经从 ctxBudgetTotal 里先占额度，剩余额度经本函数
// 传给伏笔/文风/世界观，保证「计划 + 大纲 + 增强区段」合计不超过同一个 ctxBudgetTotal。
// 阶段八 8.3：extraSections（故事骨架/阶段合账）并入同一组装器同一信封——
// 超预算时按固定裁剪次序牺牲（见 assembleViewSections 注释），不再是无上限独立追加。
func buildChapterContextSectionsWithin(pm *project.Manager, currentChapter, budget int, relevanceHint, digestInstr string, extraSections ...viewSection) string {
	if budget <= 0 {
		return "" // 额度耗尽：不追加任何区段（宁缺毋滥，不写半截截断文本）
	}
	sections := make([]viewSection, 0, 3+len(extraSections))
	if body := buildForeshadowSection(pm, currentChapter); body != "" {
		sections = append(sections, viewSection{Key: "foreshadow", Rank: 0, Body: "## 未回收伏笔（创作约束）\n"+
			"以下伏笔已埋设尚未回收，写作时不得与之矛盾；可自然推进，不要强行揭穿：\n" + body})
	}
	// 刀6 文风去重合并：style.md（显式偏好）与成稿学习指令（刀5 digest）此前
	// 是两个独立区段两个标题（§1.8「文风两处」的延续）——合并为单一区段，
	// 偏好优先、学习跟随。
	if body := joinStyleSections(styleSectionBody(pm), digestInstr); body != "" {
		sections = append(sections, viewSection{Key: "style", Rank: 1, Body: body})
	}
	sections = append(sections, extraSections...)
	if body := buildWorldviewSection(pm, relevanceHint); body != "" {
		sections = append(sections, viewSection{Key: "worldview", Rank: 3, Body: "## 世界观要点\n" + body})
	}
	return assembleViewSections(sections, budget)
}

// ── 章节计划写前硬闸与意图注入（刀1 线B，规格 §7.1-3 / §7.3）────────────
//
// 判据全在 novelgate（确定性、零 IO）；本段只做 IO 与组装：
//
//	chapters/plans.json → PlanContractIssues（六项齐备性 + 跨章关键事件去重）
//	outline.json 节点   → OutlineContractIssues（既有写前契约，转换为 PlanProblem）
//	                    → PlanGateReport（Blocking = 存在 S1/S2）
//
// planPrecheck 是硬闸的**唯一判据来源**：CreateChapter 与线C 的
// NovelChapterGatePrecheck 走同一入口，两处结论必然一致。

// readChapterPlansForGate 读全库章节计划（线A 接口 internal/project/plan_store.go:56
// ReadChapterPlans() (*types.ChapterPlanFile, error)：缺失=空正常态 / 损坏=error 不覆盖）。
// JSON 损坏一律返回 error——如实报错，绝不假装「没有计划」。
func readChapterPlansForGate(pm *project.Manager) (*types.ChapterPlanFile, error) {
	if pm == nil {
		return nil, fmt.Errorf("项目未打开：无法读取章节计划")
	}
	f, err := pm.ReadChapterPlans()
	if err != nil {
		return nil, fmt.Errorf("读取章节计划失败：%w", err)
	}
	if f == nil {
		return &types.ChapterPlanFile{Version: 1, Plans: map[string]types.ChapterPlan{}}, nil
	}
	if f.Plans == nil {
		f.Plans = map[string]types.ChapterPlan{}
	}
	return f, nil
}

// planPrecheck 生成前预检（硬闸唯一判据来源）：计划齐备性 + 跨章去重 + 大纲节点契约。
//
// chapterNum 为**本章章号**（生成前口径，见 resolveTargetChapterNum）。
// 无项目 / 计划损坏 / 大纲读失败 → 返回 error（前端据此如实提示，不假装可生成）；
// plans.json 缺失 = 空正常态（HasPlan=false → PlanContractIssues 出 S1 阻断）。
func (a *writingState) planPrecheck(chapterNum int) (*types.PlanGateReport, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("项目未打开：无法做写前预检")
	}
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章号无效（%d）：无法做写前预检", chapterNum)
	}
	plans, err := readChapterPlansForGate(pm)
	if err != nil {
		return nil, err
	}

	report := &types.PlanGateReport{ChapterNum: chapterNum}
	key := strconv.Itoa(chapterNum)
	var plan *types.ChapterPlan
	if plans != nil {
		if p, ok := plans.Plans[key]; ok {
			plan = &p
		}
	}
	report.HasPlan = plan != nil

	// others = 其它章节的计划（用于跨章关键事件去重）。
	others := make([]types.ChapterPlan, 0)
	if plans != nil {
		for k, p := range plans.Plans {
			if k == key {
				continue
			}
			// 跨章重复要点名「第 N 章」：ChapterPlan 里只有 SubIndex 承载全局序。
			// 计划自身没写时用 plans.json 的 key（章号）补齐，不编造、不覆盖已有值。
			if p.SubIndex <= 0 {
				if n, cerr := strconv.Atoi(k); cerr == nil {
					p.SubIndex = n
				}
			}
			others = append(others, p)
		}
	}
	// 稳定序：SubIndex 升序——重复点名结果不随 map 遍历顺序漂移。
	sort.Slice(others, func(i, j int) bool { return others[i].SubIndex < others[j].SubIndex })

	report.PlanProblems = novelgate.PlanContractIssues(chapterNum, plan, others)

	// 大纲节点契约（既有判据，行为不变）：节点存在才判——「节点不存在」是尚未进入
	// 大纲的正常态（生成流程本身会建节点），预检不臆造缺项；节点存在则缺项照报。
	of, oerr := pm.ReadOutlines()
	if oerr != nil {
		if !os.IsNotExist(oerr) {
			return nil, fmt.Errorf("读取大纲失败：%w", oerr)
		}
		of = &types.OutlineFile{}
	}
	if of != nil {
		if node := findOutlineNodeByNum(of.Nodes, chapterNum); node != nil {
			for _, is := range novelgate.OutlineContractIssues(*node) {
				code := is.Code
				if !strings.HasPrefix(code, "outline_") {
					code = "outline_" + code
				}
				report.OutlineIssues = append(report.OutlineIssues, types.PlanProblem{
					Code: code, Severity: is.Severity, Message: is.Message, Evidence: is.Evidence,
				})
			}
		}
	}

	// 阻断判定：计划问题 + 大纲问题合并；S1/S2 阻断（与 types.PlanProblem 注释一致）。
	for _, p := range report.PlanProblems {
		if severityBlocking(p.Severity) {
			report.Blocking = true
		}
	}
	for _, p := range report.OutlineIssues {
		if severityBlocking(p.Severity) {
			report.Blocking = true
		}
	}
	report.Allowed = !report.Blocking
	report.Missing = planMissingLabels(report)
	return report, nil
}

// planFieldLabels 问题码 → 缺失项中文标签（前端「一键补计划」按这些词给入口）。
// 只收「字段缺失」类码：跨章重复（plan_event_duplicated）不是缺字段，故不进 Missing，
// 它由 PlanProblems 的消息 + planGateError 的明细直接点名。
var planFieldLabels = map[string]string{
	novelgate.PlanMissingCode:          "章节计划",
	types.PlanProblemMissingGoal:       "叙事目标",
	types.PlanProblemMissingKeyEvents:  "关键事件（≥2 条）",
	types.PlanProblemMissingConflict:   "冲突类型",
	types.PlanProblemMissingEnding:     "结尾类型",
	types.PlanProblemMissingEmotion:    "情绪基调",
	types.PlanProblemMissingCharacters: "角色焦点",
	"outline_title_empty":              "章节标题",
	"outline_summary_empty":            "章节摘要（谁·在哪·发生什么）",
	"outline_keypoints_empty":          "关键要点",
	"outline_emotion_empty":            "情感基调",
}

// planMissingLabels 汇总「缺什么」的中文标签（计划字段 + 大纲节点字段，去重、稳定序）。
func planMissingLabels(report *types.PlanGateReport) []string {
	if report == nil {
		return nil
	}
	out := make([]string, 0, 6)
	seen := map[string]bool{}
	add := func(code string) {
		label, ok := planFieldLabels[code]
		if !ok || seen[label] {
			return
		}
		seen[label] = true
		out = append(out, label)
	}
	for _, p := range report.PlanProblems {
		add(p.Code)
	}
	for _, p := range report.OutlineIssues {
		add(p.Code)
	}
	return out
}

// planGateError 把预检报告转成可执行的中文错误：缺什么、去哪补、怎么放行。
// 文案必须含「先补章节计划」与缺失项（前端/作者据此找到补计划的入口）。
func planGateError(report *types.PlanGateReport) error {
	if report == nil {
		return fmt.Errorf("写前预检未通过：先补章节计划后再生成")
	}
	var details []string
	for _, p := range report.PlanProblems {
		if severityBlocking(p.Severity) {
			details = append(details, p.Message)
		}
	}
	for _, p := range report.OutlineIssues {
		if severityBlocking(p.Severity) {
			details = append(details, p.Message)
		}
	}
	msg := fmt.Sprintf("第%d章写前预检未通过：先补章节计划", report.ChapterNum)
	if len(report.Missing) > 0 {
		msg += "（缺：" + strings.Join(report.Missing, "、") + "）"
	}
	msg += "。请到「创作页 → 章节计划」补齐本章计划（并补上大纲节点的标题/摘要），再重新生成；" +
		"确无计划也要生成时需显式开启覆盖。"
	if len(details) > 0 {
		msg += "明细：" + strings.Join(details, "；")
	}
	return fmt.Errorf("%s", msg)
}

// findOutlineNode 按谓词在大纲树中找第一个命中节点（AP1-06 收敛：递归骨架全包
// 只此一份， Children 递归——N13：分卷大纲的章节点挂在卷节点下，只扫顶层会让
// 分卷书的各取节点路径整体失明）。前序遍历：先本层后子层，与改前各份手写递归
// 的命中序一致。「认不认分支/按什么字段认」是各调用点的语义差异，进谓词显式
// 化，不在此处顺手统一。
func findOutlineNode(nodes []types.OutlineNode, pred func(*types.OutlineNode) bool) *types.OutlineNode {
	for i := range nodes {
		if pred(&nodes[i]) {
			return &nodes[i]
		}
		if found := findOutlineNode(nodes[i].Children, pred); found != nil {
			return found
		}
	}
	return nil
}

// findOutlineNodeByNum 按章号取主线大纲节点（分支章不参与主线意图注入；同名分支
// 节点在 ensureChapterNode 里才会建，此处只认 Branch==""——语义在谓词里）。
func findOutlineNodeByNum(nodes []types.OutlineNode, num int) *types.OutlineNode {
	return findOutlineNode(nodes, func(n *types.OutlineNode) bool {
		return n.OrderIndex == num && n.Branch == ""
	})
}

// trimPlanItems 去空白并丢弃空条目（渲染与判据共用同一口径）。
func trimPlanItems(items []string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		if s := strings.TrimSpace(it); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// buildChapterPlanSection 把本章计划渲染成紧凑中文区段（§7.3 槽位 chapter_plan）：
// 叙事目标 / 关键事件（逐条）/ 冲突类型 / 结尾类型 / 情绪基调 / 角色焦点。
// 无计划或六项全空 → 返回 ""（模板按空跳过，不渲染空区段）。
func buildChapterPlanSection(plan *types.ChapterPlan) string {
	if plan == nil {
		return ""
	}
	var body strings.Builder
	if v := strings.TrimSpace(plan.NarrativeGoal); v != "" {
		body.WriteString("叙事目标：" + v + "\n")
	}
	if events := trimPlanItems(plan.KeyEvents); len(events) > 0 {
		body.WriteString("关键事件（逐条必须落地，不得遗漏或跳过）：\n")
		for _, ev := range events {
			body.WriteString("- " + ev + "\n")
		}
	}
	if v := strings.TrimSpace(plan.ConflictType); v != "" {
		body.WriteString("冲突类型：" + v + "\n")
	}
	if v := strings.TrimSpace(plan.EndingType); v != "" {
		body.WriteString("结尾类型：" + v + "\n")
	}
	if v := strings.TrimSpace(plan.EmotionalTone); v != "" {
		body.WriteString("情绪基调：" + v + "\n")
	}
	if focus := trimPlanItems(plan.CharacterFocus); len(focus) > 0 {
		body.WriteString("角色焦点：" + strings.Join(focus, "、") + "\n")
	}
	if strings.TrimSpace(body.String()) == "" {
		return ""
	}
	return truncateBudget("## 本章计划（本章的意图契约，必须完成）\n"+body.String(), ctxChapterPlanBudget)
}

// buildOutlinePointsSection 渲染大纲节点的 KeyPoints 与 Emotion（§7.3 槽位
// outline_points——这两项此前零进入生成 prompt）。node 为 nil 或两者皆空 → 返回 ""。
func buildOutlinePointsSection(node *types.OutlineNode) string {
	if node == nil {
		return ""
	}
	var parts []string
	if pts := trimPlanItems(node.KeyPoints); len(pts) > 0 {
		parts = append(parts, "必须落地的要点：\n- "+strings.Join(pts, "\n- "))
	}
	if v := strings.TrimSpace(node.Emotion); v != "" {
		parts = append(parts, "情感基调："+v)
	}
	if len(parts) == 0 {
		return ""
	}
	return truncateBudget("## 本章大纲要点（细纲口径，逐条落地）\n"+strings.Join(parts, "\n"),
		ctxOutlinePointsBudget)
}

// buildStyleSection 书级文风档案（oh-story T6 风格档案协议对齐，上游
// style-resolution.md）：项目根 style.md，作者显式文风偏好——一句可执行的
// 偏好也有效；空白/纯标题/「待补充」不算；无文件不建占位、不猜。
// 协议口径：文风只裁决**表达维度**（句长/视角/标点/对话/修辞/收尾），
// 事件、事实、信息揭露边界仍以细纲为准——事实与表达分开裁决。
// styleSectionBody 抽正文（刀6 文风合并区段的偏好半边）；buildStyleSection 为
// 兼容壳（带标题完整区段，旧消费面）。
func styleSectionBody(pm *project.Manager) string {
	if pm == nil {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(pm.Dir, "style.md"))
	if err != nil {
		return "" // 无文件不建占位（上游口径）
	}
	var lines []string
	for _, line := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if t == "" || strings.HasPrefix(t, "#") {
			continue // 标题行不是偏好
		}
		if strings.Contains(t, "待补充") {
			continue
		}
		lines = append(lines, t)
	}
	if len(lines) == 0 {
		return ""
	}
	body := strings.Join(lines, "\n")
	if r := utf8.RuneCountInString(body); r > ctxStyleBudget {
		body = string([]rune(body)[:ctxStyleBudget]) + "……"
	}
	return body
}

// buildWorldviewSection 读世界观并拆分为「维度要点」正文，每维度截断
// ctxWorldviewDimLen、整体不超过 ctxWorldviewBudget。复用 pm.ReadWorldview
// （worldview.json 优先，旧 worldview.md 兜底）；旧 md 无 "## " 维度标题时
// 整块压缩为单条要点。读失败或全空时返回 ""。
func buildWorldviewSection(pm *project.Manager, relevanceHint string) string {
	if pm == nil {
		return ""
	}
	md, err := pm.ReadWorldview()
	if err != nil || strings.TrimSpace(md) == "" {
		return ""
	}
	var lines []string
	curTitle := ""
	var cur []string
	flush := func() {
		content := strings.TrimSpace(strings.Join(cur, " "))
		if curTitle != "" && content != "" {
			lines = append(lines, fmt.Sprintf("- %s：%s", curTitle, util.Truncate(content, ctxWorldviewDimLen)))
		}
		cur = cur[:0]
	}
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			curTitle = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		if curTitle != "" {
			cur = append(cur, line)
		}
	}
	flush()
	if len(lines) == 0 {
		// 旧 md 没有任何 "## " 维度标题：整块作为单条要点压缩注入
		return "- 世界观：" + util.Truncate(strings.TrimSpace(md), ctxWorldviewDimLen*4)
	}
	// 刀6 相关性：维度标题命中本章计划/大纲要点的条目优先保留（稳定排序——
	// 命中者前移、组内保持原序；预算内截断口径不变）。无 hint 或零命中=原序。
	if strings.TrimSpace(relevanceHint) != "" {
		scored := make([]string, len(lines))
		copy(scored, lines)
		sort.SliceStable(scored, func(i, j int) bool {
			return worldviewDimHitsHint(scored[i], relevanceHint) && !worldviewDimHitsHint(scored[j], relevanceHint)
		})
		lines = scored
	}
	return truncateBudget(strings.Join(lines, "\n"), ctxWorldviewBudget)
}

// worldviewDimHitsHint 维度条目标题是否命中相关性提示的任一关键词
// （hint 分词按空白/标点切 2+ 字词，标题子串匹配）。
func worldviewDimHitsHint(dimLine, hint string) bool {
	head := dimLine
	if idx := strings.Index(head, "："); idx > 0 {
		head = head[:idx]
	}
	// 中文无词边界（FieldsFunc 对整句只切出一个超长 token）：改 rune bigram 滑窗
	// ——hint 的任意连续两字子串命中维度标题即视为相关（宽匹配可接受：本函数
	// 只做排序不做过滤，误命中最多让无关维度排前）。
	hrs := []rune(hint)
	for i := 0; i+1 < len(hrs); i++ {
		bigram := string(hrs[i : i+2])
		if strings.Contains(head, bigram) {
			return true
		}
	}
	return false
}

// ── 状态演进解析（阶段八 8.2，NovelCrafter Progressions 同型）────────────────
//
// 注入按当前章取「锚点 ≤ 当前章」的最新状态：替换字段取章号最大者（同章取
// 列表靠后者，作者改主意以新条目为准）；增补按章号升序（稳定）拼接。章号
// ≤0 视为「取最新」（全部生效）——面板/分支等无章号语境的调用方用。

// progressionReplaceFields 可替换字段的渲染白名单（characterSummaryLine 实际
// 渲染的档案字段）；白名单外的 Field 一律按增补处理（诚实降级，不静默丢弃）。
var progressionReplaceFields = map[string]bool{
	"personality": true, "background": true, "motivation": true, "status": true,
}

// progressionAddition 单条增补的渲染单元。
type progressionAddition struct {
	Chapter int
	Text    string
}

// resolveProgressions 解析角色在 chapterNum 章生效的状态演进（纯函数、确定性）。
// 返回替换字段表与增补列表；无演进数据返回零值。
func resolveProgressions(ch types.Character, chapterNum int) (map[string]string, []progressionAddition) {
	if len(ch.Progressions) == 0 {
		return nil, nil
	}
	replacements := make(map[string]string)
	bestChapter := make(map[string]int)
	var additions []progressionAddition
	for _, p := range ch.Progressions {
		if p.Text == "" || p.Chapter <= 0 {
			continue
		}
		if chapterNum > 0 && p.Chapter > chapterNum {
			continue // 未来章节的演进对当前章不可见（回看旧章同理）
		}
		if p.Field != "" && progressionReplaceFields[p.Field] {
			if p.Chapter >= bestChapter[p.Field] {
				replacements[p.Field] = p.Text
				bestChapter[p.Field] = p.Chapter
			}
			continue
		}
		additions = append(additions, progressionAddition{Chapter: p.Chapter, Text: p.Text})
	}
	sort.SliceStable(additions, func(i, j int) bool { return additions[i].Chapter < additions[j].Chapter })
	return replacements, additions
}

// buildCharacterSummary 构建角色摘要字符串（用于注入章节生成 prompt）
// 格式：每角色一行「姓名：身份·性格·身份背景·目标·关系要点·状态」，
// 各字段按 rune 截断（性格上限 charPersonalityLen），整体不超过
// charSummaryBudget；无背景/目标/关系的角色保持原有精简形态。
// never 注入模式的角色全链不可见（阶段八 8.1）；状态演进按 chapterNum 投影
// （≤0=取最新，阶段八 8.2）。
func (a *writingState) buildCharacterSummary(pm *project.Manager, chapterNum int) string {
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil || len(cf.Characters) == 0 {
		return "（暂无角色设定）"
	}
	relDigest := buildRelationDigest(cf)
	lines := make([]string, 0, len(cf.Characters))
	for _, ch := range cf.Characters {
		if normalizeInjectMode(ch.InjectMode) == injectModeNever {
			continue
		}
		lines = append(lines, characterSummaryLine(ch, relDigest, chapterNum))
	}
	if len(lines) == 0 {
		return "（暂无角色设定）"
	}
	return truncateBudget(strings.Join(lines, "\n"), charSummaryBudget)
}

// characterSummaryLine 单角色摘要行（buildCharacterSummary 与焦点分级装配共用）。
// chapterNum > 0 时先投影状态演进（替换白名单字段、增补缀「第N章起」）。
func characterSummaryLine(ch types.Character, relDigest map[string]string, chapterNum int) string {
	replacements, additions := resolveProgressions(ch, chapterNum)
	eff := func(field, base string) string {
		if t, ok := replacements[field]; ok {
			return t
		}
		return base
	}
	roleLabel := characterRoleLabel(ch.RoleType)
	personality := util.Truncate(strings.TrimSpace(eff("personality", ch.Personality)), charPersonalityLen)
	line := fmt.Sprintf("- %s：%s·%s", ch.Name, roleLabel, personality)
	if bg := strings.TrimSpace(eff("background", ch.Background)); bg != "" {
		line += "·身份：" + util.Truncate(bg, charFieldLen)
	}
	if mo := strings.TrimSpace(eff("motivation", ch.Motivation)); mo != "" {
		line += "·目标：" + util.Truncate(mo, charFieldLen)
	}
	if rel := relDigest[ch.ID]; rel != "" {
		line += "·关系：" + rel
	}
	// v4.454 与主角的关系（项目级字段）注入名册：非空才渲染，旧数据零变化。
	if pr := strings.TrimSpace(ch.ProtagonistRelation); pr != "" {
		line += "·与主角：" + util.Truncate(pr, charFieldLen)
	}
	line += "·" + eff("status", ch.Status)
	// 增补演进缀行尾：告知模型这是「第N章起」的新事实（追加语义，非替换）
	for _, ad := range additions {
		line += fmt.Sprintf("·第%d章起：%s", ad.Chapter, util.Truncate(ad.Text, charFieldLen))
	}
	return line
}

// characterRoleLabel 角色定位中文名（未知类型原样回显）。
func characterRoleLabel(roleType string) string {
	label := map[string]string{
		"protagonist": "主角", "antagonist": "反派",
		"supporting": "配角", "minor": "次要",
	}[roleType]
	if label == "" {
		return roleType
	}
	return label
}

// charRosterBudget 焦点外名册区的预算（rune）——名册只做备查提示，不挤占档案。
const charRosterBudget = 800

// ── 实体卡触发注入（阶段八 8.1，docs/gaea-longterm-plan-2026-10.md §2）────────
//
// 注入模式四态（types.Character.InjectMode，项目级字段）：
//
//	"" / "auto"  检测注入——本章计划/大纲要点/剧情要求文本命中名字或别名才升全量档案行
//	"always"     恒注入——无条件升全量档案行（关键实体兜底）
//	"muted"      检测不注入——命中也不升档，仅名册备查
//	"never"      不渲染——全链不可见（名册也不进，AI 看不见这张卡）
//
// 检测是纯字符串匹配：中文无词边界，用子串包含——ST 的全词匹配默认开但对
// 中日文有害，业界共识=中文必须关（竞品调研 §5.1）；命中判定前先剔除排除词
// 出现处，使「命中串只是排除词子串」不再触发。

const (
	injectModeAlways = "always"
	injectModeMuted  = "muted"
	injectModeNever  = "never"
)

// normalizeInjectMode 注入模式合法值收敛（未知/空值回退默认检测注入）。
func normalizeInjectMode(mode string) string {
	switch mode {
	case injectModeAlways, injectModeMuted, injectModeNever:
		return mode
	default:
		return ""
	}
}

// stripExcludeHits 把 scan 中所有角色排除词的出现处替换为 NUL 占位符——
// 之后的名字/别名匹配不再命中这些位置（如排除「王爷」后，正文里的「王爷」
// 不再让「王」命中）。纯函数、确定性。
func stripExcludeHits(scan string, chars []types.Character) string {
	for _, ch := range chars {
		for _, e := range ch.ExcludeHits {
			if e = strings.TrimSpace(e); e != "" {
				scan = strings.ReplaceAll(scan, e, "\x00")
			}
		}
	}
	return scan
}

// characterDetected 名字或任一别名是否命中（已剔除排除词的）扫描文本。
// 子串包含即命中（中文无词边界）；空名/空别名跳过。
func characterDetected(cleanScan string, ch types.Character) bool {
	if ch.Name != "" && strings.Contains(cleanScan, ch.Name) {
		return true
	}
	for _, a := range ch.Aliases {
		if a = strings.TrimSpace(a); a != "" && strings.Contains(cleanScan, a) {
			return true
		}
	}
	return false
}

// buildChapterCastSection 按本章计划角色焦点 + 实体卡触发注入装配两级角色区段
// （用户实测教训：全量名册灌进第一章会让模型把所有角色一次性写登场）。全档区
// = 计划焦点 ∪ 文本命中（名字/别名，never 除外）∪ always 兜底 − muted；其余
// 角色降为紧凑名册——备查不登场。全档区与名册区双预算封顶
// （charSummaryBudget/charRosterBudget）。反空窗纪律：全档区为空时退回全量
// 名册（仅滤 never），绝不因此空窗。状态演进按 chapterNum 投影（≤0=取最新）。
func (a *writingState) buildChapterCastSection(pm *project.Manager, plan *types.ChapterPlan, scanText string, chapterNum int) string {
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil || len(cf.Characters) == 0 {
		return "（暂无角色设定）"
	}
	var focus []string
	if plan != nil {
		focus = trimPlanItems(plan.CharacterFocus)
	}
	cleanScan := stripExcludeHits(scanText, cf.Characters)
	relDigest := buildRelationDigest(cf)
	inFull := make(map[int]bool, len(cf.Characters))
	for i, ch := range cf.Characters {
		switch normalizeInjectMode(ch.InjectMode) {
		case injectModeNever:
			continue
		case injectModeAlways:
			inFull[i] = true
		case injectModeMuted:
			// 检测不注入：命中也不升档（名册备查行在后循环统一渲染）
		default:
			if focusMatches(focus, ch.Name) || characterDetected(cleanScan, ch) {
				inFull[i] = true
			}
		}
	}
	if len(inFull) == 0 {
		return a.buildCharacterSummary(pm, chapterNum)
	}
	var focusLines, rosterLines []string
	for i, ch := range cf.Characters {
		if inFull[i] {
			focusLines = append(focusLines, characterSummaryLine(ch, relDigest, chapterNum))
		} else if normalizeInjectMode(ch.InjectMode) != injectModeNever {
			rosterLines = append(rosterLines,
				fmt.Sprintf("- %s：%s·%s", ch.Name, characterRoleLabel(ch.RoleType), effStatusOf(ch, chapterNum)))
		}
	}
	var b strings.Builder
	b.WriteString("【本章出场角色（按本章计划与文本命中装配，剧情围绕他们展开）】\n")
	b.WriteString(truncateBudget(strings.Join(focusLines, "\n"), charSummaryBudget))
	if len(rosterLines) > 0 {
		b.WriteString("\n\n【其余名册（备查——本章不登场：不给戏份、不进对话，至多行文提名，后续章节再出场）】\n")
		b.WriteString(truncateBudget(strings.Join(rosterLines, "\n"), charRosterBudget))
	}
	return b.String()
}

// effStatusOf 名册紧凑行的状态字段：状态演进有 status 替换时取投影值（8.2）。
func effStatusOf(ch types.Character, chapterNum int) string {
	replacements, _ := resolveProgressions(ch, chapterNum)
	if t, ok := replacements["status"]; ok && t != "" {
		return t
	}
	return ch.Status
}

// focusMatches 焦点项与角色名宽容匹配（作者手写焦点可能带缀饰，如「林晚（主角）」）。
func focusMatches(focus []string, name string) bool {
	for _, f := range focus {
		if f == name || strings.Contains(name, f) || strings.Contains(f, name) {
			return true
		}
	}
	return false
}

// BranchCastEntry 分支选角会议的单条 cast：作者在向导里圈定的登场角色与主角关系。
type branchCastEntry struct {
	Name     string
	Relation string // 与主角（男主）关系：师妹/死敌/恋人…
	Note     string // 可选：一句话身份/来源标注（角色库卡摘要、新建说明）
}

// parseBranchCast 解析作者圈定的选角负载：对象数组 [{"name","relation","note"}]
// 为准，兼容纯字符串数组（只有名字）；空缺/坏 JSON 返回 nil（全量名册不设限，
// 绝不因载荷问题报错）。
func parseBranchCast(charactersJSON string) []branchCastEntry {
	s := strings.TrimSpace(charactersJSON)
	if s == "" {
		return nil
	}
	var objects []struct {
		Name     string `json:"name"`
		Relation string `json:"relation"`
		Note     string `json:"note"`
	}
	if err := json.Unmarshal([]byte(s), &objects); err == nil && len(objects) > 0 {
		out := make([]branchCastEntry, 0, len(objects))
		for _, o := range objects {
			if n := strings.TrimSpace(o.Name); n != "" {
				out = append(out, branchCastEntry{n, strings.TrimSpace(o.Relation), strings.TrimSpace(o.Note)})
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	var names []string
	if err := json.Unmarshal([]byte(s), &names); err != nil {
		return nil
	}
	out := make([]branchCastEntry, 0, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, branchCastEntry{Name: n})
		}
	}
	return out
}

// buildBranchCastSection 分支请求的选角区段：作者在向导里完成「选角会议」后
// 装配——项目名册命中者给全量档案行并缀主角关系；未命中（角色库挑入/简单
// 新建）按负载自带信息成行；未被圈定的名册角色不渲染（选角会议是权威名单）。
// 空负载退回全量名册（不设限语义）。状态演进按 chapterNum 投影（≤0=取最新）。
func (a *writingState) buildBranchCastSection(pm *project.Manager, cast []branchCastEntry, chapterNum int) string {
	if len(cast) == 0 {
		return a.buildCharacterSummary(pm, chapterNum)
	}
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil {
		cf = &types.CharacterFile{}
	}
	relDigest := buildRelationDigest(cf)
	lines := make([]string, 0, len(cast))
	for _, e := range cast {
		var hit *types.Character
		for i := range cf.Characters {
			if focusMatches([]string{e.Name}, cf.Characters[i].Name) {
				hit = &cf.Characters[i]
				break
			}
		}
		if hit != nil {
			line := characterSummaryLine(*hit, relDigest, chapterNum)
			if e.Relation != "" {
				line += "·与主角关系：" + e.Relation
			}
			lines = append(lines, line)
			continue
		}
		line := fmt.Sprintf("- %s：新增角色", e.Name)
		if e.Note != "" {
			line += "·" + util.Truncate(e.Note, charFieldLen)
		}
		if e.Relation != "" {
			line += "·与主角关系：" + e.Relation
		}
		lines = append(lines, line)
	}
	body := truncateBudget(strings.Join(lines, "\n"), charSummaryBudget)
	return "【本轮分支登场角色（作者圈定；关系为主角视角）】\n" + body
}

// buildRelationDigest 把 characters.json 的 relationships 折叠为
// 「角色ID → 关系要点」，只保留双方都能对照到项目角色的关系；
// 每角色最多 charMaxRelations 条。无关系数据时返回空 map。
func buildRelationDigest(cf *types.CharacterFile) map[string]string {
	if cf == nil || len(cf.Relationships) == 0 {
		return nil
	}
	nameByID := make(map[string]string, len(cf.Characters))
	for _, ch := range cf.Characters {
		nameByID[ch.ID] = ch.Name
	}
	relationLabel := map[string]string{
		"friend": "好友", "enemy": "敌对", "family": "亲人", "mentor": "师徒",
		"rival": "宿敌", "lover": "恋人", "member": "隶属", "leader": "统领",
	}
	rels := make(map[string][]string)
	for _, r := range cf.Relationships {
		fromName, toName := nameByID[r.FromID], nameByID[r.ToID]
		if fromName == "" || toName == "" {
			continue
		}
		label := relationLabel[r.RelationType]
		if label == "" {
			label = r.RelationType
		}
		item := fmt.Sprintf("与%s为%s", toName, label)
		if d := strings.TrimSpace(r.Description); d != "" {
			item += "（" + util.Truncate(d, charRelationDescLen) + "）"
		}
		rels[r.FromID] = append(rels[r.FromID], item)
	}
	digest := make(map[string]string, len(rels))
	for id, items := range rels {
		if len(items) > charMaxRelations {
			items = items[:charMaxRelations]
		}
		digest[id] = strings.Join(items, "、")
	}
	return digest
}
