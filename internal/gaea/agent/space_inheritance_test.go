package agent

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ── S3 双空间：子代理空间继承 + 防穿越校验 ──────────────────────────
// 设计权威：docs/gaea-space-dimension-design.md §4 + §7 S3 行。
//   ① 子会话继承父 space（ctx 带 play → 子代理落在 play；缺省 = work）
//   ② 空间不一致 fail-closed（runSubAgentInternal 断言 + PrepareContinue 校验）
//   ③ skillRunner 继承（boot skillRunner 的调用形状 = sctx 带 space → RunSubAgent）
//   ④ SpaceFromContext 缺省 work

// spaceProbeTool 记录自己 Execute 时 ctx 里的空间——子代理继承链的观察哨：
// space 必须穿过 TaskTool.Execute → runSubSession → RunSubAgent → 子 runDirect
// → executeOne 才能到达这里，任何一环丢失都会让断言失败。
type spaceProbeTool struct {
	name  string
	space atomic.Value // string
	calls int32
}

func (p *spaceProbeTool) Name() string            { return p.name }
func (p *spaceProbeTool) Description() string     { return "" }
func (p *spaceProbeTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (p *spaceProbeTool) ReadOnly() bool          { return true }

func (p *spaceProbeTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	p.space.Store(SpaceFromContext(ctx))
	atomic.AddInt32(&p.calls, 1)
	return "probed:" + SpaceFromContext(ctx), nil
}

func (p *spaceProbeTool) seenSpace() string {
	if s, ok := p.space.Load().(string); ok {
		return s
	}
	return ""
}

// runTaskWithProbe 用 scripted 子代理 provider 跑一次 task 工具：第一轮让子代理
// 调 probe 工具，第二轮给最终答案，返回 probe 观察到的空间。
func runTaskWithProbe(t *testing.T, ctx context.Context) string {
	t.Helper()
	probe := &spaceProbeTool{name: "space_probe"}
	sub := &scriptedProvider{name: "sub", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "space_probe", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "final"}, {Type: provider.ChunkDone}},
	}}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task)
	parentReg.Add(probe)

	out, err := task.Execute(ctx, []byte(`{"prompt":"probe the space"}`))
	if err != nil {
		t.Fatalf("task Execute: %v", err)
	}
	if atomic.LoadInt32(&probe.calls) == 0 {
		t.Fatal("子代理没有执行 probe 工具，继承链未打通")
	}
	if !strings.Contains(out, "final") {
		t.Errorf("task output = %q, want sub-agent final answer", out)
	}
	return probe.seenSpace()
}

// TestSubagentInheritsParentSpace ①：父运行上下文带 play → 子代理工具看到 play
// （TaskTool 前台路径，executeOne 交付的 ctx 形状 = withCallContext + 空间）。
func TestSubagentInheritsParentSpace(t *testing.T) {
	ctx := WithSpace(withCallContext(context.Background(), "call_1", &testSink{}, nil), spaces.SpacePlay)
	if got := runTaskWithProbe(t, ctx); got != spaces.SpacePlay {
		t.Fatalf("子代理观察到的空间 = %q, want %q（play 父必须派生 play 子代理）", got, spaces.SpacePlay)
	}
}

// TestSubagentDefaultsToWorkSpace ①（缺省）：ctx 无空间标注（headless 直调、
// 后台 job 重建 ctx 等）→ 子代理落在 work，行为与改造前一致。
func TestSubagentDefaultsToWorkSpace(t *testing.T) {
	if got := runTaskWithProbe(t, withCallContext(context.Background(), "call_1", &testSink{}, nil)); got != spaces.SpaceWork {
		t.Fatalf("缺省子代理空间 = %q, want %q", got, spaces.SpaceWork)
	}
}

