"""I 批：一次出四份
  1. out/PROMPT-skeleton.md —— 主提示词模板的段顺序骨架（按二进制顺序）
  2. out/MEMORY-spec.md     —— 记忆子系统规格（段库合订）
  3. out/WPSCLI-placeholders.md —— 66 个占位符的用法/类型
  4. out/STREAM-events.md   —— 流式事件/增量类型清单
"""
import os
import re
from collections import Counter, defaultdict

BASE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service")
EXE = os.path.join(BASE, "lingxi.exe")
WPS = os.path.join(os.environ["LOCALAPPDATA"], r"Programs\lingxi-desktop\resources\wps-cli\wpscli.exe")
HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "out")
SEC = os.path.join(OUT, "prompts", "sections")

lx = open(EXE, "rb").read().decode("utf-8", "replace")
wp = open(WPS, "rb").read().decode("utf-8", "replace")

# ---------------------------------------------------------------- 1) 提示词骨架
print("== 1) 提示词骨架 ==")
# 主模板区（含 Prolog/Epilog 与 agents_main_v2 骨架）
lo, hi = 40_380_000, 41_070_000
region = lx[lo:hi]
tags = [(m.start(), m.group(1)) for m in re.finditer(r"<([a-z][a-z0-9_]{2,40})>", region)]
# 只保留已知段名，压缩连续重复
KNOWN = set()
for f in os.listdir(SEC):
    if f.endswith(".md"):
        KNOWN.add(f[:-3])
seq = [(o + lo, t) for o, t in tags if t in KNOWN]
compact = []
for off, t in seq:
    if compact and compact[-1][1] == t and off - compact[-1][0] < 4000:
        continue
    compact.append((off, t))
print(f"  段出现 {len(seq)} 次 → 压缩 {len(compact)} 个节点")

md = ["# 灵犀主提示词骨架（按二进制顺序）", "",
      f"- 源: `{EXE}`，区间 `{lo}`–`{hi}`",
      f"- 已知段 {len(KNOWN)} 个；该区间内段出现 {len(seq)} 次（压缩连续重复后 {len(compact)} 个节点）",
      "- 读法：这是**装配顺序**（模板 include 展开后的样子），段正文见 `prompts/sections/<tag>.md`", "",
      "## 装配顺序", "", "| # | offset | 段 |", "|---|---|---|"]
for i, (off, t) in enumerate(compact, 1):
    md.append(f"| {i} | `{off}` | `<{t}>` |")
md += ["", "## 段频次（该区间内）", "", "| 段 | 次数 |", "|---|---|"]
for t, c in Counter(t for _, t in seq).most_common(60):
    md.append(f"| `<{t}>` | {c} |")
open(os.path.join(OUT, "PROMPT-skeleton.md"), "w", encoding="utf-8").write("\n".join(md))

# ---------------------------------------------------------------- 2) 记忆规格
print("== 2) 记忆规格 ==")
MEM = ["memory_system", "core_memory", "memory_index", "memory_recall_rules", "memory_write_rules",
       "memory_management", "memory_reference_rules", "memory_disabled", "knowledge_base",
       "import_rules", "fragments_format", "rollout_summary", "existing_user_profile",
       "user_profile", "wiki"]
parts = []
for name in MEM:
    p = os.path.join(SEC, f"{name}.md")
    if os.path.exists(p):
        parts.append(open(p, encoding="utf-8").read())
# wiki_memory 工具文案（模板 define）
defs = {}
t = open(os.path.join(OUT, "TOOLS-tmpl.md"), encoding="utf-8").read()
for m in re.finditer(r"### `(wiki_memory_[a-z0-9_]+|edit_memory_[a-z0-9_]+|write_memory_[a-z0-9_]+)`\n\n```text\n(.*?)\n```", t, re.S):
    defs.setdefault(m.group(1), m.group(2).strip())

md = ["# 灵犀记忆子系统规格（段库合订）", "",
      f"- 源: `prompts/sections/`（{len(parts)} 个段）+ `TOOLS-tmpl.md`（{len(defs)} 条记忆工具文案）",
      "- 用途：这部分是灵犀「长期记忆」的完整设计说明（召回规则、写入门禁、wiki 页面格式）", ""]
for p, name in zip(parts, [n for n in MEM if os.path.exists(os.path.join(SEC, f"{n}.md"))]):
    md.append(f"\n---\n\n<!-- 段: {name} -->\n")
    md.append(p)
