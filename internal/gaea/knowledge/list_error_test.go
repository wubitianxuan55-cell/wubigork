package knowledge

// 审计 P0#6（GA3-01）：后端读失败必须如实返回 error，不得吞成空切片。
// 「表空」（可读、无条目）与「读失败」必须可区分。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func newClosedDBStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	st, err := OpenSQLite(gdb)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.CloseDatabase(dir); err != nil {
		t.Fatalf("关闭 db: %v", err)
	}
	return st
}

func TestSQLiteListReportsReadFailure(t *testing.T) {
	st := newClosedDBStore(t)
	list, err := st.List()
	if err == nil {
		t.Fatalf("DB 读失败必须返回 error，实际 err=nil list=%+v（吞错成空切片）", list)
	}
	if list != nil {
		t.Errorf("读失败时不应返回条目，实际 %+v", list)
	}
}

func TestSQLiteReadAllReportsReadFailure(t *testing.T) {
	st := newClosedDBStore(t)
	all, err := st.ReadAll()
	if err == nil {
		t.Fatalf("DB 读失败必须返回 error，实际 err=nil all=%+v（吞错成空切片）", all)
	}
	if all != nil {
		t.Errorf("读失败时不应返回条目，实际 %+v", all)
	}
}

// 对照组：库可读但无条目 → 空结果 + nil error，不得报错。
func TestSQLiteListEmptyIsNotError(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer func() { _ = db.CloseDatabase(dir) }()
	st, err := OpenSQLite(gdb)
	if err != nil {
		t.Fatal(err)
	}
	list, err := st.List()
	if err != nil {
		t.Fatalf("表空不是错误: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("空表应返回 0 条，实际 %+v", list)
	}
	all, err := st.ReadAll()
	if err != nil {
		t.Fatalf("表空不是错误: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("空表应返回 0 条，实际 %+v", all)
	}
}

// file 后端同型：条目文件读失败（不可读）不得静默当空条目跳过。
func TestFileBackendReadAllReportsUnreadableEntry(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Save(Entry{Name: "ok", Title: "正常条目", Category: CatCase, Body: "正文"}); err != nil {
		t.Fatal(err)
	}
	// 与条目同扩展名的目录：os.ReadFile 必失败（读失败而非「不存在」）。
	if err := os.MkdirAll(filepath.Join(dir, "broken.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	all, err := st.ReadAll()
	if err == nil {
		t.Fatalf("存在不可读条目时 ReadAll 必须报错，实际 all=%+v", all)
	}
	if all != nil {
		t.Errorf("读失败时不应返回条目，实际 %+v", all)
	}
	list, err := st.List()
	if err == nil {
		t.Fatalf("存在不可读条目时 List 必须报错，实际 list=%+v", list)
	}
	if !strings.Contains(err.Error(), "broken.md") {
		t.Errorf("错误应点名读失败的文件，实际: %v", err)
	}
}

// 对照组：file 后端可读但无条目 → 空结果 + nil error。
func TestFileBackendListEmptyIsNotError(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	list, err := st.List()
	if err != nil {
		t.Fatalf("空知识库不是错误: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("空知识库应返回 0 条，实际 %+v", list)
	}
}
