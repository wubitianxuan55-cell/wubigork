"""G 批修正：① 工具契约按「模板簇」做 join（避免同名 define 跨工具串味）② 路由表只保留云 API 前缀。
用法: python extract_contracts.py
"""
import os
import re
from collections import Counter, defaultdict

BASE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service")
EXE = os.path.join(BASE, "lingxi.exe")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")
lx = open(EXE, "rb").read().decode("utf-8", "replace")

# ---------------- 1) 解析 TOOLS-tmpl.md 的簇 ----------------
tmpl = open(os.path.join(OUT, "TOOLS-tmpl.md"), encoding="utf-8").read()
clusters = []          # [(cluster_no, offset, {define: body})]
for m in re.finditer(r"## 簇 (\d+) · offset `(\d+)`.*?\n(.*?)(?=\n## 簇 |\Z)", tmpl, re.S):
    no, off, body = int(m.group(1)), int(m.group(2)), m.group(3)
    defs = {}
    for dm in re.finditer(r"### `([a-z0-9_]+)`\n\n```text\n(.*?)\n```", body, re.S):
        name, val = dm.group(1), dm.group(2).strip()
        if len(val) > len(defs.get(name, "")):
            defs[name] = val
    clusters.append((no, off, defs))
print(f"簇 {len(clusters)} 个，define 总数 {sum(len(d) for _,_,d in clusters)}")

