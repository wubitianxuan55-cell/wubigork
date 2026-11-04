//go:build windows && amd64 && cgo

// pipeline_probe_test.go — 管线裸探（绕 WS 层）：OnAudio 直喂，排水槽看事件。
package realtime

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPipelineDirectProbe(t *testing.T) {
	pipe := newNativePipeline(t)
	sess := &Session{ID: "probe", out: make(chan map[string]any, 65536), done: make(chan struct{})}

	raw, err := os.ReadFile(filepath.Join("..", "native", "testdata", "eot_complete_16k.wav"))
	if err != nil {
		t.Fatal(err)
	}
	pcm := raw[44:]

	frameBytes := 320 * 2
	feed := func(b []byte) { pipe.OnAudio(sess, b) }
	for off := 0; off+frameBytes <= len(pcm); off += frameBytes {
		feed(pcm[off : off+frameBytes])
	}
	slog := pipe.vad.AudioMs()
	t.Logf("after wav: audioMs=%d triggered=%v emitted=%v", slog, pipe.vad.iter.triggered, pipe.vad.speechStartedEmitted)

	silence := make([]byte, 320*2)
	for i := 0; i < 50; i++ { // 50 帧 × 20ms = 1s 静音
		feed(silence)
		if i%10 == 9 {
			t.Logf("silence %d00ms: audioMs=%d triggered=%v", i+1, pipe.vad.AudioMs(), pipe.vad.iter.triggered)
		}
	}
	// 打概率分布：静音段前几块的 prob
	close(sess.done)
	var types []string
drain:
	for {
		select {
		case ev := <-sess.out:
			typ, _ := ev["type"].(string)
			types = append(types, typ)
		default:
			break drain
		}
	}
	t.Logf("events: %v", types)
	_ = binary.LittleEndian
	_ = time.Now()
}
