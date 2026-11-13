"""G 批：一次产出五份全量清单（只读灵犀产物）。
  1. out/GO-types.md          —— Go 类型名按包归组（全量）
  2. out/GO-error-codes.md    —— 错误码 / 状态码候选（CamelCase + snake_case）
  3. out/HTTP-routes.md       —— 路由候选按前缀归组
  4. out/PROMPT-tool-contracts.md —— 工具 → 参数 → 中文说明（符号 join 模板文案）
  5. out/WPSCLI-tool-js.md    —— wpscli 的 JS 片段按文档类型归组
用法: python extract_all.py
"""
import os
import re
from collections import Counter, defaultdict

BASE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service")
EXE = os.path.join(BASE, "lingxi.exe")
WPS = os.path.join(os.environ["LOCALAPPDATA"], r"Programs\lingxi-desktop\resources\wps-cli\wpscli.exe")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")

lx = open(EXE, "rb").read().decode("utf-8", "replace")
wp = open(WPS, "rb").read().decode("utf-8", "replace")

# ---------------------------------------------------------------- 1) Go 类型
print("== 1) Go 类型 ==")
type_re = re.compile(r"(\*?)(?:\[\])?([a-z][a-z0-9_]{1,20})\.([A-Z][A-Za-z0-9_]{2,60})")
counts = Counter()
for m in type_re.finditer(lx):
    pkg, name = m.group(2), m.group(3)
    if len(name) < 3 or name.isupper():
        continue
    counts[(pkg, name)] += 1
pkgs = defaultdict(set)
for (pkg, name), c in counts.items():
    pkgs[pkg].add((name, c))
rows = sorted(((p, len(v)) for p, v in pkgs.items()), key=lambda x: -x[1])
print(f"  类型名 {len(counts)} 个 / 包 {len(pkgs)} 个")
md = ["# 灵犀 Go 类型名清单（静态抽取，全量）", "",
      f"- 源: `{EXE}`",
      f"- 去重 (包, 类型) **{len(counts)}** 对 / 包 **{len(pkgs)}** 个",
      "- 口径：从类型串表（`*pkg.Type`、`[]*pkg.Type` 等）抽取；只反映**出现过的类型名**，不含字段",
      "", "## 包 × 类型数（前 40）", "", "| 包 | 类型数 |", "|---|---|"]
STD = {"runtime","internal","sync","reflect","syscall","unsafe","unicode","net","os","fmt",
       "strconv","strings","time","context","io","bytes","math","sort","path","crypto","encoding",
       "errors","flag","hash","html","index","log","mime","regexp","text","vendor","bufio","compress"}
for p, n in [r for r in rows if r[0] not in STD][:40]:
    md.append(f"| `{p}` | {n} |")
md += ["", "## 业务相关包的类型全量", ""]
for p in ["core", "agent", "session", "run", "skill", "connector", "mcp", "sandbox", "auth",
          "config", "store", "server", "protocol", "rpc", "prompt", "dw", "timer", "nativehost",
          "wpsclibroker", "experts"]:
    if p in pkgs:
        names = sorted({n for n, _ in pkgs[p]})
        md.append(f"### `{p}`（{len(names)}）")
        md.append("")
        md.append("、".join(f"`{n}`" for n in names[:160]))
        md.append("")
open(os.path.join(OUT, "GO-types.md"), "w", encoding="utf-8").write("\n".join(md))

# ---------------------------------------------------------------- 2) 错误码
print("== 2) 错误码 ==")
camel = Counter(re.findall(r"\b([A-Z][a-z]{2,}(?:[A-Z][a-z0-9]{2,}){1,3})\b", lx))
snake = Counter(re.findall(r"\b([a-z][a-z0-9]*(?:_[a-z0-9]+){1,3})\b", lx))
CODE_HINT = re.compile(r"(Unauthorized|Denied|Conflict|Invalid|NotFound|Exceeded|Unavailable|"
                       r"Timeout|Failed|Forbidden|Conflict|Running|Canceled|Cancelled|Rejected|"
                       r"Exhausted|Required|Mismatch|Unsupported|NotAllowed|Busy|Limit)")
