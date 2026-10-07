package workcost

// projectimport_sheets.go —— 五表各 sheet 的具体解析。
//
// 列定位一律走关键词包含匹配（colIndex），因为实测两份产物表头并不逐字一致：
// 旺平用「消耗量单位」、什邡用「消耗单位」；单价列旺平作「单价」、百锦路作
// 「不含税单价（元）」。精确相等会在一份能跑、另一份静默全空。

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// maxProbeCols 扫描表头/数据时的列上限（模版最宽 16 列，留余量）。
const maxProbeCols = 20

// ── 封面 ────────────────────────────────────────────────────────────

// parseCover 从封面提取项目名/地点/工期/组价口径自述。
//
// 两种实测形态都要认：
//   - 什邡：一整行「地块：金宏化工… · 地点：德阳市什邡市马祖镇 · 工期12个月」（冒号切分）；
//   - 旺平：标签行与值行**相邻单元格**（B列「项目」/C列值），无冒号。
func parseCover(f *excelize.File, sheet string, b *ProjectBundle) {
	maxRow := sheetMaxRow(f, sheet, 60)
	cells := make([][]string, 0, maxRow)
	for r := 1; r <= maxRow; r++ {
		cells = append(cells, rowValues(f, sheet, r, 8))
	}

	// ── 第一遍：先独立确定项目名 ──────────────────────────────────
	//
	// 必须与字段配对**分开**：什邡封面的「地块：金宏化工、东升实业」会被字段
	// 配对抢先写成项目名，标题行反而被 `b.Project == ""` 挡住（实测踩到）。
	// 标题判据 = 前 4 行里「只有一个非空格子（合并整段）+ 无冒号 + ≥4 字」，
	// 不用字符数阈值——早期 len>6 会漏掉 6 字标题（如「导出验证项目」）。
	for r, vals := range cells {
		if r+1 > 4 {
			break
		}
		if b.Project != "" {
			break
		}
		line := strings.TrimSpace(strings.Join(nonEmpty(vals), " "))
		if line == "" || strings.Contains(line, "：") || strings.Contains(line, ":") {
			continue
		}
		if len([]rune(line)) >= 4 {
			b.Project = compactSpaces(line)
		}
	}

	// ── 第二遍：字段配对与取值 ────────────────────────────────────
	for _, vals := range cells {
		line := strings.TrimSpace(strings.Join(nonEmpty(vals), " "))
		if line == "" {
			continue
		}
		merged := rowIsMergedUniform(vals)

		// ② 冒号形态（什邡）：按 · 切段后取键值。
		for _, seg := range strings.FieldsFunc(line, func(rr rune) bool { return rr == '·' || rr == '|' }) {
			k, v, ok := splitKV(seg)
			if !ok {
				continue
			}
			b.assignCoverField(k, v)
		}

		// ③ 相邻单元格形态（旺平）：本行某「短标签」格的右邻格是值。
		//    合并整段行（标题/说明）跳过——否则会从合并区里读出假配对。
		//    标签须短（≤6 字）且值更长，避免把「口径 | 只对照不参与组价」这类
		//    成果表行误当字段。
		if merged {
			continue
		}
		for c := 0; c < len(vals)-1; c++ {
			k := strings.TrimSpace(vals[c])
			v := strings.TrimSpace(vals[c+1])
			if k == "" || v == "" || strings.Contains(k, "：") {
				continue
			}
			if len([]rune(k)) > 6 || len([]rune(v)) <= len([]rune(k)) {
				continue
			}
			b.assignCoverField(k, v)
		}
	}
	// 组价口径兜底。两种实测写法：
	//   旺平：「综合单价=工序消耗量×基础资源价。管理费5%…」（含 =）
	//   什邡：「五表模版：封面 → 费用汇总（工程量×综合单价）→ …，综合单价不含
	//         管理费、利润、税金。不引用投资估算单价。」（以「五表模版：」起）
	if b.Pricing == "" {
		for _, vals := range cells {
			for _, v := range vals {
				t := compactSpaces(v)
				if t == "" {
					continue
				}
				if strings.HasPrefix(t, "五表模版") ||
					(strings.Contains(t, "综合单价") && strings.Contains(t, "=")) {
					b.Pricing = t
					break
				}
			}
			if b.Pricing != "" {
				break
			}
		}
	}
}

