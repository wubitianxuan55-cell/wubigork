package workcost

// regionfill.go — 定额地区回填。定额的地区与上游清单项目挂钩，但挂钩只在项目导入时
// 写进新建的定额；导入前就在库里的定额地区一直空着（2026-11-28 盘底：340 条定额地区
// 全空，定额级的地区筛选因此无从筛起）。
//
// 这里按「引用它的清单行属于哪个省的项目」补：只被一个省的项目引用就写上；跨省引用、
// 项目没标地区、没被清单引用的不动并说明原因——地区是口径，猜不得。先预览（dryRun）
// 给人看，确认后再落库。

import (
	"fmt"
	"strings"
)

// QuotaRegionFix 回填方案里的一行定额。
type QuotaRegionFix struct {
	Code    string `json:"code"`
	Title   string `json:"title"`
	Region  string `json:"region,omitempty"` // 要写入的省名（不回填时为空）
	Links   int    `json:"links"`            // 引用它的清单行数
	Reason  string `json:"reason,omitempty"` // 不回填的原因
	Applied bool   `json:"applied"`          // 本次已写入（预览时恒 false）
}

// BackfillQuotaRegions 按上游清单项目回填定额地区，见文件头。返回全部没标地区的定额
// （按编码排序），构造即非 nil。
func (s *Store) BackfillQuotaRegions(dryRun bool) ([]QuotaRegionFix, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	// 引用关系有两处来源：清单行的单值主定额列，和 1:N 挂接表。
	rows, err := s.db.Query(`
WITH usage AS (
  SELECT b.project_id, b.quota_code AS quota_code FROM gf_bill_items b WHERE COALESCE(b.quota_code,'')<>''
  UNION
  SELECT b.project_id, l.quota_code FROM gf_bill_quota_links l JOIN gf_bill_items b ON b.id=l.bill_item_id
)
SELECT q.code, q.title, COUNT(u.project_id),
  GROUP_CONCAT(DISTINCT NULLIF(p.region,'')),
  COUNT(DISTINCT NULLIF(p.region,''))
FROM gf_quotas q
LEFT JOIN usage u ON u.quota_code=q.code
LEFT JOIN gf_projects p ON p.id=u.project_id
WHERE COALESCE(q.region,'')=''
GROUP BY q.code, q.title ORDER BY q.code`)
	if err != nil {
		return nil, err
	}
	type pending struct {
		fix      QuotaRegionFix
		regions  string
		nRegions int
	}
	var all []pending
	for rows.Next() {
		var p pending
		var regions *string
		if err := rows.Scan(&p.fix.Code, &p.fix.Title, &p.fix.Links, &regions, &p.nRegions); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if regions != nil {
			p.regions = *regions
		}
		all = append(all, p)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	out := make([]QuotaRegionFix, 0, len(all))
	now := nowText()
	for _, p := range all {
		fix := p.fix
		switch {
		case fix.Links == 0:
			fix.Reason = "没有被清单引用，没有上游项目可依据"
		case p.nRegions == 0:
			fix.Reason = "引用它的清单项目没标地区"
		case p.nRegions > 1:
			fix.Reason = fmt.Sprintf("被 %d 个省的项目引用（%s），不猜", p.nRegions, strings.ReplaceAll(p.regions, ",", "、"))
		}
		if fix.Reason == "" {
			region, ok := NormalizeRegion(p.regions)
			if !ok {
				fix.Reason = fmt.Sprintf("项目地区「%s」不是省级行政区", p.regions)
			} else if has, err := s.DictHas(DictRegion, region); err != nil || !has {
				fix.Reason = fmt.Sprintf("项目地区「%s」不在地区字典里", region)
			} else {
				fix.Region = region
				if !dryRun {
					// 条件里再钉一次「地区仍为空」：预览到落库之间被人手工标了的不覆盖。
					if _, err := s.db.Exec(`UPDATE gf_quotas SET region=?, updated_at=? WHERE code=? AND COALESCE(region,'')=''`,
						region, now, fix.Code); err != nil {
						return nil, fmt.Errorf("回填定额 %s 地区失败: %w", fix.Code, err)
					}
					fix.Applied = true
				}
			}
		}
		out = append(out, fix)
	}
	return out, nil
}
