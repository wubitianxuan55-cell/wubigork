package app

import (
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// ─── T7-1.1 ③ 跨会话持久化并发安全 + 末轮落库 ──────────────────

// TestPersistConcurrent_NoOverwrite 两会话并发持久化：全局单写锁（persistMu）
// 串行化「全表读→内存合并→全表替换」，两会话的事实都必须落库（互不覆盖，H3）。
func TestPersistConcurrent_NoOverwrite(t *testing.T) {
	dataRoot := t.TempDir()
	if _, err := db.GetDatabase(dataRoot); err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	defer db.CloseDatabase(dataRoot)

	orch1 := whisper.NewOrchestrator("sess-a", whisper.PersonalityPresets[0])
	orch1.DataRoot = dataRoot
	orch1.FactStore.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "A喜欢吃辣", Weight: 2, Confidence: 0.9, SourceSessionID: "sess-a",
	})

	orch2 := whisper.NewOrchestrator("sess-b", whisper.PersonalityPresets[0])
	orch2.DataRoot = dataRoot
	orch2.FactStore.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "DRINK", Subject: "用户",
		Summary: "B喜欢喝咖啡", Weight: 1, Confidence: 0.9, SourceSessionID: "sess-b",
	})

	a := &whisperState{core: &core{}}
	const rounds = 8
	var wg sync.WaitGroup
	for i := 0; i < rounds; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _ = a.persistStateSync(orch1) }()
		go func() { defer wg.Done(); _ = a.persistStateSync(orch2) }()
	}
	wg.Wait()

	if got := len(repos.LoadFactsFromDB(dataRoot)); got != 2 {
		t.Fatalf("DB 事实 = %d, want 2（并发互相覆盖）", got)
	}
}

// TestPersistConcurrent_SameSessionSerialized 同一会话并发持久化：
// persistStateSync 先取状态快照（回合锁内）再持全局锁落库，无死锁且最终一致。
func TestPersistConcurrent_SameSessionSerialized(t *testing.T) {
	dataRoot := t.TempDir()
	if _, err := db.GetDatabase(dataRoot); err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	defer db.CloseDatabase(dataRoot)

	orch := whisper.NewOrchestrator("sess-s", whisper.PersonalityPresets[0])
	orch.DataRoot = dataRoot
	orch.FactStore.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "S喜欢吃辣", Weight: 2, Confidence: 0.9, SourceSessionID: "sess-s",
	})

	a := &whisperState{core: &core{}}
	const rounds = 8
	var wg sync.WaitGroup
	for i := 0; i < rounds; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = a.persistStateSync(orch) }()
	}
	wg.Wait()

	if got := len(repos.LoadFactsFromDB(dataRoot)); got != 1 {
		t.Fatalf("DB 事实 = %d, want 1", got)
	}
}

// TestDrainAndPersistAll_FinalRoundLands 末轮落库（H4）：异步记忆写入队列先 drain
// （末轮 LLM 抽取的事实进入内存 FactStore），再持久化全部会话，事实最终落库。
func TestDrainAndPersistAll_FinalRoundLands(t *testing.T) {
	dataRoot := t.TempDir()
	if _, err := db.GetDatabase(dataRoot); err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	defer db.CloseDatabase(dataRoot)

	orch := whisper.NewOrchestrator("whisper_final", whisper.PersonalityPresets[0])
	orch.DataRoot = dataRoot

	whisperSessionsMu.Lock()
	whisperSessions["whisper_final"] = orch
	whisperSessionsMu.Unlock()
	defer func() {
		whisperSessionsMu.Lock()
		delete(whisperSessions, "whisper_final")
		whisperSessionsMu.Unlock()
		whisper.ResetMemoryWriteQueues()
	}()

	a := &whisperState{core: &core{}}

	// 入队一个会抽取 1 条事实的异步记忆写入（末轮）
	whisper.EnqueueMemoryWrite(writeObsLlmStub{reply: `{"facts":[{"domain":"preference","subcategory":"FOOD","subject":"用户","summary":"喜欢吃辣","weight":0.8,"confidence":0.9,"selfRelevance":0.8}]}`}, whisper.MemoryWritePayload{
		SessionID: orch.SessionID, TurnIndex: 1,
		UserMsg: "我喜欢吃辣", AssistantText: "记住了",
		FactStore: orch.FactStore, TotalTurns: 1,
	}, a.recordMemoryWriteError)

	// Shutdown 末轮：先 drain 再 persist
	a.drainAndPersistAll()

	if got := len(repos.LoadFactsFromDB(dataRoot)); got != 1 {
		t.Fatalf("末轮落库后 DB 事实 = %d, want 1", got)
	}
}

// TestPersistStateSync_ConcurrentWithTurn 持久化与回合并发：persistStateSync 在
// 回合锁内取快照（不与主流程竞争 orch.State），落库用快照，最终状态一致（①）。
func TestPersistStateSync_ConcurrentWithTurn(t *testing.T) {
	dataRoot := t.TempDir()
	if _, err := db.GetDatabase(dataRoot); err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	defer db.CloseDatabase(dataRoot)

	orch := whisper.NewOrchestrator("sess-race", whisper.PersonalityPresets[0])
	orch.DataRoot = dataRoot
	a := &whisperState{core: &core{}}

	const turns = 20
	var wg sync.WaitGroup
	for i := 0; i < turns; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = a.persistStateSync(orch) }()
		orch.LockTurn()
		_ = orch.PreLLMTurn("你好呀，今天天气不错")
		orch.UnlockTurn()
	}
	wg.Wait()
	// 最终 settle：确保落库的是最后状态
	if err := a.persistStateSync(orch); err != nil {
		t.Fatalf("final persist: %v", err)
	}
	state, err := repos.LoadCompanionStateFromDB(dataRoot, "sess-race")
	if err != nil {
		t.Fatalf("LoadCompanionStateFromDB: %v", err)
	}
	if state == nil {
		t.Fatal("落库状态为空")
	}
	if state.Counters.TotalTurns != turns {
		t.Errorf("落库 TotalTurns = %d, want %d", state.Counters.TotalTurns, turns)
	}
}
