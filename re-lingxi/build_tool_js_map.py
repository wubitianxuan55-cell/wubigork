"""H 批：wpscli 的 JS 片段 → 工具归属映射（按二进制内物理邻近）。
  1) 34 个语义工具 / 3 个 legacy 名表
  2) 129 条 JS 算子片段 + 13 个 IIFE 外壳的 offset
  3) 每条片段向前回溯找最近的工具名 → 归属；统计占位符
产物: out/WPSCLI-tool-js-map.md
"""
import os
import re
from collections import Counter, defaultdict

WPS = os.path.join(os.environ["LOCALAPPDATA"], r"Programs\lingxi-desktop\resources\wps-cli\wpscli.exe")
OUT_JS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", "WPSCLI-jsapi-bodies.md")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")

raw = open(WPS, "rb").read().decode("utf-8", "replace")

# ---------- 工具名表 ----------
ALL = sorted(set(re.findall(r"wps_[a-z]+(?:_[a-z]+)*", raw)))
ALL = [n for n in ALL if not re.search(r"(exewps|_vers$|^wps_w$)", n)]
ALL = [n for n in ALL if not any(o != n and o.startswith(n) for o in ALL)]
GROUPS = {"wps_doc", "wps_pdf", "wps_ppt", "wps_sheet"}
LEGACY = {"wps_writer", "wps_spreadsheet", "wps_presentation"}
TOOLS = [n for n in ALL if n not in GROUPS]

# 每个工具名的所有出现位置（取后面第一次出现作为“区域起点”）
occ = {}
for n in TOOLS:
    occ[n] = [m.start() for m in re.finditer(rf"(?<![a-z_]){re.escape(n)}(?![a-z_])", raw)]

# ---------- 片段：来自已抽好的 bodies md ----------
frags = []
t = open(OUT_JS, encoding="utf-8").read()
for m in re.finditer(r"### \d+\. offset `(\d+)` · (\d+) 字符\n\n```javascript\n(.*?)\n```", t, re.S):
    frags.append((int(m.group(1)), int(m.group(2)), m.group(3)))
print(f"片段 {len(frags)} 条")

# IIFE 外壳
iifes = []
for m in re.finditer(r"\(function\s*\(\s*\)\s*\{", raw):
    s = m.start()
    e = raw.find("})()", s)
    if 0 < e - s < 4000:
        iifes.append((s, e + 4))
print(f"IIFE {len(iifes)} 个")


def nearest_tool(off, window=10_000_000):
    best, bestd = None, window + 1
    for n, offs in occ.items():
        for o in offs:
            d = abs(o - off)
            if d < bestd:
                best, bestd = n, d
    return best, (bestd if best else None)


assign = defaultdict(list)
for off, ln, body in frags:
    tool, d = nearest_tool(off)
    ph = sorted(set(re.findall(r"__[A-Z_]{3,}__", body)))
    assign[tool or "?"].append((off, ln, d, ph, body))

tool_shells = defaultdict(list)
for s, e in iifes:
    tool, d = nearest_tool(s)
    tool_shells[tool or "?"].append((s, e - s, d))

md = ["# wpscli：JS 片段 → 工具归属（物理邻近映射）", "",
      f"- 源: `{WPS}`",
      f"- 语义工具 **{len(TOOLS)}** 个（含 legacy），片段 **{len(frags)}** 条，IIFE 外壳 **{len(iifes)}** 个",
      "- 归属规则：逐片段找**最近的工具名**（不设窗口），距离列给出可靠性；距离越大越可疑", "",
      "## 工具 × 片段数", "", "| 工具 | 片段数 | 外壳数 | 用到占位符 |", "|---|---|---|---|"]
for tool in sorted(assign, key=lambda k: -len(assign[k])):
    phs = sorted({p for rec in assign[tool] for p in rec[3]})
    md.append(f"| `{tool}` | {len(assign[tool])} | {len(tool_shells.get(tool, []))} | "
              f"{', '.join(phs[:6])}{'…' if len(phs) > 6 else ''} |")

md += ["", "## 逐片段明细", "", "| offset | 工具 | 距离 | 长度 | 占位符 | 开头 |", "|---|---|---|---|---|---|"]
for tool in sorted(assign):
    for off, ln, d, ph, body in sorted(assign[tool]):
        head = re.sub(r"\s+", " ", body.strip())[:70].replace("|", "/")
        md.append(f"| `{off}` | `{tool}` | {d} | {ln} | {', '.join(ph) or '—'} | `{head}` |")

md += ["", "## 没有 JS 片段的工具（= 纯 Rust 实现，不起 WPS 进程）", ""]
empty = [n for n in TOOLS if n not in assign]
for n in empty:
    md.append(f"- `{n}`")
md += ["", "## 外壳归属", "", "| 外壳 offset | 长度 | 归属工具 | 距离 |", "|---|---|---|---|"]
for tool in sorted(tool_shells):
    for s, ln, d in sorted(tool_shells[tool]):
        md.append(f"| `{s}` | {ln} | `{tool}` | {d} |")

open(os.path.join(OUT, "WPSCLI-tool-js-map.md"), "w", encoding="utf-8").write("\n".join(md))
print("wrote WPSCLI-tool-js-map.md")
for tool in sorted(assign, key=lambda k: -len(assign[k])):
    print(f"  {tool:22} frags={len(assign[tool]):3} shells={len(tool_shells.get(tool,[])):2}")
print("无片段工具:", ", ".join(empty))
