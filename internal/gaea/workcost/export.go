package workcost

// export.go —— 五表项目工作簿的确定性导出。
//
// 实测模版（旺平矿业 / 什邡）的表间关系就是本文件的落点：
//
//	工料机价格  ：编码|类别|名称|规格|单位|不含税单价|来源|备注        ← 基础资源价
//	综合单价    ：汇总行 K=Σ明细、P=K、L/M/N=SUMIF(行类)；明细 K=I*J     ← 只含人材机
//	              明细 J = IFERROR(VLOOKUP(资源编码, 工料机价格!$A$4:$F$n,6,FALSE()),0)
//	费用汇总    ：H=综合单价!P<汇总行>、I=G*H；I 合计=ΣI；下半段取费在 E 列
//	              取费 =E<直接费>*D<费率>（税率段按累计基数）
//	封面        ：成果指标 =费用汇总!Exx
//	工程量计算  ：编码|名称|数值|单位|公式/口径|来源|去向|备注
//
// 关键决策：**导出用活公式而非算好的数值**。模版本身就是公式联动表
// （用户改费率/改资源价，全表自动重算），导出成死数字等于把工具降级成报表。
// 同时缓存值也写一份，未重算的 Excel 打开也能看到数。

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Sheet 名（导出固定，与实测模版一致）。
const (
	SheetCoverName     = "封面"
	SheetSummaryName   = "费用汇总"
	SheetUnitPriceName = "综合单价"
	SheetResourceName  = "工料机价格"
	SheetQuantityName  = "工程量计算"
)

// 各表表头行（与实测模版对齐：综合单价与工料机价格表头在第 3 行，
// 其下即数据行；费用汇总另有一行表头说明）。
const (
	headerRowCover     = 2
	headerRowUnitPrice = 3
	headerRowResource  = 3
	headerRowSummary   = 3
	headerRowQuantity  = 3
)

// 导出用列位（对齐实测模版，不随内容变化）。
var (
	unitPriceCols = []string{"序号", "清单编码", "项目名称", "单位", "行类",
		"资源编码", "资源名称", "消耗单位", "消耗量", "单价", "合价",
		"人工", "材料", "机械", "说明", "综合单价（人材机）"}
	// 命名带 export 前缀：store.go 已有包级 resourceCols（SQL 列清单常量），
	// 同名会 redeclared。
	exportResourceCols = []string{"编码", "类别", "名称", "规格/计算式", "单位",
		"不含税单价（元）", "来源", "备注"}
	summaryCols = []string{"序号", "分部", "编码", "项目名称", "工序特征", "单位",
		"工程量", "综合单价", "合价（元）", "单价位置", "依据"}
	quantityCols = []string{"编码", "名称", "数值", "单位", "公式/口径", "来源", "去向", "备注"}
	feeCols      = []string{"序号", "费用名称", "计算基数/说明", "费率", "金额（元）", "备注"}
)

// ExportOptions 导出可选附加内容。
type ExportOptions struct {
	// Project 项目名（写入封面标题；空则用 bundle 里的）。
	Project string
	// Location/Duration 封面基本信息覆盖。
	Location string
	Duration string
	// Pricing 组价口径自述（空则自动生成符合模版口径的一句）。
	Pricing string
}

