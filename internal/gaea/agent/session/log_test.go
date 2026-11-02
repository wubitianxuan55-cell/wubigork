package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// ─── append-only 写入器：seq 递增 ────────────────────────────────

func TestLogWriterSeqIncrements(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, err := OpenLog(logPath, "", "")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	seq1, err := w.Append(KindUserMessage, userLogPayload{Content: "hi"})
	if err != nil {
		t.Fatalf("append 1: %v", err)
	}
	seq2, err := w.Append("turn_done", map[string]string{"err": ""})
	if err != nil {
		t.Fatalf("append 2: %v", err)
	}
	if seq1 != 1 || seq2 != 2 {
		t.Fatalf("seq = %d,%d want 1,2", seq1, seq2)
	}
	if w.Seq() != 2 {
		t.Fatalf("writer seq = %d, want 2", w.Seq())
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// 重开续 seq：已写 2 行，下一行 seq=3
	w2, err := OpenLog(logPath, "", "")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer w2.Close()
	if w2.Seq() != 2 {
		t.Fatalf("reopened seq = %d, want 2", w2.Seq())
	}
	seq3, err := w2.Append("turn_done", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if seq3 != 3 {
		t.Fatalf("seq3 = %d, want 3", seq3)
	}

	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	for i, e := range entries {
		if e.Seq != int64(i+1) {
			t.Errorf("entry %d seq = %d, want %d", i, e.Seq, i+1)
		}
	}
	if entries[0].Kind != KindUserMessage || entries[1].Kind != "turn_done" {
		t.Errorf("kinds = %s,%s", entries[0].Kind, entries[1].Kind)
	}
}

func TestLogWriterJsonValidRejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, err := OpenLog(logPath, "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// 非法 payload：拒绝写入，seq 不变，文件不产生行
	_, err = w.AppendRaw("text", json.RawMessage(`{"broken":`))
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
	if !strings.Contains(err.Error(), "not lossless JSON") {
		t.Errorf("error should mention rejection: %v", err)
	}
	if w.Seq() != 0 {
		t.Fatalf("seq after rejected write = %d, want 0", w.Seq())
	}
	entries, _ := ReadLog(logPath)
	if len(entries) != 0 {
		t.Fatalf("log should be empty after rejected write, got %d entries", len(entries))
	}

	// 合法 payload 正常写入
	if _, err := w.AppendRaw("text", json.RawMessage(`{"text":"ok"}`)); err != nil {
		t.Fatalf("append valid: %v", err)
	}
	if w.Seq() != 1 {
		t.Fatalf("seq = %d, want 1", w.Seq())
	}
}

func TestLogWriterAppendRejectsUnmarshalable(t *testing.T) {
	w, err := OpenLog(filepath.Join(t.TempDir(), "s.gaea-log.jsonl"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	_, err = w.Append("text", make(chan int))
	if err == nil {
		t.Fatal("expected error marshaling a channel")
	}
	if w.Seq() != 0 {
		t.Fatalf("seq = %d, want 0", w.Seq())
	}
}

// ─── torn-tail 截断 ──────────────────────────────────────────────

func TestRepairLogFileTruncatesTornTail(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, _ := OpenLog(logPath, "", "")
	w.Append("turn_started", map[string]string{})
	w.Append("turn_done", map[string]string{})
	w.Close()
	// 模拟崩溃：追加一条不完整行
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString(`{"seq":3,"ts":1,"kind":"text","payload":{"te`)
	f.Close()

	truncated, err := RepairLogFile(logPath)
	if err != nil {
		t.Fatalf("RepairLogFile: %v", err)
	}
	if !truncated {
		t.Fatal("expected truncation")
	}
	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2 (torn tail removed)", len(entries))
	}
	if entries[len(entries)-1].Kind != "turn_done" {
		t.Errorf("last kind = %s, want turn_done", entries[len(entries)-1].Kind)
	}
}

func TestRepairLogFileNoopOnCleanTail(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, _ := OpenLog(logPath, "", "")
	w.Append("turn_started", map[string]string{})
	w.Close()
	truncated, err := RepairLogFile(logPath)
	if err != nil || truncated {
		t.Fatalf("clean tail: truncated=%v err=%v", truncated, err)
	}
	// 空文件 / 不存在
	truncated, err = RepairLogFile(filepath.Join(dir, "missing.jsonl"))
	if err != nil || truncated {
		t.Fatalf("missing file: truncated=%v err=%v", truncated, err)
	}
}

// ─── 合成 closers ────────────────────────────────────────────────

func TestBalanceEntriesSyntheticCloser(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, _ := OpenLog(logPath, "", "")
	w.Append("turn_started", map[string]string{})
	w.Append("tool_result", toolResultLogPayload{ID: "t1", Output: "ok"})
	w.Close()
	entries, _ := ReadLog(logPath)
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	balanced := BalanceEntries(entries)
	if len(balanced) != 3 {
		t.Fatalf("balanced = %d, want 3 (synthetic closer)", len(balanced))
	}
	last := balanced[len(balanced)-1]
	if last.Kind != "turn_done" {
		t.Errorf("closer kind = %s, want turn_done", last.Kind)
	}
	if last.Seq != 3 {
		t.Errorf("closer seq = %d, want 3", last.Seq)
	}
	var p map[string]any
	json.Unmarshal(last.Payload, &p)
	if p["synthetic"] != true {
		t.Errorf("closer payload = %v, want synthetic=true", p)
	}
}

func TestBalanceEntriesNoopOnTurnDone(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: "turn_started", Payload: json.RawMessage(`{}`)},
		{Seq: 2, Kind: "turn_done", Payload: json.RawMessage(`{}`)},
	}
	if got := BalanceEntries(entries); len(got) != 2 {
		t.Fatalf("balanced = %d, want 2 (already closed)", len(got))
	}
	if got := BalanceEntries(nil); got != nil {
		t.Fatalf("empty balance = %v, want nil", got)
	}
}

// ─── 路径 ────────────────────────────────────────────────────────

func TestLogPaths(t *testing.T) {
	got := LogPathFor(`C:\x\sessions\a.jsonl`)
	want := `C:\x\sessions\a.gaea-log.jsonl`
	if got != want {
		t.Fatalf("LogPathFor = %q, want %q", got, want)
	}
	if LogPathFor("") != "" {
		t.Fatal("LogPathFor(empty) should be empty")
	}
	cp := CheckpointPathFor(`C:\x\sessions\a.jsonl`)
	if cp != `C:\x\sessions\a.gaea-checkpoint.json` {
		t.Fatalf("CheckpointPathFor = %q", cp)
	}
}

// ─── event.Kind → 日志条目 ───────────────────────────────────────

func TestEntryFromEventAllKinds(t *testing.T) {
	usage := &provider.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, CacheHitTokens: 40, CacheMissTokens: 60}
	price := &provider.Pricing{Input: 1, Output: 2, CacheHit: 0.5, Currency: "¥"}
	events := []event.Event{
		{Kind: event.TurnStarted},
		{Kind: event.Reasoning, Text: "thinking"},
		{Kind: event.Text, Text: "hello"},
		{Kind: event.Message, Text: "hello", Reasoning: "thinking"},
		{Kind: event.ToolDispatch, Tool: event.Tool{ID: "t1", Name: "bash", Args: `{"cmd":"ls"}`}},
		{Kind: event.ToolResult, Tool: event.Tool{ID: "t1", Output: "out"}},
		{Kind: event.Usage, Usage: usage, Pricing: price, UsageSource: "main", Turn: 3},
		{Kind: event.Notice, Level: event.LevelWarn, Text: "n"},
		{Kind: event.Phase, Text: "executing"},
		{Kind: event.ApprovalRequest, Approval: event.Approval{ID: "a1", Tool: "bash", Subject: "rm"}},
		{Kind: event.AskRequest, Ask: event.Ask{ID: "q1", Questions: []event.AskQuestion{{ID: "q1", Prompt: "继续?"}}}},
		{Kind: event.TurnDone},
		{Kind: event.CompactionStarted, Compaction: event.Compaction{Trigger: "auto"}},
		{Kind: event.CompactionDone, Compaction: event.Compaction{Trigger: "auto", Messages: 3, Summary: "s"}},
		{Kind: event.Retrying, RetryAttempt: 1, RetryMax: 2},
		{Kind: event.Steer, Text: "stop"},
		// v4.26 对话流式重造：子代理完成回投（ref/parentId 可空）。
		{Kind: event.SubagentMessage, Text: "sub answer", SubagentRef: "sa_1", ParentToolID: "call-9"},
		{Kind: event.SubagentMessage, Text: "临时子代理答复"},
	}
	for i, e := range events {
		entry, err := EntryFromEvent(e, 1700000000)
		if err != nil {
			t.Fatalf("entry %d: %v", i, err)
		}
		if entry.Kind == "" || entry.Kind == "unknown" {
			t.Errorf("entry %d: unmapped kind %q", i, entry.Kind)
		}
		if entry.Ts != 1700000000 {
			t.Errorf("entry %d: ts = %d", i, entry.Ts)
		}
		if !json.Valid(entry.Payload) {
			t.Errorf("entry %d: payload not valid JSON: %s", i, entry.Payload)
		}
	}
	// 事件级 kind 字符串抽查
	if KindString(event.TurnStarted) != "turn_started" || KindString(event.Usage) != "usage" || KindString(event.ToolResult) != KindToolResult {
		t.Error("kind string mapping broken")
	}
}

