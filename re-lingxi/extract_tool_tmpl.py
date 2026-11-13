"""抽取 lingxi.exe 内嵌的 tools/*.tmpl 模板库：{{ define "x" }}…{{ end }} 块，按物理位置聚类。
用法: python extract_tool_tmpl.py [out.md]
"""
import os
import re
import sys
from collections import defaultdict

EXE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service\lingxi.exe")
raw = open(EXE, "rb").read().decode("utf-8", "replace")

# 允许 trim 标记：{{ define "x" -}} … {{- end }}
DEF = re.compile(r'\{\{-?\s*define\s+"([A-Za-z0-9_]+)"\s*-?\}\}(.{0,6000}?)\{\{-?\s*end\s*-?\}\}', re.S)

blocks = []
for m in DEF.finditer(raw):
    name, body = m.group(1), m.group(2)
    body = re.sub(r"[ \t]+", " ", body).strip()
    if not body or len(body) < 4:
        continue
    blocks.append((m.start(), name, body))

# 去重（同名同体）
seen = set()
uniq = []
for off, name, body in blocks:
    key = (name, body)
    if key in seen:
        continue
    seen.add(key)
    uniq.append((off, name, body))
uniq.sort()

# 按物理邻近聚类（间隔 > 2500 字节视为另一个模板文件）
clusters = []
cur = []
for off, name, body in uniq:
    if cur and off - cur[-1][0] > 2500:
        clusters.append(cur)
        cur = []
    cur.append((off, name, body))
if cur:
    clusters.append(cur)

md = ["# 灵犀 tools/*.tmpl 内嵌模板库（静态抽取）", "",
      f"- 源: `{EXE}`",
      f"- `{{{{ define \"…\" }}}}` 块命中 {len(blocks)} 个，去重后 **{len(uniq)}** 个，聚成 **{len(clusters)}** 个模板簇",
      "- 说明：聚类按物理邻近（间隔 > 2500 字节切分），簇 ≈ 一个 .tmpl 文件；簇标题里的 `?` 表示该簇没有可辨识的模板名，",
      "  用簇内 define 名归猜工具归属。", ""]

# 猜工具归属：看簇内 define 名/正文关键词
KEY = [("deliver_result", ["cloud_doc", "type_arg", "ref_arg"]),
       ("share_file", ["message_arg", "missing_paths"]),
       ("url_fetch", ["no_urls_provided", "status_failed", "error_prefix"]),
       ("web", ["ref_id", "action=fetch"]),
       ("view_image", ["question_arg", "paths_arg"]),
       ("image_search", ["no_queries", "image"]),
       ("timer_task", ["schedule", "add_failed", "list_success"]),
       ("search", ["no_results", "search_results"]),
       ("render_widget", ["widget"]),
       ("create_plan", ["plan"]),
       ("todo_write", ["todo"]),
       ("connector_exec", ["connector"]),
       ("bash", ["command"]),
       ("file", ["action"])]


def guess(cluster):
    txt = " ".join(n + " " + b for _, n, b in cluster).lower()
    for tool, keys in KEY:
        if any(k.lower() in txt for k in keys):
            return tool
    return "?"


for i, c in enumerate(clusters, 1):
    tool = guess(c)
    names = ", ".join(sorted({n for _, n, _ in c}))
    md.append(f"## 簇 {i} · offset `{c[0][0]}` · 猜归属 `{tool}`")
    md.append("")
    md.append(f"define: `{names}`")
    md.append("")
    for off, name, body in c:
        md.append(f"### `{name}`")
        md.append("")
        md.append("```text")
        md.append(body)
        md.append("```")
        md.append("")

out = sys.argv[1] if len(sys.argv) > 1 else "TOOLS-tmpl.md"
open(out, "w", encoding="utf-8").write("\n".join(md))
print(f"blocks={len(blocks)} uniq={len(uniq)} clusters={len(clusters)}")
for i, c in enumerate(clusters, 1):
    print(f"  簇{i} off={c[0][0]} n={len(c)} guess={guess(c)} defines={','.join(sorted({n for _,n,_ in c}))[:110]}")
