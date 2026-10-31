package pantheon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"
)

const (
	maxProcessOutput = 4 << 20
	maxPromptRunes   = 24000
	maxAgentOutput   = 200000
	// 8 分钟：plan/工具型席位（如 zcode 读大型仓库）陈词常超 4 分钟
	// （2026-10-29 实弹 zcode context deadline exceeded 逮住）
	defaultAgentLimit = 8 * time.Minute
	promptClipMarker  = "\n\n【中间内容因提示词长度限制省略】\n\n"
)

type commandSpec struct {
	Executable string
	Args       []string
	Dir        string
	Stdin      string
	Env        []string
}

// commandRunner is injectable so protocol tests never call a real model.
type commandRunner interface {
	Run(context.Context, commandSpec) (string, error)
}

type streamingCommandRunner interface {
	RunStream(context.Context, commandSpec, func(string)) (string, error)
}

type toolStreamingCommandRunner interface {
	RunStreamWithTools(context.Context, commandSpec, func(string), func(ToolTrace)) (string, error)
}

type osCommandRunner struct{}

type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
	// 头部写满后转入尾部环留：CLI 事件流的收尾（turn.completed 聚合全文、
	// 最终陈词）都在尾部，只留头部会把答案截丢——2026-10-31 实弹 zcode
	// 观点丢失根因（4MB cap 斩尾后 parse 只收到中途旁白）。
	tailStarted bool
	tail        []byte
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	if b.tailStarted {
		b.tail = append(b.tail, p...)
		if maxTail := b.limit * 3 / 4; len(b.tail) > maxTail {
			b.tail = append([]byte(nil), b.tail[len(b.tail)-maxTail:]...)
		}
		return original, nil
	}
	remaining := b.limit - b.buf.Len()
	if len(p) <= remaining {
		b.buf.Write(p)
		return original, nil
	}
	b.buf.Write(p[:remaining])
	b.truncated = true
	b.tailStarted = true
	rest := p[remaining:]
	if maxTail := b.limit * 3 / 4; len(rest) > maxTail {
		rest = rest[len(rest)-maxTail:]
	}
	b.tail = append([]byte(nil), rest...)
	return original, nil
}

func (b *cappedBuffer) String() string {
	if !b.tailStarted {
		return b.buf.String()
	}
	return b.buf.String() + "\n【进程输出超限，中段截断，以下保留完整尾部】\n" + string(b.tail)
}

func (r osCommandRunner) Run(ctx context.Context, spec commandSpec) (string, error) {
	return r.RunStream(ctx, spec, nil)
}

func (r osCommandRunner) RunStream(ctx context.Context, spec commandSpec, onDelta func(string)) (string, error) {
	return r.runStream(ctx, spec, onDelta, nil)
}

func (r osCommandRunner) RunStreamWithTools(
	ctx context.Context,
	spec commandSpec,
	onDelta func(string),
	onTool func(ToolTrace),
) (string, error) {
	return r.runStream(ctx, spec, onDelta, onTool)
}

