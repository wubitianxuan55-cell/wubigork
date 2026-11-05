package app

// chat_search_adapter_test.go — 聊天检索适配器测试（众神殿刀1 验收）：
// 清洗边界（信息词不剥）、三层契约注入头三分叉、状态归集、四入口换绑。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// stubChatWebSearch 替换聊天检索 seam（状态契约签名）并在测试结束恢复。
func stubChatWebSearch(t *testing.T, fn func(string) (chatSearchOutcome, error)) {
	t.Helper()
	orig := chatWebSearch
	chatWebSearch = fn
	t.Cleanup(func() { chatWebSearch = orig })
}

// stubChatSearchRun 替换统一检索核 seam 并在测试结束恢复。
func stubChatSearchRun(t *testing.T, fn func(ctx context.Context, query string, topK int) ([]builtin.SearchResult, []builtin.SearchAttempt, error)) {
	t.Helper()
	orig := chatSearchRun
	chatSearchRun = fn
	t.Cleanup(func() { chatSearchRun = orig })
}

// TestCleanChatSearchQuery 清洗下沉用例（原 whisper cleanSearchQuery 回归）+
// 刀1 修正：「在哪里/在哪儿/多少钱」是信息词，不再剥尾。
func TestCleanChatSearchQuery(t *testing.T) {
	cases := []struct{ in, want string }{
		{"最近几天天气怎样", "最近几天天气"},
		{"成都今天天气怎么样", "成都今天天气"},
		{"帮我搜索一下最近的新闻", "最近的新闻"},
		{"查一下今天上海天气", "今天上海天气"},
		{"什么是量子纠缠", "量子纠缠"},
		{"如何申请护照", "申请护照"},
		{"今天股价多少", "今天股价"}, // 「多少」疑问词照剥（非「多少钱」）
		{"请帮我查一下杭州到上海的高铁", "杭州到上海的高铁"},
		{"", ""},
		{"仅仅一个词", "仅仅一个词"},
		// 刀1 验收：信息词保留
		{"帮我搜索一下明天上海天气", "明天上海天气"},
		{"成都到上海在哪里买票", "成都到上海在哪里买票"},
		{"苹果价格多少钱", "苹果价格多少钱"},
		{"这个多少钱", "这个多少钱"},
	}
	for _, c := range cases {
		if got := cleanChatSearchQuery(c.in); got != c.want {
			t.Errorf("cleanChatSearchQuery(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestRunChatWebSearch_QueryCleaned 统一核收到的是清洗后的关键词（验收句：
// 「帮我搜索一下明天上海天气」→「明天上海天气」）。
func TestRunChatWebSearch_QueryCleaned(t *testing.T) {
	stubChatSearchRun(t, func(_ context.Context, query string, topK int) ([]builtin.SearchResult, []builtin.SearchAttempt, error) {
		if query != "明天上海天气" {
			t.Errorf("统一核应收清洗后查询, got %q", query)
		}
		if topK != chatSearchTopK {
			t.Errorf("topK = %d, want %d", topK, chatSearchTopK)
		}
		return []builtin.SearchResult{{Title: "上海天气", URL: "https://wttr.in/shanghai", Snippet: "晴", Source: "bing"}},
			[]builtin.SearchAttempt{{Name: "bing", Outcome: "ok", ElapsedMS: 320}}, nil
	})
	out, err := runChatWebSearch("帮我搜索一下明天上海天气")
	if err != nil {
		t.Fatalf("runChatWebSearch: %v", err)
	}
	if out.State != builtin.SearchStateOK {
		t.Errorf("State = %q, want ok", out.State)
	}
	if !strings.Contains(out.Text, "上海天气") || !strings.Contains(out.Text, "（来源：bing）") {
		t.Errorf("OK 文本应含标题与来源标注: %q", out.Text)
	}
}

// TestRunChatWebSearch_FilteredEmptySplit 刀1 关键拆分：策略过滤后空是
// EMPTY（检索发生了），不是引擎失败，更不是 ALL_FAILED。
func TestRunChatWebSearch_FilteredEmptySplit(t *testing.T) {
	stubChatSearchRun(t, func(_ context.Context, _ string, _ int) ([]builtin.SearchResult, []builtin.SearchAttempt, error) {
		return nil, []builtin.SearchAttempt{
			{Name: "bing", Outcome: "filtered-empty", ElapsedMS: 210},
			{Name: "ddg", Outcome: "error", Err: "timeout", ElapsedMS: 8000},
		}, errors.New("all engines failed")
	})
	out, err := runChatWebSearch("测试")
	if err != nil {
		t.Fatalf("滤空不该以 error 冒泡给聊天: %v", err)
	}
	if out.State != builtin.SearchStateEmpty {
		t.Errorf("State = %q, want empty（策略滤空≠ALL_FAILED）", out.State)
	}
	if !strings.Contains(out.Engines, "策略过滤后无可用结果") {
		t.Errorf("诊断应写明策略滤空: %q", out.Engines)
	}
}

// TestRunChatWebSearch_AllFailed 全部传输/解析失败 → ALL_FAILED，诊断带原因。
func TestRunChatWebSearch_AllFailed(t *testing.T) {
	stubChatSearchRun(t, func(_ context.Context, _ string, _ int) ([]builtin.SearchResult, []builtin.SearchAttempt, error) {
		return nil, []builtin.SearchAttempt{
			{Name: "bing", Outcome: "error", Err: "connection refused", ElapsedMS: 120},
			{Name: "ddg", Outcome: "error", Err: "tls handshake timeout", ElapsedMS: 8000},
		}, errors.New("all 2 engines failed")
	})
	out, _ := runChatWebSearch("测试")
	if out.State != builtin.SearchStateAllFailed {
		t.Errorf("State = %q, want all_failed", out.State)
	}
	if !strings.Contains(out.Engines, "connection refused") {
		t.Errorf("诊断应含引擎失败原因: %q", out.Engines)
	}
}

// TestInjectChatSearchResult 三层契约第二层：注入头按状态选文案（会话终稿：
// 无条件「实时搜索结果」头废——EMPTY/失败不得再套成功头）。
func TestInjectChatSearchResult(t *testing.T) {
	ok := injectChatSearchResult("问句", chatSearchOutcome{State: builtin.SearchStateOK, Query: "q", Text: "1. 结果"})
	if !strings.Contains(ok, "以下是关于此问题的实时搜索结果，请参考这些信息回答") || !strings.Contains(ok, "1. 结果") {
		t.Errorf("OK 态应注入结果头+结果: %q", ok)
	}
	empty := injectChatSearchResult("问句", chatSearchOutcome{State: builtin.SearchStateEmpty, Query: "测试汇率", Engines: "bing:无结果(200ms)"})
	if strings.Contains(empty, "实时搜索结果，请参考") {
		t.Errorf("EMPTY 态不得套成功头: %q", empty)
	}
	if !strings.Contains(empty, "联网搜索已完成") || !strings.Contains(empty, "测试汇率") || !strings.Contains(empty, "未获取到实时信息") {
		t.Errorf("EMPTY 态应如实说明: %q", empty)
	}
	failed := injectChatSearchResult("问句", chatSearchOutcome{State: builtin.SearchStateAllFailed, Engines: "bing:connection refused(120ms)"})
	if strings.Contains(failed, "实时搜索结果，请参考") {
		t.Errorf("ALL_FAILED 态不得套成功头: %q", failed)
	}
	if !strings.Contains(failed, "联网搜索暂不可用") || !strings.Contains(failed, "如实说明未获取到实时信息") {
		t.Errorf("ALL_FAILED 态应如实说明: %q", failed)
	}
}

// TestPreparePlainChatMessage_OKInjectsResults plain 路径 OK 态走新注入头。
func TestPreparePlainChatMessage_OKInjectsResults(t *testing.T) {
	a := &App{}
	stubChatWebSearch(t, func(q string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: q, Text: "1. 模拟结果"}, nil
	})
	got := a.preparePlainChatMessage("今天天气如何", false, true)
	if !strings.Contains(got, "今天天气如何") || !strings.Contains(got, "1. 模拟结果") {
		t.Errorf("OK 态应保留原文并注入结果: %q", got)
	}
}

// TestPreparePlainChatMessage_AdapterFaultFallbacks adapter 自身故障走兜底占位。
func TestPreparePlainChatMessage_AdapterFaultFallbacks(t *testing.T) {
	a := &App{}
	stubChatWebSearch(t, func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{}, errors.New("适配器故障")
	})
	got := a.preparePlainChatMessage("今天天气如何", false, true)
	if !strings.Contains(got, "联网搜索暂不可用") || !strings.Contains(got, "今天天气如何") {
		t.Errorf("adapter 故障应注入兜底占位并保留原文: %q", got)
	}
	if strings.Contains(got, "实时搜索结果，请参考") {
		t.Errorf("故障不应伪造成功结果: %q", got)
	}
}

// TestPrepareWhisperChatMessage_States 人格路径（旧实现失败静默返回原文）：
// 刀1 统一为按状态注入，失败/空结果对模型可见。
func TestPrepareWhisperChatMessage_States(t *testing.T) {
	a := &whisperState{}
	stubChatWebSearch(t, func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Text: "1. 人格搜索结果"}, nil
	})
	if got := a.prepareWhisperChatMessage("今天有什么新闻", true); !strings.Contains(got, "1. 人格搜索结果") {
		t.Errorf("人格路径 OK 态应注入结果: %q", got)
	}
	stubChatWebSearch(t, func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateAllFailed, Engines: "bing:connection refused(120ms)"}, nil
	})
	got := a.prepareWhisperChatMessage("今天有什么新闻", true)
	if !strings.Contains(got, "联网搜索暂不可用") {
		t.Errorf("人格路径失败不再静默，应注入如实占位: %q", got)
	}
}

