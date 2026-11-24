package workcost

// usage.go — 三级数据的引用面与删除护栏（2026-11-25 造价库重建）。
//
// 清单 → 定额 → 工料机 是一条引用链：清单项挂接定额（gf_bill_quota_links），
// 定额含量行引用工料机（gf_quota_items.resource_code）。本文件回答「这条数据
// 谁在用」，并给删除动作一道护栏——被引用的定额默认不删，避免留下悬空引用
// 让现算链静默缺项（工料机侧的护栏已在 DeleteResource）。

import (
	"fmt"
	"strings"
	"time"
)

// UsageRef 一条引用：谁在用这条数据。
//
// 定额的引用方是清单项（Owner=所属项目，Quantity=挂接工程量）；工料机的引用方
// 是定额含量行（Unit=定额计量单位，Quantity=含量）。
type UsageRef struct {
	ID       int64   `json:"id"`
	Code     string  `json:"code"`
	Title    string  `json:"title"`
	Owner    string  `json:"owner,omitempty"`
	Unit     string  `json:"unit,omitempty"`
	Quantity float64 `json:"quantity"`
}

// QuotaUsage 定额被哪些清单项引用（按项目、清单顺序）。挂接以引用表为准；
// 引用表没行的旧清单退回单值 quota_code 列——与 BillItems 聚合同一口径。
// 三个占位符都是同一个定额码（不用 ?1 位置参数：与匿名占位符混用时序号容易错位）。
func (s *Store) QuotaUsage(code string) ([]UsageRef, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return []UsageRef{}, nil
	}
	rows, err := s.db.Query(`SELECT b.id, b.code, b.title, p.name,
  COALESCE((SELECT l.quantity FROM gf_bill_quota_links l WHERE l.bill_item_id=b.id AND l.quota_code=?), 0)
FROM gf_bill_items b JOIN gf_projects p ON p.id=b.project_id
WHERE CASE WHEN EXISTS(SELECT 1 FROM gf_bill_quota_links x WHERE x.bill_item_id=b.id)
  THEN (SELECT COUNT(*) FROM gf_bill_quota_links l WHERE l.bill_item_id=b.id AND l.quota_code=?)
  ELSE CASE WHEN b.quota_code=? THEN 1 ELSE 0 END END > 0
ORDER BY p.name, b.sort, b.id`, code, code, code)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []UsageRef{}
	for rows.Next() {
		var r UsageRef
		if err := rows.Scan(&r.ID, &r.Code, &r.Title, &r.Owner, &r.Quantity); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ResourceUsage 工料机被哪些定额含量行引用（按定额编码）。
func (s *Store) ResourceUsage(code string) ([]UsageRef, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return []UsageRef{}, nil
	}
	rows, err := s.db.Query(`SELECT q.id, q.code, q.title, COALESCE(q.unit,''), qi.quantity
FROM gf_quota_items qi JOIN gf_quotas q ON q.code=qi.quota_code
WHERE qi.resource_code=? ORDER BY q.code`, code)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []UsageRef{}
	for rows.Next() {
		var r UsageRef
		if err := rows.Scan(&r.ID, &r.Code, &r.Title, &r.Unit, &r.Quantity); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// QuotaReferenced 仍挂接该定额的清单项数（删除定额的前置检查）。
func (s *Store) QuotaReferenced(code string) (int, error) {
	refs, err := s.QuotaUsage(code)
	if err != nil {
		return 0, err
	}
	return len(refs), nil
}

// ResourceReferenceCounts 每个工料机被多少条定额含量行引用（一次聚合查询；
// 列表导出用，不逐条扫库）。没被引用的工料机不在结果里。
func (s *Store) ResourceReferenceCounts() (map[string]int, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT resource_code, COUNT(*) FROM gf_quota_items
WHERE COALESCE(resource_code,'')!='' GROUP BY resource_code`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var code string
		var n int
		if err := rows.Scan(&code, &n); err != nil {
			continue
		}
		out[code] = n
	}
	return out, rows.Err()
}

// DetachQuotaEverywhere 解除该定额在全部清单项上的挂接（强删定额前的清理）。
// 单值主定额列同步改指该清单项剩余挂接的第一条——一条不剩才退回手填行
// （cost_audit 会点名「无计价路径」）。整批一个事务，返回受影响的清单项数。
func (s *Store) DetachQuotaEverywhere(code string) (int, error) {
	if err := s.requireDB(); err != nil {
		return 0, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, fmt.Errorf("需要定额编码")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM gf_bill_items b
WHERE b.quota_code=? OR EXISTS(SELECT 1 FROM gf_bill_quota_links l WHERE l.bill_item_id=b.id AND l.quota_code=?)`,
		code, code).Scan(&n); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`DELETE FROM gf_bill_quota_links WHERE quota_code=?`, code); err != nil {
		return 0, err
	}
	// 单值主定额列：原本指着它的清单项，改指剩余挂接的第一条；一条不剩=退回手填行。
	if _, err := tx.Exec(`UPDATE gf_bill_items SET quota_code=COALESCE(
  (SELECT l.quota_code FROM gf_bill_quota_links l WHERE l.bill_item_id=gf_bill_items.id ORDER BY l.sort, l.id LIMIT 1), '')
WHERE quota_code=?`, code); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}

