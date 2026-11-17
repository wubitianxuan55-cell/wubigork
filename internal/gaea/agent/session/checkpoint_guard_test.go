package session

// P6（codex 蒸馏 B3，第四批）持久化守卫验收：「允许滞后、禁止超前」。
// 投影（checkpoint.Seq）可以落后于权威源（日志末 seq）——重放追上即可；
// 不得超前——日志被外部截断/轮转后 checkpoint 声称的覆盖面日志无法佐证，
// 守卫丢弃检查点全量重放（灰度 warn，拍板 2026-11-17）。

import (
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func writeGuardFixture(t *testing.T, logSeqs []int64, checkpointSeq int64, checkpointText string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, err := OpenLog(logPath, "", "work")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	for range logSeqs {
		if _, err := w.Append("user_message", map[string]string{"content": "hi"}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	cpPath := filepath.Join(dir, "s.gaea-checkpoint.json")
	if checkpointSeq >= 0 {
		if err := WriteCheckpoint(cpPath, checkpointSeq, testCheckpointMessages(checkpointText), "work"); err != nil {
			t.Fatalf("WriteCheckpoint: %v", err)
		}
	}
	return cpPath, logPath
}

// 投影超前：checkpoint seq 超过日志末条 → 丢弃检查点、全量重放日志。
func TestRestoreRejectsProjectionAheadOfLog(t *testing.T) {
	cpPath, logPath := writeGuardFixture(t, []int64{1, 2, 3}, 10, "未来内容")
	msgs, last, err := Restore(cpPath, logPath)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if last != 3 {
		t.Errorf("cursor = %d, want 3（以日志为权威源）", last)
	}
	for _, m := range msgs {
		if m.Content == "未来内容" {
			t.Error("超前的 checkpoint 消息混入恢复结果——投影不得超前权威源")
		}
	}
	if len(msgs) == 0 {
		t.Error("全量重放应产出日志内的消息")
	}
}

// 投影滞后（正常态）：checkpoint 落后于日志 → checkpoint + 增量重放。
func TestRestoreAllowsProjectionBehindLog(t *testing.T) {
	cpPath, logPath := writeGuardFixture(t, []int64{1, 2, 3, 4}, 2, "检查点内容")
	msgs, last, err := Restore(cpPath, logPath)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if last != 4 {
		t.Errorf("cursor = %d, want 4", last)
	}
	found := false
	for _, m := range msgs {
		if m.Content == "检查点内容" {
			found = true
		}
	}
	if !found {
		t.Error("滞后 checkpoint 的消息应保留（允许滞后）")
	}
}

// 边界：checkpoint seq == 日志末 seq（齐平，不算超前）→ 正常。
func TestRestoreEqualSeqIsFine(t *testing.T) {
	cpPath, logPath := writeGuardFixture(t, []int64{1, 2, 3}, 3, "齐平")
	_, last, err := Restore(cpPath, logPath)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if last != 3 {
		t.Errorf("cursor = %d, want 3", last)
	}
}

// 无日志仅检查点：现状行为保持（checkpoint 即完整投影）。
func TestRestoreCheckpointOnlyUnchanged(t *testing.T) {
	dir := t.TempDir()
	cpPath := filepath.Join(dir, "s.gaea-checkpoint.json")
	if err := WriteCheckpoint(cpPath, 5, testCheckpointMessages("only"), "work"); err != nil {
		t.Fatalf("WriteCheckpoint: %v", err)
	}
	msgs, last, err := Restore(cpPath, filepath.Join(dir, "missing.gaea-log.jsonl"))
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if last != 5 || len(msgs) == 0 || msgs[0].Content != "only" {
		t.Errorf("checkpoint-only restore 变形: last=%d msgs=%v", last, msgs)
	}
}

func testCheckpointMessages(text string) []provider.Message {
	return []provider.Message{{Role: provider.RoleUser, Content: text}}
}
