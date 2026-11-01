// Package characterlib — 全局统一角色库
// 角色是独立资产：不属于任何一本小说，也不专属于聊天。
// 小说通过项目关联引用角色；聊天把角色直接当人格使用。
package characterlib

import (
	"strings"

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
	// v4.3g 参考图/画廊图：JSON []string（data URL / 远程 URL / 本地落盘路径，
	// 保存时 data URL 与远程 URL 会本地化为 portraits 目录下的文件路径）。
	ReferenceImages []string `json:"referenceImages,omitempty"`
	GalleryImages   []string `json:"galleryImages,omitempty"`
	// v4.411 一致性评分写回：与 ReferenceImages 索引对齐的分数（0-100，
	// 0=未评分不渲染徽标）。参考图增删/超限剔除时由读写两路同步维护对齐。
	ReferenceScores []int `json:"referenceScores,omitempty"`

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
	Dims          whisper.PersonalityDims  `json:"dims"`
	VoiceGuide    string                   `json:"voiceGuide,omitempty"`
	BehaviorRules string                   `json:"behaviorRules,omitempty"`
	EmotionLogic  string                   `json:"emotionLogic,omitempty"`
	HiddenPersona *whisper.PersonalityDims `json:"hiddenPersona,omitempty"`
	// 角色音色预设（TTS）：voiceId=breeze 克隆音色 id（空=design 模式），
	// desc=音色描述。两字段独立回退（合成侧 voicePersonaPreset），均空时走
	// 全局缺省链（gender 缺省描述 > 全局 breeze 描述）。
	TTSVoiceID string `json:"ttsVoiceId,omitempty"`
	TTSDesc    string `json:"ttsDesc,omitempty"`
	// 专属语音台词：档案试听段念的台词（角色签名声线）。空时试听回退
	// 对话样本首条 > 姓名问候（前端 resolveVoiceLine 同口径）。
	TTSLine string `json:"ttsLine,omitempty"`

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
	// 人格厚度全量进聊天：性格/背景是「按人设思考」的主料，口吻是语气层，
	// 行为规则/情感逻辑是行为约束，对话样本给语气实例——全部折叠进
	// VoiceGuide 单一注入点（系统提示词人格段的唯一来源）。
	voice := c.VoiceGuide
	if voice == "" {
		voice = "你是「" + c.Name + "」：措辞与态度须贯穿此人设，勿写成通用温柔助手或百科客服。"
	}
	var persona string
	if p := strings.TrimSpace(c.Personality); p != "" {
		persona += "\n性格：" + p
	}
	if b := strings.TrimSpace(c.Background); b != "" {
		persona += "\n背景：" + b
	}
	if persona != "" {
		voice += "\n【人设】" + persona
	}
	if c.BehaviorRules != "" {
		voice += "\n行为规则：" + c.BehaviorRules
	}
	if c.EmotionLogic != "" {
		voice += "\n情感逻辑：" + c.EmotionLogic
	}
	if n := len(c.DialogueSamples); n > 0 {
		max := n
		if max > 3 {
			max = 3
		}
		voice += "\n对话样本（语气实例，仿其风格）：\n- " + strings.Join(c.DialogueSamples[:max], "\n- ")
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
