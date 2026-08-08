// Package voice — 语音管道管理器
//
// 100% 对齐 Ackem voiceManager.ts 的状态机和管道设计：
//
//	状态机: idle → listening → thinking → speaking → idle
//	                        ↑                        │
//	                        └──── interrupt ──────────┘
//
// VAD：基于 RMS 能量检测，可配置沉默/打断阈值
// 管道：音频输入 → VAD → ASR → Whisper gaea引擎 → TTS → 音频输出
package voice

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/asr"
	"github.com/gaea/gaea/internal/tts"
)

// ── 回调接口 ──

// EventEmitter 事件发射器（由 App 层实现，推送到前端）
type EventEmitter interface {
	EmitVoiceState(state VoiceState)
	EmitVoiceTranscript(text string, isFinal bool)
	EmitVoiceReply(text string) // AI 回复文本（对话中显示）
	EmitVoiceTTSAudio(audio []byte, mimeType string)
	EmitVoiceTTSSpeakText(text string) // 浏览器 fallback
	EmitVoiceTTSCancel()
	EmitVoiceListening(active bool)
	EmitVoiceThinking(active bool)
	EmitVoiceError(err error)
}

// WhisperChatFn whisper 对话函数签名
// 对齐 Ackem 中调用 LLM 的方式
type WhisperChatFn func(userMsg string, personalityID string) (reply string, emotionLabel string, err error)

// TTSSynthesizeFn TTS 合成函数签名
type TTSSynthesizeFn func(text string, voiceDescription string) (audio []byte, mimeType string, err error)

// ── VoiceManager ──

// Manager 语音管道管理器（对齐 Ackem voiceManager.ts）
type Manager struct {
	config  VoiceRuntimeConfig
	emitter EventEmitter
	ctx     context.Context
	cancel  context.CancelFunc

	// 状态
	mu    sync.RWMutex
	state VoiceState

	// VAD
	vadBuffer         []byte  // 累积的音频缓冲
	vadSilenceMs      int     // 当前连续静音毫秒数
	vadSpeechDetected bool    // 是否已确认检测到语音
	speechFrames      int     // 连续语音帧计数（过滤环境噪声尖峰）
	noiseFloor        float64 // 环境噪声底噪估计（指数移动平均）
	vadTurnMs         int     // 当前轮已录时长（单轮超时保护）

	// 打断检测（P0修复: 对齐 ackem interruptSpeechMs 累积逻辑）
	interruptSpeechMs int // 连续语音打断累积毫秒数（需超阈值才触发打断）

	// ASR
	asrClient *asr.HerdsmanASR

	// 回调
	whisperChatFn WhisperChatFn
	ttsSynthFn    TTSSynthesizeFn

	// 打断与轮次控制
	playbackDoneCh chan struct{} // 当前句播放完成信号（前端回调）
	speakStopCh    chan struct{} // 每轮 speak 的停止信号（打断时 close）
	turnCancelCh   chan struct{} // 每轮对话的取消信号（thinking 阶段插话时 close）
	turnMu         sync.Mutex    // 串行化对话轮次，防止多输入并发进入管道
	interrupted    bool          // 上一轮语音回复是否被用户打断（下一轮注入模型）
	ttsActive      bool
	running        bool
}

// NewManager 创建语音管理器
func NewManager(emitter EventEmitter, config VoiceRuntimeConfig) *Manager {
	config.Validate()
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:         config,
		emitter:        emitter,
		ctx:            ctx,
		cancel:         cancel,
		state:          StateIdle,
		playbackDoneCh: make(chan struct{}, 1),
	}
}

// ── 配置与回调设置 ──

// SetASRClient 设置 ASR 客户端
func (m *Manager) SetASRClient(client *asr.HerdsmanASR) {
	m.asrClient = client
}

// SetWhisperChatFn 设置 whisper 对话回调
func (m *Manager) SetWhisperChatFn(fn WhisperChatFn) {
	m.whisperChatFn = fn
}

// SetTTSSynthesizeFn 设置 TTS 合成回调
func (m *Manager) SetTTSSynthesizeFn(fn TTSSynthesizeFn) {
	m.ttsSynthFn = fn
}

// ApplyConfig 应用配置变更（对齐 Ackem voice:apply-settings）
func (m *Manager) ApplyConfig(config VoiceRuntimeConfig) {
	config.Validate()
	m.mu.Lock()
	m.config = config
	m.mu.Unlock()
}

