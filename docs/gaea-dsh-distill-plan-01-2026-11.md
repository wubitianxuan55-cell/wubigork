# DSH 蒸馏计划 · 第一篇：基线对账与子代理全链路收口（2026-11）

> **源**：`deepseek-ai/deepseek-harness`，本轮核对 tag **`dsh-v0.2.2-rc.1`**（`66491b05`），
> 源码检出 `C:\AI\dsh-src\deepseek-harness-0.2.2-rc.1`（20,287 文件，只读）；对账基线 tag **`dsh-v0.2.2-alpha.2`**。
> **本机运行态**：npm 安装产物 `C:\AI\dsh\node_modules\@deepseek-ai\*` = `0.2.2-rc.1`，
> 叠加本机 fork **`dsh-wubi 2.1.0`** 的二进制级 patch（见 §1.3）。
> **前情**：`gaea-dsh-subagent-distill-2026-10.md`（0.2.1 判定级+刀池）、
> `gaea-dsh-subagent-deep-distill-2026-11.md`（alpha.2 机制级）、
> `gaea-dsh-subagent-ui-distill-2026-11.md`（alpha.2 UI 面）、
> `gaea-dsh-022-distill-2026-11.md`（0.2.1-rc.1→alpha.2 增量）、
> `gaea-dsh-better-sidebar-long-term-distill-plan-2026.md`。
> **本次逆向产物**（证据包，**不入本仓**）：`C:\AI\dsh-subagent-reverse\` 共 10 篇 / 约 5,900 行（清单见 §3）。
> **纪律**：取道不取器；**本篇只出计划与判定，不改 gaea 代码**；台账先行，不为蒸馏而蒸馏。

---

## 〇、本篇定位

| 问题 | 回答 |
|---|---|
| 这一批做什么 | ① 把蒸馏**基线从 alpha.2 对账到 rc.1**；② 把本次**全链路逆向**收敛成可执行资产与首批任务；③ 建立 dsh 蒸馏的**四维覆盖台账**（机制 / 前端 / UI / 第三方），此后每轮只补缺口 |
| 不做什么 | 不改 gaea 代码、不动 Wails 绑定面；不追 alpha.2 之后与子代理无关的板块（client 大改只取与 subagent 有关的部分）；不搬 UI 实现 |
| 完成判据 | 每条任务有：落点 `file:line`、验收断言、规模估算、回退口径；每条资产有：证据等级（逐行 / 抽查 / 签名级）与覆盖边界 |
| 与既有六篇关系 | **不是推翻，是补口**：既有六篇的机制判定在 rc.1 依然成立（§1.2 证明）；本次新增的是**逐行级证据 + 五个此前未覆盖的面** |
| 总量估算 | 首批 8 项（D1–D8），S=0.5 / M=1 / L=2 / XL=3 人日；合计约 **9–13 人日**，可按 D 项独立交付 |

**一句话**：本篇不是"又一个版本的蒸馏"，而是把已有的"判定级 + 机制级"升级为**逐行级 + 台账化**，并把 gaea 侧落差收敛成排期。

---

## 一、版本对账：alpha.2 → rc.1（本篇的硬依据）

### 1.1 全仓差异（`git diff --stat dsh-v0.2.2-alpha.2 HEAD`）

```
2315 files changed, 87654 insertions(+), 27240 deletions(-)
```

按顶层目录分布（前 12）：

| 目录 | 变更文件数 | 与子代理的关系 |
|---|---|---|
| `packages/client` | **609** | ⚠️ **直接相关**：`ui-subagent` / `ui-settings-subagent` / `ui-chat` 的 UI 契约可能已变 |
| `packages/experimental` | 280 | 相关：`agent-team`、`subagent-oar` |
| `snapshots/session`、`snapshots/sdk` | 171 / 65 | 间接：投影与恢复快照 |
| `apps/web` | 127 | 相关：前端宿主装配 |
| `.agents/notes` | 96 | 相关：设计决策（蒸馏史料） |
| `packages/fs` | 75 | 无关 |
| `packages/schedule` | 53 | 无关（但 preset 工具面相关） |
| `docs/upgrade-guide`、`docs/subsystems` | 51 / 45 | 相关：权威文档 |

### 1.2 子代理包差异：**机制冻结**

```
packages/subagent 全域：10 files changed, 24 insertions(+), 20 deletions(-)
  ├─ 9 × package.json           （版本号 0.2.2-alpha.2 → 0.2.2-rc.1）
  └─ 1 × subagent/tests/schedule-tools.spec.ts（26 行测试调整）
