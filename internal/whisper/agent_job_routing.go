// Package whisper — agent_job_routing.go
// 100% 对齐 ackem desktop-agent/agentJobRouting.ts
// 桌面助手任务路由：判断是否应路由到后台 job

package whisper

import "strings"

// desktopAgentTaskTriggers 可操作的桌面助手任务触发词
var desktopAgentTaskTriggers = []string{
	"帮我", "整理", "搜索文件", "查找", "打开", "关闭",
	"下载", "安装", "删除", "移动", "复制", "新建",
	"读取", "阅读", "看一下", "检查", "分析",
	"list", "search", "open", "close", "download",
	"focus", "import",
}

// IsActionableDesktopAgentTask 判断是否为可操作的桌面助手任务
func IsActionableDesktopAgentTask(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return false
	}
	for _, trigger := range desktopAgentTaskTriggers {
		if strings.Contains(text, strings.ToLower(trigger)) {
			return true
		}
	}
	return false
}

// ShouldRouteDesktopAgentToBackgroundJob 判断是否应路由到后台 job
func ShouldRouteDesktopAgentToBackgroundJob(userText string) bool {
	return IsActionableDesktopAgentTask(userText)
}

// DESKTOP_AGENT_TASK_START_ACK 任务启动确认
const DesktopAgentTaskStartAck = "DESKTOP_AGENT_TASK_START_ACK"
