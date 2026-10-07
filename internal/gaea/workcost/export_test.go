package workcost

// export_test.go —— 五表导出器的结构验证。
//
// 验证策略：**公式对拍**。模版的表间关系全在公式里（VLOOKUP 取价、SUMIF 拆
// 三费、跨表引用综合单价与直接费），所以「导出的公式与实测模版一致」就是正确
// 性的判据——比比对算好的数字更能抓住口径错误（取费内联进单价这类错，数字
// 可能碰巧对得上，公式一定不对）。

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// buildExportFixture 用真实产物解析结果做导出输入。
func buildExportFixture(t *testing.T) *ProjectBundle {
	t.Helper()
	b, err := ParseProjectWorkbook(filepath.FromSlash(fixtureWangping))
	if err != nil {
		t.Fatalf("解析夹具失败: %v", err)
	}
	return b
}

func TestExportProjectWorkbookSheets(t *testing.T) {
	b := buildExportFixture(t)
	data, err := ExportProjectWorkbook(b, ExportOptions{})
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("打开导出结果失败: %v", err)
	}
	defer func() { _ = f.Close() }()

	want := []string{SheetCoverName, SheetSummaryName, SheetUnitPriceName, SheetResourceName, SheetQuantityName}
	got := f.GetSheetList()
	if len(got) != len(want) {
		t.Fatalf("sheet 数 = %d, want %d（%v）", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sheet[%d] = %q, want %q（顺序即模版顺序）", i, got[i], want[i])
		}
	}
}

// TestExportUnitPriceFormulas 综合单价表公式对拍实测模版：
//
//	汇总行 K4 = SUM(K5:K6)（单个明细项时模版写作 SUM(K5:K5)）
//	汇总行 L4 = SUMIF(E5:E6,"人工",K5:K6)
//	汇总行 M4 = SUMIF(E5:E6,"材料",K5:K6)+SUMIF(E5:E6,"外委",K5:K6)  ← 外委并入材料桶
//	汇总行 N4 = SUMIF(E5:E6,"机械",K5:K6)
//	汇总行 P4 = K4                ← 综合单价 = 人材机小计，无取费
//	明细行 J5 = IFERROR(VLOOKUP(F5,工料机价格!$A$4:$F$n,6,FALSE()),0)
//	明细行 K5 = I5*J5             ← 含量 × 单价
func TestExportUnitPriceFormulas(t *testing.T) {
	b := buildExportFixture(t)
	data, err := ExportProjectWorkbook(b, ExportOptions{})
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("打开失败: %v", err)
	}
	defer func() { _ = f.Close() }()

	sheet := SheetUnitPriceName
	// 第一项汇总行 = 表头行 + 1 = 4；其明细行紧随。
	const summaryRow = headerRowUnitPrice + 1
	const detailRow = summaryRow + 1

	labor := formulaOf(f, sheet, "L4")
	if !strings.Contains(labor, `SUMIF(E`) || !strings.Contains(labor, `"人工"`) {
		t.Errorf("L%d 应为人工 SUMIF，得到 %q", summaryRow, labor)
	}
	material := formulaOf(f, sheet, "M4")
	if !strings.Contains(material, `"材料"`) || !strings.Contains(material, `"外委"`) {
		t.Errorf("M%d 应把外委并入材料桶（SUMIF(材料)+SUMIF(外委)），得到 %q", summaryRow, material)
	}
	machine := formulaOf(f, sheet, "N4")
	if !strings.Contains(machine, `"机械"`) {
		t.Errorf("N%d 应为机械 SUMIF，得到 %q", summaryRow, machine)
	}
	if p := formulaOf(f, sheet, "P4"); p != "K4" {
		t.Errorf("P%d 应等于 K%d（综合单价=人材机小计，取费不进单价），得到 %q", summaryRow, summaryRow, p)
	}
	if k := formulaOf(f, sheet, "K4"); !strings.HasPrefix(k, "SUM(K") {
		t.Errorf("K%d 应为明细合计 SUM，得到 %q", summaryRow, k)
	}

	price := formulaOf(f, sheet, "J5")
	if !strings.Contains(price, "VLOOKUP(F5,") || !strings.Contains(price, SheetResourceName) {
		t.Errorf("J%d 应为 VLOOKUP 取工料机价格，得到 %q", detailRow, price)
	}
	if !strings.HasPrefix(price, "IFERROR(") {
		t.Errorf("J%d 应带 IFERROR 兜底，得到 %q", detailRow, price)
	}
	if k := formulaOf(f, sheet, "K5"); k != "I5*J5" {
		t.Errorf("K%d 应为 I*J（含量×单价），得到 %q", detailRow, k)
	}
}