codes = {k: v for k, v in camel.items() if CODE_HINT.search(k) and v >= 1}
snake_codes = {k: v for k, v in snake.items()
               if re.search(r"(denied|invalid|_failed|_error|not_found|too_large|exceeded|"
                            r"unauthorized|conflict|unavailable|timeout)", k) and v >= 1}
print(f"  CamelCase 码 {len(codes)} / snake 码 {len(snake_codes)}")
md = ["# 灵犀错误码 / 状态码清单（静态抽取）", "",
      f"- 源: `{EXE}`",
      f"- CamelCase 候选 **{len(codes)}** 个、snake_case 候选 **{len(snake_codes)}** 个",
      "- 口径：按命名特征（Unauthorized/Denied/Conflict/…、_failed/_error/not_found/…）筛，含少量非错误码同形词", "",
      "## CamelCase（按字母序）", "", "| 码 | 次数 |", "|---|---|"]
for k in sorted(codes):
    md.append(f"| `{k}` | {codes[k]} |")
md += ["", "## snake_case", "", "| 码 | 次数 |", "|---|---|"]
for k in sorted(snake_codes):
    md.append(f"| `{k}` | {snake_codes[k]} |")
open(os.path.join(OUT, "GO-error-codes.md"), "w", encoding="utf-8").write("\n".join(md))

# (3) 路由 与 (4) 契约 由 extract_contracts.py 生成，此处跳过

# ---------------------------------------------------------------- 5) wpscli JS 分组
print("== 5) wpscli JS 分组 ==")
body_md = os.path.join(OUT, "WPSCLI-jsapi-bodies.md")
snippets = []
if os.path.exists(body_md):
    t = open(body_md, encoding="utf-8").read()
    for m in re.finditer(r"### \d+\. offset `(\d+)` · (\d+) 字符\n\n```javascript\n(.*?)\n```", t, re.S):
        snippets.append((int(m.group(1)), int(m.group(2)), m.group(3)))
print(f"  片段 {len(snippets)} 条")


def klass(s):
    if re.search(r"Presentations|Slides|SlideIndex|pptx|Slide\b", s):
        return "ppt"
    if re.search(r"Workbooks|Worksheets|\.Range\(|Sheets\(|Cells\(|RangeBoundaries|UsedRange", s):
        return "sheet"
    if re.search(r"\bPDF\b|ExportTo|Exports\.|pdfDoc|Pages\.", s, re.I):
        return "pdf"
    if re.search(r"ActiveDocument|Application\.Documents|Paragraphs|Sections|Bookmarks|"
                 r"ContentControls|Fields|InlineShapes|Footnotes|Endnotes", s):
        return "doc"
    if re.search(r"OCR|InlineShapes|Picture|Image", s):
        return "image"
    return "other"


buckets = defaultdict(list)
for off, ln, body in snippets:
    buckets[klass(body)].append((off, ln, body))
md = ["# wpscli 内嵌 JS 片段按文档类型分组", "",
      f"- 源: `{WPS}`",
      f"- 片段 **{len(snippets)}** 条，分组: " + "、".join(f"{k}={len(v)}" for k, v in sorted(buckets.items())), ""]
for k in ["doc", "sheet", "ppt", "pdf", "image", "other"]:
    if k not in buckets:
        continue
    md.append(f"## {k}（{len(buckets[k])}）")
    md.append("")
    md.append("| offset | 长度 | 片段开头 |")
    md.append("|---|---|---|")
    for off, ln, body in sorted(buckets[k]):
        head = re.sub(r"\s+", " ", body.strip())[:90].replace("|", "/")
        md.append(f"| `{off}` | {ln} | `{head}` |")
    md.append("")
open(os.path.join(OUT, "WPSCLI-tool-js.md"), "w", encoding="utf-8").write("\n".join(md))
print("done")
