package workcost

// ── 企业定额自生成（11.6 并行池清欠；roadmap 造价愿景#5）──────────────
//
// 多项目结算反推企业真实工料机消耗，自动形成「企业定额」——广联达有公共
// 数据但没你的私有真相，自进化飞轮即私有壁垒。
//
// 数据面事实（本刀盘底 2026-10-15）：跨项目的含量分歧在库里已有形——导入链
// 统一四态（ResolveQuotaCode）把「同原码不同内容」拆成 原码-N 变体行，各带
// 自己的 Source（项目导入：<文件名>）；内容全等的复用同一条（真统一）。因此
// 聚类键=orig_code（-N 变体的溯源字段就是原码），簇内 ≥2 条含量相异且来自
// ≥2 个项目源的变体，就是飞轮原料。
//
// 聚合口径：逐资源取各变体含量的**中位数**（对离群稳健）；覆盖度/极差如实
// 显形不掩盖分歧。宁缺勿误：标题+单位不全等的簇不成立、单源簇不成立、含量
// 全等簇不成立（没有可聚合的分歧）。企业定额落 ENT-<原码>（自有命名空间，
// 重生成=飞轮进化，只覆写 ENT-* 不碰项目数据）；预览/应用双段式，应用=作者
// 显式点名（办公记忆写入确认红线同款）。

import (
	"fmt"
	"sort"
	"strings"
)

// QuotaFlywheelPreview 扫描全量定额（含含量行）→ 飞轮候选（只读）。
func (s *Store) QuotaFlywheelPreview() (QuotaFlywheelPreview, error) {
	if err := s.requireDB(); err != nil {
		return QuotaFlywheelPreview{}, err
	}
	quotas, err := s.allQuotasWithItems()
	if err != nil {
		return QuotaFlywheelPreview{}, err
	}
	return BuildQuotaFlywheelPreview(quotas), nil
}

