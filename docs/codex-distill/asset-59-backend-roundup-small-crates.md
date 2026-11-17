# asset-59 · 后端余项收口（小件合集 · 施工级）

> 证据：`codex-rs/{history, message-history, file-search, git-utils, install-context, direct-tool-timing,
> attachment-store, file-watcher, file-system, terminal-detection, responses-api-proxy, features, secrets,
> keyring-store}/src/*`、`utils/{pty, path-uri, stream-parser, output-truncation, fuzzy-match}/src/*`
> （`rust-v0.165.0`）。截断与模糊打分已在 `asset-48` 覆盖，本篇只收"剩下的小件"。
> 目的：让 `backend-coverage-ledger.md` 的"未覆盖"清零——**每个 crate 要么给出机制，要么给出不搬理由**。

---

## 1. 一句话结论

小件里真正有"口径价值"的只有五条：
① **历史批量化有双上限**（message-history：128 行 / 64 KiB）；
② **保留上下文族有三级上限**（history：8 族 / 16 KiB 单条 / 64 KiB 单族）；
③ **git 调用必须有超时并禁用 hooks**（git-utils：5s + `NUL`/`/dev/null`）；
④ **PTY 输出有字节上限**（pty：`DEFAULT_OUTPUT_BYTES_CAP = 1 MiB`）；
⑤ **依赖等待只记"最后完成者"**（direct-tool-timing：`ToolBatchTiming` 用 pending 计数判定批次结束）。
其余小件要么是平台细节（terminal-detection / responses-api-proxy），要么是封装（keyring-store），判定不搬。

---

## 2. 逐 crate 台账

| crate | 非测试 LOC（约） | 机制 / 关键事实（file:line） | 判定 |
|---|---|---|---|
| `history` | 1.1k | 保留上下文族：`MAX_FAMILY_RECORDS = 8`、`MAX_RECORD_BYTES = 16_384`、`MAX_FAMILY_BYTES = 65_536`（`retained_context.rs:24–26`）；心跳内容类型 `"user.heartbeat"`（`heartbeat.rs:10`） | **借**（三级上限） |
| `message-history` | 0.9k | 落盘 `history.jsonl`（`lib.rs:52`）；批量写 `MAX_BATCH_ROWS = 128` / `MAX_BATCH_BYTES = 64 KiB`（`batch.rs:19–20`）；读缓冲 8192（`lib.rs:53`） | **借**（双上限） |
| `git-utils` | 3.4k | `GIT_COMMAND_TIMEOUT = 5s`（`info.rs:41`）；**禁用 hooks**：Windows `NUL` / 其它 `/dev/null`（`info.rs:42`）；未跟踪文件 diff 并发 8（`info.rs:43`）；baseline 提交信息常量（`baseline.rs:17`） | **借**（超时 + 禁 hooks） |
| `utils/pty` | ~8.9k（含测试） | `DEFAULT_OUTPUT_BYTES_CAP = 1 MiB`（`lib.rs:25`）；`ProcessMode`/`SpawnFallback`/`DescriptorPolicy`（`child_command`）；pipe 与 pty 两条 spawn 路径；`ProcessDriver` 供外部传输复用 | **借**（输出上限） |
| `direct-tool-timing` | 0.5k | `ToolBatchTiming{pending, latest}`：**只有 pending 归零才交出批次结果**（`recorder.rs:23–40`）；依赖路径 `CriticalPath`/`WorkKind`，历史不可得时记 `Unknown` | **借**（批次完成判据） |
| `file-system` | 1.2k | 读写分块 1 MiB（`lib.rs:47/49`）；`MAX_WALK_DEPTH = 64`（`lib.rs:51`）；find-up 并发探测 ≤256（`find_up.rs:11`） | **借**（分块 + 深度上限） |
| `file-watcher` | 0.85k | 订阅 + 节流（生产由使用方给间隔，如 skills watcher 10s；测试 50ms）；`WatchRegistration` 生命周期绑定 | **借**（节流口径） |
| `features` | 2.2k | `FEATURES` 静态表（`lib.rs:1057`）+ 遗留别名（`legacy.rs:11`）——特性开关**单一来源** | 旁证（asset-39 已覆盖语义） |
| `secrets` | 0.9k | keyring 服务名 `"codex"`（`lib.rs:23`）；本地密文 `local.age` v1 / `codex_auth.age`（`local.rs:40–42`）；另有 sanitizer（78 行） | 不搬（云身份体系） |
| `keyring-store` | 0.22k | 对 `keyring` crate 的薄封装，单一错误变体（`lib.rs:8–25`） | 不搬 |
| `responses-api-proxy` | 2.6k | 令牌交换：`MAX_RESPONSE_BYTES = 64 KiB`、`MAX_EXCHANGE_REQUEST_BYTES = 64 KiB`、联邦令牌寿命上限 `1h`（`auth.rs:40–44`） | 不搬（云代理） |
| `install-context` | 0.9k | 安装布局：`bin` 目录、code-mode host 可执行名、`codex-package.json` 元数据（`lib.rs:10–16`） | 不搬 |
| `terminal-detection` | 1.2k | 终端能力探测（369 行 + 平台分支） | 不搬 |
| `attachment-store` | 0.28k | 附件落盘的极薄层（212 行） | 不搬（gaea 有 attachment 包） |
| `file-search` | 0.6k | 文件名检索 + snapshot（586 + 73 行）；打分见 `asset-48` 的 fuzzy 语义 | 旁证 |
| `utils/path-uri` | 3.9k | 路径 URI 规范化（`PathUri`/`PathConvention`），被插件清单与执行器复用 | 不搬 |
| `utils/stream-parser` | 1.3k | 增量流解析器骨架 | 不搬 |
| `uds` / `stdio-to-uds` / `tcp-tunnel` / `arg0` / `ansi-escape` / `async-utils` / `config-schema` / `build-info` / `codex-home` / `v8-poc` / `test-binary-support` / `thread-manager-sample` / `collaboration-mode-templates` / `*-noop-macros` | 各 < 1k | 载体/样板/宏/彩蛋 | 不搬 |

