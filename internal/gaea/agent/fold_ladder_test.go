package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// ─── 刀1a: 压缩救援阶梯 slim 摘要档 ─────────────────────────────────────────

// errThenTextProvider fails the first Stream with a chunk error and replays
// text afterwards, recording every request it saw.
type errThenTextProvider struct {
	errText string
	text    string
	reqs    []provider.Request
}

func (p *errThenTextProvider) Name() string { return "err-then-text" }

func (p *errThenTextProvider) Stream(_ context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.reqs = append(p.reqs, req)
	ch := make(chan provider.Chunk, 2)
	if len(p.reqs) == 1 {
		ch <- provider.Chunk{Type: provider.ChunkError, Err: errors.New(p.errText)}
	} else {
		ch <- provider.Chunk{Type: provider.ChunkText, Text: p.text}
	}
	close(ch)
	return ch, nil
}

func (p *errThenTextProvider) Chat(ctx context.Context, req provider.Request) (*provider.Completion, error) {
	return provider.ChatFromStream(ctx, p, req)
}

// bigFold builds n assistant messages of size chars each.
func bigFold(n, chars int) []provider.Message {
	fold := make([]provider.Message, 0, n)
	for i := 0; i < n; i++ {
		fold = append(fold, provider.Message{
			Role:    provider.RoleAssistant,
			Content: "result " + strings.Repeat("r", chars),
		})
	}
	return fold
}

func TestSlimFoldForSummaryClipsOversizedFold(t *testing.T) {
	fold := bigFold(6, 2000)
	slimmed := slimFoldForSummary(&AgentRunner{}, fold, 600)
	if slimmed == nil {
		t.Fatal("oversized fold must be slimmed")
	}
	if len(slimmed) != len(fold) {
		t.Fatalf("slim must keep the message count: %d vs %d", len(slimmed), len(fold))
	}
	for i, m := range slimmed {
		if !strings.Contains(m.Content, "slimmed") {
			t.Fatalf("message %d must carry the slimmed marker", i)
		}
		if len(m.Content) > slimPerMsgFloorChars+200 {
			t.Fatalf("message %d not actually clipped: %d chars", i, len(m.Content))
		}
	}
	// 预算内折叠区不动（nil = 逐字重放）。
	if again := slimFoldForSummary(&AgentRunner{}, fold, 1<<20); again != nil {
		t.Fatal("fold within budget must stay verbatim (nil)")
	}
}

func TestSlimFoldForSummaryClipsToolArguments(t *testing.T) {
	fold := []provider.Message{{
		Role:      provider.RoleAssistant,
		Content:   "w",
		ToolCalls: []provider.ToolCall{{ID: "t1", Name: "write_file", Arguments: `{"path":"a.txt","content":"` + strings.Repeat("x", 4000) + `"}`}},
	}}
	slimmed := slimFoldForSummary(&AgentRunner{}, fold, 50)
	if slimmed == nil {
		t.Fatal("oversized fold must be slimmed")
	}
	args := slimmed[0].ToolCalls[0].Arguments
	if len(args) > slimCallArgCap+100 {
		t.Fatalf("tool arguments must be capped, got %d chars", len(args))
	}
	if !strings.Contains(args, "slimmed") {
		t.Fatal("clipped arguments must carry the marker")
	}
}

func TestSummarizeSlimRungShrinksOversizedRequest(t *testing.T) {
	s := NewSession("L1 system line")
	fold := bigFold(10, 2000) // ~5k tokens est
	prov := &mockProvider{name: "sum", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: "brief"}}}
	a := &AgentRunner{session: s, sink: event.Discard, prov: prov,
		compaction: CompactionConfig{Window: 6000, Ratio: 0.8}}

	out, err := a.summarize(context.Background(), fold, "", false)
	if err != nil || out != "brief" {
		t.Fatalf("summarize: %v %v", out, err)
	}
	req := prov.lastReq
	if len(req.Messages) != 1+len(fold)+1 {
		t.Fatalf("slim must keep every message: %d", len(req.Messages))
	}
	if req.Messages[0].Content != "L1 system line" {
		t.Fatal("system message must stay verbatim")
	}
	last := req.Messages[len(req.Messages)-1]
	if !strings.Contains(last.Content, "compacting") {
		t.Fatal("instruction must stay verbatim as the last user message")
	}
	marked := 0
	for _, m := range req.Messages[1 : len(req.Messages)-1] {
		if strings.Contains(m.Content, "slimmed") {
			marked++
		}
	}
	if marked == 0 {
		t.Fatal("oversized fold must be slimmed in the request")
	}
}

