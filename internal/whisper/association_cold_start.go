// Package whisper — association_cold_start.go
// 100% 对齐 ackem memory/associationColdStart.ts
// 关联图冷启动：批量 strengthenOrCreate，弥补导入/新用户仅靠 ingest 共现建边不足

package whisper

import (
	"sort"
	"strings"
)

// BatchSeedResult 批量建边结果
type BatchSeedResult struct {
	EdgesCreated    int `json:"edgesCreated"`
	FactsConsidered int `json:"factsConsidered"`
	OrphansLinked   int `json:"orphansLinked"`
}

// pickAssociationType 选择关联类型
func pickAssociationType(a, b *MemoryFact) string {
	if a.Subcategory == b.Subcategory {
		return "event_chain"
	}
	return "thematic"
}

// textOverlapScore 共享词/字重叠（无 embedding 时的兜底；含 CJK 二字 gram）
func textOverlapScore(a, b *MemoryFact) float64 {
	grams := func(s string) map[string]struct{} {
		text := strings.ToLower(s)
		set := make(map[string]struct{})
		// 二字 gram
		runes := []rune(text)
		for i := 0; i < len(runes)-1; i++ {
			set[string(runes[i:i+2])] = struct{}{}
		}
		// 分词
		parts := strings.FieldsFunc(text, func(r rune) bool {
			return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r >= 0x4e00)
		})
		for _, p := range parts {
			if len([]rune(p)) >= 2 {
				set[p] = struct{}{}
			}
		}
		return set
	}

	ta := grams(a.Subject + " " + a.Summary)
	tb := grams(b.Subject + " " + b.Summary)

	var overlap float64
	for t := range ta {
		if _, ok := tb[t]; ok {
			overlap++
		}
	}
	return overlap
}

// hasEdgeBetween 检查两个事实之间是否已有关联边
func hasEdgeBetween(index *AssociationIndex, a, b string) bool {
	for _, edge := range index.GetAssociations(a) {
		if edge.FactIDB == b || edge.FactIDA == b {
			return true
		}
	}
	return false
}

// linkForColdStart 冷启动建边：已有边 strengthenOrCreate，新边用足够强度 add
func linkForColdStart(index *AssociationIndex, factIDA, factIDB, assocType string) bool {
	if hasEdgeBetween(index, factIDA, factIDB) {
		index.StrengthenOrCreate(factIDA, factIDB, assocType, ColdStartEdgeStrength)
		return false
	}
	// 规范化顺序
	a, b := factIDA, factIDB
	if a > b {
		a, b = b, a
	}
	index.Add(Association{
		FactIDA:         a,
		FactIDB:         b,
		AssociationType: assocType,
		Strength:        ColdStartEdgeStrength,
	})
	return true
}

// BatchSeedAssociationsFromTextOverlap 基于文本重叠批量建边（优先孤儿事实）
func BatchSeedAssociationsFromTextOverlap(store *FactStore, index *AssociationIndex, minOverlap float64, maxOrphans, maxPairsPerFact int) BatchSeedResult {
	if minOverlap <= 0 {
		minOverlap = ColdStartTextMinOverlap
	}
	if maxOrphans <= 0 {
		maxOrphans = ColdStartMaxOrphans
	}
	if maxPairsPerFact <= 0 {
		maxPairsPerFact = ColdStartMaxPairsPerFact
	}

	active := store.ListActive()
	// 找孤儿（无关联的事实）
	var orphans []*Fact
	for _, f := range active {
		if len(index.GetAssociations(f.ID)) == 0 {
			orphans = append(orphans, f)
		}
	}

	targets := orphans
	if len(targets) == 0 {
		targets = active
	}
	if len(targets) > maxOrphans {
		targets = targets[:maxOrphans]
	}

	result := BatchSeedResult{FactsConsidered: len(targets)}
	isOrphan := func(id string) bool {
		for _, o := range orphans {
			if o.ID == id {
				return true
			}
		}
		return false
	}

	for _, fact := range targets {
		type scored struct {
			other *Fact
			score float64
		}
		var scores []scored
		for _, other := range active {
			if other.ID == fact.ID {
				continue
			}
			if fact.Domain != other.Domain {
				continue
			}
			s := textOverlapScore(&fact.MemoryFact, &other.MemoryFact)
			if s >= minOverlap {
				scores = append(scores, scored{other, s})
			}
		}
		sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })

		limit := maxPairsPerFact
		if limit > len(scores) {
			limit = len(scores)
		}
		linked := 0
		for _, sc := range scores[:limit] {
			if linkForColdStart(index, fact.ID, sc.other.ID, pickAssociationType(&fact.MemoryFact, &sc.other.MemoryFact)) {
				result.EdgesCreated++
				linked++
			}
		}
		if linked > 0 && isOrphan(fact.ID) {
			result.OrphansLinked++
		}
	}

	return result
}

