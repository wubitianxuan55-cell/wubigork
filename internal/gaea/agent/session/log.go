package session

// 3.0 Step 1: 会话事件日志（append-only）。
// 日志文件 <sessionDir>/<id>.gaea-log.jsonl，每行一条：
//   {"seq":N,"ts":<unix秒>,"kind":"...","payload":{...}}
// 硬不变量：
//   - seq = 已写行数（写入器单点保证，Append 持锁、单次 Write 追加）；
//   - payload 必须无损 JSON（json.Valid 校验，非法拒绝写入并报错）；
//   - 「模型可见必入日志」是不变量而非约定（事件 sink 在转发前落盘）。

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// ─── 日志事件种类 ───────────────────────────────────────────────

// 消息级补充事件（会话语义，非 event.Kind）：user/assistant/tool 三类
// 投影为 provider.Message 的基础。
const (
	KindUserMessage      = "user_message"
	KindAssistantStarted = "assistant_started"
	KindAssistantDelta   = "assistant_delta"
	KindAssistantMessage = "assistant_message"
	KindToolCall         = "tool_call"
	KindToolResult       = "tool_result"
	// system_message 仅用于旧格式迁移：旧 JSONL 里的 system 消息没有对应事件，
	// 迁移时原样落日志，投影时还原为 system 消息，保证恢复后的会话提示词不丢。
	KindSystemMessage = "system_message"
	// KindRequestHeader 是 request_header 事件：一次模型请求组装后的
	// system prompt 与工具 schema（context-view 折叠的 system/tools 数据源）。
	KindRequestHeader = "request_header"
)

// KindString 把 event.Kind 映射为日志 kind 字符串。
func KindString(k event.Kind) string {
	switch k {
	case event.TurnStarted:
		return "turn_started"
	case event.Reasoning:
		return "reasoning"
	case event.Text:
		return "text"
	case event.Message:
		return "message"
	case event.ToolDispatch:
		return "tool_dispatch"
	case event.ToolResult:
		return KindToolResult
	case event.Usage:
		return "usage"
	case event.Notice:
		return "notice"
	case event.Phase:
		return "phase"
	case event.ApprovalRequest:
		return "approval_request"
	case event.AskRequest:
		return "ask_request"
	case event.TurnDone:
		return "turn_done"
	case event.CompactionStarted:
		return "compaction_started"
	case event.CompactionDone:
		return "compaction_done"
	case event.RequestHeader:
		return KindRequestHeader
	case event.Retrying:
		return "retrying"
	case event.Steer:
		return "steer"
	}
	return "unknown"
}

// ─── 日志条目 ───────────────────────────────────────────────────

