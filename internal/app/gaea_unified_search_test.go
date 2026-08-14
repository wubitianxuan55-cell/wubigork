package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// newUnifiedSearchEnv 搭建统一检索测试环境：临时用户目录 + mock embedding 服务
// + 临时 Hephaestus.db；成本/办公/知识/向量索引全部注入隔离存储，避免触碰真实库。
func newUnifiedSearchEnv(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"data":[{"id":"bge-m3"}]}`))
			return
		}
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		data := make([]map[string]any, 0, len(req.Input))
		for i, s := range req.Input {
			vec := []float32{0, 0}
			if strings.Contains(s, "锤") || strings.Contains(s, "桩") {
				vec[0] = 1
			}
			if strings.Contains(s, "水泥") {
				vec[1] = 1
			}
			data = append(data, map[string]any{"index": i, "embedding": vec})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(srv.Close)

	gdb := db.GetDatabase(home)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() {
		db.CloseDatabase(home)
		SetAppSemanticStoreForTest(nil)
		SetAppEmbedderForTest(nil)
		knowledge.SetStoreForTest(nil)
		ResetOfficeStoreForTest()
		ResetCostStoreForTest()
	})
	SetAppSemanticStoreForTest(semantic.Open(gdb))
	SetAppEmbedderForTest(retrieval.NewEmbedder(srv.URL, "bge-m3"))
	SetOfficeStoreForTest(memory.SQLiteStoreFor(gdb, home, home))
	SetCostStoreForTest(cost.Open(gdb))

	isoStore, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	knowledge.SetStoreForTest(isoStore)
}

// TestGaeaUnifiedSearch_Combined 组合结果正确性：一次调用同时返回
// 关键词全文命中（工作区文件）与跨库语义命中（知识/成本/办公三类）。
func TestGaeaUnifiedSearch_Combined(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/方案.md", []byte("振动锤选型要点：需匹配地质条件。"), 0o644); err != nil {
		t.Fatal(err)
	}
	newUnifiedSearchEnv(t)

	// 工程知识库条目（语义跨库命中：knowledge）
	ks, err := knowledge.Global().Store()
	if err != nil {
		t.Fatal(err)
	}
	if err := ks.Save(knowledge.Entry{
		Name: "pile-guide", Title: "桩基施工要点", Category: knowledge.CatCase,
		Body: "振动锤选型需匹配地质，桩基施工工艺详见规范。",
	}); err != nil {
		t.Fatal(err)
	}

	// 显式填充嵌入的 *core：embedding 未配置时 localSearchEmbedder 会走到
	// resolveHerdsmanSearchModel（读 a.engineMgr），零值 App 的 core 为 nil 会崩溃。
	a := &App{core: &core{}}
	// 成本条目（语义跨库命中：cost）
	if err := a.hubCostStore().Save(cost.Entry{
		Name: "pile-rent", Title: "振动锤租赁台班", Category: "机械", Unit: "台班", Price: 6500,
	}); err != nil {
		t.Fatal(err)
	}
	// 办公记忆（语义跨库命中：office）
	if _, err := a.hubOfficeStore().Save(memory.Memory{
		Name: "office-pile", Title: "打桩机械台账", Description: "振动锤与静压桩机配置", Body: "记录现场机械调度。",
	}); err != nil {
		t.Fatal(err)
	}

	view, err := a.GaeaUnifiedSearch("振动锤", 10)
	if err != nil {
		t.Fatalf("GaeaUnifiedSearch: %v", err)
	}

	// 关键词段：命中工作区文件 docs/方案.md
	kwHit := false
	for _, h := range view.Keyword {
		if h.Path == "docs/方案.md" {
			kwHit = true
			if !strings.Contains(h.Snippet, "振动锤") {
				t.Errorf("关键词片段应含查询词: %q", h.Snippet)
			}
		}
	}
	if !kwHit {
		t.Fatalf("关键词未命中 docs/方案.md: %+v", view.Keyword)
	}

	// 语义段：跨库命中 knowledge/cost/office 三种
	wantSem := map[string]bool{
		"knowledge/pile-guide": false,
		"cost/pile-rent":       false,
		"office/office-pile":   false,
	}
	for _, h := range view.Semantic {
		key := h.Kind + "/" + h.Name
		if _, ok := wantSem[key]; ok {
			wantSem[key] = true
		}
	}
	for key, found := range wantSem {
		if !found {
			t.Errorf("语义未命中 %s: %+v", key, view.Semantic)
		}
	}
}

// TestGaeaUnifiedSearch_EmptyQuery 空 query 返回空视图（两组均为空数组），不报错。
func TestGaeaUnifiedSearch_EmptyQuery(t *testing.T) {
	// 显式填充嵌入的 *core：embedding 未配置时 localSearchEmbedder 会走到
	// resolveHerdsmanSearchModel（读 a.engineMgr），零值 App 的 core 为 nil 会崩溃。
	a := &App{core: &core{}}
	view, err := a.GaeaUnifiedSearch("", 10)
	if err != nil {
		t.Fatalf("空 query 不应报错: %v", err)
	}
	if len(view.Keyword) != 0 || len(view.Semantic) != 0 {
		t.Fatalf("空 query 应返回空视图: %+v", view)
	}
	if view.Keyword == nil || view.Semantic == nil {
		t.Fatalf("空 query 也应返回空数组（非 nil，JSON 序列化为 []）: %+v", view)
	}
	view2, err := a.GaeaUnifiedSearch("   ", 10)
	if err != nil {
		t.Fatalf("空白 query 不应报错: %v", err)
	}
	if len(view2.Keyword) != 0 || len(view2.Semantic) != 0 {
		t.Fatalf("空白 query 应返回空视图: %+v", view2)
	}
}

// TestGaeaUnifiedSearch_EmbeddingUnavailable embedding 不可用降级：
// semantic 为空数组而 keyword 照常返回（与现有降级行为一致）。
func TestGaeaUnifiedSearch_EmbeddingUnavailable(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/成本测算.md", []byte("本项目成本测算总金额为 100 万元。"), 0o644); err != nil {
		t.Fatal(err)
	}
	newUnifiedSearchEnv(t)
	// 模拟 embedding 未配置/引擎未启动
	SetAppEmbedderForTest(nil)

	// 显式填充嵌入的 *core：embedding 未配置时 localSearchEmbedder 会走到
	// resolveHerdsmanSearchModel（读 a.engineMgr），零值 App 的 core 为 nil 会崩溃。
	a := &App{core: &core{}}
	view, err := a.GaeaUnifiedSearch("成本", 10)
	if err != nil {
		t.Fatalf("embedding 不可用不应报错: %v", err)
	}
	if view.Semantic == nil {
		t.Fatal("semantic 应为空数组（非 nil），保证 JSON 序列化为 []")
	}
	if len(view.Semantic) != 0 {
		t.Fatalf("embedding 不可用时 semantic 应为空: %+v", view.Semantic)
	}
	kwHit := false
	for _, h := range view.Keyword {
		if h.Path == "docs/成本测算.md" {
			kwHit = true
		}
	}
	if !kwHit {
		t.Fatalf("embedding 不可用时关键词仍应命中: %+v", view.Keyword)
	}
}
