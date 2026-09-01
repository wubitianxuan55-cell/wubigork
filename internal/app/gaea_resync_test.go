package app

// v4.26 对话流式重造：事件补拉（防吞件）测试。
// 覆盖：转发层 wire seq 单调 + phase 200ms 节流；gaeaEventMap 的
// Retrying/compaction → phase 转译与 subagent_message 扁平化；最小折叠器
//（基础折叠/断号日志/悬挂 tool_result/空与未知 kind 容错）；绑定面
// GaeaResyncEvents 的早退与磁盘日志折叠。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// resyncEntry 构造一条日志条目（payload 自动序列化，非法输入直接 panic——
// 测试固定形状）。
func resyncEntry(seq int64, kind string, payload map[string]any) session.LogEntry {
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return session.LogEntry{Seq: seq, Ts: 1700000000, Kind: kind, Payload: raw}
}

// TestGaeaWireForwarderSeqAndThrottle 转发层：seq 严格单调递增；同一 phase
// 文案 200ms 内不重发（节流不消费 seq，保证转发出去的 seq 无断号）；不同
// 文案互不影响；窗口过后恢复转发。
func TestGaeaWireForwarderSeqAndThrottle(t *testing.T) {
	f := newGaeaEventForwarder()

	m1 := f.payload(event.Event{Kind: event.Text, Text: "hello"})
	m2 := f.payload(event.Event{Kind: event.Text, Text: "world"})
	if m1["seq"].(int64) != 1 || m2["seq"].(int64) != 2 {
		t.Fatalf("seq 非单调: %v, %v", m1["seq"], m2["seq"])
	}

	m3 := f.payload(event.Event{Kind: event.Phase, Text: "思考中"})
	if m3 == nil || m3["seq"].(int64) != 3 {
		t.Fatalf("首个 phase 应转发且 seq=3: %+v", m3)
	}
	if m3["kind"] != "phase" || m3["text"] != "思考中" {
		t.Fatalf("phase payload 错误: %+v", m3)
	}
	if m4 := f.payload(event.Event{Kind: event.Phase, Text: "思考中"}); m4 != nil {
		t.Fatalf("同文案 phase 在 200ms 内应被节流: %+v", m4)
	}
	// 节流丢弃不消费 seq：下一个转发事件的 seq 必须紧接 3（无断号）。
	m5 := f.payload(event.Event{Kind: event.Phase, Text: "正在检索记忆"})
	if m5 == nil || m5["seq"].(int64) != 4 {
		t.Fatalf("不同文案 phase 不受节流影响且 seq=4: %+v", m5)
	}
	if f.last() != 4 {
		t.Fatalf("last = %d, want 4", f.last())
	}
	// 窗口过后同文案恢复转发（直接回拨节流时间戳，避免真实等待）。
	f.mu.Lock()
	f.phaseLast["思考中"] = time.Now().Add(-gaeaPhaseThrottleWindow - time.Millisecond)
	f.mu.Unlock()
	m6 := f.payload(event.Event{Kind: event.Phase, Text: "思考中"})
	if m6 == nil || m6["seq"].(int64) != 5 {
		t.Fatalf("窗口过后 phase 应恢复转发: %+v", m6)
	}
	// 会话切换 reset：seq 归零（前端在本动作后重置 lastSeq）。
	f.reset()
	if f.last() != 0 {
		t.Fatalf("reset 后 last = %d, want 0", f.last())
	}
}