// TestEntryFromEventAttentionPayloads 锁定「注意力事务」判别字段的持久化：
// 实时事件里早就有的 source/reason/request 不再被日志层丢掉，结果侧事件
// 以同一 id 成对落盘（旧日志只有请求，这是阶段 1 补的洞）。
func TestEntryFromEventAttentionPayloads(t *testing.T) {
	req, err := EntryFromEvent(event.Event{
		Kind: event.ApprovalRequest,
		Approval: event.Approval{
			ID: "a1", Tool: "cost_save", Subject: "写入成本库",
			Reason: "跨库写入", Request: true, Source: event.ApprovalSourcePermissionRequest,
		},
	}, 1700000000)
	if err != nil {
		t.Fatalf("approval_request: %v", err)
	}
	var reqPayload map[string]any
	if err := json.Unmarshal(req.Payload, &reqPayload); err != nil {
		t.Fatalf("unmarshal request payload: %v", err)
	}
	if req.Kind != "approval_request" {
		t.Errorf("kind = %q, want approval_request", req.Kind)
	}
	if reqPayload["source"] != "permission_request" {
		t.Errorf("source = %v, want permission_request（判别字段必须进真相层）", reqPayload["source"])
	}
	if reqPayload["reason"] != "跨库写入" || reqPayload["request"] != true {
		t.Errorf("reason/request = %v/%v, want 跨库写入/true", reqPayload["reason"], reqPayload["request"])
	}

	resolved, err := EntryFromEvent(event.Event{
		Kind: event.ApprovalResolved,
		Resolved: event.ApprovalResolution{
			ID: "a1", Tool: "cost_save", Subject: "写入成本库",
			Source: event.ApprovalSourcePermissionRequest, Decision: "allow_once",
			Outcome: event.OutcomeAllowed, LatencyMS: 1500,
		},
	}, 1700000001)
	if err != nil {
		t.Fatalf("approval_resolved: %v", err)
	}
	var resPayload map[string]any
	if err := json.Unmarshal(resolved.Payload, &resPayload); err != nil {
		t.Fatalf("unmarshal resolved payload: %v", err)
	}
	if resolved.Kind != "approval_resolved" {
		t.Errorf("kind = %q, want approval_resolved", resolved.Kind)
	}
	for k, want := range map[string]any{
		"id": "a1", "source": "permission_request", "decision": "allow_once",
		"outcome": "allowed", "latency_ms": float64(1500),
	} {
		if resPayload[k] != want {
			t.Errorf("resolved[%s] = %v, want %v", k, resPayload[k], want)
		}
	}

	resolvedAsk, err := EntryFromEvent(event.Event{
		Kind:        event.AskResolved,
		AskResolved: event.AskResolution{ID: "q1", Outcome: event.OutcomeAnswered, Answered: 1, LatencyMS: 800},
	}, 1700000002)
	if err != nil {
		t.Fatalf("ask_resolved: %v", err)
	}
	var askPayload map[string]any
	if err := json.Unmarshal(resolvedAsk.Payload, &askPayload); err != nil {
		t.Fatalf("unmarshal ask_resolved payload: %v", err)
	}
	if resolvedAsk.Kind != "ask_resolved" || askPayload["outcome"] != "answered" || askPayload["answered"] != float64(1) {
		t.Errorf("ask_resolved = %s %v, want ask_resolved/answered/1", resolvedAsk.Kind, askPayload)
	}

	if KindString(event.ApprovalResolved) != "approval_resolved" || KindString(event.AskResolved) != "ask_resolved" {
		t.Error("resolved kind string mapping broken")
	}
}

