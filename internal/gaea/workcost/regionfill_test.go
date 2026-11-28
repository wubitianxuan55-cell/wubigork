package workcost

// regionfill_test.go — 定额地区回填：只被一个省的项目引用才写；跨省引用、项目没标地区、
// 没被引用的不动并说明原因；预览不写库；落库后不再出现在方案里。

import (
	"strings"
	"testing"
)

func TestBackfillQuotaRegions(t *testing.T) {
	s, _, cleanup := searchStore(t) // 已播种专业 市政 / 建筑工程、地区 四川省
	defer cleanup()
	seedDict(t, s, DictRegion, "重庆市")
	project := func(name, region string) int64 {
		id, err := s.UpsertBillProject(name, "", "", "", "", "", nil, false, 0)
		if err != nil {
			t.Fatalf("建项目 %s: %v", name, err)
		}
		if err := s.UpdateBillProjectProfile(id, "市政", region, ""); err != nil {
			t.Fatalf("标项目地区 %s: %v", name, err)
		}
		return id
	}
	sc, cq, noRegion := project("四川项目", "四川省"), project("重庆项目", "重庆市"), project("没标地区", "")
	for _, q := range []Quota{
		{Code: "Q-SC", Title: "只在四川用", Specialty: "市政", Unit: "m3"},
		{Code: "Q-BOTH", Title: "两个省都用", Specialty: "市政", Unit: "m3"},
		{Code: "Q-NONE", Title: "项目没标地区", Specialty: "市政", Unit: "m3"},
		{Code: "Q-UNUSED", Title: "没人用", Specialty: "市政", Unit: "m3"},
	} {
		if _, err := s.SaveQuota(q, nil); err != nil {
			t.Fatalf("存定额 %s: %v", q.Code, err)
		}
	}
	bill := func(pid int64, code, quota string) {
		t.Helper()
		if _, err := s.SaveBillItem(BillItem{ProjectID: pid, Code: code, Title: code, Unit: "m3", Quantity: 1, QuotaCode: quota}); err != nil {
			t.Fatalf("存清单行 %s: %v", code, err)
		}
	}
	bill(sc, "B1", "Q-SC")
	bill(sc, "B2", "Q-BOTH")
	bill(cq, "B1", "Q-BOTH")
	// Q-NONE 经 1:N 挂接表挂在一个没标地区的项目的清单行上
	if it, err := s.SaveBillItem(BillItem{ProjectID: noRegion, Code: "B1", Title: "手工行", Unit: "项", Quantity: 1}); err != nil {
		t.Fatal(err)
	} else if _, err := s.AttachBillQuotaLink(it.ID, "Q-NONE", 0); err != nil {
		t.Fatal(err)
	}

	regionOf := func(code string) string {
		t.Helper()
		for _, q := range s.mustListQuotas(t) {
			if q.Code == code {
				return q.Region
			}
		}
		t.Fatalf("定额 %s 不在库里", code)
		return ""
	}
	byCode := func(plan []QuotaRegionFix) map[string]QuotaRegionFix {
		out := map[string]QuotaRegionFix{}
		for _, f := range plan {
			out[f.Code] = f
		}
		return out
	}

	// 预览：给方案、不写库
	plan, err := s.BackfillQuotaRegions(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 4 {
		t.Fatalf("四条定额都还没标地区，方案应各占一行: %+v", plan)
	}
	by := byCode(plan)
	if by["Q-SC"].Region != "四川省" || by["Q-SC"].Applied || by["Q-SC"].Reason != "" || by["Q-SC"].Links != 1 {
		t.Fatalf("只被四川项目引用的应回填四川省（预览不落库）: %+v", by["Q-SC"])
	}
	if r := by["Q-BOTH"].Reason; by["Q-BOTH"].Region != "" || !strings.Contains(r, "2 个省") || !strings.Contains(r, "四川省、重庆市") {
		t.Fatalf("跨省引用的不猜: %+v", by["Q-BOTH"])
	}
	if r := by["Q-NONE"].Reason; by["Q-NONE"].Region != "" || !strings.Contains(r, "没标地区") {
		t.Fatalf("项目没标地区的不动: %+v", by["Q-NONE"])
	}
	if r := by["Q-UNUSED"].Reason; by["Q-UNUSED"].Region != "" || !strings.Contains(r, "没有被清单引用") {
		t.Fatalf("没被引用的不动: %+v", by["Q-UNUSED"])
	}
	if regionOf("Q-SC") != "" {
		t.Fatal("预览不该写库")
	}

	// 落库
	plan, err = s.BackfillQuotaRegions(false)
	if err != nil {
		t.Fatal(err)
	}
	if by = byCode(plan); !by["Q-SC"].Applied || regionOf("Q-SC") != "四川省" {
		t.Fatalf("落库后 Q-SC 应为四川省: %+v region=%q", by["Q-SC"], regionOf("Q-SC"))
	}
	for _, code := range []string{"Q-BOTH", "Q-NONE", "Q-UNUSED"} {
		if regionOf(code) != "" {
			t.Fatalf("%s 不该被写地区", code)
		}
	}
	// 再跑：已填的不在方案里
	if plan, err = s.BackfillQuotaRegions(true); err != nil || len(plan) != 3 {
		t.Fatalf("已回填的不该再出现: %d 条 err=%v", len(plan), err)
	}
}

func (s *Store) mustListQuotas(t *testing.T) []Quota {
	t.Helper()
	list, err := s.ListQuotas("", "")
	if err != nil {
		t.Fatal(err)
	}
	return list
}
