"""Deterministic text-layout checker for matplotlib engineering figures.

Import this module from a custom drawing script, call check_figure_layout(fig,
boxed_labels) after tight_layout() and before savefig(), then include the report
in the figure receipt.

Only text-text overlap is checked. Text-line crossings are intentionally not
reported because table rules, leaders, and intentional annotation lines commonly
touch text without being layout defects.
"""


def check_figure_layout(fig, boxed_labels=None, max_issues=12,
                        min_font_pt=6.0, overlap_threshold=0.18):
    """Return a JSON-serializable deterministic layout report."""
    from matplotlib.text import Text

    boxed_labels = boxed_labels or []
    fig.canvas.draw()
    renderer = fig.canvas.get_renderer()
    hidden_tick_labels = set()
    axis_decorations = set()
    for axis in fig.axes:
        axis_decorations.update(
            item for item in (
                axis.title,
                getattr(axis, "_left_title", None),
                getattr(axis, "_right_title", None),
                axis.xaxis.label,
                axis.yaxis.label,
            ) if item is not None
        )
        if getattr(axis, "axison", True):
            continue
        ticks = (
            list(axis.xaxis.get_major_ticks())
            + list(axis.xaxis.get_minor_ticks())
            + list(axis.yaxis.get_major_ticks())
            + list(axis.yaxis.get_minor_ticks())
        )
        for tick in ticks:
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
            issues.append({
                "type": kind,
                "target": short(target),
                "detail": short(detail, 180),
            })

    texts = []
    for artist in fig.findobj(Text):
        try:
            raw = artist.get_text()
            if (artist in hidden_tick_labels or not artist.get_visible()
                    or not str(raw).strip()):
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
                      "fontsize=%.1fpt < %.1fpt; %s"
                      % (size, min_font_pt, bbox_text(box)))

    # savefig(..., bbox_inches="tight") expands to include unclipped artists.
    # A text is out of bounds only when its enabled clip box actually cuts it.
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
                      "text=%s clip=%s"
                      % (bbox_text(box), bbox_text(clip_box)))

    for i in range(len(texts)):
        for j in range(i + 1, len(texts)):
            ratio = overlap_ratio(texts[i][2], texts[j][2])
            if ratio >= overlap_threshold:
                add_issue(
                    "text_overlap",
                    texts[i][1] + " ↔ " + texts[j][1],
                    "overlap=%.0f%% (threshold %.0f%%); %s; %s"
                    % (ratio * 100, overlap_threshold * 100,
                       bbox_text(texts[i][2]), bbox_text(texts[j][2])),
                )

    for item in boxed_labels:
        try:
            name = str(item.get("name") or "boxed text")
            text_box = item["text"].get_window_extent(renderer=renderer)
            node_box = item["box"].get_window_extent(renderer=renderer)
        except Exception:
            continue
        if (text_box.x0 < node_box.x0 - 1 or
                text_box.y0 < node_box.y0 - 1 or
                text_box.x1 > node_box.x1 + 1 or
                text_box.y1 > node_box.y1 + 1):
            add_issue(
                "box_overflow",
                name,
                "text=%s box=%s"
                % (bbox_text(text_box), bbox_text(node_box)),
            )

    return {
        "status": "warn" if issue_count else "ok",
        "checked_texts": len(texts),
        "issue_count": issue_count,
        "issues": issues,
        "limits": {
            "min_font_pt": min_font_pt,
            "text_overlap_ratio": overlap_threshold,
            "text_line_crossings": "not_checked",
        },
    }


def format_layout_report(report):
    """Return a compact human-readable receipt line."""
    if report.get("status") == "ok":
        return "排版自检: 通过（%d 个文本元素）" % report.get("checked_texts", 0)
    lines = [
        "⚠ 排版自检: %d 项问题（%d 个文本元素）"
        % (report.get("issue_count", 0), report.get("checked_texts", 0))
    ]
    for issue in report.get("issues", [])[:6]:
        lines.append("- %s: %s; %s"
                     % (issue.get("type"), issue.get("target"),
                        issue.get("detail")))
    return "\n".join(lines)
