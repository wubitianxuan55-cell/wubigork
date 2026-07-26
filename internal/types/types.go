package types

import "time"

// ── 项目元信息 ──────────────────────────────────────────────

// ProjectMeta 项目元信息，对应 project.json
type ProjectMeta struct {
	SchemaVersion int       `json:"schema_version"`
	Title         string    `json:"title"`
	Genre         string    `json:"genre"`
	Style         string    `json:"style"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	LastOpenedAt  time.Time `json:"last_opened_at"`
	WordCount     int       `json:"word_count"`
	Version       int       `json:"version"`
}

// ── 世界观 ──────────────────────────────────────────────────

// WorldviewSection 世界观的一个维度
type WorldviewSection struct {
	ID      string `json:"id"`      // 唯一标识，如 era / geography / factions
	Title   string `json:"title"`   // 显示名称，如 "时代背景"
	Content string `json:"content"` // 该维度的 markdown 内容
	Order   int    `json:"order"`   // 排序序号
}

// WorldviewFile worldview.json 完整文件结构
type WorldviewFile struct {
	Sections []WorldviewSection `json:"sections"`
}

// ToMarkdown 将结构化世界观编译为 markdown
func (wf *WorldviewFile) ToMarkdown() string {
	var s string
	for _, sec := range wf.Sections {
		if sec.Content == "" {
			continue
		}
		s += "## " + sec.Title + "\n\n" + sec.Content + "\n\n"
	}
	return s
}

// ConsistencyIssue 一致性检查发现的问题
type ConsistencyIssue struct {
	Severity    string `json:"severity"` // error / warning / info
	Section     string `json:"section"`  // 涉及的世界观维度
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// ConsistencyReport 一致性检查报告
type ConsistencyReport struct {
	Issues      []ConsistencyIssue `json:"issues"`
	OverallNote string             `json:"overall_note"`
}

// ── 角色与组织 ──────────────────────────────────────────────

// Character 角色定义，对应 characters.json 中一条
type Character struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RoleType    string `json:"role_type"` // protagonist / antagonist / supporting / minor
	Gender      string `json:"gender,omitempty"`
	Age         string `json:"age,omitempty"`
	Personality string `json:"personality,omitempty"`
	Background  string `json:"background,omitempty"`
	Appearance  string `json:"appearance,omitempty"`
	Figure      string `json:"figure,omitempty"` // 身材体型
	Motivation  string `json:"motivation,omitempty"`
	Arc         string `json:"arc,omitempty"` // 角色弧光轨迹
	Status      string `json:"status"`        // Alive / Dead / Missing / Transformed
	Notes       string `json:"notes,omitempty"`
	PortraitURL string `json:"portrait_url,omitempty"` // 角色剧照 URL 或 data URL
}

// Organization 组织/势力
type Organization struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type,omitempty"` // 门派/国家/商会...
	Description string   `json:"description,omitempty"`
	PowerLevel  string   `json:"power_level,omitempty"` // 实力等级
	Location    string   `json:"location,omitempty"`
	Motto       string   `json:"motto,omitempty"`
	Members     []string `json:"members,omitempty"` // 成员角色 ID
}

// Relationship 角色/组织之间的关系
type Relationship struct {
	FromID       string `json:"from_id"`
	ToID         string `json:"to_id"`
	RelationType string `json:"relation_type"` // friend / enemy / family / mentor / rival / lover / member / leader
	Description  string `json:"description,omitempty"`
	Intimacy     int    `json:"intimacy"` // -100(死敌) ~ 100(灵魂伴侣)
}

// CharacterFile characters.json 完整文件结构
type CharacterFile struct {
	Characters    []Character    `json:"characters"`
	Organizations []Organization `json:"organizations,omitempty"`
	Relationships []Relationship `json:"relationships,omitempty"`
}

// ── 大纲 ────────────────────────────────────────────────────

// OutlineNodeStatus 大纲节点的写作状态
type OutlineNodeStatus string

const (
	OutlinePlanned   OutlineNodeStatus = "planned"
	OutlineWriting   OutlineNodeStatus = "writing"
	OutlineDone      OutlineNodeStatus = "done"
	OutlineAbandoned OutlineNodeStatus = "abandoned"
)

