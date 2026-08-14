package app

import (
	"testing"

	"github.com/gaea/gaea/internal/whisper"
)

// ─── 阶段 5 T5-4b：轻语任务计划绑定测试 ───────────────────────

// appSampleTaskPlan 构造一个未完成的任务计划（第 1 步已完成、第 2 步待执行）
func appSampleTaskPlan() (whisper.DesktopTaskPlan, whisper.DesktopTaskProgress) {
	plan := whisper.DesktopTaskPlan{
		ID:          "plan-app-1",
		GoalSummary: "整理下载目录",
		CreatedAt:   "2026-08-14T10:00:00+08:00",
		Steps: []whisper.DesktopTaskStep{
			{ID: "list", Label: "列出下载目录", Action: "list_folder", Status: "pending"},
			{ID: "cleanup", Label: "删除临时文件", Action: "delete_path", Status: "pending"},
		},
	}
	return plan, whisper.DesktopTaskProgress{
		Plan: plan, CompletedStepIDs: []string{"list"},
		PendingSteps: []whisper.DesktopTaskStep{plan.Steps[1]}, AllPassed: false,
	}
}

// installTaskPlanStore 测试辅助：向进程级存储装配点注入指定数据根的隔离存储
// （并立即从磁盘恢复，模拟重启装配），返回该存储供直接操作。
func installTaskPlanStore(root string) *whisper.TaskPlanStore {
	taskPlanStoreMu.Lock()
	defer taskPlanStoreMu.Unlock()
	store := whisper.NewTaskPlanStoreWithDataRoot(root)
	_ = store.ReloadFromDisk()
	taskPlanStoreValue = store
	taskPlanStoreRoot = root
	return store
}

// TestWhisperTaskPlanStatus 有活动计划时返回完整状态快照
func TestWhisperTaskPlanStatus(t *testing.T) {
	root := t.TempDir()
	store := installTaskPlanStore(root)
	plan, progress := appSampleTaskPlan()
	store.Save("sess-1", plan, progress)
	a := &whisperState{whisperDataRoot: root}

	got := a.WhisperTaskPlanStatus()
	if active, _ := got["active"].(bool); !active {
		t.Fatalf("应返回 active=true, got %v", got)
	}
	if id, _ := got["id"].(string); id != "plan-app-1" {
		t.Errorf("id 应为 plan-app-1, got %q", id)
	}
	if goal, _ := got["goalSummary"].(string); goal != "整理下载目录" {
		t.Errorf("goalSummary 不符: %q", goal)
	}
	if updatedAt, _ := got["updatedAt"].(string); updatedAt == "" {
		t.Error("updatedAt 不应为空")
	}
	steps, ok := got["steps"].([]map[string]interface{})
	if !ok || len(steps) != 2 {
		t.Fatalf("steps 应为 2 项, got %#v", got["steps"])
	}
	if steps[0]["status"] != "completed" {
		t.Errorf("已完成步骤状态应为 completed, got %v", steps[0]["status"])
	}
	if steps[1]["status"] != "pending" {
		t.Errorf("待执行步骤状态应为 pending, got %v", steps[1]["status"])
	}
}

// TestWhisperTaskPlanStatusNoPlan 无活动计划返回 active:false
func TestWhisperTaskPlanStatusNoPlan(t *testing.T) {
	root := t.TempDir()
	installTaskPlanStore(root) // 空存储
	a := &whisperState{whisperDataRoot: root}
	got := a.WhisperTaskPlanStatus()
	if active, _ := got["active"].(bool); active {
		t.Fatalf("空存储应返回 active=false, got %v", got)
	}
}

// TestWhisperTaskPlanLazyStore 未装配时绑定首次调用惰性创建存储（不 panic，自动磁盘恢复）
func TestWhisperTaskPlanLazyStore(t *testing.T) {
	root := t.TempDir()
	a := &whisperState{whisperDataRoot: root} // 未 install
	got := a.WhisperTaskPlanStatus()
	if active, _ := got["active"].(bool); active {
		t.Fatalf("惰性空存储应返回 active=false, got %v", got)
	}
	taskPlanStoreMu.Lock()
	defer taskPlanStoreMu.Unlock()
	if taskPlanStoreValue == nil || taskPlanStoreRoot != root {
		t.Fatal("惰性装配后存储应已创建且绑定该数据根")
	}
}

// TestWhisperTaskPlanResume 恢复入口：有活动计划返回 true，无则 false
func TestWhisperTaskPlanResume(t *testing.T) {
	root := t.TempDir()
	store := installTaskPlanStore(root)
	plan, progress := appSampleTaskPlan()
	store.Save("sess-1", plan, progress)
	a := &whisperState{whisperDataRoot: root}
	if !a.WhisperTaskPlanResume() {
		t.Fatal("存在活动计划时 WhisperTaskPlanResume 应返回 true")
	}

	emptyRoot := t.TempDir()
	installTaskPlanStore(emptyRoot)
	empty := &whisperState{whisperDataRoot: emptyRoot}
	if empty.WhisperTaskPlanResume() {
		t.Fatal("无活动计划时 WhisperTaskPlanResume 应返回 false")
	}
}

// TestWhisperTaskPlanRestartResume 模拟重启：磁盘计划重载后恢复入口可用
func TestWhisperTaskPlanRestartResume(t *testing.T) {
	root := t.TempDir()
	// 第一次运行：保存活动计划后进程退出（磁盘留存）
	first := installTaskPlanStore(root)
	plan, progress := appSampleTaskPlan()
	first.Save("sess-1", plan, progress)

	// 重启：重新装配（全新存储）并从磁盘恢复
	installTaskPlanStore(root)
	second := &whisperState{whisperDataRoot: root}
	st := second.WhisperTaskPlanStatus()
	if active, _ := st["active"].(bool); !active {
		t.Fatal("重启后应有活动计划")
	}
	if !second.WhisperTaskPlanResume() {
		t.Fatal("重启后恢复入口应返回 true")
	}
	if got := second.WhisperTaskPlanStatus(); got["id"] != "plan-app-1" {
		t.Errorf("恢复后计划 id 不符: %v", got["id"])
	}
}
