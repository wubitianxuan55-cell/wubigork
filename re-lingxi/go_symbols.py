"""Go 二进制符号盘点（静态）：从 pclntab 名字串还原包结构 + 函数清单。
   适用于 lingxi.exe / lingxi-wpscli-proxy.exe / lingxi-sandbox-*.exe
用法: python go_symbols.py <exe> [out.md] [包过滤正则]
"""
import os
import re
import sys
from collections import Counter, defaultdict

EXE = sys.argv[1]
OUT = sys.argv[2] if len(sys.argv) > 2 else None
FILTER = re.compile(sys.argv[3]) if len(sys.argv) > 3 else None

raw = open(EXE, "rb").read().decode("latin-1")

# Go 符号形如 <import/path>.<Func> 或 <path>.(*Type).Method，路径里含斜杠与点号
sym_re = re.compile(r"[A-Za-z0-9_./\-]{6,200}\.[A-Za-z_][A-Za-z0-9_]{1,80}(?:\.func[0-9.]+)?")
cands = set()
for m in sym_re.finditer(raw):
    t = m.group(0)
    if "/" not in t:
        continue
    if not re.search(r"(ksogit\.kingsoft\.net|lingxi|pkg/mod|github\.com|golang\.org|go\.uber\.org|google\.golang\.org|modernc\.org|mvdan\.cc|go\.etcd\.io|go\.opentelemetry\.io)", t):
        continue
    cands.add(t)

# 归包：取最后一个 '/' 之前的路径 + 其后到第一个 '.' 的段
pkgs = defaultdict(set)
for t in cands:
    head = t.rsplit("/", 1)
    if len(head) != 2:
        continue
    pkg = head[0]
    rest = head[1]
    fn = rest.split(".")[0]
    pkgs[pkg].add(rest)

rows = sorted(((p, len(v)) for p, v in pkgs.items()), key=lambda x: -x[1])
total = sum(len(v) for v in pkgs.values())
print(f"{os.path.basename(EXE)}  symbols={total}  packages={len(pkgs)}")

if FILTER:
    print("\n== 过滤后的包 ==")
    for p, n in rows:
        if FILTER.search(p):
            print(f"  {n:5}  {p}")

if OUT:
    md = [f"# Go 符号盘点 · `{os.path.basename(EXE)}`", "",
          f"- 二进制: `{EXE}`",
          f"- 命中符号 **{total}** 个，**{len(pkgs)}** 个包（静态串扫描；不含类型/未导出内联信息）", "",
          "## 包 × 符号数（前 60）", "", "| 包 | 符号数 |", "|---|---|"]
    for p, n in rows[:60]:
        md.append(f"| `{p}` | {n} |")
    md += ["", "## 各包函数清单（按包名排序，包内符号数 ≥2 且属 lingxi/ksogit 与本仓相关的先列）", ""]
    for p, _ in rows:
        if not re.search(r"(lingxi|ksogit|wpscli|sandbox)", p):
            continue
        md.append(f"### `{p}`")
        md.append("")
        for fn in sorted(pkgs[p])[:400]:
            md.append(f"- `{fn}`")
        md.append("")
    open(OUT, "w", encoding="utf-8").write("\n".join(md))
    print("wrote", OUT)
