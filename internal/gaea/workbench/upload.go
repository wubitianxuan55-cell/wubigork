package workbench

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/wspath"
)

const (
	uploadIdleTTL      = 30 * time.Minute
	maxUploadBytes     = int64(2 << 30)
	maxUploadSessions  = 16
	maxUploadNameRunes = 255
)

// UploadManager owns chunked workspace uploads. Every session keeps an
// os.Root handle so a concurrent symlink swap cannot redirect the write.
type UploadManager struct {
	mu       sync.Mutex
	sessions map[string]*uploadSession
}

type uploadSession struct {
	root     *os.Root
	rootPath string
	file     *os.File
	relPath  string
	tempPath string
	size     int64
	last     time.Time
}

// UploadResult describes the published destination.
type UploadResult struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	IsDir  bool   `json:"isDir"`
	Exists bool   `json:"exists,omitempty"`
}

func NewUploadManager() *UploadManager {
	return &UploadManager{sessions: map[string]*uploadSession{}}
}

// Begin starts one upload below dir. relativePath may contain subdirectories
// for folder uploads, but every segment must be a legal base name.
func (m *UploadManager) Begin(root, dir, relativePath string) (string, error) {
	if m == nil {
		return "", fmt.Errorf("上传服务不可用")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	realRoot, err := wspath.RealPath(rootAbs)
	if err != nil {
		return "", err
	}
	relTarget, err := cleanUploadRelative(dir, relativePath)
	if err != nil {
		return "", err
	}
	rootHandle, err := os.OpenRoot(realRoot)
	if err != nil {
		return "", fmt.Errorf("打开工作区失败: %w", err)
	}
	parentRel := filepath.Dir(relTarget)
	if parentRel != "." {
		if err := rootHandle.MkdirAll(parentRel, 0o755); err != nil {
			_ = rootHandle.Close()
			return "", fmt.Errorf("创建上传目录失败: %w", err)
		}
	}
	if _, err := rootHandle.Lstat(relTarget); err == nil {
		_ = rootHandle.Close()
		return "", fmt.Errorf("目标已存在，请先重命名或删除: %s", filepath.ToSlash(relTarget))
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = rootHandle.Close()
		return "", fmt.Errorf("检查上传目标失败: %w", err)
	}
	var tokenBytes [12]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		_ = rootHandle.Close()
		return "", err
	}
	token := hex.EncodeToString(tokenBytes[:])
	tempRel := filepath.Join(parentRel, "."+token+".upload.tmp")
	if parentRel == "." {
		tempRel = "." + token + ".upload.tmp"
	}
	f, err := rootHandle.OpenFile(tempRel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		_ = rootHandle.Close()
		return "", fmt.Errorf("创建上传临时文件失败: %w", err)
	}
	now := time.Now()
	m.mu.Lock()
	m.sweepLocked(now)
	if len(m.sessions) >= maxUploadSessions {
		m.mu.Unlock()
		_ = f.Close()
		_ = rootHandle.Remove(tempRel)
		_ = rootHandle.Close()
		return "", fmt.Errorf("同时进行的上传过多（上限 %d）", maxUploadSessions)
	}
	m.sessions[token] = &uploadSession{
		root:     rootHandle,
		rootPath: realRoot,
		file:     f,
		relPath:  relTarget,
		tempPath: tempRel,
		last:     now,
	}
	m.mu.Unlock()
	return token, nil
}

func (m *UploadManager) Append(token, encoded string) error {
	if m == nil {
		return fmt.Errorf("上传服务不可用")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[token]
	if s == nil {
		return fmt.Errorf("上传会话不存在或已结束")
	}
	decodedLen := int64(base64.StdEncoding.DecodedLen(len(encoded)))
	if s.size+decodedLen > maxUploadBytes {
		return fmt.Errorf("上传文件过大（上限 2GB）")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("解码上传分片失败: %w", err)
	}
	n, err := s.file.Write(raw)
	if err == nil && n != len(raw) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return fmt.Errorf("写入上传分片失败: %w", err)
	}
	s.size += int64(n)
	s.last = time.Now()
	return nil
}

