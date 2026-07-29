// Package voice — 语音管道管理器
//
// 100% 对齐 Ackem voiceManager.ts 的状态机和管道设计：
//
//  状态机: idle → listening → thinking → speaking → idle
//                          ↑                        │
//                          └──── interrupt ──────────┘
//
// VAD：基于 RMS 能量检测，可配置沉默/打断阈值
// 管道：音频输入 → VAD → ASR → Whisper 伴侣引擎 → TTS → 音频输出
package voice

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/wubigork/wubigork/internal/asr"
)

// ── 回调接口 ──

// EventEmitter 事件发射器（由 App 层实现，推送到前端）
type EventEmitter interface {
	EmitVoiceState(state VoiceState)
	EmitVoiceTranscript(text string, isFinal bool)
	EmitVoiceReply(text string)              // AI 回复文本（对话中显示）
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
	vadBuffer         []byte // 累积的音频缓冲
	vadSilenceMs      int    // 当前连续静音毫秒数
	vadLastEnergy     float64
	vadSpeechDetected bool // 是否已检测到语音

	// 打断检测（P0修复: 对齐 ackem interruptSpeechMs 累积逻辑）
	interruptSpeechMs int  // 连续语音打断累积毫秒数（需超阈值才触发打断）

	// ASR
	asrClient *asr.HerdsmanASR

	// 回调
	whisperChatFn WhisperChatFn
	ttsSynthFn    TTSSynthesizeFn

	// 打断
	interruptCh chan struct{}
	ttsActive   bool
}

// NewManager 创建语音管理器
func NewManager(emitter EventEmitter, config VoiceRuntimeConfig) *Manager {
	config.Validate()
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:      config,
		emitter:     emitter,
		ctx:         ctx,
		cancel:      cancel,
		state:       StateIdle,
		interruptCh: make(chan struct{}, 1),
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
	m.ttsActive = false
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

	// 在 AI 说话时检测打断（累积阈值模式）
	if state == StateSpeaking {
		energy := rmsEnergy(chunk)
		if energy > SpeechEnergyThreshold {
			// 用户说话中：累积打断时长
			m.mu.Lock()
			m.interruptSpeechMs += ChunkMs
			m.vadSilenceMs = 0
			m.mu.Unlock()

			// 累积超过打断阈值才触发打断（对齐 ackem: 默认500ms）
			if m.interruptSpeechMs >= config.InterruptThresholdMs {
				select {
				case m.interruptCh <- struct{}{}:
				default:
				}
				m.mu.Lock()
				m.interruptSpeechMs = 0
				m.mu.Unlock()
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
	isSpeech := energy > SpeechEnergyThreshold

	m.mu.Lock()
	defer m.mu.Unlock()

	if isSpeech {
		// 语音活跃
		m.vadSilenceMs = 0
		m.vadBuffer = append(m.vadBuffer, chunk...)

		if !m.vadSpeechDetected {
			m.vadSpeechDetected = true
			slog.Debug("VAD: 检测到语音", "energy", energy)
			if m.emitter != nil {
				m.emitter.EmitVoiceListening(true)
			}
		}
	} else if m.vadSpeechDetected {
		// 静音中，但之前检测到语音
		m.vadSilenceMs += ChunkMs
		m.vadBuffer = append(m.vadBuffer, chunk...)

		if m.vadSilenceMs >= config.SilenceThresholdMs {
			// 静音超阈值 → 触发识别
			slog.Debug("VAD: 静音超阈值，触发识别", "duration_ms", m.vadSilenceMs)
			go m.handleSpeechEnd(copyBytes(m.vadBuffer))
			m.vadBuffer = nil
			m.vadSilenceMs = 0
			m.vadSpeechDetected = false
		}
	}
	// 语音未开始前的静音直接丢弃

	return nil
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
	if m.emitter != nil {
		m.emitter.EmitVoiceTranscript(text, true)
	}

	// 2. 获取情感参数
	config := m.GetConfig()
	emotionLabel := "CALM_RATIONAL" // 默认，将被 whisper 覆盖
	voiceDesc := GetVoiceDescription(emotionLabel)

	// 3. Whisper 对话
	if m.whisperChatFn == nil {
		slog.Warn("whisper 对话回调未设置")
		m.setState(StateIdle)
		return
	}

	reply, emotionLabel, err := m.whisperChatFn(text, config.PersonalityPresetID)
	if err != nil {
		slog.Error("whisper 对话失败", "error", err)
		if m.emitter != nil {
			m.emitter.EmitVoiceError(fmt.Errorf("对话失败: %w", err))
		}
		m.setState(StateIdle)
		return
	}
	// 发送回复文本到前端（用于对话显示）
	if m.emitter != nil {
		m.emitter.EmitVoiceReply(reply)
	}

	// 获取带人格修饰的语音指令
	voiceDesc = GetVoiceDescriptionWithPersonality(emotionLabel, config.PersonalityPresetID)

	// 4. TTS 合成并播放
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

	// 5. 自动恢复监听（如果是连续对话模式）
	if config.VoiceMode == VoiceModeVAD {
		time.Sleep(300 * time.Millisecond)
		m.mu.RLock()
		canListen := m.state == StateIdle
		m.mu.RUnlock()
		if canListen {
			m.Start()
		}
	}
}

// speak TTS 合成并通知前端播放（对齐 Ackem speak 流程）
func (m *Manager) speak(text string, voiceDesc string) {
	m.mu.Lock()
	m.ttsActive = true
	m.mu.Unlock()

	m.setState(StateSpeaking)
	if m.emitter != nil {
		m.emitter.EmitVoiceThinking(false)
	}

	audio, mimeType, err := m.ttsSynthFn(text, voiceDesc)
	if err != nil {
		slog.Error("TTS 合成失败", "error", err)
		if m.emitter != nil {
			m.emitter.EmitVoiceError(fmt.Errorf("语音合成失败: %w", err))
			// fallback 到浏览器 TTS
			m.emitter.EmitVoiceTTSSpeakText(text)
		}
	} else if len(audio) > 0 && m.emitter != nil {
		m.emitter.EmitVoiceTTSAudio(audio, mimeType)
	}

	m.mu.Lock()
	m.ttsActive = false
	m.mu.Unlock()
}

// transcribe 调用 ASR 进行语音识别
func (m *Manager) transcribe(audioData []byte) (string, error) {
	if m.asrClient == nil {
		return "", fmt.Errorf("ASR 客户端未设置")
	}

	base64Audio := asr.EncodeBase64(audioData)
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
	wasActive := m.ttsActive
	m.ttsActive = false
	m.mu.Unlock()

	if wasActive && m.emitter != nil {
		m.emitter.EmitVoiceTTSCancel()
	}

	slog.Debug("TTS 已打断")
}

// InterruptCh 返回打断通道
func (m *Manager) InterruptCh() <-chan struct{} {
	return m.interruptCh
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
