package app

// office_cost_library_rules_test.go — 库定规矩在工具层的契约（2026-11-25）：字典动作、
// 维度盘点、整理归类（演练不写库）、借鉴生成、定额专业固定、演练即校验、导入口径、
// 外专业挂接回执与机检。全部走隔离库（workcostTestApp），不碰用户正式库。

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// ruleSeed 给隔离库收进字典，并建一条引用链：工料机 ← 定额 A（土壤修复）← 项目 P
// 的清单行 WP01。返回项目 id 与清单行。
func ruleSeed(t *testing.T, s *workcost.Store) (int64, workcost.BillItem) {
	t.Helper()
	for _, d := range []workcost.DictItem{
		{Kind: workcost.DictSpecialty, Name: "土壤修复"},
		{Kind: workcost.DictSpecialty, Name: "市政"},
		{Kind: workcost.DictRegion, Name: "成都市区"},
	} {
		if err := s.DictAdd(d); err != nil {
			t.Fatalf("收进字典失败: %v", err)
		}
	}
	res, err := s.SaveResource(workcost.Resource{Kind: workcost.KindMaterial, Title: "C20商品混凝土", Unit: "m³", BasePrice: 450, CurrentPrice: 460})
	if err != nil {
		t.Fatalf("建工料机失败: %v", err)
	}
	if _, err := s.SaveQuota(workcost.Quota{Code: "A", Title: "垫层", Unit: "m³", Specialty: "土壤修复",
		Items: []workcost.QuotaItem{{ResourceCode: res.Code, Quantity: 1.02}}}, nil); err != nil {
		t.Fatalf("建定额失败: %v", err)
	}
	var rates workcost.RateSet
	pid, err := s.UpsertBillProject("测算P", "", "", "", "", "", &rates, false, 0)
	if err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	it, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "WP01", Title: "垫层浇筑", Unit: "m³", Quantity: 100, QuotaCode: "A"})
	if err != nil {
		t.Fatalf("建清单行失败: %v", err)
	}
	if _, err := s.AttachBillQuotaLink(it.ID, "A", 0); err != nil {
		t.Fatal(err)
	}
	return pid, it
}

// applyRun 跑 cost_project_apply / cost_audit 并要求成功。
func applyRun(t *testing.T, tool interface {
	Execute(context.Context, json.RawMessage) (string, error)
}, args map[string]any) string {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	out, err := tool.Execute(context.Background(), b)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	return out
}

