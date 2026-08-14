package app

import (
	"log/slog"
	"sync"

	"github.com/gaea/gaea/internal/whisper"
)

// ─── 阶段 5 T5-4b：轻语任务计划（持久化 + 恢复入口）────────────
//
// 装配约束：app.go 的 Startup/Shutdown 由其他子代理独占，本文件不依赖 App 生命周期，
// 采用进程级惰性装配——首次访问时按数据根创建任务计划存储并从磁盘恢复
// （whisper_data/task_plan.json，原子写），之后常驻，达到「重启后加载回内存」的效果。
var (
	taskPlanStoreMu    sync.Mutex
	taskPlanStoreRoot  string
	taskPlanStoreValue *whisper.TaskPlanStore
)

// taskPlanStore 返回轻语任务计划存储：惰性创建 + 磁盘恢复（数据根变化时重建）。
func taskPlanStore(dataRoot string) *whisper.TaskPlanStore {
	taskPlanStoreMu.Lock()
	defer taskPlanStoreMu.Unlock()
	if taskPlanStoreValue == nil || taskPlanStoreRoot != dataRoot {
		store := whisper.NewTaskPlanStoreWithDataRoot(dataRoot)
		_ = store.ReloadFromDisk() // 重启恢复：读取上次未完成的桌面任务计划
		taskPlanStoreValue = store
		taskPlanStoreRoot = dataRoot
	}
	return taskPlanStoreValue
}

// WhisperTaskPlanStatus 轻语任务计划状态快照（前端「任务计划」面板轮询）。
// 无活动计划返回 {active:false}；有则返回计划概览与步骤状态（步骤已完成按进度标记）。
func (a *whisperState) WhisperTaskPlanStatus() map[string]interface{} {
	plan := taskPlanStore(a.whisperDataRoot).ActivePlan()
	if plan == nil {
		return map[string]interface{}{"active": false}
	}
	done := make(map[string]bool, len(plan.CompletedStepIDs))
	for _, id := range plan.CompletedStepIDs {
		done[id] = true
	}
	steps := make([]map[string]interface{}, 0, len(plan.Plan.Steps))
	for _, st := range plan.Plan.Steps {
		status := st.Status
		if done[st.ID] {
			status = "completed"
		}
		steps = append(steps, map[string]interface{}{
			"id": st.ID, "label": st.Label, "status": status,
		})
	}
	return map[string]interface{}{
		"active":      true,
		"id":          plan.Plan.ID,
		"goalSummary": plan.Plan.GoalSummary,
		"steps":       steps,
		"updatedAt":   plan.UpdatedAt,
	}
}

// WhisperTaskPlanResume 任务计划恢复入口：存在未完成任务计划时，把计划重新置为
// 进行中（刷新 updatedAt 并落盘，供后续继续执行续接），返回是否成功恢复。
// 说明：当前执行流程尚无 orchestrator 级继续机制接线到计划存储，此处按约定
// 至少完成「标记 active + 落盘」；待桌面 agent 循环接入 Save 后即为续跑入口。
func (a *whisperState) WhisperTaskPlanResume() bool {
	store := taskPlanStore(a.whisperDataRoot)
	plan := store.ActivePlan()
	if plan == nil {
		return false
	}
	ok := store.Resume(plan.SessionID)
	if ok {
		slog.Info("[whisper] 任务计划已恢复", "sessionID", plan.SessionID, "planID", plan.Plan.ID)
	}
	return ok
}
