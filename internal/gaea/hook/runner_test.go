package hook

import (
	"context"
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/sandbox"
)

// --- Runner construction ---

func TestNewRunnerNil(t *testing.T) {
	var r *Runner
	if r.Enabled() {
		t.Error("nil Runner should not be enabled")
	}
	if r.Hooks() != nil {
		t.Error("nil Runner.Hooks() should be nil")
	}
}

func TestNewRunnerEmpty(t *testing.T) {
	r := NewRunner(nil, "/tmp", nil, nil)
	if r.Enabled() {
		t.Error("empty hooks Runner should not be enabled")
	}
}

func TestNewRunnerWithHooks(t *testing.T) {
	hooks := []ResolvedHook{
		{HookConfig: HookConfig{Command: "echo hi"}, Event: PreToolUse, Scope: ScopeGlobal},
	}
	r := NewRunner(hooks, "/tmp", nil, nil)
	if !r.Enabled() {
		t.Error("Runner with hooks should be enabled")
	}
	if len(r.Hooks()) != 1 {
		t.Errorf("Hooks() count = %d, want 1", len(r.Hooks()))
	}
}

// --- Runner.PreToolUse ---

func TestRunnerPreToolUseNoHooks(t *testing.T) {
	r := NewRunner(nil, "/tmp", nil, nil)
	block, msg := r.PreToolUse(context.Background(), "bash", nil)
	if block || msg != "" {
		t.Errorf("no hooks should pass: block=%v msg=%q", block, msg)
	}
}

func TestRunnerPreToolUsePass(t *testing.T) {
	hooks := []ResolvedHook{
		{HookConfig: HookConfig{Command: "allow"}, Event: PreToolUse},
	}
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		return SpawnResult{ExitCode: 0}
	}
	r := NewRunner(hooks, "/tmp", spawner, nil)
	block, msg := r.PreToolUse(context.Background(), "bash", nil)
	if block {
		t.Errorf("exit 0 should not block: msg=%q", msg)
	}
}

func TestRunnerPreToolUseBlock(t *testing.T) {
	hooks := []ResolvedHook{
		{HookConfig: HookConfig{Command: "deny"}, Event: PreToolUse},
	}
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		return SpawnResult{ExitCode: 2, Stderr: "blocked by policy"}
	}
	var notified string
	notify := func(msg string) { notified = msg }
	r := NewRunner(hooks, "/tmp", spawner, notify)
	block, msg := r.PreToolUse(context.Background(), "bash", nil)
	if !block {
		t.Error("exit 2 on PreToolUse should block")
	}
	if msg == "" {
		t.Error("block message should not be empty")
	}
	if notified == "" {
		t.Error("notify should have been called")
	}
}

// --- Runner.PostToolUse ---

func TestRunnerPostToolUseNoHooks(t *testing.T) {
	r := NewRunner(nil, "/tmp", nil, nil)
	// Should not panic.
	r.PostToolUse(context.Background(), "bash", nil, "ok")
}

func TestRunnerPostToolUseWarn(t *testing.T) {
	hooks := []ResolvedHook{
		{HookConfig: HookConfig{Command: "warn"}, Event: PostToolUse},
	}
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		return SpawnResult{ExitCode: 1, Stdout: "warning message"}
	}
	var notified string
	notify := func(msg string) { notified = msg }
	r := NewRunner(hooks, "/tmp", spawner, notify)
	r.PostToolUse(context.Background(), "bash", nil, "result")
	if notified == "" {
		t.Error("PostToolUse warn should notify")
	}
}

// --- Runner.PromptSubmit ---

func TestRunnerPromptSubmitBlock(t *testing.T) {
	hooks := []ResolvedHook{
		{HookConfig: HookConfig{Command: "gate"}, Event: UserPromptSubmit},
	}
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		return SpawnResult{ExitCode: 2, Stderr: "not allowed"}
	}
	r := NewRunner(hooks, "/tmp", spawner, nil)
	block, _ := r.PromptSubmit(context.Background(), "bad input", 1)
	if !block {
		t.Error("exit 2 on UserPromptSubmit should block")
	}
}

// --- Runner.Stop ---

func TestRunnerStopNoHooks(t *testing.T) {
	r := NewRunner(nil, "/tmp", nil, nil)
	// Should not panic.
	r.Stop(context.Background(), "last answer", 1)
}

func TestRunnerStopWithHooks(t *testing.T) {
	hooks := []ResolvedHook{
		{HookConfig: HookConfig{Command: "log"}, Event: Stop},
	}
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		return SpawnResult{ExitCode: 0}
	}
	r := NewRunner(hooks, "/tmp", spawner, nil)
	r.Stop(context.Background(), "done", 1)
}

