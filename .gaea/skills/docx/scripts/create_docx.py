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
from datetime import date

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
# 报告 = 微软雅黑 + 蓝标题；合同 = 宋体 + 黑色标题；
# 标书 = 宋体正文 + 黑体黑标题 + 章标题另起页 + 页眉下框线 + 封面五要素
# （2026-11-21 反馈 P0-2：中文标书口径——微软雅黑直接改变页数口径（实测
# 472→419 页），字体是投标显性指标；标题黑、章另起页、页眉带框线）。
TEMPLATES = {
    "通用": {"font": "宋体", "heading": (0x1F, 0x38, 0x64), "headingFont": "", "fill": "D5E8F0"},
    "公文": {"font": "仿宋", "heading": (0, 0, 0), "headingFont": "黑体", "fill": "F2F2F2"},
    "报告": {"font": "微软雅黑", "heading": (0x2E, 0x74, 0xB5), "headingFont": "", "fill": "DCE6F1"},
    "合同": {"font": "宋体", "heading": (0, 0, 0), "headingFont": "", "fill": "F2F2F2"},
    "标书": {"font": "宋体", "heading": (0, 0, 0), "headingFont": "黑体", "fill": "F2F2F2"},
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


def add_header_footer(sec, header_text, footer_text, font, size, header_border=False):
    if header_text:
        p = sec.header.paragraphs[0]
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        set_run_font(p.add_run(header_text), font, size - 2, color=(0x60, 0x60, 0x60))
        if header_border:
            # 标书口径：页眉带下框线（P0-2）。
            pPr = p._p.get_or_add_pPr()
            pBdr = OxmlElement("w:pBdr")
            bottom = OxmlElement("w:bottom")
            bottom.set(qn("w:val"), "single")
            bottom.set(qn("w:sz"), "6")
            bottom.set(qn("w:space"), "1")
            bottom.set(qn("w:color"), "000000")
            pBdr.append(bottom)
            pPr.append(pBdr)
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


# 静态目录（P0-2 建议③：toc 不留需要 Ctrl+A→F9 的域，直接渲染条目+实测页码；
# 页码由调用方两遍法回填——第一遍占位符行数与终稿一致，分页稳定）。
def add_toc_static(doc, entries, font, size):
    from docx.enum.text import WD_TAB_ALIGNMENT, WD_TAB_LEADER
    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    set_run_font(p.add_run("目 录"), font, size + 4, bold=True)
    for e in entries:
        p = doc.add_paragraph()
        lvl = int(e.get("level") or 1)
        if lvl >= 2:
            p.paragraph_format.left_indent = Cm(0.74 * (lvl - 1))
        ts = p.paragraph_format.tab_stops
        ts.add_tab_stop(Cm(15.0), WD_TAB_ALIGNMENT.RIGHT, WD_TAB_LEADER.DOTS)
        page = str(e.get("page", "—"))
        set_run_font(p.add_run(e.get("text", "") + "\t" + page), font, size if lvl == 1 else size - 0.5, bold=(lvl == 1))
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


# 标书封面（P0-2：不留占位文字，封面五要素=正本标记/项目名/包号/投标人/日期）。
def add_cover_bid(doc, title, package, bidder, copy_mark, date_text, font, size):
    # 右上角正本/副本标记。
    if copy_mark:
        p = doc.add_paragraph()
        p.alignment = WD_ALIGN_PARAGRAPH.RIGHT
        set_run_font(p.add_run(copy_mark), font, size + 2, bold=True)
    for _ in range(5):
        doc.add_paragraph()
    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    set_run_font(p.add_run(title), "黑体", size + 14, bold=True)
    if package:
        p = doc.add_paragraph()
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        set_run_font(p.add_run(package), font, size + 4)
    for _ in range(6):
        doc.add_paragraph()
    if bidder:
        p = doc.add_paragraph()
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        set_run_font(p.add_run("投标人：" + bidder), font, size + 1)
    if date_text:
        p = doc.add_paragraph()
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        set_run_font(p.add_run(date_text), font, size + 1)
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
            # 2026-11-21 复盘 A3：图片路径解析失败不再静默降级为占位文字（旧形态
            # 回执仍报成功、机检全绿，交付过 18 张图全是失败提示的整册）。改为
            # 记账，main 里统一硬失败（exit 2），不产出残缺交付物。
            try:
                doc.add_picture(b["path"], width=Cm(14))
                doc.paragraphs[-1].alignment = WD_ALIGN_PARAGRAPH.CENTER
            except Exception as e:
                FAILED_IMAGES.append(f"{b['path']} ({e})")
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
    bid = (opts.get("template") == "标书")
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
    # 标书页眉缺省取投标人（P0-2 建议④：生成后自动补页眉）。
    header_text = opts.get("header")
    if bid and not header_text and author:
        header_text = author
    add_header_footer(sec, header_text, opts.get("footer"), font, size, header_border=bid)
    if opts.get("cover"):
        if bid:
            add_cover_bid(doc, title, subtitle, author,
                          opts.get("bidCopy") or "正本",
                          opts.get("bidDate") or date.today().strftime("%Y 年 %m 月 %d 日"),
                          font, size)
        else:
            add_cover(doc, title, subtitle, author, font, size, style["heading_color"])
    if opts.get("toc"):
        if opts.get("tocEntries"):
            add_toc_static(doc, opts["tocEntries"], font, size)
        else:
            add_toc(doc, font, size)
    if bid:
        apply_bid_heading_styles(doc, size)
    render_blocks(doc, blocks, font, size, line_spacing, style)
    # default style so hand-added paragraphs use the same font
    normal = doc.styles["Normal"]
    normal.font.name = font
    normal.font.size = Pt(size)
    normal._element.rPr.rFonts.set(qn("w:eastAsia"), font)


# 标书标题样式（P0-2）：黑体黑标题落在样式表层（run 直接格式同款双保险），
# 章标题（Heading 1）另起页——样式层的 pageBreakBefore 正是 longdoc_verify
# book 读数所查面，生成端与验证端同源。
def apply_bid_heading_styles(doc, size):
    sizes = {1: size + 4, 2: size + 2, 3: size + 0.5}
    for lvl in (1, 2, 3):
        st = doc.styles[f"Heading {lvl}"]
        st.font.name = "黑体"
        st.font.size = Pt(sizes[lvl])
        st.font.bold = True
        st.font.color.rgb = RGBColor(0, 0, 0)
        rPr = st.element.get_or_add_rPr()
        rFonts = rPr.find(qn("w:rFonts"))
        if rFonts is None:
            rFonts = OxmlElement("w:rFonts")
            rPr.append(rFonts)
        rFonts.set(qn("w:ascii"), "黑体")
        rFonts.set(qn("w:hAnsi"), "黑体")
        rFonts.set(qn("w:eastAsia"), "黑体")
        if lvl == 1:
            st.paragraph_format.page_break_before = True


# 图片加载失败记账（A3：失败即整体拒绝导出，见 main 的硬失败闸）。
FAILED_IMAGES = []


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
    ap.add_argument("--template", default="通用", help="通用|公文|报告|合同|标书")
    ap.add_argument("--subtitle", default="", help="副题（标书=包号）")
    ap.add_argument("--author", default="", help="署名（标书=投标人）")
    ap.add_argument("--bid-copy", default="正本", help="标书封面正本/副本标记")
    ap.add_argument("--bid-date", default="", help="标书封面日期（缺省=今天）")
    ap.add_argument("--toc-spec", default="", help="静态目录 JSON 文件：[{text,level,page}]，代替 TOC 域（页码由两遍法回填）")
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
        subtitle = args.subtitle
        author = args.author
        opts = {
            "cover": args.cover,
            "toc": args.toc,
            "header": args.header,
            "footer": args.footer,
            "font": args.font,
            "template": args.template,
            "bidCopy": args.bid_copy,
            "bidDate": args.bid_date,
        }
        if args.toc_spec:
            with open(args.toc_spec, encoding="utf-8") as tf:
                opts["tocEntries"] = json.load(tf)

    doc = Document()
    build(doc, title, subtitle, author, opts, blocks)
    # A3 硬失败闸：有图片没嵌进去就拒绝产出——绝不导出「图全是失败提示」的整册。
    if FAILED_IMAGES:
        print("[docx] 图片加载失败 %d 处，拒绝导出（改绝对路径/补图后重试）: %s"
              % (len(FAILED_IMAGES), "; ".join(FAILED_IMAGES)), file=sys.stderr)
        return 2
    doc.save(args.output)

    # verify by reopening
    check = Document(args.output)
    print(json.dumps({"ok": True, "path": args.output, "paragraphs": len(check.paragraphs),
                      "tables": len(check.tables), "images": len(check.inline_shapes)}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