// assignCoverField 按标签把值写入对应字段（只填首次，避免多行覆盖）。
func (b *ProjectBundle) assignCoverField(k, v string) {
	v = compactSpaces(v)
	if v == "" {
		return
	}
	switch {
	case strings.Contains(k, "项目"), strings.Contains(k, "名称"):
		if b.Project == "" {
			b.Project = v
		}
	case strings.Contains(k, "地块"):
		// 「地块：金宏化工、东升实业」是标的物描述，项目名以标题行为准，
		// 仅在项目名为空时兜底。
		if b.Project == "" {
			b.Project = v
		}
	case strings.Contains(k, "地点"), strings.Contains(k, "位置"), strings.Contains(k, "地址"):
		if b.Location == "" {
			b.Location = v
		}
	case strings.Contains(k, "工期"), strings.Contains(k, "计划"):
		if b.Duration == "" {
			b.Duration = v
		}
	case strings.Contains(k, "组价"), strings.Contains(k, "计价结构"), strings.Contains(k, "口径"):
		if b.Pricing == "" {
			b.Pricing = v
		}
	}
}

// compactSpaces 把换行/连续空白折成单个空格（标题单元格常含硬换行）。
func compactSpaces(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\n", " ")), " ")
}

// splitKV 切「键：值」或「键:值」。
func splitKV(s string) (string, string, bool) {
	for _, sep := range []string{"：", ":"} {
		if i := strings.Index(s, sep); i > 0 {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+len(sep):]), true
		}
	}
	return "", "", false
}

