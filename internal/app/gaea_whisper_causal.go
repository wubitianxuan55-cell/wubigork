package app

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/whisper"
)

// causalEvidenceMaxLines 因果证据上限（防 prompt 膨胀）。
const causalEvidenceMaxLines = 8

// causalMaxHops 因果链最大边数：2 = 一因之因（A → 导致 → B → 导致 → X）。
const causalMaxHops = 2

// causalMinChainEdges 记为「链」的最小边数：单跳证据由单跳段承担，链只收
// ≥2 跳（对单跳纯重复）。
const causalMinChainEdges = 2

// causalMaxChains 多跳链上限：收益递减，超限后证据面回退单跳/关联补位。
const causalMaxChains = 4

// GaeaWhisperCausalExplain 跨事实因果推断（v4.9，审计 §C「推理仅邻接遍历」的
// 深度补口）：用户问「为什么<entity>」时，从图谱「导致」边收集多跳因果链
// （≤causalMaxHops 跳，含一因之因）+ event_chain 关联中涉及实体的事实对，
// 用当前人格口吻解释可能的因果链。只读：不改状态、不写记忆。无证据时不调
// LLM，直接返回诚实回退文案。
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
//  1. KG「导致」边的多跳因果链（≤causalMaxHops 边，链优先）；
//  2. 未被链覆盖的单跳「导致」三元组（实体在因/果侧）；
//  3. event_chain 关联中涉及实体的事实对。
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
		var causal []whisper.Triple
		for _, t := range kg.ListAll() {
			if t.Predicate == "导致" {
				causal = append(causal, t)
			}
		}
		chains, chained := buildCausalChains(entity, causal)
		for _, c := range chains {
			addLine("记忆链：" + strings.Join(c, " → 导致 → "))
		}
		for i, t := range causal {
			if len(lines) >= causalEvidenceMaxLines {
				break
			}
			if chained[i] {
				continue
			}
			if !containsEntity(t.Subject, entity) && !containsEntity(t.Object, entity) {
				continue
			}
			addLine("记忆：" + t.Subject + " → 导致 → " + t.Object)
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

// buildCausalChains 以 entity 为起点在「导致」边集合上做有界 DFS（≤causalMaxHops
// 边，双向：顺藤摸瓜找后果、顺因溯源找根源），收集节点数 ≥2 边的因果链。
// 返回链（节点按因果序：因 →…→ 果）与被链覆盖的三元组下标（调用方对单跳段
// 去重）。防环双重保险：路径内边不重走 + 链内节点不重访；链条数 ≤causalMaxChains。
func buildCausalChains(entity string, triples []whisper.Triple) ([][]string, map[int]bool) {
	var chains [][]string
	chainSeen := map[string]bool{}
	used := map[int]bool{}
	start := strings.TrimSpace(entity)

	var walk func(path []string, cur string, depth int, pathIdx map[int]bool)
	walk = func(path []string, cur string, depth int, pathIdx map[int]bool) {
		if depth >= causalMaxHops || len(chains) >= causalMaxChains {
			return
		}
		for i, t := range triples {
			if pathIdx[i] {
				continue
			}
			next, forward, ok := causalStep(t, cur)
			if !ok || sameNode(next, cur) {
				continue
			}
			dup := false
			for _, n := range path {
				if sameNode(n, next) {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			idx := map[int]bool{i: true}
			for k := range pathIdx {
				idx[k] = true
			}
			var newPath []string
			if forward { // cur 在因侧：因果序追加到尾部
				newPath = append(append([]string{}, path...), next)
			} else { // cur 在果侧：对侧是更早的因，前插保持因果序
				newPath = append([]string{next}, path...)
			}
			if depth+1 >= causalMinChainEdges {
				key := strings.Join(newPath, "\x00")
				if !chainSeen[key] {
					chainSeen[key] = true
					chains = append(chains, newPath)
					for k := range idx {
						used[k] = true
					}
				}
				if len(chains) >= causalMaxChains {
					return
				}
			}
			if depth+1 < causalMaxHops {
				walk(newPath, next, depth+1, idx)
			}
		}
	}
	if start != "" {
		walk([]string{start}, start, 0, map[int]bool{})
	}
	return chains, used
}

// causalStep 判断三元组 t 是否与 cur 相邻：返回（对侧文本, cur 是否在因侧,
// 是否相邻）。两侧同时命中视为退化，不展开。
func causalStep(t whisper.Triple, cur string) (string, bool, bool) {
	inS := containsEntity(t.Subject, cur)
	inO := containsEntity(t.Object, cur)
	switch {
	case inS && !inO:
		return t.Object, true, true
	case inO && !inS:
		return t.Subject, false, true
	}
	return "", false, false
}

// sameNode 链节点等值判断（精确等值，防环用；邻接匹配仍走 containsEntity）。
func sameNode(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
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
			"1. 只使用证据里的内容，不要编造新记忆；证据里的「记忆链」是跨多步的因果链"+
			"（A → 导致 → B → 导致 → C），解释时可以把它串成一段完整的故事；\n"+
			"2. 如果证据不足以回答，就诚实说「我还不完全清楚，但…」并给出最可能的推测；\n"+
			"3. 不要复述「证据」「记忆」等字段名，像聊天一样说出来。",
		name, entity,
	)
	if preset.VoiceGuide != "" {
		prompt += "\n\n语气参考：" + preset.VoiceGuide
	}
	return prompt
}
