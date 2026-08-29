// Package whisper — temporal_anchor_policy.go
// 100% 对齐 ackem memory/temporalAnchorPolicy.ts
// 时间锚点写入策略（ingest → temporal_anchors）

package whisper

import "strings"

// TemporalAnchorType 时间锚类型
type TemporalAnchorType string

const (
	AnchorFuzzy        TemporalAnchorType = "fuzzy"
	AnchorRecurring    TemporalAnchorType = "recurring"
	AnchorMilestone    TemporalAnchorType = "milestone"
	AnchorRelationship TemporalAnchorType = "relationship"
)

// recurringSignals 周期性信号关键词
var recurringSignals = []string{
	"生日", "纪念日", "每年", "周年",
	"过年", "中秋", "春节", "清明",
	"端午", "七夕", "元旦", "圣诞",
	"年底", "年初",
}

// DetectAnchorType 检测事实的时间锚类型
func DetectAnchorType(fact *MemoryFact, userMsg string) TemporalAnchorType {
	if fact.Subcategory == "OUR_BOND" && fact.SelfRelevance >= 4.5 {
		if fact.EmotionalContext != nil && fact.EmotionalContext.Intensity >= 0.7 {
			return AnchorRelationship
		}
	}

	if fact.SelfRelevance >= 4.0 {
		return AnchorMilestone
	}
	if fact.EmotionalContext != nil && fact.EmotionalContext.Intensity >= 0.8 {
		return AnchorMilestone
	}

	haystack := fact.Subject + " " + fact.Summary + " " + userMsg
	for _, s := range recurringSignals {
		if strings.Contains(haystack, s) {
			return AnchorRecurring
		}
	}

	return AnchorFuzzy
}

// ShouldWriteTemporalAnchorInput 锚点写入判断输入
type ShouldWriteTemporalAnchorInput struct {
	IsNew     bool
	Weight    float64
	Intensity float64
	Fact      *MemoryFact
	UserMsg   string
}

// ShouldWriteTemporalAnchor 是否应在 ingest 后写入 temporal_anchors
func ShouldWriteTemporalAnchor(input ShouldWriteTemporalAnchorInput) bool {
	if !input.IsNew {
		return false
	}

	anchorType := DetectAnchorType(input.Fact, input.UserMsg)

	// 原强门槛：高 weight + 高情绪
	if input.Weight >= 2 && input.Intensity > 0.5 {
		return true
	}

	// 周期性纪念日：允许较低 weight/情绪
	if anchorType == AnchorRecurring && input.Weight >= 1 && input.Intensity >= 0.35 {
		return true
	}

	if anchorType == AnchorRelationship && input.Intensity >= 0.4 {
		return true
	}

	if anchorType == AnchorMilestone && input.Weight >= 1 && input.Intensity >= 0.45 {
		return true
	}

	// fuzzy 不自动写入，避免锚点表噪音
	return false
}

// TemporalAnchor 时间锚点记录
type TemporalAnchor struct {
	ID                 string             `json:"id"`
	AnchorDate         string             `json:"anchorDate"`
	AnchorType         TemporalAnchorType `json:"anchorType"`
	RecurrenceRule     string             `json:"recurrenceRule,omitempty"` // v4.3a: 循环规则（对齐表列 recurrence_rule）
	LinkedFactIDs      []string           `json:"linkedFactIds"`
	EmotionalValence   float64            `json:"emotionalValence"`
	EmotionalIntensity float64            `json:"emotionalIntensity"`
	Domain             string             `json:"domain"`
	Summary            string             `json:"summary"`
}

// BuildTemporalAnchor 构建时间锚点记录
func BuildTemporalAnchor(fact *MemoryFact, anchorType TemporalAnchorType, now string) TemporalAnchor {
	summary := fact.Summary
	if len(summary) > 200 {
		summary = summary[:200]
	}
	anchor := TemporalAnchor{
		ID:            genHexID(),
		AnchorDate:    now[:10],
		AnchorType:    anchorType,
		LinkedFactIDs: []string{fact.ID},
		Domain:        fact.Domain,
		Summary:       summary,
	}
	if fact.EmotionalContext != nil {
		anchor.EmotionalValence = fact.EmotionalContext.Valence
		anchor.EmotionalIntensity = fact.EmotionalContext.Intensity
	}
	return anchor
}
