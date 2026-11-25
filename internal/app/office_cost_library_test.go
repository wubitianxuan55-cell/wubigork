package app

// office_cost_library_test.go — 造价库维护工具 cost_library 与 cost_project_apply
// 增量写修复的行为契约（2026-11-25 造价数据库推倒重建）。全部走隔离库
// （workcostTestApp），不碰用户正式库。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

func clRun(t *testing.T, tl costLibraryTool, args map[string]any) (string, error) {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("序列化入参失败: %v", err)
	}
	return tl.Execute(context.Background(), b)
}

// clMust 执行并断言回执含 want。
func clMust(t *testing.T, tl costLibraryTool, args map[string]any, want string) string {
	t.Helper()
	out, err := clRun(t, tl, args)
	if err != nil {
		t.Fatalf("cost_library %v 失败: %v", args["action"], err)
	}
	if !strings.Contains(out, want) {
		t.Fatalf("回执缺 %q：%s", want, out)
	}
	return out
}

// clMustFail 执行并断言被拒绝且错误含 want。
func clMustFail(t *testing.T, tl costLibraryTool, args map[string]any, want string) {
	t.Helper()
	out, err := clRun(t, tl, args)
	if err == nil {
		t.Fatalf("cost_library %v 应被拒绝，却成功：%s", args["action"], out)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("错误应含 %q，得到：%v", want, err)
	}
}

// clSeed 建一条引用链：工料机 R ← 定额 A（含量行）+ 定额 B（零含量）；项目 P
// 的清单行 WP01 挂 A+B（带原表单价快照）。
func clSeed(t *testing.T, s *workcost.Store) (r *workcost.Resource, pid int64, item workcost.BillItem) {
	t.Helper()
	res, err := s.SaveResource(workcost.Resource{Kind: workcost.KindMaterial, Title: "C20商品混凝土", Unit: "m³", BasePrice: 450, CurrentPrice: 460})
	if err != nil {
		t.Fatalf("建工料机失败: %v", err)
	}
	for _, q := range []workcost.Quota{
		{Code: "A", Title: "垫层", Unit: "m³", Items: []workcost.QuotaItem{{ResourceCode: res.Code, Quantity: 1.02}}},
		{Code: "B", Title: "压实", Unit: "m³"},
	} {
		if _, err := s.SaveQuota(q, nil); err != nil {
			t.Fatalf("建定额 %s 失败: %v", q.Code, err)
		}
	}
	// 直接走 store 建项目（工具的 create_project 在下面的用例里单独验）。
	var rates workcost.RateSet
	pid, err = s.UpsertBillProject("测算P", "", "", "", "", "", &rates, false, 0)
	if err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	it, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "WP01", Title: "垫层浇筑", Unit: "m³", Quantity: 100, QuotaCode: "A", ListPrice: 520})
	if err != nil {
		t.Fatalf("建清单行失败: %v", err)
	}
	for _, c := range []string{"A", "B"} {
		if _, err := s.AttachBillQuotaLink(it.ID, c, 0); err != nil {
			t.Fatalf("挂接定额 %s 失败: %v", c, err)
		}
	}
	return res, pid, it
}

