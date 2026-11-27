package app

// office_cost_library_tools.go — 造价库维护工具 cost_library（2026-11-25 造价数据库
// 推倒重建：板块只留浏览/检索/删除/导出，导入、解析、维护全部归办公「造价」
// 运行模式，与方案库「录入在方案模式、板块只管看」同一模式）。
//
// 三级数据 清单 → 定额 → 工料机 的写入口分工：
//   - cost_project_apply：测算编制的增量写（建项目/改费率/写清单行），只增不删；
//   - cost_library（本文件）：导入测算工作簿（六表）、生成测算模版、定额与工料机主数据增删改价、
//     批量调价、自生成企业定额、去重、删清单行/删项目、库级健康明细。
//
// 纪律：
//   - 标 PersistWrite（子代理不继承）；删除与批量覆盖类动作逐条过用户确认卡
//     （controller hardAsk），dry_run 演练不弹卡；
//   - 被引用的定额/工料机默认不删（force=true 才连带解除引用），不留悬空引用；
//   - 批量动作先全量校验再写入：任一条不合规整批拒绝，不留半批；
//   - 定额含量行给的资源编码必须在库——查不到就报错，绝不编造；
//   - 数据面一律锚 workcost（工料机 × 定额 = 综合单价，活重算链）。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/workcost"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

type costLibraryTool struct{ a *App }

func (costLibraryTool) Name() string       { return "cost_library" }
func (costLibraryTool) ReadOnly() bool     { return false }
func (costLibraryTool) PersistWrite() bool { return true }

// KeepSchemaDescriptions 本工具的参数语义写在字段描述里，经
// SchemaDescriptionsKeeper 保留描述下发——剥离后模型只能看键名猜结构，不知道
// import_project 也吃 specialty/region，单点浪费 9 轮（2026-11-25 铜陵反馈
// #13）。同反馈里「把 dict/classify 塞进 quota」另有根因：旧 schema 的 quota
// 对象少一个右花括号，后续参数在模型眼里就嵌在 quota 里（见
// office_cost_library_schema.go）。
func (costLibraryTool) KeepSchemaDescriptions() bool { return true }

// Description 只写动作路由与库定规矩。流程细则不在这里重复——身份提示词（造价档
// 「造价库维护」段）与 cost-estimate / cost-estimation-playbook 技能是出处；行
// 字段契约随回执发放（office_cost_library_schema.go）。
func (costLibraryTool) Description() string {
	return "维护工料机造价库（清单 → 定额 → 工料机三级数据）。【录入资料】资料格式不固定，由你读资料、整理后提交：import_open 开导入（回执带 import_add 的行字段契约）→ import_add 分批提交工料机/定额/清单行 → import_check 勾稽 → 用户确认后 import_commit 写库；综合单价表是三级树表的测算簿走 import_project 快速通道（综合单价表是唯一硬契约，其余表参考、可自由改版式）。【新做测算】template 取测算样表（版式真源），填完过 xlsx_recalc → import_project dry_run=true 导回。【整理归类】dims 盘点现有叫法（回执带 classify 契约）→ classify 分批给方案；本专业没有的定额用 borrow_quota 以外专业定额为底生成。【维护】save_quota/delete_quota 管定额；save_resource/delete_resource/set_price/bulk_price 管工料机与价格；delete_items/delete_project；flywheel/dedupe/health；字典 dict（查看）/dict_add/dict_update/dict_delete。【规矩】专业、地区、分类必须是字典值，期数 YYYY-MM；批量与删除先 dry_run=true 演练；被引用的默认不删（force=true 才解除引用）；每次正式写入都要用户确认。字段拿不准用 action=help 取契约；口径细则见 cost-estimate 技能。造价数据库板块详情有单条「编辑」钮（用户改单条错数据用）；批量导入、归类与批量维护一律经本工具或 cost_project_apply，不要直连数据库。"
}

// clActionSkeletons 各动作的正确参数骨架：UsageHint（loop guard：给刚失败的调用
// 一张骨架）与 action=help 共用同一份。
func clActionSkeletons() map[string]string {
	return map[string]string{
		"dict":           `{"action":"dict","kind":"region"}`,
		"dict_add":       `{"action":"dict_add","kind":"region","name":"铜陵"}`,
		"dict_update":    `{"action":"dict_update","kind":"region","name":"铜陵","rename":"铜陵市"}`,
		"dict_delete":    `{"action":"dict_delete","kind":"region","name":"铜陵"}`,
		"classify":       `{"action":"classify","dry_run":true,"classify":{"projects":[{"id":10,"specialty":"土壤修复"}]}}`,
		"save_quota":     `{"action":"save_quota","quota":{"code":"TR0001","title":"…","unit":"m3","specialty":"土壤修复","items":[{"resource_code":"R…","quantity":1.2}]}}`,
		"save_resource":  `{"action":"save_resource","resource":{"kind":"材料","title":"…","unit":"t","current_price":300,"region":"铜陵","price_date":"2026-06"}}`,
		"set_price":      `{"action":"set_price","resource_code":"R…","price":320,"period":"2026-06"}`,
		"import_open":    `{"action":"import_open","source":"资料/报价单.xlsx","project":{"name":"…","specialty":"土壤修复","region":"铜陵","price_period":"2026-06"},"controls":{"resources":120,"quotas":40,"bills":18}}`,
		"import_add":     `{"action":"import_add","import_id":"…","resources":[…],"quotas":[…],"bills":[…]}`,
		"import_project": `{"action":"import_project","path":".gaea/exports/….xlsx","specialty":"土壤修复","region":"铜陵","period":"2026-06","dry_run":true}`,
		"borrow_quota":   `{"action":"borrow_quota","from_code":"…","specialty":"土壤修复","dry_run":true}`,
		// 删除/去重类骨架（铜陵反馈 #16：这批动作的顶层列表参数连缺 6 次，
		// loop guard 只拦不导时模型没有骨架可抄）。
		"delete_quota":    `{"action":"delete_quota","codes":["D-01","D-02"],"dry_run":true}`,
		"delete_resource": `{"action":"delete_resource","ids":[3],"dry_run":true}`,
		"delete_items":    `{"action":"delete_items","project_id":10,"item_ids":[101,102],"dry_run":true}`,
		"dedupe":          `{"action":"dedupe","keep_id":1,"merge_ids":[2],"dry_run":true}`,
		"delete_project":  `{"action":"delete_project","project_id":10,"dry_run":true}`,
	}
}

