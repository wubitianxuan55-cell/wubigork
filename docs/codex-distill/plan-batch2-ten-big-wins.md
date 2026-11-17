# 第二批计划 · 10 个关键性大优点（codex 0.165.0 → gaea）

> 第一批（`exec-brief-T1-T5.md`）是「子弹」：三档行匹配、环境注入、审批理由、`Never` 语义、schema 预算——
> 都是小改、可断言、不改变产品形态。
> **第二批是「结构」**：改的是 gaea 怎么组织上下文、怎么决定工具、怎么在无人值守下自主、怎么协作。
> 每项都写成 asset 文档（机制 / 常量 / 落点 / 验收 / 不照搬的部分），成熟后合并成施工单。
> 状态标记：✅ 已有 asset 文档｜🔍 需继续逆向｜🚧 逆向中

## 总览

| # | 大优点 | 一句话收益 | 量级 | 状态 |
|---|---|---|---|---|
| 1 | 上下文章节化 + 快照差分 | 长会话不再每轮重发世界状态 | **省 token 最大的一处** | ✅ asset-02 |
| 2 | 工具面预算 + 延迟展开 | 本地小模型也能挂很多工具 | 首轮 schema ↓ | ✅ hotspot-06 |
| 3 | Guardian 自动评审 | 无人值守时「机器先审、人后追认」 | **自主度上限** | ✅ hotspot-04 |
| 4 | 审批七态 + 策略修订 | 一次授权按前缀终身生效 | 中断次数 ↓ | ✅ hotspot-01 |
| 5 | 执行语义 + 长驻终端 | 长任务增量返回、输出可复现 | 稳定性 | ✅ hotspot-03/10 |
| 6 | 压缩三形态 + 生命周期 | 长稿写作成本 ↓、可复现 | 成本 | ✅ hotspot-10 |
| 7 | 记忆四护栏 | 记忆可信、可审、不烧配额 | **数据安全** | ✅ asset-03 |
| 8 | 技能 RRF 多路融合 | 技能多了也选得准、中文友好 | 命中率 | ✅ asset-01 |
| 9 | 上下文碎片契约 | 注入可剥离、可审计、可防御 | **结构地基** | ✅ asset-01 §2 |
| 10 | 多智能体消息板 + 图谱 | 众神殿从同步四席升到异步协作 | **产品形态** | 🔍 待逆向 |

---

## 1. 上下文章节化 + 快照差分（省 token 最大的一处）

- **机制**：把上下文拆成 18 个带**稳定 ID** 的章节（`agents_md` / `permissions` / `tools` / `top_level_tools` /
  `environments` / `context_window_guidance` / `approved_command_prefixes` …），每章节自带 `Snapshot`，
  每轮只渲染 `render_diff(previous)` 的**变化**；`ID` 持久化进 rollout，改了历史会话就认不出自己的章节。
- **收益**：长会话里世界状态从「每轮重发」变成「变化才发」。不需要模型配合，纯工程。
- **落点**：gaea `internal/gaea/contextview`（现为展示型）→ 加章节层；`prompt` 改为按章节装配。
- **前置**：无。**风险**：历史片段识别不清会出现「同一章节两份」，需要 `matches_legacy_fragment` 等价物。
- **三步走**：① 章节表 + 可观测（不动行为）② 快照 + diff ③ 每章节独立预算。

## 2. 工具面预算 + 延迟展开

- **机制**：共享**字节**预算；名字全额优先、描述轮次均分、超限 `...` 截断、预留 omission；
  配合 `tool_search` 延迟展开与「不可用工具占位」（调用被拒但模型看得见）。
- **收益**：工具数上去之后首轮 schema 不再线性膨胀——这是本地小模型（32k）能不能挂 40+ 工具的前提。
- **落点**：`internal/gaea/tool`（已有 `Hide/HideUnlessOnly/RemovePrefix/FilteredSchemas` + `CompactDescriptor`），
  加 `budget.go` 纯函数 + 检索接入点。
