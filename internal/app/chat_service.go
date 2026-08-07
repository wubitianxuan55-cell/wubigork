package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/chat"
)

// ChatSend 统一聊天入口（聊天/轻语板块合并，后端）。
// mode 为空或 "plain" → 普通对话；否则按人格 ID 走轻语 Orchestrator（记忆/情绪）。
func (a *App) ChatSend(topicID, message, mode string) (map[string]interface{}, error) {
	if topicID == "" {
		return nil, fmt.Errorf("话题 ID 不能为空")
	}
	if mode == "" || mode == "plain" {
		return a.chatSendPlain(topicID, message)
	}
	return a.chatSendPersona(topicID, message, mode)
}

func (a *App) chatSendPlain(topicID, message string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	eng, model := a.featureModel("chat")
	reply, err := a.client.ChatSimpleStreamWithOptions(a.ctx, model,
		"你是一个热心、博学的AI助手，用中文与用户进行日常对话。", message, ai.ChatSimpleOptions{EngineID: eng})
	if err != nil {
		return nil, err
	}
	a.appendChatExchange(topicID, message, reply, "")
	return map[string]interface{}{"reply": reply, "mode": "plain", "topicID": topicID}, nil
}

func (a *App) chatSendPersona(topicID, message, mode string) (map[string]interface{}, error) {
	out, err := a.WhisperChat(message, mode)
	if err != nil {
		return nil, err
	}
	extra := ""
	if b, err := json.Marshal(map[string]interface{}{
		"emotion": out["emotion"], "trust": out["trust"], "stage": out["stage"], "totalTurns": out["totalTurns"],
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
	_, _ = a.chatStore.AppendMessage(topicID, "user", userMsg, "")
	_, _ = a.chatStore.AppendMessage(topicID, "assistant", reply, extra)
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
	topics, err := a.chatStore.ListTopics()
	if err != nil {
		return chat.Topic{}, err
	}
	for _, t := range topics {
		if t.ID == id {
			return t, nil
		}
	}
	return chat.Topic{}, fmt.Errorf("创建话题后读取失败")
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
	topics, err := a.chatStore.ListTopics()
	if err != nil {
		return chat.Topic{}, err
	}
	for _, t := range topics {
		if t.ID == id {
			return t, nil
		}
	}
	return chat.Topic{}, fmt.Errorf("导入话题后读取失败")
}
