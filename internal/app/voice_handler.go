// Package app — 语音 API 端点（Wails 绑定）
//
// 对齐 Ackem 的 voiceManager.ts IPC 通道：
//   - VoiceStart / VoiceStop → 启停语音管道
//   - VoicePushAudio → 推送麦克风 PCM 块
//   - VoiceSetMode / VoiceApplySettings → 配置管理
//   - VoiceCancelTTS → 打断
//   - VoiceSetPTTActive → 按键说话
//   - VoiceHealth → 健康检查
package app

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/asr"
	"github.com/gaea/gaea/internal/tts"
	"github.com/gaea/gaea/internal/voice"
)

// ── 语音事件名称常量（对齐 Ackem IPC channel 命名） ──

const (
	VoiceEventState      = "voice:state"
	VoiceEventTranscript = "voice:transcript"
	VoiceEventTTSAudio   = "voice:tts-audio"
	VoiceEventTTSSpeak   = "voice:tts-speak-text"
	VoiceEventTTSCancel  = "voice:tts-speak-cancel"
	VoiceEventListening  = "voice:listening"
	VoiceEventThinking   = "voice:thinking"
)

// ── App 事件发射器适配 ──

// voiceEmitter 实现 voice.EventEmitter，将语音事件桥接到 Wails 前端
type voiceEmitter struct {
	app *App
}

func (e *voiceEmitter) EmitVoiceState(state voice.VoiceState) {
	e.app.emit(VoiceEventState, map[string]interface{}{"state": string(state)})
}

func (e *voiceEmitter) EmitVoiceTranscript(text string, isFinal bool) {
	e.app.emit(VoiceEventTranscript, map[string]interface{}{"text": text, "isFinal": isFinal})
}

func (e *voiceEmitter) EmitVoiceReply(text string) {
	e.app.emit("voice:reply", map[string]interface{}{"text": text})
}

func (e *voiceEmitter) EmitVoiceTTSAudio(audio []byte, mimeType string) {
	e.app.emit(VoiceEventTTSAudio, map[string]interface{}{"audio": audio, "mimeType": mimeType})
}

func (e *voiceEmitter) EmitVoiceTTSSpeakText(text string) {
	e.app.emit(VoiceEventTTSSpeak, map[string]interface{}{"text": text})
}

func (e *voiceEmitter) EmitVoiceTTSCancel() {
	e.app.emit(VoiceEventTTSCancel, map[string]interface{}{})
}

func (e *voiceEmitter) EmitVoiceListening(active bool) {
	e.app.emit(VoiceEventListening, map[string]interface{}{"active": active})
}

func (e *voiceEmitter) EmitVoiceThinking(active bool) {
	e.app.emit(VoiceEventThinking, map[string]interface{}{"active": active})
}

func (e *voiceEmitter) EmitVoiceError(err error) {
	e.app.emit("voice:error", map[string]interface{}{"error": err.Error()})
}

// ── 初始化 ──

// initVoice 初始化语音管理器（在 Startup 中调用）
func (a *mediaState) initVoice() {
	config := voice.DefaultVoiceConfig()
	emitter := &voiceEmitter{app: a.app}
	a.voiceManager = voice.NewManager(emitter, config)

	// 设置 ASR 客户端（模型中心 STT 模型路由）
	a.applyASRClient()

	// 设置 whisper 对话回调（默认轻语人格对话，使用搜索增强版，语音也能上网查）
	a.setWhisperChatFn()

	// 设置 TTS 合成回调（复用现有 TTSSpeakBase64）
	a.voiceManager.SetTTSSynthesizeFn(func(text, voiceDescription string) ([]byte, string, error) {
		return a.synthesizeVoiceTTS(text, voiceDescription)
	})

	slog.Info("语音管理器已初始化")
}

// setWhisperChatFn 设置默认对话回调（轻语人格化对话，搜索增强）
func (a *mediaState) setWhisperChatFn() {
	a.voiceManager.SetWhisperChatFn(func(userMsg, personalityID string) (string, string, error) {
		result, err := a.app.WhisperChatWithSearch(userMsg, personalityID)
		if err != nil {
			return "", "", err
		}
		reply, _ := result["reply"].(string)
		emotion, _ := result["emotion"].(string)
		if emotion == "" {
			emotion = "CALM_RATIONAL"
		}
		return reply, emotion, nil
	})
}

