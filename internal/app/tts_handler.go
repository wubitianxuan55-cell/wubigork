package app

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wubigork/wubigork/internal/tts"
)

// ── TTS 语音朗读 ─────────────────────────────────────────────

// autoStartTTS 应用启动时自动初始化 TTS 客户端
func (a *App) autoStartTTS() {
	modelPath := a.cfg.TTSModelPath
	if modelPath == "" {
		return
	}
	if _, err := os.Stat(modelPath); err != nil {
		slog.Warn("TTS 模型文件不存在", "path", modelPath)
		return
	}

	binaryPath := a.resolveTTSBinary()
	if binaryPath == "" {
		slog.Warn("找不到 voxcpm_tts，TTS 不可用")
		return
	}

	a.ttsClient = tts.NewClient(binaryPath, modelPath, a.cfg.TTSBackend)
	slog.Info("TTS 客户端已就绪", "model", modelPath)
}

// resolveTTSBinary 查找 voxcpm_tts 可执行文件
func (a *App) resolveTTSBinary() string {
	// 1. 配置的 TTSBinaryPath
	if a.cfg.TTSBinaryPath != "" {
		if _, err := os.Stat(a.cfg.TTSBinaryPath); err == nil {
			return a.cfg.TTSBinaryPath
		}
	}

	// 2. TTSServerPath 同目录下的 voxcpm_tts.exe（兼容旧配置）
	if a.cfg.TTSServerPath != "" {
		dir := filepath.Dir(a.cfg.TTSServerPath)
		candidate := filepath.Join(dir, "voxcpm_tts.exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// 3. PATH 中查找
	if lp, err := exec.LookPath("voxcpm_tts"); err == nil {
		return lp
	}

	return ""
}

// StartTTSServer 启动 voxcpm-server（保留兼容，实际合成使用 CLI）
func (a *App) StartTTSServer(modelPath string, port int, backend string) error {
	// CLI 模式不需要启动服务，初始化客户端即可
	if modelPath == "" {
		modelPath = a.cfg.TTSModelPath
	}
	if backend == "" {
		backend = a.cfg.TTSBackend
	}
	if modelPath == "" {
		return fmt.Errorf("请先在设置中配置 GGUF 模型文件路径")
	}

	binaryPath := a.resolveTTSBinary()
	if binaryPath == "" {
		return fmt.Errorf("找不到 voxcpm_tts，请确认 VoxCPM.cpp 已安装")
	}

	a.ttsClient = tts.NewClient(binaryPath, modelPath, backend)
	slog.Info("TTS 客户端已初始化", "model", modelPath, "backend", backend)
	return nil
}

// StopTTSServer 停止 TTS（CLI 模式无需操作）
func (a *App) StopTTSServer() error {
	a.ttsClient = nil
	return nil
}

// TTSSpeak 合成语音并返回文件路径
func (a *App) TTSSpeak(text string) (string, error) {
	client, err := a.getOrInitClient()
	if err != nil {
		return "", err
	}

	outputPath := filepath.Join(os.TempDir(), tts.TempDirName, tts.OutputWAV)
	if err := client.SynthesizeToFile(text, outputPath); err != nil {
		return "", err
	}

	return outputPath, nil
}
func (a *App) TTSSpeakBase64(text string) (map[string]interface{}, error) {
	// 遍历所有引擎找 TTS 模型，优先 Herdsman
	if a.engineMgr != nil {
		for _, eid := range []string{"herdsman", "xai"} {
			eng, ok := a.engineMgr.GetEngine(eid)
			if !ok || !eng.Enabled { continue }
			for _, m := range eng.Models {
				id := strings.ToLower(m.ID)
				if m.Status != "" && m.Status != "running" { continue }
				if strings.Contains(id, "tts") || strings.Contains(id, "voice") {
					htts := tts.NewHerdsmanTTS(eng.BaseURL, m.ID, "aiden")
					if audio, err := htts.Synthesize(text); err == nil && len(audio) > 0 {
						return map[string]interface{}{
							"base64":   base64.StdEncoding.EncodeToString(audio),
							"mimeType": "audio/mp3",
						}, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("无可用 TTS 模型：请在模型中心启动一个语音模型")
}
// TTSSpeakStreaming 流式合成：逐句生成。

// TTSSpeakStreaming 流式合成：逐句生成。
// 引擎优先级：xAI TTS → Edge TTS → WinTTS (SAPI) → VoxCPM (如可用)
func (a *App) TTSSpeakStreaming(text string) error {
	sentences := tts.SplitSentences(text)
	if len(sentences) == 0 {
		return fmt.Errorf("无可朗读的文本")
	}

	var engines []tts.Synthesizer
	var metas []struct {
		Label  string
		Format string
	}

	// 0. Herdsman TTS（本地优先）
	if a.engineMgr != nil {
		herdEngine, ok := a.engineMgr.GetEngine("herdsman")
		if ok && herdEngine.Enabled {
			for _, m := range herdEngine.Models {
				id := strings.ToLower(m.ID)
				if m.Status != "" && m.Status != "running" { continue }
				if strings.Contains(id, "tts") || strings.Contains(id, "voice") {
					htts := tts.NewHerdsmanTTS(herdEngine.BaseURL, m.ID, "aiden")
					engines = append(engines, htts)
					metas = append(metas, struct{ Label string; Format string }{"herdsman-tts", "mp3"})
					break
				}
			}
		}
	}

	// 1. xAI TTS（在线，最高质量，$15/1M chars）
	if a.client != nil {
		xaiTTS := tts.NewXaiTTS(
			a.cfg.XaiAPIBaseURL,
			"eve",
			a.client.GetToken,
			&http.Client{Timeout: 30 * time.Second},
		)
		engines = append(engines, xaiTTS)
		metas = append(metas, struct {
			Label  string
			Format string
		}{"xai", "mp3"})
	}

	// 1. Edge TTS（免费在线，最自然）
	edgeTTS := tts.NewEdgeTTS()
	engines = append(engines, edgeTTS)
	metas = append(metas, struct {
		Label  string
		Format string
	}{"edge", "mp3"})

	// 2. WinTTS SAPI（离线，零延迟但机械，回退方案）
	engines = append(engines, tts.NewWinTTS())
	metas = append(metas, struct {
		Label  string
		Format string
	}{"sapi", "wav"})

	// 3. VoxCPM（本地AI模型，需提前配置，可选）
	if a.ttsClient != nil {
		engines = append(engines, a.ttsClient)
		metas = append(metas, struct {
			Label  string
			Format string
		}{"voxcpm", "wav"})
	}

	chain := tts.NewSynthesizerChain(engines...)

	slog.Info("流式 TTS 开始", "sentences", len(sentences), "total_chars", len([]rune(text)))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("TTS goroutine panic", "panic", r)
				a.emit("tts-stream", map[string]interface{}{
					"type": "error", "error": fmt.Sprintf("内部错误: %v", r),
				})
			}
		}()

		var activeEngine tts.Synthesizer
		var activeFormat string
		var activeLabel string

		for i, sentence := range sentences {
			select {
			case <-a.ctx.Done():
				return
			default:
			}

			a.emit("tts-stream", map[string]interface{}{
				"type": "progress", "index": i, "total": len(sentences), "text": sentence,
			})

			var audio []byte
			var err error

			if activeEngine != nil {
				// 复用已选引擎
				audio, err = activeEngine.Synthesize(sentence)
			}

			// 引擎未定或已选引擎失败 → 用 chain 探测
			if activeEngine == nil || err != nil {
				if err != nil {
					slog.Warn("TTS 引擎失败，重新探测", "sentence", i, "error", err)
				}
				audio, activeFormat, activeLabel, err = chain.SynthesizeWithMeta(sentence, metas)
				if err != nil {
					slog.Error("所有 TTS 引擎均失败", "error", err)
					a.emit("tts-stream", map[string]interface{}{
						"type": "error", "error": "无可用的语音引擎，请检查网络或 TTS 配置",
					})
					return
				}
				// 根据 label 定位 engine 实例，供后续复用
				switch activeLabel {
				case "xai":
					activeEngine = tts.NewXaiTTS(
						a.cfg.XaiAPIBaseURL, "eve",
						a.client.GetToken,
						&http.Client{Timeout: 30 * time.Second},
					)
				case "edge":
					activeEngine = edgeTTS
				case "sapi":
					activeEngine = tts.NewWinTTS()
				case "voxcpm":
					activeEngine = a.ttsClient
				}
				slog.Info("TTS 引擎已选择", "engine", activeLabel, "format", activeFormat)
			}

			done := i == len(sentences)-1
			a.emit("tts-stream", map[string]interface{}{
				"type":     "chunk",
				"index":    i,
				"total":    len(sentences),
				"audio":    base64.StdEncoding.EncodeToString(audio),
				"mimeType": "audio/" + activeFormat,
				"engine":   activeLabel,
				"done":     done,
			})
		}

		a.emit("tts-stream", map[string]interface{}{"type": "done", "engine": activeLabel})
	}()

	return nil
}

// getOrInitClient 获取或懒初始化 TTS 客户端
func (a *App) getOrInitClient() (*tts.Client, error) {
	if a.ttsClient != nil {
		return a.ttsClient, nil
	}

	modelPath := a.cfg.TTSModelPath
	if modelPath == "" {
		return nil, fmt.Errorf("未配置 TTS 模型路径")
	}
	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("TTS 模型文件不存在: %s", modelPath)
	}

	binaryPath := a.resolveTTSBinary()
	if binaryPath == "" {
		return nil, fmt.Errorf("找不到 voxcpm_tts 可执行文件")
	}

	a.ttsClient = tts.NewClient(binaryPath, modelPath, a.cfg.TTSBackend)
	return a.ttsClient, nil
}

// GetTTSStatus 获取 TTS 状态
func (a *App) GetTTSStatus() map[string]interface{} {
	ready := a.ttsClient != nil || (a.cfg.TTSModelPath != "" && a.resolveTTSBinary() != "")
	return map[string]interface{}{
		"running": ready,
		"port":    0,
	}
}

// GetTTSConfig 获取 TTS 配置
func (a *App) GetTTSConfig() map[string]interface{} {
	return map[string]interface{}{
		"modelPath":  a.cfg.TTSModelPath,
		"serverPath": a.cfg.TTSServerPath,
		"port":       a.cfg.TTSPort,
		"backend":    a.cfg.TTSBackend,
		"speed":      a.cfg.TTSSpeed,
	}
}

// SaveTTSConfig 保存 TTS 配置
func (a *App) SaveTTSConfig(modelPath string, serverPath string, port int, backend string, speed float64) error {
	if modelPath != "" {
		a.cfg.TTSModelPath = modelPath
	}
	if serverPath != "" {
		a.cfg.TTSServerPath = serverPath
	}
	if port > 0 {
		a.cfg.TTSPort = port
	}
	if backend != "" {
		a.cfg.TTSBackend = backend
	}
	if speed > 0 {
		a.cfg.TTSSpeed = speed
	}

	// 重建客户端
	a.ttsClient = nil
	if binaryPath := a.resolveTTSBinary(); binaryPath != "" {
		if _, err := os.Stat(a.cfg.TTSModelPath); err == nil {
			a.ttsClient = tts.NewClient(binaryPath, a.cfg.TTSModelPath, a.cfg.TTSBackend)
		}
	}

	a.emit("tts-config-changed", nil)
	return nil
}