---

## 3. 五条可借小改（每条都能单独验收）

### 3.1 历史批量化双上限（~1 小时）

gaea 的会话历史写入若是"逐条 append"，改成**攒批 + 双上限**：条数 128、字节 64 KiB，先到者触发 flush。
收益：写入次数下降，且单批内存可证。

### 3.2 保留上下文三级上限（~1 小时）

若 gaea 有"保留上下文/记忆族"概念（`memory` 包），加三级界：族数 8、单条 16 KiB、单族 64 KiB；
超限时**丢最旧**并计数。

### 3.3 git 调用超时 + 禁 hooks（~30 分钟）

所有 `git` 子进程加 `GIT_COMMAND_TIMEOUT=5s`，并显式禁用 hooks（`-c core.hooksPath=<nul>` 或设
`hooksPath` 到空设备）——防止用户仓库里的 hook 在 gaea 的只读查询里被执行。

### 3.4 PTY/子进程输出上限（~1 小时）

给 `exec` 类工具加 `DEFAULT_OUTPUT_BYTES_CAP = 1 MiB`（参照），超出即截断并提示；
配合 asset-48 的头尾保留口径。

### 3.5 批次完成只记最后完成者（~30 分钟）

若 gaea 有"一批工具调用"的概念，用 `pending` 计数判定批次结束，只保留**最后完成**那条的耗时与依赖路径，
避免"每批都存 N 条时间线"（这是 direct-tool-timing 的核心取舍）。

---

## 4. 判定不搬的统一理由（三类）

1. **云/身份**：`secrets`、`keyring-store`、`responses-api-proxy` —— gaea 单用户本地，无托管身份体系。
2. **平台/载体**：`terminal-detection`、`uds`、`stdio-to-uds`、`tcp-tunnel`、`arg0`、`ansi-escape`、
   `config-schema`、`build-info` —— 依赖终端或进程形态，Wails/Go 侧由别的东西承担。
3. **样板/彩蛋**：`v8-poc`、`test-binary-support`、`thread-manager-sample`、`collaboration-mode-templates`、
   `*-noop-macros` —— 非产品机制。

---

## 5. 验收断言

| # | 断言 |
|---|---|
| X1 | 历史写入：128 条或 64 KiB 任一到达即 flush，且每条只写一次（无重复） |
| X2 | 保留上下文：第 9 个族写入时最旧族被淘汰，单条超 16 KiB 被拒或截断（按实现二选一并固定） |
| X3 | git 查询：耗时 > 5s 被终止并返回明确错误；仓库含恶意 `pre-commit` hook 时**不执行** |
| X4 | PTY 输出超 1 MiB：截断并产生一条可见提示，进程不被误杀 |
| X5 | 批次计时：一批 5 个调用并行，`completed()` 只在第 5 个结束后返回一次，且返回**最后完成者** |

---

## 6. 未读（诚实边界）

- `file-search` 的实现细节（只读了行数与文件构成）、`features/lib.rs`(2192) 的完整特性表、
  `git-utils/{apply.rs, baseline.rs}` 全流程、`utils/pty` 的平台分支（`win.rs`/`unix_io.rs`）。
- 这些都属于"已归类、未逐行"的状态；若将来要落地其中某项，先补读对应文件再写施工卡。
