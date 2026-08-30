package costinquiry

import (
	"math"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
)

// v4.6 询价异常检测 + 价格预测：纯函数覆盖。
func TestPredictNextAndLevel(t *testing.T) {
	// 线性上涨序列：100,110,120 → 斜率 10/期 → 下期 130
	next, slope, ok := PredictNext([]float64{100, 110, 120})
	if !ok || math.Abs(next-130) > 0.01 || math.Abs(slope-10) > 0.01 {
		t.Fatalf("PredictNext([100,110,120]) = (%v, %v, %v), want (130, 10, true)", next, slope, ok)
	}
	// 单点 → 无可预测（false）
	if _, _, ok := PredictNext([]float64{50}); ok {
		t.Fatal("单点序列不应可预测")
	}
	// 空序列
	if _, _, ok := PredictNext(nil); ok {
		t.Fatal("空序列不应可预测")
	}
	// 恒定序列：斜率 0，下期=现值
	next, slope, ok = PredictNext([]float64{88, 88, 88})
	if !ok || math.Abs(next-88) > 0.01 || math.Abs(slope) > 0.01 {
		t.Fatalf("PredictNext(恒定) = (%v, %v, %v), want (88, 0, true)", next, slope, ok)
	}

	// 差幅分级：正常(<5)/关注(5-15)/异常(>15)
	cases := []struct {
		pct  float64
		want string
	}{
		{3, "正常"}, {5, "关注"}, {14.9, "关注"}, {15, "关注"}, {15.1, "异常"}, {-30, "异常"},
	}
	for _, c := range cases {
		if got := adjustLevel(c.pct); got != c.want {
			t.Errorf("adjustLevel(%v) = %q, want %q", c.pct, got, c.want)
		}
	}
}

