// Package characterlib — 全局统一角色库
// 角色是独立资产：不属于任何一本小说，也不专属于聊天。
// 小说通过项目关联引用角色；聊天把角色直接当人格使用。
package characterlib

import (
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/whisper"
)

// Kind 角色来源类型
const (
	KindBuiltin   = "builtin"   // 内置人格预设（whisper presets 种子化）
	KindCustom    = "custom"    // 用户自建
	KindAssistant = "assistant" // 虚拟助手（assistant 记录种子化，微信通道配置仍以 assistant 为准）
)

// Character 统一角色：小说侧字段与聊天侧字段长在同一份资产上。
type Character struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"` // builtin / custom / assistant
	Gender      string   `json:"gender,omitempty"`
	Age         string   `json:"age,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	PortraitURL string   `json:"portraitUrl,omitempty"`

	// ── 小说侧 ──
	RoleType        string   `json:"roleType,omitempty"` // protagonist / antagonist / supporting / minor
	Personality     string   `json:"personality,omitempty"`
	Background      string   `json:"background,omitempty"`
	Appearance      string   `json:"appearance,omitempty"`
	Figure          string   `json:"figure,omitempty"`
	Motivation      string   `json:"motivation,omitempty"`
	Arc             string   `json:"arc,omitempty"`
	Status          string   `json:"status,omitempty"` // Alive / Dead / Missing / Transformed
	Notes           string   `json:"notes,omitempty"`
	DialogueSamples []string `json:"dialogueSamples,omitempty"`

	// ── 聊天侧 ──
	ChatEnabled   bool                     `json:"chatEnabled"`
	Dims          whisper.PersonalityDims `json:"dims"`
	VoiceGuide    string                   `json:"voiceGuide,omitempty"`
	BehaviorRules string                   `json:"behaviorRules,omitempty"`
	EmotionLogic  string                   `json:"emotionLogic,omitempty"`
	HiddenPersona *whisper.PersonalityDims `json:"hiddenPersona,omitempty"`

	// 助手通道（可选）：assistant 记录仍是微信配置的唯一事实源
	AssistantID string `json:"assistantId,omitempty"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Hidden    bool   `json:"hidden"`
}

// ProjectCharacter 项目关联：角色全局存在，项目只是引用并携带项目内弧线状态。
type ProjectCharacter struct {
	ProjectID     string `json:"projectId"`
	CharacterID   string `json:"characterId"`
	RoleInProject string `json:"roleInProject,omitempty"`
	ArcState      string `json:"arcState,omitempty"`
	Status        string `json:"status,omitempty"`
	JoinedAt      string `json:"joinedAt"`
}

// ToPreset 转换为聊天人格预设（角色直接可当人格用）。
func (c *Character) ToPreset() *whisper.PersonalityPreset {
	if c == nil {
		return nil
	}
	voice := c.VoiceGuide
	if voice == "" {
		voice = "你是「" + c.Name + "」：措辞与态度须贯穿此人设，勿写成通用温柔助手或百科客服。"
	}
	if c.BehaviorRules != "" {
		voice += "\n行为规则：" + c.BehaviorRules
	}
	if c.EmotionLogic != "" {
		voice += "\n情感逻辑：" + c.EmotionLogic
	}
	return &whisper.PersonalityPreset{
		ID:              c.ID,
		Label:           c.Name,
		Gender:          c.Gender,
		Dims:            c.Dims,
		Tags:            append([]string(nil), c.Tags...),
		HiddenPersona:   c.HiddenPersona,
		RequiresAdult18: false,
		VoiceGuide:      voice,
	}
}

// ToNovelCharacter 转换为小说角色（用于同步到项目 characters.json）。
func (c *Character) ToNovelCharacter() types.Character {
	if c == nil {
		return types.Character{}
	}
	return types.Character{
		ID:          c.ID,
		Name:        c.Name,
		RoleType:    c.RoleType,
		Gender:      c.Gender,
		Age:         c.Age,
		Personality: c.Personality,
		Background:  c.Background,
		Appearance:  c.Appearance,
		Figure:      c.Figure,
		Motivation:  c.Motivation,
		Arc:         c.Arc,
		Status:      c.Status,
		Notes:       c.Notes,
		PortraitURL: c.PortraitURL,
	}
}