// UsageHint 给刚失败的调用一张正确的参数骨架（loop guard 附加值，铜陵反馈
// #14：只拦不导，模型随后又原样撞了两次）。import_add / classify 的行与组字段
// 不在常驻 schema 里：失败时直接附上字段契约，省一次 help。
func (costLibraryTool) UsageHint(args json.RawMessage) string {
	var p struct {
		Action string `json:"action"`
	}
	_ = json.Unmarshal(args, &p)
	switch p.Action {
	case "import_add":
		return "cost_library 字段契约：\n" + clImportRowsContract
	case "classify":
		return "cost_library 字段契约：\n" + clClassifyContract
	}
	if s, ok := clActionSkeletons()[p.Action]; ok {
		return "cost_library 正确参数骨架：" + s
	}
	return ""
}

// costLibraryArgs cost_library 入参。需要区分「没传」的嵌套字段用指针。
type costLibraryArgs struct {
	Action string `json:"action"`
	DryRun bool   `json:"dry_run"`
	Path   string `json:"path"`

	// 字典扁平动作（dict_add/dict_update/dict_delete）：kind+name 直接给在顶层，
	// 与嵌套 dict 对象并存（2026-11-25 铜陵反馈 #1/#13：action=dict 与参数 dict
	// 同名、嵌套对象难发对，拆成独立动作走扁平参数）。
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Rename string `json:"rename"`
	Code   string `json:"code"`
	Parent string `json:"parent"`
	Level  int    `json:"level"`
	Sort   *int   `json:"sort"`

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
	// ControlPrice import_project 显式控制价（元）：覆盖费用汇总里读到的值
	// （2026-11-26 造价反馈，百锦路 #7）。
	ControlPrice *float64 `json:"control_price"`

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

	// Misnamed 非空=有嵌套参数从 quota 里救回（值=参数名），回执头点名。
	Misnamed string `json:"-"`
	// RecvKeys 本次调用收到的顶层键面（Execute 填），缺参报错附上。
	RecvKeys string `json:"-"`
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
	"import_project": "导入测算工作簿", "template": "生成测算模版", "save_quota": "保存定额", "delete_quota": "删除定额",
	"save_resource": "保存工料机", "delete_resource": "删除工料机", "set_price": "工料机调价",
	"bulk_price": "批量调价", "delete_items": "删除清单行", "delete_project": "删除项目",
	"flywheel": "自生成企业定额", "dedupe": "工料机去重", "health": "库级健康扫描",
	"classify": "整理归类", "borrow_quota": "借鉴生成定额", "dims": "维度盘点", "dict": "字典",
	"import_open": "开导入", "import_add": "暂存导入行", "import_check": "导入勾稽", "import_commit": "提交导入", "import_discard": "丢弃导入",
	"dict_add": "新增字典项", "dict_update": "修改字典项", "dict_delete": "删除字典项",
	"help": "参数契约",
}

func (t costLibraryTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p costLibraryArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("cost_library 参数解析失败: %w", err)
	}
	if _, ok := costLibraryActionLabel[p.Action]; !ok && p.Action != "help" {
		return "", fmt.Errorf("未知 action %q（录入资料 import_open/import_add/import_check/import_commit；import_project/save_quota/delete_quota/save_resource/delete_resource/set_price/bulk_price/delete_items/delete_project/flywheel/dedupe/health/classify/borrow_quota/dims/dict/dict_add/dict_update/dict_delete/template；参数契约 action=help）", p.Action)
	}
	// help 只读不碰库：不必等 store 就绪。
	if p.Action == "help" {
		return t.clHelpReceipt(), nil
	}
	p.RecvKeys = clReceivedKeys(args)
	t.rescueMisnamedNested(&p, args)
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
	case "dict", "dict_add", "dict_update", "dict_delete":
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

