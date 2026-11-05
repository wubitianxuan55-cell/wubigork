package app

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/voice/realtime"
	"github.com/gaea/gaea/internal/whisper"
)

// voice_v2_go_legs_test.go — 进程内 LLM 腿测试（刀6 网关退役：原 8792 网关测试的
// 语义继承者。prepare/装配/流式/回滚三角在腿层验证，无 HTTP 跳数）。

func TestVoiceNativeLLMLegPreparesAndStreams(t *testing.T) {
	oldStream := voiceV2ChatLLMStream
	t.Cleanup(func() { voiceV2ChatLLMStream = oldStream })

	var gotMsgs []ai.ChatMessage
	voiceV2ChatLLMStream = func(_ context.Context, _ *mediaState, msgs []ai.ChatMessage, _ []ai.ChatToolSchema) (<-chan ai.SSEChunk, error) {
		gotMsgs = msgs
		ch := make(chan ai.SSEChunk, 4)
		ch <- ai.SSEChunk{Reasoning: "思考过程"} // reasoning delta 不转发——语音不朗读思考
		ch <- ai.SSEChunk{Content: "你好"}
		ch <- ai.SSEChunk{Content: "呀"}
		ch <- ai.SSEChunk{Done: true}
		close(ch)
		return ch, nil
	}

	a := newChatServiceTestApp(t)
	pid := "voice_leg_stream"
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
	a.mediaState = &mediaState{core: a.core, app: a}
	a.mediaState.setVoicePersonality(pid)

	var deltas []string
	full, inT, outT, err := voiceNativeLLMLeg{a: a.mediaState}.Reply(
		context.Background(), "今天想散散步", nil, "turn_1", 0, "item_1", func(d string) { deltas = append(deltas, d) },
	)
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if full != "你好呀" || len(deltas) != 2 || deltas[0] != "你好" || deltas[1] != "呀" {
		t.Fatalf("流式输出不符: full=%q deltas=%v", full, deltas)
	}
	if inT != 0 || outT != 0 {
		t.Fatalf("无 usage 块时应为零: inT=%d outT=%d", inT, outT)
	}
	if strings.Contains(full, "思考过程") {
		t.Fatalf("reasoning delta 不得进入全文: %q", full)
	}
	// 消息装配：真 fixture 的 prepare 产出稳定段（+可能动态段）；末条必为本轮 user，
	// 首条必为稳定设定 system（精确形状由 assembleMessages 专测覆盖）
	if len(gotMsgs) < 2 {
		t.Fatalf("消息装配过短: %+v", gotMsgs)
	}
	if gotMsgs[0].Role != "system" || !strings.Contains(gotMsgs[0].Content, "人格与关系稳定设定") {
		t.Fatalf("首条应为稳定设定 system: %+v", gotMsgs[0])
	}
	last := gotMsgs[len(gotMsgs)-1]
	if last.Role != "user" || last.Content != "今天想散散步" {
		t.Fatalf("末条应为本轮 user 原文: %+v", last)
	}
	// prepared 台账按协议 itemID 绑定，成功路径不回滚（提交由 take 消费）
	a.whisperState.voiceV2PrepareMu.Lock()
	prepared := a.whisperState.voiceV2PreparedLedger["item_1"]
	a.whisperState.voiceV2PrepareMu.Unlock()
	if prepared == nil {
		t.Fatal("prepared 台账应含 item_1（成功路径不回滚）")
	}
	if prepared.itemID != "item_1" || prepared.turnID != "turn_1" || prepared.turnRevision != 0 {
		t.Fatalf("协议回合元数据未绑定: %#v", prepared)
	}
}

func TestVoiceNativeLLMLegDynamicContextBeforeLastUser(t *testing.T) {
	// P1 缓存命中刀契约（applyVoiceV2PreparedPrompt 同位）：稳定段 system 在前；
	// 动态段独立 user 消息插在末条 user 之前；user 本体逐字节不动——
	// 跨轮公共前缀=[system+全部历史] 才能被引擎前缀缓存复用。
	leg := voiceNativeLLMLeg{a: &mediaState{core: &core{}}}
	msgs := leg.assembleMessages(&voiceV2PreparedTurn{pre: whisper.PreLLMResult{
		SystemPrompt:   "PERSONA_STABLE",
		DynamicContext: "EMOTION_CURRENT",
	}}, "今天有点累", []realtime.ChatMessage{
		{Role: "user", Content: "第一问"},
		{Role: "assistant", Content: "第一答"},
	})
	if len(msgs) != 5 {
		t.Fatalf("动态段应独立成消息（system+2 历史+动态+user 共 5 条），got %d: %+v", len(msgs), msgs)
	}
	if msgs[0].Role != "system" || !strings.Contains(msgs[0].Content, "PERSONA_STABLE") {
		t.Fatalf("稳定人格段应在首条 system: %+v", msgs[0])
	}
	if msgs[1].Content != "第一问" || msgs[2].Content != "第一答" {
		t.Fatalf("历史原序保持: %+v", msgs)
	}
	// 角色用 user：herdsman 35B 模板拒绝会话中 system（chat 模型接入实测），
	// user 角色任何模板都收且前缀缓存性质不变。
	dyn := msgs[3]
	if dyn.Role != "user" || !strings.Contains(dyn.Content, "EMOTION_CURRENT") {
		t.Fatalf("动态上下文应独立 user 消息且插在末条 user 前: %+v", dyn)
	}
	if msgs[4].Role != "user" || msgs[4].Content != "今天有点累" {
		t.Fatalf("user 消息本体不得被改写: %+v", msgs[4])
	}
}

