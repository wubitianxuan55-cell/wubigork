package app

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gaea/gaea/internal/types"
)

// ── 分析 ────────────────────────────────────────────────────

// AnalyzeChapter 分析指定章节
func (a *writingState) AnalyzeChapter(chapterNum int) (map[string]interface{}, error) {
	if a.analysisAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	content, err := pm.ReadChapter(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取章节失败: %w", err)
	}
	result, err := a.analysisAgent.Analyze(a.ctx, chapterNum, content)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"hook":            result.Hook,
		"conflict":        result.Conflict,
		"emotionCurve":    result.EmotionCurve,
		"keyEvents":       result.KeyEvents,
		"sceneRhythm":     result.SceneRhythm,
		"qualityScore":    result.QualityScore,
		"improvementTips": result.ImprovementTips,
		"foreshadows":     result.Foreshadows,
		"characterStates": result.CharacterStates,
	}, nil
}

// NovelChapterAnalysisV2 读取该章 V2 分析（t7 前端接线：analysis-v2.json 该章
// 条目直连，types.ChapterAnalysisResult 透传——PromptTemplateDetail 回
// prompt.Template 直连先例；顶层 snake_case 如实透传，前端视图镜像声明）。
// 无该章条目 error（面板空态引导「先分析」）；分析落盘由 AnalyzeChapter
// 的 agent 链路完成（本方法只读）。
func (a *writingState) NovelChapterAnalysisV2(chapterNum int) (types.ChapterAnalysisResult, error) {
	if a.analysisAgent == nil {
		return types.ChapterAnalysisResult{}, fmt.Errorf("请先打开项目")
	}
	pm := a.getPM()
	if pm == nil {
		return types.ChapterAnalysisResult{}, fmt.Errorf("请先打开项目")
	}
	af, err := pm.ReadAnalysisV2File()
	if err != nil {
		return types.ChapterAnalysisResult{}, fmt.Errorf("读取分析结果失败: %w", err)
	}
	for i := range af.Items {
		if af.Items[i].ChapterNum == chapterNum {
			return af.Items[i], nil
		}
	}
	return types.ChapterAnalysisResult{}, fmt.Errorf("第 %d 章尚未分析（先运行「分析本章」）", chapterNum)
}

// foreshadowUrgencyView 单条目的运行时紧急度投影（spec §2.1 ForeshadowUrgency，
// A5：运行时算不落库；D7：阈值以后端为准前端不自算）。
// 已回收/已废弃无回收压力 → 返回 nil（wire 缺省该键）。
func foreshadowUrgencyView(f types.Foreshadow, currentChapter int) *types.ForeshadowUrgency {
	if types.IsResolvedStatus(f.Status) || f.Status == types.ForeshadowAbandoned {
		return nil
	}
	u := types.ForeshadowUrgency{
		Level:             types.UrgencyLevel(f, currentChapter),
		RemainingChapters: types.ChapterNumOf(f.TargetResolveIn) - currentChapter,
		ResolveStatus:     types.ClassifyResolve(f, currentChapter),
		MustResolve:       types.ClassifyResolve(f, currentChapter) == types.ResolveMustNow,
	}
	if u.RemainingChapters < 0 {
		u.OverdueChapters = -u.RemainingChapters
	}
	return &u
}

// foreshadowUrgencyCurrentChapter 紧急度投影的当前章口径：调用方未指定时按已写章节数。
func (a *writingState) foreshadowUrgencyCurrentChapter(currentChapter int) int {
	if currentChapter > 0 {
		return currentChapter
	}
	return countWrittenChapters(a.getPM())
}

// GetForeshadows 获取伏笔列表
func (a *writingState) GetForeshadows() map[string]interface{} {
	pm := a.getPM()
	if pm == nil {
		return nil
	}
	ff, err := pm.ReadForeshadows()
	if err != nil {
		slog.Warn("读取伏笔文件失败", "error", err)
		return nil
	}
	// 紧急度投影（嵌入展平：条目既有键零变化，仅追加 urgency 键；不落库，
	// 前端写回载荷由 SaveForeshadows 的 types.Foreshadow 反序列化自然丢弃）。
	cur := a.foreshadowUrgencyCurrentChapter(0)
	items := make([]map[string]interface{}, 0, len(ff.Items))
	for i := range ff.Items {
		it := ff.Items[i]
		raw := map[string]interface{}{}
		if b, err := json.Marshal(it); err == nil {
			_ = json.Unmarshal(b, &raw)
		}
		if len(raw) == 0 {
			continue
		}
		if u := foreshadowUrgencyView(it, cur); u != nil {
			raw["urgency"] = u
		}
		items = append(items, raw)
	}
	return map[string]interface{}{
		"items":          items,
		"currentChapter": cur,
	}
}

// GetLastForeshadowSync 最近一轮章节分析的伏笔同步结果（t1-P4：SyncResult 上
// 绑定面，跳过原因可见不静默——D3）。尚未执行过分析时显式报错。
func (a *writingState) GetLastForeshadowSync() (map[string]interface{}, error) {
	if a.analysisAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	res, ok := a.analysisAgent.LastSync()
	if !ok {
		return nil, fmt.Errorf("本次会话尚未执行过章节分析，暂无伏笔同步记录")
	}
	return map[string]interface{}{
		"plantedCount":        res.PlantedCount,
		"resolvedCount":       res.ResolvedCount,
		"createdCount":        res.CreatedCount,
		"updatedIds":          res.UpdatedIDs,
		"createdIds":          res.CreatedIDs,
		"matchedByContent":    res.MatchedByContent,
		"skippedResolveCount": res.SkippedResolveCount,
		"skippedReasons":      res.SkippedReasons,
		"errors":              res.Errors,
	}, nil
}

// ReviewBook AI 全书审稿
func (a *writingState) ReviewBook() (map[string]interface{}, error) {
	if a.analysisAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	result, err := a.analysisAgent.ReviewBook(a.ctx)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"letter":         result.Letter,
		"totalScore":     result.TotalScore,
		"scores":         result.Scores,
		"peaks":          result.Peaks,
		"valleys":        result.Valleys,
		"arcCompletions": result.ArcCompletions,
	}, nil
}

// GetBookData 获取全书聚合数据（用于前端统计画布，不调 AI）
func (a *writingState) GetBookData() map[string]interface{} {
	if a.analysisAgent == nil {
		return nil
	}
	chapters, charChapters, wv, foreshadows, err := a.analysisAgent.AggregateBookData()
	if err != nil {
		slog.Warn("聚合全书数据失败", "error", err)
		return nil
	}
	return map[string]interface{}{
		"chapters":     chapters,
		"charChapters": charChapters,
		"worldview":    wv,
		"foreshadows":  foreshadows,
	}
}
