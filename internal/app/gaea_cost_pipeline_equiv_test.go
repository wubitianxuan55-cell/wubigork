package app

// 审计 GA6-04 app 绑定面钉子：costSearchAll 的管线编排收编到 cost.Enhance
// （与 tool 面共用）后，本面语义必须逐字段不变——
//   1. app 面不截断（rerank limit=20 只是精排候选上限，不是返回上限）；
//   2. 关键词命中 ≥3 不触发语义补召回（阈值唯一出处 cost.RecallBelow）；
//   3. 命中 <3 时语义补召回生效，语义命中并入结果。
// 读取错误通道（GA6-09）由 gaea_cost_search_page_error_test.go 覆盖，不重复。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// TestCostSearchAllAppFaceKeepsFullList：25 条全量命中（> rerank limit 20）
// 时 app 面不按 20 截断——limit 语义归调用面（tool 面才截断）。
func TestCostSearchAllAppFaceKeepsFullList(t *testing.T) {
	a := newCostSearchPageApp(t)
	for i := 13; i <= 25; i++ {
		e := cost.Entry{Name: "e13x" + string(rune('A'+i)), Title: "物料编号" + string(rune('A'+i)), Price: float64(i), Status: "现行"}
		if err := a.hubCostStore().Save(e); err != nil {
			t.Fatal(err)
		}
	}
	out, err := a.costSearchAll("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 25 {
		t.Fatalf("app 面返回 %d 条, want 25（不得按 rerank limit 20 截断）", len(out))
	}
}

// fakeAppEmbedServer 按「锤/液压→vec0、水泥→vec1」返回 1-hot 向量，并计数
// （计数用于断言「≥3 命中时不触发补召回」）。
func fakeAppEmbedServer(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"bge-m3"}]}`))
			return
		}
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		calls++
		data := make([]map[string]any, 0, len(req.Input))
		for i, s := range req.Input {
			vec := []float32{0, 0}
			if strings.Contains(s, "锤") || strings.Contains(s, "液压") {
				vec[0] = 1
			}
			if strings.Contains(s, "水泥") {
				vec[1] = 1
			}
			data = append(data, map[string]any{"index": i, "embedding": vec})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	return srv, &calls
}

// TestCostSearchAllRecallThresholdShared：共享管线的补召回阈值语义在 app
// 面成立——「振动锤」命中 ≥3 条时不做语义补召回（embedding 服务零请求）；
// 「打桩锤」零关键词命中时触发补召回，语义命中（HP300）并入结果。
func TestCostSearchAllRecallThresholdShared(t *testing.T) {
	a := newCostSearchPageApp(t)
	mustSave := func(name, title string, price float64) {
		t.Helper()
		if err := a.hubCostStore().Save(cost.Entry{
			Name: name, Title: title, Category: "机械", Unit: "台班", Price: price, Status: "现行",
		}); err != nil {
			t.Fatal(err)
		}
	}
	mustSave("hp300", "HP300 高频液压振动锤", 3200)

	srv, calls := fakeAppEmbedServer(t)
	defer srv.Close()
	SetAppEmbedderForTest(retrieval.NewEmbedder(srv.URL, "bge-m3"))
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	SetAppSemanticStoreForTest(semantic.Open(gdb))
	t.Cleanup(func() {
		SetAppEmbedderForTest(nil)
		SetAppSemanticStoreForTest(nil)
	})

	// ≥3 条关键词命中：不触发语义补召回。
	mustSave("vib-a", "振动锤 小型", 100)
	mustSave("vib-b", "振动锤 大型", 200)
	if _, err := a.costSearchAll("振动锤", "", ""); err != nil {
		t.Fatal(err)
	}
	if *calls != 0 {
		t.Fatalf("≥3 条命中时 embedding 服务被请求 %d 次, want 0（补召回阈值语义破坏）", *calls)
	}

	// 零关键词命中：触发补召回，语义命中 HP300 并入（打桩锤→vec0 与振动锤同向）。
	out, err := a.costSearchAll("打桩锤", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if *calls == 0 {
		t.Fatal("零命中未触发语义补召回（应请求 embedding 服务）")
	}
	found := false
	for _, s := range out {
		if s.Title == "HP300 高频液压振动锤" {
			found = true
		}
	}
	if !found {
		t.Fatalf("语义召回未并入 HP300: %+v", out)
	}
}
