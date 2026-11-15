package app

// wenxue_skills_test.go — 图文技能面单测（2026-11-15 刀2）。

import "testing"

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
