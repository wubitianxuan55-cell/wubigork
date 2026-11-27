// Package sessionquery implements session-search (v4.384, dsh ⑧ session-query
// distillation, first knife): a full-text index over past conversation
// transcripts so the model can recall what happened in earlier sessions of the
// same workspace — complementary to the memory fact store.
//
// 索引面：当前工作区+装配空间的会话事件日志（<dir>/<id>.gaea-log.jsonl）里的
// 用户消息与助手成稿消息正文（system 提示词/流式增量/工具结果等大噪声面不
// 入索引）。存储=用户级 SQLite 的 FTS5 虚表（SchemaV24，keyed by 日志路径），
// 增量维护按 (size, mtime) 新鲜度——搜索时惰性刷新，零 boot 成本。
//
// 中文降级：FTS5 unicode61 不切 CJK 子串，MATCH 零命中时回退 LIKE 子串扫描
// （对齐 whisper FTS5 先例）。空间隔离靠目录构造：Store 绑定装配空间的会话
// 目录，天然只索引/只命中本分区。
//
// 2026-11-27 AI 反馈 #1 根修（「刚结束的会话检索无命中」）：
//   - 此前只索引 assistant_message 这一种 kind，而实时落盘的助手成稿写的是
//     message——等于只搜得到用户说过的话，助手的回复一个字都搜不到；
//   - 多关键词按整句短语匹配（「造价库 导入」要求两词相邻），现在同一条消息
//     里各词都出现即命中，再退一步同一会话的不同消息里分别出现也命中；
//   - 一行坏日志曾让整个检索报错，现在跳过坏行/坏文件并在结果里注明；
//   - 读日志不再在盘上截 torn tail（正在写的会话日志被截会损坏写入）；
//   - query 留空=列出最近的会话（标题 / 起止 / 首条任务），轻量不撞预算。
package sessionquery

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// snippetHalfWindow 是 LIKE 降级路径手工摘要的半窗宽（字符）。
const snippetHalfWindow = 80

// Store 绑定一个会话目录（=工作区×空间分区）与用户级 SQLite。
type Store struct {
	db  *sql.DB
	dir string

	mu      sync.Mutex
	skipped map[string]error // 本次刷新读不出来的日志（路径 → 首个错误）
}

func NewStore(db *sql.DB, dir string) *Store {
	return &Store{db: db, dir: dir, skipped: map[string]error{}}
}

// metaRow 是一份日志的新鲜度指纹。
type metaRow struct {
	size  int64
	mtime int64
}

// Refresh 增量索引：只解析 (size, mtime) 变化或未索引过的日志文件。返回本次
// 实际索引的文件数。目录不存在（新工作区）时安静返回。单个文件读不出来只
// 记进 skipped、不中断其余文件（Search 在结果尾部注明）。
func (s *Store) Refresh() (int, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0, nil // 目录不存在：无可索引面（新工作区），非错误
	}
	indexed := 0
	present := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".gaea-log.jsonl") {
			continue
		}
		path := filepath.Join(s.dir, e.Name())
		present[path] = true
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		fresh := metaRow{size: fi.Size(), mtime: fi.ModTime().UnixNano()}
		if s.fresh(path, fresh) {
			continue
		}
		if err := s.indexFile(path, fresh); err != nil {
			// 一份坏日志不拖垮整个检索：记下来，继续刷其余的。
			s.mu.Lock()
			s.skipped[path] = err
			s.mu.Unlock()
			continue
		}
		indexed++
	}
	// 读不出的备注跟着文件走：文件已不在的清掉；新鲜文件没重读，此前的备注留着
	// （下一次检索照样看得到「这份没读全」）。
	s.mu.Lock()
	for p := range s.skipped {
		if !present[p] {
			delete(s.skipped, p)
		}
	}
	s.mu.Unlock()
	return indexed, nil
}

