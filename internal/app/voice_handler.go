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
	"os"
	"strings"

	"github.com/gaea/gaea/internal/asr"
	appconfig "github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/secure"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/realtime"
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
	if v := strings.TrimSpace(a.activeTTSVoice); v != "" {
		config.TTSVoice = v
	}
	if v := strings.TrimSpace(a.activePersonalityID); v != "" {
		config.PersonalityPresetID = v
	}
	// S1 Realtime 配置接线（兑现 S0 注释）：从 ~/.gaea_config.json 读取
	// realtime_provider / realtime_model / realtime_api_key 三项落盘值。
	// APIKey 存储口径为 secure.EncryptString 密文，这里解出内存明文注入运行时
	// 配置（先例 SetOpencodeZenKey）；解密失败 → 告警 + APIKey 置空（降级不崩，
	// realtimeReady 将为 false）。Provider 为空（未配置）→ 三项零填入，现拼接
	// 管线零变化。须在 NewManager 之前注入，随配置一并入管理器内存态。
	if provider, model, apiKey := a.realtimeRuntimeCfg(); provider != "" {
		config.RealtimeProvider = provider
		config.RealtimeModel = model
		config.RealtimeAPIKey = apiKey
	}
	emitter := &voiceEmitter{app: a.app}
	a.voiceManager = voice.NewManager(emitter, config)

	// 设置 ASR 客户端（模型中心 STT 模型路由）
	a.applyASRClient()

	// 设置 whisper 对话回调（默认轻语人格对话，使用搜索增强版，语音也能上网查）
	a.setWhisperChatFn()

	// 设置 TTS 合成回调（seam 消费者：经 synthesizeVoiceTTS → tryEngineTTS /
	// TTSSpeakBase64 统一路由到 TTS 提供者注册表）
	a.voiceManager.SetTTSSynthesizeFn(func(text, voiceDescription string) ([]byte, string, error) {
		return a.synthesizeVoiceTTS(text, voiceDescription)
	})

	// Realtime 档探测（seam 消费者，见 internal/realtime）：未配置（Provider
	// 空）→ 完全跳过，现拼接管线零变化；配置了但 New(kind) 失败（含解密失败
	// 置空 Key）→ 优雅降级仅告警，不崩、不阻塞现有语音。本刀仍不启动 realtime
	// 会话、不接管任何音频路径。
	if probeRealtime(config) {
		slog.Info("Realtime 档就绪（S1 配置已接线，未接管音频路径）", "kind", strings.TrimSpace(config.RealtimeProvider))
	}

	slog.Info("语音管理器已初始化")
}

// realtimeRuntimeCfg 读取落盘 Realtime 配置并解密 API Key（S1 接线共用：
// initVoice 注入 + VoiceHealth 实时就绪判定，与 initVoice 探测结论一致）。
// realtime_api_key 存储口径 = secure.EncryptString 密文；解密失败 → slog.Warn
// 并返回空 Key（降级不崩，realtimeReady 将为 false）。Provider 为空（未配置）
// → 返回零值，调用方按现状零变化处理。
func (a *mediaState) realtimeRuntimeCfg() (provider, model, apiKey string) {
	if a == nil || a.cfg == nil {
		return "", "", ""
	}
	provider = strings.TrimSpace(a.cfg.GetRealtimeProvider())
	if provider == "" {
		return "", "", ""
	}
	model = strings.TrimSpace(a.cfg.GetRealtimeModel())
	dec, err := secure.DecryptString(a.cfg.GetRealtimeAPIKey())
	if err != nil {
		slog.Warn("Realtime API Key 解密失败，实时语音档降级为未就绪", "error", err)
		return provider, model, ""
	}
	return provider, model, dec
}

