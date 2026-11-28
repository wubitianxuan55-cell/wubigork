package app

// gaea_workcost_library_test.go — 造价数据库板块库维护绑定的绑定面端到端（2026-11-28）：
// 字典读写（含检索别名的规矩）、定额解除全部挂接、定额地区回填（先预览后落库）。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

func TestWorkcostDictBindings(t *testing.T) {
	a, _, closeDB := workcostTestAppDB(t)
	defer closeDB()

	// 新库自带种子字典（规划专业 10 个、工料机分类 34 个）：读出来非 nil。
	if got := a.GaeaWorkcostDictList(workcost.DictSpecialty); got == nil || len(got) != 10 {
		t.Fatalf("新库的专业字典应有 10 个种子: %d 个", len(got))
	}
	if err := a.GaeaWorkcostDictSet(workcost.DictItem{Kind: workcost.DictSpecialty, Name: "市政工程新增"}, false); err != nil {
		t.Fatal(err)
	}
	if err := a.GaeaWorkcostDictSet(workcost.DictItem{Kind: workcost.DictSpecialty, Name: "市政工程新增"}, false); err != nil {
		t.Fatalf("重复新增应幂等: %v", err)
	}
	if err := a.GaeaWorkcostDictSet(workcost.DictItem{Kind: workcost.DictAlias, Name: "铲车", Parent: "装载机"}, false); err != nil {
		t.Fatal(err)
	}
	if err := a.GaeaWorkcostDictSet(workcost.DictItem{Kind: workcost.DictAlias, Name: "挖机"}, false); err == nil ||
		!strings.Contains(err.Error(), "标准说法") {
		t.Fatalf("检索别名缺标准说法应拒收: %v", err)
	}
	aliases := a.GaeaWorkcostDictList(workcost.DictAlias)
	if len(aliases) != 1 || aliases[0].Name != "铲车" || aliases[0].Parent != "装载机" {
		t.Fatalf("别名清单: %+v", aliases)
	}
	if got := a.GaeaWorkcostDictList(workcost.DictSpecialty); len(got) != 11 {
		t.Fatalf("新增后专业字典应为 11 个: %d", len(got))
	}
	// 没人用的字典值能删
	if err := a.GaeaWorkcostDictSet(workcost.DictItem{Kind: workcost.DictAlias, Name: "铲车"}, true); err != nil {
		t.Fatalf("别名应可删: %v", err)
	}
	if got := a.GaeaWorkcostDictList(workcost.DictAlias); len(got) != 0 {
		t.Fatalf("删后不该还有: %+v", got)
	}
	// 仍被数据引用的拒删
	if _, err := a.hubWorkcostStore().SaveQuota(workcost.Quota{Code: "Q1", Title: "装载机装车", Specialty: "市政工程新增", Unit: "m3"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.GaeaWorkcostDictSet(workcost.DictItem{Kind: workcost.DictSpecialty, Name: "市政工程新增"}, true); err == nil ||
		!strings.Contains(err.Error(), "仍被") {
		t.Fatalf("仍被引用的专业应拒删: %v", err)
	}
}

func TestWorkcostQuotaDetachEverywhereBinding(t *testing.T) {
	a, _, closeDB := workcostTestAppDB(t)
	defer closeDB()
	s := a.hubWorkcostStore()
	if err := s.DictAdd(workcost.DictItem{Kind: workcost.DictSpecialty, Name: "市政"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveQuota(workcost.Quota{Code: "Q1", Title: "装载机装车", Specialty: "市政", Unit: "m3"}, nil); err != nil {
		t.Fatal(err)
	}
	pid, err := s.UpsertBillProject("测算", "", "", "", "", "", nil, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"B1", "B2"} {
		if _, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: code, Title: code, Unit: "m3", Quantity: 1, QuotaCode: "Q1"}); err != nil {
			t.Fatal(err)
		}
	}
	// 仍被引用：删除被拒；解除全部挂接后可删
	if err := a.GaeaWorkcostQuotaDelete("Q1"); err == nil {
		t.Fatal("仍被清单引用的定额不该能删")
	}
	if n, err := a.GaeaWorkcostQuotaDetachEverywhere("Q1"); err != nil || n != 2 {
		t.Fatalf("解除全部挂接应影响 2 条清单行: n=%d err=%v", n, err)
	}
	if err := a.GaeaWorkcostQuotaDelete("Q1"); err != nil {
		t.Fatalf("解除挂接后应可删: %v", err)
	}
}

func TestWorkcostQuotaRegionBackfillBinding(t *testing.T) {
	a, _, closeDB := workcostTestAppDB(t)
	defer closeDB()
	s := a.hubWorkcostStore()
	for _, r := range []string{"四川省", "重庆市"} {
		if err := s.DictAdd(workcost.DictItem{Kind: workcost.DictRegion, Name: r}); err != nil {
			t.Fatal(err)
		}
	}
	pid, err := s.UpsertBillProject("四川项目", "", "", "", "", "", nil, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateBillProjectProfile(pid, "", "四川省", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveQuota(workcost.Quota{Code: "Q1", Title: "定额一", Unit: "m3"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "B1", Title: "清单一", Unit: "m3", Quantity: 1, QuotaCode: "Q1"}); err != nil {
		t.Fatal(err)
	}

	// 预览：不写库
	plan, err := a.GaeaWorkcostQuotaRegionBackfill(true)
	if err != nil || len(plan) != 1 || plan[0].Code != "Q1" || plan[0].Region != "四川省" || plan[0].Applied {
		t.Fatalf("预览方案: %+v err=%v", plan, err)
	}
	if q, _ := s.GetQuota("Q1"); q == nil || q.Region != "" {
		t.Fatalf("预览不该写库: %+v", q)
	}
	// 落库
	if plan, err = a.GaeaWorkcostQuotaRegionBackfill(false); err != nil || len(plan) != 1 || !plan[0].Applied {
		t.Fatalf("落库方案: %+v err=%v", plan, err)
	}
	if q, _ := s.GetQuota("Q1"); q == nil || q.Region != "四川省" {
		t.Fatalf("落库后地区应为四川省: %+v", q)
	}
	// 没有待回填的：空库结果也是非 nil
	if plan, err = a.GaeaWorkcostQuotaRegionBackfill(true); err != nil || plan == nil || len(plan) != 0 {
		t.Fatalf("没有待回填的应为非 nil 空切片: %#v err=%v", plan, err)
	}
}
