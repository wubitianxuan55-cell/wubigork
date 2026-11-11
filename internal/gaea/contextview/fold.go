package contextview

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// 常量：上下文分类标签（浏览器/归档重建用）。
const (
	catSystem    = "system"
	catTools     = "tools"
	catUser      = "user"
	catInject    = "inject"
	catSkill     = "skill"
	catAssistant = "assistant"
	catTool      = "tool"
)

const (
	// refContextPrefix 是 ResolveRefs 注入上下文时拼进 user 消息的前缀
	// （control.Submit）。折叠时把前缀块拆到 inject 分类，保持与效果图口径一致。
	refContextPrefix = "Referenced context:\n\n"
	maxBrief         = 160
	maxNodePreview   = 300
)

// 浏览器视图上限（对齐 dsh-context DEFAULT_BOUNDS 的诚实子集）。上限只收
// 「浏览器视图」：token 合计与差分走增量计数器（见 snapshotSurface），视图
// 截断不改变构成口径；NodesDropped/NodesFloor 向前端标注边界。
const (
	maxNodes    = 2000 // 活跃 surface 节点（dsh maxNodes 同值）
	maxArchive  = 400  // 压缩归档（dsh maxArchiveNodes 同值）
	maxEvents   = 400  // 上下文事件（dsh maxEvents 同值）
	maxFiles    = 400  // 文件活动（dsh maxFileOps 同值）
	maxCommands = 400  // 命令执行（dsh maxCommands 同值）
	maxCmdPrevw = 240  // 命令行预览截断（全文在日志 tool_dispatch）
)

// FoldTimeline 把会话日志条目序列折叠为逐请求上下文快照。纯函数：同输入
// 必同输出，retention<=0 时使用 DefaultRetention。
func FoldTimeline(entries []session.LogEntry, window int64, retention int) ContextTimeline {
	if retention <= 0 {
		retention = DefaultRetention
	}
	f := &folding{window: window}
	for _, e := range entries {
		f.apply(e)
	}
	f.killOpenCmds(0) // 收尾：未闭合命令（回合被硬杀无 turn_done）按 killed 落地
	f.trimBrowserWindow()
	tl := ContextTimeline{
		Ok:           true,
		Window:       f.window,
		Current:      f.current(),
		Stats:        f.stats,
		Events:       f.events,
		Nodes:        f.nodes,
		Archive:      f.archive,
		Files:        f.files,
		NodesDropped: f.nodesDropped,
		NodesFloor:   f.nodesFloor,
	}
	if len(f.commands) > 0 {
		tl.Commands = f.commands
		tl.CommandStats = f.commandStats()
	}
	if n := len(f.requests); n > retention {
		tl.Requests = append([]RequestRecord(nil), f.requests[n-retention:]...)
	} else {
		tl.Requests = append([]RequestRecord(nil), f.requests...)
	}
	f.timingClose()
	if f.timing.nonZero() {
		t := f.timing
		t.Tools = f.timingToolsRanked()
		tl.Timing = &t
	}
	if f.rateIn > 0 || f.rateOut > 0 || f.rateCacheHit > 0 {
		tl.Rate = &CostRate{
			InputPer1M:    f.rateIn,
			OutputPer1M:   f.rateOut,
			CacheHitPer1M: f.rateCacheHit,
			Currency:      f.currency,
		}
	}
	// Go 的 nil 切片会序列化成 JSON null，前端按数组消费（.length / for-of）
	// 会整页崩——空会话恰好四条全 nil。统一兜底为空切片。
	if tl.Requests == nil {
		tl.Requests = []RequestRecord{}
	}
	if tl.Events == nil {
		tl.Events = []ContextEvent{}
	}
	if tl.Nodes == nil {
		tl.Nodes = []SurfaceNode{}
	}
	if tl.Archive == nil {
		tl.Archive = []SurfaceNode{}
	}
	if tl.Files == nil {
		tl.Files = []FileActivity{}
	}
	if tl.Commands == nil {
		tl.Commands = []CommandRecord{}
	}
	return tl
}

// folding 是折叠器的单遍状态。
type folding struct {
	window int64
	turn   int
	step   int

	systemTok    int64  // 最新 request_header 的 system prompt 估算
	toolsTok     int64  // 最新 request_header 的 tool schema 估算
	lastSysNode  string // 已入 nodes 的 system 预览（变化才新增节点）
	lastToolsKey string // 已入 nodes 的工具名集合键（变化才新增节点）

	userTok      int64 // 活着的 user 节点 token 合计
	injectTok    int64 // 活着的 inject 节点 token 合计
	skillTok     int64 // 活着的 skill 节点 token 合计（run_skill 结果）
	assistantTok int64 // 活着的 assistant 节点 token 合计（含运行期工具调用参数重放账）
	toolTok      int64 // 活着的 tool 节点 token 合计

	// catItems 是各活分类的节点条数（user/inject/skill/assistant/tool）。
	// 与上面 token 计数器同为增量维护：浏览器视图截断（trimBrowserWindow）
	// 只动节点列表，不动这里——差分快照的口径不受视图上限影响。
	catItems map[string]int64

	nodes   []SurfaceNode
	archive []SurfaceNode

	// ─── 对比上一步（per-request surface 快照）状态：与上面各字段同趟维护 ───
	prevSurface *surfaceSnapshot // 上一次请求快照点（nil=尚无基线，首个请求 First）
	compactMark int64            // 最近一次压缩事件 seq（快照间变化=Approx 近似）

	requests []RequestRecord
	events   []ContextEvent
	stats    Stats

	pending *RequestRecord // request_header 已开、usage 未到的请求

	lastUser      string
	lastIn        []string
	lastAssistant string
	lastToolCall  string
	// brief 跳转锚点（2.5d）：brief 文本来源事件的 seq（对应 SurfaceNode）。
	// user/assistant 取消息事件自身（节点同 seq）；tool 交换在结果到达时锚到
	// 结果节点（结果行带来源 chip+全文，比派发事件更有跳转价值），结果未到前
	// 保持 assistant 锚（调用文本在 assistant 消息里）。0=暂无锚点。
	lastUserSeq      int64
	lastAssistantSeq int64
	lastToolCallSeq  int64
	files            []FileActivity
	fileIdx          map[string]int // 同工具+动作+路径+轮次+步骤 → files 下标（合并去重）
	// fileByDispatch 是 dispatch id → files 下标（v4.81 结果阶段回填命中行数；
	// LRU 淘汰重建时一并作废）。
	fileByDispatch map[string]int
	cacheHit       int64
	cacheMiss      int64
	cost           float64
	currency       string
	rateIn         float64 // 最近一次 usage 上报的单价（每 1M tokens）
	rateOut        float64
	rateCacheHit   float64
	provider       string // 最新 request_header 的供应商实例名（切换事件判定）

	// ─── 命令执行折叠（bash 工具推导）状态 ───
	openCmds map[string]*CommandRecord // dispatch id → 待收结果的命令记录
	commands []CommandRecord

	// pendingSkill 是 run_skill 派发 id → 技能名（结果节点 Name 用）。
	pendingSkill map[string]string

	// 浏览器视图边界（trimBrowserWindow 落值，FoldTimeline 导出）。
	nodesDropped int
	nodesFloor   int64

	// ─── 耗时折叠（timing）状态：与上面各字段同趟累加，不引入第二遍扫描 ───
	timing    ContextTiming
	turnStart int64 // 当前轮 turn_started ts（0=无开启轮次）
	stepStart int64 // 当前步起点（request_header ts；0=未知，退化用 lastTs）
	genStart  int64 // 当前步首 token（首条 reasoning/text 增量）ts（0=未到）
	lastTs    int64 // 上一条日志的 ts（步骤起点退化基准）
	openTools map[string]toolOpen
	toolAgg   map[string]*toolAgg
}

