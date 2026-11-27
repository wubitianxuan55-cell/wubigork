package app

// office_cost_library_schema.go — cost_library 的参数说明（2026-11-25 办公工具梳理批）。
//
// 模型每轮都为常驻 schema 付上下文，而 cost_library 一个工具曾占造价档工具面的
// 37%（1.4 万字符）。这里把参数分两层，出自同一份源文本：
//   - 常驻面 Schema()：动作路由 + 随时要用的入参（字典、定额、工料机、调价、
//     删除、导入会话与口径…）；
//   - 回执携带面：import_add 三组行的字段契约、classify 四组的字段契约——只在
//     用得上的时候发放，且发在流程必经的节点上（import_open 回执带行契约、dims
//     回执带 classify 契约；另可 action=help 随取），零额外往返。
// 字段名与 costLibraryArgs / workcost.Import* 的 json 标签逐字一致
// （TestCostLibraryArgsMatchDocumentedKeys 对 costLibraryDocumentedSchema 钉住）。
//
// 同批修的历史坑：旧 Schema 里 quota 对象少一个右花括号，resource/rows/
// classify/dict/import_* 等 31 个参数在模型眼里全嵌在 quota 里面——
// 「模型把 dict/classify 塞进 quota」（2026-11-25 铜陵反馈 #13）的根因在此。

import "encoding/json"

// clSchemaCoreA 常驻参数前段（动作、演练、字典扁平参数、定额、工料机、调价、
// 删除、去重）。
const clSchemaCoreA = `{
"type":"object",
"required":["action"],
"properties":{
  "action":{"type":"string","enum":["import_open","import_add","import_status","import_check","import_commit","import_discard","import_project","save_quota","delete_quota","save_resource","delete_resource","set_price","bulk_price","delete_items","delete_project","flywheel","dedupe","health","classify","borrow_quota","dims","dict","dict_add","dict_update","dict_delete","template","help"],"description":"动作名。录入资料 import_open → import_add → import_check → import_commit（import_status 回读暂存区）；综合单价表是三级树表的测算簿走 import_project。import_add 的行字段契约随 import_open 回执发放，classify 的随 dims 回执发放（或 action=help）。template 把测算样表复制到工作区；health/dims 只读；字典查看 dict，增改删 dict_add/dict_update/dict_delete"},
  "dry_run":{"type":"boolean","description":"演练：校验、匹配、统计照做，但不写库、不弹确认"},
  "path":{"type":"string","description":"测算工作簿路径（import_project 必填，.xlsx，综合单价表须是三级树表、其余表参考；template 可选=模版落盘路径，工作区相对，缺省落 .gaea/exports）"},
  "kind":{"type":"string","enum":["specialty","region","category"],"description":"字典类别（dict_add/dict_update/dict_delete 用）：specialty=专业 / region=地区 / category=工料机分类"},
  "name":{"type":"string","description":"字典项名称（dict_add/dict_update/dict_delete 用；分类=完整路径，如 材料/土建材料/钢材，新增分类要先有上级路径）"},
  "rename":{"type":"string","description":"dict_update：改成什么名字"},
  "code":{"type":"string","description":"专业的标准代码（dict_add/dict_update 可选，如国标清单专业代码 04=市政），机检清单编码专业段用"},
  "parent":{"type":"string","description":"上级字典项（dict_add/dict_update 可选；仅分类有层级；地区是平铺的省级行政区，无上级）"},
  "level":{"type":"integer","description":"层级（dict_add 可选，1=顶级）"},
  "sort":{"type":"integer","description":"排序（dict_update 可选）"},
  "quota":{"type":"object","description":"定额（save_quota 必填）","properties":{
    "code":{"type":"string","description":"定额编码（必填；已存在=更新）"},
    "title":{"type":"string","description":"定额名称"},
    "unit":{"type":"string","description":"计量单位"},
    "specialty":{"type":"string","description":"专业（字典值；新建定额时给。已有定额的专业固定，不改）"},
    "chapter":{"type":"string","description":"分部（本专业内的分部，归到已有分部叫法上；字段名沿用 chapter）"},
    "category_path":{"type":"string","description":"分类路径"},
    "source":{"type":"string","description":"来源（手工录入/企业定额…）"},
    "note":{"type":"string","description":"备注"},
    "items":{"type":"array","description":"工料机含量行（整组替换；每行引用一个在库的工料机编码；不给=保留原有含量行；新建定额必须给，可为空数组）","items":{"type":"object","properties":{
      "resource_code":{"type":"string","description":"工料机编码（必填，必须在库；需要新资源先用 save_resource 建）"},
      "quantity":{"type":"number","description":"含量（0=工序模板占位，不计费）"},
      "loss_rate":{"type":"number","description":"损耗率（0.03=3%）"},
      "note":{"type":"string","description":"备注"}
    }}
  }}},
  "resource":{"type":"object","description":"工料机（save_resource 必填；按 id 或 code 定位已有资源，未给的字段保持原值）","properties":{
    "id":{"type":"integer","description":"资源 id（改已有资源时给）"},
    "code":{"type":"string","description":"资源编码（按它定位已有资源；新建给了就照用、须未被占用，不给自动分配）"},
    "kind":{"type":"string","description":"类别：人工/材料/机械/外委"},
    "title":{"type":"string","description":"名称"},
    "spec":{"type":"string","description":"规格"},
    "unit":{"type":"string","description":"单位：工日/t/m³/台班"},
    "base_price":{"type":"number","description":"基准价（元）"},
    "current_price":{"type":"number","description":"现行价（元；核算取用价，现行价优先）"},
    "category_path":{"type":"string","description":"分类路径"},
    "source":{"type":"string","description":"来源：信息价/市场询价/企业定额"},
    "supplier":{"type":"string","description":"供应商"},
    "region":{"type":"string","description":"省份=省级行政区（如 四川省；只到省、不能下分；空=待整理。工料机按省分行：同一种材料每个省一行）"},
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
  "region":{"type":"string","description":"省份=省级行政区（如 四川省；只到省、不能下分）。set_price=这条价的省份；import_project=项目所在省份，工料机按它分行"},
  "specialty":{"type":"string","description":"专业（字典值）。import_project=项目默认专业，本次新建的定额专业随它；flywheel=只生成该专业的候选（同词干多专业时必给）；borrow_quota=目标专业"},
  "note":{"type":"string","description":"调价依据（set_price 可选）"},
  "control_price":{"type":"number","description":"import_project 可选：显式控制价（元），覆盖费用汇总里读到的值"},
  "rows":{"type":"array","description":"批量调价行（bulk_price 必填，单批 ≤500）","items":{"type":"object","properties":{
    "title":{"type":"string","description":"资源名称（精确匹配）"},
    "spec":{"type":"string","description":"规格（给了就参与精确匹配）"},
    "unit":{"type":"string","description":"单位（给了就参与精确匹配）"},
    "price":{"type":"number","description":"新单价（元）"},
    "period":{"type":"string","description":"期数"},
    "region":{"type":"string","description":"省份（省级行政区，如 四川省）"},
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
`

