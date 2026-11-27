package app

// office_runmode.go — 办公板块运行模式（标准/造价/方案）组合声明单表。
//
// 2026-11-21 众神殿终案（pan_1795200544148 正议 5 轮 + pan_1795202763988 续议
// 4 轮，四席+主席裁决链，用户拍板三档同批）：
//   - 三档=DSH agent preset 同构的「组合声明」：每档一份 {工具 Hide 面, 技能
//     extras, L1 模式行}，改档只加表行（将来投标/审计档）。
//   - 应用语义=DSH 全同构：会话创建时定档（新建会话若配置档 ≠ 引擎当前档，
//     先重建引擎再建会话）；当前会话不中途切换；切换=登记默认值（零重建零
//     缓存损失）。
//   - 标准档专用面 Registry.Hide（schema 不可见、协议层按名可调——Registry
//     hidden 既有语义）；造价/方案档互相收窄对方专用面。ToolHide 与 L1 模式
//     行必须同批上线（DSH-r5：缺提示行=能力无声消失）。
//   - 技能面=extras 制（wenxue_skills.go 先例）：五件专用技能只随对应档的
//     SkillExtras 进入引擎，standard 不传=天然不可见；不设硬 allowlist
//     （docx/pdf 等通用技能在造价/方案档仍可达——造价报告、方案附件是真实
//     交付面；硬名单的漏配风险由组合表+守卫测试钉住，见 office_runmode_test.go）。
//
// 数据面（续议场终审）：造价专用工具一律锚 workcost（工料机×定额=综合单价，
// 活重算链，用户定调）；costproject 是无退役标记的蒸馏旧面，不接 agent 工具
// （gaea_cost_projects.go 已加遗留面标记）。

