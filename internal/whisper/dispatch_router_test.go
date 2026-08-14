package whisper

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// ─── T6-5.1: dispatch_router.go ─────────────────────────────────

func testDispatchCandidates() []DispatchCandidate {
	return []DispatchCandidate{
		{ID: "weather", Name: "天气查询", Summary: "查询当前或未来天气", Keywords: []string{"天气"}, Score: 0.8},
	}
}

func TestRouteDispatch_ComplexTask(t *testing.T) {
	got := RouteDispatch(RouteDispatchInput{UserMessage: "帮我写一份周报"})
	if got.Decision != DispatchPlan || got.PlanTopic != "一份周报" {
		t.Errorf("复杂任务应路由到 plan, got %+v", got)
	}
}

func TestRouteDispatch_NoCandidates(t *testing.T) {
	got := RouteDispatch(RouteDispatchInput{UserMessage: "今天过得如何", LlmCall: func(string) (string, error) { return "{}", nil }})
	if got.Decision != DispatchChat || got.Reasoning != "no_candidates" {
		t.Errorf("无候选应回退 chat, got %+v", got)
	}
}

func TestRouteDispatch_NoLLMCall(t *testing.T) {
	got := RouteDispatch(RouteDispatchInput{UserMessage: "明天天气怎么样"})
	if got.Decision != DispatchChat || got.Reasoning != "no_llm_call" {
		t.Errorf("无 LLM 调用应回退 chat, got %+v", got)
	}
}

func TestRouteDispatch_LLMError(t *testing.T) {
	got := RouteDispatch(RouteDispatchInput{
		UserMessage: "明天天气怎么样", Candidates: testDispatchCandidates(),
		LlmCall: func(string) (string, error) { return "", errors.New("boom") },
	})
	if got.Decision != DispatchChat || got.Reasoning != "llm_error" {
		t.Errorf("LLM 错误应回退 chat, got %+v", got)
	}
}

func TestRouteDispatch_LLMNoMatch(t *testing.T) {
	raw := "{\"matched\":false}"

	got := RouteDispatch(RouteDispatchInput{
		UserMessage: "明天天气怎么样", Candidates: testDispatchCandidates(),
		LlmCall: func(string) (string, error) { return raw, nil },
	})
	if got.Decision != DispatchSilent || got.Reasoning != "llm_no_match" {
		t.Errorf("LLM 未匹配应 silent, got %+v", got)
	}
}

func TestRouteDispatch_UnknownCandidate(t *testing.T) {
	raw := "{\"matched\":true,\"extension_id\":\"nope\",\"confidence\":0.9}"

	got := RouteDispatch(RouteDispatchInput{
		UserMessage: "明天天气怎么样", Candidates: testDispatchCandidates(),
		LlmCall: func(string) (string, error) { return raw, nil },
	})
	if got.Decision != DispatchSilent || got.Reasoning != "unknown_candidate" {
		t.Errorf("未知候选应 silent, got %+v", got)
	}
}

func TestRouteDispatch_AutoInvoke(t *testing.T) {
	raw := "{\"matched\":true,\"extension_id\":\"weather\",\"confidence\":0.9,\"reasoning\":\"明确天气意图\"}"

	got := RouteDispatch(RouteDispatchInput{
		UserMessage: "明天天气怎么样", Candidates: testDispatchCandidates(),
		LlmCall: func(string) (string, error) { return raw, nil },
	})
	if got.Decision != DispatchAutoInvoke || got.ExtensionID != "weather" {
		t.Errorf("高置信应自动触发, got %+v", got)
	}
	if got.Confidence != 0.9 {
		t.Errorf("confidence 应透传, got %v", got.Confidence)
	}
}

func TestRouteDispatch_AskInvoke(t *testing.T) {
	raw := "{\"matched\":true,\"extension_id\":\"weather\",\"confidence\":0.7}"

	got := RouteDispatch(RouteDispatchInput{
		UserMessage: "明天天气怎么样", Candidates: testDispatchCandidates(),
		LlmCall: func(string) (string, error) { return raw, nil },
	})
	if got.Decision != DispatchAskInvoke || !strings.Contains(got.AskMessage, "天气查询") {
		t.Errorf("中置信应询问触发, got %+v", got)
	}
}

func TestRouteDispatch_SilentBelowThreshold(t *testing.T) {
	raw := "{\"matched\":true,\"extension_id\":\"weather\",\"confidence\":0.4}"

	got := RouteDispatch(RouteDispatchInput{
		UserMessage: "明天天气怎么样", Candidates: testDispatchCandidates(),
		LlmCall: func(string) (string, error) { return raw, nil },
	})
	if got.Decision != DispatchSilent || got.ExtensionID != "weather" {
		t.Errorf("低置信应 silent, got %+v", got)
	}
}

