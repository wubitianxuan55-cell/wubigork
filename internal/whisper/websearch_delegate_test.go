package whisper

// websearch_delegate_test.go — agent loop 联网委托缝测试（众神殿刀1：四入口
// 收敛后 whisper 只留缝；未接线 fail-loud、接线后诚实回执）。

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestWhisperRunWebSearch_UnwiredFailLoud 缝未接线（boot 未注入）时如实报
// 「未接线」，不静默伪装成功。
func TestWhisperRunWebSearch_UnwiredFailLoud(t *testing.T) {
	orig := WebSearchToolExec
	WebSearchToolExec = nil
	t.Cleanup(func() { WebSearchToolExec = orig })

	got := whisperRunWebSearch(context.Background(), "测试查询")
	if !strings.Contains(got, "未接线") || !strings.Contains(got, "测试查询") {
		t.Errorf("未接线应如实回执: %q", got)
	}
}

// TestWhisperRunWebSearch_Delegates 接线后透传统一核回执；query 原样不清洗。
func TestWhisperRunWebSearch_Delegates(t *testing.T) {
	orig := WebSearchToolExec
	WebSearchToolExec = func(_ context.Context, query string) (string, error) {
		if query != "site:example.invalid gaea 测试" {
			t.Errorf("query 应原样透传（含操作符不改写）, got %q", query)
		}
		return `[{"title":"t","url":"https://x","snippet":"s","source":"bing"}]`, nil
	}
	t.Cleanup(func() { WebSearchToolExec = orig })

	got := whisperRunWebSearch(context.Background(), "site:example.invalid gaea 测试")
	if !strings.Contains(got, `"title":"t"`) {
		t.Errorf("应透传统一核 JSON 回执: %q", got)
	}
}

// TestWhisperRunWebSearch_HonestFailures 统一核报错/空回执均如实呈现
// （原「搜索「x」失败」分支因旧栈几乎不返 error 而恒死，现成为活路径）。
func TestWhisperRunWebSearch_HonestFailures(t *testing.T) {
	orig := WebSearchToolExec
	t.Cleanup(func() { WebSearchToolExec = orig })

	WebSearchToolExec = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("所有搜索引擎失败：bing (120ms): connection refused")
	}
	if got := whisperRunWebSearch(context.Background(), "q"); !strings.Contains(got, "搜索「q」失败") || !strings.Contains(got, "connection refused") {
		t.Errorf("统一核报错应如实回执: %q", got)
	}

	WebSearchToolExec = func(_ context.Context, _ string) (string, error) { return "  ", nil }
	if got := whisperRunWebSearch(context.Background(), "q"); !strings.Contains(got, "无结果") {
		t.Errorf("空回执应如实报无结果: %q", got)
	}
}
