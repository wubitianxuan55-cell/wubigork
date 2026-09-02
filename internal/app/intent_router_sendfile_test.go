package app

// intent_router_sendfile_test.go — 产物推送执行层（v4.41 微信文件收发刀）：
// 登记表命中取最新、登记表落空回退 exports 双目录、双落空诚实报错、dry-run
// 预览。会话枚举/exports 根均经可替换 seam（wxEditImageInvoker 先例）注入，
// 不依赖真实工作区。

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// withWxDeliverableSeams 临时替换产物查找 seam（测毕恢复）。
func withWxDeliverableSeams(t *testing.T, sessions, exportDirs []string) {
	t.Helper()
	origS, origE := wxDeliverableSessionPaths, wxDeliverableExportDirs
	wxDeliverableSessionPaths = func(a *App) []string { return sessions }
	wxDeliverableExportDirs = func(a *App) []string { return exportDirs }
	t.Cleanup(func() {
		wxDeliverableSessionPaths = origS
		wxDeliverableExportDirs = origE
	})
}

// seedSessionWithDeliverable 在临时目录写一个真实事件日志（tool_dispatch:
// write_file 登记产物），返回会话路径。deliverable 必须真实存在（登记表
// 口径：只认解析后真实存在的本地文件）。
func seedSessionWithDeliverable(t *testing.T, deliverable string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sessions", "work")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("建会话目录: %v", err)
	}
	args, _ := json.Marshal(map[string]string{"path": deliverable, "content": "x"})
	entry, _ := json.Marshal(map[string]interface{}{
		"seq": 1, "ts": time.Now().Unix(), "kind": "tool_dispatch",
		"payload": map[string]string{"name": "write_file", "args": string(args)},
	})
	logPath := filepath.Join(dir, "s-1.gaea-log.jsonl")
	if err := os.WriteFile(logPath, append(entry, '\n'), 0o644); err != nil {
		t.Fatalf("写事件日志: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "s-1.jsonl"), nil, 0o644); err != nil {
		t.Fatalf("写会话文件: %v", err)
	}
	if lp := session.LogPathFor(filepath.Join(dir, "s-1.jsonl")); lp != logPath {
		t.Fatalf("LogPathFor 口径变化: %q", lp)
	}
	return filepath.Join(dir, "s-1.jsonl")
}

// 登记表命中：最新会话的产物登记（真实事件日志折叠）→ CardPath + 已发送文案。
func TestExecSendLatestFile_RegistryHit(t *testing.T) {
	a := newChatServiceTestApp(t)
	root := t.TempDir()
	// 两个产物：登记表按 updatedAt 倒序，最新登记的是 second.txt（同 ts 按
	// path 字典序稳定性不参与——ts 显式错开）。
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	for _, p := range []string{first, second} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("写产物: %v", err)
		}
	}
	dir := filepath.Join(t.TempDir(), "sessions", "work")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("建会话目录: %v", err)
	}
	var lines [][]byte
	mk := func(path string, ts int64) []byte {
		args, _ := json.Marshal(map[string]string{"path": path})
		b, _ := json.Marshal(map[string]interface{}{
			"seq": len(lines) + 1, "ts": ts, "kind": "tool_dispatch",
			"payload": map[string]string{"name": "write_file", "args": string(args)},
		})
		return append(b, '\n')
	}
	lines = append(lines, mk(first, 1000), mk(second, 2000))
	if err := os.WriteFile(filepath.Join(dir, "s-1.gaea-log.jsonl"), bytes.Join(lines, nil), 0o644); err != nil {
		t.Fatalf("写事件日志: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "s-1.jsonl"), nil, 0o644); err != nil {
		t.Fatalf("写会话文件: %v", err)
	}
	withWxDeliverableSeams(t, []string{filepath.Join(dir, "s-1.jsonl")}, []string{filepath.Join(root, "exports")})

	res := a.routeIntentWithResultForAssistant("把最新的报告发给我", "ast-1")
	if !res.Handled {
		t.Fatalf("应命中: %+v", res)
	}
	if res.CardPath != second {
		t.Fatalf("CardPath = %q, want 最新登记 %q", res.CardPath, second)
	}
	if !strings.HasPrefix(res.Reply, "已发送：second.txt") {
		t.Fatalf("Reply = %q", res.Reply)
	}
	if res.Action != "send_latest_file" {
		t.Fatalf("Action = %q", res.Action)
	}
}

