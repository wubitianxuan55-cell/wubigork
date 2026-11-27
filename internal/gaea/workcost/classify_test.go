package workcost

// classify_test.go — 导入链按规矩落库 / 整理归类 / 借鉴生成 / 专业一致性机检 /
// 飞轮专业内聚合 / 体检第六类 的用例（2026-11-25 库定规矩）。夹具一律手搭
// ProjectBundle，不依赖真实工作簿。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// sampleBundle 手搭一份五表 bundle：两条有价工料机、一条清单项两条消耗行。
func sampleBundle(project, file, specialty, region, period string, laborPrice float64) *ProjectBundle {
	return &ProjectBundle{
		Project: project, FileName: file, Specialty: specialty, Region: region, Period: period,
		Resources: []ProjectResource{
			{Code: "L01", Kind: "人工", Title: "普工", Unit: "工日", Price: laborPrice},
			{Code: "C20", Kind: "材料", Title: "商品混凝土", Spec: "C20", Unit: "m³", Price: 430},
		},
		Items: []ProjectItem{{Code: "WP01", Title: "垫层", Unit: "m³", Quantity: 100, Feature: "C20 垫层"}},
		Lines: []ProjectConsumption{
			{ItemCode: "WP01", ItemTitle: "垫层", ResourceCode: "L01", Quantity: 0.5},
			{ItemCode: "WP01", ItemTitle: "垫层", ResourceCode: "C20", Quantity: 1.02},
		},
	}
}

// TestImportScopeRejectsNonConforming 导入口径是规范值：字典外的专业/地区、认不出的
// 计价期，整份拒收，不带病入库。
func TestImportScopeRejectsNonConforming(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	for i, b := range []*ProjectBundle{{Specialty: "不存在专业"}, {Region: "不存在地区"}, {Period: "2026年第7期"}} {
		b.Project, b.FileName = "口径测试", "k.xlsx"
		if res := s.ApplyProjectBundle(b); len(res.Errors) == 0 {
			t.Errorf("用例 %d：不合规矩的口径应整份拒收", i)
		}
	}
	if projects, _ := s.BillProjects(); len(projects) != 0 {
		t.Fatalf("拒收后不应留下项目，得到 %d 个", len(projects))
	}
	if _, _, period, err := s.ImportScope(&ProjectBundle{Period: "2026年6月"}); err != nil || period != "2026-06" {
		t.Fatalf("「2026年6月」应归一为 2026-06，得到 %q err=%v", period, err)
	}
}