// TestSubagentBackgroundJobInheritsSpace ①（后台）：run_in_background 路径的
// jobCtx 由 jobs.Manager 的 root 新建、不带父 ctx 的 value——TaskTool 必须显式
// 补注空间，否则 play 父的后台子代理会掉回 work。
func TestSubagentBackgroundJobInheritsSpace(t *testing.T) {
	sink := &testSink{}
	jm := jobs.NewManager(sink)
	ctx := WithSpace(jobs.WithManager(withCallContext(context.Background(), "call_1", sink, nil), jm), spaces.SpacePlay)

	probe := &spaceProbeTool{name: "space_probe"}
	sub := &scriptedProvider{name: "sub", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "space_probe", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "final"}, {Type: provider.ChunkDone}},
	}}
	parentReg := tool.NewRegistry()
	task := NewTaskTool(sub, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task)
	parentReg.Add(probe)

	out, err := task.Execute(ctx, []byte(`{"prompt":"probe","run_in_background":true}`))
	if err != nil {
		t.Fatalf("task Execute (background): %v", err)
	}
	if !strings.Contains(out, "Started background task") {
		t.Fatalf("output = %q, want background job acknowledgement", out)
	}
	results := jm.Wait(context.Background(), nil, 10)
	if len(results) != 1 {
		t.Fatalf("job results = %d, want 1", len(results))
	}
	if !strings.Contains(results[0].Output, "final") {
		t.Fatalf("后台子代理未正常完成: %q", results[0].Output)
	}
	if atomic.LoadInt32(&probe.calls) == 0 {
		t.Fatal("后台子代理没有执行 probe 工具")
	}
	if got := probe.seenSpace(); got != spaces.SpacePlay {
		t.Fatalf("后台子代理观察到的空间 = %q, want %q（jobCtx 必须补注父空间）", got, spaces.SpacePlay)
	}
}

// TestSubagentSpaceMismatchFailClosed ②：携带异空间自描述的子会话到达执行链
// 时必须报错拒绝（fail-closed），子代理绝不运行。
func TestSubagentSpaceMismatchFailClosed(t *testing.T) {
	probe := &spaceProbeTool{name: "space_probe"}
	reg := tool.NewRegistry()
	reg.Add(probe)
	prov := &scriptedProvider{name: "sub", turns: [][]provider.Chunk{
		{{Type: provider.ChunkText, Text: "should never run"}, {Type: provider.ChunkDone}},
	}}

	// play 会话 vs work 上下文（ctx 无标注 = work）。
	sess := NewSession("sys")
	sess.SetSpace(spaces.SpacePlay)
	_, err := RunSubAgentWithSession(context.Background(), prov, reg, sess, "x", Options{DisableVerify: true}, event.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "space mismatch") {
		t.Fatalf("play 会话 + work ctx 应 fail-closed，got err=%v", err)
	}

	// 反向：work 会话 vs play 上下文。
	sess2 := NewSession("sys")
	sess2.SetSpace(spaces.SpaceWork)
	_, err = RunSubAgentWithSession(WithSpace(context.Background(), spaces.SpacePlay), prov, reg, sess2, "x", Options{DisableVerify: true}, event.Discard, nil)
	if err == nil || !strings.Contains(err.Error(), "space mismatch") {
		t.Fatalf("work 会话 + play ctx 应 fail-closed，got err=%v", err)
	}

	// fail-closed 意味着子代理从未运行：provider 零调用、probe 零执行。
	if prov.call != 0 {
		t.Fatalf("fail-closed 后 provider 仍被调用 %d 次，子代理不应运行", prov.call)
	}
	if atomic.LoadInt32(&probe.calls) != 0 {
		t.Fatal("fail-closed 后 probe 仍被执行，子代理不应运行")
	}
}

