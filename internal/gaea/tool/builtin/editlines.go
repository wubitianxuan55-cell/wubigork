package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(editLines{}) }

// editLines replaces a 1-based, endpoint-inclusive line range in a file — the
// line-numbered counterpart of read_file's display. Splitting is raw: lines
// are separated on "\n" and no line ending is normalized, so untouched lines
// come back byte-identical (a CRLF file stays CRLF outside the replaced
// range). Shares the encoding-aware write kernel with edit_file.
type editLines struct {
	roots   []string
	workDir string
}

func (editLines) Name() string { return "edit_lines" }

func (editLines) Description() string {
	return "Replace a line range (1-based, end_line inclusive) in a file with new_content. Empty new_content deletes the lines. Line numbers match read_file's display. Untouched lines are preserved byte-for-byte; encoding (GB18030/UTF-16) and permissions are preserved."
}

func (editLines) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"File path"},
  "start_line":{"type":"integer","description":"First line to replace (1-based)","minimum":1},
  "end_line":{"type":"integer","description":"Last line to replace (inclusive)"},
  "new_content":{"type":"string","description":"Replacement lines separated by \\n (empty string deletes the range)"}
},
"required":["path","start_line","end_line","new_content"]
}`)
}

func (editLines) ReadOnly() bool { return false }

func (editLines) CompactDescription() string     { return compactDesc["edit_lines"] }
func (editLines) CompactSchema() json.RawMessage { return compactSchema["edit_lines"] }

func (e editLines) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path       string  `json:"path"`
		StartLine  int     `json:"start_line"`
		EndLine    int     `json:"end_line"`
		NewContent *string `json:"new_content"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if p.NewContent == nil {
		return "", fmt.Errorf("new_content is required (pass \"\" to delete the line range)")
	}
	if p.StartLine < 1 {
		return "", fmt.Errorf("start_line must be >= 1 (line numbers are 1-based), got %d", p.StartLine)
	}
	if p.EndLine < p.StartLine {
		return "", fmt.Errorf("end_line (%d) must be >= start_line (%d)", p.EndLine, p.StartLine)
	}
	p.Path = resolveIn(e.workDir, p.Path)
	if err := confine(e.roots, p.Path); err != nil {
		return "", err
	}
	content, enc, err := readFileEncoded(p.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", p.Path, err)
	}
	baseline := evidence.StageBaseline(ctx, p.Path, []byte(content))

	lines, hadTrailing := splitFileLines(content)
	if p.StartLine > len(lines) {
		return "", fmt.Errorf("start_line %d is past EOF — file has %d line(s)", p.StartLine, len(lines))
	}
	if p.EndLine > len(lines) {
		return "", fmt.Errorf("end_line %d is past EOF — file has %d line(s)", p.EndLine, len(lines))
	}
	newLines := newLinesFromContent(*p.NewContent)

	merged := make([]string, 0, len(lines)-(p.EndLine-p.StartLine+1)+len(newLines))
	merged = append(merged, lines[:p.StartLine-1]...)
	merged = append(merged, newLines...)
	merged = append(merged, lines[p.EndLine:]...)

	result := ""
	if len(merged) > 0 {
		result = strings.Join(merged, "\n")
		if hadTrailing {
			// Preserve the original trailing newline.
			result += "\n"
		}
	}
	if err := atomicWriteEncoded(p.Path, result, enc); err != nil {
		return "", fmt.Errorf("write %s: %w", p.Path, err)
	}
	// v4.1 证据链：被替换行区（原文摘要）→ 新内容。
	beforeLines := lines[p.StartLine-1 : p.EndLine]
	evidence.RecordChange(ctx, evidence.ChangeRecord{
		Tool:          "edit_lines",
		Target:        p.Path,
		BeforeSummary: strings.Join(beforeLines, "\n"),
		AfterSummary:  *p.NewContent,
		BaselinePath:  baseline,
	})
	return fmt.Sprintf("replaced lines %d-%d in %s (%d→%d lines)",
		p.StartLine, p.EndLine, p.Path, p.EndLine-p.StartLine+1, len(newLines)), nil
}

// splitFileLines splits content on "\n" with no EOL normalization: a CRLF file
// keeps its "\r" as part of each line element, so untouched lines round-trip
// byte-identically. A trailing newline does not create an extra empty line;
// hadTrailing reports that the file ended with one.
func splitFileLines(content string) (lines []string, hadTrailing bool) {
	hadTrailing = strings.HasSuffix(content, "\n")
	s := strings.TrimSuffix(content, "\n")
	if s == "" {
		if hadTrailing {
			return []string{""}, true // a file holding a single blank line
		}
		return nil, false // empty file — zero lines
	}
	return strings.Split(s, "\n"), hadTrailing
}

// newLinesFromContent splits replacement text into lines the same way
// splitFileLines does, except an empty string means "zero lines" (delete the
// range) rather than "one empty line".
func newLinesFromContent(newContent string) []string {
	if newContent == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(newContent, "\n"), "\n")
}
