package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/whisper"
)

// chatPlainSystemPrompt 普通对话（聊天板块 plain 模式）的系统提示词。
const chatPlainSystemPrompt = "你是一个热心、博学的AI助手，用中文与用户进行日常对话。你具备联网搜索能力：用户询问需要实时/最新信息的问题时，系统会把搜索结果以「以下是关于此问题的实时搜索结果」注入到消息中，请优先依据这些搜索结果作答，并如实告诉用户信息来源；不要说自己无法联网搜索。"

// chatWebSearch 可注入的联网搜索函数：生产走 whisper.WebSearch，测试可替换避免真实网络请求。
var chatWebSearch = whisper.WebSearch

// ChatSend 统一聊天入口（聊天/轻语板块合并，后端）。
// mode 为空或 "plain" → 普通对话；否则按人格 ID 走轻语 Orchestrator（记忆/情绪）。
// searchEnabled 打开时自动检测搜索意图并注入实时搜索结果（普通对话同样生效）。
// thinking 打开时对本地 Qwen3 系模型开启思考模式，回复附带思考链。
func (a *App) ChatSend(topicID, message, mode string, searchEnabled, thinking, forceSearch bool) (map[string]interface{}, error) {
	if topicID == "" {
		return nil, fmt.Errorf("话题 ID 不能为空")
	}
	if mode == "" || mode == "plain" {
		return a.chatSendPlain(topicID, message, searchEnabled, thinking, forceSearch)
	}
	return a.chatSendPersona(topicID, message, mode, searchEnabled, thinking, forceSearch)
}

// preparePlainChatMessage 联网搜索增强：命中搜索意图/强制搜索时，把实时结果注入
// 为待发送的提示词上下文（不污染原始用户消息）。
func (a *App) preparePlainChatMessage(message string, searchEnabled, forceSearch bool) string {
	if forceSearch || (searchEnabled && shouldSearchWeb(message)) {
		if result, err := chatWebSearch(message); err == nil && result != "" {
			return fmt.Sprintf("%s\n\n[以下是关于此问题的实时搜索结果，请参考这些信息回答]\n%s", message, result)
		}
	}
	return message
}

func (a *App) chatSendPlain(topicID, message string, searchEnabled, thinking, forceSearch bool) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	userMessage := message
	promptMessage := a.preparePlainChatMessage(message, searchEnabled, forceSearch)
	eng, model := a.featureModel("chat")
	reply, reasoning, err := a.client.ChatSimpleStreamDetailed(a.ctx, model,
		chatPlainSystemPrompt, promptMessage, ai.ChatSimpleOptions{EngineID: eng, EnableThinking: thinking})
	if err != nil {
		return nil, err
	}
	extra := ""
	if reasoning != "" {
		if b, err := json.Marshal(map[string]interface{}{"reasoning": reasoning}); err == nil {
			extra = string(b)
		}
	}
	a.appendChatExchange(topicID, userMessage, reply, extra)
	return map[string]interface{}{"reply": reply, "reasoning": reasoning, "mode": "plain", "topicID": topicID}, nil
}

// ChatStreamPlain 普通对话真实流式入口：立即返回 runID，前端订阅
// "chat-stream:<runID>" 事件流（delta / reasoning / done / error），完成后落库。
// 联网搜索注入只进入模型上下文，落库仍保留用户原文。
func (a *App) ChatStreamPlain(topicID, message string, searchEnabled, thinking, forceSearch bool) (string, error) {
	if topicID == "" {
		return "", fmt.Errorf("话题 ID 不能为空")
	}
	if a.client == nil {
		return "", fmt.Errorf("AI 客户端未初始化")
	}
	eng, model := a.featureModel("chat")
	runID := fmt.Sprintf("cs_%d", time.Now().UnixMilli())
	go a.runChatStreamPlain(runID, topicID, message, eng, model, searchEnabled, thinking, forceSearch)
	return runID, nil
}

func (a *App) runChatStreamPlain(runID, topicID, userMessage, eng, model string, searchEnabled, thinking, forceSearch bool) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("chat stream plain panic", "panic", r)
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "error", "error": "流式生成异常"})
		}
	}()

	promptMessage := a.preparePlainChatMessage(userMessage, searchEnabled, forceSearch)
	chunks, cancel, err := a.client.ChatStreamChunks(a.ctx, model, chatPlainSystemPrompt, promptMessage,
		ai.ChatSimpleOptions{EngineID: eng, EnableThinking: thinking})
	if err != nil {
		a.emit("chat-stream:"+runID, map[string]interface{}{"type": "error", "error": err.Error()})
		return
	}
	defer cancel()

	var reply strings.Builder
	var reasoning strings.Builder
	for chunk := range chunks {
		if chunk.Error != "" {
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "error", "error": chunk.Error})
			return
		}
		if chunk.Done {
			break
		}
		if chunk.Content != "" {
			reply.WriteString(chunk.Content)
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "delta", "content": chunk.Content})
		}
		if chunk.Reasoning != "" {
			reasoning.WriteString(chunk.Reasoning)
			a.emit("chat-stream:"+runID, map[string]interface{}{"type": "reasoning", "content": chunk.Reasoning})
		}
	}

	replyStr := reply.String()
	reasoningStr := reasoning.String()
	extra := ""
	if reasoningStr != "" {
		if b, err := json.Marshal(map[string]interface{}{"reasoning": reasoningStr}); err == nil {
			extra = string(b)
		}
	}
	a.appendChatExchange(topicID, userMessage, replyStr, extra)
	a.emit("chat-stream:"+runID, map[string]interface{}{
		"type":      "done",
		"reply":     replyStr,
		"reasoning": reasoningStr,
		"topicID":   topicID,
	})
}

