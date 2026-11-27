package builtin

// movefile_version_test.go — 2026-11-27 AI 反馈 #4 回归钉：move_file 覆盖正式交付面的
// Office 件时，旧文件留一份快照到 .gaea/versions/（按原相对路径分目录、文件名是
// 旧文件的修改时间），回执给快照路径；非 Office 件与过程件不留。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMoveFileOverwriteKeepsDurableVersion(t *testing.T) {
	ws := filepath.Join(t.TempDir(), "ws")
	exports := filepath.Join(ws, "交付")
	work := filepath.Join(ws, ".gaea", "work")
	for _, d := range []string{exports, work} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m := moveFile{roots: []string{ws}, workDir: ws}
	ctx := context.Background()
	oldTime := time.Date(2026, 11, 26, 21, 45, 0, 0, time.Local)

	// 第一版落正式名，再用第二版覆盖。
	formal := filepath.Join(exports, "测算.xlsx")
	if err := os.WriteFile(formal, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(formal, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(work, "incoming.xlsx")
	if err := os.WriteFile(src, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := m.Execute(ctx, toJSON(t, map[string]any{"source": ".gaea/work/incoming.xlsx", "destination": "交付/测算.xlsx", "overwrite": true}))
	if err != nil {
		t.Fatalf("覆盖失败：%v（out=%s）", err, out)
	}
	snap := filepath.Join(ws, ".gaea", "versions", "交付", "测算", "20261126-214500.xlsx")
	if raw, err := os.ReadFile(snap); err != nil || string(raw) != "v1" {
		t.Fatalf("覆盖前的旧文件应留快照：%q（%v）", string(raw), err)
	}
	if raw, _ := os.ReadFile(formal); string(raw) != "v2" {
		t.Fatalf("正式名应是新文件：%q", string(raw))
	}
	if !strings.Contains(out, "上一版已留快照：.gaea/versions/交付/测算/20261126-214500.xlsx") {
		t.Errorf("回执应给快照路径：%s", out)
	}

	// 再覆盖一次：两份快照并列保留。
	v3 := filepath.Join(work, "incoming3.xlsx")
	if err := os.WriteFile(v3, []byte("v3"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Execute(ctx, toJSON(t, map[string]any{"source": ".gaea/work/incoming3.xlsx", "destination": "交付/测算.xlsx", "overwrite": true})); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(snap))
	if err != nil || len(entries) != 2 {
		t.Fatalf("两次覆盖应有两份快照：%v（%v）", entries, err)
	}
	if got := latestVersionSnapshot(ws, "交付/测算.xlsx"); !strings.HasSuffix(got, ".xlsx") || got == ".gaea/versions/交付/测算/20261126-214500.xlsx" {
		t.Errorf("最近一份快照应是第二次覆盖留下的：%q", got)
	}

	// 非 Office 件不留。
	if err := os.WriteFile(filepath.Join(exports, "说明.md"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "notes.md"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = m.Execute(ctx, toJSON(t, map[string]any{"source": ".gaea/work/notes.md", "destination": "交付/说明.md", "overwrite": true}))
	if err != nil || strings.Contains(out, "快照") {
		t.Errorf("非 Office 件覆盖不留快照：err=%v out=%s", err, out)
	}
	// 过程件不留。
	if err := os.WriteFile(filepath.Join(work, "draft.xlsx"), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "draft2.xlsx"), []byte("d"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = m.Execute(ctx, toJSON(t, map[string]any{"source": ".gaea/work/draft2.xlsx", "destination": ".gaea/work/draft.xlsx", "overwrite": true}))
	if err != nil || strings.Contains(out, "快照") {
		t.Errorf("过程目录里的件覆盖不留快照：err=%v out=%s", err, out)
	}
}
