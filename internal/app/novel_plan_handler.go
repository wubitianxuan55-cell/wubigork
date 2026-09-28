package app

// ── 章节计划闭环：计划生产与偏差回写（规格 docs/gaea-longform-novel-system-2026-09.md §7.2/§7.4）──
//
// 本文件是「章节计划闭环」刀1 的线C 足迹：把 types.ChapterPlan 从零消费变成
// 可读（Get）、可审批落盘（Save）、可由 AI 提案（Propose）、写后可回写偏差
// （Deviation）、生成前可硬闸（Precheck 委托线B）。
//
// 分工（§7.5）：
//   - 线A（internal/project）负责 plans.json 与 deviation 的落盘读写；
//   - 线B（internal/novelgate + create_chapter_handler.go）负责 PlanContractIssues
//     判据与 planPrecheck 硬闸；
//   - 本文件只做编排：取输入 → 调判据 → 调读写 → 如实回报，不重复实现判据、
//     不自行决定落盘格式（避免两份真相）。
//
// 纪律：AI 只**提案**（Propose 不落盘），落盘一律经作者审批（Save）；模型不可用
// 或返回不合法时如实报错，**不做规则兜底伪造计划**（假计划比没有计划更危险）。

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/novelgate"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// ── 阈值（口径=runes，不是字节）────────────────────────────────

const (
	// planEventHitRatio 关键事件命中的覆盖率阈值（偏差回写的命中判定）。
	//
	// 判定两级（见 planEventHit）：
	//  ① 归一化后整体包含 → 命中（严格）；
	//  ② 事件 ≥ planEventMinBigramRunes 时按 2-gram 覆盖率判定，覆盖率 ≥ 本阈值
	//     视为命中——模型写正文时会在事件中间加「了/的/便」等虚词（「林晚夺得玄铁剑」
	//     → 「林晚夺得了玄铁剑」），整体包含会漏判，而 2-gram 覆盖率仍为 1.0。
	//
	// 0.6 的取舍：低于 0.5 会把「只提到半句」也算达成（漏报偏差，回写失去意义）；
	// 高于 0.8 会把虚词插入较多的正常改写误判为未达成（误报偏差，作者会失去信任）。
	planEventHitRatio = 0.6

	// planEventMinBigramRunes 启用 2-gram 覆盖率判定的最短事件长度（rune）。
	// 短事件（如「下雨」）本身无冗余，只做整体包含判定，避免误命中。
	planEventMinBigramRunes = 4

	// planRawEchoRunes 解析失败时回显的原始返回前缀长度（rune）——便于作者
	// 一眼看出模型到底吐了什么，同时不把整段脏文本灌进错误信息。
	planRawEchoRunes = 200

	// planExistingEventsCap Propose 输入段里「已有章节关键事件」的条数上限：
	// 计划去重的输入需要完整，但长篇 200 章 × 4 条会撑爆 prompt，故设上限并显式
	// 告知模型列表被截断（Save 侧仍由 PlanContractIssues 做全量确定性去重兜底）。
	planExistingEventsCap = 80

	// planSuggestionEventsCap NextSuggestion 里回显的已用事件条数上限。
	planSuggestionEventsCap = 8
)

// NovelChapterPlanGet 读取该章计划。
//
// 未制定 = 正常态：返回 (nil, nil)，前端据此显示「未制定」并给「生成草案」入口
// （不报错——「没计划」是创作的正常起点，不是故障）。
func (a *writingState) NovelChapterPlanGet(chapterNum int) (*types.ChapterPlan, error) {
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章节号无效（%d）：计划按章号索引，需为正整数", chapterNum)
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	plan, ok, err := pm.ReadChapterPlan(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取第 %d 章计划失败: %w", chapterNum, err)
	}
	if !ok || plan == nil {
		return nil, nil
	}
	return plan, nil
}

