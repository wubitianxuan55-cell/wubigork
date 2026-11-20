package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/sandbox"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func TestHeredocTruncationBecomesHardFailure(t *testing.T) {
	shell := sandbox.ResolveShell()
	if shell.Kind != sandbox.ShellBash {
		t.Skip("bash unavailable; PowerShell has no heredoc syntax")
	}
	args, _ := json.Marshal(map[string]any{
		"command": "cat <<'EOF'\nhello",
	})
	out, err := (bash{shell: shell}).Execute(context.Background(), args)
	if err == nil {
		t.Fatalf("missing heredoc terminator must be a hard failure, got output=%q", out)
	}
	var failure *tool.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("want structured *tool.Failure, got %T: %v", err, err)
	}
	if failure.Subcode != "HEREDOC_TRUNCATED" ||
		failure.EffectState != tool.EffectUnknown ||
		!strings.Contains(failure.NextAction, "write_file") ||
		!strings.Contains(failure.NextAction, "edit_file") ||
		!strings.Contains(failure.Err.Error(), "实际收到") {
		t.Fatalf("failure contract = %+v", failure)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("failure should preserve the received body for diagnosis: %q", out)
	}
}

func TestHeredocTruncationJSONHardFailure(t *testing.T) {
	shell := sandbox.ResolveShell()
	if shell.Kind != sandbox.ShellBash {
		t.Skip("bash unavailable; PowerShell has no heredoc syntax")
	}
	args, _ := json.Marshal(map[string]any{
		"command":       "cat <<'EOF'\nhello",
		"output_format": "json",
	})
	out, err := (bash{shell: shell}).Execute(context.Background(), args)
	var failure *tool.Failure
	if !errors.As(err, &failure) || failure.Subcode != "HEREDOC_TRUNCATED" {
		t.Fatalf("JSON mode heredoc must hard-fail: out=%q err=%v", out, err)
	}
	if !strings.Contains(out, `"stderr"`) || !strings.Contains(out, "here-document") {
		t.Fatalf("JSON failure should preserve structured stderr: %s", out)
	}
}

func TestHeredocWarningMatcher(t *testing.T) {
	for _, stderr := range []string{
		"/usr/bin/bash: line 9: warning: here-document at line 2 delimited by end-of-file (wanted `EOF')",
		"bash: warning: here-document at line 4 delimited by end-of-file",
	} {
		if !heredocTruncated(stderr) {
			t.Fatalf("matcher missed %q", stderr)
		}
	}
	if heredocTruncated("warning: something else") {
		t.Fatal("unrelated warning must not be classified as heredoc truncation")
	}
}
