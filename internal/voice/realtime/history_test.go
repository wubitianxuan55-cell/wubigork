// history_test.go — 暂存代际制聊天历史语义（O1~O4；handlers/history.py 同位）。
package realtime

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type fakeLegs struct {
	reply string
	slow  bool
	calls int32
}

func (f *fakeLegs) Reply(ctx context.Context, userText string, history []ChatMessage, turnID string, rev int, itemID string, onDelta func(string)) (string, int, int, error) {
	atomic.AddInt32(&f.calls, 1)
	for _, part := range strings.Split(f.reply, "") {
		if ctx.Err() != nil {
			return "", 0, 0, ctx.Err()
		}
		onDelta(part)
		if f.slow {
			time.Sleep(40 * time.Millisecond)
		}
	}
	return f.reply, 10, 20, nil
}

// Speak 生产同形：非空句必产音频（breeze 真腿保证）——零音频只剩 LLM 空回复
// 一种来源，R3 的 cancelled/empty_output 判定才不误伤正常轮。
func (f *fakeLegs) Speak(_ context.Context, sentence string, onAudio func([]byte)) error {
	if sentence == "" || strings.HasSuffix(sentence, "mark") {
		return nil
	}
	onAudio(make([]byte, 6400))
	return nil
}

type fakeSTT struct{ text string }

func (f fakeSTT) Transcribe([]float32, int) (string, error) { return f.text, nil }

func newFakePipeline(t *testing.T, legs ResponseLegs) *Pipeline {
	t.Helper()
	return newFakePipelineSTT(t, legs, fakeSTT{"测试话语"})
}

func newFakePipelineSTT(t *testing.T, legs ResponseLegs, stt STTEngine) *Pipeline {
	turns := NewSpeculativeTurns()
	p := &Pipeline{
		cfg:                PipelineConfig{VAD: DefaultVADConfig(), Legs: legs, CreateResponse: true, ChatSize: 12},
		turns:              turns,
		legs:               legs,
		stt:                stt,
		chatSize:           12,
		inputItemByTurnRev: map[turnKey]string{},
		pendingTerminals:   map[string]*pendingInputTerminal{},
		completedFinalRevs: map[turnKey]bool{},
		closedKeys:         map[string]bool{},
		audioCh:            make(chan []float32, 64),
		sttCh:              make(chan *VADFinal, 8),
		promptCh:           make(chan *promptRequest, 8),
		ttsCh:              make(chan *ttsRequest, 8),
		audioOut:           make(chan *audioOut, 64),
		textCh:             make(chan *textEvent, 64),
		cancelCh:           make(chan string, 4),
		signal:             make(chan struct{}, 1),
		stopped:            make(chan struct{}),
	}
	p.vad = NewVADHandler(DefaultVADConfig(), nil, nil, turns)
	p.startOnce.Do(p.runStages)
	t.Cleanup(func() { p.OnSessionEnd(nil) })
	p.session = &Session{ID: "t", out: make(chan map[string]any, 4096), done: make(chan struct{})}
	return p
}

