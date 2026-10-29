package pantheon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	mu      sync.Mutex
	calls   []commandSpec
	outputs map[string]func(commandSpec) (string, error)
}

type streamingFakeRunner struct {
	*fakeRunner
	chunks []string
}

type toolStreamingFakeRunner struct {
	*fakeRunner
	next int
}

type blockingRunner struct {
	release <-chan struct{}
}

func (r blockingRunner) Run(ctx context.Context, _ commandSpec) (string, error) {
	select {
	case <-r.release:
		return "should not complete", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (r *fakeRunner) Run(ctx context.Context, spec commandSpec) (string, error) {
	r.mu.Lock()
	r.calls = append(r.calls, spec)
	fn := r.outputs[spec.Executable]
	r.mu.Unlock()
	if fn == nil {
		return "", errors.New("unexpected executable")
	}
	return fn(spec)
}

func (r *streamingFakeRunner) RunStream(
	ctx context.Context,
	spec commandSpec,
	onDelta func(string),
) (string, error) {
	for _, chunk := range r.chunks {
		onDelta(chunk)
	}
	return r.Run(ctx, spec)
}

func (r *toolStreamingFakeRunner) RunStreamWithTools(
	ctx context.Context,
	spec commandSpec,
	onDelta func(string),
	onTool func(ToolTrace),
) (string, error) {
	r.next++
	id := fmt.Sprintf("tool-%d", r.next)
	onTool(ToolTrace{
		ID: id, Name: "命令", Status: "started", Input: "rg --files",
	})
	onDelta("工具结果已用于核验。")
	output, err := r.Run(ctx, spec)
	status := "completed"
	if err != nil {
		status = "failed"
	}
	onTool(ToolTrace{
		ID: id, Name: "命令", Status: status, Output: "README.md\nmain.go",
	})
	return output, err
}

func fakeAdapter(id string) agentAdapter {
	status := AgentStatus{ID: id, Name: strings.ToUpper(id), Kind: "fake", Available: true, Executable: id}
	return agentAdapter{
		id: id, name: status.Name, kind: status.Kind,
		probe: func(context.Context) AgentStatus { return status },
		build: func(prompt, cwd string) (commandSpec, error) {
			return commandSpec{Executable: id, Dir: cwd, Stdin: prompt}, nil
		},
	}
}

func newTestManager(t *testing.T, runner commandRunner, adapters ...agentAdapter) *Manager {
	t.Helper()
	root := t.TempDir()
	store, err := newSessionStore(root)
	if err != nil {
		t.Fatal(err)
	}
	adapterMap := map[string]agentAdapter{}
	for _, adapter := range adapters {
		adapterMap[adapter.id] = adapter
	}
	return &Manager{
		root: root, workDir: root, store: store, runner: runner, adapters: adapterMap,
		overrides: map[string]string{}, cancels: map[string]context.CancelFunc{},
	}
}

func defaultFakeRunner() *fakeRunner {
	return &fakeRunner{outputs: map[string]func(commandSpec) (string, error){
		"alpha": func(spec commandSpec) (string, error) {
			return fakeOutputFor(idFromPrompt(spec.Stdin, "alpha"), spec.Stdin), nil
		},
		"beta": func(spec commandSpec) (string, error) {
			return fakeOutputFor(idFromPrompt(spec.Stdin, "beta"), spec.Stdin), nil
		},
	}}
}

// fakeGrokBackend 内存模型席位后端：CallGrok 只记提示词并复用协议桩输出。
type fakeGrokBackend struct {
	mu      sync.Mutex
	prompts []string
	fail    bool
}

type toolTraceGrokBackend struct {
	fakeGrokBackend
	workspace string
}

func (b *fakeGrokBackend) ProbeGrok(_ context.Context) AgentStatus {
	status := AgentStatus{ID: "grok", Name: "Grok", Kind: "Gaea 模型 · xAI"}
	if b.fail {
		status.Reason = "离线模式下云端模型席位不可用"
		return status
	}
	status.Available = true
	status.Version = "grok-test"
	return status
}

func (b *fakeGrokBackend) CallGrok(_ context.Context, prompt string) (string, error) {
	b.mu.Lock()
	b.prompts = append(b.prompts, prompt)
	b.mu.Unlock()
	return fakeOutputFor("grok", prompt), nil
}

func (b *toolTraceGrokBackend) StreamGrokWithTools(
	_ context.Context,
	prompt string,
	workspace string,
	emit func(string),
	onTool func(ToolTrace),
) (string, error) {
	b.mu.Lock()
	b.prompts = append(b.prompts, prompt)
	b.workspace = workspace
	b.mu.Unlock()
	onTool(ToolTrace{
		ID: "grok-tool-1", Name: "semantic_search", Status: "started",
		Input: `{"query":"办公资料"}`,
	})
	onTool(ToolTrace{
		ID: "grok-tool-1", Name: "semantic_search", Status: "completed",
		Output: "命中 2 条办公资料",
	})
	out := fakeOutputFor("grok", prompt)
	if emit != nil {
		emit(out)
	}
	return out, nil
}

func idFromPrompt(prompt, fallback string) string {
	if strings.Contains(prompt, "「ALPHA」") {
		return "alpha"
	}
	if strings.Contains(prompt, "「BETA」") {
		return "beta"
	}
	return fallback
}

func fakeOutputFor(id, prompt string) string {
	switch {
	case strings.Contains(prompt, "你的职责是协助整理，不是裁判"):
		data, _ := json.Marshal(map[string]any{
			"summary":        "两份意见都认为应先完成自动验证。",
			"recommendation": "先做小范围只读验证，再决定是否扩大。",
			"conditions":     []string{"Agent 均可无头调用"},
			"disagreements":  []string{"验证范围尚未完全一致"},
			"risks":          []string{"外部 Agent 可能不可用"},
			"evidenceNotes":  []string{"材料没有给出历史成功率"},
		})
		return string(data)
	case strings.Contains(prompt, "补充论证"):
		return "我吸收了前文的超时建议，补充一个验证条件：只读且可中止。当前判断保持不变，置信度中。"
	case strings.Contains(prompt, "开场陈述"):
		if id == "alpha" {
			return "我的初步判断：先做只读验证。\n依据：成本低且可中止。\n不确定：缺少历史成功率。"
		}
		return "我的初步判断：可以做验证，但必须设置超时。\n协作请求：请帮助核实失败边界。"
	default:
		return "unexpected prompt"
	}
}

func waitForStatus(t *testing.T, m *Manager, id, status string) *Session {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		session, err := m.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if session.Status == status {
			return session
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session %s did not reach %s", id, status)
	return nil
}

func TestStoreRoundTripListAndExport(t *testing.T) {
	root := t.TempDir()
	store, err := newSessionStore(root)
	if err != nil {
		t.Fatal(err)
	}
	older := &Session{
		ID: "pan_old", Title: "旧议题", Status: StatusCompleted, Phase: PhaseCompleted,
		CreatedAt: 100, UpdatedAt: 200,
		Request:      StartRequest{Topic: "旧议题", Rounds: 1, ReadOnly: true},
		Participants: []Participant{{AgentID: "codex", Name: "Codex", Status: "finished"}},
		Verdict:      &Verdict{Recommendation: "旧结论", Chairman: "Codex", CreatedAt: 200},
	}
	newer := &Session{
		ID: "pan_new", Title: "新议题", Status: StatusCompleted, Phase: PhaseCompleted,
		CreatedAt: 300, UpdatedAt: 400,
		Request: StartRequest{Topic: "新议题", SuccessCriteria: "可复核"},
		Verdict: &Verdict{Recommendation: "新结论", Chairman: "DSH", CreatedAt: 400},
	}
	for _, session := range []*Session{older, newer} {
		if err := store.save(session); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.get("pan_new")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "新议题" || got.Verdict.Recommendation != "新结论" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	list, err := store.list()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != "pan_new" {
		t.Fatalf("list order = %+v", list)
	}
	md := ExportMarkdown(got)
	for _, want := range []string{"# 众神殿议事录", "新议题", "新结论", "用户确认"} {
		if !strings.Contains(md, want) {
			t.Fatalf("export misses %q:\n%s", want, md)
		}
	}
	if err := store.delete("pan_old"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.get("pan_old"); err == nil {
		t.Fatal("deleted session still readable")
	}
}

func TestManagerRunsOrderedMultiRoundDiscussion(t *testing.T) {
	runner := defaultFakeRunner()
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	workspace := t.TempDir()

	session, err := m.Start(StartRequest{
		Topic:           "是否应该接入本机 Agent 进行议事？",
		Context:         "只读，先验证可行性。",
		SuccessCriteria: "给出条件和风险。",
		Workspace:       workspace,
		Evidence:        []EvidenceItem{{Name: "调研", Content: "已有多个竞品。"}},
		AgentIDs:        []string{"alpha", "beta"},
		ChairmanAgentID: "alpha",
		Rounds:          2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Verdict == nil || !strings.Contains(done.Verdict.Recommendation, "只读验证") {
		t.Fatalf("missing verdict: %+v", done.Verdict)
	}
	if len(done.Contributions) != 5 { // 2 agents × 2 rounds + synthesis
		t.Fatalf("contributions = %d, want 5", len(done.Contributions))
	}
	if done.CurrentRound != 2 {
		t.Fatalf("current round = %d, want 2", done.CurrentRound)
	}
	gotOrder := make([]string, 0, 4)
	for _, contribution := range done.Contributions {
		if contribution.Phase == PhaseOpening || contribution.Phase == PhaseDiscussion {
			gotOrder = append(gotOrder, fmt.Sprintf("%d:%s", contribution.Round, contribution.AgentID))
		}
	}
	wantOrder := []string{"1:alpha", "1:beta", "2:beta", "2:alpha"}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("speaking order = %v, want %v", gotOrder, wantOrder)
	}
	if done.Request.Workspace != filepath.Clean(workspace) {
		t.Fatalf("workspace = %q, want %q", done.Request.Workspace, workspace)
	}
	for _, call := range runner.calls {
		if call.Dir != filepath.Clean(workspace) {
			t.Fatalf("agent cwd = %q, want %q", call.Dir, workspace)
		}
		if !strings.Contains(call.Stdin, filepath.Clean(workspace)) {
			t.Fatalf("agent prompt does not contain workspace scope:\n%s", call.Stdin)
		}
		for _, retired := range []string{"首席裁决者", "定向质询", "匿名审阅者"} {
			if strings.Contains(call.Stdin, retired) {
				t.Fatalf("prompt still contains court-style phrase %q:\n%s", retired, call.Stdin)
			}
		}
		if !strings.Contains(call.Stdin, "目标是帮助用户提高判断质量") {
			t.Fatalf("prompt misses collaborative discussion rules:\n%s", call.Stdin)
		}
		if strings.Contains(call.Stdin, "第 2 轮补充论证") && !strings.Contains(call.Stdin, "自动轮换首位") {
			t.Fatalf("discussion prompt misses order rotation notice:\n%s", call.Stdin)
		}
		if strings.Contains(call.Stdin, "第 2 轮补充论证") {
			for _, section := range []string{"【你的独立上下文】", "【其他参与者公开分享的观点】"} {
				if !strings.Contains(call.Stdin, section) {
					t.Fatalf("discussion prompt misses context section %q:\n%s", section, call.Stdin)
				}
			}
		}
	}
	var phases []string
	for _, event := range done.Events {
		if event.Kind == EventPhase {
			phases = append(phases, event.Phase)
		}
	}
	for _, want := range []string{PhaseOpening, PhaseDiscussion, PhaseSynthesis} {
		found := false
		for _, phase := range phases {
			if phase == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing phase %s in %v", want, phases)
		}
	}
	for i, event := range done.Events {
		if event.Seq != int64(i+1) || event.SessionID != session.ID {
			t.Fatalf("event sequence mismatch at %d: %+v", i, event)
		}
	}
	signed, err := m.SignOff(session.ID, "accepted", "同意先试点")
	if err != nil {
		t.Fatal(err)
	}
	if signed.SignOff == nil || signed.SignOff.Decision != "accepted" {
		t.Fatalf("signoff missing: %+v", signed.SignOff)
	}
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(export, "## 用户确认") || !strings.Contains(export, "同意先试点") {
		t.Fatalf("export misses signoff:\n%s", export)
	}
	if !strings.Contains(export, filepath.Clean(workspace)) {
		t.Fatalf("export misses workspace:\n%s", export)
	}
}

func TestManagerRejectsMissingWorkspace(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	missing := filepath.Join(t.TempDir(), "missing-workspace")
	_, err := m.Start(StartRequest{
		Topic:     "不存在的项目",
		Workspace: missing,
		AgentIDs:  []string{"alpha", "beta"},
	})
	if err == nil || !strings.Contains(err.Error(), "议事工作空间不可访问") {
		t.Fatalf("Start error = %v, want inaccessible workspace", err)
	}
}

func TestManagerEmitsStreamingDeltas(t *testing.T) {
	base := defaultFakeRunner()
	runner := &streamingFakeRunner{fakeRunner: base, chunks: []string{"正在补充", "证据与", "推理。"}}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	var mu sync.Mutex
	var deltas []Event
	m.emit = func(event Event) {
		if event.Kind != EventAgentDelta {
			return
		}
		mu.Lock()
		deltas = append(deltas, event)
		mu.Unlock()
	}
	session, err := m.Start(StartRequest{
		Topic: "流式讨论", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, session.ID, StatusCompleted)
	mu.Lock()
	got := append([]Event(nil), deltas...)
	mu.Unlock()
	if len(got) == 0 {
		t.Fatal("no streaming delta events emitted")
	}
	if got[0].AgentID == "" || got[0].Round == 0 || got[0].Phase == "" {
		t.Fatalf("stream delta misses context: %+v", got[0])
	}
}

func TestStreamingLineParser(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{name: "delta", line: `{"type":"delta","delta":"正在补强"}`, want: "正在补强"},
		{name: "completed", line: `{"type":"item.completed","item":{"type":"agent_message","text":"完整观点"}}`, want: "完整观点"},
		{name: "zcode text", line: `{"type":"model.streaming","payload":{"kind":"text_delta","delta":"实时增量"}}`, want: "实时增量"},
		{name: "dsh text", line: `{"type":"text","text":"最终文本"}`, want: "最终文本"},
		{name: "plain", line: "普通流式输出", want: "普通流式输出\n"},
		{name: "control", line: `{"type":"thread.started","thread_id":"t1"}`, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := streamTextFromLine(tc.line); got != tc.want {
				t.Fatalf("streamTextFromLine = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStreamToolTraceParsing(t *testing.T) {
	cases := []struct {
		name       string
		line       string
		wantID     string
		wantName   string
		wantStatus string
		wantInput  string
		wantOutput string
	}{
		{
			name:   "codex start",
			line:   `{"type":"item.started","item":{"id":"item_2","type":"command_execution","command":"Get-ChildItem","aggregated_output":"","status":"in_progress"}}`,
			wantID: "item_2", wantName: "命令", wantStatus: "started", wantInput: "Get-ChildItem",
		},
		{
			name:   "codex result",
			line:   `{"type":"item.completed","item":{"id":"item_2","type":"command_execution","command":"Get-ChildItem","aggregated_output":"README.md","exit_code":0,"status":"completed"}}`,
			wantID: "item_2", wantName: "命令", wantStatus: "completed", wantInput: "Get-ChildItem", wantOutput: "README.md",
		},
		{
			name:   "zcode tool call",
			line:   `{"type":"model.streaming","payload":{"kind":"tool_call","toolCallId":"call_1","toolName":"Bash","input":{"command":"rg --files"}}}`,
			wantID: "call_1", wantName: "Bash", wantStatus: "started", wantInput: `{"command":"rg --files"}`,
		},
		{
			name:   "zcode result",
			line:   `{"type":"tool.updated","payload":{"toolCallId":"call_1","toolName":"Bash","kind":"result","duration":30,"result":{"success":true,"content":"README.md"}}}`,
			wantID: "call_1", wantName: "Bash", wantStatus: "completed", wantOutput: "README.md",
		},
		{
			name:   "dsh call",
			line:   `{"type":"tool_call","callId":"call_2","tool":"pwsh","input":{"command":"Get-ChildItem"}}`,
			wantID: "call_2", wantName: "pwsh", wantStatus: "started", wantInput: `{"command":"Get-ChildItem"}`,
		},
		{
			name:   "dsh result",
			line:   `{"type":"tool_result","callId":"call_2","status":"completed","result":"README.md"}`,
			wantID: "call_2", wantStatus: "completed", wantOutput: "README.md",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toolTraceFromLine(tc.line)
			if got == nil {
				t.Fatal("tool trace not parsed")
			}
			if got.ID != tc.wantID || got.Name != tc.wantName ||
				got.Status != tc.wantStatus || got.Input != tc.wantInput ||
				got.Output != tc.wantOutput {
				t.Fatalf("trace = %+v, want id=%q name=%q status=%q input=%q output=%q",
					got, tc.wantID, tc.wantName, tc.wantStatus, tc.wantInput, tc.wantOutput)
			}
		})
	}
}

func TestManagerEmitsToolTrajectory(t *testing.T) {
	runner := &toolStreamingFakeRunner{fakeRunner: defaultFakeRunner()}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "工具轨迹", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	var starts, results int
	for _, event := range done.Events {
		switch event.Kind {
		case EventToolStart:
			starts++
			if event.ToolID == "" || event.ToolName != "命令" || event.ToolInput != "rg --files" {
				t.Fatalf("tool start missing structured fields: %+v", event)
			}
		case EventToolResult:
			results++
			if event.ToolStatus != "completed" || event.ToolOutput == "" {
				t.Fatalf("tool result missing structured fields: %+v", event)
			}
		}
	}
	if starts < 2 || results < 2 {
		t.Fatalf("tool trace events = starts:%d results:%d", starts, results)
	}
}

func TestConcurrentSessionsKeepDistinctWorkspaces(t *testing.T) {
	runner := defaultFakeRunner()
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	firstWorkspace := t.TempDir()
	secondWorkspace := t.TempDir()
	first, err := m.Start(StartRequest{
		Topic: "项目甲议题", Workspace: firstWorkspace,
		AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.Start(StartRequest{
		Topic: "项目乙议题", Workspace: secondWorkspace,
		AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha", Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, first.ID, StatusCompleted)
	waitForStatus(t, m, second.ID, StatusCompleted)

	runner.mu.Lock()
	calls := append([]commandSpec(nil), runner.calls...)
	runner.mu.Unlock()
	for _, call := range calls {
		switch {
		case strings.Contains(call.Stdin, "项目甲议题"):
			if call.Dir != filepath.Clean(firstWorkspace) {
				t.Fatalf("first session cwd = %q, want %q", call.Dir, firstWorkspace)
			}
		case strings.Contains(call.Stdin, "项目乙议题"):
			if call.Dir != filepath.Clean(secondWorkspace) {
				t.Fatalf("second session cwd = %q, want %q", call.Dir, secondWorkspace)
			}
		}
	}
}

func TestManagerIsolatesAgentFailure(t *testing.T) {
	runner := defaultFakeRunner()
	runner.outputs["beta"] = func(commandSpec) (string, error) {
		return "", errors.New("beta unavailable")
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic: "失败隔离", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "beta", Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Verdict == nil {
		t.Fatal("alpha should still produce a verdict")
	}
	if !done.Verdict.Partial {
		t.Fatal("single-agent verdict must be marked partial")
	}
	var failed bool
	for _, p := range done.Participants {
		if p.AgentID == "beta" && p.Status == "failed" && strings.Contains(p.Error, "beta unavailable") {
			failed = true
		}
	}
	if !failed {
		t.Fatalf("beta failure not preserved: %+v", done.Participants)
	}
}

func TestManagerCancel(t *testing.T) {
	block := make(chan struct{})
	m := newTestManager(t, blockingRunner{release: block}, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer close(block)
	defer m.Close()

	session, err := m.Start(StartRequest{Topic: "取消", AgentIDs: []string{"alpha", "beta"}, Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	if err := m.Cancel(session.ID); err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCancelled)
	if done.Error == "" {
		t.Fatal("cancel reason should be visible")
	}
}

func TestParseAgentOutputFormats(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"codex", `{"type":"item.completed","item":{"type":"agent_message","text":"最终答复"}}`, "最终答复"},
		{"zcode", `{"sessionId":"s","response":"ready","eventCount":4}`, "ready"},
		{"dsh", "{\"type\":\"result\",\"result\":\"完成\"}", "完成"},
		{"plain", "直接文本", "直接文本"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseAgentOutput(tc.raw); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeRoundSettings(t *testing.T) {
	cases := []struct {
		name string
		in   StartRequest
		want StartRequest
	}{
		{
			name: "manual default",
			in:   StartRequest{},
			want: StartRequest{Rounds: 2},
		},
		{
			name: "manual input",
			in:   StartRequest{Rounds: 7},
			want: StartRequest{Rounds: 7},
		},
		{
			name: "manual minimum",
			in:   StartRequest{Rounds: 1},
			want: StartRequest{Rounds: 2},
		},
		{
			name: "manual upper bound",
			in:   StartRequest{Rounds: 99},
			want: StartRequest{Rounds: 20},
		},
		{
			name: "consensus default",
			in:   StartRequest{UntilConsensus: true},
			want: StartRequest{UntilConsensus: true, Rounds: 12, MaxRounds: 12},
		},
		{
			name: "consensus requested limit",
			in:   StartRequest{UntilConsensus: true, MaxRounds: 8},
			want: StartRequest{UntilConsensus: true, Rounds: 8, MaxRounds: 8},
		},
		{
			name: "consensus upper bound",
			in:   StartRequest{UntilConsensus: true, MaxRounds: 99},
			want: StartRequest{UntilConsensus: true, Rounds: 20, MaxRounds: 20},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			normalizeRoundSettings(&got)
			if got.Rounds != tc.want.Rounds ||
				got.MaxRounds != tc.want.MaxRounds ||
				got.UntilConsensus != tc.want.UntilConsensus {
				t.Fatalf("normalized = %+v, want rounds=%d max=%d consensus=%v",
					got, tc.want.Rounds, tc.want.MaxRounds, tc.want.UntilConsensus)
			}
		})
	}
}

func TestToolPolicyMatchesSeatCapabilities(t *testing.T) {
	local := toolPolicy(AgentStatus{ID: "codex", Name: "Codex"})
	if !strings.Contains(local, "只读工具") || !strings.Contains(local, "不得尝试写文件") {
		t.Fatalf("local tool policy missing read-only guard:\n%s", local)
	}
	grok := toolPolicy(AgentStatus{ID: "grok", Name: "Grok"})
	if !strings.Contains(grok, "Gaea 办公工具") || !strings.Contains(grok, "不得写盘") {
		t.Fatalf("grok office tool policy is not truthful:\n%s", grok)
	}
}

func TestZCodePromptRequestsStreamJSON(t *testing.T) {
	args := zcodePromptArgs("分析当前议题", `C:\AI\wubigrok`)
	if !containsAdjacent(args, "--output-format", "stream-json") {
		t.Fatalf("zcode args do not request stream-json: %v", args)
	}
	if containsValue(args, "--json") {
		t.Fatalf("zcode args still contain one-shot --json: %v", args)
	}
}

func containsAdjacent(values []string, first, second string) bool {
	for i := 0; i+1 < len(values); i++ {
		if values[i] == first && values[i+1] == second {
			return true
		}
	}
	return false
}

func containsValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestDSHAdapterInjectsDeepseekKey(t *testing.T) {
	// override 指向假 .cmd：绕过真实安装探测，单测不依赖本机 dsh
	fake := filepath.Join(t.TempDir(), "dsh.cmd")
	if err := os.WriteFile(fake, []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := newDSHAdapter(fake, "sk-test").build("议题", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !containsValue(spec.Env, "DEEPSEEK_API_KEY=sk-test") {
		t.Fatalf("dsh env missing injected key: %v", spec.Env)
	}
	if !containsValue(spec.Env, "DSH_PERMISSION_MODE=read-only") {
		t.Fatalf("dsh env lost permission mode: %v", spec.Env)
	}
	bare, err := newDSHAdapter(fake, "").build("议题", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, kv := range bare.Env {
		if strings.HasPrefix(kv, "DEEPSEEK_API_KEY=") {
			t.Fatalf("empty key must not inject (would shadow user env): %v", bare.Env)
		}
	}
}

func TestExtractConsensusSignal(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantText string
		want     string
	}{
		{"chinese yes", "当前判断保持不变。\n共识状态：一致\n剩余分歧：无", "当前判断保持不变。\n剩余分歧：无", "yes"},
		{"chinese no", "- 共识状态：仍有分歧\n剩余分歧：成本边界", "剩余分歧：成本边界", "no"},
		{"english yes", "Conclusion.\nCONSENSUS: yes", "Conclusion.", "yes"},
		{"missing", "没有结构化信号", "没有结构化信号", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, signal := extractConsensusSignal(tc.input)
			if text != tc.wantText || signal != tc.want {
				t.Fatalf("got (%q,%q), want (%q,%q)", text, signal, tc.wantText, tc.want)
			}
		})
	}
}

func consensusRunner(signal string) *fakeRunner {
	runner := defaultFakeRunner()
	for id, original := range runner.outputs {
		output := original
		runner.outputs[id] = func(spec commandSpec) (string, error) {
			text, err := output(spec)
			if err == nil && strings.Contains(spec.Stdin, "【全员一致检查】") {
				text += "\n共识状态：" + signal + "\n剩余分歧：成本边界"
			}
			return text, err
		}
	}
	return runner
}

func TestManagerStopsWhenEverySeatReachesConsensus(t *testing.T) {
	runner := consensusRunner("一致")
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic: "自动共识", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Request.Rounds != 8 || session.Request.MaxRounds != 8 {
		t.Fatalf("consensus request not normalized: %+v", session.Request)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.CurrentRound != 2 {
		t.Fatalf("consensus should stop after round 2, got round %d", done.CurrentRound)
	}
	if len(done.Contributions) != 5 {
		t.Fatalf("contributions = %d, want 5", len(done.Contributions))
	}
	var consensusEvents int
	for _, event := range done.Events {
		if event.Kind == EventConsensus {
			consensusEvents++
		}
	}
	if consensusEvents != 1 {
		t.Fatalf("consensus events = %d, want 1", consensusEvents)
	}
	if strings.Contains(done.Contributions[2].Content, "共识状态") {
		t.Fatal("machine-readable consensus marker leaked into displayed contribution")
	}
	if done.Contributions[2].Consensus != "yes" {
		t.Fatalf("contribution consensus = %q, want yes", done.Contributions[2].Consensus)
	}
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"讨论模式：不限轮次，至全员一致", "实际轮次：2"} {
		if !strings.Contains(export, want) {
			t.Fatalf("consensus export misses %q:\n%s", want, export)
		}
	}
}

func TestManagerStopsAtConsensusSafetyLimit(t *testing.T) {
	runner := consensusRunner("仍有分歧")
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic: "未形成共识", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		UntilConsensus: true, MaxRounds: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.CurrentRound != 3 {
		t.Fatalf("safety limit current round = %d, want 3", done.CurrentRound)
	}
	if len(done.Contributions) != 7 {
		t.Fatalf("contributions = %d, want 7", len(done.Contributions))
	}
	found := false
	for _, warning := range done.Warnings {
		if strings.Contains(warning, "安全上限") && strings.Contains(warning, "仍未全员一致") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing safety-limit warning: %v", done.Warnings)
	}
}

func TestClampPromptKeepsBothBriefAndInstructions(t *testing.T) {
	short := "短提示词"
	if got := clampPrompt(short); got != short {
		t.Fatalf("short prompt changed: %q", got)
	}

	begin := "【议题】保留开头"
	end := "最后输出 JSON，字段 recommendation"
	long := begin + strings.Repeat("中间讨论内容", maxPromptRunes) + end
	got := clampPrompt(long)
	if len([]rune(got)) > maxPromptRunes {
		t.Fatalf("clamped prompt length = %d, max = %d", len([]rune(got)), maxPromptRunes)
	}
	if !strings.HasPrefix(got, begin) {
		t.Fatal("clamped prompt lost the brief at the head")
	}
	if !strings.HasSuffix(got, end) {
		t.Fatal("clamped prompt lost response instructions at the tail")
	}
	if !strings.Contains(got, promptClipMarker) {
		t.Fatal("clamped prompt did not disclose omitted middle content")
	}
}

func TestRecoverInterruptedCouncilPreservesPartialRecord(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session := &Session{
		ID: "pan_interrupted", Title: "中断恢复", Status: StatusRunning, Phase: PhaseOpening,
		CreatedAt: 100, UpdatedAt: 200,
		Request: StartRequest{Topic: "中断议题", AgentIDs: []string{"alpha", "beta"}, Rounds: 2, ReadOnly: true},
		Participants: []Participant{
			{AgentID: "alpha", Name: "ALPHA", Status: "running"},
			{AgentID: "beta", Name: "BETA", Status: "finished"},
		},
		Contributions: []Contribution{{
			ID: "kept", AgentID: "beta", AgentName: "BETA", Content: "保留这条发言",
		}},
	}
	if err := m.store.save(session); err != nil {
		t.Fatal(err)
	}

	recovered, err := m.RecoverInterrupted()
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}
	got, err := m.Get(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusFailed || got.Phase != PhaseFailed {
		t.Fatalf("recovered status = %s/%s", got.Status, got.Phase)
	}
	if !strings.Contains(got.Error, "上次运行被应用中断") {
		t.Fatalf("missing interruption reason: %q", got.Error)
	}
	if len(got.Contributions) != 1 || got.Contributions[0].Content != "保留这条发言" {
		t.Fatal("partial contributions were not preserved")
	}
	if got.Participants[0].Status != "failed" || got.Participants[0].Error == "" {
		t.Fatalf("running participant not closed: %+v", got.Participants[0])
	}
	if len(got.Events) != 1 || got.Events[0].Kind != EventError {
		t.Fatalf("recovery event missing: %+v", got.Events)
	}

	again, err := m.RecoverInterrupted()
	if err != nil {
		t.Fatal(err)
	}
	if again != 0 {
		t.Fatalf("second recovery = %d, want idempotent zero", again)
	}
}

func TestStoreRejectsPathTraversal(t *testing.T) {
	store, err := newSessionStore(filepath.Join(t.TempDir(), "pantheon"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.get("../../escape"); err == nil {
		t.Fatal("path traversal should be rejected")
	}
}

func TestGrokSeatJoinsCouncilAndChairs(t *testing.T) {
	backend := &fakeGrokBackend{}
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"), newGrokAdapter(backend))
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic:    "Grok 主持可行性",
		AgentIDs: []string{"alpha", "beta", "grok"},
		Rounds:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Request.ChairmanAgentID != "grok" {
		t.Fatalf("default chairman = %q, want grok", session.Request.ChairmanAgentID)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Verdict == nil || done.Verdict.Chairman != "Grok" {
		t.Fatalf("verdict chairman = %+v", done.Verdict)
	}
	var grokOpening bool
	for _, c := range done.Contributions {
		if c.AgentID == "grok" && c.Phase == PhaseOpening {
			grokOpening = true
		}
	}
	if !grokOpening {
		t.Fatal("grok seat produced no opening contribution")
	}
	backend.mu.Lock()
	calls := len(backend.prompts)
	backend.mu.Unlock()
	// grok 席位自身：2 轮讨论 + 1 次协作汇总 = 3 次，全部走注入后端。
	if calls != 3 {
		t.Fatalf("model backend calls = %d, want 3", calls)
	}
}

func TestGrokToolAwareBackendReceivesWorkspaceAndEmitsTrajectory(t *testing.T) {
	backend := &toolTraceGrokBackend{}
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), newGrokAdapter(backend))
	defer m.Close()
	workspace := t.TempDir()
	session, err := m.Start(StartRequest{
		Topic: "Grok 办公工具", Workspace: workspace,
		AgentIDs: []string{"grok", "alpha"}, ChairmanAgentID: "alpha", Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	backend.mu.Lock()
	gotWorkspace := backend.workspace
	backend.mu.Unlock()
	if gotWorkspace != filepath.Clean(workspace) {
		t.Fatalf("Grok workspace = %q, want %q", gotWorkspace, filepath.Clean(workspace))
	}
	var starts, results int
	for _, event := range done.Events {
		if event.AgentID != "grok" {
			continue
		}
		if event.Kind == EventToolStart && event.ToolName == "semantic_search" {
			starts++
		}
		if event.Kind == EventToolResult && event.ToolName == "semantic_search" &&
			strings.Contains(event.ToolOutput, "办公资料") {
			results++
		}
	}
	if starts == 0 || results == 0 {
		t.Fatalf("Grok tool trajectory missing: starts=%d results=%d", starts, results)
	}
}

func TestGrokUnavailableFallsBackChairman(t *testing.T) {
	backend := &fakeGrokBackend{fail: true}
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"), newGrokAdapter(backend))
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic: "席位降级", AgentIDs: []string{"grok", "alpha", "beta"}, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Request.ChairmanAgentID == "grok" {
		t.Fatal("unavailable grok must not chair")
	}
	found := false
	for _, p := range session.Participants {
		if p.AgentID == "grok" && p.Status == "unavailable" && p.Error != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("grok seat not marked unavailable: %+v", session.Participants)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Verdict == nil || !done.Verdict.Partial {
		t.Fatalf("degraded council should still complete with a partial verdict: %+v", done.Verdict)
	}
}

func TestSetExecutableRejectsModelSeat(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), newGrokAdapter(&fakeGrokBackend{}))
	if _, err := m.SetExecutable("grok", `C:\nonexistent\grok.exe`); err == nil {
		t.Fatal("grok seat must reject executable override")
	}
}