// ExportProjectWorkbook 把项目 bundle 渲染成五表工作簿字节。
func ExportProjectWorkbook(b *ProjectBundle, opt ExportOptions) ([]byte, error) {
	if b == nil {
		return nil, fmt.Errorf("项目数据为空")
	}
	if len(b.Resources) == 0 && len(b.Lines) == 0 {
		return nil, fmt.Errorf("项目没有资源与消耗量，无法导出（先解析项目工作簿或录入工料机）")
	}

	f := excelize.NewFile()
	// excelize 无 SetSheetIndex（v2 API），sheet 顺序 = 新建顺序。故先把默认
	// sheet 改名成第一张「封面」，再按模版顺序依次 NewSheet；「综合单价」当前
	// 是默认 sheet，须在末尾补建并删除默认位——做法：默认 sheet 用作封面。
	if err := f.SetSheetName(f.GetSheetName(0), SheetCoverName); err != nil {
		return nil, err
	}
	for _, name := range []string{SheetSummaryName, SheetUnitPriceName, SheetResourceName, SheetQuantityName} {
		if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
	}

	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, err
	}
	money, err := f.NewStyle(&excelize.Style{NumFmt: 4}) // #,##0.00
	if err != nil {
		return nil, err
	}

	// 计算布局：综合单价表每个清单项占「1 汇总行 + 明细行」，先算出行号映射。
	layout := buildUnitPriceLayout(b)
	resourceLastRow := headerRowResource + len(b.Resources)
	priceRange := fmt.Sprintf("'%s'!$A$%d:$F$%d", SheetResourceName, headerRowResource+1, resourceLastRow)

	writeResourceSheet(f, b, bold, money)
	writeUnitPriceSheet(f, b, layout, priceRange, bold, money)
	summaryLayout := writeSummarySheet(f, b, layout, bold, money)
	writeCoverSheet(f, b, opt, summaryLayout, bold, money)
	writeQuantitySheet(f, b, bold, money)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ── 综合单价表布局 ──────────────────────────────────────────────────

// itemLayout 一个清单项在综合单价表里的行位置。
//
//	SummaryRow —— 「汇总」行（P=综合单价，K=Σ明细）
//	DetailRows —— 明细行行号（升序）
type itemLayout struct {
	Item       ProjectItem
	SummaryRow int
	DetailRows []int
}

// unitPriceLayout 全部清单项 + 明细总行数。
type unitPriceLayout struct {
	Items     []itemLayout
	TotalRows int // 表内最后一行行号
}

// buildUnitPriceLayout 按「每清单项先汇总行、后明细行」排布（对齐实测模版：
// 第 4 行 WP01 汇总、5-6 行明细；第 7 行 WP02 汇总、8-13 行明细…）。
func buildUnitPriceLayout(b *ProjectBundle) unitPriceLayout {
	byItem := map[string][]ProjectConsumption{}
	var order []string
	for _, l := range b.Lines {
		if _, ok := byItem[l.ItemCode]; !ok {
			order = append(order, l.ItemCode)
		}
		byItem[l.ItemCode] = append(byItem[l.ItemCode], l)
	}

	items := append([]ProjectItem(nil), b.Items...)
	seen := map[string]bool{}
	for _, it := range items {
		seen[it.Code] = true
	}
	for _, code := range order {
		if !seen[code] {
			items = append(items, ProjectItem{Code: code})
			seen[code] = true
		}
	}

	var out unitPriceLayout
	row := headerRowUnitPrice + 1
	for _, it := range items {
		il := itemLayout{Item: it, SummaryRow: row}
		row++
		for range byItem[it.Code] {
			il.DetailRows = append(il.DetailRows, row)
			row++
		}
		out.Items = append(out.Items, il)
	}
	out.TotalRows = row - 1
	return out
}

// ── 工料机价格表 ────────────────────────────────────────────────────