// clClassifySchema classify 的完整字段契约（不常驻；随 dims 回执与 help 发放）。
const clClassifySchema = `  "classify":{"type":"object","description":"批量整理归类（classify 用；每组 ≤500 条；没给的字段不动，值必须是字典里的）。先 dims 看现有叫法","properties":{
    "projects":{"type":"array","items":{"type":"object","properties":{
      "id":{"type":"integer","description":"项目 id（必填）"},
      "specialty":{"type":"string","description":"项目默认专业（字典值）"},
      "region":{"type":"string","description":"项目所在省份（省级行政区，如 四川省）"},
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
      "chapter":{"type":"string","description":"分部（本专业内的分部；字段名沿用 chapter）"}
    }}},
    "resources":{"type":"array","items":{"type":"object","properties":{
      "code":{"type":"string","description":"工料机编码（必填）"},
      "kind":{"type":"string","description":"类别，只在原类别不是四类之一时纠错"},
      "category_path":{"type":"string","description":"分类路径（字典值，如 材料/土建材料/钢材）"},
      "region":{"type":"string","description":"省份（省级行政区；只能给未标注地区的工料机补）"},
      "period":{"type":"string","description":"现行价的期数 YYYY-MM（只能给还没有期的工料机补）"}
    }}}
  }},
`

// clSchemaCoreB 常驻参数后段（借鉴生成、样表出口）。
const clSchemaCoreB = `  "from_code":{"type":"string","description":"借鉴生成的源定额编码（borrow_quota 必填，须是别的专业的定额）"},
  "new_code":{"type":"string","description":"借鉴生成的新定额编码（borrow_quota 可选，缺省 原码-N）"},
  "new_title":{"type":"string","description":"借鉴生成的新定额名称（borrow_quota 可选，缺省沿用源定额）"},
  "template":{"type":"object","description":"样表出口参数（action=template）：name 文件名（缺省 成本测算模版）、path 落盘路径（工作区相对，缺省 .gaea/exports）。版式、口径、工料机预填都由你在样表上完成——程序不再渲染","properties":{
    "name":{"type":"string"},"path":{"type":"string"}
  }},
`

