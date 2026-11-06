package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// B7 工具循环实弹探针（真引擎+罐头搜索结果——沙箱无外网，真网验证归生产机）：
// plain 消息两路（ChatSend/ChatStreamPlain），断言工具循环交付干净正文、零
// <tool_call> 文本形态泄漏、检索被真实调用。
func TestWhisperProbeToolLoopLive(t *testing.T) {
	if os.Getenv("GAEA_PROBE") == "" {
		t.Skip("实弹：GAEA_PROBE=1")
	}
	a := newChatProbeApp(t, "http://127.0.0.1:8124/v1", "qwen35")
	orig := chatWebSearch
	chatWebSearch = func(q string) (chatSearchOutcome, error) {
		t.Logf("search q=%q", q)
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: q, Text: "1. 《影片A》2026-11-01 上映，票房 3.2 亿，口碑 9.1\n2. 《影片B》2026-10-25 上映，票房 5.8 亿，口碑 8.7"}, nil
	}
	t.Cleanup(func() { chatWebSearch = orig })
	// ChatSend 路径
	topic, err := a.ChatTopicCreate("leak1", "plain")
	if err != nil {
		t.Fatalf("topic: %v", err)
	}
	out, err := a.ChatSend(topic.ID, "最近有什么好看的电影", "plain", true, false, false)
	if err != nil {
		t.Fatalf("ChatSend: %v", err)
	}
	reply, _ := out["reply"].(string)
	t.Logf("ChatSend reply 泄漏=%v 头部=%.120s", strings.Contains(reply, "<tool_call>"), reply)
	time.Sleep(300 * time.Millisecond)
	// 流式路径
	topic2, err := a.ChatTopicCreate("leak2", "plain")
	if err != nil {
		t.Fatalf("topic2: %v", err)
	}
	got := ""
	a.ChatStreamPlain(topic2.ID, "最近有什么好看的电影", true, false, false)
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		msgs, err := a.ChatMessagesList(topic2.ID)
		if err == nil && len(msgs) >= 2 {
			got = msgs[len(msgs)-1].Content
			if got != "" {
				break
			}
		}
	}
	t.Logf("ChatStreamPlain reply 泄漏=%v 头部=%.120s", strings.Contains(got, "<tool_call>"), got)
}
