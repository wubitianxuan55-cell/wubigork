// 价格带推荐（v4.2a「AI 组价」底座）：由相似条目计算价格带统计量
// （Min/Max/Mean/P25/中位数/P75/离散度/离群数/置信度）与推荐价，供 AI 组价
// 与前端「价格带参考」共用。纯函数、零外部依赖；分位数口径与 costref 完全一致
// （R-7 线性插值，同 Excel PERCENTILE.INC）。
package cost

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BandSource 证据链一条:相似条目快照(不带 Body)。
type BandSource struct {
	Name      string
	Title     string
	Category  string
	Unit      string
	Spec      string
	Source    string
	Region    string
	PriceDate string
	PriceType string
	Price     float64
	UpdatedAt time.Time
}

// PriceBand 相似清单的价格带推荐。
type PriceBand struct {
	Samples    int // 参与统计的样本数
	Min        float64
	Max        float64
	Mean       float64
	Median     float64 // P50
	P25        float64
	P75        float64
	SpreadPct  float64      // 离散度 (P75-P25)/Median*100
	Outliers   int          // 超出 P25-1.5IQR / P75+1.5IQR 的样本数
	Confidence string       // 高(样本>=8)/中(4-7)/低(1-3)/(0 样本返回 nil 不产生 band)
	Sources    []BandSource // 全部参与样本
}

// ComputePriceBand 由相似条目计算价格带:
//   - unit 非空时只保留条目 Unit 与 unit 一致的样本(宽松比较:strings.TrimSpace 后
//     相等;条目 Unit 为空的不排除);unit 为空不过滤;
//   - 过滤 Price<=0 的样本;
//   - 无有效样本返回 nil;
//   - 分位数用 R-7 线性插值(与 costref 同口径):n==1 时 P25=Median=P75=该值;
//   - SpreadPct 在 Median==0 时为 0;
//   - Outliers:对排序后样本,低于 Q1-1.5*IQR 或高于 Q3+1.5*IQR 计数(IQR=Q3-Q1);
//   - Sources 按 Price 升序排列,包含全部有效样本。
func ComputePriceBand(entries []Summary, unit string) *PriceBand {
	// 1. 过滤:单位(宽松比较,条目 Unit 为空不排除)+ 有效价格(>0)。
	target := strings.TrimSpace(unit)
	var sources []BandSource
	for _, e := range entries {
		if e.Price <= 0 {
			continue
		}
		if target != "" {
			u := strings.TrimSpace(e.Unit)
			if u != "" && u != target {
				continue
			}
		}
		sources = append(sources, BandSource{
			Name:      e.Name,
			Title:     e.Title,
			Category:  e.Category,
			Unit:      e.Unit,
			Spec:      e.Spec,
			Source:    e.Source,
			Region:    e.Region,
			PriceDate: e.PriceDate,
			PriceType: e.PriceType,
			Price:     e.Price,
			UpdatedAt: e.UpdatedAt,
		})
	}
	if len(sources) == 0 {
		return nil
	}

	// 2. 按价格升序(稳定排序:同价保持输入相对顺序)。
	sort.SliceStable(sources, func(i, j int) bool { return sources[i].Price < sources[j].Price })

	// 3. 统计量。
	prices := make([]float64, len(sources))
	sum := 0.0
	for i, s := range sources {
		prices[i] = s.Price
		sum += s.Price
	}
	p25 := percentile(prices, 0.25)
	median := percentile(prices, 0.5)
	p75 := percentile(prices, 0.75)

	return &PriceBand{
		Samples:    len(prices),
		Min:        prices[0],
		Max:        prices[len(prices)-1],
		Mean:       sum / float64(len(prices)),
		Median:     median,
		P25:        p25,
		P75:        p75,
		SpreadPct:  spreadPct(p25, p75, median),
		Outliers:   countOutliers(prices, p25, p75),
		Confidence: confidenceOf(len(prices)),
		Sources:    sources,
	}
}

// RecommendPrice 推荐价:mode = median(默认)/mean/p25/p75/conservative。
// 未知 mode 按 median。返回推荐价与理由文案(中文,含样本数与置信度,
// 如「基于 12 条相似条目,中位数 ¥85.00/㎡,置信度 高」)。
// b 为 nil 时返回 (0, 空串)。
func RecommendPrice(b *PriceBand, mode string) (float64, string) {
	if b == nil {
		return 0, ""
	}
	// 未知 mode 走默认分支(median)。
	price, word := b.Median, "中位数"
	switch mode {
	case "mean":
		price, word = b.Mean, "均值"
	case "p25":
		price, word = b.P25, "P25 分位"
	case "p75":
		price, word = b.P75, "P75 分位"
	case "conservative":
		price, word = b.Min, "保守价"
	}
	// PriceBand 不携带单位字段,文案按契约示例格式省略「/单位」后缀。
	return price, fmt.Sprintf("基于 %d 条相似条目,%s ¥%.2f,置信度 %s",
		b.Samples, word, price, b.Confidence)
}

// ── 统计小工具(包内私有)──────────────────────────────────────

// percentile 线性插值分位数(R-7,与 costref.percentile 同口径,同 Excel
// PERCENTILE.INC):n==1 时任何分位都等于该唯一值;空输入返回 0(调用方保证非空)。
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	rank := p * float64(n-1)
	lo := int(rank)
	hi := lo + 1
	if hi >= n {
		hi = n - 1
	}
	frac := rank - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}

// spreadPct 离散度 (P75-P25)/Median*100;Median==0 时为 0。
func spreadPct(p25, p75, median float64) float64 {
	if median == 0 {
		return 0
	}
	return (p75 - p25) / median * 100
}

// countOutliers 统计超出 Q1-1.5*IQR / Q3+1.5*IQR 的样本数(sorted 已升序,
// IQR=Q3-Q1;含离群样本仍计入 Sources,数量由该计数承担)。
func countOutliers(sorted []float64, q1, q3 float64) int {
	iqr := q3 - q1
	lo := q1 - 1.5*iqr
	hi := q3 + 1.5*iqr
	n := 0
	for _, x := range sorted {
		if x < lo || x > hi {
			n++
		}
	}
	return n
}

// confidenceOf 置信度分档:>=8 高,4-7 中,1-3 低(0 样本不产生 band)。
func confidenceOf(samples int) string {
	switch {
	case samples >= 8:
		return "高"
	case samples >= 4:
		return "中"
	default:
		return "低"
	}
}
