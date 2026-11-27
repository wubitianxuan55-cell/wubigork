package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── A4 中断交接块（Reasonix 2.x interrupted_recovery.go 蒸馏，v4.524）──────
//
// 病根：回合被流错误/取消打断后，本轮已完成的工作（写了哪些文件、跑了哪些
// 命令）只在会话历史里散落——下一轮模型凭印象重做已落盘的写入，或把被打断
// 的调用当成没做过。
//
// 机制：终局路径把本轮回执折叠成 **bounded 交接记录**（≤24 工具对、每工具
// ≤8 文件、值截 240 rune；不含原始参数/结果/推理/正文），下一条真实用户消息
// 前置 `<interrupted-turn-recovery>` 块一次性消费——「先查看工作区现状再动手；
// 不要盲目重复已完成的写入；被打断的调用确需则从头重发；用户新消息优先于
// 续跑旧任务」。
//
// 与上游的刻意差异（留档）：①gaea 终局路径本就把部分输出落库进上下文
// （v4.379 刀2 结构化块保全），上游则标记 unsafe 并排除——本块无
// unsafe_partial_output 条款；②记录存 AgentRunner 内存（桌面单运行时常驻），
// 不随会话持久化——进程重启丢失该记录属诚实边界（上游存 LocalOnly 消息）。
// ③completed/interrupted 判据复用 gaea 中断标记：结果含 "stream interrupted"
// 或无结果回执=被打断（persistInterruptedCalls 的信封文案）。

const interruptedRecoveryTag = "interrupted-turn-recovery"

const (
	maxRecoveryTools     = 24
	maxRecoveryFiles     = 8
	maxRecoveryValue     = 240
	maxRecoveryTodos     = 20
	maxRecoveryArtifacts = 16
	maxRecoveryHashBytes = 16 << 20
	// maxRecoveryTask 被打断那一轮的用户原话截到多长（rune）。
	maxRecoveryTask = 400
)

// turnRecovery 一次中断的交接记录。
type turnRecovery struct {
	Task             string // 被打断那一轮的用户原话（截断）：续跑时不必翻历史找任务
	CompletedTools   []recoveryTool
	InterruptedTools []string
	Todos            []recoveryTodo
	Resumed          bool // 进程重启后从会话日志重建的（文件指纹是恢复时的现状）
}

type recoveryTool struct {
	Name  string
	Files []recoveryArtifact
}

type recoveryArtifact struct {
	Path      string
	SHA256    string
	SizeBytes int64
}

type recoveryTodo struct {
	Content string
	Status  string
}

// isSyntheticUserTurn 合成用户消息（steer/流恢复/readiness 重试/各门与提醒
// nudge/unapplied steer）不是真实用户轮——回扫真实轮起点时跳过（上游
// SyntheticUserPrefixes 同口径）。所有门 nudge 统一以 "[system] " 开头
// （stop_gate/repeat_detect/output_continue/recall_reminder/task schema 重试/
// executeOne 重复写提醒），单前缀收口；真实用户消息以该前缀开头属可接受的
// 误判边界（上游同口径）。
func isSyntheticUserTurn(content string) bool {
	for _, prefix := range []string{
		MidTurnSteerPrefix,
		ParentSteerPrefix,
		"The previous assistant response was interrupted during streaming.",
		"Host final-answer readiness check failed.",
		"[system] ",
		unappliedSteerPrefix,
	} {
		if strings.HasPrefix(content, prefix) {
			return true
		}
	}
	return false
}

// ── B3：未消费 steer 的退出收据（Reasonix 2.x sessioninbox 蒸馏最小刀）──
//
// 病根（不需要崩溃就丢）：controller.Steer 检查 running 后、入队前，回合
// 可能恰好收尾清队列——用户看到插话已发射，消息却无声消失。上游 #7045 教训
// 同源：steer 被接受但回合没消费，绝不能静默。
//
// 机制：回合退出先关 intake 闸（新入队被拒、controller 降级普通 Send），
// 再把队列里已接受的 steer 落成会话记录——**不自动重放成新任务**（#7045），
// 文案明确「上一轮的补充指引，若仍相关才用；用户新消息优先」。上游存
// LocalOnly 消息；gaea 会话本就全文落库，这里即等价的持久收据，A4 回扫经
// unappliedSteerPrefix 识别为合成轮。
const unappliedSteerPrefix = "[unapplied steer] "