// TestCostProjectApplySaveItemsKeepsSnapshotAndLinks 三个缺陷的回归钉：
// ① 改写已有行不清原表单价；② quota_codes 真正落引用表（整体替换，回执点名
// 被解除的挂接）；③ 没提交挂接字段=不动挂接；④ update_rates 只改传入字段。
func TestCostProjectApplySaveItemsKeepsSnapshotAndLinks(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	apply := costProjectApplyTool{a: a}
	res, err := s.SaveResource(workcost.Resource{Kind: workcost.KindLabor, Title: "普工", Unit: "工日", BasePrice: 100})
	if err != nil {
		t.Fatalf("保存资源失败: %v", err)
	}
	for _, code := range []string{"Q1", "Q2", "Q3"} {
		if _, err := s.SaveQuota(workcost.Quota{Code: code, Title: "定额" + code, Unit: "m³",
			Items: []workcost.QuotaItem{{ResourceCode: res.Code, Quantity: 0.1}}}, nil); err != nil {
			t.Fatalf("保存定额 %s 失败: %v", code, err)
		}
	}
	run := func(args map[string]any) string {
		t.Helper()
		b, _ := json.Marshal(args)
		out, err := apply.Execute(context.Background(), b)
		if err != nil {
			t.Fatalf("cost_project_apply %v 失败: %v", args["action"], err)
		}
		return out
	}
	pid := parseID(t, run(map[string]any{"action": "create_project", "name": "回归钉", "management_rate": 0.1, "tax_rate": 0.09}))

	// ② quota_codes 落引用表：三条挂接全部可见。
	run(map[string]any{"action": "save_items", "project_id": pid, "items": []map[string]any{
		{"code": "WP01", "title": "挖土", "unit": "m³", "quantity": 10, "quota_code": "Q1", "quota_codes": []string{"Q2", "Q3"}, "list_price": 88.5},
	}})
	item, _ := s.FindBillItem(pid, "WP01")
	if item == nil || len(item.QuotaCodes) != 3 || item.ListPrice != 88.5 {
		t.Fatalf("首写后应挂 3 条定额且原表单价 88.5，得到 %+v", item)
	}
	// ① 只改名称、不带 list_price：原表单价保持。
	// ③ 不提交挂接字段：挂接不动。
	out := run(map[string]any{"action": "save_items", "project_id": pid, "items": []map[string]any{
		{"code": "WP01", "title": "挖土（改名）", "unit": "m³", "quantity": 10},
	}})
	if strings.Contains(out, "解除了") {
		t.Fatalf("没提交挂接字段不应解除任何挂接：%s", out)
	}
	item, _ = s.FindBillItem(pid, "WP01")
	if item.Title != "挖土（改名）" || item.ListPrice != 88.5 || len(item.QuotaCodes) != 3 || item.QuotaCode != "Q1" {
		t.Fatalf("改名后应保原表单价与 3 条挂接，得到 %+v", item)
	}
	// 整体替换：只带回主定额 → 其余两条解除，回执点名。
	out = run(map[string]any{"action": "save_items", "project_id": pid, "items": []map[string]any{
		{"code": "WP01", "title": "挖土（改名）", "unit": "m³", "quantity": 10, "quota_code": "Q1"},
	}})
	if !strings.Contains(out, "解除了 2 条") {
		t.Fatalf("整体替换应点名解除 2 条挂接：%s", out)
	}
	item, _ = s.FindBillItem(pid, "WP01")
	if len(item.QuotaCodes) != 1 || item.QuotaCodes[0] != "Q1" {
		t.Fatalf("替换后应只剩 Q1，得到 %v", item.QuotaCodes)
	}
	// 显式清空挂接：传空串主定额 → 退回手填行。
	run(map[string]any{"action": "save_items", "project_id": pid, "items": []map[string]any{
		{"code": "WP01", "title": "挖土（改名）", "unit": "m³", "quantity": 10, "quota_code": "", "price_override": 30},
	}})
	item, _ = s.FindBillItem(pid, "WP01")
	if item.QuotaCode != "" || len(item.QuotaCodes) != 0 {
		t.Fatalf("显式清空后应退回手填行，得到 %+v", item)
	}

	// ④ update_rates 只改传入字段。
	run(map[string]any{"action": "update_rates", "project_id": pid, "tax_rate": 0.03})
	projects, _ := s.BillProjects()
	for _, p := range projects {
		if p.ID != pid {
			continue
		}
		if p.TaxRate != 0.03 || p.ManagementRate != 0.1 {
			t.Fatalf("update_rates 只应改税率：税=%v 管=%v", p.TaxRate, p.ManagementRate)
		}
		if p.RegulatoryRate != 0 || p.ProfitRate != 0 {
			t.Fatalf("未传的规费/利润应保持建项目时的 0，得到 规=%v 利=%v", p.RegulatoryRate, p.ProfitRate)
		}
	}
	// 一个费率字段都不传：拒绝（不做无效写入）。
	if _, err := apply.Execute(context.Background(), []byte(clJSON(t, map[string]any{"action": "update_rates", "project_id": pid}))); err == nil {
		t.Fatal("update_rates 不带任何费率字段应被拒绝")
	}
}

// clJSON 序列化入参（app 测试包已有 mustJSON，返回字符串）。
func clJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	return string(b)
}

