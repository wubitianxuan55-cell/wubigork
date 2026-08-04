// Package voice — 语音配置与音频纯函数测试
package voice

import (
	"math"
	"testing"
)

func TestDefaultVoiceConfig(t *testing.T) {
	c := DefaultVoiceConfig()
	if c.Enabled {
		t.Error("默认应关闭总开关")
	}
	if !c.TTSEnabled {
		t.Error("默认应开启 TTS")
	}
	if c.ASRModel != ASRModelWhisperBase {
		t.Errorf("默认 ASR 应为 whisper-base, got %q", c.ASRModel)
	}
	if c.TTSEngine != TTSEngineAuto {
		t.Errorf("默认 TTS 引擎应为 auto, got %q", c.TTSEngine)
	}
	if c.VoiceMode != VoiceModeVAD {
		t.Errorf("默认输入模式应为 vad, got %q", c.VoiceMode)
	}
	if c.InputChannel != InputDual {
		t.Errorf("默认通道应为 dual, got %q", c.InputChannel)
	}
	if c.InterruptThresholdMs != 500 || c.SilenceThresholdMs != 1000 {
		t.Errorf("默认阈值应为 500/1000, got %d/%d", c.InterruptThresholdMs, c.SilenceThresholdMs)
	}
	if c.TTSVoice != "zh-CN-YunxiNeural" {
		t.Errorf("默认音色应为 Yunxi, got %q", c.TTSVoice)
	}
	if c.PersonalityPresetID != "gaea" {
		t.Errorf("默认人格应为 gaea, got %q", c.PersonalityPresetID)
	}
}

func TestValidate_ClampsThresholds(t *testing.T) {
	c := VoiceRuntimeConfig{InterruptThresholdMs: 50, SilenceThresholdMs: 100}
	c.Validate()
	if c.InterruptThresholdMs != 100 {
		t.Errorf("中断阈值低于 100 应钳到 100, got %d", c.InterruptThresholdMs)
	}
	if c.SilenceThresholdMs != 200 {
		t.Errorf("沉默阈值低于 200 应钳到 200, got %d", c.SilenceThresholdMs)
	}

	c = VoiceRuntimeConfig{InterruptThresholdMs: 9999, SilenceThresholdMs: 9999}
	c.Validate()
	if c.InterruptThresholdMs != 3000 {
		t.Errorf("中断阈值高于 3000 应钳到 3000, got %d", c.InterruptThresholdMs)
	}
	if c.SilenceThresholdMs != 5000 {
		t.Errorf("沉默阈值高于 5000 应钳到 5000, got %d", c.SilenceThresholdMs)
	}
}

func TestValidate_FillsDefaults(t *testing.T) {
	c := VoiceRuntimeConfig{} // 全空
	c.Validate()
	if c.TTSVoice != "zh-CN-YunxiNeural" {
		t.Errorf("空音色应回退 Yunxi, got %q", c.TTSVoice)
	}
	if c.TTSHerdsmanModel != "qwen3-tts-customvoice" {
		t.Errorf("空模型应回退 qwen3-tts-customvoice, got %q", c.TTSHerdsmanModel)
	}
	if c.ASRModel != ASRModelWhisperBase {
		t.Errorf("空 ASR 应回退 whisper-base, got %q", c.ASRModel)
	}
}

func TestValidate_KeepsValidValues(t *testing.T) {
	c := VoiceRuntimeConfig{
		InterruptThresholdMs: 500,
		SilenceThresholdMs:   1000,
		TTSVoice:             "zh-CN-XiaoxiaoNeural",
		TTSHerdsmanModel:     "custom",
		ASRModel:             ASRModelSherpaOnnx,
	}
	c.Validate()
	if c.InterruptThresholdMs != 500 || c.SilenceThresholdMs != 1000 {
		t.Errorf("合法值不应被修改, got %d/%d", c.InterruptThresholdMs, c.SilenceThresholdMs)
	}
	if c.TTSVoice != "zh-CN-XiaoxiaoNeural" || c.TTSHerdsmanModel != "custom" || c.ASRModel != ASRModelSherpaOnnx {
		t.Error("非空值不应被回退覆盖")
	}
}

func TestRmsEnergy_Silence(t *testing.T) {
	// 全零样本 → RMS 0
	samples := make([]byte, 3200) // 1600 个 int16
	if got := rmsEnergy(samples); got != 0 {
		t.Errorf("静音 RMS 应为 0, got %v", got)
	}
}

func TestRmsEnergy_ConstantSignal(t *testing.T) {
	// 常数 1000（小端 int16）→ RMS 1000
	samples := make([]byte, 3200)
	for i := 0; i < len(samples); i += 2 {
		v := int16(1000)
		samples[i] = byte(v)
		samples[i+1] = byte(v >> 8)
	}
	got := rmsEnergy(samples)
	if math.Abs(got-1000) > 1 {
		t.Errorf("常数信号 RMS 应 ≈1000, got %v", got)
	}
}

func TestRmsEnergy_ShortInput(t *testing.T) {
	if got := rmsEnergy([]byte{0x00}); got != 0 {
		t.Errorf("不足 2 字节应返回 0, got %v", got)
	}
	if got := rmsEnergy(nil); got != 0 {
		t.Errorf("空输入应返回 0, got %v", got)
	}
}

func TestRmsEnergy_MixedSignal(t *testing.T) {
	// 交替 ±1000 → 均方 1000²
	samples := make([]byte, 3200)
	for i := 0; i < len(samples); i += 4 {
		v := int16(1000)
		samples[i] = byte(v)
		samples[i+1] = byte(v >> 8)
		nv := int16(-1000)
		samples[i+2] = byte(nv)
		samples[i+3] = byte(nv >> 8)
	}
	got := rmsEnergy(samples)
	if math.Abs(got-1000) > 1 {
		t.Errorf("交替 ±1000 RMS 应 ≈1000, got %v", got)
	}
}

func TestCopyBytes_IndependentCopy(t *testing.T) {
	src := []byte{1, 2, 3}
	dst := copyBytes(src)
	if len(dst) != 3 || dst[0] != 1 || dst[2] != 3 {
		t.Fatalf("拷贝内容不符: %v", dst)
	}
	dst[0] = 99
	if src[0] != 1 {
		t.Error("修改拷贝不应影响原数组")
	}
}

func TestChunkBytes_Constant(t *testing.T) {
	// 16kHz × 2 字节 × 1 声道 × 200ms = 6400
	if ChunkBytes != 6400 {
		t.Errorf("ChunkBytes 应为 6400, got %d", ChunkBytes)
	}
}
