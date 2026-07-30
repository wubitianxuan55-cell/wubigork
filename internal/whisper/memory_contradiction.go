// Package whisper — memory_contradiction.go
// 100% 对齐 ackem memory/contradictionDetector.ts + prompt/memory-contradiction.ts
// 记忆矛盾检测：LLM 判断两条相似事实是否语义冲突，并建议解决策略

package whisper

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ─── ContradictionDetector ────────────────────────────────────

// ContradictionDetector 矛盾检测器
type ContradictionDetector struct{}

// NewContradictionDetector 创建检测器
func NewContradictionDetector() *ContradictionDetector {
	return &ContradictionDetector{}
}

// ContradictionCheck 矛盾检测结果
type ContradictionCheck struct {
	ConflictingFactID string `json:"conflictingFactId"`
	Judgment          string `json:"judgment"` // conflict/reinforce/unrelated
	Action            string `json:"action"`   // keep_new/keep_old/merge/flag
	Reason            string `json:"reason"`
}

// Check 检测单对新旧事实是否矛盾
func (cd *ContradictionDetector) Check(
	newFact, existingFact *MemoryFact,
	llm LlmClient,
) (*ContradictionCheck, error) {
	userPrompt := fmt.Sprintf(
		`旧事实：
  · 子类：%s
  · 主题：%s
  · 摘要：%s

新事实：
  · 子类：%s
  · 主题：%s
  · 摘要：%s`,
		existingFact.Subcategory, existingFact.Subject, existingFact.Summary,
		newFact.Subcategory, newFact.Subject, newFact.Summary,
	)

	raw, err := llm.Chat(contradictionSystemZH, userPrompt)
	if err != nil {
		return nil, err
	}

	return parseContradictionResult(raw, existingFact.ID), nil
}

// CheckBatch 批量检测多对事实（一次 LLM 调用）
func (cd *ContradictionDetector) CheckBatch(
	pairs []ContradictionPair,
	llm LlmClient,
) []*ContradictionCheck {
	if len(pairs) == 0 {
		return nil
	}

	var pairLines []string
	for i, p := range pairs {
		existingSummary := truncateStr(p.Existing.Summary, 120)
		newSummary := truncateStr(p.NewFact.Summary, 120)
		pairLines = append(pairLines, fmt.Sprintf(
			"[%d] 旧 · %s · %s：%s\n   新 · %s · %s：%s",
			i+1,
			p.Existing.Subcategory, p.Existing.Subject, existingSummary,
			p.NewFact.Subcategory, p.NewFact.Subject, newSummary,
		))
	}

	batchPrompt := fmt.Sprintf(
		`判断以下 %d 对事实的关系。每对按编号返回：
返回 JSON：{"pairs":[{"pair_idx":1,"judgment":"conflict|reinforce|unrelated","action":"keep_new|keep_old|merge|flag","reason":"..."}]}

%s`,
		len(pairs), strings.Join(pairLines, "\n\n"),
	)

	batchSystem := "你批量判断多对记忆事实之间的关系。对每对事实独立判断，只返回 JSON。"

	raw, err := llm.Chat(batchSystem, batchPrompt)
	if err != nil {
		results := make([]*ContradictionCheck, len(pairs))
		return results
	}

	return parseBatchContradictionResult(raw, pairs)
}

// ContradictionPair 一对需检测的事实
type ContradictionPair struct {
	NewFact  *MemoryFact
	Existing *MemoryFact
}

// ─── 解析 ─────────────────────────────────────────────────────

