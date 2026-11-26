package vecindex

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
)

// charEmbedder 测试用的确定性向量化：按字符袋哈希成 8 维向量——字面相近的文本向量
// 相近，够用来验证「谁该被召回」。
type charEmbedder struct {
	dim     int
	calls   int
	texts   []string
	failOn  func(texts []string) error
	mu      sync.Mutex
	latency time.Duration
}

func (e *charEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.mu.Lock()
	e.calls++
	e.texts = append(e.texts, texts...)
	fail := e.failOn
	e.mu.Unlock()
	if e.latency > 0 {
		time.Sleep(e.latency)
	}
	if fail != nil {
		if err := fail(texts); err != nil {
			return nil, err
		}
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, e.dim)
		for _, r := range strings.ToLower(t) {
			if r == ' ' {
				continue
			}
			v[int(r)%e.dim] += 1
		}
		out[i] = v
	}
	return out, nil
}

func (e *charEmbedder) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func testDB(t *testing.T) *dbHandle {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() {
		Forget(gdb)
		_ = db.CloseDatabase(dir)
	})
	return &dbHandle{gdb: gdb, st: Open(gdb)}
}

type dbHandle struct {
	gdb *sql.DB
	st  *Store
}

func staticSource(docs ...Doc) Source {
	return func(yield func([]Doc) error) error { return yield(docs) }
}

func chunkedSource(chunks [][]Doc) Source {
	return func(yield func([]Doc) error) error {
		for _, c := range chunks {
			if err := yield(c); err != nil {
				return err
			}
		}
		return nil
	}
}

func ids(hits []Hit) []string {
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.ID
	}
	return out
}

// 首次对账全量向量化；再来一次什么都不做；改一条正文只重嵌那一条；删掉一条清掉它的向量。
func TestApply_Incremental(t *testing.T) {
	h := testDB(t)
	emb := &charEmbedder{dim: 8}
	src := staticSource(
		Doc{"q1", "单斗挖掘机 挖土"},
		Doc{"q2", "自卸汽车 运土"},
		Doc{"q3", "人工开挖 基坑"},
	)

	st, err := h.st.Apply(context.Background(), "wc_quota", "bge-m3", src, emb, 0)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if st.Total != 3 || st.Embedded != 3 || st.Removed != 0 || st.Count != 3 {
		t.Fatalf("首次对账=%+v，want Total 3 / Embedded 3 / Removed 0 / Count 3", st)
	}
	meta, _ := h.st.Meta("wc_quota")
	if meta.Count != 3 || meta.Dim != 8 || meta.Model != "bge-m3" || meta.Version != 1 {
		t.Fatalf("元信息=%+v", meta)
	}

	st, err = h.st.Apply(context.Background(), "wc_quota", "bge-m3", src, emb, 0)
	if err != nil || st.Embedded != 0 || st.Removed != 0 || st.Count != 3 {
		t.Fatalf("二次对账应无事可做：%+v err=%v", st, err)
	}
	if emb.count() != 1 {
		t.Fatalf("二次对账不应再调向量化，calls=%d", emb.count())
	}

	// 改一条正文 + 删一条
	src = staticSource(Doc{"q1", "单斗挖掘机 挖土 装车"}, Doc{"q3", "人工开挖 基坑"})
	st, err = h.st.Apply(context.Background(), "wc_quota", "bge-m3", src, emb, 0)
	if err != nil || st.Embedded != 1 || st.Removed != 1 || st.Total != 2 || st.Count != 2 {
		t.Fatalf("改一条删一条：%+v err=%v", st, err)
	}
	if last := emb.texts[len(emb.texts)-1]; last != "单斗挖掘机 挖土 装车" {
		t.Fatalf("只应重嵌改动的那条，实际嵌了 %q", last)
	}
	if hits, _ := h.st.SearchVec("wc_quota", mustVec(t, emb, "汽车"), 5, nil); strings.Contains(strings.Join(ids(hits), ","), "q2") {
		t.Fatalf("删掉的条目不应再被搜到：%+v", hits)
	}

	// 换了模型：整类目重嵌
	src = staticSource(Doc{"q1", "单斗挖掘机 挖土 装车"}, Doc{"q3", "人工开挖 基坑"})
	st, err = h.st.Apply(context.Background(), "wc_quota", "qwen3-embedding-4b", src, emb, 0)
	if err != nil || st.Embedded != 2 {
		t.Fatalf("换模型应整类目重嵌：%+v err=%v", st, err)
	}
	if meta, _ = h.st.Meta("wc_quota"); meta.Model != "qwen3-embedding-4b" {
		t.Fatalf("元信息应换成新模型：%+v", meta)
	}
}

