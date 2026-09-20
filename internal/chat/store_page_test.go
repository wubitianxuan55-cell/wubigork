package chat

// v4.370：ListMessagesPage 游标分页契约——limit 截断/hasMore/beforeSeq 游标/升序返回。
import (
	"path/filepath"
	"testing"
)

func TestListMessagesPage(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "chat"))
	t.Cleanup(func() { _ = s.Close() })

	if err := s.CreateTopic("t_walk", "走查", "plain"); err != nil {
		t.Fatalf("create topic: %v", err)
	}
	var msgs []MessageInput
	for i := 0; i < 10; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		msgs = append(msgs, MessageInput{Role: role, Content: string(rune('a' + i))})
	}
	if err := s.AppendMessagesTx("t_walk", msgs); err != nil {
		t.Fatalf("append: %v", err)
	}

	// 首屏：最新 4 条，升序返回，hasMore=true（还有更早 6 条）
	page, hasMore, err := s.ListMessagesPage("t_walk", 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !hasMore || len(page) != 4 {
		t.Fatalf("首屏应 4 条+hasMore, got %d hasMore=%v", len(page), hasMore)
	}
	if page[0].Content != string(rune('g')) || page[3].Content != string(rune('j')) {
		t.Fatalf("首屏应为第 7~10 条升序, got %v..%v", page[0].Content, page[3].Content)
	}

	// 以首屏最早 seq 为游标继续拉 4 条
	page2, hasMore2, err := s.ListMessagesPage("t_walk", 4, int64(page[0].Seq))
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 4 || !hasMore2 {
		t.Fatalf("第二页应 4 条+hasMore, got %d %v", len(page2), hasMore2)
	}
	if page2[0].Content != string(rune('c')) || page2[3].Content != string(rune('f')) {
		t.Fatalf("第二页应为第 3~6 条, got %v..%v", page2[0].Content, page2[3].Content)
	}

	// 拉到底：hasMore=false
	page3, hasMore3, err := s.ListMessagesPage("t_walk", 4, int64(page2[0].Seq))
	if err != nil {
		t.Fatal(err)
	}
	if hasMore3 {
		t.Fatalf("拉到底应 hasMore=false, got %d 条", len(page3))
	}
	if len(page3) != 2 || page3[0].Content != string(rune('a')) || page3[1].Content != string(rune('b')) {
		t.Fatalf("尾页应为最早 2 条, got %d 条 %v..%v", len(page3), page3[0].Content, page3[1].Content)
	}
}
