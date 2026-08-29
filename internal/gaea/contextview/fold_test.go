package contextview

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func entry(seq int64, kind string, payload any) session.LogEntry {
	b, _ := json.Marshal(payload)
	return session.LogEntry{Seq: seq, Ts: 1700000000 + seq, Kind: kind, Payload: b}
}

func headerPayload(system string, tools ...string) any {
	ts := make([]any, 0, len(tools))
	for _, t := range tools {
		ts = append(ts, map[string]any{"name": t, "schema": `{"name":"` + t + `","parameters":{}}`})
	}
	return map[string]any{"system": system, "tools": ts, "window": 1_000_000}
}

func TestFoldEmpty(t *testing.T) {
	tl := FoldTimeline(nil, 1_000_000, 0)
	if !tl.Ok {
		t.Fatal("ok should be true")
	}
	if tl.Window != 1_000_000 {
		t.Fatalf("window = %d, want 1000000", tl.Window)
	}
	if len(tl.Requests) != 0 || len(tl.Nodes) != 0 || len(tl.Events) != 0 {
		t.Fatalf("empty fold should have no requests/nodes/events")
	}
}

// 回归：空会话的四条切片必须序列化成 [] 而不是 null——Go 的 nil 切片序列化
// 为 JSON null，前端 .length / for-of 按数组消费会整页崩（ErrorBoundary 接管）。
func TestFoldEmptySlicesMarshalAsArrays(t *testing.T) {
	cases := map[string]ContextTimeline{
		"fold":       FoldTimeline(nil, 1_000_000, 0),
		"binding":    EmptyTimeline(),
		"retention":  FoldTimeline(nil, 0, 200),
	}
	for name, tl := range cases {
		b, err := json.Marshal(tl)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		s := string(b)
		for _, field := range []string{`"requests":[]`, `"events":[]`, `"nodes":[]`, `"archive":[]`} {
			if !strings.Contains(s, field) {
				t.Fatalf("%s: %s should marshal as [], got %s", name, field, s)
			}
		}
	}
}

func TestFoldSingleRequest(t *testing.T) {
	sys := strings.Repeat("系统提示词内容", 20)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys, "read_file", "bash")),
		entry(3, "user_message", map[string]any{"content": "Referenced context:\n\n# file.md\n这是一段很长的项目资料内容用于注入上下文\n\n请帮我总结这份资料并给出建议"}),
		entry(4, "assistant_message", map[string]any{"text": "好的，我来总结。"}),
		entry(5, "tool_dispatch", map[string]any{"id": "t1", "name": "read_file", "args": `{"path":"a.go"}`, "partial": false}),
		entry(6, "tool_result", map[string]any{"id": "t1", "name": "read_file", "output": "package a 这是一段足够长的工具输出内容", "truncated": false}),
		entry(7, "usage", map[string]any{
			"promptTokens": 400, "completionTokens": 30,
			"cacheHitTokens": 380, "cacheMissTokens": 20,
			"sessionCacheHitTokens": 380, "sessionCacheMissTokens": 20,
			"turn": 1,
		}),
	}
	tl := FoldTimeline(entries, 0, 0)
	if tl.Window != 1_000_000 {
		t.Fatalf("window should come from header, got %d", tl.Window)
	}
	if len(tl.Requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(tl.Requests))
	}
	r := tl.Requests[0]
	if r.Turn != 1 || r.Step != 1 {
		t.Fatalf("turn/step = %d/%d, want 1/1", r.Turn, r.Step)
	}
	if r.PromptTokens != 400 {
		t.Fatalf("promptTokens = %d, want 400", r.PromptTokens)
	}
	if r.BriefUser != "请帮我总结这份资料并给出建议" {
		t.Fatalf("briefUser = %q, want 请帮我总结这份资料并给出建议", r.BriefUser)
	}
	if r.BriefResp != "read_file {\"path\":\"a.go\"}" {
		t.Fatalf("briefResp = %q, want read_file with args", r.BriefResp)
	}
	if tl.Current.System == 0 || tl.Current.Tools == 0 {
		t.Fatalf("system/tools should be estimated from header: %+v", tl.Current)
	}
	if tl.Current.User == 0 || tl.Current.Inject == 0 {
		t.Fatalf("user/inject should be split: %+v", tl.Current)
	}
	if tl.Current.Assistant == 0 || tl.Current.Tool == 0 {
		t.Fatalf("assistant/tool should be present: %+v", tl.Current)
	}
	if tl.Stats.ToolCalls != 1 || tl.Stats.Steps != 1 || tl.Stats.Turns != 1 || tl.Stats.Injects != 1 {
		t.Fatalf("stats wrong: %+v", tl.Stats)
	}
	if tl.Stats.CacheHitPercent != 95 {
		t.Fatalf("cacheHitPercent = %v, want 95", tl.Stats.CacheHitPercent)
	}
	// 六分类节点齐全
	cats := map[string]bool{}
	for _, n := range tl.Nodes {
		cats[n.Cat] = true
	}
	for _, want := range []string{catUser, catInject, catAssistant, catTool} {
		if !cats[want] {
			t.Fatalf("missing node category %q in %+v", want, cats)
		}
	}
}

