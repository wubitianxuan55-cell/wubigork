package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

// cost_project 的 exact=true（2026-11-27 AI 反馈 #8）：只认字面匹配、不混语义近似；
// 按编码稳定排序并可翻页；回执给字面匹配总数，翻到头如实说。
func TestCostProjectExactKeyword(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	cp := costProjectTool{a: a}
	fbDicts(t, s, workcost.DictSpecialty, "土壤修复")
	for _, r := range []workcost.Resource{
		{Code: "HNT-1", Kind: workcost.KindMaterial, Title: "商品混凝土 C30", Unit: "m³"},
		{Code: "HNT-2", Kind: workcost.KindMaterial, Title: "商品混凝土 C35", Unit: "m³"},
		{Code: "SN-1", Kind: workcost.KindMaterial, Title: "水泥", Unit: "t"},
	} {
		fbSave(t, s, r)
	}
	for _, q := range []workcost.Quota{
		{Code: "DC-1", Title: "混凝土垫层", Unit: "m³", Specialty: "土壤修复"},
		{Code: "DC-2", Title: "混凝土基础", Unit: "m³", Specialty: "土壤修复"},
		{Code: "QT-1", Title: "砌砖墙", Unit: "m³", Specialty: "土壤修复"},
	} {
		if _, err := s.SaveQuota(q, nil); err != nil {
			t.Fatal(err)
		}
	}

	all := toolRun(t, cp, map[string]any{"action": "quotas", "keyword": "混凝土", "exact": true})
	if !strings.Contains(all, "定额 2 条（字面匹配共 2 条，本次第 1~2 条）") || strings.Contains(all, "还有更多") || strings.Contains(all, "语义") {
		t.Fatalf("字面匹配应给总数、不提语义：%s", all)
	}
	page1 := toolRun(t, cp, map[string]any{"action": "quotas", "keyword": "混凝土", "exact": true, "limit": 1})
	if !strings.Contains(page1, "定额 1 条（字面匹配共 2 条，本次第 1~1 条；还有更多——offset=1 续读下页）") {
		t.Fatalf("字面匹配应可翻页：%s", page1)
	}
	page2 := toolRun(t, cp, map[string]any{"action": "quotas", "keyword": "混凝土", "exact": true, "limit": 1, "offset": 1})
	if !strings.Contains(page2, "本次第 2~2 条）") || strings.Contains(page2, "还有更多") {
		t.Fatalf("末页不该再说还有更多：%s", page2)
	}
	end := toolRun(t, cp, map[string]any{"action": "quotas", "keyword": "混凝土", "exact": true, "limit": 1, "offset": 5})
	if !strings.Contains(end, "字面匹配共 2 条，offset=5 已到末尾") {
		t.Fatalf("翻过头应如实说：%s", end)
	}
	// 不带 exact 的关键词模式：融合排序没有稳定全序，不给总数与翻页。
	hybrid := toolRun(t, cp, map[string]any{"action": "quotas", "keyword": "混凝土"})
	if strings.Contains(hybrid, "字面匹配") || !strings.Contains(hybrid, "定额 2 条") {
		t.Fatalf("常规关键词模式口径不变：%s", hybrid)
	}
	// 字面没有的不召回。
	none := toolRun(t, cp, map[string]any{"action": "quotas", "keyword": "砼", "exact": true})
	if !strings.Contains(none, "定额库无匹配") || !strings.Contains(none, "exact=true") {
		t.Fatalf("字面没有就报无匹配：%s", none)
	}
	// 工料机同理，过滤照常生效。
	res := toolRun(t, cp, map[string]any{"action": "resources", "keyword": "混凝土", "exact": true})
	if !strings.Contains(res, "资源 2 条（字面匹配共 2 条，本次第 1~2 条）") {
		t.Fatalf("工料机字面匹配：%s", res)
	}
	if res := toolRun(t, cp, map[string]any{"action": "resources", "keyword": "混凝土", "exact": true, "kind": "机械"}); !strings.Contains(res, "工料机库无匹配") {
		t.Fatalf("类型过滤应照常生效：%s", res)
	}
	// exact 不带 keyword 不生效，照旧是列表模式。
	lst := toolRun(t, cp, map[string]any{"action": "resources", "exact": true})
	if !strings.Contains(lst, "资源 3 条（共 3 条）") {
		t.Fatalf("不带 keyword 的 exact 应退回列表模式：%s", lst)
	}
}
