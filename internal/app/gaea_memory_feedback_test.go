package app

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// feedbackTestDB 隔离临时库（真迁移链）；Windows 下必须 Cleanup 关库句柄，
// 否则 t.TempDir() 清理与未关闭的 SQLite 文件竞态（unlinkat file in use，
// v4.237 在册 flaky 同族——这里从源头关掉）。
func feedbackTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	return gdb
}

// TestGaeaMemoryFeedbackEvents 助手回答反馈落 feedback 事件（v4.238 能力层）：
// 事件写入 memory_events（op=feedback，Title 点赞/点踩，摘要截断），
// 文件后端诚实报错；投影对 feedback 只出事件节点不建实体（cite/pin 同边界）。
func TestGaeaMemoryFeedbackEvents(t *testing.T) {
	database := feedbackTestDB(t)
	store := memory.SQLiteStoreFor(database, t.TempDir(), t.TempDir())
	SetOfficeStoreForTest(store)
	defer ResetOfficeStoreForTest()

	a := &App{}
	// 非法 rating 拒绝
	if err := a.GaeaMemoryFeedback("a17", "meh", "", "work"); err == nil || !strings.Contains(err.Error(), "up 或 down") {
		t.Fatalf("非法 rating 应拒绝: %v", err)
	}
	// 点赞：事件落库，Title=点赞回答，摘要截断到 eventExcerptLimit
	long := strings.Repeat("很", 600)
	if err := a.GaeaMemoryFeedback("a17", "up", long, "work"); err != nil {
		t.Fatalf("点赞反馈: %v", err)
	}
	// 点踩：Title=点踩回答
	if err := a.GaeaMemoryFeedback("a18", "down", "不满意", "work"); err != nil {
		t.Fatalf("点踩反馈: %v", err)
	}
	events, err := (&memory.EventLog{DB: database}).LoadEvents()
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	var up, down *memory.Event
	for i := range events {
		if events[i].Op != memory.OpFeedback {
			continue
		}
		if events[i].Name == "a17" {
			up = &events[i]
		}
		if events[i].Name == "a18" {
			down = &events[i]
		}
	}
	if up == nil || down == nil {
		t.Fatalf("feedback 事件缺失: %+v", events)
	}
	if up.Title != "点赞回答" || down.Title != "点踩回答" {
		t.Fatalf("Title 错误: %q / %q", up.Title, down.Title)
	}
	if len(up.Excerpt) > 280 {
		t.Fatalf("摘要应截断到 280 字节内: %d", len(up.Excerpt))
	}
	if up.Space != "work" || up.Actor != "panel" || up.SourceMessage != "a17" {
		t.Fatalf("事件字段错误: %+v", up)
	}
}

// TestGaeaMemoryFeedbackProjectionNoEntity 投影对 feedback 只出事件节点、
// 不建实体/边（与 cite「悬空拒写」同边界：反馈不是记忆事实，不得节点化）。
func TestGaeaMemoryFeedbackProjectionNoEntity(t *testing.T) {
	database := feedbackTestDB(t)
	store := memory.SQLiteStoreFor(database, t.TempDir(), t.TempDir())
	if err := store.AppendFeedbackEvent(memory.Event{
		At: 1234, Op: memory.OpFeedback, Name: "a1", Title: "点赞回答",
		Space: "work", Project: "proj", Actor: "panel",
	}); err != nil {
		t.Fatalf("AppendFeedbackEvent: %v", err)
	}
	events, err := (&memory.EventLog{DB: database}).LoadEvents()
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	graph := memory.ProjectEvents(events)
	for _, n := range graph.Nodes {
		if n.NType == memory.NodeEntity {
			t.Fatalf("feedback 不应建实体节点: %+v", n)
		}
	}
	hasEvent := false
	for _, n := range graph.Nodes {
		if n.NType == memory.NodeEvent {
			hasEvent = true
		}
	}
	if !hasEvent {
		t.Fatalf("feedback 事件节点应在图上")
	}
}

// TestGaeaMemoryFeedbackFileBackendHonest 文件后端不支持反馈：明确报错不假成功。
func TestGaeaMemoryFeedbackFileBackendHonest(t *testing.T) {
	store := memory.Store{Dir: t.TempDir(), GlobalDir: t.TempDir()}
	if err := store.AppendFeedbackEvent(memory.Event{Op: memory.OpFeedback}); err == nil {
		t.Fatalf("文件后端应报错")
	}
}
