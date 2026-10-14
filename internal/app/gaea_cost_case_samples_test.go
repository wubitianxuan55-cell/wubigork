package app

// gaea_cost_case_samples_test.go — 三期 3.3 对标升级：项目案例清单综合单价
// 并入 AI 组价分位数聚合。手填价优先、单定额现算、1:N 组合项诚实跳过、
// 名称+单位归一匹配；案例只进价格带不进 LLM 入参。

import (
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

func seedCaseProject(t *testing.T, a *App, name, location string) int64 {
	t.Helper()
	id, err := a.hubWorkcostStore().UpsertBillProject(name, "case.xlsx", "导入", location, "", "", nil, false, 0)
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
	return id
}

func mustSaveBillItem(t *testing.T, a *App, it workcost.BillItem) workcost.BillItem {
	t.Helper()
	saved, err := a.hubWorkcostStore().SaveBillItem(it)
	if err != nil {
		t.Fatalf("seed bill item %q: %v", it.Title, err)
	}
	return saved
}

// TestWhisperCaseSamples 的造价对应面：匹配/单价解析/跳过三口径一体验证。
func TestCasePriceSamplesMatchAndPrice(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()

	pid := seedCaseProject(t, a, "测试案例工程", "成都")
	// 手填价样本（外委包干/项目级调价）
	mustSaveBillItem(t, a, workcost.BillItem{ProjectID: pid, Title: "直形墙 C30", Unit: "m³",
		PriceOverride: 480.5, QuotaCodes: []string{}})
	// 单定额现算样本：Q1 = 材料 2×100 + 人工 1×50 = 250（人材机口径，无取费）
	if _, err := a.hubWorkcostStore().SaveQuota(workcost.Quota{Code: "Q1", Title: "墙浇筑定额", Unit: "m³",
		Items: []workcost.QuotaItem{
			// ResourceCode 必填（SaveQuota 丢空码行）；资源不在索引=走快照价兜底
			{QuotaCode: "Q1", ResourceCode: "R-CL30", Kind: "材料", Title: "C30 混凝土", Unit: "m³", Quantity: 2, ResourcePrice: 100},
			{QuotaCode: "Q1", ResourceCode: "R-RG", Kind: "人工", Title: "浇筑人工", Unit: "工日", Quantity: 1, ResourcePrice: 50},
		}}, nil); err != nil {
		t.Fatalf("seed quota: %v", err)
	}
	mustSaveBillItem(t, a, workcost.BillItem{ProjectID: pid, Title: "直形墙C30加厚", Unit: "m³",
		QuotaCode: "Q1", QuotaCodes: []string{"Q1"}})
	// 1:N 组合项：无单一单价，诚实跳过。QuotaCodes 是读侧从 links 表聚合的
	// （SaveBillItem 不写 links），须显式挂接成 1:N 形状；Q2 也须真实存在
	// （AttachBillQuotaLink 校验定额在库）。
	if _, err := a.hubWorkcostStore().SaveQuota(workcost.Quota{Code: "Q2", Title: "垫层定额", Unit: "m³",
		Items: []workcost.QuotaItem{
			{QuotaCode: "Q2", ResourceCode: "R-DC", Kind: "材料", Title: "垫层混凝土", Unit: "m³", Quantity: 1, ResourcePrice: 80},
		}}, nil); err != nil {
		t.Fatalf("seed quota Q2: %v", err)
	}
	combo := mustSaveBillItem(t, a, workcost.BillItem{ProjectID: pid, Title: "直形墙C30组合", Unit: "m³",
		QuotaCode: "Q1"})
	for _, code := range []string{"Q1", "Q2"} {
		if _, err := a.hubWorkcostStore().AttachBillQuotaLink(combo.ID, code, 0); err != nil {
			t.Fatalf("attach %s: %v", code, err)
		}
	}
	// 无手填价、无定额：跳过
	mustSaveBillItem(t, a, workcost.BillItem{ProjectID: pid, Title: "直形墙C30空行", Unit: "m³",
		QuotaCodes: []string{}})
	// 名称不匹配
	mustSaveBillItem(t, a, workcost.BillItem{ProjectID: pid, Title: "挖沟槽土方", Unit: "m³",
		PriceOverride: 99, QuotaCodes: []string{}})
	// 名称匹配但单位不同（m² 模板）：m³ 查询排除，空单位查询纳入
	mustSaveBillItem(t, a, workcost.BillItem{ProjectID: pid, Title: "直形墙C30模板", Unit: "m²",
		PriceOverride: 30, QuotaCodes: []string{}})

	samples := a.casePriceSamples("直形墙C30", "m³")
	if len(samples) != 2 {
		t.Fatalf("m³ 查询应得 2 样本（手填+现算）: %+v", samples)
	}
	var manual, composed bool
	for _, s := range samples {
		if s.Source != "案例" || s.Region != "成都" {
			t.Fatalf("来源/地区不符: %+v", s)
		}
		if s.Title != "案例·测试案例工程" {
			t.Fatalf("样本 Title 应溯源项目名: %+v", s)
		}
		switch {
		case s.Name == "直形墙 C30" && s.Price == 480.5:
			manual = true
		case s.Name == "直形墙C30加厚" && s.Price == 250:
			composed = true
		}
	}
	if !manual || !composed {
		t.Fatalf("应含手填价 480.5 与现算价 250 两样本: %+v", samples)
	}

	all := a.casePriceSamples("直形墙C30", "")
	if len(all) != 3 {
		t.Fatalf("空单位查询应含模板样本共 3 条: %+v", all)
	}
}

// TestCaseNameMatch 归一匹配护栏：双向包含+短侧 ≥2 字防单字误伤。
func TestCaseNameMatch(t *testing.T) {
	if !caseNameMatch(normalizeCostName("直形墙"), normalizeCostName("直形墙 C30 加厚")) {
		t.Fatal("子串应匹配")
	}
	if !caseNameMatch(normalizeCostName("直形墙C30加厚段"), normalizeCostName("直形墙C30")) {
		t.Fatal("反向包含应匹配（长查询含短条目名）")
	}
	if caseNameMatch(normalizeCostName("墙"), normalizeCostName("直形墙 C30")) {
		t.Fatal("单字短侧不得匹配（防误伤全部墙类）")
	}
	if caseNameMatch(normalizeCostName("直形墙"), normalizeCostName("挖沟槽土方")) {
		t.Fatal("无包含关系不得匹配")
	}
}

// TestGaeaCostComposeBandIncludesCaseSamples 端到端：组价视图的价格带与证据链
// 出现案例来源样本（3.3 验收口径「分位数表出现案例来源样本」），且案例不进
// LLM 拆解入参（similar 保持纯库样本）。
func TestGaeaCostComposeBandIncludesCaseSamples(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	// GaeaCostCompose 全链（检索增强/LLM 拆解降级）要读 core 字段——workcostTestApp
	// 的裸 App 无 core，检索链 localSearchEmbedder 穿透 nil 内嵌指针 panic。
	a.core = &core{cfg: &config.Config{}}

	// 库条目（同名命中检索）
	if err := a.hubCostStore().Save(cost.Entry{
		Name: "直形墙C30", Title: "直形墙C30", CategoryPath: "建筑/墙",
		Unit: "m³", Price: 500, Status: "现行",
	}); err != nil {
		t.Fatalf("seed entry: %v", err)
	}
	pid := seedCaseProject(t, a, "案例工程乙", "绵阳")
	mustSaveBillItem(t, a, workcost.BillItem{ProjectID: pid, Title: "直形墙C30", Unit: "m³",
		PriceOverride: 480.5, QuotaCodes: []string{}})

	v, err := a.GaeaCostCompose("直形墙C30", "m³")
	if err != nil {
		t.Fatal(err)
	}
	if v.Band == nil {
		t.Fatal("价格带不应为 nil")
	}
	if v.Band.Samples != 2 {
		t.Fatalf("带样本应=库1+案例1: %d", v.Band.Samples)
	}
	var caseRow *CostComposeEvidence
	for i := range v.Evidence {
		if v.Evidence[i].Source == "案例" {
			caseRow = &v.Evidence[i]
		}
	}
	if caseRow == nil {
		t.Fatalf("证据链应含案例来源样本: %+v", v.Evidence)
	}
	if caseRow.Price != 480.5 || caseRow.Title != "案例·案例工程乙" {
		t.Fatalf("案例样本内容不符: %+v", caseRow)
	}
}
