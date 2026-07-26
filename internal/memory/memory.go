package memory

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/wubigork/wubigork/internal/project"
	"github.com/wubigork/wubigork/internal/util"
)

// ── 语义记忆（BM25 检索，零外部依赖）─────────────────────────

// Memory 一条可检索的故事记忆
type Memory struct {
	ID         string  `json:"id"`
	ChapterNum int     `json:"chapter_num"`
	Text       string  `json:"text"`       // 记忆内容
	Category   string  `json:"category"`   // event / character / world / plot
	Tokens     int     `json:"tokens"`     // 估算 token 数
	Score      float64 `json:"score"`      // BM25 相关度分数（检索时填充）
}

// Index BM25 索引
type Index struct {
	memories    []Memory
	docFreq     map[string]int     // 词 → 包含该词的文档数
	docLengths  []int              // 每个文档的长度
	avgDocLen   float64
	k1          float64 // BM25 参数
	b           float64
}

// NewIndex 创建 BM25 索引
func NewIndex() *Index {
	return &Index{
		docFreq: make(map[string]int),
		k1:      1.5,
		b:       0.75,
	}
}

// tokenize 中文友好分词：按 Unicode 类别切分
func tokenize(text string) []string {
	var tokens []string
	var current []rune

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
		} else {
			if len(current) > 0 {
				tokens = append(tokens, strings.ToLower(string(current)))
				current = nil
			}
			if !unicode.IsSpace(r) && !unicode.IsPunct(r) {
				tokens = append(tokens, strings.ToLower(string(r)))
			}
		}
	}
	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}

	// 对中文做 2-gram 增强（捕获双字词组）
	runes := []rune(strings.ToLower(text))
	for i := 0; i < len(runes)-1; i++ {
		if unicode.Is(unicode.Han, runes[i]) && unicode.Is(unicode.Han, runes[i+1]) {
			tokens = append(tokens, string(runes[i:i+2]))
		}
	}

	return tokens
}

// BuildFromProject 从项目构建 BM25 索引
func BuildFromProject(pm *project.Manager) (*Index, error) {
	idx := NewIndex()

	for chapterNum := 1; ; chapterNum++ {
		summary, err := pm.ReadChapterSummary(chapterNum)
		if err != nil {
			break
		}
		if summary == nil {
			continue
		}

		text := summary.Summary
		if text == "" {
			continue
		}

		mem := Memory{
			ID:         summary.Title,
			ChapterNum: chapterNum,
			Text:       text,
			Category:   "event",
			Tokens:     util.EstimateTokens(text),
		}

		idx.Add(mem)
	}

	return idx, nil
}

// Add 添加一条记忆到索引
func (idx *Index) Add(m Memory) {
	tokens := tokenize(m.Text)
	seen := make(map[string]bool)

	for _, t := range tokens {
		if !seen[t] {
			idx.docFreq[t]++
			seen[t] = true
		}
	}

	idx.memories = append(idx.memories, m)
	idx.docLengths = append(idx.docLengths, len(tokens))

	total := 0
	for _, l := range idx.docLengths {
		total += l
	}
	if len(idx.docLengths) > 0 {
		idx.avgDocLen = float64(total) / float64(len(idx.docLengths))
	}
}

// Search 检索最相关的 K 条记忆
func (idx *Index) Search(query string, k int) []Memory {
	if len(idx.memories) == 0 {
		return nil
	}

	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	N := float64(len(idx.memories))

	type scored struct {
		mem   Memory
		score float64
	}

	var results []scored

	for i, mem := range idx.memories {
		docTokens := tokenize(mem.Text)
		tf := make(map[string]int)
		for _, t := range docTokens {
			tf[t]++
		}

		score := 0.0
		for _, qt := range queryTokens {
			f := float64(tf[qt])
			if f == 0 {
				continue
			}
			df := float64(idx.docFreq[qt])
			if df == 0 {
				df = 0.5 // 平滑
			}

			// BM25 公式
			idf := math.Log(1 + (N-df+0.5)/(df+0.5))
			numerator := f * (idx.k1 + 1)
			denominator := f + idx.k1*(1-idx.b+idx.b*float64(idx.docLengths[i])/idx.avgDocLen)
			score += idf * numerator / denominator
		}

		if score > 0 {
			results = append(results, scored{mem: mem, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if k > len(results) {
		k = len(results)
	}

	var topK []Memory
	for i := 0; i < k; i++ {
		m := results[i].mem
		m.Score = math.Round(results[i].score*100) / 100
		topK = append(topK, m)
	}

	return topK
}

// InjectIntoContext 将相关记忆注入到 AI 上下文字符串末尾
// 返回注入后的上下文和注入的记忆列表
func (idx *Index) InjectIntoContext(currentContext string, maxMemories int, maxTokens int) (string, []Memory) {
	memories := idx.Search(currentContext, maxMemories)
	if len(memories) == 0 {
		return currentContext, nil
	}

	var parts []string
	var injected []Memory
	tokenUsed := 0

	parts = append(parts, currentContext)
	parts = append(parts, "\n\n## 相关历史记忆（按相关度排序）")

	for _, m := range memories {
		if tokenUsed+m.Tokens > maxTokens {
			break
		}
		parts = append(parts, formatMemory(m))
		tokenUsed += m.Tokens
		injected = append(injected, m)
	}

	return strings.Join(parts, "\n"), injected
}

func formatMemory(m Memory) string {
	return strings.Join([]string{
		"---",
		"第" + strings.TrimPrefix(m.ID, "第 ") + "",
		"相关度: " + formatScore(m.Score),
		m.Text,
	}, "\n")
}

func formatScore(s float64) string {
	if s >= 5 {
		return "⭐⭐⭐ 高"
	} else if s >= 3 {
		return "⭐⭐ 中"
	}
	return "⭐ 低"
}

