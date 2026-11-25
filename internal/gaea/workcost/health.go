package workcost

// ── 数据健康扫描（四期 4.1，master plan 原案）────────────────
//
// 六类扫描（各有独立测试；每条发现带人可读详情，前端一键跳转修复）：
//   1. zero_price      零价资源——现行价与基准价皆为 0（无价可核算）
//   2. quota_missing_price 缺价定额子目——含量行 resource_price ≤ 0（综合
//      单价按 0 计，算出来必错）
//   3. broken_reference 引用缺失——含量行 resource_code 不在资源表（悬挂引用，
//      核算直接跳过该子目）
//   4. duplicate       重复资源防护检查——身份唯一索引（idx_gf_resources_ident，
//      V25）在库层面已消灭同身份多行；本项检查**索引本身仍在**（防未来迁移
//      误删唯一约束使防护失效）。跨身份的语义性重复（同物异规格/编码分裂）
//      由 ⑤ 检出、由 DedupeResources（作者显式裁决的跨身份合并）处置
//   5. code_mismatch   编码未统一——同 title 同地区的资源编码风格混用（助记码 vs
//      前缀序号），提示作者归一
//   6. rule_issues     不合规矩的数据（2026-11-25 库定规矩）——取值在字典外、类别
//      不在四类内、期数不是 YYYY-MM。多为升级前的存量，处置走 cost_library
//      classify（改数据）或 dict（加字典项）
//
// 全部为只读查询（零写零事务），可在概览页随开随扫。

import (
	"fmt"
)

// HealthReport 六类扫描报告（wire camelCase）。
type HealthReport struct {
	ZeroPrice     []HealthFinding     `json:"zeroPrice"`
	QuotaNoPrice  []HealthFinding     `json:"quotaNoPrice"`
	BrokenRef     []HealthFinding     `json:"brokenRef"`
	Duplicates    []DuplicateGroup    `json:"duplicates"`
	CodeMismatch  []CodeMismatchGroup `json:"codeMismatch"`
	RuleIssues    []RuleIssue         `json:"ruleIssues"`
	TotalFindings int                 `json:"totalFindings"`
}

// HealthFinding 单条发现（Resource/Quota 两用：Code 为定额码时 Kind 空）。
type HealthFinding struct {
	ID    int64  `json:"id"`
	Code  string `json:"code"` // 资源 code / 定额 code
	Title string `json:"title"`
	Kind  string `json:"kind,omitempty"`
	Unit  string `json:"unit,omitempty"`
	// Detail 人可读补充（如「现行价与基准价均为 0」）。
	Detail string `json:"detail,omitempty"`
}

// RuleIssue 一条不合规矩的数据：某个字段的取值不在库认可的范围内。
type RuleIssue struct {
	Tier  string `json:"tier"` // resource / quota / bill / project
	ID    int64  `json:"id"`
	Code  string `json:"code"` // 资源码 / 定额码 / 清单码（项目为空）
	Title string `json:"title"`
	Field string `json:"field"` // kind / region / category_path / specialty / price_date / price_period
	Value string `json:"value"` // 不合规的原值
	// Fix 处置指引。
	Fix string `json:"fix"`
}

