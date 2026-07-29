// Package whisper — desktop_router.go
// 100% 对齐 ackem desktop-agent/router.ts
// use_computer 总路由：4 层安全沙箱 → 执行 → 审计
package whisper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RouterContext 路由器上下文
type RouterContext struct {
	DataRoot   string
	SessionID  string
	TaskPlanID string
	CWD        string
	// 设置（权限开关）
	AllowAppControl   bool
	AllowFileWrite    bool
	AllowDelete       bool
	AllowDownload     bool
	AllowInstall      bool
	AllowDocumentRead bool
	DownloadDir       string
	// 确认回调（由 Wails 前端实现）
	RequestConfirm func(actionLabel, path, target string, sensitiveWarning string) (allowed bool)
}

// UseComputerResult use_computer 工具执行结果
type UseComputerResult struct {
	Success    bool   `json:"success"`
	Content    string `json:"content"`
	Summary    string `json:"summary"`
	MemoryHint string `json:"memoryHint,omitempty"`
}

// ExecuteUseComputer 执行 use_computer 的 4 层安全管道
func ExecuteUseComputer(args UseComputerArgs, ctx RouterContext) UseComputerResult {
	action := args.Action
	now := time.Now().Format(time.RFC3339)
	label := ActionLabel(action)

	// 第1层：设置级拦截
	if blockReason := checkActionSettings(action, ctx); blockReason != "" {
		AppendDesktopAgentAudit(ctx.DataRoot, DesktopAgentAuditEntry{
			TS: now, Action: string(action),
			Path: args.Path, PathTo: args.PathTo, Target: args.Target, URL: args.URL,
			Result: "blocked", Summary: blockReason,
		})
		return UseComputerResult{Success: false, Content: blockReason, Summary: blockReason}
	}

	// 第2层：关闭进程黑名单
	if CloseActions[action] {
		target := args.Target
		if target == "" {
			target = args.Path
		}
		if isBlockedCloseTarget(strings.TrimSpace(target)) {
			msg := "系统关键进程不可关闭"
			AppendDesktopAgentAudit(ctx.DataRoot, DesktopAgentAuditEntry{
				TS: now, Action: string(action), Target: target,
				Result: "blocked", Summary: msg,
			})
			return UseComputerResult{Success: false, Content: msg, Summary: msg}
		}
	}

	// 第3层：路径策略评估
	policy := evaluatePathPolicy(action, args.Path, args.PathTo, ctx.CWD)
	if !policy.OK {
		AppendDesktopAgentAudit(ctx.DataRoot, DesktopAgentAuditEntry{
			TS: now, Action: string(action),
			Path: args.Path, PathTo: args.PathTo,
			Result: "blocked", Summary: policy.HardBlockReason,
		})
		return UseComputerResult{
			Success: false,
			Content: policy.HardBlockReason, Summary: policy.HardBlockReason,
		}
	}

	// 第4层：确认弹窗 / 跳过确认
	skipConfirm := ShouldSkipDesktopAgentConfirm(ctx.DataRoot, ctx.SessionID, action, ctx.TaskPlanID)
	if !skipConfirm && ctx.RequestConfirm != nil {
		allowed := ctx.RequestConfirm(label, policy.NormalizedPath, args.Target, policy.SensitiveWarning)
		if !allowed {
			msg := "用户未允许该操作"
			AppendDesktopAgentAudit(ctx.DataRoot, DesktopAgentAuditEntry{
				TS: now, Action: string(action),
				Path: policy.NormalizedPath, PathTo: policy.NormalizedPathTo,
				Target: args.Target, URL: args.URL,
				Result: "denied", Summary: msg,
			})
			return UseComputerResult{
				Success: false, Content: msg, Summary: msg,
				MemoryHint: fmt.Sprintf("电脑助手：用户拒绝 %s", label),
			}
		}
	}

	// 执行
	execPath := args.Path
	if policy.NormalizedPath != "" {
		execPath = policy.NormalizedPath
	}
	execPathTo := args.PathTo
	if policy.NormalizedPathTo != "" {
		execPathTo = policy.NormalizedPathTo
	}

	content := ""
	if args.Options != nil {
		content = args.Options.Content
	}

	result := ExecuteDesktopAgentAction(action, execPath, execPathTo, args.Target, args.Query, args.URL, content, DesktopExecContext{
		DataRoot:    ctx.DataRoot,
		DownloadDir: ctx.DownloadDir,
		CWD:         ctx.CWD,
	})

	AppendDesktopAgentAudit(ctx.DataRoot, DesktopAgentAuditEntry{
		TS: now, Action: string(action),
		Path: execPath, PathTo: execPathTo,
		Target: args.Target, URL: args.URL,
		Result: "allowed", Summary: result.Summary,
	})

	memoryHint := ""
	if result.OK {
		memoryHint = fmt.Sprintf("电脑助手 %s：%s", label, result.Summary)
	} else {
		memoryHint = fmt.Sprintf("电脑助手：%s", result.Summary)
	}

	return UseComputerResult{
		Success:    result.OK,
		Content:    result.Content,
		Summary:    result.Summary,
		MemoryHint: memoryHint,
	}
}

