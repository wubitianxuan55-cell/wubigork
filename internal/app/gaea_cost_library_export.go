package app

// gaea_cost_library_export.go — 造价库列表导出（2026-11-25 造价数据库推倒重建：
// 板块只留浏览/检索/删除/导出，导出=把清单按当前筛选落成 xlsx）。
//
// 三类：分部分项清单（含原表单价与现算单价）/ 消耗定额（主表+含量明细附表）/
// 工料机。筛选与板块列表同口径（projectID / keyword / extra），
// 现算单价走库内 ComposeQuotas 唯一权威链。目标文件已存在时拒绝覆盖。
// （原第四类「资料条目」是旧成本库的导出口，随旧成本库退役拆除，2026-11-26。）

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

// costExportKinds 可导出的三类清单（与板块导出菜单一一对应；值为表名）。
var costExportKinds = map[string]string{
	"bill":     "分部分项清单",
	"quota":    "消耗定额",
	"resource": "工料机",
}

// GaeaCostLibraryExport 把造价库的一类清单导出为 xlsx，返回写出的路径。
//
// kind：bill=分部分项清单｜quota=消耗定额（主表+含量明细附表）｜resource=工料机｜
// template=成本测算模版。filter 与板块列表的当前筛选同口径（见
// workcost.LibraryFilter；未给的字段=不过滤）。outPath 是目标 .xlsx 的完整路径；
// 已存在则拒绝覆盖。
//
// kind=template 把测算样表复制到 outPath（2026-11-25 删渲染器拍板：样表是版式
// 真源，程序不渲染、不预填——工料机价格与费率由模型查库后填；filter 保留签名兼容，
// 不再使用）。样表填出的测算簿走 cost_library 的 import_project 导回：综合单价表是
// 三级树表即可，其余表是参考。
func (a *App) GaeaCostLibraryExport(kind string, filter workcost.LibraryFilter, outPath string) (string, error) {
	out, err := costOutPath(outPath)
	if err != nil {
		return "", err
	}
	outPath = out
	if kind == "template" {
		return a.costExportTemplate(filter, outPath)
	}
	sheet, ok := costExportKinds[kind]
	if !ok {
		return "", fmt.Errorf("不支持的导出类型 %q（bill/quota/resource/template）", kind)
	}
	filter.Keyword = strings.ToLower(strings.TrimSpace(filter.Keyword))

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetName(f.GetSheetName(0), sheet); err != nil {
		return "", err
	}
	var (
		headers []string
		rows    [][]any
		werr    error
	)
	switch kind {
	case "bill":
		headers, rows, werr = a.costExportBill(filter)
	case "quota":
		headers, rows, werr = a.costExportQuota(filter, f)
	case "resource":
		headers, rows, werr = a.costExportResource(filter)
	}
	if werr != nil {
		return "", werr
	}
	if err := costExportSheet(f, sheet, headers, rows); err != nil {
		return "", fmt.Errorf("写出「%s」失败: %w", sheet, err)
	}
	if err := costWriteFile(outPath, func(path string) error { return f.SaveAs(path) }); err != nil {
		return "", err
	}
	return filepath.ToSlash(outPath), nil
}

// costOutPath 目标路径规整：非空、补 .xlsx 扩展名、已存在则拒绝覆盖。
func costOutPath(outPath string) (string, error) {
	outPath = strings.TrimSpace(outPath)
	if outPath == "" {
		return "", fmt.Errorf("需要输出路径")
	}
	if !strings.EqualFold(filepath.Ext(outPath), ".xlsx") {
		outPath += ".xlsx"
	}
	if _, err := os.Stat(outPath); err == nil {
		return "", fmt.Errorf("文件已存在，换一个文件名：%s", outPath)
	}
	return outPath, nil
}

// costWriteFile 建目录并让 save 落盘（各导出口共用）。
func costWriteFile(outPath string, save func(path string) error) error {
	if dir := filepath.Dir(outPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建导出目录失败: %w", err)
		}
	}
	if err := save(outPath); err != nil {
		return fmt.Errorf("保存工作簿失败: %w", err)
	}
	return nil
}