import (
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// 办公运行模式三档（与 config.OfficeRunMode 归一口径一致）。
const (
	OfficeRunModeStandard = "standard"
	OfficeRunModeCost     = "cost"
	OfficeRunModeProposal = "proposal"
)

// 办公工具面两档（2026-11-21 众神殿 pan_1795224680124 终案第二轴）。
const (
	OfficeToolFaceLean     = "lean"     // 精简：新建会话默认，另关 browser+COM
	OfficeToolFaceStandard = "standard" // 标准：一键回，browser 开、产图开、COM 跟随本机
)

// toolGroupLabels 功能组中文名（面板/ModeLine 拼接共用单源）。
var toolGroupLabels = map[tool.ToolGroupName]string{
	tool.ToolGroupCAD:      "CAD 图纸",
	tool.ToolGroupSchedule: "进度计划",
	tool.ToolGroupBrowser:  "浏览器自动化",
	tool.ToolGroupCOM:      "COM 实时协同",
	tool.ToolGroupComputer: "电脑操控",
	tool.ToolGroupCost:     "造价专用",
	tool.ToolGroupLongdoc:  "长文方案",
	tool.ToolGroupMedia:    "图面与生图",
}

// ToolGroupLabel 返回组中文名（未知组返回原 id）。
func ToolGroupLabel(g tool.ToolGroupName) string {
	if s, ok := toolGroupLabels[g]; ok {
		return s
	}
	return string(g)
}

// officeCostToolNames 造价专用工具面（单源=tool.ToolGroupCost 组目录，2026-11-21
// 工具面双轴批改挂——防第二份清单漂移）。守卫测试
// TestOfficeRunModeToolFaceAllRegistered 钉死与注册表对账。
var officeCostToolNames = tool.ToolGroupMembers(tool.ToolGroupCost)

// officeLongdocToolNames 方案专用工具面（单源=tool.ToolGroupLongdoc）。
var officeLongdocToolNames = tool.ToolGroupMembers(tool.ToolGroupLongdoc)

// officeRunModeHide 返回该档 run_mode 轴需 Registry.Hide 的专用工具名单
// （boot.Options.ToolHide 的 run_mode 分量；face/组覆盖分量走 composeToolHide）。
// 2026-11-21 用户拍板改排（覆盖 pan_1795224680124 初排）：CAD 并入造价档、
// 进度并入方案档，标准档两族全关：
//   - standard：造价族+方案族+CAD+进度全藏（通用办公不带行业/纵深面）；
//   - cost：造价族+CAD 开；藏方案族+进度；
//   - proposal：方案族+进度开；藏造价族+CAD。
func officeRunModeHide(mode string) []string {
	var names []string
	add := func(g tool.ToolGroupName, on bool) {
		if !on {
			names = append(names, tool.ToolGroupMembers(g)...)
		}
	}
	add(tool.ToolGroupCost, mode == OfficeRunModeCost)
	add(tool.ToolGroupLongdoc, mode == OfficeRunModeProposal)
	add(tool.ToolGroupCAD, mode == OfficeRunModeCost)
	add(tool.ToolGroupSchedule, mode == OfficeRunModeProposal)
	sort.Strings(names)
	return names
}

// officeRunModeSkillExtras 返回该档追加的内嵌技能（boot.Options.SkillExtras
// 缝，wenxue 先例）。standard 不传=专用技能不可见（现状逐字节）。
func officeRunModeSkillExtras(mode string) []skill.Skill {
	switch mode {
	case OfficeRunModeCost:
		return []skill.Skill{
			{
				Name:        "cost-estimation-playbook",
				Description: "成本测算工作流 — workcost 结构化测算五步（建项目→套定额/手填→机检→沉淀）与口径纪律（造价会话开场自读）",
				Scope:       skill.ScopeBuiltin,
				Path:        "(builtin)",
				Body:        officeCostEstimationPlaybookBody,
			},
			{
				Name:        "soil-remediation-costing",
				Description: "土壤修复成本检查清单 — 七项成本表/场地参数/规范号（HJ 25.1/GB 36600/HJ 25.3）编制要点；地方定额与法规限值不写死，查不到标缺",
				Scope:       skill.ScopeBuiltin,
				Path:        "(builtin)",
				Body:        officeSoilRemediationCostingBody,
			},
		}
	case OfficeRunModeProposal:
		return []skill.Skill{
			{
				Name:        "proposal-longform",
				Description: "长篇技术方案编制工作流 — 章节清单/评分点映射/来源抽取/交叉复核与 longdoc_manifest+longdoc_build 用法（方案会话开场自读）",
				Scope:       skill.ScopeBuiltin,
				Path:        "(builtin)",
				Body:        officeProposalLongformBody,
			},
			{
				Name:        "proposal-review",
				Description: "技术方案评审 — 章节完备性/数据一致性/评分点覆盖三查清单，交稿前必跑",
				Scope:       skill.ScopeBuiltin,
				Path:        "(builtin)",
				Body:        officeProposalReviewBody,
			},
			{
				Name:        "soil-remediation-proposal",
				Description: "土壤修复技术方案领域检查表 — 场地调查/修复目标/技术路线比选/施工组织/监测验收章节要点与规范引用",
				Scope:       skill.ScopeBuiltin,
				Path:        "(builtin)",
				Body:        officeSoilRemediationProposalBody,
			},
		}
	default:
		return nil
	}
}

// officeRunModeLine 返回该档的 L1 运行模式行（boot.Options.ModeLine 缝）。
// 口径=DSH-r5 裁决三段式：当前模式 + 本档未载入能力组 + 切档指引；加「误选
// 恢复」行为指令（不得静默降级）。三行都是常量文本——模式恒定期间 L1 前缀
// 逐字节稳定（缓存语义与 DSH KV Cache 一致）。
func officeRunModeLine(mode string) string {
	switch mode {
	case OfficeRunModeCost:
		return officeRunModeLineCost
	case OfficeRunModeProposal:
		return officeRunModeLineProposal
	default:
		return officeRunModeLineStandard
	}
}

const officeRunModeLineStandard = `【运行模式】标准（通用办公）。本档未装配专用工具面：造价族（cost_project/cost_project_apply/cost_audit 等，工程造价成本测算）、CAD 图纸（cad_ask）、进度计划（schedule_get/apply/analyze）与方案族（longdoc_manifest/longdoc_build，长篇技术方案编制；方案库 proposal_*）。用户提出工程造价成本测算/算量组价/成本机检需求时，提示其切换到「造价」运行模式（该档带 CAD）；提出进度计划编制/工期分析需求时，提示其切换到「方案」运行模式（该档带进度计划）；提出几十页以上技术方案编制需求时，提示其切换到「方案」运行模式（参数面板切换档位，开启新会话生效）。不得把造价测算静默降级成 xlsx 手算，也不得把百页方案塞进单次文档生成。`

const officeRunModeLineCost = `【运行模式】造价——聚焦工程造价成本测算与图纸算量。结构化测算锚 workcost 工料机数据面：cost_project 读项目/清单/定额/资源，cost_project_apply 写入（建项目/更新费率/套定额或手填价，写前同批编码查重），cost_audit 只读机检（行级硬检+库级信息项）；CAD 图纸识别走 cad_ask（图框质量闸+vision 读图）。新做测算优先在测算样表上编：cost_library action=template 取样表与《使用说明书》（先读），样表的「2 块 × 4 明细」只是示例槽位、数量没有上限；填完过 xlsx_recalc 与 import_project dry_run 两道门，经用户检查同意后才正式入库（草稿不入库）。造价库（清单→定额→工料机）的批量导入、归类与批量维护只在本档：用户要把造价资料录入造价库、删除库内数据、批量调价或归类时走 cost_library（先 dry_run 演练，正式执行需用户确认；被引用的定额/工料机默认不删）——单条改错（改字段/含量/挂接/调价/口径/费率）用户可在「造价数据库」板块详情点「编辑」直接改，也可以由你用工具做；板块里没有批量导入入口，不要直连数据库改库。交付落盘（常驻单源）：bash/openpyxl 直写 .xlsx/.docx/.pptx 到工作区正式位置会被发布边界拦截隔离（授权也拦）；合规路径=过程文件写 .gaea/work/（过程目录不受限）、正式名用 move_file 落位；被拦时按回执「隔离件」路径取回，不原路重试。涉及造价库导入、整理或维护前，先读 cost-estimation-playbook 技能（工作流、录入链与勾稽细则的出处）。本档未装配进度计划与长文方案专用工具（schedule_*/longdoc_*）；用户提出进度计划编制需求或几十页以上技术方案编制需求时，提示其切换到「方案」运行模式（参数面板切换档位，开启新会话生效）。`

const officeRunModeLineProposal = `【运行模式】方案——聚焦长篇技术方案编制与进度计划。longdoc_manifest 维护章节清单并做确定性机检（缺章/空章/重复/悬空来源/未映射评分点/未闭合占位符/清单漂移），longdoc_build 按清单分章组装整册并经交付模板链导出（页数用 doc_render 渲染取，哈希由回执给出）。方案库里的历史方案可作参照：动笔前用 proposal_search 检索（scope=title 看同类方案的章节骨架；关键词或自然语言问法查段落），命中后用 proposal_read 读对应小节（加 query 只取相关段落），对照其章节骨架与评分点口径；引用须注明出处方案，别的项目的工程量/人员/设备等专属数据不得照搬。用户要把历史方案录进方案库时：PDF 源不得直接转 Markdown（表格必散架）——先 to=docx 转 Word 交用户修正格式，确认后以修正版 docx 为源转 Markdown，过「列数一致/占位符/图引存在」三项机检全零才准录；docx/pptx 源直接 format_convert 转 Markdown（同时给 output 与 preserve_images=true，保留内嵌图片），再用 proposal_save 的 outline 形态——你只列认定的标题与层级（1~4 级，父级在前逐级展开，首项 1 级），正文由工具从源文件逐字切出并托管本地图片；长文档每批 100~200 个标题，首批之后 mode=append 续录（title 不变），最后一批 finalize=true；细则读 proposal-longform 技能。成稿经用户确认后同样用 proposal_save 录入方案库。进度计划走 schedule_get/apply/analyze（读改排程/CPM 分析，schedule-edit 技能纪律）。本档未装配造价专用工具与 CAD（cost_*/cad_ask）；用户提出工程造价成本测算/算量需求时，提示其切换到「造价」运行模式（参数面板切换档位，开启新会话生效）。`

// normalizeOfficeRunMode 归一运行模式字符串（app 层兜底；config.OfficeRunMode
// 已做同样归一，两处口径一致——守卫测试钉死）。
func normalizeOfficeRunMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case OfficeRunModeCost, OfficeRunModeProposal:
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return OfficeRunModeStandard
	}
}

