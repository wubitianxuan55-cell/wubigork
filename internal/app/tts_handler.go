package app

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/tts"
	"github.com/gaea/gaea/internal/voice"
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

// TTSSpeakBase64 合成语音并返回 Base64 音频。
// 引擎优先级（注册表驱动，见 ttsProviderPipeline）：用户选中的 TTS 模型 →
// 扫描各引擎 TTS 模型 → Edge TTS → WinTTS (SAPI)。新增引擎只需注册，代码零改动。
func (a *mediaState) TTSSpeakBase64(text string) (map[string]interface{}, error) {
	audio, mime, err := a.speakFromTTSProviderSteps(text, a.ttsProviderPipeline(true))
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"base64": base64.StdEncoding.EncodeToString(audio), "mimeType": mime}, nil
}

// TTSSpeakStreaming 流式合成：逐句生成（边合成边播）。
// 合成器列表由注册表驱动（见 ttsStreamingProviders）：
// Herdsman TTS → xAI Grok TTS → Edge TTS → WinTTS (SAPI)。
func (a *mediaState) TTSSpeakStreaming(text string) error {
	sentences := tts.SplitSentences(text)
	if len(sentences) == 0 {
		return fmt.Errorf("无可朗读的文本")
	}

	providers := a.ttsStreamingProviders()
	if len(providers) == 0 {
		return fmt.Errorf("无可用的语音引擎")
	}
	chain := tts.NewTTSChain(providers...)

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

		var activeProvider tts.TTSProvider
		var activeMime string
		var activeName string

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

			if activeProvider != nil {
				audio, _, err = activeProvider.SynthesizeWithMime(sentence)
			}

			if activeProvider == nil || err != nil || len(audio) == 0 {
				if err != nil {
					slog.Warn("TTS 引擎失败，重新探测", "sentence", i, "error", err)
				}
				var name string
				audio, activeMime, name, err = chain.Synthesize(sentence)
				if err != nil {
					slog.Error("所有 TTS 引擎均失败", "error", err)
					a.emit("tts-stream", map[string]interface{}{
						"type": "error", "error": "无可用的语音引擎，请检查网络或 TTS 配置",
					})
					return
				}
				activeName = name
				activeProvider = chain.ProviderByName(name)
				slog.Info("TTS 引擎已选择", "engine", activeName, "format", activeMime)
			}

			done := i == len(sentences)-1
			mime := activeMime
			if mime == "" {
				mime = "audio/mp3"
			}
			a.emit("tts-stream", map[string]interface{}{
				"type":     "chunk",
				"index":    i,
				"total":    len(sentences),
				"audio":    base64.StdEncoding.EncodeToString(audio),
				"mimeType": mime,
				"engine":   activeName,
				"done":     done,
			})
		}

		a.emit("tts-stream", map[string]interface{}{"type": "done", "engine": activeName})
	}()

	return nil
}

// ── TTS 提供者注册表驱动（Step 3c seam 消费者） ─────────────────────────────

// ttsPipelineStep 一步 TTS 回退：提供者 + 引擎 id（cosyvoice 步骤在合成前惰性
// ensure 本地服务，避免链前面成功时仍拉起本地服务；其余引擎无需 ensure）。
type ttsPipelineStep struct {
	provider tts.TTSProvider
	engineID string // 空 = 无需 ensure
}

// ttsProviderPipeline 构建注册表驱动的 TTS 回退链（与 TTSSpeakBase64 优先级一致）：
// 用户选中模型 → 扫描引擎 TTS 模型 → Edge → SAPI。新增引擎只改注册表，代码零改动。
func (a *mediaState) ttsProviderPipeline(includeScan bool) []ttsPipelineStep {
	var steps []ttsPipelineStep

	// 1. 用户选中的 TTS 模型（模型中心）
	if a.activeTTSModel != "" && a.engineMgr != nil {
		if p, ok := a.ttsProviderForEngine(a.activeTTSEngine, a.activeTTSModel, ""); ok {
			steps = append(steps, ttsPipelineStep{provider: p, engineID: a.activeTTSEngine})
		}
	}

	// 2. 扫描所有引擎的 TTS 模型列表
	if includeScan && a.engineMgr != nil {
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
				if !isTTSModelID(id) {
					continue
				}
				if p, ok := a.ttsProviderForEngine(eid, m.ID, ""); ok {
					steps = append(steps, ttsPipelineStep{provider: p, engineID: eid})
				}
			}
		}
	}

	// 3. Edge TTS（在线，免费）
	if p, err := tts.NewTTSProvider("edge", tts.TTSConfig{}); err == nil {
		steps = append(steps, ttsPipelineStep{provider: p})
	}
	// 4. Windows SAPI（本地系统自带）
	if p, err := tts.NewTTSProvider("sapi", tts.TTSConfig{}); err == nil {
		steps = append(steps, ttsPipelineStep{provider: p})
	}
	return steps
}

