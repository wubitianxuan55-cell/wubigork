// Package repos — 关系记忆三表仓库 + 装配闭环测试（v4.3a）
package repos

import (
	"testing"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// 整秒的毫秒时间戳（RFC3339 秒级精度往返无损）
const roundMs = int64(1700000000000)

// mgTestRoot 测试数据根目录：t.TempDir() + 关闭池连接（释放 hermes.db 文件锁，保证清理可删）
func mgTestRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	return dir
}

// seedTwoFacts 落库两条事实（关联表外键依赖它们存在）
func seedTwoFacts(t *testing.T, root string) {
	t.Helper()
	now := time.Now()
	facts := []whisper.MemoryFact{
		{ID: "fact-1", Domain: "preference", Subcategory: "FOOD", Subject: "用户",
			Summary: "用户喜欢吃辣", CreatedAt: now, UpdatedAt: now},
		{ID: "fact-2", Domain: "preference", Subcategory: "DRINK", Subject: "用户",
			Summary: "用户喜欢喝冰红茶", CreatedAt: now, UpdatedAt: now},
	}
	if err := ReplaceFactsInDB(root, facts); err != nil {
		t.Fatalf("ReplaceFactsInDB: %v", err)
	}
}

func TestAssociationsRepo_RoundTrip(t *testing.T) {
	root := mgTestRoot(t)
	seedTwoFacts(t, root)
	repo, err := OpenAssociationsRepo(root)
	if err != nil {
		t.Fatalf("OpenAssociationsRepo: %v", err)
	}

	a1 := whisper.Association{FactIDA: "fact-1", FactIDB: "fact-2", AssociationType: "thematic",
		Strength: 0.8, LastActivatedAt: roundMs}
	if err := repo.SaveAll([]whisper.Association{a1}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	got, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List 应有 1 条, got %d", len(got))
	}
	g := got[0]
	if g.FactIDA != "fact-1" || g.FactIDB != "fact-2" || g.AssociationType != "thematic" {
		t.Errorf("字段不符: %+v", g)
	}
	if g.Strength != 0.8 {
		t.Errorf("strength 不符: %v", g.Strength)
	}
	if g.LastActivatedAt != roundMs {
		t.Errorf("LastActivatedAt 往返不符: %d != %d", g.LastActivatedAt, roundMs)
	}

	// 二元组 UPSERT：同对更新强度，不新增行
	if err := repo.SaveAll([]whisper.Association{
		{FactIDA: "fact-1", FactIDB: "fact-2", AssociationType: "thematic", Strength: 0.95},
	}); err != nil {
		t.Fatalf("SaveAll(更新): %v", err)
	}
	got, _ = repo.List()
	if len(got) != 1 {
		t.Fatalf("二元组 UPSERT 后应仍为 1 条, got %d", len(got))
	}
	if got[0].Strength != 0.95 {
		t.Errorf("更新后 strength 应为 0.95, got %v", got[0].Strength)
	}

	if err := repo.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, _ = repo.List()
	if len(got) != 0 {
		t.Errorf("Clear 后应为空, got %d", len(got))
	}
}

// TestAssociationsRepo_FKSkipsMissingFacts 外键护栏：两端事实未落库的边跳过而非报错
func TestAssociationsRepo_FKSkipsMissingFacts(t *testing.T) {
	root := mgTestRoot(t)
	seedTwoFacts(t, root)
	repo, err := OpenAssociationsRepo(root)
	if err != nil {
		t.Fatalf("OpenAssociationsRepo: %v", err)
	}

	// 一条引用已落库事实、一条引用幽灵事实
	items := []whisper.Association{
		{FactIDA: "fact-1", FactIDB: "fact-2", AssociationType: "thematic", Strength: 0.8},
		{FactIDA: "ghost-a", FactIDB: "ghost-b", AssociationType: "thematic", Strength: 0.5},
	}
	if err := repo.SaveAll(items); err != nil {
		t.Fatalf("SaveAll 不应因外键报错（护栏跳过）: %v", err)
	}
	got, _ := repo.List()
	if len(got) != 1 {
		t.Fatalf("只有引用已落库事实的边应写入, got %d", len(got))
	}
}

