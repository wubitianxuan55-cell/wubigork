// Package whisper — T7-1.1 并发安全测试（会话状态机锁 + 无锁 map + 节奏隔离）
package whisper

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// ─── WorkingMemory 并发安全 ──────────────────────────────────

// TestWorkingMemory_ConcurrentAccess 并发 Push 与 GetRecent/GetAll/BuildContextBlock
// 读写同一 map：无锁时并发 map 读写会 panic；加锁后应无 panic 且计数确定。
// 数据竞争由 -race 门禁兜底捕获。
func TestWorkingMemory_ConcurrentAccess(t *testing.T) {
	wm := NewWorkingMemory()
	const sessions = 8
	// 低于 WorkingMemoryMaxExchanges*2（=12）触发上限截断，保证计数确定
	const perSession = 10
	var wg sync.WaitGroup
	for s := 0; s < sessions; s++ {
		wg.Add(1)
		go func(s int) {
			defer wg.Done()
			for i := 0; i < perSession; i++ {
				wm.Push(fmt.Sprintf("s%d", s), Exchange{TurnIndex: i, UserText: "u", AssistantText: "a"})
			}
		}(s)
		wg.Add(1)
		go func(s int) {
			defer wg.Done()
			for i := 0; i < perSession; i++ {
				_ = wm.GetRecent(fmt.Sprintf("s%d", s))
				_ = wm.GetAll(fmt.Sprintf("s%d", s))
				_ = wm.BuildContextBlock(fmt.Sprintf("s%d", s))
			}
		}(s)
	}
	wg.Wait()
	for s := 0; s < sessions; s++ {
		if got := len(wm.GetAll(fmt.Sprintf("s%d", s))); got != perSession {
			t.Errorf("session s%d: %d exchanges, want %d", s, got, perSession)
		}
	}
}

// TestWorkingMemory_ConcurrentClearNoPanic 并发 Push/GetAll/Clear 无 panic
// （Clear 删除 map 条目与读写竞争，无锁会 panic；确定性断言由上一用例覆盖）。
func TestWorkingMemory_ConcurrentClearNoPanic(t *testing.T) {
	wm := NewWorkingMemory()
	var wg sync.WaitGroup
	for i := 0; i < 60; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			wm.Push(fmt.Sprintf("s%d", i%4), Exchange{TurnIndex: i, UserText: "u", AssistantText: "a"})
		}(i)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = wm.GetAll(fmt.Sprintf("s%d", i%4))
		}(i)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			wm.Clear(fmt.Sprintf("s%d", i%4))
		}(i)
	}
	wg.Wait()
}

// ─── AssociationIndex 并发安全 ───────────────────────────────

// TestAssociationIndex_ConcurrentAccess 并发 Add 与各类读写：Count 确定，无 panic。
func TestAssociationIndex_ConcurrentAccess(t *testing.T) {
	ai := NewAssociationIndex()
	const n = 200
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ai.Add(Association{FactIDA: fmt.Sprintf("fa%d", i), FactIDB: fmt.Sprintf("fb%d", i), AssociationType: "related", Strength: 0.5})
			ai.StrengthenOrCreate(fmt.Sprintf("fa%d", i), fmt.Sprintf("fb%d", i), "related", 0.3)
			ai.WeakenByFactID(fmt.Sprintf("fa%d", i), 0.9)
			ai.RecordActivation(fmt.Sprintf("fa%d", i))
			ai.ClearActivated()
			_ = ai.GetByID(fmt.Sprintf("fa%d", i))
			_ = ai.GetLastActivated()
			ai.Weaken(fmt.Sprintf("fa%d", i), 0.9)
		}(i)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = ai.GetAssociations(fmt.Sprintf("fa%d", i))
			_ = ai.ListAll()
			_ = ai.Count()
		}(i)
	}
	wg.Wait()
	if got := ai.Count(); got != n {
		t.Errorf("Count = %d, want %d", got, n)
	}
}

// ─── HabitsStore 并发安全 ────────────────────────────────────

