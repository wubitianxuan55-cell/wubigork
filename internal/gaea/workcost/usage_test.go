package workcost

import (
	"strings"
	"testing"
)

// usage_test.go — 三级数据引用面与删除护栏（2026-11-25 造价库重建）：
// 清单挂接集合的整体替换、定额/工料机被谁引用、强删前的解除挂接。

// usageFixture 建一条 清单 → 定额 → 工料机 的最小引用链：
//
//	R（工料机）← A（定额，含量行引用 R）；B（定额，无含量行）
//	P1.WP01 → A；P2.WP02 → A；P2.WP03 → B
func usageFixture(t *testing.T) (*Store, Resource, string, string) {
	t.Helper()
	s, cleanup := newWorkcostStore(t)
	t.Cleanup(cleanup)
	r, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "C20商品混凝土", Unit: "m³", BasePrice: 450})
	if err != nil {
		t.Fatalf("建资源失败: %v", err)
	}
	for _, q := range []Quota{
		{Code: "A", Title: "垫层", Unit: "m³", Items: []QuotaItem{{ResourceCode: r.Code, Quantity: 1.02}}},
		{Code: "B", Title: "零含量占位", Unit: "项"},
	} {
		if _, err := s.SaveQuota(q, nil); err != nil {
			t.Fatalf("建定额 %s 失败: %v", q.Code, err)
		}
	}
	p1, _ := s.UpsertBillProject("P1", "", "", "", "", "", nil, false, 0)
	p2, _ := s.UpsertBillProject("P2", "", "", "", "", "", nil, false, 0)
	for _, b := range []BillItem{
		{ProjectID: p1, Code: "WP01", Title: "垫层一", Unit: "m³"},
		{ProjectID: p2, Code: "WP02", Title: "垫层二", Unit: "m³"},
		{ProjectID: p2, Code: "WP03", Title: "占位项", Unit: "项"},
	} {
		saved, err := s.SaveBillItem(b)
		if err != nil {
			t.Fatalf("建清单项失败: %v", err)
		}
		code := "A"
		if b.Code == "WP03" {
			code = "B"
		}
		if _, err := s.AttachBillQuotaLink(saved.ID, code, 0); err != nil {
			t.Fatalf("挂接定额失败: %v", err)
		}
	}
	return s, *r, "A", "B"
}

// TestSaveBillItemLinksReplacesSet 整体替换语义：给什么就是什么；空集合=退回
// 手填行；悬空定额整批拒绝且不动既有挂接；同码去重保序。
func TestSaveBillItemLinksReplacesSet(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	r, err := s.SaveResource(Resource{Kind: KindLabor, Title: "普工", Unit: "工日", BasePrice: 200})
	if err != nil {
		t.Fatalf("建资源失败: %v", err)
	}
	for _, code := range []string{"A", "B", "C"} {
		if _, err := s.SaveQuota(Quota{Code: code, Title: "定额" + code, Unit: "m²", Items: []QuotaItem{{ResourceCode: r.Code, Quantity: 1}}}, nil); err != nil {
			t.Fatalf("建定额 %s 失败: %v", code, err)
		}
	}
	pid, _ := s.UpsertBillProject("P", "", "", "", "", "", nil, false, 0)
	item, err := s.SaveBillItem(BillItem{ProjectID: pid, Code: "WP01", Title: "抹灰", Unit: "m²"})
	if err != nil {
		t.Fatalf("建清单项失败: %v", err)
	}
	codes := func() string {
		links, err := s.BillQuotaLinks(item.ID)
		if err != nil {
			t.Fatalf("读挂接失败: %v", err)
		}
		var out []string
		for _, l := range links {
			out = append(out, l.QuotaCode)
		}
		return strings.Join(out, "+")
	}
	if _, err := s.SaveBillItemLinks(item.ID, []BillQuotaLinkInput{{QuotaCode: "A"}, {QuotaCode: "B"}}); err != nil {
		t.Fatalf("落挂接失败: %v", err)
	}
	if got := codes(); got != "A+B" {
		t.Fatalf("挂接应为 A+B，得到 %q", got)
	}
	got, _ := s.FindBillItem(pid, "WP01")
	if got == nil || got.QuotaCode != "A" {
		t.Fatalf("主定额列应同步成集合首项 A，得到 %+v", got)
	}
	// 整体替换：换成 [C]，A/B 解除。
	if _, err := s.SaveBillItemLinks(item.ID, []BillQuotaLinkInput{{QuotaCode: "C"}}); err != nil {
		t.Fatalf("替换挂接失败: %v", err)
	}
	if c := codes(); c != "C" {
		t.Fatalf("替换后挂接应为 C，得到 %q", c)
	}
	// 悬空定额：拒绝，既有挂接不动。
	if _, err := s.SaveBillItemLinks(item.ID, []BillQuotaLinkInput{{QuotaCode: "NOPE-1"}}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("悬空定额应拒绝，得到 err=%v", err)
	}
	if c := codes(); c != "C" {
		t.Fatalf("拒绝后既有挂接应保持 C，得到 %q", c)
	}
	// 同码去重保序。
	if _, err := s.SaveBillItemLinks(item.ID, []BillQuotaLinkInput{{QuotaCode: "A"}, {QuotaCode: " A "}, {QuotaCode: "B"}}); err != nil {
		t.Fatalf("落挂接失败: %v", err)
	}
	if c := codes(); c != "A+B" {
		t.Fatalf("同码应去重保序为 A+B，得到 %q", c)
	}
	// 空集合：退回手填行。
	if _, err := s.SaveBillItemLinks(item.ID, nil); err != nil {
		t.Fatalf("清空挂接失败: %v", err)
	}
	if c := codes(); c != "" {
		t.Fatalf("空集合应清空挂接，得到 %q", c)
	}
	got, _ = s.FindBillItem(pid, "WP01")
	if got.QuotaCode != "" {
		t.Fatalf("清空后主定额列应为空，得到 %q", got.QuotaCode)
	}
}

