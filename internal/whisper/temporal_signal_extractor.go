// Package whisper — temporal_signal_extractor.go
// 100% 对齐 ackem memory/temporalSignalExtractor.ts
// 时间信号提取器：检测用户消息中的时间信号（关键词+规则版，去embedding依赖）

package whisper

import "strings"

// TemporalSemanticSignal 时间语义信号
type TemporalSemanticSignal struct {
	Label string `json:"label"`
	Type  string `json:"type"` // exact/recurring/fuzzy
}

// TemporalAnchorSentences 预定义时间锚点句子
var TemporalAnchorSentences = []string{
	"去年这个时候", "上周的今天", "一个月前", "三个月前", "半年前",
	"上周", "上个月", "去年", "前年",
	"明天", "后天", "下周", "下个月", "明年",
	"最近", "前几天", "前阵子", "那天", "那时候",
	"生日", "纪念日", "过年", "中秋", "新年",
	"年底", "年初", "开学", "毕业", "入职",
	"上次", "好久不见", "很久没", "又过了一年",
	"每天", "每周", "每月", "每年", "经常",
}

var recurringKeywords = []string{"生日", "纪念日", "过年", "中秋", "新年", "每天", "每周", "每月", "每年", "经常", "年底", "年初"}
var exactKeywords = []string{"明天", "后天", "下周", "下个月", "明年", "上周", "上个月", "去年"}
var fuzzyKeywords = []string{"时候", "的前", "前阵子", "那天", "那时候", "好久"}

// DetectTemporalSignal 从用户消息中检测时间信号（关键词版）
func DetectTemporalSignal(msg string) *TemporalSemanticSignal {
	msg = strings.ToLower(msg)
	var bestLabel string
	var bestLen int

	for _, sentence := range TemporalAnchorSentences {
		if strings.Contains(msg, strings.ToLower(sentence)) {
			if len([]rune(sentence)) > bestLen {
				bestLen = len([]rune(sentence))
				bestLabel = sentence
			}
		}
	}

	if bestLabel == "" {
		return nil
	}

	signalType := "fuzzy"
	for _, kw := range fuzzyKeywords {
		if strings.Contains(bestLabel, kw) {
			signalType = "fuzzy"
			break
		}
	}
	if signalType == "fuzzy" {
		for _, kw := range recurringKeywords {
			if strings.Contains(bestLabel, kw) {
				signalType = "recurring"
				break
			}
		}
	}
	for _, kw := range exactKeywords {
		if strings.Contains(bestLabel, kw) {
			signalType = "exact"
			break
		}
	}

	return &TemporalSemanticSignal{Label: bestLabel, Type: signalType}
}
