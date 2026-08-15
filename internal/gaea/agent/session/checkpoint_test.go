package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// 检查点往返
func TestCheckpointRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cpPath := filepath.Join(dir, "s.gaea-checkpoint.json")
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "压缩后摘要"},
	}
	if err := WriteCheckpoint(cpPath, 42, msgs); err != nil {
		t.Fatalf("WriteCheckpoint: %v", err)
	}
	cp, err := ReadCheckpoint(cpPath)
	if err != nil {
		t.Fatalf("ReadCheckpoint: %v", err)
	}
	if cp == nil {
		t.Fatal("checkpoint is nil")
	}
	if cp.Seq != 42 {
		t.Errorf("seq = %d, want 42", cp.Seq)
	}
	if len(cp.Messages) != 2 || cp.Messages[1].Content != "压缩后摘要" {
		t.Errorf("messages = %+v", cp.Messages)
	}
}

func TestReadCheckpointMissingOrCorrupt(t *testing.T) {
	dir := t.TempDir()
	// 不存在 → nil,nil
	cp, err := ReadCheckpoint(filepath.Join(dir, "nope.json"))
	if err != nil || cp != nil {
		t.Fatalf("missing: cp=%v err=%v", cp, err)
	}
	// 损坏 → nil,nil（防御，不阻塞恢复）
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{{ not json"), 0o644)
	cp, err = ReadCheckpoint(bad)
	if err != nil || cp != nil {
		t.Fatalf("corrupt: cp=%v err=%v", cp, err)
	}
}

func TestWriteCheckpointEmptyPath(t *testing.T) {
	if err := WriteCheckpoint("", 1, nil); err == nil {
		t.Fatal("expected error for empty path")
	}
}

// 压缩后 checkpoint，恢复 = checkpoint + 从 seq 后重放
func TestRestoreCheckpointPlusTail(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")
	logPath := LogPathFor(sessionPath)
	cpPath := CheckpointPathFor(sessionPath)

	w, err := OpenLog(logPath, "")
	if err != nil {
		t.Fatal(err)
	}
	// turn1: seq 1-4（user + assistant deltas + turn_done）
	w.Append(KindUserMessage, userLogPayload{Content: "u1"})
	w.Append(KindAssistantStarted, assistantLogPayload{ID: "m1"})
	w.Append(KindAssistantDelta, map[string]string{"text": "a1"})
	w.Append("turn_done", map[string]string{})
	// 压缩：checkpoint 落在 seq 4，投影 = system + digest + 原 user
	ckMsgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "<compaction-summary>…"}}
	if err := WriteCheckpoint(cpPath, 4, ckMsgs); err != nil {
		t.Fatal(err)
	}
	// turn2: seq 5-6（user + assistant_message）
	w.Append(KindUserMessage, userLogPayload{Content: "u2"})
	w.Append(KindAssistantMessage, assistantLogPayload{Text: "a2"})
	w.Close()

	msgs, last, err := Restore(cpPath, logPath)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if last != 6 {
		t.Errorf("last seq = %d, want 6", last)
	}
	// checkpoint 消息 + tail 投影（u2, a2）
	if len(msgs) != 4 {
		t.Fatalf("messages = %d, want 4", len(msgs))
	}
	if msgs[0].Content != "sys" || msgs[1].Content != "<compaction-summary>…" {
		t.Errorf("checkpoint part = %+v", msgs[:2])
	}
	if msgs[2].Role != provider.RoleUser || msgs[2].Content != "u2" {
		t.Errorf("tail user = %+v", msgs[2])
	}
	if msgs[3].Role != provider.RoleAssistant || msgs[3].Content != "a2" {
		t.Errorf("tail assistant = %+v", msgs[3])
	}
}

// 无 checkpoint：全量从日志头重放
func TestRestoreNoCheckpoint(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, _ := OpenLog(logPath, "")
	w.Append(KindUserMessage, userLogPayload{Content: "u1"})
	w.Append(KindAssistantMessage, assistantLogPayload{Text: "a1"})
	w.Close()
	msgs, last, err := Restore(filepath.Join(dir, "nope-checkpoint.json"), logPath)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if len(msgs) != 2 || msgs[0].Content != "u1" || msgs[1].Content != "a1" {
		t.Fatalf("messages = %+v", msgs)
	}
	if last != 2 {
		t.Errorf("last = %d, want 2", last)
	}
}

// 无日志：只有 checkpoint（或两者皆无）→ checkpoint 消息本身
func TestRestoreMissingLog(t *testing.T) {
	dir := t.TempDir()
	cpPath := filepath.Join(dir, "s.gaea-checkpoint.json")
	msgs := []provider.Message{{Role: provider.RoleUser, Content: "ck"}}
	WriteCheckpoint(cpPath, 5, msgs)
	got, last, err := Restore(cpPath, filepath.Join(dir, "missing.gaea-log.jsonl"))
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if last != 5 || len(got) != 1 || got[0].Content != "ck" {
		t.Fatalf("got=%+v last=%d", got, last)
	}
}

// Restore 内部对 torn-tail 自动修复：尾行不完整也能恢复
func TestRestoreRepairsTornTail(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, _ := OpenLog(logPath, "")
	w.Append(KindUserMessage, userLogPayload{Content: "u1"})
	w.Close()
	// 模拟崩溃写入的不完整尾行
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString(`{"seq":2,"ts":1,"kind":"tex`)
	f.Close()
	msgs, last, err := Restore(filepath.Join(dir, "nope.json"), logPath)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Content != "u1" {
		t.Fatalf("messages = %+v", msgs)
	}
	if last != 1 {
		t.Errorf("last = %d, want 1 (torn tail excluded)", last)
	}
	// 文件本身也被截断修复
	data, _ := os.ReadFile(logPath)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		t.Error("log file still has torn tail")
	}
}
