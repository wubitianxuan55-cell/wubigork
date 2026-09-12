package memory

// 刀B（v4.246）写路径事务化：实体语句与事件留痕单事务提交（改前两条独立
// 写事务=WAL 下双倍 commit/fsync）；事件仍尽力而为——事件落库失败时写入
// 本体必须存活（回退单语句路径，语义同改前）。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func txTestStore(t *testing.T) (Store, *EventLog, func()) {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	s := SQLiteStoreFor(gdb, dir, "/tx-proj")
	return s, &EventLog{DB: gdb}, func() { db.CloseDatabase(dir) }
}

// 事件表不可用（模拟事件落库失败）：写入本体必须成功——尽力而为语义不改。
func TestSaveEventFailureFallbackKeepsFact(t *testing.T) {
	s, evlog, closeDB := txTestStore(t)
	defer closeDB()
	if _, err := evlog.DB.Exec(`DROP TABLE memory_events`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(Memory{Name: "keep-me", Description: "描述", Body: "正文"}); err != nil {
		t.Fatalf("Save 应在事件失败时回落成功: %v", err)
	}
	if _, ok := s.Get("keep-me"); !ok {
		t.Fatal("事件失败回落路径丢写入本体")
	}
	// 触达同理（回落路径不因事件缺失报错）。
	if err := s.Touch("keep-me"); err != nil {
		t.Fatalf("Touch 回落路径报错: %v", err)
	}
}

// 快乐路径：Save 一步提交 fact + 事件（op=save，name 对应）。
func TestSaveWritesEventAtomically(t *testing.T) {
	s, evlog, closeDB := txTestStore(t)
	defer closeDB()
	if _, err := s.Save(Memory{Name: "with-event", Description: "描述", Body: "正文"}); err != nil {
		t.Fatal(err)
	}
	events, err := evlog.LoadEvents()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if e.Op == OpSave && e.Name == "with-event" {
			found = true
		}
	}
	if !found {
		t.Fatalf("save 事件未落日志: %+v", events)
	}
}

// 未命中行的路径语义不变：Archive 不存在名=("", nil) 且零事件；Touch 不存在
// 名=nil 错误且零事件；Unarchive 不存在名=报错。
func TestMissPathsUnchanged(t *testing.T) {
	s, evlog, closeDB := txTestStore(t)
	defer closeDB()
	if _, err := s.Save(Memory{Name: "live", Description: "d", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if p, err := s.Archive("ghost"); err != nil || p != "" {
		t.Fatalf("Archive miss = (%q, %v), want (\"\", nil)", p, err)
	}
	if err := s.Touch("ghost"); err != nil {
		t.Fatalf("Touch miss 报错: %v", err)
	}
	if err := s.Unarchive("ghost"); err == nil {
		t.Fatal("Unarchive miss 应报错")
	}
	if err := s.ChangeType("ghost", TypeUser); err == nil {
		t.Fatal("ChangeType miss 应报错")
	}
	events, err := evlog.LoadEvents()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e.Name == "ghost" {
			t.Fatalf("未命中行不应留痕: %+v", e)
		}
	}
}
