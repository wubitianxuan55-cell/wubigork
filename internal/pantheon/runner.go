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
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	if b.buf.Len() >= b.limit {
		b.truncated = true
		return original, nil
	}
	remaining := b.limit - b.buf.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, _ = b.buf.Write(p)
	return original, nil
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
}

func (w *streamTextWriter) Write(p []byte) (int, error) {
	w.pending = append(w.pending, p...)
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
	if len(w.pending) == 0 {
		return
	}
	line := string(w.pending)
	w.pending = nil
	w.emitLine(line)
}

func (w *streamTextWriter) emitLine(line string) {
	if w.onTool != nil {
		if trace := toolTraceFromLine(line); trace != nil {
			w.onTool(*trace)
			return
		}
	}
	text := streamTextFromLine(line)
	if text == "" || w.onDelta == nil {
		return
	}
	for _, chunk := range streamChunks(text, 30) {
		w.onDelta(chunk)
		time.Sleep(10 * time.Millisecond)
	}
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
	// Codex 把 Harness 级告警（如未知 config 键提示）包成 item.completed/error
	// 事件，每轮都会重复播报；这不是发言内容，混进流会变成大段英文噪音。
	// 真正的调用失败走进程退出码 → agent_error，不依赖这里透传。
	if harnessErrorItem(obj) {
		return ""
	}
	kind, _ := obj["type"].(string)
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
	onlyHarnessErrors := true
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			// 纯文本行是合法输出形态，保留最终的整体兜底。
			onlyHarnessErrors = false
			continue
		}
		if harnessErrorItem(obj) {
			// Harness 告警不算发言内容；见函数尾部的空结果判定。
			continue
		}
		onlyHarnessErrors = false
		if text := extractKnownText(obj); text != "" {
			texts = append(texts, text)
		} else if text := extractLooseText(obj); text != "" {
			generic = text
		}
	}
	if len(texts) > 0 {
		return trimRunes(strings.Join(texts, "\n\n"), maxAgentOutput)
	}
	if generic != "" {
		return trimRunes(generic, maxAgentOutput)
	}
	if onlyHarnessErrors {
		// 整段输出只有 Harness 告警：按空结果交上层走 agent_error，
		// 而不是把配置警告存成席位陈词。
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
