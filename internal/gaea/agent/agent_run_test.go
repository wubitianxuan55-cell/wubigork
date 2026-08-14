package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ─── TurnResult 运行链路语义（T6-2.2）────────────────────────────
//
// 这些测试锁定 agent_run.go 的结果语义：
//   - Success=true 仅当整轮未阻断、无错误、无 suppressed；
//   - 流错误（不可恢复 / 恢复耗尽）不再丢弃已收部分文本；
//   - step-- 下限 0，step=0/负数的中断不触发额外 grace 轮。

// TestTurnResultNormalRoundSucceeds 正常轮（模型直接给最终答案）→
// Success=true、Errors 为空、Summary 为最后助手消息。
func TestTurnResultNormalRoundSucceeds(t *testing.T) {
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	a := New(prov, tool.NewRegistry(), NewSession(""), Options{DisableVerify: true}, event.Discard)
	res, err := a.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res == nil {
		t.Fatal("Run returned nil TurnResult")
	}
	if !res.Success {
		t.Errorf("normal round: Success = false, want true (Errors=%v)", res.Errors)
	}
	if len(res.Errors) != 0 {
		t.Errorf("normal round: Errors = %v, want empty", res.Errors)
	}
	if res.Summary != "done" {
		t.Errorf("Summary = %q, want %q", res.Summary, "done")
	}
}

// TestTurnResultAllBlockedFailsSuccess 整轮全部被 gate 阻断 →
// Success=false，且每条 blocked 结果都进入 Errors。
func TestTurnResultAllBlockedFailsSuccess(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "write_file", readOnly: false})
	reg.Add(fakeTool{name: "edit_file", readOnly: false})
	g := &stubGate{deny: map[string]bool{"write_file": true, "edit_file": true}}
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "write_file", `{"path":"/a.txt"}`),
			toolCallChunk("c2", "edit_file", `{"path":"/b.txt"}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "final"}, {Type: provider.ChunkDone}},
	}}
	a := New(prov, reg, NewSession(""), Options{Gate: g, DisableVerify: true}, event.Discard)
	res, err := a.Run(context.Background(), "write stuff")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Success {
		t.Error("fully-blocked round: Success = true, want false")
	}
	if len(res.Errors) != 2 {
		t.Errorf("fully-blocked round: len(Errors) = %d (%v), want 2", len(res.Errors), res.Errors)
	}
	for _, e := range res.Errors {
		if !strings.HasPrefix(e, "blocked:") {
			t.Errorf("Errors entry %q does not start with %q", e, "blocked:")
		}
	}
}

// TestTurnResultPartiallyBlockedFailsSuccess 部分调用被阻断（其余成功）→
// Success=false，Errors 只含被阻断的那一条。
func TestTurnResultPartiallyBlockedFailsSuccess(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "write_file", readOnly: false})
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	g := &stubGate{deny: map[string]bool{"write_file": true}}
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "write_file", `{"path":"/a.txt"}`),
			toolCallChunk("c2", "read_file", `{"path":"/x.go"}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "final"}, {Type: provider.ChunkDone}},
	}}
	a := New(prov, reg, NewSession(""), Options{Gate: g, DisableVerify: true}, event.Discard)
	res, err := a.Run(context.Background(), "edit it")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Success {
		t.Error("partially-blocked round: Success = true, want false")
	}
	if len(res.Errors) != 1 {
		t.Fatalf("partially-blocked round: len(Errors) = %d (%v), want 1", len(res.Errors), res.Errors)
	}
	if !strings.HasPrefix(res.Errors[0], "blocked:") {
		t.Errorf("Errors[0] = %q, want a %q result", res.Errors[0], "blocked:")
	}
}

