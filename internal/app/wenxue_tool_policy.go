package app

// wenxue_tool_policy.go — 图文板块（wenxue）会话工具策略（2026-11-15 刀1，案A）。
//
// 图文=长篇小说创作专属板块：办公域工具（office 全族/CAD/造价/进度计划/
// 浏览器自动化/图表与文档渲染等）在图文引擎装配期物理不装。名单与前端
// frontend/src/wenxue/lib/disabledDomains.ts 的 DISABLED_TOOL_NAMES 同源对齐
// （前端是兜底展示面，这里是引擎侧唯一事实面）；改名单两处同批。
//
// 语义：精确名整名匹配；带 "_" 结尾的条目按前缀匹配（browser_ 族）。
// 过滤发生在 boot.Build 装配期（Options.ToolFilter 缝）。2026-11-16 图文瘦身
// 刀：ToolFilter 语义从「addBuiltins/plugins/ExtraTools 三处」扩为「全部模型
// 面注册点」（boot 直注册的会话级元工具/大文件件/子代理管理件同样过闸，见
// boot.addGated）——被摘的 13 件见名单末段「图文瘦身刀」。物理摘除而非运行时
// 拦截；办公引擎不传 ToolFilter，逐字节现状（办公零影响红线）。

// wenxueDeniedTools 图文引擎不装配的办公域工具（family 级）。
var wenxueDeniedTools = []string{
	// office 生成/批处理
	"office_generate", "office_batch",
	// office 文档真编辑
	"docx_apply", "docx_accept", "pptx_apply",
	// xlsx 编辑族
	"xlsx_apply", "xlsx_profile", "xlsx_recalc", "xlsx_audit",
	// office COM 档（本机有 Office/WPS 才注册的那批）
	"office_open", "office_status", "office_read", "office_edit", "office_render",
	"office_save", "office_close", "office_help", "office_macro", "office_selection",
	// CAD / 造价
	"cad_ask",
	"cost_search", "cost_save", "cost_compose", "cost_indicators",
	// 跨库语义检索（specialist 面）
	"semantic_search",
	// 方案库三件（方案模式专属）
	"proposal_save", "proposal_search", "proposal_read",
	// 进度计划三件（shared 装配，图文不装）
	"schedule_get", "schedule_apply", "schedule_analyze",
	// 图表/图面/文档渲染/格式转换/截屏（小说创作无消费面）
	"chart_gen", "diagram_gen", "diagram", "doc_render",
	"format_convert", "screen_capture",
	// 办公文件流水线
	"dag_plan",
	// 交付事实库（2026-11-15 众神殿首刀除名，pan_1794754879290 裁决）：语义=
	// 办公交付任务的"确认数字/单位/期限"收集（PersistWrite），与图文写作闭环
	// 无关——回写纪律走 时间线/角色卡（wenxue-workbench）。办公实现与
	// RegistryMap 契约行不动，仅图文面不装配。
	"fact_add", "fact_list", "fact_clear",
	// 浏览器自动化族（前缀匹配）
	"browser_",

	// ── 图文瘦身刀（2026-11-16）：写作闭环外的面板杂音，13 件 ──
	// 编辑族五件：章节/角色卡修订走 chapter/character_write 整卡覆写，
	// 精确行级/多点编辑是开发域手感（read+write 覆盖兜底）。
	"edit_file", "edit_lines", "multi_edit", "replace", "move_file",
	// 大文件两件：章节体量远低于落盘/分块摘要阈值，read_file 分页已覆盖。
	"read_spill", "summarize_file",
	// 办公侧栏 UI 面：图文无消费。
	"sidebar_open",
	// 会话元工具：图文交互审批链已覆盖权限面，无计划审批面板流程。
	"request_permission", "exit_plan_mode",
	// 子代理管理三件：管理面在会话 UI tab；agent 侧保留 task 派发。
	"interrupt_agent", "subagent_list", "read_subagent_result",
}

// wenxueToolFilter 报告工具是否允许装配进图文引擎（boot.Options.ToolFilter 缝）。
func wenxueToolFilter(name string) bool {
	for _, d := range wenxueDeniedTools {
		if d != "" && d[len(d)-1] == '_' {
			if len(name) >= len(d) && name[:len(d)] == d {
				return false
			}
			continue
		}
		if name == d {
			return false
		}
	}
	return true
}
