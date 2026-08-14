package chat

import (
	"database/sql"
	"fmt"
	"time"
)

// Topic 统一聊天话题。
type Topic struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Mode      string `json:"mode"` // "plain" | personaID
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Preview   string `json:"preview,omitempty"` // 首条消息内容（侧边栏可读预览）
}

// Message 统一聊天消息。
type Message struct {
	ID        int64  `json:"id"`
	TopicID   string `json:"topic_id"`
	Role      string `json:"role"` // user | assistant
	Content   string `json:"content"`
	Extra     string `json:"extra,omitempty"`
	Seq       int    `json:"seq"`
	CreatedAt string `json:"created_at"`
}

// MessageInput 批量追加的消息项（T6-3.3 语音持久化 / T6-3.4 导入事务共用）。
type MessageInput struct {
	Role    string // user | assistant
	Content string
	Extra   string
}

// Store 统一会话存储。
type Store struct {
	db      *sql.DB
	dataDir string
}

// NewStore 创建存储（dataDir 下 chat.db）。
func NewStore(dataDir string) *Store {
	return &Store{db: GetDatabase(dataDir), dataDir: dataDir}
}

// Close 关闭底层连接。
func (s *Store) Close() error {
	if s == nil || s.dataDir == "" {
		return nil
	}
	return CloseDatabase(s.dataDir)
}

func now() string { return time.Now().Format("2006-01-02 15:04:05") }

// CreateTopic 创建话题（重复 ID 报错）。
func (s *Store) CreateTopic(id, title, mode string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	if mode == "" {
		mode = "plain"
	}
	ts := now()
	_, err := s.db.Exec(
		"INSERT INTO chat_topics(id, title, mode, created_at, updated_at) VALUES(?,?,?,?,?)",
		id, title, mode, ts, ts)
	return err
}

