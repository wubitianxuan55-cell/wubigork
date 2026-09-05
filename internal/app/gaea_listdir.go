package app

// gaea_listdir.go — GaeaListDir 核心实现（v4.96 登记 Go 小刀销账）：
//  1. 补 IsAbs 分支——绝对路径 Clean 后直接使用（不与工作区根 Join，口径对齐
//     resolvePreviewPath 的 IsAbs 分支）；相对路径行为与旧实现完全一致
//     （Join(gaeaCwd(), rel)，含 rel==""=工作区根、「../」逃逸同旧实现不拦）。
//  2. 错误不再吞——真实 OS 错误带 GAEADIR_* 结构化错误码透传（形态对齐
//     internal/gaea/tool/builtin/errcode.go 的 `Error [CODE]: message` U1 口径；
//     范围仅工作区目录列举，不全局推广），前端按码路由降级路径，不解析散文。
//
// 绑定签名例外（主代理拍板，v4.98 冻结面 581 不变）：[]DirEntry →
// ([]DirEntry, error)——方法名/绑定数/drift 均不变（gen 包装行为手改对齐
// gen_bindings 输出）；成功路径负载不变，失败从「resolve 空切片」改为
// 「reject 错误串」，前端全部调用点（FileTree/useComposerMenus/
// deliverablesTurn/VerifyArtifactsThumbs）均有 catch。

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// GAEADIR_* 结构化错误码：前端据此路由恢复路径（不存在/非目录 → 降级
// 「缺失」；权限等读失败 → 透传原文按不可用处理），不解析自然语言文案。
const (
	errCodeDirNotFound = "GAEADIR_NOT_FOUND"   // 目录不存在
	errCodeDirNotDir   = "GAEADIR_NOT_DIR"     // 路径存在但不是目录（是文件）
	errCodeDirRead     = "GAEADIR_READ_FAILED" // 读取目录失败（权限等 OS 错误透传）
)

// listDirError 按 U1 错误码口径构造结构化错误：`Error [CODE]: message`。
func listDirError(code, format string, args ...any) error {
	return fmt.Errorf("Error [%s]: %s", code, fmt.Sprintf(format, args...))
}

// osStat/osReadDir 系统调用缝（默认真实实现）：权限透传用例在 Windows 上
// 无法用 chmod 复现目录读剥夺，测试注入假错误以全平台确定性覆盖 READ_FAILED
// 分支（先例：时间/随机源缝注入）。
var (
	osStat    = os.Stat
	osReadDir = os.ReadDir
)

// listDirEntries 列出目录条目。rel 为空 = 工作区根；相对路径 Join 工作区根
// （旧行为逐字节一致）；绝对路径 Clean 后直接使用（新分支，含正斜杠写法的
// Windows 盘符路径，与前端 ToSlash 口径互通）。错误带 GAEADIR_* 码透传，
// 不再吞成空切片。
func listDirEntries(rel string) ([]DirEntry, error) {
	root := gaeaCwd()
	dir := root
	if rel != "" {
		if filepath.IsAbs(rel) {
			dir = filepath.Clean(rel)
		} else {
			dir = filepath.Join(root, rel)
		}
	}
	info, err := osStat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, listDirError(errCodeDirNotFound, "目录不存在: %s（%v）", filepath.ToSlash(dir), err)
		}
		// Stat 本身失败（父目录无权限等）：同读失败口径透传原文
		return nil, listDirError(errCodeDirRead, "读取目录失败: %s（%v）", filepath.ToSlash(dir), err)
	}
	if !info.IsDir() {
		return nil, listDirError(errCodeDirNotDir, "不是目录: %s", filepath.ToSlash(dir))
	}
	entries, err := osReadDir(dir)
	if err != nil {
		// 权限等 OS 错误：码 + 原文一并透传（不再吞）
		return nil, listDirError(errCodeDirRead, "读取目录失败: %s（%v）", filepath.ToSlash(dir), err)
	}
	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		out = append(out, DirEntry{Name: e.Name(), IsDir: e.IsDir(), Size: size})
	}
	return out, nil
}
