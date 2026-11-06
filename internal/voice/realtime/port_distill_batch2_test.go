// port_distill_batch2_test.go — 批2 蒸馏修刀契约钉（2026-11-07）。
// M4 输入门死线到期即放行 + M5 STT 失败发 failed 终态 + M6 audio.done 收口帧 +
// M7 created 补发（孤 done 封死）。台账：docs/gaea-voice-port-distill-20261107.md。
package realtime

import (
	"errors"
	"testing"
	"time"
)

// ─────────── M4：输入门死线语义 ───────────

func TestSttGateWait(t *testing.T) {
	now := time.Now()
	// 死线已到期=原样返回（holdUntil 不再持门，立即放行），不得回退 item delay
	expired := now.Add(-time.Second)
	if got := sttGateWait(expired, true, 0.4); !got.Equal(expired) {
		t.Fatalf("到期死线应原样返回, got %v", got)
	}
	// 未到期死线原样返回
	future := now.Add(300 * time.Millisecond)
	if got := sttGateWait(future, true, 0.4); !got.Equal(future) {
		t.Fatalf("未到期死线应原样返回, got %v", got)
	}
	// 无死线回退 item delay
	if got := sttGateWait(time.Time{}, false, 0.4); got.Before(now.Add(399 * time.Millisecond)) {
		t.Fatalf("无死线应回退 item delay, got %v", got)
	}
	// 都没有=零值（不等待）
	if got := sttGateWait(time.Time{}, false, 0); !got.IsZero() {
		t.Fatalf("无死线无延时应零值, got %v", got)
	}
}

// ─────────── M5：STT 失败发 failed 终态 ───────────

type failingSTT struct{}

func (failingSTT) Transcribe([]float32, int) (string, error) { return "", errors.New("boom") }

func TestSTTFailureEmitsFailedTerminal(t *testing.T) {
	p := newFakePipelineSTT(t, &fakeLegs{reply: "答"}, failingSTT{})
	p.turns.StartTurn()
	// 零宽限：输入门立放，STT 失败 → failed 终态按规则表（failed→gate 提交）发布
	p.turns.SegmentFinalized(1000, 0, 0)
	p.sttCh <- &VADFinal{Audio: make([]float32, 4096), TurnID: "turn_1", Rev: 0}

	deadline := time.Now().Add(3 * time.Second)
	gotFailed, gotCompleted := false, false
	for time.Now().Before(deadline) {
		select {
		case ev := <-p.session.out:
			switch ev["type"] {
			case "conversation.item.input_audio_transcription.failed":
				gotFailed = true
				errObj, _ := ev["error"].(map[string]any)
				if errObj == nil || errObj["code"] != "transcription_failed" {
					t.Fatalf("failed 帧形不符（error.code=transcription_failed）, got %v", ev)
				}
			case "conversation.item.input_audio_transcription.completed":
				gotCompleted = true
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
		if gotFailed {
			break
		}
	}
	if !gotFailed {
		t.Fatal("STT 失败应发 transcription.failed 终态")
	}
	if gotCompleted {
		t.Fatal("失败不得发空 transcript 的 completed（空泡病）")
	}
}

// ─────────── M6+M7：audio.done 收口 + created 补发 ───────────

func collectUntilDone(t *testing.T, p *Pipeline, respID string, d time.Duration) []string {
	t.Helper()
	deadline := time.Now().Add(d)
	var types []string
	for time.Now().Before(deadline) {
		select {
		case ev := <-p.session.out:
			tp, _ := ev["type"].(string)
			types = append(types, tp)
			if tp == "response.done" {
				if id, _ := ev["response"].(map[string]any)["id"].(string); id == respID {
					return types
				}
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	return types
}

func indexOf(types []string, want string) int {
	for i, tp := range types {
		if tp == want {
			return i
		}
	}
	return -1
}

func TestAudioDoneAndCreatedBeforeDone(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "好"})

	// A：零输出响应（R3 路径）——created 必须先于 done（M7），无 audio.done（M6 属主面）
	p.turns.StartTurn()
	rs0 := p.newResponseState("turn_1", 0, time.Time{})
	p.finishResponse(p.session, rs0, "cancelled", "empty_output")
	types := collectUntilDone(t, p, rs0.id, 3*time.Second)
	iCreated, iDone := indexOf(types, "response.created"), indexOf(types, "response.done")
	if iCreated < 0 || iDone < 0 || iCreated > iDone {
		t.Fatalf("零输出路径 created 应先于 done, got %v", types)
	}
	if indexOf(types, "response.output_audio.done") >= 0 {
		t.Fatalf("零音频响应不得发 audio.done, got %v", types)
	}

	// B：有音频响应——audio.done 在 response.done 前收口（M6）
	p.turns.StartTurn()
	rs1 := p.newResponseState("turn_2", 0, time.Time{})
	p.dispatchAudio(&audioOut{resp: rs1, pcm: make([]byte, 100)})
	p.finishResponse(p.session, rs1, "completed", "")
	types = collectUntilDone(t, p, rs1.id, 3*time.Second)
	iAudioDone := indexOf(types, "response.output_audio.done")
	iDone = indexOf(types, "response.done")
	if iAudioDone < 0 {
		t.Fatalf("有音频响应应发 audio.done, got %v", types)
	}
	if iDone < 0 || iAudioDone > iDone {
		t.Fatalf("audio.done 应先于 response.done, got %v", types)
	}
}
