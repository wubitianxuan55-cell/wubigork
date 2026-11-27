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
	// Specialty 项目默认专业（字典值；清单项自身专业为空时跟随它）。
	// Region 项目所在地区（字典值；与封面地址 Location 分开——地址是自由文本，
	// 地区是规范值）。PricePeriod 计价期（YYYY-MM）。三者空=待整理。
	Specialty   string `json:"specialty"`
	Region      string `json:"region"`
	PricePeriod string `json:"pricePeriod"`
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
	ID        int64  `json:"id"`
	ProjectID int64  `json:"projectId"`
	Code      string `json:"code"` // 清单编码（WP01/BJ01…；手填行自动 M 序号）
	// GCode 库内自有码（Q+专业+分部+流水，SchemaV39；见 gcode.go）。Code 是
	// 来源码（映射列），检索两码都吃。
	GCode         string  `json:"gcode"`
	Title         string  `json:"title"`
	Unit          string  `json:"unit"`
	Division      string  `json:"division"`      // 分部（A临建/B支护降水…）
	Quantity      float64 `json:"quantity"`      // 工程量
	QuantityExpr  string  `json:"quantityExpr"`  // 工程量计算式（留痕）
	QuotaCode     string  `json:"quotaCode"`     // 引用定额（空=手填行）
	Feature       string  `json:"feature"`       // 工序特征
	PriceOverride float64 `json:"priceOverride"` // 0=按定额现算；>0=手填单价优先
	// ListPrice 原表综合单价快照（导入时从工作簿落库，SchemaV34；0=无快照）。
	// 读取面现算口径不破坏——现算按资源现行价，快照只做对照显形与案例样本。
	ListPrice float64 `json:"listPrice"`
	Sort      int     `json:"sort"`
	// Specialty 清单项专业（字典值）。读取时已归一：行自身专业为空则取项目
	// 默认专业，都空=待整理。保存时传空=保持原值（重导入不冲掉已归类的专业）。
	Specialty string `json:"specialty"`
	// QuotaCodes 该清单项挂接的全部定额码（gf_bill_quota_links 聚合，1:N 组合；
	// 单值 QuotaCode 是主定额=旧锚点）。构造即非 nil（JSON null 崩前端）。
	QuotaCodes []string `json:"quotaCodes"`
	// LinkQuantities 各挂接定额的工程量（定额码 → 工程量；0=跟随清单工程量）。
	// 与 QuotaCodes 同源（引用表），详情里「消耗量 = 工程量 × 定额含量」用它。构造即非 nil。
	LinkQuantities map[string]float64 `json:"linkQuantities"`
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

// ValidateProjectScope 项目口径过规矩（规矩见 rules.go）：专业、地区必须在字典里
// （空=待整理），计价期归一成 YYYY-MM。返回归一后的计价期。建项目与改口径都先过它
// ——口径被拒时不留半截封面。
func (s *Store) ValidateProjectScope(specialty, region, pricePeriod string) (period, province string, err error) {
	if err = s.requireDB(); err != nil {
		return "", "", err
	}
	if err = s.checkDict(DictSpecialty, strings.TrimSpace(specialty), "项目专业"); err != nil {
		return "", "", err
	}
	// 地区只到省（不能下分）：门口归一成省级全称，调用方落归一值。
	if province, err = s.checkRegion(region, "项目地区"); err != nil {
		return "", "", err
	}
	period, err = checkPeriod(pricePeriod, "项目计价期")
	return period, province, err
}

// UpdateBillProjectProfile 改项目的专业/地区/计价期（空=保持原值）。三个都是规范
// 值（规矩见 rules.go）：专业在字典里，地区只到省（门口归一），计价期归一成 YYYY-MM。
func (s *Store) UpdateBillProjectProfile(id int64, specialty, region, pricePeriod string) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	period, province, err := s.ValidateProjectScope(specialty, region, pricePeriod)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE gf_projects SET
  specialty=COALESCE(NULLIF(?,''), specialty),
  region=COALESCE(NULLIF(?,''), region),
  price_period=COALESCE(NULLIF(?,''), price_period),
  updated_at=?