// clClassifyStub classify 在常驻面的占位说明。
const clClassifyStub = `  "classify":{"type":"object","description":"批量整理归类（顶层参数）：projects / bills / quotas / resources 四组，每组 ≤500，没给的字段不动，值必须是字典里的。各组字段契约见 dims 回执或 action=help"},
`

// clImportHeadSchema 导入会话与口径（import_open / import_commit 等的常驻参数）。
const clImportHeadSchema = `
  "import_id":{"type":"string","description":"导入会话 id（import_open 返回；import_add / import_status / import_check / import_commit / import_discard 必填）"},
  "source":{"type":"string","description":"import_open：资料路径（工作区内）。只用来回读核对提交的数与文字，不解析它的结构；xlsx / csv 支持，其他格式无法回读"},
  "project":{"type":"object","description":"import_open：项目口径与封面（只导工料机 / 定额可以不给）。专业、地区是字典值，计价期 YYYY-MM","properties":{
    "name":{"type":"string","description":"项目名（要提交清单行就必须给）"},
    "file_name":{"type":"string","description":"资料文件名"},
    "location":{"type":"string","description":"封面地址（自由文本）"},
    "region":{"type":"string","description":"项目所在省份（省级行政区，如 四川省）——工料机按省分行，行里没给地区时用它"},
    "specialty":{"type":"string","description":"项目默认专业（字典值）——定额行没给专业时用它"},
    "price_period":{"type":"string","description":"计价期 YYYY-MM——行里没给期时用它"},
    "duration":{"type":"string","description":"工期"},
    "pricing":{"type":"string","description":"计价方式"},
    "management_rate":{"type":"number","description":"企业管理费率"},
    "regulatory_rate":{"type":"number","description":"规费费率"},
    "profit_rate":{"type":"number","description":"利润费率"},
    "tax_rate":{"type":"number","description":"增值税费率"},
    "control_price":{"type":"number","description":"控制价"}
  }},
  "controls":{"type":"object","description":"import_open：资料上的控制数（可选，给了才比）——你从资料上读到的行数与合计，程序拿来对暂存数据","properties":{
    "resources":{"type":"integer","description":"工料机价格表的行数"},
    "quotas":{"type":"integer","description":"定额子目（综合单价分析）的条数"},
    "bills":{"type":"integer","description":"清单行数"},
    "bill_total":{"type":"number","description":"清单合计 Σ(工程量 × 原表综合单价)"},
    "resource_price_sum":{"type":"number","description":"工料机单价合计"}
  }},
`

