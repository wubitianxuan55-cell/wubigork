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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/asr"
	"github.com/gaea/gaea/internal/characterlib"
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
	VoiceEventReplyDelta = "voice:reply-delta"
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

func (e *voiceEmitter) EmitVoiceReplyDelta(text string) {
	e.app.emit(VoiceEventReplyDelta, map[string]interface{}{"text": text})
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
	if v := strings.TrimSpace(a.currentVoicePersonality()); v != "" {
		config.PersonalityPresetID = v
	}
	// Voice V2 档：落盘的后端选择（空 = v1 缺省）
	if a.cfg != nil {
		if v := strings.TrimSpace(a.cfg.GetVoiceBackend()); v != "" {
			config.VoiceBackend = v
		}
	}
	// S1 沿用：从 ~/.gaea_config.json 读取 realtime_provider / realtime_model /
	// realtime_api_key 三项落盘值注入运行时配置（APIKey 存储口径为
	// secure.EncryptString 密文，这里解出内存明文；解密失败 → 告警 + 置空，
	// realtimeReady 将为 false）。Provider 为空（未配置）→ 三项零填入，现拼接
	// 管线零变化。会话构造与注入在 NewManager 之后进行（见下方 S2 块）。
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
	a.voiceManager.SetTTSStreamSynthesizeFn(func(
		ctx context.Context,
		text, voiceDescription string,
		onAudio func([]byte, string) error,
	) error {
		return a.streamVoiceTTS(ctx, text, voiceDescription, onAudio)
	})

	// S2 Realtime：provider 配置时经注册表构造真实会话注入 Manager（替换 S1
	// 纯探测——探测只证明"可构造"，注入后 VoiceStart 时才拨号激活端到端事件
	// 环）。构造失败（含解密失败置空 Key）→ 优雅降级仅告警，不崩、不阻塞现有
	// 语音，拼接管线零变化；未配置（Provider 空）→ 完全跳过。
	if config.RealtimeProvider != "" {
		if sess, err := realtime.New(config.RealtimeProvider, realtime.Config{
			Model:  config.RealtimeModel,
			APIKey: config.RealtimeAPIKey,
		}); err != nil {
			slog.Warn("Realtime 会话构造失败，回退拼接语音管线", "kind", config.RealtimeProvider, "error", err)
		} else {
			a.voiceManager.SetRealtimeSession(sess)
			slog.Info("Realtime 会话已注入（S2：VoiceStart 时拨号激活端到端模式）", "kind", config.RealtimeProvider, "model", config.RealtimeModel)
		}
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
// 确定，VoiceHealth.realtimeReady 经此实时计算。S2 起 initVoice 不再用它做
// 探测（改为构造真实会话注入 Manager），本函数保留供 VoiceHealth 实时就绪
// 判定。入参三字段由落盘配置 + secure.DecryptString 解密后的明文 Key 填充，
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

	// 低延迟语音链：模型正文边生成边交给 VoiceManager，首句完成即进入 TTS。
	// 阻塞回调保留为兼容路径与旧测试 seam。
	a.voiceManager.SetWhisperChatStreamFn(func(
		ctx context.Context,
		userMsg, personalityID string,
		onDelta func(string),
	) (string, string, [4]float64, error) {
		if a.app != nil {
			if reply, handled := a.app.routeIntent(userMsg); handled {
				if onDelta != nil {
					onDelta(reply)
				}
				return reply, "CALM_RATIONAL", [4]float64{}, nil
			}
		}
		result, err := a.app.whisperChatStream(ctx, userMsg, personalityID, true, false, false, true, onDelta, nil)
		if err != nil {
			return "", "", [4]float64{}, err
		}
		reply, _ := result["reply"].(string)
		emotion, _ := result["emotion"].(string)
		if emotion == "" {
			emotion = "CALM_RATIONAL"
		}
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
	// 本地 TTS 服务兜底：cosyvoice/breeze 未就绪时有界等就绪（幂等，已就绪零开销）
	if isLocalTTSAssetEngine(engineID) {
		a.waitLocalTTSReady(engineID)
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
		if a.clientRef() == nil {
			return nil, false
		}
		voice := strings.TrimSpace(a.activeTTSVoice)
		if !tts.IsXaiVoice(voice) {
			voice = "eve"
		}
		p, err := tts.NewTTSProvider("xai", tts.TTSConfig{
			BaseURL: eng.BaseURL, Voice: voice, GetToken: a.clientRef().GetToken,
		})
		if err != nil {
			return nil, false
		}
		return p, true
	}

	// Breeze：OpenAI 兼容 JSON 路径但字段口径不同（instructions/voice，无
	// /v1/audio/info 音色探测），走 breeze kind。音色/描述解析单一来源在
	// breezeVoiceAndInstruction（流播路径共用同一优先级链）。
	if engineID == "breeze" {
		voice, desc := a.breezeVoiceAndInstruction(voiceDescription)
		p, err := tts.NewTTSProvider("breeze", tts.TTSConfig{
			BaseURL: eng.BaseURL, Model: model,
			Voice:            voice,
			VoiceDescription: desc,
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

// ttsProviderForEngineWithVoice 试听专用 provider 构造（TTSVoicePreviewBase64
// 消费）：音色/描述显式入参，不读当前人格预设与全局 breeze 描述。仅覆盖
// breeze 与 herdsman voicedesign/voxcpm 两种能表达预设的形态
// （voicePreviewCapable 已先行过滤，这里不再重复判定）。
func (a *mediaState) ttsProviderForEngineWithVoice(engineID, model, voiceID, voiceDescription string) (tts.TTSProvider, bool) {
	if a.engineMgr == nil {
		return nil, false
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok || !eng.Enabled {
		return nil, false
	}
	if engineID == "breeze" {
		p, err := tts.NewTTSProvider("breeze", tts.TTSConfig{
			BaseURL: eng.BaseURL, Model: model,
			Voice:            voiceID,
			VoiceDescription: voiceDescription,
		})
		if err != nil {
			return nil, false
		}
		return p, true
	}
	// herdsman voicedesign/voxcpm：描述即音色（无 Voice 字段，工厂按模型收敛）
	p, err := tts.NewTTSProvider("herdsman", tts.TTSConfig{
		BaseURL: eng.BaseURL, Model: model, VoiceDescription: voiceDescription,
	})
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
// S2：realtime 会话在位时以 realtimeReady 为门——输入转写与对话回复都由
// 服务端完成（事件泵走 response 事件，不经 whisperChatFn/本地 ASR），因此
// ASRReady 与 WhisperReady 双门都跳过；会话不在位时维持原双门（ASRReady +
// WhisperReady，拼接管线行为逐字节不变）。
func (a *mediaState) VoiceStart(browserASR bool) error {
	if a.voiceManager == nil {
		a.initVoice()
	}
	if a.voiceManager == nil {
		return fmt.Errorf("语音管理器未初始化")
	}
	realtimeMode := a.voiceManager.RealtimeReady()
	if !browserASR && !realtimeMode {
		if !a.voiceManager.ASRReady() {
			return fmt.Errorf("语音识别未就绪：请在模型中心启用 STT 模型（如 whisper-base / funasr-nano）并确认 herdsman 引擎可用，或改用浏览器端语音识别")
		}
	}
	// 端到端实时模式无需 whisper 对话回调（服务端经 response 事件回话）；
	// 拼接管线才依赖它。浏览器端识别（browserASR）同样依赖 whisper 对话回调
	// 生成回复，故非 realtime 时门保留。
	if !realtimeMode && !a.voiceManager.WhisperReady() {
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
	// P0a-A3（众神殿定稿）：人格双读者（V2 网关 activePersonalityID vs V1
	// config.PersonalityPresetID）必须同点生效。setVoicePersonality 原在本
	// 分支内提前调用，其后 breezeVoiceDesc/personaVoicePresets 任一失败
	// return 都会形成「V2 已新、V1 仍旧」的半写；先暂存，移到 ApplyConfig
	// 紧前统一生效，任何错误路径两个读者都不变。
	var pendingPersonality string
	havePendingPersonality := false
	if v, ok := settings["personalityPresetId"].(string); ok {
		config.PersonalityPresetID = v
		pendingPersonality = v
		havePendingPersonality = true
		if err := appconfig.Save(appconfig.KeyVoicePersonality, v); err != nil {
			slog.Warn("保存语音角色配置失败", "error", err)
		}
	}
	if v, ok := settings["voiceBackend"].(string); ok {
		previousBackend := config.VoiceBackend
		config.VoiceBackend = v
		config.Validate()
		// feature flag 落盘（KeyVoicePersonality 同款仅告警先例：非关键路径）
		if err := appconfig.Save(appconfig.KeyVoiceBackend, config.VoiceBackend); err != nil {
			slog.Warn("保存语音后端配置失败", "error", err)
		}
		if a.cfg != nil {
			a.cfg.VoiceBackend = config.VoiceBackend
		}
		if config.VoiceBackend != previousBackend {
			// sidecar 的 LLM 上游写在 YAML，运行中不能热改。切档时收掉旧进程，
			// 让下一次 Ensure 从当前配置重建；4B 一起收，避免路由版本错配。
			a.VoiceV2Stop()
		}
		// 即时生效（2026-11-01 实弹教训：切档不通知=用户在 v1 上测了半天 v2 的锅）：
		// ChatPage 订阅 voice-backend-changed 热切语音台，不用重开页面。
		a.emit("voice-backend-changed", map[string]interface{}{"backend": config.VoiceBackend})
	}
	if v, ok := settings["voiceChatLLM"].(string); ok {
		// V2 对话 LLM 位（P1 续刀）：空/local-fast=专属快模型（VoiceV2Ensure
		// 拉起的 4B），chat=跟随聊天板块模型（routeModel("chat") 经 ai.Client
		// 流式）。未知值拒绝保存（save 层同校验）；网关逐请求读配置=即时生效。
		if v != "" && v != "local-fast" && v != "chat" {
			return fmt.Errorf("voiceChatLLM 仅支持空/local-fast/chat（当前值: %s）", v)
		}
		if err := appconfig.Save(appconfig.KeyVoiceChatLLM, v); err != nil {
			slog.Warn("保存语音对话模型配置失败", "error", err)
		}
		if a.cfg != nil {
			a.cfg.VoiceChatLLM = v
		}
		a.emit("voice-chat-llm-changed", map[string]interface{}{"llm": v})
	}
	if v, ok := settings["breezeVoiceDesc"].(string); ok {
		// Breeze voice design 音色描述：只落盘（provider 在 ttsProviderForEngine
		// 处实时读 cfg，不进 voice 运行时配置——语音对话的人格化描述优先级更高）
		if err := appconfig.Save(appconfig.KeyBreezeVoiceDesc, v); err != nil {
			return fmt.Errorf("保存 Breeze 音色描述失败: %w", err)
		}
		if a.cfg != nil {
			a.cfg.BreezeVoiceDesc = v
		}
	}
	if v, ok := settings["personaVoicePresets"]; ok {
		// 角色音色预设（人格→音色）：整体规范化落盘（形状不符整批拒绝——
		// 半套预设会让人格间音色跳变难以排查）
		normalized := map[string]map[string]string{}
		if m, mok := v.(map[string]interface{}); mok {
			for pid, pv := range m {
				pm, pok := pv.(map[string]interface{})
				if !pok {
					return fmt.Errorf("角色音色预设形状不符: %s", pid)
				}
				vid, _ := pm["voiceId"].(string)
				dsc, _ := pm["desc"].(string)
				normalized[pid] = map[string]string{"voiceId": strings.TrimSpace(vid), "desc": strings.TrimSpace(dsc)}
			}
		} else if s, sok := v.(string); !sok || s != "" {
			return fmt.Errorf("personaVoicePresets 须为对象")
		}
		b, err := json.Marshal(normalized)
		if err != nil {
			return fmt.Errorf("序列化角色音色预设失败: %w", err)
		}
		if err := appconfig.Save(appconfig.KeyVoicePersonaPresets, string(b)); err != nil {
			return fmt.Errorf("保存角色音色预设失败: %w", err)
		}
		if a.cfg != nil {
			a.cfg.VoicePersonaPresets = string(b)
		}
	}

	// P0a-A3：人格双读者同点生效（见 personalityPresetId 分支注释）；紧贴
	// ApplyConfig 使 V2 网关读者与 V1 配置读者在同一成功点翻转。
	if havePendingPersonality {
		a.setVoicePersonality(pendingPersonality)
	}
	a.voiceManager.ApplyConfig(config)
	return nil
}

// VoiceGetSettings 获取当前语音设置
func (a *mediaState) VoiceGetSettings() map[string]interface{} {
	if a.voiceManager == nil {
		config := voice.DefaultVoiceConfig()
		return a.voiceSettingsMap(&config)
	}
	config := a.voiceManager.GetConfig()
	return a.voiceSettingsMap(&config)
}

// voiceSettingsMap 构建语音设置映射。
// 2026-11-01 诚实化：退役 TTSHerdsmanModel 展示链（ResolveHerdsmanTTSModel 死链）——
// 该字段不落盘、不参与合成路由（真实路由=聊天语音绑定→全局 TTS→TTSSpeakBase64
// 扫描链，见 synthesizeVoiceTTS），展示解析值只会误导（每次回显触发
// 「resolved=edge-tts」假 fallback 日志）。真实路由前端经 GetVoicePipelineConfig
// 获取，健康经 VoiceHealth 的 ttsRouteHealth 探测。
func (a *mediaState) voiceSettingsMap(c *voice.VoiceRuntimeConfig) map[string]interface{} {
	m := configToMap(c)
	// 实时语音档（S1）：只回 provider/model 与 hasKey 布尔——明文 Key 永不出后端。
	m["realtimeProvider"] = c.RealtimeProvider
	m["realtimeModel"] = c.RealtimeModel
	m["realtimeHasKey"] = c.RealtimeAPIKey != ""
	// Breeze voice design 音色描述（breeze 引擎音色面板回显用）+ 角色音色预设
	// （原始 JSON 串，前端解析渲染逐人格行）
	if a.cfg != nil {
		m["breezeVoiceDesc"] = a.cfg.BreezeVoiceDesc
		m["personaVoicePresets"] = a.cfg.VoicePersonaPresets
	}
	return m
}

// voicePersonaPreset 解析当前人格的音色预设（voice_persona_presets JSON）。
// 坏 JSON/无预设/人格为 plain（无人格）一律返回空——调用方走全局缺省链。
// voicePersonaPreset 解析当前人格（=角色库角色 id）的音色预设，三级独立回退：
// ① 角色卡字段（characterlib.Character.TTSVoiceID/TTSDesc，各自非空即用）；
// ② 全局映射 cfg.VoicePersonaPresets（旧版面板数据，兼容保留）；
// ③ gender 缺省描述（角色卡未配置时出厂不哑——仅 desc，voice 无缺省）。
// plain（无人格）/空 id/库失败 → 空，走全局缺省链。
func (a *mediaState) voicePersonaPreset(personalityID string) (voiceID, desc string) {
	if personalityID == "" || personalityID == "plain" {
		return "", ""
	}
	// ① 角色卡（一次 Get 同时取音色字段与 gender 缺省底料）
	var ch *characterlib.Character
	if a.charLib != nil {
		// Get 无行时返回 (nil, nil)——必须双守卫
		if c, err := a.charLib.Get(personalityID); err == nil && c != nil {
			ch = c
			voiceID = strings.TrimSpace(c.TTSVoiceID)
			desc = strings.TrimSpace(c.TTSDesc)
		}
	}
	// ② 全局映射（旧版面板数据兼容）
	if a.cfg != nil && a.cfg.VoicePersonaPresets != "" && (voiceID == "" || desc == "") {
		var m map[string]struct {
			VoiceID string `json:"voiceId"`
			Desc    string `json:"desc"`
		}
		if err := json.Unmarshal([]byte(a.cfg.VoicePersonaPresets), &m); err == nil {
			if p, ok := m[personalityID]; ok {
				if voiceID == "" {
					voiceID = strings.TrimSpace(p.VoiceID)
				}
				if desc == "" {
					desc = strings.TrimSpace(p.Desc)
				}
			}
		}
	}
	// ③ gender 缺省描述（角色卡/映射均未配置描述时，出厂不哑）
	if desc == "" && ch != nil {
		switch strings.ToLower(ch.Gender) {
		case "male":
			desc = "沉稳清晰的青年男性声线"
		case "female":
			desc = "温柔清晰的青年女性声线"
		}
	}
	return voiceID, desc
}

// herdsmanInstalledModelIDs 返回 Herdsman 引擎当前已安装（可用）模型 ID 列表。
// 引擎未初始化/未启用/无模型列表时返回 nil（消费方 ttsRouteHealth 按
// 「Herdsman 路由不可达」处理）。
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

// VoiceHealth 健康检查（对齐 Ackem voice:health）。
// 2026-11-01 诚实化：ttsReady 覆写为按真实合成路由第一跳探测（原值只查
// synthFn 回调注入=恒 true 的假绿）；asrReady 补 browserAsr 标注——语音对话
// 默认走 WebView2 浏览器识别，后端 ASR provider 只是 PCM 推送路径的依赖。
func (a *mediaState) VoiceHealth() map[string]interface{} {
	if a.voiceManager == nil {
		return map[string]interface{}{
			"asrReady":      false,
			"ttsReady":      false,
			"realtimeReady": false,
			"state":         "idle",
			"browserAsr":    true,
			"ttsDetail":     "语音管理器未初始化",
			"asrDetail":     "语音对话默认走浏览器识别（无需本地模型）",
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
	ready, detail := a.ttsRouteHealth()
	hc["ttsReady"] = ready
	hc["ttsDetail"] = detail
	hc["browserAsr"] = true
	hc["asrDetail"] = "语音对话默认走浏览器识别（无需本地模型）；后端 ASR 用于 PCM 推送路径"
	return hc
}

// ttsRouteHealth 按真实合成路由（聊天语音绑定→全局 TTS→TTSSpeakBase64 扫描
// 链，优先级对齐 synthesizeVoiceTTS）探测第一跳，返回 (立即可用, 说明)。
// 只读探测不拉起（拉起是合成路径行为）；本地引擎离线时合成会自动拉起（路
// 径可走通），但「立即可用」如实报 false——前端红点+「朗读时自动拉起」说明，
// 不再假绿。
func (a *mediaState) ttsRouteHealth() (bool, string) {
	engineID, modelID, hop := "", "", ""
	switch {
	case a.chatVoiceEngine != "" && a.chatVoiceModel != "":
		engineID, modelID, hop = a.chatVoiceEngine, a.chatVoiceModel, "聊天语音绑定"
	case a.activeTTSModel != "":
		engineID, modelID, hop = a.activeTTSEngine, a.activeTTSModel, "全局 TTS"
	default:
		return true, "自动路由：本地引擎优先，Edge 云端兜底"
	}
	label := hop + "：" + engineID + "/" + modelID
	switch {
	case isLocalTTSAssetEngine(engineID):
		if a.ttsReady(engineID) {
			return true, label + "（本地服务在线）"
		}
		if engineID == "breeze" {
			if _, err := breezeCmdFor(a.cfg); err != nil {
				return false, label + "——服务未启动且未检测到安装（朗读将滑落 Edge 兜底）"
			}
		}
		return false, label + "（本地服务离线，朗读时自动拉起）"
	case engineID == "xai":
		// 云端：绑定在位即可用（不实拨 API 花配额；合成失败自动滑落全局/兜底）。
		if a.clientRef() != nil {
			return true, label + "（xai 云端）"
		}
		return false, label + "——云端客户端未就绪（朗读将滑落全局/Edge 兜底）"
	default:
		// Herdsman 系：引擎启用且有已装模型即路由可达。
		if len(a.herdsmanInstalledModelIDs()) > 0 {
			return true, label + "（Herdsman）"
		}
		return false, label + "——Herdsman 未启用/无已装模型（朗读将滑落 Edge 兜底）"
	}
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
		"voiceMode":            string(c.VoiceMode),
		"interruptThresholdMs": c.InterruptThresholdMs,
		"silenceThresholdMs":   c.SilenceThresholdMs,
		"inputChannel":         string(c.InputChannel),
		"personalityPresetId":  c.PersonalityPresetID,
		"voiceBackend":         c.VoiceBackend,
	}
}
