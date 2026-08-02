package memory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// setupLegacyFiles 构造一个旧版 Markdown 记忆目录（1 条 active + 1 条 archived）。
func setupLegacyFiles(t *testing.T, userDir string) string {
	t.Helper()
	memDir := filepath.Join(userDir, "projects", "-Users-me-proj", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// active
	active := `---
name: prefers-tabs
title: Prefers tabs
description: User prefers tabs over spaces
metadata:
  type: user
---
Use tabs for indentation.
`
	if err := os.WriteFile(filepath.Join(memDir, "prefers-tabs.md"), []byte(active), 0o644); err != nil {
		t.Fatal(err)
	}
	// archived（放 .archive/）
	archiveDir := filepath.Join(memDir, ".archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	archived := `---
name: old-fact
title: Old fact
description: A stale fact
metadata:
  type: project
---
Stale body.
`
	if err := os.WriteFile(filepath.Join(archiveDir, "20260101-000000.000-old-fact.md"), []byte(archived), 0o644); err != nil {
		t.Fatal(err)
	}
	return memDir
}

func TestMigrateLegacyFileMemory(t *testing.T) {
	userDir := t.TempDir()
	setupLegacyFiles(t, userDir)
	gdb := db.GetDatabase(userDir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(userDir)

	n, err := MigrateLegacyFileMemory(userDir, gdb)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if n != 1 {
		t.Errorf("migrated projects = %d, want 1", n)
	}

	s := SQLiteStoreForProject(gdb, "-Users-me-proj")
	mems := s.List()
	if len(mems) != 1 || mems[0].Name != "prefers-tabs" || mems[0].Type != TypeUser {
		t.Errorf("List = %+v", mems)
	}
	if m, ok := s.Get("prefers-tabs"); !ok || m.Body != "Use tabs for indentation." {
		t.Errorf("Get active = %+v ok=%v", m, ok)
	}
	arch := s.ListArchived()
	if len(arch) != 1 || arch[0].Name != "old-fact" {
		t.Errorf("ListArchived = %+v", arch)
	}

	// 幂等：第二次调用应跳过（profile 标记）
	if _, err := MigrateLegacyFileMemory(userDir, gdb); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 1 || len(s.ListArchived()) != 1 {
		t.Error("second migrate should be a no-op")
	}

	// 标记已写入
	var marker string
	if err := gdb.QueryRow("SELECT value FROM profile WHERE key='legacy_memory_migrated'").Scan(&marker); err != nil || marker == "" {
		t.Errorf("migration marker missing: %q err=%v", marker, err)
	}
}

func TestMigrateLegacyFileMemoryNoProjects(t *testing.T) {
	userDir := t.TempDir()
	gdb := db.GetDatabase(userDir)
	defer db.CloseDatabase(userDir)

	n, err := MigrateLegacyFileMemory(userDir, gdb)
	if err != nil || n != 0 {
		t.Errorf("empty migrate: n=%d err=%v", n, err)
	}
	var marker string
	if err := gdb.QueryRow("SELECT value FROM profile WHERE key='legacy_memory_migrated'").Scan(&marker); err != nil || marker == "" {
		t.Errorf("marker should be written for fresh install: %q err=%v", marker, err)
	}
}
