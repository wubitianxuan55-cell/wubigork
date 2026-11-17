package control

// T7 端到端验收：Compose 的 session-facts 注入走章节化差分——
// 不变零注入 / 变更替换通知 / 清空移除通知 / 开关关=现状逐字节。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/memory"
)

func mkFact(name, desc string) memory.Memory {
	return memory.Memory{Name: name, Title: name, Description: desc}
}

// 开关开：首回合直出；同列表零注入；变更替换通知；清空移除通知。
func TestComposeSessionFactsSectionDiff(t *testing.T) {
	c := New(Options{Sink: newAttentionRecorder(), SectionDiff: true})
	c.sessionFacts = []memory.Memory{mkFact("pref-style", "简洁回复"), mkFact("proj-lang", "Go 项目")}

	out1 := c.Compose("用户消息")
	if !strings.Contains(out1, "<session-facts>") || !strings.Contains(out1, "简洁回复") {
		t.Fatalf("首回合应直出清单: %q", out1)
	}
	// 同列表：零注入。
	out2 := c.Compose("用户消息2")
	if strings.Contains(out2, "<session-facts>") || strings.Contains(out2, "简洁回复") {
		t.Fatalf("列表不变应零注入: %q", out2)
	}
	// 变更一条：替换通知 + 完整新清单。
	c.sessionFacts = append(c.sessionFacts, mkFact("new-fact", "新事实"))
	out3 := c.Compose("用户消息3")
	if !strings.Contains(out3, "替换此前给出的同类指引") || !strings.Contains(out3, "新事实") {
		t.Fatalf("变更应替换通知+新清单: %q", out3)
	}
	// 清空（promote/清除语义）：恰好一条移除通知。
	c.sessionFacts = nil
	out4 := c.Compose("用户消息4")
	if !strings.Contains(out4, "已失效") {
		t.Fatalf("清空应移除通知: %q", out4)
	}
	// 持续为空：不再发言。
	out5 := c.Compose("用户消息5")
	if strings.Contains(out5, "已失效") || strings.Contains(out5, "session-facts") {
		t.Fatalf("持续为空不应发言: %q", out5)
	}
}

// 开关关（默认）：走原全量注入，每回合都在（现状逐字节）。
func TestComposeSessionFactsDisabledWholesale(t *testing.T) {
	c := New(Options{Sink: newAttentionRecorder()})
	c.sessionFacts = []memory.Memory{mkFact("pref-style", "简洁回复")}

	out1 := c.Compose("第一条")
	out2 := c.Compose("第二条")
	if !strings.Contains(out1, "<session-facts>") || !strings.Contains(out2, "<session-facts>") {
		t.Fatalf("开关关应每回合全量注入: %q / %q", out1, out2)
	}
	// 与旧实现逐字节一致：前缀 + 正文形态。
	want := "<session-facts>\nYou saved these temporary facts this session (call promote_session_facts to make them permanent):\n- [pref-style] pref-style: 简洁回复\n</session-facts>\n\n"
	if !strings.HasPrefix(out1, want) {
		t.Fatalf("关态输出形态漂移:\n got %q\nwant prefix %q", out1, want)
	}
}
