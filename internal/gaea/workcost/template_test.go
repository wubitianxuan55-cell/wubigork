package workcost

// template_test.go — 成本测算模版（六表 + 三级行组）的验证。
//
// 三层各一组用例，缺一层都会漏掉一类错：
//   - 结构（TestTemplateStructure）：表序、列序、行组、公式链——公式对拍；
//   - 往返（TestTemplateFillParseApply）：按真人填法填一份模版 → 解析 → 落库，
//     1:N 挂接、定额含量行、系数必须原样还原（模版与库形态对应的硬证据）；
//   - 数值（TestTemplateRecalcNumeric）：LibreOffice 真重算后对人工可复核的金样——
//     公式「长得对」不等于「算出来对」（三费没乘系数就是这里抓的）。

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// 清单表/费用汇总表的首个数据行（表头在第 3 行）。用例里的行号口径：综合单价每组
// = 1 汇总行 + 2 定额块×(1 定额行 + 4 明细行) = 11 行，组 g 的汇总行 = 4 + 11g。
const tBillFirst = headerRowSummary + 1

// filledTemplate 按真人填法填好一份小模版（3 组，用 2 组），返回工作簿字节。
//
//	Q01 垫层 m³×100  ← D-A1 垫层浇筑（系数空=1）：人工 L01 0.1、材料 C20 1.02
//	                 ← D-A2 垫层夯实（系数 0.5）   ：机械 EXC 0.02
//	Q02 面层 m²×2000 ← D-B1 面层摊铺（系数空=1）  ：材料 C20 0.03、人工 L01 0.05
//	第 3 组留空（验证没用到的组不产生幽灵清单）
func filledTemplate(t *testing.T) []byte {
	t.Helper()
	data, err := ExportProjectTemplate(TemplateOptions{
		Groups: 3,
		Resources: []Resource{
			{Code: "L01", Kind: KindLabor, Title: "普工", Unit: "工日", CurrentPrice: 300},
			{Code: "C20", Kind: KindMaterial, Title: "商品混凝土", Spec: "C20", Unit: "m³", CurrentPrice: 400},
			{Code: "EXC", Kind: KindMachine, Title: "挖掘机", Unit: "台班", CurrentPrice: 1000},
		},
	})
	if err != nil {
		t.Fatalf("生成模版失败: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("打开模版失败: %v", err)
	}
	set := func(sheet, cell string, v any) {
		t.Helper()
		var err error
		switch x := v.(type) {
		case string:
			err = f.SetCellStr(sheet, cell, x)
		case float64:
			err = f.SetCellFloat(sheet, cell, x, -1, 64)
		}
		if err != nil {
			t.Fatalf("填 %s!%s 失败: %v", sheet, cell, err)
		}
	}
	up, bill := SheetUnitPriceName, SheetBillName
	// 组 0：清单 Q01，两个定额块。
	set(up, "B4", "Q01")
	set(up, "C4", "垫层")
	set(up, "D4", "m³")
	set(up, "D5", "m³")
	set(up, "F5", "D-A1")
	set(up, "G5", "垫层浇筑")
	set(up, "E6", "人工")
	set(up, "I6", "L01")
	set(up, "L6", 0.1)
	set(up, "E7", "材料")
	set(up, "I7", "C20")
	set(up, "L7", 1.02)
	set(up, "D10", "m³")
	set(up, "F10", "D-A2")
	set(up, "G10", "垫层夯实")
	set(up, "H10", 0.5)
	set(up, "E11", "机械")
	set(up, "I11", "EXC")
	set(up, "L11", 0.02)
	// 组 1：清单 Q02，一个定额块。
	set(up, "B15", "Q02")
	set(up, "C15", "面层")
	set(up, "D15", "m²")
	set(up, "D16", "m²")
	set(up, "F16", "D-B1")
	set(up, "G16", "面层摊铺")
	set(up, "E17", "材料")
	set(up, "I17", "C20")
	set(up, "L17", 0.03)
	set(up, "E18", "人工")
	set(up, "I18", "L01")
	set(up, "L18", 0.05)
	// 清单表：编码/名称/单位/分部/工程量（综合单价与合价是公式）。
	set(bill, "B4", "A路基")
	set(bill, "C4", "Q01")
	set(bill, "D4", "垫层")
	set(bill, "F4", "m³")
	set(bill, "G4", 100.0)
	set(bill, "B5", "B路面")
	set(bill, "C5", "Q02")
	set(bill, "D5", "面层")
	set(bill, "F5", "m²")
	set(bill, "G5", 2000.0)

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("写出模版失败: %v", err)
	}
	_ = f.Close()
	return buf.Bytes()
}

