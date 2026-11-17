package skill

// RRF 融合（codex 蒸馏 T14，rrf_lexical_char_v1 同构，上游全文 90 行）：
// 两路各取 50 候选，按 1/(K+名次) 累加，排序用三级 tie-break——
// 分数降序 → 最好名次升序 → id 升序。缺任何一级，同一查询两次可能给出
// 不同顺序，直接破坏「同输入同结果」验收。

const (
	// RRFK 是融合常数：不是超参，不要调。
	RRFK = 60
	// fusionMaxCandidates 是每路取进融合的候选上限。
	fusionMaxCandidates = 50
)

// Fused 是融合结果：与候选索引同序无关，IDs 按 tie-break 排好。
type Fused struct {
	IDs []int
	// QueryTermCount = max(两路查询词数)；Truncated = OR(两路截断)。
	QueryTermCount int
	QueryTruncated bool
}

// Fuse 融合多路排名（每路是候选 id 按名次的有序列表）。三级 tie-break 保证
// 同输入同输出；limit <= 0 时不截断。
func Fuse(rankings [][]int, limit int) Fused {
	score := map[int]float64{}
	bestRank := map[int]int{}
	for _, ranking := range rankings {
		for i, id := range ranking {
			rank := i + 1 // 名次从 1 起
			score[id] += 1.0 / float64(RRFK+rank)
			if prev, ok := bestRank[id]; !ok || rank < prev {
				bestRank[id] = rank
			}
		}
	}
	ids := make([]int, 0, len(score))
	for id := range score {
		ids = append(ids, id)
	}
	// 三级 tie-break：分数降序 → 最好名次升序 → id 升序。
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && fusedLess(ids[j], ids[j-1], score, bestRank); j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	return Fused{IDs: ids}
}

// fusedLess 报告 a 是否应排在 b 之前。
func fusedLess(a, b int, score map[int]float64, bestRank map[int]int) bool {
	if score[a] != score[b] {
		return score[a] > score[b]
	}
	if bestRank[a] != bestRank[b] {
		return bestRank[a] < bestRank[b]
	}
	return a < b
}