if defs:
    md += ["\n---\n", "## 记忆工具的文案（模板 define）", ""]
    for k, v in sorted(defs.items()):
        md.append(f"### `{k}`")
        md.append("")
        md.append("```text")
        md.append(v)
        md.append("```")
        md.append("")
open(os.path.join(OUT, "MEMORY-spec.md"), "w", encoding="utf-8").write("\n".join(md))
print(f"  段 {len(parts)} / 工具文案 {len(defs)}")

# ---------------------------------------------------------------- 3) 占位符契约
print("== 3) 占位符 ==")
ph_all = sorted(set(re.findall(r"__[A-Z_]{3,}__", wp)))
stmts = open(os.path.join(OUT, "WPSCLI-js-statements.md"), encoding="utf-8").read()
usage = defaultdict(list)
for ph in ph_all:
    cnt = wp.count(ph)
    i = wp.find(ph)
    ctx = re.sub(r"\s+", " ", wp[max(0, i - 60):i + len(ph) + 60]) if i >= 0 else ""
    usage[ph].append(f"×{cnt} {ctx}")


def kind(ph):
    p = ph.strip("_")
    if any(k in p for k in ("PATH", "DIR", "FILE")):
        return "路径"
    if any(k in p for k in ("SLIDE", "PH_INDEX", "PARA_INDEX", "INDEX")):
        return "索引"
    if any(k in p for k in ("RANGE", "START", "END", "PAGE", "COUNT")):
        return "数值/范围"
    if any(k in p for k in ("TEXT", "TITLE", "SUBTITLE", "VALUE", "QUERY", "FIND", "REPLACE")):
        return "文本"
    if any(k in p for k in ("BODY", "JSON", "OPERATIONS", "INSERTS")):
        return "代码/JSON"
    return "其它"


md = ["# wpscli 占位符契约（66 个）", "",
      f"- 源: `{WPS}`；占位符来自模板拼装（`__X__` 形态，运行时字符串替换）",
      f"- 共 **{len(ph_all)}** 个", "",
      "| 占位符 | 推断类型 | 二进制内出现次数 | 上下文样例 |", "|---|---|---|---|"]
for ph in ph_all:
    u = usage.get(ph) or [""]
    m = re.match(r"×(\d+)\s*(.*)", u[0])
    cnt = m.group(1) if m else "0"
    ctx = (m.group(2) if m else "")[:70].replace("|", "/")
    md.append(f"| `{ph}` | {kind(ph)} | {cnt} | `{ctx or '—'}` |")
md += ["", "## 组合占位符（相邻两个占位符说明模板拼接处）", ""]
for ph in ph_all:
    if "__" in ph.strip("_") and ph.count("__") > 2:
        md.append(f"- `{ph}` → 拆为 " + " + ".join(f"`__{x}__`" for x in ph.strip("_").split("____") if x))
open(os.path.join(OUT, "WPSCLI-placeholders.md"), "w", encoding="utf-8").write("\n".join(md))
print(f"  占位符 {len(ph_all)}")

# ---------------------------------------------------------------- 4) 流式事件
print("== 4) 流式事件 ==")
EV = Counter()
GENERIC = {"error", "ping", "usage", "done"}
for k in ["content_block_delta", "content_block_start", "content_block_stop", "char_delta",
          "line_delta", "patch_delta", "message_start", "message_stop", "message_delta",
          "tool_call_start", "tool_call_delta", "tool_result", "title_updated", "reasoning_delta",
          "thinking_delta", "done", "error", "usage", "ping"]:
    c = lx.count(f'"{k}"')          # 只认带引号的字面量
    if c:
        EV[k] = c
sse = sorted(set(re.findall(r"(?:data|event):\s*([a-z_]{3,40})", lx)))[:60]
md = ["# 灵犀流式事件清单（静态抽取）", "",
      f"- 源: `{EXE}`（app-server 与 Electron 之间的 SSE/NDJSON 事件）",
      "- 口径：只统计字面量；动态拼接的事件名不在表内", "",
      "## 事件名频次（只数带引号的字面量）", "", "| 事件 | 次数 | 备注 |", "|---|---|---|"]
for k, v in EV.most_common():
    tag = "（泛词，含非事件用法）" if k in GENERIC else ""
    md.append(f"| `{k}` | {v} | {tag} |")
md += ["", "## `data:` / `event:` 行候选", ""]
for s in sse:
    md.append(f"- `{s}`")
open(os.path.join(OUT, "STREAM-events.md"), "w", encoding="utf-8").write("\n".join(md))
print("  events:", ", ".join(f"{k}({v})" for k, v in EV.most_common(10)))
print("done")
