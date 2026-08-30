package weixin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestPush_NoPeerFails 启动后从未收到消息：Push 应报错（无回推目标），不发任何请求。
func TestPush_NoPeerFails(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(200)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t"}, nil)
	if err := s.Push("你好"); err == nil {
		t.Fatal("无活跃会话时 Push 应报错")
	}
	if hits != 0 {
		t.Fatalf("不应发起任何 HTTP 请求，实际 %d 次", hits)
	}
}

// TestPush_AfterHandleTargetsLastPeer handle 收到消息后记录最近活跃会话；
// Push 应向该 to_user_id/context_token 发送文本 item。
func TestPush_AfterHandleTargetsLastPeer(t *testing.T) {
	var mu sync.Mutex
	var lastMsg map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Msg map[string]interface{} `json:"msg"`
		}
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		mu.Lock()
		lastMsg = body.Msg
		mu.Unlock()
		w.WriteHeader(200)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t"}, func(msg, from string) (string, error) {
		return "好的", nil
	})
	s.handle(&inboundMsg{
		FromUserID:   "user-1",
		ContextToken: "ctx-1",
		ItemList: []struct {
			Type      int        `json:"type"`
			TextItem  *textItem  `json:"text_item,omitempty"`
			ImageItem *imageItem `json:"image_item,omitempty"`
			FileItem  *fileItem  `json:"file_item,omitempty"`
		}{
			{Type: 1, TextItem: &textItem{Text: "提醒我 18:00 开会"}},
		},
	})

	if from, ctx := s.LastPeer(); from != "user-1" || ctx != "ctx-1" {
		t.Fatalf("LastPeer = (%q,%q)，期望 (user-1,ctx-1)", from, ctx)
	}
	if err := s.Push("⏰ 提醒：开会"); err != nil {
		t.Fatalf("Push: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if lastMsg == nil {
		t.Fatal("Push 未发起 sendmessage 请求")
	}
	if lastMsg["to_user_id"] != "user-1" {
		t.Errorf("to_user_id = %v，期望 user-1", lastMsg["to_user_id"])
	}
	if lastMsg["context_token"] != "ctx-1" {
		t.Errorf("context_token = %v，期望 ctx-1", lastMsg["context_token"])
	}
	items, _ := lastMsg["item_list"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("item_list 长度 = %d，期望 1", len(items))
	}
	item, _ := items[0].(map[string]interface{})
	if item["type"] != float64(1) {
		t.Errorf("item type = %v，期望 1（文本）", item["type"])
	}
}

// TestLastPeer_EmptyOnFresh 从未收到消息时 LastPeer 返回空值。
func TestLastPeer_EmptyOnFresh(t *testing.T) {
	s := newTestServer(t, "tok")
	from, ctx := s.LastPeer()
	if from != "" || ctx != "" {
		t.Fatalf("全新 Server 的 LastPeer 应为空，实际 (%q,%q)", from, ctx)
	}
}
