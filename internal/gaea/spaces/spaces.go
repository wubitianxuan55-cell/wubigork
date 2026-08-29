// Package spaces 是双空间（space_id: work/play）的纯函数层：空间常量、
// 归一化校验与目录落点。S2（会话空间）与 S4（产物分区）共用本包，
// 只做路径计算，不做 IO、不依赖其他内部包（叶子包，可被 config/session 安全引用）。
//
// 设计权威：docs/gaea-space-dimension-design.md §2.2/§3/§5。
// 兼容红线：work 恒返回现状路径（.gaea/exports、.gaea/work、会话平铺目录），
// 不挪目录、不破坏既有产物链接；只有 play 落分区目录。
package spaces

import (
	"os"
	"path/filepath"
)

// 空间标识（S1.1 落地设计 §1：space_id 取值固定为 work|play）。
const (
	// SpaceWork 是办公空间（默认）：Hephaestus.db / exports / work 现状路径。
	SpaceWork = "work"
	// SpacePlay 是娱乐空间：轻语/聊天/角色库等整库天然 play 的领域。
	SpacePlay = "play"
)

// Valid 报告 s 是否为合法空间标识（严格小写，不做大小写归一——归一化交给 Normalize）。
func Valid(s string) bool {
	return s == SpaceWork || s == SpacePlay
}

// Normalize 把空间值归一化：空值与非法值一律回退 work（读端降级、写端安全默认）。
// 「旧平铺会话恒按 work 可读」的降级语义由本函数单点保证。
func Normalize(s string) string {
	if s == SpacePlay {
		return SpacePlay
	}
	return SpaceWork
}

// SpaceOr 返回 space 的有效值：非法（含空）时回退 fallback。
// 读端降级惯用法：spaces.SpaceOr(e.Space, spaces.SpaceWork)。
func SpaceOr(space, fallback string) string {
	if Valid(space) {
		return space
	}
	return fallback
}

// workspaceRoot 解析工作区根：cwd 为空时回退进程工作目录（仍失败返回 ""，
// 由调用方得到相对路径形态，行为与 config.WorkspaceSessionDir 的历史回退一致）。
func workspaceRoot(cwd string) string {
	if cwd != "" {
		return cwd
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return ""
}

// WorkspaceSessionDir 返回空间分区的会话目录 <cwd>/.gaea/sessions/<space>/
// （space 经 Normalize 归一，空值 → work）。注意：是否启用分区由
// space.mode 决定——mode=off 时调用方应使用平铺目录（config.WorkspaceSessionDir
// 的 "" 形态），本函数只做纯映射。
func WorkspaceSessionDir(cwd, space string) string {
	return filepath.Join(workspaceRoot(cwd), ".gaea", "sessions", Normalize(space))
}

// ExportsDir 返回产物导出目录：work → <cwd>/.gaea/exports（现状路径不动），
// play → <cwd>/.gaea/play/exports（设计 §5 产物路径分区）。
func ExportsDir(cwd, space string) string {
	root := workspaceRoot(cwd)
	if Normalize(space) == SpacePlay {
		return filepath.Join(root, ".gaea", "play", "exports")
	}
	return filepath.Join(root, ".gaea", "exports")
}

// WorkDir 返回过程/中间文件目录：work → <cwd>/.gaea/work（现状路径不动），
// play → <cwd>/.gaea/play/work。
func WorkDir(cwd, space string) string {
	root := workspaceRoot(cwd)
	if Normalize(space) == SpacePlay {
		return filepath.Join(root, ".gaea", "play", "work")
	}
	return filepath.Join(root, ".gaea", "work")
}

// SpaceForDir 从会话目录推导空间归属（目录分区方案的决定性优势：恢复链
// 不读 meta，space 只能从路径推导）。规则：
//   - <root>/.gaea/sessions/play[/archive] → play
//   - 其余（work 分区、平铺目录、平铺 archive）→ work（旧平铺 = work 兼容）
//
// archive 子目录向上取父目录归属（各空间目录下有自己的 archive/）。
func SpaceForDir(dir string) string {
	dir = filepath.Clean(dir)
	if filepath.Base(dir) == "archive" {
		dir = filepath.Dir(dir)
	}
	return Normalize(filepath.Base(dir))
}