// 登记表里登记的文件已不存在（被清理）→ 跳过，回退 exports 扫描。
func TestExecSendLatestFile_RegistryStaleFallsBackExports(t *testing.T) {
	a := newChatServiceTestApp(t)
	dead := seedSessionWithDeliverable(t, filepath.Join(t.TempDir(), "gone.txt")) // 不落盘 → 悬空登记
	exp := filepath.Join(t.TempDir(), "exports")
	if err := os.MkdirAll(exp, 0o755); err != nil {
		t.Fatalf("建 exports: %v", err)
	}
	live := filepath.Join(exp, "汇总.xlsx")
	if err := os.WriteFile(live, []byte("x"), 0o644); err != nil {
		t.Fatalf("写产物: %v", err)
	}
	withWxDeliverableSeams(t, []string{dead}, []string{exp})

	res := a.routeIntentWithResultForAssistant("发我最新产物", "ast-1")
	if !res.Handled || res.CardPath != live {
		t.Fatalf("应回退 exports 命中: %+v", res)
	}
}

// 登记表空 → 回退扫 exports（work + play 两根），取 mtime 最新。
func TestExecSendLatestFile_EmptyRegistryFallsBackExportsNewest(t *testing.T) {
	a := newChatServiceTestApp(t)
	workExp := filepath.Join(t.TempDir(), "exports")
	playExp := filepath.Join(t.TempDir(), "play", "exports")
	for _, d := range []string{workExp, playExp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("建 exports: %v", err)
		}
	}
	oldFile := filepath.Join(workExp, "旧报告.docx")
	newFile := filepath.Join(playExp, "新报告.docx")
	for _, p := range []string{oldFile, newFile} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("写产物: %v", err)
		}
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(oldFile, past, past); err != nil {
		t.Fatalf("设 mtime: %v", err)
	}
	withWxDeliverableSeams(t, nil, []string{workExp, playExp})

	res := a.routeIntentWithResultForAssistant("发送产物", "ast-1")
	if !res.Handled || res.CardPath != newFile {
		t.Fatalf("应取 mtime 最新的 play 产物: %+v", res)
	}
}

// 双落空（无会话登记、exports 也空）→ Handled:true 诚实回复「暂无」。
func TestExecSendLatestFile_NoneHonest(t *testing.T) {
	a := newChatServiceTestApp(t)
	missing := filepath.Join(t.TempDir(), "no-such-exports")
	withWxDeliverableSeams(t, nil, []string{missing})

	res := a.routeIntentWithResultForAssistant("把刚才的文件发给我", "ast-1")
	if !res.Handled {
		t.Fatalf("查无产物也应命中并诚实回复: %+v", res)
	}
	if res.CardPath != "" {
		t.Fatalf("不应有卡片路径: %+v", res)
	}
	if !strings.Contains(res.Reply, "暂无可发送的产物") {
		t.Fatalf("Reply = %q", res.Reply)
	}
}

// dry-run 预览：有产物出「将发送最新产物：<文件名>」，无产物出「暂无可发送的
// 产物」；预览只读、零副作用（不落卡）。
func TestPreviewSendLatestFile(t *testing.T) {
	a := newChatServiceTestApp(t)
	exp := filepath.Join(t.TempDir(), "exports")
	if err := os.MkdirAll(exp, 0o755); err != nil {
		t.Fatalf("建 exports: %v", err)
	}
	live := filepath.Join(exp, "成果.pdf")
	if err := os.WriteFile(live, []byte("x"), 0o644); err != nil {
		t.Fatalf("写产物: %v", err)
	}
	withWxDeliverableSeams(t, nil, []string{exp})

	pre := a.GaeaRouteIntent("发送产物", true)
	if !pre.Handled || !strings.Contains(pre.Reply, "将发送最新产物：成果.pdf") {
		t.Fatalf("dry-run 预览不符: %+v", pre)
	}
	if pre.CardPath != "" {
		t.Fatalf("dry-run 不得携带卡片: %+v", pre)
	}

	withWxDeliverableSeams(t, nil, []string{filepath.Join(t.TempDir(), "empty")})
	none := a.GaeaRouteIntent("发送产物", true)
	if !none.Handled || !strings.Contains(none.Reply, "暂无可发送的产物") {
		t.Fatalf("无产物预览应诚实: %+v", none)
	}
}

// resolveWorkspacePath：绝对路径原样；相对路径按工作区解析；.. 逃逸拒绝。
func TestResolveWorkspacePath(t *testing.T) {
	cwd := `C:\ws`
	if got := resolveWorkspacePath(cwd, `C:\out\报告.docx`); got != `C:\out\报告.docx` {
		t.Errorf("绝对路径应原样: %q", got)
	}
	if got := resolveWorkspacePath(cwd, `.gaea/exports/a.xlsx`); got != `C:\ws\.gaea\exports\a.xlsx` {
		t.Errorf("相对路径应按工作区解析: %q", got)
	}
	if got := resolveWorkspacePath(cwd, `..\escape.txt`); got != "" {
		t.Errorf(".. 逃逸应拒绝: %q", got)
	}
	if got := resolveWorkspacePath(cwd, "   "); got != "" {
		t.Errorf("空白应拒绝: %q", got)
	}
}
