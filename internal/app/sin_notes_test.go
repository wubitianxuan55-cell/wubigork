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

// TestSinNotesGetBinding 右栏「设定/大纲」面板的读取绑定（v4.263）：工具侧写 →
// SinNotesGet 读回同源数据；无文档 = 空清单 + 空大纲；未知/非原罪故事 id 拒绝。
func TestSinNotesGetBinding(t *testing.T) {
	a, _ := newSinCastTestApp(t)
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}

	// 空文档：Notes 非 nil（前端按数组渲染，不编造）且大纲为空
	v, err := a.SinNotesGet(story.ID)
	if err != nil {
		t.Fatalf("SinNotesGet(空): %v", err)
	}
	if v.Notes == nil || len(v.Notes) != 0 || v.Outline != "" {
		t.Fatalf("空文档视图 = %+v, want 空便签 + 空大纲", v)
	}

	// 工具侧写入（与 AI 写作同一条路径）→ 面板读回同源
	ctx := context.Background()
	notes := sinNotesTool{topicID: story.ID}
	if _, err := notes.Execute(ctx, json.RawMessage(`{"action":"write","content":"女主叫林晚，地方台记者"}`)); err != nil {
		t.Fatalf("notes write: %v", err)
	}
	outline := sinOutlineTool{topicID: story.ID}
	if _, err := outline.Execute(ctx, json.RawMessage(`{"action":"write","content":"第一章：雨夜\n第二章：旧案"}`)); err != nil {
		t.Fatalf("outline write: %v", err)
	}
	v, err = a.SinNotesGet(story.ID)
	if err != nil {
		t.Fatalf("SinNotesGet: %v", err)
	}
	if len(v.Notes) != 1 || !strings.Contains(v.Notes[0], "林晚") {
		t.Fatalf("便签读回 = %+v", v.Notes)
	}
	if !strings.Contains(v.Outline, "第二章：旧案") {
		t.Fatalf("大纲读回 = %q", v.Outline)
	}

	// 守卫：未知故事 id / 非原罪 id 拒绝
	if _, err := a.SinNotesGet("sin_missing_9"); err == nil {
		t.Error("未知故事 id 应报错")
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

// TestSinNotesSaveBinding 面板编辑保存绑定（v4.266）：基线一致才落盘；他端更新
// 后拒绝（冲突前缀）+ force 覆盖；限长与工具侧同口径（单条/大纲截断、条数上限）；
// 守卫沿用 sinTopicGuard。
func TestSinNotesSaveBinding(t *testing.T) {
	a, _ := newSinCastTestApp(t)
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}

	// 工具侧先写一版（= 用户开始编辑时的基线）
	ctx := context.Background()
	notes := sinNotesTool{topicID: story.ID}
	if _, err := notes.Execute(ctx, json.RawMessage(`{"action":"write","content":"女主叫林晚"}`)); err != nil {
		t.Fatalf("notes write: %v", err)
	}
	outline := sinOutlineTool{topicID: story.ID}
	if _, err := outline.Execute(ctx, json.RawMessage(`{"action":"write","content":"第一章：雨夜"}`)); err != nil {
		t.Fatalf("outline write: %v", err)
	}
	base, err := a.SinNotesGet(story.ID)
	if err != nil {
		t.Fatalf("SinNotesGet: %v", err)
	}
	baseJSON, _ := json.Marshal(base)

	// 正常保存：改大纲 + 改便签
	v, err := a.SinNotesSave(story.ID, string(baseJSON), "第一章：雨夜站台\n第二章：旧案", `["女主叫林晚","顾城是线人"]`, false)
	if err != nil {
		t.Fatalf("SinNotesSave: %v", err)
	}
	if len(v.Notes) != 2 || !strings.Contains(v.Outline, "雨夜站台") {
		t.Fatalf("保存结果 = %+v", v)
	}

	// 基线过期（他端又写了一笔）→ 冲突拒绝，错误带固定前缀；确认后 force 覆盖
	if _, err := notes.Execute(ctx, json.RawMessage(`{"action":"write","content":"他端插入的一条"}`)); err != nil {
		t.Fatalf("他端写: %v", err)
	}
	stale := SinNotesView{Notes: v.Notes, Outline: v.Outline}
	staleJSON, _ := json.Marshal(stale)
	_, err = a.SinNotesSave(story.ID, string(staleJSON), "被覆盖的大纲", `[]`, false)
	if err == nil || !strings.HasPrefix(err.Error(), sinConflictPrefix) {
		t.Fatalf("过期基线应冲突拒绝, got %v", err)
	}
	v, err = a.SinNotesSave(story.ID, string(staleJSON), "被覆盖的大纲", `[]`, true)
	if err != nil {
		t.Fatalf("force Save: %v", err)
	}
	if len(v.Notes) != 0 || v.Outline != "被覆盖的大纲" {
		t.Fatalf("force 结果 = %+v", v)
	}

	// 限长：大纲 4000 / 单条 2000 截断（与工具侧同常量），条数 >200 拒绝
	cur, _ := a.SinNotesGet(story.ID)
	curJSON, _ := json.Marshal(cur)
	v, err = a.SinNotesSave(story.ID, string(curJSON), strings.Repeat("纲", sinOutlineMaxRunes+30), `["`+strings.Repeat("字", sinNoteMaxRunes+30)+`"]`, false)
	if err != nil {
		t.Fatalf("限长 Save: %v", err)
	}
	if len([]rune(v.Outline)) != sinOutlineMaxRunes || len([]rune(v.Notes[0])) != sinNoteMaxRunes {
		t.Errorf("截断口径: outline=%d note=%d", len([]rune(v.Outline)), len([]rune(v.Notes[0])))
	}
	tooMany := make([]string, sinNotesMaxItems+1)
	for i := range tooMany {
		tooMany[i] = "x"
	}
	b, _ := json.Marshal(tooMany)
	if _, err := a.SinNotesSave(story.ID, string(curJSON), "", string(b), false); err == nil || !strings.Contains(err.Error(), "上限") {
		t.Errorf("条数超限应拒绝, got %v", err)
	}

	// 便签列表坏 JSON 拒绝；未知故事 id 走守卫
	if _, err := a.SinNotesSave(story.ID, string(curJSON), "", `{坏`, false); err == nil {
		t.Error("便签列表格式错误应报错")
	}
	if _, err := a.SinNotesSave("sin_missing_9", "{}", "", "[]", false); err == nil {
		t.Error("未知故事 id 应报错")
	}
}