// DuplicateGroup 重复防护发现（索引缺失告警形态；正常态为空）。
type DuplicateGroup struct {
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

// CodeMismatchGroup 同 title 编码风格混用组。
type CodeMismatchGroup struct {
	Title string          `json:"title"`
	Items []HealthFinding `json:"items"`
}

// HealthScan 六类扫描（只读）。
func (s *Store) HealthScan() (*HealthReport, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	rep := &HealthReport{}

	// ① 零价资源
	rows, err := s.db.Query(`
SELECT id, code, kind, title, unit FROM gf_resources
WHERE current_price = 0 AND base_price = 0 ORDER BY kind, title`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		f := HealthFinding{Detail: "现行价与基准价均为 0——核算取用价无价可用"}
		if err := rows.Scan(&f.ID, &f.Code, &f.Kind, &f.Title, &f.Unit); err == nil {
			rep.ZeroPrice = append(rep.ZeroPrice, f)
		}
	}
	_ = rows.Close()

	// ② 缺价定额子目（含量行单价 ≤0）
	rows, err = s.db.Query(`
SELECT qi.quota_code, qi.title, qi.unit FROM gf_quota_items qi
WHERE qi.resource_price <= 0 ORDER BY qi.quota_code, qi.title`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		f := HealthFinding{Detail: "含量行单价为 0——综合单价按 0 计入"}
		if err := rows.Scan(&f.Code, &f.Title, &f.Unit); err == nil {
			rep.QuotaNoPrice = append(rep.QuotaNoPrice, f)
		}
	}
	_ = rows.Close()

	// ③ 引用缺失（悬挂 resource_code）
	rows, err = s.db.Query(`
SELECT DISTINCT qi.resource_code, qi.title FROM gf_quota_items qi
LEFT JOIN gf_resources r ON r.code = qi.resource_code
WHERE qi.resource_code != '' AND r.id IS NULL ORDER BY qi.resource_code`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		f := HealthFinding{Detail: "引用的资源编码不在资源表中（悬挂引用）"}
		if err := rows.Scan(&f.Code, &f.Title); err == nil {
			rep.BrokenRef = append(rep.BrokenRef, f)
		}
	}
	_ = rows.Close()

	// ④ 重复资源防护：身份唯一索引在位性（V25 起库层面消灭同身份多行）
	var identCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_gf_resources_ident'`).Scan(&identCount); err != nil {
		return nil, err
	}
	if identCount == 0 {
		rep.Duplicates = append(rep.Duplicates, DuplicateGroup{
			Kind: "⚠", Title: "身份唯一索引缺失",
			Detail: "idx_gf_resources_ident 不在——同身份多行防护失效，重复入库没有语义（请核查迁移链）",
		})
	}

	// ⑤ 编码未统一（同 title **同地区**的编码风格混用：助记码 vs 前缀序号）。
	// 工料机按地区分行后，同名的各地行各用各的编码体系是正常的，跨地区不比。
	rows, err = s.db.Query(`SELECT DISTINCT title, COALESCE(region,'') FROM gf_resources ORDER BY title, region`)
	if err != nil {
		return nil, err
	}
	type titleKey struct{ title, region string }
	var keys []titleKey
	for rows.Next() {
		var k titleKey
		if rows.Scan(&k.title, &k.region) == nil {
			keys = append(keys, k)
		}
	}
	_ = rows.Close()
	for _, k := range keys {
		rows2, err := s.db.Query(`SELECT id, code, kind FROM gf_resources WHERE title=? AND region=? ORDER BY id`, k.title, k.region)
		if err != nil {
			return nil, err
		}
		var items []HealthFinding
		styles := map[string]bool{}
		for rows2.Next() {
			f := HealthFinding{Title: k.title}
			if err := rows2.Scan(&f.ID, &f.Code, &f.Kind); err == nil {
				styles[codeStyle(f.Code)] = true
				items = append(items, f)
			}
		}
		_ = rows2.Close()
		if len(styles) > 1 {
			if k.region != "" {
				for i := range items {
					items[i].Detail = "地区 " + k.region
				}
			}
			rep.CodeMismatch = append(rep.CodeMismatch, CodeMismatchGroup{Title: k.title, Items: items})
		}
	}

	// ⑥ 不合规矩的数据（库定规矩，见 rules.go）：取值在字典外 / 类别不在四类内 /
	// 期数不是 YYYY-MM。多为升级前的存量。
	rep.RuleIssues, err = s.scanRuleIssues()
	if err != nil {
		return nil, err
	}

	rep.TotalFindings = len(rep.ZeroPrice) + len(rep.QuotaNoPrice) + len(rep.BrokenRef) +
		len(rep.Duplicates) + len(rep.CodeMismatch) + len(rep.RuleIssues)
	return rep, nil
}