// TestGaeaEventMapTranslations v4.26 wire 转译：Retrying/compaction 统一为
// phase（前端 reducer 零改动），subagent_message 扁平化 text/ref/parentId。
func TestGaeaEventMapTranslations(t *testing.T) {
	r := gaeaEventMap(event.Event{Kind: event.Retrying, RetryAttempt: 1, RetryMax: 3})
	if r["kind"] != "phase" || r["text"] != "正在重试 (1/3)" {
		t.Fatalf("Retrying 转译错误: %+v", r)
	}
	cs := gaeaEventMap(event.Event{Kind: event.CompactionStarted, Compaction: event.Compaction{Trigger: "auto"}})
	if cs["kind"] != "phase" || cs["text"] != "正在压缩上下文…" {
		t.Fatalf("CompactionStarted 转译错误: %+v", cs)
	}
	cd := gaeaEventMap(event.Event{Kind: event.CompactionDone})
	if cd["kind"] != "phase" || cd["text"] != "压缩完成" {
		t.Fatalf("CompactionDone 转译错误: %+v", cd)
	}
	sm := gaeaEventMap(event.Event{Kind: event.SubagentMessage, Text: "答复", SubagentRef: "sa_1", ParentToolID: "call-1"})
	if sm["kind"] != "subagent_message" || sm["text"] != "答复" || sm["ref"] != "sa_1" || sm["parentId"] != "call-1" {
		t.Fatalf("SubagentMessage 转译错误: %+v", sm)
	}
	// ref/parentId 为空时缺省不下发（wire 线格式保持最小）。
	sm2 := gaeaEventMap(event.Event{Kind: event.SubagentMessage, Text: "临时答复"})
	if _, ok := sm2["ref"]; ok {
		t.Fatalf("空 ref 不应下发: %+v", sm2)
	}
	if _, ok := sm2["parentId"]; ok {
		t.Fatalf("空 parentId 不应下发: %+v", sm2)
	}
	// kind 名映射必须落 subagent_message（否则落盘/转译断链）。
	if got := gaeaKindName(event.SubagentMessage); got != "subagent_message" {
		t.Fatalf("gaeaKindName(SubagentMessage) = %q", got)
	}
}

// TestFoldResyncItemsBasic 最小折叠器：user/assistant/tool(dispatch+result 合并)
// /notice 四类，非对话气泡 kind（usage/phase/turn 边界/retrying/subagent_message）
// 一律跳过。
func TestFoldResyncItemsBasic(t *testing.T) {
	entries := []session.LogEntry{
		resyncEntry(1, "turn_started", map[string]any{}),
		resyncEntry(2, "user_message", map[string]any{"content": "帮我查文件"}),
		resyncEntry(3, "tool_dispatch", map[string]any{"id": "t1", "name": "read_file", "args": `{"path":"a.go"}`, "readOnly": true}),
		resyncEntry(4, "tool_result", map[string]any{"id": "t1", "output": "package a"}),
		resyncEntry(5, "assistant_message", map[string]any{"text": "查到了", "reasoning": "思考"}),
		resyncEntry(6, "notice", map[string]any{"level": "warn", "text": "上下文即将压缩"}),
		resyncEntry(7, "usage", map[string]any{"promptTokens": 10}),
		resyncEntry(8, "phase", map[string]any{"text": "思考中"}),
		resyncEntry(9, "retrying", map[string]any{"attempt": 1, "max": 3}),
		resyncEntry(10, "subagent_message", map[string]any{"text": "子代理答复", "ref": "sa_1"}),
		resyncEntry(11, "turn_done", map[string]any{}),
	}
	items := foldResyncItems(itemsNilGuard(entries))
	if len(items) != 4 {
		t.Fatalf("items = %d, want 4（usage/phase/retrying/subagent_message/turn 边界跳过）: %+v", len(items), items)
	}
	if items[0].Kind != "user" || items[0].Text != "帮我查文件" || items[0].ID != "u2" {
		t.Fatalf("user 条目错误: %+v", items[0])
	}
	if items[1].Kind != "tool" || items[1].ToolID != "t1" || items[1].Status != "done" ||
		items[1].Output != "package a" || items[1].Name != "read_file" {
		t.Fatalf("tool 合并条目错误: %+v", items[1])
	}
	if items[2].Kind != "assistant" || items[2].Text != "查到了" || items[2].Reasoning != "思考" {
		t.Fatalf("assistant 条目错误: %+v", items[2])
	}
	if items[3].Kind != "notice" || items[3].Level != "warn" || items[3].Text != "上下文即将压缩" {
		t.Fatalf("notice 条目错误: %+v", items[3])
	}
}

// itemsNilGuard 原样返回（占位保持表意：输入来自日志读取端，可能为 nil）。
func itemsNilGuard(e []session.LogEntry) []session.LogEntry { return e }