```

**判定（本篇最重要的一条）**：
`SubagentRuntime` / `SubagentManager` / `descriptor` / `catalog` / `structured` / `control` / `depth` / `lifecycle`
在 alpha.2 → rc.1 **零实质变更**。因此——

- 既有 `gaea-dsh-subagent-deep-distill-2026-11.md` 的**机制判定全部继续有效**，不需重做；
- 本次逆向的价值**不在"追新版本"**，而在：① 逐行级证据（既有档多为机制级摘述）；② 补 alpha.2 档未覆盖的五个面（§2）；
- **`gaea-dsh-subagent-ui-distill-2026-11.md` 需要按 rc.1 复核**：client 609 个文件变更，UI 契约不能假定未变。

### 1.3 本机 fork 的叠加事实（已 diff 实测）

| 文件 | 改动 | 影响 |
|---|---|---|
| `dsh-subagent/lib/index.js` | `maxDepth` 默认 **1 → 10** | 默认允许多层委派 |
| `dsh-subagent/lib/index.js` + `lib/types/child-agent.js` | `SUBAGENT_DELEGATION_CONTEXT` 整段替换为「approval auto-granted / scope 可放宽」 | **只改提示词文本**；`approvalPolicy:'never'` 与 sandbox 继承未动 → 放开的是审批维度，沙箱仍锁父级 |
| `dsh-tool-subagent/lib/index.js` | `maxDepth` 补 `.default(10)` | **Host 的深度设置被架空**（`resolveMaxDepth` 走 `configured !== undefined` 分支） |

> 结论：**gaea 若要以本机形态对齐，必须在 preset 显式写 `maxDepth`**，不能依赖 Host 设置。

---

## 二、既有版图与本次补口

| 既有文档 | 基线 | 已覆盖 | 本次补什么 |
|---|---|---|---|
| `gaea-dsh-subagent-distill-2026-10.md` | 0.2.1-alpha.2 | 判定级 + 刀池（P0/P1 已落地） | 判定不变，无需动 |
| `gaea-dsh-subagent-deep-distill-2026-11.md` | **alpha.2** | 机制级：结算四态、双闸、结构化出口、持久化、权限、模型面 | **① 逐行级生命周期**（八阶段 + 15 步释放顺序）；**② 父子三条通道**（`omitClosingText` 的两条复位路径、`sendWaking` 的 queue/steer 切换）；**③ 宿主等待循环的 seq 复核** |
| `gaea-dsh-subagent-ui-distill-2026-11.md` | **alpha.2** | UI 面：谱系树、指标行、只读三态词汇 | **① 按 rc.1 复核 client 609 文件变更**；**② 五个此前没覆盖的 UI 面**（目录切换器、后代计数触发器、portal 目录树、左栏徽标、归档确认） |
| `gaea-dsh-022-distill-2026-11.md` | rc.1→alpha.2 | 增量对账、四条 K 刀 | 补 §1.2 的"机制冻结"结论 |
| `gaea-dsh-better-sidebar-long-term-distill-plan-2026.md` | 长期计划 | 长期路线 | **本次首次给出 Tasks 页机制级拆解**（`07`），可直接作为该长期计划的第一批输入 |
| （无） | — | — | **全新面**：外部 provider（`04`）、caller 投递与 Agent Teams（`06`）、gaea 编队页落差诊断（`08`） |

---

## 三、本次逆向资产清单（证据包，不入仓）

| 编号 | 文件 | 规模 | 证据等级 | 主要可用面 |
|---|---|---|---|---|
| **D01** | `00-runtime-evidence.md` | 172 行 | 实测（diff / 运行态） | 本机 patch 三处、projection 版本对账、preset 工具面成本 |
| **D02** | `backend.md` | 1226 行 | 逐行 + 抽查 | 服务/provider 契约、RPC 安全面、归档准入、错误码全表 |
| **D03** | `frontend.md` | 1189 行 | 逐行 + 抽查 | cordis client 插件树、三投影订阅、两条写 RPC |
| **D04** | `ui.md` | 823 行 | 逐行 | 16 个可见表面、文案表、状态机、真实设计 token |
| **D05** | `04-external-providers.md` | 691 行 | 逐行 | ACP/Codex/Claude/SDK/OAR：单次执行依据、脱敏、id 冲突回滚 |
| **D06** | `05-lifecycle-and-linkage.md` | 402 行 | 逐行 | 八阶段、三通道、结构化两阶段提交、drain 家族、宿主循环 |
| **D07** | `06-caller-and-agent-teams.md` | 715 行 | 逐行 + 抽查 | caller 的 Promise 通道、workflow 消费、Teams 成员模型 |
| **D08** | `07-better-sidebar-topology.md` | 194 行 | 逐行 | Tasks 页：拓扑根、`subagents.live`、0→N 弹出、折叠分组、自研布局 |
| **D09** | `08-gaea-fleet-vs-tasks.md` | 8 章 | 只读对照 | gaea 编队页落差诊断 + P0/P1/P2 施工方案 |
| **D10** | `README.md`（总览） | 191 行 | 索引 | 12 条结论 + 6 条文档/实现不一致 + 交叉核对表 |

---

## 四、首批任务（D1–D8）

> 编号沿用 `D`（distill），与 codex 的 T/F 批区分。每项：落点 → 验收 → 规模。
>
> **执行状态（2026-11-18 首批收口，两笔提交）**：D1/D2/D3 一笔（编队拓扑会话化+折行布局+折叠聚合）、
> D4/D5/D6 一笔（aborted 结算词汇+单闸补齐+独立 Stop，wenxue 镜像同批）；D7 留档不立项（见下）；
> D8 未动（P2 demand-gated）。门禁=go vet+agent/app(117s)/trajectory 全绿、tsc -b 0、
> 改动面 ESLint 0、context 族 34/34+SubagentThread 20/20+wenxue 11/11 绿。

### D7 执行判定（2026-11-18，留档不立项）

上游防的是「投影缓存跨格式静默错读」（projection stateVersion 6/3/2）。gaea 无同构面：
`readEntriesForCached` 缓存的是**原始 LogEntry**（size+mtime 失效、折叠每次现算，
gaea_ui_contextview.go:59），不存在旧缓存当新格式读的静默错读；日志格式演进走读端宽容
降级（未知 kind 跳过/字段缺省兜底/Turn.trigger omitempty 旧日志兜底）。引入版本号机制
属于无需求机械面，按 §5「不为蒸馏而蒸馏」纪律不立项。

### D1 · 编队页拓扑"会话化"（**最高收益**）

- **现状**：`AgentTopology.tsx` 的 `matchRun()` 用 `run.ref === node.id` 否则 `task` 前缀互含**猜**匹配；
  拓扑骨架来自 `internal/gaea/trajectory/types.go:169` 的 `AgentNode`（**工具调用 ID**）。
- **要做**：给 `SubagentRunView` 补 `parentRef`，把聚合单位从"元工具调用"改为"子代理会话"。
- **落点**：`frontend/src/gaea/lib/types/subagent.ts`、`internal/gaea/trajectory/agentnet.go`、
  `internal/gaea/agent/subagent_store.go`（写 meta 时带 `parentRef`）、`AgentTopology.tsx`（删 `matchRun`）。
- **验收**：拓扑每个节点都能对应一条 `run`（无启发式匹配）；`lastText`/`lastTool`/`status` 直接来自 run。
- **规模**：L（2 人日）。**依赖（2026-11-17 开工首验，走 meta 直写路线）**：gaea 后端可读到父子——
  spawn 时 ctx 已盖 `withSubagentSessionID`（`internal/gaea/agent/task.go:515`，持久 sa_ ref 或
  ephemeral id），只需给 `SubagentMeta` 补 `parentRef`/`spawnCallId`（applySnapshot 同款一次写定），
  无需 DSH 式 header 双条件，也无需日志扫描路线。约束：ephemeral 父（`sa_ephemeral_*`）无 meta
  侧车可指，parentRef 落空按主会话子代理降级；`continue_from` 续跑路径不盖章（parentRef 记
  「谁派生了这个 run」，续跑发起方不改变血缘）。

### D2 · 拓扑布局：折行 + 可读下限 + 行距实测（P0，零后端）

- **现状**：`agentTopologyLayout.ts` 中 `width = max(measuredWidth, contentWidth)`——宽了只把画布撑宽；
  卡片高写死 `FLEET_CARD_HEIGHT = 92`；`FLEET_LEVEL_HEIGHT = 136` 固定。
- **要做**：`bandColsFor(containerWidth, 0.78)` 折行（360px→2 列、720px→4 列）；行距按当前模型最高卡片实测。
- **落点**：`frontend/src/gaea/components/context/agentTopologyLayout.ts`。
- **验收**：20+ 子代理、右栏 360px 与主区 720px 两种宽度都不出现横向滚动；卡片信息不截断到不可读。
- **规模**：M（1 人日）。**风险**：`agentTopologyLayout.test.ts` 需同步改。

### D3 · fold 聚合分组（done / idle）

- **要做**：抽 `buildFleetModel()`（对应上游 `buildTasksModel`），每父级出两类聚合卡；
  `FOLD_IDLE_MIN = 3`；点聚合卡只展开该组（`expandedGroups`），手折（`foldedIds`）优先。
- **落点**：新增 `frontend/src/gaea/components/context/fleetModel.ts`；`AgentTopology.tsx` 渲染 fold 节点。
- **验收**：50 节点树在 720px 内不溢出；展开一组不影响其它组；手折后 chevron 不失效。
- **规模**：L（2 人日）。**依赖**：D1（模型需会话节点才有稳定 id）。

### D4 · 结算通知四形态对齐

- **上游事实**：`continuation-messages.ts:106-129` 按 `stopReason` 给四类开场白；
  正文三分支（`omitClosingText` / 无收尾 / 有收尾）+ `Structured result:` + `diagnostic`。
- **要做**：核 gaea 的结算文案（`internal/gaea/agent/task.go:1161` 一带）与 `subagentStopReason` 词汇是否同构；
  差异处按上游**词汇**对齐（不搬文案）。
- **验收**：五种 `stopReason`（completed / aborted / max-tokens / refusal / error）在 gaea 各有明确对应，无"未分类"。
- **规模**：S（0.5 人日）。

### D5 · 容量口径错配自检

- **上游事实**：workflow 的 `maxConcurrentAgents` 默认 `min(16, cores-2)`，subagent 硬限 `maxActiveSubagents = 8`，
  两个池**性质不同**（协作排队 vs 硬拒绝），16 核机器第 9 个并发即 fatal。
- **要做**：核 gaea 的并行闸（`[agent] subagent_max_parallel`）与 workflow 侧并发是否存在同类错配；
  写一条启动期断言或文档约束。
- **落点**：`internal/gaea/agent/`（并行闸配置解析处）。
- **验收**：并发上限 ≤ 子代理容量，或超限时有明确降级行为（不是整批失败）。
- **规模**：S。

### D6 · 只读态四短路复核（**已核，结论收窄为「补一个独立 Stop」**）

- **上游事实**：`ui-subagent/src/client/index.ts:37-49` 四条短路——
  `unknown`→只读、`one-shot`→只读、`parentAvailable !== false`→正常、父离线但**仍在跑**→让位给「禁用输入 + 独立 Stop」。
- **gaea 实测**（`frontend/src/gaea/components/SubagentThread.tsx`）：

  | 上游短路 | gaea 对应 | 判定 |
  |---|---|---|
  | `mode === 'unknown'` → 只读 | 无该状态（ref 由 gaea 自己产生，meta 必在） | **不适用** |
  | `mode === 'one-shot'` → 只读 | `isMtTab = target.startsWith("mt_")`（`:267`）——mt_ 禁用追问与提升（`:276`、`:345`） | ✅ **已对齐** |
  | `parentAvailable !== false` → 正常 | 无「父离线」维度（父子同进程同会话） | **不适用** |
  | 父离线 + 运行中 → 禁用输入 + **独立 Stop** | `running` 时发送按钮 `disabled`（`:644`）✅；**但 SubagentThread 无独立 Stop 入口**，且前端 bridge **未暴露** `InterruptSubagent` | ⚠️ **部分对齐** |

- **gaea 领先一条**：`status === 'failed'` 时仍允许追问（`RunFollowUp` 支持 failed ref，`:595`），
  并把后端 `followUpError` 回填成失败气泡（`:309-311`、`:609-619`）——上游把失败视为终态，没有这条正向通道。
- **要做（收窄后）**：仅补「运行中独立 Stop」——后端 `agent.InterruptSubagent(ref)` 已存在
  （`internal/gaea/agent/interrupt_tool_test.go:98,174` 有覆盖），只差 bridge 暴露 + 面板按钮。
- **落点**：`frontend/src/gaea/components/SubagentThread.tsx`、`frontend/src/gaea/lib/bridge/*`（新增一个绑定）。
- **验收**：`running` 态下 Stop 可点且生效；`mt_`／`queued`／`completed`／`failed` 四态的按钮可见性各有用例；
  停止后 `status` 经轮询转为 `failed` 并带原因。
- **规模**：S（0.5 人日）。

### D7 · 投影版本三轴台账

- **上游事实**：descriptor 格式 **3**、catalog 载荷 **0/1/2**、projection stateVersion **6 / 3 / 2**；
  本机 `session_projcache` 实测正是 6 / 3 / 2（可用于判断缓存是否本版本写的）。
- **要做**：在 gaea 会话缓存/迁移逻辑里引入同类版本判定，避免把旧缓存当新格式读。
- **落点**：`internal/gaea/`（会话 projection 缓存读写处）。
- **验收**：版本不符时显式降级并记日志，不静默错读。
- **规模**：S。

### D8 · 编队页 "GRAPH / TREE 双视图"（可选）

- **上游事实**：一套 `TasksNode[]` 模型喂两种视图，折叠态与视图切换不分叉。
- **要做**：D3 落地后，加缩进树渲染器复用同一模型；右下角 toggle。
- **落点**：新增 `FleetTree.tsx`。
- **验收**：切换视图后选中态、折叠态、过滤条件全部保持。
- **规模**：M。**优先级低于 D1–D3**。

---

## 五、刀池（按 gaea 价值排序）

| 池 | 项 | 依据 | 判定 |
|---|---|---|---|
| **P0** | D1 拓扑会话化 | `08` §4 根因 A | 立项 |
| **P0** | D2 折行布局 | `08` §5 P0 | 立项（可独立先落） |
| **P1** | D3 fold 聚合 | `08` §5 P2 / `07` §5 | 立项，D1 之后 |
| **P1** | D4 结算词汇对齐 | `06` §2.4 | 立项（小刀） |
| **P1** | D5 容量错配自检 | `06` §4 发现 | 立项（自检性质） |
| **P2** | D6 补「运行中独立 Stop」 | `ui` 只读四短路 + 本次实测：2 态不适用 / 1 态已对齐 / 1 态部分对齐 | 小刀（后端 `InterruptSubagent` 已存在，只差 bridge + 按钮） |
| **P2** | D7 投影版本台账 | `05` §9.1 | 小刀 |
| **P2** | D8 双视图 | `07` §1 | demand-gated |
| **留档不立项** | 结构化输出两阶段提交（`structured.ts` 全文） | `05` §4 | gaea 当前无 `outputSchema` 需求；机制留档 |
| **留档不立项** | caller 投递的 Promise 通道 | `06` §1 | gaea 前台阻塞形态天然等价 |
| **留档不立项** | 外部 provider 全套（ACP/Codex/Claude/SDK/OAR） | `04` | 形态不跟（一次性执行不可续聊） |
| **留档不立项** | Agent Teams 成员模型 | `06` §4–§6 | gaea 无 Team 面；只留"停止复用 drain"这条纪律 |

> **落地注记（2026-11-18）**：P0 两项（D1/D2）+P1 三项（D3/D4/D5）+P2 两项（D6/D7 判定）全部处置完毕。
> D4 实做比计划收窄处：`SubagentStopReason`（P1-C 批）已存在，本批补的是 meta/UI 侧 aborted 终态；
> max-tokens/refusal 维持「宁缺勿虚报」不造轴。D5 结论改写：gaea 无上游式两池错配（排队可见非
> fatal），实缺口是 RunNew/runFollowUp 绕闸，已补齐+行为锁。D6 按 D1 会话骨架顺带修了
> root→main 选中联动（knownRefs 校验清态的隐性错位）。

---

## 六、覆盖台账（本篇初始化）

| 维度 | 状态 | 缺口 |
|---|---|---|
| **机制（host）** | ✅ 逐行（subagent 8 模块 + manager 主干 + lifecycle/structured/internal/continuation-messages 全文） | `descriptor.ts`(11 KB) / `catalog.ts`(6 KB) / `projection.ts`(7.6 KB) 的完整校验逻辑仍为"关键段"级 |
| **外部后端** | ✅ 逐行（5 个后端 + out-of-process） | OAR 变体细节 |
| **前端（client 插件）** | 🟡 逐行（`ui-subagent` / `ui-settings-subagent` / `ui-reference` 关键路径） | `ProjectionValueStore` 合并实现、`ui-chat` 的 `source.kind` 认领细节 |
| **UI 面** | ✅ 16 面（rc.1 源码）+ **rc.1 复核已闭（2026-11-18，§7 门 1）**：子代理三 UI 包实质变更仅 3 处（floatWindow 词汇先行/FileReference 右键菜单/版本号），ui-chat 35 文件不触及 U5-U7 主张面，U1–U8 全维持 | 旧档已加复核注记，无需改写 |
| **第三方插件** | ✅ better-sidebar Tasks 页（机制级） | 其余第三方页（files / items / jobs / git）未覆盖 |
| **gaea 对照** | ✅ 编队页（16 项对照） | 其余消费面（SubagentThread / useTaskCards）未按 rc.1 复核 |

**台账纪律**：每轮蒸馏收口时更新本表；"逐行"= 读过全文并可给出行号；"抽查"= 抽过 N 处且全部吻合；"签名级"= 只有类型与文档。

---

## 七、风险与拍板门

| # | 风险 / 待拍板 | 影响 | 建议 |
|---|---|---|---|
| ~~1~~ | ~~rc.1 的 client 大改是否影响既有 UI 蒸馏~~ | ~~旧档可能已过期~~ | **已闭（2026-11-18）**：复核结论=旧档 U1–U8 全维持，实质变更仅 3 处且不在主张面（详见 ui-distill 档头部复核注记）；D6 落点当时已按 gaea 实码收窄，不受影响 |
| ~~2~~ | ~~gaea 后端能否读到子会话 `parentSession`+`origin`~~ | ~~决定 D1 路线~~ | **已闭（2026-11-17 开工首验）**：ctx 已盖 `withSubagentSessionID`（task.go:515），meta 直写 `parentRef`/`spawnCallId` 即可，见 §4·D1 依赖注记 |
| 3 | `AgentNode.id` 是三块联动的公共键 | D1 改动会波及 `Fleet`/`FleetTasks`/`FleetTimeline`/`FleetEvents` | 先加 `parentRef` 并行存在，再切渲染源，最后删旧路径（三段式） |
| 4 | 本机 patch 让 host 深度设置失效 | 若照抄本机形态会丢叶子隔离 | preset 显式 `maxDepth`，不依赖设置页 |
| 5 | "为蒸馏而蒸馏" | 刀池容易膨胀 | 按本篇 §5 的"留档不立项"清单执行，不新增无需求项 |

---

## 八、方法学注记（本次踩到的坑，供后续轮次）

1. **README 会撒谎，源码不会**：本次查到 5 处文档与实现不一致——最典型是"`ui-subagent` 提供 `@` reference source"
   （实际该包根本不注册 `inputTriggers`，真身在 `ui-reference`），以及"`maxDepth` 默认 1"（本机被 patch 成 10）。
   **纪律**：机制结论以源码为准，README 只用于建立索引。
2. **引用可能不存在**：上游某个 note 名被重复引用（"production dsh excludes product subagent providers"），
   `rc.1` 全仓检索无命中——该措辞实际出自本机部署的自制预设注释。**纪律**：引用 note 前先验证路径存在。
3. **版本有三根独立轴**，不要混：descriptor 格式（3）/ catalog 载荷（0、1、2）/ projection stateVersion（6、3、2）。
   判缓存兼容时看第三轴，判恢复能力时看第一轴。
4. **对账要用 tag diff，不要用 master HEAD**：本仓 master 停在旧合并点（既有 `022` 档已记此坑），
   本轮 `alpha.2 → rc.1` 的 2315 文件差异只能靠 tag 对 tag 拿到。
5. **subagent 包已冻结**（§1.2）：后续轮次不要重复"补课"，把精力放在 client / experimental / fs / schedule 上。
6. **本机 patch 必须单独识别**：`.dshwubi.bak` 文件是识别入口；先 diff 再下结论，否则会把本机定制误当成上游机制。

---

## 九、下一步

1. ~~拍板 §7 的 5 个门（尤其 1、2）~~——门 2 已闭（meta 直写路线），门 1 未闭（rc.1 client 609 文件 UI 复核仍未跑，不阻塞本批——D6 落点已按 gaea 实码收窄）；
2. ~~按 **D2 → D1 → D3** 顺序推进~~——**已收官（2026-11-18，D1/D2/D3 一笔）**；
3. ~~D4–D7 作为小刀，随任一批顺手带走~~——**已处置（D4/D5/D6 一笔；D7 留档不立项）**；
4. ~~本批收口时更新 §6 台账，并决定是否开第二篇~~——台账已更新；**§7 五门全闭**（门 2 meta 直写/D6 已收窄落点/门 1 复核=旧档维持/门 3 三段式已执行/门 4-5 纪律遵守）。第二篇候选只剩：第三方页扩展（files/items/jobs/git，§6 台账）+ D8 双视图（demand-gated）——均无当前需求信号，按「不为蒸馏而蒸馏」挂起待实弹信号。
