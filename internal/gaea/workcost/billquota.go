// billquota.go — 工料法第⑤层：清单↔定额 1:N 组合引用（gf_bill_quota_links）。
//
// 用户拍板（2026-10-07）：「如果一个清单是几个定额组成的呢？」——标准清单计价
// 一条清单项可套多条定额子目（各带自身工程量），综合单价是几条定额的加权和。
// 此前 gf_bill_items.quota_code 单值只能表达 1:1（五表导入把清单下多行消耗
// 并成一条定额）。
//
// 口径不变：引用与工程量是**录入数据**，加总不在库里发生（综合单价分析按定额
// 分组现算展示，合价归导出的五表 Excel）。单值 quota_code 列保留为主定额
// （旧展示/导出锚点不动），导入链双写。
package workcost

import (
	"fmt"
	"strings"
	"time"
)

// BillQuotaLink 清单项挂接的一条定额引用。
type BillQuotaLink struct {
	ID         int64  `json:"id"`
	BillItemID int64  `json:"billItemId"`
	QuotaCode  string `json:"quotaCode"`
	// Quantity 该定额参与合价的工程量；0 = 跟随清单工程量（录入数据）。
	Quantity  float64 `json:"quantity"`
	Sort      int     `json:"sort"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

// BillQuotaLinks 清单项的定额引用（按 sort, id 稳定序）。构造即非 nil——
// 绑定面 nil 切片会序列化成 JSON null 崩前端（v4.467.1 教训）。
func (s *Store) BillQuotaLinks(billItemID int64) ([]BillQuotaLink, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id, bill_item_id, quota_code, quantity, sort, created_at, updated_at
FROM gf_bill_quota_links WHERE bill_item_id=? ORDER BY sort, id`, billItemID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []BillQuotaLink{}
	for rows.Next() {
		var l BillQuotaLink
		if err := rows.Scan(&l.ID, &l.BillItemID, &l.QuotaCode, &l.Quantity, &l.Sort, &l.CreatedAt, &l.UpdatedAt); err != nil {
			continue
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// AttachBillQuotaLink 挂接一条定额（同清单同定额走更新——幂等；quantity≤0
// 表示跟随清单工程量）。定额必须存在（GetQuota 校验），防挂空引用。
func (s *Store) AttachBillQuotaLink(billItemID int64, quotaCode string, quantity float64) (BillQuotaLink, error) {
	if err := s.requireDB(); err != nil {
		return BillQuotaLink{}, err
	}
	quotaCode = strings.TrimSpace(quotaCode)
	if billItemID <= 0 {
		return BillQuotaLink{}, fmt.Errorf("挂接需要清单项")
	}
	if quotaCode == "" {
		return BillQuotaLink{}, fmt.Errorf("挂接需要定额编码")
	}
	if _, err := s.GetQuota(quotaCode); err != nil {
		return BillQuotaLink{}, fmt.Errorf("定额 %s 不存在，无法挂接", quotaCode)
	}
	if quantity < 0 {
		quantity = 0
	}
	now := time.Now().Format(time.RFC3339)
	var id int64
	err := s.db.QueryRow(`SELECT id FROM gf_bill_quota_links WHERE bill_item_id=? AND quota_code=?`, billItemID, quotaCode).Scan(&id)
	if err == nil {
		if _, err := s.db.Exec(`UPDATE gf_bill_quota_links SET quantity=?, updated_at=? WHERE id=?`, quantity, now, id); err != nil {
			return BillQuotaLink{}, err
		}
		var l BillQuotaLink
		if err := s.db.QueryRow(`SELECT id, bill_item_id, quota_code, quantity, sort, created_at, updated_at FROM gf_bill_quota_links WHERE id=?`, id).
			Scan(&l.ID, &l.BillItemID, &l.QuotaCode, &l.Quantity, &l.Sort, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return BillQuotaLink{}, err
		}
		return l, nil
	}
	if !strings.Contains(err.Error(), "no rows") {
		return BillQuotaLink{}, err
	}
	var maxSort int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(sort),0) FROM gf_bill_quota_links WHERE bill_item_id=?`, billItemID).Scan(&maxSort); err != nil {
		return BillQuotaLink{}, err
	}
	res, err := s.db.Exec(`INSERT INTO gf_bill_quota_links (bill_item_id, quota_code, quantity, sort, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		billItemID, quotaCode, quantity, maxSort+1, now, now)
	if err != nil {
		return BillQuotaLink{}, err
	}
	id, _ = res.LastInsertId()
	return BillQuotaLink{ID: id, BillItemID: billItemID, QuotaCode: quotaCode, Quantity: quantity, Sort: maxSort + 1, CreatedAt: now, UpdatedAt: now}, nil
}

// DetachBillQuotaLink 解挂一条定额引用（itemID 校验归属，防跨清单误删）。
func (s *Store) DetachBillQuotaLink(billItemID, linkID int64) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM gf_bill_quota_links WHERE id=? AND bill_item_id=?`, linkID, billItemID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("引用不存在或不属于该清单项")
	}
	return nil
}

// quotaLinksByItems 批量取一组清单项的定额码（BillItems 聚合用）：一次查询，
// 按 item 分组。单值 quota_code 非空但 link 表没挂时兜底补一条（旧数据兼容）。
func (s *Store) quotaLinksByItems(itemIDs []int64, fallback map[int64]string) map[int64][]string {
	out := make(map[int64][]string, len(itemIDs))
	for _, id := range itemIDs {
		out[id] = []string{}
	}
	if len(itemIDs) == 0 {
		return out
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(itemIDs)), ",")
	args := make([]interface{}, len(itemIDs))
	for i, id := range itemIDs {
		args[i] = id
	}
	rows, err := s.db.Query(`SELECT bill_item_id, quota_code FROM gf_bill_quota_links
WHERE bill_item_id IN (`+placeholders+`) ORDER BY bill_item_id, sort, id`, args...)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var itemID int64
			var code string
			if err := rows.Scan(&itemID, &code); err != nil || code == "" {
				continue
			}
			out[itemID] = append(out[itemID], code)
		}
	}
	// 兜底：单值列有码但 link 表缺失（V30 前的旧清单）。
	for _, id := range itemIDs {
		if len(out[id]) == 0 {
			if fb := fallback[id]; fb != "" {
				out[id] = []string{fb}
			}
		}
	}
	return out
}
