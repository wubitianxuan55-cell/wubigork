// Package whisper — capability_help_reply.go
// 100% 对齐 ackem desktop-agent/routing/capabilityHelpReply.ts
// 用 LLM 合成「电脑助手能做什么」的自然语言回复

package whisper

import "strings"

// ─── 能力清单 ──────────────────────────────────────────────────

// DesktopCapability 桌面助手能力
type DesktopCapability struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Detail  string `json:"detail"`
	Enabled bool   `json:"enabled"`
}

// ListDesktopCapabilities 列出桌面助手全部能力
// 100% 对齐 ackem shared/desktopAgentCapabilityHint.ts listDesktopAgentCapabilities
func ListDesktopCapabilities() []DesktopCapability {
	return []DesktopCapability{
		{
			ID:      "investigate_games",
			Label:   "游戏清单",
			Detail:  "查找本机已安装的游戏（Steam、Epic、开始菜单等）",
			Enabled: true,
		},
		{
			ID:      "investigate_documents",
			Label:   "文件搜索",
			Detail:  "搜索桌面/文档/下载目录中的文件（PDF、Word、Excel、PPT 等）",
			Enabled: true,
		},
		{
			ID:      "open_app",
			Label:   "打开应用",
			Detail:  "帮你打开本机已安装的应用程序",
			Enabled: true,
		},
		{
			ID:      "close_app",
			Label:   "关闭应用",
			Detail:  "帮你关闭正在运行的应用程序",
			Enabled: true,
		},
		{
			ID:      "read_file",
			Label:   "读取文件",
			Detail:  "读取并总结文本文件内容（.txt .md .log 等）",
			Enabled: true,
		},
		{
			ID:      "search_files",
			Label:   "搜索文件",
			Detail:  "按名称在电脑上搜索文件",
			Enabled: true,
		},
		{
			ID:      "list_folder",
			Label:   "浏览目录",
			Detail:  "列出指定文件夹的内容",
			Enabled: true,
		},
	}
}

// formatCapabilityLines 格式化能力清单为提示词
func formatCapabilityLines() string {
	caps := ListDesktopCapabilities()
	var lines []string
	for _, c := range caps {
		status := ""
		if !c.Enabled {
			status = "（当前未开）"
		}
		lines = append(lines, "- "+c.Label+status+"："+c.Detail)
	}
	return strings.Join(lines, "\n")
}

// ─── LLM 合成 ──────────────────────────────────────────────────

// SynthesizeCapabilityHelpReply 合成能力帮助回复
// 100% 对齐 ackem capabilityHelpReply.ts synthesizeCapabilityHelpReply
// llmCall: (systemPrompt, userPrompt) → (reply, error)
func SynthesizeCapabilityHelpReply(
	userQuery string,
	llmCall func(systemPrompt, userPrompt string) (string, error),
) (string, error) {
	capabilities := formatCapabilityLines()

	systemPrompt := strings.Join([]string{
		"你是用户的 AI 伴侣。用户问电脑助手能做什么。",
		"用自然中文介绍下列已开放/未开放能力，给 1~2 个具体例子，保持人设，不要堆 action 名或路径。",
		"未标注「当前未开」的可以举例；标注未开的要说明需在设置里开启。",
	}, " ")

	userPrompt := "用户问题：" + userQuery + "\n\n当前能力清单：\n" + capabilities

	reply, err := llmCall(systemPrompt, userPrompt)
	if err != nil {
		return buildCapabilityHelpFallback(), nil
	}

	reply = strings.TrimSpace(reply)
	if reply == "" {
		return buildCapabilityHelpFallback(), nil
	}

	return reply, nil
}

// buildCapabilityHelpFallback 静态回退文本
func buildCapabilityHelpFallback() string {
	lines := []string{
		"我可以在这台电脑上帮你：",
	}
	for _, c := range ListDesktopCapabilities() {
		if c.Enabled {
			lines = append(lines, "· "+c.Label+"："+c.Detail)
		}
	}
	lines = append(lines, "", "试试对我说「帮我看看电脑上有哪些游戏」或「桌面有什么文档」。")
	return strings.Join(lines, "\n")
}
