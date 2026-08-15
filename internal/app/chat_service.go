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
// 为待发送的提示词上下文（不污染原始用户消息）。搜索失败/空结果不再静默丢弃
// （T7-2）：记 Warn 日志并注入占位提示，让模型如实说明未获取到实时信息。
func (a *App) preparePlainChatMessage(message string, searchEnabled, forceSearch bool) string {
	if forceSearch || (searchEnabled && shouldSearchWeb(message)) {
		result, err := chatWebSearch(message)
		if err != nil {
			slog.Warn("联网搜索失败，注入占位提示（不阻断对话）", "error", err)
			return searchFallbackMessage(message)
		}
		if result != "" {
			return fmt.Sprintf("%s\n\n[以下是关于此问题的实时搜索结果，请参考这些信息回答]\n%s", message, result)
		}
		slog.Warn("联网搜索返回空结果，注入占位提示")
		return searchFallbackMessage(message)
	}
	return message
}

// searchFallbackMessage 搜索失败/空结果时的占位提示：如实告知模型未获取到
// 实时信息，避免模型假装已搜索或误导用户（T7-2 可见性，不静默吞错）。
func searchFallbackMessage(message string) string {
	return fmt.Sprintf("%s\n\n[以下是关于此问题的实时搜索结果]\n（联网搜索暂不可用，请基于已有知识回答；如不确定请明确说明未获取到实时信息）", message)
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
	// T6-3.2：落库失败必须透传给调用方（前端可见），不再静默吞错。
	if err := a.appendChatExchange(topicID, userMessage, reply, extra); err != nil {
		return nil, err
	}
	return map[string]interface{}{"reply": reply, "reasoning": reasoning, "mode": "plain", "topicID": topicID}, nil
}

// newChatStreamRunID 生成流式 runID（测试可替换为固定值以订阅固定事件名）。
var newChatStreamRunID = func() string { return fmt.Sprintf("cs_%d", time.Now().UnixMilli()) }

// ChatStreamPlain 普通对话真实流式入口：立即返回 runID，前端订阅
// "chat-stream:<runID>" 事件流（delta / reasoning / done / error），完成后落库。
// 联网搜索注入只进入模型上下文，落库仍保留用户原文。
//
// T6-3.1 时序保障：delta 帧在 goroutine 内就绪即发，首帧可能早于前端订阅建立；
// 「订阅先行 + 首帧可重放」由本刀前端（ChatPage 订阅竞态修复）负责，后端不做
// emit 前等待。后端只保证终态纪律：任一失败路径（启动失败/流错误/落库失败/panic）
// 必 emit error，正常完成必 emit done；无帧超时由前端兜底（等待超时按失败展示）。
func (a *App) ChatStreamPlain(topicID, message string, searchEnabled, thinking, forceSearch bool) (string, error) {
	if topicID == "" {
		return "", fmt.Errorf("话题 ID 不能为空")
	}
	if a.client == nil {
		return "", fmt.Errorf("AI 客户端未初始化")
	}
	eng, model := a.featureModel("chat")
	runID := newChatStreamRunID()
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
	// T6-3.2 落库错误透传：delta 流不受影响，仅在最终落库失败时 emit error 终态
	// 而非 done——前端可见失败（消息已生成但未持久化）。
	if err := a.appendChatExchange(topicID, userMessage, replyStr, extra); err != nil {
		slog.Error("chat stream plain 落库失败", "runID", runID, "topicID", topicID, "error", err)
		a.emit("chat-stream:"+runID, map[string]interface{}{
			"type":  "error",
			"error": "回复生成完成但消息保存失败: " + err.Error(),
		})
		return
	}
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
	// T6-3.2：persona 路径同样把落库失败透传给调用方。
	if err := a.appendChatExchange(topicID, message, fmt.Sprint(out["reply"]), extra); err != nil {
		return nil, err
	}
	out["mode"] = mode
	out["topicID"] = topicID
	return out, nil
}

// appendChatExchange 把「用户消息 + 助手消息」原子落库；失败时返回 error
// （调用方决定透传还是转 error 事件），不再静默吞错。
func (a *App) appendChatExchange(topicID, userMsg, reply, extra string) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	if err := a.chatStore.AppendExchange(topicID, userMsg, reply, extra); err != nil {
		slog.Error("chat exchange 落库失败", "topicID", topicID, "error", err)
		return err
	}
	return nil
}

