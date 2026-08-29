package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/tool"

	"github.com/gaea/gaea/internal/gaea/evidence"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
)

func init() { tool.RegisterBuiltin(writeFile{}) }

// writeFile writes a file. roots, when non-empty, confines the target to the
// workspace (see confine); the zero value registered at init is unconfined and
// is overridden per run by ConfineWriters. workDir, when non-empty, is the
// directory a relative path resolves against (see resolveIn).
type writeFile struct {
	roots   []string
	workDir string
}

func (writeFile) Name() string { return "write_file" }

func (writeFile) Description() string {
	return "Write content to a file at the given path (overwriting existing content). Creates parent directories as needed."
}

func (writeFile) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"File path"},"content":{"type":"string","description":"Full content to write"}},"required":["path","content"]}`)
}

func (writeFile) ReadOnly() bool { return false }

func (writeFile) CompactDescription() string     { return compactDesc["write_file"] }
func (writeFile) CompactSchema() json.RawMessage { return compactSchema["write_file"] }

func (w writeFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("path is required")
	}
	p.Path = resolveIn(w.workDir, p.Path)
	if err := confine(w.roots, p.Path); err != nil {
		return "", err
	}
	if dir := filepath.Dir(p.Path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	// Preserve original file permissions.
	mode := os.FileMode(0o644)
	enc := fileenc.UTF8 // new files default to UTF-8
	before := ""        // v4.1 证据链：旧文件原文（写前捕获，仅旧文件存在时）
	if fi, err := os.Stat(p.Path); err == nil {
		mode = fi.Mode().Perm()
		// Preserve the original encoding when overwriting an existing file.
		if oldContent, existingEnc, err := readFileEncoded(p.Path); err == nil {
			enc = existingEnc
			before = oldContent
		}
	}
	if err := writeFileEncoded(p.Path, p.Content, enc, mode); err != nil {
		return "", fmt.Errorf("write %s: %w", p.Path, err)
	}
	// v4.1 证据链：整文件覆盖的 Before/After 原文摘要（Before 仅旧文件存在时）。
	evidence.RecordChange(ctx, evidence.ChangeRecord{
		Tool:          "write_file",
		Target:        p.Path,
		BeforeSummary: before,
		AfterSummary:  p.Content,
	})
	return fmt.Sprintf("wrote %d bytes to %s", len(p.Content), p.Path), nil
}
