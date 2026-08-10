package pricefeed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
)

// 四川造价信息网价格表结构（period=758 抓包简化）。
const fixtureHTML = `<!DOCTYPE html><html><head><title>四川省工程造价信息网</title></head><body>
<table id="tbPrice">
  <thead>
    <tr class="tc fb bggray">
      <td class="c1" rowspan="2">名称</td>
      <td class="c2" rowspan="2">规格</td>
      <td rowspan="2">单位</td>
      <td rowspan="2">是否含税</td>
      <td colspan="3">价格</td>
    </tr>
    <tr><td>成都市区</td><td>天府新区</td><td>简阳市</td></tr>
  </thead>
  <tr><td class="c1 tl">热轧光圆钢筋</td><td class="c2 tl">HPB300 Φ12</td><td class="c3">t</td><td class="c3">不含税</td>
    <td><a href="/detail.aspx?matcode=1&amp;area=川A-0010&amp;Period=758" class="orange">￥3181.00</a></td>
    <td><a class="orange"></a></td><td><a class="orange"></a></td></tr>
  <tr><td class="c1 tl">螺纹钢</td><td class="c2 tl">HRB400 Φ20</td><td class="c3">t</td><td class="c3">不含税</td>
    <td><a class="orange">￥3,420.00</a></td><td><a class="orange"></a></td><td><a class="orange"></a></td></tr>
  <tr><td class="c1 tl">普通硅酸盐水泥</td><td class="c2 tl">P.O 42.5</td><td class="c3">t</td><td class="c3">含税</td>
    <td><a class="orange"></a></td><td><a class="orange"></a></td><td><a class="orange"></a></td></tr>
</table></body></html>`

func newTestStore(t *testing.T) *cost.Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	return cost.Open(gdb)
}

func TestParseTable(t *testing.T) {
	rows, err := parseTable(fixtureHTML)
	if err != nil {
		t.Fatalf("parseTable failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (水泥无价跳过), got %d: %+v", len(rows), rows)
	}
	r0 := rows[0]
	if r0.Title != "热轧光圆钢筋" || r0.Spec != "HPB300 Φ12" || r0.Unit != "t" || r0.Price != 3181 || r0.Tax != "不含税" {
		t.Errorf("row0 wrong: %+v", r0)
	}
	if r1 := rows[1]; r1.Title != "螺纹钢" || r1.Price != 3420 {
		t.Errorf("row1 wrong: %+v", r1)
	}
}

func TestParseTable_NoHeader(t *testing.T) {
	if _, err := parseTable("<html><body><table><tr><td>foo</td><td>bar</td></tr></table></body></html>"); err == nil {
		t.Error("expected error for table without 名称/规格 header")
	}
}

func TestMatchRows(t *testing.T) {
	store := newTestStore(t)
	if err := store.Save(cost.Entry{Name: "rebar", Title: "热轧光圆钢筋", Unit: "t", Price: 3000, Status: "现行"}); err != nil {
		t.Fatal(err)
	}
	rows := []Row{
		{Title: "热轧光圆钢筋", Spec: "HPB300 Φ12", Unit: "t", Price: 3181, Tax: "不含税"}, // 更新
		{Title: "螺纹钢", Spec: "HRB400 Φ20", Unit: "t", Price: 3420, Tax: "不含税"},        // 新增
	}
	got := matchRows(rows, store)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].Status != "更新" || got[0].ExistingName != "rebar" || got[0].Diff != 181 || got[0].DiffPct != 6.03 {
		t.Errorf("update candidate wrong: %+v", got[0])
	}
	if got[1].Status != "新增" {
		t.Errorf("new candidate wrong: %+v", got[1])
	}
}

func TestFetch_HTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("period") != "758" {
			t.Errorf("period param missing: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(fixtureHTML))
	}))
	defer srv.Close()

	res, err := Fetch(context.Background(), Source{
		ID: "s1", Name: "四川信息价", URL: srv.URL + "/pricelist.aspx?period=758",
		Parser: "sc_table", Enabled: true,
	}, newTestStore(t))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if res.Period != "758" || res.Rows != 2 {
		t.Errorf("result wrong: %+v", res)
	}
	if len(res.Candidates) != 2 || res.Candidates[0].Price != 3181 {
		t.Errorf("candidates wrong: %+v", res.Candidates)
	}
}

func TestFetch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	_, err := Fetch(context.Background(), Source{ID: "s", Name: "x", URL: srv.URL}, nil)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("expected 403 error, got %v", err)
	}
}