func (a *App) chatSendPersona(topicID, message, mode string, searchEnabled, thinking, forceSearch bool) (map[string]interface{}, error) {
	var out map[string]interface{}
	var err error
	if forceSearch || searchEnabled {
		out, err = a.WhisperChatWithSearch(message, mode, thinking, forceSearch)
	} else {
		out, err = a.WhisperChat(message, mode, thinking)
	}
	if err != nil {
		return nil, err
	}
	extra := ""
	if b, err := json.Marshal(map[string]interface{}{
		"emotion": out["emotion"], "trust": out["trust"], "stage": out["stage"], "totalTurns": out["totalTurns"],
		"reasoning": out["reasoning"],
	}); err == nil {
		extra = string(b)
	}
	a.appendChatExchange(topicID, message, fmt.Sprint(out["reply"]), extra)
	out["mode"] = mode
	out["topicID"] = topicID
	return out, nil
}

func (a *App) appendChatExchange(topicID, userMsg, reply, extra string) {
	if a.chatStore == nil {
		return
	}
	if err := a.chatStore.AppendExchange(topicID, userMsg, reply, extra); err != nil {
		slog.Error("chat exchange 落库失败", "topicID", topicID, "error", err)
	}
}

// ── 话题 CRUD（统一会话存储）─────────────────────────────────

func (a *App) ChatTopicsList() []chat.Topic {
	if a.chatStore == nil {
		return nil
	}
	topics, _ := a.chatStore.ListTopics()
	return topics
}

func (a *App) ChatTopicCreate(title, mode string) (chat.Topic, error) {
	if a.chatStore == nil {
		return chat.Topic{}, fmt.Errorf("chat store 未初始化")
	}
	id := fmt.Sprintf("t_%d", time.Now().UnixMilli())
	if err := a.chatStore.CreateTopic(id, title, mode); err != nil {
		return chat.Topic{}, err
	}
	return a.chatStore.GetTopic(id)
}

func (a *App) ChatTopicRename(id, title string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	return a.chatStore.RenameTopic(id, title)
}

// ChatTopicSetMode 切换话题模式（plain ↔ personaID），前端模式切换器使用。
func (a *App) ChatTopicSetMode(id, mode string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	return a.chatStore.SetMode(id, mode)
}

// ChatTopicClear 清空话题消息（前端「清空当前对话」）。
func (a *App) ChatTopicClear(id string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	return a.chatStore.ClearMessages(id)
}

func (a *App) ChatTopicDelete(id string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	return a.chatStore.DeleteTopic(id)
}

func (a *App) ChatMessagesList(topicID string) []chat.Message {
	if a.chatStore == nil {
		return nil
	}
	msgs, _ := a.chatStore.ListMessages(topicID)
	return msgs
}

// ChatTopicExportMarkdown 把话题全部消息导出为 Markdown 文件，写到用户数据目录
// 的 exports/chat 下，返回绝对路径（前端可展示/复制路径）。
func (a *App) ChatTopicExportMarkdown(topicID string) (string, error) {
	if a.chatStore == nil {
		return "", fmt.Errorf("chat store 未初始化")
	}
	topic, err := a.chatStore.GetTopic(topicID)
	if err != nil {
		return "", err
	}
	title := topic.Title
	msgs, err := a.chatStore.ListMessages(topicID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	b.WriteString("> 导出时间：" + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
	for _, m := range msgs {
		role := "用户"
		if m.Role == "assistant" {
			role = "AI"
		}
		b.WriteString("## " + role + "\n\n")
		b.WriteString(m.Content + "\n\n")
	}

	dir := filepath.Join(a.whisperDataRoot, "exports", "chat")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s_%d.md", sanitizeChatFilename(title), time.Now().UnixMilli())
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// sanitizeChatFilename 把话题标题规整为安全的文件名片段。
func sanitizeChatFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' || r < 32:
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		out = "chat"
	}
	runes := []rune(out)
	if len(runes) > 40 {
		out = string(runes[:40])
	}
	return out
}

// ChatMessageInput 历史消息导入输入（旧 localStorage 话题迁入 chat.db）。
type ChatMessageInput struct {
	Role    string
	Content string
	Extra   string
}

// ChatImportTopic 创建话题并按序导入历史消息（迁移旧 localStorage 会话，不调用 AI）。
func (a *App) ChatImportTopic(title, mode string, messages []ChatMessageInput) (chat.Topic, error) {
	if a.chatStore == nil {
		return chat.Topic{}, fmt.Errorf("chat store 未初始化")
	}
	id := fmt.Sprintf("t_%d", time.Now().UnixMilli())
	if err := a.chatStore.CreateTopic(id, title, mode); err != nil {
		return chat.Topic{}, err
	}
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		if _, err := a.chatStore.AppendMessage(id, role, m.Content, m.Extra); err != nil {
			return chat.Topic{}, err
		}
	}
	return a.chatStore.GetTopic(id)
}