func (r osCommandRunner) runStream(
	ctx context.Context,
	spec commandSpec,
	onDelta func(string),
	onTool func(ToolTrace),
) (string, error) {
	if strings.TrimSpace(spec.Executable) == "" {
		return "", errors.New("Agent 可执行文件为空")
	}
	cmd := exec.CommandContext(ctx, spec.Executable, spec.Args...)
	if goruntime.GOOS == "windows" {
		// Windows agent launchers are often batch/JS wrappers. Killing only the
		// direct process can orphan the real agent child tree.
		cmd.Cancel = func() error {
			if cmd.Process == nil {
				return os.ErrProcessDone
			}
			_ = exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
			return nil
		}
	}
	cmd.Dir = spec.Dir
	cmd.Stdin = strings.NewReader(spec.Stdin)
	if len(spec.Env) > 0 {
		cmd.Env = append(os.Environ(), spec.Env...)
	}
	stdout := &cappedBuffer{limit: maxProcessOutput}
	stderr := &cappedBuffer{limit: maxProcessOutput}
	var stream *streamTextWriter
	if onDelta != nil || onTool != nil {
		stream = &streamTextWriter{onDelta: onDelta, onTool: onTool}
		cmd.Stdout = io.MultiWriter(stdout, stream)
	} else {
		cmd.Stdout = stdout
	}
	cmd.Stderr = stderr
	cmd.WaitDelay = 3 * time.Second
	err := cmd.Run()
	if stream != nil {
		stream.Flush()
	}
	out := strings.TrimSpace(stdout.buf.String())
	errText := strings.TrimSpace(stderr.buf.String())
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, fmt.Errorf("Agent 调用超时: %w", ctx.Err())
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return out, context.Canceled
	}
	if err != nil {
		if errText != "" {
			return out, fmt.Errorf("%w: %s", err, trimRunes(errText, 1200))
		}
		return out, err
	}
	if stdout.truncated {
		out += "\n\n[输出已截断]"
	}
	return out, nil
}

type streamTextWriter struct {
	onDelta func(string)
	onTool  func(ToolTrace)
	pending []byte
	// sawDelta 标记本次调用已经流式播出过增量文本；此后到达的“全文回声”
	// 事件（codex 等 CLI 在增量 delta 之后还会发 item.completed/agent_message
	// 携带完整发言）必须吞掉，否则前端拼接后整段发言输出两遍。
	sawDelta bool
}

func (w *streamTextWriter) Write(p []byte) (int, error) {
	w.pending = append(w.pending, p...)
	w.drainJSONObjects()
	for {
		idx := bytes.IndexByte(w.pending, '\n')
		if idx < 0 {
			break
		}
		line := string(w.pending[:idx])
		w.pending = append([]byte(nil), w.pending[idx+1:]...)
		w.emitLine(line)
	}
	return len(p), nil
}

func (w *streamTextWriter) Flush() {
	w.drainJSONObjects()
	if len(w.pending) == 0 {
		return
	}
	line := string(w.pending)
	w.pending = nil
	w.emitLine(line)
}

// drainJSONObjects consumes complete JSON values even when ZCode emits many
// events on one physical line. Without this incremental split, streamed text
// waits until process exit and the final event wall is discarded as malformed.
func (w *streamTextWriter) drainJSONObjects() {
	for {
		data := bytes.TrimLeft(w.pending, " \t\r\n")
		if len(data) == 0 {
			w.pending = nil
			return
		}
		if data[0] != '{' {
			w.pending = data
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			w.pending = data
			return
		}
		consumed := int(decoder.InputOffset())
		w.emitObject(obj)
		w.pending = append([]byte(nil), data[consumed:]...)
	}
}

func (w *streamTextWriter) emitLine(line string) {
	if w.onTool != nil {
		if trace := toolTraceFromLine(line); trace != nil {
			w.onTool(*trace)
			return
		}
	}
	if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "{") {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &obj); err == nil && obj != nil {
			w.emitObject(obj)
			return
		}
	}
	text := streamTextFromLine(line)
	w.emitText(text)
}

func (w *streamTextWriter) emitObject(obj map[string]interface{}) {
	if w.onTool != nil {
		if trace := toolTraceFromObject(obj); trace != nil {
			w.onTool(*trace)
			return
		}
	}
	if w.sawDelta && finalMessageEcho(obj) {
		return
	}
	w.emitText(streamTextFromObject(obj))
}