// --- Runner.PostLLMCall ---

func TestRunnerHasPostLLMCall(t *testing.T) {
	with := NewRunner([]ResolvedHook{{HookConfig: HookConfig{Command: "x"}, Event: PostLLMCall}}, "/tmp", nil, nil)
	if !with.HasPostLLMCall() {
		t.Error("a configured PostLLMCall hook should report HasPostLLMCall")
	}
	without := NewRunner([]ResolvedHook{{HookConfig: HookConfig{Command: "x"}, Event: Stop}}, "/tmp", nil, nil)
	if without.HasPostLLMCall() {
		t.Error("only a Stop hook should not report HasPostLLMCall")
	}
	if (*Runner)(nil).HasPostLLMCall() {
		t.Error("nil runner should report no PostLLMCall hook")
	}
}

func TestRunnerPostLLMCallReplacesReasoning(t *testing.T) {
	hooks := []ResolvedHook{{HookConfig: HookConfig{Command: "translate"}, Event: PostLLMCall}}
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		return SpawnResult{ExitCode: 0, Stdout: "  译文  "}
	}
	r := NewRunner(hooks, "/tmp", spawner, nil)
	if got := r.PostLLMCall(context.Background(), "raw reasoning", 2); got != "译文" {
		t.Fatalf("PostLLMCall = %q, want trimmed hook stdout", got)
	}
}

func TestRunnerPostLLMCallKeepsOriginal(t *testing.T) {
	cases := []struct {
		name  string
		hooks []ResolvedHook
		spawn SpawnResult
	}{
		{"no PostLLMCall hook", []ResolvedHook{{HookConfig: HookConfig{Command: "x"}, Event: Stop}}, SpawnResult{ExitCode: 0, Stdout: "ignored"}},
		{"empty stdout", []ResolvedHook{{HookConfig: HookConfig{Command: "x"}, Event: PostLLMCall}}, SpawnResult{ExitCode: 0, Stdout: "   "}},
		{"non-zero exit", []ResolvedHook{{HookConfig: HookConfig{Command: "x"}, Event: PostLLMCall}}, SpawnResult{ExitCode: 1, Stdout: "should be ignored"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRunner(tc.hooks, "/tmp", func(context.Context, SpawnInput) SpawnResult { return tc.spawn }, nil)
			if got := r.PostLLMCall(context.Background(), "raw", 1); got != "raw" {
				t.Fatalf("PostLLMCall = %q, want original reasoning preserved", got)
			}
		})
	}
}

// --- GA4-06: hook sandbox wiring ---

// enforceSpec is the spec a production boot hands the Runner (same source as the
// bash tool's).
func enforceSpec() sandbox.Spec {
	return sandbox.Spec{Mode: "enforce", WriteRoots: []string{`C:\ws`}, Network: true}
}

// TestRunnerSandboxInjectedIntoEveryEvent is the GA4-06 red-when-reverted case:
// with enforce + a platform that can confine, every event path must deliver the
// spec to the spawner inside SpawnInput. Deleting the injection line in
// Runner.spawn makes this fail on all five events.
func TestRunnerSandboxInjectedIntoEveryEvent(t *testing.T) {
	want := enforceSpec()
	hooks := []ResolvedHook{
		{HookConfig: HookConfig{Command: "gate"}, Event: PreToolUse},
		{HookConfig: HookConfig{Command: "post"}, Event: PostToolUse},
		{HookConfig: HookConfig{Command: "prompt"}, Event: UserPromptSubmit},
		{HookConfig: HookConfig{Command: "stop"}, Event: Stop},
		{HookConfig: HookConfig{Command: "start"}, Event: SessionStart},
	}
	events := []string{}
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		events = append(events, in.Command)
		if in.Sandbox == nil {
			t.Errorf("hook %q: SpawnInput.Sandbox = nil, want the enforce spec", in.Command)
			return SpawnResult{ExitCode: 0}
		}
		if in.Sandbox.Mode != want.Mode || in.Sandbox.Network != want.Network {
			t.Errorf("hook %q: Sandbox = %+v, want Mode=%q Network=%v", in.Command, *in.Sandbox, want.Mode, want.Network)
		}
		if len(in.Sandbox.WriteRoots) != len(want.WriteRoots) {
			t.Fatalf("hook %q: WriteRoots = %v, want %v", in.Command, in.Sandbox.WriteRoots, want.WriteRoots)
		}
		for i := range want.WriteRoots {
			if in.Sandbox.WriteRoots[i] != want.WriteRoots[i] {
				t.Errorf("hook %q: WriteRoots[%d] = %q, want %q", in.Command, i, in.Sandbox.WriteRoots[i], want.WriteRoots[i])
			}
		}
		return SpawnResult{ExitCode: 0}
	}
	// Availability is injected: this test must not depend on a real WSL2/bwrap.
	r := NewRunner(hooks, "/tmp", spawner, nil).WithSandbox(want)
	r.sandboxAvailable = func() bool { return true }

	ctx := context.Background()
	r.PreToolUse(ctx, "bash", nil)
	r.PostToolUse(ctx, "bash", nil, "ok")
	r.PromptSubmit(ctx, "hi", 1)
	r.Stop(ctx, "done", 1)
	r.SessionStart(ctx)
	if len(events) != 5 {
		t.Fatalf("spawner saw %d events (%v), want 5", len(events), events)
	}
}