// drainSteerOnExit 回合退出路径：关门 + 收据。round 内已消费的 steer 不经过
// 这里（consumeSteer 取走）。
func (a *AgentRunner) drainSteerOnExit() {
	a.steerMu.Lock()
	a.steerIntakeClosed = true
	remaining := a.steerQueue
	a.steerQueue = nil
	a.steerConsumed = false
	a.steerMu.Unlock()
	for _, it := range remaining {
		originNote := ""
		if it.origin == steerFromParent {
			originNote = " (from the parent agent, not the user)"
		} else if it.origin == steerFromSubagent {
			originNote = " (from a subagent mid-task report, not the user)"
		}
		a.session.Add(provider.Message{
			Role: provider.RoleUser,
			Content: unappliedSteerPrefix + "The previous turn ended before this mid-task guidance reached the model. " +
				"It is recorded as guidance for the recorded task — apply it only if still relevant, and the user's next message takes precedence." + originNote + "\n" + it.text,
		})
	}
	if len(remaining) > 0 {
		a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
			Text: fmt.Sprintf("%d 条插话未被本轮消费（回合已结束），已存入会话记录，下一条消息的上下文里可见", len(remaining))})
	}
}

// captureInterruptedRecovery 终局路径调用：把本轮回执折叠成交接记录，存
// pending（无工具对=无记录——纯文本轮中断没有「重做写入」风险）。
func (a *AgentRunner) captureInterruptedRecovery() {
	msgs := a.session.Messages
	start := lastRealUserTurnStart(msgs)
	if start <= 0 || start >= len(msgs) {
		return
	}
	rec := foldTurnRecovery(msgs[start:], a.todoState)
	if rec == nil {
		return
	}
	rec.Task = recoveryTask(msgs[start-1])
	a.pendingRecovery = rec
}

// lastRealUserTurnStart 最后一个真实用户轮的起点下标（用户消息的下一条）；找不到
// 返回 0。合成用户消息（steer/流恢复/readiness）与压缩摘要都属回合内部。
func lastRealUserTurnStart(msgs []provider.Message) int {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role == provider.RoleUser && !isSyntheticUserTurn(m.Content) && !isCompactionSummary(m) {
			return i + 1
		}
	}
	return 0
}

// recoveryTask 用户原话压成一行并截断。
func recoveryTask(m provider.Message) string {
	return clipRunes(strings.Join(strings.Fields(m.Content), " "), maxRecoveryTask)
}

// rebuildInterruptedRecovery 会话换入时调用（进程重启后恢复、或检出历史）：交接记录
// 原本只存在 AgentRunner 内存里，重启即丢——上一轮没收尾就从持久化消息里重建一份，
// 下一条真实用户消息前照样前置交接块（2026-11-27 AI 反馈 #15②：断点恢复时翻文件
// 现场重推状态，恢复成本高）。待办清单不在这里取：它自己会从日志重建，消费时再读。
func (a *AgentRunner) rebuildInterruptedRecovery() {
	a.pendingRecovery = nil
	if a.session == nil {
		return
	}
	msgs := a.session.Messages
	start := lastRealUserTurnStart(msgs)
	if start <= 0 || start >= len(msgs) || !turnUnfinished(msgs[start:]) {
		return
	}
	rec := foldTurnRecovery(msgs[start:], nil)
	if rec == nil {
		return
	}
	rec.Task, rec.Resumed = recoveryTask(msgs[start-1]), true
	a.pendingRecovery = rec
}

// turnUnfinished 报告一轮是否没收尾：末尾在等模型接话（最后一条是工具结果，或带
// 工具调用的助手消息）、有调用没等到结果、结果里带中断标记、或末尾是流中断的续写
// 提示。轮尾的合成用户消息（未消费的插话收据等）不算结尾。
func turnUnfinished(region []provider.Message) bool {
	end := len(region)
	for end > 0 && region[end-1].Role == provider.RoleUser && isSyntheticUserTurn(region[end-1].Content) {
		if strings.HasPrefix(region[end-1].Content, "The previous assistant response was interrupted during streaming.") {
			return true
		}
		end--
	}
	if end == 0 {
		return false
	}
	if last := region[end-1]; last.Role == provider.RoleTool || (last.Role == provider.RoleAssistant && len(last.ToolCalls) > 0) {
		return true
	}
	for _, m := range region[:end] {
		if m.Role == provider.RoleTool && strings.Contains(m.Content, "stream interrupted") {
			return true
		}
	}
	return false
}

