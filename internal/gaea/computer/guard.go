package computer

// W5 输入注入闸门（施工单 W5 三条硬约束之三：deny 表）——本地硬拒，
// 先于任何策略/审批。红线取 asset-62 §8.3 不可谈判清单的机器可判子集：
// Win 键族 / 终端 / 认证 / 密码管理器 / 安全软件 / ChatGPT·Codex 自身 UI。
// 纯字符串逻辑，平台无关（测试两平台都能跑）。
import (
	"fmt"
	"strings"
)

// forbiddenKeyFragments 键名硬拒片段（按 + 分段后逐段小写匹配）。
var forbiddenKeyFragments = []string{"win", "meta", "cmd", "super", "os"}

// dangerousWindowMatchers 目标窗口硬拒表：标题/进程路径任一命中即拒
// （大小写不敏感；进程按完整路径与 exe 基名两路匹配）。
var dangerousWindowMatchers = []struct {
	category string
	frag     []string
}{
	{"终端", []string{"cmd.exe", "powershell", "pwsh.exe", "windowsterminal", "windows terminal", "wt.exe", "conhost"}},
	{"密码管理器", []string{"1password", "bitwarden", "keepass", "lastpass", "enpass", "密码"}},
	{"认证", []string{"credential", "认证", "登录 windows", "sign in to windows", "windows 安全"}},
	{"安全软件", []string{"defender", "安全中心", "antivirus", "火绒", "360安全", "kaspersky", "mcafee", "norton"}},
	{"AI 桌面端自身", []string{"chatgpt", "codex"}},
}

// GuardChord 键名 deny 表：任一分段含 Win/Meta/Cmd/Super/OS → ErrForbiddenKey
// （不产生任何 syscall）。
func GuardChord(chord string) error {
	for _, part := range strings.Split(chord, "+") {
		p := strings.ToLower(strings.TrimSpace(part))
		if p == "" {
			continue
		}
		for _, f := range forbiddenKeyFragments {
			if p == f || strings.HasPrefix(p, f) {
				return fmt.Errorf("%w: %s（不可谈判红线：Win/Meta/Cmd/Super/OS 族组合键）", ErrForbiddenKey, chord)
			}
		}
	}
	return nil
}

// GuardWindow 目标窗口 deny 表：标题/进程命中终端/认证/密码/安全/自身 →
// 拒绝并说明命中类别（先于策略与审批，绕不过去）。
func GuardWindow(w Window) error {
	title := strings.ToLower(w.Title)
	app := strings.ToLower(w.App)
	for _, m := range dangerousWindowMatchers {
		for _, f := range m.frag {
			if strings.Contains(title, f) || strings.Contains(app, f) {
				return fmt.Errorf("目标窗口命中 %s 硬拒表（%q），拒绝注入输入: %s", m.category, f, w.Title)
			}
		}
	}
	return nil
}
