package app

// wx_agent_test.go — 微信智能体 v1 测试（2026-09）。
//
// 覆盖三块（任务1/2/3 门禁口径）：
//  1. 工具循环：httptest 模拟 OpenAI 兼容 SSE——第一轮回 tool_calls、第二轮回
//     文本 → 断言循环两轮、cards 收集、tool 结果喂回模型、最终回复；参数解析
//     失败/字段缺失 → 错误文本让模型同轮自纠；轮数上限。
//  2. 能力门 wxAgentAvailable：Caps 含/不含 "tools"、离线模式开关。
//  3. exec 适配：create_reminder 的 intent 构造（parseReminderWhen /
//     stripReminderText 可解析）、send_latest_file 的 modify_and_send 诚实
//     护栏保留、navigate_board 板块名归一。
//
// 不做端到端 WeChat 回调整条测（clawbot 通道测试已有自身基建；回调侧编排由
// whisper_state.go 薄封装 + 既有 SendFileCard 链承担）。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/channels/weixin"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/intent"
	"github.com/gaea/gaea/internal/modelengine"
	wdb "github.com/gaea/gaea/internal/whisper/db"
)

// sseToolCall 一轮 tool_calls SSE 响应（delta 拼装 + finish_reason=tool_calls）。
func sseToolCall(id, name, argsJSON string) string {
	var b strings.Builder
	b.WriteString(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"` + id + `","type":"function","function":{"name":"` + name + `","arguments":`)
	args, _ := json.Marshal(argsJSON)
	b.Write(args)
	b.WriteString(`}}]}}]}` + "\n\n")
	b.WriteString(`data: {"choices":[{"index":0,"delta":{"finish_reason":"tool_calls"}}]}` + "\n\n")
	return b.String()
}

// sseText 一轮纯文本 SSE 响应。
func sseText(text string) string {
	msg, _ := json.Marshal(text)
	return "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":" + string(msg) + "}}]}\n\n" + "data: [DONE]\n\n"
}

// wxAgentReqLog 记录 mock 收到的每次请求（断言用）。
type wxAgentReqLog struct {
	mu   sync.Mutex
	reqs []ai.ChatRequest
}

func (l *wxAgentReqLog) add(r ai.ChatRequest) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reqs = append(l.reqs, r)
}

func (l *wxAgentReqLog) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.reqs)
}

func (l *wxAgentReqLog) last() ai.ChatRequest {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reqs[len(l.reqs)-1]
}

// wxAgentPhase 分阶段响应回调：served 是含本次在内之前已服务的请求数
// （0 = 第一轮 LLM 请求）。
type wxAgentPhase func(served int, w http.ResponseWriter, r *http.Request)

// newWxAgentTestApp 构造 agent 测试 App：herdsman 引擎指向 mock SSE 服务器，
// chat 功能绑定 herdsman/test-model（目录 Caps 含 tools）。
func newWxAgentTestApp(t *testing.T, phase wxAgentPhase) (*App, *wxAgentReqLog) {
	t.Helper()
	log := &wxAgentReqLog{}
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ai.ChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		log.add(req)
		phase(log.count()-1, w, r)
	})
	srv := httptest.NewServer(wrapped)
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	root := filepath.Join(t.TempDir(), "data")

	c := &core{
		cfg:       &config.Config{Model: "test-model", XaiAPIBaseURL: "http://127.0.0.1:1"},
		engineMgr: modelengine.NewManager("", ""),
		chatStore: chat.NewStore(filepath.Join(root, "chat")),
	}
	t.Cleanup(func() { _ = c.chatStore.Close() })
	t.Cleanup(func() { _ = wdb.CloseDatabase(root) })
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Enabled: true, BaseURL: srv.URL,
		Models: []modelengine.ModelInfo{{ID: "test-model", Caps: []string{"tools", "reasoning"}}},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	if err := c.SetFeatureModel("chat", "herdsman", "test-model"); err != nil {
		t.Fatalf("SetFeatureModel: %v", err)
	}

	a := &App{core: c}
	a.ctx = context.Background()
	a.client = ai.NewClient(a.cfg)
	a.client.SetEngineManager(a.engineMgr)
	a.whisperState = &whisperState{
		core:            c,
		app:             a,
		whisperDataRoot: root,
		weixinServers:   map[string]*weixin.Server{},
	}
	a.assistantMgr = assistant.NewEmpty(root)
	return a, log
}

// ─── 1. 工具循环 ─────────────────────────────────────────────

// 第一轮回 tool_calls(generate_image)、第二轮回文本：循环两轮、cards 收集、
// tool 结果喂回模型、最终回复为第二轮文本。
func TestWxAgentLoopToolCallThenText(t *testing.T) {
	a, log := newWxAgentTestApp(t, func(n int, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if n == 0 {
			_, _ = w.Write([]byte(sseToolCall("call_1", "generate_image", `{"prompt":"一只猫"}`)))
		} else {
			_, _ = w.Write([]byte(sseText("画好了，见图。")))
		}
	})

	// seam 替换 exec（媒体域装配不在本测范围）：返回产物路径
	orig := wxAgentExecToolFn
	wxAgentExecToolFn = func(a *App, assistantID string, call ai.ChatToolCall, userMsg string) (string, string) {
		if call.Function.Name != "generate_image" {
			t.Errorf("工具名 = %q, want generate_image", call.Function.Name)
		}
		if args, err := wxAgentToolArgs(call); err != nil || args["prompt"] != "一只猫" {
			t.Errorf("参数解析 = %v, %v", args, err)
		}
		return "好，画好了。\n已生成产物：C:/tmp/cat.png", "C:/tmp/cat.png"
	}
	defer func() { wxAgentExecToolFn = orig }()

	reply, cards, err := runWxAgent(a, "ast1", "你是测试人格。", "画一只猫")
	if err != nil {
		t.Fatalf("runWxAgent: %v", err)
	}
	if n := log.count(); n != 2 {
		t.Fatalf("LLM 轮数 = %d, want 2", n)
	}
	if reply != "画好了，见图。" {
		t.Errorf("reply = %q, want 最终轮文本", reply)
	}
	if len(cards) != 1 || cards[0] != "C:/tmp/cat.png" {
		t.Errorf("cards = %v, want [C:/tmp/cat.png]", cards)
	}

	// 第一轮请求：tools 透传 + messages=[system,user]
	first := log.reqs[0]
	if len(first.Tools) != len(wxAgentTools) {
		t.Errorf("tools 数 = %d, want %d", len(first.Tools), len(wxAgentTools))
	}
	if len(first.Messages) != 2 || first.Messages[0].Role != "system" || first.Messages[1].Content != "画一只猫" {
		t.Fatalf("首轮 messages = %+v", first.Messages)
	}
	if !strings.Contains(first.Messages[0].Content, "微信智能体工作方式") {
		t.Errorf("system 提示缺少智能体指令段")
	}

	// 第二轮请求：assistant(tool_calls) + tool 结果已喂回
	second := log.last()
	if len(second.Messages) != 4 {
		t.Fatalf("第二轮 messages 数 = %d, want 4（system/user/assistant/tool）", len(second.Messages))
	}
	asst, tool := second.Messages[2], second.Messages[3]
	if asst.Role != "assistant" || len(asst.ToolCalls) != 1 || asst.ToolCalls[0].ID != "call_1" {
		t.Errorf("assistant 消息 = %+v", asst)
	}
	if tool.Role != "tool" || tool.ToolCallID != "call_1" || !strings.Contains(tool.Content, "已生成产物：C:/tmp/cat.png") {
		t.Errorf("tool 消息 = %+v", tool)
	}
}

// 工具参数不是合法 JSON：tool 结果回错误文本让模型同轮自纠（第二轮正常文本），
// 不 panic、不外抛。
func TestWxAgentToolArgsSelfCorrect(t *testing.T) {
	a, log := newWxAgentTestApp(t, func(n int, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if n == 0 {
			_, _ = w.Write([]byte(sseToolCall("call_bad", "create_reminder", `{"task_text": 不是JSON`)))
		} else {
			_, _ = w.Write([]byte(sseText("抱歉，时间没说清楚，请再说一次。")))
		}
	})

	reply, cards, err := runWxAgent(a, "ast1", "你是测试人格。", "提醒我喝水")
	if err != nil {
		t.Fatalf("runWxAgent: %v", err)
	}
	if log.count() != 2 {
		t.Fatalf("LLM 轮数 = %d, want 2（自纠一轮）", log.count())
	}
	if reply != "抱歉，时间没说清楚，请再说一次。" {
		t.Errorf("reply = %q", reply)
	}
	if len(cards) != 0 {
		t.Errorf("cards = %v, want 空", cards)
	}
	if got := log.last().Messages[3].Content; !strings.Contains(got, "参数解析失败") {
		t.Errorf("tool 结果 = %q, want 含参数错误提示（模型自纠依据）", got)
	}
}

// 字段缺失：合法 JSON 但 when_raw 缺失 → 真实 exec 层诚实回「没听懂时间」，
// 同轮喂回模型自纠。
func TestWxAgentMissingFieldHonestReply(t *testing.T) {
	a, log := newWxAgentTestApp(t, func(n int, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if n == 0 {
			_, _ = w.Write([]byte(sseToolCall("call_m", "create_reminder", `{"task_text":"喝水"}`)))
		} else {
			_, _ = w.Write([]byte(sseText("好的，请告诉我具体时间。")))
		}
	})

	if _, _, err := runWxAgent(a, "ast1", "你是测试人格。", "提醒我喝水"); err != nil {
		t.Fatalf("runWxAgent: %v", err)
	}
	if got := log.last().Messages[3].Content; !strings.Contains(got, "没听懂时间") {
		t.Errorf("tool 结果 = %q, want exec 层诚实回复（含「没听懂时间」）", got)
	}
}

// 轮数上限：每轮都回 tool_calls → wxAgentMaxRounds 轮后把积累文本 +
// 「（任务未完成，请重试）」返回，不外抛 error。
func TestWxAgentRoundLimit(t *testing.T) {
	a, log := newWxAgentTestApp(t, func(_ int, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseToolCall("call_loop", "query_status", `{}`)))
	})

	orig := wxAgentExecToolFn
	wxAgentExecToolFn = func(a *App, assistantID string, call ai.ChatToolCall, userMsg string) (string, string) {
		return "当前状态正常。", ""
	}
	defer func() { wxAgentExecToolFn = orig }()

	reply, _, err := runWxAgent(a, "ast1", "你是测试人格。", "现在什么状态")
	if err != nil {
		t.Fatalf("runWxAgent: %v", err)
	}
	if n := log.count(); n != wxAgentMaxRounds {
		t.Fatalf("LLM 轮数 = %d, want %d", n, wxAgentMaxRounds)
	}
	if !strings.HasSuffix(reply, "（任务未完成，请重试）") {
		t.Errorf("reply = %q, want 未完成标记结尾", reply)
	}
}

// 未知工具名：错误文本喂回模型（不 panic、不中断循环）。
func TestWxAgentUnknownTool(t *testing.T) {
	a, _ := newWxAgentTestApp(t, func(_ int, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseToolCall("call_u", "no_such_tool", `{}`)))
	})
	got, _ := a.wxAgentExecTool("ast1", ai.ChatToolCall{
		ID:       "call_u",
		Function: ai.ChatToolFunction{Name: "no_such_tool", Arguments: `{}`},
	}, "随便")
	if !strings.Contains(got, "未知工具") {
		t.Errorf("结果 = %q, want 未知工具提示", got)
	}
}

// ─── 2. 能力门 ───────────────────────────────────────────────

func TestWxAgentAvailable(t *testing.T) {
	stub := func(_ int, w http.ResponseWriter, r *http.Request) {}
	a, _ := newWxAgentTestApp(t, stub)

	if !wxAgentAvailable(a) {
		t.Fatal("Caps 含 tools 且非离线 → 应可用")
	}

	// 去掉能力位 → 不可用（宁缺勿滥）
	e, _ := a.engineMgr.GetEngine("herdsman")
	e.Models = []modelengine.ModelInfo{{ID: "test-model"}}
	if err := a.engineMgr.SaveEngine(*e); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	if wxAgentAvailable(a) {
		t.Fatal("Caps 不含 tools → 应不可用")
	}

	// 模型不在目录中（查不到=不含）→ 不可用：绑定引擎里另一个无 Caps 的模型
	// （SetFeatureModel 校验模型必须在引擎列表内，故用无能力位条目覆盖同一路径：
	// wxModelCaps 查无/无标记都返回 nil）
	e, _ = a.engineMgr.GetEngine("herdsman")
	e.Models = []modelengine.ModelInfo{
		{ID: "test-model", Caps: []string{"tools"}},
		{ID: "bare-model"},
	}
	if err := a.engineMgr.SaveEngine(*e); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	if err := a.SetFeatureModel("chat", "herdsman", "bare-model"); err != nil {
		t.Fatalf("SetFeatureModel: %v", err)
	}
	if wxAgentAvailable(a) {
		t.Fatal("模型无 tools 能力位 → 应不可用")
	}
	if err := a.SetFeatureModel("chat", "herdsman", "test-model"); err != nil {
		t.Fatalf("SetFeatureModel: %v", err)
	}

	// 离线模式 → 一票否决
	a.cfg.OfflineMode = true
	if wxAgentAvailable(a) {
		t.Fatal("离线模式 → 应不可用")
	}
	a.cfg.OfflineMode = false
	if !wxAgentAvailable(a) {
		t.Fatal("离线关闭后应恢复可用")
	}

	// 防御：nil / 空壳 App 不 panic
	if wxAgentAvailable(nil) {
		t.Fatal("nil App 应不可用")
	}
	if wxAgentAvailable(&App{}) {
		t.Fatal("未装配 App 应不可用")
	}
}

// ─── 3. exec 适配（intent 构造）─────────────────────────────

// create_reminder 的 intent 构造：when_raw 拼句首，execReminder 的
// parseReminderWhen 能解析、stripReminderText 剥出事项正文——零改动复用的前提。
func TestWxAgentReminderIntentConstruction(t *testing.T) {
	cases := []struct{ task, when, wantItem string }{
		{"喝水", "30分钟后", "喝水"},
		{"收作业", "明晚8点", "收作业"},
	}
	for _, c := range cases {
		it := wxAgentIntentReminder(c.task, c.when, "提醒我"+c.when+c.task)
		if it.Action != intent.ActionReminder {
			t.Fatalf("Action = %q", it.Action)
		}
		fire, stale, ok := parseReminderWhen(it.Text, time.Now())
		if !ok || stale {
			t.Fatalf("parseReminderWhen(%q) = ok=%v stale=%v, want 可解析", it.Text, ok, stale)
		}
		if fire.Before(time.Now()) {
			t.Errorf("触发时间 %v 已过", fire)
		}
		if got := stripReminderText(it.Text); got != c.wantItem {
			t.Errorf("stripReminderText(%q) = %q, want %q", it.Text, got, c.wantItem)
		}
	}
}

// send_latest_file：原文命中「整理后发给我」复合请求时保留 Target=modify_and_send
// （v4.41.2 诚实护栏）；普通消息 Target 为空（正常发送语义）。
func TestWxAgentSendLatestFileIntent(t *testing.T) {
	it := wxAgentIntentSendLatestFile("把这份报告重新整理后发给我")
	if it.Action != intent.ActionSendLatestFile || it.Target != "modify_and_send" {
		t.Fatalf("复合请求 intent = %+v, want modify_and_send（诚实护栏保留）", it)
	}
	it2 := wxAgentIntentSendLatestFile("今天天气不错")
	if it2.Target != "" {
		t.Fatalf("无关消息 Target = %q, want 空", it2.Target)
	}
}

// navigate_board 板块名归一：manifest id 直用；中文板块名借 intent.Parse
// 别名表解析成 id（execNavigate 按 boardLabel(id) 校验，直接传中文必落空）。
func TestWxAgentResolveBoard(t *testing.T) {
	a, _ := newWxAgentTestApp(t, func(_ int, w http.ResponseWriter, r *http.Request) {})
	if got := a.wxAgentResolveBoard("imagegen"); got != "imagegen" {
		t.Errorf("id 直用 = %q", got)
	}
	if got := a.wxAgentResolveBoard("绘梦"); got != "imagegen" {
		t.Errorf("中文别名 = %q, want imagegen", got)
	}
	if got := a.wxAgentResolveBoard("不存在的板块"); got != "不存在的板块" {
		t.Errorf("未知名原样返回（执行层诚实报错）= %q", got)
	}
}
