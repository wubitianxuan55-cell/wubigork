"""从 lingxi.exe 的 prompt.* / 其它导出符号还原「工具契约层」清单（v2，修正前缀归组）。"""
import re
import sys
from collections import defaultdict

P = r"C:\Users\wubi\AppData\Roaming\WPS 灵犀\app_server_5.24.0\service\lingxi.exe"
d = open(P, "rb").read().decode("latin-1")
names = sorted(set(re.findall(r"prompt\.([A-Z][A-Za-z0-9_]{3,60})", d)))

tool_keys = [n[: -len("ToolDesc")] for n in names if n.endswith("ToolDesc")]
tool_keys.sort(key=len, reverse=True)


def owner(n):
    for k in tool_keys:
        if n.startswith(k):
            return k
    return None


args = defaultdict(list)
props = defaultdict(list)
resps = defaultdict(list)
for n in names:
    k = owner(n)
    if not k:
        continue
    tail = n[len(k):]
    if tail.endswith("ArgDesc"):
        args[k].append(tail[: -len("ArgDesc")])
    elif tail.endswith("PropDesc"):
        props[k].append(tail[: -len("PropDesc")])
    elif tail.endswith(("Resp", "Err", "Reminder", "Prompt")):
        resps[k].append(tail)

all_syms = set(re.findall(r"\b([A-Z][A-Za-z0-9_]{4,60})\b", d))
extra = sorted(n for n in all_syms if re.match(r"(Deliver|GroupChat|ExecAuth|NopTool|StepLimit)", n))

md = ["# 灵犀工具契约层（由 lingxi.exe 符号还原 · v2）", "",
      f"- 二进制: `{P}`（sha256 `0b789ef5…`，REA Evidence `ev_e238164f…`）",
      f"- `prompt.*` 导出符号 **{len(names)}**，其中带 `*ToolDesc` 的注册工具 **{len(tool_keys)}** 个", "",
      "| 工具 | 参数 | 属性对象字段 | 响应/提醒族（条数） |", "|---|---|---|---|"]
for k in sorted(tool_keys):
    md.append(f"| `{k}` | {', '.join(sorted(set(args[k]))) or '—'} | "
              f"{', '.join(sorted(set(props[k]))) or '—'} | {len(set(resps[k]))} |")

md += ["", "## `share_file` 双契约", ""]
for n in names:
    if n.startswith("ShareFile"):
        md.append(f"- `{n}`")
md += ["", "→ 频道发送（`ShareFile*`）与云文档（`ShareFileCloudDoc*`）是两条路径 + 两套失败文案。", "",
       "## 非 `prompt.*` 但与之配套的符号（抽样）", ""]
for n in extra[:40]:
    md.append(f"- `{n}`")
md += ["", "## 与 A 批的对照", "",
       "- A 批：`tools/*.tmpl` 命中 **30** 个工具名（含 `deliver_result` `exec_auth` `group_chat` 等）。",
       f"- 本批：`prompt.*ToolDesc` 独立复核得 **{len(tool_keys)}** 个带描述的工具；",
       "  `deliver_result` / `exec_auth` 在本包里**没有** `*ToolDesc` 符号 → 它们不走 prompt 包，",
       "  是另一套（tools/*.tmpl + 各自 handler）出描述；`share_file` 反而有完整 prompt 契约。",
       "  ⇒ 「工具描述」在灵犀里是**双轨**：prompt 包（多数工具） + tmpl 文件（deliver_result 一族）。"]
open(sys.argv[1] if len(sys.argv) > 1 else "PROMPT-tool-contracts.md", "w", encoding="utf-8").write("\n".join(md))
print("\n".join(md[:40]))
print("\nextra 样本:", ", ".join(extra[:20]))