// GetConfig 获取当前配置
func (m *Manager) GetConfig() VoiceRuntimeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetState 获取当前状态
func (m *Manager) GetState() VoiceState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// ── 状态转换 ──

func (m *Manager) setState(state VoiceState) {
	m.mu.Lock()
	old := m.state
	m.state = state
	m.mu.Unlock()

	if old != state {
		slog.Info("语音状态变更", "from", old, "to", state)
		if m.emitter != nil {
			m.emitter.EmitVoiceState(state)
		}
	}
}

// ── 启动/停止 ──

// Start 启动语音管道（进入监听状态）
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.state != StateIdle {
		m.mu.Unlock()
		return fmt.Errorf("voice: 当前状态 %s，无法启动", m.state)
	}
	m.vadBuffer = nil
	m.vadSilenceMs = 0
	m.vadSpeechDetected = false
	m.ttsActive = false
	m.mu.Unlock()

	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	m.setState(StateListening)
	if m.emitter != nil {
		m.emitter.EmitVoiceListening(true)
	}

	slog.Info("语音管道启动", "mode", m.config.VoiceMode, "input", m.config.InputChannel)
	return nil
}

// Stop 停止语音管道
func (m *Manager) Stop() {
	m.cancel() // 取消所有 goroutine

	// 重建 context
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// 清空
	m.mu.Lock()
	m.vadBuffer = nil
	m.vadSilenceMs = 0
	m.vadSpeechDetected = false
	m.speechFrames = 0
	m.vadTurnMs = 0
	m.ttsActive = false
	m.speakStopCh = nil
	m.turnCancelCh = nil
	m.interrupted = false
	m.running = false
	m.noiseFloor = 0
	m.mu.Unlock()

	m.setState(StateIdle)
	if m.emitter != nil {
		m.emitter.EmitVoiceListening(false)
		m.emitter.EmitVoiceThinking(false)
	}

	slog.Info("语音管道停止")
}

// PushAudioChunk 推送 PCM16/16k mono 音频块
// 对齐 Ackem voice:audio-chunk IPC 通道
// P0修复: 打断检测改为累积阈值模式，对齐 ackem interruptSpeechMs 逻辑
func (m *Manager) PushAudioChunk(chunk []byte) error {
	m.mu.RLock()
	state := m.state
	config := m.config
	m.mu.RUnlock()

	if state != StateListening && state != StateSpeaking {
		return nil // 不在监听/说话状态，忽略
	}

	// 在 AI 说话时检测打断（累积阈值模式，阈值随噪声底噪自适应）
	if state == StateSpeaking {
		energy := rmsEnergy(chunk)
		if energy > m.speechThreshold() {
			// 用户说话中：累积打断时长
			m.mu.Lock()
			m.interruptSpeechMs += ChunkMs
			m.vadSilenceMs = 0
			m.mu.Unlock()

			// 累积超过打断阈值才触发打断（对齐 ackem: 默认500ms）
			if m.interruptSpeechMs >= config.InterruptThresholdMs {
				m.mu.Lock()
				m.interruptSpeechMs = 0
				m.mu.Unlock()
				m.handleInterrupt()
			}
		} else {
			// 静音：重置累积
			m.mu.Lock()
			m.interruptSpeechMs = 0
			m.mu.Unlock()
		}
		return nil
	}

	// 正常 VAD 处理
	return m.processVAD(chunk, config)
}

