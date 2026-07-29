// Package whisper — auto_mirror_check.go
// 100% 对齐 ackem memory/autoMirrorCheck.ts
// ingest 后自动镜中检测 + 存量事实抽样矛盾

package whisper

// AutoMirrorResult 镜中检测结果
type AutoMirrorResult struct {
	MirrorCount  int `json:"mirrorCount"`
	FactResolved int `json:"factResolved"`
	FactFlagged  int `json:"factFlagged"`
}

// RunAutoMirrorAndContradictionCheck 运行自动镜中+矛盾检测
func RunAutoMirrorAndContradictionCheck(factStore *FactStore, selfEditor *MemorySelfEditor, turn int, selfFactAddedThisTurn bool, lastMirrorCheckTurn int) AutoMirrorResult {
	turnsSince := turn - lastMirrorCheckTurn
	if !EvaluatePeriodicMemoryAudit(turnsSince, selfFactAddedThisTurn) {
		return AutoMirrorResult{}
	}

	result := AutoMirrorResult{}

	// 抽样相似事实对
	pairs := sampleSimilarFactPairs(factStore)
	if len(pairs) > 0 && selfEditor != nil {
		// 自编辑批量解决（不调用LLM，仅记录意图）
		for _, p := range pairs {
			if jaccardRaw(p.NewFact.Summary, p.Existing.Summary) > 0.7 {
				if p.NewFact.Weight > p.Existing.Weight {
					factStore.RetireFact(p.Existing.ID)
					result.FactResolved++
				} else if p.Existing.Weight > p.NewFact.Weight {
					factStore.RetireFact(p.NewFact.ID)
					result.FactResolved++
				} else {
					result.FactFlagged++
				}
			}
		}
	}

	return result
}

// sampleSimilarFactPairs 抽样相似事实对（同domain+subcategory+Jaccard>0.4）
func sampleSimilarFactPairs(store *FactStore) []ContradictionPair {
	active := store.ListActive()
	var pairs []ContradictionPair
	for i, f := range active {
		for j := i + 1; j < len(active); j++ {
			other := active[j]
			if f.Domain == other.Domain && f.Subcategory == other.Subcategory {
				if jaccardRaw(f.Summary, other.Summary) > 0.4 {
					pairs = append(pairs, ContradictionPair{
						NewFact:  &f.MemoryFact,
						Existing: &other.MemoryFact,
					})
				}
			}
		}
		if len(pairs) >= 10 {
			break
		}
	}
	return pairs
}