// finalMessageEcho 识别携带完整发言的收尾事件（item.completed/agent_message、
// result、response.done 等）。只在增量已经播过流时拦：纯最终态 CLI（无 delta）
// 仍要靠这些事件出文本。
func finalMessageEcho(obj map[string]interface{}) bool {
	// zcode 信封聚合事件（turn.completed 携带全文）：增量已播过流则必须吞，
	// 否则前端拼接后整段发言输出两遍（2026-10-31 实弹重复输出根因之一）。
	if strings.EqualFold(zcodeEnvelopeKind(obj), "turn.completed") {
		return extractZcodeFinalText(obj) != ""
	}
	if item, ok := obj["item"].(map[string]interface{}); ok {
		itemType, _ := item["type"].(string)
		if itemType == "agent_message" || itemType == "assistant_message" {
			return true
		}
	}
	kind, _ := obj["type"].(string)
	lowerKind := strings.ToLower(kind)
	if strings.Contains(lowerKind, "complete") || strings.Contains(lowerKind, "result") ||
		strings.Contains(lowerKind, "response") || strings.Contains(lowerKind, "assistant") ||
		strings.EqualFold(kind, "agent_message") {
		return extractKnownText(obj) != "" || extractLooseText(obj) != ""
	}
	return false
}

func (w *streamTextWriter) emitText(text string) {
	if text == "" || w.onDelta == nil {
		return
	}
	for _, chunk := range streamChunks(text, 30) {
		w.onDelta(chunk)
		time.Sleep(10 * time.Millisecond)
	}
	w.sawDelta = true
}

func streamTextFromLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if !strings.HasPrefix(line, "{") && !strings.HasPrefix(line, "[") {
		return line + "\n"
	}
	var obj map[string]interface{}
	if json.Unmarshal([]byte(line), &obj) != nil {
		return ""
	}
	return streamTextFromObject(obj)
}

func streamTextFromObject(obj map[string]interface{}) string {
	// Codex 把 Harness 级告警（如未知 config 键提示）包成 item.completed/error
	// 事件，每轮都会重复播报；这不是发言内容，混进流会变成大段英文噪音。
	// 真正的调用失败走进程退出码 → agent_error，不依赖这里透传。
	if harnessErrorItem(obj) {
		return ""
	}
	kind, _ := obj["type"].(string)
	// zcode 新信封（无顶层 type）：text_delta 直提直播；turn.completed 是
	// 聚合全文，走下方 echo 门（增量已播则吞，纯最终态则播一次）。
	if text, ok := extractTextDelta(obj); ok {
		return text
	}
	if text := extractZcodeFinalText(obj); text != "" {
		return text
	}
	if strings.EqualFold(kind, "text") {
		if text, _ := obj["text"].(string); strings.TrimSpace(text) != "" {
			return text
		}
	}
	if strings.EqualFold(kind, "model.streaming") {
		payload, _ := obj["payload"].(map[string]interface{})
		payloadKind, _ := payload["kind"].(string)
		if strings.EqualFold(payloadKind, "text_delta") {
			if text, _ := payload["delta"].(string); text != "" {
				return text
			}
		}
	}
	if text := extractStreamDelta(obj); text != "" {
		return text
	}
	if text := extractKnownText(obj); text != "" {
		return text
	}
	lowerKind := strings.ToLower(kind)
	if strings.Contains(lowerKind, "result") || strings.Contains(lowerKind, "complete") ||
		strings.Contains(lowerKind, "response") || strings.Contains(lowerKind, "assistant") {
		return extractLooseText(obj)
	}
	return ""
}

func harnessErrorItem(obj map[string]interface{}) bool {
	item, _ := obj["item"].(map[string]interface{})
	if item == nil {
		return false
	}
	itemType, _ := item["type"].(string)
	return strings.EqualFold(itemType, "error")
}

func toolTraceFromLine(line string) *ToolTrace {
	line = strings.TrimSpace(line)
	if line == "" || !strings.HasPrefix(line, "{") {
		return nil
	}
	var obj map[string]interface{}
	if json.Unmarshal([]byte(line), &obj) != nil {
		return nil
	}
	return toolTraceFromObject(obj)
}

