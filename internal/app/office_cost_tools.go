package app

// office_cost_tools.go — 造价运行模式专用工具三件（2026-11-21 众神殿
// pan_1795202763988 终案：造价三件拆分=读/写+PersistWrite/机检）。
//
//   - cost_project：只读统一读口（workcost 项目/清单/定额/资源/含量行）。
//   - cost_project_apply：写入口（建项目/更新费率/批量清单），标 PersistWrite
//     （子代理不继承持久写工具，task.go:909-917 整件过滤——终案 Codex/DSH
//     共识）；save_items 同批编码查重是**写前硬闸**（schema UNIQUE(project_id,
//     code) 同码走 UPDATE 会静默吞掉同批后行——落库后扫描永远扫不出来，
//     终案 Grok-r3 定罪）。
//   - cost_audit：只读确定性机检。四类行级硬检（身份/单位缺失、无计价路径、
//     定额悬空、资源悬空）+ 信息项永不进 HardCount（费率≤0=显式不计取不报、
//     费率>1 info、BillAuditScan 跨项目查漏 info、ListPrice 快照只记不比差）；
//     不调用 CheckComposeComponents（组价 R1–R3 不出现在本工具输出——终案
//     Grok-r3 禁令）。
//
// 数据面红线（终案三席四源一致）：一律锚 workcost（工料机×定额=综合单价，
// 活重算链）；costproject 是蒸馏旧面，不接（gaea_cost_projects.go 遗留面标记）。
// 装配点=extraToolsWithMediaRoot；空间=work（spacetags.go 静态表同批）；
// 模式面=造价档可见（standard/proposal 经 boot.Options.ToolHide 隐藏）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

// workcostNotFound 判定 store 错误是否为「编码不存在」（findings 语义）：
// store 的两类 not-found 都以「不存在」收尾（store.go GetQuota/GetResourceByCode）；
// 其余错误（库不可用/SQL 失败）是**工具失败**，不得降级成行级悬空——
// fail-closed 直接把错误抛回，避免把引擎故障误报成数据问题（终案 Grok-r3 边界）。
func workcostNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "不存在")
}

// ── cost_project：只读统一读口 ───────────────────────────────────────────

type costProjectTool struct{ a *App }

