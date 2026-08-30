package whisper

import (
	"strings"
	"testing"
)

// v4.9.1 工作人设收口：professional tag 人格（gaea/办公秘书）永久豁免碎碎念
// 节奏——微信/语音实测教训：PAD 标尺（-100..100）下 chatter 阈值形同虚设，
// 办公通道每轮回复都被压成 ≤30 字碎片 + [SPLIT]。
func TestDecideRhythm_ProfessionalPersonaExempt(t *testing.T) {
	for _, pid := range []string{"gaea", "secretary"} {
		input := RhythmInput{
			Aro: 80, Aff: 80, Stage: StageIntimate,
			PersonalityID: pid, Sincerity: 0.9, Intensity: 1.0,
		}
		// 计数器已连续 3 轮 chatter（本应强制切独白）：豁免优先于强制切换。
		counters := &RhythmCounters{Chatter: 5}
		d := DecideRhythm(input, counters)
		if d.Mode != RhythmDefault {
			t.Errorf("%s 应恒为 default（不拆分不碎片化）, got %s", pid, d.Mode)
		}
		if d.Separator != "" || d.Instruction != "" {
			t.Errorf("%s 不应带分隔符/格式指令, got %+v", pid, d)
		}
		if counters.Chatter != 0 || counters.Monologue != 0 {
			t.Errorf("%s 计数器应清零, got %+v", pid, counters)
		}
	}
}

func TestDecideRhythm_CompanionPersonaStillChatters(t *testing.T) {
	// 豁免只对 professional 生效：乐园陪伴人格的碎碎念节奏保持原设计。
	input := RhythmInput{
		Aro: 20, Aff: 30, Stage: StageFamiliar,
		PersonalityID: "genki", Sincerity: 0.5, Intensity: 0.7,
	}
	d := DecideRhythm(input, &RhythmCounters{})
	if d.Mode != RhythmChatter {
		t.Errorf("genki 应保持 chatter 原设计, got %s", d.Mode)
	}
}

// GetPreset("secretary") 应存在且带 professional tag（角色中心可选办公秘书）。
func TestSecretaryPresetRegistered(t *testing.T) {
	p := GetPreset("secretary")
	if p == nil {
		t.Fatal("secretary 预设应已注册")
	}
	if !containsTag(p.Tags, "professional") {
		t.Errorf("secretary 应带 professional tag, got %v", p.Tags)
	}
	if _, ok := PersonalityTemplates["secretary"]; !ok {
		t.Fatal("secretary 详细模板应已注册")
	}
}

// SplitOnMarker：chatter 模式的内部标记绝不能漏到任何出口。
func TestSplitOnMarker(t *testing.T) {
	got := SplitOnMarker("好呀！[SPLIT]今天去哪玩[SPLIT][SPLIT]我准备好了")
	want := []string{"好呀！", "今天去哪玩", "我准备好了"}
	if len(got) != len(want) {
		t.Fatalf("应切出 %d 条, got %v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fragment %d = %q, want %q", i, got[i], want[i])
		}
	}
	if r := SplitOnMarker("[SPLIT]  [SPLIT]"); len(r) != 0 {
		t.Errorf("全标记文本应得到空切片, got %v", r)
	}
	if !strings.Contains(strings.Join(SplitOnMarker(" A [SPLIT] B "), "|"), "B") {
		t.Error("片段应去首尾空白后保留")
	}
}
