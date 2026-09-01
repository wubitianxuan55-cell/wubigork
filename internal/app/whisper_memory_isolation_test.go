package app

import (
	"testing"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// 构造 A/B 两个角色的隔离记忆数据（facts/episodes/triples 均带归属会话）
// sessionA/sessionB 为完整会话 ID（如 whisper_personaA），测试需各自唯一避免进程级会话缓存串扰
func seedIsolatedMemory(t *testing.T, root, sessionA, sessionB string) {
	t.Helper()
	now := time.Now()
	_ = repos.InsertFact(root, whisper.MemoryFact{
		ID: "fA", Domain: "user_behavior", Subcategory: "TEST", Subject: "林晚",
		Summary: "A 的专属记忆", Weight: 1, Confidence: 0.9, Status: "active",
		SourceSessionID: sessionA, SourceTurnIndex: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = repos.InsertEpisode(root, whisper.Episode{
		ID: "epA", Summary: "A 的专属情节", EmotionalIntensity: 0.5, DominantEmotion: "joy",
		SourceSessionID: sessionA, StartTurn: 1, EndTurn: 2, CreatedAt: now,
	})
	_ = repos.InsertTriple(root, whisper.Triple{
		ID: "tA", Subject: "林晚", Predicate: "喜欢", Object: "剑",
		Confidence: 0.8, SourceFactIDs: []string{"fA"}, CreatedAt: now,
	})

	_ = repos.InsertFact(root, whisper.MemoryFact{
		ID: "fB", Domain: "user_behavior", Subcategory: "TEST", Subject: "顾长风",
		Summary: "B 的专属记忆", Weight: 1, Confidence: 0.9, Status: "active",
		SourceSessionID: sessionB, SourceTurnIndex: 1, CreatedAt: now, UpdatedAt: now,
	})
	_ = repos.InsertEpisode(root, whisper.Episode{
		ID: "epB", Summary: "B 的专属情节", EmotionalIntensity: 0.5, DominantEmotion: "sad",
		SourceSessionID: sessionB, StartTurn: 1, EndTurn: 2, CreatedAt: now,
	})
	_ = repos.InsertTriple(root, whisper.Triple{
		ID: "tB", Subject: "顾长风", Predicate: "守护", Object: "城",
		Confidence: 0.8, SourceFactIDs: []string{"fB"}, CreatedAt: now,
	})
}

// TestWhisperMemoryRestore_IsolatedBySession 角色记忆隔离：恢复时每个角色只拿到自己的记忆
func TestWhisperMemoryRestore_IsolatedBySession(t *testing.T) {
	a := newChatServiceTestApp(t)
	// 会话 ID 每次运行唯一：whisperSessions 是进程级全局缓存，固定 ID 在
	// `-count` 多次运行下会命中上次运行的 orch（跨 app 实例串扰）。
	isoA := testKind("isoA")
	isoB := testKind("isoB")
	seedIsolatedMemory(t, a.whisperDataRoot, "whisper_"+isoA, "whisper_"+isoB)

	orchA := a.getOrCreateOrch(isoA)
	if orchA.FactStore.Get("fA") == nil {
		t.Fatalf("角色 A 应恢复自己的事实 fA")
	}
	if orchA.FactStore.Get("fB") != nil {
		t.Fatalf("角色 A 恢复了 B 的事实（隔离失败）")
	}
	epsA := orchA.EpisodicStore.ListAll()
	if len(epsA) != 1 || epsA[0].ID != "epA" {
		t.Fatalf("角色 A 情节应为仅 epA: %+v", epsA)
	}
	trisA := orchA.KG.ListAll()
	if len(trisA) != 1 || trisA[0].ID != "tA" {
		t.Fatalf("角色 A 图谱应为仅 tA: %+v", trisA)
	}

	orchB := a.getOrCreateOrch(isoB)
	if orchB.FactStore.Get("fA") != nil || orchB.FactStore.Get("fB") == nil {
		t.Fatalf("角色 B 记忆串扰：A=%v B=%v", orchB.FactStore.Get("fA"), orchB.FactStore.Get("fB"))
	}
	epsB := orchB.EpisodicStore.ListAll()
	if len(epsB) != 1 || epsB[0].ID != "epB" {
		t.Fatalf("角色 B 情节应为仅 epB: %+v", epsB)
	}
	trisB := orchB.KG.ListAll()
	if len(trisB) != 1 || trisB[0].ID != "tB" {
		t.Fatalf("角色 B 图谱应为仅 tB: %+v", trisB)
	}
}

// TestWhisperPersist_PreservesOtherSessions 持久化隔离：A 写回时不能覆盖 B 的记忆
func TestWhisperPersist_PreservesOtherSessions(t *testing.T) {
	a := newChatServiceTestApp(t)
	// 会话 ID 每次运行唯一：whisperSessions 是进程级全局缓存，固定 ID 在
	// `-count` 多次运行下会命中上次运行的 orch（跨 app 实例串扰）。
	persA := testKind("persA")
	persB := testKind("persB")
	seedIsolatedMemory(t, a.whisperDataRoot, "whisper_"+persA, "whisper_"+persB)

	orchA := a.getOrCreateOrch(persA)
	// A 新增自己的记忆
	orchA.FactStore.Add(whisper.MemoryFact{
		ID: "fA2", Domain: "user_behavior", Subcategory: "TEST", Subject: "新话题",
		Summary: "A 新增记忆", Weight: 1, Confidence: 0.9, Status: "active",
		SourceSessionID: "whisper_" + persA,
	})
	orchA.EpisodicStore.Add(whisper.Episode{
		ID: "epA2", Summary: "A 新增情节", SourceSessionID: "whisper_" + persA, CreatedAt: time.Now(),
	})
	newTriple := orchA.KG.Add("新话题", "关于", "A", 0.8, []string{"fA2"})

	persistWhisperState(orchA)

	// DB 层校验：B 的旧记忆必须还在，A 的新记忆已写入
	dbFacts := repos.LoadFactsFromDB(a.whisperDataRoot)
	factIDs := make(map[string]bool, len(dbFacts))
	for _, f := range dbFacts {
		factIDs[f.ID] = true
	}
	if !factIDs["fB"] || !factIDs["fA2"] {
		t.Fatalf("事实写回覆盖了其他角色: %v", factIDs)
	}
	dbEps, _ := repos.LoadEpisodesFromDB(a.whisperDataRoot)
	epIDs := make(map[string]bool, len(dbEps))
	for _, e := range dbEps {
		epIDs[e.ID] = true
	}
	if !epIDs["epB"] || !epIDs["epA2"] {
		t.Fatalf("情节写回覆盖了其他角色: %v", epIDs)
	}
	dbTris, _ := repos.LoadTriplesFromDB(a.whisperDataRoot)
	trIDs := make(map[string]bool, len(dbTris))
	for _, tr := range dbTris {
		trIDs[tr.ID] = true
	}
	if !trIDs["tB"] || !trIDs[newTriple.ID] {
		t.Fatalf("图谱写回覆盖了其他角色: %v", trIDs)
	}
}

// TestWhisperTraces_IsolatedBySession 轮次追踪按会话隔离（角色库「追踪」页数据源）
func TestWhisperTraces_IsolatedBySession(t *testing.T) {
	a := newChatServiceTestApp(t)
	// 会话 ID 每次运行唯一（同 Restore/Persist 隔离测试的进程级缓存理由）。
	isoA := testKind("isoA")
	isoB := testKind("isoB")
	now := time.Now()
	trA := whisper.TurnTrace{Turn: 1, Timestamp: &now}
	trB := whisper.TurnTrace{Turn: 2, Timestamp: &now}
	if err := repos.AppendTurnTraceToDB(a.whisperDataRoot, "whisper_"+isoA, trA); err != nil {
		t.Fatalf("写入 A 追踪: %v", err)
	}
	if err := repos.AppendTurnTraceToDB(a.whisperDataRoot, "whisper_"+isoB, trB); err != nil {
		t.Fatalf("写入 B 追踪: %v", err)
	}
	tracesA := a.WhisperGetTraces(isoA)
	if len(tracesA) != 1 || tracesA[0].Turn != 1 {
		t.Fatalf("A 追踪应为仅 1 条: %+v", tracesA)
	}
	tracesB := a.WhisperGetTraces(isoB)
	if len(tracesB) != 1 || tracesB[0].Turn != 2 {
		t.Fatalf("B 追踪应为仅 1 条: %+v", tracesB)
	}
}
