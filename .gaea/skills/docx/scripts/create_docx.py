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

# 标题多级编号（2026-11-24：编号从标题字面文本改为 WPS/Word 原生多级列表——
# Heading 1/2/3 样式层挂 numPr 链接 multilevel abstractNum，WPS 里插删章节编号
# 自动重排；解析时剥标题字面前缀，渲染外观不变（lvlText 同款）。None=不挂自动
# 编号：公文走 GB/T 9704 字面口径（一、/（一）/1. 是规范的一部分，不联动）。
_NUM_STD = ["第%1章", "%1.%2", "%1.%2.%3"]
HEADING_NUMBERING = {
    "通用": _NUM_STD,
    "报告": _NUM_STD,
    "合同": _NUM_STD,
    "标书": _NUM_STD,
    "公文": None,
}

# 标题字面编号前缀（与 internal/app office 侧 longdocStripHeadingNumber 同源镜像；
# 顺序即剥离优先级，循环最多两遍吃「第1章 1.1」）。裸数字空格层（「1 概述」）
# 单列并带语义后过滤——「2026 年度」「12 月进度」这类以数字开头的非编号标题
# 不许误剥（Go RE2 无先行断言，两边都用「剥后看首字」实现）。
HEADING_PREFIX_RES = [
    re.compile(r"^第\s*[0-9０-９一二三四五六七八九十百]+\s*[章篇部]\s*"),
    re.compile(r"^[0-9０-９]+(?:\s*[.、]\s*[0-9０-９]+)+\s+"),
    re.compile(r"^[0-9０-９]+(?:\s*[.、]\s*[0-9０-９]+)*[、]\s*"),
    re.compile(r"^[（(]\s*[0-9０-９一二三四五六七八九十]+\s*[)）]\s*"),
    re.compile(r"^[一二三四五六七八九十百]+\s*[、.]\s*"),
]
_HEADING_BARE_NUM_RE = re.compile(r"^[0-9０-９]{1,3}\s+")
# 裸数字后接这些字=日期/量纲/序次语义，不剥（「2026 年度工作计划」「12 月进度」）。
_HEADING_BARE_NUM_KEEP = set("年月日号度第周")


def strip_heading_number(text):
    """剥标题字面编号前缀，只留纯题名（题注/正文不动——联动编号只在标题层）。"""
    for _ in range(2):
        t = text
        for rx in HEADING_PREFIX_RES:
            m = rx.match(t)
            if m:
                t = t[m.end():]
                break
        else:
            m = _HEADING_BARE_NUM_RE.match(t)
            if m and m.end() < len(t) and t[m.end()] not in _HEADING_BARE_NUM_KEEP:
                t = t[m.end():]
        if t == text:
            break
        text = t
    return text.strip()


# 题注行（与 longdoc_verify ldvCaptionRe 同源口径：行首 图/表N-M[字母]，索引区
# 行是列表项、以「-」开头不落此匹配；"> " 引用前缀兼容）。
CAPTION_LINE_RE = re.compile(r"^(?:>\s*)?(图|表)\s*([0-9]+)\s*[-–—]\s*([0-9]+)([A-Za-z])?\s+(.+)$")
# 正文引用 token（与 ldvRefRe 同源：行中 图/表N-M[字母]）。
REF_TOKEN_RE = re.compile(r"(图|表)\s*([0-9]+)\s*[-–—]\s*([0-9]+)([A-Za-z])?")


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


# numPr 在 pPr 里的合法后继（CT_PPr 序；插错位 WPS 可能拒读）。
_PPR_NUMPR_SUCCESSORS = (
    "w:suppressLineNumbers", "w:pBdr", "w:shd", "w:tabs", "w:suppressAutoHyphens",
    "w:kinsoku", "w:wordWrap", "w:overflowPunct", "w:topLinePunct", "w:autoSpaceDE",
    "w:autoSpaceDN", "w:bidi", "w:adjustRightInd", "w:snapToGrid", "w:spacing",
    "w:ind", "w:contextualSpacing", "w:mirrorIndents", "w:suppressOverlap", "w:jc",
    "w:textDirection", "w:textAlignment", "w:textboxTightWrap", "w:outlineLvl",
    "w:divId", "w:cnfStyle", "w:rPr", "w:sectPr", "w:pPrChange",
)


