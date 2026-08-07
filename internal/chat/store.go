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

// ListTopics 按创建时间列出话题。
func (s *Store) ListTopics() ([]Topic, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	rows, err := s.db.Query("SELECT id, title, mode, created_at, updated_at FROM chat_topics ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Topic
	for rows.Next() {
		var t Topic
		if err := rows.Scan(&t.ID, &t.Title, &t.Mode, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
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

// DeleteTopic 删除话题（消息级联删除）。
func (s *Store) DeleteTopic(id string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	_, err := s.db.Exec("DELETE FROM chat_topics WHERE id = ?", id)
	return err
}

// AppendMessage 追加消息（seq 自动递增；话题不存在时报错）。
func (s *Store) AppendMessage(topicID, role, content, extra string) (*Message, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("chat store 未初始化")
	}
	var seq int
	if err := s.db.QueryRow(
		"SELECT COALESCE(MAX(seq), 0) + 1 FROM chat_messages WHERE topic_id = ?", topicID).Scan(&seq); err != nil {
		return nil, err
	}
	ts := now()
	res, err := s.db.Exec(
		"INSERT INTO chat_messages(topic_id, role, content, extra, seq, created_at) VALUES(?,?,?,?,?,?)",
		topicID, role, content, extra, seq, ts)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Message{ID: id, TopicID: topicID, Role: role, Content: content, Extra: extra, Seq: seq, CreatedAt: ts}, nil
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