func TestHabitsRepo_RoundTripAndUpsert(t *testing.T) {
	root := mgTestRoot(t)
	repo, err := OpenHabitsRepo(root)
	if err != nil {
		t.Fatalf("OpenHabitsRepo: %v", err)
	}

	wk := 3
	exp := int64(roundMs + 3600_000)
	h1 := whisper.UserHabit{
		ID: "habit-1", Type: "dnd", Scope: "short_term", Weekday: &wk,
		HourStart: 22, HourEnd: 6, Confidence: 0.7, OccurrenceCount: 2,
		FirstSeenAt: roundMs, LastConfirmedAt: roundMs, ExpiresAt: &exp,
		Source: "detected", Note: "勿扰", CreatedAt: roundMs, UpdatedAt: roundMs,
	}
	if err := repo.Upsert(h1); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List 应有 1 条, got %d", len(got))
	}
	g := got[0]
	if g.ID != "habit-1" || g.Type != "dnd" || g.Scope != "short_term" {
		t.Errorf("基础字段不符: %+v", g)
	}
	if g.Weekday == nil || *g.Weekday != wk {
		t.Errorf("weekday 指针往返不符: %v", g.Weekday)
	}
	if g.ExpiresAt == nil || *g.ExpiresAt != exp {
		t.Errorf("expiresAt 指针往返不符: %v", g.ExpiresAt)
	}
	if g.SuppressTarget != "" {
		t.Errorf("suppressTarget 应保持空, got %q", g.SuppressTarget)
	}

	// 同主键 upsert：更新字段、不新增行
	h1c := h1
	h1c.Confidence = 0.9
	h1c.SuppressTarget = "target-x"
	if err := repo.Upsert(h1c); err != nil {
		t.Fatalf("Upsert(更新): %v", err)
	}
	got, _ = repo.List()
	if len(got) != 1 {
		t.Fatalf("同主键 upsert 后应仍为 1 条, got %d", len(got))
	}
	if got[0].Confidence != 0.9 || got[0].SuppressTarget != "target-x" {
		t.Errorf("upsert 更新未生效: %+v", got[0])
	}

	// SaveAll 追加另一条
	h2 := whisper.UserHabit{
		ID: "habit-2", Type: "health_reminder", Scope: "long_term",
		HourStart: 8, HourEnd: 9, Confidence: 0.9, OccurrenceCount: 5,
		FirstSeenAt: roundMs, LastConfirmedAt: roundMs,
		Source: "explicit", Note: "喝水", CreatedAt: roundMs, UpdatedAt: roundMs,
	}
	if err := repo.SaveAll([]whisper.UserHabit{h1c, h2}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	got, _ = repo.List()
	if len(got) != 2 {
		t.Fatalf("SaveAll 后应有 2 条, got %d", len(got))
	}
}

