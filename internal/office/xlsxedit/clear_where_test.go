package xlsxedit

// clear_where_test.go — 条件清格（2026-11-26 造价反馈三批）：测算样表的未用明细槽
// 在资源名称/单价/合价列挂着取价公式，交付前按「条件列为空」清掉；条件列填了的行
// 一格不碰。

import (
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func clearWhereBook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "综合单价"); err != nil {
		t.Fatal(err)
	}
	_ = f.SetCellStr("综合单价", "A1", "L01")
	_ = f.SetCellFloat("综合单价", "B1", 300, -1, 64)
	for r, code := range map[int]string{6: "L01", 7: "", 8: "C20", 9: ""} {
		axisCol := strconv.Itoa(r)
		_ = f.SetCellStr("综合单价", "I"+axisCol, code)
		_ = f.SetCellFormula("综合单价", "J"+axisCol, `IFERROR(VLOOKUP(I`+axisCol+`,$A$1:$B$1,2,FALSE()),"")`)
		_ = f.SetCellFormula("综合单价", "K"+axisCol, `IFERROR(VLOOKUP(I`+axisCol+`,$A$1:$B$1,2,FALSE()),"")`)
		if code != "" {
			_ = f.SetCellFloat("综合单价", "L"+axisCol, 0.5, -1, 64)
		}
		_ = f.SetCellFormula("综合单价", "N"+axisCol, "L"+axisCol+"*300")
	}
	return f
}

func TestApplyClearWhereBlankConditionColumn(t *testing.T) {
	f := clearWhereBook(t)
	defer func() { _ = f.Close() }()
	summary, err := applyOne(f, Op{Type: "clear_where", Sheet: "综合单价", Range: "J6:N9",
		Where: &Where{Col: "I"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "I 列 blank 的行") || !strings.Contains(summary, "10 格") {
		t.Fatalf("摘要应说明条件与清格数：%s", summary)
	}
	// 条件列为空的行（7、9）：公式与值都清掉。
	for _, cell := range []string{"J7", "K7", "N7", "J9", "K9", "N9"} {
		if fm, _ := f.GetCellFormula("综合单价", cell); fm != "" {
			t.Errorf("%s 的公式应被清掉，实得 %q", cell, fm)
		}
		if v, _ := f.GetCellValue("综合单价", cell); v != "" {
			t.Errorf("%s 的值应被清掉，实得 %q", cell, v)
		}
	}
	// 条件列填了的行（6、8）：公式与值原样。
	for _, cell := range []string{"J6", "K6", "N6", "J8", "K8", "N8"} {
		if fm, _ := f.GetCellFormula("综合单价", cell); fm == "" {
			t.Errorf("%s 的公式不该被动：%q", cell, fm)
		}
	}
	for cell, want := range map[string]string{"L6": "0.5", "L8": "0.5", "I6": "L01", "I8": "C20"} {
		if v, _ := f.GetCellValue("综合单价", cell); v != want {
			t.Errorf("%s 应原样为 %q，实得 %q", cell, want, v)
		}
	}
}

func TestApplyClearWhereValueConditionAndGuards(t *testing.T) {
	f := clearWhereBook(t)
	defer func() { _ = f.Close() }()
	// 等值条件：只清 I 列等于 L01 的那一行。
	if _, err := applyOne(f, Op{Type: "clear_where", Sheet: "综合单价", Range: "J6:N9",
		Where: &Where{Col: "I", Is: "=", Value: "L01"}}); err != nil {
		t.Fatal(err)
	}
	if fm, _ := f.GetCellFormula("综合单价", "J6"); fm != "" {
		t.Errorf("J6 应被清掉，实得 %q", fm)
	}
	for _, cell := range []string{"J7", "J8", "J9"} {
		if fm, _ := f.GetCellFormula("综合单价", cell); fm == "" {
			t.Errorf("%s 不满足条件，不该被动：%q", cell, fm)
		}
	}
	// 缺条件列 / 不认识的比较符：拒绝。
	for _, op := range []Op{
		{Type: "clear_where", Sheet: "综合单价", Range: "J6:N9"},
		{Type: "clear_where", Sheet: "综合单价", Range: "J6:N9", Where: &Where{Col: "I", Is: "like"}},
	} {
		if _, err := applyOne(f, op); err == nil {
			t.Errorf("%+v 应被拒绝", op)
		}
	}
}