// 空正文与空 ID 跳过（不嵌、也不算总数）；正文被清空的条目算过期，向量清掉。
func TestApply_SkipsEmptyAndClearsEmptied(t *testing.T) {
	h := testDB(t)
	emb := &charEmbedder{dim: 8}
	_, err := h.st.Apply(context.Background(), "k", "bge-m3", staticSource(Doc{"a", "基础"}, Doc{"", "没 ID"}, Doc{"b", "  "}), emb, 0)
	if err != nil {
		t.Fatal(err)
	}
	st, err := h.st.Apply(context.Background(), "k", "bge-m3", staticSource(Doc{"a", ""}), emb, 0)
	if err != nil || st.Total != 0 || st.Removed != 1 || st.Count != 0 {
		t.Fatalf("正文清空应清向量：%+v err=%v", st, err)
	}
	if hits, _ := h.st.SearchVec("k", mustVec(t, emb, "基础"), 5, nil); hits != nil {
		t.Fatalf("空索引不应有命中：%+v", hits)
	}
}

// 对账中途向量化失败：已写入的留在库里，错误如实返回；下一次续上。
func TestApply_PartialFailureKeepsProgress(t *testing.T) {
	h := testDB(t)
	emb := &charEmbedder{dim: 8}
	fail := true
	emb.failOn = func(texts []string) error {
		if fail && len(texts) > 0 && strings.Contains(texts[0], "第2批") {
			return errors.New("模型掉线")
		}
		return nil
	}
	var docs []Doc
	for i := 0; i < 6; i++ {
		docs = append(docs, Doc{fmt.Sprintf("d%d", i), fmt.Sprintf("第%d批 文本%d", i/3+1, i)})
	}
	src := chunkedSource([][]Doc{docs[:3], docs[3:]})
	_, err := h.st.Apply(context.Background(), "k", "bge-m3", src, emb, 3)
	if err == nil || !strings.Contains(err.Error(), "已写入 3 条") {
		t.Fatalf("第二批失败应报错并带上已写入条数：%v", err)
	}
	if meta, _ := h.st.Meta("k"); meta.Count != 3 {
		t.Fatalf("第一批应已落库：meta=%+v", meta)
	}
	fail = false
	st, err := h.st.Apply(context.Background(), "k", "bge-m3", staticSource(docs...), emb, 3)
	if err != nil || st.Embedded != 3 || st.Count != 6 {
		t.Fatalf("续做应只嵌剩下的 3 条：%+v err=%v", st, err)
	}
}

// 检索：字面相近的排前面；keep 过滤；topN 截断；余弦下限。
func TestSearch_RanksFiltersAndLimits(t *testing.T) {
	h := testDB(t)
	emb := &charEmbedder{dim: 16}
	docs := []Doc{
		{"q1", "单斗挖掘机 挖装土方"},
		{"q2", "推土机 推土"},
		{"q3", "自卸汽车 运输土方"},
		{"q4", "水泥砂浆 砌筑"},
		{"q5", "钢筋制作 安装"},
	}
	if _, err := h.st.Apply(context.Background(), "wc_quota", "bge-m3", staticSource(docs...), emb, 0); err != nil {
		t.Fatal(err)
	}
	q := mustVec(t, emb, "挖掘机")
	hits, err := h.st.SearchVec("wc_quota", q, 5, nil)
	if err != nil || len(hits) == 0 || hits[0].ID != "q1" {
		t.Fatalf("最相近的应是 q1：%+v err=%v", hits, err)
	}
	for i := 1; i < len(hits); i++ {
		if hits[i-1].Score < hits[i].Score {
			t.Fatalf("应按得分降序：%+v", hits)
		}
		if hits[i].Score < MinCosine {
			t.Fatalf("低于下限的应丢掉：%+v", hits)
		}
	}
	if got, _ := h.st.SearchVec("wc_quota", q, 2, nil); len(got) > 2 {
		t.Fatalf("topN 应截断：%+v", got)
	}
	// keep：只留专业对得上的（这里用 ID 奇偶模拟「专业」过滤）
	kept, _ := h.st.SearchVec("wc_quota", q, 5, func(id string) bool { return strings.HasSuffix(id, "2") || strings.HasSuffix(id, "4") })
	for _, k := range kept {
		if k.ID != "q2" && k.ID != "q4" {
			t.Fatalf("keep 过滤没生效：%+v", kept)
		}
	}
	// 多次检索只向量化 query 一次由调用方保证；索引器这边缓存命中后不再读库
	before := emb.count()
	for i := 0; i < 5; i++ {
		_, _ = h.st.SearchVec("wc_quota", q, 5, nil)
	}
	if emb.count() != before {
		t.Fatal("SearchVec 不应调向量化")
	}
	// 维度不一致如实报错
	_, err = h.st.SearchVec("wc_quota", make([]float32, 7), 5, nil)
	if !errors.Is(err, ErrDimMismatch) {
		t.Fatalf("维度不一致应报 ErrDimMismatch：%v", err)
	}
}