// clImportRowsSchema import_add 三组行的完整字段契约（不常驻；随 import_open
// 回执与 help 发放）。
const clImportRowsSchema = `  "resources":{"type":"array","description":"import_add：整理好的工料机行（每批 ≤200；被定额含量行引用的必须给 code）","items":{"type":"object","properties":{
    "code":{"type":"string","description":"工料机编码（不给=提交时自动分配）"},
    "kind":{"type":"string","description":"类别：人工/材料/机械/外委"},
    "title":{"type":"string","description":"名称（必填）"},
    "spec":{"type":"string","description":"规格"},
    "unit":{"type":"string","description":"单位"},
    "price":{"type":"number","description":"单价（同时作基准价与现行价）"},
    "category_path":{"type":"string","description":"分类路径（字典值；空=待整理）"},
    "region":{"type":"string","description":"省份（省级行政区；不给=项目地区）"},
    "period":{"type":"string","description":"现行价的期 YYYY-MM（不给=项目计价期）"},
    "price_type":{"type":"string","description":"价格口径：到场价/信息价/含税…"},
    "source":{"type":"string","description":"价格来源"},
    "note":{"type":"string","description":"备注"},
    "src":` + clSrcSchema + `,
    "derived":{"type":"array","items":{"type":"string"},"description":"由你换算、不是照抄的字段名（如 quantity）——回读时不核这些字段"}
  }}},
  "quotas":{"type":"array","description":"import_add：整理好的消耗定额（每批 ≤200）。同码=更新：重发只带改动的字段（没给的头字段保持原值、未给含量行保留；2026-11-27 复盘 P1/P6）","items":{"type":"object","properties":{
    "code":{"type":"string","description":"定额编码（必填；与库里的同码同专业=更新）"},
    "title":{"type":"string","description":"名称（新建必填；更新不给=保持原值）"},
    "unit":{"type":"string","description":"计量单位（更新不给=保持原值）"},
    "specialty":{"type":"string","description":"专业（字典值；新建不给=项目专业；更新不给=保持原值——显式给且与既有行不同才退回）"},
    "chapter":{"type":"string","description":"分部（专业内；字段名沿用 chapter；更新不给=保持原值）"},
    "category_path":{"type":"string","description":"分类路径（更新不给=保持原值）"},
    "source":{"type":"string","description":"来源（更新不给=保持原值）"},
    "note":{"type":"string","description":"备注（更新不给=保持原值）"},
    "items":{"type":"array","description":"含量行（引用的工料机必须已在库或已在暂存区；更新不给=保留原含量行，不会清空）","items":{"type":"object","properties":{
      "resource_code":{"type":"string","description":"工料机编码（必填）"},
      "quantity":{"type":"number","description":"含量"},
      "loss_rate":{"type":"number","description":"损耗率（0.03=3%）"},
      "note":{"type":"string","description":"备注"},
      "src":` + clSrcSchema + `
    }}},
    "src":` + clSrcSchema + `,
    "derived":{"type":"array","items":{"type":"string"},"description":"由你换算的字段名"}
  }}},
  "bills":{"type":"array","description":"import_add：整理好的清单行（每批 ≤200）","items":{"type":"object","properties":{
    "code":{"type":"string","description":"清单编码（空=自动 M 序号）"},
    "title":{"type":"string","description":"名称（必填）"},
    "unit":{"type":"string","description":"单位（必填）"},
    "division":{"type":"string","description":"分部"},
    "specialty":{"type":"string","description":"清单专业（字典值；不给=跟随项目专业）"},
    "quantity":{"type":"number","description":"工程量（必须 >0）"},
    "quantity_expr":{"type":"string","description":"工程量计算式（留痕）"},
    "quota_code":{"type":"string","description":"主定额编码"},
    "quota_codes":{"type":"array","items":{"type":"string"},"description":"其余定额编码（1:N 挂接）"},
    "quota_factors":{"type":"object","description":"定额换算系数 {定额编码: 系数}——每一个清单单位含几个定额单位（三级树表定额行的工程量列）；没列的=1。清单综合单价=Σ(定额单价×系数)，漏传则系数≠1 的定额金额偏低"},
    "feature":{"type":"string","description":"项目特征"},
    "price_override":{"type":"number","description":"手填单价（套定额的行给 0）"},
    "list_price":{"type":"number","description":"原表综合单价——单价对照用：程序现算后与它比"},
    "src":` + clSrcSchema + `,
    "derived":{"type":"array","items":{"type":"string"},"description":"由你换算的字段名"}
  }}}
`

// clImportRowsStub 三组行在常驻面的占位说明。
const clImportRowsStub = `  "resources":{"type":"array","description":"import_add：整理好的工料机行（每批 ≤200）。行字段契约见 import_open 回执或 action=help"},
  "quotas":{"type":"array","description":"import_add：整理好的消耗定额（每批 ≤200）。行字段契约见 import_open 回执或 action=help"},
  "bills":{"type":"array","description":"import_add：整理好的清单行（每批 ≤200）。行字段契约见 import_open 回执或 action=help"}
`

// clSchemaTail 收尾：关 properties、关根对象。
const clSchemaTail = `}
}`

// costLibraryResidentSchema 模型常驻可见的 schema（回执携带面用占位说明）。
func costLibraryResidentSchema() json.RawMessage {
	return json.RawMessage(clSchemaCoreA + clClassifyStub + clSchemaCoreB + clImportHeadSchema + clImportRowsStub + clSchemaTail)
}

// costLibraryDocumentedSchema 完整说明（常驻 + 回执携带面展开）：测试对账用——
// 模型最终看到的每个键都出自这里，与入参结构逐键核对。
func costLibraryDocumentedSchema() json.RawMessage {
	return json.RawMessage(clSchemaCoreA + clClassifySchema + clSchemaCoreB + clImportHeadSchema + clImportRowsSchema + clSchemaTail)
}

func (costLibraryTool) Schema() json.RawMessage { return costLibraryResidentSchema() }
