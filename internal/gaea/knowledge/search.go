package knowledge

import (
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/strutil"
)

// Filter constrains search results.
type Filter struct {
	Category string
	Phase    string
	Tag      string
	Status   string
}

// 检索合分口径（GA3-07 单源）：Search 的榜单顺序由这组常数经 combineScore
// 唯一决定。改动任何一个都会改变检索结果顺序，search_golden_test.go 的
// golden 榜单会将其捕获 —— 调参前先看那份冻结的期望顺序。
const (
	// vecScale 向量分放大系数：TF-IDF 余弦相似度取值 0~1，乘以该系数后与
	// 关键词分（整数，0~10+）直接相加 —— 关键词主导，语义召回只做增强，
	// 不致语义项压过精确命中。
	vecScale = 8
	// irrelevantVecCutoff 无关条目淘汰线：关键词零命中且向量相似度低于该值
	// 的候选不进入结果（见 isIrrelevant；空查询不淘汰）。
	irrelevantVecCutoff = 0.1
	// indexMinScore TF-IDF 索引检索的最低相似度：刻意低于 gaea/search 包
	// 内置默认 0.05，放宽召回让低分语义候选进入合分，由淘汰线二次过滤。
	indexMinScore = 0.02
	// maxSearchResults 单次检索返回条数的硬上限。
	maxSearchResults = 20
)

// scoreEntry 的字段权重：标题命中最强，标签次之，分类/阶段再次，正文最弱。
const (
	weightTitle      = 10
	weightTag        = 5
	weightField      = 3 // category 与 phase 各计一次
	weightBody       = 1
	weightEmptyQuery = 1 // 空查询的保底分：返回全部候选
)

// combineScore 是检索合分的唯一口径：关键词分（0~10+）为主，向量分
// （TF-IDF 余弦，0~1）×vecScale 增强语义召回。Search 及包内其它需要
// 关键词/向量合分的路径都必须经它取总分，禁止内联裸常数。
func combineScore(kw int, vec float64) float64 {
	return float64(kw) + vec*vecScale
}

// isIrrelevant 报告候选是否与查询无关：关键词零命中且向量相似度低于
// 淘汰线 irrelevantVecCutoff。边界口径：vec 恰等于淘汰线时不淘汰（<）。
func isIrrelevant(kw int, vec float64) bool {
	return kw == 0 && vec < irrelevantVecCutoff
}

// Search searches the store for entries matching the query and filter.
// Ranking blends the keyword score (title/tag/category/body) with TF-IDF
// vector similarity (RAG) so semantically related entries surface even when
// they share no exact keywords. Returns up to 20 results (descending).
// 读失败如实返回 error（审计 P0#6 GA3-01）：不得把库故障当「搜不到结果」。
func Search(s *Store, query string, filter Filter) ([]Entry, error) {
	entries, err := s.ReadAll()
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)

	var candidates []Entry
	for _, e := range entries {
		if filter.Category != "" && e.Category != filter.Category {
			continue
		}
		if filter.Phase != "" && e.Phase != filter.Phase {
			continue
		}
		if filter.Tag != "" && !hasTag(e.Tags, filter.Tag) {
			continue
		}
		if filter.Status != "" && e.Status != filter.Status {
			continue
		}
		candidates = append(candidates, e)
	}

	// TF-IDF 向量相似度（RAG）：查询与条目正文的语义距离。
	// 索引按过滤签名+候选指纹缓存（tfidfIndexFor），写路径失效，避免每次查询重建。
	vecScores := map[string]float64{}
	if query != "" && len(candidates) > 0 {
		idx := s.tfidfIndexFor(candidates, filter)
		for _, r := range idx.Search(query, len(candidates), indexMinScore) {
			vecScores[r.ID] = r.Score
		}
	}

	var scored []scoredEntry
	for _, e := range candidates {
		kw := scoreEntry(e, query)
		vec := vecScores[e.Name]
		// 合分与淘汰口径单源（GA3-07）：见 combineScore / isIrrelevant。
		total := combineScore(kw, vec)
		if query != "" && isIrrelevant(kw, vec) {
			continue // 无关条目（无关键词且向量极低）
		}
		scored = append(scored, scoredEntry{Entry: e, score: total})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].Name < scored[j].Name
	})

	if len(scored) > maxSearchResults {
		scored = scored[:maxSearchResults]
	}

	result := make([]Entry, len(scored))
	for i, se := range scored {
		result[i] = se.Entry
	}
	return result, nil
}

type scoredEntry struct {
	Entry
	score float64
}

// scoreEntry computes a relevance score for an entry against a query.
// Scoring: title match +weightTitle, tag match +weightTag per tag,
// category/phase match +weightField each, body match +weightBody.
func scoreEntry(e Entry, query string) int {
	if query == "" {
		return weightEmptyQuery // return all entries with default score
	}

	q := strings.ToLower(query)
	score := 0

	// Title match (highest priority).
	if containsFold(e.Title, q) {
		score += weightTitle
	}

	// Tag match.
	for _, t := range e.Tags {
		if containsFold(t, q) {
			score += weightTag
		}
	}

	// Category/Phase match.
	if containsFold(e.Category, q) {
		score += weightField
	}
	if containsFold(e.Phase, q) {
		score += weightField
	}

	// Body match.
	if containsFold(e.Body, q) {
		score += weightBody
	}

	return score
}

// hasTag checks if the tags slice contains the given tag.
func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// containsFold checks if s contains substr (case-insensitive, CJK-aware).
func containsFold(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)

	if len(substr) == 0 {
		return true
	}

	// For CJK queries, check each character.
	if strutil.ContainsHan(substr) {
		runes := []rune(s)
		subRunes := []rune(substr)
		for i := 0; i <= len(runes)-len(subRunes); i++ {
			match := true
			for j := 0; j < len(subRunes); j++ {
				if runes[i+j] != subRunes[j] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
		return false
	}

	return strings.Contains(s, substr)
}
