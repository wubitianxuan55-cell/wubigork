package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/gaea/gaea/internal/httpbridge"
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
	topics, err := a.ChatTopicsList()
	if err != nil {
		t.Fatalf("ChatTopicsList: %v", err)
	}
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
	msgs, err := a.ChatMessagesList(topics[0].ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
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
	msgs, err := a.ChatMessagesList(topic.ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
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
		var err error
		msgs, err = a.ChatMessagesList(topic.ID)
		if err == nil && len(msgs) == 2 {
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
	topics, err := a.ChatTopicsList()
	if err != nil {
		t.Fatalf("ChatTopicsList: %v", err)
	}

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
	msgs, err := a.ChatMessagesList(topics[0].ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
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
	ms, err := a.ChatMessagesList(topic.ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
	if len(ms) != 3 {
		t.Fatalf("消息数 = %d, want 3", len(ms))
	}
	for i, m := range ms {
		if m.Role != msgs[i].Role || m.Content != msgs[i].Content {
			t.Errorf("消息[%d] = %+v, want role=%q content=%q", i, m, msgs[i].Role, msgs[i].Content)
		}
	}
	topics, err := a.ChatTopicsList()
	if err != nil {
		t.Fatalf("ChatTopicsList: %v", err)
	}
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
	ms, err := a.ChatMessagesList(topic.ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
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
	topics, err := a.ChatTopicsList()
	if err != nil {
		t.Fatalf("ChatTopicsList: %v", err)
	}
	if topics[0].Mode != "gaea" {
		t.Errorf("mode = %q, want gaea", topics[0].Mode)
	}
	if err := a.ChatTopicSetMode(topic.ID, "plain"); err != nil {
		t.Fatalf("ChatTopicSetMode -> plain: %v", err)
	}
	topics, err = a.ChatTopicsList()
	if err != nil {
		t.Fatalf("ChatTopicsList: %v", err)
	}
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
	ms, err := a.ChatMessagesList(topic.ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
	if len(ms) != 0 {
		t.Fatalf("清空后消息数 = %d, want 0", len(ms))
	}
	topics, err := a.ChatTopicsList()
	if err != nil {
		t.Fatalf("ChatTopicsList: %v", err)
	}
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

// ── T6-3 新增用例：落库错误透传 / 语音落库 / 导出转义 / 文件名规整 ──

// TestChatSend_Plain_PersistError 落库失败（存储已关闭）时 ChatSend 必须把错误
// 透传给调用方（T6-3.2），前端可见失败而非假装成功。
func TestChatSend_Plain_PersistError(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatTopicCreate("落库失败", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	if err := a.chatStore.Close(); err != nil {
		t.Fatalf("关闭 chatStore: %v", err)
	}
	if _, err := a.ChatSend(topic.ID, "你好", "plain", false, false, false); err == nil {
		t.Fatal("落库失败时 ChatSend 应返回错误")
	}
}

// TestChatStreamPlain_PersistFailureEmitsError 流式路径落库失败必须 emit error
// 终态而非 done（T6-3.2）：消息已生成但未持久化，前端应看到失败。
func TestChatStreamPlain_PersistFailureEmitsError(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatTopicCreate("流式落库失败", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}

	orig := newChatStreamRunID
	newChatStreamRunID = func() string { return "cs_persist_fail" }
	t.Cleanup(func() { newChatStreamRunID = orig })

	// 通过 httpbridge SSE 订阅固定事件名，捕获 emit 内容。
	srv := httptest.NewServer(httpbridge.New(a).Handler())
	defer srv.Close()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/stream?id=chat-stream:cs_persist_fail", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("打开 SSE: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE status = %d", resp.StatusCode)
	}

	// 独立 goroutine 逐行读取，主流程用 select 超时兜底（SSE 有 15s keep-alive）。
	lines := make(chan string, 64)
	go func() {
		br := bufio.NewReader(resp.Body)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				close(lines)
				return
			}
			lines <- line
		}
	}()
	// 消费连接帧（event/data/blank 三行）。
	for i := 0; i < 3; i++ {
		select {
		case <-lines:
		case <-time.After(3 * time.Second):
			t.Fatal("等待 SSE 连接帧超时")
		}
	}

	// 关闭存储 → 流式落库必然失败。
	if err := a.chatStore.Close(); err != nil {
		t.Fatalf("关闭 chatStore: %v", err)
	}
	if _, err := a.ChatStreamPlain(topic.ID, "你好", false, false, false); err != nil {
		t.Fatalf("ChatStreamPlain: %v", err)
	}

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("流在收到 error 终态前关闭")
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var ev map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
				t.Fatalf("解析事件: %v", err)
			}
			switch ev["type"] {
			case "error":
				return // 期望的终态
			case "done":
				t.Fatal("落库失败不应 emit done")
			default:
				// delta / reasoning 帧继续等待 error
			}
		case <-time.After(3 * time.Second):
			t.Fatal("未在 3s 内收到 error 事件")
		}
	}
}

// TestChatAppendMessages 语音消息持久化：批量追加（user/assistant）落库，
// 非法角色跳过（T6-3.3）。
func TestChatAppendMessages(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatTopicCreate("语音", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	if err := a.ChatAppendMessages(topic.ID, []ChatMessageInput{
		{Role: "user", Content: "语音转写：今天天气如何"},
		{Role: "assistant", Content: "今天晴，20 度。", Extra: "{\"kind\":\"voice\"}"},
		{Role: "system", Content: "应跳过"},
	}); err != nil {
		t.Fatalf("ChatAppendMessages: %v", err)
	}
	msgs, err := a.ChatMessagesList(topic.ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("消息数 = %d, want 2（system 应跳过）", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "语音转写：今天天气如何" {
		t.Errorf("user 消息异常: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Extra != "{\"kind\":\"voice\"}" {
		t.Errorf("assistant 消息异常: %+v", msgs[1])
	}
}

// TestChatAppendMessages_UnknownTopic 向不存在的话题追加应在单事务内整体失败。
func TestChatAppendMessages_UnknownTopic(t *testing.T) {
	a := newChatServiceTestApp(t)
	if err := a.ChatAppendMessages("nope", []ChatMessageInput{
		{Role: "user", Content: "a"},
		{Role: "assistant", Content: "b"},
	}); err == nil {
		t.Fatal("向不存在的话题追加应报错")
	}
	msgs, err := a.ChatMessagesList("nope")
	if err == nil && len(msgs) != 0 {
		t.Errorf("失败事务不应残留消息: %+v", msgs)
	}
}

// TestChatTopicExportMarkdown_EscapesMarkdown 导出转义（T6-3.5）：行首井号、
// 反引号、尖括号、竖线在导出文件中必须被转义，且不产生新标题。
func TestChatTopicExportMarkdown_EscapesMarkdown(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatImportTopic("转义", "plain", []ChatMessageInput{
		{Role: "user", Content: "# 这不是标题\n## 也不是\n`inline` <b> a|b"},
	})
	if err != nil {
		t.Fatalf("ChatImportTopic: %v", err)
	}
	path, err := a.ChatTopicExportMarkdown(topic.ID)
	if err != nil {
		t.Fatalf("ChatTopicExportMarkdown: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取导出文件: %v", err)
	}
	s := string(b)
	for _, want := range []string{
		"\\# 这不是标题",
		"\\## 也不是",
		"\\`inline\\`",
		"\\<b\\>",
		"a\\|b",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("导出内容缺少转义片段 %q:\n%s", want, s)
		}
	}
	// 行首井号被转义 → 导出文档中不应出现独立的 "# 这不是标题" 标题行。
	if strings.Contains(s, "\\n# 这不是标题") || strings.Contains(s, "\\n## 也不是") {
		t.Errorf("行首井号未转义，生成了新标题:\n%s", s)
	}
}

// TestSanitizeChatFilename Windows 文件名规整：非法字符 / 保留名 / 长度 / 空值。
func TestSanitizeChatFilename(t *testing.T) {
	cases := []struct{ in, want string }{
		{"普通标题", "普通标题"},
		{"a/b\\c:d*e?f\"g<h>i|j", "a_b_c_d_e_f_g_h_i_j"},
		{"CON", "_CON"},         // Windows 保留设备名
		{"con.txt", "_con.txt"}, // 保留名带扩展名
		{"NUL", "_NUL"},
		{"nul.log", "_nul.log"},
		{"COM1", "_COM1"},
		{"lpt9", "_lpt9"},
		{"  标题  ", "标题"},
		{"标题.", "标题"}, // 尾部点号非法 → 去掉
		{"", "chat"},  // 空 → 默认名
		{"...", "chat"},
		{strings.Repeat("长", 50), strings.Repeat("长", 40)}, // 截断 40 字符
	}
	for _, c := range cases {
		if got := sanitizeChatFilename(c.in); got != c.want {
			t.Errorf("sanitizeChatFilename(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── T7-2 可见性收口：联网搜索失败占位/Notice（不静默丢弃） ──

// TestPreparePlainChatMessage_SearchErrorInjectsPlaceholder 搜索函数返回错误时，
// 注入占位提示（含失败说明），原始消息保留——失败可见而非静默丢弃。
func TestPreparePlainChatMessage_SearchErrorInjectsPlaceholder(t *testing.T) {
	a := &App{}
	orig := chatWebSearch
	chatWebSearch = func(string) (string, error) { return "", errors.New("搜索服务不可用") }
	t.Cleanup(func() { chatWebSearch = orig })

	got := a.preparePlainChatMessage("今天天气如何", false, true)
	if !strings.Contains(got, "今天天气如何") {
		t.Errorf("占位消息应保留原始消息: %q", got)
	}
	if !strings.Contains(got, "联网搜索暂不可用") {
		t.Errorf("搜索失败应注入占位提示: %q", got)
	}
	if strings.Contains(got, "实时搜索结果，请参考") {
		t.Errorf("失败时不应注入伪造的成功结果: %q", got)
	}
}

// TestPreparePlainChatMessage_EmptyResultInjectsPlaceholder 搜索返回空结果（无错误）
// 同样注入占位提示，避免模型把空搜索当成实时信息。
func TestPreparePlainChatMessage_EmptyResultInjectsPlaceholder(t *testing.T) {
	a := &App{}
	orig := chatWebSearch
	chatWebSearch = func(string) (string, error) { return "", nil }
	t.Cleanup(func() { chatWebSearch = orig })

	got := a.preparePlainChatMessage("今天天气如何", false, true)
	if !strings.Contains(got, "联网搜索暂不可用") {
		t.Errorf("空结果应注入占位提示: %q", got)
	}
}

// TestPreparePlainChatMessage_NoSearchKeepsMessage 未开启搜索且未强制时，
// 消息原样返回，不注入任何搜索内容。
func TestPreparePlainChatMessage_NoSearchKeepsMessage(t *testing.T) {
	a := &App{}
	orig := chatWebSearch
	chatWebSearch = func(string) (string, error) { return "【模拟搜索结果】", nil }
	t.Cleanup(func() { chatWebSearch = orig })

	got := a.preparePlainChatMessage("普通闲聊", false, false)
	if got != "普通闲聊" {
		t.Errorf("未开启搜索应原样返回: %q", got)
	}
}

// TestChatSend_Plain_SearchErrorStillSucceeds 搜索失败不阻断对话：ChatSend 仍
// 正常返回回复并落库（占位注入让模型如实说明，而不是整条对话报错）。
func TestChatSend_Plain_SearchErrorStillSucceeds(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatTopicCreate("搜索失败", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	orig := chatWebSearch
	chatWebSearch = func(string) (string, error) { return "", errors.New("网络不可达") }
	t.Cleanup(func() { chatWebSearch = orig })

	out, err := a.ChatSend(topic.ID, "有什么新闻", "plain", true, false, true)
	if err != nil {
		t.Fatalf("搜索失败不应导致 ChatSend 报错: %v", err)
	}
	if out["reply"] != "你好呀" {
		t.Errorf("reply = %v, want 你好呀（对话正常继续）", out["reply"])
	}
	msgs, err := a.ChatMessagesList(topic.ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
	if len(msgs) != 2 || msgs[0].Content != "有什么新闻" {
		t.Errorf("消息落库应保留用户原文: %+v", msgs)
	}
}
