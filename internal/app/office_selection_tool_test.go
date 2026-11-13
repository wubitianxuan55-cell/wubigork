package app

// office_selection_tool_test.go — 选区上下文的纯逻辑测试。
// execute 的真实路径要 COM + 用户选中，这里只钉「回执 → 给模型看的文本」：
// 未选中/出错要给可执行下一步，选中要给位置 + 原文 + 改法，长文本要截断。

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSelectionReplyNotSelected(t *testing.T) {
	out := formatSelectionReply(officeSelectionReply{Selected: false}, "word")
	if !strings.Contains(out, "没有选区") {
		t.Fatalf("未选中应明说，实际：%s", out)
	}
	if !strings.Contains(out, "office_read") {
		t.Error("未选中时应给出替代动作（office_read 定位）")
	}
}

func TestSelectionReplyError(t *testing.T) {
	out := formatSelectionReply(officeSelectionReply{Error: "no active document"}, "word")
	if !strings.Contains(out, "读不到当前选区") || !strings.Contains(out, "no active document") {
		t.Fatalf("应保留原始错误原因，实际：%s", out)
	}
	if !strings.Contains(out, "选中") {
		t.Error("应提示用户先选中")
	}
}

func TestSelectionReplyWord(t *testing.T) {
	out := formatSelectionReply(officeSelectionReply{
		App: "word", Selected: true, Text: "本公司承诺按期交付。", Start: 1450, End: 1460, Para: 12,
		Before: "前述条款不再适用，", After: "若有变更另行通知。",
	}, "word")
	for _, want := range []string{"第 12 段", "1450-1460", "选中：本公司承诺按期交付。", "上文：…前述条款不再适用，", "下文：若有变更另行通知。…", "replace_text", "expect"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Word 选区文本缺 %q：\n%s", want, out)
		}
	}
}

func TestSelectionReplyExcel(t *testing.T) {
	out := formatSelectionReply(officeSelectionReply{
		App: "excel", Selected: true, Address: "$C$3:$C$9", Sheet: "成本表", Rows: 7, Cols: 1, Text: "单价",
	}, "excel")
	for _, want := range []string{"成本表", "$C$3:$C$9", "7 行 × 1 列", "write_range"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Excel 选区文本缺 %q：\n%s", want, out)
		}
	}
}

func TestSelectionReplyPPT(t *testing.T) {
	out := formatSelectionReply(officeSelectionReply{
		App: "ppt", Selected: true, Slide: 4, Shape: "标题 1", Text: "第三季度复盘", Shapes: 2,
	}, "ppt")
	for _, want := range []string{"第 4 页", "标题 1", "共选中 2 个形状", "set_text"} {
		if !strings.Contains(out, want) {
			t.Fatalf("PPT 选区文本缺 %q：\n%s", want, out)
		}
	}
}

func TestSelectionReplyTruncatesLongText(t *testing.T) {
	long := strings.Repeat("很长的选中内容", 200) // 1400 字
	out := formatSelectionReply(officeSelectionReply{App: "word", Selected: true, Text: long, Para: 1}, "word")
	if !strings.Contains(out, "…") {
		t.Fatal("超长选区文本应截断并带省略号")
	}
	if n := len([]rune(out)); n > 2000 {
		t.Fatalf("截断后仍过长：%d 字", n)
	}
}

func TestSelectionToolContract(t *testing.T) {
	tl := officeComSelectionTool{}
	if tl.Name() != "office_selection" {
		t.Fatalf("工具名不符：%s", tl.Name())
	}
	if !tl.ReadOnly() {
		t.Error("office_selection 必须只读")
	}
	if !json.Valid(tl.Schema()) {
		t.Fatal("schema 非 JSON")
	}
	if strings.TrimSpace(tl.Description()) == "" {
		t.Error("description 不能为空")
	}
}