// TestHabitsStore_ConcurrentAccess 并发 Upsert 与 MatchHabits/All/Count。
func TestHabitsStore_ConcurrentAccess(t *testing.T) {
	hs := NewHabitsStore()
	const n = 150
	now := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			hs.Upsert(UserHabit{Type: "routine", Scope: "short_term", HourStart: i, HourEnd: i + 1, Confidence: 0.5})
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = hs.MatchHabits(now)
			_ = hs.All()
			_ = hs.Count()
		}()
	}
	wg.Wait()
	if got := hs.Count(); got != n {
		t.Errorf("Count = %d, want %d", got, n)
	}
}

// ─── ActiveRecall 并发安全 ───────────────────────────────────

// TestActiveRecall_ConcurrentAccess 并发 MarkRecalled 与 GetHistory/SelectRecallCandidate。
func TestActiveRecall_ConcurrentAccess(t *testing.T) {
	ar := NewActiveRecall()
	fs := NewFactStore()
	fs.Add(MemoryFact{Domain: "core", Subcategory: "X", Subject: "用户", Summary: "核心记忆", Weight: 10, Confidence: 0.9})
	const n = 100
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ar.MarkRecalled(fmt.Sprintf("f%d", i), i)
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ar.GetHistory()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := 0.0
			_ = ar.SelectRecallCandidate(fs, 10, &r)
		}()
	}
	wg.Wait()
	if got := len(ar.GetHistory()); got != n {
		t.Errorf("history length = %d, want %d", got, n)
	}
}

// ─── Orchestrator 回合串行化（①）─────────────────────────────

// TestOrchestrator_ConcurrentTurnsSerialized 并发回合经 LockTurn/UnlockTurn 串行化：
// 无锁时 TotalTurns 会撕裂（多 goroutine 读同一旧值互相覆盖）；加锁后应精确等于 N。
func TestOrchestrator_ConcurrentTurnsSerialized(t *testing.T) {
	o := NewOrchestrator("conc-turns", PersonalityPresets[0])
	const n = 50
	msgs := []string{"你好呀", "今天天气不错", "谢谢你陪我", "有点难过", "晚安"}
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			o.LockTurn()
			_ = o.PreLLMTurn(msgs[i%len(msgs)])
			o.UnlockTurn()
		}(i)
	}
	wg.Wait()
	if got := o.State.Counters.TotalTurns; got != n {
		t.Errorf("TotalTurns = %d, want %d（并发回合撕裂）", got, n)
	}
}

// ─── 节奏计数器隔离（④）──────────────────────────────────────

// TestRhythmCounters_IsolatedAcrossInstances 驱动实例 A 的计数器，
// 实例 B 必须不受影响（跨会话串台修复）。
func TestRhythmCounters_IsolatedAcrossInstances(t *testing.T) {
	a := &RhythmCounters{}
	b := &RhythmCounters{}
	chatter := RhythmInput{Aro: 20, Aff: 30, Stage: StageFamiliar, PersonalityID: "genki", Sincerity: 0.5, Intensity: 0.7}
	for i := 0; i < 4; i++ {
		DecideRhythm(chatter, a)
	}
	if a.Chatter == 0 && a.Monologue == 0 {
		t.Error("驱动 A 应改变其计数器")
	}
	if b.Chatter != 0 || b.Monologue != 0 {
		t.Errorf("B 计数器应保持零（串台）：chatter=%d monologue=%d", b.Chatter, b.Monologue)
	}
}

// TestRhythmCounters_ResetClearsOwnOnly Reset 只清自己的计数器，不碰他人。
func TestRhythmCounters_ResetClearsOwnOnly(t *testing.T) {
	a := &RhythmCounters{Chatter: 2, Monologue: 1}
	b := &RhythmCounters{Chatter: 1, Monologue: 2}
	a.Reset()
	if a.Chatter != 0 || a.Monologue != 0 {
		t.Errorf("Reset 应清零 A：chatter=%d monologue=%d", a.Chatter, a.Monologue)
	}
	if b.Chatter != 1 || b.Monologue != 2 {
		t.Errorf("Reset 不得影响 B：chatter=%d monologue=%d", b.Chatter, b.Monologue)
	}
}
