package retrieval

import (
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/cost"
)

// DocText 把成本条目摘要拼成向量化/精排文档串。
func DocText(e cost.Summary) string {
	var b strings.Builder
	b.WriteString(e.Title)
	if e.Spec != "" {
		b.WriteString("（" + e.Spec + "）")
	}
	if e.Unit != "" {
		b.WriteString(" 单位" + e.Unit)
	}
	b.WriteString(" 单价" + formatPrice(e.Price) + "元")
	if e.Category != "" {
		b.WriteString(" 分类" + e.Category)
	}
	if e.Source != "" {
		b.WriteString(" 来源" + e.Source)
	}
	if len(e.Tags) > 0 {
		b.WriteString(" 标签" + strings.Join(e.Tags, ","))
	}
	return b.String()
}

func formatPrice(v float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(v, 'f', 2, 64), "0"), ".")
}
