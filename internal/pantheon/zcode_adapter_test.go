package pantheon

import (
	"os"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestZCodeLongPromptUsesTemporaryTaskFile(t *testing.T) {
	prompt := strings.Repeat("众", zcodeInlinePromptRunes+512)
	arg, path, cleanup, err := prepareZCodePrompt(prompt)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if path == "" {
		t.Fatal("long prompt must be materialized to a file")
	}
	if strings.Contains(arg, strings.Repeat("众", 128)) {
		t.Fatal("temporary task file path wrapper must not inline the long prompt")
	}
	if units := len(utf16.Encode([]rune(arg))); units >= 30000 {
		t.Fatalf("ZCode argv still too long: %d UTF-16 units", units)
	}
	if !strings.Contains(arg, "Read") || !strings.Contains(arg, path) {
		t.Fatalf("wrapper must instruct ZCode to read %q:\n%s", path, arg)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != clampPrompt(prompt) {
		t.Fatal("temporary task file must contain the complete clamped prompt")
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temporary task file was not removed: %v", err)
	}
}

func TestZCodeShortPromptStaysInline(t *testing.T) {
	prompt := "短任务书"
	arg, path, cleanup, err := prepareZCodePrompt(prompt)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if path != "" {
		t.Fatalf("short prompt unexpectedly spooled to %q", path)
	}
	if arg != prompt {
		t.Fatalf("short prompt changed: %q", arg)
	}
}

func TestZCodePromptDisablesSubagentTools(t *testing.T) {
	args := zcodePromptArgs("分析当前议题", `C:\AI\wubigrok`, nil)
	if !containsAdjacent(args, "--disallowed-tools", "Agent") {
		t.Fatalf("ZCode must not receive the Agent tool: %v", args)
	}
	if !containsValue(args, "Workflow") {
		t.Fatalf("ZCode must not receive the Workflow tool: %v", args)
	}
}
