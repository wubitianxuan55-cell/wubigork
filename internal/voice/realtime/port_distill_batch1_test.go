// port_distill_batch1_test.go — 批1 蒸馏修刀契约钉（2026-11-07）。
// H1 跨 key/DROP 断流 + H2 被顶替响应 O3 回滚 + M2 audioBatch 属主清理 +
// M3 切史以本轮 user 为界 + M1 会话结束复位回声门。
// 台账：docs/gaea-voice-port-distill-20261107.md。
package realtime

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ─────────── H1：跨 key + DROP 断流 ───────────

func TestStaleResponseDeltasDropped(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "答案"})
	// 同轮 reopen：rev0 → rev1（槽位被新响应接管，旧响应未被取消——补发路径）
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	rsOld := p.newResponseState("turn_1", 0, time.Time{}) // 槽位易主
	p.turns.SegmentFinalized(2000, 800, 0)
	p.turns.SpeechStarted(2500) // 确认重开 → cur=(turn_1, rev1)
	rsNew := p.newResponseState("turn_1", 1, time.Time{})
	if p.respState != rsNew {
		t.Fatal("前置：新响应应持有槽位")
	}

	// ① 跨 key：旧响应 delta 拒绝附加（不标 dead、不发帧、不进 parts）
	p.dispatchText(&textEvent{kind: evKindAssistantDelta, resp: rsOld, transcript: "旧"})
	if rsOld.dead {
		t.Fatal("跨 key 附加拒绝不得标 dead——旧响应终态/O3 回滚仍须照常收口")
	}
	if len(rsOld.parts) != 0 {
		t.Fatalf("旧响应 delta 不得进 parts, got %v", rsOld.parts)
	}
	if got := drainTranscriptDeltas(p); got != "" {
		t.Fatalf("旧响应 delta 不得上线缆, got %q", got)
	}

	// ② 新响应 delta 正常下行
	p.dispatchText(&textEvent{kind: evKindAssistantDelta, resp: rsNew, transcript: "新"})
	if len(rsNew.parts) != 1 || rsNew.parts[0] != "新" {
		t.Fatalf("新响应 delta 应放行, got %v", rsNew.parts)
	}
	if got := drainTranscriptDeltas(p); got != "新" {
		t.Fatalf("新响应 delta 应上线缆, got %q", got)
	}

	// ③ DROP：新轮已开+旧轮已 close（失相关性）后 delta 断流并标 dead
	// （跨 key 守卫前置返回，DROP 需槽位仍在手——已提交轮恒 relevant，
	// 其陈旧流由①守卫拦；Close 不改 current，须再开新轮才失相关）
	p.turns.StartTurn() // cur=turn_2
	rs2 := p.newResponseState("turn_2", 0, time.Time{})
	p.dispatchText(&textEvent{kind: evKindAssistantDelta, resp: rs2, transcript: "a"}) // 提交 turn_2
	drainTranscriptDeltas(p)                                                           // 收掉合法帧（线缆异步）
	p.turns.Close("turn_2", 0)
	p.turns.StartTurn() // cur=turn_3 → (turn_2,0) 失相关性
	p.dispatchText(&textEvent{kind: evKindAssistantDelta, resp: rs2, transcript: "陈旧"})
	if !rs2.dead {
		t.Fatal("DROP 后应标 dead 断流")
	}
	if got := drainTranscriptDeltas(p); got != "" {
		t.Fatalf("DROP 后 delta 不得上线缆, got %q", got)
	}
}

// drainTranscriptDeltas 抽干线缆上累积的 transcript delta 文本。
func drainTranscriptDeltas(p *Pipeline) string {
	var b strings.Builder
loop:
	for {
		select {
		case ev := <-p.session.out:
			if ev["type"] == "response.output_audio_transcript.delta" {
				if s, ok := ev["delta"].(string); ok {
					b.WriteString(s)
				}
			}
		default:
			break loop
		}
	}
	return b.String()
}

