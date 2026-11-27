package agent

// interrupted_recovery_resume_test.go — 2026-11-27 AI 反馈 #15② 回归钉：中断交接记录
// 不再只活在内存里——会话换入时最后一轮没收尾就从持久化消息重建，带上任务原话；
// 已收尾的会话不重建；换会话清掉上一个会话的记录；消费时补读现行待办。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func finishedSession() *Session {
	s := NewSession("system")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "看看目录"})
	s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "l1", Name: "list_dir", Arguments: `{"path":"."}`},
	}})
	s.Add(provider.Message{Role: provider.RoleTool, ToolCallID: "l1", Name: "list_dir", Content: "a b"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "目录里有 a 和 b。"})
	return s
}

func TestRebuildInterruptedRecoveryAfterRestart(t *testing.T) {
	a := testAgentC(0, 0.8, 5)
	a.SetSession(interruptedSession()) // 恢复路径：进程重启后从磁盘换入
	if a.pendingRecovery == nil {
		t.Fatal("最后一轮没收尾，应重建交接记录")
	}
	r := a.pendingRecovery
	if !r.Resumed || r.Task != "帮我修一下构建" {
		t.Fatalf("重建的记录应带任务原话与来源标记：%+v", r)
	}
	if len(r.CompletedTools) != 1 || r.CompletedTools[0].Name != "edit_file" || len(r.InterruptedTools) != 1 || r.InterruptedTools[0] != "bash" {
		t.Fatalf("已完成 / 被打断的调用应与终局捕获一致：%+v", r)
	}
	got := a.withInterruptedRecovery("继续")
	if !strings.HasPrefix(got, "<"+interruptedRecoveryTag+">") || !strings.HasSuffix(got, "\n\n继续") {
		t.Fatalf("交接块应前置到本轮输入：\n%s", got)
	}
	for _, want := range []string{
		"task: 帮我修一下构建",
		"recovered_from: session_log",
		"not_recorded (the interrupted turn never used todo_write",
		"completed_tools:\n- edit_file",
		"interrupted_tools: bash",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("重建的交接块缺 %q：\n%s", want, got)
		}
	}
	// 一次性消费。
	if a.withInterruptedRecovery("再来") != "再来" {
		t.Fatal("交接块只应消费一次")
	}
}

// 进程被硬杀的形态：最后一条是工具结果（模型还没接话），或带工具调用却没等到结果。
func TestRebuildInterruptedRecoveryHardKill(t *testing.T) {
	for name, tail := range map[string][]provider.Message{
		"工具结果收尾": {{Role: provider.RoleTool, ToolCallID: "w1", Name: "edit_file", Content: "written"}},
		"调用没结果": {{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
			{ID: "w2", Name: "bash", Arguments: `{"command":"go test ./..."}`},
		}}},
	} {
		s := NewSession("system")
		s.Add(provider.Message{Role: provider.RoleUser, Content: "跑测试并修到绿"})
		s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
			{ID: "w1", Name: "edit_file", Arguments: `{"path":"C:\\ws\\a.go"}`},
		}})
		for _, m := range tail {
			s.Add(m)
		}
		a := testAgentC(0, 0.8, 5)
		a.SetSession(s)
		if a.pendingRecovery == nil || a.pendingRecovery.Task != "跑测试并修到绿" {
			t.Errorf("%s：应重建交接记录：%+v", name, a.pendingRecovery)
		}
	}
}

