package app

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateVoiceContext(t *testing.T) {
	short := "人格状态与最近对话"
	if got := truncateVoiceContext(short); got != short {
		t.Fatalf("短上下文不应变化: %q", got)
	}

	long := strings.Repeat("记", voiceContextCharBudget+180)
	got := truncateVoiceContext(long)
	if !strings.Contains(got, "语音低延迟模式") {
		t.Fatalf("长上下文应带压缩标记: %q", got[:min(len(got), 80)])
	}
	if n := utf8.RuneCountInString(got); n > voiceContextCharBudget+40 {
		t.Fatalf("压缩后仍过长: %d runes", n)
	}
}
