package pantheon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"testing"
)

type evidenceRunner struct {
	mu    sync.Mutex
	calls int
}

func (r *evidenceRunner) Run(context.Context, commandSpec) (string, error) {
	return "已核对文件与检索结果。\n剩余分歧：无\n置信度：中", nil
}

func (r *evidenceRunner) RunStreamWithTools(
	_ context.Context,
	_ commandSpec,
	onDelta func(string),
	onTool func(ToolTrace),
) (string, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()

	if onDelta != nil {
		onDelta("已核对。")
	}
	onTool(ToolTrace{ID: "t1", Name: "read", Status: "started", Input: "a.go"})
	onTool(ToolTrace{ID: "t1", Name: "read", Status: "completed", Output: "package a"})
	onTool(ToolTrace{ID: "t2", Name: "grep", Status: "started", Input: "cache hit"})
	onTool(ToolTrace{ID: "t2", Name: "grep", Status: "failed", Error: "not found"})
	onTool(ToolTrace{ID: "t3", Name: "grep", Status: "started", Input: "cache hit"})
	onTool(ToolTrace{ID: "t3", Name: "grep", Status: "failed", Error: "not found"})
	return "已核对文件与检索结果。\n剩余分歧：无\n置信度：中", nil
}

func toolEvidenceAdapter(id string) agentAdapter {
	adapter := fakeAdapter(id)
	status := adapter.probe(context.Background())
	status.Capabilities = append(status.Capabilities, "tool-trajectory")
	adapter.probe = func(context.Context) AgentStatus { return status }
	return adapter
}

func findContribution(session *Session, agentID string) *Contribution {
	for index := range session.Contributions {
		if session.Contributions[index].AgentID == agentID {
			return &session.Contributions[index]
		}
	}
	return nil
}