// nonEmpty 去掉空串（保留原序）。
func nonEmpty(vals []string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

// ── 工料机价格 ──────────────────────────────────────────────────────

// parseResourceSheet 解析工料机价格表 → 资源候选。
func parseResourceSheet(f *excelize.File, sheet string, b *ProjectBundle) {
	hdr := findHeaderRow(f, sheet, maxProbeCols, 8, "编码", "单位", "单价")
	if hdr == 0 {
		b.Warnings = append(b.Warnings, "工料机价格表未识别到表头，资源未导入")
		return
	}
	header := rowValues(f, sheet, hdr, maxProbeCols)
	cCode := colIndex(header, false, "编码")
	cKind := colIndex(header, false, "类别", "分类")
	cTitle := colIndex(header, false, "名称")
	cSpec := colIndex(header, false, "规格")
	cUnit := colIndex(header, false, "单位")
	cPrice := colIndex(header, false, "不含税单价", "单价")
	cSource := colIndex(header, false, "来源")
	cNote := colIndex(header, false, "备注")

	maxRow := sheetMaxRow(f, sheet, 400)
	for r := hdr + 1; r <= maxRow; r++ {
		code := cellAt(f, sheet, r, cCode)
		title := cellAt(f, sheet, r, cTitle)
		if strings.TrimSpace(code) == "" && strings.TrimSpace(title) == "" {
			continue
		}
		if strings.TrimSpace(title) == "" {
			b.Skipped = append(b.Skipped, SkippedRow{Sheet: sheet, Row: r, Reason: "资源无名称"})
			continue
		}
		cell, _ := excelize.CoordinatesToCellName(cPrice, r)
		res := ProjectResource{
			Code:         strings.TrimSpace(code),
			Kind:         normalizeKind(cellAt(f, sheet, r, cKind)),
			Title:        strings.TrimSpace(title),
			Spec:         strings.TrimSpace(cellAt(f, sheet, r, cSpec)),
			Unit:         strings.TrimSpace(cellAt(f, sheet, r, cUnit)),
			Price:        cellValue(f, sheet, cell),
			PriceFormula: formulaOf(f, sheet, cell),
			Source:       strings.TrimSpace(cellAt(f, sheet, r, cSource)),
			Note:         strings.TrimSpace(cellAt(f, sheet, r, cNote)),
		}
		if res.Price <= 0 {
			// 公式无缓存值（未用 LibreOffice 重算）→ 显形，不静默当 0 用。
			if res.PriceFormula != "" {
				b.Warnings = append(b.Warnings,
					fmt.Sprintf("资源 %s（%s）单价为公式且无缓存值：%s", res.Code, res.Title, res.PriceFormula))
			}
		}
		b.Resources = append(b.Resources, res)
	}
}

// ── 综合单价 ────────────────────────────────────────────────────────

// parseUnitPriceSheet 解析综合单价表：明细行 → 消耗量（工序）；汇总行 → 清单项。
func parseUnitPriceSheet(f *excelize.File, sheet string, b *ProjectBundle) {
	hdr := findHeaderRow(f, sheet, maxProbeCols, 8, "清单编码", "行类")
	if hdr == 0 {
		// 兼容表头写作「项目编码」「资源编码」的变体。
		hdr = findHeaderRow(f, sheet, maxProbeCols, 8, "编码", "行类")
	}
	if hdr == 0 {
		b.Warnings = append(b.Warnings, "综合单价表未识别到表头，消耗量未导入")
		return
	}
	header := rowValues(f, sheet, hdr, maxProbeCols)
	cSeq := colIndex(header, false, "序号")
	cItemCode := colIndex(header, false, "清单编码", "项目编码")
	cItemTitle := colIndex(header, false, "项目名称")
	cItemUnit := colIndex(header, false, "计量单位", "单位")
	cKind := colIndex(header, false, "行类")
	cResCode := colIndex(header, false, "资源编码", "工料机编码")
	cResTitle := colIndex(header, false, "资源名称", "工料机名称")
	cUnitQ := colIndex(header, false, "消耗量单位", "消耗单位")
	cQty := colIndex(header, false, "消耗量")
	cPrice := colIndex(header, false, "单价")
	cAmount := colIndex(header, false, "合价", "人材机合价")

	if cKind == 0 {
		b.Warnings = append(b.Warnings, "综合单价表缺「行类」列，无法区分汇总/明细行")
		return
	}

	maxRow := sheetMaxRow(f, sheet, 600)
	seenItems := map[string]bool{}
	for r := hdr + 1; r <= maxRow; r++ {
		kind := strings.TrimSpace(cellAt(f, sheet, r, cKind))
		itemCode := strings.TrimSpace(cellAt(f, sheet, r, cItemCode))
		itemTitle := strings.TrimSpace(cellAt(f, sheet, r, cItemTitle))
		if kind == "" && itemCode == "" && itemTitle == "" {
			continue
		}
		if kind == "汇总" || kind == "" {
			// 汇总行 = 清单项（综合单价的载体）。工程量在费用汇总里，此处只收身份。
			if itemCode == "" {
				continue
			}
			if seenItems[itemCode] {
				continue
			}
			seenItems[itemCode] = true
			b.Items = append(b.Items, ProjectItem{
				Code:         itemCode,
				Title:        itemTitle,
				Unit:         strings.TrimSpace(cellAt(f, sheet, r, cItemUnit)),
				CategoryPath: "综合单价/" + itemTitle,
			})
			continue
		}
		// 明细行 = 工序消耗。
		resCode := strings.TrimSpace(cellAt(f, sheet, r, cResCode))
		if resCode == "" {
			b.Skipped = append(b.Skipped, SkippedRow{Sheet: sheet, Row: r,
				Reason: fmt.Sprintf("明细行缺资源编码（%s %s）", itemCode, cellAt(f, sheet, r, cResTitle))})
			continue
		}
		cellQty, _ := excelize.CoordinatesToCellName(cQty, r)
		cellPrice, _ := excelize.CoordinatesToCellName(cPrice, r)
		cellAmount, _ := excelize.CoordinatesToCellName(cAmount, r)
		line := ProjectConsumption{
			Seq:           strings.TrimSpace(cellAt(f, sheet, r, cSeq)),
			ItemCode:      itemCode,
			ItemTitle:     itemTitle,
			ItemUnit:      strings.TrimSpace(cellAt(f, sheet, r, cItemUnit)),
			Kind:          normalizeKind(kind),
			ResourceCode:  resCode,
			ResourceTitle: strings.TrimSpace(cellAt(f, sheet, r, cResTitle)),
			Unit:          strings.TrimSpace(cellAt(f, sheet, r, cUnitQ)),
			Quantity:      cellValue(f, sheet, cellQty),
			Price:         cellValue(f, sheet, cellPrice),
			PriceFormula:  formulaOf(f, sheet, cellPrice),
			Amount:        cellValue(f, sheet, cellAmount),
		}
		b.Lines = append(b.Lines, line)
	}
}

// ── 费用汇总 ────────────────────────────────────────────────────────

// parseSummarySheet 解析费用汇总：上半清单（工程量/分部）与下半取费参数。
func parseSummarySheet(f *excelize.File, sheet string, b *ProjectBundle) {
	hdr := findHeaderRow(f, sheet, maxProbeCols, 8, "项目名称", "工程量")
	if hdr != 0 {
		header := rowValues(f, sheet, hdr, maxProbeCols)
		cDiv := colIndex(header, false, "分部")
		cCode := colIndex(header, false, "编码")
		cTitle := colIndex(header, false, "项目名称")
		cFeature := colIndex(header, false, "工序特征", "项目特征")
		cUnit := colIndex(header, false, "单位")
		cQty := colIndex(header, false, "工程量")

		byCode := map[string]*ProjectItem{}
		for i := range b.Items {
			byCode[b.Items[i].Code] = &b.Items[i]
		}
		maxRow := sheetMaxRow(f, sheet, 400)
		for r := hdr + 1; r <= maxRow; r++ {
			title := strings.TrimSpace(cellAt(f, sheet, r, cTitle))
			code := strings.TrimSpace(cellAt(f, sheet, r, cCode))
			if code == "" && title == "" {
				continue
			}
			// 下半段取费区以「费用名称」类文本起始，跳出清单区。
			if strings.Contains(title, "直接费合计") || strings.Contains(title, "费用名称") {
				break
			}
			div := strings.TrimSpace(cellAt(f, sheet, r, cDiv))
			cellQty, _ := excelize.CoordinatesToCellName(cQty, r)
			qtyExpr := formulaOf(f, sheet, cellQty)
			qty := cellValue(f, sheet, cellQty)
			if it, ok := byCode[code]; ok && code != "" {
				it.Division = div
				it.Feature = strings.TrimSpace(cellAt(f, sheet, r, cFeature))
				if it.Unit == "" {
					it.Unit = strings.TrimSpace(cellAt(f, sheet, r, cUnit))
				}
				it.Quantity = qty
				it.QuantityExpr = qtyExpr
				it.CategoryPath = "综合单价/" + firstNonEmpty(div, it.Title)
				continue
			}
			// 综合单价表没列到的清单项，按费用汇总补齐（但标记为待补消耗量）。
			if code != "" {
				b.Items = append(b.Items, ProjectItem{
					Code: code, Title: title, Unit: strings.TrimSpace(cellAt(f, sheet, r, cUnit)),
					Division: div, Feature: strings.TrimSpace(cellAt(f, sheet, r, cFeature)),
					Quantity: qty, QuantityExpr: qtyExpr,
					CategoryPath: "综合单价/" + firstNonEmpty(div, title),
				})
				b.Warnings = append(b.Warnings, fmt.Sprintf(
					"清单项 %s（%s）在综合单价表无汇总行——消耗量待补", code, title))
			}
		}
	}
	parseFeeSection(f, sheet, b)
}

// parseFeeSection 解析费用汇总下半段的取费参数（管理费/利润/规费/税/控制价）。
//
// 行形态（实测，列位不固定——序号列在最左）：
//
//	二 | 企业管理费 | 直接费 × 管理费率 | 0.05 | =E54*D55 | 不进综合单价
//	三 | 利润      | （直接费+管理费）× 利润率 | 0 | ...
//	九 | 增值税    | 税前合计 × 增值税率 | 0.03 | ...
//	十一 | 招标控制价（含税） | 招标文件10.1 | — | =工程量计算!C20
//
// ⚠️ 早期实现取「行内第一个非空单元格」当费用名，实测拿到的是序号「一/二/三」，
// 于是费率为 0 却零告警——本函数改为**全行扫描关键词**，并且费率优先取
// 「费率」列表头所在列的值（表头由 findHeaderRow 定位）。
func parseFeeSection(f *excelize.File, sheet string, b *ProjectBundle) {
	maxRow := sheetMaxRow(f, sheet, 400)
	// 定位取费区表头（含「费用名称」），拿到「费率」列号。
	feeHdr := 0
	feeRateCol := 0
	for r := 1; r <= maxRow; r++ {
		vals := rowValues(f, sheet, r, 11)
		joined := strings.Join(vals, "|")
		if strings.Contains(joined, "费用名称") || strings.Contains(joined, "费用类别") {
			feeHdr = r
			feeRateCol = colIndex(vals, false, "费率")
			break
		}
	}

	found := false
	for r := 1; r <= maxRow; r++ {
		vals := rowValues(f, sheet, r, 11)
		if len(nonEmpty(vals)) == 0 {
			continue
		}
		name := feeNameInRow(vals)
		if name == "" {
			continue
		}
		// 优先取表头定位到的费率列；退一步全行找 0<v<1 的纯小数。
		rate := func() (float64, bool) {
			if feeHdr > 0 && r > feeHdr && feeRateCol > 0 && feeRateCol <= len(vals) {
				if v, ok := parseNum(vals[feeRateCol-1]); ok && v >= 0 && v < 1 {
					return v, true
				}
			}
			return rateInRow(vals)
		}

		switch {
		case strings.Contains(name, "企业管理费") || strings.Contains(name, "管理费"):
			if v, ok := rate(); ok {
				b.Fee.ManagementRate = v
				b.Fee.ManagementFormula = strings.Join(nonEmpty(vals), " | ")
				found = true
			}
		case strings.Contains(name, "利润"):
			if v, ok := rate(); ok {
				b.Fee.ProfitRate = v
				found = true
			}
		case strings.Contains(name, "规费"):
			if v, ok := rate(); ok {
				b.Fee.RegulatoryRate = v
				found = true
			}
		case strings.Contains(name, "增值税") || strings.Contains(name, "税金"):
			if v, ok := rate(); ok {
				b.Fee.TaxRate = v
				b.Fee.TaxNote = strings.Join(nonEmpty(vals), " | ")
				found = true
			}
		case strings.Contains(name, "控制价"):
			for c := 1; c <= len(vals); c++ {
				cell, _ := excelize.CoordinatesToCellName(c, r)
				if v := cellValue(f, sheet, cell); v > 0 {
					b.Fee.ControlPrice = v
					break
				}
			}
		}
	}
	// 封面自述兜底：实测封面会写「管理费5%、利润0%、增值税3%在费用汇总」。
	if !found {
		if ok := parseRatesFromText(b.Pricing, b); ok {
			b.Warnings = append(b.Warnings, "取费费率取自封面自述（费用汇总费率列未解析到），建议人工复核")
		}
	}
	if b.Fee.ManagementRate == 0 && b.Fee.ProfitRate == 0 && b.Fee.TaxRate == 0 {
		b.Warnings = append(b.Warnings, "未解析到取费费率（管理费/利润/税），取费参数待人工确认")
	}
}

// feeNameInRow 在整行里找费用名（跳过序号「一/二/三」与纯数字/公式格）。
func feeNameInRow(vals []string) string {
	names := []string{"企业管理费", "管理费", "利润", "规费", "增值税", "税金", "控制价"}
	for _, v := range vals {
		t := strings.TrimSpace(v)
		if t == "" || strings.HasPrefix(t, "=") {
			continue
		}
		if _, isNum := parseNum(t); isNum {
			continue
		}
		for _, n := range names {
			if strings.Contains(t, n) {
				return t
			}
		}
	}
	return ""
}

// parseRatesFromText 从自述文本解析费率，支持「管理费5%」「利润0%」「增值税3%」
// 与「管理费0.05」两种写法。
func parseRatesFromText(text string, b *ProjectBundle) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	got := false
	pairs := []struct {
		keys []string
		dst  *float64
	}{
		{[]string{"管理费", "企业管理费"}, &b.Fee.ManagementRate},
		{[]string{"利润"}, &b.Fee.ProfitRate},
		{[]string{"规费"}, &b.Fee.RegulatoryRate},
		{[]string{"增值税", "税金", "税率"}, &b.Fee.TaxRate},
	}
	for _, p := range pairs {
		for _, k := range p.keys {
			idx := strings.Index(text, k)
			if idx < 0 {
				continue
			}
			if v, ok := firstRateAfter(text[idx+len(k):]); ok {
				*p.dst = v
				got = true
				break
			}
		}
	}
	return got
}

