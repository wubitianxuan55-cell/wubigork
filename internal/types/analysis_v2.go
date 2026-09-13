package types

// ── 章节分析结果（9 维 + 三维分档评分）共享契约 ─────────────────
//
// 口径来源：docs/distill/04-plot-analysis.md §8.3（C1）与 09-impl-handoff.md §5-M1。
//
// 归属决策（与规格原文的差异，已在回报中说明）：
//   t4 §8.3 建议这些类型落在 `internal/analysis/types.go`。但分析结果同时被
//   伏笔域（ForeshadowHit → syncForeshadows）与角色域（CharacterStateChangeV2
//   → 差分更新）消费，而 `internal/analysis/` 与 `internal/plotanalysis/` 是
//   两个不同实现者的 inScope。为避免同一契约被两处各定义一份而漂移，
//   统一放在 `internal/types/`（共享契约层，零反向依赖），各域直接引用。
//
// 零破坏迁移：本文件全部类型为**新增**，既有 analysis.AnalysisResult 保留不动，
// analysis-v2.json 与 analysis.json 并存（旧 JSON 不因本契约失效）。
//
// 三维分档评分（rubric 5 档，overall=(P+E+C)/3）与建议数量硬联动：
//   overall <  4.0 → 4-5 条建议
//   overall <  6.0 → 3-4 条
//   overall <  8.0 → 2-3 条
//   overall >= 8.0 → 0-1 条
// 并在 Prompt 中明令禁止打 7-8 分安全分（见 SuggestionCountForOverall）。

// Keyword 逐字 keyword 锚点（强制 8-25 字原文复制）。
//
// Pos 为**正文字符偏移**（rune 偏移，不是 byte 偏移）：
//
//	0..n  命中位置
//	-1    未命中
//
// 定位必须做坐标反投影（去标点匹配后把索引映射回原文），
// 不得沿用 MuMu 仅 ±50 字符的窗口校正（t4 §8.4）。
type Keyword struct {
	Text   string `json:"text"`   // 从原文逐字复制，8-25 字
	Pos    int    `json:"pos"`    // 正文字符偏移；-1 = 未命中
	Length int    `json:"length"` // 命中的字符长度
}

// Hook 开头/中段/结尾钩子。
type Hook struct {
	Type     string  `json:"type"` // 悬念|情感|冲突|认知
	Content  string  `json:"content"`
	Strength int     `json:"strength"` // 1-10
	Position string  `json:"position"` // 开头|中段|结尾
	Keyword  Keyword `json:"keyword"`
}

// ForeshadowHit 分析命中/产出的伏笔条目。
//
// ReferenceStableID 复用 gaea 既有稳定 ID 口径 sha256(category+chapterFile+description)[:8]
// （analysis.GenerateStableID），不做序号 ID（MuMu 用序号，只能先删后建）。
type ForeshadowHit struct {
	Title             string   `json:"title"`
	Content           string   `json:"content"`
	Type              string   `json:"type"` // planted|resolved
	Strength          int      `json:"strength"`
	Subtlety          int      `json:"subtlety"`
	Category          string   `json:"category"` // identity|mystery|item|relationship|event|ability|prophecy
	IsLongTerm        bool     `json:"is_long_term"`
	RelatedChars      []string `json:"related_characters,omitempty"`
	EstimateResolve   int      `json:"estimated_resolve_chapter,omitempty"`
	ReferenceChapter  int      `json:"reference_chapter,omitempty"`
	ReferenceStableID string   `json:"reference_stable_id,omitempty"`
	Keyword           Keyword  `json:"keyword"`
}

// Conflict 冲突维度。
type Conflict struct {
	Types              []string `json:"types"` // 人与人|人与己|人与环境|人与社会
	Parties            []string `json:"parties,omitempty"`
	Level              int      `json:"level"` // 1-10
	Description        string   `json:"description"`
	ResolutionProgress float64  `json:"resolution_progress"` // 0.0-1.0
}

// EmotionalArc 情感曲线维度。
type EmotionalArc struct {
	PrimaryEmotion    string   `json:"primary_emotion"`
	Intensity         int      `json:"intensity"` // 1-10
	Curve             string   `json:"curve"`
	SecondaryEmotions []string `json:"secondary_emotions,omitempty"`
}

// CharacterStateChangeV2 角色状态变化（与既有 CharacterStateChange 并存）。
type CharacterStateChangeV2 struct {
	Name                string            `json:"name"`
	OldState            string            `json:"old_state,omitempty"`
	NewState            string            `json:"new_state,omitempty"`
	PsychologicalChange string            `json:"psychological_change,omitempty"`
	KeyEvent            string            `json:"key_event,omitempty"`
	SurvivalStatus      string            `json:"survival_status,omitempty"` // active|deceased|missing|retired
	RelationshipChanges map[string]string `json:"relationship_changes,omitempty"`
}

// PlotPoint 情节推进点。
type PlotPoint struct {
	Content    string  `json:"content"`
	Type       string  `json:"type"`       // revelation|conflict|resolution|transition
	Importance float64 `json:"importance"` // 0.0-1.0
	Impact     string  `json:"impact"`
	Keyword    Keyword `json:"keyword"`
}

