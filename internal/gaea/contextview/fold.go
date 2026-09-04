package contextview

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// 常量：上下文分类标签（浏览器/归档重建用）。
const (
	catSystem    = "system"
	catTools     = "tools"
	catUser      = "user"
	catInject    = "inject"
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
	tl := ContextTimeline{
		Ok:      true,
		Window:  f.window,
		Current: f.current(),
		Stats:   f.stats,
		Events:  f.events,
		Nodes:   f.nodes,
		Archive: f.archive,
		Files:   f.files,
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
	assistantTok int64 // 活着的 assistant 节点 token 合计
	toolTok      int64 // 活着的 tool 节点 token 合计

	nodes   []SurfaceNode
	archive []SurfaceNode

	// ─── 对比上一步（per-request surface 快照）状态：与上面各字段同趟维护 ───
	prevSurface *surfaceSnapshot // 上一次请求快照点（nil=尚无基线，首个请求 First）
	compactMark int64            // 最近一次压缩事件 seq（快照间变化=Approx 近似）

	requests []RequestRecord
	events   []ContextEvent
	stats    Stats

	pending *RequestRecord // request_header 已开、usage 未到的请求

	lastUser       string
	lastIn         []string
	lastAssistant  string
	lastToolCall   string
	files          []FileActivity
	fileIdx        map[string]int // 同工具+动作+路径+轮次+步骤 → files 下标（合并去重）
	cacheHit       int64
	cacheMiss      int64
	cost           float64
	currency       string

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
	if rec.BriefResp == "" {
		rec.BriefResp = f.lastToolCall
	}
	if rec.BriefResp == "" {
		rec.BriefResp = f.lastAssistant
	}
	f.requests = append(f.requests, *rec)
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
		f.nodes = append(f.nodes, SurfaceNode{Seq: e.Seq, Cat: catSystem, Tokens: f.systemTok, Text: briefOf(h.System, maxNodePreview)})
	}
	toolsKey := strings.Join(toolNames, "|")
	if toolsTotal > 0 && f.lastToolsKey != toolsKey {
		f.lastToolsKey = toolsKey
		f.nodes = append(f.nodes, SurfaceNode{Seq: e.Seq, Cat: catTools, Tokens: toolsTotal, Text: briefOf(strings.Join(toolNames, " · "), maxNodePreview)})
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
	}
	if rec.BriefResp == "" {
		rec.BriefResp = f.lastAssistant
	}
	// 对比上一步：快照点与 Category 同拍（header 组装时），delta 与分类构成自洽。
	f.attachRequestDelta(&rec)
	f.pending = &rec
}

// applyUser 处理用户消息；带 "Referenced context:" 前缀的拆分为 inject + user。
func (f *folding) applyUser(e session.LogEntry) {
	var p userPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	inject, user := splitInjected(p.Content)
	if user != "" {
		tok := estimateTokens(user)
		f.userTok += tok
		f.lastUser = briefOf(user, maxBrief)
		f.nodes = append(f.nodes, SurfaceNode{Seq: e.Seq, Cat: catUser, Tokens: tok, Text: briefOf(user, maxNodePreview)})
	}
	if inject != "" {
		tok := estimateTokens(inject)
		f.injectTok += tok
		f.stats.Injects++
		src := briefOf(strings.TrimSpace(strings.TrimPrefix(inject, refContextPrefix)), 80)
		f.lastIn = append(f.lastIn, src)
		f.nodes = append(f.nodes, SurfaceNode{Seq: e.Seq, Cat: catInject, Tokens: tok, Text: briefOf(inject, maxNodePreview)})
	}
}

// applyAssistant 处理 assistant 文本（message / assistant_message）。
func (f *folding) applyAssistant(e session.LogEntry) {
	var p assistantPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	if p.Text == "" {
		return
	}
	tok := estimateTokens(p.Text)
	f.assistantTok += tok
	f.lastAssistant = briefOf(p.Text, maxBrief)
	for _, tc := range p.ToolCalls {
		if tc.Name != "" {
			f.lastToolCall = briefOf(tc.Name+" "+tc.Args, maxBrief)
		}
	}
	f.nodes = append(f.nodes, SurfaceNode{Seq: e.Seq, Cat: catAssistant, Tokens: tok, Text: briefOf(p.Text, maxNodePreview)})
}

