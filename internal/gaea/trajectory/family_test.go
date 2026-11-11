package trajectory

import (
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func famEntry(seq int64, kind string, payload any) session.LogEntry {
	b, _ := json.Marshal(payload)
	return session.LogEntry{Seq: seq, Ts: 1750000000 + seq, Kind: kind, Payload: b}
}

func TestFoldFamilyEvents(t *testing.T) {
	parent := []session.LogEntry{
		famEntry(1, "user_message", map[string]any{"content": "帮我把报表拆三份"}),
		famEntry(2, "tool_dispatch", map[string]any{"id": "t1", "name": "task", "args": `{"prompt":"拆报表"}`}),
		famEntry(3, "tool_dispatch", map[string]any{"id": "t2", "name": "read_file", "args": `{"path":"a"}`}),
		famEntry(4, "subagent_message", map[string]any{"text": "拆好了：三份都在 docs/", "ref": "sa_1"}),
		famEntry(5, "message", map[string]any{"text": "完成"}),
	}
	subs := map[string][]session.LogEntry{
		"sa_1": {
			famEntry(10, "user_message", map[string]any{"content": "拆报表"}),
			famEntry(11, "message", map[string]any{"text": "干了一半"}),
			famEntry(12, "assistant_message", map[string]any{"text": "拆好了"}),
		},
	}
	ev := FoldFamilyEvents(parent, subs)
	if len(ev) != 7 {
		t.Fatalf("events = %d, want 7（task 派发入流、read_file 不入、双视角产出都在）: %+v", len(ev), ev)
	}
	// 排序：Ts 升序
	for i := 1; i < len(ev); i++ {
		if ev[i].Ts < ev[i-1].Ts {
			t.Fatalf("events not sorted at %d", i)
		}
	}
	// 首条=主会话 user 消息
	if ev[0].Agent != FamilyAgentMain || ev[0].Kind != FamilyKindMessage || ev[0].Role != "user" {
		t.Fatalf("first event: %+v", ev[0])
	}
	// task 派发在流、read_file 不在
	dispatches := 0
	for _, e := range ev {
		if e.Kind == FamilyKindDispatch {
			dispatches++
		}
	}
	if dispatches != 2 { // parent task ×1 + sa 首条 user ×1
		t.Fatalf("dispatches = %d, want 2", dispatches)
	}
	// 回报事件 agent=sa_1 且带 ref
	foundReport := false
	for _, e := range ev {
		if e.Kind == FamilyKindReport {
			foundReport = true
			if e.Agent != "sa_1" || e.Ref != "sa_1" {
				t.Fatalf("report event: %+v", e)
			}
		}
	}
	if !foundReport {
		t.Fatal("report event missing")
	}
	// refs 清单：main 在前 + sa_1
	refs := FamilyEventRefs(ev)
	if len(refs) != 2 || refs[0] != FamilyAgentMain || refs[1] != "sa_1" {
		t.Fatalf("refs = %v", refs)
	}
}

func TestFoldFamilyEventsCapAndEmpty(t *testing.T) {
	if got := FoldFamilyEvents(nil, nil); got == nil || len(got) != 0 {
		t.Fatalf("empty fold should be non-nil empty slice, got %+v", got)
	}
	var parent []session.LogEntry
	for i := 0; i < maxFamilyEvents+10; i++ {
		parent = append(parent, famEntry(int64(i+1), "message", map[string]any{"text": "x"}))
	}
	ev := FoldFamilyEvents(parent, nil)
	if len(ev) != maxFamilyEvents {
		t.Fatalf("capped = %d, want %d", len(ev), maxFamilyEvents)
	}
	// 410 条保最新 400：丢 seq 1..10，首条幸存 seq=11
	if ev[0].Seq != 11 {
		t.Fatalf("cap should drop oldest, first seq = %d want 11", ev[0].Seq)
	}
}
