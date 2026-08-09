#!/usr/bin/env python3
"""Create a polished .docx from Markdown (or a JSON spec) using python-docx.

Primary usage (recommended — the agent already produces Markdown deliverables):
    python create_docx.py input.md output.docx [--title 标题] [--cover] [--toc]
                          [--header 页眉文字] [--footer 页脚文字] [--font 宋体]

JSON spec mode (fine-grained control: images, custom tables, page size):
    python create_docx.py spec.json output.docx --spec

Spec schema (UTF-8 JSON):
    {
      "title": "...", "subtitle": "...", "author": "...",
      "pageSize": "A4" | "Letter",
      "cover": true, "toc": true,
      "header": "...", "footer": "第 {page} 页",
      "font": "宋体", "fontSize": 12, "lineSpacing": 1.5,
      "blocks": [
        {"type": "heading1|heading2|heading3|paragraph", "text": "..."},
        {"type": "list", "ordered": false, "items": ["a", "b"]},
        {"type": "table", "header": ["c1", "c2"], "rows": [["a", "b"]]},
        {"type": "image", "path": "img.png"},
        {"type": "pagebreak"}
      ]
    }

Markdown mode parses: #/##/### headings, paragraphs, |-| tables,
-/1. lists, ``` fenced code, and images ![alt](path).
"""
import argparse
import json
import re
import sys

try:
    import docx
except ImportError:
    import subprocess
    print("[docx] python-docx not found, installing...", file=sys.stderr)
    subprocess.check_call([sys.executable, "-m", "pip", "install", "python-docx"])

from docx import Document
from docx.enum.section import WD_SECTION
from docx.enum.table import WD_TABLE_ALIGNMENT
from docx.enum.text import WD_ALIGN_PARAGRAPH, WD_LINE_SPACING
from docx.oxml import OxmlElement
from docx.oxml.ns import qn
from docx.shared import Cm, Inches, Pt, RGBColor


FALLBACK_FONT = "宋体"

# 模板预设：字体 / 标题色 / 标题字体 / 表头底色。
# 通用 = 宋体 + 深蓝标题；公文 = 仿宋正文 + 黑体标题（黑色）；
# 报告 = 微软雅黑 + 蓝标题；合同 = 宋体 + 黑色标题。
TEMPLATES = {
    "通用": {"font": "宋体", "heading": (0x1F, 0x38, 0x64), "headingFont": "", "fill": "D5E8F0"},
    "公文": {"font": "仿宋", "heading": (0, 0, 0), "headingFont": "黑体", "fill": "F2F2F2"},
    "报告": {"font": "微软雅黑", "heading": (0x2E, 0x74, 0xB5), "headingFont": "", "fill": "DCE6F1"},
    "合同": {"font": "宋体", "heading": (0, 0, 0), "headingFont": "", "fill": "F2F2F2"},
}


def set_run_font(run, name, size, bold=False, color=None):
    run.font.name = name
    run.font.size = Pt(size)
    run.font.bold = bold
    if color:
        run.font.color.rgb = RGBColor(*color)
    # East-Asian font must be set explicitly or Chinese renders in the fallback face
    rPr = run._element.get_or_add_rPr()
    rFonts = rPr.find(qn("w:rFonts"))
    if rFonts is None:
        rFonts = OxmlElement("w:rFonts")
        rPr.append(rFonts)
    rFonts.set(qn("w:ascii"), name)
    rFonts.set(qn("w:hAnsi"), name)
    rFonts.set(qn("w:eastAsia"), name)


def add_field(paragraph, instr):
    """Insert a Word field (e.g. PAGE) into a paragraph."""
    r = paragraph.add_run()
    fldChar1 = OxmlElement("w:fldChar")
    fldChar1.set(qn("w:fldCharType"), "begin")
    instrText = OxmlElement("w:instrText")
    instrText.set(qn("xml:space"), "preserve")
    instrText.text = instr
    fldChar2 = OxmlElement("w:fldChar")
    fldChar2.set(qn("w:fldCharType"), "end")
    r._r.append(fldChar1)
    r._r.append(instrText)
    r._r.append(fldChar2)
    return r


def setup_section(doc, page_size, margins_cm):
    sec = doc.sections[0]
    if page_size == "Letter":
        sec.page_width, sec.page_height = Inches(8.5), Inches(11)
    else:
        sec.page_width, sec.page_height = Cm(21.0), Cm(29.7)
    sec.top_margin = Cm(margins_cm[0])
    sec.bottom_margin = Cm(margins_cm[1])
    sec.left_margin = Cm(margins_cm[2])
    sec.right_margin = Cm(margins_cm[3])
    return sec


def add_header_footer(sec, header_text, footer_text, font, size):
    if header_text:
        p = sec.header.paragraphs[0]
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        set_run_font(p.add_run(header_text), font, size - 2, color=(0x60, 0x60, 0x60))
    if footer_text:
        p = sec.footer.paragraphs[0]
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        if "{page}" in footer_text:
            before, after = footer_text.split("{page}", 1)
            if before:
                set_run_font(p.add_run(before), font, size - 2, color=(0x60, 0x60, 0x60))
            add_field(p, "PAGE")
            if after:
                set_run_font(p.add_run(after), font, size - 2, color=(0x60, 0x60, 0x60))
        else:
            set_run_font(p.add_run(footer_text), font, size - 2, color=(0x60, 0x60, 0x60))