// toolOpen 是一次已派发未收结果的工具调用（按调用 id 配对）。
type toolOpen struct {
	name string
	ts   int64
}

// toolAgg 是单个工具名的耗时累计。
type toolAgg struct {
	calls int
	ms    int64
}

func (f *folding) apply(e session.LogEntry) {
	switch e.Kind {
	case "turn_started":
		f.turn++
		f.step = 0
		// stats.Turns 同步计数（真实数据探针定罪：旧日志无 usage 事件时
		// stats.Turns/Steps 恒 0，但请求与轮次明明存在——只在 usage 路径
		// 更新是漏账；这里改为事件驱动，usage 路径的赋值保持幂等）。
		f.stats.Turns = f.turn
		f.timingTurnStart(e)
	case "request_header":
		f.applyHeader(e)
		f.timingStepStart(e)
	case "user_message":
		f.applyUser(e)
	case "reasoning", "text":
		// 主回合的推理/文本增量落盘（v4.62 起子代理增量 wire-only 不落盘），
		// 首条增量即该步「首 token」——真实 TTFT 的秒粒度近似。
		f.timingToken(e)
	case "message", "assistant_message":
		f.applyAssistant(e)
		f.timingAssistant(e)
	case "tool_dispatch":
		f.applyToolDispatch(e)
		f.timingToolDispatch(e)
	case "tool_result":
		f.applyToolResult(e)
		f.timingToolResult(e)
	case "usage":
		f.applyUsage(e)
		f.timingUsage(e)
	case "turn_done":
		f.closePending(e)
		f.killOpenCmds(e.Ts)
		f.timingTurnDone(e)
	case "compaction_done":
		f.applyCompaction(e)
	}
	// 上一条日志 ts（下一步骤起点 header 缺失时的退化基准）。
	if e.Ts > 0 {
		f.lastTs = e.Ts
	}
}

// closePending 在回合结束（turn_done）时关闭仍未收到 usage 的待完成请求：
// 用当前估算分类落一条 estimated 记录——旧日志（迁移/兜底投影）与不报用量
// 的提供方因此仍有趋势柱，诚实标注「估算」不伪造用量数字。
func (f *folding) closePending(e session.LogEntry) {
	rec := f.pending
	if rec == nil {
		return
	}
	rec.Estimated = true
	// 刷新到回合末的当前估算构成（旧日志没有逐请求用量，回合末构成是
	// 「该请求发生时模型可见上下文」的最诚实近似）。
	rec.Category = f.current()
	rec.Ts = e.Ts
	rec.BriefUserSeq = f.lastUserSeq
	rec.BriefRespSeq = f.lastToolCallSeq
	if rec.BriefRespSeq == 0 {
		rec.BriefRespSeq = f.lastAssistantSeq
	}
	if rec.BriefResp == "" {
		rec.BriefResp = f.lastToolCall
	}
	if rec.BriefResp == "" {
		rec.BriefResp = f.lastAssistant
	}
	f.requests = append(f.requests, *rec)
	// stats.Steps 同步（同上：estimated 关闭的请求也计步，旧日志不再恒 0）。
	if rec.Step > f.stats.Steps {
		f.stats.Steps = rec.Step
	}
	f.pending = nil
}

