package ai

// W0 多模态旁路 golden（电脑操控施工单 2026-11-19 W0 验收 1/2）：
// 无图路径必须与旧结构体序列化逐字节一致；有图路径展开 content parts
// 且 "images" 键永不出现。

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChatMessageMarshal_NoImagesUnchanged(t *testing.T) {
	cases := []struct {
		name string
		msg  ChatMessage
		want string
	}{
		{
			name: "user text",
			msg:  ChatMessage{Role: "user", Content: "hi"},
			want: `{"role":"user","content":"hi"}`,
		},
		{
			name: "assistant tool calls",
			msg: ChatMessage{Role: "assistant", ToolCalls: []ChatToolCall{{
				ID:   "1",
				Type: "function",
				Function: ChatToolFunction{
					Name:      "f",
					Arguments: "{}",
				},
			}}},
			want: `{"role":"assistant","content":"","tool_calls":[{"id":"1","type":"function","function":{"name":"f","arguments":"{}"}}]}`,
		},
		{
			name: "tool result",
			msg:  ChatMessage{Role: "tool", Content: "ok", ToolCallID: "9", Name: "t"},
			want: `{"role":"tool","content":"ok","tool_call_id":"9","name":"t"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.msg)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(b) != tc.want {
				t.Fatalf("无图路径必须逐字节一致\n got: %s\nwant: %s", b, tc.want)
			}
		})
	}
}

func TestChatMessageMarshal_WithImages(t *testing.T) {
	const dataURL = "data:image/jpeg;base64,AAAA"
	b, err := json.Marshal(ChatMessage{Role: "user", Content: "hi", Images: []string{dataURL}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if strings.Contains(s, `"images"`) {
		t.Fatalf("images 键不得出现在 JSON 里: %s", s)
	}
	var got struct {
		Role    string `json:"role"`
		Content []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			ImageURL *struct {
				URL string `json:"url"`
			} `json:"image_url"`
		} `json:"content"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Role != "user" || len(got.Content) != 2 {
		t.Fatalf("应为 text+image_url 两 parts: role=%s parts=%d", got.Role, len(got.Content))
	}
	if got.Content[0].Type != "text" || got.Content[0].Text != "hi" {
		t.Fatalf("首 part 应为 text: %+v", got.Content[0])
	}
	if got.Content[1].Type != "image_url" || got.Content[1].ImageURL == nil || got.Content[1].ImageURL.URL != dataURL {
		t.Fatalf("次 part 应为 image_url data URL: %+v", got.Content[1])
	}
}

func TestChatMessageMarshal_WithImagesToolMessage(t *testing.T) {
	// tool 结果带图：role/tool_call_id/name 照旧，content 展开 parts。
	b, err := json.Marshal(ChatMessage{Role: "tool", Content: "shot", Images: []string{"data:image/jpeg;base64,BBBB"}, ToolCallID: "7", Name: "computer_state"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"tool_call_id":"7"`) || !strings.Contains(s, `"name":"computer_state"`) {
		t.Fatalf("tool 消息元数据不得丢: %s", s)
	}
	if strings.Contains(s, `"images"`) {
		t.Fatalf("images 键不得出现: %s", s)
	}
	if !strings.Contains(s, `"content":[{`) {
		t.Fatalf("有图 tool 消息 content 应为 parts 数组: %s", s)
	}
}
