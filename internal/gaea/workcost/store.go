// store.go —— 工料法三层骨架的存储访问（gf_resources / gf_resource_prices /
// gf_quotas / gf_quota_items）。
//
// 与 cost / costproject 同模式：存于 Hephaestus.db，db 为 nil 时降级为不可用
// store（各方法返回空/错误，不 panic）。表由 db.SchemaV25/V26 建立，本包不做
// 幂等自建（正式迁移已收编，避免双路径 DDL 漂移）。
package workcost

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/strutil"
)

// Store 工料法存储（工料机资源库 + 消耗定额库共用一个句柄，语义上仍是两层）。
type Store struct {
	db *sql.DB
}

// Open 打开工料法存储；gdb 为 nil 时返回不可用 store。
func Open(gdb *sql.DB) *Store {
	s := &Store{db: gdb}
	// 造价自有编码存量补授（幂等；无 gcode 列的旧夹具静默跳过）。失败只记日志
	// 不阻断开库——新行授码在各自写路径上自会走 nextGCode，下次开库再补存量。
	if gdb != nil {
		if err := s.EnsureGCodes(); err != nil {
			slog.Warn("cost: 造价自有编码补授失败（下次开库重试）", "error", err)
		}
	}
	return s
}

// Available 报告存储是否可用。
func (s *Store) Available() bool { return s != nil && s.db != nil }

// DB 暴露底层句柄（app 层需要跨表事务时用；正常读写仍走本包方法）。
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

func (s *Store) requireDB() error {
	if !s.Available() {
		return fmt.Errorf("工料法存储不可用")
	}
	return nil
}

// nowText 统一时间戳格式（RFC3339；与 cost/costproject 落库口径一致）。
func nowText() string { return time.Now().UTC().Format(time.RFC3339) }

// ── 工料机资源库 ────────────────────────────────────────────────────

