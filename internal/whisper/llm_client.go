// Package whisper — llm_client.go
// LLM 客户端接口：whisper 模块通过此接口调用 wubigrok 模型中心
// App 层注入具体实现（ChatSimpleStream 等）

package whisper

// ─── LlmClient ───────────────────────────────────────────────

// LlmClient LLM 调用接口（由 App 层注入 wubigrok 模型中心实现）
type LlmClient interface {
	// Chat 同步聊天（用于事实抽取、情节生成等后台任务）
	Chat(systemPrompt, userPrompt string) (string, error)
}

// ─── FactExtractionResult ────────────────────────────────────

// FactExtractionResult LLM 事实抽取结果
type FactExtractionResult struct {
	Facts []ExtractedFact `json:"facts"`
}

// ExtractedFact 抽取的原始事实
type ExtractedFact struct {
	Domain       string   `json:"domain"`
	Subcategory  string   `json:"subcategory"`
	Subject      string   `json:"subject"`
	Summary      string   `json:"summary"`
	Weight       float64  `json:"weight,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
	SelfRelevance float64 `json:"selfRelevance,omitempty"`
	Triggers     []string `json:"triggers,omitempty"`
}

// ─── EpisodeExtractionResult ─────────────────────────────────

// EpisodeExtractionResult LLM 情节抽取结果
type EpisodeExtractionResult struct {
	Summary            string   `json:"summary"`
	EmotionalIntensity float64  `json:"emotionalIntensity"`
	DominantEmotion    string   `json:"dominantEmotion"`
	Keywords           []string `json:"keywords"`
}

// ─── ConsolidationResult ─────────────────────────────────────

// ConsolidationResult LLM 记忆整合结果
type ConsolidationResult struct {
	Insights     []ConsolidatedInsight `json:"insights"`
	Associations []ConsolidatedAssoc   `json:"associations"`
}

// ConsolidatedInsight 整合洞察
type ConsolidatedInsight struct {
	Domain      string   `json:"domain"`
	Subcategory string   `json:"subcategory"`
	Subject     string   `json:"subject"`
	Summary     string   `json:"summary"`
	Weight      float64  `json:"weight"`
	Confidence  float64  `json:"confidence"`
	DerivedFrom []string `json:"derivedFrom"`
}

// ConsolidatedAssoc 整合关联
type ConsolidatedAssoc struct {
	FactIDA         string  `json:"factIdA"`
	FactIDB         string  `json:"factIdB"`
	AssociationType string  `json:"associationType"`
	Strength        float64 `json:"strength"`
}

// ─── ContradictionResult ─────────────────────────────────────

// ContradictionResult LLM 矛盾检测结果
type ContradictionResult struct {
	Resolutions []ContradictionResolution `json:"resolutions"`
}

// ContradictionResolution 矛盾解决方案
type ContradictionResolution struct {
	PairIndex int    `json:"pair_idx"`
	Judgment  string `json:"judgment"` // conflict / reinforce / unrelated
	Action    string `json:"action"`   // keep_new / keep_old / merge / flag
	Reason    string `json:"reason"`
}
