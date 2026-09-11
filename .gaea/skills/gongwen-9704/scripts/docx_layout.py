#!/usr/bin/env python3
"""docx 排版事实提取(公文规范技能取数口)。

从 .docx 的 word/document.xml 提取排版事实,输出 JSON,供对照 GB/T 9704
排版细则逐项判定。仅用标准库(zipfile + ElementTree),无第三方依赖。

用法: python docx_layout.py <file.docx>

输出结构:
  margins: {top_mm, bottom_mm, left_mm, right_mm}  页边距(毫米,twips 换算)
  margins_found: bool                               是否存在页面设置
  paragraphs: [{text, font, size_half_pt, line_twips, line_rule,
                first_line_chars, first_line_twips, center}]  逐段事实
"""
import json
import sys
import zipfile
import xml.etree.ElementTree as ET

TWIPS_PER_MM = 56.6929


def local(tag):
    return tag.rsplit("}", 1)[-1]


def attrs(el):
    return {local(k): v for k, v in el.attrib.items()}


def attr_int(d, key):
    try:
        return int(d.get(key, "0") or "0")
    except ValueError:
        return 0


def extract(path):
    with zipfile.ZipFile(path) as z:
        data = z.read("word/document.xml")
    root = ET.fromstring(data)

    margins = {}
    state = {"margins_found": False}
    paragraphs = []

    def walk(el, cur, in_run):
        name = local(el.tag)
        if name == "p":
            nxt = {"text": "", "font": "", "size_half_pt": 0, "line_twips": 0,
                   "line_rule": "", "first_line_chars": 0, "first_line_twips": 0,
                   "center": False}
            for child in el:
                nxt = walk(child, nxt, False)
            paragraphs.append(nxt)
            return cur
        if name == "r":
            for child in el:
                walk(child, cur, True)
            return cur
        if name == "rPr":
            # rFonts/sz 在 rPr 下:run 级优先,段级(段落标记)仅作缺省
            for child in el:
                cn, ca = local(child.tag), attrs(child)
                if cn == "rFonts":
                    f = ca.get("eastAsia") or ca.get("ascii") or ""
                    if in_run:
                        if not cur["font"]:
                            cur["font"] = f
                    elif not cur["font"]:
                        cur["font"] = f
                elif cn == "sz":
                    v = attr_int(ca, "val")
                    if in_run:
                        if not cur["size_half_pt"]:
                            cur["size_half_pt"] = v
                    elif not cur["size_half_pt"]:
                        cur["size_half_pt"] = v
            return cur
        if name == "spacing":
            ca = attrs(el)
            cur["line_twips"] = attr_int(ca, "line")
            cur["line_rule"] = ca.get("lineRule", "")
            return cur
        if name == "ind":
            ca = attrs(el)
            cur["first_line_chars"] = attr_int(ca, "firstLineChars")
            cur["first_line_twips"] = attr_int(ca, "firstLine")
            return cur
        if name == "jc":
            if attrs(el).get("val") == "center":
                cur["center"] = True
            return cur
        if name == "t":
            cur["text"] += el.text or ""
            return cur
        if name == "sectPr":
            for child in el.iter():
                if local(child.tag) == "pgMar":
                    ca = attrs(child)
                    state["margins_found"] = True
                    for side in ("top", "bottom", "left", "right"):
                        tw = attr_int(ca, side)
                        margins[side + "_mm"] = round(tw / TWIPS_PER_MM, 1) if tw else 0.0
                    break
            for child in el:
                walk(child, cur, in_run)
            return cur
        for child in el:
            walk(child, cur, in_run)
        return cur

    walk(root, None, False)
    return {"margins": margins, "margins_found": state["margins_found"], "paragraphs": paragraphs}


def main():
    if len(sys.argv) != 2:
        print(__doc__, file=sys.stderr)
        sys.exit(2)
    print(json.dumps(extract(sys.argv[1]), ensure_ascii=False, indent=1))


if __name__ == "__main__":
    main()
