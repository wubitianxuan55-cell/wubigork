package workcost

// projectapply.go —— 把解析出的项目 bundle 落库（资源库 + 消耗定额库 + 清单层）。
//
// 这是「最近项目录入」的执行端：一份五表产物 → 工料机资源 + 每清单项一条消耗
// 定额（定额子目 = 清单项/工序）+ 分部分项清单。综合单价是核算输出，由
// ComposeQuota 现算，不落库（避免把派生值当输入存下来又和资源价漂移）。
//
// 文件向库的规矩靠（2026-11-25 用户拍板「数据库要定规矩，不是比着几个文件去
// 适配」，规矩见 rules.go）：
//   - 项目专业/地区/计价期是调用方给的规范值，文件里没有就不猜（空=待整理）；
//     新定额的专业 = 项目专业——定额专业随来源定，跨专业不并成一条；
//   - 工料机按地区分行：同身份不同地区是两行，一个地区的价不顶掉另一个地区的；
//     每条有价的工料机记一条价格事实（期 + 地区 + 来源），同簿重导幂等；
//   - 文件里不合规矩的行（类别不在四类内）拒收并点名，不冒充合格数据。
//
// 幂等：资源按身份 UPSERT；定额按编码解析规则复用；清单按（项目,编码）更新。
// 重复导入同一份文件不会制造重复数据。

import (
	"fmt"
	"strings"
)

// ApplyProjectResult 一次项目落库的结果。
type ApplyProjectResult struct {
	Project     string `json:"project"`
	ResourceNew int    `json:"resourceNew"`
	ResourceUpd int    `json:"resourceUpd"`
	QuotaNew    int    `json:"quotaNew"`
	QuotaUpd    int    `json:"quotaUpd"`
	Lines       int    `json:"lines"`
	// ResourceRenamed 工料机编码被别的资源占用、消歧成「码-N」的条数（含量行已
	// 按别名落到消歧码上，不会悬空）。两份价格表各写各的 L01 时会发生。
	ResourceRenamed int `json:"resourceRenamed"`
	// QuotaDerived 跨专业另立（借鉴外专业同码定额）的定额条数。
	QuotaDerived int `json:"quotaDerived"`
	// PriceFacts 写入的价格事实条数。
	PriceFacts int      `json:"priceFacts"`
	Errors     []string `json:"errors"`
}

// truncateRunes 截到 n 个字符（回执里引用文件原文用，避免整段说明刷屏）。
func truncateRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
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

