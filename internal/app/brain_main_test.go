package app

import (
	"testing"

	gaeadb "github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

func TestMainBrainProfileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gdb := gaeadb.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("gaea db unavailable")
	}
	defer gdb.Close()
	mb := &mainBrain{profile: memory.NewProfileStore(gdb), kb: nil}
	if err := mb.Write("甲方A", "偏好", "保守报价"); err != nil {
		t.Fatal(err)
	}
	hits, err := mb.Search("保守报价")
	if err != nil || len(hits) == 0 {
		t.Fatalf("hits=%+v err=%v", hits, err)
	}
	if hits[0].Entity != "甲方A" {
		t.Fatalf("entity = %q", hits[0].Entity)
	}
}

func TestLinkStoreSQLiteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gdb := gaeadb.GetDatabase(dir)
	defer gdb.Close()
	ls := NewLinkStore(gdb)
	if err := ls.Add("甲方A", BrainRight, "fact-1"); err != nil {
		t.Fatal(err)
	}
	if err := ls.Add("甲方A", BrainLeft, "proposal:p1"); err != nil {
		t.Fatal(err)
	}
	refs, err := ls.ListByEntity("甲方A")
	if err != nil || len(refs) != 2 {
		t.Fatalf("refs=%+v err=%v", refs, err)
	}
}
