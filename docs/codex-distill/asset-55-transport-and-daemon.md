# asset-55 · 传输与守护进程（B2 · 施工级 · 判定以"旁证 + 可借实践"为主）

> 证据：`codex-rs/app-server-transport/src/transport/{mod.rs, stdio, unix_socket.rs, websocket.rs, remote_control/*}`、
> `app-server-daemon/src/{lib.rs, update_loop.rs, backend/pid.rs, backend/pid_identity.rs, backend/windows.rs,
> prepare_install.rs, managed_install.rs, release_pruning.rs, settings.rs, remote_control_client.rs}`（`rust-v0.165.0`）。
> gaea 侧：Wails 单进程（`internal/app/app.go`），无独立 daemon —— 只读对照，无对应改造点。
> **本篇判定：实现不搬，借五条常量化纪律。**

---

## 1. 一句话结论

codex 有**四种传输**（stdio / unix socket / WebSocket / off）与**一个独立 daemon**（负责拉起、升级、重启、PID 身份）。
它值得抄的不是"也去做 daemon"，而是**五条被写成常量的纪律**：
有界通道（`CHANNEL_CAPACITY = 128`）、WebSocket 出站队列放大到 `32K`（并**编译期断言**它大于前者）、
单条解压消息 `64 MiB` 上限、控制 socket **`0o600`** 私有权限 + **启动锁** + **恢复文件**、
以及"运维动作全部带超时"（启动 `30s/10s`、停止强杀 `10s`、安装器 stderr 只留 `2 KiB` 且 drain `1s`）。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| 传输枚举与 URL 解析 | `transport/mod.rs` | 84–89 `AppServerTransport::{Stdio, UnixSocket{path}, WebSocket{bind}, Off}`；92–96 解析错误三型；103 支持的 `--listen` 形式 |
| 有界通道 | `transport/mod.rs` | 21–24 `CHANNEL_CAPACITY = 128`（注释给出理由："交互式 CLI 足够，兼顾吞吐与内存"） |
| WebSocket | `transport/websocket.rs` | 54 `WEBSOCKET_OUTBOUND_CHANNEL_CAPACITY = 32 * 1024`；55 编译期 `assert!(> CHANNEL_CAPACITY)`；57 `MAX_DECOMPRESSED_MESSAGE_SIZE = 64 << 20` |
| 私有控制面 | `transport/unix_socket.rs` | 34 `CONTROL_SOCKET_MODE = 0o600`；37 `MAX_UNFRAGMENTED_MESSAGE_BYTES_HEADER`；325 `verified_local_daemon_socket`；370 `AppServerStartupLock`；513 `DaemonSocketFileGuard` |
| 路径与恢复 | `transport/mod.rs` | 56–59 控制 socket 目录/文件名、启动锁名、`loaded-threads.json`；61–81 三个路径构造函数 |
| 守护生命周期 | `daemon/src/lib.rs` | 51 `START_POLL_INTERVAL=50ms`；55/57 `START_TIMEOUT=30s`（Windows 10s）；61–67 pid/lock/settings 文件族 |
| 进程身份 | `daemon/src/backend/pid.rs` | 31 `STOP_POLL_INTERVAL=50ms`；32 `STOP_FORCE_TIMEOUT=10s`；33 `START_TIMEOUT=10s`；34 `STDERR_LOG_TAIL_BYTES=4096` |
| 更新循环 | `daemon/src/update_loop.rs` | 68 `INITIAL_UPDATE_DELAY=5min`；69 `RESTART_RETRY_INTERVAL=50ms`；72 `INSTALLER_STDERR_TAIL_BYTES=2KiB`；73 `INSTALLER_STDERR_DRAIN_TIMEOUT=1s`；75/77 平台安装脚本 URL |
| 远端控制 | `transport/remote_control/{mod.rs(939), enroll.rs(740), client_tracker.rs(868), segment.rs(442), protocol.rs(368), server_api.rs(367), controller.rs(348), websocket.rs(3400)}` | 注册/接线/分段/协议/控制面五层；`REMOTE_CONTROL_DISABLED_ENV_VAR` 与 requirements 层禁用 |