// OutlineNode 大纲树节点，对应 outline.json 中的一条
type OutlineNode struct {
	ID          string            `json:"id"`
	ParentID    string            `json:"parent_id,omitempty"`
	Title       string            `json:"title"`
	Summary     string            `json:"summary"` // 本章计划：谁、在哪、发生什么
	SceneIdeas  []string          `json:"scene_ideas,omitempty"`
	Characters  []string          `json:"characters,omitempty"` // 出场角色 ID
	KeyPoints   []string          `json:"key_points,omitempty"`
	Emotion     string            `json:"emotion,omitempty"` // 本章情感基调
	Status      OutlineNodeStatus `json:"status"`
	ChapterFile string            `json:"chapter_file,omitempty"` // 对应 chapters/NNN.md 或 NNN{a,b,c}.md
	SceneRefs   []string          `json:"scene_refs,omitempty"`   // v4: 关联的场景 ID 列表
	OrderIndex  int               `json:"order_index"`
	Branch      string            `json:"branch,omitempty"`       // 分支字母: ""=主线, "a"/"b"/"c"=分支
	Children    []OutlineNode     `json:"children,omitempty"`
}

// OutlineFile outline.json 完整文件结构
type OutlineFile struct {
	StoryThread string         `json:"story_thread,omitempty"` // 故事主线：核心冲突、叙事方向
	Nodes       []OutlineNode `json:"nodes"`
}

// ── 章节与摘要 ──────────────────────────────────────────────

// ChapterSummary 章节摘要，生成后自动写入 chapters/NNN-summary.json
type ChapterSummary struct {
	Title              string             `json:"title"`
	Summary            string             `json:"summary"`             // 3-5 句摘要
	KeyEvents          []string           `json:"key_events"`          // 关键情节点
	CharactersAppeared []string           `json:"characters_appeared"` // 出场角色 ID
	ForeshadowChanges  []ForeshadowChange `json:"foreshadow_changes,omitempty"`
	EmotionTone        string             `json:"emotion_tone"`
	QualityEstimate    int                `json:"quality_estimate"` // 1-10
}

// ── 伏笔 ────────────────────────────────────────────────────

// ForeshadowStatus 伏笔状态
type ForeshadowStatus string

const (
	ForeshadowPlanted  ForeshadowStatus = "planted"
	ForeshadowHinted   ForeshadowStatus = "hinted"
	ForeshadowRevealed ForeshadowStatus = "revealed"
)

// Foreshadow 伏笔追踪条目
type Foreshadow struct {
	ID          string           `json:"id"`       // stable_id = {type}_{chapter}_{content_hash}
	Category    string           `json:"category"` // character / plot / world / relationship
	Description string           `json:"description"`
	PlantedIn   string           `json:"planted_in"`            // 章节文件名
	RevealedIn  string           `json:"revealed_in,omitempty"` // 回收章节
	Status      ForeshadowStatus `json:"status"`
	IsLongTerm  bool             `json:"is_long_term"`
}

// ForeshadowChange 每章的伏笔变化
type ForeshadowChange struct {
	ForeshadowID string `json:"foreshadow_id"`
	Action       string `json:"action"` // planted / hinted / revealed
	Description  string `json:"description"`
}

// ForeshadowFile foreshadows.json 完整文件结构
type ForeshadowFile struct {
	Items []Foreshadow `json:"items"`
}

// ── Lorebook 词条 ────────────────────────────────────────────

// LorebookEntry 世界观词条（角色名/地名/道具/概念等）
type LorebookEntry struct {
	Key      string `json:"key"`      // 触发词（如「青云宗」「灵石」）
	Content  string `json:"content"`  // 词条设定（50-300字）
	Category string `json:"category"` // character / location / item / concept
}

// LorebookFile lorebook.json 完整文件结构
type LorebookFile struct {
	Entries []LorebookEntry `json:"entries"`
}

// ── 场景（v4 原子写作单元）────────────────────────────────

// SceneStatus 场景写作状态
type SceneStatus string

const (
	SceneDraft    SceneStatus = "draft"
	SceneRevising SceneStatus = "revising"
	SceneDone     SceneStatus = "done"
	ScenePaused   SceneStatus = "paused"
)

