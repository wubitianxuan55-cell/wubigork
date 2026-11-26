package xlsxedit

// audit_empty_smoke_test.go — 「空串结果 ≠ 未重算」的实弹钉，默认跳过：
//
//	GAEA_SMOKE_RECALC=1 go test ./internal/office/xlsxedit -run TestSmokeRecalcEmpty -v
//
// 测算样表的未用明细槽挂着 =IFERROR(VLOOKUP(空键),"")：LibreOffice 重算后结果
// 是空串，此前审计一律记「未重算」，交付门把 480 个空槽判成「未达」（2026-11-26
// 造价反馈三批）。本用例拿真 LibreOffice 的产物钉住两种形态的区分。

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func auditKinds(t *testing.T, path string) map[string]string {
	t.Helper()
	rep, err := Audit(path)
	if err != nil {
		t.Fatalf("审计失败：%v", err)
	}
	out := map[string]string{}
	for _, f := range rep.Findings {
		out[f.Cell] = f.Kind
	}
	return out
}

func TestSmokeRecalcEmptyStringResultIsNotMissingCache(t *testing.T) {
	smokeRecalcReady(t)
	path := filepath.Join(t.TempDir(), "empty.xlsx")
	f := excelize.NewFile()
	_ = f.SetCellStr("Sheet1", "A1", "k")
	_ = f.SetCellFloat("Sheet1", "B1", 1, -1, 64)
	_ = f.SetCellFormula("Sheet1", "C1", `IFERROR(VLOOKUP("zzz",A1:B1,2,FALSE()),"")`)
	_ = f.SetCellFormula("Sheet1", "D1", "1+1")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	// 未重算：两格都没有缓存值，都是 missing_cache（这才是「没重算」）。
	before := auditKinds(t, path)
	if before["C1"] != "missing_cache" || before["D1"] != "missing_cache" || len(before) != 2 {
		t.Fatalf("未重算时两格都应记 missing_cache：%v", before)
	}

	if _, err := Recalc(path); err != nil {
		t.Fatalf("重算失败：%v", err)
	}
	f2, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ctC, _ := f2.GetCellType("Sheet1", "C1")
	vC, _ := f2.GetCellValue("Sheet1", "C1")
	ctD, _ := f2.GetCellType("Sheet1", "D1")
	vD, _ := f2.GetCellValue("Sheet1", "D1")
	_ = f2.Close()
	t.Logf("重算后 C1 type=%q value=%q；D1 type=%q value=%q", ctC, vC, ctD, vD)

	// 重算后：数值结果不再有发现；空串结果记 empty_result，不再算未重算。
	after := auditKinds(t, path)
	if _, has := after["D1"]; has {
		t.Errorf("算出数值的公式不应再有发现：%v", after)
	}
	if after["C1"] != "empty_result" {
		t.Fatalf("空串结果应记 empty_result（不是 missing_cache）：%v", after)
	}
}
