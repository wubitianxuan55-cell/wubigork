//go:build windows && amd64 && cgo

// pipeline_test.go — 管线端到端回放（真引擎）：wav 经 WS 喂入 → 断言协议事件序列。
package realtime

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/voice/native"
)

const pipelineModelsRoot = `C:\AI\models\voice`

func newNativePipeline(t *testing.T) *Pipeline {
	t.Helper()
	for _, p := range []string{
		filepath.Join(pipelineModelsRoot, "model.int8.onnx"),
		filepath.Join(pipelineModelsRoot, "smart-turn-v3.2-cpu.onnx"),
		filepath.Join(pipelineModelsRoot, "silero_vad_v5_16k.onnx"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("模型缺失: %s", p)
		}
	}
	eng, err := native.New(native.Config{
		SmartTurnModelPath:   filepath.Join(pipelineModelsRoot, "smart-turn-v3.2-cpu.onnx"),
		ParaformerModelPath:  filepath.Join(pipelineModelsRoot, "model.int8.onnx"),
		ParaformerTokensPath: filepath.Join(pipelineModelsRoot, "tokens.txt"),
	})
	if err != nil {
		t.Fatalf("引擎装配: %v", err)
	}
	// 清理 LIFO：管线阶段线程是常驻 goroutine——先停管线（最后注册=最先执行），
	// 等 worker 退出 select 后再 Destroy 引擎，避免 use-after-Destroy 竞态。
	silero, err := eng.NewSileroVAD(filepath.Join(pipelineModelsRoot, "silero_vad_v5_16k.onnx"))
	if err != nil {
		eng.Close()
		t.Fatalf("silero 装配: %v", err)
	}
	pipe := NewPipeline(PipelineConfig{VAD: DefaultVADConfig()}, silero,
		nativeEOTAdapter{eng}, nativeSTTAdapter{eng})
	t.Cleanup(func() {
		pipe.OnSessionEnd(nil)
		time.Sleep(300 * time.Millisecond) // 等 VAD/STT worker 从 ORT 调用返回
		silero.Destroy()
		eng.Close()
	})
	return pipe
}

type nativeEOTAdapter struct{ e *native.Engine }

func (a nativeEOTAdapter) Predict(pcm []float32) (float32, time.Duration, error) {
	return a.e.EndOfTurnTiming(pcm, native.SampleRate)
}

type nativeSTTAdapter struct{ e *native.Engine }

func (a nativeSTTAdapter) Transcribe(pcm []float32, sampleRate int) (string, error) {
	return a.e.Transcribe(pcm, sampleRate)
}

// TestPipelineReplayUtterance — 回放断言（刀3/4 协议面，无响应腿）：
// 整句话（含 64ms 停顿引起的 fragment+reopen）只发一次 speech_started（item 复用）；
// speech_stopped/committed/transcription 被持（等响应提交，刀5 发布）。
// 转写/Smart Turn/reopen 全链在日志叙事中断言（TestPipelineDirectProbe）。
func TestPipelineReplayUtterance(t *testing.T) {
	pipe := newNativePipeline(t)
	srv := &Server{Addr: "127.0.0.1:0", Handler: pipe}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(srv.Stop)

	conn := dial(t, srv)
	recvUntil(t, conn, func(ev map[string]any) bool { return mustType(t, ev) == "session.created" }, 2*time.Second)

	// 读 wav → 320 样本帧（模拟前端 20ms 粒度）
	raw, err := os.ReadFile(filepath.Join("..", "native", "testdata", "eot_complete_16k.wav"))
	if err != nil {
		t.Fatalf("wav: %v", err)
	}
	const headerLen = 44
	pcmBytes := raw[headerLen:]
	frameBytes := 320 * 2
	for off := 0; off+frameBytes <= len(pcmBytes); off += frameBytes {
		_ = conn.WriteJSON(map[string]any{
			"type":  "input_audio_buffer.append",
			"audio": base64.StdEncoding.EncodeToString(pcmBytes[off : off+frameBytes]),
		})
	}
	// 1s 尾静音（ws_probe 同配方）：VAD 要 64ms 静音才判段尾
	silence := make([]byte, 16000*2)
	for off := 0; off+frameBytes <= len(silence); off += frameBytes {
		_ = conn.WriteJSON(map[string]any{
			"type":  "input_audio_buffer.append",
			"audio": base64.StdEncoding.EncodeToString(silence[off : off+frameBytes]),
		})
	}

	// 收 3s 内的协议帧：断言 speech_started 恰一次（fragment+reopen item 复用），
	// 且无任何持发帧提前泄漏（stopped/committed/transcription 只随响应提交发布）。
	deadline := time.Now().Add(3 * time.Second)
	started := 0
	itemIDs := map[string]bool{}
	var leaked []string
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, r, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var ev map[string]any
		if json.Unmarshal(r, &ev) != nil {
			continue
		}
		typ, _ := ev["type"].(string)
		switch typ {
		case "input_audio_buffer.speech_started":
			started++
			if id, _ := ev["item_id"].(string); id != "" {
				itemIDs[id] = true
			}
		case "input_audio_buffer.speech_stopped",
			"input_audio_buffer.committed",
			"conversation.item.created",
			"conversation.item.input_audio_transcription.completed":
			leaked = append(leaked, typ)
		}
	}
	if started != 1 {
		t.Fatalf("speech_started 次数=%d want 1（重开必须复用 item）", started)
	}
	if len(itemIDs) != 1 {
		t.Fatalf("item 复用失败: %v", itemIDs)
	}
	if len(leaked) != 0 {
		t.Fatalf("持发帧提前泄漏: %v", leaked)
	}
	_ = strings.TrimSpace
}