WHERE id=?`, specialty, province, period, time.Now().Format(time.RFC3339), id)
	return err
}

// billItemCols 清单行查询列（FindBillItem / BillItems 共用，避免两处列序漂移）。
// 专业读取时已归一：行自身为空则取项目默认专业。
const billItemCols = `
SELECT i.id, i.project_id, i.code, COALESCE(i.gcode,''), i.title, i.unit, i.division, i.quantity, i.quantity_expr, i.quota_code,
  i.feature, i.price_override, i.list_price, i.sort,
  COALESCE(NULLIF(i.specialty,''), NULLIF(p.specialty,''), '')
FROM gf_bill_items i JOIN gf_projects p ON p.id=i.project_id`

func scanBillItem(sc scanner) (*BillItem, error) {
	var it BillItem
	if err := sc.Scan(&it.ID, &it.ProjectID, &it.Code, &it.GCode, &it.Title, &it.Unit, &it.Division,
		&it.Quantity, &it.QuantityExpr, &it.QuotaCode, &it.Feature, &it.PriceOverride, &it.ListPrice, &it.Sort,
		&it.Specialty); err != nil {
		return nil, err
	}
	return &it, nil
}

// SaveBillItem 新增/更新一条清单项（同项目同编码走更新；code 空自动生成 M 序号）。
// 专业传空=保持原值（新行为空=跟随项目默认专业）——重导入不冲掉已归类的专业。
// sort 传 0=保持原序（新行追加到末尾）：造价模式重发已有行不带序号，不应把行顶到
// 列表最前面。
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
	item.Specialty = strings.TrimSpace(item.Specialty)
	if err := s.checkDict(DictSpecialty, item.Specialty, "清单项专业"); err != nil {
		return item, err
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
		_, err = s.db.Exec(`UPDATE gf_bill_items SET title=?, unit=?, division=?, quantity=?, quantity_expr=?, quota_code=?, feature=?, price_override=?, list_price=?,
  sort=CASE WHEN ?<>0 THEN ? ELSE sort END,
  specialty=COALESCE(NULLIF(?,''), specialty) WHERE id=?`,
			item.Title, item.Unit, item.Division, item.Quantity, item.QuantityExpr, item.QuotaCode, item.Feature, item.PriceOverride, item.ListPrice,
			item.Sort, item.Sort, item.Specialty, existing)
		if err != nil {
			return item, err
		}
		if item.Specialty == "" {
			// 传空=保持原值：返回值回读库里的专业，回执不至于把已归类的专业显示成空。
			_ = s.db.QueryRow(`SELECT specialty FROM gf_bill_items WHERE id=?`, existing).Scan(&item.Specialty)
		}
		// 返回值按绑定面契约构造即非 nil（nil 切片 JSON null 崩前端）。
		if item.QuotaCodes == nil {
			item.QuotaCodes = []string{}
		}
		return item, nil
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
	// 新清单行授自有码（Q + 专业2 + 分部2 + 流水3，见 gcode.go）。
	gcode, err := s.newBillGCode(s.db, &item)
	if err != nil {
		return item, err
	}
	item.GCode = gcode
	res, err := s.db.Exec(`INSERT INTO gf_bill_items (project_id, code, gcode, title, unit, division, quantity, quantity_expr, quota_code, feature, price_override, list_price, sort, specialty) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ProjectID, item.Code, item.GCode, item.Title, item.Unit, item.Division, item.Quantity, item.QuantityExpr, item.QuotaCode, item.Feature, item.PriceOverride, item.ListPrice, item.Sort, item.Specialty)
	if err != nil {
		return item, err
	}
	item.ID, _ = res.LastInsertId()
	if item.QuotaCodes == nil {
		item.QuotaCodes = []string{}
	}
	return item, nil
}