// 缓存失效：对账写入后版本号变，下一次检索看到新向量，不用重启也不用手工清缓存。
func TestSearch_SeesIndexUpdates(t *testing.T) {
	h := testDB(t)
	emb := &charEmbedder{dim: 16}
	src := staticSource(Doc{"a", "打桩机 施工"})
	_, _ = h.st.Apply(context.Background(), "k", "bge-m3", src, emb, 0)
	q := mustVec(t, emb, "桩机 施工")
	if hits, _ := h.st.SearchVec("k", q, 5, nil); len(hits) != 1 {
		t.Fatalf("先应只有 a：%+v", hits)
	}
	_, _ = h.st.Apply(context.Background(), "k", "bge-m3", staticSource(Doc{"a", "打桩机 施工"}, Doc{"b", "桩机 施工"}), emb, 0)
	hits, _ := h.st.SearchVec("k", q, 5, nil)
	if len(hits) != 2 || hits[0].ID != "b" {
		t.Fatalf("写入后应立刻搜到新条目并排最前：%+v", hits)
	}
}

// Open 对同一个 *sql.DB 只给一个实例（内存缓存共享）；nil / 关闭的库按不可用处理。
func TestOpen_SharesStorePerDB(t *testing.T) {
	h := testDB(t)
	if Open(h.gdb) != h.st {
		t.Fatal("同一个库应拿到同一个实例")
	}
	if Open(nil) != nil {
		t.Fatal("nil 库应返回 nil")
	}
	var nilStore *Store
	if nilStore.Available() {
		t.Fatal("nil 实例应按不可用处理")
	}
	if meta, _ := nilStore.Meta("k"); meta.Count != 0 {
		t.Fatalf("nil 实例的元信息应为空：%+v", meta)
	}
	if hits, err := nilStore.SearchVec("k", []float32{1}, 5, nil); hits != nil || err != nil {
		t.Fatalf("nil 实例检索应返回空：%+v %v", hits, err)
	}
}

// 不可用：没模型 / 表没建，都如实报错而不是静默成功。
func TestApply_Unavailable(t *testing.T) {
	h := testDB(t)
	if _, err := h.st.Apply(context.Background(), "k", "bge-m3", staticSource(Doc{"a", "x"}), nil, 0); err == nil {
		t.Fatal("没有 embedder 应报错")
	}
	if _, err := Open(nil).Apply(context.Background(), "k", "bge-m3", staticSource(Doc{"a", "x"}), &charEmbedder{dim: 4}, 0); err == nil {
		t.Fatal("没有库应报错")
	}
}