def add_toc(doc, font, size):
    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    set_run_font(p.add_run("目 录"), font, size + 4, bold=True)
    toc = doc.add_paragraph()
    add_field(toc, r'TOC \o "1-3" \h \z \u')
    # TOC needs a page break after it
    doc.add_page_break()


def add_cover(doc, title, subtitle, author, font, size, heading_color):
    for _ in range(6):
        doc.add_paragraph()
    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    set_run_font(p.add_run(title), font, size + 10, bold=True, color=heading_color)
    if subtitle:
        p = doc.add_paragraph()
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        set_run_font(p.add_run(subtitle), font, size + 2, color=(0x40, 0x40, 0x40))
    if author:
        for _ in range(4):
            doc.add_paragraph()
        p = doc.add_paragraph()
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        set_run_font(p.add_run(author), font, size - 1, color=(0x60, 0x60, 0x60))
    doc.add_page_break()


def add_table(doc, header, rows, font, size, fill):
    ncols = len(header)
    if ncols == 0:
        return
    table = doc.add_table(rows=1, cols=ncols)
    table.style = "Table Grid"
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    hdr = table.rows[0].cells
    for i, h in enumerate(header):
        hdr[i].text = ""
        p = hdr[i].paragraphs[0]
        set_run_font(p.add_run(str(h)), font, size, bold=True)
        # light shading on header row
        tcPr = hdr[i]._tc.get_or_add_tcPr()
        shd = OxmlElement("w:shd")
        shd.set(qn("w:val"), "clear")
        shd.set(qn("w:fill"), fill)
        tcPr.append(shd)
    for row in rows:
        cells = table.add_row().cells
        for i, val in enumerate(row):
            if i >= ncols:
                break
            cells[i].text = ""
            p = cells[i].paragraphs[0]
            set_run_font(p.add_run(str(val)), font, size - 0.5)
    doc.add_paragraph()


def render_blocks(doc, blocks, font, size, line_spacing, style):
    heading_color = style["heading_color"]
    heading_font = style["heading_font"] or font
    table_fill = style["table_fill"]
    for b in blocks:
        btype = b.get("type", "paragraph")
        if btype in ("heading1", "heading2", "heading3"):
            level = int(btype[-1])
            sizes = {1: size + 4, 2: size + 2, 3: size + 0.5}
            p = doc.add_paragraph()
            p.paragraph_format.space_before = Pt(12 if level <= 2 else 8)
            p.paragraph_format.space_after = Pt(6)
            set_run_font(p.add_run(b.get("text", "")), heading_font, sizes[level], bold=True, color=heading_color)
            p.style = doc.styles[f"Heading {level}"]
            for r in p.runs:
                set_run_font(r, heading_font, sizes[level], bold=True, color=heading_color)
        elif btype == "paragraph":
            p = doc.add_paragraph()
            p.paragraph_format.line_spacing = line_spacing
            set_run_font(p.add_run(b.get("text", "")), font, size)
        elif btype == "list":
            for item in b.get("items", []):
                p = doc.add_paragraph(style="List Number" if b.get("ordered") else "List Bullet")
                p.paragraph_format.line_spacing = line_spacing
                set_run_font(p.add_run(str(item)), font, size)
        elif btype == "table":
            add_table(doc, b.get("header", []), b.get("rows", []), font, size, table_fill)
        elif btype == "image":
            try:
                doc.add_picture(b["path"], width=Cm(14))
                doc.paragraphs[-1].alignment = WD_ALIGN_PARAGRAPH.CENTER
            except Exception as e:
                doc.add_paragraph(f"[图片加载失败: {e}]")
        elif btype == "pagebreak":
            doc.add_page_break()
        elif btype == "code":
            p = doc.add_paragraph()
            p.paragraph_format.line_spacing = 1.15
            set_run_font(p.add_run(b.get("text", "")), "Consolas", size - 1)
            pPr = p._p.get_or_add_pPr()
            shd = OxmlElement("w:shd")
            shd.set(qn("w:val"), "clear")
            shd.set(qn("w:fill"), "F2F2F2")
            pPr.append(shd)


MARKDOWN_TABLE_RE = re.compile(r"^\s*\|.*\|\s*$")