// applyToolDispatch 记录工具调用的简短回复（供步骤 brief 的 Response 行）。
func (f *folding) applyToolDispatch(e session.LogEntry) {
	var p toolDispatchPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	if p.Partial || p.Name == "" {
		return
	}
	f.lastToolCall = briefOf(p.Name+" "+p.Args, maxBrief)
	f.recordFile(e.Seq, e.Ts, p.Name, p.Args, "", false)
}

// applyToolResult 处理工具结果节点；截断记一次 prune 事件。
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
	f.toolTok += tok
	f.stats.ToolCalls++
	f.nodes = append(f.nodes, SurfaceNode{Seq: e.Seq, Cat: catTool, Tokens: tok, Text: briefOf(text, maxNodePreview), Tool: p.Name, Err: p.Err != ""})
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
	f.recordFile(e.Seq, e.Ts, p.Name, "", p.Output, true)
}

// maxFiles 是文件活动时间线保留条数（浏览视图上限，超出丢最旧）。
const maxFiles = 200

// recordFile 把一次工具调用折叠进文件活动时间线：优先从工具参数提取路径
// （read/write/move/dir 白名单工具），参数无路径时允许从结果输出提取
// （resultOK=true，仅白名单工具）。同一工具+动作+路径在同一轮同一步骤内
// 重复调用合并为一条（刷新时间戳），避免刷屏。
func (f *folding) recordFile(seq, ts int64, tool, args, output string, resultOK bool) {
	action, ok := fileActionByTool[tool]
	if !ok {
		return
	}
	path := extractPathFromArgs(args)
	if path == "" && resultOK {
		path = extractPathFromOutput(tool, output)
	}
	if path == "" {
		return
	}
	rec := FileActivity{Seq: seq, Ts: ts, Turn: f.turn, Step: f.step, Tool: tool, Action: action, Path: briefOf(path, maxFilePreview)}
	if i, ok := f.fileIdx[fileActivityKey(rec)]; ok {
		f.files[i] = rec // 同一步骤同路径重复调用：合并刷新
		return
	}
	if f.fileIdx == nil {
		f.fileIdx = map[string]int{}
	}
	f.files = append(f.files, rec)
	f.fileIdx[fileActivityKey(rec)] = len(f.files) - 1
	if len(f.files) > maxFiles {
		f.files = append([]FileActivity(nil), f.files[len(f.files)-maxFiles:]...)
		f.fileIdx = map[string]int{}
		for i, x := range f.files {
			f.fileIdx[fileActivityKey(x)] = i
		}
	}
}

// fileActivityKey 生成文件活动的去重键（同轮同步骤同工具同路径合并）。
func fileActivityKey(f FileActivity) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d", f.Tool, f.Action, f.Path, f.Turn, f.Step)
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
			Seq:       e.Seq,
			Ts:        e.Ts,
			Turn:      f.turn,
			Step:      f.step,
			Category:  f.current(),
			BriefUser: f.lastUser,
			BriefIn:   append([]string(nil), f.lastIn...),
			BriefResp: f.lastToolCall,
		}
		if rec.BriefResp == "" {
			rec.BriefResp = f.lastAssistant
		}
		// 旧日志无 request_header 的退化路径：快照点退到 usage 关闭时（该步
		// 输出已入 surface，与 header 路径口径有差——诚实近似，delta 同拍
		// Category，仍自洽）。
		f.attachRequestDelta(rec)
	} else {
		rec.Step = f.step
		rec.Ts = e.Ts
		// header 在用户消息/工具调用之前发出：关闭时刷新 brief，让
		// 「输入→回复」反映该请求实际看到的最近内容。
		rec.BriefUser = f.lastUser
		rec.BriefIn = append([]string(nil), f.lastIn...)
		rec.BriefResp = f.lastToolCall
		if rec.BriefResp == "" {
			rec.BriefResp = f.lastAssistant
		}
	}
	rec.PromptTokens = p.PromptTokens
	rec.OutputTokens = p.CompletionTokens
	rec.CacheHitTokens = p.CacheHitTokens
	rec.CacheMissTokens = p.CacheMissTokens
	rec.Category = scaleCategory(rec.Category, p.PromptTokens, rec.Category.Total())
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
		f.cost += (float64(p.CacheHitTokens)*p.CacheHitPrice +
			float64(p.CacheMissTokens)*p.Input +
			float64(p.CompletionTokens)*p.Output) / 1e6
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
		case catAssistant:
			f.assistantTok -= n.Tokens
		case catTool:
			f.toolTok -= n.Tokens
		}
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