// probeRealtime Realtime 档就绪探测（seam 消费者）：未配置（RealtimeProvider 空）
// → 跳过返回 false（现拼接管线零变化）；配置了但注册表构造失败 → slog.Warn
// 优雅降级返回 false（不崩、不阻塞现有语音）。注册表构造不做网络 I/O、结果
// 确定，VoiceHealth.realtimeReady 亦经此实时计算（与 initVoice 探测结论一致）。
// S1：入参三字段由落盘配置 + secure.DecryptString 解密后的明文 Key 填充，
// 见 realtimeRuntimeCfg。
func probeRealtime(cfg voice.VoiceRuntimeConfig) bool {
	kind := strings.TrimSpace(cfg.RealtimeProvider)
	if kind == "" {
		return false
	}
	if _, err := realtime.New(kind, realtime.Config{
		Model:  cfg.RealtimeModel,
		APIKey: cfg.RealtimeAPIKey,
	}); err != nil {
		slog.Warn("Realtime 档不可用，回退现拼接语音管线（S0 优雅降级）", "kind", kind, "error", err)
		return false
	}
	return true
}

// setWhisperChatFn 设置默认对话回调（轻语人格化对话，搜索增强）
func (a *mediaState) setWhisperChatFn() {
	a.voiceManager.SetWhisperChatFn(func(userMsg, personalityID string) (string, string, [4]float64, error) {
		// v4.5 指令中枢（S4.3）：语音文本先过统一意图路由——命中即能力执行，
		// 回复文本经下方同一 TTS 流程播报；未命中走原轻语对话管道。
		if a.app != nil {
			if reply, handled := a.app.routeIntent(userMsg); handled {
				return reply, "CALM_RATIONAL", [4]float64{}, nil
			}
		}
		result, err := a.app.WhisperChatWithSearch(userMsg, personalityID, false, false)
		if err != nil {
			return "", "", [4]float64{}, err
		}
		reply, _ := result["reply"].(string)
		emotion, _ := result["emotion"].(string)
		if emotion == "" {
			emotion = "CALM_RATIONAL"
		}
		// v4.6 Mood→TTS 闭环：透传 whisper 长期心境 4D EWMA（全 0 = 未播种）。
		// 语音路径是进程内直调（Go [4]float64）；Wails 绑定路径经 JSON 往返
		// 是 []float64——两种形态都接。
		var mood [4]float64
		switch raw := result["mood"].(type) {
		case [4]float64:
			mood = raw
		case []float64:
			if len(raw) == 4 {
				copy(mood[:], raw)
			}
		}
		return reply, emotion, mood, nil
	})
}

// applyASRClient 配置 ASR 客户端（模型中心 STT 模型引擎路由）

// applyASRClient 配置 ASR 客户端（模型中心 STT 模型引擎路由）
// 优先级：用户选中的 STT 模型（模型中心）→ 扫描各引擎 STT 模型 → 默认 herdsman whisper-base
func (a *mediaState) applyASRClient() {
	if a.voiceManager == nil || a.engineMgr == nil {
		return
	}

	// 1. 用户选中的 ASR 模型（模型中心）
	if a.activeASREngine != "" && a.activeASRModel != "" {
		if eng, ok := a.engineMgr.GetEngine(a.activeASREngine); ok && eng.Enabled {
			a.voiceManager.SetASRProvider(a.newASRProvider(eng.BaseURL, a.activeASRModel))
			slog.Info("ASR 提供者已配置（用户选择）", "engine", a.activeASREngine, "model", a.activeASRModel)
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
				a.voiceManager.SetASRProvider(a.newASRProvider(eng.BaseURL, m.ID))
				slog.Info("ASR 提供者已配置（自动扫描）", "engine", eid, "model", m.ID)
				return
			}
		}
	}

	// 3. 默认 herdsman whisper-base
	if eng, ok := a.engineMgr.GetEngine("herdsman"); ok && eng.Enabled {
		a.voiceManager.SetASRProvider(a.newASRProvider(eng.BaseURL, "whisper-base"))
		slog.Info("ASR 提供者已配置（默认）", "baseURL", eng.BaseURL)
	}
}

// newASRProvider 经 ASR 注册表构造 herdsman 提供者（seam 消费者；kind 固定
// "herdsman"——当前唯一实现，后续新增引擎只改注册表，调用方零改动）。
func (a *mediaState) newASRProvider(baseURL, model string) asr.ASRProvider {
	p, err := asr.NewASRProvider("herdsman", asr.ASRConfig{BaseURL: baseURL, Model: model})
	if err != nil {
		slog.Warn("ASR 提供者构造失败", "error", err)
		return nil
	}
	return p
}

