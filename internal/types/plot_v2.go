package types

import "time"

// ── 章节规划契约 ChapterPlan（t4 C2）──────────────────────────
//
// 口径来源：docs/distill/04-plot-analysis.md §8.3（C2）。
// 分批展开：batch_size=5 + 跨批差异化上下文（全量已有章节摘要 + 已用 key_events 最后 20 条）
// + sub_index 全局重写 + 双序重排。落盘建议 `chapters/plans.json`（map[chapterNum]ChapterPlan），
// 与既有 ChapterSummary 并存，不替换。
type ChapterPlan struct {
	SubIndex       int      `json:"sub_index"` // 大纲内相对序（稳定，全局重写后的权威序）
	Title          string   `json:"title"`
	PlotSummary    string   `json:"plot_summary"` // 200-300 字
	KeyEvents      []string `json:"key_events"`   // 2-4 条，跨章不得重复
	CharacterFocus []string `json:"character_focus"`
	EmotionalTone  string   `json:"emotional_tone"`
	NarrativeGoal  string   `json:"narrative_goal"`
	ConflictType   string   `json:"conflict_type"`
	EndingType     string   `json:"ending_type"` // 悬念|冲突升级|情节转折|情感收尾|自然过渡
	EstimatedWords int      `json:"estimated_words,omitempty"`
}

// ChapterPlanEndingType 结尾类型枚举（§8.3 的 5 值）。
type ChapterPlanEndingType string

const (
	EndingSuspense   ChapterPlanEndingType = "悬念"
	EndingEscalation ChapterPlanEndingType = "冲突升级"
	EndingTwist      ChapterPlanEndingType = "情节转折"
	EndingEmotional  ChapterPlanEndingType = "情感收尾"
	EndingNatural    ChapterPlanEndingType = "自然过渡"
)

// ── 重写版本与恢复（t4 C3）──────────────────────────────────
//
// 口径来源：docs/distill/04-plot-analysis.md §8.3（C3）与 09-impl-handoff.md §3-11。
//
// gaea 的**强制增量**：MuMu 的局部重写直接拼接落库、无快照无 undo；整章重写有快照
// 但**没有 restore 路由**。因此 gaea 必须自带版本恢复——RewriteVersion 携带
// OriginalContent 全文快照（partial 模式也存全文，保证可回滚），由 restore 路径回写。

// RewriteMode 重写模式。
type RewriteMode string

const (
	RewriteModeWhole   RewriteMode = "whole"   // 整章重写
	RewriteModePartial RewriteMode = "partial" // 局部重写（按 StartPos/EndPos 区间）
	RewriteModeDeslop  RewriteMode = "deslop"  // 去 AI 味（复用 novelstyle.DeSlopRewrite）
)

// RewriteStatus 重写版本状态。
type RewriteStatus string

const (
	RewritePending   RewriteStatus = "pending"
	RewriteRunning   RewriteStatus = "running"
	RewriteCompleted RewriteStatus = "completed"
	RewriteFailed    RewriteStatus = "failed"
	RewriteApplied   RewriteStatus = "applied"   // 已落到正文
	RewriteDiscarded RewriteStatus = "discarded" // 已丢弃（仍保留快照，可再应用）
)

// RewriteSource 重写输入来源。
type RewriteSource string

const (
	RewriteSourceCustom      RewriteSource = "custom"               // 纯自定义指令
	RewriteSourceSuggestions RewriteSource = "analysis_suggestions" // 分析建议
	RewriteSourceMixed       RewriteSource = "mixed"                // 两者混合
)

// RewriteLengthMode 长度模式。
const (
	LengthSimilar  = "similar"
	LengthExpand   = "expand"
	LengthCondense = "condense"
	LengthCustom   = "custom"
)

