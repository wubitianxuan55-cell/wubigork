// Package repos — fts 全文搜索仓库测试
package repos

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// 测试临时数据根目录（每个用例独立子目录，避免库级单例互踩）
func ftsTestRoot(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "gaea-fts-test", t.Name())
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0o755)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// seedFact 插入一条事实并重建索引
func seedFact(t *testing.T, root string, f whisper.MemoryFact) string {
	t.Helper()
	if f.ID == "" {
		f.ID = "fact-" + t.Name() + "-" + f.Summary
	}
	f.CreatedAt = time.Now()
	f.UpdatedAt = time.Now()
	if err := ReplaceFactsInDB(root, []whisper.MemoryFact{f}); err != nil {
		t.Fatalf("ReplaceFactsInDB: %v", err)
	}
	if err := RebuildFactsFTS(root); err != nil {
		t.Fatalf("RebuildFactsFTS: %v", err)
	}
	return f.ID
}

func TestRebuildFactsFTS_IndexesAll(t *testing.T) {
	root := ftsTestRoot(t)
	seedFact(t, root, whisper.MemoryFact{
		Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "最好的朋友叫阿珍",
		Triggers: []string{"阿珍"}, Weight: 2, Confidence: 0.9,
	})
	seedFact(t, root, whisper.MemoryFact{
		Domain: "DAILY_LIFE", Subcategory: "ROUTINES", Subject: "用户", Summary: "她喜欢喝美式咖啡",
		Weight: 1, Confidence: 0.7,
	})

	ids, err := SearchFactIDsFTS(root, "咖啡", 10)
	if err != nil {
		t.Fatalf("SearchFactIDsFTS: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("「咖啡」应命中 1 条（LIKE 降级命中摘要「美式咖啡」），got %d: %v", len(ids), ids)
	}
}

func TestSearchFactIDsFTS_ChineseSubstringFallback(t *testing.T) {
	root := ftsTestRoot(t)
	id := seedFact(t, root, whisper.MemoryFact{
		Domain: "DAILY_LIFE", Subcategory: "ROUTINES", Subject: "用户", Summary: "她喜欢喝美式咖啡",
		Triggers: []string{"咖啡"}, Weight: 2, Confidence: 0.9,
	})

	// unicode61 整词匹配「咖啡」能命中；「美式」是「美式咖啡」整词的子串，MATCH 空 → 降级 LIKE
	ids, err := SearchFactIDsFTS(root, "美式", 10)
	if err != nil {
		t.Fatalf("SearchFactIDsFTS: %v", err)
	}
	if len(ids) != 1 || ids[0] != id {
		t.Errorf("「美式」应 LIKE 降级命中 id=%s, got %v", id, ids)
	}
}

func TestSearchFactIDsFTS_NoMatch(t *testing.T) {
	root := ftsTestRoot(t)
	seedFact(t, root, whisper.MemoryFact{
		Domain: "DAILY_LIFE", Subcategory: "ROUTINES", Subject: "用户", Summary: "她喜欢喝美式咖啡",
		Weight: 1, Confidence: 0.7,
	})

	ids, err := SearchFactIDsFTS(root, "量子力学", 10)
	if err != nil {
		t.Fatalf("SearchFactIDsFTS: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("无关查询应返回空, got %v", ids)
	}
}

func TestBuildLikePatterns(t *testing.T) {
	pats := buildLikePatterns("她喜欢喝咖啡")
	// 整句 + 去重 2-gram：她喜/喜欢/欢喝/喝咖/咖啡
	if len(pats) != 6 {
		t.Errorf("应得 1 整句 + 5 个 2-gram, got %d: %v", len(pats), pats)
	}
	// 含「咖啡」2-gram（中文关键词召回的关键）
	found := false
	for _, p := range pats {
		if p == "咖啡" {
			found = true
		}
	}
	if !found {
		t.Errorf("2-gram 应含「咖啡」, got %v", pats)
	}
}

func TestSearchEpisodesFTS_LikeFallback(t *testing.T) {
	root := ftsTestRoot(t)
	ep := whisper.Episode{
		ID: "ep-1", Summary: "一起喝了咖啡聊到深夜", Keywords: []string{"咖啡"},
		EmotionalIntensity: 0.7, CreatedAt: time.Now(),
	}
	if err := ReplaceEpisodesInDB(root, []whisper.Episode{ep}); err != nil {
		t.Fatalf("ReplaceEpisodesInDB: %v", err)
	}
	if err := RebuildEpisodesFTS(root); err != nil {
		t.Fatalf("RebuildEpisodesFTS: %v", err)
	}

	ids, err := SearchEpisodeIDsFTS(root, "咖啡", 10)
	if err != nil {
		t.Fatalf("SearchEpisodeIDsFTS: %v", err)
	}
	if len(ids) != 1 || ids[0] != "ep-1" {
		t.Errorf("「咖啡」应命中 ep-1, got %v", ids)
	}
}

// 复用 db 包连接（避免与生产单例冲突的隔离验证）
func TestFTS_IndexHasData(t *testing.T) {
	root := ftsTestRoot(t)
	seedFact(t, root, whisper.MemoryFact{
		Domain: "SOCIAL", Subcategory: "FRIENDS", Subject: "用户", Summary: "阿珍喜欢摄影",
		Weight: 2, Confidence: 0.9,
	})

	sqlDB, err := db.GetDatabase(root)
	if err != nil {
		t.Fatalf("db.GetDatabase: %v", err)
	}
	var n int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM memory_facts_fts").Scan(&n); err != nil {
		t.Fatalf("查询 FTS 表: %v", err)
	}
	if n != 1 {
		t.Errorf("FTS 索引应有 1 行, got %d", n)
	}
}
