// Package whisper — memory_ingest.go
// 100% 对齐 ackem memory/ingest.ts
// 记忆摄入管线：LLM事实抽取 → 去重写入 → 三元组 → 退役 → 情节生成

package whisper

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/util"
)

// ─── 事实抽取提示词 ──────────────────────────────────────────

const factExtractionPrompt = `你是一个记忆提取助手。从以下对话中提取关于用户的重要事实。

规则：
1. 只提取用户明确说出的信息，不要推测
2. 每条事实包含 domain/subcategory/subject/summary
3. domain 可选: user_profile/user_behavior/user_state/relationship/companion_reply
4. subcategory 可选: BASIC_PROFILE/PRAISE/VULNERABILITIES/MOOD/OUR_BOND/PREFERENCE/HABIT/SELF_NARRATIVE
5. weight 范围 0.1-1.0，重要信息给高权重
6. confidence 范围 0.3-1.0，明确陈述给高置信度
7. selfRelevance 范围 0-1.0，与用户自身相关度

输出 JSON: {"facts":[{"domain":"...","subcategory":"...","subject":"...","summary":"...","weight":0.5,"confidence":0.7,"selfRelevance":0.6}]}`

// episodeExtractionPrompt 情节记忆摘要提示（对齐 ackem prompt/memory-episode.ts）
const episodeExtractionPrompt = `你是情节记忆摘要器。将以下对话片段总结为一条叙事摘要。

规则：第三人称"用户"和"gaea"；keyQuote 必须一字不差从原文复制（≤15字）；情绪关键词最多3个；标注时间语境；摘要≤200字。

输出 JSON: {"summary":"...","emotionalIntensity":0.7,"dominantEmotion":"...","keywords":["...","..."],"keyQuote":"...","emotionKeywords":["...","..."],"timeContext":"..."}`

// ─── IngestOptions ───────────────────────────────────────────

type IngestOptions struct {
	SkipLlmExtraction bool
	AdultPrivacyLevel string
	DataRoot          string
}

// ─── MemoryIngestPipeline ────────────────────────────────────

type MemoryIngestPipeline struct {
	llm               LlmClient // gaea 模型中心注入
	episodeEmotionMax float64
	onError           MemoryWriteErrorSink // T6-5.3 错误回传（可观测性）
}

// NewMemoryIngestPipeline 创建摄入管线
func NewMemoryIngestPipeline(llm LlmClient) *MemoryIngestPipeline {
	return &MemoryIngestPipeline{llm: llm}
}

// SetErrorSink 注册异步记忆写入错误回传（T6-5.3）：任一错误路径（LLM 失败/
// JSON 解析失败）在记录 slog 后同步调用 sink；nil 可解除。默认仅记录日志。
func (p *MemoryIngestPipeline) SetErrorSink(fn MemoryWriteErrorSink) {
	p.onError = fn
}

// reportError 错误路径统一出口：slog 由调用方记录，这里只负责回传。
func (p *MemoryIngestPipeline) reportError(sessionID, phase string, err error) {
	if p.onError != nil {
		p.onError(sessionID, phase, err)
	}
}

// ─── AfterTurn ───────────────────────────────────────────────

func (p *MemoryIngestPipeline) AfterTurn(args IngestTurnArgs) {
	ec := CaptureEmotionalContext(args.L1, args.L2)

	// 1. LLM 事实抽取（异步后台任务）
	if p.llm != nil && !args.Opts.SkipLlmExtraction {
		p.extractFactsViaLLM(args, ec)
	}

	// 2. 自动退役
	if args.TotalTurns > 0 && args.TotalTurns%AutoRetireCheckInterval == 0 && args.FactStore != nil {
		args.FactStore.AutoRetire()
	}

	// 3. 三元组提取
	if args.KG != nil && args.FactStore != nil {
		p.extractTriples(args.FactStore, args.SessionID, args.TurnIndex, args.KG)
	}

	// 4. 情节生成
	if args.EpisodicStore != nil && len(args.RecentExchanges) >= 3 {
		p.maybeGenerateEpisode(args)
	}

	_ = ec
}

// ─── LLM 事实抽取 ────────────────────────────────────────────

