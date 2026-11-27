package skill

import _ "embed"

// cost_estimate_assets.go — cost-estimate 技能的样表资产（2026-11-25 用户拍板：
// 「删除渲染器思路……提供样本给模型，再进行约束」——测算模版的版式真源是
// 一份人工精修、实测可导回造价库的样表工作簿，不再是程序渲染器）。
//
// 双层取用（app 侧 costEstimateSample 实现）：
//  1. 工作区覆盖：`<工作区>/.gaea/skills/cost-estimate/样表-成本测算模版.xlsx`
//     存在则优先——用户改版式换个文件即可，不用等发版（领域模板进技能不进内核，
//     与 recalc.py 同一纪律）；
//  2. 内嵌种子：下面这份随程序走，任何工作区开箱即得。
//
// 2026-11-27 用户令「给成本测算表增加一个使用说明书 md，一个和他绑定」：样表
// 旁绑定一份《使用说明书》，随样表一起发放（template 动作与板块导出同口径）。
// 样表的「2 块 × 4 明细」只是示例槽位——数量没有上限，这层用法与加行配方写在
// 说明书里；说明书可被工作区同名文件覆盖。

//go:embed assets/cost-estimate-sample.xlsx
var costEstimateSampleXLSX []byte

// CostEstimateSampleXLSX 返回内嵌样表字节（xlsx）。
func CostEstimateSampleXLSX() []byte { return costEstimateSampleXLSX }

// CostEstimateSampleOverrideRel 是工作区样表覆盖路径（相对工作区根，正斜杠）。
const CostEstimateSampleOverrideRel = ".gaea/skills/cost-estimate/样表-成本测算模版.xlsx"

//go:embed assets/成本测算表-使用说明书.md
var costEstimateManualMD []byte

// CostEstimateManual 返回内嵌的使用说明书字节（Markdown）。
func CostEstimateManual() []byte { return costEstimateManualMD }

// CostEstimateManualOverrideRel 是工作区说明书覆盖路径（相对工作区根，正斜杠）。
const CostEstimateManualOverrideRel = ".gaea/skills/cost-estimate/成本测算表-使用说明书.md"
