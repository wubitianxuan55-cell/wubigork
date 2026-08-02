package whisper

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// ─── memory_fact: Fact 状态 ──────────────────────────────────

func TestFactIsActive(t *testing.T) {
	active := &Fact{MemoryFact: MemoryFact{Status: "active"}, Active: true}
	if !active.IsActive() {
		t.Error("Active+status=active 应 IsActive")
	}
	inactive := &Fact{MemoryFact: MemoryFact{Status: "active"}, Active: false}
	if inactive.IsActive() {
		t.Error("Active=false 不应 IsActive")
	}
	archived := &Fact{MemoryFact: MemoryFact{Status: "archived"}, Active: true}
	if archived.IsActive() {
		t.Error("status=archived 不应 IsActive")
	}
}

func TestFactIsCore(t *testing.T) {
	core := &Fact{MemoryFact: MemoryFact{Status: "active"}, Active: true, RawTier: "core"}
	if !core.IsCore() {
		t.Error("core tier 应 IsCore")
	}
	normal := &Fact{MemoryFact: MemoryFact{Status: "active"}, Active: true, RawTier: "normal"}
	if normal.IsCore() {
		t.Error("normal tier 不应 IsCore")
	}
	// 非激活的 core 也不是 core
	inactiveCore := &Fact{MemoryFact: MemoryFact{Status: "active"}, Active: false, RawTier: "core"}
	if inactiveCore.IsCore() {
		t.Error("未激活的 core 不应 IsCore")
	}
}

// ─── memory_fact: 新增与去重 ─────────────────────────────────

func TestFactStore_AddAndGet(t *testing.T) {
	fs := NewFactStore()
	f := fs.Add(MemoryFact{Domain: "preference", Subcategory: "FOOD", Subject: "用户", Summary: "喜欢吃辣", Weight: 2})
	if f.ID == "" {
		t.Fatal("新增事实无 ID")
	}
	if f.Status != "active" || !f.Active {
		t.Error("新增事实应 active")
	}
	got := fs.Get(f.ID)
	if got == nil || got.Summary != "喜欢吃辣" {
		t.Error("Get 返回不一致")
	}
	if fs.Count() != 1 {
		t.Errorf("Count = %d, want 1", fs.Count())
	}
}

func TestFactStore_DedupMergesWeight(t *testing.T) {
	fs := NewFactStore()
	f1 := fs.Add(MemoryFact{Domain: "preference", Subcategory: "FOOD", Subject: "用户", Summary: "用户喜欢吃辣", Weight: 1})
	// 相同事实再次添加 → 合并（weight boost），不新增
	f2 := fs.Add(MemoryFact{Domain: "preference", Subcategory: "FOOD", Subject: "用户", Summary: "用户喜欢吃辣", Weight: 1})
	if f1.ID != f2.ID {
		t.Errorf("去重后应返回同一事实: %s vs %s", f1.ID, f2.ID)
	}
	if fs.Count() != 1 {
		t.Errorf("去重后 Count = %d, want 1", fs.Count())
	}
	if f1.Weight < 1.5 {
		t.Errorf("合并后 weight = %f, want >= 1.5", f1.Weight)
	}
}

func TestFactStore_DifferentSummaryNoDedup(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Domain: "preference", Subcategory: "FOOD", Subject: "用户", Summary: "喜欢吃辣", Weight: 1})
	fs.Add(MemoryFact{Domain: "preference", Subcategory: "FOOD", Subject: "用户", Summary: "喜欢看电影", Weight: 1})
	if fs.Count() != 2 {
		t.Errorf("不同摘要应各自新增, Count = %d", fs.Count())
	}
}

func TestFactStore_ListActive_ExcludesRetired(t *testing.T) {
	fs := NewFactStore()
	f1 := fs.Add(MemoryFact{Domain: "preference", Subcategory: "FOOD", Subject: "用户", Summary: "用户喜欢吃辣的食物", Weight: 1})
	fs.Add(MemoryFact{Domain: "interest", Subcategory: "MOVIE", Subject: "用户", Summary: "用户喜欢看科幻电影", Weight: 1})
	fs.RetireFact(f1.ID)
	active := fs.ListActive()
	if len(active) != 1 {
		t.Errorf("退休后活跃数 = %d, want 1", len(active))
	}
	if active[0].ID == f1.ID {
		t.Error("退休事实不应在活跃列表")
	}
}

