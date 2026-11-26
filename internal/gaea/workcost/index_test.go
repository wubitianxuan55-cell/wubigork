package workcost

// index_test.go — 造价库语义索引接入面：三类文档正文、全量文档源、写入打脏标记。

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/vecindex"
)

// indexEmbedder 测试用的确定性向量化：记录被嵌入的正文（验证「只嵌该嵌的」）。
type indexEmbedder struct {
	mu    sync.Mutex
	texts []string
}

func (e *indexEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.mu.Lock()
	e.texts = append(e.texts, texts...)
	e.mu.Unlock()
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{1, 0, 0, 1}
	}
	return out, nil
}

func (e *indexEmbedder) reset() {
	e.mu.Lock()
	e.texts = nil
	e.mu.Unlock()
}

func (e *indexEmbedder) snapshot() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.texts...)
}

func waitIndex(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待 3 秒内索引条件未满足")
}

// 文档正文只放「是什么」：名称 / 编码 / 规格等进；空字段整段不要，不留「单位」这类孤词。
func TestIndexDocs_KeepIdentityDropEmpty(t *testing.T) {
	if got, want := ResourceIndexDoc("A1-12", "R0001", "中砂", "中粗砂", "材料", "m³", "材料/土建材料/砂石"),
		"中砂 中粗砂 材料 A1-12 R0001 单位m³ 材料/土建材料/砂石"; got != want {
		t.Fatalf("工料机正文=%q, want %q", got, want)
	}
	if got, want := ResourceIndexDoc("", "", "中砂", "", "材料", "", ""), "中砂 材料"; got != want {
		t.Fatalf("空字段应整段丢弃: %q, want %q", got, want)
	}
	if got, want := QuotaIndexDoc("SZ-DL-001", "", "沥青混凝土面层", "市政", "道路工程", "t", "", "综合工日、沥青摊铺机"),
		"沥青混凝土面层 SZ-DL-001 专业市政 章节道路工程 单位t 组成：综合工日、沥青摊铺机"; got != want {
		t.Fatalf("定额正文=%q, want %q", got, want)
	}
	if got := QuotaIndexDoc("SZ-DL-002", "", "拆除旧路面", "", "", "m2", "", ""); strings.Contains(got, "组成") {
		t.Fatalf("没有含量行就不该出现「组成」: %q", got)
	}
	if got, want := BillIndexDoc("010101001", "Q0001", "挖基础土方", "深度 2m 以内", "m3", "测试项目", "", ""),
		"挖基础土方 010101001 Q0001 单位m3 项目特征：深度 2m 以内 项目：测试项目"; got != want {
		t.Fatalf("清单正文=%q, want %q", got, want)
	}
}

// 一条写语句涉及的索引类目：按表名映射，读语句与无关表不算。
func TestDirtyKinds(t *testing.T) {
	cases := []struct {
		query string
		want  []string
	}{
		{"INSERT INTO gf_resources(code, title) VALUES(?, ?)", []string{IndexKindResource}},
		{"insert or replace into gf_projects(name) values(?)", []string{IndexKindBill}},
		{"UPDATE gf_quota_items SET quantity=? WHERE quota_code=?", []string{IndexKindQuota}},
		{"DELETE FROM gf_bill_items WHERE project_id=?", []string{IndexKindBill}},
		{"UPDATE gf_quotas SET gcode=? WHERE id=?", []string{IndexKindQuota}},
		{"SELECT code FROM gf_resources WHERE title LIKE ?", nil},
		{"INSERT INTO gf_resource_prices(resource_id, price) VALUES(?, ?)", nil},
		{"DELETE FROM gf_bill_quota_links WHERE bill_item_id=?", nil},

		{"UPDATE OR IGNORE gf_resources SET title=? WHERE id=?", []string{IndexKindResource}},
		{"DELETE FROM gf_quota_items WHERE quota_code=?; INSERT INTO gf_quota_items(quota_code) VALUES(?);", []string{IndexKindQuota}},
		{"UPDATE cost_entries SET price=0", nil}}
	for _, c := range cases {
		got := dirtyKinds(c.query)
		if strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("dirtyKinds(%q) = %v, want %v", c.query, got, c.want)
		}
	}
}

