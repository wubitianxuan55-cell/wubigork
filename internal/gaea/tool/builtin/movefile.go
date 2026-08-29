package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(moveFile{}) }

// moveFile moves/renames a file. Both endpoints must be inside the write
// roots. The primary path is os.Rename; when that fails (the classic case is
// a cross-volume move, which rename cannot do) it falls back to
// copy-then-remove, preserving content and permission bits. After a move both
// the source and the destination path are stale — the agent layer invalidates
// both cache entries (execute_one.go move_file branch).
type moveFile struct {
	roots   []string
	workDir string
}

func (moveFile) Name() string { return "move_file" }

func (moveFile) Description() string {
	return "Move or rename a file. Creates the destination's parent directories as needed. Fails when the destination exists unless overwrite:true is passed. Works across volumes (falls back to copy+remove when rename can't)."
}

func (moveFile) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "source":{"type":"string","description":"Path of the file to move"},
  "destination":{"type":"string","description":"New path for the file"},
  "overwrite":{"type":"boolean","description":"Replace the destination if it exists (default false)"}
},
"required":["source","destination"]
}`)
}

func (moveFile) ReadOnly() bool { return false }

func (moveFile) CompactDescription() string     { return compactDesc["move_file"] }
func (moveFile) CompactSchema() json.RawMessage { return compactSchema["move_file"] }

func (m moveFile) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Overwrite   bool   `json:"overwrite"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Source == "" {
		return "", fmt.Errorf("source is required")
	}
	if p.Destination == "" {
		return "", fmt.Errorf("destination is required")
	}
	src := resolveIn(m.workDir, p.Source)
	dst := resolveIn(m.workDir, p.Destination)
	// Both endpoints are writes — each must sit inside the workspace roots.
	if err := confine(m.roots, src); err != nil {
		return "", err
	}
	if err := confine(m.roots, dst); err != nil {
		return "", err
	}
	srcFi, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("move %s: source not found: %w", src, err)
	}
	if !srcFi.Mode().IsRegular() {
		return "", fmt.Errorf("move %s: source is not a regular file (directories are not supported)", src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	if dstFi, err := os.Lstat(dst); err == nil {
		if dstFi.IsDir() {
			return "", fmt.Errorf("move %s: destination %s is a directory", src, dst)
		}
		if !p.Overwrite {
			return "", fmt.Errorf("move %s: destination %s already exists (pass overwrite:true to replace it)", src, dst)
		}
		// Remove the target first: os.Rename onto an existing file fails on
		// Windows, and the copy fallback below must not merge with it either.
		if err := os.Remove(dst); err != nil {
			return "", fmt.Errorf("move %s: replace existing destination %s: %w", src, dst, err)
		}
	}
	if err := os.Rename(src, dst); err != nil {
		// Rename fails across volumes (EXDEV) and in a few other cases —
		// fall back to copy+remove so a move still happens.
		if copyErr := copyFileContents(src, dst, srcFi.Mode().Perm()); copyErr != nil {
			return "", fmt.Errorf("move %s → %s: rename: %v; copy fallback: %w", src, dst, err, copyErr)
		}
		if rmErr := os.Remove(src); rmErr != nil {
			return "", fmt.Errorf("move %s → %s: copied but could not remove source: %w", src, dst, rmErr)
		}
	}
	// v4.1 证据链：记录移动映射（Target=目标路径；Before/After 标注来源→去向）。
	evidence.RecordChange(ctx, evidence.ChangeRecord{
		Tool:          "move_file",
		Target:        dst,
		BeforeSummary: "→ moved from " + src,
		AfterSummary:  dst,
	})
	return fmt.Sprintf("moved %s → %s", src, dst), nil
}

// copyFileContents copies src to dst with the given permission bits. Used as
// the cross-volume fallback of move_file (os.Rename cannot cross filesystems).
// The file is created writable and chmod'ed afterwards: creating it directly
// with a read-only perm would block the very writes that fill it (Windows
// applies the read-only attribute at creation time).
func copyFileContents(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst) // don't leave a partial copy behind
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}
	if err := os.Chmod(dst, perm); err != nil {
		return err
	}
	return nil
}
