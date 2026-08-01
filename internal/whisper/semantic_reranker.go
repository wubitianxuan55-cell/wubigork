// Package whisper — semantic_reranker.go
// 100% 对齐 ackem memory/semanticReranker.ts
// LLM 语义重排序：对粗排候选记忆做精排打分

package whisper

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ─── 常量 ──────────────────────────────────────────────────────

const maxRerankCandidates = 20

const rerankSystemPrompt = `你是一个记忆相关性裁判。用户说了一句话，系统检索到若干条候选记忆。你需要判断每条记忆与用户当前消息的语义相关性。

评分标准：
- 10：直接相关（用户正在谈论这个确切的主题）
- 7-9：高度相关（用户话题与记忆深层关联）
- 4-6：部分相关（某些关键词或主题重叠）
- 1-3：弱相关（勉强有联系）
- 0：完全无关

仅输出 JSON 数组，每条包含 factId 和 score：
[{"id":"事实ID","score":8},{"id":"事实ID","score":3},...]
按 score 从高到低排序。`

// ─── 重排序器 ──────────────────────────────────────────────────

// SemanticReranker LLM 语义重排序器
type SemanticReranker struct{}

// rerankScore LLM 返回的评分
type rerankScore struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

// Rerank 对候选记忆做语义精排
// candidates: 粗排候选（最多20条）
// query: 用户当前消息
// llmCall: LLM调用函数 (system, user → jsonString, error)
// topK: 最终返回条数
func (r *SemanticReranker) Rerank(
	candidates []MemoryFact,
	query string,
	llmCall func(system, user string) (string, error),
	topK int,
) []MemoryFact {
	if len(candidates) <= 1 {
		return candidates
	}
	if topK <= 0 {
		topK = 6
	}

	// 最多取前20条
	pool := candidates
	if len(pool) > maxRerankCandidates {
		pool = pool[:maxRerankCandidates]
	}

	// 构建候选列表
	var items []string
	for _, f := range pool {
		summary := f.Summary
		if len([]rune(summary)) > 100 {
			summary = string([]rune(summary)[:100])
		}
		items = append(items, fmt.Sprintf("ID:%s | [%s] %s：%s",
			f.ID, f.Subcategory, f.Subject, summary))
	}
	itemsText := strings.Join(items, "\n")

	// 调用 LLM
	userPrompt := fmt.Sprintf("用户消息：%s\n\n候选记忆：\n%s", query, itemsText)
	raw, err := llmCall(rerankSystemPrompt, userPrompt)
	if err != nil {
		// LLM 失败 → 保持粗排顺序
		if len(candidates) > topK {
			return candidates[:topK]
		}
		return candidates
	}

	// 解析评分
	var scores []rerankScore
	if err := json.Unmarshal([]byte(raw), &scores); err != nil {
		// 解析失败 → 回退
		if len(candidates) > topK {
			return candidates[:topK]
		}
		return candidates
	}

	if len(scores) == 0 {
		if len(candidates) > topK {
			return candidates[:topK]
		}
		return candidates
	}

	// 按分数排序
	scoreMap := make(map[string]float64)
	for _, s := range scores {
		scoreMap[s.ID] = s.Score
	}

	// 筛选有评分的候选
	var scored []MemoryFact
	for _, f := range candidates {
		if _, ok := scoreMap[f.ID]; ok {
			scored = append(scored, f)
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scoreMap[scored[i].ID] > scoreMap[scored[j].ID]
	})

	if len(scored) > topK {
		scored = scored[:topK]
	}

	if len(scored) == 0 && len(candidates) > 0 {
		if len(candidates) > topK {
			return candidates[:topK]
		}
		return candidates
	}

	return scored
}