// TestExportSummaryLinksUnitPrice 费用汇总的 H 列必须跨表引用综合单价 P 列，
// 而不是复制数值——这是「改消耗量/资源价后全表重算」联动的前提。
func TestExportSummaryLinksUnitPrice(t *testing.T) {
	b := buildExportFixture(t)
	data, err := ExportProjectWorkbook(b, ExportOptions{})
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("打开失败: %v", err)
	}
	defer func() { _ = f.Close() }()

	sheet := SheetSummaryName
	firstItem := headerRowSummary + 1
	h := formulaOf(f, sheet, "H"+itoa(firstItem))
	if !strings.Contains(h, SheetUnitPriceName) || !strings.Contains(h, "!P") {
		t.Errorf("H%d 应引用「%s」表 P 列，得到 %q", firstItem, SheetUnitPriceName, h)
	}
	i := formulaOf(f, sheet, "I"+itoa(firstItem))
	if !strings.Contains(i, "*") {
		t.Errorf("I%d 应为 工程量×综合单价，得到 %q", firstItem, i)
	}

	// 直接费合计必须是 SUM 而不是硬编码数字。
	direct := findFeeRowFormula(t, f, sheet, "人材机直接费合计", "E")
	if !strings.HasPrefix(direct, "SUM(") {
		t.Errorf("直接费合计应为 SUM 公式，得到 %q", direct)
	}
}

// TestExportFeeChainFormulas 取费链公式对拍：企管=直接费×费率、利润=(直接费+企管)×
// 利润率、税前=SUM(直接费..规费)、税=税前×税率、总造价=税前+税。
func TestExportFeeChainFormulas(t *testing.T) {
	b := buildExportFixture(t)
	data, err := ExportProjectWorkbook(b, ExportOptions{})
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("打开失败: %v", err)
	}
	defer func() { _ = f.Close() }()

	sheet := SheetSummaryName
	mgmt := findFeeRowFormula(t, f, sheet, "企业管理费", "E")
	if !strings.Contains(mgmt, "*D") {
		t.Errorf("企业管理费应为 基数×费率，得到 %q", mgmt)
	}
	profit := findFeeRowFormula(t, f, sheet, "利润", "E")
	if !strings.Contains(profit, "(") || !strings.Contains(profit, "+") || !strings.Contains(profit, "*D") {
		t.Errorf("利润应为（直接费+管理费）×费率，得到 %q", profit)
	}
	preTax := findFeeRowFormula(t, f, sheet, "税前合计（不含税造价）", "E")
	// 必须是**显式逐项相加**，不能用 SUM(直接费行:规费行)——区间起点就是直接费
	// 行，SUM 会把直接费再算一遍（实测踩到：税前合计多出一个直接费+管理费）。
	for _, must := range []string{"+"} {
		if !strings.Contains(preTax, must) {
			t.Errorf("税前合计应为逐项相加，得到 %q", preTax)
		}
	}
	if strings.HasPrefix(preTax, "SUM(") {
		t.Errorf("税前合计不得用 SUM 区间（会把区间起点的直接费算重），得到 %q", preTax)
	}
	tax := findFeeRowFormula(t, f, sheet, "增值税", "E")
	if !strings.Contains(tax, "*D") {
		t.Errorf("增值税应为 税前×税率，得到 %q", tax)
	}
	total := findFeeRowFormula(t, f, sheet, "含税总造价", "E")
	if !strings.Contains(total, "+") {
		t.Errorf("含税总造价应为 税前+税，得到 %q", total)
	}

	// 费率数值：旺平实测 管理费 5% / 利润 0 / 税 3%。
	rates := map[string]float64{"企业管理费": b.Fee.ManagementRate, "利润": b.Fee.ProfitRate, "增值税": b.Fee.TaxRate}
	for name, want := range rates {
		row := findRowByLabel(t, f, sheet, name)
		if row == 0 {
			t.Fatalf("未找到取费行 %q", name)
		}
		got, err := f.GetCellValue(sheet, "D"+itoa(row))
		if err != nil {
			t.Fatalf("读 D%d 失败: %v", row, err)
		}
		if n, ok := parseNum(got); !ok || n != want {
			t.Errorf("%s 费率 = %q, want %v", name, got, want)
		}
	}
}

