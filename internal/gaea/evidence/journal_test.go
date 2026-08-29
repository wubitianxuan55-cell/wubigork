package evidence

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
