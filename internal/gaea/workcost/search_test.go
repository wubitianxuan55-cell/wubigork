package workcost

// search_test.go — 造价库混合检索：关键词退化、编码精确短路、同义词经向量路召回、精排重排、
// 向量路不可用时让路、清单跨项目检索。

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/vecindex"
)

// bagEmbedder 测试用的「语义」向量化：按词桶给 one-hot（同义词落同一桶），
// 都不命中的落最后一维——无关文本之间余弦为 0，被索引的下限挡掉。
type bagEmbedder struct {
	buckets [][]string
}

var searchBuckets = [][]string{
	{"挖掘机", "挖机"},
	{"水泥", "砼", "混凝土"},
	{"压路机", "碾压"},
}

func (e *bagEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, len(e.buckets)+1)
		v[len(e.buckets)] = 1
		for b, words := range e.buckets {
			for _, w := range words {
				if strings.Contains(t, w) {
					v = make([]float32, len(e.buckets)+1)
					v[b] = 1
					break
				}
			}
		}
		out[i] = v
	}
	return out, nil
}

// pickReranker 把正文含 marker 的那条排到最前，其余保持原序。
type pickReranker struct{ marker string }

func (r pickReranker) Rerank(_ context.Context, _ string, docs []string, _ int) ([]int, error) {
	out := []int{}
	for i, d := range docs {
		if strings.Contains(d, r.marker) {
			out = append(out, i)
		}
	}
	for i := range docs {
		if !strings.Contains(docs[i], r.marker) {
			out = append(out, i)
		}
	}
	return out, nil
}

func searchStore(t *testing.T) (*Store, *sql.DB, func()) {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase 返回 nil")
	}
	s := Open(gdb)
	seedDict(t, s, DictSpecialty, "市政", "建筑工程")
	seedDict(t, s, DictRegion, "四川省")
	return s, gdb, func() {
		vecindex.Forget(gdb)
		db.CloseDatabase(dir)
	}
}

func buildIndex(t *testing.T, gdb *sql.DB, s *Store, emb vecindex.Embedder) *vecindex.Store {
	t.Helper()
	vs := vecindex.Open(gdb)
	for kind, src := range s.IndexSources() {
		if _, err := vs.Apply(context.Background(), kind, "bge-m3", src, emb, 128); err != nil {
			t.Fatalf("建索引 %s: %v", kind, err)
		}
	}
	return vs
}

func saveSearchFixtures(t *testing.T, s *Store) {
	t.Helper()
	for _, r := range []Resource{
		{Kind: KindMachine, Title: "单斗挖掘机 挖土", Spec: "斗容 1m3", Unit: "m3"},
		{Kind: KindMachine, Title: "静压桩机", Unit: "台班"},
		{Kind: KindMaterial, Title: "水泥砂浆 砌筑", Unit: "m3"},
		{Kind: KindMaterial, Title: "混凝土外加剂", Unit: "t"},
		{Kind: KindLabor, Title: "综合工日", Unit: "工日"},
	} {
		if _, err := s.SaveResource(r); err != nil {
			t.Fatalf("存资源 %s: %v", r.Title, err)
		}
	}
	for _, q := range []Quota{
		{Code: "SZ-TF-001", Title: "挖掘机挖土", Specialty: "市政", Chapter: "土方工程", Unit: "m3"},
		{Code: "JZ-QT-001", Title: "砌砖墙", Specialty: "建筑工程", Chapter: "砌筑工程", Unit: "m3"},
	} {
		if _, err := s.SaveQuota(q, nil); err != nil {
			t.Fatalf("存定额 %s: %v", q.Code, err)
		}
	}
}

func hitTitles[T any](hits []Hit[T], title func(T) string) []string {
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = title(h.Item)
	}
	return out
}