SIGN = {
    "deliver_result": ["ref_arg", "type_arg"],
    "share_file": ["message_arg", "missing_paths"],
    "view_image": ["all_load_failed", "too_many_images"],
    "render_widget": ["code_arg"],
    "bash": ["justification_arg", "run_in_background_arg"],
    "connector_exec": ["call_missing_tool"],
    "connector_manage": ["endpoint_arg", "add_missing_mode"],
    "execute_jupyter": ["kernel_failed_user"],
    "timer_task": ["add_success", "list_success"],
    "url_fetch": ["no_urls_provided"],
    "image_search": ["no_queries", "no_results"],
    "search": ["no_results", "search_results"],
    "await": ["cancelled", "exec_error"],
    "todo_write": ["todos_arg", "merge_arg"],
    "create_plan": ["edits_arg", "overview_arg"],
    "manage_skills": ["names_arg", "path_arg"],
    "web": ["ref_id", "action"],
    "members": ["members_action_arg"],
    "file": ["start_line_arg", "end_line_arg"],
    "wiki_memory": ["new_content_arg", "old_text_arg"],
    "ask_user_question": ["option_label_prop", "option_description_prop"],
    "gen_image": ["aspect_ratio_arg", "input_images_arg"],
    "switch_mode": ["target_mode_arg"],
    "memory": ["edit_memory_old_not_found"],
}
tool_cluster = {}
for tool, sigs in SIGN.items():
    best = None
    for no, off, defs in clusters:
        score = sum(1 for s in sigs if s in defs)
        if score >= max(1, len(sigs) // 2):
            if best is None or score > best[0]:
                best = (score, no)
    if best:
        tool_cluster[tool] = best[1]
print("工具→簇:", tool_cluster)

# ---------------- 2) 符号 → 工具/参数 ----------------
prompt = sorted(set(re.findall(r"prompt\.([A-Z][A-Za-z0-9_]{3,60})", lx)))
tools = sorted({n[: -len("ToolDesc")] for n in prompt if n.endswith("ToolDesc")}, key=len, reverse=True)


def snake(s):
    parts = re.findall(r"[A-Z]+(?![a-z])|[A-Z][a-z]+|[a-z0-9]+", s)
    out = "_".join(p.lower() for p in parts)
    for bad, good in (("task_i_d", "task_id"), ("todo_i_d", "todo_id"), ("i_d", "id"),
                      ("input_u_r_l", "input_url"), ("max_tokens", "max_tokens")):
        out = out.replace(bad, good)
    return out


def owner(n):
    for k in tools:
        if n.startswith(k):
            return k, n[len(k):]
    return None, None


args = defaultdict(list)
for n in prompt:
    k, tail = owner(n)
    if k and tail and tail.endswith("ArgDesc"):
        args[k].append(tail[: -len("ArgDesc")])

# 工具别名：prompt 包里的驼峰 → 模板簇的归属名
ALIAS = {"GenImage": "image_search", "GetMemory": "memory", "WriteMemory": "memory",
         "EditMemory": "memory", "WikiMemory": "memory", "ShareFileCloudDoc": "deliver_result",
         "ShareFile": "share_file", "ProjectInsight": "project_insight", "SwitchMode": "switch_mode"}

md = ["# 灵犀工具契约（符号 × 模板簇 join · 修正版）", "",
      f"- 工具 **{len(tools)}** 个（`prompt.*ToolDesc`）；模板簇 **{len(clusters)}** 个",
      "- join 规则：先按签名 define 把工具锁定到**簇**，再在簇内找「参数 snake_case → `<arg>_arg`」。",
      "- 标 `簇N` 的文案来自该簇；标 `—` 的表示簇内无对应文案（参数说明在别处拼）。", "",
      "| 工具 | 参数 | 簇 | 文案 |", "|---|---|---|---|"]
GENERIC = {"action", "brief", "description", "path", "paths", "content", "query", "type", "name",
           "value", "id", "enabled", "timeout", "timeout_ms", "options", "header", "headers",
           "format", "message", "question", "questions", "code", "text", "reason", "status"}
GLOBAL, GLOBAL_N = {}, {}
for _, _, d in clusters:
    for k, v in d.items():
        GLOBAL_N[k] = GLOBAL_N.get(k, 0) + 1
        if len(v) > len(GLOBAL.get(k, "")):
            GLOBAL[k] = v
GLOBAL_UNIQUE = {k for k, n in GLOBAL_N.items() if n == 1}
hit = miss = 0
for t in sorted(tools):
    alias = ALIAS.get(t, snake(t))
    cno = tool_cluster.get(alias) or tool_cluster.get(t)
    defs = next((d for no, _, d in clusters if no == cno), {})
    for a in sorted(set(args[t])):
        sn = snake(a)
        src = f"簇{cno}" if cno else "簇?"
        body = defs.get(f"{sn}_arg") or defs.get(sn) or ""
        if not body and sn.endswith("_id"):
            body = defs.get(f"{sn[:-3]}_id_arg", "")
        if not body:                     # 簇内没有 → 仅在「非通用名 + 全局唯一」时回退
            cand = f"{sn}_arg" if f"{sn}_arg" in GLOBAL else sn
            if cand in GLOBAL_UNIQUE and sn not in GENERIC:
                body = GLOBAL[cand]
                src = "全局唯一"
            elif cand in GLOBAL:
                src = "歧义"
        if body:
            hit += 1
        else:
            miss += 1
        md.append(f"| `{t}` | `{sn}` | {src} | {(body[:130].replace('|','/') or '—')} |")
md += ["", f"- 簇内命中 **{hit}** / 未命中 **{miss}**", ""]
open(os.path.join(OUT, "PROMPT-tool-contracts.md"), "w", encoding="utf-8").write("\n".join(md))
print(f"contracts: hit={hit} miss={miss}")

# ---------------- 3) 路由表：只留云 API ----------------
route_re = re.compile(r"/(?:api|office|kdocs|kdg|open)/[a-zA-Z0-9_/{}\.\-]{3,90}")
routes = Counter()
for m in route_re.finditer(lx):
    r = m.group(0).rstrip(".,;)'\"")
    if re.search(r"([A-Z]|protoc|etcdserverpb|genproto|googlev|\.go\b|internal/)", r):
        continue
    if r.count("/") > 9:
        continue
    routes[r] += 1
groups = defaultdict(list)
for r, c in routes.items():
    key = "/".join([s for s in r.split("/") if s][:3])
    groups[key].append((r, c))
md = ["# 灵犀云 API 路由清单（静态抽取）", "",
      f"- 源: `{EXE}`（app-server 作为客户端的云端路由）",
      f"- 去重路由 **{len(routes)}** 条 / 前缀组 **{len(groups)}** 个", "",
      "## 前缀组（按条数）", "", "| 前缀 | 条数 |", "|---|---|"]
for k in sorted(groups, key=lambda x: -len(groups[x])):
    md.append(f"| `{k}` | {len(groups[k])} |")
md += ["", "## 全量", ""]
for k in sorted(groups):
    md.append(f"### `{k}`")
    md.append("")
    for r, c in sorted(groups[k]):
        md.append(f"- `{r}`" + (f"  ×{c}" if c > 1 else ""))
    md.append("")
open(os.path.join(OUT, "HTTP-routes.md"), "w", encoding="utf-8").write("\n".join(md))
print(f"routes: {len(routes)} / groups {len(groups)}")
