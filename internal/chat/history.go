package chat

import (
	"fmt"
)

// history.go —— E 档聊天历史导入的原子替换面（chat-demo backup.ts 语义对齐的
// Go 侧简化：SQLite 天然单事务，无需 IndexedDB staging——校验全部通过前
// 不动正式数据，任一步失败整体回滚）。
//
// 消息 id 与 seq 显式保留：分支树 bt.p/c 与轮次排序都引用消息 id，
// 导入后续写 AppendMessage 的自增 id 从 max(id)+1 自然衔接。

// ReplaceAllTx 单事务原子替换全部话题与消息。topics 缺 preview（派生列，
// ListTopics 子查询按首条消息现算）；messages 需带合法 TopicID/ID/Seq。
func (s *Store) ReplaceAllTx(topics []Topic, msgs []Message) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("chat store 未初始化")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("DELETE FROM chat_messages"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM chat_topics"); err != nil {
		return err
	}
	for i := range topics {
		t := &topics[i]
		if _, err := tx.Exec(
			"INSERT INTO chat_topics(id, title, mode, created_at, updated_at) VALUES(?,?,?,?,?)",
			t.ID, t.Title, t.Mode, t.CreatedAt, t.UpdatedAt); err != nil {
			return fmt.Errorf("写入话题 %s: %w", t.ID, err)
		}
	}
	for i := range msgs {
		m := &msgs[i]
		if _, err := tx.Exec(
			"INSERT INTO chat_messages(id, topic_id, role, content, extra, seq, created_at) VALUES(?,?,?,?,?,?,?)",
			m.ID, m.TopicID, m.Role, m.Content, m.Extra, m.Seq, m.CreatedAt); err != nil {
			return fmt.Errorf("写入消息 %d: %w", m.ID, err)
		}
	}
	return tx.Commit()
}