// applyHeader 记录请求头（system prompt + 工具 schema），并开启一个待完成的
// 请求记录（等 usage 带上真实用量后关闭）。
func (f *folding) applyHeader(e session.LogEntry) {
	var h requestHeaderPayload
	if err := json.Unmarshal(e.Payload, &h); err != nil {
		return
	}
	if h.Window > 0 {
		f.window = int64(h.Window)
	}
	// 供应商实例戳（v4.542 刀1）：请求头带 provider 实例名，折叠出模型切换
	// 事件（dsh switch 事件同款）。旧日志无戳（零值），不造事件。
	if h.Provider != "" {
		if f.provider != "" && f.provider != h.Provider {
			f.events = append(f.events, ContextEvent{
				Kind:  "switch",
				Seq:   e.Seq,
				From:  f.provider,
				To:    h.Provider,
				Turn:  f.turn,
				Step:  f.step,
				Ts:    e.Ts,
			})
		}
		f.provider = h.Provider
	}
	f.systemTok = estimateTokens(h.System)
	var toolsTotal int64
	var toolNames []string
	for _, t := range h.Tools {
		toolsTotal += estimateTokens(t.Schema)
		if t.Name != "" {
			toolNames = append(toolNames, t.Name)
		}
	}
	f.toolsTok = toolsTotal
	// 模型可见节点：system prompt / 工具集合只在「变化」时新增一条
	// （每步重复入列会刷屏；节点文本是预览，全文在日志 request_header 里）。
	if h.System != "" && f.lastSysNode != h.System {
		f.lastSysNode = h.System
		f.addNode(SurfaceNode{Seq: e.Seq, Ts: e.Ts, Cat: catSystem, Tokens: f.systemTok, Text: briefOf(h.System, maxNodePreview)})
	}
	toolsKey := strings.Join(toolNames, "|")
	if toolsTotal > 0 && f.lastToolsKey != toolsKey {
		f.lastToolsKey = toolsKey
		f.addNode(SurfaceNode{Seq: e.Seq, Ts: e.Ts, Cat: catTools, Tokens: toolsTotal, Text: briefOf(strings.Join(toolNames, " · "), maxNodePreview)})
	}

	rec := RequestRecord{
		Seq:       e.Seq,
		Ts:        e.Ts,
		Turn:      f.turn,
		Step:      f.step + 1,
		Category:  f.current(),
		BriefUser: f.lastUser,
		BriefIn:   append([]string(nil), f.lastIn...),
		BriefResp: f.lastToolCall,
		// 跳转锚点与 brief 文本同源同拍：user 锚最后一条用户消息节点，resp 锚
		// 优先工具结果节点、退化 assistant 消息节点（0=前端不渲染跳转）。
		BriefUserSeq: f.lastUserSeq,
		BriefRespSeq: f.lastToolCallSeq,
	}
	if rec.BriefRespSeq == 0 {
		rec.BriefRespSeq = f.lastAssistantSeq
	}
	if rec.BriefResp == "" {
		rec.BriefResp = f.lastAssistant
	}
	// 对比上一步：快照点与 Category 同拍（header 组装时），delta 与分类构成自洽。
	f.attachRequestDelta(&rec)
	// v4.212：前缀稳定判定（request_header 编译摘要 → 趋势柱证据链）。
	if h.PrefixStable != nil {
		rec.Prefix = &PrefixVerdict{
			Stable:       *h.PrefixStable,
			PrefixBytes:  h.PrefixBytes,
			AppendCount:  h.AppendCount,
			RewriteCount: h.RewriteCount,
			Hash:         h.CompileHash,
			PrevHash:     h.PrevCompileHash,
		}
	}
	f.pending = &rec
}

// applyUser 处理用户消息；带 "Referenced context:" 前缀的拆分为 inject + user。
func (f *folding) applyUser(e session.LogEntry) {
	var p userPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	// 2.5b 后半：用户消息里的图片附件引用（@路径）计入 stats.Images
	// （此前恒 0 的死字段；引用出现次数口径）。
	f.stats.Images += countImageRefs(p.Content)
	inject, user := splitInjected(p.Content)
	if user != "" {
		tok := estimateTokens(user)
		f.userTok += tok
		f.lastUser = briefOf(user, maxBrief)
		f.lastUserSeq = e.Seq
		f.addNode(SurfaceNode{Seq: e.Seq, Ts: e.Ts, Cat: catUser, Tokens: tok, Text: briefOf(user, maxNodePreview)})
	}
	if inject != "" {
		tok := estimateTokens(inject)
		f.injectTok += tok
		f.stats.Injects++
		src := briefOf(strings.TrimSpace(strings.TrimPrefix(inject, refContextPrefix)), 80)
		f.lastIn = append(f.lastIn, src)
		f.addNode(SurfaceNode{Seq: e.Seq, Ts: e.Ts, Cat: catInject, Tokens: tok, Text: briefOf(inject, maxNodePreview)})
		// inject 事件（1:1 上游 Context Events Inject 行：来源·摘要 + delta）；
		// 此前只统计不落事件，事件流看不到注入何时发生。
		f.events = append(f.events, ContextEvent{
			Kind:   "inject",
			Seq:    e.Seq,
			Delta:  tok,
			Source: src,
			Turn:   f.turn,
			Step:   f.step,
			Ts:     e.Ts,
		})
	}
}

// applyAssistant 处理 assistant 文本（message / assistant_message），三拆
// 思考/正文/工具调用参数（对齐 dsh-context assistantReasoning/Answer/Call）。
// 口径：token 账只认 message/assistant_message（流式增量事件不重复计）；
// 工具调用参数两条来源互斥不重计——运行期日志 message 不内嵌 tool_calls
// （EventLogSink 只落 text/reasoning），参数在 tool_dispatch 事件里另账累计
// （见 applyToolDispatch）；迁移日志（ToLogEntries）反之，tool_calls 内嵌于
// assistant_message、无派发事件。
func (f *folding) applyAssistant(e session.LogEntry) {
	var p assistantPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	ans := estimateTokens(p.Text)
	reas := estimateTokens(p.Reasoning)
	var call int64
	for _, tc := range p.ToolCalls {
		if tc.Name != "" {
			call += estimateTokens(tc.Name + " " + tc.Args)
		}
		// legacy 命令推导（v4.542 真实数据探针定罪：迁移日志零 tool_dispatch，
		// 命令时间线恒空而 bash 结果在——内嵌 tool_calls 即派发事实，按其 id
		// 开挂起命令，后续 tool_result 既有配对照常收尾）。Start=消息 ts
		// （秒级粒度下与真实 exec 起点同拍）；运行期日志 message 不内嵌
		// tool_calls，两路径互斥不重计。
		if tc.Name == "bash" && tc.ID != "" {
			f.openCommand(e, tc.ID, tc.Name, tc.Args)
		}
	}
	total := ans + reas + call
	if total == 0 {
		return
	}
	f.assistantTok += total
	f.stats.AssistantReasoning += reas
	f.stats.AssistantAnswerTokens += ans
	f.stats.AssistantCallTokens += call
	if p.Text != "" {
		f.lastAssistant = briefOf(p.Text, maxBrief)
		f.lastAssistantSeq = e.Seq
	}
	for _, tc := range p.ToolCalls {
		if tc.Name != "" {
			f.lastToolCall = briefOf(tc.Name+" "+tc.Args, maxBrief)
			f.lastToolCallSeq = e.Seq
		}
	}
	f.addNode(SurfaceNode{
		Seq: e.Seq, Ts: e.Ts, Cat: catAssistant, Tokens: total,
		Text: briefOf(p.Text, maxNodePreview),
		Reasoning: reas, AnswerTokens: ans, CallTokens: call,
	})
}

