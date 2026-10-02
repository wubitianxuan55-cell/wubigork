package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/prompt"
)

// TestBranchPositionContext 位置分类与教义注入（v4.452.0：每10章一阶段，阶段内
// 起承转合）：开篇（章1/无前文）、新阶段开篇（11/21/…）、起（阶段内2-3）、
// 承（4-6）、转（7-9）、阶段收官（10/20/…）、常规推进（章号未知兜底）。
func TestBranchPositionContext(t *testing.T) {
	if got := branchPositionContext(1, ""); !strings.Contains(got, "开篇") || !strings.Contains(got, "哪扇门") {
		t.Errorf("第1章应为开篇教义，got:\n%s", got)
	}
	if got := branchPositionContext(0, ""); !strings.Contains(got, "开篇") {
		t.Errorf("无章号无前文应为开篇教义，got:\n%s", got)
	}
	if got := branchPositionContext(0, "前文摘要"); !strings.Contains(got, "常规推进") {
		t.Errorf("无章号有前文应为常规教义，got:\n%s", got)
	}
	got := branchPositionContext(10, "前文")
	if !strings.Contains(got, "阶段收官") || !strings.Contains(got, "第1~10章") {
		t.Errorf("第10章应为阶段收官（合，1~10），got:\n%s", got)
	}
	if got := branchPositionContext(20, "前文"); !strings.Contains(got, "阶段收官") || !strings.Contains(got, "第11~20章") {
		t.Errorf("第20章应为阶段收官（11~20），got:\n%s", got)
	}
	if got := branchPositionContext(11, "前文"); !strings.Contains(got, "新阶段开篇") {
		t.Errorf("第11章应为新阶段开篇教义，got:\n%s", got)
	}
	// 阶段内起承转合
	if got := branchPositionContext(3, "前文"); !strings.Contains(got, "阶段之「起」") || !strings.Contains(got, "第1~10章") {
		t.Errorf("第3章应为起教义，got:\n%s", got)
	}
	if got := branchPositionContext(5, "前文"); !strings.Contains(got, "阶段之「承」") {
		t.Errorf("第5章应为承教义，got:\n%s", got)
	}
	if got := branchPositionContext(8, "前文"); !strings.Contains(got, "阶段之「转」") {
		t.Errorf("第8章应为转教义，got:\n%s", got)
	}
	// 跨阶段：第14章=第2阶段（11~20）内第4章=承
	if got := branchPositionContext(14, "前文"); !strings.Contains(got, "阶段之「承」") || !strings.Contains(got, "第11~20章") {
		t.Errorf("第14章应为第2阶段承教义，got:\n%s", got)
	}
	// 第45章=第5阶段（41~50）内第5章=承
	if got := branchPositionContext(45, "前文"); !strings.Contains(got, "阶段之「承」") || !strings.Contains(got, "第41~50章") {
		t.Errorf("第45章应为第5阶段承教义，got:\n%s", got)
	}
}

// TestStagePhaseOf 阶段模型边界：每10章一阶段内的起承转合映射。
func TestStagePhaseOf(t *testing.T) {
	cases := map[int]stagePhase{
		0: phaseUnknown, -3: phaseUnknown,
		1: phaseOpening,
		2: phaseQi, 3: phaseQi,
		4: phaseCheng, 5: phaseCheng, 6: phaseCheng,
		7: phaseZhuan, 8: phaseZhuan, 9: phaseZhuan,
		10: phaseHe,
		11: phaseStageStart,
		12: phaseQi, 16: phaseCheng, 19: phaseZhuan, 20: phaseHe,
	}
	for num, want := range cases {
		if _, _, got := stagePhaseOf(num); got != want {
			t.Errorf("stagePhaseOf(%d) phase = %v, want %v", num, got, want)
		}
	}
	if s, p, _ := stagePhaseOf(45); s != 5 || p != 5 {
		t.Errorf("stagePhaseOf(45) = (%d,%d), want (5,5)", s, p)
	}
	if s, e := stageBounds(5); s != 41 || e != 50 {
		t.Errorf("stageBounds(5) = (%d,%d), want (41,50)", s, e)
	}
}

