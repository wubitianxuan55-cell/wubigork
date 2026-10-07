package workcost

// projectapply.go —— 把解析出的项目 bundle 落库（资源库 + 消耗定额库）。
//
// 这是「最近项目录入」的执行端：一份五表产物 → 工料机资源 + 每清单项一条消耗
// 定额（定额子目 = 清单项/工序）。**只落工料法前两层**；综合单价是核算输出，
// 由 ComposeQuota 现算，不落库（避免把派生值当输入存下来又和资源价漂移）。
//
// 幂等：资源按 (kind,title,spec,unit) 身份 UPSERT；定额按 code UPSERT。
// 重复导入同一份文件不会制造重复数据。

import (
	"fmt"
	"strings"
)

// ApplyProjectResult 一次项目落库的结果。
type ApplyProjectResult struct {
	Project     string   `json:"project"`
	ResourceNew int      `json:"resourceNew"`
	ResourceUpd int      `json:"resourceUpd"`
	QuotaNew    int      `json:"quotaNew"`
	QuotaUpd    int      `json:"quotaUpd"`
	Lines       int      `json:"lines"`
	Errors      []string `json:"errors"`
}

// QuotaCodeForItem 清单项编码 → 定额编码。
// 模版里清单编码本身已是稳定工序码（WP01/SF02/BJ03），直接用；缺失时按标题
// 兜底生成（保证每条清单项都有可引用锚点）。
func QuotaCodeForItem(it ProjectItem) string {
	if c := strings.TrimSpace(it.Code); c != "" {
		return c
	}
	if slug := SlugFromTitle(it.Title); slug != "" {
		return "AUTO-" + slug
	}
	return ""
}

// SlugFromTitle 标题 → 稳定 slug（仅保留 ASCII 字母数字与汉字，压缩连字符）。
func SlugFromTitle(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= 0x4e00 && r <= 0x9fff:
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
		if b.Len() >= 48 {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}

// ApplyProjectBundle 把项目 bundle 落库：先资源、后定额（定额引用资源编码）。
//
// 资源清单以「工料机价格表」为准；若消耗行引用了资源表里没有的编码，该资源
// 会以零价补建并记入 Errors——不静默丢弃，因为丢一条资源等于让某条工序的
// 综合单价偏低而无人知晓。
func (s *Store) ApplyProjectBundle(b *ProjectBundle) ApplyProjectResult {
	res := ApplyProjectResult{Project: b.Project, Lines: len(b.Lines)}
	if err := s.requireDB(); err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res
	}

	// ① 资源：文档编码优先保留（它是消耗量表 VLOOKUP 的锚点）。
	idx, err := s.ResourceIndex()
	if err != nil {
		res.Errors = append(res.Errors, "读取资源索引失败: "+err.Error())
		return res
	}
	byCode := map[string]Resource{}
	for _, pr := range b.Resources {
		r := Resource{
			Code:         pr.Code,
			Kind:         pr.Kind,
			Title:        pr.Title,
			Spec:         pr.Spec,
			Unit:         pr.Unit,
			BasePrice:    pr.Price,
			CurrentPrice: pr.Price,
			Source:       pr.Source,
			Note:         pr.Note,
			Status:       "现行",
		}
		if r.Code == "" {
			res.Errors = append(res.Errors, fmt.Sprintf("资源「%s」无编码，已跳过", pr.Title))
			continue
		}
		if _, existed := idx[r.Code]; existed {
			res.ResourceUpd++
		} else {
			res.ResourceNew++
		}
		saved, err := s.SaveResource(r)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("资源 %s（%s）落库失败: %v", r.Code, r.Title, err))
			continue
		}
		byCode[saved.Code] = *saved
		idx[saved.Code] = *saved
	}

	// ② 消耗行按清单项分组。
	grouped := map[string][]QuotaItem{}
	order := []string{}
	itemTitle := map[string]string{}
	itemUnit := map[string]string{}
	for _, l := range b.Lines {
		if _, seen := grouped[l.ItemCode]; !seen {
			order = append(order, l.ItemCode)
		}
		it := QuotaItem{
			ResourceCode: l.ResourceCode,
			Kind:         l.Kind,
			Title:        l.ResourceTitle,
			Unit:         l.Unit,
			Quantity:     l.Quantity,
			// ResourcePrice 刻意留 0：核算时按资源**现行价**取值，避免把文档里
			// 的旧价当快照钉死（快照只在资源缺失时兜底）。
			ResourcePrice: 0,
		}
		if r, ok := idx[l.ResourceCode]; ok {
			it.Kind = r.Kind
			it.Title = r.Title
			it.Unit = r.Unit
		} else {
			res.Errors = append(res.Errors,
				fmt.Sprintf("消耗行引用了未定义的资源编码 %s（清单项 %s），按零价核算", l.ResourceCode, l.ItemCode))
		}
		grouped[l.ItemCode] = append(grouped[l.ItemCode], it)
		itemTitle[l.ItemCode] = l.ItemTitle
		itemUnit[l.ItemCode] = l.ItemUnit
	}

	// ③ 定额：以解析出的清单项为骨架（保证即使无消耗行的项也留下痕迹），
	// 再并入只在消耗行里出现的编码。
	items := append([]ProjectItem(nil), b.Items...)
	known := map[string]bool{}
	for _, it := range items {
		known[it.Code] = true
	}
	for _, code := range order {
		if !known[code] {
			items = append(items, ProjectItem{Code: code, Title: itemTitle[code], Unit: itemUnit[code]})
			known[code] = true
		}
	}

	for _, it := range items {
		code := QuotaCodeForItem(it)
		if code == "" {
			res.Errors = append(res.Errors, fmt.Sprintf("清单项「%s」无法生成定额编码，已跳过", it.Title))
			continue
		}
		title := strings.TrimSpace(it.Title)
		if title == "" {
			title = itemTitle[it.Code]
		}
		unit := strings.TrimSpace(it.Unit)
		if unit == "" {
			unit = itemUnit[it.Code]
		}
		q := Quota{
			Code:         code,
			Title:        title,
			Specialty:    "土壤修复",
			Unit:         unit,
			CategoryPath: firstNonEmpty(it.CategoryPath, "综合单价/"+title),
			Source:       fmt.Sprintf("项目导入：%s", b.FileName),
			Note:         it.Feature,
			Items:        grouped[it.Code],
		}
		existed, err := s.quotaExists(code)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("查询定额 %s 是否存在失败: %v", code, err))
		} else if existed {
			res.QuotaUpd++
		} else {
			res.QuotaNew++
		}
		if _, err := s.SaveQuota(q, idx); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("定额 %s（%s）落库失败: %v", code, title, err))
		}
	}
	return res
}

// quotaExists 判断定额编码是否已存在。
func (s *Store) quotaExists(code string) (bool, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM gf_quotas WHERE code=?`, code).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}
