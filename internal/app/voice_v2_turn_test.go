package app

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/whisper"
)

func TestVoiceV2CommitTurnPersistsOnceAndSchedulesPersonaMemory(t *testing.T) {
	a := newChatServiceTestApp(t)
	pid := "voice_commit_test"
	sessionID := "whisper_" + pid
	whisperSessionsMu.Lock()
	delete(whisperSessions, sessionID)
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		whisperSessionsMu.Lock()
		delete(whisperSessions, sessionID)
		whisperSessionsMu.Unlock()
	})

	oldEnqueue, oldPersist := voiceV2EnqueueMemory, voiceV2PersistState
	var mu sync.Mutex
	var payloads []whisper.MemoryWritePayload
	var persists int
	voiceV2EnqueueMemory = func(_ *whisperState, payload whisper.MemoryWritePayload) {
		mu.Lock()
		payloads = append(payloads, payload)
		mu.Unlock()
	}
	voiceV2PersistState = func(_ *whisperState, _ *whisper.Orchestrator) {
		mu.Lock()
		persists++
		mu.Unlock()
	}
	t.Cleanup(func() {
		voiceV2EnqueueMemory, voiceV2PersistState = oldEnqueue, oldPersist
	})

	topic, err := a.ChatTopicCreate("语音记忆", pid)
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	result, err := a.whisperState.VoiceV2CommitTurn(
		topic.ID, "我最近开始晨跑，通常六点起床", "记住啦，你喜欢清晨的空气。", "resp_v2_1", "",
	)
	if err != nil {
		t.Fatalf("VoiceV2CommitTurn: %v", err)
	}
	if result["memoryScheduled"] != true || result["totalTurns"] != 1 {
		t.Fatalf("人格收尾结果不符: %#v", result)
	}

	msgs, err := a.chatStore.ListMessages(topic.ID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 2 || msgs[0].Content != "我最近开始晨跑，通常六点起床" {
		t.Fatalf("聊天事务不完整: %#v", msgs)
	}
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(msgs[1].Extra), &meta); err != nil {
		t.Fatalf("assistant extra JSON: %v (%q)", err, msgs[1].Extra)
	}
	if meta["source"] != "voice_v2" || meta["voice_v2_response_id"] != "resp_v2_1" {
		t.Fatalf("response_id 未持久化: %#v", meta)
	}

	mu.Lock()
	gotPayload := len(payloads)
	gotPersists := persists
	var captured whisper.MemoryWritePayload
	if gotPayload > 0 {
		captured = payloads[0]
	}
	mu.Unlock()
	if gotPayload != 1 || gotPersists != 1 {
		t.Fatalf("memory enqueue=%d persist=%d, want 1/1", gotPayload, gotPersists)
	}
	if captured.TurnIndex != 1 || !strings.Contains(captured.UserMsg, "晨跑") ||
		!strings.Contains(captured.AssistantText, "清晨") {
		t.Fatalf("记忆载荷不完整: %#v", captured)
	}

	dup, err := a.whisperState.VoiceV2CommitTurn(
		topic.ID, "我最近开始晨跑，通常六点起床", "记住啦，你喜欢清晨的空气。", "resp_v2_1", "",
	)
	if err != nil {
		t.Fatalf("duplicate commit: %v", err)
	}
	if dup["duplicate"] != true || dup["memoryScheduled"] != false {
		t.Fatalf("重复回执未判重: %#v", dup)
	}
	msgs, _ = a.chatStore.ListMessages(topic.ID)
	mu.Lock()
	gotPayload = len(payloads)
	gotPersists = persists
	mu.Unlock()
	if len(msgs) != 2 || gotPayload != 1 || gotPersists != 1 {
		t.Fatalf("重复提交产生副作用: messages=%d enqueue=%d persist=%d", len(msgs), gotPayload, gotPersists)
	}
}