// ApplyProjectBundle 把项目 bundle 落库：先资源、后定额（定额引用资源编码）、
// 最后清单层。
//
// 资源清单以「工料机价格」为准；若消耗行引用了资源表里没有的编码，按零价
// 核算并记入 Errors——不静默丢弃，因为丢一条资源等于让某条工序的综合单价
// 偏低而无人知晓。
func (s *Store) ApplyProjectBundle(b *ProjectBundle) ApplyProjectResult {
	// Errors 构造即非 nil：nil 切片经 Wails JSON 序列化成 null，前端 res.errors.length 会崩。
	res := ApplyProjectResult{Project: b.Project, Lines: len(b.Lines), Errors: []string{}}
	if err := s.requireDB(); err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res
	}

	// ⓪ 导入口径过规矩（ImportScope）：不过规矩整份拒收——带病入库比不入更糟。
	specialty, region, period, err := s.ImportScope(b)
	if err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res
	}

	// ① 资源：文档编码优先保留（它是消耗量表 VLOOKUP 的锚点）。
	idx, err := s.ResourceIndex()
	if err != nil {
		res.Errors = append(res.Errors, "读取资源索引失败: "+err.Error())
		return res
	}
	// alias 工作簿编码 → 库内规范编码：同身份的既有行保留原码、撞码行消歧成「码-N」，
	// 本次的含量行一律按别名落到规范编码上（此前换码后含量行会悬空或错指）。
	alias := map[string]string{}
	canonical := func(code string) string {
		if c, ok := alias[code]; ok {
			return c
		}
		return code
	}
	for _, pr := range b.Resources {
		r := Resource{
			Code:         strings.TrimSpace(pr.Code),
			Kind:         normalizeKind(pr.Kind),
			Title:        strings.TrimSpace(pr.Title),
			Spec:         strings.TrimSpace(pr.Spec),
			Unit:         strings.TrimSpace(pr.Unit),
			BasePrice:    pr.Price,
			CurrentPrice: pr.Price,
			Region:       region,
			Source:       strings.TrimSpace(pr.Source),
			Note:         strings.TrimSpace(pr.Note),
			Status:       "现行",
		}
		if r.Code == "" && r.Title == "" {
			continue
		}
		if r.Code == "" {
			res.Errors = append(res.Errors, fmt.Sprintf("资源「%s」无编码，已跳过", r.Title))
			continue
		}
		if r.Title == "" {
			res.Errors = append(res.Errors, fmt.Sprintf("资源 %s 无名称，已跳过", r.Code))
			continue
		}
		// 编码得像编码：整格同一段说明文字的「行」不是工料机（规矩见 rules.go）。
		if ok, why := CodeWellFormed(r.Code); !ok {
			res.Errors = append(res.Errors, fmt.Sprintf("工料机价格表里有一行不是工料机（%s），已拒收——若是说明文字请挪出数据区或留空编码列：%s", why, truncateRunes(r.Code, 24)))
			continue
		}
		if !isKnownKind(r.Kind) {
			res.Errors = append(res.Errors, fmt.Sprintf("资源 %s（%s）的类别「%s」不在 人工/材料/机械/外委 之内，已拒收——把文件里这一行的类别改成四类之一再导入",
				r.Code, r.Title, pr.Kind))
			continue
		}
		prev, err := s.findResourceByIdentity(r)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("资源 %s（%s）身份回查失败: %v", r.Code, r.Title, err))
			continue
		}
		if prev != nil {
			// 同身份既有行：改价不改身份。这次导入没带的分类/来源/备注不被清空，
			// 价格读成 0（公式无缓存值）也不顶掉已有价。
			r.CategoryPath = prev.CategoryPath
			if r.Source == "" {
				r.Source = prev.Source
			}
			if r.Note == "" {
				r.Note = prev.Note
			}
			if pr.Price <= 0 {
				r.BasePrice, r.CurrentPrice, r.PriceDate = prev.BasePrice, prev.CurrentPrice, prev.PriceDate
			} else {
				r.PriceDate = period
			}
		} else if pr.Price > 0 {
			r.PriceDate = period
		}
		if prev == nil {
			// 新行但编码已被别的工料机占用：消歧成「码-N」，不靠 UNIQUE 约束报错了事。
			if _, taken := idx[r.Code]; taken {
				free, e := s.FreeResourceCode(r.Code)
				if e != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("资源编码 %s 消歧失败: %v", r.Code, e))
					continue
				}
				r.Code = free
				res.ResourceRenamed++
			}
		}
		saved, err := s.SaveResource(r)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("资源 %s（%s）落库失败: %v", pr.Code, r.Title, err))
			continue
		}
		alias[strings.TrimSpace(pr.Code)] = saved.Code
		if prev != nil {
			res.ResourceUpd++
		} else {
			res.ResourceNew++
		}
		if pr.Price > 0 && period != "" {
			if err := s.RecordPriceFact(saved.ID, pr.Price, period, saved.Region, "",
				"项目导入："+b.FileName, ""); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("资源 %s（%s）价格事实落库失败: %v", saved.Code, saved.Title, err))
			} else {
				res.PriceFacts++
			}
		}
		idx[saved.Code] = *saved
	}

	// ② 消耗行按清单项分组（资源码一律落到库内规范编码上）。
	grouped := map[string][]QuotaItem{}
	order := []string{}
	itemTitle := map[string]string{}
	itemUnit := map[string]string{}
	for _, l := range b.Lines {
		if _, seen := grouped[l.ItemCode]; !seen {
			order = append(order, l.ItemCode)
		}
		rc := canonical(strings.TrimSpace(l.ResourceCode))
		it := QuotaItem{
			ResourceCode: rc,
			Kind:         l.Kind,
			Title:        l.ResourceTitle,
			Unit:         l.Unit,
			Quantity:     l.Quantity,
			// ResourcePrice 刻意留 0：核算时按资源**现行价**取值，避免把文档里
			// 的旧价当快照钉死（快照只在资源缺失时兜底）。
			ResourcePrice: 0,
		}
		if r, ok := idx[rc]; ok {
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

	// 定额编码解析：原码 → 全局唯一定额编码。清单层的 quota_code 引用解析码。
	resolved := map[string]string{}
	src := fmt.Sprintf("项目导入：%s", b.FileName)
	for _, it := range items {
		origCode := QuotaCodeForItem(it)
		if origCode == "" {
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
		rr, err := s.ResolveQuota(origCode, title, unit, it.Feature, src, specialty)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("定额编码解析失败 %s（%s）: %v", origCode, title, err))
			continue
		}
		resolved[it.Code] = rr.Code
		if rr.Reuse {
			res.QuotaUpd++
		} else {
			res.QuotaNew++
		}
		if rr.Derived != "" {
			res.QuotaDerived++
		}
		var existing *Quota
		if rr.Reuse {
			var err error
			if existing, err = s.GetQuota(rr.Code); err != nil {
				existing = nil
			}
		}
		q := Quota{
			Code:         rr.Code,
			OrigCode:     origCode,
			Title:        title,
			Specialty:    specialty,
			Unit:         unit,
			CategoryPath: firstNonEmpty(it.CategoryPath, "综合单价/"+title),
			Source:       src,
			Note:         it.Feature,
			DerivedFrom:  rr.Derived,
			Items:        grouped[it.Code],
		}
		if existing != nil {
			// 专业随来源定：同项目重导入时项目专业改了，它的定额跟着改；尚未标注
			// 专业的既有定额被复用时顺手标上。其余情况专业不动（定额专业固定）。
			if specialty != "" && existing.Specialty != specialty &&
				(existing.Source == src || existing.Specialty == "") {
				if err := s.setQuotaSpecialty(rr.Code, specialty); err == nil {
					existing.Specialty = specialty
				}
			}
			// 既有定额的头字段不因这次导入没带而被清空。
			q.Specialty = firstNonEmpty(specialty, existing.Specialty)
			q.Chapter = existing.Chapter
			q.DerivedFrom = firstNonEmpty(rr.Derived, existing.DerivedFrom)
			if existing.OrigCode != "" {
				q.OrigCode = existing.OrigCode
			}
			switch {
			case !rr.SameSource && len(existing.Items) > 0:
				// 同专业同内容复用（别的项目）：既有定额是本专业的标准，已有
				// 消耗行**保持不动**（不因新项目的同名项覆盖标准）；空定额则
				// 采用本项目的消耗行补齐。
				continue
			case len(q.Items) == 0:
				// 同项目重导入但本次工作簿没给消耗行：保留既有消耗行，不清空。
				q.Items = existing.Items
			}
		}
		if _, err := s.SaveQuota(q, idx); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("定额 %s（%s）落库失败: %v", rr.Code, title, err))
		}
	}

	// ④ 清单层落库（SchemaV28）：分部分项清单（编码/名称/单位/**工程量**）
	// 是工程实体的一等数据——此前只活在 xlsx 里，费用汇总面板只能空手试算。
	// 工程量优先取综合单价表的 Items.Quantity，为 0 时回退工程量计算表。
	s.applyBillItems(b, items, resolved, &res)
	return res
}