// 同一类目的两次对账串行（不交错写）；不同类目互不阻塞。
func TestApply_SerializedPerKind(t *testing.T) {
	h := testDB(t)
	emb := &charEmbedder{dim: 8, latency: 30 * time.Millisecond}
	var docs []Doc
	for i := 0; i < 20; i++ {
		docs = append(docs, Doc{fmt.Sprintf("d%02d", i), fmt.Sprintf("条目 %d 内容", i)})
	}
	var wg sync.WaitGroup
	results := make([]Stats, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			st, err := h.st.Apply(context.Background(), "k", "bge-m3", staticSource(docs...), emb, 4)
			if err != nil {
				t.Errorf("Apply: %v", err)
			}
			results[i] = st
		}(i)
	}
	wg.Wait()
	for i, st := range results {
		t.Logf("第 %d 次：total=%d embedded=%d removed=%d count=%d dur=%s", i+1, st.Total, st.Embedded, st.Removed, st.Count, st.Dur.Round(time.Millisecond))
	}
	if meta, _ := h.st.Meta("k"); meta.Count != 20 {
		t.Fatalf("并发对账后条数应恰为 20： %+v", meta)
	}
}

func mustVec(t *testing.T, emb Embedder, text string) []float32 {
	t.Helper()
	v, err := Query(context.Background(), emb, text)
	if err != nil || v == nil {
		t.Fatalf("向量化 %q: %v", text, err)
	}
	return v
}

// ── 内存矩阵 ─────────────────────────────────────────────────────────

// int8 量化后的排序与 float64 精确余弦对账：随机向量上 top-10 至少重合 9 条、首条一致。
func TestMatrix_QuantizedRankingMatchesExact(t *testing.T) {
	const n, dim, queries = 3000, 64, 25
	rng := rand.New(rand.NewSource(7))
	m := newMatrix("m", dim, n, 1)
	exact := make([][]float64, n)
	for i := 0; i < n; i++ {
		v := randomVec(rng, dim)
		m.add(fmt.Sprintf("v%04d", i), v)
		exact[i] = toFloat64(v)
	}
	missTop, overlap := 0, 0.0
	for q := 0; q < queries; q++ {
		qv := randomVec(rng, dim)
		got := m.search(qv, 10, nil, 0)
		want := exactTop(exact, qv, 10)
		if got[0].ID != want[0].id {
			missTop++
		}
		set := map[string]bool{}
		for _, h := range got {
			set[h.ID] = true
		}
		k := 0
		for _, w := range want {
			if set[w.id] {
				k++
			}
		}
		overlap += float64(k) / 10
		if k < 9 {
			t.Fatalf("第 %d 个查询：量化后 top-10 只重合 %d 条", q, k)
		}
	}
	if missTop > 1 {
		t.Fatalf("%d 个查询的首条不一致", missTop)
	}
	t.Logf("top-10 平均重合率 %.3f", overlap/queries)
}

// 并行路径（行数过阈值）与串行路径结果一致：把阈值临时调到 0 强制走并行。
func TestMatrix_ParallelEqualsSequential(t *testing.T) {
	old := parallelRows
	parallelRows = 0
	t.Cleanup(func() { parallelRows = old })

	const n, dim = 5000, 32
	rng := rand.New(rand.NewSource(11))
	m := newMatrix("m", dim, n, 1)
	for i := 0; i < n; i++ {
		m.add(fmt.Sprintf("v%04d", i), randomVec(rng, dim))
	}
	qv := randomVec(rng, dim)
	par := m.search(qv, 15, nil, 0)
	sequential := newMatrix("m", dim, n, 1)
	sequential.ids, sequential.q, sequential.coef = m.ids, m.q, m.coef
	parallelRows = 1 << 30
	seq := sequential.search(qv, 15, nil, 0)
	if len(par) != len(seq) {
		t.Fatalf("并行 %d 条 / 串行 %d 条", len(par), len(seq))
	}
	for i := range par {
		if par[i].ID != seq[i].ID || math.Abs(par[i].Score-seq[i].Score) > 1e-9 {
			t.Fatalf("第 %d 条不一致：并行 %+v / 串行 %+v", i, par[i], seq[i])
		}
	}
}

// 性能护栏：20 万条 × 128 维（int8 约 25MB）单次检索应在百毫秒量级。
func TestMatrix_LargeSearchLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("short 模式跳过大表性能用例")
	}
	const n, dim = 200000, 128
	m := newMatrix("m", dim, n, 1)
	v := make([]float32, dim)
	for i := 0; i < n; i++ {
		for k := range v {
			v[k] = float32((i*31+k*17)%97) / 97 // 确定性、有区分度的伪随机
		}
		m.add(fmt.Sprintf("v%06d", i), v)
	}
	qv := make([]float32, dim)
	for k := range qv {
		qv[k] = float32((k*13)%89) / 89
	}
	m.search(qv, 50, nil, 0) // 预热
	start := time.Now()
	const rounds = 5
	for i := 0; i < rounds; i++ {
		m.search(qv, 50, nil, 0)
	}
	per := time.Since(start) / rounds
	t.Logf("20 万条 × 128 维单次检索 %s", per)
	if per > 500*time.Millisecond {
		t.Fatalf("单次检索 %s，超过 500ms", per)
	}
}

