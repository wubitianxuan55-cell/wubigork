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
		defer func() {
			if r := recover(); r != nil {
				slog.Error("copilot: stream goroutine panic recovered", "panic", r)
				resultCh <- SSEChunk{Error: "生成异常: " + fmt.Sprint(r)}
			}
		}()
		defer close(resultCh)

		// send 在 ctx 取消时安全退出，避免消费者提前返回后永久阻塞泄漏
		send := func(ch SSEChunk) bool {
			select {
			case resultCh <- ch:
				return true
			case <-ctx2.Done():
				return false
			}
		}

		chunks, err := c.ChatStream(ctx2, req)
		if err != nil {
			send(SSEChunk{Error: err.Error()})
			return
		}

		for chunk := range chunks {
			if chunk.Error != "" {
				send(chunk)
				return
			}
			if chunk.Done {
				send(chunk)
				return
			}
			send(chunk)
		}
	}()

	return resultCh, nil
}

// ── Cmd+K 命令编辑 ──────────────────────────────────────────

// CmdKEdit 根据自然语言指令编辑选中文本。
// engineID 指定引擎（功能绑定路由后传入；空=活跃引擎回退），model 指定模型名。
// selectedText: 用户选中的文本，instruction: 自然语言编辑指令（如"用更紧张的节奏重写"）
// styleProfile: 可选的风格指导
// 返回编辑后的文本
func (c *Client) CmdKEdit(ctx context.Context, engineID, model string, selectedText string, instruction string, styleProfile string) (string, error) {
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
		EngineID:    engineID,
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

// OfficeEditText 根据自然语言指令编辑选中的办公文本（框选即改）。
// 与 CmdKEdit（小说向）不同：提示词面向办公文档——措辞严谨、关键信息
// （数字/日期/单位/专有名词）不变、输出纯文本。
func (c *Client) OfficeEditText(ctx context.Context, engineID, model string, selectedText string, instruction string) (string, error) {
	styleInstruction := "\n5. 保留文档原语气与行文风格，不擅自添加营销腔或夸张表达"

	systemPrompt := fmt.Sprintf(`你是专业办公文档编辑助手。根据用户指令精确编辑给定文本。

核心规则：
1. 仅编辑用户选中的文本，不要添加解释、标题、批注或额外内容
2. 严格遵循编辑指令（润色/改写/精简/翻译/扩写/换措辞等）
3. 保持关键信息不变：数字、日期、单位、金额、专有名词、条款含义
4. 输出纯文本，不要 Markdown 标记、引号包裹或前后缀说明
%s`, styleInstruction)

	userPrompt := fmt.Sprintf(`编辑指令：%s

原始文本：
---
%s
---

请输出编辑后的文本：`, instruction, selectedText)

	reply, err := c.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userPrompt, ChatSimpleOptions{
		Temperature: 0.3,
		MaxTokens:   util.Max(len([]rune(selectedText))*2, 1024),
		EngineID:    engineID,
	})
	if err != nil {
		return "", fmt.Errorf("AI 编辑失败: %w", err)
	}

	// 清理可能的 markdown 包裹
	cleaned := strings.TrimSpace(reply)
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned, nil
}

// XlsxEditOps 根据表格上下文与用户指令，规划 Excel 单元格操作（JSON 数组）。
// 返回的字符串由调用方用 util.ExtractJSON 解析为 xlsxedit.Op 列表。
func (c *Client) XlsxEditOps(ctx context.Context, engineID, model string, contextJSON string, selection string, instruction string) (string, error) {
	systemPrompt := `你是 Excel 表格操作规划器。根据表格上下文与用户指令，输出严格 JSON 数组，每个元素是一个操作对象。
支持的操作类型（sheet 使用给定工作表名）：
{"type":"set_formula","sheet":"..","target":"B4","formula":"SUM(B2:B3)"}  在指定单元格写入公式（不含前导 =）
{"type":"set_value","sheet":"..","target":"B4","value":100}  写入常量（数字不带引号）
{"type":"fill_range","sheet":"..","range":"B2:B10","value":0}  常量填充整个区域
{"type":"transform","sheet":"..","range":"C2:C10","formula":"=B2*0.13"}  按区域逐行写公式，模板写第一行公式，相对行引用自动下移
{"type":"replace","sheet":"..","range":"A1:A20","find":"旧","replace":"新"}  区域内查找替换文本
{"type":"split_column","sheet":"..","col":"A","sep":"-","newCols":["B","C"],"headers":["省","市"]}  按分隔符拆分列到新列（可选写表头）
{"type":"clean","sheet":"..","range":"B2:B10","trim":true,"upper":false,"lower":false}  清洗：去空格/转大写/转小写
{"type":"set_style","sheet":"..","range":"A1:D1","style":{"bold":true,"italic":false,"underline":false,"fontSize":12,"fontColor":"9C0006","fill":"FFF2CC","numFmt":"0.00%","align":"center","wrap":true}}  设置样式，只写需要改的字段，其余保留（bold/italic/underline/wrap 用 true/false；颜色 RRGGBB 不带 #；numFmt 如 "0.00" 或 "0.00%"）
{"type":"merge_cells","sheet":"..","range":"A1:C1"}  合并区域（跨列表头常用）
{"type":"unmerge_cells","sheet":"..","range":"A1:C1"}  取消合并区域
{"type":"set_col_width","sheet":"..","col":"A","width":18}  设置列宽（8~40 合理）
规则：
1. 只输出 JSON 数组，不要任何解释、Markdown 或注释
2. target/range 必须落在数据区域内，列字母大写
3. 只做用户要求的最小改动，不得臆造数据（计算类优先用公式而非写死结果）
4. 用户选区是目标单元格时优先用 set_formula/set_value；列级操作（拆分/清洗/变换）才用区域
5. 样式/合并/列宽用对应类型；用户没提外观就不要擅自改样式`

	userPrompt := fmt.Sprintf(`表格上下文（JSON）：
%s

用户选区：%s
用户指令：%s

请输出操作 JSON 数组：`, contextJSON, selection, instruction)

	reply, err := c.ChatSimpleStreamWithOptions(ctx, model, systemPrompt, userPrompt, ChatSimpleOptions{
		Temperature: 0.2,
		MaxTokens:   2048,
		EngineID:    engineID,
	})
	if err != nil {
		return "", fmt.Errorf("表格操作规划失败: %w", err)
	}
	return reply, nil
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
		defer func() {
			if r := recover(); r != nil {
				slog.Error("copilot: stream goroutine panic recovered", "panic", r)
				resultCh <- SSEChunk{Error: "生成异常: " + fmt.Sprint(r)}
			}
		}()
		defer close(resultCh)

		// send 在 ctx 取消时安全退出，避免消费者提前返回后永久阻塞泄漏
		send := func(ch SSEChunk) bool {
			select {
			case resultCh <- ch:
				return true
			case <-ctx.Done():
				return false
			}
		}

		req := &ChatRequest{
			Model:       model,
			Messages:    []ChatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}},
			MaxTokens:   2048,
			Temperature: 0.75,
			Stream:      true,
		}

		chunks, err := c.ChatStream(ctx, req)
		if err != nil {
			send(SSEChunk{Error: err.Error()})
			return
		}

		for chunk := range chunks {
			if chunk.Error != "" {
				send(chunk)
				return
			}
			if chunk.Done {
				send(chunk)
				return
			}
			send(chunk)
		}
	}()

	return resultCh, nil
}
