package app

// gaea_cost_library_export.go — 造价库列表导出（2026-11-25 造价数据库推倒重建：
// 板块只留浏览/检索/删除/导出，导出=把清单按当前筛选落成 xlsx）。
//
// 四类：分部分项清单（含原表单价与现算单价）/ 消耗定额（主表+含量明细附表）/
// 工料机 / 资料条目。筛选与板块列表同口径（projectID / keyword / extra），
// 现算单价走库内 ComposeQuotas 唯一权威链。目标文件已存在时拒绝覆盖。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

// costExportKinds 可导出的四类清单（与板块导出菜单一一对应；值为表名）。
var costExportKinds = map[string]string{
	"bill":     "分部分项清单",
	"quota":    "消耗定额",
	"resource": "工料机",
	"entry":    "资料条目",
}

// GaeaCostLibraryExport 把造价库的一类清单导出为 xlsx，返回写出的路径。
//
// kind：bill=分部分项清单｜quota=消耗定额（主表+含量明细附表）｜resource=工料机｜
// entry=资料条目。筛选与板块列表同口径：projectID（仅清单，0=全部项目）、
// keyword（名称/编码/特征/定额码包含，忽略大小写）、extra（清单=分部，定额=专业，
// 工料机=类别；空=不过滤）。outPath 是目标 .xlsx 的完整路径；已存在则拒绝覆盖。
func (a *App) GaeaCostLibraryExport(kind string, projectID int64, keyword string, extra string, outPath string) (string, error) {
	sheet, ok := costExportKinds[kind]
	if !ok {
		return "", fmt.Errorf("不支持的导出类型 %q（bill/quota/resource/entry）", kind)
	}
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
	kw := strings.ToLower(strings.TrimSpace(keyword))
	extra = strings.TrimSpace(extra)

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetName(f.GetSheetName(0), sheet); err != nil {
		return "", err
	}
	var (
		headers []string
		rows    [][]any
		err     error
	)
	switch kind {
	case "bill":
		headers, rows, err = a.costExportBill(projectID, kw, extra)
	case "quota":
		headers, rows, err = a.costExportQuota(kw, extra, f)
	case "resource":
		headers, rows, err = a.costExportResource(kw, extra)
	default:
		headers, rows, err = a.costExportEntry(kw)
	}
	if err != nil {
		return "", err
	}
	if err := costExportSheet(f, sheet, headers, rows); err != nil {
		return "", fmt.Errorf("写出「%s」失败: %w", sheet, err)
	}
	if dir := filepath.Dir(outPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("创建导出目录失败: %w", err)
		}
	}
	if err := f.SaveAs(outPath); err != nil {
		return "", fmt.Errorf("保存工作簿失败: %w", err)
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

// costExportPrice 大于 0 才写数值，否则留空（0 价在造价数据里是「未录入」，
// 导成一列 0 会被下游当成真价）。
func costExportPrice(v float64) any {
	if v > 0 {
		return v
	}
	return ""
}

// costExportBill 分部分项清单：项目 → 清单行，现算单价=挂接定额的综合单价之和。
func (a *App) costExportBill(projectID int64, kw, division string) ([]string, [][]any, error) {
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
		if projectID > 0 && p.ID != projectID {
			continue
		}
		items, err := s.BillItems(p.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			if division != "" && it.Division != division {
				continue
			}
			if !costExportMatch(kw, it.Title, it.Code, it.Feature, it.QuotaCode, strings.Join(it.QuotaCodes, " ")) {
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
	headers := []string{"项目", "编码", "名称", "项目特征", "单位", "分部", "工程量", "工程量计算式",
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
			r.project.Name, r.item.Code, r.item.Title, r.item.Feature, r.item.Unit, r.item.Division,
			r.item.Quantity, r.item.QuantityExpr, strings.Join(r.item.QuotaCodes, "+"),
			costExportPrice(r.item.PriceOverride), costExportPrice(r.item.ListPrice), costExportPrice(unit),
		})
	}
	return headers, rows, nil
}

// costExportQuota 消耗定额：主表 + 含量明细附表（附表单独成页）。
func (a *App) costExportQuota(kw, specialty string, f *excelize.File) ([]string, [][]any, error) {
	s := a.hubWorkcostStore()
	if !s.Available() {
		return nil, nil, fmt.Errorf("工料机造价库不可用")
	}
	quotas, err := s.ListQuotas(specialty, "")
	if err != nil {
		return nil, nil, err
	}
	var picked []workcost.Quota
	for _, q := range quotas {
		if !costExportMatch(kw, q.Title, q.Code, q.OrigCode, q.Chapter) {
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
	headers := []string{"定额编码", "原码", "名称", "单位", "专业", "章节", "来源", "含量行数", "被清单引用", "现算综合单价（元·人材机）"}
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
		rows = append(rows, []any{q.Code, q.OrigCode, q.Title, q.Unit, q.Specialty, q.Chapter, q.Source,
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
			detail = append(detail, []any{q.Code, q.Title, it.Kind, it.ResourceCode, it.Title, it.Spec, it.Unit,
				costExportPrice(it.Quantity), loss, costExportPrice(price), costExportPrice(amount)})
		}
	}
	const detailSheet = "含量明细"
	if _, err := f.NewSheet(detailSheet); err != nil {
		return nil, nil, err
	}
	detailHeaders := []string{"定额编码", "定额名称", "行类", "资源编码", "资源名称", "规格", "单位", "含量", "损耗率", "取用价（元）", "金额（元）"}
	if err := costExportSheet(f, detailSheet, detailHeaders, detail); err != nil {
		return nil, nil, fmt.Errorf("写出「%s」失败: %w", detailSheet, err)
	}
	return headers, rows, nil
}

// costExportResource 工料机：主数据 + 被多少条定额含量行引用。
func (a *App) costExportResource(kw, kind string) ([]string, [][]any, error) {
	s := a.hubWorkcostStore()
	if !s.Available() {
		return nil, nil, fmt.Errorf("工料机造价库不可用")
	}
	list, err := s.ListResources(kind, "")
	if err != nil {
		return nil, nil, err
	}
	counts, _ := s.ResourceReferenceCounts()
	headers := []string{"编码", "类别", "名称", "规格", "单位", "现行价（元）", "基准价（元）", "取用价（元）", "损耗率",
		"分类", "来源", "供应商", "地区", "价格期数", "价格口径", "有效期至", "状态", "备注", "被定额引用"}
	rows := make([][]any, 0, len(list))
	for _, r := range list {
		if !costExportMatch(kw, r.Title, r.Code, r.Spec, r.CategoryPath) {
			continue
		}
		loss := ""
		if r.LossRate > 0 {
			loss = fmt.Sprintf("%.1f%%", r.LossRate*100)
		}
		rows = append(rows, []any{r.Code, r.Kind, r.Title, r.Spec, r.Unit,
			costExportPrice(r.CurrentPrice), costExportPrice(r.BasePrice), costExportPrice(r.EffectivePrice()), loss,
			r.CategoryPath, r.Source, r.Supplier, r.Region, r.PriceDate, r.PriceType, r.ValidUntil, r.Status, r.Note,
			counts[r.Code]})
	}
	return headers, rows, nil
}

// costExportEntry 资料条目全量（它没有界面了，这里是唯一的导出口）。
func (a *App) costExportEntry(kw string) ([]string, [][]any, error) {
	store := a.hubCostStore()
	if !store.Available() {
		return nil, nil, fmt.Errorf("资料条目库不可用")
	}
	list, err := store.List()
	if err != nil {
		return nil, nil, err
	}
	headers := []string{"名称", "标题", "编码", "分类路径", "单位", "单价（元）", "人工费", "材料费", "机械费",
		"规格", "来源", "地区", "价格期数", "价格口径", "有效期至", "状态", "更新时间"}
	rows := make([][]any, 0, len(list))
	for _, e := range list {
		if !costExportMatch(kw, e.Title, e.Name, e.Code, e.Spec) {
			continue
		}
		updated := ""
		if !e.UpdatedAt.IsZero() {
			updated = e.UpdatedAt.Local().Format("2006-01-02 15:04")
		}
		rows = append(rows, []any{e.Name, e.Title, e.Code, e.CategoryPath, e.Unit,
			costExportPrice(e.Price), costExportPrice(e.LaborFee), costExportPrice(e.MaterialFee), costExportPrice(e.MachineFee),
			e.Spec, e.Source, e.Region, e.PriceDate, e.PriceType, e.ValidUntil, e.Status, updated})
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