func TestFactStore_PrivacyFilter(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Subcategory: "SECRET", Subject: "用户", Summary: "亲密信息", Weight: 1, PrivacyLevel: "intimate"})
	fs.Add(MemoryFact{Subcategory: "FOOD", Subject: "用户", Summary: "正常信息", Weight: 1, PrivacyLevel: "normal"})
	// 非 adultMode：过滤 intimate
	filtered := fs.PrivacyFilter(false)
	for _, f := range filtered {
		if f.PrivacyLevel == "intimate" {
			t.Errorf("非 adultMode 应过滤 intimate: %+v", f)
		}
	}
	if len(filtered) != 1 {
		t.Errorf("过滤后数量 = %d, want 1", len(filtered))
	}
	// adultMode：全部放行
	if got := len(fs.PrivacyFilter(true)); got != 2 {
		t.Errorf("adultMode 放行数 = %d, want 2", got)
	}
}

// ─── memory_fact: 相关性评分 ─────────────────────────────────

func TestScoreRelevance_ValenceMatchBoosts(t *testing.T) {
	now := time.Now()
	// 事实效价与当前情绪 valence 匹配 → boost；不匹配 → 无 boost
	match := &Fact{MemoryFact: MemoryFact{Weight: 5, Status: "active", CreatedAt: now.AddDate(0, 0, -10), SelfRelevance: 0.8,
		EmotionalContext: &EmotionalContext{Valence: 0.5, Intensity: 0.5}}, Active: true}
	mismatch := &Fact{MemoryFact: MemoryFact{Weight: 5, Status: "active", CreatedAt: now.AddDate(0, 0, -10), SelfRelevance: 0.8,
		EmotionalContext: &EmotionalContext{Valence: 0.9, Intensity: 0.5}}, Active: true}
	matched := ScoreRelevance(match, now, 0.5, 60)
	unmatched := ScoreRelevance(mismatch, now, 0.5, 60)
	if matched <= unmatched {
		t.Errorf("效价匹配应更高: matched=%.2f unmatched=%.2f", matched, unmatched)
	}
}

func TestScoreRelevance_NewerScoresHigher(t *testing.T) {
	now := time.Now()
	old := &Fact{MemoryFact: MemoryFact{Weight: 5, Status: "active", CreatedAt: now.AddDate(0, 0, -100), UpdatedAt: now.AddDate(0, 0, -100), SelfRelevance: 0.8}, Active: true}
	recent := &Fact{MemoryFact: MemoryFact{Weight: 5, Status: "active", CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Minute), SelfRelevance: 0.8}, Active: true}
	if ScoreRelevance(recent, now, 0, 0) <= ScoreRelevance(old, now, 0, 0) {
		t.Error("新事实相关度应更高")
	}
}

func TestComputeMoodBoost(t *testing.T) {
	// 自相关高 + 正面 → 正提升
	boost := ComputeMoodBoost(0.9, 0.8, 50)
	if boost < 0 {
		t.Errorf("正面组合提升应为正: %f", boost)
	}
	// 自相关低 → 无提升
	low := ComputeMoodBoost(0.1, 0.8, 50)
	if low > boost {
		t.Error("低自相关提升应更低")
	}
}

// ─── memory_fact: 核心事实选择 ───────────────────────────────

func TestFactStore_SelectCoreFacts(t *testing.T) {
	fs := NewFactStore()
	for i := 0; i < 5; i++ {
		fs.Add(MemoryFact{Summary: "事实", Weight: 5}) // 高权重 → core tier
	}
	cores := fs.SelectCoreFacts(3)
	if len(cores) > 3 {
		t.Errorf("SelectCoreFacts(3) 返回 %d, want <= 3", len(cores))
	}
	for _, f := range cores {
		if !f.IsCore() {
			t.Errorf("选择结果应全为 core: %+v", f)
		}
	}
}

func TestFactStore_AutoRetire(t *testing.T) {
	fs := NewFactStore()
	// 旧且低权重的应被退休
	old := fs.Add(MemoryFact{Summary: "旧事实", Weight: 0.1})
	old.CreatedAt = time.Now().AddDate(0, 0, -100)
	fs.Add(MemoryFact{Summary: "新事实", Weight: 5})
	retired := fs.AutoRetire()
	if retired == 0 {
		t.Error("AutoRetire 应退休至少 1 个旧事实")
	}
}