// TestCostLibraryDict 字典动作：查看、新增（演练不写库、幂等）、分类上级必须先在、
// 改名、被引用的不能删而没被引用的能删。
func TestCostLibraryDict(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	dict := func(op string, kv map[string]any) map[string]any {
		kv["op"] = op
		return map[string]any{"action": "dict", "dict": kv}
	}

	// 专业字典预设了规划专业表（SchemaV40）；地区、工料机分类仍是空的。
	out := clMust(t, tl, map[string]any{"action": "dict"}, fmt.Sprintf("专业 %d 项", len(db.PlannedSpecialties)))
	if !strings.Contains(out, "只读") {
		t.Fatalf("不带 dict 对象应是只读查看：%s", out)
	}
	dryAdd := dict("add", map[string]any{"kind": "specialty", "name": "市政", "code": "04"})
	dryAdd["dry_run"] = true
	clMust(t, tl, dryAdd, "**将新增专业**「市政」")
	if ok, _ := s.DictHas(workcost.DictSpecialty, "市政"); ok {
		t.Fatal("演练不应写字典")
	}
	clMust(t, tl, dict("add", map[string]any{"kind": "specialty", "name": "市政", "code": "04"}), "已新增")
	clMust(t, tl, dict("add", map[string]any{"kind": "specialty", "name": "市政"}), "已在字典里")
	out = clMust(t, tl, dict("list", map[string]any{"kind": "specialty"}), "市政（代码 04）")
	if !strings.Contains(out, fmt.Sprintf("专业 %d 项", len(db.PlannedSpecialties)+1)) {
		t.Fatalf("按种类查看应只列该种类（规划专业 + 刚加的市政）：%s", out)
	}
	// 上级先存在：周转材料不在 V41 种子里 → 先加父，才能加子。材料已在种子
	// 里（重复新增幂等=已在字典里），材料/钢材 仍可正常新增。
	clMustFail(t, tl, dict("add", map[string]any{"kind": "category", "name": "周转材料/钢材"}), "上级分类「周转材料」不在字典里")
	clMust(t, tl, dict("add", map[string]any{"kind": "category", "name": "周转材料"}), "已新增")
	clMust(t, tl, dict("add", map[string]any{"kind": "category", "name": "材料"}), "已在字典里")
	clMust(t, tl, dict("add", map[string]any{"kind": "category", "name": "材料/钢材"}), "已新增")
	clMustFail(t, tl, dict("add", map[string]any{"kind": "nope", "name": "x"}), "字典新增需要 kind=")

	// 改名：没给 rename 时只改其他字段；给了 rename 才改名。
	clMust(t, tl, dict("update", map[string]any{"kind": "category", "name": "材料/钢材", "rename": "材料/钢筋"}), "已修改")
	clMust(t, tl, dict("update", map[string]any{"kind": "category", "name": "材料/钢筋", "sort": 3}), "已修改")
	clMustFail(t, tl, dict("update", map[string]any{"kind": "specialty", "name": "不存在"}), "没有")
	// 删除：仍被工料机引用的不能删；没被引用的能删。
	if _, err := s.SaveResource(workcost.Resource{Kind: workcost.KindMaterial, Title: "螺纹钢筋", Unit: "t", CategoryPath: "材料/钢筋"}); err != nil {
		t.Fatal(err)
	}
	clMust(t, tl, dict("delete", map[string]any{"kind": "category", "name": "材料/钢筋"}), "仍被 1 条数据引用")
	clMust(t, tl, dict("delete", map[string]any{"kind": "category", "name": "材料"}), "已删除")
	clMustFail(t, tl, dict("delete", map[string]any{"kind": "specialty", "name": "不存在"}), "没有")
}

// TestCostLibraryDims 维度盘点：字典、项目口径、分布与待整理计数。
func TestCostLibraryDims(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	ruleSeed(t, s)
	out := clMust(t, tl, map[string]any{"action": "dims"}, "## 造价库维度盘点（只读）")
	for _, want := range []string{fmt.Sprintf("专业 %d 项", len(db.PlannedSpecialties)+1), "地区 1 项：成都市区", "测算P（清单 1）｜专业 未标注｜地区 未标注",
		"定额专业分布", "土壤修复 1", "待整理", "工料机未标地区 1", "定额未归章节 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("盘点应含 %q：%s", want, out)
		}
	}
}

