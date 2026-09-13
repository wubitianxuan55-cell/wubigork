package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// ── 伏笔分层注入（t1-P1，docs/distill/01-foreshadow-spec.md §4）───────
//
// 替代旧「未回收伏笔（创作约束）」单层注入：按计划回收章把伏笔分成
// 必须回收（L1）/ 超期（L2）/ 近期参考（L3）/ 本章计划埋入（L4）四层，
// 另加 L5 兜底层承接存量无计划条目（详见 types.ForeshadowLayer 注释）。
// 分层判定唯一入口 = types.ForeshadowLayerOf；本文件只做排序与渲染。
//
// 预算/条数口径（spec §4.3）：条数封顶 ctxForeshadowMaxItems、正文封顶
// ctxForeshadowBudget，按 L1→L2→L3→L4→L5 优先级顺序消耗，超限即停；
// L2 ≤3 条、L3 ≤5 条（L1/L4 不限，受总预算约束）。空层不输出标题。
//
// 全部容错：读失败 / 无数据静默返回 ""，绝不因增强失败中断生成主链路。

// 分层单条内容截断（spec §4.2：L1 内容 100 / L2 内容 80 / L4 内容 80）。
const (
	fsLayerMustLen    = 100
	fsLayerShortLen   = 80
	fsLayerOverdueMax = 3 // L2 条数上限（spec §4.1）
	fsLayerNearMax    = 5 // L3 条数上限
)

// foreshadowEntry 渲染中间结构：条目 + 所属层 + 当前章号。
type foreshadowEntry struct {
	f     types.Foreshadow
	layer types.ForeshadowLayer
	cur   int
}

// buildForeshadowSection 组装分层伏笔调度区段正文。currentChapter = 正在
// 写的章号（≤0 时所有回收层退化为 L5 兜底，等价旧单层注入口径）。
func buildForeshadowSection(pm *project.Manager, currentChapter int) string {
	if pm == nil {
		return ""
	}
	ff, err := pm.ReadForeshadows()
	if err != nil || ff == nil || len(ff.Items) == 0 {
		return ""
	}

	// 分层（唯一判定入口），跳过 None。
	var entries []foreshadowEntry
	for _, f := range ff.Items {
		l := types.ForeshadowLayerOf(f, currentChapter, types.ForeshadowLookaheadDefault)
		if l == types.ForeshadowLayerNone {
			continue
		}
		entries = append(entries, foreshadowEntry{f: f, layer: l, cur: currentChapter})
	}
	if len(entries) == 0 {
		return ""
	}

	layerBodies := make([]string, 0, 5)
	// 预算含区段头与调度规则行：整区（含标题）不超过 ctxForeshadowBudget。
	preamble := "## 伏笔调度（分层约束）\n" +
		"调度规则：标注「本章必须回收」的务必在本章内完成回收；「已超期」的若情节允许应优先安排回收；" +
		"「请勿在本章回收」的不得提前揭晓或回收；「本章计划埋入」的应自然埋入（暗示而非说明）；" +
		"其余已埋入伏笔为既定事实，写作时不得与之矛盾，不要强行提前揭穿：\n"
	usedRunes, usedItems := runeLen(preamble), 0
	appendLayer := func(layer types.ForeshadowLayer, maxItems int, render func(foreshadowEntry) string) {
		if usedItems >= ctxForeshadowMaxItems {
			return
		}
		group := entriesOfLayer(entries, layer)
		if len(group) == 0 {
			return
		}
		sortForeshadowGroup(group, layer)
		header := layerHeader(layer)
		var b strings.Builder
		b.WriteString(header)
		entryRunes, written := 0, 0
		for _, e := range group {
			if maxItems > 0 && written >= maxItems {
				break
			}
			if usedItems >= ctxForeshadowMaxItems {
				break
			}
			line := render(e)
			if usedRunes+entryRunes+runeLen(line)+1 > ctxForeshadowBudget {
				break
			}
			b.WriteString(line)
			b.WriteString("\n")
			entryRunes += runeLen(line) + 1
			usedItems++
			written++
		}
		if written > 0 {
			usedRunes += entryRunes + runeLen(header)
			layerBodies = append(layerBodies, strings.TrimSuffix(b.String(), "\n"))
		}
	}

	appendLayer(types.ForeshadowLayerMust, 0, renderMustEntry)
	appendLayer(types.ForeshadowLayerOverdue, fsLayerOverdueMax, renderOverdueEntry)
	appendLayer(types.ForeshadowLayerNear, fsLayerNearMax, renderNearEntry)
	appendLayer(types.ForeshadowLayerPlant, 0, renderPlantEntry)
	appendLayer(types.ForeshadowLayerNoPlan, 0, renderNoPlanEntry)

	if len(layerBodies) == 0 {
		return ""
	}
	return preamble + strings.Join(layerBodies, "\n")
}

