package workcost

// search_alias_test.go — 检索别名：字典里登记「砼」→「混凝土」后，关键词路双向展开；
// 仅字面匹配不展开；别名条目自身必须带标准说法。

import (
	"context"
	"strings"
	"testing"
)

func mustSearchResources(t *testing.T, h *Searcher, kw string) []ResourceHit {
	t.Helper()
	hits, err := h.Resources(context.Background(), SearchQuery{Text: kw})
	if err != nil {
		t.Fatalf("检索 %q: %v", kw, err)
	}
	return hits
}

func TestSearch_AliasesExpandBothWays(t *testing.T) {
	s, _, cleanup := searchStore(t)
	defer cleanup()
	saveSearchFixtures(t, s)           // 有「混凝土外加剂」，没有名称里写「砼」的
	h := NewSearcher(s, nil, nil, nil) // 不开向量路：只看关键词路的别名展开

	// 没登记别名：「砼」对不上「混凝土外加剂」
	if hits := mustSearchResources(t, h, "砼"); len(hits) != 0 {
		t.Fatalf("没登记别名时不该命中: %+v", hits)
	}
	if err := s.DictAdd(DictItem{Kind: DictAlias, Name: "砼", Parent: "混凝土"}); err != nil {
		t.Fatal(err)
	}
	if hits := mustSearchResources(t, h, "砼"); len(hits) != 1 || hits[0].Item.Title != "混凝土外加剂" || hits[0].Via != ViaKeyword {
		t.Fatalf("查「砼」应经别名命中「混凝土外加剂」并标关键词: %+v", hits)
	}
	// 仅字面匹配：要的是「含这个词的全列出来」，不按别名展开
	if hits := mustSearchResources(t, h, "砼"); len(hits) != 1 {
		t.Fatalf("前置：别名命中 1 条: %+v", hits)
	}
	if hits, _ := h.Resources(context.Background(), SearchQuery{Text: "砼", Exact: true}); len(hits) != 0 {
		t.Fatalf("仅字面匹配不该按别名展开: %+v", hits)
	}
	// 反向：有人把别名写进了名称（「商砼」），查标准说法也要能搜到它
	if _, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "商砼 C30", Unit: "m3"}); err != nil {
		t.Fatal(err)
	}
	got := hitTitles(mustSearchResources(t, h, "混凝土"), func(r Resource) string { return r.Title })
	if len(got) != 2 {
		t.Fatalf("查「混凝土」应同时命中名称写「砼」的行: %v", got)
	}
}

func TestDictAliasRules(t *testing.T) {
	s, _, cleanup := searchStore(t)
	defer cleanup()
	if err := s.DictAdd(DictItem{Kind: DictAlias, Name: "铲车"}); err == nil || !strings.Contains(err.Error(), "标准说法") {
		t.Fatalf("别名缺标准说法应拒收: %v", err)
	}
	if err := s.DictAdd(DictItem{Kind: DictAlias, Name: "土方", Parent: "土方"}); err == nil {
		t.Fatal("别名与标准说法是同一个词应拒收")
	}
	if err := s.DictAdd(DictItem{Kind: DictAlias, Name: "铲车", Parent: "装载机"}); err != nil {
		t.Fatal(err)
	}
	if aliases, _ := s.SearchAliases(); aliases["铲车"] != "装载机" {
		t.Fatalf("别名表: %v", aliases)
	}
	// 别名不引用任何数据，可以随时删
	if err := s.DictDelete(DictAlias, "铲车"); err != nil {
		t.Fatalf("别名应可删: %v", err)
	}
}

func TestExpandAliases(t *testing.T) {
	aliases := map[string]string{"砼": "混凝土", "铲车": "装载机", "PC": "预制混凝土"}
	got := expandAliases("", aliases)
	if len(got) != 0 {
		t.Fatalf("空查询词不展开: %v", got)
	}
	got = expandAliases("挖掘机", aliases)
	if len(got) != 0 {
		t.Fatalf("查询词里没有别名也不含标准说法时不展开: %v", got)
	}
	// 别名 → 标准说法（保留查询词的其余部分）
	got = expandAliases("铲车 装车", aliases)
	if len(got) != 1 || got[0] != "装载机 装车" {
		t.Fatalf("别名展开: %v", got)
	}
	// 大小写不敏感
	got = expandAliases("pc 构件", aliases)
	if len(got) != 1 || got[0] != "预制混凝土 构件" {
		t.Fatalf("大小写不敏感: %v", got)
	}
	// 标准说法 → 别名；不产出原词，也不把只沾边的别名（「预制混凝土」不是「混凝土」）带出来
	got = expandAliases("混凝土", aliases)
	if len(got) != 1 || got[0] != "砼" {
		t.Fatalf("标准说法反向展开应只得「砼」: %v", got)
	}
}
