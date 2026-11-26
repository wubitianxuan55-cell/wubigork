package workcost

import (
	"context"
	"testing"
)

// 工料机按地区分行后的检索面（2026-11-26 造价反馈，百锦路 #4）：带 region 只看
// 该地区的行；不带则各地区的行都在（查「同身份跨地区」）。列表路与关键词路
// 都要认这个过滤。
func TestSearch_RegionFilterAndCounts(t *testing.T) {
	s, _, cleanup := searchStore(t)
	defer cleanup()
	seedDict(t, s, DictRegion, "重庆")
	for _, r := range []Resource{
		{Code: "C20-CD", Kind: KindMaterial, Title: "商品混凝土", Spec: "C20", Unit: "m³", Region: "成都", CurrentPrice: 430},
		{Code: "C20-CQ", Kind: KindMaterial, Title: "商品混凝土", Spec: "C20", Unit: "m³", Region: "重庆", CurrentPrice: 455},
	} {
		if _, err := s.SaveResource(r); err != nil {
			t.Fatalf("建 %s 失败: %v", r.Code, err)
		}
	}
	idx := NewSearcher(s, nil, nil, nil)
	codes := func(hits []ResourceHit) string {
		out := ""
		for _, h := range hits {
			out += h.Item.Code + ","
		}
		return out
	}

	cases := []struct {
		name string
		q    SearchQuery
		want string
	}{
		{"列表路按地区过滤", SearchQuery{Region: "成都"}, "C20-CD,"},
		{"关键词路也按地区过滤", SearchQuery{Text: "混凝土", Region: "重庆"}, "C20-CQ,"},
		{"不给地区：各地区的行都在", SearchQuery{Text: "混凝土"}, "C20-CD,C20-CQ,"},
		{"「全部」等于不过滤", SearchQuery{Region: "全部"}, "C20-CD,C20-CQ,"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hits, err := idx.Resources(context.Background(), tc.q)
			if err != nil {
				t.Fatal(err)
			}
			if got := codes(hits); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}

	if n, err := s.CountResources("", "成都"); err != nil || n != 1 {
		t.Fatalf("成都的工料机应 1 条: %d %v", n, err)
	}
	if n, err := s.CountResources("材料", ""); err != nil || n != 2 {
		t.Fatalf("材料的工料机应 2 条: %d %v", n, err)
	}
}