// entriesOfLayer 取某层全部条目。
func entriesOfLayer(entries []foreshadowEntry, layer types.ForeshadowLayer) []foreshadowEntry {
	var out []foreshadowEntry
	for _, e := range entries {
		if e.layer == layer {
			out = append(out, e)
		}
	}
	return out
}

// sortForeshadowGroup 层内确定性排序：L1/L4 重要性降序（缺省 0.5），
// L2/L3 计划章升序（最早到期先出），L5 埋入章升序。
func sortForeshadowGroup(group []foreshadowEntry, layer types.ForeshadowLayer) {
	importance := func(f types.Foreshadow) float64 {
		if f.Importance > 0 {
			return f.Importance
		}
		return 0.5
	}
	switch layer {
	case types.ForeshadowLayerMust, types.ForeshadowLayerPlant:
		sort.SliceStable(group, func(i, j int) bool {
			return importance(group[i].f) > importance(group[j].f)
		})
	case types.ForeshadowLayerOverdue, types.ForeshadowLayerNear:
		sort.SliceStable(group, func(i, j int) bool {
			return types.ChapterNumOf(group[i].f.TargetResolveIn) < types.ChapterNumOf(group[j].f.TargetResolveIn)
		})
	case types.ForeshadowLayerNoPlan:
		sort.SliceStable(group, func(i, j int) bool {
			return types.ChapterNumOf(group[i].f.PlantedIn) < types.ChapterNumOf(group[j].f.PlantedIn)
		})
	}
}

// layerHeader 层标题（spec §4.2 模板；空层由调用方保证不输出）。
func layerHeader(layer types.ForeshadowLayer) string {
	switch layer {
	case types.ForeshadowLayerMust:
		return "【🎯 本章必须回收的伏笔 - 请务必在本章完成回收】\n"
	case types.ForeshadowLayerOverdue:
		return "【⚠️ 超期待回收伏笔 - 请尽快回收】\n"
	case types.ForeshadowLayerNear:
		return "【📋 近期待回收伏笔（仅供参考，请勿在本章回收）】\n" +
			"⚠️ 以下伏笔尚未到回收时机，本章请勿提前回收，仅作为剧情背景了解\n"
	case types.ForeshadowLayerPlant:
		return "【✨ 本章计划埋入伏笔】\n"
	case types.ForeshadowLayerNoPlan:
		return "【📌 已埋入未规划回收章的伏笔（背景约束）】\n"
	}
	return ""
}

// displayTitle 展示标题：Title 优先；无 Title 时回退截断描述（isFallback=true，
// 调用方不再重复输出内容行，避免同文两现）。
func displayTitle(f types.Foreshadow, fallbackLen int) (title string, isFallback bool) {
	if t := strings.TrimSpace(f.Title); t != "" {
		return t, false
	}
	d := strings.TrimSpace(f.Description)
	if d == "" {
		d = f.ID
	}
	return util.Truncate(d, fallbackLen), true
}

