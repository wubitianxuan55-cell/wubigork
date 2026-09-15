package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/rewrite"
	"github.com/gaea/gaea/internal/types"
)

// ── 驱动式整章重写（t4-C3 首刀：whole 模式 + 版本库）──────────────
//
// 口径来源：docs/distill/04-plot-analysis.md §5.2（chapter_regenerator.py +
// schemas/regeneration.py）与 types/plot_v2.go 契约（v4.278 落库）。
// gaea 强制增量：MuMu 只在前端内存里「放弃」且无 restore 路由——本刀版本库
// 自带恢复（写回 OriginalContent + 审计字段）。生成后不自动落章：应用走
// NovelApplyRewriteVersion（对齐 MuMu「前端确认后 PUT」语义）。
// 首刀范围：whole 模式（v3 章节文件）；partial 已放开（下方分支）；v4 场景工程
// 与 deslop 下刀待放开。

// NovelChapterRewrite 驱动式整章重写：归一请求 → 取分析建议 → 构建修改指令 →
// LLM 重写（温度 0.7，MuMu L92）→ 输出清理 → diff 统计 → 版本落盘（completed，
// 不自动应用）。同步返回版本摘要与新全文（前端对比确认后调 Apply）。
func (a *writingState) NovelChapterRewrite(chapterNum int, reqJSON string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.client == nil {
		return nil, fmt.Errorf("AI client not ready")
	}
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章号非法")
	}
	var req types.RewriteRequest
	if strings.TrimSpace(reqJSON) != "" {
		if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
			return nil, fmt.Errorf("解析重写请求失败: %w", err)
		}
	}
	req, err := rewrite.NormalizeRequest(req)
	if err != nil {
		return nil, err
	}
	// partial 选段局部重写：只重写选中片段、前后文原样拼接（t4-C3 partial 刀）。
	// 整章（whole）路径零变化。
	if req.Mode == types.RewriteModePartial {
		return a.novelChapterRewritePartial(pm, chapterNum, req)
	}

	// 建议来源：analysis-v2.json 该章 Result.Suggestions（t4-C1 产物）
	var suggestions []string
	if req.Source != types.RewriteSourceCustom {
		af, err := pm.ReadAnalysisV2File()
		if err != nil {
			return nil, fmt.Errorf("读取分析结果失败: %w", err)
		}
		for i := range af.Items {
			if af.Items[i].ChapterNum == chapterNum {
				suggestions = af.Items[i].Result.Suggestions
				break
			}
		}
		if len(suggestions) == 0 {
			return nil, fmt.Errorf("该章节暂无分析结果，请先执行章节分析")
		}
	}

	// v4 场景章：读拼接视图（v3 走 fallback 零变化）——重写整章语义以全章为对象。
	original, err := pm.ReadChapterAsStitch(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取章节失败: %w", err)
	}

	instruction := rewrite.BuildModificationInstruction(req, suggestions)
	tmpl := a.eng.Get("rewrite-chapter")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 rewrite-chapter 模板文件")
	}
	systemPrompt := substituteWordCount(tmpl.BuildSystemPrompt(""), req.TargetWordCount)
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"chapter_content":          original,
		"modification_instruction": instruction,
		"prev_summary":             buildPrevSummaryWindow(readOutlineNodes(pm), chapterNum, prevSummaryResolver(pm)),
	})

	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return nil, fmt.Errorf("未找到可用模型（可能离线）")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	raw, err := a.client.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		EngineID: eng, Feature: "novel", Temperature: 0.7, MaxTokens: 8192, TimeoutMinutes: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("重写生成失败: %w", err)
	}
	newContent := rewrite.CleanRewriteOutput(raw)
	if strings.TrimSpace(newContent) == "" {
		return nil, fmt.Errorf("重写结果为空，已放弃（不落任何数据）")
	}

	diff := rewrite.ComputeDiff(original, newContent)
	v := &types.RewriteVersion{
		ChapterNum:        chapterNum,
		Mode:              types.RewriteModeWhole,
		Status:            types.RewriteCompleted,
		Source:            req.Source,
		SuggestionIdxs:    req.SuggestionIdxs,
		CustomInstr:       req.CustomInstructions,
		FocusAreas:        req.FocusAreas,
		Preserve:          req.Preserve,
		OriginalContent:   original,
		OriginalWordCount: len([]rune(original)),
		NewContent:        newContent,
		NewWordCount:      len([]rune(newContent)),
		Similarity:        diff.Similarity,
	}
	if err := pm.SaveRewriteVersion(v); err != nil {
		return nil, fmt.Errorf("保存重写版本失败: %w", err)
	}
	slog.Info("整章重写完成", "chapter", chapterNum, "version", v.ID,
		"similarity", diff.Similarity, "change", diff.Change)

	return map[string]interface{}{
		"versionId":         v.ID,
		"status":            string(v.Status),
		"similarity":        diff.Similarity,
		"change":            diff.Change,
		"changePercent":     diff.ChangePercent,
		"originalWordCount": diff.OriginalLen,
		"newWordCount":      diff.NewLen,
		"newContent":        newContent,
	}, nil
}