// firstRateAfter 在片段开头若干字符内找百分比或小数费率。
func firstRateAfter(s string) (float64, bool) {
	// 跳过「按」「为」「：」等连接字符，最多看 12 个字符。
	r := []rune(strings.TrimSpace(s))
	if len(r) > 14 {
		r = r[:14]
	}
	seg := string(r)
	if i := strings.Index(seg, "%"); i > 0 {
		// 向前取数字。
		j := i - 1
		for j >= 0 && (seg[j] >= '0' && seg[j] <= '9' || seg[j] == '.') {
			j--
		}
		if j < i-1 {
			if n, ok := parseNum(seg[j+1 : i]); ok {
				return n / 100, true
			}
		}
	}
	for _, piece := range strings.FieldsFunc(seg, func(rr rune) bool {
		return !(rr >= '0' && rr <= '9' || rr == '.')
	}) {
		if n, ok := parseNum(piece); ok && n >= 0 && n < 1 {
			return n, true
		}
	}
	return 0, false
}

// rateInRow 在一行里找费率：优先「费率」列（值形如 0.05 或 5%），
// 退一步从含「%」或百分数描述的单元格里取。
func rateInRow(vals []string) (float64, bool) {
	// 先找显式百分比。
	for _, v := range vals {
		t := strings.TrimSpace(v)
		if strings.HasSuffix(t, "%") {
			if n, ok := parseNum(t); ok {
				return n, true
			}
		}
	}
	// 再找形如「费率」列标题右邻的纯小数（0<v<1）。
	for i, v := range vals {
		if strings.Contains(v, "费率") {
			if i+1 < len(vals) {
				if n, ok := parseNum(vals[i+1]); ok && n > 0 && n < 1 {
					return n, true
				}
			}
		}
		_ = i
	}
	// 最后：行内任何 0<v<1 的纯小数（费率列通常紧跟说明列）。
	for _, v := range vals {
		t := strings.TrimSpace(v)
		if t == "" || strings.HasPrefix(t, "=") {
			continue
		}
		if n, ok := parseNum(t); ok && n > 0 && n < 1 {
			return n, true
		}
	}
	return 0, false
}

