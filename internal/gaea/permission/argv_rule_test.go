package permission

import (
	"context"
	"encoding/json"
	"testing"
)

// T10（codex 蒸馏）验收：argv 前缀规则——「记住这条命令」不放行整类命令。

// 解析与渲染往返；非法形态拒绝。
func TestParseArgvRuleText(t *testing.T) {
	tool, r, ok := ParseArgvRuleText(`bash prefix ["git","status","--porcelain"]`)
	if !ok || tool != "bash" || r.First != "git" || len(r.Rest) != 2 {
		t.Fatalf("解析失败: %q %v", tool, r)
	}
	if got := ArgvRuleText(tool, r); got != `bash prefix ["git","status","--porcelain"]` {
		t.Fatalf("往返不符: %q", got)
	}
	for _, bad := range []string{
		"prefix []",
		"prefix [\"git\"]",       // 空 tool——解析文本形态时 tool 不得为空
		"bash prefix git",        // 缺列表
		"bash prefix [\",...\"]", // 空 token
		"bash(git status)",       // 既有 glob 形态不进 argv 解析
	} {
		if _, _, ok := ParseArgvRuleText(bad); ok {
			t.Fatalf("非法形态应拒绝: %q", bad)
		}
	}
}

// 验收①：git status 放行；git status; rm -rf /tmp/x 仍要审批。
func TestArgvRuleSegmentSafety(t *testing.T) {
	r := ArgvRule{First: "git", Rest: []string{"status"}}
	if !r.Matches("git status") {
		t.Fatalf("精确命中应匹配")
	}
	if !r.Matches("git status --porcelain") {
		t.Fatalf("前缀扩展应匹配")
	}
	if r.Matches("git status; rm -rf /tmp/x") {
		t.Fatalf("串接提权不得匹配——这是 argv 规则存在的意义")
	}
	if r.Matches("git statusx") {
		t.Fatalf("token 边界外不得匹配")
	}
	if r.Matches("git") {
		t.Fatalf("缺 rest token 不得匹配")
	}
	// 引号形态：git commit -m "fix bug" 的 token 剥引号。
	rc, ok := ArgvRuleFromCommand(`git commit -m "fix bug"`, 0)
	if !ok || rc.First != "git" || len(rc.Rest) != 3 || rc.Rest[2] != "fix bug" {
		t.Fatalf("引号剥除不符: %+v %v", rc, ok)
	}
}

// Policy 装配：allow 列表里的 argv 形态进 ArgvAllow；Decide 把 Ask 升级为
// Allow 仅限命中的命令；Deny 优先级仍最高。
func TestPolicyArgvAllowDecide(t *testing.T) {
	p := New("ask",
		[]string{`bash prefix ["git","status"]`},
		nil,
		[]string{"bash(*rm -rf*)"})
	args := func(cmd string) json.RawMessage { return json.RawMessage(`{"command":"` + cmd + `"}`) }
	if d := p.Decide("bash", false, args("git status")); d != Allow {
		t.Fatalf("git status 应放行, got %v", d)
	}
	if d := p.Decide("bash", false, args("git status --short")); d != Allow {
		t.Fatalf("前缀扩展应放行, got %v", d)
	}
	if d := p.Decide("bash", false, args("git status; rm -rf /tmp/x")); d != Deny {
		t.Fatalf("串接深删应被 deny 规则拦截, got %v", d)
	}
	// Deny 优先级不受 argv 放行影响；非 deny 的串接回到 Ask。
	if d := p.Decide("bash", false, args("git status; echo pwned")); d != Ask {
		t.Fatalf("串接提权应回到 Ask, got %v", d)
	}
}

// 验收②（Session 作用域）：gate 的 SessionArgv 命中放行；换新集（/new 语义）
// 后失效。Persistent 作用域走 Policy.ArgvAllow，不受会话轮换影响。
// 用非只读命令（git commit）——git status 走只读判定直接放行，到不了 Ask；
// DenyUnapproved=true 让「无规则」可区分（否则 headless 自治恒放行）。
func TestGateSessionArgvScope(t *testing.T) {
	g := NewGate(New("ask", nil, nil, nil), nil)
	g.DenyUnapproved = true
	g.SessionArgv = NewArgvRuleSet()
	g.SessionArgv.Add(ArgvRule{First: "git", Rest: []string{"commit", "-m", "wip"}})
	if ok, _, _ := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit -m wip"}`), false); !ok {
		t.Fatalf("会话规则命中应免审批放行")
	}
	// /new：换新集 → 失效。
	g.SessionArgv = NewArgvRuleSet()
	if ok, _, _ := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit -m wip"}`), false); ok {
		t.Fatalf("会话轮换后 Session 作用域应失效")
	}
	// Persistent：策略里的 argv 规则不受影响。
	g2 := NewGate(New("ask", []string{`bash prefix ["git","commit","-m","wip"]`}, nil, nil), nil)
	g2.DenyUnapproved = true
	if ok, _, _ := g2.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit -m wip"}`), false); !ok {
		t.Fatalf("Persistent 作用域应仍生效")
	}
}

// Clear 必须原地清空：Controller /new 保留同一个集合并让已注入 Gate 的引用
// 同步失效；仅替换 Controller 字段会留下旧 Gate 继续放行。
func TestArgvRuleSetClearInPlace(t *testing.T) {
	s := NewArgvRuleSet()
	s.Add(ArgvRule{First: "git", Rest: []string{"commit", "-m", "wip"}})
	g := NewGate(New("ask", nil, nil, nil), nil)
	g.DenyUnapproved = true
	g.SessionArgv = s
	if ok, _, _ := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit -m wip"}`), false); !ok {
		t.Fatal("清空前规则应生效")
	}
	s.Clear()
	if s.Len() != 0 || g.SessionArgv.Len() != 0 {
		t.Fatalf("Clear 应原地清空: set=%d gate=%d", s.Len(), g.SessionArgv.Len())
	}
	if ok, _, _ := g.Check(context.Background(), "bash", json.RawMessage(`{"command":"git commit -m wip"}`), false); ok {
		t.Fatal("Clear 后旧 Gate 不得继续放行")
	}
}