// applyToolDispatch 记录工具调用的简短回复（供步骤 brief 的 Response 行），
// 并给运行期工具调用参数记重放账（assistant 分类子计数；见 applyAssistant
// 口径注）、给 bash 命令与 run_skill 技能开挂起记录。
func (f *folding) applyToolDispatch(e session.LogEntry) {
	var p toolDispatchPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	if p.Partial || p.Name == "" {
		return
	}
	// 运行期参数重放账：工具名+参数会随助手消息进入后续请求的上下文
	// （迁移日志无派发事件、由 applyAssistant 的内嵌 tool_calls 记账）。
	// 挂到最后一条 assistant 节点上（参数紧跟该消息入上下文）：节点携带
	// 子计数，压缩时随节点一起回收，不留虚账。
	callTok := estimateTokens(p.Name + " " + p.Args)
	f.attachCallTokens(callTok)
	f.lastToolCall = briefOf(p.Name+" "+p.Args, maxBrief)
	// 参数带图片路径（识图 image_path 等）同样计入图片引用数。
	f.stats.Images += countImageRefs(p.Args)
	f.openCommand(e, p.ID, p.Name, p.Args)
	f.pendingSkillName(e.Seq, p.ID, p.Name, p.Args)
	f.recordFile(e.Seq, e.Ts, p.Name, p.Args, "", false, p.ID)
}

// openCommand 把 bash 派发折进命令执行时间线（挂起态，结果到达时关闭）。
// 非 bash 工具不构成命令记录（dsh 命令卡口径=shell 执行）。
func (f *folding) openCommand(e session.LogEntry, dispatchID, tool, args string) {
	if tool != "bash" {
		return
	}
	var a struct {
		Command    string `json:"command"`
		Background bool   `json:"run_in_background"`
	}
	if err := json.Unmarshal([]byte(args), &a); err != nil || a.Command == "" {
		return // 非 JSON/无命令：诚实不造记录
	}
	if f.openCmds == nil {
		f.openCmds = map[string]*CommandRecord{}
	}
	rec := &CommandRecord{
		Seq:        e.Seq,
		Time:       e.Ts,
		Tool:       tool,
		Cmd:        briefOf(a.Command, maxCmdPrevw),
		Turn:       f.turn,
		Step:       f.step,
		Start:      e.Ts,
		Background: a.Background,
		Derived:    true,
	}
	f.openCmds[dispatchID] = rec
}

// pendingSkillName 记 run_skill 派发的技能名（结果节点 Name 展示用）。
func (f *folding) pendingSkillName(seq int64, dispatchID, tool, args string) {
	if tool != "run_skill" {
		return
	}
	var a struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(args), &a); err != nil || a.Name == "" {
		return
	}
	if f.pendingSkill == nil {
		f.pendingSkill = map[string]string{}
	}
	f.pendingSkill[dispatchID] = a.Name
}

// attachCallTokens 把工具调用参数重放账挂到最后一条 assistant 节点（Tokens
// 与 CallTokens 同加，压缩随节点回收；stats 子计数同步累计）。找不到宿主
// 节点（极端：回合第一个动作就是工具调用）只进计数器——该笔在压缩时无法
// 回收，诚实近似（幅度=单次调用参数，残留到会话末）。
func (f *folding) attachCallTokens(tok int64) {
	if tok == 0 {
		return
	}
	f.assistantTok += tok
	f.stats.AssistantCallTokens += tok
	for i := len(f.nodes) - 1; i >= 0; i-- {
		if f.nodes[i].Cat == catAssistant {
			f.nodes[i].Tokens += tok
			f.nodes[i].CallTokens += tok
			return
		}
	}
}

// killOpenCmds 把仍未收到结果的命令落为 killed（回合中断/日志截断）。
// ts>0 用事件时间戳收尾；ts=0 是折叠收尾兜底（End 留零=未知）。
func (f *folding) killOpenCmds(ts int64) {
	for id, rec := range f.openCmds {
		rec.Status = CommandKilled
		rec.End = ts
		f.commands = append(f.commands, *rec)
		delete(f.openCmds, id)
	}
}

// closeCommand 用工具结果关闭挂起的命令记录（成功/失败 + 诚实可提取的
// 退出码与输出量）。exit 只认 bash JSON 信封（output_format=json）里的
// exit_code 字段；plain 输出无退出码，留零不猜。
func (f *folding) closeCommand(dispatchID string, e session.LogEntry, p toolResultPayload) {
	rec, ok := f.openCmds[dispatchID]
	if !ok {
		return
	}
	delete(f.openCmds, dispatchID)
	rec.ResultSeq = e.Seq
	rec.End = e.Ts
	if p.Err != "" {
		rec.Status = CommandFailed
		rec.Err = true
	} else {
		rec.Status = CommandCompleted
	}
	rec.OutChars = int64(utf8.RuneCountInString(p.Output))
	rec.Truncated = p.Truncated
	var env struct {
		ExitCode *float64 `json:"exit_code"`
	}
	if err := json.Unmarshal([]byte(p.Output), &env); err == nil && env.ExitCode != nil {
		rec.Exit = int64(*env.ExitCode)
	}
	f.commands = append(f.commands, *rec)
}