func TestVoiceV2CommitTurnPlainSkipsPersonaMemory(t *testing.T) {
	a := newChatServiceTestApp(t)
	oldEnqueue := voiceV2EnqueueMemory
	calls := 0
	voiceV2EnqueueMemory = func(_ *whisperState, _ whisper.MemoryWritePayload) { calls++ }
	t.Cleanup(func() { voiceV2EnqueueMemory = oldEnqueue })

	topic, err := a.ChatTopicCreate("普通语音", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	result, err := a.whisperState.VoiceV2CommitTurn(
		topic.ID, "今天几号", "今天是十一月一日。", "resp_plain_1", "",
	)
	if err != nil {
		t.Fatalf("VoiceV2CommitTurn(plain): %v", err)
	}
	if result["memoryScheduled"] != false || calls != 0 {
		t.Fatalf("plain 不应写人格记忆: result=%#v calls=%d", result, calls)
	}
	msgs, err := a.chatStore.ListMessages(topic.ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("plain 聊天未落库: n=%d err=%v", len(msgs), err)
	}
}

func TestVoiceV2CommitTurnRejectsInvalidInputAndUnknownTopic(t *testing.T) {
	a := newChatServiceTestApp(t)
	if _, err := a.whisperState.VoiceV2CommitTurn("", "u", "a", "r", ""); err == nil {
		t.Fatal("空 topic 应拒绝")
	}
	if _, err := a.whisperState.VoiceV2CommitTurn("t", "u", "a", "", ""); err == nil {
		t.Fatal("空 responseID 应拒绝")
	}
	if _, err := a.whisperState.VoiceV2CommitTurn("missing", "u", "a", "r", ""); err == nil {
		t.Fatal("不存在的话题应拒绝")
	}
}

func TestVoiceV2PreparedTurnRollsBackAndCommitReusesSamePreLLM(t *testing.T) {
	a := newChatServiceTestApp(t)
	pid := "voice_prepared_test"
	sessionID := "whisper_" + pid
	whisperSessionsMu.Lock()
	delete(whisperSessions, sessionID)
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		a.whisperState.rollbackVoiceV2Prepared()
		whisperSessionsMu.Lock()
		delete(whisperSessions, sessionID)
		whisperSessionsMu.Unlock()
	})

	oldEnqueue, oldPersist := voiceV2EnqueueMemory, voiceV2PersistState
	var mu sync.Mutex
	enqueueCount := 0
	voiceV2EnqueueMemory = func(_ *whisperState, _ whisper.MemoryWritePayload) {
		mu.Lock()
		enqueueCount++
		mu.Unlock()
	}
	voiceV2PersistState = func(_ *whisperState, _ *whisper.Orchestrator) {}
	t.Cleanup(func() {
		voiceV2EnqueueMemory, voiceV2PersistState = oldEnqueue, oldPersist
	})

	orch := a.whisperState.getOrCreateOrch(pid)
	user := "我今晚有点难过，能陪我聊会吗"
	prepared := a.whisperState.prepareVoiceV2Turn(pid, user, "item-1", "turn-1", 0)
	if prepared == nil {
		t.Fatal("生成前预推进为空")
	}
	if got := orch.State.Counters.TotalTurns; got != 1 {
		t.Fatalf("预推进轮次 = %d, want 1", got)
	}
	if strings.TrimSpace(prepared.pre.SystemPrompt) == "" || strings.TrimSpace(prepared.pre.DynamicContext) == "" {
		t.Fatalf("预推进上下文不完整: system=%d dynamic=%d",
			len(prepared.pre.SystemPrompt), len(prepared.pre.DynamicContext))
	}
	if !a.whisperState.rollbackVoiceV2PreparedTurn(prepared) {
		t.Fatal("预推进回合未回滚")
	}
	if got := orch.State.Counters.TotalTurns; got != 0 {
		t.Fatalf("回滚后轮次 = %d, want 0", got)
	}

	if got := a.whisperState.prepareVoiceV2Turn(pid, user, "item-2", "turn-2", 0); got == nil {
		t.Fatal("二次预推进为空")
	}
	topic, err := a.ChatTopicCreate("预推进提交", pid)
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	result, err := a.whisperState.VoiceV2CommitTurn(
		topic.ID, user, "我在，慢慢说。", "resp_prepared_1", "item-2",
	)
	if err != nil {
		t.Fatalf("VoiceV2CommitTurn prepared: %v", err)
	}
	if result["totalTurns"] != 1 {
		t.Fatalf("提交不应再次 PreLLM 推进到 2: %#v", result)
	}
	if got := orch.State.Counters.TotalTurns; got != 1 {
		t.Fatalf("最终轮次 = %d, want 1", got)
	}
	mu.Lock()
	gotEnqueue := enqueueCount
	mu.Unlock()
	if gotEnqueue != 1 {
		t.Fatalf("预推进提交记忆任务 = %d, want 1", gotEnqueue)
	}
}

