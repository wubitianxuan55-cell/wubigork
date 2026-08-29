package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/evidence"
)

// v4.1 证据链：写盘工具在成功变更后经 evidence.RecordChange 上报
// Before/After 原文摘要；ctx 无台账时静默（不 panic、不记录）。
func TestWriteToolsReportEvidence(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "hello world")

	ledger := evidence.NewChangeLedger()
	ctx := evidence.WithChanges(context.Background(), ledger)

	if _, err := (editFile{}).Execute(ctx, mustArgs(t, map[string]any{
		"path": path, "old_string": "hello", "new_string": "hi",
	})); err != nil {
		t.Fatalf("edit_file: %v", err)
	}
	recs := ledger.Records()
	if len(recs) != 1 {
		t.Fatalf("edit_file records = %d, want 1", len(recs))
	}
	if recs[0].Tool != "edit_file" || recs[0].Target != path ||
		recs[0].BeforeSummary != "hello" || recs[0].AfterSummary != "hi" {
		t.Fatalf("edit_file evidence mismatch: %+v", recs[0])
	}

	newPath := filepath.Join(dir, "new.txt")
	if _, err := (writeFile{}).Execute(ctx, mustArgs(t, map[string]any{
		"path": newPath, "content": "brand new",
	})); err != nil {
		t.Fatalf("write_file: %v", err)
	}
	recs = ledger.Records()
	if len(recs) != 2 {
		t.Fatalf("after write_file records = %d, want 2", len(recs))
	}
	if recs[1].Tool != "write_file" || recs[1].BeforeSummary != "" || recs[1].AfterSummary != "brand new" {
		t.Fatalf("write_file evidence mismatch: %+v", recs[1])
	}

	dst := filepath.Join(dir, "moved.txt")
	if _, err := (moveFile{}).Execute(ctx, mustArgs(t, map[string]any{
		"source": newPath, "destination": dst,
	})); err != nil {
		t.Fatalf("move_file: %v", err)
	}
	recs = ledger.Records()
	if len(recs) != 3 {
		t.Fatalf("after move_file records = %d, want 3", len(recs))
	}
	if recs[2].Tool != "move_file" || recs[2].Target != dst || !strings.Contains(recs[2].BeforeSummary, "moved from") {
		t.Fatalf("move_file evidence mismatch: %+v", recs[2])
	}
}

func TestWriteToolsWithoutLedgerNoop(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.txt", "hello")
	// 无台账 ctx：不 panic、不记录
	if _, err := (editFile{}).Execute(context.Background(), mustArgs(t, map[string]any{
		"path": path, "old_string": "hello", "new_string": "hi",
	})); err != nil {
		t.Fatalf("edit_file: %v", err)
	}
	if _, err := os.ReadFile(path); err != nil {
		t.Fatalf("file should still exist after edit: %v", err)
	}
}
