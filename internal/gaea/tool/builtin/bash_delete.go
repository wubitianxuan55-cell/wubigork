package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ── 递归删除硬拒绝（Reasonix 2.x shellsafe/delete 蒸馏，v4.524 刀 A6）──────
//
// 对「灾难面」的递归删除在**任何形态下**硬拒绝——不走审批、无豁免（上游
// 语义：人已批准/Auto/Yolo 都拦）。灾难面=文件系统根/卷根、home 本身及其
// 祖先、工作目录本身及其祖先（删掉工作区=毁掉整个项目含 .git）。
//
// 取道不取器的刻意偏差（记录在案）：上游对「工作区外」的递归删除也一律拒
// （boundedDeleteTarget 只认工作区+授权写根），因为它有审批阶梯可把这类操
// 作路由给人；gaea bash 无审批 UX，一刀切会断掉合法的 temp 清理（书架隔离、
// clean-tmp 都是既有实弹流），故本 guard 只拦灾难面。语法引擎形态对齐上游：
// 包装解包（sudo/env/nohup…）/未知基座递归参数扫描/find -exec 嵌套/未知旗标
// fail-closed/字面性字符集——管线输入（find|xargs）按上游精神以 find 起点为
// 删除树根分类（上游直接拒，此处略宽、灾难保护不减）。
//
// 诚实边界：这是静态字符串分类，不是 OS 沙箱——符号链接交换（TOCTOU）不在
// 防护面内，与上游声明一致。

// deleteExtent 递归删除分析结论（上游 DeleteExtent 同构）。
type deleteExtent struct {
	recursive bool
	// unknownTarget=true 表示删除集合的根无法静态确定（裸 xargs/未知包装/
	// 空目标）——fail-closed。
	unknownTarget bool
	// unknownOption=true 表示出现解析器不认识的旗标（上游 delete_option 语义）。
	unknownOption bool
	targets       []string
}

// deleteVerdict 递归删除分类结论。
type deleteVerdict struct {
	reject bool
	reason string // reject 时给模型的诚实说明（含具体路径与出路）
}

// guardRecursiveDelete 在命令起跑前分类递归删除目标。workDir 是本次调用实际
// 的 cmd.Dir（状态锚已解析；空串=进程 cwd，与 exec 语义一致）。
// shellPowerShell 为 true 时按 PowerShell 词法归类。
func guardRecursiveDelete(command, workDir string, shellPowerShell bool) deleteVerdict {
	for _, seg := range splitShellSegments(command) {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		fields := splitShellFields(seg)
		if len(fields) == 0 {
			continue
		}
		ext := analyzeDeleteCommand(fields[0], fields[1:], shellPowerShell)
		if !ext.recursive {
			continue
		}
		targets := ext.targets
		if ext.unknownTarget {
			targets = append(targets, "\x00nonliteral")
		}
		if ext.unknownOption {
			return deleteVerdict{reject: true, reason: "recursive delete uses options the safety " +
				"parser cannot classify; use known options (rm: -f -i -I -r -R -v -d --recursive --force; " +
				"Remove-Item: -Recurse -Force -Path) with explicit values"}
		}
		for _, raw := range targets {
			if !literalDeletePath(raw) {
				return deleteVerdict{reject: true, reason: "recursive delete target is not a literal path " +
					"(variable/glob/command substitution/pipeline input): re-run with explicit literal paths " +
					"so the delete can be checked"}
			}
			if v := classifyDeleteTarget(raw, workDir); v.reject {
				return v
			}
		}
	}
	return deleteVerdict{}
}