func toolTraceFromObject(obj map[string]interface{}) *ToolTrace {
	kind := strings.ToLower(traceString(obj["type"]))
	switch kind {
	case "item.started", "item.completed":
		item, _ := obj["item"].(map[string]interface{})
		if item == nil {
			return nil
		}
		itemType := strings.ToLower(traceString(item["type"]))
		if itemType == "" || itemType == "agent_message" || itemType == "assistant_message" ||
			itemType == "reasoning" || itemType == "error" || itemType == "todo_list" {
			return nil
		}
		status := "started"
		if kind == "item.completed" {
			status = "completed"
			if code, ok := item["exit_code"].(float64); ok && code != 0 {
				status = "failed"
			}
			if traceString(item["error"]) != "" || strings.EqualFold(traceString(item["status"]), "failed") {
				status = "failed"
			}
		}
		return &ToolTrace{
			ID:     traceString(item["id"]),
			Name:   codexToolName(itemType, item),
			Status: status,
			Input:  codexToolInput(itemType, item),
			Output: previewTraceValue(firstTraceValue(
				item, "aggregated_output", "result", "output", "changes",
			)),
			Error: traceString(item["error"]),
		}
	case "tool_call":
		return &ToolTrace{
			ID:     firstTraceString(obj, "callId", "toolCallId", "id"),
			Name:   firstTraceString(obj, "tool", "toolName", "name"),
			Status: "started",
			Input:  previewTraceValue(firstTraceValue(obj, "input", "arguments", "args")),
		}
	case "tool_result":
		status := "completed"
		result, _ := obj["result"].(map[string]interface{})
		if strings.Contains(strings.ToLower(traceString(obj["status"])), "fail") ||
			strings.Contains(strings.ToLower(traceString(obj["status"])), "error") ||
			(result != nil && result["success"] == false) {
			status = "failed"
		}
		return &ToolTrace{
			ID:     firstTraceString(obj, "callId", "toolCallId", "id"),
			Name:   firstTraceString(obj, "tool", "toolName", "name"),
			Status: status,
			Output: traceResultOutput(firstTraceValue(obj, "result", "output", "content")),
			Error:  firstTraceString(obj, "error", "err"),
		}
	case "tool.updated":
		payload, _ := obj["payload"].(map[string]interface{})
		if payload == nil {
			return nil
		}
		payloadKind := strings.ToLower(traceString(payload["kind"]))
		status := ""
		switch payloadKind {
		case "result":
			status = "completed"
			result, _ := payload["result"].(map[string]interface{})
			if result != nil && result["success"] == false {
				status = "failed"
			}
		default:
			return nil
		}
		return &ToolTrace{
			ID:     firstTraceString(payload, "toolCallId", "id"),
			Name:   firstTraceString(payload, "toolName", "name"),
			Status: status,
			Input:  previewTraceValue(firstTraceValue(payload, "input", "arguments")),
			Output: traceResultOutput(firstTraceValue(payload, "result", "output", "content")),
			Error:  firstTraceString(payload, "error", "err"),
		}
	case "model.streaming":
		payload, _ := obj["payload"].(map[string]interface{})
		if payload == nil {
			return nil
		}
		payloadKind := strings.ToLower(traceString(payload["kind"]))
		if payloadKind != "tool_call" {
			return nil
		}
		return &ToolTrace{
			ID:     firstTraceString(payload, "toolCallId", "id"),
			Name:   firstTraceString(payload, "toolName", "name"),
			Status: "started",
			Input:  previewTraceValue(firstTraceValue(payload, "input", "arguments")),
		}
	}
	return nil
}

func codexToolName(itemType string, item map[string]interface{}) string {
	switch itemType {
	case "command_execution":
		return "命令"
	case "mcp_tool_call":
		server := traceString(item["server"])
		tool := firstTraceString(item, "tool", "name")
		if server != "" && tool != "" {
			return server + "." + tool
		}
		if tool != "" {
			return tool
		}
		return "MCP 工具"
	case "file_change":
		return "文件变更"
	case "web_search":
		return "网页搜索"
	case "collab_tool_call":
		return "子代理"
	case "computer_call":
		return "电脑操作"
	default:
		if name := firstTraceString(item, "tool", "name"); name != "" {
			return name
		}
		return itemType
	}
}