// NovelChapterPlanSave 作者审批落盘：解析 → 确定性校验（齐备性 + 跨章去重）→
// 整表读-改-写。
//
// 存在 S1/S2 即拒绝并点名问题（含重复事件原文）；S3/S4 仅提示不阻断（与
// PlanGateReport 的 Blocking 口径一致）。写失败如实返回，不做「内存成功」假象。
func (a *writingState) NovelChapterPlanSave(chapterNum int, planJSON string) error {
	if chapterNum <= 0 {
		return fmt.Errorf("章节号无效（%d）：计划按章号索引，需为正整数", chapterNum)
	}
	if strings.TrimSpace(planJSON) == "" {
		return fmt.Errorf("计划内容为空：请先填写计划（剧情概要 / 关键事件 / 叙事目标等）再保存")
	}
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}

	var plan types.ChapterPlan
	if err := json.Unmarshal([]byte(planJSON), &plan); err != nil {
		return fmt.Errorf("计划 JSON 解析失败: %w", err)
	}
	if plan.SubIndex <= 0 {
		plan.SubIndex = chapterNum // 前端通常只送七字段，序号按本章号补齐
	}

	pf, err := pm.ReadChapterPlans()
	if err != nil {
		return fmt.Errorf("读取计划表失败: %w", err)
	}
	if pf == nil {
		pf = &types.ChapterPlanFile{Version: 1}
	}
	if pf.Plans == nil {
		pf.Plans = map[string]types.ChapterPlan{}
	}
	if pf.Version == 0 {
		pf.Version = 1
	}

	// others = 除本章外的全部计划（跨章去重的确定性输入，含已写章节的既定计划）。
	others := planOthers(pf, chapterNum)
	if problems := novelgate.PlanContractIssues(chapterNum, &plan, others); len(problems) > 0 {
		if msg := planBlockingMessage(problems); msg != "" {
			return fmt.Errorf("%s", msg)
		}
	}

	key := strconv.Itoa(chapterNum)
	pf.Plans[key] = plan // 整表读-改-写：其它章原样保留
	if err := pm.WriteChapterPlans(pf); err != nil {
		return fmt.Errorf("写入计划表失败: %w", err)
	}
	return nil
}

// NovelChapterPlanPropose AI 计划草案（**不落盘**）。
//
// 输入 = 本章大纲节点（Title/Summary/KeyPoints/Emotion/Characters）+ OutlineFile.StoryThread
// + 前文摘要窗口（复用 buildPrevSummaryWindow 口径）+ 其它章已有关键事件 + 该章
// analysis-v2 载荷摘要（若存在）。输出七字段 JSON。
//
// 生成后先本地跑一遍 PlanContractIssues：不通过即如实报错（**不返回半成品**，
// 也绝不规则兜底伪造计划）；模型不可用/超时同样如实上抛。
func (a *writingState) NovelChapterPlanPropose(chapterNum int) (*types.ChapterPlan, error) {
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章节号无效（%d）：计划按章号索引，需为正整数", chapterNum)
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未就绪：请先在「模型中心」配置可用引擎，再生成计划草案")
	}

	of, err := pm.ReadOutlines()
	if err != nil {
		return nil, fmt.Errorf("读取大纲失败: %w", err)
	}
	node := findOutlineNodeByChapter(of.Nodes, chapterNum)
	if node == nil {
		return nil, fmt.Errorf("未找到第 %d 章的大纲节点：请先在大纲页为本章补节点（标题 / 摘要 / 关键要点）", chapterNum)
	}

	tmpl := a.eng.Get("chapter-plan")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 chapter-plan 模板文件（prompts/chapter-plan.json）")
	}

	pf, err := pm.ReadChapterPlans()
	if err != nil {
		return nil, fmt.Errorf("读取计划表失败: %w", err)
	}
	if pf == nil {
		pf = &types.ChapterPlanFile{Version: 1, Plans: map[string]types.ChapterPlan{}}
	}
	others := planOthers(pf, chapterNum)

	nodeJSON, err := json.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("序列化本章大纲节点失败: %w", err)
	}

	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"story_thread":        strings.TrimSpace(of.StoryThread),
		"chapter_outline":     string(nodeJSON),
		"prev_summary":        buildPrevSummaryWindow(of.Nodes, chapterNum, prevSummaryResolver(pm)),
		"existing_key_events": planExistingKeyEventsText(others),
		"chapter_analysis":    planAnalysisDigest(pm, chapterNum),
	})
	systemPrompt := tmpl.BuildSystemPrompt("")

	eng, model, _ := a.routeModel("novel")
	// 计划草案是短 JSON（数百 token），不套 play 域 max_output_tokens 护栏：
	// 护栏语义是「play 内容钳制」，与结构性产物无关。
	opts := ai.ChatSimpleOptions{EngineID: eng, Feature: "novel"}
	reply, err := a.client.ChatSimpleStreamWithOptions(a.ctx, model, systemPrompt, userPrompt, opts)
	if err != nil {
		return nil, fmt.Errorf("计划草案生成失败（模型不可用或超时）：%w", err)
	}

	plan, err := parseChapterPlanReply(reply)
	if err != nil {
		return nil, err
	}
	if plan.SubIndex <= 0 {
		plan.SubIndex = chapterNum
	}
	if strings.TrimSpace(plan.Title) == "" {
		plan.Title = node.Title // 标题非七字段之一：模型缺省时用大纲节点标题补齐
	}

	plan.Title = strings.TrimSpace(plan.Title)
	if problems := novelgate.PlanContractIssues(chapterNum, plan, others); len(problems) > 0 {
		if msg := planBlockingMessage(problems); msg != "" {
			return nil, fmt.Errorf("AI 计划草案未通过本地校验，已丢弃（不落盘、不返回半成品）：%s", msg)
		}
	}
	return plan, nil
}

