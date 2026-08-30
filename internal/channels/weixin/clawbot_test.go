package weixin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
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

// ─── v4.8 子项 d：入站防线 ──────────────────────────────────

// itemElem 与 inboundMsg.ItemList 的匿名元素类型完全一致（type alias 保证
// 类型同一，可直接构造 ItemList）。
type itemElem = struct {
	Type      int        `json:"type"`
	TextItem  *textItem  `json:"text_item,omitempty"`
	ImageItem *imageItem `json:"image_item,omitempty"`
	FileItem  *fileItem  `json:"file_item,omitempty"`
}

// newHandleCaptureServer 构造记录 chatFn 输入的测试 Server。
func newHandleCaptureServer() (*Server, *string, *bool) {
	got := ""
	called := false
	srv := New(Config{BotToken: "tok", AssistantID: "t"}, func(userMsg, fromUser string) (string, error) {
		got = userMsg
		called = true
		return "ok", nil
	})
	srv.sendFn = func(toUser, contextToken, text string) error { return nil }
	return srv, &got, &called
}

// TestRateLimiter_SlidingWindow 滑动窗口语义：窗口内限流、per-peer 隔离、
// 时钟可注入（整窗滑过后恢复）。
func TestRateLimiter_SlidingWindow(t *testing.T) {
	now := time.Now()
	rl := newRateLimiter(2, time.Minute)
	rl.clock = func() time.Time { return now }

	if !rl.Allow("u1") || !rl.Allow("u1") {
		t.Fatal("窗口内前 2 条应放行")
	}
	if rl.Allow("u1") {
		t.Fatal("窗口内第 3 条应被拒")
	}
	if !rl.Allow("u2") {
		t.Fatal("per-peer 隔离：u2 不受 u1 影响")
	}
	now = now.Add(30 * time.Second)
	if rl.Allow("u1") {
		t.Fatal("半窗滑动后旧记录未过期，仍应受限")
	}
	now = now.Add(31 * time.Second)
	if !rl.Allow("u1") {
		t.Fatal("整窗滑过后应恢复放行")
	}
}

// TestHandle_RateLimitFixedReplyNoLLM 超限发固定文案且不触发 LLM，窗口滑过恢复。
func TestHandle_RateLimitFixedReplyNoLLM(t *testing.T) {
	var calls int
	var lastSent string
	srv := New(Config{BotToken: "tok", AssistantID: "t"}, func(userMsg, fromUser string) (string, error) {
		calls++
		return "llm", nil
	})
	srv.sendFn = func(toUser, contextToken, text string) error { lastSent = text; return nil }
	now := time.Now()
	srv.limiter.clock = func() time.Time { return now }
	textMsg := func() *inboundMsg {
		return &inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "hi"}}}}
	}

	for i := 0; i < wxRateLimit; i++ {
		srv.handle(textMsg())
	}
	if calls != wxRateLimit {
		t.Fatalf("窗口内 %d 条应全部放行, chatFn 实际触发 %d 次", wxRateLimit, calls)
	}

	// 第 21 条（同一窗口）：固定文案回复，不触发 LLM
	srv.handle(textMsg())
	if calls != wxRateLimit {
		t.Fatalf("超限消息不应触发 chatFn, 实际 %d 次", calls)
	}
	if lastSent != rateLimitedText {
		t.Fatalf("超限回复 = %q, want 固定文案 %q", lastSent, rateLimitedText)
	}

	// 窗口滑过：恢复放行
	now = now.Add(wxRateWindow + time.Second)
	srv.handle(textMsg())
	if calls != wxRateLimit+1 {
		t.Fatalf("窗口滑过后应恢复触发 chatFn, 实际 %d 次", calls)
	}
}