func codexToolInput(itemType string, item map[string]interface{}) string {
	if itemType == "command_execution" {
		return previewTraceValue(item["command"])
	}
	for _, key := range []string{"arguments", "input", "query", "path"} {
		if value, ok := item[key]; ok {
			return previewTraceValue(value)
		}
	}
	return ""
}

func firstTraceString(obj map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := traceString(obj[key]); value != "" {
			return value
		}
	}
	return ""
}

func firstTraceValue(obj map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		if value, ok := obj[key]; ok && value != nil {
			return value
		}
	}
	return nil
}

func traceString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func previewTraceValue(value interface{}) string {
	if value == nil {
		return ""
	}
	var text string
	switch typed := value.(type) {
	case string:
		text = typed
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		text = string(data)
	}
	return trimRunes(strings.TrimSpace(text), 1400)
}

func traceResultOutput(value interface{}) string {
	if obj, ok := value.(map[string]interface{}); ok {
		for _, key := range []string{"content", "output", "text", "message"} {
			if text := traceString(obj[key]); text != "" {
				return trimRunes(text, 1400)
			}
		}
	}
	return previewTraceValue(value)
}

func extractStreamDelta(value interface{}) string {
	obj, ok := value.(map[string]interface{})
	if !ok {
		return ""
	}
	for _, key := range []string{"delta", "text_delta", "content_delta", "output_text", "reasoning_delta"} {
		switch value := obj[key].(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return value
			}
		case map[string]interface{}:
			if text := extractLooseText(value); text != "" {
				return text
			}
		}
	}
	for _, key := range []string{"data", "item", "message", "delta"} {
		if nested, ok := obj[key].(map[string]interface{}); ok {
			if text := extractStreamDelta(nested); text != "" {
				return text
			}
		}
	}
	return ""
}

func streamChunks(text string, maxChunks int) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	if maxChunks < 1 {
		maxChunks = 1
	}
	size := 32
	if chunks := (len(runes) + size - 1) / size; chunks > maxChunks {
		size = (len(runes) + maxChunks - 1) / maxChunks
	}
	out := make([]string, 0, (len(runes)+size-1)/size)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
	}
	return out
}

func trimRunes(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "…"
}

func tailRunes(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[len(r)-limit:])
}

// promptExceedsBudget reports whether clampPrompt will clip this prompt. The
// caller records it as an input-truncation stop signal instead of letting the
// loss silently pass as substantive disagreement.
func promptExceedsBudget(prompt string) bool {
	return len([]rune(prompt)) > maxPromptRunes
}

func clampPrompt(s string) string {
	runes := []rune(s)
	if len(runes) <= maxPromptRunes {
		return s
	}
	// Keep both ends: the opening carries the brief and collaboration rules,
	// while the tail carries the active task and response-format contract.
	// A plain head truncation silently removes the instructions and turns long
	// councils into "unexpected prompt" failures.
	marker := []rune(promptClipMarker)
	budget := maxPromptRunes - len(marker)
	head := budget / 3
	tail := budget - head
	return string(runes[:head]) +
		promptClipMarker +
		string(runes[len(runes)-tail:])
}

