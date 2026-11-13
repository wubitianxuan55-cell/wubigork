"""抽取 wpscli.exe 内的 JSAPI「算子片段」（各 op 的 JS 主体），与 13 个 IIFE 外壳分开。
   口径：含 Application./ActivePresentation/ActiveDocument/Sheets/Shapes 等 WPS 对象模型的
   可读文本片段，且看起来是语句（含 ; / = / JSON.stringify）。
用法: python extract_js_body.py [out.md]
"""
import re
import sys

P = r"C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe"
raw = open(P, "rb").read().decode("utf-8", "replace")

OBJ = re.compile(r"(Application\.|ActiveDocument|ActivePresentation|ActiveSheet|Documents\.Item|"
                 r"Presentations\.Add|Workbooks\.Add|\.Shapes|\.Range\(|\.Slides|\.SaveAs|"
                 r"JSON\.stringify|OCR\()")
STMT = re.compile(r"(;|=[^=]|\bvar\b|\bfunction\b)")

# 取可读串（≥40 字符），过滤掉非 JS 噪声
cands = []
for m in re.finditer(r"[\x20-\x7e\u00a0-\uffff]{40,4000}", raw):
    t = m.group(0)
    if not OBJ.search(t):
        continue
    if not STMT.search(t):
        continue
    if "\\x00" in t:
        continue
    # 去掉明显的二进制夹带
    if sum(1 for c in t if ord(c) < 9 or (13 < ord(c) < 32)) > 3:
        continue
    cands.append((m.start(), t))

# 去重 + 剔除外壳（含 __BODY__ 且仅外壳结构）
shell = re.compile(r"^\(function\s*\(\)\s*\{\s*(//.*\n)?\s*try\s*\{\s*(Application\.DisplayAlerts = false;)?", re.S)
seen = {}
for off, t in cands:
    if t in seen:
        continue
    seen[t] = off

items = sorted(seen.items(), key=lambda kv: kv[1])
md = ["# wpscli.exe 内嵌 JSAPI 算子片段（静态抽取）", "",
      f"- 二进制: `{P}`",
      f"- 命中含 WPS 对象模型的 JS 文本片段 **{len(items)}** 条（去重后）",
      "- 用途：看清每个 op 真正怎么驱动 WPS JSAPI（选区/范围/形状/导出），供 gaea COM 侧对齐语义", ""]
for i, (t, off) in enumerate(items, 1):
    md.append(f"### {i}. offset `{off}` · {len(t)} 字符")
    md.append("")
    md.append("```javascript")
    md.append(t.strip())
    md.append("```")
    md.append("")
open(sys.argv[1] if len(sys.argv) > 1 else "WPSCLI-jsapi-bodies.md", "w", encoding="utf-8").write("\n".join(md))
print("snippets:", len(items))
for t, off in items[:12]:
    print(f"  {off:>9} {len(t):>5}  {t.strip()[:110]!r}")