// VoiceSetChatTarget 切换语音对话目标
// target:
//   - "gaea"    → 直接与默认平台 AI 助手 gaea 对话（通用对话，无人格）
//   - "whisper" → 轻语人格化对话（搜索增强，默认）
func (a *mediaState) VoiceSetChatTarget(target string) error {
	if a.voiceManager == nil {
		a.initVoice()
	}
	switch target {
	case "gaea":
		a.voiceManager.SetWhisperChatFn(func(userMsg, _ string) (string, string, error) {
			result, err := a.app.ChatGeneral(userMsg)
			if err != nil {
				return "", "", err
			}
			reply, _ := result["reply"].(string)
			if reply == "" {
				return "", "", fmt.Errorf("gaea 对话返回空回复")
			}
			return reply, "CALM_RATIONAL", nil
		})
	case "whisper":
		a.setWhisperChatFn()
	default:
		return fmt.Errorf("未知对话目标: %s", target)
	}
	slog.Info("语音对话目标已切换", "target", target)
	return nil
}

// applyASRClient 配置 ASR 客户端（模型中心 STT 模型引擎路由）
// 优先级：用户选中的 STT 模型（模型中心）→ 扫描各引擎 STT 模型 → 默认 herdsman whisper-base
func (a *mediaState) applyASRClient() {
	if a.voiceManager == nil || a.engineMgr == nil {
		return
	}

	// 1. 用户选中的 ASR 模型（模型中心）
	if a.activeASREngine != "" && a.activeASRModel != "" {
		if eng, ok := a.engineMgr.GetEngine(a.activeASREngine); ok && eng.Enabled {
			a.voiceManager.SetASRClient(asr.NewHerdsmanASR(eng.BaseURL, a.activeASRModel))
			slog.Info("ASR 客户端已配置（用户选择）", "engine", a.activeASREngine, "model", a.activeASRModel)
			return
		}
		slog.Warn("用户选中的 ASR 引擎不可用，自动扫描", "engine", a.activeASREngine)
	}

	// 2. 扫描所有引擎的 STT 模型
	for _, eid := range []string{"herdsman", "ollama", "xai", "deepseek"} {
		eng, ok := a.engineMgr.GetEngine(eid)
		if !ok || !eng.Enabled {
			continue
		}
		for _, m := range eng.Models {
			if isSTTModel(m.ID) {
				a.voiceManager.SetASRClient(asr.NewHerdsmanASR(eng.BaseURL, m.ID))
				slog.Info("ASR 客户端已配置（自动扫描）", "engine", eid, "model", m.ID)
				return
			}
		}
	}

	// 3. 默认 herdsman whisper-base
	if eng, ok := a.engineMgr.GetEngine("herdsman"); ok && eng.Enabled {
		a.voiceManager.SetASRClient(asr.NewHerdsmanASR(eng.BaseURL, "whisper-base"))
		slog.Info("ASR 客户端已配置（默认）", "baseURL", eng.BaseURL)
	}
}

// isSTTModel 判断模型 ID 是否为语音识别模型（与模型中心 classifyModel 的 stt 分类一致）
func isSTTModel(id string) bool {
	l := strings.ToLower(id)
	return strings.Contains(l, "whisper") || strings.Contains(l, "sherpa") ||
		strings.Contains(l, "zipformer") || strings.Contains(l, "asr") ||
		strings.Contains(l, "funasr")
}

