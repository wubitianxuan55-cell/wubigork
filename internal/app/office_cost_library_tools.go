package app

// office_cost_library_tools.go — 造价库维护工具 cost_library（2026-11-25 造价数据库
// 推倒重建：板块只留浏览/检索/删除/导出，导入、解析、维护全部归办公「造价」
// 运行模式，与方案库「录入在方案模式、板块只管看」同一模式）。
//
// 三级数据 清单 → 定额 → 工料机 的写入口分工：
//   - cost_project_apply：测算编制的增量写（建项目/改费率/写清单行），只增不删；
//   - cost_library（本文件）：导入五表工作簿、定额与工料机主数据增删改调价、
//     批量调价、资源化、自生成企业定额、去重、删清单行/删项目、库级健康明细。
//
// 纪律：
//   - 标 PersistWrite（子代理不继承）；删除与批量覆盖类动作逐条过用户确认卡
//     （controller hardAsk），dry_run 演练不弹卡；
//   - 被引用的定额/工料机默认不删（force=true 才连带解除引用），不留悬空引用；
//   - 批量动作先全量校验再写入：任一条不合规整批拒绝，不留半批；
//   - 定额含量行给的资源编码必须在库——查不到就报错，绝不编造；
//   - 数据面一律锚 workcost（工料机 × 定额 = 综合单价，活重算链）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

type costLibraryTool struct{ a *App }

func (costLibraryTool) Name() string       { return "cost_library" }
func (costLibraryTool) ReadOnly() bool     { return false }
func (costLibraryTool) PersistWrite() bool { return true }

func (costLibraryTool) Description() string {
	return "维护工料机造价库（三级数据：清单 → 定额 → 工料机）。import_project 导入五表成本测算工作簿（项目+清单+费率+工料机+定额一次落库）；save_quota/delete_quota 维护消耗定额（含量行整组替换）；save_resource/delete_resource/set_price 维护工料机（调价写历史并推进现行价，引用它的定额综合单价自动重算）；bulk_price 批量调价（名称+单位+规格精确匹配，歧义跳过）；delete_items/delete_project 删清单行/整个项目；seed 把资料条目资源化；flywheel 跨项目含量聚合生成企业定额；dedupe 合并重复工料机；health 列库级健康发现。批量与删除动作先 dry_run=true 演练（不写库、不弹确认），核对后去掉 dry_run 正式执行；被引用的定额/工料机默认不删，确要删加 force=true。每次正式写入都需要用户确认。造价数据库板块只读，改库一律经本工具或 cost_project_apply，不要直连数据库。"
}

