package agent

// complete_block_test.go — pan_1795306273871 首刀 A 回归钉：完成块恒附
// （unreported 不再静默）、blocked 结构化阻塞展开、artifacts 宿主核验
// （有产出回执=verified/无=unverified，绝不升级）、干净面保持 compact 一行。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func sessWithMessages(msgs ...provider.Message) *Session {
	s := NewSession("sys")
	for _, m := range msgs {
		s.Add(m)
	}
	return s
}

func asstToolCall(name, args string) provider.Message {
	return provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "c" + name, Name: name, Arguments: args}}}
}

func cbToolResult(callID, content string) provider.Message {
	return provider.Message{Role: provider.RoleTool, ToolCallID: callID, Content: content}
}

func TestCompletionBlockUnreported(t *testing.T) {
	sess := sessWithMessages(asstToolCall("bash", `{"command":"echo hi"}`), cbToolResult("cbash", "hi"))
	out := childCompletionBlock(sess)
	if !strings.Contains(out, "unreported") {
		t.Fatalf("无报告应显 unreported（不再静默）：%q", out)
	}
}

func TestCompletionBlockBlockedStructured(t *testing.T) {
	report := `{"status":"blocked","summary":"卡在登录态","blockers":[{"reason":"平台需扫码登录","needed_from_parent":"用户扫码后告知"}],"unresolved":["后续重跑第3章"]}`
	sess := sessWithMessages(asstToolCall("complete_subtask", report), cbToolResult("ccomplete_subtask", "accepted"))
	out := childCompletionBlock(sess)
	for _, want := range []string{"status=blocked", "blockers=1", "平台需扫码登录", "用户扫码后告知", "unresolved=1", "后续重跑第3章"} {
		if !strings.Contains(out, want) {
			t.Errorf("blocked 展开缺 %q：%s", want, out)
		}
	}
}

func TestCompletionBlockArtifactsVerification(t *testing.T) {
	report := `{"status":"complete","summary":"产出两件","criteria":[],"artifacts":[{"path":"交付/整册.docx"},{"path":"交付/另一件.xlsx"}]}`
	// office_generate 成功回执声明 output_path=交付/整册.docx → verified；
	// 另一件无任何回执 → unverified（缺证据不升级）。
	sess := sessWithMessages(
		asstToolCall("office_generate", `{"markdown":"x","format":"docx","output_path":"交付/整册.docx"}`),
		cbToolResult("coffice_generate", `{"ok":true}`),
		asstToolCall("complete_subtask", report),
		cbToolResult("ccomplete_subtask", "accepted"),
	)
	out := childCompletionBlock(sess)
	if !strings.Contains(out, "交付/整册.docx [verified]") {
		t.Errorf("有产出回执应 verified：%s", out)
	}
	if !strings.Contains(out, "交付/另一件.xlsx [unverified]") {
		t.Errorf("缺证据应 unverified：%s", out)
	}
	if !strings.Contains(out, "unverified 1") {
		t.Errorf("compact 行应计 unverified=1：%s", out)
	}
}

func TestCompletionBlockCleanStaysCompact(t *testing.T) {
	report := `{"status":"complete","summary":"两判据全 backed","acceptance_criteria":[{"id":"c1","status":"satisfied","evidence":[{"kind":"diff","paths":["a.md"]}]},{"id":"c2","status":"satisfied","evidence":[{"kind":"files","paths":["a.md"]}]}]}`
	sess := sessWithMessages(
		asstToolCall("write_file", `{"path":"a.md","content":"x"}`),
		cbToolResult("cwrite_file", "ok"),
		asstToolCall("complete_subtask", report),
		cbToolResult("ccomplete_subtask", "accepted"),
	)
	out := childCompletionBlock(sess)
	if !strings.Contains(out, "status=complete criteria=2/2 backed artifacts=0") {
		t.Fatalf("干净面 compact 行：%s", out)
	}
	if strings.Contains(out, "\n  ") {
		t.Fatalf("干净面不得展开 full（并行多子的上下文税）：%s", out)
	}
}

// 嵌套中间层双向 send_message（action-4）：带 agent_id 走向下版、缺省走向上版。
func TestDualSendMessageDirection(t *testing.T) {
	up := &sendMessageTool{ref: "sa_mid1"}
	down := NewSendMessageTool(NewSubagentStore(t.TempDir()), nil)
	d := &dualSendMessageTool{up: up, down: down}
	ctx := context.Background()
	// 缺 agent_id → 向上（本测试无 live parent → 向上版的诚实拒绝文案）。
	_, upErr := d.Execute(ctx, json.RawMessage(`{"message":"hi"}`))
	if upErr == nil || !strings.Contains(upErr.Error(), "live parent") {
		t.Fatalf("缺 agent_id 应走向上版（live parent 拒绝）： %v", upErr)
	}
	// 带 agent_id → 向下版（store=nil 的子树守卫面前如实出错，与向上版不同文案）。
	_, downErr := d.Execute(ctx, json.RawMessage(`{"message":"hi","agent_id":"sa_kid1"}`))
	if downErr == nil {
		t.Fatal("带 agent_id 应走向下版")
	}
	if strings.Contains(downErr.Error(), "live parent") {
		t.Fatalf("带 agent_id 不得落回向上版：%v", downErr)
	}
}
