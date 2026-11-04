// Package voice — 语音域公共面（V2 native/realtime + 朗读 TTS 资产）。
//
// V1 拼接链（voice.Manager/ASR/云 Realtime/VoiceMode·InputChannel·阈值族）
// 已于 2026-11-05 随「V1 语音系统退役」拍板整体删除；本文件只保留仍有消费方
// 的最小面：朗读音色/人格配置形状 + WAV 封装采样率常量。
package voice

// VoiceRuntimeConfig 朗读/音色设置形状（V1 运行参数族已删；仅存消费方读取面）。
type VoiceRuntimeConfig struct {
	// Edge TTS 音色（朗读扫描链兜底音色）
	TTSVoice string `json:"ttsVoice"`

	// 人格预设 ID（朗读/台词按人格解析音色预设）
	PersonalityPresetID string `json:"personalityPresetId"`
}

// DefaultVoiceConfig 默认朗读配置
func DefaultVoiceConfig() VoiceRuntimeConfig {
	return VoiceRuntimeConfig{
		TTSVoice:            "zh-CN-YunxiNeural",
		PersonalityPresetID: "gaea",
	}
}

// Validate 校验配置合法性
func (c *VoiceRuntimeConfig) Validate() {
	if c.TTSVoice == "" {
		c.TTSVoice = "zh-CN-YunxiNeural"
	}
}

// ── 音频参数常量（WAV 封装消费） ──

const (
	// SampleRate 16kHz（16k PCM16 WAV 封装，V2 上行/朗读共用口径）
	SampleRate = 16000
	// BitDepth 16bit
	BitDepth = 16
	// Channels 单声道
	Channels = 1
	// ChunkMs 每帧时长（毫秒）
	ChunkMs = 200
	// ChunkBytes 每帧字节数 = SampleRate * (BitDepth/8) * Channels * (ChunkMs/1000)
	ChunkBytes = SampleRate * 2 * Channels * ChunkMs / 1000
)