// TestCostLibraryClassify 整理归类：演练给方案不写库；正式落库；冲突点名不执行。
func TestCostLibraryClassify(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	pid, item := ruleSeed(t, s)

	classify := func(kv map[string]any) map[string]any {
		return map[string]any{"action": "classify", "classify": kv}
	}
	plan := map[string]any{
		"projects": []map[string]any{{"id": pid, "specialty": "土壤修复", "region": "成都市区", "price_period": "2026年6月"}},
		"bills":    []map[string]any{{"id": item.ID, "division": "垫层工程"}},
		"quotas":   []map[string]any{{"code": "A", "chapter": "混凝土工程"}, {"code": "A", "specialty": "市政"}},
		"resources": []map[string]any{
			{"code": "NOPE", "region": "成都市区"},
			{"code": c20Code(t, s), "region": "不在字典的地方", "category_path": "不在字典的分类"},
		},
	}
	args := classify(plan)
	args["dry_run"] = true
	out := clMust(t, tl, args, "演练（未写库）：整理归类")
	for _, want := range []string{"将改动", "项目专业「土壤修复」：1 条", "项目计价期「2026-06」：1 条",
		"定额章节·土壤修复「混凝土工程」：1 条", "未执行 4 条", "[quota] A「垫层」 specialty → 市政", "borrow_quota",
		"[resource] NOPE「」：工料机不存在", "region → 不在字典的地方", "category_path → 不在字典的分类", "不在地区字典里", "不在工料机分类字典里"} {
		if !strings.Contains(out, want) {
			t.Errorf("演练回执应含 %q：%s", want, out)
		}
	}
	if p := findBillProject(s, pid); p.Specialty != "" || p.Region != "" {
		t.Fatal("演练不应写库")
	}
	if q, _ := s.GetQuota("A"); q.Chapter != "" {
		t.Fatal("演练不应写定额章节")
	}
	out = clMust(t, tl, classify(plan), "已完成：整理归类")
	if !strings.Contains(out, "未执行 4 条") {
		t.Fatalf("正式执行的回执同样要点名冲突：%s", out)
	}
	p := findBillProject(s, pid)
	if p.Specialty != "土壤修复" || p.Region != "成都市区" || p.PricePeriod != "2026-06" {
		t.Fatalf("项目口径应已落库，得到 %+v", p)
	}
	// 定额 A 只被土壤修复专业的清单引用，要改成市政 → 冲突不改；章节照归。
	if q, _ := s.GetQuota("A"); q.Chapter != "混凝土工程" || q.Specialty != "土壤修复" {
		t.Fatalf("章节应已归、专业不应被改，得到 %+v", q)
	}
	clMustFail(t, tl, map[string]any{"action": "classify"}, "需要 classify 对象")
	clMustFail(t, tl, classify(map[string]any{}), "四组都是空的")
	// 字典外的值不报错，进冲突清单。
	out = clMust(t, tl, classify(map[string]any{"projects": []map[string]any{{"id": pid, "specialty": "不在字典的专业"}}}), "未执行 1 条")
	if !strings.Contains(out, "不在专业字典里") {
		t.Fatalf("冲突原因应道出值不在字典里：%s", out)
	}
}

// TestCostLibraryBorrowQuota 借鉴生成：演练不写库；正式生成并改挂；幂等复用；同专业拒。
func TestCostLibraryBorrowQuota(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	pid, item := ruleSeed(t, s)

	clMustFail(t, tl, map[string]any{"action": "borrow_quota", "from_code": "A", "specialty": "土壤修复"}, "同专业")
	clMustFail(t, tl, map[string]any{"action": "borrow_quota", "from_code": "A"}, "需要 from_code")
	clMustFail(t, tl, map[string]any{"action": "borrow_quota", "from_code": "A", "specialty": "不在字典的专业"}, "不在专业字典里")
	out := clMust(t, tl, map[string]any{"action": "borrow_quota", "from_code": "A", "specialty": "市政",
		"item_ids": []int64{item.ID, 999999}, "dry_run": true}, "**将生成本专业定额** [A-2]「垫层」（市政专业）")
	for _, want := range []string{"溯源：借鉴 土壤修复 A", "将改挂清单行 1 条", "未改挂 1 条", "清单行 #999999 不存在"} {
		if !strings.Contains(out, want) {
			t.Errorf("演练回执应含 %q：%s", want, out)
		}
	}
	if q, _ := s.GetQuota("A-2"); q != nil {
		t.Fatal("演练不应写库")
	}
	clMust(t, tl, map[string]any{"action": "borrow_quota", "from_code": "A", "specialty": "市政", "item_ids": []int64{item.ID}}, "已生成本专业定额")
	got, _ := s.BillItems(pid)
	if len(got) != 1 || got[0].QuotaCode != "A-2" {
		t.Fatalf("清单行应已改挂到借鉴生成的定额，得到 %+v", got)
	}
	clMust(t, tl, map[string]any{"action": "borrow_quota", "from_code": "A", "specialty": "市政"}, "复用既有的本专业定额")
}