// SceneMeta 场景元数据，存储为 scenes/MMM-slug.meta.json
type SceneMeta struct {
	ID        string      `json:"id"`                  // 唯一标识，如 "001-opening"
	Slug      string      `json:"slug"`                // URL 友好短名
	Title     string      `json:"title"`               // 场景名
	Summary   string      `json:"summary"`             // 一句话概要
	POVCharID string      `json:"pov_char_id,omitempty"` // POV 角色 ID
	Location  string      `json:"location,omitempty"`  // 地点
	TimeOfDay string      `json:"time_of_day,omitempty"` // 时间（黎明/早晨/下午/黄昏/夜晚/深夜）
	Emotion   string      `json:"emotion,omitempty"`   // 情感基调
	Tags      []string    `json:"tags,omitempty"`      // 标签: climax/action/dialogue/...
	Status    SceneStatus `json:"status"`
	WordCount int         `json:"word_count"`
	Order     int         `json:"order"` // 在章节内的排序
}

// Scene 场景完整数据（元数据 + 正文）
type Scene struct {
	Meta    SceneMeta `json:"meta"`
	Content string    `json:"content"` // markdown 正文
}

// ── 快照（场景版本历史）───────────────────────────────────

// Snapshot 单个快照 — 存储行级增量 diff
type Snapshot struct {
	ID        string     `json:"id"`                // 快照 ID（时间戳纳秒）
	SceneID   string     `json:"scene_id"`          // 所属场景 ID
	Timestamp time.Time  `json:"timestamp"`         // 快照时间
	Label     string     `json:"label,omitempty"`   // 可选标签，如 "AI 改写前"
	Trigger   string     `json:"trigger,omitempty"` // 触发原因: "manual" / "ai-rewrite" / "ai-generate"
	DiffLines []DiffLine `json:"diff_lines"`        // 行级 diff（相对于上一快照或空内容）
	WordCount int        `json:"word_count"`        // 快照时的字数
}

// DiffLine 行级差异条目 — 简单 unified diff 格式
type DiffLine struct {
	Type    string `json:"type"`    // "same" / "add" / "del"
	Content string `json:"content"` // 该行文本
	LineNum int    `json:"line_num"`
}

// SnapshotChain 一个场景的快照链（按时间排序）
type SnapshotChain struct {
	SceneID   string     `json:"scene_id"`
	Snapshots []Snapshot `json:"snapshots"`
}

// ── 故事记忆（语义检索用）───────────────────────────────────
//
// TODO Phase 4: 集成 ChromaDB 语义检索。以下类型当前未使用，仅为接口预定义。

// StoryMemory 用于 ChromaDB 语义检索的记忆条目
type StoryMemory struct {
	ID         string    `json:"id"`
	Text       string    `json:"text"`
	Category   string    `json:"category"` // event / character / world / plot
	ChapterRef string    `json:"chapter_ref"`
	CreatedAt  time.Time `json:"created_at"`
	Embedding  []float32 `json:"embedding,omitempty"`
}

// MemoryResult 语义检索返回结果
type MemoryResult struct {
	Memory   StoryMemory `json:"memory"`
	Distance float64     `json:"distance"`
}

// ── 项目上下文（注入 AI prompt）─────────────────────────────

// ContextPriority 上下文优先级
type ContextPriority string

const (
	PriorityP0 ContextPriority = "P0" // 必须传（如当前大纲）
	PriorityP1 ContextPriority = "P1" // 重要（如角色概要）
	PriorityP2 ContextPriority = "P2" // 参考（如历史记忆）
)

// ProjectContext AI 生成时使用的完整项目上下文
type ProjectContext struct {
	Project        ProjectMeta      `json:"project"`
	Worldview      string           `json:"worldview"`
	Characters     []Character      `json:"characters"`
	Organizations  []Organization   `json:"organizations"`
	Relationships  []Relationship   `json:"relationships"`
	Outlines       []OutlineNode    `json:"outlines"`
	CurrentOutline *OutlineNode     `json:"current_outline,omitempty"`
	PrevChapter    string           `json:"prev_chapter,omitempty"` // 上一章正文（截断）
	PrevSummary    *ChapterSummary  `json:"prev_summary,omitempty"`
	AllSummaries   []ChapterSummary `json:"all_summaries,omitempty"` // 所有章节摘要
	Foreshadows    []Foreshadow     `json:"foreshadows"`
	StoryThread    string           `json:"story_thread,omitempty"`  // 故事主线
	VolumeContext  string           `json:"volume_context,omitempty"` // 当前章的父卷上下文
	Memories       []MemoryResult   `json:"memories,omitempty"`
}