def _make_numpr(ilvl, num_id):
    numpr = OxmlElement("w:numPr")
    il = OxmlElement("w:ilvl")
    il.set(qn("w:val"), str(ilvl))
    numpr.append(il)
    ni = OxmlElement("w:numId")
    ni.set(qn("w:val"), str(num_id))
    numpr.append(ni)
    return numpr


# Heading 1/2/3 样式级挂多级编号：multilevel abstractNum（lvl 内 pStyle 链接对应
# 标题样式——这正是 Word/WPS「链接到标题的多级列表」的存储形态）+ 样式 numPr。
# 层内元素顺序按 CT_Lvl 序：start, numFmt, pStyle, suff, lvlText, lvlJc。
def apply_heading_numbering(doc, fmts):
    numbering = doc.part.numbering_part.element
    abs_ids = [int(e.get(qn("w:abstractNumId"))) for e in numbering.findall(qn("w:abstractNum"))]
    num_ids = [int(e.get(qn("w:numId"))) for e in numbering.findall(qn("w:num"))]
    abs_id = (max(abs_ids) if abs_ids else -1) + 1
    num_id = (max(num_ids) if num_ids else 0) + 1
    absnum = OxmlElement("w:abstractNum")
    absnum.set(qn("w:abstractNumId"), str(abs_id))
    mlt = OxmlElement("w:multiLevelType")
    mlt.set(qn("w:val"), "multilevel")
    absnum.append(mlt)
    style_ids = ["Heading1", "Heading2", "Heading3"]
    for lvl in range(len(fmts)):
        el = OxmlElement("w:lvl")
        el.set(qn("w:ilvl"), str(lvl))
        start = OxmlElement("w:start")
        start.set(qn("w:val"), "1")
        el.append(start)
        fmt = OxmlElement("w:numFmt")
        fmt.set(qn("w:val"), "decimal")
        el.append(fmt)
        pst = OxmlElement("w:pStyle")
        pst.set(qn("w:val"), style_ids[lvl])
        el.append(pst)
        suff = OxmlElement("w:suff")
        suff.set(qn("w:val"), "space")
        el.append(suff)
        txt = OxmlElement("w:lvlText")
        txt.set(qn("w:val"), fmts[lvl])
        el.append(txt)
        jc = OxmlElement("w:lvlJc")
        jc.set(qn("w:val"), "left")
        el.append(jc)
        absnum.append(el)
    # abstractNum 必须全部位于 num 之前（CT_Numbering 序）。
    nums = numbering.findall(qn("w:num"))
    if nums:
        nums[0].addprevious(absnum)
    else:
        numbering.append(absnum)
    num = OxmlElement("w:num")
    num.set(qn("w:numId"), str(num_id))
    aref = OxmlElement("w:abstractNumId")
    aref.set(qn("w:val"), str(abs_id))
    num.append(aref)
    numbering.append(num)
    for lvl in range(len(fmts)):
        st = doc.styles[f"Heading {lvl + 1}"]  # 样式名查找；pStyle 链接用 styleId
        ppr = st.element.get_or_add_pPr()
        ppr.insert_element_before(_make_numpr(lvl, num_id), *_PPR_NUMPR_SUCCESSORS)
    return num_id


# 段落级 numId=0 覆写：把该段落从编号序列摘出（整册大标题在册但不编号，
# 章号从第 1 章起算——与静态目录计数同口径）。
def unnumber_paragraph(p):
    ppr = p._p.get_or_add_pPr()
    ppr.insert_element_before(_make_numpr(0, 0), *_PPR_NUMPR_SUCCESSORS)


