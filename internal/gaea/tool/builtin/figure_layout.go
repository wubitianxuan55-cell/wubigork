package builtin

import (
	"fmt"
	"strings"
)

// D4（2026-11-20 缺陷清单）：图件生成后的确定性排版体检。
//
// 视觉模型擅长"图里有什么"，不适合判断精确几何关系。chart_gen / diagram_gen
// 在 Python 渲染进程内直接读取 matplotlib Text 的 renderer bbox，做文本-文本
// 重叠、文本出界、节点框内溢出、最小字号与 bold 可用性检查；结果随工具回执
// 返回。文本与线条交叉有意不报：表格线穿过字底边等正常排版会被大量误伤。
const figureLayoutPython = `
def check_figure_layout(fig, boxed_labels=None, max_issues=12,
                        min_font_pt=6.0, overlap_threshold=0.18):
    """Return a deterministic text-layout report for a matplotlib figure.

    boxed_labels: optional list of {"name": str, "text": Text, "box": Patch}.
    Only text-text overlap is checked; text-line crossings are intentionally not.
    """
    from matplotlib.text import Text

    boxed_labels = boxed_labels or []
    fig.canvas.draw()
    renderer = fig.canvas.get_renderer()
    hidden_tick_labels = set()
    axis_decorations = set()
    for axis in fig.axes:
        axis_decorations.update(
            item for item in (
                axis.title, getattr(axis, "_left_title", None),
                getattr(axis, "_right_title", None),
                axis.xaxis.label, axis.yaxis.label
            ) if item is not None
        )
        if getattr(axis, "axison", True):
            continue
        for tick in list(axis.xaxis.get_major_ticks()) + list(axis.xaxis.get_minor_ticks()) + \
                list(axis.yaxis.get_major_ticks()) + list(axis.yaxis.get_minor_ticks()):
            hidden_tick_labels.add(tick.label1)
            hidden_tick_labels.add(tick.label2)

    def short(value, limit=46):
        value = " ".join(str(value).split())
        return value if len(value) <= limit else value[:limit - 1] + "…"

    def bbox_text(box):
        return "x=%.0f..%.0f y=%.0f..%.0f" % (box.x0, box.x1, box.y0, box.y1)

    def overlap_ratio(a, b):
        width = max(0.0, min(a.x1, b.x1) - max(a.x0, b.x0))
        height = max(0.0, min(a.y1, b.y1) - max(a.y0, b.y0))
        inter = width * height
        smaller = min(max(0.0, a.width) * max(0.0, a.height),
                      max(0.0, b.width) * max(0.0, b.height))
        return inter / smaller if smaller > 0 else 0.0

    issue_count = 0
    issues = []

    def add_issue(kind, target, detail):
        nonlocal issue_count
        issue_count += 1
        if len(issues) < max_issues:
            issues.append({"type": kind, "target": short(target),
                           "detail": short(detail, 180)})

    texts = []
    for artist in fig.findobj(Text):
        try:
            raw = artist.get_text()
            if artist in hidden_tick_labels or not artist.get_visible() or not str(raw).strip():
                continue
            box = artist.get_window_extent(renderer=renderer)
            if box.width <= 0 or box.height <= 0:
                continue
            size = float(artist.get_fontsize())
        except Exception:
            continue
        label = short(raw, 64)
        texts.append((artist, label, box))
        if size < min_font_pt - 0.01:
            add_issue("font_too_small", label,
                      "fontsize=%.1fpt < %.1fpt; %s" % (size, min_font_pt, bbox_text(box)))

    # savefig(..., bbox_inches="tight") expands the output to contain artists
    # that opt out of clipping. "Out of bounds" therefore means clipped by the
    # artist's clip box/axes, not merely outside the pre-tight figure rectangle
    # (axis labels and annotations legitimately live there).
    for artist, label, box in texts:
        if artist in axis_decorations or not artist.get_clip_on():
            continue
        clip_box = artist.get_clip_box()
        if clip_box is None and artist.axes is not None:
            clip_box = artist.axes.bbox
        if clip_box is None:
            continue
        if (box.x0 < clip_box.x0 - 1 or box.y0 < clip_box.y0 - 1 or
                box.x1 > clip_box.x1 + 1 or box.y1 > clip_box.y1 + 1):
            add_issue("text_out_of_bounds", label,
                      "text=%s clip=%s" % (bbox_text(box), bbox_text(clip_box)))

    for i in range(len(texts)):
        for j in range(i + 1, len(texts)):
            ratio = overlap_ratio(texts[i][2], texts[j][2])
            if ratio >= overlap_threshold:
                add_issue("text_overlap", texts[i][1] + " ↔ " + texts[j][1],
                          "overlap=%.0f%% (threshold %.0f%%); %s; %s" %
                          (ratio * 100, overlap_threshold * 100,
                           bbox_text(texts[i][2]), bbox_text(texts[j][2])))

    for item in boxed_labels:
        try:
            name = str(item.get("name") or "boxed text")
            text_box = item["text"].get_window_extent(renderer=renderer)
            node_box = item["box"].get_window_extent(renderer=renderer)
        except Exception:
            continue
        if (text_box.x0 < node_box.x0 - 1 or text_box.y0 < node_box.y0 - 1 or
                text_box.x1 > node_box.x1 + 1 or text_box.y1 > node_box.y1 + 1):
            add_issue("box_overflow", name,
                      "text=%s box=%s" % (bbox_text(text_box), bbox_text(node_box)))

    return {
        "status": "warn" if issue_count else "ok",
        "checked_texts": len(texts),
        "issue_count": issue_count,
        "issues": issues,
        "limits": {
            "min_font_pt": min_font_pt,
            "text_overlap_ratio": overlap_threshold,
            "text_line_crossings": "not_checked"
        }
    }
`