// NovelChapterPlanDeviation 计划 vs 实际（写后回写；确定性优先，零 LLM）。
//
// 四类判据：未达成关键事件 / 结尾类型不符 / 情绪漂移 / 跨章重复事件。
// 无计划 → {HasPlan:false}；有计划未分析 → {Analyzed:false}；两者都是**正常态**，
// 不报错。计算完成后尽力落盘 WritePlanDeviation：失败只记日志，不影响返回
// （偏差报告是写后增强情报，不能反过来阻断作者）。
func (a *writingState) NovelChapterPlanDeviation(chapterNum int) (*types.PlanDeviation, error) {
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章节号无效（%d）：计划按章号索引，需为正整数", chapterNum)
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	plan, ok, err := pm.ReadChapterPlan(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取第 %d 章计划失败: %w", chapterNum, err)
	}
	if !ok || plan == nil {
		return &types.PlanDeviation{
			ChapterNum:     chapterNum,
			HasPlan:        false,
			Summary:        "尚未制定本章计划：先立计划（作者审批落盘），生成后才有偏差可回写",
			NextSuggestion: planNextSuggestion(nil, &types.PlanDeviation{ChapterNum: chapterNum}, nil, nil),
		}, nil
	}

	dev := &types.PlanDeviation{ChapterNum: chapterNum, HasPlan: true}

	// 跨章去重输入：计划表读取失败按「无其它计划」处理（去重是增强判据，
	// 不因表读取异常把整份偏差报告变成错误）。
	pf, err := pm.ReadChapterPlans()
	if err != nil {
		slog.Warn("计划偏差：读取计划表失败，跳过跨章去重", "chapter", chapterNum, "error", err)
		pf = nil
	}
	others := planOthers(pf, chapterNum)

	// ④ 跨章重复事件：复用 PlanContractIssues 的去重判据口径（同一条判据只有一份实现）。
	dev.DuplicateEvents = planDuplicateEvents(chapterNum, plan, others)

	item := planAnalysisItem(pm, chapterNum)
	if item == nil {
		dev.Analyzed = false
		dev.PlannedEnding = plan.EndingType
		dev.PlannedEmotion = plan.EmotionalTone
		dev.Summary = "本章尚未分析：先运行「分析本章」，再回写计划偏差"
		dev.NextSuggestion = planNextSuggestion(plan, dev, others, nil)
		return dev, nil // 未分析是正常态：不落盘半份报告，等分析完成再算
	}
	dev.Analyzed = true

	// ① 未达成关键事件：计划事件在分析载荷（plot_points 文本 + summary）里命中判定。
	haystack := planAnalysisHaystack(&item.Result)
	for _, ev := range plan.KeyEvents {
		if strings.TrimSpace(ev) == "" {
			continue
		}
		if !planEventHit(ev, haystack) {
			dev.MissingEvents = append(dev.MissingEvents, ev)
		}
	}

	// ② 结尾类型不符：计划 EndingType vs 分析载荷「结尾钩子」类型映射。
	//    分析载荷缺结尾钩子（字段缺省）→ 无法判定，**不报漂移**。
	dev.PlannedEnding = plan.EndingType
	if actual, known := planActualEnding(&item.Result); known {
		dev.ActualEnding = actual
		if strings.TrimSpace(plan.EndingType) != "" && !planSameLabel(plan.EndingType, actual) {
			dev.EndingMismatch = true
		}
	}

	// ③ 情绪漂移：计划 EmotionalTone vs 分析载荷 emotional_arc.primary_emotion。
	//    缺省 → 无法判定，不报漂移。
	dev.PlannedEmotion = plan.EmotionalTone
	dev.ActualEmotion = strings.TrimSpace(item.Result.EmotionalArc.PrimaryEmotion)
	if strings.TrimSpace(plan.EmotionalTone) != "" && dev.ActualEmotion != "" &&
		!planSameLabel(plan.EmotionalTone, dev.ActualEmotion) {
		dev.EmotionDrift = true
	}

	dev.Summary = planDeviationSummary(plan, dev)
	dev.NextSuggestion = planNextSuggestion(plan, dev, others, &item.Result)

	// 尽力落盘：失败只记日志（不影响返回，也不改写已算出的偏差）。
	if err := pm.WritePlanDeviation(dev); err != nil {
		slog.Warn("计划偏差落盘失败（不影响返回）", "chapter", chapterNum, "error", err)
	}
	return dev, nil
}

