package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// TestParseRetrievalEvalSet 查询集文档格式解析：```json 代码块 → 条目；
// 缺代码块 / JSON 非法 → 错误。
func TestParseRetrievalEvalSet(t *testing.T) {
	doc := "# 检索质量受控测评查询集\n\n> 说明\n\n" +
		"```json\n" +
		"[\n" +
		"  {\"query\": \"打桩设备 台班价\", \"expected\": [{\"kind\": \"cost\", \"name\": \"hp300\"}]},\n" +
		"  {\"query\": \"P.O 42.5 水泥 价格\", \"expected\": [{\"kind\": \"cost\", \"name\": \"cement\"}, {\"kind\": \"file\", \"name\": \"docs/水泥.md\"}]}\n" +
		"]\n" +
		"```\n"
	items, err := parseRetrievalEvalSet([]byte(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Query != "打桩设备 台班价" || len(items[0].Expected) != 1 ||
		items[0].Expected[0].Kind != "cost" || items[0].Expected[0].Name != "hp300" {
		t.Errorf("item0 = %+v", items[0])
	}
	if len(items[1].Expected) != 2 || items[1].Expected[1].Kind != "file" ||
		items[1].Expected[1].Name != "docs/水泥.md" {
		t.Errorf("item1 = %+v", items[1])
	}

	// 缺少 JSON 代码块 → 错误
	if _, err := parseRetrievalEvalSet([]byte("# 无查询集\n")); err == nil {
		t.Error("缺少 json 代码块应报错")
	}
	// JSON 非法 → 错误
	if _, err := parseRetrievalEvalSet([]byte("```json\nnot-json\n```\n")); err == nil {
		t.Error("非法 JSON 应报错")
	}
}

// TestRetrievalEvalRecallMath 召回率计算与命中判定单测：
// 精确命中 / name 子串命中 / kind 不匹配不算 / 部分命中 / 空预期。
func TestRetrievalEvalRecallMath(t *testing.T) {
	// 精确命中
	if r := evalRecall([]string{"cost:hp300"}, []string{"cost:hp300", "knowledge:x"}); r != 1 {
		t.Errorf("exact recall = %v, want 1", r)
	}
	// name 子串命中（hit name 包含预期 name）
	if r := evalRecall([]string{"cost:水泥"}, []string{"cost:水泥（散装）"}); r != 1 {
		t.Errorf("substring recall = %v, want 1", r)
	}
	// 反向子串（预期 name 包含 hit name）也算命中
	if r := evalRecall([]string{"file:docs/投标文件模板.md（2026）"}, []string{"file:docs/投标文件模板.md"}); r != 1 {
		t.Errorf("reverse substring recall = %v, want 1", r)
	}
	// kind 不同不算命中
	if r := evalRecall([]string{"cost:水泥"}, []string{"knowledge:水泥"}); r != 0 {
		t.Errorf("kind mismatch recall = %v, want 0", r)
	}
	// 部分命中：1/2
	if r := evalRecall([]string{"cost:a", "file:b"}, []string{"cost:a"}); r != 0.5 {
		t.Errorf("partial recall = %v, want 0.5", r)
	}
	// 空预期记 0
	if r := evalRecall(nil, []string{"cost:a"}); r != 0 {
		t.Errorf("empty expected recall = %v, want 0", r)
	}
}

// TestRetrievalEvalRunMini 端到端测评：mock embedder（关键词→确定性命中向量）+
// 临时 semantic store（四库种子数据），迷你查询集内嵌（不依赖 docs 文件），
// 断言逐条 recall、recall@10、0.8 阈值 passed 判定。
func TestRetrievalEvalRunMini(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() {
		db.CloseDatabase(dir)
		SetAppSemanticStoreForTest(nil)
		SetAppEmbedderForTest(nil)
		ResetCostStoreForTest()
		ResetOfficeStoreForTest()
		knowledge.ResetForTest()
	})

	// mock embedder：含关键词的文本得到确定性命中向量
	// （锤/桩/打桩 → dim0；水泥 → dim1；投标 → dim2），余弦 1 命中、0 淘汰。
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
			vec := []float32{0, 0, 0}
			if strings.Contains(s, "锤") || strings.Contains(s, "桩") || strings.Contains(s, "打桩") {
				vec[0] = 1
			}
			if strings.Contains(s, "水泥") {
				vec[1] = 1
			}
			if strings.Contains(s, "投标") {
				vec[2] = 1
			}
			data = append(data, map[string]any{"index": i, "embedding": vec})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()

	e := retrieval.NewEmbedder(srv.URL, "bge-m3")
	st := semantic.Open(gdb)
	SetAppSemanticStoreForTest(st)
	SetAppEmbedderForTest(e)

	// 成本库种子：3 条
	cs := cost.Open(gdb)
	SetCostStoreForTest(cs)
	seeds := []cost.Entry{
		{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Spec: "300kW", Source: "市场询价", Status: "现行"},
		{Name: "cement", Title: "P.O 42.5 水泥", Category: "材料", Unit: "吨", Price: 480, Source: "定额", Status: "现行"},
		{Name: "digger220", Title: "挖掘机 220", Category: "机械", Unit: "台班", Price: 2600, Source: "市场询价", Status: "现行"},
	}
	for _, s := range seeds {
		if err := cs.Save(s); err != nil {
			t.Fatal(err)
		}
	}

	// 知识库种子：3 条（OpenSQLite + SetStoreForTest 隔离真实用户库）
	ks, err := knowledge.OpenSQLite(gdb)
	if err != nil {
		t.Fatal(err)
	}
	knowledge.SetStoreForTest(ks)
	knowSeeds := []knowledge.Entry{
		{Name: "vib-guide", Title: "振动锤选型要点", Category: knowledge.CatExperience, Body: "根据地质与桩型选择激振力", Status: "现行"},
		{Name: "bid-format", Title: "投标文件格式要求", Category: knowledge.CatOther, Body: "商务标与技术标目录", Status: "现行"},
		{Name: "pile-notes", Title: "桩基施工要点", Category: knowledge.CatCase, Body: "打桩顺序与垂直度控制", Status: "现行"},
	}
	for _, s := range knowSeeds {
		if err := ks.Save(s); err != nil {
			t.Fatal(err)
		}
	}

	// 办公记忆种子：1 条（SQLite 后端 name 会 slug 化，用纯 ASCII name）
	ms := memory.SQLiteStoreFor(gdb, dir, dir)
	SetOfficeStoreForTest(ms)
	if _, err := ms.Save(memory.Memory{
		Name:        "vib-select",
		Title:       "桩基施工-振动锤选型",
		Description: "振动锤选型需匹配地质",
		Tags:        []string{"振动锤", "桩基"},
		Body:        "HP300 用于 800 管桩",
	}); err != nil {
		t.Fatal(err)
	}

	// 工作区文件向量：直接预热 "file" kind（检索只查已持久化向量，不走扫描）。
	if _, err := st.Ensure(context.Background(), e, "file", []semantic.Doc{
		{ID: "docs/投标文件模板.md", Text: "投标文件格式要求 商务标 技术标"},
		{ID: "docs/水泥价格表.md", Text: "P.O 42.5 水泥 吨 480"},
	}); err != nil {
		t.Fatal(err)
	}

	// 迷你查询集（内嵌，不依赖 docs 文件）：每条均有确定性预期命中。
	items := []RetrievalEvalItem{
		{Query: "打桩锤 台班 价格", Expected: []RetrievalEvalHit{{Kind: "cost", Name: "hp300"}}},
		{Query: "P.O 42.5 水泥 多少钱", Expected: []RetrievalEvalHit{{Kind: "cost", Name: "cement"}, {Kind: "file", Name: "docs/水泥价格表.md"}}},
		{Query: "振动锤 选型 要点", Expected: []RetrievalEvalHit{{Kind: "office", Name: "vib-select"}, {Kind: "knowledge", Name: "vib-guide"}}},
		{Query: "投标文件 格式 要求", Expected: []RetrievalEvalHit{{Kind: "file", Name: "docs/投标文件模板.md"}, {Kind: "knowledge", Name: "bid-format"}}},
	}
	a := &App{}
	rep, err := a.runRetrievalEval(items)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Total != 4 {
		t.Errorf("total = %d, want 4", rep.Total)
	}
	if rep.Threshold != 0.8 {
		t.Errorf("threshold = %v, want 0.8", rep.Threshold)
	}
	if len(rep.PerQuery) != 4 {
		t.Fatalf("perQuery = %d, want 4", len(rep.PerQuery))
	}
	for _, pq := range rep.PerQuery {
		if pq.Recall != 1 {
			t.Errorf("query %q recall = %v, want 1（topHits=%v expected=%v）", pq.Query, pq.Recall, pq.TopHits, pq.Expected)
		}
		if len(pq.TopHits) == 0 {
			t.Errorf("query %q 无命中", pq.Query)
		}
	}
	if rep.RecallAt10 != 1 {
		t.Errorf("recallAt10 = %v, want 1", rep.RecallAt10)
	}
	if !rep.Passed {
		t.Error("passed = false, want true（recallAt10=1 >= 0.8）")
	}
}

// TestRetrievalEvalRunEngineUnavailable 引擎不可用（本地 embedding 未配置）：
// 返回 error 提示先启用 Herdsman bge-m3，不触碰真实库。
func TestRetrievalEvalRunEngineUnavailable(t *testing.T) {
	// 先清掉可能残留的测试注入，保证走「未配置」分支。
	SetAppEmbedderForTest(nil)
	SetAppSemanticStoreForTest(nil)
	a := &App{core: &core{}} // core 非空：engineMgr 为 nil 即「embedding 未配置」
	_, err := a.runRetrievalEval([]RetrievalEvalItem{
		{Query: "打桩锤", Expected: []RetrievalEvalHit{{Kind: "cost", Name: "hp300"}}},
	})
	if err == nil {
		t.Fatal("embedding 未配置应返回 error")
	}
	if !strings.Contains(err.Error(), "bge-m3") {
		t.Errorf("error 应提示启用 bge-m3: %v", err)
	}
}
