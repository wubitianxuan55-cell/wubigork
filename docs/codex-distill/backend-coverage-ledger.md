# codex 0.165.0 后端**全量覆盖台账**（122 个 crate → 覆盖 / 部分 / 未覆盖 / 不搬）

> 目的：把 `codex-rs` 的**每一个 crate** 都钉到"哪一篇覆盖、判定是什么、剩下多少没读"，
> 让"后端已逆完"这句话可核对——与 `frontend-coverage-ledger.md` 同一口径，只是对象从
> `tui/src` 的 160 个文件换成后端的 122 个 crate。
>
> 证据根：`C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs`（开源树，与 `rust-v0.165.0` 同版本）；
> 二进制侧旁证见 `C:\AI\re\codex-0.165.0\evidence\`（`crate-map.txt` / `strings.txt` / `rpc-*.txt`）与
> `docs\hotspot-01…11`。
>
> 本文只做**调研与判定**，不含任何 gaea 代码改动。

---

## 0. 三条口径（先说清，否则数字会漂）

1. **LOC 一律按非测试估读**：脚本统计 `*.rs` 行数时把 `tests/`、`*_tests.rs`、`#[cfg(test)]` 尾巴一起算了，
   所以表中给的是**含测试的文件总行数**，并在有实测非测试值时单列。只看**相对大小**，不当精确值用。
2. **"覆盖"= 有独立资产文档且读到机制级**（能回答"为什么这么做、常量是多少、失败怎么收敛"）；
   **"部分"**= 文档只碰到协议/决策层，实现层没读；**"未覆盖"**= 没有文档锚点；
   **"不搬"**= 已判定不移植（附理由分类，见 §5）。
3. 判定依据是"**gaea 用得上**"而不是"代码多不多"：`core` 44 万行里真正值得搬的只有几个子系统，
   `windows-sandbox-rs` 2.8 万行基本不可搬。

---

## 1. 总览：按功能域分组

| 功能域 | crate 数 | 含测试 LOC | 判定分布（覆盖/部分/未覆盖/不搬） | 结论 |
|---|---|---|---|---|
| 会话编排与服务端（app-server 家族） | 7 | 261k | 0 / 2 / 5 / 0 | **最大空白**：只抽过协议枚举，实现层没读 |
| 智能体内核（core 及其子模块） | 1 | 442k | 0 / 1 / 0 / 0 | 部分：回合/工具/配置/agent 四个子系统未读 |
| 存储与状态（thread-store / state / rollout / rollout-trace / history） | 6 | 98k | 1 / 1 / 4 / 0 | rollout 已覆盖；thread-store、state/runtime 全空白 |
| 工具与执行（tools / exec / exec-server / shell-command / apply-patch / execpolicy） | 7 | 107k | 3 / 4 / 0 / 0 | 协议层已覆盖，执行服务实现层空白 |
| 扩展与插件（ext / core-plugins / plugin / skills / memories / context-fragments / message-board…） | 11 | 138k | 6 / 2 / 3 / 0 | 单点覆盖，**扩展框架与市场层空白** |
| MCP（rmcp-client / codex-mcp） | 2 | 66k | 0 / 1 / 1 / 0 | 传输矩阵与工具缓存空白 |
| 模型与网络（codex-api / http-client / model-provider* / network-proxy / responses-api-proxy / websocket-*） | 9 | 79k | 1 / 2 / 6 / 0 | network-proxy 已覆盖，客户端池与 SSE 空白 |
| 观测（analytics / otel / rollout-trace / diagnostics / direct-tool-timing） | 5 | 41k | 0 / 1 / 4 / 0 | 全空白（前端用量页有篇，后端上报没读） |
| 认证与云（login / chatgpt / aws-auth / workload-identity / cloud-* / backend-client…） | 14 | 54k | 0 / 1 / 13 / 0 | 判定不搬（gaea 自有认证与云侧） |
| 沙箱与 OS（sandboxing / linux-sandbox / windows-sandbox-* / bwrap / mxc-sandbox / process-hardening） | 7 | 59k | 0 / 1 / 0 / 6 | 只借数据模型（P2-5），实现不搬 |
| 前端 tui | 1 | 434k | 20 篇 + 台账 | 已收口（见 `frontend-coverage-ledger.md`） |
| 其它小件（utils 子 crate / file-* / git-utils / uds / ansi-escape / arg0…） | ≈53 | ≈90k | 0 / 3 / ≈20 / ≈30 | 按需取用，单项 < 8k |

