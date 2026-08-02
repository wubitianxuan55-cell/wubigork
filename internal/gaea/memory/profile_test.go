package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func TestProfileStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)
	ps := NewProfileStore(gdb)

	m := Memory{Name: "prefers-tabs", Title: "Prefers tabs", Description: "User prefers tabs", Type: TypeUser, Kind: KindSemantic, Body: "Use tabs."}
	if err := ps.Save(m); err != nil {
		t.Fatal(err)
	}
	got, ok := ps.Get("prefers-tabs")
	if !ok || got.Description != "User prefers tabs" || got.Body != "Use tabs." {
		t.Errorf("Get = %+v ok=%v", got, ok)
	}
	if !ps.Has("prefers-tabs") {
		t.Error("Has should be true")
	}
	all := ps.All()
	if len(all) != 1 || all[0].Name != "prefers-tabs" {
		t.Errorf("All = %+v", all)
	}
	// 覆盖更新
	if err := ps.Save(Memory{Name: "prefers-tabs", Description: "Updated", Type: TypeUser, Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if all := ps.All(); len(all) != 1 || all[0].Description != "Updated" {
		t.Errorf("upsert failed: %+v", all)
	}
	if err := ps.Delete("prefers-tabs"); err != nil {
		t.Fatal(err)
	}
	if _, ok := ps.Get("prefers-tabs"); ok {
		t.Error("deleted profile still exists")
	}
}

func TestRememberRoutesUserTypeToProfile(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)

	fileStore := Store{Dir: t.TempDir()}
	tl := NewRememberTool(fileStore, NewProfileStore(gdb))

	// type=user → 主脑画像（不落 facts）
	userArgs, _ := json.Marshal(map[string]string{"name": "prefers-tabs", "description": "User prefers tabs", "body": "Use tabs.", "type": "user"})
	if _, err := tl.Execute(context.Background(), userArgs); err != nil {
		t.Fatal(err)
	}
	if len(fileStore.List()) != 0 {
		t.Error("user fact should NOT be written to project facts")
	}
	if _, ok := NewProfileStore(gdb).Get("prefers-tabs"); !ok {
		t.Error("user fact should be written to main-brain profile")
	}

	// type=feedback → 办公 facts（不进画像）
	fbArgs, _ := json.Marshal(map[string]string{"name": "check-units", "description": "Always verify units", "body": "Check SI units.", "type": "feedback"})
	if _, err := tl.Execute(context.Background(), fbArgs); err != nil {
		t.Fatal(err)
	}
	if len(fileStore.List()) != 1 || fileStore.List()[0].Name != "check-units" {
		t.Errorf("feedback fact should be in facts: %+v", fileStore.List())
	}
	if _, ok := NewProfileStore(gdb).Get("check-units"); ok {
		t.Error("feedback fact should NOT be in profile")
	}
}

func TestProfileBlockAggregatesFromProfile(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	defer db.CloseDatabase(dir)

	// 主脑画像一条 + 旧 facts 一条（user）→ 都显示
	if err := NewProfileStore(gdb).Save(Memory{Name: "from-profile", Description: "From profile", Type: TypeUser, Body: "b"}); err != nil {
		t.Fatal(err)
	}
	fileStore := Store{Dir: t.TempDir()}
	if _, err := fileStore.Save(Memory{Name: "from-facts", Description: "From facts", Type: TypeUser, Body: "b"}); err != nil {
		t.Fatal(err)
	}
	set := &Set{DB: gdb, Store: fileStore}
	block := set.ProfileBlock()
	for _, want := range []string{"From profile", "From facts", "User Profile"} {
		if !strings.Contains(block, want) {
			t.Errorf("ProfileBlock missing %q: %s", want, block)
		}
	}
}
