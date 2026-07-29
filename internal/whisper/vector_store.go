// Package whisper — vector_store.go
// 100% 对齐 ackem memory/vectorStore.ts
// 向量存储：TF-IDF 稀疏向量 + 可选稠密向量，支持余弦相似度检索
// 补充现有的 vector_search.go（基础版），增加 VectorStore 类 + 稠密向量支持

package whisper

import (
	"math"
	"sort"
	"strings"
	"sync"
)

// VectorStore TF-IDF 向量存储（含可选稠密向量缓存）
type VectorStore struct {
	mu sync.RWMutex

	// TF-IDF 稀疏向量
	idf         map[string]float64
	vectors     []sparseVector
	termToID    map[string]int
	nextID      int
	lastHash    string

	// 稠密向量缓存（embedding 空间）
	denseVectors   []denseVector
	denseLastHash  string
}

type sparseVector struct {
	factID string
	vec    map[int]float64
	norm   float64
}

type denseVector struct {
	factID string
	vec    []float64
	norm   float64
}

// NewVectorStore 创建向量存储
func NewVectorStore() *VectorStore {
	return &VectorStore{
		idf:      make(map[string]float64),
		termToID: make(map[string]int),
	}
}

// tokenize 对事实做中文分词+小写处理
func (vs *VectorStore) tokenize(f *Fact) []string {
	text := strings.ToLower(f.Subject + " " + f.Summary)
	// 按非字母数字+中文字符分割
	var tokens []string
	for _, field := range []string{f.Subject, f.Summary} {
		// 2-gram for CJK
		runes := []rune(field)
		for i := 0; i < len(runes)-1; i++ {
			tokens = append(tokens, strings.ToLower(string(runes[i:i+2])))
		}
		// word-level
		for _, w := range strings.Fields(strings.ToLower(field)) {
			if len(w) >= 2 {
				tokens = append(tokens, w)
			}
		}
	}
	_ = text
	return tokens
}

// Build 从活跃事实重建 TF-IDF 向量
func (vs *VectorStore) Build(facts []*Fact) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// 构建 hash 检测变更
	totalUpdated := int64(0)
	for _, f := range facts {
		totalUpdated += f.UpdatedAt.UnixMilli()
	}
	hash := string(rune(len(facts))) + "-" + string(rune(totalUpdated))
	if hash == vs.lastHash && len(vs.vectors) > 0 {
		return
	}
	vs.lastHash = hash

	vs.termToID = make(map[string]int)
	vs.idf = make(map[string]float64)
	vs.vectors = nil
	vs.nextID = 0

	// 收集所有文档的 token 集
	var docs [][]string
	for _, f := range facts {
		if !f.IsActive() {
			continue
		}
		tokens := vs.tokenize(f)
		docs = append(docs, tokens)
	}

	// 计算文档频率
	for _, tokens := range docs {
		seen := make(map[string]bool)
		for _, t := range tokens {
			if seen[t] {
				continue
			}
			seen[t] = true
			vs.idf[t]++
		}
	}

	// 计算 IDF
	N := float64(len(docs))
	for term, df := range vs.idf {
		vs.idf[term] = math.Log((1+N)/(1+df)) + 1
		vs.termToID[term] = vs.nextID
		vs.nextID++
	}

	// 构建向量
	activeIdx := 0
	for _, f := range facts {
		if !f.IsActive() {
			continue
		}
		tokens := docs[activeIdx]
		activeIdx++

		tf := make(map[string]int)
		maxTF := 0
		for _, t := range tokens {
			tf[t]++
			if tf[t] > maxTF {
				maxTF = tf[t]
			}
		}

		vec := make(map[int]float64)
		var norm float64
		for term, count := range tf {
			id, ok := vs.termToID[term]
			if !ok {
				continue
			}
			idfVal, ok := vs.idf[term]
			if !ok {
				continue
			}
			val := (float64(count) / float64(maxTF)) * idfVal
			vec[id] = val
			norm += val * val
		}
		norm = math.Sqrt(norm)
		if norm == 0 {
			norm = 1
		}
		vs.vectors = append(vs.vectors, sparseVector{factID: f.ID, vec: vec, norm: norm})
	}
}