// analyzeDeleteCommand 单段删除分析（上游 AnalyzeRecursiveDelete 同构）：
// 先解包装（sudo/env/nohup…），再按基座分发；未知基座做递归参数扫描
// （递归删除藏在任意位置参数里=unknownTarget fail-closed）。
func analyzeDeleteCommand(name string, args []string, powerShell bool) deleteExtent {
	base, rest, known := unwrapDeleteCommand(name, args)
	if !known {
		return scanRecursiveArgs(rest, powerShell)
	}
	switch base {
	case "cmd":
		return cmdDelete(rest)
	case "xargs":
		return scanRecursiveArgs(rest, powerShell)
	case "find":
		return findDelete(rest)
	case "rm", "remove-item", "ri", "del", "erase", "rd", "rmdir":
		return directDelete(base, rest, powerShell)
	default:
		// 无关基座（echo/ls/go …）不放删除语义——上游同构：只有解包失败与
		// xargs 走递归参数扫描，避免 `echo rm -rf x` 之类误伤。
		return deleteExtent{}
	}
}

// scanRecursiveArgs 未知基座的递归参数扫描（上游 recursiveArgumentDelete）：
// 任何位置参数自身构成递归删除调用（如 `sudo -k rm -rf /` 解包失败后的
// 残参）⇒ 删除树根未知，fail-closed。
func scanRecursiveArgs(args []string, powerShell bool) deleteExtent {
	for i, arg := range args {
		e := analyzeDeleteCommand(arg, args[i+1:], powerShell)
		if e.recursive {
			return deleteExtent{recursive: true, unknownTarget: true}
		}
	}
	return deleteExtent{}
}

// unwrapDeleteCommand 解包装层（上游 unwrapDeleteCommand 同构）：command/env/
// exec/builtin/sudo/nohup 后找真实基座；env 赋值与白名单旗标跳过，未知旗标
// known=false 转递归参数扫描。
func unwrapDeleteCommand(name string, args []string) (string, []string, bool) {
	base := executableBase(name)
	for {
		switch base {
		case "command", "env", "exec", "builtin", "sudo", "nohup":
		default:
			return base, args, true
		}
		wrapper := base
		for len(args) > 0 {
			arg := args[0]
			if wrapper == "command" && (arg == "-v" || arg == "-V") {
				return "", nil, true
			}
			if arg == "--" {
				args = args[1:]
				break
			}
			if wrapper == "env" && isShellAssignment(arg) || wrapper == "command" && arg == "-p" {
				args = args[1:]
				continue
			}
			if arg == "" {
				return "", args, false
			}
			if strings.HasPrefix(arg, "-") {
				switch arg {
				case "-E", "-n", "-i", "--ignore-environment", "--non-interactive":
					args = args[1:]
					continue
				case "-u", "-g", "--user", "--group":
					if len(args) < 2 {
						return "", args, false
					}
					args = args[2:]
					continue
				default:
					return "", args, false
				}
			}
			break
		}
		if len(args) == 0 {
			return "", nil, true
		}
		base, args = executableBase(args[0]), args[1:]
	}
}

// executableBase 取可执行基座名（去目录前缀+小写）。
func executableBase(name string) string {
	base := name
	if i := strings.LastIndexAny(base, `/\\`); i >= 0 {
		base = base[i+1:]
	}
	return strings.ToLower(base)
}

// isShellAssignment 形如 NAME=VALUE 的环境赋值。
func isShellAssignment(arg string) bool {
	eq := strings.Index(arg, "=")
	return eq > 0 && !strings.HasPrefix(arg, "-") && isIdent(arg[:eq])
}

func isIdent(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return len(s) > 0
}

// cmdDelete 处理 `cmd /c rd /s /q X`（Git Bash 可达 cmd.exe）：/s=递归，
// 复杂字符（^%&|）⇒ 动态负载 fail-closed。
func cmdDelete(args []string) deleteExtent {
	if len(args) == 0 || !strings.EqualFold(args[0], "/c") {
		return deleteExtent{}
	}
	if len(args) == 2 {
		args = splitShellFields(args[1])
	} else {
		args = args[1:]
	}
	if len(args) == 0 {
		return deleteExtent{}
	}
	recursive, complex := false, false
	for _, arg := range args {
		recursive = recursive || strings.EqualFold(arg, "/s")
		complex = complex || strings.ContainsAny(arg, "^%&|\r\n")
	}
	if recursive && complex {
		return deleteExtent{recursive: true, unknownTarget: true}
	}
	return analyzeDeleteCommand(args[0], args[1:], true)
}

