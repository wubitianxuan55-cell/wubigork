// Package whisper — task_plan_prompt.go
// 100% 对齐 ackem desktop-agent/task-plan/taskPlanPrompt.ts
// 任务计划提示词

package whisper

import (
	"fmt"
	"strings"
)

func BuildTaskPlanSystemHint(plan DesktopTaskPlan) string {
	var lines []string
	for i, s := range plan.Steps {
		pathStr := ""
		if s.Path != "" {
			pathStr = " path=" + s.Path
		}
		lines = append(lines, fmt.Sprintf("%d. %s → use_computer action=%s%s", i+1, s.Label, s.Action, pathStr))
	}
	parts := []string{
		"【电脑助手 · Agent 任务计划（Codex 式闭环）】",
		"目标：" + plan.GoalSummary,
		"工作流：规划 → 逐步 use_computer 执行 → 系统自检验收 → 全部通过后才允许交付。",
		"门禁：验收未通过时禁止输出 persona 总结、禁止声称已完成；必须继续调用 use_computer。",
		"", "步骤清单：",
	}
	parts = append(parts, lines...)
	parts = append(parts, "", "仅当系统验收显示全部通过，最后一轮才可只输出交付总结。")
	return strings.Join(parts, "\n")
}

func BuildTaskPlanContinuationPrompt(progress DesktopTaskProgress) string {
	done := len(progress.CompletedStepIDs)
	total := len(progress.Plan.Steps)
	var pendingLines []string
	for i, s := range progress.PendingSteps {
		pendingLines = append(pendingLines, fmt.Sprintf("%d. [待完成] %s（action=%s）", i+1, s.Label, s.Action))
	}
	parts := []string{fmt.Sprintf("【任务验收 · %d/%d 步已通过】", done, total)}
	if len(pendingLines) > 0 {
		parts = append(parts, "以下步骤尚未通过验收：")
		parts = append(parts, pendingLines...)
	}
	next := ""
	if len(progress.PendingSteps) > 0 {
		s := progress.PendingSteps[0]
		next = fmt.Sprintf("请立即调用 use_computer 完成下一步：%s。不要输出 persona 总结。", s.Label)
	}
	parts = append(parts, "", next)
	return strings.Join(parts, "\n")
}

func BuildTaskPlanFollowUpHonestyBlock(progress DesktopTaskProgress) string {
	var done []string
	for _, s := range progress.Plan.Steps {
		for _, id := range progress.CompletedStepIDs {
			if s.ID == id {
				done = append(done, s.Label)
				break
			}
		}
	}
	parts := []string{"【验收已通过】以下步骤均已执行并验收："}
	for _, l := range done {
		parts = append(parts, "- "+l)
	}
	parts = append(parts, "请基于真实完成的操作向用户交付一条完整回复。")
	return strings.Join(parts, "\n")
}

func BuildTaskPlanIncompleteDelivery(progress DesktopTaskProgress) string {
	var doneLabels, pendingLabels []string
	for _, s := range progress.Plan.Steps {
		found := false
		for _, id := range progress.CompletedStepIDs {
			if s.ID == id {
				found = true
				break
			}
		}
		if found {
			doneLabels = append(doneLabels, "- 已完成："+s.Label)
		} else {
			pendingLabels = append(pendingLabels, "- 未完成："+s.Label)
		}
	}
	parts := []string{"多步骤任务尚未全部完成。"}
	if len(doneLabels) > 0 {
		parts = append(parts, strings.Join(doneLabels, "\n"))
	}
	parts = append(parts, strings.Join(pendingLabels, "\n"))
	return strings.Join(parts, "\n")
}
