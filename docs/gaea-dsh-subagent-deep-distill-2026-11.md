# DSH 0.2.2-alpha.2 子代理板块深挖蒸馏 —— 机制级（2026-11-02）

> **源**：`clones/deepseek-harness`，机制面按 **dsh-v0.2.2-alpha.2** 判定（tag diff 单独注明）；工作树检出
> 停在 alpha.2，两版差异点以 tag diff 引用。子代理包 src 全量 6360 行：
> manager(1475)/index(646)/tool-subagent(491)/types(366)/child-agent(290)/descriptor(269)/out-of-process(218)/
> model-selection(196)/projection(190)/activation(186)/list-agents(178)/catalog(170)/lifecycle(169)/
> continuation-messages(168)/structured(155)/control-types(149)/list-children(133)/list-models(113)/
> control(112)/internal(110)/depth(52)。
> **与前档关系**：docs/gaea-dsh-subagent-distill-2026-10.md=判定级+刀池（P0/P1 已落地 e19543bb/f5aa2892，
> P2 在册）；本档=机制级深读，逐条给「机制+纪律+gaea 判定」。纪律：取道不取器。
> gaea 侧对证：task.go:1161 结算路径/subagentStopReason 词汇/list_tool.go 模型面枚举/批 1 并行闸（均已读）。

## 一、形态总图（重申，不翻案）

上游=durable child Session + **驻留纪元（Activation）** + SubagentManager 全权生命周期；provider 只贡献
DATA（ContinuableCreateSpec「DATA, never a capability」——provider 无从看见 Agent/turn/teardown）；
外部 provider=一次性执行不可续聊（requireLocalActivation 抛 NOT_CONTINUABLE）。gaea=前台阻塞 task
（运行到完成）+ jobs 后台二分+sidecar meta。**形态判定维持 2026-10 档：不跟**。本档全部价值在机制纪律。

## 二、机制清单与判定

### A. 生命周期与结算

**A1 结算准入四态（settlementState）**：closed（已关）→ retry（poke 已被取代=观察过期）→ wait
（**ownedChildren 非空：孩子活着父不得结算**；或 inbox 有待跑 turn/step 且无新鲜失败）→ ready。
判定：**「父等子全落地才结算」是并发编排的硬纪律**——gaea 前台阻塞形态天然满足（同批 join），后台 job
侧 completion 注记也无孤儿面。销号（等价已足），留档给将来任何异步化。

**A2 结算竞速双闸**：ready 后 flushFinalState → ChildLock 内**复查 seq 稳定性**（session.seq 变了=有写
竞入，retry）→ runMaintenance 占 idle 相位才能 close（别人已占=retry）。判定：flush-复查-占相位三步
是「结算不可撕裂」的通用配方；gaea 阻塞形态无此竞速面。销号留档。

**A3 结算投递三路**：parent closing→`inject`（不唤醒）；idle→`queue`（新回合）；running→`steer`（下一
步界）。**投递尽力而为**（失败只 log 不抛）——父模型的结算通知是通知，**result promise 才是调用方权威
通道**。判定：双通道权威分离纪律。gaea 无异步结算面，销号；将来 send_message 立项时照单。

**A4 announced 诚实边界**：materialization 回滚（首次受理前）=「调用方被告知不存在的孩子」→ teardown
**不欠结算账**（不投通知）。判定：记账边界=「公布过才欠账」。gaea 排队态已做等价诚实（queued 不算
running）。销号。

**A5 失败活性（failureAt）**：只记**未被后续 committed turn/end 覆盖的失败**（失败后孩子自愈成功→失败
撤销）；**记录的失败赢过取消**——停一个已经失败的孩子，不把失败改写成取消。判定：**失败赢过取消**
是 stopReason 分类的补充纪律，gaea classifySubagentStop（ctx.Err 判据）在「cancel 与失败竞速」极小窗口
可能把 error 读成 interrupted——上游判法=先查记录的 turn 终态再查取消信号。**小刀候选 SA-C**（把
classifySubagentStop 改为先看 runErr 非 cancel 类则归 error 的次序，一行级）。

