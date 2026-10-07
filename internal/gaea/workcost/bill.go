// bill.go — 工料法第④层：分部分项工程量清单（gf_projects + gf_bill_items）。
//
// 用户定调（2026-10-07）：「清单、定额、工料机是什么关系？我让你录入的
// 清单呢？」——工料机=资源价格，定额=每单位子目消耗多少工料机，**清单=
// 工程实体的分部分项列表（编码/名称/单位/工程量），每条套定额，
// 工程量 × 综合单价 = 合价**。清单是导入模版时最该落库的一层，此前缺失。
package workcost

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// BillProject 一次导入=一个项目（五表封面 + 取费区费率——录入数据）。
// 同名走更新（幂等导入）。费率只是数据：加总计算不在库里发生（导出五表
// 活公式工作簿由 Excel 算），这里只负责存取。
type BillProject struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FileName string `json:"fileName"`
	Source   string `json:"source"`
	Location string `json:"location"`
	Duration string `json:"duration"`
	Pricing  string `json:"pricing"`
	// 费率（录入数据）：企管/规费/利润/增值税 + 利润基数含规费开关 + 控制价。
	// 零值语义与 workcost.RateSet 一致：任一费率 ≤0 = 显式不计取该段。
	ManagementRate           float64 `json:"managementRate"`
	RegulatoryRate           float64 `json:"regulatoryRate"`
	ProfitRate               float64 `json:"profitRate"`
	TaxRate                  float64 `json:"taxRate"`
	ProfitIncludesRegulatory bool    `json:"profitIncludesRegulatory"`
	ControlPrice             float64 `json:"controlPrice"`
	ItemCount                int     `json:"itemCount"` // 清单项数（读时聚合）
	CreatedAt                string  `json:"createdAt"`
	UpdatedAt                string  `json:"updatedAt"`
}

// BillItem 一条清单项：工程量 × 引用定额（或手填单价）→ 合价。
//
// QuotaCode 非空 = 套定额，综合单价按定额现算（量价分离，资源调价自动传导）；
// PriceOverride > 0 = 手填单价优先（外委包干类，或对定额价的项目级调整）。
// 合价刻意不入列：读时用当前价重算。
type BillItem struct {
	ID            int64   `json:"id"`
	ProjectID     int64   `json:"projectId"`
	Code          string  `json:"code"` // 清单编码（WP01/BJ01…；手填行自动 M 序号）
	Title         string  `json:"title"`
	Unit          string  `json:"unit"`
	Division      string  `json:"division"`      // 分部（A临建/B支护降水…）
	Quantity      float64 `json:"quantity"`      // 工程量
	QuantityExpr  string  `json:"quantityExpr"`  // 工程量计算式（留痕）
	QuotaCode     string  `json:"quotaCode"`     // 引用定额（空=手填行）
	Feature       string  `json:"feature"`       // 工序特征
	PriceOverride float64 `json:"priceOverride"` // 0=按定额现算；>0=手填单价优先
	Sort          int     `json:"sort"`
}