// foldTurnRecovery 把一轮的消息与待办折叠成交接记录（终局捕获与重启后重建共用）。
// 没有任何工具对返回 nil。
func foldTurnRecovery(region []provider.Message, todos []evidence.TodoItem) *turnRecovery {
	results := map[string]provider.Message{}
	for _, m := range region {
		if m.Role == provider.RoleTool && m.ToolCallID != "" {
			results[m.ToolCallID] = m
		}
	}
	rec := &turnRecovery{}
	for _, t := range todos {
		if len(rec.Todos) >= maxRecoveryTodos {
			break
		}
		rec.Todos = append(rec.Todos, recoveryTodo{
			Content: clipRecoveryValue(t.Content),
			Status:  canonicalTodoStatus(t.Status),
		})
	}
	seenInterrupted := map[string]bool{}
	hashAttempts := 0
	for _, m := range region {
		if m.Role != provider.RoleAssistant || len(m.ToolCalls) == 0 {
			continue
		}
		for _, tc := range m.ToolCalls {
			res, ok := results[tc.ID]
			interrupted := !ok || strings.Contains(res.Content, "stream interrupted")
			if interrupted {
				if !seenInterrupted[tc.Name] {
					seenInterrupted[tc.Name] = true
					rec.InterruptedTools = append(rec.InterruptedTools, tc.Name)
				}
				continue
			}
			rt := recoveryTool{Name: tc.Name}
			if p := extractRecoveryPath(tc.Name, tc.Arguments); p != "" {
				withHash := hashAttempts < maxRecoveryArtifacts
				if withHash {
					hashAttempts++
				}
				rt.Files = append(rt.Files, recoveryArtifactFor(p, withHash))
			}
			if len(rec.CompletedTools) < maxRecoveryTools {
				rec.CompletedTools = append(rec.CompletedTools, rt)
			}
		}
	}
	if len(rec.CompletedTools) == 0 && len(rec.InterruptedTools) == 0 {
		return nil
	}
	return rec
}

// extractRecoveryPath extends the edit-tool path extractor to generation tools
// whose artifact key is "output". Recovery treats the output path as an
// artifact, not as a file-edit stale anchor, so this stays local to the
// checkpoint code.
func extractRecoveryPath(name, args string) string {
	if p := strings.TrimSpace(extractFilePath(name, args)); p != "" {
		return p
	}
	var payload struct {
		Output     string `json:"output"`
		OutputPath string `json:"output_path"`
	}
	if json.Unmarshal([]byte(args), &payload) != nil {
		return ""
	}
	if strings.TrimSpace(payload.OutputPath) != "" {
		return payload.OutputPath
	}
	return payload.Output
}

// pendingRecoveryBlock 渲染交接块（bounded：≤24 工具、每工具 ≤8 文件、
// 值截 240 rune；不含原始参数/结果/推理）。
func pendingRecoveryBlock(r *turnRecovery) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<%s>\n", interruptedRecoveryTag)
	b.WriteString("The previous turn was interrupted. Treat these as host-verified recovery facts, not as a new task.\n")
	if r.Task != "" {
		fmt.Fprintf(&b, "task: %s\n", html.EscapeString(r.Task))
	}
	if r.Resumed {
		b.WriteString("recovered_from: session_log (rebuilt after a process restart; file fingerprints reflect the workspace at resume time)\n")
	}
	b.WriteString("task_checkpoint:\n")
	if len(r.Todos) == 0 {
		b.WriteString("  not_recorded (the interrupted turn never used todo_write; treat completed_tools below as the progress record)\n")
	} else {
		groups := map[string][]string{"completed": nil, "in_progress": nil, "pending": nil}
		for _, todo := range r.Todos {
			status := todo.Status
			if _, ok := groups[status]; !ok {
				status = "pending"
			}
			groups[status] = append(groups[status], html.EscapeString(clipRecoveryValue(todo.Content)))
		}
		for _, status := range []string{"completed", "in_progress", "pending"} {
			items := groups[status]
			if len(items) == 0 {
				fmt.Fprintf(&b, "  %s: none\n", status)
				continue
			}
			fmt.Fprintf(&b, "  %s: %s\n", status, strings.Join(items, " | "))
		}
	}
	if len(r.CompletedTools) == 0 {
		b.WriteString("completed_tools: none\n")
	} else {
		b.WriteString("completed_tools:\n")
		shown := r.CompletedTools
		if len(shown) > maxRecoveryTools {
			shown = shown[:maxRecoveryTools]
		}
		for _, t := range shown {
			fmt.Fprintf(&b, "- %s", html.EscapeString(strings.TrimSpace(t.Name)))
			files := t.Files
			if len(files) > maxRecoveryFiles {
				files = files[:maxRecoveryFiles]
			}
			if len(files) > 0 {
				clipped := make([]string, 0, len(files))
				for _, f := range files {
					label := html.EscapeString(clipRecoveryValue(f.Path))
					if f.SHA256 != "" {
						label += fmt.Sprintf(" [sha256=%s size=%d]", f.SHA256, f.SizeBytes)
					}
					clipped = append(clipped, label)
				}
				fmt.Fprintf(&b, " files=%s", strings.Join(clipped, ","))
			}
			b.WriteByte('\n')
		}
		if len(r.CompletedTools) > maxRecoveryTools {
			fmt.Fprintf(&b, "- ... %d additional completed tool pair(s) omitted\n", len(r.CompletedTools)-maxRecoveryTools)
		}
	}
	if len(r.InterruptedTools) == 0 {
		b.WriteString("interrupted_tools: none\n")
	} else {
		shown := r.InterruptedTools
		if len(shown) > maxRecoveryTools {
			shown = shown[:maxRecoveryTools]
		}
		fmt.Fprintf(&b, "interrupted_tools: %s", strings.Join(shown, ", "))
		if len(r.InterruptedTools) > maxRecoveryTools {
			fmt.Fprintf(&b, ", ... %d additional call(s) omitted", len(r.InterruptedTools)-maxRecoveryTools)
		}
		b.WriteByte('\n')
	}
	b.WriteString("Before continuing, inspect the current workspace and prior completed tool results. " +
		"Do not blindly repeat completed writes. Re-issue any interrupted tool call from scratch with " +
		"complete arguments if it is still needed. The user's new message takes precedence: if it is " +
		"unrelated to the interrupted task, do not resume that task.\n")
	fmt.Fprintf(&b, "</%s>", interruptedRecoveryTag)
	return b.String()
}