// costExportTemplate 测算样表出口（板块「导出 → 测算模版」；2026-11-25 用户拍板
// 「删除渲染器思路，提供样本给模型，再进行约束」）：把样表复制到目标路径，
// 《使用说明书》绑着落在旁边（2026-11-27）。版式真源=样表文件（工作区覆盖优先、
// 内嵌种子兜底，见 office_cost_template_sample.go），不再按库渲染、不再预填——
// 工料机价格与费率口径由模型查库填。filter 参数保留绑定签名兼容，内容不再使用。
func (a *App) costExportTemplate(_ workcost.LibraryFilter, outPath string) (string, error) {
	// 说明书路径不回传：绑定签名只回样表路径（前端提示用），说明书就在它旁边。
	if _, _, err := writeCostEstimateSample(outPath); err != nil {
		return "", err
	}
	return filepath.ToSlash(outPath), nil
}

// costExportMatch 关键词包含（已小写）；空关键词恒命中。
func costExportMatch(kw string, fields ...string) bool {
	if kw == "" {
		return true
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), kw) {
			return true
		}
	}
	return false
}

// costExportPick 规范值筛选的命中判定：筛选为空=不过滤；筛选是「未标注」哨兵
// （workcost.FilterNone）=只取该字段为空的行；否则精确相等。
func costExportPick(filter, value string) bool {
	switch strings.TrimSpace(filter) {
	case "":
		return true
	case workcost.FilterNone:
		return strings.TrimSpace(value) == ""
	default:
		return strings.TrimSpace(value) == strings.TrimSpace(filter)
	}
}

// costExportPrice 大于 0 才写数值，否则留空（0 价在造价数据里是「未录入」，
// 导成一列 0 会被下游当成真价）。
func costExportPrice(v float64) any {
	if v > 0 {
		return v
	}
	return ""
}

// costExportBill 分部分项清单：项目 → 清单行，现算单价=挂接定额的综合单价之和。
func (a *App) costExportBill(f workcost.LibraryFilter) ([]string, [][]any, error) {
	s := a.hubWorkcostStore()
	if !s.Available() {
		return nil, nil, fmt.Errorf("工料机造价库不可用")
	}
	projects, err := s.BillProjects()
	if err != nil {
		return nil, nil, err
	}
	type row struct {
		project workcost.BillProject
		item    workcost.BillItem
	}
	var picked []row
	for _, p := range projects {
		if f.ProjectID > 0 && p.ID != f.ProjectID {
			continue
		}
		items, err := s.BillItems(p.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			if !costExportPick(f.Division, it.Division) || !costExportPick(f.Specialty, it.Specialty) {
				continue
			}
			if !costExportMatch(f.Keyword, it.Title, it.Code, it.GCode, it.Feature, it.QuotaCode, strings.Join(it.QuotaCodes, " ")) {
				continue
			}
			picked = append(picked, row{project: p, item: it})
		}
	}
	codeSet := map[string]bool{}
	var codes []string
	for _, r := range picked {
		for _, c := range r.item.QuotaCodes {
			if c != "" && !codeSet[c] {
				codeSet[c] = true
				codes = append(codes, c)
			}
		}
	}
	composes, err := s.ComposeQuotas(codes)
	if err != nil {
		return nil, nil, err
	}
	headers := []string{"项目", "专业", "编码", "自有码", "名称", "项目特征", "单位", "分部", "工程量", "工程量计算式",
		"挂接定额", "手填单价（元）", "原表单价（元）", "现算单价（元·人材机）"}
	rows := make([][]any, 0, len(picked))
	for _, r := range picked {
		unit := 0.0
		for _, c := range r.item.QuotaCodes {
			if cp := composes[c]; cp != nil {
				unit += cp.CompositePrice
			}
		}
		rows = append(rows, []any{
			r.project.Name, r.item.Specialty, r.item.Code, r.item.GCode, r.item.Title, r.item.Feature, r.item.Unit, r.item.Division,
			r.item.Quantity, r.item.QuantityExpr, strings.Join(r.item.QuotaCodes, "+"),
			costExportPrice(r.item.PriceOverride), costExportPrice(r.item.ListPrice), costExportPrice(unit),
		})
	}
	return headers, rows, nil
}