// UpsertBillProject 落项目封面与费率（同名更新），返回项目 ID。
// rates 为 nil 时费率不覆盖（保留已存值）——手工建项目只给封面、导入给全量。
func (s *Store) UpsertBillProject(name, fileName, source, location, duration, pricing string, rates *RateSet, profitIncludesRegulatory bool, controlPrice float64) (int64, error) {
	if err := s.requireDB(); err != nil {
		return 0, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("项目需要名称")
	}
	now := time.Now().Format(time.RFC3339)
	var id int64
	err := s.db.QueryRow(`SELECT id FROM gf_projects WHERE name=?`, name).Scan(&id)
	if err == nil {
		if rates != nil {
			_, err = s.db.Exec(`UPDATE gf_projects SET file_name=?, source_path=?, location=?, duration=?, pricing=?,
  management_rate=?, regulatory_rate=?, profit_rate=?, tax_rate=?, profit_includes_regulatory=?, control_price=?, updated_at=? WHERE id=?`,
				fileName, source, location, duration, pricing,
				rates.ManagementRate, rates.RegulatoryRate, rates.ProfitRate, rates.TaxRate,
				boolToInt(profitIncludesRegulatory), controlPrice, now, id)
		} else {
			_, err = s.db.Exec(`UPDATE gf_projects SET file_name=?, source_path=?, location=?, duration=?, pricing=?, updated_at=? WHERE id=?`,
				fileName, source, location, duration, pricing, now, id)
		}
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	if !strings.Contains(err.Error(), "no rows") {
		return 0, err
	}
	flag := boolToInt(profitIncludesRegulatory)
	if rates == nil {
		rates = &RateSet{ManagementRate: 0.1, RegulatoryRate: 0.02, ProfitRate: 0.07, TaxRate: 0.09}
	}
	res, err := s.db.Exec(`INSERT INTO gf_projects (name, file_name, source_path, location, duration, pricing,
  management_rate, regulatory_rate, profit_rate, tax_rate, profit_includes_regulatory, control_price, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		name, fileName, source, location, duration, pricing,
		rates.ManagementRate, rates.RegulatoryRate, rates.ProfitRate, rates.TaxRate, flag, controlPrice, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// boolToInt SQLite 布尔落库方言（Go database/sql 无 bool 驱动方言，统一 0/1）。
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// SaveBillItem 新增/更新一条清单项（同项目同编码走更新；code 空自动生成 M 序号）。
func (s *Store) SaveBillItem(item BillItem) (BillItem, error) {
	if err := s.requireDB(); err != nil {
		return item, err
	}
	item.Title = strings.TrimSpace(item.Title)
	item.Code = strings.TrimSpace(item.Code)
	if item.Title == "" {
		return item, fmt.Errorf("清单项需要名称")
	}
	if item.ProjectID <= 0 {
		return item, fmt.Errorf("清单项需要归属项目")
	}
	if item.Code == "" {
		n, err := s.nextBillCode(item.ProjectID)
		if err != nil {
			return item, err
		}
		item.Code = fmt.Sprintf("M%03d", n)
	}
	var existing int64
	err := s.db.QueryRow(`SELECT id FROM gf_bill_items WHERE project_id=? AND code=?`, item.ProjectID, item.Code).Scan(&existing)
	if err == nil {
		item.ID = existing
		_, err = s.db.Exec(`UPDATE gf_bill_items SET title=?, unit=?, division=?, quantity=?, quantity_expr=?, quota_code=?, feature=?, price_override=?, sort=? WHERE id=?`,
			item.Title, item.Unit, item.Division, item.Quantity, item.QuantityExpr, item.QuotaCode, item.Feature, item.PriceOverride, item.Sort, existing)
		return item, err
	}
	if !strings.Contains(err.Error(), "no rows") {
		return item, err
	}
	if item.Sort == 0 {
		// 追加到末尾：当前最大 sort + 1。
		var maxSort int
		if err := s.db.QueryRow(`SELECT COALESCE(MAX(sort),0) FROM gf_bill_items WHERE project_id=?`, item.ProjectID).Scan(&maxSort); err == nil {
			item.Sort = maxSort + 1
		}
	}
	res, err := s.db.Exec(`INSERT INTO gf_bill_items (project_id, code, title, unit, division, quantity, quantity_expr, quota_code, feature, price_override, sort) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		item.ProjectID, item.Code, item.Title, item.Unit, item.Division, item.Quantity, item.QuantityExpr, item.QuotaCode, item.Feature, item.PriceOverride, item.Sort)
	if err != nil {
		return item, err
	}
	item.ID, _ = res.LastInsertId()
	return item, nil
}

// nextBillCode 手填行自动编号（项目内 M 序号，跳过已用号）。
func (s *Store) nextBillCode(projectID int64) (int, error) {
	rows, err := s.db.Query(`SELECT code FROM gf_bill_items WHERE project_id=? AND code LIKE 'M%'`, projectID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	maxSeq := 0
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		digits := strings.TrimPrefix(code, "M")
		if len(digits) != 3 {
			continue
		}
		n := 0
		for _, ch := range digits {
			if ch < '0' || ch > '9' {
				n = -1
				break
			}
			n = n*10 + int(ch-'0')
		}
		if n > maxSeq {
			maxSeq = n
		}
	}
	return maxSeq + 1, rows.Err()
}

// DeleteBillItem 删除一条清单项。
func (s *Store) DeleteBillItem(id int64) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM gf_bill_items WHERE id=?`, id)
	return err
}

// BillItems 项目的清单（按 sort, id 稳定序）。
func (s *Store) BillItems(projectID int64) ([]BillItem, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id, project_id, code, title, unit, division, quantity, quantity_expr, quota_code, feature, price_override, sort
FROM gf_bill_items WHERE project_id=? ORDER BY sort, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []BillItem
	for rows.Next() {
		var it BillItem
		if err := rows.Scan(&it.ID, &it.ProjectID, &it.Code, &it.Title, &it.Unit, &it.Division,
			&it.Quantity, &it.QuantityExpr, &it.QuotaCode, &it.Feature, &it.PriceOverride, &it.Sort); err != nil {
			continue
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// BillProjects 项目列表（含费率与清单项数，最新在前）。
func (s *Store) BillProjects() ([]BillProject, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT p.id, p.name, p.file_name, p.source_path, p.location, p.duration, p.pricing,
  p.management_rate, p.regulatory_rate, p.profit_rate, p.tax_rate, p.profit_includes_regulatory, p.control_price,
  p.created_at, p.updated_at, (SELECT COUNT(*) FROM gf_bill_items b WHERE b.project_id=p.id) AS item_count
FROM gf_projects p ORDER BY p.updated_at DESC, p.id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []BillProject
	for rows.Next() {
		var p BillProject
		var flag int
		if err := rows.Scan(&p.ID, &p.Name, &p.FileName, &p.Source, &p.Location, &p.Duration, &p.Pricing,
			&p.ManagementRate, &p.RegulatoryRate, &p.ProfitRate, &p.TaxRate, &flag, &p.ControlPrice,
			&p.CreatedAt, &p.UpdatedAt, &p.ItemCount); err != nil {
			continue
		}
		p.ProfitIncludesRegulatory = flag != 0
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateBillProjectRates 只改项目费率（录入面：费率是数据，随时可调）。
func (s *Store) UpdateBillProjectRates(id int64, rates RateSet, profitIncludesRegulatory bool, controlPrice float64) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE gf_projects SET management_rate=?, regulatory_rate=?, profit_rate=?, tax_rate=?,
  profit_includes_regulatory=?, control_price=?, updated_at=? WHERE id=?`,
		rates.ManagementRate, rates.RegulatoryRate, rates.ProfitRate, rates.TaxRate,
		boolToInt(profitIncludesRegulatory), controlPrice, time.Now().Format(time.RFC3339), id)
	return err
}