def parse_markdown(md_text):
    """Parse Markdown into blocks: headings, paragraphs, lists, tables, code."""
    blocks = []
    lines = md_text.splitlines()
    i = 0
    while i < len(lines):
        line = lines[i]
        stripped = line.strip()
        if not stripped:
            i += 1
            continue
        # fenced code
        if stripped.startswith("```"):
            code = []
            i += 1
            while i < len(lines) and not lines[i].strip().startswith("```"):
                code.append(lines[i])
                i += 1
            i += 1
            blocks.append({"type": "code", "text": "\n".join(code)})
            continue
        # headings
        m = re.match(r"^(#{1,3})\s+(.*)$", stripped)
        if m:
            blocks.append({"type": f"heading{len(m.group(1))}", "text": m.group(2).strip()})
            i += 1
            continue
        # table (collect consecutive |-| rows)
        if MARKDOWN_TABLE_RE.match(stripped):
            rows = []
            while i < len(lines) and MARKDOWN_TABLE_RE.match(lines[i].strip()):
                cells = [c.strip() for c in lines[i].strip().strip("|").split("|")]
                rows.append(cells)
                i += 1
            if len(rows) >= 2:
                # row 1 = header, row 2 = separator (---) — drop it
                header = rows[0]
                data = [r for r in rows[2:]] if len(rows) > 2 else []
                blocks.append({"type": "table", "header": header, "rows": data})
            continue
        # list (bulleted or numbered), collect consecutive
        list_m = re.match(r"^(\s*)([-*+]|\d+[.)])\s+(.*)$", stripped)
        if list_m and (list_m.group(2) in ("-", "*", "+") or list_m.group(2)[0].isdigit()):
            ordered = list_m.group(2)[0].isdigit()
            items = []
            while i < len(lines):
                lm = re.match(r"^\s*([-*+]|\d+[.)])\s+(.*)$", lines[i].strip())
                if not lm:
                    break
                items.append(lm.group(2))
                i += 1
            blocks.append({"type": "list", "ordered": ordered, "items": items})
            continue
        # image line
        img = re.match(r"^!\[([^\]]*)\]\(([^)]+)\)\s*$", stripped)
        if img:
            blocks.append({"type": "image", "path": img.group(2), "alt": img.group(1)})
            i += 1
            continue
        # paragraph — merge consecutive non-empty lines
        para = [stripped]
        i += 1
        while i < len(lines):
            s = lines[i].strip()
            if not s or s.startswith("#") or s.startswith("```") or MARKDOWN_TABLE_RE.match(s) or re.match(r"^([-*+]|\d+[.)])\s+", s):
                break
            para.append(s)
            i += 1
        blocks.append({"type": "paragraph", "text": "\n".join(para)})
    return blocks


def build(doc, title, subtitle, author, opts, blocks):
    tmpl = TEMPLATES.get(opts.get("template") or "通用") or TEMPLATES["通用"]
    font = opts.get("font") or tmpl["font"] or FALLBACK_FONT
    style = {
        "heading_color": tmpl["heading"],
        "heading_font": tmpl.get("headingFont") or "",
        "table_fill": tmpl.get("fill") or "D5E8F0",
    }
    size = float(opts.get("fontSize") or 12)
    line_spacing = float(opts.get("lineSpacing") or 1.5)
    margins_cm = opts.get("marginsCm") or [2.54, 2.54, 3.17, 3.17]
    page_size = opts.get("pageSize") or "A4"
    sec = setup_section(doc, page_size, margins_cm)
    add_header_footer(sec, opts.get("header"), opts.get("footer"), font, size)
    if opts.get("cover"):
        add_cover(doc, title, subtitle, author, font, size, style["heading_color"])
    if opts.get("toc"):
        add_toc(doc, font, size)
    render_blocks(doc, blocks, font, size, line_spacing, style)
    # default style so hand-added paragraphs use the same font
    normal = doc.styles["Normal"]
    normal.font.name = font
    normal.font.size = Pt(size)
    normal._element.rPr.rFonts.set(qn("w:eastAsia"), font)


def main():
    ap = argparse.ArgumentParser(description="Create a .docx from Markdown or JSON spec")
    ap.add_argument("input", help="Markdown (.md) or JSON spec file")
    ap.add_argument("output", help="output .docx path")
    ap.add_argument("--spec", action="store_true", help="input is a JSON spec")
    ap.add_argument("--title", default="")
    ap.add_argument("--cover", action="store_true")
    ap.add_argument("--toc", action="store_true")
    ap.add_argument("--header", default="")
    ap.add_argument("--footer", default="第 {page} 页")
    ap.add_argument("--font", default="")
    ap.add_argument("--template", default="通用", help="通用|公文|报告|合同")
    args = ap.parse_args()

    with open(args.input, encoding="utf-8") as f:
        raw = f.read()

    if args.spec:
        spec = json.loads(raw)
        opts = spec
        title = spec.get("title") or "未命名文档"
        subtitle = spec.get("subtitle") or ""
        author = spec.get("author") or ""
        blocks = spec.get("blocks", [])
    else:
        blocks = parse_markdown(raw)
        title = args.title or next((b["text"] for b in blocks if b["type"] == "heading1"), "未命名文档")
        subtitle = ""
        author = ""
        opts = {
            "cover": args.cover,
            "toc": args.toc,
            "header": args.header,
            "footer": args.footer,
            "font": args.font,
            "template": args.template,
        }

    doc = Document()
    build(doc, title, subtitle, author, opts, blocks)
    doc.save(args.output)

    # verify by reopening
    check = Document(args.output)
    print(json.dumps({"ok": True, "path": args.output, "paragraphs": len(check.paragraphs), "tables": len(check.tables)}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