func (p *MemoryIngestPipeline) extractFactsViaLLM(args IngestTurnArgs, ec EmotionalContext) {
	if p.llm == nil || args.FactStore == nil {
		return
	}

	userPrompt := fmt.Sprintf("用户消息：%s\n助手回复：%s", args.UserMsg, args.CompanionMsg)
	reply, err := p.llm.Chat(factExtractionPrompt, userPrompt)
	if err != nil {
		// 记忆摄入为后台尽力而为任务（AfterTurn 无返回值）：LLM 失败仅记录日志，不中断主对话流程
		slog.Error("whisper: LLM 事实抽取失败", "sessionID", args.SessionID, "turnIndex", args.TurnIndex, "error", err)
		p.reportError(args.SessionID, "llm_extract", err) // T6-5.3 错误回传
		return
	}

	var result FactExtractionResult
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &result); err != nil {
		slog.Error("whisper: LLM 事实抽取结果 JSON 解析失败", "sessionID", args.SessionID, "turnIndex", args.TurnIndex, "error", err)
		p.reportError(args.SessionID, "json_parse", err) // T6-5.3 错误回传
		return
	}

	for _, f := range result.Facts {
		// Canon 守卫：过滤与创造者矛盾的事实
		if vetCreatorContradicting(f) {
			continue
		}

		// v5.40: 用户事实守卫 — 只从用户自述写入档案
		gf := GuardableFact{
			Domain:      f.Domain,
			Subcategory: f.Subcategory,
			Subject:     f.Subject,
			Summary:     f.Summary,
		}
		filtered := FilterExtractedUserFacts([]GuardableFact{gf}, args.UserMsg)
		if len(filtered) == 0 {
			continue
		}

		args.FactStore.Add(MemoryFact{
			Domain:           f.Domain,
			Subcategory:      f.Subcategory,
			Subject:          f.Subject,
			Summary:          f.Summary,
			Weight:           f.Weight,
			Confidence:       f.Confidence,
			SelfRelevance:    f.SelfRelevance,
			Triggers:         f.Triggers,
			SourceSessionID:  args.SessionID,
			SourceTurnIndex:  args.TurnIndex,
			EmotionalContext: &ec,
			PrivacyLevel:     args.Opts.AdultPrivacyLevel,
			FactLayer:        "raw",
		})
	}
}

// vetCreatorContradicting 检查是否与创造者设定矛盾
func vetCreatorContradicting(f ExtractedFact) bool {
	if f.Subcategory == "OUR_BOND" && strings.Contains(f.Summary, "创造") {
		return true
	}
	if f.Subject == "Jason" && f.Domain == "user_profile" {
		return true
	}
	return false
}

// ─── 三元组提取 ──────────────────────────────────────────────

func (p *MemoryIngestPipeline) extractTriples(fs *FactStore, sessionID string, turnIndex int, kg *KnowledgeGraph) {
	for _, f := range fs.ListActive() {
		if f.SourceSessionID != sessionID || f.SourceTurnIndex != turnIndex {
			continue
		}
		for _, t := range extractBasicTriples(f) {
			kg.Add(t.Subject, t.Predicate, t.Object, t.Confidence, t.SourceFactIDs)
		}
	}
}

func extractBasicTriples(f *Fact) []Triple {
	switch f.Subcategory {
	case "BASIC_PROFILE":
		return []Triple{{Subject: "用户", Predicate: f.Subject, Object: f.Summary, Confidence: f.Confidence, SourceFactIDs: []string{f.ID}}}
	case "PRAISE":
		return []Triple{{Subject: "用户", Predicate: "赞赏", Object: "gaea", Confidence: f.Confidence, SourceFactIDs: []string{f.ID}}}
	case "VULNERABILITIES":
		return []Triple{{Subject: "用户", Predicate: "表达脆弱", Object: f.Summary, Confidence: f.Confidence, SourceFactIDs: []string{f.ID}}}
	case "OUR_BOND":
		return []Triple{{Subject: "用户", Predicate: "关系", Object: f.Summary, Confidence: f.Confidence, SourceFactIDs: []string{f.ID}}}
	default:
		return []Triple{{Subject: f.Subject, Predicate: f.Domain, Object: f.Summary, Confidence: f.Confidence, SourceFactIDs: []string{f.ID}}}
	}
}

