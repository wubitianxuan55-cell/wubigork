// Package whisper — 关系记忆装配测试（v4.3a）：restore/sync 双向同步 + Reseed 重建
package whisper

import "testing"

// 整秒的毫秒时间戳（RFC3339 秒级精度往返无损）
const roundTs = int64(1700000000000)

func TestMemoryGraphRestore_BackfillsAndReseeds(t *testing.T) {
	orch := NewOrchestrator("sess-mg", PersonalityPresets[0])

	// 模拟重启恢复：事实已恢复进 FactStore，State 装载了关联/习惯
	f1 := orch.FactStore.Add(MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "用户喜欢吃辣", Weight: 2, Confidence: 0.8, SelfRelevance: 0.8,
	})
	f2 := orch.FactStore.Add(MemoryFact{
		Domain: "preference", Subcategory: "DRINK", Subject: "用户",
		Summary: "用户喜欢喝冰红茶", Weight: 2, Confidence: 0.8, SelfRelevance: 0.8,
	})
	orch.FactStore.Add(MemoryFact{ // f3：孤儿事实（无关联），Reseed 应为其建边
		Domain: "preference", Subcategory: "SNACK", Subject: "用户",
		Summary: "用户喜欢吃甜食", Weight: 2, Confidence: 0.8, SelfRelevance: 0.8,
	})

	wk := 3
	orch.State.Associations = []Association{
		{FactIDA: f1.ID, FactIDB: f2.ID, AssociationType: "thematic", Strength: 0.7, LastActivatedAt: roundTs},
	}
	orch.State.Habits = []UserHabit{
		{ID: "habit-1", Type: "health_reminder", Scope: "long_term", Weekday: &wk,
			HourStart: 8, HourEnd: 9, Confidence: 0.9, OccurrenceCount: 5,
			FirstSeenAt: roundTs, LastConfirmedAt: roundTs, Source: "explicit",
			Note: "喝水", CreatedAt: roundTs, UpdatedAt: roundTs},
	}
	orch.State.Counters.TotalTurns = 12 // 恢复过的会话（重启后非首轮）

	orch.restoreMemoryGraphFromState()

	// 1. 关联索引重建
	if orch.AssocIndex.Count() < 1 {
		t.Fatalf("关联索引应已重建, got %d", orch.AssocIndex.Count())
	}
	edges := orch.AssocIndex.GetAssociations(f1.ID)
	found := false
	for _, e := range edges {
		if (e.FactIDA == f1.ID && e.FactIDB == f2.ID) || (e.FactIDA == f2.ID && e.FactIDB == f1.ID) {
			found = true
		}
	}
	if !found {
		t.Errorf("f1-f2 边应存在于索引: %+v", edges)
	}

	// 2. 习惯库回填（保留原 ID）
	if orch.HabitsStore.Count() != 1 || orch.HabitsStore.All()[0].ID != "habit-1" {
		t.Fatalf("习惯库应回填 habit-1, got %+v", orch.HabitsStore.All())
	}

	// 3. Reseed 已运行：f3 是孤儿（无关联），与 f1/f2 文本重叠 → 新增边
	if orch.AssocIndex.Count() < 3 {
		t.Errorf("Reseed 应新增 f3 的边, 总数应 >= 3, got %d", orch.AssocIndex.Count())
	}

	// 幂等：再次调用不重复重建
	before := orch.AssocIndex.Count()
	habitsBefore := orch.HabitsStore.Count()
	orch.restoreMemoryGraphFromState()
	if orch.AssocIndex.Count() != before || orch.HabitsStore.Count() != habitsBefore {
		t.Errorf("restoreMemoryGraphFromState 应幂等: assoc %d->%d habit %d->%d",
			before, orch.AssocIndex.Count(), habitsBefore, orch.HabitsStore.Count())
	}
}

// TestMemoryGraphRestore_SkippedForFreshSession 全新会话保持原行为：不重建、不补边
func TestMemoryGraphRestore_SkippedForFreshSession(t *testing.T) {
	orch := NewOrchestrator("sess-fresh", PersonalityPresets[0])
	orch.FactStore.Add(MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "用户喜欢吃辣", Weight: 2, Confidence: 0.8,
	})
	orch.FactStore.Add(MemoryFact{
		Domain: "preference", Subcategory: "DRINK", Subject: "用户",
		Summary: "用户喜欢喝冰红茶", Weight: 2, Confidence: 0.8,
	})

	orch.restoreMemoryGraphFromState()

	if orch.AssocIndex.Count() != 0 {
		t.Errorf("全新会话（无恢复历史）不应重建关联图, got %d", orch.AssocIndex.Count())
	}
	if orch.HabitsStore.Count() != 0 {
		t.Errorf("全新会话不应回填习惯, got %d", orch.HabitsStore.Count())
	}
}

// TestMemoryGraph_SyncToState 回合末同步：关联/习惯快照进 State 且深拷贝独立
func TestMemoryGraph_SyncToState(t *testing.T) {
	orch := NewOrchestrator("sess-sync", PersonalityPresets[0])

	orch.AssocIndex.Add(Association{FactIDA: "fact-a", FactIDB: "fact-b", AssociationType: "thematic", Strength: 0.6})
	wk := 1
	orch.HabitsStore.Upsert(UserHabit{
		ID: "habit-s", Type: "dnd", Scope: "short_term", Weekday: &wk,
		HourStart: 22, HourEnd: 6, Confidence: 0.7, OccurrenceCount: 1,
		FirstSeenAt: roundTs, LastConfirmedAt: roundTs, Source: "detected",
		CreatedAt: roundTs, UpdatedAt: roundTs,
	})

	orch.syncMemoryGraphToState()

	if len(orch.State.Associations) != 1 || orch.State.Associations[0].Strength != 0.6 {
		t.Errorf("State.Associations 应含 1 条关联: %+v", orch.State.Associations)
	}
	if len(orch.State.Habits) != 1 || orch.State.Habits[0].ID != "habit-s" {
		t.Errorf("State.Habits 应含 habit-s: %+v", orch.State.Habits)
	}

	// CloneFullState 深拷贝：修改副本不影响内存态
	clone := CloneFullState(orch.State)
	clone.Associations[0].Strength = 0.01
	if orch.State.Associations[0].Strength == 0.01 {
		t.Error("CloneFullState 未深拷贝 Associations")
	}
	clone.Habits[0].Confidence = 0.0
	if orch.State.Habits[0].Confidence == 0.0 {
		t.Error("CloneFullState 未深拷贝 Habits")
	}
	clone.TemporalAnchors = append(clone.TemporalAnchors, TemporalAnchor{ID: "x"})
	if len(orch.State.TemporalAnchors) != 0 {
		t.Error("CloneFullState 未深拷贝 TemporalAnchors 切片")
	}
}
