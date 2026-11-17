package agent

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAgentTruncationHelpersKeepTailAndValidUTF8(t *testing.T) {
	input := "HEAD-" + strings.Repeat("中", 200) + "-TAIL🎉"
	for name, got := range map[string]string{
		"truncateStr":  truncateStr(input, 180),
		"truncateText": truncateText(input, 80),
	} {
		if !utf8.ValidString(got) {
			t.Fatalf("%s produced invalid UTF-8: %q", name, got)
		}
		if !strings.HasPrefix(got, "HEAD") {
			t.Fatalf("%s lost head: %q", name, got)
		}
		if !strings.HasSuffix(got, "-TAIL🎉") {
			t.Fatalf("%s lost tail: %q", name, got)
		}
	}
}
