package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

	out, err := a.ChatSend(topics[0].ID, "你好", "plain", false, false, false)
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

// TestChatSend_Plain_SearchKeepsOriginalUserMessage 联网搜索注入只应进入模型上下文，
// 落库的用户消息必须保留原文（避免搜索结果污染历史与话题预览）。
func TestChatSend_Plain_SearchKeepsOriginalUserMessage(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatTopicCreate("搜索", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}

	orig := chatWebSearch
	chatWebSearch = func(query string) (string, error) { return "【模拟搜索结果】", nil }
	t.Cleanup(func() { chatWebSearch = orig })

	if _, err := a.ChatSend(topic.ID, "今天有什么新闻", "plain", false, false, true); err != nil {
		t.Fatalf("ChatSend(forceSearch): %v", err)
	}
	msgs := a.ChatMessagesList(topic.ID)
	if len(msgs) != 2 {
		t.Fatalf("消息数 = %d, want 2", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "今天有什么新闻" {
		t.Errorf("落库用户消息应保留原文，实际 role=%q content=%q", msgs[0].Role, msgs[0].Content)
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "你好呀" {
		t.Errorf("assistant 消息异常: %+v", msgs[1])
	}
}

// TestChatStreamPlain_StreamsAndPersists 普通对话真实流式：返回 runID，异步下发分块并落库。
func TestChatStreamPlain_StreamsAndPersists(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatTopicCreate("流式", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}

	runID, err := a.ChatStreamPlain(topic.ID, "你好", false, false, false)
	if err != nil {
		t.Fatalf("ChatStreamPlain: %v", err)
	}
	if runID == "" {
		t.Fatal("runID 不应为空")
	}

	deadline := time.Now().Add(3 * time.Second)
	var msgs []chat.Message
	for time.Now().Before(deadline) {
		msgs = a.ChatMessagesList(topic.ID)
		if len(msgs) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(msgs) != 2 {
		t.Fatalf("消息数 = %d, want 2", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "你好" {
		t.Errorf("user 消息 = %+v, want 你好", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "你好呀" {
		t.Errorf("assistant 消息 = %+v, want 你好呀", msgs[1])
	}
}

// TestChatSend_Persona 统一入口 persona 模式：走轻语 Orchestrator，返回情绪元数据并落库。
func TestChatSend_Persona(t *testing.T) {
	a := newChatServiceTestApp(t)
	if _, err := a.ChatTopicCreate("轻语", "gaea"); err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	topics := a.ChatTopicsList()

	out, err := a.ChatSend(topics[0].ID, "你好", "gaea", false, false, false)
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

// TestChatImportTopic 旧 localStorage 话题导入：模式保留 + 消息按序落库 + 首条预览。
func TestChatImportTopic(t *testing.T) {
	a := newChatServiceTestApp(t)
	msgs := []ChatMessageInput{
		{Role: "user", Content: "还记得我喜欢喝什么吗"},
		{Role: "assistant", Content: "记得，你喜欢喝热可可"},
		{Role: "user", Content: "太好了"},
	}
	topic, err := a.ChatImportTopic("旧轻语会话", "gaea", msgs)
	if err != nil {
		t.Fatalf("ChatImportTopic: %v", err)
	}
	if topic.Mode != "gaea" {
		t.Errorf("mode = %q, want gaea", topic.Mode)
	}
	ms := a.ChatMessagesList(topic.ID)
	if len(ms) != 3 {
		t.Fatalf("消息数 = %d, want 3", len(ms))
	}
	for i, m := range ms {
		if m.Role != msgs[i].Role || m.Content != msgs[i].Content {
			t.Errorf("消息[%d] = %+v, want role=%q content=%q", i, m, msgs[i].Role, msgs[i].Content)
		}
	}
	topics := a.ChatTopicsList()
	if len(topics) != 1 {
		t.Fatalf("话题数 = %d, want 1", len(topics))
	}
	if topics[0].Preview != "还记得我喜欢喝什么吗" {
		t.Errorf("preview = %q, want 首条消息内容", topics[0].Preview)
	}
}

// TestChatImportTopic_SkipsBadRoles 非法角色消息应被跳过，不影响导入。
func TestChatImportTopic_SkipsBadRoles(t *testing.T) {
	a := newChatServiceTestApp(t)
	msgs := []ChatMessageInput{
		{Role: "user", Content: "你好"},
		{Role: "system", Content: "不应写入"},
		{Role: "assistant", Content: "你好呀"},
	}
	topic, err := a.ChatImportTopic("跳过非法角色", "plain", msgs)
	if err != nil {
		t.Fatalf("ChatImportTopic: %v", err)
	}
	ms := a.ChatMessagesList(topic.ID)
	if len(ms) != 2 {
		t.Fatalf("消息数 = %d, want 2（system 应被跳过）", len(ms))
	}
}

// TestChatTopicSetMode 话题模式切换：plain ↔ persona 可来回切换并持久化。
func TestChatTopicSetMode(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatTopicCreate("模式切换", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	if err := a.ChatTopicSetMode(topic.ID, "gaea"); err != nil {
		t.Fatalf("ChatTopicSetMode -> gaea: %v", err)
	}
	topics := a.ChatTopicsList()
	if topics[0].Mode != "gaea" {
		t.Errorf("mode = %q, want gaea", topics[0].Mode)
	}
	if err := a.ChatTopicSetMode(topic.ID, "plain"); err != nil {
		t.Fatalf("ChatTopicSetMode -> plain: %v", err)
	}
	topics = a.ChatTopicsList()
	if topics[0].Mode != "plain" {
		t.Errorf("mode = %q, want plain", topics[0].Mode)
	}
}

// TestChatTopicClear 清空话题消息：删除全部消息但保留话题本身。
func TestChatTopicClear(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatImportTopic("清空测试", "plain", []ChatMessageInput{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "你好呀"},
	})
	if err != nil {
		t.Fatalf("ChatImportTopic: %v", err)
	}
	if err := a.ChatTopicClear(topic.ID); err != nil {
		t.Fatalf("ChatTopicClear: %v", err)
	}
	if ms := a.ChatMessagesList(topic.ID); len(ms) != 0 {
		t.Fatalf("清空后消息数 = %d, want 0", len(ms))
	}
	topics := a.ChatTopicsList()
	if len(topics) != 1 || topics[0].ID != topic.ID {
		t.Fatalf("话题应保留: %+v", topics)
	}
}

// TestChatTopicExportMarkdown 导出为 Markdown：含标题、用户/AI 分段与消息原文。
func TestChatTopicExportMarkdown(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatImportTopic("导出:测试/会话", "plain", []ChatMessageInput{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "你好呀"},
	})
	if err != nil {
		t.Fatalf("ChatImportTopic: %v", err)
	}
	path, err := a.ChatTopicExportMarkdown(topic.ID)
	if err != nil {
		t.Fatalf("ChatTopicExportMarkdown: %v", err)
	}
	if !strings.HasSuffix(path, ".md") {
		t.Fatalf("导出文件应为 .md: %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取导出文件: %v", err)
	}
	s := string(b)
	for _, want := range []string{"# 导出:测试/会话", "## 用户", "## AI", "你好", "你好呀"} {
		if !strings.Contains(s, want) {
			t.Errorf("导出内容缺少 %q:\n%s", want, s)
		}
	}
}
