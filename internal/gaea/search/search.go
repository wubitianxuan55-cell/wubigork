// Package search — 共享向量检索层（TF-IDF + 中文 bigram 余弦相似度）
//
// 提取自轻语 vector_search.go 的通用实现（算法同源），供知识库等模块做
// RAG 检索。轻语 hermes.db 保留其自有副本不动（右脑自治），新模块一律复用本包。
package search

import (
	"math"
	"sort"
	"strings"
)

// Vector 是 TF-IDF 词向量（词 → 权重）。
type Vector map[string]float64

// Doc 是待索引文档。
type Doc struct {
	ID   string
	Text string
}

// ScoredDoc 带相似度分数的文档。
type ScoredDoc struct {
	ID    string
	Score float64
}

// TfidfIndex 是内存 TF-IDF 倒排索引：整词（≥2 字）+ CJK bigram 分词，
// TF 以 maxTF 归一化，IDF 采用平滑 log 公式，检索用余弦相似度。
type TfidfIndex struct {
	vectors  []Vector
	docIDs   []string
	idfCache map[string]float64
	docCount int
}

// NewTfidfIndex 创建空索引。
func NewTfidfIndex() *TfidfIndex {
	return &TfidfIndex{idfCache: make(map[string]float64)}
}

// Tokenize 中文分词：CJK 标点分割后取整词（≥2 字）与相邻 bigram，去重。
func Tokenize(text string) []string {
	var tokens []string
	seen := make(map[string]bool)

	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == '，' || r == '。' || r == '！' || r == '？' || r == '、' ||
			r == '；' || r == '：' || r == ' ' || r == '\t' || r == '\n' ||
			r == '“' || r == '”' || r == '"' || r == '（' || r == '）' ||
			r == '【' || r == '】' || r == '.' || r == ',' || r == '!' || r == '?'
	})
	for _, f := range fields {
		f = strings.TrimSpace(f)
		runes := []rune(f)
		if len(runes) >= 2 && !seen[f] {
			seen[f] = true
			tokens = append(tokens, f)
		}
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

// Build 构建索引：统计 DF → 计算 IDF，TF 以 maxTF 归一化。
func (idx *TfidfIndex) Build(docs []Doc) {
	idx.vectors = make([]Vector, len(docs))
	idx.docIDs = make([]string, len(docs))
	idx.docCount = len(docs)

	df := make(map[string]int)
	for i, d := range docs {
		idx.docIDs[i] = d.ID
		tokens := Tokenize(d.Text)
		tf := make(map[string]int)
		for _, t := range tokens {
			tf[t]++
		}
		idx.vectors[i] = make(Vector)
		maxTF := 0
		for _, c := range tf {
			if c > maxTF {
				maxTF = c
			}
		}
		for t, c := range tf {
			idx.vectors[i][t] = float64(c) / float64(maxTF)
		}
		for t := range tf {
			df[t]++
		}
	}

	N := float64(idx.docCount)
	for t, d := range df {
		idx.idfCache[t] = math.Log((1+N)/(1+float64(d))) + 1
	}
}

// Search 检索 topK 个相似文档，分数低于 minScore 的丢弃。
func (idx *TfidfIndex) Search(query string, topK int, minScore float64) []ScoredDoc {
	if minScore <= 0 {
		minScore = 0.05
	}
	queryVec := make(Vector)
	for _, t := range Tokenize(query) {
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

	var results []ScoredDoc
	for i, vec := range idx.vectors {
		score := Cosine(queryVec, vec)
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

// Cosine 计算两个 TF-IDF 向量的余弦相似度。
func Cosine(a, b Vector) float64 {
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