// TestCostLibraryQuotaSpecialtyIsFixed 定额专业固定：已有定额 save_quota 改不了专业，
// 错误指路借鉴生成；新建定额回执显形专业（未标注也显形）。
func TestCostLibraryQuotaSpecialtyIsFixed(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	ruleSeed(t, s)

	clMustFail(t, tl, map[string]any{"action": "save_quota", "quota": map[string]any{"code": "A", "specialty": "市政"}}, "专业固定不改")
	clMust(t, tl, map[string]any{"action": "save_quota", "quota": map[string]any{"code": "A", "title": "垫层"}}, "**更新定额** A「垫层」（m³）｜专业 土壤修复")
	clMustFail(t, tl, map[string]any{"action": "save_quota", "quota": map[string]any{"code": "Q9", "title": "x", "unit": "m", "specialty": "不在字典的专业",
		"items": []any{}, "dry_run": true}}, "不在专业字典里")
	clMust(t, tl, map[string]any{"action": "save_quota", "quota": map[string]any{"code": "Q9", "title": "新定额", "unit": "m",
		"specialty": "市政", "items": []any{}}}, "**新建定额** Q9「新定额」（m）｜专业 市政")
	clMust(t, tl, map[string]any{"action": "save_quota", "quota": map[string]any{"code": "Q10", "title": "待归类的定额", "unit": "m",
		"items": []any{}}}, "专业 未标注")
}

func c20Code(t *testing.T, s *workcost.Store) string {
	t.Helper()
	list, err := s.ListResources("", "C20")
	if err != nil || len(list) == 0 {
		t.Fatalf("找不到 C20 工料机: %v %+v", err, list)
	}
	return list[0].Code
}

// TestCostLibraryDryRunValidates 演练即校验：保存工料机、保存定额、调价的演练对字典外
// 的地区/专业、认不出的期数、四类之外的类别一律当场拒收，不等到正式执行。
func TestCostLibraryDryRunValidates(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	ruleSeed(t, s)
	res, _ := s.ListResources("", "C20")

	save := func(resource map[string]any) map[string]any {
		return map[string]any{"action": "save_resource", "dry_run": true, "resource": resource}
	}
	clMustFail(t, tl, save(map[string]any{"kind": "材料", "title": "新料", "unit": "t", "region": "不在字典的地方"}), "不在地区字典里")
	clMustFail(t, tl, save(map[string]any{"kind": "材料", "title": "新料", "unit": "t", "price_date": "2026年第7期"}), "不是年月写法")
	clMustFail(t, tl, save(map[string]any{"kind": "一段备注", "title": "新料", "unit": "t"}), "工料机类别须为")
	clMust(t, tl, save(map[string]any{"kind": "材料", "title": "新料", "unit": "t", "region": "成都市区", "price_date": "2026年12月"}), "演练通过")
	clMustFail(t, tl, map[string]any{"action": "set_price", "dry_run": true, "resource_code": res[0].Code, "price": 480, "period": "2026年第7期"}, "不是年月写法")
	clMustFail(t, tl, map[string]any{"action": "set_price", "dry_run": true, "resource_code": res[0].Code, "price": 480, "region": "不在字典的地方"}, "不在地区字典里")
	// 「2026年12月」能归一；演练回执显形归一后的期数。
	clMust(t, tl, map[string]any{"action": "set_price", "dry_run": true, "resource_code": res[0].Code, "price": 480, "period": "2026年12月"}, "期数 2026-12")
}