// ── 清单挂接集合的整体替换 ────────────────────────────────────────────

// BillQuotaLinkInput 一条待落库的定额挂接（Quantity=该定额参与合价的工程量，
// 0=跟随清单工程量）。
type BillQuotaLinkInput struct {
	QuotaCode string
	Quantity  float64
}

// SaveBillItemLinks 整体替换一条清单项的定额挂接集合。
//
// 造价模式 save_items 的 quota_code/quota_codes 落点（2026-11-25：此前这两个
// 参数只进回显不落引用表，多定额挂接其实没保存）。整体替换语义——清单项原有
// 的挂接全部解除，换成 links；主定额列 quota_code 同步成首项（空集合=清单项
// 退回手填行）。定额必须存在；同码去重保序。调用方没给挂接集合时不要调用它。
func (s *Store) SaveBillItemLinks(billItemID int64, links []BillQuotaLinkInput) ([]BillQuotaLink, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM gf_bill_items WHERE id=?`, billItemID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, fmt.Errorf("清单项 #%d 不存在", billItemID)
	}
	seen := map[string]bool{}
	clean := make([]BillQuotaLinkInput, 0, len(links))
	for _, l := range links {
		code := strings.TrimSpace(l.QuotaCode)
		if code == "" || seen[code] {
			continue
		}
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM gf_quotas WHERE code=?`, code).Scan(&n); err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, fmt.Errorf("定额 %s 不存在，无法挂接", code)
		}
		seen[code] = true
		q := l.Quantity
		if q < 0 {
			q = 0
		}
		clean = append(clean, BillQuotaLinkInput{QuotaCode: code, Quantity: q})
	}

	now := time.Now().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM gf_bill_quota_links WHERE bill_item_id=?`, billItemID); err != nil {
		return nil, err
	}
	primary := ""
	out := make([]BillQuotaLink, 0, len(clean))
	for i, l := range clean {
		res, err := tx.Exec(`INSERT INTO gf_bill_quota_links (bill_item_id, quota_code, quantity, sort, created_at, updated_at)
VALUES (?,?,?,?,?,?)`, billItemID, l.QuotaCode, l.Quantity, i+1, now, now)
		if err != nil {
			return nil, err
		}
		id, _ := res.LastInsertId()
		if i == 0 {
			primary = l.QuotaCode
		}
		out = append(out, BillQuotaLink{ID: id, BillItemID: billItemID, QuotaCode: l.QuotaCode,
			Quantity: l.Quantity, Sort: i + 1, CreatedAt: now, UpdatedAt: now})
	}
	if _, err := tx.Exec(`UPDATE gf_bill_items SET quota_code=? WHERE id=?`, primary, billItemID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}