// NovelChapterGatePrecheck 生成前硬闸预检：纯委托线B（判据唯一来源在
// writingState.planPrecheck，本方法不做任何二次判定或包装）。
func (a *writingState) NovelChapterGatePrecheck(chapterNum int) (*types.PlanGateReport, error) {
	return a.planPrecheck(chapterNum)
}

// ── 输入组装（纯函数，确定性）──────────────────────────────────

// planOthers 汇总「除本章外」的全部计划，按章号升序返回。
//
// 升序（而非 map 遍历序）保证：跨章去重问题单的顺序稳定，前端展示与测试断言
// 不随 map 遍历抖动。
func planOthers(pf *types.ChapterPlanFile, chapterNum int) []types.ChapterPlan {
	if pf == nil || len(pf.Plans) == 0 {
		return nil
	}
	type entry struct {
		num int
		key string
	}
	entries := make([]entry, 0, len(pf.Plans))
	for k := range pf.Plans {
		n, err := strconv.Atoi(strings.TrimSpace(k))
		if err != nil || n == chapterNum {
			continue // 非章号键（脏数据）不参与去重，也不阻断读取
		}
		entries = append(entries, entry{num: n, key: k})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].num < entries[j].num })
	out := make([]types.ChapterPlan, 0, len(entries))
	for _, e := range entries {
		out = append(out, pf.Plans[e.key])
	}
	return out
}

