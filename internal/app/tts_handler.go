package app

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wubigork/wubigork/internal/tts"
)

// ── TTS 语音朗读 ─────────────────────────────────────────────

// GetTTSConfig 获取 TTS 配置（VoxCPM 已移除，返回空配置）
func (a *App) GetTTSConfig() map[string]interface{} {
	return map[string]interface{}{
		"modelPath":  "",
		"serverPath": "",
		"port":       0,
		"backend":    "",
		"speed":      1.0,
	}
}

// SaveTTSConfig 保存 TTS 配置（VoxCPM 已移除，无操作）
func (a *App) SaveTTSConfig(modelPath string, serverPath string, port int, backend string, speed float64) error {
	return nil
}

// GetTTSStatus 获取 TTS 状态（VoxCPM 已移除，始终返回未运行）
func (a *App) GetTTSStatus() map[string]interface{} {
	return map[string]interface{}{
		"running": false,
		"port":    0,
	}
}

// StartTTSServer 启动 TTS 服务（VoxCPM 已移除，无操作）
func (a *App) StartTTSServer(modelPath string, port int, backend string) error {
	return nil
}

// StopTTSServer 停止 TTS 服务（VoxCPM 已移除，无操作）
func (a *App) StopTTSServer() error {
	return nil
}

// TTSSpeak 合成语音并返回文件路径（VoxCPM 已移除，返回错误）
func (a *App) TTSSpeak(text string) (string, error) {
	return "", fmt.Errorf("VoxCPM 已移除，请使用朗读按钮（Base64 模式）")
}

// SetActiveTTSModel 设置用户选中的 TTS 模型
func (a *App) SetActiveTTSModel(engineID, modelID string) error {
	a.activeTTSEngine = engineID
	a.activeTTSModel = modelID
	return nil
}

// GetActiveTTSModel 获取用户选中的 TTS 模型
func (a *App) GetActiveTTSModel() map[string]string {
	return map[string]string{
		"engine": a.activeTTSEngine,
		"model":  a.activeTTSModel,
	}
}

func (a *App) TTSSpeakBase64(text string) (map[string]interface{}, error) {
	// 1. 用户选中的 TTS 模型
	if a.activeTTSModel != "" && a.engineMgr != nil {
		if eng, ok := a.engineMgr.GetEngine(a.activeTTSEngine); ok && eng.Enabled {
			htts := tts.NewHerdsmanTTS(eng.BaseURL, a.activeTTSModel, "Cherry")
			if audio, err := htts.Synthesize(text); err == nil && len(audio) > 0 {
				return map[string]interface{}{"base64": base64.StdEncoding.EncodeToString(audio), "mimeType": "audio/mp3"}, nil
			}
			slog.Debug("用户选中TTS模型失败，尝试其他模型", "engine", a.activeTTSEngine, "model", a.activeTTSModel)
		}
	}

	// 2. 扫描所有引擎的 TTS 模型列表
	if a.engineMgr != nil {
		for _, eid := range []string{"herdsman", "ollama", "xai", "deepseek"} {
			eng, ok := a.engineMgr.GetEngine(eid)
			if !ok || !eng.Enabled {
				continue
			}
			for _, m := range eng.Models {
				id := strings.ToLower(m.ID)
				if m.Status != "" && m.Status != "running" {
					continue
				}
				if strings.Contains(id, "tts") || strings.Contains(id, "voice") || strings.Contains(id, "speech") {
					voice := "Cherry"
					// 为已知模型匹配最佳声音
					if strings.Contains(id, "edge") {
						voice = "zh-CN-YunxiNeural"
					} else if strings.Contains(id, "qwen3") {
						if strings.Contains(id, "voicedesign") {
							voice = ""
						} else {
							voice = "Cherry"
						}
					}
					htts := tts.NewHerdsmanTTS(eng.BaseURL, m.ID, voice)
					if audio, err := htts.Synthesize(text); err == nil && len(audio) > 0 {
						slog.Info("TTS 自动选择模型", "engine", eid, "model", m.ID)
						return map[string]interface{}{"base64": base64.StdEncoding.EncodeToString(audio), "mimeType": "audio/mp3"}, nil
					}
				}
			}
		}
	}

	// 3. Edge TTS（在线，免费）
	if edge := tts.NewEdgeTTS(); edge != nil {
		if audio, err := edge.Synthesize(text); err == nil && len(audio) > 0 {
			return map[string]interface{}{"base64": base64.StdEncoding.EncodeToString(audio), "mimeType": "audio/mp3"}, nil
		}
	}

	// 4. Windows SAPI（本地系统自带）
	if sapi := tts.NewWinTTS(); sapi != nil {
		if audio, err := sapi.Synthesize(text); err == nil && len(audio) > 0 {
			return map[string]interface{}{"base64": base64.StdEncoding.EncodeToString(audio), "mimeType": "audio/wav"}, nil
		}
	}

	return nil, fmt.Errorf("无可用 TTS 模型：请在模型中心启动一个语音模型")
}

// TTSSpeakStreaming 流式合成：逐句生成。
// 引擎优先级：Herdsman TTS → xAI TTS → Edge TTS → WinTTS (SAPI)
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
			for model, voice := range map[string]string{
				"edge-tts": "zh-CN-YunxiNeural", "qwen3-tts-customvoice": "Cherry", "qwen3-tts-voicedesign": "",
			} {
				htts := tts.NewHerdsmanTTS(herdEngine.BaseURL, model, voice)
				engines = append(engines, htts)
				metas = append(metas, struct{ Label string; Format string }{"herdsman-" + model, "mp3"})
			}
		}
	}

	// 1. Edge TTS（免费在线）
	edgeTTS := tts.NewEdgeTTS()
	engines = append(engines, edgeTTS)
	metas = append(metas, struct{ Label string; Format string }{"edge", "mp3"})

	// 3. WinTTS SAPI（离线）
	engines = append(engines, tts.NewWinTTS())
	metas = append(metas, struct{ Label string; Format string }{"sapi", "wav"})

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
				audio, err = activeEngine.Synthesize(sentence)
			}

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
				switch activeLabel {
				case "edge":
					activeEngine = edgeTTS
				case "sapi":
					activeEngine = tts.NewWinTTS()
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
