package attachment

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/gaea/wspath"
)

const maxCreateAttempts = 1000

var (
	attachmentNow  = time.Now
	attachmentRand = rand.Read
	pathSeq        atomic.Uint64
)

// Store confines every attachment operation to one fixed upload directory.
// os.Root supplies a directory handle and prevents lexical traversal and
// symlink/junction escape even if directory entries are replaced concurrently.
type Store struct {
	rootPath string
	root     *os.Root
}

// Open ensures rootPath is a real directory (not a symlink/junction), then opens
// it as an os.Root for all subsequent operations.
func Open(rootPath string) (*Store, error) {
	if strings.TrimSpace(rootPath) == "" {
		return nil, fmt.Errorf("attachment root is empty")
	}
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		return nil, fmt.Errorf("create attachment root: %w", err)
	}
	abs, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	real, err := wspath.RealPath(abs)
	if err != nil {
		return nil, err
	}
	if !samePath(abs, real) {
		return nil, fmt.Errorf("attachment root must not be a symlink or junction")
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fmt.Errorf("open attachment root: %w", err)
	}
	return &Store{rootPath: abs, root: root}, nil
}

func (s *Store) Close() error {
	if s == nil || s.root == nil {
		return nil
	}
	return s.root.Close()
}

// SaveImageDataURL decodes and stores a data URL after enforcing the image limit
// on the encoded payload length.
func (s *Store) SaveImageDataURL(dataURL string) (string, error) {
	const prefix = "data:"
	const marker = ";base64,"
	if !strings.HasPrefix(dataURL, prefix) {
		return "", fmt.Errorf("unsupported image data URL")
	}
	i := strings.Index(dataURL, marker)
	if i <= len(prefix) {
		return "", fmt.Errorf("unsupported image data URL")
	}
	mime := strings.ToLower(dataURL[len(prefix):i])
	encoded := dataURL[i+len(marker):]
	if err := rejectEncodedImageBeforeDecode(encoded); err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode image attachment: %w", err)
	}
	return s.SaveImageBytes(mime, raw)
}

func (s *Store) SaveImageBytes(declaredMime string, raw []byte) (string, error) {
	if int64(len(raw)) > MaxImageBytes {
		return "", ErrImageTooLarge
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("image attachment is empty")
	}
	mime := detectedImageMime(raw)
	if mime == "" {
		return "", fmt.Errorf("attachment data is not a supported image")
	}
	if declaredMime != "" && imageExt(declaredMime) == "" {
		return "", fmt.Errorf("unsupported image type: %s", declaredMime)
	}
	return s.writeOne("image", imageExt(mime), raw)
}

// SaveImageFile copies one explicit source file after symlink/identity and size
// checks. The destination is written through the fixed root.
func (s *Store) SaveImageFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("image source must not be a symlink")
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > MaxImageBytes {
		return "", ErrImageTooLarge
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	opened, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !os.SameFile(info, opened) {
		return "", fmt.Errorf("image source changed while opening")
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxImageBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(raw)) > MaxImageBytes {
		return "", ErrImageTooLarge
	}
	return s.SaveImageBytes("", raw)
}

// SaveFileBase64 enforces the 25 MiB limit before base64 decoding.
func (s *Store) SaveFileBase64(fileName, encoded string) (string, error) {
	if err := RejectEncodedFileBeforeDecode(encoded); err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode file attachment: %w", err)
	}
	return s.SaveFileBytes(fileName, raw)
}

func (s *Store) SaveFileBytes(fileName string, raw []byte) (string, error) {
	if int64(len(raw)) > MaxFileBytes {
		return "", ErrFileTooLarge
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("file attachment is empty")
	}
	return s.writeOne("file", safeFileExt(fileName), raw)
}

// ImageDataURL reads only a regular image beneath the fixed root.
func (s *Store) ImageDataURL(path string) (string, error) {
	rel, err := s.relative(path)
	if err != nil {
		return "", err
	}
	info, err := s.root.Stat(rel)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > MaxImageBytes {
		return "", ErrImageTooLarge
	}
	f, err := s.root.Open(rel)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	raw, err := io.ReadAll(io.LimitReader(f, MaxImageBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(raw)) > MaxImageBytes {
		return "", ErrImageTooLarge
	}
	mime := detectedImageMime(raw)
	if mime == "" {
		return "", fmt.Errorf("attachment is not an image")
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func (s *Store) relative(path string) (string, error) {
	p := filepath.Clean(filepath.FromSlash(path))
	if filepath.IsAbs(p) {
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		if !wspath.Within(s.rootPath, abs) {
			return "", fmt.Errorf("attachment path is outside upload root")
		}
		p, err = filepath.Rel(s.rootPath, abs)
		if err != nil {
			return "", err
		}
	}
	if p == "." || p == "" || p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("attachment path is outside upload root")
	}
	return p, nil
}

func (s *Store) writeOne(prefix, ext string, raw []byte) (string, error) {
	if s == nil || s.root == nil {
		return "", fmt.Errorf("attachment store is closed")
	}
	for range maxCreateAttempts {
		token, err := attachmentToken()
		if err != nil {
			return "", err
		}
		tmp := "." + token + ".tmp"
		final := prefix + "-" + token + ext
		f, err := s.root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("create attachment temp: %w", err)
		}
		ok := false
		defer func() {
			if !ok {
				_ = s.root.Remove(tmp)
			}
		}()
		n, werr := f.Write(raw)
		if werr == nil && n != len(raw) {
			werr = io.ErrShortWrite
		}
		if werr == nil {
			werr = f.Close()
		} else {
			_ = f.Close()
		}
		if werr != nil {
			return "", fmt.Errorf("write attachment: %w", werr)
		}
		if err := s.root.Rename(tmp, final); err != nil {
			return "", fmt.Errorf("publish attachment: %w", err)
		}
		ok = true
		return filepath.Join(s.rootPath, final), nil
	}
	return "", fmt.Errorf("create unique attachment path")
}

func attachmentToken() (string, error) {
	buf := make([]byte, 6)
	if _, err := attachmentRand(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%06d-%s",
		attachmentNow().Format("20060102-150405.000000"), pathSeq.Add(1), hex.EncodeToString(buf)), nil
}

func safeFileExt(name string) string {
	ext := strings.ToLower(filepath.Ext(filepath.Base(name)))
	if len(ext) < 2 || len(ext) > 16 {
		return ".bin"
	}
	for _, r := range ext[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ".bin"
		}
	}
	return ext
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
