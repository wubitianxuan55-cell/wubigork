"""wpscli.exe 逆向辅助：rmcp 版本、自有模块路径、vendored crate 版本、JSAPI 脚本块位置。"""
import re
import sys

P = r"C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe"
d = open(P, "rb").read().decode("utf-8", "replace")

m = re.search(r"rmcp[-/]v?([0-9]+\.[0-9]+\.[0-9]+)", d)
print("rmcp version:", m.group(1) if m else "not found")

# 自有模块：形如 H:\master\src\<mod>.rs 或 src\jsapi\<mod>.rs
own = sorted(set(re.findall(r"(?:src|jsapi|tools|handlers|wps)[\\/]([a-z0-9_]+)\.rs", d)))
print("\nown .rs modules:", len(own))
print(", ".join(own))

# crate 版本：形如 rmcp-0.12.0 / tokio-1.48.0 路径片段
vers = sorted(set(re.findall(r"vendor[\\/]([a-z0-9_\-]+)[\\/]", d)))
print("\nvendor dirs:", len(vers))

# JS 脚本块
blocks = [m.start() for m in re.finditer(r"\(function\s*\(\)\s*\{", d)]
print("\nJS blocks:", len(blocks), "range", blocks[:3], "...", blocks[-3:] if blocks else None)

# 每个 JS 块前后 200 字符里的工具线索
seen = []
for off in blocks:
    ctx = d[max(0, off - 260):off]
    ctx = re.sub(r"[\x00-\x1f]", " ", ctx)
    tail = re.sub(r"[\x00-\x1f]", " ", d[off:off + 120])
    seen.append((off, ctx.strip()[-120:], tail.strip()[:80]))
for off, pre, body in seen[:26]:
    print(f"{off:>9} | pre: {pre!r} | js: {body!r}")