// allQuotasWithItems 全量定额+含量行（两条查询内存拼装，避免逐定额 N+1）。
func (s *Store) allQuotasWithItems() ([]Quota, error) {
	rows, err := s.db.Query(`SELECT code, COALESCE(orig_code,''), title, COALESCE(unit,''), COALESCE(source,'')
FROM gf_quotas ORDER BY code`)
	if err != nil {
		return nil, err
	}
	byCode := map[string]*Quota{}
	var order []string
	for rows.Next() {
		var code, orig, title, unit, source string
		if e := rows.Scan(&code, &orig, &title, &unit, &source); e != nil {
			continue
		}
		byCode[code] = &Quota{Code: code, OrigCode: orig, Title: title, Unit: unit, Source: source}
		order = append(order, code)
	}
	if e := rows.Err(); e != nil {
		_ = rows.Close()
		return nil, e
	}
	_ = rows.Close()

	irows, err := s.db.Query(`SELECT quota_code, COALESCE(resource_code,''), COALESCE(kind,''), COALESCE(title,''),
COALESCE(spec,''), COALESCE(unit,''), quantity
FROM gf_quota_items ORDER BY quota_code, sort, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = irows.Close() }()
	for irows.Next() {
		var qc string
		var it QuotaItem
		if e := irows.Scan(&qc, &it.ResourceCode, &it.Kind, &it.Title, &it.Spec, &it.Unit, &it.Quantity); e != nil {
			continue
		}
		if q, ok := byCode[qc]; ok {
			q.Items = append(q.Items, it)
		}
	}
	out := make([]Quota, 0, len(order))
	for _, code := range order {
		out = append(out, *byCode[code])
	}
	return out, irows.Err()
}

// QuotaFlywheelApply 作者点名聚类词干（orig_code 词干）后落库：重建候选（与
// Preview 同一纯函数，天然一致）→ SaveQuota 按 code upsert 整组替换。ENT-* 是
// 自有命名空间，重生成=飞轮进化，绝不覆写项目导入/全局定额。
func (s *Store) QuotaFlywheelApply(origCodes []string) (*QuotaFlywheelResult, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	quotas, err := s.allQuotasWithItems()
	if err != nil {
		return nil, err
	}
	prev := BuildQuotaFlywheelPreview(quotas)
	res := &QuotaFlywheelResult{Saved: []string{}, Skipped: []QuotaFlywheelSkip{}}
	want := map[string]bool{}
	for _, c := range origCodes {
		if k := strings.TrimSpace(c); k != "" {
			want[k] = true
		}
	}
	for _, cand := range prev.Candidates {
		if !want[cand.OrigCode] {
			continue
		}
		delete(want, cand.OrigCode)
		if _, err := s.SaveQuota(BuildEnterpriseQuota(cand), nil); err != nil {
			res.Skipped = append(res.Skipped, QuotaFlywheelSkip{
				Key: cand.OrigCode, Title: cand.Title, Reason: fmt.Sprintf("落库失败: %v", err)})
			continue
		}
		res.Saved = append(res.Saved, cand.NewCode)
	}
	for k := range want {
		res.Skipped = append(res.Skipped, QuotaFlywheelSkip{Key: k, Reason: "不是有效候选（预览不存在或未成立，宁缺勿误）"})
	}
	return res, nil
}

// QuotaFlywheelMember 聚类成员（同工作内容的一个定额变体）。
type QuotaFlywheelMember struct {
	Code      string `json:"code"`
	OrigCode  string `json:"origCode"`
	Source    string `json:"source"`
	ItemCount int    `json:"itemCount"`
}

// QuotaFlywheelRow 候选企业定额的一条聚合含量行。
type QuotaFlywheelRow struct {
	ResourceCode string  `json:"resourceCode"`
	Kind         string  `json:"kind"`
	Title        string  `json:"title"`
	Spec         string  `json:"spec,omitempty"`
	Unit         string  `json:"unit,omitempty"`
	Quantity     float64 `json:"quantity"` // 各变体含量的中位数
	Coverage     int     `json:"coverage"` // 出现在几个变体（≤成员数）
	Min          float64 `json:"min"`      // 分歧幅度如实显形
	Max          float64 `json:"max"`
	// Majority 覆盖过半（出现在多于半数变体）——false 时作者应留神该行
	// 可能是个别项目的特有工艺。
	Majority bool `json:"majority"`
}

// QuotaFlywheelCandidate 一个工作内容的企业定额候选。
type QuotaFlywheelCandidate struct {
	OrigCode string `json:"origCode"`
	NewCode  string `json:"newCode"` // ENT-<orig_code>
	Title    string `json:"title"`
	Unit     string `json:"unit"`
	// Members 成员变体（按 code 稳定序）。
	Members []QuotaFlywheelMember `json:"members"`
	Rows    []QuotaFlywheelRow    `json:"rows"`
	// Sources 参与聚合的不同项目源数。
	Sources int `json:"sources"`
}

// QuotaFlywheelSkip 一条不成立的簇（人可读原因）。
type QuotaFlywheelSkip struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

// QuotaFlywheelPreview 报告：扫到的簇数 + 候选/不成立分区。
type QuotaFlywheelPreview struct {
	Clusters   int                      `json:"clusters"`
	Candidates []QuotaFlywheelCandidate `json:"candidates"`
	Skipped    []QuotaFlywheelSkip      `json:"skipped"`
}

// QuotaFlywheelResult 应用回执。
type QuotaFlywheelResult struct {
	Saved   []string            `json:"saved"`
	Skipped []QuotaFlywheelSkip `json:"skipped"`
}

// quotaClusterKey 聚类键。导入链的 -N 消歧变体 orig_code 一律保留原码
// （ResolveQuotaCode 溯源语义），所以键=orig_code 本身，**不做任何文本剥尾**
// ——原码自身的合法数字尾（WP-100）与消歧尾（WP-100-2）形状相同，靠文本
// 区分必然误并（首版 quotaStem 剥尾被测试当场抓获）。orig_code 空的定额
// 退回自身 code，天然单例成不了簇。
func quotaClusterKey(q Quota) string {
	if oc := strings.TrimSpace(q.OrigCode); oc != "" {
		return oc
	}
	return strings.TrimSpace(q.Code)
}

// quotaFingerprint 含量指纹：逐行 资源码|数量 六位舍入拼接（行序无关）。
func quotaFingerprint(items []QuotaItem) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%s|%.6f", it.ResourceCode, it.Quantity))
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

// medianFloat 中位数（偶数取两中平均）。
func medianFloat(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// BuildQuotaFlywheelPreview 纯函数：全量定额（含含量行）→ 飞轮候选。
// 聚类键=orig_code（消歧变体的溯源字段，见 quotaClusterKey）；成员要求
// Source 非空（无溯源的定额不参与聚合——来源不明的东西不该变成企业标准）。
func BuildQuotaFlywheelPreview(quotas []Quota) QuotaFlywheelPreview {
	type cluster struct {
		stem    string
		members []Quota
	}
	clusters := map[string]*cluster{}
	var order []string
	for _, q := range quotas {
		if strings.TrimSpace(q.Source) == "" || strings.HasPrefix(q.Code, "ENT-") {
			continue
		}
		stem := quotaClusterKey(q)
		if stem == "" {
			continue
		}
		c, ok := clusters[stem]
		if !ok {
			c = &cluster{stem: stem}
			clusters[stem] = c
			order = append(order, stem)
		}
		c.members = append(c.members, q)
	}
	sort.Strings(order)

	out := QuotaFlywheelPreview{Clusters: len(clusters), Candidates: []QuotaFlywheelCandidate{}, Skipped: []QuotaFlywheelSkip{}}
	for _, stem := range order {
		c := clusters[stem]
		ms := c.members
		titleOf := func() string {
			for _, m := range ms {
				if strings.TrimSpace(m.Title) != "" {
					return m.Title
				}
			}
			return stem
		}
		fail := func(reason string) {
			out.Skipped = append(out.Skipped, QuotaFlywheelSkip{Key: stem, Title: titleOf(), Reason: reason})
		}
		// 带含量行的成员才有聚合价值。
		var withItems []Quota
		for _, m := range ms {
			if len(m.Items) > 0 {
				withItems = append(withItems, m)
			}
		}
		if len(withItems) < 2 {
			fail(fmt.Sprintf("只有 %d 个变体带含量行（共 %d 条定额），无可聚合的跨项目分歧", len(withItems), len(ms)))
			continue
		}
		// 标题+单位全等 ≥2 才算同一工作内容（防自然尾码 -N 误并）。
		core := map[string]int{}
		coreKey := ""
		for _, m := range withItems {
			k := strings.TrimSpace(m.Title) + "\x00" + strings.TrimSpace(m.Unit)
			core[k]++
			if core[k] >= 2 {
				coreKey = k
			}
		}
		if coreKey == "" {
			fail("没有 ≥2 条同名同单位的变体（标题或单位互异，宁缺勿误）")
			continue
		}
		var members []Quota
		for _, m := range withItems {
			if strings.TrimSpace(m.Title)+"\x00"+strings.TrimSpace(m.Unit) == coreKey {
				members = append(members, m)
			}
		}
		// 跨项目：源去重 ≥2（同一文件重导入产生的多行不算跨项目）。
		srcSet := map[string]bool{}
		for _, m := range members {
			srcSet[m.Source] = true
		}
		if len(srcSet) < 2 {
			fail(fmt.Sprintf("%d 条变体全部来自同一来源（%s），不是跨项目分歧", len(members), members[0].Source))
			continue
		}
		// 含量相异 ≥2：全等簇没有可聚合的东西（统一四态本该已并）。
		fp := map[string]bool{}
		for _, m := range members {
			fp[quotaFingerprint(m.Items)] = true
		}
		if len(fp) < 2 {
			fail("各变体含量完全一致（没有分歧可聚合；如重复出现可走编码统一收编）")
			continue
		}

		sort.Slice(members, func(i, j int) bool { return members[i].Code < members[j].Code })
		cand := QuotaFlywheelCandidate{
			OrigCode: stem,
			NewCode:  "ENT-" + stem,
			Title:    strings.TrimSpace(members[0].Title),
			Unit:     strings.TrimSpace(members[0].Unit),
			Sources:  len(srcSet),
		}
		for _, m := range members {
			cand.Members = append(cand.Members, QuotaFlywheelMember{
				Code: m.Code, OrigCode: m.OrigCode, Source: m.Source, ItemCount: len(m.Items),
			})
		}
		// 逐资源聚合：数量取中位数，覆盖度/极差如实显形。
		per := map[string][]float64{}
		meta := map[string]QuotaItem{}
		for _, m := range members {
			seen := map[string]bool{}
			for _, it := range m.Items {
				k := it.ResourceCode
				if seen[k] {
					continue
				}
				seen[k] = true
				per[k] = append(per[k], it.Quantity)
				if _, ok := meta[k]; !ok {
					meta[k] = it
				}
			}
		}
		keys := make([]string, 0, len(per))
		for k := range per {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			qs := per[k]
			mt := meta[k]
			row := QuotaFlywheelRow{
				ResourceCode: k,
				Kind:         mt.Kind,
				Title:        mt.Title,
				Spec:         mt.Spec,
				Unit:         mt.Unit,
				Quantity:     medianFloat(qs),
				Coverage:     len(qs),
			}
			row.Min, row.Max = qs[0], qs[0]
			for _, v := range qs {
				if v < row.Min {
					row.Min = v
				}
				if v > row.Max {
					row.Max = v
				}
			}
			// 严格过半才算多数派（出现在多于半数变体）。
			row.Majority = len(qs)*2 > len(members)
			cand.Rows = append(cand.Rows, row)
		}
		out.Candidates = append(out.Candidates, cand)
	}
	return out
}

// BuildEnterpriseQuota 由候选构造落库定额（含量行=中位数，ResourcePrice 留 0
// 按现行价核算——导入链同款口径）。
func BuildEnterpriseQuota(cand QuotaFlywheelCandidate) Quota {
	q := Quota{
		Code:     cand.NewCode,
		OrigCode: cand.OrigCode,
		Title:    cand.Title,
		Unit:     cand.Unit,
		Source:   fmt.Sprintf("企业聚合：%d 项目", cand.Sources),
		Note: fmt.Sprintf("自 %d 个项目变体聚合（%s）；含量=各变体中位数，分歧幅度见预览",
			len(cand.Members), strings.Join(memberCodes(cand), " ")),
		Items: []QuotaItem{},
	}
	if cat := enterpriseCategory(cand); cat != "" {
		q.CategoryPath = cat
	}
	for _, r := range cand.Rows {
		q.Items = append(q.Items, QuotaItem{
			ResourceCode: r.ResourceCode,
			Kind:         r.Kind,
			Title:        r.Title,
			Spec:         r.Spec,
			Unit:         r.Unit,
			Quantity:     r.Quantity,
		})
	}
	return q
}

func memberCodes(cand QuotaFlywheelCandidate) []string {
	cs := make([]string, 0, len(cand.Members))
	for _, m := range cand.Members {
		cs = append(cs, m.Code)
	}
	return cs
}

// enterpriseCategory 企业定额归「企业定额」分类路径（库树可辨识）。
func enterpriseCategory(cand QuotaFlywheelCandidate) string {
	if strings.TrimSpace(cand.Title) == "" {
		return ""
	}
	return "企业定额/" + cand.Title
}
