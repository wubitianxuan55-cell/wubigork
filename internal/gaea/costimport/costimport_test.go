package costimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
)

func newTestStore(t *testing.T) *cost.Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	return cost.Open(gdb)
}

func TestParseCSV_MappingAndMatch(t *testing.T) {
	store := newTestStore(t)
	if err := store.Save(cost.Entry{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Source: "市场询价", Status: "现行"}); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	csvPath := filepath.Join(dir, "报价单.csv")
	csv := "序号,材料名称,规格型号,单位,单价(元),供应商,备注\n" +
		"1,HP300 高频液压振动锤,300kW,台班,\"3,200.00\",XX租赁,\n" +
		"2,P.O 42.5 水泥,,吨,480 元,海螺,\n" +
		"3,临时材料,,,,-,\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}

	pv, err := Parse(csvPath, store)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d: %+v", len(pv.Rows), pv.Rows)
	}

	// 第一行：映射 + 价格归一化 + 命中既有条目（覆盖更新）。
	r0 := pv.Rows[0]
	if r0.Title != "HP300 高频液压振动锤" || r0.Spec != "300kW" || r0.Unit != "台班" {
		t.Errorf("row0 mapped wrong: %+v", r0)
	}
	if r0.Price != 3200 {
		t.Errorf("price parse = %v, want 3200", r0.Price)
	}
	if r0.ExistingName != "hp300" || !strings.Contains(r0.MatchNote, "覆盖更新") {
		t.Errorf("expected existing match on hp300, got %+v", r0)
	}
	if r0.Skip {
		t.Errorf("row0 should not be skipped: %+v", r0)
	}

	// 第二行：新增条目。
	if r1 := pv.Rows[1]; r1.MatchNote != "新增" || r1.Price != 480 || r1.Source != "海螺" {
		t.Errorf("row1 wrong: %+v", r1)
	}

	// 第三行：缺单价 → Skip。
	if r2 := pv.Rows[2]; !r2.Skip || !strings.Contains(r2.SkipReason, "单价") {
		t.Errorf("row2 should be skipped: %+v", r2)
	}
}

func TestParseXLSX(t *testing.T) {
	dir := t.TempDir()
	xlsxPath := filepath.Join(dir, "测算表.xlsx")
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "科目")
	_ = f.SetCellValue(sheet, "B1", "单位")
	_ = f.SetCellValue(sheet, "C1", "单价")
	_ = f.SetCellValue(sheet, "D1", "规格")
	_ = f.SetCellValue(sheet, "A2", "挖掘机 220")
	_ = f.SetCellValue(sheet, "B2", "台班")
	_ = f.SetCellValue(sheet, "C2", 2600)
	_ = f.SetCellValue(sheet, "D2", "220kW")
	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatal(err)
	}

	pv, err := Parse(xlsxPath, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(pv.Rows))
	}
	r := pv.Rows[0]
	if r.Title != "挖掘机 220" || r.Unit != "台班" || r.Price != 2600 || r.Spec != "220kW" {
		t.Errorf("xlsx row wrong: %+v", r)
	}
}

func TestRawTableAndSlugConsistency(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.tsv")
	if err := os.WriteFile(p, []byte("名称\t单位\t价格\n水泥\t吨\t480\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cols, rows, err := RawTable(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 3 || len(rows) != 1 || rows[0][0] != "水泥" {
		t.Errorf("RawTable = %v %v", cols, rows)
	}
	// 导入行名称与 cost.SlugName 一致（保证与 cost_save 覆盖同一键）。
	if got := cost.SlugName("P.O 42.5 水泥"); got != "p-o-42-5-水泥" {
		t.Errorf("slug = %q", got)
	}
}
