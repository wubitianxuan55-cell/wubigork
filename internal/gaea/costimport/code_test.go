package costimport

// 条目匹配键导入测试（v4.178.0）：表头「定额编号/编码」列映射 + 编码优先匹配
// （命中覆盖、带码未命中即新增、同标题不同编码不误覆盖）。

import (
	"testing"
	"strings"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
)

func TestMapColumnsCode(t *testing.T) {
	colMap := mapColumns([]string{"定额编号", "项目编码", "名称", "单价", "编号", "备注"})
	if colMap[0] != fieldCode || colMap[1] != fieldCode {
		t.Fatalf("编码列映射 = %v/%v, want fieldCode", colMap[0], colMap[1])
	}
	if colMap[2] != fieldTitle || colMap[3] != fieldPrice {
		t.Fatalf("常规列映射破坏: %v", colMap)
	}
	if _, mapped := colMap[4]; mapped {
		t.Fatalf("「编号」应保持噪声列不映射, got %v", colMap[4])
	}
}

func codeTestStore(t *testing.T) *cost.Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	return cost.Open(gdb)
}

func TestMatchRowsCodePriority(t *testing.T) {
	s := codeTestStore(t)
	if err := s.Save(cost.Entry{Name: "a1-12", Title: "人工挖沟槽土方 一类土", Code: "A1-12", Price: 40}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(cost.Entry{Name: "a1-13", Title: "人工挖沟槽土方 一类土", Code: "A1-13", Price: 55}); err != nil {
		t.Fatal(err)
	}

	rows := MatchRows([]Row{
		// 编码命中（全角形态归一化后命中 A1-12）→ 覆盖提示。
		{Title: "别的写法标题", Code: "ａ1－12", Price: 42},
		// 同标题、编码未命中 → 新增（不做标题兜底，避免跨子目误覆盖）。
		{Title: "人工挖沟槽土方 一类土", Code: "A1-99", Price: 66},
		// 无编码、标题精确命中 → 原标题覆盖语义不变。
		{Title: "人工挖沟槽土方 一类土", Price: 44},
	}, s)

	if rows[0].ExistingName != "a1-12" || rows[0].Skip {
		t.Fatalf("row0 编码匹配 = %q skip=%v", rows[0].ExistingName, rows[0].Skip)
	}
	if rows[0].MatchNote == "" || !containsCodNote(rows[0].MatchNote) {
		t.Fatalf("row0 MatchNote = %q, want 编码命中提示", rows[0].MatchNote)
	}
	if rows[1].ExistingName != "" || rows[1].MatchNote != "新增" {
		t.Fatalf("row1 应新增, got %q/%q", rows[1].ExistingName, rows[1].MatchNote)
	}
	if rows[2].ExistingName != "a1-12" && rows[2].ExistingName != "a1-13" {
		t.Fatalf("row2 标题匹配失效: %q", rows[2].ExistingName)
	}
}

func containsCodNote(s string) bool {
	return strings.HasPrefix(s, "编码命中，将覆盖更新")
}
