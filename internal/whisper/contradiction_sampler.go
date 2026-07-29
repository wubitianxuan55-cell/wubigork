// Package whisper — contradiction_sampler.go
// 100% 对齐 ackem memory/factContradictionSampler.ts
// 从存量事实中抽样矛盾候选对（用于定期矛盾检测）

package whisper

import "sort"

const (
	contradictionMinWeight      = 1.0
	contradictionSimilarityThresh = 0.25
	contradictionSamplePairs    = 5
)

// ContradictionCandidate 矛盾候选对
type ContradictionCandidate struct {
	NewFact  *Fact
	Existing *Fact
}

// SampleSimilarFactPairs 从存量事实中抽样 Jaccard 相似的候选对
func SampleSimilarFactPairs(fs *FactStore, maxPairs int) []ContradictionCandidate {
	if maxPairs <= 0 {
		maxPairs = contradictionSamplePairs
	}

	active := fs.ListActive()
	// 排除 consolidated 层，按 updatedAt 降序
	var filtered []*Fact
	for _, f := range active {
		if f.FactLayer != "consolidated" {
			filtered = append(filtered, f)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})

	type pair struct {
		newFact  *Fact
		existing *Fact
		sim      float64
	}
	var pairs []pair

	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			a, b := filtered[i], filtered[j]
			if a.Subcategory != b.Subcategory {
				continue
			}
			if a.Weight < contradictionMinWeight || b.Weight < contradictionMinWeight {
				continue
			}
			sim := factCharJaccard(a, b)
			if sim < contradictionSimilarityThresh {
				continue
			}
			// 较新的作为 newFact
			newer, older := a, b
			if b.UpdatedAt.After(a.UpdatedAt) {
				newer, older = b, a
			}
			pairs = append(pairs, pair{newFact: newer, existing: older, sim: sim})
		}
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].sim > pairs[j].sim })
	if len(pairs) > maxPairs {
		pairs = pairs[:maxPairs]
	}

	result := make([]ContradictionCandidate, len(pairs))
	for i, p := range pairs {
		result[i] = ContradictionCandidate{NewFact: p.newFact, Existing: p.existing}
	}
	return result
}

// factCharJaccard 两条事实的字符集 Jaccard 相似度
func factCharJaccard(a, b *Fact) float64 {
	aSet := make(map[rune]bool)
	for _, r := range a.Subject + a.Summary {
		aSet[r] = true
	}
	bSet := make(map[rune]bool)
	for _, r := range b.Subject + b.Summary {
		bSet[r] = true
	}
	intersect := 0
	for r := range aSet {
		if bSet[r] {
			intersect++
		}
	}
	union := len(aSet) + len(bSet) - intersect
	if union == 0 {
		return 0
	}
	return float64(intersect) / float64(union)
}
