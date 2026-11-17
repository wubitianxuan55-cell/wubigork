# 第四批蒸馏计划：后端波（B 批 · 只出计划与调研，不改代码）

> 定位：前三批是 **T1–T15（后端机制落地）** 与 **F 批 34 条（前端落地）**；
> 本批是**后端覆盖补全**——把 `backend-coverage-ledger.md` §4 的 Top 12 缺口收敛成可排期的条目。
>
> **本篇是"研究视图"（12 项全部已产出文档）。落地排序看 `plan-batch4-landing-priorities.md`（决策视图）**：
> 过"可感知 / 可度量 / 成本≤收益"三道闸后只留 9 项，其中 **Wave 1 五件（B1 关闭语义、B12 截断口径、B5 归因、
> B9 出网安全、B7 构造即验证）合计约 4.5 天**，B2/B11/B7 主体等直接归档不排期。
>
> 本批**只出文档**：读源码、写机制说明与施工卡，不动 `internal/`、不动 `frontend/`。
> 落地要等 T 批收口 + 用户拍板（§8）。

---

## 1. 为什么要这一批

- 前端已按台账收口（`tui` 160 文件全部钉住），**后端只有 20 个 crate 到机制级**，`app-server` 家族（271k）
  与 `thread-store`/`state`（62k）基本空白——而这三块恰好是 **gaea 会话编排与持久化** 最直接的对标物。
- 前两批读的是**"机制单点"**（审批、压缩、预算、护栏），缺的是**"编排骨架"**：
  一次回合怎么被准入、怎么被 drain、线程状态怎么广播、历史怎么存与分页、后台任务怎么设上限。
- 这三件事在 gaea 里都已经有实现（`control/controller*`、SQLite 会话、`bindings`），
  但没有"上游是怎么收敛失败与关闭"的对照——正是本批要补的。

## 2. 与前序批次的去重（先定，否则会返工）

| 本批条目 | 会撞的既有项 | 切法（只做哪一半） |
|---|---|---|
| B3 持久化 | T9 压缩三形态 | T9 只做**上下文压缩**；B3 只做**落盘形态**（行文件 + 元数据索引 + 迁移 + 页回收） |
| B5 工具注册/派发 | T8 工具面预算 | T8 做**对外披露与预算**；B5 做**内部注册、运行时选择、request_metadata 归因** |
| B6 扩展框架 | asset-17 插件、asset-05 hooks | asset-17/05 是**资源与事件**；B6 是**扩展 trait 与能力注入**（`extension-api`） |
| B9 网络与模型 | asset-19 network-proxy、asset-25/26 API 语义 | 已覆盖**决策与语义**；B9 只补**连接池、路由分流、模型供给刷新** |
| B10 观测 | asset-40（前端用量页） | 前端只做**展示**；B10 做**事件族、批量、脱敏、导出** |
| B1 编排 | T10/T11 审批与 Guardian | 审批/Guardian 是**决策点内容**；B1 是**决策点怎么被调度与关闭** |

## 3. 条目清单（12 条，分三波）

> **已交付（12/12）**：B1 `asset-47`、B2 `asset-55`、B3 `asset-52`、B4 `asset-49`、B5 `asset-50`、B6 `asset-51`、
> B7 `asset-57`、B8 `asset-58`、B9 `asset-53`、B10 `asset-54`、B11 `asset-56`、B12 `asset-48`。
> 另出**收口三篇**：`asset-59`（小件合集，未覆盖清零）、`asset-60`（app-server 派发/订阅/脱敏）、
> `asset-61`（配置层与多智能体运行面）。

### 波 A · 编排骨架（4 条，先做，读代码为主）

**B1 · 回合准入与优雅关闭**（大，1.5–2 天出文档）
- 证据：`app-server/src/{turn_admission.rs, connection_rpc_gate.rs, connection_cleanup.rs, thread_status.rs}`。
- 要写清的四条：① `admit()` 与 `begin_drain()` 同锁的因果；② 关闸后"排队不启动、已启动跑完"的两段语义；
  ③ auth 所有权变更 = 连接关闭（`is_current()`）；④ 活跃数用 `watch` 广播而不是轮询。
- gaea 落点：`internal/gaea/control`（回合准入）、`internal/app`（窗口关闭/切换时的 drain）。
- 验收口径：文档给出"关窗时正在跑的回合怎么处理"的**决策表**（4 种状态 × 3 种结果）。

