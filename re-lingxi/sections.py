"""把 lingxi.exe 内嵌提示词按 `<section>…</section>` 自定界标签切成段库。
   产出 out/prompts/sections/<tag>.md（同名多版本用 --- 分隔），并打印段库索引。
用法: python sections.py
"""
import os
import re
from collections import defaultdict

EXE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service\lingxi.exe")
raw = open(EXE, "rb").read().decode("utf-8", "replace")
outdir = "out/prompts/sections"
os.makedirs(outdir, exist_ok=True)

pat = re.compile(r"<([a-z][a-z0-9_]{2,40})>((?:(?!</\1>).){40,20000}?)</\1>", re.S)
secs = defaultdict(list)
for m in pat.finditer(raw):
    tag, body = m.group(1), m.group(2)
    body = body.strip()
    if not body or body in secs[tag]:
        continue
    secs[tag].append(body)

index = []
for tag, bodies in sorted(secs.items()):
    uniq = sorted(bodies, key=len, reverse=True)
    p = os.path.join(outdir, f"{tag}.md")
    with open(p, "w", encoding="utf-8") as fh:
        fh.write(f"# <{tag}> · {len(uniq)} 个版本（按长度降序）\n\n")
        for i, b in enumerate(uniq, 1):
            fh.write(f"\n## 版本 {i}（{len(b)} 字符）\n\n```text\n{b}\n```\n")
    index.append((tag, len(uniq), len(uniq[0]), sum(len(x) for x in uniq)))

print(f"{'section':34}{'版本':>4}{'最长':>8}{'合计':>8}")
for tag, n, mx, tot in sorted(index, key=lambda x: -x[3]):
    print(f"<{tag + '>':34}{n:4}{mx:8}{tot:8}")
print(f"\n段标签 {len(index)} 个，目录: {outdir}")
