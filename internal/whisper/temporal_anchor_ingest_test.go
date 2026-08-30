package whisper

import (
	"strings"
	"testing"
)

// TestMaybeWriteTemporalAnchor_RecurringBirthdayWrites 周期性纪念日（生日）命中
// AnchorRecurring 分支 → 经 sink 写入锚点（写路径接线回归）。
func TestMaybeWriteTemporalAnchor_RecurringBirthdayWrites(t *testing.T) {
	p := NewMemoryIngestPipeline(nil)
	fs := NewFactStore()
	fact := fs.Add(MemoryFact{
		Domain: "user_profile", Subcategory: "BASIC_PROFILE", Subject: "生日",
		Summary: "我的生日是 5 月 20 日", Weight: 1, Confidence: 0.9, SelfRelevance: 0.8,
	})
	var got []TemporalAnchor
	p.maybeWriteTemporalAnchor(IngestTurnArgs{
		UserMsg:            "记住，5 月 20 日是我的生日",
		TemporalAnchorSink: func(a TemporalAnchor) { got = append(got, a) },
	}, fact, EmotionalContext{Valence: 0.4, Intensity: 0.5}, true)

	if len(got) != 1 {
		t.Fatalf("应写入 1 个锚点, got %d", len(got))
	}
	a := got[0]
	if a.AnchorType != AnchorRecurring {
		t.Errorf("类型应为 recurring, got %s", a.AnchorType)
	}
	if len(a.LinkedFactIDs) != 1 || a.LinkedFactIDs[0] != fact.ID {
		t.Errorf("LinkedFactIDs 应指向事实 %s, got %+v", fact.ID, a.LinkedFactIDs)
	}
	if a.AnchorDate == "" || a.Summary == "" {
		t.Errorf("锚点应含日期与摘要: %+v", a)
	}
}

// TestMaybeWriteTemporalAnchor_WeakFactSkips 低权重/无纪念日信号 → 不写锚点。
func TestMaybeWriteTemporalAnchor_WeakFactSkips(t *testing.T) {
	p := NewMemoryIngestPipeline(nil)
	fs := NewFactStore()
	fact := fs.Add(MemoryFact{
		Domain: "user_behavior", Subcategory: "HABIT", Subject: "喝水",
		Summary: "今天喝了三杯水", Weight: 0.4, Confidence: 0.7, SelfRelevance: 0.3,
	})
	written := false
	p.maybeWriteTemporalAnchor(IngestTurnArgs{
		UserMsg:            "今天喝了三杯水",
		TemporalAnchorSink: func(a TemporalAnchor) { written = true },
	}, fact, EmotionalContext{Intensity: 0.2}, true)
	if written {
		t.Fatal("弱事实不应写锚点")
	}
}

// TestAfterTurn_AnchorSinkWired 整条摄入链：LLM 抽取生日事实 → 锚点策略命中 →
// sink 收到 AnchorRecurring 锚点（MemoryWritePayload → IngestTurnArgs 透传回归）。
func TestAfterTurn_AnchorSinkWired(t *testing.T) {
	llm := &mockLlm{chatFunc: func(system, user string) (string, error) {
		return `{"facts":[{"domain":"user_profile","subcategory":"BASIC_PROFILE","subject":"生日","summary":"我的生日是 5 月 20 日","weight":1.0,"confidence":0.9,"selfRelevance":0.8}]}`, nil
	}}
	p := NewMemoryIngestPipeline(llm)
	fs := NewFactStore()
	var got []TemporalAnchor
	p.AfterTurn(IngestTurnArgs{
		SessionID: "s1", TurnIndex: 1,
		UserMsg: "记住，5 月 20 日是我的生日", CompanionMsg: "好的，我记住了",
		L2:        EmotionState{Aff: 50, Sec: 30}, // (|Aff|+|Sec|)/200 = 0.4 ≥ 0.35 触发锚点门槛
		FactStore: fs, TotalTurns: 1,
		TemporalAnchorSink: func(a TemporalAnchor) { got = append(got, a) },
	})
	if len(got) != 1 {
		t.Fatalf("整条链应写入 1 个锚点, got %d", len(got))
	}
	if got[0].AnchorType != AnchorRecurring {
		t.Errorf("应为 recurring, got %s", got[0].AnchorType)
	}
	if !strings.Contains(got[0].Summary, "生日") {
		t.Errorf("摘要应含生日: %q", got[0].Summary)
	}
}

// TestOrchestrator_AddTemporalAnchor 回合外追加锚点（State 内存态）。
func TestOrchestrator_AddTemporalAnchor(t *testing.T) {
	o := &Orchestrator{}
	o.AddTemporalAnchor(TemporalAnchor{ID: "a1"})
	o.AddTemporalAnchor(TemporalAnchor{ID: "a2"})
	if len(o.State.TemporalAnchors) != 2 {
		t.Fatalf("应追加 2 个锚点, got %d", len(o.State.TemporalAnchors))
	}
	if o.State.TemporalAnchors[0].ID != "a1" || o.State.TemporalAnchors[1].ID != "a2" {
		t.Errorf("追加顺序不符: %+v", o.State.TemporalAnchors)
	}
}