func TestFactStore_UpdateFact(t *testing.T) {
	fs := NewFactStore()
	f := fs.Add(MemoryFact{Summary: "原始", Weight: 1})
	fs.UpdateFact(f.ID, map[string]interface{}{"summary": "更新后"})
	got := fs.Get(f.ID)
	if got.Summary != "更新后" {
		t.Errorf("更新后摘要 = %q", got.Summary)
	}
}

// ─── attention_budget ────────────────────────────────────────

func TestAttentionBudget_Default(t *testing.T) {
	am := NewAttentionManager()
	st := am.State()
	if len(st.LastProactiveAt) != 0 {
		t.Errorf("默认应无 proactive 记录, got %d", len(st.LastProactiveAt))
	}
}

func TestAttentionBudget_BudgetExceeded(t *testing.T) {
	am := NewAttentionManager()
	now := time.Now()
	// 未超限
	if am.IsBudgetExceeded(now) {
		t.Error("初始不应超限")
	}
	// 大量 proactive → 超限
	for i := 0; i < 100; i++ {
		am.RecordProactive(now)
	}
	if !am.IsBudgetExceeded(now) {
		t.Error("大量 proactive 后应超限")
	}
}

func TestAttentionBudget_GlobalDnd(t *testing.T) {
	am := NewAttentionManager()
	now := time.Now()
	if am.IsGlobalDndActive(now) {
		t.Error("初始不应 DND")
	}
	// 设置 DND 到未来
	am.SetGlobalDnd(now.Add(2*time.Hour), "测试")
	if !am.IsGlobalDndActive(now) {
		t.Error("设置 DND 后应生效")
	}
	// reason 不匹配 → 不清除
	am.ClearGlobalDnd("其他原因")
	if !am.IsGlobalDndActive(now) {
		t.Error("reason 不匹配不应清除 DND")
	}
	// reason 匹配 → 清除
	am.ClearGlobalDnd("测试")
	if am.IsGlobalDndActive(now) {
		t.Error("清除 DND 后应失效")
	}
	// DND 过期自动失效
	am.SetGlobalDnd(now.Add(-time.Hour), "")
	if am.IsGlobalDndActive(now) {
		t.Error("过期 DND 应失效")
	}
}

// ─── memory_fact: Restore / ListAll（DB 持久化贯通）────────────

func TestFactStore_ListAll(t *testing.T) {
	fs := NewFactStore()
	fs.Add(MemoryFact{Domain: "preference", Subcategory: "FOOD", Subject: "用户", Summary: "喜欢吃辣"})
	f2 := fs.Add(MemoryFact{Domain: "user_state", Subcategory: "MOOD", Subject: "用户", Summary: "最近睡不好"})
	fs.RetireFact(f2.ID)

	all := fs.ListAll()
	if len(all) != 2 {
		t.Fatalf("ListAll 应含退役事实，got %d want 2", len(all))
	}
	active := fs.ListActive()
	if len(active) != 1 {
		t.Fatalf("ListActive 应只剩 1 条，got %d", len(active))
	}
}

func TestFactStore_Restore(t *testing.T) {
	fs := NewFactStore()
	// 模拟从 hermes.db 加载：active + retired 混合
	fs.Restore([]MemoryFact{
		{ID: "f1", Domain: "preference", Subcategory: "FOOD", Subject: "用户", Summary: "喜欢吃辣", Status: "active", Weight: 2},
		{ID: "f2", Domain: "user_state", Subcategory: "MOOD", Subject: "用户", Summary: "最近睡不好", Status: "retired", Weight: 1},
		{ID: "f3", Domain: "user_profile", Subcategory: "BASIC_PROFILE", Subject: "用户", Summary: "程序员", Status: "archived", Weight: 1},
	})

	if got := fs.Count(); got != 1 {
		t.Fatalf("Restore 后 Count（活跃）应 1（仅 active），got %d", got)
	}
	if len(fs.ListActive()) != 1 {
		t.Fatal("仅 status=active 应算活跃，retired/archived 不应算")
	}
	if got := len(fs.ListAll()); got != 3 {
		t.Fatalf("ListAll 应保留全部 3 条（含退役），got %d", got)
	}
	// ID 与状态保留
	f2 := fs.Get("f2")
	if f2 == nil {
		t.Fatal("Restore 后应按原 ID 可查")
	}
	if f2.IsActive() {
		t.Error("retired 事实恢复后不应 active")
	}
	// 空 status 按 active 处理
	fs2 := NewFactStore()
	fs2.Restore([]MemoryFact{{ID: "x", Domain: "d", Subcategory: "s", Subject: "u", Summary: "无状态", Status: ""}})
	if fs2.Count() != 1 {
		t.Error("空 status 事实应视为 active")
	}
}

