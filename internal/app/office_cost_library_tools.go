package app

// office_cost_library_tools.go — 造价库维护工具 cost_library（2026-11-25 造价数据库
// 推倒重建：板块只留浏览/检索/删除/导出，导入、解析、维护全部归办公「造价」
// 运行模式，与方案库「录入在方案模式、板块只管看」同一模式）。
//
// 三级数据 清单 → 定额 → 工料机 的写入口分工：
//   - cost_project_apply：测算编制的增量写（建项目/改费率/写清单行），只增不删；
//   - cost_library（本文件）：导入测算工作簿（六表）、生成测算模版、定额与工料机主数据增删改价、
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
	"time"

	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

type costLibraryTool struct{ a *App }

func (costLibraryTool) Name() string       { return "cost_library" }
func (costLibraryTool) ReadOnly() bool     { return false }
func (costLibraryTool) PersistWrite() bool { return true }

func (costLibraryTool) Description() string {
	return "维护工料机造价库（三级数据：清单 → 定额 → 工料机）。库定规矩：专业、地区、工料机分类是字典（dict 查看与新增），期数统一 YYYY-MM；写进库的值必须在字典里，空=待整理。【录入资料】造价资料格式不固定，由你读资料、整理后整批提交：import_open 开导入（给资料路径、项目口径、资料上的控制数）→ import_add 分批提交整理好的工料机 / 定额 / 清单行（每行带来源位置 src，不合规的行当场退回并说明原因）→ import_check 勾稽（程序回到资料里逐行核对你提交的数与文字，对控制数、对单价）→ 用户确认后 import_commit 写库。只有确认资料是标准测算工作簿（六表模版：封面/清单/费用汇总/综合单价/工料机价格/工程量计算；旧版五表不再兼容）时才可走 import_project 快速通道，演练回执出现「将被拒收」的行就改走模型读表。【测算模版】做新的成本测算先 template 生成六表空白测算表（口径 specialty/region/period/category 可选，工料机价格与费率按库预填），把路径交给用户；填法与测算口径见 cost-estimate 技能，填完走 import_project（先 dry_run=true）导回。【整理归类】文件不一定带专业/地区/分类：dims 盘点现有叫法，classify 分批归类（dry_run 出方案）；清单套定额同专业优先，本专业没有就 borrow_quota 以外专业定额为底生成本专业的。【维护】save_quota/delete_quota 维护消耗定额（含量行整组替换；定额专业固定，已有定额不改专业）；save_resource/delete_resource/set_price 维护工料机（工料机按地区分行；调价写历史并推进现行价）；bulk_price 批量调价；delete_items/delete_project 删清单行/整个项目；seed 资料条目资源化；flywheel 跨项目含量聚合生成企业定额（只在同专业内聚合）；dedupe 合并重复工料机；health 列库级健康发现。批量与删除动作先 dry_run=true 演练（不写库、不弹确认），核对后去掉 dry_run 正式执行；被引用的定额/工料机默认不删，确要删加 force=true。每次正式写入都需要用户确认。造价数据库板块只读，改库一律经本工具或 cost_project_apply，不要直连数据库。"
}

