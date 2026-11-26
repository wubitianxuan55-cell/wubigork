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
//     组价 R1–R3 校验不出现在本工具输出（终案 Grok-r3 禁令；旧组价校验引擎已随
//     旧成本库退役，2026-11-26）。
//
// 数据面红线（终案三席四源一致）：一律锚 workcost（工料机×定额=综合单价，
// 活重算链）；costproject 是蒸馏旧面，不接（gaea_cost_projects.go 遗留面标记）。
// 装配点=extraToolsWithMediaRoot；空间=work（spacetags.go 静态表同批）；
// 模式面=造价档可见（standard/proposal 经 boot.Options.ToolHide 隐藏）。

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
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
	return "读取工料机造价库（workcost）：测算项目（封面+四费率+控制价）、项目清单行、消耗定额（含含量行）、工料机资源。带 keyword 检索时编码精确优先，索引就绪可按同义说法召回（标〈语义〉）；bills 跨项目找清单项。测算前先用它了解定额与资源存量；写入口用 cost_project_apply，机检用 cost_audit。"
}
func (t costProjectTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"required":["action"],
"properties":{
  "action":{"type":"string","enum":["projects","items","summary","quotas","quota","resources","bills"],"description":"projects=项目列表；items=某项目清单行（需 project_id）；summary=现算汇总（逐行综合单价/合价/人材机拆分+项目取费链——金额一律以它为准，不要自建 SQL 算价链）；quotas=定额检索（specialty/keyword 过滤）；quota=单个定额详情（含含量行，需 code）；resources=工料机资源检索（kind/keyword 过滤）；bills=跨项目清单行检索（keyword/specialty 过滤，project_id 可限定项目）"},
  "project_id":{"type":"integer","description":"项目 id（action=items/summary 必填；bills 可选）"},
  "code":{"type":"string","description":"定额编码（action=quota 必填）"},
  "specialty":{"type":"string","description":"定额专业过滤（可选）"},
  "kind":{"type":"string","description":"资源类型过滤（人工/材料/机械等，可选）"},
  "region":{"type":"string","description":"工料机地区过滤（resources 可选，字典值；工料机按地区分行）"},
  "keyword":{"type":"string","description":"检索词（可选）：编码精确命中优先；名称或规格含它即可召回；索引就绪时同义说法也能召回"},
  "limit":{"type":"integer","description":"返回条数上限（默认 50，上限 200；回执给总数与是否还有更多）"}
}
}`)
}

func (t costProjectTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action    string `json:"action"`
		ProjectID int64  `json:"project_id"`
		Code      string `json:"code"`
		Specialty string `json:"specialty"`
		Kind      string `json:"kind"`
		Region    string `json:"region"`
		Keyword   string `json:"keyword"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("cost_project 参数解析失败: %w", err)
	}
	// limit 钳位到 [1, 200]（2026-11-26 造价反馈，百锦路 #4）：此前 limit>200 或
	// ≤0 一律回落 50——模型照文档给 300 反而只拿到 50 条，截断得毫无提示。
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 200 {
		p.Limit = 200
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
			fmt.Fprintf(&b, "- #%d %s（清单 %d 行，封面地点=%s 计价=%s 控制价=%.2f%s）\n", pr.ID, pr.Name, pr.ItemCount, pr.Location, pr.Pricing, pr.ControlPrice,
				clProjectScope(pr.Specialty, pr.Region, pr.PricePeriod))
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
				quotas = it.QuotaCode + (func() string {
					if quotas != "" {
						return "+" + quotas
					}
					return ""
				}())
			}
			// 行尾带数字 id（2026-11-26 造价反馈，旺平 #6）：classify.bills / delete_items
			// / borrow_quota 的 item_ids 都要它，契约写着「items 可查」而此处不打印。
			line := fmt.Sprintf("- [%s] %s（%s）量=%g id=%d", it.Code, it.Title, it.Unit, it.Quantity, it.ID)
			if sp := strings.TrimSpace(it.Specialty); sp != "" {
				line += " 专业=" + sp
			}
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
	case "summary":
		if p.ProjectID <= 0 {
			return "", fmt.Errorf("action=summary 需要 project_id（先用 action=projects 列项目）")
		}
		return t.runSummary(p.ProjectID)
	case "quotas":
		// 多取一条判断「还有更多」；列表模式（无检索词）另给总数。
		hits, err := t.a.costSearcher().Quotas(ctx, workcost.SearchQuery{Text: p.Keyword, Specialty: p.Specialty, Limit: p.Limit + 1})
		if err != nil {
			return "", err
		}
		hasMore := len(hits) > p.Limit
		if hasMore {
			hits = hits[:p.Limit]
		}
		quotas := make([]workcost.Quota, len(hits))
		for i, h := range hits {
			quotas[i] = h.Item
		}
		if len(quotas) == 0 {
			return fmt.Sprintf("定额库无匹配（specialty=%q keyword=%q）。可让用户提供定额编码或改用手填价。", p.Specialty, p.Keyword), nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "定额 %d 条%s%s:\n", len(quotas),
			clListScale(t.listTotal(p.Keyword == "", func() (int, error) { return s.CountQuotas(p.Specialty) }), len(quotas), hasMore), clViaNote(hits))
		for i, q := range quotas {
			// 列表是轻量查询、不带含量行：此前照样打印「含量行 N」，N 恒为 0，
			// 模型误以为导入失败而重复补录（2026-11-26 造价反馈，旺平 #6）。
			fmt.Fprintf(&b, "- [%s] %s（%s，%s，引用 %d 次）%s\n", q.Code, q.Title, q.Unit, clDim(q.Specialty, "未标注专业"), q.UsageCount, clViaMark(hits[i].Via))
		}
		b.WriteString("含量行用 action=quota 看单条详情（列表不带含量行）。\n")
		if strings.TrimSpace(p.Specialty) == "" {
			b.WriteString("清单套定额同专业优先：带上 specialty 只看本专业的定额；本专业没有匹配的，再用 cost_library borrow_quota 借鉴生成。\n")
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
		hits, err := t.a.costSearcher().Resources(ctx, workcost.SearchQuery{Text: p.Keyword, Kind: p.Kind, Region: p.Region, Limit: p.Limit + 1})
		if err != nil {
			return "", err
		}
		hasMore := len(hits) > p.Limit
		if hasMore {
			hits = hits[:p.Limit]
		}
		resources := make([]workcost.Resource, len(hits))
		for i, h := range hits {
			resources[i] = h.Item
		}
		if len(resources) == 0 {
			return fmt.Sprintf("工料机库无匹配（kind=%q region=%q keyword=%q）。", p.Kind, p.Region, p.Keyword), nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "资源 %d 条%s%s：\n", len(resources),
			clListScale(t.listTotal(p.Keyword == "", func() (int, error) { return s.CountResources(p.Kind, p.Region) }), len(resources), hasMore), clViaNote(hits))
		for i, r := range resources {
			// id=数字主键（铜陵反馈 #18：dedupe 的 keep_id/merge_ids 只吃数字 id，
			// 只读口不给 id 时消歧链路当场卡死）。
			fmt.Fprintf(&b, "- [%s] %s %s（%s，现行价=%.2f，地区=%s，期=%s，id=%d）%s\n", r.Code, r.Kind, r.Title, r.Unit, r.CurrentPrice,
				clDim(r.Region, "未标注"), clDim(r.PriceDate, "无"), r.ID, clViaMark(hits[i].Via))
		}
		if p.Region == "" && p.Keyword == "" {
			b.WriteString("同一种工料机按地区分行（每地区一行）：只看某地区带 region；查同身份各地区的行用 keyword=名称、不带 region。\n")
		}
		return b.String(), nil
	case "bills":
		// 跨项目清单行检索：找某个清单项在哪些项目里出现过、按名称 / 项目特征找清单项。
		hits, err := t.a.costSearcher().BillItems(ctx, workcost.SearchQuery{
			Text: p.Keyword, Specialty: p.Specialty, ProjectID: p.ProjectID, Limit: p.Limit + 1,
		})
		if err != nil {
			return "", err
		}
		hasMore := len(hits) > p.Limit
		if hasMore {
			hits = hits[:p.Limit]
		}
		if len(hits) == 0 {
			return fmt.Sprintf("清单库无匹配（specialty=%q keyword=%q project_id=%d）。", p.Specialty, p.Keyword, p.ProjectID), nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "清单行 %d 条（跨项目检索%s）%s：\n", len(hits),
			clListScale(-1, len(hits), hasMore), clViaNote(hits))
		for _, h := range hits {
			it := h.Item
			line := fmt.Sprintf("- 项目 #%d [%s] %s（%s）", it.ProjectID, it.Code, it.Title, it.Unit)
			if sp := strings.TrimSpace(it.Specialty); sp != "" {
				line += " 专业=" + sp
			}
			if len(it.QuotaCodes) > 0 {
				line += " 定额=" + strings.Join(it.QuotaCodes, "+")
			}
			b.WriteString(line + clViaMark(h.Via) + "\n")
		}
		return b.String(), nil
	default:
		return "", fmt.Errorf("未知 action %q（projects/items/quotas/quota/resources/bills）", p.Action)
	}
}