// ─────────── H2：被顶替响应的 O3 回滚 ───────────

func TestSupersededResponseRollsBackProvisional(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "答案"})
	p.turns.StartTurn()
	rsOld := p.newResponseState("turn_1", 0, time.Time{})
	p.newResponseState("turn_1", 1, time.Time{}) // 槽位易主
	p.muHist.Lock()
	p.chat = append(p.chat,
		ChatMessage{Role: "user", Content: "问", turnKey: "turn_1"},
		ChatMessage{Role: "assistant", Content: "旧答案尸体", respKey: rsOld.key, provisional: true},
		ChatMessage{Role: "assistant", Content: "已定稿历史", respKey: "other-key", provisional: false},
	)
	p.muHist.Unlock()

	p.finishResponse(p.session, rsOld, "completed", "")

	p.muHist.Lock()
	contents := make([]string, 0, len(p.chat))
	for _, m := range p.chat {
		contents = append(contents, m.Role+":"+m.Content)
	}
	p.muHist.Unlock()
	joined := strings.Join(contents, "|")
	if strings.Contains(joined, "旧答案尸体") {
		t.Fatalf("被顶替响应的暂存必须回滚, chat=%v", contents)
	}
	if !strings.Contains(joined, "assistant:已定稿历史") {
		t.Fatalf("已定稿历史不得误伤, chat=%v", contents)
	}
	if !p.closedKeys[rsOld.key] {
		t.Fatal("被顶替响应应落墓碑（残余哨兵幂等）")
	}
	for ev := range drainAll(p) {
		if strings.Contains(ev, rsOld.id) {
			t.Fatalf("被顶替响应不得发线缆帧, got %s", ev)
		}
	}
}

// drainAll 抽干会话出站帧，返回 type 列表。
func drainAll(p *Pipeline) <-chan string {
	ch := make(chan string, 256)
	go func() {
		defer close(ch)
	loop:
		for {
			select {
			case ev := <-p.session.out:
				if s, ok := ev["type"].(string); ok {
					ch <- s
				}
			default:
				break loop
			}
		}
	}()
	return ch
}

// ─────────── M2：audioBatch 属主清理 ───────────

func TestAudioBatchClearedOnCancelled(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "答案"})
	p.turns.StartTurn()
	rs := p.newResponseState("turn_1", 0, time.Time{})

	// cancelled：残余批按属主清零，不得混入下一响应首批
	p.dispatchAudio(&audioOut{resp: rs, pcm: make([]byte, 100)})
	if p.audioBatchKey != rs.key || len(p.audioBatch) != 100 {
		t.Fatalf("前置：残余批应归属当前响应, key=%q len=%d", p.audioBatchKey, len(p.audioBatch))
	}
	p.finishResponse(p.session, rs, "cancelled", "x")
	if len(p.audioBatch) != 0 || p.audioBatchKey != "" {
		t.Fatalf("cancelled 终态应清残余批, len=%d key=%q", len(p.audioBatch), p.audioBatchKey)
	}

	// completed：尾批 flush 属主不变（既有语义回归）
	rs2 := p.newResponseState("turn_2", 0, time.Time{})
	p.dispatchAudio(&audioOut{resp: rs2, pcm: make([]byte, 100)})
	p.finishResponse(p.session, rs2, "completed", "")
	if len(p.audioBatch) != 0 {
		t.Fatalf("completed 尾批应 flush, len=%d", len(p.audioBatch))
	}
}

// ─────────── M3：切史以本轮 user 为界 ───────────

func TestLastUserIndexOfTurn(t *testing.T) {
	h := []ChatMessage{
		{Role: "user", turnKey: "turn_1"},
		{Role: "assistant"},
		{Role: "user", turnKey: "turn_2"},
		{Role: "assistant", respKey: "old"}, // 同轮 reopen 残留的暂存 assistant
	}
	if got := lastUserIndexOfTurn(h, "turn_2"); got != 2 {
		t.Fatalf("应定位本轮 user@2, got %d", got)
	}
	if got := lastUserIndexOfTurn(h, "turn_9"); got != -1 {
		t.Fatalf("无本轮 user 应 -1, got %d", got)
	}
}