// AnalysisScene 场景维度（9 维之一）。
//
// 命名说明：不叫 Scene，因为 types 包已有 v4 场景工程的 Scene 类型（内容 + 元数据）。
type AnalysisScene struct {
	Location   string `json:"location"`
	Atmosphere string `json:"atmosphere"`
	Duration   string `json:"duration"`
}

// AnalysisScores 三维分档评分（rubric 5 档）。
//
// Overall 必须等于 (Pacing+Engagement+Coherence)/3 的舍入值（±0.5 容差），
// 由 ComputeOverallScore 提供唯一实现，禁止各域自算。
type AnalysisScores struct {
	Pacing        float64 `json:"pacing"`     // 1.0-10.0，一位小数
	Engagement    float64 `json:"engagement"` // 1.0-10.0，一位小数
	Coherence     float64 `json:"coherence"`  // 1.0-10.0，一位小数
	Overall       float64 `json:"overall"`    // (P+E+C)/3
	Justification string  `json:"score_justification"`
}

// AnalysisDimensions 9 维分析的完整载荷。
//
// 9 维 = hooks / foreshadows / conflict / emotional_arc / character_states /
// plot_points / scenes / pacing(节奏) / dialogue_ratio+description_ratio(文白比)
// ——与 t4 §8.3 的 AnalysisV2 字段逐一对齐，另加 scores/suggestions/summary 三项输出。
type AnalysisResultV2 struct {
	Hooks            []Hook                   `json:"hooks"`
	Foreshadows      []ForeshadowHit          `json:"foreshadows"`
	Conflict         Conflict                 `json:"conflict"`
	EmotionalArc     EmotionalArc             `json:"emotional_arc"`
	CharacterStates  []CharacterStateChangeV2 `json:"character_states"`
	PlotPoints       []PlotPoint              `json:"plot_points"`
	Scenes           []AnalysisScene          `json:"scenes"`
	Pacing           string                   `json:"pacing"` // slow|moderate|fast|varied
	DialogueRatio    float64                  `json:"dialogue_ratio"`
	DescriptionRatio float64                  `json:"description_ratio"`
	Scores           AnalysisScores           `json:"scores"`
	PlotStage        string                   `json:"plot_stage"`
	Suggestions      []string                 `json:"suggestions"` // 数量与 Overall 硬联动
	Summary          string                   `json:"summary"`     // 供记忆/检索用
}

// ChapterAnalysisResult 落盘单元：analysis-v2.json 的一条。
//
// 与 AnalysisResultV2 分离，是为了让「分析载荷」可被独立复用（例如单测直接构造
// 载荷），而落盘层携带章节元信息。
type ChapterAnalysisResult struct {
	ChapterNum     int              `json:"chapter_num"`
	ChapterFile    string           `json:"chapter_file,omitempty"`
	AnalyzedAt     string           `json:"analyzed_at,omitempty"` // ISO8601
	Engine         string           `json:"engine,omitempty"`
	Model          string           `json:"model,omitempty"`
	AnalyzerSource string           `json:"analyzer_source,omitempty"` // llm|manual|import
	Result         AnalysisResultV2 `json:"result"`
}

// ── 评分联动规则（唯一实现，禁止各域复制）────────────────────

// ComputeOverallScore 由三维分算总分：overall = (P+E+C)/3。
//
// 保留一位小数（与 t4 「一位小数」要求一致），便于前端直接渲染。
func ComputeOverallScore(pacing, engagement, coherence float64) float64 {
	sum := pacing + engagement + coherence
	avg := sum / 3.0
	// 一位小数四舍五入，避免 6.666666666666667 之类浮点噪声落盘。
	return round1(avg)
}

func round1(v float64) float64 {
	if v >= 0 {
		return float64(int64(v*10+0.5)) / 10
	}
	return float64(int64(v*10-0.5)) / 10
}

// SuggestionCountForOverall 建议数量与 overall 的硬联动区间。
//
// 返回 [min, max]。规格（t4 §8.2 C1）：
//
//	overall <  4.0 → 4-5 条
//	overall <  6.0 → 3-4 条
//	overall <  8.0 → 2-3 条
//	overall >= 8.0 → 0-1 条
//
// 目的是禁止「打 7-8 分安全分」——分数越低建议越多，作者无法靠给中间分逃避修改。
func SuggestionCountForOverall(overall float64) (min, max int) {
	switch {
	case overall < 4.0:
		return 4, 5
	case overall < 6.0:
		return 3, 4
	case overall < 8.0:
		return 2, 3
	default:
		return 0, 1
	}
}

// KeywordMinRunes / KeywordMaxRunes keyword 锚点必须逐字复制的字数区间（含端点）。
const (
	KeywordMinRunes = 8
	KeywordMaxRunes = 25
)

// ValidKeywordRuneCount 报告 keyword 的字数是否落在 8-25 区间。
//
// 调用方必须用 utf8.RuneCountInString 取 n，禁止用 len()（handoff §3-14）。
func ValidKeywordRuneCount(n int) bool {
	return n >= KeywordMinRunes && n <= KeywordMaxRunes
}