// applyBillItems 落项目封面+费率+分部分项清单（幂等：同项目同编码走更新）。
// 费率来自模版「费用汇总」取费区（录入数据落库，不做任何加总）。
func (s *Store) applyBillItems(b *ProjectBundle, items []ProjectItem, resolved map[string]string, res *ApplyProjectResult) {
	projectName := strings.TrimSpace(b.Project)
	if projectName == "" {
		projectName = strings.TrimSuffix(b.FileName, ".xlsx")
	}
	if projectName == "" {
		res.Errors = append(res.Errors, "清单落库跳过：工作簿无项目名（封面未解析到）")
		return
	}
	rates := RateSet{
		ManagementRate: b.Fee.ManagementRate,
		RegulatoryRate: b.Fee.RegulatoryRate,
		ProfitRate:     b.Fee.ProfitRate,
		TaxRate:        b.Fee.TaxRate,
	}
	projID, err := s.UpsertBillProject(projectName, b.FileName, b.Path, b.Location, b.Duration, b.Pricing,
		&rates, false, b.Fee.ControlPrice)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Sprintf("项目 %q 落库失败: %v", projectName, err))
		return
	}
	// 项目的专业/地区/计价期随导入口径（已过规矩）；空值保持原值。
	if err := s.UpdateBillProjectProfile(projID, b.Specialty, b.Region, b.Period); err != nil {
		res.Errors = append(res.Errors, fmt.Sprintf("项目 %q 的专业/地区/计价期落库失败: %v", projectName, err))
	}
	qtyByCode := map[string]float64{}
	for _, q := range b.Quantities {
		if q.Code != "" && q.Value > 0 {
			qtyByCode[q.Code] = q.Value
		}
	}
	for i, it := range items {
		qty := it.Quantity
		if qty == 0 {
			qty = qtyByCode[it.Code]
		}
		bill := BillItem{
			ProjectID:    projID,
			Code:         it.Code,
			Title:        firstNonEmpty(it.Title, "（未命名清单项）"),
			Unit:         it.Unit,
			Division:     it.Division,
			Quantity:     qty,
			QuantityExpr: it.QuantityExpr,
			QuotaCode:    firstNonEmpty(resolved[it.Code], it.Code), // 统一码（与定额库锚点一致，便于索引引用）
			Feature:      it.Feature,
			ListPrice:    it.ListPrice, // 原表综合单价快照（重导入随工作簿刷新）
			Sort:         i + 1,
			// 专业留空=跟随项目默认专业（整理归类时可逐行覆盖，重导入不冲掉）。
		}
		saved, err := s.SaveBillItem(bill)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("清单项 %s（%s）落库失败: %v", it.Code, it.Title, err))
			continue
		}
		// 1:N 引用表双写（SchemaV30）：单值 quota_code 仍是主定额锚点，link 表
		// 是引用事实全量——挂接编辑/多定额组合以此为准。quantity=清单工程量
		// （各定额自身工程量后续在挂接面调整）。
		if saved.QuotaCode != "" {
			if _, err := s.AttachBillQuotaLink(saved.ID, saved.QuotaCode, saved.Quantity); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("清单项 %s 挂接定额 %s 失败: %v", it.Code, saved.QuotaCode, err))
			}
		}
	}
}

