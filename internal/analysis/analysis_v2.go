package analysis

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/types"
)

// ── 分析代理 V2 化（t4-C1，docs/distill/04-plot-analysis.md §8.3）────────
//
// Analyze 的一次 LLM 调用现产出 types.AnalysisResultV2（9 维 + 三维评分 +
// 建议联动 + keyword 锚点），落盘 analysis-v2.json（按章号 upsert），
// 并派生旧 wire 形状（AnalysisResult）供 AnalyzeChapter 绑定零破坏返回。
// 服务端权威项：overall 一律重算（模型不可自定），三维钳位 [1,10]，
// 建议条数按 SuggestionCountForOverall 联动上限裁剪（不代拟、不 fabricate）。

// normalizeAnalysisV2 服务端归一：评分钳制 + overall 重算 + 建议条数联动。
// 返回值即归一后的同一指针（就地改写）。
func normalizeAnalysisV2(v2 *types.AnalysisResultV2) *types.AnalysisResultV2 {
	if v2 == nil {
		return nil
	}
	v2.Scores.Pacing = clampScore1to10(v2.Scores.Pacing)
	v2.Scores.Engagement = clampScore1to10(v2.Scores.Engagement)
	v2.Scores.Coherence = clampScore1to10(v2.Scores.Coherence)
	// overall 权威重算（C1：系统按三维均值算，模型值丢弃）
	v2.Scores.Overall = types.ComputeOverallScore(v2.Scores.Pacing, v2.Scores.Engagement, v2.Scores.Coherence)
	// 建议条数与 overall 硬联动（t4 §8.2 C1）：只裁上限，不足不代拟
	_, maxSug := types.SuggestionCountForOverall(v2.Scores.Overall)
	if len(v2.Suggestions) > maxSug {
		v2.Suggestions = v2.Suggestions[:maxSug]
	}
	return v2
}

// clampScore1to10 评分钳制到 [1,10]（一位小数由 ComputeOverallScore 统一舍入，
// 单维保留模型给的一位小数原值）。
func clampScore1to10(v float64) float64 {
	if v < 1.0 {
		return 1.0
	}
	if v > 10.0 {
		return 10.0
	}
	return v
}

// persistAnalysisV2 落盘 analysis-v2.json（按章号 upsert）。容错注入：
// 落盘失败只记日志，绝不中断分析主链路。
func (a *Agent) persistAnalysisV2(chapterNum int, v2 *types.AnalysisResultV2) {
	engine, model := a.featureModel()
	car := types.ChapterAnalysisResult{
		ChapterNum:     chapterNum,
		ChapterFile:    fmt.Sprintf("%03d.md", chapterNum),
		AnalyzedAt:     time.Now().Format(time.RFC3339),
		Engine:         engine,
		Model:          model,
		AnalyzerSource: "llm",
		Result:         *v2,
	}
	if err := a.pm.UpsertAnalysisV2(car); err != nil {
		slog.Warn("analysis-v2.json 落盘失败（不影响分析返回）", "chapter", chapterNum, "error", err)
	}
}

// persistStoryMemories 记忆生产者（t3-P1）：从 V2 载荷按规则表抽取本章
// 故事记忆并整章替换落盘 memories/MMM-<n>-memory.json（确定性 ID+整文件
// 替换=重分析幂等，规避 MuMu D1）。容错注入：失败只记日志。
func (a *Agent) persistStoryMemories(chapterNum int, v2 *types.AnalysisResultV2, chapterContent string) {
	mems := ExtractStoryMemories(chapterNum, fmt.Sprintf("%03d.md", chapterNum), v2, chapterContent)
	now := time.Now().Format(time.RFC3339)
	for i := range mems {
		mems[i].CreatedAt = now
	}
	if err := a.pm.WriteChapterMemories(chapterNum, mems); err != nil {
		slog.Warn("章节记忆落盘失败（不影响分析返回）", "chapter", chapterNum, "error", err)
		return
	}
	if len(mems) > 0 {
		slog.Info("章节记忆已登记", "chapter", chapterNum, "count", len(mems))
	}
}

