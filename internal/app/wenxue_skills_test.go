package app

// wenxue_skills_test.go — 图文技能面单测（2026-11-15 刀2）。

import (
	"strings"
	"testing"
)

func TestWenxueSkillFilterDeniesOfficeSkills(t *testing.T) {
	for _, n := range []string{
		"gongwen-9704", "office-gongwen", "cad-takeoff", "office-edit",
		"schedule-edit", "schedule-dcma", "cost-compose", "format-convert",
		"chart-builder", "doc-assemble",
		// 办公文档技能：索引隐藏（导出链走路径找脚本，不受影响）
		"docx", "pptx", "pdf", "xlsx", "pptx-template",
	} {
		if wenxueSkillFilter(n) {
			t.Fatalf("办公域技能 %s 应对图文引擎隐藏", n)
		}
	}
}

func TestWenxueSkillFilterAllowsNovelDomainAndMeta(t *testing.T) {
	for _, n := range []string{
		"novel-agents", "novel-deslop", "novel-review",
		"story-long-write", "story-deslop", "wenxue-workbench",
		"genui", "skill-creator",
	} {
		if !wenxueSkillFilter(n) {
			t.Fatalf("小说域/元技能 %s 应对图文引擎可见", n)
		}
	}
}

func TestWenxueSkillExtrasSelfPassFilter(t *testing.T) {
	// 内嵌三件必须自证通过 filter（extras 先 append 后 filter 的装配序约束）。
	for _, sk := range wenxueSkillExtras() {
		if !wenxueSkillFilter(sk.Name) {
			t.Fatalf("内嵌技能 %s 未通过自身 allowlist（装配序会把它滤掉）", sk.Name)
		}
		if sk.Body == "" || sk.Description == "" {
			t.Fatalf("内嵌技能 %s 缺正文或描述", sk.Name)
		}
	}
}

// TestWenxueWorkbenchV1Body workbench v1.0 三行+自称放宽（2026-11-18 众神殿
// pan_1795007052552 v1.0 冻结包）：自称从「长篇小说创作专属板块」放宽为
// 「小说创作工作台」；过程文件行（.gaea/work/<任务名>/ work 锚——路径单源
// 归 workbench+工具链）；导出句中立不写固定根；记忆确认制。固定导出根字面
// 不得进 workbench（文学导出走 ExportsDir 按空间落点）。
func TestWenxueWorkbenchV1Body(t *testing.T) {
	for _, sub := range []string{
		"小说创作工作台",
		".gaea/work/<任务名>/",
		"导出件以执行提示词或导出工具给出的默认落点为准",
		"才调用 remember",
	} {
		if !strings.Contains(wenxueWorkbenchBody, sub) {
			t.Errorf("workbench 正文缺 v1.0 锚 %q", sub)
		}
	}
	for _, banned := range []string{
		"长篇小说创作专属板块", // 旧自称（范围已放宽到各类小说）
		".gaea/exports/",  // 固定导出根不得进文学侧文本
	} {
		if strings.Contains(wenxueWorkbenchBody, banned) {
			t.Errorf("workbench 正文含禁词 %q", banned)
		}
	}
}

// TestStoryLongWriteGating story-long-write 网文门控自述（2026-11-18 众神殿
// 余项）：板块范围已放宽到各类小说，网文工艺件须在索引描述与正文头双处
// 声明适用域——非网文书籍不套用黄金三章/爽点密度（与文学身份稿同款约束）。
func TestStoryLongWriteGating(t *testing.T) {
	for _, sk := range wenxueSkillExtras() {
		if sk.Name != "story-long-write" {
			continue
		}
		for _, sub := range []string{"网文题材专用", "未声明网文题材时不套用"} {
			if !strings.Contains(sk.Description, sub) {
				t.Errorf("story-long-write 描述缺门控锚 %q（got %q）", sub, sk.Description)
			}
		}
		for _, sub := range []string{"适用域：网文题材", "不套用本章指标"} {
			if !strings.Contains(sk.Body, sub) {
				t.Errorf("story-long-write 正文缺适用域锚 %q", sub)
			}
		}
		return
	}
	t.Fatal("内嵌技能面缺 story-long-write")
}
