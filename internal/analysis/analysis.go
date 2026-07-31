package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// Agent 情节分析子代理 — 9维度分析 + 伏笔追踪
type Agent struct {
	client ai.LLMClient
	pm     *project.Manager
	cfg    *config.Config
	eng    *prompt.Engine
}

// New 创建分析 Agent
func New(client ai.LLMClient, pm *project.Manager, cfg *config.Config, eng *prompt.Engine) *Agent {
	return &Agent{client: client, pm: pm, cfg: cfg, eng: eng}
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Hook            string                 `json:"hook"`             // 开头钩子
	Foreshadows     []ForeshadowAction     `json:"foreshadows"`      // 伏笔变化
	Conflict        string                 `json:"conflict"`         // 冲突分析
	EmotionCurve    string                 `json:"emotion_curve"`    // 情感曲线
	CharacterStates []CharacterStateChange `json:"character_states"` // 角色状态变化
	KeyEvents       []string               `json:"key_events"`       // 关键情节点
	SceneRhythm     string                 `json:"scene_rhythm"`     // 场景节奏
	QualityScore    int                    `json:"quality_score"`    // 1-10
	ImprovementTips []string               `json:"improvement_tips"` // 改进建议
}

// ForeshadowAction 伏笔动作
type ForeshadowAction struct {
	Category    string `json:"category"` // character / plot / world / relationship
	Action      string `json:"action"`   // planted / hinted / revealed
	Description string `json:"description"`
	StableID    string `json:"stable_id,omitempty"`
}

// CharacterStateChange 角色状态变化
type CharacterStateChange struct {
	Name     string `json:"name"`
	OldState string `json:"old_state"`
	NewState string `json:"new_state"`
}

// Analyze 分析章节，返回分析结果 + 更新伏笔文件
func (a *Agent) Analyze(ctx context.Context, chapterNum int, chapterContent string) (*AnalysisResult, error) {
	chars, err := a.pm.ReadCharacters()
	if err != nil {
		slog.Warn("分析: 读取角色失败", "error", err)
	}
	charsJSON := string(util.MustMarshalCompact(chars))

	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		slog.Warn("分析: 读取伏笔失败", "error", err)
	}
	foreshadowsJSON := string(util.MustMarshalCompact(ff))

	tmpl := a.eng.Get("analysis-chapter")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 analysis-chapter 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"chapter_content":      chapterContent, // Grok 1M 上下文窗口，不再截断
		"existing_characters":  string(charsJSON),
		"existing_foreshadows": string(foreshadowsJSON),
	})

	// ── 调用 LLM + JSON 解析重试 ──
	caller := func(ctx context.Context, sys, usr string) (string, error) {
		return a.client.ChatSimpleStreamWithOptions(ctx, a.cfg.Model, sys, usr, ai.ChatSimpleOptions{
			Temperature:     a.cfg.AnalysisTemperature,
			ReasoningEffort: a.cfg.ReasoningEffort,
			MaxTokens:       4096,
			TimeoutMinutes:  10,
		})
	}
	jsonStr, err := util.RetryJSON(ctx, caller, systemPrompt, userPrompt, 2)
	if err != nil {
		return nil, err
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析分析结果 JSON 失败: %w", err)
	}

	// 同步伏笔到文件
	a.syncForeshadows(chapterNum, &result)

	// 更新角色状态
	a.syncCharacterStates(&result)

	return &result, nil
}

// syncForeshadows 将分析出的伏笔变化同步到 foreshadows.json
func (a *Agent) syncForeshadows(chapterNum int, result *AnalysisResult) {
	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		slog.Warn("syncForeshadows: 读取伏笔失败", "error", err)
	}
	if ff == nil {
		ff = &types.ForeshadowFile{Items: []types.Foreshadow{}}
	}

	chapterFile := fmt.Sprintf("%03d.md", chapterNum)

	for _, action := range result.Foreshadows {
		if action.Action == "planted" {
			// 新建伏笔
			stableID := GenerateStableID(action.Category, chapterFile, action.Description)
			ff.Items = append(ff.Items, types.Foreshadow{
				ID:          stableID,
				Category:    action.Category,
				Description: action.Description,
				PlantedIn:   chapterFile,
				Status:      types.ForeshadowPlanted,
				IsLongTerm:  false,
			})
		} else if action.Action == "revealed" || action.Action == "hinted" {
			// 更新已有伏笔状态
			for i := range ff.Items {
				if ff.Items[i].ID == action.StableID ||
					ff.Items[i].Description == action.Description {
					if action.Action == "revealed" {
						ff.Items[i].Status = types.ForeshadowRevealed
						ff.Items[i].RevealedIn = chapterFile
					} else {
						ff.Items[i].Status = types.ForeshadowHinted
					}
				}
			}
		}
	}

	// 清洗：AI 重复生成的伏笔去重
	seen := make(map[string]bool)
	var deduped []types.Foreshadow
	for _, f := range ff.Items {
		if !seen[f.ID] {
			seen[f.ID] = true
			deduped = append(deduped, f)
		}
	}
	ff.Items = deduped

	a.pm.WriteForeshadows(ff)
}

