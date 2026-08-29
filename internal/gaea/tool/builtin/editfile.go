package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(editFile{}) }

// editFile replaces an exact old_string with new_string inside a file.
// roots, when non-empty, confines the target to the workspace (see confine);
// the zero value registered at init is unconfined and is overridden per run by
// ConfineWriters. workDir, when non-empty, is the directory a relative path
// resolves against (see resolveIn).
type editFile struct {
	roots   []string
	workDir string
}

func (editFile) Name() string { return "edit_file" }

func (editFile) Description() string {
	return "Replace an exact old_string with new_string in a file. new_string may be an empty string to delete. Fails when old_string is missing or matches more than once (pass replace_all:true for all occurrences). Encoding (GB18030/UTF-16) and file permissions are preserved."
}

func (editFile) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"File path"},
  "old_string":{"type":"string","description":"Exact text to replace (must be non-empty and unique unless replace_all)"},
  "new_string":{"type":"string","description":"Replacement text (empty string deletes old_string)"},
  "replace_all":{"type":"boolean","description":"Replace every occurrence (default false)"}
},
"required":["path","old_string","new_string"]
}`)
}

func (editFile) ReadOnly() bool { return false }

func (editFile) CompactDescription() string     { return compactDesc["edit_file"] }
func (editFile) CompactSchema() json.RawMessage { return compactSchema["edit_file"] }

func (e editFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path       string `json:"path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	if p.OldString == "" {
		return "", fmt.Errorf("old_string is required and must be non-empty (pass new_string:\"\" to delete text, or use write_file to create a file)")
	}
	p.Path = resolveIn(e.workDir, p.Path)
	if err := confine(e.roots, p.Path); err != nil {
		return "", err
	}
	content, enc, err := readFileEncoded(p.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", p.Path, err)
	}
	newContent, n, err := replaceInContent(content, p.OldString, p.NewString, p.ReplaceAll)
	if err != nil {
		return "", fmt.Errorf("edit %s: %w", p.Path, err)
	}
	if err := atomicWriteEncoded(p.Path, newContent, enc); err != nil {
		return "", fmt.Errorf("write %s: %w", p.Path, err)
	}
	// v4.1 证据链：记录本次替换的原文/新文摘要（old_string/new_string 即变更
	// 区域的原文摘要，非展示截断文本）。ctx 无台账时静默跳过。
	evidence.RecordChange(ctx, evidence.ChangeRecord{
		Tool:          "edit_file",
		Target:        p.Path,
		BeforeSummary: p.OldString,
		AfterSummary:  p.NewString,
	})
	return fmt.Sprintf("edited %s: %d occurrence(s) replaced (+%d/-%d bytes)",
		p.Path, n, n*len(p.NewString), n*len(p.OldString)), nil
}

// ── shared replacement / write kernel (edit_file + multi_edit + edit_lines) ──

// replaceInContent replaces old with new inside content and returns the new
// content plus the number of replacements made. With replaceAll=false it
// demands exactly one match: zero occurrences is an error (with a CRLF
// diagnostic, the dominant cause of phantom misses on Windows files), and
// more than one is an error telling the model to widen the context or opt
// into replace_all. edit_file and multi_edit both route through here so the
// two tools cannot drift apart on match semantics.
func replaceInContent(content, old, new string, replaceAll bool) (string, int, error) {
	n := strings.Count(content, old)
	if n == 0 {
		if strings.Contains(content, strings.ReplaceAll(old, "\n", "\r\n")) {
			return "", 0, fmt.Errorf(
				"old_string not found: the file uses CRLF (Windows) line endings — " +
					"your old_string has bare \\n. Re-read the file and copy the text exactly, " +
					"or edit line-wise with edit_lines")
		}
		return "", 0, fmt.Errorf(
			"old_string not found in file (%d occurrences). Re-read the file with read_file "+
				"and retry with the exact current content", n)
	}
	if n > 1 && !replaceAll {
		return "", 0, fmt.Errorf(
			"old_string matches %d locations; include more surrounding context to make it unique, "+
				"or pass replace_all:true to replace all %d occurrences", n, n)
	}
	if replaceAll {
		return strings.ReplaceAll(content, old, new), n, nil
	}
	return strings.Replace(content, old, new, 1), 1, nil
}

// atomicWriteEncoded encodes content back to the file's original encoding and
// writes it atomically (temp file + rename), preserving the original file's
// permission bits. Missing files get UTF-8 / 0644. readFileEncoded/Encode stay
// paired here so a GB18030 or UTF-16 file is never silently re-coded.
func atomicWriteEncoded(path, content string, enc fileenc.Kind) error {
	perm := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	return fileutil.AtomicWrite(path, fileenc.Encode(content, enc), perm)
}
