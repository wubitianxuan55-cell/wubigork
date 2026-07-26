package chapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wubigork/wubigork/internal/ai"
	"github.com/wubigork/wubigork/internal/config"
	ctxpkg "github.com/wubigork/wubigork/internal/context"
	"github.com/wubigork/wubigork/internal/project"
	"github.com/wubigork/wubigork/internal/prompt"
	"github.com/wubigork/wubigork/internal/types"
	"github.com/wubigork/wubigork/internal/util"
)

// Agent 章节创作主代理 — 流式生成章节 + 自动摘要
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
func (a *Agent) Generate(ctx context.Context, outlineNodeID string, skillName string, autoAnalyze bool, targetWords int) (<-chan ai.SSEChunk, int, error) {
	// 1. 加载上下文
	gc, err := a.buildGenerateContext(ctx, outlineNodeID)
	if err != nil {
		return nil, 0, err
	}

	// 2. 构建 prompt
	tmpl := a.eng.Get("chapter-generate")
	if tmpl == nil {
		return nil, 0, fmt.Errorf("缺少 chapter-generate 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt(skillName)
	userPrompt := tmpl.BuildUserPrompt(gc.contextMap)

	// 3. 流式生成 + 质量检查 + 自动重试
	resultCh := make(chan ai.SSEChunk, 64)
	go a.streamAndPostProcess(ctx, gc, systemPrompt, userPrompt, targetWords, resultCh)

	return resultCh, gc.chapterNum, nil
}

// generateContext Generate 中间数据结构
type generateContext struct {
	chapterNum        int
	projCtx           *types.ProjectContext
	prevChapter       string
	prevSummary       *types.ChapterSummary
	activeForeshadows []types.Foreshadow
	contextMap        map[string]string
}

// buildGenerateContext 构建生成上下文 — 提取自 Generate()
func (a *Agent) buildGenerateContext(_ context.Context, outlineNodeID string) (*generateContext, error) {
	projCtx, err := a.pm.LoadContext(outlineNodeID)
	if err != nil {
		return nil, fmt.Errorf("加载上下文失败: %w", err)
	}

	chapterNum := 1
	if projCtx.CurrentOutline != nil && projCtx.CurrentOutline.OrderIndex > 0 {
		chapterNum = projCtx.CurrentOutline.OrderIndex
	}

	prevChapter := ""
	var prevSummary *types.ChapterSummary
	if chapterNum > 1 {
		prevChapter, err = a.pm.ReadChapter(chapterNum - 1)
		if err != nil {
			slog.Warn("chapter: 读取上一章失败", "chapter", chapterNum-1, "error", err)
		}
		prevSummary, err = a.pm.ReadChapterSummary(chapterNum - 1)
		if err != nil {
			slog.Warn("chapter: 读取上一章摘要失败", "chapter", chapterNum-1, "error", err)
		}
	}

	var allSummaries []types.ChapterSummary
	for i := 1; i < chapterNum; i++ {
		s, err := a.pm.ReadChapterSummary(i)
		if err == nil && s != nil {
			allSummaries = append(allSummaries, *s)
		}
	}

	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		slog.Warn("chapter: 读取伏笔失败", "error", err)
	}
	var activeForeshadows []types.Foreshadow
	if ff != nil {
		for _, f := range ff.Items {
			if f.Status == types.ForeshadowPlanted || f.Status == types.ForeshadowHinted {
				activeForeshadows = append(activeForeshadows, f)
			}
		}
	}

	contextMap := map[string]string{}
	if projCtx.CurrentOutline != nil {
		contextMap["outline_node"] = fmt.Sprintf("标题: %s\n摘要: %s\n关键点: %s\n出场角色: %s",
			projCtx.CurrentOutline.Title,
			projCtx.CurrentOutline.Summary,
			strings.Join(projCtx.CurrentOutline.KeyPoints, " / "),
			strings.Join(projCtx.CurrentOutline.Characters, "、"),
		)
	}
	contextMap["prev_chapter"] = prevChapter
	if prevSummary != nil {
		contextMap["prev_summary"] = prevSummary.Summary
	}
	contextMap["character_status"] = buildCharacterStatus(projCtx.Characters)
	if len(allSummaries) > 0 {
		var ss []string
		for _, s := range allSummaries {
			ss = append(ss, s.Summary)
		}
		contextMap["all_summaries"] = strings.Join(ss, "\n")
	}
	if len(activeForeshadows) > 0 {
		var fs []string
		for _, f := range activeForeshadows {
			fs = append(fs, fmt.Sprintf("[%s] %s", f.Category, f.Description))
		}
		contextMap["active_foreshadows"] = strings.Join(fs, "\n")
	}
	contextMap["worldview"] = projCtx.Worldview
	contextMap["story_thread"] = projCtx.StoryThread
	contextMap["volume_context"] = projCtx.VolumeContext
	contextMap["volume_context"] = projCtx.VolumeContext
	contextMap["all_characters"] = buildCharacterList(projCtx.Characters)
	if len(projCtx.Organizations) > 0 {
		var orgs []string
		for _, o := range projCtx.Organizations {
			orgs = append(orgs, fmt.Sprintf("%s [%s]: %s", o.Name, o.Type, o.Description))
		}
		contextMap["organizations"] = strings.Join(orgs, "\n")
	}

	// ── Lorebook 触发式注入 ──────────────────────────────────
	if lorebook := a.injectLorebook(contextMap); lorebook != "" {
		contextMap["lorebook_injections"] = lorebook
	}

	return &generateContext{
		chapterNum:        chapterNum,
		projCtx:           projCtx,
		prevChapter:       prevChapter,
		prevSummary:       prevSummary,
		activeForeshadows: activeForeshadows,
		contextMap:        contextMap,
	}, nil
}

// streamAndPostProcess 流式生成 + 质检重试 + 后处理 — 提取自 Generate() goroutine
func (a *Agent) streamAndPostProcess(ctx context.Context, gc *generateContext, systemPrompt, userPrompt string, targetWords int, resultCh chan<- ai.SSEChunk) {
	defer close(resultCh)

	maxRetries := a.cfg.QualityMaxRetries
	qualityThreshold := a.cfg.QualityThreshold
	if maxRetries <= 0 {
		maxRetries = 2
	}
	if qualityThreshold <= 0 {
		qualityThreshold = 6
	}
	var fullContent string
	currentSystemPrompt := systemPrompt
	currentUserPrompt := userPrompt

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req := &ai.ChatRequest{
			Model: a.cfg.Model,
			Messages: []ai.ChatMessage{
				{Role: "system", Content: currentSystemPrompt},
				{Role: "user", Content: currentUserPrompt},
			},
			MaxTokens:   util.Max(targetWords*3, 4096),
			Temperature: a.cfg.DefaultTemperature,
		}

		chunks, streamErr := a.client.ChatStream(ctx, req)
		if streamErr != nil {
			resultCh <- ai.SSEChunk{Error: fmt.Sprintf("生成失败: %v", streamErr)}
			return
		}

		var sb strings.Builder
		streamLoop:
		for chunk := range chunks {
			if chunk.Error != "" {
				resultCh <- chunk
				return
			}
			if chunk.Done {
				break streamLoop
			}
			sb.WriteString(chunk.Content)
			resultCh <- chunk
		}
		fullContent = sb.String()
		if len(strings.TrimSpace(fullContent)) == 0 {
			resultCh <- ai.SSEChunk{Error: "生成内容为空"}
			return
		}

		if attempt < maxRetries {
			review, revErr := a.quickQualityCheck(ctx, fullContent, gc.projCtx.CurrentOutline.Title, gc.prevChapter)
			if revErr != nil {
				slog.Warn("质量检查失败，跳过重试", "error", revErr, "attempt", attempt)
				break
			}
			if review.Score >= qualityThreshold {
				slog.Info("章节质量通过", "score", review.Score, "attempt", attempt)
				break
			}
			slog.Info("章节质量不达标，自动重试", "score", review.Score, "threshold", qualityThreshold, "attempt", attempt)
			resultCh <- ai.SSEChunk{
				Content: fmt.Sprintf("\n\n---\n[AI 正在根据审稿意见重写... 评分 %d/10 → 目标 ≥%d]\n", review.Score, qualityThreshold),
			}
			currentSystemPrompt = systemPrompt + fmt.Sprintf("\n\n## 上一版的问题和修改要求\n上一版评分 %d/10。请根据以下修改方案重写：\n%s", review.Score, review.RevisePlan)
		}
	}

	// 后处理：保存 + 摘要 + 场景
	if err := a.pm.WriteChapter(gc.chapterNum, fullContent); err != nil {
		resultCh <- ai.SSEChunk{Error: fmt.Sprintf("保存章节失败: %v", err)}
		return
	}
	summary, err := a.generateSummary(ctx, fullContent)
	if err == nil && summary != nil {
		summary.Title = fmt.Sprintf("第 %d 章", gc.chapterNum)
		a.pm.WriteChapterSummary(gc.chapterNum, summary)
	}
	if err := a.saveAsScenes(gc.chapterNum, fullContent, summary); err != nil {
		slog.Warn("v4: 写入场景格式失败", "chapter", gc.chapterNum, "error", err)
	}
	resultCh <- ai.SSEChunk{Done: true}
}