// TestApplyProjectBundle_RegionsFactsAndCodeAlias 工料机按地区分行 + 价格事实 + 撞码消歧：
//   - 文件里类别不在四类内的工料机行被拒收并点名；
//   - 同一份表按另一个地区再导：工料机另立该地区的行（编码被占则消歧成「码-N」），
//     含量行按别名落到新码上不悬空，原地区的价不被顶掉；
//   - 每条有价的工料机记一条价格事实（期 + 地区），同簿同期重导幂等。
func TestApplyProjectBundle_RegionsFactsAndCodeAlias(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复")
	seedDict(t, s, DictRegion, "四川省", "重庆市")

	b := sampleBundle("测试项目", "t.xlsx", "土壤修复", "四川省", "2026-06", 180)
	b.Resources = append(b.Resources, ProjectResource{Code: "BAD", Kind: "一段备注文字", Title: "脏类别", Unit: "项", Price: 1})
	res := s.ApplyProjectBundle(b)
	if res.ResourceNew != 2 || len(res.Errors) != 1 || !strings.Contains(res.Errors[0], "已拒收") {
		t.Fatalf("两个合规工料机入库、脏类别行拒收点名，得到 %+v", res)
	}
	if res.PriceFacts != 2 {
		t.Fatalf("应记 2 条价格事实，得到 %d", res.PriceFacts)
	}
	projects, _ := s.BillProjects()
	if len(projects) != 1 || projects[0].Specialty != "土壤修复" || projects[0].Region != "四川省" || projects[0].PricePeriod != "2026-06" {
		t.Fatalf("项目口径应随导入落库，得到 %+v", projects)
	}
	items, _ := s.BillItems(projects[0].ID)
	if len(items) != 1 || items[0].Specialty != "土壤修复" {
		t.Fatalf("清单行专业应跟随项目默认专业，得到 %+v", items)
	}
	if len(items[0].LinkQuantities) != 1 || items[0].LinkQuantities["WP01"] != 100 {
		t.Fatalf("清单行应带挂接明细（定额码 → 工程量），得到 %+v", items[0].LinkQuantities)
	}
	q, _ := s.GetQuota("WP01")
	if q.Specialty != "土壤修复" || q.Chapter != "" || len(q.Items) != 2 {
		t.Fatalf("新定额专业应随项目专业、章节留空待整理，得到 %+v", q)
	}
	l01, _ := s.GetResourceByCode("L01")
	facts, _ := s.ResourcePrices(l01.ID)
	if len(facts) != 1 || facts[0].Price != 180 || facts[0].Period != "2026-06" || facts[0].Region != "四川省" {
		t.Fatalf("价格事实应记 价 + 期 + 地区，得到 %+v", facts)
	}

	// 同一份表按重庆再导：工料机另立重庆的行，编码消歧，含量行跟着改指。
	cq := sampleBundle("测试项目", "t.xlsx", "土壤修复", "重庆市", "2026-07", 200)
	res = s.ApplyProjectBundle(cq)
	if res.ResourceNew != 2 || res.ResourceUpd != 0 || res.ResourceRenamed != 2 {
		t.Fatalf("重庆再导应新立 2 行、消歧 2 个码，得到 %+v", res)
	}
	cqL01, err := s.GetResourceByCode("L01-2")
	if err != nil || cqL01.Region != "重庆市" || cqL01.CurrentPrice != 200 {
		t.Fatalf("重庆的人工应落在 L01-2，得到 %+v err=%v", cqL01, err)
	}
	if cd, _ := s.GetResourceByCode("L01"); cd.CurrentPrice != 180 || cd.Region != "四川省" {
		t.Fatalf("成都的价不应被重庆的导入顶掉，得到 %+v", cd)
	}
	q, _ = s.GetQuota("WP01")
	for _, it := range q.Items {
		if it.ResourceCode == "L01" || it.ResourceCode == "C20" {
			t.Fatalf("含量行应按别名落到重庆的码上，仍在指成都的 %s", it.ResourceCode)
		}
	}
	hl, _ := s.HealthScan()
	if len(hl.BrokenRef) != 0 {
		t.Fatalf("换码后不应有悬挂引用，得到 %+v", hl.BrokenRef)
	}
	facts, _ = s.ResourcePrices(cqL01.ID)
	if len(facts) != 1 || facts[0].Period != "2026-07" || facts[0].Region != "重庆市" {
		t.Fatalf("重庆行的价格事实应记 2026-07 / 重庆市，得到 %+v", facts)
	}
	// 同簿同期重导：价格事实不堆重复行。
	total := func() int {
		var n int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM gf_resource_prices`).Scan(&n)
		return n
	}
	before := total()
	s.ApplyProjectBundle(cq)
	if after := total(); after != before {
		t.Fatalf("同簿同期重导不应堆价格事实：%d → %d", before, after)
	}
}

// TestApplyProjectBundle_SpecialtyFollowsSource 定额专业随来源定、跨专业不并：
//   - 外专业同码同内容另立新定额并记借鉴来源；
//   - 同专业同内容复用同一条；
//   - 归过类的清单专业不被重导入冲掉；
//   - 同项目重导入时项目专业改了，它自己的定额跟着改。
func TestApplyProjectBundle_SpecialtyFollowsSource(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复", "市政")

	if res := s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "土壤修复", "", "", 180)); res.QuotaNew != 1 || res.QuotaDerived != 0 {
		t.Fatalf("首导应新建 1 条定额，得到 %+v", res)
	}
	// 别的项目、别的专业、同码同内容：不并，另立并记借鉴。
	res := s.ApplyProjectBundle(sampleBundle("市政项目", "b.xlsx", "市政", "", "", 180))
	if res.QuotaNew != 1 || res.QuotaDerived != 1 {
		t.Fatalf("外专业同码同内容应另立 1 条并计入借鉴，得到 %+v", res)
	}
	bq, _ := s.GetQuota("WP01-2")
	if bq.Specialty != "市政" || bq.DerivedFrom != "借鉴 土壤修复 WP01" {
		t.Fatalf("另立的定额应是市政专业并记借鉴来源，得到 %+v", bq)
	}
	projects, _ := s.BillProjects()
	for _, p := range projects {
		items, _ := s.BillItems(p.ID)
		want := "WP01"
		if p.Name == "市政项目" {
			want = "WP01-2"
		}
		if len(items) != 1 || items[0].QuotaCode != want {
			t.Fatalf("%s 的清单应挂 %s，得到 %+v", p.Name, want, items)
		}
	}
	// 第三个土壤修复项目、同内容：复用 WP01，不再另立。
	if res := s.ApplyProjectBundle(sampleBundle("修复项目二", "c.xlsx", "土壤修复", "", "", 180)); res.QuotaNew != 0 || res.QuotaUpd != 1 {
		t.Fatalf("同专业同内容应复用，得到 %+v", res)
	}

	// 「修复项目」重导入并把项目专业纠正为市政：它自己的定额 WP01 跟着改；
	// 借鉴出来的 WP01-2 是市政项目的，不动。
	s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "市政", "", "", 180))
	if q, _ := s.GetQuota("WP01"); q.Specialty != "市政" {
		t.Fatalf("同项目重导入时项目专业改了，它的定额应跟着改，得到 %q", q.Specialty)
	}
	if q, _ := s.GetQuota("WP01-2"); q.Specialty != "市政" {
		t.Fatalf("别的项目的定额不应被动到，得到 %q", q.Specialty)
	}
}

// TestReimportKeepsClassifiedBillSpecialty 归过类的清单专业不被重导入冲掉。
func TestReimportKeepsClassifiedBillSpecialty(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复", "市政")
	s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "土壤修复", "", "", 180))
	projects, _ := s.BillProjects()
	items, _ := s.BillItems(projects[0].ID)
	if _, err := s.ApplyClassify(ClassifyPlan{Bills: []ClassifyBill{{ID: items[0].ID, Specialty: "市政"}}}); err != nil {
		t.Fatal(err)
	}
	s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "土壤修复", "", "", 180))
	got, _ := s.BillItems(projects[0].ID)
	if len(got) != 1 || got[0].Specialty != "市政" {
		t.Fatalf("重导入不应冲掉已归类的清单专业，得到 %+v", got)
	}
}

// TestClassify 整理归类：演练不写库；落库按目标值汇总；字典外的值与不存在的条目进
// 冲突清单，不执行。
func TestClassify(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复")
	seedDict(t, s, DictRegion, "四川省")
	seedDict(t, s, DictCategory, "人工", "材料/混凝土")

	s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "", "", "", 180)) // 专业留空=待整理
	projects, _ := s.BillProjects()
	pid := projects[0].ID
	items, _ := s.BillItems(pid)
	l01, _ := s.GetResourceByCode("L01")
	c20, _ := s.GetResourceByCode("C20")

	plan := ClassifyPlan{
		Projects: []ClassifyProject{{ID: pid, Specialty: "土壤修复", Region: "四川省", PricePeriod: "2026年6月"}},
		Bills:    []ClassifyBill{{ID: items[0].ID, Division: "垫层工程"}, {ID: 999999, Specialty: "土壤修复"}},
		Quotas:   []ClassifyQuota{{Code: "WP01", Specialty: "土壤修复", Chapter: "混凝土工程"}, {Code: "NOPE", Chapter: "x"}},
		Resources: []ClassifyResource{
			{Code: "L01", CategoryPath: "人工", Region: "不在字典的地方"},
			{Code: "C20", CategoryPath: "材料/混凝土", Region: "四川省", Period: "2026年6月"},
			{Code: "MISSING", Region: "四川省"},
		},
	}
	// 演练：校验与冲突照出，一行不写。
	preview, err := s.PreviewClassify(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Conflicts) != 4 {
		t.Fatalf("演练应报 4 条冲突（清单行不存在 / 定额不存在 / L01 地区不在字典 / 工料机不存在），得到 %+v", preview.Conflicts)
	}
	if q, _ := s.GetQuota("WP01"); q.Specialty != "" || q.Chapter != "" {
		t.Fatal("演练不应写库")
	}
	if p := projects[0]; p.Specialty != "" {
		t.Fatal("演练不应改项目口径")
	}
	// 落库。
	done, err := s.ApplyClassify(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Conflicts) != 4 {
		t.Fatalf("落库时同样 4 条冲突不执行，得到 %+v", done.Conflicts)
	}
	// 工料机逐字段生效：L01 的地区不在字典被拦，但同一行合法的分类照常写入——两条工料机都算改动。
	if done.Applied.Projects != 1 || done.Applied.Bills != 1 || done.Applied.Quotas != 1 || done.Applied.Resources != 2 {
		t.Fatalf("改动条数（按条目计）不符: %+v", done.Applied)
	}
	summary := map[string]int{}
	for _, sm := range done.Summary {
		summary[sm.Dimension+"="+sm.Value] += sm.Count
	}
	for _, k := range []string{"项目专业=土壤修复", "项目地区=四川省", "项目计价期=2026-06", "清单分部=垫层工程",
		"定额专业=土壤修复", "定额分部·土壤修复=混凝土工程", "工料机分类=材料/混凝土", "工料机分类=人工", "工料机地区=四川省", "工料机期数=2026-06"} {
		if summary[k] != 1 {
			t.Errorf("方案里应有「%s」1 条，得到 %+v", k, done.Summary)
		}
	}
	// 工料机 C20 补期时，现行价还没有价格事实的补记一条。
	if facts, _ := s.ResourcePrices(c20.ID); len(facts) != 1 || facts[0].Price != 430 || facts[0].Period != "2026-06" {
		t.Fatalf("补期时应按现行价补记价格事实，得到 %+v", facts)
	}
	if got, _ := s.BillItems(pid); len(got) != 1 || got[0].Specialty != "土壤修复" || got[0].Division != "垫层工程" {
		t.Fatalf("清单行应已归到土壤修复并整理分部，得到 %+v", got)
	}
	// 冲突是逐字段的：L01 的地区不在字典没执行，但同一行合法的分类照常写入。
	if r, _ := s.GetResource(l01.ID); r.Region != "" || r.CategoryPath != "人工" {
		t.Fatalf("L01 地区应保持未标注、分类应已写入，得到 地区=%q 分类=%q", r.Region, r.CategoryPath)
	}
	// 再跑一遍：全都已是指定值，没有需要改动的条目。
	again, _ := s.PreviewClassify(ClassifyPlan{Projects: plan.Projects})
	if len(again.Summary) != 0 {
		t.Fatalf("已是目标值的条目不应再计入方案，得到 %+v", again.Summary)
	}
}

// TestClassifyQuotaSpecialtyIsFixed 定额专业固定的判定：补空值、按引用它的清单项纠错、
// 被多个专业共用则不改。
func TestClassifyQuotaSpecialtyIsFixed(t *testing.T) {
	cases := []struct {
		name       string
		cur        string
		refs       []string
		next       string
		wantOK     bool
		wantReason string
	}{
		{"未标注、无人引用 → 可标", "", nil, "市政", true, ""},
		{"未标注、引用它的都是市政 → 可标市政", "", []string{"市政"}, "市政", true, ""},
		{"未标注、引用它的都是市政 → 标土壤修复不行", "", []string{"市政"}, "土壤修复", false, "要标就标「市政」"},
		{"未标注、被两个专业引用 → 不标", "", []string{"市政", "土壤修复"}, "市政", false, "多个专业"},
		{"已标土壤修复、引用它的全是市政 → 纠错为市政", "土壤修复", []string{"市政"}, "市政", true, ""},
		{"已标土壤修复、被两个专业共用 → 不改", "土壤修复", []string{"市政", "土壤修复"}, "市政", false, "保持原专业"},
		{"已标土壤修复、无人引用 → 来源不明不改", "土壤修复", nil, "市政", false, "来源不明"},
		{"已标土壤修复、引用它的是市政、要标地下水修复 → 不改", "土壤修复", []string{"市政"}, "地下水修复", false, "borrow_quota"},
	}
	for _, c := range cases {
		ok, reason := quotaSpecialtyChange(c.cur, c.next, "Q", c.refs)
		if ok != c.wantOK || (!ok && !strings.Contains(reason, c.wantReason)) {
			t.Errorf("%s：得到 (%v, %q)", c.name, ok, reason)
		}
	}
	// 落库链路：被两个专业共用的定额，classify 进冲突清单、库里没动。
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复", "市政")
	s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "土壤修复", "", "", 180))
	pid2, _ := s.UpsertBillProject("市政项目", "b.xlsx", "", "", "", "", nil, false, 0)
	if err := s.UpdateBillProjectProfile(pid2, "市政", "", ""); err != nil {
		t.Fatal(err)
	}
	it, err := s.SaveBillItem(BillItem{ProjectID: pid2, Code: "M001", Title: "垫层", Unit: "m³", Quantity: 1, QuotaCode: "WP01"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AttachBillQuotaLink(it.ID, "WP01", 1); err != nil {
		t.Fatal(err)
	}
	res, err := s.ApplyClassify(ClassifyPlan{Quotas: []ClassifyQuota{{Code: "WP01", Specialty: "市政"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 || res.Applied.Quotas != 0 {
		t.Fatalf("被两个专业共用的定额应进冲突清单，得到 %+v", res)
	}
	if q, _ := s.GetQuota("WP01"); q.Specialty != "土壤修复" {
		t.Fatalf("定额专业不应被改，得到 %q", q.Specialty)
	}
}

// TestBorrowQuota 借鉴生成：以外专业定额为底生成本专业新定额（含量行照搬、溯源记
// 借鉴），可把清单行改挂过去；幂等复用；同专业不能借鉴；演练不写库。
func TestBorrowQuota(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复", "市政")
	s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "土壤修复", "", "", 180))
	projects, _ := s.BillProjects()
	items, _ := s.BillItems(projects[0].ID)

	if _, err := s.BorrowQuota(BorrowRequest{FromCode: "WP01", Specialty: "土壤修复"}, true); err == nil ||
		!strings.Contains(err.Error(), "同专业") {
		t.Fatalf("同专业不应借鉴，err=%v", err)
	}
	if _, err := s.BorrowQuota(BorrowRequest{FromCode: "WP01", Specialty: "不在字典的专业"}, true); err == nil ||
		!strings.Contains(err.Error(), "不在专业字典里") {
		t.Fatalf("目标专业必须在字典里，err=%v", err)
	}

	preview, err := s.BorrowQuota(BorrowRequest{FromCode: "WP01", Specialty: "市政", ItemIDs: []int64{items[0].ID, 999999}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if preview.New.Code != "WP01-2" || len(preview.New.Items) != 2 || len(preview.Relinked) != 1 || len(preview.Conflicts) != 1 {
		t.Fatalf("演练结果不符: %+v", preview)
	}
	if q, _ := s.GetQuota("WP01-2"); q != nil {
		t.Fatal("演练不应写库")
	}
	if got, _ := s.BillItems(projects[0].ID); got[0].QuotaCode != "WP01" {
		t.Fatal("演练不应改挂")
	}

	done, err := s.BorrowQuota(BorrowRequest{FromCode: "WP01", Specialty: "市政", ItemIDs: []int64{items[0].ID}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if done.Existed {
		t.Fatal("首次生成不应标记为复用")
	}
	nq, _ := s.GetQuota("WP01-2")
	if nq.Specialty != "市政" || nq.DerivedFrom != "借鉴 土壤修复 WP01" || len(nq.Items) != 2 || nq.Title != "垫层" {
		t.Fatalf("生成的本专业定额不符: %+v", nq)
	}
	if nq.Items[0].ResourceCode != "L01" || nq.Items[1].ResourceCode != "C20" {
		t.Fatalf("含量行应照搬资源码，得到 %+v", nq.Items)
	}
	got, _ := s.BillItems(projects[0].ID)
	if len(got) != 1 || got[0].QuotaCode != "WP01-2" || len(got[0].QuotaCodes) != 1 || got[0].QuotaCodes[0] != "WP01-2" {
		t.Fatalf("清单行应已改挂到新定额，得到 %+v", got)
	}
	again, err := s.BorrowQuota(BorrowRequest{FromCode: "WP01", Specialty: "市政"}, true)
	if err != nil || !again.Existed || again.New.Code != "WP01-2" {
		t.Fatalf("同源借鉴应复用既有定额，得到 %+v err=%v", again, err)
	}
}

// TestAuditSpecialtyGaps 清单与定额专业对不上的两条机检：清单挂着外专业定额；国标
// 清单编码的专业段（专业字典项带标准代码时）与清单专业不一致。
func TestAuditSpecialtyGaps(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复")
	if err := s.DictAdd(DictItem{Kind: DictSpecialty, Name: "市政", Code: "04"}); err != nil {
		t.Fatal(err)
	}
	s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "土壤修复", "", "", 180))
	projects, _ := s.BillProjects()
	pid := projects[0].ID
	s.BorrowQuota(BorrowRequest{FromCode: "WP01", Specialty: "市政"}, true)

	// 把清单行改挂到市政定额上，并把编码改成市政国标码。
	if _, err := s.db.Exec(`UPDATE gf_bill_items SET quota_code='WP01-2', code='040101001' WHERE project_id=?`, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE gf_bill_quota_links SET quota_code='WP01-2' WHERE bill_item_id IN (SELECT id FROM gf_bill_items WHERE project_id=?)`, pid); err != nil {
		t.Fatal(err)
	}
	rep, err := s.BillAuditScan(pid)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, f := range rep.SpecialtyGaps {
		kinds[f.Kind] = true
	}
	if !kinds["specialty_gap"] || !kinds["code_gap"] {
		t.Fatalf("应同时报清单挂了外专业定额与编码专业段不一致，得到 %+v", rep.SpecialtyGaps)
	}
	// 项目专业改成市政后，清单与定额、编码专业段都一致，两项都消失。
	if _, err := s.ApplyClassify(ClassifyPlan{Projects: []ClassifyProject{{ID: pid, Specialty: "市政"}}}); err != nil {
		t.Fatal(err)
	}
	rep, err = s.BillAuditScan(pid)
	if err != nil || len(rep.SpecialtyGaps) != 0 {
		t.Fatalf("不应再报，得到 %+v err=%v", rep.SpecialtyGaps, err)
	}
}