// ── 工程量计算 ──────────────────────────────────────────────────────

// parseQuantitySheet 解析工程量计算表（供清单项工程量取值与追溯）。
func parseQuantitySheet(f *excelize.File, sheet string, b *ProjectBundle) {
	hdr := findHeaderRow(f, sheet, maxProbeCols, 8, "编码", "名称", "单位")
	if hdr == 0 {
		return
	}
	header := rowValues(f, sheet, hdr, maxProbeCols)
	cCode := colIndex(header, false, "编码")
	cTitle := colIndex(header, false, "名称")
	cVal := colIndex(header, false, "数值")
	cUnit := colIndex(header, false, "单位")
	cNote := colIndex(header, false, "备注")

	maxRow := sheetMaxRow(f, sheet, 300)
	for r := hdr + 1; r <= maxRow; r++ {
		code := strings.TrimSpace(cellAt(f, sheet, r, cCode))
		title := strings.TrimSpace(cellAt(f, sheet, r, cTitle))
		if code == "" && title == "" {
			continue
		}
		cell, _ := excelize.CoordinatesToCellName(cVal, r)
		b.Quantities = append(b.Quantities, ProjectQuantity{
			Code:  code,
			Title: title,
			Value: cellValue(f, sheet, cell),
			Unit:  strings.TrimSpace(cellAt(f, sheet, r, cUnit)),
			Note:  strings.TrimSpace(cellAt(f, sheet, r, cNote)),
		})
	}
}

