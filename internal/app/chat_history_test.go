package app

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// E 档历史导入导出测试：roundtrip 保真（含 extra.bt 分支字段）、坏包拒绝、
// 导入失败原子性（原数据原样）。
func TestChatHistory_ExportImportRoundtrip(t *testing.T) {
	a := newChatServiceTestAppWithHandler(t, nil)
	a.whisperDataRoot = t.TempDir()
	topic, err := a.ChatTopicCreate("树形话题", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	// 带分支字段与业务 extra 的消息（roundtrip 必须原样保真）
	u1, _ := a.chatStore.AppendMessage(topic.ID, "user", "问题一", `{"bt":{"p":0,"c":0}}`)
	_, _ = a.chatStore.AppendMessage(topic.ID, "assistant", "回答一", `{"bt":{"p":` +
		jsonInt(u1.ID) + `},"answered_by":{"engine":"herdsman"},"stopped":true}`)

	path, err := a.ChatExportHistory()
	if err != nil {
		t.Fatalf("ChatExportHistory: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("导出文件不存在: %v", err)
	}

	// 当前数据 → 换成一份新历史（一个空话题），再导回来验证 roundtrip
	newZip := filepath.Join(t.TempDir(), "mini.zip")
	if err := writeChatHistoryZip(newZip, nil, nil); err != nil {
		t.Fatalf("writeChatHistoryZip: %v", err)
	}
	n, err := a.ChatImportHistory(newZip)
	if err != nil || n != 0 {
		t.Fatalf("导入空历史: n=%d err=%v", n, err)
	}
	if topics, _ := a.chatStore.ListTopics(); len(topics) != 0 {
		t.Fatalf("导入空历史后话题应清零: %d", len(topics))
	}

	n, err = a.ChatImportHistory(path)
	if err != nil {
		t.Fatalf("ChatImportHistory: %v", err)
	}
	if n != 1 {
		t.Fatalf("导入话题数 = %d, want 1", n)
	}
	msgs, err := a.chatStore.ListMessages(topic.ID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("消息数 = %d, want 2", len(msgs))
	}
	if msgs[0].ID != u1.ID {
		t.Errorf("消息 id 未保真: %d, want %d", msgs[0].ID, u1.ID)
	}
	if !extraFlagTrue(msgs[1].Extra, "stopped") {
		t.Errorf("extra 停止标记丢失: %s", msgs[1].Extra)
	}
	var bt struct {
		P int64 `json:"p"`
		C int64 `json:"c"`
	}
	if err := json.Unmarshal([]byte(msgs[0].Extra), &bt); err != nil || bt.P != 0 {
		t.Errorf("extra.bt 保真失败: %s (%v)", msgs[0].Extra, err)
	}
	// 导入后追加消息自增 id 衔接（> 导入内最大 id）
	after, err := a.chatStore.AppendMessage(topic.ID, "user", "导入后续写", "")
	if err != nil {
		t.Fatalf("导入后追加: %v", err)
	}
	maxID := u1.ID
	if msgs[1].ID > maxID {
		maxID = msgs[1].ID
	}
	if after.ID <= maxID {
		t.Errorf("导入后自增 id 未衔接: %d, want > %d", after.ID, maxID)
	}
}

func TestChatHistory_RejectsBadArchive(t *testing.T) {
	a := newChatServiceTestAppWithHandler(t, nil)
	a.whisperDataRoot = t.TempDir()
	topic, err := a.ChatTopicCreate("原有数据", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	_, _ = a.chatStore.AppendMessage(topic.ID, "user", "原有消息", "")

	dir := t.TempDir()
	// 非 zip
	notZip := filepath.Join(dir, "not.zip")
	if err := os.WriteFile(notZip, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ChatImportHistory(notZip); err == nil {
		t.Fatal("非 zip 应拒绝")
	}
	// zip 但缺 manifest
	badZip := filepath.Join(dir, "bad.zip")
	if err := writeChatHistoryZip(badZip, nil, nil); err != nil {
		t.Fatal(err)
	}
	_ = badZip // 该包本身合法（空历史）——缺 manifest 的包单独构造
	missingManifest := filepath.Join(dir, "missing_manifest.zip")
	f, _ := os.Create(missingManifest)
	zw := newZipWriterForTest(f)
	w, _ := zw.Create("topics.json")
	_, _ = w.Write([]byte("[]"))
	_ = zw.Close()
	_ = f.Close()
	if _, err := a.ChatImportHistory(missingManifest); err == nil {
		t.Fatal("缺 manifest 应拒绝")
	}
	// 错误版本
	wrongVersion := filepath.Join(dir, "wrong_version.zip")
	f2, _ := os.Create(wrongVersion)
	zw2 := newZipWriterForTest(f2)
	entry, _ := zw2.Create("manifest.json")
	_, _ = entry.Write([]byte(`{"format":"gaea_chat_history","version":99}`))
	entry2, _ := zw2.Create("topics.json")
	_, _ = entry2.Write([]byte("[]"))
	entry3, _ := zw2.Create("messages.json")
	_, _ = entry3.Write([]byte("[]"))
	_ = zw2.Close()
	_ = f2.Close()
	if _, err := a.ChatImportHistory(wrongVersion); err == nil {
		t.Fatal("错误版本应拒绝")
	}
	// 失败后原数据原样（原子性）
	msgs, _ := a.chatStore.ListMessages(topic.ID)
	if len(msgs) != 1 || msgs[0].Content != "原有消息" {
		t.Fatalf("失败导入动了原数据: %+v", msgs)
	}
}

func jsonInt(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func newZipWriterForTest(f *os.File) *zip.Writer {
	return zip.NewWriter(f)
}
