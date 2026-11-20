package agent

// 图文板块执行纪律契约测试（2026-11-18 众神殿 pan_1795007052552 v1.0 冻结包
// 「文学 B」）：独立成文红线（不与办公 SingleModelPrompt 共用正文）+ 路径三句
// 终版（过程 .gaea/work/<任务名>/ 字面量、导出不写固定根、定稿仅
// wenxue_final_write）+ 验收合同禁词（办公域工具名/办公自称不进文学执行串）。

import (
	"strings"
	"testing"
)

// TestWenxueSingleModelPromptContract 文学执行常量契约：在场锚+禁词+独立成文。
func TestWenxueSingleModelPromptContract(t *testing.T) {
	p := WenxueSingleModelPrompt

	// 在场锚（文学 B 清单各节 + DSH r4 2.4 三句路径终版）。
	for _, sub := range []string{
		".gaea/work/<任务名>/",          // 过程文件句（work 锚字面量）
		"wenxue_chapter_write",          // 写章：草稿
		"wenxue_candidate_write",        // 写章：多方案
		"wenxue_final_write",            // 定稿唯一通道
		"wenxue_quality_check",          // 成批章节体检
		"complete_step",                 // 签退
		"主动调用 remember",               // 记忆确认制
		"不承诺「数据不出机」",           // 数据边界（文学允许外部检索与生图）
		"宿主当前空间的默认导出落点",       // 导出句空间中立
	} {
		if !strings.Contains(p, sub) {
			t.Errorf("文学执行常量缺在场锚 %q", sub)
		}
	}

	// 禁词（验收合同⑤：文学最终串不得含办公域工具名与办公自称；导出根字面
	// 不进文学稿——定稿/导出走工具落点，写死根会绕过 wenxue_final_write）。
	for _, banned := range []string{
		".gaea/exports/",
		"fact_add", "fact_list", "fact_clear",
		"format_convert", "semantic_search", "screen_capture",
		"office_", "通用办公助手", "办公规范",
	} {
		if strings.Contains(p, banned) {
			t.Errorf("文学执行常量含禁词 %q（办公域语义/固定导出根不得进文学稿）", banned)
		}
	}
}

// TestWenxueSingleModelPromptIndependentFromOffice 两份执行常量独立成文
// （用户拍板：共享原则不共享文本；运行期零字符串替换）。
func TestWenxueSingleModelPromptIndependentFromOffice(t *testing.T) {
	if WenxueSingleModelPrompt == SingleModelPrompt {
		t.Fatal("文学执行常量不得与办公执行常量共用正文（独立成稿红线）")
	}
	// 办公常量独有锚不出现在文学常量（分叉生效的方向性检查——若两常量
	// 意外合并，办公独有段会整体出现）。
	for _, officeOnly := range []string{
		"事实底座（一稿多用）",
		"办公领域质量检查点",
		"最终交付物",
	} {
		if strings.Contains(WenxueSingleModelPrompt, officeOnly) {
			t.Errorf("文学执行常量含办公独有段锚 %q（应走独立常量分叉）", officeOnly)
		}
	}
}

// TestSingleModelPromptMemoryConsent 记忆句口径（2026-11-21 回调拍板：
// auto-dream 删除后 remember 是唯一模型侧入口，「只有用户明确要求才调用」
// 过紧致模型实际停记——恢复主动调用，确认制由确认卡承载）。
func TestSingleModelPromptMemoryConsent(t *testing.T) {
	if !strings.Contains(SingleModelPrompt, "主动调用 remember") {
		t.Error("办公执行常量缺「主动调用 remember」（口径回调后应为主动制+确认卡）")
	}
	if strings.Contains(SingleModelPrompt, "只在用户明确要求记住") {
		t.Error("办公执行常量仍含旧确认制句「只在用户明确要求记住」")
	}
}

// TestSingleModelPromptCapabilityDiscovery 办公执行常量 v1.0 增量②：检查点
// 段末尾两行能力发现路由（schema budget 截断/deferred 隐藏时的兜底路由——
// 身份层不点名工具后，能力可见性由 schema+这两行共同保证）。
func TestSingleModelPromptCapabilityDiscovery(t *testing.T) {
	for _, sub := range []string{
		"**能力发现**",
		"先看本回合 schema",
		"tool_search",
		"不猜工具名",
	} {
		if !strings.Contains(SingleModelPrompt, sub) {
			t.Errorf("办公执行常量缺能力发现锚 %q", sub)
		}
	}
}