# 单结果域（fldSimple）：缓存值=构建期实况，WPS/Word 打开或 F9 时重算。
def add_field_simple(paragraph, instr, cached, font, size, bold=False):
    fld = OxmlElement("w:fldSimple")
    fld.set(qn("w:instr"), instr)
    run = paragraph.add_run(cached)
    set_run_font(run, font, size, bold=bold)
    fld.append(run._r)
    paragraph._p.append(fld)
    return fld


_BOOKMARK_SEQ = [100]


def bookmark_start(p, name):
    """书签起点：落 pPr 之后（段落内容最前）。配对 bookmark_end 在标签 run
    群之后调用——REF 域回显只吞编号标签，不吞题名。"""
    bid = _BOOKMARK_SEQ[0]
    _BOOKMARK_SEQ[0] += 1
    bs = OxmlElement("w:bookmarkStart")
    bs.set(qn("w:id"), str(bid))
    bs.set(qn("w:name"), name)
    ppr = p._p.find(qn("w:pPr"))
    p._p.insert(list(p._p).index(ppr) + 1 if ppr is not None else 0, bs)
    return bid


def bookmark_end(p, bid):
    be = OxmlElement("w:bookmarkEnd")
    be.set(qn("w:id"), str(bid))
    p._p.append(be)


def add_caption(doc, b, font, size, caption_marks, caption_style=None):
    """题注：章内序=SEQ 域且**序列名编码章号**（SEQ 图2——按章独立计数，不用
    \\s 1 重启开关：LO 无视该开关会连排成 图2-2，实测坑）。WPS/Word 里插删
    图表 F9 同章重排；章号=字面缓存值（LO 对 STYLEREF 1 \\s 只回显标题文本，
    实测弃用——章插删时题注章号不随动，重出整册即同步）。书签只套编号标签，
    供正文 REF 域回显（LO Reference 按书签内容实时求值，缓存字面兜底）。
    模板可给 caption={font,size,bold,center} 定题注形态（如黑体 12 居中）。"""
    p = doc.add_paragraph()
    if caption_style and caption_style.get("center"):
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    cfont = (caption_style or {}).get("font") or font
    csize = (caption_style or {}).get("size") or size
    cbold = bool((caption_style or {}).get("bold", False))
    name = (caption_marks or {}).get(caption_key(b))
    if not name:
        set_run_font(p.add_run(f"{b['kind']}{b['chap']}-{b['seq']}{b['suffix']} {b['title']}"), cfont, csize, bold=cbold)
        return p
    bid = bookmark_start(p, name)
    set_run_font(p.add_run(f"{b['kind']}{b['chap']}-"), cfont, csize, bold=cbold)
    add_field_simple(p, f"SEQ {b['kind']}{b['chap']} \\* ARABIC", b["seq"], cfont, csize, bold=cbold)
    if b["suffix"]:
        set_run_font(p.add_run(b["suffix"]), cfont, csize, bold=cbold)
    bookmark_end(p, bid)
    set_run_font(p.add_run(" " + b["title"]), cfont, csize, bold=cbold)
    return p


def emit_with_refs(p, text, font, size, caption_marks):
    """正文渲染：图/表N-M 引用 token（有对应题注的）包成 REF 域（缓存=原字面），
    其余原样——书签不存在的不动，避免把年份区间之类的数字误接域。"""
    if not caption_marks:
        set_run_font(p.add_run(text), font, size)
        return
    pos = 0
    for m in REF_TOKEN_RE.finditer(text):
        name = caption_marks.get(f"{m.group(1)}{m.group(2)}-{m.group(3)}")
        if not name:
            continue
        if m.start() > pos:
            set_run_font(p.add_run(text[pos:m.start()]), font, size)
        run = p.add_run(m.group(0))
        set_run_font(run, font, size)
        fld = OxmlElement("w:fldSimple")
        fld.set(qn("w:instr"), f"REF {name} \\h")
        fld.append(run._r)
        p._p.append(fld)
        pos = m.end()
    if pos < len(text):
        set_run_font(p.add_run(text[pos:]), font, size)