// commandStats 从命令时间线汇总会话摘要（runs/failed/ms/bg）。ms 是秒级
// 时间戳差值×1000 的诚实近似（与 Timing 同口径）。
func (f *folding) commandStats() *CommandStats {
	s := &CommandStats{Runs: len(f.commands)}
	for _, c := range f.commands {
		if c.Err {
			s.Failed++
		}
		if c.Background {
			s.Bg++
		}
		if c.End > c.Start {
			s.Ms += (c.End - c.Start) * 1000
		}
	}
	return s
}

// applyToolResult 处理工具结果节点；截断记一次 prune 事件。run_skill 的
// 结果单列 skill 分类（技能内容不藏在 tool 里，对齐 dsh 第七分类）；bash
// 结果关闭挂起的命令记录。
func (f *folding) applyToolResult(e session.LogEntry) {
	var p toolResultPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	text := p.Output
	if p.Err != "" {
		text = "[error] " + p.Err
	}
	tok := estimateTokens(text)
	f.closeCommand(p.ID, e, p)
	if p.Name == "run_skill" {
		f.skillTok += tok
		f.stats.Skills++
		f.addNode(SurfaceNode{Seq: e.Seq, Ts: e.Ts, Cat: catSkill, Tokens: tok, Text: briefOf(text, maxNodePreview), Tool: p.Name, Name: f.takeSkillName(p.ID), Err: p.Err != ""})
	} else {
		f.toolTok += tok
		f.stats.ToolCalls++
		f.addNode(SurfaceNode{Seq: e.Seq, Ts: e.Ts, Cat: catTool, Tokens: tok, Text: briefOf(text, maxNodePreview), Tool: p.Name, Err: p.Err != ""})
	}
	// 结果到达：最近一次工具交换的 brief 锚点改到结果节点（来源 chip+全文，
	// 跳转价值高于派发事件——派发事件本身没有浏览器节点）。
	f.lastToolCallSeq = e.Seq
	if p.Truncated {
		f.stats.Prunes++
		f.events = append(f.events, ContextEvent{
			Kind:   "prune",
			Seq:    e.Seq,
			Source: p.Name,
			Turn:   f.turn,
			Step:   f.step,
			Ts:     e.Ts,
		})
	}
	// 参数里没有路径、但结果携带文件路径的工具（screen_capture 等）在结果
	// 阶段补记；已从参数记过的工具跳过（避免重复行）。
	f.recordFile(e.Seq, e.Ts, p.Name, "", p.Output, true, p.ID)
}

// takeSkillName 取回并清除挂起的技能名（结果节点一次性消费）。
func (f *folding) takeSkillName(dispatchID string) string {
	name, ok := f.pendingSkill[dispatchID]
	if ok {
		delete(f.pendingSkill, dispatchID)
	}
	return name
}

// recordFile 把一次工具调用折叠进文件活动时间线：优先从工具参数提取路径
// （read/write/move/dir 白名单工具），参数无路径时允许从结果输出提取
// （resultOK=true，仅白名单工具）。同一工具+动作+路径在同一轮同一步骤内
// 重复调用合并为一条（刷新时间戳），避免刷屏。
func (f *folding) recordFile(seq, ts int64, tool, args, output string, resultOK bool, dispatchID string) {
	action, ok := fileActionByTool[tool]
	if !ok {
		return
	}
	path := extractPathFromArgs(args)
	if path == "" && resultOK {
		path = extractPathFromOutput(tool, output)
	}
	if path == "" {
		// 无路径但行已在册（grep 等「参数带路径、结果无路径」的工具）：结果
		// 阶段按 dispatch id 回填结果侧信息（命中行数）。
		if dispatchID != "" && resultOK {
			if i, ok := f.fileByDispatch[dispatchID]; ok && tool == "grep" {
				f.files[i].Hits = int64(countLines(output)) // 写入端截断时为命中行数下界（诚实近似）
			}
		}
		return
	}
	rec := FileActivity{Seq: seq, Ts: ts, Turn: f.turn, Step: f.step, Tool: tool, Action: action, Path: briefOf(path, maxFilePreview)}
	if action == "write" {
		rec.Added, rec.Removed = fileDeltaFromArgs(tool, args)
	}
	if tool == "grep" && resultOK {
		rec.Hits = int64(countLines(output))
	}
	if i, ok := f.fileIdx[fileActivityKey(rec)]; ok {
		f.files[i] = rec // 同一步骤同路径重复调用：合并刷新
		if dispatchID != "" {
			f.fileByDispatch[dispatchID] = i
		}
		return
	}
	if f.fileIdx == nil {
		f.fileIdx = map[string]int{}
	}
	f.files = append(f.files, rec)
	f.fileIdx[fileActivityKey(rec)] = len(f.files) - 1
	if dispatchID != "" {
		if f.fileByDispatch == nil {
			f.fileByDispatch = map[string]int{}
		}
		f.fileByDispatch[dispatchID] = len(f.files) - 1
	}
	if len(f.files) > maxFiles {
		f.files = append([]FileActivity(nil), f.files[len(f.files)-maxFiles:]...)
		f.fileIdx = map[string]int{}
		f.fileByDispatch = map[string]int{} // 下标已整体重排，旧映射作废
		for i, x := range f.files {
			f.fileIdx[fileActivityKey(x)] = i
		}
	}
}

// fileActivityKey 生成文件活动的去重键（同轮同步骤同工具同路径合并）。
func fileActivityKey(f FileActivity) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d", f.Tool, f.Action, f.Path, f.Turn, f.Step)
}

