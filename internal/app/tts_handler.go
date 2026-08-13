package app

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/tts"
)

// ── TTS 语音朗读 ──────────────────────────────────────────────────────────────

// GetTTSConfig 获取旧版子进程 TTS 配置（已废弃：TTS 改为模型中心引擎模式，返回空配置）
func (a *mediaState) GetTTSConfig() map[string]interface{} {
	return map[string]interface{}{
		"modelPath":  "",
		"serverPath": "",
		"port":       0,
		"backend":    "",
		"speed":      1.0,
	}
}

// SaveTTSConfig 保存旧版子进程 TTS 配置（已废弃：TTS 改为模型中心引擎模式，无操作）
func (a *mediaState) SaveTTSConfig(modelPath string, serverPath string, port int, backend string, speed float64) error {
	return nil
}

// GetTTSStatus 获取旧版子进程 TTS 状态（已废弃：TTS 改为模型中心引擎模式，始终返回未运行）
func (a *mediaState) GetTTSStatus() map[string]interface{} {
	return map[string]interface{}{
		"running": false,
		"port":    0,
	}
}

// StartTTSServer 启动旧版子进程 TTS 服务（已废弃：TTS 改为模型中心引擎模式，无操作）
func (a *mediaState) StartTTSServer(modelPath string, port int, backend string) error {
	return nil
}

// StopTTSServer 停止旧版子进程 TTS 服务（已废弃：TTS 改为模型中心引擎模式，无操作）
func (a *mediaState) StopTTSServer() error {
	return nil
}

// TTSSpeak 合成语音并返回文件路径（旧版接口已废弃，返回错误）
func (a *mediaState) TTSSpeak(text string) (string, error) {
	return "", fmt.Errorf("请使用朗读按钮（Base64/流式模式）")
}

// TTSSpeakBase64 合成语音并返回 Base64 音频
// 引擎优先级：用户选中的 TTS 模型 → 扫描各引擎 TTS 模型 → Edge TTS → WinTTS (SAPI)
func (a *mediaState) TTSSpeakBase64(text string) (map[string]interface{}, error) {
	// 1. 用户选中的 TTS 模型
	if a.activeTTSModel != "" && a.engineMgr != nil {
		if eng, ok := a.engineMgr.GetEngine(a.activeTTSEngine); ok && eng.Enabled {
			if audio, mime, ok := a.tryEngineTTS(a.activeTTSEngine, a.activeTTSModel, text, ""); ok {
				if mime == "" {
					mime = "audio/mp3"
				}
				return map[string]interface{}{"base64": base64.StdEncoding.EncodeToString(audio), "mimeType": mime}, nil
			}
			slog.Debug("用户选中TTS模型失败，尝试其他模型", "engine", a.activeTTSEngine, "model", a.activeTTSModel)
		}
	}

	// 2. 扫描所有引擎的 TTS 模型列表
	if a.engineMgr != nil {
		for _, eid := range []string{"herdsman", "cosyvoice", "ollama", "xai", "deepseek"} {
			eng, ok := a.engineMgr.GetEngine(eid)
			if !ok || !eng.Enabled {
				continue
			}
			for _, m := range eng.Models {
				id := strings.ToLower(m.ID)
				if m.Status != "" && m.Status != "running" {
					continue
				}
				if strings.Contains(id, "tts") || strings.Contains(id, "voice") || strings.Contains(id, "speech") || strings.Contains(id, "voxcpm") {
					if audio, mime, ok := a.tryEngineTTS(eid, m.ID, text, ""); ok {
						slog.Info("TTS 自动选择模型", "engine", eid, "model", m.ID)
						if mime == "" {
							mime = "audio/mp3"
						}
						return map[string]interface{}{"base64": base64.StdEncoding.EncodeToString(audio), "mimeType": mime}, nil
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

	return nil, fmt.Errorf("无可用的 TTS 模型：请在模型中心启动一个语音模型")
}

// TTSSpeakStreaming 流式合成：逐句生成。
// 引擎优先级：Herdsman TTS → Edge TTS → WinTTS (SAPI)
func (a *mediaState) TTSSpeakStreaming(text string) error {
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
			for _, model := range []string{"edge-tts", "qwen3-tts-customvoice", "qwen3-tts-voicedesign", "voxcpm2"} {
				voice := a.ttsVoiceForModel(model)
				htts := tts.NewHerdsmanTTS(herdEngine.BaseURL, model, voice)
				engines = append(engines, htts)
				format := "mp3"
				if strings.Contains(strings.ToLower(model), "qwen3") || strings.Contains(strings.ToLower(model), "voxcpm") {
					format = "wav"
				}
				metas = append(metas, struct {
					Label  string
					Format string
				}{"herdsman-" + model, format})
			}
		}
	}

	// 0.5 xAI Grok TTS（若全局语音模型选择了 xAI，流式朗读优先走云端）
	if a.activeTTSEngine == "xai" && a.activeTTSModel == "grok-tts" && a.client != nil && a.engineMgr != nil {
		if xaiEng, ok := a.engineMgr.GetEngine("xai"); ok && xaiEng.Enabled {
			voice := strings.TrimSpace(a.activeTTSVoice)
			if !tts.IsXaiVoice(voice) {
				voice = "eve"
			}
			xtts := tts.NewXaiTTS(xaiEng.BaseURL, voice, a.client.GetToken, nil)
			engines = append(engines, xtts)
			metas = append(metas, struct {
				Label  string
				Format string
			}{"xai", "mpeg"})
		}
	}

	// 1. Edge TTS（免费在线）
	edgeTTS := tts.NewEdgeTTS()
	engines = append(engines, edgeTTS)
	metas = append(metas, struct {
		Label  string
		Format string
	}{"edge", "mp3"})

	// 2. WinTTS SAPI（离线）
	engines = append(engines, tts.NewWinTTS())
	metas = append(metas, struct {
		Label  string
		Format string
	}{"sapi", "wav"})

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
