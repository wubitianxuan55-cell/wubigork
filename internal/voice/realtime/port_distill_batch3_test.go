// port_distill_batch3_test.go — 批3 蒸馏修刀契约钉（2026-11-07）。
// M12 关闭清理序（槽位晚于 OnSessionEnd 释放 + 晚到帧墓碑保留）+ M13 会话级
// interrupt_response/create_response 开关生效 + M14 silence_duration_ms 运行时可调。
// 台账：docs/gaea-voice-port-distill-20261107.md。
package realtime

import (
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// ─────────── M12：晚到帧墓碑保留 ───────────

func TestOnSessionEndTombstonesResponse(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "答"})
	p.turns.StartTurn()
	rs := p.newResponseState("turn_1", 0, time.Time{})
	p.installResponse(rs) // 两态（审计 A3）：安装动作移入发送循环，测试显式模拟
	wire := p.session.out // OnSessionEnd 会置 p.session=nil，先捕获线缆

	p.OnSessionEnd(nil)

	if !rs.dead {
		t.Fatal("会话结束应标 dead")
	}
	p.muTerm.Lock()
	tombstoned := p.closedKeys[rs.key]
	p.muTerm.Unlock()
	if !tombstoned {
		t.Fatal("会话结束应落墓碑——晚到帧需保留丢弃依据（M12）")
	}
	// 残余 done 哨兵不得落入下一会话（墓碑拦截，无线缆帧）
	p.audioOut <- &audioOut{resp: rs, done: true}
	for p.drainOnce() {
	}
	select {
	case ev := <-wire:
		t.Fatalf("墓碑后残余哨兵不得装配终态帧, got %v", ev["type"])
	default:
	}
}

// blockingHandler M12 挂具：OnSessionEnd 阻塞至放行，验证槽位晚于终态清理释放。
// 其余四个接口点显式 no-op（内嵌 nil 接口在连接期 OnSessionConfig 即炸）。
type blockingHandler struct {
	release chan struct{}
	ended   chan struct{}
}

func (h *blockingHandler) OnAudio(*Session, []byte)                {}
func (h *blockingHandler) OnSessionConfig(*Session, SessionConfig) {}
func (h *blockingHandler) OnResponseCreate(*Session)               {}
func (h *blockingHandler) OnResponseCancel(*Session)               {}
func (h *blockingHandler) OnPlaybackDrained(*Session)              {}
func (h *blockingHandler) OnSessionEnd(s *Session) {
	close(h.ended)
	<-h.release
}

func TestSlotReleasedAfterSessionEnd(t *testing.T) {
	bh := &blockingHandler{release: make(chan struct{}), ended: make(chan struct{})}
	srv := testServer(t, bh)
	conn1 := dial(t, srv)
	_ = conn1.Close() // 触发断开序（读循环退出 → OnSessionEnd 阻塞中）

	select {
	case <-bh.ended:
	case <-time.After(3 * time.Second):
		t.Fatal("OnSessionEnd 未被调用")
	}

	// OnSessionEnd 进行中：槽位必须仍被持有（新连接被拒=session_limit_reached）
	u := url.URL{Scheme: "ws", Host: srv.Addr, Path: "/v1/realtime"}
	conn2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("第二连接应能建连（以拒绝帧回绝）: %v", err)
	}
	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	var rejected bool
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var ev map[string]any
		if err := conn2.ReadJSON(&ev); err != nil {
			break // 1008 关闭帧
		}
		if ev["type"] == "error" || ev["error"] != nil {
			rejected = true
			break
		}
	}
	if !rejected {
		t.Fatal("OnSessionEnd 未完成时槽位不得释放（M12 关闭序）")
	}
	_ = conn2.Close()

	close(bh.release) // 放行终态清理 → 槽位释放 → 新连接可入
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn3, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err == nil {
			_ = conn3.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("OnSessionEnd 完成并释放槽位后新连接应能入")
}

// ─────────── M13：会话级开关生效 ───────────

func TestSessionConfigSwitches(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "答"})
	if !p.interruptResponseOn() || !p.createResponseOn() {
		t.Fatal("缺省应开（barge-in 开、create_response 用 cfg 构造值）")
	}

	no := false
	yes := true
	p.OnSessionConfig(nil, SessionConfig{VAD: ServerVadConfig{InterruptResponse: &no, CreateResponse: &no}})
	if p.interruptResponseOn() || p.createResponseOn() {
		t.Fatal("session.update 关闭后两开关应生效为关")
	}

	// barge-in 关：speech_started(interrupt) 不再取消当前响应
	p.turns.StartTurn()
	rs := p.newResponseState("turn_1", 0, time.Time{})
	p.dispatchText(&textEvent{kind: evKindSpeechStarted, started: &VADStarted{TurnID: "turn_1", Rev: 1, InterruptResponse: true}})
	select {
	case <-rs.ctx.Done():
		t.Fatal("interrupt_response=off 不得取消当前响应")
	default:
	}

	// create_response 关：STT final 不再排队生成（无线缆 response.created）
	st := fakeSTT{"测试话语"}
	p.stt = st
	p.turns.SegmentFinalized(2000, 0, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_2", Rev: 0}
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		select {
		case ev := <-p.session.out:
			if ev["type"] == "response.created" {
				t.Fatal("create_response=off 不得自动生成响应")
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// 会话结束复位（上游 runtime_config 每连接一份）
	p.OnSessionEnd(nil)
	p.interruptResponse = &yes // 防御：直接驱过的指针不遮蔽断言
	p.OnSessionConfig(nil, SessionConfig{})
	_ = yes
}

// ─────────── M14：silence_duration_ms 运行时可调 ───────────

func TestSetMinSilenceMsAndRestore(t *testing.T) {
	h := NewVADHandler(DefaultVADConfig(), nil, nil, NewSpeculativeTurns())
	def := DefaultVADConfig().SampleRate * DefaultVADConfig().MinSilenceMs / 1000
	if h.iter.minSilence != def {
		t.Fatalf("前置：构造值 %d", h.iter.minSilence)
	}
	h.SetMinSilenceMs(200)
	if h.iter.minSilence != DefaultVADConfig().SampleRate*200/1000 {
		t.Fatalf("运行时应生效, got %d", h.iter.minSilence)
	}
	h.SetMinSilenceMs(-5) // 负值钳 0
	if h.iter.minSilence != 0 {
		t.Fatalf("负值应钳 0, got %d", h.iter.minSilence)
	}
	h.OnSessionEnd()
	if h.iter.minSilence != def {
		t.Fatalf("会话结束应回落构造值, got %d", h.iter.minSilence)
	}
}

func TestParseSessionUpdateSilenceDuration(t *testing.T) {
	// parseSessionUpdate 收的是 session.update 的 session 字段（sessionUpdateWire 形状）
	raw := []byte(`{"type":"realtime","audio":{"input":{"turn_detection":{"type":"server_vad","threshold":0.6,"silence_duration_ms":200,"interrupt_response":false}}}}`)
	cfg, err := parseSessionUpdate(raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if cfg.VAD.SilenceDurationMs == nil || *cfg.VAD.SilenceDurationMs != 200 {
		t.Fatalf("silence_duration_ms 应解析, got %v", cfg.VAD.SilenceDurationMs)
	}
	if cfg.VAD.Threshold == nil || *cfg.VAD.Threshold != 0.6 {
		t.Fatal("threshold 应照常解析")
	}
	if cfg.VAD.InterruptResponse == nil || *cfg.VAD.InterruptResponse {
		t.Fatal("interrupt_response 应解析为 false")
	}
}
