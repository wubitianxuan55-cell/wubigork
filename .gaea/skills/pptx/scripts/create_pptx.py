#!/usr/bin/env python3
"""Generate a clean 16:9 .pptx deck from a JSON spec.

Usage:
    python create_pptx.py spec.json output.pptx
    python create_pptx.py --demo            # write demo.json + demo.pptx in cwd

Spec schema (UTF-8 JSON):
    {
      "title": "...", "subtitle": "...", "author": "...",
      "slides": [
        {"title": "...", "points": ["...", {"text": "...", "level": 1}], "notes": "..."}
      ]
    }
"""
import json
import os
import subprocess
import sys


DEMO_SPEC = {
    "title": "演示文稿示例",
    "subtitle": "由 gaea pptx 技能生成",
    "author": "gaea",
    "slides": [
        {"title": "为什么需要演示文稿", "points": ["结构清晰，重点突出", "一页一个观点，避免堆砌", "先大纲后成稿"]},
        {"title": "使用方法", "points": [{"text": "准备大纲与要点", "level": 0}, {"text": "生成 spec JSON", "level": 0}, {"text": "运行 create_pptx.py", "level": 0}], "notes": "演示如何生成"},
    ],
}


NAVY = (0x1F, 0x38, 0x64)
BLUE = (0x2E, 0x74, 0xB5)
GRAY = (0x40, 0x40, 0x40)
FONT = "Microsoft YaHei"


def ensure_pptx():
    try:
        import pptx  # noqa: F401
    except ImportError:
        print("[pptx] python-pptx not found, installing...", file=sys.stderr)
        subprocess.check_call([sys.executable, "-m", "pip", "install", "python-pptx"])


def style_run(run, size, bold=False, color=GRAY, name=FONT):
    from pptx.dml.color import RGBColor
    from pptx.oxml.ns import qn
    from pptx.util import Pt

    font = run.font
    font.size = Pt(size)
    font.bold = bold
    font.name = name
    font.color.rgb = RGBColor(*color)
    # Apply the same face to CJK runs (a:ea) so Chinese renders consistently.
    rPr = run._r.get_or_add_rPr()
    ea = rPr.find(qn("a:ea"))
    if ea is None:
        ea = rPr.makeelement(qn("a:ea"), {})
        rPr.append(ea)
    ea.set("typeface", name)


def add_bar(slide, left, top, width, height_in, color=BLUE):
    from pptx.dml.color import RGBColor
    from pptx.util import Inches

    bar = slide.shapes.add_shape(1, Inches(left), Inches(top), Inches(width), Inches(height_in))
    bar.fill.solid()
    bar.fill.fore_color.rgb = RGBColor(*color)
    bar.line.fill.background()
    return bar


def add_textbox(slide, left, top, width, height):
    from pptx.util import Inches
    tb = slide.shapes.add_textbox(Inches(left), Inches(top), Inches(width), Inches(height))
    tf = tb.text_frame
    tf.word_wrap = True
    return tf


def build(spec, out_path):
    ensure_pptx()
    from pptx import Presentation
    from pptx.util import Inches

    prs = Presentation()
    prs.slide_width = Inches(13.333)
    prs.slide_height = Inches(7.5)
    blank = prs.slide_layouts[6]

    title = spec.get("title") or "未命名演示文稿"
    subtitle = spec.get("subtitle") or ""
    author = spec.get("author") or ""

    # ── Cover slide ──
    cover = prs.slides.add_slide(blank)
    tf = add_textbox(cover, 0.9, 2.4, 11.5, 1.7)
    p = tf.paragraphs[0]
    r = p.add_run()
    r.text = title
    style_run(r, 40, bold=True, color=NAVY)
    if subtitle:
        tf2 = add_textbox(cover, 0.9, 4.1, 11.5, 0.7)
        p2 = tf2.paragraphs[0]
        r2 = p2.add_run()
        r2.text = subtitle
        style_run(r2, 20, color=GRAY)
    add_bar(cover, 0.92, 4.85, 2.2, 0.08)
    if author:
        tf3 = add_textbox(cover, 0.9, 6.6, 11.5, 0.5)
        p3 = tf3.paragraphs[0]
        r3 = p3.add_run()
        r3.text = author
        style_run(r3, 14, color=GRAY)

    # ── Content slides ──
    for item in spec.get("slides", []):
        slide = prs.slides.add_slide(blank)
        slide_title = item.get("title") or " "
        tf = add_textbox(slide, 0.9, 0.55, 11.5, 0.95)
        p = tf.paragraphs[0]
        r = p.add_run()
        r.text = slide_title
        style_run(r, 28, bold=True, color=NAVY)
        add_bar(slide, 0.9, 1.5, 11.5, 0.06)

        # 图表/图片：可选，位于标题栏下方居中（用于 xlsx 数据图表联动）
        image = item.get("image")
        if image and os.path.exists(image):
            try:
                slide.shapes.add_picture(image, Inches(1.4), Inches(1.9), width=Inches(10.5))
                continue
            except Exception:
                pass

        body = add_textbox(slide, 0.95, 1.9, 11.4, 5.3)
        first = True
        for point in item.get("points", []):
            if isinstance(point, dict):
                text = point.get("text", "")
                level = int(point.get("level") or 0)
            else:
                text = str(point)
                level = 0
            text = text.strip()
            if not text:
                continue
            p = body.paragraphs[0] if first else body.add_paragraph()
            first = False
            p.space_after = 9 if level == 0 else 5
            if level == 0:
                marker = p.add_run()
                marker.text = "▪ "
                style_run(marker, 18, bold=True, color=BLUE)
                run = p.add_run()
                run.text = text
                style_run(run, 18, color=GRAY)
            else:
                p.level = 1
                run = p.add_run()
                run.text = "– " + text
                style_run(run, 15, color=(0x60, 0x60, 0x60))

        notes = item.get("notes")
        if notes:
            slide.notes_slide.notes_text_frame.text = str(notes)

    os.makedirs(os.path.dirname(os.path.abspath(out_path)) or ".", exist_ok=True)
    prs.save(out_path)

    # ── Verify: reopen and count slides ──
    check = Presentation(out_path)
    return {"ok": True, "path": os.path.abspath(out_path), "slides": len(check.slides)}


def main():
    if len(sys.argv) == 2 and sys.argv[1] == "--demo":
        with open("demo.json", "w", encoding="utf-8") as f:
            json.dump(DEMO_SPEC, f, ensure_ascii=False, indent=2)
        result = build(DEMO_SPEC, "demo.pptx")
        print(json.dumps(result, ensure_ascii=False))
        return 0
    if len(sys.argv) != 3:
        print(__doc__)
        return 2
    spec_path, out_path = sys.argv[1], sys.argv[2]
    with open(spec_path, encoding="utf-8") as f:
        spec = json.load(f)
    result = build(spec, out_path)
    print(json.dumps(result, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