// TestRunSubAgentWithSessionInheritsEmptySpace continue_from 路径：装载的子会话
// 无空间自描述（旧转录）时标记父上下文空间，运行成功后会话空间保持一致。
func TestRunSubAgentWithSessionInheritsEmptySpace(t *testing.T) {
	probe := &spaceProbeTool{name: "space_probe"}
	reg := tool.NewRegistry()
	reg.Add(probe)
	prov := &scriptedProvider{name: "sub", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "space_probe", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "continued"}, {Type: provider.ChunkDone}},
	}}
	sess := NewSession("sys") // Space() == ""（旧转录形态）
	out, err := RunSubAgentWithSession(WithSpace(context.Background(), spaces.SpacePlay), prov, reg, sess, "continue", Options{DisableVerify: true}, event.Discard, nil)
	if err != nil {
		t.Fatalf("RunSubAgentWithSession: %v", err)
	}
	if !strings.Contains(out, "continued") {
		t.Errorf("output = %q, want final answer", out)
	}
	if got := sess.Space(); got != spaces.SpacePlay {
		t.Fatalf("装载的子会话空间 = %q, want %q（继承后必须落定）", got, spaces.SpacePlay)
	}
	if got := probe.seenSpace(); got != spaces.SpacePlay {
		t.Fatalf("子代理工具观察到的空间 = %q, want %q", got, spaces.SpacePlay)
	}
}

// TestSkillRunnerShapeInheritsSpace ③：boot.go skillRunner 的调用形状——
// sctx 是 run_skill 工具收到的调用 ctx（父 Run 已注入空间），直接交给
// agent.RunSubAgent。技能子代理必须与 task 子代理同样继承空间。
func TestSkillRunnerShapeInheritsSpace(t *testing.T) {
	probe := &spaceProbeTool{name: "space_probe"}
	reg := tool.NewRegistry()
	reg.Add(probe)
	prov := &scriptedProvider{name: "skill-sub", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "space_probe", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "skill done"}, {Type: provider.ChunkDone}},
	}}

	// skillRunner: sctx = WithSpace(调用ctx, play) → agent.RunSubAgent(sctx, ...)
	sctx := WithSpace(context.Background(), spaces.SpacePlay)
	out, err := RunSubAgent(sctx, prov, reg, "skill sys", "do it", Options{DisableVerify: true}, event.Discard, nil)
	if err != nil {
		t.Fatalf("RunSubAgent (skillRunner shape): %v", err)
	}
	if !strings.Contains(out, "skill done") {
		t.Errorf("output = %q, want final answer", out)
	}
	if got := probe.seenSpace(); got != spaces.SpacePlay {
		t.Fatalf("技能子代理观察到的空间 = %q, want %q", got, spaces.SpacePlay)
	}
}

// TestSpaceFromContextDefaultsToWork ④：SpaceFromContext 缺省 work + withSpace
// 归一化语义（空值/非法值 → work，play 透传）。
func TestSpaceFromContextDefaultsToWork(t *testing.T) {
	if got := SpaceFromContext(context.Background()); got != spaces.SpaceWork {
		t.Fatalf("SpaceFromContext(Background) = %q, want %q", got, spaces.SpaceWork)
	}
	if got := SpaceFromContext(WithSpace(context.Background(), spaces.SpacePlay)); got != spaces.SpacePlay {
		t.Fatalf("SpaceFromContext(play) = %q, want %q", got, spaces.SpacePlay)
	}
	// 空值与非法值经 Normalize 归一为 work（space.mode=off 平铺形态兼容）。
	for _, in := range []string{"", "bogus", "WORK"} {
		if got := SpaceFromContext(WithSpace(context.Background(), in)); got != spaces.SpaceWork {
			t.Fatalf("SpaceFromContext(WithSpace(%q)) = %q, want %q", in, got, spaces.SpaceWork)
		}
	}
	// 防御：绕过 withSpace 塞入裸值也归一（私有 key 同包可构造）。
	raw := context.WithValue(context.Background(), spaceContextKey{}, "bogus")
	if got := SpaceFromContext(raw); got != spaces.SpaceWork {
		t.Fatalf("SpaceFromContext(raw %q) = %q, want %q", "bogus", got, spaces.SpaceWork)
	}
}