// LogEntry 是日志中的一行。Payload 保持原始 JSON 无损（不重新编解码）。
type LogEntry struct {
	Seq  int64           `json:"seq"`
	Ts   int64           `json:"ts"`
	Kind string          `json:"kind"`
	// Space 是会话空间自描述（S2 双空间）：写端由 LogWriter 按会话目录归属
	// 写入（"work"/"play"）；space.mode=off 的平铺日志无此字段。读端空值一律
	// 降级 work（spaces.SpaceOr），且保持原始空值以便 RewindLog 逐字节重写。
	Space   string          `json:"space,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

// userLogPayload 是 user_message / system_message 的 payload。
type userLogPayload struct {
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// requestHeaderLogPayload 是 request_header 事件的 payload：一次模型请求
// 组装后的 system prompt 与工具 schema（context-view 折叠的数据源）。
type requestHeaderLogPayload struct {
	System string                  `json:"system,omitempty"`
	Tools  []requestToolLogPayload `json:"tools,omitempty"`
	Window int                     `json:"window,omitempty"`
}

// requestToolLogPayload 是 request_header 中单个工具 schema。
type requestToolLogPayload struct {
	Name   string `json:"name"`
	Schema string `json:"schema,omitempty"`
}

// toolCallLogPayload 是 tool_call / tool_dispatch 事件中的一次工具调用。
type toolCallLogPayload struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Args     string `json:"args"`
	ReadOnly bool   `json:"readOnly,omitempty"`
	Partial  bool   `json:"partial,omitempty"`
	ParentID string `json:"parentId,omitempty"`
}

// assistantLogPayload 是 assistant_message / message 事件的 payload。
type assistantLogPayload struct {
	ID                 string               `json:"id,omitempty"`
	Text               string               `json:"text,omitempty"`
	Reasoning          string               `json:"reasoning,omitempty"`
	ReasoningSignature string               `json:"reasoning_signature,omitempty"`
	ToolCalls          []toolCallLogPayload `json:"tool_calls,omitempty"`
}

// toolResultLogPayload 是 tool_result 事件的 payload。
type toolResultLogPayload struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Output    string `json:"output,omitempty"`
	Err       string `json:"err,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

// usageLogPayload 是 usage 事件的 payload（token 统计/成本派生 API 的输入）。
type usageLogPayload struct {
	PromptTokens           int64   `json:"promptTokens,omitempty"`
	CompletionTokens       int64   `json:"completionTokens,omitempty"`
	TotalTokens            int64   `json:"totalTokens,omitempty"`
	CacheHitTokens         int64   `json:"cacheHitTokens,omitempty"`
	CacheMissTokens        int64   `json:"cacheMissTokens,omitempty"`
	ReasoningTokens        int64   `json:"reasoningTokens,omitempty"`
	SessionCacheHitTokens  int64   `json:"sessionCacheHitTokens,omitempty"`
	SessionCacheMissTokens int64   `json:"sessionCacheMissTokens,omitempty"`
	Input                  float64 `json:"input,omitempty"`
	Output                 float64 `json:"output,omitempty"`
	CacheHitPrice          float64 `json:"cacheHitPrice,omitempty"`
	Currency               string  `json:"currency,omitempty"`
	Source                 string  `json:"source,omitempty"`
	Turn                   int     `json:"turn,omitempty"`
}

// ─── 日志路径 ───────────────────────────────────────────────────

// LogPathFor 返回会话文件对应的事件日志路径：<dir>/<id>.gaea-log.jsonl。
// 空会话路径返回空串。
func LogPathFor(sessionPath string) string {
	if sessionPath == "" {
		return ""
	}
	dir := filepath.Dir(sessionPath)
	base := BranchID(sessionPath)
	if base == "" {
		return ""
	}
	return filepath.Join(dir, base+".gaea-log.jsonl")
}

// CheckpointPathFor 返回会话对应的检查点路径：<dir>/<id>.gaea-checkpoint.json。
func CheckpointPathFor(sessionPath string) string {
	if sessionPath == "" {
		return ""
	}
	dir := filepath.Dir(sessionPath)
	base := BranchID(sessionPath)
	if base == "" {
		return ""
	}
	return filepath.Join(dir, base+".gaea-checkpoint.json")
}

// ─── append-only 写入器 ─────────────────────────────────────────

// LogWriter 是单点顺序写入器：Append 持锁串行，seq = 已写行数。
// 打开时自动修复 torn-tail（截断最后的不完整行），保证后续 seq 连续。
type LogWriter struct {
	mu     sync.Mutex
	f      *os.File
	path   string
	// space 是本日志的空间自描述值（"work"/"play"；""=不写 space 字段，
	// space.mode=off 平铺日志的旧行为形态）。由 OpenLog 在打开时确定。
	space  string
	seq    int64
	closed bool
}

// OpenLog 打开（必要时创建）一个事件日志。若文件已存在，先修复 torn-tail
// 并把 seq 续到已写行数；不存在时新建。legacyPath 非空且日志不存在、而
// 旧格式会话文件存在时，自动把旧消息迁移进新日志（旧文件保留）。
// space 是会话空间自描述值（"work"/"play"；""=不写 space 字段），随每行
// 日志统一写入（AppendRaw/formatLogLine 单点）。
func OpenLog(logPath, legacyPath, space string) (*LogWriter, error) {
	if logPath == "" {
		return nil, errors.New("empty log path")
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir log dir: %w", err)
	}

	existed := false
	if st, err := os.Stat(logPath); err == nil {
		existed = true
		if st.Size() > 0 {
			if _, err := RepairLogFile(logPath); err != nil {
				return nil, fmt.Errorf("repair log: %w", err)
			}
		}
	}

	// 首次写入且存在旧格式会话：迁移旧消息为初始日志条目（旧文件保留）。
	if !existed && legacyPath != "" {
		if _, err := os.Stat(legacyPath); err == nil {
			if _, err := MigrateLegacyToLog(logPath, legacyPath, space); err != nil {
				return nil, fmt.Errorf("migrate legacy session: %w", err)
			}
		}
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log: %w", err)
	}
	w := &LogWriter{f: f, path: logPath, space: space}
	w.seq = countLogLines(logPath)
	return w, nil
}

// Append 追加一条事件并返回其 seq。payload 必须可无损序列化为 JSON；
// 若 marshaled 结果非法（json.Valid 不过），拒绝写入并返回错误。
func (w *LogWriter) Append(kind string, payload any) (int64, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}
	return w.AppendRaw(kind, raw)
}

