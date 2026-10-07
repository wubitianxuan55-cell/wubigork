package workcost

// export_recalc_test.go —— 导出结果的**数值**验证（LibreOffice 重算 + 金样对拍）。
//
// 结构验证（export_test.go）只证明公式「长得对」；本文件证明公式「算出来也对」：
// 导出 → LibreOffice 重算公式缓存值 → 读关键金额，与人工核算的金样比对。
//
// 默认跳过（依赖本机 LibreOffice，约数秒）。显式开启：
//
//	WORKCOST_RECALC=1 go test ./internal/gaea/workcost/ -run TestExportRecalc
//
// 金样来源：旺平矿业产物的「综合单价」表与「费用汇总」表——
// 第一项 WP01 测量放线 = 技术工 20 工日×350 + 普通工 20 工日×300 = 13000。
// 这是人工可复核的口算值，任何口径漂移（外委未并入材料桶、取费混进单价、
// 含量与单价错位）都会让它变红。

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

const recalcScript = "../../../.gaea/skills/xlsx/scripts/recalc.py"

func TestExportRecalcNumericGoldSample(t *testing.T) {
	if os.Getenv("WORKCOST_RECALC") != "1" {
		t.Skip("未设置 WORKCOST_RECALC=1，跳过 LibreOffice 重算验证")
	}
	py := os.Getenv("WORKCOST_PYTHON")
	soffice := os.Getenv("WORKCOST_SOFFICE")
	if py == "" || soffice == "" {
		t.Skip("需设置 WORKCOST_PYTHON 与 WORKCOST_SOFFICE")
	}
	if _, err := os.Stat(recalcScript); err != nil {
		t.Skipf("重算脚本不可用: %v", err)
	}

	b := buildExportFixture(t)
	data, err := ExportProjectWorkbook(b, ExportOptions{})
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "recalc.xlsx")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		t.Fatalf("写文件失败: %v", err)
	}

	cmd := exec.Command(py, recalcScript, out)
	cmd.Env = append(os.Environ(), "SOFFICE_PATH="+soffice)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("重算失败: %v\n%s", err, string(outBytes))
	}

	f, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatalf("重算后打开失败: %v", err)
	}
	defer func() { _ = f.Close() }()

	// ① 综合单价表第一项汇总行 P 列 = 人材机合价。
	//    旺平第一项 WP01 测量放线 = 技术工 20 工日×350 + 普通工 20 工日×300 = 13000。
	const summaryRow = headerRowUnitPrice + 1
	got := readNum(t, f, SheetUnitPriceName, "P"+itoa(summaryRow))
	const wantFirstItem = 13000.0
	if diff := got - wantFirstItem; diff > 0.02 || diff < -0.02 {
		t.Errorf("第一项综合单价 = %.2f, want %.2f（人工可复核金样）", got, wantFirstItem)
	}

	// ② 费用汇总：直接费合计 = Σ 各清单项合价（两条独立路径必须一致）。
	directRow := findRowByLabel(t, f, SheetSummaryName, "人材机直接费合计")
	if directRow == 0 {
		t.Fatal("未找到直接费合计行")
	}
	direct := readNum(t, f, SheetSummaryName, "E"+itoa(directRow))
	itemSum := 0.0
	for i := headerRowSummary + 1; i < directRow; i++ {
		// I 列是公式（=G*H），必须取**缓存值**——cellAt 走 cellText 会拿到
		// "=G4*H4"，parseNum 失败，itemSum 恒为 0（首版就这么漏的）。
		v, err := f.GetCellValue(SheetSummaryName, "I"+itoa(i))
		if err != nil || v == "" {
			continue
		}
		if n, ok := parseNum(v); ok {
			itemSum += n
		}
	}
	closeTo(t, "直接费合计（应等于各清单项合价之和）", direct, itemSum)

	// ②b 独立金样：按 bundle 口算全部清单项合价（工程量×Σ(含量×资源价)）。
	//     这条断言不调用生产算式，是真正的独立校验。
	goldDirect := expectedDirectFee(b)
	if diff := direct - goldDirect; diff > 1 || diff < -1 {
		t.Errorf("直接费合计 = %.2f, want %.2f（按 bundle 独立口算）", direct, goldDirect)
	}

	// ③ 取费链：企管 = 直接费×5%；税前 = 直接费+企管+利润+规费（不得重复计直接费）。
	mgmt := readNum(t, f, SheetSummaryName, "E"+itoa(findRowByLabel(t, f, SheetSummaryName, "企业管理费")))
	closeTo(t, "企业管理费", mgmt, direct*b.Fee.ManagementRate)

	profit := readNum(t, f, SheetSummaryName, "E"+itoa(findRowByLabel(t, f, SheetSummaryName, "利润")))
	reg := readNum(t, f, SheetSummaryName, "E"+itoa(findRowByLabel(t, f, SheetSummaryName, "规费")))
	pretax := readNum(t, f, SheetSummaryName, "E"+itoa(findRowByLabel(t, f, SheetSummaryName, "税前合计（不含税造价）")))
	closeTo(t, "税前合计（=直接费+企管+利润+规费）", pretax, direct+mgmt+profit+reg)
	// 反向：税前合计不得接近直接费的两倍——早期 SUM(直接费行:规费行) 的错正是如此。
	if pretax > direct*1.9 {
		t.Fatalf("税前合计 %.2f 接近直接费 %.2f 的两倍——区间求和把直接费算重了", pretax, direct)
	}

	total := readNum(t, f, SheetSummaryName, "E"+itoa(findRowByLabel(t, f, SheetSummaryName, "含税总造价")))
	tax := total - pretax
	closeTo(t, "增值税（=税后-税前）", tax, pretax*b.Fee.TaxRate)

	// ④ 封面指标必须与费用汇总一致（跨表联动生效）。
	coverTotal := readNum(t, f, SheetCoverName, "B"+itoa(findRowByLabel(t, f, SheetCoverName, "含税工程造价")))
	closeTo(t, "封面含税工程造价", coverTotal, total)

	t.Logf("重算结果：直接费 %.2f（独立口算 %.2f）/ 企管 %.2f / 税前 %.2f / 税 %.2f / 含税总造价 %.2f",
		direct, goldDirect, mgmt, pretax, tax, total)
}

// expectedDirectFee 按 bundle 独立口算直接费（不调用生产算式）：
// 清单项合价 = 工程量 × Σ(消耗量 × 资源价)，再累加。
func expectedDirectFee(b *ProjectBundle) float64 {
	price := map[string]float64{}
	for _, r := range b.Resources {
		price[r.Code] = r.Price
	}
	unitPrice := map[string]float64{}
	for _, l := range b.Lines {
		p, ok := price[l.ResourceCode]
		if !ok {
			p = l.Price
		}
		unitPrice[l.ItemCode] += l.Quantity * p
	}
	sum := 0.0
	for _, it := range b.Items {
		sum += it.Quantity * unitPrice[it.Code]
	}
	return Round2(sum)
}

func readNum(t *testing.T, f *excelize.File, sheet, cell string) float64 {
	t.Helper()
	v, err := f.GetCellValue(sheet, cell)
	if err != nil {
		t.Fatalf("读 %s!%s 失败: %v", sheet, cell, err)
	}
	n, ok := parseNum(v)
	if !ok {
		t.Fatalf("%s!%s 不是数值: %q", sheet, cell, v)
	}
	return n
}
