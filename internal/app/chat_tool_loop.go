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
	"log/slog"
	"regexp"
	"strconv"
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
	Reply        string
	Reasoning    string
	Usage        *ai.ChatUsage
	FirstToken   time.Time
	Searches     int
	SearchRounds int
	SearchRuns   []chatSearchTrace
}

// chatToolLoopEvent is the small progress contract shared by the plain stream
// and the existing search loop. It deliberately carries no page text.
type chatToolLoopEvent struct {
	Phase          string
	Round          int
	Search         int
	Query          string
	State          builtin.SearchState
	DegradedReason string
	Sources        int
	ElapsedMS      int64
}

type chatSearchTrace struct {
	Query          string
	State          builtin.SearchState
	DegradedReason string
	Sources        int
	ElapsedMS      int64
	References     []chatSearchReference
}

// chatSearchReference is the compact, persisted provenance shown by the chat
// search card. Snippets are capped in the adapter and never hold full pages.
type chatSearchReference struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Source  string `json:"source,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

func notifyToolLoop(onEvent func(chatToolLoopEvent), ev chatToolLoopEvent) {
	if onEvent != nil {
		onEvent(ev)
	}
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
	onEvent func(chatToolLoopEvent),
	onDelta, onReasoning func(string),
) (res toolLoopResult, retErr error) {
	toolDef, ok := webSearchToolDef()
	if !ok {
		// 工具本体未注册（理论不可达）：降级为无工具直答，不失败
		res.Reasoning = ""
		notifyToolLoop(onEvent, chatToolLoopEvent{Phase: "fallback", Round: 1})
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
	defer func() { res.Searches = searches }()
	for round := 0; round < toolLoopMaxRounds; round++ {
		res.SearchRounds = round + 1
		notifyToolLoop(onEvent, chatToolLoopEvent{Phase: "thinking", Round: round + 1, Search: searches})
		useTools := searches < toolSearchMax && round < toolLoopMaxRounds-1
		roundTools := tools
		if !useTools {
			roundTools = nil // 末轮/配额尽：无工具强收敛正文
			// 强收敛指令（实机病例：剥工具后模型不会「放弃搜索直接答」，反而
			// 以文本形态继续吐 <tool_call>——145 字全标记剥完即空回复）。system
			// 追加被本地 qwen3.5 无视（19:58 复跑实证），改 user 角色 nudge——
			// 本地对用户轮指令服从性远高于 system（社区通识）。中间 system 会
			// 炸 qwen 模板（500 病例），user 轮模板合法。此消息仅循环内部，
			// 不进落库/提示历史。
			msgs := make([]ai.ChatMessage, 0, len(messages)+1)
			msgs = append(msgs, messages...)
			msgs = append(msgs, ai.ChatMessage{Role: "user", Content: "（搜索到此为止。请立刻基于上方已获取的搜索结果与你的知识给出最终回答；不要再调用工具，也不要输出任何工具调用格式的文本。）"})
			messages = msgs
		}
		var sbContent, sbReasoning strings.Builder
		var firstToken time.Time
		// 正文每轮先完整缓冲，解析 / 过滤后才允许外发。工具调用可能以原生
		// tool_calls 或正文标记出现，只有流结束后才能确定本轮是否要执行工具；
		// 提前直发会把 raw <tool_call> 同时写进 delta、done 和数据库。
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
			}
			if chunk.Reasoning != "" {
				sbReasoning.WriteString(chunk.Reasoning)
				if onReasoning != nil {
					onReasoning(chunk.Reasoning)
				}
			}
		}
		cancelStream()
		slog.Info("[whisper] 工具循环轮次", "round", round, "nativeCalls", len(toolCalls), "contentLen", len([]rune(sbContent.String())), "searches", searches)
		content := sbContent.String()
		if useTools {
			// 修复解析：原生 toolCalls 之外，正文里的文本形态工具调用块提取
			// 执行；标记从正文剥离，剩余散文补发外流。
			markupCalls, cleaned := extractMarkupToolCalls(content)
			content = cleaned
			toolCalls = append(toolCalls, markupCalls...)
			if len(toolCalls) == 0 {
				// 模型在仍带工具的轮次直接作答也是有效终态：必须交付缓冲正文，
				// 不能只写 res.Reply 后返回（流式调用方只消费 onDelta）。
				if strings.TrimSpace(content) == "" {
					return res, errString("模型返回为空，未生成可交付正文")
				}
				notifyToolLoop(onEvent, chatToolLoopEvent{Phase: "finalizing", Round: round + 1})
				if onDelta != nil {
					onDelta(content)
				}
				res.Reply += content
				res.Reasoning += sbReasoning.String()
				return res, nil
			}
		} else {
			// 末轮已经没有工具，但模型仍可能吐文本工具调用。先完整过滤，再让
			// 同一份 canonical 正文进入 delta、done 与落库。
			content = stripToolCallMarkup(content)
			if strings.TrimSpace(content) == "" {
				return res, errString("模型只返回了工具调用标记，未生成可交付正文")
			}
			notifyToolLoop(onEvent, chatToolLoopEvent{Phase: "finalizing", Round: round + 1})
			if onDelta != nil {
				onDelta(content)
			}
		}
		if useTools && content != "" && onDelta != nil {
			onDelta(content)
		}
		res.Reply += content
		res.Reasoning += sbReasoning.String()
		if !useTools {
			return res, nil
		}
		// 执行工具并回填 assistant(tool_calls)+tool(result) 对
		assistantMsg := ai.ChatMessage{Role: "assistant", Content: content, ToolCalls: toolCalls}
		messages = append(messages, assistantMsg)
		for _, tc := range toolCalls {
			out, run := a.executeChatToolCall(&searches, tc, onEvent)
			if run != nil {
				res.SearchRuns = append(res.SearchRuns, *run)
			}
			messages = append(messages, ai.ChatMessage{
				Role: "tool", ToolCallID: tc.ID, Name: tc.Function.Name, Content: out,
			})
		}
	}
	return res, nil
}

// toolCallMarkupRE 文本形态工具调用块（qwen Hermes 风格）：模型偶发把调用
// 写成正文 XML 而非原生 tool_calls（实机病例：热门电影查询）。
var toolCallMarkupRE = regexp.MustCompile(`(?s)<tool_call>\s*<function=([A-Za-z0-9_]+)>([\s\S]*?)</function>\s*</tool_call>`)

var markupParamRE = regexp.MustCompile(`<parameter=([A-Za-z0-9_]+)>\s*([\s\S]*?)\s*</parameter>`)

var toolCallJSONBlockRE = regexp.MustCompile(`(?s)<tool_call>\s*([\s\S]*?)\s*</tool_call>`)

var toolCallFenceRE = regexp.MustCompile("(?s)```tool_call[^\\n]*\\r?\\n([\\s\\S]*?)```")

// extractMarkupToolCalls 从正文提取文本形态工具调用块并返回「剥离标记后的
// 散文」。参数统一打包成 {name: value} JSON（arguments 字符串形态，与原生
// 调用同构进 executeChatToolCall）；解析不了整块丢弃不外漏。
func extractMarkupToolCalls(content string) ([]ai.ChatToolCall, string) {
	if !strings.Contains(content, "<tool_call>") && !strings.Contains(content, "```tool_call") {
		return nil, content
	}
	var calls []ai.ChatToolCall
	seq := 0
	cleaned := toolCallMarkupRE.ReplaceAllStringFunc(content, func(block string) string {
		m := toolCallMarkupRE.FindStringSubmatch(block)
		if m == nil {
			return ""
		}
		name := m[1]
		params := map[string]string{}
		for _, pm := range markupParamRE.FindAllStringSubmatch(m[2], -1) {
			params[pm[1]] = strings.TrimSpace(pm[2])
		}
		args, err := json.Marshal(params)
		if err != nil {
			return ""
		}
		seq++
		calls = append(calls, ai.ChatToolCall{
			ID:       "markup_" + strconv.Itoa(seq),
			Type:     "function",
			Function: ai.ChatToolFunction{Name: name, Arguments: string(args)},
		})
		return ""
	})
	cleaned = toolCallJSONBlockRE.ReplaceAllStringFunc(cleaned, func(block string) string {
		m := toolCallJSONBlockRE.FindStringSubmatch(block)
		if m == nil {
			return ""
		}
		calls = append(calls, decodeMarkupJSONCalls(m[1], &seq)...)
		return ""
	})
	cleaned = toolCallFenceRE.ReplaceAllStringFunc(cleaned, func(block string) string {
		m := toolCallFenceRE.FindStringSubmatch(block)
		if m == nil {
			return ""
		}
		calls = append(calls, decodeMarkupJSONCalls(m[1], &seq)...)
		return ""
	})
	return calls, strings.TrimSpace(cleaned)
}

