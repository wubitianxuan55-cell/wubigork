package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
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
// 首刀范围：whole 模式（v3 章节文件）；v4 场景工程与 partial 下刀放开。

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
	// 场景级工程（正文按场景存储）的整章重写下刀支持；纯文件章照常。
	if sm := pm.SceneManager(chapterNum); sm != nil {
		if metas, err := sm.List(); err == nil && len(metas) > 0 {
			return nil, fmt.Errorf("该章为场景级工程（正文按场景存储），整章重写暂未开放（下一刀支持）")
		}
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

	original, err := pm.ReadChapter(chapterNum)
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
		EngineID: eng, Temperature: 0.7, MaxTokens: 8192, TimeoutMinutes: 10,
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
