// projectimport.go —— 测算工作簿解析器。
//
// **硬契约只有综合单价表**（2026-11-27 用户拍板）：它必须是三级树表（样表 v5.1 起），
// 其余五张表是参考表——交给模型自由发挥，模板只是参考，有就读、没有或认不出就跳过：
//
//	综合单价   —— 序号|层级|编码|名称|单位|工程量|单价|金额|清单工程量|合价|人工|材料|机械|备注【硬约束】
//	              行型看「层级」：清单 → 定额 → 人材机；换算系数在定额行「工程量」列
//	清单       —— 序号|分部|编码|项目名称|工序特征|单位|工程量|综合单价|合价【参考】
//	              补工程量 / 分部 / 工序特征；列按列名定位，可按单项工程改版式
//	工料机价格 —— 编码|自有码|类别|名称|规格|单位|不含税单价|来源|备注【参考】
//	封面       —— 书名带 + 项目名（B2；「项目名称」标签值同步）/地点/工期/口径自述【参考】
//	费用汇总   —— 分部构成段 + 取费链（管理费/利润/规费/税/控制价），行 16–23【参考】
//	工程量计算 —— 编码|名称|数值|单位|公式/口径|来源|去向|备注【参考】
//
// 更早的「行类」版式（旺平 2026-10-07 / 什邡 2026-10-06 / 旧内置样表）与五表模版
// 已退役：用户拍板以后不再上传，解析器不认（表头不是三级树表即拒收）。
//
// 本解析器把工作簿映射到工料法三层：资源库 / 消耗定额（含量行）/ 分部分项清单，
// 并把工程量与取费参数一并带出。**只解析、不落库**——落库由调用方在用户确认后执行。
//
// 公式单元格（台班单价 =900+F4*90+F5、数量 =1/3/2）：优先取公式串保留口径，
// 同时取缓存值作为数值。无缓存值时数值为 0 并记入 Warnings（不静默当 0 用）。
package workcost

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Sheet 名常量（模版固定命名；容错见 sheetByName）。
const (
	sheetCover      = "封面"
	sheetBill       = "清单"
	sheetSummary    = "费用汇总"
	sheetUnitPrice  = "综合单价"
	sheetResourcePr = "工料机价格"
	sheetQuantity   = "工程量计算"
)

// 备选 sheet 名（什邡工程量表名为「工程量核算」，清单表名为「清单测算」）。
var sheetAliases = map[string][]string{
	sheetBill:       {"清单", "分部分项清单"},
	sheetSummary:    {"费用汇总", "清单测算"},
	sheetQuantity:   {"工程量计算", "工程量核算"},
	sheetUnitPrice:  {"综合单价"},
	sheetResourcePr: {"工料机价格"},
}

// ProjectResource 解析出的工料机资源（工料机价格表一行）。
type ProjectResource struct {
	Code         string  `json:"code"`
	GCode        string  `json:"gcode"` // 库内自有码（六表模版携带；回导对行用）
	Kind         string  `json:"kind"`
	Title        string  `json:"title"`
	Spec         string  `json:"spec"`
	Unit         string  `json:"unit"`
	Price        float64 `json:"price"`
	PriceFormula string  `json:"priceFormula"` // 台班公式原文（留口径追溯）
	Source       string  `json:"source"`
	Note         string  `json:"note"`
}

// ProjectConsumption 解析出的消耗量行（综合单价表明细行 = 工序消耗）。
type ProjectConsumption struct {
	Seq           string  `json:"seq"` // 1.1 / 2.3 明细序号
	ItemCode      string  `json:"itemCode"`
	ItemTitle     string  `json:"itemTitle"`
	ItemUnit      string  `json:"itemUnit"`
	Kind          string  `json:"kind"` // 层级：人工/材料/机械/外委（已归一成库内类别）
	ResourceCode  string  `json:"resourceCode"`
	ResourceTitle string  `json:"resourceTitle"`
	Unit          string  `json:"unit"`
	Quantity      float64 `json:"quantity"`
	Price         float64 `json:"price"`
	PriceFormula  string  `json:"priceFormula"`
	Amount        float64 `json:"amount"`
	// QuotaCode 本明细行归属的定额码（层级=定额那一行的编码；三级树表的人材机
	// 必挂在定额块下，解析端保证非空）。
	QuotaCode string `json:"quotaCode"`
}