type figureLayoutReport struct {
	Status       string              `json:"status"`
	CheckedTexts int                 `json:"checked_texts"`
	IssueCount   int                 `json:"issue_count"`
	Issues       []figureLayoutIssue `json:"issues"`
	Limits       figureLayoutLimits  `json:"limits"`
}

type figureLayoutIssue struct {
	Type   string `json:"type"`
	Target string `json:"target"`
	Detail string `json:"detail"`
}

type figureLayoutLimits struct {
	MinFontPt         float64 `json:"min_font_pt"`
	TextOverlapRatio  float64 `json:"text_overlap_ratio"`
	TextLineCrossings string  `json:"text_line_crossings"`
}

// formatFigureLayout turns a machine-readable report into a compact receipt.
// Passing reports stay visible on purpose: the caller can distinguish "checked
// and clean" from "tool has no layout check".
func formatFigureLayout(report figureLayoutReport, fontNote string) string {
	var b strings.Builder
	switch {
	case report.Status == "":
		b.WriteString("排版自检: 未返回报告")
	case report.Status == "ok":
		fmt.Fprintf(&b, "排版自检: 通过（检查 %d 个文本元素；文本-线条交叉不判）", report.CheckedTexts)
	default:
		fmt.Fprintf(&b, "⚠ 排版自检: %d 项问题（检查 %d 个文本元素）", report.IssueCount, report.CheckedTexts)
		for i, issue := range report.Issues {
			if i >= 6 {
				fmt.Fprintf(&b, "\n  - 另有 %d 项未展开", report.IssueCount-i)
				break
			}
			fmt.Fprintf(&b, "\n  - %s：%s", layoutIssueLabel(issue.Type), issue.Target)
			if issue.Detail != "" {
				b.WriteString("；")
				b.WriteString(issue.Detail)
			}
		}
	}
	if strings.TrimSpace(fontNote) != "" {
		b.WriteString("\n⚠ ")
		b.WriteString(strings.TrimSpace(fontNote))
	}
	return b.String()
}

func layoutIssueLabel(kind string) string {
	switch kind {
	case "text_overlap":
		return "文本重叠"
	case "text_out_of_bounds":
		return "文本出界"
	case "box_overflow":
		return "框内溢出"
	case "font_too_small":
		return "字号过小"
	default:
		return kind
	}
}
