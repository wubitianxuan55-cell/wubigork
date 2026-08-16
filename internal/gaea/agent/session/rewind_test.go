package session

import (
	"path/filepath"
	"testing"
)

// makeTwoTurnLog 写一个两轮会话的真实事件日志（OpenLog+Append，与运行期同构）：
//   轮 0: user「帮我写周报」→ turn_started → tool_dispatch(write_file report.md)
//         → tool_result → assistant_message「完成」→ turn_done
//   轮 1: user「改成英文」→ turn_started → assistant_message「Done」→ turn_done
// 返回日志路径。
func makeTwoTurnLog(t *testing.T, dir string) string {
	t.Helper()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, err := OpenLog(logPath, "")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	defer w.Close()
	appendSeq := func(kind string, payload any) {
		t.Helper()
		if _, err := w.Append(kind, payload); err != nil {
			t.Fatalf("Append %s: %v", kind, err)
		}
	}
	appendSeq(KindUserMessage, userLogPayload{Content: "帮我写周报"})
	appendSeq("turn_started", map[string]any{})
	appendSeq("tool_dispatch", toolCallLogPayload{ID: "c1", Name: "write_file", Args: `{"path":"report.md"}`})
	appendSeq(KindToolResult, toolResultLogPayload{ID: "c1", Name: "write_file", Output: "ok"})
	appendSeq(KindAssistantMessage, assistantLogPayload{Text: "完成"})
	appendSeq("turn_done", map[string]any{})
	appendSeq(KindUserMessage, userLogPayload{Content: "改成英文"})
	appendSeq("turn_started", map[string]any{})
	appendSeq(KindAssistantMessage, assistantLogPayload{Text: "Done"})
	appendSeq("turn_done", map[string]any{})
	return logPath
}

func TestUserTurnRanges(t *testing.T) {
	dir := t.TempDir()
	entries, err := ReadLog(makeTwoTurnLog(t, dir))
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	ranges := UserTurnRanges(entries)
	if len(ranges) != 2 {
		t.Fatalf("UserTurnRanges = %d 个回合, want 2", len(ranges))
	}
	r0 := ranges[0]
	if r0.Turn != 0 || r0.FirstSeq != 1 || r0.LastSeq != 6 || r0.Prompt != "帮我写周报" {
		t.Errorf("range0 = %+v, want {Turn:0 FirstSeq:1 LastSeq:6 Prompt:帮我写周报}", r0)
	}
	if len(r0.Files) != 1 || r0.Files[0] != "report.md" {
		t.Errorf("range0.Files = %v, want [report.md]", r0.Files)
	}
	if r0.Time <= 0 {
		t.Errorf("range0.Time = %d, want >0（turn_started 时间）", r0.Time)
	}
	r1 := ranges[1]
	if r1.Turn != 1 || r1.FirstSeq != 7 || r1.LastSeq != 10 || r1.Prompt != "改成英文" {
		t.Errorf("range1 = %+v, want {Turn:1 FirstSeq:7 LastSeq:10 Prompt:改成英文}", r1)
	}
	if len(r1.Files) != 0 {
		t.Errorf("range1.Files = %v, want 空（该轮无写文件）", r1.Files)
	}
}

func TestUserTurnRangesEmpty(t *testing.T) {
	dir := t.TempDir()
	w, err := OpenLog(filepath.Join(dir, "e.gaea-log.jsonl"), "")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	w.Close()
	entries, err := ReadLog(filepath.Join(dir, "e.gaea-log.jsonl"))
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if got := UserTurnRanges(entries); len(got) != 0 {
		t.Fatalf("空日志 UserTurnRanges = %+v, want 空", got)
	}
}

func TestUserTurnRangesDedupFiles(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, err := OpenLog(logPath, "")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	if _, err := w.Append(KindUserMessage, userLogPayload{Content: "hi"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if _, err := w.Append("tool_dispatch", toolCallLogPayload{ID: "a", Name: "edit_file", Args: `{"path":"a.md"}`}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if _, err := w.Append("tool_dispatch", toolCallLogPayload{ID: "b", Name: "edit_file", Args: `{"path":"a.md"}`}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if _, err := w.Append("tool_dispatch", toolCallLogPayload{ID: "c", Name: "write_file", Args: `{"path":"b.md"}`}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// 非写工具不提取
	if _, err := w.Append("tool_dispatch", toolCallLogPayload{ID: "d", Name: "bash", Args: `{"command":"ls"}`}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	w.Close()

	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	ranges := UserTurnRanges(entries)
	if len(ranges) != 1 {
		t.Fatalf("ranges = %+v, want 1", ranges)
	}
	files := ranges[0].Files
	if len(files) != 2 || files[0] != "a.md" || files[1] != "b.md" {
		t.Errorf("Files = %v, want [a.md b.md]（去重保序）", files)
	}
}

func TestRewindLogTruncates(t *testing.T) {
	dir := t.TempDir()
	logPath := makeTwoTurnLog(t, dir)
	if err := RewindLog(logPath, 6); err != nil {
		t.Fatalf("RewindLog(6): %v", err)
	}
	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 6 {
		t.Fatalf("截断后条目 = %d, want 6", len(entries))
	}
	if last := entries[len(entries)-1]; last.Kind != "turn_done" {
		t.Errorf("最后一条 kind = %s, want turn_done（第 0 轮完整保留）", last.Kind)
	}
	// 再截断到 0 → 清空（空文件，OpenLog 续 seq 从 0）
	if err := RewindLog(logPath, 0); err != nil {
		t.Fatalf("RewindLog(0): %v", err)
	}
	entries2, err := ReadLog(logPath)
	if err != nil {
		t.Fatalf("ReadLog after clear: %v", err)
	}
	if len(entries2) != 0 {
		t.Fatalf("清空后条目 = %d, want 0", len(entries2))
	}
	// 清空后可继续写：seq 从 1 重新计（OpenLog 的 countLogLines 对空文件返回 0）
	w, err := OpenLog(logPath, "")
	if err != nil {
		t.Fatalf("OpenLog after clear: %v", err)
	}
	seq, err := w.Append(KindUserMessage, userLogPayload{Content: "再来"})
	if err != nil {
		t.Fatalf("Append after clear: %v", err)
	}
	if seq != 1 {
		t.Errorf("清空后首条 seq = %d, want 1", seq)
	}
	w.Close()
}

func TestRewindLogMissing(t *testing.T) {
	if err := RewindLog(filepath.Join(t.TempDir(), "nope.gaea-log.jsonl"), 3); err == nil {
		t.Fatal("日志不存在应报错")
	}
}