// TestRunnerSandboxNotInjectedWithoutEnforce is the backward-compatibility half:
// nil spec (no WithSandbox at all) or Mode != "enforce" must leave
// SpawnInput.Sandbox nil, i.e. the pre-GA4-06 behaviour for every existing
// caller/test is byte-for-byte preserved.
func TestRunnerSandboxNotInjectedWithoutEnforce(t *testing.T) {
	hooks := []ResolvedHook{{HookConfig: HookConfig{Command: "gate"}, Event: PreToolUse}}
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		if in.Sandbox != nil {
			t.Errorf("SpawnInput.Sandbox = %+v, want nil when not enforce", *in.Sandbox)
		}
		return SpawnResult{ExitCode: 0}
	}
	cases := []struct {
		name string
		spec *sandbox.Spec
	}{
		{"no WithSandbox (zero spec)", nil},
		{"mode off", &sandbox.Spec{Mode: "off", WriteRoots: []string{`C:\ws`}}},
		{"mode empty", &sandbox.Spec{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRunner(hooks, "/tmp", spawner, nil)
			if tc.spec != nil {
				r = r.WithSandbox(*tc.spec)
			}
			if _, msg := r.PreToolUse(context.Background(), "bash", nil); msg != "" {
				t.Errorf("non-enforce spec should not warn, got %q", msg)
			}
		})
	}
}

// TestRunnerSandboxUnavailableWarnsOnce: enforce + platform cannot confine ⇒ the
// hook still runs (SpawnInput.Sandbox stays nil, no lying) and the user is told
// exactly once, even across many events.
func TestRunnerSandboxUnavailableWarnsOnce(t *testing.T) {
	hooks := []ResolvedHook{
		{HookConfig: HookConfig{Command: "gate"}, Event: PreToolUse},
		{HookConfig: HookConfig{Command: "post"}, Event: PostToolUse},
		{HookConfig: HookConfig{Command: "stop"}, Event: Stop},
	}
	var notified []string
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		if in.Sandbox != nil {
			t.Errorf("unavailable platform must not get a sandbox spec, got %+v", *in.Sandbox)
		}
		return SpawnResult{ExitCode: 0}
	}
	r := NewRunner(hooks, "/tmp", spawner, func(m string) { notified = append(notified, m) })
	r = r.WithSandbox(enforceSpec())
	r.sandboxAvailable = func() bool { return false }

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		r.PreToolUse(ctx, "bash", nil)
		r.PostToolUse(ctx, "bash", nil, "ok")
		r.Stop(ctx, "done", i)
	}
	if len(notified) != 1 {
		t.Fatalf("notify called %d times (%q), want exactly once", len(notified), notified)
	}
	if !contains(notified[0], "unconfined") || !contains(notified[0], "sandbox") {
		t.Errorf("warning %q should say the sandbox is not applied and commands run unconfined", notified[0])
	}
}

// TestRunnerSandboxAppliedFlagFalseWarns covers the other false signal from
// sandbox.Command: the platform probed available (so the spec *is* injected) but
// the spawner reports it could not actually confine. The wrap's bool must not be
// swallowed (hook.go previously discarded it with `argv, _ :=`).
func TestRunnerSandboxAppliedFlagFalseWarns(t *testing.T) {
	hooks := []ResolvedHook{{HookConfig: HookConfig{Command: "gate"}, Event: PreToolUse}}
	var notified []string
	spawner := func(_ context.Context, in SpawnInput) SpawnResult {
		if in.Sandbox == nil {
			t.Error("available platform should still receive the spec")
		}
		// sandboxApplied defaults to false = "not actually confined".
		return SpawnResult{ExitCode: 0}
	}
	r := NewRunner(hooks, "/tmp", spawner, func(m string) { notified = append(notified, m) })
	r = r.WithSandbox(enforceSpec())
	r.sandboxAvailable = func() bool { return true }

	r.PreToolUse(context.Background(), "bash", nil)
	r.PreToolUse(context.Background(), "bash", nil)
	if len(notified) != 1 {
		t.Fatalf("notify called %d times (%q), want exactly once", len(notified), notified)
	}
	if !contains(notified[0], "unconfined") {
		t.Errorf("warning %q should state the command ran unconfined", notified[0])
	}
}

