//go:build windows && amd64 && cgo

// engine_windows_test.go — 三引擎裸测（真实 DLL + 真实模型）。
// 前置：模型在 C:\AI\models\voice\；DLL 目录经 PATH 或 GAEA_VOICE_NATIVE_DLL_DIR
// 提供（sherpa DLL 是链接期加载，必须在测试进程启动前可寻达）：
//
//	MODLIB="$(go env GOMODCACHE)/github.com/k2-fsa/sherpa-onnx-go-windows/v2@v2.0.1/lib/x86_64-pc-windows-gnu"
//	PATH="$MODLIB:$PATH" CC=/c/msys64/ucrt64/bin/gcc.exe CGO_ENABLED=1 go test ./internal/voice/native/ -v
package native

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const modelsRoot = `C:\AI\models\voice`

func modelsReady() bool {
	if _, err := os.Stat(filepath.Join(modelsRoot, "model.int8.onnx")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(modelsRoot, "smart-turn-v3.2-cpu.onnx")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(modelsRoot, "tokens.txt")); err != nil {
		return false
	}
	return true
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	if !modelsReady() {
		t.Skipf("模型缺失（%s）", modelsRoot)
	}
	e, err := New(Config{
		ORTRuntimePath:       filepath.Join(modelsRoot, "onnxruntime.dll"),
		SmartTurnModelPath:   filepath.Join(modelsRoot, "smart-turn-v3.2-cpu.onnx"),
		ParaformerModelPath:  filepath.Join(modelsRoot, "model.int8.onnx"),
		ParaformerTokensPath: filepath.Join(modelsRoot, "tokens.txt"),
		SileroModelPath:      filepath.Join(modelsRoot, "silero_vad.onnx"),
	})
	if err != nil {
		t.Fatalf("引擎装配失败: %v", err)
	}
	t.Cleanup(func() { e.Close() })
	return e
}

func goldenExpectations(t *testing.T) map[string]struct {
	P                float64 `json:"p"`
	ExpectedComplete bool    `json:"expected_complete"`
} {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "golden.json"))
	if err != nil {
		t.Fatalf("golden.json 缺失: %v", err)
	}
	var out map[string]struct {
		P                float64 `json:"p"`
		ExpectedComplete bool    `json:"expected_complete"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("golden.json 解析失败: %v", err)
	}
	return out
}

// TestEOTProbabilities — Go EOT 与 venv 金样同判：complete/incomplete 判决一致、
// 概率接近、推理耗时在 sidecar 量级（金样 20-30ms；CPU 会话放宽到 120ms）。
func TestEOTProbabilities(t *testing.T) {
	e := newTestEngine(t)
	golden := goldenExpectations(t)
	for _, name := range []string{"eot_complete", "eot_incomplete"} {
		t.Run(name, func(t *testing.T) {
			pcm := loadGoldenWav(t, name)
			p, took, err := e.EndOfTurnTiming(pcm, SampleRate)
			if err != nil {
				t.Fatalf("EOT: %v", err)
			}
			want := golden[name]
			got := p > 0.5
			if got != want.ExpectedComplete {
				t.Fatalf("判决翻转: p=%v want_complete=%v（金样 p=%v）", p, want.ExpectedComplete, want.P)
			}
			if math.Abs(float64(p)-want.P) > 0.05 {
				t.Fatalf("概率偏离金样: p=%v vs %v", p, want.P)
			}
			if took > 120*time.Millisecond {
				t.Fatalf("EOT 推理过慢: %v（sidecar 实测 ~20ms）", took)
			}
			t.Logf("p=%v (golden %v) took=%v", p, want.P, took)
		})
	}
}

// TestSTTBreezeUtterances — paraformer int8 转写两条 breeze 合成句，非空即过
// （文本对齐门在刀4 stt_compare 语料 A/B，这里只锁链路+中文形态）。
func TestSTTBreezeUtterances(t *testing.T) {
	e := newTestEngine(t)
	for _, name := range []string{"eot_complete", "eot_incomplete"} {
		pcm := loadGoldenWav(t, name)
		text, err := e.Transcribe(pcm, SampleRate)
		if err != nil {
			t.Fatalf("Transcribe(%s): %v", name, err)
		}
		if len([]rune(text)) == 0 {
			t.Fatalf("%s: 转写为空", name)
		}
		for _, r := range text {
			if r == ' ' {
				t.Fatalf("%s: 转写结果含空格（sidecar 语义为去空格）", name)
			}
		}
		t.Logf("%s → %q (%d 字, %v ms 音频)", name, text, len([]rune(text)), len(pcm)/16)
	}
}

// TestVADSegments — silero 在两条 wav 上各切出 ≥1 段语音。
func TestVADSegments(t *testing.T) {
	e := newTestEngine(t)
	for _, name := range []string{"eot_complete", "eot_incomplete"} {
		pcm := loadGoldenWav(t, name)
		segs := e.VADSegments(pcm)
		if len(segs) == 0 {
			t.Fatalf("%s: VAD 零语音段", name)
		}
		t.Logf("%s → %d 段", name, len(segs))
	}
}
