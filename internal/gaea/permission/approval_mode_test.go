package permission

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// T4（codex 蒸馏）验收：Never 语义 + Windows 危险删除命令。

// never 模式 → Deny 且理由含 never；其余模式保持 Ask。
func TestNormalizeAskNever(t *testing.T) {
	d, reason := NormalizeAsk("never", "bash")
	if d != Deny || !strings.Contains(reason, "never") || !strings.Contains(reason, "bash") {
		t.Fatalf("never = (%v,%q), want Deny + never 理由", d, reason)
	}
	for _, mode := range []string{"", "on-request", "untrusted", "ask", "NEVER "} {
		if mode == "NEVER " {
			// 大小写/空白归一后仍是 never。
			if d, _ := NormalizeAsk(mode, "bash"); d != Deny {
				t.Fatalf("NEVER(含空白) = %v, want Deny", d)
			}
			continue
		}
		if d, _ := NormalizeAsk(mode, "bash"); d != Ask {
			t.Fatalf("mode=%q = %v, want Ask（现状不变）", mode, d)
		}
	}
}

// Gate 接线：never 下 Ask 工具硬拒绝；Allow 规则命中的工具照常放行；
// 默认 "" 全量等价现状。
func TestGateNeverModeWiring(t *testing.T) {
	g := NewGate(New("", nil, nil, []string{"bash(rm*)"}), nil)
	g.AlwaysAsk = map[string]bool{"cost_save": true}
	g.ApprovalMode = "never"

	// Ask（未命中 allow）：硬拒绝 + 理由含 never。
	allow, reason, err := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit"}`), false)
	if err != nil || allow || !strings.Contains(reason, "never") {
		t.Fatalf("never Ask = (%v,%q,%v), want 拒绝+never 理由", allow, reason, err)
	}
	// hardAsk 工具同样硬拒绝。
	allow, reason, err = g.Check(context.Background(), "cost_save", json.RawMessage(`{"title":"x"}`), false)
	if err != nil || allow || !strings.Contains(reason, "never") {
		t.Fatalf("never hardAsk = (%v,%q,%v), want 拒绝+never 理由", allow, reason, err)
	}
	// 默认模式：回到现状（headless Ask 自治放行）。
	g2 := NewGate(New("", nil, nil, []string{"bash(rm*)"}), nil)
	g2.AlwaysAsk = map[string]bool{"cost_save": true}
	if allow, _, _ := g2.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit"}`), false); !allow {
		t.Fatalf("默认模式现状被改变")
	}
}

// 危险删除判定：直呼 / PowerShell / cmd / bash -c 四形态。
func TestDangerousWindowsCommand(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want bool
	}{
		{"直呼 rm", []string{"rm", "-rf", "build"}, true},
		{"PowerShell 直调", []string{"Remove-Item", "-Force", "x"}, true},
		{"cmd /c del", []string{"cmd", "/c", "del foo"}, true},
		{"cmd /c del /q", []string{"cmd", "/c", "del /q foo"}, true},
		{"cmd 绝对路径", []string{`C:\Windows\System32\cmd.exe`, "/c", "rd /s /q foo"}, true},
		{"powershell -Command", []string{"powershell", "-Command", "Remove-Item x"}, true},
		{"powershell 绝对路径", []string{`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, "-Command", "Remove-Item -Force x"}, true},
		{"bash -c rm", []string{"bash", "-c", "cd /tmp && rm -rf build"}, true},
		{"rm 绝对路径", []string{"/usr/bin/rm", "-rf", "build"}, true},
		{"bash -c 分段尾删", []string{"bash", "-c", "echo hi; rmdir tmp"}, true},
		{"wsl 直通深删", []string{"wsl", "rm", "-rf", "build"}, true},
		{"wsl --exec 深删", []string{"wsl", "--exec", "rm", "-rf", "/tmp/x"}, true},
		{"wsl 带引导开关", []string{"wsl", "--cd", "/tmp", "rm", "-rf", "x"}, true},
		{"草稿目录单文件删除", []string{"rm", "-f", "_tmp_extract/repro/probe.py"}, false},
		{"系统临时单文件删除", []string{"Remove-Item", "-Force", `C:\Users\alice\AppData\Local\Temp\probe.tmp`}, false},
		{"POSIX临时单文件删除", []string{"rm", "/tmp/probe.tmp"}, false},
		{"双横线后草稿删除", []string{"rm", "--", "draft.bak"}, false},
		{"后缀草稿文件删除", []string{"rm", "draft.bak"}, false},
		{"非草稿单文件仍拦", []string{"rm", "-f", "deliverable.md"}, true},
		{"草稿目录递归仍拦", []string{"rm", "-rf", "_tmp_extract/repro"}, true},
		{"草稿白名单禁路径穿越", []string{"rm", "-f", "_tmp_extract/../deliverable.md"}, true},
		{"草稿白名单禁通配符", []string{"rm", "-f", "_tmp_extract/*.py"}, true},
		{"未知删除旗标仍拦", []string{"Remove-Item", "-WhatIf", "_tmp/probe.py"}, true},
		{"wsl 未知开关保守放行", []string{"wsl", "--weird", "rm", "-rf", "x"}, false},
		{"wsl 无命令", []string{"wsl"}, false},
		{"wsl 只读命令", []string{"wsl", "ls", "-la"}, false},
		{"未知命令不拦", []string{"bash", "-c", "git push origin main"}, false},
		{"读命令不拦", []string{"bash", "-c", "ls -la"}, false},
		{"空参数不拦", []string{}, false},
		{"powershell 未知开关保守放行", []string{"powershell", "-WeirdFlag", "x"}, false},
		{"cmd 未知开关保守放行", []string{"cmd", "/z", "del foo"}, false},
	}
	for _, tc := range cases {
		got, _ := DangerousWindowsCommand(tc.argv)
		if got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
	// 验收例：Delete-Item -Force x → 拦截且理由提到强制删除。
	ok, why := DangerousWindowsCommand([]string{"Delete-Item", "-Force", "x"})
	if !ok || !strings.Contains(why, "forced") {
		t.Fatalf("Delete-Item -Force 拦截理由不符: (%v,%q)", ok, why)
	}
}

// Gate 接线：普通删除直接放行；灾难面由 builtin.bash 执行期守卫拒绝。
func TestGateAllowsDirectBashDeletes(t *testing.T) {
	// 权限策略允许后，不得再被静态 rm 黑名单重复拦截。
	g := NewGate(New("allow", []string{"bash"}, nil, nil), nil)
	allow, reason, err := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"rm -rf build"}`), false)
	if err != nil || !allow {
		t.Fatalf("rm -rf build = (%v,%q,%v), want 直接放行", allow, reason, err)
	}
	allow, reason, err = g.Check(context.Background(), "bash", json.RawMessage(`{"command":"rm deliverable.md"}`), false)
	if err != nil || !allow {
		t.Fatalf("rm file = (%v,%q,%v), want 直接放行", allow, reason, err)
	}
	allow, reason, err = g.Check(context.Background(), "bash", json.RawMessage(`{"command":"Remove-Item -Recurse -Force .\\dist"}`), false)
	if err != nil || !allow {
		t.Fatalf("Remove-Item child dir = (%v,%q,%v), want 直接放行", allow, reason, err)
	}
	// 普通命令不受影响。
	if allow, _, err := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"echo hi"}`), false); err != nil || !allow {
		t.Fatalf("普通命令 = (%v,%v), want 放行", allow, err)
	}
}