// renderMustEntry L1：ID+标题 / 埋入章 / 内容（≤100）/ 回收提示（仅非空）。
func renderMustEntry(e foreshadowEntry) string {
	f := e.f
	title, isFallback := displayTitle(f, fsLayerMustLen)
	var b strings.Builder
	fmt.Fprintf(&b, "- ID:%s | %s%s", f.ID, title, longTermTag(f))
	if !isFallback {
		fmt.Fprintf(&b, "\n  埋入章节：第%d章", types.ChapterNumOf(f.PlantedIn))
		if d := strings.TrimSpace(f.Description); d != "" {
			fmt.Fprintf(&b, "\n  伏笔内容：%s", util.Truncate(d, fsLayerMustLen))
		}
	}
	if n := strings.TrimSpace(f.ResolutionNotes); n != "" {
		fmt.Fprintf(&b, "\n  回收提示：%s", util.Truncate(n, fsLayerMustLen))
	}
	return b.String()
}

// renderOverdueEntry L2：超期 N 章标注 + 原计划章 + 内容（≤80）。
func renderOverdueEntry(e foreshadowEntry) string {
	f := e.f
	title, isFallback := displayTitle(f, fsLayerShortLen)
	target := types.ChapterNumOf(f.TargetResolveIn)
	var b strings.Builder
	fmt.Fprintf(&b, "- ID:%s | %s [已超期%d章]%s", f.ID, title, e.cur-target, longTermTag(f))
	if !isFallback {
		fmt.Fprintf(&b, "\n  埋入章节：第%d章，原计划第%d章回收",
			types.ChapterNumOf(f.PlantedIn), target)
		if d := strings.TrimSpace(f.Description); d != "" {
			fmt.Fprintf(&b, "\n  伏笔内容：%s", util.Truncate(d, fsLayerShortLen))
		}
	}
	return b.String()
}

// renderNearEntry L3：单行参考（标题 + 计划章 + 剩余章数），无内容行。
func renderNearEntry(e foreshadowEntry) string {
	f := e.f
	title, _ := displayTitle(f, 50)
	target := types.ChapterNumOf(f.TargetResolveIn)
	return fmt.Sprintf("- %s（计划第%d章回收，还有%d章）", title, target, target-e.cur)
}

// renderPlantEntry L4：标题 + 内容（≤80）+ 埋入提示（仅非空）。
func renderPlantEntry(e foreshadowEntry) string {
	f := e.f
	title, _ := displayTitle(f, 30)
	var b strings.Builder
	b.WriteString("- " + title)
	if d := strings.TrimSpace(f.Description); d != "" {
		fmt.Fprintf(&b, "\n  伏笔内容：%s", util.Truncate(d, fsLayerShortLen))
	}
	if h := strings.TrimSpace(f.HintText); h != "" {
		fmt.Fprintf(&b, "\n  埋入提示：%s", util.Truncate(h, fsLayerShortLen))
	}
	return b.String()
}

// renderNoPlanEntry L5（gaea 扩展兜底）：旧行为等价——ID + 内容 + 埋入章
// （埋入章未知时省略章号，不输出「第0章」噪声）。
func renderNoPlanEntry(e foreshadowEntry) string {
	f := e.f
	d := strings.TrimSpace(f.Description)
	if d == "" {
		d = f.ID
	}
	chapter := ""
	if n := types.ChapterNumOf(f.PlantedIn); n > 0 {
		chapter = fmt.Sprintf("（埋入第%d章）", n)
	}
	return fmt.Sprintf("- ID:%s | %s%s%s", f.ID, util.Truncate(d, fsLayerMustLen), longTermTag(f), chapter)
}

// longTermTag 长线标记。
func longTermTag(f types.Foreshadow) string {
	if f.IsLongTerm {
		return "（长线）"
	}
	return ""
}
