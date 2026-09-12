package app

// 原罪便签/大纲工具落盘测试：往返矩阵、上限、路径守卫、损坏自愈。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mustSinNotesPath 测试内构造便签路径（id 非法直接失败）。
func mustSinNotesPath(t *testing.T, topicID string) string {
	t.Helper()
	p, err := sinNotesPath(topicID)
	if err != nil {
		t.Fatalf("sinNotesPath(%q): %v", topicID, err)
	}
	return p
}

// TestSinNotesStore 便签往返：list/read/write/set/delete + 序号越界 + 空正文
// 拒绝 + 未知 action + 落盘合法 JSON + 无残留临时文件。
func TestSinNotesStore(t *testing.T) {
	sinTestHome(t)
	ctx := context.Background()
	tl := sinNotesTool{topicID: "sin_1000_1"}

	if out, err := tl.Execute(ctx, json.RawMessage(`{"action":"list"}`)); err != nil || !strings.Contains(out, "还没有便签") {
		t.Fatalf("空便签 list = %q, err=%v", out, err)
	}
	if out, err := tl.Execute(ctx, json.RawMessage(`{"action":"write","content":"女主叫林晚，地方台记者"}`)); err != nil ||
		!strings.Contains(out, "#0") {
		t.Fatalf("write = %q, err=%v", out, err)
	}
	if _, err := tl.Execute(ctx, json.RawMessage(`{"action":"write","content":"顾城是她三年前的线人"}`)); err != nil {
		t.Fatalf("第二次 write: %v", err)
	}
	if out, err := tl.Execute(ctx, json.RawMessage(`{"action":"list"}`)); err != nil || !strings.Contains(out, "#1") {
		t.Fatalf("list = %q, err=%v", out, err)
	}
	if out, err := tl.Execute(ctx, json.RawMessage(`{"action":"read","index":1}`)); err != nil || !strings.Contains(out, "线人") {
		t.Fatalf("read#1 = %q, err=%v", out, err)
	}
	if out, err := tl.Execute(ctx, json.RawMessage(`{"action":"set","index":1,"content":"顾城是她的旧同事"}`)); err != nil ||
		!strings.Contains(out, "已更新便签 #1") {
		t.Fatalf("set = %q, err=%v", out, err)
	}
	if out, err := tl.Execute(ctx, json.RawMessage(`{"action":"read","index":1}`)); err != nil || !strings.Contains(out, "旧同事") {
		t.Fatalf("set 未生效: %q, err=%v", out, err)
	}
	if _, err := tl.Execute(ctx, json.RawMessage(`{"action":"read","index":9}`)); err == nil ||
		!strings.Contains(err.Error(), "0~1") {
		t.Errorf("越界应给人话错误: %v", err)
	}
	if _, err := tl.Execute(ctx, json.RawMessage(`{"action":"write","content":"   "}`)); err == nil {
		t.Error("空正文应被拒")
	}
	if _, err := tl.Execute(ctx, json.RawMessage(`{"action":"unknown"}`)); err == nil {
		t.Error("未知 action 应报错")
	}
	if _, err := tl.Execute(ctx, json.RawMessage(`{"action":""}`)); err == nil {
		t.Error("缺 action 应报错")
	}
	if out, err := tl.Execute(ctx, json.RawMessage(`{"action":"delete","index":0}`)); err != nil ||
		!strings.Contains(out, "剩 1 条") {
		t.Fatalf("delete = %q, err=%v", out, err)
	}

	// 落盘：合法 JSON、版本号在位、无 .tmp 残留（原子写）、目录归属原罪。
	path := mustSinNotesPath(t, "sin_1000_1")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("便签文件未落盘: %v", err)
	}
	var doc sinNotesDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("便签文件不是合法 JSON: %v", err)
	}
	if doc.Version != sinNotesVersion || len(doc.Notes) != 1 {
		t.Errorf("文档内容 = %+v", doc)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("不应残留 .tmp 临时文件")
	}
	if dir := filepath.Dir(path); filepath.Base(dir) != "notes" || !strings.Contains(dir, "sin") {
		t.Errorf("便签应落原罪自有目录, got %s", dir)
	}
}

