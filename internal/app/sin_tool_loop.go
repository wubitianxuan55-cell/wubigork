package app

// ── 原罪工具循环（v4.262 刀）──
//
// 把 SinStream 从「单轮流式」升级为「有界多轮工具循环」，形状与办公 agent 的
// 工具循环同源但范围小得多：轮次上限 4，最后一轮不带 tools 强制收尾成正文，
// 工具集只有原罪域内那 5 个（sin_tools.go 注册表），不碰办公工作区/命令/记忆面。
//
// 三个不变式：
//  1. 正文 = 模型输出。每轮 content 按轮序拼接落库，后端不改写一个字；工具结果
//     只喂回模型，不落进正文。
//  2. 工具是增强不是前置条件。首轮带工具直接失败（模型/端点不支持 tools）→ 去掉
//     工具重试一次并给 notice，写作不中断；工具自己失败只记进轨迹，不炸整轮。
//  3. 过程可见。dispatch/result 逐条实时下发（前端过程卡据此渲染），收尾时同一
//     份轨迹随 done.tools 与消息 extra.tools 落库（重开故事可还原）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
)

const (
	// sinToolRoundsMax 工具循环轮次上限（含收尾轮）。防模型拿工具原地打转：
	// 到顶后最后一轮不带 tools，必须给出正文。
	sinToolRoundsMax = 4
	// sinToolOutputMaxRunes 单个工具结果的长度上限（模型与过程卡看同一份）。
	// 检索/抓取天然很长（web_fetch 单页上限 1 MiB），不设闸门一次就能把上下文
	// 与费用打穿。
	sinToolOutputMaxRunes = 6000
	// sinToolCallMaxPerTool 同一个工具在一轮里的调用次数上限。真机走查实测：
	// 不设闸门时模型会拿 web_search 一路查到轮次封顶（12 次），正文只剩几十字——
	// 工具是手段，写作是目的。
	sinToolCallMaxPerTool = 3
	// sinToolCallMaxTotal 一轮里所有工具的调用次数上限（跨工具防打转）。
	sinToolCallMaxTotal = 8
	// sinFinalizeNudge 收尾轮注入的收束令。真机走查实测：不点名收尾时，模型会在
	// 末轮继续要工具（该轮不带 tools），这一轮就没有正文 → 整轮报错、什么都没落。
	sinFinalizeNudge = "（工具阶段结束：现在直接输出这一轮的故事正文，不要再调用工具。）"
)

// sinToolBudget 本轮工具调用预算：同名次数 + 总量双闸门。超限不执行，如实
// 回绝并把「用已有信息收尾」写进结果文本（模型看得见，用户也看得见）。
type sinToolBudget struct {
	perTool map[string]int
	total   int
}

func (b *sinToolBudget) allow(name string) (bool, string) {
	if b.perTool == nil {
		b.perTool = map[string]int{}
	}
	if b.perTool[name] >= sinToolCallMaxPerTool {
		return false, fmt.Sprintf("本轮 %s 已调用 %d 次（上限 %d）：请用已有信息继续写作，不要再调用它",
			name, b.perTool[name], sinToolCallMaxPerTool)
	}
	if b.total >= sinToolCallMaxTotal {
		return false, fmt.Sprintf("本轮工具调用已达上限（%d 次）：请用已有信息把这一轮写完",
			sinToolCallMaxTotal)
	}
	b.perTool[name]++
	b.total++
	return true, ""
}

// sinRoundResult 单轮流式结果。
type sinRoundResult struct {
	content   string
	reasoning string
	calls     []ai.ChatToolCall
	usage     *ai.ChatUsage
}

// sinStreamRound 跑一轮流式请求，把 delta/reasoning 实时下发给前端。
// ctx 是整轮上下文：SinCancel 取消它即断开本轮底层请求。
func (a *App) sinStreamRound(ctx context.Context, runID, model string, opts ai.ChatSimpleOptions, messages []ai.ChatMessage, tools []ai.ChatToolSchema, run *sinRun) (sinRoundResult, error) {
	chunks, cancel, err := a.client.ChatStreamMessages(ctx, model, messages, opts, tools)
	if err != nil {
		return sinRoundResult{}, err
	}
	defer cancel()

	var res sinRoundResult
	var content, reasoning strings.Builder
	for chunk := range chunks {
		if run.cancelled.Load() {
			break
		}
		if chunk.Error != "" {
			// 取消引发的读错误按「已停止」收尾（不当失败报）
			if run.cancelled.Load() {
				break
			}
			return res, fmt.Errorf("%s", chunk.Error)
		}
		if chunk.Done {
			res.calls = chunk.ToolCalls
			res.usage = chunk.Usage
			break
		}
		if chunk.Content != "" {
			content.WriteString(chunk.Content)
			a.emit("sin-stream:"+runID, map[string]interface{}{"type": "delta", "content": chunk.Content})
		}
		if chunk.Reasoning != "" {
			reasoning.WriteString(chunk.Reasoning)
			a.emit("sin-stream:"+runID, map[string]interface{}{"type": "reasoning", "content": chunk.Reasoning})
		}
	}
	res.content = content.String()
	res.reasoning = reasoning.String()
	return res, nil
}

