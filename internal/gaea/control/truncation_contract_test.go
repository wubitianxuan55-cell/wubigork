package control

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateSummaryKeepsHeadTailAndUTF8(t *testing.T) {
	input := "HEAD-" + strings.Repeat("中", 300) + "-TAIL🎉"
	got := truncateSummary(input)
	if !utf8.ValidString(got) {
		t.Fatalf("invalid UTF-8: %q", got)
	}
	if !strings.HasPrefix(got, "HEAD") || !strings.HasSuffix(got, "-TAIL🎉") {
		t.Fatalf("summary did not preserve head/tail: %q", got)
	}
	if n := utf8.RuneCountInString(got); n > 240 {
		t.Fatalf("summary runes=%d over 240: %q", n, got)
	}
}
