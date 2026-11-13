"""F 批：灵犀线协议词汇（lingxi.exe 的 Go json tag）+ Electron IPC 通道表（主进程 bundle）。
只读灵犀产物，不进入 gaea 代码。
用法: python extract_wire.py
"""
import os
import re
import sys
from collections import Counter, defaultdict

EXE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service\lingxi.exe")
MAIN = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", "x", "out", "main", "index.js")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")

txt = open(EXE, "rb").read().decode("utf-8", "replace")

# ---------- 1) json tag 词汇表 ----------
tags = re.findall(r'json:"([A-Za-z0-9_]+)(?:,[a-z,]*)?', txt)
cnt = Counter(tags)
print(f"json tag 命中 {len(tags)} 次，去重 {len(cnt)} 个")

# ---------- 2) Go 类型名（结构体）----------
types = Counter(re.findall(r"\*?([a-z0-9_]+)\.([A-Z][A-Za-z0-9_]{3,50})\b", txt))
pkg_types = defaultdict(set)
for (pkg, name), c in types.items():
    if pkg in {"core", "agent", "session", "run", "tool", "skill", "config", "auth", "sandbox",
               "connector", "prompt", "store", "server", "protocol", "rpc", "mcp", "dw", "timer"}:
        pkg_types[pkg].add(name)
print("类型样例包:", {k: len(v) for k, v in sorted(pkg_types.items())})

# ---------- 3) 开关/字段语义分组（按 tag 高频）----------
md = ["# 灵犀线协议词汇表（静态抽取）", "",
      f"- 源: `{EXE}`（sha256 `0b789ef5…`）",
      f"- `json:\"…\"` 命中 {len(tags)} 次 / 去重 **{len(cnt)}** 个字段名",
      "- 口径：Go 结构体 tag 是**线协议字段名的唯一权威**（schema 由 `jsonschema-go` 运行时按 tag 生成，不是内嵌 JSON）", ""]

md += ["## 高频字段（出现 ≥6 次）", "", "| 字段 | 次数 |", "|---|---|"]
for k, v in cnt.most_common(80):
    if v >= 6:
        md.append(f"| `{k}` | {v} |")

md += ["", "## 按语义分组（人工归类，取各组的字段）", ""]
GROUPS = {
    "会话/运行": ["session_id", "run_id", "agent_id", "group_id", "message_id", "request_id",
                 "parent_session_id", "title", "status", "state", "mode", "pinned"],
    "工具调用": ["tool", "tool_name", "tool_call_id", "arguments", "input", "output", "result",
                "error", "error_code", "brief", "justification", "sandbox_permissions"],
    "鉴权": ["auth_mode", "wps_sid", "token", "refresh_token", "expires_at", "device_id",
            "account_id", "user_id", "credits", "quota"],
    "沙箱": ["read_roots", "write_roots", "deny_read_paths", "deny_write_paths", "proxy_ports",
            "allow_local_binding", "permission_profile", "cap_sids", "sandbox_data_dir"],
    "技能": ["skill", "skills", "skill_dir", "skill_name", "enabled", "location", "installed",
            "official", "manifest"],
    "连接器/MCP": ["connector", "connector_id", "server", "endpoint", "headers", "command",
                  "transport", "stdio", "streamable_http", "mcp"],
    "内容块": ["type", "text", "content", "blocks", "delta", "finish_reason", "usage",
              "input_tokens", "output_tokens"],
}
seen = set()
for g, keys in GROUPS.items():
    hit = [k for k in keys if k in cnt]
    seen |= set(hit)
    md.append(f"### {g}")
    md.append("")
    md.append("、".join(f"`{k}`({cnt[k]})" for k in hit) or "—")
    md.append("")

md += ["## 全量字段名（去重，按字母序）", "", "```text"]
for k in sorted(cnt):
    md.append(f"{k}  x{cnt[k]}")
md += ["```", "",
       "## 使用方式", "",
       "- 想核「某个 RPC 的参数字段是否叫 X」→ 直接在本表查；表里没有 = 该字段不在二进制里（大概率不存在）。",
       "- 想看某个结构体的完整字段集 → 需要 Ghidra 级反编译（tag 与 struct 的绑定关系不在字符串里）。"]

open(os.path.join(OUT, "WIRE-json-tags.md"), "w", encoding="utf-8").write("\n".join(md))
print("wrote WIRE-json-tags.md")

# ---------- 4) Electron IPC 通道 ----------
if os.path.exists(MAIN):
    js = open(MAIN, encoding="utf-8", errors="replace").read()
    ch = Counter(re.findall(r"""["']([a-z][A-Za-z0-9]*:[A-Za-z0-9_]+)["']""", js))
    print(f"IPC 通道候选 {len(ch)} 个")
    md = ["# 灵犀 Electron IPC 通道表（静态抽取）", "",
          f"- 源: `{MAIN}`（主进程 bundle）",
          f"- `命名空间:方法` 形态候选 **{len(ch)}** 个",
          "- 口径：只看字面量，动态拼接的通道不在表内", "", "| 通道 | 引用次数 |", "|---|---|"]
    for k, v in ch.most_common():
        md.append(f"| `{k}` | {v} |")
    open(os.path.join(OUT, "IPC-channels.md"), "w", encoding="utf-8").write("\n".join(md))
    print("wrote IPC-channels.md")
    print("top:", ", ".join(f"{k}({v})" for k, v in ch.most_common(12)))
