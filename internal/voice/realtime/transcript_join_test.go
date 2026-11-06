// transcript_join_test.go — done 转写拼接契约（2026-11-06 实弹翻车钉）。
// parts 是 delta 粒度，done 转写必须等于流式 delta 原样拼接；句级 " " join
// 挂在 delta 粒度上即「你 好 世 界」逐字撕裂（落库/面板/TTS 记忆全被污染）。
package realtime

import (
	"strings"
	"testing"
	"time"
)

func TestTranscriptDoneJoinsDeltasVerbatim(t *testing.T) {
	p := newFakePipeline(t, &fakeLegs{reply: "你好，世界"})
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
loop:
	for {
		select {
		case ev := <-p.session.out:
			if ev["type"] == "response.output_audio_transcript.done" {
				if s, ok := ev["transcript"].(string); ok {
					doneText = s
				}
			}
		default:
			break loop
		}
	}
	if doneText != "你好，世界" {
		t.Fatalf("done 转写应为流式 delta 原样拼接, got %q", doneText)
	}
	if strings.Contains(doneText, " ") {
		t.Fatalf("done 转写不得含拼接空格, got %q", doneText)
	}
}
