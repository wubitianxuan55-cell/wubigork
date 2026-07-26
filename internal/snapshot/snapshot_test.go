package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStore_CaptureAndRestore(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-snapshot-test")
	defer os.RemoveAll(dir)

	store := NewStore(dir)

	// v1
	content1 := "line one\nline two\nline three"
	snap1, err := store.Capture("test-scene", content1, "v1", "manual")
	if err != nil {
		t.Fatalf("Capture v1 failed: %v", err)
	}

	// v2 (修改中间行 + 新增)
	content2 := "line one\nline two modified\nline three\nline four"
	snap2, err := store.Capture("test-scene", content2, "v2", "ai-rewrite")
	if err != nil {
		t.Fatalf("Capture v2 failed: %v", err)
	}

	// 列出
	snaps, err := store.List("test-scene")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}

	// 恢复到 v1
	restored1, err := store.Restore(snap1.ID, "test-scene")
	if err != nil {
		t.Fatalf("Restore v1 failed: %v", err)
	}
	if strings.TrimSpace(restored1) != strings.TrimSpace(content1) {
		t.Fatalf("restore v1 mismatch:\n got: %q\nwant: %q", restored1, content1)
	}

	// 恢复到 v2
	restored2, err := store.Restore(snap2.ID, "test-scene")
	if err != nil {
		t.Fatalf("Restore v2 failed: %v", err)
	}
	if strings.TrimSpace(restored2) != strings.TrimSpace(content2) {
		t.Fatalf("restore v2 mismatch:\n got: %q\nwant: %q", restored2, content2)
	}
}

func TestStore_Diff(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-snapshot-diff")
	defer os.RemoveAll(dir)

	store := NewStore(dir)

	content1 := "line one\nline two\nline three"
	snap1, _ := store.Capture("diff-scene", content1, "v1", "manual")

	content2 := "line one\nline two modified\nline four"
	snap2, _ := store.Capture("diff-scene", content2, "v2", "manual")

	diffs, err := store.Diff("diff-scene", snap1.ID, snap2.ID)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if len(diffs) == 0 {
		t.Fatal("expected non-empty diff")
	}

	// 验证 diff 包含预期的变更
	hasAdd, hasDel := false, false
	for _, d := range diffs {
		if d.Type == "add" {
			hasAdd = true
		}
		if d.Type == "del" {
			hasDel = true
		}
	}
	if !hasAdd || !hasDel {
		t.Fatalf("diff should contain both add and del lines, got %d lines", len(diffs))
	}
}

func TestStore_EmptyScene(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-snapshot-empty")
	defer os.RemoveAll(dir)

	store := NewStore(dir)
	snaps, err := store.List("nonexistent")
	if err != nil {
		t.Fatalf("List on nonexistent scene failed: %v", err)
	}
	if len(snaps) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(snaps))
	}
}

func TestStore_ChineseContent(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-snapshot-chinese")
	defer os.RemoveAll(dir)

	store := NewStore(dir)

	content := "青云宗坐落于云雾缭绕的苍山之上。\n宗门大殿由千年寒玉砌成，\n门下弟子三千，皆为剑修。"
	snap, err := store.Capture("zh-scene", content, "中文快照", "manual")
	if err != nil {
		t.Fatalf("Capture Chinese content failed: %v", err)
	}

	restored, err := store.Restore(snap.ID, "zh-scene")
	if err != nil {
		t.Fatalf("Restore Chinese content failed: %v", err)
	}
	if strings.TrimSpace(restored) != strings.TrimSpace(content) {
		t.Fatalf("Chinese content mismatch:\n got: %q\nwant: %q", restored, content)
	}
}