// novelChapterRewritePartial 选段局部重写（NovelChapterRewrite 的 partial 分支）：
// 选区 ±50 模糊重锚 → ±500 上下文截取 → 长度模式四档 → 同一 rewrite-chapter 模板
// 只重写选段 → 清理输出 → 前后文原样拼接 → 版本落全文快照（回滚=整章恢复，
// 对齐 whole；MuMu 局部重写无快照无 undo 是 F5 缺陷）。
// 口径：docs/distill/04-plot-analysis.md §5.3；rune 偏移由前端换算，后端全程 rune。
func (a *writingState) novelChapterRewritePartial(pm *project.Manager, chapterNum int, req types.RewriteRequest) (map[string]interface{}, error) {
	// v4 场景章守卫：选段拼接回写会破坏场景结构且无法对齐边界，V1 不做
	//（观察池：选段→场景映射）。整章重写对场景章已开放（whole 路径）。
	if pm.IsV4() {
		if metas, err := pm.SceneManager(chapterNum).List(); err == nil && len(metas) > 0 {
			return nil, fmt.Errorf("场景工程章暂不支持局部重写（选段与场景边界无法对齐），请使用整章重写")
		}
	}
	original, err := pm.ReadChapter(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取章节失败: %w", err)
	}
	runes := []rune(original)
	start, end, err := rewrite.ResolveSelection(original, req.StartPos, req.EndPos, req.SelectedText)
	if err != nil {
		return nil, err
	}
	selected := string(runes[start:end])
	// ±500 rune 前后文（仅进指令供参考，不进 chapter_content）
	ctxBefore := string(runes[max(0, start-500):start])
	ctxAfter := string(runes[end:min(len(runes), end+500)])

	spec := rewrite.PartialLengthSpec(req.LengthMode, len([]rune(selected)), req.TargetWordCount)
	instruction := rewrite.BuildPartialInstruction(req.CustomInstructions, selected, ctxBefore, ctxAfter, spec)

	tmpl := a.eng.Get("rewrite-chapter")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 rewrite-chapter 模板文件")
	}
	systemPrompt := substituteWordCount(tmpl.BuildSystemPrompt(""), spec.MaxRunes)
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"chapter_content":          selected, // 只给选段；选区外上下文在修改指令里
		"modification_instruction": instruction,
		"prev_summary":             buildPrevSummaryWindow(readOutlineNodes(pm), chapterNum, prevSummaryResolver(pm)),
	})

	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return nil, fmt.Errorf("未找到可用模型（可能离线）")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	// MaxTokens 按 spec 计算（whole 固定 8192 不动，仅本分支）
	raw, err := a.client.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		EngineID: eng, Feature: "novel", Temperature: 0.7, MaxTokens: rewrite.PartialMaxTokens(spec), TimeoutMinutes: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("重写生成失败: %w", err)
	}
	newSelected := rewrite.CleanRewriteOutput(raw)
	if strings.TrimSpace(newSelected) == "" {
		return nil, fmt.Errorf("重写结果为空，已放弃（不落任何数据）")
	}
	newFull := string(runes[:start]) + newSelected + string(runes[end:])
	diff := rewrite.ComputeDiff(selected, newSelected) // 相似度/变化按选段口径

	v := &types.RewriteVersion{
		ChapterNum:        chapterNum,
		Mode:              types.RewriteModePartial,
		Status:            types.RewriteCompleted,
		Source:            req.Source,
		CustomInstr:       req.CustomInstructions,
		StartPos:          start,
		EndPos:            end,
		LengthMode:        req.LengthMode,
		OriginalContent:   original, // 全文快照（回滚=整章恢复）
		OriginalWordCount: len(runes),
		NewContent:        newFull,
		NewWordCount:      len([]rune(newFull)),
		Similarity:        diff.Similarity,
	}
	if req.LengthMode == rewrite.LengthModeCustom {
		v.TargetWords = req.TargetWordCount
	}
	if err := pm.SaveRewriteVersion(v); err != nil {
		return nil, fmt.Errorf("保存重写版本失败: %w", err)
	}
	slog.Info("局部重写完成", "chapter", chapterNum, "version", v.ID,
		"start", start, "end", end, "similarity", diff.Similarity, "change", diff.Change)

	return map[string]interface{}{
		"versionId":            v.ID,
		"status":               string(v.Status),
		"similarity":           diff.Similarity,
		"change":               diff.Change,
		"changePercent":        diff.ChangePercent,
		"originalWordCount":    len(runes),
		"newWordCount":         len([]rune(newFull)),
		"newContent":           newFull,
		"mode":                 string(types.RewriteModePartial),
		"selectedWordCount":    diff.OriginalLen,
		"newSelectedWordCount": diff.NewLen,
		"lengthMode":           req.LengthMode,
		"startPos":             start,
		"endPos":               end,
	}, nil
}