> 说明：`tui` 已在上一批收口，本台账**不重复**；它下面的 `mermaid`、`ansi-escape` 等按"前端附属"计。

---

## 2. 逐 crate 台账（含测试 LOC / 现有锚点 / 判定）

> `锚点` 列指向本目录已有文档；`—` 表示无锚点。行序按功能域，同域内按 LOC 降序。

### A. 会话编排与服务端（最大空白）

| crate | LOC | 锚点 | 判定 | 备注（本轮实测） |
|---|---|---|---|---|
| `app-server` | 190,866 | asset-47（准入/drain/状态广播） | **部分** | 已覆盖：`turn_admission`、`connection_rpc_gate`、`connection_cleanup`、`thread_status`、`request_serialization`、关闭判定（`lib.rs`）。仍空白：`request_processors/`（thread 6.5k / turn 1.6k / plugins 2.4k / config 1.3k / mcp 0.8k …）、`message_processor` 网关路由、`bespoke_event_handling`(3.9k)、`config_manager_service`(862)、`skills_watcher`、`mcp_refresh`、`turn_cost_worker`(511) |
| `app-server-protocol` | 33,974 | 枚举旁证（`rpc-*.txt`） | **部分** | 180 `ClientRequest`/11 `ServerRequest` 已抽；**分层与版本策略、序列化契约、通知族语义**未写 |
| `app-server-transport` | 17,850 | **asset-55**（传输矩阵 + 五条纪律） | **覆盖** | 四传输（stdio/unix/ws/off）、`ConnectionAuth` 所有权、有界通道 128、WS 出站 32K + 编译期断言、解压 64MiB、控制 socket `0o600` + 启动锁 + 恢复文件 |
| `app-server-daemon` | 11,287 | **asset-55** | **覆盖** | 守护生命周期（启动 50ms 轮询 / 30s(win 10s) 上限 / 停止强杀 10s）、update_loop（5min 初始延迟 / 安装器 stderr 2KiB）、pid 身份族、`loaded-threads.json` |
| `app-server-client` | 3,713 | — | **未覆盖** | 客户端侧调用封装，可作"协议使用者"旁证 |
| `app-server-test-client` | 3,620 | — | 旁证 | 测试驱动，不搬 |
| `app-server-protocol-noop-macros` | 17 | — | 不搬 | 宏样板 |

### B. 智能体内核（`core`，442k）