func TestVoiceV2PreparedTurnUsesItemIDWhenTextRepeats(t *testing.T) {
	a := newChatServiceTestApp(t)
	pid := "voice_prepared_item_test"
	sessionID := "whisper_" + pid
	whisperSessionsMu.Lock()
	delete(whisperSessions, sessionID)
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		a.whisperState.rollbackVoiceV2Prepared()
		whisperSessionsMu.Lock()
		delete(whisperSessions, sessionID)
		whisperSessionsMu.Unlock()
	})

	user := "再说一遍"
	first := a.whisperState.prepareVoiceV2Turn(pid, user, "item-a", "turn-a", 0)
	if first == nil {
		t.Fatal("第一轮预推进为空")
	}

	if got, blocked, mismatch := a.whisperState.takeVoiceV2PreparedTurn(pid, user, "item-b"); got != nil || !blocked || mismatch {
		t.Fatalf("相同文本不得被另一 item 认领: got=%v blocked=%v mismatch=%v", got != nil, blocked, mismatch)
	}
	got, blocked, _ := a.whisperState.takeVoiceV2PreparedTurn(pid, user, "item-a")
	if blocked || got != first {
		t.Fatalf("相同 item 未取回原快照: got=%v blocked=%v", got != nil, blocked)
	}
}

// P0a-A2（众神殿定稿）：item 命中但人格错配——同锁回滚旧人格预推进、清槽、
// 返回可区分 mismatch；不得走新人格 PreLLMTurn 路径。
func TestVoiceV2PreparedTurnTakeRejectsCrossPersonality(t *testing.T) {
	a := newChatServiceTestApp(t)
	pidA, pidB := "voice_mm_a", "voice_mm_b"
	whisperSessionsMu.Lock()
	delete(whisperSessions, "whisper_"+pidA)
	delete(whisperSessions, "whisper_"+pidB)
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		a.whisperState.rollbackVoiceV2Prepared()
		whisperSessionsMu.Lock()
		delete(whisperSessions, "whisper_"+pidA)
		delete(whisperSessions, "whisper_"+pidB)
		whisperSessionsMu.Unlock()
	})

	orchA := a.whisperState.getOrCreateOrch(pidA)
	user := "这句要按谁的人格收尾"
	if got := a.whisperState.prepareVoiceV2Turn(pidA, user, "item-x", "turn-x", 0); got == nil {
		t.Fatal("人格 A 预推进为空")
	}
	if got := orchA.State.Counters.TotalTurns; got != 1 {
		t.Fatalf("预推进后 A 轮次 = %d, want 1", got)
	}

	prepared, blocked, mismatch := a.whisperState.takeVoiceV2PreparedTurn(pidB, user, "item-x")
	if prepared != nil || blocked || !mismatch {
		t.Fatalf("跨人格认领应返回可区分 mismatch: prepared=%v blocked=%v mismatch=%v",
			prepared != nil, blocked, mismatch)
	}
	if len(a.whisperState.voiceV2PreparedEntries()) > 0 {
		t.Fatal("错配后 prepared 槽未清空")
	}
	if got := orchA.State.Counters.TotalTurns; got != 0 {
		t.Fatalf("错配回滚后 A 轮次 = %d, want 0（幽灵预推进）", got)
	}
	// 错配已清槽：正常路径不被残留快照污染
	if p2, blocked2, mm2 := a.whisperState.takeVoiceV2PreparedTurn(pidB, user, "item-x"); p2 != nil || blocked2 || mm2 {
		t.Fatalf("错配后重复认领应空手而回: %v %v %v", p2 != nil, blocked2, mm2)
	}
}