**A6 epochStopReason=子代理自己的日志为权威**：「teardown 成功说明不了模型是否出错——从处置推原因会把
失败报成完成」。折叠规则：max-tokens→max-tokens；aborted/interrupted→aborted；error→error；
**blocked（hook deny/政策 pre-step 拒绝）→refusal**（「活被拒不是活被做」，父不得当完成）；completed/无
turn→droppedUnrun?aborted:completed；**未名原因→error 绝不静默成功**。判定：gaea 3 词汇（宁缺勿虚报）
同判；「拒绝」语义 gaea 无 face（无 hook deny）。留档：若将来加 pre-step 政策闸，refusal 随行。

**A7 输出选择**：最后一条**非空** assistant 消息（usage-only 空内容消息跳过）→无则累积流→再无则 []。
completed 但声明过 outputSchema 而无 capture→**降级 error**（答应过结构化没交=失败）。判定：gaea
extractLastAssistantMessage 同构；降级纪律随 SA-A（结构化出口）走。

### B. 持久化与恢复

**B1 descriptor v3=版本化显式快照**：`subagent/descriptor` 会话事件（log-only：不进模型历史、存活于
压缩）在子首回合内追加**一次**；显式字段快照而**非整个 merge-extensible AgentOptions 对象**——「无关扩
展值不得让续跑只因不是 JSON 而失败」；新增组合输入=刻意的版本变更，绝不做隐式多余字段；schema 逐字段
校验+未知键拒绝；**首个 descriptor 事件权威**（后来者不可改写声明组合）。**刻意省略** subagentDepth
（信任持久 header 单调地板）与 outputSchema/maxTokens（一次性激活旋钮不入持久组合——冷恢复既不还原旧
预算也不继承父当前预算，用恢复路由的默认）。判定：**gaea SubagentStore meta 侧车已是等价物**（runs 表
+ref/状态/标题），版本化+「显式快照不打包 options」纪律值得对照——gaea meta 若将来加字段，走版本号
不要裸加。留档。

**B2 深度单调地板公式**：`max(header.delegationDepth ?? 0, runtime.subagentDepth ?? 0)`——运行时选项
**只可加深不可变浅**，「恢复的孩子带着新鲜选项从零计数就会冒充顶层委派」。gaea 深度记账已销号（无递
归面，2026-10-29 盘点）；**嵌套若开，此公式即实现**。留档。

**B3 会话 meta 持久五件**：cwd/agentPreset（**从父 LIVE scope 链读而非 header**——空跑中换过预设的父以
新组合委派，header 还写旧名；不记录则冷读解析部署默认、用孩子从没有过的工具集重建回合）/parentSession/
isSeeded/delegationDepth（durable）。origin:'subagent' 仅导航分类，**descriptor 才是模式与续跑能力的权威**
。判定：gaea meta 侧车等价；「LIVE scope 权威于 header」纪律对照 gaea 无预设系统，销号。

**B4 委派政策捕获三纪律**：①**首个 await 前同步捕获**（「更晚的父切换属于父的未来」）；②只捕父的**显式
override**（部署默认/一次性授予永不传）；③approval 钉 `never`（**子代理在委派时刻定死的沙箱范围内行动，
其 ask 被确定性拒绝**，无论父自己什么政策）。落盘=source:'delegation' 事件且**排在 fork seed 之后**（新
政策赢陈旧种子）。判定：gaea S3 防穿越（空间面 fail-closed）已占同位；「显式才传+钉死审批」纪律与
V5 提案信任洞（Approver=nil 子代理审批全放行）直接相关——**SA-B 候选：task 子代理审批语义收口**（gaea
子代理当前无审批概念，若 V5 拆刀先修信任洞，此处是上游参考实现）。留池随 V5。

### C. 组合与权限

**C1 applyChildComposition 单调用纪律**：join 父预设组合**先**、孩子自有注册（persona 遮蔽节+tools.restrict）
**后**，一个函数封死——「没 join 预设的孩子=空工具注册表+没有父的 prompt 节」这个缺陷在调用点**不可
表示**。判定：gaea buildSubReg 组装单点同构。销号（形态不同）。

