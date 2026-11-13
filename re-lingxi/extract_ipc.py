"""F 批补 3：按 `X = require("electron")` → `X.ipcMain.handle(...)` 形态抽 IPC 通道。"""
import os
import re
from collections import Counter, defaultdict

MAIN = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", "x", "out", "main", "index.js")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")
js = open(MAIN, encoding="utf-8", errors="replace").read()

# 1) electron 模块别名
aliases = set(re.findall(r'([A-Za-z_$][\w$]{0,8})\s*=\s*require\("electron"\)', js))
print("electron module aliases:", aliases)

# 2) 解构别名（{ipcMain: X}）
for m in re.finditer(r'ipcMain:\s*([A-Za-z_$][\w$]{0,8})', js):
    aliases.add(m.group(1) + "::ipcMain")
for m in re.finditer(r'ipcRenderer:\s*([A-Za-z_$][\w$]{0,8})', js):
    aliases.add(m.group(1) + "::ipcRenderer")

calls = []
for a in aliases:
    if "::" in a:
        obj, kind = a.split("::")
        pat = re.escape(obj) + r'\.(handle|handleOnce|on|once|invoke|send|sendSync)\(\s*["\']([^"\']{3,64})["\']'
        for m in re.finditer(pat, js):
            calls.append((kind + "." + m.group(1), m.group(2)))
    else:
        pat = (re.escape(a) + r'\.(ipcMain|ipcRenderer)\.(handle|handleOnce|on|once|invoke|send|sendSync)'
               r'\(\s*["\']([^"\']{3,64})["\']')
        for m in re.finditer(pat, js):
            calls.append((m.group(1) + "." + m.group(2), m.group(3)))

print("call sites:", len(calls))
allch = Counter(ch for _, ch in calls)
by_ns = defaultdict(set)
for ch in allch:
    if ":" in ch:
        ns, meth = ch.split(":", 1)
        by_ns[ns].add(meth)
print("unique channels:", len(allch), "namespaces:", len(by_ns))

md = ["# 灵犀 Electron IPC 通道表（静态精确抽取）", "",
      f"- 源: `{MAIN}`",
      f"- electron 别名: `{sorted(aliases)}`",
      f"- 调用点 {len(calls)} 个 → 去重通道 **{len(allch)}** 个 / 命名空间 **{len(by_ns)}** 个", "",
      "## 命名空间 × 方法数", "", "| 命名空间 | 方法数 | 样例 |", "|---|---|---|"]
for ns in sorted(by_ns, key=lambda k: -len(by_ns[k])):
    ms = sorted(by_ns[ns])
    md.append(f"| `{ns}` | {len(ms)} | {', '.join(ms[:8])}{'…' if len(ms) > 8 else ''} |")
md += ["", "## 全量通道", ""]
for ch in sorted(allch):
    md.append(f"- `{ch}`")
open(os.path.join(OUT, "IPC-channels.md"), "w", encoding="utf-8").write("\n".join(md))
print("wrote IPC-channels.md")
for ns in sorted(by_ns, key=lambda k: -len(by_ns[k]))[:18]:
    print(f"  {ns:16} {len(by_ns[ns])}")
