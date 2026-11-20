package permission

import "strings"

// Windows 破坏性删除命令判定（codex 蒸馏 T4）：夜间/无人值守的误删要有硬
// 拦截面。纯文本判定、零平台依赖（argv 语义按 Windows 壳约定），认不出的
// 一律返回 false——保守姿态：不确定不等于危险，宁可漏报不误报。

// windowsDeleteCmdlets 是删除类命令集合：PowerShell cmdlet 全名 + 别名
// （DELETE_CMDLETS 同构，另收 delete-item 兜验收例）；rm 同时是 bash/MSYS
// 的常用删除命令，深删不分壳。
var windowsDeleteCmdlets = map[string]bool{
	"remove-item": true,
	"delete-item": true,
	"ri":          true,
	"rm":          true,
	"del":         true,
	"erase":       true,
	"rd":          true,
	"rmdir":       true,
}

// powershellStartupFlags 是认识的 PowerShell 启动开关（含带值开关）：
// 之外的 dash token 一律保守放行（返回不拦截）。
var powershellStartupFlags = map[string]bool{
	"-noprofile": true, "-noninteractive": true, "-nologo": true,
	"-executionpolicy": true, "-command": true, "-c": true, "-file": true,
}

// cmdLeadingFlags 是 cmd.exe 的引导开关：剥离后首个 token 视为命令行。
var cmdLeadingFlags = map[string]bool{
	"/c": true, "/k": true, "/d": true, "/s": true, "/q": true,
	"/v": true, "/x": true, "/a": true, "/u": true,
}

// DangerousWindowsCommand 保留命令形态诊断；运行时 Gate 已不再调用它，
// 删除是否放行由 builtin.bash 结合实际 cwd/工作区的 guardRecursiveDelete
// 判定。普通 rm/Remove-Item 不应在这里被“看到即拦”。
// argv[0] 是可执行文件，其余为参数；返回（是否拦截, 理由）。
// 三层识别：argv[0] 直呼删除命令（PowerShell 直调 cmdlet / bash 直调 rm）；
// powershell/pwsh 壳：剥认识的启动开关后取命令首词；cmd 壳：剥 /c 等引导
// 开关后取命令首词。bash/sh -c 同理取命令串，按 && ; | 分段扫首词，
// 连式深删也逃不掉。带 -Force/-rf 判高危并写进理由。
func DangerousWindowsCommand(argv []string) (bool, string) {
	if len(argv) == 0 {
		return false, ""
	}
	prog := normalizeExe(argv[0])
	// 段落：从待检命令串来；直呼形态即 argv[1:]。
	var segments []string
	switch {
	case windowsDeleteCmdlets[prog]:
		return deleteVerdict(prog, argv[1:])
	case prog == "powershell" || prog == "pwsh":
		cmd, ok := powershellCommand(argv[1:])
		if !ok {
			return false, ""
		}
		segments = splitShellSegments(cmd)
	case prog == "cmd":
		cmd, ok := cmdCommand(argv[1:])
		if !ok {
			return false, ""
		}
		segments = splitShellSegments(cmd)
	case prog == "bash" || prog == "sh" || prog == "zsh":
		cmd, ok := shellDashC(argv[1:])
		if !ok {
			return false, ""
		}
		segments = splitShellSegments(cmd)
	case prog == "wsl":
		// 审计补（2026-11-17）：wsl 直通形态（`wsl [--cd <p>] … rm -rf x` /
		// `wsl --exec rm …`）此前逃过判定——wsl 不走 -c，命令是其尾部参数。
		cmd, ok := wslCommand(argv[1:])
		if !ok {
			return false, ""
		}
		segments = splitShellSegments(cmd)
	default:
		return false, ""
	}
	for _, seg := range segments {
		words := strings.Fields(seg)
		if len(words) == 0 {
			continue
		}
		head := normalizeExe(words[0])
		if windowsDeleteCmdlets[head] {
			if blocked, why := deleteVerdict(head, words[1:]); blocked {
				return true, why
			}
			continue
		}
	}
	return false, ""
}

