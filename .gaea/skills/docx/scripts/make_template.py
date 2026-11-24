#!/usr/bin/env python3
"""make_template.py — 从格式样例 docx 抽取格式模板 JSON（登记一次，次次复用）。

用法：
    python make_template.py 样例.docx [输出.json] [--name 模板名] [--base 标书]

抽取面：页面尺寸/边距、正文字体字号行距、Heading 1~3（字体/字号/加粗/颜色/
另起页/段前后距）、标题多级编号 lvlText（样式层链接时）、页眉页脚文本。
抽取产物是可编辑骨架——登记前人工核对一遍（样例里 run 直接格式覆盖样式时，
以渲染实貌为准补字段）。schema 见 create_docx.py 顶部「格式模板」注释。
"""
import argparse
import json
import sys

from docx import Document
from docx.oxml.ns import qn


def _color_hex(rPr):
    if rPr is None:
        return None
    c = rPr.find(qn("w:color"))
    if c is not None and c.get(qn("w:val")) not in (None, "auto"):
        return c.get(qn("w:val"))
    return None


def _style_font(st):
    rPr = st.element.find(qn("w:rPr"))
    name = None
    if rPr is not None:
        rf = rPr.find(qn("w:rFonts"))
        if rf is not None:
            name = rf.get(qn("w:eastAsia")) or rf.get(qn("w:ascii"))
    return name or (st.font.name or None), _color_hex(rPr)


def _heading_lvl_text(numbering, st):
    """样式层链接了多级编号时抽该级 lvlText（第%1章/%1.%2 形态）。"""
    if numbering is None:
        return None
    ppr = st.element.find(qn("w:pPr"))
    if ppr is None:
        return None
    numpr = ppr.find(qn("w:numPr"))
    if numpr is None:
        return None
    nid = numpr.find(qn("w:numId"))
    ilvl = numpr.find(qn("w:ilvl"))
    if nid is None:
        return None
    num = numbering.find(qn("w:num") + f'[@{qn("w:numId")}="{nid.get(qn("w:val"))}"]')
    if num is None:
        return None
    aref = num.find(qn("w:abstractNumId"))
    if aref is None:
        return None
    absnum = numbering.find(qn("w:abstractNum") + f'[@{qn("w:abstractNumId")}="{aref.get(qn("w:val"))}"]')
    if absnum is None:
        return None
    lvl = absnum.findall(qn("w:lvl"))
    idx = int(ilvl.get(qn("w:val"))) if ilvl is not None else 0
    if 0 <= idx < len(lvl):
        t = lvl[idx].find(qn("w:lvlText"))
        if t is not None:
            return t.get(qn("w:val"))
    return None


def main():
    ap = argparse.ArgumentParser(description="从格式样例 docx 抽格式模板 JSON 骨架")
    ap.add_argument("sample", help="格式样例 .docx")
    ap.add_argument("output", nargs="?", default="", help="输出 JSON 路径（缺省打印 stdout）")
    ap.add_argument("--name", default="", help="模板名（缺省=样例文件名）")
    ap.add_argument("--base", default="标书", choices=["通用", "公文", "报告", "合同", "标书"],
                    help="继承内置面（封面形态/页眉框线按 base 推）")
    args = ap.parse_args()

    doc = Document(args.sample)
    sec = doc.sections[0]

    def cm(emu):
        return round(emu / 360000.0, 2)

    page = "A4"
    if sec.page_width and sec.page_width / 914400.0 > 8.6:
        page = "Letter"

    normal = doc.styles["Normal"]
    normal_name, _ = _style_font(normal)
    numbering = None
    try:
        numbering = doc.part.numbering_part.element
    except Exception:
        pass

    headings = {}
    lvl_fmts = []
    for lvl in (1, 2, 3):
        try:
            st = doc.styles[f"Heading {lvl}"]
        except KeyError:
            continue
        name, color = _style_font(st)
        d = {}
        if name:
            d["font"] = name
        if st.font.size:
            d["size"] = round(st.font.size.pt, 1)
        if st.font.bold is not None:
            d["bold"] = bool(st.font.bold)
        if color:
            d["color"] = color
        pf = st.paragraph_format
        if pf.page_break_before:
            d["pageBreakBefore"] = True
        if pf.space_before:
            d["spaceBefore"] = round(pf.space_before.pt, 1)
        if pf.space_after:
            d["spaceAfter"] = round(pf.space_after.pt, 1)
        headings[str(lvl)] = d
        lt = _heading_lvl_text(numbering, st)
        if lt:
            lvl_fmts.append((lvl, lt))

    out = {
        "name": args.name or args.sample.rsplit("/", 1)[-1].rsplit("\\", 1)[-1].rsplit(".", 1)[0],
        "base": args.base,
        "description": "由格式样例抽取（make_template.py），登记前人工核对一遍——样例 run 直接格式覆盖样式时以渲染实貌为准补字段。",
        "font": normal_name or "宋体",
        "fontSize": round(normal.font.size.pt, 1) if normal.font.size else 12,
        "pageSize": page,
        "marginsCm": [cm(sec.top_margin), cm(sec.bottom_margin), cm(sec.left_margin), cm(sec.right_margin)],
        "headings": headings,
    }
    if lvl_fmts:
        out["numbering"] = [lt for _, lt in sorted(lvl_fmts)]
    # 页眉页脚文本（首节；域只给指令名）
    hdr = sec.header
    if hdr is not None and hdr.paragraphs:
        t = "".join(r.text for p in hdr.paragraphs for r in p.runs).strip()
        if t:
            out["headerText"] = t
    ftr = sec.footer
    if ftr is not None and ftr.paragraphs:
        t = "".join(r.text for p in ftr.paragraphs for r in p.runs).strip()
        if t:
            out["footerText"] = t

    js = json.dumps(out, ensure_ascii=False, indent=2)
    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(js + "\n")
        print(json.dumps({"ok": True, "path": args.output}, ensure_ascii=False))
    else:
        print(js)
    return 0


if __name__ == "__main__":
    sys.exit(main())
