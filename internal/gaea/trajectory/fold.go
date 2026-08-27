package trajectory

import (
	"encoding/json"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// 预览上限：轨迹是浏览视图，正文/推理/输出做展示级截断（全文在日志里）。
const (
	maxTextPreview = 2000
	maxOutPreview  = 4000
)

// FoldTrajectory 把会话日志条目折叠为轨迹快照。纯函数：同输入必同输出。
func FoldTrajectory(entries []session.LogEntry) Trajectory {
	f := &folding{}
	for _, e := range entries {
		f.apply(e)
	}
	if f.cur != nil {
		f.turns = append(f.turns, *f.cur)
	}
	return Trajectory{Ok: true, Turns: f.turns, BetweenTurns: f.between}
}

type folding struct {
	turns   []Turn
	between []Record
	cur     *Turn
	step    int

	lastSystem string // 上一个 request_header 的 system（change 检测）
	lastTools  int

	headerTs  int64
	assistant *Record          // 正在累积的 assistant 记录
	toolByID  map[string]*Record // tool ID → 记录
	toolStart map[string]int64 // tool ID → dispatch ts（duration 计算）
}

func (f *folding) apply(e session.LogEntry) {
	switch e.Kind {
	case "turn_started":
		if f.cur != nil {
			f.turns = append(f.turns, *f.cur)
		}
		f.cur = &Turn{Turn: len(f.turns) + 1, StartedAt: e.Ts}
		f.step = 0
		f.assistant = nil
		f.toolByID = nil
		f.toolStart = nil
		f.headerTs = 0
	case "user_message":
		f.applyUser(e)
	case "request_header":
		f.applyHeader(e)
	case "reasoning":
		if text := ePayloadText(e); text != "" {
			r := f.assistantRecord(e)
			r.Assistant.Reasoning = joinPreview(r.Assistant.Reasoning, text, maxTextPreview)
		}
	case "text", "message":
		if text := ePayloadText(e); text != "" {
			r := f.assistantRecord(e)
			r.Assistant.Text = joinPreview(r.Assistant.Text, text, maxTextPreview)
		}
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
	case "turn_done":
		if f.cur != nil {
			f.cur.End = &TurnEnd{Seq: e.Seq, Ts: e.Ts, Err: payloadTurnErr(e)}
			f.turns = append(f.turns, *f.cur)
			f.cur = nil
			f.assistant = nil
			f.toolByID = nil
			f.toolStart = nil
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
	f.assistant = nil
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
		f.toolByID = map[string]*Record{}
		f.toolStart = map[string]int64{}
	}
	r, ok := f.toolByID[p.ID]
	if !ok {
		r = &Record{
			Seq:  e.Seq,
			Kind: "tool",
			Ts:   e.Ts,
			Step: f.step,
			Tool: &ToolRec{ID: p.ID, Name: p.Name, Status: "running", ParentID: p.ParentID},
		}
		f.appendRecord(*r)
		r = &f.cur.Records[len(f.cur.Records)-1]
		f.toolByID[p.ID] = r
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
	var r *Record
	if f.toolByID != nil {
		r = f.toolByID[p.ID]
	}
	if r == nil {
		r = &Record{Seq: e.Seq, Kind: "tool", Ts: e.Ts, Step: f.step, Tool: &ToolRec{ID: p.ID, Name: p.Name, Status: "ok"}}
		f.appendRecord(*r)
		r = &f.cur.Records[len(f.cur.Records)-1]
		f.toolByID = map[string]*Record{p.ID: r}
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

// assistantRecord 返回（必要时新建）当前累积的 assistant 记录。
func (f *folding) assistantRecord(e session.LogEntry) *Record {
	if f.cur == nil {
		return nil
	}
	if f.assistant == nil {
		r := Record{Seq: e.Seq, Kind: "assistant", Ts: e.Ts, Step: f.step, Assistant: &AssistantRec{}}
		f.appendRecord(r)
		f.assistant = &f.cur.Records[len(f.cur.Records)-1]
	}
	return f.assistant
}

func (f *folding) appendRecord(r Record) {
	f.cur.Records = append(f.cur.Records, r)
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

func estimateTokens(s string) int64 {
	return int64(float64(len(s)) * 0.25)
}