// TestCostLibraryImportProjectScope 导入口径是规范值：字典外的专业、认不出的计价期演练
// 当场拒收；口径随导入落库；新定额专业随项目；有价的工料机各记一条价格事实。
func TestCostLibraryImportProjectScope(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	fixture, err := filepath.Abs(filepath.Join("..", "gaea", "workcost", "testdata", "什邡2个污染地块施工成本测算表_补项.xlsx"))
	if err != nil || fileMissing(fixture) {
		t.Skipf("夹具不可用（%v），跳过", err)
	}
	clMustFail(t, tl, map[string]any{"action": "import_project", "dry_run": true, "path": fixture, "specialty": "不存在专业"}, "导入口径不合规矩")
	clMustFail(t, tl, map[string]any{"action": "import_project", "dry_run": true, "path": fixture, "period": "2026年第7期"}, "不是年月写法")
	clMust(t, tl, map[string]any{"action": "dict", "dict": map[string]any{"op": "add", "kind": "specialty", "name": "土壤修复"}}, "已在字典里，无需新增") // 规划专业（SchemaV40）
	clMust(t, tl, map[string]any{"action": "dict", "dict": map[string]any{"op": "add", "kind": "region", "name": "成都市区"}}, "已新增")
	clMust(t, tl, map[string]any{"action": "import_project", "dry_run": true, "path": fixture,
		"specialty": "土壤修复", "region": "成都市区", "period": "2026年6月"}, "**导入口径**: 专业 土壤修复｜地区 成都市区｜计价期 2026-06")
	if projects, _ := s.BillProjects(); len(projects) != 0 {
		t.Fatal("演练不应落库")
	}
	// 这份夹具的工料机价格表里有一行是整格同一段说明文字：不是工料机，演练点名、正式拒收。
	out := clMust(t, tl, map[string]any{"action": "import_project", "dry_run": true, "path": fixture,
		"specialty": "土壤修复", "region": "成都市区"}, "**将被拒收的工料机行")
	if !strings.Contains(out, "说明文字不是编码") {
		t.Fatalf("演练应道出那一行为什么不是工料机：%s", out)
	}
	clMust(t, tl, map[string]any{"action": "import_project", "path": fixture,
		"specialty": "土壤修复", "region": "成都市区", "period": "2026年6月"}, "**价格事实**")
	projects, _ := s.BillProjects()
	if len(projects) != 1 || projects[0].Specialty != "土壤修复" || projects[0].Region != "成都市区" || projects[0].PricePeriod != "2026-06" {
		t.Fatalf("项目口径应随导入落库，得到 %+v", projects)
	}
	quotas, _ := s.ListQuotas("", "")
	if len(quotas) == 0 {
		t.Fatal("导入后应有定额")
	}
	for _, q := range quotas {
		if q.Specialty != "土壤修复" {
			t.Fatalf("新定额的专业应随项目专业，得到 %+v", q)
		}
	}
	list, _ := s.ListResources("", "")
	priced, facts := 0, 0
	for _, r := range list {
		if ok, _ := workcost.CodeWellFormed(r.Code); !ok {
			t.Fatalf("编码不像编码的行不应入库：%.40q", r.Code)
		}
		if r.Region != "成都市区" {
			t.Fatalf("导入的工料机应带地区，得到 %+v", r)
		}
		if r.EffectivePrice() > 0 {
			priced++
			if r.PriceDate != "2026-06" {
				t.Fatalf("有价的工料机应带计价期，得到 %+v", r)
			}
		}
		ps, _ := s.ResourcePrices(r.ID)
		facts += len(ps)
	}
	if facts == 0 || facts != priced {
		t.Fatalf("每条有价的工料机应有一条价格事实：有价 %d / 事实 %d", priced, facts)
	}
}

