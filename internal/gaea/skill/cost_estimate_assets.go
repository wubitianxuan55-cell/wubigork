package skill

import _ "embed"

// cost_estimate_assets.go — cost-estimate 技能的样表资产（2026-11-25 用户拍板：
// 「删除渲染器这条思路……提供样本给模型，再进行约束」——测算模版的版式真源是
// 一份人工精修、实测可导回造价库的样表工作簿，不再是程序渲染器）。
//
// 双层取用（app 侧 costEstimateSample 实现）：
//  1. 工作区覆盖：`<工作区>/.gaea/skills/cost-estimate/样表-成本测算模版.xlsx`
//     存在则优先——用户改版式换个文件即可，不用等发版（领域模板进技能不进内核，
//     与 recalc.py 同一纪律）；
//  2. 内嵌种子：下面这份随程序走，任何工作区开箱即得。

//go:embed assets/cost-estimate-sample.xlsx
var costEstimateSampleXLSX []byte

// CostEstimateSampleXLSX 返回内嵌样表字节（xlsx）。
func CostEstimateSampleXLSX() []byte { return costEstimateSampleXLSX }

// CostEstimateSampleOverrideRel 是工作区样表覆盖路径（相对工作区根，正斜杠）。
const CostEstimateSampleOverrideRel = ".gaea/skills/cost-estimate/样表-成本测算模版.xlsx"
