package contextview

// 图片 token 估算（2.5b 后半）：官方 patch 口径——28×28 像素 = 1 个视觉
// token，先按档位把长边缩放到上限再计，最后按档位封顶。来源：Anthropic
// 官方文档 platform.claude.com/docs/en/build-with-claude/vision（2026-09
// 核实，官方例：1000×1000 → 1296 tokens）。社区旧式 (w×h)/750 是 28²=784
// px² 网格的连续近似且忽略档位缩放，系统性高估大图，弃用。
//
// gaea 的识图链路面向 OpenAI 兼容端点（vision 链路不动），各供应商真实
// 计费口径不同——本估算作为展示层的统一近似（标准档），诚实标注口径，
// 不伪造精确计费。
const (
	// 标准档：多数模型，长边缩到 ≤1568px，最终 token 数上限 1568。
	imgStdMaxEdge = 1568
	imgStdMaxToks = 1568
	// 高分辨率档（Claude 4.7+），长边 ≤2576px，上限 4784 tokens。
	imgHighMaxEdge = 2576
	imgHighMaxToks = 4784
	// patch 边长：28×28 像素 = 1 token。
	imgPatchPx = 28
)

// ImageTokenEstimate 是一张图片按官方 patch 口径的双档估算。
type ImageTokenEstimate struct {
	// ScaledW/ScaledH 是标准档缩放后的尺寸（缩略卡「缩放后尺寸→实际计费
	// token」成对展示的尺寸半边；原图更小则等于原尺寸）。
	ScaledW int `json:"scaledW,omitempty"`
	ScaledH int `json:"scaledH,omitempty"`
	// StdTokens 标准档估算；HighTokens 高分辨率档估算（悬停详情）。
	StdTokens  int64 `json:"stdTokens,omitempty"`
	HighTokens int64 `json:"highTokens,omitempty"`
}

// EstimateImageTokens 按官方 patch 口径估算图片 token（w/h 需 >0）。
func EstimateImageTokens(w, h int) ImageTokenEstimate {
	sw, sh := scaleToEdge(w, h, imgStdMaxEdge)
	hw, hh := scaleToEdge(w, h, imgHighMaxEdge)
	std := capTokens(patchTokens(sw, sh), imgStdMaxToks)
	high := capTokens(patchTokens(hw, hh), imgHighMaxToks)
	return ImageTokenEstimate{ScaledW: sw, ScaledH: sh, StdTokens: std, HighTokens: high}
}

// scaleToEdge 等比缩放使长边 = maxEdge（短边同比、四舍五入）；原图长边
// 已 ≤ maxEdge 时不放大原样返回。
func scaleToEdge(w, h, maxEdge int) (int, int) {
	if w <= 0 || h <= 0 {
		return w, h
	}
	long := w
	if h > long {
		long = h
	}
	if long <= maxEdge {
		return w, h
	}
	sw := w * maxEdge / long
	sh := h * maxEdge / long
	if sw < 1 {
		sw = 1
	}
	if sh < 1 {
		sh = 1
	}
	return sw, sh
}

// patchTokens = ⌈w/28⌉ × ⌈h/28⌉（每 28×28 像素 patch 计 1 token，不足一
// patch 的边缘向上取整）。
func patchTokens(w, h int) int64 {
	cw := (w + imgPatchPx - 1) / imgPatchPx
	ch := (h + imgPatchPx - 1) / imgPatchPx
	return int64(cw) * int64(ch)
}

// capTokens 按档位上限封顶（官方口径：缩放计数后最终 token 数 capped）。
func capTokens(v int64, cap int64) int64 {
	if v > cap {
		return cap
	}
	return v
}
