package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/util"
)

// ── Ghost Text 内联补全 ─────────────────────────────────────

// GhostComplete 流式返回光标后的补全建议（Ghost Text）
// currentText: 光标前已有的文本（用于 AI 理解上下文续写）
// styleProfile: 可选的风格指导（如 "" 则使用默认）
// 返回 SSE chunk 流，前端实时显示 ghost text
func (c *Client) GhostComplete(ctx context.Context, model string, currentText string, styleProfile string) (<-chan SSEChunk, error) {
	// 截断上下文：只取最后 2000 字符，确保低延迟
	trimmed := currentText
	if len([]rune(trimmed)) > 2000 {
		runes := []rune(trimmed)
		trimmed = string(runes[len(runes)-2000:])
	}

	styleInstruction := ""
	if styleProfile != "" {
		styleInstruction = fmt.Sprintf("\n写作风格要求：%s", styleProfile)
	}

	systemPrompt := fmt.Sprintf(`你是专业小说续写助手。根据已有文本，自然流畅地续写下一段。
规则：
1. 续写必须自然地延续原文的语气、节奏和视角
2. 输出纯续写内容，不要加任何解释、标注或前缀
3. 如果原文是对话未结束，补全对话；如果是叙述，续写叙述
4. 中文写作，保持与原文一致的文风
5. 一次续写 20-80 字即可，不要过长%s`, styleInstruction)

	userPrompt := fmt.Sprintf("已有文本：\n```\n%s\n```\n\n请自然续写：", trimmed)

	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req := &ChatRequest{
		Model:       model,
		Messages:    []ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}},
		MaxTokens:   256,
		Temperature: 0.8,  // 稍高温度增加多样性
		Stream:      true,
		TopP:        0.95,
	}

	resultCh := make(chan SSEChunk, 32)

	go func() {
		defer close(resultCh)

		chunks, err := c.ChatStream(ctx2, req)
		if err != nil {
			resultCh <- SSEChunk{Error: err.Error()}
			return
		}

		for chunk := range chunks {
			if chunk.Error != "" {
				resultCh <- chunk
				return
			}
			if chunk.Done {
				resultCh <- chunk
				return
			}
			resultCh <- chunk
		}
	}()

	return resultCh, nil
}

// ── Cmd+K 命令编辑 ──────────────────────────────────────────

// CmdKEdit 根据自然语言指令编辑选中文本
// selectedText: 用户选中的文本，instruction: 自然语言编辑指令（如"用更紧张的节奏重写"）
// styleProfile: 可选的风格指导
// 返回编辑后的文本
func (c *Client) CmdKEdit(ctx context.Context, model string, selectedText string, instruction string, styleProfile string) (string, error) {
	styleInstruction := ""
	if styleProfile != "" {
		styleInstruction = fmt.Sprintf("\n整体风格要求：%s", styleProfile)
	}

	systemPrompt := fmt.Sprintf(`你是专业小说文本编辑。根据用户指令精确编辑给定文本。

核心规则：
1. 仅编辑用户选中的文本，不要添加额外内容
2. 严格遵循用户的编辑指令
3. 保持文本原有的关键信息（人物、事件、地点）不变
4. 输出纯编辑后的文本，不要加任何解释、标注、引号包裹
5. 中文输出%s`, styleInstruction)

	userPrompt := fmt.Sprintf(`编辑指令：%s

原始文本：
---
%s
---

请输出编辑后的文本：`, instruction, selectedText)

	reply, err := c.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userPrompt, ChatSimpleOptions{
		Temperature: 0.6,
		MaxTokens:   util.Max(len([]rune(selectedText))*2, 1024),
	})
	if err != nil {
		return "", fmt.Errorf("Cmd+K 编辑失败: %w", err)
	}

	// 清理可能的 markdown 包裹
	cleaned := strings.TrimSpace(reply)
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	return cleaned, nil
}

// ── Beat-to-Prose ────────────────────────────────────────────

// Beat 一个叙事节拍（简短动作描述，如"Elara 推开大门，发现大厅空无一人"）
type Beat struct {
	ID          string `json:"id"`
	Description string `json:"description"` // 简短动作描述 (10-50字)
	Order       int    `json:"order"`
}

