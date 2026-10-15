// Package voice — 结构化情绪→TTS 参数测试（v4.3d 立；11.6 补全 9 标签）。
//
// 覆盖 GetEmotionVoiceParams：ANGRY/CALM 精确值、其余标签方向与 Edge 参数
// 一致且 Emotion 词宁缺勿误留空、未知/空标签零值、双路径覆盖面一致。
package voice

import (
	"strings"
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

func TestGetEmotionVoiceParams_OtherKnownLabelsFollowEdgeDirection(t *testing.T) {
	// 11.6 补全后：全部 9 个 L2 标签都有非中性结构化参数，且 Speed/Pitch
	// 方向与同标签的 EdgeRate/EdgePitch 逐标签一致（两条路径情感方向统一）。
	cases := []struct {
		label     string
		edgeRate  string // 如 "-10%"；"0%"=不指定语速
		edgePitch string // 如 "+5Hz"；"0Hz"=不指定音高
		wantPitch int    // 5Hz≈1 半音，不足半音取整（-2Hz→0）
	}{
		{"SWEET_ATTACHMENT", "-10%", "+5Hz", 1},
		{"SHY_HEARTBEAT", "-5%", "+3Hz", 1},
		{"QUIET_FOND", "-15%", "-2Hz", 0},
		{"TSUNDERE", "+5%", "+5Hz", 1},
		{"COLD_DETACHED", "0%", "-5Hz", -1},
		{"HURT_GRIEVANCE", "-10%", "-5Hz", -1},
		{"FEARFUL_OBEDIENT", "-5%", "-3Hz", -1},
	}
	sign := func(v float64) int {
		switch {
		case v > 0:
			return 1
		case v < 0:
			return -1
		default:
			return 0
		}
	}
	rateSign := func(s string) int {
		switch {
		case strings.HasPrefix(s, "+"):
			return 1
		case strings.HasPrefix(s, "-"):
			return -1
		default:
			return 0
		}
	}
	for _, c := range cases {
		p := GetEmotionVoiceParams(c.label)
		if p == (tts.TTSParams{}) {
			t.Errorf("标签 %s 应有非中性参数，got 零值", c.label)
			continue
		}
		if rs, ps := rateSign(c.edgeRate), sign(p.Speed-1); rs != 0 && ps != rs {
			t.Errorf("%s: Speed %v 与 EdgeRate %s 方向相反", c.label, p.Speed, c.edgeRate)
		}
		if p.Pitch != float64(c.wantPitch) {
			t.Errorf("%s: Pitch = %v, want %d（EdgePitch %s 按半音取整）", c.label, p.Pitch, c.wantPitch, c.edgePitch)
		}
		// Emotion 词宁缺勿误：除 angry/calm 外一律留空。
		if p.Emotion != "" {
			t.Errorf("%s: Emotion 应留空（引擎支持面未证实），got %q", c.label, p.Emotion)
		}
	}
}

func TestEmotionTTSParamsMap_AllNineLabelsNonNeutral(t *testing.T) {
	// 9 个 L2 标签全在表（11.6 补全），且无零值条目。
	if len(EmotionTTSParamsMap) != 9 {
		t.Errorf("EmotionTTSParamsMap 应恰好含 9 个非中性条目, got %d", len(EmotionTTSParamsMap))
	}
	for label, p := range EmotionTTSParamsMap {
		if p == (tts.TTSParams{}) {
			t.Errorf("映射表条目 %s 不应为零值", label)
		}
	}
	// 表键集与 EmotionVoiceMap 完全一致（两条路径同覆盖面，防漏标签）。
	for label := range EmotionVoiceMap {
		if _, ok := EmotionTTSParamsMap[label]; !ok {
			t.Errorf("标签 %s 在描述路径有映射但结构化路径缺失", label)
		}
	}
	for label := range EmotionTTSParamsMap {
		if _, ok := EmotionVoiceMap[label]; !ok {
			t.Errorf("标签 %s 在结构化路径有映射但描述路径缺失", label)
		}
	}
}
