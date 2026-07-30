// Package whisper — episode_extractor.go
// 100% 对齐 ackem memory/episodeExtractor.ts
// LLM 情节摘要提取器：从多轮对话中提取情节摘要

package whisper

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

const (
	episodeExtractMsgTrunc  = 300
	episodeSummaryMaxChars  = 200
)

// EpisodeExtractor LLM 情节提取器
type EpisodeExtractor struct{}

// NewEpisodeExtractor 创建提取器
func NewEpisodeExtractor() *EpisodeExtractor { return &EpisodeExtractor{} }

// Extract 从对话交换中提取情节摘要
func (ee *EpisodeExtractor) Extract(
	exchanges []struct{ User, Assistant string },
	turnStart int,
	llm LlmClient,
) (*EpisodeExtractionResult, error) {
	var lines []string
	for i, ex := range exchanges {
		userText := truncateStr(ex.User, episodeExtractMsgTrunc)
		asstText := truncateStr(ex.Assistant, episodeExtractMsgTrunc)
		lines = append(lines, fmt.Sprintf("[第%d轮]\n用户：%s\ngaea：%s", turnStart+i, userText, asstText))
	}
	dialogue := strings.Join(lines, "\n\n")

	raw, err := llm.Chat(episodeSystemPrompt, fmt.Sprintf("对话片段：\n%s", dialogue))
	if err != nil {
		return nil, err
	}

	return parseEpisodeResult(raw), nil
}

func parseEpisodeResult(raw string) *EpisodeExtractionResult {
	tryParse := func(s string) *EpisodeExtractionResult {
		var j struct {
			Summary            string   `json:"summary"`
			EmotionalIntensity float64  `json:"emotionalIntensity"`
			DominantEmotion    string   `json:"dominantEmotion"`
			Keywords           []string `json:"keywords"`
			KeyQuote           string   `json:"keyQuote"`
			EmotionKeywords    []string `json:"emotionKeywords"`
			TimeContext        string   `json:"timeContext"`
		}
		if err := json.Unmarshal([]byte(s), &j); err != nil || j.Summary == "" {
			return nil
		}
		ei := j.EmotionalIntensity
		if ei == 0 {
			ei = 0.5
		}
		ei = math.Max(0, math.Min(1, ei))
		de := j.DominantEmotion
		if de == "" {
			de = "中性"
		}
		kw := j.Keywords
		if len(kw) > 5 {
			kw = kw[:5]
		}
		ek := j.EmotionKeywords
		if len(ek) > 3 {
			ek = ek[:3]
		}
		// keyQuote 最多 15 字
		kq := j.KeyQuote
		if len([]rune(kq)) > 15 {
			kq = string([]rune(kq)[:15])
		}
		return &EpisodeExtractionResult{
			Summary:            truncateStr(j.Summary, episodeSummaryMaxChars),
			EmotionalIntensity: ei,
			DominantEmotion:    de,
			Keywords:           kw,
			KeyQuote:           kq,
			EmotionKeywords:    ek,
			TimeContext:        j.TimeContext,
		}
	}

	trimmed := strings.TrimSpace(raw)
	if r := tryParse(trimmed); r != nil {
		return r
	}
	i := strings.Index(trimmed, "{")
	j := strings.LastIndex(trimmed, "}")
	if i >= 0 && j > i {
		return tryParse(trimmed[i : j+1])
	}
	return nil
}

// episodeSystemPrompt 情节记忆摘要系统提示
// 100% 对齐 ackem prompt/memory-episode.ts
const episodeSystemPrompt = `你是情节记忆摘要器。将对话片段总结为一条叙事摘要。

── 规则 ──
- 使用第三人称"用户"和"gaea"
- 提炼对话的核心事件和情绪转折
- keyQuote 必须一字不差地从原文复制，绝对禁止润色或改写，截取最核心的 15 字以内
- 输出关键情绪词，最多 3 个，按强度排序
- 标注时间语境（"今天下午""昨天深夜""上周五"）
- 摘要 ≤200 字

── 输出格式 ──
严格 JSON：
{"summary":"用户今天...","emotionalIntensity":0.7,"dominantEmotion":"焦虑","keywords":["工作","压力"],"keyQuote":"用户原话（≤15字）","emotionKeywords":["焦虑","委屈"],"timeContext":"今天下午"}`