// deleteVerdict（D5，2026-11-20 缺陷清单）：草稿区白名单——非递归删除且
// 全部目标落在草稿形态（_tmp*/_draft 目录、.bak/.tmp/.old 后缀、系统临时
// 根）时不拦。实测「删自己刚写坏的探针文件」被全量拦截，绕行产物=目录里
// .bak/_v2/_v3 冗余堆、反而增加误读旧文件概率；递归删除、目录删除（rd/
// rmdir）与未知旗标形态一律保守拦截（灾难面守卫另在 bash_delete.go 分层）。
func deleteVerdict(head string, rest []string) (bool, string) {
	inherentRecursive := head == "rd" || head == "rmdir"
	recursive := inherentRecursive
	var targets []string
	options := true
	for _, tok := range rest {
		if tok == "" {
			continue
		}
		if options && tok == "--" {
			options = false
			continue
		}
		// PowerShell uses '-'; cmd's switches use '/' without another path
		// separator. `/tmp/probe.tmp` is a POSIX absolute path, not a switch.
		isOption := options && (tok[0] == '-' || tok[0] == '/' && !strings.Contains(tok[1:], "/"))
		if isOption {
			t := strings.ToLower(strings.TrimLeft(tok, "-/"))
			switch t {
			case "r", "rf", "fr", "re", "re ", "recurse", "recursive":
				recursive = true
			case "force", "f", "i", "v", "q", "q:":
				// 确认跳过类旗标：不改变递归性
			case "path", "literalpath", "include", "exclude", "filter":
				// PowerShell 带值开关：本层不追踪取值，保守拦截
				return true, deleteReason(head, rest)
			default:
				return true, deleteReason(head, rest)
			}
			continue
		}
		targets = append(targets, tok)
	}
	if recursive || len(targets) == 0 {
		return true, deleteReason(head, rest)
	}
	for _, t := range targets {
		if !draftDeleteTarget(t) {
			return true, deleteReason(head, rest)
		}
	}
	return false, ""
}

// draftDeleteTarget 报告路径 token 是否落在草稿/临时形态（大小写不敏感、
// 斜杠归一）。只认完整路径段（_tmp* / _draft）与临时后缀；含上行目录、
// 通配符、变量或命令替换的 token 一律放弃白名单，防 `_tmp/../交付物`
// 借白名单穿越到正式文件。
func draftDeleteTarget(tok string) bool {
	t := strings.ToLower(strings.Trim(tok, `"'`))
	t = strings.ReplaceAll(t, `\`, "/")
	if t == "" {
		return false
	}
	if strings.Contains(t, "..") || strings.ContainsAny(t, "*?$`") || strings.Contains(t, "\x00") {
		return false
	}
	if strings.HasSuffix(t, ".bak") || strings.HasSuffix(t, ".tmp") || strings.HasSuffix(t, ".old") ||
		strings.HasPrefix(t, "/tmp/") ||
		strings.HasPrefix(t, "c:/users/") && strings.Contains(t, "/appdata/local/temp/") {
		return true
	}
	for _, seg := range strings.Split(t, "/") {
		if seg == "_draft" || strings.HasPrefix(seg, "_tmp") {
			return true
		}
	}
	return false
}

// deleteReason 组装拦截理由：命令名 + 高危标记（-Force/-rf 组合）。
func deleteReason(cmd string, rest []string) string {
	force := hasForceFlag(rest)
	why := "destructive delete command"
	if force {
		why = "destructive forced delete command"
	}
	return why + " (" + cmd + ") is blocked by the safety gate — " +
		"do not retry the blocked target; delete a concrete child path directly instead. " +
		".trash/ is not required for ordinary deletes"
}