// P0a-A2 集成面：生成按人格 A 预推进、提交按 topic.Mode=B → 聊天照存、
// 人格记忆不写、A 的预推进回滚、B 状态零污染。
func TestVoiceV2CommitMismatchKeepsChatAndSkipsPersonaMemory(t *testing.T) {
	a := newChatServiceTestApp(t)
	pidA, pidB := "voice_mm_commit_a", "voice_mm_commit_b"
	whisperSessionsMu.Lock()
	delete(whisperSessions, "whisper_"+pidA)
	delete(whisperSessions, "whisper_"+pidB)
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		a.whisperState.rollbackVoiceV2Prepared()
		whisperSessionsMu.Lock()
		delete(whisperSessions, "whisper_"+pidA)
		delete(whisperSessions, "whisper_"+pidB)
		whisperSessionsMu.Unlock()
	})

	oldEnqueue, oldPersist := voiceV2EnqueueMemory, voiceV2PersistState
	var mu sync.Mutex
	enqueueCount := 0
	voiceV2EnqueueMemory = func(_ *whisperState, _ whisper.MemoryWritePayload) {
		mu.Lock()
		enqueueCount++
		mu.Unlock()
	}
	voiceV2PersistState = func(_ *whisperState, _ *whisper.Orchestrator) {}
	t.Cleanup(func() {
		voiceV2EnqueueMemory, voiceV2PersistState = oldEnqueue, oldPersist
	})

	orchA := a.whisperState.getOrCreateOrch(pidA)
	user := "跨人格提交集成"
	if got := a.whisperState.prepareVoiceV2Turn(pidA, user, "item-m", "turn-m", 0); got == nil {
		t.Fatal("人格 A 预推进为空")
	}
	topic, err := a.ChatTopicCreate("错配提交", pidB)
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	result, err := a.whisperState.VoiceV2CommitTurn(
		topic.ID, user, "聊天先落库，记忆不写。", "resp_mm_1", "item-m",
	)
	if err != nil {
		t.Fatalf("VoiceV2CommitTurn: %v", err)
	}
	warning, _ := result["warning"].(string)
	if result["memoryScheduled"] != false || !strings.Contains(warning, "人格错配") {
		t.Fatalf("错配提交应拒记忆并留可读告警: %#v", result)
	}
	msgs, err := a.chatStore.ListMessages(topic.ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("聊天照存不符: n=%d err=%v", len(msgs), err)
	}
	if got := orchA.State.Counters.TotalTurns; got != 0 {
		t.Fatalf("A 预推进未回滚: TotalTurns=%d", got)
	}
	orchB := a.whisperState.getOrCreateOrch(pidB)
	if got := orchB.State.Counters.TotalTurns; got != 0 {
		t.Fatalf("B 不得被错配提交推进: TotalTurns=%d", got)
	}
	mu.Lock()
	got := enqueueCount
	mu.Unlock()
	if got != 0 {
		t.Fatalf("错配提交不得进人格记忆队列: enqueue=%d", got)
	}
}