// TestAuditGapsStayWithinSpecialty 跨项目查漏只在同专业间比：别的专业同行的分部不算
// 「多数同行都有」；定额缺组成也只在同专业同名组里比。
func TestAuditGapsStayWithinSpecialty(t *testing.T) {
	projects := []auditProject{
		{ID: 1, Name: "修复一", FileName: "a.xlsx", ItemCount: 5, Specialty: "土壤修复", Divisions: map[string]int{"临建": 3, "监测": 2}},
		{ID: 2, Name: "修复二", FileName: "b.xlsx", ItemCount: 5, Specialty: "土壤修复", Divisions: map[string]int{"临建": 3}},
		{ID: 3, Name: "修复三", FileName: "c.xlsx", ItemCount: 5, Specialty: "土壤修复", Divisions: map[string]int{"临建": 2, "监测": 3}},
		{ID: 4, Name: "市政一", FileName: "d.xlsx", ItemCount: 5, Specialty: "市政", Divisions: map[string]int{"道路面层": 5}},
		{ID: 5, Name: "市政二", FileName: "e.xlsx", ItemCount: 5, Specialty: "市政", Divisions: map[string]int{"道路面层": 5}},
	}
	// 被审的修复二缺「监测」：同专业的另两个项目都有（2/2，100%）→ 报。
	gaps := AuditBillGaps(projects, 2)
	if len(gaps) != 1 || gaps[0].Target != "监测" {
		t.Fatalf("同专业的同行都有的分部缺失应报，得到 %+v", gaps)
	}
	for _, g := range gaps {
		if g.Target == "道路面层" {
			t.Fatal("别的专业的分部不应进入比对")
		}
	}
	// 同专业同码族缺组成才比；跨专业同名（同族码）不比（2026-11-27 复盘 P11 起
	// 分组键=专业+同码族，见 audit.go quotaFamily）。
	quotas := []auditQuota{
		{Code: "A", Title: "垫层", Unit: "m³", Specialty: "土壤修复", Resource: map[string]string{"C20": "混凝土", "L01": "普工"}},
		{Code: "A-2", Title: "垫层", Unit: "m³", Specialty: "土壤修复", Resource: map[string]string{"C20": "混凝土", "L01": "普工"}},
		{Code: "A-3", Title: "垫层", Unit: "m³", Specialty: "土壤修复", Resource: map[string]string{"C20": "混凝土"}},
		{Code: "A", Title: "垫层", Unit: "m³", Specialty: "市政", Resource: map[string]string{}},
	}
	qg := AuditQuotaGaps(quotas)
	for _, g := range qg {
		if !strings.Contains(g.Detail, "A-3") {
			t.Fatalf("别的专业的同名定额不应进入比对，得到 %+v", g)
		}
	}
	if len(qg) == 0 {
		t.Fatal("同专业组里缺组成的定额应报")
	}
}

