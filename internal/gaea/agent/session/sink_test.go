package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// 事件 sink：有会话路径时把事件落日志
func TestEventLogSinkWritesLog(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")
	logPath := LogPathFor(sessionPath)

	var got []event.Event
	sink := NewEventLogSink(dir, event.FuncSink(func(e event.Event) { got = append(got, e) }))
	sink.SetPathSource(func() string { return sessionPath })

	sink.Emit(event.Event{Kind: event.TurnStarted})
	sink.Emit(event.Event{Kind: event.Text, Text: "hi"})
	sink.Emit(event.Event{Kind: event.TurnDone})
	sink.Close()

	// 前端 sink 收到全部事件
	if len(got) != 3 {
		t.Fatalf("inner events = %d, want 3", len(got))
	}

	// 日志已落盘
	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("log entries = %d, want 3", len(entries))
	}
	wantKinds := []string{"turn_started", "text", "turn_done"}
	for i, k := range wantKinds {
		if entries[i].Kind != k {
			t.Errorf("entry %d kind = %s, want %s", i, entries[i].Kind, k)
		}
		if entries[i].Seq != int64(i+1) {
			t.Errorf("entry %d seq = %d, want %d", i, entries[i].Seq, i+1)
		}
	}
}

// 无会话路径：只转发不落盘
func TestEventLogSinkNoPathNoWrite(t *testing.T) {
	dir := t.TempDir()
	var got []event.Event
	sink := NewEventLogSink(dir, event.FuncSink(func(e event.Event) { got = append(got, e) }))
	// 未注入 path source
	sink.Emit(event.Event{Kind: event.TurnStarted})
	sink.Close()

	if len(got) != 1 {
		t.Fatalf("inner events = %d, want 1", len(got))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("session dir should stay empty, got %d entries", len(entries))
	}
}

// 会话路径切换：写进新会话的日志
func TestEventLogSinkPathSwitch(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "a.jsonl")
	sink := NewEventLogSink(dir, event.Discard)
	sink.SetPathSource(func() string { return cur })
	sink.Emit(event.Event{Kind: event.TurnStarted})
	// 切换会话
	cur = filepath.Join(dir, "b.jsonl")
	sink.Emit(event.Event{Kind: event.TurnStarted})
	sink.Close()

	entriesA, err := ReadLog(LogPathFor(filepath.Join(dir, "a.jsonl")))
	if err != nil {
		t.Fatal(err)
	}
	entriesB, err := ReadLog(LogPathFor(filepath.Join(dir, "b.jsonl")))
	if err != nil {
		t.Fatal(err)
	}
	if len(entriesA) != 1 || len(entriesB) != 1 {
		t.Fatalf("a=%d b=%d, want 1 each", len(entriesA), len(entriesB))
	}
}
