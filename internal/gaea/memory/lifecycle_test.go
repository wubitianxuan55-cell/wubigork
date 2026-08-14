package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
)

// T6-8.2 生命周期清理：SQLite 后端按归档时间硬删超期事实，活跃 List 不受影响。
func TestCleanupArchivedRemovesExpiredSQLite(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for _, n := range []string{"fact-old", "fact-mid", "fact-new"} {
		if _, err := s.Save(Memory{Name: n, Description: n + " desc", Body: "b"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{"fact-old", "fact-mid", "fact-new"} {
		if _, err := s.Archive(n); err != nil {
			t.Fatal(err)
		}
	}
	// 回拨归档时间：old=100 天前（超期）、mid=10 天前（未到期）、new=1 天前
	backdate := func(name string, days int) {
		t.Helper()
		ts := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
		if _, err := gdb.Exec(`UPDATE facts SET updated_at=? WHERE name=?`, ts, name); err != nil {
			t.Fatal(err)
		}
	}
	backdate("fact-old", 100)
	backdate("fact-mid", 10)
	backdate("fact-new", 1)

	removed, err := s.CleanupArchived(time.Now().Add(-ArchivedRetention))
	if err != nil {
		t.Fatalf("CleanupArchived: %v", err)
	}
	if len(removed) != 1 || removed[0].Name != "fact-old" {
		t.Fatalf("removed = %+v, want only fact-old", removed)
	}
	// 溯源字段随被删行返回（审计侧可用）
	if removed[0].Description != "fact-old desc" || removed[0].ArchivedAt.IsZero() {
		t.Fatalf("removed row provenance lost: %+v", removed[0])
	}

	arch := s.ListArchived()
	if len(arch) != 2 {
		t.Fatalf("ListArchived = %d, want 2 (mid + new)", len(arch))
	}
	for _, am := range arch {
		if am.Name == "fact-old" {
			t.Fatal("fact-old 不应仍在归档中")
		}
	}
	if len(s.List()) != 0 {
		t.Fatal("活跃列表不应包含归档事实")
	}
}

// T6-8.2 无超期条目时清理为空操作；异常空 updated_at 不误删。
func TestCleanupArchivedNoExpiredSQLite(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	if _, err := s.Save(Memory{Name: "fresh", Description: "d", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Archive("fresh"); err != nil {
		t.Fatal(err)
	}
	// 空 updated_at 的归档（异常数据）也保留
	if _, err := gdb.Exec(`UPDATE facts SET updated_at='' WHERE name='fresh'`); err != nil {
		t.Fatal(err)
	}

	removed, err := s.CleanupArchived(time.Now().Add(-ArchivedRetention))
	if err != nil {
		t.Fatalf("CleanupArchived: %v", err)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %+v, want none", removed)
	}
	if arch := s.ListArchived(); len(arch) != 1 {
		t.Fatalf("归档应保留, got %d", len(arch))
	}
}

// T6-8.2 分页：总量 + 翻页 + 越界 + limit 钳制。
func TestListArchivedPagedSQLite(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	for i := 0; i < 5; i++ {
		name := "fact-" + string(rune('a'+i))
		if _, err := s.Save(Memory{Name: name, Description: name, Body: "b"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Archive(name); err != nil {
			t.Fatal(err)
		}
		// 归档时间交错，验证倒序
		ts := time.Now().UTC().Add(-time.Duration(i) * 24 * time.Hour).Format(time.RFC3339)
		if _, err := gdb.Exec(`UPDATE facts SET updated_at=? WHERE name=?`, ts, name); err != nil {
			t.Fatal(err)
		}
	}

	page1, total, err := s.ListArchivedPaged(2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 || len(page1) != 2 {
		t.Fatalf("page1: len=%d total=%d, want 2/5", len(page1), total)
	}
	// 最新在前：fact-a 是 4 天前（最新）
	if page1[0].Name != "fact-a" || page1[1].Name != "fact-b" {
		t.Fatalf("page1 order = %s,%s", page1[0].Name, page1[1].Name)
	}

	page3, total3, err := s.ListArchivedPaged(2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if total3 != 5 || len(page3) != 1 || page3[0].Name != "fact-e" {
		t.Fatalf("page3: %+v total=%d", page3, total3)
	}

	beyond, totalB, err := s.ListArchivedPaged(2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(beyond) != 0 || totalB != 5 {
		t.Fatalf("beyond: len=%d total=%d, want 0/5", len(beyond), totalB)
	}

	// limit 钳制：<=0 → 默认 50；>200 → 200
	big, _, err := s.ListArchivedPaged(0, 0)
	if err != nil || len(big) != 5 {
		t.Fatalf("default limit: %d err=%v, want 5", len(big), err)
	}
	big2, _, err := s.ListArchivedPaged(500, 0)
	if err != nil || len(big2) != 5 {
		t.Fatalf("clamped limit: %d err=%v, want 5", len(big2), err)
	}
}

// T6-8.2 文件后端清理：.archive 下超期文件被删、未到期保留。
func TestCleanupArchivedFileBackend(t *testing.T) {
	storeDir := t.TempDir()
	s := Store{Dir: storeDir}
	if _, err := s.Save(Memory{Name: "keep", Description: "keep", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(Memory{Name: "gone", Description: "gone", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Archive("keep"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Archive("gone"); err != nil {
		t.Fatal(err)
	}

	// 手工把 gone 的归档文件时间戳前缀改成 100 天前
	archiveDir := filepath.Join(storeDir, ".archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "gone.md") {
			old := filepath.Join(archiveDir, e.Name())
			ts := time.Now().UTC().Add(-100 * 24 * time.Hour).Format("20060102-150405.000")
			neu := filepath.Join(archiveDir, ts+"-gone.md")
			if err := os.Rename(old, neu); err != nil {
				t.Fatal(err)
			}
		}
	}

	removed, err := s.CleanupArchived(time.Now().Add(-ArchivedRetention))
	if err != nil {
		t.Fatalf("CleanupArchived: %v", err)
	}
	if len(removed) != 1 || removed[0].Name != "gone" {
		t.Fatalf("removed = %+v, want gone only", removed)
	}
	arch := s.ListArchived()
	if len(arch) != 1 || arch[0].Name != "keep" {
		t.Fatalf("archived = %+v, want keep only", arch)
	}
}

// T6-8.2 文件后端分页：与 sqlite 同语义。
func TestListArchivedPagedFileBackend(t *testing.T) {
	storeDir := t.TempDir()
	s := Store{Dir: storeDir}
	for i := 0; i < 3; i++ {
		name := "f" + string(rune('a'+i))
		if _, err := s.Save(Memory{Name: name, Description: name, Body: "b"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Archive(name); err != nil {
			t.Fatal(err)
		}
	}
	page, total, err := s.ListArchivedPaged(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(page) != 2 {
		t.Fatalf("page: len=%d total=%d, want 2/3", len(page), total)
	}
	if _, total2, err := s.ListArchivedPaged(10, 0); err != nil || total2 != 3 {
		t.Fatalf("full page: total=%d err=%v", total2, err)
	}
}