func writeResourceSheet(f *excelize.File, b *ProjectBundle, bold, money int) {
	sheet := SheetResourceName
	writeHeader(f, sheet, headerRowResource, exportResourceCols)
	styleRow(f, sheet, headerRowResource, len(exportResourceCols), bold)

	for i, r := range b.Resources {
		row := headerRowResource + 1 + i
		setCell(f, sheet, 1, row, r.Code)
		setCell(f, sheet, 2, row, firstNonEmpty(r.Kind, KindMaterial))
		setCell(f, sheet, 3, row, r.Title)
		// 规格列在实测模版里承载「计算式」（如「折旧900+柴油×90L+司机」）；
		// 台班公式原文优先展示，便于追溯口径。
		setCell(f, sheet, 4, row, firstNonEmpty(r.Spec, r.PriceFormula))
		setCell(f, sheet, 5, row, r.Unit)
		setCellNumber(f, sheet, 6, row, r.Price)
		setCell(f, sheet, 7, row, r.Source)
		setCell(f, sheet, 8, row, r.Note)
		applyStyle(f, sheet, 6, row, money)
	}
	// 表头说明（模版第 1-2 行）。
	setCell(f, sheet, 1, 1, "基础资源单价（F列为核算取用价）。台班费可用公式（折旧+柴油×油耗+人工）计算。管理费、利润、税金不在本表。")
	setCell(f, sheet, 1, 2, "综合单价明细按资源编码 VLOOKUP 本表 F 列；改资源价后，所有引用该资源的综合单价在下次核算时同步重算。")
}

// ── 综合单价表 ──────────────────────────────────────────────────────

func writeUnitPriceSheet(f *excelize.File, b *ProjectBundle, layout unitPriceLayout, priceRange string, bold, money int) {
	sheet := SheetUnitPriceName
	writeHeader(f, sheet, headerRowUnitPrice, unitPriceCols)
	styleRow(f, sheet, headerRowUnitPrice, len(unitPriceCols), bold)
	setCell(f, sheet, 1, 1, "综合单价（只含人材机，不含管理费/利润/税金）。汇总行 K=SUM(明细)、P=K、L/M/N=SUMIF(行类)；明细 J=VLOOKUP 工料机价格。")
	setCell(f, sheet, 1, 2, "消耗量按工序计算；外委独立行类但三费归集并入材料桶（材料 = SUMIF(材料)+SUMIF(外委)）。")

	// 明细行按清单项分组取用。
	byItem := map[string][]ProjectConsumption{}
	for _, l := range b.Lines {
		byItem[l.ItemCode] = append(byItem[l.ItemCode], l)
	}

	for _, il := range layout.Items {
		sr := il.SummaryRow
		details := byItem[il.Item.Code]

		// 汇总行身份列。
		setCell(f, sheet, 1, sr, firstNonEmpty(il.Item.Code, ""))
		setCell(f, sheet, 2, sr, il.Item.Code)
		setCell(f, sheet, 3, sr, il.Item.Title)
		setCell(f, sheet, 4, sr, il.Item.Unit)
		setCell(f, sheet, 5, sr, "汇总")
		setCell(f, sheet, 15, sr, "人材机小计；综合单价见P列（不含管利税）")

		first, last := sr+1, sr
		if len(il.DetailRows) > 0 {
			first = il.DetailRows[0]
			last = il.DetailRows[len(il.DetailRows)-1]
		}
		// K = Σ明细合价；L/M/N = SUMIF(行类)；外委并入材料桶（实测模版口径）。
		kRange := fmt.Sprintf("K%d:K%d", first, last)
		setFormula(f, sheet, 11, sr, fmt.Sprintf("SUM(%s)", kRange))
		setFormula(f, sheet, 12, sr, fmt.Sprintf(`SUMIF(E%d:E%d,"人工",%s)`, first, last, kRange))
		setFormula(f, sheet, 13, sr, fmt.Sprintf(`SUMIF(E%d:E%d,"材料",%s)+SUMIF(E%d:E%d,"外委",%s)`,
			first, last, kRange, first, last, kRange))
		setFormula(f, sheet, 14, sr, fmt.Sprintf(`SUMIF(E%d:E%d,"机械",%s)`, first, last, kRange))
		// P = K（综合单价 = 人材机合价）。
		setFormula(f, sheet, 16, sr, fmt.Sprintf("K%d", sr))

		// 明细行。
		for i, l := range details {
			row := il.DetailRows[i]
			setCell(f, sheet, 1, row, l.Seq)
			setCell(f, sheet, 2, row, l.ItemCode)
			setCell(f, sheet, 3, row, l.ItemTitle)
			setCell(f, sheet, 4, row, l.ItemUnit)
			setCell(f, sheet, 5, row, firstNonEmpty(l.Kind, KindMaterial))
			setCell(f, sheet, 6, row, l.ResourceCode)
			setCell(f, sheet, 7, row, l.ResourceTitle)
			setCell(f, sheet, 8, row, l.Unit)
			setCellNumber(f, sheet, 9, row, l.Quantity)
			// J = VLOOKUP(资源编码, 工料机价格!$A$4:$F$n, 6, FALSE)（带 IFERROR 兜底 0）。
			setFormula(f, sheet, 10, row,
				fmt.Sprintf(`IFERROR(VLOOKUP(F%d,%s,6,FALSE()),0)`, row, priceRange))
			// K = I*J。
			setFormula(f, sheet, 11, row, fmt.Sprintf("I%d*J%d", row, row))
			applyStyle(f, sheet, 10, row, money)
			applyStyle(f, sheet, 11, row, money)
		}
		applyStyle(f, sheet, 11, sr, money)
		applyStyle(f, sheet, 12, sr, money)
		applyStyle(f, sheet, 13, sr, money)
		applyStyle(f, sheet, 14, sr, money)
		applyStyle(f, sheet, 16, sr, money)
	}
}

