package workcost

// rules_test.go — 库定规矩的用例（2026-11-25）：字典校验、期数归一、工料机按地区
// 分行、跨专业不并、整理归类、借鉴生成。另给全部测试夹具一个 seedDict——库定规矩
// 之后，带专业/地区/分类的写入必须先有字典项。

import (
	"strings"
	"testing"
)

// seedDict 测试夹具：把专业/地区/分类收进字典。
func seedDict(t *testing.T, s *Store, kind string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := s.DictAdd(DictItem{Kind: kind, Name: n}); err != nil {
			t.Fatalf("字典 %s 加「%s」失败: %v", kind, n, err)
		}
	}
}

func TestNormalizePeriod(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "", true},
		{"2026-06", "2026-06", true},
		{"2026年6月", "2026-06", true},
		{"2026年11月", "2026-11", true},
		{"2026.6", "2026-06", true},
		{"2026/6", "2026-06", true},
		{"202606", "2026-06", true},
		{" 2026 年 6 月 ", "2026-06", true},
		{"2026年第7期", "", false}, // 信息价期数不猜
		{"2026-13", "", false},
		{"26-06", "", false},
		{"一季度", "", false},
	}
	for _, c := range cases {
		got, ok := NormalizePeriod(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalizePeriod(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestDictValidationOnWrite 规范值校验落在存储层：字典外的专业/地区/分类、认不出的
// 期数、四类之外的类别，一律拒收；空值=待整理，放行。
func TestDictValidationOnWrite(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	if _, err := s.SaveQuota(Quota{Code: "Q1", Title: "挖土方", Unit: "m³", Specialty: "市政"}, nil); err == nil ||
		!strings.Contains(err.Error(), "不在专业字典里") {
		t.Fatalf("字典外的专业应被拒收，err=%v", err)
	}
	if _, err := s.SaveQuota(Quota{Code: "Q1", Title: "挖土方", Unit: "m³"}, nil); err != nil {
		t.Fatalf("专业留空（待整理）应放行: %v", err)
	}
	seedDict(t, s, DictSpecialty, "市政")
	if _, err := s.SaveQuota(Quota{Code: "Q2", Title: "挖土方", Unit: "m³", Specialty: "市政"}, nil); err != nil {
		t.Fatalf("字典里的专业应通过: %v", err)
	}

	if _, err := s.SaveResource(Resource{Code: "C20", Kind: "材料", Title: "商品混凝土", Region: "广东省"}); err == nil ||
		!strings.Contains(err.Error(), "不在地区字典里") {
		t.Fatalf("字典外的地区应被拒收，err=%v", err)
	}
	seedDict(t, s, DictRegion, "四川省")
	if _, err := s.SaveResource(Resource{Code: "C20", Kind: "材料", Title: "商品混凝土", Region: "四川省",
		CategoryPath: "材料/混凝土"}); err == nil || !strings.Contains(err.Error(), "不在工料机分类字典里") {
		t.Fatalf("字典外的分类应被拒收，err=%v", err)
	}
	if _, err := s.SaveResource(Resource{Code: "C20", Kind: "材料", Title: "商品混凝土", Region: "四川省",
		PriceDate: "2026年第7期"}); err == nil || !strings.Contains(err.Error(), "不是年月写法") {
		t.Fatalf("认不出的期数应被拒收，err=%v", err)
	}
	if _, err := s.SaveResource(Resource{Code: "C20", Kind: "地磅摊销估价", Title: "地磅"}); err == nil ||
		!strings.Contains(err.Error(), "不在 人工/材料/机械/外委 之内") {
		t.Fatalf("四类之外的类别应被拒收，err=%v", err)
	}
	// 认得的写法归一成 YYYY-MM 落库。
	r, err := s.SaveResource(Resource{Code: "C20", Kind: "材料", Title: "商品混凝土", Region: "四川省", PriceDate: "2026年6月"})
	if err != nil {
		t.Fatalf("落库失败: %v", err)
	}
	if r.PriceDate != "2026-06" {
		t.Fatalf("期数应归一成 2026-06，得到 %q", r.PriceDate)
	}
}

// TestResourceRegionIsIdentity 工料机按地区分行：同身份不同地区是两行，编码各归各；
// 一个地区的价不顶掉另一个地区的价；未标注地区的行不并入任何已标注地区的行。
func TestResourceRegionIsIdentity(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictRegion, "四川省", "重庆市")

	cd, err := s.SaveResource(Resource{Code: "C20", Kind: "材料", Title: "商品混凝土", Spec: "C20", Unit: "m³",
		Region: "四川省", CurrentPrice: 430, BasePrice: 430})
	if err != nil {
		t.Fatalf("成都行落库失败: %v", err)
	}
	cq, err := s.SaveResource(Resource{Code: "CL0007", Kind: "材料", Title: "商品混凝土", Spec: "C20", Unit: "m³",
		Region: "重庆市", CurrentPrice: 455, BasePrice: 455})
	if err != nil {
		t.Fatalf("重庆行落库失败: %v", err)
	}
	if cd.ID == cq.ID {
		t.Fatal("同身份不同地区应是两行")
	}
	got, _ := s.GetResource(cd.ID)
	if got.CurrentPrice != 430 {
		t.Fatalf("成都行的价被顶掉了：%v", got.CurrentPrice)
	}
	// 未标注地区的行：第三行，与前两行互不相干。
	if _, err := s.SaveResource(Resource{Code: "C20-2", Kind: "材料", Title: "商品混凝土", Spec: "C20", Unit: "m³",
		CurrentPrice: 400, BasePrice: 400}); err != nil {
		t.Fatalf("未标注地区的行落库失败: %v", err)
	}
	list, _ := s.ListResources("", "商品混凝土")
	if len(list) != 3 {
		t.Fatalf("应有 3 行（成都/重庆/未标注），得到 %d", len(list))
	}
	// 同地区同身份再存：复用该行，编码保留原值（入参带另一个码也不换锚点）。
	again, err := s.SaveResource(Resource{Code: "OTHER", Kind: "材料", Title: "商品混凝土", Spec: "C20", Unit: "m³",
		Region: "四川省", CurrentPrice: 435})
	if err != nil || again.ID != cd.ID || again.Code != "C20" {
		t.Fatalf("同地区同身份应复用且保留原码：id=%d code=%s err=%v", again.ID, again.Code, err)
	}
}

// TestFreeResourceCode 撞码消歧：码被别的资源占用时取最小空位「码-N」。
func TestFreeResourceCode(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	if _, err := s.SaveResource(Resource{Kind: "人工", Title: "普工", Code: "L01"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.FreeResourceCode("L01"); got != "L01-2" {
		t.Fatalf("占用的码应消歧成 L01-2，得到 %s", got)
	}
	if _, err := s.SaveResource(Resource{Kind: "人工", Title: "技工", Code: "L01-2"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.FreeResourceCode("L01"); got != "L01-3" {
		t.Fatalf("应取最小空位 L01-3，得到 %s", got)
	}
	if got, _ := s.FreeResourceCode("L09"); got != "L09" {
		t.Fatalf("未占用的码应原样返回，得到 %s", got)
	}
}

// TestNormalizeRegion 地区只到省（2026-11-27 拍板）：全称、去尾缀简称、「省/市」两级
// 写法收成省级；市县名与别名简称不收——不能下分。
func TestNormalizeRegion(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"四川省", "四川省", true},
		{"  四川  ", "四川省", true},
		{"四川省/成都市", "四川省", true},
		{"四川/成都/武侯区", "四川省", true},
		{"重庆市", "重庆市", true},
		{"内蒙古自治区", "内蒙古自治区", true},
		{"内蒙古", "内蒙古自治区", true},
		{"新疆", "新疆维吾尔自治区", true},
		{"香港", "香港特别行政区", true},
		{"黑龙江省", "黑龙江省", true},
		{"", "", true}, // 空=待整理
		{"成都市区", "", false},
		{"成都", "", false},
		{"乐山市", "", false},
		{"皖A", "", false},
		{"川", "", false},
	}
	for _, c := range cases {
		got, ok := NormalizeRegion(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("NormalizeRegion(%q) = (%q, %v)，想得 (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestCheckRegionProvincesOnly checkRegion 在归一后过字典：省名不在字典里照旧拒
// （新省份走 dict add，需用户确认）；下分写法在门口就被收掉或拒掉。
func TestCheckRegionProvincesOnly(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	seedDict(t, s, DictRegion, "四川省")

	if got, err := s.checkRegion("四川省/成都市", "测试"); err != nil || got != "四川省" {
		t.Fatalf("下分写法应收成省名: got=%q err=%v", got, err)
	}
	if _, err := s.checkRegion("成都市区", "测试"); err == nil || !strings.Contains(err.Error(), "不是省级行政区") {
		t.Fatalf("市名应被拒并道出只到省，err=%v", err)
	}
	if _, err := s.checkRegion("广东省", "测试"); err == nil || !strings.Contains(err.Error(), "不在地区字典里") {
		t.Fatalf("省名不在字典里照旧拒收，err=%v", err)
	}
	if got, err := s.checkRegion("", "测试"); err != nil || got != "" {
		t.Fatalf("空=待整理放行: got=%q err=%v", got, err)
	}
	// DictAdd 地区只收省：市名进不了字典。
	if err := s.DictAdd(DictItem{Kind: DictRegion, Name: "乐山市"}); err == nil || !strings.Contains(err.Error(), "不能下分") {
		t.Fatalf("字典新增市级名应被拒，err=%v", err)
	}
	if err := s.DictAdd(DictItem{Kind: DictRegion, Name: "广东省"}); err != nil {
		t.Fatalf("字典新增省名应通过: %v", err)
	}
	if got, err := s.checkRegion("广东/广州", "测试"); err != nil || got != "广东省" {
		t.Fatalf("入库后省名两极写法放行: got=%q err=%v", got, err)
	}
}
