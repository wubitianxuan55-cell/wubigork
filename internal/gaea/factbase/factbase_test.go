package factbase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAddUpsertAndClear(t *testing.T) {
	b := &Base{}
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	b.Add("工期", "90 日历天", "招标文件 P3", "项目", now)
	b.Add("修复目标", "砷 ≤ 60 mg/kg", "招标文件 P5", "数据", now)
	b.Add("工期", "120 日历天", "答疑纪要", "项目", now.Add(time.Hour)) // upsert
	if len(b.Facts) != 2 {
		t.Fatalf("want 2 facts after upsert, got %d", len(b.Facts))
	}
	for _, f := range b.Facts {
		if f.Key == "工期" && f.Value != "120 日历天" {
			t.Fatalf("upsert failed: value=%q source=%q", f.Value, f.Source)
		}
	}
	b.Add("工期", "", "", "", now) // empty value removes
	if len(b.Facts) != 1 {
		t.Fatalf("empty value should remove the fact, got %d facts", len(b.Facts))
	}
	b.Clear()
	if len(b.Facts) != 0 {
		t.Fatalf("clear should empty the base")
	}
}

func TestMarkdownEscapesPipes(t *testing.T) {
	b := &Base{}
	b.Add("修复目标", "砷 ≤ 60 mg/kg（GB 36600-2018）", "标准 GB 36600-2018 | 表1", "数据", time.Now())
	md := b.Markdown()
	if !strings.Contains(md, "GB 36600-2018 \\| 表1") {
		t.Fatalf("pipe in source not escaped:\n%s", md)
	}
	if !strings.Contains(md, "## 事实底座") {
		t.Fatalf("markdown missing header:\n%s", md)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "20260809-123.jsonl")
	fp := PathFor(path)
	if !strings.HasSuffix(fp, "20260809-123-facts.json") {
		t.Fatalf("unexpected fact path: %s", fp)
	}
	s := NewStore(fp)
	if err := s.Add("工期", "90 日历天", "招标文件", "项目"); err != nil {
		t.Fatalf("add: %v", err)
	}
	got, err := s.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(got.Facts) != 1 || got.Facts[0].Value != "90 日历天" {
		t.Fatalf("round-trip mismatch: %+v", got.Facts)
	}
	md, err := s.Markdown()
	if err != nil || !strings.Contains(md, "90 日历天") {
		t.Fatalf("markdown after reload missing value: %q err=%v", md, err)
	}
	if err := s.Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, _ = s.Snapshot()
	if len(got.Facts) != 0 {
		t.Fatalf("clear did not persist")
	}
}

func TestStoreEmptyPath(t *testing.T) {
	s := NewStore("")
	if err := s.Add("k", "v", "", ""); err == nil {
		t.Fatalf("expected error for empty path")
	}
	b, err := s.Snapshot()
	if err != nil || len(b.Facts) != 0 {
		t.Fatalf("empty store should snapshot empty: %v %+v", err, b.Facts)
	}
}

func TestPathForEdge(t *testing.T) {
	if got := PathFor(""); got != "" {
		t.Fatalf("empty path should stay empty, got %q", got)
	}
	if got := PathFor(`C:\x\sessions\a.jsonl`); got != `C:\x\sessions\a-facts.json` {
		t.Fatalf("windows path mismatch: %s", got)
	}
	if _, err := os.Stat(PathFor("")); err == nil {
		t.Fatalf("empty path stat should fail")
	}
}
