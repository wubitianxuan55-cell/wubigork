package agent

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func TestShouldMidTurnSteer_AllBlockedIsNotFailure(t *testing.T) {
	a := New(nil, nil, NewSession(""), Options{}, nil)
	calls := []provider.ToolCall{
		{Name: "write_file", Arguments: `{"path":"/x"}`},
		{Name: "edit_file", Arguments: `{"old":"a","new":"b"}`},
	}
	// All "blocked:" results — permission denies, completely normal.
	results := []string{
		"blocked: write_file denied by permission",
		"blocked: edit_file denied by permission",
	}
	if a.shouldMidTurnSteer(calls, results) {
		t.Error("all-blocked batch should NOT trigger steer — it is normal with permission gating")
	}
}

func TestShouldMidTurnSteer_RealFailuresTrigger(t *testing.T) {
	a := New(nil, nil, NewSession(""), Options{}, nil)
	calls := []provider.ToolCall{
		{Name: "read_file", Arguments: `{"path":"/x"}`},
		{Name: "read_file", Arguments: `{"path":"/y"}`},
	}
	// Real errors should trigger.
	results := []string{
		"error: no such file",
		"error: permission denied",
	}
	if !a.shouldMidTurnSteer(calls, results) {
		t.Error("real failures should trigger steer")
	}
}

func TestShouldMidTurnSteer_MixedBlockedAndFailedTriggers(t *testing.T) {
	a := New(nil, nil, NewSession(""), Options{}, nil)
	calls := []provider.ToolCall{
		{Name: "write_file", Arguments: `{"path":"/x"}`},
		{Name: "read_file", Arguments: `{"path":"/y"}`},
		{Name: "read_file", Arguments: `{"path":"/z"}`},
	}
	// One blocked, two real failures — non-blocked failures >= 2 should trigger.
	results := []string{
		"blocked: writer denied by permission",
		"error: no such file",
		"error: permission denied",
	}
	if !a.shouldMidTurnSteer(calls, results) {
		t.Error("mixed blocked+real failures (2 real) should trigger steer")
	}
}

func TestShouldMidTurnSteer_SingleFailureNotTriggered(t *testing.T) {
	a := New(nil, nil, NewSession(""), Options{}, nil)
	calls := []provider.ToolCall{
		{Name: "read_file", Arguments: `{"path":"/x"}`},
		{Name: "read_file", Arguments: `{"path":"/y"}`},
	}
	// One failure, one success — not enough for steer.
	results := []string{
		"error: no such file",
		"file content here",
	}
	if a.shouldMidTurnSteer(calls, results) {
		t.Error("single failure should NOT trigger steer")
	}
}

// B2 失败即值（2026-11-10）：普通工具失败自 V8.9 起就是信封 JSON，旧前缀
// 口径数不到=steer 从未见过真实工具失败（盲计根修的钉死测试）。
func TestShouldMidTurnSteer_EnvelopeFailuresTrigger(t *testing.T) {
	a := New(nil, nil, NewSession(""), Options{}, nil)
	calls := []provider.ToolCall{
		{Name: "bash", Arguments: `{"command":"x"}`},
		{Name: "edit_file", Arguments: `{"path":"/y"}`},
	}
	// 信封形态失败（executeOne 收口的真实输出形态）。
	results := []string{
		`{"ok":false,"success":false,"code":"exec_error","error":"command exited: exit status 1","retryable":true,"effect_state":"unknown","data":{"tool":"bash","detail":""}}`,
		`{"ok":false,"success":false,"code":"conflict","error":"REVISION_CONFLICT: b.txt 自读取后已被修改","retryable":true,"effect_state":"none","data":{"tool":"edit_file"}}`,
	}
	if !a.shouldMidTurnSteer(calls, results) {
		t.Error("信封失败应触发 steer（旧口径盲计根修）")
	}
	// 单个信封失败不成批——不触发。
	single := []string{results[0], `{"ok":true,"success":true,"code":"ok","data":{"tool":"read_file","result":"..."}}`}
	if a.shouldMidTurnSteer(calls, single) {
		t.Error("单失败+成功不应触发")
	}
	// 信封 code=blocked 归被拦（计划模式正常路径），全拦不触发。
	blocked := []string{
		`{"ok":false,"success":false,"code":"blocked","error":"plan mode"}`,
		`{"ok":false,"success":false,"code":"blocked","error":"plan mode"}`,
	}
	if a.shouldMidTurnSteer(calls, blocked) {
		t.Error("信封 blocked 应归被拦不触发 steer")
	}
}

func TestToolOutputClassAndCompact(t *testing.T) {
	// 旧前缀口径全保留（防回退）。
	for _, s := range []string{
		"error: x", "blocked: y", "precheck blocked: z", "suppressed: w", "tool panic: p",
	} {
		if !hasToolErrorPrefix(s) {
			t.Errorf("旧前缀 %q 应判失败", s)
		}
	}
	failed, blocked := toolOutputClass("blocked: plan mode")
	if failed || !blocked {
		t.Errorf("blocked 应归被拦: %v %v", failed, blocked)
	}
	// 信封失败/成功/非信封 JSON 三态。
	if f, b := toolOutputClass(`{"ok":false,"code":"exec_error","error":"boom"}`); !f || b {
		t.Errorf("信封失败应 failed: %v %v", f, b)
	}
	if f, b := toolOutputClass(`{"ok":true,"code":"ok"}`); f || b {
		t.Errorf("成功信封不应失败: %v %v", f, b)
	}
	if f, b := toolOutputClass(`{"rows":3}`); f || b {
		t.Errorf("普通 JSON 不误判: %v %v", f, b)
	}
	// 压缩：信封→一行 error: 前缀（turnToolErrors 形态稳定）。
	got := compactToolError(`{"ok":false,"code":"conflict","error":"REVISION_CONFLICT: x","data":{"detail":"很长的输出……"}}`)
	if got != "error: REVISION_CONFLICT: x" {
		t.Fatalf("信封压缩形态错: %q", got)
	}
	if got := compactToolError("error: raw"); got != "error: raw" {
		t.Fatalf("裸前缀应原样: %q", got)
	}
}