// NovelListRewriteVersions 重写版本索引（时间倒序，无全文下发）。
func (a *writingState) NovelListRewriteVersions(chapterNum int) ([]types.RewriteVersionIndex, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	return pm.ListRewriteVersions(chapterNum)
}

// NovelGetRewriteVersion 读单个版本全文（对比页用，含 Original/New 快照）。
func (a *writingState) NovelGetRewriteVersion(chapterNum int, versionID string) (*types.RewriteVersion, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	return pm.GetRewriteVersion(chapterNum, versionID)
}

// NovelApplyRewriteVersion 应用重写版本：新内容写回正文 + 状态 applied。
// 已 applied 再应用幂等成功。
func (a *writingState) NovelApplyRewriteVersion(chapterNum int, versionID string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	v, err := pm.GetRewriteVersion(chapterNum, versionID)
	if err != nil {
		return nil, err
	}
	switch v.Status {
	case types.RewriteApplied:
		return map[string]interface{}{"applied": true, "idempotent": true}, nil
	case types.RewriteCompleted, types.RewriteDiscarded:
		// 可应用
	default:
		return nil, fmt.Errorf("版本状态 %q 不可应用", v.Status)
	}
	if strings.TrimSpace(v.NewContent) == "" {
		return nil, fmt.Errorf("版本无新内容，不可应用")
	}
	if err := pm.WriteChapter(chapterNum, v.NewContent); err != nil {
		return nil, fmt.Errorf("写回正文失败: %w", err)
	}
	// v4 场景章：整章新全文与旧场景边界无法对齐——重置为单场景（CreateChapter
	// 整章重写完成点同款 rebuildScenesFromBlob；v3/无场景章 no-op）。
	rebuildScenesFromBlob(pm, chapterNum)
	now := time.Now()
	v.Status = types.RewriteApplied
	v.AppliedAt = &now
	if err := pm.UpdateRewriteVersion(v); err != nil {
		return nil, err
	}
	slog.Info("重写版本已应用", "chapter", chapterNum, "version", v.ID)
	return map[string]interface{}{"applied": true}, nil
}

// NovelDiscardRewriteVersion 丢弃重写版本（快照保留，可再应用——MuMu 的
// 「放弃只清前端内存」在 gaea 升级为持久状态）。
func (a *writingState) NovelDiscardRewriteVersion(chapterNum int, versionID string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	v, err := pm.GetRewriteVersion(chapterNum, versionID)
	if err != nil {
		return err
	}
	if v.Status != types.RewriteCompleted {
		return fmt.Errorf("版本状态 %q 不可丢弃（仅 completed 可丢弃）", v.Status)
	}
	v.Status = types.RewriteDiscarded
	return pm.UpdateRewriteVersion(v)
}

