package app

// create_chapter_context.go — 章节生成的上下文编译与写前门辅助族
// （P5 破防专项拆自 create_chapter_handler.go：预算常量/前文摘要窗口/增强区段
// 组装/计划门辅助与区段渲染/世界观与文风构建/角色摘要——纯函数为主，搬移零
// 功能变更）。

import (
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
	ctxForeshadowLineLen  = 100  // 单条伏笔描述截断
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
	ctxPrevWindowChapters = 10  // 只注入本章之前的最近 N 章
	ctxPrevChapterLen     = 180 // 窗口内单章摘要截断（rune）
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
	return strings.Join(parts, "\n\n")
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

// buildChapterContextSectionsWithin 同 buildChapterContextSections，但预算可调：
// 刀1 的章节计划/大纲要点区段已经从 ctxBudgetTotal 里先占额度，剩余额度经本函数
// 传给伏笔/文风/世界观，保证「计划 + 大纲 + 增强区段」合计不超过同一个 ctxBudgetTotal。
func buildChapterContextSectionsWithin(pm *project.Manager, currentChapter, budget int, relevanceHint, digestInstr string) string {
	if budget <= 0 {
		return "" // 额度耗尽：不追加任何区段（宁缺毋滥，不写半截截断文本）
	}
	sections := make([]string, 0, 3)
	if body := buildForeshadowSection(pm, currentChapter); body != "" {
		sections = append(sections, "## 未回收伏笔（创作约束）\n"+
			"以下伏笔已埋设尚未回收，写作时不得与之矛盾；可自然推进，不要强行揭穿：\n"+body)
	}
	// 刀6 文风去重合并：style.md（显式偏好）与成稿学习指令（刀5 digest）此前
	// 是两个独立区段两个标题（§1.8「文风两处」的延续）——合并为单一区段，
	// 偏好优先、学习跟随。
	if body := joinStyleSections(styleSectionBody(pm), digestInstr); body != "" {
		sections = append(sections, body)
	}
	if body := buildWorldviewSection(pm, relevanceHint); body != "" {
		sections = append(sections, "## 世界观要点\n"+body)
	}
	return joinWithBudget(budget, sections...)
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
		if planSeverityBlocking(p.Severity) {
			report.Blocking = true
		}
	}
	for _, p := range report.OutlineIssues {
		if planSeverityBlocking(p.Severity) {
			report.Blocking = true
		}
	}
	report.Allowed = !report.Blocking
	report.Missing = planMissingLabels(report)
	return report, nil
}

