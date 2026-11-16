package builtin

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/sandbox"
)

func TestBashDescriptionNamesResolvedInterpreter(t *testing.T) {
	bashTool := bash{shell: sandbox.Shell{Kind: sandbox.ShellBash, Path: "bash"}}
	for label, text := range map[string]string{
		"description": bashTool.Description(),
		"compact":     bashTool.CompactDescription(),
	} {
		if !strings.Contains(text, "bash -c") || !(strings.Contains(text, "not PowerShell") || strings.Contains(text, "不是 PowerShell") || strings.Contains(text, "do not send PowerShell")) {
			t.Fatalf("%s should identify bash and reject PowerShell syntax: %s", label, text)
		}
	}

	psTool := bash{shell: sandbox.Shell{Kind: sandbox.ShellPowerShell, Path: "powershell.exe"}}
	text := psTool.Description()
	if !strings.Contains(text, "Windows PowerShell") || !strings.Contains(text, "Get-ChildItem") {
		t.Fatalf("PowerShell description should name the interpreter and syntax: %s", text)
	}
}