// costExportQuota 消耗定额：主表 + 含量明细附表（附表单独成页）。
func (a *App) costExportQuota(f workcost.LibraryFilter, file *excelize.File) ([]string, [][]any, error) {
	s := a.hubWorkcostStore()
	if !s.Available() {
		return nil, nil, fmt.Errorf("工料机造价库不可用")
	}
	// 专业筛选为空或哨兵时取全量，再按哨兵/章节收窄（「未标注专业」那一桶也要能导）。
	spec := ""
	if f.Specialty != "" && f.Specialty != workcost.FilterNone {
		spec = f.Specialty
	}
	quotas, err := s.ListQuotas(spec, "")
	if err != nil {
		return nil, nil, err
	}
	var picked []workcost.Quota
	for _, q := range quotas {
		if !costExportPick(f.Specialty, q.Specialty) || !costExportPick(f.Chapter, q.Chapter) {
			continue
		}
		if !costExportMatch(f.Keyword, q.Title, q.Code, q.GCode, q.OrigCode, q.Chapter) {
			continue
		}
		picked = append(picked, q)
	}
	var codes []string
	for _, q := range picked {
		codes = append(codes, q.Code)
	}
	composes, err := s.ComposeQuotas(codes)
	if err != nil {
		return nil, nil, err
	}
	usage := map[string]int{}
	for _, q := range picked {
		refs, err := s.QuotaUsage(q.Code)
		if err == nil {
			usage[q.Code] = len(refs)
		}
	}
	headers := []string{"定额编码", "自有码", "原码", "名称", "单位", "专业", "分部", "地区", "借鉴来源", "来源", "含量行数", "被清单引用", "现算综合单价（元·人材机）"}
	rows := make([][]any, 0, len(picked))
	detail := make([][]any, 0)
	for _, q := range picked {
		full, err := s.GetQuota(q.Code)
		if err != nil || full == nil {
			continue
		}
		unit := 0.0
		if cp := composes[q.Code]; cp != nil {
			unit = cp.CompositePrice
		}
		rows = append(rows, []any{q.Code, q.GCode, q.OrigCode, q.Title, q.Unit, q.Specialty, q.Chapter, q.Region, q.DerivedFrom, q.Source,
			len(full.Items), usage[q.Code], costExportPrice(unit)})
		for _, it := range full.Items {
			price := 0.0
			if r, err := s.GetResourceByCode(it.ResourceCode); err == nil {
				price = r.EffectivePrice()
			} else if it.ResourcePrice > 0 {
				price = it.ResourcePrice
			}
			loss := ""
			if it.LossRate > 0 {
				loss = fmt.Sprintf("%.1f%%", it.LossRate*100)
			}
			amount := 0.0
			if it.Quantity > 0 && price > 0 {
				amount = it.Quantity * price * (1 + it.LossRate)
			}
			detail = append(detail, []any{q.Code, q.GCode, q.Title, it.Kind, it.ResourceCode, it.Title, it.Spec, it.Unit,
				costExportPrice(it.Quantity), loss, costExportPrice(price), costExportPrice(amount)})
		}
	}
	const detailSheet = "含量明细"
	if _, err := file.NewSheet(detailSheet); err != nil {
		return nil, nil, err
	}
	detailHeaders := []string{"定额编码", "自有码", "定额名称", "行类", "资源编码", "资源名称", "规格", "单位", "含量", "损耗率", "取用价（元）", "金额（元）"}
	if err := costExportSheet(file, detailSheet, detailHeaders, detail); err != nil {
		return nil, nil, fmt.Errorf("写出「%s」失败: %w", detailSheet, err)
	}
	return headers, rows, nil
}