// synthesizeVoiceTTS 语音管道专用的 TTS 合成（带情感参数）
// 优先级：用户选中的 TTS 模型（模型中心，voicedesign 注入情感描述）→ TTSSpeakBase64 统一路由
func (a *mediaState) synthesizeVoiceTTS(text, voiceDescription string) ([]byte, string, error) {
	// 1. 用户选中的 TTS 模型（模型中心）优先
	if a.activeTTSModel != "" && a.engineMgr != nil {
		if eng, ok := a.engineMgr.GetEngine(a.activeTTSEngine); ok && eng.Enabled {
			var htts *tts.HerdsmanTTS
			if strings.Contains(strings.ToLower(a.activeTTSModel), "voicedesign") {
				htts = tts.NewHerdsmanTTSWithDesc(eng.BaseURL, a.activeTTSModel, voiceDescription)
			} else {
				htts = tts.NewHerdsmanTTS(eng.BaseURL, a.activeTTSModel, "Cherry")
			}
			if audio, err := htts.Synthesize(text); err == nil && len(audio) > 0 {
				return audio, "audio/mp3", nil
			}
			slog.Debug("用户选中 TTS 模型失败，回退自动路由", "engine", a.activeTTSEngine, "model", a.activeTTSModel)
		}
	}

	// 2. 统一路由（TTSSpeakBase64：扫描引擎 TTS 模型 → Edge → SAPI）
	return a.synthesizeFromBase64(text)
}

// synthesizeFromBase64 通过 TTSSpeakBase64 路由合成并解码
func (a *mediaState) synthesizeFromBase64(text string) ([]byte, string, error) {
	result, err := a.TTSSpeakBase64(text)
	if err != nil {
		return nil, "", err
	}
	b64, _ := result["base64"].(string)
	mime, _ := result["mimeType"].(string)
	if b64 == "" {
		return nil, "", fmt.Errorf("TTS 返回空音频")
	}
	if mime == "" {
		mime = "audio/mp3"
	}
	audio, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, "", fmt.Errorf("TTS base64 解码失败: %w", err)
	}
	return audio, mime, nil
}

// ── Wails API 端点 ──

// VoiceStart 启动语音管道
func (a *mediaState) VoiceStart() error {
	if a.voiceManager == nil {
		a.initVoice()
	}
	return a.voiceManager.Start()
}

// VoiceStop 停止语音管道
func (a *mediaState) VoiceStop() error {
	if a.voiceManager == nil {
		return nil
	}
	a.voiceManager.Stop()
	return nil
}

// VoicePushAudio 推送麦克风 PCM 音频块（16kHz/16bit/mono）
// 对齐 Ackem voice:audio-chunk IPC
func (a *mediaState) VoicePushAudio(chunk []byte) error {
	if a.voiceManager == nil {
		return fmt.Errorf("语音管理器未初始化")
	}
	return a.voiceManager.PushAudioChunk(chunk)
}

// VoiceSetMode 设置语音输入模式
// mode: "vad" | "ptt" | "off"
func (a *mediaState) VoiceSetMode(mode string) error {
	if a.voiceManager == nil {
		a.initVoice()
	}
	config := a.voiceManager.GetConfig()
	switch mode {
	case "vad":
		config.VoiceMode = voice.VoiceModeVAD
	case "ptt":
		config.VoiceMode = voice.VoiceModePTT
	case "off":
		config.VoiceMode = voice.VoiceModeOff
	default:
		return fmt.Errorf("未知模式: %s", mode)
	}
	a.voiceManager.ApplyConfig(config)
	return nil
}

// VoiceSetInputChannel 设置输入通道
// channel: "dual" | "voice-only" | "text-only"
func (a *mediaState) VoiceSetInputChannel(channel string) error {
	if a.voiceManager == nil {
		a.initVoice()
	}
	config := a.voiceManager.GetConfig()
	switch channel {
	case "dual":
		config.InputChannel = voice.InputDual
	case "voice-only":
		config.InputChannel = voice.InputVoiceOnly
	case "text-only":
		config.InputChannel = voice.InputTextOnly
	default:
		return fmt.Errorf("未知通道: %s", channel)
	}
	a.voiceManager.ApplyConfig(config)
	return nil
}