// ── 费用汇总表布局 ──────────────────────────────────────────────────

// summaryLayout 费用汇总表的关键行号（供封面公式引用）。
type summaryLayout struct {
	ItemFirstRow  int
	ItemLastRow   int
	DirectFeeRow  int // 人材机直接费合计行
	MgmtRow       int // 企业管理费
	ProfitRow     int
	RegulatoryRow int
	PreTaxRow     int // 税前合计
	TaxRow        int // 增值税
	TotalRow      int // 含税总造价
	ControlRow    int // 招标控制价
}

func writeSummarySheet(f *excelize.File, b *ProjectBundle, layout unitPriceLayout, bold, money int) summaryLayout {
	sheet := SheetSummaryName
	setCell(f, sheet, 1, 1, "费用汇总。上半是工程量×综合单价；下半管理费、利润、税金不进综合单价。")
	setCell(f, sheet, 1, 2, "综合单价列引用「综合单价」表 P 列；改费率只改下半段 D 列。")
	writeHeader(f, sheet, headerRowSummary, summaryCols)
	styleRow(f, sheet, headerRowSummary, len(summaryCols), bold)

	var sl summaryLayout
	sl.ItemFirstRow = headerRowSummary + 1
	row := sl.ItemFirstRow
	for i, il := range layout.Items {
		setCell(f, sheet, 1, row, fmt.Sprintf("%d", i+1))
		setCell(f, sheet, 2, row, il.Item.Division)
		setCell(f, sheet, 3, row, il.Item.Code)
		setCell(f, sheet, 4, row, il.Item.Title)
		setCell(f, sheet, 5, row, il.Item.Feature)
		setCell(f, sheet, 6, row, il.Item.Unit)
		setCellNumber(f, sheet, 7, row, il.Item.Quantity)
		// H = 综合单价!P<汇总行>
		setFormula(f, sheet, 8, row, fmt.Sprintf("'%s'!P%d", SheetUnitPriceName, il.SummaryRow))
		// I = G*H
		setFormula(f, sheet, 9, row, fmt.Sprintf("G%d*H%d", row, row))
		setCell(f, sheet, 10, row, fmt.Sprintf("'%s'!P%d", SheetUnitPriceName, il.SummaryRow))
		applyStyle(f, sheet, 8, row, money)
		applyStyle(f, sheet, 9, row, money)
		row++
	}
	sl.ItemLastRow = row - 1

	// 人材机直接费合计（E 列 = ΣI，与实测模版 E52/E38 口径一致）。
	sl.DirectFeeRow = row
	setCell(f, sheet, 4, row, "人材机直接费合计")
	setFormula(f, sheet, 5, row, fmt.Sprintf("SUM(I%d:I%d)", sl.ItemFirstRow, sl.ItemLastRow))
	styleRow(f, sheet, row, len(summaryCols), bold)
	applyStyle(f, sheet, 5, row, money)
	row += 2 // 空一行

	// 取费区表头。
	writeHeader(f, sheet, row, feeCols)
	styleRow(f, sheet, row, len(feeCols), bold)
	row++

	directRef := fmt.Sprintf("E%d", sl.DirectFeeRow)

	// 一、直接费（引用上半段合计；无费率，故走普通写行）。
	setCell(f, sheet, 1, row, "一")
	setCell(f, sheet, 2, row, "分部分项人材机直接费合计")
	setCell(f, sheet, 3, row, fmt.Sprintf("上表 E%d", sl.DirectFeeRow))
	setCell(f, sheet, 4, row, "—")
	setFormula(f, sheet, 5, row, directRef)
	setCell(f, sheet, 6, row, "综合单价不含管利税")
	applyStyle(f, sheet, 5, row, money)
	row++

	// 二、企业管理费 = 直接费 × 管理费率。
	sl.MgmtRow = row
	mgmtRef := fmt.Sprintf("E%d", row)
	writeRateRow(f, sheet, row, "二", "企业管理费", "直接费 × 管理费率",
		b.Fee.ManagementRate, fmt.Sprintf("%s*D%d", directRef, row), "不进综合单价")
	row++

	// 三、利润 =（直接费+管理费）× 利润率。
	sl.ProfitRow = row
	profitRef := fmt.Sprintf("E%d", row)
	writeRateRow(f, sheet, row, "三", "利润", "（直接费+管理费）× 利润率",
		b.Fee.ProfitRate, fmt.Sprintf("(%s+%s)*D%d", directRef, mgmtRef, row), "不进综合单价")
	row++

	// 四、规费 = 直接费 × 规费率（各专业口径不同，可为 0）。
	sl.RegulatoryRow = row
	regRef := fmt.Sprintf("E%d", row)
	writeRateRow(f, sheet, row, "四", "规费", "直接费 × 规费率",
		b.Fee.RegulatoryRate, fmt.Sprintf("%s*D%d", directRef, row), "本测算按项目口径计取")
	row++

	// 五、税前合计 = 直接费+管理费+利润+规费。
	//
	// ⚠️ 必须**显式逐项相加**，不能用 SUM(直接费行:规费行)——区间的起点就是
	// 直接费行，SUM 会把它连同后面各项一起再算一遍（实测踩到：税前合计恰好
	// 多出一个直接费 + 一个管理费）。这类「区间起点即被加项」的错在结构测试里
	// 看不出来（公式形态看着合理），只有重算数值才暴露。
	sl.PreTaxRow = row
	preTaxRef := fmt.Sprintf("E%d", row)
	setCell(f, sheet, 1, row, "五")
	setCell(f, sheet, 2, row, "税前合计（不含税造价）")
	setCell(f, sheet, 3, row, "直接费+管理费+利润+规费")
	setCell(f, sheet, 4, row, "—")
	setFormula(f, sheet, 5, row, fmt.Sprintf("%s+%s+%s+%s", directRef, mgmtRef, profitRef, regRef))
	setCell(f, sheet, 6, row, "措施费/暂列金额如有需另行列示")
	applyStyle(f, sheet, 5, row, money)
	row++

	// 六、增值税 = 税前合计 × 税率。
	sl.TaxRow = row
	taxRef := fmt.Sprintf("E%d", row)
	writeRateRow(f, sheet, row, "六", "增值税", "税前合计 × 增值税率",
		b.Fee.TaxRate, fmt.Sprintf("%s*D%d", preTaxRef, row), "建筑服务一般计税 9% / 简易计税 3%")
	row++

	// 七、含税总造价。
	sl.TotalRow = row
	setCell(f, sheet, 1, row, "七")
	setCell(f, sheet, 2, row, "含税总造价")
	setCell(f, sheet, 3, row, "税前合计+增值税")
	setCell(f, sheet, 4, row, "—")
	setFormula(f, sheet, 5, row, fmt.Sprintf("%s+%s", preTaxRef, taxRef))
	setCell(f, sheet, 6, row, "本工程含税造价")
	styleRow(f, sheet, row, len(feeCols), bold)
	applyStyle(f, sheet, 5, row, money)
	row++

	// 八、招标控制价（若给出）。
	sl.ControlRow = row
	setCell(f, sheet, 1, row, "八")
	setCell(f, sheet, 2, row, "招标控制价（含税）")
	setCell(f, sheet, 3, row, "招标文件口径")
	setCell(f, sheet, 4, row, "—")
	if b.Fee.ControlPrice > 0 {
		setCellNumber(f, sheet, 5, row, b.Fee.ControlPrice)
	} else {
		setCell(f, sheet, 5, row, "—")
	}
	setCell(f, sheet, 6, row, "投标不得超过，否则废标")
	row++
	applyStyle(f, sheet, 5, sl.ControlRow, money)
	return sl
}

