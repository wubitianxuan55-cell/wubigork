package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/wspath"
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
	return "Move or rename a file. Creates the destination's parent directories as needed. Fails when the destination exists unless overwrite:true is passed. Works across volumes (falls back to copy+remove when rename can't). 覆盖的是正式交付面的 Office 件（docx/xlsx/pptx）时，旧文件会留一份快照到 .gaea/versions/ 下（按原相对路径分目录、文件名是旧文件的修改时间），回执给快照路径——同名文件反复交付也有版本可回退。"
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

func (m moveFile) WritePaths(args json.RawMessage) ([]string, error) {
	var p struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Source) == "" || strings.TrimSpace(p.Destination) == "" {
		return nil, fmt.Errorf("source and destination are required")
	}
	// 源在系统临时目录时不声明为写路径（2026-11-25 铜陵反馈 #2）：向工作区内
	// move 是「发布」不是越界写，agent 层只按写面校验终点；Execute 里对源做
	// 同一条 withinTempDir 校验，两处口径一致。
	src := resolveIn(m.workDir, p.Source)
	if withinTempDir(src) {
		return []string{resolveIn(m.workDir, p.Destination)}, nil
	}
	return []string{
		src,
		resolveIn(m.workDir, p.Destination),
	}, nil
}

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
	// Destination is always a write and must sit inside the workspace roots.
	// The source may live in the OS temp directory (2026-11-25 铜陵反馈 #2：
	// 办公技能的合规落盘指引就是「脚本先落系统临时目录 → move_file 移入正式名」，
	// 而此前 source 越界即拒，推荐路径自己走不通，交付一个 xlsx 要绕 4 次调用)：
	// read side of temp is fine — a move INTO the workspace publishes the file,
	// it cannot smuggle writes out. Anything else outside the roots stays denied.
	if err := confine(m.roots, dst); err != nil {
		return "", err
	}
	if err := confine(m.roots, src); err != nil {
		if !withinTempDir(src) {
			return "", err
		}
	}
	srcFi, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("move %s: source not found: %w", src, err)
	}
	if !srcFi.Mode().IsRegular() {
		return "", fmt.Errorf("move %s: source is not a regular file (directories are not supported)", src)
	}
	// v4.1b：移动前源文件基线快照（回滚原料）。
	srcContent, _ := os.ReadFile(src)
	baseline := evidence.StageBaseline(ctx, src, srcContent)
	snapshotted := "" // 被覆盖的上一版留在了哪（工作区相对路径；空=没留）
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
		// 占用预检（2026-11-26 造价反馈，旺平 #5）：用户开着交付表时这里必然
		// 失败——WPS/Excel 打开文档时拒绝写共享，试开写句柄即可提前点名。
		if err := probeNotOccupied(dst); err != nil {
			return "", err
		}
		// 覆盖正式名下的 Office 交付件前留一份上一版（2026-11-27 AI 反馈 #4：同一
		// 文件名反复覆盖交付，没有版本痕迹，用户只看到同名文件变了）。留不下快照
		// 不拦覆盖——交付是主事，快照是附带。
		kept, kerr := snapshotDurable(m.workDir, dst, dstFi)
		if kerr != nil {
			slog.Warn("move_file: 上一版快照没留下", "dst", dst, "error", kerr)
		}
		if err := os.Remove(dst); err != nil {
			return "", fmt.Errorf("move %s: replace existing destination %s: %w%s", src, dst, err, occupiedHintFor(err))
		}
		snapshotted = kept
	}
	if err := os.Rename(src, dst); err != nil {
		// Rename fails across volumes (EXDEV) and in a few other cases —
		// fall back to copy+remove so a move still happens.
		if copyErr := copyFileContents(src, dst, srcFi.Mode().Perm()); copyErr != nil {
			return "", fmt.Errorf("move %s → %s: rename: %v; copy fallback: %w%s", src, dst, err, copyErr, occupiedHintFor(err, copyErr))
		}
		if rmErr := os.Remove(src); rmErr != nil {
			// D3 失败即值（dsh-ops 蒸馏 2026-11-10）：目标已完整写出、源未删——
			// 副作用已发生（applied），盲目重试 move 会 No such file/重复复制。
			next := "目标已完整写出、源未删：先核实目标内容，再决定手动删源或视为完成"
			if occupiedHintFor(rmErr) != "" {
				next = "源文件被占用（多半在 WPS/Excel 里开着）：目标已完整写出；关掉占用进程后删源，或视为完成"
			}
			return "", &tool.Failure{
				Code:        tool.CodeExecError,
				Subcode:     "SOURCE_REMOVE_FAILED",
				Err:         fmt.Errorf("move %s → %s: copied but could not remove source: %w%s", src, dst, rmErr, occupiedHintFor(rmErr)),
				EffectState: tool.EffectApplied,
				NextAction:  next,
			}
		}
	}
	// v4.1 证据链：记录移动映射（Target=目标路径；Before/After 标注来源→去向）。
	evidence.RecordChange(ctx, evidence.ChangeRecord{
		Tool:          "move_file",
		Target:        dst,
		BeforeSummary: "→ moved from " + src,
		AfterSummary:  dst,
		BaselinePath:  baseline,
	})
	if snapshotted == "" {
		return fmt.Sprintf("moved %s → %s", src, dst), nil
	}
	return fmt.Sprintf("moved %s → %s\n上一版已留快照：%s（覆盖前的旧文件；要回退就把它 move_file 回原路径）", src, dst, snapshotted), nil
}