// AppendRaw 追加一条已序列化的 payload。json.Valid 校验失败时拒绝写入。
func (w *LogWriter) AppendRaw(kind string, raw json.RawMessage) (int64, error) {
	if !json.Valid(raw) {
		return 0, fmt.Errorf("reject invalid payload (not lossless JSON): %s", truncateForError(raw))
	}
	if kind == "" {
		return 0, errors.New("empty log kind")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, errors.New("log writer closed")
	}
	line := formatLogLine(w.seq+1, time.Now().Unix(), kind, w.space, raw)
	n, err := w.f.Write(line)
	if err != nil {
		return 0, fmt.Errorf("append log line: %w", err)
	}
	if n != len(line) {
		return 0, fmt.Errorf("short write: %d of %d bytes", n, len(line))
	}
	w.seq++
	return w.seq, nil
}

// Seq 返回当前已写行数（下一个 seq 的前驱）。
func (w *LogWriter) Seq() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.seq
}

// Close 关闭底层文件。
func (w *LogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	return w.f.Close()
}

// ─── 读取与修复 ─────────────────────────────────────────────────

// ReadLog 读回全部完整行。torn-tail 的修复由 RepairLogFile / ReadLogRepaired 负责；
// 本函数对不完整尾部行返回错误（调用方自行决定策略）。
func ReadLog(path string) ([]LogEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseLogBytes(data)
}

// ReadLogRepaired 先修复 torn-tail（截断最后的不完整行）再读回。
func ReadLogRepaired(path string) ([]LogEntry, error) {
	if _, err := RepairLogFile(path); err != nil {
		return nil, err
	}
	return ReadLog(path)
}

// RepairLogFile 截断文件末尾的不完整行（最后一个换行之后没有换行的残留）。
// 返回是否发生了截断。文件不存在返回 false, nil；文件以换行结尾或为空时不截断。
func RepairLogFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if len(data) == 0 || data[len(data)-1] == '\n' {
		return false, nil
	}
	last := bytesLastIndexByte(data, '\n')
	if err := os.Truncate(path, int64(last+1)); err != nil {
		return false, fmt.Errorf("truncate torn tail: %w", err)
	}
	return true, nil
}

// BalanceEntries 为「完整中断」的 turn 补合成 closers：若流末尾不是 turn_done，
// 追加一条合成 turn_done（不写盘，仅在内存重放流中补平衡）。
func BalanceEntries(entries []LogEntry) []LogEntry {
	if len(entries) == 0 {
		return entries
	}
	last := entries[len(entries)-1]
	if last.Kind == "turn_done" {
		return entries
	}
	cp := append([]LogEntry(nil), entries...)
	cp = append(cp, LogEntry{
		Seq:  last.Seq + 1,
		Ts:   last.Ts,
		Kind: "turn_done",
		Payload: mustMarshal(map[string]any{
			"synthetic": true,
			"reason":    "interrupted-turn",
		}),
	})
	return cp
}