// TestFoldResyncItemsGapSeq 断号场景：torn-tail 修复/外部删行导致日志 seq
// 不连续时，折叠仍按事件序产出、ID 用条目自带 seq，不炸不重排。
func TestFoldResyncItemsGapSeq(t *testing.T) {
	entries := []session.LogEntry{
		resyncEntry(1, "user_message", map[string]any{"content": "第一问"}),
		resyncEntry(2, "assistant_message", map[string]any{"text": "第一答"}),
		resyncEntry(7, "user_message", map[string]any{"content": "第二问"}),
		resyncEntry(9, "assistant_message", map[string]any{"text": "第二答"}),
	}
	items := foldResyncItems(entries)
	if len(items) != 4 {
		t.Fatalf("items = %d, want 4", len(items))
	}
	wantIDs := []string{"u1", "a2", "u7", "a9"}
	for i, id := range wantIDs {
		if items[i].ID != id {
			t.Fatalf("items[%d].ID = %q, want %q", i, items[i].ID, id)
		}
	}
}

// TestFoldResyncItemsDeltaMerge 流式 delta 合并：text/reasoning 逐步累积进同
// 一 assistant 条目（与实时渲染单气泡/投影回放同形态）；assistant_message 到达
// 时原地替换为全文（超集语义）；user/notice 边界关闭累积。
func TestFoldResyncItemsDeltaMerge(t *testing.T) {
	entries := []session.LogEntry{
		resyncEntry(1, "user_message", map[string]any{"content": "问"}),
		resyncEntry(2, "reasoning", map[string]any{"reasoning": "先想"}),
		resyncEntry(3, "reasoning", map[string]any{"reasoning": "一半"}),
		resyncEntry(4, "text", map[string]any{"text": "答："}),
		resyncEntry(5, "text", map[string]any{"text": "42"}),
		resyncEntry(6, "assistant_message", map[string]any{"text": "答：42，完整", "reasoning": "先想一半"}),
		resyncEntry(7, "notice", map[string]any{"level": "info", "text": "提示"}),
		resyncEntry(8, "text", map[string]any{"text": "新气泡"}),
	}
	items := foldResyncItems(entries)
	if len(items) != 4 {
		t.Fatalf("items = %d, want 4: %+v", len(items), items)
	}
	if items[1].Text != "答：42，完整" || items[1].Reasoning != "先想一半" {
		t.Fatalf("assistant 合并/替换错误: %+v", items[1])
	}
	if items[3].Text != "新气泡" {
		t.Fatalf("notice 后应新开 assistant 条目: %+v", items[3])
	}
	if items[1].ID != "a2" || items[3].ID != "a8" {
		t.Fatalf("ID 应取首个 delta/新条目的 seq: %+v %+v", items[1], items[3])
	}
}

// TestFoldResyncItemsHangingResult 悬挂 tool_result（无前置 dispatch，日志被
// 裁剪/仅存结果）：自建完成态条目，不丢事件；error 结果落 status=error。
func TestFoldResyncItemsHangingResult(t *testing.T) {
	entries := []session.LogEntry{
		resyncEntry(1, "tool_result", map[string]any{"id": "tx", "name": "bash", "err": "boom"}),
		resyncEntry(2, "tool_result", map[string]any{"id": "ty", "name": "ls", "output": "a.go"}),
	}
	items := foldResyncItems(entries)
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Status != "error" || items[0].Err != "boom" || items[0].ToolID != "tx" {
		t.Fatalf("悬挂 error result 条目错误: %+v", items[0])
	}
	if items[1].Status != "done" || items[1].Output != "a.go" {
		t.Fatalf("悬挂 ok result 条目错误: %+v", items[1])
	}
}

// TestFoldResyncItemsEmptyAndUnknown 容错：空日志 → 非 nil 空切片（前端按
// 数组消费不崩）；未知 kind / 坏 payload → 跳过不 panic。
func TestFoldResyncItemsEmptyAndUnknown(t *testing.T) {
	if items := foldResyncItems(nil); items == nil || len(items) != 0 {
		t.Fatalf("空日志应返回非 nil 空切片: %#v", items)
	}
	entries := []session.LogEntry{
		resyncEntry(1, "future_kind", map[string]any{"whatever": true}),
		{Seq: 2, Ts: 1700000000, Kind: "user_message", Payload: json.RawMessage(`not-json`)},
		resyncEntry(3, "tool_dispatch", map[string]any{"id": "", "name": "x"}), // 无 ID 跳过
		resyncEntry(4, "user_message", map[string]any{"content": "好的"}),
	}
	items := foldResyncItems(entries)
	if len(items) != 1 || items[0].Text != "好的" {
		t.Fatalf("未知 kind/坏 payload/空 ID 应被跳过: %+v", items)
	}
}

