//go:build !windows

package permission

// DangerousWindowsCommand 非 Windows 平台无破坏性命令硬拦截表
//（真表在 dangerous_windows.go）。gaea 运行面只在 Windows，此文件仅为
// linux 交叉编译门禁补齐（GOOS=linux go build ./internal/gaea/...），
// 恒放行=与平台能力一致（无 bash/cmd/PowerShell 可删）。
func DangerousWindowsCommand(argv []string) (bool, string) {
	return false, ""
}
