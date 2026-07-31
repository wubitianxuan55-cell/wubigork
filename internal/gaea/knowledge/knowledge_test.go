package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEntryRoundtrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	e := Entry{
		Name:      "test-entry-001",
		Title:     "测试知识条目",
		Category:  CatExperience,
		Tags:      []string{"修复", "化学氧化"},
		Status:    "草稿",
		Version:   1,
		Author:    "测试人",
		CreatedAt: now,
		UpdatedAt: now,
		Source:    "实践总结",
		Body:      "这是一条测试知识条目。\n\n包含多个段落。",
	}

	if err := s.Save(e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(e.Name)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Title != e.Title {
		t.Errorf("Title = %q, want %q", got.Title, e.Title)
	}
	if got.Category != e.Category {
		t.Errorf("Category = %q, want %q", got.Category, e.Category)
	}
	if len(got.Tags) != len(e.Tags) {
		t.Errorf("Tags = %v, want %v", got.Tags, e.Tags)
	}
	if got.Version != e.Version {
		t.Errorf("Version = %d, want %d", got.Version, e.Version)
	}
	if !strings.Contains(got.Body, "测试知识条目") {
		t.Errorf("Body missing expected text: %s", got.Body)
	}
}

func TestSaveUpdatesIndex(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	e := Entry{
		Name:     "index-test",
		Title:    "索引测试",
		Category: CatStandard,
		Body:     "body text",
	}
	if err := s.Save(e); err != nil {
		t.Fatal(err)
	}

	idx := s.Index()
	if !strings.Contains(idx, "index-test") {
		t.Error("INDEX.md should contain entry name")
	}
	if !strings.Contains(idx, "索引测试") {
		t.Error("INDEX.md should contain entry title")
	}
}

func TestDeleteRemovesFromIndex(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	e := Entry{Name: "delete-me", Title: "待删除", Category: CatOther, Body: "x"}
	if err := s.Save(e); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("delete-me"); err != nil {
		t.Fatal(err)
	}

	// File should be gone.
	path := filepath.Join(dir, "delete-me.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}

	// Entry should not appear in List.
	for _, entry := range s.List() {
		if entry.Name == "delete-me" {
			t.Error("deleted entry should not appear in List")
		}
	}

	// Index should not contain the entry.
	idx := s.Index()
	if strings.Contains(idx, "delete-me") {
		t.Error("INDEX.md should not contain deleted entry")
	}
}

func TestSearchByCategory(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	s.Save(Entry{Name: "case-1", Title: "案例一", Category: CatCase, Body: "某化工厂修复案例"})
	s.Save(Entry{Name: "std-1", Title: "标准一", Category: CatStandard, Body: "HJ 25.1 技术要求"})
	s.Save(Entry{Name: "exp-1", Title: "经验一", Category: CatExperience, Body: "施工经验总结"})

	results := Search(s, "", Filter{Category: CatCase})
	if len(results) != 1 {
		t.Fatalf("expected 1 case entry, got %d", len(results))
	}
	if results[0].Name != "case-1" {
		t.Errorf("got %s, want case-1", results[0].Name)
	}
}

func TestSearchKeywordTitlePriority(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	s.Save(Entry{Name: "title-match", Title: "化学氧化技术总结", Category: CatExperience, Body: "正文中不提化学氧化"})
	s.Save(Entry{Name: "body-match", Title: "其他技术", Category: CatExperience, Body: "这里提到化学氧化方法"})

	results := Search(s, "化学氧化", Filter{})
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	// Title match should score higher.
	if results[0].Name != "title-match" {
		t.Errorf("expected title-match first, got %s", results[0].Name)
	}
}

func TestSearchTagFilter(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	s.Save(Entry{Name: "a", Title: "A", Category: CatOther, Body: "x", Tags: []string{"修复"}})
	s.Save(Entry{Name: "b", Title: "B", Category: CatOther, Body: "x", Tags: []string{"调查"}})

	results := Search(s, "", Filter{Tag: "修复"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result with tag 修复, got %d", len(results))
	}
}

func TestListReturnsAll(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	s.Save(Entry{Name: "e1", Title: "E1", Category: CatOther, Body: "x"})
	s.Save(Entry{Name: "e2", Title: "E2", Category: CatOther, Body: "x"})

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}
}

func TestOpenCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "knowledge")
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open should create directories: %v", err)
	}
	if s.Dir != dir {
		t.Errorf("Dir = %q, want %q", s.Dir, dir)
	}
}

func TestEntryWithAllFields(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	e := Entry{
		Name:       "full-entry",
		Title:      "完整条目",
		Category:   CatRegulation,
		Phase:      "调查",
		Discipline: "环境工程",
		Tags:       []string{"法规", "土壤", "地下水"},
		Status:     "已审核",
		Version:    3,
		Author:     "张三",
		Reviewer:   "李四",
		CreatedAt:  now,
		UpdatedAt:  now,
		Source:     "生态环境部",
		Body:       "## 正文\n\n法规内容摘要。",
	}

	if err := s.Save(e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(e.Name)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Phase != "调查" {
		t.Errorf("Phase = %q", got.Phase)
	}
	if got.Discipline != "环境工程" {
		t.Errorf("Discipline = %q", got.Discipline)
	}
	if got.Reviewer != "李四" {
		t.Errorf("Reviewer = %q", got.Reviewer)
	}
	if got.Version != 3 {
		t.Errorf("Version = %d", got.Version)
	}
	if len(got.Tags) != 3 {
		t.Errorf("Tags = %v", got.Tags)
	}
}