// TestRunnerSandboxWarnNoNotify: the warning path must be nil-safe (many callers
// pass nil notify) and must not panic on a nil Runner.
func TestRunnerSandboxWarnNoNotify(t *testing.T) {
	hooks := []ResolvedHook{{HookConfig: HookConfig{Command: "gate"}, Event: PreToolUse}}
	r := NewRunner(hooks, "/tmp", func(context.Context, SpawnInput) SpawnResult { return SpawnResult{ExitCode: 0} }, nil)
	r = r.WithSandbox(enforceSpec())
	r.sandboxAvailable = func() bool { return false }
	r.PreToolUse(context.Background(), "bash", nil) // must not panic

	var nilRunner *Runner
	nilRunner.WithSandbox(enforceSpec()).warnSandboxUnavailableOnce()
	if nilRunner.sandboxUsable() {
		t.Error("nil Runner must not claim sandbox availability")
	}
}

// stubSandboxCommand replaces the sandbox seam for one test. ok=true returns a
// portable "wrap" that really runs (a harmless no-op executable) so the spawn is
// exercised end to end; ok=false reports "cannot confine", exactly as the
// platform implementations do when WSL2/bwrap/sandbox-exec is missing or
// WrapCommand fails (seatbelt_windows.go:15-24). Never touches the real
// sandbox.Available, so no test depends on a real WSL2 distro.
func stubSandboxCommand(t *testing.T, ok bool) {
	t.Helper()
	old := hookSandboxCommand
	hookSandboxCommand = func(_ sandbox.Spec, _ sandbox.Shell, command string) ([]string, bool) {
		if !ok {
			return []string{command}, false
		}
		if runtime.GOOS == "windows" {
			return []string{"rundll32.exe", "advapi32.dll,ProcessIdleTasks"}, true
		}
		return []string{"true"}, true
	}
	t.Cleanup(func() { hookSandboxCommand = old })
}

// TestDefaultSpawnerHonoursCommandBool pins the `argv, _ :=` regression at the
// spawner: when the wrap fails (bool false) the result must say so, because
// sandboxApplied is the only evidence that the command was confined. With the
// bool discarded the hook reports "confined" while running naked.
func TestDefaultSpawnerHonoursCommandBool(t *testing.T) {
	for _, ok := range []bool{true, false} {
		stubSandboxCommand(t, ok)
		spec := sandbox.Spec{Mode: "enforce", WriteRoots: []string{t.TempDir()}}
		res := DefaultSpawner(context.Background(), SpawnInput{
			Command: "exit 0", Timeout: 5 * time.Second, Sandbox: &spec,
		})
		if res.sandboxApplied != ok {
			t.Errorf("Command ok=%v → sandboxApplied=%v, want %v", ok, res.sandboxApplied, ok)
		}
	}
}

// TestDefaultSpawnerUnappliedSandboxStillRuns: an unapplied wrap must not turn
// into a skipped or failed hook — it runs unconfined (a hook that silently stops
// running would be a worse failure than an unconfined hook).
func TestDefaultSpawnerUnappliedSandboxStillRuns(t *testing.T) {
	stubSandboxCommand(t, false)
	spec := sandbox.Spec{Mode: "enforce", WriteRoots: []string{t.TempDir()}}
	res := DefaultSpawner(context.Background(), SpawnInput{Command: "exit 0", Timeout: 5 * time.Second, Sandbox: &spec})
	if res.ExitCode != 0 || res.SpawnErr != nil {
		t.Errorf("unconfined fallback should still run the hook: code=%d err=%v", res.ExitCode, res.SpawnErr)
	}
}