func TestRebuildSkipsFinishedTurns(t *testing.T) {
	a := testAgentC(0, 0.8, 5)
	a.SetSession(finishedSession())
	if a.pendingRecovery != nil {
		t.Fatalf("已收尾的轮不该重建：%+v", a.pendingRecovery)
	}
	// 轮尾挂着未消费的插话收据：轮本身已完成，不算没收尾。
	s := finishedSession()
	s.Add(provider.Message{Role: provider.RoleUser, Content: unappliedSteerPrefix + "The previous turn ended before this mid-task guidance reached the model. 顺便看看 b"})
	a.SetSession(s)
	if a.pendingRecovery != nil {
		t.Fatalf("插话收据不该把已完成的轮判成没收尾：%+v", a.pendingRecovery)
	}
	// 空会话、只有用户消息：没有工具对，没有记录。
	a.SetSession(NewSession("system"))
	onlyUser := NewSession("system")
	onlyUser.Add(provider.Message{Role: provider.RoleUser, Content: "在吗"})
	a.SetSession(onlyUser)
	if a.pendingRecovery != nil {
		t.Fatalf("没有工具对不该有记录：%+v", a.pendingRecovery)
	}
	// 上一个会话的内存记录随换入清掉，不串台。
	a.pendingRecovery = &turnRecovery{CompletedTools: []recoveryTool{{Name: "write_file"}}}
	a.SetSession(finishedSession())
	if a.pendingRecovery != nil {
		t.Fatal("换会话应清掉上一个会话的交接记录")
	}
}

// 重启后首轮：Run 里前置交接块排在「从日志重建待办」之前——消费时要先把清单
// 从日志重建出来，用过待办工具的轮才不会被写成未记录。
func TestRebuildTodosFromLogOnConsume(t *testing.T) {
	s := NewSession("system")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "整理资料"})
	s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "t1", Name: "todo_write", Arguments: `{"todos":[{"content":"整理资料","status":"in_progress"},{"content":"写回执","status":"pending"}]}`},
	}})
	s.Add(provider.Message{Role: provider.RoleTool, ToolCallID: "t1", Name: "todo_write", Content: "ok"})
	s.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "w9", Name: "bash", Arguments: `{"command":"ls"}`},
	}})
	a := testAgentC(0, 0.8, 5)
	a.SetSession(s)
	if a.pendingRecovery == nil {
		t.Fatal("应重建交接记录")
	}
	if len(a.CanonicalTodoState()) != 0 {
		t.Fatal("前置条件下待办清单尚未重建")
	}
	got := a.withInterruptedRecovery("继续")
	for _, want := range []string{"in_progress: 整理资料", "pending: 写回执"} {
		if !strings.Contains(got, want) {
			t.Errorf("待办应从日志重建进交接块，缺 %q：\n%s", want, got)
		}
	}
	if strings.Contains(got, "not_recorded") {
		t.Errorf("用过待办工具的轮不该写成未记录：\n%s", got)
	}
}

func TestTurnUnfinished(t *testing.T) {
	call := provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{
		{ID: "w1", Name: "bash", Arguments: `{}`},
	}}
	result := provider.Message{Role: provider.RoleTool, ToolCallID: "w1", Name: "bash", Content: "ok"}
	cases := []struct {
		name   string
		region []provider.Message
		want   bool
	}{
		{"空", nil, false},
		{"工具结果收尾", []provider.Message{call, result}, true},
		{"调用没结果", []provider.Message{call}, true},
		{"助手收尾", []provider.Message{call, result, {Role: provider.RoleAssistant, Content: "完成"}}, false},
		{"结果带中断标记", []provider.Message{call, {Role: provider.RoleTool, ToolCallID: "w1", Name: "bash", Content: "stream interrupted"}, {Role: provider.RoleAssistant, Content: "改了一半"}}, true},
		{"末尾是插话收据", []provider.Message{call, result, {Role: provider.RoleAssistant, Content: "完成"}, {Role: provider.RoleUser, Content: unappliedSteerPrefix + "x"}}, false},
		{"末尾是流中断续写提示", []provider.Message{{Role: provider.RoleAssistant, Content: "写了一半"}, {Role: provider.RoleUser, Content: "The previous assistant response was interrupted during streaming."}}, true},
	}
	for _, c := range cases {
		if got := turnUnfinished(c.region); got != c.want {
			t.Errorf("%s：turnUnfinished=%v，期望 %v", c.name, got, c.want)
		}
	}
}
