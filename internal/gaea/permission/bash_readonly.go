package permission

import "strings"

// readOnlyBashCommands is the set of commands considered read-only — they
// don't modify filesystem state, network state, or process state. Each
// entry is the first word of a bash command (lowercased). Commands not in
// this set that might also be read-only (e.g. "git log") are handled
// separately by isReadOnlyBashSubject.
//
// v4.374（Reasonix v1.15→v1.38 蒸馏，对齐上游 shellsafe/effect.go 与
// permission/bash_approval.go 的修正）：
//   - 移除 "env"——env 可包装任意命令（`env VAR=x rm …`），属间接执行。
//   - 移除 "awk"/"sed"——两者程序体是一门可执行 shell 的迷你语言（awk 的
//     system()/`| getline`、GNU sed 的 `e` 命令；`-f` 脚本文件同样可以），
//     字符串层无法安全分类，fail-closed 一律交回审批（上游同洞已修；
//     上游保留 -f 例外依赖其「已记忆规则」模型，gaea 自动放行无此前提）。
var readOnlyBashCommands = map[string]bool{
	"cat": true, "head": true, "tail": true, "less": true, "more": true,
	"ls": true, "find": true, "locate": true, "which": true, "whereis": true, "type": true,
	"grep": true, "egrep": true, "fgrep": true, "rg": true,
	"echo": true, "printf": true,
	"pwd": true, "whoami": true, "id": true, "uname": true, "hostname": true,
	"date": true, "printenv": true,
	"wc": true, "sort": true, "uniq": true, "cut": true, "tr": true,
	"stat": true, "file": true, "du": true, "df": true,
	"ps": true, "top": true, "htop": true,
	"diff": true, "cmp": true, "comm": true,
	"man": true, "info": true, "help": true,
	"true": true, "false": true, "test": true, "[": true,
	"basename": true, "dirname": true, "realpath": true, "readlink": true,
}

// readOnlyBashPrefixes are command prefixes where the second word
// determines read-only status. Each maps to the set of read-only
// subcommands.
//
// v4.374（Reasonix 蒸馏）：cargo 移除 check/doc——两者会执行 crate 的
// build.rs（任意代码），上游只保留 search（shellsafe effect.go 同口径）。
var readOnlyBashPrefixes = map[string]map[string]bool{
	"git": {
		"log": true, "status": true, "diff": true, "show": true,
		"tag":   true,
		"blame": true, "grep": true, "ls-files": true, "ls-tree": true,
		"rev-parse": true, "rev-list": true, "describe": true, "reflog": true,
		"shortlog": true, "whatchanged": true, "cherry": true,
		"cat-file": true, "for-each-ref": true, "name-rev": true,
	},
	"go": {
		"vet": true, "doc": true, "list": true,
		"version": true, "env": true,
	},
	"npm": {
		"ls": true, "list": true, "view": true, "info": true,
		"outdated": true, "audit": true,
	},
	"cargo": {
		"search": true,
	},
	"docker": {
		"ps": true, "images": true, "inspect": true, "logs": true,
		"stats": true, "info": true, "version": true,
	},
	"kubectl": {
		"get": true, "describe": true, "logs": true, "explain": true,
		"api-resources": true, "api-versions": true,
	},
}

// isReadOnlyBashSubject returns true when a bash command is a known
// read-only operation. The subject is the JSON arg value extracted by
// Subject() — for bash it is the raw command string.
func isReadOnlyBashSubject(subject string) bool {
	cmd := strings.TrimSpace(subject)
	if cmd == "" {
		return false
	}
	if containsShellSyntax(cmd) {
		return false
	}
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	base := strings.ToLower(fields[0])

	// Check single-word read-only commands.
	if readOnlyBashCommands[base] {
		if hasUnsafeReadOnlyArgs(base, fields[1:]) {
			return false
		}
		return true
	}

	// Check prefix commands (git log, go vet, etc.).
	if len(fields) > 1 {
		if sub, ok := readOnlyBashPrefixes[base]; ok {
			subcmd := strings.ToLower(fields[1])
			return sub[subcmd] && !hasUnsafePrefixArgs(base, subcmd, fields[2:])
		}
	}
	return false
}

func containsShellSyntax(cmd string) bool {
	return strings.ContainsAny(cmd, ";|&<>\n`") || strings.Contains(cmd, "$(")
}

func hasUnsafeReadOnlyArgs(base string, args []string) bool {
	switch base {
	case "find":
		// v4.374: 补 -ok/-okdir（交互式执行，同 -exec 危险；Reasonix 同口径）。
		return hasAnyArg(args, "-exec", "-execdir", "-delete", "-ok", "-okdir")
	case "sort":
		return hasArgWithPrefix(args, "-o") || hasAnyArg(args, "--output") || hasArgWithPrefix(args, "--output=")
	}
	return false
}

func hasUnsafePrefixArgs(base, subcmd string, args []string) bool {
	switch base {
	case "git":
		switch subcmd {
		case "diff", "show", "log", "whatchanged":
			// v4.374（Reasonix shellsafe 同口径）：--ext-diff/--textconv 会
			// 执行仓库 .gitattributes 配置的外部程序；--output 已有守卫。
			if hasAnyArg(args, "--output", "--ext-diff", "--textconv") ||
				hasArgWithPrefix(args, "--output=") {
				return true
			}
		case "cat-file":
			// --filters 走 clean/smudge filter=可执行外部程序。
			return hasAnyArg(args, "--filters")
		case "grep":
			// --open-files-in-pager 打开 pager=可执行外部程序。
			return hasAnyArg(args, "--open-files-in-pager")
		}
	case "go":
		if subcmd == "env" {
			return hasAnyArg(args, "-w", "-u")
		}
	}
	return false
}

func hasArgWithPrefix(args []string, prefix string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}

func hasAnyArg(args []string, unsafe ...string) bool {
	for _, arg := range args {
		for _, candidate := range unsafe {
			if arg == candidate {
				return true
			}
		}
	}
	return false
}

// dangerousBashPatterns are glob-like patterns that match destructive
// commands. Used only for a UI warning — the deny list is the actual
// enforcement mechanism.
var dangerousBashPatterns = []struct {
	pattern string
	label   string
}{
	{"rm -rf*", "recursive delete"},
	{"rm -r *", "recursive delete"},
	{"rm -fr*", "recursive delete"},
	{"git push*--force*", "force push"},
	{"git push*-f*", "force push"},
	{"git reset --hard*", "hard reset"},
	{"git clean -f*", "force clean"},
	{"chmod 777*", "world-writable"},
	{"chmod -R 777*", "world-writable recursive"},
	{"chown *", "ownership change"},
	{"sudo *", "superuser"},
	{"mkfs*", "filesystem format"},
	{"dd if=*", "raw device write"},
	{"fdisk*", "partition table"},
	{"> /dev/*", "device overwrite"},
}

// BashDangerWarning returns a short label if subject matches a known
// dangerous pattern, or "" when the command looks safe. This is a visual
// hint only — the Policy rules are the authority.
func BashDangerWarning(subject string) string {
	s := strings.TrimSpace(subject)
	for _, d := range dangerousBashPatterns {
		if matchGlob(d.pattern, s) {
			return d.label
		}
	}
	return ""
}
