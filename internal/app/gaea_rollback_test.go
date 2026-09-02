package app

// v4.32 线A 收 v4.28 B1 欠账：GaeaRollbackRecord 恢复写盘前先把目标当前
// 内容快照（恢复动作自身成为带基线的完整证据卡，时间线里可「再恢复」=
// 撤销恢复）；target 不存在/快照失败降级不阻断。
// v4.33 线A：write_file 守卫改同口径截断比较（>8KB 未手改不再误报）；
// rollback 卡接入「恢复后已被手工修改」守卫（撤销恢复前先校验）。
// journal 搭建沿用 channel B 测试模式：t.TempDir + isolateWorkspaceTo 配置
// 隔离，无进程级全局态，-count=2 可重复。

import (
	"os"
	"path/filepath"
	"strings"
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

// rollbackAppendCardTool 落一张指定 Tool 的证据卡进工作区 journal。
func rollbackAppendCardTool(t *testing.T, ws, id, tool, baselinePath, afterSummary string) {
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
		Tool:          tool,
		Target:        "doc.md",
		BeforeSummary: "before content\n",
		AfterSummary:  afterSummary,
		BaselinePath:  baselinePath,
		Status:        evidence.StatusPendingVerify,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
}

// rollbackAppendCard 落一张 write_file 证据卡进工作区 journal。
func rollbackAppendCard(t *testing.T, ws, id, baselinePath, afterSummary string) {
	rollbackAppendCardTool(t, ws, id, "write_file", baselinePath, afterSummary)
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

// bigRollbackContent 造一份 >SummaryLimit 的文件内容（write_file 卡落库摘要
// 会被 RecordChange/ClampSummary 截到 SummaryLimit——>8KB 卡守卫回归的原料）。
func bigRollbackContent() string {
	return strings.Repeat("a", evidence.SummaryLimit+128) + "\n"
}

// TestGaeaRollbackRecord_WriteFileLargeNoFalseReject v4.33 线A 钉死误报回归：
// >8KB write_file 卡（AfterSummary 落库时已截断）+ 文件从未手改 → 回滚成功。
// 修正前精确比较 curS != AfterSummary 对该场景恒误拒。
func TestGaeaRollbackRecord_WriteFileLargeNoFalseReject(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	target := filepath.Join(ws, "doc.md")
	big := bigRollbackContent()
	if err := os.WriteFile(target, []byte(big), 0o644); err != nil {
		t.Fatalf("写目标文件: %v", err)
	}
	bl := rollbackWriteBaseline(t, ws, "baseline content\n")
	// 卡内摘要按 RecordChange 落库口径截断（>8KB 卡的实况）
	rollbackAppendCard(t, ws, "ev_rb_big1", bl, evidence.ClampSummary(big))

	if err := (&App{}).GaeaRollbackRecord("ev_rb_big1"); err != nil {
		t.Fatalf(">8KB 未手改文件回滚不应被误拒: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "baseline content\n" {
		t.Fatalf("target = %q, %v; want baseline content", got, err)
	}
}

// TestGaeaRollbackRecord_WriteFileHandEditReject >8KB write_file 卡 + 手改点
// 落在摘要窗口内（窗口尾部，仍 <SummaryLimit）→ 拒绝且目标不被覆盖。
// （同口径截断的固有边界：手改点整体落在 SummaryLimit 之外的尾部检测不到，
// 宁漏勿误——与守卫设计一致。）
func TestGaeaRollbackRecord_WriteFileHandEditReject(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	target := filepath.Join(ws, "doc.md")
	big := bigRollbackContent()
	// 手改摘要窗口尾部一个字节：改后 ClampSummary(cur) != 存储摘要
	edited := big[:evidence.SummaryLimit-1] + "X" + big[evidence.SummaryLimit:]
	if err := os.WriteFile(target, []byte(edited), 0o644); err != nil {
		t.Fatalf("写目标文件: %v", err)
	}
	bl := rollbackWriteBaseline(t, ws, "baseline content\n")
	rollbackAppendCard(t, ws, "ev_rb_big2", bl, evidence.ClampSummary(big))

	err := (&App{}).GaeaRollbackRecord("ev_rb_big2")
	if err == nil {
		t.Fatal("手改后的文件回滚应被拒绝")
	}
	if !strings.Contains(err.Error(), "已被手工修改") {
		t.Fatalf("拒绝文案应说明手工修改冲突, got %q", err.Error())
	}
	got, rerr := os.ReadFile(target)
	if rerr != nil || string(got) != edited {
		t.Fatalf("拒绝后 target 不应被覆盖, got %q, %v", got, rerr)
	}
}

// TestGaeaRollbackRecord_RollbackCardGuard v4.33 线A 接入 rollback 卡守卫：
// 恢复后手工改过文件 → 对 rollback 卡再回滚（撤销恢复）被拒绝且不覆盖；
// 未手改路径（可再回滚）由 TestGaeaRollbackRecord_UndoRestore 覆盖。
func TestGaeaRollbackRecord_RollbackCardGuard(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	target := filepath.Join(ws, "doc.md")
	if err := os.WriteFile(target, []byte("current content\n"), 0o644); err != nil {
		t.Fatalf("写目标文件: %v", err)
	}
	bl := rollbackWriteBaseline(t, ws, "baseline content\n")
	rollbackAppendCard(t, ws, "ev_rb_g1", bl, "current content\n")

	// 第一次回滚：current → baseline，产生 rollback 卡（AfterSummary=基线内容）
	if err := (&App{}).GaeaRollbackRecord("ev_rb_g1"); err != nil {
		t.Fatalf("第一次回滚: %v", err)
	}
	rc := rollbackLastRecord(t, ws)
	if rc.ID == "ev_rb_g1" || rc.Tool != "rollback" {
		t.Fatalf("未找到新追加的 rollback 卡: %+v", rc)
	}
	// 恢复后手工编辑目标
	if err := os.WriteFile(target, []byte("baseline content edited\n"), 0o644); err != nil {
		t.Fatalf("手工编辑目标: %v", err)
	}
	// 对 rollback 卡再回滚 → 守卫拒绝
	err := (&App{}).GaeaRollbackRecord(rc.ID)
	if err == nil {
		t.Fatal("恢复后手工改过的文件再回滚应被拒绝")
	}
	if !strings.Contains(err.Error(), "已被手工修改") {
		t.Fatalf("拒绝文案应说明手工修改冲突, got %q", err.Error())
	}
	got, rerr := os.ReadFile(target)
	if rerr != nil || string(got) != "baseline content edited\n" {
		t.Fatalf("拒绝后 target 不应被覆盖, got %q, %v", got, rerr)
	}
}

// TestGaeaRollbackRecord_RollbackCardLegacyNoSummary rollback 旧卡无
// AfterSummary → 跳过守卫（宁漏勿误，对齐 write_file/edit_like 旧行为语义）。
func TestGaeaRollbackRecord_RollbackCardLegacyNoSummary(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	target := filepath.Join(ws, "doc.md")
	if err := os.WriteFile(target, []byte("handwritten state\n"), 0o644); err != nil {
		t.Fatalf("写目标文件: %v", err)
	}
	bl := rollbackWriteBaseline(t, ws, "baseline content\n")
	rollbackAppendCardTool(t, ws, "ev_rb_old", "rollback", bl, "")

	if err := (&App{}).GaeaRollbackRecord("ev_rb_old"); err != nil {
		t.Fatalf("无 AfterSummary 的旧 rollback 卡应跳过守卫照常回滚: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "baseline content\n" {
		t.Fatalf("target = %q, %v; want baseline content", got, err)
	}
}
