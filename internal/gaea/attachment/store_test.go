package attachment

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const tinyPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "uploads"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSaveImageDataURL(t *testing.T) {
	s := openTestStore(t)
	path, err := s.SaveImageDataURL("data:image/png;base64," + tinyPNG)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filepath.Base(path), "image-") || !strings.HasSuffix(path, ".png") {
		t.Fatalf("image path = %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stored image not visible: %v", err)
	}
}

func TestSaveImageRejectsSpoofedMime(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.SaveImageDataURL("data:image/png;base64,aGk="); err == nil {
		t.Fatal("spoofed image should fail content sniffing")
	}
}

func TestLimitsAreCheckedBeforeBase64Decode(t *testing.T) {
	if err := RejectEncodedFileBeforeDecode(strings.Repeat("!", 36<<20)); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("oversize base64 check = %v, want ErrFileTooLarge", err)
	}
	s := openTestStore(t)
	_, err := s.SaveFileBase64("bad.bin", strings.Repeat("!", 36<<20))
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("SaveFileBase64 oversize = %v, want size error before decode", err)
	}
}

func TestSameTimestampPathsDoNotCollide(t *testing.T) {
	s := openTestStore(t)
	oldNow, oldRand := attachmentNow, attachmentRand
	attachmentNow = func() time.Time { return time.Date(2026, 10, 23, 1, 2, 3, 0, time.UTC) }
	attachmentRand = func(p []byte) (int, error) {
		for i := range p {
			p[i] = 1
		}
		return len(p), nil
	}
	t.Cleanup(func() { attachmentNow, attachmentRand = oldNow, oldRand })

	raw, err := base64.StdEncoding.DecodeString(tinyPNG)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.SaveImageBytes("image/png", raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.SaveImageBytes("image/png", raw)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("same-timestamp paths collided: %q", first)
	}
}

func TestSaveFileSanitizesNameAndStaysInRoot(t *testing.T) {
	s := openTestStore(t)
	path, err := s.SaveFileBase64("../../escape.PDF", base64.StdEncoding.EncodeToString([]byte("payload")))
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(s.rootPath, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("stored path escaped root: %q (%v)", path, err)
	}
	if filepath.Base(path) == "escape.PDF" || !strings.HasSuffix(path, ".pdf") {
		t.Fatalf("file name was not prefixed/sanitized: %q", path)
	}
}

func TestOpenRejectsSymlinkRoot(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "uploads")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Open(link); err == nil {
		t.Fatal("symlink upload root should be rejected")
	}
}

func TestImageDataURLRejectsEscapeAndSymlink(t *testing.T) {
	s := openTestStore(t)
	outside := filepath.Join(filepath.Dir(s.rootPath), "secret.png")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImageDataURL(outside); err == nil {
		t.Fatal("outside path should be rejected")
	}
	link := filepath.Join(s.rootPath, "link.png")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := s.ImageDataURL(link); err == nil {
		t.Fatal("symlink escape should be rejected by os.Root")
	}
}
