package app

// wx_agent.go — 微信智能体 v1（LLM 工具调用派发，2026-09）。
//
// 架构（用户拍板）：微信消息 → LLM 工具调用派发（模型自己决定调哪个板块能力）
// → 本地执行（复用 intent_router.go 既有 exec 层，零改动适配）→ 结果回微信。
// 关键词意图路由（routeIntent）降级为兜底——模型有了工具可调，就不再需要
// 「重新整理后发给我」类的嘴上幻觉：要执行动作必须调工具，拿真实结果说话。
//
// 复用口径：工具参数 JSON → 构造 intent.Intent{Action,Target,Text} → 直接调
// execNavigate / execGenerateImage / execStatus / execReminder /
// execReadScreen / execSendLatestFile。exec 层签名零改动；CardPath 产物交回
// 调用侧（whisper_state.go 回调）经既有 SendFileCard 链回推。
//
// 循环消费 API：ai.Client.ChatStream（internal/ai 公开入口，全引擎共用）——
// SSEChunk.Content 为增量正文；finish_reason=tool_calls 或 [DONE] 时 Done 块
// 携带按 index 拼装完整的 ToolCalls（client.go parseStreamEvents），无需自拼。
//
// 锁语义（与 whisperChat 对齐）：助手名注入与 PreLLMTurn 都在 LockTurn 持锁
// 窗口内（whisper_handler.go whisperChat 同款）；agent 循环本身不持回合锁
// （多轮 LLM 最长 60s，锁内会阻塞同人格 GUI 轻语），成功后重入锁做收尾
// （WM/FinalizeTurn），失败则回滚 PreLLMTurn 的状态推进（降级路径 whisperChat
// 会重走完整回合，不回滚就一消息双计数）。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/intent"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// ─── 参数 ────────────────────────────────────────────────────

const (
	// wxAgentMaxRounds 工具循环轮数上限（每轮一次 LLM 流式调用）。
	wxAgentMaxRounds = 4
	// wxAgentCallTimeout 单次微信回调内 agent 循环总时长上限。clawbot 的
	// handle 对 chatFn 无外层超时（pollLoop 串行消费消息），必须自带上限，
	// 否则一次卡死会拖住该助手后续所有消息。
	wxAgentCallTimeout = 60 * time.Second
	// wxAgentToolCap 引擎目录能力位：模型目录 Caps 含 "tools" 才启用 agent
	// （glm_catalog.json / model_catalog.json 有值，其余引擎常空——查不到=
	// 不含，宁缺勿滥，降级老路由）。
	wxAgentToolCap = "tools"
)

// ─── 工具 schema 表（6 个能力）──────────────────────────────
//
// name 英文 snake_case（OpenAI 工具名规范），description 中文。参数即 exec
// 适配所需的全部信息。