// ── 话题 CRUD（统一会话存储）─────────────────────────────────

// ChatTopicsList 列出全部话题（T6-3.2：读错返回 error，前端可见失败而非空列表）。
func (a *App) ChatTopicsList() ([]chat.Topic, error) {
	if a.chatStore == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	topics, err := a.chatStore.ListTopics()
	if err != nil {
		slog.Error("chat 话题列表读取失败", "error", err)
		return nil, err
	}
	return topics, nil
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

// ChatMessagesList 列出话题全部消息（T6-3.2：读错返回 error，前端可见失败而非空列表）。
func (a *App) ChatMessagesList(topicID string) ([]chat.Message, error) {
	if a.chatStore == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	msgs, err := a.chatStore.ListMessages(topicID)
	if err != nil {
		slog.Error("chat 消息列表读取失败", "topicID", topicID, "error", err)
		return nil, err
	}
	return msgs, nil
}

// ChatAppendMessages 把一组消息追加到已有话题（T6-3.3 语音消息持久化）：
// 单事务批量写入，全部成功或全部回滚；role 仅接受 user/assistant，其余跳过。
// 语音消息目前只在前端内存——前端可在收到语音识别结果后调用本绑定落库。
func (a *App) ChatAppendMessages(topicID string, messages []ChatMessageInput) error {
	if a.chatStore == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	inputs := make([]chat.MessageInput, 0, len(messages))
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		inputs = append(inputs, chat.MessageInput{Role: role, Content: m.Content, Extra: m.Extra})
	}
	return a.chatStore.AppendMessagesTx(topicID, inputs)
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
		// T6-3.5：消息原文写前转义 Markdown 结构敏感字符，避免破坏导出文档
		// （如行首井号生成新标题）；保留换行与加粗/斜体等基础格式。
		b.WriteString(escapeMarkdownContent(m.Content) + "\n\n")
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

// windowsReservedNames Windows 保留设备名（大小写不敏感，含扩展名同样保留）。
var windowsReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// escapeMarkdownContent 转义消息原文中会破坏 Markdown 文档结构的字符
// （行首井号、反引号、尖括号、竖线），保留换行与基础格式（加粗/斜体/列表不受影响）。
func escapeMarkdownContent(s string) string {
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(escapeMarkdownLine(line))
	}
	return b.String()
}

// escapeMarkdownLine 单行转义：行首（可带前导空格）的 # 转义为 \#（避免生成标题），
// 反引号/尖括号/竖线全部转义为 \X（避免代码块/HTML/表格结构被破坏）。
func escapeMarkdownLine(line string) string {
	runes := []rune(line)
	firstNonSpace := 0
	for firstNonSpace < len(runes) && runes[firstNonSpace] == ' ' {
		firstNonSpace++
	}
	var b strings.Builder
	for i, r := range runes {
		switch {
		case i == firstNonSpace && r == '#':
			b.WriteString("\\#")
		case r == '`' || r == '<' || r == '>' || r == '|':
			b.WriteString("\\")
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sanitizeChatFilename 把话题标题规整为安全的文件名片段：
// 非法字符替换为 _、去首尾空白与尾部点号、截断 40 字符、规避 Windows 保留名。
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
	out := strings.Trim(b.String(), " .")
	if out == "" {
		out = "chat"
	}
	runes := []rune(out)
	if len(runes) > 40 {
		out = string(runes[:40])
	}
	base := out
	if i := strings.LastIndexByte(out, '.'); i > 0 {
		base = out[:i]
	}
	if windowsReservedNames[strings.ToUpper(base)] {
		out = "_" + out
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
// T6-3.4：建话题 + 全部消息改走单事务（ImportTopicTx），中途失败整体回滚，
// 不再出现「话题已建但消息只落了一半」的脏状态。
func (a *App) ChatImportTopic(title, mode string, messages []ChatMessageInput) (chat.Topic, error) {
	if a.chatStore == nil {
		return chat.Topic{}, fmt.Errorf("chat store 未初始化")
	}
	id := fmt.Sprintf("t_%d", time.Now().UnixMilli())
	inputs := make([]chat.MessageInput, 0, len(messages))
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		inputs = append(inputs, chat.MessageInput{Role: role, Content: m.Content, Extra: m.Extra})
	}
	if err := a.chatStore.ImportTopicTx(id, title, mode, inputs); err != nil {
		return chat.Topic{}, err
	}
	return a.chatStore.GetTopic(id)
}