// recalcBytes 用 LibreOffice 重算公式缓存值后重新打开（需 WORKCOST_RECALC=1 与
// WORKCOST_PYTHON / WORKCOST_SOFFICE；未设则跳过）。
func recalcBytes(t *testing.T, data []byte) *excelize.File {
	t.Helper()
	if os.Getenv("WORKCOST_RECALC") != "1" {
		t.Skip("未设置 WORKCOST_RECALC=1，跳过 LibreOffice 重算验证")
	}
	py, soffice := os.Getenv("WORKCOST_PYTHON"), os.Getenv("WORKCOST_SOFFICE")
	if py == "" || soffice == "" {
		t.Skip("需设置 WORKCOST_PYTHON 与 WORKCOST_SOFFICE")
	}
	if _, err := os.Stat(recalcScript); err != nil {
		t.Skipf("重算脚本不可用: %v", err)
	}
	out := filepath.Join(t.TempDir(), "recalc.xlsx")
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
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// TestTemplateStructure 模版结构：六表顺序、综合单价三级行组与公式链、清单表引用、
// 费用汇总只留取费链且直接费跨表累计清单合价、录入列数据有效性。
func TestTemplateStructure(t *testing.T) {
	data, err := ExportProjectTemplate(TemplateOptions{Groups: 3})
	if err != nil {
		t.Fatalf("生成模版失败: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("打开模版失败: %v", err)
	}
	defer func() { _ = f.Close() }()

	want := []string{SheetCoverName, SheetBillName, SheetSummaryName, SheetUnitPriceName, SheetResourceName, SheetQuantityName}
	if got := f.GetSheetList(); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("sheet 序 = %v, want %v", got, want)
	}

	up := SheetUnitPriceName
	header, _ := f.GetRows(up)
	if strings.Join(header[headerRowUnitPrice-1], "|") != strings.Join(unitPriceCols, "|") {
		t.Errorf("综合单价表头与列序不符：\n got %v\nwant %v", header[headerRowUnitPrice-1], unitPriceCols)
	}
	// 行组：汇总 → 定额 → 明细…（第 3 组只放汇总行也成立）。
	for _, c := range []struct{ cell, want string }{
		{"E4", "汇总"}, {"E5", "定额"}, {"E10", "定额"}, {"E15", "汇总"}, {"E16", "定额"}, {"E21", "定额"}, {"E26", "汇总"},
	} {
		if got, _ := f.GetCellValue(up, c.cell); got != c.want {
			t.Errorf("%s!%s 行类 = %q, want %q", up, c.cell, got, c.want)
		}
	}
	// 汇总行：综合单价=Σ定额块贡献；三费=Σ块 系数×块内 SUMIF（系数≠1 时三费仍与综合单价同口径）。
	wantFormula := map[string]string{
		"N4": `SUMIF(E5:E14,"定额",N5:N14)`,
		"O4": `IF(H5="",1,H5)*SUMIF(E6:E9,"人工",N6:N9)+IF(H10="",1,H10)*SUMIF(E11:E14,"人工",N11:N14)`,
		"P4": `IF(H5="",1,H5)*(SUMIF(E6:E9,"材料",N6:N9)+SUMIF(E6:E9,"外委",N6:N9))+IF(H10="",1,H10)*(SUMIF(E11:E14,"材料",N11:N14)+SUMIF(E11:E14,"外委",N11:N14))`,
		"Q4": `IF(H5="",1,H5)*SUMIF(E6:E9,"机械",N6:N9)+IF(H10="",1,H10)*SUMIF(E11:E14,"机械",N11:N14)`,
		"S4": "N4",
		// 定额行：定额单价=Σ块内明细；贡献=单价×系数。
		"M5": "SUM(N6:N9)",
		"N5": `IF(H5="",1,H5)*M5`,
		// 明细行：单价 VLOOKUP 工料机价格 G 列；名称/单位自动带出；合价=含量×单价。
		"M6": `IFERROR(VLOOKUP(I6,'工料机价格'!$A$4:$G$63,7,FALSE()),0)`,
		"J6": `IFERROR(VLOOKUP(I6,'工料机价格'!$A$4:$G$63,4,FALSE()),"")`,
		"N6": "L6*M6",
		// 第 2 组汇总行公式压自己的行区间（不是第 1 组的）。
		"N15": `SUMIF(E16:E25,"定额",N16:N25)`,
	}
	for cell, want := range wantFormula {
		if got := formulaOf(f, up, cell); got != want {
			t.Errorf("%s!%s 公式\n got %s\nwant %s", up, cell, got, want)
		}
	}

	// 清单表：综合单价引用综合单价表各组汇总行 S 列；合价=工程量×综合单价。
	bill := SheetBillName
	for _, c := range []struct{ cell, want string }{
		{"H4", `'综合单价'!S4`}, {"H5", `'综合单价'!S15`}, {"H6", `'综合单价'!S26`},
		{"I4", "G4*H4"},
	} {
		if got := formulaOf(f, bill, c.cell); got != c.want {
			t.Errorf("%s!%s 公式 = %q, want %q", bill, c.cell, got, c.want)
		}
	}

	// 费用汇总：只有取费链；直接费=清单表合价的 SUM（跨表累计，不写死数）。
	fee := SheetSummaryName
	direct := findFeeRowFormula(t, f, fee, "人材机直接费合计", "E")
	lastBill := tBillFirst + 3 - 1
	if want := "SUM('清单'!I4:I" + itoa(lastBill) + ")"; direct != want {
		t.Errorf("直接费合计 = %q, want %q", direct, want)
	}
	for name, want := range map[string]float64{"企业管理费": 0.10, "利润": 0.07, "规费": 0.02, "增值税": 0.09} {
		row := findRowByLabel(t, f, fee, name)
		if row == 0 {
			t.Fatalf("费用汇总缺「%s」行", name)
		}
		got, _ := f.GetCellValue(fee, "D"+itoa(row))
		if n, ok := parseNum(got); !ok || n != want {
			t.Errorf("%s 费率 = %q, want %v（应预填默认取费口径）", name, got, want)
		}
	}
	for r := 1; r <= headerRowSummary; r++ {
		rows, _ := f.GetRows(fee)
		if r <= len(rows) {
			for _, c := range rows[r-1] {
				if c == "项目名称" || c == "工程量" {
					t.Fatalf("费用汇总表不该有清单区表头（行 %d 出现 %q）", r, c)
				}
			}
		}
	}

	// 工料机价格表：自有码列在编码后，单价在 G 列。
	rh, _ := f.GetRows(SheetResourceName)
	if strings.Join(rh[headerRowResource-1], "|") != strings.Join(priceSheetCols, "|") {
		t.Errorf("工料机价格表头不符：%v", rh[headerRowResource-1])
	}
	// 录入列数据有效性：行类（综合单价）与类别（工料机价格）。
	if dvs, _ := f.GetDataValidations(up); len(dvs) == 0 {
		t.Error("综合单价表行类列应有数据有效性（下拉）")
	}
	if dvs, _ := f.GetDataValidations(SheetResourceName); len(dvs) == 0 {
		t.Error("工料机价格表类别列应有数据有效性（下拉）")
	}
}

// TestTemplateFillParseApply 填好的模版 → 解析 → 落库：三级行组原样还原成库的
// 1:N 挂接——这是「模版依据造价库设计、与库对应」的硬证据。
func TestTemplateFillParseApply(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "filled.xlsx")
	if err := os.WriteFile(path, filledTemplate(t), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := ParseProjectWorkbook(path)
	if err != nil {
		t.Fatalf("解析填好的模版失败: %v", err)
	}

	// ① 解析：2 条清单（第 3 组留空 → 无幽灵清单）、3 个定额块、6 条明细各归其块。
	if len(b.Items) != 2 || b.Items[0].Code != "Q01" || b.Items[1].Code != "Q02" {
		t.Fatalf("清单项应为 Q01/Q02 两条，得到 %+v", b.Items)
	}
	if b.Items[0].Quantity != 100 || b.Items[1].Quantity != 2000 || b.Items[0].Division != "A路基" {
		t.Errorf("清单工程量/分部应取自清单表，得到 %+v", b.Items)
	}
	type blockKey struct{ item, code, title, unit string }
	var blocks []blockKey
	for _, blk := range b.QuotaBlocks {
		blocks = append(blocks, blockKey{blk.ItemCode, blk.Code, blk.Title, blk.Unit})
	}
	wantBlocks := []blockKey{{"Q01", "D-A1", "垫层浇筑", "m³"}, {"Q01", "D-A2", "垫层夯实", "m³"}, {"Q02", "D-B1", "面层摊铺", "m²"}}
	if len(blocks) != len(wantBlocks) {
		t.Fatalf("定额块 = %v, want %v", blocks, wantBlocks)
	}
	for i := range wantBlocks {
		if blocks[i] != wantBlocks[i] {
			t.Errorf("定额块[%d] = %v, want %v", i, blocks[i], wantBlocks[i])
		}
	}
	if f := b.QuotaBlocks[1].Factor; f != 0.5 {
		t.Errorf("D-A2 的系数应为 0.5，得到 %v", f)
	}
	owner := map[string][]string{} // 定额码 → 明细资源码
	for _, l := range b.Lines {
		owner[l.QuotaCode] = append(owner[l.QuotaCode], l.ResourceCode)
	}
	for code, want := range map[string]string{"D-A1": "L01,C20", "D-A2": "EXC", "D-B1": "C20,L01"} {
		if got := strings.Join(owner[code], ","); got != want {
			t.Errorf("定额 %s 的明细应归 %s，得到 %q", code, want, got)
		}
	}
	if len(b.Skipped) != 0 {
		t.Errorf("没用到的预置定额块/空组应静默跳过，Skipped = %+v", b.Skipped)
	}

	// ② 落库。
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	res := s.ApplyProjectBundle(b)
	if len(res.Errors) > 0 {
		t.Fatalf("落库不应有错误：%v", res.Errors)
	}
	if res.QuotaNew != 3 {
		t.Errorf("应新建 3 条定额，得到 %+v", res)
	}
	for code, want := range map[string]int{"D-A1": 2, "D-A2": 1, "D-B1": 2} {
		q, err := s.GetQuota(code)
		if err != nil {
			t.Fatalf("定额 %s 不在库: %v", code, err)
		}
		if len(q.Items) != want {
			t.Errorf("定额 %s 应有 %d 行含量，得到 %d", code, want, len(q.Items))
		}
		if q.GCode == "" || q.GCode[:1] != "D" {
			t.Errorf("新定额应授 D 族自有码，得到 %q", q.GCode)
		}
	}
	// 定额单价按库内资源现行价现算（工料机预填价随簿落库）。
	for code, want := range map[string]float64{"D-A1": 438, "D-A2": 20, "D-B1": 27} {
		cp, _, err := s.ComposeQuota(code, nil)
		if err != nil {
			t.Fatalf("核算 %s 失败: %v", code, err)
		}
		if diff := cp.CompositePrice - want; diff > 0.001 || diff < -0.001 {
			t.Errorf("定额 %s 现算单价 = %.4f, want %.2f", code, cp.CompositePrice, want)
		}
	}

	// ③ 1:N 挂接：Q01 挂两条定额（系数 0.5 的块工程量=清单量×系数），Q02 挂一条。
	projects, _ := s.BillProjects()
	if len(projects) != 1 {
		t.Fatalf("应只有 1 个项目，得到 %+v", projects)
	}
	q01, err := s.FindBillItem(projects[0].ID, "Q01")
	if err != nil || q01 == nil {
		t.Fatalf("清单 Q01 不在库: %v", err)
	}
	if strings.Join(q01.QuotaCodes, ",") != "D-A1,D-A2" {
		t.Errorf("Q01 应挂 D-A1+D-A2，得到 %v", q01.QuotaCodes)
	}
	if q01.LinkQuantities["D-A1"] != 100 || q01.LinkQuantities["D-A2"] != 50 {
		t.Errorf("Q01 各定额工程量应 100 / 50（100×0.5），得到 %v", q01.LinkQuantities)
	}
	if q01.GCode == "" || q01.GCode[:1] != "Q" {
		t.Errorf("新清单行应授 Q 族自有码，得到 %q", q01.GCode)
	}
	q02, _ := s.FindBillItem(projects[0].ID, "Q02")
	if q02 == nil || strings.Join(q02.QuotaCodes, ",") != "D-B1" || q02.LinkQuantities["D-B1"] != 2000 {
		t.Errorf("Q02 应挂 D-B1、工程量 2000，得到 %+v", q02)
	}
	// 模版费率随簿落库（预填的默认取费口径）。
	if projects[0].ManagementRate != 0.10 || projects[0].TaxRate != 0.09 {
		t.Errorf("项目费率应取自模版费用汇总，得到 %+v", projects[0])
	}

	// ④ 幂等：同一份簿再导一遍，定额/挂接不增不减。
	res2 := s.ApplyProjectBundle(b)
	if len(res2.Errors) > 0 {
		t.Fatalf("重导入不应有错误：%v", res2.Errors)
	}
	links, _ := s.BillQuotaLinks(q01.ID)
	if len(links) != 2 {
		t.Errorf("重导入后 Q01 应仍是 2 条挂接，得到 %d", len(links))
	}
	if n := len(mustListQuotas(t, s, "")); n != 3 {
		t.Errorf("重导入后定额应仍是 3 条，得到 %d", n)
	}
}

// TestTemplateRecalcNumeric 填好的模版经 LibreOffice 真重算后的数值金样（全部
// 人工口算，不调用生产算式）：
//
//	Q01 = (0.1×300 + 1.02×400)×1 + (0.02×1000)×0.5 = 438 + 10 = 448
//	     三费：人工 30、材料 408、机械 20×0.5 = 10（三费之和 = 综合单价）
//	Q02 = 0.03×400 + 0.05×300 = 27
//	直接费 = 100×448 + 2000×27 = 98800；企管 10%、利润 (直接费+企管)×7%、规费 2%、增值税 9%
func TestTemplateRecalcNumeric(t *testing.T) {
	f := recalcBytes(t, filledTemplate(t))
	up := SheetUnitPriceName
	closeTo(t, "Q01 综合单价", readNum(t, f, up, "S4"), 448)
	closeTo(t, "Q01 人工（三费须按定额系数折算）", readNum(t, f, up, "O4"), 30)
	closeTo(t, "Q01 材料", readNum(t, f, up, "P4"), 408)
	closeTo(t, "Q01 机械（系数 0.5）", readNum(t, f, up, "Q4"), 10)
	closeTo(t, "定额 D-A1 单价", readNum(t, f, up, "M5"), 438)
	closeTo(t, "定额 D-A2 单价", readNum(t, f, up, "M10"), 20)
	closeTo(t, "定额 D-A2 对清单的贡献（×0.5）", readNum(t, f, up, "N10"), 10)
	closeTo(t, "Q02 综合单价", readNum(t, f, up, "S15"), 27)
	// 名称/单位自动带出（VLOOKUP 工料机价格表）。
	if v, _ := f.GetCellValue(up, "J6"); v != "普工" {
		t.Errorf("明细行资源名称应自动带出「普工」，得到 %q", v)
	}
	// 没用的组：综合单价为 0，不污染合计。
	closeTo(t, "未填的第 3 组综合单价", readNum(t, f, up, "S26"), 0)

	bill := SheetBillName
	closeTo(t, "清单 Q01 合价", readNum(t, f, bill, "I4"), 44800)
	closeTo(t, "清单 Q02 合价", readNum(t, f, bill, "I5"), 54000)
	closeTo(t, "未填的清单行合价", readNum(t, f, bill, "I6"), 0)

	fee := SheetSummaryName
	direct := readNum(t, f, fee, "E"+itoa(findRowByLabel(t, f, fee, "人材机直接费合计")))
	closeTo(t, "直接费合计", direct, 98800)
	mgmt := readNum(t, f, fee, "E"+itoa(findRowByLabel(t, f, fee, "企业管理费")))
	closeTo(t, "企业管理费", mgmt, 9880)
	profit := readNum(t, f, fee, "E"+itoa(findRowByLabel(t, f, fee, "利润")))
	closeTo(t, "利润", profit, (98800+9880)*0.07)
	reg := readNum(t, f, fee, "E"+itoa(findRowByLabel(t, f, fee, "规费")))
	closeTo(t, "规费", reg, 1976)
	preTax := readNum(t, f, fee, "E"+itoa(findRowByLabel(t, f, fee, "税前合计（不含税造价）")))
	closeTo(t, "税前合计（逐项相加，不得把直接费算重）", preTax, direct+mgmt+profit+reg)
	total := readNum(t, f, fee, "E"+itoa(findRowByLabel(t, f, fee, "含税总造价")))
	closeTo(t, "增值税", total-preTax, preTax*0.09)
}

// TestExportBlocksRoundTrip 带三级行组的 bundle 导出 → 再解析，定额块、系数、
// 明细归属原样还原；无明细的定额块导出后公式不得循环引用。
func TestExportBlocksRoundTrip(t *testing.T) {
	b := &ProjectBundle{
		Project: "往返项目", FileName: "rt.xlsx",
		Resources: []ProjectResource{
			{Code: "L01", Kind: "人工", Title: "普工", Unit: "工日", Price: 300},
			{Code: "C20", Kind: "材料", Title: "商品混凝土", Unit: "m³", Price: 400},
			{Code: "EXC", Kind: "机械", Title: "挖掘机", Unit: "台班", Price: 1000},
		},
		Items: []ProjectItem{{Code: "Q01", Title: "垫层", Unit: "m³", Quantity: 100}, {Code: "Q02", Title: "面层", Unit: "m²", Quantity: 2000}},
		Lines: []ProjectConsumption{
			{ItemCode: "Q01", ItemTitle: "垫层", QuotaCode: "D-A1", Kind: "人工", ResourceCode: "L01", Quantity: 0.1},
			{ItemCode: "Q01", ItemTitle: "垫层", QuotaCode: "D-A1", Kind: "材料", ResourceCode: "C20", Quantity: 1.02},
			{ItemCode: "Q01", ItemTitle: "垫层", QuotaCode: "D-A2", Kind: "机械", ResourceCode: "EXC", Quantity: 0.02},
			{ItemCode: "Q02", ItemTitle: "面层", QuotaCode: "D-B1", Kind: "材料", ResourceCode: "C20", Quantity: 0.03},
		},
		QuotaBlocks: []ProjectQuotaBlock{
			{ItemCode: "Q01", Code: "D-A1", Title: "垫层浇筑", Unit: "m³"},
			{ItemCode: "Q01", Code: "D-A2", Title: "垫层夯实", Unit: "m³", Factor: 0.5},
			{ItemCode: "Q02", Code: "D-B1", Title: "面层摊铺", Unit: "m²"},
			{ItemCode: "Q02", Code: "D-B2", Title: "空块（无明细）", Unit: "m²"},
		},
	}
	data, err := ExportProjectWorkbook(b, ExportOptions{})
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}

	// 结构：Q01 两块（2+1 条明细）、Q02 两块（1+0 条明细）；空块的定额单价是数值 0。
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("打开失败: %v", err)
	}
	up := SheetUnitPriceName
	for _, c := range []struct{ cell, want string }{
		{"E4", "汇总"}, {"E5", "定额"}, {"F5", "D-A1"}, {"E8", "定额"}, {"F8", "D-A2"}, {"H8", "0.5"},
		{"E9", "机械"}, {"E10", "汇总"}, {"E11", "定额"}, {"F11", "D-B1"}, {"E13", "定额"}, {"F13", "D-B2"},
	} {
		if got, _ := f.GetCellValue(up, c.cell); got != c.want {
			t.Errorf("%s!%s = %q, want %q", up, c.cell, got, c.want)
		}
	}
	// 空块：定额单价写数值 0（空区间 SUM 经 Excel 规范化会吞进定额行自身 → 循环引用）。
	if fo := formulaOf(f, up, "M13"); fo != "" {
		t.Errorf("无明细的定额块单价应为数值 0，得到公式 %q", fo)
	}
	// Q02 的三费只压有明细的块（D-B1），不含空块 D-B2。
	if o := formulaOf(f, up, "O10"); !strings.Contains(o, "H11)*SUMIF(E12:E12,") || strings.Contains(o, "H13)*") {
		t.Errorf("Q02 人工公式应只含 D-B1 块，得到 %q", o)
	}
	_ = f.Close()

	// 往返：再解析还原同样的块/系数/归属。
	dir := t.TempDir()
	path := filepath.Join(dir, "rt.xlsx")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	b2, err := ParseProjectWorkbook(path)
	if err != nil {
		t.Fatalf("再解析失败: %v", err)
	}
	if len(b2.QuotaBlocks) != 4 || len(b2.Lines) != len(b.Lines) || len(b2.Items) != len(b.Items) {
		t.Fatalf("往返后 块/明细/清单 = %d/%d/%d, want %d/%d/%d", len(b2.QuotaBlocks), len(b2.Lines), len(b2.Items), 4, len(b.Lines), len(b.Items))
	}
	for i, blk := range b2.QuotaBlocks {
		want := b.QuotaBlocks[i]
		if blk.ItemCode != want.ItemCode || blk.Code != want.Code || blk.Title != want.Title || blk.Unit != want.Unit || blk.Factor != want.Factor {
			t.Errorf("块[%d] 往返后 = %+v, want %+v", i, blk, want)
		}
	}
	for i, l := range b2.Lines {
		if l.QuotaCode != b.Lines[i].QuotaCode || l.ResourceCode != b.Lines[i].ResourceCode || l.Quantity != b.Lines[i].Quantity {
			t.Errorf("明细[%d] 往返后 = %+v, want %+v", i, l, b.Lines[i])
		}
	}
}