def find_title_block_index(blocks, title):
    """与文档标题同文的首个一级标题块位次（整册大标题不进编号序列）。
    块文本按原文比对（剥前缀在两侧对称做）。"""
    t = strip_heading_number((title or "").strip())
    if not t:
        return -1
    for idx, b in enumerate(blocks):
        if b.get("type") == "heading1" and strip_heading_number(b.get("text", "")) == t:
            return idx
    return -1


def decorate_toc_entries(entries, fmts, title):
    """静态目录条目编号与正文多级编号同源计算：剥字面前缀、按 1~2 级计数器
    重新拼编号（编号引擎渲染什么，目录就写什么）。文档标题条目不带编号
    （正文里它也不编号）。"""
    if not fmts or not entries:
        return entries
    title_key = strip_heading_number((title or "").strip())
    c1 = c2 = 0
    title_done = False
    out = []
    for e in entries:
        e = dict(e)
        lvl = int(e.get("level") or 1)
        text = strip_heading_number(str(e.get("text", "")).strip())
        numbered = True
        if lvl == 1 and text == title_key and not title_done:
            title_done = True
            numbered = False
        if numbered and lvl == 1:
            c1 += 1
            c2 = 0
            text = fmts[0].replace("%1", str(c1)) + " " + text
        elif numbered and lvl == 2:
            c2 += 1
            text = fmts[1].replace("%1", str(c1)).replace("%2", str(c2)) + " " + text
        e["text"] = text
        out.append(e)
    return out


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


def _fld_char_run(p, ftype):
    r = p.add_run()
    fc = OxmlElement("w:fldChar")
    fc.set(qn("w:fldCharType"), ftype)
    r._r.append(fc)


def _fld_instr_run(p, instr):
    r = p.add_run()
    it = OxmlElement("w:instrText")
    it.set(qn("xml:space"), "preserve")
    it.text = instr
    r._r.append(it)


# 目录=TOC 域带实测缓存（2026-11-24 联动化）：复杂域跨段——首条目段携带
# begin/instr/separate、末条目段携带 end，中间段落即域缓存结果。打开文档看到的
# 缓存=两遍法实测的真实条目+页码（打开即准确，不吃「旧页码」坑）；WPS/Word 里
# F9 或打印预览时域重算，条目/编号/页码从标题全联动重排。LibreOffice 渲染链
# 显示缓存结果，doc_render 预览不受影响。
def add_toc_static(doc, entries, font, size):
    from docx.enum.text import WD_TAB_ALIGNMENT, WD_TAB_LEADER
    if not entries:
        return
    p = doc.add_paragraph()
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    set_run_font(p.add_run("目 录"), font, size + 4, bold=True)
    last = len(entries) - 1
    for i, e in enumerate(entries):
        p = doc.add_paragraph()
        lvl = int(e.get("level") or 1)
        if lvl >= 2:
            p.paragraph_format.left_indent = Cm(0.74 * (lvl - 1))
        ts = p.paragraph_format.tab_stops
        ts.add_tab_stop(Cm(15.0), WD_TAB_ALIGNMENT.RIGHT, WD_TAB_LEADER.DOTS)
        if i == 0:
            _fld_char_run(p, "begin")
            _fld_instr_run(p, r'TOC \o "1-2" \h \z \u')
            _fld_char_run(p, "separate")
        page = str(e.get("page", "—"))
        set_run_font(p.add_run(e.get("text", "") + "\t" + page), font, size if lvl == 1 else size - 0.5, bold=(lvl == 1))
        if i == last:
            _fld_char_run(p, "end")
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