// listTotal 列表模式（无检索词）给总数；关键词检索不数总数（返回 -1=未知）。
// 数总数失败不拦列表，按未知处理。
func (costProjectTool) listTotal(listMode bool, count func() (int, error)) int {
	if !listMode {
		return -1
	}
	n, err := count()
	if err != nil {
		return -1
	}
	return n
}

// clListScale 列表回执头部的规模说明（2026-11-26 造价反馈，百锦路 #4：只给
// 「N 条（限 M）」不给总数，模型不知道是搜全了还是被截断，只能一轮轮换词猜）。
// total<0=总数未知；hasMore=本次没列全。
func clListScale(total, shown int, hasMore bool) string {
	const more = "提高 limit（上限 200）或加过滤条件缩小范围"
	switch {
	case hasMore && total >= 0:
		return fmt.Sprintf("（共 %d 条，本次列 %d 条；还有更多——%s）", total, shown, more)
	case hasMore:
		return fmt.Sprintf("（本次列 %d 条；还有更多——%s）", shown, more)
	case total >= 0:
		return fmt.Sprintf("（共 %d 条）", total)
	}
	return ""
}

// clViaNote / clViaMark 检索结果的命中方式标注。只有非关键词命中的才标：名称不含查询词
// 却出现的，是向量路按同义说法召回的（〈语义〉）；查询词正好是编码 / 自有码的标〈编码〉。
func clViaNote[T any](hits []workcost.Hit[T]) string {
	for _, h := range hits {
		if h.Via != workcost.ViaKeyword && h.Via != workcost.ViaList {
			return "（标〈编码〉= 编码精确命中；标〈语义〉= 名称不含查询词、由向量路按同义说法召回）"
		}
	}
	return ""
}

func clViaMark(via string) string {
	switch via {
	case workcost.ViaCode:
		return "〈编码〉"
	case workcost.ViaSemantic:
		return "〈语义〉"
	}
	return ""
}