// normalizeOfficeToolFace 归一工具面档（app 层兜底；config.OfficeToolFace 已做
// 同样归一——守卫测试钉死两处口径）。
func normalizeOfficeToolFace(face string) string {
	if strings.EqualFold(strings.TrimSpace(face), OfficeToolFaceStandard) {
		return OfficeToolFaceStandard
	}
	return OfficeToolFaceLean
}

// ── 工具面双轴组合器（2026-11-21 众神殿 pan_1795224680124 终案）──────────────
//
// 装配顺序（终案冻结）：run_mode → tool_face 默认 → 用户组覆盖 → 汇总 Hide
// 名单（排序固定）→ skillFilter 跟随最终启用组 → 实际隐藏组排序 → ModeLine
// 确定性拼接（相同配置逐字节相同；不写时间/计数/随机序）。

// composeToolHide 汇总 Registry.Hide 名单：CAD/进度两档默认关；browser/COM
// 精简档关、标准档开；造价/方案族按 run_mode；图面与生图两档默认开（用户
// 签注「产图不改默认」，可覆盖关）。computer 组跳过——它走 [computer_use]
// 注册闸（关=不注册），不进 Hide 名单。输出升序（确定性）。
func composeToolHide(runMode, face string, overrides map[string]bool) []string {
	hidden := map[string]bool{}
	addGroup := func(g tool.ToolGroupName, defOn bool) {
		on := defOn
		if v, ok := overrides[string(g)]; ok {
			on = v // 用户组覆盖显式优先（true=强制开/false=强制关）
		}
		if !on {
			for _, n := range tool.ToolGroupMembers(g) {
				hidden[n] = true
			}
		}
	}
	// 2026-11-21 用户拍板改排：CAD 随造价档、进度随方案档，标准档两族全关。
	addGroup(tool.ToolGroupCAD, runMode == OfficeRunModeCost)
	addGroup(tool.ToolGroupSchedule, runMode == OfficeRunModeProposal)
	addGroup(tool.ToolGroupBrowser, face == OfficeToolFaceStandard)
	addGroup(tool.ToolGroupCOM, face == OfficeToolFaceStandard)
	addGroup(tool.ToolGroupCost, runMode == OfficeRunModeCost)
	addGroup(tool.ToolGroupLongdoc, runMode == OfficeRunModeProposal)
	addGroup(tool.ToolGroupMedia, true)
	out := make([]string, 0, len(hidden))
	for name := range hidden {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// toolGroupFullyHidden 报告组全体成员是否都在隐藏集（组开关「启用」判据；
// 成员未注册的组——如本机无 Office 的 COM——按注册面判定，缺席≠隐藏）。
func toolGroupFullyHidden(hideSet map[string]bool, g tool.ToolGroupName) bool {
	for _, n := range tool.ToolGroupMembers(g) {
		if !hideSet[n] {
			return false
		}
	}
	return len(tool.ToolGroupMembers(g)) > 0
}

// composeModeLine 组装 L1 模式行（run_mode 基线行 + 【工具面】段）。终案 DSH-r3
// 修订：6 条常量只作两档预置，覆盖组合按「实际隐藏组（排序固定）」确定性拼
// 接。COM 组仅在本机装了 Office/WPS 时列名（未装机成员根本没注册，列了就是
// 说谎）。空串零注入由调用方保证（恒非空——两段都有内容）。
func composeModeLine(runMode, face string, hide []string) string {
	base := officeRunModeLine(runMode)
	hideSet := map[string]bool{}
	for _, n := range hide {
		hideSet[n] = true
	}
	var groups []string
	if toolGroupFullyHidden(hideSet, tool.ToolGroupCAD) {
		groups = append(groups, ToolGroupLabel(tool.ToolGroupCAD))
	}
	if toolGroupFullyHidden(hideSet, tool.ToolGroupSchedule) {
		groups = append(groups, ToolGroupLabel(tool.ToolGroupSchedule))
	}
	if toolGroupFullyHidden(hideSet, tool.ToolGroupBrowser) {
		groups = append(groups, ToolGroupLabel(tool.ToolGroupBrowser))
	}
	if toolGroupFullyHidden(hideSet, tool.ToolGroupCOM) && officeComAvailable() {
		groups = append(groups, ToolGroupLabel(tool.ToolGroupCOM))
	}
	faceName := "精简"
	if face == OfficeToolFaceStandard {
		faceName = "标准"
	}
	if len(groups) == 0 {
		return base + "\n【工具面】" + faceName + "——全部功能组在列（组开关在能力面板工具页登记，新会话生效）。"
	}
	return base + "\n【工具面】" + faceName + "——未载入组：" + strings.Join(groups, "、") +
		"（schema 不可见、协议层按名可调；组开关在能力面板工具页登记，新会话生效）。"
}

// officeRunModeSkillFilter 技能门（boot.Options.SkillFilter 窄 deny 谓词）：
// 拒绝名单 {schedule-edit, cad-takeoff}，**跟随最终启用组**——schedule 组开启
// 时 schedule-edit 恢复、cad 组开启时 cad-takeoff 恢复（终案 Codex-r3 修正：
// 过滤条件读最终启用组而非只读工具面）。两组全开=返回 nil（现状逐字节：
// install_skill 照常注册、run_skill 活读）。
func officeRunModeSkillFilter(hide []string) func(string) bool {
	hideSet := map[string]bool{}
	for _, n := range hide {
		hideSet[n] = true
	}
	scheduleOff := toolGroupFullyHidden(hideSet, tool.ToolGroupSchedule)
	cadOff := toolGroupFullyHidden(hideSet, tool.ToolGroupCAD)
	if !scheduleOff && !cadOff {
		return nil
	}
	return func(name string) bool {
		if scheduleOff && name == "schedule-edit" {
			return false
		}
		if cadOff && name == "cad-takeoff" {
			return false
		}
		return true
	}
}

// toolGroupOverridesSig 组覆盖签名（GaeaNewSession 判面重建的比较键：键排序
// 拼接，相同覆盖逐字节相同）。
func toolGroupOverridesSig(overrides map[string]bool) string {
	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+boolFlag(overrides[k]))
	}
	return strings.Join(parts, ",")
}

func boolFlag(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