// planExistingKeyEventsText 渲染「已有章节关键事件」输入段（Propose 模板槽位）。
// 无列表返回 ""（模板空槽位不渲染空区段）；超上限截断并显式告知模型。
func planExistingKeyEventsText(others []types.ChapterPlan) string {
	var lines []string
	truncated := 0
	for _, p := range others {
		for _, ev := range p.KeyEvents {
			ev = strings.TrimSpace(ev)
			if ev == "" {
				continue
			}
			if len(lines) >= planExistingEventsCap {
				truncated++
				continue
			}
			lines = append(lines, "- "+ev)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	body := strings.Join(lines, "\n")
	if truncated > 0 {
		body += fmt.Sprintf("\n（列表过长，另有 %d 条未列出：仍不得与之重复）", truncated)
	}
	return body
}

// planAnalysisDigest 渲染该章 analysis-v2 载荷摘要（Propose 模板槽位，容错）：
// summary + plot_points + suggestions。无该章分析返回 ""。
func planAnalysisDigest(pm *project.Manager, chapterNum int) string {
	item := planAnalysisItem(pm, chapterNum)
	if item == nil {
		return ""
	}
	r := &item.Result
	var blocks []string
	if s := strings.TrimSpace(r.Summary); s != "" {
		blocks = append(blocks, "摘要："+s)
	}
	if pts := planJoinCapped(planPlotPointTexts(r), 12); pts != "" {
		blocks = append(blocks, "情节推进点：\n"+pts)
	}
	if sug := planJoinCapped(r.Suggestions, 6); sug != "" {
		blocks = append(blocks, "修改建议：\n"+sug)
	}
	if len(blocks) == 0 {
		return ""
	}
	return strings.Join(blocks, "\n")
}

// planPlotPointTexts 取非空情节推进点文本（保序）。
func planPlotPointTexts(r *types.AnalysisResultV2) []string {
	out := make([]string, 0, len(r.PlotPoints))
	for _, p := range r.PlotPoints {
		if c := strings.TrimSpace(p.Content); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// planJoinCapped 把字符串列表渲染成 "- x" 行（上限 n 条）。
func planJoinCapped(items []string, n int) string {
	var lines []string
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		if len(lines) >= n {
			break
		}
		lines = append(lines, "- "+it)
	}
	return strings.Join(lines, "\n")
}

// findOutlineNodeByChapter 按章号找主线大纲节点（Branch=="" 为主线；与
// ensureChapterNode 的定号口径同源）。递归进 Children（卷/章树）。
func findOutlineNodeByChapter(nodes []types.OutlineNode, chapterNum int) *types.OutlineNode {
	for i := range nodes {
		if nodes[i].OrderIndex == chapterNum && nodes[i].Branch == "" {
			return &nodes[i]
		}
		if got := findOutlineNodeByChapter(nodes[i].Children, chapterNum); got != nil {
			return got
		}
	}
	return nil
}

// planAnalysisItem 读取该章 V2 分析载荷；无该章条目或读取失败返回 nil（正常态，
// 读取失败只记日志——分析载荷缺失不该把偏差查询变成错误）。
func planAnalysisItem(pm *project.Manager, chapterNum int) *types.ChapterAnalysisResult {
	af, err := pm.ReadAnalysisV2File()
	if err != nil {
		slog.Warn("读取 analysis-v2 失败", "chapter", chapterNum, "error", err)
		return nil
	}
	if af == nil {
		return nil
	}
	for i := range af.Items {
		if af.Items[i].ChapterNum == chapterNum {
			return &af.Items[i]
		}
	}
	return nil
}

// ── 模型返回解析 ──────────────────────────────────────────────

// parseChapterPlanReply 解析模型返回的七字段计划 JSON。
// 解析失败如实报错并回显原始返回前 planRawEchoRunes 字（作者据此判断是模型
// 跑偏还是引擎故障）——绝不吞掉原始返回后返回空计划。
func parseChapterPlanReply(reply string) (*types.ChapterPlan, error) {
	raw := strings.TrimSpace(reply)
	if raw == "" {
		return nil, fmt.Errorf("计划草案生成失败：模型返回为空（请检查引擎是否可用后重试）")
	}
	var plan types.ChapterPlan
	if err := json.Unmarshal([]byte(util.ExtractJSON(raw)), &plan); err != nil {
		return nil, fmt.Errorf("计划草案 JSON 解析失败: %w（原始返回前 %d 字：%s）",
			err, planRawEchoRunes, planRawEcho(raw))
	}
	return &plan, nil
}

// planRawEcho 取原始返回前 n 个 rune（超出加省略号）用于错误回显。
func planRawEcho(s string) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= planRawEchoRunes {
		return s
	}
	return string([]rune(s)[:planRawEchoRunes]) + "…"
}

// ── 判据编排 ──────────────────────────────────────────────────

// planBlockingMessage 把 S1/S2 问题单渲染成中文拒绝理由；无阻断问题返回 ""。
// 跨章重复事件额外点名（问题单里 Evidence 承载重复事件原文）。
func planBlockingMessage(problems []types.PlanProblem) string {
	var blocking []types.PlanProblem
	var dups []string
	for _, p := range problems {
		if !planBlockingSeverity(p.Severity) {
			continue // S3/S4 仅提示，不阻断落盘
		}
		blocking = append(blocking, p)
		if p.Code == types.PlanProblemEventDuplicated {
			if ev := strings.TrimSpace(p.Evidence); ev != "" {
				dups = append(dups, ev)
			} else if m := strings.TrimSpace(p.Message); m != "" {
				dups = append(dups, m)
			}
		}
	}
	if len(blocking) == 0 {
		return ""
	}
	parts := make([]string, 0, len(blocking))
	for _, p := range blocking {
		line := fmt.Sprintf("[%s] %s", p.Code, strings.TrimSpace(p.Message))
		// 重复事件的 Evidence 与末尾「重复事件：…」点名重复，这里不重复贴。
		if ev := strings.TrimSpace(p.Evidence); ev != "" && p.Code != types.PlanProblemEventDuplicated {
			line += "（" + ev + "）"
		}
		parts = append(parts, line)
	}
	msg := "计划未通过校验，已拒绝落盘：" + strings.Join(parts, "；")
	if len(dups) > 0 {
		msg += "。重复事件：" + strings.Join(planDedupStrings(dups), "、")
	}
	return msg
}

// planBlockingSeverity S1/S2 视为阻断（与 types.PlanGateReport.Blocking 口径一致）。
func planBlockingSeverity(sev string) bool {
	s := strings.ToUpper(strings.TrimSpace(sev))
	return s == "S1" || s == "S2"
}

// planDuplicateEvents 复用 PlanContractIssues 的去重判据口径，只取重复事件问题，
// 产出重复事件文本列表（Evidence 优先，缺省回退 Message）。
func planDuplicateEvents(chapterNum int, plan *types.ChapterPlan, others []types.ChapterPlan) []string {
	problems := novelgate.PlanContractIssues(chapterNum, plan, others)
	var dups []string
	for _, p := range problems {
		if p.Code != types.PlanProblemEventDuplicated {
			continue
		}
		if ev := strings.TrimSpace(p.Evidence); ev != "" {
			dups = append(dups, ev)
			continue
		}
		if m := strings.TrimSpace(p.Message); m != "" {
			dups = append(dups, m)
		}
	}
	return planDedupStrings(dups)
}

// planDedupStrings 去重并保序（重复事件常按事件粒度重复点名）。
func planDedupStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// ── 偏差判据（确定性，纯函数）──────────────────────────────────

// planAnalysisHaystack 偏差命中判定的文本池：分析载荷的 plot_points 文本 +
// summary（规格 §7.4 的判据输入口径）。归一化在 planEventHit 内做。
func planAnalysisHaystack(r *types.AnalysisResultV2) string {
	var parts []string
	for _, p := range r.PlotPoints {
		if c := strings.TrimSpace(p.Content); c != "" {
			parts = append(parts, c)
		}
		if im := strings.TrimSpace(p.Impact); im != "" {
			parts = append(parts, im)
		}
	}
	if s := strings.TrimSpace(r.Summary); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n")
}

// planEventHit 计划关键事件的命中判定（归一化包含 + 2-gram 覆盖率，阈值见
// planEventHitRatio / planEventMinBigramRunes）。
func planEventHit(event, haystack string) bool {
	ev := planNormalize(event)
	hay := planNormalize(haystack)
	if ev == "" || hay == "" {
		return false
	}
	if strings.Contains(hay, ev) {
		return true
	}
	if utf8.RuneCountInString(ev) < planEventMinBigramRunes {
		return false
	}
	grams := planBigrams(ev)
	if len(grams) == 0 {
		return false
	}
	hit := 0
	for _, g := range grams {
		if strings.Contains(hay, g) {
			hit++
		}
	}
	return float64(hit)/float64(len(grams)) >= planEventHitRatio
}

// planBigrams 相邻二元组（rune 粒度），保序不去重——重复二元组本身就是更强的
// 命中证据（「哈哈哈」的 hh 出现两次，命中一次不该算全中）。
func planBigrams(s string) []string {
	r := []rune(s)
	if len(r) < 2 {
		return nil
	}
	out := make([]string, 0, len(r)-1)
	for i := 0; i+1 < len(r); i++ {
		out = append(out, string(r[i:i+2]))
	}
	return out
}

// planNormalize 事件/文本归一化：去空白（含全角空格与各类换行）→ 全角 ASCII
// 转半角 → 转小写（规格 §7.4 的判据归一化口径）。
func planNormalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '\u3000', '\u00a0':
			continue
		}
		if r >= 0xFF01 && r <= 0xFF5E { // 全角 ！..～ → 半角
			r -= 0xFEE0
		}
		b.WriteRune(toLowerRune(r))
	}
	return b.String()
}

