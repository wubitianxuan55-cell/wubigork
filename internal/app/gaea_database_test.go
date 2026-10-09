package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	whisperdb "github.com/gaea/gaea/internal/whisper/db"
)

// seedDatabaseRoot 在临时工作区造会话/附件存档，返回根路径。
// 调用方须先 workspaceTestIsolate 并把 ga.cfg.Workspace 指向该根。
func seedDatabaseRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	sess := filepath.Join(root, ".gaea", "sessions", "work")
	if err := os.MkdirAll(sess, 0o755); err != nil {
		t.Fatal(err)
	}
	arch := filepath.Join(root, ".gaea", "sessions", "work", "archive")
	if err := os.MkdirAll(arch, 0o755); err != nil {
		t.Fatal(err)
	}
	ups := filepath.Join(root, ".gaea", "uploads")
	if err := os.MkdirAll(ups, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(p, body string) {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(sess, "s1.jsonl"), strings.Repeat("a", 100))
	write(filepath.Join(sess, "s2.jsonl"), strings.Repeat("b", 50))
	write(filepath.Join(arch, "old.jsonl"), strings.Repeat("c", 30))
	write(filepath.Join(ups, "attach-1-report.pdf"), strings.Repeat("d", 40))
	write(filepath.Join(ups, "paste-2.png"), strings.Repeat("e", 20))
	// 非 .jsonl 不应计入会话
	write(filepath.Join(sess, "notes.txt"), "x")
	return root
}

// withDatabaseWorkspace 隔离环境并把当前工作区指向种子根。
func withDatabaseWorkspace(t *testing.T) string {
	t.Helper()
	restore := workspaceTestIsolate(t)
	t.Cleanup(restore)
	root := seedDatabaseRoot(t)
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: root}
	t.Cleanup(func() { ga.cfg = oldCfg })
	return root
}

// GaeaDatabaseOverview：跨工作区计数/字节/归档拆分；非 .jsonl 不计。
func TestGaeaDatabaseOverview(t *testing.T) {
	root := withDatabaseWorkspace(t)

	a := &App{}
	ov := a.GaeaDatabaseOverview()
	if len(ov.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(ov.Projects))
	}
	p := ov.Projects[0]
	if !p.Current || p.Name == "" {
		t.Fatalf("project = %+v", p)
	}
	if p.SessionCount != 2 || p.ArchivedCount != 1 {
		t.Fatalf("sessions = %d/%d, want 2/1（归档拆分）", p.SessionCount, p.ArchivedCount)
	}
	if p.SessionBytes != 180 {
		t.Fatalf("sessionBytes = %d, want 180", p.SessionBytes)
	}
	if p.UploadCount != 2 || p.UploadBytes != 60 {
		t.Fatalf("uploads = %d/%d, want 2/60", p.UploadCount, p.UploadBytes)
	}
	if ov.TotalSessions != 2 || ov.TotalArchived != 1 || ov.TotalUploads != 2 || ov.TotalBytes != 240 {
		t.Fatalf("totals = %+v", ov)
	}
	if p.LatestMod <= 0 {
		t.Fatalf("latestMod = %d, want >0（会话+附件 mtime 最大值）", p.LatestMod)
	}
	_ = root
}

// GaeaUploadsList：白名单 root 之外拒绝（返回空）；kind 按前缀分类；新→旧。
func TestGaeaUploadsList(t *testing.T) {
	root := withDatabaseWorkspace(t)

	a := &App{}
	rows := a.GaeaUploadsList(root)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0].ModTime < rows[1].ModTime {
		t.Fatalf("顺序错：应新→旧")
	}
	kinds := map[string]bool{}
	for _, r := range rows {
		kinds[r.Kind] = true
		if !filepath.IsAbs(r.Path) || !strings.Contains(filepath.ToSlash(r.Path), ".gaea/uploads") {
			t.Fatalf("row path = %s", r.Path)
		}
	}
	if !kinds["attach"] || !kinds["paste"] {
		t.Fatalf("kinds = %v, want attach+paste", kinds)
	}
	// 白名单外 root：空列表（不报错——绑定面无 error 出口）
	if rows := a.GaeaUploadsList(t.TempDir()); len(rows) != 0 {
		t.Fatalf("非白名单 root 应返回空，got %d", len(rows))
	}
}

// GaeaDeleteUpload：白名单内可删；uploads 外/不存在/空串拒。
func TestGaeaDeleteUpload(t *testing.T) {
	root := withDatabaseWorkspace(t)

	a := &App{}
	target := filepath.Join(root, ".gaea", "uploads", "attach-1-report.pdf")
	if err := a.GaeaDeleteUpload(target); err != nil {
		t.Fatalf("delete upload: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("文件应已删除")
	}
	// uploads 外（工作区内其它位置）拒绝
	outside := filepath.Join(root, ".gaea", "sessions", "work", "s1.jsonl")
	if err := a.GaeaDeleteUpload(outside); err == nil {
		t.Fatal("uploads 外路径应拒绝")
	}
	// 任意绝对路径拒绝
	if err := a.GaeaDeleteUpload(filepath.Join(t.TempDir(), "x.txt")); err == nil {
		t.Fatal("白名单外路径应拒绝")
	}
	// 空串拒绝
	if err := a.GaeaDeleteUpload(""); err == nil {
		t.Fatal("空路径应拒绝")
	}
	// 白名单内但不存在的文件拒绝
	if err := a.GaeaDeleteUpload(filepath.Join(root, ".gaea", "uploads", "nope.bin")); err == nil {
		t.Fatal("不存在的文件应拒绝")
	}
}

// MemoryHubOverview 新计数：sessionCount/uploadCount 随数据库存档走。
// 注意：①whisperDataRoot 经 whisperState 指针提升——裸 App 必须构造
// whisperState（nil 解引用，characterlib_handler_test 同款）；②总览会经
// GetDatabase 单例打开 Hephaestus.db/hermes.db——cleanup LIFO 先关库再删
// t.TempDir（Windows 锁，否则 RemoveAll panic）。
func TestMemoryHubOverviewDatabaseCounts(t *testing.T) {
	withDatabaseWorkspace(t)
	t.Cleanup(func() { _ = db.CloseDatabase(gaeaConfig.MemoryUserDir()) })

	root := t.TempDir()
	t.Cleanup(func() { _ = whisperdb.CloseDatabase(root) })
	a := &App{core: &core{}, whisperState: &whisperState{core: &core{}, app: &App{}, whisperDataRoot: root}}
	ov := a.GaeaMemoryHubOverview()
	if ov.SessionCount != 3 {
		t.Fatalf("sessionCount = %d, want 3（活跃 2+归档 1）", ov.SessionCount)
	}
	if ov.UploadCount != 2 {
		t.Fatalf("uploadCount = %d, want 2", ov.UploadCount)
	}
}
