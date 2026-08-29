// Package voice — 结构化情绪→TTS 参数测试（v4.3d）
//
// 覆盖 GetEmotionVoiceParams（返回 tts.TTSParams）：ANGRY/CALM 非中性映射、
// 其余已知标签中性默认、未知/空标签零值。
package voice

import (
	"testing"

	"github.com/gaea/gaea/internal/tts"
)

func TestGetEmotionVoiceParams_Angry(t *testing.T) {
	p := GetEmotionVoiceParams("ANGRY_ATTACK")
	if p.Speed != 1.1 {
		t.Errorf("ANGRY_ATTACK Speed 应为 1.1, got %v", p.Speed)
	}
	if p.Pitch != 2 {
		t.Errorf("ANGRY_ATTACK Pitch 应为 2, got %v", p.Pitch)
	}
	if p.Emotion != "angry" {
		t.Errorf("ANGRY_ATTACK Emotion 应为 angry, got %q", p.Emotion)
	}
	if p.Style != "" {
		t.Errorf("ANGRY_ATTACK Style 应为空(不指定), got %q", p.Style)
	}
}

func TestGetEmotionVoiceParams_Calm(t *testing.T) {
	p := GetEmotionVoiceParams("CALM_RATIONAL")
	if p.Speed != 0.9 {
		t.Errorf("CALM_RATIONAL Speed 应为 0.9, got %v", p.Speed)
	}
	if p.Pitch != -1 {
		t.Errorf("CALM_RATIONAL Pitch 应为 -1, got %v", p.Pitch)
	}
	if p.Emotion != "calm" {
		t.Errorf("CALM_RATIONAL Emotion 应为 calm, got %q", p.Emotion)
	}
	if p.Style != "" {
		t.Errorf("CALM_RATIONAL Style 应为空(不指定), got %q", p.Style)
	}
}

func TestGetEmotionVoiceParams_UnknownIsZeroValue(t *testing.T) {
	p := GetEmotionVoiceParams("UNKNOWN_EMOTION")
	if p != (tts.TTSParams{}) {
		t.Errorf("未知标签应返回零值 TTSParams, got %+v", p)
	}
}

func TestGetEmotionVoiceParams_EmptyIsZeroValue(t *testing.T) {
	p := GetEmotionVoiceParams("")
	if p != (tts.TTSParams{}) {
		t.Errorf("空标签应返回零值 TTSParams, got %+v", p)
	}
}

func TestGetEmotionVoiceParams_OtherKnownLabelsNeutral(t *testing.T) {
	// 其余 7 个 L2 标签走中性默认（Speed/Pitch=0 表示引擎默认，Style/Emotion 空）
	neutral := []string{
		"SWEET_ATTACHMENT",
		"SHY_HEARTBEAT",
		"QUIET_FOND",
		"TSUNDERE",
		"COLD_DETACHED",
		"HURT_GRIEVANCE",
		"FEARFUL_OBEDIENT",
	}
	for _, label := range neutral {
		if p := GetEmotionVoiceParams(label); p != (tts.TTSParams{}) {
			t.Errorf("标签 %s 应为中性默认(零值), got %+v", label, p)
		}
	}
}

func TestEmotionTTSParamsMap_OnlyNonNeutralEntries(t *testing.T) {
	// 数据表里只应存在非中性条目（ANGRY/CALM），避免意外出现零值条目
	if len(EmotionTTSParamsMap) != 2 {
		t.Errorf("EmotionTTSParamsMap 应恰好含 2 个非中性条目, got %d", len(EmotionTTSParamsMap))
	}
	for label, p := range EmotionTTSParamsMap {
		if p == (tts.TTSParams{}) {
			t.Errorf("映射表条目 %s 不应为零值", label)
		}
	}
}