// toLowerRune ASCII 大写转小写；中文等非 ASCII 原样（避免 unicode.ToLower 对
// 全角字符的额外映射影响判据可预期性）。
func toLowerRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

// planEndingHookMap 分析载荷「结尾钩子」类型 → 计划结尾类型枚举（types 五值）
// 的映射：Hook.Type 取值域是 悬念|情感|冲突|认知（analysis_v2.go:41）。
var planEndingHookMap = map[string]string{
	"悬念": string(types.EndingSuspense),
	"情感": string(types.EndingEmotional),
	"冲突": string(types.EndingEscalation),
	"认知": string(types.EndingTwist),
}

// planActualEnding 从分析载荷取「实际结尾类型」。
// 判定来源：hooks 中 position=结尾 的 hook 类型（分析侧唯一表征结尾形态的字段），
// 按其类型映射到计划结尾枚举；模型直接给出枚举值时原样采用。
// 无结尾钩子 / 类型为空 → (,"" false) = 无法判定（不报漂移）。
func planActualEnding(r *types.AnalysisResultV2) (string, bool) {
	for _, h := range r.Hooks {
		if strings.TrimSpace(h.Position) != "结尾" {
			continue
		}
		t := strings.TrimSpace(h.Type)
		if t == "" {
			continue
		}
		if mapped, ok := planEndingHookMap[t]; ok {
			return mapped, true
		}
		return t, true
	}
	return "", false
}