// TestCostProjectApplySpecialty 建项目带专业/地区/计价期（规范值，字典外的当场拒收）；
// 清单行带专业；挂了外专业定额回执点名不拦；cost_audit 报清单与定额专业不一致。
func TestCostProjectApplySpecialty(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	apply := costProjectApplyTool{a: a}
	audit := costAuditTool{a: a}
	ruleSeed(t, s)
	clMust(t, tl, map[string]any{"action": "dict", "dict": map[string]any{"op": "add", "kind": "region", "name": "重庆中心城区"}}, "已新增")

	// 字典外的专业：封面不落、当场拒收。
	b, _ := json.Marshal(map[string]any{"action": "create_project", "name": "市政项目", "specialty": "不存在专业"})
	if _, err := apply.Execute(context.Background(), b); err == nil || !strings.Contains(err.Error(), "不合规矩") {
		t.Fatalf("字典外的专业应被拒收，err=%v", err)
	}
	for _, p := range mustProjects(t, s) {
		if p.Name == "市政项目" {
			t.Fatal("口径被拒时不应留下项目封面")
		}
	}
	out := applyRun(t, apply, map[string]any{"action": "create_project", "name": "市政项目",
		"specialty": "市政", "region": "重庆中心城区", "price_period": "2026年6月"})
	if !strings.Contains(out, "｜专业 市政｜地区 重庆中心城区｜计价期 2026-06") {
		t.Fatalf("建档回执应显形口径：%s", out)
	}
	var pid int64
	for _, p := range mustProjects(t, s) {
		if p.Name == "市政项目" {
			pid = p.ID
		}
	}
	// 挂了一条土壤修复的定额：回执点名，不拦。
	out = applyRun(t, apply, map[string]any{"action": "save_items", "project_id": pid, "items": []map[string]any{
		{"code": "S01", "title": "垫层", "unit": "m³", "quantity": 10, "quota_code": "A"},
	}})
	if !strings.Contains(out, "1 行挂接了别的专业的定额") || !strings.Contains(out, "borrow_quota") {
		t.Fatalf("挂了外专业定额应点名并指路：%s", out)
	}
	// 清单行自己标专业、与定额一致：不点名。
	out = applyRun(t, apply, map[string]any{"action": "save_items", "project_id": pid, "items": []map[string]any{
		{"code": "S02", "title": "垫层", "unit": "m³", "quantity": 10, "quota_code": "A", "specialty": "土壤修复"},
	}})
	if strings.Contains(out, "挂了别的专业的定额") {
		t.Fatalf("清单专业与定额专业一致时不应点名：%s", out)
	}
	got, _ := s.BillItems(pid)
	if len(got) != 2 || got[0].Specialty != "市政" || got[1].Specialty != "土壤修复" {
		t.Fatalf("清单行专业应各行其是（S02 覆盖项目默认专业），得到 %+v", got)
	}
	// 没提交专业的已有行：重发不改专业。
	applyRun(t, apply, map[string]any{"action": "save_items", "project_id": pid, "items": []map[string]any{
		{"code": "S02", "title": "垫层", "unit": "m³", "quantity": 12},
	}})
	if got, _ = s.BillItems(pid); got[1].Specialty != "土壤修复" || got[1].Quantity != 12 {
		t.Fatalf("没提交专业时应保持原值，得到 %+v", got[1])
	}
	// 机检：只报 S01 这一行。
	out = applyRun(t, audit, map[string]any{"project_id": pid})
	if n := strings.Count(out, "[INFO] specialty_gap"); n != 1 || !strings.Contains(out, "S01") {
		t.Fatalf("机检应只报挂着外专业定额的那一行：%s", out)
	}
}

func mustProjects(t *testing.T, s *workcost.Store) []workcost.BillProject {
	t.Helper()
	projects, err := s.BillProjects()
	if err != nil {
		t.Fatal(err)
	}
	return projects
}