// ProjectItem 解析出的清单项（综合单价表汇总行 + 费用汇总的工程量）。
type ProjectItem struct {
	Code         string  `json:"code"` // 清单编码（WP01/BJ01…）
	Title        string  `json:"title"`
	Unit         string  `json:"unit"`
	Division     string  `json:"division"` // 分部（A临建/B支护降水…）
	Feature      string  `json:"feature"`  // 工序特征
	Quantity     float64 `json:"quantity"` // 工程量（费用汇总引用工程量表）
	QuantityExpr string  `json:"quantityExpr"`
	// ListPrice 原表综合单价（汇总行/费用汇总的「综合单价」列，原表口径快照；
	// 0=工作簿未给）。落 gf_bill_items.list_price 做对照显形与案例样本，不参与现算。
	ListPrice     float64 `json:"listPrice"`
	ListPriceExpr string  `json:"listPriceExpr"` // 综合单价计算式原文（公式无缓存值时告警用）
	CategoryPath  string  `json:"categoryPath"`  // 综合单价/分部
}

// ProjectQuotaBlock 三级行组里的一个定额块：清单项下挂的一条定额（编码/名称/
// 工程量系数）。回导时按 ResolveQuota 逐块解析（撞码消歧同款），1:N 挂接落
// gf_bill_quota_links；系数 0=跟随清单工程量（与挂接面语义一致）。
type ProjectQuotaBlock struct {
	ItemCode string `json:"itemCode"` // 所属清单编码
	Code     string `json:"code"`     // 定额编码（工作簿原码，解析端消歧）
	Title    string `json:"title"`    // 定额名称（空=沿用清单标题/库内既有）
	Unit     string `json:"unit"`     // 定额计量单位（空=跟随清单单位）
	// Factor 换算系数 = 每一个清单单位包含几个定额单位（定额行「工程量」列；
	// 清单 m³、定额 100m³ 时 =0.01）。0=跟随清单工程量（即按 1 计）；系数必须进
	// 金额链：清单综合单价 = Σ(定额单价 × 系数)。
	Factor float64 `json:"factor"`
}

// ProjectFeeParams 解析出的取费参数（费用汇总下半段）。费率之外带各段金额
// （工作簿缓存值；没重算的簿取不到，恒 0）——导入回执拿它与库内现算对账，
// 差异按性质拆开说（2026-11-25 铜陵反馈 #15：122,785 元差异里 98% 是利润
// 基数口径分叉，被当成「精度舍位」手算了 5 轮）。
type ProjectFeeParams struct {
	ManagementRate    float64 `json:"managementRate"`
	RegulatoryRate    float64 `json:"regulatoryRate"`
	ProfitRate        float64 `json:"profitRate"`
	TaxRate           float64 `json:"taxRate"`
	ControlPrice      float64 `json:"controlPrice"`
	ManagementFormula string  `json:"managementFormula"`
	TaxNote           string  `json:"taxNote"`
	// HasAmounts 报告金额段是否取到了缓存值（至少直接费与含税总造价非零）。
	HasAmounts       bool    `json:"hasAmounts"`
	DirectAmount     float64 `json:"directAmount"`
	ManagementAmount float64 `json:"managementAmount"`
	ProfitAmount     float64 `json:"profitAmount"`
	RegulatoryAmount float64 `json:"regulatoryAmount"`
	PreTaxAmount     float64 `json:"preTaxAmount"`
	TaxAmount        float64 `json:"taxAmount"`
	TotalAmount      float64 `json:"totalAmount"`
}