def render_blocks(doc, blocks, font, size, line_spacing, style,
                  number_headings=True, unnumber_idx=-1, caption_marks=None):
    table_fill = style["table_fill"]
    heading_styles = style["headingStyles"]
    for bidx, b in enumerate(blocks):
        btype = b.get("type", "paragraph")
        if btype in ("heading1", "heading2", "heading3"):
            level = int(btype[-1])
            hs = heading_styles[level]
            p = doc.add_paragraph()
            p.paragraph_format.space_before = Pt(hs["spaceBefore"])
            p.paragraph_format.space_after = Pt(hs["spaceAfter"])
            # 联动开=剥字面前缀（编号由多级列表引擎渲染）；联动关=字面原样。
            htext = strip_heading_number(b.get("text", "")) if number_headings else b.get("text", "")
            set_run_font(p.add_run(htext), hs["font"], hs["size"], bold=hs["bold"], color=hs["color"])
            p.style = doc.styles[f"Heading {level}"]
            for r in p.runs:
                set_run_font(r, hs["font"], hs["size"], bold=hs["bold"], color=hs["color"])
            # 整册大标题（与 --title 同文的首个一级标题）：在册但不进编号序列
            if number_headings and bidx == unnumber_idx:
                unnumber_paragraph(p)
        elif btype == "paragraph":
            p = doc.add_paragraph()
            p.paragraph_format.line_spacing = line_spacing
            emit_with_refs(p, b.get("text", ""), font, size, caption_marks if number_headings else None)
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
        elif btype == "caption":
            add_caption(doc, b, font, size, caption_marks if number_headings else None,
                        caption_style=style.get("caption"))
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
        # headings（字面编号前缀在渲染层按联动开关剥——off 模式必须原样保留，
        # 剥了没引擎接管=编号凭空消失）
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
        # paragraph — merge consecutive non-empty lines; 题注行单独拆出（转域）
        buf = []
        while i < len(lines):
            s = lines[i].strip()
            if not s or s.startswith("#") or s.startswith("```") or MARKDOWN_TABLE_RE.match(s) or re.match(r"^([-*+]|\d+[.)])\s+", s):
                break
            if CAPTION_LINE_RE.match(s):
                flush_para(blocks, buf)
                blocks.append(caption_block(s))
            else:
                buf.append(s)
            i += 1
        flush_para(blocks, buf)
        continue
    return blocks


def flush_para(blocks, buf):
    if buf:
        blocks.append({"type": "paragraph", "text": "\n".join(buf)})
        buf.clear()


def caption_block(line):
    m = CAPTION_LINE_RE.match(line.strip())
    return {"type": "caption", "kind": m.group(1), "chap": m.group(2),
            "seq": m.group(3), "suffix": m.group(4) or "", "title": m.group(5).strip()}


def caption_key(b):
    return f"{b['kind']}{b['chap']}-{b['seq']}"


def caption_bookmark_name(b):
    # 书签名禁连字符；首字母须为字母（下划线前缀同 Word _Ref 习惯）。
    return f"_Gaea{b['kind']}{b['chap']}_{b['seq']}"


# ── 格式模板（2026-11-24 方案格式模板线）────────────────────────────
# 登记模板=JSON 文件（格式参数一次性登记，方案编写直接引用，不再每次调格式）。
# 解析优先级由 Go 侧 resolveDocxTemplate 定（工作区 .gaea/templates/docx/ >
# 技能 format_templates/），python 只收 --template-file <绝对路径>。
# schema（v1）：base=内置模板继承面；font/fontSize/lineSpacing/pageSize/
# marginsCm/tableFill 全局；headings={"1".."3":{font,size,bold,color,
# pageBreakBefore,spaceBefore,spaceAfter}}（缺省级推 base+内置偏移）；
# numbering=false 字面 | ["第%1章","%1.%2","%1.%2.%3"] 自定层级 | 缺省继承 base；
# headerBorder/coverStyle("bid"|"standard") 页眉框线与封面形态。
_INHERIT = object()


def _parse_color(v, fallback):
    if v is None:
        return fallback
    if isinstance(v, (list, tuple)):
        return (int(v[0]), int(v[1]), int(v[2]))
    v = str(v).strip().lstrip("#")
    return (int(v[0:2], 16), int(v[2:4], 16), int(v[4:6], 16))