// T4：回合发起者判别字段进真相层。唤醒轮此前只能靠 wakeJobInput 的
// "[system]" 文本前缀认出来——提示文案一改就误分类，也无法把人的回合与
// 后台唤醒回合分开计数。类型化后 turn_started.payload.trigger 直接落盘；
// 空 trigger（旧事件/旧日志）保持 {} 与旧格式逐字节一致。
func TestEntryFromEventTurnTrigger(t *testing.T) {
	wake, err := EntryFromEvent(event.Event{Kind: event.TurnStarted, Trigger: event.TriggerWake}, 1700000000)
	if err != nil {
		t.Fatalf("turn_started: %v", err)
	}
	var p map[string]any
	if err := json.Unmarshal(wake.Payload, &p); err != nil {
		t.Fatalf("unmarshal turn_started payload: %v", err)
	}
	if wake.Kind != "turn_started" || p["trigger"] != "wake" {
		t.Errorf("turn_started = %s %v, want turn_started/trigger=wake", wake.Kind, p)
	}
	legacy, err := EntryFromEvent(event.Event{Kind: event.TurnStarted}, 1700000001)
	if err != nil {
		t.Fatalf("legacy turn_started: %v", err)
	}
	if string(legacy.Payload) != "{}" {
		t.Errorf("legacy payload = %s, want {}（旧格式不变）", legacy.Payload)
	}
}

