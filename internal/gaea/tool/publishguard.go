package tool

// PublishGuard — P1b 发布边界（2026-11-09 众神殿 pan_1794234593182 裁决 action-4）。
//
// 办公侧主对话 bash 无 OS 沙箱（gaea_handler.go 显式 Sandbox.Bash="off"），扩展名
// 字符串扫描可被重定向/脚本/改名绕过（裁决风险条），因此硬禁落在「工作区发布边界」：
// Begin 对工作区根做 size+mtime 快照 → 工具执行 → Scan 比对出 durable 档
// （v1=docx/xlsx/pptx 三格式，D1 口径）的新增/修改 → Quarantine 移出正式工作区，
// 调用方 fail-loud 回显期望路径集（备忘 :23 fail-loud 继承）。
//
// 三分档口径（备忘 :21 继承）：durable 硬禁（本文件）；derived（渲染 PDF/PNG 挂父
// 哈希）与 ephemeral（结束查残留）属 P2/后续档，本文件不冒充全量治理。
//
// 边界（action-2 spike 验证对象）：前台路径由调用方在执行后 Scan；后台 bash 跨回合
// 存活，须在 job 完成回调（jobs.Manager.SetOnCompletion）再 Scan——前台 Scan 对
// 「尚未落盘的后台写」恒为 clean，spike 实证。

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultDurableExts v1 durable 档扩展名（D1：v1 只承诺 Office 三格式；非 Office
// durable（bash 写 .md/.csv 交付物）明确列范围外风险，P3/P4 再按交付意图扩展）。
func DefaultDurableExts() map[string]bool {
	return map[string]bool{".docx": true, ".xlsx": true, ".pptx": true}
}

// PublishViolation 一次 Scan 发现的 durable 越界写。
type PublishViolation struct {
	Path    string // 工作区内绝对路径
	RelPath string // 相对所属根的路径（ToSlash 规范化）
	Root    string // 所属根
	Kind    string // "created"（快照中不存在）/ "modified"（size 或 mtime 变化）
	Size    int64
}

// PublishGuard 工作区 durable 写发布边界。单次生命周期：Begin →（执行）→ Scan →
// Quarantine。非并发安全：同一线程内顺序调用；后台路径可在 job 完成回调线程
// 复用同一实例（Begin 在启动时做过快照即可）。
type PublishGuard struct {
	roots          []string
	quarantine     string
	durable        map[string]bool
	base           map[string]publishFileState
	source         string // D6 账本来源标注（foreground/background）
	commandPreview string
	lastLedger     *quarantineLedger
}

// WithSource 标注账本来源与命令预览（D6：归属证据——谁在何时因何命令隔离）。
func (g *PublishGuard) WithSource(source, command string) *PublishGuard {
	g.source = source
	g.commandPreview = command
	return g
}

// LastQuarantineDest 最近一次隔离的目的地 stamp 目录（fail-loud 回显用）。
func (g *PublishGuard) LastQuarantineDest() string {
	if g.lastLedger == nil {
		return ""
	}
	return filepath.Join(g.quarantine, g.lastLedger.rec.Stamp)
}

type publishFileState struct {
	size  int64
	mtime time.Time
}

// NewPublishGuard roots 为受守卫的工作区根（绝对路径）；quarantine 为越界件
// 隔离目录（建议落在工作区内 .gaea/ 下，保证 Rename 同卷）；durable 为空时取
// DefaultDurableExts。
func NewPublishGuard(roots []string, quarantine string, durable map[string]bool) *PublishGuard {
	if durable == nil {
		durable = DefaultDurableExts()
	}
	return &PublishGuard{
		roots:      append([]string(nil), roots...),
		quarantine: quarantine,
		durable:    durable,
	}
}

// Begin 执行前快照。对每个根做整树遍历，记录 durable 扩展名文件的 size+mtime。
// 目录不进快照（durable 判据只看文件），遍历剪掉软链目录防环、隔离目录防自查、
// `~$` Office 锁件目录防误捕。
func (g *PublishGuard) Begin() error {
	g.base = make(map[string]publishFileState)
	for _, root := range g.roots {
		if err := g.walk(root, func(path string, st fs.FileInfo) {
			g.base[path] = publishFileState{size: st.Size(), mtime: st.ModTime()}
		}); err != nil {
			return err
		}
	}
	return nil
}

// Scan 执行后比对。返回 durable 档的新增/修改清单；不存在越界时返回 nil。
func (g *PublishGuard) Scan() ([]PublishViolation, error) {
	if g.base == nil {
		return nil, fmt.Errorf("publishguard: Scan before Begin")
	}
	cur := make(map[string]publishFileState)
	for _, root := range g.roots {
		if err := g.walk(root, func(path string, st fs.FileInfo) {
			cur[path] = publishFileState{size: st.Size(), mtime: st.ModTime()}
		}); err != nil {
			return nil, err
		}
	}
	var out []PublishViolation
	for path, st := range cur {
		before, existed := g.base[path]
		if !existed {
			out = append(out, g.violationFor(path, st, "created"))
			continue
		}
		if before.size != st.size || !before.mtime.Equal(st.mtime) {
			out = append(out, g.violationFor(path, st, "modified"))
		}
	}
	return out, nil
}

