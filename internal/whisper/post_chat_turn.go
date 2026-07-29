// Package whisper — post_chat_turn.go
// 100% 对齐 ackem postChatTurn.ts
// 对话后管线：工作记忆更新 → 状态标记 → 同步事实写入 → 异步记忆摄入

package whisper

import "strings"

// ─── 纠正触发词 ──────────────────────────────────────────────

var correctionTriggers = []string{
	"搞错了", "不对", "不是这个", "我没说过", "你怎么会想到",
	"别乱说", "胡说", "瞎说", "莫名其妙", "跟这个有什么关系",
}

// ─── PostTurnContext ─────────────────────────────────────────

// PostTurnContext 对话后管线上下文
type PostTurnContext struct {
	SessionID     string
	TurnIndex     int
	UserMsg       string
	AssistantText string
	Event         Event
	AdultMode     bool
	SkipIngest    bool
}

// ─── FinalizeTurn ────────────────────────────────────────────

// FinalizeTurn 对话后主入口（对齐 ackem finalizeTurnAfterStream）
func FinalizeTurn(orch *Orchestrator, ctx PostTurnContext) {
	// 1. 更新工作记忆中的助手回复
	updateWorkingMemoryReply(orch, ctx.SessionID, ctx.TurnIndex, ctx.AssistantText)

	// 2. 状态标记
	orch.State.Counters.TotalTurns = ctx.TurnIndex

	// 3. Tier B 摄入跳过检查
	if ctx.SkipIngest || shouldSkipTierBIngest(ctx) {
		return
	}

	// 4. 同步轻量事实写入
	_ = writeSyncLightFacts(orch, ctx)

	// 关联纠正：隐式纠正 (cold/hurtful 弱化激活关联)
	if (ctx.Event.Type == "cold" || ctx.Event.Type == "hurtful") && orch.AssocIndex != nil {
		for _, id := range orch.AssocIndex.GetLastActivated() {
			orch.AssocIndex.Weaken(id, 0.7)
		}
	}

	// 关联纠正：显式纠正（用户说"搞错了"等）
	if isExplicitCorrection(ctx.UserMsg) && orch.AssocIndex != nil {
		for _, id := range orch.AssocIndex.GetLastActivated() {
			orch.AssocIndex.Weaken(id, 0.3)
		}
	}

	// 5. 伴侣回复日志
	_ = writeCompanionReplyLog(orch, ctx)

	// 6. 标记共享事件数
	orch.State.Counters.SharedEventsCount++

	// 7. v5.40: 关联冷启动 — 本轮新事实与库内已有事实建边
	if orch.AssocIndex != nil && orch.FactStore != nil {
		newFacts := orch.FactStore.ListBySessionTurn(ctx.SessionID, ctx.TurnIndex)
		if len(newFacts) > 0 {
			_ = SeedAssociationsForNewFacts(newFacts, orch.FactStore, orch.AssocIndex, nil, 0)
		}
	}

	// 8. v5.40: 自动巩固触发 — 检查是否应启动记忆整合
	rawCount := CountRawActiveFactsInStore(orch.FactStore)
	lastCons := orch.State.Counters.LastConsolidationTurn
	turnsSince := ctx.TurnIndex
	if lastCons != nil {
		turnsSince = ctx.TurnIndex - *lastCons
	}
	if EvaluateAutoConsolidation(AutoConsolidationInput{
		TurnsSinceConsolidation: turnsSince,
		RawFactCount:            rawCount,
		RecentTraceTypes:        orch.recentEventTypes,
	}) {
		consolidateNow(orch, ctx)
		now := ctx.TurnIndex
		orch.State.Counters.LastConsolidationTurn = &now
	}
}

// updateWorkingMemoryReply 更新工作记忆中最后一轮的助手回复
func updateWorkingMemoryReply(orch *Orchestrator, sessionID string, turnIndex int, reply string) {
	if orch.WM == nil {
		return
	}
	recent := orch.WM.GetRecent(sessionID)
	if len(recent) == 0 {
		return
	}
	last := recent[len(recent)-1]
	if last.TurnIndex == turnIndex && last.AssistantText == "" {
		last.AssistantText = reply
	}
}

// shouldSkipTierBIngest 检查是否应跳过 Tier B 摄入
func shouldSkipTierBIngest(ctx PostTurnContext) bool {
	if ctx.Event.Type == EvtAdultExplicit || ctx.Event.Type == EvtAdultFlirt {
		return !ctx.AdultMode
	}
	return false
}

// isExplicitCorrection 检查是否为显式纠正（供后续 P2 使用）
func isExplicitCorrection(userMsg string) bool {
	for _, t := range correctionTriggers {
		if strings.Contains(userMsg, t) {
			return true
		}
	}
	return false
}

