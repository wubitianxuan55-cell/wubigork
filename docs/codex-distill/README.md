# codex 0.165.0 → gaea 蒸馏 · 计划与施工文档

> 本目录只放**计划与施工文档**（给人/执行 agent 看的）。
> **逆向原始资料不在仓库里**——证据包（264 万条串、crate 表、反编译伪代码、内嵌模型目录、
> 上游源码树）全在仓库外：`C:\AI\re\codex-0.165.0\`（约 136 MB，勿复制进仓库）。
> **跨轮次状态锚点**：`C:\AI\re\codex-0.165.0\docs\MEMORY-逆向状态.md`（进度、铁律、证据根、已确认机制、下一步）。
> **第二证据根（桌面端插件包，2026-11-19 起）**：`asset-62`（电脑操控）不是从 `codex-0.165.0` 源码逆的，
> 证据在 `%USERPROFILE%\.codex\zh-cn-patched\<hash>\app\resources\`（Electron 包解包 + `cua_node` 运行时 + `@oai/{sky,cua,cua-repl}`），
> 原始证据包（含实测 helper 响应 JSON、PE 导入表、全量串表）在仓库外 `C:\AI\_distill\codex-computer-use\`。

## 文件

| 文件 | 用途 |
|---|---|
| `batch2-execution-plan.md` | **第二批执行计划（v2）**：对齐第一批落地现状（T1/T2/T4 已落、T3 在改、T5 开半）、冲突面分析、三波排序（T6/T12/T14/T13 → T7/T8/T9 → T10/T11/T15）、去重表与门禁 |
| `batch3-execution-plan.md` | **第三批计划（前端落地 · 只出计划）**：F27–F46 二十篇施工卡收敛为 **34 条**（波 A 正确性 9 / 波 B 结构 10 / 波 C 功能 15）、10 条顺序约束、验收口径、开关回退矩阵、7 条风险、5 个拍板门、3 个 PR 组与工作量估算 |
| `master-brief-T6-T15.md` | **总施工单（第二批收口）**：T6–T15 三批排期、与 T1–T5 的去重归并、每项硬验收与回退开关、共同纪律与门禁命令、汇报格式 |
| `exec-brief-T1-T5.md` | **施工单**：T1 三档行匹配 / T2 执行环境注入 / T3 审批拒绝理由+超时挂起 / T4 `Never` 语义+Windows 危险命令 / T5 工具 schema 字节预算。每项含现状 file:line、改动函数签名、验收断言、禁区、门禁命令、回报格式 |
| `landing-checklist-v3.md` | 落地清单 v3：机制事实 → gaea 落点 → 修正后的规模 → 口径风险 6 条 → 建议顺序 |
| `asset-01-skill-selection-context-fragments.md` | 资产：技能选择实验族（RRF k=60、字符 n-gram 硬上限、多查询拆分）+ 上下文碎片契约（标记对/归属/稳定分类） |
| `asset-02-world-state-context-assembly.md` | 资产：回合上下文装配（18 个章节 + 快照 diff 渲染 + 独立预算），对应 gaea `contextview` 的改造三档 |
| `asset-03-memory-pipeline.md` | 资产：记忆管线四条护栏（抽取前脱敏 / 背压门 / 合并基线差分 / 跨进程锁位置语义），含与 gaea memory 包的逐项对照 |
| `asset-04-message-board.md` | 资产：多智能体消息板（Post/Thread/Channel/Subscription 结构、权限「reply 不蕴含 post」、预览正文分离+字符偏移续读、拒绝错误预算化），含众神殿三档落地建议 |
| `asset-05-hooks.md` | 资产：Hooks 14 事件 + 4 执行器（Command/McpTool/Prompt/Agent）+ Sync/Async + 运行观测 + 插件指纹信任；含 gaea 三档落地 |
| `asset-06-agent-roles-and-graph.md` | 资产：智能体角色数据化（description/config_layer/昵称候选）+ 生成拓扑四不变式；含第 10 项收口结论（消息板值得借、图谱不搬） |
| `asset-07-guardian-implementable.md` | 资产：Guardian 施工级——输出 JSON Schema（仅 outcome 必填）、三层容错解析与默认值、错误与风险分家、超时可取消、拒绝熔断常量、三刀施工与验收 |
| `asset-08-section-implementation.md` | 资产：章节化施工细节——section 五成员、render_diff 三种返回、替换/移除通知纪律、`tools` 与 `top_level_tools` 分工、Go 最小模板与五个单测 |
| `asset-09-compaction-implementable.md` | 资产：压缩施工级——四种触发（含 ModelDownshift / CompHashChanged）、三形态、新窗口保留白名单与 64k 预算、`input_goal_ids` 继承、hook 否决回退，含 gaea 四件补齐 |
| `asset-10-exec-semantics.md` | 资产：执行语义施工级——pending 消费式 vs transcript 累积式双缓冲（不重复输出的关键）、头尾 50/50 保留 + 省略标记、双闸门与 50ms 收尾上限、终端生命周期与 chunk_id |
| `asset-11-tool-surface-budget.md` | 资产：工具面预算三层——目录差分（name→hash，不变即零输出）、检索展开（BM25，limit=8，只物化选中）、字节预算兜底；含 gaea 三步施工与验收 |
| `asset-12-fragment-contract.md` | 资产：上下文碎片契约施工级——role/kind/markers/provenance 四组成、归属七类（含「工具上下文即使框架代写也归 Tool」）、渲染与兼容规则、Go 契约与第 1/3/6 三项接线 |
| `asset-13-approval-implementable.md` | 资产：审批七态施工级——规则文件格式（prefix_rule + 首 token 索引）、glob 与 argv 前缀的粒度差异（`git status*` 会误放行 `; rm -rf`）、作用域三态、TimedOut/Abort 区分、追认 |
| `asset-14-memory-guardrails.md` | 资产：记忆四护栏施工级——脱敏四条正则（原文+替换顺序+保留键名）、三道前置门（含限流查询失败按通过）、基线差分与有界渲染、产物校验三要点、保留期 14 天；含 Go 版代码与验收 |
| `asset-15-skill-selection-rrf.md` | 资产：技能选择施工级——RRF 融合 90 行（K=60、三级 tie-break、截断标志合并）、整数权重打分表（256/128/64/24/16/6/4/1 + 覆盖词数平方加成）、字符 n-gram 与全维硬上限 |
| `asset-16-realtime-voice.md` | 资产（评估）：实时语音会话——云端双工判定不做，但借三条交接机制（转写落地为真回合 / 标记化委托片段 / XML 转义+有界保留）与全部硬常量 |
| `asset-17-plugins.md` | 资产：插件生态——插件是「资源包」不是代码（skills/mcp/hooks/apps + interface 元数据）、`LoadedPlugin` 运行时语义（包级与技能级两级禁用）、版本段防穿越校验；明确不做云端市场 |
| `asset-18-agent-identity.md` | 资产（判定不做）：跨进程身份——云端 Ed25519+JWT+JWKS 注册体系，本地单用户无相应；仅记三条小实践（派生上下文串、audience/issuer 钉死、重试分类） |
| `asset-19-network-proxy-deep.md` | 资产：出网代理深入——体积预算「槽位租约」（8×16MiB 入/出，内存上界钉死）、凭据代理与 MITM（不做）、连接归因魔数帧；含 gaea 两条小改与验收 |
| `asset-20-code-mode.md` | 资产（判定不做）：code-mode —— V8 脚本编排（一次工具调用编排多次），记下三条通用实践：危险全局删除清单、取消优先+独立超时、错误 512B 上限 |
| `asset-21-exec-server.md` | 资产：执行服务协议——网络决策三态 Allow/Deny/Ask + 审计一体（source/scope/override + host 253/reason 1024 限额）、能力发现快照（指针比较复用）；进程拆分层明确不做 |
| `asset-22-rollout-storage.md` | 资产：会话记录存储——只压 7 天冷数据、锁可陈旧接管（6h）、并发 2 与单任务 5h 上限、压缩率打点；含 gaea（SQLite）的「冷会话归档→校验→删行→VACUUM」适配方案 |
| `asset-23-hook-output-contract.md` | 资产：Hooks 输出契约——JSON 决策替代退出码（allow/deny/ask + updatedInput + additionalContext）、两套决策词汇与优先级、匹配器三级编译（通配/多值/正则）、「能发现≠能启用」的信任规则 |
| `asset-24-memory-citations-usage.md` | 资产：记忆引用与使用反馈——codex 的 citation 块（路径:行号+note）与「用 shell 读命令推断使用」；对照结论：gaea 已有 citation 闭环与注入评测（更完整），只需补定位精度与版本绑定 |
| `asset-25-api-errors-and-rate-limits.md` | 资产：API 层错误分类与限流——18 个错误变体（策略类单列、窗口超限与配额超限分开）、`retry_after` 结构化、code×status 双重校验、错误单一映射点；含 gaea 三条核对项 |
| `asset-26-responses-shape.md` | 资产：Responses 请求体与流式事件——字段顺序即接口（路由字段前置）、15 个事件变体分五类、`end_turn` 防 None / 实际模型认服务端 / 推理 token 不重复计的语义；判定不切协议只借语义 |
| `plan-batch2-ten-big-wins.md` | **第二批计划**：10 个关键性大优点（上下文章节化+差分 / 工具面预算 / Guardian / 审批七态+策略修订 / 执行语义 / 压缩三形态 / 记忆四护栏 / 技能 RRF / 碎片契约 / 多智能体消息板），含推进节奏与总验收口径 |
| **前端批次 F（2026-11-17，只出文档、不改代码）** | |
| `frontend-distill-plan.md` | **前端总清单**：codex tui 分层 → gaea 组件映射、七篇索引、波次依赖、与 T1–T15 去重、核出的 gaea 现存缺陷（§6.5）、跨篇收敛的 6 个真缺口与优先级 |
| `asset-27-streaming-render-two-region.md` | 资产：流式渲染双区——`emitted ≤ enqueued ≤ render.lines.len()` 不变量、提交动画队列（120fps 上限、8行/120ms 两档齿轮）、表格三态 holdback、代码围栏快路径与 4KB/512KB/10000 行限额、8192B 预览上限 |
| `asset-28-transcript-view-interaction.md` | 资产：转录视图——条目模型与 consolidation（流式 cell → 源驱动 cell 不重编号）、逻辑行/物理行/制表符三分坐标、resize 以 cells 为真源重排、折叠披露（按"保留的活动身份"）、搜索增量扫描与页脚状态机、选区三档、跟随/未读/书签、prompt header 与分页浮层 |
| `asset-29-markdown-render-and-copy.md` | 资产：Markdown——源表渲染、键值表、行内指令（流式生效、path 编码、掩码保偏移）、本地/网页链接、列表间距、数学、Mermaid、表格复制成 GFM 源表与转义集、两类硬上限 |
| `asset-30-diff-render.md` | 资产：Diff——无中段折叠（卡片层 3 行预览）、正文硬换行不截断、单列行号与全补丁预扫描列宽、512KiB/10000 行/4KiB 整块退让、hunk 级高亮；gaea 三处阈值互相架空 |
| `asset-31-bottom-pane-and-overlays.md` | 资产：输入面板——收键阶梯（warnings → questions → 栈顶 → banner → composer）、视图栈不变量、审批覆盖层默认焦点≠默认执行、LIFO 队列倒序 drain、粘贴突发阈值（3 字符/8ms/120ms） |
| `asset-32-keymap-system.md` | 资产：键位系统——上下文位集 × (context,action) 身份、单键表 + 独立和弦层（1s 过期丢弃首键、leader 常驻、1024 上限）、五道加载期冲突闸、帮助/HUD/pending 同源 |
| `asset-33-status-motion-clipboard.md` | 资产（判定为主）：状态卡字段表、限额按"剩余"口径、动效偏好启动播种+JS 显式降级、帧率上限、剪贴板 text/plain+text/html 双表示、视图切换按键守卫 |
| **前端波 2（2026-11-17，只出文档、不改代码）——消息流之外的四个面** | |
| `frontend-distill-plan-wave2.md` | **波 2 清单**：为什么波 1 只覆盖了消息流、卡片/输入/会话/回退四面索引、跨篇收敛的 5 条真缺口与优先级、与波 1 及 T6–T15 的去重、"判定不搬"清单、波 3 待立项 |
| `asset-34-tool-exec-card-family.md` | 资产：工具/执行卡片族——`HistoryCell` 七投影契约 + 四条旁路、高度用实际换行行数测、语义探索组（read/list/search）与 call_id 路由失配即报错、live 输出三层预算（1MiB/50 行/单行头尾 5KB）、`ActivityDisclosure::OutputLines(hidden)` 量化披露、状态词派生 |
| `asset-35-input-popup-candidates.md` | 资产：输入弹层——统一 @ 弹层 + 三档检索模式、类型优先排序（Task 同类保持原序）、异步三元守卫（pending/display/waiting）、Esc 身份三元组（query+token+occurrence）、Tab 六道判据、斜杠 exact→prefix 两段与别名隐藏 |
| `asset-36-session-surface.md` | 资产：会话面——游标分页 + 预取阈值 + `seen_key` 去重、边搜边续页与 search token、转录预览独立三态（先画 loading 再顶替）、名字解析同名必报歧义跨页不确定拒绝猜、归档三态且检查键位占用 |
| `asset-37-backtrack-and-hint-single-source.md` | 资产：回退与提示——Esc 阶梯 + 草稿非空守卫 + 两段式防误触、回退判据（回合边界/历史完整/非在飞/非隐藏）与失败回填 composer、提示纪律（不切开/3→2→1 换档/度量与渲染同源）、footer 提示优先级阶梯 |
| **前端波 3（2026-11-17，只出文档、不改代码）——清扫收尾：从"消息流之外"扩到全部前端模块** | |
| `frontend-coverage-ledger.md` | **全量覆盖台账**：`tui/src` 160 个顶层文件 + 24 个子目录逐个钉到"覆盖篇 / 旁证 / 不搬"，含判定分布与不搬六类理由、维护约定。核对"前端已逆完"看这里 |
| `asset-38-multi-agent-surfaces.md` | 资产：多智能体面——四组状态机（`SystemError→NeedsYou`）、徽标五态、分组键用 git 身份、刷新回落+删除环形回卷+`usize::MAX` 假行、空态五分支、agent picker 与子代理卡 preview grapheme 上限 |
| `asset-39-settings-theme-identity.md` | 资产：设置面——五段分区与 `ALL` 单一来源、配置项五元组含 `keywords`、`DependentToggle` 的 `Unavailable` 不隐藏、主题实时预览（一删一增保底）、可访问性偏好由系统注入、多选器契约、动作派生文案 |
| `asset-40-usage-analytics-and-status-cards.md` | 资产：用量与统计——状态卡可复制/列宽预留、限额条 20 段 + 15 分钟陈旧阈值、分档固定顺序数组、统计三段分离（数据形状→归一化→渲染）、`units`/`data_freshness` 与数据同框、热力图 52×7 常量 |
| `asset-41-onboarding-and-trust.md` | 资产：首启与信任——步骤状态自报（`Hidden` 不计分母）、取消给出路、信任文案四象限、远程公告硬边界（256/2048/禁控制字符）、启动草稿恢复四阶段、本人验证三态 |
| `asset-42-integrations-elicitation-and-hooks.md` | 资产：集成面——schema 驱动表单（空 schema 两种含义）、审批作为注入字段 `__approval` 五值、hooks 两页浏览、记忆双开关、连接器鉴权结构化失败、反馈受众分流 |
| `asset-43-worktree-and-environment.md` | 资产：工作树与环境——`Owner` 四态与动作守卫、工作树工具化、目标落文件 + 引用句（长内容不占上下文）、Git 摘要缺字段即省略、PR 只收 open |
| `asset-44-overlays-pager-and-text-metrics.md` | 资产：浮层与度量——一个翻页器三种内容、行索引滚动、重放上限的决策逻辑、按显示宽度截断且保留样式、`max_width==0` 返空 |
| `asset-45-motion-tips-and-visuals.md` | 资产：动效与视觉件——Reduced 换静态符号（非隐藏）、延迟期不显示、系统偏好只读一次且不改配置、空态动画白名单、内联可视化沙箱前置 + 2MiB + 伪域名占位、提示外置 |
| **后端覆盖台账（2026-11-17，只出文档）** | |
| `backend-coverage-ledger.md` | **后端全量台账**：`codex-rs` 122 个 crate 逐个钉到「覆盖 / 部分 / 未覆盖 / 不搬」，按九大功能域分组给含测试 LOC、判定分布、真缺口 Top 12（B1–B12）与八条实测机制速记。核对"后端已逆完"看这里 |
| `plan-batch4-backend-wave.md` | **第四批计划（B 批 · 后端波，研究视图）**：B1–B12 分三波（编排骨架 4 / 存储扩展 4 / 网络观测执行 4）、与 T1–T15 及 F 批的六处去重切法、六条顺序约束、验收口径、五条风险、3 个拍板门 |
| `plan-batch4-landing-priorities.md` | **第四批落地优先级（决策视图，2026-11-17 整理）**：三道闸（可感知 / 可度量 / 成本≤收益）+ 三条反例 → 12 项收益矩阵 → 只留 9 项：**Wave 1 五件约 4.5 天**（B1 关闭语义 · B12 截断口径修 UTF-8 真 bug · B5 工具归因闭合 · B9 出网安全 · B7 构造即验证）、Wave 2 四件约 6.5 天（step 快照 / 持久化守卫 / 观测最小集 / panic 隔离）、Wave 3 一件；**明确归档 8 项不做**（daemon、远程执行、插件市场、EMA、codex-mcp 服务端、otel 全套、trace bundle、抄实现） |
| `plan-batch3-landing-review.md` | **第三批复核（决策视图，2026-11-17）**：34 条 → **11 条 / ≈12 人日**。四条纠错：① 双壳是工期乘数而非结构项（实测 **308 个同名双份**，文学壳 `ApprovalModal.tsx:41` 同样有 `"3"` bug）→ **单源化前置**；② A6 已落地（`markdownCut.ts` v4.368）；③ 波 C 12 条是产品功能 → 转 backlog；④ B8 与第四批 P2 必须同源。含逐条判定（留/并/延/背/档）、三波重排、3 个拍板门、自查命令 |
| `asset-47-turn-admission-and-drain.md` | 资产（施工级）：**回合准入与优雅关闭**（B1）——`admit()`/`begin_drain()` 同锁、「在跑回合 + 已准入请求」两计数同零才退、连接闸门三态（关闸后排队不启动 / 已启动跑完 / 换账号即关闭）、30s 连接收尾上限、按资源串行化队列（Exclusive/SharedRead + 丢弃已断连接项）、per-thread 派生状态与两枚 active flag；含 gaea 四处对照、三刀施工卡、8 条断言 |
| `asset-48-output-truncation-and-fuzzy.md` | 资产（施工级）：**输出截断口径与模糊打分**（B12）——`TruncationPolicy=Bytes|Tokens` 4:1 互折、头尾各半、两种 marker 语义（含/不含预算）、1.2 序列化余量只乘一次、MCP 结果迭代缩放（JSON 转义膨胀）、多模态预算行走与省略可见、子序列模糊打分与前缀奖励；含 gaea 四套截断口径对照、两刀施工卡、7 条断言 |
| `asset-49-turn-state-machine.md` | 资产（施工级）：**回合状态机**（B4）——step 0 归属 `run_turn`（压缩失败先保输入再返回）、step 设置是不可变快照 + 运行中走受限更新（锁外预检/锁内复检、`Arc::ptr_eq` 目标确认、十项禁止变化、`Applied/Rejected/TargetUnavailable` 三态）、输入双通道（回合内 + 会话邮箱，先订阅再检查、插话即取消采样、`trigger_turn` 决定是否多采一次）、预算提醒 claim 一次性、重放只认"最新可用压缩点"与三态清除段；含 gaea 六维对照、三刀施工卡、9 条断言 |
| `asset-53-http-client-and-model-supply.md` | 资产（施工级）：**HTTP 客户端与模型供给**（B9）——路由感知连接池（≤16 路由、构造互斥、per-hop 重定向决策）、出网策略 permit 四态（`Unavailable/Destination/Revoked/UnsupportedTransport`）、TLS 后端只回退一次、`models_cache.json` + 300s TTL、SSE 头部权威元数据与限流多快照；结论：gaea 的重试层更成熟不用搬，只搬三件 |
| `asset-54-observability-three-pipelines.md` | 资产（施工级）：**观测三件**（B10）——analytics 事实流（队列 256/10s/flush 25s/去重 4096 + reducer 五类上限）、otel（meter `codex`、缓冲 256/1KiB/16 tags、enable/disable/**suspend** 三态与逐指标同意）、rollout-trace（bundle 四件套 sched 三处 v2、复用 protocol 事件、禁用零成本） |
| `asset-55-transport-and-daemon.md` | 资产（施工级 · 判定不搬实现）：**传输与守护进程**（B2）——四传输矩阵、有界通道 128、WS 出站 32K + 编译期断言、解压 64MiB、控制 socket `0o600` + 启动锁 + 恢复文件、运维动作全带超时；只借五条常量化纪律 |
| `asset-56-exec-server-implementation.md` | 资产（施工级 · 判定不搬实现）：**执行服务实现层**（B11）——入站大小按消息种类分档、pong 看门狗、序号永不回绕、帧上限先于解析、退避 1s→30s 且稳定 120s 才重置；含六条断言与"配置成功 ≠ 沙箱生效"的探测立场 |
| `asset-57-plugin-marketplace.md` | 资产（施工级 · 视拍板）：**插件市场与远程分发**（B7）——四道闸（清单先语法后解析 / 来源三态白名单 / 构造即验证 fail-closed / 同步只做一次 + 120s 上限）、容量纪律（3h 目录、8 条已加载、256 探测并发） |
| `asset-58-mcp-both-ends.md` | 资产（施工级）：**MCP 双端**（B8）——客户端四传输 + 适配器组合、OAuth 发现 5s、EMA 16 服务器/10 跳/30s、elicitation 三方法族 + 提前取消上限、服务端工具目录缓存 32/30min 且键含身份指纹、启动期 deadline 解耦 |
| `asset-50-tool-registry-and-dispatch.md` | 资产（施工级）：**工具注册与派发**（B5）——审批插在 `ToolOrchestrator::run` 第一段、沙箱首试被拒后**二次审批**；"注册 ≠ 可见"只隔 `exposure` 一个字段（`Hidden` 仍可派发但强制不可并行）；并发 = 工具自声明 `supports_parallel` + 读写闸门 + `FuturesOrdered` 按流序；归因是 best-effort；对照出 gaea `auditFunc` 第二参恒空、无 `call_id`/回合号 |
| `asset-51-extension-framework.md` | 资产（施工级）：**扩展框架**（B6）——没有 `trait Extension`，只有 16 槽位注册表（14 contributor 向量 + event_sink + turn_start_admission）；扩展**结构上没有失败能力**（回调返回 `()`，仅 3 槽可否定）；只能新增工具与观测、不能改他人 schema 或拦截调用；隔离靠"不提供能力"（tokio 仅 time，无网络无进程）；gaea 无扩展点 |
| `asset-52-thread-store-and-state.md` | 资产（施工级）：**线程存储与状态库**（B3）——权威源 + "允许滞后禁止超前"一致性规则、投影游标与守卫、分页游标形状、分段去重、回滚三件套分工、revert 的"旧文件不动 + 指针 CAS"、页回收常量组、陈旧锁接管判据 |
| `asset-59-backend-roundup-small-crates.md` | 资产（收口）：**后端余项小件合集**——5 条口径（历史批量 128/64KiB、保留上下文族 8/16KiB/64KiB、git 5s 超时 + 禁 hooks、PTY 输出 1MiB、批次只记最后完成者）+ 判定不搬三类理由；使台账"未覆盖"清零 |
| `asset-60-app-server-dispatch-and-redaction.md` | 资产（施工级）：**app-server 派发/订阅/脱敏**——三段式派发（连接前置 → 调度准入 → 域处理器）、线程解订阅三态（`NotLoaded`/`Unsubscribed`/幂等）、恢复响应按客户端名白名单脱敏（只改响应、`ImageGeneration` 整项丢弃）、接口级上限（sample 4KB/64KiB、扩展告警 256B） |
| `asset-61-config-and-agent-runtime.md` | 资产（施工级）：**配置层与多智能体运行面**——默认值即口径（200 项 / 10MiB / 65% / 30s / 三处提示词各 2000B）、`toml_edit` 保注释改写、托管层优先、子代理三条资源上限（关停失败 64 / 根上下文 3 / 授权消息 16）、静态名称词表 |
| `asset-62-computer-use-sky-windows.md` | 资产（施工级 · **新面**）：**电脑操控（Computer Use / `@oai/sky`）Windows 全链路**（B9）——三层结构（插件 + JS REPL + 原生 swift helper）、行 JSON 与 native pipe 双传输 + 超时表（10s / 15s）、17 个方法、UIA 树文本格式（含 75/73/163 截断几何）、索引与 `screenshotId` 失效闸门、WGC 截图与 DPI 复原失败即禁用坐标、按 app 授权闸门与四档确认策略、Escape→130 中断链路；含 gaea 三档落点与 10 条验收断言。**证据源是桌面端插件包，不在 `codex-0.165.0` 源码树里** |
| `asset-63-computer-use-full-reverse.md` | 资产（**代码级还原清单**）：电脑操控六层全量逆向（B9 续）——Electron 前端（asar 19 791 文件的 offset 表、主进程 computer-use 桥/`cua_repl` env 组装/overlay 配置下发、renderer 设置与审批键族）、`@oai/{sky,cua,cua-repl}` 全量 dist、原生 helper 的 5 134 条 Swift 反射字段与 UIA/WGC 状态机、启动器反汇编、自建 capstone xref 反汇编法（无需 Ghidra）；含还原度诚实清单与复现命令 |
| `asset-46-updates-notifications-daemon.md` | 资产：更新与运行面——更新动作由安装上下文八选一、版本不可比返回 `None`、`Older`/`Different` 分家、通知后端能力探测并回报生效项、恢复只做一次不循环 |

## 读到哪一层

- 只想知道「改哪里、怎么验收」→ 只看 `exec-brief-T1-T5.md`。
- 想知道「为什么这么改、codex 原机制是什么」→ 看 `landing-checklist-v3.md` 与对应资产文档。
- **只做前端** → 先看 `frontend-distill-plan.md` §2.6（收敛出的 6 个真缺口 + 优先级），再按项翻对应 `asset-27…33`；
  若要看"消息流之外"（工具卡 / 输入弹层 / 会话列表 / Esc 回退），走 `frontend-distill-plan-wave2.md` §3，再翻 `asset-34…37`；
  **要确认"前端是否已全部逆完" → 直接看 `frontend-coverage-ledger.md`**（160 个顶层文件逐个标注覆盖篇/旁证/不搬），
  波 3 的九篇（`asset-38…46`）补齐了多智能体、设置、统计、首启、集成、工作树、浮层、动效、更新运行九个面；
  前端证据以上游源码树为准（`C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs\tui\src`），二进制的 `crate-map.txt`
  只能证明 `tui` crate 的文件清单（371 个），不能证明前端行为。
- 想溯源到二进制证据（字面量偏移、函数 VA、伪代码、源码 tag）→ 去仓库外 `C:\AI\re\codex-0.165.0\docs\hotspot-*.md`，
  或者直接看 `work/decomp/*.c`。

## 三条纪律（施工时必须守）

1. 每项任务**先跑基线测试记数字**，改完只许持平或增加用例；不为过测试放宽生产逻辑。
2. 新能力**默认关闭**（T5 的 schema 预算、将来的 Guardian），关时行为与现状逐字节等价。
3. **机制可以照抄，文案必须自己写**——不要把握取出来的提示词原文搬进产品。

## 上游对应版本

`openai/codex` tag `rust-v0.165.0`（commit `e374836b`），与本机安装的 `@openai/codex@0.165.0` 二进制同版本。


