---

## 3. 传输矩阵

| 传输 | 形态 | 适用 | 关键约束 |
|---|---|---|---|
| `stdio://` | 父子进程管道 | 单客户端（IDE 插件） | 连接关闭 = 会话结束（app-server 里有 `single_client_mode` + `stdio_closed` 分支） |
| `unix://[PATH]` | 本地 socket | 本机多客户端 | 目录 `app-server-control/`、文件 `app-server-control.sock`、权限 `0o600` |
| `ws://IP:PORT` | 网络 socket | 远程/多客户端 | 出站队列 `32K`、单条解压 `64 MiB`、不可分片消息有独立头部上限 |
| `off` | 不监听 | 仅进程内 | 显式关闭，不是"默认失败" |

`--listen` URL 解析失败有三型错误（不支持的形式 / unix 路径解析失败 / ws 地址非法），
且错误信息里**直接列出合法形式**——这是"错误即文档"的写法。

---

## 4. 五条常量化纪律（本篇真正要拿的东西）

1. **有界通道 128**：`CHANNEL_CAPACITY` 的注释把"为什么是 128"写清了（交互式 CLI 够用、兼顾内存）。
   —— 有界队列 + 注释理由，比"无界通道 + 事后 OOM 排查"便宜得多。
2. **出站队列分层放大**：WebSocket 出站 `32K`，并用**编译期断言**保证它大于 `CHANNEL_CAPACITY`：
   ```rust
   const WEBSOCKET_OUTBOUND_CHANNEL_CAPACITY: usize = 32 * 1024;
   const _: () = assert!(WEBSOCKET_OUTBOUND_CHANNEL_CAPACITY > CHANNEL_CAPACITY);
   ```
   这条断言防的是"将来有人把入站容量调大而出站没跟上"。
3. **解压后大小上限**：`MAX_DECOMPRESSED_MESSAGE_SIZE = 64 MiB`——压缩炸弹防护必须是**解压后**口径。
4. **私有控制面**：socket 权限 `0o600` + 启动锁文件 + `loaded-threads.json` 恢复快照。
   三者组合起来保证"同一 `CODEX_HOME` 只有一个活着的 daemon，重启能把线程捞回来"。
5. **运维动作全带超时**：启动轮询 `50ms`、启动上限 `30s/10s`、停止强杀 `10s`、安装器 stderr 只留 `2 KiB`
   且 drain `1s`、重启重试 `50ms`——**没有一步是"等它自己好"**。

---

## 5. 守护进程生命周期

```
① 读 settings.json（`STATE_DIR_NAME = "app-server-daemon"`）
② 取操作锁 daemon.lock（带 OPERATION_LOCK_TIMEOUT）
③ 检查 daemon.pid / 旧版 app-server.pid（pid_identity 校验"这个 pid 还是那个进程"）
④ 未运行 → 拉起 backend（Windows 走 backend/windows.rs，POSIX 走 pid 后端），轮询 50ms 至 30s
⑤ 运行中 → 通过控制 socket（0o600）投递控制命令
⑥ 停止：先优雅，10s 后强杀；stderr 只保留最后 4 KiB 用于诊断
⑦ 退出前写 loaded-threads.json（供 daemon_thread_recovery 还原线程）
```

`update_loop` 是独立循环：**初始延迟 5 分钟**（避免启动即升级），安装脚本按平台取
`install.sh` / `install.ps1`，安装器失败时只截 `2 KiB` stderr 且 drain 上限 `1s`（不阻塞关闭），
重启间隔 `50ms`。

---

## 6. 远端控制（remote control）