// fileDeltaFromArgs 从写类工具参数确定性提取行级增量（v4.81，dsh
// ±added/−removed 同款语义；诚实口径——非 JSON/缺键/类型不符一律零值不猜）：
//   - write_file {content} → +行
//   - edit_file {old_string, new_string} → +new 行 / −old 行
//   - multi_edit {edits: [{old_string, new_string}...]} → 逐条求和
//   - edit_lines {new_content, start_line, end_line} → +new 行 / −(end−start+1) 行
func fileDeltaFromArgs(tool, args string) (added, removed int64) {
	if args == "" {
		return 0, 0
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(args), &m); err != nil {
		return 0, 0
	}
	str := func(key string) string {
		s, _ := m[key].(string)
		return s
	}
	switch tool {
	case "write_file":
		return int64(countLines(str("content"))), 0
	case "edit_file":
		return int64(countLines(str("new_string"))), int64(countLines(str("old_string")))
	case "multi_edit":
		edits, _ := m["edits"].([]any)
		for _, it := range edits {
			em, _ := it.(map[string]any)
			if em == nil {
				continue
			}
			o, _ := em["old_string"].(string)
			nw, _ := em["new_string"].(string)
			added += int64(countLines(nw))
			removed += int64(countLines(o))
		}
		return added, removed
	case "edit_lines":
		added = int64(countLines(str("new_content")))
		sf, _ := m["start_line"].(float64)
		ef, _ := m["end_line"].(float64)
		if sf > 0 && ef >= sf {
			removed = int64(ef) - int64(sf) + 1
		}
		return added, removed
	}
	return 0, 0
}

// fileActionByTool 是「工具 → 文件活动动作」白名单。动作语义：read=读取/
// 搜索/识别输入；write=写入/编辑/生成产物；move=移动/重命名；dir=列目录。
var fileActionByTool = map[string]string{
	"read_file":      "read",
	"grep":           "read",
	"vision":         "read",
	"format_convert": "read",
	"write_file":     "write",
	"edit_file":      "write",
	"multi_edit":     "write",
	"edit_lines":     "write",
	"chart_gen":      "write",
	"diagram_gen":    "write",
	"screen_capture": "write",
	"move_file":      "move",
	"replace":        "write",
	"ls":             "dir",
}

// filePathFromOutputTools 是「路径只能从结果输出提取」的工具白名单
// （参数无路径键，输出携带生成/保存的文件路径）。
var filePathFromOutputTools = map[string]bool{"screen_capture": true}

// filePathArgKeys 是工具参数里的路径键候选（按优先级）。桌面办公扩展工具
// （docx/xlsx/preview 等）统一用 rel/path 键，内置工具用 path/source 等。
var filePathArgKeys = []string{"path", "rel", "source", "destination", "image_path", "output"}

// extractPathFromArgs 从工具参数 JSON 里确定性提取路径（按 filePathArgKeys
// 优先级取第一个非空字符串；非 JSON 或取不到返回空）。
func extractPathFromArgs(args string) string {
	if args == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(args), &m); err != nil {
		return ""
	}
	for _, k := range filePathArgKeys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// extractPathFromOutput 从工具结果输出提取路径（仅 filePathFromOutputTools
// 白名单；输出可能是 JSON 或纯文本，纯文本里不猜路径——诚实不造数）。
func extractPathFromOutput(tool, output string) string {
	if !filePathFromOutputTools[tool] || output == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(output), &m); err != nil {
		return ""
	}
	for _, k := range filePathArgKeys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// maxFilePreview 是文件活动路径的展示上限（浏览视图截断，全文在日志里）。
const maxFilePreview = 240

// applyUsage 用真实用量关闭待完成请求（或为旧日志创建请求记录），并把分类
// 按实际 promptTokens 等比缩放锚定。
func (f *folding) applyUsage(e session.LogEntry) {
	var p usagePayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	if p.Turn > 0 {
		f.turn = p.Turn
	}
	f.step++
	rec := f.pending
	if rec == nil {
		rec = &RequestRecord{
			Seq:          e.Seq,
			Ts:           e.Ts,
			Turn:         f.turn,
			Step:         f.step,
			Category:     f.current(),
			BriefUser:    f.lastUser,
			BriefIn:      append([]string(nil), f.lastIn...),
			BriefResp:    f.lastToolCall,
			BriefUserSeq: f.lastUserSeq,
			BriefRespSeq: f.lastToolCallSeq,
		}
		if rec.BriefResp == "" {
			rec.BriefResp = f.lastAssistant
		}
		if rec.BriefRespSeq == 0 {
			rec.BriefRespSeq = f.lastAssistantSeq
		}
		// 旧日志无 request_header 的退化路径：快照点退到 usage 关闭时（该步
		// 输出已入 surface，与 header 路径口径有差——诚实近似，delta 同拍
		// Category，仍自洽）。
		f.attachRequestDelta(rec)
	} else {
		rec.Step = f.step
		rec.Ts = e.Ts
		// header 在用户消息/工具调用之前发出：关闭时刷新 brief，让
		// 「输入→回复」反映该请求实际看到的最近内容。锚点同拍刷新。
		rec.BriefUser = f.lastUser
		rec.BriefIn = append([]string(nil), f.lastIn...)
		rec.BriefResp = f.lastToolCall
		rec.BriefUserSeq = f.lastUserSeq
		rec.BriefRespSeq = f.lastToolCallSeq
		if rec.BriefResp == "" {
			rec.BriefResp = f.lastAssistant
		}
		if rec.BriefRespSeq == 0 {
			rec.BriefRespSeq = f.lastAssistantSeq
		}
	}
	rec.PromptTokens = p.PromptTokens
	rec.OutputTokens = p.CompletionTokens
	rec.CacheHitTokens = p.CacheHitTokens
	rec.CacheMissTokens = p.CacheMissTokens
	rec.Category = scaleCategory(rec.Category, p.PromptTokens, rec.Category.Total())
	// 逐请求成本（usage 带价格时按该事件单价计得；洞察页逐日归因数据源）。
	reqCost := (float64(p.CacheHitTokens)*p.CacheHitPrice +
		float64(p.CacheMissTokens)*p.Input +
		float64(p.CompletionTokens)*p.Output) / 1e6
	if p.CacheHitPrice > 0 || p.Input > 0 || p.Output > 0 {
		rec.Cost = reqCost
	}
	f.requests = append(f.requests, *rec)
	f.pending = nil

	// 会话级缓存命中率与费用估算（pricing 字段存在时）。
	if p.SessionCacheHitTokens > 0 || p.SessionCacheMissTokens > 0 {
		f.cacheHit = p.SessionCacheHitTokens
		f.cacheMiss = p.SessionCacheMissTokens
	} else {
		f.cacheHit += p.CacheHitTokens
		f.cacheMiss += p.CacheMissTokens
	}
	if f.cacheHit+f.cacheMiss > 0 {
		f.stats.CacheHitPercent = float64(f.cacheHit) * 100 / float64(f.cacheHit+f.cacheMiss)
	}
	if p.Currency != "" {
		f.currency = p.Currency
	}
	if p.CacheHitPrice > 0 || p.Input > 0 || p.Output > 0 {
		f.cost += reqCost
	}
	// 2.5e 成本 hover：记录最近一次非零单价（供应商口径，每 1M tokens）。
	if p.Input > 0 {
		f.rateIn = p.Input
	}
	if p.Output > 0 {
		f.rateOut = p.Output
	}
	if p.CacheHitPrice > 0 {
		f.rateCacheHit = p.CacheHitPrice
	}
	f.stats.CostEstimate = f.cost
	f.stats.Steps = f.step
	f.stats.Turns = f.turn
}

