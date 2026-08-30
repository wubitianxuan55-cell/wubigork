package app

import (
	"encoding/json"

	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/office/xlsxedit"
	"github.com/xuri/excelize/v2"
)

// buildVerifyXlsx 造一个两列工作簿：A1=42、B1 公式 =A1*2，返回路径。
func buildVerifyXlsx(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "verify-deep.xlsx")
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetCellValue("Sheet1", "A1", 42); err != nil {
		t.Fatalf("SetCellValue: %v", err)
	}
	if err := f.SetCellFormula("Sheet1", "B1", "A1*2"); err != nil {
		t.Fatalf("SetCellFormula: %v", err)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	return path
}

func openVerifyXlsx(t *testing.T, path string) *excelize.File {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func mustOpsJSON(t *testing.T, ops []xlsxedit.Op) string {
	t.Helper()
	b, err := json.Marshal(ops)
	if err != nil {
		t.Fatalf("marshal ops: %v", err)
	}
	return string(b)
}

func TestVerifyXlsxChannelADeep_ClaimsPass(t *testing.T) {
	f := openVerifyXlsx(t, buildVerifyXlsx(t))
	rec := evidence.ChangeRecord{OpsJSON: mustOpsJSON(t, []xlsxedit.Op{
		{Type: "set_value", Sheet: "Sheet1", Target: "A1", Value: 42},
		{Type: "set_formula", Sheet: "Sheet1", Target: "B1", Formula: "=A1*2"},
	})}
	got := verifyXlsxChannelADeep(f, rec)
	if !strings.Contains(got, "引用级复核 2 项声明与实况一致") {
		t.Errorf("声明应全部核对通过, got %q", got)
	}
	if strings.Contains(got, "fail") {
		t.Errorf("不应失败, got %q", got)
	}
}

func TestVerifyXlsxChannelADeep_ValueMismatch(t *testing.T) {
	f := openVerifyXlsx(t, buildVerifyXlsx(t))
	rec := evidence.ChangeRecord{OpsJSON: mustOpsJSON(t, []xlsxedit.Op{
		{Type: "set_value", Sheet: "Sheet1", Target: "A1", Value: 43},
	})}
	got := verifyXlsxChannelADeep(f, rec)
	if !strings.HasPrefix(strings.TrimSpace(got), "；fail:") {
		t.Errorf("值不符应 fail, got %q", got)
	}
	if !strings.Contains(got, "A1 预期「43」实际「42」") {
		t.Errorf("应给出预期/实际, got %q", got)
	}
}

func TestVerifyXlsxChannelADeep_FormulaMismatch(t *testing.T) {
	f := openVerifyXlsx(t, buildVerifyXlsx(t))
	rec := evidence.ChangeRecord{OpsJSON: mustOpsJSON(t, []xlsxedit.Op{
		{Type: "set_formula", Sheet: "Sheet1", Target: "B1", Formula: "=A1*3"},
	})}
	got := verifyXlsxChannelADeep(f, rec)
	if !strings.Contains(got, "公式预期") || !strings.Contains(got, "fail") {
		t.Errorf("公式不符应 fail 并给出预期/实际, got %q", got)
	}
}

func TestVerifyXlsxChannelADeep_NumericNormalization(t *testing.T) {
	f := openVerifyXlsx(t, buildVerifyXlsx(t))
	rec := evidence.ChangeRecord{OpsJSON: mustOpsJSON(t, []xlsxedit.Op{
		{Type: "set_value", Sheet: "Sheet1", Target: "A1", Value: "42.0"},
	})}
	got := verifyXlsxChannelADeep(f, rec)
	if !strings.Contains(got, "实况一致") {
		t.Errorf("数值 42.0 与 42 应等值, got %q", got)
	}
}

func TestVerifyXlsxChannelADeep_ReplaceOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "verify-replace.xlsx")
	f0 := excelize.NewFile()
	// 文件实况 = 已应用替换后的文本；证据卡声明替换发生，核对实况是否如实。
	if err := f0.SetCellValue("Sheet1", "C1", "新方案合计"); err != nil {
		t.Fatalf("SetCellValue: %v", err)
	}
	if err := f0.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	f0.Close()
	f := openVerifyXlsx(t, path)

	ok := verifyXlsxChannelADeep(f, evidence.ChangeRecord{OpsJSON: mustOpsJSON(t, []xlsxedit.Op{
		{Type: "replace", Sheet: "Sheet1", Target: "C1", Find: "旧方案", Replace: "新方案"},
	})})
	if !strings.Contains(ok, "实况一致") {
		t.Errorf("替换如落盘应通过, got %q", ok)
	}

	bad := verifyXlsxChannelADeep(f, evidence.ChangeRecord{OpsJSON: mustOpsJSON(t, []xlsxedit.Op{
		{Type: "replace", Sheet: "Sheet1", Target: "C1", Find: "不存在", Replace: "别的"},
	})})
	if !strings.Contains(bad, "fail") {
		t.Errorf("声明替换未落盘应 fail, got %q", bad)
	}
}

func TestVerifyXlsxChannelADeep_NoOpsJSONLegacy(t *testing.T) {
	f := openVerifyXlsx(t, buildVerifyXlsx(t))
	got := verifyXlsxChannelADeep(f, evidence.ChangeRecord{})
	if !strings.Contains(got, "不适用") {
		t.Errorf("旧卡（无 opsJson）应声明比对不适用, got %q", got)
	}
}

func TestVerifyXlsxChannelADeep_SkipsBatchOps(t *testing.T) {
	f := openVerifyXlsx(t, buildVerifyXlsx(t))
	rec := evidence.ChangeRecord{OpsJSON: mustOpsJSON(t, []xlsxedit.Op{
		{Type: "set_style", Sheet: "Sheet1", Target: "A1"},
		{Type: "fill_range", Sheet: "Sheet1", Range: "A1:B2", Value: 1},
	})}
	got := verifyXlsxChannelADeep(f, rec)
	if !strings.Contains(got, "无可核对项") || !strings.Contains(got, "跳过 2 项") {
		t.Errorf("批量/样式类应诚实跳过, got %q", got)
	}
}

func TestCellValueEquals(t *testing.T) {
	cases := []struct {
		a, b  string
		match bool
	}{
		{"42", "42.0", true},
		{" 42 ", "42", true},
		{"合计", "合计", true},
		{"合计", "总计", false},
		{"42", "43", false},
		{"", "", true},
	}
	for i, c := range cases {
		if got := cellValueEquals(c.a, c.b); got != c.match {
			t.Errorf("case %d: cellValueEquals(%q,%q)=%v 期望 %v", i, c.a, c.b, got, c.match)
		}
	}
}

func TestCompactFormula(t *testing.T) {
	if compactFormula("=A1 * 2") != "A1*2" || compactFormula("A1*2") != "A1*2" {
		t.Errorf("公式归一应剥 = 前缀与空白: %q / %q", compactFormula("=A1 * 2"), compactFormula("A1*2"))
	}
	if compactFormula(" =SUM(A1:A3) ") != "SUM(A1:A3)" {
		t.Errorf("SUM 归一意外: %q", compactFormula(" =SUM(A1:A3) "))
	}
}