// ─── 消息 → 日志条目 ─────────────────────────────────────────────

func TestToLogEntries(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleAssistant, Content: "a1", ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read_file", Arguments: "{}"}}},
		{Role: provider.RoleTool, Content: "r1", ToolCallID: "c1", Name: "read_file"},
	}
	entries := ToLogEntries(msgs)
	if len(entries) != 7 {
		t.Fatalf("entries = %d, want 7 (回合边界 + 合成 request_header)", len(entries))
	}
	wantKinds := []string{KindSystemMessage, "turn_started", KindUserMessage, KindRequestHeader, KindAssistantMessage, KindToolResult, "turn_done"}
	for i, k := range wantKinds {
		if entries[i].Kind != k {
			t.Errorf("entry %d kind = %s, want %s", i, entries[i].Kind, k)
		}
		if entries[i].Seq != int64(i+1) {
			t.Errorf("entry %d seq = %d, want %d", i, entries[i].Seq, i+1)
		}
	}
	// 投影往返：回合边界不投影为消息
	if projected := ProjectMessages(entries); len(projected) != 4 {
		t.Fatalf("projected = %d messages, want 4", len(projected))
	}
	// 合成 request_header 携带真实 system 与工具名
	var hdr struct {
		System string `json:"system"`
		Tools  []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(entries[3].Payload, &hdr); err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if hdr.System != "sys" || len(hdr.Tools) != 1 || hdr.Tools[0].Name != "read_file" {
		t.Fatalf("synthesized header = %+v", hdr)
	}
}

// ─── 并发追加（写入器单点保证）───────────────────────────────────

func TestLogWriterConcurrentAppend(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "s.gaea-log.jsonl")
	w, err := OpenLog(logPath, "", "")
	if err != nil {
		t.Fatal(err)
	}
	const n = 64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := w.Append("text", map[string]int{"i": i}); err != nil {
				t.Errorf("append %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	seq := w.Seq()
	w.Close()
	if seq != n {
		t.Fatalf("seq = %d, want %d", seq, n)
	}
	entries, err := ReadLog(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != n {
		t.Fatalf("entries = %d, want %d (无丢失)", len(entries), n)
	}
	seen := map[int64]bool{}
	for _, e := range entries {
		if seen[e.Seq] {
			t.Fatalf("duplicate seq %d", e.Seq)
		}
		seen[e.Seq] = true
	}
}