// rescueMisnamedNested 救回发错位置的参数（2026-11-25 铜陵反馈 #13/#16 实测）。
// 两类形态，都从原始 args 里探（强类型字段会静默丢掉未知键）：
//
//  1. 嵌套对象发进第一个嵌套槽 quota：{"action":"dict","quota":{"dict":{…}}}、
//     {"action":"classify","quota":{"classify":{…}}}（#13）。
//  2. 顶层标量/列表发进 quota：{"action":"delete_quota","quota":{"codes":[…]}}、
//     {"action":"import_project","quota":{"region":"…"}}（#16/#17，log 实证
//     delete_quota/dedupe/delete_resource/import_project 全中）；极端形态还有
//     同键重复 {"quota":{好},"quota":{}}（Go 解码后者胜，好内容被吃掉）与
//     "code":"[\"D-01\",…]"（数组 JSON 串进标量键）。
//
// 目标字段为空且能认出已知键时搬过去；救回了记进 p.Misnamed，回执头点一句，
// 让模型知道参数被挪过位。save_quota 的 quota 本就是主参数，不参与。
func (t costLibraryTool) rescueMisnamedNested(p *costLibraryArgs, args json.RawMessage) {
	if p.Action == "save_quota" {
		return
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(args, &top); err != nil {
		return
	}
	probe := clQuotaProbe(args, top)
	// probe 为空不提前返回：串化数组 code→codes（③）不依赖 quota。
	moved := ""
	// 嵌套对象（#13）：classify / dict。
	if raw, ok := probe["classify"]; ok && p.Classify == nil {
		var cp workcost.ClassifyPlan
		if err := json.Unmarshal(raw, &cp); err == nil &&
			len(cp.Projects)+len(cp.Bills)+len(cp.Quotas)+len(cp.Resources) > 0 {
			p.Classify = &cp
			moved = "classify"
		}
	}
	if raw, ok := probe["dict"]; ok && p.Dict == nil {
		var d costLibraryDict
		if err := json.Unmarshal(raw, &d); err == nil && (d.Op != "" || d.Kind != "" || d.Name != "") {
			p.Dict = &d
			moved = "dict"
		}
	}
	// 顶层标量/列表（#16/#17）：目标字段为空才搬，不覆盖调用方正着给的值。
	scoped := map[string]func(json.RawMessage) bool{
		"codes":      func(raw json.RawMessage) bool { return p.codesEmpty() && clDecodeStrings(raw, &p.Codes) },
		"ids":        func(raw json.RawMessage) bool { return len(p.IDs) == 0 && clDecodeInts(raw, &p.IDs) },
		"merge_ids":  func(raw json.RawMessage) bool { return len(p.MergeIDs) == 0 && clDecodeInts(raw, &p.MergeIDs) },
		"item_ids":   func(raw json.RawMessage) bool { return len(p.ItemIDs) == 0 && clDecodeInts(raw, &p.ItemIDs) },
		"keep_id":    func(raw json.RawMessage) bool { return p.KeepID <= 0 && clDecodeInt(raw, &p.KeepID) },
		"project_id": func(raw json.RawMessage) bool { return p.ProjectID <= 0 && clDecodeInt(raw, &p.ProjectID) },
		"force":      func(raw json.RawMessage) bool { return !p.Force && clDecodeBool(raw, &p.Force) },
		"region":     func(raw json.RawMessage) bool { return p.Region == "" && clDecodeString(raw, &p.Region) },
		"specialty":  func(raw json.RawMessage) bool { return p.Specialty == "" && clDecodeString(raw, &p.Specialty) },
		"period":     func(raw json.RawMessage) bool { return p.Period == "" && clDecodeString(raw, &p.Period) },
		"path":       func(raw json.RawMessage) bool { return p.Path == "" && clDecodeString(raw, &p.Path) },
		// dict_add/dict_update 扁平动作家族（同类错位一并救）。
		"kind":   func(raw json.RawMessage) bool { return p.Kind == "" && clDecodeString(raw, &p.Kind) },
		"name":   func(raw json.RawMessage) bool { return p.Name == "" && clDecodeString(raw, &p.Name) },
		"rename": func(raw json.RawMessage) bool { return p.Rename == "" && clDecodeString(raw, &p.Rename) },
		"parent": func(raw json.RawMessage) bool { return p.Parent == "" && clDecodeString(raw, &p.Parent) },
		"level":  func(raw json.RawMessage) bool { return p.Level == 0 && clDecodeIntVal(raw, &p.Level) },
		"sort":   func(raw json.RawMessage) bool { return p.Sort == nil && clDecodeIntPtr(raw, &p.Sort) },
		"code": func(raw json.RawMessage) bool {
			// delete_* 的 code 留给下面 code→codes 分支（那是数组串化的错拼，
			// 不是字典标准代码）。
			if p.Action == "delete_quota" || p.Action == "delete_resource" {
				return false
			}
			return p.Code == "" && clDecodeString(raw, &p.Code)
		},
	}
	names := make([]string, 0, 4)
	for key, adopt := range scoped {
		raw, ok := probe[key]
		if !ok || len(raw) == 0 {
			continue
		}
		if adopt(raw) {
			names = append(names, key)
		}
	}
	// "code":"[\"D-01\",…]"（#16 log 实证）：delete_* 动作下 code 是数组串化
	// 的错拼——按 JSON 数组或逗号/空白列表拆进 codes（顶层与 quota 内嵌都查）。
	// dict_add 的 code（字典标准代码）是合法标量，不受影响。
	if p.codesEmpty() && (p.Action == "delete_quota" || p.Action == "delete_resource") {
		for _, src := range []map[string]json.RawMessage{top, probe} {
			if raw, ok := src["code"]; ok && len(raw) > 0 {
				if clDecodeStrings(raw, &p.Codes) {
					names = append(names, "code→codes")
					break
				}
			}
		}
	}
	if moved != "" {
		names = append(names, moved)
		p.Quota = nil
	}
	if len(names) > 0 {
		sort.Strings(names)
		p.Misnamed = strings.Join(names, ",")
	}
}

// clQuotaProbe 汇总顶层所有 "quota" 对象出现次的键面（重复键容忍：Go 的 map
// 解码后者胜，{"quota":{好},"quota":{}} 会把好内容吃掉——铜陵会话 log 实证
// 过这个形态）。浅合并，键冲突时**非空者胜**；非 quota 顶层键不进 probe。
func clQuotaProbe(args json.RawMessage, top map[string]json.RawMessage) map[string]json.RawMessage {
	dec := json.NewDecoder(bytes.NewReader(args))
	dec.UseNumber()
	probe := map[string]json.RawMessage{}
	merge := func(v map[string]json.RawMessage) {
		for k, raw := range v {
			if prev, ok := probe[k]; ok && len(prev) > 0 && string(prev) != "null" {
				continue // 已有非空值，保留
			}
			probe[k] = raw
		}
	}
	if err := clScanKeyObjects(dec, "quota", merge); err != nil {
		// 兜底：流式扫描失败退回 map 解码（丢重复键容忍，不丢救回能力）。
		if raw, ok := top["quota"]; ok && len(raw) > 0 {
			var once map[string]json.RawMessage
			if json.Unmarshal(raw, &once) == nil {
				merge(once)
			}
		}
	}
	if raw, ok := top["quota"]; ok && len(raw) > 0 && len(probe) == 0 {
		var once map[string]json.RawMessage
		if json.Unmarshal(raw, &once) == nil {
			merge(once)
		}
	}
	return probe
}

// clScanKeyObjects 在顶层对象里找出每一次 key 的对象值，逐个交给 seen（重复
// 键每次都回调）。只在 depth=1 工作，嵌套对象里的同名键不算。
func clScanKeyObjects(dec *json.Decoder, key string, seen func(map[string]json.RawMessage)) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("顶层不是对象")
	}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		name, _ := keyTok.(string)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return err
		}
		if name == key {
			var obj map[string]json.RawMessage
			if json.Unmarshal(value, &obj) == nil {
				seen(obj)
			}
		}
	}
	_, err = dec.Token() // 收尾 '}'
	return err
}

// clDecodeStrings 数组或「逗号/空白分隔的串」→ []string（#16 log 实证模型会
// 把 codes 发成串化数组或逗号列表）。
func clDecodeStrings(raw json.RawMessage, out *[]string) bool {
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		*out = arr
		return true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "[") {
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			*out = arr
			return true
		}
		return false
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '，' || r == ' ' || r == '\t' || r == ';' || r == '；' })
	if len(parts) == 0 {
		return false
	}
	*out = parts
	return true
}

