#!/usr/bin/env python3
"""docx 公文生成器(GB/T 9704-2012 空白公文骨架,规范技能「生成」半边)。

与 docx_layout.py(取数/校验)同标准库口径(zipfile + 手写 XML),无第三方依赖。
生成即合规:页边距/红头字色/标题字号/正文仿宋三号/行距/首行缩进/页码一字线
逐项对齐 SKILL.md 排版细则表,生成后可直接用 docx_layout.py 复检。

用法:
  python docx_gen.py --org "××市人民政府" --title "关于××的通知" \
      [--docnum "×政发〔2026〕1号"] [--to "各区、县人民政府："] \
      [--date "2026年9月12日"] [--body-file 正文.txt | --body "第一段\n第二段"] \
      [--cc "抄送：××"] [--print-org "××市人民政府办公室"] [-o out.docx]

正文段落自动分层:行首「一、/二、…」→ 黑体;「（一）…」→ 楷体;其余 → 仿宋。
缺正文时落一段「（正文内容）」占位,占位不算不合格(判定纪律:不适用≠缺失)。
"""
import argparse
import datetime
import os
import sys
import zipfile
from xml.sax.saxutils import escape

W_NS = 'xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"'
R_NS = 'xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"'

SONG_TITLE = "方正小标宋简体"   # 红头/标题字体(建议值,字体名不硬判定)
FANGSONG = "仿宋_GB2312"        # 正文字体
HEI = "黑体"                    # 一级层级标题
KAI = "楷体"                    # 二级层级标题
RED = "FF0000"

# GB/T 9704 页边距(mm → twips,1mm≈56.6929;容差 ±2mm)
MARGIN = {"top": 2098, "bottom": 1984, "left": 1588, "right": 1474}  # 37/35/28/26mm
A4_W, A4_H = 11906, 16838

SZ_BODY = 32    # 三号 16pt
SZ_TITLE = 44   # 二号 22pt
SZ_HEADER = 72  # 红头大字 36pt(≥44 即满足字色规则的红头字号口径)
SZ_PAGE = 28    # 页码 4号 14pt
LINE = "560"    # 行距 28 磅固定值


def para(text, font=FANGSONG, size=SZ_BODY, color="", center=False, right=False,
         indent=True, line=LINE, extra_pPr=""):
    """一个 w:p:段级 spacing/ind/jc + run 级 rPr(字体/字号/字色)。"""
    ppr = []
    if line:
        ppr.append(f'<w:spacing w:line="{line}" w:lineRule="exact"/>')
    if indent:
        ppr.append('<w:ind w:firstLineChars="200" w:firstLine="640"/>')
    if center:
        ppr.append('<w:jc w:val="center"/>')
    if right:
        ppr.append('<w:jc w:val="right"/>')
    if extra_pPr:
        ppr.append(extra_pPr)
    rpr = f'<w:rFonts w:eastAsia="{font}" w:ascii="{font}"/>'
    if color:
        rpr += f'<w:color w:val="{color}"/>'
    rpr += f'<w:sz w:val="{size}"/><w:szCs w:val="{size}"/>'
    run = f'<w:r><w:rPr>{rpr}</w:rPr><w:t xml:space="preserve">{escape(text)}</w:t></w:r>' if text else ""
    return f'<w:p><w:pPr>{"".join(ppr)}</w:pPr>{run}</w:p>'


def level_font(text):
    """正文行首分层:一、/二、…→黑体;(一)…→楷体;其余→仿宋。"""
    t = text.lstrip()
    for prefix in "一二三四五六七八九十":
        if t.startswith(prefix + "、"):
            return HEI
    if t.startswith("（") and ("）" in t[:6]):
        return KAI
    return FANGSONG


def document_xml(org, docnum, title, to, date, body_paras, cc, print_org):
    ps = []
    ps.append(para(org, font=SONG_TITLE, size=SZ_HEADER, color=RED, center=True, indent=False))
    if docnum:
        ps.append(para(docnum, center=True, indent=False))
    # 发文字号下的红色分隔线(GB/T 9704 版头要素;校验脚本对 pBdr 不取数,不影响)
    ps.append('<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="12" w:space="1" '
              'w:color="FF0000"/></w:pBdr><w:spacing w:line="100" w:lineRule="exact"/>'
              '</w:pPr></w:p>')
    if title:
        ps.append(para(title, font=SONG_TITLE, size=SZ_TITLE, center=True, indent=False))
    if to:
        ps.append(para(to, indent=False))
    ps.extend(body_paras)
    if date:
        ps.append(para(date, right=True, indent=False))
    if cc:
        ps.append(para(cc, indent=False))
    if print_org:
        ps.append(para(print_org + "  " + date, indent=False))
    sect = (f'<w:sectPr><w:pgSz w:w="{A4_W}" w:h="{A4_H}"/>'
            f'<w:pgMar w:top="{MARGIN["top"]}" w:right="{MARGIN["right"]}" '
            f'w:bottom="{MARGIN["bottom"]}" w:left="{MARGIN["left"]}"/>'
            f'<w:footerReference w:type="default" r:id="rId1"/></w:sectPr>')
    return (f'<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
            f'<w:document {W_NS} {R_NS}><w:body>{"".join(ps)}{sect}</w:body></w:document>')


