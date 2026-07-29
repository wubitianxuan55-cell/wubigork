// Package whisper — semantic_reranker.go
// 100% 对齐 ackem memory/semanticReranker.ts
// LLM 语义重排序：对 TF-IDF 粗排结果用 LLM 精排打分

package whisper

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const rerankTemperature = 0.0

// SemanticReranker LLM 语义重排序器
type SemanticReranker struct{}

// NewSemanticReranker 创建重排序器
func NewSemanticReranker() *SemanticReranker { return &SemanticReranker{} }

// Rerank 对候选事实 LLM 精排
func (sr *SemanticReranker) Rerank(candidates []*Fact, query string, llm LlmClient, topK int) []*Fact {
	if len(candidates) <= 1 || topK <= 0 {
		if topK > 0 && len(candidates) > topK {
			return candidates[:topK]
		}
		return candidates
	}

	// 取前 20 个候选构建 prompt
	limit := 20
	if len(candidates) < limit {
		limit = len(candidates)
	}
	var items []string
	for _, f := range candidates[:limit] {
		items = append(items, fmt.Sprintf("ID:%s | [%s] %s：%s",
			f.ID, f.Subcategory, f.Subject, truncateStr(f.Summary, 100)))
	}

	raw, err := llm.Chat(rerankSystemPrompt, fmt.Sprintf(
		"用户消息：%s\n\n候选记忆：\n%s", query, strings.Join(items, "\n")))
	if err != nil {
		// fallback: 保持原始顺序
		if topK > 0 && len(candidates) > topK {
			return candidates[:topK]
		}
		return candidates
	}

	scores := parseRerankScores(raw)
	if len(scores) == 0 {
		if topK > 0 && len(candidates) > topK {
			return candidates[:topK]
		}
		return candidates
	}

	// 按分数排序
	scoreMap := make(map[string]float64)
	for _, s := range scores {
		scoreMap[s.ID] = s.Score
	}

	var reranked []*Fact
	for _, f := range candidates {
		if _, ok := scoreMap[f.ID]; ok {
			reranked = append(reranked, f)
		}
	}
	sort.Slice(reranked, func(i, j int) bool {
		return scoreMap[reranked[i].ID] > scoreMap[reranked[j].ID]
	})

	if topK > 0 && len(reranked) > topK {
		reranked = reranked[:topK]
	}
	return reranked
}

type rerankScore struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

func parseRerankScores(raw string) []rerankScore {
	trimmed := strings.TrimSpace(raw)
	i := strings.Index(trimmed, "[")
	j := strings.LastIndex(trimmed, "]")
	if i < 0 || j <= i {
		return nil
	}
	var scores []rerankScore
	if err := json.Unmarshal([]byte(trimmed[i:j+1]), &scores); err != nil {
		return nil
	}
	return scores
}

const rerankSystemPrompt = `你是一个记忆相关性裁判。用户说了一句话，系统检索到若干条候选记忆。你需要判断每条记忆与用户当前消息的语义相关性。

评分标准：
- 10：直接相关（用户正在谈论这个确切的主题）
- 7-9：高度相关（用户话题与记忆深层关联）
- 4-6：部分相关（某些关键词或主题重叠）
- 1-3：弱相关（勉强有联系）
- 0：完全无关

仅输出 JSON 数组，每条包含 id 和 score：
[{"id":"事实ID","score":8},{"id":"事实ID","score":3},...]
按 score 从高到低排序。`
