package workcost

// render.go — 六表测算工作簿的唯一渲染器（2026-11-25 用户拍板：模版五表改成六表，
// 费用汇总拆成 清单表 + 费用汇总表；综合单价是「清单 → 定额 → 工料机」三级行组）。
//
// 导出与空白测算模版共用这一个渲染器：空白模版就是「空清单 + 预填工料机 + 若干
// 空白组」的一次渲染。此前两处各写一套版式（封面 / 清单表 / 取费链各两份，靠人肉
// 对齐），合并后版式只有这一处——解析端（import 链）认的就是这里写出来的形态。
//
// 分工：本文件只管「怎么把数据摆成表」——表头、行组、公式链。测算的行业口径与
// 填法（每列含义、系数怎么取、常见错法）不在这里，在 cost-estimate 技能。

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Sheet 名（导出、模版与解析共用一套）。
const (
	SheetCoverName     = "封面"
	SheetBillName      = "清单"
	SheetSummaryName   = "费用汇总"
	SheetUnitPriceName = "综合单价"
	SheetResourceName  = "工料机价格"
	SheetQuantityName  = "工程量计算"
)

// 表头行行号（写侧的排布基准；解析端按表头名定位，不依赖行号）。
// 清单表与费用汇总表共用 headerRowSummary。
const (
	headerRowCover     = 2
	headerRowUnitPrice = 3
	headerRowResource  = 3
	headerRowSummary   = 3
	headerRowQuantity  = 3
)

// priceSheetCols 工料机价格表列（自有码列在编码后 → 单价在 G 列）。
var priceSheetCols = []string{"编码", "自有码", "类别", "名称", "规格/计算式", "单位",
	"不含税单价（元）", "来源", "备注"}

// unitPriceCols 综合单价表列（三级行组：行类后插 定额编码/定额名称/系数）。
var unitPriceCols = []string{"序号", "清单编码", "项目名称", "单位", "行类",
	"定额编码", "定额名称", "系数", "资源编码", "资源名称", "消耗单位", "消耗量",
	"单价", "合价", "人工", "材料", "机械", "说明", "综合单价（人材机）"}

// billSheetCols 清单表列（纯清单行；费用汇总只留累计后的取费链）。
var billSheetCols = []string{"序号", "分部", "编码", "项目名称", "工序特征", "单位",
	"工程量", "综合单价", "合价（元）"}

// feeCols 费用汇总表列（只有累计后的取费，没有清单区）。
var feeCols = []string{"序号", "费用名称", "计算基数/说明", "费率", "金额（元）", "备注"}

// quantityCols 工程量计算表列。
var quantityCols = []string{"编码", "名称", "数值", "单位", "公式/口径", "来源", "去向", "备注"}

// 综合单价表列位（19 列的字母序，公式引用用）。
const (
	tupColItemCode   = 2  // B 清单编码
	tupColKind       = 5  // E 行类
	tupColQuotaCode  = 6  // F 定额编码
	tupColQuotaTitle = 7  // G 定额名称
	tupColFactor     = 8  // H 系数
	tupColResCode    = 9  // I 资源编码
	tupColResTitle   = 10 // J 资源名称
	tupColResUnit    = 11 // K 消耗单位
	tupColQty        = 12 // L 消耗量
	tupColPrice      = 13 // M 单价
	tupColAmount     = 14 // N 合价
	tupColLabor      = 15 // O 人工
	tupColMaterial   = 16 // P 材料
	tupColMachine    = 17 // Q 机械
	tupColNote       = 18 // R 说明
	tupColUnitPrice  = 19 // S 综合单价
)

// summaryLayout 费用汇总表的关键行号（供封面指标引用）。
type summaryLayout struct {
	DirectFeeRow  int // 人材机直接费合计行
	MgmtRow       int // 企业管理费
	ProfitRow     int
	RegulatoryRow int
	PreTaxRow     int // 税前合计
	TaxRow        int // 增值税
	TotalRow      int // 含税总造价
	ControlRow    int // 招标控制价
}

// ── 渲染输入 ──────────────────────────────────────────────────────────

// resourceRow 工料机价格表的一行。
type resourceRow struct {
	Code         string
	GCode        string // 库内自有码（有则写；解析回读时带上）
	Kind         string
	Title        string
	Spec         string
	Unit         string
	Price        float64
	Source, Note string
}