func TestSummarizeFitsStaysVerbatim(t *testing.T) {
	s := NewSession("L1 system line")
	fold := bigFold(3, 200)
	prov := &mockProvider{name: "sum", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: "brief"}}}
	a := &AgentRunner{session: s, sink: event.Discard, prov: prov,
		compaction: CompactionConfig{Window: 1 << 20, Ratio: 0.8}}

	if _, err := a.summarize(context.Background(), fold, "", false); err != nil {
		t.Fatalf("summarize: %v", err)
	}
	req := prov.lastReq
	for i, m := range fold {
		if req.Messages[1+i].Content != m.Content {
			t.Fatalf("fold message %d must be replayed byte-for-byte when it fits", i)
		}
		if strings.Contains(req.Messages[1+i].Content, "slimmed") {
			t.Fatal("fitting fold must not be slimmed")
		}
	}
}

func TestSummarizeRetryDowngradesToSlimAfterProviderError(t *testing.T) {
	s := NewSession("L1 system line")
	// 估算判 fits（全量档会照发），provider 真实拒绝——重试档必须降 slim。
	fold := bigFold(6, 1400)
	prov := &errThenTextProvider{errText: "400 context_length_exceeded", text: "rescued"}
	a := &AgentRunner{session: s, sink: event.Discard, prov: prov,
		compaction: CompactionConfig{Window: 8000, Ratio: 0.8}}

	out, err := a.summarizeWithRetry(context.Background(), fold, "")
	if err != nil || out != "rescued" {
		t.Fatalf("slim retry must rescue: %q %v", out, err)
	}
	if len(prov.reqs) != 2 {
		t.Fatalf("expected full attempt + slim retry, got %d calls", len(prov.reqs))
	}
	markers := func(req provider.Request) int {
		n := 0
		for _, m := range req.Messages {
			if strings.Contains(m.Content, "slimmed") {
				n++
			}
		}
		return n
	}
	if markers(prov.reqs[0]) != 0 {
		t.Fatal("first attempt must be the verbatim full form")
	}
	if markers(prov.reqs[1]) == 0 {
		t.Fatal("retry after a provider overflow must degrade to the slim form")
	}
}

// ─── 刀1b: 投影截断终级 ─────────────────────────────────────────────────────

// rescueSession builds system + user + n old units (assistant + fat tool
// result) + a small verbatim tail.
func rescueSession(t *testing.T, n, toolChars int) *Session {
	return rescueSession2(t, n, 4, toolChars)
}

// rescueSession2 sizes the assistant and tool halves independently. Fat
// assistant text is the case prune can never touch (it only elides tool
// results) — exactly the content a still-over compaction leaves behind.
func rescueSession2(t *testing.T, n, asstChars, toolChars int) *Session {
	t.Helper()
	s := NewSession("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "do it"})
	for i := 0; i < n; i++ {
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "step " + strings.Repeat("w", asstChars),
			ToolCalls: []provider.ToolCall{{ID: string(rune('a' + i)), Name: "bash", Arguments: `{}`}}})
		s.Add(provider.Message{Role: provider.RoleTool, ToolCallID: string(rune('a' + i)), Name: "bash",
			Content: "output " + strings.Repeat("o", toolChars)})
	}
	s.Add(provider.Message{Role: provider.RoleUser, Content: "recent tail question"})
	return s
}