// directDelete rm/Remove-Item 族直接删除：递归旗标+目标提取+未知旗标
// fail-closed（上游 posixDeleteOption/powerShellDeleteOption 同构）。
func directDelete(base string, args []string, powerShell bool) deleteExtent {
	ext := deleteExtent{}
	options := true
	psParameters := powerShell && (base == "remove-item" || base == "ri" || base == "rm" ||
		base == "del" || base == "erase" || base == "rd" || base == "rmdir" || base == "")
	// cmd 语法旗标（/s /q）：rd/rmdir/del 在 PowerShell/cmd 语境下走 cmd 词法
	// （上游 cmdParameters 口径）。
	cmdParameters := powerShell && (base == "del" || base == "erase" || base == "rd" || base == "rmdir")
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if options && arg == "--" {
			options = false
			continue
		}
		if options && strings.HasPrefix(arg, "-") {
			if psParameters {
				i += psDeleteOption(&ext, arg, args[i+1:])
			} else {
				posixDeleteOption(&ext, arg)
			}
			continue
		}
		if options && cmdParameters && strings.HasPrefix(arg, "/") {
			cmdDeleteOption(&ext, arg)
			continue
		}
		ext.targets = append(ext.targets, arg)
	}
	if !ext.recursive {
		return deleteExtent{}
	}
	if len(ext.targets) == 0 {
		// 递归删除无显式目标（xargs 语境/笔误）⇒ 删除树根未知
		return deleteExtent{recursive: true, unknownTarget: true}
	}
	return ext
}

// psDeleteOption PowerShell 旗标：recurse 可缩写、冒号内联值（-Path:C:\x）、
// -Recurse:$false 不算递归、取值参数吞下一个词、未知旗标 fail-closed。
// 返回消耗掉的额外参数个数。
func psDeleteOption(ext *deleteExtent, arg string, rest []string) int {
	key, value, inline := strings.Cut(strings.TrimPrefix(strings.ToLower(arg), "-"), ":")
	switch {
	case key != "" && strings.HasPrefix("recurse", key):
		ext.recursive = ext.recursive || value != "$false"
	case key != "" && strings.HasPrefix("force", key):
	case key == "literalpath" || key == "path":
	case key == "erroraction" || key == "ea" || key == "warningaction" || key == "wa" ||
		key == "informationaction" || key == "confirm":
		if !inline {
			if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
				return 1
			}
			ext.unknownOption = true
		}
	case key == "verbose" || key == "debug" || key == "whatif":
	default:
		ext.unknownOption = true
	}
	return 0
}

// posixDeleteOption rm 旗标（上游同构）：短旗标字符白名单 rfRvidI、长旗标
// 白名单（--recursive 可前缀缩写），其余 unknownOption fail-closed。
func posixDeleteOption(ext *deleteExtent, arg string) {
	if strings.HasPrefix(arg, "--") {
		if strings.HasPrefix("--recursive", arg) {
			ext.recursive = true
		} else if arg != "--force" && arg != "--verbose" && arg != "--dir" {
			ext.unknownOption = true
		}
		return
	}
	ext.recursive = ext.recursive || strings.ContainsAny(arg[1:], "rR")
	ext.unknownOption = ext.unknownOption || strings.Trim(arg[1:], "rfRvidI") != ""
}

// cmdDeleteOption cmd 词法旗标（上游同构）：/s=递归，/q /f 已知，其余 fail-closed。
func cmdDeleteOption(ext *deleteExtent, arg string) {
	switch strings.ToLower(arg) {
	case "/s":
		ext.recursive = true
	case "/q", "/f":
	default:
		ext.unknownOption = true
	}
}

