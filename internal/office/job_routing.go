// Package office — job_routing.go
package office

import "strings"

var taskTriggers = []string{
	"帮我", "整理", "搜索文件", "查找", "打开", "关闭",
	"下载", "安装", "删除", "移动", "复制", "新建",
	"读取", "阅读", "看一下", "检查", "分析",
	"list", "search", "open", "close", "download",
	"列出", "显示", "创建", "统计",
}

func IsTask(text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" { return false }
	for _, trigger := range taskTriggers {
		if strings.Contains(text, strings.ToLower(trigger)) { return true }
	}
	return false
}