// TestCostLibraryQuotaLifecycle 定额增改删：含量行引用的工料机不在库=拒绝；
// 更新时没给的头字段保持原值；被清单引用默认拒删，force 才连带解除。
func TestCostLibraryQuotaLifecycle(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	res, _, item := clSeed(t, s)
	tl := costLibraryTool{a: a}

	// 含量行引用不存在的工料机：拒绝，不编造。
	clMustFail(t, tl, map[string]any{"action": "save_quota", "quota": map[string]any{
		"code": "NEW-1", "title": "新定额", "unit": "m²",
		"items": []map[string]any{{"resource_code": "GHOST-9", "quantity": 1}},
	}}, "不在库")
	if q, _ := s.GetQuota("NEW-1"); q != nil {
		t.Fatal("被拒绝的定额不应落库")
	}
	// 无资源编码的含量行：拒绝（存储层不保存这种行，放行等于静默丢行）。
	clMustFail(t, tl, map[string]any{"action": "save_quota", "quota": map[string]any{
		"code": "NEW-1", "title": "新定额", "unit": "m²",
		"items": []map[string]any{{"kind": "人工", "title": "普工", "quantity": 0.5}},
	}}, "缺 resource_code")
	// 新建：演练不写库 → 正式写入 → 现算综合单价。
	clMust(t, tl, map[string]any{"action": "save_quota", "dry_run": true, "quota": map[string]any{
		"code": "NEW-1", "title": "新定额", "unit": "m²", "items": []map[string]any{{"resource_code": res.Code, "quantity": 2}},
	}}, "演练（未写库）")
	if q, _ := s.GetQuota("NEW-1"); q != nil {
		t.Fatal("演练不应写库")
	}
	out := clMust(t, tl, map[string]any{"action": "save_quota", "quota": map[string]any{
		"code": "NEW-1", "title": "新定额", "unit": "m²",
		"items": []map[string]any{{"resource_code": res.Code, "quantity": 2}},
	}}, "现算综合单价")
	if !strings.Contains(out, "材料 1 行") || !strings.Contains(out, "¥920.00") {
		t.Fatalf("含量构成应为材料 1 行、综合单价 2×460=920：%s", out)
	}
	// 更新：只改名称，头字段与含量行保持原值。
	clMust(t, tl, map[string]any{"action": "save_quota", "quota": map[string]any{"code": "NEW-1", "title": "改名定额"}}, "保留原有 1 行")
	q, _ := s.GetQuota("NEW-1")
	if q.Title != "改名定额" || q.Unit != "m²" || len(q.Items) != 1 {
		t.Fatalf("更新后应保留头字段与含量行，得到 %+v", q)
	}

	// 删除：被清单引用 → 默认拒绝（回执点名引用数）。
	clMustFail(t, tl, map[string]any{"action": "delete_quota", "codes": []string{"A"}}, "被 1 条清单项引用")
	if _, err := s.GetQuota("A"); err != nil {
		t.Fatal("被拒绝的删除不应生效")
	}
	// 不存在的编码：整批拒绝。
	clMustFail(t, tl, map[string]any{"action": "delete_quota", "codes": []string{"B", "NOPE"}}, "不存在")
	// force 演练：列出将解除的挂接，不写库。
	clMust(t, tl, map[string]any{"action": "delete_quota", "dry_run": true, "force": true, "codes": []string{"A"}}, "将删除定额 1 条")
	if _, err := s.GetQuota("A"); err != nil {
		t.Fatalf("演练不应删除定额，得到 err=%v", err)
	}
	clMust(t, tl, map[string]any{"action": "delete_quota", "force": true, "codes": []string{"A"}}, "解除清单挂接 1 条")
	if _, err := s.GetQuota("A"); !workcostNotFound(err) {
		t.Fatalf("定额 A 应已删除，得到 err=%v", err)
	}
	got, _ := s.FindBillItem(item.ProjectID, item.Code)
	// 主定额列改指剩余挂接的第一条（B）；一条不剩才退回手填行。
	if got == nil || got.QuotaCode != "B" || len(got.QuotaCodes) != 1 || got.QuotaCodes[0] != "B" {
		t.Fatalf("强删后清单行应只剩挂接 B 且主定额指向它，得到 %+v", got)
	}
}