// TestHandle_EmptyReplyNotSent 回调返回空串=产物已走 SendFileCard 图片卡片
// （v4.8.3 接线约定），handle 必须跳过推送——否则发空文本泡。
func TestHandle_EmptyReplyNotSent(t *testing.T) {
	var sent []string
	srv := New(Config{BotToken: "tok", AssistantID: "t"}, func(string, string) (string, error) {
		return "", nil // 模拟 whisper_state 图卡片路径：已由 seam 送出
	})
	srv.sendFn = func(toUser, contextToken, text string) error {
		sent = append(sent, text)
		return nil
	}
	srv.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "hi"}}}})
	if len(sent) != 0 {
		t.Fatalf("空回复不应触发推送, 实际 sent=%v", sent)
	}
}

// TestHandle_RateLimitPerPeerIsolation u1 打满后 u2 首条仍放行。
func TestHandle_RateLimitPerPeerIsolation(t *testing.T) {
	var calls int
	srv := New(Config{BotToken: "tok", AssistantID: "t"}, func(string, string) (string, error) {
		calls++
		return "ok", nil
	})
	srv.sendFn = func(string, string, string) error { return nil }
	now := time.Now()
	srv.limiter.clock = func() time.Time { return now }

	for i := 0; i < wxRateLimit; i++ {
		srv.handle(&inboundMsg{FromUserID: "u1", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "hi"}}}})
	}
	srv.handle(&inboundMsg{FromUserID: "u2", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "hi"}}}})
	if calls != wxRateLimit+1 {
		t.Fatalf("u2 不应受 u1 限频影响, chatFn 实际 %d 次", calls)
	}
}

// TestHandle_TextTruncatedAt4KB 超长文本 4KB 截断（rune 安全 + 固定标记），
// 未超长原样保留。
func TestHandle_TextTruncatedAt4KB(t *testing.T) {
	srv, got, _ := newHandleCaptureServer()

	long := strings.Repeat("汉", 3000) // 9000 字节
	srv.handle(&inboundMsg{FromUserID: "u1", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: long}}}})
	if !strings.Contains(*got, truncatedMark) {
		t.Fatalf("超长文本应带截断标记, got=%d 字节", len(*got))
	}
	if !utf8.ValidString(*got) {
		t.Fatal("截断后应仍是合法 UTF-8")
	}
	body := strings.TrimSuffix(*got, truncatedMark)
	if len(body) > wxMaxTextBytes {
		t.Fatalf("截断后正文 %d 字节应 ≤ %d", len(body), wxMaxTextBytes)
	}
	if !strings.HasPrefix(long, body) {
		t.Fatal("截断正文应是原文前缀")
	}

	*got = ""
	srv.handle(&inboundMsg{FromUserID: "u1", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "短消息"}}}})
	if *got != "短消息" {
		t.Fatalf("未超长文本不应改动: %q", *got)
	}
}

// TestHandle_MediaItemsOverCap 多媒体超过 5 条：只处理前 5 条，超出拼
// 「…等 N 个文件」提示行，识别也只对前 5 条调用。
func TestHandle_MediaItemsOverCap(t *testing.T) {
	srv, got, _ := newHandleCaptureServer()
	var recogCalls int
	srv.MediaRecognizer = func(u string) (string, error) { recogCalls++; return "图中文字内容", nil }

	var items []itemElem
	for i := 0; i < 7; i++ {
		items = append(items, itemElem{Type: 3, ImageItem: &imageItem{Name: fmt.Sprintf("图%d.png", i), URL: fmt.Sprintf("https://cdn/%d.png", i)}})
	}
	srv.handle(&inboundMsg{FromUserID: "u1", ItemList: items})

	if n := strings.Count(*got, "识别内容"); n != wxMaxMediaItems {
		t.Fatalf("只应识别前 %d 条, 实际 %d 处「识别内容」", wxMaxMediaItems, n)
	}
	if recogCalls != wxMaxMediaItems {
		t.Fatalf("识别只应对前 %d 条调用, 实际 %d 次", wxMaxMediaItems, recogCalls)
	}
	if !strings.Contains(*got, "…等 2 个文件") {
		t.Fatalf("超出条数应拼「…等 N 个文件」提示行: %q", *got)
	}
	if !strings.Contains(*got, "内容暂无法读取") {
		t.Fatal("有未处理条目时应保留占位包装")
	}

	// 无识别器时：前 5 条出占位标签，超出聚合
	srv2, got2, _ := newHandleCaptureServer()
	srv2.handle(&inboundMsg{FromUserID: "u1", ItemList: items})
	if n := strings.Count(*got2, "图片消息"); n != wxMaxMediaItems {
		t.Fatalf("应只出现前 %d 条占位标签, 实际 %d 处「图片消息」", wxMaxMediaItems, n)
	}
	if !strings.Contains(*got2, "…等 2 个文件") {
		t.Fatalf("无识别器时超限同样聚合: %q", *got2)
	}
}

