package contextview

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// ─── 刀2（v4.542）跨会话洞察纯函数黄金测试 ───

func insightsFold(path string, modTime int64, entries []session.LogEntry) InsightsSessionFold {
	return InsightsSessionFold{
		Path:     path,
		Title:    "T " + path,
		SpaceID:  "work",
		ModTime:  modTime,
		Timeline: FoldTimeline(entries, 1_000_000, 0),
	}
}

// insightRequest 造一笔真实 usage（input/output=1000/1M，cacheHitPrice=0）：
// 请求成本 = (miss+completion)×1000/1e6，可预测。
// 刀2 遗留夹具 bug 修正（完善批定罪）：entry helper 硬编码 Ts=1700000000+seq，
// ts 参数从未进日志——逐日归因全落 2023 年同一天，RangeFilter 恒红且当时被
// tail 截断漏验。此处显式覆写 Ts 让夹具语义与参数一致。
func insightRequest(seq int64, ts int64, turn int, prompt int64) session.LogEntry {
	e := entry(seq, "usage", map[string]any{
		"promptTokens": prompt, "completionTokens": 10,
		"cacheHitTokens": 5, "cacheMissTokens": prompt - 5,
		"input": 1000.0, "output": 1000.0,
		"turn": turn,
	})
	e.Ts = ts
	return e
}

func TestBuildInsightsRangeFilter(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.Local)
	day := func(d int) int64 {
		return time.Date(2026, 11, d, 10, 0, 0, 0, time.Local).Unix()
	}
	// 会话 A：今天 + 40 天前各一笔；会话 B：仅 3 天前一笔。
	fa := insightsFold("a.jsonl", day(12), []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		insightRequest(2, day(12), 1, 100),
		insightRequest(3, day(2), 2, 200),
		entry(4, "turn_done", map[string]any{}),
	})
	fb := insightsFold("b.jsonl", day(9), []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		insightRequest(2, day(9), 1, 50),
		entry(3, "turn_done", map[string]any{}),
	})
	folds := []InsightsSessionFold{fa, fb}

	all := BuildInsights(InsightsAll, now, folds)
	if all.Kpi.Sessions != 2 || all.Kpi.Requests != 3 {
		t.Fatalf("all: sessions=%d requests=%d, want 2/3", all.Kpi.Sessions, all.Kpi.Requests)
	}
	if all.Kpi.Tokens != (100+10)+(200+10)+(50+10) {
		t.Fatalf("all: tokens=%d", all.Kpi.Tokens)
	}
	if all.Kpi.Cost < 0.36 || all.Kpi.Cost > 0.37 {
		t.Fatalf("all: cost=%f, want 0.365", all.Kpi.Cost)
	}
	if len(all.Days) != 3 {
		t.Fatalf("all: days=%d, want 3", len(all.Days))
	}

	seven := BuildInsights(Insights7d, now, folds)
	if seven.Kpi.Sessions != 2 || seven.Kpi.Requests != 2 {
		t.Fatalf("7d: sessions=%d requests=%d, want 2/2（A 的今天笔 + B 的 3 天前笔）", seven.Kpi.Sessions, seven.Kpi.Requests)
	}
	if len(seven.Days) != 2 {
		t.Fatalf("7d: days=%d, want 2", len(seven.Days))
	}

	today := BuildInsights(InsightsToday, now, folds)
	if today.Kpi.Sessions != 1 || today.Kpi.Requests != 1 {
		t.Fatalf("today: sessions=%d requests=%d, want 1/1", today.Kpi.Sessions, today.Kpi.Requests)
	}
}

func TestBuildInsightsEstimatedSplit(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.Local)
	ts := now.Add(-time.Hour).Unix()
	// 旧日志：无 usage（turn_done 估算关闭）——tokens 走估算构成分列。
	fold := insightsFold("old.jsonl", ts, []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("system prompt here")),
		entry(3, "user_message", map[string]any{"content": "hello"}),
		entry(4, "turn_done", map[string]any{}),
	})
	out := BuildInsights(InsightsAll, now, []InsightsSessionFold{fold})
	if out.Kpi.Tokens != 0 {
		t.Fatalf("estimated tokens must not pollute billed Tokens, got %d", out.Kpi.Tokens)
	}
	if out.Kpi.EstimatedTokens == 0 {
		t.Fatal("EstimatedTokens should carry estimate-only sessions")
	}
	if len(out.Sessions) != 1 || !out.Sessions[0].Estimated {
		t.Fatalf("session should be marked estimated: %+v", out.Sessions)
	}
}