// DeleteBillProject 整体删除导入的项目：项目 + 其全部分部分项清单 + **独占
// 定额**（source 指向本项目且删除清单后不再被任何清单项引用）。资源不删——
// 工料机是全局主数据，可能被他项目定额引用。
func (s *Store) DeleteBillProject(id int64) (int, error) {
	if err := s.requireDB(); err != nil {
		return 0, err
	}
	var fileName string
	if err := s.db.QueryRow(`SELECT file_name FROM gf_projects WHERE id=?`, id).Scan(&fileName); err != nil {
		return 0, fmt.Errorf("项目 %d 不存在", id)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	// ① 清单项。
	if _, err := tx.Exec(`DELETE FROM gf_bill_items WHERE project_id=?`, id); err != nil {
		return 0, err
	}
	// ② 独占定额：本项目导入、且删完清单后已无任何清单项引用（他项目引用
	// 的共享定额——编码统一复用的产物——保留）。
	rows, err := tx.Query(`SELECT code FROM gf_quotas WHERE source=? AND code NOT IN
  (SELECT quota_code FROM gf_bill_items WHERE COALESCE(quota_code,'') != '')`,
		fmt.Sprintf("项目导入：%s", fileName))
	if err != nil {
		return 0, err
	}
	var orphans []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err == nil {
			orphans = append(orphans, code)
		}
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, code := range orphans {
		if _, err := tx.Exec(`DELETE FROM gf_quota_items WHERE quota_code=?`, code); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(`DELETE FROM gf_quotas WHERE code=?`, code); err != nil {
			return 0, err
		}
	}
	// ③ 项目封面。
	if _, err := tx.Exec(`DELETE FROM gf_projects WHERE id=?`, id); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(orphans), nil
}

// NormalizeText 价格匹配用的文本归一：去空白 + 转小写（中文不变形）。
func NormalizeText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '　':
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// MatchResourceByTitleSpec 按（归一化标题+规格+单位）匹配工料机资源。
// 信息价发布联动资源现行价的匹配入口。多条命中视为歧义返回 nil（宁缺勿误
// ——把价推进到错的资源比不推进更糟）。
func (s *Store) MatchResourceByTitleSpec(title, spec, unit string) (*Resource, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	nt, ns, nu := NormalizeText(title), NormalizeText(spec), NormalizeText(unit)
	if nt == "" {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT id, code, kind, title, spec, unit FROM gf_resources WHERE status != '停用'`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var hit *Resource
	for rows.Next() {
		var r Resource
		if err := rows.Scan(&r.ID, &r.Code, &r.Kind, &r.Title, &r.Spec, &r.Unit); err != nil {
			continue
		}
		rt, rs := NormalizeText(r.Title), NormalizeText(r.Spec)
		if rt != nt || (ns != "" && rs != ns) {
			continue
		}
		if nu != "" && NormalizeText(r.Unit) != nu {
			continue
		}
		if hit != nil {
			return nil, nil // 歧义：多条同身份
		}
		hit = &r
	}
	return hit, rows.Err()
}
