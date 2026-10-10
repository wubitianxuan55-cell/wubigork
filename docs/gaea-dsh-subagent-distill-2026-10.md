# DSH 0.2.1 子代理板块蒸馏 —— 侦察与刀池（2026-10-10）

> **源（真身已核实）**：GitHub **deepseek-ai/deepseek-harness**（dsh，TS monorepo，Cordis 插件架构）。
> 本仓克隆：`clones/deepseek-harness` @ tag **dsh-v0.2.1-alpha.2**（= main HEAD，14988 文件，只读零搬运）。
> 前情：`docs/gaea-dsh-016-distill-2026-09.md` 基于 v0.1.6-alpha.2（源原在 `/c/AI/deepseek-harness`，现只剩 `.dsh`
> 运行态且被在跑 DSH 自主流水线进程锁定——**勿动该目录**，本轮起以 clones/ 为准）。
> 背景与时机：016 蒸馏留池④「子代理控制面（send_message/interrupt/cold-resume/委派深度记账）」当时判定
> 「上游 09-16/17 活跃大改，等其稳定」。v0.1.6→0.1.7→0.2.0→0.2.1 三跳后已收敛，且 0.2.x 做了**委派形态
> 的架构级重构**（前台/后台二分退役，统一受管 activation），正是补课窗口。
> 纪律：取道不取器；GoalCard 系已撤下勿再提；MCP/平台化/跨设备在册排除。

## 一、板块地图（v0.2.1）

| 包 | 职责 |
|---|---|
| `packages/subagent/subagent` | 核心服务：SubagentManager（启动/消息受理/冷恢复/生命周期）、activation 登记、深度记账（depth.ts）、父属目录与投影、结算消息 |
| `packages/subagent/tool-subagent` | 模型面委派工具 `subagent`：发射即返回 childId；完成由运行时以「结算通知」交付父模型 |
| `packages/subagent/tool-subagent-control` | 全局控制工具 `send_message`（邻接双向）/ `interrupt_agent`（祖先授权中断）/ `list_agents`（持久枚举） |
| `subagent-spawn-in-process` / `subagent-fork-in-process` | 本地 provider：新鲜子代理 / 父历史种子（= gaea 的 task 与 fork） |
| `subagent-claude-code` / `subagent-codex` / `subagent-acp` / `subagent-dsh-sdk` | 外部 provider：一次性执行（不可续消息）、进程外隔离 |
| 官方迁移指南 | `docs/upgrade-guide/v0.2.1-alpha.1/subagent-activations`（activation 统一）、`native-subagent-bundle-tools`（捆绑工具全局化） |

## 二、上游机制清单与判定

### 1. 统一受管 activation（0.2.x 形态重构）

`run_in_background` 参数与前台/Job 后台二分**整个删除**（升级指南 activations）：每次委派受理后立即返回
`{kind:'activation', subagentId}`，最终答案经「结算通知」（createSettlementMessage，continuation-messages.ts）
作为**持久的 user 消息**写进父会话，父模型在下一回合消费。结算通知带：一行结束概要（按 stopReason 措辞）
+ 子代理收尾全文 + structured 结果（JSON）+ diagnostic。

**判定**：形态二分是架构级取舍——gaea 的前台阻塞 task（同批 N 路并行、回合内闭环）+ run_in_background
jobs（跨回合，`DrainCompletedNote` 已把完成摘要注入下一回合模型输入，control/input.go:53）组合语义自洽，
全面异步化属大改，**留池不跟**。但其中两道是真增量：

- **stopReason 词汇表**：`completed / aborted / error / max-tokens / refusal`（merge-extensible），父模型据此
  分辨「正常完成/被中断/失败/写满/拒单」——gaea 现在只有成功文本或 error 字符串，被回合取消打断的子代理
  与真失败在父模型眼里无法区分。
- **diagnostic 安全详情纪律**：provider 写的失败详情强制 ≤4096 UTF-8 字节、不含工具输入/文件内容/环境值/
  凭据/裸协议载荷，与 output 分离呈现。gaea 的错误串（error 透传 Go err 链）无此卫生面约束。

### 2. send_message 邻接双向（tool-subagent-control/src/index.ts）

向「直接父或直接子」投递消息：运行中=下一**步界**注入；idle=起新回合；absent 的本地 continuable 子代理=
**从持久化冷恢复**后投递；返回送达回执（messageId）而非答案。归因强制分离：`agent-message`（模型自己写的）
vs `subagent-settled`（运行时的结算账目）异 kind——“transcript 把两者合并就是给子代理记它没说过的话”。