// TestRunnerSandboxUnappliedWarnsThroughDefaultSpawner is the end-to-end twin of
// TestRunnerSandboxAppliedFlagFalseWarns: probe says "available", the real
// DefaultSpawner reaches Runner.spawn, and the wrap fails — the user must be
// told, exactly once, that the hook ran unconfined. This is the case that the
// discarded bool used to hide.
func TestRunnerSandboxUnappliedWarnsThroughDefaultSpawner(t *testing.T) {
	stubSandboxCommand(t, false)
	hooks := []ResolvedHook{{HookConfig: HookConfig{Command: "exit 0"}, Event: PreToolUse}}
	var notified []string
	r := NewRunner(hooks, t.TempDir(), nil, func(m string) { notified = append(notified, m) })
	r = r.WithSandbox(sandbox.Spec{Mode: "enforce", WriteRoots: []string{t.TempDir()}})
	r.sandboxAvailable = func() bool { return true }

	ctx := context.Background()
	r.PreToolUse(ctx, "bash", nil)
	r.PreToolUse(ctx, "bash", nil)
	if len(notified) != 1 {
		t.Fatalf("notify called %d times (%q), want exactly once", len(notified), notified)
	}
	if !contains(notified[0], "unconfined") {
		t.Errorf("warning %q should state the hook ran unconfined", notified[0])
	}
}

// --- FormatOutcome ---

func TestFormatOutcomePass(t *testing.T) {
	o := Outcome{
		Hook:     ResolvedHook{HookConfig: HookConfig{Command: "echo hi"}, Event: PreToolUse, Scope: ScopeProject},
		Decision: DecisionPass,
	}
	msg := FormatOutcome(o)
	if msg == "" {
		t.Error("FormatOutcome should not be empty")
	}
}

func TestFormatOutcomeWithDetail(t *testing.T) {
	o := Outcome{
		Hook:      ResolvedHook{HookConfig: HookConfig{Command: "check"}, Event: PreToolUse, Scope: ScopeGlobal},
		Decision:  DecisionBlock,
		Stderr:    "forbidden",
		Truncated: true,
	}
	msg := FormatOutcome(o)
	if !contains(msg, "forbidden") {
		t.Errorf("should include stderr: %s", msg)
	}
	if !contains(msg, "truncated") {
		t.Errorf("should mention truncation: %s", msg)
	}
}

// --- clipRunes ---

func TestClipRunes(t *testing.T) {
	if got := clipRunes("short", 10); got != "short" {
		t.Errorf("clipRunes short = %q", got)
	}
	if got := clipRunes("hello world", 5); got != "hello…" {
		t.Errorf("clipRunes = %q", got)
	}
	if got := clipRunes("", 5); got != "" {
		t.Errorf("clipRunes empty = %q", got)
	}
	if got := clipRunes("abc", 0); got != "" {
		t.Errorf("clipRunes max=0 = %q", got)
	}
}

// --- payload JSON ---

func TestPayloadJSON(t *testing.T) {
	args := json.RawMessage(`{"command":"echo hi"}`)
	p := Payload{
		Event:    PreToolUse,
		Cwd:      "/tmp",
		ToolName: "bash",
		ToolArgs: args,
		Turn:     1,
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Payload
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Event != PreToolUse {
		t.Errorf("Event = %q", decoded.Event)
	}
	if decoded.ToolName != "bash" {
		t.Errorf("ToolName = %q", decoded.ToolName)
	}
	if decoded.Turn != 1 {
		t.Errorf("Turn = %d", decoded.Turn)
	}
}

// --- capping behavior ---

func TestCappedBuffer(t *testing.T) {
	var cb cappedBuffer
	// Write within cap.
	n, err := cb.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Errorf("small write: n=%d err=%v", n, err)
	}
	if cb.truncated {
		t.Error("should not be truncated yet")
	}
	if cb.String() != "hello" {
		t.Errorf("String() = %q", cb.String())
	}

	// Write beyond cap.
	big := make([]byte, outputCapBytes+1000)
	for i := range big {
		big[i] = 'x'
	}
	n, err = cb.Write(big)
	if err != nil || n != len(big) {
		t.Errorf("big write: n=%d err=%v", n, err)
	}
	if !cb.truncated {
		t.Error("should be truncated after exceeding cap")
	}
}

// --- IsBlocking ---

func TestIsBlocking(t *testing.T) {
	if !IsBlocking(PreToolUse) {
		t.Error("PreToolUse should be blocking")
	}
	if !IsBlocking(UserPromptSubmit) {
		t.Error("UserPromptSubmit should be blocking")
	}
	if IsBlocking(PostToolUse) {
		t.Error("PostToolUse should not be blocking")
	}
	if IsBlocking(Stop) {
		t.Error("Stop should not be blocking")
	}
}

// --- defaultTimeout ---

func TestDefaultTimeout(t *testing.T) {
	if defaultTimeout(PreToolUse) != 5*time.Second {
		t.Errorf("PreToolUse timeout = %v", defaultTimeout(PreToolUse))
	}
	if defaultTimeout(PostToolUse) != 30*time.Second {
		t.Errorf("PostToolUse timeout = %v", defaultTimeout(PostToolUse))
	}
}

// helper
func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