// parseLogBytes 按行切分并解码（逐行构建字节切片，不受 Scanner 行缓冲上限约束）。
func parseLogBytes(data []byte) ([]LogEntry, error) {
	var entries []LogEntry
	start := 0
	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == '\n' {
			if i > start {
				var e LogEntry
				if err := json.Unmarshal(data[start:i], &e); err != nil {
					return nil, fmt.Errorf("decode log line: %w", err)
				}
				entries = append(entries, e)
			}
			start = i + 1
		}
	}
	return entries, nil
}

// countLogLines 统计文件中的完整行数（修复后的日志行数 = 下一个 seq 的前驱）。
func countLogLines(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	entries, err := parseLogBytes(data)
	if err != nil {
		return 0
	}
	return int64(len(entries))
}

// formatLogLine 组装一行日志（单次 Write 原子追加）。space 为空时省略
// space 字段（space.mode=off 平铺日志与旧行为逐字节一致）。
func formatLogLine(seq, ts int64, kind, space string, raw json.RawMessage) []byte {
	line := struct {
		Seq     int64           `json:"seq"`
		Ts      int64           `json:"ts"`
		Kind    string          `json:"kind"`
		Space   string          `json:"space,omitempty"`
		Payload json.RawMessage `json:"payload"`
	}{seq, ts, kind, space, raw}
	b, err := json.Marshal(line)
	if err != nil {
		panic(err) // 内部固定形状，不可能失败
	}
	return append(b, '\n')
}

