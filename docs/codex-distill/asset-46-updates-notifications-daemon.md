# 资产 46 · 更新与运行面：命令随安装方式变、版本不可比就不比、恢复只做一次

> 证据根（只读）：`update_action.rs`（7.5KB）、`update_versions.rs`（6.4KB）、`updates.rs`、`updates_cache.rs`、
> `update_prompt.rs`（12.9KB）、`notifications/{mod,osc9,bel}.rs`、`daemon_{startup,recovery,telemetry}.rs`、
> `startup_error.rs`、`session_archive_commands.rs`、`session_queue_commands.rs`。
> 对照根（只读）：`frontend/src/pages/SettingsPage.tsx`（更新区）、`pages/modelcenter/*`（引擎版本检查）、
> `scripts/sync-version.ps1`（三处版本同步）、`build/`（打包与冒烟）。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、更新动作是**八选一**，由安装方式派生

`update_action.rs:8-27`：

```rust
enum UpdateAction {
  Daemon(DaemonUpdateSource),   // 恢复终端后替换本地 daemon
  NpmGlobalLatest, BunGlobalLatest, VitePlusGlobalLatest, PnpmGlobalLatest,
  BrewUpgrade, StandaloneUnix, StandaloneWindows,
}
```

`from_install_context(context)`（`:31-40`）：按 `InstallMethod`（Npm/Bun/VitePlus/Pnpm/Brew/Standalone{platform}）映射，
未知方式返回 `None`——**不知道就"不提供更新动作"，而不是猜一条命令**。

每条变体在注释里直接写着要执行的命令（`npm install -g @openai/codex@latest`、`brew upgrade codex`、
`CODEX_NON_INTERACTIVE=1` 的 standalone 脚本…），文案与行为同源。

## 二、版本比较：不可比就返回 `None`

`update_versions.rs:1-41`：

| 函数 | 行为 |
|---|---|
| `is_newer(latest, current)` | 两边都能解析才返回 `Some(l > c)`，任一失败返回 **`None`**（不是 `false`） |
| `extract_version_from_latest_tag(tag)` | 要求 `rust-v` 前缀，否则**报错**（不静默取后半段） |
| `is_source_build_version(v)` | 只有 `(0,0,0)` 才算源码构建 |
| `ServerVersionNoticeKind{Older, Different}` | 两种"不一致"分开 |
| `server_version_notice_kind(client, server)` | 稳定版比**发布优先级**；预发布版只在**同一 release line** 内排序；本地构建/跨 line 比**身份**（不等即 `Different`） |

`updates_cache.rs`（1.7KB）：检查结果有缓存，避免每次启动都打网络。

> 这套口径的价值：它在回答"**我该不该提示用户升级**"之前，先回答"**我能不能比较这两个版本**"。
> 不能比的时候给 `None`（不提示），比的时候区分"更旧"与"不同"（后者不一定是落后，可能是本地构建）。

## 三、通知后端按终端能力选，且能回报"我是哪种"

`notifications/mod.rs:13-40`：`DesktopNotificationBackend{Osc9, Bel}`；
`for_method(Auto | Osc9 | Bel)` 在 `Auto` 时用 `supports_osc9(terminal_info())` 决定，否则按用户指定；
`method()` 反向把当前实现报回去（设置页显示真实生效项，不是用户选项的副本）。

## 四、daemon 恢复的三条纪律写在文件头

`daemon_recovery.rs:1-2`：

> 「显式地从本地 daemon 不兼容中恢复。**重启是受管操作，需要一次新鲜确认，之后只做一次兼容性检查，绝不循环。**」

`check(startup, target, config, cli_kv_overrides, managed_daemon)`（`:30-…`）先跑兼容性探测（`daemon_startup::compatibility_warning`），
有问题才进恢复流程；`managed_daemon` 是参数——**只有受管进程才允许重启**。

`daemon_startup.rs`（10KB）+ `daemon_telemetry.rs`（2.5KB）：启动与遥测分开，遥测不参与启动判定。
`startup_error.rs:9` `LocalStateDbStartupError`：本地库启动失败的**独立错误类型**（而不是一个 `String`）。

