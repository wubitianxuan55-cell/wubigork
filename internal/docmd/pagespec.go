package docmd

// pagespec.go — PDF 页码范围规格的解析与截断工具（docmd.go 拆分）。
// 职责：页规格（"1-5"/"1,3,5"/"3"）解析（parsePageSpecBounds/pageInRange）、
// 转换页数上限截断（capPageSpec/clampPageSpec）与 OCR 渲染边界（pageBounds）。
// 供文本提取路径（pdf.go）与 OCR 路径（ocr.go）共用。

import (
	"fmt"
	"strings"
)

// capPageSpec 应用单次页数上限（<=0 不限）到页码范围规格，返回收敛后的规格
// 与是否截断。
//
// 口径（2026-11-27 AI 反馈 #6/#13）：上限卡的是**单次请求的页数**，不是页码
// 大小。此前卡的是「起始页 ≤ 上限」——676 页的扫描件转完前 500 页后，想用
// pages=501-676 补转剩余部分会被拒（「请求页码超出转换上限」），唯一绕法是自装
// pymupdf 拆文件。现在：不给 pages → 总页数超限时取前 maxPages 页并标记截断；
// 给了 pages → 任意区间都行，只要区间里的页数 ≤ maxPages；超了报错并给出分段
// 写法，不静默裁剪（静默裁剪 + 回执不报 = 残缺文本被当完整文本录进库）。
func capPageSpec(spec string, maxPages, total int) (string, bool, error) {
	if maxPages <= 0 {
		return spec, false, nil
	}
	if spec == "" {
		if total > maxPages {
			return fmt.Sprintf("1-%d", maxPages), true, nil
		}
		return spec, false, nil
	}
	f, l, ok := parsePageSpecBounds(spec)
	if !ok {
		return spec, false, nil // 解析不了：原样交给下游，由下游报页码写法错误
	}
	if n := pageSpecCount(spec, total); n > maxPages {
		return "", false, fmt.Errorf("请求的页数 %d（pages=%s）超过单次上限 %d 页：请分段转换（如 pages=%d-%d、%d-%d），或加 all_pages=true 一次转完",
			n, spec, maxPages, f, f+maxPages-1, f+maxPages, l)
	}
	return spec, false, nil
}

// pageSpecCount 规格在 total 页的文档里实际覆盖的页数（越界部分不计；total<=0
// = 页数未知，按规格字面计）。
func pageSpecCount(spec string, total int) int {
	n := 0
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
				if _, err := fmt.Sscanf(ps[1], "%d", &e); err != nil || e < s {
					e = s
				}
			}
			if total > 0 && e > total {
				e = total
			}
			if s <= e && (total <= 0 || s <= total) {
				n += e - s + 1
			}
			continue
		}
		var p int
		if _, err := fmt.Sscanf(part, "%d", &p); err == nil && p >= 1 && (total <= 0 || p <= total) {
			n++
		}
	}
	return n
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