// quickQualityCheck 轻量级质量检查（MM-StoryAgent 的 success_check_fn）。
// 评分 1-10，低于阈值时返回具体的 revise_plan 供重试使用。
func (a *Agent) quickQualityCheck(ctx context.Context, content, outlineTitle, prevChapter string) (*ChapterReviewResult, error) {
	// 截断内容用于快速检查（避免超长内容拖慢检查速度）
	checkContent := content
	if len([]rune(content)) > 4000 {
		checkContent = string([]rune(content)[:4000])
	}

	systemPrompt := `你是快速质量审查员。评估章节质量，只关注最关键的 3 个维度：
1. 可读性：文字是否流畅自然，无 AI 套话
2. 情节推进：是否有效推进故事（对照大纲）
3. 角色一致性：角色行为是否符合作者设定

评分标准：1-3=需大幅重写，4-5=需局部修改，6-7=合格可发布，8-10=优秀。
输出 JSON：{"score": 1-10, "strengths": ["优点"], "weaknesses": ["问题"], "revise_plan": "修改方案"}`

	userPrompt := fmt.Sprintf("大纲: %s\n上一章结尾: %s\n\n正文:\n%s", outlineTitle, util.Truncate(prevChapter, 200), checkContent)

	reply, err := a.client.ChatSimpleStreamWithOptions(ctx, a.cfg.Model, systemPrompt, userPrompt, ai.ChatSimpleOptions{
		Temperature: 0.1,
		MaxTokens:   1024,
	})
	if err != nil {
		return nil, err
	}

	var result ChapterReviewResult
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &result); err != nil {
		return nil, fmt.Errorf("parse quick quality check: %w", err)
	}
	return &result, nil
}

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

