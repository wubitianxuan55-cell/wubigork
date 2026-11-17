package tool

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"unicode"
)

// 检索展开（codex 蒸馏 T8 三层之三，tool_search 同构）：工具多时首轮只下发
// 常用集，其余藏进索引；模型调 tool_search 才把选中的 ≤limit 个物化成 schema。
// 索引条目只存检索文本与 schema 引用（同一份 json.RawMessage，不复制），
// 内存不因工具多而翻倍。检索是自包含 BM25（k1=1.2, b=0.75）——上游用 bm25
// crate，gaea 无该依赖且不引入新外部依赖，整数化实现保持确定性
// （同分按名字典序，同输入同结果）。
//
// 排除纪律：tool_search 自身与 websearch 不进索引——避免「搜索工具出现在
// 搜索结果里」。
//
// 接线（boot 侧隐藏策略 + tool_search 工具注册）挂小批：隐藏哪些工具是
// 产品拍板项；本文件只提供索引与检索，不接线 = 零行为变化。

// ToolSearchDefaultLimit 是单次检索物化的上限（上游同值）。
const ToolSearchDefaultLimit = 8

// SearchEntry 是一条检索索引项。
type SearchEntry struct {
	Name       string
	Namespace  string          // 工具来源（SourceOf），空=内建
	SearchText string          // 检索文本（名 + 描述的拼接）
	Schema     json.RawMessage // 指向注册表同一份 canonical schema，不复制
}

// SearchIndex 是只读检索索引（构造后并发读安全）。
type SearchIndex struct {
	entries []SearchEntry
	docs    []map[string]int // 各条目的词频表
	totLen  float64          // 平均文档长度
}

// BuildSearchIndex 从注册表构建索引：包含隐藏工具（藏起来但模型找得到），
// 排除 tool_search / websearch 自身。
func BuildSearchIndex(reg *Registry) *SearchIndex {
	var entries []SearchEntry
	for _, name := range reg.Names() {
		if name == "tool_search" || name == "websearch" {
			continue
		}
		t, ok := reg.Get(name)
		if !ok {
			continue // 幽灵名守卫
		}
		searchText := name + "\n" + t.Description()
		entries = append(entries, SearchEntry{
			Name:       name,
			Namespace:  SourceOf(name),
			SearchText: searchText,
		})
	}
	return NewSearchIndex(entries)
}

// NewSearchIndex 用给定条目建索引（测试与自定义装配用）。
func NewSearchIndex(entries []SearchEntry) *SearchIndex {
	ix := &SearchIndex{entries: entries}
	total := 0.0
	for _, e := range entries {
		tf := termFreqs(tokenizeSearch(e.SearchText))
		ix.docs = append(ix.docs, tf)
		n := 0
		for _, c := range tf {
			n += c
		}
		total += float64(n)
	}
	if len(entries) > 0 {
		ix.totLen = total / float64(len(entries))
	}
	return ix
}

// Search 按查询检索，返回得分降序、同分按名字典序的至多 limit 条（limit<=0
// 取默认 8）。查询为空返回空。
func (ix *SearchIndex) Search(query string, limit int) []SearchEntry {
	if limit <= 0 {
		limit = ToolSearchDefaultLimit
	}
	terms := tokenizeSearch(query)
	if len(terms) == 0 {
		return nil
	}
	type hit struct {
		idx   int
		score float64
	}
	n := float64(len(ix.entries))
	var hits []hit
	for i, tf := range ix.docs {
		score := 0.0
		dl := 0.0
		for _, c := range tf {
			dl += float64(c)
		}
		for _, t := range terms {
			f := float64(tf[t])
			if f == 0 {
				continue
			}
			df := 0
			for _, other := range ix.docs {
				if other[t] > 0 {
					df++
				}
			}
			idf := math.Log(1 + (n-float64(df)+0.5)/(float64(df)+0.5))
			denom := f + 1.2*(1-0.75+0.75*dl/maxFloat(ix.totLen, 1))
			score += idf * (f * 2.2) / denom
		}
		if score > 0 {
			hits = append(hits, hit{idx: i, score: score})
		}
	}
	sort.Slice(hits, func(a, b int) bool {
		if hits[a].score != hits[b].score {
			return hits[a].score > hits[b].score
		}
		return ix.entries[hits[a].idx].Name < ix.entries[hits[b].idx].Name
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]SearchEntry, len(hits))
	for i, h := range hits {
		out[i] = ix.entries[h.idx]
	}
	return out
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// tokenizeSearch 检索文本分词：ASCII 词 + CJK 二字窗口（与 skill 词法路同思路，
// 各领域包自带小实现避免交叉依赖——tool 不能反向引 skill）。
func tokenizeSearch(s string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) == 0 {
			return
		}
		runes := cur
		cur = cur[:0]
		if unicode.Is(unicode.Han, runes[0]) {
			if len(runes) == 1 {
				out = append(out, string(runes))
				return
			}
			for i := 0; i+1 < len(runes); i++ {
				out = append(out, string(runes[i:i+2]))
			}
			return
		}
		out = append(out, strings.ToLower(string(runes)))
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur = append(cur, unicode.ToLower(r))
			continue
		}
		flush()
	}
	flush()
	return out
}

// termFreqs 词频表。
func termFreqs(tokens []string) map[string]int {
	m := make(map[string]int, len(tokens))
	for _, t := range tokens {
		m[t]++
	}
	return m
}
