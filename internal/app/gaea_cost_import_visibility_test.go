package app

// T7-2 可见性收口：GaeaCostImportAIParse 的 textFallback 截断（与 vision
// 6000 rune 对齐）与 GaeaCostImportApply 整批事务（部分行失败→整批回滚）测试。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	gconfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/modelengine"
)

// ── textFallback 截断（与 vision 6000 rune 对齐） ─────────────────────

// TestTruncateModelInput_ShortPassthrough 短文本原样返回（不截断）。
func TestTruncateModelInput_ShortPassthrough(t *testing.T) {
	in := "这是一段很短的报价文本"
	if got := truncateModelInput(in); got != in {
		t.Errorf("短文本不应截断: %q", got)
	}
}

// TestTruncateModelInput_LongTruncatedTo6000 超长文本截断到 6000 rune 并
// 带截断标注（rune 感知，多字节字符不撕裂）。
func TestTruncateModelInput_LongTruncatedTo6000(t *testing.T) {
	in := strings.Repeat("报价单文本", 2000) // 25000 rune
	got := truncateModelInput(in)
	if !strings.Contains(got, "…（已截断）") {
		t.Errorf("超长文本应带截断标注")
	}
	content := strings.TrimSuffix(got, "\n…（已截断）")
	if len([]rune(content)) != maxModelInputRunes {
		t.Errorf("截断后内容 = %d rune, want %d", len([]rune(content)), maxModelInputRunes)
	}
	if !strings.HasPrefix(got, strings.Repeat("报价单文本", 1000)[:6000]) {
		t.Error("截断应保留原文开头")
	}
}

// TestTruncateModelInput_ExactBoundary 恰好 6000 rune 不截断；6001 截断。
func TestTruncateModelInput_ExactBoundary(t *testing.T) {
	exact := strings.Repeat("a", maxModelInputRunes)
	if got := truncateModelInput(exact); got != exact {
		t.Errorf("恰好 6000 不应截断")
	}
	got := truncateModelInput(exact + "b")
	if !strings.Contains(got, "…（已截断）") {
		t.Error("6001 应截断")
	}
}

// costAIParseApp 构造 GaeaCostImportAIParse 全链路 App：herdsman 指向 mock LLM，
// 注入 bridge client（AI 解析走办公模型流式通道）；成本库隔离到临时目录。
func costAIParseApp(t *testing.T, handler http.HandlerFunc) (*App, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	gdb := db.GetDatabase(gconfig.MemoryUserDir())
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(gconfig.MemoryUserDir()) })
	SetCostStoreForTest(cost.Open(gdb))
	t.Cleanup(ResetCostStoreForTest)

	cfg := &config.Config{FuncOfficeEnabled: true, FuncOfficeEngine: "herdsman", FuncOfficeModel: "m1"}
	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "herdsman", Enabled: true, BaseURL: srv.URL, Models: []modelengine.ModelInfo{{ID: "m1"}}}); err != nil {
		t.Fatal(err)
	}
	client := ai.NewClient(cfg)
	client.SetEngineManager(mgr)
	bridge.SetClient(client)
	t.Cleanup(func() { bridge.SetClient(nil) })
	a := &App{core: &core{cfg: cfg, engineMgr: mgr, client: client}}
	a.ctx = context.Background()
	return a, srv
}

