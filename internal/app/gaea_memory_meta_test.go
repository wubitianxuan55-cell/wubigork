package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

func newOfficeMemoryTestEnv(t *testing.T) memory.Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	store := memory.SQLiteStoreFor(gdb, dir, dir)
	t.Cleanup(func() {
		db.CloseDatabase(dir)
		ResetOfficeStoreForTest()
	})
	SetOfficeStoreForTest(store)
	return store
}

func TestGaeaMemoryDuplicatesAndMerge(t *testing.T) {
	store := newOfficeMemoryTestEnv(t)
	a := &App{}
	if _, err := store.Save(memory.Memory{Name: "a", Title: "桩基施工要点", Description: "振动锤选型", Tags: []string{"桩基"}, Body: "要点A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(memory.Memory{Name: "b", Title: "桩基施工 要点", Description: "振动锤选型需匹配地质", Tags: []string{"振动锤", "桩基"}, Body: "要点B"}); err != nil {
		t.Fatal(err)
	}

	dups := a.GaeaMemoryDuplicates(0.5)
	if len(dups) != 1 {
		t.Fatalf("duplicates = %+v, want 1 对", dups)
	}
	if dups[0].Keep != "a" || dups[0].Dup != "b" {
		t.Errorf("duplicates = %+v, want a→b", dups)
	}

	merged, err := a.GaeaMemoryMerge("a", []string{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if merged != "a" {
		t.Errorf("merged = %q, want a", merged)
	}
	all := store.List()
	if len(all) != 1 || all[0].Name != "a" {
		t.Fatalf("after merge facts = %+v, want only a", all)
	}
	if len(all[0].Tags) != 2 || !strings.Contains(all[0].Body, "合并自") {
		t.Errorf("merged fact wrong: tags=%v body=%q", all[0].Tags, all[0].Body)
	}
}
