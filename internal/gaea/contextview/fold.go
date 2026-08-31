package contextview

import (
	"encoding/json"
	"fmt"
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
}

func (f *folding) apply(e session.LogEntry) {
	switch e.Kind {
	case "turn_started":
		f.turn++
		f.step = 0
	case "request_header":
		f.applyHeader(e)
	case "user_message":
		f.applyUser(e)
	case "message", "assistant_message":
		f.applyAssistant(e)
	case "tool_dispatch":
		f.applyToolDispatch(e)
	case "tool_result":
		f.applyToolResult(e)
	case "usage":
		f.applyUsage(e)
	case "turn_done":
		f.closePending(e)
	case "compaction_done":
		f.applyCompaction(e)
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
	f.nodes = append(f.nodes, SurfaceNode{Seq: e.Seq, Cat: catTool, Tokens: tok, Text: briefOf(text, maxNodePreview)})
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
