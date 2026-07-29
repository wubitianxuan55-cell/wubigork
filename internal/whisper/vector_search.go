// Package whisper — vector_search.go
// 对齐 ackem memory/vectorStore.ts + memory/semanticSearch.ts + memory/factEmbeddingCache.ts
// 向量存储与语义搜索：TF-IDF 稀疏向量 + 稠密向量 + Jaccard 字符集 + 余弦相似度

package whisper

import (
	"math"
	"sort"
	"strings"
)

// ─── 余弦相似度 ─────────────────────────────────────────────────

// CosineSimilarity 余弦相似度（对齐 ackem factEmbeddingCache.ts）
func CosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, normA, normB float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

// ─── TF-IDF 向量 ───────────────────────────────────────────────

// TFIDFVector TF-IDF 稀疏向量
type TFIDFVector map[string]float64

// TfidfIndex TF-IDF 索引
type TfidfIndex struct {
	vectors   []TFIDFVector     // 每个文档的向量
	docIDs    []string          // 文档 ID
	idfCache  map[string]float64 // 词 → IDF
	docCount  int
}

// NewTfidfIndex 创建 TF-IDF 索引
func NewTfidfIndex() *TfidfIndex {
	return &TfidfIndex{idfCache: make(map[string]float64)}
}

// tokenizeForTFIDF 中文分词（≥2字符词 + CJK bigram）
func tokenizeForTFIDF(text string) []string {
	var tokens []string
	seen := make(map[string]bool)

	// CJK 标点分割
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == '，' || r == '。' || r == '！' || r == '？' || r == '、' ||
			r == '；' || r == '：' || r == ' ' || r == '\t' || r == '\n' ||
			r == '“' || r == '”' || r == '"' || r == '（' || r == '）' ||
			r == '【' || r == '】' || r == '.' || r == ',' || r == '!' || r == '?'
	})
	for _, f := range fields {
		f = strings.TrimSpace(f)
		runes := []rune(f)
		if len(runes) >= 2 {
			if !seen[f] {
				seen[f] = true
				tokens = append(tokens, f)
			}
		}
		// bigram
		for i := 0; i < len(runes)-1; i++ {
			bigram := string(runes[i : i+2])
			if !seen[bigram] {
				seen[bigram] = true
				tokens = append(tokens, bigram)
			}
		}
	}
	return tokens
}

// Build 构建索引（批量添加文档）
func (idx *TfidfIndex) Build(docs []struct{ ID, Text string }) {
	idx.vectors = make([]TFIDFVector, len(docs))
	idx.docIDs = make([]string, len(docs))
	idx.docCount = len(docs)

	// 统计 DF
	df := make(map[string]int)
	for i, d := range docs {
		idx.docIDs[i] = d.ID
		tokens := tokenizeForTFIDF(d.Text)
		tf := make(map[string]int)
		for _, t := range tokens {
			tf[t]++
		}
		idx.vectors[i] = make(TFIDFVector)
		maxTF := 0
		for _, c := range tf {
			if c > maxTF {
				maxTF = c
			}
		}
		for t, c := range tf {
			idx.vectors[i][t] = float64(c) / float64(maxTF)
		}
		// 统计文档频次
		seen := make(map[string]bool)
		for t := range tf {
			if !seen[t] {
				seen[t] = true
				df[t]++
			}
		}
	}

	// 计算 IDF
	N := float64(idx.docCount)
	for t, d := range df {
		idx.idfCache[t] = math.Log((1+N)/(1+float64(d))) + 1
	}
}

// Search 搜索 topK 最相似的文档
func (idx *TfidfIndex) Search(query string, topK int, minScore float64) []ScoredDoc {
	if minScore <= 0 {
		minScore = 0.05
	}

	queryTokens := tokenizeForTFIDF(query)
	queryVec := make(TFIDFVector)
	for _, t := range queryTokens {
		queryVec[t] = queryVec[t] + 1
	}
	var maxTF float64
	for _, c := range queryVec {
		if c > maxTF {
			maxTF = c
		}
	}
	if maxTF > 0 {
		for t := range queryVec {
			queryVec[t] = (queryVec[t] / maxTF) * idx.idfCache[t]
		}
	}
	// 计算余弦相似度
	var results []ScoredDoc
	for i, vec := range idx.vectors {
		score := tfidfCosine(queryVec, vec)
		if score >= minScore {
			results = append(results, ScoredDoc{ID: idx.docIDs[i], Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > topK {
		results = results[:topK]
	}
	return results
}

func tfidfCosine(a, b TFIDFVector) float64 {
	var dot, normA, normB float64
	for t, va := range a {
		if vb, ok := b[t]; ok {
			dot += va * vb
		}
		normA += va * va
	}
	for _, vb := range b {
		normB += vb * vb
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

// ScoredDoc 带分数的文档
type ScoredDoc struct {
	ID    string
	Score float64
}

// ─── 语义搜索（Jaccard）─────────────────────────────────────────

const semanticSearchMinSimilarity = 0.12
const semanticSearchTopK = 5

// SearchBySemantics 基于 Jaccard 字符集的语义搜索
func SearchBySemantics(facts []*Fact, query string) []ScoredDoc {
	var results []ScoredDoc
	for _, f := range facts {
		if !f.IsActive() {
			continue
		}
		text := f.Subject + " " + f.Summary
		charScore := charJaccard(query, text)
		kwScore := keywordJaccard(query, text) * 1.2
		score := math.Max(charScore, kwScore)
		if score >= semanticSearchMinSimilarity {
			results = append(results, ScoredDoc{ID: f.ID, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > semanticSearchTopK {
		results = results[:semanticSearchTopK]
	}
	return results
}

// charJaccard 字符集 Jaccard 相似度
func charJaccard(a, b string) float64 {
	setA := make(map[rune]bool)
	for _, r := range a {
		setA[r] = true
	}
	setB := make(map[rune]bool)
	for _, r := range b {
		setB[r] = true
	}

	intersect := 0
	for r := range setA {
		if setB[r] {
			intersect++
		}
	}
	union := len(setA) + len(setB) - intersect
	if union == 0 {
		return 0
	}
	return float64(intersect) / float64(union)
}

// keywordJaccard 关键词级 Jaccard
func keywordJaccard(a, b string) float64 {
	wordsA := strings.FieldsFunc(strings.ToLower(a), func(r rune) bool {
		return r == ' ' || r == '，' || r == '。' || r == '、' || r == '；'
	})
	wordsB := strings.FieldsFunc(strings.ToLower(b), func(r rune) bool {
		return r == ' ' || r == '，' || r == '。' || r == '、' || r == '；'
	})

	setA := make(map[string]bool)
	for _, w := range wordsA {
		if len([]rune(w)) >= 2 {
			setA[w] = true
		}
	}
	setB := make(map[string]bool)
	for _, w := range wordsB {
		if len([]rune(w)) >= 2 {
			setB[w] = true
		}
	}

	intersect := 0
	for w := range setA {
		if setB[w] {
			intersect++
		}
	}
	union := len(setA) + len(setB) - intersect
	if union == 0 {
		return 0
	}
	return float64(intersect) / float64(union)
}