// ─── 耗时折叠（对齐 dsh-context TimingTotals 的诚实近似版） ──────────
// 日志 ts 为秒级（time.Now().Unix()），所有时长按秒差 ×1000 记 ms；只记
// 非负差值（时钟回拨/同秒日志静默跳过），无法支撑的指标留零不伪造。

// timingStepStart 记录当前步的起点（request_header ts = 请求组装发出时刻）。
func (f *folding) timingStepStart(e session.LogEntry) {
	f.stepStart = e.Ts
	f.genStart = 0
}

// timingTurnStart 开启一轮活跃时长窗口；步骤状态跨轮不残留。
func (f *folding) timingTurnStart(e session.LogEntry) {
	f.turnStart = e.Ts
	f.stepStart = 0
	f.genStart = 0
}

// timingToken 记该步首 token：首条 reasoning/text 增量。TTFT = 首 token −
// 步骤起点（request_header ts；旧迁移日志每轮只有一条 header，后续步骤退
// 化为「前一事件 ts」起算——两种口径都只计非负差值）。
func (f *folding) timingToken(e session.LogEntry) {
	if f.genStart != 0 {
		return
	}
	f.genStart = e.Ts
	if start, ok := f.timingStepBase(e.Ts); ok {
		f.timing.TTFTMs += (e.Ts - start) * 1000
	}
}

// timingAssistant 关闭一步生成：calls 计数（message/assistant_message 条数
// = 主回合模型调用次数）；gen = 消息收尾 − 首 token。该步无任何已落盘增量
// 时（纯工具调用步），消息本身即首个产物——等待整体记入 TTFT，gen 留 0。
func (f *folding) timingAssistant(e session.LogEntry) {
	f.timing.Calls++
	if f.genStart != 0 {
		if e.Ts > f.genStart {
			f.timing.GenMs += (e.Ts - f.genStart) * 1000
		}
	} else if start, ok := f.timingStepBase(e.Ts); ok {
		f.timing.TTFTMs += (e.Ts - start) * 1000
	}
	f.genStart = 0
}

// timingToolDispatch 记挂起工具调用（按 id 配对）。partial 为流式预发射
// （完整派发随后再来），与折叠口径一致跳过，避免从过早的 ts 起算。
func (f *folding) timingToolDispatch(e session.LogEntry) {
	var p toolDispatchPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil || p.Partial || p.ID == "" || p.Name == "" {
		return
	}
	if f.openTools == nil {
		f.openTools = map[string]toolOpen{}
	}
	f.openTools[p.ID] = toolOpen{name: p.Name, ts: e.Ts}
}

// timingToolResult 配对关闭工具调用：ms = 结果 − 派发，并行调用重复计
// （与 dsh 同口径，如实不去重）。迁移日志的 tool_result 没有对应 dispatch，
// 不配对不计数；中断轮的未配对派发同样不计。
func (f *folding) timingToolResult(e session.LogEntry) {
	var p toolResultPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil || p.ID == "" {
		return
	}
	o, ok := f.openTools[p.ID]
	if !ok {
		return
	}
	delete(f.openTools, p.ID)
	if o.ts <= 0 || e.Ts < o.ts {
		return
	}
	ms := (e.Ts - o.ts) * 1000
	f.timing.ToolsMs += ms
	f.timing.ToolCalls++
	name := p.Name
	if name == "" {
		name = o.name
	}
	if f.toolAgg == nil {
		f.toolAgg = map[string]*toolAgg{}
	}
	agg := f.toolAgg[name]
	if agg == nil {
		agg = &toolAgg{}
		f.toolAgg[name] = agg
	}
	agg.calls++
	agg.ms += ms
}

