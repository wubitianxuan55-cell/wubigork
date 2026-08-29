package knowledge

import (
	"sort"
	"strings"
	"unicode"
)

// Filter constrains search results.
type Filter struct {
	Category string
	Phase    string
	Tag      string
	Status   string
}

// Search searches the store for entries matching the query and filter.
// Ranking blends the keyword score (title/tag/category/body) with TF-IDF
// vector similarity (RAG) so semantically related entries surface even when
// they share no exact keywords. Returns up to 20 results (descending).
func Search(s *Store, query string, filter Filter) []Entry {
	entries := s.ReadAll()
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
		for _, r := range idx.Search(query, len(candidates), 0.02) {
			vecScores[r.ID] = r.Score
		}
	}

	var scored []scoredEntry
	for _, e := range candidates {
		kw := scoreEntry(e, query)
		vec := vecScores[e.Name]
		// 关键词分（0~10+）为主，向量分（0~1）×8 增强语义召回。
		total := float64(kw) + vec*8
		if query != "" && kw == 0 && vec < 0.1 {
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

	if len(scored) > 20 {
		scored = scored[:20]
	}

	result := make([]Entry, len(scored))
	for i, se := range scored {
		result[i] = se.Entry
	}
	return result
}

type scoredEntry struct {
	Entry
	score float64
}

// scoreEntry computes a relevance score for an entry against a query.
// Scoring: title match +10, tag match +5, category/phase match +3, body match +1.
func scoreEntry(e Entry, query string) int {
	if query == "" {
		return 1 // return all entries with default score
	}

	q := strings.ToLower(query)
	score := 0

	// Title match (highest priority).
	if containsFold(e.Title, q) {
		score += 10
	}

	// Tag match.
	for _, t := range e.Tags {
		if containsFold(t, q) {
			score += 5
		}
	}

	// Category/Phase match.
	if containsFold(e.Category, q) {
		score += 3
	}
	if containsFold(e.Phase, q) {
		score += 3
	}

	// Body match.
	if containsFold(e.Body, q) {
		score += 1
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
	if isCJStr(substr) {
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

// isCJStr returns true if the string contains CJK characters.
func isCJStr(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
