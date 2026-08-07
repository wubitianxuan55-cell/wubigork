package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/channels/weixin"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	wdb "github.com/gaea/gaea/internal/whisper/db"
)

// newChatServiceTestApp 构造统一聊天入口测试 App：herdsman 指向 mock LLM。
func newChatServiceTestApp(t *testing.T) *App {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"你好呀\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	root := filepath.Join(t.TempDir(), "data")

	c := &core{
		cfg:       &config.Config{Model: "grok-4.20", XaiAPIBaseURL: "http://127.0.0.1:1"},
		engineMgr: modelengine.NewManager("", ""),
		chatStore: chat.NewStore(filepath.Join(root, "chat")),
	}
	t.Cleanup(func() { _ = c.chatStore.Close() })
	t.Cleanup(func() { _ = wdb.CloseDatabase(root) })
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Enabled: true, BaseURL: srv.URL,
		Models: []modelengine.ModelInfo{{ID: "qwen3-8b"}},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	if err := c.SetFeatureModel("chat", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel: %v", err)
	}

	a := &App{core: c}
	a.ctx = context.Background()
	a.client = ai.NewClient(a.cfg)
	a.client.SetEngineManager(a.engineMgr)
	a.whisperState = &whisperState{
		core:            c,
		app:             a,
		whisperDataRoot: root,
		weixinServers:   map[string]*weixin.Server{},
	}
	a.assistantMgr = assistant.NewEmpty(root)
	return a
}

// TestChatSend_Plain 统一入口 plain 模式：回复 + 话题消息落库。
func TestChatSend_Plain(t *testing.T) {
	a := newChatServiceTestApp(t)
	if _, err := a.ChatTopicCreate("闲聊", "plain"); err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	topics := a.ChatTopicsList()
	if len(topics) != 1 {
		t.Fatalf("话题数 = %d, want 1", len(topics))
	}

	out, err := a.ChatSend(topics[0].ID, "你好", "plain")
	if err != nil {
		t.Fatalf("ChatSend: %v", err)
	}
	if out["reply"] != "你好呀" {
		t.Errorf("reply = %v", out["reply"])
	}
	if out["mode"] != "plain" {
		t.Errorf("mode = %v", out["mode"])
	}
	msgs := a.ChatMessagesList(topics[0].ID)
	if len(msgs) != 2 || msgs[0].Role != "user" || msgs[1].Role != "assistant" {
		t.Errorf("消息落库异常: %+v", msgs)
	}
}

// TestChatSend_Persona 统一入口 persona 模式：走轻语 Orchestrator，返回情绪元数据并落库。
func TestChatSend_Persona(t *testing.T) {
	a := newChatServiceTestApp(t)
	if _, err := a.ChatTopicCreate("轻语", "gaea"); err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	topics := a.ChatTopicsList()

	out, err := a.ChatSend(topics[0].ID, "你好", "gaea")
	if err != nil {
		t.Fatalf("ChatSend persona: %v", err)
	}
	if out["reply"] != "你好呀" {
		t.Errorf("reply = %v", out["reply"])
	}
	if _, ok := out["emotion"]; !ok {
		t.Error("persona 模式应返回情绪元数据")
	}
	if _, ok := out["totalTurns"]; !ok {
		t.Error("persona 模式应返回会话轮次")
	}
	// 轻语会话已创建
	if st := a.WhisperGetState("gaea"); st["error"] != nil {
		t.Errorf("Orchestrator 未创建: %v", st["error"])
	}
	msgs := a.ChatMessagesList(topics[0].ID)
	if len(msgs) != 2 {
		t.Errorf("消息落库异常: %d 条", len(msgs))
	}
	// WhisperChat 的异步记忆写入/持久化 goroutine 需要时间落库，等待后清理才能释放 hermes.db
	time.Sleep(500 * time.Millisecond)
}
