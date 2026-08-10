package builtin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

func TestKnowledgeAddAndSearch(t *testing.T) {
	// 隔离真实知识库/数据库：注入临时文件 store。
	isoStore, isoErr := knowledge.Open(t.TempDir())
	if isoErr != nil {
		t.Fatal(isoErr)
	}
	knowledge.SetStoreForTest(isoStore)

	ka := knowledgeAdd{}
	result, err := ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title":    "测试知识条目",
		"category": "经验总结",
		"body":     "这是一条测试知识条目，用于验证知识库功能。",
	}))
	if err != nil {
		t.Fatalf("knowledgeAdd failed: %v", err)
	}
	if !strings.Contains(result, "已保存") {
		t.Errorf("expected success message, got: %s", result)
	}

	// Now search for it.
	ks := knowledgeSearch{}
	searchResult, err := ks.Execute(nil, toJSON(t, map[string]interface{}{
		"query": "测试知识条目",
	}))
	if err != nil {
		t.Fatalf("knowledgeSearch failed: %v", err)
	}
	if !strings.Contains(searchResult, "测试知识条目") {
		t.Errorf("expected to find added entry, got: %s", searchResult)
	}
}

func TestKnowledgeSearchOverview(t *testing.T) {
	isoStore, isoErr := knowledge.Open(t.TempDir())
	if isoErr != nil {
		t.Fatal(isoErr)
	}
	knowledge.SetStoreForTest(isoStore)

	// Add one entry first.
	ka := knowledgeAdd{}
	_, err := ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title":    "概览测试",
		"category": "规范标准",
		"body":     "概览测试正文",
	}))
	if err != nil {
		t.Fatal(err)
	}

	// Search with empty query should return overview.
	ks := knowledgeSearch{}
	result, err := ks.Execute(nil, toJSON(t, map[string]interface{}{}))
	if err != nil {
		t.Fatalf("knowledgeSearch overview failed: %v", err)
	}
	if !strings.Contains(result, "知识库概览") {
		t.Errorf("expected 知识库概览, got: %s", result)
	}
	if !strings.Contains(result, "规范标准") {
		t.Errorf("expected 规范标准 category, got: %s", result)
	}
}

func TestKnowledgeAddGeneratesUniqueName(t *testing.T) {
	isoStore, isoErr := knowledge.Open(t.TempDir())
	if isoErr != nil {
		t.Fatal(isoErr)
	}
	knowledge.SetStoreForTest(isoStore)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	ka := knowledgeAdd{}
	// Add the same title twice.
	r1, err := ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title":    "相同标题",
		"category": "其他",
		"body":     "第一次",
	}))
	if err != nil {
		t.Fatal(err)
	}
	r2, err := ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title":    "相同标题",
		"category": "其他",
		"body":     "第二次",
	}))
	if err != nil {
		t.Fatal(err)
	}
	// The file paths should be different.
	if r1 == r2 {
		t.Error("expected different file paths for duplicate titles")
	}
}

// TestKnowledgeSearchSemanticRecall 验证关键词召回不足时走本地 embedding 补召回：
// 「打桩锤」不命中任何标题/正文，但语义上应召回含「振动锤选型」的条目。
func TestKnowledgeSearchSemanticRecall(t *testing.T) {
	isoStore, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	knowledge.SetStoreForTest(isoStore)
	_ = isoStore.Save(knowledge.Entry{Name: "cement", Title: "水泥材料", Category: knowledge.CatMaterial, Body: "P.O 42.5 水泥", Status: "现行"})
	_ = isoStore.Save(knowledge.Entry{Name: "pile", Title: "桩基施工要点", Category: knowledge.CatCase, Body: "振动锤选型需匹配地质条件", Status: "现行"})

	dbDir := t.TempDir()
	gdb := db.GetDatabase(dbDir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dbDir)
	SetSemanticStoreForTest(semantic.Open(gdb))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
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
	defer srv.Close()
	SetEmbedderForTest(retrieval.NewEmbedder(srv.URL, "bge-m3"))

	ks := knowledgeSearch{}
	res, err := ks.Execute(nil, toJSON(t, map[string]interface{}{"query": "打桩锤"}))
	if err != nil {
		t.Fatalf("knowledgeSearch failed: %v", err)
	}
	if !strings.Contains(res, "桩基施工要点") {
		t.Fatalf("semantic recall should surface 桩基施工要点: %s", res)
	}
	SetEmbedderForTest(nil)
	SetSemanticStoreForTest(nil)
}

func TestKnowledgeSearchByCategory(t *testing.T) {
	isoStore, isoErr := knowledge.Open(t.TempDir())
	if isoErr != nil {
		t.Fatal(isoErr)
	}
	knowledge.SetStoreForTest(isoStore)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	ka := knowledgeAdd{}
	ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title": "案例A", "category": "工程案例", "body": "案例A正文",
	}))
	ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title": "规范B", "category": "规范标准", "body": "规范B正文",
	}))

	ks := knowledgeSearch{}
	result, err := ks.Execute(nil, toJSON(t, map[string]interface{}{
		"category": "工程案例",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "案例A") {
		t.Errorf("expected 案例A in filtered results, got: %s", result)
	}
	if strings.Contains(result, "规范B") {
		t.Errorf("did not expect 规范B in filtered results")
	}
}

func toJSON(t *testing.T, m map[string]interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}