def normalize_template(raw, base_key):
    """登记模板 JSON → 归一规格（builtin 键走 TEMPLATES 同一出口）。"""
    base = TEMPLATES.get(base_key) or TEMPLATES["通用"]
    size = float(raw.get("fontSize") or 12)
    base_sizes = {1: size + 4, 2: size + 2, 3: size + 0.5}
    font = raw.get("font") or base["font"] or FALLBACK_FONT
    heading_font_default = raw.get("headingFont") or base.get("headingFont") or ""
    headings_raw = raw.get("headings") or {}
    cap = raw.get("caption") or {}
    spec = {
        "bid": bool(raw.get("bid", base_key == "标书")),
        "font": font,
        "fontSize": size,
        "lineSpacing": float(raw.get("lineSpacing") or 1.5),
        "pageSize": raw.get("pageSize") or "A4",
        "marginsCm": raw.get("marginsCm") or [2.54, 2.54, 3.17, 3.17],
        "tableFill": raw.get("tableFill") or base.get("fill") or "D5E8F0",
        "coverStyle": raw.get("coverStyle") or ("bid" if base_key == "标书" else "standard"),
        "headerBorder": bool(raw.get("headerBorder", base_key == "标书")),
        # 页眉/页脚文本缺省（调用时传参优先；页脚 Go 链恒传「第 {page} 页」，此
        # 字段主要给 CLI 直用面）。
        "headerText": raw.get("headerText") or "",
        "footerText": raw.get("footerText") or "",
        "numbering": raw.get("numbering", _INHERIT),
        "caption": None,
        "headingStyles": {},
    }
    if cap:
        spec["caption"] = {
            "font": cap.get("font") or font,
            "size": float(cap["size"]) if cap.get("size") is not None else size,
            "bold": bool(cap.get("bold", False)),
            "center": bool(cap.get("center", True)),
        }
    for lvl in (1, 2, 3):
        d = headings_raw.get(str(lvl)) or {}
        spec["headingStyles"][lvl] = {
            "font": d.get("font") or heading_font_default or font,
            "size": float(d["size"]) if d.get("size") is not None else base_sizes[lvl],
            "bold": bool(d.get("bold", True)),
            "color": _parse_color(d.get("color"), base.get("heading", (0, 0, 0))),
            "pageBreakBefore": bool(d.get("pageBreakBefore", lvl == 1 and base_key == "标书")),
            "spaceBefore": float(d.get("spaceBefore", 12 if lvl <= 2 else 8)),
            "spaceAfter": float(d.get("spaceAfter", 6)),
        }
    return spec


def builtin_spec(tmpl_key):
    return normalize_template({}, tmpl_key)


def load_format_template(path):
    with open(path, encoding="utf-8") as f:
        raw = json.load(f)
    return normalize_template(raw, raw.get("base") or "通用")


# 标题样式落样式表层（run 直接格式同款双保险；章标题另起页按模板）——样式层
# 的 pageBreakBefore/字体正是 longdoc_verify book 读数所查面，生成端与验证端同源。
def apply_heading_styles(doc, spec):
    for lvl in (1, 2, 3):
        hs = spec["headingStyles"][lvl]
        st = doc.styles[f"Heading {lvl}"]
        st.font.name = hs["font"]
        st.font.size = Pt(hs["size"])
        st.font.bold = hs["bold"]
        st.font.color.rgb = RGBColor(*hs["color"])
        rPr = st.element.get_or_add_rPr()
        rFonts = rPr.find(qn("w:rFonts"))
        if rFonts is None:
            rFonts = OxmlElement("w:rFonts")
            rPr.append(rFonts)
        rFonts.set(qn("w:ascii"), hs["font"])
        rFonts.set(qn("w:hAnsi"), hs["font"])
        rFonts.set(qn("w:eastAsia"), hs["font"])
        if hs["pageBreakBefore"]:
            st.paragraph_format.page_break_before = True


