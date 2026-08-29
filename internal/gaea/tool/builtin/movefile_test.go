package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMoveFileBasic(t *testing.T) {
	dir := t.TempDir()
	src := writeTemp(t, dir, "a.txt", "payload\n")
	dst := filepath.Join(dir, "b.txt")

	out := execTool(t, moveFile{workDir: dir}, map[string]any{
		"source": "a.txt", "destination": "b.txt",
	})
	if !strings.Contains(out, "moved "+src+" → "+dst) {
		t.Errorf("output = %q", out)
	}
	if content, err := os.ReadFile(dst); err != nil || string(content) != "payload\n" {
		t.Errorf("destination = %q err=%v", content, err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source must be gone, stat err=%v", err)
	}
}

func TestMoveFileCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "payload\n")
	dst := filepath.Join(dir, "nested", "deep", "b.txt")

	execTool(t, moveFile{workDir: dir}, map[string]any{
		"source": "a.txt", "destination": dst,
	})
	if content, err := os.ReadFile(dst); err != nil || string(content) != "payload\n" {
		t.Errorf("destination = %q err=%v", content, err)
	}
}

func TestMoveFileDestinationExists(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "src.txt", "new content\n")
	writeTemp(t, dir, "dst.txt", "old content\n")

	// Without overwrite: refuse, leave both files intact.
	_, err := moveFile{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"source": "src.txt", "destination": "dst.txt",
	}))
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v, want destination-exists refusal", err)
	}
	if content, _, _ := readFileEncoded(filepath.Join(dir, "dst.txt")); content != "old content\n" {
		t.Errorf("destination must be untouched without overwrite, got %q", content)
	}
	if content, _, _ := readFileEncoded(filepath.Join(dir, "src.txt")); content != "new content\n" {
		t.Errorf("source must be untouched without overwrite, got %q", content)
	}

	// With overwrite: replace.
	execTool(t, moveFile{workDir: dir}, map[string]any{
		"source": "src.txt", "destination": "dst.txt", "overwrite": true,
	})
	if content, _, _ := readFileEncoded(filepath.Join(dir, "dst.txt")); content != "new content\n" {
		t.Errorf("destination = %q, want overwrite", content)
	}
	if _, err := os.Stat(filepath.Join(dir, "src.txt")); !os.IsNotExist(err) {
		t.Errorf("source must be gone after overwrite move, stat err=%v", err)
	}
}

func TestMoveFileDestinationIsDirRejected(t *testing.T) {
	dir := t.TempDir()
	writeTemp(t, dir, "a.txt", "x\n")
	dstDir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := moveFile{workDir: dir}.Execute(context.Background(), mustArgs(t, map[string]any{
		"source": "a.txt", "destination": "subdir", "overwrite": true,
	}))
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("err = %v, want directory refusal (even with overwrite)", err)
	}
}

func TestMoveFileSourceValidation(t *testing.T) {
	dir := t.TempDir()
	mover := moveFile{workDir: dir}
	if _, err := mover.Execute(context.Background(), mustArgs(t, map[string]any{
		"source": "missing.txt", "destination": "out.txt",
	})); err == nil || !strings.Contains(err.Error(), "source not found") {
		t.Errorf("missing source: err=%v", err)
	}
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := mover.Execute(context.Background(), mustArgs(t, map[string]any{
		"source": "subdir", "destination": "out.txt",
	})); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("directory source: err=%v", err)
	}
}

func TestMoveFileConfinedBothEndpoints(t *testing.T) {
	outside := t.TempDir()
	ws := t.TempDir()
	mover := moveFile{workDir: ws, roots: []string{ws}}
	outsideFile := writeTemp(t, outside, "secret.txt", "x\n")
	writeTemp(t, ws, "local.txt", "y\n")

	// Source outside the roots.
	if _, err := mover.Execute(context.Background(), mustArgs(t, map[string]any{
		"source": outsideFile, "destination": "local2.txt",
	})); err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Errorf("source outside roots: err=%v", err)
	}
	// Destination outside the roots.
	if _, err := mover.Execute(context.Background(), mustArgs(t, map[string]any{
		"source": "local.txt", "destination": outsideFile,
	})); err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Errorf("destination outside roots: err=%v", err)
	}
}

func TestMoveFileSymlinkEscapeBlocked(t *testing.T) {
	outside := t.TempDir()
	ws := t.TempDir()
	target := writeTemp(t, outside, "target.txt", "x\n")
	link := filepath.Join(ws, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}
	_, err := moveFile{workDir: ws, roots: []string{ws}}.Execute(context.Background(), mustArgs(t, map[string]any{
		"source": "link.txt", "destination": "moved.txt",
	}))
	if err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Fatalf("moving through a symlink must be confined, err=%v", err)
	}
}

func TestCopyFileContentsFallback(t *testing.T) {
	// White-box: the cross-volume fallback (copy+remove) preserves content
	// and permission bits.
	dir := t.TempDir()
	src := writeTemp(t, dir, "src.txt", "copy me\n")
	before, err := os.Stat(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst.txt")

	if err := copyFileContents(src, dst, before.Mode().Perm()); err != nil {
		t.Fatalf("copyFileContents: %v", err)
	}
	if content, err := os.ReadFile(dst); err != nil || string(content) != "copy me\n" {
		t.Errorf("dst = %q err=%v", content, err)
	}
	after, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm() != before.Mode().Perm() {
		t.Errorf("perm: before %v, after %v", before.Mode().Perm(), after.Mode().Perm())
	}
}
