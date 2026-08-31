package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// 旧格式检测
func TestDetectLegacy(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")

	// 无旧文件 → 非 legacy
	legacy, _, err := DetectLegacy(sessionPath)
	if err != nil || legacy {
		t.Fatalf("no legacy file: legacy=%v err=%v", legacy, err)
	}

	// 有旧文件无日志 → legacy
	s := New("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	s.Save(sessionPath)
	legacy, p, err := DetectLegacy(sessionPath)
	if err != nil || !legacy || p != sessionPath {
		t.Fatalf("legacy: legacy=%v p=%q err=%v", legacy, p, err)
	}

	// 有日志 → 非 legacy
	w, _ := OpenLog(LogPathFor(sessionPath), "", "")
	w.Append(KindUserMessage, userLogPayload{Content: "hi"})
	w.Close()
	legacy, _, err = DetectLegacy(sessionPath)
	if err != nil || legacy {
		t.Fatalf("with log: legacy=%v err=%v", legacy, err)
	}
}

// 迁移：旧 jsonl 读入 → 写新日志，旧文件保留
func TestMigrateLegacyToLog(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")
	logPath := LogPathFor(sessionPath)

	// 造旧格式会话
	s := New("golden system prompt")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "你好"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "好的", ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read_file", Arguments: "{}"}}})
	s.Add(provider.Message{Role: provider.RoleTool, Content: "body", ToolCallID: "c1", Name: "read_file"})
	if err := s.Save(sessionPath); err != nil {
		t.Fatal(err)
	}

	n, err := MigrateLegacyToLog(logPath, sessionPath, "")
	if err != nil {
		t.Fatalf("MigrateLegacyToLog: %v", err)
	}
	if n != 7 {
		t.Fatalf("migrated entries = %d, want 7 (回合边界 + 合成 request_header)", n)
	}

	// 旧文件保留
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatal("legacy session file must be kept")
	}

	// 新日志内容
	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 7 {
		t.Fatalf("log entries = %d, want 7", len(entries))
	}
	wantKinds := []string{KindSystemMessage, "turn_started", KindUserMessage, KindRequestHeader, KindAssistantMessage, KindToolResult, "turn_done"}
	for i, k := range wantKinds {
		if entries[i].Kind != k {
			t.Errorf("entry %d kind = %s, want %s", i, entries[i].Kind, k)
		}
	}
	// 投影往返：迁移后日志投影 == 原消息
	projected := ProjectMessages(entries)
	if len(projected) != 4 || projected[0].Content != "golden system prompt" || projected[3].Content != "body" {
		t.Fatalf("projected = %+v", projected)
	}

	// 追加续 seq：迁移后 OpenLog 的 seq = 7
	w, err := OpenLog(logPath, sessionPath, "")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if w.Seq() != 7 {
		t.Fatalf("seq after migration = %d, want 7", w.Seq())
	}
	seq, _ := w.Append("turn_done", map[string]string{})
	if seq != 8 {
		t.Fatalf("append seq = %d, want 8", seq)
	}
}

// 拒绝重复迁移
func TestMigrateRefusesDoubleMigration(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")
	logPath := LogPathFor(sessionPath)
	s := New("")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	s.Save(sessionPath)
	if _, err := MigrateLegacyToLog(logPath, sessionPath, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateLegacyToLog(logPath, sessionPath, ""); err == nil {
		t.Fatal("expected error on double migration")
	} else if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ReadEntriesFor 看板读端兜底：事件日志优先；缺失时回退 legacy 会话投影
// （含回合边界），纯读不落盘。
func TestReadEntriesForLegacyFallback(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")
	s := New("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "你好"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "好的"})
	if err := s.Save(sessionPath); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadEntriesFor(sessionPath)
	if err != nil {
		t.Fatalf("ReadEntriesFor: %v", err)
	}
	if len(entries) != 6 {
		t.Fatalf("entries = %d, want 6（system + 回合边界 + 合成 header + user + assistant + turn_done）", len(entries))
	}
	// 纯读：不产生事件日志文件
	if _, err := os.Stat(LogPathFor(sessionPath)); !os.IsNotExist(err) {
		t.Fatalf("fallback must not create event log: %v", err)
	}
	// 回合边界存在（轨迹折叠可见）
	if entries[1].Kind != "turn_started" || entries[5].Kind != "turn_done" {
		t.Fatalf("turn boundaries missing: %+v", entries)
	}

	// 事件日志存在时优先读日志
	w, err := OpenLog(LogPathFor(sessionPath), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Append(KindUserMessage, userLogPayload{Content: "新消息"}); err != nil {
		t.Fatal(err)
	}
	w.Close()
	entries, err = ReadEntriesFor(sessionPath)
	if err != nil {
		t.Fatalf("ReadEntriesFor with log: %v", err)
	}
	if len(entries) != 1 || entries[0].Kind != KindUserMessage {
		t.Fatalf("log precedence wrong: %+v", entries)
	}
}

// ReadEntriesFor：会话与日志都不存在 → os.ErrNotExist（调用方按空快照处理）。
func TestReadEntriesForMissing(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadEntriesFor(filepath.Join(dir, "nope.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("err = %v, want os.ErrNotExist", err)
	}
}

// OpenLog 自动迁移（首次保存写新日志）
func TestOpenLogAutoMigrates(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "s.jsonl")
	logPath := LogPathFor(sessionPath)
	s := New("sys")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "迁移我"})
	s.Save(sessionPath)

	w, err := OpenLog(logPath, sessionPath, "")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if w.Seq() != 5 {
		t.Fatalf("seq = %d, want 5 (auto-migrated, 回合边界 + 合成 header)", w.Seq())
	}
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatal("legacy file must be kept after auto-migration")
	}
	entries, _ := ReadLog(logPath)
	if len(entries) != 5 || entries[0].Kind != KindSystemMessage {
		t.Fatalf("auto-migrated entries = %+v", entries)
	}
}
