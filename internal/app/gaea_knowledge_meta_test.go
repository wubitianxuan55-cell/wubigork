package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/modelengine"
)

func newKnowledgeMetaEnv(t *testing.T) *knowledge.Store {
	t.Helper()
	isoStore, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	knowledge.SetStoreForTest(isoStore)
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() {
		db.CloseDatabase(dir)
		SetKnowledgeHistoryDBForTest(nil)
		knowledge.SetStoreForTest(nil)
	})
	SetKnowledgeHistoryDBForTest(gdb)
	return isoStore
}

func TestSaveKnowledgeVersionedAndHistory(t *testing.T) {
	store := newKnowledgeMetaEnv(t)
	a := &App{}

	if err := saveKnowledgeVersioned(store, knowledge.Entry{
		Name: "pile", Title: "桩基施工要点", Category: knowledge.CatCase,
		Body: "旧正文", Status: "现行",
	}); err != nil {
		t.Fatal(err)
	}
	if err := saveKnowledgeVersioned(store, knowledge.Entry{
		Name: "pile", Title: "桩基施工要点", Category: knowledge.CatCase,
		Body: "新正文", Status: "现行", Version: 1,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("pile")
	if err != nil || got.Version != 2 {
		t.Fatalf("version = %d, err=%v; want 2", got.Version, err)
	}
	hist := a.GaeaKnowledgeHistory("pile")
	if len(hist) != 1 || hist[0].Body != "旧正文" || hist[0].Version != 1 {
		t.Errorf("history = %+v, want 1 条旧正文 v1", hist)
	}
}

func TestGaeaKnowledgeExport(t *testing.T) {
	store := newKnowledgeMetaEnv(t)
	if err := store.Save(knowledge.Entry{Name: "pile", Title: "桩基施工要点", Category: knowledge.CatCase, Body: "要点", Status: "现行"}); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	dir := t.TempDir()
	n, err := a.GaeaKnowledgeExport(dir)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("exported %d, want 1", n)
	}
	b, err := os.ReadFile(filepath.Join(dir, knowledge.FileName(knowledge.Entry{Name: "pile", Title: "桩基施工要点"})))
	if err != nil || len(b) == 0 {
		t.Fatalf("export file missing: %v", err)
	}
}

func TestGaeaKnowledgeReview(t *testing.T) {
	store := newKnowledgeMetaEnv(t)
	a := &App{}
	if err := store.Save(knowledge.Entry{Name: "draft1", Title: "待审条目", Category: knowledge.CatCase, Body: "正文", Status: "草稿", Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := a.GaeaKnowledgeReview("draft1", true, "张三"); err != nil {
		t.Fatal(err)
	}
	e, err := store.Get("draft1")
	if err != nil || e.Status != "现行" || e.Reviewer != "张三" {
		t.Fatalf("after review = %+v, err=%v; want 现行 + 审核人", e, err)
	}
	hist := a.GaeaKnowledgeHistory("draft1")
	if len(hist) != 1 || hist[0].Note != "审核通过" {
		t.Errorf("history = %+v, want 审核通过 留档", hist)
	}
}

func TestGaeaKnowledgeMerge(t *testing.T) {
	store := newKnowledgeMetaEnv(t)
	a := &App{}
	_ = store.Save(knowledge.Entry{Name: "a", Title: "桩基施工要点", Category: knowledge.CatCase, Tags: []string{"桩基"}, Body: "正文A", Status: "现行", Version: 1})
	_ = store.Save(knowledge.Entry{Name: "b", Title: "桩基施工要点（修订）", Category: knowledge.CatCase, Tags: []string{"振动锤", "桩基"}, Body: "正文B", Status: "现行", Version: 1})

	merged, err := a.GaeaKnowledgeMerge("a", []string{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if merged != "a" {
		t.Errorf("merged = %q, want a", merged)
	}
	if _, err := store.Get("b"); err == nil {
		t.Error("source b should be deleted")
	}
	ea, err := store.Get("a")
	if err != nil {
		t.Fatal(err)
	}
	if len(ea.Tags) != 2 {
		t.Errorf("merged tags = %v, want 并集 2 个", ea.Tags)
	}
	hist := a.GaeaKnowledgeHistory("a")
	if len(hist) != 1 || !strings.Contains(hist[0].Note, "合并自") {
		t.Errorf("history = %+v, want 合并自 留档", hist)
	}
}

// 知识库绑定面 nil 切片回归钉（对齐 v4.467.1 造价同款契约）：无标签条目的
// Tags 在 store 内为 nil，Wails 序列化成 JSON null，前端 entry.tags.length
// 崩进 ErrorBoundary（2026-10-06 真机日志实锤）。所有知识库绑定出口切片
// 构造即非 nil；此处同时钉 Go 值与 JSON 序列化两层。
func TestKnowledgeBindingsNonNilSlices(t *testing.T) {
	store := newKnowledgeMetaEnv(t)
	// 检索族地雷：裸 &App{} 的 core 为 nil，语义召回读提升字段 engineMgr 即崩；
	// 空 engine manager = herdsman 断通道，召回/精排自然降级跳过。
	a := &App{core: &core{engineMgr: modelengine.NewManager("", "")}}

	if err := store.Save(knowledge.Entry{Name: "notags", Title: "桩基施工要点", Category: knowledge.CatCase, Body: "正文", Status: "现行"}); err != nil {
		t.Fatal(err)
	}

	list := a.GaeaKnowledgeList()
	if len(list) != 1 || list[0].Tags == nil {
		t.Fatalf("GaeaKnowledgeList Tags = nil, want 非 nil 空切片（JSON null 崩前端）")
	}
	search := a.GaeaKnowledgeSearch("桩基", "all", "all", "all")
	if len(search) != 1 || search[0].Tags == nil {
		t.Fatalf("GaeaKnowledgeSearch Tags = nil, want 非 nil 空切片")
	}
	entry := a.GaeaKnowledgeGet("notags")
	if entry == nil || entry.Tags == nil {
		t.Fatalf("GaeaKnowledgeGet Tags = nil, want 非 nil 空切片")
	}
	if hist := a.GaeaKnowledgeHistory("notags"); hist == nil {
		t.Error("GaeaKnowledgeHistory 空历史 = nil, want 非 nil 空切片")
	}
	if sim := a.GaeaKnowledgeFindSimilar("毫不相关的标题xyz"); sim == nil {
		t.Error("GaeaKnowledgeFindSimilar 无命中 = nil, want 非 nil 空切片")
	}

	// JSON 序列化层：绑定出口不得再出现 "tags":null
	blob, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), `"tags":null`) {
		t.Errorf("序列化出现 tags:null: %s", blob)
	}
}
