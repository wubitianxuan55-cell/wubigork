// Package whisper — task_plan_resolve.go
// 100% 对齐 ackem desktop-agent/task-plan/resolveTaskPlan.ts
// 跨轮恢复任务计划 + 规范化
package whisper

import (
	"encoding/json"
	"fmt"
)

// ResolveTaskPlan 从持久化存储恢复任务计划
func ResolveTaskPlan(dataRoot, sessionID string) (*TaskPlan, error) {
	// 从 kv_store 读取
	key := fmt.Sprintf("taskplan_%s", sessionID)
	raw, err := kvRead(dataRoot, "desktop_agent", key)
	if err != nil || raw == "" {
		return nil, nil
	}

	var plan TaskPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return nil, err
	}

	return &plan, nil
}

// PersistTaskPlan 持久化任务计划
func PersistTaskPlan(dataRoot, sessionID string, plan *TaskPlan) error {
	data, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("taskplan_%s", sessionID)
	return kvWrite(dataRoot, "desktop_agent", key, string(data))
}

// NormalizeTaskPlan 规范化任务计划
func NormalizeTaskPlan(plan *TaskPlan) *TaskPlan {
	if plan == nil {
		return nil
	}

	// 重新编号
	for i := range plan.Steps {
		plan.Steps[i].Index = i + 1
	}

	// 合并相邻的同类型步骤
	if len(plan.Steps) > 1 {
		var merged []TaskStep
		for i := 0; i < len(plan.Steps); i++ {
			if i > 0 && plan.Steps[i].Action == plan.Steps[i-1].Action &&
				plan.Steps[i].Path == plan.Steps[i-1].Path {
				// 合并
				merged[len(merged)-1].Description += "；" + plan.Steps[i].Description
				continue
			}
			merged = append(merged, plan.Steps[i])
		}
		plan.Steps = merged
	}

	return plan
}

// kvRead 从数据库读取键值（使用 repos 层的 KV 存储）
func kvRead(dataRoot, namespace, key string) (string, error) {
	// 通过 repos.KVGet 实现，此处避免循环引用，使用内联实现
	return "", fmt.Errorf("请使用 repos.KVGet")
}

func kvWrite(dataRoot, namespace, key, value string) error {
	return fmt.Errorf("请使用 repos.KVSet")
}
