package app

// chat_tool_loop.go — B7 有界工具循环（2026-11-06 用户拍板：搜索注册为真工具，
// 模型自己判断要不要搜、搜什么；按钮语义升级为「允许联网」总开关，关键词
// 门控退役）。引擎门控已实弹验证（8124 llama-server 真吐 tool_calls，
// web_search {"query":"成都明天天气"}）。
//
// 设计约束：
//   - 有界：每次回合至多 2 次搜索、3 轮模型调用；末轮去掉工具强制收敛正文
//   - 单工具面：只暴露 web_search（读写面其余工具归 agent/办公链，不进聊天）
//   - 流式透传：每轮正文/思考 delta 即刻外发（模型的「先搜一下」旁白自然可见）
//   - 零风险回滚面：searchEnabled=false 时不传工具，请求与旧版逐字节一致
//   - voice（lowLatency）不走本循环，保留 B5/B6 规则+拟查询路径（延迟敏感）

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// webSearchToolDef web_search 工具定义——从 builtin 注册表取工具本体权威
// 定义（Name/Description/Schema 单源，办公 agent 同源；B7 修正：不手写
// schema，工具本体的 url 直抓等参数面聊天免费继承）。查无此工具返回 false，
// 调用方降级为无工具直答。
func webSearchToolDef() (ai.ChatToolSchema, bool) {
	t, ok := tool.LookupBuiltin("web_search")
	if !ok {
		return ai.ChatToolSchema{}, false
	}
	return ai.ChatToolSchema{
		Type: "function",
		Function: ai.ChatToolFunctionSpec{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		},
	}, true
}

// toolSearchMax 每回合搜索上限；toolLoopMaxRounds 模型调用轮上限（末轮无工具强收敛）。
const (
	toolSearchMax     = 2
	toolLoopMaxRounds = 3
)

// toolLoopResult 一次有界循环的产出（对齐 whisperChatCore 消费面）。
type toolLoopResult struct {
	Reply      string
	Reasoning  string
	Usage      *ai.ChatUsage
	FirstToken time.Time
	Searches   int
}

// runBoundedSearchToolLoop 有界工具循环主体。messages 会被就地追加（assistant
// tool_calls 与 tool 结果对），调用方传入初始 [system, history..., user]。
// onDelta/onReasoning 为空时不外发（阻塞面聚合）。
func (a *App) runBoundedSearchToolLoop(
	ctx context.Context,
	engine, model string,
	messages []ai.ChatMessage,
	opts ai.ChatSimpleOptions,
	mustSearch bool,
	onDelta, onReasoning func(string),
) (toolLoopResult, error) {
	res := toolLoopResult{}
	toolDef, ok := webSearchToolDef()
	if !ok {
		// 工具本体未注册（理论不可达）：降级为无工具直答，不失败
		res.Reasoning = ""
		return res, nil
	}
	tools := []ai.ChatToolSchema{toolDef}
	if mustSearch && len(messages) > 0 {
		// 强制搜索：指令并入首条 system 内容（2026-11-06 实机 500 病例：
		// 曾在末条 user 前插 system 消息，qwen 模板拒绝「System message
		// must be at the beginning」——中间 system 直接炸引擎）。首条追加
		// 不改数组结构，模板合法；该轮前缀一次性冷 prefill 可接受。
		msgs := make([]ai.ChatMessage, len(messages))
		copy(msgs, messages)
		msgs[0].Content += "\n\n[本轮必须先调用 web_search 获取实时信息，再基于结果回答。]"
		messages = msgs
	}
	searches := 0
	for round := 0; round < toolLoopMaxRounds; round++ {
		useTools := searches < toolSearchMax && round < toolLoopMaxRounds-1
		roundTools := tools
		if !useTools {
			roundTools = nil // 末轮/配额尽：无工具强收敛正文
		}
		var sbContent, sbReasoning strings.Builder
		var firstToken time.Time
		chunks, cancelStream, err := a.clientRef().ChatStreamMessages(ctx, model, messages, opts, roundTools)
		if err != nil {
			return res, err
		}
		var toolCalls []ai.ChatToolCall
		for chunk := range chunks {
			if chunk.Error != "" {
				cancelStream()
				return res, errString(chunk.Error)
			}
			if chunk.Done {
				// 客户端在 Done 统一 flush 拼装完成的工具调用（流中分片不重复计）
				toolCalls = chunk.ToolCalls
				res.Usage = chunk.Usage
				break
			}
			if (chunk.Content != "" || chunk.Reasoning != "") && firstToken.IsZero() {
				firstToken = time.Now()
				if res.FirstToken.IsZero() {
					res.FirstToken = firstToken
				}
			}
			if chunk.Content != "" {
				sbContent.WriteString(chunk.Content)
				if onDelta != nil {
					onDelta(chunk.Content)
				}
			}
			if chunk.Reasoning != "" {
				sbReasoning.WriteString(chunk.Reasoning)
				if onReasoning != nil {
					onReasoning(chunk.Reasoning)
				}
			}
		}
		cancelStream()
		res.Reply += sbContent.String()
		res.Reasoning += sbReasoning.String()
		if len(toolCalls) == 0 || !useTools {
			return res, nil
		}
		// 执行工具并回填 assistant(tool_calls)+tool(result) 对
		assistantMsg := ai.ChatMessage{Role: "assistant", Content: sbContent.String(), ToolCalls: toolCalls}
		messages = append(messages, assistantMsg)
		for _, tc := range toolCalls {
			out := a.executeChatToolCall(&searches, tc)
			messages = append(messages, ai.ChatMessage{
				Role: "tool", ToolCallID: tc.ID, Name: tc.Function.Name, Content: out,
			})
		}
	}
	return res, nil
}

// executeChatToolCall 执行单次工具调用（当前仅 web_search），返回回填 tool
// 消息的文本面。搜索配额在调用前检查，超限回占位文案（模型侧自然收敛）。
func (a *App) executeChatToolCall(searches *int, tc ai.ChatToolCall) string {
	if tc.Function.Name != "web_search" {
		return "[不支持的工具，请直接基于已有知识回答]"
	}
	if *searches >= toolSearchMax {
		return "[搜索次数已达上限，请基于已获取的信息直接回答]"
	}
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil || strings.TrimSpace(args.Query) == "" {
		return "[工具参数解析失败，请直接基于已有知识回答]"
	}
	*searches++
	out, err := chatWebSearch(args.Query)
	if err != nil {
		return "[联网搜索暂不可用，请基于已有知识回答，并如实说明未获取到实时信息]"
	}
	switch out.State {
	case builtin.SearchStateOK, builtin.SearchStateDegraded:
		return out.Text + "\n" + searchDateAnchor(time.Now())
	default:
		return "[各搜索引擎均未返回相关结果，请基于已有知识回答，并如实说明]"
	}
}

// errString 轻量 error 包装（循环内 chunk.Error → error 返回）。
type strErr string

func (e strErr) Error() string { return string(e) }

func errString(s string) error {
	if s == "" {
		return nil
	}
	return strErr(s)
}