// applyCompaction 标记被压缩的节点为 gone 并移入归档，记录 compact 事件。
func (f *folding) applyCompaction(e session.LogEntry) {
	f.stats.Compacts++
	var reclaimed int64
	kept := f.nodes[:0]
	for _, n := range f.nodes {
		if n.Cat == catSystem || n.Cat == catTools || n.Gone != nil {
			kept = append(kept, n)
			continue
		}
		gone := e.Seq
		n.Gone = &gone
		f.archive = append(f.archive, n)
		switch n.Cat {
		case catUser:
			f.userTok -= n.Tokens
		case catInject:
			f.injectTok -= n.Tokens
		case catSkill:
			f.skillTok -= n.Tokens
		case catAssistant:
			f.assistantTok -= n.Tokens
		case catTool:
			f.toolTok -= n.Tokens
		}
		f.catItems[n.Cat]--
		reclaimed += n.Tokens
	}
	f.nodes = kept
	f.events = append(f.events, ContextEvent{
		Kind:  "compact",
		Seq:   e.Seq,
		Delta: -reclaimed,
		Turn:  f.turn,
		Step:  f.step,
		Ts:    e.Ts,
	})
	// 对比上一步：压缩改写差分基线（被压节点移入归档），其后首个请求的
	// delta 标 Approx 近似。
	f.compactMark = e.Seq
}

// current 返回当前七分类构成（系统/工具来自最新 header 估算）。
func (f *folding) current() Category {
	return Category{
		System:    f.systemTok,
		Tools:     f.toolsTok,
		User:      f.userTok,
		Inject:    f.injectTok,
		Skill:     f.skillTok,
		Assistant: f.assistantTok,
		Tool:      f.toolTok,
	}
}

// addNode 追加浏览器节点并维护活分类条数计数（快照差分的口径源，与节点
// 列表解耦——视图截断不动计数，见 snapshotSurface）。
func (f *folding) addNode(n SurfaceNode) {
	f.nodes = append(f.nodes, n)
	switch n.Cat {
	case catSystem, catTools:
		// header 节点只在变化时入列，是「快照」不是逐项元素：快照侧按
		// systemTok/toolsTok 整值聚合（恒 1 项），不计条数。
	default:
		if f.catItems == nil {
			f.catItems = map[string]int64{}
		}
		f.catItems[n.Cat]++
	}
}

// trimBrowserWindow 收口浏览器视图上限：节点/归档/事件/文件/命令超出上限
// 时丢最旧（token 合计与差分走增量计数器，不受影响）。NodesDropped/
// NodesFloor 向前端标注节点保真边界。
func (f *folding) trimBrowserWindow() {
	if len(f.nodes) > maxNodes {
		dropped := f.nodes[:len(f.nodes)-maxNodes]
		for _, n := range dropped {
			switch n.Cat {
			case catSystem, catTools:
			default:
				f.catItems[n.Cat]--
			}
		}
		f.nodes = append([]SurfaceNode(nil), f.nodes[len(f.nodes)-maxNodes:]...)
		f.nodesDropped = len(dropped)
	}
	if len(f.archive) > maxArchive {
		f.archive = append([]SurfaceNode(nil), f.archive[len(f.archive)-maxArchive:]...)
	}
	if len(f.events) > maxEvents {
		f.events = append([]ContextEvent(nil), f.events[len(f.events)-maxEvents:]...)
	}
	if len(f.files) > maxFiles {
		f.files = append([]FileActivity(nil), f.files[len(f.files)-maxFiles:]...)
	}
	if len(f.commands) > maxCommands {
		f.commands = append([]CommandRecord(nil), f.commands[len(f.commands)-maxCommands:]...)
	}
	if len(f.nodes) > 0 {
		f.nodesFloor = f.nodes[0].Seq
	}
}

// ─── 对比上一步（per-request surface 快照与差分） ───────────────────

// surfaceSnapshot 是一次请求快照点的活上下文聚合（项数/token 按七分类）。
type surfaceSnapshot struct {
	items       map[string]int64
	tokens      map[string]int64
	compactMark int64
}

// catDeltaOrder 是差分遍历的分类顺序（稳定，不依赖 map 迭代序）。
var catDeltaOrder = [...]string{catSystem, catTools, catUser, catInject, catSkill, catAssistant, catTool}