// scanRuleIssues 扫出取值不在库认可范围内的数据（只读）。
func (s *Store) scanRuleIssues() ([]RuleIssue, error) {
	// GLOB 没有分组与多选，YYYY-MM 的月份段拆成两条模式。
	const (
		canonA = "[0-9][0-9][0-9][0-9]-0[1-9]"
		canonB = "[0-9][0-9][0-9][0-9]-1[0-2]"
	)
	notPeriod := func(col string) string {
		return col + " <> '' AND " + col + " NOT GLOB '" + canonA + "' AND " + col + " NOT GLOB '" + canonB + "'"
	}
	rows, err := s.db.Query(`
SELECT 'resource', r.id, r.code, r.title, 'kind', r.kind, '改成 人工/材料/机械/外委 之一（cost_library classify）'
FROM gf_resources r WHERE r.kind NOT IN ('人工','材料','机械','外委')
UNION ALL
SELECT 'resource', r.id, r.code, r.title, 'region', r.region, '地区不在字典：加进字典（dict add）或改成字典里的地区（classify）'
FROM gf_resources r LEFT JOIN gf_dicts d ON d.kind='region' AND d.name=r.region
WHERE r.region<>'' AND d.id IS NULL
UNION ALL
SELECT 'resource', r.id, r.code, r.title, 'category_path', r.category_path, '分类不在字典：加进字典（dict add）或改成字典里的分类（classify）'
FROM gf_resources r LEFT JOIN gf_dicts d ON d.kind='category' AND d.name=r.category_path
WHERE r.category_path<>'' AND d.id IS NULL
UNION ALL
SELECT 'resource', r.id, r.code, r.title, 'price_date', r.price_date, '期数不是 YYYY-MM：换算后重设（classify 的 period）'
FROM gf_resources r WHERE ` + notPeriod("r.price_date") + `
UNION ALL
SELECT 'quota', q.id, q.code, q.title, 'specialty', q.specialty, '专业不在字典：加进字典（dict add）或改成字典里的专业（classify）'
FROM gf_quotas q LEFT JOIN gf_dicts d ON d.kind='specialty' AND d.name=q.specialty
WHERE q.specialty<>'' AND d.id IS NULL
UNION ALL
SELECT 'quota', q.id, q.code, q.title, 'price_date', q.price_date, '期数不是 YYYY-MM'
FROM gf_quotas q WHERE ` + notPeriod("q.price_date") + `
UNION ALL
SELECT 'bill', b.id, b.code, b.title, 'specialty', b.specialty, '专业不在字典'
FROM gf_bill_items b LEFT JOIN gf_dicts d ON d.kind='specialty' AND d.name=b.specialty
WHERE b.specialty<>'' AND d.id IS NULL
UNION ALL
SELECT 'project', p.id, '', p.name, 'specialty', p.specialty, '专业不在字典'
FROM gf_projects p LEFT JOIN gf_dicts d ON d.kind='specialty' AND d.name=p.specialty
WHERE p.specialty<>'' AND d.id IS NULL
UNION ALL
SELECT 'project', p.id, '', p.name, 'region', p.region, '地区不在字典'
FROM gf_projects p LEFT JOIN gf_dicts d ON d.kind='region' AND d.name=p.region
WHERE p.region<>'' AND d.id IS NULL
UNION ALL
SELECT 'project', p.id, '', p.name, 'price_period', p.price_period, '期数不是 YYYY-MM'
FROM gf_projects p WHERE ` + notPeriod("p.price_period") + `
ORDER BY 1, 5, 3`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []RuleIssue{}
	for rows.Next() {
		var it RuleIssue
		if err := rows.Scan(&it.Tier, &it.ID, &it.Code, &it.Title, &it.Field, &it.Value, &it.Fix); err != nil {
			continue
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// codeStyle 编码风格：助记码（字母 ≥2 连续，如 EXC/HP300 开头段）vs 前缀序号
// （单字母+数字，如 M001/P012）。
func codeStyle(code string) string {
	letters := 0
	for _, r := range code {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			letters++
			continue
		}
		break
	}
	if letters >= 2 {
		return "mnemonic"
	}
	return "prefix"
}

// DedupeResult 去重回执（4.2）。
type DedupeResult struct {
	Kept             string `json:"kept"`             // 保留资源 code
	Merged           int    `json:"merged"`           // 合并掉的资源数
	RedirectedItems  int    `json:"redirectedItems"`  // 定额含量引用重定向行数
	RedirectedPrices int    `json:"redirectedPrices"` // 调价历史并入条数
}

// DedupeResources 资源去重：keep 保留，merges 合并掉——定额含量引用重定向到
// keep 的 code，调价历史并入 keep（resource_id 改写），删除 merged 条目。
// 整批一个事务：身份校验失败/任一步失败整体回滚。
func (s *Store) DedupeResources(keepID int64, mergeIDs []int64) (*DedupeResult, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	keep, err := s.GetResource(keepID)
	if err != nil {
		return nil, err
	}
	res := &DedupeResult{Kept: keep.Code}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range mergeIDs {
		if id == keepID {
			return nil, fmt.Errorf("merge 列表包含保留条目本身（id=%d）", id)
		}
		m, err := s.GetResource(id)
		if err != nil {
			return nil, err
		}
		// 身份允许不同（跨身份语义合并——「编码未统一」扫描的处置动作）：
		// 作者在健康页显式点名 keep/merge 即裁决，规格差异大的合并后作者可在
		// 资源编辑里改 keep 的规格。此处只做存在性校验。
		// 引用重定向：定额含量行 resource_code → keep.Code
		r1, err := tx.Exec(
			`UPDATE gf_quota_items SET resource_code=?, updated_at=datetime('now') WHERE resource_code=?`,
			keep.Code, m.Code)
		if err != nil {
			return nil, err
		}
		n1, _ := r1.RowsAffected()
		res.RedirectedItems += int(n1)
		// 调价历史并入 keep
		r2, err := tx.Exec(`UPDATE gf_resource_prices SET resource_id=? WHERE resource_id=?`, keepID, id)
		if err != nil {
			return nil, err
		}
		n2, _ := r2.RowsAffected()
		res.RedirectedPrices += int(n2)
		// 删除 merged 条目（引用已重定向，可直接删）
		if _, err := tx.Exec(`DELETE FROM gf_resources WHERE id=?`, id); err != nil {
			return nil, err
		}
		res.Merged++
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}