// rateFromFeeHeader 已不再需要（费率写数值，读者改 D 列即可）——保留说明：
// 实测模版的费率就在 D 列硬编码数值（0.05 / 0 / 0.03），不引用别处，故导出
// 直接写数值，用户改 D 列即全表重算。

// writeRateRow 写一行**取费**行（含费率）：序号/名称/基数说明/费率/金额公式/备注。
func writeRateRow(f *excelize.File, sheet string, row int, seq, name, base string, rate float64, amountFormula, note string) {
	setCell(f, sheet, 1, row, seq)
	setCell(f, sheet, 2, row, name)
	setCell(f, sheet, 3, row, base)
	setCellNumber(f, sheet, 4, row, rate)
	setFormula(f, sheet, 5, row, amountFormula)
	setCell(f, sheet, 6, row, note)
}

// ── 封面 ────────────────────────────────────────────────────────────

func writeCoverSheet(f *excelize.File, b *ProjectBundle, opt ExportOptions, sl summaryLayout, bold, money int) {
	sheet := SheetCoverName
	title := firstNonEmpty(opt.Project, b.Project, "项目")
	loc := firstNonEmpty(opt.Location, b.Location)
	dur := firstNonEmpty(opt.Duration, b.Duration)
	pricing := firstNonEmpty(opt.Pricing, b.Pricing,
		"综合单价=工序消耗量×基础资源价（只含人材机）；管理费、利润、税金在费用汇总下半段单列。")

	setCell(f, sheet, 1, headerRowCover, title)
	styleRow(f, sheet, headerRowCover, 7, bold)
	setCell(f, sheet, 1, headerRowCover+1, "五表模版：封面 → 费用汇总 → 综合单价（只含人材机）→ 工料机价格 → 工程量计算。")
	if loc != "" {
		setCell(f, sheet, 1, headerRowCover+2, "地点："+loc)
	}
	if dur != "" {
		setCell(f, sheet, 1, headerRowCover+3, "工期："+dur)
	}
	setCell(f, sheet, 1, headerRowCover+4, "组价口径："+pricing)

	// 成果指标（引用费用汇总，保持公式联动）。
	r := headerRowCover + 6
	setCell(f, sheet, 1, r, "测算成果")
	styleRow(f, sheet, r, 4, bold)
	r++
	setCell(f, sheet, 1, r, "指标")
	setCell(f, sheet, 2, r, "金额 / 指标")
	setCell(f, sheet, 3, r, "口径")
	styleRow(f, sheet, r, 3, bold)
	r++
	coverRows := []struct {
		label string
		ref   string
		note  string
	}{
		{"含税工程造价", fmt.Sprintf("'%s'!E%d", SheetSummaryName, sl.TotalRow), "费用汇总：直接费+管理费+利润+规费+增值税"},
		{"人材机直接费", fmt.Sprintf("'%s'!E%d", SheetSummaryName, sl.DirectFeeRow), "清单合价合计（综合单价不含管利税）"},
		{"企业管理费", fmt.Sprintf("'%s'!E%d", SheetSummaryName, sl.MgmtRow), "直接费 × 管理费率"},
		{"利润", fmt.Sprintf("'%s'!E%d", SheetSummaryName, sl.ProfitRow), "（直接费+管理费）× 利润率"},
		{"规费", fmt.Sprintf("'%s'!E%d", SheetSummaryName, sl.RegulatoryRow), "直接费 × 规费率"},
		{"税前合计（不含税造价）", fmt.Sprintf("'%s'!E%d", SheetSummaryName, sl.PreTaxRow), "直接费+管理费+利润+规费"},
		{"增值税", fmt.Sprintf("'%s'!E%d", SheetSummaryName, sl.TaxRow), "税前合计 × 税率"},
	}
	for _, cr := range coverRows {
		setCell(f, sheet, 1, r, cr.label)
		setFormula(f, sheet, 2, r, cr.ref)
		setCell(f, sheet, 3, r, cr.note)
		applyStyle(f, sheet, 2, r, money)
		r++
	}
	if b.Fee.ControlPrice > 0 {
		setCell(f, sheet, 1, r, "招标控制价（含税）")
		setFormula(f, sheet, 2, r, fmt.Sprintf("'%s'!E%d", SheetSummaryName, sl.ControlRow))
		setCell(f, sheet, 3, r, "投标不得超过")
		applyStyle(f, sheet, 2, r, money)
	}
}