// ImportScope 导入口径过规矩：专业、地区必须在字典里（空=待整理，放行），计价期
// 归一成 YYYY-MM（空=不记期）。返回归一后的三个值；不过规矩返回的 error 道出
// 该怎么改。导入的演练与正式落库都先过它——口径不对就整份拒收，不带病入库。
func (s *Store) ImportScope(b *ProjectBundle) (specialty, region, period string, err error) {
	specialty = strings.TrimSpace(b.Specialty)
	region = strings.TrimSpace(b.Region)
	if err = s.checkDict(DictSpecialty, specialty, "项目专业"); err != nil {
		return "", "", "", err
	}
	if err = s.checkDict(DictRegion, region, "项目地区"); err != nil {
		return "", "", "", err
	}
	period, ok := NormalizePeriod(b.Period)
	if !ok {
		return "", "", "", fmt.Errorf("计价期「%s」不是年月写法——统一写成 YYYY-MM，如 2026-06", strings.TrimSpace(b.Period))
	}
	return specialty, region, period, nil
}

// QuotaResolution 导入链定额编码的解析结果。
type QuotaResolution struct {
	// Code 落库用的全局唯一定额编码。
	Code string
	// Reuse 复用既有定额（同项目重导入，或同专业同内容）。
	Reuse bool
	// SameSource 复用的是本项目自己的定额（同项目重导入）——头字段与消耗行随
	// 工作簿刷新；别的项目的同专业同内容定额则保持不动。
	SameSource bool
	// Derived 借鉴溯源「借鉴 <专业> <码>」——外专业同码另立本专业新定额时非空。
	Derived string
}

