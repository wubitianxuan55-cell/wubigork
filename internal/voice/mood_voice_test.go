package voice

import "testing"

// v4.6 Mood→TTS 闭环：长期心境连续韵律映射。
func TestMoodToVoiceDescription(t *testing.T) {
	cases := []struct {
		name string
		mood [4]float64
		want string
	}{
		{
			name: "未播种/全中性 → 空（回退标签静态预设）",
			mood: [4]float64{0, 0, 0, 0},
			want: "",
		},
		{
			name: "低落（审计场景：今天心情差，冷静回答也低气压）",
			mood: [4]float64{-40, -35, -30, -10},
			want: "用低沉带着一丝不安平缓的语气说",
		},
		{
			name: "轻微低落只触发低频维",
			mood: [4]float64{-20, 0, 0, 0},
			want: "用低沉的语气说",
		},
		{
			name: "兴奋期（轻快温暖略带强势）",
			mood: [4]float64{30, 0, 25, 20},
			want: "用温暖轻快略带强势的语气说",
		},
		{
			name: "不安但亲和（依赖感强的低安全感）",
			mood: [4]float64{25, -30, -8, -20},
			want: "用温暖带着一丝不安柔和顺从的语气说",
		},
		{
			name: "阈值边界：刚好低于 AffLow 不触发",
			mood: [4]float64{-14, 0, 0, 0},
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MoodToVoiceDescription(c.mood); got != c.want {
				t.Fatalf("MoodToVoiceDescription(%v) = %q, want %q", c.mood, got, c.want)
			}
		})
	}
}