func TestToolEvidenceCountsAndRepeatFailure(t *testing.T) {
	runner := &evidenceRunner{}
	m := newTestManager(t, runner, toolEvidenceAdapter("alpha"), toolEvidenceAdapter("beta"))
	session, err := m.Start(StartRequest{
		Title: "证据摘要", Topic: "核对工具证据", AgentIDs: []string{"alpha", "beta"},
		ChairmanAgentID: "alpha", Rounds: 1, Protocol: ProtocolIndependent, ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	contribution := findContribution(done, "alpha")
	if contribution == nil {
		t.Fatal("alpha contribution missing")
	}
	evidence := contribution.ToolEvidence
	if evidence == nil {
		t.Fatal("tool-trajectory 席位必须产出证据")
	}
	if evidence.Calls != 3 || evidence.OK != 1 || evidence.Failed != 2 {
		t.Fatalf("evidence = %+v, want calls=3 ok=1 failed=2", evidence)
	}
	if evidence.RepeatFailures != 2 || evidence.RepeatTool != "grep" {
		t.Fatalf("repeat = %+v, want grep x2", evidence)
	}
	if len(evidence.Tools) < 2 || evidence.Tools[0].Name != "grep" || evidence.Tools[0].Count != 2 {
		t.Fatalf("tools = %+v, want grep=2 first", evidence.Tools)
	}
}

func TestToolEvidenceNilForUnobservableSeat(t *testing.T) {
	runner := &toolStreamingFakeRunner{fakeRunner: defaultFakeRunner()}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	session, err := m.Start(StartRequest{
		Title: "无轨迹", Topic: "模型席不上报", AgentIDs: []string{"alpha", "beta"},
		ChairmanAgentID: "alpha", Rounds: 1, Protocol: ProtocolIndependent, ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	contribution := findContribution(done, "alpha")
	if contribution == nil {
		t.Fatal("alpha contribution missing")
	}
	if contribution.ToolEvidence != nil {
		t.Fatalf("无 tool-trajectory 能力的席位必须是 nil，got %+v", contribution.ToolEvidence)
	}
}

func TestToolEvidenceLineHonesty(t *testing.T) {
	if got := toolEvidenceLine(nil); got != "该席不上报工具轨迹" {
		t.Fatalf("nil line = %q", got)
	}
	line := toolEvidenceLine(&ToolEvidence{Calls: 3, OK: 1, Failed: 1, RepeatFailures: 2, RepeatTool: "grep"})
	for _, want := range []string{"3 次工具调用", "1 成功", "1 失败", "1 未回报", "同一工具连续失败 2 次：grep"} {
		if !strings.Contains(line, want) {
			t.Fatalf("line %q missing %q", line, want)
		}
	}
}

// 两本账对账线（2026-11-07 轻刀②）：非空计数与指纹只取宿主看到的成功返回；
// 空白输出、失败结果、未回报调用都不能充当「有内容」的证据。
func TestEvidenceNonEmptyAndDigest(t *testing.T) {
	events := []Event{
		{Kind: EventToolStart, ToolID: "r1", ToolName: "read"},
		{Kind: EventToolResult, ToolID: "r1", ToolName: "read", ToolStatus: "completed", ToolOutput: "package a"},
		{Kind: EventToolStart, ToolID: "r2", ToolName: "read"},
		{Kind: EventToolResult, ToolID: "r2", ToolName: "read", ToolStatus: "completed", ToolOutput: "package b"},
		{Kind: EventToolStart, ToolID: "r3", ToolName: "read"},
		{Kind: EventToolResult, ToolID: "r3", ToolName: "read", ToolStatus: "completed", ToolOutput: "  \n\t "},
		{Kind: EventToolStart, ToolID: "w1", ToolName: "web_search"},
		{Kind: EventToolResult, ToolID: "w1", ToolName: "web_search", ToolStatus: "failed", ToolOutput: "partial body", Error: "timeout"},
		{Kind: EventToolStart, ToolID: "g1", ToolName: "grep"},
	}
	evidence := evidenceFromEvents(events)
	if evidence.OK != 3 || evidence.Failed != 1 || evidence.NonEmpty != 2 {
		t.Fatalf("evidence = %+v, want ok=3 failed=1 nonEmpty=2（空白输出与失败结果都不算非空）", evidence)
	}
	byName := map[string]ToolCount{}
	for _, tool := range evidence.Tools {
		byName[tool.Name] = tool
	}
	read := byName["read"]
	if read.Count != 3 || read.OK != 3 || read.NonEmpty != 2 {
		t.Fatalf("read = %+v, want count=3 ok=3 nonEmpty=2", read)
	}
	if read.Digest != toolOutputDigest("package a") {
		t.Fatalf("read digest = %q, want digest of the first non-empty output", read.Digest)
	}
	if len(read.Digest) != 16 || strings.Contains(read.Digest, "package") {
		t.Fatalf("digest must be 16 hex chars and never the raw output, got %q", read.Digest)
	}
	if web := byName["web_search"]; web.OK != 0 || web.NonEmpty != 0 || web.Digest != "" {
		t.Fatalf("failed result must not anchor non-empty evidence: %+v", web)
	}
	if grep := byName["grep"]; grep.Count != 1 || grep.OK != 0 || grep.Digest != "" {
		t.Fatalf("unreported call counts as a call only: %+v", grep)
	}
	// 确定性：同一事件序列回放出同一指纹，存档复核才有意义。
	if again := evidenceFromEvents(events); again.Tools[0].Digest != evidence.Tools[0].Digest {
		t.Fatal("digest must be deterministic across replays")
	}
}

func TestEvidenceNonEmptyNotCappedByToolsLimit(t *testing.T) {
	events := make([]Event, 0, 8)
	for i := 0; i < 8; i++ {
		events = append(events, Event{
			Kind: EventToolResult, ToolID: fmt.Sprintf("t%d", i), ToolName: fmt.Sprintf("tool%d", i),
			ToolStatus: "completed", ToolOutput: "x",
		})
	}
	evidence := evidenceFromEvents(events)
	if len(evidence.Tools) != 6 {
		t.Fatalf("tools list must stay capped at 6, got %d", len(evidence.Tools))
	}
	if evidence.NonEmpty != 8 {
		t.Fatalf("NonEmpty must count every tool, not the capped list; got %d", evidence.NonEmpty)
	}
}

func TestEvidenceMismatchNote(t *testing.T) {
	claim := "【工具核验】已读 a.go：package a"
	cases := []struct {
		name     string
		evidence *ToolEvidence
		content  string
		warn     bool
	}{
		{"通道不上报轨迹时不置一词", nil, claim, false},
		{"有非空返回即可对证", &ToolEvidence{Calls: 1, OK: 1, NonEmpty: 1}, claim, false},
		{"未标注工具核验不对账", &ToolEvidence{Calls: 1, OK: 1}, "【未核验】凭记忆推断", false},
		{"标注了但只有空返回", &ToolEvidence{Calls: 1, OK: 1}, claim, true},
		{"标注了但一次工具都没跑", &ToolEvidence{}, claim, true},
	}
	for _, tc := range cases {
		note := evidenceMismatchNote(tc.evidence, tc.content)
		if (note != "") != tc.warn {
			t.Fatalf("%s: note = %q, want warn=%v", tc.name, note, tc.warn)
		}
		if tc.warn && !strings.Contains(note, "不可对证") {
			t.Fatalf("%s: note must state the fact plainly, got %q", tc.name, note)
		}
	}
}

func TestToolEvidenceLineShowsNonEmpty(t *testing.T) {
	if line := toolEvidenceLine(&ToolEvidence{Calls: 2, OK: 2, NonEmpty: 1}); !strings.Contains(line, "1 次有非空返回") {
		t.Fatalf("line %q missing non-empty count", line)
	}
	if line := toolEvidenceLine(&ToolEvidence{Calls: 1, OK: 1}); strings.Contains(line, "非空") {
		t.Fatalf("zero non-empty must stay silent in the line, got %q", line)
	}
}

// 端到端：ToolOutput 经 recordToolTrace → 事件 → 证据，非空锚落在贡献上。
func TestToolEvidenceNonEmptyWiredEndToEnd(t *testing.T) {
	runner := &evidenceRunner{}
	m := newTestManager(t, runner, toolEvidenceAdapter("alpha"), toolEvidenceAdapter("beta"))
	session, err := m.Start(StartRequest{
		Title: "非空锚", Topic: "核对非空返回", AgentIDs: []string{"alpha", "beta"},
		ChairmanAgentID: "alpha", Rounds: 1, Protocol: ProtocolIndependent, ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	contribution := findContribution(done, "alpha")
	if contribution == nil || contribution.ToolEvidence == nil {
		t.Fatal("alpha evidence missing")
	}
	evidence := contribution.ToolEvidence
	if evidence.NonEmpty != 1 {
		t.Fatalf("NonEmpty = %d, want 1（read 返回 package a，grep 两次失败）", evidence.NonEmpty)
	}
	found := false
	for _, tool := range evidence.Tools {
		if tool.Name == "read" {
			found = true
			if tool.Digest != toolOutputDigest("package a") {
				t.Fatalf("read digest = %q, want digest of the host-observed preview", tool.Digest)
			}
		}
	}
	if !found {
		t.Fatalf("read tool missing from evidence: %+v", evidence.Tools)
	}
}

// emptyToolClaimRunner 的工具「成功」但返回为空，陈词却标注【工具核验】。
type emptyToolClaimRunner struct{}

const emptyToolClaimText = "【工具核验】a.go 第一行是 package a。\n剩余分歧：无\n置信度：中"

func (emptyToolClaimRunner) Run(context.Context, commandSpec) (string, error) {
	return emptyToolClaimText, nil
}

func (emptyToolClaimRunner) RunStreamWithTools(
	_ context.Context,
	_ commandSpec,
	_ func(string),
	onTool func(ToolTrace),
) (string, error) {
	onTool(ToolTrace{ID: "e1", Name: "read", Status: "started", Input: "a.go"})
	onTool(ToolTrace{ID: "e1", Name: "read", Status: "completed", Output: ""})
	return emptyToolClaimText, nil
}

// 端到端：空返回却标注【工具核验】的陈词，定稿里必须带宿主对账警示。
func TestFinalDocumentFlagsUnverifiableToolClaim(t *testing.T) {
	m := newTestManager(t, emptyToolClaimRunner{}, toolEvidenceAdapter("alpha"), toolEvidenceAdapter("beta"))
	session, err := m.Start(StartRequest{
		Title: "对账线", Topic: "空返回的工具核验", AgentIDs: []string{"alpha", "beta"},
		ChairmanAgentID: "alpha", Rounds: 1, Protocol: ProtocolIndependent, ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	contribution := findContribution(done, "alpha")
	if contribution == nil || contribution.ToolEvidence == nil {
		t.Fatal("alpha evidence missing")
	}
	if contribution.ToolEvidence.OK != 1 || contribution.ToolEvidence.NonEmpty != 0 {
		t.Fatalf("evidence = %+v, want ok=1 nonEmpty=0", contribution.ToolEvidence)
	}
	doc := RenderFinalDocument(done)
	if !strings.Contains(doc, "宿主对账：本席标注【工具核验】") {
		t.Fatalf("final document must flag the unverifiable claim:\n%s", doc)
	}
}

// M1 输入侧对账锚：指纹口径与 toolOutputDigest 一致，且确定可复算。
func TestPromptSnapshotDeterministic(t *testing.T) {
	sha, runes := promptSnapshot("你好hello")
	if len(sha) != 16 || runes != 7 {
		t.Fatalf("snapshot = (%q, %d), want 16-hex and 7 runes", sha, runes)
	}
	if strings.Contains(sha, "你好") {
		t.Fatalf("snapshot must never embed the prompt itself: %q", sha)
	}
	full := sha256.Sum256([]byte("你好hello"))
	if want := hex.EncodeToString(full[:])[:16]; sha != want {
		t.Fatalf("snapshot = %q, want sha256 prefix %q", sha, want)
	}
	if again, sameRunes := promptSnapshot("你好hello"); again != sha || sameRunes != runes {
		t.Fatal("snapshot must be deterministic across replays")
	}
}

// 截断语义：超长任务书的锚必须取 clampPrompt 之后的那段——带省略标注、
// 不超预算上限，而不是对被丢掉的原文撒谎。
func TestSeatPromptSnapshotAnchorsPostClamp(t *testing.T) {
	big := strings.Repeat("长", maxPromptRunes+100)
	sha, runes := seatPromptSnapshot(big)
	if runes > maxPromptRunes {
		t.Fatalf("runes = %d, must reflect the clamped prompt (≤ %d)", runes, maxPromptRunes)
	}
	if runes != len([]rune(clampPrompt(big))) {
		t.Fatalf("runes = %d, want clamped length %d", runes, len([]rune(clampPrompt(big))))
	}
	if want, _ := promptSnapshot(clampPrompt(big)); sha != want {
		t.Fatalf("sha = %q, want digest of the clamped prompt %q", sha, want)
	}
}

// snapshotRunner 捕获实际送达的 Stdin，用真实命令面验证「哈希=实发段」。
type snapshotRunner struct {
	mu     sync.Mutex
	stdins []string
}

func (r *snapshotRunner) Run(_ context.Context, spec commandSpec) (string, error) {
	r.mu.Lock()
	r.stdins = append(r.stdins, spec.Stdin)
	r.mu.Unlock()
	return "已核对。\n剩余分歧：无\n置信度：中", nil
}

// 端到端：贡献上的任务书指纹必须等于该席实际收到的任务书指纹。
func TestContributionCarriesPromptSnapshot(t *testing.T) {
	runner := &snapshotRunner{}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	session, err := m.Start(StartRequest{
		Title: "任务书快照", Topic: "对账输入侧", AgentIDs: []string{"alpha", "beta"},
		ChairmanAgentID: "alpha", Rounds: 1, Protocol: ProtocolIndependent, ReadOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	contribution := findContribution(done, "alpha")
	if contribution == nil {
		t.Fatal("alpha contribution missing")
	}
	if contribution.PromptSha256 == "" || contribution.PromptRunes == 0 {
		t.Fatalf("prompt snapshot missing: sha=%q runes=%d", contribution.PromptSha256, contribution.PromptRunes)
	}
	runner.mu.Lock()
	defer runner.mu.Unlock()
	matched := 0
	for _, stdin := range runner.stdins {
		if sha, runes := promptSnapshot(stdin); sha == contribution.PromptSha256 && runes == contribution.PromptRunes {
			matched++
		}
	}
	if matched != 1 {
		t.Fatalf("exactly one delivered prompt must match the contribution anchor, got %d (captured %d)", matched, len(runner.stdins))
	}
}