// planSeverityBlocking S1/S2 视为阻断（S3/S4 仅提示）。
func planSeverityBlocking(severity string) bool {
	return severity == "S1" || severity == "S2"
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
		if planSeverityBlocking(p.Severity) {
			details = append(details, p.Message)
		}
	}
	for _, p := range report.OutlineIssues {
		if planSeverityBlocking(p.Severity) {
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

// findOutlineNodeByNum 按章号取主线大纲节点（分支章不参与主线意图注入；同名分支
// 节点在 ensureChapterNode 里才会建，此处只认 Branch==""）。递归到 Children
// （N13）：分卷大纲的章节点挂在卷节点下，只扫顶层会让分卷书的章计划/意图注入
// 整体失明。
func findOutlineNodeByNum(nodes []types.OutlineNode, num int) *types.OutlineNode {
	for i := range nodes {
		if nodes[i].OrderIndex == num && nodes[i].Branch == "" {
			return &nodes[i]
		}
		if found := findOutlineNodeByNum(nodes[i].Children, num); found != nil {
			return found
		}
	}
	return nil
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

// joinWithBudget 用空行拼接区段；合计超过 budget（rune）时对每段截断至
// budget/段数（扣除拼接符开销），保证新增上下文不会撑爆生成 prompt。
func joinWithBudget(budget int, sections ...string) string {
	if len(sections) == 0 {
		return ""
	}
	joined := strings.Join(sections, "\n\n")
	if runeLen(joined) > budget {
		joiner := 2 * (len(sections) - 1) // "\n\n" 拼接开销
		per := (budget - joiner) / len(sections)
		for i := range sections {
			sections[i] = truncateBudget(sections[i], per)
		}
		joined = strings.Join(sections, "\n\n")
	}
	return joined
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

// buildCharacterSummary 构建角色摘要字符串（用于注入章节生成 prompt）
// 格式：每角色一行「姓名：身份·性格·身份背景·目标·关系要点·状态」，
// 各字段按 rune 截断（性格上限 charPersonalityLen），整体不超过
// charSummaryBudget；无背景/目标/关系的角色保持原有精简形态。
func (a *writingState) buildCharacterSummary(pm *project.Manager) string {
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil || len(cf.Characters) == 0 {
		return "（暂无角色设定）"
	}
	relDigest := buildRelationDigest(cf)
	lines := make([]string, 0, len(cf.Characters))
	for _, ch := range cf.Characters {
		lines = append(lines, characterSummaryLine(ch, relDigest))
	}
	return truncateBudget(strings.Join(lines, "\n"), charSummaryBudget)
}

// characterSummaryLine 单角色摘要行（buildCharacterSummary 与焦点分级装配共用）。
func characterSummaryLine(ch types.Character, relDigest map[string]string) string {
	roleLabel := characterRoleLabel(ch.RoleType)
	personality := util.Truncate(strings.TrimSpace(ch.Personality), charPersonalityLen)
	line := fmt.Sprintf("- %s：%s·%s", ch.Name, roleLabel, personality)
	if bg := strings.TrimSpace(ch.Background); bg != "" {
		line += "·身份：" + util.Truncate(bg, charFieldLen)
	}
	if mo := strings.TrimSpace(ch.Motivation); mo != "" {
		line += "·目标：" + util.Truncate(mo, charFieldLen)
	}
	if rel := relDigest[ch.ID]; rel != "" {
		line += "·关系：" + rel
	}
	line += "·" + ch.Status
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

// buildChapterCastSection 按本章计划的角色焦点装配两级角色区段（用户实测教训：
// 全量名册灌进第一章会让模型把所有角色一次性写登场）。焦点角色给全量档案行
// （本章登场、剧情围绕他们展开）；其余角色降为紧凑名册——备查不登场。计划
// 缺失或焦点没有命中任何角色时退回全量名册（行为与旧口径一致，绝不因此空窗）。
func (a *writingState) buildChapterCastSection(pm *project.Manager, plan *types.ChapterPlan) string {
	if plan == nil || len(trimPlanItems(plan.CharacterFocus)) == 0 {
		return a.buildCharacterSummary(pm)
	}
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil || len(cf.Characters) == 0 {
		return "（暂无角色设定）"
	}
	focus := trimPlanItems(plan.CharacterFocus)
	relDigest := buildRelationDigest(cf)
	inFocus := make(map[int]bool)
	var focusLines, rosterLines []string
	for i, ch := range cf.Characters {
		if focusMatches(focus, ch.Name) {
			inFocus[i] = true
			focusLines = append(focusLines, characterSummaryLine(ch, relDigest))
		}
	}
	if len(focusLines) == 0 {
		return a.buildCharacterSummary(pm)
	}
	for i, ch := range cf.Characters {
		if !inFocus[i] {
			rosterLines = append(rosterLines,
				fmt.Sprintf("- %s：%s·%s", ch.Name, characterRoleLabel(ch.RoleType), ch.Status))
		}
	}
	var b strings.Builder
	b.WriteString("【本章出场角色（按计划角色焦点，剧情围绕他们展开）】\n")
	b.WriteString(strings.Join(focusLines, "\n"))
	if len(rosterLines) > 0 {
		b.WriteString("\n\n【其余名册（备查——本章不登场：不给戏份、不进对话，至多行文提名，后续章节再出场）】\n")
		b.WriteString(truncateBudget(strings.Join(rosterLines, "\n"), charRosterBudget))
	}
	return b.String()
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