// timingUsage 关闭一步（usage = 该次请求收尾），防止 header 缺失的日志把
// 上一步的起点残留进下一步。
func (f *folding) timingUsage(session.LogEntry) {
	f.stepStart = 0
	f.genStart = 0
}

// timingTurnDone 累加本轮活跃时长并复位步骤状态。已派发未收结果的工具
// 保留在配对表里（id 全局唯一，误配对不可能；结果若永不到来则不计）。
func (f *folding) timingTurnDone(e session.LogEntry) {
	f.addWall(f.turnStart, e.Ts)
	f.turnStart = 0
	f.stepStart = 0
	f.genStart = 0
}

// timingClose 收尾：末轮无 turn_done（完整中断，BalanceEntries 不走本路径）
// 时以最后一条日志 ts 收尾。
func (f *folding) timingClose() {
	f.addWall(f.turnStart, f.lastTs)
}

func (f *folding) addWall(start, end int64) {
	if start > 0 && end >= start {
		f.timing.WallMs += (end - start) * 1000
	}
}

// timingStepBase 返回当前步的起点 ts：优先 request_header，缺失时退化为
// 前一事件 ts（dsh「步骤起点」口径的日志可得近似）。
func (f *folding) timingStepBase(ts int64) (int64, bool) {
	switch {
	case f.stepStart > 0 && f.stepStart <= ts:
		return f.stepStart, true
	case f.stepStart == 0 && f.lastTs > 0 && f.lastTs <= ts:
		return f.lastTs, true
	}
	return 0, false
}

// timingToolsRanked 把每工具 {calls, ms} 累计按 ms 降序（并列按名称，保证
// 确定性）排为排行条目，截断 20（dsh 上限同款）。
func (f *folding) timingToolsRanked() []ToolTiming {
	if len(f.toolAgg) == 0 {
		return nil
	}
	names := make([]string, 0, len(f.toolAgg))
	for n := range f.toolAgg {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := f.toolAgg[names[i]], f.toolAgg[names[j]]
		if a.ms != b.ms {
			return a.ms > b.ms
		}
		return names[i] < names[j]
	})
	if len(names) > 20 {
		names = names[:20]
	}
	out := make([]ToolTiming, 0, len(names))
	for _, n := range names {
		agg := f.toolAgg[n]
		out = append(out, ToolTiming{Name: n, Calls: agg.calls, Ms: agg.ms})
	}
	return out
}

// nonZero 报告是否有任何可报的耗时数据（全零 → Timing 整体省略）。
func (t ContextTiming) nonZero() bool {
	return t.WallMs > 0 || t.TTFTMs > 0 || t.GenMs > 0 || t.Calls > 0 ||
		t.ToolsMs > 0 || t.ToolCalls > 0 || len(t.Tools) > 0
}

// current 返回当前六分类构成（系统/工具来自最新 header 估算）。
func (f *folding) current() Category {
	return Category{
		System:    f.systemTok,
		Tools:     f.toolsTok,
		User:      f.userTok,
		Inject:    f.injectTok,
		Assistant: f.assistantTok,
		Tool:      f.toolTok,
	}
}

// ─── 对比上一步（per-request surface 快照与差分） ───────────────────

// surfaceSnapshot 是一次请求快照点的活节点聚合（项数/token 按六分类）。
type surfaceSnapshot struct {
	items       map[string]int64
	tokens      map[string]int64
	compactMark int64
}

// catDeltaOrder 是差分遍历的分类顺序（稳定，不依赖 map 迭代序）。
var catDeltaOrder = [...]string{catSystem, catTools, catUser, catInject, catAssistant, catTool}

// snapshotSurface 聚合当前模型可见 surface。system/tools 走最新 header 的
// 整体估算——节点只在变化时入列且旧头不回收，逐条聚合会重计历史头；
// user/inject/assistant/tool 活节点逐条聚合（离开 surface 的节点已被移入
// 归档，不在此列）。
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
	for _, n := range f.nodes {
		if n.Cat == catSystem || n.Cat == catTools {
			continue
		}
		s.items[n.Cat]++
		s.tokens[n.Cat] += n.Tokens
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
}