| 子模块 | LOC | 锚点 | 判定 | 备注 |
|---|---|---|---|---|
| `core/src/tools` | 46,588 | **asset-50**（注册/派发/审批插入点/并行） | **覆盖** | `registry`（重名与 `exposure` 静默丢弃）、`spec_plan`（`exposure.is_direct()` 决定可见性、`Hidden` 强制不可并行）、`orchestrator`（第一段即审批 + 沙箱拒绝后**二次审批**）、`parallel`（自声明并行 + 读写闸门 + `FuturesOrdered`）、`executed_tool_calls`（best-effort 归因） |
| `core/src/config` | 30,658 | **asset-61**（默认值口径/编辑保注释/托管层） | **覆盖** | 默认值常量组（200 项 / 10MiB / 65% / 30s / 三处提示词各 2000B）、`toml_edit` 保注释改写、`managed_features` 托管优先、`runtime_refresh` 全生效或不生效 |
| `core/src/agent` | 14,270 | **asset-61** | **覆盖** | 角色兜底名、类型不可用明确报错、静态名称词表、三条资源上限（关停失败 64 / 根上下文 3 / 授权消息 16）、`spawn` 派生链 |
| `core/src/session` | 43,743 | asset-09/10/12 + **asset-49（回合状态机）** | **部分** | 已覆盖：`turn.rs`（step 0 归属 + 压缩失败保输入）、`step_activation.rs`（快照 + 受限更新 + 十项禁止变化）、`input_queue.rs`（回合内/邮箱双通道 + 先订阅再检查 + 插话取消）、`token_budget.rs`（claim 一次性 + 优先级）、`rollout_reconstruction.rs`（最新可用压缩点 + 三态清除段）。仍空白：`session.rs`(1992)、`mcp.rs`(1365)、`handlers.rs`(786)、`environment.rs`、`config_refresh.rs`、`async_compaction.rs` 细节 |
| `core/src/config` | 30,658 | asset-02（提及） | **部分** | 分层与 `requirements_layers`、`edit`(911)、loader 未读；与 `config` crate 重叠 |
| `core/src/agent` | 14,270 | asset-06 | **部分** | `control/spawn.rs`(1535) 子智能体派生细节未读 |
| `core/src/context` + `context_manager` | 16,343 | asset-02/08/12 | **覆盖** | 章节化/快照差分已施工级 |
| `core/src/guardian` | 9,936 | asset-07 | **覆盖** | |
| `core/src/unified_exec` | 6,599 | asset-10 | **部分** | `process_manager`(1855) 进程生命周期未单列 |
| `core/src/tasks` / `thread_manager` / `codex_thread` / `environment_selection` | ≈5.1k | — | **未覆盖** | 环境选择（本地/工作树/远程）与线程管理器是编排入口 |
| `core/src/state` / `plugins` / `compact` / `mcp_tool_call` / `exec_policy` / `hook_runtime` | ≈4.6k | asset-09/13/05 | **覆盖**（多篇分摊） | |
| `core/tests/suite` | 189,985 | — | 不搬 | 但**是最好的行为说明书**：`approvals`、`unified_exec`、`compact`、`guardian_review`、`turn_input_submission` 等可直接读断言 |

### C. 存储与状态

| crate | LOC | 锚点 | 判定 | 备注 |
|---|---|---|---|---|
| `thread-store` | 34,827 | **asset-52** | **覆盖** | `src/local` 30k：读/列表/搜索/分段分页/rollout 迁移/回滚三件套/归档三态/行解析/发布/元数据更新；权威源与"允许滞后禁止超前"一致性规则 |
| `state` | 27,045 | **asset-52** | **覆盖** | `src/runtime` 19k：threads/memories/goals/logs/reclamation/queued_items/backfill/recovery/section_order；页回收常量组 + 陈旧锁接管判据 |
| `rollout` | 17,810 | asset-22 | **覆盖** | 压缩归档已施工级 |
| `rollout-trace` | 13,281 | — | **未覆盖** | trace 落盘与回放，与 otel 互补 |
| `history` | 3,920 | — | **未覆盖** | 命令历史（持久化+去重） |
| `message-history` | 1,270 | — | **未覆盖** | 消息级历史索引 |
| `attachment-store` | 283 | — | 未覆盖（小） | 附件落盘 |

### D. 工具与执行

| crate | LOC | 锚点 | 判定 | 备注 |
|---|---|---|---|---|
| `exec-server` | 66,045 | asset-21（协议层）+ **asset-56**（实现纪律） | **覆盖** | 协议 + 五条实现纪律（入站分档 8KiB/2×、pong 看门狗 60s、序号不回绕、帧上限 256KiB 先于解析、退避 1s→30s+120s 稳定期）；`server/`、`remote/` 细节未读 |
| `tools` | 7,665 | asset-11（部分） | **部分** | `src/json_schema`(533) 与 spec 生成 |
| `apply-patch` | 5,097 | hotspot-05 | **覆盖** | |
| `exec` | 12,270 | asset-10（部分） | **部分** | 事件协议与进程包装 |
| `shell-command` | 9,920 | asset-10 | **覆盖** | |
| `execpolicy` | 2,698 | asset-13 | **覆盖** | |
| `exec-server-protocol` | 2,865 | asset-21 | **覆盖** | |
| `exec-server-cli` | 2,877 | asset-21/56（域判定不搬） | **未覆盖** | 0.170 新增（hotspot-13） |

