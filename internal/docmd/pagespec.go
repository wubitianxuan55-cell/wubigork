package docmd

// pagespec.go — PDF 页码范围规格的解析与截断工具（docmd.go 拆分）。
// 职责：页规格（"1-5"/"1,3,5"/"3"）解析（parsePageSpecBounds/pageInRange）、
// 转换页数上限截断（capPageSpec/clampPageSpec）与 OCR 渲染边界（pageBounds）。
// 供文本提取路径（pdf.go）与 OCR 路径（ocr.go）共用。

import (
	"fmt"
	"strings"
)

// capPageSpec 应用 maxPages 上限（<=0 不限）到页码范围规格，返回收敛后的规格
// 与是否截断。给定规格整体超出上限时返回错误，避免静默输出空文档。
func capPageSpec(spec string, maxPages, total int) (string, bool, error) {
	if maxPages <= 0 {
		return spec, false, nil
	}
	last := total
	if spec != "" {
		if f, l, ok := parsePageSpecBounds(spec); ok {
			last = l
			if f > maxPages {
				return "", false, fmt.Errorf("请求页码超出转换上限（最大 %d 页）", maxPages)
			}
		}
	}
	if last <= maxPages {
		return spec, false, nil
	}
	if spec == "" {
		return fmt.Sprintf("1-%d", maxPages), true, nil
	}
	return clampPageSpec(spec, maxPages), true, nil
}

// parsePageSpecBounds 解析 "1-5"/"1,3,5"/"3" 得到覆盖的 (first,last)。
func parsePageSpecBounds(spec string) (first, last int, ok bool) {
	first, last = 1<<30, 0
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			parts := strings.SplitN(part, "-", 2)
			var s, e int
			if _, err := fmt.Sscanf(parts[0], "%d", &s); err != nil {
				continue
			}
			e = s
			if len(parts) > 1 {
				if _, err := fmt.Sscanf(parts[1], "%d", &e); err != nil {
					e = s
				}
			}
			if s < first {
				first = s
			}
			if e > last {
				last = e
			}
			ok = true
		} else {
			var p int
			if _, err := fmt.Sscanf(part, "%d", &p); err != nil {
				continue
			}
			if p < first {
				first = p
			}
			if p > last {
				last = p
			}
			ok = true
		}
	}
	if !ok {
		return 1, 0, false
	}
	return first, last, true
}

// clampPageSpec 把 "1-3,7-9,11" 这类规格裁到 max 页以内，丢弃越界段。
func clampPageSpec(spec string, max int) string {
	var parts []string
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			ps := strings.SplitN(part, "-", 2)
			var s, e int
			if _, err := fmt.Sscanf(ps[0], "%d", &s); err != nil {
				continue
			}
			e = s
			if len(ps) > 1 {
				if _, err := fmt.Sscanf(ps[1], "%d", &e); err != nil {
					e = s
				}
			}
			if s > max {
				continue
			}
			if e > max {
				e = max
			}
			if s == e {
				parts = append(parts, fmt.Sprintf("%d", s))
			} else {
				parts = append(parts, fmt.Sprintf("%d-%d", s, e))
			}
		} else {
			var p int
			if _, err := fmt.Sscanf(part, "%d", &p); err != nil {
				continue
			}
			if p <= max {
				parts = append(parts, fmt.Sprintf("%d", p))
			}
		}
	}
	return strings.Join(parts, ",")
}

// pageBounds 返回规格覆盖的 (first,last) 页边界（OCR 渲染范围用）。
func pageBounds(spec string, total int) (int, int) {
	if spec == "" {
		return 1, total
	}
	if f, l, ok := parsePageSpecBounds(spec); ok {
		return f, l
	}
	return 1, total
}

func pageInRange(page int, spec string) bool {
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			parts := strings.SplitN(part, "-", 2)
			start, end := 1, 9999
			if s, err := fmt.Sscanf(parts[0], "%d", &start); err != nil || s != 1 {
				continue
			}
			if len(parts) > 1 {
				if s, err := fmt.Sscanf(parts[1], "%d", &end); err != nil || s != 1 {
					end = start
				}
			}
			if page >= start && page <= end {
				return true
			}
		} else {
			var pn int
			if _, err := fmt.Sscanf(part, "%d", &pn); err == nil && pn == page {
				return true
			}
		}
	}
	return false
}