func (costLibraryTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"required":["action"],
"properties":{
  "action":{"type":"string","enum":["import_open","import_add","import_check","import_commit","import_discard","import_project","save_quota","delete_quota","save_resource","delete_resource","set_price","bulk_price","delete_items","delete_project","seed","flywheel","dedupe","health","classify","borrow_quota","dims","dict","template"],"description":"见工具说明；template 生成六表测算模版（只读库、只新增文件）；录入资料走模型读表导入 import_open → import_add → import_check → import_commit；health/dims 只读；dict 不给 dict 对象=查看字典"},
  "dry_run":{"type":"boolean","description":"演练：校验、匹配、统计照做，但不写库、不弹确认"},
  "path":{"type":"string","description":"测算工作簿路径（import_project 必填，.xlsx，六表模版）"},
  "quota":{"type":"object","description":"定额（save_quota 必填）","properties":{
    "code":{"type":"string","description":"定额编码（必填；已存在=更新）"},
    "title":{"type":"string","description":"定额名称"},
    "unit":{"type":"string","description":"计量单位"},
    "specialty":{"type":"string","description":"专业（字典值；新建定额时给。已有定额的专业固定，不改）"},
    "chapter":{"type":"string","description":"章节（本专业内的册章，归到已有章节叫法上）"},
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
    "region":{"type":"string","description":"地区（字典值；空=待整理。工料机按地区分行：同一种材料每个地区一行）"},
    "price_date":{"type":"string","description":"现行价的期数，YYYY-MM，如 2026-06"},
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
  "period":{"type":"string","description":"期数，YYYY-MM（set_price/import_project 可选；import_project 不给=导入当月）。信息价期数请换算成年月"},
  "region":{"type":"string","description":"地区（字典值）。set_price=这条价的地区；import_project=项目所在地区，工料机按它分行"},
  "specialty":{"type":"string","description":"专业（字典值）。import_project=项目默认专业，本次新建的定额专业随它；flywheel=只生成该专业的候选（同词干多专业时必给）；borrow_quota=目标专业"},
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
  "merge_ids":{"type":"array","items":{"type":"integer"},"description":"合并掉的工料机 id（dedupe 必填）"},
  "classify":{"type":"object","description":"批量整理归类（classify 用；每组 ≤500 条；没给的字段不动，值必须是字典里的）。先 dims 看现有叫法","properties":{
    "projects":{"type":"array","items":{"type":"object","properties":{
      "id":{"type":"integer","description":"项目 id（必填）"},
      "specialty":{"type":"string","description":"项目默认专业（字典值）"},
      "region":{"type":"string","description":"项目所在地区（字典值）"},
      "price_period":{"type":"string","description":"计价期，YYYY-MM"}
    }}},
    "bills":{"type":"array","items":{"type":"object","properties":{
      "id":{"type":"integer","description":"清单行 id（必填，cost_project action=items 可查）"},
      "specialty":{"type":"string","description":"清单专业（字典值；覆盖项目默认专业）"},
      "division":{"type":"string","description":"分部（整理后的叫法）"}
    }}},
    "quotas":{"type":"array","items":{"type":"object","properties":{
      "code":{"type":"string","description":"定额编码（必填）"},
      "specialty":{"type":"string","description":"专业——定额专业固定：只补空值，或按引用它的清单项纠错；对不上会进冲突清单，改用 borrow_quota"},
      "chapter":{"type":"string","description":"章节（本专业内的册章）"}
    }}},
    "resources":{"type":"array","items":{"type":"object","properties":{
      "code":{"type":"string","description":"工料机编码（必填）"},
      "kind":{"type":"string","description":"类别，只在原类别不是四类之一时纠错"},
      "category_path":{"type":"string","description":"分类路径（字典值，如 材料/土建材料/钢材）"},
      "region":{"type":"string","description":"地区（字典值；只能给未标注地区的工料机补）"},
      "period":{"type":"string","description":"现行价的期数 YYYY-MM（只能给还没有期的工料机补）"}
    }}}
  }},
  "from_code":{"type":"string","description":"借鉴生成的源定额编码（borrow_quota 必填，须是别的专业的定额）"},
  "new_code":{"type":"string","description":"借鉴生成的新定额编码（borrow_quota 可选，缺省 原码-N）"},
  "new_title":{"type":"string","description":"借鉴生成的新定额名称（borrow_quota 可选，缺省沿用源定额）"},
  "template":{"type":"object","description":"测算模版口径（action=template；都可不给=不限）： specialty 专业、region 地区、period 计价期（YYYY-MM）、kind 工料机类别、category 工料机分类（给上级连带下级）、name 文件名","properties":{
    "specialty":{"type":"string"},"region":{"type":"string"},"period":{"type":"string"},
    "kind":{"type":"string","enum":["人工","材料","机械","外委"]},"category":{"type":"string"},"name":{"type":"string"}
  }},
  "dict":{"type":"object","description":"字典：专业 specialty / 地区 region / 工料机分类 category。不给本对象或 op=list=查看；add=新增；update=改名/改上级/改标准代码（改名级联到所有引用它的数据）；delete=删除（仍被引用的不能删）","properties":{
    "op":{"type":"string","enum":["list","add","update","delete"]},
    "kind":{"type":"string","enum":["specialty","region","category"]},
    "name":{"type":"string","description":"字典项名称（分类=完整路径，如 材料/土建材料/钢材；新增分类要先有上级路径）"},
    "rename":{"type":"string","description":"update：改成什么名字"},
    "code":{"type":"string","description":"专业的标准代码（如国标清单专业代码 04=市政），机检清单编码专业段用"},
    "parent":{"type":"string","description":"上级字典项（地区：省→市→价格区）"},
    "level":{"type":"integer","description":"层级（1=顶级）"},
    "sort":{"type":"integer","description":"排序"},
    "note":{"type":"string","description":"备注"}
  }},
` + clImportSchema + `}
}}`)
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

	// 导入口径 / 飞轮专业筛选 / 借鉴生成的目标专业（规范值，字典校验在存储层）。
	Specialty string `json:"specialty"`
	// 借鉴生成（borrow_quota）。
	FromCode string `json:"from_code"`
	NewCode  string `json:"new_code"`
	NewTitle string `json:"new_title"`
	// 整理归类（classify）。
	Classify *workcost.ClassifyPlan `json:"classify"`
	// 字典（dict）。
	Dict *costLibraryDict `json:"dict"`
	// Template 测算模版口径（action=template；不给=不限口径）。
	Template *costLibraryTemplate `json:"template"`

	// 模型读表导入（import_*）。JSON 键与工具说明逐字一致（有用例钉住）。
	ImportID  string                    `json:"import_id"`
	Source    string                    `json:"source"`
	Project   *workcost.ImportProject   `json:"project"`
	Controls  *workcost.ImportControls  `json:"controls"`
	Resources []workcost.ImportResource `json:"resources"`
	Quotas    []workcost.ImportQuota    `json:"quotas"`
	Bills     []workcost.ImportBill     `json:"bills"`
}

// costLibraryDict 字典操作的入参。op 空=list。update 时没给的字段保持原值
// （指针区分「没给」）。
type costLibraryDict struct {
	Op     string  `json:"op"`
	Kind   string  `json:"kind"`
	Name   string  `json:"name"`
	Rename string  `json:"rename"`
	Code   string  `json:"code"`
	Parent string  `json:"parent"`
	Level  int     `json:"level"`
	Sort   *int    `json:"sort"`
	Note   *string `json:"note"`
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
	"import_project": "导入测算工作簿（六表）", "template": "生成测算模版", "save_quota": "保存定额", "delete_quota": "删除定额",
	"save_resource": "保存工料机", "delete_resource": "删除工料机", "set_price": "工料机调价",
	"bulk_price": "批量调价", "delete_items": "删除清单行", "delete_project": "删除项目",
	"seed": "资料条目资源化", "flywheel": "自生成企业定额", "dedupe": "工料机去重", "health": "库级健康扫描",
	"classify": "整理归类", "borrow_quota": "借鉴生成定额", "dims": "维度盘点", "dict": "字典",
	"import_open": "开导入", "import_add": "暂存导入行", "import_check": "导入勾稽", "import_commit": "提交导入", "import_discard": "丢弃导入",
	"dict_add": "新增字典项", "dict_update": "修改字典项", "dict_delete": "删除字典项",
}

func (t costLibraryTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p costLibraryArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("cost_library 参数解析失败: %w", err)
	}
	if _, ok := costLibraryActionLabel[p.Action]; !ok {
		return "", fmt.Errorf("未知 action %q（录入资料 import_open/import_add/import_check/import_commit；import_project/save_quota/delete_quota/save_resource/delete_resource/set_price/bulk_price/delete_items/delete_project/seed/flywheel/dedupe/health/classify/borrow_quota/dims/dict/template）", p.Action)
	}
	s := t.a.hubWorkcostStore()
	if !s.Available() {
		return "", fmt.Errorf("工料机造价库不可用（Hephaestus.db 未就绪）")
	}
	switch p.Action {
	case "import_project":
		return t.importProject(s, p)
	case "template":
		return t.template(s, p)
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
	case "classify":
		return t.classify(s, p)
	case "borrow_quota":
		return t.borrowQuota(s, p)
	case "dims":
		return t.dims(s)
	case "dict":
		return t.dict(s, p)
	case "import_open":
		return t.importOpen(s, p)
	case "import_add":
		return t.importAdd(s, p)
	case "import_check":
		return t.importCheck(s, p)
	case "import_commit":
		return t.importCommit(s, p)
	case "import_discard":
		return t.importDiscard(s, p)
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

// ── import_project：导入测算工作簿（六表模版） ────────────────────────

func (t costLibraryTool) importProject(s *workcost.Store, p costLibraryArgs) (string, error) {
	rel := strings.TrimSpace(p.Path)
	if rel == "" {
		return "", fmt.Errorf("import_project 需要工作簿路径 path（.xlsx）")
	}
	if !strings.EqualFold(filepath.Ext(rel), ".xlsx") {
		return "", fmt.Errorf("import_project 只收 .xlsx 测算工作簿（六表模版；收到 %s）", filepath.Ext(rel))
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
	// 导入口径是规范值（库定规矩）：专业、地区必须在字典里，计价期 YYYY-MM。
	// 文件或用户没给就留空（待整理），不从文件里猜；给了不对的当场拒收。
	b.Specialty = strings.TrimSpace(p.Specialty)
	b.Region = strings.TrimSpace(p.Region)
	b.Period = strings.TrimSpace(p.Period)
	if b.Period == "" {
		b.Period = workcost.CurrentPeriod()
	}
	specialty, region, period, err := s.ImportScope(b)
	if err != nil {
		return "", fmt.Errorf("导入口径不合规矩，未导入：%w", err)
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
		fmt.Fprintf(&out, "｜封面地点 %s", b.Location)
	}
	if b.Duration != "" {
		fmt.Fprintf(&out, "｜工期 %s", b.Duration)
	}
	out.WriteString("\n")
	fmt.Fprintf(&out, "**导入口径**: 专业 %s｜地区 %s｜计价期 %s\n",
		clDim(specialty, "未标注"), clDim(region, "未标注"), period)
	out.WriteString("口径说明：专业是项目默认专业，本次新建的定额专业随它；地区决定工料机按地区分行；未标注的导入后用 classify 整理。\n")
	fmt.Fprintf(&out, "**规模**: 工料机 %d 条｜清单项 %d 条｜消耗行 %d 条", len(b.Resources), len(b.Items), len(b.Lines))
	if len(b.QuotaBlocks) > 0 {
		fmt.Fprintf(&out, "｜定额块 %d 个（三级行组：一个清单可挂多条定额）", len(b.QuotaBlocks))
	}
	out.WriteString("\n")
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
	// 文件里不合规矩的工料机行导入时会被拒收——演练就点名，不等到正式导入才暴露：
	// 类别不在四类之内，或编码根本不是编码（整格同一段说明文字的「行」）。
	var rejected []string
	for _, pr := range b.Resources {
		code, title := strings.TrimSpace(pr.Code), strings.TrimSpace(pr.Title)
		if code == "" && title == "" {
			continue
		}
		ok, why := workcost.CodeWellFormed(code)
		switch {
		case !ok:
			rejected = append(rejected, fmt.Sprintf("「%s」%s", clExcerpt(title, 24), why))
		case !workcost.IsKnownKind(pr.Kind):
			rejected = append(rejected, fmt.Sprintf("[%s] %s（类别「%s」）", code, title, pr.Kind))
		}
	}
	if len(rejected) > 0 {
		fmt.Fprintf(&out, "**将被拒收的工料机行 %d 条**（编码不是编码，或类别不在 人工/材料/机械/外委 之内；列前 %d 条）——说明文字请挪出数据区，类别请改成四类之一:\n", len(rejected), clCap(len(rejected)))
		for i, line := range rejected {
			if i >= clCap(len(rejected)) {
				break
			}
			fmt.Fprintf(&out, "- %s\n", line)
		}
	}
	if p.DryRun {
		out.WriteString("\n落库口径：工料机按身份与地区 UPSERT（编码被占则消歧成「码-N」，含量行跟着落规范编码）；定额按专业内统一编码（同专业同内容复用，外专业同码另立并记借鉴来源），综合单价表的定额块逐块落成定额，一个清单挂多条定额时按 1:N 落挂接（定额行的系数进该定额的工程量）；有价的工料机各记一条价格事实（期 + 地区）。确认无误后去掉 dry_run 正式导入。\n")
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
	if res.ResourceRenamed > 0 {
		fmt.Fprintf(&out, "**编码消歧**: %d 条工料机的编码已被别的资源占用，落成了「码-N」（含量行已跟着改指，不会悬空）\n", res.ResourceRenamed)
	}
	if res.QuotaDerived > 0 {
		fmt.Fprintf(&out, "**跨专业另立**: %d 条定额的同码定额属于别的专业，已另立本专业新定额并记借鉴来源\n", res.QuotaDerived)
	}
	fmt.Fprintf(&out, "**价格事实**: %d 条（期 %s%s）\n", res.PriceFacts, clDim(period, "无"),
		map[bool]string{true: "｜地区 " + region, false: ""}[region != ""])
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
	if specialty == "" || region == "" {
		out.WriteString("\n本次导入口径没标全：确认项目专业与地区后用 cost_library classify 补上（项目 + 它的清单行与新定额）。")
	}
	return out.String(), nil
}

// ── template：生成测算模版 ────────────────────────────────────────────

// costLibraryTemplate 测算模版的口径（action=template；都可不给=不限）。
type costLibraryTemplate struct {
	Specialty string `json:"specialty"`
	Region    string `json:"region"`
	Period    string `json:"period"`
	Kind      string `json:"kind"`
	Category  string `json:"category"`
	Name      string `json:"name"`
}

// template 按库生成六表测算模版，落到工作区 .gaea/exports。只读库、只新增一个
// 文件，不写库（控制面按只读动作放行）。口径里专业/地区不在字典里、期数不是年月
// 时当场拒——模版封面写明的口径必须是规范值，回导时 ImportScope 才不会拦下它。
func (t costLibraryTool) template(_ *workcost.Store, p costLibraryArgs) (string, error) {
	in := p.Template
	if in == nil {
		in = &costLibraryTemplate{}
	}
	filter := workcost.LibraryFilter{
		Specialty: strings.TrimSpace(in.Specialty), Region: strings.TrimSpace(in.Region),
		Period: strings.TrimSpace(in.Period), Kind: strings.TrimSpace(in.Kind), Category: strings.TrimSpace(in.Category),
	}
	if err := t.a.costTemplateScopeCheck(filter); err != nil {
		return "", fmt.Errorf("模版口径不合规矩，未生成：%w", err)
	}
	opt, err := t.a.costTemplateOptions(filter)
	if err != nil {
		return "", err
	}
	exportsDir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "成本测算模版"
	} else {
		// 给了名字就同时当封面标题——导回库时封面标题就是项目名，两者必须一致。
		opt.Project = name
	}
	out := filepath.Join(exportsDir, fmt.Sprintf("%s-%s.xlsx", sanitizeFileName(name), time.Now().Format("20060102-150405")))
	data, err := workcost.ExportProjectTemplate(opt)
	if err != nil {
		return "", fmt.Errorf("生成测算模版失败: %w", err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", fmt.Errorf("写出测算模版失败: %w", err)
	}
	written := out
	if rel, rerr := filepath.Rel(gaeaCwd(), out); rerr == nil && !strings.HasPrefix(rel, "..") {
		written = filepath.ToSlash(rel)
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "**测算模版**: %s\n", written)
	fmt.Fprintf(&buf, "**口径**: 专业 %s｜地区 %s｜计价期 %s｜工料机类别 %s｜分类 %s\n",
		clDim(filter.Specialty, "不限"), clDim(filter.Region, "不限"), clDim(filter.Period, "不限"),
		clDim(filter.Kind, "全部"), clDim(filter.Category, "全部"))
	fmt.Fprintf(&buf, "**工料机价格表**: 已从造价库预填 %d 条（现行价，带库内自有码；类别不在四类之内的存量行不预填）\n", len(opt.Resources))
	fmt.Fprintf(&buf, "**费用汇总表**: 企管 %s｜规费 %s｜利润 %s｜增值税 %s——%s\n",
		clPct(opt.Fee.ManagementRate), clPct(opt.Fee.RegulatoryRate), clPct(opt.Fee.ProfitRate), clPct(opt.Fee.TaxRate), opt.FeeSource)
	buf.WriteString("填法与测算口径见 cost-estimate 技能。要点：综合单价每组 = 1 清单汇总行 + 定额块（定额行填 单位/定额编码/定额名称/系数，明细行填 行类/资源编码/消耗量），一个清单套几条定额就建几个块；封面标题改成真实项目名；没填的组导回库时自动跳过。\n")
	fmt.Fprintf(&buf, "**下一步**: 把路径交给用户去填；填完用 cost_library import_project（path=%s，先 dry_run=true 演练，specialty/region 给本次项目的口径）导回库。", written)
	return buf.String(), nil
}

// costTemplateScopeCheck 模版口径过规矩：专业、地区必须在字典里（空=不限放行），
// 计价期须是 YYYY-MM（空=不限）。
func (a *App) costTemplateScopeCheck(f workcost.LibraryFilter) error {
	s := a.hubWorkcostStore()
	if !s.Available() {
		return fmt.Errorf("工料机造价库不可用")
	}
	if f.Period != "" {
		if _, ok := workcost.NormalizePeriod(f.Period); !ok {
			return fmt.Errorf("计价期「%s」不是年月写法——统一写成 YYYY-MM，如 2026-06", f.Period)
		}
	}
	if ok, err := s.DictHas(workcost.DictSpecialty, f.Specialty); err != nil {
		return err
	} else if f.Specialty != "" && !ok {
		return fmt.Errorf("专业「%s」不在专业字典里——先 dict 查看现有专业，或 cost_library action=dict op=add 新增（需用户确认）", f.Specialty)
	}
	if ok, err := s.DictHas(workcost.DictRegion, f.Region); err != nil {
		return err
	} else if f.Region != "" && !ok {
		return fmt.Errorf("地区「%s」不在地区字典里——先 dict 查看现有地区，或 cost_library action=dict op=add 新增（需用户确认）", f.Region)
	}
	return nil
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
		if in.Specialty != nil && strings.TrimSpace(*in.Specialty) != strings.TrimSpace(existing.Specialty) {
			// 定额专业是固有属性（随来源定、定了不改）：同样一件事各专业各有定额。
			return "", fmt.Errorf("定额 %s 已是「%s」专业，专业固定不改——要「%s」专业的请 cost_library borrow_quota（from_code=%s, specialty=%s）借鉴生成本专业的",
				code, existing.Specialty, strings.TrimSpace(*in.Specialty), code, strings.TrimSpace(*in.Specialty))
		}
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
	// 专业、期数等头字段过规矩（演练同样校验）。
	if err := s.ValidateQuota(&q); err != nil {
		return "", err
	}

	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "save_quota"))
	if isNew {
		fmt.Fprintf(&out, "**新建定额** %s「%s」（%s）｜专业 %s｜含量行 %d 行\n", code, q.Title, q.Unit, clDim(q.Specialty, "未标注"), len(q.Items))
	} else {
		how := "保留原有"
		if in.Items != nil {
			how = "整组替换为"
		}
		fmt.Fprintf(&out, "**更新定额** %s「%s」（%s）｜专业 %s｜含量行%s %d 行\n", code, q.Title, q.Unit, q.Specialty, how, len(q.Items))
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
	// 演练与落库走同一套规矩校验（类别四选一、地区/分类在字典里、期数 YYYY-MM）——
	// 不让演练显示通过、正式执行才被拒。
	if err := s.ValidateResource(&r); err != nil {
		return "", err
	}

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
	// 期数与地区先过规矩（演练同样校验）：期数归一成 YYYY-MM，地区在字典里。
	period, err := s.CheckPrice(r, p.Period, p.Region)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "set_price"))
	fmt.Fprintf(&out, "**[%s] %s（%s）**: %s → %s", r.Code, r.Title, r.Unit, clMoney(r.EffectivePrice()), clMoney(*p.Price))
	meta := []string{}
	if period != "" {
		meta = append(meta, "期数 "+period)
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
	saved, err := s.UpdateResourcePrice(r.ID, *p.Price, period, strings.TrimSpace(p.Region), priceType, "造价模式调价", strings.TrimSpace(p.Note))
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
		fmt.Fprintf(&out, "**同专业跨项目含量分歧簇 %d 个**｜不成立 %d 个（聚合只在专业内进行，不同专业的同词干定额是两个候选）\n", len(prev.Candidates), len(prev.Skipped))
		for i, c := range prev.Candidates {
			if i >= clCap(len(prev.Candidates)) {
				fmt.Fprintf(&out, "- …共 %d 个候选\n", len(prev.Candidates))
				break
			}
			fmt.Fprintf(&out, "- %s｜%s → %s「%s」（%s）: %d 个变体｜来源项目 %d 个\n", clDim(c.Specialty, "未标注专业"), c.OrigCode, c.NewCode, c.Title, c.Unit, len(c.Members), c.Sources)
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
		out.WriteString("\n挑出要生成的簇，用 stems=[origCode…] 去掉 dry_run 正式生成（ENT-* 企业定额，含量=各变体中位数）；同一词干在多个专业下各有一个候选时，带 specialty 指明专业。")
		return out.String(), nil
	}
	res, err := s.QuotaFlywheelApply(stems, strings.TrimSpace(p.Specialty))
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

// ── classify / borrow_quota / dims / dict：库定规矩与整理归类 ───────────

// clDim 维度值的显示：空值显示占位说明。
func clDim(v, empty string) string {
	if s := strings.TrimSpace(v); s != "" {
		return s
	}
	return empty
}

// clCleanIDs 去重保序的正整数 id 列表。
func clCleanIDs(in []int64) []int64 {
	var out []int64
	seen := map[int64]bool{}
	for _, id := range in {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// clExcerpt 截到 n 个字符（回执里引用文件原文用）。
func clExcerpt(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

// dims 维度盘点（只读）：模型整理归类前先看它对齐叫法，不另造新词。
func (t costLibraryTool) dims(s *workcost.Store) (string, error) {
	d, err := s.Dims()
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString("## 造价库维度盘点（只读）\n\n")
	dictLine := func(kind, label string, cap int) string {
		items := d.Dicts[kind]
		names := make([]string, 0, len(items))
		for i, it := range items {
			if i >= cap {
				names = append(names, fmt.Sprintf("…共 %d 项", len(items)))
				break
			}
			n := it.Name
			if it.Code != "" {
				n += "(" + it.Code + ")"
			}
			if it.Parent != "" {
				n += "↑" + it.Parent
			}
			names = append(names, n)
		}
		if len(names) == 0 {
			return fmt.Sprintf("%s 0 项", label)
		}
		return fmt.Sprintf("%s %d 项：%s", label, len(items), strings.Join(names, "、"))
	}
	fmt.Fprintf(&out, "**字典**:\n- %s\n- %s\n- %s\n",
		dictLine(workcost.DictSpecialty, "专业", 60), dictLine(workcost.DictRegion, "地区", 80),
		dictLine(workcost.DictCategory, "工料机分类", 40))
	if len(d.Projects) > 0 {
		out.WriteString("**项目**:\n")
		for _, p := range d.Projects {
			fmt.Fprintf(&out, "- #%d %s（清单 %d）｜专业 %s｜地区 %s｜计价期 %s｜封面地点 %s\n",
				p.ID, p.Name, p.ItemCount, clDim(p.Specialty, "未标注"), clDim(p.Region, "未标注"),
				clDim(p.PricePeriod, "未标注"), clDim(p.Location, "—"))
		}
	}
	dist := func(title string, cs []workcost.DimCount, cap int) {
		parts := make([]string, 0, len(cs))
		for i, c := range cs {
			if i >= cap {
				parts = append(parts, fmt.Sprintf("…共 %d 种取值", len(cs)))
				break
			}
			parts = append(parts, fmt.Sprintf("%s %d", clDim(c.Name, "未标注"), c.Count))
		}
		fmt.Fprintf(&out, "**%s**: %s\n", title, strings.Join(parts, "｜"))
	}
	dist("清单专业分布", d.BillSpecialties, 30)
	dist("定额专业分布", d.QuotaSpecialties, 30)
	if len(d.Chapters) > 0 {
		parts := make([]string, 0, len(d.Chapters))
		for i, c := range d.Chapters {
			if i >= 40 {
				parts = append(parts, "…")
				break
			}
			parts = append(parts, fmt.Sprintf("%s/%s %d", clDim(c.Specialty, "未标注"), c.Chapter, c.Count))
		}
		fmt.Fprintf(&out, "**定额章节分布**（专业/章节）: %s\n", strings.Join(parts, "｜"))
	}
	dist("工料机地区分布", d.Regions, 40)
	dist("工料机期数分布", d.Periods, 24)
	dist("工料机分类分布", d.Categories, 40)
	if len(d.Divisions) > 0 {
		out.WriteString("**清单分部分布**（项目 → 分部 条数）:\n")
		projectOf := map[int64]string{}
		for _, p := range d.Projects {
			projectOf[p.ID] = p.Name
		}
		groups := map[int64][]string{}
		var order []int64
		for _, v := range d.Divisions {
			if _, seen := groups[v.ProjectID]; !seen {
				order = append(order, v.ProjectID)
			}
			groups[v.ProjectID] = append(groups[v.ProjectID], fmt.Sprintf("%s %d", v.Division, v.Count))
		}
		for i, pid := range order {
			if i >= 12 {
				fmt.Fprintf(&out, "- …（共 %d 个项目有分部）\n", len(order))
				break
			}
			fmt.Fprintf(&out, "- #%d %s：%s\n", pid, projectOf[pid], strings.Join(groups[pid], "、"))
		}
	}
	p := d.Pending
	fmt.Fprintf(&out, "**待整理**: 项目未标专业 %d｜清单无专业 %d｜定额未标专业 %d｜定额未归章节 %d｜工料机未标地区 %d｜工料机未细分 %d｜工料机无期数 %d\n",
		p.ProjectSpecialty, p.BillSpecialty, p.QuotaSpecialty, p.QuotaChapter, p.ResourceRegion, p.ResourceCategory, p.ResourcePeriod)
	out.WriteString("\n规矩：专业、地区、工料机分类只能用字典里的值（没有的先 dict.op=add，经用户确认）；期数写 YYYY-MM；地区写到最细一级（信息价发布区挂在所属市下）。定额专业固定——classify 只补空值或按引用它的清单项纠错，对不上的用 borrow_quota 借鉴生成。")
	return out.String(), nil
}

// classify 批量整理归类（演练与正式共用同一套校验与冲突判定）。
func (t costLibraryTool) classify(s *workcost.Store, p costLibraryArgs) (string, error) {
	if p.Classify == nil {
		return "", fmt.Errorf("classify 需要 classify 对象（projects / bills / quotas / resources 四组，至少给一组；先 action=dims 看现有叫法）")
	}
	plan := *p.Classify
	for _, g := range []struct {
		name string
		n    int
	}{{"projects", len(plan.Projects)}, {"bills", len(plan.Bills)}, {"quotas", len(plan.Quotas)}, {"resources", len(plan.Resources)}} {
		if g.n > costLibraryMaxRows {
			return "", fmt.Errorf("classify.%s 单批最多 %d 条（当前 %d），请分批提交", g.name, costLibraryMaxRows, g.n)
		}
	}
	if len(plan.Projects)+len(plan.Bills)+len(plan.Quotas)+len(plan.Resources) == 0 {
		return "", fmt.Errorf("classify 四组都是空的：至少给一组要归类的条目")
	}
	var (
		res *workcost.ClassifyResult
		err error
	)
	if p.DryRun {
		res, err = s.PreviewClassify(plan)
	} else {
		res, err = s.ApplyClassify(plan)
	}
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "classify"))
	a := res.Applied
	fmt.Fprintf(&out, "**%s**: 项目 %d｜清单 %d｜定额 %d｜工料机 %d\n",
		map[bool]string{true: "将改动", false: "已改动"}[p.DryRun], a.Projects, a.Bills, a.Quotas, a.Resources)
	if n := len(res.Summary); n > 0 {
		fmt.Fprintf(&out, "**方案**（按目标值，共 %d 项）:\n", n)
		for i, sm := range res.Summary {
			if i >= clCap(n) {
				fmt.Fprintf(&out, "- …共 %d 项\n", n)
				break
			}
			fmt.Fprintf(&out, "- %s「%s」：%d 条\n", sm.Dimension, clDim(sm.Value, "未标注"), sm.Count)
		}
	} else {
		out.WriteString("没有需要改动的条目（都已是指定值，或全部进了冲突清单）。\n")
	}
	if n := len(res.Conflicts); n > 0 {
		fmt.Fprintf(&out, "**未执行 %d 条**（冲突，逐条裁决；列前 %d 条）:\n", n, clCap(n))
		for i, c := range res.Conflicts {
			if i >= clCap(n) {
				break
			}
			// 有目标值的冲突写成「字段 → 值」；条目本身不存在的只写原因。
			what := ""
			if c.Field != "" {
				what = " " + c.Field + " → " + c.Value
			}
			fmt.Fprintf(&out, "- [%s] %s「%s」%s：%s\n", c.Tier, c.Code, c.Title, what, c.Reason)
		}
	}
	if p.DryRun {
		out.WriteString("\n演练未写库。核对方案与冲突后去掉 dry_run 正式执行（弹确认）。")
	}
	return out.String(), nil
}

// borrowQuota 借鉴生成：以外专业定额为底生成本专业定额，可同时改挂清单行。
func (t costLibraryTool) borrowQuota(s *workcost.Store, p costLibraryArgs) (string, error) {
	if strings.TrimSpace(p.FromCode) == "" || strings.TrimSpace(p.Specialty) == "" {
		return "", fmt.Errorf("borrow_quota 需要 from_code（源定额编码）与 specialty（目标专业，字典值）；可选 new_code / new_title / item_ids（把哪些清单行改挂到新定额）")
	}
	res, err := s.BorrowQuota(workcost.BorrowRequest{
		FromCode: p.FromCode, Specialty: p.Specialty, NewCode: p.NewCode, Title: p.NewTitle,
		ItemIDs: clCleanIDs(p.ItemIDs),
	}, !p.DryRun)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString(clHead(p.DryRun, "borrow_quota"))
	fmt.Fprintf(&out, "**源定额** [%s]「%s」（%s，%s专业）｜含量行 %d 行\n",
		res.From.Code, res.From.Title, res.From.Unit, clDim(res.From.Specialty, "未标注"), len(res.From.Items))
	how := map[bool]string{true: "将生成本专业定额", false: "已生成本专业定额"}[p.DryRun]
	if res.Existed {
		how = "复用既有的本专业定额"
	}
	fmt.Fprintf(&out, "**%s** [%s]「%s」（%s专业）｜溯源：%s\n", how, res.New.Code, res.New.Title, res.New.Specialty, res.New.DerivedFrom)
	for i, it := range res.New.Items {
		if i >= 8 {
			fmt.Fprintf(&out, "- …共 %d 行\n", len(res.New.Items))
			break
		}
		fmt.Fprintf(&out, "- %s %s（%s）含量 %g\n", it.Kind, it.Title, it.Unit, it.Quantity)
	}
	if n := len(res.Relinked); n > 0 {
		cross := 0
		fmt.Fprintf(&out, "**%s改挂清单行 %d 条**（列前 %d 条）:\n",
			map[bool]string{true: "将", false: "已"}[p.DryRun], n, clCap(n))
		for i, r := range res.Relinked {
			if r.Specialty != "" && r.Specialty != res.New.Specialty {
				cross++
			}
			if i >= clCap(n) {
				continue
			}
			fmt.Fprintf(&out, "- [%s] %s（%s｜清单专业 %s｜定额工程量 %g）\n", r.Code, r.Title, r.Project, clDim(r.Specialty, "未标注"), r.Quantity)
		}
		if cross > 0 {
			fmt.Fprintf(&out, "注意：其中 %d 行的清单专业不是「%s」——改挂后 cost_audit 会点名清单与定额专业不一致，先用 classify 核对这些行的专业。\n", cross, res.New.Specialty)
		}
	}
	if n := len(res.Conflicts); n > 0 {
		fmt.Fprintf(&out, "**未改挂 %d 条**:\n", n)
		for i, c := range res.Conflicts {
			if i >= clCap(n) {
				break
			}
			fmt.Fprintf(&out, "- %s\n", c)
		}
	}
	if p.DryRun {
		out.WriteString("\n演练未写库。含量行照搬自源定额——按本项目特征核对；去掉 dry_run 正式生成并改挂（弹确认），要调含量再用 save_quota。")
	}
	return out.String(), nil
}

// dict 字典：查看 / 新增 / 修改 / 删除（专业、地区、工料机分类）。
func (t costLibraryTool) dict(s *workcost.Store, p costLibraryArgs) (string, error) {
	var in costLibraryDict
	if p.Dict != nil {
		in = *p.Dict
	}
	op := strings.TrimSpace(in.Op)
	if op == "" {
		op = "list"
	}
	in.Kind = strings.TrimSpace(in.Kind)
	in.Name = strings.TrimSpace(in.Name)
	in.Rename = strings.TrimSpace(in.Rename)
	in.Code = strings.TrimSpace(in.Code)
	in.Parent = strings.TrimSpace(in.Parent)
	switch op {
	case "list":
		items, err := s.DictList(in.Kind)
		if err != nil {
			return "", err
		}
		var out strings.Builder
		out.WriteString("## 造价库字典（只读）\n\n")
		for _, kind := range []string{workcost.DictSpecialty, workcost.DictRegion, workcost.DictCategory} {
			if in.Kind != "" && in.Kind != kind {
				continue
			}
			var lines []string
			for _, it := range items {
				if it.Kind != kind {
					continue
				}
				if kind == workcost.DictCategory && in.Kind == "" && len(lines) >= 12 {
					lines = append(lines, "…（看全部：dict.op=list 且 dict.kind=category）")
					break
				}
				n := it.Name
				if it.Code != "" {
					n += "（代码 " + it.Code + "）"
				}
				if it.Parent != "" {
					n += "↑" + it.Parent
				}
				if used, err := s.DictUsage(kind, it.Name); err == nil && used > 0 {
					n += fmt.Sprintf(" 被引用 %d", used)
				}
				lines = append(lines, n)
			}
			fmt.Fprintf(&out, "**%s %d 项**", workcost.DictKindLabel[kind], len(lines))
			if len(lines) > 0 {
				out.WriteString("：" + strings.Join(lines, "、"))
			}
			out.WriteString("\n")
		}
		out.WriteString("\n新增走 dict.op=add（经用户确认）；值必须先在字典里，才能写进数据。")
		return out.String(), nil
	case "add":
		if !workcost.ValidDictKind(in.Kind) {
			return "", fmt.Errorf("dict.kind 须为 specialty（专业）/ region（地区）/ category（工料机分类），收到 %q", in.Kind)
		}
		if in.Name == "" {
			return "", fmt.Errorf("新增%s需要 dict.name", workcost.DictKindLabel[in.Kind])
		}
		if in.Kind == workcost.DictCategory {
			// 分类是路径：每一段都得是字典项（库定规矩），上级不在就先加上级。
			if i := strings.LastIndex(in.Name, "/"); i > 0 {
				if parent := strings.TrimSpace(in.Name[:i]); parent != "" {
					if ok, err := s.DictHas(workcost.DictCategory, parent); err != nil {
						return "", err
					} else if !ok {
						return "", fmt.Errorf("上级分类「%s」不在字典里——先 dict.op=add 把上级加进去，再加「%s」", parent, in.Name)
					}
				}
			}
		}
		if in.Kind == workcost.DictRegion && in.Parent != "" {
			if ok, err := s.DictHas(workcost.DictRegion, in.Parent); err != nil {
				return "", err
			} else if !ok {
				return "", fmt.Errorf("上级地区「%s」不在字典里——先 dict.op=add 把上级加进去", in.Parent)
			}
		}
		exists, err := s.DictHas(in.Kind, in.Name)
		if err != nil {
			return "", err
		}
		var out strings.Builder
		out.WriteString(clHead(p.DryRun, "dict_add"))
		if exists {
			fmt.Fprintf(&out, "%s「%s」已在字典里，无需新增。", workcost.DictKindLabel[in.Kind], in.Name)
			return out.String(), nil
		}
		fmt.Fprintf(&out, "**将新增%s**「%s」", workcost.DictKindLabel[in.Kind], in.Name)
		if in.Code != "" {
			fmt.Fprintf(&out, "｜代码 %s", in.Code)
		}
		if in.Parent != "" {
			fmt.Fprintf(&out, "｜上级 %s", in.Parent)
		}
		out.WriteString("\n")
		if p.DryRun {
			out.WriteString("\n演练未写库。去掉 dry_run 正式新增（弹确认）。新增只是把值收进字典；已有数据要用它还得 classify。")
			return out.String(), nil
		}
		if err := s.DictAdd(workcost.DictItem{Kind: in.Kind, Name: in.Name, Code: in.Code, Parent: in.Parent,
			Level: in.Level, Sort: 0, Note: clDeref(in.Note)}); err != nil {
			return "", err
		}
		out.WriteString("已新增。接下来用 classify 把数据归到它下面。")
		return out.String(), nil
	case "update":
		if !workcost.ValidDictKind(in.Kind) || in.Name == "" {
			return "", fmt.Errorf("dict.op=update 需要 dict.kind 与 dict.name（改哪一条）")
		}
		items, err := s.DictList(in.Kind)
		if err != nil {
			return "", err
		}
		var cur *workcost.DictItem
		for i := range items {
			if items[i].Name == in.Name {
				cur = &items[i]
				break
			}
		}
		if cur == nil {
			return "", fmt.Errorf("%s字典里没有「%s」", workcost.DictKindLabel[in.Kind], in.Name)
		}
		sort := cur.Sort
		if in.Sort != nil {
			sort = *in.Sort
		}
		note := cur.Note
		if in.Note != nil {
			note = strings.TrimSpace(*in.Note)
		}
		parent := cur.Parent
		if in.Parent != "" {
			parent = in.Parent
		}
		code := cur.Code
		if in.Code != "" {
			code = in.Code
		}
		used, err := s.DictUsage(in.Kind, in.Name)
		if err != nil {
			return "", err
		}
		var out strings.Builder
		out.WriteString(clHead(p.DryRun, "dict_update"))
		fmt.Fprintf(&out, "**将修改%s**「%s」", workcost.DictKindLabel[in.Kind], in.Name)
		if in.Rename != "" {
			fmt.Fprintf(&out, " → 改名为「%s」（级联 %d 条引用它的数据）", in.Rename, used)
		}
		fmt.Fprintf(&out, "｜代码 %s｜上级 %s", clDim(code, "—"), clDim(parent, "—"))
		out.WriteString("\n")
		if p.DryRun {
			out.WriteString("\n演练未写库。去掉 dry_run 正式修改（弹确认）。")
			return out.String(), nil
		}
		cascaded, err := s.DictUpdate(in.Kind, in.Name, in.Rename, parent, code, sort, note)
		if err != nil {
			return "", err
		}
		out.WriteString("已修改。")
		for table, n := range cascaded {
			fmt.Fprintf(&out, " %s 改了 %d 条；", table, n)
		}
		return out.String(), nil
	default: // delete
		if !workcost.ValidDictKind(in.Kind) || in.Name == "" {
			return "", fmt.Errorf("dict.op=delete 需要 dict.kind 与 dict.name")
		}
		used, err := s.DictUsage(in.Kind, in.Name)
		if err != nil {
			return "", err
		}
		var out strings.Builder
		out.WriteString(clHead(p.DryRun, "dict_delete"))
		fmt.Fprintf(&out, "**将删除%s**「%s」", workcost.DictKindLabel[in.Kind], in.Name)
		if used > 0 {
			fmt.Fprintf(&out, "｜仍被 %d 条数据引用——先把它们 classify 到别的%s，才能删", used, workcost.DictKindLabel[in.Kind])
			out.WriteString("\n")
			return out.String(), nil
		}
		out.WriteString("\n")
		if p.DryRun {
			out.WriteString("\n演练未写库。去掉 dry_run 正式删除（弹确认）。")
			return out.String(), nil
		}
		if err := s.DictDelete(in.Kind, in.Name); err != nil {
			return "", err
		}
		out.WriteString("已删除。")
		return out.String(), nil
	}
}

func clDeref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
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
	section("编码未统一（同地区同名资源编码风格混用）", len(hr.CodeMismatch), "确认保留哪一条后 cost_library dedupe 合并", mismatch)
	rule := make([]string, 0, len(hr.RuleIssues))
	for _, r := range hr.RuleIssues {
		rule = append(rule, fmt.Sprintf("[%s] %s「%s」的 %s =「%s」：%s", r.Tier, r.Code, r.Title, r.Field, r.Value, r.Fix))
	}
	section("不合规矩的数据（取值在字典外 / 类别不在四类内 / 期数不是 YYYY-MM）", len(hr.RuleIssues), "按条目后的处置指引：classify 改数据，或 dict add 把值收进字典", rule)
	if hr.TotalFindings == 0 {
		out.WriteString("库级扫描全绿。")
	}
	return out.String(), nil
}