// mustListQuotas 定额列表（失败即 Fatal）。
func mustListQuotas(t *testing.T, s *Store, specialty string) []Quota {
	t.Helper()
	list, err := s.ListQuotas(specialty, "")
	if err != nil {
		t.Fatalf("读定额列表失败: %v", err)
	}
	return list
}

// TestFindHeaderRowIgnoresInstructionLine 表头探测不得被表头上方的填表说明带偏：
// 说明行把「清单编码」「行类」写进同一格时，按整行匹配会认成表头，所有列错位到
// 第 1 列（2026-11-25 测算模版往返用例抓到的真因）。
func TestFindHeaderRowIgnoresInstructionLine(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	const sheet = "综合单价"
	_ = f.SetSheetName(f.GetSheetName(0), sheet)
	_ = f.SetCellStr(sheet, "A1", "综合单价（只含人材机）。")
	_ = f.SetCellStr(sheet, "A2", "每组=1 汇总行+2 个定额块。汇总行只填 清单编码/名称/单位；明细行填 行类/资源编码/消耗量。")
	for i, h := range unitPriceCols {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRowUnitPrice)
		_ = f.SetCellStr(sheet, cell, h)
	}
	if got := findHeaderRow(f, sheet, maxProbeCols, 8, "清单编码", "行类"); got != headerRowUnitPrice {
		t.Fatalf("表头行 = %d, want %d（说明行被认成了表头）", got, headerRowUnitPrice)
	}
	header := rowValues(f, sheet, headerRowUnitPrice, maxProbeCols)
	for col, want := range map[string]int{"清单编码": 2, "行类": 5, "定额编码": 6, "资源编码": 9, "消耗量": 12} {
		if got := colIndex(header, false, col); got != want {
			t.Errorf("%s 列 = %d, want %d", col, got, want)
		}
	}
	// 只有一格有字的行永远不是表头；关键词挤在同一格的行也不是。
	if headerRowHasKeys([]string{"项目名称 工程量 单位"}, []string{"项目名称", "工程量"}) {
		t.Error("单格说明行不应被认成表头")
	}
	if !headerRowHasKeys([]string{"序号", "项目名称", "工程量"}, []string{"工程量", "项目名称"}) {
		t.Error("关键词落在不同列的多列表头应被认成表头（与书写顺序无关）")
	}
}
