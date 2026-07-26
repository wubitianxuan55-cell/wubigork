package chapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wubigork/wubigork/internal/ai"
	"github.com/wubigork/wubigork/internal/config"
	"github.com/wubigork/wubigork/internal/project"
	"github.com/wubigork/wubigork/internal/prompt"
	"github.com/wubigork/wubigork/internal/types"
	"github.com/wubigork/wubigork/internal/util"
)

// Agent 章节创作主代理 — 流式生成章节 + 自动摘要
type Agent struct {
	client ai.LLMClient
	pm     *project.Manager
	cfg    *config.Config
	eng    *prompt.Engine
}

// New 创建章节 Agent
func New(client ai.LLMClient, pm *project.Manager, cfg *config.Config, eng *prompt.Engine) *Agent {
	return &Agent{client: client, pm: pm, cfg: cfg, eng: eng}
}
// skillName 可选：注入指定 Skill 的写作指导
// autoAnalyze 可选：生成后自动触发分析
// ── 辅助函数 ─────────────────────────────────────────────────
// ── 辅助函数 ─────────────────────────────────────────────────
func (a *Agent) generateSummary(ctx context.Context, chapterContent string) (*types.ChapterSummary, error) {
	tmpl := a.eng.Get("chapter-summary")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 chapter-summary 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"chapter_content": chapterContent, // Grok 1M 上下文，全文传入
	})

	// 摘要任务：低温度确保精确
	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, a.cfg.Model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		Temperature: 0.15,
		MaxTokens:   2048,
	})
	if err != nil {
		return nil, err
	}

	var summary types.ChapterSummary
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &summary); err != nil {
		return nil, fmt.Errorf("解析摘要 JSON 失败: %w", err)
	}
	return &summary, nil
}

// GenerateSceneIllustration 为章节生成场景插图（Aurora 图片生成）
// chapterContent: 本章正文, summary: 章节摘要, characters: 出场角色列表, worldview: 世界观描述
func (a *Agent) GenerateSceneIllustration(ctx context.Context, chapterContent string, summary *types.ChapterSummary, characters []types.Character, worldview string) (*ai.ImageGenerationResponse, error) {
	// 构建 Aurora prompt
	var promptBuilder strings.Builder
	promptBuilder.WriteString("小说场景插图。")

	// 世界观背景
	promptBuilder.WriteString("世界观: ")
	promptBuilder.WriteString(util.Truncate(worldview, 200))
	promptBuilder.WriteString("。")

	// 章节关键事件
	if summary != nil {
		promptBuilder.WriteString(" 场景: ")
		promptBuilder.WriteString(summary.Summary)
		promptBuilder.WriteString("。")
	}

	// 出场角色外貌
	if len(characters) > 0 {
		promptBuilder.WriteString(" 角色: ")
		for i, ch := range characters {
			if i >= 3 {
				break // 最多3个角色
			}
			promptBuilder.WriteString(fmt.Sprintf("%s(%s, %s)", ch.Name, ch.Appearance, ch.Personality))
			if i < len(characters)-1 && i < 2 {
				promptBuilder.WriteString("; ")
			}
		}
		promptBuilder.WriteString("。")
	}

	// 风格指令
	promptBuilder.WriteString(" 风格: 数字油画，电影级光影，高细节，16:9构图。")

	slog.Info("生成场景插图", "prompt", util.Truncate(promptBuilder.String(), 120))

	req := &ai.ImageGenerationRequest{
		Model:  "grok-imagine-image-quality",
		Prompt: promptBuilder.String(),
		N:      1,
		Size:   "1024x576", // 16:9 宽屏
	}

	return a.client.GenerateImage(ctx, req)
}

// ── Reviser-Reviewer 自动改进循环（蒸馏自 MM-StoryAgent） ──

// ChapterReviewResult 章节质量审查结果
type ChapterReviewResult struct {
	Score      int      `json:"score"`       // 1-10 综合评分
	Strengths  []string `json:"strengths"`   // 优点
	Weaknesses []string `json:"weaknesses"`  // 问题
	RevisePlan string   `json:"revise_plan"` // 修改方案（可直接作为 prompt 注入）
}

