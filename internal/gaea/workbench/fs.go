// Package workbench provides the file-system authority for the desktop
// workbench. Rendering stays in React; path resolution, mutations, uploads,
// watching and optimistic-concurrency checks live here.
package workbench

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/filefilter"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// Entry is one directory row. Symlink follows the link only to classify its
// target; Broken marks a dangling link so the UI can report it honestly.
type Entry struct {
	Name       string `json:"name"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
	ModTime    int64  `json:"modTime"`
	IsSymlink  bool   `json:"isSymlink,omitempty"`
	Broken     bool   `json:"broken,omitempty"`
	LinkTarget string `json:"linkTarget,omitempty"`
	Compact    bool   `json:"compact,omitempty"`
}

// Level is one directory listing in a batch response. A per-level error keeps
// one unreadable directory from blanking the whole tree.
type Level struct {
	Path    string  `json:"path"`
	Entries []Entry `json:"entries"`
	Error   string  `json:"error,omitempty"`
}

// MutationResult identifies the created or renamed row.
type MutationResult struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
}

// Kind reports whether rel names a directory.
func Kind(root, rel string) (bool, error) {
	abs, _, err := resolveWithin(root, rel)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return false, fmt.Errorf("读取路径信息失败: %w", err)
	}
	return info.IsDir(), nil
}

// ListBatch lists every path in one call with one shared exclusion matcher.
// Paths are workspace-relative and use either slash spelling; "" is the root.
func ListBatch(root string, paths []string, exclude []string) []Level {
	matcher := filefilter.Compile(exclude)
	out := make([]Level, 0, len(paths))
	for _, rel := range paths {
		entries, err := listDirWithMatcher(root, rel, matcher)
		level := Level{Path: slashRel(rel), Entries: entries}
		if level.Entries == nil {
			level.Entries = []Entry{}
		}
		if err != nil {
			level.Error = err.Error()
		}
		out = append(out, level)
	}
	return out
}

// ListDir lists one directory. The lexical path is confined to root; symlink
// rows are classified from their targets and never followed for traversal.
func ListDir(root, rel string) ([]Entry, error) {
	return ListDirFiltered(root, rel, nil)
}

// ListDirFiltered applies the shared default and user exclusion policy.
func ListDirFiltered(root, rel string, exclude []string) ([]Entry, error) {
	return listDirWithMatcher(root, rel, filefilter.Compile(exclude))
}

func listDirWithMatcher(root, rel string, matcher *filefilter.Matcher) ([]Entry, error) {
	abs, cleanRel, err := resolveWithin(root, rel)
	if err != nil {
		return nil, err
	}
	cleanSlash := slashRel(cleanRel)
	if cleanSlash != "" && matcher.Match(cleanSlash, filepath.Base(cleanRel)) {
		return []Entry{}, nil
	}
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("目录不存在: %s", slashRel(cleanRel))
		}
		return nil, fmt.Errorf("读取目录失败: %s（%w）", slashRel(cleanRel), err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("不是目录: %s", slashRel(cleanRel))
	}
	rows, err := os.ReadDir(abs)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %s（%w）", slashRel(cleanRel), err)
	}
	out := make([]Entry, 0, len(rows))
	for _, row := range rows {
		entry := Entry{Name: row.Name()}
		li, lerr := os.Lstat(filepath.Join(abs, row.Name()))
		if lerr == nil {
			entry.ModTime = li.ModTime().UnixMilli()
			entry.IsSymlink = li.Mode()&os.ModeSymlink != 0
			if !entry.IsSymlink {
				entry.IsDir = li.IsDir()
				entry.Size = li.Size()
			}
		}
		if entry.IsSymlink {
			target := filepath.Join(abs, row.Name())
			if link, linkErr := os.Readlink(target); linkErr == nil {
				entry.LinkTarget = filepath.ToSlash(link)
			}
			ti, statErr := os.Stat(target)
			if statErr != nil {
				entry.Broken = true
			} else {
				entry.IsDir = ti.IsDir()
				entry.Size = ti.Size()
				entry.ModTime = ti.ModTime().UnixMilli()
			}
		}
		childRel := slashRel(filepath.Join(cleanRel, row.Name()))
		if matcher.Match(childRel, row.Name()) {
			continue
		}
		if entry.IsDir && !entry.IsSymlink {
			entry.Compact = compactSingleDirectory(filepath.Join(abs, row.Name()), childRel, matcher)
		}
		out = append(out, entry)
	}
	return out, nil
}

// compactSingleDirectory reports whether dir contains exactly one visible
// non-symlink directory. It reads incrementally and stops at the second visible
// row, so a huge multi-child directory never pays for a full duplicate listing.
func compactSingleDirectory(dir, rel string, matcher *filefilter.Matcher) bool {
	f, err := os.Open(dir)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	var only os.DirEntry
	visible := 0
	for {
		rows, readErr := f.ReadDir(16)
		for _, row := range rows {
			childRel := slashRel(filepath.Join(filepath.FromSlash(rel), row.Name()))
			if matcher.Match(childRel, row.Name()) {
				continue
			}
			visible++
			if visible > 1 {
				return false
			}
			only = row
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return false
		}
	}
	if visible != 1 || only == nil {
		return false
	}
	info, err := os.Lstat(filepath.Join(dir, only.Name()))
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

// Rename changes one row's base name without moving it across directories.
func Rename(root, rel, name string) (MutationResult, error) {
	abs, cleanRel, err := resolveWithin(root, rel)
	if err != nil {
		return MutationResult{}, err
	}
	if cleanRel == "" {
		return MutationResult{}, fmt.Errorf("不能重命名工作区根目录")
	}
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return MutationResult{}, err
	}
	parentAbs := filepath.Dir(abs)
	if err := ensureExistingDirWithin(root, parentAbs); err != nil {
		return MutationResult{}, err
	}
	dst := filepath.Join(parentAbs, name)
	if _, err := os.Lstat(dst); err == nil {
		return MutationResult{}, fmt.Errorf("目标已存在: %s", name)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return MutationResult{}, fmt.Errorf("检查目标失败: %w", err)
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return MutationResult{}, fmt.Errorf("源路径不存在: %s", slashRel(cleanRel))
	}
	if err := fileutil.RenameWithRetry(abs, dst); err != nil {
		return MutationResult{}, fmt.Errorf("重命名失败: %w", err)
	}
	newRel := filepath.Join(filepath.Dir(cleanRel), name)
	if cleanRel == filepath.Base(cleanRel) {
		newRel = name
	}
	return MutationResult{Path: slashRel(newRel), Name: name, IsDir: info.IsDir()}, nil
}

// CreateFolder creates one directory below an existing directory row.
func CreateFolder(root, parent, name string) (MutationResult, error) {
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return MutationResult{}, err
	}
	parentAbs, cleanParent, err := resolveWithin(root, parent)
	if err != nil {
		return MutationResult{}, err
	}
	if err := ensureExistingDirWithin(root, parentAbs); err != nil {
		return MutationResult{}, err
	}
	dst := filepath.Join(parentAbs, name)
	if err := os.Mkdir(dst, 0o755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return MutationResult{}, fmt.Errorf("目标已存在: %s", name)
		}
		return MutationResult{}, fmt.Errorf("创建目录失败: %w", err)
	}
	return MutationResult{Path: slashRel(filepath.Join(cleanParent, name)), Name: name, IsDir: true}, nil
}

// CreateFile creates an empty UTF-8 file below an existing directory row.
func CreateFile(root, parent, name string) (MutationResult, error) {
	name = strings.TrimSpace(name)
	if err := ValidateName(name); err != nil {
		return MutationResult{}, err
	}
	parentAbs, cleanParent, err := resolveWithin(root, parent)
	if err != nil {
		return MutationResult{}, err
	}
	if err := ensureExistingDirWithin(root, parentAbs); err != nil {
		return MutationResult{}, err
	}
	dst := filepath.Join(parentAbs, name)
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return MutationResult{}, fmt.Errorf("目标已存在: %s", name)
		}
		return MutationResult{}, fmt.Errorf("创建文件失败: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(dst)
		return MutationResult{}, fmt.Errorf("关闭新文件失败: %w", err)
	}
	return MutationResult{Path: slashRel(filepath.Join(cleanParent, name)), Name: name}, nil
}

// DeletePaths removes selected rows. Symlink rows remove the link itself, not
// its target. Work continues after a failure and returns all failed paths.
func DeletePaths(root string, paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("未选择要删除的路径")
	}
	var failures []string
	for _, rel := range paths {
		abs, cleanRel, err := resolveWithin(root, rel)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", slashRel(rel), err))
			continue
		}
		if cleanRel == "" {
			failures = append(failures, "不能删除工作区根目录")
			continue
		}
		if _, err := os.Lstat(abs); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			failures = append(failures, fmt.Sprintf("%s: %v", slashRel(cleanRel), err))
			continue
		}
		if err := os.RemoveAll(abs); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", slashRel(cleanRel), err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("部分路径删除失败: %s", strings.Join(failures, "；"))
	}
	return nil
}

// ExportPath saves one file, or one directory as a ZIP, to dest. It writes a
// sibling temp file first so an interrupted export never leaves a partial
// destination file.
func ExportPath(root, rel, dest string) error {
	if strings.TrimSpace(dest) == "" {
		return nil
	}
	abs, cleanRel, err := resolveReadable(root, rel)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("读取待导出路径失败: %w", err)
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("创建导出目录失败: %w", err)
	}
	tmp, err := os.CreateTemp(parent, "."+filepath.Base(dest)+".*.tmp")
	if err != nil {
		return fmt.Errorf("创建导出临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpPath)
		}
	}()
	if info.IsDir() {
		err = writeZip(tmp, abs, cleanRel)
	} else {
		err = copyFileBytes(tmp, abs)
	}
	if err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("同步导出文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭导出文件失败: %w", err)
	}
	if err := fileutil.RenameWithRetry(tmpPath, dest); err != nil {
		return fmt.Errorf("发布导出文件失败: %w", err)
	}
	ok = true
	return nil
}

// ExportPaths saves one path directly, or packs a multi-selection into one ZIP.
// Each selected root keeps its base name inside the archive.
func ExportPaths(root string, paths []string, dest string) error {
	if strings.TrimSpace(dest) == "" {
		return nil
	}
	if len(paths) == 0 {
		return fmt.Errorf("未选择要保存的路径")
	}
	if len(paths) == 1 {
		return ExportPath(root, paths[0], dest)
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("创建导出目录失败: %w", err)
	}
	tmp, err := os.CreateTemp(parent, "."+filepath.Base(dest)+".*.tmp")
	if err != nil {
		return fmt.Errorf("创建导出临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpPath)
		}
	}()
	zw := zip.NewWriter(tmp)
	used := map[string]int{}
	for _, rel := range paths {
		abs, cleanRel, err := resolveReadable(root, rel)
		if err != nil {
			_ = zw.Close()
			return err
		}
		info, err := os.Stat(abs)
		if err != nil {
			_ = zw.Close()
			return fmt.Errorf("读取待导出路径失败: %w", err)
		}
		base := filepath.Base(abs)
		if cleanRel == "" {
			base = filepath.Base(filepath.Clean(root))
		}
		if n := used[base]; n > 0 {
			used[base] = n + 1
			base = fmt.Sprintf("%s-%d", base, n+1)
		} else {
			used[base] = 1
		}
		if info.IsDir() {
			if err := addZipDir(zw, abs, base); err != nil {
				_ = zw.Close()
				return err
			}
		} else if err := addZipFile(zw, abs, base); err != nil {
			_ = zw.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("完成 ZIP 失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("同步导出文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭导出文件失败: %w", err)
	}
	if err := fileutil.RenameWithRetry(tmpPath, dest); err != nil {
		return fmt.Errorf("发布导出文件失败: %w", err)
	}
	ok = true
	return nil
}

func writeZip(dst io.Writer, absDir, relDir string) error {
	zw := zip.NewWriter(dst)
	base := filepath.Base(absDir)
	if err := addZipDir(zw, absDir, base); err != nil {
		_ = zw.Close()
		return fmt.Errorf("打包目录 %s 失败: %w", slashRel(relDir), err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("完成 ZIP 失败: %w", err)
	}
	return nil
}

func addZipDir(zw *zip.Writer, absDir, base string) error {
	err := filepath.WalkDir(absDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := filepath.ToSlash(filepath.Join(base, rel))
		if d.IsDir() {
			_, err := zw.Create(name + "/")
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = name
		h.Method = zip.Deflate
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	return err
}

func addZipFile(zw *zip.Writer, abs, name string) error {
	info, err := os.Stat(abs)
	if err != nil {
		return err
	}
	h, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	h.Name = filepath.ToSlash(name)
	h.Method = zip.Deflate
	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	return copyFileBytes(w, abs)
}

func copyFileBytes(dst io.Writer, src string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(dst, f); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}
	return nil
}

// ValidateName validates a single cross-platform base name.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("名称不能为空")
	}
	if !utf8.ValidString(name) {
		return fmt.Errorf("名称不是有效 UTF-8")
	}
	if strings.ContainsAny(name, `/\`) || strings.ContainsRune(name, 0) {
		return fmt.Errorf("名称不能包含路径分隔符")
	}
	if strings.ContainsAny(name, `<>:"|?*`) || strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return fmt.Errorf("名称包含 Windows 非法字符或尾缀")
	}
	stem := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	switch stem {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return fmt.Errorf("名称是 Windows 保留设备名")
	}
	return nil
}

func resolveWithin(root, rel string) (string, string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(rel) == "" {
		return filepath.Clean(rootAbs), "", nil
	}
	if filepath.IsAbs(rel) {
		return "", "", fmt.Errorf("只接受工作区相对路径: %s", rel)
	}
	abs, err := wspath.ResolveRelWithin(rootAbs, rel)
	if err != nil {
		return "", "", fmt.Errorf("路径越出工作区: %s", rel)
	}
	cleanRel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return "", "", err
	}
	if cleanRel == "." {
		cleanRel = ""
	}
	return abs, cleanRel, nil
}

func ensureExistingDirWithin(root, abs string) error {
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("父目录不可用: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("父路径不是目录: %s", abs)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	realRoot, err := wspath.RealPath(rootAbs)
	if err != nil {
		return err
	}
	realDir, err := wspath.RealPath(abs)
	if err != nil {
		return err
	}
	if !wspath.Within(realRoot, realDir) {
		return fmt.Errorf("父目录通过符号链接越出工作区")
	}
	return nil
}

func slashRel(rel string) string {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." {
		return ""
	}
	return rel
}
