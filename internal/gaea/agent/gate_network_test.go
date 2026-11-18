package agent

// P8（codex 蒸馏 B6 折算，第四批）验收：
// ① 断网闸拒绝网络能力工具、放行本地工具、allow=true 零介入；
// ② panic 隔离（已有机制验收落袋）：注入 panic 工具后宿主存活、
//    回合失败可解释（errMsg 带 panic）、同批其余工具照常完成。

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

type allowAllGate struct{}

func (allowAllGate) Check(context.Context, string, json.RawMessage, bool) (bool, string, error) {
	return true, "", nil
}

func TestNetworkGateBlocksNetworkToolsOnly(t *testing.T) {
	inner := allowAllGate{}
	g := WrapNetworkGate(inner, false)

	for _, name := range []string{"web_search", "web_fetch", "browser_navigate", "browser_run_js", "mcp__github__create_issue"} {
		allow, reason, err := g.Check(context.Background(), name, nil, false)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if allow {
			t.Errorf("%s: 断网闸放行了网络工具", name)
		}
		if !strings.Contains(reason, "network") {
			t.Errorf("%s: 拒绝理由不可操作: %q", name, reason)
		}
	}
	// 本地工具（含 bash——它有自己的子进程网络开关）照常放行。
	for _, name := range []string{"bash", "read_file", "edit_file", "todo_write"} {
		allow, _, err := g.Check(context.Background(), name, nil, false)
		if err != nil || !allow {
			t.Errorf("%s: 本地工具被误拒 allow=%v err=%v", name, allow, err)
		}
	}
	// 开关闭合时零介入：原闸门原样返回（现状红线）。
	if got := WrapNetworkGate(inner, true); got != any(inner) {
		t.Errorf("allow=true 必须原样返回内层闸门（%T）", got)
	}
}

// fakeAmender 是修正案的可编排桩：Allowed 走静态清单，Ask 返回编排值。
type fakeAmender struct {
	entries   []string
	enabled   bool
	askAllow  bool
	askPersist bool
	asks      int
	persisted []string
}

func (f *fakeAmender) Allowed(toolName, target string) bool {
	for _, e := range f.entries {
		if e == toolName || (target != "" && (target == e || strings.HasSuffix(target, "."+e))) {
			return true
		}
	}
	return false
}

func (f *fakeAmender) Enabled() bool { return f.enabled }

func (f *fakeAmender) Ask(context.Context, string, string) (bool, bool) {
	f.asks++
	return f.askAllow, f.askPersist
}

func (f *fakeAmender) Persist(entry string) error {
	f.persisted = append(f.persisted, entry)
	return nil
}

// 刀1（差距档案）验收：①开关关/Amendment nil = 现状拒绝逐字节等价；
// ②台账命中（精确/子域/工具名）不弹卡放行且仍走内层闸；③Enabled+Ask
// allow → 放行 + persist 传呼；Ask deny → 拒绝可解释；④target 提取
// url→host，无 url 退化工具名。
func TestNetworkGateAmendmentLedger(t *testing.T) {
	args := json.RawMessage(`{"url":"https://docs.example.com/x?q=1"}`)
	am := &fakeAmender{entries: []string{"example.com", "web_search"}}
	g := WrapNetworkGateWith(allowAllGate{}, false, am).(*NetworkGate)

	// 台账命中：host 子域匹配 + 工具名精确，不弹卡。
	if am.asks != 0 {
		t.Fatalf("前置弹卡次数应为 0")
	}
	if ok, _, err := g.Check(context.Background(), "web_fetch", args, false); err != nil || !ok {
		t.Fatalf("台账 host 命中应放行: ok=%v err=%v", ok, err)
	}
	if ok, _, err := g.Check(context.Background(), "web_search", nil, false); err != nil || !ok {
		t.Fatalf("台账工具名命中应放行: ok=%v err=%v", ok, err)
	}
	if am.asks != 0 {
		t.Fatalf("台账命中不应弹卡, asks=%d", am.asks)
	}
	// 未命中 + 开关关：现状拒绝（不弹卡）。
	if ok, reason, _ := g.Check(context.Background(), "browser_navigate", nil, false); ok || !strings.Contains(reason, "sandbox network is disabled") {
		t.Fatalf("未命中且开关关应按现状拒绝: ok=%v reason=%q", ok, reason)
	}
	if am.asks != 0 {
		t.Fatalf("开关关不应弹卡, asks=%d", am.asks)
	}
}