// ─── 设置检查 ────────────────────────────────────────────────────

func checkActionSettings(action DesktopAgentAction, ctx RouterContext) string {
	if AppActions[action] && !ctx.AllowAppControl {
		return "应用控制未开启"
	}
	if WriteActions[action] {
		if action == ActionDeletePath && !ctx.AllowDelete {
			return "删除操作未开启"
		}
		if action != ActionDeletePath && !ctx.AllowFileWrite {
			return "文件写入未开启"
		}
	}
	if DownloadActions[action] {
		if (action == ActionDownloadAndInstall || action == ActionRunInstaller) && !ctx.AllowInstall {
			return "安装操作未开启"
		}
		if !ctx.AllowDownload {
			return "下载未开启"
		}
	}
	if DocumentReadActions[action] && !ctx.AllowDocumentRead {
		return "文档读取未开启"
	}
	return ""
}

// ─── 路径策略 ────────────────────────────────────────────────────

// PolicyCheck 策略检查结果
type PolicyCheck struct {
	OK               bool
	NormalizedPath   string
	NormalizedPathTo string
	SensitiveWarning string
	PathMissing      bool
	HardBlockReason  string
}

// isBlockedCloseTarget 检查是否为禁止关闭的系统进程
func isBlockedCloseTarget(target string) bool {
	blocked := map[string]bool{
		"csrss.exe": true, "winlogon.exe": true, "lsass.exe": true,
		"services.exe": true, "smss.exe": true, "system": true,
		"registry": true, "explorer.exe": true,
	}
	name := strings.TrimSpace(strings.ToLower(target))
	if name == "" {
		return false
	}
	if blocked[name] {
		return true
	}
	if !strings.HasSuffix(name, ".exe") {
		return blocked[name+".exe"]
	}
	return false
}

func evaluatePathPolicy(action DesktopAgentAction, path, pathTo, cwd string) PolicyCheck {
	// open_app/close_app/focus_app 不需要 path 检查
	if AppActions[action] && action != ActionCopyPath && action != ActionMovePath {
		return PolicyCheck{OK: true}
	}

	// 需要 path 但缺参
	if path == "" && action != ActionOpenFolder {
		return PolicyCheck{OK: false, HardBlockReason: "缺少路径参数"}
	}

	// 规范化路径
	normalized := normalizePath(path, cwd)
	if normalized == "" && action != ActionOpenFolder {
		return PolicyCheck{OK: false, HardBlockReason: "无效路径"}
	}

	var normalizedTo string
	if pathTo != "" {
		normalizedTo = normalizePath(pathTo, cwd)
	}

	result := PolicyCheck{OK: true, NormalizedPath: normalized, NormalizedPathTo: normalizedTo}

	// 检查路径是否存在
	if normalized != "" {
		if _, err := os.Stat(normalized); err != nil {
			result.PathMissing = true
		}
	}

	// 敏感路径警告
	if isSensitivePath(normalized) {
		result.SensitiveWarning = "目标位于系统目录，请确认操作安全"
	}

	// 硬阻断：System32 写入
	if WriteActions[action] && isHardBlockedWritePath(normalized) {
		result.OK = false
		result.HardBlockReason = "禁止写入系统目录"
	}

	return result
}

// ─── 路径处理 ────────────────────────────────────────────────────

func normalizePath(p, cwd string) string {
	if p == "" {
		return ""
	}
	// 展开 ~
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		p = filepath.Join(home, p[1:])
	}
	// 展开 %ENV%
	p = os.ExpandEnv(p)
	// 转绝对路径
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	// 规范化 + 禁止 .. 逃逸
	clean := filepath.Clean(p)
	if !strings.HasPrefix(clean, filepath.VolumeName(clean)+string(os.PathSeparator)) &&
		!strings.HasPrefix(clean, string(os.PathSeparator)) {
		return p // 保持原样
	}
	return clean
}

func isSensitivePath(p string) bool {
	lower := strings.ToLower(p)
	sensitive := []string{
		"c:\\windows",
		"c:\\program files",
		"c:\\program files (x86)",
	}
	for _, s := range sensitive {
		if strings.HasPrefix(lower, s) {
			return true
		}
	}
	return false
}

func isHardBlockedWritePath(p string) bool {
	lower := strings.ToLower(p)
	blocked := []string{
		"c:\\windows\\system32",
		"c:\\windows\\syswow64",
	}
	for _, b := range blocked {
		if strings.HasPrefix(lower, b) {
			return true
		}
	}
	return false
}
