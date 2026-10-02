// Package bm25 轻量本地 BM25 排序（零 token、零网络）：中英文混合分词
// （CJK 重叠二元组 + 字母数字词），对粗召回候选打分排序。供成本库等
// 结构化库在 SQL/子串召回之后做本地相关度排序，数据量大也不依赖模型。
//
// 审计 GA3-03/GA3-04/IN2-11 收敛落点：BM25 打分公式唯一实现（Params.
// TermScore，k1/b 参数化）；分词单源 + Options 开关组合复现各消费方冻结
// 口径（行为逐字节不变）。同足迹内刻意未并入的兄弟口径逐项列于 Options
// 对照表并附快照测试。
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

// Options 是分词口径开关：各消费方在行为冻结前提下以开关组合逐字节复现
// 自己的现有输出（统一=单源实现，不改变任何一方的 token 流）。
//
// 口径对照表（审计 GA3-04 现场复核结论）：
//
//	消费者                      口径                                       接入方式
//	internal/gaea/cost          DefaultTokens（无停用词、两字串整串即二元组不重复）   本包默认
//	internal/gaea/memory        {Stopwords: cjkStopwords}（停用词整 token 丢弃、两字串不去重） TokenizeOpts 开关
//	internal/gaea/search        FieldFunc 特定标点集分字段、整词≥2 rune、字段内含符号 bigram、全局去重、不小写化 —— 不同算法族，行为冻结无法并入，刻意保留（快照见 bm25_test）
//	internal/memory             字母数字连排整串 token（CJK 属字母）+ 非 space/punct 单 rune token + 全文 Han 相邻二元组后置追加 —— 不同算法族，刻意保留
//	internal/gaea/textsim       仅 Han + 字母数字词、集合去重 Dice —— 集合语义不同族，刻意保留
//	internal/novelstyle         词表贪心最长匹配（文体统计用途，非检索）—— 不同用途出族，刻意保留
type Options struct {
	// Stopwords 非 nil 时，命中集合的整串/二元组 token 被丢弃
	//（internal/gaea/memory 口径：cjkStopwords）。
	Stopwords map[string]bool
	// DedupeTwoChar 为 true 时，两字 CJK 串的整串与二元组相同，只输出一次
	//（成本库口径）。false 时两者都输出（gaea/memory 口径）。
	DedupeTwoChar bool
}

// DefaultTokens 是成本库 Ranker/Cache 的分词口径（与收敛前 bm25.Tokenize 一致）。
var DefaultTokens = Options{DedupeTwoChar: true}

// Params 是 BM25 打分参数（k1 词频饱和、b 长度归一）。不同库口径不同：
// 成本库与 gaea 记忆为标准 1.2/0.75，故事记忆为 1.5/0.75——统一公式、
// 参数化口径（IN2-11）。
type Params struct {
	K1 float64
	B  float64
}

// DefaultParams 是标准 BM25 参数（k1=1.2, b=0.75），即收敛前成本库/gaea
// 记忆的参数。
var DefaultParams = Params{K1: 1.2, B: 0.75}

// TermScore 是 BM25 单词条贡献的唯一实现：
//
//	idf * (tf*(k1+1)) / (tf + k1*(1-b + b*docLen/avgLen))，
//	idf = ln((N-df+0.5)/(df+0.5) + 1)
//
// 三套 BM25（成本库 Ranker、gaea 记忆 SearchIndex、故事记忆 Index）共用；
// 表达式求值顺序与各原实现逐字节一致，任何一侧改动都会同时打破三方快照。
func (p Params) TermScore(tf, docLen, avgLen, df, totalDocs float64) float64 {
	idf := math.Log((totalDocs-df+0.5)/(df+0.5) + 1.0)
	norm := 1 - p.B + p.B*docLen/avgLen
	return idf * (tf * (p.K1 + 1)) / (tf + p.K1*norm)
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

// Ranker 是 BM25 打分器（默认 k1=1.2, b=0.75 标准参数；可经 NewRankerOpts
// 换分词口径与打分参数）。
type Ranker struct {
	docs      []Doc
	totalDocs int
	avgLen    float64
	docLen    []int
	postings  map[string][]posting
	tokens    Options
	params    Params
}

type posting struct {
	doc int
	tf  int
}

// NewRanker 以默认口径构建 BM25 索引（成本库路径，行为与收敛前一致）。
func NewRanker(docs []Doc) *Ranker {
	return NewRankerOpts(docs, DefaultTokens, DefaultParams)
}

// NewRankerOpts 以指定分词口径与打分参数构建 BM25 索引
// （internal/gaea/memory SearchIndex 的薄封装入口：停用词口径 + 默认参数）。
func NewRankerOpts(docs []Doc, tokens Options, params Params) *Ranker {
	r := &Ranker{
		docs:     docs,
		postings: map[string][]posting{},
		docLen:   make([]int, len(docs)),
		tokens:   tokens,
		params:   params,
	}
	var totalLen int
	for i, d := range docs {
		docTokens := TokenizeOpts(d.Text, tokens)
		r.docLen[i] = len(docTokens)
		totalLen += len(docTokens)
		if len(docTokens) == 0 {
			continue
		}
		r.totalDocs++
		tf := map[string]int{}
		for _, tok := range docTokens {
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

// Empty 报告索引中是否存在任何可命中词条（≥2 字 token 的倒排为空 =
// 建了索引但无有效词条）。internal/gaea/memory 以此维持「空索引直接返回
// nil、不走 substring fallback」的收敛前语义。
func (r *Ranker) Empty() bool {
	return r == nil || len(r.postings) == 0
}

// Rank 按查询打分并返回降序结果（仅含至少命中一个查询词的文档）。
func (r *Ranker) Rank(query string) []Scored {
	if r == nil || r.totalDocs == 0 {
		return nil
	}
	queryTokens := TokenizeOpts(query, r.tokens)
	if len(queryTokens) == 0 {
		return nil
	}

	scores := make(map[int]float64)
	nDocs := float64(r.totalDocs)
	for _, token := range queryTokens {
		if len(token) < 2 {
			continue
		}
		postings, ok := r.postings[token]
		if !ok {
			continue
		}
		n := float64(len(postings))
		for _, p := range postings {
			tf := float64(p.tf)
			docLen := float64(r.docLen[p.doc])
			scores[p.doc] += r.params.TermScore(tf, docLen, r.avgLen, n, nDocs)
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

// Tokenize 中英文混合分词（成本库默认口径）：CJK 连续串输出整串 + 重叠
// 二元组（两字串不重复），字母/数字/下划线连排为词，其余字符为分隔符，
// 小写化。等价 TokenizeOpts(text, DefaultTokens)。
func Tokenize(text string) []string {
	return TokenizeOpts(text, DefaultTokens)
}

// TokenizeOpts 按 Options 口径分词：算法与 Tokenize 单源，开关决定停用词
// 过滤与两字串去重（各消费方冻结口径的复现点，见 Options 对照表）。
func TokenizeOpts(text string, o Options) []string {
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
		if !o.Stopwords[full] {
			tokens = append(tokens, full)
		}
		for i := 0; i+1 < len(cjkBuf); i++ {
			bigram := string([]rune{cjkBuf[i], cjkBuf[i+1]})
			if o.DedupeTwoChar && bigram == full { // 两字词整串即二元组，避免重复计数
				continue
			}
			if o.Stopwords[bigram] {
				continue
			}
			tokens = append(tokens, bigram)
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