func parseContradictionResult(raw string, existingFactID string) *ContradictionCheck {
	tryParse := func(s string) (string, string, string) {
		var j struct {
			Judgment string `json:"judgment"`
			Action   string `json:"action"`
			Reason   string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(s), &j); err != nil {
			return "", "", ""
		}
		return j.Judgment, j.Action, j.Reason
	}

	trimmed := strings.TrimSpace(raw)
	judgment, action, reason := tryParse(trimmed)

	if judgment == "" {
		i := strings.Index(trimmed, "{")
		j := strings.LastIndex(trimmed, "}")
		if i >= 0 && j > i {
			judgment, action, reason = tryParse(trimmed[i : j+1])
		}
	}

	if judgment == "" {
		return nil
	}

	// 规范化 judgment
	validJudgments := map[string]bool{"conflict": true, "reinforce": true, "unrelated": true}
	if !validJudgments[judgment] {
		judgment = "unrelated"
	}

	// 规范化 action
	validActions := map[string]bool{"keep_new": true, "keep_old": true, "merge": true, "flag": true}
	if !validActions[action] {
		action = "keep_new"
	}

	conflictingID := ""
	if judgment == "conflict" {
		conflictingID = existingFactID
	}

	return &ContradictionCheck{
		ConflictingFactID: conflictingID,
		Judgment:          judgment,
		Action:            action,
		Reason:            reason,
	}
}

func parseBatchContradictionResult(raw string, pairs []ContradictionPair) []*ContradictionCheck {
	results := make([]*ContradictionCheck, len(pairs))

	trimmed := strings.TrimSpace(raw)
	i := strings.Index(trimmed, "{")
	j := strings.LastIndex(trimmed, "}")
	if i < 0 || j <= i {
		return results
	}

	var parsed struct {
		Pairs []struct {
			PairIdx  int    `json:"pair_idx"`
			Judgment string `json:"judgment"`
			Action   string `json:"action"`
			Reason   string `json:"reason"`
		} `json:"pairs"`
	}
	if err := json.Unmarshal([]byte(trimmed[i:j+1]), &parsed); err != nil {
		return results
	}

	resultMap := make(map[int]int) // pair_idx -> pairs index
	for idx, item := range parsed.Pairs {
		resultMap[item.PairIdx-1] = idx
	}

	for pi, p := range pairs {
		if itemIdx, ok := resultMap[pi]; ok {
			item := parsed.Pairs[itemIdx]
			validJudgments := map[string]bool{"conflict": true, "reinforce": true, "unrelated": true}
			judgment := item.Judgment
			if !validJudgments[judgment] {
				judgment = "unrelated"
			}
			validActions := map[string]bool{"keep_new": true, "keep_old": true, "merge": true, "flag": true}
			action := item.Action
			if !validActions[action] {
				action = "flag"
			}
			conflictingID := ""
			if judgment == "conflict" {
				conflictingID = p.Existing.ID
			}
			results[pi] = &ContradictionCheck{
				ConflictingFactID: conflictingID,
				Judgment:          judgment,
				Action:            action,
				Reason:            item.Reason,
			}
		}
	}

	return results
}

// ─── 矛盾检测 System Prompt ───────────────────────────────────

const contradictionSystemZH = `你判断两条记忆事实之间的关系。输入两条事实（来自同一个AIgaea对用户的记忆），输出它们的关系：

关系类型：
- "strong_conflict"：完全矛盾（"喜欢猫" vs "讨厌猫"）
- "weak_conflict"：部分矛盾（"喜欢安静" vs "昨天去酒吧玩得很开心"）
- "complement"：互补（"喜欢咖啡" + "每天喝美式" → 合并）
- "reinforce"：互相强化（"怕黑" + "晚上不敢关灯"）
- "unrelated"：关键词相似但实际不同（"喜欢猫" vs "喜欢猫主题的电影"）

对于 conflict，建议 action：
- "keep_new"：新事实更可信（旧事实可能是错误抽取或用户已改变）
- "keep_old"：旧事实更可靠（新事实可能是上下文误解）
- "merge"：两条都部分正确，合并摘要
- "flag"：不确定，标注让用户确认

判断时考虑：
- 同子类矛盾更可能是真矛盾
- 跨领域事实一般不判为 strong_conflict
- 旧事实超过 30 天，默认信任新事实
- 旧事实在 7 天内，默认信任旧事实
- 用户明确说"搞错了""我之前说错了" → keep_new

仅输出JSON：{"judgment":"...","action":"...","reason":"简短说明"}`