// ── 工程量计算表 ────────────────────────────────────────────────────

func writeQuantitySheet(f *excelize.File, b *ProjectBundle, bold, money int) {
	sheet := SheetQuantityName
	setCell(f, sheet, 1, 1, "工程量计算（黄底为输入；费用汇总工程量引用本表）。")
	writeHeader(f, sheet, headerRowQuantity, quantityCols)
	styleRow(f, sheet, headerRowQuantity, len(quantityCols), bold)
	for i, q := range b.Quantities {
		row := headerRowQuantity + 1 + i
		setCell(f, sheet, 1, row, q.Code)
		setCell(f, sheet, 2, row, q.Title)
		setCellNumber(f, sheet, 3, row, q.Value)
		setCell(f, sheet, 4, row, q.Unit)
		setCell(f, sheet, 5, row, "输入")
		setCell(f, sheet, 8, row, q.Note)
	}
}

// ── 单元格写入小工具 ────────────────────────────────────────────────

func setCell(f *excelize.File, sheet string, col, row int, v string) {
	if strings.TrimSpace(v) == "" {
		return
	}
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return
	}
	_ = f.SetCellStr(sheet, cell, v)
}

func setCellNumber(f *excelize.File, sheet string, col, row int, v float64) {
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return
	}
	_ = f.SetCellFloat(sheet, cell, v, -1, 64)
}

func setFormula(f *excelize.File, sheet string, col, row int, formula string) {
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return
	}
	_ = f.SetCellFormula(sheet, cell, formula)
}

func writeHeader(f *excelize.File, sheet string, row int, cols []string) {
	for i, h := range cols {
		setCell(f, sheet, i+1, row, h)
	}
}

func styleRow(f *excelize.File, sheet string, row, cols, style int) {
	if cols <= 0 {
		return
	}
	last, err := excelize.ColumnNumberToName(cols)
	if err != nil {
		return
	}
	first, err := excelize.CoordinatesToCellName(1, row)
	if err != nil {
		return
	}
	end, err := excelize.CoordinatesToCellName(cols, row)
	if err != nil {
		return
	}
	_ = last
	_ = f.SetCellStyle(sheet, first, end, style)
}

func applyStyle(f *excelize.File, sheet string, col, row, style int) {
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return
	}
	_ = f.SetCellStyle(sheet, cell, cell, style)
}