// ProjectQuantity 解析出的工程量计算条目（工程量计算表）。
type ProjectQuantity struct {
	Code  string  `json:"code"`
	Title string  `json:"title"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Note  string  `json:"note"`
}

// SkippedRow 解析时跳过的行（显形而非静默丢弃）。
type SkippedRow struct {
	Sheet string `json:"sheet"`
	Row   int    `json:"row"`
	// Reason 跳过原因。
	Reason string `json:"reason"`
	// Excerpt 该行原文摘要（非空格逐格拼接，截断）——只给计数不给位置时，
	// 一句脚注被当成资源行要另写脚本逐行扫才找得到（2026-11-25 铜陵反馈 #4）。
	Excerpt string `json:"excerpt"`
}

// ProjectBundle 一份项目工作簿的完整解析结果（待用户确认后落库）。
type ProjectBundle struct {
	Path      string               `json:"path"`
	FileName  string               `json:"fileName"`
	Project   string               `json:"project"`
	Location  string               `json:"location"`
	Duration  string               `json:"duration"`
	Pricing   string               `json:"pricing"` // 封面的组价口径自述
	Resources []ProjectResource    `json:"resources"`
	Items     []ProjectItem        `json:"items"`
	Lines     []ProjectConsumption `json:"lines"`
	// QuotaBlocks 三级行组（测算模版「清单→定额→工料机」）：每个定额块=清单下
	// 挂的一条定额（编码/名称/工程量系数）。空=旧 1:1 形态；apply 逐块 ResolveQuota
	// 落定额并把 1:N 挂接写进 gf_bill_quota_links（库的一等模型）。
	QuotaBlocks []ProjectQuotaBlock `json:"quotaBlocks"`
	// BillQuantityRows 工程量取自清单表的清单项条数（其余取自综合单价「清单工程量」
	// 列，或没有工程量）——导入回执据此写明工程量来源。
	BillQuantityRows int               `json:"billQuantityRows"`
	Fee              ProjectFeeParams  `json:"fee"`
	Quantities       []ProjectQuantity `json:"quantities"`
	Skipped          []SkippedRow      `json:"skipped"`
	Warnings         []string          `json:"warnings"`
	Sheets           []string          `json:"sheets"`
	// 导入口径——规范值，由调用方给（规矩见 rules.go）；文件里没有就不猜，空=待整理：
	// Specialty 项目默认专业（字典值），本次新建的定额专业随它；Region 项目所在
	// 地区（字典值），工料机按它分行；Period 计价期（YYYY-MM），工料机价格事实的期。
	// 解析器不填这三个字段。
	Specialty string `json:"specialty"`
	Region    string `json:"region"`
	Period    string `json:"period"`

	// CoverScope 封面上写明的口径（「专业：X」「地区：Y」「计价期：Z」）——
	// 只是**文件里的自述**，不是规范值：导入时调用方没给口径才拿它兜底，且仍要
	// 过字典校验（见 app 侧 importProject）。与上面三个字段分开，保持「规范值
	// 由调用方给」的契约不破。
	CoverSpecialty string `json:"coverSpecialty"`
	CoverRegion    string `json:"coverRegion"`
	CoverPeriod    string `json:"coverPeriod"`
}

// skip 记一行跳过（带原文摘要）。
func (b *ProjectBundle) skip(f *excelize.File, sheet string, row int, reason string) {
	b.Skipped = append(b.Skipped, SkippedRow{Sheet: sheet, Row: row, Reason: reason, Excerpt: rowExcerpt(f, sheet, row, maxProbeCols)})
}

// rowExcerpt 一行的原文摘要：非空格按列序拼接，每格截到 16 字、最多 6 格。
func rowExcerpt(f *excelize.File, sheet string, row, maxCol int) string {
	var parts []string
	for c := 1; c <= maxCol && len(parts) < 6; c++ {
		v := strings.TrimSpace(cellAt(f, sheet, row, c))
		if v == "" {
			continue
		}
		r := []rune(v)
		if len(r) > 16 {
			v = string(r[:16]) + "…"
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, " | ")
}

// IsEstimateWorkbook 报告该工作簿是不是测算簿（供导入入口分流）：有「综合单价」表
// 就算（表名、别名，或表名含「综合单价」）。
//
// 2026-11-27 用户拍板：导入的**硬契约只有综合单价表**（三级树表结构）；其余五张
// 表是参考，交给模型自由发挥——清单表按单项工程改版式、封面与费用汇总重排都不拦。
// 此前要求「清单 + 综合单价 + 工料机价格」三张表同在才认，少一张整份拒收。
func IsEstimateWorkbook(sheets []string) bool {
	for _, s := range sheets {
		n := strings.TrimSpace(s)
		for _, a := range sheetAliases[sheetUnitPrice] {
			if n == a {
				return true
			}
		}
		if strings.Contains(n, sheetKeyWords[sheetUnitPrice]) {
			return true
		}
	}
	return false
}

// ParseProjectWorkbook 解析一份测算工作簿。
//
// 综合单价表是核心与唯一硬约束（缺了、或不是三级树表 → 整份拒收）；其余表有就
// 读、没有或认不出就跳过并告警：
//   - 清单表：补工程量 / 分部 / 工序特征，按列名关键词定位，列序、加列、按单项
//     工程分段的标题行都不影响；没有这一行时工程量退回综合单价表「清单工程量」列。
//   - 工料机价格：资源与价格；没有时消耗行按库内既有编码解析。
//   - 封面 / 费用汇总 / 工程量计算：项目名、费率、工程量溯源，都只是补充。
func ParseProjectWorkbook(path string) (*ProjectBundle, error) {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, fmt.Errorf("打开工作簿失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	b := &ProjectBundle{
		Path:     filepath.ToSlash(path),
		FileName: filepath.Base(path),
		Sheets:   f.GetSheetList(),
		// 切片构造即非 nil：nil 经 Wails 序列化成 JSON null，前端
		// `bundle.warnings.length` / `bundle.items.length` 会崩（与 Compose 同根）。
		Resources:  []ProjectResource{},
		Items:      []ProjectItem{},
		Lines:      []ProjectConsumption{},
		Quantities: []ProjectQuantity{},
		Skipped:    []SkippedRow{},
		Warnings:   []string{},
	}
	// taken 已经认领的表：一张表只当一种用途（「清单综合单价」先归综合单价）。
	taken := map[string]bool{}
	pick := func(canonical string) string {
		sn := sheetByName(f, canonical, taken)
		taken[sn] = true
		return sn
	}
	upSheet := pick(sheetUnitPrice)
	if upSheet == "" {
		return nil, fmt.Errorf("不是测算工作簿：找不到「%s」表——它是测算簿的核心与唯一硬约束（三级树表：层级 / 编码 / 名称 三列表头），实得表：%v。其余表只是参考，缺了或改了版式不拦导入；别的格式的资料请走模型读表导入（import_open → import_add → import_check → import_commit）",
			sheetUnitPrice, b.Sheets)
	}
	// 综合单价读不了 = 整份拒收：定额与消耗行是工作簿的主体，缺了它们清单行
	// 会全部「无消耗」入库，带病入库比不入更糟。
	if err := parseUnitPriceSheet(f, upSheet, b); err != nil {
		return nil, err
	}
	if sn := pick(sheetResourcePr); sn != "" {
		parseResourceSheet(f, sn, b)
	} else {
		b.Warnings = append(b.Warnings, "工作簿没有工料机价格表：消耗行引用的工料机按库内既有编码解析，库内没有的按零价核算")
	}
	if sn := pick(sheetBill); sn != "" {
		parseBillSheet(f, sn, b)
	}
	if sn := pick(sheetCover); sn != "" {
		parseCover(f, sn, b)
	}
	if sn := pick(sheetSummary); sn != "" {
		parseSummarySheet(f, sn, b)
	} else {
		b.Warnings = append(b.Warnings, "工作簿没有费用汇总表：取费费率未读到（落库为 0，各段不计取）——要取费口径就把费用汇总表带上，或导入后 cost_project_apply action=update_rates 补")
	}
	if sn := pick(sheetQuantity); sn != "" {
		parseQuantitySheet(f, sn, b)
	}

	b.Warnings = append(b.Warnings, crossCheck(b)...)
	return b, nil
}

// sheetKeyWords 各表表名的关键词（包含匹配）：表名常带后缀（「综合单价分析表」
// 「工料机价格表」），按关键词也认。空 = 只认名称与别名。
var sheetKeyWords = map[string]string{
	sheetCover:      "封面",
	sheetBill:       "清单",
	sheetSummary:    "费用汇总",
	sheetUnitPrice:  "综合单价",
	sheetResourcePr: "工料机",
}

// sheetByName 按名称找 sheet：先按名称 / 别名精确匹配，再按关键词包含匹配；
// taken 里的表不再认领。找不到返回空串。
func sheetByName(f *excelize.File, canonical string, taken map[string]bool) string {
	list := f.GetSheetList()
	names := sheetAliases[canonical]
	if len(names) == 0 {
		names = []string{canonical}
	}
	for _, want := range names {
		for _, sn := range list {
			if strings.TrimSpace(sn) == want && !taken[sn] {
				return sn
			}
		}
	}
	if key := sheetKeyWords[canonical]; key != "" {
		for _, sn := range list {
			if !taken[sn] && strings.Contains(strings.TrimSpace(sn), key) {
				return sn
			}
		}
	}
	return ""
}

// ── 单元格读取 ──────────────────────────────────────────────────────

// cellText 取单元格文本（公式取公式原文，否则取显示值）。
func cellText(f *excelize.File, sheet, cell string) string {
	if formula, err := f.GetCellFormula(sheet, cell); err == nil && strings.TrimSpace(formula) != "" {
		return "=" + strings.TrimSpace(formula)
	}
	v, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

// cellValue 取单元格数值：有缓存值时用它；无缓存但公式指向另一单元格
// （如 =工料机价格!F44）则递归取一次目标值；否则 0。
//
// 读的是**存储值**，不是显示值：含量列的格式常是 0.####（最多四位小数），按
// 显示值读，0.00015 读成 0.0001、0.00002 读成 0——沿江包1 实测活性炭 / PAM
// 两行含量被吃掉，「废水处理运行」表里 34.70、读进库 33.91，表上还看不出来
// （表内公式用的正是存储值）。
func cellValue(f *excelize.File, sheet, cell string) float64 {
	if v, err := f.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true}); err == nil {
		if n, ok := parseNum(v); ok {
			return n
		}
	}
	// 公式：尝试解析简单引用（='表'!F44 或 =F44）取目标缓存的公式或值。
	formula, err := f.GetCellFormula(sheet, cell)
	if err != nil || strings.TrimSpace(formula) == "" {
		return 0
	}
	return resolveFormulaRef(f, sheet, formula)
}

// resolveFormulaRef 解析形如 =工料机价格!F44 / ='工料机价格'!F44 / =F44 的引用。
// 仅支持单层引用（模版里的跨表取价就是这一形态）；算术公式（台班费）不做
// 求值——那需要电子表格引擎，且原表已存缓存值，走 cellValue 第一段即可。
func resolveFormulaRef(f *excelize.File, sheet, formula string) float64 {
	expr := strings.TrimPrefix(strings.TrimSpace(formula), "=")
	expr = strings.ReplaceAll(expr, "$", "")
	target := expr
	if i := strings.Index(expr, "!"); i >= 0 {
		refSheet := strings.Trim(strings.TrimSpace(expr[:i]), "'")
		target = strings.TrimSpace(expr[i+1:])
		if refSheet == "" || !isCellRef(target) {
			return 0
		}
		if v, err := f.GetCellValue(refSheet, target, excelize.Options{RawCellValue: true}); err == nil {
			if n, ok := parseNum(v); ok {
				return n
			}
		}
		// 目标本身也是公式 → 递归一次（更深层的算术公式不求值）。
		if fo, err := f.GetCellFormula(refSheet, target); err == nil && strings.TrimSpace(fo) != "" {
			return resolveFormulaRef(f, refSheet, fo)
		}
		return 0
	}
	if !isCellRef(target) {
		return 0
	}
	return 0
}

// isCellRef 判断是否为纯单元格引用（列字母+行号）。
func isCellRef(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	for i < len(s) && (s[i] >= 'A' && s[i] <= 'Z' || s[i] >= 'a' && s[i] <= 'z') {
		i++
	}
	if i == 0 || i == len(s) {
		return false
	}
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// parseNum 宽松解析数字（含千分位、百分号、全角空格）。
func parseNum(s string) (float64, bool) {
	t := strings.TrimSpace(s)
	t = strings.ReplaceAll(t, ",", "")
	t = strings.ReplaceAll(t, "，", "")
	t = strings.ReplaceAll(t, "\u00a0", "")
	if t == "" {
		return 0, false
	}
	if strings.HasSuffix(t, "%") {
		if v, err := strconv.ParseFloat(strings.TrimSuffix(t, "%"), 64); err == nil {
			return v / 100, true
		}
		return 0, false
	}
	v, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// rowValues 读一行显示值（1-based 行号，maxCol 列）。
//
// ⚠️ 合并单元格：excelize 对合并区域内的**每个**格子都返回同一文本（实测
// B2..G2 全等），直接拼接会把标题重复 N 次、把合并的说明串误当「标签-值」对。
// 故这里对**相邻重复**去重：合并区一定连续，去重即还原合并语义（保留列位）。
func rowValues(f *excelize.File, sheet string, row, maxCol int) []string {
	out := make([]string, 0, maxCol)
	prev := ""
	for c := 1; c <= maxCol; c++ {
		cell, err := excelize.CoordinatesToCellName(c, row)
		if err != nil {
			out = append(out, "")
			continue
		}
		v := cellText(f, sheet, cell)
		if v != "" && v == prev {
			out = append(out, "")
			continue
		}
		out = append(out, v)
		if v != "" {
			prev = v
		}
	}
	return out
}

// rowIsMergedUniform 判断该行是否为「水平合并的整段文本」（非空格子文本全同）。
// 标题行/整段说明行属此形态，其中的「标签-值」配对扫描必须跳过。
func rowIsMergedUniform(vals []string) bool {
	first := ""
	n := 0
	for _, v := range vals {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if n == 0 {
			first = v
		} else if v != first {
			return false
		}
		n++
	}
	return n >= 2
}

// findHeaderRow 在前 limit 行内查找表头行（返回 1-based 行号，0=未找到）。
// 判据见 headerRowHasKeys：各关键词落在不同的列上，且该行不止一格有字。
func findHeaderRow(f *excelize.File, sheet string, maxCol, limit int, keys ...string) int {
	for r := 1; r <= limit; r++ {
		if headerRowHasKeys(rowValues(f, sheet, r, maxCol), keys) {
			return r
		}
	}
	return 0
}

// headerRowHasKeys 表头行判据：每个关键词落在**不同的列**上，且该行不止一格有字。
//
// 不能按整行拼接匹配：模版与实测工作簿常在表头上方放一行填表说明，「汇总行只填
// 清单编码…明细行填行类…」这类话把几个表头词写进同一格，整行匹配会把说明行
// 认成表头，所有列随之错位到第 1 列——清单项数碰巧还对，定额块与明细全丢
// （2026-11-25 测算模版往返用例抓到）。
func headerRowHasKeys(vals, keys []string) bool {
	nonEmpty := 0
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			nonEmpty++
		}
	}
	if nonEmpty < 2 {
		return false
	}
	used := map[int]bool{}
	for _, k := range keys {
		col := colIndex(vals, false, k)
		if col == 0 || used[col] {
			return false
		}
		used[col] = true
	}
	return true
}

// colIndex 在表头行里定位列（1-based；0=未找到）。按 keys 依次匹配，
// exact=true 时要求去空格后完全相等。
func colIndex(header []string, exact bool, keys ...string) int {
	for _, k := range keys {
		k = strings.TrimSpace(k)
		for i, h := range header {
			hn := strings.TrimSpace(h)
			if exact {
				if hn == k {
					return i + 1
				}
				continue
			}
			if strings.Contains(hn, k) {
				return i + 1
			}
		}
	}
	return 0
}