// TestCostLibraryResourceLifecycle 工料机增改删调价：新建必须给类别；身份重复
// 拒绝；按编码更新时没给的字段保持原值；改编码不支持；被引用默认拒删。
func TestCostLibraryResourceLifecycle(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	res, _, _ := clSeed(t, s)
	tl := costLibraryTool{a: a}

	clMustFail(t, tl, map[string]any{"action": "save_resource", "resource": map[string]any{"title": "没给类别的材料", "unit": "t"}}, "必须给类别")
	// 同身份再存（不给编码）：按身份更新，入参没带的字段从原行带过来——价格不能被清成 0
	// （2026-11-25 测试逮到的缺陷：先合并后定目标，一次「同身份再存」把基准价/现行价写成 0）。
	clMust(t, tl, map[string]any{"action": "save_resource", "resource": map[string]any{
		"kind": "材料", "title": "C20商品混凝土", "unit": "m³", "note": "同身份再存",
	}}, "更新工料机")
	if same, _ := s.GetResourceByCode(res.Code); same.BasePrice != 450 || same.CurrentPrice != 460 || same.Note != "同身份再存" {
		t.Fatalf("同身份再存应保留价格，得到 %+v", same)
	}
	out := clMust(t, tl, map[string]any{"action": "save_resource", "resource": map[string]any{
		"kind": "材料", "title": "中砂", "unit": "m³", "current_price": 120, "source": "市场询价",
	}}, "新建工料机")
	if !strings.Contains(out, "按现行价补齐") {
		t.Fatalf("新建没给基准价时应用现行价补齐：%s", out)
	}
	sand, err := s.ListResources("", "中砂")
	if err != nil || len(sand) != 1 || sand[0].BasePrice != 120 || sand[0].CurrentPrice != 120 {
		t.Fatalf("新建的工料机应基准价=现行价=120，得到 %+v err=%v", sand, err)
	}
	// 把中砂改名成已有资源的身份：身份重复，拒绝。
	clMustFail(t, tl, map[string]any{"action": "save_resource", "resource": map[string]any{"code": sand[0].Code, "title": "C20商品混凝土"}}, "重复")
	// 按编码更新：只改现行价，其余保持；回执带影响面。
	out = clMust(t, tl, map[string]any{"action": "save_resource", "resource": map[string]any{
		"code": res.Code, "current_price": 480,
	}}, "更新工料机")
	if !strings.Contains(out, "1 条定额含量行引用它") || !strings.Contains(out, "¥460.00 → ¥480.00") {
		t.Fatalf("改价回执应给出取用价变化与影响面：%s", out)
	}
	got, _ := s.GetResourceByCode(res.Code)
	if got.CurrentPrice != 480 || got.BasePrice != 450 || got.Title != "C20商品混凝土" || got.Unit != "m³" {
		t.Fatalf("更新后没给的字段应保持原值，得到 %+v", got)
	}
	// 改编码不支持。
	clMustFail(t, tl, map[string]any{"action": "save_resource", "resource": map[string]any{"code": "NEWCODE", "id": got.ID}}, "不支持改工料机编码")

	// 调价：演练不写库 → 正式写历史并推进现行价。库定规矩：期数写 YYYY-MM，
	// 地区必须在地区字典里。
	if err := s.DictAdd(workcost.DictItem{Kind: workcost.DictRegion, Name: "乐山"}); err != nil {
		t.Fatal(err)
	}
	clMust(t, tl, map[string]any{"action": "set_price", "dry_run": true, "resource_code": res.Code, "price": 500, "period": "2026-11"}, "演练（未写库）")
	if got, _ = s.GetResourceByCode(res.Code); got.CurrentPrice != 480 {
		t.Fatal("演练不应改价")
	}
	clMust(t, tl, map[string]any{"action": "set_price", "resource_code": res.Code, "price": 500, "period": "2026-11", "region": "乐山"}, "现行价已更新为 ¥500.00")
	hist, _ := s.ResourcePrices(got.ID)
	if len(hist) != 1 || hist[0].Period != "2026-11" || hist[0].Region != "乐山" {
		t.Fatalf("应留下一条调价历史，得到 %+v", hist)
	}
	// 旧写法「2026年11月」能归一；信息价期数「2026-11」认不出，拒收不猜。
	clMust(t, tl, map[string]any{"action": "set_price", "resource_code": res.Code, "price": 510, "period": "2026年12月"}, "现行价已更新为 ¥510.00")
	clMustFail(t, tl, map[string]any{"action": "set_price", "resource_code": res.Code, "price": 520, "period": "2026年第12期"}, "不是年月写法")
	clMustFail(t, tl, map[string]any{"action": "set_price", "resource_code": res.Code, "price": 520, "region": "不在字典的地方"}, "不在地区字典里")

	// 删除：被定额含量行引用 → 拒绝；未被引用的可删；force 才连带。
	clMustFail(t, tl, map[string]any{"action": "delete_resource", "codes": []string{res.Code}}, "仍被定额含量行引用")
	clMust(t, tl, map[string]any{"action": "delete_resource", "dry_run": true, "force": true, "ids": []int64{got.ID}}, "解除 1 条含量行引用")
	if _, err := s.GetResource(got.ID); err != nil {
		t.Fatal("演练不应删除")
	}
	sandRow := sand[0]
	clMust(t, tl, map[string]any{"action": "delete_resource", "ids": []int64{sandRow.ID}}, "已删除工料机 1 条")
	if _, err := s.GetResource(sandRow.ID); !workcostNotFound(err) {
		t.Fatal("未被引用的工料机应已删除")
	}
	clMust(t, tl, map[string]any{"action": "delete_resource", "force": true, "ids": []int64{got.ID}}, "1 条含量行引用悬空")
	if _, err := s.GetResource(got.ID); !workcostNotFound(err) {
		t.Fatal("force 删除后工料机应不存在")
	}
}