// TestExportCoverReferencesSummary 封面指标必须跨表引用费用汇总（保持联动）。
func TestExportCoverReferencesSummary(t *testing.T) {
	b := buildExportFixture(t)
	data, err := ExportProjectWorkbook(b, ExportOptions{})
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("打开失败: %v", err)
	}
	defer func() { _ = f.Close() }()

	sheet := SheetCoverName
	row := findRowByLabel(t, f, sheet, "含税工程造价")
	if row == 0 {
		t.Fatal("封面应有「含税工程造价」指标行")
	}
	fo := formulaOf(f, sheet, "B"+itoa(row))
	if !strings.Contains(fo, SheetSummaryName) || !strings.Contains(fo, "!E") {
		t.Errorf("封面指标应引用「%s」E 列，得到 %q", SheetSummaryName, fo)
	}
}

// TestExportThenReparseRoundTrip 导出→再解析 应还原同样的清单项/资源/消耗量，
// 保证导出不是单向死路（能做「导出→改价→导回」的闭环）。
func TestExportThenReparseRoundTrip(t *testing.T) {
	b := buildExportFixture(t)
	data, err := ExportProjectWorkbook(b, ExportOptions{Project: "导出验证项目", Location: "测试地点", Duration: "12个月"})
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	// 写临时文件后走文件解析路径（ParseProjectWorkbook 只接路径）。
	dir := t.TempDir()
	out := filepath.Join(dir, "roundtrip.xlsx")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		t.Fatalf("写临时文件失败: %v", err)
	}
	b2, err := ParseProjectWorkbook(out)
	if err != nil {
		t.Fatalf("再解析失败: %v", err)
	}
	if len(b2.Resources) != len(b.Resources) {
		t.Errorf("资源数 往返后 = %d, want %d", len(b2.Resources), len(b.Resources))
	}
	if len(b2.Lines) != len(b.Lines) {
		t.Errorf("消耗行数 往返后 = %d, want %d", len(b2.Lines), len(b.Lines))
	}
	if len(b2.Items) != len(b.Items) {
		t.Errorf("清单项数 往返后 = %d, want %d", len(b2.Items), len(b.Items))
	}
	// 封面字段往返。
	if b2.Project != "导出验证项目" {
		t.Errorf("项目名 往返后 = %q, want 导出验证项目", b2.Project)
	}
	if b2.Location != "测试地点" {
		t.Errorf("地点 往返后 = %q, want 测试地点", b2.Location)
	}
	if b2.Duration != "12个月" {
		t.Errorf("工期 往返后 = %q, want 12个月", b2.Duration)
	}
}

// TestExportRejectsEmptyBundle 空项目不许导出（避免产出空壳表被当成成果）。
func TestExportRejectsEmptyBundle(t *testing.T) {
	if _, err := ExportProjectWorkbook(&ProjectBundle{}, ExportOptions{}); err == nil {
		t.Fatal("空项目应拒绝导出")
	}
	if _, err := ExportProjectWorkbook(nil, ExportOptions{}); err == nil {
		t.Fatal("nil 项目应拒绝导出")
	}
}

// ── 测试小工具 ──────────────────────────────────────────────────────

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// findRowByLabel 找标签所在行号：**先精确匹配**，再退回包含匹配。
// （必须精确优先——「利润」是「（直接费+管理费）× 利润率」这类说明串的子串，
// 纯包含匹配会先搜到参数说明行，导致读到空费率而误报。）
func findRowByLabel(t *testing.T, f *excelize.File, sheet, label string) int {
	t.Helper()
	rows, err := f.GetRows(sheet)
	if err != nil {
		return 0
	}
	for i, r := range rows {
		for _, c := range r {
			if strings.TrimSpace(c) == label {
				return i + 1
			}
		}
	}
	for i, r := range rows {
		for _, c := range r {
			if strings.Contains(c, label) {
				return i + 1
			}
		}
	}
	return 0
}

// findFeeRowFormula 找 label 所在行的指定列公式。
func findFeeRowFormula(t *testing.T, f *excelize.File, sheet, label, col string) string {
	t.Helper()
	row := findRowByLabel(t, f, sheet, label)
	if row == 0 {
		t.Fatalf("未找到行 %q", label)
	}
	return formulaOf(f, sheet, col+itoa(row))
}
