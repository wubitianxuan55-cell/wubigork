package app

// chat_plain_history_test.go — plain 聊天多轮上下文（2026-11-11「下一轮不知道
// 上一轮」根修）测试：历史装载的顺序/重试去重/预算窗口/降级路径。hermetic
//（临时目录 sqlite，不碰真库不拉引擎）。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/chat"
)

func newPlainHistoryApp(t *testing.T) *App {
	t.Helper()
	store := chat.NewStore(t.TempDir())
	t.Cleanup(func() { _ = store.Close() }) // Windows：sqlite 句柄不关 TempDir 清理会炸
	if err := store.CreateTopic("t1", "测试话题", "plain"); err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	return &App{core: &core{chatStore: store}}
}

func TestPlainChatRequestMessages_OrdersHistoryBeforeCurrent(t *testing.T) {
	a := newPlainHistoryApp(t)
	if err := a.chatStore.AppendExchange("t1", "我叫小明", "你好小明", ""); err != nil {
		t.Fatal(err)
	}
	if err := a.chatStore.AppendExchange("t1", "我几岁？", "你刚才没说", ""); err != nil {
		t.Fatal(err)
	}

	msgs := a.plainChatRequestMessages("t1", "我今年 18")
	if msgs[0].Role != "system" {
		t.Fatalf("首条应 system，got %s", msgs[0].Role)
	}
	if msgs[len(msgs)-1].Role != "user" || msgs[len(msgs)-1].Content != "我今年 18" {
		t.Fatalf("末条应为当前 user，got %#v", msgs[len(msgs)-1])
	}
	mid := msgs[1 : len(msgs)-1]
	want := []struct{ role, content string }{
		{"user", "我叫小明"}, {"assistant", "你好小明"},
		{"user", "我几岁？"}, {"assistant", "你刚才没说"},
	}
	if len(mid) != len(want) {
		t.Fatalf("历史条数 = %d, want %d: %#v", len(mid), len(want), mid)
	}
	for i, w := range want {
		if mid[i].Role != w.role || mid[i].Content != w.content {
			t.Fatalf("history[%d] = %s/%q, want %s/%q", i, mid[i].Role, mid[i].Content, w.role, w.content)
		}
	}
}

func TestPlainChatHistory_RetryTrimsTrailingPair(t *testing.T) {
	a := newPlainHistoryApp(t)
	if err := a.chatStore.AppendExchange("t1", "第一问", "第一答", ""); err != nil {
		t.Fatal(err)
	}
	// 重试场景：库尾是同一句 user + 旧回答（后端 AppendExchange 会再追加）
	if err := a.chatStore.AppendExchange("t1", "第一问", "重发前的旧答", ""); err != nil {
		t.Fatal(err)
	}

	hist := a.plainChatHistoryMessages("t1", "第一问")
	// trim-all 语义：尾部同文对（含更早的同文轮）全部裁掉——重试堆叠全清，
	// 本轮请求只带 [system, 当前 user]
	if len(hist) != 0 {
		t.Fatalf("重试去重后历史 = %d 条, want 0（同文对全裁）: %#v", len(hist), hist)
	}
	for _, m := range hist {
		if m.Content == "重发前的旧答" {
			t.Fatalf("旧回答不应残留在重试轮上下文: %#v", hist)
		}
	}
}

func TestPlainChatHistory_BudgetDropsOldest(t *testing.T) {
	a := newPlainHistoryApp(t)
	long := strings.Repeat("甲", 3000) // 3 条长消息 = 9000 字符 > 8000 预算
	for i := 0; i < 3; i++ {
		if err := a.chatStore.AppendExchange("t1", long, "答", ""); err != nil {
			t.Fatal(err)
		}
	}

	hist := a.plainChatHistoryMessages("t1", "新问")
	total := 0
	for _, m := range hist {
		total += len([]rune(m.Content))
	}
	if total > plainChatHistoryMaxChars {
		t.Fatalf("历史总量 %d 超预算 %d", total, plainChatHistoryMaxChars)
	}
	// 最旧的一条被整条丢弃，保留最近的
	if len(hist) == 0 || strings.Contains(hist[0].Content, strings.Repeat("甲", 3000)) && len(hist) == 6 {
		t.Fatalf("应从最旧侧丢弃: %d 条", len(hist))
	}
}

func TestPlainChatHistory_EmptyTopicAndNoStore(t *testing.T) {
	a := newPlainHistoryApp(t)
	if hist := a.plainChatHistoryMessages("fresh", "你好"); hist != nil {
		t.Fatalf("空话题历史应为 nil, got %#v", hist)
	}

	var noStore *App = &App{}
	if hist := noStore.plainChatHistoryMessages("t1", "你好"); hist != nil {
		t.Fatalf("无 store 应降级 nil, got %#v", hist)
	}
	msgs := noStore.plainChatRequestMessages("t1", "你好")
	if len(msgs) != 2 || msgs[0].Role != "system" || msgs[1].Role != "user" {
		t.Fatalf("降级应退回单轮 [system,user]: %#v", msgs)
	}
}
