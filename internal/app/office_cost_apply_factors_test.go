package app

// office_cost_apply_factors_test.go —— cost_project_apply save_items 的定额换算系数
// （2026-11-27，三级树表样表的定额行「工程量」列）：挂接量 = 清单工程量 × 系数；
// 重发一行没带系数时沿用原挂接的比例——导入带进来的系数不会被整体替换清成「跟随」。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

func TestCostProjectApplySaveItemsQuotaFactors(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	apply := costProjectApplyTool{a: a}
	res, err := s.SaveResource(workcost.Resource{Kind: workcost.KindMaterial, Title: "商品混凝土", Unit: "m³", BasePrice: 400})
	if err != nil {
		t.Fatalf("保存资源失败: %v", err)
	}
	for _, code := range []string{"D-A1", "D-A2"} {
		if _, err := s.SaveQuota(workcost.Quota{Code: code, Title: "定额" + code, Unit: "m³",
			Items: []workcost.QuotaItem{{ResourceCode: res.Code, Quantity: 0.1}}}, nil); err != nil {
			t.Fatalf("保存定额 %s 失败: %v", code, err)
		}
	}
	exec := func(args map[string]any) (string, error) {
		b, _ := json.Marshal(args)
		return apply.Execute(context.Background(), b)
	}
	created, err := exec(map[string]any{"action": "create_project", "name": "系数"})
	if err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	pid := parseID(t, created)

	run := func(args map[string]any) string {
		t.Helper()
		out, err := exec(args)
		if err != nil {
			t.Fatalf("cost_project_apply %v 失败: %v", args["action"], err)
		}
		return out
	}
	mustFail := func(args map[string]any, want string) {
		t.Helper()
		out, err := exec(args)
		if err == nil || !strings.Contains(err.Error()+out, want) {
			t.Fatalf("应报「%s」，实得 err=%v out=%s", want, err, out)
		}
	}
	// item 一条挂 D-A1 + D-A2、工程量 100 的清单行；extra 覆盖字段。
	item := func(extra map[string]any) map[string]any {
		row := map[string]any{"title": "垫层", "unit": "m³", "quantity": 100.0, "quota_code": "D-A1", "quota_codes": []string{"D-A2"}}
		for k, v := range extra {
			row[k] = v
		}
		return row
	}
	save := func(rows ...map[string]any) map[string]any {
		list := make([]any, 0, len(rows))
		for _, r := range rows {
			list = append(list, r)
		}
		return map[string]any{"action": "save_items", "project_id": pid, "items": list}
	}
	wantLinks := func(code string, a1, a2 float64) {
		t.Helper()
		it, err := s.FindBillItem(pid, code)
		if err != nil || it == nil {
			t.Fatalf("清单 %s 不在库: %v", code, err)
		}
		links, err := s.BillQuotaLinks(it.ID)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]float64{}
		for _, lk := range links {
			got[lk.QuotaCode] = lk.Quantity
		}
		if len(got) != 2 || got["D-A1"] != a1 || got["D-A2"] != a2 {
			t.Fatalf("%s 的挂接量应 D-A1=%v / D-A2=%v，实得 %v", code, a1, a2, got)
		}
	}

	// 新行带系数：D-A1 ×1.5、D-A2 ×0.5 → 挂接量 150 / 50；回执点名带系数的行。
	out := run(save(item(map[string]any{"code": "Q01", "quota_factors": map[string]any{"D-A1": 1.5, "D-A2": 0.5}})))
	if !strings.Contains(out, "其中 1 行的定额挂接量 = 清单工程量 × 换算系数") {
		t.Errorf("回执应点名带换算系数的行：%s", out)
	}
	wantLinks("Q01", 150, 50)

	// 重发只改名称、带回挂接集合但不带系数：沿用原比例，不被清成跟随清单工程量。
	run(save(item(map[string]any{"code": "Q01", "title": "垫层（改名）"})))
	wantLinks("Q01", 150, 50)

	// 改工程量到 200：挂接量按系数跟着走（300 / 100）。
	run(save(item(map[string]any{"code": "Q01", "quantity": 200.0})))
	wantLinks("Q01", 300, 100)

	// 只给系数、不带挂接集合：套在该行原有挂接上（D-A1 改成跟随）。
	run(save(map[string]any{"code": "Q01", "title": "垫层（改名）", "unit": "m³", "quantity": 200.0,
		"quota_factors": map[string]any{"D-A1": 1}}))
	wantLinks("Q01", 0, 100)

	// 负系数、新行只给系数：整批拒，一行不写。
	mustFail(save(item(map[string]any{"code": "Q02", "quota_factors": map[string]any{"D-A1": -1}})), "换算系数 -1 不能为负")
	mustFail(save(map[string]any{"code": "Q03", "title": "垫层", "unit": "m³", "quantity": 100.0,
		"quota_factors": map[string]any{"D-A1": 1.5}}), "须与挂接集合同批提交")
	for _, code := range []string{"Q02", "Q03"} {
		if it, _ := s.FindBillItem(pid, code); it != nil {
			t.Errorf("被拒的批次不应写入 %s", code)
		}
	}
}
