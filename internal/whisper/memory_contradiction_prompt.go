// Package whisper — memory_contradiction_prompt.go
// 100% 对齐 ackem prompt/memory-contradiction.ts
// 矛盾检测 prompt：新旧事实关系判断

package whisper

// ContradictionTemperature 矛盾检测 LLM 温度
const ContradictionTemperature = 0.1

// ContradictionSystemPrompt 矛盾检测 system prompt
// 100% 对齐 ackem CONTRADICTION_SYSTEM_ZH
const ContradictionSystemPrompt = `你判断两条记忆事实之间的关系。输入两条事实（来自同一个AI伴侣对用户的记忆），输出它们的关系：

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

// ContradictionFactPair 矛盾检测的事实对
type ContradictionFactPair struct {
	Subcategory string
	Subject     string
	Summary     string
}

// BuildContradictionPrompt 构建矛盾检测 user prompt
func BuildContradictionPrompt(newFact, existingFact ContradictionFactPair) string {
	return `旧事实：
  · 子类：` + existingFact.Subcategory + `
  · 主题：` + existingFact.Subject + `
  · 摘要：` + existingFact.Summary + `

新事实：
  · 子类：` + newFact.Subcategory + `
  · 主题：` + newFact.Subject + `
  · 摘要：` + newFact.Summary
}