// v4.6 OCR 报价单自动入询价库飞轮：同源同标题幂等更新（重复导入不产生重复点）。
func TestUpsertBySourceKeyDedup(t *testing.T) {
	s := newTestStore(t)
	r := Record{
		Title: "热轧光圆钢筋", Spec: "HPB300 Φ12", Unit: "t", Price: 3750,
		Source: "OCR报价", Supplier: "供应商报价单.pdf", PriceDate: "2026-08",
	}
	id1, err := s.UpsertBySourceKey(r)
	if err != nil {
		t.Fatalf("upsert1: %v", err)
	}
	// 同一报价单再次导入：价格刷新，不新增行
	r.Price = 3780
	id2, err := s.UpsertBySourceKey(r)
	if err != nil {
		t.Fatalf("upsert2: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("重复导入应更新同一数据点: %d vs %d", id1, id2)
	}
	got, err := s.Get(id1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Price != 3780 {
		t.Fatalf("价格未刷新: %v", got.Price)
	}
	// 不同供应商 → 新数据点
	r.Supplier = "另一家.pdf"
	id3, err := s.UpsertBySourceKey(r)
	if err != nil {
		t.Fatalf("upsert3: %v", err)
	}
	if id3 == id1 {
		t.Fatal("不同供应商应生成新数据点")
	}
}

// v4.6 调差建议带异常分级与预测：SuggestAdjustments 输出 Level/PredictedNext。
func TestSuggestAdjustmentsLevelAndPrediction(t *testing.T) {
	s := newTestStore(t)
	mustSave(t, s, Record{Title: "水泥", Spec: "P.O 42.5", Price: 480, PriceDate: "2026-06", Source: "信息价"})
	mustSave(t, s, Record{Title: "水泥", Spec: "P.O 42.5", Price: 520, PriceDate: "2026-07", Source: "信息价"})
	mustSave(t, s, Record{Title: "水泥", Spec: "P.O 42.5", Price: 560, PriceDate: "2026-08", Source: "信息价"})

	// 成本库现价 400 → 最新询价 560，差幅 +40% = 异常；序列 480→520→560 预测
	// 下期 600。
	got := s.SuggestAdjustments([]cost.Summary{{Name: "cement", Title: "水泥", Price: 400, Unit: "t"}})
	if len(got) != 1 {
		t.Fatalf("SuggestAdjustments = %d 条, want 1", len(got))
	}
	if got[0].Level != "异常" {
		t.Fatalf("level = %q, want 异常", got[0].Level)
	}
	if math.Abs(got[0].PredictedNext-600) > 0.01 {
		t.Fatalf("predictedNext = %v, want 600", got[0].PredictedNext)
	}
	if got[0].PredictionNote == "" {
		t.Fatal("预测说明为空")
	}
}

// newTestStore 建临时目录 sqlite 询价库(cost_test.go 同款方式)。
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	return Open(gdb)
}

func mustSave(t *testing.T, s *Store, r Record) int64 {
	t.Helper()
	id, err := s.Save(r)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// TestMatchTitle 标题归一化:全角空格/括号内容(中英文/嵌套)/大小写/空白。
func TestMatchTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{" 水泥 ", "水泥"},                    // 首尾空白
		{"水　泥", "水泥"},                     // 全角空格
		{"水泥(42.5)", "水泥"},                // 英文括号+内容
		{"水泥（P.O 42.5）", "水泥"},            // 中文括号+内容
		{"HP300 高频液压振动锤", "hp300高频液压振动锤"}, // 大小写+空白
		{"水 泥", "水泥"},                     // 半角空格
		{"a(b(c))", "a"},                  // 嵌套括号
		{"水泥（42.5）(P.O)", "水泥"},           // 混合括号
		{"", ""},
	}
	for _, c := range cases {
		if got := MatchTitle(c.in); got != c.want {
			t.Errorf("MatchTitle(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSaveNewAndUpdate 新建默认值/时间戳;更新保留 created_at 刷新 updated_at。
func TestSaveNewAndUpdate(t *testing.T) {
	s := newTestStore(t)
	id := mustSave(t, s, Record{Title: "水泥", Spec: "P.O 42.5", Unit: "吨", Price: 480})
	got, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "水泥" || got.Spec != "P.O 42.5" || got.Unit != "吨" || got.Price != 480 {
		t.Fatalf("roundtrip = %+v", got)
	}
	if got.Source != "手动询价" || got.Status != "现行" {
		t.Errorf("defaults source=%q status=%q, want 手动询价/现行", got.Source, got.Status)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Errorf("timestamps zero: %+v", got)
	}
	if time.Since(got.CreatedAt) > time.Minute {
		t.Errorf("created_at not now: %v", got.CreatedAt)
	}
	// 显式 Source/Status 保留。
	id2 := mustSave(t, s, Record{Title: "钢筋", Price: 4000, Source: "信息价", Status: "已过期"})
	if id2 == id {
		t.Fatal("new record should get distinct id")
	}
	got2, _ := s.Get(id2)
	if got2.Source != "信息价" || got2.Status != "已过期" {
		t.Errorf("explicit source/status = %q/%q", got2.Source, got2.Status)
	}
	// 更新:同 id,created_at 保留,updated_at 刷新。
	createdAt := got.CreatedAt
	if _, err := s.Save(Record{ID: id, Title: "水泥", Spec: "P.O 42.5R", Unit: "吨", Price: 500}); err != nil {
		t.Fatal(err)
	}
	got3, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got3.ID != id || got3.Price != 500 || got3.Spec != "P.O 42.5R" {
		t.Errorf("update not applied: %+v", got3)
	}
	if !got3.CreatedAt.Equal(createdAt) {
		t.Errorf("created_at changed on update: %v -> %v", createdAt, got3.CreatedAt)
	}
	if got3.UpdatedAt.Before(got3.CreatedAt) {
		t.Errorf("updated_at before created_at: %+v", got3)
	}
	// 空标题报错。
	if _, err := s.Save(Record{Title: "  "}); err == nil {
		t.Error("empty title should error")
	}
}

// TestListQueryAndLimit 关键词(标题/规格/供应商/地区/备注,大小写不敏感)与 limit。
func TestListQueryAndLimit(t *testing.T) {
	s := newTestStore(t)
	mustSave(t, s, Record{Title: "水泥", Spec: "P.O 42.5", Supplier: "峨胜", Region: "成都", Note: "袋装", Price: 480})
	mustSave(t, s, Record{Title: "HP300 高频液压振动锤", Spec: "300kW", Supplier: "山河智能", Region: "成都市区", Note: "含燃油", Price: 3200})
	mustSave(t, s, Record{Title: "钢筋", Spec: "HRB400E 25mm", Supplier: "攀钢", Region: "绵阳", Price: 4200})
	mustSave(t, s, Record{Title: "碎石", Spec: "5-31.5mm", Supplier: "宏发料场", Region: "德阳", Note: "机制砂配套", Price: 85})

	if got := s.List("", 0); len(got) != 4 {
		t.Fatalf("List all = %d, want 4", len(got))
	} else if got[0].Title != "碎石" {
		t.Errorf("updated_at DESC first = %q, want 碎石(last saved)", got[0].Title)
	}
	if got := s.List("", 2); len(got) != 2 {
		t.Errorf("limit 2 = %d, want 2", len(got))
	}
	if got := s.List("", -1); len(got) != 4 {
		t.Errorf("limit<=0 default 100 = %d, want 4", len(got))
	}
	if got := s.List("水泥", 0); len(got) != 1 || got[0].Title != "水泥" {
		t.Errorf("title query = %+v", got)
	}
	if got := s.List("300kW", 0); len(got) != 1 || got[0].Title != "HP300 高频液压振动锤" {
		t.Errorf("spec query = %+v", got)
	}
	if got := s.List("攀钢", 0); len(got) != 1 || got[0].Title != "钢筋" {
		t.Errorf("supplier query = %+v", got)
	}
	if got := s.List("德阳", 0); len(got) != 1 || got[0].Title != "碎石" {
		t.Errorf("region query = %+v", got)
	}
	if got := s.List("袋装", 0); len(got) != 1 || got[0].Title != "水泥" {
		t.Errorf("note query = %+v", got)
	}
	if got := s.List("hp300", 0); len(got) != 1 {
		t.Errorf("case-insensitive query = %+v", got)
	}
	if got := s.List("不存在的词", 0); len(got) != 0 {
		t.Errorf("no-match query = %+v", got)
	}
}

// TestGetNotFoundAndDelete Get 不存在报错;Delete 后不可再读。
func TestGetNotFoundAndDelete(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Get(9999); err == nil {
		t.Error("Get missing should error")
	}
	id := mustSave(t, s, Record{Title: "临时", Price: 1})
	if err := s.Delete(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(id); err == nil {
		t.Error("Get after delete should error")
	}
	if got := s.List("", 0); len(got) != 0 {
		t.Errorf("list after delete = %d, want 0", len(got))
	}
}

// TestListExpiring 到期预警边界:today-1/today/today+days 恰在边界、
// 无有效期记录跳过、非法日期跳过、升序(已过期最前)。
func TestListExpiring(t *testing.T) {
	s := newTestStore(t)
	now := time.Now()
	day := func(d int) string { return now.AddDate(0, 0, d).Format("2006-01-02") }
	mustSave(t, s, Record{Title: "过期水泥", Price: 1, ValidUntil: day(-1)})
	mustSave(t, s, Record{Title: "今天到期", Price: 1, ValidUntil: day(0)})
	mustSave(t, s, Record{Title: "七天后", Price: 1, ValidUntil: day(7)})
	mustSave(t, s, Record{Title: "八天后", Price: 1, ValidUntil: day(8)})
	mustSave(t, s, Record{Title: "长期有效", Price: 1})                           // 无有效期,跳过
	mustSave(t, s, Record{Title: "非法日期", Price: 1, ValidUntil: "2025-13-99"}) // 非法,跳过
	mustSave(t, s, Record{Title: "乱码", Price: 1, ValidUntil: "not-a-date"})   // 非法,跳过

	got := s.ListExpiring(7)
	if len(got) != 3 {
		t.Fatalf("ListExpiring(7) = %d, want 3: %+v", len(got), got)
	}
	// 按 valid_until 升序:已过期最前,依次 今天、边界日。
	if got[0].Title != "过期水泥" || got[1].Title != "今天到期" || got[2].Title != "七天后" {
		t.Errorf("order = %+v", got)
	}
	// days<=0:只返回已过期(<=today)。
	got0 := s.ListExpiring(0)
	if len(got0) != 2 || got0[0].Title != "过期水泥" || got0[1].Title != "今天到期" {
		t.Errorf("ListExpiring(0) = %+v, want 2 条已过期", got0)
	}
	if gotNeg := s.ListExpiring(-3); len(gotNeg) != 2 {
		t.Errorf("ListExpiring(-3) = %+v, want 2 条已过期", gotNeg)
	}
	// days=6:今天+7 超出边界,只剩 2 条。
	if got6 := s.ListExpiring(6); len(got6) != 2 {
		t.Errorf("ListExpiring(6) = %+v, want 2", got6)
	}
}

// TestSuggestAdjustments 调差建议:精确匹配/规格差异匹配/无匹配/多条取最新期
// (空期数视为最旧)/差幅阈值 2% 边界/排序/空输入/库空。
func TestSuggestAdjustments(t *testing.T) {
	s := newTestStore(t)
	mustSave(t, s, Record{Title: "水泥", Price: 110, PriceDate: "2026-09-01", Source: "信息价", Unit: "吨"})
	mustSave(t, s, Record{Title: "钢筋", Price: 95, PriceDate: "2026-08-01", Source: "信息价"})
	mustSave(t, s, Record{Title: "钢筋", Price: 105, PriceDate: "2026-09-01", Source: "供应商比价"}) // 多条取最新
	mustSave(t, s, Record{Title: "石灰", Price: 102, PriceDate: "2026-09-01"})                  // 差幅恰 2.0%
	mustSave(t, s, Record{Title: "粉煤灰", Price: 102.3, PriceDate: "2026-09-01"})               // 2.3%
	mustSave(t, s, Record{Title: "沥青", Price: 90})                                            // 空期数=最旧
	mustSave(t, s, Record{Title: "沥青", Price: 120, PriceDate: "2026-09-01"})                  // 最新
	mustSave(t, s, Record{Title: "碎石", Price: 85, PriceDate: "2026-09-01"})                   // -15%

	entries := []cost.Summary{
		{Name: "cement", Title: "水泥", Unit: "吨", Price: 100},         // 精确匹配 +10%
		{Name: "rebar", Title: "钢筋(HRB400E)", Unit: "吨", Price: 100}, // 规格差异匹配 +5%(取最新)
		{Name: "sand", Title: "砂石", Unit: "吨", Price: 60},            // 无匹配
		{Name: "lime", Title: "石灰", Unit: "吨", Price: 100},           // 2.0% 边界:不提示
		{Name: "flyash", Title: "粉煤灰", Unit: "吨", Price: 100},        // 2.3%:提示
		{Name: "asphalt", Title: "沥青", Unit: "吨", Price: 100},        // 取最新 120 +20%
		{Name: "gravel", Title: "碎石", Unit: "吨", Price: 100},         // -15%
	}
	got := s.SuggestAdjustments(entries)
	if len(got) != 5 {
		t.Fatalf("suggestions = %d, want 5: %+v", len(got), got)
	}
	byName := map[string]AdjustSuggestion{}
	for _, g := range got {
		byName[g.EntryName] = g
	}
	// 精确匹配:字段逐项。
	c := byName["cement"]
	if c.EntryTitle != "水泥" || c.EntryPrice != 100 || c.LatestPrice != 110 ||
		c.Diff != 10 || c.DiffPct != 10 || c.Unit != "吨" || c.LatestSource != "信息价" || c.LatestDate != "2026-09-01" {
		t.Errorf("cement = %+v", c)
	}
	// 规格差异匹配 + 多条取最新期。
	r := byName["rebar"]
	if r.LatestPrice != 105 || r.Diff != 5 || r.DiffPct != 5 || r.LatestDate != "2026-09-01" {
		t.Errorf("rebar = %+v", r)
	}
	// 空期数视为最旧。
	a := byName["asphalt"]
	if a.LatestPrice != 120 || a.Diff != 20 || a.DiffPct != 20 {
		t.Errorf("asphalt = %+v", a)
	}
	// 负差幅保留符号。
	g := byName["gravel"]
	if g.Diff != -15 || g.DiffPct != -15 {
		t.Errorf("gravel = %+v", g)
	}
	// 阈值边界:恰 2.0% 不提示,2.3% 提示。
	if _, ok := byName["lime"]; ok {
		t.Error("lime (exactly 2%) should NOT be suggested")
	}
	if f, ok := byName["flyash"]; !ok || f.DiffPct != 2.3 {
		t.Errorf("flyash = %+v", f)
	}
	// 按 |DiffPct| 降序:20, 15, 10, 5, 2.3。
	wantOrder := []string{"asphalt", "gravel", "cement", "rebar", "flyash"}
	for i, w := range wantOrder {
		if got[i].EntryName != w {
			t.Fatalf("order[%d] = %q, want %q (all: %+v)", i, got[i].EntryName, w, got)
		}
	}
	// 空输入 / 库空 → nil。
	if got := s.SuggestAdjustments(nil); got != nil {
		t.Errorf("nil entries = %+v", got)
	}
	if got := s.SuggestAdjustments([]cost.Summary{}); got != nil {
		t.Errorf("empty entries = %+v", got)
	}
	s2 := newTestStore(t)
	if got := s2.SuggestAdjustments([]cost.Summary{{Name: "x", Title: "水泥", Price: 100}}); got != nil {
		t.Errorf("empty store = %+v", got)
	}
}

// TestStoreUnavailable gdb=nil 时不可用:Save/Get 报错,查询返回 nil。
func TestStoreUnavailable(t *testing.T) {
	s := Open(nil)
	if s.Available() {
		t.Error("nil db should be unavailable")
	}
	if _, err := s.Save(Record{Title: "x"}); err == nil {
		t.Error("save on nil db should error")
	}
	if _, err := s.Get(1); err == nil {
		t.Error("get on nil db should error")
	}
	if got := s.List("", 0); got != nil {
		t.Errorf("list on nil db = %+v", got)
	}
	if got := s.ListExpiring(7); got != nil {
		t.Errorf("expiring on nil db = %+v", got)
	}
	if got := s.SuggestAdjustments([]cost.Summary{{Name: "x", Title: "y", Price: 1}}); got != nil {
		t.Errorf("suggest on nil db = %+v", got)
	}
	if err := s.Delete(1); err != nil {
		t.Errorf("delete on nil db should be no-op, got %v", err)
	}
}
