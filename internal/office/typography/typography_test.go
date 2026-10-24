package typography

import (
	"math"
	"testing"
)

// 单位换算：正例 3 条 + 反例 2 条（零/负）。
// 反例是行为断言：零进零出、负进负出（线性换算，不取绝对值、不 panic），
// 退化输入的行为由此固定下来，防止后续改成静默翻转符号。
func TestUnitConversions(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		// 正例
		{"CmToTwips(2.54)=1440（1 英寸 = 1440 twips）", CmToTwips(2.54), 1440},
		{"PtToHalfPoints(10.5)=21（五号 10.5pt 的 w:sz 值）", PtToHalfPoints(10.5), 21},
		{"PtToTwips(12)=240（小四 12pt = 1 字符宽）", PtToTwips(12), 240},
		// 反例：零 / 负
		{"CmToTwips(0)=0（零进零出）", CmToTwips(0), 0},
		{"PtToTwips(-1)=-20（负进负出，不翻转）", PtToTwips(-1), -20},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s：got %d, want %d", c.name, c.got, c.want)
		}
	}
}

// 字号双向转换：16 档全量往返（号 → pt → 号）必须还原到同一档。
// 档位 pt 值互不相同，精确 pt 输入时最近档距离为 0，往返必然自洽——
// 若有人往表里加重复 pt 值，此测试会以「档位互异」的形式暴露。
func TestCNSizeRoundTrip(t *testing.T) {
	if len(CNSizeNames) != 16 {
		t.Fatalf("中文字号表应 16 档（初号~八号 + 小号档），got %d", len(CNSizeNames))
	}
	for name, pt := range CNSizeNames {
		gotPt, ok := CNSizeToPt(name)
		if !ok || gotPt != pt {
			t.Errorf("CNSizeToPt(%q) = (%v, %v), want (%v, true)", name, gotPt, ok, pt)
		}
		back, ok := PtToCNSize(pt)
		if !ok || back != name {
			t.Errorf("PtToCNSize(%v) = (%q, %v), want (%q, true)", pt, back, ok, name)
		}
	}
}

// 字号查表正反例：区间内取最近档；超界 / NaN / 未知名返回 false。
func TestCNSizeLookup(t *testing.T) {
	cases := []struct {
		name   string
		pt     float64
		want   string
		wantOK bool
	}{
		{"10.5 是五号本档", 10.5, "五号", true},
		{"14.6 距小三 0.4、距四号 0.6，取小三", 14.6, "小三", true},
		{"99 超出初号上界", 99, "", false},
		{"4.9 低于八号下界", 4.9, "", false},
		{"5 恰为八号下界（含边界）", 5, "八号", true},
		{"NaN 拒绝", math.NaN(), "", false},
	}
	for _, c := range cases {
		got, ok := PtToCNSize(c.pt)
		if ok != c.wantOK || got != c.want {
			t.Errorf("PtToCNSize(%v) = (%q, %v), want (%q, %v)", c.pt, got, ok, c.want, c.wantOK)
		}
	}
	if _, ok := CNSizeToPt("不存在的号"); ok {
		t.Error(`CNSizeToPt("不存在的号") 应返回 false`)
	}
}

// GB 页边距与 A4 版心自洽：A4 宽 − 左右边距 ≈ 版心宽。
//
// 容差 ±2 twips 的来源：TwipsPerCm=567 是精确值 1440/2.54 ≈ 566.93 的
// 取整近似，两次 CmToTwips 各引入 ≤0.5 twip 的舍入，合计 ≤1 twip；
// 取 2 留一倍安全余量。按精确值换算版心宽 = 11906 − 5.4 × 566.93 ≈ 8844.6。
func TestGBMarginA4Grid(t *testing.T) {
	margins := CmToTwips(GBMarginLeftMm/10.0) + CmToTwips(GBMarginRightMm/10.0)
	got := A4WidthTwips - margins
	const want = 8845 // 精确口径 8844.6 四舍五入
	if got < want-2 || got > want+2 {
		t.Errorf("GB 版心宽 = %d twips, want %d ±2（左边距 %dmm + 右边距 %dmm）", got, want, GBMarginLeftMm, GBMarginRightMm)
	}
}

// 封面高度预算自洽：外层表占满 A4 页高，内容上限必须严格小于它。
func TestCoverBudget(t *testing.T) {
	if BannerTableHeightTwips != A4HeightTwips {
		t.Errorf("封面外层表高度预算应等于 A4 页高：BannerTableHeightTwips=%d, A4HeightTwips=%d", BannerTableHeightTwips, A4HeightTwips)
	}
	if CoverContentMaxTwips >= BannerTableHeightTwips {
		t.Errorf("封面内容上限必须小于外层表预算：%d >= %d", CoverContentMaxTwips, BannerTableHeightTwips)
	}
}
