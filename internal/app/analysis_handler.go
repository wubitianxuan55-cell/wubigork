package app

import (
	"fmt"
	"log/slog"
)

// ── 分析 ────────────────────────────────────────────────────

// AnalyzeChapter 分析指定章节
func (a *App) AnalyzeChapter(chapterNum int) (map[string]interface{}, error) {
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

// GetForeshadows 获取伏笔列表
func (a *App) GetForeshadows() map[string]interface{} {
	pm := a.getPM()
	if pm == nil {
		return nil
	}
	ff, err := pm.ReadForeshadows()
	if err != nil {
		slog.Warn("读取伏笔文件失败", "error", err)
		return nil
	}
	return map[string]interface{}{
		"items": ff.Items,
	}
}

// ReviewBook AI 全书审稿
func (a *App) ReviewBook() (map[string]interface{}, error) {
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
func (a *App) GetBookData() map[string]interface{} {
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