func TestFoldScalesToActual(t *testing.T) {
	// 估算合计很小，usage 实际 promptTokens 很大 → 等比放大且钳制在 4 倍内。
	sys := strings.Repeat("s", 200)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys)),
		entry(3, "user_message", map[string]any{"content": "hi 一段足够长的用户输入内容"}),
		entry(4, "usage", map[string]any{"promptTokens": 10_000, "completionTokens": 10, "turn": 1}),
	}
	tl := FoldTimeline(entries, 0, 0)
	r := tl.Requests[0]
	if r.Category.Total() <= 0 {
		t.Fatalf("scaled total should be positive: %+v", r.Category)
	}
	if r.Category.Total() > 10_000*2 {
		t.Fatalf("scaled total %d should be clamped near actual", r.Category.Total())
	}
}

func TestFoldLegacyWithoutHeader(t *testing.T) {
	// 旧日志没有 request_header：usage 事件仍生成请求记录（估算分类）。
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "这是一段足够长的用户输入内容"}),
		entry(3, "usage", map[string]any{"promptTokens": 100, "completionTokens": 10, "turn": 1}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(tl.Requests))
	}
	if tl.Requests[0].Category.User == 0 {
		t.Fatalf("legacy request should carry estimated user tokens: %+v", tl.Requests[0].Category)
	}
}

func TestFoldCompaction(t *testing.T) {
	sys := strings.Repeat("s", 200)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload(sys)),
		entry(3, "user_message", map[string]any{"content": "task one 一段足够长的用户输入"}),
		entry(4, "tool_result", map[string]any{"id": "t1", "name": "grep", "output": "many lines of output that will be compacted away later"}),
		entry(5, "usage", map[string]any{"promptTokens": 500, "completionTokens": 20, "turn": 1}),
		entry(6, "compaction_done", map[string]any{"trigger": "ratio", "summary": "summary text", "messages": 5, "archive": []string{"x"}}),
	}
	tl := FoldTimeline(entries, 0, 0)
	if tl.Stats.Compacts != 1 {
		t.Fatalf("compacts = %d, want 1", tl.Stats.Compacts)
	}
	if len(tl.Events) != 1 || tl.Events[0].Kind != "compact" || tl.Events[0].Delta >= 0 {
		t.Fatalf("compact event wrong: %+v", tl.Events)
	}
	// 压缩后 user/tool 节点应进归档、current 回落。
	if len(tl.Archive) == 0 {
		t.Fatal("archive should contain gone nodes")
	}
	if tl.Current.User != 0 || tl.Current.Tool != 0 {
		t.Fatalf("current should drop after compaction: %+v", tl.Current)
	}
	if tl.Current.System == 0 {
		t.Fatalf("system prompt survives compaction: %+v", tl.Current)
	}
	for _, n := range tl.Nodes {
		if n.Gone != nil {
			t.Fatalf("live nodes should not be gone: %+v", n)
		}
	}
}

func TestRetention(t *testing.T) {
	var entries []session.LogEntry
	seq := int64(1)
	for i := 0; i < 5; i++ {
		entries = append(entries, entry(seq, "turn_started", map[string]any{}))
		seq++
		entries = append(entries, entry(seq, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1, "turn": i + 1}))
		seq++
	}
	tl := FoldTimeline(entries, 0, 2)
	if len(tl.Requests) != 2 {
		t.Fatalf("retention: requests = %d, want 2", len(tl.Requests))
	}
	if tl.Requests[0].Turn != 4 {
		t.Fatalf("retention should keep the newest: first turn = %d, want 4", tl.Requests[0].Turn)
	}
}

func TestPruneEvent(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "tool_result", map[string]any{"id": "t1", "name": "bash", "output": "x", "truncated": true}),
	}
	tl := FoldTimeline(entries, 0, 0)
	if tl.Stats.Prunes != 1 {
		t.Fatalf("prunes = %d, want 1", tl.Stats.Prunes)
	}
	if len(tl.Events) != 1 || tl.Events[0].Kind != "prune" {
		t.Fatalf("prune event missing: %+v", tl.Events)
	}
}