func (m *UploadManager) Finish(token string) (UploadResult, error) {
	if m == nil {
		return UploadResult{}, fmt.Errorf("上传服务不可用")
	}
	m.mu.Lock()
	s := m.sessions[token]
	delete(m.sessions, token)
	m.mu.Unlock()
	if s == nil {
		return UploadResult{}, fmt.Errorf("上传会话不存在或已结束")
	}
	_ = s.file.Close()
	defer func() { _ = s.root.Close() }()
	if s.size == 0 {
		_ = s.root.Remove(s.tempPath)
		return UploadResult{}, fmt.Errorf("不能上传空文件")
	}
	if err := s.root.Rename(s.tempPath, s.relPath); err != nil {
		_ = s.root.Remove(s.tempPath)
		return UploadResult{}, fmt.Errorf("发布上传文件失败: %w", err)
	}
	return UploadResult{
		Path: filepath.ToSlash(s.relPath),
		Name: filepath.Base(s.relPath),
		Size: s.size,
	}, nil
}

func (m *UploadManager) Abort(token string) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	s := m.sessions[token]
	delete(m.sessions, token)
	m.mu.Unlock()
	if s == nil {
		return nil
	}
	_ = s.file.Close()
	_ = s.root.Remove(s.tempPath)
	return s.root.Close()
}

func (m *UploadManager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = map[string]*uploadSession{}
	m.mu.Unlock()
	for _, s := range sessions {
		_ = s.file.Close()
		_ = s.root.Remove(s.tempPath)
		_ = s.root.Close()
	}
}

// ImportFiles copies files chosen by the native file dialog into targetDir.
// Existing destinations are refused so an upload cannot silently overwrite a
// workspace file.
func ImportFiles(root, targetDir string, sources []string) ([]UploadResult, error) {
	rootHandle, err := openRealRoot(root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rootHandle.Close() }()
	cleanDir, err := cleanWorkspaceDir(targetDir)
	if err != nil {
		return nil, err
	}
	if cleanDir != "." {
		if err := rootHandle.MkdirAll(cleanDir, 0o755); err != nil {
			return nil, fmt.Errorf("创建上传目录失败: %w", err)
		}
	}
	out := make([]UploadResult, 0, len(sources))
	cleanup := func() {
		for _, result := range out {
			_ = rootHandle.Remove(result.Path)
		}
	}
	for _, source := range sources {
		info, err := os.Lstat(source)
		if err != nil {
			cleanup()
			return out, fmt.Errorf("读取待上传文件失败: %w", err)
		}
		if !info.Mode().IsRegular() {
			cleanup()
			return out, fmt.Errorf("待上传项不是普通文件: %s", filepath.Base(source))
		}
		if err := ValidateName(filepath.Base(source)); err != nil {
			cleanup()
			return out, err
		}
		result, err := copyExternalIntoRoot(rootHandle, cleanDir, source, filepath.Base(source))
		if err != nil {
			cleanup()
			return out, err
		}
		out = append(out, result)
	}
	return out, nil
}

// ImportFolder copies one selected directory under targetDir while preserving
// its own name and internal layout. Symlink entries are skipped.
func ImportFolder(root, targetDir, sourceDir string) ([]UploadResult, error) {
	info, err := os.Lstat(sourceDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("待上传目录无效: %s", sourceDir)
	}
	base := filepath.Base(filepath.Clean(sourceDir))
	if err := ValidateName(base); err != nil {
		return nil, err
	}
	rootHandle, err := openRealRoot(root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rootHandle.Close() }()
	cleanDir, err := cleanWorkspaceDir(targetDir)
	if err != nil {
		return nil, err
	}
	destBase := filepath.Join(cleanDir, base)
	if _, err := rootHandle.Lstat(destBase); err == nil {
		return nil, fmt.Errorf("目标已存在，请先重命名或删除: %s", filepath.ToSlash(destBase))
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("检查上传目标失败: %w", err)
	}
	out := []UploadResult{}
	err = filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return rootHandle.MkdirAll(destBase, 0o755)
		}
		targetRel := filepath.Join(destBase, rel)
		if d.IsDir() {
			return rootHandle.MkdirAll(targetRel, 0o755)
		}
		result, err := copyExternalIntoRoot(rootHandle, filepath.Dir(targetRel), path, filepath.Base(targetRel))
		if err != nil {
			return err
		}
		out = append(out, result)
		return nil
	})
	if err != nil {
		_ = rootHandle.RemoveAll(destBase)
		return out, fmt.Errorf("上传文件夹失败: %w", err)
	}
	return out, nil
}