**C2 SUBAGENT_DELEGATION_CONTEXT 固定权限声明**：runtime-context 注入而非 system prompt 段（部署
prompt 跨父跨子保持 uniform）：「你的权限范围在 start 时已定死、不可从会话内扩权——需审批的操作自动
拒绝；任务超出范围时**不要重试被拒操作**，在回复里声明限制让委派方处理」。判定：**反重试指引**是
prompt 纪律真增量——gaea task 人格面无此声明；gaea 子代理无审批闸所以「自动拒绝」暂无 face，但「不
要重试被拒操作」对 gaea 已有的白名单 fail-loud 工具错误同样适用。**SA-D 小刀候选**：task CompactDescription
追加被拒操作反重试一句。低价值（小改）。

**C3 路由继承=请求头权威**：子继承父**实跑路由**（requestHeader 的 provider/model/effort——请求时选择
后的结果），创建 options 只是首请求前 fallback；**换路由不点 effort→清 effort 让新模型用自己默认**（路由
自有 effort 不跨路由携带）。判定：gaea task 子代理恒继承父构造期 prov/contextWindow（task.go:801）——
gaea 回合内路由不变所以等价；**「路由变→effort/温度类路由私有参数一并重置」纪律**随 P2 模型路由池走。

### D. 模型面

**D1 措辞由能力推导（诚实面的机械保证）**：工具 description 按 provider 能力拼装——inheritsParentContext
→fork 措辞（「种子含全部已完成回合，**看不到进行中的回合**；prompt 只说新增」）/spawn 措辞（「自包含、
不共享会话上下文」）；prepareContinuable 有无→「孩子用 send_message 交结果，运行中可 steer、结束后可
续聊」vs「**完成通知含最终答案，此后端不接受后续消息**」。判定：gaea task Description 是静态文案，fork
（continue_from 装载）与 spawn 共用同一张嘴——**措辞没有跟着「是否见父历史」分叉**。**SA-A 配套小刀**：
continue_from 路径注入「该子已见父历史前缀」语境差异（一行文案级）。

**D2 共享引导只挂第一工具**：多委派工具实例同 scope，system prompt 节只渲染**第一个可见**工具名
（"Start independent delegations with X together in one assistant message and continue useful work while
they run"）。判定：gaea 单 task 工具无此面。销号。

**D3 list_agents 模型面纪律**：one-shot 子**省略**（「不可被 send_message 续聊，模型永不选中」——发现仍
遍历它们算 descendants）；status 只报 running/inactive（**「报告回合活动而不暴露是否驻留」**——进程内驻
留与否是实现细节）；诊断行 corrupt/unsupported/unavailable；**反轮询文案**「完成会通知你，无需轮询状态」
；descendants 行带 parent+depth 且「深于 1 只接受 interrupt_agent」。判定：gaea subagent_list 已同构且
更细（queued/running/completed/failed/interrupted+stale 标注+动作指引尾注）；「stale 只标注不改写 meta」
与上游「不暴露驻留」不同路但同为诚实面。**销号（已对齐）**。

**D4 能力校验 fail-loud at load**：maxDepth 数字要 provider depthLimit 能力否则 mount 炸（'provider-managed'
逃生口给进程外 provider）；空 toolFilter（allow/deny 双缺）load 拒（「而不是让每次委派失败」）；provider
中途换=工具随卸载；路由 preflight 期间 provider 换人→「retry the delegation」。判定：gaea UnknownToolNames
fail-loud 同族已做；provider 概念 gaea 无。销号。

**D5 modelSelection 采样纪律**：新顶层会话在 agent/created 采样 Host 设置；子会话从父**已记录政策**继承
（父缺省时查 parentSession 的 projection）；采样结果**记录进会话投影**——后改设置不变异 live 会话（per-agent
fiber 捕获政策）。P2 模型路由池口径（022 档已录），不重复。

### E. 结构化出口（本轮最大真增量）