// processVAD 处理 VAD（对齐 Ackem voiceManager VAD 逻辑）
func (m *Manager) processVAD(chunk []byte, config VoiceRuntimeConfig) error {
	energy := rmsEnergy(chunk)

	m.mu.Lock()
	defer m.mu.Unlock()

	threshold := m.speechThresholdLocked()

	// 环境噪声学习：仅未确认语音时，低于当前阈值的帧（静音/环境噪声）
	// 更新底噪（快速向下、慢速向上）；语音帧不参与，避免说话声抬高底噪
	// 导致后续语音被误判为静音。
	if !m.vadSpeechDetected && energy < threshold {
		m.updateNoiseFloor(energy)
		threshold = m.speechThresholdLocked() // 底噪抬高后阈值同步抬高
	}

	isSpeech := energy > threshold

	if isSpeech {
		// 语音活跃
		m.vadSilenceMs = 0
		m.vadTurnMs += ChunkMs
		m.vadBuffer = append(m.vadBuffer, chunk...)

		if !m.vadSpeechDetected {
			// 连续多帧语音才确认（过滤环境噪声尖峰）；确认前的开头帧
			// 保留在缓冲里，避免截掉句首。
			m.speechFrames++
			if m.speechFrames >= MinSpeechFrames {
				m.vadSpeechDetected = true
				slog.Debug("VAD: 检测到语音", "energy", energy, "threshold", threshold)
				if m.emitter != nil {
					m.emitter.EmitVoiceListening(true)
				}
			}
		}
	} else if m.vadSpeechDetected {
		// 静音中，但之前检测到语音
		m.vadSilenceMs += ChunkMs
		m.vadTurnMs += ChunkMs
		m.vadBuffer = append(m.vadBuffer, chunk...)

		if m.vadSilenceMs >= config.SilenceThresholdMs {
			// 静音超阈值 → 触发识别
			slog.Debug("VAD: 静音超阈值，触发识别", "duration_ms", m.vadSilenceMs)
			go m.handleSpeechEnd(copyBytes(m.vadBuffer))
			m.resetVAD()
		} else if m.vadTurnMs >= MaxTurnMs {
			// 单轮超时保护：环境噪声让静音迟迟不来时，强制触发识别
			slog.Debug("VAD: 录音超时，强制识别", "turn_ms", m.vadTurnMs)
			go m.handleSpeechEnd(copyBytes(m.vadBuffer))
			m.resetVAD()
		}
	} else {
		// 未确认语音前的静音/噪声：丢弃缓冲并重置帧计数
		m.speechFrames = 0
		m.vadBuffer = nil
	}

	return nil
}

// speechThresholdLocked 返回当前语音判定阈值（调用方需持有锁）：
// 固定最小阈值与自适应噪声底噪阈值取较大值，环境杂音大时自动抬高。
func (m *Manager) speechThresholdLocked() float64 {
	t := float64(SpeechEnergyThreshold)
	if m.noiseFloor*NoiseFloorFactor > t {
		t = m.noiseFloor * NoiseFloorFactor
	}
	return t
}

// speechThreshold 返回当前语音判定阈值（读锁封装）
func (m *Manager) speechThreshold() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.speechThresholdLocked()
}

// updateNoiseFloor 更新噪声底噪估计：快速向下跟随静音/停顿，
// 慢速向上跟随环境噪声缓慢变化，避免被突发语音尖峰污染。
func (m *Manager) updateNoiseFloor(energy float64) {
	if m.noiseFloor <= 0 {
		m.noiseFloor = energy
		return
	}
	if energy < m.noiseFloor {
		m.noiseFloor = energy
	} else {
		m.noiseFloor += (energy - m.noiseFloor) * noiseFloorUpAlpha
	}
}

// resetVAD 清空当前 VAD 轮次状态（触发识别后调用）
func (m *Manager) resetVAD() {
	m.vadBuffer = nil
	m.vadSilenceMs = 0
	m.vadSpeechDetected = false
	m.speechFrames = 0
	m.vadTurnMs = 0
}

// ── 语音识别 → Whisper → TTS 管道 ──

// handleSpeechEnd 处理语音结束（对齐 Ackem voiceManager flush → ASR → whisper → TTS）
func (m *Manager) handleSpeechEnd(audioData []byte) {
	m.setState(StateThinking)
	if m.emitter != nil {
		m.emitter.EmitVoiceThinking(true)
		m.emitter.EmitVoiceListening(false)
	}

	// 1. ASR 转文字
	text, err := m.transcribe(audioData)
	if err != nil {
		slog.Error("ASR 识别失败", "error", err)
		if m.emitter != nil {
			m.emitter.EmitVoiceError(fmt.Errorf("语音识别失败: %w", err))
		}
		m.setState(StateIdle)
		return
	}

	if text == "" {
		slog.Debug("ASR: 识别结果为空，忽略")
		m.setState(StateIdle)
		return
	}

	slog.Info("ASR 识别结果", "text", text)
	m.runReply(text)
}

// runReply 串行执行一轮语音对话。所有输入入口（浏览器识别 / 后端 ASR）
// 共用同一把 turnMu，避免上一轮还在播 TTS 时新输入并发进入对话管道。
func (m *Manager) runReply(text string) {
	m.turnMu.Lock()
	defer m.turnMu.Unlock()
	m.handleReply(text)
}

