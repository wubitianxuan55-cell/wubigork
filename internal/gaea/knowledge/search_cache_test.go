package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSearchCacheReusedAcrossQueries 断言：同一过滤条件下重复 Search 只构建
// 一次 TF-IDF 索引，且结果与首次完全一致（用构建计数器断言，不做计时断言）。
func TestSearchCacheReusedAcrossQueries(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Entry{Name: "a", Title: "化学氧化技术总结", Category: CatExperience, Body: "污染场地修复案例，涉及化学氧化药剂投加。"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Entry{Name: "b", Title: "语音陪伴", Category: CatOther, Body: "语音识别与合成实现对话交互。"}); err != nil {
		t.Fatal(err)
	}

	first := Search(s, "化学氧化", Filter{})
	if len(first) == 0 {
		t.Fatal("expected results")
	}
	if got := s.tfidf.buildsCount(); got != 1 {
		t.Fatalf("builds after first search = %d, want 1", got)
	}

	// 重复查询（含空查询与带过滤条件查询）都不得触发重建：
	// 空查询根本不建索引；带过滤条件走不同签名，只建一次。
	for i := 0; i < 3; i++ {
		again := Search(s, "化学氧化", Filter{})
		if len(again) != len(first) {
			t.Fatalf("repeat search result count = %d, want %d", len(again), len(first))
		}
		for j := range again {
			if again[j].Name != first[j].Name {
				t.Fatalf("repeat search result[%d] = %s, want %s", j, again[j].Name, first[j].Name)
			}
		}
	}
	_ = Search(s, "", Filter{}) // 空查询不触达索引
	if got := s.tfidf.buildsCount(); got != 1 {
		t.Fatalf("builds after repeated searches = %d, want 1", got)
	}

	results := Search(s, "化学氧化", Filter{Category: CatExperience})
	if got := s.tfidf.buildsCount(); got != 2 {
		t.Fatalf("builds after first filtered search = %d, want 2 (new filter signature)", got)
	}
	if len(results) == 0 || results[0].Name != "a" {
		t.Errorf("filtered search = %+v, want entry a first", results)
	}
	_ = Search(s, "化学氧化", Filter{Category: CatExperience})
	if got := s.tfidf.buildsCount(); got != 2 {
		t.Fatalf("builds after repeated filtered search = %d, want 2", got)
	}
}

// TestSearchCacheInvalidatedByWrite 断言：写路径（Save/Delete）使缓存失效，
// 下次 Search 重建索引且结果反映最新数据。
func TestSearchCacheInvalidatedByWrite(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Entry{Name: "a", Title: "化学氧化技术总结", Category: CatExperience, Body: "污染场地修复案例。"}); err != nil {
		t.Fatal(err)
	}

	if len(Search(s, "化学氧化", Filter{})) == 0 {
		t.Fatal("expected results before write")
	}
	if got := s.tfidf.buildsCount(); got != 1 {
		t.Fatalf("builds after first search = %d, want 1", got)
	}

	// 新增条目 → 失效 → 重建，且新条目可被搜到。
	if err := s.Save(Entry{Name: "b", Title: "原位化学氧化案例", Category: CatCase, Body: "某场地原位化学氧化修复工程。"}); err != nil {
		t.Fatal(err)
	}
	results := Search(s, "化学氧化", Filter{})
	if got := s.tfidf.buildsCount(); got != 2 {
		t.Fatalf("builds after save+search = %d, want 2", got)
	}
	foundNew := false
	for _, r := range results {
		if r.Name == "b" {
			foundNew = true
		}
	}
	if !foundNew {
		t.Errorf("new entry b not found after rebuild: %+v", results)
	}

	// 删除条目 → 失效 → 重建，被删条目不再出现。
	if err := s.Delete("b"); err != nil {
		t.Fatal(err)
	}
	results = Search(s, "化学氧化", Filter{})
	if got := s.tfidf.buildsCount(); got != 3 {
		t.Fatalf("builds after delete+search = %d, want 3", got)
	}
	for _, r := range results {
		if r.Name == "b" {
			t.Errorf("deleted entry b still in results: %+v", results)
		}
	}
}

// TestSearchCacheSeesOutOfBandEdit 断言：文件后端绕过 Store 直改磁盘时，
// 下一次 Search 仍能看到改动 —— 旧实现（每次重建）的行为必须保持，
// 候选内容指纹兜底保证缓存不掩盖带外编辑。
func TestSearchCacheSeesOutOfBandEdit(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Entry{Name: "a", Title: "旧标题", Category: CatOther, Body: "初始内容。"}); err != nil {
		t.Fatal(err)
	}

	if results := Search(s, "旧标题", Filter{}); len(results) != 1 {
		t.Fatalf("before edit, results = %+v, want [a]", results)
	}
	if got := s.tfidf.buildsCount(); got != 1 {
		t.Fatalf("builds after first search = %d, want 1", got)
	}

	// 直接改写文件，绕过 Store.Save（不触发 invalidate）。改后的内容与
	// 查询词零重合：若缓存未失效，陈旧索引仍会返回 [a]，测试即失败。
	updated := Entry{Name: "a", Title: "另一篇", Category: CatOther, Body: "完全无关。"}
	content := RenderFrontmatter(updated) + updated.Body
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if results := Search(s, "旧标题", Filter{}); len(results) != 0 {
		t.Errorf("stale index leaked: results = %+v, want empty after out-of-band edit", results)
	}
	if got := s.tfidf.buildsCount(); got != 2 {
		t.Fatalf("builds after out-of-band edit + search = %d, want 2 (fingerprint miss)", got)
	}
}
