package cjk

import "testing"

// CJK 部首归一（ZCode 蒸馏产出道 CJK 刀）：验收口径=⽂→文、不改全角
// 标点、混排逐字只动部首区。表移植自 ZCode cjk.ts（Apache-2.0）。
func TestNormalizeCJKCompat(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"康熙段 文", "⽂", "文"},
		{"康熙段 一", "⼀", "一"},
		{"康熙段 人", "⼈", "人"},
		{"补充段 黄", "⻩", "黄"},
		{"补充段 衤", "⻂", "衤"},
		{"补充段 讠", "⻈", "讠"},
		{"补充段 龙", "⻰", "龙"},
		{"PDF 场景混排", "⻩⾊预警与⽂档管理", "黄色预警与文档管理"},
		{"全角标点不动", "，。！？：；（）", "，。！？：；（）"},
		{"ASCII 不动", "hello ZCode 42", "hello ZCode 42"},
		{"普通汉字不动", "黄文龙一", "黄文龙一"},
		{"空串", "", ""},
		{"无部首", "普通文本", "普通文本"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeCJKCompat(tc.in); got != tc.want {
				t.Fatalf("NormalizeCJKCompat(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
	// 幂等：归一后的文本再过一遍不变（等价目标都在 BMP 汉字区，不会再命中）。
	if got := NormalizeCJKCompat(NormalizeCJKCompat("⽂⻩")); got != "文黄" {
		t.Fatalf("幂等失败: %q", got)
	}
}
