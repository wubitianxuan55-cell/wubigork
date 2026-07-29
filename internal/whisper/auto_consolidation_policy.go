// Package whisper — auto_consolidation_policy.go
// 100% 对齐 ackem memory/autoConsolidationPolicy.ts
// 自动记忆整合触发策略：统一 chat/ingest 的整合间隔、事实数门槛与有意义事件密度判断

package whisper

// meaningfulL0 有意义的事件类型集合
var meaningfulL0 = map[string]bool{
	"vulnerable": true,
	"praise":     true,
	"apology":    true,
	"hurtful":    true,
}

// CountRawActiveFacts 统计 raw 层活跃事实
func CountRawActiveFacts(facts []*Fact) int {
	count := 0
	for _, f := range facts {
		if f.FactLayer == "" || f.FactLayer == "raw" {
			if f.IsActive() {
				count++
			}
		}
	}
	return count
}

// CountRawActiveFactsInStore 统计 store 中 raw 层活跃事实
func CountRawActiveFactsInStore(store *FactStore) int {
	return CountRawActiveFacts(store.ListActive())
}

// AutoConsolidationInput 自动巩固评估输入
type AutoConsolidationInput struct {
	TurnsSinceConsolidation int
	RawFactCount            int
	RecentTraceTypes        []string // L0 event type 列表
}

// EvaluateAutoConsolidation 评估是否应触发自动巩固
func EvaluateAutoConsolidation(input AutoConsolidationInput) bool {
	if input.RawFactCount < ConsolidationMinFacts {
		return false
	}

	turns := input.TurnsSinceConsolidation
	if turns < ConsolidationMinTurns {
		return false
	}
	if turns >= ConsolidationMaxTurns {
		return true
	}
	if turns >= ConsolidationIntervalTurns {
		return true
	}

	traces := input.RecentTraceTypes
	if len(traces) >= ConsolidationMinTurns {
		meaningful := 0
		for _, t := range traces {
			if meaningfulL0[t] {
				meaningful++
			}
		}
		if float64(meaningful)/float64(len(traces)) > ConsolidationMeaningfulDensity {
			return true
		}
	}
	return false
}
