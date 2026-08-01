// Package voice — 语音运行时配置
//
// 100% 对齐 Ackem voiceRuntimeConfig.ts 的设计
// 管理语音聊天管道的所有可配置参数
package voice

// VoiceMode 语音输入模式
type VoiceMode string

const (
	VoiceModeVAD VoiceMode = "vad" // 语音活动检测（自动）
	VoiceModePTT VoiceMode = "ptt" // 按键说话（手动）
	VoiceModeOff VoiceMode = "off" // 关闭语音
)

// InputChannel 输入通道模式
type InputChannel string

const (
	InputDual      InputChannel = "dual"       // 语音+文字双通道
	InputVoiceOnly InputChannel = "voice-only" // 仅语音
	InputTextOnly  InputChannel = "text-only"  // 仅文字
)

// TTSEngine TTS 引擎类型
type TTSEngine string

const (
	TTSEngineAuto            TTSEngine = "auto"             // 自动选择
	TTSEngineHerdsman        TTSEngine = "herdsman"         // 本地 Herdsman
	TTSEngineEdge            TTSEngine = "edge-tts"         // 微软 Edge（在线）
	TTSEngineSAPI            TTSEngine = "local-sapi"       // Windows SAPI
	TTSEngineHerdsmanQwen3   TTSEngine = "qwen3-tts"        // Qwen3-TTS via Herdsman
	TTSEngineHerdsmanEdgeTTS TTSEngine = "herdsman-edge-tts" // Edge TTS via Herdsman
)

// ASRModel ASR 模型类型
type ASRModel string

const (
	ASRModelWhisperBase    ASRModel = "whisper-base"                           // 通用 Whisper base
	ASRModelSherpaOnnx     ASRModel = "sherpa-onnx-streaming-zipformer-zh-14m" // 实时流式
	ASRModelFunASR         ASRModel = "funasr"                                 // FunASR 中文优化
)

// VoiceRuntimeConfig 语音运行时配置（对齐 Ackem voiceRuntimeConfig.ts）
type VoiceRuntimeConfig struct {
	// 总开关
	Enabled bool `json:"enabled"`

	// TTS 开关
	TTSEnabled bool `json:"ttsEnabled"`

	// ASR 模型
	ASRModel ASRModel `json:"asrModel"`

	// TTS 引擎
	TTSEngine TTSEngine `json:"ttsEngine"`

	// Edge TTS 音色（当引擎为 edge-tts 时使用）
	TTSVoice string `json:"ttsVoice"`

	// Herdsman TTS 模型
	TTSHerdsmanModel string `json:"ttsHerdsmanModel"`

	// 语音输入模式
	VoiceMode VoiceMode `json:"voiceMode"`

	// 打断阈值（毫秒）：AI 说话时检测到用户声音超过此时间则打断
	InterruptThresholdMs int `json:"interruptThresholdMs"`

	// 沉默阈值（毫秒）：用户停止说话后等待此时间才结束录音
	SilenceThresholdMs int `json:"silenceThresholdMs"`

	// 输入通道模式
	InputChannel InputChannel `json:"inputChannel"`

	// 人格预设 ID（影响情感语调）
	PersonalityPresetID string `json:"personalityPresetId"`
}

// DefaultVoiceConfig 默认语音配置
func DefaultVoiceConfig() VoiceRuntimeConfig {
	return VoiceRuntimeConfig{
		Enabled:              false,
		TTSEnabled:           true,
		ASRModel:             ASRModelWhisperBase,
		TTSEngine:            TTSEngineAuto,
		TTSVoice:             "zh-CN-YunxiNeural",
		TTSHerdsmanModel:     "qwen3-tts-customvoice",
		VoiceMode:            VoiceModeVAD,
		InterruptThresholdMs: 500,
		SilenceThresholdMs:   1000,
		InputChannel:         InputDual,
		PersonalityPresetID:  "gaea",
	}
}

// Validate 校验配置合法性
func (c *VoiceRuntimeConfig) Validate() {
	if c.InterruptThresholdMs < 100 {
		c.InterruptThresholdMs = 100
	}
	if c.InterruptThresholdMs > 3000 {
		c.InterruptThresholdMs = 3000
	}
	if c.SilenceThresholdMs < 200 {
		c.SilenceThresholdMs = 200
	}
	if c.SilenceThresholdMs > 5000 {
		c.SilenceThresholdMs = 5000
	}
	if c.TTSVoice == "" {
		c.TTSVoice = "zh-CN-YunxiNeural"
	}
	if c.TTSHerdsmanModel == "" {
		c.TTSHerdsmanModel = "qwen3-tts-customvoice"
	}
	if c.ASRModel == "" {
		c.ASRModel = ASRModelWhisperBase
	}
}

// ── 音频参数常量（对齐 Ackem 的音频采集配置） ──

const (
	// 采样率：16kHz（Herdsman ASR 要求）
	SampleRate = 16000
	// 位深度：16bit
	BitDepth = 16
	// 声道数：单声道
	Channels = 1
	// 每帧时长（毫秒）：用于 VAD 和流式传输
	ChunkMs = 200
	// 每帧字节数 = SampleRate * (BitDepth/8) * Channels * (ChunkMs/1000)
	ChunkBytes = SampleRate * 2 * Channels * ChunkMs / 1000
)

// ── 语音状态（对齐 Ackem voiceManager 的状态机） ──

// VoiceState 语音管道状态
type VoiceState string

const (
	StateIdle      VoiceState = "idle"
	StateListening VoiceState = "listening"
	StateThinking  VoiceState = "thinking"
	StateSpeaking  VoiceState = "speaking"
)

// ── VAD 参数 ──

const (
	// 语音能量阈值（RMS，对齐 Ackem SPEECH_ENERGY_THRESHOLD = 0.012）
	// Go 端使用 int16 样本的均方根阈值
	SpeechEnergyThreshold = 400 // int16 RMS ≈ 400 对应 float32 RMS ≈ 0.012
)
