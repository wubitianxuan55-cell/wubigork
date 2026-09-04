package contextview

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// TestEstimateImageTokensOfficialExample 用官方文档样例锚定口径：
// 1000×1000 → 1296 tokens（platform.claude.com/docs/en/build-with-claude/vision，
// 2026-09 核实）。28×28 px = 1 token，标准档长边 1568 不触发缩放。
func TestEstimateImageTokensOfficialExample(t *testing.T) {
	est := EstimateImageTokens(1000, 1000)
	if est.StdTokens != 1296 {
		t.Fatalf("1000x1000 stdTokens = %d, want 1296（官方例）", est.StdTokens)
	}
	if est.ScaledW != 1000 || est.ScaledH != 1000 {
		t.Fatalf("1000x1000 不应缩放，got %dx%d", est.ScaledW, est.ScaledH)
	}
}

func TestEstimateImageTokensPatchRounding(t *testing.T) {
	// ⌈28/28⌉×⌈28/28⌉ = 1；不足一 patch 的边缘向上取整。
	if got := patchTokens(28, 28); got != 1 {
		t.Fatalf("28x28 = %d, want 1", got)
	}
	if got := patchTokens(29, 1); got != 2 {
		t.Fatalf("29x1 = %d, want 2（⌈29/28⌉=2 × ⌈1/28⌉=1）", got)
	}
}

func TestEstimateImageTokensTierScalingAndCaps(t *testing.T) {
	// 大图：4000×3000 标准档先缩到 1568×1176（56×42=2352 patch）再封顶 1568。
	est := EstimateImageTokens(4000, 3000)
	if est.ScaledW != 1568 || est.ScaledH != 1176 {
		t.Fatalf("4000x3000 标准档缩放 = %dx%d, want 1568x1176", est.ScaledW, est.ScaledH)
	}
	if est.StdTokens != 1568 {
		t.Fatalf("4000x3000 stdTokens = %d, want 1568（封顶）", est.StdTokens)
	}
	// 高分辨率档：长边 2576 → 2576×1932 → 92×69=6348 → 封顶 4784。
	if est.HighTokens != 4784 {
		t.Fatalf("4000x3000 highTokens = %d, want 4784（封顶）", est.HighTokens)
	}
	// 小图两档都不缩放：500×200 → ⌈500/28⌉=18 × ⌈200/28⌉=8 = 144。
	small := EstimateImageTokens(500, 200)
	if small.StdTokens != 144 || small.HighTokens != 144 {
		t.Fatalf("500x200 = std %d / high %d, want 144/144", small.StdTokens, small.HighTokens)
	}
	if small.ScaledW != 500 || small.ScaledH != 200 {
		t.Fatalf("500x200 不应缩放，got %dx%d", small.ScaledW, small.ScaledH)
	}
}

func TestExtractImageRefs(t *testing.T) {
	refs := ExtractImageRefs(`看这张 @C:\data\报告图.png 和 out\chart.JPG，以及 https://x.com/a.png?t=1 与 notes.md`)
	// 注：https 带查询参数不以扩展名收尾 → 不入选；md 不入选。
	if len(refs) != 2 {
		t.Fatalf("refs = %v, want 2", refs)
	}
	if refs[0] != `C:\data\报告图.png` {
		t.Fatalf("refs[0] = %q", refs[0])
	}
	if refs[1] != `out\chart.JPG` {
		t.Fatalf("refs[1] = %q（扩展名大小写不敏感）", refs[1])
	}

	// markdown 图片语法：括号是切词符，路径可被提出。
	md := ExtractImageRefs(`生成结果 ![图表](out/plot.webp) 请查收`)
	if len(md) != 1 || md[0] != "out/plot.webp" {
		t.Fatalf("md refs = %v", md)
	}

	// 去重 + 上限。
	many := ExtractImageRefs("a.png b.png c.png d.png e.png a.png")
	if len(many) != 4 {
		t.Fatalf("many = %v, want 4（上限）", many)
	}
}

func TestExtractImageRefsFromArgs(t *testing.T) {
	// JSON 字符串里的 \\ 转义解析后是单反斜杠。
	args := `{"image_path":"C:\\imgs\\wx.png","prompt":"描述图片","ratio":[1,2]}`
	refs := ExtractImageRefsFromArgs(args)
	if len(refs) != 1 || refs[0] != `C:\imgs\wx.png` {
		t.Fatalf("refs = %v", refs)
	}
	// 非 JSON：诚实不猜（原始 JSON 的 \\ 转义会路径失配）。
	if got := ExtractImageRefsFromArgs(`image_path=C:\\x.png`); got != nil {
		t.Fatalf("非 JSON 应返回 nil, got %v", got)
	}
	// 数组参数（multi 形状）+ 键序稳定。
	arr := ExtractImageRefsFromArgs(`{"paths":["a/1.png","b/2.svg","c/3.txt"]}`)
	if len(arr) != 2 || arr[0] != "a/1.png" || arr[1] != "b/2.svg" {
		t.Fatalf("arr = %v", arr)
	}
}

func TestCountImageRefs(t *testing.T) {
	if got := countImageRefs(`@C:\a.png 看 @C:\a.png 和 b.jpg`); got != 2 {
		t.Fatalf("count = %d, want 2（同条内去重）", got)
	}
	if got := countImageRefs("没有图片 refs.md"); got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
}

func TestFoldImagesStat(t *testing.T) {
	// stats.Images 原为恒 0 死字段；2.5b 后半起按「引用出现次数」诚实计数。
	entries := []session.LogEntry{
		entry(1, "user_message", map[string]any{"content": "看这张 @C:/x/a.png"}),
		entry(2, "tool_dispatch", map[string]any{"id": "t1", "name": "vision", "args": `{"image_path":"C:\\x\\b.jpg"}`, "partial": false}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if tl.Stats.Images != 2 {
		t.Fatalf("stats.Images = %d, want 2", tl.Stats.Images)
	}
}