// TestWhisperWebSearch_States 绑定入口：state 字段按状态回执，error 字段不再恒死。
func TestWhisperWebSearch_States(t *testing.T) {
	a := &whisperState{}
	stubChatWebSearch(t, func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Text: "1. 绑定结果"}, nil
	})
	m, err := a.WhisperWebSearch("测试")
	if err != nil || m["state"] != "ok" || m["result"] != "1. 绑定结果" {
		t.Errorf("OK 态回执异常: m=%v err=%v", m, err)
	}
	stubChatWebSearch(t, func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateEmpty, Engines: "bing:无结果(200ms)"}, nil
	})
	m, err = a.WhisperWebSearch("测试")
	if err != nil || m["state"] != "empty" || m["result"] != "" {
		t.Errorf("EMPTY 态回执异常: m=%v err=%v", m, err)
	}
	if !strings.Contains(m["error"].(string), "无结果") {
		t.Errorf("EMPTY 态 error 字段应如实填诊断: %v", m["error"])
	}
}

// TestChatPlainSystemPrompt_HonestContract 三层契约第三层：plain 提示词不再
// 反向施压（删「不要说自己无法联网搜索」），并要求未获取到实时信息时如实说明。
func TestChatPlainSystemPrompt_HonestContract(t *testing.T) {
	if strings.Contains(chatPlainSystemPrompt, "不要说自己无法联网搜索") {
		t.Error("plain 提示词仍含「不要说自己无法联网搜索」反向施压")
	}
	if !strings.Contains(chatPlainSystemPrompt, "如实告知用户未获取到实时信息") {
		t.Error("plain 提示词应要求未获取到实时信息时如实说明")
	}
}