// 全量文档源：三类都按库里的键吐出全部条目；正文含识别信息、不含会变的数；删掉的不再出现。
func TestIndexSources_StreamAllKinds(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "市政")

	labor, err := s.SaveResource(Resource{Kind: KindLabor, Title: "综合工日", Unit: "工日", BasePrice: 250, CurrentPrice: 265})
	if err != nil {
		t.Fatal(err)
	}
	asphalt, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "细粒式沥青混凝土 AC-13", Spec: "厚5cm", Unit: "t",
		BasePrice: 4000, CurrentPrice: 4200})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveQuota(Quota{
		Code: "SZ-DL-001", Title: "细粒式沥青混凝土面层", Specialty: "市政", Chapter: "道路工程", Unit: "t",
		Items: []QuotaItem{
			{ResourceCode: labor.Code, Quantity: 0.048},
			{ResourceCode: asphalt.Code, Quantity: 1.0},
		},
	}, nil); err != nil {
		t.Fatal(err)
	}
	projID, err := s.UpsertBillProject("百锦路道路工程", "test.xlsx", "/tmp/test.xlsx", "", "", "", nil, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	item, err := s.SaveBillItem(BillItem{ProjectID: projID, Code: "040202008001", Title: "沥青混凝土面层", Unit: "m2",
		Feature: "厚 5cm，AC-13", Quantity: 1250})
	if err != nil {
		t.Fatal(err)
	}
	gone, err := s.SaveResource(Resource{Kind: KindMachine, Title: "轮胎式压路机", Unit: "台班"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteResource(gone.ID, true); err != nil {
		t.Fatal(err)
	}

	docs := map[string]map[string]string{}
	for kind, src := range s.IndexSources() {
		docs[kind] = map[string]string{}
		if err := src(func(batch []vecindex.Doc) error {
			for _, d := range batch {
				docs[kind][d.ID] = d.Text
			}
			return nil
		}); err != nil {
			t.Fatalf("%s 文档源: %v", kind, err)
		}
	}

	res := docs[IndexKindResource]
	if len(res) != 2 {
		t.Fatalf("工料机应剩 2 条（已删的不出现），实际 %d: %v", len(res), res)
	}
	if got := res[asphalt.Code]; !strings.Contains(got, "细粒式沥青混凝土 AC-13") || !strings.Contains(got, "厚5cm") ||
		!strings.Contains(got, "材料") {
		t.Fatalf("工料机正文缺识别信息: %q", got)
	}
	for _, n := range []string{"4000", "4200", "250", "265"} {
		if strings.Contains(res[asphalt.Code], n) || strings.Contains(res[labor.Code], n) {
			t.Fatalf("工料机正文不该含单价: %v", res)
		}
	}

	q := docs[IndexKindQuota]
	if len(q) != 1 {
		t.Fatalf("定额应 1 条，实际 %d: %v", len(q), q)
	}
	if got := q["SZ-DL-001"]; !strings.Contains(got, "细粒式沥青混凝土面层") || !strings.Contains(got, "专业市政") ||
		!strings.Contains(got, "章节道路工程") || !strings.Contains(got, "组成：综合工日、细粒式沥青混凝土 AC-13") {
		t.Fatalf("定额正文缺字段或含量行顺序不对: %q", got)
	}

	b := docs[IndexKindBill]
	if len(b) != 1 {
		t.Fatalf("清单应 1 条，实际 %d: %v", len(b), b)
	}
	got := b[strconv.FormatInt(item.ID, 10)]
	if got == "" {
		t.Fatalf("清单行的键应是行 ID %d: %v", item.ID, b)
	}
	if !strings.Contains(got, "沥青混凝土面层") || !strings.Contains(got, "040202008001") ||
		!strings.Contains(got, "项目特征：厚 5cm，AC-13") || !strings.Contains(got, "项目：百锦路道路工程") {
		t.Fatalf("清单正文缺字段: %q", got)
	}
	if strings.Contains(got, "1250") {
		t.Fatalf("清单正文不该含工程量: %q", got)
	}
}

// 写入打脏标记走通：经 Store 的写路径触发后台索引器的增量对账；只嵌新条目；
// 调价不在正文里，不重嵌；删除后向量被清掉。
func TestIndexDirtyMarking_DriveIncrementalReconcile(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase 返回 nil")
	}
	t.Cleanup(func() {
		vecindex.Forget(gdb)
		db.CloseDatabase(dir)
	})
	s := Open(gdb)
	seedDict(t, s, DictSpecialty, "市政")
	seedDict(t, s, DictRegion, "成都")
	emb := &indexEmbedder{}
	ix := vecindex.NewIndexer(vecindex.Open(gdb), "bge-m3", func() vecindex.Embedder { return emb },
		s.IndexSources(), IndexKinds...)
	ix.Debounce, ix.ReconcileEvery, ix.RetryAfter = 20*time.Millisecond, 0, 20*time.Millisecond
	vecindex.SetDefault(ix)
	t.Cleanup(func() { vecindex.SetDefault(nil) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); ix.Run(ctx) }()
	defer func() { cancel(); <-done }()

	allLast := func() bool {
		for _, k := range IndexKinds {
			if _, ok := ix.Last(k); !ok {
				return false
			}
		}
		return true
	}
	waitIndex(t, allLast) // 启动先对一轮（空库）

	// 新增工料机 → wc_quota / wc_bill 不动，只嵌这一条
	emb.reset()
	sand, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "中砂", Spec: "中粗砂", Unit: "m³",
		BasePrice: 110, CurrentPrice: 110})
	if err != nil {
		t.Fatal(err)
	}
	waitIndex(t, func() bool { return len(emb.snapshot()) > 0 })
	if got := emb.snapshot(); len(got) != 1 || !strings.HasPrefix(got[0], "中砂") {
		t.Fatalf("新增一条工料机应只嵌这一条，实际 %v", got)
	}

	// 调价：写的是价格表 + 工料机表的现价列，正文没变 → 对账跑了但不重嵌
	emb.reset()
	if _, err := s.UpdateResourcePrice(sand.ID, 130, "2026-11", "成都", "信息价", "测试", ""); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	if got := emb.snapshot(); len(got) != 0 {
		t.Fatalf("调价不该重嵌，实际嵌了 %v", got)
	}

	// 定额（含量行落在 gf_quota_items）→ 只嵌这一条定额
	emb.reset()
	if _, err := s.SaveQuota(Quota{Code: "SZ-DL-001", Title: "浆砌片石", Specialty: "市政", Chapter: "防护工程", Unit: "m3",
		Items: []QuotaItem{{ResourceCode: sand.Code, Quantity: 1.2}}}, nil); err != nil {
		t.Fatal(err)
	}
	waitIndex(t, func() bool { return len(emb.snapshot()) > 0 })
	if got := emb.snapshot(); len(got) != 1 || !strings.HasPrefix(got[0], "浆砌片石") {
		t.Fatalf("新增一条定额应只嵌这一条，实际 %v", got)
	}

	// 删除工料机 → 它的向量被清掉
	emb.reset()
	if err := s.DeleteResource(sand.ID, true); err != nil {
		t.Fatal(err)
	}
	waitIndex(t, func() bool {
		m, _ := vecindex.Open(gdb).Meta(IndexKindResource)
		return m.Count == 0
	})
	if got := emb.snapshot(); len(got) != 0 {
		t.Fatalf("删除不该触发嵌入，实际 %v", got)
	}
}
