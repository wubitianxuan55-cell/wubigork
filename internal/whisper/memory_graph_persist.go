// Package whisper — memory_graph_persist.go
// v4.3a 会客厅·记忆持久化闭环：内存态 ↔ FullState 的双向同步 + 重启后关联图重建。
//
// 落库本身由 app 层经 repos 完成（whisper 主包不直接依赖 db/repos，避免循环引用）：
//   - 保存路径：FinalizeTurn 末尾 syncMemoryGraphToState 把 AssocIndex/HabitsStore
//     快照进 State.Associations/State.Habits → app persistStateAsync 的
//     CloneFullState 快照随 companion_state 落库时，repos 装配进三表；
//   - 恢复路径：app restoreWhisperState 读回三表填入 State.Associations/Habits/
//     TemporalAnchors → 首个回合 PreLLMTurn 开头 restoreMemoryGraphFromState 重建
//     索引/习惯库并调用 ReseedAssociationGraph（失败仅记日志，不阻断启动）。

package whisper

import "log/slog"

// restoreMemoryGraphFromState 启动后首个回合：从 State 回填关联索引/习惯库，
// 并基于记忆事实重建关联图（ReseedAssociationGraph，fail-open）。
// 幂等：memoryGraphRestored 置位后不再执行。
func (o *Orchestrator) restoreMemoryGraphFromState() {
	if o == nil || o.memoryGraphRestored {
		return
	}
	o.memoryGraphRestored = true

	// 1. 关联索引重建（表数据已随 companion_state 恢复进 State.Associations）
	if o.AssocIndex != nil && o.AssocIndex.Count() == 0 && len(o.State.Associations) > 0 {
		for _, a := range o.State.Associations {
			o.AssocIndex.Add(a)
		}
	}

	// 2. 习惯库回填（保留原 ID/时间戳，不走检测逻辑）
	if o.HabitsStore != nil && o.HabitsStore.Count() == 0 && len(o.State.Habits) > 0 {
		for _, h := range o.State.Habits {
			o.HabitsStore.Upsert(h)
		}
	}

	// 3. 关联图重建：仅对"恢复过的会话"执行（有历史轮次或已恢复关联），
	//    冷启动补边；全新会话保持原有逐轮 SeedAssociationsForNewFacts 行为。
	if o.FactStore == nil || o.AssocIndex == nil {
		return
	}
	if o.State.Counters.TotalTurns == 0 && len(o.State.Associations) == 0 {
		return
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("whisper: ReseedAssociationGraph 重建关联图 panic 已恢复", "sessionID", o.SessionID, "panic", r)
			}
		}()
		result := ReseedAssociationGraph(o.FactStore, o.AssocIndex, nil)
		slog.Info("whisper: 重启后重建关联图完成", "sessionID", o.SessionID,
			"edgesCreated", result.EdgesCreated, "factsConsidered", result.FactsConsidered,
			"orphansLinked", result.OrphansLinked)
	}()
}

// syncMemoryGraphToState 回合末把内存关联索引/习惯库快照进 State
// （时间锚点直接以 State.TemporalAnchors 为内存态，无需同步）。
// 在 FinalizeTurn 末尾调用：app 的 persistStateAsync 随后 CloneFullState 快照落库，
// memory_associations / user_habits 随 companion_state 装配更新。
func (o *Orchestrator) syncMemoryGraphToState() {
	if o == nil {
		return
	}
	if o.AssocIndex != nil {
		o.State.Associations = o.AssocIndex.ListAll()
	}
	if o.HabitsStore != nil {
		all := o.HabitsStore.All()
		o.State.Habits = make([]UserHabit, 0, len(all))
		for _, h := range all {
			if h != nil {
				o.State.Habits = append(o.State.Habits, *h)
			}
		}
	}
}