// VoiceApplySettings 应用语音设置（对齐 Ackem voice:apply-settings）
func (a *mediaState) VoiceApplySettings(settings map[string]interface{}) error {
	if a.voiceManager == nil {
		a.initVoice()
	}
	config := a.voiceManager.GetConfig()

	if v, ok := settings["enabled"].(bool); ok {
		config.Enabled = v
	}
	if v, ok := settings["ttsEnabled"].(bool); ok {
		config.TTSEnabled = v
	}
	if v, ok := settings["voiceMode"].(string); ok {
		config.VoiceMode = voice.VoiceMode(v)
	}
	if v, ok := settings["inputChannel"].(string); ok {
		config.InputChannel = voice.InputChannel(v)
	}
	if v, ok := settings["ttsVoice"].(string); ok {
		config.TTSVoice = v
	}
	if v, ok := settings["ttsEngine"].(string); ok {
		config.TTSEngine = voice.TTSEngine(v)
	}
	if v, ok := settings["asrModel"].(string); ok {
		config.ASRModel = voice.ASRModel(v)
		// 动态切换 ASR 模型
		if a.voiceManager != nil {
			eng, engOk := a.engineMgr.GetEngine("herdsman")
			if engOk && eng.Enabled {
				a.voiceManager.SetASRClient(asr.NewHerdsmanASR(eng.BaseURL, v))
			}
		}
	}
	if v, ok := settings["interruptThresholdMs"].(float64); ok {
		config.InterruptThresholdMs = int(v)
	}
	if v, ok := settings["silenceThresholdMs"].(float64); ok {
		config.SilenceThresholdMs = int(v)
	}
	if v, ok := settings["personalityPresetId"].(string); ok {
		config.PersonalityPresetID = v
	}

	a.voiceManager.ApplyConfig(config)
	return nil
}

// VoiceGetSettings 获取当前语音设置
func (a *mediaState) VoiceGetSettings() map[string]interface{} {
	if a.voiceManager == nil {
		config := voice.DefaultVoiceConfig()
		return configToMap(&config)
	}
	config := a.voiceManager.GetConfig()
	return configToMap(&config)
}

// VoiceCancelTTS 打断当前 TTS 播放（对齐 Ackem voice:cancel-tts）
func (a *mediaState) VoiceCancelTTS() error {
	if a.voiceManager == nil {
		return nil
	}
	a.voiceManager.CancelTTS()
	return nil
}

// VoiceSetPTTActive PTT 按键按下/释放（对齐 Ackem voice:ptt-active）
func (a *mediaState) VoiceSetPTTActive(active bool) error {
	if a.voiceManager == nil {
		return fmt.Errorf("语音管理器未初始化")
	}
	a.voiceManager.SetPTTActive(active)
	return nil
}

// VoiceHealth 健康检查（对齐 Ackem voice:health）
func (a *mediaState) VoiceHealth() map[string]interface{} {
	if a.voiceManager == nil {
		return map[string]interface{}{
			"asrReady": false,
			"ttsReady": false,
			"state":    "idle",
		}
	}
	return a.voiceManager.HealthCheck()
}

// VoiceRestartService 重启语音服务（重新检测 ASR/TTS 可用性）
func (a *mediaState) VoiceRestartService() error {
	if a.voiceManager != nil {
		a.voiceManager.Stop()
	}
	a.initVoice()
	return nil
}

// VoiceGetState 获取当前语音状态
func (a *mediaState) VoiceGetState() map[string]interface{} {
	if a.voiceManager == nil {
		return map[string]interface{}{
			"active":    false,
			"listening": false,
			"speaking":  false,
			"thinking":  false,
		}
	}
	s := a.voiceManager.GetState()
	return map[string]interface{}{
		"active":    s != voice.StateIdle,
		"listening": s == voice.StateListening,
		"speaking":  s == voice.StateSpeaking,
		"thinking":  s == voice.StateThinking,
		"state":     string(s),
	}
}

// ── 工具函数 ──

func configToMap(c *voice.VoiceRuntimeConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled":              c.Enabled,
		"ttsEnabled":           c.TTSEnabled,
		"asrModel":             string(c.ASRModel),
		"ttsEngine":            string(c.TTSEngine),
		"ttsVoice":             c.TTSVoice,
		"ttsHerdsmanModel":     c.TTSHerdsmanModel,
		"voiceMode":            string(c.VoiceMode),
		"interruptThresholdMs": c.InterruptThresholdMs,
		"silenceThresholdMs":   c.SilenceThresholdMs,
		"inputChannel":         string(c.InputChannel),
		"personalityPresetId":  c.PersonalityPresetID,
	}
}