// isSTTModel 判断模型 ID 是否为语音识别模型。
// 3.0 Step 3c：委托 modelengine.ClassifyModelByName（Track G 已导出；关键词表与
// 历史 isSTTModel 完全一致，避免双源漂移）。
func isSTTModel(id string) bool {
	return modelengine.ClassifyModelByName(id) == "stt"
}

// synthesizeVoiceTTS 语音管道专用的 TTS 合成（带情感参数）
// 优先级：功能绑定「聊天语音」→ 模型中心全局 TTS → TTSSpeakBase64 统一路由
func (a *mediaState) synthesizeVoiceTTS(text, voiceDescription string) ([]byte, string, error) {
	// 1. 功能绑定「聊天语音」优先（模型中心 → 功能绑定）
	if a.chatVoiceEngine != "" && a.chatVoiceModel != "" {
		if audio, mime, ok := a.tryEngineTTS(a.chatVoiceEngine, a.chatVoiceModel, text, voiceDescription); ok {
			return audio, mime, nil
		}
		slog.Debug("聊天语音绑定模型失败，回退全局 TTS", "engine", a.chatVoiceEngine, "model", a.chatVoiceModel)
	}
	// 2. 全局 TTS（模型中心 → 语音模型）
	if a.activeTTSModel != "" {
		if audio, mime, ok := a.tryEngineTTS(a.activeTTSEngine, a.activeTTSModel, text, voiceDescription); ok {
			return audio, mime, nil
		}
		slog.Debug("用户选中 TTS 模型失败，回退自动路由", "engine", a.activeTTSEngine, "model", a.activeTTSModel)
	}

	// 3. 统一路由（TTSSpeakBase64：扫描引擎 TTS 模型 → Edge → SAPI）
	return a.synthesizeFromBase64(text)
}

// tryEngineTTS 用指定引擎+模型尝试 TTS 合成（seam 消费者：统一路由到 TTS 提供者
// 注册表）。xAI 走云端 Grok TTS（"xai" kind），其余引擎走 Herdsman 风格
// /v1/audio/speech（"herdsman" kind）；引擎构造分支（voicedesign/voxcpm/voiceclone）
// 由 herdsman 工厂收敛，这里零分支。
func (a *mediaState) tryEngineTTS(engineID, model, text, voiceDescription string) ([]byte, string, bool) {
	// 本地 TTS 服务兜底：cosyvoice 未就绪时自动拉起（幂等，已就绪零开销）
	if engineID == "cosyvoice" {
		a.ensureLocalTTSService(engineID)
	}
	p, ok := a.ttsProviderForEngine(engineID, model, voiceDescription)
	if !ok {
		return nil, "", false
	}
	audio, mime, err := p.SynthesizeWithMime(text)
	if err != nil || len(audio) == 0 {
		return nil, "", false
	}
	if mime == "" {
		mime = "audio/mp3"
	}
	return audio, mime, true
}