// TestCostLibraryBulkPriceAndBillDelete 批量调价（演练不写库、歧义跳过）与
// 清单层删除（不属于该项目的行整批拒绝、项目删除演练不写库）。
func TestCostLibraryBulkPriceAndBillDelete(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	res, pid, item := clSeed(t, s)
	tl := costLibraryTool{a: a}

	out := clMust(t, tl, map[string]any{"action": "bulk_price", "dry_run": true, "rows": []map[string]any{
		{"title": res.Title, "unit": res.Unit, "price": 470},
		{"title": "查无此物", "price": 1},
	}}, "匹配 1 行｜跳过 1 行")
	if !strings.Contains(out, "未找到同名资源") {
		t.Fatalf("跳过原因应显形：%s", out)
	}
	if got, _ := s.GetResourceByCode(res.Code); got.CurrentPrice != 460 {
		t.Fatal("演练不应改价")
	}
	clMust(t, tl, map[string]any{"action": "bulk_price", "rows": []map[string]any{
		{"title": res.Title, "unit": res.Unit, "price": 470, "period": "2026-11"},
	}}, "已更新 1 条工料机的现行价")
	if got, _ := s.GetResourceByCode(res.Code); got.CurrentPrice != 470 {
		t.Fatalf("批量调价后现行价应为 470，得到 %v", got.CurrentPrice)
	}

	// 清单行删除：不属于该项目的 id 整批拒绝。
	clMustFail(t, tl, map[string]any{"action": "delete_items", "project_id": pid, "item_ids": []int64{item.ID, 999999}}, "不属于项目")
	clMust(t, tl, map[string]any{"action": "delete_items", "dry_run": true, "project_id": pid, "item_ids": []int64{item.ID}}, "将删除清单行 1 条")
	if items, _ := s.BillItems(pid); len(items) != 1 {
		t.Fatal("演练不应删除清单行")
	}
	clMust(t, tl, map[string]any{"action": "delete_items", "project_id": pid, "item_ids": []int64{item.ID}}, "已删除清单行 1 条")
	if items, _ := s.BillItems(pid); len(items) != 0 {
		t.Fatal("清单行应已删除")
	}

	// 项目删除：演练不写库，正式删除连带独占定额。
	_, pid2, _ := clSeed(t, s)
	clMustFail(t, tl, map[string]any{"action": "delete_project", "project_id": 999999}, "不存在")
	clMust(t, tl, map[string]any{"action": "delete_project", "dry_run": true, "project_id": pid2}, "将删除项目")
	if p := findBillProject(s, pid2); p == nil {
		t.Fatal("演练不应删除项目")
	}
	clMust(t, tl, map[string]any{"action": "delete_project", "project_id": pid2}, "已删除项目")
	if p := findBillProject(s, pid2); p != nil {
		t.Fatal("项目应已删除")
	}
}