// Search 使用 TF-IDF 向量搜索
func (vs *VectorStore) Search(query string, topK int) []ScoreResult {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	if topK <= 0 {
		topK = 5
	}

	// 构建查询向量
	queryTokens := vs.tokenizeQuery(query)
	qTF := make(map[string]int)
	maxTF := 0
	for _, t := range queryTokens {
		qTF[t]++
		if qTF[t] > maxTF {
			maxTF = qTF[t]
		}
	}

	qVec := make(map[int]float64)
	var qNorm float64
	for term, count := range qTF {
		id, ok := vs.termToID[term]
		if !ok {
			continue
		}
		idfVal := vs.idf[term]
		val := (float64(count) / float64(maxTF)) * idfVal
		qVec[id] = val
		qNorm += val * val
	}
	qNorm = math.Sqrt(qNorm)
	if qNorm == 0 {
		return nil
	}

	type scored struct {
		factID string
		score  float64
	}
	var results []scored
	for _, sv := range vs.vectors {
		var dot float64
		for id, qv := range qVec {
			if fv, ok := sv.vec[id]; ok {
				dot += qv * fv
			}
		}
		score := dot / (qNorm * sv.norm)
		if score > 0.05 {
			results = append(results, scored{sv.factID, score})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
	if len(results) > topK {
		results = results[:topK]
	}

	var out []ScoreResult
	for _, r := range results {
		out = append(out, ScoreResult{FactID: r.factID, Score: r.score})
	}
	return out
}

// ScoreResult 搜索结果
type ScoreResult struct {
	FactID string  `json:"factId"`
	Score  float64 `json:"score"`
}

func (vs *VectorStore) tokenizeQuery(query string) []string {
	runes := []rune(query)
	var tokens []string
	for i := 0; i < len(runes)-1; i++ {
		tokens = append(tokens, strings.ToLower(string(runes[i:i+2])))
	}
	for _, w := range strings.Fields(strings.ToLower(query)) {
		if len(w) >= 2 {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

// BuildDense 从外部 embedding 构建稠密向量缓存
func (vs *VectorStore) BuildDense(facts []*Fact, embeddings [][]float64) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.denseVectors = nil

	for i, f := range facts {
		if !f.IsActive() || i >= len(embeddings) {
			continue
		}
		vec := embeddings[i]
		var norm float64
		for _, v := range vec {
			norm += v * v
		}
		norm = math.Sqrt(norm)
		if norm == 0 {
			norm = 1
		}
		vs.denseVectors = append(vs.denseVectors, denseVector{factID: f.ID, vec: vec, norm: norm})
	}
}

// SearchDense 使用稠密向量搜索
func (vs *VectorStore) SearchDense(queryEmbed []float64, topK int) []ScoreResult {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	if topK <= 0 {
		topK = 5
	}

	var qNorm float64
	for _, v := range queryEmbed {
		qNorm += v * v
	}
	qNorm = math.Sqrt(qNorm)
	if qNorm == 0 {
		return nil
	}

	type scored struct {
		factID string
		score  float64
	}
	var results []scored
	for _, dv := range vs.denseVectors {
		var dot float64
		for i, qv := range queryEmbed {
			if i < len(dv.vec) {
				dot += qv * dv.vec[i]
			}
		}
		score := dot / (qNorm * dv.norm)
		if score > 0.05 {
			results = append(results, scored{dv.factID, score})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
	if len(results) > topK {
		results = results[:topK]
	}

	var out []ScoreResult
	for _, r := range results {
		out = append(out, ScoreResult{FactID: r.factID, Score: r.score})
	}
	return out
}
