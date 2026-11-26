package app

// wenxue_tool_policy_test.go — 图文工具策略单测（2026-11-15 刀1）。
// 钉死三件事：办公域 deny 名单全被拒、浏览器前缀族整族被拒、通用底座/元工具
// 与图文专属前缀全放行。名单与 frontend/src/wenxue/lib/disabledDomains.ts
// DISABLED_TOOL_NAMES 同源（改名单两处同批，本测试兜底引擎侧）。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// TestWenxueToolFilterDeniesRunModeGroups 造价 / 长文方案两组整组不装
// （2026-11-26 工具面实测：两组里 6 件此前混进图文模型实见面）。按组反查，
// 组成员新增自动跟随。
func TestWenxueToolFilterDeniesRunModeGroups(t *testing.T) {
	for _, g := range []tool.ToolGroupName{tool.ToolGroupCost, tool.ToolGroupLongdoc} {
		members := tool.ToolGroupMembers(g)
		if len(members) == 0 {
			t.Fatalf("工具组 %s 无成员（单源表被清空？）", g)
		}
		for _, n := range members {
			if wenxueToolFilter(n) {
				t.Fatalf("运行模式专属组 %s 的成员 %s 应被图文引擎拒绝装配", g, n)
			}
		}
	}
}

func TestWenxueToolFilterDeniesOfficeFamily(t *testing.T) {
	denied := []string{
		// office 生成/批处理/COM 档
		"office_generate", "office_batch", "office_open",
		"office_read", "office_edit", "office_render", "office_save",
		"office_close", "office_selection",
		// 文档真编辑
		"docx_apply", "pptx_apply",
		// xlsx 族
		"xlsx_apply", "xlsx_profile", "xlsx_recalc", "xlsx_audit",
		// 域工具
		"cad_ask", "cost_search", "cost_save", "cost_indicators",
		"semantic_search",
		"schedule_get", "schedule_apply", "schedule_analyze",
		"chart_gen", "diagram_gen", "diagram", "doc_render",
		"format_convert", "screen_capture",
	}
	for _, n := range denied {
		if wenxueToolFilter(n) {
			t.Fatalf("办公域工具 %s 应被图文引擎拒绝装配", n)
		}
	}
}

func TestWenxueToolFilterDeniesFactFamily(t *testing.T) {
	// 2026-11-15 众神殿首刀除名（pan_1794754879290）：交付事实库是办公语义，
	// 图文回写纪律走 时间线/角色卡，不装 fact_*。办公实现不动。
	for _, n := range []string{"fact_add", "fact_list", "fact_clear"} {
		if wenxueToolFilter(n) {
			t.Fatalf("交付事实库工具 %s 应被图文引擎拒绝装配", n)
		}
	}
}

func TestWenxueToolFilterDeniesBrowserPrefixFamily(t *testing.T) {
	for _, n := range []string{
		"browser_navigate", "browser_new_tab", "browser_click", "browser_type",
		"browser_press", "browser_scroll", "browser_read", "browser_snapshot",
		"browser_tabs", "browser_switch_tab", "browser_run_js", "browser_close",
	} {
		if wenxueToolFilter(n) {
			t.Fatalf("浏览器自动化工具 %s 应被前缀规则拒绝装配", n)
		}
	}
}

func TestWenxueToolFilterAllowsGenericBase(t *testing.T) {
	allowed := []string{
		// 通用底座（图文会话保留面）
		"read_file", "write_file", "grep", "ls",
		"bash", "bash_output", "kill_shell", "wait",
		"web_fetch", "web_search", "vision", "ocr", "image_gen",
		"translate_text", "routine_llm",
		// 会话级元工具保留面（2026-11-16 瘦身刀起过闸，名单内摘、保留面在此）
		"task", "ask", "todo_write", "complete_step",
		"remember", "forget", "promote_session_facts", "memory_get", "memory_search",
		"run_skill", "read_skill", "install_skill",
		"session_search", "recall_archived", "genui_validate",
	}
	for _, n := range allowed {
		if !wenxueToolFilter(n) {
			t.Fatalf("通用工具 %s 不应被图文引擎拒绝装配", n)
		}
	}
}

func TestWenxueToolFilterDeniesSlimmedFamily(t *testing.T) {
	// 2026-11-16 图文瘦身刀 13 件：编辑族五件/大文件两件/办公侧栏/会话元工具
	// 两件/子代理管理三件（管理面在 UI tab，agent 侧保留 task）。写作闭环外
	// 的面板杂音，物理不装——boot 直注册件经 boot.addGated 同闸生效。
	// （编辑族的 edit_lines/multi_edit/replace 与 cost_compose 已于 2026-11-25
	// 全局退役，名单与用例同步收掉。）
	for _, n := range []string{
		"edit_file", "move_file",
		"read_spill", "summarize_file", "sidebar_open",
		"request_permission", "exit_plan_mode",
		"interrupt_agent", "subagent_list", "read_subagent_result",
	} {
		if wenxueToolFilter(n) {
			t.Fatalf("瘦身刀工具 %s 应被图文引擎拒绝装配", n)
		}
	}
}