// hasForceFlag 报告参数里是否带强制删除标记（PowerShell -Force / bash -rf、
// -fr、独立 -r -f）。
func hasForceFlag(rest []string) bool {
	for _, tok := range rest {
		t := strings.ToLower(strings.TrimLeft(tok, "-/"))
		switch t {
		case "force", "rf", "fr", "r":
			return true
		case "f":
			// 独立 -f 不算强删（常见于覆盖确认跳过），配 -r 才判。
		}
	}
	return false
}

// normalizeExe 归一命令名：小写、去引号、去 .exe/.cmd/.ps1 后缀。
func normalizeExe(s string) string {
	s = strings.Trim(strings.ToLower(strings.TrimSpace(s)), `"`)
	// 绝对路径同样是合法入口：`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	// 或 MSYS 的 /usr/bin/rm 不能因为前缀路径绕过直呼命令判定。
	if i := strings.LastIndexAny(s, `/\`); i >= 0 {
		s = s[i+1:]
	}
	for _, suf := range []string{".exe", ".cmd", ".bat", ".ps1"} {
		if strings.HasSuffix(s, suf) {
			s = strings.TrimSuffix(s, suf)
			break
		}
	}
	return s
}

// powershellCommand 从 PowerShell 参数里取命令串；遇到不认识的 dash token
// 直接放弃（保守：不拦截）。-Command/-c 后的下一个 token 即命令串。
func powershellCommand(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		tok := strings.ToLower(args[i])
		if !strings.HasPrefix(tok, "-") {
			// -File 场景：脚本路径不判（交给调用方闸门），继续扫。
			continue
		}
		if !powershellStartupFlags[tok] {
			return "", false
		}
		if tok == "-command" || tok == "-c" {
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", false
		}
		if tok == "-executionpolicy" {
			i++ // 带值开关，跳过值
		}
	}
	return "", false
}

// cmdCommand 从 cmd.exe 参数里剥引导开关取命令串；无法识别的引导 token
// 直接放弃（保守）。
func cmdCommand(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		tok := strings.ToLower(args[i])
		if strings.HasPrefix(tok, "/") && !cmdLeadingFlags[tok] {
			return "", false
		}
		if tok == "/c" || tok == "/k" {
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", false
		}
	}
	return "", false
}

// shellDashC 从 bash/sh 参数里取 -c 的命令串。
func shellDashC(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" {
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", false
		}
	}
	return "", false
}

// wsl 启动开关：带值开关（跳过下一个 token）与命令起始标记。
var (
	wslValueFlags = map[string]bool{
		"--cd": true, "--user": true, "-u": true,
		"--distribution": true, "-d": true, "--system": true, "-s": true,
	}
	wslFlagFlags = map[string]bool{
		"--exec": true, "-e": true, "--": true, "--shell-type": true,
		"--system-distribution": true, "--debug-shell": true,
	}
)

// wslCommand 从 wsl 参数里取目标命令串：剥引导开关后其余 token 即命令
// （wsl 无 -c 形态）。遇无法识别的开关直接放弃（保守：认不出不拦）。
// 无命令参数（如裸 `wsl` 进交互 shell）也返回 false——交互态无从判定。
func wslCommand(args []string) (string, bool) {
	i := 0
	for i < len(args) {
		tok := strings.ToLower(args[i])
		switch {
		case wslValueFlags[tok]:
			i += 2 // 开关 + 值
			continue
		case wslFlagFlags[tok]:
			rest := args[i+1:]
			if len(rest) == 0 {
				return "", false
			}
			return strings.Join(rest, " "), true
		case strings.HasPrefix(tok, "-"):
			return "", false // 未知开关：保守放行
		default:
			return strings.Join(args[i:], " "), true
		}
	}
	return "", false
}

// splitShellSegments / isAmpAmpBoundary 已并入 argv_rule.go（T10 引入
// 引号感知版分段，一处一处纪律）：本文件经它分段后逐段判定。