// sinRunToolCall 执行一次工具调用：下发 tool_dispatch / tool_result、登记轨迹，
// 返回喂回模型的 tool 消息。未知工具如实回错（可用清单照列，不猜、不静默吞）。
// budget 是调用预算：超限不执行（模型拿不到新结果，只能收尾），但照发 process 帧，
// 让用户看得见「它想再调、被拦了」。
func (a *App) sinRunToolCall(ctx context.Context, runID string, tools []sinTool, call ai.ChatToolCall, trace *[]sinToolTrace, budget *sinToolBudget) ai.ChatMessage {
	name := call.Function.Name
	args := strings.TrimSpace(call.Function.Arguments)
	if args == "" {
		args = "{}" // 无参调用：给合法空对象，别让工具的 Unmarshal 报 syntax error
	}
	t := sinToolByName(tools, name)
	tr := sinToolTrace{
		ID:       call.ID,
		Name:     name,
		Args:     args,
		ReadOnly: t != nil && t.ReadOnly(),
	}
	a.emit("sin-stream:"+runID, sinToolDispatchFrame(tr))

	start := time.Now()
	var output string
	allowed, refuse := true, ""
	if budget != nil {
		allowed, refuse = budget.allow(name)
	}
	switch {
	case !allowed:
		tr.Error = refuse
		output = "工具未执行：" + refuse
	case t == nil:
		tr.Error = "未知工具：" + name
		output = "工具 " + name + " 不存在。可用工具：" + strings.Join(sinToolNames(tools), "、")
	default:
		out, err := t.Execute(ctx, json.RawMessage(args))
		if err != nil {
			tr.Error = err.Error()
			output = "工具执行失败：" + err.Error()
		} else {
			tr.Output = sinClampToolOutput(out)
			output = tr.Output
		}
	}
	tr.ElapsedMS = time.Since(start).Milliseconds()
	*trace = append(*trace, tr)

	a.emit("sin-stream:"+runID, sinToolResultFrame(tr))
	return ai.ChatMessage{Role: "tool", ToolCallID: call.ID, Name: name, Content: output}
}

// sinToolDispatchFrame / sinToolResultFrame 是前端过程卡的线格式契约。
// 独立成纯函数：字段名一旦漂移，前端渲染就静默变空，所以在这里单点固定并由
// 单测逐键锁定（与 sinToolTrace 的 JSON 标签同一套字段名）。
func sinToolDispatchFrame(tr sinToolTrace) map[string]interface{} {
	return map[string]interface{}{
		"type":      "tool_dispatch",
		"id":        tr.ID,
		"name":      tr.Name,
		"args":      tr.Args,
		"read_only": tr.ReadOnly,
	}
}

func sinToolResultFrame(tr sinToolTrace) map[string]interface{} {
	return map[string]interface{}{
		"type":       "tool_result",
		"id":         tr.ID,
		"name":       tr.Name,
		"output":     tr.Output,
		"error":      tr.Error,
		"elapsed_ms": tr.ElapsedMS,
	}
}

// sinClampToolOutput 截断过长的工具结果并留可见标记（截断就如实说，不静默切）。
func sinClampToolOutput(s string) string {
	if len([]rune(s)) <= sinToolOutputMaxRunes {
		return s
	}
	return truncateRunes(s, sinToolOutputMaxRunes) + "\n（结果过长已截断）"
}

// sinToolNames 本轮工具名（未知工具回错时列给模型）。
func sinToolNames(tools []sinTool) []string {
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		out = append(out, t.Name())
	}
	return out
}

// sinAccumulateUsage 累加多轮 token 用量：工具循环 = 同一轮里发了多次请求，
// 成本必须如实合计（否则计费面板会少算）。只合计顶层 prompt/completion/total
// ——EstimateCostCNY 只用这两个数。
func sinAccumulateUsage(total, add *ai.ChatUsage) *ai.ChatUsage {
	if add == nil {
		return total
	}
	if total == nil {
		total = &ai.ChatUsage{}
	}
	total.PromptTokens += add.PromptTokens
	total.CompletionTokens += add.CompletionTokens
	total.TotalTokens += add.TotalTokens
	total.PromptCacheHitTokens += add.PromptCacheHitTokens
	total.PromptCacheMissTokens += add.PromptCacheMissTokens
	return total
}