// TestCostLibraryHealthAndSeed 库级健康（零价资源点名）与资料条目资源化。
func TestCostLibraryHealthAndSeed(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	if _, err := s.SaveResource(workcost.Resource{Kind: workcost.KindMaterial, Title: "零价材料", Unit: "t"}); err != nil {
		t.Fatalf("建零价资源失败: %v", err)
	}
	out := clMust(t, tl, map[string]any{"action": "health"}, "零价资源（现行价与基准价均为 0）：1 条")
	if !strings.Contains(out, "零价材料") {
		t.Fatalf("健康扫描应点名零价资源：%s", out)
	}
	// 资料条目资源化：演练不写库 → 正式入库。
	if err := a.hubCostStore().Save(cost.Entry{Name: "cement-42", Title: "P.O 42.5 水泥", Category: "材料", Unit: "t", Price: 380}); err != nil {
		t.Fatalf("写资料条目失败: %v", err)
	}
	clMust(t, tl, map[string]any{"action": "seed", "dry_run": true}, "可资源化 1 条")
	if list, _ := s.ListResources("", "水泥"); len(list) != 0 {
		t.Fatal("演练不应入库")
	}
	clMust(t, tl, map[string]any{"action": "seed"}, "新建 1 / 更新 0")
	if list, _ := s.ListResources("", "水泥"); len(list) != 1 {
		t.Fatalf("资源化后应有 1 条水泥资源，得到 %d", len(list))
	}
	// 重复执行幂等：全部走更新。
	clMust(t, tl, map[string]any{"action": "seed"}, "新建 0 / 更新 1")
}

// TestCostLibraryImportProject 导入五表工作簿：演练不写库并给出规模；正式导入
// 落项目、清单、工料机与定额；非 xlsx 与越界路径被拒。
func TestCostLibraryImportProject(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	fixture, err := filepath.Abs(filepath.Join("..", "gaea", "workcost", "testdata", "什邡2个污染地块施工成本测算表_补项.xlsx"))
	if err != nil || fileMissing(fixture) {
		t.Skipf("夹具不可用（%v），跳过", err)
	}
	clMustFail(t, tl, map[string]any{"action": "import_project", "path": "报价单.csv"}, "只收 .xlsx")
	clMustFail(t, tl, map[string]any{"action": "import_project", "path": "不存在的成本测算表.xlsx"}, "读不到工作簿")

	out := clMust(t, tl, map[string]any{"action": "import_project", "dry_run": true, "path": fixture}, "演练（未写库）")
	if !strings.Contains(out, "**规模**: 工料机") || !strings.Contains(out, "清单项") {
		t.Fatalf("演练回执应给出规模：%s", out)
	}
	if projects, _ := s.BillProjects(); len(projects) != 0 {
		t.Fatal("演练不应落库")
	}
	clMust(t, tl, map[string]any{"action": "import_project", "path": fixture}, "**落库**")
	projects, _ := s.BillProjects()
	if len(projects) != 1 {
		t.Fatalf("正式导入后应有 1 个项目，得到 %d", len(projects))
	}
	if projects[0].ItemCount == 0 {
		t.Fatal("导入后应有清单行")
	}
	quotas, _ := s.ListQuotas("", "")
	if len(quotas) == 0 {
		t.Fatal("导入后应有定额")
	}
	// 重导入幂等：同一份文件再导一次，项目数不变。
	clMust(t, tl, map[string]any{"action": "import_project", "path": fixture}, "同名项目")
	if again, _ := s.BillProjects(); len(again) != 1 {
		t.Fatalf("重导入应幂等（仍是 1 个项目），得到 %d", len(again))
	}
}

func fileMissing(p string) bool {
	_, err := os.Stat(p)
	return err != nil
}

// TestCostLibraryPromptAnchors 造价档的模式行、身份稿、技能正文都必须点名
// cost_library 与「板块只读、维护在本档」的口径——缺了就是库维护能力无声消失
// （用户在板块里找不到导入按钮，模型也不接）。其余档不携带该句。
func TestCostLibraryPromptAnchors(t *testing.T) {
	sources := map[string]string{
		"模式行": officeRunModeLine(OfficeRunModeCost),
		"身份稿": officeModeSystemPrompt(OfficeRunModeCost, config.Default()),
		"技能":  officeCostEstimationPlaybookBody,
	}
	for name, src := range sources {
		for _, want := range []string{"cost_library", "dry_run", "只读"} {
			if !strings.Contains(src, want) {
				t.Fatalf("造价档%s缺库维护锚 %q", name, want)
			}
		}
	}
	for _, want := range []string{"import_project", "整体替换", "save_quota", "set_price"} {
		if !strings.Contains(sources["技能"], want) || !strings.Contains(sources["身份稿"], want) {
			t.Fatalf("造价档技能或身份稿缺库维护细则锚 %q", want)
		}
	}
	for _, mode := range []string{OfficeRunModeStandard, OfficeRunModeProposal} {
		if strings.Contains(officeRunModeLine(mode), "cost_library") {
			t.Fatalf("%s 档模式行不应携带造价库维护句（造价族在该档隐藏）", mode)
		}
	}
}
