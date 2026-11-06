// selfhost.go — 离线门隔离服务端（2026-11-07）：进程内拉真引擎管线（随机端口），
// 不碰生产 8791 单槽（单槽被探针抢走=用户实盘会话被打断）。供 -self-host 模式
// 使用：三场景协议回归 / corpus EOT 门 / eot_analyze 日志切片（Smart Turn 判决
// 行+slog 落盘 outDir/selfhost_server.log）。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/voice/native"
	"github.com/gaea/gaea/internal/voice/realtime"
)

// selfhostEOT smart-turn 适配（native.Engine 公有方法直 delegate）。
type selfhostEOT struct{ e *native.Engine }

func (a selfhostEOT) Predict(pcm []float32) (float32, time.Duration, error) {
	return a.e.EndOfTurnTiming(pcm, native.SampleRate)
}

// selfhostSTT paraformer 适配。
type selfhostSTT struct{ e *native.Engine }

func (a selfhostSTT) Transcribe(pcm []float32, sampleRate int) (string, error) {
	return a.e.Transcribe(pcm, sampleRate)
}

// selfhostLegs 假响应腿：罐头正文+静音 PCM（bargein 场景要下行音频可插话；
// 静音合成=回声门不误触，断言面更干净）。TTS 每句出 3 块 6400B 静音。
type selfhostLegs struct{ reply string }

func (l *selfhostLegs) Reply(_ context.Context, _ string, _ []realtime.ChatMessage, _ string, _ int, _ string, onDelta func(string)) (string, int, int, error) {
	for _, r := range l.reply {
		onDelta(string(r))
	}
	return l.reply, 3, 5, nil
}

func (l *selfhostLegs) Speak(_ context.Context, sentence string, onAudio func([]byte)) error {
	if sentence == "" || sentence == "\x00mark" {
		return nil
	}
	// 300ms/块≈真实 TTS 延迟剖面——假腿毫秒级完成会让 cancel 场景的
	// response.cancel 永远迟到（响应先 completed=测试伪象非管线病）
	for i := 0; i < 3; i++ {
		time.Sleep(300 * time.Millisecond)
		onAudio(make([]byte, 6400))
	}
	return nil
}

// selfHostServer 拉起隔离管线服务端。返回 (shutdown, ws URL, error)。
// create_response=开（bargein/cancel 场景需要响应面）；LLM/TTS 全假腿——
// 在靶的只有 VAD/STT/EOT/轮次机/协议层。
func selfHostServer(modelsRoot, outDir string) (func(), string, error) {
	modelMust := func(name string) string {
		p := filepath.Join(modelsRoot, name)
		if _, err := os.Stat(p); err != nil {
			return ""
		}
		return p
	}
	eng, err := native.New(native.Config{
		SmartTurnModelPath:   modelMust("smart-turn-v3.2-cpu.onnx"),
		ParaformerModelPath:  modelMust("model.int8.onnx"),
		ParaformerTokensPath: modelMust("tokens.txt"),
	})
	if err != nil {
		return nil, "", fmt.Errorf("原生引擎装配: %w", err)
	}
	sileroOnnx := modelMust("silero_vad_v5_16k.onnx")
	silero, err := eng.NewSileroVAD(sileroOnnx)
	if err != nil {
		eng.Close()
		return nil, "", fmt.Errorf("silero 装配: %w", err)
	}
	// 服务端日志落盘（eot_analyze 切片消费）
	logPath := filepath.Join(outDir, "selfhost_server.log")
	lf, err := os.Create(logPath)
	if err != nil {
		silero.Destroy()
		eng.Close()
		return nil, "", err
	}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(lf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	legs := &selfhostLegs{reply: "好的"}
	pipe := realtime.NewPipeline(realtime.PipelineConfig{
		VAD:            realtime.DefaultVADConfig(),
		Legs:           legs,
		CreateResponse: true,
	}, silero, selfhostEOT{eng}, selfhostSTT{eng})
	srv := &realtime.Server{Addr: "127.0.0.1:0", Handler: pipe}
	if err := srv.Start(); err != nil {
		slog.SetDefault(prev)
		lf.Close()
		silero.Destroy()
		eng.Close()
		return nil, "", err
	}
	url := "ws://" + srv.Addr + "/v1/realtime"
	slog.Info("self-host 隔离管线就绪", "url", url, "log", logPath)
	shutdown := func() {
		srv.Stop()
		_ = http.DefaultClient.CloseIdleConnections
		slog.SetDefault(prev)
		lf.Close()
		silero.Destroy()
		eng.Close()
	}
	return shutdown, url, nil
}
