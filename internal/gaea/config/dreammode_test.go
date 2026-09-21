package config

import "testing"

// v4.377 自动做梦建议制：DreamMode 归一化——默认/空/非法 → suggest，
// 仅显式 auto/off 保留（旧行为回退开关）。
func TestDreamModeNormalize(t *testing.T) {
	if m := Default().DreamMode(); m != "suggest" {
		t.Fatalf("Default().DreamMode() = %q, want suggest", m)
	}
	cases := []struct {
		in   string
		want string
	}{
		{"", "suggest"},
		{"suggest", "suggest"},
		{"auto", "auto"},
		{"off", "off"},
		{"bogus", "suggest"},
		{"AUTO", "suggest"}, // 大小写敏感：非法值归一 suggest，不猜意图
	}
	for _, tc := range cases {
		c := &Config{Dream: DreamConfig{Mode: tc.in}}
		if m := c.DreamMode(); m != tc.want {
			t.Fatalf("DreamMode(%q) = %q, want %q", tc.in, m, tc.want)
		}
	}
	var nilCfg *Config
	if m := nilCfg.DreamMode(); m != "suggest" {
		t.Fatalf("nil DreamMode() = %q, want suggest", m)
	}
}
