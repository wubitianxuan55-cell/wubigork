package app

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/whisper"
)

// causalEvidenceMaxLines 因果证据上限（防 prompt 膨胀）。
const causalEvidenceMaxLines = 8

// GaeaWhisperCausalExplain 跨事实因果推断（v4.9，审计 §C「推理仅邻接遍历」的
// 深度补口）：用户问「为什么<entity>」时，从图谱「导致」三元组 + event_chain
// 关联中收集证据，用当前人格口吻解释可能的因果链。只读：不改状态、不写记忆。
// 无证据时不调 LLM，直接返回诚实回退文案。
func (a *whisperState) GaeaWhisperCausalExplain(entity, personalityID string) (string, error) {
	entity = strings.TrimSpace(entity)
	if entity == "" {
		return "", fmt.Errorf("entity 为空")
	}
	orch := a.getOrCreateOrch(personalityID)
	if orch == nil {
		return "", fmt.Errorf("轻语会话未就绪")
	}
	evidence := buildCausalEvidence(entity, orch.KG, orch.AssocIndex, orch.FactStore)
	if evidence == "" {
		return fmt.Sprintf("关于「%s」，我还没有足够的记忆来推断因果。多和我聊聊相关的事，我会慢慢记住，之后就能陪你一起想明白了。", entity), nil
	}
	if a.client == nil {
		return "", fmt.Errorf("model client not initialized")
	}

	featEng, featModel := a.featureModel("chat")
	engine := orch.EngineID
	if featEng != "" {
		engine = featEng
	}
	model := orch.ModelName
	if featModel != "" {
		model = featModel
	}
	if model == "" {
		return "", fmt.Errorf("未绑定模型")
	}

	reply, _, err := a.client.ChatSimpleStreamDetailed(
		a.ctx, model, buildCausalExplainSystemPrompt(orch.Preset, entity), evidence,
		ai.ChatSimpleOptions{EngineID: engine})
	if err != nil {
		return "", fmt.Errorf("因果解释生成失败: %w", err)
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "", fmt.Errorf("因果解释生成为空")
	}
	return reply, nil
}

// buildCausalEvidence 汇总实体相关因果证据（确定性、有上限）：
//  1. KG「导致」三元组（实体出现在因/果侧）；
//  2. event_chain 关联中涉及实体的事实对。
func buildCausalEvidence(entity string, kg *whisper.KnowledgeGraph, ai *whisper.AssociationIndex, fs *whisper.FactStore) string {
	var lines []string
	seen := map[string]bool{}
	addLine := func(l string) {
		if l == "" || seen[l] {
			return
		}
		seen[l] = true
		if len([]rune(l)) > 80 {
			l = string([]rune(l)[:80]) + "…"
		}
		lines = append(lines, l)
	}

	if kg != nil {
		for _, t := range kg.ListAll() {
			if t.Predicate != "导致" {
				continue
			}
			if !containsEntity(t.Subject, entity) && !containsEntity(t.Object, entity) {
				continue
			}
			addLine("记忆：" + t.Subject + " → 导致 → " + t.Object)
			if len(lines) >= causalEvidenceMaxLines {
				break
			}
		}
	}
	if ai != nil && fs != nil {
		for _, a := range ai.ListAll() {
			if a.AssociationType != "event_chain" || len(lines) >= causalEvidenceMaxLines {
				continue
			}
			fA := fs.Get(a.FactIDA)
			fB := fs.Get(a.FactIDB)
			if fA == nil || fB == nil {
				continue
			}
			if !containsEntity(fA.Subject, entity) && !containsEntity(fB.Subject, entity) {
				continue
			}
			addLine(fmt.Sprintf("关联：%s（%s）→ %s（%s）",
				fA.Subject, truncateRunes(fA.Summary, 40),
				fB.Subject, truncateRunes(fB.Summary, 40)))
		}
	}
	return strings.Join(lines, "\n")
}

// containsEntity 双向包含判断（实体可能是长句的子串，反之亦然）。
func containsEntity(s, entity string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	e := strings.ToLower(strings.TrimSpace(entity))
	return s != "" && e != "" && (strings.Contains(s, e) || strings.Contains(e, s))
}

// buildCausalExplainSystemPrompt 人格口吻系统提示：只用证据、不编造、证据不足
// 诚实说明、像聊天一样说出来（≤200 字）。
func buildCausalExplainSystemPrompt(preset whisper.PersonalityPreset, entity string) string {
	name := preset.Label
	if name == "" {
		name = "gaea"
	}
	prompt := fmt.Sprintf(
		"你是 %s，一位陪伴用户的 AI。用户在问「为什么%s」。\n"+
			"请基于下方记忆证据，用第一人称、温柔自然的口吻解释可能的因果链条（中文，200 字以内）。规则：\n"+
			"1. 只使用证据里的内容，不要编造新记忆；\n"+
			"2. 如果证据不足以回答，就诚实说「我还不完全清楚，但…」并给出最可能的推测；\n"+
			"3. 不要复述「证据」「记忆」等字段名，像聊天一样说出来。",
		name, entity,
	)
	if preset.VoiceGuide != "" {
		prompt += "\n\n语气参考：" + preset.VoiceGuide
	}
	return prompt
}
