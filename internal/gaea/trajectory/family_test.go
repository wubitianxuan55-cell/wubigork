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

// 刀11：FamilyAgentRows 成员卡行——源日志直算口径（不吃事件流截断），
// main 在前、子代理按首条消息时间升序；委派/消息/回报/用量四计数各归各位。
func TestFamilyAgentRows(t *testing.T) {
	parent := []session.LogEntry{
		famEntry(1, "user_message", map[string]any{"content": "拆报表"}),
		famEntry(2, "tool_dispatch", map[string]any{"id": "t1", "name": "task", "args": `{"prompt":"拆报表"}`}),
		famEntry(3, "tool_dispatch", map[string]any{"id": "t2", "name": "read_file", "args": `{"path":"a"}`}),
		famEntry(4, "subagent_message", map[string]any{"text": "拆好了", "ref": "sa_1"}),
		famEntry(5, "subagent_message", map[string]any{"text": "也好了", "ref": "sa_2"}),
		famEntry(6, "message", map[string]any{"text": "完成"}),
	}
	subs := map[string][]session.LogEntry{
		// sa_2 首条 Ts 更小（famEntry Ts 随 seq 增）→ 排在 sa_1 前（首条消息时间升序）
		"sa_2": {
			famEntry(5, "user_message", map[string]any{"content": "也来一份"}),
			famEntry(6, "assistant_message", map[string]any{"text": "好"}),
		},
		"sa_1": {
			famEntry(10, "user_message", map[string]any{"content": "拆报表"}),
			famEntry(11, "user_message", map[string]any{"content": "再拆"}),
			famEntry(12, "message", map[string]any{"text": "干了一半"}),
			famEntry(13, "assistant_message", map[string]any{"text": "拆好了"}),
		},
	}
	rows := FamilyAgentRows(parent, subs)
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3: %+v", len(rows), rows)
	}
	main := rows[0]
	if main.Ref != FamilyAgentMain {
		t.Fatalf("first row ref = %q, want main", main.Ref)
	}
	// main：消息 2（user 1+assistant 1；subagent_message 不计入）、委派 1
	// （task 入账，read_file 不算）、回报 0
	if main.Messages != 2 || main.Dispatches != 1 || main.Reports != 0 {
		t.Fatalf("main = %+v, want messages 2 / dispatches 1 / reports 0", main)
	}
	if rows[1].Ref != "sa_2" || rows[2].Ref != "sa_1" {
		t.Fatalf("sub order = %s,%s, want sa_2 first (earlier first-msg Ts)", rows[1].Ref, rows[2].Ref)
	}
	sa1 := rows[2]
	// sa_1：首条 user=委派本体不计消息 → 消息 3（user 再拆+message+assistant）
	if sa1.Dispatches != 1 || sa1.Messages != 3 {
		t.Fatalf("sa_1 = %+v, want dispatches 1 / messages 3", sa1)
	}
	if sa1.Reports != 1 {
		t.Fatalf("sa_1 reports = %d, want 1", sa1.Reports)
	}
}

// 空家族：main 行恒在（零计数），无 panic。
func TestFamilyAgentRowsEmpty(t *testing.T) {
	rows := FamilyAgentRows(nil, nil)
	if len(rows) != 1 || rows[0].Ref != FamilyAgentMain || rows[0].Messages != 0 {
		t.Fatalf("rows = %+v, want single zero main row", rows)
	}
}