// SaveResource 新建/更新工料机资源。
//
// 身份键 = (Kind, Title, Spec, Unit, Region)，由唯一索引 idx_gf_resources_ident
// 硬约束；同身份再次保存走 UPDATE（不制造重复资源）。Code 为空时自动分配稳定
// 编码（人工/材料/机械 + 序号），保证定额引用锚点存在。
//
// 规矩（rules.go）在这里把关：类别必须是四种之一；地区、分类必须在字典里
// （空=待整理）；期数归一成 YYYY-MM。地区进了身份（2026-11-25 工料机按地区
// 分行）后严格按地区匹配——未标注地区的行不并入任何已标注地区的行，宁分勿并，
// 一个地区的价不顶掉另一个地区的价。
//
// 命中既有行时**编码保留原值**（编码是定额含量行的引用锚点）：入参带来的另一个
// 编码由调用方按别名解析到原码（见 ApplyProjectBundle）。
func (s *Store) SaveResource(r Resource) (*Resource, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	// 规矩校验与演练（cost_library save_resource dry_run）共用同一个入口。
	if err := s.ValidateResource(&r); err != nil {
		return nil, err
	}
	if r.Status == "" {
		r.Status = "现行"
	}
	r.Code = strings.TrimSpace(r.Code)

	now := nowText()
	// 按身份回查既有资源：命中则沿用其 id/code（改价不改身份）。
	var existID int64
	var existCode string
	err := s.db.QueryRow(
		`SELECT id, code FROM gf_resources WHERE kind=? AND title=? AND spec=? AND unit=? AND region=?`,
		r.Kind, r.Title, r.Spec, r.Unit, r.Region).Scan(&existID, &existCode)
	switch {
	case err == nil:
		if r.ID > 0 && r.ID != existID {
			return nil, fmt.Errorf("身份（类别+名称+规格+单位+地区）与工料机 %s「%s」重复——直接改那一条（给它的 id 或 code），或调整规格/单位/地区", existCode, r.Title)
		}
		r.ID = existID
		r.Code = existCode
	case err == sql.ErrNoRows:
		if r.Code == "" {
			code, e := s.nextResourceCode(r.Kind, r.Title)
			if e != nil {
				return nil, e
			}
			r.Code = code
		}
	default:
		return nil, err
	}

	tags, _ := json.Marshal(nonNilTags(r.Tags))
	if r.ID > 0 {
		if _, err = s.db.Exec(`
UPDATE gf_resources SET code=?, kind=?, title=?, spec=?, unit=?, base_price=?, current_price=?,
  category_path=?, source=?, supplier=?, region=?, price_date=?, price_type=?, valid_until=?,
  loss_rate=?, note=?, tags=?, status=?, updated_at=?
WHERE id=?`,
			r.Code, r.Kind, r.Title, r.Spec, r.Unit, r.BasePrice, r.CurrentPrice,
			r.CategoryPath, r.Source, r.Supplier, r.Region, r.PriceDate, r.PriceType, r.ValidUntil,
			r.LossRate, r.Note, string(tags), r.Status, now, r.ID); err != nil {
			return nil, err
		}
	} else {
		// 新身份行授自有码（R + 类别1 + 分类2 + 流水4，见 gcode.go）。
		gcode, e := s.newResourceGCode(s.db, &r)
		if e != nil {
			return nil, e
		}
		r.GCode = gcode
		res, e := s.db.Exec(`
INSERT INTO gf_resources(code, gcode, kind, title, spec, unit, base_price, current_price,
  category_path, source, supplier, region, price_date, price_type, valid_until,
  loss_rate, note, tags, status, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			r.Code, r.GCode, r.Kind, r.Title, r.Spec, r.Unit, r.BasePrice, r.CurrentPrice,
			r.CategoryPath, r.Source, r.Supplier, r.Region, r.PriceDate, r.PriceType, r.ValidUntil,
			r.LossRate, r.Note, string(tags), r.Status, now, now)
		if e != nil {
			if strings.Contains(e.Error(), "UNIQUE constraint failed: gf_resources.code") {
				return nil, fmt.Errorf("工料机编码 %s 已被占用", r.Code)
			}
			return nil, e
		}
		r.ID, _ = res.LastInsertId()
	}
	r.UpdatedAt = now
	if r.CreatedAt == "" {
		r.CreatedAt = now
	}
	return &r, nil
}

// ValidateResource 工料机入库前的规矩校验（SaveResource 与演练共用，规矩见 rules.go）：
// 类别必须是四种之一；地区、分类必须在字典里（空=待整理）；期数归一成 YYYY-MM。
// 通过后 r 的这几个字段已是归一后的值。
func (s *Store) ValidateResource(r *Resource) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	r.Kind = normalizeKind(r.Kind)
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" {
		return fmt.Errorf("工料机资源需要名称")
	}
	r.Spec = strings.TrimSpace(r.Spec)
	r.Unit = strings.TrimSpace(r.Unit)
	r.Region = strings.TrimSpace(r.Region)
	r.CategoryPath = strings.TrimSpace(r.CategoryPath)
	if !isKnownKind(r.Kind) {
		return fmt.Errorf("工料机「%s」的类别「%s」不在 人工/材料/机械/外委 之内", r.Title, r.Kind)
	}
	if err := s.checkDict(DictRegion, r.Region, "工料机地区"); err != nil {
		return err
	}
	if err := s.checkDict(DictCategory, r.CategoryPath, "工料机分类"); err != nil {
		return err
	}
	period, err := checkPeriod(r.PriceDate, "工料机价格期数")
	if err != nil {
		return err
	}
	r.PriceDate = period
	return nil
}

// CheckPrice 一次调价的口径校验（UpdateResourcePrice 与演练共用）：期数归一成
// YYYY-MM；地区必须在字典里；工料机按地区分行，一条行只代表一个地区的价——行上
// 已归属别的地区则拒绝。返回归一后的期数。
func (s *Store) CheckPrice(r *Resource, period, region string) (string, error) {
	normalized, err := checkPeriod(period, "调价期数")
	if err != nil {
		return "", err
	}
	region = strings.TrimSpace(region)
	if err := s.checkDict(DictRegion, region, "调价地区"); err != nil {
		return "", err
	}
	if region != "" && r.Region != "" && region != r.Region {
		return "", fmt.Errorf("工料机 %s「%s」已归属地区「%s」，不能记「%s」的价——请给「%s」的那条工料机记价（库里没有就先 save_resource 新建）",
			r.Code, r.Title, r.Region, region, region)
	}
	return normalized, nil
}

// findResourceByIdentity 按身份（类别+名称+规格+单位+地区）找既有行；没有返回 nil。
func (s *Store) findResourceByIdentity(r Resource) (*Resource, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	row, err := scanResource(s.db.QueryRow(resourceCols+` WHERE kind=? AND title=? AND spec=? AND unit=? AND region=?`,
		normalizeKind(r.Kind), strings.TrimSpace(r.Title), strings.TrimSpace(r.Spec), strings.TrimSpace(r.Unit), strings.TrimSpace(r.Region)))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// ResolveResourcePrev 全等身份（含地区）没命中时的第二跳：身份四要素
// （类别+名称+规格+单位）相同、**仅地区缺失或不同**的既有行视为同一资源，
// 更新原行而不是消歧成新行（铜陵反馈 #18：旧 60 行已标地区、重导入带着项目
// 地区，全等身份查不到 → 撞码消歧落成「码-N」重复行，还得 classify+dedupe
// 两步清理）。恰好一行命中才并；0 行或多行返回 nil（多行=真歧义，维持消歧）。
func (s *Store) ResolveResourcePrev(r Resource) (*Resource, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(resourceCols+` WHERE kind=? AND title=? AND spec=? AND unit=?`,
		normalizeKind(r.Kind), strings.TrimSpace(r.Title), strings.TrimSpace(r.Spec), strings.TrimSpace(r.Unit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []Resource
	for rows.Next() {
		row, err := scanResource(rows)
		if err != nil {
			return nil, err
		}
		hits = append(hits, *row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(hits) != 1 {
		return nil, nil
	}
	return &hits[0], nil
}

// FreeResourceCode 编码可用性：未被占用原样返回；已占用则返回「码-N」消歧码
// （N 从 2 起，取最小空位），与定额码 ResolveQuotaCode 的消歧规则同款。导入链
// 用它处理「两份工料机价格表用了同一个码」（L01 在每个项目里都是人工）。
func (s *Store) FreeResourceCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", nil
	}
	taken := func(c string) (bool, error) {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM gf_resources WHERE code=?`, c).Scan(&n); err != nil {
			return false, err
		}
		return n > 0, nil
	}
	used, err := taken(code)
	if err != nil || !used {
		return code, err
	}
	for n := 2; n <= 999; n++ {
		c := fmt.Sprintf("%s-%d", code, n)
		used, err := taken(c)
		if err != nil {
			return "", err
		}
		if !used {
			return c, nil
		}
	}
	return "", fmt.Errorf("资源编码 %s 已用尽可用后缀", code)
}

