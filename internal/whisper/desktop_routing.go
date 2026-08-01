// Package whisper — desktop_routing.go
// 100% 对齐 ackem desktop-agent/routing/
package whisper

import "strings"

// CapabilityRoute 能力路由结果
type CapabilityRoute struct {
	CanHandle  bool   `json:"canHandle"`
	Capability string `json:"capability,omitempty"`
	HelpReply  string `json:"helpReply,omitempty"`
}

// DesktopCapabilities 桌面助手能力列表
var DesktopCapabilities = []string{
	"list_folder", "search_files", "read_text", "open_app",
	"close_app", "copy_path", "move_path", "mkdir",
	"delete_path", "download_file", "stat_file",
}

// ResolveDesktopCapability 解析请求能否由桌面助手处理
func ResolveDesktopCapability(userMsg string) CapabilityRoute {
	lower := strings.ToLower(userMsg)

	// 简单关键词匹配
	keywordMap := map[string][]string{
		"list_folder":   {"列出", "查看目录", "有什么文件", "文件夹里"},
		"search_files":  {"搜索", "查找", "找一下", "有没有"},
		"read_text":     {"读取", "打开文件", "读一下", "查看文件"},
		"open_app":      {"打开", "启动", "运行"},
		"close_app":     {"关闭", "退出", "结束"},
		"delete_path":   {"删除", "清理", "移除"},
		"download_file": {"下载", "保存"},
	}

	for cap, keywords := range keywordMap {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return CapabilityRoute{
					CanHandle:  true,
					Capability: cap,
				}
			}
		}
	}

	return CapabilityRoute{
		CanHandle: false,
		HelpReply: "我暂时无法在电脑上帮你做这件事。你可以试试让我帮你：查看文件、搜索目录、打开应用、整理文件。",
	}
}