// snapshotSurface 聚合当前模型可见上下文。全分类走增量计数器（token 见
// userTok 等字段、条数见 catItems）——与浏览器节点列表解耦，视图截断不
// 改变差分口径。system/tools 是「整值快照」分类：最新 header 的整体估算
// 记 1 项（节点只在变化时入列且旧头不回收，逐条聚合会重计历史头）。
func (f *folding) snapshotSurface() *surfaceSnapshot {
	s := &surfaceSnapshot{
		items:       map[string]int64{},
		tokens:      map[string]int64{},
		compactMark: f.compactMark,
	}
	if f.systemTok > 0 {
		s.items[catSystem] = 1
		s.tokens[catSystem] = f.systemTok
	}
	if f.toolsTok > 0 {
		s.items[catTools] = 1
		s.tokens[catTools] = f.toolsTok
	}
	for _, cat := range []string{catUser, catInject, catSkill, catAssistant, catTool} {
		if n := f.catItems[cat]; n > 0 {
			s.items[cat] = n
		}
	}
	for cat, tok := range map[string]int64{
		catUser:      f.userTok,
		catInject:    f.injectTok,
		catSkill:     f.skillTok,
		catAssistant: f.assistantTok,
		catTool:      f.toolTok,
	} {
		if tok > 0 {
			s.tokens[cat] = tok
		}
	}
	return s
}

// attachRequestDelta 计算该请求相对上一次请求的 surface 净变化（Signed：
// +=新增/膨胀，−=移除/瘦身），挂到 rec.Delta，并把当前 surface 记为新基线。
// prev=nil（首个请求）时 First=true，差值即全量构成；两次快照间发生过压缩
// 时 Approx=true（基线被结构性改写，诚实标注近似）。ByCat 只含有变化的
// 分类，按 |tokens| 降序（并列按名称稳定排序，与 Tools 排行同纪律）。
func (f *folding) attachRequestDelta(rec *RequestRecord) {
	cur := f.snapshotSurface()
	prev := f.prevSurface
	d := &RequestDelta{First: prev == nil, Approx: prev != nil && prev.compactMark != cur.compactMark}
	empty := surfaceSnapshot{items: map[string]int64{}, tokens: map[string]int64{}}
	base := prev
	if base == nil {
		base = &empty
	}
	for _, c := range catDeltaOrder {
		di := cur.items[c] - base.items[c]
		dt := cur.tokens[c] - base.tokens[c]
		if di == 0 && dt == 0 {
			continue
		}
		d.Items += di
		d.Tokens += dt
		d.ByCat = append(d.ByCat, CatDelta{Cat: c, Items: di, Tokens: dt})
	}
	sort.SliceStable(d.ByCat, func(i, j int) bool {
		ai, aj := d.ByCat[i].Tokens, d.ByCat[j].Tokens
		if ai < 0 {
			ai = -ai
		}
		if aj < 0 {
			aj = -aj
		}
		if ai != aj {
			return ai > aj
		}
		return d.ByCat[i].Cat < d.ByCat[j].Cat
	})
	f.prevSurface = cur
	rec.Delta = d
}

// splitInjected 把带 "Referenced context:" 前缀的 user 消息拆为 inject 与 user。
func splitInjected(s string) (inject, user string) {
	if !strings.HasPrefix(s, refContextPrefix) {
		return "", s
	}
	rest := s[len(refContextPrefix):]
	if i := strings.Index(rest, "\n\n"); i >= 0 {
		return refContextPrefix + rest[:i], rest[i+2:]
	}
	return s, ""
}

// ─── payload 解析结构（与 session 日志 JSON 对齐，独立副本避免耦合） ───

type userPayload struct {
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type assistantPayload struct {
	ID        string `json:"id,omitempty"`
	Text      string `json:"text,omitempty"`
	Reasoning string `json:"reasoning,omitempty"`
	ToolCalls []struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name"`
		Args string `json:"args"`
	} `json:"tool_calls,omitempty"`
}

type toolDispatchPayload struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Args    string `json:"args"`
	Partial bool   `json:"partial,omitempty"`
}

type toolResultPayload struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Output    string `json:"output,omitempty"`
	Err       string `json:"err,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

type usagePayload struct {
	PromptTokens           int64   `json:"promptTokens,omitempty"`
	CompletionTokens       int64   `json:"completionTokens,omitempty"`
	CacheHitTokens         int64   `json:"cacheHitTokens,omitempty"`
	CacheMissTokens        int64   `json:"cacheMissTokens,omitempty"`
	SessionCacheHitTokens  int64   `json:"sessionCacheHitTokens,omitempty"`
	SessionCacheMissTokens int64   `json:"sessionCacheMissTokens,omitempty"`
	Input                  float64 `json:"input,omitempty"`
	Output                 float64 `json:"output,omitempty"`
	CacheHitPrice          float64 `json:"cacheHitPrice,omitempty"`
	Currency               string  `json:"currency,omitempty"`
	Source                 string  `json:"source,omitempty"`
	Turn                   int     `json:"turn,omitempty"`
}

type requestHeaderPayload struct {
	System string `json:"system,omitempty"`
	Tools  []struct {
		Name   string `json:"name"`
		Schema string `json:"schema,omitempty"`
	} `json:"tools,omitempty"`
	Window int `json:"window,omitempty"`
	// Provider 是供应商实例名（v4.542 刀1 追加列，旧日志零值=无戳不折叠
	// 切换事件）。
	Provider string `json:"provider,omitempty"`
	// v4.212（5.2）前缀稳定判定（追加列，旧日志零值=未计算不折叠）。
	CompileHash     string `json:"compileHash,omitempty"`
	PrevCompileHash string `json:"prevCompileHash,omitempty"`
	PrefixStable    *bool  `json:"prefixStable,omitempty"`
	PrefixBytes     int64  `json:"prefixBytes,omitempty"`
	AppendCount     int    `json:"appendCount,omitempty"`
	RewriteCount    int    `json:"rewriteCount,omitempty"`
}
