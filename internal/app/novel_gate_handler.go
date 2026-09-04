package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// ── 章节生成门（GenerationGate）编排 ────────────────────────────
//
// 把既有的四路审阅能力（情节分析 / 章节质量审查 / 一致性深检 / AI 味检测）
// 串成一个可手动触发的单章门禁，返回一份合并报告。
//
// 降级原则（诚实降级，绝不编造 AI 结果）：
//   - 无项目 / 章节正文读不出来 → 直接返回 error（门禁本身失败）；
//   - 某一路失败 → 只缺省该字段（对应键为 nil），其余路照常返回，不中断整体；
//   - `review` 与 `analysis` 依赖子代理，子代理为 nil 或缺省时同样降级为 nil。

// RunChapterGate 单章「生成门」：合并 分析+审阅+一致性深检+AI味 四路结果为一报告。
// 返回 map 键：chapterNum / analysis / review / consistency / aiTaste。
// 任意一路失败都不会中断，只把对应键置为 nil（诚实降级）。
func (a *writingState) RunChapterGate(chapterNum int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	content, err := pm.ReadChapterAsStitch(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取第%d章正文失败: %w", chapterNum, err)
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	title, prevHint := gateReviewContext(pm, chapterNum)

	result := map[string]interface{}{
		"chapterNum":  chapterNum,
		"analysis":    nil,
		"review":      nil,
		"consistency": nil,
		"aiTaste":     nil,
	}

	// ── 1. 情节分析（analysisAgent 缺省时降级）──
	if a.analysisAgent == nil {
		slog.Warn("生成门: analysisAgent 未初始化，analysis 路缺省")
	} else if res, aerr := a.analysisAgent.Analyze(ctx, chapterNum, content); aerr != nil {
		slog.Warn("生成门: 情节分析失败，analysis 路缺省", "chapter", chapterNum, "error", aerr)
	} else if res == nil {
		slog.Warn("生成门: 情节分析返回空，analysis 路缺省", "chapter", chapterNum)
	} else {
		result["analysis"] = map[string]interface{}{
			"hook":             res.Hook,
			"foreshadows":      res.Foreshadows,
			"character_states": res.CharacterStates,
			"key_events":       res.KeyEvents,
			"quality_score":    res.QualityScore,
			"improvement_tips": res.ImprovementTips,
		}
	}

	// ── 2. 章节质量审查（chapterAgent 缺省时降级）──
	if a.chapterAgent == nil {
		slog.Warn("生成门: chapterAgent 未初始化，review 路缺省")
	} else if res, rerr := a.chapterAgent.ReviewChapter(ctx, content, title, prevHint); rerr != nil {
		slog.Warn("生成门: 章节审查失败，review 路缺省", "chapter", chapterNum, "error", rerr)
	} else if res == nil {
		slog.Warn("生成门: 章节审查返回空，review 路缺省", "chapter", chapterNum)
	} else {
		result["review"] = map[string]interface{}{
			"score":       res.Score,
			"strengths":   res.Strengths,
			"weaknesses":  res.Weaknesses,
			"revise_plan": res.RevisePlan,
		}
	}

	// ── 3. 一致性深检（窗口 20 章，失败缺省）──
	if cons, cerr := a.CheckConsistencyDeep(20); cerr != nil {
		slog.Warn("生成门: 一致性深检失败，consistency 路缺省", "chapter", chapterNum, "error", cerr)
	} else if cons == nil {
		slog.Warn("生成门: 一致性深检返回空，consistency 路缺省", "chapter", chapterNum)
	} else {
		result["consistency"] = map[string]interface{}{
			"issues":       cons["issues"],
			"total_issues": cons["total_issues"],
			"summary":      cons["summary"],
		}
	}

	// ── 4. AI 味检测（确定性打分，不调 LLM）──
	if ts, terr := novelstyle.ScoreTextNoRef(content); terr != nil {
		slog.Warn("生成门: AI 味检测失败，aiTaste 路缺省", "chapter", chapterNum, "error", terr)
	} else if ts == nil {
		slog.Warn("生成门: AI 味检测返回空，aiTaste 路缺省", "chapter", chapterNum)
	} else {
		result["aiTaste"] = map[string]interface{}{
			"score":  ts.Score,
			"issues": ts.Issues,
		}
	}

	return result, nil
}

// gateReviewContext 解析审查输入：title 取自本章摘要（Title/Summary）/大纲标题，
// prevHint 取自上一章摘要尾部。
func gateReviewContext(pm *project.Manager, chapterNum int) (title, prevHint string) {
	if s, err := pm.ReadChapterSummary(chapterNum); err == nil && s != nil {
		title = s.Title
		if title == "" {
			title = util.Truncate(s.Summary, 80)
		}
	}
	if title == "" {
		title = gateOutlineTitle(pm, chapterNum)
	}
	if title == "" {
		title = fmt.Sprintf("第%d章", chapterNum)
	}

	if chapterNum > 1 {
		if ps, err := pm.ReadChapterSummary(chapterNum - 1); err == nil && ps != nil {
			prevHint = util.Truncate(ps.Summary, 200)
		}
	}
	return title, prevHint
}

// gateOutlineTitle 从大纲中按 ChapterFile 匹配章节标题（递归搜索，未命中返回空串）。
func gateOutlineTitle(pm *project.Manager, chapterNum int) string {
	of, err := pm.ReadOutlines()
	if err != nil || of == nil {
		return ""
	}
	target := fmt.Sprintf("%03d.md", chapterNum)
	for _, n := range of.Nodes {
		if t := gateOutlineNodeTitle(n, target); t != "" {
			return t
		}
	}
	return ""
}

func gateOutlineNodeTitle(n types.OutlineNode, target string) string {
	if n.ChapterFile == target {
		if n.Title != "" {
			return n.Title
		}
		return n.Summary
	}
	for _, c := range n.Children {
		if t := gateOutlineNodeTitle(c, target); t != "" {
			return t
		}
	}
	return ""
}
