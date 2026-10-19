package modelengine

import (
	"testing"
)

// ── 近期调用环形缓冲（运行监测页数据源）─────────────────────────

// TestRecentCalls_NewestFirstAndCap RecordCall 计账同时入环，RecentCalls 新→旧。
func TestRecentCalls_NewestFirstAndCap(t *testing.T) {
	m := NewManager("", "")
	m.RecordCall(ModelCallUsage{EngineID: "xai", Model: "grok-4.7", InputTokens: 10, OutputTokens: 1})
	m.RecordCall(ModelCallUsage{EngineID: "herdsman", Model: "m2", InputTokens: 20, OutputTokens: 2})
	m.RecordCall(ModelCallUsage{EngineID: "tianxuan", Model: "flash", InputTokens: 30, OutputTokens: 3})

	got := m.RecentCalls(2)
	if len(got) != 2 {
		t.Fatalf("RecentCalls(2) 长度 = %d, want 2", len(got))
	}
	if got[0].EngineID != "tianxuan" || got[1].EngineID != "herdsman" {
		t.Errorf("顺序 = [%s %s], want [tianxuan herdsman]（新→旧）", got[0].EngineID, got[1].EngineID)
	}
	all := m.RecentCalls(0)
	if len(all) != 3 {
		t.Errorf("RecentCalls(0) 长度 = %d, want 全部 3", len(all))
	}
}

// TestRecentCalls_RingSlidingOut 超容量滑出最旧（recentCallsCap 条上限）。
func TestRecentCalls_RingSlidingOut(t *testing.T) {
	m := NewManager("", "")
	for i := 0; i < recentCallsCap+10; i++ {
		m.RecordCall(ModelCallUsage{EngineID: "xai", Model: "m", OutputTokens: int64(i)})
	}
	got := m.RecentCalls(0)
	if len(got) != recentCallsCap {
		t.Fatalf("留存 = %d, want %d（滑出最旧）", len(got), recentCallsCap)
	}
	want := int64(recentCallsCap + 9) // 最后一次
	if got[0].OutputTokens != want {
		t.Errorf("最新一条 out_tok = %d, want %d", got[0].OutputTokens, want)
	}
}
