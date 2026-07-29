// Package whisper — prepare_turn_context.go
// 对齐 ackem engine/prepareTurnContext.ts
// 回合上下文准备：预计算检索参数（无 embedding 路径）

package whisper

import "time"

// PreparedTurnContext 预计算上下文
type PreparedTurnContext struct {
	Retrieval    RetrievalResult
	RelevanceHint RelevanceHint
	TemporalCtx  TemporalContext
	EmbedMs      int64
	RetrieveMs   int64
}

// PrepareTurnContext 准备回合上下文
func PrepareTurnContext(
	msg string,
	state FullState,
	fs *FactStore,
	retriever *MemoryRetriever,
	sessionID string,
	turnIndex int,
	memoryBudgetChars int,
	adultMode bool,
) *PreparedTurnContext {
	

	// 计算检索预算（预留工作记忆空间）
	retrievalBudget := memoryBudgetChars - WorkingMemoryCharBudget
	if retrievalBudget < 1500 {
		retrievalBudget = 1500
	}

	// 相关性提示
	hint := ComputeRelevanceHint(state.Relationship, state.Emotion, turnIndex)

	// 时间上下文
	gapHours := time.Since(state.LastActive).Hours()
	temporalCtx := BuildTemporalContext(gapHours, time.Now())

	// 检索
	tEmbed := time.Now()
	memRetrieval := retriever.Retrieve(msg, hint, retrievalBudget, state.Emotion.Aff/100, state.Emotion.Aff, &temporalCtx, sessionID, adultMode)
	retrieveMs := time.Since(tEmbed).Milliseconds()

	return &PreparedTurnContext{
		Retrieval:    memRetrieval,
		RelevanceHint: hint,
		TemporalCtx:  temporalCtx,
		EmbedMs:      0, // 无 embedding
		RetrieveMs:   retrieveMs,
	}
}