func parseAgentOutput(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var texts []string
	var generic string
	var stream strings.Builder
	zcodeFinal := ""
	sawEnvelopeFinish := false
	sawPlainLine := false
	onlyHarnessErrors := true
	consumeObject := func(obj map[string]interface{}) {
		if harnessErrorItem(obj) {
			// Harness 告警不算发言内容；见函数尾部的空结果判定。
			return
		}
		onlyHarnessErrors = false
		if text := extractZcodeFinalText(obj); text != "" {
			zcodeFinal = text
			sawEnvelopeFinish = true
			return
		}
		if kind := zcodeEnvelopeKind(obj); strings.EqualFold(kind, "finish") {
			sawEnvelopeFinish = true
		}
		if text := extractKnownText(obj); text != "" {
			texts = append(texts, text)
			return
		}
		if delta, ok := extractTextDelta(obj); ok {
			// zcode 新版桌面运行时（eventId/payload 信封）把答案拆成
			// text_delta 增量：按序收集，见函数尾部重装分支
			// （2026-10-30 实弹 200k 事件墙乱码线）。
			stream.WriteString(delta)
			return
		}
		if text := extractLooseText(obj); text != "" {
			generic = text
		}
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			// ZCode 会把多个完整 JSON 事件以空格拼接在同一行（无换行），
			// json.Unmarshal 因此失败。用 Decoder 按连续 JSON 值拆开，
			// 避免整段 eventId/messageId/payload 事件墙被存成席位发言。
			objects, complete := decodeJSONObjectSequence(line)
			if len(objects) > 0 {
				for _, item := range objects {
					consumeObject(item)
				}
				if complete {
					continue
				}
			}
			if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
				// 被输出上限斩断的 JSON 残段：是事件流的一部分，不是发言
				// 文本——不得触发整体兜底把事件墙原样入库（2026-10-31 实弹
				// zcode 第 2 轮 200k 墙的入口）。
				continue
			}
			// 纯文本行是合法输出形态，保留最终的整体兜底。
			sawPlainLine = true
			onlyHarnessErrors = false
			continue
		}
		consumeObject(obj)
	}
	if len(texts) > 0 {
		return trimRunes(strings.Join(texts, "\n\n"), maxAgentOutput)
	}
	if zcodeFinal != "" {
		return trimRunes(zcodeFinal, maxAgentOutput)
	}
	if generic != "" {
		return trimRunes(generic, maxAgentOutput)
	}
	if rebuilt := strings.TrimSpace(stream.String()); rebuilt != "" {
		// 逐段 delta 重装只信「信封收尾完成」的流：finish/turn.completed 缺席
		// 意味着调用中途夭折（超时/被杀），此时只有过程旁白、没有最终陈词——
		// 宁可空结果走 agent_error，也不把旁白存成观点（2026-10-31 实弹
		// zcode 第 1/3 轮 149/187 字旁白冒充观点的根因）。
		if sawEnvelopeFinish {
			return trimRunes(rebuilt, maxAgentOutput)
		}
		return ""
	}
	if onlyHarnessErrors {
		// 整段输出只有 Harness 告警：按空结果交上层走 agent_error，
		// 而不是把配置警告存成席位陈词。
		return ""
	}
	if !sawPlainLine {
		// 单个 JSON 对象（如 synthesis 裁决 JSON，键集在 loose 清单之外）
		// 原样放行给上层解析；多行 JSONL 事件流且没认出任何文本（未知新
		// schema 的纯事件墙）则空结果走 agent_error——宁可席位失败可见，
		// 也不把事件流原样存成席位陈词。
		if json.Valid([]byte(raw)) {
			return trimRunes(raw, maxAgentOutput)
		}
		return ""
	}
	// A CLI may emit a single JSON object with a response field.
	var obj map[string]interface{}
	if json.Unmarshal([]byte(raw), &obj) == nil {
		if text := extractLooseText(obj); text != "" {
			return trimRunes(text, maxAgentOutput)
		}
	}
	return trimRunes(raw, maxAgentOutput)
}

func decodeJSONObjectSequence(raw string) ([]map[string]interface{}, bool) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	objects := make([]map[string]interface{}, 0, 8)
	for {
		var obj map[string]interface{}
		err := decoder.Decode(&obj)
		if errors.Is(err, io.EOF) {
			return objects, len(objects) > 0
		}
		if err != nil {
			return objects, false
		}
		objects = append(objects, obj)
	}
}