func TestRouteDispatch_PersonalityModifier(t *testing.T) {
	raw := "{\"matched\":true,\"extension_id\":\"weather\",\"confidence\":0.8}"

	// deredere 1.15：0.8*1.15=0.92 → auto_invoke
	got := RouteDispatch(RouteDispatchInput{
		UserMessage: "明天天气怎么样", Candidates: testDispatchCandidates(),
		PersonalityID: "deredere", LlmCall: func(string) (string, error) { return raw, nil },
	})
	if got.Decision != DispatchAutoInvoke {
		t.Errorf("deredere 乘数应触发 auto, got %+v", got)
	}
	// tsundere 0.9：0.8*0.9=0.72 → ask_invoke
	got2 := RouteDispatch(RouteDispatchInput{
		UserMessage: "明天天气怎么样", Candidates: testDispatchCandidates(),
		PersonalityID: "tsundere", LlmCall: func(string) (string, error) { return raw, nil },
	})
	if got2.Decision != DispatchAskInvoke {
		t.Errorf("tsundere 乘数应降为 ask, got %+v", got2)
	}
}

func TestRouteDispatch_ConfidenceFallbackToScore(t *testing.T) {
	raw := "{\"matched\":true,\"extension_id\":\"weather\"}"

	// LLM 未给 confidence → 使用候选 Score 0.8 → ask_invoke
	got := RouteDispatch(RouteDispatchInput{
		UserMessage: "明天天气怎么样", Candidates: testDispatchCandidates(),
		LlmCall: func(string) (string, error) { return raw, nil },
	})
	if got.Decision != DispatchAskInvoke || got.Confidence != 0.8 {
		t.Errorf("应回退候选 Score, got %+v", got)
	}
}

func TestIsComplexTaskRequest(t *testing.T) {
	trueCases := []string{"帮我做一份计划", "帮我查一下资料", "帮我整理桌面文件", "写一个脚本", "翻译这句话", "计算一下"}
	for _, msg := range trueCases {
		if !isComplexTaskRequest(msg) {
			t.Errorf("%q 应为复杂任务", msg)
		}
	}
	falseCases := []string{"今天天气不错", "晚安", "在吗", "哦"}
	for _, msg := range falseCases {
		if isComplexTaskRequest(msg) {
			t.Errorf("%q 不应是复杂任务", msg)
		}
	}
}

func TestExtractComplexTopic(t *testing.T) {
	cases := map[string]string{
		"帮我写一份周报":   "一份周报",
		"帮我做一顿饭":     "一顿饭",
		"帮我整理一下桌面":   "一下桌面",
	}
	for msg, want := range cases {
		if got := extractComplexTopic(msg); got != want {
			t.Errorf("extractComplexTopic(%q) = %q, want %q", msg, got, want)
		}
	}
	// 无前缀 → 截断原文
	if got := extractComplexTopic("随便聊聊今天发生的事"); got == "" {
		t.Error("无前缀时不应为空")
	}
}

func TestCollectBuiltinCandidates(t *testing.T) {
	got := collectBuiltinCandidates("明天天气怎么样")
	if len(got) != 1 || got[0].ID != "weather" {
		t.Errorf("天气消息应命中 weather, got %v", got)
	}
	got = collectBuiltinCandidates("记得提醒我开会")
	if len(got) != 1 || got[0].ID != "reminder" {
		t.Errorf("提醒消息应命中 reminder, got %v", got)
	}
	if got := collectBuiltinCandidates("随便聊聊"); len(got) != 0 {
		t.Errorf("无关消息应无候选, got %v", got)
	}
}

func TestParseDispatchLlmMatch(t *testing.T) {
	raw := "{\"matched\":true,\"extension_id\":\"weather\",\"confidence\":0.88,\"reasoning\":\"r\"}"

	got := parseDispatchLlmMatch(raw)
	if !got.Matched || got.ExtensionID != "weather" || got.Confidence != 0.88 {
		t.Errorf("解析匹配失败: %+v", got)
	}
	// 无 JSON 返回零值
	if got := parseDispatchLlmMatch("纯文本"); got.Matched {
		t.Error("无 JSON 应返回零值")
	}
}

func TestBuildDispatchLlmPrompt(t *testing.T) {
	got := buildDispatchLlmPrompt("帮我查天气", testDispatchCandidates(), "最近对话", "平静理性", "记忆块", "工作", timeNow())
	for _, want := range []string{"帮我查天气", "weather", "查询当前或未来天气", "时间："} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt 缺少 %q", want)
		}
	}
	// 空上下文不渲染对应行
	got2 := buildDispatchLlmPrompt("x", testDispatchCandidates(), "", "", "", "", timeNow())
	if strings.Contains(got2, "最近对话：") {
		t.Error("空上下文不应渲染最近对话行")
	}
}

func timeNow() time.Time { return time.Unix(1700000000, 0).UTC() }