// TestCostLibraryArgsMatchDocumentedKeys 入参结构的 JSON 键必须与工具说明里写的键一字
// 不差——错一个字母，模型按说明传的值就会被静默丢掉（实测踩到 pricePeriod /
// categoryPath 与说明里的 price_period / category_path 不一致）。逐键用说明里的写法
// 喂进去，断言值落到了对应字段。
func TestCostLibraryArgsMatchDocumentedKeys(t *testing.T) {
	schema := costLibraryDocumentedSchema()
	documented := map[string]string{
		// 入参键 → 说明文本里应出现的同名键（防说明与结构各自漂移）。嵌套
		// dict 对象（op/kind/name…）不再对模型说明（增改删走 dict_add/
		// dict_update/dict_delete 扁平动作），下面的解码核对留作兼容路径。
		"classify": `"classify"`, "projects": `"projects"`, "bills": `"bills"`, "quotas": `"quotas"`, "resources": `"resources"`,
		"price_period": `"price_period"`, "category_path": `"category_path"`, "division": `"division"`, "chapter": `"chapter"`, "period": `"period"`,
		"kind": `"kind"`, "name": `"name"`, "rename": `"rename"`, "code": `"code"`, "parent": `"parent"`, "sort": `"sort"`, "note": `"note"`,
		"specialty": `"specialty"`, "region": `"region"`, "from_code": `"from_code"`, "new_code": `"new_code"`, "new_title": `"new_title"`, "item_ids": `"item_ids"`,
	}
	raw := string(schema)
	for _, needle := range documented {
		if !strings.Contains(raw, needle) {
			t.Errorf("工具说明里应有键 %s", needle)
		}
	}
	var p costLibraryArgs
	if err := json.Unmarshal([]byte(`{
		"action":"classify","specialty":"市政","region":"成都市区","period":"2026-06","from_code":"A","new_code":"A-9","new_title":"垫层","item_ids":[7],
		"classify":{
			"projects":[{"id":1,"specialty":"土壤修复","region":"成都市区","price_period":"2026-06"}],
			"bills":[{"id":2,"specialty":"市政","division":"垫层工程"}],
			"quotas":[{"code":"A","specialty":"市政","chapter":"混凝土工程"}],
			"resources":[{"code":"L01","kind":"人工","category_path":"人工/普工","region":"成都市区","period":"2026-06"}]
		},
		"dict":{"op":"add","kind":"specialty","name":"市政","rename":"道路","code":"04","parent":"x","level":1,"sort":3,"note":"n"}
	}`), &p); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		what string
		got  string
		want string
	}{
		{"specialty", p.Specialty, "市政"},
		{"region", p.Region, "成都市区"},
		{"period", p.Period, "2026-06"},
		{"from_code", p.FromCode, "A"},
		{"new_code", p.NewCode, "A-9"},
		{"new_title", p.NewTitle, "垫层"},
		{"projects[0].specialty", p.Classify.Projects[0].Specialty, "土壤修复"},
		{"projects[0].region", p.Classify.Projects[0].Region, "成都市区"},
		{"projects[0].price_period", p.Classify.Projects[0].PricePeriod, "2026-06"},
		{"bills[0].division", p.Classify.Bills[0].Division, "垫层工程"},
		{"bills[0].specialty", p.Classify.Bills[0].Specialty, "市政"},
		{"quotas[0].chapter", p.Classify.Quotas[0].Chapter, "混凝土工程"},
		{"resources[0].category_path", p.Classify.Resources[0].CategoryPath, "人工/普工"},
		{"resources[0].region", p.Classify.Resources[0].Region, "成都市区"},
		{"resources[0].period", p.Classify.Resources[0].Period, "2026-06"},
		{"resources[0].kind", p.Classify.Resources[0].Kind, "人工"},
		{"dict.op", p.Dict.Op, "add"},
		{"dict.kind", p.Dict.Kind, "specialty"},
		{"dict.name", p.Dict.Name, "市政"},
		{"dict.rename", p.Dict.Rename, "道路"},
		{"dict.code", p.Dict.Code, "04"},
		{"dict.parent", p.Dict.Parent, "x"},
		{"dict.level", itoa(p.Dict.Level), "1"},
		{"dict.sort", itoa(*p.Dict.Sort), "3"},
		{"dict.note", *p.Dict.Note, "n"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q（入参键与工具说明不一致？）", c.what, c.got, c.want)
		}
	}
	if len(p.ItemIDs) != 1 || p.ItemIDs[0] != 7 {
		t.Errorf("item_ids = %v, want [7]", p.ItemIDs)
	}
	// cost_project_apply 同理。
	var ap costProjectApplyArgs
	if err := json.Unmarshal([]byte(`{"action":"create_project","specialty":"市政","region":"成都市区","price_period":"2026-06",
		"items":[{"code":"A","specialty":"土壤修复"}]}`), &ap); err != nil {
		t.Fatal(err)
	}
	if ap.Specialty != "市政" || ap.Region != "成都市区" || ap.PricePeriod != "2026-06" || *ap.Items[0].Specialty != "土壤修复" {
		t.Fatalf("cost_project_apply 入参键与说明不一致： %+v", ap)
	}
	applySchema := string(costProjectApplyTool{}.Schema())
	for _, needle := range []string{`"specialty"`, `"region"`, `"price_period"`} {
		if !strings.Contains(applySchema, needle) {
			t.Errorf("cost_project_apply 说明里应有键 %s", needle)
		}
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