// clDecodeInts 数组（或单元素标量）→ []int64。
func clDecodeInts(raw json.RawMessage, out *[]int64) bool {
	var arr []int64
	if err := json.Unmarshal(raw, &arr); err == nil {
		*out = arr
		return true
	}
	var one int64
	if err := json.Unmarshal(raw, &one); err == nil {
		*out = []int64{one}
		return true
	}
	return false
}

func clDecodeInt(raw json.RawMessage, out *int64) bool {
	var v int64
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	*out = v
	return true
}

func clDecodeIntVal(raw json.RawMessage, out *int) bool {
	var v int
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	*out = v
	return true
}

func clDecodeIntPtr(raw json.RawMessage, out **int) bool {
	var v int
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	*out = &v
	return true
}

func clDecodeBool(raw json.RawMessage, out *bool) bool {
	var v bool
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	*out = v
	return true
}

func clDecodeString(raw json.RawMessage, out *string) bool {
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	if strings.TrimSpace(v) == "" {
		return false
	}
	*out = v
	return true
}

func (p *costLibraryArgs) codesEmpty() bool { return len(p.Codes) == 0 }

// clRecvSuffix 缺参报错的键面附注（「（本次收到键：…）」；解析不出=空串）。
func clRecvSuffix(p costLibraryArgs) string {
	if p.RecvKeys == "" {
		return ""
	}
	return fmt.Sprintf("（本次收到键：%s——缺的键没送达，多半发进了别的键里；按骨架顶层直给）", p.RecvKeys)
}

// clRoundingNote 舍入口径对账（铜陵反馈 #20）：库内单价按 2 位小数入库、
// 三费逐段舍到分再求小计；工作簿若按全精度编制（公式链不 ROUND），导入后
// cost_project summary 与工作簿会出现尾差——实测 45 条清单差 747.65 元，
// 模型手算了 5 轮才确认是舍位不是口径分叉。演练期就从 bundle 模拟两条链
// 对比清单综合单价，命中即给 INFO 与对齐办法。
func clRoundingNote(b *workcost.ProjectBundle) string {
	factorOf := map[string]float64{}
	for _, blk := range b.QuotaBlocks {
		f := blk.Factor
		if f <= 0 {
			f = 1
		}
		factorOf[blk.ItemCode+"\x00"+blk.Code] = f
	}
	flagged := 0
	exposure := 0.0
	for _, item := range b.Items {
		if item.ListPrice <= 0 {
			continue
		}
		// 逐定额块聚合明细（QuotaCode 空=旧 1:1 形态：整项一条隐式块，系数 1）。
		type block struct{ raw, lib float64 }
		blocks := map[string]*block{}
		var order []string
		for _, l := range b.Lines {
			if l.ItemCode != item.Code {
				continue
			}
			key := l.QuotaCode
			blk, ok := blocks[key]
			if !ok {
				blk = &block{}
				blocks[key] = blk
				order = append(order, key)
			}
			amt := l.Quantity * l.Price // 工作簿合价口径（不含损耗列——六表明细无损耗）
			blk.raw += amt
			blk.lib += math.Round(l.Price*100) / 100 * l.Quantity
		}
		if len(blocks) == 0 {
			continue
		}
		simRaw, simLib := 0.0, 0.0
		for _, key := range order {
			f := 1.0
			if key != "" {
				if v, ok := factorOf[item.Code+"\x00"+key]; ok {
					f = v
				}
			}
			blk := blocks[key]
			simRaw += blk.raw * f
			// 库内链：单价 2 位入库 → 三费逐段舍到分 → 小计（综合单价）舍到分。
			simLib += math.Round((math.Round(blk.lib*100)/100)*f*100) / 100
		}
		diffRaw := math.Abs(item.ListPrice - simRaw)
		diffLib := math.Abs(item.ListPrice - simLib)
		if diffRaw <= 0.02 && diffLib > 0.05 {
			flagged++
			if item.Quantity > 0 {
				exposure += item.Quantity * (simLib - simRaw)
			}
		}
	}
	if flagged == 0 {
		return ""
	}
	return fmt.Sprintf("ℹ 舍入口径 INFO：%d 条清单的综合单价按全精度编制（工作簿链未 ROUND）。库内单价按 2 位小数入库、三费逐段舍到分——导入后 cost_project summary 与工作簿差约 %+.2f 元（尾差不是口径分叉）。对齐办法：按样表的三层 ROUND 补齐——明细 H=ROUND(F×G,2)、定额行 H=ROUND(SUM(明细 H),2)、清单行 H=ROUND(Σ(定额 H×换算系数),2)，清单表综合单价 =ROUND(综合单价!H<组清单行>,2)（实测 45 条清单可逐行吻合）。\n",
		flagged, exposure)
}

