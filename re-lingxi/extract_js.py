"""从 wpscli.exe 抽取内嵌 JSAPI 脚本（每个工具 op 的 JS 实现）与占位符契约。
   静态：IIFE 以 (function() { ... })() 形式内嵌，前后夹杂 op 名/错误串。
用法: python extract_js.py [out.md]
"""
import re
import sys

P = r"C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe"
raw = open(P, "rb").read().decode("utf-8", "replace")


def match_iife(s, start):
    """从 start 处的 '(' 起做括号配对，跳过字符串/模板串。返回结束下标。"""
    depth = 0
    i = start
    n = len(s)
    quote = None
    while i < n:
        c = s[i]
        if quote:
            if c == "\\":
                i += 2
                continue
            if c == quote:
                quote = None
            i += 1
            continue
        if c in "'\"`":
            quote = c
            i += 1
            continue
        if c == "(":
            depth += 1
        elif c == ")":
            depth -= 1
            if depth == 0:
                return i + 1
        i += 1
    return -1


starts = [m.start() for m in re.finditer(r"\(function\s*\(\)\s*\{", raw)]
scripts = []
for s in starts:
    e = match_iife(raw, s)
    if e < 0:
        continue
    body = raw[s:e]
    if len(body) < 60:
        continue
    pre = re.sub(r"[\x00-\x1f]", " ", raw[max(0, s - 320):s]).strip()
    scripts.append((s, e, pre[-200:], body))

# 去重（同样的脚本可能在多处重复）
seen = {}
for s, e, pre, body in scripts:
    seen.setdefault(body, (s, e, pre))
print(f"raw blocks={len(scripts)} unique={len(seen)}")

ph = sorted(set(re.findall(r"__[A-Z_]{3,}__", raw)))

md = ["# wpscli.exe 内嵌 JSAPI 脚本（静态抽取）", "",
      f"- 二进制: `{P}`（sha256 `796966c1…`，见 REA Evidence `ev_bd7498ba…`）",
      f"- 命中 IIFE 块 {len(scripts)} 个，去重后 **{len(seen)} 个**",
      "- 抽取方式：二进制内 UTF-8 字符串做括号配对（跳过字符串字面量），未执行任何 WPS 进程",
      f"- 占位符契约（{len(ph)}）：" + "、".join(f"`{x}`" for x in ph), "",
      "## 脚本清单（按文件偏移）", ""]
for idx, (body, (s, e, pre)) in enumerate(sorted(seen.items(), key=lambda kv: kv[1][0]), 1):
    md.append(f"### {idx}. offset `{s}` · {len(body)} 字符")
    if pre:
        md.append("")
        md.append(f"上下文（op 名/错误串）：`{pre}`")
    md.append("")
    md.append("```javascript")
    md.append(body)
    md.append("```")
    md.append("")
open(sys.argv[1] if len(sys.argv) > 1 else "WPSCLI-jsapi-scripts.md", "w", encoding="utf-8").write("\n".join(md))
print("placeholders:", len(ph))
print(", ".join(ph))