// TestSinNotesCapsAndOutline 上限与大纲：单条便签截断、大纲读写与清空、
// 清空大纲不动便签（同文件两键互不干扰）。
func TestSinNotesCapsAndOutline(t *testing.T) {
	sinTestHome(t)
	ctx := context.Background()
	tl := sinNotesTool{topicID: "sin_1000_2"}
	long := strings.Repeat("字", sinNoteMaxRunes+50)
	if _, err := tl.Execute(ctx, json.RawMessage(`{"action":"write","content":"`+long+`"}`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	path := mustSinNotesPath(t, "sin_1000_2")
	if doc := loadSinNotes(path); len([]rune(doc.Notes[0])) > sinNoteMaxRunes {
		t.Errorf("单条便签应截断到 %d rune, got %d", sinNoteMaxRunes, len([]rune(doc.Notes[0])))
	}

	outline := sinOutlineTool{topicID: "sin_1000_2"}
	if out, err := outline.Execute(ctx, json.RawMessage(`{"action":"read"}`)); err != nil || !strings.Contains(out, "还没有大纲") {
		t.Fatalf("空大纲 read = %q, err=%v", out, err)
	}
	if out, err := outline.Execute(ctx, json.RawMessage(`{"action":"write","content":"第一章：雨夜\n第二章：旧案"}`)); err != nil ||
		!strings.Contains(out, "已更新故事大纲") {
		t.Fatalf("outline write = %q, err=%v", out, err)
	}
	if out, err := outline.Execute(ctx, json.RawMessage(`{"action":"read"}`)); err != nil || !strings.Contains(out, "第二章：旧案") {
		t.Fatalf("outline read = %q, err=%v", out, err)
	}
	if _, err := outline.Execute(ctx, json.RawMessage(`{"action":"wipe"}`)); err == nil {
		t.Error("未知 outline action 应报错")
	}
	if out, err := outline.Execute(ctx, json.RawMessage(`{"action":"write","content":""}`)); err != nil || !strings.Contains(out, "已清空") {
		t.Fatalf("outline 清空 = %q, err=%v", out, err)
	}
	doc := loadSinNotes(path)
	if len(doc.Notes) != 1 || doc.Outline != "" {
		t.Errorf("清空大纲不应动便签: %+v", doc)
	}
}

// TestSinNotesPathGuard 故事 id 越权（路径分隔/上跳/空/非法字符）必须 fail-closed：
// 不拼出目录外的路径，也不写入。
func TestSinNotesPathGuard(t *testing.T) {
	sinTestHome(t)
	for _, bad := range []string{"", "  ", "../evil", "a/b", `a\b`, "sin.1", ".", "..", "sin 1"} {
		if _, err := sinNotesPath(bad); err == nil {
			t.Errorf("非法故事 id %q 应被拒", bad)
		}
	}
	if _, err := sinNotesPath("sin_1726_1"); err != nil {
		t.Errorf("合法 id 应通过: %v", err)
	}
	tl := sinNotesTool{topicID: "../evil"}
	if _, err := tl.Execute(context.Background(), json.RawMessage(`{"action":"write","content":"x"}`)); err == nil {
		t.Error("越权 topicID 的写入应报错")
	}
}

// TestSinNotesCorruptFileFallsBackToEmpty 损坏文件只当空文档（辅助数据不阻断
// 故事创作），后续写入自愈成合法 JSON。
func TestSinNotesCorruptFileFallsBackToEmpty(t *testing.T) {
	sinTestHome(t)
	path := mustSinNotesPath(t, "sin_1000_3")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("{坏 JSON"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	tl := sinNotesTool{topicID: "sin_1000_3"}
	if out, err := tl.Execute(context.Background(), json.RawMessage(`{"action":"list"}`)); err != nil ||
		!strings.Contains(out, "还没有便签") {
		t.Fatalf("损坏文件应回退空文档: %q, err=%v", out, err)
	}
	if _, err := tl.Execute(context.Background(), json.RawMessage(`{"action":"write","content":"重来"}`)); err != nil {
		t.Fatalf("写入应自愈: %v", err)
	}
	if doc := loadSinNotes(path); len(doc.Notes) != 1 || doc.Notes[0] != "重来" {
		t.Errorf("自愈后文档 = %+v", doc)
	}
}
