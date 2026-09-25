package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spill"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func (a *AgentRunner) executeOne(ctx context.Context, call provider.ToolCall) toolOutcome {
	t, ok := a.tools.Get(call.Name)
	if !ok {
		return toolOutcome{
			output: tool.WrapError(tool.CodeUnknownTool, fmt.Sprintf("unknown tool %q", call.Name), nil),
			errMsg: fmt.Sprintf("unknown tool %q", call.Name),
		}
	}

	// V10.13 → v4.376: 成功循环检测降级 advisory — 硬阻断已退役（上游
	// reasonix 裁决：实测硬阻断误伤多于收益），计数照记、提醒走批后的
	// advisoryRepeatSuccess，真实工具结果不再被 blocked 替换。

	// Centralised pre-execution checks via the ToolDispatcher (production path).
	// When dispatcher is nil (test/benchmark paths), gate/hooks are
	// checked inline — preserving backward compatibility with existing tests.
	// checked inline — preserving backward compatibility with existing tests.
	if a.dispatcher != nil {
		cr := a.dispatcher.Check(ctx, call.Name, json.RawMessage(call.Arguments), t.ReadOnly())
		if !cr.Allowed {
			return toolOutcome{
				output:  cr.Reason,
				blocked: cr.Blocked,
				errMsg:  cr.Reason,
			}
		}
	} else {
		if a.hooks != nil {
			allow, modifiedArgs, reason := a.hooks.PermissionRequest(ctx, call.Name, json.RawMessage(call.Arguments))
			if !allow {
				return toolOutcome{
					output:  "blocked by PermissionRequest hook: " + reason,
					blocked: true,
					errMsg:  "blocked by PermissionRequest hook",
				}
			}
			if len(modifiedArgs) > 0 {
				call.Arguments = string(modifiedArgs)
			}
		}
		// Audit P1: load the gate once, atomically — the controller may swap
		// it mid-turn (SetPermLevel) while this goroutine runs, so a plain
		// field read could tear. One Load yields one whole gate (old or new).
		if gw := a.gate.Load(); gw != nil {
			allow, reason, err := gw.g.Check(ctx, call.Name, json.RawMessage(call.Arguments), t.ReadOnly())
			if err != nil {
				return toolOutcome{
					output:  fmt.Sprintf("blocked: %s (%v)", reason, err),
					blocked: true,
					errMsg:  fmt.Sprintf("blocked: %v", err),
				}
			}
			if !allow {
				return toolOutcome{
					output:  "blocked: " + reason,
					blocked: true,
					errMsg:  "blocked by permission policy",
				}
			}
		}
		if a.hooks != nil {
			if block, msg := a.hooks.PreToolUse(ctx, call.Name, json.RawMessage(call.Arguments)); block {
				if msg == "" {
					msg = "blocked by a PreToolUse hook"
				}
				return toolOutcome{
					output:  "blocked: " + msg,
					blocked: true,
					errMsg:  "blocked by PreToolUse hook",
				}
			}
		}
	}
	// Phase 1 DSpark: 确定性预检查 — 在文件编辑工具实际执行前，
	// 验证 old_string / anchor 是否存在于目标文件中。
	// 预检查命中时返回诊断消息，阻止必然失败的操作，节省一整轮 API 调用。
	// 缓存安全: 纯运行时判断，返回内容作为本轮新 tool_result 追加在末尾。
	if msg := a.precheckTool(call.Name, json.RawMessage(call.Arguments)); msg != "" {
		return toolOutcome{
			output:  msg,
			blocked: true,
			errMsg:  msg,
		}
	}
	// V10.28: stale anchor 守卫 — 同一轮内已编辑的文件必须先 read_file 才能再编辑
	if !t.ReadOnly() && isFileWriter(call.Name) {
		if path := extractFilePath(call.Name, call.Arguments); path != "" {
			// 刀2（Reasonix fileops observation 蒸馏）：跨轮外部修改检测——
			// 模型观察过的版本与磁盘当前版本不一致即拦，防静默覆盖外部修改。
			if msg := a.checkFileStale(path); msg != "" {
				return toolOutcome{output: msg, blocked: true, errMsg: msg}
			}
			// turnMu guards the per-turn stale maps against the other executeOne
			// goroutines running this batch (audit P0 race fix). Only the map
			// reads happen under the lock; the message is built outside it.
			a.turnMu.Lock()
			written := a.staleWrittenFiles != nil && a.staleWrittenFiles[path]
			read := a.staleReadFiles != nil && a.staleReadFiles[path]
			a.turnMu.Unlock()
			if written && !read {
				msg := fmt.Sprintf("blocked: [stale content] %q was already modified this turn. Re-read it with read_file first so your edit anchors (old_string/anchors) match the current file content.", path)
				return toolOutcome{output: msg, blocked: true, errMsg: msg}
			}
		}
	}
	// V4.2: tool result cache — avoid redundant disk IO for repeat file reads
	if call.Name == "read_file" && a.tc != nil {
		var ra struct {
			Path   string `json:"path"`
			Offset int    `json:"offset"`
		}
		if err := json.Unmarshal(json.RawMessage(call.Arguments), &ra); err == nil && ra.Path != "" {
			if cached, ok := a.tc.Get(ra.Path, ra.Offset); ok {
				return toolOutcome{output: cached}
			}
		}
	}

	cctx := withCallContext(ctx, call.ID, a.sink, a.asker)
	// request_permission：把交互式 PermissionRequester 盖章到工具 ctx（与
	// asker 同纪律——SetPermissionRequester 在回合启动前注入，nil 不盖章，
	// headless 运行保持「无交互用户」语义）。
	cctx = WithPermissionRequester(cctx, a.permReq)
	// plan gate：计划模式闸盖章（exit_plan_mode 经 PlanGateFromContext 读取；
	// 与 asker 同位同纪律，runner 恒实现故不判空）。
	cctx = WithPlanGate(cctx, a)
	if a.evidence != nil {
		// strictVerify 默认为 false（由 NewLedger 设定）。Plan Mode 下由外部
		// 显式设为 true 后保留，不在每轮工具调用时重置。
		cctx = evidence.WithLedger(cctx, a.evidence)
	}
	// v4.1 证据链：把变更台账盖章进工具 ctx（与 WithLedger 同位注入）——
	// 写盘工具（edit_file/write_file/move_file）成功后经 evidence.RecordChange 上报。
	cctx = evidence.WithChanges(cctx, a.changes)
	if a.jobs != nil {
		cctx = jobs.WithManager(cctx, a.jobs)
	}
	if a.memQueue != nil {
		cctx = memory.WithQueue(cctx, a.memQueue)
	}
	// S1.2 A：把会话空间盖章到工具调用 ctx（与 WithQueue 同注入点）。ctx 已由
	// agent.Run 注入会话空间（SpaceFromContext 缺省 work）——remember/memory_get
	// 等记忆工具经 memory.SpaceFromContext 读取，写读都限定本空间。
	cctx = memory.WithSpace(cctx, SpaceFromContext(ctx))
	if a.sessionSaver != nil {
		cctx = memory.WithSessionSaver(cctx, a.sessionSaver)
	}
	// v4.379 spill 泄洪库盖章（read_spill 工具经 ctx 取库，与 jobs/queue 同注入点）。
	cctx = spill.WithStore(cctx, a.spill)
	if a.promoter != nil {
		cctx = memory.WithPromoter(cctx, a.promoter)
	}
	var result string
	var err error
	start := time.Now()
	if ct, ok := t.(tool.ContextualTool); ok {
		tc := tool.ToolContext{
			SessionID:  a.sessionID,
			AgentName:  "agent",
			ToolCallID: call.ID,
			Messages:   a.session.Messages,
		}
		result, err = ct.ExecuteWithContext(cctx, tc, json.RawMessage(call.Arguments))
	} else {
		result, err = t.Execute(cctx, json.RawMessage(call.Arguments))
	}
	duration := time.Since(start).Milliseconds()

	// V4.2: cache successful file reads; invalidate writes
	if a.tc != nil {
		switch call.Name {
		case "read_file":
			if err == nil {
				var ra struct {
					Path   string `json:"path"`
					Offset int    `json:"offset"`
				}
				if json.Unmarshal(json.RawMessage(call.Arguments), &ra) == nil && ra.Path != "" {
					a.tc.Set(ra.Path, ra.Offset, result)
				}
			}
		case "edit_file", "write_file", "multi_edit", "edit_lines", "delete_range", "delete_symbol":
			var wa struct {
				Path string `json:"path"`
			}
			if json.Unmarshal(json.RawMessage(call.Arguments), &wa) == nil && wa.Path != "" {
				a.tc.InvalidatePath(wa.Path)
			}
		case "move_file":
			// A move invalidates both endpoints: args carry no "path" key, so
			// the generic branch above would silently miss them (S0.6 risk 2).
			var ma struct {
				Source      string `json:"source"`
				Destination string `json:"destination"`
			}
			if json.Unmarshal(json.RawMessage(call.Arguments), &ma) == nil {
				if ma.Source != "" {
					a.tc.InvalidatePath(ma.Source)
				}
				if ma.Destination != "" {
					a.tc.InvalidatePath(ma.Destination)
				}
			}
		}
	}

	// 刀2（Reasonix fileops observation 蒸馏）：版本观察记账——真实磁盘读
	// 成功后记录指纹（缓存命中早退不经过这里，不假装观察过）；写成功后刷新
	// 指纹（后续编辑站在自己刚写的内容上，自身写入不算「外部修改」）。
	if err == nil {
		switch {
		case call.Name == "read_file":
			if p := extractFilePath(call.Name, call.Arguments); p != "" {
				a.observeFile(p)
			}
		case isFileWriter(call.Name):
			if p := extractFilePath(call.Name, call.Arguments); p != "" {
				a.observeFile(p)
			}
		}
	}

	// V3.2: audit trail — log every tool execution
	if a.auditFunc != nil {
		outcome := "success"
		errMsg := ""
		if err != nil {
			outcome = "error"
			errMsg = err.Error()
		}
		a.auditFunc(call.Name, "", t.ReadOnly(), outcome, errMsg, len(result), duration)
	}

	// V3.0: notify workspace observer of successful edits.
	if err == nil && !t.ReadOnly() && a.dispatcher != nil {
		if path := extractFilePath(call.Name, call.Arguments); path != "" {
			a.dispatcher.NotifyEdit(path)
		}
	}

	if a.evidence != nil {
		if call.Name == "complete_step" {
			if err == nil {
				a.evidence.Record(evidence.ReceiptFromToolCall(call.Name, json.RawMessage(call.Arguments), true, t.ReadOnly()))
			}
		} else {
			a.evidence.Record(evidence.ReceiptFromToolCall(call.Name, json.RawMessage(call.Arguments), err == nil, t.ReadOnly()))
		}
	}
	// PostToolUse hooks observe the result (they can't block); fired whether the
	// call succeeded or errored, since the tool did run.
	if a.hooks != nil {
		a.hooks.PostToolUse(ctx, call.Name, json.RawMessage(call.Arguments), result)
	}
	if err != nil {
		// Errors from tool execution are agent-recoverable (bad args, wrong file,
		// command failed) — the model can fix them on the next turn. Errors from
		// unknown-tool / blocked / panic are NOT recoverable.
		recoverable := true
		detail := strings.TrimSpace(result)
		// V10.13: 参数非法 JSON 时附带工具 schema，帮助模型一次修正。
		// 移植自 Reasonix malformed-args schema echo。
		if !json.Valid([]byte(call.Arguments)) {
			detail = strings.TrimRight(detail, "\n") + "\nThe arguments were not valid JSON. Re-emit them exactly per this schema:\n" + string(t.Schema())
		}
		env := tool.WrapError(tool.CodeExecError, firstLine(err.Error()), map[string]any{"tool": call.Name, "detail": detail})
		body, truncMsg := truncateToolOutput(env)
		return toolOutcome{output: body, errMsg: firstLine(err.Error()), recoverable: recoverable, truncated: truncMsg != "", truncMsg: truncMsg}
	}
	// V10.13: 记录成功签名用于循环检测
	a.recordRepeatSuccess(call, t)
	// V10.27: 追踪后台任务启停模式，用于检测 start-kill 循环
	// turnMu guards the bg flags/streak — sibling executeOne goroutines in the
	// same batch and the run loop's checkBgStartKillCycle touch them too
	// (audit P0 race fix).
	a.turnMu.Lock()
	switch call.Name {
	case "bash":
		var bp struct {
			RunInBackground bool `json:"run_in_background"`
		}
		if json.Unmarshal([]byte(call.Arguments), &bp) == nil && bp.RunInBackground {
			a.bgJobStartedThisTurn = true
		} else {
			// 前台 bash — 证明模型愿意等结果，重置循环计数
			a.bgStartKillStreak = 0
		}
	case "bash_output", "wait":
		a.bgOutputReadThisTurn = true
		a.bgStartKillStreak = 0 // 读取了输出 — 正常使用模式
	case "kill_shell":
		a.bgJobKilledThisTurn = true
	}
	a.turnMu.Unlock()
	// V10.28: 追踪 stale anchor — 记录本轮成功的读写操作
	// Locked: the lazy map init + writes race with the other executeOne
	// goroutines of this batch (audit P0 — "concurrent map writes" crash).
	a.turnMu.Lock()
	if path := extractFilePath(call.Name, call.Arguments); path != "" {
		if t.ReadOnly() && call.Name == "read_file" {
			if a.staleReadFiles == nil {
				a.staleReadFiles = make(map[string]bool)
			}
			a.staleReadFiles[path] = true
			if a.staleWrittenFiles != nil {
				delete(a.staleWrittenFiles, path) // 刷新后重置
			}
		} else if !t.ReadOnly() && isFileWriter(call.Name) {
			if a.staleWrittenFiles == nil {
				a.staleWrittenFiles = make(map[string]bool)
			}
			a.staleWrittenFiles[path] = true
		}
	}
	a.turnMu.Unlock()
	// A foreground `task` sub-agent just finished — its result is the final answer.
	if a.hooks != nil && call.Name == "task" && !isBackgroundTaskCall(call.Arguments) {
		a.hooks.SubagentStop(ctx, result)
	}
	// 刀1（v4.379，dsh spill-policy 蒸馏）：泄洪捕获必须在 SmartCompress 之前
	// ——按工具压缩与全局截断都会销毁中段，泄洪保的是原文；预览仍交给既有
	// 压缩管线。best-effort：库关着/工具豁免/不足阈值/超限拒收一律返回 ""
	// 零影响，泄洪失败绝不把成功调用变错误或藏掉内联结果。
	rawBytes := len(result)
	spillID := ""
	if a.spill != nil && spillCandidate(call.Name) && rawBytes >= spillMinBytes {
		spillID = a.spill.Save(call.Name, result)
	}
	result = SmartCompress(call.Name, result)
	if spillID != "" {
		// locator 前置：单行 JSON 信封被字节帽头部截断、多行结果 head+tail
		// 两种形态下头部都必保（见 spillNotice 注释的生存性结论）。
		result = spillNotice(spillID, rawBytes) + "\n" + result
	}
	env := tool.WrapResult(tool.CodeOK, map[string]any{"tool": call.Name, "result": result})
	body, truncMsg := truncateToolOutput(env)
	return toolOutcome{output: body, truncated: truncMsg != "", truncMsg: truncMsg}
}