// NovelRestoreRewriteVersion 从已应用版本恢复原文（gaea 强制增量：MuMu 无
// restore 路由）：OriginalContent 写回正文 + RestoredAt/RestoredFrom 审计。
func (a *writingState) NovelRestoreRewriteVersion(chapterNum int, versionID string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	v, err := pm.GetRewriteVersion(chapterNum, versionID)
	if err != nil {
		return nil, err
	}
	if v.Status != types.RewriteApplied {
		return nil, fmt.Errorf("仅已应用的版本可恢复原文（当前 %q）", v.Status)
	}
	if strings.TrimSpace(v.OriginalContent) == "" {
		return nil, fmt.Errorf("版本缺原文快照，无法恢复")
	}
	if err := pm.WriteChapter(chapterNum, v.OriginalContent); err != nil {
		return nil, fmt.Errorf("恢复正文失败: %w", err)
	}
	// v4 场景章恢复同语义：原文写回后重置单场景（与 Apply 对称）。
	rebuildScenesFromBlob(pm, chapterNum)
	now := time.Now()
	v.RestoredAt = &now
	v.RestoredFrom = v.ID
	if err := pm.UpdateRewriteVersion(v); err != nil {
		return nil, err
	}
	slog.Info("重写版本已恢复原文", "chapter", chapterNum, "version", v.ID)
	return map[string]interface{}{"restored": true}, nil
}

// prevSummaryResolver 返回章节摘要回退链解析器（大纲 Summary → 章节摘要文件，
// 与 t3 首刀前文窗口同源口径）。
func prevSummaryResolver(pm *project.Manager) func(types.OutlineNode) string {
	return func(n types.OutlineNode) string {
		if s := strings.TrimSpace(n.Summary); s != "" {
			return s
		}
		if cs, err := pm.ReadChapterSummary(n.OrderIndex); err == nil && cs != nil {
			if s := strings.TrimSpace(cs.Summary); s != "" {
				return s
			}
		}
		return ""
	}
}

// readOutlineNodes 读取大纲节点（读取失败返回空——前文衔接是增强数据）。
func readOutlineNodes(pm *project.Manager) []types.OutlineNode {
	of, err := pm.ReadOutlines()
	if err != nil || of == nil {
		return []types.OutlineNode{}
	}
	return of.Nodes
}

// NovelChapterSuggestions 读取该章分析建议（analysis-v2.json 的
// Result.Suggestions，供重写建议驱动 UI 勾选）。无分析结果返回空数组
// （正常态，前端提示先分析），不报错。
func (a *writingState) NovelChapterSuggestions(chapterNum int) ([]string, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	af, err := pm.ReadAnalysisV2File()
	if err != nil {
		return nil, fmt.Errorf("读取分析结果失败: %w", err)
	}
	for i := range af.Items {
		if af.Items[i].ChapterNum == chapterNum {
			return af.Items[i].Result.Suggestions, nil
		}
	}
	return []string{}, nil
}

// NovelChapterAnnotations 读取该章分析标注（keyword→正文 rune 偏移，前端
// 内联高亮数据源）。缺档且该章有 V2 分析时按需重建（存量分析免重跑）。
func (a *writingState) NovelChapterAnnotations(chapterNum int) ([]types.Annotation, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	f, err := pm.ReadChapterAnnotations(chapterNum)
	if err != nil {
		return nil, err
	}
	if len(f.Items) > 0 || f.ChapterNum != 0 {
		return f.Items, nil
	}
	// 缺档重建：需该章 V2 分析与正文都可得
	af, err := pm.ReadAnalysisV2File()
	if err != nil {
		return nil, err
	}
	var v2 *types.AnalysisResultV2
	for i := range af.Items {
		if af.Items[i].ChapterNum == chapterNum {
			v2 = &af.Items[i].Result
			break
		}
	}
	if v2 == nil {
		return []types.Annotation{}, nil
	}
	content, err := pm.ReadChapter(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取章节失败: %w", err)
	}
	anns := analysis.BuildAnnotations(content, v2)
	if err := pm.SaveChapterAnnotations(chapterNum, anns); err != nil {
		slog.Warn("标注按需重建落盘失败", "chapter", chapterNum, "error", err)
	}
	return anns, nil
}
