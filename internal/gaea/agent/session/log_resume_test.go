package session

// 刀A（v4.245）回合边界续接：干净关闭后凭「seq+文件大小」重开跳过全量
// 修复+逐行解析；外部触碰（追加/截断/删除）一律回落 OpenLog 全量路径，
// 语义与改前逐字节一致。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// 续接快路径：大小未变 ⇒ seq 续上、追加后文件完整可读。
func TestOpenLogResumingFastPath(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")

	w, err := OpenLog(logPath, "", "work")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := w.AppendRaw("text", json.RawMessage(`{"text":"x"}`)); err != nil {
			t.Fatal(err)
		}
	}
	seq := w.Seq()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}

	r, err := OpenLogResuming(logPath, "", "work", seq, st.Size())
	if err != nil {
		t.Fatal(err)
	}
	if r.Seq() != 3 {
		t.Fatalf("续接 seq = %d, want 3", r.Seq())
	}
	if _, err := r.AppendRaw("turn_started", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(entries))
	}
	for i, e := range entries {
		if e.Seq != int64(i+1) {
			t.Errorf("entry %d seq = %d, want %d", i, e.Seq, i+1)
		}
	}
}

// 外部追加 ⇒ 大小不符 ⇒ 回落全量路径，seq 按全部完整行续计。
func TestOpenLogResumingFallbackOnExternalAppend(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")

	w, err := OpenLog(logPath, "", "work")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.AppendRaw("text", json.RawMessage(`{"text":"x"}`)); err != nil {
		t.Fatal(err)
	}
	seq := w.Seq()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// 外部工具在回合间追加一条合法行（设计红线：日志回合间可被外部触碰）。
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	external, _ := json.Marshal(struct {
		Seq     int64           `json:"seq"`
		Ts      int64           `json:"ts"`
		Kind    string          `json:"kind"`
		Payload json.RawMessage `json:"payload"`
	}{2, 1, "notice", json.RawMessage(`{"text":"external"}`)})
	if _, err := f.Write(append(external, '\n')); err != nil {
		t.Fatal(err)
	}
	f.Close()

	r, err := OpenLogResuming(logPath, "", "work", seq, seq*10) // 大小必然不符
	if err != nil {
		t.Fatal(err)
	}
	if r.Seq() != 2 {
		t.Fatalf("回落 seq = %d, want 2（外部行计入）", r.Seq())
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

// 外部截断（torn-tail）⇒ 回落全量路径 ⇒ 先修复再续计完整行。
func TestOpenLogResumingFallbackOnTruncation(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")

	w, err := OpenLog(logPath, "", "work")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := w.AppendRaw("text", json.RawMessage(`{"text":"x"}`)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	// 模拟崩溃残留：截掉最后一行的一半。
	full, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, full[:len(full)-20], 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := OpenLogResuming(logPath, "", "work", 2, int64(len(full)))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if r.Seq() != 1 {
		t.Fatalf("截断回落 seq = %d, want 1（torn-tail 修复后）", r.Seq())
	}
}

// 文件被删 ⇒ 回落全量路径 ⇒ 新建，seq 归零。
func TestOpenLogResumingFallbackOnDeleted(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	if err := os.WriteFile(logPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}
	r, err := OpenLogResuming(logPath, "", "work", 5, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if r.Seq() != 0 {
		t.Fatalf("删除回落 seq = %d, want 0", r.Seq())
	}
}

// sink 端到端：跨回合 seq 连续，turn_done 后记下续接点。
func TestEventLogSinkResumesAcrossTurns(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")
	logPath := LogPathFor(sessionPath)

	sink := NewEventLogSink(dir, event.Discard)
	sink.SetPathSource(func() string { return sessionPath })

	sink.Emit(event.Event{Kind: event.TurnStarted})
	sink.Emit(event.Event{Kind: event.Text, Text: "第一回合"})
	sink.Emit(event.Event{Kind: event.TurnDone})
	if sink.resumePath != logPath || sink.resumeSeq != 3 || sink.resumeSize == 0 {
		t.Fatalf("续接点未记: path=%q seq=%d size=%d", sink.resumePath, sink.resumeSeq, sink.resumeSize)
	}
	sink.Emit(event.Event{Kind: event.TurnStarted})
	sink.Emit(event.Event{Kind: event.Text, Text: "第二回合"})
	sink.Emit(event.Event{Kind: event.TurnDone})
	sink.Close()

	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Fatalf("entries = %d, want 6", len(entries))
	}
	for i, e := range entries {
		if e.Seq != int64(i+1) {
			t.Errorf("entry %d seq = %d, want %d（跨回合必须连续）", i, e.Seq, i+1)
		}
	}
}

// sink 端到端回落：回合间外部追加合法行后，下一回合回落全量路径仍续对 seq。
func TestEventLogSinkFallbackAfterExternalWrite(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")
	logPath := LogPathFor(sessionPath)

	sink := NewEventLogSink(dir, event.Discard)
	sink.SetPathSource(func() string { return sessionPath })
	sink.Emit(event.Event{Kind: event.TurnStarted})
	sink.Emit(event.Event{Kind: event.TurnDone})

	// 外部追加一条合法行（回合间日志可被外部触碰的设计红线）。
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	external, _ := json.Marshal(struct {
		Seq     int64           `json:"seq"`
		Ts      int64           `json:"ts"`
		Kind    string          `json:"kind"`
		Payload json.RawMessage `json:"payload"`
	}{3, 1, "notice", json.RawMessage(`{}`)})
	if _, err := f.Write(append(external, '\n')); err != nil {
		t.Fatal(err)
	}
	f.Close()

	sink.Emit(event.Event{Kind: event.TurnStarted})
	sink.Emit(event.Event{Kind: event.TurnDone})
	sink.Close()

	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("entries = %d, want 5", len(entries))
	}
	if entries[2].Seq != 3 || entries[4].Seq != 5 {
		t.Fatalf("外部行后 seq 断裂: %+v", entries)
	}
}
