package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/prompt"
)

// TestBranchPositionContext 四位置分类与教义注入：开篇（章1/无前文）、阶段收官
// （章号 % 20 == 0）、新阶段开篇（21/41/…）、常规推进。
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
	got := branchPositionContext(20, "前文")
	if !strings.Contains(got, "阶段收官") || !strings.Contains(got, "第1~20章") {
		t.Errorf("第20章应为阶段收官（1~20），got:\n%s", got)
	}
	if got := branchPositionContext(21, "前文"); !strings.Contains(got, "新阶段开篇") {
		t.Errorf("第21章应为新阶段开篇教义，got:\n%s", got)
	}
	if got := branchPositionContext(45, "前文"); !strings.Contains(got, "常规推进") {
		t.Errorf("第45章应为常规教义，got:\n%s", got)
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
