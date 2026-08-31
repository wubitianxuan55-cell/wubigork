package session

// 3.0 Step 1 运行时接线：事件日志模式下 Save→Load 往返、legacy 回退、
// 用户/system 消息落日志、LastLogSeq 游标。会话层未显式 SetLogFormat 时
// 保持 legacy 行为（配置层缺省 event 由 boot/宿主注入，见 config.EffectiveLogFormat）。

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// 验收 3.1：event 模式 Save → Load 往返一致（Save 双写：legacy 镜像 +
// 事件日志；Load 优先 Restore 日志投影）。
func TestEventModeSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")

	s := New("sys")
	s.SetLogFormat("event")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "u1"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "a1",
		ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read_file", Arguments: "{}"}}})
	s.Add(provider.Message{Role: provider.RoleTool, Content: "r1", ToolCallID: "c1", Name: "read_file"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "done"})

	if err := s.Save(path); err != nil {
		t.Fatalf("event Save: %v", err)
	}
	// 双写：legacy 镜像 + 事件日志
	if !HasEventLog(path) {
		t.Fatal("event Save 未产生事件日志")
	}
	entries, err := ReadLog(LogPathFor(path))
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 8 {
		t.Fatalf("entries = %d, want 8（system+回合边界+合成 header+user+assistant+tool+assistant+turn_done）", len(entries))
	}

	// 事件日志模式 Load → Restore（checkpoint 无 → 全量重放）
	loaded, err := LoadWithFormat(path, "event")
	if err != nil {
		t.Fatalf("LoadWithFormat: %v", err)
	}
	if !loaded.IsEventMode() {
		t.Fatal("LoadedWithFormat(event) 应处于事件日志模式")
	}
	if loaded.LogSeq() != int64(len(entries)) {
		t.Fatalf("log seq = %d, want %d", loaded.LogSeq(), len(entries))
	}
	want := s.Snapshot()
	if got := loaded.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("event 往返不一致:\n got: %+v\nwant: %+v", got, want)
	}

	// legacy Load 仍整文件读回（镜像与旧行为逐字节一致）
	legacy, err := Load(path)
	if err != nil {
		t.Fatalf("legacy Load: %v", err)
	}
	if got := legacy.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy 镜像不一致:\n got: %+v\nwant: %+v", got, want)
	}

	// 缺省（未 SetLogFormat）Save 不产生事件日志：legacy 行为不变
	s2 := New("")
	s2.Add(provider.Message{Role: provider.RoleUser, Content: "x"})
	p2 := filepath.Join(dir, "legacy.jsonl")
	if err := s2.Save(p2); err != nil {
		t.Fatalf("legacy Save: %v", err)
	}
	if HasEventLog(p2) {
		t.Fatal("legacy 模式 Save 不应产生事件日志")
	}
}

// 验收 4：无日志时 LoadWithFormat("event") 回退 legacy（行为与 Load 一致）。
func TestLoadWithFormatFallsBackToLegacy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.jsonl")
	s := New("")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "u"})
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadWithFormat(path, "event")
	if err != nil {
		t.Fatalf("LoadWithFormat(event) 无日志应回退 legacy: %v", err)
	}
	if loaded.IsEventMode() {
		t.Fatal("无日志时应回退 legacy（非事件模式）")
	}
	if got := loaded.Snapshot(); len(got) != 1 || got[0].Content != "u" {
		t.Fatalf("回退结果 = %+v", got)
	}
	// 缺省格式 = legacy
	loaded2, err := LoadWithFormat(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if loaded2.IsEventMode() {
		t.Fatal("缺省格式不应是事件模式")
	}
	// 文件不存在 → os.IsNotExist 语义保留
	if _, err := LoadWithFormat(filepath.Join(dir, "missing.jsonl"), "event"); err == nil {
		t.Fatal("缺失文件应返回错误（IsNotExist）")
	}
}

// AppendUserMessage/AppendSystemMessage + LastLogSeq 游标（检查点 flush 依赖）。
func TestAppendMessageHelpersAndLastLogSeq(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")

	seq, err := AppendUserMessage(logPath, "", "", "你好")
	if err != nil || seq != 1 {
		t.Fatalf("AppendUserMessage = (%d, %v), want (1, nil)", seq, err)
	}
	seq, err = AppendSystemMessage(logPath, "", "", "中断摘要")
	if err != nil || seq != 2 {
		t.Fatalf("AppendSystemMessage = (%d, %v), want (2, nil)", seq, err)
	}
	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Kind != KindUserMessage || entries[1].Kind != KindSystemMessage {
		t.Fatalf("entries = %+v", entries)
	}
	if LastLogSeq(logPath) != 2 {
		t.Fatalf("LastLogSeq = %d, want 2", LastLogSeq(logPath))
	}
	if LastLogSeq(filepath.Join(dir, "nope.jsonl")) != 0 {
		t.Fatal("缺失日志 LastLogSeq 应为 0")
	}
	// 追加后重开续 seq（写入器单点保证）
	seq, err = AppendUserMessage(logPath, "", "", "第二条")
	if err != nil || seq != 3 {
		t.Fatalf("reopen append = (%d, %v), want (3, nil)", seq, err)
	}
	if LastLogSeq(logPath) != 3 {
		t.Fatalf("LastLogSeq after reopen = %d, want 3", LastLogSeq(logPath))
	}
}