// workbookBlock 一个定额块及其明细行。DetailCount 是明细行数——空白块给预置行数，
// 不带定额码与明细。
type workbookBlock struct {
	Quota       ProjectQuotaBlock
	Lines       []ProjectConsumption
	DetailCount int
}

// workbookItem 一个清单项及其定额块。
type workbookItem struct {
	Item   ProjectItem
	Blocks []workbookBlock
}

// workbookModel 六表渲染的输入。导出 = 解析结果装进来；空白模版 = 无清单 +
// BlankGroups 个空白组 + 预填工料机。
type workbookModel struct {
	// Title 封面标题（空=「项目」）。
	Title string
	// Scope 封面口径三件：专业 / 地区 / 计价期（空=不写）。导出与模版都不替解析器
	// 填 bundle 的口径——文件里写了什么口径由读表的人或模型提给导入。
	Scope      [3]string
	CoverLines []string // 封面说明行（逐行写在口径三件之后）
	// Indicators 封面带「测算成果」指标块（跨表引用费用汇总；导出用）。
	Indicators bool
	// Resources 工料机价格表的预填行。
	Resources []resourceRow
	// PriceBlank 工料机价格表在预填行之后预留的空行（空白模版：VLOOKUP 区间
	// 覆盖它，后来加的行也查得到价）。
	PriceBlank int
	// Items 清单项（含定额块与明细）。
	Items []workbookItem
	// BlankGroups 清单之后追加的空白组数（空白模版）。
	BlankGroups  int
	Fee          RateSet
	ControlPrice float64
	Quantities   []ProjectQuantity
	// Blank 空白模版态：录入列加下拉，明细行的名称/单位用公式带出。
	Blank bool
}

