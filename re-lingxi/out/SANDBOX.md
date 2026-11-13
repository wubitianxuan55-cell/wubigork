# 灵犀 Windows 沙箱两件（静态盘点）

| 件 | 语言 | 规模 | 来源 |
|---|---|---|---|
| `lingxi-sandbox-setup.exe` | Rust | 14.6 MB | Codex 派生（`codex_execpolicy`、`codex_otel`、`codex_home`） |
| `lingxi-sandbox-runner.exe` | Rust | 8.2 MB | Codex 派生（`codex_protocol::permissions`） |

## setup 的 provisioning payload（字段原文）

```text
versionoffline_usernameonline_usernamecodex_homecommand_cwdread_rootswrite_rootsdeny_read_pathsdeny_write_pathsproxy_portsallow_local_bindingotelreal_usermoderuntimerefresh_onlyfullinteractive-provisionprovision-onlyread-acls-only
versionoffline_usernameonline_usernamecreated_atproxy_portsallow_local_bindingread_rootswrite_rootsofflineonlineusernamepassword$D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;�)
```

→ 字段集：`version / offline_username / online_username / codex_home / command_cwd / read_roots / write_roots / deny_read_paths / deny_write_paths / proxy_ports / allow_local_binding / otel / real_user / mode / runtime / refresh_only`；
模式枚举：`full` / `interactive-provision` / `provision-only` / `read-acls-only`。

## WFP（网络边界）

`FwpmEngineOpen` / `FwpmFilterAdd` / `FwpmTransactionCommit`；provider 名 **“Lingxi Windows Sandbox WFP”**，
描述 **“Persistent WFP provider for Lingxi Windows sandbox filters”**；`LingxiSandboxUsers` 组 + `CapSids`（4 成员）。
→ 网络限制是**按受限 SID 挂 WFP 过滤器**，不是应用层代理。

## ACL（文件边界）

setup marker 的 SDDL 原文：`D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;<sid>)`（SYSTEM + Administrators + 本沙箱用户）；
deny-read 状态落 `deny_read_acl_state.json`，且有守卫 **“refusing to apply a deny-read ACE to filesystem root”**。

## runner 的调用契约（字段原文）

```text
command cwd env permission_profile workspace_roots codex_home real_codex_home cap_sids network_proxy_restricting_sid timeout_ms tty stdin_open use_private_de…
```

→ 一次沙箱执行 = (command, cwd, env) + 权限档 + workspace roots + cap sids + **网络 SID**。
  `project-roots://codex-project-ro…` 与 `.agents.codex/tmp`、`.codex.sandbox` 是它保留的路径约定。