// TestFlywheelScopesBySpecialty 飞轮只在专业内聚合：同词干、不同专业的定额不聚；
// 同专业两条才成候选且带专业；生成的企业定额带专业，同词干多专业时须指明。
func TestFlywheelScopesBySpecialty(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复", "市政")
	if _, err := s.SaveResource(Resource{Kind: KindLabor, Title: "普工", Unit: "工日", Code: "L01"}); err != nil {
		t.Fatal(err)
	}
	save := func(code, specialty, src string, qty float64) {
		if _, err := s.SaveQuota(Quota{Code: code, OrigCode: "WP-100", Title: "施工便道", Unit: "m", Specialty: specialty, Source: src,
			Items: []QuotaItem{{ResourceCode: "L01", Quantity: qty}}}, nil); err != nil {
			t.Fatal(err)
		}
	}
	save("WP-100", "土壤修复", "项目导入：A.xlsx", 1)
	save("WP-100-2", "市政", "项目导入：B.xlsx", 2)
	quotas, _ := s.allQuotasWithItems()
	if prev := BuildQuotaFlywheelPreview(quotas); len(prev.Candidates) != 0 {
		t.Fatalf("跨专业的同词干定额不应聚成候选，得到 %+v", prev.Candidates)
	}
	save("WP-100-3", "市政", "项目导入：C.xlsx", 3)
	quotas, _ = s.allQuotasWithItems()
	prev := BuildQuotaFlywheelPreview(quotas)
	if len(prev.Candidates) != 1 || prev.Candidates[0].Specialty != "市政" {
		t.Fatalf("同专业两条含量相异应成一个（市政）候选，得到 %+v", prev.Candidates)
	}
	res, err := s.QuotaFlywheelApply([]string{"WP-100"}, "")
	if err != nil || len(res.Saved) != 1 {
		t.Fatalf("只有一个专业的候选时可直接生成，得到 %+v err=%v", res, err)
	}
	if ent, _ := s.GetQuota("ENT-WP-100"); ent.Specialty != "市政" {
		t.Fatalf("企业定额应带专业，得到 %q", ent.Specialty)
	}
	// 再加一条土壤修复的变体：同词干两个专业各一个候选，不指明专业不猜。
	save("WP-100-4", "土壤修复", "项目导入：D.xlsx", 4)
	res, err = s.QuotaFlywheelApply([]string{"WP-100"}, "")
	if err != nil || len(res.Skipped) != 1 || !strings.Contains(res.Skipped[0].Reason, "带 specialty") {
		t.Fatalf("同词干多专业不带 specialty 应跳过并说明，得到 %+v err=%v", res, err)
	}
	res, err = s.QuotaFlywheelApply([]string{"WP-100"}, "土壤修复")
	if err != nil || len(res.Saved) != 1 || res.Saved[0] != "ENT-WP-100-2" {
		t.Fatalf("指明土壤修复后应生成到不撞车的 ENT-WP-100-2，得到 %+v err=%v", res, err)
	}
	if ent, _ := s.GetQuota("ENT-WP-100-2"); ent.Specialty != "土壤修复" {
		t.Fatalf("企业定额应带专业，得到 %q", ent.Specialty)
	}
	if ent, _ := s.GetQuota("ENT-WP-100"); ent.Specialty != "市政" {
		t.Fatalf("市政的企业定额不应被覆盖，得到 %q", ent.Specialty)
	}
}