// snapshotDurable move_file 覆盖正式交付面的 Office 件（docx/xlsx/pptx）前，把旧
// 文件留一份到 <工作区>/.gaea/versions/<原相对路径去扩展名>/<旧文件修改时间>.<扩展名>。
// 过程件（.gaea/work/、草稿、探针）、非 Office 件、非工作区装配都不留。返回快照的
// 工作区相对路径。
func snapshotDurable(workDir, dst string, fi os.FileInfo) (string, error) {
	ext := strings.ToLower(filepath.Ext(dst))
	if workDir == "" || !tool.DefaultDurableExts()[ext] {
		return "", nil
	}
	rel, err := filepath.Rel(workDir, dst)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", nil
	}
	if tool.PublishExempt(rel) {
		return "", nil
	}
	dir := durableVersionDir(workDir, rel, ext)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	stamp := fi.ModTime().Format("20060102-150405")
	snap := filepath.Join(dir, stamp+ext)
	for i := 2; ; i++ {
		if _, err := os.Lstat(snap); os.IsNotExist(err) {
			break
		}
		snap = filepath.Join(dir, fmt.Sprintf("%s-%d%s", stamp, i, ext))
	}
	if err := copyFileContents(dst, snap, fi.Mode().Perm()); err != nil {
		return "", err
	}
	_ = os.Chtimes(snap, fi.ModTime(), fi.ModTime())
	got, err := filepath.Rel(workDir, snap)
	if err != nil {
		return "", nil
	}
	return filepath.ToSlash(got), nil
}

// durableVersionDir 一份正式交付件的上一版快照目录。
func durableVersionDir(workDir, rel, ext string) string {
	return filepath.Join(workDir, ".gaea", "versions", strings.TrimSuffix(rel, ext))
}

// latestVersionSnapshot 某正式件（工作区相对路径）最近一份上一版快照的工作区
// 相对路径；没有返回空串。快照文件名是旧文件的修改时间，按名字排序即按时间排序。
func latestVersionSnapshot(workDir, relPath string) string {
	rel := filepath.FromSlash(relPath)
	dir := durableVersionDir(workDir, rel, strings.ToLower(filepath.Ext(rel)))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	last := ""
	for _, e := range entries {
		if !e.IsDir() && e.Name() > last {
			last = e.Name()
		}
	}
	if last == "" {
		return ""
	}
	got, err := filepath.Rel(workDir, filepath.Join(dir, last))
	if err != nil {
		return ""
	}
	return filepath.ToSlash(got)
}

// occupiedHint 是 Windows 共享/锁冲突（文件正被别的进程打开）时附在错误后的
// 处置提示。2026-11-26 造价反馈（旺平 #5）：用户开着交付表时 move_file 落盘
// 失败只透传 OS 原话，模型摸不着头脑转去试别的；提前点名占用并给两条出路。
const occupiedHint = "——文件正被占用（多半是你在 WPS/Excel 里开着它）：关掉后重试；要在开着的情况下改内容，走 office_edit 实时编辑"

// isFileOccupied 报告 err 是否为共享冲突 / 锁冲突（ERROR_SHARING_VIOLATION=32、
// ERROR_LOCK_VIOLATION=33）。按错误码字面比较而不引用 syscall 常量：那两个
// 常量只在 windows 构建里有，字面量在各平台都能编译（非 windows 恒 false）。
func isFileOccupied(err error) bool {
	if runtime.GOOS != "windows" || err == nil {
		return false
	}
	return errors.Is(err, syscall.Errno(32)) || errors.Is(err, syscall.Errno(33))
}

// occupiedHintFor 任一 err 是占用类错误就返回处置提示，否则空串。
func occupiedHintFor(errs ...error) string {
	for _, err := range errs {
		if isFileOccupied(err) {
			return occupiedHint
		}
	}
	return ""
}

// probeNotOccupied 在动目标之前试开一次写句柄：WPS/Excel 打开文档时拒绝写
// 共享，这一步就会报共享冲突。其他打开失败（只读属性等）不在这里拦，交给
// 后续步骤按原口径报。
func probeNotOccupied(dst string) error {
	f, err := os.OpenFile(dst, os.O_RDWR, 0)
	if err != nil {
		if isFileOccupied(err) {
			return fmt.Errorf("move → %s: %w%s", dst, err, occupiedHint)
		}
		return nil
	}
	_ = f.Close()
	return nil
}

// withinTempDir reports whether path resolves inside the OS temp directory
// (os.TempDir, symlink-free on both endpoints).
func withinTempDir(path string) bool {
	tmp, err := wspath.RealPath(os.TempDir())
	if err != nil {
		return false
	}
	real, err := wspath.RealPath(path)
	if err != nil {
		return false
	}
	return wspath.Within(tmp, real)
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
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst) // don't leave a partial copy behind
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	if err := os.Chmod(dst, perm); err != nil {
		return err
	}
	return nil
}