// Quarantine 把越界件移出正式工作区（隔离目录下按时间戳子目录保留相对结构），
// 返回已移动的原始绝对路径（调用方 fail-loud 回显）。移动失败（文件被占用等）
// 时 error 非空且未移动件仍留在正式工作区——调用方必须把 error 当硬失败上报。
// D6（dsh-ops 蒸馏 2026-11-10）：先记账后放句柄——任何 MkdirAll/Rename 之前
// 先落 pending 账本记录，移动后补终态；账本失败不阻断隔离（硬边界优先），
// 该 stamp 视为无主永不自动删。
func (g *PublishGuard) Quarantine(vs []PublishViolation) ([]string, error) {
	var moved []string
	var errs []string
	led := beginQuarantineRecord(g.quarantine, g.source, g.commandPreview, vs)
	g.lastLedger = led
	stamp := led.rec.Stamp
	for _, v := range vs {
		dst := filepath.Join(g.quarantine, stamp, filepath.FromSlash(v.RelPath))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", v.Path, err))
			continue
		}
		if err := os.Rename(v.Path, dst); err != nil {
			// 跨卷/占用兜底：copy+remove。remove 失败=越界件仍在正式工作区，如实报错。
			if cerr := copyPublishFile(v.Path, dst); cerr != nil {
				errs = append(errs, fmt.Sprintf("%s: rename=%v copy=%v", v.Path, err, cerr))
				continue
			}
			if rerr := os.Remove(v.Path); rerr != nil {
				errs = append(errs, fmt.Sprintf("%s: copy 成功但移除失败 %v（越界件仍在正式工作区）", v.Path, rerr))
				continue
			}
		}
		moved = append(moved, v.Path)
	}
	led.finish(moved, errs)
	if len(errs) > 0 {
		return moved, fmt.Errorf("quarantine 部分失败：%s", strings.Join(errs, "; "))
	}
	return moved, nil
}

// ExpectedPathsNote fail-loud 提示文案（备忘 :23：回显期望路径集）。expected 为
// 声明写路径（bash 场景为空=无声明）；dest（可选）为隔离目的地 stamp 目录
// （D6：会话 JSONL 成为第二份归属证据，用户找得到文件）；输出直接进工具回执。
func ExpectedPathsNote(violations []PublishViolation, expected []string, dest ...string) string {
	var b strings.Builder
	b.WriteString("发布边界拦截：以下 Office 交付文件（durable 档）不得经本工具写入正式工作区，已隔离：")
	for _, v := range violations {
		b.WriteString("\n  - " + v.RelPath + "（" + v.Kind + "）")
	}
	if len(dest) > 0 && dest[0] != "" {
		b.WriteString("\n隔离位置：" + dest[0])
	}
	if len(expected) == 0 {
		b.WriteString("\ndurable 交付物只能经结构化入口：office_generate（从零生成）或 docx_apply/docx_accept/pptx_apply/xlsx_apply（编辑）。")
	} else {
		b.WriteString("\n声明写路径集：" + strings.Join(expected, "; "))
	}
	return b.String()
}

// --- internals ---

// publishPrunedDirs 依赖/构建目录剪枝（大仓快照开销控制：node_modules 万级
// 条目会拖慢每次 bash 调用）。这些目录不是交付物落点，其中的 durable 写不做
// 发布边界判据（如实记档的范围裁剪，非漏拦）。
var publishPrunedDirs = map[string]bool{
	".git": true, "node_modules": true, "__pycache__": true,
	".venv": true, "venv": true, "target": true,
}

func (g *PublishGuard) walk(root string, record func(path string, st fs.FileInfo)) error {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 根尚未创建=空快照，合法
		}
		return err
	}
	if !info.IsDir() {
		if g.isDurable(root) {
			record(root, info)
		}
		return nil
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil // 执行期间被删：比对自然缺席
			}
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path == root {
				return nil
			}
			// 隔离目录（防自查）、软链目录（防环）、依赖目录（开销）、~$ Office
			// 锁件目录剪枝。
			if g.withinQuarantine(path) || d.Type()&fs.ModeSymlink != 0 ||
				strings.HasPrefix(name, "~$") || publishPrunedDirs[name] {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if strings.HasPrefix(name, "~$") {
			return nil // Word/Excel/WPS 锁件=ephemeral，不入 durable 判据
		}
		if !g.isDurable(path) {
			return nil
		}
		st, serr := d.Info()
		if serr != nil {
			return serr
		}
		record(path, st)
		return nil
	})
}

func (g *PublishGuard) isDurable(path string) bool {
	return g.durable[strings.ToLower(filepath.Ext(path))]
}

func (g *PublishGuard) withinQuarantine(path string) bool {
	if g.quarantine == "" {
		return false
	}
	rel, err := filepath.Rel(g.quarantine, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func (g *PublishGuard) violationFor(path string, st publishFileState, kind string) PublishViolation {
	v := PublishViolation{Path: path, Kind: kind, Size: st.size}
	for _, root := range g.roots {
		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		v.Root = root
		v.RelPath = filepath.ToSlash(rel)
		break
	}
	return v
}

func copyPublishFile(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
}
