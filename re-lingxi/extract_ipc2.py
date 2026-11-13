"""F 批补 4：IPC 通道来自 bundle 里的常量表 `t.XxxYyy="ns:method"`，按此抽取全表。"""
import os
import re
from collections import defaultdict

MAIN = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", "x", "out", "main", "index.js")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")
js = open(MAIN, encoding="utf-8", errors="replace").read()

pairs = re.findall(r'\b([A-Za-z_$][\w$]*)\.([A-Z][A-Za-z0-9_]{2,60})\s*=\s*"([a-z][A-Za-z0-9]*(?::[A-Za-z0-9_]+){1,3})"', js)
print("常量→通道 命中:", len(pairs))
seen = {}
for mod, const, ch in pairs:
    seen[(const, ch)] = True
uniq = sorted(seen)
print("去重:", len(uniq))

by_ns = defaultdict(list)
for const, ch in uniq:
    ns = ch.split(":", 1)[0]
    by_ns[ns].append((const, ch))

md = ["# 灵犀 Electron IPC 通道全表（常量表抽取）", "",
      f"- 源: `{MAIN}`",
      f"- 形态: `t.AgentSteerRun=\"agent:steerRun\"`（bundle 内导出的通道常量表）",
      f"- 命中 {len(pairs)} 次 → 去重 **{len(uniq)}** 个通道，**{len(by_ns)}** 个命名空间", "",
      "## 命名空间 × 通道数", "", "| 命名空间 | 通道数 |", "|---|---|"]
for ns in sorted(by_ns, key=lambda k: -len(by_ns[k])):
    md.append(f"| `{ns}` | {len(by_ns[ns])} |")
md += ["", "## 全量（常量名 → 通道）", ""]
for ns in sorted(by_ns):
    md.append(f"### `{ns}`")
    md.append("")
    for const, ch in sorted(by_ns[ns], key=lambda x: x[1]):
        md.append(f"- `{const}` → `{ch}`")
    md.append("")
open(os.path.join(OUT, "IPC-channels.md"), "w", encoding="utf-8").write("\n".join(md))
print("wrote IPC-channels.md")
for ns in sorted(by_ns, key=lambda k: -len(by_ns[k]))[:18]:
    print(f"  {ns:16} {len(by_ns[ns])}")
