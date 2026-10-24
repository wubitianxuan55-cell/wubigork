package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
	_ "github.com/gaea/gaea/internal/gaea/tool/builtin" // 注册 builtin 工具
)

// readinessScriptedRunner 组装一个带 todo_write/complete_step/bash 的 Run 测试台。
func readinessScriptedRunner(t *testing.T, prov *scriptedProvider) *AgentRunner {
	t.Helper()
	isolateProgressCwd(t)
	todoWrite, ok := tool.LookupBuiltin("todo_write")
	if !ok {
		t.Fatal("todo_write builtin not registered")
	}
	completeStep, ok := tool.LookupBuiltin("complete_step")
	if !ok {
		t.Fatal("complete_step builtin not registered")
	}
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "bash", readOnly: false})
	reg.Add(todoWrite)
	reg.Add(completeStep)
	return New(prov, reg, NewSession(""), Options{}, event.Discard)
}

func todoWriteCall(id string, items string) provider.Chunk {
	return toolCallChunk(id, "todo_write", `{"todos":[`+items+`]}`)
}

// TestFinalReadinessGateBlocksFakeCompletion 钉死闸激活（v4.524 前传 nil 恒
// 放行的死闸）：回合内把 todo 从 pending 标成 completed 却从未跑
// complete_step，最终回答必须被拒；连续两轮同 reason=没有推进，停滞早停，
// 并挂 A4 交接块（verify/readiness nudge 是合成轮，回扫必须越过它们）。
func TestFinalReadinessGateBlocksFakeCompletion(t *testing.T) {
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{todoWriteCall("w1", `{"content":"A","status":"pending"}`), {Type: provider.ChunkDone}},
		{todoWriteCall("w2", `{"content":"A","status":"completed"}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	a := readinessScriptedRunner(t, prov)
	_, err := a.Run(context.Background(), "run the task")
	if err == nil {
		t.Fatal("Run must fail when final-answer readiness is never satisfied")
	}
	if !strings.Contains(err.Error(), "final-answer readiness failed 2 times") {
		t.Fatalf("err = %q, want stall early-stop at 2 blocks", err)
	}
	if !strings.Contains(err.Error(), "complete_step missing for: A") {
		t.Fatalf("err = %q, want the missing todo named", err)
	}
	if a.pendingRecovery == nil {
		t.Fatal("readiness-refused turn must leave an A4 recovery handover (capture now spans past [system] nudges)")
	}
}

// TestFinalReadinessGatePassesAfterCompleteStep 恢复路径：被拒一次后模型补跑
// complete_step（引用同回合 bash 收据），下一轮最终回答放行。
func TestFinalReadinessGatePassesAfterCompleteStep(t *testing.T) {
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{todoWriteCall("w1", `{"content":"A","status":"pending"}`), {Type: provider.ChunkDone}},
		{todoWriteCall("w2", `{"content":"A","status":"completed"}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
		{
			toolCallChunk("b1", "bash", `{"command":"go test ./..."}`),
			toolCallChunk("c1", "complete_step", `{
				"step":"A",
				"result":"A is done",
				"evidence":[{"kind":"verification","summary":"go test ./... passed","command":"go test ./..."}]
			}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	a := readinessScriptedRunner(t, prov)
	if _, err := a.Run(context.Background(), "run the task"); err != nil {
		t.Fatalf("Run after complete_step sign-off: %v", err)
	}
}

// TestIsSyntheticUserTurnSystemNudges 钉 [system] 门 nudge 前缀入白名单——
// A4 回扫与 recall_archived 靠它区分真实用户轮。
func TestIsSyntheticUserTurnSystemNudges(t *testing.T) {
	synthetic := []string{
		taskGateNudgeHeader + " details",
		stopGateOrchestrateVerifyNudge,
		goalGateNudge,
		"[system] Goal not yet met: x",
		repeatedStepNudge,
		outputLengthNudge,
		recallReminderNudge,
		schemaRetryNudge,
		repeatSuccessAdvisory,
		MidTurnSteerPrefix + "\nhello",
		"Host final-answer readiness check failed. fix it",
		"The previous assistant response was interrupted during streaming. Continue.",
	}
	for _, s := range synthetic {
		if !isSyntheticUserTurn(s) {
			t.Errorf("isSyntheticUserTurn(%q) = false, want true", truncateForTest(s))
		}
	}
	real := []string{"请帮我修这个 bug", "The system prompt says hello", "system: restart the service"}
	for _, s := range real {
		if isSyntheticUserTurn(s) {
			t.Errorf("isSyntheticUserTurn(%q) = true, want false", s)
		}
	}
}

func truncateForTest(s string) string {
	if len(s) > 40 {
		return s[:40] + "…"
	}
	return s
}
