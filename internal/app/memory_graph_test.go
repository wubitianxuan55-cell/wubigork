package app

import (
	"testing"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// TestGaeaMemoryGraph_WhisperTriples 轻语三元组入图：实体节点（去重合并）+ 关系边。
func TestGaeaMemoryGraph_WhisperTriples(t *testing.T) {
	dir := t.TempDir()
	db.GetDatabase(dir)
	defer db.CloseDatabase(dir)

	// 写入事实（供 whisper 节点）+ 三元组
	fs := whisper.NewFactStore()
	fs.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "喜欢吃辣", Weight: 2, Confidence: 0.9,
	})
	all := fs.ListAll()
	raw := make([]whisper.MemoryFact, 0, len(all))
	for _, f := range all {
		raw = append(raw, f.MemoryFact)
	}
	if err := repos.ReplaceFactsInDB(dir, raw); err != nil {
		t.Fatalf("ReplaceFactsInDB: %v", err)
	}
	if err := repos.ReplaceTriplesInDB(dir, []whisper.Triple{
		{ID: "t1", Subject: "用户", Predicate: "喜欢", Object: "辣", Confidence: 1},
		{ID: "t2", Subject: "用户", Predicate: "职业", Object: "程序员", Confidence: 0.8},
	}); err != nil {
		t.Fatalf("ReplaceTriplesInDB: %v", err)
	}

	a := &App{whisperState: &whisperState{whisperDataRoot: dir}}
	g := a.GaeaMemoryGraph()

	hasNode := make(map[string]bool)
	for _, n := range g.Nodes {
		hasNode[n.ID] = true
	}
	for _, id := range []string{"t:用户", "t:辣", "t:程序员"} {
		if !hasNode[id] {
			t.Fatalf("三元组实体节点缺失: %s (nodes=%v)", id, hasNode)
		}
	}
	// 实体去重：两个三元组共享"用户" → 只有一个 t:用户 节点
	count := 0
	for _, n := range g.Nodes {
		if n.ID == "t:用户" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("t:用户 节点应去重为 1 个, got %d", count)
	}

	// 关系边：用户→辣 / 用户→程序员（predicate 作边类型）
	linkCnt := 0
	for _, l := range g.Links {
		if l.Source == "t:用户" && (l.Target == "t:辣" || l.Target == "t:程序员") {
			linkCnt++
		}
	}
	if linkCnt != 2 {
		t.Fatalf("三元组关系边 = %d, want 2", linkCnt)
	}
}
