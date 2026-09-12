package app

// 原罪生成取消（v4.258，Composer「停止」）：中止底层流式请求，已生成的部分
// 照常落库（extra.cancelled=true）——不静默丢内容、不把「已停止」当失败报。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// newSinSlowStreamApp 构造一个「先吐第一段、400ms 后才吐第二段」的假 LLM，
// 供取消测试在中间窗口内打断。
func newSinSlowStreamApp(t *testing.T) *App {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"第一段。\"}}]}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		select {
		case <-time.After(400 * time.Millisecond):
		case <-r.Context().Done():
			return // 客户端已中止：不再吐后续内容（真机取消语义）
		}
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"第二段。\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	c := &core{
		cfg:       &config.Config{Model: "grok-4.20", XaiAPIBaseURL: "http://127.0.0.1:1"},
		engineMgr: modelengine.NewManager("", ""),
		chatStore: chat.NewStore(filepath.Join(t.TempDir(), "chat")),
	}
	t.Cleanup(func() { _ = c.chatStore.Close() })
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Enabled: true, BaseURL: srv.URL,
		Models: []modelengine.ModelInfo{{ID: "qwen3-8b"}},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	a := &App{core: c}
	a.ctx = context.Background()
	a.client = ai.NewClient(a.cfg)
	a.client.SetEngineManager(a.engineMgr)
	if err := a.core.SetFeatureModel("sin", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(sin): %v", err)
	}
	return a
}

func waitSinMessages(t *testing.T, a *App, topicID string, want int, timeout time.Duration) []chat.Message {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		msgs, err := a.SinMessages(topicID)
		if err == nil && len(msgs) >= want {
			return msgs
		}
		time.Sleep(15 * time.Millisecond)
	}
	msgs, _ := a.SinMessages(topicID)
	t.Fatalf("等待 %d 条消息超时，现有 %d 条", want, len(msgs))
	return nil
}

func TestSinCancelStopsStreamingAndKeepsPartial(t *testing.T) {
	a := newSinSlowStreamApp(t)
	story, err := a.SinTopicCreate("慢流")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写开场"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	// 等首帧进入生成（400ms 窗口内取消）
	time.Sleep(150 * time.Millisecond)
	if err := a.SinCancel(story.ID); err != nil {
		t.Fatalf("SinCancel: %v", err)
	}

	msgs := waitSinMessages(t, a, story.ID, 2, 3*time.Second)
	if msgs[0].Role != "user" || msgs[0].Content != "写开场" {
		t.Errorf("用户消息应保留: %+v", msgs[0])
	}
	assistant := msgs[1]
	if !strings.Contains(assistant.Content, "第一段。") {
		t.Errorf("已生成的部分应保留: %q", assistant.Content)
	}
	if strings.Contains(assistant.Content, "第二段。") {
		t.Errorf("取消后不应包含后续内容: %q", assistant.Content)
	}
	if !strings.Contains(assistant.Extra, `"cancelled":true`) {
		t.Errorf("extra 应标记 cancelled: %q", assistant.Extra)
	}
	// 在途流已清：再次取消如实报错（前端据此提示「没有正在生成的故事」）
	if err := a.SinCancel(story.ID); err == nil {
		t.Error("无在途流时 SinCancel 应报错")
	}
}

func TestSinCancelGuards(t *testing.T) {
	a := newSinSlowStreamApp(t)
	if err := a.SinCancel(""); err == nil {
		t.Error("空 topicID 应报错")
	}
	chatTopic, err := a.ChatTopicCreate("闲聊", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	if err := a.SinCancel(chatTopic.ID); err == nil {
		t.Error("非 sin 话题应报错（跨板块守卫）")
	}
}