func (t costProjectTool) Name() string   { return "cost_project" }
func (t costProjectTool) ReadOnly() bool { return true }
func (t costProjectTool) Description() string {
	return "读取工料机造价库（workcost）：测算项目（封面+四费率+控制价）、项目清单行、消耗定额（含含量行）、工料机资源。测算前先用它了解定额与资源存量；写入口用 cost_project_apply，机检用 cost_audit。"
}
func (t costProjectTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"required":["action"],
"properties":{
  "action":{"type":"string","enum":["projects","items","quotas","quota","resources"],"description":"projects=项目列表；items=某项目清单行（需 project_id）；quotas=定额检索（specialty/keyword 过滤）；quota=单个定额详情（含含量行，需 code）；resources=工料机资源检索（kind/keyword 过滤）"},
  "project_id":{"type":"integer","description":"项目 id（action=items 必填）"},
  "code":{"type":"string","description":"定额编码（action=quota 必填）"},
  "specialty":{"type":"string","description":"定额专业过滤（可选）"},
  "kind":{"type":"string","description":"资源类型过滤（人工/材料/机械等，可选）"},
  "keyword":{"type":"string","description":"关键词过滤（可选）"},
  "limit":{"type":"integer","description":"返回条数上限（默认 50，最大 200）"}
}
}`)
}

func (t costProjectTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action    string `json:"action"`
		ProjectID int64  `json:"project_id"`
		Code      string `json:"code"`
		Specialty string `json:"specialty"`
		Kind      string `json:"kind"`
		Keyword   string `json:"keyword"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("cost_project 参数解析失败: %w", err)
	}
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 50
	}
	s := t.a.hubWorkcostStore()
	if !s.Available() {
		return "", fmt.Errorf("工料机造价库不可用（Hephaestus.db 未就绪）")
	}
	switch p.Action {
	case "projects":
		projects, err := s.BillProjects()
		if err != nil {
			return "", err
		}
		if len(projects) == 0 {
			return "造价库暂无测算项目。用 cost_project_apply action=create_project 建项目开始测算。", nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "测算项目 %d 个：\n", len(projects))
		for _, pr := range projects {
			fmt.Fprintf(&b, "- #%d %s（清单 %d 行，地区=%s 计价=%s 控制价=%.2f）\n", pr.ID, pr.Name, pr.ItemCount, pr.Location, pr.Pricing, pr.ControlPrice)
		}
		return b.String(), nil
	case "items":
		if p.ProjectID <= 0 {
			return "", fmt.Errorf("action=items 需要 project_id（先用 action=projects 列项目）")
		}
		items, err := s.BillItems(p.ProjectID)
		if err != nil {
			return "", err
		}
		if len(items) == 0 {
			return "该项目暂无清单行。用 cost_project_apply action=save_items 编制清单。", nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "项目 #%d 清单 %d 行（合价由库按当前价现算，此处不显示）：\n", p.ProjectID, len(items))
		for _, it := range items {
			quotas := strings.Join(it.QuotaCodes, "+")
			if it.QuotaCode != "" && !strings.Contains(quotas, it.QuotaCode) {
				quotas = it.QuotaCode + (func() string { if quotas != "" { return "+" + quotas }; return "" }())
			}
			line := fmt.Sprintf("- [%s] %s（%s）量=%g", it.Code, it.Title, it.Unit, it.Quantity)
			if quotas != "" {
				line += fmt.Sprintf(" 定额=%s", quotas)
			}
			if it.PriceOverride > 0 {
				line += fmt.Sprintf(" 手填价=%.2f", it.PriceOverride)
			}
			if it.ListPrice > 0 {
				line += fmt.Sprintf(" 导入快照=%.2f", it.ListPrice)
			}
			b.WriteString(line + "\n")
		}
		return b.String(), nil
	case "quotas":
		quotas, err := s.ListQuotas(p.Specialty, p.Keyword)
		if err != nil {
			return "", err
		}
		if len(quotas) > p.Limit {
			quotas = quotas[:p.Limit]
		}
		if len(quotas) == 0 {
			return fmt.Sprintf("定额库无匹配（specialty=%q keyword=%q）。可让用户提供定额编码或改用手填价。", p.Specialty, p.Keyword), nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "定额 %d 条（限 %d）：\n", len(quotas), p.Limit)
		for _, q := range quotas {
			fmt.Fprintf(&b, "- [%s] %s（%s，含量行 %d，引用 %d 次）\n", q.Code, q.Title, q.Unit, len(q.Items), q.UsageCount)
		}
		return b.String(), nil
	case "quota":
		if strings.TrimSpace(p.Code) == "" {
			return "", fmt.Errorf("action=quota 需要 code")
		}
		q, err := s.GetQuota(p.Code)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "定额 [%s] %s（%s，专业=%s）：\n", q.Code, q.Title, q.Unit, q.Specialty)
		for _, it := range q.Items {
			fmt.Fprintf(&b, "- %s %s（%s）含量=%g 损耗=%g%%\n", it.ResourceCode, it.Title, it.Unit, it.Quantity, it.LossRate*100)
		}
		return b.String(), nil
	case "resources":
		resources, err := s.ListResources(p.Kind, p.Keyword)
		if err != nil {
			return "", err
		}
		if len(resources) > p.Limit {
			resources = resources[:p.Limit]
		}
		if len(resources) == 0 {
			return fmt.Sprintf("工料机库无匹配（kind=%q keyword=%q）。", p.Kind, p.Keyword), nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "资源 %d 条（限 %d）：\n", len(resources), p.Limit)
		for _, r := range resources {
			fmt.Fprintf(&b, "- [%s] %s %s（%s，现行价=%.2f %s）\n", r.Code, r.Kind, r.Title, r.Unit, r.CurrentPrice, r.PriceDate)
		}
		return b.String(), nil
	default:
		return "", fmt.Errorf("未知 action %q（projects/items/quotas/quota/resources）", p.Action)
	}
}

// ── cost_project_apply：写入口（PersistWrite）─────────────────────────────

type costProjectApplyTool struct{ a *App }