// persistAnnotations 标注层（t4-C4）：从 V2 载荷构建 keyword→正文偏移标注
// 并落盘 analysis/annotations/MMM.json。容错注入：失败只记日志。
func (a *Agent) persistAnnotations(chapterNum int, content string, v2 *types.AnalysisResultV2) {
	anns := BuildAnnotations(content, v2)
	if err := a.pm.SaveChapterAnnotations(chapterNum, anns); err != nil {
		slog.Warn("章节标注落盘失败（不影响分析返回）", "chapter", chapterNum, "error", err)
		return
	}
	slog.Info("章节标注已登记", "chapter", chapterNum, "count", len(anns))
}

// pacingLabel 节奏英文枚举 → 中文（旧 wire 是自由中文文本）。
func pacingLabel(p string) string {
	switch strings.TrimSpace(p) {
	case "slow":
		return "节奏偏慢"
	case "moderate":
		return "节奏适中"
	case "fast":
		return "节奏偏快"
	case "varied":
		return "节奏张弛交替"
	}
	return strings.TrimSpace(p)
}

// deriveLegacyAnalysis 从 V2 派生旧 AnalysisResult wire 形状——
// AnalyzeChapter 绑定返回键零变化，前端零改动（t4-C1 兼容口径）。
func deriveLegacyAnalysis(v2 *types.AnalysisResultV2) *AnalysisResult {
	if v2 == nil {
		return nil
	}
	old := &AnalysisResult{
		Conflict:        v2.Conflict.Description,
		SceneRhythm:     pacingLabel(v2.Pacing),
		ImprovementTips: v2.Suggestions,
		Foreshadows:     v2.Foreshadows,
	}
	// hook：取首个钩子的内容（旧 wire 是单句钩子效果）
	if len(v2.Hooks) > 0 {
		old.Hook = strings.TrimSpace(v2.Hooks[0].Content)
	}
	// 情感曲线：主导情绪 + 轨迹
	arc := v2.EmotionalArc
	parts := make([]string, 0, 2)
	if s := strings.TrimSpace(arc.PrimaryEmotion); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(arc.Curve); s != "" {
		parts = append(parts, s)
	}
	old.EmotionCurve = strings.Join(parts, "：")
	// 关键事件：情节推进点前 5 条
	for i, pp := range v2.PlotPoints {
		if i >= 5 {
			break
		}
		if c := strings.TrimSpace(pp.Content); c != "" {
			old.KeyEvents = append(old.KeyEvents, c)
		}
	}
	// 质量分：overall 四舍五入到 1-10 整数
	if q := int(v2.Scores.Overall + 0.5); q >= 1 {
		old.QualityScore = q
	}
	// 角色状态：V2 差分字段映射
	for _, cs := range v2.CharacterStates {
		old.CharacterStates = append(old.CharacterStates, CharacterStateChange{
			Name:     cs.Name,
			OldState: cs.OldState,
			NewState: cs.NewState,
		})
	}
	return old
}

// syncCharacterStatesV2 角色状态同步（V2 差分载荷）：按名字命中即写 NewState。
// 空 NewState 不写（差分缺省=无变化，不猜值）。
func (a *Agent) syncCharacterStatesV2(v2 *types.AnalysisResultV2) {
	if v2 == nil || len(v2.CharacterStates) == 0 {
		return
	}
	chars, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("syncCharacterStates: 读取角色失败", "error", err)
		return
	}
	if chars == nil {
		return
	}
	changed := false
	for i := range chars.Characters {
		for _, sc := range v2.CharacterStates {
			if chars.Characters[i].Name == sc.Name && strings.TrimSpace(sc.NewState) != "" {
				chars.Characters[i].Status = sc.NewState
				changed = true
			}
		}
	}
	if changed {
		a.pm.WriteCharacters(chars)
	}
}