// 没有模型：退化为关键词检索（与原 LIKE 列表同序同集），每条标「关键词」；空查询词按列表返回。
func TestSearch_KeywordOnlyWithoutModels(t *testing.T) {
	s, _, cleanup := searchStore(t)
	defer cleanup()
	saveSearchFixtures(t, s)
	h := NewSearcher(s, nil, nil, nil)

	hits, err := h.Resources(context.Background(), SearchQuery{Text: "水泥"})
	if err != nil {
		t.Fatal(err)
	}
	if got := hitTitles(hits, func(r Resource) string { return r.Title }); len(got) != 1 || got[0] != "水泥砂浆 砌筑" {
		t.Fatalf("关键词应只命中名称含「水泥」的一条: %v", got)
	}
	if hits[0].Via != ViaKeyword {
		t.Fatalf("命中方式=%q, want keyword", hits[0].Via)
	}
	// 类型过滤照常生效：机械类里没有含「水泥」的
	if hits, _ = h.Resources(context.Background(), SearchQuery{Text: "水泥", Kind: KindMachine}); len(hits) != 0 {
		t.Fatalf("类型过滤没生效: %+v", hits)
	}
	// 空查询词 = 列表
	list, err := h.Resources(context.Background(), SearchQuery{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Via != ViaList {
		t.Fatalf("空查询词应按列表返回 2 条: %+v", list)
	}
	// 条数上限
	if hits, _ = h.Resources(context.Background(), SearchQuery{Text: "机", Limit: 1}); len(hits) != 1 {
		t.Fatalf("Limit=1 应只返回 1 条: %+v", hits)
	}
}

// 编码 / 自有码精确命中：只返回它，恒排最前，不再补近似。
func TestSearch_ExactCodeShortCircuits(t *testing.T) {
	s, _, cleanup := searchStore(t)
	defer cleanup()
	saveSearchFixtures(t, s)
	h := NewSearcher(s, nil, nil, nil)

	hits, err := h.Quotas(context.Background(), SearchQuery{Text: "sz-tf-001"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Item.Code != "SZ-TF-001" || hits[0].Via != ViaCode {
		t.Fatalf("编码精确应只返回那一条: %+v", hits)
	}
	// 工料机按自有码（gcode）精确命中
	all, err := h.Resources(context.Background(), SearchQuery{})
	if err != nil || len(all) == 0 {
		t.Fatalf("列表: %v %v", err, all)
	}
	gcode := all[0].Item.GCode
	if gcode == "" {
		t.Skip("资源没有自有码（旧库），跳过")
	}
	rh, err := h.Resources(context.Background(), SearchQuery{Text: gcode})
	if err != nil || len(rh) != 1 || rh[0].Item.Code != all[0].Item.Code || rh[0].Via != ViaCode {
		t.Fatalf("自有码精确应只返回那一条: %+v err=%v", rh, err)
	}
}

// 同义说法只经向量路召回：查询「挖机」，库里只有「单斗挖掘机」（LIKE 对不上）。
func TestSearch_SemanticRecallsSynonyms(t *testing.T) {
	s, gdb, cleanup := searchStore(t)
	defer cleanup()
	saveSearchFixtures(t, s)
	emb := &bagEmbedder{buckets: searchBuckets}
	h := NewSearcher(s, buildIndex(t, gdb, s, emb), func() SearchEmbedder { return emb }, nil)

	hits, err := h.Resources(context.Background(), SearchQuery{Text: "挖机"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || !strings.HasPrefix(hits[0].Item.Title, "单斗挖掘机") || hits[0].Via != ViaSemantic {
		t.Fatalf("「挖机」应经向量路召回「单斗挖掘机」: %+v", hits)
	}
	// 关键词命中与语义命中并存时：名称含查询词的在前，只有向量路命中的在后并标「语义」
	hits, err = h.Resources(context.Background(), SearchQuery{Text: "水泥"})
	if err != nil {
		t.Fatal(err)
	}
	got := hitTitles(hits, func(r Resource) string { return r.Title })
	if len(got) != 2 || got[0] != "水泥砂浆 砌筑" || got[1] != "混凝土外加剂" {
		t.Fatalf("关键词命中应排在语义命中前: %v", got)
	}
	if hits[0].Via != ViaKeyword || hits[1].Via != ViaSemantic || hits[0].Score <= hits[1].Score {
		t.Fatalf("命中方式 / 分值不对: %+v", hits)
	}
	// 定额同样走通：查「砼」→ 没有同桶的定额 → 空；查向量能对上的「挖机」→ 召回「挖掘机挖土」
	qh, err := h.Quotas(context.Background(), SearchQuery{Text: "挖机"})
	if err != nil {
		t.Fatal(err)
	}
	if len(qh) != 1 || qh[0].Item.Code != "SZ-TF-001" || qh[0].Via != ViaSemantic {
		t.Fatalf("定额语义召回不对: %+v", qh)
	}
	// 专业过滤套在向量路上：只看「建筑工程」时，市政的定额不出现
	qh, _ = h.Quotas(context.Background(), SearchQuery{Text: "挖机", Specialty: "建筑工程"})
	if len(qh) != 0 {
		t.Fatalf("专业过滤没套到向量路: %+v", qh)
	}
}

// 候选超过 8 条时请精排：精排点名的排到最前，其余保持融合序。
func TestSearch_RerankReordersWhenManyCandidates(t *testing.T) {
	s, _, cleanup := searchStore(t)
	defer cleanup()
	for i := 0; i < 9; i++ {
		title := "土方工序 " + string(rune('A'+i))
		if i == 4 {
			title = "履带式液压挖掘机 土方"
		}
		if _, err := s.SaveResource(Resource{Kind: KindMachine, Title: title, Unit: "台班"}); err != nil {
			t.Fatal(err)
		}
	}
	h := NewSearcher(s, nil, nil, func() SearchReranker { return pickReranker{marker: "履带式"} })
	hits, err := h.Resources(context.Background(), SearchQuery{Text: "土方"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 9 || !strings.HasPrefix(hits[0].Item.Title, "履带式") {
		t.Fatalf("精排应把「履带式」排到最前: %v", hitTitles(hits, func(r Resource) string { return r.Title }))
	}
	// 候选不足 9 条时不请精排：去掉两条后融合序不变
	if err := s.DeleteResource(hits[1].Item.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteResource(hits[2].Item.ID, true); err != nil {
		t.Fatal(err)
	}
	h2 := NewSearcher(s, nil, nil, func() SearchReranker { return nil })
	hits, err = h2.Resources(context.Background(), SearchQuery{Text: "土方"})
	if err != nil || len(hits) != 7 {
		t.Fatalf("7 条: %v %v", len(hits), err)
	}
	// 前一步删掉的是精排后排在第 2、3 位的「A」「B」，剩下按编码序从 C 起
	if !strings.HasPrefix(hits[0].Item.Title, "土方工序 C") {
		t.Fatalf("候选不足时不该被精排打乱: %v", hitTitles(hits, func(r Resource) string { return r.Title }))
	}
}

// 向量路不可用（模型没开 / 索引没建 / 精排报错）一律让路，关键词路照常，不报错。
func TestSearch_VectorPathYieldsWhenUnavailable(t *testing.T) {
	s, gdb, cleanup := searchStore(t)
	defer cleanup()
	saveSearchFixtures(t, s)
	emb := &bagEmbedder{buckets: searchBuckets}
	// 模型没开：embed 返回 nil
	h := NewSearcher(s, vecindex.Open(gdb), func() SearchEmbedder { return nil }, nil)
	if hits, err := h.Resources(context.Background(), SearchQuery{Text: "挖掘机"}); err != nil || len(hits) != 1 {
		t.Fatalf("模型没开时关键词路照常: %+v err=%v", hits, err)
	}
	// 索引没建（类目里没有向量）：不报错
	if hits, err := h.Resources(context.Background(), SearchQuery{Text: "挖机"}); err != nil || len(hits) != 0 {
		t.Fatalf("索引没建时应返回空: %+v err=%v", hits, err)
	}
	// 索引建好但查询向量维度对不上：静默让路
	vs := buildIndex(t, gdb, s, emb)
	h = NewSearcher(s, vs, func() SearchEmbedder {
		return &bagEmbedder{buckets: append(append([][]string{}, searchBuckets...), []string{"降维凑数"})}
	}, nil)
	if hits, err := h.Resources(context.Background(), SearchQuery{Text: "水泥"}); err != nil || len(hits) != 1 {
		t.Fatalf("维度对不上时关键词路照常: %+v err=%v", hits, err)
	}
	// 精排出错：按融合分返回
	h = NewSearcher(s, vs, func() SearchEmbedder { return emb }, func() SearchReranker { return errReranker{} })
	if hits, err := h.Resources(context.Background(), SearchQuery{Text: "水泥"}); err != nil || len(hits) != 2 {
		t.Fatalf("精排出错应按融合分返回: %+v err=%v", hits, err)
	}
}

type errReranker struct{}

func (errReranker) Rerank(context.Context, string, []string, int) ([]int, error) {
	return nil, context.DeadlineExceeded
}

// 清单检索：跨项目；project_id / specialty 过滤对关键词路与向量路都生效。
func TestSearch_BillItemsAcrossProjects(t *testing.T) {
	s, gdb, cleanup := searchStore(t)
	defer cleanup()
	pa, err := s.UpsertBillProject("百锦路道路", "a.xlsx", "/tmp/a.xlsx", "", "", "综合单价只含人材机", nil, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := s.UpsertBillProject("厂房地坪", "b.xlsx", "/tmp/b.xlsx", "", "", "", nil, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range []BillItem{
		{ProjectID: pa, Code: "040101001", Title: "挖路槽土方", Unit: "m3", Specialty: "市政"},
		{ProjectID: pa, Code: "040202008001", Title: "沥青混凝土面层", Unit: "m2", Specialty: "市政"},
		{ProjectID: pb, Code: "010101001", Title: "挖基础土方", Unit: "m3", Specialty: "建筑工程"},
	} {
		if _, err := s.SaveBillItem(it); err != nil {
			t.Fatalf("存清单 %s: %v", it.Title, err)
		}
	}
	emb := &bagEmbedder{buckets: searchBuckets}
	h := NewSearcher(s, buildIndex(t, gdb, s, emb), func() SearchEmbedder { return emb }, nil)

	// 跨项目关键词：两个项目的「土方」都在
	hits, err := h.BillItems(context.Background(), SearchQuery{Text: "土方"})
	if err != nil || len(hits) != 2 {
		t.Fatalf("跨项目应命中 2 条: %+v err=%v", hits, err)
	}
	// 只看一个项目
	hits, err = h.BillItems(context.Background(), SearchQuery{Text: "土方", ProjectID: pb})
	if err != nil || len(hits) != 1 || hits[0].Item.Code != "010101001" {
		t.Fatalf("项目过滤: %+v err=%v", hits, err)
	}
	// 向量路 + 专业过滤：查「砼」→ 沥青混凝土面层（市政）；限定建筑工程时为空
	hits, err = h.BillItems(context.Background(), SearchQuery{Text: "砼"})
	if err != nil || len(hits) != 1 || hits[0].Item.Code != "040202008001" || hits[0].Via != ViaSemantic {
		t.Fatalf("清单语义召回: %+v err=%v", hits, err)
	}
	if hits, _ = h.BillItems(context.Background(), SearchQuery{Text: "砼", Specialty: "建筑工程"}); len(hits) != 0 {
		t.Fatalf("专业过滤没套到向量路: %+v", hits)
	}
	// 空查询词：跨项目按列表
	if hits, err = h.BillItems(context.Background(), SearchQuery{}); err != nil || len(hits) != 3 {
		t.Fatalf("跨项目列表: %+v err=%v", hits, err)
	}
	// 挂接定额照常带出（与 BillItems 同口径）：非 nil 即可
	for _, x := range hits {
		if x.Item.QuotaCodes == nil {
			t.Fatalf("QuotaCodes 应为非 nil 切片: %+v", x.Item)
		}
	}
}