// costExportResource 工料机：主数据 + 被多少条定额含量行引用。
func (a *App) costExportResource(f workcost.LibraryFilter) ([]string, [][]any, error) {
	s := a.hubWorkcostStore()
	if !s.Available() {
		return nil, nil, fmt.Errorf("工料机造价库不可用")
	}
	kind := f.Kind
	if kind == workcost.FilterNone || kind == workcost.FilterOutside {
		kind = "" // 「未标注类别」没有这一桶（类别必为四类之一）：取全量后按哨兵收窄
	}
	list, err := s.ListResources(kind, "")
	if err != nil {
		return nil, nil, err
	}
	counts, _ := s.ResourceReferenceCounts()
	headers := []string{"编码", "自有码", "类别", "名称", "规格", "单位", "现行价（元）", "基准价（元）", "取用价（元）", "损耗率",
		"分类", "来源", "供应商", "地区", "价格期数", "价格口径", "有效期至", "状态", "备注", "被定额引用"}
	rows := make([][]any, 0, len(list))
	for _, r := range list {
		// 「类别待整理」= 类别不在四类之内的行（与列表的 OUTSIDE 哨兵同口径）。
		if f.Kind == workcost.FilterOutside && workcost.IsKnownKind(r.Kind) {
			continue
		}
		// 分类是路径：选上级分类连带它的下级（「材料/土建材料」导出钢材、水泥等全部）。
		switch {
		case f.Category == "" || f.Category == workcost.FilterNone:
			if !costExportPick(f.Category, r.CategoryPath) {
				continue
			}
		default:
			if r.CategoryPath != f.Category && !strings.HasPrefix(r.CategoryPath, f.Category+"/") {
				continue
			}
		}
		if !costExportPick(f.Region, r.Region) || !costExportPick(f.Period, r.PriceDate) {
			continue
		}
		if !costExportMatch(f.Keyword, r.Title, r.Code, r.GCode, r.Spec, r.CategoryPath) {
			continue
		}
		loss := ""
		if r.LossRate > 0 {
			loss = fmt.Sprintf("%.1f%%", r.LossRate*100)
		}
		rows = append(rows, []any{r.Code, r.GCode, r.Kind, r.Title, r.Spec, r.Unit,
			costExportPrice(r.CurrentPrice), costExportPrice(r.BasePrice), costExportPrice(r.EffectivePrice()), loss,
			r.CategoryPath, r.Source, r.Supplier, r.Region, r.PriceDate, r.PriceType, r.ValidUntil, r.Status, r.Note,
			counts[r.Code]})
	}
	return headers, rows, nil
}

// costExportSheet 流式写一张表：表头加粗冻结，列宽按内容估（中文按 2 个字符宽）。
func costExportSheet(f *excelize.File, sheet string, headers []string, rows [][]any) error {
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}
	widths := make([]float64, len(headers))
	for i, h := range headers {
		widths[i] = float64(costExportWidth(h))
	}
	for _, row := range rows {
		for i, v := range row {
			if i >= len(widths) {
				break
			}
			if w := costExportWidth(v); float64(w) > widths[i] {
				widths[i] = float64(w)
			}
		}
	}
	for i, w := range widths {
		if w > 42 {
			widths[i] = 42
		}
		if w < 9 {
			widths[i] = 9
		}
		if err := sw.SetColWidth(i+1, i+1, widths[i]); err != nil {
			return err
		}
	}
	head := make([]interface{}, len(headers))
	for i, h := range headers {
		head[i] = h
	}
	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := sw.SetRow(cell, head, excelize.RowOpts{StyleID: bold}); err != nil {
		return err
	}
	for ri, row := range rows {
		vals := make([]interface{}, len(headers))
		copy(vals, row)
		cell, _ := excelize.CoordinatesToCellName(1, ri+2)
		if err := sw.SetRow(cell, vals); err != nil {
			return err
		}
	}
	if err := sw.Flush(); err != nil {
		return err
	}
	return f.SetPanes(sheet, &excelize.Panes{Freeze: true, Split: false, XSplit: 0, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
}

// costExportWidth 估显示宽度（rune 计，CJK 粗算 2）。
func costExportWidth(v any) int {
	var s string
	switch x := v.(type) {
	case string:
		s = x
	case float64:
		s = fmt.Sprintf("%.2f", x)
	case int:
		s = fmt.Sprintf("%d", x)
	default:
		s = fmt.Sprint(x)
	}
	n := 0
	for _, r := range s {
		if r > 0x2E7F {
			n += 2
		} else {
			n++
		}
	}
	return n + 2
}