// TestQuotaUsageAndResourceUsage 引用面：定额被哪些清单项引用（含引用表没行的
// 旧数据兜底）、工料机被哪些定额含量行引用。
func TestQuotaUsageAndResourceUsage(t *testing.T) {
	s, r, _, _ := usageFixture(t)
	refs, err := s.QuotaUsage("A")
	if err != nil {
		t.Fatalf("QuotaUsage 失败: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("定额 A 应被 2 条清单项引用，得到 %d（%+v）", len(refs), refs)
	}
	if refs[0].Owner != "P1" || refs[1].Owner != "P2" {
		t.Fatalf("引用方应按项目归属，得到 %q / %q", refs[0].Owner, refs[1].Owner)
	}
	if n, _ := s.QuotaReferenced("A"); n != 2 {
		t.Fatalf("QuotaReferenced(A) = %d, want 2", n)
	}
	// 没被引用的定额：空切片（非 nil，绑定面契约）。
	if refs, err := s.QuotaUsage("B"); err != nil || len(refs) != 1 {
		t.Fatalf("定额 B 应被 WP03 引用 1 次，得到 %d err=%v", len(refs), err)
	}
	if refs, err := s.QuotaUsage("MISSING"); err != nil || refs == nil || len(refs) != 0 {
		t.Fatalf("不存在的定额应返回空切片，得到 %+v err=%v", refs, err)
	}
	// 旧数据兜底：只有单值 quota_code、引用表没行，照样算引用。
	pid, _ := s.UpsertBillProject("P3", "", "", "", "", "", nil, false, 0)
	if _, err := s.SaveBillItem(BillItem{ProjectID: pid, Code: "WP09", Title: "旧数据行", Unit: "m³", QuotaCode: "A"}); err != nil {
		t.Fatalf("建旧数据清单项失败: %v", err)
	}
	if refs, err := s.QuotaUsage("A"); err != nil || len(refs) != 3 {
		t.Fatalf("单值列兜底后定额 A 应有 3 条引用，得到 %d err=%v", len(refs), err)
	}
	// 工料机引用面。
	rrefs, err := s.ResourceUsage(r.Code)
	if err != nil {
		t.Fatalf("ResourceUsage 失败: %v", err)
	}
	if len(rrefs) != 1 || rrefs[0].Code != "A" || rrefs[0].Quantity != 1.02 {
		t.Fatalf("工料机应只被定额 A 的含量行引用（含量 1.02），得到 %+v", rrefs)
	}
}

// TestDetachQuotaEverywhere 强删前的清理：引用表与单值列同步解除，清单项退回手填行。
func TestDetachQuotaEverywhere(t *testing.T) {
	s, _, _, _ := usageFixture(t)
	n, err := s.DetachQuotaEverywhere("A")
	if err != nil {
		t.Fatalf("DetachQuotaEverywhere 失败: %v", err)
	}
	if n != 2 {
		t.Fatalf("应解除 2 条清单项的挂接，得到 %d", n)
	}
	if refs, _ := s.QuotaUsage("A"); len(refs) != 0 {
		t.Fatalf("解除后定额 A 不应再被引用，得到 %d 条", len(refs))
	}
	projects, _ := s.BillProjects()
	for _, p := range projects {
		items, _ := s.BillItems(p.ID)
		for _, it := range items {
			if it.Code == "WP01" && (it.QuotaCode != "" || len(it.QuotaCodes) != 0) {
				t.Fatalf("WP01 应退回手填行，得到 quota_code=%q codes=%v", it.QuotaCode, it.QuotaCodes)
			}
		}
	}
}