func TestDetectAnomalies(t *testing.T) {
	history := map[string][]History{
		"rebar": {{Name: "rebar", Price: 3000, FetchedAt: "2026-08-01T00:00:00Z"}},
	}
	lookup := func(name string) []History { return history[name] }

	cands := []Candidate{
		{Title: "螺纹钢", ExistingName: "rebar", ExistingPrice: 3000, Price: 3750, Status: "更新"},   // +25% 异常
		{Title: "水泥", ExistingName: "cement", ExistingPrice: 480, Price: 504, Status: "更新"},      // +5% 正常
		{Title: "碎石", ExistingName: "gravel", ExistingPrice: 0, Price: 80, Status: "更新"},         // 无基准，跳过
		{Title: "新钢筋", ExistingName: "", ExistingPrice: 0, Price: 4000, Status: "新增"},           // 新增不判
		{Title: "旧水泥", ExistingName: "cement2", ExistingPrice: 480, Price: 480, Status: "无变化"}, // 无变化不判
	}
	got := DetectAnomalies(cands, lookup)
	if !got[0].Anomaly || !strings.Contains(got[0].AnomalyReason, "+25.0%") {
		t.Errorf("rebar should be anomaly: %+v", got[0])
	}
	if got[1].Anomaly {
		t.Errorf("cement 5%% jump should be normal: %+v", got[1])
	}
	if got[2].Anomaly || got[3].Anomaly || got[4].Anomaly {
		t.Errorf("gravel/new/nochange should not be anomaly: %+v", got[2:])
	}
}

func TestDetectAnomaliesUsesHistoryAsBase(t *testing.T) {
	// 现价 3200，但最近历史发布价是 2500（现价可能已过时）→ 以 2500 为基准。
	history := map[string][]History{
		"rebar": {{Name: "rebar", Price: 2500, FetchedAt: "2026-08-01T00:00:00Z"}},
	}
	cands := []Candidate{
		{Title: "螺纹钢", ExistingName: "rebar", ExistingPrice: 3200, Price: 3150, Status: "更新"},
	}
	got := DetectAnomalies(cands, func(name string) []History { return history[name] })
	// (3150-2500)/2500 = +26% → 异常（若用现价 3200 则 -1.6% 正常）。
	if !got[0].Anomaly {
		t.Errorf("should use history price as base: %+v", got[0])
	}
}

func TestPeriodFromURL(t *testing.T) {
	if got := periodFromURL("http://x/pricelist.aspx?period=758&x=1"); got != "758" {
		t.Errorf("period = %q", got)
	}
	if got := periodFromURL("http://x/list.aspx"); got != "" {
		t.Errorf("period = %q", got)
	}
}

// TestParseRealSichuanPage 用本地缓存的四川造价信息网真实页面验证解析器
// （仅当文件存在时运行，例如 C:\Users\<user>\AppData\Local\Temp\cq_price_758.html）。
func TestParseRealSichuanPage(t *testing.T) {
	path := os.Getenv("CQ_PRICE_FIXTURE")
	if path == "" {
		path = os.Getenv("TEMP") + "\\cq_price_758.html"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skip("真实页面 fixture 不存在，跳过")
	}
	rows, err := parseTable(string(b))
	if err != nil {
		t.Fatalf("real page parse failed: %v", err)
	}
	if len(rows) < 10 {
		t.Fatalf("expected >=10 rows from real page, got %d", len(rows))
	}
	t.Logf("real page parsed %d rows; first: %+v", len(rows), rows[0])
}

// TestFetchLiveSichuan 真实源端到端抓取验证（仅 PRICEFEED_LIVE=1 时运行）。
func TestFetchLiveSichuan(t *testing.T) {
	if os.Getenv("PRICEFEED_LIVE") != "1" {
		t.Skip("PRICEFEED_LIVE=1 时运行真实抓取验证")
	}
	res, err := Fetch(context.Background(), Source{
		ID: "live-sc", Name: "四川造价信息网（期 758）",
		URL: "http://202.61.90.35:8032/pubpages/pricelist.aspx?period=758",
		Parser: "sc_table", Enabled: true,
	}, newTestStore(t))
	if err != nil {
		t.Fatalf("live fetch failed: %v", err)
	}
	if res.Rows < 10 || res.Period != "758" {
		t.Fatalf("live result wrong: rows=%d period=%q", res.Rows, res.Period)
	}
	t.Logf("live fetch OK: %d rows, first candidate: %+v", res.Rows, res.Candidates[0])
}