// ─── v4.8 子项 a：防御解析矩阵（原始 JSON 走真实 unmarshal）──

// TestUnmarshal_DefensiveMatrix 多态/怪异负载走真实 json.Unmarshal 后进
// handle：类型不匹配降级零值（整条消息解析不失败）、未知 type 静默、
// 超长字段不 panic。用例 raw 为单条 msg 对象，mk 按 pollResp 形态包一层
// `{"msgs":[...]}`（对齐 getupdates 真实响应结构）。
func TestUnmarshal_DefensiveMatrix(t *testing.T) {
	mk := func(t *testing.T, raw string) *inboundMsg {
		t.Helper()
		var resp pollResp
		wrapped := `{"msgs":[` + raw + `]}`
		if err := json.Unmarshal([]byte(wrapped), &resp); err != nil {
			t.Fatalf("防御解析不应失败: %v", err)
		}
		if len(resp.Msgs) != 1 {
			t.Fatalf("应解析出 1 条消息, 实际 %d", len(resp.Msgs))
		}
		return &resp.Msgs[0]
	}

	cases := []struct {
		name         string
		raw          string
		wantChat     bool
		wantContains []string
		wantMaxLen   int // >0 时校验 chatFn 收到文本的字节上限
	}{
		{
			name:         "url是数组形态",
			raw:          `{"from_user_id":"u1","item_list":[{"type":3,"image_item":{"name":"a.png","url":["https://x/1"],"file_id":"f1"}}]}`,
			wantChat:     true,
			wantContains: []string{"图片消息（a.png）", "内容暂无法读取"},
		},
		{
			name:         "url是对象形态",
			raw:          `{"from_user_id":"u1","item_list":[{"type":3,"image_item":{"name":"b.png","url":{"href":"https://x/2"}}}]}`,
			wantChat:     true,
			wantContains: []string{"图片消息（b.png）"},
		},
		{
			name:         "file_id是数字",
			raw:          `{"from_user_id":"u1","item_list":[{"type":3,"image_item":{"file_id":12345,"url":"https://x/a.png"}}]}`,
			wantChat:     true,
			wantContains: []string{"图片消息"},
		},
		{
			name:     "全缺字段静默",
			raw:      `{"from_user_id":"u1","item_list":[{"type":3}]}`,
			wantChat: false,
		},
		{
			name: "未知type2/4/99静默",
			raw:  `{"from_user_id":"u1","item_list":[{"type":2},{"type":4},{"type":99}]}`,
		},
		{
			name:         "emoji与中文文件名及带query的URL",
			raw:          `{"from_user_id":"u1","item_list":[{"type":3,"image_item":{"name":"🐱 猫猫📷.png","url":"https://x/%E7%8C%AB.png?sig=ab&x=1"}}]}`,
			wantChat:     true,
			wantContains: []string{"🐱 猫猫📷.png"},
		},
		{
			name:         "图文混合提示前置附言保留",
			raw:          `{"from_user_id":"u1","item_list":[{"type":1,"text_item":{"text":"帮我看图"}},{"type":3,"image_item":{"name":"图.png"}}]}`,
			wantChat:     true,
			wantContains: []string{"图片消息（图.png）", "附言：帮我看图"},
		},
		{
			name: "多媒体混合超上限聚合",
			raw: `{"from_user_id":"u1","item_list":[` +
				`{"type":3,"image_item":{"name":"a.png"}},{"type":3,"image_item":{"name":"b.png"}},{"type":3,"image_item":{"name":"c.png"}},` +
				`{"type":4,"file_item":{"name":"d.pdf"}},{"type":4,"file_item":{"name":"e.pdf"}},` +
				`{"type":4,"file_item":{"name":"f.pdf"}},{"type":4,"file_item":{"name":"g.pdf"}}]}`,
			wantChat:     true,
			wantContains: []string{"…等 2 个文件"},
		},
		{
			name: "100KB超长name与url不panic",
			raw: fmt.Sprintf(`{"from_user_id":"u1","item_list":[{"type":3,"image_item":{"name":%q,"url":"https://x/%s"}}]}`,
				strings.Repeat("长", 50000), strings.Repeat("u", 100000)),
			wantChat:     true,
			wantContains: []string{truncatedMark},
			wantMaxLen:   wxMaxTextBytes + len(truncatedMark),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, got, called := newHandleCaptureServer()
			srv.handle(mk(t, tc.raw))
			if *called != tc.wantChat {
				t.Fatalf("chatFn 触发 = %v, want %v（got=%q）", *called, tc.wantChat, *got)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(*got, want) {
					t.Fatalf("提示应含 %q: %q", want, *got)
				}
			}
			if tc.wantMaxLen > 0 && len(*got) > tc.wantMaxLen {
				t.Fatalf("提示应 ≤ %d 字节, 实际 %d", tc.wantMaxLen, len(*got))
			}
		})
	}
}