// ttsProviderForEngine 经 TTS 提供者注册表构造指定引擎+模型的提供者：
// xAI → "xai" kind；其余 OpenAI 兼容引擎（herdsman/cosyvoice/ollama/deepseek）→
// "herdsman" kind。引擎未启用/缺少凭据时返回 false（与历史 tryEngineTTS 一致）。
func (a *mediaState) ttsProviderForEngine(engineID, model, voiceDescription string) (tts.TTSProvider, bool) {
	if a.engineMgr == nil {
		return nil, false
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok || !eng.Enabled {
		return nil, false
	}
	if engineID == "xai" {
		if a.client == nil {
			return nil, false
		}
		voice := strings.TrimSpace(a.activeTTSVoice)
		if !tts.IsXaiVoice(voice) {
			voice = "eve"
		}
		p, err := tts.NewTTSProvider("xai", tts.TTSConfig{
			BaseURL: eng.BaseURL, Voice: voice, GetToken: a.client.GetToken,
		})
		if err != nil {
			return nil, false
		}
		return p, true
	}

	// 其余引擎：OpenAI 兼容 /v1/audio/speech（Herdsman 风格）
	cfg := tts.TTSConfig{BaseURL: eng.BaseURL, Model: model, VoiceDescription: voiceDescription}
	l := strings.ToLower(model)
	switch {
	case strings.Contains(l, "voiceclone"):
		// voiceclone 参考音频/文本来自环境变量（与历史 tryEngineTTS 一致）
		cfg.RefAudio = strings.TrimSpace(os.Getenv("HERDSMAN_TTS_REF_AUDIO"))
		cfg.RefText = strings.TrimSpace(os.Getenv("HERDSMAN_TTS_REF_TEXT"))
	case strings.Contains(l, "voicedesign"), strings.Contains(l, "voxcpm"):
		// voicedesign/voxcpm 无描述时用空音色（工厂按模型收敛；历史行为一致）
	default:
		cfg.Voice = a.ttsVoiceForModel(model)
	}
	p, err := tts.NewTTSProvider("herdsman", cfg)
	if err != nil {
		return nil, false
	}
	return p, true
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

// VoiceStart 启动语音管道。browserASR 为 true 时使用浏览器端
// Web Speech API 识别（无需 herdsman STT 模型），后端只负责对话与 TTS。
func (a *mediaState) VoiceStart(browserASR bool) error {
	if a.voiceManager == nil {
		a.initVoice()
	}
	if !browserASR {
		if a.voiceManager == nil || !a.voiceManager.ASRReady() {
			return fmt.Errorf("语音识别未就绪：请在模型中心启用 STT 模型（如 whisper-base / funasr）并确认 herdsman 引擎可用，或改用浏览器端语音识别")
		}
	}
	if !a.voiceManager.WhisperReady() {
		return fmt.Errorf("语音对话引擎未就绪，请检查模型中心配置")
	}
	return a.voiceManager.Start()
}

// VoiceChatText 浏览器端识别文本直接进入对话管道（跳过 herdsman ASR）
func (a *mediaState) VoiceChatText(text string) error {
	if a.voiceManager == nil {
		a.initVoice()
	}
	if a.voiceManager == nil {
		return fmt.Errorf("语音管理器未初始化")
	}
	a.voiceManager.HandleUserText(text)
	return nil
}

// VoicePlaybackDone 前端 TTS 播放完成回调，释放后端状态机继续监听
func (a *mediaState) VoicePlaybackDone() error {
	if a.voiceManager == nil {
		return nil
	}
	a.voiceManager.PlaybackDone()
	return nil
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
		a.activeTTSVoice = v
		if err := appconfig.Save(appconfig.KeyTTSVoice, v); err != nil {
			slog.Warn("保存 TTS 音色配置失败", "error", err)
		}
	}
	// 实时语音 Realtime 档（v4.8.1 S1）：provider/model 明文落盘；APIKey 明文进
	// 内存（本进程实时用）、DPAPI 密文落盘（仅本机可解）。保存失败返回错误——
	// 静默丢 key 会让「已配置」假象带到下次启动（这里不复用 TTSVoice 的仅告警
	// 先例，key 属凭据，丢=配置作废）。
	if v, ok := settings["realtimeProvider"].(string); ok {
		if err := appconfig.Save(appconfig.KeyRealtimeProvider, v); err != nil {
			return fmt.Errorf("保存实时语音供应商失败: %w", err)
		}
		config.RealtimeProvider = v
	}
	if v, ok := settings["realtimeModel"].(string); ok {
		if err := appconfig.Save(appconfig.KeyRealtimeModel, v); err != nil {
			return fmt.Errorf("保存实时语音模型失败: %w", err)
		}
		config.RealtimeModel = v
	}
	if v, ok := settings["realtimeAPIKey"].(string); ok {
		key := strings.TrimSpace(v)
		cipher := ""
		if key != "" {
			c, err := secure.EncryptString(key)
			if err != nil {
				return fmt.Errorf("加密实时语音 Key 失败: %w", err)
			}
			cipher = c
		}
		if err := appconfig.Save(appconfig.KeyRealtimeAPIKey, cipher); err != nil {
			return fmt.Errorf("保存实时语音 Key 失败: %w", err)
		}
		config.RealtimeAPIKey = key
	}
	if v, ok := settings["ttsEngine"].(string); ok {
		config.TTSEngine = voice.TTSEngine(v)
	}
	if v, ok := settings["asrModel"].(string); ok {
		config.ASRModel = voice.ASRModel(v)
		// 动态切换 ASR 模型（经注册表构造 herdsman 提供者）
		if a.voiceManager != nil {
			eng, engOk := a.engineMgr.GetEngine("herdsman")
			if engOk && eng.Enabled {
				a.voiceManager.SetASRProvider(a.newASRProvider(eng.BaseURL, v))
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
		a.activePersonalityID = v
		if err := appconfig.Save(appconfig.KeyVoicePersonality, v); err != nil {
			slog.Warn("保存语音角色配置失败", "error", err)
		}
	}

	a.voiceManager.ApplyConfig(config)
	return nil
}

// VoiceGetSettings 获取当前语音设置（H0-3：TTSHerdsmanModel 动态解析为已装模型）
func (a *mediaState) VoiceGetSettings() map[string]interface{} {
	if a.voiceManager == nil {
		config := voice.DefaultVoiceConfig()
		return a.voiceSettingsMap(&config)
	}
	config := a.voiceManager.GetConfig()
	return a.voiceSettingsMap(&config)
}

// voiceSettingsMap 构建语音设置映射；TTSHerdsmanModel 经 ResolveHerdsmanTTSModel
// 动态解析（配置值优先，未安装则按优先级选已装模型；拿不到已装列表时等价于原逻辑）。
func (a *mediaState) voiceSettingsMap(c *voice.VoiceRuntimeConfig) map[string]interface{} {
	resolved, usedFallback, resolvedFromInstalled := voice.ResolveHerdsmanTTSModel(c.TTSHerdsmanModel, a.herdsmanInstalledModelIDs())
	if usedFallback || resolvedFromInstalled {
		slog.Info("TTS 默认模型动态解析",
			"configured", c.TTSHerdsmanModel,
			"resolved", resolved,
			"usedFallback", usedFallback,
			"resolvedFromInstalled", resolvedFromInstalled,
		)
	}
	c.TTSHerdsmanModel = resolved
	m := configToMap(c)
	// 前端提示：解析是否走了回退 / 结果是否来自已装列表
	m["ttsHerdsmanModelFallback"] = usedFallback
	m["ttsHerdsmanModelFromInstalled"] = resolvedFromInstalled
	// 实时语音档（S1）：只回 provider/model 与 hasKey 布尔——明文 Key 永不出后端。
	m["realtimeProvider"] = c.RealtimeProvider
	m["realtimeModel"] = c.RealtimeModel
	m["realtimeHasKey"] = c.RealtimeAPIKey != ""
	return m
}

// herdsmanInstalledModelIDs 返回 Herdsman 引擎当前已安装（可用）模型 ID 列表。
// 引擎未初始化/未启用/无模型列表时返回 nil（调用方按「无已装信息」处理，行为等价于原逻辑）。
func (a *mediaState) herdsmanInstalledModelIDs() []string {
	if a.engineMgr == nil {
		return nil
	}
	eng, ok := a.engineMgr.GetEngine("herdsman")
	if !ok || !eng.Enabled {
		return nil
	}
	if len(eng.Models) == 0 {
		return nil
	}
	ids := make([]string, 0, len(eng.Models))
	for _, m := range eng.Models {
		if strings.TrimSpace(m.ID) != "" {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
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
			"asrReady":      false,
			"ttsReady":      false,
			"realtimeReady": false,
			"state":         "idle",
		}
	}
	hc := a.voiceManager.HealthCheck()
	// S1 Realtime 档：realtimeReady = 配置了 provider 且 seam 构造成功（经注册
	// 表实时探测，构造无网络 I/O；未配置恒为 false）。APIKey 以落盘密文解密后
	// 注入，与 initVoice 探测结论一致。
	provider, model, apiKey := a.realtimeRuntimeCfg()
	hc["realtimeReady"] = probeRealtime(voice.VoiceRuntimeConfig{
		RealtimeProvider: provider,
		RealtimeModel:    model,
		RealtimeAPIKey:   apiKey,
	})
	return hc
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
