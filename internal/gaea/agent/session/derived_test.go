package session

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func usageEntry(t *testing.T, seq int64, p usageLogPayload) LogEntry {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return LogEntry{Seq: seq, Kind: "usage", Payload: b}
}

// 派生标题：首条 user_message
func TestDeriveTitle(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: KindSystemMessage, Payload: mustMarshal(userLogPayload{Content: "sys"})},
		{Seq: 2, Kind: KindUserMessage, Payload: mustMarshal(userLogPayload{Content: "  请帮我调试 auth 模块  "})},
		{Seq: 3, Kind: KindUserMessage, Payload: mustMarshal(userLogPayload{Content: "第二条"})},
	}
	if got := DeriveTitle(entries); got != "请帮我调试 auth 模块" {
		t.Errorf("title = %q", got)
	}
}

func TestDeriveTitleTruncates(t *testing.T) {
	long := strings.Repeat("长", 200)
	entries := []LogEntry{{Seq: 1, Kind: KindUserMessage, Payload: mustMarshal(userLogPayload{Content: long})}}
	got := DeriveTitle(entries)
	if len([]rune(got)) != TitlePreviewMax {
		t.Errorf("title runes = %d, want %d", len([]rune(got)), TitlePreviewMax)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("title should end with …, got %q", got)
	}
}

func TestDeriveTitleEmpty(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: "turn_started", Payload: mustMarshal(map[string]string{})},
		{Seq: 2, Kind: KindUserMessage, Payload: mustMarshal(userLogPayload{Content: "   "})},
	}
	if got := DeriveTitle(entries); got != "" {
		t.Errorf("title = %q, want empty", got)
	}
	if got := DeriveTitle(nil); got != "" {
		t.Errorf("nil title = %q, want empty", got)
	}
}

// 派生统计：usage 事件累加 + 成本
func TestDeriveStatsAccumulates(t *testing.T) {
	entries := []LogEntry{
		usageEntry(t, 1, usageLogPayload{
			PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150,
			CacheHitTokens: 40, CacheMissTokens: 60,
			Input: 1, Output: 2, CacheHitPrice: 0.5, Currency: "¥", Source: "main",
		}),
		usageEntry(t, 2, usageLogPayload{
			PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300,
			CacheHitTokens: 100, CacheMissTokens: 100,
			Input: 1, Output: 2, CacheHitPrice: 0.5, Currency: "¥", Source: "subagent",
		}),
	}
	st := DeriveStats(entries)
	if st.PromptTokens != 300 || st.CompletionTokens != 150 || st.TotalTokens != 450 {
		t.Errorf("tokens = %+v", st)
	}
	if st.CacheHitTokens != 140 || st.CacheMissTokens != 160 {
		t.Errorf("cache = %+v", st)
	}
	if st.UsageCount != 2 {
		t.Errorf("usage count = %d", st.UsageCount)
	}
	if st.Currency != "¥" {
		t.Errorf("currency = %q", st.Currency)
	}
	// 成本：(40*0.5 + 60*1 + 50*2)/1e6 + (100*0.5 + 100*1 + 100*2)/1e6
	want := (40*0.5+60+100)/1e6 + (50+100+200)/1e6
	if math.Abs(st.Cost-want) > 1e-9 {
		t.Errorf("cost = %v, want %v", st.Cost, want)
	}
	// 按 source 拆分
	if math.Abs(st.MainCost-((40*0.5+60+100)/1e6)) > 1e-9 {
		t.Errorf("main cost = %v", st.MainCost)
	}
	if math.Abs(st.SubagentCost-((50+100+200)/1e6)) > 1e-9 {
		t.Errorf("subagent cost = %v", st.SubagentCost)
	}
}

func TestDeriveStatsIgnoresNonUsage(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: KindUserMessage, Payload: mustMarshal(userLogPayload{Content: "x"})},
		{Seq: 2, Kind: "notice", Payload: mustMarshal(map[string]any{})},
	}
	st := DeriveStats(entries)
	if st.UsageCount != 0 || st.TotalTokens != 0 || st.Cost != 0 {
		t.Errorf("stats = %+v", st)
	}
}
