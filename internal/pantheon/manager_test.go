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
	status := AgentStatus{
		ID: id, Name: strings.ToUpper(id), Kind: "fake", Available: true, Executable: id,
		Capabilities: []string{"image-input"},
	}
	return agentAdapter{
		id: id, name: status.Name, kind: status.Kind,
		runMode: "内存桩（只读）",
		probe:   func(context.Context) AgentStatus { return status },
		build: func(prompt, cwd string, _ []string) (commandSpec, error) {
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
		agentCalls: map[string]map[string]*activeAgentCall{}, runs: map[string]*Session{},
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
	readPaths []string
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
	readPaths []string,
	emit func(string),
	onTool func(ToolTrace),
) (string, error) {
	b.mu.Lock()
	b.prompts = append(b.prompts, prompt)
	b.workspace = workspace
	b.readPaths = append([]string(nil), readPaths...)
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

func TestManagerPersistsCallRefsAcrossContributionEvidence(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"))
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic: "Attempt 身份贯通", AgentIDs: []string{"alpha"},
		Protocol: ProtocolIndependent, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	completed := waitForStatus(t, m, session.ID, StatusCompleted)
	if len(completed.Contributions) < 2 {
		t.Fatalf("contributions = %d, want opening + synthesis", len(completed.Contributions))
	}
	for _, contribution := range completed.Contributions {
		if contribution.CallID == "" || contribution.AttemptID == "" || contribution.Attempt < 1 {
			t.Fatalf("contribution missing call identity: %+v", contribution)
		}
		var sawStart, sawOutput bool
		for _, event := range completed.Events {
			if event.CallID != contribution.CallID ||
				event.AttemptID != contribution.AttemptID {
				continue
			}
			sawStart = sawStart || event.Kind == EventAgentStart
			sawOutput = sawOutput || event.Kind == EventAgentOutput
		}
		if !sawStart || !sawOutput {
			t.Fatalf("contribution lacks matching start/output evidence: %+v", contribution)
		}
	}
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

func TestStoreRepairsLegacyZCodeEventWallOnRead(t *testing.T) {
	store, err := newSessionStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	readableWall := `{"eventId":"e1","messageId":"msg_hidden","type":"session.titleUpdated","payload":{"title":"提示词"}} ` +
		`{"eventId":"e2","type":"model.streaming","payload":{"kind":"text_delta","delta":"可读"}} ` +
		`{"eventId":"e3","type":"model.streaming","payload":{"kind":"text_delta","delta":"观点"}}`
	metadataOnly := `{"eventId":"e1","messageId":"msg_hidden","type":"session.titleUpdated","payload":{"title":"提示词"}}`
	session := &Session{
		ID: "pan_legacy_zcode", Title: "历史 ZCode", Status: StatusCompleted,
		Phase: PhaseCompleted, CreatedAt: 1, UpdatedAt: 2,
		Request: StartRequest{Topic: "历史", Rounds: 1, ReadOnly: true},
		Contributions: []Contribution{
			{ID: "c1", AgentID: "zcode", Content: readableWall, CreatedAt: 3},
			{ID: "c2", AgentID: "zcode", Content: metadataOnly, CreatedAt: 4},
		},
		Events: []Event{{
			Seq: 1, SessionID: "pan_legacy_zcode", Kind: EventAgentOutput,
			AgentID: "zcode", Text: readableWall, At: 3,
		}},
	}
	if err := store.save(session); err != nil {
		t.Fatal(err)
	}
	got, err := store.get(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Contributions[0].Content != "可读观点" {
		t.Fatalf("repaired contribution = %q", got.Contributions[0].Content)
	}
	if strings.Contains(got.Contributions[1].Content, "messageId") ||
		!strings.Contains(got.Contributions[1].Content, "请重试本轮观点") {
		t.Fatalf("unparsed legacy wall was not replaced honestly: %q", got.Contributions[1].Content)
	}
	if got.Events[0].Text != "可读观点" {
		t.Fatalf("repaired event = %q", got.Events[0].Text)
	}
}

// 历史文件可能带 "agentIds": null（AgentIDs 无 omitempty 的 nil 切片契约），
// 读回必须归一化成空切片——前端记录页裸 .map，null 直达会崩页。
func TestStoreGetNormalizesNullAgentIDs(t *testing.T) {
	root := t.TempDir()
	store, err := newSessionStore(root)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "sessions", "pan_null")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"id":"pan_null","title":"空席历史件","status":"completed","phase":"completed",
		"createdAt":1,"updatedAt":2,"request":{"topic":"历史文件","rounds":1,"readOnly":true,"agentIds":null},
		"participants":null,"events":null,"contributions":null}`
	if err := os.WriteFile(filepath.Join(dir, "session.json"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := store.get("pan_null")
	if err != nil {
		t.Fatal(err)
	}
	if got.Request.AgentIDs == nil || len(got.Request.AgentIDs) != 0 {
		t.Fatalf("agentIDs not normalized: %#v", got.Request.AgentIDs)
	}
	if got.Participants == nil || got.Events == nil || got.Contributions == nil {
		t.Fatalf("slices not normalized: %+v", got)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"agentIds":null`) {
		t.Fatalf("agentIds still null after normalize: %s", data)
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
	// 2 席×2 轮 + 第 1 轮主席小结 + synthesis = 6（2026-11-03 主席轮间小结机制）。
	if len(done.Contributions) != 6 {
		t.Fatalf("contributions = %d, want 6", len(done.Contributions))
	}
	if done.CurrentRound != 2 {
		t.Fatalf("current round = %d, want 2", done.CurrentRound)
	}
	// 第 1 轮并行开场（2026-11-03）：落盘序=完成序（与直播一致，防定稿重排
	// 跳变），两位都在即可；第 2 轮起串行，轮内恒守既定序（rotate 后 beta 先）。
	round1 := map[string]bool{}
	gotOrder := make([]string, 0, 4)
	for _, contribution := range done.Contributions {
		if contribution.Phase == PhaseOpening || contribution.Phase == PhaseDiscussion {
			key := fmt.Sprintf("%d:%s", contribution.Round, contribution.AgentID)
			if contribution.Round == 1 {
				round1[key] = true
				continue
			}
			gotOrder = append(gotOrder, key)
		}
	}
	if len(round1) != 2 || !round1["1:alpha"] || !round1["1:beta"] {
		t.Fatalf("round-1 parallel contributions missing: %v", round1)
	}
	wantOrder := []string{"2:beta", "2:alpha"}
	if !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("round-2 speaking order = %v, want %v", gotOrder, wantOrder)
	}
	if done.Request.Workspace != filepath.Clean(workspace) {
		t.Fatalf("workspace = %q, want %q", done.Request.Workspace, workspace)
	}
	for _, call := range runner.calls {
		if strings.Contains(call.Stdin, "轮间小结") {
			continue // 主席小结是对内议程提炼，不携带工作空间范围。
		}
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

func TestManagerWritesChairFinalDocument(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Title:    "最终文档验证",
		Topic:    "讨论结束后是否生成最终文档？",
		AgentIDs: []string{"alpha", "beta"},
		Rounds:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.ResultPath == "" {
		t.Fatal("completed session missing resultPath")
	}
	data, err := os.ReadFile(done.ResultPath)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	for _, want := range []string{
		"# 最终文档验证",
		"## 最终建议",
		done.Verdict.Recommendation,
		done.Verdict.Chairman,
		"## 各席观点",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("final document misses %q:\n%s", want, doc)
		}
	}
	path, err := m.ResultPath(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if path != done.ResultPath {
		t.Fatalf("ResultPath = %q, want %q", path, done.ResultPath)
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
		{
			// 新版信封：顶层 type=turn.completed，聚合全文直播时作为最终态播出。
			name: "zcode new envelope final",
			line: `{"eventId":"x","type":"turn.completed","payload":{"response":"聚合全文","resultType":"success"}}`,
			want: "聚合全文",
		},
		{name: "dsh text", line: `{"type":"text","text":"最终文本"}`, want: "最终文本"},
		{name: "plain", line: "普通流式输出", want: "普通流式输出\n"},
		{name: "control", line: `{"type":"thread.started","thread_id":"t1"}`, want: ""},
		{
			// Codex 每轮把未知 config 键告警包成 item.completed/error 重播，
			// 属于 Harness 噪音，不得混进流式发言（2026-10-30 实弹乱码线）。
			name: "codex harness warning",
			line: `{"type":"item.completed","item":{"type":"error","message":"Codex is ignoring 1 unrecognized configuration setting."}}`,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := streamTextFromLine(tc.line); got != tc.want {
				t.Fatalf("streamTextFromLine = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseAgentOutputSplitsConcatenatedZCodeEvents(t *testing.T) {
	// 2026-11-02 实弹探针：新版桌面运行时的回合完成在顶层 type=turn.completed
	// （payload 无 kind 键），response 全文在 payload 里。
	raw := `{"eventId":"e1","messageId":"msg_bad_wall_payload","type":"session.titleUpdated","payload":{"title":"提示词"}} {"eventId":"e2","type":"model.streaming","payload":{"kind":"reasoning_delta","delta":"内部推理"}} {"eventId":"e3","type":"model.streaming","payload":{"kind":"text_delta","delta":"最终"}} {"eventId":"e4","type":"model.streaming","payload":{"kind":"text_delta","delta":"答复"}} {"eventId":"e5","type":"model.streaming","payload":{"kind":"finish","done":true}} {"eventId":"e6","type":"turn.completed","payload":{"response":"最终答复","usage":{"outputTokens":43}}}`
	got := parseAgentOutput(raw)
	if got != "最终答复" {
		t.Fatalf("parseAgentOutput = %q, want 最终答复", got)
	}
	if strings.Contains(got, "messageId") || strings.Contains(got, "payload") {
		t.Fatalf("event metadata leaked into answer: %q", got)
	}
}

func TestStreamWriterSplitsConcatenatedJSONWithoutNewlines(t *testing.T) {
	var deltas []string
	var traces []ToolTrace
	writer := &streamTextWriter{
		onDelta: func(text string) { deltas = append(deltas, text) },
		onTool:  func(trace ToolTrace) { traces = append(traces, trace) },
	}
	toolEvent := `{"type":"model.streaming","payload":{"kind":"tool_call","toolCallId":"call_1","toolName":"Bash","input":{"command":"rg --files"}}}`
	textEvent := `{"eventId":"e2","messageId":"msg_hidden","type":"model.streaming","payload":{"kind":"text_delta","delta":"可读输出"}}`
	stream := toolEvent + " " + textEvent
	for _, part := range []string{stream[:90], stream[90:190], stream[190:]} {
		if _, err := writer.Write([]byte(part)); err != nil {
			t.Fatal(err)
		}
	}
	writer.Flush()

	if strings.Join(deltas, "") != "可读输出" {
		t.Fatalf("deltas = %q, want 可读输出", strings.Join(deltas, ""))
	}
	if len(traces) != 1 || traces[0].ID != "call_1" || traces[0].Name != "Bash" {
		t.Fatalf("tool traces = %+v", traces)
	}
}

func TestStreamWriterSuppressesFullTextEchoAfterDeltas(t *testing.T) {
	// 2026-10-30 实弹重复输出根修：codex 等 CLI 先流增量 delta，收尾再发
	// item.completed/agent_message 全文事件；增量已播过流，全文必须吞掉，
	// 否则前端拼接后整段发言输出两遍（只影响流式席位，grok 模型席无此问题）。
	var deltas []string
	writer := &streamTextWriter{
		onDelta: func(text string) { deltas = append(deltas, text) },
	}
	stream := strings.Join([]string{
		`{"type":"item.delta","delta":"第一"}`,
		`{"type":"item.delta","delta":"段增量"}`,
		`{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"第一段增量"}}`,
		`{"type":"thread.completed","usage":{"tokens":10}}`,
		// 新版信封（顶层 type）：聚合全文在增量之后到达，同样必须吞。
		`{"eventId":"x","type":"turn.completed","payload":{"response":"第一段增量","resultType":"success"}}`,
	}, "\n")
	if _, err := writer.Write([]byte(stream)); err != nil {
		t.Fatal(err)
	}
	writer.Flush()
	if got := strings.Join(deltas, ""); got != "第一段增量" {
		t.Fatalf("deltas = %q, want 第一段增量（全文回声未抑制）", got)
	}

	// 无增量的纯最终态 CLI：全文事件是唯一文本来源，不得拦截。
	var finalOnly []string
	plain := &streamTextWriter{
		onDelta: func(text string) { finalOnly = append(finalOnly, text) },
	}
	if _, err := plain.Write([]byte(`{"type":"item.completed","item":{"type":"agent_message","text":"只有全文"}}`)); err != nil {
		t.Fatal(err)
	}
	plain.Flush()
	if got := strings.Join(finalOnly, ""); got != "只有全文" {
		t.Fatalf("final-only deltas = %q, want 只有全文", got)
	}

	// 新版信封无增量：turn.completed 聚合全文是唯一文本来源，同样播出一次。
	var envelopeOnly []string
	envelope := &streamTextWriter{
		onDelta: func(text string) { envelopeOnly = append(envelopeOnly, text) },
	}
	if _, err := envelope.Write([]byte(`{"eventId":"x","type":"turn.completed","payload":{"response":"信封全文"}}`)); err != nil {
		t.Fatal(err)
	}
	envelope.Flush()
	if got := strings.Join(envelopeOnly, ""); got != "信封全文" {
		t.Fatalf("envelope-only deltas = %q, want 信封全文", got)
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

func TestManagerSkipCurrentAgentContinuesToNext(t *testing.T) {
	block := make(chan struct{})
	m := newTestManager(t, blockingRunner{release: block}, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer close(block)
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic: "跳过当前席位", AgentIDs: []string{"alpha", "beta"}, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	var current string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := m.Get(session.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, participant := range got.Participants {
			if participant.Status == "running" {
				current = participant.AgentID
				break
			}
		}
		if current != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current == "" {
		t.Fatal("no agent started running")
	}
	time.Sleep(10 * time.Millisecond)
	if err := m.SkipAgent(session.ID, current); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(2 * time.Second)
	var advanced *Session
	for time.Now().Before(deadline) {
		got, err := m.Get(session.ID)
		if err != nil {
			t.Fatal(err)
		}
		var nextRunning bool
		for _, participant := range got.Participants {
			if participant.AgentID != current && participant.Status == "running" {
				nextRunning = true
			}
		}
		if nextRunning {
			advanced = got
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if advanced == nil {
		t.Fatal("discussion did not advance to the next agent")
	}
	var skipped bool
	for _, participant := range advanced.Participants {
		if participant.AgentID == current && participant.Status == "skipped" {
			skipped = true
		}
	}
	if !skipped {
		t.Fatalf("skipped participant state missing: %+v", advanced.Participants)
	}
	var skippedEvent bool
	for _, event := range advanced.Events {
		if event.Kind == EventAgentSkipped && event.AgentID == current {
			skippedEvent = true
		}
		if event.Kind == EventAgentError && event.AgentID == current {
			t.Fatalf("skip must not be recorded as an error: %+v", event)
		}
	}
	if !skippedEvent {
		t.Fatal("agent_skipped event missing")
	}
	if err := m.Cancel(session.ID); err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, session.ID, StatusCancelled)
}

// residueRunner 模拟进程被杀后管道余量继续流入：ctx 取消后仍向 onDelta 发射
// 残留增量再返回（实弹 2026-10-31：agent_skipped 事件之后 1.6s 内还有
// tool/delta 事件从被跳过席位的管道排空）。
type residueRunner struct{}

func (residueRunner) Run(ctx context.Context, _ commandSpec) (string, error) {
	return "", ctx.Err()
}

func (residueRunner) RunStream(ctx context.Context, _ commandSpec, onDelta func(string)) (string, error) {
	<-ctx.Done()
	onDelta("跳过后管道残留增量")
	return "", ctx.Err()
}

func TestManagerSkipDropsResidueDeltas(t *testing.T) {
	m := newTestManager(t, residueRunner{}, fakeAdapter("alpha"), fakeAdapter("beta"))
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
		Topic: "跳过后残影不得入流", AgentIDs: []string{"alpha", "beta"}, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	var current string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := m.Get(session.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, participant := range got.Participants {
			if participant.Status == "running" {
				current = participant.AgentID
				break
			}
		}
		if current != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current == "" {
		t.Fatal("no agent started running")
	}
	if err := m.SkipAgent(session.ID, current); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(2 * time.Second)
	var advanced bool
	for time.Now().Before(deadline) {
		got, err := m.Get(session.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, participant := range got.Participants {
			if participant.AgentID != current && participant.Status == "running" {
				advanced = true
			}
		}
		if advanced {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !advanced {
		t.Fatal("discussion did not advance after skip")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, delta := range deltas {
		if delta.AgentID == current {
			t.Fatalf("残影增量未被丢弃：%+v", delta)
		}
	}
}

func TestSupersededAgentCallIsAborted(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner())
	// 多槽（会话×席位，2026-11-03 并行刀）：同席替换判定按 (sessionID, agentID)。
	oldCall := &activeAgentCall{sessionID: "pan_test", CallRef: CallRef{AttemptID: "att_1"}}
	currentCall := &activeAgentCall{sessionID: "pan_test", agentID: "alpha", CallRef: CallRef{AttemptID: "att_2"}}
	m.agentCalls["pan_test"] = map[string]*activeAgentCall{"alpha": currentCall}

	if !m.agentCallAborted(oldCall) {
		t.Fatal("superseded attempt must be rejected")
	}
	if m.agentCallAborted(currentCall) {
		t.Fatal("current attempt must remain writable")
	}
}

func TestManagerRetryCurrentAgentKeepsRoundPosition(t *testing.T) {
	block := make(chan struct{})
	m := newTestManager(t, blockingRunner{release: block}, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer close(block)
	defer m.Close()

	session, err := m.Start(StartRequest{
		// 圆桌接力协议=串行语义：并行开场（independent 默认）下其它席同时
		// running 是合法态，「retry 不推进到其它席」的断言只在串行下可判。
		Topic: "重试当前席位", Protocol: ProtocolCouncil, AgentIDs: []string{"alpha", "beta"}, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	var current string
	var firstStart int64
	var firstCall CallRef
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := m.Get(session.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, participant := range got.Participants {
			if participant.Status == "running" {
				current = participant.AgentID
				firstStart = participant.StartedAt
				firstCall = participant.CallRef
				break
			}
		}
		if current != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current == "" {
		t.Fatal("no agent started running")
	}
	if firstCall.CallID == "" || firstCall.AttemptID == "" || firstCall.Attempt != 1 {
		t.Fatalf("first attempt identity missing: %+v", firstCall)
	}
	time.Sleep(10 * time.Millisecond)
	if err := m.RetryAgent(session.ID, current); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(2 * time.Second)
	var retried *Session
	for time.Now().Before(deadline) {
		got, err := m.Get(session.ID)
		if err != nil {
			t.Fatal(err)
		}
		var retryEvent bool
		var currentRunning bool
		for _, event := range got.Events {
			if event.Kind == EventAgentRetry && event.AgentID == current {
				retryEvent = true
			}
		}
		for _, participant := range got.Participants {
			if participant.AgentID == current &&
				participant.Status == "running" &&
				participant.StartedAt > firstStart {
				currentRunning = true
			}
			if participant.AgentID != current && participant.Status == "running" {
				t.Fatalf("retry advanced to another agent: %+v", participant)
			}
		}
		if retryEvent && currentRunning {
			retried = got
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if retried == nil {
		t.Fatal("current agent was not retried in place")
	}
	var retriedCall CallRef
	for _, participant := range retried.Participants {
		if participant.AgentID == current {
			retriedCall = participant.CallRef
		}
	}
	if retriedCall.CallID != firstCall.CallID {
		t.Fatalf("retry changed logical call ID: %+v -> %+v", firstCall, retriedCall)
	}
	if retriedCall.AttemptID == firstCall.AttemptID || retriedCall.Attempt != 2 {
		t.Fatalf("retry did not create a new attempt: %+v -> %+v", firstCall, retriedCall)
	}
	if retriedCall.RetryOfAttempt != firstCall.AttemptID {
		t.Fatalf("retry lineage = %q, want %q", retriedCall.RetryOfAttempt, firstCall.AttemptID)
	}
	var sawRetry, sawRetryStart bool
	for _, event := range retried.Events {
		if event.Kind == EventAgentError && event.AgentID == current {
			t.Fatalf("retry must not be recorded as an error: %+v", event)
		}
		if event.Kind == EventAgentRetry && event.AgentID == current {
			sawRetry = event.CallID == firstCall.CallID && event.AttemptID == firstCall.AttemptID
		}
		if event.Kind == EventAgentStart &&
			event.AgentID == current &&
			event.Attempt == 2 &&
			event.AttemptID == retriedCall.AttemptID {
			sawRetryStart = event.CallID == firstCall.CallID &&
				event.RetryOfAttempt == firstCall.AttemptID
		}
	}
	if !sawRetry || !sawRetryStart {
		t.Fatalf("retry events missing attempt lineage: retry=%v start=%v events=%+v",
			sawRetry, sawRetryStart, retried.Events)
	}
	if err := m.Cancel(session.ID); err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, session.ID, StatusCancelled)
}

func TestManagerRestartReusesBriefAndCancelsOldRun(t *testing.T) {
	block := make(chan struct{})
	m := newTestManager(t, blockingRunner{release: block}, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer close(block)
	defer m.Close()

	imagePath := filepath.Join(m.root, "diagram.png")
	if err := os.WriteFile(imagePath, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := m.Start(StartRequest{
		Title:       "卡死恢复",
		Topic:       "验证重新开始会保留议事配置",
		Workspace:   m.root,
		Evidence:    []EvidenceItem{{Name: "材料", Content: "保留这段证据"}},
		Attachments: []Attachment{{Name: "diagram.png", Path: imagePath}},
		AgentIDs:    []string{"alpha", "beta"},
		Rounds:      3,
	})
	if err != nil {
		t.Fatal(err)
	}

	restarted, err := m.Restart(original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restarted.ID == "" || restarted.ID == original.ID {
		t.Fatalf("restart id = %q, original = %q", restarted.ID, original.ID)
	}
	want := original.Request
	got := restarted.Request
	if got.Title != want.Title || got.Topic != want.Topic ||
		got.Workspace != want.Workspace || got.Rounds != want.Rounds ||
		!reflect.DeepEqual(got.AgentIDs, want.AgentIDs) ||
		!reflect.DeepEqual(got.Evidence, want.Evidence) ||
		!reflect.DeepEqual(got.Attachments, want.Attachments) {
		t.Fatalf("restart did not preserve brief:\n got %+v\nwant %+v", got, want)
	}

	waitForStatus(t, m, original.ID, StatusCancelled)
	if err := m.Cancel(restarted.ID); err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, restarted.ID, StatusCancelled)
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
		{
			// 告警项夹在正式发言里：只保留发言，不把告警兜底成陈词。
			name: "warning beside message",
			raw: `{"type":"item.completed","item":{"type":"error","message":"Codex is ignoring 1 unrecognized configuration setting."}}
{"type":"item.completed","item":{"type":"agent_message","text":"最终答复"}}`,
			want: "最终答复",
		},
		{
			// 整段只有告警：按空结果处理，上层走 agent_error 而非存假陈词。
			name: "only warning",
			raw:  `{"type":"item.completed","item":{"type":"error","message":"Codex is ignoring 1 unrecognized configuration setting."}}`,
			want: "",
		},
		{
			// zcode 新版桌面运行时：答案只存在于 text_delta 增量流里
			// （reasoning_delta 不算发言），按序重装全文（2026-10-30 实弹线）。
			// 增量重装须见到**回合级**完成（turn.completed）——payload.kind=
			// finish 是每次模型请求的收尾，工具型回合有 N 个，不算回合完成。
			name: "zcode text_delta stream with turn completion",
			raw: `{"type":"model.streaming","payload":{"kind":"reasoning_delta","delta":"思考"}}
{"type":"model.streaming","payload":{"kind":"text_delta","delta":"最终"}}
{"type":"model.streaming","payload":{"kind":"text_delta","delta":"答复"}}
{"type":"model.streaming","payload":{"kind":"finish","done":true}}
{"type":"turn.completed","payload":{"response":"最终答复","usage":{"outputTokens":43}}}`,
			want: "最终答复",
		},
		{
			// 2026-11-02 实弹 zcode 第 2/3 轮（511/301 字旁白冒充陈词）的根因
			// 回归：CLI 中途静默退出（exit 0），只有 text_delta 旁白和模型请求
			// 级 finish、没有 turn.completed/result——必须返回空走 agent_error。
			name: "zcode bare finish without turn completion",
			raw: `{"type":"model.streaming","payload":{"kind":"reasoning_delta","delta":"思考"}}
{"type":"model.streaming","payload":{"kind":"text_delta","delta":"收到，我先做两件只读核验。"}}
{"type":"model.streaming","payload":{"kind":"text_delta","delta":"再查两个子目录确认是否藏在不存在的目录里："}}
{"type":"model.streaming","payload":{"kind":"finish","done":true}}`,
			want: "",
		},
		{
			// 信封聚合事件（turn.completed.response）是收尾权威全文：即使增量
			// 缺失或尾部被截断也能恢复答案（2026-10-31 实弹 zcode 格式）。
			// 旧版布局：payload.kind=turn.completed。
			name: "zcode envelope aggregated final",
			raw: `{"eventId":"a","payload":{"kind":"reasoning_delta","delta":"推理"}}
{"eventId":"b","payload":{"kind":"text_delta","delta":"收到"}}
{"eventId":"c","payload":{"kind":"finish","done":true}}
{"eventId":"d","payload":{"kind":"turn.completed","response":"收到，这是完整最终陈词。","usage":{"outputTokens":43}}}`,
			want: "收到，这是完整最终陈词。",
		},
		{
			// 新版布局（2026-11-02 实弹探针）：顶层 type=turn.completed，
			// payload 无 kind 键，response/usage 都在 payload 里。
			name: "zcode new envelope turn.completed",
			raw: `{"eventId":"a","type":"model.streaming","payload":{"kind":"text_delta","delta":"收到"}}
{"eventId":"b","type":"turn.completed","payload":{"response":"收到，正式陈述全文。","resultType":"success","usage":{"outputTokens":43}}}`,
			want: "收到，正式陈述全文。",
		},
		{
			// 回合失败（turn.failed）：即使有增量残留也不得入档。
			name: "zcode turn.failed",
			raw: `{"type":"model.streaming","payload":{"kind":"text_delta","delta":"旁白片段"}}
{"type":"turn.failed","payload":{"error":"provider overload"}}`,
			want: "",
		},
		{
			// 中途夭折（超时/被杀）：只有 text_delta 旁白、无 finish 收尾——
			// 返回空走 agent_error，不把过程旁白冒充席位观点（2026-10-31 实弹
			// zcode 第 1/3 轮 149/187 字旁白的根因）。
			name: "zcode killed stream has no finish",
			raw: `{"eventId":"a","payload":{"kind":"text_delta","delta":"找到了一批核心文档。"}}
{"eventId":"b","payload":{"kind":"text_delta","delta":"继续读规划正文。"}}`,
			want: "",
		},
		{
			// 纯 JSON 事件流但没有任何可识别文本（未知新 schema）：
			// 返回空走 agent_error，不把事件墙原样存成席位陈词。
			name: "unknown event wall",
			raw: `{"type":"session.titleUpdated","payload":{"title":"t"}}
{"type":"session.updated","payload":{"seq":2}}`,
			want: "",
		},
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
			name: "manual single round allowed",
			in:   StartRequest{Rounds: 1},
			want: StartRequest{Rounds: 1},
		},
		{
			name: "manual zero default",
			in:   StartRequest{Rounds: 0},
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
	if !strings.Contains(local, "只读工具") || !strings.Contains(local, "写面纪律") ||
		!strings.Contains(local, "每场专属 seatwork 目录") || !strings.Contains(local, "产物不是证据") {
		t.Fatalf("local tool policy missing the scoped write-face guard:\n%s", local)
	}
	grok := toolPolicy(AgentStatus{ID: "grok", Name: "Grok"})
	if !strings.Contains(grok, "Gaea 办公工具") || !strings.Contains(grok, "不得写盘") {
		t.Fatalf("grok office tool policy is not truthful:\n%s", grok)
	}
}

func TestCollaborationRulesForbidImplicitDelegation(t *testing.T) {
	rules := collaborationRules()
	for _, want := range []string{"亲自完成本轮发言", "不得派生 subagent", "二次委派"} {
		if !strings.Contains(rules, want) {
			t.Fatalf("collaboration rules missing %q:\n%s", want, rules)
		}
	}
}

func TestSupportsImageAttachmentChannels(t *testing.T) {
	for _, capability := range []string{"image-input", "image-path-tools", "shared-image-summary"} {
		if !supportsImageAttachment(AgentStatus{Capabilities: []string{capability}}) {
			t.Fatalf("capability %q must satisfy image attachment preflight", capability)
		}
	}
	if supportsImageAttachment(AgentStatus{}) {
		t.Fatal("seat without an image channel must be rejected")
	}
}

func TestManagerRejectsImagesForUnsupportedSeat(t *testing.T) {
	withoutImages := fakeAdapter("alpha")
	withoutImages.probe = func(context.Context) AgentStatus {
		return AgentStatus{
			ID: "alpha", Name: "Alpha", Kind: "fake", Available: true, Executable: "alpha",
		}
	}
	m := newTestManager(t, defaultFakeRunner(), withoutImages, fakeAdapter("beta"))
	defer m.Close()

	image := filepath.Join(m.root, "diagram.png")
	if err := os.WriteFile(image, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := m.Start(StartRequest{
		Topic:       "图片能力门",
		Attachments: []Attachment{{Name: "diagram.png", Path: image}},
		AgentIDs:    []string{"alpha", "beta"},
		Rounds:      1,
	})
	if err == nil || !strings.Contains(err.Error(), "Alpha 不支持图片附件") {
		t.Fatalf("Start error = %v, want unsupported image capability", err)
	}
}

func TestNormalizeAttachmentsValidatesAndDeduplicatesImages(t *testing.T) {
	root := t.TempDir()
	image := filepath.Join(root, "diagram.png")
	if err := os.WriteFile(image, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizeAttachments([]Attachment{
		{Path: image, Name: "图一"},
		{Path: image, Name: "重复"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("normalized attachments = %d, want 1", len(got))
	}
	if got[0].MIME != "image/png" || got[0].Size != 3 || got[0].Name != "图一" {
		t.Fatalf("normalized attachment = %+v", got[0])
	}

	text := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(text, []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeAttachments([]Attachment{{Path: text}}); err == nil {
		t.Fatal("text file must not pass image attachment validation")
	}
}

func TestZCodePromptRequestsStreamJSON(t *testing.T) {
	args := zcodePromptArgs("分析当前议题", `C:\AI\wubigrok`, []string{`C:\AI\wubigrok\diagram.png`})
	if !containsAdjacent(args, "--output-format", "stream-json") {
		t.Fatalf("zcode args do not request stream-json: %v", args)
	}
	if !containsAdjacent(args, "--attach", `C:\AI\wubigrok\diagram.png`) {
		t.Fatalf("zcode args lost image attachment: %v", args)
	}
	if containsValue(args, "--json") {
		t.Fatalf("zcode args still contain one-shot --json: %v", args)
	}
}

// codexOverride 用假可执行文件代替真实 codex：spec 构造只做路径解析与拼参，
// 断言不应依赖本机是否装了 codex（测试环境 PATH 常缺 npm 目录，真实路径会
// 让整段断言静默 SKIP）。
func codexOverride(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codex.cmd")
	if err := os.WriteFile(path, []byte("@echo off\r\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCodexExecEnablesLiveSearchAndImageInput(t *testing.T) {
	image := `C:\AI\wubigrok\diagram.png`
	spec, err := newCodexAdapter(codexOverride(t)).build("分析图片", `C:\AI\wubigrok`, []string{image})
	if err != nil {
		t.Skipf("codex executable unavailable: %v", err)
	}
	if !containsAdjacent(spec.Args, "-c", `web_search="live"`) {
		t.Fatalf("codex args do not enable live web search: %v", spec.Args)
	}
	// 2026-11-03 用户拍板「都能上网查资料」：read-only 沙箱默认禁网（直连
	// GitHub API 实测被拦），放宽 workspace-write+network——写限工作区。
	// 2026-11-07 A5②：沙箱改走 -c sandbox_mode（resume 子命令拒收 -s，两种
	// 调用形态统一用 config 覆盖）。
	if !containsAdjacent(spec.Args, "-c", `sandbox_mode="workspace-write"`) {
		t.Fatalf("codex args must use workspace-write sandbox for network: %v", spec.Args)
	}
	if !containsAdjacent(spec.Args, "-c", `sandbox_workspace_write.network_access=true`) {
		t.Fatalf("codex args must enable network access: %v", spec.Args)
	}
	if containsAdjacent(spec.Args, "--sandbox", "read-only") {
		t.Fatalf("codex args must not regress to read-only: %v", spec.Args)
	}
	if !containsAdjacent(spec.Args, "--image", image) {
		t.Fatalf("codex args lost image attachment: %v", spec.Args)
	}
	// A5②：非 ephemeral——thread 必须落 rollout，下一轮才有的续。
	if containsValue(spec.Args, "--ephemeral") {
		t.Fatalf("codex exec must not be ephemeral (rollout required for resume): %v", spec.Args)
	}
}

func TestCodexResumeSpecCarriesSessionID(t *testing.T) {
	image := `C:\AI\wubigrok\diagram.png`
	adapter := newCodexAdapter(codexOverride(t))
	if adapter.buildResume == nil {
		t.Fatal("codex adapter must declare buildResume (A5②)")
	}
	spec, err := adapter.buildResume("继续", `C:\AI\wubigrok`, []string{image}, "01a1b609-a539-74a0-aee5-f3e152643663")
	if err != nil {
		t.Skipf("codex executable unavailable: %v", err)
	}
	if !containsAdjacent(spec.Args, "resume", "01a1b609-a539-74a0-aee5-f3e152643663") {
		t.Fatalf("resume spec must carry session id: %v", spec.Args)
	}
	if !containsAdjacent(spec.Args, "-c", `sandbox_mode="workspace-write"`) {
		t.Fatalf("resume spec must set sandbox via -c (resume rejects -s): %v", spec.Args)
	}
	if containsValue(spec.Args, "--ephemeral") {
		t.Fatalf("resume spec must not be ephemeral: %v", spec.Args)
	}
	// A5② 实弹修正（pan_1794058438005）：resume 子命令拒收 -C（exec 顶层参数），
	// 工作目录走 commandSpec.Dir；两形态都不得再出现。
	if containsValue(spec.Args, "-C") {
		t.Fatalf("resume spec must not carry -C (resume rejects it): %v", spec.Args)
	}
	if !containsAdjacent(spec.Args, "--image", image) {
		t.Fatalf("resume spec lost image attachment: %v", spec.Args)
	}
	// resumeID 为空必须等价 build（接口契约：不硬错，回落新建）。
	fresh, err := adapter.buildResume("开场", `C:\AI\wubigrok`, nil, "  ")
	if err != nil {
		t.Fatalf("empty resumeID must fall back to fresh exec: %v", err)
	}
	if containsValue(fresh.Args, "resume") {
		t.Fatalf("empty resumeID must not emit resume subcommand: %v", fresh.Args)
	}
	if containsValue(fresh.Args, "-C") {
		t.Fatalf("fresh exec spec must not carry -C (Dir field covers cwd): %v", fresh.Args)
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
	spec, err := newDSHAdapter(fake, "sk-test").build("议题", t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !containsValue(spec.Env, "DEEPSEEK_API_KEY=sk-test") {
		t.Fatalf("dsh env missing injected key: %v", spec.Env)
	}
	if !containsValue(spec.Env, "DSH_PERMISSION_MODE=read-only") {
		t.Fatalf("dsh env lost permission mode: %v", spec.Env)
	}
	bare, err := newDSHAdapter(fake, "").build("议题", t.TempDir(), nil)
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
	// 2026-11-03 V2 P1：剩余分歧行也抽走（Contribution.Residual），停机门读它。
	// 旧用例曾把「剩余分歧：无」留在正文里当预期——grok r2 实证这正是
	// 「自报一致+正文藏分歧」能同时成立的洞，已翻转。
	cases := []struct {
		name         string
		input        string
		wantText     string
		want         string
		wantResidual string
	}{
		{"chinese yes", "当前判断保持不变。\n共识状态：一致\n剩余分歧：无", "当前判断保持不变。", "yes", "无"},
		{"chinese no", "- 共识状态：仍有分歧\n剩余分歧：成本边界", "", "no", "成本边界"},
		{"english yes", "Conclusion.\nCONSENSUS: yes", "Conclusion.", "yes", ""},
		{"missing", "没有结构化信号", "没有结构化信号", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text, signal, residual := extractConsensusSignal(tc.input)
			if text != tc.wantText || signal != tc.want || residual != tc.wantResidual {
				t.Fatalf("got (%q,%q,%q), want (%q,%q,%q)", text, signal, residual, tc.wantText, tc.want, tc.wantResidual)
			}
		})
	}
}

func TestExtractConsensusSignalDetails(t *testing.T) {
	input := `当前判断保持不变。
共识状态：仍有分歧
剩余分歧：接口契约未冻结
分歧依据：pan_stream-beta-r1-123 的接口差异清单
分歧对象：pan_stream-beta-r1-123
翻转条件：接口差异低于 3 项且有迁移验证
下一步核验：读取 api/schema.json 并与 v2 客户端对拍
置信度：中`
	details := extractConsensusSignalDetails(input)
	if details.Text != "当前判断保持不变。\n置信度：中" {
		t.Fatalf("clean text = %q", details.Text)
	}
	if details.Consensus != "no" || details.Residual != "接口契约未冻结" {
		t.Fatalf("signal/residual = %q/%q", details.Consensus, details.Residual)
	}
	if details.ResidualEvidence != "pan_stream-beta-r1-123 的接口差异清单" ||
		details.ResidualAgainst != "pan_stream-beta-r1-123" ||
		details.ResidualFlipCondition != "接口差异低于 3 项且有迁移验证" ||
		details.ResidualVerification != "读取 api/schema.json 并与 v2 客户端对拍" {
		t.Fatalf("structured details missing: %+v", details)
	}
}

func TestIssueCounterpointsUsesOnlyExplicitKnownContributions(t *testing.T) {
	history := []Contribution{
		{ID: "pan_demo-alpha-r1-100", AgentID: "alpha", AgentName: "Alpha"},
		{ID: "pan_demo-beta-r1-200", AgentID: "beta", AgentName: "Beta"},
	}
	contribution := Contribution{
		AgentID:         "beta",
		AgentName:       "Beta",
		ResidualAgainst: "pan_demo-alpha-r1-100, pan_demo-beta-r1-200, pan_demo-unknown-r1-999",
	}
	refs := issueCounterpoints(contribution, history)
	if len(refs) != 1 || refs[0].AgentID != "alpha" {
		t.Fatalf("counterpoints = %+v, want only alpha's explicit contribution", refs)
	}
	issue := &Issue{AgentID: "beta", AgentName: "Beta"}
	applyIssueAttribution(issue, contribution, refs)
	if len(issue.DissentAgentNames) != 1 || issue.DissentAgentNames[0] != "Beta" ||
		len(issue.ClaimAgentNames) != 1 || issue.ClaimAgentNames[0] != "Alpha" ||
		len(issue.CounterpointIDs) != 1 || issue.CounterpointIDs[0] != "pan_demo-alpha-r1-100" {
		t.Fatalf("attribution = %+v", issue)
	}
}

func TestParseVerdictCarriesStructuredIssueReviews(t *testing.T) {
	raw := `{"summary":"结论","recommendation":"建议","issueReviews":[` +
		`{"issueId":"issue-a","outcome":"建议撤回","reason":"证据已补齐","evidence":"pan_stream-alpha-r2-100"},` +
		`{"issueId":"issue-b","outcome":"needs-verification","reason":"先做对照","evidence":"未提供"}]}`
	verdict := parseVerdict(raw, "Chair", 0)
	if len(verdict.IssueReviews) != 2 {
		t.Fatalf("issue reviews = %+v", verdict.IssueReviews)
	}
	if verdict.IssueReviews[0].Outcome != "resolved" ||
		verdict.IssueReviews[1].Outcome != "needs_verification" {
		t.Fatalf("normalized outcomes = %+v", verdict.IssueReviews)
	}
}

func TestValidateIssueReviewsKeepsUnknownAndMissingVisible(t *testing.T) {
	issues := []Issue{{ID: "issue-a"}, {ID: "issue-b"}}
	reviews := []IssueReview{
		{IssueID: "issue-a", Outcome: "resolved", Reason: "已解决"},
		{IssueID: "issue-a", Outcome: "retain", Reason: "重复"},
		{IssueID: "issue-ghost", Outcome: "retain", Reason: "虚构"},
	}
	valid, warnings := validateIssueReviews(reviews, issues)
	if len(valid) != 1 || valid[0].IssueID != "issue-a" {
		t.Fatalf("valid reviews = %+v", valid)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "未附支持证据") ||
		!strings.Contains(joined, "重复审阅分歧 issue-a") ||
		!strings.Contains(joined, "不存在的分歧 ID：issue-ghost") ||
		!strings.Contains(joined, "遗漏了分歧审阅：issue-b") {
		t.Fatalf("warnings = %s", joined)
	}
}

func consensusRunner(signal string) *fakeRunner {
	runner := defaultFakeRunner()
	for id, original := range runner.outputs {
		output := original
		runner.outputs[id] = func(spec commandSpec) (string, error) {
			text, err := output(spec)
			if err == nil && strings.Contains(spec.Stdin, "全员一致检查") {
				// 2026-11-03 V2 P1：自报「一致」须配「剩余分歧：无」——
				// 非无的保留会（正确地）阻止停机。
				residual := "成本边界"
				if signal == "一致" || signal == "yes" {
					residual = "无"
				}
				text += "\n共识状态：" + signal + "\n剩余分歧：" + residual
				if residual != "无" {
					text += "\n分歧依据：当前只有单席估算【未核验】" +
						"\n翻转条件：完成 8 组同任务对照" +
						"\n下一步核验：由 alpha 记录单席与多席同任务成本"
				}
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
	// 2 席×2 轮 + 第 1 轮主席小结 + 汇总 = 6（2026-11-03 主席轮间小结机制）。
	if len(done.Contributions) != 6 {
		t.Fatalf("contributions = %d, want 6", len(done.Contributions))
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
	// 2026-11-03 主席小结插位后按下标取贡献会拿错——按「末轮讨论贡献」定位。
	var lastRoundAlpha *Contribution
	for i := range done.Contributions {
		c := done.Contributions[i]
		if c.AgentID == "alpha" && c.Round == 2 && c.Phase == PhaseDiscussion {
			lastRoundAlpha = &done.Contributions[i]
		}
	}
	if lastRoundAlpha == nil {
		t.Fatal("round-2 alpha contribution missing")
	}
	if strings.Contains(lastRoundAlpha.Content, "共识状态") {
		t.Fatal("machine-readable consensus marker leaked into displayed contribution")
	}
	if lastRoundAlpha.Consensus != "yes" {
		t.Fatalf("contribution consensus = %q, want yes", lastRoundAlpha.Consensus)
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
	// 2 席×3 轮 + r1/r2 两次主席小结 + 汇总 = 9（2026-11-03 主席轮间小结机制）。
	if len(done.Contributions) != 9 {
		t.Fatalf("contributions = %d, want 9", len(done.Contributions))
	}
	found := false
	for _, warning := range done.Warnings {
		if strings.Contains(warning, "安全上限") &&
			strings.Contains(warning, "未达成全员一致") &&
			strings.Contains(warning, "实质分歧 2 席（alpha、beta）") {
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
	if got.Status != StatusInterrupted || got.Phase != PhaseOpening {
		t.Fatalf("recovered status = %s/%s", got.Status, got.Phase)
	}
	if !strings.Contains(got.Error, "上次运行被应用中断") {
		t.Fatalf("missing interruption reason: %q", got.Error)
	}
	if len(got.Contributions) != 1 || got.Contributions[0].Content != "保留这条发言" {
		t.Fatal("partial contributions were not preserved")
	}
	// 重启不是失败：席位应当落 finished_unverified，而不是被写成"失败"。
	if got.Participants[0].Status != SeatFinishedUnverified || got.Participants[0].Error == "" {
		t.Fatalf("running participant not closed as unverified: %+v", got.Participants[0])
	}
	if got.Participants[0].Evidence == "" || got.Participants[0].Observed.Origin != OriginRecovery {
		t.Fatalf("recovery observation not recorded: %+v", got.Participants[0].Observed)
	}
	// 已收声的席位不许被重启波及。
	if got.Participants[1].Status != SeatFinished {
		t.Fatalf("finished participant was disturbed: %+v", got.Participants[1])
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
		Rounds:   2, // 显式两轮：单轮现已放行，不再隐式抬高
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
	// grok 席位自身：2 轮讨论 + 1 次第 1 轮主席小结 + 1 次协作汇总 = 4 次
	// （2026-11-03 主席轮间小结机制；最后一轮不再小结），全部走注入后端。
	if calls != 4 {
		t.Fatalf("model backend calls = %d, want 4", calls)
	}
}

func TestGrokToolAwareBackendReceivesWorkspaceAndEmitsTrajectory(t *testing.T) {
	backend := &toolTraceGrokBackend{}
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), newGrokAdapter(backend))
	defer m.Close()
	workspace := t.TempDir()
	image := filepath.Join(workspace, "diagram.png")
	if err := os.WriteFile(image, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	session, err := m.Start(StartRequest{
		Topic: "Grok 办公工具", Workspace: workspace,
		Attachments: []Attachment{{Path: image, Name: "diagram.png"}},
		AgentIDs:    []string{"grok", "alpha"}, ChairmanAgentID: "alpha", Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	backend.mu.Lock()
	gotWorkspace := backend.workspace
	gotReadPaths := append([]string(nil), backend.readPaths...)
	backend.mu.Unlock()
	if gotWorkspace != filepath.Clean(workspace) {
		t.Fatalf("Grok workspace = %q, want %q", gotWorkspace, filepath.Clean(workspace))
	}
	// 2026-11-03 论证文件化：附件图路径之外，归档发言全文路径进 readPaths
	// 白名单。捕获的是最后一次 grok 调用（其自身贡献尚未入库），汇总席由
	// 主席 alpha 承担，因此排除 grok 最后一条讨论贡献与汇总本身。
	if len(gotReadPaths) == 0 || gotReadPaths[0] != filepath.Clean(image) {
		t.Fatalf("Grok image read roots = %v, want first %q", gotReadPaths, filepath.Clean(image))
	}
	tr, err := newTranscript(m.root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	var lastGrokDiscourse string
	for _, c := range done.Contributions {
		if c.AgentID == "grok" && c.Phase != PhaseSynthesis {
			lastGrokDiscourse = c.ID
		}
	}
	for _, c := range done.Contributions {
		if c.Phase == PhaseSynthesis || c.ID == lastGrokDiscourse {
			continue
		}
		want := tr.contributionPath(c.ID)
		found := false
		for _, p := range gotReadPaths {
			if p == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("archived contribution %s missing from Grok read paths:\n%v", c.ID, gotReadPaths)
		}
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

// 2026-11-08 众神殿结论第三步（写面拆分）：codex 子进程 cwd 落每场专属
// seatwork 目录（真实创建、按席位隔离），共享工作区不再是可写面；他席仍用
// 共享工作区。
func TestCodexSeatWorkDirConfinesWrites(t *testing.T) {
	workspace := t.TempDir()
	codexPath := codexOverride(t)
	runner := defaultFakeRunner()
	var codexDir, alphaDir string
	runner.outputs[codexPath] = func(spec commandSpec) (string, error) {
		codexDir = spec.Dir
		return fakeOutputFor("codex", spec.Stdin), nil
	}
	runner.outputs["alpha"] = func(spec commandSpec) (string, error) {
		alphaDir = spec.Dir
		return fakeOutputFor("alpha", spec.Stdin), nil
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), newCodexAdapter(codexPath))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "写面拆分", AgentIDs: []string{"alpha", "codex"}, ChairmanAgentID: "alpha",
		Workspace: workspace, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if findContribution(done, "codex") == nil {
		t.Fatal("codex contribution missing")
	}
	if codexDir == "" {
		t.Fatal("codex spec never captured")
	}
	if !strings.HasSuffix(filepath.Clean(codexDir), filepath.Join("seatwork", "codex")) {
		t.Fatalf("codex cwd = %q, want <root>/sessions/<id>/seatwork/codex", codexDir)
	}
	if !strings.Contains(filepath.Clean(codexDir), session.ID) {
		t.Fatalf("codex cwd %q must be session-scoped (%s)", codexDir, session.ID)
	}
	if _, err := os.Stat(codexDir); err != nil {
		t.Fatalf("seatwork dir must be created before dispatch: %v", err)
	}
	if filepath.Clean(codexDir) == filepath.Clean(workspace) {
		t.Fatal("shared workspace must no longer be codex's writable cwd")
	}
	if alphaDir == "" || filepath.Clean(alphaDir) != filepath.Clean(workspace) {
		t.Fatalf("non-seat-dir seats must keep the shared workspace, alpha cwd = %q", alphaDir)
	}
}
