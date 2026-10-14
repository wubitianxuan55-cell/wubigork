package workcost

// ── 资源价批量维护（三期 3.4，master plan 原案）──────────────
//
// Excel 导入资源调价（名称/单位/新价/期数），批量推进现行价+写调价历史——
// 1486 条规模下调价不再逐条。
//
// 匹配纪律（造价线「宁缺勿误」同款）：名称+单位（+规格，若提供）三要素精确
// 匹配；零命中或多行歧义一律跳过进报告（人可读原因），绝不模糊猜。应用=
// 整批一个事务（全部可匹配行一次落库，中途失败整体回滚=干净重试）。
//
// 双段式（与既有 GaeaCostImportPreview/Apply 同形态）：Preview 先看匹配结果，
// 作者确认后 Apply 才落库。

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// BulkPriceRow Excel 一行调价（原案列：名称/单位/新价/期数；规格/地区/备注可选）。
type BulkPriceRow struct {
	Title  string  `json:"title"`
	Spec   string  `json:"spec,omitempty"`
	Unit   string  `json:"unit,omitempty"`
	Price  float64 `json:"price"`
	Period string  `json:"period,omitempty"`
	Region string  `json:"region,omitempty"`
	Note   string  `json:"note,omitempty"`
	// Row 序号（1 起，Excel 数据行序；报告回指用）。
	Row int `json:"row"`
}