- **前置**：先量一次真实会话的首轮 schema token 做基线。**开关默认关闭**。

## 3. Guardian 自动评审（自主度上限）

- **机制**：动作级（不是工具级）评审——`Command / Execve / ApplyPatch / McpToolCall / NetworkAccess /
  RequestPermissions / WriteStdin` 七类；证据包带**出处**（用户消息 / 工具输出 / 继承消息 / harness 配置分源）；
  输出 `outcome / risk_level / user_authorization` 三档；**`None` 绝不等于放行**；三层预算（输入 / 请求 / 会话，
  `INPUT_TOKEN_MARGIN=256`）；评审在独立线程，失败不进 reuse 缓存；拒绝可事后追认。
- **收益**：夜间批处理能自己拍板低风险动作、把高风险留给人；同时是提示注入防御的结构性答案。
- **落点**：新增 `internal/gaea/guardian`（prompt/budget/decision）+ 复用 `internal/ai` + 接 `controller_approval`。
- **前置**：第 4 项（七态）先落地，否则「谁拒的、能不能追认」无处表达。**默认关闭**，按会话/任务开启。
- **三刀**：① 证据分源（不花模型钱）② 评审调用 + 预算 ③ 决策接线 + 追认。

## 4. 审批七态 + 策略修订

- **机制**：`Approved / ApprovedExecpolicyAmendment{proposed} / ApprovedForSession / ApprovedMcpPolicyAmendment /
  NetworkPolicyAmendment{...} / Denied{rejection} / TimedOut / Abort`；`ApprovedExecpolicyAmendment` 落
  `add_prefix_rule(cmd, Allow)` 并持久化；`Denied` 的理由回喂模型；`TimedOut` 是「无人应答」而非「用户拒绝」。
- **收益**：一次「允许并记住」按命令前缀终身生效，中断次数明显下降；夜间不再因为没人点按钮而死。
- **落点**：`control/controller_approval.go`（已有五决策 + glob 规则 + 自动放行账本，缺理由/超时/追认三处）。
- **前置**：无，最小的一刀。**验收**：拒绝必带理由进下一轮；超时不产生 deny 账本；追认可重放一次。

## 5. 执行语义 + 长驻终端

- **机制**：`yield_time_ms` + `max_output_tokens` 双闸门（先到先返回），`terminal_id` 为一等对象，
  `write_stdin` 续取增量；固定注入 10 个环境变量（`NO_COLOR/TERM=dumb/LC_*=C.UTF-8/PAGER,GIT_PAGER,GH_PAGER=cat/CODEX_CI=1`）；
  常量：`MAX_STDIN_APPROVAL_BYTES=8000`、`LATE_NETWORK_DENIAL_GRACE_PERIOD=100ms`、`POST_EXIT_CLOSE_WAIT_CAP=50ms`。
- **收益**：长命令能边跑边看（而不是等十分钟一次性返回）；输出可比 → 前缀缓存不漂；`git log` 类命令不再卡分页器。
- **落点**：`internal/gaea/proc`（纯新增 `env.go`）+ `agent/batch_executor.go` 接时间片。
- **前置**：无。**风险**：时间片语义会改变工具结果的时序契约，需要一次性对齐前端展示。

## 6. 压缩三形态 + 生命周期

- **机制**：local / remote / **token-budget**（跳过摘要、直接装新窗口）三形态；触发 `Manual`/`Auto`；
  无论哪种都必须发 `ContextCompaction(started/completed)` 条目并跑 pre/post hooks，hook 停则 `TurnAborted`。
- **收益**：长稿写作不必每次付一次摘要成本；UI / hook / 统计三处口径统一，不会因为「这种压缩没发事件」而分叉。
- **落点**：`agent/compact.go`（已有三入口 + 规划器 + `CompactionStarted/Done` + `PreCompact`），补新窗口形态 + `PostCompact`。

