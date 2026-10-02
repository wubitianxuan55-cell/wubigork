// Package whisper — desktop_router.go
// 100% 对齐 ackem desktop-agent/router.ts
// use_computer 总路由：4 层安全沙箱 → 执行 → 审计
package whisper

import (
	"fmt"
	"log/slog"
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

// appendAuditOrFail 安全审计留痕失败必须留痕（审计 P1 IN4-07）：审计写盘
// 失败时把失败本身记入 slog——安全审计「静默没写」等于审计不存在，被拦截
// 的危险操作不能因为落盘失败就无迹可查。
func appendAuditOrFail(dataRoot string, e DesktopAgentAuditEntry) {
	if err := AppendDesktopAgentAudit(dataRoot, e); err != nil {
		slog.Error("desktop agent audit write failed", "action", e.Action, "result", e.Result, "error", err)
	}
}

// ExecuteUseComputer 执行 use_computer 的 4 层安全管道
func ExecuteUseComputer(args UseComputerArgs, ctx RouterContext) UseComputerResult {
	action := args.Action
	now := time.Now().Format(time.RFC3339)
	label := ActionLabel(action)

	// 第1层：设置级拦截
	if blockReason := checkActionSettings(action, ctx); blockReason != "" {
		appendAuditOrFail(ctx.DataRoot, DesktopAgentAuditEntry{
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
			appendAuditOrFail(ctx.DataRoot, DesktopAgentAuditEntry{
				TS: now, Action: string(action), Target: target,
				Result: "blocked", Summary: msg,
			})
			return UseComputerResult{Success: false, Content: msg, Summary: msg}
		}
	}

	// 第3层：路径策略评估
	policy := evaluatePathPolicy(action, args.Path, args.PathTo, ctx.CWD)
	if !policy.OK {
		appendAuditOrFail(ctx.DataRoot, DesktopAgentAuditEntry{
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
			appendAuditOrFail(ctx.DataRoot, DesktopAgentAuditEntry{
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
		// 此处已在第 4 层确认之后（或用户已开启自动批准），执行器的
		// Confirmed 门是对「绕过本管线直接调用执行器」的纵深防御。
		Confirmed: true,
	})

	appendAuditOrFail(ctx.DataRoot, DesktopAgentAuditEntry{
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

// isBlockedCloseTarget 检查是否为禁止关闭的系统进程。
//
// 判据必须与执行侧同源：closeAppTarget 过去走 PowerShell 的
// `Get-Process -Name '<name>'`，而 -Name 支持 `*`/`?`/`[..]` 通配符，本函数却是
// 精确 map 匹配 ⇒ `explorer*`、`*` 不命中黑名单，却能在执行侧展开成
// 「explorer 进程」或「本机全部进程」。因此这里先做名字形状拒绝
// （hasProcessNameWildcard），再接原有黑名单。
func isBlockedCloseTarget(target string) bool {
	if hasProcessNameWildcard(target) {
		return true
	}
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

// hasProcessNameWildcard 判定目标名不是「合法进程名」，而是路径或通配符形态。
// 合法进程名只含字母数字与 `._-`；出现通配符（`*` `?` `[` `]`）或路径分隔符
// （`\` `/`）即拒绝——这类名字在判据里永远匹配不到黑名单，但执行侧若把它交给
// 支持通配符的接口就会展开成别的进程（批次十三 AP4-07 同族坑：判据空间与执行
// 空间必须同源）。
func hasProcessNameWildcard(target string) bool {
	if strings.TrimSpace(target) == "" {
		return false
	}
	return strings.ContainsAny(target, `*?[]\/`)
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

	// 需要 pathTo 的写类操作缺参必须硬阻断：copy/move 的 pathTo 为空时
	// filepath.Dir("") == "."，执行侧会静默把源文件写到「进程当前目录」下的同名
	// 文件（实测 CWD=internal/whisper），既不落点明确也无任何告警。
	if WriteActions[action] && (action == ActionCopyPath || action == ActionMovePath) && strings.TrimSpace(pathTo) == "" {
		return PolicyCheck{OK: false, HardBlockReason: "缺少目标路径参数"}
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

	// 硬阻断：下载类操作的落点同样受写入硬阻断约束。
	// action 自身在 AppActions 之外，但执行侧（desktop_executor.go:100-120）会把
	// 下载物写到 path 或默认下载目录——这是实打实的文件写入，不能因为「action 不在
	// WriteActions 里」而漏过系统目录硬阻断（实测：evaluatePathPolicy(
	// ActionDownloadAndInstall,"C:\Windows\System32\evil.exe") 曾返回 OK=true）。
	if DownloadActions[action] && normalized != "" && isHardBlockedWritePath(normalized) {
		result.OK = false
		result.HardBlockReason = "禁止写入系统目录"
	}

	return result
}

// ─── 路径处理 ────────────────────────────────────────────────────

// normalizePath 把用户口述路径转成可检查的绝对路径：~ 展开、$var/${var} 展开
// （os.ExpandEnv 不认 %var% Windows 写法——保持原样，见 desktop_test.go 已知
// 限制）、相对路径拼 cwd、Clean 词法消解 ..。注意：这里**不做** cwd 圈定——
// 绝对路径是桌面操作的合法输入；越界防护由 evaluatePathPolicy 对解析后路径
// 施加的敏感目录警告与写硬阻断承担（.. 逃逸到 System32 的写入同样被拦，
// TestNormalizePathEscapeStillGuarded 钉死此属性）。
func normalizePath(p, cwd string) string {
	if p == "" {
		return ""
	}
	// 展开 ~
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		p = filepath.Join(home, p[1:])
	}
	// 展开 $var/${var}（%var% 非 os.ExpandEnv 语法，保持原样）
	p = os.ExpandEnv(p)
	// 转绝对路径
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	// 词法规范化：Clean 后路径不再含 ..；此后各检查都看到真实目标
	clean := filepath.Clean(p)
	if !strings.HasPrefix(clean, filepath.VolumeName(clean)+string(os.PathSeparator)) &&
		!strings.HasPrefix(clean, string(os.PathSeparator)) {
		return p // 保持原样
	}
	return clean
}

// ─── 前缀判据的形态归一 ──────────────────────────────────────────
//
// 「路径前缀」判据必须先把路径归一到同一个词法形态，否则判据空间与执行空间分叉：
// Windows 的 `\\?\C:\Windows\System32\x`（扩展长度前缀）、`\\.\C:\...`（设备前缀）、
// `//localhost/C$/Windows/System32/x`（本机管理共享 UNC）、`C:/Windows/System32/x`
// （正斜杠）在文件系统眼里都是同一个位置，但字面 `strings.HasPrefix` 全部不命中。
// 实测（2026-10-02，非管理员）：这些形态下 isHardBlockedWritePath/isSensitivePath
// 均为 false，即「既不拦也不告警」。
//
// 已知残留限制（诚实登记，不假装覆盖）：8.3 短名（`C:\PROGRA~1\...`）与符号链接
// 需要访问文件系统才能解析，本函数只做词法归一，不用 EvalSymlinks（避免在纯策略
// 函数里做 IO）。

// canonicalizeForMatch 把路径归一到「盘符 + 反斜杠 + 小写」的可比较形态。
func canonicalizeForMatch(p string) string {
	if p == "" {
		return ""
	}
	s := strings.ToLower(strings.ReplaceAll(p, "/", `\`))
	// 扩展长度前缀：\\?\C:\... → C:\...
	if strings.HasPrefix(s, `\\?\`) {
		s = s[4:]
		for strings.HasPrefix(s, `\`) {
			s = s[1:]
		}
	}
	// 设备前缀：\\.\C:\... → C:\...
	if strings.HasPrefix(s, `\\.\`) {
		s = s[4:]
		for strings.HasPrefix(s, `\`) {
			s = s[1:]
		}
	}
	// 本机管理共享：\\localhost\c$\... / \\<本机名>\c$\... → c:\...
	if strings.HasPrefix(s, `\\`) {
		if i := strings.Index(s[2:], `\`); i >= 0 {
			host := s[2 : 2+i]
			rest := s[2+i+1:]
			local := strings.ToLower(os.Getenv("COMPUTERNAME"))
			if host == "localhost" || host == "." || (local != "" && host == local) {
				if len(rest) >= 2 && rest[1] == '$' {
					s = rest[0:1] + `:` + rest[2:]
				}
			}
		}
	}
	// 折叠剩余重复分隔符
	for strings.Contains(s, `\\`) {
		s = strings.ReplaceAll(s, `\\`, `\`)
	}
	// 去掉末段的尾随空格与点（Win32 会截断：`System32 ` 与 `System32.` 都是 System32）
	if i := strings.LastIndex(s, `\`); i >= 0 {
		head, tail := s[:i+1], s[i+1:]
		tail = strings.TrimRight(tail, " .")
		s = head + tail
	} else {
		s = strings.TrimRight(s, " .")
	}
	return s
}

// withinRoot 判断 p 是否落在 root 之内（按路径段边界，避免 `c:\windows\system32foo`
// 被 `c:\windows\system32` 误判为命中）。
func withinRoot(p, root string) bool {
	c := canonicalizeForMatch(p)
	r := strings.TrimRight(canonicalizeForMatch(root), `\`)
	if r == "" || c == "" {
		return false
	}
	return c == r || strings.HasPrefix(c, r+`\`)
}

// isSensitivePath 目标位于系统目录（只产出一句确认提示，不阻断）。
func isSensitivePath(p string) bool {
	sensitive := []string{
		`c:\windows`,
		`c:\program files`,
		`c:\program files (x86)`,
	}
	for _, s := range sensitive {
		if withinRoot(p, s) {
			return true
		}
	}
	return false
}

// isHardBlockedWritePath 写类操作的硬阻断清单。清单只有两个根（`C:\Windows\System32`
// 与 `C:\Windows\SysWOW64`）——`C:\Windows\Temp` 这类「允许 + 敏感告警」的既有设计
// 由 desktop_test.go:79 钉死，不在此处扩大。
func isHardBlockedWritePath(p string) bool {
	blocked := []string{
		`c:\windows\system32`,
		`c:\windows\syswow64`,
	}
	for _, b := range blocked {
		if withinRoot(p, b) {
			return true
		}
	}
	return false
}
