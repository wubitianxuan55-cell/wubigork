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

	if got, blocked := a.whisperState.takeVoiceV2PreparedTurn(pid, user, "item-b"); got != nil || !blocked {
		t.Fatalf("相同文本不得被另一 item 认领: got=%v blocked=%v", got != nil, blocked)
	}
	got, blocked := a.whisperState.takeVoiceV2PreparedTurn(pid, user, "item-a")
	if blocked || got != first {
		t.Fatalf("相同 item 未取回原快照: got=%v blocked=%v", got != nil, blocked)
	}
}
