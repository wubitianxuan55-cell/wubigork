package main

import "testing"

// TestMaskToken 脱敏格式：只保留尾 4 位，前缀固定为 ***（T6-1.3）。
func TestMaskToken(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "***"},
		{"abcd", "***"},
		{"abcde", "***bcde"},
		{"abcdefgh", "***efgh"},
		{"tok-123456", "***3456"},
	}
	for _, c := range cases {
		if got := maskToken(c.in); got != c.want {
			t.Errorf("maskToken(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
