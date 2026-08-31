package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// TodoItem mirrors the todo_write item shape the host needs for step matching.
type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm,omitempty"`
	Level      int    `json:"level,omitempty"`
}

// TodoStepMatch is the result of matching complete_step.step against the latest
// successful todo_write list in this turn.
type TodoStepMatch struct {
	Found      bool
	Index      int
	Content    string
	Status     string
	ActiveForm string
}

// Receipt is the host-runtime record of one tool call. It stays in memory for
// the current agent turn and is not serialized into prompts or session state.
type Receipt struct {
	ToolName string          `json:"tool_name"`
	Args     json.RawMessage `json:"args,omitempty"`
	Success  bool            `json:"success"`
	Command  string          `json:"command,omitempty"`
	Step     string          `json:"step,omitempty"`
	TodoStep *TodoStepMatch  `json:"todo_step,omitempty"`
	Paths    []string        `json:"paths,omitempty"`
	Read     bool            `json:"read,omitempty"`
	Write    bool            `json:"write,omitempty"`
	Todos    []TodoItem      `json:"todos,omitempty"`
}

// Ledger stores the receipts available to complete_step for the current turn.
type Ledger struct {
	mu           sync.Mutex
	receipts     []Receipt
	strictVerify bool // V10.8: only enforce complete_step evidence in Plan Mode
}

func NewLedger() *Ledger { return &Ledger{} }

// SetStrictVerification enables/disables strict evidence verification.
// Should be set to true only in Plan Mode where complete_step receipts are required.
func (l *Ledger) SetStrictVerification(v bool) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.strictVerify = v
}

// StrictVerification reports whether strict evidence checking is enabled.
func (l *Ledger) StrictVerification() bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.strictVerify
}

// Reset clears receipts between user turns.
func (l *Ledger) Reset() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.receipts = nil
}