// clReceivedKeys 本次调用实际收到的顶层键（缺参报错附上，铜陵反馈 #16：模型
// 反复缺同一批键时，回显键面让它立刻看出「键没发出去/发错位置」）。
func clReceivedKeys(args json.RawMessage) string {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(args, &top); err != nil {
		return ""
	}
	keys := make([]string, 0, len(top))
	for k := range top {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
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

// clCapRef 「后续要拿去引用」的映射类清单的上限（码映射、地区并入/分行）。
// 这类行截断了模型就得另查库补齐（2026-11-26 造价反馈，太原 #7：15 条映射只
// 见 12 条），所以放得比一般明细宽得多。
func clCapRef(n int) int {
	if n > 60 {
		return 60
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

// sameResourceIdentity 身份相等（类别按库内归一口径比较）。地区在身份里
// （2026-11-25 工料机按地区分行）：同身份不同地区是两行。save_resource 此前
// 按四要素判重，带另一个地区新建同身份行被拒（2026-11-26 造价反馈，百锦路 #3）。
func sameResourceIdentity(a, b workcost.Resource) bool {
	return workcost.NormalizeKind(a.Kind) == workcost.NormalizeKind(b.Kind) &&
		strings.TrimSpace(a.Title) == strings.TrimSpace(b.Title) &&
		strings.TrimSpace(a.Spec) == strings.TrimSpace(b.Spec) &&
		strings.TrimSpace(a.Unit) == strings.TrimSpace(b.Unit) &&
		strings.TrimSpace(a.Region) == strings.TrimSpace(b.Region)
}

// clMergeResource 把入参合并到目标工料机上（target=nil 即按入参自身成一条新
// 资源）：没给的字段保持原值，类别归一，数值域校验。先定目标、再合并——
// 顺序反了会让没带的字段（价格等）落成 0。
func clMergeResource(target *workcost.Resource, in *costLibraryResource) (workcost.Resource, error) {
	r := workcost.Resource{Status: "现行"}
	if target != nil {
		r = *target
	} else if c := strings.TrimSpace(in.Code); c != "" {
		// 新建带码：照用（2026-11-26 造价反馈，百锦路 #3：显式提交 ALU-CD 被静默
		// 换成自动码，编码与调用意图不符、无事前提示）。已有行的编码不改
		// （编码是含量行的引用锚点），那一步由调用方先拒。
		r.Code = c
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

// ── import_project：导入测算工作簿（综合单价表是硬契约，其余表参考）──────

func (t costLibraryTool) importProject(s *workcost.Store, p costLibraryArgs) (string, error) {
	rel := strings.TrimSpace(p.Path)
	if rel == "" {
		return "", fmt.Errorf("import_project 需要工作簿路径 path（.xlsx）")
	}
	if !strings.EqualFold(filepath.Ext(rel), ".xlsx") {
		return "", fmt.Errorf("import_project 只收 .xlsx 测算工作簿（综合单价表须是三级树表；收到 %s）", filepath.Ext(rel))
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
	// 控制价显式覆盖（2026-11-26 造价反馈，百锦路 #7）：费用汇总读歪（把
	// 「控制价利用率」当成控制价）时给条正路，不必回去改表。
	controlExplicit := false
	if p.ControlPrice != nil && *p.ControlPrice >= 0 {
		b.Fee.ControlPrice = *p.ControlPrice
		controlExplicit = true
		kept := b.Warnings[:0]
		for _, w := range b.Warnings {
			if !strings.HasPrefix(w, workcost.ControlPriceSuspectPrefix) {
				kept = append(kept, w)
			}
		}
		b.Warnings = kept
	}
	// 导入口径是规范值（库定规矩）：专业、地区必须在字典里，计价期 YYYY-MM。
	// 调用方没给时拿封面自述兜底（2026-11-25 铜陵反馈 #3：模版封面写了口径，
	// 调用方漏给参数就直接「未标注」，导入后还得返工 classify）。封面值仍要过
	// 字典校验：不过关就退回待整理并回执点名，不硬拦导入。
	callerSpecialty, callerRegion := strings.TrimSpace(p.Specialty), strings.TrimSpace(p.Region)
	b.Specialty = callerSpecialty
	b.Region = callerRegion
	b.Period = strings.TrimSpace(p.Period)
	coverUsed := []string{}
	coverScopeWarn := ""
	if b.Specialty == "" && strings.TrimSpace(b.CoverSpecialty) != "" {
		b.Specialty = strings.TrimSpace(b.CoverSpecialty)
		coverUsed = append(coverUsed, "专业 "+b.Specialty)
	}
	if b.Region == "" && strings.TrimSpace(b.CoverRegion) != "" {
		b.Region = strings.TrimSpace(b.CoverRegion)
		coverUsed = append(coverUsed, "地区 "+b.Region)
	}
	if b.Period == "" {
		if cp := strings.TrimSpace(b.CoverPeriod); cp != "" {
			if norm, ok := workcost.NormalizePeriod(cp); ok {
				b.Period = norm
				coverUsed = append(coverUsed, "计价期 "+norm)
			}
		}
	}
	if b.Period == "" {
		b.Period = workcost.CurrentPeriod()
	}
	specialty, region, period, err := s.ImportScope(b)
	if err != nil && len(coverUsed) > 0 {
		// 封面兜底值不过关（不在字典/期数不规范）→ 退回调用方原值重试。
		coverRejected := err.Error()
		b.Specialty, b.Region = callerSpecialty, callerRegion
		if b.Period == "" || b.Period == strings.TrimSpace(b.CoverPeriod) {
			b.Period = workcost.CurrentPeriod()
		}
		if s2, r2, p2, e2 := s.ImportScope(b); e2 == nil {
			specialty, region, period, err = s2, r2, p2, e2
			coverScopeWarn = fmt.Sprintf("封面口径（%s）未采用：%s——按待整理导入，确认口径后 classify 补。", strings.Join(coverUsed, "｜"), coverRejected)
		}
	}
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
	if coverScopeWarn != "" {
		out.WriteString("⚠ " + coverScopeWarn + "\n")
	} else if len(coverUsed) > 0 {
		fmt.Fprintf(&out, "口径出处：%s 取自封面自述（调用方没给参数时的兜底）。\n", strings.Join(coverUsed, "｜"))
	}
	out.WriteString("口径说明：专业是项目默认专业，本次新建的定额专业随它；地区决定工料机按地区分行；未标注的导入后用 classify 整理。\n")
	if note := clRoundingNote(b); note != "" {
		out.WriteString(note)
	}
	fmt.Fprintf(&out, "**规模**: 工料机 %d 条｜清单项 %d 条｜消耗行 %d 条", len(b.Resources), len(b.Items), len(b.Lines))
	if len(b.QuotaBlocks) > 0 {
		fmt.Fprintf(&out, "｜定额块 %d 个（三级树表：一个清单可挂多条定额）", len(b.QuotaBlocks))
	}
	out.WriteString("\n")
	// 工程量来源写明：清单表是参考表、可以按单项工程改版式——回执说清读到了几条，
	// 剩下的是从综合单价「清单工程量」列兜底的，用户才知道改版式有没有被认出来。
	if n := len(b.Items); n > 0 {
		zero := 0
		for _, it := range b.Items {
			if it.Quantity == 0 {
				zero++
			}
		}
		switch rest := n - b.BillQuantityRows; {
		case rest == 0:
			fmt.Fprintf(&out, "**工程量来源**: 清单表 %d 条\n", n)
		case zero == 0:
			fmt.Fprintf(&out, "**工程量来源**: 清单表 %d 条｜综合单价「清单工程量」列 %d 条\n", b.BillQuantityRows, rest)
		default:
			fmt.Fprintf(&out, "**工程量来源**: 清单表 %d 条｜综合单价「清单工程量」列 %d 条｜无工程量 %d 条\n",
				b.BillQuantityRows, rest-zero, zero)
		}
	}
	fmt.Fprintf(&out, "**取费**: 企管 %s｜规费 %s｜利润 %s｜增值税 %s", clPct(b.Fee.ManagementRate), clPct(b.Fee.RegulatoryRate), clPct(b.Fee.ProfitRate), clPct(b.Fee.TaxRate))
	if b.Fee.ControlPrice > 0 {
		fmt.Fprintf(&out, "｜控制价 %s", clMoney(b.Fee.ControlPrice))
		if controlExplicit {
			out.WriteString("（调用方显式给定）")
		}
	}
	out.WriteString("\n")
	// 取费链对账（2026-11-25 铜陵反馈 #15）：拿工作簿费用汇总的缓存金额反推
	// 每段基数，与模版口径比；差异按性质拆开说，不让模型手算 5 轮再判错性质。
	if b.Fee.HasAmounts {
		if findings := workcost.CheckFeeChain(b.Fee); len(findings) > 0 {
			fmt.Fprintf(&out, "**取费链核对：%d 处与模版口径不同**（库内现算与六表导出按模版口径；导入后两账必差——确认口径后改工作簿费率列，或明确哪边为准）:\n", len(findings))
			for _, f := range findings {
				fmt.Fprintf(&out, "- %s：工作簿按「%s」=%.2f，模版/库内按「%s」应=%.2f（差 %+.2f）\n",
					f.Item, f.Workbook, f.Amount, f.Template, f.Expected, f.Diff)
			}
		} else {
			out.WriteString("**取费链核对**: 各段基数与模版口径一致（按工作簿缓存金额反推核对）。\n")
		}
	} else {
		out.WriteString("**取费链核对**: 工作簿费用汇总无缓存金额（未重算），导入后用 cost_project summary 与工作簿对账。\n")
	}
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
		// 逐行定位（表名+行号+内容摘要）：只给计数时，一句脚注被当成资源行
		// 要另写脚本逐行扫才找得到（2026-11-25 铜陵反馈 #4）。
		fmt.Fprintf(&out, "**跳过行明细**（列前 %d 条）:\n", clCap(len(b.Skipped)))
		for i, r := range b.Skipped {
			if i >= clCap(len(b.Skipped)) {
				break
			}
			if r.Excerpt != "" {
				fmt.Fprintf(&out, "- %s!%d｜%s｜%s\n", r.Sheet, r.Row, clExcerpt(r.Excerpt, 60), r.Reason)
			} else {
				fmt.Fprintf(&out, "- %s!%d｜%s\n", r.Sheet, r.Row, r.Reason)
			}
		}
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
		for i, rn := range res.Renames {
			if i >= clCapRef(len(res.Renames)) {
				fmt.Fprintf(&out, "- …共 %d 条\n", len(res.Renames))
				break
			}
			fmt.Fprintf(&out, "- %s\n", rn)
		}
	}
	if len(res.RegionMerged) > 0 {
		// 铜陵反馈 #18 的并入面（收窄后的口径见 regionMergeDecision）。
		fmt.Fprintf(&out, "**地区并入**: %d 条工料机按身份四要素并入既有行（本次没给地区→继承原行地区；原行未标地区或地区不是字典值→规范成导入地区；未新建）\n", len(res.RegionMerged))
		for i, m := range res.RegionMerged {
			if i >= clCapRef(len(res.RegionMerged)) {
				fmt.Fprintf(&out, "- …共 %d 条\n", len(res.RegionMerged))
				break
			}
			fmt.Fprintf(&out, "- %s\n", m)
		}
	}
	if len(res.RegionRows) > 0 {
		// 百锦路反馈 #3：同身份在别的地区已有规范行时不再改写原行地区。
		fmt.Fprintf(&out, "**按地区分行新建**: %d 条工料机同身份在别的地区已有行——原行不动，新建本地区行（含量行已改指新码）\n", len(res.RegionRows))
		for i, m := range res.RegionRows {
			if i >= clCapRef(len(res.RegionRows)) {
				fmt.Fprintf(&out, "- …共 %d 条\n", len(res.RegionRows))
				break
			}
			fmt.Fprintf(&out, "- %s\n", m)
		}
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

// ── template：测算样表（版式真源）────────────────────────────────────

// costLibraryTemplate 样表出口参数（action=template）。
type costLibraryTemplate struct {
	Name string `json:"name"`
	// Path 落盘路径（工作区相对；缺省 .gaea/exports 带时间戳）。顶层 path 同义
	// （2026-11-25 铜陵反馈 #5）。
	Path string `json:"path"`
}

// template 把测算样表复制到工作区（2026-11-25 用户拍板「删除渲染器这条思路，
// 提供样本给模型，再进行约束」）：版式真源是一份人工精修、实测可导回造价库的
// 样表；本动作只复制文件，不再程序渲染、不再预填库内工料机与费率——预填是
// 模型的活（查库填表）。只读库、只新增一个文件，不写库（控制面按只读动作放行）。
// 样表来源：工作区覆盖（.gaea/skills/cost-estimate/样表-成本测算模版.xlsx）优先，
// 内嵌种子兜底（见 skill.cost_estimate_assets.go）。
func (t costLibraryTool) template(_ *workcost.Store, p costLibraryArgs) (string, error) {
	in := p.Template
	if in == nil {
		in = &costLibraryTemplate{}
	}
	if strings.TrimSpace(in.Path) == "" {
		in.Path = p.Path // 顶层 path 同义
	}
	var out string
	if want := strings.TrimSpace(in.Path); want != "" {
		if !strings.EqualFold(filepath.Ext(want), ".xlsx") {
			return "", fmt.Errorf("template.path 只收 .xlsx（收到 %s）", filepath.Ext(want))
		}
		abs, err := wspath.ResolveRelWithin(gaeaCwd(), want)
		if err != nil {
			return "", fmt.Errorf("template.path 必须是工作区相对路径（拒 .. 穿越）：%s", want)
		}
		out = abs
	} else {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			name = "成本测算模版"
		}
		exportsDir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
		out = filepath.Join(exportsDir, fmt.Sprintf("%s-%s.xlsx", sanitizeFileName(name), time.Now().Format("20060102-150405")))
	}
	_, source, err := writeCostEstimateSample(out)
	if err != nil {
		return "", err
	}
	written := out
	if rel, rerr := filepath.Rel(gaeaCwd(), out); rerr == nil && !strings.HasPrefix(rel, "..") {
		written = filepath.ToSlash(rel)
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "**测算样表**: %s（来源：%s）\n", written, source)
	buf.WriteString("版式以此为准——样表是测算模版的版式真源（人工精修、实测可导回）。程序不渲染、不预填：工料机价格表是空白骨架，由你检索库价后逐行填（cost_project action=resources / cost_library dims 查库，明细行按「编码」列引用、价格经公式带出；填了编码与名称的行单价不能空，空价按 0 计）；费用汇总费率按项目填。\n")
	buf.WriteString("导入契约——只有综合单价表是硬约束，其余表是参考、可自由改版式（清单表按单项工程划分、加列、调列序都行）：\n")
	buf.WriteString("- 综合单价表：表名含「综合单价」；三级树表，表头要有 层级 / 编码 / 名称 / 工程量 列；行型只看「层级」列：清单 / 定额 / 人工·材料·机械·外委；组 = 1 清单行 + 定额块（块 = 1 定额行 + 明细行）；清单编码填在清单行的编码列。样表每组预置 2 块、每块 4 行明细：定额块数按清单的真实工序构成定，1 块完全合法（第二块留空，不要删行）；要 3 块以上、一块超过 4 条明细、或清单超过 30 条，做法见 cost-estimate 技能「组与块怎么增减」。\n")
	buf.WriteString("- 手填的列：层级 B、编码 C（清单编码 / 定额编码 / 工料机编码共用），清单行与定额行的名称 D、单位 E，明细行的含量 F（定额行的 F 是换算系数，样表预置 1）；明细行的名称 D、单位 E、单价 G 与各行的金额 / 合价 / 三费是公式，不要手填。清单条数定了先把末尾未用的组删掉再填（配方见技能），交付时没有空骨架。\n")
	buf.WriteString("- 公式链照样表复制（VLOOKUP 取价、ROUND 金额链、SUMIF 三费、系数乘定额金额），不许用手算数值覆盖公式；两层 ROUND 都要在——定额行 H=ROUND(SUM(明细 H),2)、清单行 H=ROUND(Σ(定额 H×系数),2)、清单表综合单价=ROUND(综合单价!H<组清单行>,2)。\n")
	buf.WriteString("- 其余表（参考）：清单表给清单项补工程量 / 分部，列名保留 编码 / 名称 / 单位 / 工程量 即可（「单项工程」列可顶替「分部」），没有这一行时工程量取综合单价「清单工程量」列（先 xlsx_recalc 才有值）；工料机价格表按 编码 / 类别 / 名称 / 单位 / 不含税单价 逐行填；封面 B2 写真实项目名（导回库即项目名，「项目名称」标签值与它同步）；费用汇总费率在 D17–D21（0 = 该段不计取）。\n")
	buf.WriteString("交付两道门（全过才算完成）：xlsx_recalc（回执自带审计，写「零错误，达到交付门」才算过，不必再调 xlsx_audit；回执单列的「空结果」公式不算问题，未用槽位的公式可用 xlsx_apply clear_where 清）→ cost_library import_project dry_run=true（回执核对规模与口径）。\n")
	fmt.Fprintf(&buf, "**下一步**: 按项目改内容（封面概况、清单、综合单价、费用汇总费率、工程量）；填完 import_project（path=%s，先 dry_run=true，specialty/region 给本项目口径）导回库。填法与口径见 cost-estimate 技能。", written)
	return buf.String(), nil
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
		return "", fmt.Errorf("delete_quota 需要 codes（定额编码列表，顶层直给）%s", clRecvSuffix(p))
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
	// 身份唯一：类别+名称+规格+单位+地区 相同的另一条资源不能并存（同身份
	// 不同地区是两行，工料机按地区分行）。
	for i := range all {
		if all[i].ID != r.ID && sameResourceIdentity(all[i], r) {
			return "", fmt.Errorf("身份（类别+名称+规格+单位+地区）与工料机 %s「%s」重复——同地区同身份只能有一行：直接改那一条（给它的 id 或 code）；要建另一个地区的行就换 region（工料机按地区分行），或调整规格/单位", all[i].Code, all[i].Title)
		}
	}
	// 给了编码的新建：编码不能被别的资源占用。
	if target == nil && r.Code != "" {
		for i := range all {
			if all[i].Code == r.Code {
				return "", fmt.Errorf("工料机编码 %s 已被「%s」占用——换一个编码，或不给 code 由程序分配", r.Code, all[i].Title)
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
		// 演练就把将落库的编码亮出来：新建带码时此前演练不显码、落库才见
		// （2026-11-26 造价反馈，百锦路 #3）。
		if r.Code != "" {
			fmt.Fprintf(&out, "**编码**: %s\n", r.Code)
		} else if target == nil {
			out.WriteString("**编码**: 未给，落库时自动分配（以正式回执为准）\n")
		}
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
		return "", fmt.Errorf("delete_resource 需要 ids 或 codes（工料机 id 或编码列表，至少一个，顶层直给）%s", clRecvSuffix(p))
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
		return "", fmt.Errorf("delete_items 需要 project_id 与 item_ids（清单行 id 列表，cost_project action=items 可查，顶层直给）%s", clRecvSuffix(p))
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

// ── flywheel / dedupe / health：库级维护 ─────────────────────

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
		return "", fmt.Errorf("dedupe 需要 keep_id（保留）与 merge_ids（合并掉的工料机 id，顶层直给）%s", clRecvSuffix(p))
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
		fmt.Fprintf(&out, "**定额分部分布**（专业/分部）: %s\n", strings.Join(parts, "｜"))
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
	fmt.Fprintf(&out, "**待整理**: 项目未标专业 %d｜清单无专业 %d｜定额未标专业 %d｜定额未归分部 %d｜工料机未标地区 %d｜工料机未细分 %d｜工料机无期数 %d\n",
		p.ProjectSpecialty, p.BillSpecialty, p.QuotaSpecialty, p.QuotaChapter, p.ResourceRegion, p.ResourceCategory, p.ResourcePeriod)
	// 待整理明细（2026-11-26 造价反馈，旺平 #6）：只给计数时，模型只能逐条枚举
	// 去找是哪几条。每类列前若干条，清单行带数字 id（classify.bills 要用）。
	pendingCounts := map[string]int{
		"ProjectSpecialty": p.ProjectSpecialty, "BillSpecialty": p.BillSpecialty,
		"QuotaSpecialty": p.QuotaSpecialty, "QuotaChapter": p.QuotaChapter,
		"ResourceRegion": p.ResourceRegion, "ResourceCategory": p.ResourceCategory, "ResourcePeriod": p.ResourcePeriod,
	}
	if samples, serr := s.PendingSamples(20); serr == nil {
		var lines []string
		for _, k := range workcost.PendingSampleKeys {
			items := samples[k.Field]
			if len(items) == 0 {
				continue
			}
			if n := pendingCounts[k.Field]; n > len(items) {
				items = append(items, fmt.Sprintf("…等 %d 条", n))
			}
			lines = append(lines, fmt.Sprintf("- %s：%s", k.Title, strings.Join(items, "、")))
		}
		if len(lines) > 0 {
			out.WriteString("**待整理明细**（每类列前 20 条；清单行的 id= 用在 classify.bills）:\n" + strings.Join(lines, "\n") + "\n")
		}
	}
	out.WriteString("\n规矩：专业、地区、工料机分类只能用字典里的值（没有的先 dict_add，经用户确认）；期数写 YYYY-MM；地区写到最细一级（信息价发布区挂在所属市下）。定额专业固定——classify 只补空值或按引用它的清单项纠错，对不上的用 borrow_quota 借鉴生成。")
	out.WriteString("\n\n" + clClassifyContract)
	return out.String(), nil
}

// classify 批量整理归类（演练与正式共用同一套校验与冲突判定）。
func (t costLibraryTool) classify(s *workcost.Store, p costLibraryArgs) (string, error) {
	if p.Classify == nil {
		return "", fmt.Errorf("classify 需要 classify 对象（顶层参数，projects / bills / quotas / resources 四组至少给一组）。字段契约：\n%s\n先 action=dims 看现有叫法", clClassifyContract)
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
	if p.Misnamed != "" {
		fmt.Fprintf(&out, "⚠ 参数已救回：%s 原本发在 quota 里，已挪到正确位置——这些键都是顶层参数，直接顶层直给。\n", p.Misnamed)
	}
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
// dict_add / dict_update / dict_delete 是独立扁平动作（kind+name 给在顶层，
// 2026-11-25 铜陵反馈 #1/#13），进来先并成同一入参形态再走 op 分支。
func (t costLibraryTool) dict(s *workcost.Store, p costLibraryArgs) (string, error) {
	var in costLibraryDict
	if p.Dict != nil {
		in = *p.Dict
	}
	switch p.Action {
	case "dict_add":
		in.Op = "add"
	case "dict_update":
		in.Op = "update"
	case "dict_delete":
		in.Op = "delete"
	}
	// 扁平顶层字段兜底（嵌套对象里显式给的优先）。
	if in.Kind == "" {
		in.Kind = p.Kind
	}
	if in.Name == "" {
		in.Name = p.Name
	}
	if in.Rename == "" {
		in.Rename = p.Rename
	}
	if in.Code == "" {
		in.Code = p.Code
	}
	if in.Parent == "" {
		in.Parent = p.Parent
	}
	if in.Level == 0 {
		in.Level = p.Level
	}
	if in.Sort == nil {
		in.Sort = p.Sort
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
					lines = append(lines, "…（看全部：action=dict 且 kind=category）")
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
		out.WriteString("\n新增走 dict_add（经用户确认）；值必须先在字典里，才能写进数据。")
		return out.String(), nil
	case "add":
		if !workcost.ValidDictKind(in.Kind) {
			return "", fmt.Errorf("字典新增需要 kind=specialty/region/category（收到 %q）。骨架：{\"action\":\"dict_add\",\"kind\":\"region\",\"name\":\"铜陵\"}", in.Kind)
		}
		if in.Name == "" {
			return "", fmt.Errorf("新增%s需要 name。骨架：{\"action\":\"dict_add\",\"kind\":\"%s\",\"name\":\"铜陵\"}", workcost.DictKindLabel[in.Kind], in.Kind)
		}
		if in.Kind == workcost.DictCategory {
			// 分类是路径：每一段都得是字典项（库定规矩），上级不在就先加上级。
			if i := strings.LastIndex(in.Name, "/"); i > 0 {
				if parent := strings.TrimSpace(in.Name[:i]); parent != "" {
					if ok, err := s.DictHas(workcost.DictCategory, parent); err != nil {
						return "", err
					} else if !ok {
						return "", fmt.Errorf("上级分类「%s」不在字典里——先用 dict_add 把上级加进去，再加「%s」", parent, in.Name)
					}
				}
			}
		}
		if in.Kind == workcost.DictRegion && in.Parent != "" {
			if ok, err := s.DictHas(workcost.DictRegion, in.Parent); err != nil {
				return "", err
			} else if !ok {
				return "", fmt.Errorf("上级地区「%s」不在字典里——先用 dict_add 把上级加进去", in.Parent)
			}
		}
		exists, err := s.DictHas(in.Kind, in.Name)
		if err != nil {
			return "", err
		}
		var out strings.Builder
		out.WriteString(clHead(p.DryRun, "dict_add"))
		if p.Misnamed != "" {
			fmt.Fprintf(&out, "⚠ 参数已救回：%s 原本发在 quota 里，已挪到正确位置——下次按骨架直接给顶层字段。\n", p.Misnamed)
		}
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
			return "", fmt.Errorf("dict_update 需要 kind 与 name（改哪一条）")
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
			return "", fmt.Errorf("dict_delete 需要 kind 与 name")
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
