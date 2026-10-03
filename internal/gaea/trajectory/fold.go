package trajectory

import (
	"encoding/json"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/util"
)

// 预览上限：轨迹是浏览视图，正文/推理/输出做展示级截断（全文在日志里）。
const (
	maxTextPreview = 2000
	maxOutPreview  = 4000
)

// FoldTrajectory 把会话日志条目折叠为轨迹快照。纯函数：同输入必同输出。
func FoldTrajectory(entries []session.LogEntry) Trajectory {
	f := &folding{assistantIdx: -1}
	for _, e := range entries {
		f.apply(e)
	}
	if f.cur != nil {
		f.turns = append(f.turns, *f.cur)
	}
	out := Trajectory{Ok: true, Turns: f.turns, BetweenTurns: f.between}
	// Go 的 nil 切片会序列化成 JSON null，前端按数组消费（.length / flatMap）
	// 会整页崩——空会话 Turns 恰好是 nil。统一兜底为空切片。
	if out.Turns == nil {
		out.Turns = []Turn{}
	}
	for i := range out.Turns {
		if out.Turns[i].Records == nil {
			out.Turns[i].Records = []Record{}
		}
	}
	if out.BetweenTurns == nil {
		out.BetweenTurns = []Record{}
	}
	return out
}

type folding struct {
	turns   []Turn
	between []Record
	cur     *Turn
	step    int

	lastSystem string // 上一个 request_header 的 system（change 检测）
	lastTools  int

	headerTs int64
	// assistantIdx / toolByID 存「f.cur.Records 下标」而不是 *Record：Records
	// 每次增长都会重新分配底层数组，早先取出的元素指针会指向旧数组——之后经
	// 它写 Record 自身字段（DurationMs 等）会静默写进废弃数组而丢失（Tool /
	// Assistant 是堆上指针，内容写才侥幸可见）。下标 + 取值时重算与切片增长无关。
	assistantIdx int              // 正在累积的 assistant 记录下标（-1 = 无）
	toolByID     map[string]int   // tool ID → f.cur.Records 下标
	toolStart    map[string]int64 // tool ID → dispatch ts（duration 计算）

	// betweenSealed 标记「轮间 assistant 累积已在回合边界封口」：下一段无回合
	// 增量另起一条轮间记录，不与上一段（另一个被截断的尾部）合并——否则两轮
	// 之间的 usage 会互相覆盖成一条，用量失真。
	betweenSealed bool
}

func (f *folding) apply(e session.LogEntry) {
	switch e.Kind {
	case "turn_started":
		if f.cur != nil {
			f.turns = append(f.turns, *f.cur)
		}
		f.cur = &Turn{Turn: len(f.turns) + 1, StartedAt: e.Ts}
		f.step = 0
		f.assistantIdx = -1
		f.toolByID = nil
		f.toolStart = nil
		f.headerTs = 0
		f.betweenSealed = true
	case "user_message":
		f.applyUser(e)
	case "request_header":
		f.applyHeader(e)
	case "reasoning":
		if text := ePayloadText(e); text != "" {
			// assistantRecord 永不返回 nil（无回合时降级到轮间区段）；判空是
			// 防后续改动的兜底，绝不让 nil 走到解引用。
			if r := f.assistantRecord(e); r != nil {
				r.Assistant.Reasoning = joinPreview(r.Assistant.Reasoning, text, maxTextPreview)
			}
		}
	case "text", "message":
		if text := ePayloadText(e); text != "" {
			if r := f.assistantRecord(e); r != nil {
				r.Assistant.Text = joinPreview(r.Assistant.Text, text, maxTextPreview)
			}
		}
	case "assistant_message":
		// 迁移/投影产物的完整 assistant 消息（ToLogEntries 内嵌工具调用），
		// 运行期事件流用 "message" + tool_dispatch，此处两种形状折叠结果一致。
		f.applyAssistantMessage(e)
	case "tool_dispatch":
		f.applyToolDispatch(e)
	case "tool_result":
		f.applyToolResult(e)
	case "usage":
		f.applyUsage(e)
	case "compaction_done":
		f.applyCompaction(e)
	case "ask_request":
		f.applyAsk(e)
	case "approval_request":
		f.applyApproval(e)
	case "subagent_message":
		// v4.26 对话流式重造：子代理完成回投（旧日志无此 kind，不会进入该
		// 分支——fold 对未知 kind 的既有跳过行为不变，golden 不受影响）。
		f.applySubagentMessage(e)
	case "turn_done":
		if f.cur != nil {
			// v4.31 轮级耗时：turn_done.Ts − turn_started.Ts（秒）×1000（ms，
			// 与 Record.DurationMs 同换算先例）；Ts ≤ StartedAt（时钟异常）
			// 保持 0，omitempty 序列化省略——历史轮读盘折叠自带耗时。
			if e.Ts > f.cur.StartedAt {
				f.cur.DurationMs = (e.Ts - f.cur.StartedAt) * 1000
			}
			f.cur.End = &TurnEnd{Seq: e.Seq, Ts: e.Ts, Err: payloadTurnErr(e)}
			f.turns = append(f.turns, *f.cur)
			f.cur = nil
			f.assistantIdx = -1
			f.toolByID = nil
			f.toolStart = nil
			f.betweenSealed = true
		}
	}
}