var wxAgentTools = []ai.ChatToolSchema{
	{
		Type: "function",
		Function: ai.ChatToolFunctionSpec{
			Name:        "navigate_board",
			Description: "打开或切换桌面端的板块（首页/轻语/小说/绘梦/办公/造价库/编程/记忆中枢/模型中心/角色库/设置/微信助手）。用户想打开、进入、切换某个板块时调用。",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"board": {"type": "string", "description": "板块 id，优先用英文 id：home chat novel imagegen gaea cost code memoryhub modelcenter characterlib settings weixin；也可以传中文板块名（如「绘梦」）"}
				},
				"required": ["board"]
			}`),
		},
	},
	{
		Type: "function",
		Function: ai.ChatToolFunctionSpec{
			Name:        "generate_image",
			Description: "生成一张新图片（绘梦板块能力，默认模型与尺寸，同步等待生成完成后返回落盘路径）。用户想「画一张…」时调用。",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"prompt": {"type": "string", "description": "画面描述，保留用户原话的关键要素"}
				},
				"required": ["prompt"]
			}`),
		},
	},
	{
		Type: "function",
		Function: ai.ChatToolFunctionSpec{
			Name:        "create_reminder",
			Description: "设一个到点经微信回推的提醒（离线代办）。用户说「提醒我…」时调用。",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"task_text": {"type": "string", "description": "提醒事项正文，如「喝水」「起来走动」"},
					"when_raw": {"type": "string", "description": "触发时间的中文表达，原样保留用户的说法，如「30分钟后」「明天早上9点」「下午3点半」"}
				},
				"required": ["task_text", "when_raw"]
			}`),
		},
	},
	{
		Type: "function",
		Function: ai.ChatToolFunctionSpec{
			Name:        "send_latest_file",
			Description: "把最新一份产物文件（报告/表格/文档等）经微信文件卡发送给用户。只能发送已有文件，不能修改文档内容——用户要求「改完再发」时调用后也会得到如实的能力边界说明。",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	},
	{
		Type: "function",
		Function: ai.ChatToolFunctionSpec{
			Name:        "query_status",
			Description: "查询当前模型引擎/图像后端的运行状态摘要。用户问「现在用的什么模型」时调用。",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	},
	{
		Type: "function",
		Function: ai.ChatToolFunctionSpec{
			Name:        "read_screen",
			Description: "截取电脑屏幕并识别屏幕上的文字（OCR），返回文本内容。用户想「看看屏幕上写了什么」时调用。",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	},
}

// wxAgentInstruction 智能体行为指令（拼在人格 SystemPrompt 之后）。
const wxAgentInstruction = `【微信智能体工作方式】
你正在微信里和用户对话，可以调用工具代替用户在这台电脑上执行操作。铁律：
1. 用户要你「做某事」（打开板块、画图、改图、设提醒、发文件、查状态、读屏幕）时，必须先调用对应工具，用工具的真实结果回复；绝不假装已执行。
2. 没有能完成该任务的工具时，如实说明做不到并给替代建议；绝不声称「已发送/已整理好/已保存」。
3. 工具结果指出参数有误时，在同一轮修正参数重试；连续失败就如实告知用户。
4. 回复用中文，口语化、简洁（微信场景，一般不超过三句话），不要输出Markdown标题或代码块。`

// wxAgentFallbackSystemPrompt SystemPrompt 获取失败/为空时的兜底指令串
// （不空跑：人格块缺席时至少带上行为铁律与助手身份）。
const wxAgentFallbackSystemPrompt = "你是用户的桌面 AI 助手 gaea，正通过微信与用户对话。\n\n" + wxAgentInstruction

// ─── 能力门 ──────────────────────────────────────────────────

// wxAgentAvailable 微信智能体能力门：App 就绪 + 非离线模式 + chat 路由解析出的
// 模型在目录中 Caps 含 "tools"。任一不满足返回 false（调用方降级 routeIntent
// 老路——快路径语义保留，零回归）。
func wxAgentAvailable(a *App) bool {
	if a == nil || a.core == nil || a.cfg == nil || a.client == nil || a.engineMgr == nil {
		return false
	}
	if a.GetOfflineMode() {
		return false // 全局离线模式：本地引擎目录通常无能力位，宁缺勿滥整体关闭
	}
	engine, model, _ := a.routeModel("chat")
	if engine == "" || model == "" {
		return false
	}
	e, ok := a.engineMgr.GetEngine(engine)
	if !ok {
		return false
	}
	return slices.Contains(wxModelCaps(e, model), wxAgentToolCap)
}

// wxModelCaps 查模型在引擎目录中的能力标记；模型不在目录中返回 nil
// （=不含 tools，宁缺勿滥）。ID 比较大小写不敏感（路由可能回传任意大小写）。
func wxModelCaps(e *modelengine.EngineConfig, model string) []string {
	if e == nil {
		return nil
	}
	for i := range e.Models {
		if strings.EqualFold(e.Models[i].ID, model) {
			return e.Models[i].Caps
		}
	}
	return nil
}

// ─── 工具循环 ────────────────────────────────────────────────

// runWxAgent 微信智能体主循环：messages = [system(人格+智能体指令), user(消息)]，
// 每轮 ChatStream；有 tool_calls → 构造 intent 调 exec 层 → 结果以 tool 消息
// 喂回模型继续；无 tool_calls → 文本即最终回复。轮数上限 wxAgentMaxRounds，
// 超限把已积累文本 +「（任务未完成，请重试）」返回。参数防御：JSON 解析失败/
// 字段缺失 → tool 结果回错误文本让模型同轮自纠，绝不 panic。
//
// 返回 cards：工具产物绝对路径（生成图片/改图产物/最新产物文件），调用侧逐张
// SendFileCard 回推；模型侧只拿到路径文本。
func runWxAgent(a *App, assistantID, systemPrompt, userMsg string) (reply string, cards []string, err error) {
	if a == nil || a.client == nil {
		return "", nil, errors.New("模型客户端未初始化")
	}
	engine, model, _ := a.routeModel("chat")
	if model == "" {
		return "", nil, errors.New("无可用聊天模型")
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, wxAgentCallTimeout)
	defer cancel()

	msgs := []ai.ChatMessage{
		{Role: "system", Content: systemPrompt + "\n\n" + wxAgentInstruction},
		{Role: "user", Content: userMsg},
	}

	var roundTexts []string // 每轮 assistant 正文（最终回复/超限兜底取最后非空段）
	for round := 0; round < wxAgentMaxRounds; round++ {
		req := &ai.ChatRequest{
			Model:       model,
			EngineID:    engine,
			Messages:    msgs,
			Tools:       wxAgentTools,
			MaxTokens:   4096,
			Temperature: 0.7, // 与 ChatSimple 客户端缺省一致
		}
		chunks, cerr := a.client.ChatStream(ctx, req)
		if cerr != nil {
			return "", cards, cerr
		}
		var text strings.Builder
		var toolCalls []ai.ChatToolCall
		var streamErr string
	consume:
		for {
			select {
			case <-ctx.Done():
				return "", cards, fmt.Errorf("微信智能体超时或已取消（%ds 上限）", int(wxAgentCallTimeout.Seconds()))
			case chunk, ok := <-chunks:
				if !ok {
					break consume
				}
				if chunk.Error != "" {
					streamErr = chunk.Error
					break consume
				}
				if chunk.Done {
					toolCalls = chunk.ToolCalls // finish_reason=tool_calls 时携带完整拼装结果
					break consume
				}
				text.WriteString(chunk.Content)
			}
		}
		if streamErr != "" {
			return "", cards, fmt.Errorf("%s", streamErr)
		}
		roundTexts = append(roundTexts, text.String())

		if len(toolCalls) == 0 {
			return wxLastNonEmpty(roundTexts), cards, nil
		}

		// assistant 消息（含 tool_calls）入栈，逐个执行并把结果以 tool 消息回喂
		msgs = append(msgs, ai.ChatMessage{Role: "assistant", Content: text.String(), ToolCalls: toolCalls})
		for _, tc := range toolCalls {
			result, card := wxAgentExecToolFn(a, assistantID, tc, userMsg)
			if card != "" {
				cards = append(cards, card)
			}
			msgs = append(msgs, ai.ChatMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
		}
	}
	// 轮数超限：把已积累文本 + 未完成标记返回（不返回 error——产物卡片照常回推）
	return wxLastNonEmpty(roundTexts) + "（任务未完成，请重试）", cards, nil
}

// wxLastNonEmpty 取最后一段非空文本（末轮只发工具调用无正文时回退历史正文）。
func wxLastNonEmpty(texts []string) string {
	for i := len(texts) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(texts[i]); s != "" {
			return s
		}
	}
	return ""
}

// wxAgentExecToolFn 工具执行 seam（测试替换；生产 = wxAgentExecTool）。
var wxAgentExecToolFn = func(a *App, assistantID string, call ai.ChatToolCall, userMsg string) (string, string) {
	return a.wxAgentExecTool(assistantID, call, userMsg)
}

// wxAgentExecTool 单个工具调用 → 构造 intent.Intent → 调对应 exec 执行函数
// （intent_router.go 既有执行层，零改动复用）。返回 (喂回模型的 tool 结果文本,
// 产物绝对路径)。
func (a *App) wxAgentExecTool(assistantID string, call ai.ChatToolCall, userMsg string) (string, string) {
	args, err := wxAgentToolArgs(call)
	if err != nil {
		return "工具参数解析失败：" + err.Error() + "。请重新调用并给出合法 JSON 参数。", ""
	}
	arg := func(key string) string { return wxAgentArgString(args, key) }
	missing := func(key string) (string, string) {
		return "缺少参数 " + key + "。请重新调用并补上该参数。", ""
	}

	switch call.Function.Name {
	case "navigate_board":
		board := arg("board")
		if board == "" {
			return missing("board")
		}
		it := wxAgentIntentNavigate(a.wxAgentResolveBoard(board), userMsg)
		reply, ok := a.execNavigate(it)
		if !ok {
			return "打开失败：没有找到板块「" + board + "」。可用板块 id：home chat novel imagegen gaea cost code memoryhub modelcenter characterlib settings weixin。", ""
		}
		return reply, ""
	case "generate_image":
		prompt := arg("prompt")
		if prompt == "" {
			return missing("prompt")
		}
		reply, ok, card := a.execGenerateImage(wxAgentIntentGenerateImage(prompt, userMsg))
		if !ok {
			return "生图能力暂不可用（媒体域未装配）。", ""
		}
		if card != "" {
			return reply + "\n已生成产物：" + card, card
		}
		return reply, ""
	case "create_reminder":
		task, when := arg("task_text"), arg("when_raw")
		if task == "" && when == "" {
			return "缺少参数 task_text 与 when_raw。请重新调用，例如 {\"task_text\":\"喝水\",\"when_raw\":\"30分钟后\"}。", ""
		}
		reply, ok := a.execReminder(wxAgentIntentReminder(task, when, userMsg))
		if !ok {
			return "提醒能力暂不可用。", ""
		}
		return reply, ""
	case "send_latest_file":
		reply, ok, card := a.execSendLatestFile(wxAgentIntentSendLatestFile(userMsg))
		if !ok {
			return "发送文件能力暂不可用。", ""
		}
		if card != "" {
			return reply + "\n已找到产物：" + card, card
		}
		return reply, ""
	case "query_status":
		reply, ok := a.execStatus(wxAgentIntentStatus(userMsg))
		if !ok {
			return "状态查询暂不可用。", ""
		}
		return reply, ""
	case "read_screen":
		reply, ok := a.execReadScreen(wxAgentIntentReadScreen(userMsg))
		if !ok {
			return "读屏能力暂不可用。", ""
		}
		return reply, ""
	case "":
		return "工具名为空。请从可用工具中选择一个调用。", ""
	default:
		return "未知工具：" + call.Function.Name + "。请从可用工具中选择，或如实告知用户做不到。", ""
	}
}

// ─── intent 构造（纯函数，测试直测）──────────────────────────

// wxAgentIntentNavigate 导航：Target=板块 id（wxAgentResolveBoard 已归一）。
func wxAgentIntentNavigate(board, text string) *intent.Intent {
	return &intent.Intent{Action: intent.ActionNavigate, Target: board, Text: text}
}

// wxAgentIntentGenerateImage 生图：Target=画面描述（execGenerateImage 消费口径）。
func wxAgentIntentGenerateImage(prompt, text string) *intent.Intent {
	return &intent.Intent{Action: intent.ActionGenerateImage, Target: prompt, Text: text}
}

// wxAgentIntentReminder 提醒：execReminder 只消费 it.Text（parseReminderWhen
// 解析中文时间 + stripReminderText 剥出事项正文）——把 when_raw 拼在句首还原
// 自然语序（「30分钟后 喝水」），两个正则都能命中。
func wxAgentIntentReminder(taskText, whenRaw, text string) *intent.Intent {
	return &intent.Intent{
		Action: intent.ActionReminder,
		Text:   strings.TrimSpace(whenRaw + " " + taskText),
	}
}

// wxAgentIntentSendLatestFile 产物推送：v4.41.2 诚实护栏保留——原文命中
// 「整理/修改…后发给我」复合请求时保留 Target=modify_and_send（exec 层给
// 诚实能力答复，绝不把未修改的旧产物冒充结果发出去）。
func wxAgentIntentSendLatestFile(text string) *intent.Intent {
	target := ""
	if it := intent.Parse(text); it != nil && it.Action == intent.ActionSendLatestFile {
		target = it.Target
	}
	return &intent.Intent{Action: intent.ActionSendLatestFile, Target: target, Text: text}
}

// wxAgentIntentStatus 状态查询：Target=model（execStatus 忽略 Target，保持与
// routeIntent 路径同构）。
func wxAgentIntentStatus(text string) *intent.Intent {
	return &intent.Intent{Action: intent.ActionStatus, Target: "model", Text: text}
}

// wxAgentIntentReadScreen 读屏：Target=screen（captureScreenTarget 缺省语义 =
// 整个虚拟屏）。
func wxAgentIntentReadScreen(text string) *intent.Intent {
	return &intent.Intent{Action: intent.ActionReadScreen, Target: "screen", Text: text}
}

// ─── 参数解析（防御式，绝不 panic）──────────────────────────

// wxAgentToolArgs 解析工具调用参数 JSON（空参数视为空对象）。
func wxAgentToolArgs(call ai.ChatToolCall) (map[string]any, error) {
	raw := strings.TrimSpace(call.Function.Arguments)
	if raw == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// wxAgentArgString 取字符串参数（缺失/非字符串一律空串 → 由调用侧按缺参自纠）。
func wxAgentArgString(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

// ─── 板块名归一 ─────────────────────────────────────────────

// wxAgentResolveBoard 工具 board 参数 → 板块 id：参数已是 manifest id 直用；
// 否则借 intent.Parse 的板块别名表（「绘梦」→ imagegen）解析一次——模型传
// 中文名也能打开（execNavigate 按 boardLabel(id) 校验，直接传中文必落空）。
func (a *App) wxAgentResolveBoard(board string) string {
	board = strings.TrimSpace(board)
	if board == "" || a.boardLabel(board) != "" {
		return board
	}
	if it := intent.Parse("打开" + board); it != nil && it.Action == intent.ActionNavigate {
		return it.Target
	}
	return board
}

// ─── 回合封装（whisperState 侧）─────────────────────────────

// runWxAgentTurn 微信回调侧的 agent 回合封装：回合锁内取 PreLLMTurn 人格
// SystemPrompt（whisperChat 同款锁语义）→ 工具循环 → 成功收尾回合（WM/
// FinalizeTurn/追踪/异步记忆），失败回滚状态推进并返回 error（调用方降级
// routeIntent → whisperChatAsAssistant 老路）。
func (w *whisperState) runWxAgentTurn(assistantID, personalityID, assistantName, userMsg string) (string, []string, error) {
	orch := w.getOrCreateOrch(personalityID)
	if orch == nil {
		return "", nil, errors.New("会话编排器未就绪")
	}

	// 助手名注入与 PreLLMTurn 都必须在持锁窗口内（同人格多助手共享
	// orchestrator，锁外直写互相覆盖且与 PreLLMTurn 锁内读构成数据竞争）。
	orch.LockTurn()
	if assistantName != "" {
		orch.AssistantName = assistantName
	}
	prevState := whisper.CloneFullState(orch.State)
	pre := orch.PreLLMTurn(userMsg)
	orch.UnlockTurn()

	systemPrompt := pre.SystemPrompt
	if strings.TrimSpace(systemPrompt) == "" {
		systemPrompt = wxAgentFallbackSystemPrompt
	}

	reply, cards, err := runWxAgent(w.app, assistantID, systemPrompt, userMsg)
	if err != nil {
		// 回滚 PreLLMTurn 的状态推进：降级路径（routeIntent / whisperChat）
		// 会重走完整回合——不回滚则一条消息双计数（TotalTurns/情绪双步进）。
		// （PreLLMTurn 在 State 之外的副作用——rhythm 计数、习惯库 upsert——
		// 量级可忽略，不做逆向补偿。）
		orch.LockTurn()
		orch.State = prevState
		orch.UnlockTurn()
		slog.Warn("[wx-agent] 智能体回合失败，将降级老路由", "assistant", assistantID, "err", err)
		return "", nil, err
	}

	// 节奏标记归一（v4.9.1 同款）：[SPLIT] 是内部协议，微信出口必须归一为换行。
	if strings.Contains(reply, whisper.SplitMarker) {
		reply = strings.Join(whisper.SplitOnMarker(reply), "\n")
	}

	w.finalizeWxAgentTurn(orch, pre, userMsg, reply)
	return reply, cards, nil
}

// finalizeWxAgentTurn agent 成功后的回合收尾（镜像 whisperChat 尾段）：WM 推入
// + FinalizeTurn 状态机落定 + 轮次追踪落库 + 异步记忆写入/状态持久化——让
// agent 回合与轻语回合一样进入记忆（「画一只猫」→「再画一只狗」上下文连续）。
func (w *whisperState) finalizeWxAgentTurn(orch *whisper.Orchestrator, pre whisper.PreLLMResult, userMsg, reply string) {
	orch.LockTurn()
	orch.WM.Push(orch.SessionID, whisper.Exchange{
		TurnIndex: orch.State.Counters.TotalTurns, UserText: userMsg, AssistantText: reply,
	})
	l1Snap := orch.State.Relationship
	l2Snap := orch.State.Emotion
	turns := orch.State.Counters.TotalTurns
	whisper.FinalizeTurn(orch, whisper.PostTurnContext{
		SessionID: orch.SessionID, TurnIndex: turns,
		UserMsg: userMsg, AssistantText: reply,
		Event: pre.Event, AdultMode: orch.AdultMode,
	})
	orch.UnlockTurn()

	if err := repos.AppendTurnTraceToDB(orch.DataRoot, orch.SessionID, pre.Trace); err != nil {
		slog.Error("[wx-agent] 轮次追踪落库失败", "sessionID", orch.SessionID, "error", err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("whisper: memory write goroutine panic recovered", "panic", r)
				w.recordMemoryWriteError(orch.SessionID, "panic", fmt.Errorf("memory write panic: %v", r))
			}
		}()
		whisper.EnqueueMemoryWrite(w, whisper.MemoryWritePayload{
			SessionID: orch.SessionID, TurnIndex: turns,
			UserMsg: userMsg, AssistantText: reply,
			L1: l1Snap, L2: l2Snap,
			FactStore: orch.FactStore, TotalTurns: turns, KG: orch.KG,
			EpisodicStore:      orch.EpisodicStore,
			RecentExchanges:    buildRecentExchanges(orch),
			TemporalAnchorSink: func(a whisper.TemporalAnchor) { orch.AddTemporalAnchor(a) },
			AdultMode:          orch.AdultMode,
		}, w.recordMemoryWriteError)
	}()

	go w.persistStateAsync(orch)
}
