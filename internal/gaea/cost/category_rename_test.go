package cost

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// TestSaveCategoryRenameChineseSubtree（2026-09-19 审计 P0 回归）：分类改名重写
// 子树条目路径时，偏移必须按 UTF-8 字符计数——原实现用 Go len()（字节数）喂
// SQLite substr（按字符计数），中文路径下子树后缀整段截断、条目被重挂到改名
// 节点本身。同刀覆盖：换父也重写路径、直接子条目叶子名同步、环检测、名称含
// "/" 拒绝。
func TestSaveCategoryRenameChineseSubtree(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	// 中文两级树：人工（一级）→ 人工/普工（二级，其下再挂三级路径形态）。
	l1, err := s.SaveCategory(0, "人工", 0, 0)
	if err != nil {
		t.Fatalf("SaveCategory 人工: %v", err)
	}
	l2, err := s.SaveCategory(l1, "普工", 0, 0)
	if err != nil {
		t.Fatalf("SaveCategory 普工: %v", err)
	}
	// 直接子条目 + 深层子树条目（纯中文路径，字节长度≠字符数）。
	if err := s.Save(Entry{Name: "e1", Category: "普工", CategoryPath: "人工/普工", Price: 100, Status: "现行"}); err != nil {
		t.Fatalf("Save e1: %v", err)
	}
	if err := s.Save(Entry{Name: "e2", Category: "混凝土", CategoryPath: "人工/普工/混凝土浇筑", Price: 200, Status: "现行"}); err != nil {
		t.Fatalf("Save e2: %v", err)
	}

	// 改名「人工」→「人工费」：子树路径前缀应整树重写，后缀原样保留。
	if _, err := s.SaveCategory(0, "人工费", 0, l1); err != nil {
		t.Fatalf("改名: %v", err)
	}
	g1, err := s.Get("e1")
	if err != nil {
		t.Fatalf("Get e1: %v", err)
	}
	if g1.CategoryPath != "人工费/普工" || g1.Category != "普工" {
		t.Fatalf("直接子条目 = path %q cat %q, want 人工费/普工 + 叶子名同步", g1.CategoryPath, g1.Category)
	}
	g2, err := s.Get("e2")
	if err != nil {
		t.Fatalf("Get e2: %v", err)
	}
	if g2.CategoryPath != "人工费/普工/混凝土浇筑" {
		t.Fatalf("深层条目路径 = %q, want 人工费/普工/混凝土浇筑（字节偏移会把后缀截成乱码）", g2.CategoryPath)
	}

	// 换父不移名：「普工」从 人工费 移到根——原实现门控在改名上，路径不重写。
	if _, err := s.SaveCategory(0, "普工", 0, l2); err != nil {
		t.Fatalf("换父: %v", err)
	}
	g1, _ = s.Get("e1")
	if g1.CategoryPath != "普工" {
		t.Fatalf("换父后直接子条目 = %q, want 普工", g1.CategoryPath)
	}
	g2, _ = s.Get("e2")
	if g2.CategoryPath != "普工/混凝土浇筑" {
		t.Fatalf("换父后深层条目 = %q, want 普工/混凝土浇筑", g2.CategoryPath)
	}
}

// TestSaveCategoryGuards：环检测（挂自己/挂自己的子孙拒绝）与名称含 / 拒绝。
func TestSaveCategoryGuards(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	a, _ := s.SaveCategory(0, "甲", 0, 0)
	b, _ := s.SaveCategory(a, "乙", 0, 0)

	if _, err := s.SaveCategory(a, "甲改", 0, a); err == nil || !strings.Contains(err.Error(), "自己名下") {
		t.Fatalf("挂到自己应拒绝, got %v", err)
	}
	if _, err := s.SaveCategory(b, "甲改", 0, a); err == nil || !strings.Contains(err.Error(), "环") {
		t.Fatalf("挂到自己的子孙应拒绝, got %v", err)
	}
	if _, err := s.SaveCategory(0, "带/名字", 0, 0); err == nil {
		t.Fatal("名称含 / 应拒绝")
	}
	// 合法更新不受影响。
	if _, err := s.SaveCategory(a, "乙改名", 0, b); err != nil {
		t.Fatalf("合法改名: %v", err)
	}
}
