package chat

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s := NewStore(filepath.Join(t.TempDir(), "chat"))
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestStore_TopicCRUD(t *testing.T) {
	s := newTestStore(t)

	if err := s.CreateTopic("t1", "闲聊", "plain"); err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	if err := s.CreateTopic("t2", "轻语", "gaea"); err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	// 重复创建应报错
	if err := s.CreateTopic("t1", "重复", "plain"); err == nil {
		t.Fatal("重复 topic ID 应报错")
	}

	topics, err := s.ListTopics()
	if err != nil {
		t.Fatalf("ListTopics: %v", err)
	}
	if len(topics) != 2 {
		t.Fatalf("话题数 = %d, want 2", len(topics))
	}
	if topics[0].Title != "闲聊" || topics[0].Mode != "plain" {
		t.Errorf("topic[0] = %+v", topics[0])
	}

	if err := s.RenameTopic("t1", "新标题"); err != nil {
		t.Fatalf("RenameTopic: %v", err)
	}
	topics, _ = s.ListTopics()
	if topics[0].Title != "新标题" {
		t.Errorf("重命名失败: %+v", topics[0])
	}
}

func TestStore_GetTopic(t *testing.T) {
	s := newTestStore(t)
	if err := s.CreateTopic("t1", "预览", "plain"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendMessage("t1", "user", "首条消息", ""); err != nil {
		t.Fatal(err)
	}

	topic, err := s.GetTopic("t1")
	if err != nil {
		t.Fatalf("GetTopic: %v", err)
	}
	if topic.Title != "预览" || topic.Preview != "首条消息" {
		t.Errorf("GetTopic = %+v", topic)
	}

	if _, err := s.GetTopic("nope"); err == nil {
		t.Fatal("不存在的 topic 应报错")
	}
}

func TestStore_AppendExchange(t *testing.T) {
	s := newTestStore(t)
	if err := s.CreateTopic("t1", "交换", "plain"); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendExchange("t1", "你好", "你好呀", `{"emotion":"calm"}`); err != nil {
		t.Fatalf("AppendExchange: %v", err)
	}

	msgs, err := s.ListMessages("t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("消息数 = %d, want 2", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "你好" || msgs[0].Seq != 1 {
		t.Errorf("user 消息异常: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "你好呀" || msgs[1].Extra != `{"emotion":"calm"}` || msgs[1].Seq != 2 {
		t.Errorf("assistant 消息异常: %+v", msgs[1])
	}

	// 不存在的话题应整体回滚，不残留任何消息。
	if err := s.AppendExchange("nope", "a", "b", ""); err == nil {
		t.Fatal("不存在话题应报错")
	}
	if msgs, _ := s.ListMessages("nope"); len(msgs) != 0 {
		t.Errorf("失败事务不应残留消息: %+v", msgs)
	}
}

func TestStore_MessagesAndCascadeDelete(t *testing.T) {
	s := newTestStore(t)
	if err := s.CreateTopic("t1", "话题", "plain"); err != nil {
		t.Fatal(err)
	}

	m1, err := s.AppendMessage("t1", "user", "你好", "")
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	if m1.Seq != 1 {
		t.Errorf("第一条 seq = %d, want 1", m1.Seq)
	}
	if _, err := s.AppendMessage("t1", "assistant", "你好！", "{}"); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	msgs, err := s.ListMessages("t1")
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("消息数 = %d, want 2", len(msgs))
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "你好！" || msgs[1].Extra != "{}" {
		t.Errorf("msgs[1] = %+v", msgs[1])
	}

	// 级联删除：删话题后消息一并清除
	if err := s.DeleteTopic("t1"); err != nil {
		t.Fatalf("DeleteTopic: %v", err)
	}
	msgs, _ = s.ListMessages("t1")
	if len(msgs) != 0 {
		t.Errorf("级联删除失败：话题删除后仍有 %d 条消息", len(msgs))
	}
}

func TestStore_AppendToMissingTopic(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.AppendMessage("nope", "user", "hi", ""); err == nil {
		t.Fatal("向不存在的话题追加消息应报错")
	}
}
