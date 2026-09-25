package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

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

	// lastSync 最近一轮伏笔同步结果（t1-P4 上绑定面：跳过原因可见，D3 不静默）。
	syncMu   sync.RWMutex
	lastSync *SyncResult
}

// LastSync 返回最近一轮伏笔同步结果；从未执行过同步时 ok=false。
func (a *Agent) LastSync() (SyncResult, bool) {
	a.syncMu.RLock()
	defer a.syncMu.RUnlock()
	if a.lastSync == nil {
		return SyncResult{}, false
	}
	return *a.lastSync, true
}

// New 创建分析 Agent
func New(client ai.LLMClient, pm *project.Manager, cfg *config.Config, eng *prompt.Engine) *Agent {
	return &Agent{client: client, pm: pm, cfg: cfg, eng: eng}
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Hook            string                 `json:"hook"`             // 开头钩子
	Foreshadows     []types.ForeshadowHit  `json:"foreshadows"`      // 伏笔变化（v2 契约：planted|resolved）
	Conflict        string                 `json:"conflict"`         // 冲突分析
	EmotionCurve    string                 `json:"emotion_curve"`    // 情感曲线
	CharacterStates []CharacterStateChange `json:"character_states"` // 角色状态变化
	KeyEvents       []string               `json:"key_events"`       // 关键情节点
	SceneRhythm     string                 `json:"scene_rhythm"`     // 场景节奏
	QualityScore    int                    `json:"quality_score"`    // 1-10
	ImprovementTips []string               `json:"improvement_tips"` // 改进建议
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
	charsJSON := string(util.MustMarshalCompact(chars.PromptView()))

	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		slog.Warn("分析: 读取伏笔失败", "error", err)
	}
	var foreshadowSlots string
	if ff != nil {
		// 分析侧候选清单三层渲染（spec §6.1）：替代整包 JSON 直塞——
		// 已埋入条目带 ID 与逐条回填指令，模型才能把回收挂回正确条目。
		foreshadowSlots = RenderForeshadowCandidates(ff.Items, chapterNum)
	}

	tmpl := a.eng.Get("analysis-chapter")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 analysis-chapter 模板文件")
	}

	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"chapter_content":      chapterContent, // Grok 1M 上下文窗口，不再截断
		"existing_characters":  string(charsJSON),
		"existing_foreshadows": foreshadowSlots,
	})

	// ── 调用 LLM + JSON 解析重试 ──
	caller := func(ctx context.Context, sys, usr string) (string, error) {
		return a.client.ChatSimpleStreamWithOptions(ctx, a.novelModelName(), sys, usr, ai.ChatSimpleOptions{EngineID: a.novelEngineName(),
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

	// V2 解析（t4-C1：9 维结构化 + 三维评分，契约 types.AnalysisResultV2）
	var v2 types.AnalysisResultV2
	if err := json.Unmarshal([]byte(jsonStr), &v2); err != nil {
		return nil, fmt.Errorf("解析分析结果 JSON 失败: %w", err)
	}
	normalizeAnalysisV2(&v2)

	// 落盘 analysis-v2.json（按章号 upsert；容错：失败不影响分析返回）
	a.persistAnalysisV2(chapterNum, &v2)

	// 记忆生产者（t3-P1）：规则表抽取本章故事记忆落盘（自动回填设定库）
	a.persistStoryMemories(chapterNum, &v2, chapterContent)

	// 标注层（t4-C4）：keyword→正文偏移，落盘供前端内联高亮
	a.persistAnnotations(chapterNum, chapterContent, &v2)

	// 同步伏笔到文件（t1-P2：三级匹配 + SyncResult 可追溯，D3 不静默跳过）
	syncRes := a.SyncForeshadows(chapterNum, v2.Foreshadows)
	if len(syncRes.Errors) > 0 || syncRes.SkippedResolveCount > 0 ||
		syncRes.PlantedCount > 0 || syncRes.ResolvedCount > 0 {
		slog.Info("分析: 伏笔同步完成",
			"chapter", chapterNum,
			"planted", syncRes.PlantedCount, "resolved", syncRes.ResolvedCount,
			"created", syncRes.CreatedCount, "matchedByContent", syncRes.MatchedByContent,
			"skipped", syncRes.SkippedResolveCount, "errors", len(syncRes.Errors))
		for _, s := range syncRes.SkippedReasons {
			slog.Debug("分析: 伏笔跳过", "kind", s.Kind, "ref", s.RefID, "reason", s.Message)
		}
		for _, e := range syncRes.Errors {
			slog.Warn("分析: 伏笔同步单条错误", "error", e)
		}
	}

	// 更新角色状态（t5 状态机差分更新器）
	a.syncCharacterStatesV2(chapterNum, &v2)

	// 派生旧 wire 形状：AnalyzeChapter 绑定返回键零变化，前端零改动
	return deriveLegacyAnalysis(&v2), nil
}

// syncCharacterStates 已由 syncCharacterStatesV2 替代（t4-C1 差分载荷），见 analysis_v2.go。

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
			if types.IsResolvedStatus(f.Status) {
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
		return a.client.ChatSimpleStreamWithOptions(ctx, a.novelModelName(), sys, usr, ai.ChatSimpleOptions{EngineID: a.novelEngineName(),
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

// featureModel 小说功能级模型（持久化绑定 func_novel，运行中切换即时生效；空=全局）
func (a *Agent) featureModel() (engine, model string) {
	return a.cfg.GetFeatureModel("novel")
}

// novelModelName 小说功能绑定模型名（空 = 让客户端按引擎解析默认/全局模型）
func (a *Agent) novelModelName() string {
	_, m := a.featureModel()
	return m
}

// novelEngineName 小说功能绑定引擎名（空 = 全局激活引擎）
func (a *Agent) novelEngineName() string {
	e, _ := a.featureModel()
	return e
}
