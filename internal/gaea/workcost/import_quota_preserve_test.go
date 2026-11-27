package workcost

import (
	"context"
	"strings"
	"testing"
)

// ── 2026-11-27 造价库批量改造复盘（软件不足与改进建议）刀A 回归 ─────────────
//
// P1：import_add 同码更新此前整行覆盖——不带 chapter 的更新把 277 条有章节归档的
// 定额清了空，含量行不给还会被 saveQuota 的先删后插静默清掉。落码口径：同码更新
// =没给的头字段保持原值，未给含量行=保留原含量行。
// P2/P15：quotas/resources 列表加 offset 翻页（排序带编码兜底，窗口稳定），配合
// 回执总数自证穷尽，不再靠前缀扫描。
// 地区限省：库内一律省级全称，查询词归一（「四川」→「四川省」）后精确匹配。

func importStore(t *testing.T) *Store {
	t.Helper()
	s, cleanup := newWorkcostStore(t)
	t.Cleanup(cleanup)
	seedDict(t, s, DictSpecialty, "土壤修复")
	seedDict(t, s, DictRegion, "四川省", "重庆市")
	return s
}

func TestImportQuotaUpdatePreservesHeader(t *testing.T) {
	s := importStore(t)
	if _, err := s.SaveResource(Resource{Code: "L01", Kind: KindLabor, Title: "普工", Unit: "工日", CurrentPrice: 100, Region: "四川省"}); err != nil {
		t.Fatalf("建工料机失败: %v", err)
	}
	seed := Quota{Code: "D-01", Title: "场地平整", Unit: "m²", Specialty: "土壤修复",
		Chapter: "01 场地整理", CategoryPath: "道路/基层", Note: "原注", Region: "四川省", PriceDate: "2026-05",
		Items: []QuotaItem{{ResourceCode: "L01", Quantity: 1}}}
	if _, err := s.SaveQuota(seed, nil); err != nil {
		t.Fatalf("建既有定额失败: %v", err)
	}

	// 更新会话只给编码与含量行：头字段必须原样保留；项目地区（重庆市）不得改写
	// 既有行地区（首标纪律）。
	imp, err := s.OpenImport("", ImportProject{Name: "更新会话", Region: "重庆市", PricePeriod: "2026-11"}, ImportControls{})
	if err != nil {
		t.Fatalf("开导入会话失败: %v", err)
	}
	if _, err := imp.AddQuotas(s, []ImportQuota{{Code: "D-01",
		Items: []ImportQuotaItem{{ResourceCode: "L01", Quantity: 2}}}}); err != nil {
		t.Fatalf("定额入暂存失败: %v", err)
	}
	res, err := imp.Commit(s)
	if err != nil {
		t.Fatalf("提交失败: %v", err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("不应有写入错误: %v", res.Errors)
	}
	got, err := s.GetQuota("D-01")
	if err != nil {
		t.Fatalf("回读失败: %v", err)
	}
	if got.Title != "场地平整" || got.Unit != "m²" || got.Specialty != "土壤修复" ||
		got.Chapter != "01 场地整理" || got.CategoryPath != "道路/基层" || got.Note != "原注" ||
		got.Region != "四川省" || got.PriceDate != "2026-05" {
		t.Fatalf("未给字段被覆盖：%+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].ResourceCode != "L01" || got.Items[0].Quantity != 2 {
		t.Fatalf("含量行应更新为 L01×2，实得 %+v", got.Items)
	}

	// 显式给的字段照常生效。
	imp2, err := s.OpenImport("", ImportProject{Name: "更新会话二", Region: "重庆市", PricePeriod: "2026-11"}, ImportControls{})
	if err != nil {
		t.Fatalf("开第二次会话失败: %v", err)
	}
	if _, err := imp2.AddQuotas(s, []ImportQuota{{Code: "D-01", Title: "场地平整（改）", Chapter: "02 路基"}}); err != nil {
		t.Fatalf("第二次入暂存失败: %v", err)
	}
	if _, err := imp2.Commit(s); err != nil {
		t.Fatalf("第二次提交失败: %v", err)
	}
	got2, err := s.GetQuota("D-01")
	if err != nil {
		t.Fatalf("二次回读失败: %v", err)
	}
	if got2.Title != "场地平整（改）" || got2.Chapter != "02 路基" {
		t.Fatalf("显式字段未生效：%+v", got2)
	}
	if got2.CategoryPath != "道路/基层" || len(got2.Items) != 1 {
		t.Fatalf("二次更新把没给的字段/含量行丢了：%+v", got2)
	}
}

func TestQuotaListOffsetPaging(t *testing.T) {
	s := importStore(t)
	ctx := context.Background()
	for _, c := range []string{"D-A01", "D-A02", "D-A03", "D-A04", "D-A05"} {
		if _, err := s.SaveQuota(Quota{Code: c, Title: "工序" + c, Unit: "m²", Specialty: "土壤修复"}, nil); err != nil {
			t.Fatalf("建 %s 失败: %v", c, err)
		}
	}
	h := NewSearcher(s, nil, nil, nil)
	seen := ""
	for offset := 0; ; offset += 2 {
		hits, err := h.Quotas(ctx, SearchQuery{Specialty: "土壤修复", Limit: 2, Offset: offset})
		if err != nil {
			t.Fatalf("翻页失败: %v", err)
		}
		if len(hits) > 2 {
			t.Fatalf("单页不应超过 limit: %d", len(hits))
		}
		if len(hits) == 0 {
			break
		}
		for _, h := range hits {
			seen += h.Item.Code + ","
		}
		if offset > 50 {
			t.Fatalf("翻页未收敛（回环？）")
		}
	}
	want := "D-A01,D-A02,D-A03,D-A04,D-A05,"
	if seen != want {
		t.Fatalf("翻页应不重不漏拿全量：got %q want %q", seen, want)
	}
}

func TestResourceRegionQueryNormalized(t *testing.T) {
	s := importStore(t)
	h := NewSearcher(s, nil, nil, nil)
	if _, err := s.SaveResource(Resource{Code: "L01", Kind: KindLabor, Title: "普工", Unit: "工日", CurrentPrice: 100, Region: "四川省"}); err != nil {
		t.Fatalf("建工料机失败: %v", err)
	}
	ctx := context.Background()
	for _, region := range []string{"四川省", "四川"} {
		hits, err := h.Resources(ctx, SearchQuery{Region: region, Limit: 10})
		if err != nil {
			t.Fatalf("region=%s 检索失败: %v", region, err)
		}
		if len(hits) != 1 {
			t.Fatalf("region=%s 应命中 1 条，实得 %d", region, len(hits))
		}
	}
	hits, err := h.Resources(ctx, SearchQuery{Region: "重庆市", Limit: 10})
	if err != nil {
		t.Fatalf("检索失败: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("重庆市不应命中四川省的行")
	}
}

// P7：显式给了不同专业才退回，文案指路「去掉 specialty 或借鉴生成」，不再说
// 「另起编码」误导模型建重复定额；不给专业=沿用既有行，不退回。
func TestImportQuotaSpecialtyGateWording(t *testing.T) {
	s := importStore(t)
	seedDict(t, s, DictSpecialty, "固废")
	if _, err := s.SaveResource(Resource{Code: "L01", Kind: KindLabor, Title: "普工", Unit: "工日", CurrentPrice: 100, Region: "四川省"}); err != nil {
		t.Fatalf("建工料机失败: %v", err)
	}
	if _, err := s.SaveQuota(Quota{Code: "D-01", Title: "场地平整", Unit: "m²", Specialty: "土壤修复"}, nil); err != nil {
		t.Fatalf("建既有定额失败: %v", err)
	}
	imp, err := s.OpenImport("", ImportProject{Name: "闸门会话"}, ImportControls{})
	if err != nil {
		t.Fatalf("开导入失败: %v", err)
	}
	res, err := imp.AddQuotas(s, []ImportQuota{{Code: "D-01", Title: "场地平整", Specialty: "固废"}})
	if err != nil {
		t.Fatalf("入暂存失败: %v", err)
	}
	if len(res.Rejected) != 1 {
		t.Fatalf("显式改专业应退回 1 行: %+v", res)
	}
	msg := strings.Join(res.Rejected[0].Reasons, "；")
	if !strings.Contains(msg, "去掉行上的 specialty") || !strings.Contains(msg, "borrow_quota") || strings.Contains(msg, "另起编码") {
		t.Fatalf("退回文案不对：%s", msg)
	}
	res2, err := imp.AddQuotas(s, []ImportQuota{{Code: "D-01"}})
	if err != nil {
		t.Fatalf("入暂存失败: %v", err)
	}
	if len(res2.Rejected) != 0 || res2.Accepted != 1 {
		t.Fatalf("不给专业应沿用既有行不退回: %+v", res2)
	}
}