func TestTemporalAnchorsRepo_DeleteExpiredBoundary(t *testing.T) {
	root := mgTestRoot(t)
	repo, err := OpenTemporalAnchorsRepo(root)
	if err != nil {
		t.Fatalf("OpenTemporalAnchorsRepo: %v", err)
	}

	anchors := []whisper.TemporalAnchor{
		{ID: "t1", AnchorDate: "2024-12-31", AnchorType: whisper.AnchorFuzzy, Summary: "旧锚点"},
		{ID: "t2", AnchorDate: "2025-01-01", AnchorType: whisper.AnchorRecurring, RecurrenceRule: "YEARLY",
			LinkedFactIDs: []string{"fact-1", "fact-2"}, EmotionalValence: 0.5, EmotionalIntensity: 0.8, Domain: "SOCIAL", Summary: "周年"},
		{ID: "t3", AnchorDate: "2025-06-30", AnchorType: whisper.AnchorMilestone, Summary: "里程碑"},
		{ID: "t4", AnchorDate: "2026-01-01", AnchorType: whisper.AnchorRelationship, Summary: "未来锚点"},
	}
	if err := repo.SaveAll(anchors); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	got, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("List 应有 4 条, got %d", len(got))
	}
	// 复合字段往返
	var t2 *whisper.TemporalAnchor
	for i := range got {
		if got[i].ID == "t2" {
			t2 = &got[i]
		}
	}
	if t2 == nil {
		t.Fatal("t2 未找到")
	}
	if len(t2.LinkedFactIDs) != 2 || t2.LinkedFactIDs[0] != "fact-1" || t2.LinkedFactIDs[1] != "fact-2" {
		t.Errorf("linkedFactIds 往返不符: %v", t2.LinkedFactIDs)
	}
	if t2.RecurrenceRule != "YEARLY" || t2.EmotionalValence != 0.5 || t2.EmotionalIntensity != 0.8 || t2.Domain != "SOCIAL" {
		t.Errorf("t2 复合字段不符: %+v", t2)
	}

	// 边界：before 为 "2025-01-01" → 仅删 anchor_date < "2025-01-01" 的（t1）
	n, err := repo.DeleteExpired("2025-01-01")
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if n != 1 {
		t.Fatalf("DeleteExpired(\"2025-01-01\") 应删 1 条, got %d", n)
	}
	// 再删 < "2026-01-01" → t2、t3
	n, _ = repo.DeleteExpired("2026-01-01")
	if n != 2 {
		t.Fatalf("DeleteExpired(\"2026-01-01\") 应删 2 条, got %d", n)
	}
	got, _ = repo.List()
	if len(got) != 1 || got[0].ID != "t4" {
		t.Fatalf("应只剩 t4, got %+v", got)
	}

	if err := repo.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, _ = repo.List()
	if len(got) != 0 {
		t.Errorf("Clear 后应为空, got %d", len(got))
	}
}

// TestReplaceFactsInDB_SurvivesAssociationsFK 关键回归：关联表有行时，
// 事实全表替换（DELETE+重插）必须成功（事务内延迟外键检查）。
func TestReplaceFactsInDB_SurvivesAssociationsFK(t *testing.T) {
	root := mgTestRoot(t)
	seedTwoFacts(t, root)
	assocRepo, err := OpenAssociationsRepo(root)
	if err != nil {
		t.Fatalf("OpenAssociationsRepo: %v", err)
	}
	if err := assocRepo.SaveAll([]whisper.Association{
		{FactIDA: "fact-1", FactIDB: "fact-2", AssociationType: "thematic", Strength: 0.8},
	}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}

	// 同批事实再次全表替换（app 每次持久化都走这条路）——必须成功
	now := time.Now()
	facts := []whisper.MemoryFact{
		{ID: "fact-1", Domain: "preference", Subcategory: "FOOD", Subject: "用户",
			Summary: "用户喜欢吃辣", CreatedAt: now, UpdatedAt: now},
		{ID: "fact-2", Domain: "preference", Subcategory: "DRINK", Subject: "用户",
			Summary: "用户喜欢喝冰红茶", CreatedAt: now, UpdatedAt: now},
	}
	if err := ReplaceFactsInDB(root, facts); err != nil {
		t.Fatalf("有关联行时 ReplaceFactsInDB 应成功（延迟外键）: %v", err)
	}

	// 关联行仍在
	got, _ := assocRepo.List()
	if len(got) != 1 {
		t.Fatalf("事实替换后关联应保留, got %d", len(got))
	}

	// 清空结构化数据（含 memory_facts 与 memory_associations）也必须成功
	if err := db.ClearStructuredData(root); err != nil {
		t.Fatalf("ClearStructuredData 应成功（延迟外键）: %v", err)
	}
	got, _ = assocRepo.List()
	if len(got) != 0 {
		t.Errorf("清空后关联应为空, got %d", len(got))
	}
}