// ─── 情节生成 ────────────────────────────────────────────────

func (p *MemoryIngestPipeline) maybeGenerateEpisode(args IngestTurnArgs) {
	intensity := CaptureEmotionalContext(args.L1, args.L2).Intensity
	if intensity > p.episodeEmotionMax {
		p.episodeEmotionMax = intensity
	}

	interval := EpisodeIntervalTurnsLow
	if p.episodeEmotionMax > EpisodeEmotionIntensityThreshold {
		interval = EpisodeIntervalTurns
	}

	if args.TotalTurns <= 0 || args.TotalTurns%interval != 0 {
		return
	}

	summary := ""
	if p.llm != nil {
		summary = p.generateEpisodeViaLLM(args)
	}
	if summary == "" {
		summary = buildEpisodeSummary(args.RecentExchanges)
	}
	if summary == "" {
		return
	}

	var prevID *string
	if latest := args.EpisodicStore.Latest(); latest != nil {
		prevID = &latest.ID
	}

	args.EpisodicStore.Add(Episode{
		Summary:            summary,
		EmotionalIntensity: p.episodeEmotionMax,
		DominantEmotion:    args.L2.PrimaryLabel,
		Keywords:           extractEpisodeKeywords(args.RecentExchanges),
		PrevEpisodeID:      prevID,
		SourceSessionID:    args.SessionID,
		StartTurn:          args.TurnIndex - len(args.RecentExchanges) + 1,
		EndTurn:            args.TurnIndex,
	})
	p.episodeEmotionMax = 0
}

func (p *MemoryIngestPipeline) generateEpisodeViaLLM(args IngestTurnArgs) string {
	var lines []string
	for _, e := range args.RecentExchanges {
		lines = append(lines, "用户："+e.User)
		lines = append(lines, "gaea："+e.Assistant)
	}
	reply, err := p.llm.Chat(episodeExtractionPrompt, strings.Join(lines, "\n"))
	if err != nil {
		return ""
	}
	var result EpisodeExtractionResult
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &result); err != nil {
		return ""
	}
	return result.Summary
}

// ─── IngestTurnArgs ──────────────────────────────────────────

type IngestTurnArgs struct {
	SessionID       string
	TurnIndex       int
	UserMsg         string
	CompanionMsg    string
	L1              L1State
	L2              EmotionState
	FactStore       *FactStore
	TotalTurns      int
	EpisodicStore   *EpisodicStore
	RecentExchanges []ExchangePair
	KG              *KnowledgeGraph
	Opts            IngestOptions
}

type ExchangePair struct {
	User      string
	Assistant string
}

// ─── 辅助 ────────────────────────────────────────────────────

func buildEpisodeSummary(exchanges []ExchangePair) string {
	if len(exchanges) == 0 {
		return ""
	}
	first := exchanges[0].User
	last := ""
	if len(exchanges) >= 2 {
		last = exchanges[len(exchanges)-1].Assistant
	}
	if len(first) > 80 {
		first = first[:80]
	}
	if len(last) > 80 {
		last = last[:80]
	}
	if last != "" {
	return "用户说「" + first + "…」→ gaea回应「" + last + "…」"
	}
	return "用户说「" + first + "…」"
}

func extractEpisodeKeywords(exchanges []ExchangePair) []string {
	seen := make(map[string]bool)
	var kw []string
	for _, e := range exchanges {
		for _, w := range extractKeywords(e.User) {
			if !seen[w] {
				seen[w] = true
				kw = append(kw, w)
			}
		}
	}
	if len(kw) > 10 {
		kw = kw[:10]
	}
	return kw
}

func extractKeywords(text string) []string {
	var words []string
	runes := []rune(text)
	start := 0
	for i, r := range runes {
		if r == '，' || r == '。' || r == ' ' || r == '！' || r == '？' || r == '\n' {
			if i-start >= 2 {
				words = append(words, string(runes[start:i]))
			}
			start = i + 1
		}
	}
	if len(runes)-start >= 2 {
		words = append(words, string(runes[start:]))
	}
	return words
}