// handleReply 处理已识别文本：Whisper 对话 → 流式 TTS 播放 → 恢复监听。
// 同时被 handleSpeechEnd（后端 ASR）与 HandleUserText（浏览器端识别）复用。
func (m *Manager) handleReply(text string) {
	config := m.GetConfig()

	// 防重入：若上一轮仍在说话/思考（例如浏览器连续识别），先停掉旧语音
	m.mu.RLock()
	state := m.state
	m.mu.RUnlock()
	if state == StateSpeaking || state == StateThinking {
		m.CancelTTS()
	}

	// 本轮取消通道：thinking 阶段（LLM 生成中）被插话打断时关闭，
	// 生成完成后跳过播放，让排队的新输入接话。
	m.mu.Lock()
	m.turnCancelCh = make(chan struct{})
	turnCancel := m.turnCancelCh
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.turnCancelCh = nil
		m.mu.Unlock()
	}()

	m.setState(StateThinking)
	if m.emitter != nil {
		m.emitter.EmitVoiceThinking(true)
		m.emitter.EmitVoiceListening(false)
	}

	// 发送识别文本到前端（用于对话显示）
	if m.emitter != nil {
		m.emitter.EmitVoiceTranscript(text, true)
	}

	// Whisper 对话
	if m.whisperChatFn == nil {
		slog.Warn("whisper 对话回调未设置")
		m.setState(StateIdle)
		return
	}

	// 上一轮被打断时，把这一事实告诉模型（对齐 Hermes 的 SPEECH_INTERRUPTED_NOTE）
	userMsg := text
	if m.takeInterrupted() {
		userMsg = "[注意：用户打断了你上一轮未说完的语音回复。] " + text
	}

	reply, emotionLabel, err := m.whisperChatFn(userMsg, config.PersonalityPresetID)
	if err != nil {
		slog.Error("whisper 对话失败", "error", err)
		if m.emitter != nil {
			m.emitter.EmitVoiceError(fmt.Errorf("对话失败: %w", err))
		}
		m.setState(StateIdle)
		return
	}

	// 生成期间被打断：不播放旧回复，让排队的新输入接话
	select {
	case <-turnCancel:
		slog.Debug("生成期间被打断，跳过播放旧回复")
		m.setState(StateIdle)
		if m.emitter != nil {
			m.emitter.EmitVoiceThinking(false)
		}
		return
	default:
	}

	// 发送回复文本到前端（用于对话显示）
	if m.emitter != nil {
		m.emitter.EmitVoiceReply(reply)
	}

	// 获取带人格修饰的语音指令
	voiceDesc := GetVoiceDescriptionWithPersonality(emotionLabel, config.PersonalityPresetID)

	// 流式 TTS：逐句合成并播放（支持 barge-in 打断）
	if config.TTSEnabled && m.ttsSynthFn != nil {
		m.speak(reply, voiceDesc)
	} else {
		// 仅文本模式：通知前端用浏览器 TTS
		if m.emitter != nil {
			m.emitter.EmitVoiceTTSSpeakText(reply)
		}
	}

	m.setState(StateIdle)
	if m.emitter != nil {
		m.emitter.EmitVoiceThinking(false)
	}

	// 自动恢复监听（如果是连续对话模式）
	if config.VoiceMode == VoiceModeVAD && m.isRunning() {
		time.Sleep(300 * time.Millisecond)
		m.mu.RLock()
		canListen := m.state == StateIdle && m.running
		m.mu.RUnlock()
		if canListen {
			m.Start()
		}
	}
}

// HandleUserText 浏览器端（Web Speech API）识别出的文本直接进入对话管道，
// 跳过后端 herdsman ASR，识别更快。AI 正在说话/思考时调用会先打断
// （barge-in），让用户立刻接话。
func (m *Manager) HandleUserText(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	// 插话打断：立即停掉当前 AI 语音，新输入随后进入串行管道
	m.mu.RLock()
	state := m.state
	m.mu.RUnlock()
	if state == StateSpeaking || state == StateThinking {
		m.interruptTurn()
		m.CancelTTS()
	}

	go m.runReply(text)
}

// sentenceAudio 一句 TTS 合成结果
type sentenceAudio struct {
	audio    []byte
	mimeType string
}

