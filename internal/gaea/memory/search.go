package memory

import (
	"math"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/bm25"
)

// SearchIndex is an inverted index over saved memories with BM25 ranking.
// Built once at load time and queried by the memory_search tool.
// V10.18+: BM25 term weighting replaces pure token count; Docs (AGENTS.md etc.)
// are indexed alongside Store memories for unified search.
//
// 审计 IN2-11/GA3-03 收敛（2026-10）：本类型是 bm25.Ranker 的薄封装——倒排、
// tf/df 统计与打分公式全部下沉到 internal/gaea/bm25（唯一实现），本层只保留
// gaea 记忆特有语义：name 键映射、kind 过滤、preview、以及 BM25 零命中时的
// substring fallback。分词口径经 bm25.Options{Stopwords: cjkStopwords}
// 逐字节复现收敛前行为（停用词整 token 丢弃、两字串不去重），检索结果顺序
// 与分值由快照测试钉死。
type SearchIndex struct {
	// ranker 是唯一打分实现（k1=1.2/b=0.75 默认参数，停用词分词口径）。
	ranker *bm25.Ranker
	// names 按 ranker 的文档 ID（索引顺序）映射回 memory slug / "doc:" 前缀。
	names []string
	// previews stores memory name → one-line description for display.
	previews map[string]string
	// kinds stores memory name → Kind (for filtering).
	kinds map[string]Kind
}

// SearchMatch is a single search result with a BM25 relevance score and preview.
type SearchMatch struct {
	Name    string // memory slug
	Score   float64
	Preview string // one-line description for display (from frontmatter)
	Kind    Kind   // semantic / episodic / procedural
}

// cjkStopwords are common CJK stopwords that add noise to search indexes.
// 口径归属（GA3-03）：只有 gaea 记忆分词过滤停用词；成本库（bm25 默认口径）
// 不过滤。作为 bm25.Options.Stopwords 开关值传入单源分词器。
var cjkStopwords = map[string]bool{
	"的": true, "了": true, "是": true, "在": true, "和": true,
	"也": true, "就": true, "都": true, "而": true, "及": true,
	"与": true, "或": true, "一个": true, "这个": true, "那个": true,
	"什么": true, "怎么": true, "如何": true, "为什么": true,
}

// memTokenizeOpts 是本包的分词口径（收敛前的本地实现行为）。
var memTokenizeOpts = bm25.Options{Stopwords: cjkStopwords}

// tokenize splits text into lowercase alphanumeric and CJK tokens.
// CJK characters are emitted as overlapping bigrams; stopwords are filtered.
// 收敛后为 bm25.TokenizeOpts 的口径化薄封装（recall.go 等包内消费点不变）。
func tokenize(text string) []string {
	return bm25.TokenizeOpts(text, memTokenizeOpts)
}

// BuildSearchIndex reads all memory files in the store and docs from the
// provided doc list, then builds an inverted index with BM25 statistics.
// Returns nil when there is nothing to index.
func (s Store) BuildSearchIndex(docs []Source) *SearchIndex {
	memories := s.List()
	if len(memories) == 0 && len(docs) == 0 {
		return nil
	}

	idx := &SearchIndex{
		previews: make(map[string]string),
		kinds:    make(map[string]Kind),
	}
	var docsForRanker []bm25.Doc

	// Index Store memories: text = title + description + body, lowercased.
	for _, m := range memories {
		text := strings.ToLower(m.Title + " " + m.Description + " " + m.Body)
		idx.addDoc(&docsForRanker, m.Name, text, m.Title, m.Description, m.Kind)
	}

	// Index Docs (AGENTS.md etc.) under a "doc:" prefix namespace.
	for _, d := range docs {
		name := "doc:" + d.Path
		text := strings.ToLower(d.Body)
		idx.addDoc(&docsForRanker, name, text, filepathBase(d.Path), "", KindSemantic)
	}

	idx.ranker = bm25.NewRankerOpts(docsForRanker, memTokenizeOpts, bm25.DefaultParams)
	return idx
}

// filepathBase returns the last element of a path, avoiding import of path/filepath.
func filepathBase(path string) string {
	path = strings.TrimRight(path, "/\\")
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// addDoc 把一条文档登记进索引：与收敛前一致，零 token 的文档不产生 preview/
// kind（因此也不会被 fallback 命中）；正文交给 bm25.Ranker 建倒排。
func (idx *SearchIndex) addDoc(docs *[]bm25.Doc, name, text, title, desc string, kind Kind) {
	tokens := bm25.TokenizeOpts(text, memTokenizeOpts)
	if len(tokens) == 0 {
		return
	}
	id := len(*docs)
	*docs = append(*docs, bm25.Doc{ID: id, Text: text})
	idx.names = append(idx.names, name)
	idx.kinds[name] = NormalizeKind(string(kind))

	// Preview.
	preview := title
	if preview == "" {
		preview = strings.ReplaceAll(name, "-", " ")
	}
	if desc != "" {
		preview += " — " + desc
	}
	idx.previews[name] = preview
}

// Search finds memories matching the query with BM25 ranking.
// Returns results sorted by score descending. To filter by kind, use SearchByKind.
func (idx *SearchIndex) Search(query string) []SearchMatch {
	return idx.searchFiltered(query, "")
}

// SearchByKind finds memories of a specific kind matching the query.
// Empty kind means no filter (same as Search).
func (idx *SearchIndex) SearchByKind(query string, kind Kind) []SearchMatch {
	return idx.searchFiltered(query, kind)
}

func (idx *SearchIndex) searchFiltered(query string, kind Kind) []SearchMatch {
	// 收敛前语义：nil 接收器或「索引无任何可命中词条」直接 nil（不走
	// fallback）；ranker.Empty() 恰为收敛前 len(entries)==0 的等价判定。
	if idx == nil || idx.ranker.Empty() {
		return nil
	}
	lowered := strings.ToLower(query)
	queryTokens := tokenize(lowered)
	if len(queryTokens) == 0 {
		return nil
	}

	// BM25 scoring via the single implementation (bm25). 收敛前 kind 过滤
	// 在累加循环内，事后过滤与其可观测行为等价：BM25 零命中 ⟺ Rank 零结果，
	// 全被 kind 过滤 ⟺ 收敛前 scores 为空 → 同样进入 fallback。
	var scores []SearchMatch
	for _, s := range idx.ranker.Rank(lowered) {
		name := idx.names[s.ID]
		if kind != "" && idx.kinds[name] != kind {
			continue
		}
		scores = append(scores, SearchMatch{
			Name:    name,
			Score:   s.Score,
			Preview: idx.previews[name],
			Kind:    idx.kinds[name],
		})
	}

	// Substring fallback: when BM25 finds nothing, try raw substring match.
	if len(scores) == 0 {
		for name, preview := range idx.previews {
			if kind != "" && idx.kinds[name] != kind {
				continue
			}
			if strings.Contains(strings.ToLower(preview), queryTokens[0]) {
				scores = append(scores, SearchMatch{
					Name:    name,
					Score:   0.1,
					Preview: preview,
					Kind:    idx.kinds[name],
				})
			}
		}
	}

	if len(scores) == 0 {
		return nil
	}

	// Sort by score descending, then alphabetically.
	sort.Slice(scores, func(i, j int) bool {
		if math.Abs(scores[i].Score-scores[j].Score) > 0.001 {
			return scores[i].Score > scores[j].Score
		}
		return scores[i].Name < scores[j].Name
	})

	return scores
}