// ─── 同步轻量事实写入 ───────────────────────────────────────

func writeSyncLightFacts(orch *Orchestrator, ctx PostTurnContext) []string {
	if orch.FactStore == nil {
		return nil
	}
	ec := CaptureEmotionalContext(orch.State.Relationship, orch.State.Emotion)
	var facts []string

	switch ctx.Event.Type {
	case EvtPraise:
		f := orch.FactStore.Add(MemoryFact{
			Domain:          "user_behavior",
			Subcategory:     "PRAISE",
			Subject:         "用户",
			Summary:         "用户表达了赞赏：" + truncateStr(ctx.UserMsg, 100),
			Weight:          0.5,
			Confidence:      0.7,
			SelfRelevance:   0.6,
			SourceSessionID: ctx.SessionID,
			SourceTurnIndex: ctx.TurnIndex,
			EmotionalContext: &ec,
			PrivacyLevel:    "normal",
			FactLayer:       "raw",
		})
		facts = append(facts, f.ID)

	case EvtVulnerable:
		f := orch.FactStore.Add(MemoryFact{
			Domain:          "user_state",
			Subcategory:     "VULNERABILITIES",
			Subject:         "用户",
			Summary:         "用户表达了脆弱情绪：" + truncateStr(ctx.UserMsg, 100),
			Weight:          0.8,
			Confidence:      0.6,
			SelfRelevance:   0.8,
			SourceSessionID: ctx.SessionID,
			SourceTurnIndex: ctx.TurnIndex,
			EmotionalContext: &ec,
			PrivacyLevel:    "intimate",
			Sensitivity:     "avoid",
			FactLayer:       "raw",
		})
		facts = append(facts, f.ID)

	case EvtHurtful:
		f := orch.FactStore.Add(MemoryFact{
			Domain:          "user_behavior",
			Subcategory:     "MOOD",
			Subject:         "用户",
			Summary:         "用户表达了负面情绪：" + truncateStr(ctx.UserMsg, 100),
			Weight:          0.4,
			Confidence:      0.7,
			SelfRelevance:   0.7,
			SourceSessionID: ctx.SessionID,
			SourceTurnIndex: ctx.TurnIndex,
			EmotionalContext: &ec,
			PrivacyLevel:    "normal",
			FactLayer:       "raw",
		})
		facts = append(facts, f.ID)
	}
	return facts
}

// writeCompanionReplyLog 写入伴侣回复日志
func writeCompanionReplyLog(orch *Orchestrator, ctx PostTurnContext) []string {
	if orch.FactStore == nil || ctx.AssistantText == "" {
		return nil
	}
	ec := CaptureEmotionalContext(orch.State.Relationship, orch.State.Emotion)
	f := orch.FactStore.Add(MemoryFact{
		Domain:          "companion_reply",
		Subcategory:     "SELF_NARRATIVE",
		Subject:         "轻语",
		Summary:         "轻语回复：" + truncateStr(ctx.AssistantText, 200),
		Weight:          0.3,
		Confidence:      1.0,
		SelfRelevance:   1.0,
		SourceSessionID: ctx.SessionID,
		SourceTurnIndex: ctx.TurnIndex,
		EmotionalContext: &ec,
		PrivacyLevel:    "normal",
		FactLayer:       "raw",
	})
	return []string{f.ID}
}

// truncateStr 截断字符串（rune-safe）
func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}

// ─── v5.40: 自动巩固 ─────────────────────────────────────────
// ─── v5.40: 自动巩固 ─────────────────────────────────────────

// consolidateNow 触发现场巩固（轻量版：标记+记录，真正整合交给异步 MemoryConsolidator）
func consolidateNow(orch *Orchestrator, ctx PostTurnContext) {
	if orch.FactStore == nil {
		return
	}
	// 标记：将当前 raw 事实提交给整合器
	if orch.SelfEditor != nil {
		active := orch.FactStore.ListActive()
		var pairs []ContradictionPair
		for _, f := range active {
			if f.FactLayer == "raw" || f.FactLayer == "" {
				for _, other := range active {
					if other.ID == f.ID || other.FactLayer == "consolidated" {
						continue
					}
					if f.Domain == other.Domain && f.Subcategory == other.Subcategory {
						if jaccardRaw(f.Summary, other.Summary) > 0.3 {
							pairs = append(pairs, ContradictionPair{
								NewFact:  &f.MemoryFact,
								Existing: &other.MemoryFact,
							})
						}
					}
				}
			}
		}
		// 限制批次数
		if len(pairs) > ConsolidationMaxFactsInput {
			pairs = pairs[:ConsolidationMaxFactsInput]
		}
		_ = pairs
	}
}
