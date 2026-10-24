package tool

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPathAccessPolicySeparatesReadAndWriteRoots(t *testing.T) {
	workspace := t.TempDir()
	writeRoot := t.TempDir()
	readRoot := t.TempDir()
	outside := t.TempDir()

	policy := NewPathAccessPolicy(workspace, []string{readRoot}, []string{writeRoot}, nil)
	ctx := context.Background()

	for _, path := range []string{
		filepath.Join(workspace, "inside.txt"),
		filepath.Join(writeRoot, "write.txt"),
		filepath.Join(readRoot, "read.txt"),
	} {
		if err := policy.CheckRead(ctx, path); err != nil {
			t.Errorf("CheckRead(%q) = %v, want allowed", path, err)
		}
	}
	if err := policy.CheckWrite(ctx, filepath.Join(readRoot, "write.txt")); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("allow_read must not extend writes: %v", err)
	}
	if err := policy.CheckWrite(ctx, filepath.Join(writeRoot, "write.txt")); err != nil {
		t.Fatalf("allow_write path should be writable: %v", err)
	}
	if err := policy.CheckRead(ctx, filepath.Join(outside, "secret.txt")); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("outside read = %v, want ErrAccessDenied", err)
	}
	if err := policy.CheckWrite(ctx, filepath.Join(outside, "secret.txt")); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("outside write = %v, want ErrAccessDenied", err)
	}
}

func TestPathAccessPolicyExplicitSelectedFileIsReadOnly(t *testing.T) {
	workspace := t.TempDir()
	selectedDir := t.TempDir()
	selected := filepath.Join(selectedDir, "picked.pdf")
	sibling := filepath.Join(selectedDir, "private.pdf")

	policy := NewPathAccessPolicy(workspace, nil, nil, func(path string) bool {
		return filepath.Clean(path) == filepath.Clean(selected)
	})
	ctx := context.Background()
	if err := policy.CheckRead(ctx, selected); err != nil {
		t.Fatalf("selected file should be readable: %v", err)
	}
	if err := policy.CheckPreview(ctx, selected); err != nil {
		t.Fatalf("selected file should be previewable: %v", err)
	}
	if err := policy.CheckWrite(ctx, selected); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("selected file must not become writable: %v", err)
	}
	if err := policy.CheckRead(ctx, sibling); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("sibling outside the selection must stay denied: %v", err)
	}
}

func TestPathAccessPolicyRejectsSymlinkEscape(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(workspace, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation unavailable on this host: %v", err)
	}

	policy := NewPathAccessPolicy(workspace, nil, nil, nil)
	if err := policy.CheckRead(context.Background(), filepath.Join(link, "secret.txt")); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("symlink escape read = %v, want ErrAccessDenied", err)
	}
	if err := policy.CheckWrite(context.Background(), filepath.Join(link, "new.txt")); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("symlink escape write = %v, want ErrAccessDenied", err)
	}
}

func TestPathAccessPolicyRejectsJunctionEscape(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("directory junctions are Windows-specific")
	}
	workspace := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(workspace, "junction")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput(); err != nil {
		t.Skipf("junction creation unavailable: %v (%s)", err, out)
	}

	policy := NewPathAccessPolicy(workspace, nil, nil, nil)
	if err := policy.CheckRead(context.Background(), filepath.Join(link, "secret.txt")); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("junction escape read = %v, want ErrAccessDenied", err)
	}
	if err := policy.CheckWrite(context.Background(), filepath.Join(link, "new.txt")); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("junction escape write = %v, want ErrAccessDenied", err)
	}
}

func TestPathAccessPolicyCaseAndTraversal(t *testing.T) {
	workspace := t.TempDir()
	inside := filepath.Join(workspace, "sub", "file.txt")
	policy := NewPathAccessPolicy(workspace, nil, nil, nil)
	if err := policy.CheckRead(context.Background(), inside); err != nil {
		t.Fatalf("inside read: %v", err)
	}
	if runtime.GOOS == "windows" {
		if err := policy.CheckRead(context.Background(), strings.ToUpper(inside)); err != nil {
			t.Fatalf("Windows path matching must be case-insensitive: %v", err)
		}
		if err := policy.CheckRead(context.Background(), `Z:\gaea-access-outside.txt`); !errors.Is(err, ErrAccessDenied) {
			t.Fatalf("cross-volume path = %v, want ErrAccessDenied", err)
		}
	}
	escape := filepath.Join(workspace, "..", "outside.txt")
	if err := policy.CheckRead(context.Background(), escape); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf(".. escape = %v, want ErrAccessDenied", err)
	}
}

func TestDenyAccessPolicyFailsClosed(t *testing.T) {
	policy := NewDenyAccessPolicy()
	ctx := context.Background()
	for name, check := range map[string]func(context.Context, string) error{
		"read":    policy.CheckRead,
		"preview": policy.CheckPreview,
		"write":   policy.CheckWrite,
	} {
		if err := check(ctx, filepath.Join(t.TempDir(), "x")); !errors.Is(err, ErrAccessDenied) {
			t.Errorf("%s = %v, want ErrAccessDenied", name, err)
		}
	}
}
