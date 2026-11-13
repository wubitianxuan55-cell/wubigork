"""H 批 v3：wpscli JS 片段里的 WPS 对象模型 API 调用清点（按文档类型）。
产物: out/WPSCLI-js-api-inventory.md
"""
import os
import re
from collections import Counter, defaultdict

OUT_JS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", "WPSCLI-jsapi-bodies.md")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")
t = open(OUT_JS, encoding="utf-8").read()

frags = [(int(m.group(1)), m.group(3)) for m in
         re.finditer(r"### \d+\. offset `(\d+)` · (\d+) 字符\n\n```javascript\n(.*?)\n```", t, re.S)]


def klass(s):
    for pat, name in [(r"Presentations|\.Slides|SlideIndex|pptx", "ppt"),
                      (r"Workbooks|Worksheets|\.Range\(|Sheets\(|Cells\(|UsedRange", "sheet"),
                      (r"\bPDF\b|ExportTo|Exports\.|pdfDoc", "pdf"),
                      (r"ActiveDocument|Application\.Documents|Paragraphs|Sections|Bookmarks|Fields", "doc"),
                      (r"OCR|InlineShapes|Picture", "image")]:
        if re.search(pat, s):
            return name
    return "other"


API = re.compile(r"\b(?:Application|ActiveDocument|ActivePresentation|ActiveSheet)\s*\.\s*"
                 r"([A-Za-z_][A-Za-z0-9_]*)")
METHOD = re.compile(r"\.([A-Z][A-Za-z0-9_]{2,40})\s*\(")
PROP = re.compile(r"\.([A-Z][A-Za-z0-9_]{2,40})\s*=(?!=)")

per = defaultdict(lambda: {"frags": 0, "app": Counter(), "meth": Counter(), "prop": Counter()})
for off, body in frags:
    k = klass(body)
    per[k]["frags"] += 1
    per[k]["app"].update(API.findall(body))
    per[k]["meth"].update(METHOD.findall(body))
    per[k]["prop"].update(PROP.findall(body))

md = ["# wpscli JS 片段里的 WPS 对象模型调用清点", "",
      f"- 源: `{OUT_JS}`（{len(frags)} 条片段）",
      "- 口径：`Application.X` / `.Method(` / `.Prop =` 三类静态计数；碎片片段会低估",
      "", "## 总量", "", "| 类型 | 片段数 | Application.* | 方法调用 | 属性赋值 |", "|---|---|---|---|---|"]
for k in sorted(per, key=lambda x: -per[x]["frags"]):
    md.append(f"| {k} | {per[k]['frags']} | {sum(per[k]['app'].values())} | "
              f"{sum(per[k]['meth'].values())} | {sum(per[k]['prop'].values())} |")

for k in ["doc", "sheet", "ppt", "pdf", "image", "other"]:
    if k not in per:
        continue
    md += ["", f"## {k}", ""]
    if per[k]["app"]:
        md.append("`Application.*`：" + "、".join(f"`{n}`×{c}" for n, c in per[k]["app"].most_common(20)))
        md.append("")
    if per[k]["meth"]:
        md.append("方法调用：" + "、".join(f"`{n}`" for n, c in per[k]["meth"].most_common(40)))
        md.append("")
    if per[k]["prop"]:
        md.append("属性赋值：" + "、".join(f"`{n}`" for n, c in per[k]["prop"].most_common(30)))
        md.append("")

open(os.path.join(OUT, "WPSCLI-js-api-inventory.md"), "w", encoding="utf-8").write("\n".join(md))
print("wrote WPSCLI-js-api-inventory.md")
for k in sorted(per, key=lambda x: -per[x]["frags"]):
    print(f"  {k:6} frags={per[k]['frags']:3} app={sum(per[k]['app'].values()):3} "
          f"meth={sum(per[k]['meth'].values()):4} prop={sum(per[k]['prop'].values()):3}")
    print("        app:", ", ".join(n for n, _ in per[k]["app"].most_common(8)))
