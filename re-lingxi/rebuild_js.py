"""H 批 v4：重建「真 JS 语句」片段集，并按文档类型清点对象模型调用。
判定：含 `var ` / `return ` / `function` 之一，且含 `(` … `)` 与 `;`，且不含成片中文。
产物: out/WPSCLI-js-statements.md
"""
import os
import re
from collections import Counter, defaultdict

WPS = os.path.join(os.environ["LOCALAPPDATA"], r"Programs\lingxi-desktop\resources\wps-cli\wpscli.exe")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")
raw = open(WPS, "rb").read().decode("utf-8", "replace")

CJK = re.compile(r"[\u4e00-\u9fff]")
cands = []
for m in re.finditer(r"[\x20-\x7e\u00a0-\uffff]{40,6000}", raw):
    s = m.group(0)
    if not re.search(r"(var |return |function|Application\.)", s):
        continue
    if not (re.search(r"\(", s) and re.search(r"\)", s) and ";" in s):
        continue
    if len(CJK.findall(s)) > len(s) * 0.12:      # 中文过半的当描述文本，剔除
        continue
    if s.count("\x00"):
        continue
    cands.append((m.start(), s))

# 去重（保留最长）
best = {}
for off, s in cands:
    key = s.strip()[:120]
    if key not in best or len(s) > len(best[key][1]):
        best[key] = (off, s)
items = sorted(best.values(), key=lambda x: x[0])
print(f"候选 {len(cands)} → 去重 {len(items)}")


def klass(s):
    for pat, name in [(r"Presentations|\.Slides|SlideIndex|pptx", "ppt"),
                      (r"Workbooks|Worksheets|\.Range\(|Sheets\(|Cells\(|UsedRange", "sheet"),
                      (r"\bPDF\b|ExportTo|Exports\.|pdfDoc", "pdf"),
                      (r"ActiveDocument|Application\.Documents|Paragraphs|Sections|Bookmarks", "doc"),
                      (r"OCR|InlineShapes|Picture", "image")]:
        if re.search(pat, s):
            return name
    return "other"


METH = re.compile(r"\.([A-Z][A-Za-z0-9_]{2,40})\s*\(")
PROP = re.compile(r"\.([A-Z][A-Za-z0-9_]{2,40})\s*=(?!=)")
OBJ = re.compile(r"\b(Application|ActiveDocument|ActivePresentation|ActiveSheet|ActiveWindow)")

per = defaultdict(lambda: {"n": 0, "meth": Counter(), "prop": Counter(), "obj": Counter()})
rows = []
for off, s in items:
    k = klass(s)
    per[k]["n"] += 1
    per[k]["meth"].update(METH.findall(s))
    per[k]["prop"].update(PROP.findall(s))
    per[k]["obj"].update(OBJ.findall(s))
    rows.append((off, k, s))

md = ["# wpscli：真 JS 语句片段（收紧判定）", "",
      f"- 源: `{WPS}`",
      f"- 候选 {len(cands)} → 去重 **{len(items)}** 条（要求含 `var/return/function` + 括号 + 分号，且中文占比 < 12%）",
      "", "## 按文档类型", "", "| 类型 | 片段数 | 对象模型入口 | 方法调用数 | 属性赋值数 |", "|---|---|---|---|---|"]
for k in sorted(per, key=lambda x: -per[x]["n"]):
    md.append(f"| {k} | {per[k]['n']} | {sum(per[k]['obj'].values())} | "
              f"{sum(per[k]['meth'].values())} | {sum(per[k]['prop'].values())} |")

for k in sorted(per, key=lambda x: -per[x]["n"]):
    md += ["", f"## {k}（{per[k]['n']}）", ""]
    if per[k]["meth"]:
        md.append("方法：" + "、".join(f"`{n}`" for n, _ in per[k]["meth"].most_common(30)))
        md.append("")
    if per[k]["prop"]:
        md.append("属性赋值：" + "、".join(f"`{n}`" for n, _ in per[k]["prop"].most_common(20)))
        md.append("")

md += ["", "## 片段清单", "", "| offset | 类型 | 开头 |", "|---|---|---|"]
for off, k, s in rows:
    head = re.sub(r"\s+", " ", s.strip())[:100].replace("|", "/")
    md.append(f"| `{off}` | {k} | `{head}` |")

open(os.path.join(OUT, "WPSCLI-js-statements.md"), "w", encoding="utf-8").write("\n".join(md))
print("wrote WPSCLI-js-statements.md")
for k in sorted(per, key=lambda x: -per[x]["n"]):
    print(f"  {k:6} n={per[k]['n']:3} obj={sum(per[k]['obj'].values()):3} "
          f"meth={sum(per[k]['meth'].values()):4} prop={sum(per[k]['prop'].values()):3} "
          f"| {', '.join(n for n, _ in per[k]['meth'].most_common(6))}")