**E1 structured_output seam**：子域注册**真 schema 工具** `structured_output`（每子自己的 scope，并发互
不干扰、处置无全局残留）+尾注 scoped prompt 节：「拿到最终答案后**必须**调 structured_output 交参，参数
须严格匹配 schema；**不要用纯文本收尾——只有工具调用算结果**」。捕获纪律：capture 只在权威 tools/result
成功后提交（PTC 模式还等外层 run_code 结果）；**input 从 capture staging 起关闭**（此后消息 INPUT_CLOSED
拒收）；失败结果**丢弃 capture 并重开 input**；单调工具守卫防后续调用重开已完成的 structured run；
completed 无 capture→降级 error（A7）。判定：**真增量 P1 候选 SA-A**。gaea task 出口=纯文本（最后一条
assistant 消息），父模型侧消费=读文本；「委派→要 JSON 回来」在造价/识图/数据整理等办公场景是高频刚需
（任务函数式复用）。落地形态=gaea 子会话注册临时 structured_output 工具+尾注+捕获+降级，出口进 task
结果（文本+structured 双轨）。中等刀（新工具+捕获 seam+测试）。

### F. 并发与序列化

**F1 ActivationPool 共享槽**：容量槽**沿连续 continuable 父链共享**（Activation 及全部受管后代共池）；
满=**拒绝**（ACTIVATION_LIMIT_REACHED+可行动文案「等现有孩子完成或用手头代理完成工作」）而非排队。
判定：gaea 批 1=排队等槽（+queued 诚实态），上游=拒绝+行动指引。两语义各有理：**gaea 排队对「同批
多路」更顺（task 发射方不空手）**；上游拒绝对模型自主节奏更诚实。**维持 gaea 现状，销号**；留档：
错误文案的可行动性（gaea 排队态已含等槽语义，无此需求）。

**F2 ChildLock 每 child 串行**：delivery/release/disposal 关键段按 childId promise 链线性化，tail 吞
rejection（一段失败不毒化后来者）。判定：gaea subRuns+互斥锁面小，无等价需求。销号。

**F3 生命周期边诚实**：residency 前失败**不发 edge**（「编造一个=报告孩子从没有过的生命周期」）；每监听
器独立异常遏制（同步抛/异步拒都 log，不饿死同伴、不改运行、不破坏 teardown）。判定：gaea 事件面
（TurnStarted 等）无子代理专属 edge；监听器遏制纪律=通用工程纪律。留档。

**F4 drain 拆卸序**：结构 disposer 先注册、drain 后注册——reverse unwind 使 drain 先于 scope 释放跑
（**child-first 拆卸**，防结构处置越过孩子序）；drainDescendants 按精确 live 根聚合。判定：gaea jobs.Kill
级联（runningDescendants）已做同族。销号。

**F5 interrupt 全语义**（重核）：祖先授权=materialize 时观测的**精确 live 祖先链**（WeakSet，防运行时
滞留）+`ctx.agents.get(id)===caller` 身份核验（非仅 id 相等）；user 授权=精确父 id；自打断拒；**幂等**
（closing 中=直接返回，不重复发信号）；本地=`cancel({kind},{keepInbox:true})`——**只停当前回合，inbox
保留**，idle 后 send_message 可续；外部=abort+dispose 永久停；**「Subagents it started will keep running」
诚实面**（后代不级联）。判定：gaea P0-A 同构（subRuns cancel）；级联差异=gaea 后台 job 走 jobs.Kill
级联而前台 task 中断不级联（无嵌套面），等价。**已落地销号**。

**F6 send_message 全语义**（重核）：邻接双向（直接子/直接父）；running=下一步界 steer、idle=起新回合、
**外部 NOT_CONTINUABLE 拒**；**返回送达回执非答案**；双归因 agent-message（relay form）vs subagent-settled
（notice form）——「transcript 把两者合并=替孩子记它没说过的话」；子任务尾注 withContinuableReturnGuidance
（父 id+finish 前 send_message 交自包含结果+**中间发现改变父下一步时也发**+发消息不结束回合+父共享工作
区但**不自动收到你的 transcript/工具输出/推理**）；0.2.2 新增 omitClosingText 去重（finish 前发过话→结算
通知省收尾全文）。判定：维持 P2 demand-gated（022 档口径不变）；**归因分离+尾注纪律**随立项照单。

