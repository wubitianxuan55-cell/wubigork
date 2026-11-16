package app

import "testing"

// B1 错误分类单测（chat-demo describe.ts 语义对齐）：Go 侧把无结构引擎错误
// 字符串归入稳定 code，前端按码出文案与动作。未命中必须回落 ""（前端展示原文，
// 零行为变化）。
func TestClassifyChatStreamError(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want string
	}{
		{"持久化失败显式文案", "回复生成完成但消息保存失败: disk full", "persistence"},
		{"空回复显式文案", "本轮未生成可交付正文，请重试", "empty_reply"},
		{"连接拒绝", `流式请求失败: Post "http://127.0.0.1:8080/v1/chat": dial tcp 127.0.0.1:8080: connection refused`, "engine_unreachable"},
		{"连接重置", "流式请求失败: read: connection reset by peer", "engine_unreachable"},
		{"连接失败中文", "无法连接引擎，请检查本机引擎状态", "engine_unreachable"},
		{"鉴权失败", "API 错误 (HTTP 401): invalid api key", "auth"},
		{"限流", "API 错误 (HTTP 429): rate limited", "engine_busy"},
		{"服务端繁忙", "API 错误 (HTTP 503): server overloaded", "engine_busy"},
		{"请求超时", "请求超时或已取消", "timeout"},
		{"空闲超时", "stream idle timeout", "timeout"},
		{"上下文超限", "context length exceeded", "context_overflow"},
		{"未命中回落空码", "未知怪错", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyChatStreamError(tc.msg); got != tc.want {
				t.Fatalf("classifyChatStreamError(%q) = %q, want %q", tc.msg, got, tc.want)
			}
		})
	}
}