// isBackgroundTaskCall reports whether a `task` call set run_in_background.
func isBackgroundTaskCall(args string) bool {
	var p struct {
		RunInBackground bool `json:"run_in_background"`
	}
	_ = json.Unmarshal([]byte(args), &p)
	return p.RunInBackground
}

// toolReadOnly reports a tool's ReadOnly classification by name.
func (a *AgentRunner) toolReadOnly(name string) bool {
	t, ok := a.tools.Get(name)
	return ok && t.ReadOnly()
}

// ── V10.13: 成功循环检测 — 移植自 Reasonix ──────────────────────────

// repeatSuccessAllowed 是同一写工具签名触发 advisory 的成功次数阈值。
// 2 次给模型自我修正的空间；第 3 次起提醒（不再阻止执行——v4.376 降级
// 裁决，Distilled from Reasonix repeat-tool-reminder 纯 advisory 立场）。
const repeatSuccessAllowed = 2

// repeatSuccessAdvisory 是同签名写工具第 3 次成功后的批后提醒。合成 user
// 消息（turn 尾，不动缓存稳定前缀），每签名每轮至多一条。
const repeatSuccessAdvisory = "[system] The same write tool call has now succeeded %d times with identical arguments in this turn (tool %q). The writes are real — re-running the identical call again is unlikely to help. Verify the result with a read or test command, change the approach, or explain the blocker in your final answer."