// ListTopics 按创建时间列出话题，并用一条相关子查询取出每条话题的首条消息预览
// （避免逐话题 N+1 查询）。
func (s *Store) ListTopics() ([]Topic, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	rows, err := s.db.Query(`
		SELECT t.id, t.title, t.mode, t.created_at, t.updated_at,
		       COALESCE((SELECT m.content FROM chat_messages m
		                 WHERE m.topic_id = t.id ORDER BY m.seq LIMIT 1), '') AS preview
		FROM chat_topics t
		ORDER BY t.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Topic
	for rows.Next() {
		var t Topic
		if err := rows.Scan(&t.ID, &t.Title, &t.Mode, &t.CreatedAt, &t.UpdatedAt, &t.Preview); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTopic 按 ID 读取单个话题（含首条消息预览；不存在时返回错误）。
func (s *Store) GetTopic(id string) (Topic, error) {
	if s == nil || s.db == nil {
		return Topic{}, fmt.Errorf("chat store 未初始化")
	}
	var t Topic
	err := s.db.QueryRow(`
		SELECT id, title, mode, created_at, updated_at,
		       COALESCE((SELECT m.content FROM chat_messages m
		                 WHERE m.topic_id = chat_topics.id ORDER BY m.seq LIMIT 1), '')
		FROM chat_topics WHERE id = ?`, id).
		Scan(&t.ID, &t.Title, &t.Mode, &t.CreatedAt, &t.UpdatedAt, &t.Preview)
	if err != nil {
		if err == sql.ErrNoRows {
			return Topic{}, fmt.Errorf("话题不存在: %s", id)
		}
		return Topic{}, err
	}
	return t, nil
}

// RenameTopic 重命名话题。
func (s *Store) RenameTopic(id, title string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	res, err := s.db.Exec("UPDATE chat_topics SET title = ?, updated_at = ? WHERE id = ?", title, now(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("话题不存在: %s", id)
	}
	return nil
}

// SetMode 切换话题模式（plain ↔ personaID）。
func (s *Store) SetMode(id, mode string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	if mode == "" {
		mode = "plain"
	}
	res, err := s.db.Exec("UPDATE chat_topics SET mode = ?, updated_at = ? WHERE id = ?", mode, now(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("话题不存在: %s", id)
	}
	return nil
}

// ClearMessages 清空话题全部消息（保留话题与模式）。
func (s *Store) ClearMessages(id string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	_, err := s.db.Exec("DELETE FROM chat_messages WHERE topic_id = ?", id)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec("UPDATE chat_topics SET updated_at = ? WHERE id = ?", now(), id)
	return nil
}

// DeleteTopic 删除话题（消息级联删除）。
func (s *Store) DeleteTopic(id string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	_, err := s.db.Exec("DELETE FROM chat_topics WHERE id = ?", id)
	return err
}

// AppendMessage 追加消息（seq 自动递增；话题不存在时报错）。
// 用 RETURNING 一次性拿回自增 id 与 seq，避免「先查 MAX(seq)+1 再插入」的竞态窗口。
func (s *Store) AppendMessage(topicID, role, content, extra string) (*Message, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	ts := now()
	var m Message
	m.TopicID = topicID
	m.Role = role
	m.Content = content
	m.Extra = extra
	m.CreatedAt = ts
	err := s.db.QueryRow(`
		INSERT INTO chat_messages(topic_id, role, content, extra, seq, created_at)
		VALUES (?, ?, ?, ?,
		        COALESCE((SELECT MAX(seq) + 1 FROM chat_messages WHERE topic_id = ?), 1),
		        ?)
		RETURNING id, seq`,
		topicID, role, content, extra, topicID, ts).Scan(&m.ID, &m.Seq)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.Exec("UPDATE chat_topics SET updated_at = ? WHERE id = ?", ts, topicID)
	return &m, nil
}

// AppendExchange 以单事务原子写入「用户消息 + 助手消息」，并刷新话题 updated_at。
// 任一写入失败则整体回滚，避免出现只落库半条交换的情况。
func (s *Store) AppendExchange(topicID, userContent, assistantContent, assistantExtra string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ts := now()
	nextSeq := func() (int, error) {
		var seq int
		if err := tx.QueryRow(
			"SELECT COALESCE(MAX(seq), 0) + 1 FROM chat_messages WHERE topic_id = ?", topicID).Scan(&seq); err != nil {
			return 0, err
		}
		return seq, nil
	}

	seq1, err := nextSeq()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(
		"INSERT INTO chat_messages(topic_id, role, content, extra, seq, created_at) VALUES(?,?,?,?,?,?)",
		topicID, "user", userContent, "", seq1, ts); err != nil {
		return err
	}
	seq2, err := nextSeq()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(
		"INSERT INTO chat_messages(topic_id, role, content, extra, seq, created_at) VALUES(?,?,?,?,?,?)",
		topicID, "assistant", assistantContent, assistantExtra, seq2, ts); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE chat_topics SET updated_at = ? WHERE id = ?", ts, topicID); err != nil {
		return err
	}
	return tx.Commit()
}

// appendMessagesTx 在既有事务内按序追加消息（seq 连续递增），并刷新话题 updated_at。
// 供 AppendMessagesTx / ImportTopicTx 复用；任一条失败则调用方整体回滚。
func appendMessagesTx(tx *sql.Tx, topicID string, msgs []MessageInput, ts string) error {
	var seq int
	if err := tx.QueryRow(
		"SELECT COALESCE(MAX(seq), 0) FROM chat_messages WHERE topic_id = ?", topicID).Scan(&seq); err != nil {
		return err
	}
	for _, m := range msgs {
		seq++
		if _, err := tx.Exec(
			"INSERT INTO chat_messages(topic_id, role, content, extra, seq, created_at) VALUES(?,?,?,?,?,?)",
			topicID, m.Role, m.Content, m.Extra, seq, ts); err != nil {
			return err
		}
	}
	_, err := tx.Exec("UPDATE chat_topics SET updated_at = ? WHERE id = ?", ts, topicID)
	return err
}

// AppendMessagesTx 在单事务内批量追加消息（T6-3.3 语音消息持久化）：
// 全部成功或全部回滚；话题不存在（外键约束）时整体失败。
func (s *Store) AppendMessagesTx(topicID string, msgs []MessageInput) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := appendMessagesTx(tx, topicID, msgs, now()); err != nil {
		return err
	}
	return tx.Commit()
}

// ImportTopicTx 在单事务内创建话题并导入历史消息（T6-3.4 导入事务）：
// 建话题 + 全部消息要么一起成功，要么一起回滚，不残留半截数据。
func (s *Store) ImportTopicTx(id, title, mode string, msgs []MessageInput) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	if mode == "" {
		mode = "plain"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ts := now()
	if _, err := tx.Exec(
		"INSERT INTO chat_topics(id, title, mode, created_at, updated_at) VALUES(?,?,?,?,?)",
		id, title, mode, ts, ts); err != nil {
		return err
	}
	if err := appendMessagesTx(tx, id, msgs, ts); err != nil {
		return err
	}
	return tx.Commit()
}

// ListMessages 列出话题全部消息（按 seq）。
func (s *Store) ListMessages(topicID string) ([]Message, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	rows, err := s.db.Query(
		"SELECT id, topic_id, role, content, extra, seq, created_at FROM chat_messages WHERE topic_id = ? ORDER BY seq",
		topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.TopicID, &m.Role, &m.Content, &m.Extra, &m.Seq, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
