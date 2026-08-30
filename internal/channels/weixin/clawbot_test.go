package weixin

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestServer 构造不联网的测试 Server：notify 通知全部注入替换为空实现。
// getUpdates 由各测试单独注入（避免真实 HTTP）。
func newTestServer(t *testing.T, botToken string) *Server {
	t.Helper()
	srv := New(Config{BotToken: botToken, AssistantID: "t"}, nil)
	srv.notifyStartFn = func() {}
	srv.notifyStopFn = func() {}
	return srv
}

// S4.5 发图即识别第一刀：非文本消息（图片/文件）转为模型可见提示行——
// 纯图片消息触发 chatFn 并带「收到图片」提示；图文混合时提示前置附言保留；
// 纯文本消息行为不变（不注入提示）。
func TestHandle_NonTextMessageBecomesHint(t *testing.T) {
	var got string
	srv := New(Config{BotToken: "tok", AssistantID: "t"}, func(userMsg, fromUser string) (string, error) {
		got = userMsg
		return "ok", nil
	})
	srv.sendFn = func(toUser, contextToken, text string) error { return nil }

	// 纯图片消息：item.type 非 1 + image_item（探明字段 name/url 防御性解析）
	srv.handle(&inboundMsg{
		FromUserID:   "u1",
		ContextToken: "ctx1",
		ItemList: []struct {
			Type      int        `json:"type"`
			TextItem  *textItem  `json:"text_item,omitempty"`
			ImageItem *imageItem `json:"image_item,omitempty"`
			FileItem  *fileItem  `json:"file_item,omitempty"`
		}{
			{Type: 3, ImageItem: &imageItem{Name: "照片.jpg", URL: "https://x/img.jpg"}},
		},
	})
	if !strings.Contains(got, "图片消息（照片.jpg）") || !strings.Contains(got, "内容暂无法读取") {
		t.Fatalf("纯图片消息提示 = %q, want 图片消息提示", got)
	}
	if from, _ := srv.LastPeer(); from != "u1" {
		t.Fatalf("LastPeer 未更新: %q", from)
	}

	// 图文混合：提示前置 + 附言保留原文
	got = ""
	srv.handle(&inboundMsg{
		FromUserID:   "u1",
		ContextToken: "ctx1",
		ItemList: []struct {
			Type      int        `json:"type"`
			TextItem  *textItem  `json:"text_item,omitempty"`
			ImageItem *imageItem `json:"image_item,omitempty"`
			FileItem  *fileItem  `json:"file_item,omitempty"`
		}{
			{Type: 3, ImageItem: &imageItem{Name: "图.png"}},
			{Type: 1, TextItem: &textItem{Text: "帮我看下这张图"}},
		},
	})
	if !strings.Contains(got, "图片消息（图.png）") || !strings.Contains(got, "帮我看下这张图") {
		t.Fatalf("图文混合提示 = %q, want 提示+附言", got)
	}

	// 纯文本：不含提示
	got = ""
	srv.handle(&inboundMsg{
		FromUserID: "u1",
		ItemList: []struct {
			Type      int        `json:"type"`
			TextItem  *textItem  `json:"text_item,omitempty"`
			ImageItem *imageItem `json:"image_item,omitempty"`
			FileItem  *fileItem  `json:"file_item,omitempty"`
		}{
			{Type: 1, TextItem: &textItem{Text: "你好"}},
		},
	})
	if got != "你好" {
		t.Fatalf("纯文本 = %q, want 你好（零提示注入）", got)
	}
}