func (t costProjectApplyTool) Name() string        { return "cost_project_apply" }
func (t costProjectApplyTool) ReadOnly() bool      { return false }
func (t costProjectApplyTool) PersistWrite() bool  { return true }
func (t costProjectApplyTool) Description() string {
	return "写入工料机造价库（workcost）：create_project 建测算项目（封面+四费率，费率 0=该段显式不计取）、update_rates 更新项目费率、save_items 批量写清单行（套定额用 quota_code/quota_codes，手填价用 price_override；同批编码必须唯一——重复编码整批拒绝，不落一行）。沉淀回成本库走 cost_save（用户确认门），本工具不写成本库。"
}
func (t costProjectApplyTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"required":["action"],
"properties":{
  "action":{"type":"string","enum":["create_project","update_rates","save_items"],"description":"create_project=建项目；update_rates=更新项目费率；save_items=批量写清单行（幂等：同批+库内同编码走更新）"},
  "name":{"type":"string","description":"项目名（create_project 必填，同名=更新封面）"},
  "file_name":{"type":"string","description":"源工作簿名（可选）"},
  "source":{"type":"string","description":"数据来源说明（可选）"},
  "location":{"type":"string","description":"地区（可选）"},
  "duration":{"type":"string","description":"工期（可选）"},
  "pricing":{"type":"string","description":"计价方式（可选）"},
  "management_rate":{"type":"number","description":"企业管理费率（0=不计取）"},
  "regulatory_rate":{"type":"number","description":"规费费率（0=不计取）"},
  "profit_rate":{"type":"number","description":"利润费率（0=不计取）"},
  "tax_rate":{"type":"number","description":"增值税费率（0=不计取）"},
  "profit_includes_regulatory":{"type":"boolean","description":"利润取费基数是否含规费"},
  "control_price":{"type":"number","description":"控制价（可选）"},
  "project_id":{"type":"integer","description":"项目 id（update_rates/save_items 必填）"},
  "items":{"type":"array","description":"清单行数组（save_items 必填，单批 ≤100 行）","items":{"type":"object","properties":{
    "code":{"type":"string","description":"清单编码（空=自动 M 序号；同批内不得重复）"},
    "title":{"type":"string","description":"清单项名称（必填）"},
    "unit":{"type":"string","description":"单位（必填）"},
    "division":{"type":"string","description":"分部（可选）"},
    "quantity":{"type":"number","description":"工程量（必须>0）"},
    "quantity_expr":{"type":"string","description":"工程量计算式（留痕，可选）"},
    "quota_code":{"type":"string","description":"主定额编码（手填行留空）"},
    "quota_codes":{"type":"array","items":{"type":"string"},"description":"组合定额编码列表（可选，1:N 挂接）"},
    "feature":{"type":"string","description":"工序特征（可选）"},
    "price_override":{"type":"number","description":"手填单价（套定额行留 0=按定额现算）"}
  }}}
}
}`)
}

func (t costProjectApplyTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action                   string  `json:"action"`
		Name                     string  `json:"name"`
		FileName                 string  `json:"file_name"`
		Source                   string  `json:"source"`
		Location                 string  `json:"location"`
		Duration                 string  `json:"duration"`
		Pricing                  string  `json:"pricing"`
		ManagementRate           float64 `json:"management_rate"`
		RegulatoryRate           float64 `json:"regulatory_rate"`
		ProfitRate               float64 `json:"profit_rate"`
		TaxRate                  float64 `json:"tax_rate"`
		ProfitIncludesRegulatory bool    `json:"profit_includes_regulatory"`
		ControlPrice             float64 `json:"control_price"`
		ProjectID                int64   `json:"project_id"`
		Items                    []struct {
			Code         string   `json:"code"`
			Title        string   `json:"title"`
			Unit         string   `json:"unit"`
			Division     string   `json:"division"`
			Quantity     float64  `json:"quantity"`
			QuantityExpr string   `json:"quantity_expr"`
			QuotaCode    string   `json:"quota_code"`
			QuotaCodes   []string `json:"quota_codes"`
			Feature      string   `json:"feature"`
			PriceOverride float64 `json:"price_override"`
		} `json:"items"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("cost_project_apply 参数解析失败: %w", err)
	}
	s := t.a.hubWorkcostStore()
	if !s.Available() {
		return "", fmt.Errorf("工料机造价库不可用（Hephaestus.db 未就绪）")
	}
	switch p.Action {
	case "create_project":
		if strings.TrimSpace(p.Name) == "" {
			return "", fmt.Errorf("create_project 需要 name")
		}
		rates := workcost.RateSet{
			ManagementRate: p.ManagementRate, RegulatoryRate: p.RegulatoryRate,
			ProfitRate: p.ProfitRate, TaxRate: p.TaxRate,
		}
		id, err := s.UpsertBillProject(p.Name, p.FileName, p.Source, p.Location, p.Duration, p.Pricing, &rates, p.ProfitIncludesRegulatory, p.ControlPrice)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("项目已建档：#%d %s（费率按提交值落库，0=该段显式不计取）。下一步 action=save_items 编制清单。", id, p.Name), nil
	case "update_rates":
		if p.ProjectID <= 0 {
			return "", fmt.Errorf("update_rates 需要 project_id")
		}
		rates := workcost.RateSet{
			ManagementRate: p.ManagementRate, RegulatoryRate: p.RegulatoryRate,
			ProfitRate: p.ProfitRate, TaxRate: p.TaxRate,
		}
		if err := s.UpdateBillProjectRates(p.ProjectID, rates, p.ProfitIncludesRegulatory, p.ControlPrice); err != nil {
			return "", err
		}
		return fmt.Sprintf("项目 #%d 费率已更新（0=显式不计取）。", p.ProjectID), nil
	case "save_items":
		if p.ProjectID <= 0 {
			return "", fmt.Errorf("save_items 需要 project_id（先用 action=create_project 建项目）")
		}
		if len(p.Items) == 0 {
			return "", fmt.Errorf("save_items 需要 items 数组")
		}
		if len(p.Items) > 100 {
			return "", fmt.Errorf("单批最多 100 行（当前 %d），请分批提交", len(p.Items))
		}
		// 写前硬闸（终案 Grok-r3）：先**全量校验后写入**——任何一行不合规即
		// 整批拒绝、一行不落（不留半批脏数据）。
		//   ① 同批编码唯一：库 UNIQUE(project_id,code) 对同码走 UPDATE——同批
		//      两行同码会被后行静默吞掉，落库后扫描永远扫不出来，必须在调用
		//      SaveBillItem 之前拦截。
		//   ② 行字段：title/unit 必填、quantity>0。
		//   ③ 定额存在性：悬空定额写前拦下；资源悬空留给 cost_audit（含量行走
		//      读口，写前不扫资源）。
		seen := map[string]int{}
		for i, it := range p.Items {
			if code := strings.TrimSpace(it.Code); code != "" {
				if prev, ok := seen[code]; ok {
					return "", fmt.Errorf("同批编码重复：%s（第 %d 行与第 %d 行）。整批未写入——去重或改码后重新提交", code, prev+1, i+1)
				}
				seen[code] = i
			}
			if strings.TrimSpace(it.Title) == "" || strings.TrimSpace(it.Unit) == "" {
				return "", fmt.Errorf("第 %d 行缺 title/unit。整批未写入", i+1)
			}
			if it.Quantity <= 0 {
				return "", fmt.Errorf("第 %d 行工程量必须>0。整批未写入", i+1)
			}
			codes := append([]string(nil), it.QuotaCodes...)
			if it.QuotaCode != "" {
				codes = append([]string{it.QuotaCode}, codes...)
			}
			dup := map[string]bool{}
			for _, c := range codes {
				c = strings.TrimSpace(c)
				if c == "" || dup[c] {
					continue
				}
				dup[c] = true
				if _, err := s.GetQuota(c); err != nil && !workcostNotFound(err) {
					return "", fmt.Errorf("第 %d 行定额 %s 校验失败: %w（整批未写入）", i+1, c, err)
				} else if workcostNotFound(err) {
					return "", fmt.Errorf("第 %d 行定额 %s 不存在（整批未写入）：核对编码或让用户提供，不要编造定额号", i+1, c)
				}
			}
		}
		saved := 0
		for _, it := range p.Items {
			item := workcost.BillItem{
				ProjectID: p.ProjectID, Code: strings.TrimSpace(it.Code),
				Title: strings.TrimSpace(it.Title), Unit: strings.TrimSpace(it.Unit),
				Division: it.Division, Quantity: it.Quantity, QuantityExpr: it.QuantityExpr,
				QuotaCode: strings.TrimSpace(it.QuotaCode), Feature: it.Feature,
				PriceOverride: it.PriceOverride,
			}
			if len(it.QuotaCodes) > 0 {
				item.QuotaCodes = it.QuotaCodes
			} else {
				item.QuotaCodes = []string{} // 构造即非 nil（JSON null 崩前端）
			}
			if _, err := s.SaveBillItem(item); err != nil {
				return "", fmt.Errorf("第 %d 行写入失败（前面行已落库，修复该行后重新提交同批即可按编码幂等更新）: %w", saved+1, err)
			}
			saved++
		}
		return fmt.Sprintf("项目 #%d 清单已写入 %d/%d 行（写前校验：编码唯一/工程量>0/定额存在）。用 cost_audit 跑机检后再交付。", p.ProjectID, saved, len(p.Items)), nil
	default:
		return "", fmt.Errorf("未知 action %q（create_project/update_rates/save_items）", p.Action)
	}
}