func TestBuildInsightsTallies(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.Local)
	ts := now.Add(-time.Hour).Unix()
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys")),
		entry(3, "tool_dispatch", map[string]any{"id": "s1", "name": "run_skill", "args": `{"name":"pdf-toolkit"}`}),
		entry(4, "tool_result", map[string]any{"id": "s1", "name": "run_skill", "output": "body"}),
		entry(5, "tool_dispatch", map[string]any{"id": "t1", "name": "read_file", "args": `{"path":"a"}`}),
		entry(6, "tool_result", map[string]any{"id": "t1", "name": "read_file", "output": "x"}),
		entry(7, "tool_dispatch", map[string]any{"id": "t2", "name": "read_file", "args": `{"path":"b"}`}),
		entry(8, "tool_result", map[string]any{"id": "t2", "name": "read_file", "output": "y"}),
		entry(9, "tool_dispatch", map[string]any{"id": "b1", "name": "bash", "args": `{"command":"FOO=1 /usr/bin/go test ./..."}`}),
		entry(10, "tool_result", map[string]any{"id": "b1", "name": "bash", "output": "ok"}),
		entry(11, "usage", map[string]any{"promptTokens": 100, "completionTokens": 10}),
		entry(12, "turn_done", map[string]any{}),
	}
	out := BuildInsights(InsightsAll, now, []InsightsSessionFold{insightsFold("x.jsonl", ts, entries)})
	if out.Kpi.ToolCalls != 3 || out.Kpi.Skills != 1 {
		t.Fatalf("kpi toolCalls=%d skills=%d, want 3/1", out.Kpi.ToolCalls, out.Kpi.Skills)
	}
	if len(out.Skills) != 1 || out.Skills[0].Name != "pdf-toolkit" {
		t.Fatalf("skills tally: %+v", out.Skills)
	}
	if len(out.Tools) != 2 || out.Tools[0].Name != "read_file" || out.Tools[0].Count != 2 || out.Tools[1].Name != "bash" {
		t.Fatalf("tools tally (bash 也是工具调用): %+v", out.Tools)
	}
	if len(out.Programs) != 1 || out.Programs[0].Name != "go" {
		t.Fatalf("programs tally (env prefix + abs path basename): %+v", out.Programs)
	}
}

func TestBuildInsightsSessionSortAndTurns(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.Local)
	ts := now.Add(-time.Hour).Unix()
	// 两轮（turn 1/2 各两笔请求）→ Turns=2（轮次去重，非请求数）。
	fa := insightsFold("a.jsonl", ts, []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		insightRequest(2, ts, 1, 10),
		insightRequest(3, ts+1, 1, 20),
		insightRequest(4, ts+2, 2, 30),
		entry(5, "turn_done", map[string]any{}),
	})
	fb := insightsFold("b.jsonl", ts, []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		insightRequest(2, ts, 1, 500),
		entry(3, "turn_done", map[string]any{}),
	})
	out := BuildInsights(InsightsAll, now, []InsightsSessionFold{fa, fb})
	if out.Sessions[0].Path != "b.jsonl" {
		t.Fatalf("sessions should sort by tokens desc: %+v", out.Sessions[0])
	}
	if out.Kpi.Turns != 3 { // A 两轮 + B 一轮
		t.Fatalf("kpi turns=%d, want 3", out.Kpi.Turns)
	}
}

func TestInsightsJSONShape(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.Local)
	out := BuildInsights(Insights7d, now, nil)
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, field := range []string{`"days":[]`, `"skills":[]`, `"tools":[]`, `"programs":[]`, `"sessions":[]`} {
		if !strings.Contains(s, field) {
			t.Fatalf("%s should marshal as [], got %s", field, s)
		}
	}
}

// TestBuildInsightsSessionNameLists 校验刀8 会话名称清单（统计行点击过滤
// 的依据）：skills/tools/programs 去重入卡；无对应事件缺省。
func TestBuildInsightsSessionNameLists(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.Local)
	ts := now.Add(-time.Hour).Unix()
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys")),
		entry(3, "tool_dispatch", map[string]any{"id": "s1", "name": "run_skill", "args": `{"name":"pdf-toolkit"}`}),
		entry(4, "tool_result", map[string]any{"id": "s1", "name": "run_skill", "output": "body"}),
		entry(5, "tool_dispatch", map[string]any{"id": "t1", "name": "read_file", "args": `{"path":"a"}`}),
		entry(6, "tool_result", map[string]any{"id": "t1", "name": "read_file", "output": "x"}),
		entry(7, "tool_dispatch", map[string]any{"id": "b1", "name": "bash", "args": `{"command":"FOO=1 /usr/bin/go test ./..."}`}),
		entry(8, "tool_result", map[string]any{"id": "b1", "name": "bash", "output": "ok"}),
		entry(9, "usage", map[string]any{"promptTokens": 100, "completionTokens": 10}),
		entry(10, "turn_done", map[string]any{}),
	}
	out := BuildInsights(InsightsAll, now, []InsightsSessionFold{insightsFold("x.jsonl", ts, entries)})
	sess := out.Sessions[0]
	if len(sess.Skills) != 1 || sess.Skills[0] != "pdf-toolkit" {
		t.Fatalf("Skills list: %v", sess.Skills)
	}
	if len(sess.Tools) != 2 || sess.Tools[0] != "bash" || sess.Tools[1] != "read_file" {
		t.Fatalf("Tools list (字典序): %v", sess.Tools)
	}
	if len(sess.Programs) != 1 || sess.Programs[0] != "go" {
		t.Fatalf("Programs list: %v", sess.Programs)
	}
}