// applyAssistantMessage 折叠迁移/投影产物的 assistant_message：正文与推理
// 并入 assistant 记录，内嵌工具调用展开为 tool 记录（与运行期 tool_dispatch
// 同一归并面，后续 tool_result 按 ID 合并）。
func (f *folding) applyAssistantMessage(e session.LogEntry) {
	var p struct {
		ID        string `json:"id,omitempty"`
		Text      string `json:"text,omitempty"`
		Reasoning string `json:"reasoning,omitempty"`
		ToolCalls []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Args string `json:"args"`
		} `json:"tool_calls,omitempty"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	if p.Text != "" {
		if r := f.assistantRecord(e); r != nil {
			r.Assistant.Text = joinPreview(r.Assistant.Text, p.Text, maxTextPreview)
		}
	}
	if p.Reasoning != "" {
		if r := f.assistantRecord(e); r != nil {
			r.Assistant.Reasoning = joinPreview(r.Assistant.Reasoning, p.Reasoning, maxTextPreview)
		}
	}
	for _, tc := range p.ToolCalls {
		if f.cur == nil || tc.ID == "" {
			continue
		}
		if f.toolByID == nil {
			f.toolByID = map[string]int{}
			f.toolStart = map[string]int64{}
		}
		r := f.toolRecordAt(tc.ID)
		if r == nil {
			r = f.appendRecord(Record{
				Seq:  e.Seq,
				Kind: "tool",
				Ts:   e.Ts,
				Step: f.step,
				Tool: &ToolRec{ID: tc.ID, Name: tc.Name, Status: "running"},
			})
			f.toolByID[tc.ID] = len(f.cur.Records) - 1
			f.toolStart[tc.ID] = e.Ts
		}
		r.Tool.Name = tc.Name
		if tc.Args != "" {
			r.Tool.Args = preview(tc.Args, maxOutPreview)
		}
	}
}

func (f *folding) applyUser(e session.LogEntry) {
	var p struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil || p.Content == "" {
		return
	}
	if f.cur == nil {
		return
	}
	f.appendRecord(Record{Seq: e.Seq, Kind: "user", Ts: e.Ts, User: &UserRec{Text: preview(p.Content, maxTextPreview)}})
}

func (f *folding) applyHeader(e session.LogEntry) {
	var p struct {
		System string `json:"system,omitempty"`
		Tools  []struct {
			Name   string `json:"name"`
			Schema string `json:"schema,omitempty"`
		} `json:"tools,omitempty"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	f.step++
	change := ""
	switch {
	case f.lastSystem == "" && f.lastTools == 0:
		change = "initial"
	case p.System != f.lastSystem && len(p.Tools) != f.lastTools:
		change = "system-and-tools"
	case p.System != f.lastSystem:
		change = "system"
	case len(p.Tools) != f.lastTools:
		change = "tools"
	}
	f.lastSystem = p.System
	f.lastTools = len(p.Tools)
	f.headerTs = e.Ts
	f.assistantIdx = -1
	var tokens int64
	tokens += estimateTokens(p.System)
	for _, t := range p.Tools {
		tokens += estimateTokens(t.Schema)
	}
	r := Record{
		Seq:  e.Seq,
		Kind: "header",
		Ts:   e.Ts,
		Step: f.step,
		Header: &HeaderRec{
			System:    preview(p.System, maxTextPreview),
			ToolCount: len(p.Tools),
			Tokens:    tokens,
			Change:    change,
		},
	}
	if f.cur != nil {
		f.appendRecord(r)
	} else {
		f.between = append(f.between, r)
	}
}

