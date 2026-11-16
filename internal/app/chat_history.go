package app

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/chat"
)

// chat_history.go —— E 档聊天历史 ZIP 导入导出（chat-demo backup.ts 语义对齐；
// 落库侧单事务原子替换见 internal/chat/history.go）。
//
// 格式 gaea_chat_history v1：ZIP 三条目
//   manifest.json   {"format","version","exported_at"}——格式与版本判定首条目
//   topics.json     全部话题（含分支树元数据所在的消息 extra 原样保留）
//   messages.json   全部消息（id/seq/extra 原样——bt 分支字段与 answered_by 保真）
//
// 导入前自动落一份 pre-import 备份（失败仅日志不阻塞导入）；导入单事务，
// 校验或写库失败原数据原样。

const (
	chatHistoryFormat        = "gaea_chat_history"
	chatHistoryFormatVersion = 1
	// messages.json 单条目读取上限（防恶意超大解压；正常历史远小于此）
	chatHistoryMaxEntryBytes = 512 << 20
)

type chatHistoryManifest struct {
	Format     string `json:"format"`
	Version    int    `json:"version"`
	ExportedAt string `json:"exported_at"`
}

// ChatExportHistory 全量导出聊天历史为 ZIP，返回落盘路径。
// 落点与单话题 Markdown 导出同目录（exports/chat）。
func (a *App) ChatExportHistory() (string, error) {
	if a.chatStore == nil {
		return "", fmt.Errorf("chat store 未初始化")
	}
	topics, err := a.chatStore.ListTopics()
	if err != nil {
		return "", err
	}
	allMsgs := make([]chat.Message, 0, 1024)
	for _, t := range topics {
		msgs, err := a.chatStore.ListMessages(t.ID)
		if err != nil {
			return "", fmt.Errorf("读取话题 %s 消息: %w", t.ID, err)
		}
		allMsgs = append(allMsgs, msgs...)
	}
	dir := filepath.Join(a.whisperDataRoot, "exports", "chat")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("gaea_chat_history_%d.zip", time.Now().UnixMilli()))
	if err := writeChatHistoryZip(path, topics, allMsgs); err != nil {
		_ = os.Remove(path) // 半截产物不留盘
		return "", err
	}
	slog.Info("chat history exported", "path", path, "topics", len(topics), "messages", len(allMsgs))
	return path, nil
}

func writeChatHistoryZip(path string, topics []chat.Topic, msgs []chat.Message) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := zip.NewWriter(f)
	writeEntry := func(name string, v interface{}) error {
		fw, err := w.Create(name)
		if err != nil {
			return err
		}
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = fw.Write(b)
		return err
	}
	manifest := chatHistoryManifest{
		Format:     chatHistoryFormat,
		Version:    chatHistoryFormatVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeEntry("manifest.json", manifest); err != nil {
		return err
	}
	if err := writeEntry("topics.json", topics); err != nil {
		return err
	}
	if err := writeEntry("messages.json", msgs); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return f.Close()
}

// ChatImportHistory 从 ZIP 导入并原子替换全部本地历史，返回导入话题数。
func (a *App) ChatImportHistory(srcPath string) (int, error) {
	if a.chatStore == nil {
		return 0, fmt.Errorf("chat store 未初始化")
	}
	topics, msgs, err := readChatHistoryZip(srcPath)
	if err != nil {
		return 0, err
	}
	// 导入前自动备份（失败仅日志——备份不可用不应阻塞用户显式导入意图）
	if backupPath, backupErr := a.ChatExportHistory(); backupErr != nil {
		slog.Warn("chat history pre-import backup failed", "error", backupErr)
	} else {
		slog.Info("chat history pre-import backup", "path", backupPath)
	}
	if err := a.chatStore.ReplaceAllTx(topics, msgs); err != nil {
		return 0, fmt.Errorf("导入替换失败（原数据未动）: %w", err)
	}
	slog.Info("chat history imported", "path", srcPath, "topics", len(topics), "messages", len(msgs))
	return len(topics), nil
}

func readChatHistoryZip(srcPath string) ([]chat.Topic, []chat.Message, error) {
	zr, err := zip.OpenReader(srcPath)
	if err != nil {
		return nil, nil, fmt.Errorf("不是有效的历史导出包: %w", err)
	}
	defer func() { _ = zr.Close() }()
	readEntry := func(name string) ([]byte, bool, error) {
		for _, f := range zr.File {
			if f.Name != name {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, true, err
			}
			b, err := io.ReadAll(io.LimitReader(rc, chatHistoryMaxEntryBytes))
			_ = rc.Close()
			return b, true, err
		}
		return nil, false, nil
	}
	var manifest chatHistoryManifest
	{
		b, found, err := readEntry("manifest.json")
		if err != nil {
			return nil, nil, err
		}
		if !found {
			return nil, nil, fmt.Errorf("历史包缺 manifest.json（不是 gaea 聊天导出格式）")
		}
		if err := json.Unmarshal(b, &manifest); err != nil {
			return nil, nil, fmt.Errorf("manifest.json 解析失败: %w", err)
		}
		if manifest.Format != chatHistoryFormat || manifest.Version != chatHistoryFormatVersion {
			return nil, nil, fmt.Errorf("历史包格式不符: %s v%d（本机支持 %s v%d）",
				manifest.Format, manifest.Version, chatHistoryFormat, chatHistoryFormatVersion)
		}
	}
	var topics []chat.Topic
	{
		b, found, err := readEntry("topics.json")
		if err != nil {
			return nil, nil, err
		}
		if !found {
			return nil, nil, fmt.Errorf("历史包缺 topics.json")
		}
		if err := json.Unmarshal(b, &topics); err != nil {
			return nil, nil, fmt.Errorf("topics.json 解析失败: %w", err)
		}
	}
	var msgs []chat.Message
	{
		b, found, err := readEntry("messages.json")
		if err != nil {
			return nil, nil, err
		}
		if !found {
			return nil, nil, fmt.Errorf("历史包缺 messages.json")
		}
		if err := json.Unmarshal(b, &msgs); err != nil {
			return nil, nil, fmt.Errorf("messages.json 解析失败: %w", err)
		}
	}
	// 校验：话题 id 唯一；消息 topic_id 引用存在、角色合法、id>0、seq>=1。
	// extra 原样保留（bt 分支字段/answered_by/search 摘要），未知内容不做解读。
	topicIDs := make(map[string]bool, len(topics))
	for i := range topics {
		if topics[i].ID == "" {
			return nil, nil, fmt.Errorf("topics.json 存在空 id")
		}
		if topicIDs[topics[i].ID] {
			return nil, nil, fmt.Errorf("topics.json 话题 id 重复: %s", topics[i].ID)
		}
		topicIDs[topics[i].ID] = true
	}
	kept := msgs[:0]
	for _, m := range msgs {
		if m.ID <= 0 || m.Seq <= 0 || !topicIDs[m.TopicID] {
			return nil, nil, fmt.Errorf("messages.json 存在非法引用（topic=%s id=%d seq=%d）", m.TopicID, m.ID, m.Seq)
		}
		if m.Role != "user" && m.Role != "assistant" {
			return nil, nil, fmt.Errorf("messages.json 存在未知角色: %s", m.Role)
		}
		kept = append(kept, m)
	}
	return topics, kept, nil
}