// findDelete find 两种删除形态：-delete 直删、-exec/-execdir 嵌套删除负载
// （rm -rf {} 之类）；-follow/-L 符号链接跟入 ⇒ unknownOption。起点=删除树根。
func findDelete(args []string) deleteExtent {
	ext := deleteExtent{}
	var extra []string
	for i, arg := range args {
		switch arg {
		case "-delete":
			ext.recursive = true
		case "-follow", "-L":
			ext.unknownOption = true
		case "-exec", "-execdir":
			if i+1 >= len(args) {
				continue
			}
			nested := findExecDelete(args[i+1:])
			ext.recursive = ext.recursive || nested.recursive
			ext.unknownOption = ext.unknownOption || nested.unknownOption
			if nested.recursive {
				extra = append(extra, nested.targets...)
				if arg == "-execdir" {
					extra = append(extra, "\x00nonliteral") // 相对当前命中项，树根不可静态定
				}
			}
		}
	}
	var starts []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		starts = append(starts, arg)
	}
	if len(starts) == 0 {
		starts = []string{"."}
	}
	ext.targets = append(starts, extra...)
	return ext
}

// findExecDelete 分析 find -exec 的负载（到 ; 或 + 截止），剥 {} 占位。
func findExecDelete(args []string) deleteExtent {
	tail := args[1:]
	for j, v := range tail {
		if v == ";" || v == "+" {
			tail = tail[:j]
			break
		}
	}
	ext := analyzeDeleteCommand(args[0], tail, false)
	var targets []string
	for _, t := range ext.targets {
		if t != "{}" {
			targets = append(targets, t)
		}
	}
	ext.targets = targets
	return ext
}

// splitShellSegments 按 && || ; | 切分命令为段（引号内退化切开：只影响
// 分类精度，不影响 fail-closed 方向——灾难目标在切开后的某段里仍会被拦）。
func splitShellSegments(command string) []string {
	segs := strings.FieldsFunc(command, func(r rune) bool {
		return r == ';' || r == '|'
	})
	var out []string
	for _, s := range segs {
		out = append(out, strings.Split(s, "&&")...)
	}
	return out
}

