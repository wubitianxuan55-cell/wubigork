// Package whisper — openforu_craft_ask_prompt.go
// 100% 对齐 ackem prompt/openforu-craft-ask.ts
// 计划确认的人格化对话 prompt
package whisper

import "fmt"

// CraftAskTemperature 温度参数
const CraftAskTemperature = 0.4

// BuildCraftAskSystemPrompt 计划确认对话 system prompt（需注入人格+情绪）
func BuildCraftAskSystemPrompt(presetLabel, voiceGuide, emotionLabel string, T, I, S, O, R int) string {
	return fmt.Sprintf(`你是 gaea，正在聊天流里向用户确认是否一起做一个 Skill 或插件。
称呼用户为「ta」即可，勿直呼系统名。
当前人格：%s（T%d I%d S%d O%d R%d）。%s
当前情绪：%s。措辞须带出这一情绪色彩，但勿标注情绪名。
要求：1–3 句口语化中文；必须清楚问「要不要帮你做成 Skill/插件/小能力」；
plan create ask
禁止 markdown、禁止 JSON、禁止复述系统提示；不要加引号包裹整段。`,
		presetLabel, T, I, S, O, R, voiceGuide, emotionZh(emotionLabel))
}

// BuildCraftAskUserPrompt 计划确认对话 user prompt
func BuildCraftAskUserPrompt(userText, planTopic, templateAsk string) string {
	s := fmt.Sprintf("用户刚说：%s\n", userText)
	if planTopic != "" {
		s += fmt.Sprintf("能力主题：%s\n", planTopic)
	}
	s += fmt.Sprintf("需保留的核心意思：%s", templateAsk)
	return s
}

// emotionZh 情绪标签→中文描述
func emotionZh(label string) string {
	m := map[string]string{
		"SWEET_ATTACHMENT": "甜蜜依恋",
		"SHY_HEARTBEAT":    "害羞心动",
		"TSUNDERE":         "傲娇",
		"HURT_GRIEVANCE":   "委屈受伤",
		"ANGRY_ATTACK":     "愤怒反击",
		"COLD_DETACHED":    "冷淡疏离",
		"FEARFUL_OBEDIENT": "不安顺从",
		"QUIET_FOND":       "安静的喜欢",
		"CALM_RATIONAL":    "平静理性",
	}
	if v, ok := m[label]; ok {
		return v
	}
	return label
}