## 五、会话归档/队列是命令，不是 UI 特例

`session_archive_commands.rs`（14.9KB）+ `session_queue_commands.rs`（5.4KB，含 JSON-RPC 错误码常量
`JSONRPC_INVALID_REQUEST=-32600` / `METHOD_NOT_FOUND=-32601` / `THREAD_QUEUE_ADD_METHOD="thread/queue/add"`）：
归档与队列走**命令通道**，UI 只是调用方；错误按协议码分类返回。

## 六、gaea 现状核对

| gaea | 现状 | 差在哪 |
|---|---|---|
| `pages/modelcenter/*` + 引擎更新检查（30min 缓存） | 已有"检查更新 + 缓存" | 版本比较口径未区分"更旧/不同/不可比"；无"更新动作随安装方式变"（Windows 只有一种装法，短期无必要） |
| `scripts/sync-version.ps1` + 三处版本（app_info/wails.json/versioninfo.rc） | 版本源有同步脚本 | 缺"客户端 vs 服务端版本不一致"的提示分类（gaea 无服务端） |
| 全仓 | 无桌面通知后端（无 toast/系统通知） | 长任务完成/审批待办目前只在应用内提示 |
| 全仓 | 无 daemon/受管进程恢复 | Wails 单进程，无此问题；但**"重启只做一次检查、不循环"**这条纪律对"引擎重启"仍适用 |
| 全仓 | 无队列命令通道（有 steer/queue 概念） | `session_queue_commands` 的协议化错误码可借鉴 |

## 七、三档判定

**档一（照搬语义）**

1. **不可比返回 `None`**：任何版本/口径比较失败时"不提示"，而不是当作"需要更新"。
2. **区分更旧与不同**：`Older` / `Different` 两态（本地构建 vs 落后）。
3. **更新动作由安装上下文派生**，未知方式不猜。
4. **恢复只做一次检查，绝不循环**。

**档二（改造搬）**

5. **通知后端能力探测 + 回报生效项**：将来加桌面通知时，设置页显示"实际生效：系统通知/应用内"，而非选项副本。
6. **协议化错误码**：队列/归档类操作返回结构化 code，UI 按 code 给文案（现在多为 message 字符串）。

**档三（不搬）**

7. `Osc9`/`Bel` 终端通知协议、`Daemon` 更新源、npm/bun/pnpm/brew 等安装方式分支；
8. `daemon_telemetry` 与 `startup_error` 的本地库错误类型（gaea 有自己的启动错误处理）。

## 八、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F46-A 版本比较口径** | `lib/versionCompare.ts` | `compareVersions(a,b) -> -1\|0\|1\|null` + `noticeKind(client, server)` | 单测：畸形版本 ⇒ null；`0.0.0` ⇒ Different；同 line 预发布按序 |
| **F46-B 恢复不循环** | 引擎重启路径 | 一次检查 + 一次重启 + 结果落日志 | 单测：失败后不再自动重试（计数器=1） |
| **F46-C 通知生效项** | 设置页（若加通知） | 能力探测 + 显示实际后端 | 单测：不支持 ⇒ 回落"应用内"并显示 |

**开关**：A 纯函数；B 改现有重启路径（低风险但需日志证据）；C 视产品是否做通知。

## 九、风险

1. **`sync-version.ps1` 已是三处同步的单一入口**：加"版本比较"不要另建第二份版本读取逻辑。
2. **"不可比就不提示"** 可能掩盖真问题：需要在日志里留一条 `version_compare_failed`，否则永远静默。
3. **重启幂等**：gaea 的引擎重启若被用户连点，需 in-flight 去重（与 `gitStatusStore` 同款）。

## 十、证据自查

- 更新动作八选一：`update_action.rs:8-40`。
- 版本口径：`update_versions.rs:1-41`；缓存：`updates_cache.rs`。
- 通知后端：`notifications/mod.rs:13-40`。
- 恢复纪律：`daemon_recovery.rs:1-40`；启动/遥测分层：`daemon_startup.rs`、`daemon_telemetry.rs`、`startup_error.rs:9`。
- 命令通道：`session_queue_commands.rs:23-25`。
