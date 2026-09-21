// Package sessionquery implements session-search (v4.384, dsh ⑧ session-query
// distillation, first knife): a full-text index over past conversation
// transcripts so the model can recall what happened in earlier sessions of the
// same workspace — complementary to the memory fact store.
//
// 索引面：当前工作区+装配空间的会话事件日志（<dir>/<id>.gaea-log.jsonl）里的
// user_message / assistant_message 正文（system 提示词/工具结果等大噪声面不
// 入索引）。存储=用户级 SQLite 的 FTS5 虚表（SchemaV24，keyed by 日志路径），
// 增量维护按 (size, mtime) 新鲜度——搜索时惰性刷新，零 boot 成本。
//
// 中文降级：FTS5 unicode61 不切 CJK 子串，MATCH 零命中时回退 LIKE 子串扫描
// （对齐 whisper FTS5 先例）。空间隔离靠目录构造：Store 绑定装配空间的会话
// 目录，天然只索引/只命中本分区。
package sessionquery

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// IndexSniffChars 是 LIKE 降级路径手工摘要的半窗宽（字符）。
const snippetHalfWindow = 80

// Store 绑定一个会话目录（=工作区×空间分区）与用户级 SQLite。
type Store struct {
	db  *sql.DB
	dir string
}

func NewStore(db *sql.DB, dir string) *Store {
	return &Store{db: db, dir: dir}
}

// metaRow 是一份日志的新鲜度指纹。
type metaRow struct {
	size  int64
	mtime int64
}

// Refresh 增量索引：只解析 (size, mtime) 变化或未索引过的日志文件。返回本次
// 实际索引的文件数。目录不存在（新工作区）时安静返回。
func (s *Store) Refresh() (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0, nil // 目录不存在：无可索引面（新工作区），非错误
	}
	indexed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".gaea-log.jsonl") {
			continue
		}
		path := filepath.Join(s.dir, e.Name())
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		fresh := metaRow{size: fi.Size(), mtime: fi.ModTime().UnixNano()}
		if s.fresh(path, fresh) {
			continue
		}
		if err := s.indexFile(path, fresh); err != nil {
			return indexed, err
		}
		indexed++
	}
	return indexed, nil
}

// fresh 报告该日志是否已按当前指纹索引过。
func (s *Store) fresh(path string, cur metaRow) bool {
	row := s.db.QueryRow(`SELECT size, mtime FROM session_index_meta WHERE path = ?`, path)
	var size, mtime int64
	if err := row.Scan(&size, &mtime); err != nil {
		return false
	}
	return size == cur.size && mtime == cur.mtime
}

// indexFile 重索引一份日志：先清该文件旧行再全量插入（FTS5 标准表的普通
// DELETE 即可，无需特殊命令）。
func (s *Store) indexFile(path string, cur metaRow) error {
	entries, err := session.ReadLogRepaired(path)
	if err != nil {
		return fmt.Errorf("read session log %s: %w", path, err)
	}
	sessionID := strings.TrimSuffix(filepath.Base(path), ".gaea-log.jsonl")
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM session_fts WHERE path = ?`, path); err != nil {
		return err
	}
	for _, e := range entries {
		role, content, ok := indexableContent(e)
		if !ok {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO session_fts(content, path, session_id, role, seq, ts) VALUES(?,?,?,?,?,?)`,
			content, path, sessionID, role, e.Seq, e.Ts); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO session_index_meta(path, size, mtime) VALUES(?,?,?)
		ON CONFLICT(path) DO UPDATE SET size=excluded.size, mtime=excluded.mtime`,
		path, cur.size, cur.mtime); err != nil {
		return err
	}
	return tx.Commit()
}

// indexableContent 抽取可索引正文：仅 user_message / assistant_message
// （system 提示词与工具结果是大噪声面，不入索引）。
func indexableContent(e session.LogEntry) (role, content string, ok bool) {
	switch e.Kind {
	case session.KindUserMessage:
		var p struct {
			Content string `json:"content"`
		}
		if json.Unmarshal(e.Payload, &p) != nil || strings.TrimSpace(p.Content) == "" {
			return "", "", false
		}
		return "user", p.Content, true
	case session.KindAssistantMessage:
		var p struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(e.Payload, &p) != nil || strings.TrimSpace(p.Text) == "" {
			return "", "", false
		}
		return "assistant", p.Text, true
	}
	return "", "", false
}

// Search 全文检索：FTS5 MATCH 优先（ASCII 词干友好），零命中回退 LIKE 子串
// 扫描（CJK 子串——unicode61 不切 CJK）。limit<=0 取 8，上限 32。
func (s *Store) Search(query string, limit int) ([]Hit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if limit <= 0 {
		limit = 8
	}
	if limit > 32 {
		limit = 32
	}
	if _, err := s.Refresh(); err != nil {
		return nil, err
	}
	hits, err := s.searchFTS(query, limit)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		hits, err = s.searchLike(query, limit)
	}
	return hits, err
}

func (s *Store) searchFTS(query string, limit int) ([]Hit, error) {
	// 短语引号化：用户输入里的内嵌引号双写转义，其余按整短语匹配。
	phrase := `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
	rows, err := s.db.Query(
		`SELECT path, session_id, role, seq, ts, snippet(session_fts, 0, '', '', ' … ', 24)
		 FROM session_fts WHERE session_fts MATCH ? ORDER BY ts DESC LIMIT ?`, phrase, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectHits(rows)
}