// speak 流式 TTS：逐句合成并播放（Hermes 式"边合成边播"）。
// 合成侧 goroutine 预合成下一句，播放侧等当前句播放完成（前端回调
// PlaybackDone）后再取下一句，从而隐藏合成延迟；打断时 close(speakStopCh)，
// 合成与播放两侧立即退出，不再播后续句子。
func (m *Manager) speak(text string, voiceDesc string) {
	m.mu.Lock()
	m.ttsActive = true
	m.speakStopCh = make(chan struct{})
	stop := m.speakStopCh
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.ttsActive = false
		m.speakStopCh = nil
		m.mu.Unlock()
	}()

	m.setState(StateSpeaking)
	if m.emitter != nil {
		m.emitter.EmitVoiceThinking(false)
	}

	sentences := tts.SplitSentences(text)
	if len(sentences) == 0 {
		sentences = []string{text}
	}

	ctx := m.ctx // 捕获：Stop() 会替换 m.ctx，避免读到新 context 而不退出
	audioCh := make(chan sentenceAudio, 1)
	seen := make(map[string]struct{})

	// 合成侧：逐句合成并送入队列（预取下一句，隐藏合成延迟）
	go func() {
		defer close(audioCh)
		for _, sentence := range sentences {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			default:
			}

			// 口语化清洗 + 去重（对齐 Hermes 的 _strip_markdown_for_tts）
			cleaned := cleanForSpeech(sentence)
			if cleaned == "" {
				continue
			}
			if _, dup := seen[cleaned]; dup {
				continue
			}
			seen[cleaned] = struct{}{}

			audio, mimeType, err := m.ttsSynthFn(cleaned, voiceDesc)
			if err != nil {
				slog.Error("TTS 合成失败", "error", err)
				if m.emitter != nil {
					m.emitter.EmitVoiceError(fmt.Errorf("语音合成失败: %w", err))
					m.emitter.EmitVoiceTTSSpeakText(sentence) // fallback 浏览器 TTS
				}
				return
			}
			if len(audio) == 0 {
				continue
			}
			select {
			case audioCh <- sentenceAudio{audio: audio, mimeType: mimeType}:
			case <-ctx.Done():
				return
			case <-stop:
				return
			}
		}
	}()

	// 播放侧：等当前句播完 → 取下一句；打断/停止随时退出
	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case sa, ok := <-audioCh:
			if !ok {
				return // 所有句子已合成并播完
			}

			// 每句用独立的 ack 通道，避免上一句的残留信号误放行下一句
			ackCh := make(chan struct{}, 1)
			m.mu.Lock()
			m.playbackDoneCh = ackCh
			m.mu.Unlock()

			if m.emitter != nil {
				m.emitter.EmitVoiceTTSAudio(sa.audio, sa.mimeType)
			}

			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ackCh:
				// 当前句播放完成，继续下一句
			}
		}
	}
}

// transcribe 调用 ASR 进行语音识别
func (m *Manager) transcribe(audioData []byte) (string, error) {
	if m.asrClient == nil {
		return "", fmt.Errorf("ASR 客户端未设置")
	}

	wavAudio := wrapPCMAsWAV(audioData)
	base64Audio := asr.EncodeBase64(wavAudio)
	result, err := m.asrClient.TranscribeBase64(base64Audio, "audio/wav")
	if err != nil {
		return "", err
	}

	return asr.NormalizeTranscription(result.Text), nil
}

// ── 打断 ──

// CancelTTS 打断当前 TTS 播放（对齐 Ackem voice:cancel-tts）
func (m *Manager) CancelTTS() {
	m.mu.Lock()
	stop := m.speakStopCh
	m.speakStopCh = nil
	wasActive := m.ttsActive
	m.ttsActive = false
	m.interrupted = true // 用户打断：下一轮告诉模型
	m.mu.Unlock()

	if stop != nil {
		close(stop) // 让 speak 的合成/播放循环立即退出
	}

	if wasActive && m.emitter != nil {
		m.emitter.EmitVoiceTTSCancel()
	}

	slog.Debug("TTS 已打断")
}

// interruptTurn 取消当前轮次（thinking 阶段插话时关闭轮次取消通道，
// 让生成完成后的回复不再播放）。与 CancelTTS 分工：CancelTTS 停播放，
// interruptTurn 停"将要播放"。
func (m *Manager) interruptTurn() {
	m.mu.Lock()
	tc := m.turnCancelCh
	m.turnCancelCh = nil
	m.mu.Unlock()
	if tc != nil {
		close(tc)
	}
}

// takeInterrupted 读取并清除"上一轮被打断"标记（每轮 handleReply 开头调用）。
func (m *Manager) takeInterrupted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	v := m.interrupted
	m.interrupted = false
	return v
}