### E. 扩展与插件

| crate | LOC | 锚点 | 判定 | 备注 |
|---|---|---|---|---|
| `ext`（18 个子 crate） | 73,324 | 多篇分摊 | **部分** | 已覆盖：`skills`(19.7k)/`guardian-v2`(17.6k)/`memories`(2.4k)/`mcp`(4.1k 部分)。空白：`extension-api`(2.3k，**扩展 trait/能力/生命周期契约**)、`goal`(3.2k)、`queue`(0.6k)、`items`(0.5k)、`web-search`(0.8k)、`history-notes`(1.1k)、`image-generation`(1.6k)、`git-attribution`(0.4k)、`connectors`、`guardian-v3`、`multi-env-copy-file` |
| `core-plugins` | 47,012 | **asset-57**（市场/来源策略/清单/升级） | **覆盖** | `remote`、`marketplace_add`/`upgrade`/`policy`、`manifest`（先语法后解析）、`command_migration`、`startup_sync`；容量：featured 3h、远程同步 120s、已加载 8 条 |
| `plugin` | 892 | asset-17 | **覆盖** | 插件元数据 |
| `plugin-host` | 396 | asset-17/57（域判定不搬） | **未覆盖** | 0.170 新增（hotspot-13） |
| `http-headers-helper` | 191 | — | **未覆盖** | 0.170 新增小件（hotspot-13） |
| `core-plugin-common` | 207 | — | 旁证 | 共享类型 |
| `skills` | 2,488 | asset-01/15 | **覆盖** | |
| `memories` | 5,962 | asset-03/14/24 | **覆盖** | |
| `context-fragments` | 625 | asset-01/12 | **覆盖** | |
| `agent-roles` | 547 | asset-06 | **覆盖** | |
| `agent-graph-store` | 436 | asset-06（判定不搬） | 不搬 | |
| `message-board` | 5,377 | asset-04 | **覆盖** | |
| `agent-message-board-client` | 912 | asset-04 | **覆盖** | |
| `agent-message-board-host` | 261 | — | **未覆盖** | 0.170 新增：消息板宿主落地（hotspot-13） |
| `message-board-mcp` | 710 | — | **未覆盖** | 0.170 新增：消息板 over MCP（hotspot-13） |
| `collaboration-mode-templates` | 2 | — | 不搬 | |
| `features` | 3,758 | asset-39（提及） | 旁证 | 特性开关矩阵 |

### F. MCP

| crate | LOC | 锚点 | 判定 | 备注 |
|---|---|---|---|---|
| `rmcp-client` | 38,473 | **asset-58**（传输矩阵/鉴权上限/elicitation） | **覆盖** | 四传输 + 适配器组合、OAuth 发现 5s、EMA 16 服务器/10 跳/30s、事件通知 2MiB、stdio 单行 8MiB、提前取消 1024 |
| `codex-mcp` | 27,137 | **asset-58** | **覆盖** | 服务端：连接管理 + 工具目录缓存（32 条 / 30min / 身份指纹键）+ 启动期 deadline |

### G. 模型与网络