// TestPlotBranchTemplateRendersStoryPosition 模板联通：plot-branch-browser 有
// story_position 槽位且注入内容渲染进 user prompt（空内容不渲染）。
func TestPlotBranchTemplateRendersStoryPosition(t *testing.T) {
	eng := prompt.NewEngine("../../prompts")
	tmpl := eng.Get("plot-branch-browser")
	if tmpl == nil {
		t.Fatal("缺少 plot-branch-browser 模板")
	}
	if _, ok := tmpl.Inputs["story_position"]; !ok {
		t.Fatal("plot-branch-browser 缺 story_position 槽位")
	}

	rendered := tmpl.BuildUserPrompt(map[string]string{
		"story_position": "【本章位置与分支任务：开篇】三分支=三扇门",
	})
	if !strings.Contains(rendered, "三分支=三扇门") {
		t.Errorf("story_position 未渲染:\n%s", rendered)
	}

	bare := tmpl.BuildUserPrompt(map[string]string{})
	if strings.Contains(bare, "本章位置与分支任务") {
		t.Errorf("空注入不应出现位置标题:\n%s", bare)
	}
}

// TestPlanPositionContext 章节计划的位置注入（分支同款阶段模型）：各段任务
// 随位置切换，章号未知返回空串（模板空槽位零渲染）。
func TestPlanPositionContext(t *testing.T) {
	if got := planPositionContext(1); !strings.Contains(got, "全书开篇") || !strings.Contains(got, "引爆点") {
		t.Errorf("第1章计划应为全书开篇任务，got:\n%s", got)
	}
	if got := planPositionContext(3); !strings.Contains(got, "阶段之「起」") || !strings.Contains(got, "不透支后段") {
		t.Errorf("第3章计划应为起任务，got:\n%s", got)
	}
	if got := planPositionContext(5); !strings.Contains(got, "阶段之「承」") {
		t.Errorf("第5章计划应为承任务，got:\n%s", got)
	}
	got := planPositionContext(8)
	if !strings.Contains(got, "阶段之「转」") || !strings.Contains(got, "终极摊牌留给收官章") {
		t.Errorf("第8章计划应为转任务，got:\n%s", got)
	}
	if got := planPositionContext(10); !strings.Contains(got, "阶段收官「合」") || !strings.Contains(got, "第1~10章") {
		t.Errorf("第10章计划应为合任务（1~10），got:\n%s", got)
	}
	if got := planPositionContext(11); !strings.Contains(got, "新阶段开篇") || !strings.Contains(got, "第1~10章") {
		t.Errorf("第11章计划应承接上一阶段（1~10）余波，got:\n%s", got)
	}
	if got := planPositionContext(20); !strings.Contains(got, "阶段收官「合」") || !strings.Contains(got, "第11~20章") {
		t.Errorf("第20章计划应为合任务（11~20），got:\n%s", got)
	}
	if got := planPositionContext(0); got != "" {
		t.Errorf("章号未知应返回空串（零注入），got:\n%s", got)
	}
}

// TestChapterPlanTemplateRendersStoryPosition 模板联通：chapter-plan 有
// story_position 槽位且注入内容渲染进 user prompt（空内容不渲染）。
func TestChapterPlanTemplateRendersStoryPosition(t *testing.T) {
	eng := prompt.NewEngine("../../prompts")
	tmpl := eng.Get("chapter-plan")
	if tmpl == nil {
		t.Fatal("缺少 chapter-plan 模板")
	}
	if _, ok := tmpl.Inputs["story_position"]; !ok {
		t.Fatal("chapter-plan 缺 story_position 槽位")
	}

	rendered := tmpl.BuildUserPrompt(map[string]string{
		"story_position": "【本章阶段位置与计划任务：阶段之「转」】反转与高压",
	})
	if !strings.Contains(rendered, "反转与高压") {
		t.Errorf("story_position 未渲染:\n%s", rendered)
	}

	bare := tmpl.BuildUserPrompt(map[string]string{})
	if strings.Contains(bare, "本章阶段位置与计划任务") {
		t.Errorf("空注入不应出现位置标题:\n%s", bare)
	}
}