// FindBillItem 按项目与编码找清单项；不存在返回 nil（编码为空恒返回 nil——
// 空编码走 SaveBillItem 自动编号，本来就是新行）。
func (s *Store) FindBillItem(projectID int64, code string) (*BillItem, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	if projectID <= 0 || code == "" {
		return nil, nil
	}
	it, err := scanBillItem(s.db.QueryRow(billItemCols+` WHERE i.project_id=? AND i.code=?`, projectID, code))
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	// 与 BillItems 同口径聚合挂接码（引用表为准，单值列兜底），构造即非 nil。
	it.attachLinks(s.quotaLinkRowsByItems([]int64{it.ID}, map[int64]string{it.ID: it.QuotaCode})[it.ID])
	return it, nil
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

// DeleteBillItem 删除一条清单项，连同其在 gf_bill_quota_links 的定额挂接
// （1:N 引用随清单项消亡；V30 后不清理会留孤儿引用，该清单 ID 复用或
// 导出聚合时复活成幽灵定额）。
func (s *Store) DeleteBillItem(id int64) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM gf_bill_quota_links WHERE bill_item_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM gf_bill_items WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// BillItems 项目的清单（按 sort, id 稳定序）。每行带挂接定额码与各定额的工程量
// （构造即非 nil）。
func (s *Store) BillItems(projectID int64) ([]BillItem, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(billItemCols+` WHERE i.project_id=? ORDER BY i.sort, i.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []BillItem{}
	var ids []int64
	fallback := map[int64]string{}
	for rows.Next() {
		it, err := scanBillItem(rows)
		if err != nil {
			continue
		}
		ids = append(ids, it.ID)
		fallback[it.ID] = it.QuotaCode
		out = append(out, *it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 1:N 引用聚合（link 表全量事实；单值列兜底）。
	links := s.quotaLinkRowsByItems(ids, fallback)
	for i := range out {
		out[i].attachLinks(links[out[i].ID])
	}
	return out, nil
}

// attachLinks 把挂接明细装到清单行上：定额码集合 + 各定额工程量。
func (it *BillItem) attachLinks(links []quotaLinkRow) {
	codes := make([]string, 0, len(links))
	qtys := make(map[string]float64, len(links))
	for _, l := range links {
		if l.Code == "" {
			continue
		}
		codes = append(codes, l.Code)
		qtys[l.Code] = l.Quantity
	}
	it.QuotaCodes = codes
	it.LinkQuantities = qtys
}

// BillProjects 项目列表（含费率与清单项数，最新在前）。
func (s *Store) BillProjects() ([]BillProject, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT p.id, p.name, p.file_name, p.source_path, p.location, p.duration, p.pricing,
  p.specialty, p.region, p.price_period,
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
			&p.Specialty, &p.Region, &p.PricePeriod,
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

	// ① 清单项（1:N 定额挂接引用随清单项一并清理，防孤儿）。
	if _, err := tx.Exec(`DELETE FROM gf_bill_quota_links WHERE bill_item_id IN (SELECT id FROM gf_bill_items WHERE project_id=?)`, id); err != nil {
		return 0, err
	}
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
//
// 地区（规矩：工料机按地区分行）：给了地区就先在该地区的行里找，该地区没有
// 候选时退到尚未整理地区的行；别的地区的同名行不参与。不给地区=全库找，
// 多地区同名即歧义。
func (s *Store) MatchResourceByTitleSpec(title, spec, unit, region string) (*Resource, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	nt, ns, nu := NormalizeText(title), NormalizeText(spec), NormalizeText(unit)
	region = strings.TrimSpace(region)
	if nt == "" {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT id, code, kind, title, spec, unit, region FROM gf_resources WHERE status != '停用'`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var hit, unlabeled *Resource
	exactN, unlabeledN := 0, 0
	for rows.Next() {
		var r Resource
		if err := rows.Scan(&r.ID, &r.Code, &r.Kind, &r.Title, &r.Spec, &r.Unit, &r.Region); err != nil {
			continue
		}
		rt, rs := NormalizeText(r.Title), NormalizeText(r.Spec)
		if rt != nt || (ns != "" && rs != ns) {
			continue
		}
		if nu != "" && NormalizeText(r.Unit) != nu {
			continue
		}
		if region != "" && r.Region != "" && r.Region != region {
			continue // 别的地区的行不参与
		}
		if region == "" || r.Region == region {
			exactN++
			hit = &r
		} else {
			unlabeledN++
			unlabeled = &r
		}
	}
	if exactN > 1 || (exactN == 0 && unlabeledN > 1) {
		return nil, nil // 歧义：多条同身份
	}
	if exactN == 1 {
		return hit, rows.Err()
	}
	return unlabeled, rows.Err()
}