// syncCharacterStates 更新角色状态
func (a *Agent) syncCharacterStates(result *AnalysisResult) {
	if len(result.CharacterStates) == 0 {
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

	for i := range chars.Characters {
		for _, sc := range result.CharacterStates {
			if chars.Characters[i].Name == sc.Name {
				chars.Characters[i].Status = sc.NewState
			}
		}
	}

	a.pm.WriteCharacters(chars)
}

// GenerateStableID 生成稳定的伏笔 ID
func GenerateStableID(category, chapterFile, description string) string {
	h := sha256.Sum256([]byte(category + chapterFile + description))
	return fmt.Sprintf("%s_%s_%x", category, chapterFile[:3], h[:8])
}

// ── 全书发展编辑 ──────────────────────────────────────────

// BookReviewResult AI 全书审稿结果
type BookReviewResult struct {
	Letter         string          `json:"letter"`
	TotalScore     int             `json:"total_score"`
	Scores         map[string]int  `json:"scores"`
	Peaks          []int           `json:"peaks"`
	Valleys        []int           `json:"valleys"`
	ArcCompletions []ArcCompletion `json:"arc_completions"`
}

// ArcCompletion 角色弧光完成度
type ArcCompletion struct {
	Character string `json:"character"`
	Progress  int    `json:"progress"`
	Note      string `json:"note"`
}

// BookChapterData 聚合的全书章节数据（逐章统计，不重复调 AI）
type BookChapterData struct {
	ChapterNum   int    `json:"chapter_num"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	WordCount    int    `json:"word_count"`
	QualityScore int    `json:"quality_score"`
	EmotionTone  string `json:"emotion_tone"`
	KeyEvents    string `json:"key_events"`
	Characters   string `json:"characters"`
}

// AggregateBookData 聚合全书数据用于全书审稿
func (a *Agent) AggregateBookData() ([]BookChapterData, map[string][]int, string, string, error) {
	var chapters []BookChapterData
	charChapters := make(map[string][]int)

	for i := 1; ; i++ {
		content, err := a.pm.ReadChapter(i)
		if err != nil {
			break
		}
		summary, err := a.pm.ReadChapterSummary(i)
		if err != nil {
			slog.Warn("读取章节摘要失败", "chapter", i, "error", err)
		}

		ch := BookChapterData{
			ChapterNum: i,
			WordCount:  len([]rune(content)),
		}
		if summary != nil {
			ch.Title = summary.Title
			ch.Summary = summary.Summary
			ch.QualityScore = summary.QualityEstimate
			ch.EmotionTone = summary.EmotionTone
			ch.KeyEvents = strings.Join(summary.KeyEvents, " / ")
			ch.Characters = strings.Join(summary.CharactersAppeared, "、")
			for _, c := range summary.CharactersAppeared {
				charChapters[c] = append(charChapters[c], i)
			}
		}
		chapters = append(chapters, ch)
	}

	// 世界观
	wv, err := a.pm.ReadWorldview()
	if err != nil {
		slog.Warn("读取世界观失败", "error", err)
	}
	if wv == "" {
		wv = "（暂无）"
	}

	// 伏笔
	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		slog.Warn("AggregateBookData: 读取伏笔失败", "error", err)
	}
	foreshadowsData := "（暂无伏笔）"
	if ff != nil {
		var fs []string
		revealed, total := 0, len(ff.Items)
		for _, f := range ff.Items {
			status := "埋设"
			if f.Status == types.ForeshadowRevealed {
				status = "已回收"
				revealed++
			} else if f.Status == types.ForeshadowHinted {
				status = "已暗示"
			}
			fs = append(fs, fmt.Sprintf("[%s] %s (%s in %s)", f.Category, f.Description, status, f.PlantedIn))
		}
		foreshadowsData = fmt.Sprintf("总计%d个伏笔，已回收%d个\n", total, revealed) + strings.Join(fs, "\n")
	}

	return chapters, charChapters, wv, foreshadowsData, nil
}

// ReviewBook AI 全书审稿
func (a *Agent) ReviewBook(ctx context.Context) (*BookReviewResult, error) {
	chapters, _, wv, foreshadowData, err := a.AggregateBookData()
	if err != nil {
		return nil, err
	}
	if len(chapters) == 0 {
		return nil, fmt.Errorf("暂无章节数据")
	}

	// 构建综合数据文本
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("全书共%d章。\n\n", len(chapters)))
	for _, ch := range chapters {
		sb.WriteString(fmt.Sprintf("第%d章 %s: %s [品质%d/情绪%s] (事件:%s) (角色:%s)\n",
			ch.ChapterNum, ch.Title, ch.Summary,
			ch.QualityScore, ch.EmotionTone,
			ch.KeyEvents, ch.Characters))
	}
	sb.WriteString("\n---\n世界观:\n" + wv)
	sb.WriteString("\n\n---\n伏笔:\n" + foreshadowData)

	tmpl := a.eng.Get("book-review")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 book-review 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"book_data": sb.String(),
	})

	// ── 全书审稿 + JSON 解析重试 ──
	caller := func(ctx context.Context, sys, usr string) (string, error) {
		return a.client.ChatSimpleStreamWithOptions(ctx, a.cfg.Model, sys, usr, ai.ChatSimpleOptions{
			Temperature:     a.cfg.AnalysisTemperature,
			ReasoningEffort: a.cfg.ReasoningEffort,
			MaxTokens:       4096,
			TimeoutMinutes:  15,
		})
	}
	jsonStr, err := util.RetryJSON(ctx, caller, systemPrompt, userPrompt, 2)
	if err != nil {
		return nil, fmt.Errorf("全书审稿失败: %w", err)
	}

	var result BookReviewResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("解析审稿结果失败: %w", err)
	}

	return &result, nil
}