// ── 辅助函数 ─────────────────────────────────────────────────

func buildCharacterStatus(characters []types.Character) string {
	var lines []string
	for _, ch := range characters {
		lines = append(lines, fmt.Sprintf("- %s [%s]: %s", ch.Name, ch.RoleType, ch.Status))
	}
	return strings.Join(lines, "\n")
}

func buildCharacterList(characters []types.Character) string {
	var lines []string
	for _, ch := range characters {
		lines = append(lines, fmt.Sprintf("- %s [%s]: %s %s",
			ch.Name, ch.RoleType, ch.Personality, util.Truncate(ch.Background, 60)))
	}
	return strings.Join(lines, "\n")
}
// injectLorebook 从 lorebook 中查找与当前章节上下文匹配的词条并格式化注入文本
func (a *Agent) injectLorebook(contextMap map[string]string) string {
	eng := ctxpkg.NewEngine(a.pm)
	if err := eng.Load(); err != nil {
		slog.Warn("chapter: 加载 lorebook 失败", "error", err)
		return ""
	}

	// 构建搜索文本：大纲节点 + 上一章正文
	searchText := contextMap["outline_node"] + " " + contextMap["prev_chapter"]

	// 最多注入约 1500 tokens 的 lorebook 词条
	triggered := eng.FindTriggers(searchText, 1500)
	if len(triggered) == 0 {
		return ""
	}

	var parts []string
	for _, rule := range triggered {
		parts = append(parts, fmt.Sprintf("- **%s** [%s]: %s",
			rule.Entry.Key, rule.Entry.Category, rule.Entry.Content))
	}
	return strings.Join(parts, "\n")
}
