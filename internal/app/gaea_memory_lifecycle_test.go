package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// T6-8.2 归档清理绑定：超期归档硬删（溯源审计落盘），活跃事实不受影响。
func TestGaeaMemoryCleanupArchived(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for _, n := range []string{"old-fact", "recent-fact"} {
		if _, err := s.Save(memory.Memory{Name: n, Description: n + " desc", Body: "b"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Archive(n); err != nil {
			t.Fatal(err)
		}
	}
	// 回拨 old-fact 归档时间到 100 天前（超期）
	ts := time.Now().UTC().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := gdb.Exec(`UPDATE facts SET updated_at=? WHERE name=?`, ts, "old-fact"); err != nil {
		t.Fatal(err)
	}

	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()
	t.Setenv("GAEA_DATA_ROOT", dir)

	a := &App{}
	n, err := a.GaeaMemoryCleanupArchived()
	if err != nil {
		t.Fatalf("CleanupArchived: %v", err)
	}
	if n != 1 {
		t.Fatalf("cleaned = %d, want 1", n)
	}
	// 活跃 List 不受影响（本来就没有活跃事实）；归档只剩 recent-fact
	if arch := s.ListArchived(); len(arch) != 1 || arch[0].Name != "recent-fact" {
		t.Fatalf("archived = %+v, want recent-fact only", arch)
	}
	// 溯源审计落盘
	path := filepath.Join(dir, "memory", "purge-audit.jsonl")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("purge audit missing: %v", err)
	}
	if !strings.Contains(string(b), "old-fact") || !strings.Contains(string(b), "old-fact desc") {
		t.Fatalf("purge audit lacks provenance: %s", b)
	}
	// 幂等：再次清理返回 0
	n2, err := a.GaeaMemoryCleanupArchived()
	if err != nil || n2 != 0 {
		t.Fatalf("second cleanup: n=%d err=%v, want 0/nil", n2, err)
	}
}

// T6-8.2 归档分页绑定：总量 + 分页条目 + 越界空页。
func TestGaeaMemoryArchivedListPaged(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for i := 0; i < 3; i++ {
		name := "fact-" + string(rune('a'+i))
		if _, err := s.Save(memory.Memory{Name: name, Description: name, Body: "b"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Archive(name); err != nil {
			t.Fatal(err)
		}
	}
	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()

	a := &App{}
	page, err := a.GaeaMemoryArchivedList(2, 0)
	if err != nil {
		t.Fatalf("ArchivedList: %v", err)
	}
	if page.Total != 3 || len(page.Items) != 2 {
		t.Fatalf("page: %d items, total %d, want 2/3", len(page.Items), page.Total)
	}
	if page.Items[0].Name == "" || page.Items[0].ArchivedAt == "" {
		t.Fatalf("item view incomplete: %+v", page.Items[0])
	}

	page2, err := a.GaeaMemoryArchivedList(2, 4)
	if err != nil || len(page2.Items) != 0 || page2.Total != 3 {
		t.Fatalf("beyond page: %+v err=%v", page2, err)
	}
}

// 记忆统一层第一刀：归档分页绑定携带保留期天数（前端「归档保留 N 天」文案）。
func TestGaeaMemoryArchivedList_RetentionDays(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()

	a := &App{}
	page, err := a.GaeaMemoryArchivedList(10, 0)
	if err != nil {
		t.Fatalf("ArchivedList: %v", err)
	}
	if page.RetentionDays != 90 {
		t.Fatalf("RetentionDays = %d, want 90（与 memory.ArchivedRetention 一致）", page.RetentionDays)
	}
}

// 记忆统一层第一刀：GaeaMemoryUnarchive 恢复归档事实回活跃列表；
// 未归档/不存在返回错误；恢复后归档列表减一。
func TestGaeaMemoryUnarchive(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	if _, err := s.Save(memory.Memory{Name: "fact-a", Description: "d", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Archive("fact-a"); err != nil {
		t.Fatal(err)
	}
	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()

	a := &App{}
	if err := a.GaeaMemoryUnarchive("fact-a"); err != nil {
		t.Fatalf("Unarchive: %v", err)
	}
	if list := s.List(); len(list) != 1 || list[0].Name != "fact-a" {
		t.Fatalf("恢复后活跃列表: %+v", list)
	}
	if arch := s.ListArchived(); len(arch) != 0 {
		t.Fatalf("恢复后归档应为空: %+v", arch)
	}

	// 未归档（活跃事实）恢复报错
	if err := a.GaeaMemoryUnarchive("fact-a"); err == nil {
		t.Fatal("活跃事实的恢复应报错")
	}
	// 不存在的事实恢复报错
	if err := a.GaeaMemoryUnarchive("nope"); err == nil {
		t.Fatal("不存在事实的恢复应报错")
	}
}
