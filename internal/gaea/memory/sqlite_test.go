package memory

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestSQLiteStoreSaveListGet(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)

	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	if s.Index() != "" {
		t.Error("empty store should have empty index")
	}

	path, err := s.Save(Memory{
		Name:        "prefers-tabs",
		Title:       "Prefers tabs",
		Description: "User prefers tabs over spaces",
		Type:        TypeUser,
		Kind:        KindSemantic,
		Body:        "Use tabs for indentation.",
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if path == "" || !strings.HasPrefix(path, "hephaestus.db://") {
		t.Errorf("unexpected path: %q", path)
	}

	mems := s.List()
	if len(mems) != 1 || mems[0].Name != "prefers-tabs" || mems[0].Type != TypeUser {
		t.Fatalf("List = %+v", mems)
	}
	if m, ok := s.Get("prefers-tabs"); !ok || m.Body != "Use tabs for indentation." || m.Kind != KindSemantic {
		t.Errorf("Get = %+v ok=%v", m, ok)
	}
	if idx := s.Index(); idx == "" || !strings.Contains(idx, "prefers-tabs") {
		t.Errorf("Index = %q", idx)
	}

	// 覆盖保存（UPSERT 更新 body，不重复）
	if _, err := s.Save(Memory{Name: "prefers-tabs", Description: "Updated", Type: TypeUser, Body: "new body"}); err != nil {
		t.Fatal(err)
	}
	if mems := s.List(); len(mems) != 1 || mems[0].Body != "new body" {
		t.Errorf("upsert failed: %+v", mems)
	}
}

func TestSQLiteStoreArchiveChangeTypeDelete(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")

	mustSave := func(name, typ string) {
		t.Helper()
		if _, err := s.Save(Memory{Name: name, Description: "d", Type: Type(typ), Body: "b"}); err != nil {
			t.Fatal(err)
		}
	}
	mustSave("fact-a", "project")
	mustSave("fact-b", "user")

	if err := s.ChangeType("fact-a", TypeFeedback); err != nil {
		t.Fatal(err)
	}
	mems := s.List()
	for _, m := range mems {
		if m.Name == "fact-a" && m.Type != TypeFeedback {
			t.Errorf("ChangeType failed: %+v", m)
		}
	}

	if _, err := s.Archive("fact-b"); err != nil {
		t.Fatal(err)
	}
	if mems := s.List(); len(mems) != 1 || mems[0].Name != "fact-a" {
		t.Errorf("after archive List = %+v", mems)
	}
	arch := s.ListArchived()
	if len(arch) != 1 || arch[0].Name != "fact-b" || arch[0].ArchivedAt.IsZero() {
		t.Errorf("ListArchived = %+v", arch)
	}

	// Delete = archive
	if err := s.Delete("fact-a"); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 0 {
		t.Error("expected empty List after delete")
	}
	if len(s.ListArchived()) != 2 {
		t.Errorf("ListArchived = %d, want 2", len(s.ListArchived()))
	}
}

func TestSQLiteStorePerProjectIsolation(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)

	a := SQLiteStoreFor(gdb, dir, "/proj/a")
	b := SQLiteStoreFor(gdb, dir, "/proj/b")
	if _, err := a.Save(Memory{Name: "shared-name", Description: "in A", Body: "a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Save(Memory{Name: "shared-name", Description: "in B", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if la := a.List(); len(la) != 1 || la[0].Body != "a" {
		t.Errorf("project A = %+v", la)
	}
	if lb := b.List(); len(lb) != 1 || lb[0].Body != "b" {
		t.Errorf("project B = %+v", lb)
	}
}