func (costLibraryTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"required":["action"],
"properties":{
  "action":{"type":"string","enum":["import_project","save_quota","delete_quota","save_resource","delete_resource","set_price","bulk_price","delete_items","delete_project","seed","flywheel","dedupe","health"],"description":"见工具说明；health 不需要其它参数"},
  "dry_run":{"type":"boolean","description":"演练：校验、匹配、统计照做，但不写库、不弹确认"},
  "path":{"type":"string","description":"五表工作簿路径（import_project 必填，.xlsx）"},
  "quota":{"type":"object","description":"定额（save_quota 必填）","properties":{
    "code":{"type":"string","description":"定额编码（必填；已存在=更新）"},
    "title":{"type":"string","description":"定额名称"},
    "unit":{"type":"string","description":"计量单位"},
    "specialty":{"type":"string","description":"专业"},
    "chapter":{"type":"string","description":"章节"},
    "category_path":{"type":"string","description":"分类路径"},
    "source":{"type":"string","description":"来源（手工录入/企业定额…）"},
    "note":{"type":"string","description":"备注"},
    "items":{"type":"array","description":"工料机含量行（整组替换；每行引用一个在库的工料机编码；不给=保留原有含量行；新建定额必须给，可为空数组）","items":{"type":"object","properties":{
      "resource_code":{"type":"string","description":"工料机编码（必填，必须在库；需要新资源先用 save_resource 建）"},
      "quantity":{"type":"number","description":"含量（0=工序模板占位，不计费）"},
      "loss_rate":{"type":"number","description":"损耗率（0.03=3%）"},
      "note":{"type":"string","description":"备注"}
    }}
  }},
  "resource":{"type":"object","description":"工料机（save_resource 必填；按 id 或 code 定位已有资源，未给的字段保持原值）","properties":{
    "id":{"type":"integer","description":"资源 id（改已有资源时给）"},
    "code":{"type":"string","description":"资源编码（改已有资源时给；新建可留空自动分配）"},
    "kind":{"type":"string","description":"类别：人工/材料/机械/外委"},
    "title":{"type":"string","description":"名称"},
    "spec":{"type":"string","description":"规格"},
    "unit":{"type":"string","description":"单位：工日/t/m³/台班"},
    "base_price":{"type":"number","description":"基准价（元）"},
    "current_price":{"type":"number","description":"现行价（元；核算取用价，现行价优先）"},
    "category_path":{"type":"string","description":"分类路径"},
    "source":{"type":"string","description":"来源：信息价/市场询价/企业定额"},
    "supplier":{"type":"string","description":"供应商"},
    "region":{"type":"string","description":"地区"},
    "price_date":{"type":"string","description":"价格期数，如 2026年第7期"},
    "price_type":{"type":"string","description":"价格口径：到场价/信息价/含税…"},
    "valid_until":{"type":"string","description":"有效期至（YYYY-MM-DD，可空）"},
    "loss_rate":{"type":"number","description":"损耗率（0.03=3%）"},
    "note":{"type":"string","description":"备注/计算式"},
    "tags":{"type":"array","items":{"type":"string"},"description":"标签"},
    "status":{"type":"string","description":"状态：现行/停用"}
  }},
  "resource_id":{"type":"integer","description":"工料机 id（set_price 用；与 resource_code 二选一）"},
  "resource_code":{"type":"string","description":"工料机编码（set_price 用；与 resource_id 二选一）"},
  "price":{"type":"number","description":"新单价（set_price 必填，元）"},
  "price_type":{"type":"string","description":"调价口径（set_price 可选，默认 到场价）"},
  "period":{"type":"string","description":"期数（set_price 可选）"},
  "region":{"type":"string","description":"地区（set_price 可选）"},
  "note":{"type":"string","description":"调价依据（set_price 可选）"},
  "rows":{"type":"array","description":"批量调价行（bulk_price 必填，单批 ≤500）","items":{"type":"object","properties":{
    "title":{"type":"string","description":"资源名称（精确匹配）"},
    "spec":{"type":"string","description":"规格（给了就参与精确匹配）"},
    "unit":{"type":"string","description":"单位（给了就参与精确匹配）"},
    "price":{"type":"number","description":"新单价（元）"},
    "period":{"type":"string","description":"期数"},
    "region":{"type":"string","description":"地区"},
    "note":{"type":"string","description":"备注"}
  }}},
  "codes":{"type":"array","items":{"type":"string"},"description":"编码列表：delete_quota 填定额编码；delete_resource 填工料机编码"},
  "ids":{"type":"array","items":{"type":"integer"},"description":"工料机 id 列表（delete_resource 用，与 codes 二选一）"},
  "force":{"type":"boolean","description":"仍被引用时一并解除引用（清单项退回手填行/定额含量行悬空），默认 false=拒绝"},
  "project_id":{"type":"integer","description":"项目 id（delete_items/delete_project 必填）"},
  "item_ids":{"type":"array","items":{"type":"integer"},"description":"清单行 id 列表（delete_items 必填）"},
  "stems":{"type":"array","items":{"type":"string"},"description":"要生成的聚类词干 orig_code（flywheel，取自 dry_run 预览的 origCode）"},
  "keep_id":{"type":"integer","description":"保留的工料机 id（dedupe 必填）"},
  "merge_ids":{"type":"array","items":{"type":"integer"},"description":"合并掉的工料机 id（dedupe 必填）"}
}
}`)
}

// costLibraryArgs cost_library 入参。需要区分「没传」的嵌套字段用指针。
type costLibraryArgs struct {
	Action string `json:"action"`
	DryRun bool   `json:"dry_run"`
	Path   string `json:"path"`

	Quota    *costLibraryQuota       `json:"quota"`
	Resource *costLibraryResource    `json:"resource"`
	Rows     []workcost.BulkPriceRow `json:"rows"`
	Codes    []string                `json:"codes"`
	IDs      []int64                 `json:"ids"`
	Force    bool                    `json:"force"`

	ResourceID   int64    `json:"resource_id"`
	ResourceCode string   `json:"resource_code"`
	Price        *float64 `json:"price"`
	PriceType    string   `json:"price_type"`
	Period       string   `json:"period"`
	Region       string   `json:"region"`
	Note         string   `json:"note"`

	ProjectID int64   `json:"project_id"`
	ItemIDs   []int64 `json:"item_ids"`

	Stems    []string `json:"stems"`
	KeepID   int64    `json:"keep_id"`
	MergeIDs []int64  `json:"merge_ids"`
}

type costLibraryQuota struct {
	Code         string                  `json:"code"`
	Title        *string                 `json:"title"`
	Unit         *string                 `json:"unit"`
	Specialty    *string                 `json:"specialty"`
	Chapter      *string                 `json:"chapter"`
	CategoryPath *string                 `json:"category_path"`
	Source       *string                 `json:"source"`
	Note         *string                 `json:"note"`
	Items        *[]costLibraryQuotaItem `json:"items"`
}

type costLibraryQuotaItem struct {
	ResourceCode string  `json:"resource_code"`
	Quantity     float64 `json:"quantity"`
	LossRate     float64 `json:"loss_rate"`
	Note         string  `json:"note"`
}

type costLibraryResource struct {
	ID           int64    `json:"id"`
	Code         string   `json:"code"`
	Kind         *string  `json:"kind"`
	Title        *string  `json:"title"`
	Spec         *string  `json:"spec"`
	Unit         *string  `json:"unit"`
	BasePrice    *float64 `json:"base_price"`
	CurrentPrice *float64 `json:"current_price"`
	CategoryPath *string  `json:"category_path"`
	Source       *string  `json:"source"`
	Supplier     *string  `json:"supplier"`
	Region       *string  `json:"region"`
	PriceDate    *string  `json:"price_date"`
	PriceType    *string  `json:"price_type"`
	ValidUntil   *string  `json:"valid_until"`
	LossRate     *float64 `json:"loss_rate"`
	Note         *string  `json:"note"`
	Tags         []string `json:"tags"`
	Status       *string  `json:"status"`
}

// costLibraryMaxRows 批量调价的单批上限。
const costLibraryMaxRows = 500

// costLibraryActionLabel 动作中文名（回执头与确认卡摘要共用口径）。
var costLibraryActionLabel = map[string]string{
	"import_project": "导入五表工作簿", "save_quota": "保存定额", "delete_quota": "删除定额",
	"save_resource": "保存工料机", "delete_resource": "删除工料机", "set_price": "工料机调价",
	"bulk_price": "批量调价", "delete_items": "删除清单行", "delete_project": "删除项目",
	"seed": "资料条目资源化", "flywheel": "自生成企业定额", "dedupe": "工料机去重", "health": "库级健康扫描",
}

func (t costLibraryTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p costLibraryArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("cost_library 参数解析失败: %w", err)
	}
	if _, ok := costLibraryActionLabel[p.Action]; !ok {
		return "", fmt.Errorf("未知 action %q（import_project/save_quota/delete_quota/save_resource/delete_resource/set_price/bulk_price/delete_items/delete_project/seed/flywheel/dedupe/health）", p.Action)
	}
	s := t.a.hubWorkcostStore()
	if !s.Available() {
		return "", fmt.Errorf("工料机造价库不可用（Hephaestus.db 未就绪）")
	}
	switch p.Action {
	case "import_project":
		return t.importProject(s, p)
	case "save_quota":
		return t.saveQuota(s, p)
	case "delete_quota":
		return t.deleteQuota(s, p)
	case "save_resource":
		return t.saveResource(s, p)
	case "delete_resource":
		return t.deleteResource(s, p)
	case "set_price":
		return t.setPrice(s, p)
	case "bulk_price":
		return t.bulkPrice(s, p)
	case "delete_items":
		return t.deleteItems(s, p)
	case "delete_project":
		return t.deleteProject(s, p)
	case "seed":
		return t.seed(s, p)
	case "flywheel":
		return t.flywheel(s, p)
	case "dedupe":
		return t.dedupe(s, p)
	default: // health
		return t.health(s)
	}
}

// ── 小工具 ───────────────────────────────────────────────────────────

func clMoney(v float64) string { return fmt.Sprintf("¥%.2f", v) }

func clPct(v float64) string {
	if v <= 0 {
		return "不计取"
	}
	return fmt.Sprintf("%.2f%%", v*100)
}

// clHead 回执首行：演练 / 正式。
func clHead(dryRun bool, action string) string {
	if dryRun {
		return "🔍 演练（未写库）：" + costLibraryActionLabel[action] + "\n"
	}
	return "✅ 已完成：" + costLibraryActionLabel[action] + "\n"
}

// clCap 明细列举行数上限。
func clCap(n int) int {
	if n > 12 {
		return 12
	}
	return n
}

// clCleanCodes 去空白、去空、去重保序。
func clCleanCodes(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, c := range in {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

// clResolveResource 按 id 或编码定位工料机。
func clResolveResource(s *workcost.Store, id int64, code string) (*workcost.Resource, error) {
	if id > 0 {
		return s.GetResource(id)
	}
	if c := strings.TrimSpace(code); c != "" {
		return s.GetResourceByCode(c)
	}
	return nil, fmt.Errorf("需要 resource_id 或 resource_code（cost_project action=resources 可查）")
}

// sameResourceIdentity 身份四要素相等（类别按库内归一口径比较）。
func sameResourceIdentity(a, b workcost.Resource) bool {
	return workcost.NormalizeKind(a.Kind) == workcost.NormalizeKind(b.Kind) &&
		strings.TrimSpace(a.Title) == strings.TrimSpace(b.Title) &&
		strings.TrimSpace(a.Spec) == strings.TrimSpace(b.Spec) &&
		strings.TrimSpace(a.Unit) == strings.TrimSpace(b.Unit)
}

// clMergeResource 把入参合并到目标工料机上（target=nil 即按入参自身成一条新
// 资源）：没给的字段保持原值，类别归一，数值域校验。先定目标、再合并——
// 顺序反了会让没带的字段（价格等）落成 0。
func clMergeResource(target *workcost.Resource, in *costLibraryResource) (workcost.Resource, error) {
	r := workcost.Resource{Status: "现行"}
	if target != nil {
		r = *target
	}
	if in.Kind != nil {
		r.Kind = strings.TrimSpace(*in.Kind)
	}
	if in.Title != nil {
		r.Title = strings.TrimSpace(*in.Title)
	}
	if in.Spec != nil {
		r.Spec = strings.TrimSpace(*in.Spec)
	}
	if in.Unit != nil {
		r.Unit = strings.TrimSpace(*in.Unit)
	}
	if in.BasePrice != nil {
		r.BasePrice = *in.BasePrice
	}
	if in.CurrentPrice != nil {
		r.CurrentPrice = *in.CurrentPrice
	}
	if in.CategoryPath != nil {
		r.CategoryPath = strings.TrimSpace(*in.CategoryPath)
	}
	if in.Source != nil {
		r.Source = strings.TrimSpace(*in.Source)
	}
	if in.Supplier != nil {
		r.Supplier = strings.TrimSpace(*in.Supplier)
	}
	if in.Region != nil {
		r.Region = strings.TrimSpace(*in.Region)
	}
	if in.PriceDate != nil {
		r.PriceDate = strings.TrimSpace(*in.PriceDate)
	}
	if in.PriceType != nil {
		r.PriceType = strings.TrimSpace(*in.PriceType)
	}
	if in.ValidUntil != nil {
		r.ValidUntil = strings.TrimSpace(*in.ValidUntil)
	}
	if in.LossRate != nil {
		r.LossRate = *in.LossRate
	}
	if in.Note != nil {
		r.Note = strings.TrimSpace(*in.Note)
	}
	if in.Tags != nil {
		r.Tags = in.Tags
	}
	if in.Status != nil {
		r.Status = strings.TrimSpace(*in.Status)
	}
	switch workcost.NormalizeKind(r.Kind) {
	case workcost.KindLabor, workcost.KindMaterial, workcost.KindMachine, workcost.KindOutsourced:
	default:
		return r, fmt.Errorf("工料机类别须为 人工/材料/机械/外委（收到 %q）", r.Kind)
	}
	r.Kind = workcost.NormalizeKind(r.Kind)
	if r.Title == "" {
		return r, fmt.Errorf("工料机需要名称（resource.title）")
	}
	if r.BasePrice < 0 || r.CurrentPrice < 0 || r.LossRate < 0 || r.LossRate >= 1 {
		return r, fmt.Errorf("价格不能为负、损耗率须在 [0,1)（收到 基准价 %v / 现行价 %v / 损耗率 %v）", r.BasePrice, r.CurrentPrice, r.LossRate)
	}
	return r, nil
}

// ── import_project：导入五表工作簿 ───────────────────────────────────

func (t costLibraryTool) importProject(s *workcost.Store, p costLibraryArgs) (string, error) {
	rel := strings.TrimSpace(p.Path)
	if rel == "" {
		return "", fmt.Errorf("import_project 需要工作簿路径 path（.xlsx）")
	}
	if !strings.EqualFold(filepath.Ext(rel), ".xlsx") {
		return "", fmt.Errorf("import_project 只收 .xlsx 五表工作簿（收到 %s）", filepath.Ext(rel))
	}
	abs, display := resolvePreviewPath(rel)
	if abs == "" {
		return "", fmt.Errorf("读不到工作簿 %s（不存在，或在工作区之外）", rel)
	}
	if info, err := os.Stat(abs); err != nil || info.IsDir() {
		return "", fmt.Errorf("读不到工作簿 %s（不存在，或在工作区之外）", rel)
	}
	b, err := workcost.ParseProjectWorkbook(abs)
	if err != nil {
		return "", err
	}
	projectName := strings.TrimSpace(b.Project)
	if projectName == "" {
		projectName = strings.TrimSuffix(b.FileName, ".xlsx")
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "import_project"))
	fmt.Fprintf(&out, "**工作簿**: %s（%d 张表）\n", display, len(b.Sheets))
	fmt.Fprintf(&out, "**项目**: %s", projectName)
	if b.Location != "" {
		fmt.Fprintf(&out, "｜地区 %s", b.Location)
	}
	if b.Duration != "" {
		fmt.Fprintf(&out, "｜工期 %s", b.Duration)
	}
	out.WriteString("\n")
	fmt.Fprintf(&out, "**规模**: 工料机 %d 条｜清单项 %d 条｜消耗行 %d 条\n", len(b.Resources), len(b.Items), len(b.Lines))
	fmt.Fprintf(&out, "**取费**: 企管 %s｜规费 %s｜利润 %s｜增值税 %s", clPct(b.Fee.ManagementRate), clPct(b.Fee.RegulatoryRate), clPct(b.Fee.ProfitRate), clPct(b.Fee.TaxRate))
	if b.Fee.ControlPrice > 0 {
		fmt.Fprintf(&out, "｜控制价 %s", clMoney(b.Fee.ControlPrice))
	}
	out.WriteString("\n")
	projects, _ := s.BillProjects()
	existingID := int64(0)
	for i := range projects {
		if projects[i].Name == projectName {
			existingID = projects[i].ID
			fmt.Fprintf(&out, "**同名项目**: 已存在 #%d（清单 %d 行）——本次更新封面/费率，同编码清单行更新（原表单价随工作簿刷新）\n", projects[i].ID, projects[i].ItemCount)
			break
		}
	}
	if len(b.Skipped) > 0 {
		reasons := map[string]int{}
		for _, r := range b.Skipped {
			reasons[r.Reason]++
		}
		keys := make([]string, 0, len(reasons))
		for k := range reasons {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(&out, "**跳过 %d 行**: ", len(b.Skipped))
		for i, k := range keys {
			if i > 0 {
				out.WriteString("；")
			}
			fmt.Fprintf(&out, "%s ×%d", k, reasons[k])
		}
		out.WriteString("\n")
	}
	if len(b.Warnings) > 0 {
		fmt.Fprintf(&out, "**告警 %d 条**（公式无缓存值/列未识别等，列前 %d 条）:\n", len(b.Warnings), clCap(len(b.Warnings)))
		for i, w := range b.Warnings {
			if i >= clCap(len(b.Warnings)) {
				break
			}
			fmt.Fprintf(&out, "- %s\n", w)
		}
	}
	if !workcost.DetectedFiveSheet(b.Sheets) {
		out.WriteString("\n注意：这份工作簿不是标准五表模版（缺「工料机价格」或「综合单价」表），解析结果可能不完整——核对上面的规模；非五表的工作簿请读表后用 cost_project_apply（create_project + save_items）编制。\n")
	}
	if p.DryRun {
		out.WriteString("\n落库口径：工料机按编码与身份 UPSERT；定额按统一编码（同内容复用，冲突加 -N 消歧）。确认无误后去掉 dry_run 正式导入。")
		return out.String(), nil
	}
	res := s.ApplyProjectBundle(b)
	itemCount := 0
	if projects, err := s.BillProjects(); err == nil {
		for i := range projects {
			if projects[i].Name == projectName {
				existingID, itemCount = projects[i].ID, projects[i].ItemCount
			}
		}
	}
	fmt.Fprintf(&out, "**落库**: 工料机 新建 %d/更新 %d｜定额 新建 %d/更新 %d｜消耗行 %d｜清单 %d 行\n",
		res.ResourceNew, res.ResourceUpd, res.QuotaNew, res.QuotaUpd, res.Lines, itemCount)
	if n := len(res.Errors); n > 0 {
		fmt.Fprintf(&out, "**未落成 %d 条**（列前 %d 条，逐条核对）:\n", n, clCap(n))
		for i, e := range res.Errors {
			if i >= clCap(n) {
				break
			}
			fmt.Fprintf(&out, "- %s\n", e)
		}
	}
	if existingID > 0 {
		fmt.Fprintf(&out, "\n下一步：cost_audit project_id=%d 跑机检；造价数据库板块切回即见。", existingID)
	} else {
		out.WriteString("\n下一步：cost_audit 跑机检；造价数据库板块切回即见。")
	}
	return out.String(), nil
}

// ── save_quota / delete_quota：消耗定额 ──────────────────────────────

func (t costLibraryTool) saveQuota(s *workcost.Store, p costLibraryArgs) (string, error) {
	in := p.Quota
	if in == nil || strings.TrimSpace(in.Code) == "" {
		return "", fmt.Errorf("save_quota 需要 quota.code（定额编码；已存在=更新）")
	}
	code := strings.TrimSpace(in.Code)
	existing, err := s.GetQuota(code)
	isNew := workcostNotFound(err)
	if err != nil && !isNew {
		return "", err
	}
	q := workcost.Quota{Code: code, Status: "现行"}
	if !isNew {
		q = *existing // 没给的头字段保持原值
	}
	setStr := func(dst *string, v *string) {
		if v != nil {
			*dst = strings.TrimSpace(*v)
		}
	}
	setStr(&q.Title, in.Title)
	setStr(&q.Unit, in.Unit)
	setStr(&q.Specialty, in.Specialty)
	setStr(&q.Chapter, in.Chapter)
	setStr(&q.CategoryPath, in.CategoryPath)
	setStr(&q.Source, in.Source)
	setStr(&q.Note, in.Note)
	if q.Title == "" {
		return "", fmt.Errorf("定额 %s 需要名称（quota.title）", code)
	}
	if in.Items == nil && isNew {
		return "", fmt.Errorf("新建定额必须给含量行 quota.items（没有消耗也传空数组 []=零含量占位）")
	}
	if in.Items != nil {
		// 含量行整组替换。每行都必须引用在库的工料机编码：存储层不保存无编码的
		// 含量行（SaveQuota 对空 ResourceCode 直接跳过），放行等于静默丢行；
		// 查不到编码也报错——绝不编造。需要新资源先用 save_resource 建好。
		items := make([]workcost.QuotaItem, 0, len(*in.Items))
		var missing []string
		for i, it := range *in.Items {
			code := strings.TrimSpace(it.ResourceCode)
			if code == "" {
				return "", fmt.Errorf("第 %d 行含量行缺 resource_code：含量行必须引用在库的工料机（先用 cost_library save_resource 建资源，再把编码填进来）", i+1)
			}
			if it.Quantity < 0 {
				return "", fmt.Errorf("第 %d 行含量为负数", i+1)
			}
			if it.LossRate < 0 || it.LossRate >= 1 {
				return "", fmt.Errorf("第 %d 行损耗率 %v 不在 [0,1) 区间（0.03=3%%）", i+1, it.LossRate)
			}
			r, err := s.GetResourceByCode(code)
			if err != nil {
				if !workcostNotFound(err) {
					return "", err
				}
				missing = append(missing, code)
				continue
			}
			// 展示字段从资源库派生（名称/规格/单位单一事实源），行级覆盖才用入参。
			items = append(items, workcost.QuotaItem{
				ResourceCode: code, Kind: r.Kind, Title: r.Title, Spec: r.Spec, Unit: r.Unit,
				Quantity: it.Quantity, LossRate: it.LossRate, Note: strings.TrimSpace(it.Note), Sort: i,
			})
		}
		if len(missing) > 0 {
			return "", fmt.Errorf("含量行引用的工料机不在库：%s。先用 cost_library save_resource 建资源（或用 cost_project action=resources 核对编码），不要编造编码", strings.Join(missing, "、"))
		}
		q.Items = items
	}

	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "save_quota"))
	if isNew {
		fmt.Fprintf(&out, "**新建定额** %s「%s」（%s）｜含量行 %d 行\n", code, q.Title, q.Unit, len(q.Items))
	} else {
		how := "保留原有"
		if in.Items != nil {
			how = "整组替换为"
		}
		fmt.Fprintf(&out, "**更新定额** %s「%s」（%s）｜含量行%s %d 行\n", code, q.Title, q.Unit, how, len(q.Items))
	}
	labor, material, machine, outsourced := 0, 0, 0, 0
	for _, it := range q.Items {
		switch it.Kind {
		case workcost.KindLabor:
			labor++
		case workcost.KindMachine:
			machine++
		case workcost.KindOutsourced:
			outsourced++
		default:
			material++
		}
	}
	fmt.Fprintf(&out, "**含量构成**: 人工 %d 行｜材料 %d 行｜机械 %d 行｜外委 %d 行\n", labor, material, machine, outsourced)
	if p.DryRun {
		out.WriteString("\n演练通过。去掉 dry_run 正式保存。")
		return out.String(), nil
	}
	if _, err := s.SaveQuota(q, nil); err != nil {
		return "", fmt.Errorf("定额 %s 保存失败: %w", code, err)
	}
	if c, _, err := s.ComposeQuota(code, nil); err == nil {
		fmt.Fprintf(&out, "**现算综合单价**（人材机）: %s", clMoney(c.CompositePrice))
		for _, w := range c.Warnings {
			fmt.Fprintf(&out, "\n- ⚠ %s", w)
		}
		out.WriteString("\n")
	}
	out.WriteString("\n引用它的清单项下次读取即按新含量重算；用 cost_audit 复核相关项目。")
	return out.String(), nil
}

func (t costLibraryTool) deleteQuota(s *workcost.Store, p costLibraryArgs) (string, error) {
	codes := clCleanCodes(p.Codes)
	if len(codes) == 0 {
		return "", fmt.Errorf("delete_quota 需要 codes（定额编码列表）")
	}
	type victim struct {
		code  string
		title string
		items int
		refs  []workcost.UsageRef
	}
	victims := make([]victim, 0, len(codes))
	var blocked []string
	for _, code := range codes {
		q, err := s.GetQuota(code)
		if err != nil {
			if workcostNotFound(err) {
				return "", fmt.Errorf("定额 %s 不存在（整批未删除）", code)
			}
			return "", err
		}
		refs, err := s.QuotaUsage(code)
		if err != nil {
			return "", err
		}
		victims = append(victims, victim{code: q.Code, title: q.Title, items: len(q.Items), refs: refs})
		if len(refs) > 0 && !p.Force {
			blocked = append(blocked, fmt.Sprintf("%s（被 %d 条清单项引用）", code, len(refs)))
		}
	}
	if len(blocked) > 0 {
		return "", fmt.Errorf("以下定额仍被清单项引用，未删除：%s。先在这些清单行解除挂接（cost_project_apply save_items 重发该行且不带该定额），或确认后加 force=true（引用一并解除，相关清单项退回手填行）", strings.Join(blocked, "、"))
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "delete_quota"))
	fmt.Fprintf(&out, "**%s定额 %d 条**（含全部含量行）", map[bool]string{true: "将删除", false: "已删除"}[p.DryRun], len(victims))
	attached := 0
	for _, v := range victims {
		attached += len(v.refs)
		if p.DryRun {
			fmt.Fprintf(&out, "\n- %s「%s」含量行 %d 行", v.code, v.title, v.items)
			if len(v.refs) > 0 {
				fmt.Fprintf(&out, "｜解除 %d 条清单挂接", len(v.refs))
			}
		}
	}
	if attached > 0 {
		fmt.Fprintf(&out, "｜解除清单挂接 %d 条（这些清单行现在是手填行，cost_audit 会点名）", attached)
	}
	out.WriteString("\n")
	if p.DryRun {
		out.WriteString("\n删除不可恢复；确认后去掉 dry_run 正式删除。")
		return out.String(), nil
	}
	for i, v := range victims {
		if len(v.refs) > 0 {
			if _, err := s.DetachQuotaEverywhere(v.code); err != nil {
				return "", fmt.Errorf("定额 %s 解除挂接失败（它之前的 %d 条已删除，其余未处理）: %w", v.code, i, err)
			}
		}
		if err := s.DeleteQuota(v.code); err != nil {
			return "", fmt.Errorf("定额 %s 删除失败（它之前的 %d 条已删除，其余未处理）: %w", v.code, i, err)
		}
	}
	return out.String(), nil
}

// ── save_resource / delete_resource / set_price：工料机 ─────────────

func (t costLibraryTool) saveResource(s *workcost.Store, p costLibraryArgs) (string, error) {
	in := p.Resource
	if in == nil {
		return "", fmt.Errorf("save_resource 需要 resource 对象（改已有资源给 id 或 code；新建至少给 kind+title+unit）")
	}
	all, err := s.ListResources("", "")
	if err != nil {
		return "", err
	}
	// 定位目标：id 优先，其次编码。
	var target *workcost.Resource
	switch {
	case in.ID > 0:
		for i := range all {
			if all[i].ID == in.ID {
				target = &all[i]
				break
			}
		}
		if target == nil {
			return "", fmt.Errorf("工料机 #%d 不存在", in.ID)
		}
	case strings.TrimSpace(in.Code) != "":
		code := strings.TrimSpace(in.Code)
		for i := range all {
			if all[i].Code == code {
				target = &all[i]
				break
			}
		}
		if target == nil && in.Title == nil {
			return "", fmt.Errorf("工料机编码 %s 不存在；新建资源请给 title", code)
		}
	}
	if target != nil && strings.TrimSpace(in.Code) != "" && strings.TrimSpace(in.Code) != target.Code {
		return "", fmt.Errorf("不支持改工料机编码（%s → %s）：编码是定额含量行的引用锚点，改了会断开引用。要换锚点请新建资源后用 cost_library dedupe 合并", target.Code, strings.TrimSpace(in.Code))
	}
	// 新建且没给编码：入参自身字段先探一次身份——同身份已有资源=改它（与
	// SaveResource 的身份 UPSERT 同语义）。必须先定目标再合并：否则入参没带的
	// 字段（价格等）不会从原行带过来，一次「同身份再存」就把价格清成 0。
	if target == nil && strings.TrimSpace(in.Code) == "" {
		probe, err := clMergeResource(nil, in)
		if err != nil {
			return "", err
		}
		for i := range all {
			if sameResourceIdentity(all[i], probe) {
				target = &all[i]
				break
			}
		}
	}
	if target == nil && in.Kind == nil {
		return "", fmt.Errorf("新建工料机必须给类别 kind（人工/材料/机械/外委），不替你默认")
	}
	r, err := clMergeResource(target, in)
	if err != nil {
		return "", err
	}
	// 身份唯一：类别+名称+规格+单位 相同的另一条资源不能并存。
	for i := range all {
		if all[i].ID != r.ID && sameResourceIdentity(all[i], r) {
			return "", fmt.Errorf("身份（类别+名称+规格+单位）与工料机 %s「%s」重复——直接改那一条（给它的 id 或 code），或调整规格/单位", all[i].Code, all[i].Title)
		}
	}
	// 给了编码的新建：编码不能被别的资源占用。
	if target == nil && r.Code != "" {
		for i := range all {
			if all[i].Code == r.Code {
				return "", fmt.Errorf("工料机编码 %s 已被「%s」占用", r.Code, all[i].Title)
			}
		}
	}
	oldEffective := 0.0
	if target != nil {
		oldEffective = target.EffectivePrice()
	}
	priceChanged := target == nil || oldEffective != r.EffectivePrice()

	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "save_resource"))
	if target != nil {
		fmt.Fprintf(&out, "**更新工料机** [%s] %s %s（%s）", target.Code, target.Kind, target.Title, target.Unit)
		if oldEffective != r.EffectivePrice() {
			fmt.Fprintf(&out, "｜取用价 %s → %s", clMoney(oldEffective), clMoney(r.EffectivePrice()))
		}
	} else {
		fmt.Fprintf(&out, "**新建工料机** %s「%s」（%s）｜取用价 %s", r.Kind, r.Title, r.Unit, clMoney(r.EffectivePrice()))
	}
	out.WriteString("\n")
	if target != nil && priceChanged {
		refs, _ := s.ResourceUsage(target.Code)
		fmt.Fprintf(&out, "**影响面**: %d 条定额含量行引用它，综合单价下次读取按新价重算\n", len(refs))
	}
	if p.DryRun {
		out.WriteString("\n演练通过。去掉 dry_run 正式保存。")
		return out.String(), nil
	}
	saved, err := s.SaveResource(r)
	if err != nil {
		return "", fmt.Errorf("工料机保存失败: %w", err)
	}
	// 与绑定面 GaeaWorkcostResourceSave 同款收尾：没给基准价时用现行价补齐，
	// 否则现行价回落后 EffectivePrice 会取到 0。
	if saved.BasePrice <= 0 && saved.CurrentPrice > 0 {
		fixed := *saved
		fixed.BasePrice = fixed.CurrentPrice
		if again, err := s.SaveResource(fixed); err == nil {
			saved = again
			fmt.Fprintf(&out, "**基准价**: 按现行价补齐为 %s\n", clMoney(saved.CurrentPrice))
		}
	}
	fmt.Fprintf(&out, "**编码**: %s\n\n定额含量行用这个编码引用它；批量改价用 cost_library bulk_price。", saved.Code)
	return out.String(), nil
}

func (t costLibraryTool) deleteResource(s *workcost.Store, p costLibraryArgs) (string, error) {
	if len(p.IDs) == 0 && len(clCleanCodes(p.Codes)) == 0 {
		return "", fmt.Errorf("delete_resource 需要 ids 或 codes（工料机 id 或编码列表，至少一个）")
	}
	type victim struct {
		r    workcost.Resource
		refs int
	}
	victims := []victim{}
	seen := map[int64]bool{}
	add := func(r *workcost.Resource) {
		if seen[r.ID] {
			return
		}
		seen[r.ID] = true
		victims = append(victims, victim{r: *r})
	}
	for _, id := range p.IDs {
		r, err := s.GetResource(id)
		if err != nil {
			if workcostNotFound(err) {
				return "", fmt.Errorf("工料机 #%d 不存在（整批未删除）", id)
			}
			return "", err
		}
		add(r)
	}
	for _, c := range clCleanCodes(p.Codes) {
		r, err := s.GetResourceByCode(c)
		if err != nil {
			if workcostNotFound(err) {
				return "", fmt.Errorf("工料机 %s 不存在（整批未删除）", c)
			}
			return "", err
		}
		add(r)
	}
	var blocked []string
	for i := range victims {
		refs, err := s.ResourceUsage(victims[i].r.Code)
		if err != nil {
			return "", err
		}
		victims[i].refs = len(refs)
		if len(refs) > 0 && !p.Force {
			blocked = append(blocked, fmt.Sprintf("%s「%s」（%d 条含量行引用）", victims[i].r.Code, victims[i].r.Title, len(refs)))
		}
	}
	if len(blocked) > 0 {
		return "", fmt.Errorf("以下工料机仍被定额含量行引用，未删除：%s。先改那些定额的含量行（cost_library save_quota），或确认后加 force=true（含量行引用悬空，cost_audit 会点名）", strings.Join(blocked, "、"))
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "delete_resource"))
	fmt.Fprintf(&out, "**%s工料机 %d 条**", map[bool]string{true: "将删除", false: "已删除"}[p.DryRun], len(victims))
	attached := 0
	for _, v := range victims {
		attached += v.refs
		if p.DryRun {
			fmt.Fprintf(&out, "\n- [%s] %s %s（%s）", v.r.Code, v.r.Kind, v.r.Title, v.r.Unit)
			if v.refs > 0 {
				fmt.Fprintf(&out, "｜解除 %d 条含量行引用", v.refs)
			}
		}
	}
	if attached > 0 {
		fmt.Fprintf(&out, "｜%d 条含量行引用悬空（cost_audit 会点名）", attached)
	}
	out.WriteString("\n")
	if p.DryRun {
		out.WriteString("\n删除不可恢复；确认后去掉 dry_run 正式删除。")
		return out.String(), nil
	}
	for i, v := range victims {
		if err := s.DeleteResource(v.r.ID, p.Force); err != nil {
			return "", fmt.Errorf("工料机 %s 删除失败（它之前的 %d 条已删除，其余未处理）: %w", v.r.Code, i, err)
		}
	}
	return out.String(), nil
}

func (t costLibraryTool) setPrice(s *workcost.Store, p costLibraryArgs) (string, error) {
	r, err := clResolveResource(s, p.ResourceID, p.ResourceCode)
	if err != nil {
		return "", err
	}
	if p.Price == nil || *p.Price <= 0 {
		return "", fmt.Errorf("set_price 需要正数 price（新单价，元）")
	}
	refs, err := s.ResourceUsage(r.Code)
	if err != nil {
		return "", err
	}
	priceType := strings.TrimSpace(p.PriceType)
	if priceType == "" {
		priceType = "到场价"
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "set_price"))
	fmt.Fprintf(&out, "**[%s] %s（%s）**: %s → %s", r.Code, r.Title, r.Unit, clMoney(r.EffectivePrice()), clMoney(*p.Price))
	meta := []string{}
	if v := strings.TrimSpace(p.Period); v != "" {
		meta = append(meta, "期数 "+v)
	}
	if v := strings.TrimSpace(p.Region); v != "" {
		meta = append(meta, "地区 "+v)
	}
	meta = append(meta, "口径 "+priceType)
	fmt.Fprintf(&out, "｜%s\n", strings.Join(meta, "｜"))
	fmt.Fprintf(&out, "**影响面**: %d 条定额含量行引用它，综合单价下次读取按新价重算\n", len(refs))
	if p.DryRun {
		out.WriteString("\n演练通过。去掉 dry_run 正式调价（写一条调价历史并推进现行价）。")
		return out.String(), nil
	}
	saved, err := s.UpdateResourcePrice(r.ID, *p.Price, strings.TrimSpace(p.Period), strings.TrimSpace(p.Region), priceType, "造价模式调价", strings.TrimSpace(p.Note))
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&out, "现行价已更新为 %s，调价历史已留痕。", clMoney(saved.CurrentPrice))
	return out.String(), nil
}

// ── bulk_price：批量调价 ─────────────────────────────────────────────

func (t costLibraryTool) bulkPrice(s *workcost.Store, p costLibraryArgs) (string, error) {
	if len(p.Rows) == 0 {
		return "", fmt.Errorf("bulk_price 需要 rows（批量调价行：title/spec/unit/price/period/region/note）")
	}
	if len(p.Rows) > costLibraryMaxRows {
		return "", fmt.Errorf("单批最多 %d 行（当前 %d），请分批提交", costLibraryMaxRows, len(p.Rows))
	}
	rows := make([]workcost.BulkPriceRow, 0, len(p.Rows))
	for i, r := range p.Rows {
		r.Title = strings.TrimSpace(r.Title)
		r.Spec = strings.TrimSpace(r.Spec)
		r.Unit = strings.TrimSpace(r.Unit)
		r.Period = strings.TrimSpace(r.Period)
		r.Region = strings.TrimSpace(r.Region)
		r.Note = strings.TrimSpace(r.Note)
		if r.Row <= 0 {
			r.Row = i + 1
		}
		rows = append(rows, r)
	}
	matches, skips, err := s.MatchBulkPrices(rows)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "bulk_price"))
	fmt.Fprintf(&out, "**匹配 %d 行｜跳过 %d 行**（共 %d 行）\n", len(matches), len(skips), len(rows))
	if len(matches) > 0 {
		fmt.Fprintf(&out, "**将调价**（列前 %d 行）:\n", clCap(len(matches)))
		for i, m := range matches {
			if i >= clCap(len(matches)) {
				break
			}
			fmt.Fprintf(&out, "- [%s] %s（%s）%s → %s", m.Code, m.Title, m.Unit, clMoney(m.OldPrice), clMoney(m.NewPrice))
			if m.Period != "" {
				fmt.Fprintf(&out, "｜%s", m.Period)
			}
			out.WriteString("\n")
		}
	}
	if len(skips) > 0 {
		fmt.Fprintf(&out, "**跳过**（宁缺勿误，列前 %d 行）:\n", clCap(len(skips)))
		for i, sk := range skips {
			if i >= clCap(len(skips)) {
				break
			}
			fmt.Fprintf(&out, "- 第 %d 行「%s」：%s\n", sk.Row, sk.Title, sk.Reason)
		}
	}
	if p.DryRun {
		out.WriteString("\n演练通过。去掉 dry_run 正式调价（整批一个事务，写调价历史并推进现行价）。")
		return out.String(), nil
	}
	res, err := s.ApplyBulkPrices(rows)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&out, "\n已更新 %d 条工料机的现行价；跳过行未改动。", res.Updated)
	return out.String(), nil
}

// ── delete_items / delete_project：清单层删除 ───────────────────────

func (t costLibraryTool) deleteItems(s *workcost.Store, p costLibraryArgs) (string, error) {
	if p.ProjectID <= 0 || len(p.ItemIDs) == 0 {
		return "", fmt.Errorf("delete_items 需要 project_id 与 item_ids（清单行 id 列表，cost_project action=items 可查）")
	}
	items, err := s.BillItems(p.ProjectID)
	if err != nil {
		return "", err
	}
	want := map[int64]bool{}
	for _, id := range p.ItemIDs {
		want[id] = true
	}
	var targets []workcost.BillItem
	var missing []string
	for _, it := range items {
		if want[it.ID] {
			targets = append(targets, it)
			delete(want, it.ID)
		}
	}
	for id := range want {
		missing = append(missing, fmt.Sprintf("#%d", id))
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("这些清单行不属于项目 #%d（整批未删除）：%s", p.ProjectID, strings.Join(missing, "、"))
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "delete_items"))
	fmt.Fprintf(&out, "**%s清单行 %d 条**（定额挂接随行清理）", map[bool]string{true: "将删除", false: "已删除"}[p.DryRun], len(targets))
	for _, it := range targets {
		codes := clCleanCodes(append([]string{it.QuotaCode}, it.QuotaCodes...))
		line := fmt.Sprintf("\n- [%s] %s（%s）量=%g", it.Code, it.Title, it.Unit, it.Quantity)
		if len(codes) > 0 {
			line += "｜定额 " + strings.Join(codes, "+")
		}
		if it.PriceOverride > 0 {
			line += fmt.Sprintf("｜手填价 %s", clMoney(it.PriceOverride))
		}
		out.WriteString(line)
	}
	out.WriteString("\n")
	if p.DryRun {
		out.WriteString("\n删除不可恢复；确认后去掉 dry_run 正式删除。")
		return out.String(), nil
	}
	for i, it := range targets {
		if err := s.DeleteBillItem(it.ID); err != nil {
			return "", fmt.Errorf("清单行 %s 删除失败（它之前的 %d 条已删除，其余未处理）: %w", it.Code, i, err)
		}
	}
	return out.String(), nil
}

func (t costLibraryTool) deleteProject(s *workcost.Store, p costLibraryArgs) (string, error) {
	if p.ProjectID <= 0 {
		return "", fmt.Errorf("delete_project 需要 project_id（cost_project action=projects 可查）")
	}
	proj := findBillProject(s, p.ProjectID)
	if proj == nil {
		return "", fmt.Errorf("项目 #%d 不存在", p.ProjectID)
	}
	items, err := s.BillItems(p.ProjectID)
	if err != nil {
		return "", err
	}
	// 独占定额估算：本项目的清单行引用、且不被其他项目引用的定额随项目删除。
	ownIDs := map[int64]bool{}
	quotaCodes := map[string]bool{}
	for _, it := range items {
		ownIDs[it.ID] = true
		for _, c := range clCleanCodes(append([]string{it.QuotaCode}, it.QuotaCodes...)) {
			quotaCodes[c] = true
		}
	}
	exclusive := 0
	for code := range quotaCodes {
		refs, err := s.QuotaUsage(code)
		if err != nil {
			return "", err
		}
		only := true
		for _, ref := range refs {
			if !ownIDs[ref.ID] {
				only = false
				break
			}
		}
		if only {
			exclusive++
		}
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "delete_project"))
	fmt.Fprintf(&out, "**%s项目 #%d「%s」**: 清单 %d 行｜连带删除独占定额 %d 条（共享定额与工料机保留）",
		map[bool]string{true: "将删除", false: "已删除"}[p.DryRun], proj.ID, proj.Name, len(items), exclusive)
	out.WriteString("\n")
	if p.DryRun {
		out.WriteString("\n删除不可恢复；确认后去掉 dry_run 正式删除。需要重新录入时用 import_project 导入原工作簿。")
		return out.String(), nil
	}
	if _, err := s.DeleteBillProject(p.ProjectID); err != nil {
		return "", err
	}
	return out.String(), nil
}

// ── seed / flywheel / dedupe / health：库级维护 ─────────────────────

func (t costLibraryTool) seed(s *workcost.Store, p costLibraryArgs) (string, error) {
	inputs := t.a.workcostSeedInputs()
	if len(inputs) == 0 {
		return "资料条目为空（或读取失败），没有可资源化的条目。", nil
	}
	opt := workcost.SeedOptions{}
	preview := workcost.PreviewSeed(inputs, opt)
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "seed"))
	fmt.Fprintf(&out, "**扫描资料条目 %d 条 → 可资源化 %d 条**（人工 %d｜材料 %d｜机械 %d｜外委 %d）\n",
		preview.Total, len(preview.Candidates), preview.Counts.Labor, preview.Counts.Material, preview.Counts.Machine, preview.Counts.Outsourced)
	if len(preview.Skipped) > 0 {
		keys := make([]string, 0, len(preview.Skipped))
		for k := range preview.Skipped {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Fprintf(&out, "**跳过原因**: ")
		for i, k := range keys {
			if i > 0 {
				out.WriteString("；")
			}
			fmt.Fprintf(&out, "%s ×%d", k, preview.Skipped[k])
		}
		out.WriteString("\n")
	}
	if p.DryRun {
		for i, c := range preview.Candidates {
			if i >= clCap(len(preview.Candidates)) {
				break
			}
			fmt.Fprintf(&out, "- [%s] %s（%s）%s\n", c.Kind, c.Title, c.Unit, clMoney(c.Price))
		}
		out.WriteString("\n演练通过。去掉 dry_run 正式资源化（同身份资源走更新，重复执行不产生重复）。")
		return out.String(), nil
	}
	cands, _ := workcost.BuildSeedCandidates(inputs, opt)
	if len(cands) == 0 {
		return "", fmt.Errorf("没有可资源化的条目（资料条目全部不满足规则）")
	}
	res := s.ApplySeed(cands)
	fmt.Fprintf(&out, "**入库**: 新建 %d / 更新 %d / 失败 %d\n", res.Created, res.Updated, len(res.Errors))
	for i, e := range res.Errors {
		if i >= clCap(len(res.Errors)) {
			break
		}
		fmt.Fprintf(&out, "- %s\n", e)
	}
	return out.String(), nil
}

func (t costLibraryTool) flywheel(s *workcost.Store, p costLibraryArgs) (string, error) {
	prev, err := s.QuotaFlywheelPreview()
	if err != nil {
		return "", err
	}
	stems := clCleanCodes(p.Stems)
	if p.DryRun || len(stems) == 0 {
		var out strings.Builder
		out.WriteString("🔍 演练（未写库）：自生成企业定额\n")
		fmt.Fprintf(&out, "**跨项目含量分歧簇 %d 个**｜不成立 %d 个\n", len(prev.Candidates), len(prev.Skipped))
		for i, c := range prev.Candidates {
			if i >= clCap(len(prev.Candidates)) {
				fmt.Fprintf(&out, "- …共 %d 个候选\n", len(prev.Candidates))
				break
			}
			fmt.Fprintf(&out, "- %s → %s「%s」（%s）: %d 个变体｜来源项目 %d 个\n", c.OrigCode, c.NewCode, c.Title, c.Unit, len(c.Members), c.Sources)
		}
		if len(prev.Skipped) > 0 {
			fmt.Fprintf(&out, "**不成立的簇**（列前 %d 个）:\n", clCap(len(prev.Skipped)))
			for i, sk := range prev.Skipped {
				if i >= clCap(len(prev.Skipped)) {
					break
				}
				fmt.Fprintf(&out, "- %s「%s」：%s\n", sk.Key, sk.Title, sk.Reason)
			}
		}
		out.WriteString("\n挑出要生成的簇，用 stems=[origCode…] 去掉 dry_run 正式生成（ENT-* 企业定额，含量=各变体中位数）。")
		return out.String(), nil
	}
	res, err := s.QuotaFlywheelApply(stems)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString("✅ 已完成：自生成企业定额\n")
	fmt.Fprintf(&out, "**已生成 %d 条**", len(res.Saved))
	for _, code := range res.Saved {
		fmt.Fprintf(&out, "\n- %s", code)
	}
	if len(res.Skipped) > 0 {
		fmt.Fprintf(&out, "\n**跳过 %d 个**:\n", len(res.Skipped))
		for _, sk := range res.Skipped {
			fmt.Fprintf(&out, "- %s：%s\n", sk.Key, sk.Reason)
		}
	}
	return out.String(), nil
}

func (t costLibraryTool) dedupe(s *workcost.Store, p costLibraryArgs) (string, error) {
	if p.KeepID <= 0 || len(p.MergeIDs) == 0 {
		return "", fmt.Errorf("dedupe 需要 keep_id（保留）与 merge_ids（合并掉的工料机 id）")
	}
	keep, err := s.GetResource(p.KeepID)
	if err != nil {
		return "", err
	}
	seen := map[int64]bool{keep.ID: true}
	type doomed struct {
		r        workcost.Resource
		refRows  int
		histRows int
	}
	var doomedList []doomed
	var mergeIDs []int64
	for _, id := range p.MergeIDs {
		if seen[id] {
			if id == keep.ID {
				return "", fmt.Errorf("merge_ids 不能包含保留条目本身（id=%d）", id)
			}
			continue
		}
		seen[id] = true
		m, err := s.GetResource(id)
		if err != nil {
			return "", fmt.Errorf("合并条目读取失败（整批未合并）: %w", err)
		}
		refs, err := s.ResourceUsage(m.Code)
		if err != nil {
			return "", err
		}
		hist, _ := s.ResourcePrices(id)
		doomedList = append(doomedList, doomed{r: *m, refRows: len(refs), histRows: len(hist)})
		mergeIDs = append(mergeIDs, id)
	}
	if len(doomedList) == 0 {
		return "", fmt.Errorf("merge_ids 去重后为空")
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "dedupe"))
	fmt.Fprintf(&out, "**保留** [%s] %s %s（%s）", keep.Code, keep.Kind, keep.Title, keep.Unit)
	fmt.Fprintf(&out, "｜**%s合并 %d 条**:", map[bool]string{true: "将", false: "已"}[p.DryRun], len(doomedList))
	redirectItems, redirectPrices := 0, 0
	for _, d := range doomedList {
		redirectItems += d.refRows
		redirectPrices += d.histRows
		fmt.Fprintf(&out, "\n- [%s] %s %s（%s）｜含量行引用 %d 行｜调价历史 %d 条", d.r.Code, d.r.Kind, d.r.Title, d.r.Unit, d.refRows, d.histRows)
	}
	fmt.Fprintf(&out, "\n**重定向**: 定额含量引用 %d 行改指 %s｜调价历史并入 %d 条\n", redirectItems, keep.Code, redirectPrices)
	if p.DryRun {
		out.WriteString("\n演练通过。合并不可恢复；确认后去掉 dry_run 正式合并。")
		return out.String(), nil
	}
	res, err := s.DedupeResources(keep.ID, mergeIDs)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&out, "已合并 %d 条进「%s」。", res.Merged, res.Kept)
	return out.String(), nil
}

// health 库级健康明细（只读）。
func (t costLibraryTool) health(s *workcost.Store) (string, error) {
	hr, err := s.HealthScan()
	if err != nil {
		return "", err
	}
	var out strings.Builder
	fmt.Fprintf(&out, "## 造价库健康扫描（只读）\n\n共 %d 项发现。\n\n", hr.TotalFindings)
	section := func(title string, n int, fix string, lines []string) {
		fmt.Fprintf(&out, "**%s：%d 条**", title, n)
		if n > 0 {
			fmt.Fprintf(&out, "（处置：%s）", fix)
		}
		out.WriteString("\n")
		for i, l := range lines {
			if i >= clCap(n) {
				fmt.Fprintf(&out, "- …共 %d 条\n", n)
				break
			}
			fmt.Fprintf(&out, "- %s\n", l)
		}
		out.WriteString("\n")
	}
	zero := make([]string, 0, len(hr.ZeroPrice))
	for _, f := range hr.ZeroPrice {
		zero = append(zero, fmt.Sprintf("[%s] %s（%s）%s", f.Code, f.Title, f.Unit, f.Detail))
	}
	section("零价资源（现行价与基准价均为 0）", len(hr.ZeroPrice), "cost_library set_price 或 save_resource 补价", zero)
	noPrice := make([]string, 0, len(hr.QuotaNoPrice))
	for _, f := range hr.QuotaNoPrice {
		noPrice = append(noPrice, fmt.Sprintf("定额 %s「%s」", f.Code, f.Title))
	}
	section("缺价定额子目（含量行单价为 0）", len(hr.QuotaNoPrice), "先给它引用的资源补价", noPrice)
	broken := make([]string, 0, len(hr.BrokenRef))
	for _, f := range hr.BrokenRef {
		broken = append(broken, fmt.Sprintf("资源编码 %s（定额子目「%s」）", f.Code, f.Title))
	}
	section("悬挂引用（含量行引用的资源不在库）", len(hr.BrokenRef), "cost_library save_resource 补建该编码的资源，或 save_quota 改含量行", broken)
	dup := make([]string, 0, len(hr.Duplicates))
	for _, g := range hr.Duplicates {
		dup = append(dup, fmt.Sprintf("%s「%s」%s", g.Kind, g.Title, g.Detail))
	}
	section("重复防护（身份唯一索引应在位）", len(hr.Duplicates), "身份唯一索引缺失，属库结构问题，如实告知用户", dup)
	mismatch := make([]string, 0, len(hr.CodeMismatch))
	for _, g := range hr.CodeMismatch {
		codes := make([]string, 0, len(g.Items))
		for _, it := range g.Items {
			codes = append(codes, fmt.Sprintf("%s(id=%d)", it.Code, it.ID))
		}
		mismatch = append(mismatch, fmt.Sprintf("「%s」：%s", g.Title, strings.Join(codes, " / ")))
	}
	section("编码未统一（同名资源编码风格混用）", len(hr.CodeMismatch), "确认保留哪一条后 cost_library dedupe 合并", mismatch)
	if hr.TotalFindings == 0 {
		out.WriteString("库级扫描全绿。")
	}
	return out.String(), nil
}