五层结构：`enroll`（注册）→ `client_tracker`（在线客户端）→ `segment`（消息分段）→
`protocol`（帧格式）→ `server_api`/`controller`（服务端 API 与控制面）。
禁用有两个入口：环境变量 `REMOTE_CONTROL_DISABLED_ENV_VAR` 与**managed requirements 层**
（`RemoteControlDisabledByRequirements`）——即"用户关"和"管理员关"是两条独立通道，后者优先。

---

## 7. gaea 对照

| 维度 | codex | gaea | 结论 |
|---|---|---|---|
| 进程模型 | daemon + 多客户端传输 | Wails 单进程 + 内嵌 WebView（`internal/app/app.go`） | **不做 daemon**（拍板门 3 未开） |
| 通道 | 有界 128 / WS 出站 32K / 解压 64MiB | Wails binding 直调，无显式通道上限 | 可借"有界 + 编译期/测试期断言" |
| 控制面权限 | socket `0o600` + 启动锁 + 恢复文件 | 无 socket | 不适用 |
| 超时纪律 | 启动/停止/安装器全带超时与截断 | `App.Shutdown` 是顺序 teardown（`app.go:686–740`） | **可借**：teardown 每一步加超时与截断日志 |
| 升级 | 独立 update_loop（5min 延迟、平台脚本） | 桌面端更新另有实现（未读） | 只借"启动后延迟 + 安装器输出截断" |

---

## 8. 施工卡（改造成三条，不是做 daemon）

### 刀 1 · 关停路径加超时与截断（约 2 小时）

- `internal/app/app.go:686` 的每一步（pantheon / engine loops / voice / weixin / whisper drain / DB close）
  包一层 `withTimeout(step, d)`，超时只记 warn 继续下一步（**不允许一步卡死整个退出**）。
- 日志截断：任何外部进程 stderr 只保留最后 4 KiB（参照 `STDERR_LOG_TAIL_BYTES=4096`）。

### 刀 2 · 通道与队列上界显式化（约 2 小时）

- 给 `event` 总线与 `jobs` 完成注记队列加显式容量常量 + 满时策略（丢最旧 / 背压），
  并写一条单测断言"容量常量 > 0 且满了不 panic"。

### 刀 3 · 私密运行目录（约 1 小时）

- 若 gaea 将来落任何本地 socket / 锁文件，统一放 `CODEX_HOME` 等价目录（gaea 的 `whisperDataRoot`），
  权限 `0o600`，并留一份"上次未完成"快照文件供恢复。

---

## 9. 验收断言

| # | 断言 |
|---|---|
| D1 | 关停任一步骤挂起：超时后继续下一步，整体退出时间 ≤ 各步超时之和 |
| D2 | 外部进程 stderr 超长：只保留最后 4 KiB，且不因读取阻塞退出 |
| D3 | 事件队列达到容量上限：按策略丢弃并有计数，不 panic、不无界增长 |
| D4 | 私密目录：新建的 socket/锁文件权限为 `0o600`（POSIX） |
| D5 | 恢复快照：进程异常退出后，下次启动能读到 `loaded-threads.json` 等价物并给出"上次未完成"提示 |

---

## 10. 不搬 / 风险 / 未读

- **不搬**：四种传输与 URL 解析、daemon 进程管理、PID 身份校验、update_loop、remote_control 五层。
- **不搬**：WebSocket 分段与 `MAX_UNFRAGMENTED_MESSAGE_BYTES_HEADER`（gaea 无 WS 服务端）。
- **风险**：① 若照搬"关停加超时"要防止把必须完成的事（DB flush）也超时掉——**DB 关闭不可超时跳过**；
  ② 队列上界会改变现有事件顺序表现（须先用快照测试固定现有序列）。
- **未读（诚实边界）**：`transport/remote_control/*` 全文（只读结构）、`daemon/src/{prepare_install, managed_install,
  release_pruning, settings, remote_control_client}.rs` 全文、`backend/windows.rs` 具体实现。