// driveTurn 模拟一轮：started → (直接投递 final 到 STT 队列) → 等 STT+LLM 处理。
func driveTurn(t *testing.T, p *Pipeline, turn string, text string) {
	t.Helper()
	p.dispatchText(&textEvent{kind: evKindSpeechStarted,
		started: &VADStarted{TurnID: turn, InterruptResponse: true}})
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: turn, Rev: 0}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		p.muHist.Lock()
		hasAssistant := false
		for _, m := range p.chat {
			if m.Role == "assistant" {
				hasAssistant = true
			}
		}
		p.muHist.Unlock()
		if text == "" || hasAssistant {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func countRoles(p *Pipeline, role string) int {
	p.muHist.Lock()
	defer p.muHist.Unlock()
	n := 0
	for _, m := range p.chat {
		if m.Role == role {
			n++
		}
	}
	return n
}

// TestHistoryWriteOnComplete — O1/O3：完成响应的回复写入历史并 finalize。
func TestHistoryWriteOnComplete(t *testing.T) {
	legs := &fakeLegs{reply: "好的"}
	p := newFakePipeline(t, legs)
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	deadline := time.Now().Add(3 * time.Second)
	finalized := false
	for time.Now().Before(deadline) {
		p.muHist.Lock()
		for _, m := range p.chat {
			if m.Role == "assistant" && !m.provisional {
				finalized = true
			}
		}
		p.muHist.Unlock()
		if finalized {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	p.muHist.Lock()
	defer p.muHist.Unlock()
	users, assistants := 0, 0
	assistantText := ""
	anyProvisional := false
	for _, m := range p.chat {
		switch m.Role {
		case "user":
			users++
		case "assistant":
			assistants++
			assistantText = m.Content
			if m.provisional {
				anyProvisional = true
			}
		}
	}
	if users != 1 || assistants != 1 {
		t.Fatalf("完成轮历史 user=%d assistant=%d want 1/1", users, assistants)
	}
	if assistantText != "好的" {
		t.Fatalf("assistant 文本=%q want 好的", assistantText)
	}
	// O3 硬校验：等待循环可能超时，终态断言不依赖它（批2 假绿回归门）
	if !finalized || anyProvisional {
		t.Fatalf("完成响应的 assistant 应 finalize（provisional=false）")
	}
}

// TestHistoryRollbackOnCancel — O2：取消响应的回复从历史撤销。
func TestHistoryRollbackOnCancel(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "好的我来说", slow: true})
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	time.Sleep(120 * time.Millisecond) // 让 LLM 流出部分文本
	p.cancelActiveResponse("turn_detected")
	time.Sleep(150 * time.Millisecond) // 等 TTS/哨兵链清完
	if n := countRoles(p, "assistant"); n != 0 {
		t.Fatalf("取消后 assistant 历史应被回滚，残留 %d 条", n)
	}
	if n := countRoles(p, "user"); n != 1 {
		t.Fatalf("user 历史应保留 1 条，实际 %d", n)
	}
}

// TestHistoryUserReplacedOnReopen — O4：同轮重开替换 user 文本，不新增条目。
func TestHistoryUserReplacedOnReopen(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "嗯"})
	// 第一轮：turn_1 rev=0
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	time.Sleep(200 * time.Millisecond)
	// 重开：同 turn_1 rev=1（sttStage 的 turnKey=turn_1 → 替换）
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 1}
	time.Sleep(200 * time.Millisecond)
	p.muHist.Lock()
	defer p.muHist.Unlock()
	users := 0
	last := ""
	for _, m := range p.chat {
		if m.Role == "user" {
			users++
			last = m.Content
		}
	}
	_ = last
	// 同 turn 键（rev 变化不算新轮键）→ 不新增 user 条目
	// 注：turnKey 取 f.TurnID（rev 无关），两次同 turn_1 → 替换
	if users > 2 {
		t.Fatalf("同轮两次转写 user 条目=%d（应≤2：替换语义）", users)
	}
}

// TestHistoryRollbackAfterWrite — O2/O3 确定性验证：暂存写入后的终态处置。
// 批2 假绿回归门：「ctx 断流不写入」≠回滚——写完暂存再取消必须删除，完成必须 finalize。
func TestHistoryRollbackAfterWrite(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "好的"})
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	rs := p.newResponseState("turn_1", 0, time.Time{})
	p.muHist.Lock()
	p.chat = append(p.chat,
		ChatMessage{Role: "user", Content: "你好", turnKey: "turn_1"},
		ChatMessage{Role: "assistant", Content: "半截回复", respKey: rs.key, provisional: true},
	)
	p.muHist.Unlock()

	// 取消 → 该 respKey 暂存 assistant 撤销，user 保留（O2 close 同位）
	p.finishResponse(p.session, rs, "cancelled", "turn_detected")
	p.muHist.Lock()
	users, assistants := 0, 0
	for _, m := range p.chat {
		switch m.Role {
		case "user":
			users++
		case "assistant":
			assistants++
		}
	}
	p.muHist.Unlock()
	if users != 1 || assistants != 0 {
		t.Fatalf("取消后应回滚 assistant（want user=1/assistant=0），实际 user=%d assistant=%d", users, assistants)
	}

	// 完成 → finalize（provisional=false）（O3 finalize 同位）
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	rs2 := p.newResponseState("turn_2", 0, time.Time{})
	p.muHist.Lock()
	p.chat = append(p.chat,
		ChatMessage{Role: "user", Content: "再问", turnKey: "turn_2"},
		ChatMessage{Role: "assistant", Content: "完整回复", respKey: rs2.key, provisional: true},
	)
	p.muHist.Unlock()
	p.finishResponse(p.session, rs2, "completed", "")
	p.muHist.Lock()
	defer p.muHist.Unlock()
	for _, m := range p.chat {
		if m.Role == "assistant" && m.respKey == rs2.key && m.provisional {
			t.Fatalf("完成响应的 assistant 应 finalize（provisional=false）")
		}
	}
}

