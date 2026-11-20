package builtin

import (
	"strings"
	"testing"
)

func TestVisionLayoutRequestsCarryDeterministicBoundary(t *testing.T) {
	for _, prompt := range []string{
		"检查有没有文字重叠", "是否出界", "标签溢出了吗", "check text overlap", "layout quality",
	} {
		got := withVisionLayoutBoundary("visual answer", prompt)
		if !strings.Contains(got, "[排版边界]") || !strings.Contains(got, "layout") {
			t.Fatalf("prompt %q did not carry layout boundary: %q", prompt, got)
		}
	}
	if got := withVisionLayoutBoundary("visual answer", "提取图中所有文字"); got != "visual answer" {
		t.Fatalf("semantic/OCR prompt should stay untouched: %q", got)
	}
}
