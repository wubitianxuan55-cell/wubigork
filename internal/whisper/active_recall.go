// Package whisper — 主动回忆（对齐 ackem memory/activeRecall.ts）
package whisper

import (
	"math"
	"math/rand"
)

// ─── RecallRecord 回忆记录 ────────────────────────────────────

type RecallRecord struct {
	FactID        string `json:"factId"`
	RecalledAtTurn int   `json:"recalledAtTurn"`
}

// ─── ActiveRecall 主动回忆选择器 ──────────────────────────────

type ActiveRecall struct {
	history []RecallRecord
}

func NewActiveRecall() *ActiveRecall {
	return &ActiveRecall{}
}

// SelectRecallCandidate 挑选主动回忆候选（无副作用）
func (ar *ActiveRecall) SelectRecallCandidate(
	store *FactStore,
	currentTurn int,
	rng *float64,
) *struct {
	Prompt string
	FactID string
} {
	roll := rand.Float64()
	if rng != nil {
		roll = *rng
	}
	if roll >= ActiveRecallProbability {
		return nil
	}

	active := store.ListActive()
	// 筛选核心记忆（简化为高weight事实）
	var cores []MemoryFact
	for _, f := range active {
		if f.Weight >= CoreMemoryWeightThreshold {
			cores = append(cores, f)
		}
	}
	if len(cores) == 0 {
		return nil
	}

	// 排除最近已回忆的
	recentIDs := make(map[string]bool)
	for _, r := range ar.history {
		if currentTurn-r.RecalledAtTurn < ActiveRecallMinInterval {
			recentIDs[r.FactID] = true
		}
	}
	var candidates []MemoryFact
	for _, c := range cores {
		if !recentIDs[c.ID] {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	// 基础权重：selfRelevance × turnsSinceRecall
	weights := make([]float64, len(candidates))
	totalW := 0.0
	for i, c := range candidates {
		turnsSinceRecall := ActiveRecallMinInterval
		for _, r := range ar.history {
			if r.FactID == c.ID {
				gap := currentTurn - r.RecalledAtTurn
				if gap < turnsSinceRecall {
					turnsSinceRecall = gap
				}
			}
		}
		weights[i] = c.SelfRelevance * math.Min(1, float64(turnsSinceRecall)/float64(ActiveRecallMinInterval))
		totalW += weights[i]
	}

	if totalW <= 0 {
		return nil
	}

	// 加权随机采样
	r := roll * totalW
	cumulative := 0.0
	selected := candidates[0]
	for i, w := range weights {
		cumulative += w
		if r <= cumulative {
			selected = candidates[i]
			break
		}
	}

	return &struct {
		Prompt string
		FactID string
	}{
		Prompt: formatRecall(selected),
		FactID: selected.ID,
	}
}

// MarkRecalled 标记已回忆
func (ar *ActiveRecall) MarkRecalled(factID string, currentTurn int) {
	ar.history = append(ar.history, RecallRecord{FactID: factID, RecalledAtTurn: currentTurn})
	if len(ar.history) > 100 {
		ar.history = ar.history[len(ar.history)-50:]
	}
}

func (ar *ActiveRecall) GetHistory() []RecallRecord {
	result := make([]RecallRecord, len(ar.history))
	copy(result, ar.history)
	return result
}

// ─── 回忆格式化 ───────────────────────────────────────────────

func formatRecall(fact MemoryFact) string {
	sub := fact.Subject
	sum := fact.Summary
	phrases := []string{
		"说起来，之前记得" + sub + "。" + truncStr(sum, 40),
		"突然想到，你之前说过" + sub + "。现在还是这样吗？",
		"对了，" + sub + "的事我一直记着。" + truncStr(sum, 50),
		"我记得你之前" + sub + "，最近有什么新的变化吗？",
	}
	return truncStr(phrases[rand.Intn(len(phrases))], 120)
}

func truncStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