// TestCompanionState_MemoryGraphRoundTrip 装配闭环：SaveCompanionStateToDB 落三表，
// LoadCompanionStateFromDB 读回回填 State（表为规范存储）。
func TestCompanionState_MemoryGraphRoundTrip(t *testing.T) {
	root := mgTestRoot(t)
	seedTwoFacts(t, root)

	wk := 2
	state := whisper.FullState{
		Version: "1",
		Emotion: whisper.EmotionState{PrimaryLabel: "CALM_RATIONAL"},
		Associations: []whisper.Association{
			{FactIDA: "fact-1", FactIDB: "fact-2", AssociationType: "thematic", Strength: 0.8, LastActivatedAt: roundMs},
		},
		Habits: []whisper.UserHabit{
			{ID: "habit-1", Type: "dnd", Scope: "short_term", Weekday: &wk,
				HourStart: 22, HourEnd: 6, Confidence: 0.7, OccurrenceCount: 1,
				FirstSeenAt: roundMs, LastConfirmedAt: roundMs, Source: "detected",
				Note: "勿扰", CreatedAt: roundMs, UpdatedAt: roundMs},
		},
		TemporalAnchors: []whisper.TemporalAnchor{
			{ID: "anchor-1", AnchorDate: "2025-06-15", AnchorType: whisper.AnchorRecurring,
				RecurrenceRule: "YEARLY", LinkedFactIDs: []string{"fact-1"},
				EmotionalValence: 0.4, EmotionalIntensity: 0.6, Domain: "SOCIAL", Summary: "相识周年"},
		},
	}

	if err := SaveCompanionStateToDB(root, "sess-1", state); err != nil {
		t.Fatalf("SaveCompanionStateToDB: %v", err)
	}
	loaded, err := LoadCompanionStateFromDB(root, "sess-1")
	if err != nil {
		t.Fatalf("LoadCompanionStateFromDB: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded 为 nil")
	}
	if len(loaded.Associations) != 1 || loaded.Associations[0].FactIDA != "fact-1" ||
		loaded.Associations[0].Strength != 0.8 || loaded.Associations[0].LastActivatedAt != roundMs {
		t.Errorf("Associations 往返不符: %+v", loaded.Associations)
	}
	if len(loaded.Habits) != 1 || loaded.Habits[0].ID != "habit-1" ||
		loaded.Habits[0].Weekday == nil || *loaded.Habits[0].Weekday != wk {
		t.Errorf("Habits 往返不符: %+v", loaded.Habits)
	}
	if len(loaded.TemporalAnchors) != 1 || loaded.TemporalAnchors[0].ID != "anchor-1" ||
		loaded.TemporalAnchors[0].RecurrenceRule != "YEARLY" ||
		len(loaded.TemporalAnchors[0].LinkedFactIDs) != 1 {
		t.Errorf("TemporalAnchors 往返不符: %+v", loaded.TemporalAnchors)
	}

	// JSON 回退：清空三表后，state_json 内嵌字段仍能回填
	for _, clear := range []func() error{
		func() error { a, _ := OpenAssociationsRepo(root); return a.Clear() },
		func() error { h, _ := OpenHabitsRepo(root); return h.Clear() },
		func() error { t2, _ := OpenTemporalAnchorsRepo(root); return t2.Clear() },
	} {
		if err := clear(); err != nil {
			t.Fatalf("清表失败: %v", err)
		}
	}
	fallback, err := LoadCompanionStateFromDB(root, "sess-1")
	if err != nil {
		t.Fatalf("LoadCompanionStateFromDB(fallback): %v", err)
	}
	if len(fallback.Associations) != 1 || len(fallback.Habits) != 1 || len(fallback.TemporalAnchors) != 1 {
		t.Fatalf("JSON 回退应保留三组数据, got assoc=%d habit=%d anchor=%d",
			len(fallback.Associations), len(fallback.Habits), len(fallback.TemporalAnchors))
	}
}