func (f *folding) applyToolDispatch(e session.LogEntry) {
	var p struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Args     string `json:"args"`
		ReadOnly bool   `json:"readOnly,omitempty"`
		Partial  bool   `json:"partial,omitempty"`
		ParentID string `json:"parentId,omitempty"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	if f.cur == nil {
		return
	}
	if f.toolByID == nil {
		f.toolByID = map[string]int{}
		f.toolStart = map[string]int64{}
	}
	r := f.toolRecordAt(p.ID)
	if r == nil {
		r = f.appendRecord(Record{
			Seq:  e.Seq,
			Kind: "tool",
			Ts:   e.Ts,
			Step: f.step,
			Tool: &ToolRec{ID: p.ID, Name: p.Name, Status: "running", ParentID: p.ParentID},
		})
		f.toolByID[p.ID] = len(f.cur.Records) - 1
		f.toolStart[p.ID] = e.Ts
	}
	r.Tool.Name = p.Name
	r.Tool.ReadOnly = p.ReadOnly
	r.Tool.ParentID = p.ParentID
	if !p.Partial && p.Args != "" {
		r.Tool.Args = preview(p.Args, maxOutPreview)
	}
}

func (f *folding) applyToolResult(e session.LogEntry) {
	var p struct {
		ID        string `json:"id"`
		Name      string `json:"name,omitempty"`
		Output    string `json:"output,omitempty"`
		Err       string `json:"err,omitempty"`
		Truncated bool   `json:"truncated,omitempty"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	if f.cur == nil {
		return
	}
	r := f.toolRecordAt(p.ID)
	if r == nil {
		r = f.appendRecord(Record{Seq: e.Seq, Kind: "tool", Ts: e.Ts, Step: f.step, Tool: &ToolRec{ID: p.ID, Name: p.Name, Status: "ok"}})
		// 既有口径（本条未动）：结果先于派发到达、或该 ID 从未派发时整体重建
		// 两张表——保持原行为，避免本条坐标外的语义漂移。
		f.toolByID = map[string]int{p.ID: len(f.cur.Records) - 1}
		f.toolStart = map[string]int64{p.ID: e.Ts}
	}
	if p.Name != "" {
		r.Tool.Name = p.Name
	}
	r.Tool.Truncated = p.Truncated
	if p.Err != "" {
		r.Tool.Err = preview(p.Err, maxOutPreview)
		r.Tool.Status = "error"
		r.Tool.Output = preview(p.Output, maxOutPreview)
	} else {
		r.Tool.Status = "ok"
		r.Tool.Output = preview(p.Output, maxOutPreview)
	}
	if start, ok := f.toolStart[p.ID]; ok && e.Ts > start {
		r.DurationMs = (e.Ts - start) * 1000
	}
}