func (s *Store) searchLike(query string, limit int) ([]Hit, error) {
	rows, err := s.db.Query(
		`SELECT path, session_id, role, seq, ts, content FROM session_fts
		 WHERE content LIKE '%' || ? || '%' ORDER BY ts DESC LIMIT ?`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Hit
	for rows.Next() {
		var h Hit
		var raw string
		if err := rows.Scan(&h.Path, &h.SessionID, &h.Role, &h.Seq, &h.Ts, &raw); err != nil {
			return nil, err
		}
		h.raw = raw
		h.Snippet = manualSnippet(raw, query)
		out = append(out, h)
	}
	return out, rows.Err()
}

type hitRow struct {
	path      string
	sessionID string
	role      string
	seq       int64
	ts        int64
	raw       string
}

func collectHits(rows *sql.Rows) ([]Hit, error) {
	var out []Hit
	for rows.Next() {
		var r hitRow
		var snippet sql.NullString
		if err := rows.Scan(&r.path, &r.sessionID, &r.role, &r.seq, &r.ts, &snippet); err != nil {
			return nil, err
		}
		h := Hit{SessionID: r.sessionID, Role: r.role, Seq: r.seq, Ts: r.ts, Path: r.path, raw: r.raw}
		h.Snippet = snippet.String
		if h.Snippet == "" {
			h.Snippet = manualSnippet(r.raw, "")
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Hit 是一条会话检索命中。
type Hit struct {
	SessionID string
	Role      string
	Seq       int64
	Ts        int64
	Snippet   string
	Path      string
	// raw 是 LIKE 路径的完整正文（snippet 手工裁窗用）；FTS 路径为空。
	raw string
}

// manualSnippet 在原文中定位 query 首现位置裁窗（LIKE 降级路径——没有
// FTS5 snippet() 可用）；找不到时返回头部截断。
func manualSnippet(content, query string) string {
	r := []rune(content)
	lower := strings.ToLower(content)
	lq := strings.ToLower(query)
	idx := -1
	if query != "" {
		idx = strings.Index(lower, lq)
	}
	if idx < 0 {
		if len(r) > snippetHalfWindow*2 {
			return string(r[:snippetHalfWindow*2]) + " …"
		}
		return content
	}
	// 字节偏移转 rune 偏移（content 前缀的 rune 数）。
	ri := len([]rune(content[:idx]))
	start := ri - snippetHalfWindow
	if start < 0 {
		start = 0
	}
	end := ri + len([]rune(query)) + snippetHalfWindow
	if end > len(r) {
		end = len(r)
	}
	out := string(r[start:end])
	if start > 0 {
		out = "… " + out
	}
	if end < len(r) {
		out += " …"
	}
	return out
}

// FormatHits 渲染工具可读结果（确定性序已由 SQL 保证）。
func FormatHits(query string, hits []Hit) string {
	if len(hits) == 0 {
		return fmt.Sprintf("No past session matches %q.", query)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Found %d past-session matches for %q (newest first):\n", len(hits), query)
	for i, h := range hits {
		when := time.Unix(h.Ts, 0).Format("2006-01-02 15:04")
		fmt.Fprintf(&b, "\n%d. [%s %s @ %s | seq %d]\n   %s", i+1, h.SessionID, h.Role, when, h.Seq, h.Snippet)
	}
	return b.String()
}
