package wspath

// 本文件是「工作区路径归一 / 穿越防护」的唯一真相源（审计 2026-10-02 AP5-09）：
// 收敛前同一语义在 17+ 处各写各的（Clean 前缀、Rel 前缀、段级 ".."、正白名单），
// 且 internal/app/gaea_lint.go 等站点零校验——`rel="../../x"` 经 filepath.Join
// 清洗后直接落到工作区外被 os.Stat/os.ReadFile 消费。
//
// 两个原语，两种语义层，不要互相替代：
//
//	Within(root, target)            「target 是否落在 root 内」（含 root 本身）——
//	                                绝对/相对皆可，多根站点循环调用；
//	ResolveRelWithin(root, rel)      「把工作区相对路径 rel 解析成 root 内的绝对
//	                                路径，越界即错」——rel 必须是相对形态。
//
// 为什么不做成同一个函数：Within 的入参是**已解析路径**（调用方自持根白名单策略，
// 如 imagePathWithinAny / withinReadRoots / builtin.confine），ResolveRelWithin 的
// 入参是**未解析的相对路径**且必须回传解析结果 + 具体错因（绝对路径 / 卷相对 /
// 穿越三类错误调用方要分头提示）。硬并会让「允许绝对路径」的站点（预览、列举、
// 论文转换、证据复核）被迫自造旁路，正是本批要消灭的重复来源。
//
// 已知不覆盖（留池，勿在本层偷偷加）：符号链接/目录联接逃逸（需 EvalSymlinks，
// 见 internal/gaea/tool/builtin/confine.go 的 realPath 口径）。

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// 三类拒绝原因（调用方 errors.Is 分辨，措辞由调用方自持）。
var (
	// ErrPathAbsolute rel 是绝对路径（含 Windows 盘符、UNC、`\\?\` 设备路径）。
	ErrPathAbsolute = errors.New("wspath: 绝对路径不是工作区相对路径")
	// ErrPathVolume rel 是卷相对路径（Windows `C:foo` 形态：带卷名但非绝对）。
	ErrPathVolume = errors.New("wspath: 卷相对路径不是工作区相对路径")
	// ErrPathEscape rel 归一后越出 root（`..` 穿越、跨盘符等）。
	ErrPathEscape = errors.New("wspath: 路径越出根目录")
)

// Within 判断 target 是否落在 root 内（target == root 或在其子树下）。
//
// 口径：两侧 Abs + Clean 后用 filepath.Rel 判定——跨盘符返回 false；靠
// Rel 而非字符串前缀，故 `C:\ws-other` 不算 `C:\ws` 之内（前缀法会误判）。
// Windows 上大小写不敏感由 filepath.Rel 承担（实测：Rel(`C:\ws`,`c:\WS\x`)
// = `x`），因此 `C:\WS\x` 判为在 `C:\ws` 内——与文件系统语义一致。
//
// root/target 为空或解析失败一律 false（fail-closed；空 root 不再是「无约束」）。
// 语义与 internal/gaea/tool/builtin 的 within(root,path) 逐条一致（该处已改为
// 转调本函数），两者输入的绝对化方式不同：本函数自行 Abs（相对入参按进程 cwd
// 解析），builtin 传入的是 realPath 结果。
func Within(root, target string) bool {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(target) == "" {
		return false
	}
	absRoot, err := filepath.Abs(filepath.FromSlash(root))
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(filepath.FromSlash(target))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return false // 跨盘符等：不可比 → 不在内
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// ResolveRelWithin 把工作区相对路径 rel 解析为 root 内的绝对路径。
//
// 契约（逐条写清，别猜）：
//
//	rel == ""          → 返回 Clean(root) 本身（「工作区根」是合法落点，
//	                     与 GaeaListDir("") 的既有语义一致）
//	rel == "."         → 同上，返回 root 本身
//	rel == "sub/../b"  → 允许：Clean 后 `b` 仍在 root 内（不过度拒绝，
//	                     GaeaListDir 既有用例依赖 `sub/..\sub` 形态）
//	rel 含 `..` 逃逸    → ErrPathEscape（`..`、`../x`、`..\x`、`a/../../x`、
//	                     `C:\a\..\..\..\x` 归一后越界者）
//	rel 绝对路径        → ErrPathAbsolute（`C:\x`、`C:/x`、`\\srv\share\x`、`\\?\C:\x`）
//	rel 卷相对          → ErrPathVolume（`C:foo`：带卷名但非绝对）
//	rel 无卷根路径      → 允许且落在 root 内：Windows 上 `\x`/`/x` 的 IsAbs 为
//	                     false，filepath.Join 视作 root 下 `root\x`（不逃逸）
//	root 为空/不可解析  → 错误（fail-closed）
//
// 返回的是**未解析符号链接**的绝对路径（Clean + Abs 口径）；需要防符号链接
// 逃逸的站点自行再走 EvalSymlinks（见包注释的留池项）。
// 本函数不做 TrimSpace（调用方自持该口径：GaeaReadFile/GaeaWriteFile/
// resolveWorkspacePath 各自先 trim，其它站点不 trim）。
func ResolveRelWithin(root, rel string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("%w: 根目录为空", ErrPathEscape)
	}
	absRoot, err := filepath.Abs(filepath.FromSlash(root))
	if err != nil {
		return "", fmt.Errorf("wspath: 根目录解析失败 %q: %w", root, err)
	}
	slashed := filepath.FromSlash(rel)
	if filepath.IsAbs(slashed) {
		return "", fmt.Errorf("%w: %s", ErrPathAbsolute, rel)
	}
	if vol := filepath.VolumeName(slashed); vol != "" {
		return "", fmt.Errorf("%w: %s", ErrPathVolume, rel)
	}
	clean := "."
	if rel != "" {
		clean = filepath.Clean(slashed)
	}
	joined := filepath.Join(absRoot, clean)
	if !Within(absRoot, joined) {
		return "", fmt.Errorf("%w: %s", ErrPathEscape, rel)
	}
	return joined, nil
}
