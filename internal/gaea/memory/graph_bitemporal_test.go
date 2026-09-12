package memory

// 投影双时间轴（SchemaV21）测试：双时间轴标注烘进事件节点 Desc（desc 随投影
// 进 mem_graph_nodes 物化，「物化=日志投影」的不变量因此保持）。常态两轴同刻
// 不标注（零噪音）；分歧≥1 秒才透出「发生·记录」；时间轴未盖章（内存构造的
// 事件 RecordedAt=0）也不标注。

import (
	"strings"
	"testing"
	"time"
)

func TestEventDescTimeNote(t *testing.T) {
	now := time.Now().UnixMilli()
	hoursInMs := func(h int64) int64 { return h * int64(time.Hour/time.Millisecond) }

	// 常态：同毫秒写入（At 与 RecordedAt 几毫秒差）不标注
	e := Event{Seq: 1, Op: OpSave, Name: "x", At: now - 3, RecordedAt: now}
	if got := eventDesc(e) + eventTimeNote(e); got != "" {
		t.Errorf("同刻事件 desc = %q, want 空串（零噪音）", got)
	}
	// 分歧：事实在过去（回填形态）→ 标注含两个格式化时间
	e = Event{Seq: 2, Op: OpSave, Name: "x", Desc: "补录的事实", At: now - hoursInMs(72), RecordedAt: now}
	got := eventDesc(e) + eventTimeNote(e)
	if !strings.Contains(got, "补录的事实") || !strings.Contains(got, "发生") || !strings.Contains(got, "记录") {
		t.Errorf("分歧事件 desc = %q, want 原文案+「发生」与「记录」标注", got)
	}
	// 分歧反向：声明未来事实时间同样透出（abs 差，不吃方向）
	e = Event{Seq: 3, Op: OpSave, Name: "x", At: now + hoursInMs(48), RecordedAt: now}
	if got := eventTimeNote(e); got == "" {
		t.Errorf("未来事实时间标注 = 空串, want 非空（abs 差口径）")
	}
	// 阈值边界：恰好一秒内不标注
	e = Event{Seq: 4, Op: OpSave, Name: "x", At: now - 999, RecordedAt: now}
	if got := eventTimeNote(e); got != "" {
		t.Errorf("999ms 差标注 = %q, want 空串", got)
	}
	// 时间轴未盖章（V21 前内存构造口径 RecordedAt=0）不标注
	e = Event{Seq: 5, Op: OpSave, Name: "x", At: now - hoursInMs(72)}
	if got := eventTimeNote(e); got != "" {
		t.Errorf("未盖章事件标注 = %q, want 空串", got)
	}
}

func TestProjectEventsBakesTimeNoteIntoDesc(t *testing.T) {
	now := time.Now().UnixMilli()
	events := []Event{
		{Seq: 1, At: now - 72*int64(time.Hour/time.Millisecond), RecordedAt: now, Op: OpSave, Name: "backfill", Project: "p", Desc: "补录"},
		{Seq: 2, At: now, RecordedAt: now, Op: OpTouch, Name: "backfill", Project: "p"},
	}
	g := ProjectEvents(events)
	byID := map[string]GraphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	if n := byID["ev:1"]; !strings.Contains(n.Desc, "发生") {
		t.Errorf("ev:1 desc = %q, want 含双轴标注", n.Desc)
	}
	if n := byID["ev:2"]; strings.Contains(n.Desc, "记录") {
		t.Errorf("ev:2 desc = %q, want 无标注（同刻常态）", n.Desc)
	}
}