## 7. 记忆四护栏

- **机制**：① 抽取前 `redact_secrets` 脱敏（保留类型计数）② 限流时**跳过整理**并记 `skipped: rate_limited`
  ③ 合并写基线差分（删除 >10% / 修改 >30% 拦下等确认）④ 跨进程锁文件放在**记忆目录之外**（重置会换 inode）。
- **收益**：记忆从「黑箱自动写」变成「可信、可审、可控成本」。gaea 的数据结构已领先，缺的就是这圈护栏。
- **落点**：`memory/{distill,remember,store,file_backend}.go` + `eventlog.go` 记事件。

## 8. 技能 RRF 多路融合

- **机制**：`RRF_K=60`、候选截断 50、同分用「最好名次 → id」定序；字符 n-gram 全维硬上限
  （查询/文档 4 KiB、术语 64/256、gram 512、候选 1000、结果 50、gram 2–5 字符、字段权重 8/4/1）；
  多查询按连接词拆 ≤8 视图再合并，原查询名次优先。
- **收益**：技能多了仍选得准；**中文无需分词**；无向量依赖、可离线、结果确定。
- **落点**：`internal/gaea/skill` + 现成 `bm25` / `textsim`。

## 9. 上下文碎片契约（结构地基）

- **机制**：每个注入片段实现统一契约——`role` / `content_kind`（`<feature>.<name>` 稳定分类）/
  `markers()` 起止标记 / `body()`；归属**生产时绑定**，绝不从渲染文本反推；
  `matches_text` 用类型标记判定「这段是不是我」，供压缩与迁移识别。
- **收益**：三个老问题一次解决——压缩后重注入不干净、说不清「这段谁写的」、工具输出与用户指令无法区分。
- **落点**：`internal/gaea/prompt` + `contextview`；最小版只加标记对 + `Kind` 字段。
- **前置**：第 1 项（章节化）最好先行，两者天然互补。

## 10. 多智能体消息板 + 图谱（待逆向）

- **机制线索**：crate `agent-message-board` / `agent-graph-store` / `agent-roles`（discovery/loader）；
  工具面直接暴露成消息板 API：`create_channel / get_channels / list_threads / search_posts / read_thread /
  read_post / subscribe / unsubscribe / post`；`tools/handlers/multi_agents_v2/*`（spawn / followup_task /
  interrupt_agent / list_agents / message_tool / send_message / wait）。
- **收益**：gaea 的众神殿现在是**同步四席**，消息板能把它变成「异步协作 + 可回放」——这是产品形态级的差异。
- **下一步**：先读 `ext/agent-message-board/src/*` 与 `core/src/agent_message_board.rs`，
  搞清频道/帖子的数据结构与订阅语义，再判断值不值得做。
- **风险**：如果 gaea 的多智能体一直是「一次性并跑四席」，异步消息板属于过度设计——先用真实用例验证需求。

---

## 推进节奏（长期目标：`goal-ff3aca53…`，上限 30 轮）

- 每轮推进 1–2 项：**先读源码写 asset → 判定可移植性 → 同步到 `docs/codex-distill/`**；
- 逆向侧产物（证据、伪代码、源码树）**不进仓库**，只在 `C:\AI\re\codex-0.165.0`；
- asset 成熟（机制清楚 + 落点明确 + 验收可断言）时合并成施工单，施工单编号接在 T1–T5 之后（T6、T7…）。

## 总验收口径（四项，任何 asset 都必须能回答）

1. **可断言**：验收是「跑什么命令 / 看什么数字」，不是「感觉更好了」；
2. **可回退**：新能力默认关闭，关时与现状逐字节等价；
3. **可解释**：每个决策能说清依据（哪条规则 / 哪段证据 / 哪个章节）；
4. **不复制原文**：机制可照抄，提示词与文案必须自己写。