// BatchSeedAssociationsFromEmbeddings 基于 embedding 相似度批量建边（优先孤儿事实）
func BatchSeedAssociationsFromEmbeddings(store *FactStore, index *AssociationIndex, embedCache *FactEmbeddingCache, minCosine float64, maxOrphans, maxPairsPerFact int, sameDomainOnly bool) BatchSeedResult {
	if minCosine <= 0 {
		minCosine = ColdStartMinCosine
	}
	if maxOrphans <= 0 {
		maxOrphans = ColdStartMaxOrphans
	}
	if maxPairsPerFact <= 0 {
		maxPairsPerFact = ColdStartMaxPairsPerFact
	}

	active := store.ListActive()
	if len(active) < 2 {
		return BatchSeedResult{}
	}

	var orphans []*Fact
	for _, f := range active {
		if len(index.GetAssociations(f.ID)) == 0 {
			orphans = append(orphans, f)
		}
	}

	targets := orphans
	if len(targets) == 0 {
		targets = active
	}
	if len(targets) > maxOrphans {
		targets = targets[:maxOrphans]
	}

	result := BatchSeedResult{FactsConsidered: len(targets)}
	isOrphan := func(id string) bool {
		for _, o := range orphans {
			if o.ID == id {
				return true
			}
		}
		return false
	}

	for _, fact := range targets {
		embA := embedCache.Get(fact.ID)
		if len(embA) == 0 {
			continue
		}

		type scored struct {
			other  *Fact
			cosine float64
		}
		var scores []scored
		for _, other := range active {
			if other.ID == fact.ID {
				continue
			}
			if sameDomainOnly && fact.Domain != other.Domain {
				continue
			}
			embB := embedCache.Get(other.ID)
			if len(embB) == 0 {
				continue
			}
			c := CosineSimilarity(embA, embB)
			if c >= minCosine {
				scores = append(scores, scored{other, c})
			}
		}
		sort.Slice(scores, func(i, j int) bool { return scores[i].cosine > scores[j].cosine })

		limit := maxPairsPerFact
		if limit > len(scores) {
			limit = len(scores)
		}
		linked := 0
		for _, sc := range scores[:limit] {
			if linkForColdStart(index, fact.ID, sc.other.ID, pickAssociationType(&fact.MemoryFact, &sc.other.MemoryFact)) {
				result.EdgesCreated++
				linked++
			}
		}
		if linked > 0 && isOrphan(fact.ID) {
			result.OrphansLinked++
		}
	}

	return result
}

// SeedAssociationsForNewFacts ingest 单轮新增事实 → 与库内 active 事实建边（含单条冷启动）
func SeedAssociationsForNewFacts(newFacts []*Fact, store *FactStore, index *AssociationIndex, embedCache *FactEmbeddingCache, minCosine float64) int {
	if len(newFacts) == 0 {
		return 0
	}
	if minCosine <= 0 {
		minCosine = 0.7
	}
	active := store.ListActive()
	created := 0

	// Embedding 分支
	if embedCache != nil && embedCache.Size() > 0 {
		for _, fact := range newFacts {
			embA := embedCache.Get(fact.ID)
			if len(embA) == 0 {
				continue
			}
			for _, other := range active {
				if other.ID == fact.ID {
					continue
				}
				if fact.Domain != other.Domain {
					continue
				}
				embB := embedCache.Get(other.ID)
				if len(embB) == 0 {
					continue
				}
				threshold := minCosine
				if len(newFacts) == 1 && threshold > 0.55 {
					threshold = 0.55
				}
				if CosineSimilarity(embA, embB) < threshold {
					continue
				}
				if linkForColdStart(index, fact.ID, other.ID, pickAssociationType(&fact.MemoryFact, &other.MemoryFact)) {
					created++
				}
			}
		}
	}

	// 新事实之间的关联
	if len(newFacts) >= 2 {
		for i := 0; i < len(newFacts); i++ {
			for j := i + 1; j < len(newFacts); j++ {
				a, b := newFacts[i], newFacts[j]
				if a.Domain != b.Domain {
					continue
				}
				if linkForColdStart(index, a.ID, b.ID, pickAssociationType(&a.MemoryFact, &b.MemoryFact)) {
					created++
				}
			}
		}
	}

	// 兜底：无关联建立时，文本重叠 / 最近邻弱边
	if created == 0 {
		created += seedSingleOrNoteAssociations(newFacts, store, index)
	}

	return created
}

// seedSingleOrNoteAssociations 单条新事实或 NOTE：文本重叠 / 最近邻弱边
func seedSingleOrNoteAssociations(newFacts []*Fact, store *FactStore, index *AssociationIndex) int {
	created := 0
	for _, fact := range newFacts {
		active := store.ListActive()
		var filtered []*Fact
		for _, f := range active {
			if f.ID != fact.ID {
				filtered = append(filtered, f)
			}
		}
		if len(filtered) == 0 {
			continue
		}

		type scored struct {
			other *Fact
			score float64
		}
		var scores []scored
		for _, other := range filtered {
			s := textOverlapScore(&fact.MemoryFact, &other.MemoryFact)
			if s >= 1 {
				scores = append(scores, scored{other, s})
			}
		}
		sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })

		if len(scores) > 0 {
			if linkForColdStart(index, fact.ID, scores[0].other.ID, pickAssociationType(&fact.MemoryFact, &scores[0].other.MemoryFact)) {
				created++
				continue
			}
		}

		// NOTE 或单条新事实：最近邻
		if fact.Subcategory == "NOTE" || len(newFacts) == 1 {
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
			})
			if len(filtered) > 0 {
				if linkForColdStart(index, fact.ID, filtered[0].ID, "thematic") {
					created++
				}
			}
		}
	}
	return created
}

// ReseedAssociationGraph 导入/索引重建后：embedding 优先，否则文本重叠兜底
func ReseedAssociationGraph(store *FactStore, index *AssociationIndex, embedCache *FactEmbeddingCache) BatchSeedResult {
	active := store.ListActive()
	if len(active) < 2 {
		return BatchSeedResult{}
	}

	if embedCache != nil && embedCache.Size() > 0 {
		return BatchSeedAssociationsFromEmbeddings(store, index, embedCache, 0, 0, 0, true)
	}

	return BatchSeedAssociationsFromTextOverlap(store, index, 0, 0, 0)
}