// TestTurnResultSuppressedCounts 参数风暴抑制的重复调用计入 Errors →
// Success=false（整轮含 suppressed 即非成功）。
func TestTurnResultSuppressedCounts(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "read_file", `{"path":"/x"}`),
			toolCallChunk("c2", "read_file", `{"path":"/x"}`),
			toolCallChunk("c3", "read_file", `{"path":"/x"}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "final"}, {Type: provider.ChunkDone}},
	}}
	a := New(prov, reg, NewSession(""), Options{
		ParamStorm:    &ParamStormOptions{Threshold: 3},
		DisableVerify: true,
	}, event.Discard)
	res, err := a.Run(context.Background(), "read x repeatedly")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Success {
		t.Error("round with a suppressed call: Success = true, want false")
	}
	if len(res.Errors) != 1 {
		t.Fatalf("round with a suppressed call: len(Errors) = %d (%v), want 1", len(res.Errors), res.Errors)
	}
	if !strings.HasPrefix(res.Errors[0], "suppressed:") {
		t.Errorf("Errors[0] = %q, want a %q result", res.Errors[0], "suppressed:")
	}
}

// TestTurnResultErrorsCappedAtFive 一轮超过 5 个失败结果时，Errors 截断到 5。
func TestTurnResultErrorsCappedAtFive(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "write_file", readOnly: false})
	g := &stubGate{deny: map[string]bool{"write_file": true}}
	chunks := []provider.Chunk{}
	for i := 0; i < 6; i++ {
		chunks = append(chunks, toolCallChunk(fmt.Sprintf("c%d", i), "write_file", `{"path":"/same.txt"}`))
	}
	chunks = append(chunks, provider.Chunk{Type: provider.ChunkDone})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		chunks,
		{{Type: provider.ChunkText, Text: "final"}, {Type: provider.ChunkDone}},
	}}
	a := New(prov, reg, NewSession(""), Options{Gate: g, DisableVerify: true}, event.Discard)
	res, err := a.Run(context.Background(), "write six times")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Success {
		t.Error("six denied calls: Success = true, want false")
	}
	if len(res.Errors) != 5 {
		t.Errorf("len(Errors) = %d, want the 5-entry cap (%v)", len(res.Errors), res.Errors)
	}
}

// TestBuildTurnResultSuccessSemantics 直接锁定 buildTurnResult 的 Success 计算：
// 只要 Errors 含 error/blocked/suppressed/tool-panic 中任意一条，Success 即为 false。
func TestBuildTurnResultSuccessSemantics(t *testing.T) {
	cases := []struct {
		name    string
		errors  []string
		success bool
	}{
		{"no issues", nil, true},
		{"error counts", []string{"error: boom"}, false},
		{"blocked counts", []string{"blocked: denied by policy"}, false},
		{"precheck blocked counts", []string{"precheck blocked: old_string not found in a.go"}, false},
		{"suppressed counts", []string{"suppressed: duplicate tool call (param storm breaker)"}, false},
		{"tool panic counts", []string{"tool panic: nil deref"}, false},
		{"mixed counts", []string{"blocked: x", "error: y"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := buildTurnResult(nil, nil, c.errors, "summary")
			if res.Success != c.success {
				t.Errorf("Success = %v, want %v (Errors=%v)", res.Success, c.success, c.errors)
			}
		})
	}
}

// TestStreamErrorPreservesPartialText 流中断三次恢复耗尽后走终止错误路径：
// 每次中断的已收部分文本都必须留在会话里，且结果 Summary 为最后一段。
func TestStreamErrorPreservesPartialText(t *testing.T) {
	interrupted := func(text string) []provider.Chunk {
		return []provider.Chunk{
			{Type: provider.ChunkText, Text: text},
			{Type: provider.ChunkError, Err: &provider.StreamInterruptedError{Err: errors.New("stream reset")}},
		}
	}
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		interrupted("partial-1"),
		interrupted("partial-2"),
		interrupted("partial-3"),
		interrupted("partial-4"),
	}}
	a := New(prov, tool.NewRegistry(), NewSession(""), Options{DisableVerify: true}, event.Discard)
	res, err := a.Run(context.Background(), "go")
	if err == nil {
		t.Fatal("expected a terminal stream error after recoveries are exhausted")
	}
	if res == nil {
		t.Fatal("Run returned nil TurnResult on stream error")
	}
	if !strings.Contains(err.Error(), "stream reset") {
		t.Errorf("err = %v, want it to carry the stream error", err)
	}
	var joined strings.Builder
	for _, m := range a.session.Messages {
		joined.WriteString(m.Content)
		joined.WriteString("\n")
	}
	for _, want := range []string{"partial-1", "partial-2", "partial-3", "partial-4"} {
		if !strings.Contains(joined.String(), want) {
			t.Errorf("session lost partial text %q; got:\n%s", want, joined.String())
		}
	}
	if res.Summary != "partial-4" {
		t.Errorf("Summary = %q, want the last partial chunk %q", res.Summary, "partial-4")
	}
	if prov.call != 4 {
		t.Errorf("provider calls = %d, want 4 (3 recoveries + 1 terminal)", prov.call)
	}
}

// TestStepZeroInterruptNoExtraGraceRound maxSteps=1 时首轮即被流中断：
// step=0 的中断不再让 step 变负去蹭额外的恢复轮/grace 轮——只发生一次模型调用，
// 部分文本仍保留在会话中，turn 以 max_steps 错误结束。
func TestStepZeroInterruptNoExtraGraceRound(t *testing.T) {
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			{Type: provider.ChunkText, Text: "partial-0"},
			{Type: provider.ChunkError, Err: &provider.StreamInterruptedError{Err: errors.New("stream reset")}},
		},
	}}
	a := New(prov, tool.NewRegistry(), NewSession(""), Options{MaxSteps: 1, DisableVerify: true}, event.Discard)
	res, err := a.Run(context.Background(), "go")
	if err == nil {
		t.Fatal("expected a max_steps error when the only budgeted round is interrupted")
	}
	if !strings.Contains(err.Error(), "max_steps") {
		t.Errorf("err = %v, want the max_steps pause message", err)
	}
	if res == nil {
		t.Fatal("Run returned nil TurnResult")
	}
	if prov.call != 1 {
		t.Errorf("provider calls = %d, want 1 — a step=0 interruption must not grant an extra recovery+grace round", prov.call)
	}
	found := false
	for _, m := range a.session.Messages {
		if strings.Contains(m.Content, "partial-0") {
			found = true
		}
	}
	if !found {
		t.Error("partial text was not preserved in the session")
	}
}

// TestGraceRoundBoundaryAtStepZero maxSteps=1：step=0 调用工具 → 触发一次
// grace 轮收尾。总共恰好 2 次模型调用（1 工具轮 + 1 grace 轮），Success=true。
func TestGraceRoundBoundaryAtStepZero(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "read_file", `{"path":"/x"}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "final"}, {Type: provider.ChunkDone}},
	}}
	a := New(prov, reg, NewSession(""), Options{MaxSteps: 1, DisableVerify: true}, event.Discard)
	res, err := a.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Success {
		t.Errorf("grace-round turn with a successful tool call: Success = false, want true (%v)", res.Errors)
	}
	if prov.call != 2 {
		t.Errorf("provider calls = %d, want 2 (1 tool round + 1 grace round)", prov.call)
	}
}

