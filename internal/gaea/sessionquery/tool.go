package sessionquery

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// searchTool 是 session_search 工具（v4.384，dsh ⑧ session-query 蒸馏首刀）：
// 模型检索本工作区过往会话的用户与助手成稿消息——跨会话记忆与 memory 事实库
// 互补（memory 存提炼后的事实，本工具查原始对话）。只读；绑定装配空间会话
// 目录，空间隔离由目录构造保证。
//
// 2026-11-27 AI 反馈 #1：query 留空=列出最近的会话（标题 / 起止 / 首条任务）——
// 「刚结束的会话在哪」不必靠猜关键词去撞。
type searchTool struct{ store *Store }

// NewSearchTool 返回绑定 (db, sessionDir) 的 session_search 工具。db 为用户级
// SQLite（GetDatabase），dir 为装配空间的会话目录。
func NewSearchTool(db *sql.DB, sessionDir string) tool.Tool {
	return searchTool{store: NewStore(db, sessionDir)}
}

func (searchTool) Name() string { return "session_search" }

func (searchTool) Description() string {
	return "Search past conversation transcripts of this workspace (user and assistant messages across earlier sessions, including the one that just ended). " +
		"Use to recall what was discussed/done before: prior decisions, past errors and fixes, earlier versions of a plan. " +
		"Multiple keywords all need to appear in the same session (same message first, then anywhere in that session). " +
		"Leave query empty to list the most recent sessions with title, time span and first task. " +
		"Complements memory: memory holds distilled facts, this searches the raw conversations."
}

func (searchTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Keywords to search in past conversations (space-separated; all must appear in the same session). Empty = list the most recent sessions (title / time span / first task)."},"limit":{"type":"integer","description":"Max matches (or sessions) to return (default 8, max 32)."}}}`)
}

func (searchTool) ReadOnly() bool { return true }

func (t searchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(p.Query) == "" {
		var list []SessionMeta
		if err := t.withBusyRetry(ctx, func() error {
			var rerr error
			list, rerr = t.store.Recent(p.Limit)
			return rerr
		}); err != nil {
			return "", err
		}
		out := FormatRecent(list)
		if n, first := t.store.Skipped(); n > 0 {
			out += fmt.Sprintf("\n⚠ 有 %d 份会话日志没能完整读出，已跳过读不出的部分（如 %s）。\n", n, first)
		}
		return out, nil
	}
	// SQLITE_BUSY 短重试（2026-11-25 铜陵反馈 #11）：会话库与主库共用一个
	// SQLite 文件，写入回合撞上即偶发 busy——一次失败重试即过，别让模型白跑
	// 一轮（3 次 × 60ms 退避，覆盖写入事务的尾窗口）。
	var hits []Hit
	if err := t.withBusyRetry(ctx, func() error {
		var rerr error
		hits, rerr = t.store.Search(p.Query, p.Limit)
		return rerr
	}); err != nil {
		return "", err
	}
	return t.store.FormatHits(p.Query, hits), nil
}

// withBusyRetry 在 SQLITE_BUSY 时短重试（3 次 × 60ms 退避）。
func (searchTool) withBusyRetry(ctx context.Context, call func() error) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = call()
		if err == nil || !strings.Contains(err.Error(), "SQLITE_BUSY") {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 60 * time.Millisecond):
		}
	}
	return err
}