| crate | LOC | 锚点 | 判定 | 备注 |
|---|---|---|---|---|
| `network-proxy` | 31,637 | asset-19 | **覆盖** | 决策层与预算租约 |
| `codex-api` | 16,726 | asset-25/26（协议语义）+ **asset-53**（SSE 头部吸收/限流快照） | **覆盖** | 端点/SSE 元数据（openai-model、x-request-id、x-codex-turn-state、x-reasoning-included、models-etag）、限流多快照解析 |
| `http-client` | 11,553 | **asset-53** | **覆盖** | 路由感知连接池（≤16 路由）、per-hop 重定向决策（≤10 跳）、出网策略 permit 四态、TLS 后端回退一次、per-OS 代理 |
| `model-provider` / `model-provider-info` / `models-manager` | 11,238 | **asset-53** | **覆盖** | 供给抽象 + 鉴权恢复两态（`NotConfigured/Recovered`）+ 目录缓存（`models_cache.json`，TTL 300s）+ 平台 provider（Bedrock/GCP） |
| `responses-api-proxy` | 2,594 | — | 未覆盖（小） | Responses 代理层 |
| `websocket-client` / `websocket-auth` | 2,359 | — | 未覆盖（小） | WS 鉴权握手 |
| `ollama` / `lmstudio` | 1,419 | — | 不搬 | 本地模型接入，gaea 走自有 provider |
| `tcp-tunnel` | 1,350 | — | 不搬 | 云端调试隧道 |
| `uds` / `stdio-to-uds` | 1,187 | — | 不搬 | 进程间载体 |

### H. 观测