**判定**：真增量但**大改**——gaea 的 task 是「调用=运行到完成」，无常驻（resident）概念；idle-wake 与
冷恢复投递需要把子代理生命周期从 tool call 里拆出来。gaea 已有的近邻：SteerSubagent（v4.243，运行中
直穿改向，仅父→子）、continue_from（模型显式带 ref 续跑）、SubagentStore 持久化。**留池 P2**，真实需求
（长跑子代理中途纠偏/回报）出现时立项；届时归因分离纪律照单全收。

### 3. interrupt_agent（模型面中断）

`interrupt_agent(agent_id)`：祖先授权（exact live caller 的 recorded lineage 必须包含目标）或人类父地址；
fire-and-return——cancel 信号发出即返回，目标观察到信号才真正停；未认领 inbox 保留，被打断者 idle 后
唤醒 send 恢复 FIFO 队列；外部执行永久停止。

**判定**：**真增量 P0**。gaea 底座近全有：`subRunners`（ref→live AgentRunner 登记，task.go）只差一个
cancel 面；后台任务已有 `jobs.Manager.Kill(id)`（含 runningDescendants 级联）。补：runSubAgentInternal
为子会话派生可取消 ctx 并登记，新工具 `interrupt_agent(ref)` 取消之（前台）或转发 Kill（后台 job）。
gaea 单父谱系简单（ref 只在父子结果/卡片间可见），邻接授权≈ref 可见性，无需 dsh 的完整 lineage 校验。

### 4. list_agents（模型面持久枚举）

`list_agents({scope: children|descendants})`：读父属 subagentCatalog，行={id,label,status:running|inactive,
depth}；损坏/不可读产生 `corrupt/unavailable` 诊断行，只停该分支；外部行 mode:external 不可续。

**判定**：**真增量 P1**。gaea 的分工可见面全在前端轮询（GaeaSubagentRuns），**模型面没有枚举工具**——
模型只记得 task 结果尾行 `Subagent reference: sa_…` 的那些；多轮之后想 steer/continue 一个旧子代理时
没有入口。落点：新工具 `subagent_list`（读 SubagentStore meta 目录，ref/标题/状态/工具调用数，children
口径即可，descendants 无意义——gaea 一层深）。

### 5. 委派深度记账（depth.ts）

深度存在**会话头 delegationDepth 持久单调**：「runtime options 可能加深但永不能降浅——恢复的子代理带
全新 options，从零计它就会伪装成 top-level 再 delegate」；maxDepth 每次委派时对父当前深度校验；
数字帽要求 provider depthLimit 能力旗标，mount 期 fail-loud。

**判定**：gaea 用「注册表剔除元工具」表达**一层深不变式**（task/run_skill/install_skill 不进子代理注册
表；continue_from 续跑同样不回注 task 工具），不变式已安全。深度记账=可配置 N 层的能力，**真实需求待
现，留池 P2**；届时**单调持久教训必须随行**（gaea transcript 侧车需记 depth，续跑不得归零）。

### 6. 子代理模型路由 per-call

委派参数可带 provider/model/reasoning_effort；`list_subagent_models` 工具枚举可达路由；白名单策略
（per-session 采样+子会话继承）；**route preflight**（接受子代理前先验证 LLM 路由可达，避免收了单跑不了）。

**判定**：gaea 有 config `subagent_model`（全局）+ `subagent_models`（per-skill）；per-call 路由+白名单
属配置面扩张，**留池 P2** 等真实多模型路由需求。**route preflight 教训取道**：将来落地时受理前先 ping。

### 7. toolFilter：未知名 fail-loud

委派的工具过滤 allow/deny，落到子代理是 scoped `tools.restrict()`：**prompt 与执行一体不可见**（一个可见
性），**未知名响亮报错**；tool-subagent mount 时空白名单也 fail-loud（「配置了 toolFilter 却既无 allow 也无
deny=移除该键」，防 `allow: []` 默认值把所有工具静默清空）。

**判定**：白名单本身 gaea 已有（task `tools` 参数）且更早。**真增量小刀 P0**：gaea `FilterRegistry`
（task.go）对模型给的未知名**静默 `continue`**——模型在 tools 里写错一个名，子代理就无声少一个工具，
父模型无从得知。对齐上游改 fail-loud：构建子注册表时收集未知名，存在则在 Execute 前返回错误列出。
（注意 FilterRegistry 同时服务技能 AllowedTools——人写的技能清单错名同样该响；SubagentMetaTools 的
排除维持静默，那是设计不是过滤错名。）