// TestUnmarshal_FieldCoercion 多态字段降级细节：file_id 数字→字符串、
// size 字符串→数值、url 数组→空串，均不报错。
func TestUnmarshal_FieldCoercion(t *testing.T) {
	raw := `{"msgs":[{"from_user_id":"u1","item_list":[` +
		`{"type":3,"image_item":{"file_id":12345,"url":"https://x/a.png"}},` +
		`{"type":4,"file_item":{"name":"a.pdf","size":"2048","url":["bad"]}}]}]}`
	var resp pollResp
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("多态字段不应让解析失败: %v", err)
	}
	if len(resp.Msgs) != 1 || len(resp.Msgs[0].ItemList) != 2 {
		t.Fatalf("应解析出 1 条消息 2 个 item, 实际 msgs=%d", len(resp.Msgs))
	}
	img := resp.Msgs[0].ItemList[0].ImageItem
	if img == nil || img.FileID != "12345" || img.URL != "https://x/a.png" {
		t.Fatalf("image_item 降级结果 = %+v", img)
	}
	fi := resp.Msgs[0].ItemList[1].FileItem
	if fi == nil || fi.Size != 2048 || fi.URL != "" || fi.Name != "a.pdf" {
		t.Fatalf("file_item 降级结果 = %+v", fi)
	}
}

// ─── v4.8 子项 b：图片识别管线 ──────────────────────────────

