#!/usr/bin/env python3
"""docx 排版事实提取(公文规范技能取数口)。

从 .docx 的 word/document.xml 与 word/footer*.xml 提取排版事实,输出 JSON,
供对照 GB/T 9704 排版细则逐项判定。仅用标准库(zipfile + ElementTree),
无第三方依赖。

用法: python docx_layout.py <file.docx>

输出结构:
  margins: {top_mm, bottom_mm, left_mm, right_mm}  页边距(毫米,twips 换算)
  margins_found: bool                               是否存在页面设置
  paragraphs: [{text, font, size_half_pt, line_twips, line_rule,
                first_line_chars, first_line_twips, center}]  逐段事实
  footers: [{part, ref_types, has_page_field, size_half_pt, text}]
            页脚事实,按部件名排序;无页脚 = 空数组(数据不足,不报错)
  colors: [{val, count, max_size_half_pt, sample}]
            正文指定字色事实,按出现次数降序;无 = 空数组
"""
import json
import re
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


def has_page_instr(instr):
    # NUMPAGES 也含 PAGE 子串,按独立词匹配才认页码字段
    return re.search(r"\bPAGE\b", instr or "") is not None


def parse_footer(data):
    """单个 footer part 的事实:页码字段/最大字号/全文(含 PAGE 前后一字线文本)。"""
    info = {"has_page_field": False, "size_half_pt": 0, "text": ""}
    try:
        root = ET.fromstring(data)
    except ET.ParseError:
        return info
    for el in root.iter():
        name = local(el.tag)
        if name == "fldSimple":
            if has_page_instr(attrs(el).get("instr", "")):
                info["has_page_field"] = True
        elif name == "instrText":
            if has_page_instr(el.text or ""):
                info["has_page_field"] = True
        elif name == "sz":
            # 页脚通常仅页码一处字号,取最大即可
            v = attr_int(attrs(el), "val")
            if v > info["size_half_pt"]:
                info["size_half_pt"] = v
        elif name == "t":
            info["text"] += el.text or ""
    info["text"] = info["text"].strip()[:80]
    return info


def load_footer_rels(z):
    """document.xml.rels 里 footer 关系的 rId → 部件名;文件缺失或坏 XML 容错为空。"""
    try:
        data = z.read("word/_rels/document.xml.rels")
    except KeyError:
        return {}
    try:
        root = ET.fromstring(data)
    except ET.ParseError:
        return {}
    out = {}
    for rel in root.iter():
        if local(rel.tag) != "Relationship":
            continue
        ca = attrs(rel)
        target = ca.get("Target", "")
        if not ca.get("Type", "").endswith("/footer") or not target:
            continue
        # Target 相对 word/ 目录;包内绝对写法带前导斜杠
        out[ca.get("Id", "")] = target.lstrip("/") if target.startswith("/") else "word/" + target
    return out


def extract(path):
    with zipfile.ZipFile(path) as z:
        data = z.read("word/document.xml")
        rels = load_footer_rels(z)
        # 存在哪些 footer 部件即报哪些;字节先取出,解析放 walk 之后
        footer_parts = sorted(n for n in z.namelist()
                              if n.startswith("word/footer") and n.endswith(".xml"))
        footer_data = [(p, z.read(p)) for p in footer_parts]
    root = ET.fromstring(data)

    margins = {}
    state = {"margins_found": False, "footer_refs": [], "colors": {}, "pending_color": ""}
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
            state["pending_color"] = ""  # 字色只归属本 run 的文本,不串到下一 run
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
                elif cn == "color":
                    # 只采 run 级指定色;val="auto" 表跟随正文,非指定色不采
                    v = ca.get("val", "").strip()
                    if in_run and v and v.lower() != "auto":
                        rec = state["colors"].setdefault(
                            v, {"count": 0, "max_size_half_pt": 0, "sample": ""})
                        rec["count"] += 1
                        state["pending_color"] = v
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
            if in_run and state["pending_color"]:
                rec = state["colors"][state["pending_color"]]
                if len(rec["sample"]) < 20:
                    rec["sample"] = (rec["sample"] + (el.text or ""))[:20]
                if cur["size_half_pt"] > rec["max_size_half_pt"]:
                    rec["max_size_half_pt"] = cur["size_half_pt"]
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
                if local(child.tag) == "footerReference":
                    ca = attrs(child)
                    state["footer_refs"].append(
                        {"id": ca.get("id", ""), "type": ca.get("type", "")})
                walk(child, cur, in_run)
            return cur
        for child in el:
            walk(child, cur, in_run)
        return cur

    walk(root, None, False)

    # sectPr 的 footerReference 只记 r:id,经 rels 反查归位到具体 footer 部件
    refs_by_part = {}
    for fr in state["footer_refs"]:
        part = rels.get(fr["id"])
        if part:
            refs_by_part.setdefault(part, []).append(fr["type"])
    footers = []
    for part, fdata in footer_data:
        info = parse_footer(fdata)
        footers.append({"part": part, "ref_types": refs_by_part.get(part, []), **info})
    colors = [
        {"val": v, "count": rec["count"], "max_size_half_pt": rec["max_size_half_pt"],
         "sample": rec["sample"]}
        for v, rec in sorted(state["colors"].items(), key=lambda kv: (-kv[1]["count"], kv[0]))
    ]
    return {"margins": margins, "margins_found": state["margins_found"],
            "paragraphs": paragraphs, "footers": footers, "colors": colors}


def main():
    if len(sys.argv) != 2:
        print(__doc__, file=sys.stderr)
        sys.exit(2)
    print(json.dumps(extract(sys.argv[1]), ensure_ascii=False, indent=1))


if __name__ == "__main__":
    main()