func TestVoiceNativeLLMLegPlainMemoryFallback(t *testing.T) {
	// plain 会话（prepared=nil）：无记忆无人格时不注入 system，历史原序+本轮 user。
	leg := voiceNativeLLMLeg{a: &mediaState{core: &core{}}}
	msgs := leg.assembleMessages(nil, "你好", []realtime.ChatMessage{{Role: "user", Content: "第一问"}})
	if len(msgs) != 2 {
		t.Fatalf("应为 2 条，got %d: %+v", len(msgs), msgs)
	}
	if msgs[0].Content != "第一问" || msgs[1].Content != "你好" {
		t.Fatalf("消息装配不符: %+v", msgs)
	}
}

func TestVoiceNativeLLMLegStreamErrorRollsBack(t *testing.T) {
	oldStream := voiceV2ChatLLMStream
	t.Cleanup(func() { voiceV2ChatLLMStream = oldStream })
	voiceV2ChatLLMStream = func(_ context.Context, _ *mediaState, _ []ai.ChatMessage, _ []ai.ChatToolSchema) (<-chan ai.SSEChunk, error) {
		return nil, fmt.Errorf("模型不可用")
	}

	a := newChatServiceTestApp(t)
	pid := "voice_leg_rollback"
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
	a.mediaState = &mediaState{core: a.core, app: a}
	a.mediaState.setVoicePersonality(pid)

	_, _, _, err := voiceNativeLLMLeg{a: a.mediaState}.Reply(
		context.Background(), "我今天有点累", nil, "turn_1", 0, "item_r1", func(string) {},
	)
	if err == nil {
		t.Fatal("流式失败应返回错误")
	}
	orch := a.whisperState.getOrCreateOrch(pid)
	if got := orch.State.Counters.TotalTurns; got != 0 {
		t.Fatalf("失败后预推进未回滚: totalTurns=%d", got)
	}
	a.whisperState.voiceV2PrepareMu.Lock()
	pending := len(a.whisperState.voiceV2PreparedLedger) > 0
	a.whisperState.voiceV2PrepareMu.Unlock()
	if pending {
		t.Fatal("失败后仍残留 prepared turn")
	}
}

func TestVoiceNativeLLMLegCancelRollsBack(t *testing.T) {
	// barge-in/取消：ctx 断流=预推进无人消费，回滚防幽灵（网关客户端断开同位）。
	oldStream := voiceV2ChatLLMStream
	t.Cleanup(func() { voiceV2ChatLLMStream = oldStream })
	voiceV2ChatLLMStream = func(ctx context.Context, _ *mediaState, _ []ai.ChatMessage, _ []ai.ChatToolSchema) (<-chan ai.SSEChunk, error) {
		ch := make(chan ai.SSEChunk, 2)
		ch <- ai.SSEChunk{Content: "半截"}
		go func() {
			<-ctx.Done()
			ch <- ai.SSEChunk{Done: true}
			close(ch)
		}()
		return ch, nil
	}

	a := newChatServiceTestApp(t)
	pid := "voice_leg_cancel"
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
	a.mediaState = &mediaState{core: a.core, app: a}
	a.mediaState.setVoicePersonality(pid)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, _, err := voiceNativeLLMLeg{a: a.mediaState}.Reply(
		ctx, "打断测试", nil, "turn_1", 0, "item_c1", func(string) {},
	)
	if err == nil {
		t.Fatal("取消应返回错误")
	}
	a.whisperState.voiceV2PrepareMu.Lock()
	pending := len(a.whisperState.voiceV2PreparedLedger) > 0
	a.whisperState.voiceV2PrepareMu.Unlock()
	if pending {
		t.Fatal("取消后仍残留 prepared turn")
	}
}

func TestVoiceV2LongTermContextUsesActivePersonalityMemory(t *testing.T) {
	fs := whisper.NewFactStore()
	fs.Add(whisper.MemoryFact{
		Summary:       "用户希望语音回答保持简短",
		Weight:        1.8,
		Confidence:    0.95,
		SelfRelevance: 1,
		Status:        "active",
	})
	orch := whisper.NewOrchestrator("whisper_voice_gateway_test", whisper.PersonalityPresets[0])
	orch.FactStore = fs
	sessionID := "whisper_voice_gateway_test"
	whisperSessionsMu.Lock()
	old, hadOld := whisperSessions[sessionID]
	whisperSessions[sessionID] = orch
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		whisperSessionsMu.Lock()
		if hadOld {
			whisperSessions[sessionID] = old
		} else {
			delete(whisperSessions, sessionID)
		}
		whisperSessionsMu.Unlock()
	})

	a := &mediaState{
		core: &core{},
		app:  &App{whisperState: &whisperState{core: &core{}}},
	}
	a.setVoicePersonality("voice_gateway_test")
	memory := a.voiceV2LongTermContext("今天想听点轻松的")
	if !strings.Contains(memory, "Gaea 长期记忆") || !strings.Contains(memory, "保持简短") {
		t.Fatalf("未注入活跃人格长期记忆: %q", memory)
	}
	a.setVoicePersonality("plain")
	if got := a.voiceV2LongTermContext("你好"); got != "" {
		t.Fatalf("plain 模式不应注入人格记忆: %q", got)
	}
}
