package intent

import "testing"

// TestParseFallback 受控校验表驱动（v4.8 LLM 兜底最后一道闸）。
func TestParseFallback(t *testing.T) {
	ok := []struct {
		name string
		raw  string
		want Action
		tgt  string
	}{
		{"导航", `{"action":"navigate","target":"imagegen","confidence":0.9}`, ActionNavigate, "imagegen"},
		{"状态", `{"action":"status","target":"model","confidence":0.95}`, ActionStatus, "model"},
		{"读屏", `{"action":"read_screen","target":"screen:2","confidence":0.88}`, ActionReadScreen, "screen:2"},
		{"围栏噪音", "好的，分类如下：\n```json\n{\"action\":\"navigate\",\"target\":\"cost\",\"confidence\":0.9}\n```", ActionNavigate, "cost"},
	}
	for _, tc := range ok {
		t.Run(tc.name, func(t *testing.T) {
			it := ParseFallback(tc.raw)
			if it == nil {
				t.Fatalf("应命中: %s", tc.raw)
			}
			if it.Action != tc.want || it.Target != tc.tgt {
				t.Errorf("= %s/%q, want %s/%q", it.Action, it.Target, tc.want, tc.tgt)
			}
		})
	}

	reject := []struct {
		name string
		raw  string
	}{
		{"坏JSON", "这不是 JSON"},
		{"生图不放行", `{"action":"generate_image","target":"猫","confidence":0.99}`},
		{"提醒不放行", `{"action":"reminder","target":"","confidence":0.99}`},
		{"none不放行", `{"action":"none","target":"","confidence":0.99}`},
		{"低置信", `{"action":"navigate","target":"chat","confidence":0.5}`},
		{"零置信", `{"action":"status","target":"model","confidence":0}`},
		{"导航缺target", `{"action":"navigate","target":"","confidence":0.95}`},
		{"空对象", `{}`},
		{"未知动作", `{"action":"open_app","target":"wechat","confidence":0.99}`},
	}
	for _, tc := range reject {
		t.Run(tc.name, func(t *testing.T) {
			if it := ParseFallback(tc.raw); it != nil {
				t.Fatalf("应拒绝: %s → %s/%q", tc.raw, it.Action, it.Target)
			}
		})
	}
}
