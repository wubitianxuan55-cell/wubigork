// Package whisper — opening.go
// 100% 对齐 ackem desktop-agent/opening.ts
// 桌面助手开场白生成

package whisper

import "fmt"

// GenerateDesktopAgentOpening 生成桌面助手开场白
func GenerateDesktopAgentOpening(companionName, emotionLabel string, llm LlmClient) string {
	systemPrompt := fmt.Sprintf(
		`你是 %s，用户的 AI 伴侣。电脑助手模式刚开启。用一两句自然、温柔的中文主动问用户今天想在电脑上帮你做什么（例如整理文件、读文档、打开软件）。不要列技术命令，不要自称机器人。当前情绪：%s。`,
		companionName, emotionLabel,
	)
	userPrompt := "[系统] 电脑助手模式已开启，请向用户开场。"

	reply, err := llm.Chat(systemPrompt, userPrompt)
	if err != nil || reply == "" {
		return "今天想让我在电脑上帮你做点什么？"
	}
	return reply
}