// ResolveQuota 导入链定额编码解析规则（定额专业随来源定、跨专业不并）：
//
//  1. 原码全局未用 → 直接用原码（新定额，专业 = 项目专业）；
//  2. 原码被同源占用（orig_code 与源文件都相同 = 同项目重导入）→ 复用（幂等）；
//  3. 原码被他项目占用，标题+单位+工序特征全等，且专业相同（或任一方未标注）→
//     复用同一条（同专业内真统一：同内容共享一条定额，多项目引用同一锚点）；
//  4. 原码被他项目占用、内容全等但**专业不同** → 不并：另立本专业新定额（原码-N），
//     Derived 记「借鉴 <既有专业> <原码>」——同样一件事各专业各有定额；
//  5. 其余冲突 → 原码-N 消歧（N 从 2 起），orig_code 保留原码可溯源。
func (s *Store) ResolveQuota(origCode, title, unit, feature, source, specialty string) (QuotaResolution, error) {
	var (
		id                              int64
		orig, src, et, eu, en, esp, edf string
	)
	err := s.db.QueryRow(`SELECT id, COALESCE(orig_code,''), source, title, unit, COALESCE(note,''), COALESCE(specialty,''), COALESCE(derived_from,'')
FROM gf_quotas WHERE code=?`, origCode).Scan(&id, &orig, &src, &et, &eu, &en, &esp, &edf)
	if err == nil {
		if orig == origCode && src == source {
			return QuotaResolution{Code: origCode, Reuse: true, SameSource: true}, nil // 同项目重导入
		}
		sameContent := et == title && eu == unit && en == feature
		sameSpecialty := esp == "" || specialty == "" || esp == specialty
		switch {
		case sameContent && sameSpecialty:
			return QuotaResolution{Code: origCode, Reuse: true}, nil // 同专业同内容 → 统一复用
		case sameContent:
			// 跨专业同内容：另立本专业新定额，溯源记借鉴。
			code, err := s.freeQuotaCode(origCode)
			if err != nil {
				return QuotaResolution{}, err
			}
			return QuotaResolution{Code: code, Derived: "借鉴 " + esp + " " + origCode}, nil
		}
		code, err := s.freeQuotaCode(origCode)
		if err != nil {
			return QuotaResolution{}, err
		}
		return QuotaResolution{Code: code}, nil
	}
	if strings.Contains(err.Error(), "no rows") {
		return QuotaResolution{Code: origCode}, nil
	}
	return QuotaResolution{}, err
}

// freeQuotaCode 原码-N 消歧码（N 从 2 起，取最小空位）。
func (s *Store) freeQuotaCode(origCode string) (string, error) {
	for n := 2; ; n++ {
		cand := fmt.Sprintf("%s-%d", origCode, n)
		var dummy int64
		if err := s.db.QueryRow(`SELECT id FROM gf_quotas WHERE code=?`, cand).Scan(&dummy); err != nil {
			return cand, nil // 空位 → 消歧码
		}
	}
}

// setQuotaSpecialty 只改定额的专业（导入链内用：随来源项目纠正或补标）。
func (s *Store) setQuotaSpecialty(code, specialty string) error {
	if err := s.checkDict(DictSpecialty, specialty, "定额专业"); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE gf_quotas SET specialty=?, updated_at=? WHERE code=?`, specialty, nowText(), code)
	return err
}
