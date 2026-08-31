package app

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/whisper"
)

// GaeaWhisperMemoryRetell 让 gaea 以当前人格口吻把一段记忆「重述成故事」
// （LLM 叙事重述，区别于确定性原文回放）。kind: episode | anchor；id 为对应
// 记忆 ID；personalityID 决定人格口吻与模型绑定。只读：不改状态、不写记忆。
func (a *whisperState) GaeaWhisperMemoryRetell(kind, id, personalityID string) (string, error) {
	orch := a.getOrCreateOrch(personalityID)
	if orch == nil {
		return "", fmt.Errorf("轻语会话未就绪")
	}
	if a.client == nil {
		return "", fmt.Errorf("model client not initialized")
	}
	contextText, err := a.resolveRetellMemory(kind, id)
	if err != nil {
		return "", err
	}

	// v4.16 离线裂缝收口：与 plain 聊天同源走 routeModel（离线过滤 + 全局/兜底一致）。
	engine, model, _ := a.routeModel("chat")
	if model == "" {
		return "", fmt.Errorf("未绑定模型")
	}

	reply, _, err := a.client.ChatSimpleStreamDetailed(
		a.ctx, model, buildMemoryRetellSystemPrompt(orch.Preset), contextText,
		ai.ChatSimpleOptions{EngineID: engine})
	if err != nil {
		return "", fmt.Errorf("重述生成失败: %w", err)
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "", fmt.Errorf("重述生成为空")
	}
	return reply, nil
}

// resolveRetellMemory 解析记忆上下文：复用确定性回放（buildEpisodeReplay/
// GaeaWhisperAnchorReplay），保证重述基于真实原文而非仅有摘要。
func (a *whisperState) resolveRetellMemory(kind, id string) (string, error) {
	switch kind {
	case "episode":
		view, err := a.app.GaeaWhisperEpisodeReplay(id)
		if err != nil {
			return "", err
		}
		return buildMemoryRetellContext(view.Summary, view.DominantEmotion, view.Dialogue), nil
	case "anchor":
		view, err := a.app.GaeaWhisperAnchorReplay(id)
		if err != nil {
			return "", err
		}
		dialogue := []WhisperReplayLine{}
		emotion := ""
		if view.EpisodeReplay != nil {
			dialogue = view.EpisodeReplay.Dialogue
			emotion = view.EpisodeReplay.DominantEmotion
		}
		return buildMemoryRetellContext(view.Summary, emotion, dialogue), nil
	default:
		return "", fmt.Errorf("未知记忆类型: %s（支持 episode/anchor）", kind)
	}
}

// buildMemoryRetellContext 组装重述输入：摘要 + 情绪 + 原始对话（对话可为空，
// 重述将退化为基于摘要的叙述）。
func buildMemoryRetellContext(summary, emotion string, dialogue []WhisperReplayLine) string {
	var sb strings.Builder
	sb.WriteString("【记忆摘要】\n")
	sb.WriteString(summary)
	if emotion != "" {
		sb.WriteString("\n\n【当时的情绪】\n")
		sb.WriteString(emotion)
	}
	if len(dialogue) > 0 {
		sb.WriteString("\n\n【当时的对话】\n")
		for _, line := range dialogue {
			role := "gaea"
			if line.Role == "user" {
				role = "对方"
			}
			sb.WriteString(role + "：" + line.Text + "\n")
		}
	}
	return sb.String()
}

// buildMemoryRetellSystemPrompt 人格口吻系统提示：第一人称重述、别用「用户」、
// ≤300 字、不复述字段名、结尾带当下感受。
func buildMemoryRetellSystemPrompt(preset whisper.PersonalityPreset) string {
	name := preset.Label
	if name == "" {
		name = "gaea"
	}
	prompt := fmt.Sprintf(
		"你是 %s，一位陪伴用户的 AI。现在用户想重温一段你们共同经历的记忆。\n"+
			"请以你的口吻、用第一人称把这段记忆重述成一段温暖自然的叙述（中文，300 字以内），"+
			"像老友翻开旧相册那样讲出来——不要用「用户」称呼对方，直接说「你」；"+
			"不要复述任何系统字段名；结尾可以自然带一句当下的感受。",
		name,
	)
	if preset.VoiceGuide != "" {
		prompt += "\n\n语气参考：" + preset.VoiceGuide
	}
	return prompt
}