// runSummary 现算汇总口（2026-11-21 造价门实测 4.1：金额是造价的核心产出，
// 此前 items 只给量不给价，模型被迫自建 SQL 四表 join 算价——两条算价链任何
// 口径偏差（损耗率/资源价覆盖/快照回退）都会出两份金额。本口复用库内
// ComposeQuotas 现算链（唯一权威），取费公式与测算样表费用汇总的取费链同源：
// 管理费=直接费×率；利润=(直接费+管理费)×率；规费=直接费×率；税前=四项显式
// 相加；税=税前×率。手填价行人材机归属约定=**整价归材料**（无组价依据时唯一
// 能保持 人+材+机=合价 恒等式的口径，此前靠模型自定——2026-11-21 拍板入文档）。
func (t costProjectTool) runSummary(projectID int64) (string, error) {
	s := t.a.hubWorkcostStore()
	if !s.Available() {
		return "", fmt.Errorf("工料机造价库不可用（Hephaestus.db 未就绪）")
	}
	var proj *workcost.BillProject
	projects, err := s.BillProjects()
	if err != nil {
		return "", err
	}
	for i := range projects {
		if projects[i].ID == projectID {
			proj = &projects[i]
			break
		}
	}
	if proj == nil {
		return "", fmt.Errorf("项目 #%d 不存在（cost_project action=projects 可查）", projectID)
	}
	items, err := s.BillItems(projectID)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "该项目暂无清单行——无汇总可算。用 cost_project_apply action=save_items 编制清单。", nil
	}
	// 定额码收集 → ComposeQuotas 一次批量现算（唯一权威链）。
	codeSet := map[string]bool{}
	var codes []string
	for _, it := range items {
		for _, c := range append([]string{it.QuotaCode}, it.QuotaCodes...) {
			if c = strings.TrimSpace(c); c != "" && !codeSet[c] {
				codeSet[c] = true
				codes = append(codes, c)
			}
		}
	}
	composes, err := s.ComposeQuotas(codes)
	if err != nil {
		return "", fmt.Errorf("现算链失败: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "项目 #%d %s 现算汇总（%d 行，口径=库内现算链，人材机综合单价不含管利税）：\n", proj.ID, proj.Name, len(items))
	var direct, sumLabor, sumMaterial, sumMachine, sumOther float64
	unpriced := 0
	warnSeen := map[string]bool{}
	// 分部小计的累加器（按首次出现顺序列出）。
	byDivision := map[string]bool{}
	divAmount, divRows := map[string]float64{}, map[string]int{}
	var divOrder []string
	// 与导入快照不一致的行数与按工程量合计的差额。
	snapDiff, snapDelta := 0, 0.0
	for _, it := range items {
		qty := it.Quantity
		row := fmt.Sprintf("- [%s] %s（%s）量=%g", it.Code, it.Title, it.Unit, qty)
		var total, labor, material, machine, other float64
		if it.PriceOverride > 0 {
			total = it.PriceOverride * qty
			material = total // 手填价约定：整价归材料
			row += fmt.Sprintf(" 手填价=%.2f", it.PriceOverride)
		} else {
			// 定额挂接（含 1:N 组合）：link.Quantity>0 用定额自身工程量，0=跟随清单量。
			links, lerr := s.BillQuotaLinks(it.ID)
			linkQty := map[string]float64{}
			if lerr == nil {
				for _, l := range links {
					linkQty[strings.TrimSpace(l.QuotaCode)] = l.Quantity
				}
			}
			used := map[string]bool{}
			for _, c := range append([]string{it.QuotaCode}, it.QuotaCodes...) {
				if c = strings.TrimSpace(c); c == "" || used[c] {
					continue
				}
				used[c] = true
				cp := composes[c]
				if cp == nil {
					continue // 悬空定额由 cost_audit 硬检点名；此处不重复
				}
				lq := qty
				if l := linkQty[c]; l > 0 {
					lq = l
				}
				total += cp.Subtotal * lq
				labor += cp.LaborFee * lq
				material += cp.MaterialFee * lq
				machine += cp.MachineFee * lq
				other += cp.OtherFee * lq
				for _, w := range cp.Warnings {
					warnSeen[w] = true
				}
			}
		}
		if total <= 0 && it.PriceOverride <= 0 {
			unpriced++
			fmt.Fprintf(&b, "%s → 无现算（缺定额挂接/悬空，cost_audit 会点名）\n", row)
			continue
		}
		unit := 0.0
		if qty > 0 {
			// 清单综合单价取到分、合价=单价×工程量（2026-11-26 造价反馈，百锦路
			// #6 / 太原 #9）：与六表工作簿 =ROUND(综合单价!H<组清单行>,2) 同口径。此前本口
			// 用全精度 S×量，与工作簿「取到分再乘」差出按量放大的尾数，模型只能
			// 逐行对账去反推舍入口径。
			raw := total
			unit = workcost.Round2(total / qty)
			total = unit * qty
			if raw > 0 {
				// 三费拆分随合价等比缩放，保住 人+材+机(+其他)=合价 的恒等式。
				k := total / raw
				labor, material, machine, other = labor*k, material*k, machine*k, other*k
			}
			if it.PriceOverride > 0 {
				material = total
			}
		}
		direct += total
		sumLabor += labor
		sumMaterial += material
		sumMachine += machine
		sumOther += other
		div := strings.TrimSpace(it.Division)
		if div == "" {
			div = "（未分部）"
		}
		if _, seen := byDivision[div]; !seen {
			byDivision[div] = true
			divOrder = append(divOrder, div)
		}
		divAmount[div] += total
		divRows[div]++
		fmt.Fprintf(&b, "%s 综合单价=%.2f 合价=%.2f（人=%.2f 材=%.2f 机=%.2f", row, unit, total, labor, material, machine)
		if other > 0 {
			fmt.Fprintf(&b, " 其他=%.2f", other)
		}
		if d := strings.TrimSpace(it.Division); d != "" {
			fmt.Fprintf(&b, " 分部=%s", d)
		}
		// 与导入快照（原表综合单价）对账（2026-11-26 造价反馈三批：三场都在
		// 「工作簿 ↔ 库现算」的尾差上手工逐行对了几轮）。差到分就逐行点名。
		if it.ListPrice > 0 {
			if diff := unit - it.ListPrice; math.Abs(diff) >= 0.005 {
				snapDiff++
				snapDelta += diff * qty
				fmt.Fprintf(&b, " 导入快照=%.2f（差 %+.2f）", it.ListPrice, diff)
			}
		}
		b.WriteString("）\n")
	}
	if unpriced > 0 {
		fmt.Fprintf(&b, "（另有 %d 行无现算，未计入合计）\n", unpriced)
	}
	if snapDiff > 0 {
		fmt.Fprintf(&b, "与导入快照（原表综合单价）不一致 %d 行，按工程量合计差 %+.2f——导入后调过资源价或含量，属正常重算；本该一致却对不上，先查工作簿两层 ROUND（定额行 M、清单 H），再逐行看定额含量。\n", snapDiff, snapDelta)
	}
	// 分部小计（2026-11-26 造价反馈，百锦路 #6）：与旧测算对账时按分部看差额，
	// 不必逐行誊录手算。
	if len(divOrder) > 0 {
		fmt.Fprintf(&b, "── 分部小计（人材机直接费）──\n")
		for _, d := range divOrder {
			fmt.Fprintf(&b, "- %s：%.2f（%d 行）\n", d, divAmount[d], divRows[d])
		}
	}
	// 取费链走库内唯一实现 ComposeProjectFees（逐段舍到分、认「利润基数含规费」
	// 开关），与六表导出同源。此前这里手写了一遍公式：不逐段舍到分、也不认
	// 开关——开关打开时 summary 与导出两份利润对不上。
	fees := workcost.ComposeProjectFees(direct, workcost.RateSet{
		ManagementRate: proj.ManagementRate, RegulatoryRate: proj.RegulatoryRate,
		ProfitRate: proj.ProfitRate, TaxRate: proj.TaxRate,
	}, workcost.FeePolicy{ProfitBaseIncludesRegulatory: proj.ProfitIncludesRegulatory}, 0, 0, proj.ControlPrice)
	fmt.Fprintf(&b, "── 项目汇总（费率 管=%g 规=%g 利=%g 税=%g）──\n", proj.ManagementRate, proj.RegulatoryRate, proj.ProfitRate, proj.TaxRate)
	fmt.Fprintf(&b, "人材机直接费=%.2f（人=%.2f 材=%.2f 机=%.2f）\n", fees.DirectFee, sumLabor, sumMaterial, sumMachine)
	fmt.Fprintf(&b, "管理费=%.2f｜利润=%.2f｜规费=%.2f｜税前=%.2f｜增值税=%.2f\n", fees.ManagementFee, fees.ProfitFee, fees.RegulatoryFee, fees.PreTaxTotal, fees.TaxFee)
	fmt.Fprintf(&b, "含税合计=%.2f", fees.Total)
	if proj.ControlPrice > 0 {
		fmt.Fprintf(&b, "（占控制价 %.1f%%）", fees.ControlUtilPct)
	}
	b.WriteString("\n舍入口径：每条定额单价舍到分；清单综合单价=Σ(系数×定额单价) 再舍到分；合价=综合单价×工程量；取费链逐段舍到分——与六表工作簿的 =ROUND(…,2) 同口径，两边应逐分一致。\n")
	b.WriteString("口径：综合单价=人材机口径（不含管利税）；手填价行人材机归属=整价归材料；导出 xlsx 以本链同源公式生成，两边金额必须一致——不一致=口径分叉，以库内现算为准排查。\n")
	if len(warnSeen) > 0 {
		keys := make([]string, 0, len(warnSeen))
		for w := range warnSeen {
			keys = append(keys, w)
		}
		sort.Strings(keys)
		fmt.Fprintf(&b, "现算警告 %d 条：%s\n", len(keys), strings.Join(keys, "；"))
	}
	return b.String(), nil
}

// ── cost_project_apply：写入口（PersistWrite）─────────────────────────────

type costProjectApplyTool struct{ a *App }

func (t costProjectApplyTool) Name() string       { return "cost_project_apply" }
func (t costProjectApplyTool) ReadOnly() bool     { return false }
func (t costProjectApplyTool) PersistWrite() bool { return true }
func (t costProjectApplyTool) Description() string {
	return "写入工料机造价库（workcost）：create_project 建测算项目（封面+四费率，费率 0=该段显式不计取；可带专业/地区/计价期，都是字典里的规范值），update_rates 更新项目费率（只改传入的字段），save_items 批量写清单行（套定额用 quota_code/quota_codes，手填价用 price_override；同批编码必须唯一——重复编码整批拒绝，不落一行）。quota_code 与 quota_codes 合起来就是该行的全部定额挂接（整体替换）：只改名称等字段时，把原挂接原样带回。清单套定额同专业优先：先用 cost_project action=quotas 带 specialty 检索本专业定额；本专业没有就用 cost_library borrow_quota 以外专业定额为底生成本专业的再挂——回执会点名挂了外专业定额的行。本工具只增不删；删除、导入工作簿、改定额与工料机走 cost_library。"
}
func (t costProjectApplyTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"required":["action"],
"properties":{
  "action":{"type":"string","enum":["create_project","update_rates","save_items"],"description":"create_project=建项目；update_rates=更新项目费率（未传的费率保持原值）；save_items=批量写清单行（幂等：同批+库内同编码走更新）"},
  "name":{"type":"string","description":"项目名（create_project 必填，同名=更新封面）"},
  "file_name":{"type":"string","description":"源工作簿名（可选）"},
  "source":{"type":"string","description":"数据来源说明（可选）"},
  "location":{"type":"string","description":"封面地址（可选，自由文本）"},
  "region":{"type":"string","description":"项目所在地区（字典值，可选；与封面地址分开）——工料机按地区分行，取价归价都看它"},
  "price_period":{"type":"string","description":"计价期，YYYY-MM（可选）"},
  "specialty":{"type":"string","description":"项目默认专业（字典值，可选）——清单与它套的定额都按这个专业走"},
  "duration":{"type":"string","description":"工期（可选）"},
  "pricing":{"type":"string","description":"计价方式（可选）"},
  "management_rate":{"type":"number","description":"企业管理费率（create_project 未传=0 不计取）"},
  "regulatory_rate":{"type":"number","description":"规费费率"},
  "profit_rate":{"type":"number","description":"利润费率"},
  "tax_rate":{"type":"number","description":"增值税费率"},
  "profit_includes_regulatory":{"type":"boolean","description":"利润取费基数是否含规费"},
  "control_price":{"type":"number","description":"控制价（可选）"},
  "project_id":{"type":"integer","description":"项目 id（update_rates/save_items 必填）"},
  "items":{"type":"array","description":"清单行数组（save_items 必填，单批 ≤100 行）","items":{"type":"object","properties":{
    "code":{"type":"string","description":"清单编码（空=自动 M 序号；同批内不得重复）"},
    "title":{"type":"string","description":"清单项名称（必填）"},
    "unit":{"type":"string","description":"单位（必填）"},
    "division":{"type":"string","description":"分部（可选）"},
    "specialty":{"type":"string","description":"清单专业（字典值，可选；不给=已有行保持原值，新行跟随项目默认专业）"},
    "quantity":{"type":"number","description":"工程量（必须>0）"},
    "quantity_expr":{"type":"string","description":"工程量计算式（留痕，可选）"},
    "quota_code":{"type":"string","description":"主定额编码（与 quota_codes 合起来=该行全部挂接，整体替换；手填行留空）"},
    "quota_codes":{"type":"array","items":{"type":"string"},"description":"其余定额编码（1:N 挂接；与 quota_code 同一批落库）"},
    "quota_factors":{"type":"object","description":"定额换算系数 {定额编码: 系数}（每一个清单单位含几个定额单位；须与 quota_code/quota_codes 同批给）。挂接量=清单工程量×系数；没列的定额沿用该行原有挂接的比例"},
    "feature":{"type":"string","description":"工序特征（可选）"},
    "price_override":{"type":"number","description":"手填单价（套定额行留 0=按定额现算）"},
    "list_price":{"type":"number","description":"原表综合单价快照（可选；不传=已有行保持原值，新行为 0）"}
  }}}
}
}`)
}

// costProjectApplyArgs cost_project_apply 的入参。费率与挂接字段用指针：区分
// 「没传」与「传了 0 / 空串」——update_rates 没传的费率保持原值（2026-11-25 修：
// 此前按 0 覆盖，改一个费率会把其余三个清零）；save_items 没传挂接字段=不动
// 该行的定额挂接。
type costProjectApplyArgs struct {
	Action                   string                 `json:"action"`
	Name                     string                 `json:"name"`
	FileName                 string                 `json:"file_name"`
	Source                   string                 `json:"source"`
	Location                 string                 `json:"location"`
	Region                   string                 `json:"region"`
	PricePeriod              string                 `json:"price_period"`
	Specialty                string                 `json:"specialty"`
	Duration                 string                 `json:"duration"`
	Pricing                  string                 `json:"pricing"`
	ManagementRate           *float64               `json:"management_rate"`
	RegulatoryRate           *float64               `json:"regulatory_rate"`
	ProfitRate               *float64               `json:"profit_rate"`
	TaxRate                  *float64               `json:"tax_rate"`
	ProfitIncludesRegulatory *bool                  `json:"profit_includes_regulatory"`
	ControlPrice             *float64               `json:"control_price"`
	ProjectID                int64                  `json:"project_id"`
	Items                    []costProjectApplyItem `json:"items"`
}

type costProjectApplyItem struct {
	Code         string   `json:"code"`
	Title        string   `json:"title"`
	Unit         string   `json:"unit"`
	Division     string   `json:"division"`
	Specialty    *string  `json:"specialty"` // 不给=已有行保持原值，新行跟随项目默认专业
	Quantity     float64  `json:"quantity"`
	QuantityExpr string   `json:"quantity_expr"`
	QuotaCode    *string  `json:"quota_code"`
	QuotaCodes   []string `json:"quota_codes"`
	// QuotaFactors 定额换算系数（定额编码 → 每一个清单单位包含几个定额单位），须与
	// quota_code / quota_codes 同批提交：挂接量 = 清单工程量 × 系数。没列的定额沿用
	// 该行原有挂接的比例，没有原挂接 = 跟随清单工程量。
	QuotaFactors  map[string]float64 `json:"quota_factors"`
	Feature       string             `json:"feature"`
	PriceOverride float64            `json:"price_override"`
	ListPrice     *float64           `json:"list_price"`
}

// anyRate 是否至少传了一个费率（partial 传参合法：没传的按 0=不计取）。
func (p costProjectApplyArgs) anyRate() bool {
	return p.ManagementRate != nil || p.RegulatoryRate != nil || p.ProfitRate != nil || p.TaxRate != nil
}

// rateValues 四费率取值，未传的按 0（显式不计取）。
func (p costProjectApplyArgs) rateValues() workcost.RateSet {
	return workcost.RateSet{
		ManagementRate: p.floatOr(p.ManagementRate, 0), RegulatoryRate: p.floatOr(p.RegulatoryRate, 0),
		ProfitRate: p.floatOr(p.ProfitRate, 0), TaxRate: p.floatOr(p.TaxRate, 0),
	}
}

func (p costProjectApplyArgs) boolOr(v *bool, def bool) bool {
	if v != nil {
		return *v
	}
	return def
}

func (p costProjectApplyArgs) floatOr(v *float64, def float64) float64 {
	if v != nil {
		return *v
	}
	return def
}

// linksGiven 该行是否提交了定额挂接字段（提交了=整体替换该行的挂接集合）。
func (it costProjectApplyItem) linksGiven() bool {
	return it.QuotaCode != nil || it.QuotaCodes != nil || it.QuotaFactors != nil
}

// linkInputs 挂接集合（主定额在前，其余保序；空码与重复码剔除）。
//
// 挂接量 = 清单工程量 × 换算系数：系数取 quota_factors；没列的定额沿用该行原有挂接
// 的比例（原挂接量 ÷ 原清单工程量）——重发一行只改名称、或导入时带进来的系数 ≠1，
// 都不会被整体替换清成「跟随」。没有原挂接、或系数恰为 1 = 0（跟随清单工程量）。
// existing / old = 该行落库前的样子（新行传 nil）。
func (it costProjectApplyItem) linkInputs(existing *workcost.BillItem, old []workcost.BillQuotaLink) []workcost.BillQuotaLinkInput {
	oldQty := map[string]float64{}
	for _, l := range old {
		oldQty[strings.TrimSpace(l.QuotaCode)] = l.Quantity
	}
	var out []workcost.BillQuotaLinkInput
	seen := map[string]bool{}
	add := func(c string) {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			return
		}
		seen[c] = true
		f, listed := it.QuotaFactors[c]
		if !listed && existing != nil && existing.Quantity > 0 && oldQty[c] > 0 {
			f = oldQty[c] / existing.Quantity
		}
		var q float64
		if it.Quantity > 0 && f > 0 && f != 1 {
			q = it.Quantity * f
		}
		out = append(out, workcost.BillQuotaLinkInput{QuotaCode: c, Quantity: q})
	}
	if it.QuotaCode == nil && it.QuotaCodes == nil {
		// 只给了系数、没给挂接集合：把系数套在该行原有挂接上（新行没有原挂接，
		// 写前校验已拦下）。老数据可能只有单值主定额、没有挂接行——以它为准，
		// 别把这一行清成无定额。
		for _, l := range old {
			add(l.QuotaCode)
		}
		if len(out) == 0 && existing != nil {
			add(existing.QuotaCode)
		}
		return out
	}
	if it.QuotaCode != nil {
		add(*it.QuotaCode)
	}
	for _, c := range it.QuotaCodes {
		add(c)
	}
	return out
}

// badFactor 该行的换算系数有没有不合法的（负数）；有则返回报错文案。
func (it costProjectApplyItem) badFactor(row int) string {
	for code, f := range it.QuotaFactors {
		if f < 0 {
			return fmt.Sprintf("第 %d 行定额 %s 的换算系数 %g 不能为负。整批未写入", row, code, f)
		}
	}
	return ""
}

// clProjectScope 项目口径的回显片段：专业/地区/计价期（一个都没给=空串）。
func clProjectScope(specialty, region, period string) string {
	specialty = strings.TrimSpace(specialty)
	region = strings.TrimSpace(region)
	period = strings.TrimSpace(period)
	if specialty == "" && region == "" && period == "" {
		return ""
	}
	return fmt.Sprintf("｜专业 %s｜地区 %s｜计价期 %s", clDim(specialty, "未标注"), clDim(region, "未标注"), clDim(period, "未标注"))
}

// findBillProject 按 id 找项目封面（含费率）；不存在返回 nil。
func findBillProject(s *workcost.Store, id int64) *workcost.BillProject {
	projects, err := s.BillProjects()
	if err != nil {
		return nil
	}
	for i := range projects {
		if projects[i].ID == id {
			return &projects[i]
		}
	}
	return nil
}

func (t costProjectApplyTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p costProjectApplyArgs
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
		rates := &workcost.RateSet{}
		if p.anyRate() {
			full := p.rateValues()
			rates = &full
		} else if projects, err := s.BillProjects(); err == nil {
			// 一个费率都没传：新建项目落显式全零（0=不计取，不猜默认值）；同名项目
			// 走封面更新时不动它已存的费率。
			for i := range projects {
				if projects[i].Name == strings.TrimSpace(p.Name) {
					rates = nil
					break
				}
			}
		}
		// 项目的专业/地区/计价期是规范值（库定规矩）：专业、地区必须在字典里，期数
		// YYYY-MM；不给=保持原值（待整理）。先过规矩再建封面——口径被拒不留半截封面。
		specialty, region := strings.TrimSpace(p.Specialty), strings.TrimSpace(p.Region)
		period, err := s.ValidateProjectScope(specialty, region, strings.TrimSpace(p.PricePeriod))
		if err != nil {
			return "", fmt.Errorf("项目口径不合规矩，未建档：%w", err)
		}
		id, err := s.UpsertBillProject(p.Name, p.FileName, p.Source, p.Location, p.Duration, p.Pricing, rates, p.boolOr(p.ProfitIncludesRegulatory, false), p.floatOr(p.ControlPrice, 0))
		if err != nil {
			return "", err
		}
		if err := s.UpdateBillProjectProfile(id, specialty, region, period); err != nil {
			return "", fmt.Errorf("项目封面已建（#%d），但专业/地区/计价期未写入：%w", id, err)
		}
		return fmt.Sprintf("项目已建档：#%d %s（费率按提交值落库，0=该段显式不计取）%s。下一步 action=save_items 编制清单。",
			id, p.Name, clProjectScope(specialty, region, period)), nil
	case "update_rates":
		if p.ProjectID <= 0 {
			return "", fmt.Errorf("update_rates 需要 project_id")
		}
		if !p.anyRate() && p.ProfitIncludesRegulatory == nil && p.ControlPrice == nil {
			return "", fmt.Errorf("update_rates 需要至少一个费率字段（management_rate/regulatory_rate/profit_rate/tax_rate/profit_includes_regulatory/control_price）")
		}
		proj := findBillProject(s, p.ProjectID)
		if proj == nil {
			return "", fmt.Errorf("项目 #%d 不存在（cost_project action=projects 可查）", p.ProjectID)
		}
		// 只改传入的字段，其余保持原值。
		rates := workcost.RateSet{
			ManagementRate: proj.ManagementRate, RegulatoryRate: proj.RegulatoryRate,
			ProfitRate: proj.ProfitRate, TaxRate: proj.TaxRate,
		}
		if p.ManagementRate != nil {
			rates.ManagementRate = *p.ManagementRate
		}
		if p.RegulatoryRate != nil {
			rates.RegulatoryRate = *p.RegulatoryRate
		}
		if p.ProfitRate != nil {
			rates.ProfitRate = *p.ProfitRate
		}
		if p.TaxRate != nil {
			rates.TaxRate = *p.TaxRate
		}
		if err := s.UpdateBillProjectRates(p.ProjectID, rates,
			p.boolOr(p.ProfitIncludesRegulatory, proj.ProfitIncludesRegulatory),
			p.floatOr(p.ControlPrice, proj.ControlPrice)); err != nil {
			return "", err
		}
		return fmt.Sprintf("项目 #%d 费率已更新：管=%g 规=%g 利=%g 税=%g（0=显式不计取；未传的字段保持原值）。",
			p.ProjectID, rates.ManagementRate, rates.RegulatoryRate, rates.ProfitRate, rates.TaxRate), nil
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
		quotaSpec := map[string]string{} // 定额码 → 专业（回执点名外专业挂接用）
		proj := findBillProject(s, p.ProjectID)
		if proj == nil {
			return "", fmt.Errorf("项目 #%d 不存在（cost_project action=projects 可查）", p.ProjectID)
		}
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
			if msg := it.badFactor(i + 1); msg != "" {
				return "", fmt.Errorf("%s", msg)
			}
			if it.QuotaFactors != nil && it.QuotaCode == nil && it.QuotaCodes == nil {
				// 只给系数 = 套在该行原有挂接上：新行没有原挂接可套。
				got, err := s.FindBillItem(p.ProjectID, strings.TrimSpace(it.Code))
				if err != nil || got == nil {
					return "", fmt.Errorf("第 %d 行只给了 quota_factors、没给 quota_code / quota_codes，而它不是已有行——换算系数须与挂接集合同批提交。整批未写入", i+1)
				}
			}
			for _, in := range it.linkInputs(nil, nil) {
				c := in.QuotaCode
				q, err := s.GetQuota(c)
				if workcostNotFound(err) {
					return "", fmt.Errorf("第 %d 行定额 %s 不存在（整批未写入）：核对编码或让用户提供，不要编造定额号", i+1, c)
				} else if err != nil {
					return "", fmt.Errorf("第 %d 行定额 %s 校验失败: %w（整批未写入）", i+1, c, err)
				}
				quotaSpec[c] = strings.TrimSpace(q.Specialty)
			}
		}
		saved, droppedLinks, crossSpecialty := 0, 0, 0
		factorRows := 0 // 挂接量带换算系数的行数（回执点名，免得模型以为都是跟随清单工程量）
		for i, it := range p.Items {
			code := strings.TrimSpace(it.Code)
			item := workcost.BillItem{
				ProjectID: p.ProjectID, Code: code,
				Title: strings.TrimSpace(it.Title), Unit: strings.TrimSpace(it.Unit),
				Division: it.Division, Quantity: it.Quantity, QuantityExpr: it.QuantityExpr,
				Feature:       it.Feature,
				PriceOverride: it.PriceOverride,
			}
			// 已有行：没传的字段保持原值——原表单价快照（2026-11-25 修：此前整行
			// 重写会把 list_price 清零）、没提交挂接字段时的主定额、已归类的专业。
			var existing *workcost.BillItem
			var oldLinks []workcost.BillQuotaLink // 该行原有挂接：没给系数的定额沿用它的比例
			if code != "" {
				got, err := s.FindBillItem(p.ProjectID, code)
				if err != nil {
					return "", fmt.Errorf("第 %d 行查重失败: %w（整批未写入）", i+1, err)
				}
				existing = got
				if existing != nil {
					if old, lerr := s.BillQuotaLinks(existing.ID); lerr == nil {
						oldLinks = old
					}
				}
			}
			if existing != nil {
				item.ListPrice = existing.ListPrice
				item.QuotaCode = existing.QuotaCode
				item.Specialty = existing.Specialty
			}
			if it.ListPrice != nil {
				item.ListPrice = *it.ListPrice
			}
			if it.Specialty != nil {
				item.Specialty = strings.TrimSpace(*it.Specialty)
			}
			links := it.linkInputs(existing, oldLinks)
			if it.linksGiven() {
				// 提交了挂接字段=整体替换；主定额列先写集合首项，落库后同步引用表。
				item.QuotaCode = ""
				if len(links) > 0 {
					item.QuotaCode = links[0].QuotaCode
				}
			}
			item.QuotaCodes = []string{} // 构造即非 nil（JSON null 崩前端）
			savedItem, err := s.SaveBillItem(item)
			if err != nil {
				return "", fmt.Errorf("第 %d 行写入失败（前面行已落库，修复该行后重新提交同批即可按编码幂等更新）: %w", saved+1, err)
			}
			if it.linksGiven() {
				// 原有挂接里不在新集合中的=被解除。回执点名：模型只带回主定额时，
				// 1:N 的其余挂接会在这里静默丢掉。
				if existing != nil {
					keep := map[string]bool{}
					for _, in := range links {
						keep[in.QuotaCode] = true
					}
					for _, l := range oldLinks {
						if !keep[l.QuotaCode] {
							droppedLinks++
						}
					}
				}
				if _, err := s.SaveBillItemLinks(savedItem.ID, links); err != nil {
					return "", fmt.Errorf("第 %d 行定额挂接写入失败（清单行已落库，修正后重发该行即可）: %w", i+1, err)
				}
				for _, in := range links {
					if in.Quantity > 0 {
						factorRows++
						break
					}
				}
				// 同专业优先（规矩，不拦只点名）：清单专业与挂接定额专业对不上，
				// 说明没先借鉴生成本专业定额。
				rowSpec := strings.TrimSpace(item.Specialty)
				if rowSpec == "" {
					rowSpec = strings.TrimSpace(proj.Specialty)
				}
				for _, in := range links {
					if qs := quotaSpec[in.QuotaCode]; qs != "" && rowSpec != "" && qs != rowSpec {
						crossSpecialty++
						break
					}
				}
			}
			saved++
		}
		receipt := fmt.Sprintf("项目 #%d 清单已写入 %d/%d 行（写前校验：编码唯一/工程量>0/定额存在）。用 cost_audit 跑机检后再交付。", p.ProjectID, saved, len(p.Items))
		if droppedLinks > 0 {
			receipt += fmt.Sprintf("\n注意：本次按整体替换解除了 %d 条原有定额挂接（不在提交的 quota_code/quota_codes 里）。若非本意，把原挂接补回后重发对应行。", droppedLinks)
		}
		if crossSpecialty > 0 {
			receipt += fmt.Sprintf("\n注意：%d 行挂接了别的专业的定额（清单专业与定额专业不一致）——清单套定额同专业优先：用 cost_library borrow_quota 以外专业定额为底生成本专业定额，再把这些行改挂过去。cost_audit 会点名。", crossSpecialty)
		}
		if factorRows > 0 {
			receipt += fmt.Sprintf("\n其中 %d 行的定额挂接量 = 清单工程量 × 换算系数（quota_factors；没给系数的定额沿用该行原有挂接的比例）。要让某条定额改回跟随清单工程量，给它传系数 1。", factorRows)
		}
		return receipt, nil
	default:
		return "", fmt.Errorf("未知 action %q（create_project/update_rates/save_items）", p.Action)
	}
}

// ── cost_audit：只读确定性机检 ───────────────────────────────────────────

type costAuditTool struct{ a *App }

func (t costAuditTool) Name() string   { return "cost_audit" }
func (t costAuditTool) ReadOnly() bool { return true }
func (t costAuditTool) Description() string {
	return "对测算项目跑确定性机检（只读，不改数据）：行级硬检=身份/单位缺失、无计价路径（无定额且无手填价）、定额悬空、资源悬空（HardCount>0 必须修复后再交付；资源悬空回执附可改的回存骨架）；信息项不阻塞=费率>1、规格档错配、引用的资源属于别的地区（region_mismatch）、跨项目查漏（只在同专业间比）、清单与定额专业对不上（specialty_gap：清单挂着外专业定额，应先借鉴生成本专业的；code_gap：国标清单编码的专业段与清单专业不一致）、库级健康扫描。费率≤0 是显式不计取，不报错。"
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
	quotaOrder := []string{}              // 定额码出场序（跨地区检查按它遍历，回执稳定）
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
				quotaOrder = append(quotaOrder, c)
			}
			if quotaCache[c] == nil && !reportedDangling[c] {
				reportedDangling[c] = true
				add("dangling_quota", fmt.Sprintf("%s %s 引用定额 %s 不存在", row, it.Title, c), true)
			}
		}
	}
	// 硬检四：资源悬空——仅本项目已挂定额（且定额存在）的含量行，ResourceCode
	// 非空但资源表查不到。资源缺失时快照价兜底可追溯（QuotaItem.ResourcePrice），
	// 但引用面断裂仍是硬伤。回执附可应用的修法骨架（2026-11-26 造价反馈，百锦路
	// #8：此前只点名，模型得先拼出整组含量行再 save_quota，每条分两次手工修）。
	checkedRes := map[string]bool{}
	missingRes := map[string]bool{}
	danglingByQuota := map[string][]string{} // 定额码 → 悬空的资源码
	var danglingQuotas []string
	for _, it := range items {
		codes := append([]string{it.QuotaCode}, it.QuotaCodes...)
		for _, c := range codes {
			c = strings.TrimSpace(c)
			if c == "" || quotaCache[c] == nil {
				continue
			}
			for _, qi := range quotaCache[c].Items {
				rc := strings.TrimSpace(qi.ResourceCode)
				if rc == "" {
					continue
				}
				if !checkedRes[rc] {
					checkedRes[rc] = true
					if _, err := s.GetResourceByCode(rc); err != nil && workcostNotFound(err) {
						missingRes[rc] = true
					} else if err != nil {
						return "", fmt.Errorf("资源 %s 校验失败: %w", rc, err)
					}
				}
				if !missingRes[rc] {
					continue
				}
				if _, dup := danglingByQuota[c]; !dup || !clHas(danglingByQuota[c], rc) {
					if !dup {
						danglingQuotas = append(danglingQuotas, c)
					}
					danglingByQuota[c] = append(danglingByQuota[c], rc)
				}
			}
		}
	}
	for _, c := range danglingQuotas {
		add("dangling_resource", fmt.Sprintf("定额 %s 含量行引用资源 %s 不存在%s", c, strings.Join(danglingByQuota[c], "、"), clDanglingFix(quotaCache[c], danglingByQuota[c])), true)
	}
	// 跨地区一致性（2026-11-26 造价反馈，百锦路 #8）：定额引用的资源属于别的地区
	// ——价是那个地区的，与本项目口径不符。算价链按编码取价、不看地区，此前只能
	// 靠模型自己的成本经验发现（实测 ALLU 借了铜陵行 2673.30，而本地区口径是 800）。
	// 按项目定额的出场序逐条看，同一资源只报一次。
	type foreignRef struct {
		res    *workcost.Resource
		quotas []string
	}
	foreignOrder := []string{}
	foreign := map[string]*foreignRef{}
	if strings.TrimSpace(proj.Region) != "" {
		for _, c := range quotaOrder {
			q := quotaCache[c]
			if q == nil {
				continue
			}
			for _, qi := range q.Items {
				rc := strings.TrimSpace(qi.ResourceCode)
				if rc == "" || missingRes[rc] {
					continue // 悬空的由硬检点名
				}
				fr, seen := foreign[rc]
				if !seen {
					r, err := s.GetResourceByCode(rc)
					if err != nil {
						if workcostNotFound(err) {
							continue
						}
						return "", fmt.Errorf("资源 %s 校验失败: %w", rc, err)
					}
					if r == nil || strings.TrimSpace(r.Region) == "" || strings.TrimSpace(r.Region) == strings.TrimSpace(proj.Region) {
						continue
					}
					fr = &foreignRef{res: r}
					foreign[rc] = fr
					foreignOrder = append(foreignOrder, rc)
				}
				if !clHas(fr.quotas, c) {
					fr.quotas = append(fr.quotas, c)
				}
			}
		}
	}

	// 规格档错配检（2026-11-21 造价门实测 4.2/六：C60 混凝土被当 C20 用进表，
	// 虚高 1.07 万——title 里的强度档与清单特征档比对是确定性可判的。INFO 级：
	// 暂代（库内无同档，如 C10-C20 代 C20）是合法操作，报警给用户确认而非阻断）。
	for _, it := range items {
		itemGrades := costSpecGrades(it.Title + " " + it.Feature)
		if len(itemGrades) == 0 {
			continue
		}
		quotaGrades := map[string]bool{}
		for _, c := range append([]string{it.QuotaCode}, it.QuotaCodes...) {
			if c = strings.TrimSpace(c); c == "" || quotaCache[c] == nil {
				continue
			}
			for g := range costSpecGrades(quotaCache[c].Title) {
				quotaGrades[g] = true
			}
			for _, qi := range quotaCache[c].Items {
				for g := range costSpecGrades(qi.Title + " " + qi.Spec) {
					quotaGrades[g] = true
				}
			}
		}
		if len(quotaGrades) == 0 {
			continue
		}
		match := false
		for g := range itemGrades {
			if quotaGrades[g] {
				match = true
				break
			}
		}
		if !match {
			ig := make([]string, 0, len(itemGrades))
			for g := range itemGrades {
				ig = append(ig, g)
			}
			sort.Strings(ig)
			qg := make([]string, 0, len(quotaGrades))
			for g := range quotaGrades {
				qg = append(qg, g)
			}
			sort.Strings(qg)
			add("spec_grade_mismatch", fmt.Sprintf("#%d[%s] %s 清单特征档 %v 与定额/资源档 %v 不一致——确认规格档或换资源（暂代需在交付说明注明）", it.ID, it.Code, it.Title, ig, qg), false)
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
	// 数据指纹（2026-11-21 造价门实测 5.1：同轮改数据后复检，审计结果可能逐字节
	// 相同而被上层「结果同一性去重」折叠成 cached 提示——指纹让新旧结果可分辨；
	// 改费率/改行数必变指纹）。
	fmt.Fprintf(&b, "数据指纹：清单 %d 行｜费率 管=%g 规=%g 利=%g 税=%g\n", len(items),
		proj.ManagementRate, proj.RegulatoryRate, proj.ProfitRate, proj.TaxRate)
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
			fmt.Fprintf(&b, "- [INFO] rate_over_one: %s 费率 %g>1——库按「倍数」乘算（5 就是 5 倍）；本意若是 5%% 应填 0.05，确认后去 cost_project_apply 改费率\n", r.name, r.v)
			infoCount++
		}
	}
	// 跨地区：被本项目定额引用的、属于别的地区的资源。
	for i, rc := range foreignOrder {
		if i >= 20 {
			fmt.Fprintf(&b, "- [INFO] region_mismatch: …另有 %d 个资源属于别的地区\n", len(foreignOrder)-20)
			infoCount++
			break
		}
		fr := foreign[rc]
		fmt.Fprintf(&b, "- [INFO] region_mismatch: 资源 %s「%s」属「%s」，与项目地区「%s」不同，被本项目定额 %s 引用——价是那个地区的；本地区有同身份的行就换码（cost_project action=resources keyword=名称），没有就 cost_library save_resource 建本地区的行\n",
			rc, fr.res.Title, fr.res.Region, strings.TrimSpace(proj.Region), strings.Join(fr.quotas, "、"))
		infoCount++
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
		// 清单与定额专业对不上（清单套定额同专业优先）：同样一件事各专业各有定额。
		for _, f := range scan.SpecialtyGaps {
			fmt.Fprintf(&b, "- [INFO] %s: %s（%s）\n", f.Kind, f.Detail, f.Target)
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
		b.WriteString("硬检全绿，可进入交付。")
	} else {
		b.WriteString("存在硬检问题：修复后重新机检，不要带硬伤交付。")
	}
	return b.String(), nil
}

// clHas 报告 s 是否含 v。
func clHas(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// clDanglingFix 资源悬空的可应用修法（2026-11-26 造价反馈，百锦路 #8）：此前只
// 点名，模型得先把整组含量行拼出来再 save_quota，漏一行就把别的含量行冲掉。
// 给出「整组含量行原样 + 悬空行标出」的骨架，改一个 resource_code 即可回存。
func clDanglingFix(q *workcost.Quota, dangling []string) string {
	if q == nil {
		return ""
	}
	bad := map[string]bool{}
	for _, d := range dangling {
		bad[d] = true
	}
	items := make([]string, 0, len(q.Items))
	for i, it := range q.Items {
		if i >= 12 {
			items = append(items, fmt.Sprintf("…（另有 %d 行，回存时原样带上）", len(q.Items)-12))
			break
		}
		rc := strings.TrimSpace(it.ResourceCode)
		if bad[rc] {
			rc += "←改成在库编码"
		}
		items = append(items, fmt.Sprintf(`{"resource_code":"%s","quantity":%g}`, rc, it.Quantity))
	}
	return fmt.Sprintf("\n    修法二选一：① 资源确实要新建——cost_library action=save_resource 建 %s；② 编码写错——把下面骨架里标「←」的 resource_code 改成在库编码后整组回存（save_quota 的 items 是整组替换，其余行务必原样带上）：\n    {\"action\":\"save_quota\",\"quota\":{\"code\":\"%s\",\"items\":[%s]}}",
		strings.Join(dangling, "、"), q.Code, strings.Join(items, ","))
}

// costSpecGrades 抽取文本里的规格强度档（2026-11-21 造价门实测 4.2：C60 充当
// C20 用进表的确定性防线）。覆盖混凝土 C10-C80 与砂浆 M5-M30 两族（其余档位
// 体系（沥青 AC/钢筋 HPB）形态歧义大，不冒充全量识别）；边界锚定防 PC200/
// M046 编号误报。
var costSpecGradeRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9])([CM])(\d{1,3})`)

func costSpecGrades(text string) map[string]bool {
	out := map[string]bool{}
	for _, m := range costSpecGradeRe.FindAllStringSubmatch(text, -1) {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		switch m[1] {
		case "C":
			if n >= 10 && n <= 80 { // 混凝土强度等级（三位数捕获后域内校验，等价前瞻）
				out["C"+strconv.Itoa(n)] = true
			}
		case "M":
			if n >= 5 && n <= 30 { // 砂浆强度等级
				out["M"+strconv.Itoa(n)] = true
			}
		}
	}
	return out
}
