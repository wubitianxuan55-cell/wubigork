package visual

import "testing"

// TestExtractEmotionValueDeterministic 审计 P1 AP7-06：情绪评分此前依赖 map
// 迭代序（Go 随机），多关键词标签每次取值不同。改有序表后：多关键词标签取
// 表序首个命中，重复评分恒等。
func TestExtractEmotionValueDeterministic(t *testing.T) {
	cases := []struct {
		emotion     string
		wantTension float64
		wantValence float64
	}{
		// 多关键词：表序首个命中（紧张 8 优先于温馨 2；紧张 -1 优先于温馨 4）。
		{"紧张而温馨", 8, -1},
		{"恐惧与温馨交织", 9, -4},
		// 单关键词与默认值。
		{"温馨", 2, 4},
		{"高兴", 5, 5},
		{"平淡叙述", 5, 0},
	}
	for _, c := range cases {
		for i := 0; i < 50; i++ { // 随机 map 序时代此断言间歇红；固定表序后恒绿
			tension, valence := extractEmotionValue(c.emotion)
			if tension != c.wantTension || valence != c.wantValence {
				t.Fatalf("extractEmotionValue(%q) = (%v,%v), want (%v,%v)",
					c.emotion, tension, valence, c.wantTension, c.wantValence)
			}
		}
	}
}