### 8. 结算通知的持久归因分离

`agent-message` vs `subagent-settled` 异 kind 异 form（relay vs notice），显式防「运行时的账目被当成子代理
说的话」。

**判定**：gaea `SubagentMessage` 事件独立 kind=subagent_message 落账、旧日志读端跳过、且 ParentToolID
挂父 task 卡——**等价物已在，销号**。

### 9. 已蒸馏项复核（016 五刀+后续，上游仍同构）

- outputSchema 两阶段捕获/terminal guard（v4.380 刀2）↔ 上游 outputSchema/structured 捕获：同构，销号。
- fork 父历史种子（v4.380 刀1）↔ subagent-fork-in-process：同构；上游额外按 `inheritsParentContext`
  切换工具描述措辞（fork 子代理「已见已完成回合」，fresh 的「不见」）——gaea task schema 的 fork 描述已
  说明种子语义，够用，销号。
- spill（v4.379）↔ 上游 compaction 域，非子代理板块，不涉。

### 10. 不取（辨伪/在册排除）

- **外部 provider 生态**（claude-code/codex/acp/dsh-sdk 作为子代理后端、进程外隔离 out-of-process.ts）：
  平台化方向，在册排除。
- **per-child persona / cwd 捕获**：gaea 工作区模型不同（cwd=工作区根），无多 persona 需求前不做。
- **waitForChildren 父收尾 join**（进行中后代拖住 host 完成、inbox 停泊者不拖）：与全异步形态绑定，
  gaea 前台天然拖住回合、后台刻意不拖，语义自洽；P2-E 异步化时再评。

## 三、超前项（gaea 有、上游无）

- **有限并行闸**（8b9f1613：subagent_max_parallel 默认 3 帽 8 + queued 排队诚实化）：上游只有
  「shared capacity controller may delay an operation」一句抽象合约，无具体实现。gaea 独立增量。
- **子代理证据链 Journal 落账**（v4.221，写盘成证据卡按会话归因）：上游无对应物。
- **子代理直穿改向**（v4.243 SteerSubagent）：上游的运行中注入走 send_message 步界语义，gaea 队列式
  直穿是独立实现。

## 四、刀池（建议排序，全在现有底座上、零绑定变更风险低）

| 序 | 刀 | 上游机制 | 落点 | 规模 |
|---|---|---|---|---|
| P0-A | `interrupt_agent` 模型面中断 | §3 | task.go：runSubAgentInternal 派生可取消 ctx 挂 subRunners；新工具 interrupt_agent(ref)——live 前台取消 ctx、后台 job 转发 Manager.Kill；测试：取消后子会话停止+重复幂等+未知 ref 报错 | 中 |
| P0-B | FilterRegistry 未知名 fail-loud | §7 | task.go FilterRegistry：收集未知名非空则报错列出（申报行为变化：此前静默跳过）；SubagentMetaTools 排除路径不受影响 | 小 |
| P1-C | 结算语义强化 | §1 | task.go：子代理终态分类（完成/被取消/出错/max-tokens/拒答——gaea 侧判据：runErr 类型+ctx.Err()+输出）注入 task 结果头一行与 SubagentMessage 摘要；错误串卫生面（截 4096、剥离参数回显） | 中 |
| P1-D | `subagent_list` 模型面枚举 | §4 | 新工具读 SubagentStore meta（ref/标题/状态/调用数），配合 steer/continue_from 闭环；注册进父注册表、元工具剔除清单同步 | 中 |
| P2-E | send_message 邻接双向/idle-wake/冷恢复投递 | §2 | 需常驻概念重构，等长跑子代理真实需求；归因分离纪律随行 | 大 |
| P2-F | per-call 模型路由+route preflight | §6 | 等真实多模型路由需求 | 中 |
| P2-G | 委派深度记账（单调持久教训随行） | §5 | 等可配置 N 层需求 | 中 |

> 落地纪律提醒：P0-A/P1-D 是**新工具**（注册表面，非 Wails 绑定面），但前端 `tool_icons.ts`/`tools.ts`
> 的 subjectOf/summarize 需同步认领（不认领则走 Wrench 通用图标+空摘要，功能不坏）；P0-B 申报行为变化
> （模型给错名从静默变报错，属诚实化）；每刀独立可发版。