// advisoryRepeatSuccess scans the repeat-success counter for signatures that
// just crossed the threshold and injects one advisory per signature. Returns
// true when a nudge was injected (caller continues the loop). Counting
// happened in executeOne; the injection happens here on the run-loop
// goroutine because session.Add is not goroutine-safe with the batch.
func (a *AgentRunner) advisoryRepeatSuccess() bool {
	type crossed struct {
		name  string
		count int
	}
	var newly []crossed
	a.turnMu.Lock()
	for sig, count := range a.repeatSuccessCounts {
		if count <= repeatSuccessAllowed || a.repeatSuccessNudged[sig] {
			continue
		}
		if a.repeatSuccessNudged == nil {
			a.repeatSuccessNudged = make(map[string]bool)
		}
		a.repeatSuccessNudged[sig] = true
		name := sig
		if i := strings.IndexByte(sig, 0); i >= 0 {
			name = sig[:i]
		}
		newly = append(newly, crossed{name: name, count: count})
	}
	a.turnMu.Unlock()
	if len(newly) == 0 {
		return false
	}
	// 确定性序：多签名同批越线时注入顺序稳定。
	sort.Slice(newly, func(i, j int) bool { return newly[i].name < newly[j].name })
	for _, c := range newly {
		a.session.Add(provider.Message{
			Role:    provider.RoleUser,
			Content: fmt.Sprintf(repeatSuccessAdvisory, c.count, c.name),
		})
		a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo,
			Text: fmt.Sprintf("repeat-success advisory: %q succeeded %d times with identical arguments", c.name, c.count)})
	}
	return true
}

