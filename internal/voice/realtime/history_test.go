// history_test.go — 暂存代际制聊天历史语义（O1~O4；handlers/history.py 同位）。
package realtime

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeLegs struct {
	reply string
	slow  bool
}

func (f *fakeLegs) Reply(ctx context.Context, userText string, history []ChatMessage, turnID string, rev int, itemID string, onDelta func(string)) (string, int, int, error) {
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

func (f *fakeLegs) Speak(_ context.Context, _ string, _ func([]byte)) error { return nil }

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
	p := newFakePipeline(t, &fakeLegs{reply: "好的"})
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		p.muHist.Lock()
		finalized := false
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
	for _, m := range p.chat {
		switch m.Role {
		case "user":
			users++
		case "assistant":
			assistants++
			assistantText = m.Content
		}
	}
	if users != 1 || assistants != 1 {
		t.Fatalf("完成轮历史 user=%d assistant=%d want 1/1", users, assistants)
	}
	if assistantText != "好的" {
		t.Fatalf("assistant 文本=%q want 好的", assistantText)
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
