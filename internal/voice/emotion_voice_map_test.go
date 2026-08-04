// Package voice — 情绪→TTS 语音参数映射测试
package voice

import "testing"

func TestGetEmotionVoiceParams_KnownEmotion(t *testing.T) {
	p := GetEmotionVoiceParams("SWEET_ATTACHMENT")
	if p.VoiceDescription == "" {
		t.Error("已知情绪应返回非空 VoiceDescription")
	}
	if p.EdgeRate == "" || p.EdgePitch == "" {
		t.Errorf("已知情绪应返回 rate/pitch, got %q, %q", p.EdgeRate, p.EdgePitch)
	}
	if p.EdgeRate != "-10%" || p.EdgePitch != "+5Hz" {
		t.Errorf("SWEET_ATTACHMENT 应映射 -10%%/ +5Hz, got %q, %q", p.EdgeRate, p.EdgePitch)
	}
}

func TestGetEmotionVoiceParams_AllMappedEmotions(t *testing.T) {
	// 映射表里的每个情绪都应返回完整参数（无空字段）
	for label, p := range EmotionVoiceMap {
		if p.VoiceDescription == "" {
			t.Errorf("情绪 %s 缺少 VoiceDescription", label)
		}
		if p.EdgeRate == "" || p.EdgePitch == "" {
			t.Errorf("情绪 %s 缺少 Edge 参数: %q/%q", label, p.EdgeRate, p.EdgePitch)
		}
		if p.WinTTSNote == "" {
			t.Errorf("情绪 %s 缺少 WinTTSNote", label)
		}
	}
}

func TestGetEmotionVoiceParams_UnknownFallback(t *testing.T) {
	p := GetEmotionVoiceParams("UNKNOWN_EMOTION")
	if p.VoiceDescription != "用冷静平淡的语气说" {
		t.Errorf("未知情绪应回退 CALM_RATIONAL, got %q", p.VoiceDescription)
	}
	if p.EdgeRate != "0%" || p.EdgePitch != "0Hz" {
		t.Errorf("未知情绪应回退 0%%/0Hz, got %q, %q", p.EdgeRate, p.EdgePitch)
	}
}

func TestGetVoiceDescription_ReturnsInstruction(t *testing.T) {
	got := GetVoiceDescription("ANGRY_ATTACK")
	if got != "用愤怒尖锐的语气说" {
		t.Errorf("ANGRY_ATTACK 指令应含愤怒, got %q", got)
	}
}

func TestGetEdgeTTSParams(t *testing.T) {
	rate, pitch := GetEdgeTTSParams("HURT_GRIEVANCE")
	if rate != "-10%" || pitch != "-5Hz" {
		t.Errorf("HURT_GRIEVANCE 应 -10%%/-5Hz, got %q, %q", rate, pitch)
	}
}

func TestModifyWithPersonality_KnownModifier(t *testing.T) {
	got := ModifyWithPersonality("用温柔甜蜜的语气说", "tsundere")
	if got != "用傲娇又温柔甜蜜的语气说" {
		t.Errorf("tsundere 应插入傲娇又, got %q", got)
	}
}

func TestModifyWithPersonality_UnknownModifier(t *testing.T) {
	got := ModifyWithPersonality("用温柔甜蜜的语气说", "unknown-persona")
	if got != "用温柔甜蜜的语气说" {
		t.Errorf("未知人格应原样返回, got %q", got)
	}
}

func TestModifyWithPersonality_EmptyDescription(t *testing.T) {
	got := ModifyWithPersonality("", "tsundere")
	if got != "" {
		t.Errorf("空描述应原样返回, got %q", got)
	}
}

func TestGetVoiceDescriptionWithPersonality(t *testing.T) {
	got := GetVoiceDescriptionWithPersonality("SWEET_ATTACHMENT", "gaea")
	if got != "用温厚又温柔甜蜜的语气说" {
		t.Errorf("gaea 人格应插入温厚又, got %q", got)
	}
}

func TestGetVoiceDescriptionWithPersonality_NoModifier(t *testing.T) {
	got := GetVoiceDescriptionWithPersonality("SWEET_ATTACHMENT", "nope")
	if got != "用温柔甜蜜的语气说" {
		t.Errorf("无修饰人格应返回原始描述, got %q", got)
	}
}