// recordingLegs 捕获历次请求的 history（M3 断言面）。
type recordingLegs struct {
	fakeLegs
	mu        chan struct{}
	histories [][]ChatMessage
}

func (f *recordingLegs) Reply(ctx context.Context, userText string, history []ChatMessage, turnID string, rev int, itemID string, onDelta func(string)) (string, int, int, error) {
	snap := make([]ChatMessage, len(history))
	copy(snap, history)
	f.histories = append(f.histories, snap)
	return f.fakeLegs.Reply(ctx, userText, history, turnID, rev, itemID, onDelta)
}

func TestPromptHistoryCutsAtCurrentTurnUser(t *testing.T) {
	legs := &recordingLegs{fakeLegs: fakeLegs{reply: "好"}}
	p := newFakePipeline(t, legs)

	// 常规轮：history 应切到本轮 user 之前（user 由 userText 携带）
	p.turns.StartTurn()
	p.turns.SegmentFinalized(1000, 800, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}
	waitClosedKeys(p, 3*time.Second)
	// 快照=[user(turn_1)] → 切后=[]
	if len(legs.histories) == 0 {
		t.Fatal("LLM 未被调用")
	}
	if n := len(legs.histories[0]); n != 0 {
		t.Fatalf("常规轮 history 应为空（user 由 userText 携带）, got %d 条", n)
	}

	// M3 主场景直驱（提交后不可重开，真实残留=旧响应未提交时的暂存）：
	// 快照=[user(turn_1), assistant(旧暂存)] → 切后=[]（旧暂存剥离+user 由 userText 携带）
	rs2 := p.newResponseState("turn_1", 0, time.Time{})
	p.promptCh <- &promptRequest{resp: rs2, userText: "第一问补充", history: []ChatMessage{
		{Role: "user", Content: "第一问", turnKey: "turn_1"},
		{Role: "assistant", Content: "旧暂存", respKey: "old", provisional: true},
	}}
	waitHistories(legs, 2, 3*time.Second)
	if len(legs.histories) < 2 {
		t.Fatal("第二轮 LLM 未被调用")
	}
	last := legs.histories[len(legs.histories)-1]
	if len(last) != 0 {
		t.Fatalf("history 应切至本轮 user 之前, got %v", last)
	}
}

func waitHistories(legs *recordingLegs, n int, d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if len(legs.histories) >= n {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func waitClosedKeys(p *Pipeline, d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		p.muTerm.Lock()
		n := len(p.closedKeys)
		p.muTerm.Unlock()
		if n > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// ─────────── M1：会话结束复位回声门 ───────────

func TestSessionEndResetsEchoGuard(t *testing.T) {
	h := NewVADHandler(DefaultVADConfig(), nil, nil, NewSpeculativeTurns())
	h.SetThreshold(0.62)
	h.EchoGuardOn()
	if h.iter.threshold != echoGuardThreshold {
		t.Fatalf("前置：开门应 %v", echoGuardThreshold)
	}
	h.OnSessionEnd()
	if h.echoGuard {
		t.Fatal("会话结束应复位 echoGuard——否则下会话开门早退空转")
	}
	if h.baseThreshold != DefaultVADConfig().Threshold {
		t.Fatalf("baseThreshold 应回落配置值, got %v", h.baseThreshold)
	}
	if h.iter.threshold != DefaultVADConfig().Threshold {
		t.Fatalf("threshold 应回落配置值, got %v", h.iter.threshold)
	}
	// 下会话回声门可正常开启（0.5→0.75）
	h.EchoGuardOn()
	if !h.echoGuard || h.iter.threshold != echoGuardThreshold {
		t.Fatalf("下会话开门应生效, guard=%v th=%v", h.echoGuard, h.iter.threshold)
	}
}
