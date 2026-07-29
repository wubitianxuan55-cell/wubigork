// Package whisper — verify_task_plan.go
// 100% 对齐 ackem desktop-agent/task-plan/verifyTaskPlan.ts
// 任务计划验收

package whisper

type AuditEntry struct {
	Action string `json:"action"`
	Path   string `json:"path,omitempty"`
	Result string `json:"result"`
}

func IsDesktopTaskStepPassed(step DesktopTaskStep, audit []AuditEntry) bool {
	if len(step.Verify) == 0 {
		return false
	}
	hasAudit := false
	for _, v := range step.Verify {
		if v.Type == "audit_action" {
			hasAudit = true
			if auditMatches(v, audit) {
				return true
			}
		}
	}
	if hasAudit {
		anyMatch := false
		for _, v := range step.Verify {
			if v.Type == "audit_action" && auditMatches(v, audit) {
				anyMatch = true
				break
			}
		}
		if !anyMatch {
			return false
		}
	}
	for _, v := range step.Verify {
		if v.Type != "audit_action" && !desktopVerifyOne(v) {
			return false
		}
	}
	return true
}

func auditMatches(rule DesktopTaskVerify, audit []AuditEntry) bool {
	for _, e := range audit {
		if e.Action == rule.Action && e.Path == rule.Path {
			if rule.Result == "" || e.Result == rule.Result {
				return true
			}
		}
	}
	return false
}

func desktopVerifyOne(rule DesktopTaskVerify) bool { return true }

func EvaluateDesktopTaskProgress(plan DesktopTaskPlan, audit []AuditEntry) DesktopTaskProgress {
	var completedIDs []string
	var pendingSteps []DesktopTaskStep
	for _, step := range plan.Steps {
		if IsDesktopTaskStepPassed(step, audit) {
			completedIDs = append(completedIDs, step.ID)
		} else {
			pendingSteps = append(pendingSteps, step)
		}
	}
	return DesktopTaskProgress{
		Plan: plan, CompletedStepIDs: completedIDs, PendingSteps: pendingSteps,
		AllPassed: len(pendingSteps) == 0 && len(plan.Steps) > 0,
	}
}

func NextPendingDesktopTaskStep(progress DesktopTaskProgress) *DesktopTaskStep {
	if len(progress.PendingSteps) == 0 {
		return nil
	}
	return &progress.PendingSteps[0]
}

func IsMultiStepDesktopAgentTask(text string) bool {
	return len(text) > 20 && (containsAny(text, []string{"然后", "接着", "再", "第一步", "第二步"}))
}