// Skipped 报告本次刷新读不出来的日志数与其中第一份的路径（0, "" = 全部读出）。
func (s *Store) Skipped() (int, string) {
	if s == nil {
		return 0, ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.skipped) == 0 {
		return 0, ""
	}
	first := ""
	for p := range s.skipped {
		if first == "" || p < first {
			first = p
		}
	}
	return len(s.skipped), first
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

// readLogTolerant 读一份会话日志：只认完整的行，解不出来的行跳过，末尾没写完
// 的半行直接忽略——不在盘上做 torn-tail 修复（正在写的日志被截会损坏写入）。
func readLogTolerant(path string) (entries []session.LogEntry, badLines int, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	for len(data) > 0 {
		i := bytes.IndexByte(data, '\n')
		var line []byte
		if i < 0 {
			break // 末尾半行（正在写）：忽略
		}
		line, data = data[:i], data[i+1:]
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var e session.LogEntry
		if json.Unmarshal(line, &e) != nil || e.Kind == "" {
			badLines++
			continue
		}
		entries = append(entries, e)
	}
	return entries, badLines, nil
}

// indexFile 重索引一份日志：先清该文件旧行再全量插入（FTS5 标准表的普通
// DELETE 即可，无需特殊命令）。
func (s *Store) indexFile(path string, cur metaRow) error {
	entries, badLines, err := readLogTolerant(path)
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
	s.mu.Lock()
	if badLines > 0 {
		s.skipped[path] = fmt.Errorf("%d 行解不出来（已跳过，其余行照常索引）", badLines)
	} else {
		delete(s.skipped, path) // 这次读全了：撤掉此前的备注
	}
	s.mu.Unlock()
	return tx.Commit()
}

// indexableContent 抽取可索引正文：用户消息（user_message）与助手成稿消息。
// 助手成稿有两种落盘形态：assistant_message（迁移 / 子代理提升写入）与
// message（实时落盘的 event.Message）——只认前者时，实时会话里助手的回复全部
// 搜不到（2026-11-27 AI 反馈 #1 的根因）。流式增量 text、推理、工具结果不入索引。
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
	case session.KindAssistantMessage, liveAssistantMessageKind:
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

// liveAssistantMessageKind 是实时落盘的助手成稿消息 kind（KindString 对
// event.Message 的映射，见 agent/session/log.go）。
const liveAssistantMessageKind = "message"

// splitTerms 把查询按空白拆成关键词（去重、保序）。
func splitTerms(query string) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range strings.Fields(query) {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// quotePhrase 把一个词包成 FTS5 短语（内嵌引号双写转义）。
func quotePhrase(t string) string {
	return `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
}

// Search 全文检索，三级放宽：FTS5 短语（各词 AND）→ 同一条消息里各词都作为
// 子串出现 → 同一会话的不同消息里分别出现。limit<=0 取 8，上限 32。
func (s *Store) Search(query string, limit int) ([]Hit, error) {
	terms := splitTerms(query)
	if len(terms) == 0 {
		return nil, errors.New("query is required")
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
	hits, err := s.searchFTS(terms, limit)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		if hits, err = s.searchLike(terms, limit, ""); err != nil {
			return nil, err
		}
	}
	if len(hits) == 0 && len(terms) > 1 {
		if hits, err = s.searchAcross(terms, limit); err != nil {
			return nil, err
		}
	}
	return hits, nil
}

// searchFTS FTS5 匹配：单词按整短语，多词各短语 AND。
func (s *Store) searchFTS(terms []string, limit int) ([]Hit, error) {
	parts := make([]string, len(terms))
	for i, t := range terms {
		parts[i] = quotePhrase(t)
	}
	rows, err := s.db.Query(
		`SELECT path, session_id, role, seq, ts, snippet(session_fts, 0, '', '', ' … ', 24)
		 FROM session_fts WHERE session_fts MATCH ? ORDER BY ts DESC LIMIT ?`, strings.Join(parts, " AND "), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return collectHits(rows)
}

// searchLike 子串扫描：每词都要在 content 里出现（CJK 子串——unicode61 不切
// CJK）。within 非空时只在这一个会话里找，取最新的一条。
func (s *Store) searchLike(terms []string, limit int, within string) ([]Hit, error) {
	var conds []string
	var args []any
	for _, t := range terms {
		conds = append(conds, "content LIKE '%' || ? || '%'")
		args = append(args, t)
	}
	q := `SELECT path, session_id, role, seq, ts, content FROM session_fts WHERE ` + strings.Join(conds, " AND ")
	if within != "" {
		q += ` AND session_id = ?`
		args = append(args, within)
	}
	q += ` ORDER BY ts DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Hit
	for rows.Next() {
		var h Hit
		var raw string
		if err := rows.Scan(&h.Path, &h.SessionID, &h.Role, &h.Seq, &h.Ts, &raw); err != nil {
			return nil, err
		}
		h.raw = raw
		h.Snippet = manualSnippet(raw, terms[0])
		out = append(out, h)
	}
	return out, rows.Err()
}

// searchAcross 同一会话的不同消息里各词分别出现：「沿江包2」「造价库」「导入」
// 散在三场对话的不同回合里也找得到那个会话。每个词取该会话里最新的一条命中。
func (s *Store) searchAcross(terms []string, limit int) ([]Hit, error) {
	type sessKey struct{ id, path string }
	latest := map[sessKey]map[string]Hit{} // 会话 → 词 → 最新命中
	order := map[sessKey]int64{}           // 会话 → 最新命中的 ts（排序用）
	for _, t := range terms {
		rows, err := s.db.Query(
			`SELECT path, session_id, role, seq, ts, content FROM session_fts
			 WHERE content LIKE '%' || ? || '%' ORDER BY ts DESC`, t)
		if err != nil {
			return nil, err
		}
		first := map[sessKey]Hit{} // 每个会话在这个词下的最新一条（ts 倒序的首条）
		for rows.Next() {
			var k sessKey
			var raw string
			var h Hit
			if err := rows.Scan(&k.path, &k.id, &h.Role, &h.Seq, &h.Ts, &raw); err != nil {
				_ = rows.Close()
				return nil, err
			}
			if _, seen := first[k]; seen {
				continue
			}
			h.Path, h.SessionID, h.raw = k.path, k.id, raw
			h.Snippet = manualSnippet(raw, t)
			h.Across = true
			first[k] = h
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if len(first) == 0 {
			return nil, nil // 有词在任何消息里都没出现：交集为空
		}
		if len(latest) == 0 {
			for k, h := range first {
				latest[k] = map[string]Hit{t: h}
				order[k] = h.Ts
			}
			continue
		}
		// 取交集：上一轮的会话里，这个词也命中了的才留下。
		next := map[sessKey]map[string]Hit{}
		for k, hs := range latest {
			h, ok := first[k]
			if !ok {
				continue
			}
			hs[t] = h
			next[k] = hs
			if h.Ts > order[k] {
				order[k] = h.Ts
			}
		}
		latest = next
	}
	keys := make([]sessKey, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return order[keys[i]] > order[keys[j]] })
	var out []Hit
	for _, k := range keys {
		if len(out) >= limit {
			break
		}
		// 按查询里的词序给出，读起来顺。
		for _, t := range terms {
			if h, ok := latest[k][t]; ok {
				out = append(out, h)
			}
		}
	}
	return out, nil
}

// Stats 报告索引规模（会话数 / 消息条数），零命中时随回执给出，让模型知道
// 索引是活的、只是这个词没出现过。
func (s *Store) Stats() (sessions, messages int) {
	if s == nil || s.db == nil {
		return 0, 0
	}
	_ = s.db.QueryRow(`SELECT COUNT(DISTINCT session_id), COUNT(*) FROM session_fts`).Scan(&sessions, &messages)
	return sessions, messages
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
	// Across 标记「同一会话的不同消息里分别命中」的放宽路结果。
	Across bool
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
func (s *Store) FormatHits(query string, hits []Hit) string {
	var b strings.Builder
	if len(hits) == 0 {
		sessions, messages := s.Stats()
		fmt.Fprintf(&b, "No past session matches %q.（已索引本工作区 %d 个会话的 %d 条消息；换关键词重试——多个关键词要出现在同一场会话里；query 留空可列出最近的会话；要整场回顾用 @会话:<标题或文件名>）\n",
			query, sessions, messages)
	} else {
		fmt.Fprintf(&b, "Found %d past-session matches for %q (newest first):\n", len(hits), query)
		for i, h := range hits {
			when := time.Unix(h.Ts, 0).Format("2006-01-02 15:04")
			fmt.Fprintf(&b, "\n%d. [%s %s @ %s | seq %d]\n   %s", i+1, h.SessionID, h.Role, when, h.Seq, h.Snippet)
			if h.Across {
				b.WriteString("\n   （各关键词出现在这场会话的不同消息里）")
			}
		}
	}
	if n, first := s.Skipped(); n > 0 {
		fmt.Fprintf(&b, "\n⚠ 有 %d 份会话日志没能完整读出，已跳过读不出的部分（如 %s）。\n", n, filepath.Base(first))
	}
	return b.String()
}

// FormatHits 渲染一份现成的命中列表（不含索引规模与跳过说明；保留给旧调用方）。
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

// ── 最近会话清单 ───────────────────────────────────────────────────────

// SessionMeta 一场会话的概要（标题 / 起止 / 首条任务 / 收尾回复 / 消息条数）。
type SessionMeta struct {
	ID          string
	Title       string
	Started     time.Time
	LastActive  time.Time
	UserMsgs    int
	AsstMsgs    int
	FirstPrompt string
	// LastReply 最后一条助手成稿的开头（压成一行）：这场会话是怎么收的尾
	// （2026-11-27 AI 反馈 #14：跨会话接力时下一场先看上一场的结论）。
	LastReply string
}

// Recent 列出最近的会话（按最后活动时间倒序），数据来自索引：一场会话至少
// 要有一条成稿消息才会出现。
func (s *Store) Recent(limit int) ([]SessionMeta, error) {
	if limit <= 0 {
		limit = 8
	}
	if limit > 32 {
		limit = 32
	}
	if _, err := s.Refresh(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT session_id, path, MIN(ts), MAX(ts), SUM(role='user'), SUM(role='assistant')
		 FROM session_fts GROUP BY session_id, path ORDER BY MAX(ts) DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	type row struct {
		id, path     string
		start, last  int64
		users, assts sql.NullInt64
	}
	var raws []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.path, &r.start, &r.last, &r.users, &r.assts); err != nil {
			_ = rows.Close()
			return nil, err
		}
		raws = append(raws, r)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	titles := loadTitles(s.dir)
	out := make([]SessionMeta, 0, len(raws))
	for _, r := range raws {
		m := SessionMeta{ID: r.id, Started: time.Unix(r.start, 0), LastActive: time.Unix(r.last, 0),
			UserMsgs: int(r.users.Int64), AsstMsgs: int(r.assts.Int64), Title: titles[r.id]}
		if st, ok := startTimeFromID(r.id); ok {
			m.Started = st
		}
		// 首条用户消息（压成一句话）= 这场会话的任务。
		var first string
		if err := s.db.QueryRow(`SELECT content FROM session_fts WHERE session_id = ? AND role = 'user' ORDER BY seq ASC LIMIT 1`, r.id).Scan(&first); err == nil {
			m.FirstPrompt = oneLine(first, 80)
		}
		// 最后一条助手成稿的开头 = 这场会话怎么收的尾；要看全文用 handoff。
		var last string
		if err := s.db.QueryRow(`SELECT content FROM session_fts WHERE session_id = ? AND role = 'assistant' ORDER BY seq DESC LIMIT 1`, r.id).Scan(&last); err == nil {
			m.LastReply = oneLine(last, 160)
		}
		out = append(out, m)
	}
	return out, nil
}

// loadTitles 读会话目录的标题注册表（<dir>/.titles.json：{文件名: 标题}）。
// 键是会话的旧 jsonl 文件名（<id>.jsonl）；缺失或损坏=没有标题。
func loadTitles(dir string) map[string]string {
	out := map[string]string{}
	raw, err := os.ReadFile(filepath.Join(dir, ".titles.json"))
	if err != nil {
		return out
	}
	var m map[string]string
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	for k, v := range m {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		k = strings.TrimSuffix(k, ".jsonl")
		out[k] = v
	}
	return out
}

// startTimeFromID 从会话 id 的时间戳段取开始时间（20261127-123417.443964200-gaea）。
func startTimeFromID(id string) (time.Time, bool) {
	if len(id) < 15 || id[8] != '-' {
		return time.Time{}, false
	}
	stamp := id[:8] + id[9:15]
	t, err := time.ParseInLocation("20060102150405", stamp, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// oneLine 把一段文本压成一行并截断。
func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// FormatRecent 渲染最近会话清单。
func FormatRecent(list []SessionMeta) string {
	if len(list) == 0 {
		return "本工作区还没有可列的过往会话（索引为空）。"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "最近的 %d 场会话（按最后活动时间倒序；要回顾某一场用 session_search 带关键词检索，或 @会话:<标题或文件名> 注入快照；接续上一场的工作先看它的交接卡：session_search handoff=<会话 id>）：\n", len(list))
	for i, m := range list {
		title := m.Title
		if title == "" {
			title = "（无标题）"
		}
		task := m.FirstPrompt
		if strings.TrimSpace(task) == "" {
			task = "（没有用户消息）"
		}
		fmt.Fprintf(&b, "\n%d. %s｜%s\n   %s → %s｜用户 %d 条 / 助手 %d 条\n   任务：%s\n",
			i+1, m.ID, title,
			m.Started.Format("01-02 15:04"), m.LastActive.Format("01-02 15:04"), m.UserMsgs, m.AsstMsgs,
			task)
		if m.LastReply != "" {
			fmt.Fprintf(&b, "   收尾：%s\n", m.LastReply)
		}
	}
	return b.String()
}