// P0a-A5g（众神殿定稿）：plain 早退在 take 之前，本轮 prepared 必须同锁回收，
// 否则幽灵预推进污染同一人格的下一条文本回合。
func TestVoiceV2CommitPlainDropsPreparedTurn(t *testing.T) {
	a := newChatServiceTestApp(t)
	pid := "voice_plain_drop"
	whisperSessionsMu.Lock()
	delete(whisperSessions, "whisper_"+pid)
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		a.whisperState.rollbackVoiceV2Prepared()
		whisperSessionsMu.Lock()
		delete(whisperSessions, "whisper_"+pid)
		whisperSessionsMu.Unlock()
	})

	orch := a.whisperState.getOrCreateOrch(pid)
	user := "普通话题也带预推进"
	if got := a.whisperState.prepareVoiceV2Turn(pid, user, "item-p", "turn-p", 0); got == nil {
		t.Fatal("预推进为空")
	}
	if got := orch.State.Counters.TotalTurns; got != 1 {
		t.Fatalf("预推进后轮次 = %d, want 1", got)
	}
	topic, err := a.ChatTopicCreate("普通语音丢弃", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	result, err := a.whisperState.VoiceV2CommitTurn(
		topic.ID, user, "plain 只落聊天。", "resp_plain_drop", "item-p",
	)
	if err != nil {
		t.Fatalf("VoiceV2CommitTurn(plain): %v", err)
	}
	if result["memoryScheduled"] != false {
		t.Fatalf("plain 不写记忆: %#v", result)
	}
	if len(a.whisperState.voiceV2PreparedEntries()) > 0 {
		t.Fatal("plain 提交后 prepared 残留（A5g）")
	}
	if got := orch.State.Counters.TotalTurns; got != 0 {
		t.Fatalf("预推进未随 plain 提交回收: TotalTurns=%d", got)
	}
	msgs, err := a.chatStore.ListMessages(topic.ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("plain 聊天未落库: n=%d err=%v", len(msgs), err)
	}
}

// P0a-A5f 后端面：生成中关闭的精确回滚；提交与 abort 竞争只允许一方成功。
func TestVoiceV2AbortTurnRollsBackPrepared(t *testing.T) {
	a := newChatServiceTestApp(t)
	pid := "voice_abort_test"
	whisperSessionsMu.Lock()
	delete(whisperSessions, "whisper_"+pid)
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		a.whisperState.rollbackVoiceV2Prepared()
		whisperSessionsMu.Lock()
		delete(whisperSessions, "whisper_"+pid)
		whisperSessionsMu.Unlock()
	})

	orch := a.whisperState.getOrCreateOrch(pid)
	user := "生成中关闭这一轮"
	if got := a.whisperState.prepareVoiceV2Turn(pid, user, "item-ab", "turn-ab", 0); got == nil {
		t.Fatal("预推进为空")
	}
	if got := orch.State.Counters.TotalTurns; got != 1 {
		t.Fatalf("预推进后轮次 = %d, want 1", got)
	}

	// item 不匹配：不误杀他人的 prepared
	if r := a.whisperState.VoiceV2AbortTurn("item-other"); r["aborted"] != false || r["reason"] != "item-mismatch" {
		t.Fatalf("异 item abort 应拒绝: %#v", r)
	}
	if len(a.whisperState.voiceV2PreparedEntries()) == 0 {
		t.Fatal("拒绝路径不得清槽")
	}
	// item 命中：回滚并清槽
	r := a.whisperState.VoiceV2AbortTurn("item-ab")
	if r["aborted"] != true || r["personalityID"] != pid {
		t.Fatalf("abort 回执不符: %#v", r)
	}
	if len(a.whisperState.voiceV2PreparedEntries()) > 0 {
		t.Fatal("abort 后 prepared 残留")
	}
	if got := orch.State.Counters.TotalTurns; got != 0 {
		t.Fatalf("abort 后轮次 = %d, want 0", got)
	}
	// 空手 abort：诚实回报
	if r := a.whisperState.VoiceV2AbortTurn(""); r["aborted"] != false || r["reason"] != "no-prepared" {
		t.Fatalf("无 prepared abort 应 no-prepared: %#v", r)
	}
}