// sameResourceIdentity 两行是否同一身份（类别+名称+规格+单位+地区）。
func sameResourceIdentity(a, b Resource) bool {
	return normalizeKind(a.Kind) == normalizeKind(b.Kind) &&
		strings.TrimSpace(a.Title) == strings.TrimSpace(b.Title) &&
		strings.TrimSpace(a.Spec) == strings.TrimSpace(b.Spec) &&
		strings.TrimSpace(a.Unit) == strings.TrimSpace(b.Unit) &&
		strings.TrimSpace(a.Region) == strings.TrimSpace(b.Region)
}

// nextResourceCode 分配稳定资源编码。
//
// 编码风格对齐实测产物「工料机价格」表：**ASCII 助记码**（EXC 挖掘机 / ALU
// 筛分斗 / C20 商品混凝土 / HDPE 防渗膜 / TRK 运输），人工类走 L01/L02 式
// 编号。助记码比「材-0001」式序号可读得多，也正是源文件里 VLOOKUP 的锚点。
//
// 分配规则：先由名称提取助记码（ASCII 字母数字串优先，如 C20/HDPE/EXC），
// 撞码则追加 -2、-3…；纯中文名提取不到助记码时回退类别前缀 + 序号。
func (s *Store) nextResourceCode(kind, title string) (string, error) {
	base := mnemonicOf(title, kind)
	if base == "" {
		base = kindPrefix(kind)
		// 纯中文名无 ASCII 助记码 → 前缀 + 三位序号（P001/L002/M003）。
		// 对齐实测产物的人工编号风格（L01/L02），且比裸前缀「P」自解释得多：
		// 裸前缀在列表里一堆同名 P/M/L 无法区分。
		seq, err := s.nextPrefixSeq(base)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%03d", base, seq), nil
	}
	// 助记码可能撞码（EXC 挖掘机 vs EXC 挖掘机租赁），探测可用后缀。
	for i := 1; i <= 999; i++ {
		code := base
		if i > 1 {
			code = fmt.Sprintf("%s-%d", base, i)
		}
		var exist int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM gf_resources WHERE code=?`, code).Scan(&exist); err != nil {
			return "", err
		}
		if exist == 0 {
			return code, nil
		}
	}
	return "", fmt.Errorf("资源编码 %s 已用尽可用后缀", base)
}

// mnemonicOf 从名称提取 ASCII 助记码（最长 8 字符），对齐实测产物风格。
//
// 规则（从实测命名「C20商品混凝土 / HDPE防渗膜1.5mm / EXC挖掘机 / TRK 运输」
// 反推）：
//   - 以**字母**开头才提取（纯数字开头如「12 号槽钢」无有效助记码 → 走类别前缀）；
//   - 取开头的连续字母段，ASCII 标点（. - _ 空格）不断链，故「P.O 42.5」→ PO42；
//   - 紧随字母段之后的连续数字段并入（「C20…」→ C20）；
//   - 遇中文即刻结束、**不跨中文续接**（否则「HDPE防渗膜1.5mm」会误得 HDPE15MM）。
func mnemonicOf(title, kind string) string {
	runes := []rune(strings.TrimSpace(title))
	isLetter := func(r rune) bool { return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' }
	isDigit := func(r rune) bool { return r >= '0' && r <= '9' }
	isSep := func(r rune) bool { return r == '.' || r == '-' || r == '_' || r == ' ' }

	// 必须字母开头。
	if len(runes) == 0 || !isLetter(runes[0]) {
		return ""
	}
	var b strings.Builder
	i := 0
	// ① 字母段（标点不断链）。
	for i < len(runes) {
		r := runes[i]
		if isLetter(r) {
			b.WriteRune(r)
			i++
			continue
		}
		if isSep(r) {
			i++
			continue
		}
		break
	}
	// ② 紧随的数字段（遇非数字即停，不跨字符续接）。
	for i < len(runes) && isDigit(runes[i]) {
		b.WriteRune(runes[i])
		i++
	}

	tok := strings.ToUpper(b.String())
	if len(tok) < 2 {
		return ""
	}
	if len(tok) > 8 {
		tok = tok[:8]
	}
	return tok
}

// kindPrefix 无 ASCII 名称时的类别前缀。
func kindPrefix(kind string) string {
	switch kind {
	case KindLabor:
		return "L"
	case KindMachine:
		return "M"
	default:
		return "P" // 材料（Material 的 M 已给机械，取 P 避免撞前缀）
	}
}

// nextPrefixSeq 取该前缀下已用的最大序号 + 1（用于「前缀+三位序号」编码）。
func (s *Store) nextPrefixSeq(prefix string) (int, error) {
	rows, err := s.db.Query(`SELECT code FROM gf_resources WHERE code LIKE ?`, prefix+"%")
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
		digits := strings.TrimPrefix(code, prefix)
		if digits == code {
			continue
		}
		if n, ok := parseNum(digits); ok && n > 0 {
			if v := int(n); v > maxSeq {
				maxSeq = v
			}
		}
	}
	return maxSeq + 1, rows.Err()
}

// GetResource 按 id 读取资源。
func (s *Store) GetResource(id int64) (*Resource, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	row := s.db.QueryRow(resourceCols+` WHERE id=?`, id)
	r, err := scanResource(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("工料机资源 %d 不存在", id)
		}
		return nil, err
	}
	return r, nil
}

// GetResourceByCode 按编码读取资源（定额引用解析用）。
func (s *Store) GetResourceByCode(code string) (*Resource, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	row := s.db.QueryRow(resourceCols+` WHERE code=?`, strings.TrimSpace(code))
	r, err := scanResource(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("工料机资源 %q 不存在", code)
		}
		return nil, err
	}
	return r, nil
}

// ListResources 返回资源列表（kind/keyword 均可为空 = 不过滤），按类别序 + 名称。
func (s *Store) ListResources(kind, keyword string) ([]Resource, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	q := resourceCols + ` WHERE 1=1`
	var args []any
	if k := strings.TrimSpace(kind); k != "" && k != "全部" {
		q += ` AND kind=?`
		args = append(args, k)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		q += ` AND (title LIKE ? OR spec LIKE ? OR code LIKE ? OR COALESCE(gcode,'') LIKE ? OR category_path LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like, like, like, like)
	}
	q += ` ORDER BY kind, title, spec`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Resource
	for rows.Next() {
		r, e := scanResource(rows)
		if e != nil {
			continue
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// DeleteResource 删除资源。默认拒绝删除仍被消耗定额引用的资源（引用完整性），
// force=true 时连带清理其调价历史（定额行保留为孤立引用，读取侧显形为缺失）。
func (s *Store) DeleteResource(id int64, force bool) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	r, err := s.GetResource(id)
	if err != nil {
		return err
	}
	if !force {
		var n int
		if e := s.db.QueryRow(
			`SELECT COUNT(*) FROM gf_quota_items WHERE resource_code=?`, r.Code).Scan(&n); e != nil {
			return e
		}
		if n > 0 {
			return fmt.Errorf("资源「%s」仍被 %d 条消耗定额引用，不能删除（可改为归档）", r.Title, n)
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM gf_resource_prices WHERE resource_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM gf_resources WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateResourcePrice 登记一次资源调价：写调价历史 + 推进现行价。
//
// 这是「信息价一涨、所有引用该资源的综合单价同步重算」的唯一入口——现行价
// 变了，引用它的定额子目与综合单价在下次读取时自然按新价重算，无需逐条改价。
//
// 行上的期与口径随现行价走（2026-11-25）：price_date/price_type 更新为本次的值
// （没给保持原值）。地区只记在价格事实上：工料机按地区分行，一条行只代表一个
// 地区的价——行上已归属别的地区则拒绝（要记别的地区的价，请给那个地区的那条）；
// 行上地区为空表示尚未整理，不由调价顺手改身份（走 classify）。
func (s *Store) UpdateResourcePrice(id int64, price float64, period, region, priceType, source, note string) (*Resource, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	if price <= 0 {
		return nil, fmt.Errorf("调价须为正数（收到 %v）", price)
	}
	r, err := s.GetResource(id)
	if err != nil {
		return nil, err
	}
	period, err = s.CheckPrice(r, period, region)
	if err != nil {
		return nil, err
	}
	region = strings.TrimSpace(region)
	priceType = strings.TrimSpace(priceType)
	now := nowText()
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
INSERT INTO gf_resource_prices(resource_id, price, period, region, price_type, source, fetched_at, note)
VALUES(?,?,?,?,?,?,?,?)`, id, price, period, region, priceType, source, now, note); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`
UPDATE gf_resources SET current_price=?,
  price_date=CASE WHEN ?<>'' THEN ? ELSE price_date END,
  price_type=CASE WHEN ?<>'' THEN ? ELSE price_type END,
  updated_at=?
WHERE id=?`, price, period, period, priceType, priceType, now, id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	r.CurrentPrice = price
	if period != "" {
		r.PriceDate = period
	}
	if priceType != "" {
		r.PriceType = priceType
	}
	r.UpdatedAt = now
	return r, nil
}

// RecordPriceFact 登记一条价格事实（工料机 × 地区 × 期），不动现行价。
// 幂等键 = (资源, 期, 地区, 来源)：同键已有行则更新价格，重复导入不堆积重复行。
func (s *Store) RecordPriceFact(resourceID int64, price float64, period, region, priceType, source, note string) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	if resourceID <= 0 || price <= 0 {
		return nil
	}
	normalized, err := checkPeriod(period, "价格期数")
	if err != nil {
		return err
	}
	period = normalized
	region = strings.TrimSpace(region)
	if err := s.checkDict(DictRegion, region, "价格地区"); err != nil {
		return err
	}
	now := nowText()
	res, err := s.db.Exec(`UPDATE gf_resource_prices SET price=?, price_type=?, fetched_at=?, note=?
WHERE resource_id=? AND period=? AND region=? AND source=?`,
		price, priceType, now, note, resourceID, period, region, source)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = s.db.Exec(`INSERT INTO gf_resource_prices(resource_id, price, period, region, price_type, source, fetched_at, note)
VALUES(?,?,?,?,?,?,?,?)`, resourceID, price, period, region, priceType, source, now, note)
	return err
}

// ResourcePrices 返回某资源的调价历史（新→旧）。
func (s *Store) ResourcePrices(resourceID int64) ([]ResourcePrice, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`
SELECT id, resource_id, price, period, region, price_type, source, fetched_at, note
FROM gf_resource_prices WHERE resource_id=? ORDER BY id DESC`, resourceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []ResourcePrice
	for rows.Next() {
		var p ResourcePrice
		if e := rows.Scan(&p.ID, &p.ResourceID, &p.Price, &p.Period, &p.Region,
			&p.PriceType, &p.Source, &p.FetchedAt, &p.Note); e != nil {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ResourceIndex 按 code 建资源索引（核算批量解析用，避免逐行查库）。
func (s *Store) ResourceIndex() (map[string]Resource, error) {
	list, err := s.ListResources("", "")
	if err != nil {
		return nil, err
	}
	idx := make(map[string]Resource, len(list))
	for _, r := range list {
		idx[r.Code] = r
	}
	return idx, nil
}

// ── 消耗定额库 ──────────────────────────────────────────────────────

// SaveQuota 新建/更新消耗定额（含其工料机含量行；整组替换）。
// 含量行的 Title/Spec/Unit/Kind 若为空，自动从被引用的资源补齐（定额表只需
// 填 resource_code + quantity，展示字段由资源库派生，避免两处维护同一事实）。
func (s *Store) SaveQuota(q Quota, idx map[string]Resource) (*Quota, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	saved, err := s.saveQuota(tx, q, idx)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return saved, nil
}

// ValidateQuota 定额入库前的规矩校验（SaveQuota 与演练共用，规矩见 rules.go）：
// 编码、名称必填；专业必须在字典里（空=待整理）；地区、期数同理，期数归一成
// YYYY-MM。通过后 q 的这几个字段已是归一后的值。
func (s *Store) ValidateQuota(q *Quota) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	q.Code = strings.TrimSpace(q.Code)
	q.Title = strings.TrimSpace(q.Title)
	if q.Code == "" {
		return fmt.Errorf("消耗定额需要编码")
	}
	if q.Title == "" {
		return fmt.Errorf("消耗定额需要名称")
	}
	q.Specialty = strings.TrimSpace(q.Specialty)
	q.Region = strings.TrimSpace(q.Region)
	if err := s.checkDict(DictSpecialty, q.Specialty, "定额专业"); err != nil {
		return err
	}
	if err := s.checkDict(DictRegion, q.Region, "定额地区"); err != nil {
		return err
	}
	period, err := checkPeriod(q.PriceDate, "定额价格期数")
	if err != nil {
		return err
	}
	q.PriceDate = period
	return nil
}

// saveQuota 定额落库的主体：在调用方给的句柄/事务上执行（借鉴生成要与改挂清单行
// 同一个事务，所以不开自己的事务）。
func (s *Store) saveQuota(ex executor, q Quota, idx map[string]Resource) (*Quota, error) {
	if err := s.ValidateQuota(&q); err != nil {
		return nil, err
	}
	if q.Status == "" {
		q.Status = "现行"
	}
	if idx == nil {
		var err error
		if idx, err = s.ResourceIndex(); err != nil {
			return nil, err
		}
	}
	now := nowText()

	var existID int64
	var createdAt string
	e := ex.QueryRow(`SELECT id, created_at FROM gf_quotas WHERE code=?`, q.Code).Scan(&existID, &createdAt)
	// 注意：e 在命中/未命中两路上都会被复用，故下面各分支一律显式判定并立即
	// 处置，不把「上一步的错误值」带到后续成功路径（这正是 SaveResource 早期
	// 版本踩过的坑：回查的 ErrNoRows 残留，insert 成功却带错返回）。
	switch e {
	case nil:
		q.ID = existID
		if createdAt != "" {
			q.CreatedAt = createdAt
		}
		if _, err := ex.Exec(`
UPDATE gf_quotas SET title=?, specialty=?, chapter=?, unit=?, category_path=?,
  base_labor=?, base_material=?, base_machine=?, source=?, region=?, price_date=?,
  note=?, status=?, orig_code=?, derived_from=?, updated_at=? WHERE id=?`,
			q.Title, q.Specialty, q.Chapter, q.Unit, q.CategoryPath,
			q.BaseLabor, q.BaseMaterial, q.BaseMachine, q.Source, q.Region, q.PriceDate,
			q.Note, q.Status, q.OrigCode, q.DerivedFrom, now, q.ID); err != nil {
			return nil, err
		}
	case sql.ErrNoRows:
		// 新定额授自有码（D + 专业2 + 章节2 + 流水3，见 gcode.go）。
		gcode, e1 := s.newQuotaGCode(ex, &q)
		if e1 != nil {
			return nil, e1
		}
		q.GCode = gcode
		res, e2 := ex.Exec(`
INSERT INTO gf_quotas(code, gcode, title, specialty, chapter, unit, category_path,
  base_labor, base_material, base_machine, source, region, price_date, note, status, orig_code, derived_from, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			q.Code, q.GCode, q.Title, q.Specialty, q.Chapter, q.Unit, q.CategoryPath,
			q.BaseLabor, q.BaseMaterial, q.BaseMachine, q.Source, q.Region, q.PriceDate,
			q.Note, q.Status, q.OrigCode, q.DerivedFrom, now, now)
		if e2 != nil {
			return nil, e2
		}
		q.ID, _ = res.LastInsertId()
	default:
		return nil, e
	}
	if q.CreatedAt == "" {
		q.CreatedAt = now
	}
	q.UpdatedAt = now

	if _, err := ex.Exec(`DELETE FROM gf_quota_items WHERE quota_code=?`, q.Code); err != nil {
		return nil, err
	}
	for i := range q.Items {
		it := &q.Items[i]
		it.QuotaCode = q.Code
		it.ResourceCode = strings.TrimSpace(it.ResourceCode)
		if it.ResourceCode == "" {
			continue
		}
		if r, ok := idx[it.ResourceCode]; ok {
			if strings.TrimSpace(it.Kind) == "" {
				it.Kind = r.Kind
			}
			if strings.TrimSpace(it.Title) == "" {
				it.Title = r.Title
			}
			if strings.TrimSpace(it.Spec) == "" {
				it.Spec = r.Spec
			}
			if strings.TrimSpace(it.Unit) == "" {
				it.Unit = r.Unit
			}
			if it.ResourcePrice <= 0 {
				it.ResourcePrice = r.EffectivePrice()
			}
		}
		it.Kind = normalizeKind(it.Kind)
		if it.Quantity < 0 {
			it.Quantity = 0
		}
		if _, err := ex.Exec(`
INSERT INTO gf_quota_items(quota_code, resource_code, kind, title, spec, unit,
  quantity, resource_price, loss_rate, note, sort, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			it.QuotaCode, it.ResourceCode, it.Kind, it.Title, it.Spec, it.Unit,
			it.Quantity, it.ResourcePrice, it.LossRate, it.Note, i, now, now); err != nil {
			return nil, err
		}
	}
	return &q, nil
}

// GetQuota 按编码读取定额及其含量行。
func (s *Store) GetQuota(code string) (*Quota, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	var q Quota
	err := s.db.QueryRow(`
SELECT id, code, COALESCE(gcode,''), COALESCE(orig_code,''), title, specialty, chapter, unit, category_path,
  base_labor, base_material, base_machine, source, region, price_date, note, status, created_at, updated_at,
  COALESCE(derived_from,''),
  (SELECT COUNT(*) FROM gf_bill_items b WHERE b.quota_code=gf_quotas.code)
FROM gf_quotas WHERE code=?`, code).Scan(
		&q.ID, &q.Code, &q.GCode, &q.OrigCode, &q.Title, &q.Specialty, &q.Chapter, &q.Unit, &q.CategoryPath,
		&q.BaseLabor, &q.BaseMaterial, &q.BaseMachine, &q.Source, &q.Region, &q.PriceDate,
		&q.Note, &q.Status, &q.CreatedAt, &q.UpdatedAt, &q.DerivedFrom, &q.UsageCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("消耗定额 %q 不存在", code)
		}
		return nil, err
	}
	items, err := s.quotaItemsOf(code)
	if err != nil {
		return nil, err
	}
	q.Items = items
	return &q, nil
}

// quotaItemsOf 读取定额的含量行（按 sort,id）。
func (s *Store) quotaItemsOf(code string) ([]QuotaItem, error) {
	rows, err := s.db.Query(`
SELECT id, quota_code, resource_code, kind, title, spec, unit, quantity, resource_price, loss_rate, note, sort
FROM gf_quota_items WHERE quota_code=? ORDER BY sort, id`, code)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []QuotaItem
	for rows.Next() {
		var it QuotaItem
		if e := rows.Scan(&it.ID, &it.QuotaCode, &it.ResourceCode, &it.Kind, &it.Title,
			&it.Spec, &it.Unit, &it.Quantity, &it.ResourcePrice, &it.LossRate,
			&it.Note, &it.Sort); e != nil {
			continue
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ListQuotas 返回定额列表（specialty/keyword 可空），不含含量行（列表轻量）。
func (s *Store) ListQuotas(specialty, keyword string) ([]Quota, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	q := `
SELECT id, code, COALESCE(gcode,''), COALESCE(orig_code,''), title, specialty, chapter, unit, category_path,
  base_labor, base_material, base_machine, source, region, price_date, note, status, created_at, updated_at,
  COALESCE(derived_from,''),
  (SELECT COUNT(*) FROM gf_bill_items b WHERE b.quota_code=gf_quotas.code)
FROM gf_quotas WHERE 1=1`
	var args []any
	if sp := strings.TrimSpace(specialty); sp != "" && sp != "全部" {
		q += ` AND specialty=?`
		args = append(args, sp)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		q += ` AND (title LIKE ? OR code LIKE ? OR COALESCE(gcode,'') LIKE ? OR chapter LIKE ?)`
		like := "%" + kw + "%"
		args = append(args, like, like, like, like)
	}
	q += ` ORDER BY specialty, chapter, code`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Quota
	for rows.Next() {
		var it Quota
		if e := rows.Scan(&it.ID, &it.Code, &it.GCode, &it.OrigCode, &it.Title, &it.Specialty, &it.Chapter, &it.Unit,
			&it.CategoryPath, &it.BaseLabor, &it.BaseMaterial, &it.BaseMachine, &it.Source,
			&it.Region, &it.PriceDate, &it.Note, &it.Status, &it.CreatedAt, &it.UpdatedAt, &it.DerivedFrom, &it.UsageCount); e != nil {
			continue
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// DeleteQuota 删除定额及其含量行。
func (s *Store) DeleteQuota(code string) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM gf_quota_items WHERE quota_code=?`, code); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM gf_quotas WHERE code=?`, code); err != nil {
		return err
	}
	return tx.Commit()
}

// ── 核算入口 ────────────────────────────────────────────────────────

// ComposeQuota 按定额核算综合单价：定额含量 × 资源**现行价**（缺失回退快照价）。
//
// overrides 允许项目级覆盖：key=resource_code，value=(含量, 单价)；含量或单价
// 为负表示不覆盖该项。这正是「全局定额 + 项目级覆盖」的核算落点——定额说标准，
// 项目说实际，核算按项目覆盖后的口径算，未覆盖项一律取全局标准。
//
// json 标签必须写：本类型经绑定面（GaeaWorkcostQuotaCompose）直达前端，缺标签
// 时 Wails 输出 PascalCase 而前端按 camelCase 读，得到 undefined（WireShape 门禁
// 会拦下这类遗漏）。
type ComposeOverride struct {
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// ComposeQuota 核算一条定额的综合单价（**只含人材机**，不含管理费/利润/税金）。
func (s *Store) ComposeQuota(code string, overrides map[string]ComposeOverride) (*Compose, *Quota, error) {
	q, err := s.GetQuota(code)
	if err != nil {
		return nil, nil, err
	}
	idx, err := s.ResourceIndex()
	if err != nil {
		return nil, nil, err
	}
	lines, warnings := q.ComposeLines(idx, overrides)
	c := ComposeUnitPrice(lines)
	c.Warnings = append(warnings, c.Warnings...)
	return &c, q, nil
}

// ComposeQuotas 批量核算（清单库参考价列专用）：一次建资源索引，逐码复用——
// 逐行调用 ComposeQuota 时每个编码都全量扫资源表建索引（63 行清单 = 63 次全扫，
// 真机首屏 longtask ~700ms）。不存在的编码不写入结果（调用方按缺失回退显示）。
func (s *Store) ComposeQuotas(codes []string) (map[string]*Compose, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	idx, err := s.ResourceIndex()
	if err != nil {
		return nil, err
	}
	out := make(map[string]*Compose, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		q, err := s.GetQuota(code)
		if err != nil {
			continue // 不存在/读失败：缺失即回退信号，不中断整批
		}
		lines, warnings := q.ComposeLines(idx, nil)
		c := ComposeUnitPrice(lines)
		c.Warnings = append(warnings, c.Warnings...)
		out[code] = &c
	}
	return out, nil
}

// ComposeLines 把定额含量行解析为核算组成行（资源现行价优先，缺失回退快照价）。
// 返回的 warnings 记录孤立引用（资源已删除）——不静默归零，让「单价偏低」有据可查。
func (q *Quota) ComposeLines(idx map[string]Resource, overrides map[string]ComposeOverride) ([]ComposeLine, []string) {
	var lines []ComposeLine
	var warnings []string
	for _, it := range q.Items {
		kind := it.Kind
		title := it.Title
		unit := it.Unit
		price := it.ResourcePrice
		loss := it.LossRate
		if r, ok := idx[it.ResourceCode]; ok {
			kind = r.Kind
			title = r.Title
			unit = r.Unit
			if p := r.EffectivePrice(); p > 0 {
				price = p
			}
			if loss <= 0 {
				loss = r.LossRate
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("资源 %s（%s）已删除，按定额快照价 %.2f 核算", it.ResourceCode, it.Title, it.ResourcePrice))
		}
		qty := it.Quantity
		if ov, ok := overrides[it.ResourceCode]; ok {
			if ov.Quantity >= 0 {
				qty = ov.Quantity
			}
			if ov.Price >= 0 {
				price = ov.Price
			}
		}
		lines = append(lines, ComposeLine{
			Kind: kind, Title: title, Unit: unit,
			Quantity: qty, Price: price, LossRate: loss,
		})
	}
	sort.SliceStable(lines, func(i, j int) bool {
		return kindRank(lines[i].Kind) < kindRank(lines[j].Kind)
	})
	return lines, warnings
}

// kindRank 类别排序权重（人工→材料→机械→外委，未知排末尾）。
func kindRank(k string) int {
	switch strings.TrimSpace(k) {
	case KindLabor:
		return 0
	case KindMaterial:
		return 1
	case KindMachine:
		return 2
	case KindOutsourced:
		return 3
	default:
		return 4
	}
}

// normalizeKind 归一工料机类别：容错「人工费/材料费/机械费/外委」等写法，
// 未知写法原样保留（不猜类别，由核算层 warn 显形）。
func normalizeKind(k string) string {
	t := strings.TrimSpace(k)
	switch t {
	case "", "人材机", "工料机":
		return KindMaterial
	}
	base := strings.TrimSuffix(t, "费")
	switch base {
	case "人工", "普工", "技工", "劳务":
		return KindLabor
	case "材料", "主材", "辅材", "设备":
		return KindMaterial
	case "机械", "机械台班", "台班", "机具":
		return KindMachine
	case "外委", "委外", "外购", "外包", "分包":
		return KindOutsourced
	}
	if strings.Contains(base, "人工") || strings.Contains(base, "工日") {
		return KindLabor
	}
	if strings.Contains(base, "机械") || strings.Contains(base, "台班") {
		return KindMachine
	}
	if strings.Contains(base, "外委") || strings.Contains(base, "委外") {
		return KindOutsourced
	}
	if strings.Contains(base, "材料") {
		return KindMaterial
	}
	return t
}

// NormalizeKind 导出归一（app/导入层复用同一口径）。
func NormalizeKind(k string) string { return normalizeKind(k) }

// ── 扫描辅助 ────────────────────────────────────────────────────────

// resourceCols 资源表查询列（Get/List 共用，避免两处列序漂移）。
const resourceCols = `
SELECT id, code, COALESCE(gcode,''), kind, title, spec, unit, base_price, current_price, category_path,
  source, supplier, region, price_date, price_type, valid_until, loss_rate, note, tags, status,
  created_at, updated_at
FROM gf_resources`

// scanner 抽象 *sql.Row / *sql.Rows 的公共 Scan。
type scanner interface {
	Scan(dest ...any) error
}

func scanResource(sc scanner) (*Resource, error) {
	var r Resource
	var tags string
	if err := sc.Scan(&r.ID, &r.Code, &r.GCode, &r.Kind, &r.Title, &r.Spec, &r.Unit,
		&r.BasePrice, &r.CurrentPrice, &r.CategoryPath, &r.Source, &r.Supplier,
		&r.Region, &r.PriceDate, &r.PriceType, &r.ValidUntil, &r.LossRate,
		&r.Note, &tags, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	r.Tags = strutil.ParseTagsJSON(tags)
	return &r, nil
}

func nonNilTags(t []string) []string {
	if t == nil {
		return []string{}
	}
	return t
}