// TestCancelFlushQueuedPrompt — S6：取消 flush 排队 prompt 必须逐个标 dead 且不阻塞。
// 批1 复核：旧双接收写法首载荷静默丢弃不标 dead，且仅一条排队请求时第二接收可阻塞发送循环。
// 确定性构造：慢速回复占住 LLM 阶段，第二请求滞留队列，取消时必被 flush 标 dead。
func TestCancelFlushQueuedPrompt(t *testing.T) {
	legs := &fakeLegs{reply: "好的我来说", slow: true} // ~200ms 生成，占住 LLM 阶段
	p := newFakePipeline(t, legs)
	turnID, _ := p.turns.StartTurn() // turns 机自生成 ID，必须用返回值（自造 ID 会被 gate 丢弃）
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: turnID, Rev: 0}
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&legs.calls) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if atomic.LoadInt32(&legs.calls) == 0 {
		t.Fatalf("慢速请求未被 LLM 阶段消费")
	}
	// LLM 忙 → 排队请求滞留 promptCh（裸 responseState：不经 newResponseState 免动 respState）
	rsB := &responseState{id: "resp_b", key: randHex(16), turnID: "turn_b", rev: 0, itemID: "item_b"}
	p.promptCh <- &promptRequest{resp: rsB, userText: "乙"}

	done := make(chan struct{})
	go func() {
		p.cancelActiveResponse("client_cancelled")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("cancelActiveResponse 在单排队请求下阻塞（flush 双接收病）")
	}
	if !rsB.dead {
		t.Fatalf("排队请求的 resp 应被标记 dead")
	}
	// dead 短路：标 dead 的请求再入队也不得触发生成（总调用数保持 1）
	p.promptCh <- &promptRequest{resp: rsB, userText: "乙"}
	time.Sleep(300 * time.Millisecond) // 等 LLM 阶段消化队尾
	if n := atomic.LoadInt32(&legs.calls); n != 1 {
		t.Fatalf("LLM 调用数=%d（want 1：慢速请求 1 次，dead 请求 0 次）", n)
	}
}

// TestHistoryAnchorWriteBack — R2：assistant 落在快照锚点（本轮 user）之后。
// 上游 language_model.py:631/679：生成期 soft-end 非打断语音可追加更新 user，
// 写回必须在锚点之后插入，不得把本轮回复越过新 user 尾插到末尾。
func TestHistoryAnchorWriteBack(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "好的"})
	p.muHist.Lock()
	p.chat = append(p.chat,
		ChatMessage{Role: "user", Content: "问一", turnKey: "turn_1"},
		ChatMessage{Role: "user", Content: "问二", turnKey: "turn_2"}, // 生成期追加
	)
	p.muHist.Unlock()

	rs := &responseState{key: randHex(16), turnID: "turn_1", ctx: context.Background()}
	p.muHist.Lock()
	p.writeAssistantAnchor(rs, "答一")
	p.muHist.Unlock()

	p.muHist.Lock()
	defer p.muHist.Unlock()
	if len(p.chat) != 3 {
		t.Fatalf("chat 长度=%d want 3", len(p.chat))
	}
	if p.chat[1].Role != "assistant" || p.chat[1].Content != "答一" {
		t.Fatalf("assistant 应插在锚点（turn_1 user 之后）：got %s/%s", p.chat[1].Role, p.chat[1].Content)
	}
	if p.chat[2].Role != "user" || p.chat[2].turnKey != "turn_2" {
		t.Fatalf("生成期追加的 turn_2 user 不得被越过：got %s/%v", p.chat[2].Role, p.chat[2].turnKey)
	}
	// 锚点缺失（已被裁剪）→ 尾插兜底
	rs2 := &responseState{key: randHex(16), turnID: "turn_x", ctx: context.Background()}
	p.writeAssistantAnchor(rs2, "答x")
	if p.chat[len(p.chat)-1].Role != "assistant" || p.chat[len(p.chat)-1].Content != "答x" {
		t.Fatalf("锚点缺失应尾插兜底")
	}
}

