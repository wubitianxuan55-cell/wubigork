package typography

// 中文排版阈值表（K2）：把散落在技能文本里的排版数值收敛成一个可断言的
// Go 包。纯常量 + 纯函数，无 I/O、无第三方依赖，供技能层与机检共用一个
// 真相源。
//
// 数值出处：
//   - GB/T 9704-2012《党政机关公文格式》第 5 节（页边距与版面规格）；
//   - ZCode 技能参考 docx/references/common-rules.md（字号/行距/缩进惯用值）。

import "math"

// 单位换算基准。
const (
	// TwipsPerCm 1cm = 567 twips（精确值 1440/2.54 ≈ 566.93，业界惯用取整）。
	TwipsPerCm = 567

	// TwipsPerInch 1 inch = 1440 twips（OOXML 精确定义）。
	TwipsPerInch = 1440

	// TwipsPerPt 1pt = 20 twips（w:spacing / w:ind 的长度单位）。
	TwipsPerPt = 20

	// HalfPointsPerPt 1pt = 2 half-points（w:sz / w:szCs 以半点计）。
	// 注意：这与 TwipsPerPt=20 不是同一个数——把 1pt 当 20 半点是 w:sz 的
	// 经典错误（10.5pt 会被写成 210 而不是 21），K2 首版即因此挂过一次测试。
	HalfPointsPerPt = 2
)

// 版面与行距。
const (
	// A4WidthTwips / A4HeightTwips A4 版面（210mm × 297mm）。
	A4WidthTwips  = 11906
	A4HeightTwips = 16838

	// LineSpacing13x 1.3 倍行距的 Word line 值（line 以 240 分之一行为
	// 单位：240 × 1.3 = 312，w:spacing/@w:line）。
	LineSpacing13x = 312

	// BodyIndentSimSun 正文首行缩进 2 字符（宋体小四 12pt：2 × 12pt = 480 twips）。
	BodyIndentSimSun = 480

	// BodyIndentYaHei 正文首行缩进 2 字符（雅黑 10.5pt：2 × 10.5pt = 420 twips）。
	BodyIndentYaHei = 420
)

// GB/T 9704-2012 第 5 节页边距（毫米）：上 37、下 35、左 28、右 26。
const (
	GBMarginTopMm    = 37
	GBMarginBottomMm = 35
	GBMarginLeftMm   = 28
	GBMarginRightMm  = 26
)

// 封面高度预算。
const (
	// BannerTableHeightTwips 封面外层表高度预算 = A4 整页高。
	BannerTableHeightTwips = 16838

	// CoverContentMaxTwips 封面内容高度上限 = 页高 − 顶部预留 1200 twips（60pt）。
	CoverContentMaxTwips = 15638
)

// CmToTwips 厘米 → twips，四舍五入。零进零出、负进负出（线性换算，不取
// 绝对值）。
func CmToTwips(cm float64) int {
	return int(math.Round(cm * TwipsPerCm))
}

// PtToHalfPoints 磅 → 半点（w:sz 单位），四舍五入。
func PtToHalfPoints(pt float64) int {
	return int(math.Round(pt * HalfPointsPerPt))
}

// PtToTwips 磅 → twips（1pt = 20 twips），四舍五入。
func PtToTwips(pt float64) int {
	return int(math.Round(pt * TwipsPerPt))
}