func TestTruncateRescueElidesOldToolResults(t *testing.T) {
	a := testAgentC(4000, 0.8, 5)
	a.session = rescueSession(t, 8, 2000) // ~16KB of tool output, est ≫ target

	if !a.truncateRescue() {
		t.Fatal("rescue must change an over-target session")
	}
	elided := 0
	tailIntact := false
	for _, m := range a.session.Messages {
		if strings.Contains(m.Content, "dropped by truncation rescue") {
			elided++
		}
		if m.Content == "recent tail question" {
			tailIntact = true
		}
	}
	if elided == 0 {
		t.Fatal("fat old tool results must be elided")
	}
	if !tailIntact {
		t.Fatal("recent tail must survive the rescue")
	}
	if a.EstimateContextTokens() >= int(float64(4000)*0.8) {
		t.Logf("warning: rescue elisions did not reach the target (kept by floors?)")
	}
}

func TestTruncateRescueDropsWholeUnits(t *testing.T) {
	// 结果全被 elide 后仍超 → 整单元丢最老 + marker。
	a := testAgentC(600, 0.8, 5)
	a.session = rescueSession(t, 12, 8000)

	if !a.truncateRescue() {
		t.Fatal("rescue must engage the drop-units rung when elisions are not enough")
	}
	marker := ""
	for _, m := range a.session.Messages {
		if strings.HasPrefix(m.Content, "[earlier conversation truncated") {
			marker = m.Content
		}
	}
	if marker == "" {
		t.Fatal("dropping units must install the truncation marker")
	}
	if !strings.Contains(marker, "messages removed") {
		t.Fatalf("marker must count removed messages: %q", marker)
	}
}

func TestTruncateRescueKeepsErrorResultsWhenPolicySays(t *testing.T) {
	a := testAgentC(4000, 0.8, 5)
	a.keepPolicy = KeepErrors
	a.session = rescueSession(t, 8, 2000)
	// 第一条工具结果改为错误输出——KeepErrors 下截断救援必须豁免。
	a.session.Messages[3].Content = "error: build failed " + strings.Repeat("e", 2000)

	a.truncateRescue()
	if !strings.Contains(a.session.Messages[3].Content, "build failed") {
		t.Fatal("KeepErrors results must survive the truncation rescue")
	}
}

func TestTruncateRescueNoWindowNoRegionNoTarget(t *testing.T) {
	if (&AgentRunner{compaction: CompactionConfig{Window: 0}, sink: event.Discard}).truncateRescue() {
		t.Fatal("no window means no rescue")
	}
	a := testAgentC(4000, 0.8, 5)
	s := NewSession("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	a.session = s
	if a.truncateRescue() {
		t.Fatal("no foldable region means no rescue")
	}
	// 估算已在水位下：即使有区域也不动。
	b := testAgentC(1<<20, 0.8, 5)
	b.session = rescueSession(t, 3, 100)
	if b.truncateRescue() {
		t.Fatal("session under the high-water mark must not be touched")
	}
}

func TestTryOverflowRecoveryFallsToTruncationRescue(t *testing.T) {
	// 压缩落地了（机械摘要推进版本）但估算仍在窗口之上——「窗口太小压
	// 不完」的困境里截断终级接管（上游 rescueOrFail 的 at-or-above-ceiling
	// 语义）。大块 assistant 正文是 prune 永远够不到、compact 尾保护又留
	// 下的内容：rescue 的 target/4 保护区刚好够到它。
	a := testAgentC(1500, 0.8, 5)
	a.session = rescueSession2(t, 8, 3000, 0)
	version := a.session.RewriteVersion()

	if !a.tryOverflowRecovery(context.Background(), errors.New("context_length_exceeded")) {
		t.Fatal("post-compaction over-window session must fall through to the truncation rescue")
	}
	if a.session.RewriteVersion() == version {
		t.Fatal("rescue must advance the rewrite version")
	}
	marker := false
	for _, m := range a.session.Messages {
		if strings.HasPrefix(m.Content, "[earlier conversation truncated") {
			marker = true
		}
	}
	if !marker {
		t.Fatal("rescue must drop units and install the truncation marker")
	}
}