// TestHealthRuleIssues 体检第六类：类别不在四类内、取值在字典外、期数不是 YYYY-MM 的
// 数据被点名；「编码未统一」只比同地区。
func TestHealthRuleIssues(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictRegion, "四川省", "重庆市")
	// 绕过存储层校验，直接摆出升级前形态的不合规数据。
	for _, q := range []string{
		`INSERT INTO gf_resources(code, kind, title, unit, price_date, tags) VALUES('X1','一段备注','脏类别','项','2026.4','[]')`,
		`INSERT INTO gf_resources(code, kind, title, unit, region, tags) VALUES('X2','材料','字典外地区','t','不在字典的地区','[]')`,
		`INSERT INTO gf_quotas(code, title, specialty, unit) VALUES('QX','字典外专业','不在字典的专业','m')`,
	} {
		if _, err := s.db.Exec(q); err != nil {
			t.Fatalf("摆不合规数据失败: %v", err)
		}
	}
	rep, err := s.HealthScan()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, r := range rep.RuleIssues {
		seen[r.Code+"."+r.Field]++
	}
	for _, k := range []string{"X1.kind", "X1.price_date", "X2.region", "QX.specialty"} {
		if seen[k] != 1 {
			t.Errorf("应点名 %s，得到 %+v", k, rep.RuleIssues)
		}
	}
	// 同名资源分布在不同地区、编码风格不同：这是按地区分行的正常形态，不报编码未统一。
	if _, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "商品混凝土", Spec: "C20", Unit: "m³", Region: "四川省", Code: "C20"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "商品混凝土", Spec: "C20", Unit: "m³", Region: "重庆市", Code: "P001"}); err != nil {
		t.Fatal(err)
	}
	rep, _ = s.HealthScan()
	for _, g := range rep.CodeMismatch {
		if g.Title == "商品混凝土" {
			t.Fatalf("跨地区的同名资源不应报编码未统一，得到 %+v", g)
		}
	}
}

