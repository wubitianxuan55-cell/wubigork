"""D 批产物生成：native-host 协议、sandbox 策略面、kdocs-cli 命令面。
用法: python gen_d_batch.py
产物: re-lingxi/out/NATIVE-HOST.md / SANDBOX.md / KDOCS-cli.md
"""
import os
import re

BASE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service")
KDB = os.path.join(os.environ["LOCALAPPDATA"], r"Programs\lingxi-desktop\resources\kdocs-cli\kdocs-cli.exe")
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")


def load(p):
    return open(p, "rb").read().decode("utf-8", "replace")


def uniq(a):
    seen, out = set(), []
    for x in a:
        if x not in seen:
            seen.add(x)
            out.append(x)
    return out


CUT = re.compile(r"(reflect\.|sync\.|MapIter|runtime\.|strconv\.|json\.|http2|crypto/|godebug|"
                 r"strings\.Reader|internal error|cannot convert|non-empty pointer|using value obtained|"
                 r"unaddressable|flow control|method ABI|file type does not|multiple Read|slice bounds|"
                 r"nil type passed|of non-struct|of non-slice|of non-byte|of nil value)")


def clean(s, limit=140):
    """在相邻 Go 运行时字符串处截断，只留消息本身。"""
    s = re.sub(r"\s+", " ", s).strip()
    m = CUT.search(s)
    if m and m.start() > 8:
        s = s[:m.start()].strip()
    return s[:limit]


def nh_messages(text):
    """按 [NH-...] 标记切出每条消息，遇到下一条标记或 Go 运行时串即止。"""
    out = []
    starts = [m.start() for m in re.finditer(r"\[NH-(?:MSG|CDP|HTTP|HB)\]", text)]
    for i, s in enumerate(starts):
        end = starts[i + 1] if i + 1 < len(starts) else s + 200
        out.append(clean(text[s:end]))
    return uniq(out)




# ---------- native-host ----------
nh = load(os.path.join(BASE, "lingxi-native-host.exe"))
nh_syms = sorted(set(re.findall(r"nativehost\.([A-Za-z_][A-Za-z0-9_]{2,60})", nh)))
nh_msgs = nh_messages(nh)
nh_errs = uniq(re.findall(r"(?:native-host|port-cleanup)[: ][^\x00]{0,110}", nh))

md = ["# lingxi-native-host.exe（静态盘点）", "",
      "- 二进制: `%s`（6.7 MB，**Go**，module `…/cmd/native-host`）" % os.path.join(BASE, "lingxi-native-host.exe"),
      "- 包: `internal/nativehost`；符号 314 个 / 40 包", "",
      "## 生命周期符号", ""]
for s in nh_syms:
    md.append(f"- `nativehost.{s}`")
md += ["", "## 消息面（原文）", ""]
for m in nh_msgs:
    md.append(f"- `{clean(m)}`")
md += ["", "## 错误/清理串（原文）", ""]
for e in nh_errs[:18]:
    md.append(f"- `{clean(e)}`")
md += ["", "## 结构读法", "",
       "1. 起一个 **回环 HTTP 服务**（`ListenWithCleanup`/`StartServer`），把端口写到 **端口文件**（`PortFilePath`/`writePortFile`），",
       "   并用 `netstat -ano | findstr :<port>` + `queryPortPids`/`killProcess` 清掉上一轮残留（`KillStaleNativeHost`）。",
       "2. 浏览器侧「插件」连上来后走 `[NH-MSG]` 消息：`session_start` / `session_end` / `cdp.event` / `cdp.response`；",
       "   未连插件时 CDP 请求被拒：`[NH-CDP] rejected: plugin not connected`。",
       "3. 消息编解码是 `marshalNM`/`UnmarshalNM`/`ReadNM`/`WriteNM`，带 `withBodyLimit` 的体长上限；`Content-Length` 出现 11 次 → HTTP 体分帧。",
       "4. 心跳/防挂死：`StdinIdleMonitor`（stdin 空闲监控）+ `response timeout: id=%d method=%s`。",
       "",
       "→ 对 gaea 的价值：这是「**本地 HTTP + 插件长连 + CDP 转发**」的完整最小实现，比直接 embed 浏览器简单，且天然跨进程。"]
open(os.path.join(OUT, "NATIVE-HOST.md"), "w", encoding="utf-8").write("\n".join(md))

# ---------- sandbox ----------
setup = load(os.path.join(BASE, "lingxi-sandbox-setup.exe"))
runner = load(os.path.join(BASE, "lingxi-sandbox-runner.exe"))
pol = uniq(re.findall(r"(versionoffline_username[^\x00]{0,260})", setup))
md = ["# 灵犀 Windows 沙箱两件（静态盘点）", "",
      "| 件 | 语言 | 规模 | 来源 |", "|---|---|---|---|",
      "| `lingxi-sandbox-setup.exe` | Rust | 14.6 MB | Codex 派生（`codex_execpolicy`、`codex_otel`、`codex_home`） |",
      "| `lingxi-sandbox-runner.exe` | Rust | 8.2 MB | Codex 派生（`codex_protocol::permissions`） |", "",
      "## setup 的 provisioning payload（字段原文）", "", "```text"]