**B2 · 传输与守护进程**（中，1 天）
- 证据：`app-server-transport/src/*`、`app-server-daemon/src/*`。
- 关注：多传输如何统一 `ConnectionAuth`、framing 与背压、daemon 的 socket 发现/版本握手/重启恢复。
- 判定倾向：**暂不落地**，只作为 B1 的前置知识（gaea 单进程 Wails，没有多传输需求）。

**B4 · 回合状态机**（大，2 天）
- 证据：`core/src/session/{turn.rs, turn_context.rs, turn_input.rs, step_activation.rs, pending_input.rs, mcp.rs, rollout_reconstruction.rs}`。
- 要写清：一步的边界（模型调用 + 工具批次）、step 激活/停用条件、待处理输入的消费与丢弃、
  rollout 重建（历史重放）如何保证幂等。
- gaea 落点：`internal/gaea/control/controller*`（现有单轮驱动）。

**B12 · 高价值小件**（小，0.5 天，可与 A 波并行）
- 证据：`utils/output-truncation`、`utils/pty`、`utils/fuzzy-match`、`file-search`、`git-utils`、`history`。
- 只取三条：截断口径（头尾保留比例）、pty 会话生命周期、模糊打分（与前端 asset-35 的排序对齐）。

### 波 B · 存储与扩展（4 条）

**B3 · 线程存储与状态库**（大，2–2.5 天）
- 证据：`thread-store/src/local/*`（`read_thread`/`list_threads`/`search_threads`/`segment_paging`/
  `rollout_migration`/`rollback_plan`/`rollback_replay`/`archive`/`line_parser`/`publish`）、
  `state/src/runtime/{threads,memories,goals,reclamation,queued_items,backfill,recovery}.rs`、`state/src/migrations.rs`。
- 要写清：① 线程 = rollout 行文件 + 元数据索引的**双写与一致性**；
  ② 游标分页 + 片段化历史（不整份加载）；③ 迁移/回填的可重放性；
  ④ 页回收预算（`IDLE 60s / ACTIVE 50ms / CONTENTION 500ms / PASS 100ms`、`MIN_FREE 64MiB`、
  `PASS_PAGES 1024`、`BATCH_PAGES 64`、per-home 文件锁选主、NFS 跳过）。
- gaea 落点：SQLite 会话表 + `docs/gaea-chat-branch-tree-design-2026-11.md`。

**B5 · 工具注册与派发**（中，1.5 天）
- 证据：`core/src/tools/{registry.rs, spec_plan.rs, runtimes/*, handlers/*, approvals.rs, executed_tool_calls/request_metadata.rs}`。
- 要写清：规格→运行时→审批三段边界；`spec_plan` 的生成时机；`request_metadata` 归因（谁调用了哪个工具、几个回合）。

**B6 · 扩展框架（extension-api）**（中，1.5 天）
- 证据：`ext/extension-api/src/*` + `ext/{goal,queue,items,web-search,history-notes,image-generation}/src/*`。
- 要写清：能力声明、生命周期钩子、主机服务注入（`HostSkillsService`、`TurnStartAdmission` 的来源）、
  扩展如何拿到"当前回合/线程"上下文。
- gaea 落点：工具/插件注册表的分层。

**B7 · 插件市场与远程分发**（中，1 天，**待拍板**）
- 证据：`core-plugins/src/{remote,marketplace_add,marketplace_upgrade,marketplace_policy,manifest,command_migration,startup_sync}`。
- 要写清：清单→拉取→策略校验→升级/迁移的链路；策略拒绝时的用户可见错误。
- 前置：gaea 是否要做插件生态（§8 拍板门 1）。

### 波 C · 网络、观测与执行服务（4 条）

**B8 · MCP 双端**（大，2 天）
- 证据：`rmcp-client/src/**`（含 `oauth`、`mcp_2026_*` 传输）、`codex-mcp/src/{mcp,connection_manager,codex_apps}`。
- 要写清：传输矩阵选择规则（stdio / streamable HTTP / WS 的能力差异）、OAuth 失效处理、
  工具缓存与失效、把自身工具暴露为 MCP 的命名与权限。
- 与 asset-42（集成面表单）不重复：那篇只到"页面怎么画"。

**B9 · HTTP 客户端与模型供给**（中，1.5 天）
- 证据：`http-client/src/{route_aware_client_pool,outbound_proxy,windows_proxy_auth}.rs`、`model-provider*`、`codex-api/src/{endpoint,sse}`。
- 要写清：路由感知连接池（按端点分流、池上限）、出网代理优先级、模型目录刷新与回退。

**B10 · 观测三件**（中，1.5 天）
- 证据：`analytics/src/*`、`otel/src/{metrics,events}`、`rollout-trace/src/*`。
- 要写清：事件族与字段、批量与丢弃策略、脱敏边界、trace 落盘与回放。
- 与前端 asset-40 去重：只写"后端怎么产生与上报"。

