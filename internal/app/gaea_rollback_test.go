package app

// v4.32 线A 收 v4.28 B1 欠账：GaeaRollbackRecord 恢复写盘前先把目标当前
// 内容快照（恢复动作自身成为带基线的完整证据卡，时间线里可「再恢复」=
// 撤销恢复）；target 不存在/快照失败降级不阻断。
// journal 搭建沿用 channel B 测试模式：t.TempDir + isolateWorkspaceTo 配置
// 隔离，无进程级全局态，-count=2 可重复。

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/evidence"
)

const rollbackSession = "rb-session"

// rollbackWriteBaseline 在工作区回滚目录造一张基线快照，返回其路径。
func rollbackWriteBaseline(t *testing.T, ws, content string) string {
	t.Helper()
	bl := filepath.Join(ws, ".gaea", "work", "rollback", "doc-md-1.before")
	if err := os.MkdirAll(filepath.Dir(bl), 0o755); err != nil {
		t.Fatalf("MkdirAll rollback dir: %v", err)
	}
	if err := os.WriteFile(bl, []byte(content), 0o644); err != nil {
		t.Fatalf("写基线快照: %v", err)
	}
	return bl
}

// rollbackAppendCard 落一张 write_file 证据卡进工作区 journal。
func rollbackAppendCard(t *testing.T, ws, id, baselinePath, afterSummary string) {
	t.Helper()
	st, err := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	if err != nil {
		t.Fatalf("OpenJournal: %v", err)
	}
	if err := st.Append(evidence.ChangeRecord{
		ID:            id,
		SessionID:     rollbackSession,
		Space:         "work",
		Turn:          1,
		Tool:          "write_file",
		Target:        "doc.md",
		BeforeSummary: "before content\n",
		AfterSummary:  afterSummary,
		BaselinePath:  baselinePath,
		Status:        evidence.StatusPendingVerify,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
}

// rollbackLastRecord 返回会话最新一张证据卡（List 旧→新，取末尾）。
func rollbackLastRecord(t *testing.T, ws string) evidence.ChangeRecord {
	t.Helper()
	st, err := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	if err != nil {
		t.Fatalf("OpenJournal: %v", err)
	}
	recs, err := st.List(rollbackSession)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(recs) == 0 {
		t.Fatal("journal 为空")
	}
	return recs[len(recs)-1]
}

// TestGaeaRollbackRecord_SnapshotBeforeRestore 恢复后：新 rollback 卡
// BaselinePath 指向的快照字节 == 恢复前内容；target 字节 == 基线内容；
// Before/After 摘要分别为恢复前/后原文（完整证据卡）。
func TestGaeaRollbackRecord_SnapshotBeforeRestore(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	target := filepath.Join(ws, "doc.md")
	if err := os.WriteFile(target, []byte("current content\n"), 0o644); err != nil {
		t.Fatalf("写目标文件: %v", err)
	}
	bl := rollbackWriteBaseline(t, ws, "baseline content\n")
	rollbackAppendCard(t, ws, "ev_rb_1", bl, "current content\n")

	if err := (&App{}).GaeaRollbackRecord("ev_rb_1"); err != nil {
		t.Fatalf("GaeaRollbackRecord: %v", err)
	}

	// target 已恢复为基线内容
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "baseline content\n" {
		t.Fatalf("target = %q, %v; want baseline content", got, err)
	}
	// 新卡：完整字段
	newRec := rollbackLastRecord(t, ws)
	if newRec.Tool != "rollback" || newRec.Status != evidence.StatusPendingVerify {
		t.Fatalf("新卡 Tool/Status = %q/%q, want rollback/pending_verify", newRec.Tool, newRec.Status)
	}
	if newRec.Target != "doc.md" || newRec.SessionID != rollbackSession {
		t.Errorf("新卡 Target/SessionID = %q/%q, want doc.md/%s", newRec.Target, newRec.SessionID, rollbackSession)
	}
	if newRec.BaselinePath == "" {
		t.Fatal("新卡应携带恢复前快照 BaselinePath")
	}
	if filepath.Dir(newRec.BaselinePath) != filepath.Dir(bl) {
		t.Errorf("快照目录 = %q, want 与原基线同目录 %q", filepath.Dir(newRec.BaselinePath), filepath.Dir(bl))
	}
	snap, err := os.ReadFile(newRec.BaselinePath)
	if err != nil || string(snap) != "current content\n" {
		t.Errorf("快照字节 = %q, %v; want 恢复前内容 current content", snap, err)
	}
	if newRec.BeforeSummary != "current content\n" {
		t.Errorf("BeforeSummary = %q, want 恢复前内容原文", newRec.BeforeSummary)
	}
	if newRec.AfterSummary != "baseline content\n" {
		t.Errorf("AfterSummary = %q, want 基线内容原文（恢复后文件实况）", newRec.AfterSummary)
	}
}

// TestGaeaRollbackRecord_UndoRestore 撤销恢复：对新追加的 rollback 卡再调
// 一次回滚 → target 回到恢复前内容（恢复动作自身可逆）。
func TestGaeaRollbackRecord_UndoRestore(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	target := filepath.Join(ws, "doc.md")
	if err := os.WriteFile(target, []byte("current content\n"), 0o644); err != nil {
		t.Fatalf("写目标文件: %v", err)
	}
	bl := rollbackWriteBaseline(t, ws, "baseline content\n")
	rollbackAppendCard(t, ws, "ev_rb_2", bl, "current content\n")

	// 第一次回滚：current → baseline
	if err := (&App{}).GaeaRollbackRecord("ev_rb_2"); err != nil {
		t.Fatalf("第一次回滚: %v", err)
	}
	undoCard := rollbackLastRecord(t, ws)
	if undoCard.ID == "ev_rb_2" || undoCard.Tool != "rollback" {
		t.Fatalf("未找到新追加的 rollback 卡: %+v", undoCard)
	}
	// 第二次回滚（对 rollback 卡）= 撤销恢复：baseline → current
	if err := (&App{}).GaeaRollbackRecord(undoCard.ID); err != nil {
		t.Fatalf("撤销恢复: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "current content\n" {
		t.Fatalf("撤销后 target = %q, %v; want 回到恢复前内容", got, err)
	}
	// 撤销卡自身同样带基线（快照 = 撤销前内容 = baseline 内容）
	undo2 := rollbackLastRecord(t, ws)
	if undo2.BaselinePath == "" {
		t.Fatal("撤销卡应携带快照")
	}
	snap, err := os.ReadFile(undo2.BaselinePath)
	if err != nil || string(snap) != "baseline content\n" {
		t.Errorf("撤销卡快照 = %q, %v; want baseline content", snap, err)
	}
}

// TestGaeaRollbackRecord_MissingTarget target 不存在 → 恢复成功（target 变为
// 基线内容）且新卡无 BaselinePath（降级不阻断，行为与无快照时一致）。
func TestGaeaRollbackRecord_MissingTarget(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	bl := rollbackWriteBaseline(t, ws, "baseline content\n")
	rollbackAppendCard(t, ws, "ev_rb_3", bl, "")

	if err := (&App{}).GaeaRollbackRecord("ev_rb_3"); err != nil {
		t.Fatalf("target 不存在时恢复不应失败: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(ws, "doc.md"))
	if err != nil || string(got) != "baseline content\n" {
		t.Fatalf("target = %q, %v; want baseline content", got, err)
	}
	newRec := rollbackLastRecord(t, ws)
	if newRec.BaselinePath != "" {
		t.Errorf("target 不存在时新卡 BaselinePath = %q, want 空", newRec.BaselinePath)
	}
	if newRec.BeforeSummary != "" {
		t.Errorf("无快照时 BeforeSummary = %q, want 空", newRec.BeforeSummary)
	}
	if newRec.AfterSummary != "baseline content\n" || newRec.Tool != "rollback" {
		t.Errorf("新卡 AfterSummary/Tool = %q/%q, want baseline content/rollback", newRec.AfterSummary, newRec.Tool)
	}
}