// repairLegacyZCodeEventWall keeps historical records usable after older builds
// saved ZCode's concatenated JSON transport as the visible contribution.
func repairLegacyZCodeEventWall(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, `{"eventId"`) ||
		!strings.Contains(trimmed, `"messageId"`) {
		return text
	}
	if parsed := parseAgentOutput(trimmed); parsed != "" && parsed != trimmed {
		return parsed
	}
	// 历史墙多来自被截断/中途夭折的流（无 finish 收尾），活线的完成门会把
	// 它们判空。抢救已入库的历史记录时放宽：按增量重装可读正文即可。
	if salvaged := salvageZcodeDeltaText(trimmed); salvaged != "" {
		return salvaged
	}
	return "ZCode 返回了未解析的事件流；请重试本轮观点。"
}

// salvageZcodeDeltaText 按 text_delta 增量重装正文，不做完成门——仅用于
// 历史事件墙的读取期抢救，活线解析走 parseAgentOutput 的严格路径。
func salvageZcodeDeltaText(raw string) string {
	var stream strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		objects, _ := decodeJSONObjectSequence(line)
		for _, obj := range objects {
			if delta, ok := extractTextDelta(obj); ok {
				stream.WriteString(delta)
			}
		}
	}
	return strings.TrimSpace(stream.String())
}

// extractTextDelta 识别 zcode stream-json 的 text_delta 增量载荷。
func extractTextDelta(obj map[string]interface{}) (string, bool) {
	payload, _ := obj["payload"].(map[string]interface{})
	if payload == nil {
		return "", false
	}
	if !strings.EqualFold(traceString(payload["kind"]), "text_delta") {
		return "", false
	}
	delta, _ := payload["delta"].(string)
	return delta, delta != ""
}

// zcodeEnvelopeKind 返回 zcode 新版运行时（eventId/payload 信封，无顶层 type）
// 事件的 payload.kind；非信封事件返回空。
func zcodeEnvelopeKind(obj map[string]interface{}) string {
	payload, _ := obj["payload"].(map[string]interface{})
	if payload == nil {
		return ""
	}
	kind, _ := payload["kind"].(string)
	return kind
}

// extractZcodeFinalText 提取信封聚合全文事件（payload.kind=turn.completed 携带
// response 全文）。这是收尾权威文本，优先于逐段 text_delta 重装——进程输出被
// 截断时尾部的聚合事件是唯一完整的最终答案（2026-10-31 实弹）。
func extractZcodeFinalText(obj map[string]interface{}) string {
	if !strings.EqualFold(zcodeEnvelopeKind(obj), "turn.completed") {
		return ""
	}
	payload, _ := obj["payload"].(map[string]interface{})
	text, _ := payload["response"].(string)
	return strings.TrimSpace(text)
}

func extractKnownText(obj map[string]interface{}) string {
	kind, _ := obj["type"].(string)
	if kind == "agent_message" || kind == "assistant_message" || kind == "result" {
		for _, key := range []string{"text", "response", "result", "message", "output"} {
			if text, ok := obj[key].(string); ok && strings.TrimSpace(text) != "" {
				return text
			}
		}
	}
	if item, ok := obj["item"].(map[string]interface{}); ok {
		itemType, _ := item["type"].(string)
		if itemType == "agent_message" || itemType == "assistant_message" {
			if text, ok := item["text"].(string); ok {
				return text
			}
		}
	}
	return ""
}

func extractLooseText(obj map[string]interface{}) string {
	for _, key := range []string{"response", "result", "text", "message", "output", "content"} {
		if text, ok := obj[key].(string); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	for _, key := range []string{"item", "message", "result"} {
		if nested, ok := obj[key].(map[string]interface{}); ok {
			if text := extractLooseText(nested); text != "" {
				return text
			}
		}
	}
	return ""
}

func localAppData() string {
	if v := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); v != "" {
		return v
	}
	if v, err := os.UserConfigDir(); err == nil {
		return filepath.Dir(v)
	}
	return ""
}

func executableExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func windowsPath(parts ...string) string {
	if goruntime.GOOS != "windows" {
		return ""
	}
	base := localAppData()
	if base == "" {
		return ""
	}
	return filepath.Join(append([]string{base}, parts...)...)
}