// renderWorkbook 把渲染输入摆成六表工作簿字节。
func renderWorkbook(m workbookModel) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	// excelize 无 SetSheetIndex（v2 API），sheet 顺序 = 新建顺序：默认 sheet 用作封面。
	if err := f.SetSheetName(f.GetSheetName(0), SheetCoverName); err != nil {
		return nil, err
	}
	for _, name := range []string{SheetBillName, SheetSummaryName, SheetUnitPriceName, SheetResourceName, SheetQuantityName} {
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

	// 工料机价格表的取价区间（综合单价明细 VLOOKUP 引用）：自有码列插在编码后，
	// 单价在 G 列（区间第 7 列）。
	priceRange := fmt.Sprintf("'%s'!$A$%d:$G$%d", SheetResourceName, headerRowResource+1,
		headerRowResource+len(m.Resources)+m.PriceBlank)
	layout := planRows(m)

	writePriceSheet(f, m, bold, money)
	if err := writeUnitPriceSheet(f, m, layout, priceRange, bold, money); err != nil {
		return nil, err
	}
	billRows := writeBillSheet(f, m, layout, bold, money)
	feeLayout := writeFeeSheet(f, m.Fee, m.ControlPrice, billRows, bold, money)
	writeCoverSheet(f, m, feeLayout, bold, money)
	writeQuantitySheet(f, m, bold)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ── 综合单价表布局 ──────────────────────────────────────────────────

// blockSpan 一个定额块在综合单价表里的行区间：定额行 + 其明细行（可无明细）。
type blockSpan struct {
	QuotaRow int
	First    int // 首条明细行
	Last     int // 末条明细行
}

func (sp blockSpan) hasDetail() bool { return sp.First > 0 && sp.Last >= sp.First }

// itemLayout 一个清单项在综合单价表里的行位置：汇总行 + 各定额块。
type itemLayout struct {
	Item       ProjectItem
	SummaryRow int
	Blocks     []blockSpan
}

// unitPriceLayout 全部清单项（空白组在最后）的行位置。
type unitPriceLayout struct {
	Items     []itemLayout
	TotalRows int // 表内最后一行行号
}

// planRows 按「每清单项：汇总行 →（定额行 → 明细行）×块数」排布；空白组每个
// 块给预置的明细行数（没有定额码与明细）。
func planRows(m workbookModel) unitPriceLayout {
	var out unitPriceLayout
	row := headerRowUnitPrice + 1
	add := func(it ProjectItem, blocks []workbookBlock) {
		il := itemLayout{Item: it, SummaryRow: row}
		row++
		for _, wb := range blocks {
			bl := blockSpan{QuotaRow: row}
			row++
			for i := 0; i < wb.DetailCount; i++ {
				if bl.First == 0 {
					bl.First = row
				}
				bl.Last = row
				row++
			}
			il.Blocks = append(il.Blocks, bl)
		}
		out.Items = append(out.Items, il)
	}
	for _, wi := range m.Items {
		add(wi.Item, wi.Blocks)
	}
	for g := 0; g < m.BlankGroups; g++ {
		blocks := make([]workbookBlock, TemplateQuotaBlocksPerGroup)
		for i := range blocks {
			blocks[i].DetailCount = TemplateDetailRowsPerBlock
		}
		add(ProjectItem{}, blocks)
	}
	out.TotalRows = row - 1
	return out
}

// coefExpr 定额系数表达式：系数空=1（每清单单位含 1 个定额单位）。
func coefExpr(quotaRow int) string {
	return fmt.Sprintf(`IF(H%d="",1,H%d)`, quotaRow, quotaRow)
}

// blockSumIF 块内明细按行类求和：SUMIF(E,"行类",N)；多个行类相加（外委并入材料桶）。
func blockSumIF(sp blockSpan, kinds ...string) string {
	terms := make([]string, 0, len(kinds))
	for _, k := range kinds {
		terms = append(terms, fmt.Sprintf(`SUMIF(E%d:E%d,%q,N%d:N%d)`, sp.First, sp.Last, k, sp.First, sp.Last))
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return "(" + strings.Join(terms, "+") + ")"
}

// threeFeeFormula 清单汇总行的三费公式：Σ各块 系数×(块内该行类的明细合价)。
//
// 不能直接对全部明细行 SUMIF——定额系数≠1 的块，它的明细合价是**定额单位**口径，
// 对清单的贡献要乘系数。直接 SUMIF 会让「三费之和 ≠ 综合单价」（2026-11-25 手算
// 金样时抓到：系数 0.5 的机械块综合单价算 10、三费却显示 20）。无明细的块不进
// 公式（空区间经 Excel 规范化会吞进定额行自身 → 循环引用）。
func threeFeeFormula(spans []blockSpan, kinds ...string) string {
	var parts []string
	for _, sp := range spans {
		if sp.hasDetail() {
			parts = append(parts, coefExpr(sp.QuotaRow)+"*"+blockSumIF(sp, kinds...))
		}
	}
	if len(parts) == 0 {
		return "0"
	}
	return strings.Join(parts, "+")
}

// writeGroupSummary 写清单汇总行：行类=「汇总」、N=Σ定额块贡献、O/P/Q=三费、S=N。
func writeGroupSummary(f *excelize.File, sheet string, sr int, spans []blockSpan, money int) {
	setCell(f, sheet, tupColKind, sr, "汇总")
	first, last := sr+1, sr
	if n := len(spans); n > 0 {
		tail := spans[n-1]
		first, last = spans[0].QuotaRow, tail.QuotaRow
		if tail.hasDetail() {
			last = tail.Last
		}
	}
	setFormula(f, sheet, tupColAmount, sr, fmt.Sprintf(`SUMIF(E%d:E%d,"定额",N%d:N%d)`, first, last, first, last))
	setFormula(f, sheet, tupColLabor, sr, threeFeeFormula(spans, "人工"))
	setFormula(f, sheet, tupColMaterial, sr, threeFeeFormula(spans, "材料", "外委"))
	setFormula(f, sheet, tupColMachine, sr, threeFeeFormula(spans, "机械"))
	setFormula(f, sheet, tupColUnitPrice, sr, fmt.Sprintf("N%d", sr))
	for c := tupColAmount; c <= tupColMachine; c++ {
		applyStyle(f, sheet, c, sr, money)
	}
	applyStyle(f, sheet, tupColUnitPrice, sr, money)
}

// writeBlockRows 写一个定额块的公式：定额行 M=Σ明细 N（定额单价）、N=M×系数；
// 明细行 M=VLOOKUP 资源价、N=L×M；lookupNames 时 J/K 也用 VLOOKUP 带出名称与
// 消耗单位（空白模版——录入资源编码即自动带出；导出写静态值保住源文件原文）。
func writeBlockRows(f *excelize.File, sheet string, qr int, detailRows []int, priceRange string, money int, lookupNames bool) blockSpan {
	sp := blockSpan{QuotaRow: qr}
	if n := len(detailRows); n > 0 {
		sp.First, sp.Last = detailRows[0], detailRows[n-1]
	}
	setCell(f, sheet, tupColKind, qr, "定额")
	if sp.hasDetail() {
		setFormula(f, sheet, tupColPrice, qr, fmt.Sprintf("SUM(N%d:N%d)", sp.First, sp.Last))
	} else {
		setCellNumber(f, sheet, tupColPrice, qr, 0) // 无明细的块：定额单价 0（空区间 SUM 会循环引用）
	}
	setFormula(f, sheet, tupColAmount, qr, fmt.Sprintf("%s*M%d", coefExpr(qr), qr))
	applyStyle(f, sheet, tupColPrice, qr, money)
	applyStyle(f, sheet, tupColAmount, qr, money)
	for _, r := range detailRows {
		if lookupNames {
			setFormula(f, sheet, tupColResTitle, r, fmt.Sprintf(`IFERROR(VLOOKUP(I%d,%s,4,FALSE()),"")`, r, priceRange))
			setFormula(f, sheet, tupColResUnit, r, fmt.Sprintf(`IFERROR(VLOOKUP(I%d,%s,6,FALSE()),"")`, r, priceRange))
		}
		setFormula(f, sheet, tupColPrice, r, fmt.Sprintf(`IFERROR(VLOOKUP(I%d,%s,7,FALSE()),0)`, r, priceRange))
		setFormula(f, sheet, tupColAmount, r, fmt.Sprintf("L%d*M%d", r, r))
		applyStyle(f, sheet, tupColPrice, r, money)
		applyStyle(f, sheet, tupColAmount, r, money)
	}
	return sp
}

// ── 各表 ──────────────────────────────────────────────────────────────

// writePriceSheet 工料机价格表：预填行 + 空白模版的类别下拉。
func writePriceSheet(f *excelize.File, m workbookModel, bold, money int) {
	sheet := SheetResourceName
	writeHeader(f, sheet, headerRowResource, priceSheetCols)
	styleRow(f, sheet, headerRowResource, len(priceSheetCols), bold)
	for i, r := range m.Resources {
		row := headerRowResource + 1 + i
		setCell(f, sheet, 1, row, r.Code)
		setCell(f, sheet, 2, row, r.GCode)
		setCell(f, sheet, 3, row, firstNonEmpty(r.Kind, KindMaterial))
		setCell(f, sheet, 4, row, r.Title)
		setCell(f, sheet, 5, row, r.Spec)
		setCell(f, sheet, 6, row, r.Unit)
		setCellNumber(f, sheet, 7, row, r.Price)
		setCell(f, sheet, 8, row, r.Source)
		setCell(f, sheet, 9, row, r.Note)
		applyStyle(f, sheet, 7, row, money)
	}
	setCell(f, sheet, 1, 1, "基础资源单价（G 列为核算取用价）。台班费可用公式（折旧+柴油×油耗+人工）计算。管理费、利润、税金不在本表。")
	setCell(f, sheet, 1, 2, "综合单价明细按资源编码 VLOOKUP 本表 G 列取价；改价后引用它的综合单价随之重算。「自有码」是造价库给这一行发的库内码，导回库时靠它对上行，不要改。")
	if !m.Blank {
		return
	}
	dv := excelize.NewDataValidation(true)
	dv.SetSqref(fmt.Sprintf("C%d:C%d", headerRowResource+1, headerRowResource+len(m.Resources)+m.PriceBlank))
	if err := dv.SetDropList([]string{"人工", "材料", "机械", "外委"}); err != nil {
		return
	}
	_ = f.AddDataValidation(sheet, dv)
}

// writeUnitPriceSheet 综合单价表（三级行组）。
//
// 公式链（列位见 tupCol* 常量）：
//
//	明细行  M=VLOOKUP 资源价、N=L×M、J/K=VLOOKUP 带出资源名称/消耗单位（空白模版）
//	定额行  M=Σ块内明细 N（定额单价）、N=M×IF(H="",1,H)（×系数=对清单综合单价的贡献）
//	汇总行  N=SUMIF(E,"定额",N)、O/P/Q=Σ块 系数×SUMIF(人工/材料+外委/机械)、S=N
func writeUnitPriceSheet(f *excelize.File, m workbookModel, layout unitPriceLayout, priceRange string, bold, money int) error {
	sheet := SheetUnitPriceName
	writeHeader(f, sheet, headerRowUnitPrice, unitPriceCols)
	styleRow(f, sheet, headerRowUnitPrice, len(unitPriceCols), bold)
	setCell(f, sheet, 1, 1, "综合单价（只含人材机，不含管理费/利润/税金）。三级行组：清单汇总行 → 定额行 → 工料机明细行；定额单价=Σ明细合价，清单综合单价=Σ(定额单价×系数)。")
	setCell(f, sheet, 1, 2, "每组 = 1 汇总行 + 若干定额块（块 = 1 定额行 + 明细行）。外委独立行类、三费归集并入材料桶。")

	for k, il := range layout.Items {
		sr := il.SummaryRow
		setCell(f, sheet, tupColItemCode, sr, il.Item.Code)
		setCell(f, sheet, 3, sr, il.Item.Title)
		setCell(f, sheet, 4, sr, il.Item.Unit)
		setCell(f, sheet, tupColNote, sr, "人材机小计；综合单价见S列（不含管利税）")

		// 行位置与输入按序对位：前 len(m.Items) 项是清单项，其后是空白组（没有输入块）。
		var inputs []workbookBlock
		if k < len(m.Items) {
			inputs = m.Items[k].Blocks
		}
		spans := make([]blockSpan, 0, len(il.Blocks))
		for i, bl := range il.Blocks {
			var wb workbookBlock
			if i < len(inputs) {
				wb = inputs[i]
			}
			qr := bl.QuotaRow
			setCell(f, sheet, tupColItemCode, qr, il.Item.Code)
			setCell(f, sheet, tupColQuotaCode, qr, wb.Quota.Code)
			setCell(f, sheet, tupColQuotaTitle, qr, wb.Quota.Title)
			setCell(f, sheet, 4, qr, wb.Quota.Unit) // 定额计量单位（可与清单单位不同）
			if wb.Quota.Factor > 0 {
				setCellNumber(f, sheet, tupColFactor, qr, wb.Quota.Factor)
			}
			var detailRows []int
			if bl.hasDetail() {
				for r := bl.First; r <= bl.Last; r++ {
					detailRows = append(detailRows, r)
				}
			}
			for j, l := range wb.Lines {
				if j >= len(detailRows) {
					break
				}
				row := detailRows[j]
				setCell(f, sheet, 1, row, l.Seq)
				setCell(f, sheet, tupColItemCode, row, l.ItemCode)
				setCell(f, sheet, 3, row, l.ItemTitle)
				setCell(f, sheet, 4, row, l.ItemUnit)
				setCell(f, sheet, tupColKind, row, firstNonEmpty(l.Kind, KindMaterial))
				setCell(f, sheet, tupColResCode, row, l.ResourceCode)
				setCell(f, sheet, tupColResTitle, row, l.ResourceTitle)
				setCell(f, sheet, tupColResUnit, row, l.Unit)
				setCellNumber(f, sheet, tupColQty, row, l.Quantity)
			}
			spans = append(spans, writeBlockRows(f, sheet, qr, detailRows, priceRange, money, m.Blank))
		}
		writeGroupSummary(f, sheet, sr, spans, money)
	}
	if !m.Blank {
		return nil
	}
	// 行类列下拉：人工/材料/机械/外委从这里挑（汇总/定额已写死；别的写法导入端拒收）。
	dv := excelize.NewDataValidation(true)
	dv.SetSqref(fmt.Sprintf("E%d:E%d", headerRowUnitPrice+1, layout.TotalRows))
	if err := dv.SetDropList([]string{"汇总", "定额", "人工", "材料", "机械", "外委"}); err != nil {
		return err
	}
	return f.AddDataValidation(sheet, dv)
}

// writeBillSheet 清单表：每个清单项一行（空白组给空行），综合单价引用综合单价表
// 各组汇总行 S 列、合价=工程量×综合单价。返回各行号（费用汇总的直接费合计引用）。
func writeBillSheet(f *excelize.File, m workbookModel, layout unitPriceLayout, bold, money int) []int {
	sheet := SheetBillName
	setCell(f, sheet, 1, 1, "分部分项清单。综合单价引用「综合单价」表，合价=工程量×综合单价（公式，联动重算）。")
	writeHeader(f, sheet, headerRowSummary, billSheetCols)
	styleRow(f, sheet, headerRowSummary, len(billSheetCols), bold)

	rows := make([]int, 0, len(layout.Items))
	for i, il := range layout.Items {
		row := headerRowSummary + 1 + i
		setCell(f, sheet, 1, row, fmt.Sprintf("%d", i+1))
		setCell(f, sheet, 2, row, il.Item.Division)
		setCell(f, sheet, 3, row, il.Item.Code)
		setCell(f, sheet, 4, row, il.Item.Title)
		setCell(f, sheet, 5, row, il.Item.Feature)
		setCell(f, sheet, 6, row, il.Item.Unit)
		if il.Item.Quantity != 0 {
			setCellNumber(f, sheet, 7, row, il.Item.Quantity)
		}
		setFormula(f, sheet, 8, row, fmt.Sprintf("'%s'!S%d", SheetUnitPriceName, il.SummaryRow))
		setFormula(f, sheet, 9, row, fmt.Sprintf("G%d*H%d", row, row))
		applyStyle(f, sheet, 8, row, money)
		applyStyle(f, sheet, 9, row, money)
		rows = append(rows, row)
	}
	return rows
}

// writeFeeSheet 费用汇总表：清单累计后的工程费（=清单表合价之和）+ 取费链。
// 清单行在「清单」表——解析按表名分流，本表只有取费段。
func writeFeeSheet(f *excelize.File, fee RateSet, controlPrice float64, billRows []int, bold, money int) summaryLayout {
	sheet := SheetSummaryName
	setCell(f, sheet, 1, 1, "费用汇总：清单累计后的工程费 + 取费链。清单行在「清单」表；改费率只改费率列。")
	writeHeader(f, sheet, headerRowSummary, feeCols)
	styleRow(f, sheet, headerRowSummary, len(feeCols), bold)

	var sl summaryLayout
	row := headerRowSummary + 1

	// 一、人材机直接费合计 = Σ 清单表合价。
	sl.DirectFeeRow = row
	directRef := fmt.Sprintf("E%d", row)
	setCell(f, sheet, 1, row, "一")
	setCell(f, sheet, 2, row, "分部分项人材机直接费合计")
	setCell(f, sheet, 3, row, "「"+SheetBillName+"」表合价合计")
	setCell(f, sheet, 4, row, "—")
	if len(billRows) > 0 {
		setFormula(f, sheet, 5, row, fmt.Sprintf("SUM('%s'!I%d:I%d)", SheetBillName, billRows[0], billRows[len(billRows)-1]))
	} else {
		setCell(f, sheet, 5, row, "—")
	}
	setCell(f, sheet, 6, row, "综合单价不含管利税")
	styleRow(f, sheet, row, len(feeCols), bold)
	applyStyle(f, sheet, 5, row, money)
	row += 2

	writeHeader(f, sheet, row, feeCols)
	styleRow(f, sheet, row, len(feeCols), bold)
	row++

	// 二、企业管理费 = 直接费 × 管理费率。
	sl.MgmtRow = row
	mgmtRef := fmt.Sprintf("E%d", row)
	writeRateRow(f, sheet, row, "二", "企业管理费", "直接费 × 管理费率",
		fee.ManagementRate, fmt.Sprintf("%s*D%d", directRef, row), "不进综合单价")
	row++

	// 三、利润 =（直接费+管理费）× 利润率。
	sl.ProfitRow = row
	profitRef := fmt.Sprintf("E%d", row)
	writeRateRow(f, sheet, row, "三", "利润", "（直接费+管理费）× 利润率",
		fee.ProfitRate, fmt.Sprintf("(%s+%s)*D%d", directRef, mgmtRef, row), "不进综合单价")
	row++

	// 四、规费 = 直接费 × 规费率（各专业口径不同，可为 0）。
	sl.RegulatoryRow = row
	regRef := fmt.Sprintf("E%d", row)
	writeRateRow(f, sheet, row, "四", "规费", "直接费 × 规费率",
		fee.RegulatoryRate, fmt.Sprintf("%s*D%d", directRef, row), "本测算按项目口径计取")
	row++

	// 五、税前合计 = 直接费+管理费+利润+规费。
	//
	// ⚠️ 必须**显式逐项相加**，不能用 SUM(直接费行:规费行)——区间的起点就是
	// 直接费行，SUM 会把它连同后面各项一起再算一遍（实测踩到：税前合计恰好
	// 多出一个直接费 + 一个管理费）。
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
		fee.TaxRate, fmt.Sprintf("%s*D%d", preTaxRef, row), "建筑服务一般计税 9% / 简易计税 3%")
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
	if controlPrice > 0 {
		setCellNumber(f, sheet, 5, row, controlPrice)
	} else {
		setCell(f, sheet, 5, row, "—")
	}
	setCell(f, sheet, 6, row, "投标不得超过，否则废标")
	applyStyle(f, sheet, 5, sl.ControlRow, money)
	return sl
}

// writeRateRow 写一行取费行：序号/名称/基数说明/费率/金额公式/备注。
func writeRateRow(f *excelize.File, sheet string, row int, seq, name, base string, rate float64, amountFormula, note string) {
	setCell(f, sheet, 1, row, seq)
	setCell(f, sheet, 2, row, name)
	setCell(f, sheet, 3, row, base)
	setCellNumber(f, sheet, 4, row, rate)
	setFormula(f, sheet, 5, row, amountFormula)
	setCell(f, sheet, 6, row, note)
}

// writeCoverSheet 封面：标题、六表顺序、口径三件、说明行；导出再带「测算成果」
// 指标块（跨表引用费用汇总，保持联动）。
func writeCoverSheet(f *excelize.File, m workbookModel, sl summaryLayout, bold, money int) {
	sheet := SheetCoverName
	setCell(f, sheet, 1, headerRowCover, firstNonEmpty(m.Title, "项目"))
	styleRow(f, sheet, headerRowCover, 7, bold)
	setCell(f, sheet, 1, headerRowCover+1, "六表模版：封面 → 清单 → 费用汇总 → 综合单价（只含人材机）→ 工料机价格 → 工程量计算。")
	row := headerRowCover + 2
	for i, label := range []string{"专业", "地区", "计价期"} {
		if v := strings.TrimSpace(m.Scope[i]); v != "" {
			setCell(f, sheet, 1, row, label+"："+v)
			row++
		}
	}
	for _, line := range m.CoverLines {
		setCell(f, sheet, 1, row, line)
		row++
	}
	if !m.Indicators {
		return
	}
	row++
	setCell(f, sheet, 1, row, "测算成果")
	styleRow(f, sheet, row, 4, bold)
	row++
	setCell(f, sheet, 1, row, "指标")
	setCell(f, sheet, 2, row, "金额 / 指标")
	setCell(f, sheet, 3, row, "口径")
	styleRow(f, sheet, row, 3, bold)
	row++
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
		setCell(f, sheet, 1, row, cr.label)
		setFormula(f, sheet, 2, row, cr.ref)
		setCell(f, sheet, 3, row, cr.note)
		applyStyle(f, sheet, 2, row, money)
		row++
	}
	if m.ControlPrice > 0 {
		setCell(f, sheet, 1, row, "招标控制价（含税）")
		setFormula(f, sheet, 2, row, fmt.Sprintf("'%s'!E%d", SheetSummaryName, sl.ControlRow))
		setCell(f, sheet, 3, row, "投标不得超过")
		applyStyle(f, sheet, 2, row, money)
	}
}

// writeQuantitySheet 工程量计算表：表头 + 已有的计算条目（空白模版只有表头，
// 解析端按实际内容探测行数，无需预写空行）。
func writeQuantitySheet(f *excelize.File, m workbookModel, bold int) {
	sheet := SheetQuantityName
	setCell(f, sheet, 1, 1, "工程量计算（数值列为输入；清单表的工程量可直接填数或引用本表）。")
	writeHeader(f, sheet, headerRowQuantity, quantityCols)
	styleRow(f, sheet, headerRowQuantity, len(quantityCols), bold)
	for i, q := range m.Quantities {
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
	first, err := excelize.CoordinatesToCellName(1, row)
	if err != nil {
		return
	}
	end, err := excelize.CoordinatesToCellName(cols, row)
	if err != nil {
		return
	}
	_ = f.SetCellStyle(sheet, first, end, style)
}

func applyStyle(f *excelize.File, sheet string, col, row, style int) {
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return
	}
	_ = f.SetCellStyle(sheet, cell, cell, style)
}

// firstNonEmpty 第一个非空串（去首尾空白；全空返回空串）。
func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