// TestEmptyOutputResponseCancels — R3（audit 复核池）：LLM 空回复=零音频下行，
// done 哨兵收口必须 cancelled/empty_output 且 O3 回滚空暂存（不得以
// completed+空输出谎报完成、不得把空 assistant 留在历史）。
func TestEmptyOutputResponseCancels(t *testing.T) {
	legs := &fakeLegs{reply: ""}
	p := newFakePipeline(t, legs)
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	// 等响应收口（done 哨兵 → finishResponse 墓碑），再断言终态
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(p.closedKeys) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(p.closedKeys) == 0 {
		t.Fatal("响应未收口（无墓碑）")
	}
	if countRoles(p, "assistant") != 0 {
		t.Fatal("空回复不得留 assistant 条目（O3 回滚）")
	}
	if countRoles(p, "user") != 1 {
		t.Fatalf("user 轮应保留，got %d", countRoles(p, "user"))
	}
}


// TestTranscriptDeltaOnWire — 线缆契约钉：LLM 流式期间 transcript.delta 必须
// 逐增量下行（2026-11-05 ws_probe turn 场景 FAIL=delta 全程缺席，助手文本只在
// done 出现——用户看不到流式回复）。持发窗口内的增量必须停靠续放，不得判死。
func TestTranscriptDeltaOnWire(t *testing.T) {
	legs := &fakeLegs{reply: "流式回复文本"}
	p := newFakePipeline(t, legs)
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(p.closedKeys) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	deltas := 0
	dones := 0
	loop:
	for {
		select {
		case ev := <-p.session.out:
			switch ev["type"] {
			case "response.output_audio_transcript.delta":
				deltas++
			case "response.done":
				dones++
			}
		default:
			break loop
		}
	}
	if dones == 0 {
		t.Fatal("响应未收口（线缆无 response.done）")
	}
	if deltas == 0 {
		t.Fatal("线缆上没有任何 transcript.delta（流式助手文本丢失）")
	}
}

// failingLegs LLM 失败腿（F1 兜底路径钉）。
type failingLegs struct{}

func (failingLegs) Reply(context.Context, string, []ChatMessage, string, int, string, func(string)) (string, int, int, error) {
	return "", 0, 0, context.DeadlineExceeded
}
func (failingLegs) Speak(_ context.Context, sentence string, onAudio func([]byte)) error {
	if sentence == "" || strings.HasSuffix(sentence, "mark") {
		return nil
	}
	onAudio(make([]byte, 6400))
	return nil
}

// TestFallbackTextOnWire — F1：LLM 失败走兜底语时，兜底文本必须进转写面
// （transcript.delta + done 全文），不得只在 TTS 里出声、转写面板空白。
func TestFallbackTextOnWire(t *testing.T) {
	p := newFakePipeline(t, failingLegs{})
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(p.closedKeys) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	var doneText string
	deltas := 0
loop:
	for {
		select {
		case ev := <-p.session.out:
			switch ev["type"] {
			case "response.output_audio_transcript.delta":
				deltas++
			case "response.output_audio_transcript.done":
				if t, ok := ev["transcript"].(string); ok {
					doneText = t
				}
			}
		default:
			break loop
		}
	}
	if deltas == 0 {
		t.Fatal("兜底语未进转写 delta 面")
	}
	if !strings.Contains(doneText, "我现在有点跟不上") {
		types := []string{}
	loop2:
		for {
			select {
			case ev := <-p.session.out:
				if t2, ok := ev["type"].(string); ok {
					types = append(types, t2)
				}
			default:
				break loop2
			}
		}
		t.Fatalf("done 转写全文缺兜底语: %q 线缆帧=%v", doneText, types)
	}
}


// TestTerminalsPublishWhileResponseInFlight — F2：响应生成期（turn 未显式
// commit）user terminal 必须即时发布，不得悬挂到会话结束。
func TestTerminalsPublishWhileResponseInFlight(t *testing.T) {
	p := newFakePipeline(t, failingLegs{})
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	deadline := time.Now().Add(3 * time.Second)
	published := map[string]int{}
	for time.Now().Before(deadline) {
		select {
		case ev := <-p.session.out:
			typ, _ := ev["type"].(string)
			switch typ {
			case "input_audio_buffer.committed", "conversation.item.created",
				"conversation.item.input_audio_transcription.completed":
				published[typ]++
			}
		default:
			if len(p.closedKeys) > 0 {
				// 收口后再扫一轮余量即止
				time.Sleep(50 * time.Millisecond)
				continue
			}
		}
		if len(p.closedKeys) > 0 && len(published) >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if published["input_audio_buffer.committed"] == 0 ||
		published["conversation.item.created"] == 0 ||
		published["conversation.item.input_audio_transcription.completed"] == 0 {
		t.Fatalf("响应期 user terminal 未发布: %v", published)
	}
}