func decodeMarkupJSONCalls(raw string, seq *int) []ai.ChatToolCall {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil
	}
	items := []interface{}{decoded}
	if arr, ok := decoded.([]interface{}); ok {
		items = arr
	}
	var calls []ai.ChatToolCall
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		nameValue := obj["name"]
		argsValue := obj["arguments"]
		if argsValue == nil {
			argsValue = obj["parameters"]
		}
		if fn, ok := obj["function"].(map[string]interface{}); ok {
			nameValue = fn["name"]
			argsValue = fn["arguments"]
		}
		name, _ := nameValue.(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		args := "{}"
		switch v := argsValue.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				args = v
			}
		case nil:
		default:
			if b, err := json.Marshal(v); err == nil {
				args = string(b)
			}
		}
		*seq++
		calls = append(calls, ai.ChatToolCall{
			ID:       "markup_json_" + strconv.Itoa(*seq),
			Type:     "function",
			Function: ai.ChatToolFunction{Name: name, Arguments: args},
		})
	}
	return calls
}

// executeChatToolCall 执行单次工具调用（当前仅 web_search），返回回填 tool
// 消息的文本面。搜索配额在调用前检查，超限回占位文案（模型侧自然收敛）。
func (a *App) executeChatToolCall(
	searches *int,
	tc ai.ChatToolCall,
	onEvent func(chatToolLoopEvent),
) (string, *chatSearchTrace) {
	if tc.Function.Name != "web_search" {
		return "[不支持的工具，请直接基于已有知识回答]", nil
	}
	if *searches >= toolSearchMax {
		return "[搜索次数已达上限，请基于已获取的信息直接回答]", nil
	}
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil || strings.TrimSpace(args.Query) == "" {
		return "[工具参数解析失败，请直接基于已有知识回答]", nil
	}
	*searches++
	query := strings.TrimSpace(args.Query)
	notifyToolLoop(onEvent, chatToolLoopEvent{Phase: "searching", Search: *searches, Query: query})
	startedAt := time.Now()
	out, err := chatWebSearch(query)
	run := &chatSearchTrace{Query: query, ElapsedMS: time.Since(startedAt).Milliseconds()}
	if err != nil {
		run.State = builtin.SearchStateAllFailed
		notifyToolLoop(onEvent, chatToolLoopEvent{
			Phase: "search_done", Search: *searches, Query: query,
			State: run.State, ElapsedMS: run.ElapsedMS,
		})
		return "[联网搜索暂不可用，请基于已有知识回答，并如实说明未获取到实时信息]", run
	}
	switch out.State {
	case builtin.SearchStateOK, builtin.SearchStateDegraded:
		run.State = out.State
		run.DegradedReason = out.DegradedReason
		run.Sources = out.Sources
		run.References = out.References
		notifyToolLoop(onEvent, chatToolLoopEvent{
			Phase: "search_done", Search: *searches, Query: query,
			State: run.State, DegradedReason: run.DegradedReason,
			Sources: run.Sources, ElapsedMS: run.ElapsedMS,
		})
		return out.Text + "\n" + searchDateAnchor(time.Now()), run
	default:
		if out.State == "" {
			run.State = builtin.SearchStateAllFailed
		} else {
			run.State = out.State
		}
		run.DegradedReason = out.DegradedReason
		run.Sources = out.Sources
		run.References = out.References
		notifyToolLoop(onEvent, chatToolLoopEvent{
			Phase: "search_done", Search: *searches, Query: query,
			State: run.State, DegradedReason: run.DegradedReason,
			Sources: run.Sources, ElapsedMS: run.ElapsedMS,
		})
		return "[各搜索引擎均未返回相关结果，请基于已有知识回答，并如实说明]", run
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

// markupDeltaFilter 流式外发过滤器（B7 残刀三，非工具轮防线）：模型凭 qwen
// 工具训练先验会在「允许联网=关」时仍以文本形态吐 <tool_call> 标记（实机
// 19:14 两病例），直发面需要边流边吞标记段。规则：检出 "<tool_call" 即吞到
// "</tool_call>" 之后（未闭合则其后全吞——标记模式的尾部都是垃圾）；尾部
// 疑似标记前缀悬挂不放行，等下一 delta 定夺；其余 '<' 照常放行。
type markupDeltaFilter struct {
	pending string
}

const markupOpenToken = "<tool_call"

func (f *markupDeltaFilter) feed(delta string) string {
	f.pending += delta
	var out strings.Builder
	for {
		i := strings.Index(f.pending, markupOpenToken)
		if i < 0 {
			keep := 0
			max := len(markupOpenToken) - 1
			if max > len(f.pending) {
				max = len(f.pending)
			}
			for k := max; k >= 1; k-- {
				if strings.HasSuffix(f.pending, markupOpenToken[:k]) {
					keep = k
					break
				}
			}
			out.WriteString(f.pending[:len(f.pending)-keep])
			f.pending = f.pending[len(f.pending)-keep:]
			return out.String()
		}
		out.WriteString(f.pending[:i])
		j := strings.Index(f.pending[i:], "</tool_call>")
		if j < 0 {
			f.pending = ""
			return out.String()
		}
		f.pending = f.pending[i+j+len("</tool_call>"):]
	}
}

// flush 流结束时放行悬挂尾巴（残留未闭合标记按垃圾丢弃：以 "<tool" 类起始
// 且未闭合的尾巴极小概率是正常散文，吞掉比漏给用户便宜）。
func (f *markupDeltaFilter) flush() string {
	rest := f.pending
	f.pending = ""
	trimmed := strings.TrimSpace(rest)
	if strings.HasPrefix(trimmed, markupOpenToken) ||
		strings.Contains(trimmed, markupOpenToken) ||
		(trimmed != "" && strings.HasPrefix(markupOpenToken, trimmed)) {
		return ""
	}
	return rest
}

// stripToolCallMarkup 非流式面的最终净化（落库/返回值/TTS 面）。
func stripToolCallMarkup(s string) string {
	var f markupDeltaFilter
	cleaned := f.feed(s) + f.flush()
	_, cleaned = extractMarkupToolCalls(cleaned)
	return strings.TrimSpace(cleaned)
}