// splitShellFields 按空白切词（引号折叠为字面内容，够分类用）。
func splitShellFields(seg string) []string {
	var fields []string
	var cur strings.Builder
	quote := byte(0)
	flush := func() {
		if cur.Len() > 0 {
			fields = append(fields, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(seg); i++ {
		c := seg[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			} else {
				cur.WriteByte(c)
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
		case ' ', '\t', '\n', '\r':
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return fields
}

// literalDeletePath 目标字面性（上游 literalDeletePath 同构）：空串/变量/
// glob/花括号展开/反引号命令替换/`~user` 具名展开都不可静态定界。
func literalDeletePath(target string) bool {
	if target == "\x00nonliteral" {
		return false
	}
	if target == "" || strings.ContainsAny(target, "$%*?[]{}`\x00") {
		return false
	}
	if strings.HasPrefix(target, "~") && len(target) > 1 && target[1] != '/' && target[1] != '\\' {
		return false // ~user 形态
	}
	return true
}

// classifyDeleteTarget 单目标灾难面判定：相对路径对 workDir 解析，~ 展开
// 到 home；命中根/home 系/workDir 系给拒绝结论。
// 次序敏感：Windows 的 filepath.Clean 会把 "/" 规范化成 "\\"（且对 "C:"
// 补点成 "C:."）——MSYS 翻译与 POSIX 命名空间分类必须打在原始词上、先于
// 一切 Clean。
func classifyDeleteTarget(raw, workDir string) deleteVerdict {
	if raw == "\x00nonliteral" || !literalDeletePath(raw) {
		return deleteVerdict{reject: true, reason: "recursive delete target is not a literal path " +
			"(variable/glob/command substitution/pipeline input): re-run with explicit literal paths " +
			"so the delete can be checked"}
	}
	p := raw
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return deleteVerdict{reject: true, reason: fmt.Sprintf("cannot resolve home for recursive delete target %q", raw)}
		}
		p = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "~/"))
	}
	// ① 原始词卷根（"C:" "C:\" "/"——Clean 前的形态才可识别盘符相对路径）
	if isVolumeRoot(p) {
		return rejectDelete(raw, p, "a filesystem/volume root")
	}
	// ② Windows + Git Bash：MSYS 盘符挂载（/c/foo → C:\foo）翻译成原生路径
	if runtime.GOOS == "windows" {
		if vol := msysDrivePath(p); vol != "" {
			p = vol
		}
	}
	// ③ Windows 宿主的 POSIX 绝对路径（/home/... 等落在 Git 安装根下）：留在
	// POSIX 命名空间分类——与 workDir 同形态（测试/MSYS 环境）才比较；真实
	// 灾难面只剩 "/" 本身（①已拦）。
	if runtime.GOOS == "windows" && strings.HasPrefix(p, "/") {
		if workDir != "" && posixSameOrAncestor(strings.ToLower(p), strings.ToLower(filepath.ToSlash(filepath.Clean(workDir)))) {
			return rejectDelete(raw, p, "the working directory (workspace) or one of its ancestors")
		}
		return deleteVerdict{}
	}
	// ④ 常规原生路径
	p = filepath.Clean(p)
	if isVolumeRoot(p) {
		return rejectDelete(raw, p, "a filesystem/volume root")
	}
	if !filepath.IsAbs(p) {
		p = filepath.Clean(filepath.Join(workDirOrCwd(workDir), p))
		if isVolumeRoot(p) {
			return rejectDelete(raw, p, "a filesystem/volume root")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		home = filepath.Clean(home)
		if sameOrAncestor(p, home) {
			return rejectDelete(raw, p, "the home directory or one of its ancestors")
		}
	}
	if workDir != "" {
		wd := filepath.Clean(workDir)
		if sameOrAncestor(p, wd) {
			return rejectDelete(raw, p, "the working directory (workspace) or one of its ancestors")
		}
	}
	return deleteVerdict{}
}

// posixSameOrAncestor POSIX 命名空间的同目录/祖先判定（纯字符串，不经
// filepath——Windows 的 Clean/Rel 都按 "\\" 语义走，会破坏 POSIX 形态）：
// p == dir 或 dir 在 p 子树内（dir 以 p+"/" 为前缀）。
func posixSameOrAncestor(p, dir string) bool {
	return p == dir || strings.HasPrefix(dir, p+"/")
}

// sameOrAncestor 报告 p 是否等于 dir 或是 dir 的（真）祖先（删目录本身与其
// 祖先同罪）。判定方向：Rel(p, dir) 无 ".." 前缀 ⇔ dir 在 p 子树内 ⇔ p 是
// dir 的祖先——Rel(dir, p) 方向会把「p 在保护区外但非祖先」误判进来。
// Windows 路径大小写不敏感，比较前统一小写（仅用于分类，不参与实际删除）。
func sameOrAncestor(p, dir string) bool {
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
		dir = strings.ToLower(dir)
	}
	if p == dir {
		return true
	}
	rel, err := filepath.Rel(p, dir)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..")
}

// msysDrivePath 把 Git Bash 的盘符挂载形态翻译成 Windows 路径：/c/foo →
// C:\foo；非盘符形态返回空串。仅 Windows 宿主调用。
func msysDrivePath(p string) string {
	if len(p) < 3 || p[0] != '/' || p[2] != '/' {
		return ""
	}
	c := p[1]
	if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
		return ""
	}
	return strings.ToUpper(string(c)) + ":" + strings.ReplaceAll(p[2:], "/", "\\")
}

// isVolumeRoot 识别文件系统根与卷根：/、C:\、C:（盘符当前目录——rm 递归
// 删「整个卷的当前目录」同样是灾难面）。
func isVolumeRoot(p string) bool {
	if p == "/" || p == string(filepath.Separator) {
		return true
	}
	if len(p) == 2 && p[1] == ':' { // "C:"
		return true
	}
	if len(p) == 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/') { // "C:\"
		return true
	}
	return false
}

func workDirOrCwd(workDir string) string {
	if workDir != "" {
		return workDir
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func rejectDelete(raw, resolved, what string) deleteVerdict {
	return deleteVerdict{reject: true, reason: fmt.Sprintf(
		"refusing recursive delete of %q (resolves to %s): this is %s. "+
			"Delete specific subdirectories or files instead; if the whole workspace must be "+
			"reset, ask the user to do it", raw, resolved, what)}
}
