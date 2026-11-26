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

// maxProbeCols 扫描表头/数据时的列上限（模版最宽 19 列，留余量）。
const maxProbeCols = 20

// maxSheetRows 各表扫描的行数上限。空白模版默认 30 组 × 11 行，groups 可调；
// 上限按「几百个清单项的测算」留足，真实项目离它还远。
const maxSheetRows = 5000

// ── 封面 ────────────────────────────────────────────────────────────

// parseCover 从封面提取项目名/地点/工期/组价口径自述。
//
// 两种实测形态都要认：
//   - 什邡：一整行「地块：金宏化工… · 地点：德阳市什邡市马祖镇 · 工期12个月」（冒号切分）；
//   - 旺平：标签行与值行**相邻单元格**（B列「项目」/C列值），无冒号；
//   - 三级树表（现行样表）：标签格与值格之间隔着细分隔列（标签 B:C、值 D:J），
//     标签「项目名称」的值要往右找下一个非空格；顶部是书名带。
//
// 项目名（2026-11-27）：封面明示的「项目名称」标签值 与 顶部标题带最左一段，两处
// 取舍见 coverProjectName。此前把标题带整行非空格拼起来——v5.5 把书名放回 G2
// 后，项目名被读成「XX项目 施 工 成 本 测 算 书」；标签与值之间隔一列时地点 /
// 工期也配不上对。
func parseCover(f *excelize.File, sheet string, b *ProjectBundle) {
	maxRow := sheetMaxRow(f, sheet, 60)
	cells := make([][]string, 0, maxRow)
	for r := 1; r <= maxRow; r++ {
		cells = append(cells, rowValues(f, sheet, r, 8))
	}

	// ── 第〇遍：封面明示的项目名（「项目名称」标签格右边的值）──────────
	declared := ""
	for _, vals := range cells {
		if rowIsMergedUniform(vals) {
			continue
		}
		for c, v := range vals {
			if k := strings.TrimSuffix(strings.TrimSpace(v), "："); k != "项目名称" && k != "工程名称" {
				continue
			}
			if nv := nextNonEmpty(vals, c); nv != "" && !isCoverPlaceholder(nv) {
				declared = compactSpaces(nv)
			}
		}
	}

	// ── 第一遍：顶部标题带（最左一段）──────────────────────────────
	//
	// 必须与字段配对**分开**：什邡封面的「地块：金宏化工、东升实业」会被字段
	// 配对抢先写成项目名，标题行反而被 `b.Project == ""` 挡住（实测踩到）。
	// 标题判据 = 前 4 行里「无冒号 + ≥4 字」，只取该行**最左**一段非空格——
	// 右侧是书名等装饰（v5.5 的 G2「施 工 成 本 测 算 书」），拼进来项目名就成了
	// 「XX项目 施 工 成 本 测 算 书」。不用字符数阈值——早期 len>6 会漏掉 6 字
	// 标题（如「导出验证项目」）。
	band := ""
	for r, vals := range cells {
		if r+1 > 4 {
			break
		}
		ne := nonEmpty(vals)
		if len(ne) == 0 {
			continue
		}
		line := strings.TrimSpace(ne[0])
		if strings.Contains(line, "：") || strings.Contains(line, ":") {
			continue
		}
		if len([]rune(line)) >= 4 {
			band = compactSpaces(line)
			break
		}
	}
	b.Project, b.Warnings = coverProjectName(band, declared, b.Warnings)

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
			// 值 = 右边下一个非空格：三级树表封面的标签（B:C）与值（D:J）之间
			// 隔着一列细分隔，只看紧邻格配不上对（地点 / 工期读成空）。
			v := nextNonEmpty(vals, c)
			if k == "" || v == "" || strings.Contains(k, "：") || isCoverPlaceholder(v) {
				continue
			}
			if len([]rune(k)) > 6 || len([]rune(v)) <= len([]rune(k)) {
				continue
			}
			b.assignCoverField(k, v)
		}
	}
	// ── 口径自述（六表模版封面写明「专业 / 地区 / 计价期」）────────────
	//
	// 只记「文件里写了什么」，不当规范值用：导入时调用方没给口径才拿它兜底，
	// 且仍要过字典校验（app 侧 importProject）。两种写法都认——冒号整格
	// 「专业：土壤修复」；标签格 + 右邻值格。标签须**恰为**这几个词：上面的
	// 短标签配对按长度判（值须比标签长），「地区 | 铜陵」这类两字值走不进去。
	for _, vals := range cells {
		merged := rowIsMergedUniform(vals)
		for c, v := range vals {
			t := strings.TrimSpace(v)
			if t == "" {
				continue
			}
			if k, val, ok := splitKV(t); ok {
				b.assignCoverScope(k, val)
				continue
			}
			if merged {
				continue
			}
			k := strings.TrimSuffix(t, "：")
			nv := nextNonEmpty(vals, c)
			if nv == "" || isCoverPlaceholder(nv) {
				continue
			}
			switch {
			case k == "专业" || k == "地区" || k == "计价期" || k == "期数":
				b.assignCoverScope(k, nv)
			case strings.Contains(k, "专业") && strings.Contains(k, "计价期"):
				// 三级树表封面合写「专业 / 计价期」：值按 / 拆两段。
				left, right, _ := strings.Cut(nv, "/")
				b.assignCoverScope("专业", strings.TrimSpace(left))
				b.assignCoverScope("计价期", strings.TrimSpace(right))
			}
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
				if strings.HasPrefix(t, "五表模版") || strings.HasPrefix(t, "六表模版") ||
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

// assignCoverScope 把封面自述的口径记到 Cover* 字段（只填首次）。标签取整个
// 冒号前缀，所以「项目专业」「定额专业」这类不会被误认成口径自述。
func (b *ProjectBundle) assignCoverScope(k, v string) {
	v = compactSpaces(v)
	if v == "" {
		return
	}
	switch strings.TrimSpace(k) {
	case "专业":
		if b.CoverSpecialty == "" {
			b.CoverSpecialty = v
		}
	case "地区":
		if b.CoverRegion == "" {
			b.CoverRegion = v
		}
	case "计价期", "期数", "价格期":
		if b.CoverPeriod == "" {
			b.CoverPeriod = v
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
	case strings.Contains(k, "组价"), strings.Contains(k, "计价结构"), strings.Contains(k, "口径"), strings.Contains(k, "计价方式"):
		if b.Pricing == "" {
			b.Pricing = v
		}
	}
}

// compactSpaces 把换行/连续空白折成单个空格（标题单元格常含硬换行）。
func compactSpaces(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\n", " ")), " ")
}

// coverProjectName 定封面项目名：顶部标题带最左一段（band）与「项目名称」标签值
// （declared）二选一。
//
//   - 只有一处：用它；
//   - 两处一致（一方包含另一方也算）：标题带优先——v5.3 起项目名就落在 B2，
//     这是样表写明的契约位置；
//   - 标题带是书名（含「测算书」）：它是版式装饰，项目名以标签值为准（v5.1 封面
//     的 B2 是「XX工程 · 施工成本测算书」，真项目名在标签右边）；
//   - 其余不一致：仍按标题带导，但点名两处写法——多半是拿上一个项目的簿改的，
//     改了一处漏了一处；导错项目名会把两个项目的数据并到一起。
func coverProjectName(band, declared string, warns []string) (string, []string) {
	band, declared = strings.TrimSpace(band), strings.TrimSpace(declared)
	switch {
	case declared == "":
		return band, warns
	case band == "" || band == declared || strings.Contains(band, declared) || strings.Contains(declared, band):
		return firstNonEmpty(band, declared), warns
	case strings.Contains(band, "测算书"):
		return declared, warns
	}
	warns = append(warns, fmt.Sprintf(
		"封面项目名两处不一致：标题「%s」≠「项目名称」标签值「%s」——按标题「%s」导入（样表两处应同步填写；若应以标签值为准，改标题后重导）",
		band, declared, band))
	return band, warns
}

// isCoverPlaceholder 封面标签值的占位写法：整格包在全角括号里（「（与 B2 同步
// 填写）」「（省 / 市 / 区）」），或写着「填写 / 同步」。
func isCoverPlaceholder(v string) bool {
	v = strings.TrimSpace(v)
	return strings.HasPrefix(v, "（") && strings.HasSuffix(v, "）") ||
		strings.Contains(v, "填写") || strings.Contains(v, "同步")
}

// nextNonEmpty vals 里 c 右边第一个非空格（合并整段行去重后，标签与值之间会留空格）。
func nextNonEmpty(vals []string, c int) string {
	for nc := c + 1; nc < len(vals); nc++ {
		if nv := strings.TrimSpace(vals[nc]); nv != "" {
			return nv
		}
	}
	return ""
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
	cGCode := colIndex(header, false, "自有码")
	cKind := colIndex(header, false, "类别", "分类")
	cTitle := colIndex(header, false, "名称")
	cSpec := colIndex(header, false, "规格")
	cUnit := colIndex(header, false, "单位")
	cPrice := colIndex(header, false, "不含税单价", "单价")
	cSource := colIndex(header, false, "来源")
	cNote := colIndex(header, false, "备注")

	// 行数上限按空白模版的规模留足（默认 30 组 × 11 行；groups 可调到数百组）。
	maxRow := sheetMaxRow(f, sheet, maxSheetRows)
	for r := hdr + 1; r <= maxRow; r++ {
		code := cellAt(f, sheet, r, cCode)
		title := cellAt(f, sheet, r, cTitle)
		if strings.TrimSpace(code) == "" && strings.TrimSpace(title) == "" {
			continue
		}
		if strings.TrimSpace(title) == "" {
			b.skip(f, sheet, r, "资源无名称")
			continue
		}
		cell, _ := excelize.CoordinatesToCellName(cPrice, r)
		res := ProjectResource{
			Code:         strings.TrimSpace(code),
			GCode:        strings.TrimSpace(cellAt(f, sheet, r, cGCode)),
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

// treeLevelItem / treeLevelQuota 综合单价「层级」列的两个结构行取值；其余已知
// 类别（人工 / 材料 / 机械 / 外委）是人材机明细行。
const (
	treeLevelItem  = "清单"
	treeLevelQuota = "定额"
)

// parseUnitPriceSheet 解析综合单价表——三级树表（清单 → 定额 → 人材机）。返回
// error 表示这张表读不了、整份工作簿不该导。
//
// 行型只看「层级」列：清单 / 定额 / 人工·材料·机械·外委。三种行共用 编码 /
// 名称 / 单位 / 工程量 / 单价 / 金额 一组列，含义随行型变：
//
//	清单行  编码=清单编码    名称=项目名称            工程量=1          单价=综合单价
//	定额行  编码=定额编码    名称=定额名称            工程量=换算系数（空=1）
//	人材机  编码=工料机编码  名称 / 单位 / 单价=取价公式  工程量=含量（每定额单位）
//
// 不按固定 11 行网格推行号：样表是 1 清单 + 2 块 ×（1 定额 + 4 明细），实填表是
// 变长组；A 列多级序号（1 / 1.1 / 1.1.1）只是版式，不作判据。
//
// 旧「行类」版式不认（2026-11-27 用户拍板退役，以后不再上传）：表头不是三级
// 树表就拒收——放着继续导，定额与消耗行整批丢失，清单行全部「无消耗」入库。
func parseUnitPriceSheet(f *excelize.File, sheet string, b *ProjectBundle) error {
	hdr := findHeaderRow(f, sheet, maxProbeCols, 8, "层级", "编码", "名称")
	if hdr == 0 {
		return fmt.Errorf("「%s」不是三级树表（表头要有 层级 / 编码 / 名称 三列，行型看层级列：清单 / 定额 / 人工·材料·机械·外委），读不了——旧版「行类」版式已停用，请用最新样表（cost_library action=template）重做后再导", sheet)
	}
	header := rowValues(f, sheet, hdr, maxProbeCols)
	cSeq := colIndex(header, false, "序号")
	cLevel := colIndex(header, false, "层级")
	cCode := colIndex(header, false, "编码")
	cTitle := colIndex(header, false, "名称")
	cUnit := colIndex(header, false, "单位")
	cQty := colIndex(header, false, "工程量")
	cPrice := colIndex(header, false, "单价")
	cAmount := colIndex(header, false, "金额")
	// 清单工程量列（清单行引清单表工程量的公式，重算后有缓存值）：清单表改了版式
	// 或没有这一行时，工程量退回这一列。
	cBillQty := colIndex(header, false, "清单工程量")
	if cLevel == 0 || cCode == 0 || cQty == 0 {
		return fmt.Errorf("「%s」表头缺 层级 / 编码 / 工程量 列，读不了三级树表", sheet)
	}

	// 明细行的名称 / 单位是取价公式（VLOOKUP 工料机价格）：取缓存值，没缓存时
	// 回退工料机价格表里的同一编码。
	resByCode := make(map[string]ProjectResource, len(b.Resources))
	for _, r := range b.Resources {
		if r.Code != "" {
			resByCode[r.Code] = r
		}
	}
	num := func(col, r int) float64 {
		if col <= 0 {
			return 0
		}
		cell, _ := excelize.CoordinatesToCellName(col, r)
		return cellValue(f, sheet, cell)
	}
	expr := func(col, r int) string {
		if col <= 0 {
			return ""
		}
		cell, _ := excelize.CoordinatesToCellName(col, r)
		return formulaOf(f, sheet, cell)
	}
	maxRow := sheetMaxRow(f, sheet, maxSheetRows)
	seenItems := map[string]bool{}
	// 当前清单组 / 定额块：空 = 上方的结构行不可用（缺编码、被跳过），其后的行
	// 不得错挂到上一组 / 上一块。
	itemCode, itemTitle, itemUnit := "", "", ""
	quotaCode := ""
	for r := hdr + 1; r <= maxRow; r++ {
		level := strings.TrimSpace(cellAt(f, sheet, r, cLevel))
		code := strings.TrimSpace(cellAt(f, sheet, r, cCode))
		title := strings.TrimSpace(cellAt(f, sheet, r, cTitle))
		switch {
		case level == treeLevelItem:
			quotaCode, itemCode = "", ""
			if code == "" {
				if title != "" {
					b.skip(f, sheet, r, fmt.Sprintf("清单行「%s」缺清单编码，整组跳过", title))
				}
				continue // 样表里没用到的预置清单行（编码、名称都没填）
			}
			if seenItems[code] {
				b.skip(f, sheet, r, fmt.Sprintf("清单编码 %s 重复出现，只取第一组", code))
				continue
			}
			seenItems[code] = true
			itemCode, itemTitle, itemUnit = code, title, strings.TrimSpace(cellAt(f, sheet, r, cUnit))
			it := ProjectItem{Code: code, Title: title, Unit: itemUnit, CategoryPath: "综合单价/" + firstNonEmpty(title, code)}
			it.ListPrice, it.ListPriceExpr = num(cPrice, r), expr(cPrice, r)
			it.Quantity, it.QuantityExpr = num(cBillQty, r), expr(cBillQty, r)
			// 公式无缓存值（没用 xlsx_recalc 重算）→ 显形，不静默当 0。
			if it.ListPrice <= 0 && it.ListPriceExpr != "" {
				b.Warnings = append(b.Warnings,
					fmt.Sprintf("清单项 %s（%s）综合单价为公式且无缓存值：%s——先 xlsx_recalc 再导", code, title, it.ListPriceExpr))
			}
			b.Items = append(b.Items, it)
		case level == treeLevelQuota:
			quotaCode = ""
			if code == "" {
				if title != "" {
					b.skip(f, sheet, r, fmt.Sprintf("定额行「%s」缺定额编码（清单 %s）", title, itemCode))
				}
				continue // 没用到的预置定额块
			}
			if itemCode == "" {
				b.skip(f, sheet, r, fmt.Sprintf("定额 %s（%s）上方没有可用的清单行——清单行缺编码或被跳过，整块跳过", code, title))
				continue
			}
			factor := num(cQty, r)
			if factor < 0 {
				b.Warnings = append(b.Warnings,
					fmt.Sprintf("定额 %s（清单 %s）的换算系数为负（%g），已按 1 计", code, itemCode, factor))
				factor = 0
			}
			quotaCode = code
			b.QuotaBlocks = append(b.QuotaBlocks, ProjectQuotaBlock{
				ItemCode: itemCode, Code: code, Title: title,
				Unit: strings.TrimSpace(cellAt(f, sheet, r, cUnit)), Factor: factor,
			})
		case level != "" && isKnownKind(normalizeKind(level)):
			if code == "" && num(cQty, r) == 0 {
				continue // 没用到的明细槽（取价公式是样表预置的，不算「行上有东西」）
			}
			if quotaCode == "" {
				b.skip(f, sheet, r, fmt.Sprintf("明细行「%s」上方没有可用的定额行（定额行缺编码或被跳过），整行跳过", code))
				continue
			}
			if code == "" {
				b.skip(f, sheet, r, fmt.Sprintf("明细行有含量 %g 却没有工料机编码（清单 %s / 定额 %s）", num(cQty, r), itemCode, quotaCode))
				continue
			}
			line := ProjectConsumption{
				Seq: strings.TrimSpace(cellAt(f, sheet, r, cSeq)),
				// 归属当前清单组 / 定额块（结构行上读到的身份）。
				ItemCode: itemCode, ItemTitle: itemTitle, ItemUnit: itemUnit,
				Kind:          normalizeKind(level),
				ResourceCode:  code,
				ResourceTitle: cellDisplay(f, sheet, r, cTitle),
				Unit:          cellDisplay(f, sheet, r, cUnit),
				Quantity:      num(cQty, r),
				Price:         num(cPrice, r),
				PriceFormula:  expr(cPrice, r),
				Amount:        num(cAmount, r),
				QuotaCode:     quotaCode,
			}
			if res, ok := resByCode[code]; ok {
				line.ResourceTitle = firstNonEmpty(line.ResourceTitle, res.Title)
				line.Unit = firstNonEmpty(line.Unit, res.Unit)
			}
			b.Lines = append(b.Lines, line)
		default:
			qty := num(cQty, r)
			if code == "" && qty == 0 {
				continue // 空槽 / 小计行 / 说明行
			}
			if level == "" {
				// 漏填层级的明细行（太原 #2 同款）：工作簿里它仍计入定额金额，
				// 库内却缺这条消耗，两边对不上还查不出为什么——必须点名。
				b.skip(f, sheet, r, fmt.Sprintf("明细行「%s」层级为空，已跳过——工作簿里它仍计入定额单价，库内会缺这条消耗；把层级填成 人工/材料/机械/外委 后重导",
					firstNonEmpty(code, fmt.Sprintf("含量 %g", qty))))
				continue
			}
			b.skip(f, sheet, r, fmt.Sprintf("层级「%s」不是 清单/定额/人工/材料/机械/外委（编码 %s），已跳过", level, code))
		}
	}
	return nil
}

// ── 清单（参考表） ──────────────────────────────────────────────────

// billHeaderScanRows 清单表找表头搜到第几行（表头上面可能有标题带、说明行、按
// 单项工程分的汇总块）。
const billHeaderScanRows = 30

// parseBillSheet 解析清单表——**参考表**（2026-11-27 用户拍板：导入的硬契约只有
// 综合单价表，其余表交给模型自由发挥，模板只是参考）。它给综合单价表里已识别的
// 清单项补工程量 / 分部 / 工序特征；综合单价表没列到的编码补进来，标记待补消耗量。
//
// 列一律按列名关键词定位，不按列位：模型可以调列序、加列（如加一列「单项工程」顶替
// 「分部」）、按单项工程分段插标题行——标题行没有编码，自然跳过。工程量以本表为
// 准；本表没有这一行、或工程量格空着时，保留综合单价表「清单工程量」列读到的值。
func parseBillSheet(f *excelize.File, sheet string, b *ProjectBundle) {
	// 表头可能被标题带、说明行、单项工程汇总块压到第 3 行以下，搜到第 30 行；优先
	// 取同时带「编码」列的那一行——上面的单项工程汇总块也有「名称 / 工程量」列，
	// 但它不是清单明细。
	hdr := 0
	for r := 1; r <= billHeaderScanRows; r++ {
		vals := rowValues(f, sheet, r, maxProbeCols)
		if !headerRowHasKeys(vals, []string{"名称", "工程量"}) {
			continue
		}
		if hdr == 0 {
			hdr = r
		}
		if colIndex(vals, false, "清单编码", "编码") > 0 {
			hdr = r
			break
		}
	}
	if hdr == 0 {
		b.Warnings = append(b.Warnings, "清单表未识别到表头（要有 名称 / 工程量 这两列）——工程量只用综合单价表「清单工程量」列读到的值，分部与工序特征未导入")
		return
	}
	header := rowValues(f, sheet, hdr, maxProbeCols)
	// 列名优先取更具体的（清单编码 / 项目名称 / 计量单位），免得被「单项工程名称」
	// 「单位工程」这类列劫持；「单项工程」「单位工程」列可以顶替「分部」。
	cDiv := colIndex(header, false, "分部", "单项工程", "单位工程")
	cCode := colIndex(header, false, "清单编码", "编码")
	cTitle := colIndex(header, false, "项目名称", "名称")
	cFeature := colIndex(header, false, "工序特征", "项目特征")
	cUnit := colIndex(header, false, "计量单位", "单位")
	if cUnit > 0 && strings.Contains(header[cUnit-1], "工程") {
		cUnit = colIndex(header, true, "单位") // 「单位工程」不是计量单位
	}
	cQty := colIndex(header, false, "工程量")
	cUP := colIndex(header, false, "综合单价")
	if cCode == 0 || cTitle == 0 || cQty == 0 {
		b.Warnings = append(b.Warnings, fmt.Sprintf("清单表表头缺 编码 / 名称 / 工程量 列（认到的表头：%s），清单表未导入", strings.Join(nonEmpty(header), " | ")))
		return
	}

	byCode := map[string]*ProjectItem{}
	for i := range b.Items {
		byCode[b.Items[i].Code] = &b.Items[i]
	}
	maxRow := sheetMaxRow(f, sheet, maxSheetRows)
	for r := hdr + 1; r <= maxRow; r++ {
		title := strings.TrimSpace(cellAt(f, sheet, r, cTitle))
		code := strings.TrimSpace(cellAt(f, sheet, r, cCode))
		if code == "" && title == "" {
			continue
		}
		// 「小计/合计」行不是清单项（与综合单价表同规则跳过）。
		if strings.Contains(title, "小计") || strings.Contains(title, "合计") ||
			strings.Contains(code, "小计") || strings.Contains(code, "合计") {
			continue
		}
		div := strings.TrimSpace(cellAt(f, sheet, r, cDiv))
		cellQty, _ := excelize.CoordinatesToCellName(cQty, r)
		qtyExpr := formulaOf(f, sheet, cellQty)
		qty := cellValue(f, sheet, cellQty)
		if it, ok := byCode[code]; ok && code != "" {
			if it.Title == "" {
				it.Title = title // 综合单价的清单行只填了编码时，名称以清单表为准
			}
			it.Division = firstNonEmpty(div, it.Division)
			it.Feature = strings.TrimSpace(cellAt(f, sheet, r, cFeature))
			if it.Unit == "" {
				it.Unit = strings.TrimSpace(cellAt(f, sheet, r, cUnit))
			}
			// 工程量以清单表为准；这一格空着时保留综合单价表「清单工程量」列的值。
			if qty > 0 || it.Quantity == 0 {
				it.Quantity, it.QuantityExpr = qty, qtyExpr
				if qty > 0 {
					b.BillQuantityRows++
				}
			}
			if cUP > 0 && it.ListPrice <= 0 {
				cellUP, _ := excelize.CoordinatesToCellName(cUP, r)
				it.ListPrice = cellValue(f, sheet, cellUP)
				it.ListPriceExpr = formulaOf(f, sheet, cellUP)
			}
			it.CategoryPath = "综合单价/" + firstNonEmpty(it.Division, it.Title)
			continue
		}
		if code != "" {
			it := ProjectItem{
				Code: code, Title: title, Unit: strings.TrimSpace(cellAt(f, sheet, r, cUnit)),
				Division: div, Feature: strings.TrimSpace(cellAt(f, sheet, r, cFeature)),
				Quantity: qty, QuantityExpr: qtyExpr,
				CategoryPath: "综合单价/" + firstNonEmpty(div, title),
			}
			if cUP > 0 {
				cellUP, _ := excelize.CoordinatesToCellName(cUP, r)
				it.ListPrice = cellValue(f, sheet, cellUP)
				it.ListPriceExpr = formulaOf(f, sheet, cellUP)
			}
			b.Items = append(b.Items, it)
			b.Warnings = append(b.Warnings, fmt.Sprintf(
				"清单项 %s（%s）在综合单价表无汇总行——消耗量待补", code, title))
		}
	}
}

// ── 费用汇总 ────────────────────────────────────────────────────────

// parseSummarySheet 解析费用汇总表：只有清单累计后的工程费 + 取费链（六表模版
// 2026-11-25 用户拍板「费用汇总拆成 清单表+费用汇总表；旧版用模型解析，不用兼
// 容」）。清单行一律来自「清单」表（parseBillSheet），本表只走取费段。
func parseSummarySheet(f *excelize.File, sheet string, b *ProjectBundle) {
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
	// 定位取费区表头（含「费用名称」），拿到「费率」「金额」列号。
	feeHdr := 0
	feeRateCol := 0
	feeAmtCol := 0
	for r := 1; r <= maxRow; r++ {
		vals := rowValues(f, sheet, r, 11)
		joined := strings.Join(vals, "|")
		if strings.Contains(joined, "费用名称") || strings.Contains(joined, "费用类别") {
			feeHdr = r
			feeRateCol = colIndex(vals, false, "费率")
			feeAmtCol = colIndex(vals, false, "金额")
			break
		}
	}

	found := false
	for r := 1; r <= maxRow; r++ {
		vals := rowValues(f, sheet, r, 11)
		if len(nonEmpty(vals)) == 0 {
			continue
		}
		// 各段金额（缓存值）：按列序第一个命中取费词的格子定段，所以「分部小计」
		// 块的表头（占直接费）与合计行不会被当成取费行。
		if kind := feeRowKind(vals); kind != "" {
			if v := feeRowAmount(f, sheet, r, vals, feeHdr > 0 && r > feeHdr, feeAmtCol); v > 0 {
				b.Fee.setAmount(kind, v)
			}
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
			// 只认纯控制价行，先到先得（2026-11-26 造价反馈，百锦路 #7）：此前
			// 任何含「控制价」的行都命中、后行覆盖前行——「控制价利用率 62.64%」
			// 排在真控制价之后，8,913,200 被读成了 0.63。
			if !isControlPriceLabel(name) || b.Fee.ControlPrice > 0 {
				continue
			}
			for c := 1; c <= len(vals); c++ {
				cell, _ := excelize.CoordinatesToCellName(c, r)
				if v := cellValue(f, sheet, cell); v > 0 {
					b.Fee.ControlPrice = v
					break
				}
			}
		}
	}
	if b.Fee.ControlPrice > 0 && b.Fee.ControlPrice < ControlPriceSuspectFloor {
		// 读到了但小得不像一个工程的招标控制价：多半是取到了同一行/同名行
		// 的比率。演练与正式回执都看得到告警，别等 summary 算出「占控制价
		// 140000%」才发现。
		b.Warnings = append(b.Warnings, fmt.Sprintf("%s %.2f，疑似误读（费用汇总里含「控制价」的行取到了非金额值）——核对费用汇总的控制价行，或 import_project 带 control_price 显式给定",
			ControlPriceSuspectPrefix, b.Fee.ControlPrice))
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

// ControlPriceSuspectPrefix 控制价疑似误读告警的前缀（调用方显式给了 control_price
// 时按它摘掉这条告警）。
const ControlPriceSuspectPrefix = "控制价读到了"

// ControlPriceSuspectFloor 控制价低于它就提示疑似误读（元）。一个工程的招标
// 控制价不会只有几百块；取到的是比率（0.63）或百分比读数时落在这以下。
const ControlPriceSuspectFloor = 1000

// controlPriceNoiseWords 出现在行名里就说明这一行说的是控制价的派生量
// （利用率 / 下浮率 / 占比 / 差额），不是控制价本身。
var controlPriceNoiseWords = []string{"率", "比", "占", "下浮", "差", "结余"}

// isControlPriceLabel 报告费用汇总的行名是不是在说控制价本身。
func isControlPriceLabel(name string) bool {
	if !strings.Contains(name, "控制价") {
		return false
	}
	for _, w := range controlPriceNoiseWords {
		if strings.Contains(name, w) {
			return false
		}
	}
	return true
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

	maxRow := sheetMaxRow(f, sheet, maxSheetRows)
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
	// ④ 工程量：清单表与综合单价「清单工程量」列都没给值的清单项，落库后合价为 0
	// （清单表是参考表，可以没有——所以这条必须显形，不能让它悄悄过去）。
	noQty := 0
	for _, it := range b.Items {
		if it.Quantity == 0 {
			noQty++
		}
	}
	if noQty > 0 {
		warns = append(warns, fmt.Sprintf(
			"%d 条清单项的工程量为 0（清单表没有这一行、或综合单价表「清单工程量」列没有缓存值）——落库后这些项合价为 0；先 xlsx_recalc，或在清单表补上工程量", noQty))
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

// cellDisplay 取第 row 行第 col 列的显示值（col=0 返回空）：公式格取缓存值，
// 不取公式原文——三级树表明细行的名称 / 单位是 VLOOKUP 取价公式，cellText
// 会把整条公式当成名称。
func cellDisplay(f *excelize.File, sheet string, row, col int) string {
	if col <= 0 {
		return ""
	}
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return ""
	}
	v, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
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
