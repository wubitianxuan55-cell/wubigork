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
// 模型检索本工作区过往会话的 user/assistant 正文——跨会话记忆与 memory 事实库
// 互补（memory 存提炼后的事实，本工具查原始对话）。只读；绑定装配空间会话
// 目录，空间隔离由目录构造保证。
type searchTool struct{ store *Store }

// NewSearchTool 返回绑定 (db, sessionDir) 的 session_search 工具。db 为用户级
// SQLite（GetDatabase），dir 为装配空间的会话目录。
func NewSearchTool(db *sql.DB, sessionDir string) tool.Tool {
	return searchTool{store: NewStore(db, sessionDir)}
}

func (searchTool) Name() string { return "session_search" }

func (searchTool) Description() string {
	return "Search past conversation transcripts of this workspace (user and assistant messages across earlier sessions). " +
		"Use to recall what was discussed/done before: prior decisions, past errors and fixes, earlier versions of a plan. " +
		"Returns newest-first matches with session id, role and a snippet. Chinese queries match by substring fallback. " +
		"Complements memory: memory holds distilled facts, this searches the raw conversations."
}

func (searchTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Text to search for in past conversations (verbatim words or short phrase works best)."},"limit":{"type":"integer","description":"Max matches to return (default 8, max 32)."}},"required":["query"]}`)
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
		return "", fmt.Errorf("query is required")
	}
	// SQLITE_BUSY 短重试（2026-11-25 铜陵反馈 #11）：会话库与主库共用一个
	// SQLite 文件，写入回合撞上即偶发 busy——一次失败重试即过，别让模型白跑
	// 一轮（3 次 × 60ms 退避，覆盖写入事务的尾窗口）。
	var hits []Hit
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		hits, err = t.store.Search(p.Query, p.Limit)
		if err == nil || !strings.Contains(err.Error(), "SQLITE_BUSY") {
			break
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 60 * time.Millisecond):
		}
	}
	if err != nil {
		return "", err
	}
	return FormatHits(p.Query, hits), nil
}