// recordRepeatSuccess 记录一次成功的写工具调用，用于循环检测。
// turnMu covers the lazy map init and the increment — executeOne runs in
// parallel goroutines and unsynchronized map access here is a fatal
// "concurrent map writes" crash (audit P0 race fix).
func (a *AgentRunner) recordRepeatSuccess(call provider.ToolCall, t tool.Tool) {
	sig, ok := repeatSuccessSignature(call, t)
	if !ok {
		return
	}
	a.turnMu.Lock()
	if a.repeatSuccessCounts == nil {
		a.repeatSuccessCounts = make(map[string]int)
	}
	a.repeatSuccessCounts[sig]++
	a.turnMu.Unlock()
}

// repeatSuccessSignature 为写工具调用计算可比较的签名。
// 只读工具不参与（不会修改文件状态）；仅对写文件工具和写入型 bash 签名。
func repeatSuccessSignature(call provider.ToolCall, t tool.Tool) (string, bool) {
	if t.ReadOnly() {
		return "", false
	}
	switch call.Name {
	case "write_file", "edit_file", "multi_edit", "edit_lines", "move_file", "delete_range", "delete_symbol":
		return call.Name + "\x00" + canonicalToolArgs(call.Arguments), true
	case "bash":
		var p struct {
			Command         string `json:"command"`
			RunInBackground bool   `json:"run_in_background"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &p); err != nil {
			return "", false
		}
		if p.RunInBackground || !isShellFileWriteCommand(p.Command) {
			return "", false
		}
		return "bash\x00" + normalizeShellCommand(p.Command), true
	default:
		return "", false
	}
}

// canonicalToolArgs 将 JSON 参数规范化为紧凑可比较形式。
func canonicalToolArgs(raw string) string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return strings.TrimSpace(raw)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, b); err != nil {
		return string(b)
	}
	return compact.String()
}

// normalizeShellCommand 规范化 shell 命令（合并空白）。
func normalizeShellCommand(command string) string {
	return strings.Join(strings.Fields(command), " ")
}

// isShellFileWriteCommand 判断 shell 命令是否会写入文件。
func isShellFileWriteCommand(command string) bool {
	lower := strings.ToLower(command)
	switch {
	case shellPythonOpenWrites(lower):
		return true
	case strings.Contains(lower, "set-content") || strings.Contains(lower, "add-content") || strings.Contains(lower, "out-file"):
		return true
	case strings.Contains(lower, "sed -i") || strings.Contains(lower, "perl -pi"):
		return true
	case hasShellWriteRedirect(command):
		return true
	default:
		return false
	}
}

// shellPythonOpenWrites 检测 Python open() 调用是否以写模式打开文件。
func shellPythonOpenWrites(lower string) bool {
	if !strings.Contains(lower, "open(") {
		return false
	}
	if strings.Contains(lower, ".write(") {
		return true
	}
	for _, marker := range []string{", 'w", `, "w`, ", 'a", `, "a`, ", 'x", `, "x`, "mode='w", `mode="w`, "mode='a", `mode="a`, "mode='x", `mode="x`} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// hasShellWriteRedirect 检测 shell 命令是否包含写重定向（> 非 2>）。
func hasShellWriteRedirect(command string) bool {
	var quote rune
	var prev rune
	for _, r := range command {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			prev = r
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			prev = r
			continue
		}
		if r == '>' {
			if prev == '2' {
				prev = r
				continue
			}
			return true
		}
		prev = r
	}
	return false
}

// isFileWriter reports whether the tool name targets a specific file for writing.
func isFileWriter(name string) bool {
	switch name {
	case "edit_file", "write_file", "multi_edit", "edit_lines", "move_file", "delete_range", "delete_symbol":
		return true
	}
	return false
}