func randomVec(rng *rand.Rand, dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32(rng.NormFloat64())
	}
	return v
}

func toFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	var norm float64
	for i, x := range v {
		out[i] = float64(x)
		norm += float64(x) * float64(x)
	}
	norm = math.Sqrt(norm)
	for i := range out {
		out[i] /= norm
	}
	return out
}

type exactHit struct {
	id    string
	score float64
}

func exactTop(rows [][]float64, qv []float32, topN int) []exactHit {
	q := toFloat64(qv)
	all := make([]exactHit, len(rows))
	for i, r := range rows {
		var s float64
		for k := range r {
			s += r[k] * q[k]
		}
		all[i] = exactHit{fmt.Sprintf("v%04d", i), s}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].score > all[j].score })
	return all[:topN]
}

// ── 融合排序 ─────────────────────────────────────────────────────────

func TestOrder(t *testing.T) {
	in := []Cand{
		{ID: "vec-only-strong", Vec: 0.82},
		{ID: "title-hit", Title: true},
		{ID: "field-hit", Field: true},
		{ID: "both", Title: true, Vec: 0.7},
		{ID: "vec-only-weak", Vec: 0.5},
		{ID: "exact-code", Exact: true},
	}
	got := ids2(Order(in))
	want := []string{"exact-code", "both", "vec-only-strong", "title-hit", "vec-only-weak", "field-hit"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("顺序=%v，want %v", got, want)
	}
	// 同分：按 ID 字典序，结果确定
	tie := Order([]Cand{{ID: "b", Title: true}, {ID: "a", Title: true}})
	if tie[0].ID != "a" {
		t.Fatalf("同分应按 ID 排序：%+v", tie)
	}
	if !(Cand{ID: "x", Field: true}).Keyword() || (Cand{ID: "y", Vec: 0.9}).Keyword() {
		t.Fatal("Keyword 只应认关键词路的命中")
	}
}

func ids2(c []Cand) []string {
	out := make([]string, len(c))
	for i, x := range c {
		out[i] = x.ID
	}
	return out
}

// ── 后台索引器 ───────────────────────────────────────────────────────

// 脏标记合并：连打多次只触发一轮；启动先对一轮；对账后 Last 有数。
func TestIndexer_MarksCoalesceAndRun(t *testing.T) {
	h := testDB(t)
	emb := &charEmbedder{dim: 8}
	docs := []Doc{{"a", "第一条"}, {"b", "第二条"}}
	ix := NewIndexer(h.st, "", func() Embedder { return emb }, map[string]Source{"k": staticSource(docs...)}, "k")
	ix.Debounce, ix.ReconcileEvery = 40*time.Millisecond, 0
	if ix.model != "bge-m3" {
		t.Fatalf("模型名缺省应为 bge-m3，得到 %q", ix.model)
	}
	SetDefault(ix)
	t.Cleanup(func() { SetDefault(nil) })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); ix.Run(ctx) }()
	for i := 0; i < 50; i++ {
		MarkDirty("k")
	}
	waitFor(t, 3*time.Second, func() bool { _, ok := ix.Last("k"); return ok })
	cancel()
	<-done

	if meta, _ := h.st.Meta("k"); meta.Count != 2 {
		t.Fatalf("索引器应已建好索引：%+v", meta)
	}
	if st, _ := ix.Last("k"); st.Embedded != 2 {
		t.Fatalf("Last=%+v", st)
	}
}

