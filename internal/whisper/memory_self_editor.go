// Package whisper — memory_self_editor.go
// 100% 对齐 ackem memory/memorySelfEditor.ts
// 记忆自编辑：批量矛盾检测+自主更新/合并/退役事实，记录编辑日志

package whisper

import (
	"time"
)

// EditLogEntry 编辑日志条目
type EditLogEntry struct {
	At            string `json:"at"`
	Action        string `json:"action"`
	TargetFactID  string `json:"targetFactId"`
	RelatedFactID string `json:"relatedFactId,omitempty"`
	Reason        string `json:"reason"`
}

// MemorySelfEditor 记忆自编辑器
type MemorySelfEditor struct {
	detector *ContradictionDetector
	editLog  []EditLogEntry
}

// NewMemorySelfEditor 创建记忆自编辑器
func NewMemorySelfEditor() *MemorySelfEditor {
	return &MemorySelfEditor{
		detector: NewContradictionDetector(),
	}
}

// BatchResolve 批量检查多条新事实与相似已有事实，一次 LLM 调用完成所有判断
func (e *MemorySelfEditor) BatchResolve(pairs []ContradictionPair, store *FactStore, llm LlmClient) []string {
	var results []string

	// 过滤无效配对：新事实不能是 consolidated 层
	var validPairs []ContradictionPair
	for _, p := range pairs {
		if p.NewFact.ID != p.Existing.ID && p.NewFact.FactLayer != "consolidated" {
			validPairs = append(validPairs, p)
		}
	}
	if len(validPairs) == 0 {
		return results
	}

	// 批量检定
	var checks []*ContradictionCheck
	if len(validPairs) >= 2 {
		checks = e.detector.CheckBatch(validPairs, llm)
	} else {
		check, err := e.detector.Check(validPairs[0].NewFact, validPairs[0].Existing, llm)
		if err == nil && check != nil {
			checks = []*ContradictionCheck{check}
		}
	}

	for i, check := range checks {
		if check == nil {
			continue
		}
		if i >= len(validPairs) {
			break
		}
		pair := validPairs[i]
		result := e.applyResolution(*check, pair.NewFact, pair.Existing, store)
		if result != "" {
			results = append(results, result)
		}
	}

	return results
}

// applyResolution 执行检定结果
func (e *MemorySelfEditor) applyResolution(check ContradictionCheck, newFact, existing *MemoryFact, store *FactStore) string {
	switch check.Judgment {
	case "reinforce":
		summary := existing.Summary
		if len(newFact.Summary) > len(existing.Summary) {
			summary = newFact.Summary
		}
		weight := existing.Weight
		if newFact.Weight > weight {
			weight = newFact.Weight
		}
		weight += SelfEditReinforceWeightBoost

		store.UpdateFact(existing.ID, map[string]interface{}{
			"summary": summary,
			"weight":  weight,
		})
		store.RetireFact(newFact.ID)
		e.log("merge_reinforce", newFact.ID, existing.ID, check.Reason)
		return "强化并合并：" + check.Reason

	case "conflict":
		switch check.Action {
		case "keep_new":
			store.RetireFact(existing.ID)
			e.log("retire_old_conflict", existing.ID, newFact.ID, check.Reason)
			return "退役旧事实（冲突，保留新）：" + check.Reason
		case "keep_old":
			store.RetireFact(newFact.ID)
			e.log("retire_new_conflict", newFact.ID, existing.ID, check.Reason)
			return "退役新事实（冲突，保留旧）：" + check.Reason
		case "merge":
			summary := existing.Summary
			if len(newFact.Summary) >= len(existing.Summary) {
				summary = newFact.Summary
			}
			weight := existing.Weight
			if newFact.Weight > weight {
				weight = newFact.Weight
			}
			store.UpdateFact(existing.ID, map[string]interface{}{
				"summary": summary,
				"weight":  weight,
			})
			store.RetireFact(newFact.ID)
			e.log("merge_conflict", newFact.ID, existing.ID, check.Reason)
			return "合并冲突事实：" + check.Reason
		case "flag":
			e.log("flag", newFact.ID, existing.ID, check.Reason)
			return "标记为需人工确认：" + check.Reason
		}
	}
	return ""
}

func (e *MemorySelfEditor) log(action, targetFactID, relatedFactID, reason string) {
	e.editLog = append(e.editLog, EditLogEntry{
		At:            time.Now().Format(time.RFC3339),
		Action:        action,
		TargetFactID:  targetFactID,
		RelatedFactID: relatedFactID,
		Reason:        reason,
	})
	if len(e.editLog) > SelfEditLogMax {
		e.editLog = e.editLog[len(e.editLog)-SelfEditLogKeep:]
	}
}

// GetEditLog 获取编辑日志
func (e *MemorySelfEditor) GetEditLog() []EditLogEntry {
	result := make([]EditLogEntry, len(e.editLog))
	copy(result, e.editLog)
	return result
}

// ClearLog 清空日志
func (e *MemorySelfEditor) ClearLog() {
	e.editLog = nil
}
