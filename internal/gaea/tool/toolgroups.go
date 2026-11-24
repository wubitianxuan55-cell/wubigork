package tool

// toolgroups.go — 办公工具功能组目录（2026-11-21 众神殿 pan_1795224680124 终案）。
//
// 用户要的「组开关」按功能语义切（关 CAD 不能连 browser 一起关），而
// RegistryMap.Source 是装配来源（CAD/browser/cost 同属 builtin-domain）——
// 本文件是独立的功能组单源：组→成员名单 + name→组反查 + 三守卫（组成员必须
// 在 RegistryMap 在册、一个可配置工具只属一组、未知组 fail-closed 由归一层
// 兜底）。不逐行改 RegistryEntry 五字段结构（Codex r3 终案口径）。
//
// 组语义（与 [office.tool_groups] 覆盖、tool_face 双轴配套）：
//   cad/schedule   两档默认关（进度/CAD 纵深按需开启）
//   browser        lean 关 / standard 开（环境面 12 件）
//   com            lean 关 / standard 跟随本机（officeComAvailable 注册闸，
//                  未装 Office/WPS 的机器该组成员根本不注册）
//   computer       不参与 Hide——注册闸 [computer_use].enabled（关=不注册）
//   cost/longdoc   按 run_mode（standard 档藏、cost/proposal 档各开自己族）
//   media          两档默认开（用户签注：产图不改默认）
//   其余=core      不可配置（文件/命令/办公交付/记忆/技能面等通用底座）

// ToolGroupName 是功能组标识（配置键/面板组开关/ModeLine 拼接共用）。
type ToolGroupName string

const (
	ToolGroupCAD      ToolGroupName = "cad"
	ToolGroupSchedule ToolGroupName = "schedule"
	ToolGroupBrowser  ToolGroupName = "browser"
	ToolGroupCOM      ToolGroupName = "com"
	ToolGroupComputer ToolGroupName = "computer"
	ToolGroupCost     ToolGroupName = "cost"
	ToolGroupLongdoc  ToolGroupName = "longdoc"
	ToolGroupMedia    ToolGroupName = "media"
)

// toolGroupMembers 功能组→成员名单（单源）。computer 组仅作面板展示与文档
// 归类：它由 [computer_use].enabled 注册闸控制，Hide 装配器必须跳过。
var toolGroupMembers = map[ToolGroupName][]string{
	ToolGroupCAD:      {"cad_ask"},
	ToolGroupSchedule: {"schedule_get", "schedule_apply", "schedule_analyze"},
	ToolGroupBrowser: {
		"browser_click", "browser_close", "browser_navigate", "browser_new_tab",
		"browser_press", "browser_read", "browser_run_js", "browser_scroll",
		"browser_snapshot", "browser_switch_tab", "browser_tabs", "browser_type",
	},
	ToolGroupCOM: {
		"office_open", "office_status", "office_read", "office_edit", "office_render",
		"office_save", "office_close", "office_help", "office_macro", "office_selection",
	},
	ToolGroupComputer: {"computer_windows", "computer_state", "computer_launch", "computer_click", "computer_type", "computer_key"},
	ToolGroupCost: {
		"cost_search", "cost_save", "cost_compose", "cost_indicators",
		"cost_project", "cost_project_apply", "cost_audit",
		"cost_library", // 造价库维护（2026-11-25 造价数据库推倒重建：导入/解析/维护归造价模式）
	},
	ToolGroupLongdoc: {"longdoc_manifest", "longdoc_build", "longdoc_verify"},
	ToolGroupMedia:   {"image_gen", "chart_gen", "diagram", "diagram_gen"},
}

// ToolGroups 返回可配置组清单（稳定序：配置渲染/面板展示按此序）。
func ToolGroups() []ToolGroupName {
	return []ToolGroupName{
		ToolGroupCAD, ToolGroupSchedule, ToolGroupBrowser, ToolGroupCOM,
		ToolGroupComputer, ToolGroupCost, ToolGroupLongdoc, ToolGroupMedia,
	}
}

// ToolGroupMembers 返回组成员快照（拷贝，防调用方改单源）。
func ToolGroupMembers(g ToolGroupName) []string {
	out := append([]string(nil), toolGroupMembers[g]...)
	return out
}

// toolGroupOf name→组反查（启动时一次构建；构建即跑单组归属守卫）。
var toolGroupOf = buildToolGroupOf()

func buildToolGroupOf() map[string]ToolGroupName {
	m := make(map[string]ToolGroupName, 48)
	for g, members := range toolGroupMembers {
		for _, name := range members {
			if prev, dup := m[name]; dup {
				panic("toolgroups: 工具跨组重复归属 " + name + " (" + string(prev) + "/" + string(g) + ")")
			}
			m[name] = g
		}
	}
	return m
}

// ToolGroupOf 返回工具所属功能组；""=core（非组员，不可配置、恒可见）。
func ToolGroupOf(name string) ToolGroupName {
	return toolGroupOf[name]
}

// ValidToolGroup 报告组 id 是否在册（配置覆盖归一的 fail-closed 判据：
// 未知组丢弃并按无效值处理，不静默当成组名 Hide）。
func ValidToolGroup(g ToolGroupName) bool {
	_, ok := toolGroupMembers[g]
	return ok
}
