package ai

import (
	"net/http"
	"testing"
)

// Z0 蒸馏刀：xAI 会话粘性路由头——同会话请求路由一致提升服务端前缀缓存
// 命中（对话档第 2/5 轮收口）。验收口径（真机）：同会话连续两轮
// usage.prompt_tokens_details.cached_tokens 差分应显著为正。
func TestApplyConvHeader(t *testing.T) {
	cases := []struct {
		name    string
		engine  string
		convID  string
		wantSet bool
	}{
		{"xai 带会话键 设头", "xai", "sess-123", true},
		{"xai 空键 不设", "xai", "", false},
		{"其他引擎带键 不设", "tianxuan", "sess-123", false},
		{"空引擎 不设", "", "sess-123", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			applyConvHeader(h, tc.engine, tc.convID)
			got := h.Get("x-grok-conv-id")
			if tc.wantSet && got != tc.convID {
				t.Fatalf("期望头 x-grok-conv-id=%q，实得 %q", tc.convID, got)
			}
			if !tc.wantSet && got != "" {
				t.Fatalf("期望不设头，实得 %q", got)
			}
		})
	}
}