func (f *folding) applyUsage(e session.LogEntry) {
	var p struct {
		PromptTokens     int64 `json:"promptTokens,omitempty"`
		CompletionTokens int64 `json:"completionTokens,omitempty"`
		CacheHitTokens   int64 `json:"cacheHitTokens,omitempty"`
		CacheMissTokens  int64 `json:"cacheMissTokens,omitempty"`
		ReasoningTokens  int64 `json:"reasoningTokens,omitempty"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	r := f.assistantRecord(e)
	if r == nil { // 永不发生（assistantRecord 不下传 nil）；兜底绝不解引用 nil
		return
	}
	r.Assistant.Usage = &Usage{
		PromptTokens:     p.PromptTokens,
		CompletionTokens: p.CompletionTokens,
		CacheHitTokens:   p.CacheHitTokens,
		CacheMissTokens:  p.CacheMissTokens,
		ReasoningTokens:  p.ReasoningTokens,
	}
	if f.headerTs > 0 && e.Ts > f.headerTs {
		r.DurationMs = (e.Ts - f.headerTs) * 1000
	}
}

func (f *folding) applyCompaction(e session.LogEntry) {
	var p struct {
		Trigger string `json:"trigger"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	rec := Record{
		Seq:     e.Seq,
		Kind:    "compact",
		Ts:      e.Ts,
		Compact: &CompactRec{Trigger: p.Trigger, Summary: preview(p.Summary, maxTextPreview)},
	}
	if f.cur != nil {
		f.appendRecord(rec)
	} else {
		f.between = append(f.between, rec)
	}
}

func (f *folding) applyAsk(e session.LogEntry) {
	var p struct {
		Questions []struct {
			Prompt string `json:"prompt"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	question := ""
	if len(p.Questions) > 0 {
		question = p.Questions[0].Prompt
	}
	rec := Record{
		Seq:  e.Seq,
		Kind: "ask",
		Ts:   e.Ts,
		Ask:  &AskRec{Question: preview(question, maxTextPreview)},
	}
	if f.cur != nil {
		f.appendRecord(rec)
	} else {
		f.between = append(f.between, rec)
	}
}

func (f *folding) applyApproval(e session.LogEntry) {
	var p struct {
		Tool    string `json:"tool"`
		Subject string `json:"subject"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	rec := Record{
		Seq:      e.Seq,
		Kind:     "approval",
		Ts:       e.Ts,
		Approval: &ApprovalRec{Tool: p.Tool, Subject: p.Subject},
	}
	if f.cur != nil {
		f.appendRecord(rec)
	} else {
		f.between = append(f.between, rec)
	}
}

// applySubagentMessage 折叠子代理完成回投（v4.26）：最终答复文本（展示级
// 截断，全文在日志里）+ transcript 引用 + 父 task 调用 ID。事件在 task 调用
// 执行中到达，折叠进当前回合；无回合（日志尾部悬挂）时归轮间，与 compact
// /ask/approval 同风格。
func (f *folding) applySubagentMessage(e session.LogEntry) {
	var p struct {
		Text     string `json:"text"`
		Ref      string `json:"ref,omitempty"`
		ParentID string `json:"parentId,omitempty"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return
	}
	rec := Record{
		Seq:      e.Seq,
		Kind:     "subagent",
		Ts:       e.Ts,
		Subagent: &SubagentRec{Ref: p.Ref, Text: preview(p.Text, maxTextPreview), ParentID: p.ParentID},
	}
	if f.cur != nil {
		f.appendRecord(rec)
	} else {
		f.between = append(f.between, rec)
	}
}

// assistantRecord 返回（必要时新建）当前累积的 assistant 记录，**永不返回 nil**。
//
// 有回合时按 f.assistantIdx 定位。无回合——即残缺/迁移/被截断的日志：首条即
// usage / reasoning / text 增量，或整段缺 turn_started——时如实降级：记录落进
// 轮间区段 f.between，与 applyHeader / applyCompaction / applyAsk /
// applyApproval / applySubagentMessage 的既有分流同款，而不是把 nil 交给调用点
// 去解引用（审计 GA5-03：轨迹看板 panic）。
//
// 连续多条无回合的 assistant 增量（推理 → 正文 → usage）合并进同一条轮间记录：
// 只看轮间区段尾部，不跨 header/compact/ask 等记录合并——与「一次请求一条
// assistant 记录」的轮内口径一致。
func (f *folding) assistantRecord(e session.LogEntry) *Record {
	if f.cur == nil {
		if !f.betweenSealed {
			if n := len(f.between); n > 0 && f.between[n-1].Kind == "assistant" {
				return &f.between[n-1]
			}
		}
		f.betweenSealed = false
		return f.appendRecord(Record{
			Seq: e.Seq, Kind: "assistant", Ts: e.Ts, Step: f.step, Assistant: &AssistantRec{},
		})
	}
	if r := f.curAssistant(); r != nil {
		return r
	}
	r := f.appendRecord(Record{
		Seq: e.Seq, Kind: "assistant", Ts: e.Ts, Step: f.step, Assistant: &AssistantRec{},
	})
	f.assistantIdx = len(f.cur.Records) - 1
	return r
}

// curAssistant 按下标取当前累积的 assistant 记录。下标越界、错位（零值 folding
// 或后续改动）或指向的不是 assistant 记录时返回 nil，由调用点走新建分支——
// 绝不解引用失效下标。
func (f *folding) curAssistant() *Record {
	if f.cur == nil || f.assistantIdx < 0 || f.assistantIdx >= len(f.cur.Records) {
		return nil
	}
	if f.cur.Records[f.assistantIdx].Kind != "assistant" {
		return nil
	}
	return &f.cur.Records[f.assistantIdx]
}

// toolRecordAt 按 ID 取当前回合里对应的工具记录。toolByID 存的是下标而不是
// *Record（见 folding 字段注释：切片增长会让元素指针失效）；下标越界或指向的
// 不是工具记录时返回 nil，由调用点走新建分支。
func (f *folding) toolRecordAt(id string) *Record {
	if f.cur == nil || f.toolByID == nil {
		return nil
	}
	i, ok := f.toolByID[id]
	if !ok || i < 0 || i >= len(f.cur.Records) {
		return nil
	}
	if f.cur.Records[i].Kind != "tool" {
		return nil
	}
	return &f.cur.Records[i]
}

// appendRecord 追加记录并返回它——调用方直接用返回值，不要再自己取
// `f.cur.Records[len(f.cur.Records)-1]` 下标（长度依赖会随改动失效）。
// 无当前回合（残缺日志）时不解引用 nil：如实降级到轮间区段。
func (f *folding) appendRecord(r Record) *Record {
	if f.cur == nil {
		f.between = append(f.between, r)
		return &f.between[len(f.between)-1]
	}
	f.cur.Records = append(f.cur.Records, r)
	return &f.cur.Records[len(f.cur.Records)-1]
}

func preview(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func joinPreview(a, b string, max int) string {
	if a == "" {
		return preview(b, max)
	}
	return preview(a+b, max)
}

func ePayloadText(e session.LogEntry) string {
	var p struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		return ""
	}
	return p.Text
}

func payloadTurnErr(e session.LogEntry) string {
	var p struct {
		Err string `json:"err"`
	}
	_ = json.Unmarshal(e.Payload, &p)
	return p.Err
}

// estimateTokens 按仓库既有口径估算文本 token 数（bytes÷4 floor；批 52 收口
// 转调 util.EstimateTokensLen 单源）。
func estimateTokens(s string) int64 {
	return util.EstimateTokensLen(s)
}
