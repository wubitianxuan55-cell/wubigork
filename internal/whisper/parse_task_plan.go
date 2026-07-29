// Package whisper — parse_task_plan.go
// 100% 对齐 ackem desktop-agent/task-plan/parseTaskPlan.ts
// 规则解析多步骤本机任务（V1：桌面建夹→写文件→打开→删除）

package whisper

import (
	"regexp"
	"time"
)

func ParseDesktopAgentTaskPlan(userText string) *DesktopTaskPlan {
	if !IsMultiStepDesktopAgentTask(userText) {
		return nil
	}
	steps := buildDesktopTaskSteps(userText)
	if len(steps) == 0 {
		return nil
	}
	return &DesktopTaskPlan{
		ID: genHexID(), GoalSummary: userText, Steps: steps,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
}

func buildDesktopTaskSteps(text string) []DesktopTaskStep {
	desktop := "~/Desktop"
	var steps []DesktopTaskStep

	folderRE := regexp.MustCompile(`桌面(?:上)?(?:建|创建|新建).*?[「"'']?([^「"'\s,，。；;]+)[「"'']?.*文件夹`)
	folderMatch := folderRE.FindStringSubmatch(text)
	var folderName, folderPath string
	if len(folderMatch) > 1 {
		folderName = folderMatch[1]
		folderPath = desktop + "/" + folderName
	}

	fileRE := regexp.MustCompile(`写(?:入|个|一个)?\s*[「"'']?([^「"'\s,，。；;]+)[「"'']?`)
	fileMatch := fileRE.FindStringSubmatch(text)
	var fileName, filePath string
	if len(fileMatch) > 1 {
		fileName = fileMatch[1]
		if folderPath != "" {
			filePath = folderPath + "/" + fileName
		} else {
			filePath = desktop + "/" + fileName
		}
	}

	wantsOpen := regexp.MustCompile(`打开|看看|读一下|查看内容`).MatchString(text)
	wantsDelete := regexp.MustCompile(`删掉|删除|移除`).MatchString(text)

	if folderPath != "" {
		steps = append(steps, DesktopTaskStep{
			ID: "mkdir", Label: "创建文件夹 " + folderName,
			Action: "mkdir", Path: folderPath, Status: "pending",
			Verify: []DesktopTaskVerify{
				{Type: "path_exists", Path: folderPath},
				{Type: "is_directory", Path: folderPath},
			},
		})
	}
	if filePath != "" {
		steps = append(steps, DesktopTaskStep{
			ID: "write_file", Label: "写入文件 " + fileName,
			Action: "write_text", Path: filePath, Status: "pending",
			Options: map[string]interface{}{"content": "hello"},
			Verify: []DesktopTaskVerify{
				{Type: "path_exists", Path: filePath},
				{Type: "file_min_bytes", Path: filePath, MinBytes: 1},
			},
		})
	}
	if filePath != "" && wantsOpen {
		steps = append(steps, DesktopTaskStep{
			ID: "inspect_file", Label: "读取 " + fileName,
			Action: "read_text", Path: filePath, Status: "pending",
			Verify: []DesktopTaskVerify{
				{Type: "audit_action", Action: "read_text", Path: filePath, Result: "allowed"},
			},
		})
	}
	if filePath != "" && wantsDelete {
		steps = append(steps, DesktopTaskStep{
			ID: "delete_file", Label: "删除 " + fileName,
			Action: "delete_path", Path: filePath, Status: "pending",
			Verify: []DesktopTaskVerify{
				{Type: "path_absent", Path: filePath},
			},
		})
	}
	return steps
}