// TestDims 维度盘点：字典、分布与待整理计数对得上。
func TestDims(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复")
	seedDict(t, s, DictRegion, "四川省")
	seedDict(t, s, DictCategory, "材料/混凝土")
	s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "", "", "", 180))
	s.ApplyClassify(ClassifyPlan{Resources: []ClassifyResource{{Code: "C20", CategoryPath: "材料/混凝土", Region: "四川省", Period: "2026-06"}}})
	d, err := s.Dims()
	if err != nil {
		t.Fatal(err)
	}
	// 专业字典预设了规划专业表（SchemaV40）；分类自 SchemaV41 起预设分类树（34 项），
	// 地区仍只收库里已有的取值。本测另加 材料/混凝土 → 分类 = 种子 + 1。
	if len(d.Dicts[DictSpecialty]) != len(db.PlannedSpecialties) ||
		len(d.Dicts[DictRegion]) != 1 ||
		len(d.Dicts[DictCategory]) != len(db.PlannedCategories)+1 {
		t.Fatalf("字典盘点不符: %+v", d.Dicts)
	}
	if d.Pending.ProjectSpecialty != 1 || d.Pending.BillSpecialty != 1 || d.Pending.QuotaSpecialty != 1 || d.Pending.QuotaChapter != 1 {
		t.Fatalf("项目/清单/定额都还没归专业与章节: %+v", d.Pending)
	}
	if d.Pending.ResourceRegion != 1 || d.Pending.ResourceCategory != 1 || d.Pending.ResourcePeriod != 1 {
		t.Fatalf("两条工料机只整理了 C20，另一条应仍待整理: %+v", d.Pending)
	}
	if len(d.Projects) != 1 || len(d.Divisions) != 0 {
		t.Fatalf("项目与分部盘点不符: %+v / %+v", d.Projects, d.Divisions)
	}
}