// TestGaeaResyncEventsBinding 绑定面：引擎未初始化早退（空 items 非 nil，
// seq 取转发层当前值）；真实 controller + 磁盘日志 → 折叠 items 返回。
func TestGaeaResyncEventsBinding(t *testing.T) {
	// 早退路径：无 controller（保存/恢复包级状态，避免污染其他测试）。
	oldCtrl := ga.ctrl
	oldCfg := ga.cfg
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.ctrl = oldCtrl
		ga.cfg = oldCfg
		ga.mu.Unlock()
	})
	ga.mu.Lock()
	ga.ctrl = nil
	ga.mu.Unlock()

	a := &App{}
	res := a.GaeaResyncEvents(0)
	if res.Items == nil || len(res.Items) != 0 {
		t.Fatalf("无 controller 应返回非 nil 空 items: %#v", res.Items)
	}
	if res.Seq != ga.wire.last() {
		t.Fatalf("res.Seq = %d, want 转发层当前 seq %d", res.Seq, ga.wire.last())
	}

	// 完整路径：boot 构建真实 controller（mock provider，无网络），写事件日志，
	// 验证 GaeaResyncEvents 从磁盘折叠出对话视图。chdir 临时目录避免污染仓库
	// （与 TestGaeaHistoryGolden 同款装配，不改 APPDATA——boot.Build 打开全局
	// Hephaestus.db，临时 APPDATA 会被句柄锁住）。
	oldwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldwd) }()
	_ = os.Chdir(t.TempDir())

	const kind = "test-mock-gaea-resync"
	provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock"), nil
	})
	cfg := gaeaConfig.Default()
	cfg.DefaultModel = "mock"
	cfg.Providers = []gaeaConfig.ProviderEntry{{
		Name: "mock", Kind: kind, Model: "grok-3", ContextWindow: 1_000_000,
	}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) { return cfg, nil })
	defer gaeaConfig.SetLoader(nil)

	ctrl, err := gaeaBoot.Build(context.Background(), gaeaBoot.Options{
		Model: "mock", RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		SessionDir: gaeaConfig.WorkspaceSessionDir("", ""),
	})
	if err != nil {
		t.Fatalf("boot.Build: %v", err)
	}
	defer ctrl.Close()

	path := filepath.Join(gaeaConfig.WorkspaceSessionDir("", ""), "resync-session.jsonl")
	ctrl.Resume(agent.NewSession("sys"), path)

	// 写入当前会话的事件日志：一轮 user → tool(dispatch+result) → notice。
	lp := session.LogPathFor(path)
	w, err := session.OpenLog(lp, "", "")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	defer w.Close()
	appendMust := func(kind string, payload map[string]any) {
		t.Helper()
		if _, err := w.Append(kind, payload); err != nil {
			t.Fatalf("append %s: %v", kind, err)
		}
	}
	appendMust("user_message", map[string]any{"content": "帮我查文件"})
	appendMust("tool_dispatch", map[string]any{"id": "t1", "name": "read_file", "args": `{"path":"a.go"}`})
	appendMust("tool_result", map[string]any{"id": "t1", "output": "package a"})
	appendMust("notice", map[string]any{"level": "info", "text": "检索完成"})
	_ = w.Close()

	ga.mu.Lock()
	ga.ctrl = ctrl
	ga.mu.Unlock()

	res2 := a.GaeaResyncEvents(3)
	if len(res2.Items) != 3 {
		t.Fatalf("items = %d, want 3（dispatch+result 合并为一）: %+v", len(res2.Items), res2.Items)
	}
	if res2.Items[0].Kind != "user" || res2.Items[0].Text != "帮我查文件" {
		t.Fatalf("items[0] 错误: %+v", res2.Items[0])
	}
	if res2.Items[1].Kind != "tool" || res2.Items[1].Status != "done" || res2.Items[1].Output != "package a" {
		t.Fatalf("items[1] 错误（dispatch+result 应合并为完成态）: %+v", res2.Items[1])
	}
	if res2.Items[2].Kind != "notice" {
		t.Fatalf("items[2] 错误: %+v", res2.Items[2])
	}
	if res2.Seq != ga.wire.last() {
		t.Fatalf("res2.Seq = %d, want %d", res2.Seq, ga.wire.last())
	}
}
