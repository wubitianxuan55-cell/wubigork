package app

import (
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// GaeaMemoryDuplicates 向量臂（Nebula 蒸馏 P3）：语义同义对（换说法重复，词法
// 臂抓不到）进同一评审面板，Source=vector；embedder 缺席时该臂静默缺席。
func TestGaeaMemoryDuplicatesVectorArm(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	gdb := db.GetDatabase(gaeaConfig.MemoryUserDir())
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(gaeaConfig.MemoryUserDir())

	s := memory.SQLiteStoreFor(gdb, gaeaConfig.MemoryUserDir(), "/Users/me/proj")
	// hotpot-v2 与 hotpot 预置同向向量（cos=1，换说法重复，词法不命中）；
	// orthogonal 独立向量；三行全种 → counts 与 docs 一致 → Ensure 跳过（stub
	// embedder 不会被真实调用）。
	// 描述/正文刻意互不含同词（词法臂抓不到），只有向量同向才该成对。
	for _, m := range []memory.Memory{
		{Name: "hotpot", Description: "喜欢吃辣", Body: "无辣不欢"},
		{Name: "hotpot-v2", Description: "锅底要麻", Body: "重口拉满"},
		{Name: "orthogonal", Description: "周会模板", Body: "每周五同步进展"},
	} {
		if _, err := s.Save(m); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range []struct{ id, vec string }{
		{"hotpot", "[1,0]"},
		{"hotpot-v2", "[1,0]"},
		{"orthogonal", "[0,1]"},
	} {
		if _, err := gdb.Exec(`INSERT INTO semantic_vectors(kind,id,vec,doc,updated_at) VALUES('office',?,?, '', '')`, r.id, r.vec); err != nil {
			t.Fatal(err)
		}
	}

	SetOfficeStoreForTest(s)
	defer ResetOfficeStoreForTest()
	SetAppSemanticStoreForTest(semantic.Open(gdb))
	defer SetAppSemanticStoreForTest(nil)
	// 零值 Embedder 可构造且不会被真实调用（counts 已一致跳过 Ensure）。
	SetAppEmbedderForTest(&retrieval.Embedder{BaseURL: "http://127.0.0.1:9", Model: "stub"})
	defer SetAppEmbedderForTest(nil) // 无独立 Reset（与既有测试同口径：归 nil=走真实解析）

	a := &App{core: &core{}}
	views := a.GaeaMemoryDuplicates(0.55)
	var vecPair *MemoryDuplicateView
	for i := range views {
		if views[i].Source == "vector" {
			vecPair = &views[i]
		}
	}
	if vecPair == nil {
		t.Fatalf("向量臂缺席：views=%+v", views)
	}
	if (vecPair.Keep != "hotpot" || vecPair.Dup != "hotpot-v2") && (vecPair.Keep != "hotpot-v2" || vecPair.Dup != "hotpot") {
		t.Fatalf("向量对错位：%+v", vecPair)
	}
	if vecPair.Score < 0.85 {
		t.Fatalf("向量对分=%v 低于 0.85", vecPair.Score)
	}
	// orthogonal 不应与任何条目成对（正交向量）。
	for _, v := range views {
		if v.Source == "vector" && (v.Keep == "orthogonal" || v.Dup == "orthogonal") {
			t.Fatalf("正交向量误成对：%+v", v)
		}
	}

	// embedder 缺席 → 向量臂静默缺席，词法臂（fuzzy）独立工作。
	SetAppEmbedderForTest(nil)
	a2 := &App{core: &core{}}
	for _, v := range a2.GaeaMemoryDuplicates(0.55) {
		if v.Source == "vector" {
			t.Fatalf("无 embedder 仍出向量臂：%+v", v)
		}
	}
}
