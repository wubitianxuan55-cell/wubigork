package cost

// 审计 GA6-05 收敛钉子：MatchIndex 三索引键口径 + ParsePrice 双方言（含全角
// ￥/全角逗号归一与 RequireDigit 差异）。红=有人动了键归一或方言参数。

import "testing"

func TestMatchIndexKeys(t *testing.T) {
	ix := NewMatchIndex([]Summary{
		{Name: "rebar", Title: "热轧光圆钢筋", Code: "ａ-１ １２", Price: 3000}, // 全角编码：NormalizeCode 归一为 A-1-12
		{Name: "cement", Title: "P.O 42.5 水泥", Price: 480},
	})

	// 编码键 = NormalizeCode 口径（全角转半角、去空白、大写）。
	if e, ok := ix.MatchByCode("A-1 12"); !ok || e.Name != "rebar" {
		t.Errorf("MatchByCode 全角入库/半角查询应命中: got (%+v,%v)", e, ok)
	}
	// 带码未命中不回退标题（宁新增勿误配，调用方策略的前提）。
	if _, ok := ix.MatchByCode("B-9-9"); ok {
		t.Error("带码未命中应返回 false（不回退）")
	}
	if _, ok := ix.MatchByCode(""); ok {
		t.Error("未录编码（空）应返回 false")
	}

	// 标题键 = lower+trim；大小写/首尾空白不敏感。
	if e, ok := ix.MatchByTitle("  热轧光圆钢筋 "); !ok || e.Name != "rebar" {
		t.Errorf("MatchByTitle trim 应命中: got (%+v,%v)", e, ok)
	}

	// 名称（SlugName）兜底：标题索引未收时按名称键回退。
	if e, ok := ix.MatchByTitle("Rebar"); !ok || e.Name != "rebar" {
		t.Errorf("MatchByTitle SlugName 兜底应命中: got (%+v,%v)", e, ok)
	}
	if _, ok := ix.MatchByTitle("不存在"); ok {
		t.Error("未命中应返回 false")
	}

	// LoadMatchIndex：不可用 store=空索引（全部按新增处理的前提）。
	if ix := LoadMatchIndex(nil, "test"); ix == nil {
		t.Fatal("nil store 应返回空索引而非 nil")
	} else if _, ok := ix.MatchByTitle("热轧光圆钢筋"); ok {
		t.Error("空索引不应命中")
	}
}

func TestParsePriceDialects(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"3,200.00", 3200, true},
		{"3，200", 3200, true},    // 全角逗号千分位
		{"3200 元", 3200, true},   // 元后缀
		{"￥3181.00", 3181, true}, // 全角￥（反向证据②钉子：拍掉归一既有用例红）
		{"¥3181.00", 3181, true}, // 半角￥
		{" 480 ", 480, true},     // 首尾空白
		{"0", 0, false},          // 非正数无效
		{"-5", 0, false},         // 负数无效
		{"", 0, false},           // 空
		{"元", 0, false},          // 无数字
		{"abc", 0, false},        // 非数字
	}
	for _, c := range cases {
		got, ok := ParsePrice(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParsePrice(%q) = (%v,%v), want (%v,%v)", c.in, got, ok, c.want, c.ok)
		}
	}

	// pricefeed 方言：数字预检 + 两位舍入（发布价精度）。
	if v, ok := ParsePriceWithOpts("￥3181.00", PriceOpts{RequireDigit: true, Round2: true}); !ok || v != 3181 {
		t.Errorf("pricefeed 方言全角￥ = (%v,%v), want (3181,true)", v, ok)
	}
	if v, ok := ParsePriceWithOpts("3141.596", PriceOpts{Round2: true}); !ok || v != 3141.6 {
		t.Errorf("pricefeed 方言舍入 = (%v,%v), want (3141.6,true)", v, ok)
	}
	if v, ok := ParsePriceWithOpts("3141.596", PriceOpts{}); !ok || v != 3141.596 {
		t.Errorf("costimport 方言不取整 = (%v,%v), want (3141.596,true)", v, ok)
	}
	// RequireDigit 差异（历史口径钉子，收敛裁决见申报单）："Inf"/"NaN" 是
	// ParseFloat 特殊字面量——costimport 方言会当价格返回，pricefeed 方言
	// 的数字预检挡掉。本刀不改行为，只把差异参数化钉住。
	if v, ok := ParsePriceWithOpts("Inf", PriceOpts{}); !ok || v <= 0 {
		t.Errorf("costimport 方言 Inf 历史口径 = (%v,%v)，want 有限性不设防", v, ok)
	}
	if _, ok := ParsePriceWithOpts("Inf", PriceOpts{RequireDigit: true}); ok {
		t.Error("pricefeed 方言数字预检应挡 Inf")
	}
}
