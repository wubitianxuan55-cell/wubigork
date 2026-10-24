package typography

// 中文字号 ↔ 磅值对照：号 → pt。号值互不相同，是字号转换与机检共用的
// 单一真相源（出处：docx/references/common-rules.md 中文字号档位，
// 初号~八号含小号档，共 16 档）。

import (
	"math"
	"sort"
)

// CNSizeNames 中文字号 → 磅值（如 "五号" → 10.5）。
var CNSizeNames = map[string]float64{
	"初号": 42, "小初": 36,
	"一号": 26, "小一": 24,
	"二号": 22, "小二": 18,
	"三号": 16, "小三": 15,
	"四号": 14, "小四": 12,
	"五号": 10.5, "小五": 9,
	"六号": 7.5, "小六": 6.5,
	"七号": 5.5, "八号": 5,
}

// cnSize 一档字号（名称 + 磅值）。
type cnSize struct {
	name string
	pt   float64
}

// cnSizesByPt 按 pt 升序的档位表，供 PtToCNSize 做确定性最近档查找。
var cnSizesByPt = sortedCNSizes()

func sortedCNSizes() []cnSize {
	out := make([]cnSize, 0, len(CNSizeNames))
	for name, pt := range CNSizeNames {
		out = append(out, cnSize{name: name, pt: pt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pt < out[j].pt })
	return out
}

// CNSizeToPt 中文名 → 磅值；查无此号第二个返回值为 false。
func CNSizeToPt(name string) (float64, bool) {
	pt, ok := CNSizeNames[name]
	return pt, ok
}

// PtToCNSize 磅值 → 最近的中文字号档。超出档位覆盖区间 [八号 5pt,
// 初号 42pt]（含边界）或输入 NaN 时返回 false；区间内取 pt 距离最近的档，
// 等距时取较小档（确定性，不依赖 map 遍历顺序）。
func PtToCNSize(pt float64) (string, bool) {
	if len(cnSizesByPt) == 0 || math.IsNaN(pt) {
		return "", false
	}
	lo, hi := cnSizesByPt[0], cnSizesByPt[len(cnSizesByPt)-1]
	if pt < lo.pt || pt > hi.pt {
		return "", false
	}
	best, bestDist := 0, math.Abs(pt-cnSizesByPt[0].pt)
	for i := 1; i < len(cnSizesByPt); i++ {
		if d := math.Abs(pt - cnSizesByPt[i].pt); d < bestDist { // 严格小于：等距保留较小档
			best, bestDist = i, d
		}
	}
	return cnSizesByPt[best].name, true
}