// TestRunDirectStampsSessionSpace 端到端：agent Run 把会话空间注入运行 ctx——
// 工具经 SpaceFromContext 读到会话空间（agent_run.go 的注入点语义）。
func TestRunDirectStampsSessionSpace(t *testing.T) {
	probe := &spaceProbeTool{name: "space_probe"}
	reg := tool.NewRegistry()
	reg.Add(probe)
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "space_probe", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	sess := NewSession("")
	sess.SetSpace(spaces.SpacePlay)
	a := New(prov, reg, sess, Options{DisableVerify: true}, event.Discard)
	if _, err := a.Run(context.Background(), "hello"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := probe.seenSpace(); got != spaces.SpacePlay {
		t.Fatalf("工具观察到的运行空间 = %q, want %q（Run 必须把会话空间注入 ctx）", got, spaces.SpacePlay)
	}
	// 缺省：会话无空间标注（平铺形态）→ ctx 落 work。
	probe2 := &spaceProbeTool{name: "space_probe"}
	reg2 := tool.NewRegistry()
	reg2.Add(probe2)
	prov2 := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{toolCallChunk("c1", "space_probe", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "done"}, {Type: provider.ChunkDone}},
	}}
	a2 := New(prov2, reg2, NewSession(""), Options{DisableVerify: true}, event.Discard)
	if _, err := a2.Run(context.Background(), "hello"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := probe2.seenSpace(); got != spaces.SpaceWork {
		t.Fatalf("缺省运行空间 = %q, want %q", got, spaces.SpaceWork)
	}
}

// TestPrepareContinueSpaceMismatch ②（前瞻 C）：SubagentStore 的 continue 路径
// 校验请求空间与 ref 空间一致（meta 带 space 落盘 + 旧 meta 无字段按 work 降级）。
func TestPrepareContinueSpaceMismatch(t *testing.T) {
	store := NewSubagentStore(t.TempDir())

	run, err := store.PrepareFresh("sys", spaces.SpacePlay)
	if err != nil {
		t.Fatalf("PrepareFresh: %v", err)
	}
	if got := run.Session.Space(); got != spaces.SpacePlay {
		t.Fatalf("PrepareFresh 子会话空间 = %q, want %q", got, spaces.SpacePlay)
	}
	run.Session.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	if err := store.SaveCompleted(run); err != nil {
		t.Fatalf("SaveCompleted: %v", err)
	}

	// meta 落盘带 space 字段。
	meta, err := store.loadMeta(run.Ref)
	if err != nil {
		t.Fatalf("loadMeta: %v", err)
	}
	if meta.Space != spaces.SpacePlay {
		t.Fatalf("meta.Space = %q, want %q", meta.Space, spaces.SpacePlay)
	}

	// 同空间续跑放行。
	cont, err := store.PrepareContinue(run.Ref, spaces.SpacePlay)
	if err != nil {
		t.Fatalf("PrepareContinue(play): %v", err)
	}
	cont.Release()

	// 跨空间续跑 fail-closed。
	if _, err := store.PrepareContinue(run.Ref, spaces.SpaceWork); err == nil || !strings.Contains(err.Error(), "space mismatch") {
		t.Fatalf("play ref + work 请求应 fail-closed，got err=%v", err)
	}

	// 旧 meta 无 space 字段 → 读端降级 work：work 请求放行、play 请求拒绝。
	legacyRef := run.Ref
	metaPath := store.metaPath(legacyRef)
	b, err := json.Marshal(map[string]any{"ref": legacyRef, "status": "completed"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	cont2, err := store.PrepareContinue(legacyRef, "")
	if err != nil {
		t.Fatalf("legacy meta + work 请求应放行，got err=%v", err)
	}
	cont2.Release()
	if _, err := store.PrepareContinue(legacyRef, spaces.SpacePlay); err == nil || !strings.Contains(err.Error(), "space mismatch") {
		t.Fatalf("legacy meta（降级 work）+ play 请求应 fail-closed，got err=%v", err)
	}
}