// ── cost_audit：只读确定性机检 ───────────────────────────────────────────

type costAuditTool struct{ a *App }

func (t costAuditTool) Name() string   { return "cost_audit" }
func (t costAuditTool) ReadOnly() bool { return true }
func (t costAuditTool) Description() string {
	return "对测算项目跑确定性机检（只读，不改数据）：行级硬检=身份/单位缺失、无计价路径（无定额且无手填价）、定额悬空、资源悬空（HardCount>0 必须修复后再交付）；信息项不阻塞=费率>1、跨项目查漏、库级健康扫描。费率≤0 是显式不计取，不报错。"
}
func (t costAuditTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"required":["project_id"],
"properties":{
  "project_id":{"type":"integer","description":"项目 id（cost_project action=projects 可查）"},
  "health":{"type":"boolean","description":"附带库级健康扫描（默认 false）"}
}
}`)
}

func (t costAuditTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		ProjectID int64 `json:"project_id"`
		Health    bool  `json:"health"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("cost_audit 参数解析失败: %w", err)
	}
	if p.ProjectID <= 0 {
		return "", fmt.Errorf("需要 project_id")
	}
	s := t.a.hubWorkcostStore()
	if !s.Available() {
		return "", fmt.Errorf("工料机造价库不可用（Hephaestus.db 未就绪）")
	}
	// 项目封面（费率 info 面）。
	var proj *workcost.BillProject
	projects, err := s.BillProjects()
	if err != nil {
		return "", err
	}
	for i := range projects {
		if projects[i].ID == p.ProjectID {
			proj = &projects[i]
			break
		}
	}
	if proj == nil {
		return "", fmt.Errorf("项目 #%d 不存在（cost_project action=projects 可查）", p.ProjectID)
	}
	items, err := s.BillItems(p.ProjectID)
	if err != nil {
		return "", err
	}

	type auditFinding struct {
		code, detail string
		hard         bool
	}
	findings := []auditFinding{}
	add := func(code, detail string, hard bool) {
		findings = append(findings, auditFinding{code: code, detail: detail, hard: hard})
	}

	// 硬检一/二：行级字段（终案 Grok-r3 workcost 断言表）。
	quotaCache := map[string]*workcost.Quota{}
	reportedDangling := map[string]bool{} // 同一悬空定额码只报一次（QuotaCode 与 QuotaCodes 双挂同码不去重会双报）
	for _, it := range items {
		row := fmt.Sprintf("#%d[%s]", it.ID, it.Code)
		if strings.TrimSpace(it.Title) == "" || strings.TrimSpace(it.Unit) == "" {
			add("missing_identity", fmt.Sprintf("%s %s 缺名称或单位", row, it.Title), true)
			continue // 身份缺失行的后续判定无意义
		}
		// 套定额行 PriceOverride==0 是正常现算，不判；负手填价必错。
		if it.Quantity <= 0 {
			add("unpriced_line", fmt.Sprintf("%s %s 工程量=%g 必须>0", row, it.Title, it.Quantity), true)
		} else if it.PriceOverride < 0 {
			add("unpriced_line", fmt.Sprintf("%s %s 手填价=%g 为负", row, it.Title, it.PriceOverride), true)
		} else if it.PriceOverride <= 0 && it.QuotaCode == "" && len(it.QuotaCodes) == 0 {
			add("unpriced_line", fmt.Sprintf("%s %s 既无定额挂接也无手填价", row, it.Title), true)
		}
		// 硬检三：定额悬空（QuotaCodes∪QuotaCode 任一不在定额表）。同码只查
		// 一次（nil 也缓存=不存在已判）；库故障=工具失败抛回，不降级成悬空。
		codes := append([]string{it.QuotaCode}, it.QuotaCodes...)
		for _, c := range codes {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if _, cached := quotaCache[c]; !cached {
				got, err := s.GetQuota(c)
				if err != nil && !workcostNotFound(err) {
					return "", fmt.Errorf("定额 %s 校验失败: %w", c, err)
				}
				quotaCache[c] = got
			}
			if quotaCache[c] == nil && !reportedDangling[c] {
				reportedDangling[c] = true
				add("dangling_quota", fmt.Sprintf("%s %s 引用定额 %s 不存在", row, it.Title, c), true)
			}
		}
	}
	// 硬检四：资源悬空——仅本项目已挂定额（且定额存在）的含量行，ResourceCode
	// 非空但资源表查不到。资源缺失时快照价兜底可追溯（QuotaItem.ResourcePrice），
	// 但引用面断裂仍是硬伤。
	checkedRes := map[string]bool{}
	for _, it := range items {
		codes := append([]string{it.QuotaCode}, it.QuotaCodes...)
		for _, c := range codes {
			c = strings.TrimSpace(c)
			if c == "" || quotaCache[c] == nil {
				continue
			}
			for _, qi := range quotaCache[c].Items {
				rc := strings.TrimSpace(qi.ResourceCode)
				if rc == "" || checkedRes[rc] {
					continue
				}
				checkedRes[rc] = true
				if _, err := s.GetResourceByCode(rc); err != nil && workcostNotFound(err) {
					add("dangling_resource", fmt.Sprintf("定额 %s 含量行引用资源 %s 不存在", c, rc), true)
				} else if err != nil {
					return "", fmt.Errorf("资源 %s 校验失败: %w", rc, err)
				}
			}
		}
	}

	// 信息项（永不进 HardCount）。
	hardCount := 0
	for _, f := range findings {
		if f.hard {
			hardCount++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "项目 #%d %s 机检回执：\n硬检 HardCount=%d（必须修复后再交付）\n", proj.ID, proj.Name, hardCount)
	for _, f := range findings {
		tag := "INFO"
		if f.hard {
			tag = "HARD"
		}
		fmt.Fprintf(&b, "- [%s] %s: %s\n", tag, f.code, f.detail)
	}
	infoCount := 0
	// 费率：≤0=显式不计取（无结论不写「异常」）；>1 只 info。
	for _, r := range []struct {
		name string
		v    float64
	}{{"企业管理费", proj.ManagementRate}, {"规费", proj.RegulatoryRate}, {"利润", proj.ProfitRate}, {"增值税", proj.TaxRate}} {
		if r.v > 1 {
			fmt.Fprintf(&b, "- [INFO] rate_over_one: %s 费率 %.2f>1，请确认口径\n", r.name, r.v)
			infoCount++
		}
	}
	// 跨项目查漏（BillAuditScan 只做 info，永不进 HardCount）。
	if scan, err := s.BillAuditScan(p.ProjectID); err == nil && scan != nil {
		for _, f := range scan.BillGaps {
			fmt.Fprintf(&b, "- [INFO] bill_gap: %s（%s）\n", f.Detail, f.Target)
			infoCount++
		}
		for _, f := range scan.QuotaGaps {
			fmt.Fprintf(&b, "- [INFO] quota_gap: %s（%s）\n", f.Detail, f.Target)
			infoCount++
		}
	} else if err != nil {
		fmt.Fprintf(&b, "- [INFO] bill_gap: 跨项目查漏不可用（%v）\n", err)
		infoCount++
	}
	if p.Health {
		if hr, err := s.HealthScan(); err == nil {
			fmt.Fprintf(&b, "- [INFO] health: 库级健康发现 %d 条（零价/悬空引用/重复/编码错位，明细在模型中心工料机页）\n", hr.TotalFindings)
			infoCount++
		} else {
			fmt.Fprintf(&b, "- [INFO] health: 库级健康扫描不可用（%v）\n", err)
			infoCount++
		}
	}
	fmt.Fprintf(&b, "信息项 %d 条（不阻塞）。", infoCount)
	if hardCount == 0 {
		b.WriteString("硬检全绿，可进入沉淀（cost_save）与交付。")
	} else {
		b.WriteString("存在硬检问题：修复后重新机检，不要带硬伤交付。")
	}
	return b.String(), nil
}