// ReviewChapter 审查章节质量（Reviewer 角色）。
// 蒸馏自 MM-StoryAgent 的 revise-recheck 循环：生成内容后自动审查，不合格则反馈改进方案。
func (a *Agent) ReviewChapter(ctx context.Context, chapterContent string, outlineNodeTitle string, prevChapterHint string) (*ChapterReviewResult, error) {
	tmpl := a.eng.Get("chapter-review")
	if tmpl == nil {
		// 回退：使用内置 reviewer prompt
		return a.reviewChapterFallback(ctx, chapterContent, outlineNodeTitle, prevChapterHint)
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"chapter_content":     chapterContent,
		"outline_node_title":  outlineNodeTitle,
		"prev_chapter_hint":   prevChapterHint,
	})

	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, a.cfg.Model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		Temperature:     0.15, // 审查需精确
		MaxTokens:       2048,
		TimeoutMinutes:  5,
	})
	if err != nil {
		return nil, fmt.Errorf("审查章节失败: %w", err)
	}

	var result ChapterReviewResult
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &result); err != nil {
		return nil, fmt.Errorf("解析审查结果 JSON 失败: %w", err)
	}
	return &result, nil
}

// reviewChapterFallback 内置 reviewer（无模板时的回退方案）
func (a *Agent) reviewChapterFallback(ctx context.Context, chapterContent, outlineNodeTitle, prevChapterHint string) (*ChapterReviewResult, error) {
	systemPrompt := `你是资深小说编辑，负责审查章节质量。评估以下维度：
1. 情节推进：是否有效推进故事线
2. 角色刻画：角色行为是否符合人设
3. 文笔质量：句式变化、是否存在 AI 套话
4. 节奏控制：场景切换是否流畅
5. 情感张力：读者情绪曲线是否合理

输出 JSON：
{"score": 1-10, "strengths": ["优点列表"], "weaknesses": ["问题列表"], "revise_plan": "具体的修改方案，可直接作为 prompt 注入到重写流程中"}`

	userPrompt := fmt.Sprintf("大纲节点: %s\n上一章结尾: %s\n\n本章正文:\n%s\n\n请审查并给出修改方案。", outlineNodeTitle, prevChapterHint, chapterContent)

	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, a.cfg.Model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		Temperature: 0.15,
		MaxTokens:   2048,
	})
	if err != nil {
		return nil, err
	}

	var result ChapterReviewResult
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &result); err != nil {
		return nil, fmt.Errorf("解析审查结果 JSON 失败: %w", err)
	}
	return &result, nil
}

// ── v4 场景存储 ─────────────────────────────────────────────

// saveAsScenes 将章节全文存为 v4 场景格式（单场景迁移）
// v4 项目：写为 scenes/001-chapter.md + meta.json
// v3 项目：跳过（由 WriteChapter 处理）
func (a *Agent) saveAsScenes(chapterNum int, content string, summary *types.ChapterSummary) error {
	if !a.pm.IsV4() {
		return nil // v3 项目不写场景
	}

	sm := a.pm.SceneManager(chapterNum)

	// 检查是否已有场景
	existing, _ := sm.List()
	if len(existing) > 0 {
		// 已有场景，更新第一个场景的内容
		sceneObj, err := sm.Read(existing[0].ID)
		if err != nil {
			return err
		}
		sceneObj.Content = content
		if summary != nil {
			sceneObj.Meta.Summary = summary.Summary
			sceneObj.Meta.Emotion = summary.EmotionTone
		}
		return sm.Write(sceneObj)
	}

	// 创建新场景
	title := fmt.Sprintf("第%d章", chapterNum)
	if summary != nil && summary.Title != "" {
		title = summary.Title
	}

	sceneObj, err := sm.Create("chapter", title)
	if err != nil {
		return err
	}
	sceneObj.Content = content
	if summary != nil {
		sceneObj.Meta.Summary = summary.Summary
		sceneObj.Meta.Emotion = summary.EmotionTone
		sceneObj.Meta.Status = types.SceneDone
	}
	return sm.Write(sceneObj)
}