**B11 · 执行服务实现层**（中，1 天，**待拍板**）
- 证据：`exec-server/src/{server,discoverV2,noise_relay,remote,client,sandbox_integrity,no_follow}`。
- 要写清：执行器发现（v2）的判定、加密 relay 的作用域、远端执行失败回退。
- 前置：gaea 是否要远程执行（§8 拍板门 2）。不拍板则本篇只写"判定不做 + 三条可借实践"。

## 4. 顺序约束（违反就返工）

1. **B1 先于 B2**：没有准入/drain 语义，传输层写出来也无法判断"关连接时谁该活"。
2. **B4 先于 B5**：工具派发挂在回合 step 上，先定 step 边界再定派发。
3. **B3 与 B6 可并行**，但都要在 B4 之后读——它们都消费"回合/线程"的数据形状。
4. **B8 先于 B9**：MCP 传输与 HTTP 客户池共用出网策略，先定 MCP 再定池。
5. **B10 最后**：它记录的是 B1/B3/B4 的形状，先写会返工。
6. 三条纪律仍生效：先记基线数字；新能力默认关闭；**机制照抄、文案自写**。

## 5. 验收口径（每篇文档最小证据）

| 类别 | 最小证据（缺一不可） |
|---|---|
| 机制说明 | `file:line` 级证据 ≥ 3 处；能回答"为什么不那样做" |
| 常量/预算 | 每个数值给出来源文件；标清是生产值还是测试值 |
| 状态机 | 状态表（状态 × 事件 × 结果），含终态与不可达态 |
| 失败路径 | 至少 2 条失败/取消路径，写清错误如何收敛 |
| 与 gaea 对照 | 现有实现位置（`file:line`）+ 差在哪一句 |

## 6. 门禁（本批只读，不跑测试）

- 本批**不改代码**，因此不跑 `npm run build` / `go test` / `smoke.ps1`；
  只在 §7 汇报里声明"本轮零代码改动"。
- 文档自检：每个新文档必须在本台账/crate 表里登记，否则算未完成。

## 7. 风险登记

| # | 风险 | 影响 | 缓解 |
|---|---|---|---|
| 1 | `core` 44 万行无边，读着读着跑偏 | 工期炸 | 只按 B4/B5 的**子目录白名单**读，不碰 `tests/suite`（除引用断言） |
| 2 | `app-server` 的 tests 占 124k，实现只 31k | 容易把测试当实现 | 先读 `src/`，测试只用来确认断言 |
| 3 | B3 与现有 gaea SQLite schema 冲突 | 落地时返工 | 本篇只写"上游形态 + 差异"，不出 schema 建议 |
| 4 | B8 MCP 有 `mcp_2026_*` 实验协议 | 结论半年失效 | 标清"哪些是稳定契约、哪些是实验分支" |
| 5 | 12 条一起读，文档互相引用成环 | 无法独立阅读 | 每篇自带"前置知识"小节，不假设读者读过别篇 |

## 8. 需要拍板的 3 个门

1. **插件生态**（B7）：gaea 要不要做插件市场/远程分发？不拍板 → B7 降级为"只写判定与三条可借实践"。
2. **远程执行**（B11）：要不要跨机执行器？不拍板 → 与 asset-21 一致，只留判定。
3. **多入口**（B2）：除 Wails 单窗口外是否要 daemon/多客户端？不拍板 → B2 只作 B1 前置知识，不出施工卡。

## 9. 建议排期（文档工作量，不含落地）

| 波 | 条目 | 文档工作量 | 产出 |
|---|---|---|---|
| A | B1 / B4 / B12（B2 降级） | 3.5–4 天 | 3 篇资产 + 1 篇旁证 |
| B | B3 / B5 / B6（B7 视拍板） | 5–6 天 | 3–4 篇资产 |
| C | B8 / B9 / B10（B11 视拍板） | 4.5–5 天 | 3–4 篇资产 |

合计 **9–11 篇**（编号接 `asset-47…`），分三波交付，每波结束更新 `backend-coverage-ledger.md` 与 `README.md`。

## 10. 汇报格式（每篇完成贴一段）

```
[B<n>] <标题>
- 证据：<file:line ×3+>
- 机制一句话：<...>
- 关键常量/状态：<...>
- gaea 现状：<file:line>；差：<...>
- 判定：覆盖 / 旁证 / 不搬（理由）
- 台账已更新：backend-coverage-ledger.md §2 §3
```
