package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// precheckTool runs a fast deterministic check before a writer tool executes.
// Returns "" when the call is likely to succeed, or a diagnostic message when
// it is predictably going to fail — saving the model an entire API roundtrip.
//
// 缓存安全: 纯运行时判断，不修改任何进入 API 消息数组的内容。
// 阻止调用时返回的消息作为本轮新的 tool_result 追加在消息末尾，
// 不改变已有前缀。
func (a *AgentRunner) precheckTool(name string, args json.RawMessage) string {
	switch name {
	case "edit_file":
		return a.precheckEditFile(args)
	case "multi_edit":
		return a.precheckMultiEdit(args)
	}
	return ""
}

// precheckEditFile verifies that old_string exists in the target file before
// letting edit_file run. Uses the toolCache when available; falls back to a
// direct read. This catches the single most common agent failure pattern:
// the model sends an old_string that doesn't match the current file content.
func (a *AgentRunner) precheckEditFile(raw json.RawMessage) string {
	var p struct {
		Path      string `json:"path"`
		OldString string `json:"old_string"`
	}
	if err := json.Unmarshal(raw, &p); err != nil || p.Path == "" || p.OldString == "" {
		return "" // can't check — let the real Execute handle it
	}

	content, ok := a.readFileForPrecheck(p.Path)
	if !ok {
		return "" // can't read — let the real Execute report the error
	}

	if strings.Contains(content, p.OldString) {
		return "" // found — let it proceed
	}

	// old_string not found — give the model actionable diagnostics
	preview := p.OldString
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}
	filePreview := content
	if len(filePreview) > 200 {
		filePreview = filePreview[:200] + "..."
	}
	return fmt.Sprintf(
		"precheck blocked: old_string not found in %s.\n"+
			"  searched for: %q\n"+
			"  file content (first 200 chars): %q\n"+
			"  suggestion: use read_file to see the current content, then retry with the exact string.",
		p.Path, preview, filePreview,
	)
}

// precheckMultiEdit checks each edit in a multi_edit batch against the target
// file. Returns "" when all old_strings are present.
func (a *AgentRunner) precheckMultiEdit(raw json.RawMessage) string {
	var p struct {
		Path  string `json:"path"`
		Edits []struct {
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		} `json:"edits"`
	}
	if err := json.Unmarshal(raw, &p); err != nil || p.Path == "" || len(p.Edits) == 0 {
		return ""
	}

	content, ok := a.readFileForPrecheck(p.Path)
	if !ok {
		return ""
	}

	for i, e := range p.Edits {
		// S0.6: empty old_string is rejected uniformly (precheck and the
		// multi_edit Execute must agree), so precheck never green-lights a
		// call the executor is guaranteed to reject.
		if e.OldString == "" {
			return fmt.Sprintf(
				"precheck blocked: multi_edit[%d] old_string is empty. Empty old_string is rejected — "+
					"every edit must match existing text (use edit_file with new_string:\"\" to delete, "+
					"or write_file to create a file).",
				i)
		}
		if !strings.Contains(content, e.OldString) {
			preview := e.OldString
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			return fmt.Sprintf(
				"precheck blocked: multi_edit[%d] old_string not found in %s: %q. "+
					"Re-read the file and retry.",
				i, p.Path, preview,
			)
		}
	}
	return ""
}

// readFileForPrecheck returns file content for precheck purposes. Uses the
// toolCache when available (faster, no disk IO); falls back to a direct read.
func (a *AgentRunner) readFileForPrecheck(path string) (string, bool) {
	// Try the tool cache first (cached read_file results from this turn).
	if a.tc != nil {
		if content, hit := a.tc.Get(path, 0); hit {
			return content, true
		}
	}
	// Fall back to a direct read.
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	content := string(b)
	// Populate the tool cache so the subsequent Execute() call (edit_file,
	// multi_edit, etc.) reuses this read instead of hitting the disk again.
	// V10.13: 消除 precheck→execute 的重复文件 I/O，对大文件效果显著。
	if a.tc != nil {
		a.tc.Set(path, 0, content)
	}
	return content, true
}