| crate | LOC | 锚点 | 判定 | 备注 |
|---|---|---|---|---|
| `analytics` | 16,902 | **asset-54**（事实流/归约/上界） | **覆盖** | 队列 256 / 10s / flush 25s / 去重 4096；reducer 五类上限（256/16/8/64B/256） |
| `otel` | 9,730 | **asset-54** | **覆盖** | meter `codex`、ms 边界、缓冲 256/1KiB/16 tags、enable/disable/**suspend** 三态、逐指标同意 |
| `rollout-trace` | 13,281 | **asset-54** | **覆盖** | bundle 四件套（manifest/trace.jsonl/payloads/state.json，三处 schema v2）、复用 protocol 事件、禁用零成本 |
| `diagnostics` | 238 | — | 未覆盖（小） | |
| `direct-tool-timing` | 494 | — | 未覆盖（小） | |

### I. 认证与云（判定不搬，只留语义）

| crate | LOC | 判定 | 理由 |
|---|---|---|---|
| `login` | 21,656 | 不搬（借语义） | gaea 自有认证；只借 token 过期/刷新/失效三态与 keyring 兜底 |
| `chatgpt` / `backend-client` / `cloud-*` / `connectors` / `aws-auth` / `workload-identity` / `agent-identity` / `keyring-store` / `secrets` / `codex-backend-openapi-models` / `codex-client` | ≈40k | 不搬 | 云侧或身份体系，本地单用户无对应（`agent-identity` 已在 asset-18 判定） |

### J. 沙箱与 OS

| crate | LOC | 判定 | 理由 |
|---|---|---|---|
| `sandboxing` | 10,974 | 不搬（借模型） | 策略数据模型（读/写/网络三态）已在 P2-5 采用 |
| `linux-sandbox` | 11,812 | 不搬 | landlock/seccomp；Wails Go 侧不接管 |
| `windows-sandbox-rs` / `windows-sandbox-service` | 34,226 | 不搬 | AppContainer/WFP/提权代理 |
| `bwrap` / `mxc-sandbox` / `process-hardening` | ≈3.2k | 不搬 | OS 层 |

### K. 其它小件（未单列的 ≈53 个 crate）

`utils/*` 子 crate（`pty` 8.9k、`path-uri` 3.9k、`stream-parser` 1.3k、`output-truncation` 685、`fuzzy-match` 149 …）、
`file-search`(1,365)、`file-watcher`(1,446)、`file-system`(1,188)、`git-utils`(4,588)、`worktree`(1,937)、
`install-context`(882)、`codex-home`(293)、`build-info`(322)、`stdio-to-uds`、`ansi-escape`、`async-utils`、
`arg0`(756)、`terminal-detection`(1,189)、`runtime-agent-instance`(1,562)、`thread-manager-sample`(491)、
`v8-poc`(73)、`test-binary-support`(70)、`config-schema`(19)。
→ 判定：**两项已覆盖**（`utils/output-truncation` + `utils/fuzzy-match` → asset-48，含 `utils/string` 的
`truncate.rs`）；其余未覆盖但按需取用（单项都 < 9k，剩下唯一值得读的是 `pty` 的会话生命周期）。

---

## 3. 判定分布（127 个 crate；0.170 对齐 +5 见 hotspot-13）

| 判定 | 数量（约） | 说明 |
|---|---|---|
| 覆盖（有独立资产文档、机制级） | **≈35** | 本轮新增：app-server 准入/关闭（47）+ 派发与脱敏（60）、transport/daemon（55）、thread-store/state（52）、core-plugins（57）、rmcp-client/codex-mcp（58）、http-client/model-provider 家族（53）、analytics/otel/rollout-trace（54）、exec-server（56）、core/tools（50）、core/session（49）、core/config+agent（61）、截断与模糊（48）、小件合集（59） |
| 部分（只到协议/入口层，细节未逐行） | **≈6** | `app-server`（request_processors 细节）、`app-server-protocol`、`app-server-client`、`core`（除上述子系统外的边缘文件）、`tools` crate、`exec` |
| 未覆盖 | **5** | 0.170 新增 crate（agent-message-board-host / message-board-mcp / plugin-host / exec-server-cli / http-headers-helper，见 hotspot-13；域判定不搬，暂不排期） |
| 不搬（附理由） | **≈81** | 云/身份、OS 沙箱、终端载体、进程载体、样板与彩蛋（§5） |

> 合计 127（`ext` 的 18 个子 crate 与 `utils` 的 25 个子 crate 按父目录计，明细见 §2 各行）。
| 不搬（附理由） | ≈72 | 云/身份/OS 沙箱/终端载体/测试与样板（§5） |

---

## 4. 真缺口 Top 12（按 价值×规模×新颖 排序）

> 判据：能否给 gaea 带来**它现在没有的机制**；不看行数多少。

| # | 目标 | 关键证据文件 | 一句话机制 | gaea 落点 | 规模 |
|---|---|---|---|---|---|
| B1 | **app-server 会话编排与回合闸门** ✅ `asset-47` | `app-server/src/{turn_admission.rs, connection_rpc_gate.rs, connection_cleanup.rs, thread_status.rs, message_processor.rs, request_processors/thread_processor.rs}` | 连接级准入 + 优雅 drain + 线程活跃标记：`admit()` 与 `begin_drain()` 同一把锁；关闸后**排队的不再启动、已拿 token 的跑完**；`is_closed()` 同时看 `TaskTracker` 与 auth 所有权 | `internal/gaea/control`（回合准入）/ `internal/app`（Wails 桥的关闭语义） | 大（已出施工卡三刀 + 8 条断言） |
| B2 | **app-server-transport + daemon** ✅ `asset-55`（判定不搬实现） | `app-server-transport/src/*`、`app-server-daemon/src/*` | 多传输统一 `ConnectionAuth`；守护进程 socket 发现 + 版本握手 + 重启恢复 | 桥接层（若做多入口） | 中 |
| B3 | **thread-store 与 state/runtime 持久化** ✅ `asset-52` | `thread-store/src/local/*`（read/list/search/segment_paging/rollout_migration/rollback_plan）、`state/src/runtime/{threads,memories,goals,reclamation,queued_items}.rs`、`state/src/migrations.rs` | 线程即"rollout 行文件 + 元数据索引"双写；游标分页 + 片段化历史 + 迁移回放；sqlite 页回收 worker（`IDLE 60s / ACTIVE 50ms / CONTENTION 500ms / PASS 100ms`、`MIN_FREE 64MiB`、`PASS_PAGES 1024`、`BATCH_PAGES 64`、per-home 文件锁选主、NFS 跳过） | gaea SQLite + 分支树 | 大 |
| B4 | **core/session 回合状态机** ✅ `asset-49` | `core/src/session/{turn.rs, step_activation.rs, step_settings.rs, input_queue.rs, turn_input.rs, token_budget.rs, rollout_reconstruction.rs}` | 一步 = 一次模型调用 + 工具批次；step 0 由 `run_turn` 拥有且失败先保输入；设置是快照、运行中只走受限更新（十项禁止变化、`ptr_eq` 目标确认、三态结果）；输入双通道 + 先订阅再检查 + 插话即取消采样；重放只认"最新可用压缩点" | `internal/gaea/control/controller*` | 大（已出三刀 + 9 条断言） |
| B5 | **core/tools 注册与派发** ✅ `asset-50` | `core/src/tools/{registry.rs, spec_plan.rs, runtimes/*, handlers/*, approvals.rs, executed_tool_calls/request_metadata.rs}` | 工具规格→运行时→审批三段；spec_plan 生成与 request_metadata 归因 | 工具注册表（已有 `gaea-tool-registry-map`） | 中 |
| B6 | **ext/extension-api 扩展框架** ✅ `asset-51` | `ext/extension-api/src/*`、`ext/{goal,queue,items,web-search,history-notes,image-generation}/src/*` | 扩展 = 能力声明 + 生命周期钩子 + 主机服务注入（`HostSkillsService`、`TurnStartAdmission` 都从这里出） | 插件/扩展层 | 中 |
| B7 | **core-plugins 市场与远程分发** ✅ `asset-57` | `core-plugins/src/{remote,marketplace_add,marketplace_upgrade,marketplace_policy,manifest,command_migration,startup_sync}/*` | 插件清单 → 远程拉取 → 策略校验 → 升级/迁移 | 若做插件生态 | 中 |
| B8 | **rmcp-client + codex-mcp** ✅ `asset-58` | `rmcp-client/src/**`（含 `oauth`、`mcp_2026_*` 测试）、`codex-mcp/src/{mcp,connection_manager,codex_apps}` | 传输矩阵选择、OAuth、工具缓存与失效、把自身工具暴露为 MCP | MCP 接入（asset-42 只到表单层） | 大 |
| B9 | **http-client + model-provider** ✅ `asset-53` | `http-client/src/{route_aware_client_pool,outbound_proxy,windows_proxy_auth}.rs`、`model-provider*`、`codex-api/src/{endpoint,sse}` | 路由感知连接池、出网代理、按端点分流；模型供给抽象与目录刷新 | 出网与模型接入 | 中 |
| B10 | **analytics + otel + rollout-trace** ✅ `asset-54` | `analytics/src/*`、`otel/src/{metrics,events}`、`rollout-trace/src/*` | 事件族 → 批量 → 脱敏 → 导出；trace 落盘可回放 | 可观测性 | 中 |
| B11 | **exec-server 实现层** ✅ `asset-56`（判定不搬实现） | `exec-server/src/{server,discoverV2,noise_relay,remote,client,sandbox_integrity,no_follow}` | 执行器发现（v2）、加密 relay、远端执行与完整性校验 | 若做远程执行 | 中 |
| B12 | **零散高价值小件** ✅ `asset-48`（截断+打分；`pty` 未读） | `utils/output-truncation`、`utils/string/truncate.rs`、`utils/fuzzy-match`、`utils/pty`、`file-search`、`git-utils`、`history` | 截断口径（Bytes/Tokens 互折 4:1、头尾 50/50、两种 marker 语义、1.2 余量只乘一次、MCP 迭代缩放、图片豁免）、模糊打分子序列 + 前缀奖励 | 各处小改 | 小（已出两刀 + 7 条断言） |

**本轮实测速记（已读到代码，可作 B1/B3 的起点）**

1. **准入与 drain 同锁**：`TurnAdmission::admit()` 与 `begin_drain()` 取同一把 `Mutex`，permit 的 `Drop` 递减；
   这样"关闸瞬间还在准备中的请求"不会漏过去，也不会被当成已完成。
2. **连接闸门三态**：`ConnectionRpcGate::run()` 先拿 `TaskTracker` token 再跑 future；`close()` 只关"是否接受"，
   `shutdown()` 才 `tasks.wait()`；`is_closed()` 把 **auth 所有权变更**也算成关闭——换账号后旧队列直接作废。
3. **清理任务专职化**：`ConnectionCleanupTasks` 用 `JoinSet` + `reap_next()`（空时 `pending()` 挂起）避免忙轮询，
   `abort()` 用于硬退。
4. **线程活跃标记**：`ThreadWatchManager` 用 `watch::Sender<usize>` 广播"运行中回合数"，
   活跃守卫 `ThreadWatchActiveGuard` 在 drop 时回落，并推 `ThreadStatusChangedNotification`。
5. **配置写入是分层合并**：`ConfigManagerService` 走 `ConfigLayerStack` + `MergeStrategy` + `OverriddenMetadata`，
   写盘前解析符号链接并原子写（`resolve_symlink_write_paths` + `write_atomically`），错误码收敛成 `ConfigWriteErrorCode`。
6. **热刷新分级**：MCP 重载有 `ReloadPolicy::Strict / BestEffort` 两档（同一个状态机，
   失败时严格档上抛、宽松档只 warn）；skills 走文件监听 + **10s 节流**（测试 50ms）+ `SkillsChangedNotification`。
7. **页回收有明确预算**：见 B3 常量；且"每 home 一个文件锁选主、NFS 不开维护连接"——避免多进程同时压 VACUUM。
8. **成本轮询独立 worker**：`TurnCostWorker` 150s 轮询 / 15s 超时 / 观测通道 16384 / 最多跟踪 4096 回合 /
   100 次查询 / 5 次停滞即放弃——是"后台任务必须有上限"的范式。

---

## 5. "不搬"清单与理由（六类）

1. **云侧与身份**：`login`、`chatgpt`、`backend-client`、`cloud-*`、`connectors`、`aws-auth`、`workload-identity`、
   `agent-identity`、`keyring-store`、`secrets`、`codex-backend-openapi-models`、`codex-client` —— gaea 单用户本地，无多租户/云注册。
2. **OS 级沙箱**：`sandboxing`（仅借策略模型）、`linux-sandbox`、`windows-sandbox-rs/-service`、`bwrap`、`mxc-sandbox`、
   `process-hardening` —— 由宿主/Go 侧承担，且是实现细节不是可搬机制。
3. **进程与传输载体**：`uds`、`stdio-to-uds`、`tcp-tunnel`、`daemon_*` 的 socket 细节、`arg0`、`terminal-detection`。
4. **测试与样板**：`app-server-test-client`、`thread-manager-sample`、`test-binary-support`、`v8-poc`、
   `config-schema`、`*-noop-macros`、`collaboration-mode-templates`。
5. **本地模型接入**：`ollama`、`lmstudio` —— gaea 走自有 provider 层。
6. **已判定不做的能力**：`code-mode*`（asset-20）、`agent-graph-store`（asset-06）、`realtime-webrtc` 的双工实现（asset-16，只借交接机制）。

---

## 6. 维护约定

- 新增后端资产文档时**必须同时改本台账**（§2 对应行 + §3 计数），否则视为未完成。
- 判定从"未覆盖"改为"覆盖"时，写清**读到了哪一层**（协议层 / 实现层 / 常量级）。
- 上游版本变化（0.166+）只需重跑：
  `Get-ChildItem <rs> -Directory` → 与本台账 crate 清单 diff；再按 crate 跑 `*-tests.rs` 数量对比，
  新增 crate 默认进"未覆盖"，删除 crate 的锚点要一起删。
- 本表的 LOC 只作相对大小参考；要精确数字就用 §6 的脚本口径重算（含/不含测试分别给）。

---

## 7. 与既有批次的关系

- 第一批（T1–T5）、第二批（T6–T15）、前端批（F，34 条）都是**落地施工单**；本台账是**覆盖地图**。
- 本台账 §4 的 B1–B12 是下一批（**第四批 · 后端波**）的候选，施工卡见 `plan-batch4-backend-wave.md`。
- 去重：B3 与 T9（压缩）、B5 与 T8（工具面预算）、B10 与前端 F（用量页）都有交叠，
  已在批 4 计划里逐条标注"只做哪一半"。
