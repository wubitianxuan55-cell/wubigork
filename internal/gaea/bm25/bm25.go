// Package bm25 轻量本地 BM25 排序（零 token、零网络）：中英文混合分词
// （CJK 重叠二元组 + 字母数字词），对粗召回候选打分排序。供成本库等
// 结构化库在 SQL/子串召回之后做本地相关度排序，数据量大也不依赖模型。
package bm25

import (
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// Doc 是参与打分的一条文档。
type Doc struct {
	ID   int
	Text string
}

// Scored 是打分结果（ID 对应 Doc.ID，Score 越高越相关）。
type Scored struct {
	ID    int
	Score float64
}

// Cache 是 BM25 打分器的按 key 缓存（T7-3：避免每请求全量重建倒排索引）。
// key 由调用方决定（如按项目/成本库 + 数据版本），文档集合不变时复用已构建
// 的 Ranker；数据更新后调用 Invalidate/InvalidateAll 失效。线程安全。
type Cache struct {
	mu sync.Mutex
	m  map[string]*Ranker
}

// NewCache 创建空缓存。
func NewCache() *Cache {
	return &Cache{m: make(map[string]*Ranker)}
}

// Get 返回 key 对应的打分器：命中直接复用（build 不调用）；未命中调用 build
// 构建一次并缓存。build 每次只会在不命中时执行，返回的 Ranker 不可变安全复用。
func (c *Cache) Get(key string, build func() []Doc) *Ranker {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.m[key]; ok {
		return r
	}
	r := NewRanker(build())
	c.m[key] = r
	return r
}

// Invalidate 删除指定 key 的缓存（该 key 数据更新后调用）。
func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
}

// InvalidateAll 清空全部缓存。
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[string]*Ranker)
}

// Ranker 是 BM25 打分器（k1=1.2, b=0.75 标准参数）。
type Ranker struct {
	docs      []Doc
	totalDocs int
	avgLen    float64
	docLen    []int
	postings  map[string][]posting
}

type posting struct {
	doc int
	tf  int
}

const (
	k1 = 1.2
	b  = 0.75
)

// NewRanker 构建 BM25 索引（对给定文档集合）。
func NewRanker(docs []Doc) *Ranker {
	r := &Ranker{
		docs:     docs,
		postings: map[string][]posting{},
		docLen:   make([]int, len(docs)),
	}
	var totalLen int
	for i, d := range docs {
		tokens := Tokenize(d.Text)
		r.docLen[i] = len(tokens)
		totalLen += len(tokens)
		if len(tokens) == 0 {
			continue
		}
		r.totalDocs++
		tf := map[string]int{}
		for _, tok := range tokens {
			if len(tok) < 2 {
				continue
			}
			tf[tok]++
		}
		for tok, count := range tf {
			r.postings[tok] = append(r.postings[tok], posting{doc: i, tf: count})
		}
	}
	if r.totalDocs > 0 {
		r.avgLen = float64(totalLen) / float64(r.totalDocs)
	}
	return r
}

// Rank 按查询打分并返回降序结果（仅含至少命中一个查询词的文档）。
func (r *Ranker) Rank(query string) []Scored {
	if r == nil || r.totalDocs == 0 {
		return nil
	}
	queryTokens := Tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	scores := make(map[int]float64)
	for _, token := range queryTokens {
		if len(token) < 2 {
			continue
		}
		postings, ok := r.postings[token]
		if !ok {
			continue
		}
		n := float64(len(postings))
		idf := math.Log((float64(r.totalDocs)-n+0.5)/(n+0.5) + 1.0)
		for _, p := range postings {
			tf := float64(p.tf)
			docLen := float64(r.docLen[p.doc])
			norm := 1 - b + b*docLen/r.avgLen
			scores[p.doc] += idf * (tf * (k1 + 1)) / (tf + k1*norm)
		}
	}

	out := make([]Scored, 0, len(scores))
	for id, score := range scores {
		out = append(out, Scored{ID: id, Score: score})
	}
	sort.Slice(out, func(i, j int) bool {
		if math.Abs(out[i].Score-out[j].Score) > 0.0001 {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Tokenize 中英文混合分词：CJK 连续串输出整串 + 重叠二元组，
// 字母/数字串按词输出，其余字符为分隔符。小写化。
func Tokenize(text string) []string {
	var tokens []string
	var current strings.Builder

	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	var cjkBuf []rune
	flushCJK := func() {
		if len(cjkBuf) == 0 {
			return
		}
		full := string(cjkBuf)
		tokens = append(tokens, full)
		for i := 0; i+1 < len(cjkBuf); i++ {
			bigram := string([]rune{cjkBuf[i], cjkBuf[i+1]})
			if bigram != full { // 两字词整串即二元组，避免重复计数
				tokens = append(tokens, bigram)
			}
		}
		cjkBuf = cjkBuf[:0]
	}

	for _, r := range text {
		switch {
		case isCJK(r):
			flush()
			cjkBuf = append(cjkBuf, unicode.ToLower(r))
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
			flushCJK()
			current.WriteRune(unicode.ToLower(r))
		default:
			flushCJK()
			flush()
		}
	}
	flushCJK()
	flush()
	return tokens
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}
