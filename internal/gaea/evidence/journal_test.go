package evidence

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestChangeLedgerLifecycle(t *testing.T) {
	l := NewChangeLedger()
	l.Add(ChangeRecord{Tool: "edit_file", Target: "a.txt", BeforeSummary: "x", AfterSummary: "y"})
	l.Add(ChangeRecord{Tool: "write_file", Target: "b.txt", BeforeSummary: "", AfterSummary: "z"})
	if got := len(l.Records()); got != 2 {
		t.Fatalf("Records len = %d, want 2", got)
	}
	l.Reset()
	if got := len(l.Records()); got != 0 {
		t.Fatalf("after Reset len = %d, want 0", got)
	}
	// nil 台账安全
	var nilLedger *ChangeLedger
	nilLedger.Reset()
	nilLedger.Add(ChangeRecord{})
	if got := nilLedger.Records(); got != nil {
		t.Fatalf("nil ledger Records = %v, want nil", got)
	}
}

func TestRecordChangeCtx(t *testing.T) {
	// 无台账 ctx：静默跳过
	RecordChange(context.Background(), ChangeRecord{Tool: "edit_file"})

	// 有台账 ctx：记录并截断摘要
	l := NewChangeLedger()
	ctx := WithChanges(context.Background(), l)
	big := strings.Repeat("a", SummaryLimit+100)
	RecordChange(ctx, ChangeRecord{Tool: "edit_file", BeforeSummary: big, AfterSummary: "ok"})
	recs := l.Records()
	if len(recs) != 1 {
		t.Fatalf("records len = %d, want 1", len(recs))
	}
	if len(recs[0].BeforeSummary) != SummaryLimit {
		t.Fatalf("BeforeSummary truncated to %d, want %d", len(recs[0].BeforeSummary), SummaryLimit)
	}
	if recs[0].Status != StatusPendingVerify {
		t.Fatalf("Status = %q, want %q", recs[0].Status, StatusPendingVerify)
	}
}

// TestClampSummary v4.33 线A 单点截断 helper：短串原样；超限按字节截到
// SummaryLimit（不做 rune 安全截断，钉死与 RecordChange 落库一致的历史行为）；
// 且与 RecordChange 落库摘要逐字节同口径。
func TestClampSummary(t *testing.T) {
	short := "hello\n"
	if got := ClampSummary(short); got != short {
		t.Errorf("ClampSummary(短串) = %q, want 原样", got)
	}
	if got := ClampSummary(""); got != "" {
		t.Errorf("ClampSummary(\"\") = %q, want 空", got)
	}
	// 超限：恰好截到 SummaryLimit，内容 == s[:SummaryLimit]
	big := strings.Repeat("a", SummaryLimit+100) + "tail"
	got := ClampSummary(big)
	if len(got) != SummaryLimit || got != big[:SummaryLimit] {
		t.Fatalf("ClampSummary(>8KB) len=%d, want %d 且等于 s[:SummaryLimit]", len(got), SummaryLimit)
	}
	// 历史行为：边界切进 UTF-8 中间字节（末字节落在多字节字符中间），
	// 保持按字节截断，不引入 rune 安全新语义。
	mixed := strings.Repeat("a", SummaryLimit-2) + "中" // "中" 3 字节，第 SummaryLimit 字节切进其首字节后
	clamped := ClampSummary(mixed)
	if len(clamped) != SummaryLimit || clamped != mixed[:SummaryLimit] {
		t.Fatalf("ClampSummary(mixed) 应为 mixed[:SummaryLimit]（按字节），len=%d", len(clamped))
	}
	if utf8.ValidString(clamped) {
		t.Errorf("切进 UTF-8 中间字节的结果应保持非法 UTF-8（历史字节口径），got 合法串")
	}
	// 与 RecordChange 落库口径逐字节一致
	l := NewChangeLedger()
	RecordChange(WithChanges(context.Background(), l), ChangeRecord{Tool: "write_file", AfterSummary: big, BeforeSummary: mixed})
	recs := l.Records()
	if len(recs) != 1 {
		t.Fatalf("records len = %d, want 1", len(recs))
	}
	if recs[0].AfterSummary != got {
		t.Errorf("RecordChange 落库 AfterSummary 与 ClampSummary(big) 不一致")
	}
	if recs[0].BeforeSummary != clamped {
		t.Errorf("RecordChange 落库 BeforeSummary 与 ClampSummary(mixed) 不一致")
	}
}

func TestJournalStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := OpenJournal(dir)
	if err != nil {
		t.Fatalf("OpenJournal: %v", err)
	}
	recs := []ChangeRecord{
		{ID: "id-1", SessionID: "sess/1", Space: "work", Turn: 1, Tool: "edit_file", Target: "a.txt", BeforeSummary: "x", AfterSummary: "y", At: 1000, Status: StatusPendingVerify},
		{ID: "id-2", SessionID: "sess/1", Space: "work", Turn: 1, Tool: "write_file", Target: "b.txt", BeforeSummary: "", AfterSummary: "z", At: 1001, Status: StatusPendingVerify},
	}
	for _, r := range recs {
		if err := st.Append(r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	got, err := st.List("sess/1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List len = %d, want 2", len(got))
	}
	if got[0].Target != "a.txt" || got[1].Target != "b.txt" {
		t.Fatalf("round-trip order/content mismatch: %+v", got)
	}
	// 会话标识清洗为安全文件名
	if _, err := os.Stat(filepath.Join(dir, "sess_1.jsonl")); err != nil {
		t.Fatalf("sanitized journal file missing: %v", err)
	}
	// 不存在会话 → 空列表不报错
	if got, err := st.List("nope"); err != nil || len(got) != 0 {
		t.Fatalf("List(nope) = %v, %v; want empty nil", got, err)
	}
}

// TestStageBaselineTo v4.32 薄导出 helper：命名/权限与 StageBaseline 约定
// 一致（<sessionKey(target) 截 120>-<unixnano>.before，0644），内容逐字节
// 落盘，空 dir 静默降级。
func TestStageBaselineTo(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rollback")
	target := filepath.Join(t.TempDir(), "sub", "doc.md")
	path := StageBaselineTo(dir, target, []byte("content-1"))
	if path == "" {
		t.Fatal("StageBaselineTo 返回空路径")
	}
	if filepath.Dir(path) != dir {
		t.Errorf("快照目录 = %q, want %q", filepath.Dir(path), dir)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "content-1" {
		t.Fatalf("快照内容 = %q, %v; want content-1", raw, err)
	}
	// 命名约定：<sessionKey(target) 截 120>-<unixnano>.before
	wantPrefix := sessionKeyOf(target)
	if len(wantPrefix) > 120 {
		wantPrefix = wantPrefix[:120]
	}
	base := filepath.Base(path)
	if !strings.HasPrefix(base, wantPrefix+"-") || !strings.HasSuffix(base, ".before") {
		t.Errorf("文件名 %q 不符合 %s-<nano>.before 约定", base, wantPrefix)
	}
	// 同 target 两次快照 → 两个不同文件，内容各自正确
	path2 := StageBaselineTo(dir, target, []byte("content-2"))
	if path2 == "" || path2 == path {
		t.Fatalf("第二次快照 path = %q, want 新路径", path2)
	}
	raw2, err := os.ReadFile(path2)
	if err != nil || string(raw2) != "content-2" {
		t.Errorf("第二次快照内容 = %q, %v; want content-2", raw2, err)
	}
	// 空 dir：静默降级返回 ""
	if got := StageBaselineTo("", target, []byte("x")); got != "" {
		t.Errorf("空 dir 应返回空, got %q", got)
	}
}

// TestStageBaselineCtx 无台账 ctx 保持原有静默降级；有台账 ctx 时落
// BaselineDir（重构后仍走 StageBaselineTo 同一套命名）。
func TestStageBaselineCtx(t *testing.T) {
	if got := StageBaseline(context.Background(), "a.txt", []byte("x")); got != "" {
		t.Errorf("无台账 ctx 应返回空, got %q", got)
	}
	// 有台账但未配置基线目录 → ""
	l := NewChangeLedger()
	if got := StageBaseline(WithChanges(context.Background(), l), "a.txt", []byte("x")); got != "" {
		t.Errorf("未配置 BaselineDir 应返回空, got %q", got)
	}
	dir := filepath.Join(t.TempDir(), "rollback")
	l.SetBaselineDir(dir)
	path := StageBaseline(WithChanges(context.Background(), l), "a.txt", []byte("via ctx"))
	if path == "" || filepath.Dir(path) != dir {
		t.Fatalf("StageBaseline path = %q, want 在 %q 下", path, dir)
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != "via ctx" {
		t.Errorf("快照内容 = %q, %v; want via ctx", raw, err)
	}
}

func TestWriteTurnMarkdown(t *testing.T) {
	dir := t.TempDir()
	st, err := OpenJournal(filepath.Join(dir, "journal"))
	if err != nil {
		t.Fatalf("OpenJournal: %v", err)
	}
	recs := []ChangeRecord{
		{ID: "id-1", SessionID: "s1", Turn: 3, Tool: "edit_file", Target: "a.txt", BeforeSummary: "old", AfterSummary: "new", At: 1700000000000, Status: StatusPendingVerify},
	}
	exports := filepath.Join(dir, "exports")
	path, err := st.WriteTurnMarkdown(filepath.Join(exports, "journal"), "s1", 3, recs)
	if err != nil {
		t.Fatalf("WriteTurnMarkdown: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read markdown: %v", err)
	}
	s := string(raw)
	for _, want := range []string{"# 回合证据 Journal", "回合：3", "edit_file → a.txt", "变更前（原文摘要）", "old", "new"} {
		if !strings.Contains(s, want) {
			t.Errorf("markdown missing %q", want)
		}
	}
	// 空记录 → 不写文件
	if _, err := st.WriteTurnMarkdown(filepath.Join(exports, "journal"), "s1", 4, nil); err != nil {
		t.Fatalf("empty WriteTurnMarkdown should not error: %v", err)
	}
}
