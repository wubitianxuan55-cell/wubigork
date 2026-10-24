package agent

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── A4 中断交接块（Reasonix 2.x interrupted_recovery 蒸馏）测试 ──────────

func interruptedSession() *Session {
	s := NewSession("system")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "帮我修一下构建"})
	// 已完成的写入对
	s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "w1", Name: "edit_file", Arguments: `{"path":"C:\\ws\\a.go","content":"x"}`},
	}})
	s.Add(provider.Message{Role: provider.RoleTool, ToolCallID: "w1", Name: "edit_file", Content: "written"})
	// 被打断的调用（persistInterruptedCalls 信封文案）
	s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "b1", Name: "bash", Arguments: `{"command":"go test ./..."}`},
	}})
	s.Add(provider.Message{Role: provider.RoleTool, ToolCallID: "b1", Name: "bash",
		Content: `{"code":"exec_error","error":"stream interrupted: this call was received but not executed (stream aborted before dispatch); re-issue it if still needed"}`})
	// 部分正文
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "我已经改好了 a.go，接下来…"})
	return s
}

func TestCaptureInterruptedRecovery(t *testing.T) {
	a := testAgentC(0, 0.8, 5)
	a.session = interruptedSession()
	a.captureInterruptedRecovery()
	if a.pendingRecovery == nil {
		t.Fatal("有工具对的终局应产出交接记录")
	}
	if len(a.pendingRecovery.CompletedTools) != 1 || a.pendingRecovery.CompletedTools[0].Name != "edit_file" {
		t.Fatalf("edit_file 应在已完成清单: %+v", a.pendingRecovery.CompletedTools)
	}
	if len(a.pendingRecovery.CompletedTools[0].Files) != 1 || !strings.Contains(a.pendingRecovery.CompletedTools[0].Files[0], "a.go") {
		t.Fatalf("已完成写入应带文件: %+v", a.pendingRecovery.CompletedTools[0])
	}
	if len(a.pendingRecovery.InterruptedTools) != 1 || a.pendingRecovery.InterruptedTools[0] != "bash" {
		t.Fatalf("被打断调用应入清单: %+v", a.pendingRecovery.InterruptedTools)
	}
}

func TestCaptureSkipsSyntheticUserTurns(t *testing.T) {
	a := testAgentC(0, 0.8, 5)
	s := interruptedSession()
	// 流恢复合成消息之后再有一对工具——回扫必须越过合成消息
	s.Add(provider.Message{Role: provider.RoleUser, Content: streamRecoveryMessage(true)})
	s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "w2", Name: "write_file", Arguments: `{"path":"C:\\ws\\b.go"}`},
	}})
	s.Add(provider.Message{Role: provider.RoleTool, ToolCallID: "w2", Name: "write_file", Content: "written"})
	a.session = s
	a.captureInterruptedRecovery()
	if a.pendingRecovery == nil {
		t.Fatal("应产出记录")
	}
	names := map[string]bool{}
	for _, c := range a.pendingRecovery.CompletedTools {
		names[c.Name] = true
	}
	if !names["edit_file"] || !names["write_file"] {
		t.Fatalf("回扫应覆盖整轮（含合成消息之前的工作）: %v", names)
	}
}

func TestCaptureNoToolPairsNoRecord(t *testing.T) {
	a := testAgentC(0, 0.8, 5)
	s := NewSession("system")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hello"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "部分回答"})
	a.session = s
	a.captureInterruptedRecovery()
	if a.pendingRecovery != nil {
		t.Fatal("纯文本轮中断不应产出记录")
	}
}

func TestWithInterruptedRecoveryOneShot(t *testing.T) {
	a := testAgentC(0, 0.8, 5)
	a.session = interruptedSession()
	a.captureInterruptedRecovery()

	wrapped := a.withInterruptedRecovery("现在继续")
	if !strings.Contains(wrapped, "<"+interruptedRecoveryTag+">") {
		t.Fatalf("下一条用户消息应前置交接块: %q", wrapped)
	}
	if !strings.Contains(wrapped, "现在继续") || !strings.Contains(wrapped, "takes precedence") {
		t.Fatal("块应含用户输入与优先级条款")
	}
	if a.withInterruptedRecovery("再一条") != "再一条" {
		t.Fatal("记录应一次性消费")
	}
}

func TestRecoveryBlockBounded(t *testing.T) {
	r := &turnRecovery{}
	for i := 0; i < 30; i++ {
		files := make([]string, 0, 12)
		for j := 0; j < 12; j++ {
			files = append(files, strings.Repeat("f", 300))
		}
		r.CompletedTools = append(r.CompletedTools, recoveryTool{Name: "tool", Files: files})
	}
	for i := 0; i < 30; i++ {
		r.InterruptedTools = append(r.InterruptedTools, "toolx")
	}
	block := pendingRecoveryBlock(r)
	if strings.Count(block, "- tool") != maxRecoveryTools {
		t.Fatalf("已完成工具应封顶 %d: %d", maxRecoveryTools, strings.Count(block, "- tool"))
	}
	if !strings.Contains(block, "additional completed tool pair(s) omitted") ||
		!strings.Contains(block, "additional call(s) omitted") {
		t.Fatal("省略应如实计数")
	}
	if strings.Contains(block, strings.Repeat("f", 300)) {
		t.Fatal("文件值应截 240 rune")
	}
}