for p in pol[:2]:
    md.append(p)
md += ["```", "",
       "→ 字段集：`version / offline_username / online_username / codex_home / command_cwd / read_roots / write_roots / "
       "deny_read_paths / deny_write_paths / proxy_ports / allow_local_binding / otel / real_user / mode / runtime / refresh_only`；",
       "模式枚举：`full` / `interactive-provision` / `provision-only` / `read-acls-only`。", "",
       "## WFP（网络边界）", "",
       "`FwpmEngineOpen` / `FwpmFilterAdd` / `FwpmTransactionCommit`；provider 名 **“Lingxi Windows Sandbox WFP”**，",
       "描述 **“Persistent WFP provider for Lingxi Windows sandbox filters”**；`LingxiSandboxUsers` 组 + `CapSids`（4 成员）。",
       "→ 网络限制是**按受限 SID 挂 WFP 过滤器**，不是应用层代理。", "",
       "## ACL（文件边界）", "",
       "setup marker 的 SDDL 原文：`D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;<sid>)`（SYSTEM + Administrators + 本沙箱用户）；",
       "deny-read 状态落 `deny_read_acl_state.json`，且有守卫 **“refusing to apply a deny-read ACE to filesystem root”**。", "",
       "## runner 的调用契约（字段原文）", "", "```text",
       "command cwd env permission_profile workspace_roots codex_home real_codex_home cap_sids "
       "network_proxy_restricting_sid timeout_ms tty stdin_open use_private_de…", "```", "",
       "→ 一次沙箱执行 = (command, cwd, env) + 权限档 + workspace roots + cap sids + **网络 SID**。",
       "  `project-roots://codex-project-ro…` 与 `.agents.codex/tmp`、`.codex.sandbox` 是它保留的路径约定。"]
open(os.path.join(OUT, "SANDBOX.md"), "w", encoding="utf-8").write("\n".join(md))

# ---------- kdocs-cli ----------
kd = load(KDB)
usage = uniq([re.sub(r"\s+", " ", t) for t in re.findall(r"Usage:[^\x00]{0,140}", kd)])
paths = sorted(set(re.findall(r"/(?:office|api|v[0-9]+|kdocs|drive)[a-z0-9_/{}\-\.]{4,70}", kd)))
apps = uniq(re.findall(r"<operation type>|<操作类型>|auth set-token|call <tool_name>|feedback \[--reason", kd))
toks = sorted(set(re.findall(r"\b([a-z][a-z0-9]+(?:_[a-z0-9]+){1,3})\b", kd)))
md = ["# kdocs-cli.exe（静态盘点）", "",
      "- 二进制: `%s`（7.3 MB，**Go 1.23.3**，module `kdocs-cli` `(devel)`，仅 3 个依赖 → 纯 std 客户端）" % KDB,
      "- sha256 `24b3e7865d4b1eccbb893868e7ee7384e5ae94a08c4d3c234bc6ab41a74fae28`", "",
      "## CLI 形状（Usage 原文）", ""]
for u in usage[:10]:
    md.append(f"- `{u}`")
md += ["", "## 端点（路径原文）", ""]
for p in paths[:30]:
    md.append(f"- `{p}`")
md += ["", "## 动作/参数词表", "",
       f"- snake_case 候选 **{len(toks)}** 个（动作名与入参名混在同一张字符串表，静态无法完全区分）",
       "- 明确是动作的样本：" + "、".join(f"`{t}`" for t in
                                    ["create_doc", "block_insert", "block_delete", "block_query", "block_update",
                                     "add_fields", "add_calculated_field", "add_chart", "ap_file", "create_records"][:10]),
       "", "## 与 gaea 的关系", "",
       "灵犀把云文档能力做成 **独立 CLI（`kdocs-cli <service> <action> [json_params]`）**，",
       "由 `KDOCS_CLI_PATH` + `WPS_SID` 驱动，技能脚本只做参数拼装。gaea 若不接金山云，",
       "这一层整体不移植；但「云能力=独立 CLI、技能只拼参数」这个**分层**值得照搬（比把云 SDK 编进主进程轻）。"]
open(os.path.join(OUT, "KDOCS-cli.md"), "w", encoding="utf-8").write("\n".join(md))

print("wrote NATIVE-HOST.md / SANDBOX.md / KDOCS-cli.md")
print("nh symbols:", len(nh_syms), "nh msgs:", len(nh_msgs))
print("kdocs paths:", len(paths), "tokens:", len(toks))