// 模型不可用：待办保留、不报错；模型回来后自动补上。
func TestIndexer_RetriesWhenModelUnavailable(t *testing.T) {
	h := testDB(t)
	emb := &charEmbedder{dim: 8}
	var available atomic.Bool
	ix := NewIndexer(h.st, "bge-m3", func() Embedder {
		if !available.Load() {
			return nil
		}
		return emb
	}, map[string]Source{"k": staticSource(Doc{"a", "内容"})}, "k")
	ix.Debounce, ix.RetryAfter, ix.ReconcileEvery = 20*time.Millisecond, 30*time.Millisecond, 0
	if _, err := ix.ApplyNow(context.Background(), "k"); err == nil {
		t.Fatal("模型不可用时 ApplyNow 应报错")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); ix.Run(ctx) }()
	time.Sleep(80 * time.Millisecond)
	if meta, _ := h.st.Meta("k"); meta.Count != 0 {
		t.Fatalf("模型不可用时不应建索引：%+v", meta)
	}
	available.Store(true)
	waitFor(t, 3*time.Second, func() bool { meta, _ := h.st.Meta("k"); return meta.Count == 1 })
	cancel()
	<-done
}

// 兜底定时对账：漏打脏标记的改动，到点也会被追上。
func TestIndexer_ReconcileCatchesMissedMarks(t *testing.T) {
	h := testDB(t)
	emb := &charEmbedder{dim: 8}
	var mu sync.Mutex
	cur := []Doc{{"a", "内容一"}}
	ix := NewIndexer(h.st, "bge-m3", func() Embedder { return emb }, map[string]Source{
		"k": func(yield func([]Doc) error) error {
			mu.Lock()
			d := append([]Doc(nil), cur...)
			mu.Unlock()
			return yield(d)
		},
	}, "k")
	ix.Debounce, ix.ReconcileEvery = 20*time.Millisecond, 60*time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); ix.Run(ctx) }()
	waitFor(t, 3*time.Second, func() bool { meta, _ := h.st.Meta("k"); return meta.Count == 1 })
	mu.Lock()
	cur = []Doc{{"a", "内容一"}, {"b", "内容二"}} // 没打脏标记
	mu.Unlock()
	waitFor(t, 3*time.Second, func() bool { meta, _ := h.st.Meta("k"); return meta.Count == 2 })
	cancel()
	<-done
}

// 没有登记默认索引器时 MarkDirty 什么都不做；Mark 空参 = 全部类目。
func TestIndexer_MarkAllAndNoDefault(t *testing.T) {
	MarkDirty("k") // 不应 panic
	ix := NewIndexer(nil, "bge-m3", func() Embedder { return nil }, nil, "a", "b")
	ix.Mark()
	if len(ix.pending) != 2 {
		t.Fatalf("Mark 空参应标记全部类目：%v", ix.pending)
	}
	var nilIx *Indexer
	nilIx.Mark("a") // 不应 panic
}

// 最近一轮对账的错误记在案（模型不可用 / 出错），成功后清除；Model / Kinds 取值。
func TestIndexer_RecordsLastError(t *testing.T) {
	h := testDB(t)
	var available atomic.Bool
	emb := &charEmbedder{dim: 8}
	ix := NewIndexer(h.st, "bge-m3", func() Embedder {
		if !available.Load() {
			return nil
		}
		return emb
	}, map[string]Source{"k": staticSource(Doc{"a", "内容"})}, "k")
	if ix.Model() != "bge-m3" || len(ix.Kinds()) != 1 || ix.Kinds()[0] != "k" {
		t.Fatalf("Model / Kinds: %q %v", ix.Model(), ix.Kinds())
	}
	if _, ok := ix.LastError("k"); ok {
		t.Fatal("还没对账过不该有错误记录")
	}
	if _, err := ix.ApplyNow(context.Background(), "k"); err == nil {
		t.Fatal("模型不可用时 ApplyNow 应报错")
	}
	if err, ok := ix.LastError("k"); !ok || err == nil {
		t.Fatalf("模型不可用应记在案: %v %v", err, ok)
	}
	available.Store(true)
	if _, err := ix.ApplyNow(context.Background(), "k"); err != nil {
		t.Fatal(err)
	}
	if _, ok := ix.LastError("k"); ok {
		t.Fatal("成功后应清除错误记录")
	}
	var nilIx *Indexer
	if nilIx.Model() != "" || nilIx.Kinds() != nil {
		t.Fatal("nil 索引器的取值应为零值")
	}
}

func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("等待 %s 内条件未满足", d)
}