// GenerateBeats 从大纲节点生成章节级 Beat 列表（3-8个节拍）
func (c *Client) GenerateBeats(ctx context.Context, model string, outlineSummary string, prevChapterSummary string) ([]Beat, error) {
	systemPrompt := `你是小说节拍规划师。将大纲节点拆分为 3-8 个叙事节拍（Beats）。
每个 Beat 是一句简短的动作描述（10-50字），按时间顺序排列。
输出严格 JSON 数组格式。

示例输出：
[{"description": "Elara推开大殿的门，冷风扑面而来"}, {"description": "她看到王座上空无一人，只有一封信"}, {"description": "信上写着'你父亲还活着'"}]`

	userPrompt := fmt.Sprintf("大纲摘要：%s\n上一章结尾：%s\n\n请为此章生成 3-8 个叙事节拍：", outlineSummary, prevChapterSummary)

	reply, err := c.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userPrompt, ChatSimpleOptions{
		Temperature: 0.7,
		MaxTokens:   1024,
	})
	if err != nil {
		return nil, fmt.Errorf("生成节拍失败: %w", err)
	}

	var beats []Beat
	jsonStr := util.ExtractJSON(reply)
	if err := json.Unmarshal([]byte(jsonStr), &beats); err != nil {
		// 尝试修复：JSON 数组可能被包裹在对象里
		if idx := strings.Index(jsonStr, "["); idx >= 0 {
			if end := strings.LastIndex(jsonStr, "]"); end > idx {
				jsonStr = jsonStr[idx : end+1]
				if err2 := json.Unmarshal([]byte(jsonStr), &beats); err2 != nil {
					return nil, fmt.Errorf("解析节拍 JSON 失败: %w (原始: %s)", err, util.Truncate(reply, 200))
				}
			}
		} else {
			return nil, fmt.Errorf("解析节拍 JSON 失败: %w (原始: %s)", err, util.Truncate(reply, 200))
		}
	}

	// 分配 ID 和 Order
	for i := range beats {
		beats[i].ID = fmt.Sprintf("beat-%d", i+1)
		beats[i].Order = i + 1
	}

	slog.Info("生成节拍完成", "count", len(beats))
	return beats, nil
}

// GenerateProseFromBeat 从单个 Beat 流式生成 Prose
// beat: 当前节拍，allBeats: 全部节拍（含前后，用于上下文连贯）
// 返回 SSE chunk 流
func (c *Client) GenerateProseFromBeat(ctx context.Context, model string, beat Beat, allBeats []Beat, contextMap map[string]string) (<-chan SSEChunk, error) {
	// 构建上下文
	var contextParts []string
	for k, v := range contextMap {
		if v != "" {
			contextParts = append(contextParts, fmt.Sprintf("%s: %s", k, v))
		}
	}

	// 构建节拍上下文（前后各一个节拍）
	var beatContext string
	if len(allBeats) > 0 {
		var beatDescs []string
		for _, b := range allBeats {
			marker := ""
			if b.ID == beat.ID {
				marker = " ← 当前"
			}
			beatDescs = append(beatDescs, fmt.Sprintf("  %d. %s%s", b.Order, b.Description, marker))
		}
		beatContext = "全部节拍：\n" + strings.Join(beatDescs, "\n")
	}

	systemPrompt := fmt.Sprintf(`你是小说章节写手。根据给定的叙事节拍（Beat）展开成完整的叙事段落。

规则：
1. 严格围绕当前 Beat 展开，不要跳到后续 Beat
2. 保持与上下 Beat 的自然过渡感
3. 中文写作，文笔细腻有画面感
4. 输出纯正文，不要加任何标注、标题或解释

背景信息：
%s`, strings.Join(contextParts, "\n"))

	userPrompt := fmt.Sprintf("%s\n\n请展开当前节拍「%s」为完整叙事段落：", beatContext, beat.Description)

	resultCh := make(chan SSEChunk, 64)

	go func() {
		defer close(resultCh)

		req := &ChatRequest{
			Model:       model,
			Messages:    []ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}},
			MaxTokens:   2048,
			Temperature: 0.75,
			Stream:      true,
		}

		chunks, err := c.ChatStream(ctx, req)
		if err != nil {
			resultCh <- SSEChunk{Error: err.Error()}
			return
		}

		for chunk := range chunks {
			if chunk.Error != "" {
				resultCh <- chunk
				return
			}
			if chunk.Done {
				resultCh <- chunk
				return
			}
			resultCh <- chunk
		}
	}()

	return resultCh, nil
}
