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
		lines = append(lines, fmt.Sprintf("[第%d轮]\n用户：%s\n伴侣：%s", turnStart+i, userText, asstText))
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
		return &EpisodeExtractionResult{
			Summary:            truncateStr(j.Summary, episodeSummaryMaxChars),
			EmotionalIntensity: ei,
			DominantEmotion:    de,
			Keywords:           kw,
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

const episodeSystemPrompt = `你是情节摘要提取器。从对话片段中提取一段情节摘要。

输出 JSON：
{
  "summary": "用第三人称简述发生了什么（≤200字）",
  "emotionalIntensity": 0-1之间的数字（情绪强度），
  "dominantEmotion": "主导情绪标签",
  "keywords": ["关键词1", "关键词2"]
}

聚焦于：关系转折、重要事件、情绪高峰期、脆弱时刻、重大决定。日常寒暄返回低 intensity。`