## 三、刀池更新（本轮新增；2026-11-02 执行批后销项状态）

| # | 刀 | 性质 | 状态 |
|---|---|---|---|
| SA-A | structured_output 结构化出口 | ~~真增量 P1~~ | **销号（执行批盘点改判）**：gaea task 已有完整 output_schema 实现（schema 注入子指令+终答 JSON 校验+schemaRetryNudge 同会话有界纠错+非 JSON 诚实降级前缀）——调研时 grep 漏检（搜 structured 未搜 output_schema）。上游工具调用通道属形态差异（无解析歧义），gaea 文本+纠错是等价替代，取道不取器 |
| SA-B | 委派审批语义收口 | 纪律收编 | 在册（随 V5 信任洞拆刀） |
| SA-C | stopReason「失败赢过取消」 | ~~小刀~~ | **销号（执行批盘点改判）**：判据缺失——gaea classifySubagentStop 只有 (runErr,runCtx) 二元组，取消会以普通传输错误浮出（list_tool_test.go:33 钉住 runCtx 取消+普通错误=interrupted=架构真相）；上游判据来自子代理自己的日志（epochStopReason 读 turn/end），gaea 需回合级终态记录才能实现，不值得 |
| SA-D | 被拒操作反重试指引 | 小刀 | **已落地（2026-11-02 执行批）**：task CompactDescription 追加段 |
| SA-E | fork/continue_from 措辞分叉 | 小刀 | **已落地（2026-11-02 执行批）**：CompactDescription 追加「continue_from续跑的子代理已带原任务与transcript,prompt只补新增部分」 |

留档（不立项）：descriptor 版本化显式快照（B1，meta 加字段时走版本号）；深度单调地板公式（B2，嵌套若开）；
refusal 词汇（A6，政策闸若建）；「LIVE scope 权威于 header」（B3）；监听器遏制（F3）。

销号（本轮重核维持）：结算准入/竞速双闸/投递三路/announced（gaea 形态无面）；list_agents（已对齐更细）；
能力 fail-loud/provider 面；ChildLock/drain 序/interrupt 级联；ActivationPool 拒绝语义（gaea 排队维持）；
并发槽共享/措辞第一工具。

P2 池终态（2026-11-02 收官执行批回填）：
- **send_message 已落地（子→父半边）**：steerFromSubagent 来源+署名信封（ref 必填，agent-message 与 user
  归因分离纪律）+send_message 工具（送达回执非答案/发消息不结束任务/父看不到过程须自包含）+派发点
  WithRunner 盖章+runSubAgentInternal WithSubagentRef 署名源+双消费点（下一步界消费/drain 兜底 originNote）
  。上游「父→子」「冷恢复投递」两半边不落（父→子=既有 SteerSubagent；冷恢复=需常驻概念，demand-gated
  维持）；omitClosingText 去重无消费面（gaea 结算走任务结果非通知，无同文双载）。
- **模型路由已落地（配置授权清单形态）**：`[agent] subagent_routes` 清单（空=路由关闭=只许父路由）+
  task engine/model 参数（须同时给+逐字核验+未授权 fail-loud 点名可用路由）+boot 接线 wubigrok bridge
  显式 Engine+Model+CompactDescription 动态列授权路由（发现面，零新增工具）。上游「创建时授权记录+不
  重采样+会话投影」不落（gaea 授权=配置层一次性，无会话级采样面）；窗口/计价继承父（引擎无窗口元数据，
  K1 自学习兜底）。
- **session-reference 销号**：已由阶段十 10.3 落地（control/refs_session.go @会话 快照注入，v4.494——
  021 档成文早于 10.3，池口径过时）。
- 深度记账：销号维持（B2 公式留档）。**至此 dsh 子代理蒸馏刀池全部销项。**

## 四、方法学注记

- 工作树检出=alpha.2，tag diff 补 0.2.2 增量（omitClosingText/preset-selection/modelSelection 拆分）——
  「克隆检出≠调研基线」时，读文件前先 `git log --oneline -1` 对准，差异走 `git diff tagA tagB -- path`。