// TestHandle_MediaRecognizerPipeline 识别成功替换占位提示（内容截前 300
// rune）；失败/空结果保留占位；无 URL 不调用识别器；识别器为 nil 行为与
// 现状一致（其余用例覆盖）。
func TestHandle_MediaRecognizerPipeline(t *testing.T) {
	// 成功路
	srv, got, _ := newHandleCaptureServer()
	var gotURL string
	srv.MediaRecognizer = func(u string) (string, error) { gotURL = u; return strings.Repeat("字", 400), nil }
	srv.handle(&inboundMsg{FromUserID: "u1", ItemList: []itemElem{
		{Type: 3, ImageItem: &imageItem{Name: "截图.png", URL: "https://cdn/img.png"}},
	}})
	if gotURL != "https://cdn/img.png" {
		t.Fatalf("识别器应收到图片 URL: %q", gotURL)
	}
	if strings.Contains(*got, "内容暂无法读取") {
		t.Fatalf("识别成功不应保留占位提示: %q", *got)
	}
	want := "（用户发来图片「截图.png」，识别内容：" + strings.Repeat("字", 300) + "…）"
	if !strings.Contains(*got, want) {
		t.Fatalf("识别提示应含 300 rune 截断: %q", *got)
	}

	// 失败路：保留占位提示
	srv2, got2, _ := newHandleCaptureServer()
	srv2.MediaRecognizer = func(u string) (string, error) { return "", errors.New("识别引擎不可用") }
	srv2.handle(&inboundMsg{FromUserID: "u1", ItemList: []itemElem{
		{Type: 3, ImageItem: &imageItem{Name: "截图.png", URL: "https://cdn/img.png"}},
	}})
	if !strings.Contains(*got2, "图片消息（截图.png）") || !strings.Contains(*got2, "内容暂无法读取") {
		t.Fatalf("识别失败应保留占位提示: %q", *got2)
	}

	// 空结果路：与失败同路
	srv3, got3, _ := newHandleCaptureServer()
	srv3.MediaRecognizer = func(u string) (string, error) { return "  ", nil }
	srv3.handle(&inboundMsg{FromUserID: "u1", ItemList: []itemElem{
		{Type: 3, ImageItem: &imageItem{URL: "https://cdn/img.png"}},
	}})
	if !strings.Contains(*got3, "内容暂无法读取") {
		t.Fatalf("识别空结果应保留占位提示: %q", *got3)
	}

	// 无 URL：不调用识别器
	srv4, got4, _ := newHandleCaptureServer()
	called := false
	srv4.MediaRecognizer = func(u string) (string, error) { called = true; return "desc", nil }
	srv4.handle(&inboundMsg{FromUserID: "u1", ItemList: []itemElem{
		{Type: 3, ImageItem: &imageItem{Name: "无URL.png"}},
	}})
	if called {
		t.Fatal("URL 为空时不应调用识别器")
	}
	if !strings.Contains(*got4, "图片消息（无URL.png）") {
		t.Fatalf("无 URL 走占位标签: %q", *got4)
	}
}

// ─── v4.8 子项 c：产物回推 seam ─────────────────────────────

// TestSendFileCard_TextFallback 文本降级卡片：产名 + 去向 + caption，经
// Push→Send 发往最近活跃会话（httptest 捕获 sendmessage 负载）；caption
// 为空不追加换行；无活跃会话报错。
func TestSendFileCard_TextFallback(t *testing.T) {
	var mu sync.Mutex
	var lastText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Msg struct {
				ItemList []struct {
					TextItem struct {
						Text string `json:"text"`
					} `json:"text_item"`
				} `json:"item_list"`
			} `json:"msg"`
		}
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		mu.Lock()
		if len(body.Msg.ItemList) > 0 {
			lastText = body.Msg.ItemList[0].TextItem.Text
		}
		mu.Unlock()
		w.WriteHeader(200)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: t.TempDir()}, func(string, string) (string, error) { return "ok", nil })
	// 先产生活跃会话（回推目标）
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "画一张猫"}}}})

	if err := s.SendFileCard(`C:\out\书房绘梦.png`, "一只橘猫"); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
	mu.Lock()
	sent := lastText
	mu.Unlock()
	if !strings.Contains(sent, "🖼 产物已生成：书房绘梦.png") {
		t.Fatalf("卡片应含产物名: %q", sent)
	}
	if !strings.Contains(sent, "书房·绘梦") {
		t.Fatalf("卡片应含去向说明: %q", sent)
	}
	if !strings.Contains(sent, "一只橘猫") {
		t.Fatalf("卡片应含 caption: %q", sent)
	}

	// caption 为空：不追加空行
	if err := s.SendFileCard(`C:\out\a.png`, ""); err != nil {
		t.Fatalf("SendFileCard(空 caption): %v", err)
	}
	mu.Lock()
	sent = lastText
	mu.Unlock()
	if strings.HasSuffix(sent, "\n") {
		t.Fatalf("空 caption 不应追加换行: %q", sent)
	}

	// 无活跃会话：报错（复用 Push 语义）
	srv2 := New(Config{BotToken: "tok", AssistantID: "t", CapturePath: t.TempDir()}, nil)
	if err := srv2.SendFileCard(`C:\out\a.png`, ""); err == nil {
		t.Fatal("无活跃会话时 SendFileCard 应报错")
	}
}