// BulkPriceSkip 一条跳过（人可读原因）。
type BulkPriceSkip struct {
	Row    int    `json:"row"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

// BulkPriceMatch 预览的一条匹配（资源现状速览）。
type BulkPriceMatch struct {
	Row          int     `json:"row"`
	ResourceID   int64   `json:"resourceId"`
	Code         string  `json:"code"`
	Title        string  `json:"title"`
	Spec         string  `json:"spec,omitempty"`
	Unit         string  `json:"unit,omitempty"`
	OldPrice     float64 `json:"oldPrice"`
	NewPrice     float64 `json:"newPrice"`
	Period       string  `json:"period,omitempty"`
	Region       string  `json:"region,omitempty"`
	Note         string  `json:"note,omitempty"`
	ResourceKind string  `json:"resourceKind,omitempty"`
}

// BulkPricePreview 预览报告：解析行 + 匹配/跳过分区。
type BulkPricePreview struct {
	Total    int              `json:"total"` // 解析到的数据行数
	Matched  []BulkPriceMatch `json:"matched"`
	Skipped  []BulkPriceSkip  `json:"skipped"`
	RowError []BulkPriceSkip  `json:"rowErrors"` // 解析层错误（列缺失/价格非法）
}

// BulkPriceResult 应用回执。
type BulkPriceResult struct {
	Updated int              `json:"updated"`
	Skipped []BulkPriceSkip  `json:"skipped"`
	Total   int              `json:"total"`
	Applied []BulkPriceMatch `json:"applied"`
}

// ParseBulkPriceXlsx 解析批量调价 Excel：首个工作表，表头行（含「名称」列）
// 起识别列位，其后逐行为数据。列头同义映射：名称=名称/资源名称/材料名称；
// 单位=单位/计量单位；新价=新价/现行价/调整价/价格；期数=期数/信息价期数/价格期数。
func ParseBulkPriceXlsx(raw []byte) ([]BulkPriceRow, error) {
	f, err := excelize.OpenReader(bytes.NewReader(raw), excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return nil, fmt.Errorf("Excel 打开失败：%w", err)
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel 无工作表")
	}
	sheet := sheets[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败：%w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel 至少需要表头行+一行数据")
	}

	// 列位识别（表头行）：找不到「名称」列即报错（防错表）。
	col := map[string]int{}
	for i, cell := range rows[0] {
		col[normalizeHeader(cell)] = i
	}
	iTitle := pickCol(col, "名称", "资源名称", "材料名称", "资源")
	if iTitle < 0 {
		return nil, fmt.Errorf("表头缺少「名称」列（识别到表头：%s）", strings.Join(rows[0], " / "))
	}
	iSpec := pickCol(col, "规格", "规格型号")
	iUnit := pickCol(col, "单位", "计量单位")
	iPrice := pickCol(col, "新价", "现行价", "调整价", "价格")
	iPeriod := pickCol(col, "期数", "信息价期数", "价格期数")
	iRegion := pickCol(col, "地区", "区域")
	iNote := pickCol(col, "备注", "说明")
	if iPrice < 0 {
		return nil, fmt.Errorf("表头缺少「新价」列（识别到表头：%s）", strings.Join(rows[0], " / "))
	}

	out := make([]BulkPriceRow, 0, len(rows)-1)
	for ri, row := range rows[1:] {
		num := ri + 2 // Excel 行号（1 起+表头行）
		title := cellStringAt(row, iTitle)
		if strings.TrimSpace(title) == "" {
			continue // 空行跳过（尾部残留常见）
		}
		r := BulkPriceRow{
			Title: strings.TrimSpace(title),
			Row:   num,
		}
		if iSpec >= 0 && iSpec < len(row) {
			r.Spec = strings.TrimSpace(row[iSpec])
		}
		if iUnit >= 0 && iUnit < len(row) {
			r.Unit = strings.TrimSpace(row[iUnit])
		}
		if iPeriod >= 0 && iPeriod < len(row) {
			r.Period = strings.TrimSpace(row[iPeriod])
		}
		if iRegion >= 0 && iRegion < len(row) {
			r.Region = strings.TrimSpace(row[iRegion])
		}
		if iNote >= 0 && iNote < len(row) {
			r.Note = strings.TrimSpace(row[iNote])
		}
		if iPrice < len(row) {
			r.Price = parsePrice(row[iPrice])
		}
		out = append(out, r)
	}
	return out, nil
}

// MatchBulkPrices 逐行匹配资源（纯读，Preview 与 Apply 共用的匹配真相）。
// 匹配键=名称精确 + 单位精确（若行提供）+ 规格精确（若行提供）；名称命中多行
// 时按单位/规格收窄，仍歧义=跳过。
func (s *Store) MatchBulkPrices(rows []BulkPriceRow) (matches []BulkPriceMatch, skips []BulkPriceSkip, err error) {
	if err = s.requireDB(); err != nil {
		return nil, nil, err
	}
	for _, r := range rows {
		if strings.TrimSpace(r.Title) == "" {
			skips = append(skips, BulkPriceSkip{Row: r.Row, Title: r.Title, Reason: "名称为空"})
			continue
		}
		if r.Price <= 0 {
			skips = append(skips, BulkPriceSkip{Row: r.Row, Title: r.Title, Reason: fmt.Sprintf("价格非法（%v），调价须为正数", r.Price)})
			continue
		}
		cands, err := s.listResourcesByTitle(r.Title)
		if err != nil {
			return nil, nil, err
		}
		// 收窄：单位/规格精确匹配（行提供了才参与过滤）。
		var filtered []Resource
		for _, c := range cands {
			if r.Unit != "" && c.Unit != r.Unit {
				continue
			}
			if r.Spec != "" && c.Spec != r.Spec {
				continue
			}
			filtered = append(filtered, c)
		}
		if len(filtered) == 0 {
			reason := "未找到同名资源"
			if len(cands) > 0 {
				reason = fmt.Sprintf("同名 %d 条但单位/规格不匹配", len(cands))
			}
			skips = append(skips, BulkPriceSkip{Row: r.Row, Title: r.Title, Reason: reason})
			continue
		}
		if len(filtered) > 1 {
			skips = append(skips, BulkPriceSkip{Row: r.Row, Title: r.Title,
				Reason: fmt.Sprintf("匹配到 %d 条同身份资源（歧义），请先做资源去重", len(filtered))})
			continue
		}
		c := filtered[0]
		matches = append(matches, BulkPriceMatch{
			Row: r.Row, ResourceID: c.ID, Code: c.Code, Title: c.Title, Spec: c.Spec,
			Unit: c.Unit, OldPrice: c.CurrentPrice, NewPrice: r.Price,
			Period: r.Period, Region: r.Region, Note: r.Note, ResourceKind: c.Kind,
		})
	}
	return matches, skips, nil
}

// ApplyBulkPrices 批量调价应用：Preview 的匹配行整批一个事务落库（写调价
// 历史 + 推现行价，与单条 UpdateResourcePrice 同语义）。跳过行不进事务，
// 返回报告。rows 应来自 Preview（作者确认过的同一批）。
func (s *Store) ApplyBulkPrices(rows []BulkPriceRow) (*BulkPriceResult, error) {
	matches, skips, err := s.MatchBulkPrices(rows)
	if err != nil {
		return nil, err
	}
	res := &BulkPriceResult{Total: len(rows), Skipped: skips, Applied: []BulkPriceMatch{}}
	if len(matches) == 0 {
		return res, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	now := nowText()
	for _, m := range matches {
		if _, err := tx.Exec(`
INSERT INTO gf_resource_prices(resource_id, price, period, region, price_type, source, fetched_at, note)
VALUES(?,?,?,?,?,?,?,?)`,
			m.ResourceID, m.NewPrice, m.Period, m.Region, "bulk_import", "批量调价导入", now, m.Note); err != nil {
			return nil, fmt.Errorf("写入调价历史（%s）：%w", m.Title, err)
		}
		if _, err := tx.Exec(
			`UPDATE gf_resources SET current_price=?, updated_at=? WHERE id=?`,
			m.NewPrice, now, m.ResourceID); err != nil {
			return nil, fmt.Errorf("推进现行价（%s）：%w", m.Title, err)
		}
		res.Applied = append(res.Applied, m)
		res.Updated++
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// listResourcesByTitle 名称精确匹配的资源（大小写/首尾空白归一后比对）。
func (s *Store) listResourcesByTitle(title string) ([]Resource, error) {
	rows, err := s.db.Query(
		`SELECT id, code, kind, title, spec, unit, base_price, current_price FROM gf_resources WHERE title=?`, strings.TrimSpace(title))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Resource
	for rows.Next() {
		var r Resource
		if err := rows.Scan(&r.ID, &r.Code, &r.Kind, &r.Title, &r.Spec, &r.Unit, &r.BasePrice, &r.CurrentPrice); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// normalizeHeader 表头归一：去空白+小写。
func normalizeHeader(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), ""))
}

// pickCol 按同义名序列找列位（归一后精确相等）。
func pickCol(col map[string]int, names ...string) int {
	for _, n := range names {
		if i, ok := col[normalizeHeader(n)]; ok {
			return i
		}
	}
	return -1
}

// cellStringAt 越界安全的字符串取格。
func cellStringAt(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

// parsePrice 价格解析（容忍千分位逗号与「元」后缀；解析失败返回 0=行级错误）。
func parsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSuffix(s, "元")
	s = strings.TrimSpace(s)
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
		return 0
	}
	return v
}
