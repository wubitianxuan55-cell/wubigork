"""E 批探针 3：kdocs-cli 的 wps.* JSAPI 动作命名空间 + 主进程 IAB 请求分发。"""
import os
import re
from collections import Counter

KD = os.path.join(os.environ["LOCALAPPDATA"], r"Programs\lingxi-desktop\resources\kdocs-cli\kdocs-cli.exe")
MAIN = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", "x", "out", "main", "index.js")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")

kd = open(KD, "rb").read().decode("utf-8", "replace")
acts = sorted(set(re.findall(r"wps\.[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+", kd)))
ns = Counter(a.split(".")[1] for a in acts)
print(f"wps.* 动作 {len(acts)} 个，命名空间 {len(ns)} 个")
for k, v in ns.most_common(30):
    print(f"  {k:22} {v}")

md = ["# kdocs-cli 的 `wps.*` JSAPI 动作命名空间（静态抽取）", "",
      f"- 源: `{KD}`",
      f"- 命中 `wps.<ns>.<op>` 形态动作 **{len(acts)}** 个，命名空间 **{len(ns)}** 个", "",
      "## 命名空间 × 动作数", "", "| 命名空间 | 动作数 |", "|---|---|"]
for k, v in ns.most_common():
    md.append(f"| `wps.{k}` | {v} |")
md += ["", "## 全量动作", ""]
for a in acts:
    md.append(f"- `{a}`")
open(os.path.join(OUT, "KDOCS-wps-actions.md"), "w", encoding="utf-8").write("\n".join(md))

print("\n== 主进程 IAB：分发与路由线索 ==")
js = open(MAIN, encoding="utf-8", errors="replace").read()
for k in ["iab", "IAB", "56305", "local-file", "handleRequest", "pathname", "req.url"]:
    print(f"  {k:14} {js.count(k)}")

i = js.find("handleRequest")
seg = js[max(0, i - 100):i + 1500] if i > 0 else ""
print("\n  handleRequest ctx:", re.sub(r"\s+", " ", seg)[:1200])

for m in list(re.finditer(r'["\'`](/(?:screenshot|tabs|navigate|evaluate|click|type|snapshot|health|status|close|open)[a-z0-9/_\-]*)["\'`]', js))[:20]:
    print("  route:", m.group(1))