// ── 对账 ────────────────────────────────────────────────────────────

// crossCheck 解析结果自洽校验：消耗量合计应与清单项一一对应，资源引用必须
// 落在工料机表内。返回问题清单（不阻断导入，但必须显形）。
func crossCheck(b *ProjectBundle) []string {
	var warns []string
	// ① 资源引用完整性：每条消耗行的资源编码必须在资源表里。
	codes := map[string]bool{}
	for _, r := range b.Resources {
		if r.Code != "" {
			codes[r.Code] = true
		}
	}
	missing := map[string]int{}
	for _, l := range b.Lines {
		if l.ResourceCode != "" && !codes[l.ResourceCode] {
			missing[l.ResourceCode]++
		}
	}
	for code, n := range missing {
		warns = append(warns, fmt.Sprintf("消耗量引用了工料机表中不存在的资源编码 %s（%d 处）", code, n))
	}
	// ② 清单项覆盖：每个清单项都应有消耗行（否则综合单价为空）。
	withLines := map[string]bool{}
	for _, l := range b.Lines {
		withLines[l.ItemCode] = true
	}
	empty := 0
	for _, it := range b.Items {
		if !withLines[it.Code] {
			empty++
		}
	}
	if empty > 0 {
		warns = append(warns, fmt.Sprintf("%d 个清单项没有任何工料机消耗行（综合单价为空）", empty))
	}
	// ③ 汇总一致性：Σ(消耗量金额) 应与清单项综合单价小计一致——
	// 此处只报总数，供调用方与费用汇总对账。
	if len(b.Lines) == 0 {
		warns = append(warns, "未解析到任何工料机消耗行")
	}
	return warns
}

// ── 小工具 ──────────────────────────────────────────────────────────

// cellAt 取第 row 行第 col 列的文本（col=0 返回空）。
func cellAt(f *excelize.File, sheet string, row, col int) string {
	if col <= 0 {
		return ""
	}
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return ""
	}
	return cellText(f, sheet, cell)
}

// formulaOf 取单元格公式原文（无公式返回空）。
func formulaOf(f *excelize.File, sheet, cell string) string {
	fo, err := f.GetCellFormula(sheet, cell)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(fo)
}

// sheetMaxRow 取 sheet 实际最大行（带上限保护与最小兜底）。
func sheetMaxRow(f *excelize.File, sheet string, limit int) int {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return 0
	}
	n := len(rows)
	if n > limit {
		return limit
	}
	return n
}

// firstNonEmpty 返回首个非空值。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