// ─── memory_graph: 并发安全 / Restore / ListAll ──────────────

func TestKnowledgeGraph_ConcurrentSafe(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Add("用户", "喜欢", "辣", 1, nil)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kg.Add("实体", "关系", "对象", 0.5, nil)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = kg.Query("辣", 5)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = kg.ListAll()
		}()
	}
	wg.Wait()
	if kg.Size() != 21 {
		t.Fatalf("并发 Add 后 Size = %d, want 21", kg.Size())
	}
}

func TestKnowledgeGraph_RestoreAndListAll(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.Restore([]Triple{
		{ID: "t1", Subject: "用户", Predicate: "喜欢", Object: "辣", Confidence: 1},
		{ID: "t2", Subject: "用户", Predicate: "职业", Object: "程序员", Confidence: 0.8},
	})
	if kg.Size() != 2 {
		t.Fatalf("Restore 后 Size = %d, want 2", kg.Size())
	}
	// 倒排索引重建：Query 可命中
	if got := len(kg.Query("辣", 5)); got != 1 {
		t.Fatalf("Restore 后 Query(辣) = %d, want 1", got)
	}
	// ListAll 保留原 ID
	all := kg.ListAll()
	if len(all) != 2 || all[0].ID != "t1" || all[1].ID != "t2" {
		t.Fatalf("ListAll 应保留原 ID: %+v", all)
	}
	// Add 后新三元组在 ListAll 中
	kg.Add("用户", "喜欢", "咖啡", 0.9, nil)
	if got := kg.Size(); got != 3 {
		t.Fatalf("Add 后 Size = %d, want 3", got)
	}
}

// ─── memory_episode: 并发安全 ─────────────────────────────────

func TestEpisodicStore_ConcurrentSafe(t *testing.T) {
	es := NewEpisodicStore()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es.Add(Episode{Summary: "测试情节", DominantEmotion: "平静", EmotionalIntensity: 0.3})
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = es.Search("测试", 3)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = es.Count()
		}()
	}
	wg.Wait()
	if got := es.Count(); got != 20 {
		t.Fatalf("并发 Add 后 Count = %d, want 20", got)
	}
}

// ─── orchestrator: TierB 情节记忆注入 ────────────────────────

func TestOrchestrator_BuildTierBBlockIncludesEpisodes(t *testing.T) {
	orch := NewOrchestrator("sess-t", PersonalityPresets[0])
	orch.EpisodicStore.Add(Episode{
		Summary: "用户说喜欢雨天散步", DominantEmotion: "开心",
		EmotionalIntensity: 0.7, Keywords: []string{"雨"},
	})
	orch.FactStore.Add(MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "喜欢吃辣", Weight: 2,
	})
	block := orch.buildTierBBlock("你喜欢雨天散步吗", 60, 1)
	if !strings.Contains(block, "相关记忆片段") || !strings.Contains(block, "雨天散步") {
		t.Fatalf("buildTierBBlock 应注入情节记忆片段: %s", block)
	}
}

func TestAttentionBudget_ProactiveLimit(t *testing.T) {
	am := NewAttentionManager()
	now := time.Now()
	am.SetProactiveLimit(3)
	am.RecordProactive(now)
	am.RecordProactive(now)
	if am.IsBudgetExceeded(now) {
		t.Error("2 次 < 3 次限制不应超限")
	}
	am.RecordProactive(now)
	if !am.IsBudgetExceeded(now) {
		t.Error("达到限制应超限（>= 语义）")
	}
}
