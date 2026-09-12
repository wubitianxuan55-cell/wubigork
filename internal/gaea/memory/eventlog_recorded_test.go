package memory

// 双时间轴（SchemaV21）事件日志测试：AppendEvent 服务端盖章记录时间（调用方
// 传值一律被覆盖——事实时间可声明，记录时间不可伪造）、事实时间 At 保留调用
// 方声明、V21 前旧行（recorded_at=0）读取归一回落 At。

import (
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestAppendEventStampsRecordedAt(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	l := &EventLog{DB: gdb}
	// 事实时间声明在过去三天前（回填/导入形态），且恶意带上伪造的 RecordedAt
	past := time.Now().Add(-72 * time.Hour).UnixMilli()
	forged := time.Now().Add(72 * time.Hour).UnixMilli()
	if err := l.AppendEvent(Event{At: past, RecordedAt: forged, Op: OpSave, Name: "backfill-fact"}); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}

	events, err := l.LoadEvents()
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	e := events[0]
	if e.At != past {
		t.Errorf("事实时间 At = %d, want %d（调用方声明必须保留）", e.At, past)
	}
	now := time.Now().UnixMilli()
	if e.RecordedAt <= past || e.RecordedAt > now {
		t.Errorf("RecordedAt = %d, want 服务端盖章的写入时刻（%d, %d] 区间内", e.RecordedAt, past, now)
	}
	if e.RecordedAt == forged {
		t.Errorf("RecordedAt = 调用方伪造值，盖章被跳过")
	}
}

func TestLoadEventsNormalizesLegacyRow(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	// 直插一条 V21 前口径的旧行（recorded_at=0，绕过 AppendEvent 盖章）
	past := time.Now().Add(-24 * time.Hour).UnixMilli()
	if _, err := gdb.Exec(`INSERT INTO memory_events(at, op, name, recorded_at) VALUES(?, 'save', 'legacy-row', 0)`, past); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	events, err := (&EventLog{DB: gdb}).LoadEvents()
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	e := events[0]
	if e.RecordedAt != e.At {
		t.Errorf("旧行归一 RecordedAt = %d, want 回落 At=%d", e.RecordedAt, e.At)
	}
}

func TestAppendCiteEventsAlsoStamped(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	(&EventLog{DB: gdb}).AppendCiteEvents([]string{"hit-key"}, []string{"ghost-key"}, "work")

	events, err := (&EventLog{DB: gdb}).LoadEvents()
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	for _, e := range events {
		if e.RecordedAt == 0 {
			t.Errorf("op=%s name=%s RecordedAt=0，cite 路径盖章缺失", e.Op, e.Name)
		}
	}
}