// withInterruptedRecovery 消费 pending 记录：有则前置块到本轮输入并清零
// （一次性——下一回合隐式消费，不原地改历史，上游同口径）。记录里没有待办时
// 取一次现行清单——重启后重建的记录拿不到内存里的待办，清单自己会从日志重建。
func (a *AgentRunner) withInterruptedRecovery(input string) string {
	if a.pendingRecovery == nil {
		return input
	}
	if len(a.pendingRecovery.Todos) == 0 && a.pendingRecovery.Resumed {
		// 重启后重建的记录拿不到内存里的待办；而 Run 里「前置交接块」排在「从日志
		// 重建待办」之前——这里先把清单从日志重建出来，用过待办工具的轮才不会被
		// 写成未记录。终局捕获的记录在捕获时已带上当时的清单，不走这里。
		a.rebuildTodoState(a.session.Messages)
		a.pendingRecovery.Todos = foldRecoveryTodos(a.CanonicalTodoState())
	}
	block := pendingRecoveryBlock(a.pendingRecovery)
	a.pendingRecovery = nil
	return block + "\n\n" + input
}

// foldRecoveryTodos 待办清单折叠进交接记录（截 20 条）。
func foldRecoveryTodos(todos []evidence.TodoItem) []recoveryTodo {
	var out []recoveryTodo
	for _, t := range todos {
		if len(out) >= maxRecoveryTodos {
			break
		}
		out = append(out, recoveryTodo{Content: clipRecoveryValue(t.Content), Status: canonicalTodoStatus(t.Status)})
	}
	return out
}

// clipRunes 按 rune 截断（不补省略号）。
func clipRunes(s string, n int) string {
	if rs := []rune(s); len(rs) > n {
		return string(rs[:n])
	}
	return s
}

// clipRecoveryValue 值截断（240 rune + 省略号）。
func clipRecoveryValue(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRecoveryValue {
		return value
	}
	return string(runes[:maxRecoveryValue]) + "…"
}

// recoveryArtifactFor records the artifact path with a bounded content
// fingerprint when the path names a regular file. The hash lets the resumed
// agent verify "this exact artifact already exists" instead of inferring it from
// a tool-name list alone. Large/unreadable files keep the path without a hash so
// recovery capture stays bounded and fail-open.
func recoveryArtifactFor(path string, withHash bool) recoveryArtifact {
	art := recoveryArtifact{Path: clipRecoveryValue(path)}
	if !withHash {
		return art
	}
	abs := path
	if !filepath.IsAbs(abs) {
		if cwd, err := os.Getwd(); err == nil {
			abs = filepath.Join(cwd, abs)
		}
	}
	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxRecoveryHashBytes {
		return art
	}
	f, err := os.Open(abs)
	if err != nil {
		return art
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return art
	}
	art.SHA256 = hex.EncodeToString(h.Sum(nil))
	art.SizeBytes = info.Size()
	return art
}