// speakFromTTSProviderSteps 依次尝试各步骤合成，返回第一个成功的 (音频, MIME)。
// cosyvoice 步骤首次合成前惰性 ensure 本地服务（幂等，已就绪零开销）。
// 参数版本委托 speakFromTTSProviderStepsParams（零值参数 = 引擎默认）。
func (a *mediaState) speakFromTTSProviderSteps(text string, steps []ttsPipelineStep) ([]byte, string, error) {
	return a.speakFromTTSProviderStepsParams(text, steps, tts.TTSParams{})
}

// speakFromTTSProviderStepsParams 带合成参数（v4.3d：speed/style/emotion 透传
// 各引擎，edge SSML 参数化），失败依次回退下一提供者。
func (a *mediaState) speakFromTTSProviderStepsParams(text string, steps []ttsPipelineStep, params tts.TTSParams) ([]byte, string, error) {
	for _, st := range steps {
		if st.engineID == "cosyvoice" {
			a.ensureLocalTTSService(st.engineID)
		}
		audio, mime, err := st.provider.SynthesizeWithParams(text, params)
		if err != nil || len(audio) == 0 {
			slog.Debug("TTS 引擎失败，尝试下一个", "engine", st.provider.Name(), "error", err)
			continue
		}
		if mime == "" {
			mime = "audio/mp3"
		}
		return audio, mime, nil
	}
	return nil, "", fmt.Errorf("无可用的 TTS 模型：请在模型中心启动一个语音模型")
}

// TTSSpeakBase64WithParams 带参数合成语音并返回 Base64 音频（v4.3d）：
// params 为 TTS 风格/情绪/语速参数（零值 = 引擎默认，行为同 TTSSpeakBase64）。
func (a *mediaState) TTSSpeakBase64WithParams(text string, params tts.TTSParams) (map[string]interface{}, error) {
	audio, mime, err := a.speakFromTTSProviderStepsParams(text, a.ttsProviderPipeline(true), params)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"base64": base64.StdEncoding.EncodeToString(audio), "mimeType": mime}, nil
}

// GaeaTTSVoiceParams 返回情绪标签对应的结构化 TTS 参数（v4.3d，shared）：
// 前端预览/调试与语音管道共用；未知情绪返回零值（引擎默认）。
func (a *App) GaeaTTSVoiceParams(emotion string) tts.TTSParams {
	return voice.GetEmotionVoiceParams(emotion)
}

// ttsStreamingProviders 构建流式 TTS 的合成器链（注册表驱动）：
// Herdsman TTS（本地优先）→ xAI Grok TTS（选中时）→ Edge TTS → WinTTS SAPI。
func (a *mediaState) ttsStreamingProviders() []tts.TTSProvider {
	var providers []tts.TTSProvider

	// 0. Herdsman TTS（本地优先）
	if a.engineMgr != nil {
		if herdEngine, ok := a.engineMgr.GetEngine("herdsman"); ok && herdEngine.Enabled {
			for _, model := range []string{"edge-tts", "qwen3-tts-customvoice", "qwen3-tts-voicedesign", "voxcpm2"} {
				if p, err := tts.NewTTSProvider("herdsman", tts.TTSConfig{
					BaseURL: herdEngine.BaseURL,
					Model:   model,
					Voice:   a.ttsVoiceForModel(model),
				}); err == nil {
					providers = append(providers, p)
				}
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
			if p, err := tts.NewTTSProvider("xai", tts.TTSConfig{
				BaseURL: xaiEng.BaseURL, Voice: voice, GetToken: a.client.GetToken,
			}); err == nil {
				providers = append(providers, p)
			}
		}
	}

	// 1. Edge TTS（免费在线）
	if p, err := tts.NewTTSProvider("edge", tts.TTSConfig{}); err == nil {
		providers = append(providers, p)
	}
	// 2. WinTTS SAPI（离线）
	if p, err := tts.NewTTSProvider("sapi", tts.TTSConfig{}); err == nil {
		providers = append(providers, p)
	}
	return providers
}

// isTTSModelID 判断模型 ID 是否为 TTS 合成模型（与模型中心 tts 分类口径一致；
// modelengine.classifyModelKind 未导出，保留本地判断，见轨道报告）。
func isTTSModelID(id string) bool {
	return strings.Contains(id, "tts") || strings.Contains(id, "voice") ||
		strings.Contains(id, "speech") || strings.Contains(id, "voxcpm")
}