// planSameLabel 两个标签是否一致：归一化后相等或互相包含（「紧张递进」与
// 「紧张」视为同一走向的两种粒度，不判漂移——情绪标签本就是粗粒度描述）。
func planSameLabel(a, b string) bool {
	na, nb := planNormalize(a), planNormalize(b)
	if na == "" || nb == "" {
		return true // 缺省=无法判定，调用方已各自判空；这里保守不算不符
	}
	return na == nb || strings.Contains(na, nb) || strings.Contains(nb, na)
}

// planDeviationSummary 一句话中文偏差概要（示例口径见规格 §7.4 与类型注释）。
func planDeviationSummary(plan *types.ChapterPlan, dev *types.PlanDeviation) string {
	var parts []string
	total := 0
	for _, ev := range plan.KeyEvents {
		if strings.TrimSpace(ev) != "" {
			total++
		}
	}
	if total > 0 {
		parts = append(parts, fmt.Sprintf("%d/%d 关键事件达成", total-len(dev.MissingEvents), total))
	}
	if dev.EndingMismatch {
		parts = append(parts, fmt.Sprintf("结尾类型不符（计划：%s / 实际：%s）", dev.PlannedEnding, dev.ActualEnding))
	}
	if dev.EmotionDrift {
		parts = append(parts, fmt.Sprintf("情绪漂移（计划：%s / 实际：%s）", dev.PlannedEmotion, dev.ActualEmotion))
	}
	if len(dev.DuplicateEvents) > 0 {
		parts = append(parts, "与其它章重复的事件："+strings.Join(dev.DuplicateEvents, "、"))
	}
	if len(parts) == 0 {
		return "关键事件全部达成，结尾与情绪均与计划一致"
	}
	return strings.Join(parts, "；")
}