// 未知消息类型（无 text/image/file 项）：不 panic、不触发 chatFn（协议未知
// 时静默降级，避免把垃圾喂给模型）。
func TestHandle_UnknownItemSilentlyIgnored(t *testing.T) {
	called := false
	srv := New(Config{BotToken: "tok", AssistantID: "t"}, func(userMsg, fromUser string) (string, error) {
		called = true
		return "", nil
	})
	srv.sendFn = func(toUser, contextToken, text string) error { return nil }
	srv.handle(&inboundMsg{
		FromUserID: "u1",
		ItemList: []struct {
			Type      int        `json:"type"`
			TextItem  *textItem  `json:"text_item,omitempty"`
			ImageItem *imageItem `json:"image_item,omitempty"`
			FileItem  *fileItem  `json:"file_item,omitempty"`
		}{
			{Type: 99}, // 无任何负载项
		},
	})
	if called {
		t.Fatal("未知空项不应触发 chatFn")
	}
}

// TestStop_IdempotentNoPanic 二次 Stop（以及从未 Start 直接 Stop）都不应 panic、无副作用。
func TestStop_IdempotentNoPanic(t *testing.T) {
	srv := newTestServer(t, "tok")
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	srv.Stop()
	srv.Stop() // 二次 Stop：close of closed channel 的旧缺陷在此修复
	if srv.IsRunning() {
		t.Error("Stop 后 IsRunning 应为 false")
	}

	// 从未 Start 直接 Stop 也应安全
	srv2 := newTestServer(t, "tok")
	srv2.Stop()
	srv2.Stop()
}

// TestStartAfterStop_RestartsPolling Stop→Start 后轮询真正恢复（不是空转）：
// 用注入 getUpdatesFn 的调用计数（atomic）验证重启后轮询继续增长。
func TestStartAfterStop_RestartsPolling(t *testing.T) {
	srv := newTestServer(t, "tok")
	var polls atomic.Int64
	srv.getUpdatesFn = func(req *pollReq, timeout time.Duration) (*pollResp, error) {
		polls.Add(1)
		time.Sleep(2 * time.Millisecond) // 模拟长轮询节奏，避免空转计数
		return &pollResp{}, nil
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if first := polls.Load(); first == 0 {
		t.Fatal("Start 后应发生轮询")
	}
	if srv.SessionExpired() {
		t.Error("启动初期 sessionExpired 应为 false")
	}

	srv.Stop()
	time.Sleep(15 * time.Millisecond) // 等旧 pollLoop 退出
	before := polls.Load()

	if err := srv.Start(); err != nil { // 重启（stopCh 已关闭，应重建）
		t.Fatalf("重启 Start: %v", err)
	}
	if srv.SessionExpired() {
		t.Error("重启后 sessionExpired 应被重置为 false")
	}
	time.Sleep(60 * time.Millisecond)
	if after := polls.Load(); after <= before {
		t.Fatalf("Stop→Start 后轮询未恢复: before=%d after=%d", before, after)
	}
	srv.Stop()
}

// TestSessionExpired_TriggersCallback getUpdates 返回 errcode=-14 sessExp 时：
// 回调被触发一次、sessionExpired=true、轮询退出（不再继续调用 getUpdates）。
func TestSessionExpired_TriggersCallback(t *testing.T) {
	srv := newTestServer(t, "tok")
	expiredCh := make(chan struct{})
	srv.OnSessionExpired = func() { close(expiredCh) }
	var calls atomic.Int64
	srv.getUpdatesFn = func(req *pollReq, timeout time.Duration) (*pollResp, error) {
		calls.Add(1)
		return &pollResp{ErrCode: sessExp}, nil
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case <-expiredCh:
	case <-time.After(2 * time.Second):
		t.Fatal("会话过期回调未被触发")
	}
	if !srv.SessionExpired() {
		t.Error("sessionExpired 应为 true")
	}

	// 回调后轮询应停止（不再 5 分钟空转）：短暂等待后 getUpdates 调用数不得再增长
	time.Sleep(40 * time.Millisecond)
	if n := calls.Load(); n != 1 {
		t.Errorf("会话过期后轮询应停止, getUpdates 调用 %d 次（期望 1）", n)
	}
	srv.Stop()
}