// TestGaeaCostImportAIParse_TextFallbackTruncated 全链路：PDF 文本回退路径的
// 超长文本送入模型前被截断到 6000 rune（与 vision 对齐），模型收到的提示
// 词携带截断标注，不再整段塞入。
func TestGaeaCostImportAIParse_TextFallbackTruncated(t *testing.T) {
	var gotUser string
	a, _ := costAIParseApp(t, func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(400)
			return
		}
		msgs := req["messages"].([]any)
		gotUser, _ = msgs[len(msgs)-1].(map[string]any)["content"].(string)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"choices":[{"index":0,"delta":{"content":"[{\"title\":\"测试条目\",\"price\":100}]"}}]}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})

	// PDF 扩展名 → RawTable 失败 → 走 textFallback；注入超长提取文本。
	path := filepath.Join(t.TempDir(), "报价单.pdf")
	if err := os.WriteFile(path, []byte("pdf bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	origExtract := extractImportText
	extractImportText = func(string) (string, error) {
		return "材料名称 规格 单位 单价\n" + strings.Repeat("水泥 P.O42.5 吨 480 元\n", 3000), nil
	}
	t.Cleanup(func() { extractImportText = origExtract })

	pv, err := a.GaeaCostImportAIParse(path)
	if err != nil {
		t.Fatalf("GaeaCostImportAIParse: %v", err)
	}
	if len(pv.Rows) == 0 {
		t.Fatal("应解析出候选行")
	}
	if gotUser == "" {
		t.Fatal("模型未收到提示词")
	}
	if !strings.Contains(gotUser, "…（已截断）") {
		t.Errorf("提示词应带截断标注")
	}
	if len([]rune(gotUser)) > maxModelInputRunes+100 {
		t.Errorf("提示词应被截断到 ~6000 rune, got %d", len([]rune(gotUser)))
	}
}

// ── GaeaCostImportApply 整批事务 ──────────────────────────────────────

// TestGaeaCostImportApply_AllRowsCommit 全合法批次单事务提交。
func TestGaeaCostImportApply_AllRowsCommit(t *testing.T) {
	a := newVisionTestApp(t)
	n, err := a.GaeaCostImportApply([]CostEntry{
		{CostSummary: CostSummary{Title: "P.O 42.5 水泥", Price: 480, Unit: "吨"}},
		{CostSummary: CostSummary{Title: "HP300 高频液压振动锤", Price: 3200, Unit: "台班"}},
		{
			CostSummary: CostSummary{
				Title: "挖一般土方", Category: "土方工程",
				CategoryPath: "综合单价/道路工程/土方工程", Unit: "m³", Price: 3.79,
				LaborFee: 0, MaterialFee: 0, MachineFee: 3,
			},
			ManagementFee: 0.09, ProfitFee: 0.3, AdvanceFee: 0.09, TaxRate: 9,
			Components: []CostComponentView{
				{Kind: "机械", Title: "挖土方(甩土)", Unit: "m³", Quantity: 1, Price: 3, Amount: 3, Note: "3元/m³"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if n != 3 {
		t.Errorf("写入条数 = %d, want 3", n)
	}
	list := a.hubCostStore().List()
	if len(list) != 3 {
		t.Fatalf("库内条目 = %d, want 3: %+v", len(list), list)
	}
	// 综合单价子目：人材机二级（合计 + 组成行）+ 费率 全链路写回。
	got := a.GaeaCostGet(cost.SlugName("挖一般土方"))
	if got == nil {
		t.Fatal("挖一般土方 未写入")
	}
	if got.LaborFee != 0 || got.MaterialFee != 0 || got.MachineFee != 3 ||
		got.ManagementFee != 0.09 || got.ProfitFee != 0.3 || got.AdvanceFee != 0.09 || got.TaxRate != 9 {
		t.Errorf("费率/人材机合计 = %+v", got)
	}
	if len(got.Components) != 1 || got.Components[0].Title != "挖土方(甩土)" ||
		got.Components[0].Amount != 3 {
		t.Errorf("组成行 = %+v", got.Components)
	}
}

// TestGaeaCostImportApply_InvalidRowRollsBackAll 批次中任一行无效（价格为负）：
// 整个批次拒绝，合法行也不得写入（单事务全成或全回滚）。
func TestGaeaCostImportApply_InvalidRowRollsBackAll(t *testing.T) {
	a := newVisionTestApp(t)

	// 先写入一条基线，用于确认回滚不影响既有数据。
	if _, err := a.GaeaCostImportApply([]CostEntry{{CostSummary: CostSummary{Title: "基线条目", Price: 10}}}); err != nil {
		t.Fatalf("基线写入: %v", err)
	}

	_, err := a.GaeaCostImportApply([]CostEntry{
		{CostSummary: CostSummary{Title: "钢材", Price: 5000, Unit: "吨"}},
		{CostSummary: CostSummary{Title: "沙子", Price: -1, Unit: "吨"}}, // 无效行：负价
		{CostSummary: CostSummary{Title: "碎石", Price: 200, Unit: "吨"}},
	})
	if err == nil {
		t.Fatal("含无效行的批次应报错")
	}
	if !strings.Contains(err.Error(), "第 2 行无效") {
		t.Errorf("错误应指明第 2 行: %v", err)
	}

	list := a.hubCostStore().List()
	if len(list) != 1 {
		t.Fatalf("回滚后库内应只剩基线条目, got %d: %+v", len(list), list)
	}
	if list[0].Title != "基线条目" {
		t.Errorf("基线条目应保留: %+v", list[0])
	}
}

// TestGaeaCostImportApply_EmptyTitleRollsBackAll 空标题行同样导致整批失败。
func TestGaeaCostImportApply_EmptyTitleRollsBackAll(t *testing.T) {
	a := newVisionTestApp(t)
	_, err := a.GaeaCostImportApply([]CostEntry{
		{CostSummary: CostSummary{Title: "合法条目", Price: 10}},
		{CostSummary: CostSummary{Price: 99}}, // 标题为空
	})
	if err == nil {
		t.Fatal("空标题行应使整批失败")
	}
	if len(a.hubCostStore().List()) != 0 {
		t.Fatal("整批应回滚，无任何条目写入")
	}
}

// TestGaeaCostImportApply_EmptyBatchNoop 空批次直接返回 0，无副作用。
func TestGaeaCostImportApply_EmptyBatchNoop(t *testing.T) {
	a := newVisionTestApp(t)
	n, err := a.GaeaCostImportApply(nil)
	if err != nil || n != 0 {
		t.Fatalf("空批次应返回 (0,nil), got (%d,%v)", n, err)
	}
	if len(a.hubCostStore().List()) != 0 {
		t.Fatal("空批次不应写库")
	}
}

// TestNormalizeCostEntryForTx 归一化：slug 名称、默认分类/状态、CategoryPath。
func TestNormalizeCostEntryForTx(t *testing.T) {
	e, err := normalizeCostEntryForTx(CostEntry{CostSummary: CostSummary{Title: "P.O 42.5 水泥", Price: 480}})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if e.Name != cost.SlugName("P.O 42.5 水泥") {
		t.Errorf("Name = %q", e.Name)
	}
	if e.Category != "其他" || e.Status != "现行" || e.CategoryPath != "其他" {
		t.Errorf("默认值异常: %+v", e)
	}
	if e.CreatedAt.IsZero() || e.UpdatedAt.IsZero() {
		t.Error("时间戳不应为零值")
	}
}
