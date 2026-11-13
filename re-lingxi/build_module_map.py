"""H 批 v2：用 Rust 模块路径（src\\commands\\*.rs）划区，把 JS 片段归到模块家族。
产物: out/WPSCLI-module-map.md
"""
import os
import re
from collections import Counter, defaultdict

WPS = os.path.join(os.environ["LOCALAPPDATA"], r"Programs\lingxi-desktop\resources\wps-cli\wpscli.exe")
OUT_JS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", "WPSCLI-jsapi-bodies.md")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")
raw = open(WPS, "rb").read().decode("utf-8", "replace")

# 1) 模块路径标记
mods = [(m.start(), m.group(1)) for m in re.finditer(r"(?:src|jsapi)[\\/]([a-z0-9_]+(?:\.[a-z0-9_]+)*)\.rs", raw)]
mods = [(o, n) for o, n in mods if n]
names = Counter(n for _, n in mods)
print("模块标记:", len(mods), "种:", len(names))
for n, c in names.most_common(20):
    print(f"   {n:26} {c}")

# 2) 片段
frags = []
t = open(OUT_JS, encoding="utf-8").read()
for m in re.finditer(r"### \d+\. offset `(\d+)` · (\d+) 字符\n\n```javascript\n(.*?)\n```", t, re.S):
    frags.append((int(m.group(1)), int(m.group(2)), m.group(3)))

# 3) 归属：取最近的模块标记（同距离取后出现的）
mods_sorted = sorted(mods)


def nearest_module(off):
    lo, hi = 0, len(mods_sorted) - 1
    while lo < hi:
        mid = (lo + hi) // 2
        if mods_sorted[mid][0] < off:
            lo = mid + 1
        else:
            hi = mid
    span = []
    if lo > 0:
        span.append((mods_sorted[lo - 1][0], mods_sorted[lo - 1][1]))
    if lo < len(mods_sorted):
        span.append((mods_sorted[lo][0], mods_sorted[lo][1]))
    if not span:
        return "?", None
    o, n = min(span, key=lambda x: abs(x[0] - off))
    return n, abs(o - off)


by_mod = defaultdict(list)
for off, ln, body in frags:
    mod, d = nearest_module(off)
    by_mod[mod].append((off, ln, d, body))

md = ["# wpscli：JS 片段 → Rust 模块家族（按模块路径标记划区）", "",
      f"- 源: `{WPS}`",
      f"- 模块路径标记 {len(mods)} 个 / {len(names)} 种；片段 {len(frags)} 条",
      "- 归属：取片段 offset 最近的 `src/*.rs` 或 `jsapi/*.rs` 路径标记；距离列给可靠性", "",
      "## 模块 × 片段数", "", "| 模块 | 片段数 |", "|---|---|"]
for m in sorted(by_mod, key=lambda k: -len(by_mod[k])):
    md.append(f"| `{m}.rs` | {len(by_mod[m])} |")

md += ["", "## 逐片段", "", "| offset | 模块 | 距离 | 长度 | 开头 |", "|---|---|---|---|---|"]
for m in sorted(by_mod):
    for off, ln, d, body in sorted(by_mod[m]):
        head = re.sub(r"\s+", " ", body.strip())[:80].replace("|", "/")
        md.append(f"| `{off}` | `{m}.rs` | {d} | {ln} | `{head}` |")

md += ["", "## 工具家族 ↔ 模块（按命名对应，非反编译结论）", "",
       "| 工具前缀 | 预期模块 | 依据 |", "|---|---|---|",
       "| `wps_doc_*` | `commands/doc.rs` | 工具名与模块名同族；描述里出现 `src\\commands\\doc.rs` |",
       "| `wps_sheet_*` | `commands/sheet*.rs` | 同上 |",
       "| `wps_ppt_*` | `commands/ppt*.rs` | 同上 |",
       "| `wps_pdf_*` | `jsapi/pdf_scripts.rs` + `commands/pdf*.rs` | `pdf_scripts.rs` 已命中 |",
       "| `wps_status` / `wps_search` / `wps_file_info` | `doctor.rs` / 无 | 无 JS 片段 |"]

open(os.path.join(OUT, "WPSCLI-module-map.md"), "w", encoding="utf-8").write("\n".join(md))
print("wrote WPSCLI-module-map.md")
for m in sorted(by_mod, key=lambda k: -len(by_mod[k]))[:12]:
    print(f"  {m:24} {len(by_mod[m])}")
