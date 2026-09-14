package types

import "fmt"

// ── 小说故事记忆（t3-P1 记忆生产者）────────────────────────────
//
// 口径来源：docs/distill/03-long-range-consistency.md §1.1（MuMu StoryMemory
// 字段子集）/ §12.1/§12.2（gaea 落盘与确定性 ID 决策）。
// 存储 = 每章独立文件 memories/MMM-<n>-memory.json（整章重写替换，
// 幂等且天然规避 MuMu D1「重分析不清旧数据」）；不落向量列，
// 语义召回（t3-P2）由 semantic.Store 的 kind="story_memory" 承担。

// StoryMemoryType 记忆类型值域（对齐 MuMu models/memory.py:17-26 注释枚举，
// 含其注释漏记但实际写入的 chapter_summary）。
const (
	MemoryTypeChapterSummary = "chapter_summary"
	MemoryTypeHook           = "hook"
	MemoryTypeForeshadow     = "foreshadow"
	MemoryTypePlotPoint      = "plot_point"
	MemoryTypeCharacterEvent = "character_event"
)

// 伏笔记忆的 is_foreshadow 三值（MuMu :47 口径：0=普通 / 1=已埋下 / 2=已回收）。
const (
	ForeshadowFlagNone     = 0
	ForeshadowFlagPlanted  = 1
	ForeshadowFlagResolved = 2
)

// StoryMemory 单条故事记忆。字段子集对齐 §12.1；无 Position 字段——
// MuMu 的 chapter_position 依赖 keyword 偏移标注（t4-C4 未落地），
// 不引入恒 0 死字段（§5.1 D11 原则：不留永空字段）。
type StoryMemory struct {
	ID                string   `json:"id"`                      // 确定性 ID：<MMM>-<type>-<ordinal>
	ChapterNum        int      `json:"chapter_num"`             // 所属章号（= story_timeline）
	ChapterFile       string   `json:"chapter_file,omitempty"`  // 章节文件名（如 012.md）
	Type              string   `json:"type"`                    // StoryMemoryType 值域
	Title             string   `json:"title,omitempty"`         // 短题（钩子类型/条目标题）
	Content           string   `json:"content"`                 // 记忆正文
	Importance        float64  `json:"importance"`              // 0.0-1.0（规则表赋值）
	IsForeshadow      int      `json:"is_foreshadow,omitempty"` // 仅 foreshadow 类：1 埋下 / 2 回收
	RelatedCharacters []string `json:"related_characters,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	TextLen           int      `json:"text_len,omitempty"`   // Content 的 rune 长度
	CreatedAt         string   `json:"created_at,omitempty"` // ISO8601
}

// StoryMemoryID 确定性记忆 ID（spec §12.2：<MMM>-<type>-<ordinal>）。
// 同章同类型重提取产出同 ID——幂等的前提；整章文件替换语义下先删后写同批完成。
func StoryMemoryID(chapterNum int, memType string, ordinal int) string {
	return fmt.Sprintf("%03d-%s-%d", chapterNum, memType, ordinal)
}

// StoryMemoryFile 单章记忆文件结构（memories/MMM-<n>-memory.json）。
type StoryMemoryFile struct {
	Items []StoryMemory `json:"items"`
}