func openRealRoot(root string) (*os.Root, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	realRoot, err := wspath.RealPath(rootAbs)
	if err != nil {
		return nil, err
	}
	return os.OpenRoot(realRoot)
}

func cleanWorkspaceDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ".", nil
	}
	if filepath.IsAbs(dir) {
		return "", fmt.Errorf("上传目录必须是工作区相对路径")
	}
	dir = filepath.Clean(filepath.FromSlash(dir))
	if dir == "." {
		return ".", nil
	}
	if dir == ".." || strings.HasPrefix(dir, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("上传目录越出工作区")
	}
	for _, part := range strings.FieldsFunc(filepath.ToSlash(dir), func(r rune) bool { return r == '/' }) {
		if err := ValidateName(part); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func copyExternalIntoRoot(root *os.Root, dir, source, name string) (UploadResult, error) {
	targetRel := filepath.Join(dir, name)
	if dir == "." {
		targetRel = name
	}
	if _, err := root.Lstat(targetRel); err == nil {
		return UploadResult{}, fmt.Errorf("目标已存在，请先重命名或删除: %s", filepath.ToSlash(targetRel))
	} else if !errors.Is(err, os.ErrNotExist) {
		return UploadResult{}, fmt.Errorf("检查上传目标失败: %w", err)
	}
	src, err := os.Open(source)
	if err != nil {
		return UploadResult{}, fmt.Errorf("打开待上传文件失败: %w", err)
	}
	defer func() { _ = src.Close() }()
	var tokenBytes [8]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		return UploadResult{}, err
	}
	tmpRel := filepath.Join(dir, "."+hex.EncodeToString(tokenBytes[:])+".import.tmp")
	if dir == "." {
		tmpRel = "." + hex.EncodeToString(tokenBytes[:]) + ".import.tmp"
	}
	dst, err := root.OpenFile(tmpRel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return UploadResult{}, fmt.Errorf("创建导入临时文件失败: %w", err)
	}
	ok := false
	defer func() {
		_ = dst.Close()
		if !ok {
			_ = root.Remove(tmpRel)
		}
	}()
	n, err := io.Copy(dst, src)
	if err != nil {
		return UploadResult{}, fmt.Errorf("复制上传文件失败: %w", err)
	}
	if err := dst.Close(); err != nil {
		return UploadResult{}, fmt.Errorf("关闭导入文件失败: %w", err)
	}
	if err := root.Rename(tmpRel, targetRel); err != nil {
		return UploadResult{}, fmt.Errorf("发布上传文件失败: %w", err)
	}
	ok = true
	return UploadResult{Path: filepath.ToSlash(targetRel), Name: name, Size: n}, nil
}

func (m *UploadManager) sweepLocked(now time.Time) {
	for token, s := range m.sessions {
		if now.Sub(s.last) < uploadIdleTTL {
			continue
		}
		delete(m.sessions, token)
		_ = s.file.Close()
		_ = s.root.Remove(s.tempPath)
		_ = s.root.Close()
	}
}

func cleanUploadRelative(dir, relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", fmt.Errorf("上传文件名不能为空")
	}
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("上传文件名不能是绝对路径")
	}
	if strings.TrimSpace(dir) != "" && filepath.IsAbs(dir) {
		return "", fmt.Errorf("上传目录必须是工作区相对路径")
	}
	parts := strings.FieldsFunc(filepath.ToSlash(relativePath), func(r rune) bool { return r == '/' })
	if len(parts) == 0 {
		return "", fmt.Errorf("上传文件名不能为空")
	}
	for _, part := range parts {
		if len([]rune(part)) > maxUploadNameRunes {
			return "", fmt.Errorf("上传文件名过长: %s", part)
		}
		if err := ValidateName(part); err != nil {
			return "", err
		}
	}
	all := parts
	if strings.TrimSpace(dir) != "" {
		dir = filepath.Clean(filepath.FromSlash(strings.TrimSpace(dir)))
		if dir == "." || dir == ".." || strings.HasPrefix(dir, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("上传目录越出工作区")
		}
		all = append(strings.FieldsFunc(filepath.ToSlash(dir), func(r rune) bool { return r == '/' }), parts...)
		for _, part := range all[:len(all)-len(parts)] {
			if err := ValidateName(part); err != nil {
				return "", err
			}
		}
	}
	return filepath.Join(all...), nil
}