// mustMarshal 编码 payload；失败 panic（仅用于合成条目等内部固定形状）。
func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func bytesLastIndexByte(b []byte, c byte) int {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// truncateForError 截断非法 payload 的预览用于报错（不超过 120 字符）。
func truncateForError(raw json.RawMessage) string {
	s := string(raw)
	r := []rune(s)
	if len(r) > 120 {
		return string(r[:120]) + "…"
	}
	return s
}

// EntryFromEvent 把一条 event.Event 转换为日志条目（事件级映射）。
// 16 种 event.Kind 全部落日志；投影时仅 user/assistant/tool 三类还原为消息。
func EntryFromEvent(e event.Event, ts int64) (LogEntry, error) {
	kind := KindString(e.Kind)
	var payload any
	switch e.Kind {
	case event.TurnStarted, event.Steer:
		payload = map[string]string{}
	case event.Reasoning, event.Text, event.Phase, event.Message:
		payload = map[string]string{"text": e.Text, "reasoning": e.Reasoning}
	case event.ToolDispatch:
		payload = toolCallLogPayload{
			ID:       e.Tool.ID,
			Name:     e.Tool.Name,
			Args:     e.Tool.Args,
			ReadOnly: e.Tool.ReadOnly,
			Partial:  e.Tool.Partial,
			ParentID: e.Tool.ParentID,
		}
	case event.ToolResult:
		payload = toolResultLogPayload{
			ID:        e.Tool.ID,
			Name:      e.Tool.Name,
			Output:    e.Tool.Output,
			Err:       e.Tool.Err,
			Truncated: e.Tool.Truncated,
		}
	case event.Usage:
		payload = usagePayloadFromEvent(e)
	case event.Notice:
		level := "info"
		if e.Level == event.LevelWarn {
			level = "warn"
		}
		payload = map[string]any{"level": level, "text": e.Text}
	case event.ApprovalRequest:
		payload = map[string]any{"id": e.Approval.ID, "tool": e.Approval.Tool, "subject": e.Approval.Subject}
	case event.AskRequest:
		payload = askPayloadFromEvent(e)
	case event.TurnDone:
		errText := ""
		if e.Err != nil {
			errText = e.Err.Error()
		}
		payload = map[string]string{"err": errText}
	case event.CompactionStarted, event.CompactionDone:
		payload = map[string]any{
			"trigger":  e.Compaction.Trigger,
			"messages": e.Compaction.Messages,
			"summary":  e.Compaction.Summary,
			"archive":  e.Compaction.Archive,
			"quality":  e.Compaction.Quality,
		}
	case event.RequestHeader:
		tools := make([]requestToolLogPayload, 0, len(e.Header.Tools))
		for _, t := range e.Header.Tools {
			tools = append(tools, requestToolLogPayload{Name: t.Name, Schema: t.Schema})
		}
		payload = requestHeaderLogPayload{
			System: e.Header.System,
			Tools:  tools,
			Window: e.Header.Window,
		}
	case event.Retrying:
		payload = map[string]int{"attempt": e.RetryAttempt, "max": e.RetryMax}
	default:
		payload = map[string]string{}
	}
	return LogEntry{Seq: 0, Ts: ts, Kind: kind, Payload: mustMarshal(payload)}, nil
}

// usagePayloadFromEvent 把 Usage 事件 + Pricing 折叠为无损 payload。
func usagePayloadFromEvent(e event.Event) usageLogPayload {
	var (
		prompt, completion, total      int64
		cacheHit, cacheMiss, reasoning int64
		sessHit, sessMiss              int64
	)
	if e.Usage != nil {
		prompt = int64(e.Usage.PromptTokens)
		completion = int64(e.Usage.CompletionTokens)
		total = int64(e.Usage.TotalTokens)
		cacheHit = int64(e.Usage.CacheHitTokens)
		cacheMiss = int64(e.Usage.CacheMissTokens)
		reasoning = int64(e.Usage.ReasoningTokens)
		sessHit = int64(e.Usage.SessionCacheHitTokens)
		sessMiss = int64(e.Usage.SessionCacheMissTokens)
	}
	p := usageLogPayload{
		PromptTokens:           prompt,
		CompletionTokens:       completion,
		TotalTokens:            total,
		CacheHitTokens:         cacheHit,
		CacheMissTokens:        cacheMiss,
		ReasoningTokens:        reasoning,
		SessionCacheHitTokens:  sessHit,
		SessionCacheMissTokens: sessMiss,
		Source:                 e.UsageSource,
		Turn:                   e.Turn,
	}
	if e.Pricing != nil {
		p.Input = e.Pricing.Input
		p.Output = e.Pricing.Output
		p.CacheHitPrice = e.Pricing.CacheHit
		p.Currency = e.Pricing.Currency
	}
	return p
}

// askPayloadFromEvent 把 AskRequest 折叠为无损 payload。
func askPayloadFromEvent(e event.Event) map[string]any {
	qs := make([]map[string]any, 0, len(e.Ask.Questions))
	for _, q := range e.Ask.Questions {
		opts := make([]map[string]string, 0, len(q.Options))
		for _, o := range q.Options {
			opts = append(opts, map[string]string{"label": o.Label, "description": o.Description})
		}
		qs = append(qs, map[string]any{
			"id": q.ID, "header": q.Header, "prompt": q.Prompt, "options": opts, "multi": q.Multi,
		})
	}
	m := map[string]any{"id": e.Ask.ID, "questions": qs}
	return m
}

// ─── 消息 → 日志条目（迁移/落盘用） ──────────────────────────────

// ToLogEntries 把 provider.Message 列表序列化为日志条目（旧格式迁移与
// checkpoint 投影时使用）。system 消息映射为 system_message；assistant 的
// 工具调用随 assistant_message 内嵌（与 runtime 事件流 tool_dispatch 等价的
// 投影结果一致）。
//
// 条目带回合边界：每条 user 消息前写 turn_started、下一条 user 消息前
// （或流末尾）写 turn_done——让迁移/投影产生的日志对轨迹折叠（trajectory）
// 与上下文折叠（contextview）同样成立。边界条目不投影为消息，恢复路径
// 逐字节不受影响。
//
// 每个回合额外合成一条 request_header：system = 已见 system 消息拼接，tools =
// 该轮 assistant 实际用到的工具名集合（schema 未知，只给名字的最小诚实形状）。
// 旧会话因此能得到系统/工具分类估算与趋势柱（无 usage 事件时 contextview
// 在 turn_done 用估算关闭请求，见 Estimated）。
func ToLogEntries(msgs []provider.Message) []LogEntry {
	// 预扫描：每条 user 消息序号 → 该轮使用的工具名集合（去重保序）。
	turnToolNames := map[int][]string{}
	turnIdx := -1
	for _, m := range msgs {
		if m.Role == provider.RoleUser {
			turnIdx++
		}
		if m.Role != provider.RoleAssistant || turnIdx < 0 {
			continue
		}
		for _, tc := range m.ToolCalls {
			if tc.Name == "" {
				continue
			}
			dup := false
			for _, n := range turnToolNames[turnIdx] {
				if n == tc.Name {
					dup = true
					break
				}
			}
			if !dup {
				turnToolNames[turnIdx] = append(turnToolNames[turnIdx], tc.Name)
			}
		}
	}
	var out []LogEntry
	now := time.Now().Unix()
	seq := int64(1)
	appendEntry := func(e LogEntry) {
		e.Seq = seq
		e.Ts = now
		out = append(out, e)
		seq++
	}
	turnOpen := false
	closeTurn := func() {
		if !turnOpen {
			return
		}
		appendEntry(LogEntry{Kind: "turn_done", Payload: mustMarshal(map[string]string{})})
		turnOpen = false
	}
	systemText := ""
	curTurn := -1
	for _, m := range msgs {
		switch m.Role {
		case provider.RoleSystem:
			systemText = joinPromptPart(systemText, m.Content)
			appendEntry(LogEntry{Kind: KindSystemMessage,
				Payload: mustMarshal(userLogPayload{Content: m.Content, Name: m.Name})})
		case provider.RoleUser:
			closeTurn()
			appendEntry(LogEntry{Kind: "turn_started", Payload: mustMarshal(map[string]string{})})
			turnOpen = true
			curTurn++
			appendEntry(LogEntry{Kind: KindUserMessage,
				Payload: mustMarshal(userLogPayload{Content: m.Content, Name: m.Name})})
			// 合成 request_header：system prompt（真实）+ 该轮工具名集合
			// （schema 未知的最小诚实形状，估算 token 用）。放在 user 之后，
			// 与运行期事件序一致（用户消息先落日志、模型请求再发 header）。
			if systemText != "" || len(turnToolNames[curTurn]) > 0 {
				tools := make([]requestToolLogPayload, 0, len(turnToolNames[curTurn]))
				for _, n := range turnToolNames[curTurn] {
					tools = append(tools, requestToolLogPayload{Name: n})
				}
				appendEntry(LogEntry{Kind: KindRequestHeader,
					Payload: mustMarshal(requestHeaderLogPayload{System: systemText, Tools: tools})})
			}
		case provider.RoleAssistant:
			tcs := make([]toolCallLogPayload, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, toolCallLogPayload{ID: tc.ID, Name: tc.Name, Args: tc.Arguments})
			}
			appendEntry(LogEntry{Kind: KindAssistantMessage,
				Payload: mustMarshal(assistantLogPayload{
					ID:                 m.ToolCallID,
					Text:               m.Content,
					Reasoning:          m.ReasoningContent,
					ReasoningSignature: m.ReasoningSignature,
					ToolCalls:          tcs,
				})})
		case provider.RoleTool:
			appendEntry(LogEntry{Kind: KindToolResult,
				Payload: mustMarshal(toolResultLogPayload{
					ID: m.ToolCallID, Name: m.Name, Output: m.Content,
				})})
		default:
			continue // 未知角色不落日志（保无损，拒绝猜测）
		}
	}
	closeTurn()
	return out
}

// joinPromptPart 拼接 system prompt 片段（与运行时 systemPromptFromMessages
// 同风格：非空片段间空行分隔）。
func joinPromptPart(a, b string) string {
	if b == "" {
		return a
	}
	if a == "" {
		return b
	}
	return a + "\n\n" + b
}