// RewriteVersion 一次重写的完整版本记录（含可回滚快照）。
type RewriteVersion struct {
	ID             string          `json:"id"` // uuid
	ChapterNum     int             `json:"chapter_num"`
	Mode           RewriteMode     `json:"mode"`   // whole|partial|deslop
	Status         RewriteStatus   `json:"status"` // pending|running|completed|failed|applied|discarded
	Source         RewriteSource   `json:"source"` // custom|analysis_suggestions|mixed
	SuggestionIdxs []int           `json:"suggestion_idxs,omitempty"`
	CustomInstr    string          `json:"custom_instructions,omitempty"`
	FocusAreas     []string        `json:"focus_areas,omitempty"`
	Preserve       *PreserveConfig `json:"preserve_elements,omitempty"`

	// ── partial 专用 ──
	StartPos    int    `json:"start_pos,omitempty"`
	EndPos      int    `json:"end_pos,omitempty"`
	LengthMode  string `json:"length_mode,omitempty"` // similar|expand|condense|custom
	TargetWords int    `json:"target_word_count,omitempty"`

	// ── 内容与度量 ──
	// OriginalContent 是**全文**快照（partial 模式也是全文，保证可回滚）。
	OriginalContent   string `json:"original_content"`
	OriginalWordCount int    `json:"original_word_count"`
	NewContent        string `json:"new_content,omitempty"`
	NewWordCount      int    `json:"new_word_count,omitempty"`
	// Similarity 0-100（difflib 等价物）。
	Similarity float64 `json:"similarity,omitempty"`
	// BeforeAIScore / AfterAIScore 复用 novelstyle.TasteScore.Score（0-100）。
	BeforeAIScore int `json:"before_ai_score,omitempty"`
	AfterAIScore  int `json:"after_ai_score,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`

	// RestoredAt / RestoredFrom 由 restore 路径填写，记录「本次恢复」的审计信息。
	RestoredAt   *time.Time `json:"restored_at,omitempty"`
	RestoredFrom string     `json:"restored_from,omitempty"`
}

// PreserveConfig 重写时必须保留的元素。
type PreserveConfig struct {
	Structure       bool     `json:"preserve_structure"`
	Dialogues       []string `json:"preserve_dialogues,omitempty"`
	PlotPoints      []string `json:"preserve_plot_points,omitempty"`
	CharacterTraits bool     `json:"preserve_character_traits"`
}

// RewriteVersionIndex rewrites/<chapterNum>/index.json 的一条（时间倒序列出用）。
//
// 列表接口**不得下发全文**（OriginalContent/NewContent），避免 MuMu 式的
// 1.74 MB 全量下发；全文按需走单版本读取。
type RewriteVersionIndex struct {
	ID         string        `json:"id"`
	ChapterNum int           `json:"chapter_num"`
	Mode       RewriteMode   `json:"mode"`
	Status     RewriteStatus `json:"status"`
	Similarity float64       `json:"similarity,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
}

// RewriteRequest 重写请求（Wails 绑定入参）。
type RewriteRequest struct {
	Source             RewriteSource   `json:"source"`
	SuggestionIdxs     []int           `json:"suggestion_indices,omitempty"`
	CustomInstructions string          `json:"custom_instructions,omitempty"`
	FocusAreas         []string        `json:"focus_areas,omitempty"`
	Preserve           *PreserveConfig `json:"preserve_elements,omitempty"`
	StyleID            int             `json:"style_id,omitempty"`
	TargetWordCount    int             `json:"target_word_count,omitempty"` // 500..10000，默认 3000
	AutoApply          bool            `json:"auto_apply"`                  // 默认 false
	Mode               RewriteMode     `json:"mode,omitempty"`
	StartPos           int             `json:"start_pos,omitempty"`
	EndPos             int             `json:"end_pos,omitempty"`
	LengthMode         string          `json:"length_mode,omitempty"`
	// SelectedText 选中文本（partial）：前端选区原文，后端据此做 ±50 模糊重锚。
	// rune 偏移（StartPos/EndPos）由前端从 code-unit selectionStart/End 换算。
	SelectedText string `json:"selected_text,omitempty"`
}

// ── 分析标注层（t4 C4）──────────────────────────────────────

// Annotation 分析标注：keyword → 正文偏移的内联高亮单元。
//
// Pos/Length 为 rune 偏移与 rune 长度；Pos = -1 表示未命中（前端不高亮，仅列条目）。
type Annotation struct {
	Type       string   `json:"type"` // hook|foreshadow|plot_point|conflict|character|suggestion
	Title      string   `json:"title,omitempty"`
	Content    string   `json:"content"`
	Importance float64  `json:"importance,omitempty"`
	Pos        int      `json:"pos"`
	Length     int      `json:"length,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// AnnotationFile analysis/annotations.json 的落盘结构。
type AnnotationFile struct {
	ChapterNum int          `json:"chapter_num,omitempty"`
	Items      []Annotation `json:"items"`
}