- 上游注释是纪律本体（多数机制注释写明「为什么」而不止「是什么」）——蒸馏读注释优先于读签名。

## 五、第二轮深读补遗（同日续，源码全覆盖收官）

> 本轮补读：服务门面 index.ts(646) 全文、manager 余段（startLocal/startExternal/materializeTracked/
> announce/rollback/acquireOwnership/coldResume/deliverFollowup/sendToParent/submitAdmitted）、spawn/fork
> provider 全文、catalog/list-children/projection/control-types/out-of-process/archive-admission/control.ts、
> model-selection 族。**至此子代理核心包 6360 行全部读毕**（tool-subagent 491+control 290 首轮已读）。

### G. 服务门面与生命周期补遗

**G1 门面合同（index.ts）**：Config=`maxActiveSubagents` 默认 **8** volatile、`maxDepth` 默认 1（gaea
默认 3 硬帽 8——上游默认更宽且无硬帽概念）。sendMessage 合同确认：**absent 的直接子代理=从持久化冷
恢复后投递**；caller 取消只拥有受理前。interrupt 合同确认：未认领 inbox/Activation/已公布后代全保留；
absent 目标=受理 no-op（manager-less 组合亦 no-op）。`prompt` 浏览器面：`BrowserPromptSource{kind:'user',
rpcId, clientTimeZone}` 与普通会话 prompt 共用 session-request-id brand（同一身份词汇）；时区校验（UTC
或 IANA Area/Location）；**图片部件先经 attachment store 入库再投递——「子 inbox 只收 Host 持久化附件」**
；容量满冷恢复→`subagent/delivery-unavailable`。`interruptByParent`：**不查 catalog/历史/持久化/父
Agent**——「原语单独对 live Activation 授权=父 Agent 离线时孩子仍可被打断」；absent/idle/已完成=受理
no-op。registerProvider：effect 作用域+HMR 安全，移除=阻新启动不撤已受理运行，双注册 fail-loud，
无 start 无 prepareContinuable 拒注册。assertCapabilities 按**请求字段在位性**逐项核五能力，第一缺失即拒
——「never accepted-then-ignored」。

**G2 本地 provider=纯 DATA 贡献者（spawn/fork 全读）**：spawn 的 prepareContinuable 返回 `{}`（无种子）、
capabilities 全 true、inheritsParentContext=false。fork：**种子=父日志最后一个 `turn/end` 含在内的平衡
前缀**（「进行中回合不平衡、不可作为合法子会话回放」；无完成回合=空种子从头）；**种子创建时捕获一次**
——成为子自己的持久 transcript，冷恢复回放该前缀、**不重新 fork 父的更新历史**；inheritsParentContext=true。
两 provider 的 inject 都**刻意不含 tools**（「子工厂已在 setup 提供；加了会变 apply 时机与委派工具在模型
可见清单的位置」）。gaea 对照：task=fresh（spawn 同构）/fork=种子父历史/continue_from=装载既有 transcript
——fork 种子「平衡前缀+一次捕获」纪律已同构，销号留档。

**G3 发现面（catalog/list-children）**：发现数据=**父属 durable catalog 事件**（`subagent/catalog` 版本化
v0/v1/v2 追加进父日志，v2 增 external；chunked-list 投影防大目录）——持久在父、重启幸存、无 sidecar
一致性面。读取=live-preferred 观察（`using` 即用即放）「不装载不恢复任何 Agent」。descendants 前序遍历：
external=叶子；one-shot/unknown=遍历节点（unknown→unsupported 诊断，可读目录仍遍历）；不可读→
corrupt/unavailable **只断该分支**；**「不在可达目录里的会话被省略」**。诊断三词精确语义：corrupt=读到
无效/冲突数据、unavailable=读不了、unsupported=父目录记了未知模式。gaea 对照：SubagentStore meta 侧车
语义等价，「损坏单条跳过」已同构（P1-D）；chunked 分块 gaea 不需要（cap 50 截断已够）。销号。