// TestDictCrudAndCascade 字典增改删：改名级联到引用它的数据；分类改名级联下级路径；
// 仍被引用的字典项不能删。
func TestDictCrudAndCascade(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictSpecialty, "土壤修复")
	seedDict(t, s, DictCategory, "材料", "材料/混凝土", "材料/混凝土/商品混凝土")
	s.ApplyProjectBundle(sampleBundle("修复项目", "a.xlsx", "土壤修复", "", "", 180))
	if _, err := s.SaveResource(Resource{Code: "C30", Kind: KindMaterial, Title: "商品混凝土", Spec: "C30", Unit: "m³",
		CategoryPath: "材料/混凝土/商品混凝土"}); err != nil {
		t.Fatal(err)
	}
	// 重复新增幂等。
	if err := s.DictAdd(DictItem{Kind: DictSpecialty, Name: "土壤修复"}); err != nil {
		t.Fatalf("重复新增应幂等: %v", err)
	}
	// 专业改名：级联到项目、清单行、定额。
	if _, err := s.DictUpdate(DictSpecialty, "土壤修复", "污染土壤修复", "", "", 0, "初始"); err != nil {
		t.Fatal(err)
	}
	projects, _ := s.BillProjects()
	if projects[0].Specialty != "污染土壤修复" {
		t.Fatalf("专业改名应级联到项目，得到 %q", projects[0].Specialty)
	}
	if q, _ := s.GetQuota("WP01"); q.Specialty != "污染土壤修复" {
		t.Fatalf("专业改名应级联到定额，得到 %q", q.Specialty)
	}
	// 分类改名：下级路径与工料机一起换前缀。
	if _, err := s.DictUpdate(DictCategory, "材料/混凝土", "材料/砼", "", "", 0, ""); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.DictHas(DictCategory, "材料/砼/商品混凝土"); !ok {
		t.Fatal("分类改名应级联到下级路径")
	}
	r, _ := s.GetResourceByCode("C30")
	if r.CategoryPath != "材料/砼/商品混凝土" {
		t.Fatalf("分类改名应级联到工料机，得到 %q", r.CategoryPath)
	}
	// 被引用的不能删；没被引用的能删。
	if err := s.DictDelete(DictSpecialty, "污染土壤修复"); err == nil {
		t.Fatal("仍被引用的字典项不应能删")
	}
	if err := s.DictDelete(DictCategory, "材料"); err != nil {
		t.Fatalf("未被引用的字典项应能删: %v", err)
	}
	if items, _ := s.DictList(DictCategory); len(items) != len(db.PlannedCategories)-1+2 {
		t.Fatalf("分类字典应剩 %d 项（种子 −已删材料 +本测砼两条），得到 %d 项", len(db.PlannedCategories)-1+2, len(items))
	}
}