FOOTER_XML = (
    f'<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:ftr {W_NS}>'
    '<w:p><w:pPr><w:jc w:val="center"/></w:pPr>'
    f'<w:r><w:rPr><w:rFonts w:eastAsia="宋体"/><w:sz w:val="{SZ_PAGE}"/></w:rPr>'
    '<w:t xml:space="preserve">— </w:t></w:r>'
    f'<w:fldSimple w:instr=" PAGE "><w:r><w:rPr><w:rFonts w:eastAsia="宋体"/>'
    f'<w:sz w:val="{SZ_PAGE}"/></w:rPr><w:t>1</w:t></w:r></w:fldSimple>'
    f'<w:r><w:rPr><w:rFonts w:eastAsia="宋体"/><w:sz w:val="{SZ_PAGE}"/></w:rPr>'
    '<w:t xml:space="preserve"> —</w:t></w:r></w:p></w:ftr>'
)

STYLES_XML = (
    f'<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles {W_NS}>'
    '<w:docDefaults><w:rPrDefault><w:rPr>'
    f'<w:rFonts w:eastAsia="{FANGSONG}" w:ascii="{FANGSONG}"/>'
    f'<w:sz w:val="{SZ_BODY}"/><w:szCs w:val="{SZ_BODY}"/>'
    '</w:rPr></w:rPrDefault></w:docDefaults></w:styles>'
)

CONTENT_TYPES = (
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
    '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">'
    '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
    '<Default Extension="xml" ContentType="application/xml"/>'
    '<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>'
    '<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>'
    '<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>'
    '</Types>'
)

ROOT_RELS = (
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
    '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
    '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>'
    '</Relationships>'
)

DOC_RELS = (
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
    '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
    '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>'
    '</Relationships>'
)


def build_docx(path, org, docnum, title, to, date, body_lines, cc, print_org):
    if not body_lines:
        body_lines = ["（正文内容）"]
    body_paras = [para(t, font=level_font(t)) for t in body_lines if t.strip()]
    doc = document_xml(org, docnum, title, to, date, body_paras, cc, print_org)
    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("[Content_Types].xml", CONTENT_TYPES)
        z.writestr("_rels/.rels", ROOT_RELS)
        z.writestr("word/document.xml", doc)
        z.writestr("word/_rels/document.xml.rels", DOC_RELS)
        z.writestr("word/styles.xml", STYLES_XML)
        z.writestr("word/footer1.xml", FOOTER_XML)


def default_date():
    n = datetime.date.today()
    return f"{n.year}年{n.month}月{n.day}日"


def default_docnum():
    n = datetime.date.today()
    return f"×政发〔{n.year}〕1号"


def main(argv=None):
    ap = argparse.ArgumentParser(description="GB/T 9704 空白公文 docx 生成器(标准库)")
    ap.add_argument("--org", required=True, help="发文机关(红头,红色大字)")
    ap.add_argument("--title", required=True, help="公文标题(关于××的文种)")
    ap.add_argument("--docnum", default=default_docnum(), help="发文字号(六角括号〔〕)")
    ap.add_argument("--to", default="各有关单位：", help="主送机关(顶格称谓行)")
    ap.add_argument("--date", default=default_date(), help="成文日期(阿拉伯数字)")
    body = ap.add_mutually_exclusive_group()
    body.add_argument("--body-file", help="正文文本文件(UTF-8,每行一段)")
    body.add_argument("--body", help="正文(\\n 分段)")
    ap.add_argument("--cc", default="抄送：××。", help="版记-抄送")
    ap.add_argument("--print-org", default="××市人民政府办公室", help="版记-印发机关")
    ap.add_argument("-o", "--out", default="公文.docx", help="输出路径")
    args = ap.parse_args(argv)

    if args.body_file:
        with open(args.body_file, "r", encoding="utf-8") as f:
            lines = f.read().splitlines()
    elif args.body:
        lines = args.body.replace("\\n", "\n").splitlines()
    else:
        lines = []

    build_docx(args.out, args.org, args.docnum, args.title, args.to,
               args.date, lines, args.cc, args.print_org)
    print(args.out)
    return 0


if __name__ == "__main__":
    sys.exit(main())