**G4 投影族（projection.ts）——UA-1 数据假设修正**：上游「活跃耗时」=**回合区间累计投影**（subagentTiming：
`settledMs`=descriptor 后已完成回合毫秒累计 + `active` 开区间 + `lastTurnCompleted`；descriptor 前
pendingTurnStart 升格），**不是 createdAt→updatedAt 墙钟**——墙钟把驻留闲置也算进去，上游算真活跃。
identity 投影=descriptor 折叠（mode/label）。**刀池影响：UA-1 若做「活跃耗时」，轻量路线=墙钟（诚实标注
含闲置）或后端补回合区间记账（中等刀）；口径留拍板，不擅动。**

**G5 进程外纪律（out-of-process）**：进程外 backend 的 capabilities advertisement=**NONE**——「进程外
孩子无法履行父强制的启动特性→服务在 start 前拒绝，绝不 accepted-then-ignored」。diagnostic 4096 字节
截断**预留截断后缀字节再 rune 回走**（后缀占位算进预算）。never-reject settlement（子级失败 resolve 不
reject）。进程机制本身（spawn/env 清洗/托管拆除）归 dsh-subprocess seam，本包只管合同。

**G6 归档联动（archive-admission）**：工作区 `workspace/session-activity` 应答 running 子代理后代
（kind:'subagent' 带描述）+`workspace/session-stop` 按父同款取消各后代；**单子取消失败只 log 不挡兄弟**。
谱系按数据读+环守卫；**「fork 共享谱系字段但无 origin=独立会话，永不持其源」**——022 档 origin 判别纪律
的实战用例。gaea 对照：gaea 无会话归档概念，销号。

**G7 远程面错误码分诊（control.ts）**：zod 校验严于品牌字符串编解码；**rejectPrompt 分诊纪律**——「调用
方可行动的受理失败保留自己的码（NOT_RESUMABLE/UNAUTHORIZED/…），其余一律 internal」**不暴露 provider
详情**。gaea 对照：Wails 绑定面错误=Go err 链透传，单进程无 provider 边界；远程面若将来出现（web 端）
照此分诊。留档。

**G8 materializeTracked 全序收官（manager 尾段）**：create 路径的 descriptor+delegated 政策+workingDirectory
set 全在**未公布创建窗**内追加；resume 路径只回放不追加；applyChildComposition 两路共用。structured
runtime 挂载带 readiness 谓词（ownedChildren 空+双 inbox 空→capture 可定稿）。ancestry=`WeakSet([子
agent, ...父谱系])`；公布后准入三复查（admitting/signal/exact live parent）→acquireOwnership。**inbox 三
事件（inserted/claimed/discarded）都唤醒结算观察器**。announce=**受理后才起结算观察**；external 自然
结算=close(stop:false) 不发取消。失败→rollbackUnpublished（dispose+删+放槽+放所有权+result.reject）。
**初始 prompt source={kind:'user'}+delivery 'queue'；withContinuableReturnGuidance 只在子的工具注册表看
得到标准 send_message 时附加**（「指引随工具走」——部署无 send_message 就不教孩子发消息）。close=记忆化
幂等事务；disposeRoots 聚合失败带计数。deliverFollowup 循环：ChildLock 内 resident?提交:coldResume；
**关闭中=等关闭完再重试**（send-vs-dispose cutoff 同关键段）；丢 cutoff→assertAdmitting+throwIfAborted+
重试。coldResume：观察→authorizeLineage(header.parentSession)→descriptor 折叠（仅 inheritedEventCount
之后）→**须 continuable 否则 NOT_RESUMABLE「choose a different target」**→materialize 全从 descriptor
重建（provider/route/composition）。submitAdmitted：source 无=父署名 agent 消息、有=**保留宿主源描述符
（不冒充 Agent）**；acquireOwnership 在 deliver 前。assertImageCapable：resolveModelInfo inputModalities
无 image→MODEL_DOES_NOT_SUPPORT_IMAGES；无 llm/无路由=顺延投影裁决。

### 刀池影响（第二轮）

- **UA-1 口径修正**：活跃耗时=回合区间累计投影（G4），墙钟方案须诚实标注「含闲置」——口径留拍板。
- 其余刀池（SA-A~E/K1~K4/UA-2）不变；G 系全部为纪律留档与销号，无新立项。