// planNextSuggestion 下一章计划建议（纯文本，规则生成，不调模型；仅建议不落盘）。
//
// 输入：本章偏差 + 其它章已有关键事件；输出可执行的中文清单，帮作者把「写完
// 这一章得到的情报」带回「下一章怎么写」的决策（规格 §1.5 的缺口）。
func planNextSuggestion(plan *types.ChapterPlan, dev *types.PlanDeviation, others []types.ChapterPlan, result *types.AnalysisResultV2) string {
	nextNum := dev.ChapterNum + 1
	var lines []string
	add := func(s string) {
		lines = append(lines, fmt.Sprintf("%d. %s", len(lines)+1, s))
	}

	if plan == nil {
		add(fmt.Sprintf("第 %d 章尚未制定计划：先在大纲页补节点摘要与关键要点，再点「生成计划草案」并审批落盘（缺计划会被生成前硬闸拦下）", dev.ChapterNum))
		return "下一章计划建议：\n" + strings.Join(lines, "\n")
	}

	if dev.Analyzed {
		if len(dev.MissingEvents) > 0 {
			add("本章未达成的关键事件（" + strings.Join(dev.MissingEvents, "、") + "）：要么在第 " +
				strconv.Itoa(nextNum) + " 章开头补上，要么显式改写主线放弃它——不要沉默地丢弃")
		} else {
			add(fmt.Sprintf("本章 %d 条关键事件已全部落地：第 %d 章可以推进新进展，不要回炉复述", len(plan.KeyEvents), nextNum))
		}
		if dev.EndingMismatch {
			add("本章实际以「" + dev.ActualEnding + "」收尾（计划为「" + dev.PlannedEnding +
				"」）：第 " + strconv.Itoa(nextNum) + " 章须承接这个实际结尾，而不是按原计划的开场接续")
		}
		if dev.EmotionDrift {
			add("情绪实际落到「" + dev.ActualEmotion + "」（计划「" + dev.PlannedEmotion +
				"」）：第 " + strconv.Itoa(nextNum) + " 章的情绪基调建议从「" + dev.ActualEmotion + "」接续，避免无过渡的情绪跳变")
		}
	} else {
		add("本章尚未分析：先运行「分析本章」拿到 plot_points 与情绪/结尾判据，再据此定第 " + strconv.Itoa(nextNum) + " 章计划")
	}

	if len(dev.DuplicateEvents) > 0 {
		add("以下事件已在本项目用过，第 " + strconv.Itoa(nextNum) + " 章关键事件不得重复：" + strings.Join(dev.DuplicateEvents, "、"))
	}

	if result != nil {
		switch strings.TrimSpace(result.Pacing) {
		case "slow":
			add("本章节奏偏慢（pacing=slow）：第 " + strconv.Itoa(nextNum) + " 章计划建议把冲突提前到开场，减少铺垫段落")
		case "fast":
			add("本章节奏偏快（pacing=fast）：第 " + strconv.Itoa(nextNum) + " 章计划建议给关键转折留出场景，避免一路狂奔")
		}
	}

	if used := planRecentKeyEvents(others); used != "" {
		add("已有关键事件（跨章不得重复，节选）：\n" + used)
	}
	if plan.NarrativeGoal != "" {
		add("本章叙事目标「" + plan.NarrativeGoal + "」：第 " + strconv.Itoa(nextNum) +
			" 章计划应写清它推进了故事主线的哪一步")
	}
	return "下一章计划建议：\n" + strings.Join(lines, "\n")
}

// planRecentKeyEvents 渲染其它章已有关键事件节选（按章号升序，条数上限
// planSuggestionEventsCap）。
func planRecentKeyEvents(others []types.ChapterPlan) string {
	var items []string
	for _, p := range others {
		for _, ev := range p.KeyEvents {
			ev = strings.TrimSpace(ev)
			if ev == "" {
				continue
			}
			if len(items) >= planSuggestionEventsCap {
				return planJoinCapped(items, planSuggestionEventsCap)
			}
			items = append(items, ev)
		}
	}
	return planJoinCapped(items, planSuggestionEventsCap)
}