// TestMaxStepsRecoveryNoExtraGraceRound maxSteps=2 时第 2 轮被流中断：
// 恢复重试仍占用同一轮（不额外多出 grace 轮），总共恰好 4 次模型调用。
func TestMaxStepsRecoveryNoExtraGraceRound(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "read_file", `{"path":"/x"}`),
			{Type: provider.ChunkDone},
		},
		{
			{Type: provider.ChunkText, Text: "partial-1"},
			{Type: provider.ChunkError, Err: &provider.StreamInterruptedError{Err: errors.New("stream reset")}},
		},
		{
			toolCallChunk("c2", "read_file", `{"path":"/y"}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "final"}, {Type: provider.ChunkDone}},
	}}
	a := New(prov, reg, NewSession(""), Options{MaxSteps: 2, DisableVerify: true}, event.Discard)
	res, err := a.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Success {
		t.Errorf("recovery round with successful calls: Success = false, want true (%v)", res.Errors)
	}
	// 工具@0、中断@1+恢复重试、工具@1-重试、grace@2 → 4 次调用；恢复不产生额外一轮。
	if prov.call != 4 {
		t.Errorf("provider calls = %d, want 4 (tool@0, interrupted@1 + retry, tool@1-retry, grace@2)", prov.call)
	}
}
