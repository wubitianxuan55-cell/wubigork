package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func TestChapterBranchSummaryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0755); err != nil {
		t.Fatal(err)
	}
	pm := &Manager{Dir: dir, Meta: &types.ProjectMeta{Title: "测试"}}
	summary := &types.ChapterSummary{Title: "第3a章", Summary: "分支剧情摘要"}

	if err := pm.WriteChapterBranchSummary(3, "a", summary); err != nil {
		t.Fatal(err)
	}
	got, err := pm.ReadChapterBranchSummary(3, "a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "第3a章" || got.Summary != "分支剧情摘要" {
		t.Fatalf("got %+v", got)
	}
	if want := filepath.Join(dir, "chapters", "003a-summary.json"); pm.ChapterBranchSummaryPath(3, "a") != want {
		t.Fatalf("path = %s, want %s", pm.ChapterBranchSummaryPath(3, "a"), want)
	}
}