func TestNetworkGateAmendmentAskFlow(t *testing.T) {
	args := json.RawMessage(`{"url":"https://api.example.com/v1"}`)
	am := &fakeAmender{enabled: true, askAllow: true, askPersist: true}
	g := WrapNetworkGateWith(allowAllGate{}, false, am).(*NetworkGate)

	ok, _, err := g.Check(context.Background(), "web_fetch", args, false)
	if err != nil || !ok {
		t.Fatalf("弹卡允许应放行: ok=%v err=%v", ok, err)
	}
	if am.asks != 1 || len(am.persisted) != 1 || am.persisted[0] != "api.example.com" {
		t.Fatalf("persist 传呼不符: asks=%d persisted=%v", am.asks, am.persisted)
	}

	// 拒绝分支：文案可解释（带 declined）。
	am2 := &fakeAmender{enabled: true, askAllow: false}
	g2 := WrapNetworkGateWith(allowAllGate{}, false, am2).(*NetworkGate)
	ok, reason, _ := g2.Check(context.Background(), "web_fetch", args, false)
	if ok || !strings.Contains(reason, "declined") {
		t.Fatalf("弹卡拒绝应给出可解释文案: ok=%v reason=%q", ok, reason)
	}
}

func TestNetworkGateAmendmentTargetExtraction(t *testing.T) {
	if got := amendmentTarget("web_fetch", json.RawMessage(`{"url":"https://Docs.Example.COM:443/a"}`)); got != "Docs.Example.COM" {
		t.Fatalf("host 提取不符: %q", got)
	}
	if got := amendmentTarget("mcp__x__noargs", nil); got != "mcp__x__noargs" {
		t.Fatalf("无 url 应退化工具名: %q", got)
	}
	if got := amendmentTarget("web_fetch", json.RawMessage(`{"path":"/tmp/a"}`)); got != "web_fetch" {
		t.Fatalf("非 url subject 应退化工具名: %q", got)
	}
}

// headless 只读台账：清单命中放行、Enabled 恒 false（永不弹卡）。
func TestNetworkLedgerOnly(t *testing.T) {
	am := NetworkLedgerOnly([]string{"example.com"})
	g := WrapNetworkGateWith(allowAllGate{}, false, am).(*NetworkGate)
	args := json.RawMessage(`{"url":"https://x.example.com/"}`)
	if ok, _, _ := g.Check(context.Background(), "web_fetch", args, false); !ok {
		t.Fatal("headless 台账命中应放行")
	}
	if ok, _, _ := g.Check(context.Background(), "browser_navigate", nil, false); ok {
		t.Fatal("headless 未命中应拒绝（无卡可弹）")
	}
	if am.Enabled() {
		t.Fatal("NetworkLedgerOnly 必须 Enabled=false")
	}
}

// panic 工具：执行即 panic（P8 隔离面的注入源）。
type panicTool struct{ name string }

func (p panicTool) Name() string            { return p.name }
func (p panicTool) Description() string     { return "" }
func (p panicTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (p panicTool) ReadOnly() bool          { return false }
func (p panicTool) Execute(context.Context, json.RawMessage) (string, error) {
	panic("boom: injected tool panic")
}

func TestPanicToolIsolatedBatchSurvives(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(panicTool{name: "crash_tool"})
	var healthyCalls int32
	reg.Add(fakeTool{name: "healthy", readOnly: true, calls: &healthyCalls})
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{
			toolCallChunk("c1", "crash_tool", `{}`),
			toolCallChunk("c2", "healthy", `{}`),
			{Type: provider.ChunkDone},
		},
		{{Type: provider.ChunkText, Text: "recovered"}, {Type: provider.ChunkDone}},
	}}
	sink := &collectingSink{}
	a := New(prov, reg, NewSession(""), Options{DisableVerify: true}, sink)
	res, err := a.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run: %v（panic 工具不得拖死回合）", err)
	}
	if res == nil || res.Summary != "recovered" {
		t.Fatalf("回合未恢复完成: %+v", res)
	}
	_, results := stampedToolEvents(t, sink.all())
	var crashErrMsg string
	for _, e := range results {
		if e.Tool.Name == "crash_tool" {
			crashErrMsg = e.Tool.Err
		}
	}
	if !strings.Contains(crashErrMsg, "panic") {
		t.Errorf("panic 工具的错误不可解释: %q", crashErrMsg)
	}
	if atomic.LoadInt32(&healthyCalls) == 0 {
		t.Error("同批健康工具未执行（隔离应只吞 panic，不吞批次）")
	}
}
