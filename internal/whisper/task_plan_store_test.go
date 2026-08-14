package whisper

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── 阶段 5 T5-4b：任务计划持久化测试 ─────────────────────────

// sampleTaskPlan 构造一个未完成的任务计划（第 1 步已完成、第 2 步待执行）
func sampleTaskPlan() (DesktopTaskPlan, DesktopTaskProgress) {
	plan := DesktopTaskPlan{
		ID:          "plan-sample-1",
		GoalSummary: "在桌面创建项目文件夹并写入 README",
		CreatedAt:   "2026-08-14T10:00:00+08:00",
		Steps: []DesktopTaskStep{
			{ID: "mkdir", Label: "创建文件夹 proj", Action: "mkdir", Path: "~/Desktop/proj", Status: "pending"},
			{ID: "write_file", Label: "写入文件 README.md", Action: "write_text", Path: "~/Desktop/proj/README.md", Status: "pending"},
		},
	}
	return plan, DesktopTaskProgress{
		Plan: plan, CompletedStepIDs: []string{"mkdir"},
		PendingSteps: []DesktopTaskStep{plan.Steps[1]}, AllPassed: false,
	}
}

func planFileExists(t *testing.T, dir string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(dir, taskPlanFileName))
	return err == nil
}

// TestTaskPlanStorePersistRoundTrip 持久化往返：写入 → 新存储重载 → 内容一致
func TestTaskPlanStorePersistRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewTaskPlanStoreWithDataRoot(dir)
	plan, progress := sampleTaskPlan()
	store.Save("sess-1", plan, progress)

	if !planFileExists(t, dir) {
		t.Fatal("活动计划保存后 task_plan.json 应存在")
	}

	// 模拟重启：全新存储从磁盘加载
	fresh := NewTaskPlanStoreWithDataRoot(dir)
	if err := fresh.ReloadFromDisk(); err != nil {
		t.Fatalf("ReloadFromDisk 失败: %v", err)
	}
	got := fresh.Load("sess-1")
	if got == nil {
		t.Fatal("重载后应能取回活动计划")
	}
	if got.Plan.GoalSummary != plan.GoalSummary {
		t.Errorf("goal 不一致: got %q want %q", got.Plan.GoalSummary, plan.GoalSummary)
	}
	if len(got.CompletedStepIDs) != 1 || got.CompletedStepIDs[0] != "mkdir" {
		t.Errorf("已完成步骤未恢复: %v", got.CompletedStepIDs)
	}
	if got.Status != "active" {
		t.Errorf("状态应为 active, got %q", got.Status)
	}
	if len(got.Plan.Steps) != 2 {
		t.Errorf("步骤数应为 2, got %d", len(got.Plan.Steps))
	}
}

// TestTaskPlanStoreCompleteClearsFile 完成时清除：全通过后文件删除、重载为空
func TestTaskPlanStoreCompleteClearsFile(t *testing.T) {
	dir := t.TempDir()
	store := NewTaskPlanStoreWithDataRoot(dir)
	plan, _ := sampleTaskPlan()
	store.Save("sess-1", plan, DesktopTaskProgress{
		Plan: plan, CompletedStepIDs: []string{"mkdir", "write_file"},
		PendingSteps: nil, AllPassed: true,
	})

	if planFileExists(t, dir) {
		t.Fatal("任务全部完成后 task_plan.json 应被清除")
	}
	fresh := NewTaskPlanStoreWithDataRoot(dir)
	if err := fresh.ReloadFromDisk(); err != nil {
		t.Fatalf("ReloadFromDisk 失败: %v", err)
	}
	if p := fresh.ActivePlan(); p != nil {
		t.Fatalf("完成后不应有活动计划, got %+v", p)
	}
}

// TestTaskPlanStoreClear 取消/清除：Clear 后文件删除、重载为空
func TestTaskPlanStoreClear(t *testing.T) {
	dir := t.TempDir()
	store := NewTaskPlanStoreWithDataRoot(dir)
	plan, progress := sampleTaskPlan()
	store.Save("sess-1", plan, progress)
	if !planFileExists(t, dir) {
		t.Fatal("保存后文件应存在")
	}

	store.Clear("sess-1")
	if store.Load("sess-1") != nil {
		t.Fatal("Clear 后内存应无该计划")
	}
	if planFileExists(t, dir) {
		t.Fatal("Clear 后 task_plan.json 应被删除")
	}
	fresh := NewTaskPlanStoreWithDataRoot(dir)
	_ = fresh.ReloadFromDisk()
	if fresh.ActivePlan() != nil {
		t.Fatal("清除后重载不应恢复任何计划")
	}
}

// TestTaskPlanStoreActivePlanAndResume 恢复入口：ActivePlan 选取 + Resume 语义
func TestTaskPlanStoreActivePlanAndResume(t *testing.T) {
	dir := t.TempDir()
	store := NewTaskPlanStoreWithDataRoot(dir)
	if store.ActivePlan() != nil {
		t.Fatal("空存储 ActivePlan 应为 nil")
	}

	plan, progress := sampleTaskPlan()
	store.Save("sess-1", plan, progress)
	act := store.ActivePlan()
	if act == nil || act.SessionID != "sess-1" {
		t.Fatalf("ActivePlan 应返回 sess-1, got %+v", act)
	}

	if !store.Resume("sess-1") {
		t.Fatal("存在活动计划时 Resume 应返回 true")
	}
	if store.Resume("no-such-session") {
		t.Fatal("不存在的会话 Resume 应返回 false")
	}
	// 恢复后计划仍 active 且已落盘（可被新存储重载）
	fresh := NewTaskPlanStoreWithDataRoot(dir)
	_ = fresh.ReloadFromDisk()
	if p := fresh.Load("sess-1"); p == nil || p.Status != "active" {
		t.Fatal("Resume 后计划应保持 active 并可重载")
	}
}

// TestTaskPlanStoreMemoryModeNoFile 纯内存模式（无数据根）不落盘
func TestTaskPlanStoreMemoryModeNoFile(t *testing.T) {
	store := NewTaskPlanStore()
	plan, progress := sampleTaskPlan()
	store.Save("sess-1", plan, progress)
	if store.ActivePlan() == nil {
		t.Fatal("纯内存模式应保留活动计划")
	}
	if planFileExists(t, t.TempDir()) {
		t.Fatal("无数据根不应产生文件")
	}
}