def build(doc, title, subtitle, author, opts, blocks):
    template_file = opts.get("templateFile") or ""
    if template_file:
        spec = load_format_template(template_file)
        base_key = "标书" if spec["bid"] else "通用"
    else:
        tmpl_key = opts.get("template") or "通用"
        spec = builtin_spec(tmpl_key)
        base_key = tmpl_key
    bid = spec["bid"]
    n = spec["numbering"]
    if n is _INHERIT or n is None:
        numbering_fmts = HEADING_NUMBERING.get(base_key, _NUM_STD)
    elif n is False:
        numbering_fmts = None
    else:
        numbering_fmts = [str(x) for x in n]
    # 显式开关压过模板缺省（auto=按模板；off=文学成册等字面口径——「序章」这类
    # 无编号章名被挂上「第1章」是内容损伤）。
    hn = str(opts.get("headingNumbering") or "auto").lower()
    if hn == "off":
        numbering_fmts = None
    elif hn == "on" and numbering_fmts is None:
        numbering_fmts = _NUM_STD
    font = opts.get("font") or spec["font"] or FALLBACK_FONT
    style = {
        "table_fill": spec["tableFill"],
        "headingStyles": spec["headingStyles"],
        "caption": spec["caption"],
    }
    size = float(opts.get("fontSize") or spec["fontSize"] or 12)
    line_spacing = float(opts.get("lineSpacing") or spec["lineSpacing"])
    sec = setup_section(doc, opts.get("pageSize") or spec["pageSize"],
                        opts.get("marginsCm") or spec["marginsCm"])
    # 页眉缺省序：调用传参 > 模板登记 > 标书取投标人（P0-2 建议④）。
    header_text = opts.get("header") or spec["headerText"]
    if not header_text and bid and author:
        header_text = author
    footer_text = opts.get("footer")
    if footer_text in (None, "") and spec["footerText"]:
        footer_text = spec["footerText"]
    add_header_footer(sec, header_text, footer_text, font, size, header_border=spec["headerBorder"])
    cover_color = spec["headingStyles"][1]["color"]
    if opts.get("cover"):
        if spec["coverStyle"] == "bid":
            add_cover_bid(doc, title, subtitle, author,
                          opts.get("bidCopy") or "正本",
                          opts.get("bidDate") or date.today().strftime("%Y 年 %m 月 %d 日"),
                          font, size)
        else:
            add_cover(doc, title, subtitle, author, font, size, cover_color)
    if opts.get("toc"):
        if opts.get("tocEntries"):
            add_toc_static(doc, decorate_toc_entries(opts["tocEntries"], numbering_fmts, title), font, size)
        else:
            add_toc(doc, font, size)
    apply_heading_styles(doc, spec)
    if numbering_fmts:
        apply_heading_numbering(doc, numbering_fmts)
    # 题注书签表：撞号时首题注持签（撞号本身由 longdoc_verify 机检拦）。
    caption_marks = {}
    if numbering_fmts:
        for b in blocks:
            if b.get("type") == "caption":
                caption_marks.setdefault(caption_key(b), caption_bookmark_name(b))
    render_blocks(doc, blocks, font, size, line_spacing, style,
                  number_headings=numbering_fmts is not None,
                  unnumber_idx=find_title_block_index(blocks, title),
                  caption_marks=caption_marks)
    # default style so hand-added paragraphs use the same font
    normal = doc.styles["Normal"]
    normal.font.name = font
    normal.font.size = Pt(size)
    normal._element.rPr.rFonts.set(qn("w:eastAsia"), font)


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
    ap.add_argument("--heading-numbering", default="auto", choices=["auto","on","off"], help="标题/图表编号联动：auto=按模板（公文 off），off=全字面（文学成册）")
    ap.add_argument("--template-file", default="", help="登记格式模板 JSON（格式参数一次性登记：字体/标题层级/编号/页眉框线/封面形态）；给定时代替 --template")
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
            "headingNumbering": args.heading_numbering,
            "templateFile": args.template_file,
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