// Record appends a receipt. Failed receipts are retained for auditability but
// are never accepted by the HasSuccessful* matchers.
func (l *Ledger) Record(r Receipt) {
	if l == nil {
		return
	}
	r.Command = strings.TrimSpace(r.Command)
	r.Step = strings.TrimSpace(r.Step)
	r.Paths = normalizePaths(r.Paths)
	r.Todos = normalizeTodos(r.Todos)
	if r.Args != nil {
		cp := make(json.RawMessage, len(r.Args))
		copy(cp, r.Args)
		r.Args = cp
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if r.Success && r.ToolName == "complete_step" && r.Step != "" && r.TodoStep == nil {
		if match := latestTodoStep(r.Step, l.receipts); match.Found {
			r.TodoStep = &match
		}
	}
	l.receipts = append(l.receipts, r)
}

func (l *Ledger) HasSuccessfulCommand(command string) bool {
	command = strings.TrimSpace(command)
	if l == nil || command == "" {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.receipts {
		if r.Success && r.ToolName == "bash" && r.Command == command {
			return true
		}
	}
	return false
}

// SuccessfulCommands returns up to limit distinct successful bash commands
// recorded this turn, for use in diagnostic hints.
func (l *Ledger) SuccessfulCommands(limit int) []string {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	seen := map[string]bool{}
	var cmds []string
	for _, r := range l.receipts {
		if !r.Success || r.Command == "" || r.ToolName != "bash" || seen[r.Command] {
			continue
		}
		seen[r.Command] = true
		cmds = append(cmds, r.Command)
		if len(cmds) >= limit {
			break
		}
	}
	return cmds
}

func (l *Ledger) HasSuccessfulWrite(paths []string) bool {
	return l.hasSuccessfulPaths(paths, func(r Receipt) bool { return r.Write })
}

func (l *Ledger) HasSuccessfulReadOrWrite(paths []string) bool {
	return l.hasSuccessfulPaths(paths, func(r Receipt) bool { return r.Read || r.Write })
}

func (l *Ledger) MatchLatestTodoStep(step string) (TodoStepMatch, bool) {
	step = strings.TrimSpace(step)
	if l == nil || step == "" {
		return TodoStepMatch{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := len(l.receipts) - 1; i >= 0; i-- {
		r := l.receipts[i]
		if !r.Success || r.ToolName != "todo_write" {
			continue
		}
		return matchTodoStep(step, r.Todos), true
	}
	return TodoStepMatch{}, false
}

// UnverifiedCompletedTodos reports current completed todos that transitioned
// from the latest prior successful todo_write receipt without a matching
// successful complete_step receipt earlier in the same turn. If this turn has no
// prior todo_write baseline, hasBaseline is false and callers should preserve
// the existing loose validation behavior.
func (l *Ledger) UnverifiedCompletedTodos(current []TodoItem) (missing []TodoStepMatch, hasBaseline bool) {
	current = normalizeTodos(current)
	if l == nil {
		return nil, false
	}

	l.mu.Lock()
	receipts := append([]Receipt(nil), l.receipts...)
	l.mu.Unlock()

	var previous []TodoItem
	for i := len(receipts) - 1; i >= 0; i-- {
		r := receipts[i]
		if !r.Success || r.ToolName != "todo_write" {
			continue
		}
		previous = r.Todos
		hasBaseline = true
		break
	}
	if !hasBaseline {
		return nil, false
	}

	for i, t := range current {
		if todoStatus(t.Status) != "completed" {
			continue
		}
		index := i + 1
		if previousTodoCompleted(index, t, previous) {
			continue
		}
		if hasSuccessfulCompleteStepForTodo(receipts, index, current) {
			continue
		}
		missing = append(missing, TodoStepMatch{
			Found:      true,
			Index:      index,
			Content:    t.Content,
			Status:     todoStatus(t.Status),
			ActiveForm: t.ActiveForm,
		})
	}
	return missing, true
}

func (l *Ledger) hasSuccessfulPaths(paths []string, accept func(Receipt) bool) bool {
	wanted := pathSet(normalizePaths(paths))
	if l == nil || len(wanted) == 0 {
		return false
	}
	found := map[string]bool{}

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.receipts {
		if !r.Success || !accept(r) {
			continue
		}
		for _, p := range r.Paths {
			if _, ok := wanted[p]; ok {
				found[p] = true
			}
		}
	}
	return len(found) == len(wanted)
}

type contextKey struct{}

func WithLedger(ctx context.Context, ledger *Ledger) context.Context {
	if ledger == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, ledger)
}

func FromContext(ctx context.Context) (*Ledger, bool) {
	ledger, ok := ctx.Value(contextKey{}).(*Ledger)
	return ledger, ok && ledger != nil
}

func ReceiptFromToolCall(toolName string, args json.RawMessage, success bool, readOnly bool) Receipt {
	r := Receipt{
		ToolName: toolName,
		Args:     args,
		Success:  success,
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(args, &fields); err == nil {
		if toolName == "bash" {
			r.Command = stringField(fields, "command")
		}
		if toolName == "complete_step" {
			r.Step = stringField(fields, "step")
			// Also extract step_index (1-based integer) if step is empty.
			if r.Step == "" {
				if raw, ok := fields["step_index"]; ok {
					var idx float64
					if json.Unmarshal(raw, &idx) == nil && idx > 0 {
						r.Step = fmt.Sprintf("%d", int(idx))
					}
				}
			}
		}
		if toolName == "todo_write" {
			r.Todos = todoItemsField(fields, "todos")
		}
		r.Paths = extractPaths(fields)
	}

	if isWriterTool(toolName) {
		r.Write = true
	} else if isReaderTool(toolName) || (readOnly && len(r.Paths) > 0) {
		r.Read = true
	}
	return r
}

func isWriterTool(name string) bool {
	switch name {
	case "write_file", "edit_file", "multi_edit", "edit_lines", "move_file", "notebook_edit", "delete_range", "delete_symbol":
		return true
	default:
		return false
	}
}

func isReaderTool(name string) bool {
	switch name {
	case "read_file", "ls", "grep":
		return true
	default:
		return false
	}
}

// isProducerTool 报告工具是否为「生成/导出类」产物工具：产物路径出现在
// output 参数中（format_convert 导出 Markdown、chart_gen/diagram_gen 落盘
// png/svg）。v4.23 前的前端启发式（WRITE_TOOL_NAMES + 正文扩展名白名单）
// 不覆盖这三类，是产物漏登的主因之一，v4.24 权威产物登记表显式纳入。
func isProducerTool(name string) bool {
	switch name {
	case "format_convert", "chart_gen", "diagram_gen":
		return true
	default:
		return false
	}
}

// IsDeliverableTool 报告工具是否会向工作区落盘产物（v4.24 C1 权威产物
// 登记表的登记口径）：写类 8 种（isWriterTool：write_file/edit_file/
// multi_edit/edit_lines/move_file/notebook_edit/delete_range/delete_symbol，
// 与前端 lib/changes.ts WRITE_TOOL_NAMES 一致，并已核对工具注册表全集）
// + 生成/导出类 3 种（isProducerTool）。
// bash/screen_capture 不登记：产物路径不出现在结构化参数里（bash 由命令
// 决定、screen_capture 落 .gaea/uploads/ 由系统生成路径），无法从参数权威
// 提取，不做启发式猜测。
func IsDeliverableTool(name string) bool {
	return isWriterTool(name) || isProducerTool(name)
}

// ExtractDeliverablePaths 从工具调用参数中提取产物路径（交付口径，纯函数）。
// 与 extractPaths（证据链「改动面」口径）的差异：不收 source——move_file 的
// 交付物是 destination，源路径不是交付物（与前端 lib/changes.ts
// extractDeliverablePaths 对齐）。键按工具类别分派：
//   - 写类（isWriterTool）：path / file_path / notebook_path / destination
//     单值 + paths / file_paths 数组 + edits[].path / edits[].file_path
//     （multi_edit/edit_file 编辑片段）；
//   - 生成/导出类（isProducerTool）：output 落盘参数——path 在这三类工具里
//     是输入源文件（如 format_convert 的 docx），不是交付物，不登记。
// 去重保持出现顺序，空白路径跳过。
func ExtractDeliverablePaths(name string, args json.RawMessage) []string {
	if len(args) == 0 {
		return nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(args, &fields); err != nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	push := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	if isProducerTool(name) {
		push(stringField(fields, "output"))
		return out
	}
	for _, key := range []string{"path", "file_path", "notebook_path", "destination"} {
		push(stringField(fields, key))
	}
	for _, key := range []string{"paths", "file_paths"} {
		for _, p := range stringSliceField(fields, key) {
			push(p)
		}
	}
	if raw, ok := fields["edits"]; ok {
		var edits []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &edits); err == nil {
			for _, e := range edits {
				for _, key := range []string{"path", "file_path"} {
					push(stringField(e, key))
				}
			}
		}
	}
	return out
}

func extractPaths(fields map[string]json.RawMessage) []string {
	var paths []string
	for _, key := range []string{"path", "file_path", "notebook_path"} {
		if s := stringField(fields, key); s != "" {
			paths = append(paths, s)
		}
	}
	for _, key := range []string{"paths", "file_paths"} {
		paths = append(paths, stringSliceField(fields, key)...)
	}
	return paths
}

func stringField(fields map[string]json.RawMessage, key string) string {
	raw, ok := fields[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func stringSliceField(fields map[string]json.RawMessage, key string) []string {
	raw, ok := fields[key]
	if !ok {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return values
}

func todoItemsField(fields map[string]json.RawMessage, key string) []TodoItem {
	raw, ok := fields[key]
	if !ok {
		return nil
	}
	var todos []TodoItem
	if err := json.Unmarshal(raw, &todos); err != nil {
		return nil
	}
	return normalizeTodos(todos)
}

func normalizeTodos(todos []TodoItem) []TodoItem {
	out := make([]TodoItem, 0, len(todos))
	for _, t := range todos {
		t.Content = strings.TrimSpace(t.Content)
		t.Status = strings.TrimSpace(t.Status)
		t.ActiveForm = strings.TrimSpace(t.ActiveForm)
		out = append(out, t)
	}
	return out
}

func todoStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "pending"
	}
	return status
}

func previousTodoCompleted(index int, current TodoItem, previous []TodoItem) bool {
	if index >= 1 && index <= len(previous) {
		p := previous[index-1]
		if todoStatus(p.Status) == "completed" && sameTodoIdentity(current, p) {
			return true
		}
	}
	for _, p := range previous {
		if todoStatus(p.Status) == "completed" && sameTodoIdentity(current, p) {
			return true
		}
	}
	return false
}

func sameTodoIdentity(a, b TodoItem) bool {
	return sameStepText(a.Content, b.Content) || sameStepText(a.ActiveForm, b.ActiveForm)
}

func hasSuccessfulCompleteStepForTodo(receipts []Receipt, index int, current []TodoItem) bool {
	for _, r := range receipts {
		if !r.Success || r.ToolName != "complete_step" || strings.TrimSpace(r.Step) == "" {
			continue
		}
		if r.TodoStep != nil && r.TodoStep.Found {
			if index >= 1 && index <= len(current) && sameTodoMatch(current[index-1], *r.TodoStep) {
				return true
			}
			continue
		}
		match := matchTodoStep(r.Step, current)
		if match.Found && match.Index == index {
			return true
		}
	}
	return false
}

func latestTodoStep(step string, receipts []Receipt) TodoStepMatch {
	for i := len(receipts) - 1; i >= 0; i-- {
		r := receipts[i]
		if !r.Success || r.ToolName != "todo_write" {
			continue
		}
		return matchTodoStep(step, r.Todos)
	}
	return TodoStepMatch{}
}

func sameTodoMatch(todo TodoItem, match TodoStepMatch) bool {
	return sameStepText(todo.Content, match.Content) || sameStepText(todo.ActiveForm, match.ActiveForm)
}

func matchTodoStep(step string, todos []TodoItem) TodoStepMatch {
	if n, ok := parseStepIndex(step); ok && n >= 1 && n <= len(todos) {
		t := todos[n-1]
		return TodoStepMatch{Found: true, Index: n, Content: t.Content, Status: t.Status, ActiveForm: t.ActiveForm}
	}
	for i, t := range todos {
		if sameStepText(step, t.Content) || sameStepText(step, t.ActiveForm) {
			return TodoStepMatch{Found: true, Index: i + 1, Content: t.Content, Status: t.Status, ActiveForm: t.ActiveForm}
		}
	}
	return TodoStepMatch{}
}

func parseStepIndex(step string) (int, bool) {
	step = strings.TrimSpace(strings.TrimSuffix(step, "."))
	n, err := strconv.Atoi(step)
	return n, err == nil
}

func sameStepText(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return a != "" && b != "" && strings.EqualFold(a, b)
}

func pathSet(paths []string) map[string]bool {
	out := map[string]bool{}
	for _, p := range paths {
		if p != "" {
			out[p] = true
		}
	}
	return out
}

func normalizePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = normalizePath(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, `\`, `/`)
	p = filepath.Clean(filepath.FromSlash(p))
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

// IncompleteTodos returns the items of a todo list that are not completed.
func IncompleteTodos(todos []TodoItem) []TodoStepMatch {
	incomplete := make([]TodoStepMatch, 0)
	for j, t := range todos {
		status := todoStatus(t.Status)
		if status == "completed" {
			continue
		}
		incomplete = append(incomplete, TodoStepMatch{
			Found:      true,
			Index:      j + 1,
			Content:    t.Content,
			Status:     status,
			ActiveForm: t.ActiveForm,
		})
	}
	return incomplete
}

// MatchStep resolves a complete_step.step (number, title, or drift-tolerant
// variant) against a todo list, returning the matched item.
func MatchStep(step string, todos []TodoItem) (TodoStepMatch, bool) {
	m := matchTodoStep(step, todos)
	return m, m.Found
}
