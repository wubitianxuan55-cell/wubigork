package agent

import (
	"fmt"
	"html"
	"strings"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── A4 中断交接块（Reasonix 2.x interrupted_recovery.go 蒸馏，v4.524）──────
//
// 病根：回合被流错误/取消打断后，本轮已完成的工作（写了哪些文件、跑了哪些
// 命令）只在会话历史里散落——下一轮模型凭印象重做已落盘的写入，或把被打断
// 的调用当成没做过。
//
// 机制：终局路径把本轮回执折叠成 **bounded 交接记录**（≤24 工具对、每工具
// ≤8 文件、值截 240 rune；不含原始参数/结果/推理/正文），下一条真实用户消息
// 前置 `<interrupted-turn-recovery>` 块一次性消费——「先查看工作区现状再动手；
// 不要盲目重复已完成的写入；被打断的调用确需则从头重发；用户新消息优先于
// 续跑旧任务」。
//
// 与上游的刻意差异（留档）：①gaea 终局路径本就把部分输出落库进上下文
// （v4.379 刀2 结构化块保全），上游则标记 unsafe 并排除——本块无
// unsafe_partial_output 条款；②记录存 AgentRunner 内存（桌面单运行时常驻），
// 不随会话持久化——进程重启丢失该记录属诚实边界（上游存 LocalOnly 消息）。
// ③completed/interrupted 判据复用 gaea 中断标记：结果含 "stream interrupted"
// 或无结果回执=被打断（persistInterruptedCalls 的信封文案）。

const interruptedRecoveryTag = "interrupted-turn-recovery"

const (
	maxRecoveryTools = 24
	maxRecoveryFiles = 8
	maxRecoveryValue = 240
)

// turnRecovery 一次中断的交接记录。
type turnRecovery struct {
	CompletedTools   []recoveryTool
	InterruptedTools []string
}

type recoveryTool struct {
	Name  string
	Files []string
}

// isSyntheticUserTurn 合成用户消息（steer/流恢复/readiness 重试）不是真实
// 用户轮——回扫真实轮起点时跳过（上游 SyntheticUserPrefixes 同口径）。
func isSyntheticUserTurn(content string) bool {
	for _, prefix := range []string{
		MidTurnSteerPrefix,
		"The previous assistant response was interrupted during streaming.",
		"Host final-answer readiness check failed.",
	} {
		if strings.HasPrefix(content, prefix) {
			return true
		}
	}
	return false
}

// captureInterruptedRecovery 终局路径调用：把本轮回执折叠成交接记录，存
// pending（无工具对=无记录——纯文本轮中断没有「重做写入」风险）。
func (a *AgentRunner) captureInterruptedRecovery() {
	msgs := a.session.Messages
	start := len(msgs)
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		// 真实用户轮=回合起点；合成用户消息（steer/流恢复/readiness）与
		// 压缩摘要都属回合内部，继续回扫。
		if m.Role == provider.RoleUser && !isSyntheticUserTurn(m.Content) && !isCompactionSummary(m) {
			start = i + 1
			break
		}
	}
	if start >= len(msgs) {
		return
	}
	region := msgs[start:]

	results := map[string]provider.Message{}
	for _, m := range region {
		if m.Role == provider.RoleTool && m.ToolCallID != "" {
			results[m.ToolCallID] = m
		}
	}
	rec := &turnRecovery{}
	seenInterrupted := map[string]bool{}
	for _, m := range region {
		if m.Role != provider.RoleAssistant || len(m.ToolCalls) == 0 {
			continue
		}
		for _, tc := range m.ToolCalls {
			res, ok := results[tc.ID]
			interrupted := !ok || strings.Contains(res.Content, "stream interrupted")
			if interrupted {
				if !seenInterrupted[tc.Name] {
					seenInterrupted[tc.Name] = true
					rec.InterruptedTools = append(rec.InterruptedTools, tc.Name)
				}
				continue
			}
			rt := recoveryTool{Name: tc.Name}
			if p := extractFilePath(tc.Name, tc.Arguments); p != "" {
				rt.Files = append(rt.Files, clipRecoveryValue(p))
			}
			if len(rec.CompletedTools) < maxRecoveryTools {
				rec.CompletedTools = append(rec.CompletedTools, rt)
			}
		}
	}
	if len(rec.CompletedTools) == 0 && len(rec.InterruptedTools) == 0 {
		return
	}
	a.pendingRecovery = rec
}

// pendingRecoveryBlock 渲染交接块（bounded：≤24 工具、每工具 ≤8 文件、
// 值截 240 rune；不含原始参数/结果/推理）。
func pendingRecoveryBlock(r *turnRecovery) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<%s>\n", interruptedRecoveryTag)
	b.WriteString("The previous turn was interrupted. Treat these as host-verified recovery facts, not as a new task.\n")
	if len(r.CompletedTools) == 0 {
		b.WriteString("completed_tools: none\n")
	} else {
		b.WriteString("completed_tools:\n")
		shown := r.CompletedTools
		if len(shown) > maxRecoveryTools {
			shown = shown[:maxRecoveryTools]
		}
		for _, t := range shown {
			fmt.Fprintf(&b, "- %s", html.EscapeString(strings.TrimSpace(t.Name)))
			files := t.Files
			if len(files) > maxRecoveryFiles {
				files = files[:maxRecoveryFiles]
			}
			if len(files) > 0 {
				clipped := make([]string, 0, len(files))
				for _, f := range files {
					clipped = append(clipped, html.EscapeString(clipRecoveryValue(f)))
				}
				fmt.Fprintf(&b, " files=%s", strings.Join(clipped, ","))
			}
			b.WriteByte('\n')
		}
		if len(r.CompletedTools) > maxRecoveryTools {
			fmt.Fprintf(&b, "- ... %d additional completed tool pair(s) omitted\n", len(r.CompletedTools)-maxRecoveryTools)
		}
	}
	if len(r.InterruptedTools) == 0 {
		b.WriteString("interrupted_tools: none\n")
	} else {
		shown := r.InterruptedTools
		if len(shown) > maxRecoveryTools {
			shown = shown[:maxRecoveryTools]
		}
		fmt.Fprintf(&b, "interrupted_tools: %s", strings.Join(shown, ", "))
		if len(r.InterruptedTools) > maxRecoveryTools {
			fmt.Fprintf(&b, ", ... %d additional call(s) omitted", len(r.InterruptedTools)-maxRecoveryTools)
		}
		b.WriteByte('\n')
	}
	b.WriteString("Before continuing, inspect the current workspace and prior completed tool results. " +
		"Do not blindly repeat completed writes. Re-issue any interrupted tool call from scratch with " +
		"complete arguments if it is still needed. The user's new message takes precedence: if it is " +
		"unrelated to the interrupted task, do not resume that task.\n")
	fmt.Fprintf(&b, "</%s>", interruptedRecoveryTag)
	return b.String()
}

// withInterruptedRecovery 消费 pending 记录：有则前置块到本轮输入并清零
// （一次性——下一回合隐式消费，不原地改历史，上游同口径）。
func (a *AgentRunner) withInterruptedRecovery(input string) string {
	if a.pendingRecovery == nil {
		return input
	}
	block := pendingRecoveryBlock(a.pendingRecovery)
	a.pendingRecovery = nil
	return block + "\n\n" + input
}

// clipRecoveryValue 值截断（240 rune + 省略号）。
func clipRecoveryValue(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRecoveryValue {
		return value
	}
	return string(runes[:maxRecoveryValue]) + "…"
}