// ── PTT 模式 ──

// SetPTTActive PTT 按下/释放（对齐 Ackem voice:ptt-active）
func (m *Manager) SetPTTActive(active bool) {
	m.mu.RLock()
	config := m.config
	state := m.state
	m.mu.RUnlock()

	if config.VoiceMode != VoiceModePTT {
		return
	}

	if active {
		// 按下：开始录音
		if state == StateIdle {
			m.Start()
		}
	} else {
		// 释放：结束录音并识别
		if state == StateListening {
			m.mu.Lock()
			buf := copyBytes(m.vadBuffer)
			m.vadBuffer = nil
			m.vadSilenceMs = 0
			m.vadSpeechDetected = false
			m.mu.Unlock()

			if len(buf) > 0 {
				go m.handleSpeechEnd(buf)
			}
			m.setState(StateIdle)
		}
	}
}

// ── 健康检查 ──

// HealthCheck 健康检查（对齐 Ackem voice:health）
func (m *Manager) HealthCheck() map[string]interface{} {
	asrReady := m.asrClient != nil
	ttsReady := m.ttsSynthFn != nil

	return map[string]interface{}{
		"asrReady":  asrReady,
		"ttsReady":  ttsReady,
		"state":     string(m.GetState()),
		"mode":      string(m.GetConfig().VoiceMode),
		"ttsEngine": string(m.GetConfig().TTSEngine),
		"asrModel":  string(m.GetConfig().ASRModel),
	}
}

// ── 工具函数 ──

// ── TTS 口语化清洗 ──

var (
	markdownFenceRe  = regexp.MustCompile("(?s)```.*?```")
	markdownInlineRe = regexp.MustCompile("`[^`]*`")
	markdownLinkRe   = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	urlRe            = regexp.MustCompile(`https?://\S+`)
	markdownHeaderRe = regexp.MustCompile(`(?m)^#{1,6}\s*`)
)

// cleanForSpeech 清洗一句 TTS 文本（对齐 Hermes 的 _strip_markdown_for_tts）：
// 去掉 markdown 代码块/行内代码/链接/标题/强调符号与裸 URL，压缩空白。
// 避免 Edge/SAPI 把 "**加粗**" 或 "[链接](url)" 念出来。
func cleanForSpeech(s string) string {
	s = markdownFenceRe.ReplaceAllString(s, " ")
	s = markdownInlineRe.ReplaceAllString(s, " ")
	s = markdownLinkRe.ReplaceAllString(s, "$1")
	s = urlRe.ReplaceAllString(s, "")
	s = markdownHeaderRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return strings.Join(strings.Fields(s), " ")
}

// rmsEnergy 计算 int16 PCM 块的均方根能量
// 对齐 Ackem audioWav.ts getRMS()
func rmsEnergy(samples []byte) float64 {
	if len(samples) < 2 {
		return 0
	}

	var sum float64
	count := len(samples) / 2
	for i := 0; i < len(samples)-1; i += 2 {
		// 小端序 int16
		val := int16(samples[i]) | int16(samples[i+1])<<8
		sum += float64(val) * float64(val)
	}

	mean := sum / float64(count)
	return math.Sqrt(mean)
}

func copyBytes(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

// handleInterrupt performs a barge-in: stop the AI's speech so the user can
// take the floor. Called synchronously from PushAudioChunk once the
// accumulated speech duration exceeds the interrupt threshold, so the stop
// signal reaches speak() on the same audio path without an extra goroutine.
func (m *Manager) handleInterrupt() {
	m.mu.RLock()
	state := m.state
	m.mu.RUnlock()
	if state != StateSpeaking {
		return
	}
	m.CancelTTS()
}

// PlaybackDone is called by the frontend when one TTS sentence finishes
// playing. It releases speak() so the next sentence can start.
func (m *Manager) PlaybackDone() {
	m.mu.RLock()
	state := m.state
	ch := m.playbackDoneCh
	m.mu.RUnlock()
	if state != StateSpeaking || ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}

// isRunning reports whether the voice pipeline is in an active Start/Stop
// cycle (used to avoid auto-restarting listening after Stop).
func (m *Manager) isRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// ASRReady reports whether an ASR client has been configured.
func (m *Manager) ASRReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.asrClient != nil
}

// WhisperReady reports whether the whisper chat callback has been configured.
func (m *Manager) WhisperReady() bool {
	return m.whisperChatFn != nil
}
