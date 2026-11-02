# DSH 0.2.1-rc.1 → 0.2.2-alpha.2 增量蒸馏 —— 调研与刀池（2026-11-02）

> **源**：`clones/deepseek-harness`，本轮 fetch 增量 tag **rc.3 / rc.4 / rc.5 / 0.2.2-alpha.2**；
> 调研基线取最高 tag **dsh-v0.2.2-alpha.2**（已含 master 全部内容，见「版本景观」）。
> 前情：子代理板块（docs/gaea-dsh-subagent-distill-2026-10.md，P0/P1 已落地 e19543bb/f5aa2892）与
> 核心域（docs/gaea-dsh-021-core-distill-2026-10.md）均基于 dsh-v0.2.1-alpha.2；本轮补齐其后的全部上游演进。
> 方法：`git diff --name-only alpha.2 0.2.2-alpha.2 -- docs/upgrade-guide/` 得 **57 篇新指南**（含补写进旧版
> 目录的 alpha.1/alpha.2 指南），逐篇读 gaea 相关子集；子代理包定向 src diff（183 文件 +3273/−1319）。
> 纪律不变：取道不取器；GoalCard/MCP/平台化/跨设备在册排除。

## 〇、版本景观与方法学坑

- **master ≠ 最新**：origin/master HEAD 停在 0.2.1-rc.2 合并点（687f282e86）；rc.3/rc.4/rc.5/0.2.2-alpha.2
  在 release 分支上（tag 可达）。调研基线取最高 tag，勿以 master HEAD 为准。
- **坑①**：`git rev-list --count alpha.2..rc.2` 得 22393——worktree 分支合并史全混进 range，**不可作变更
  规模依据**；增量调研的正道=升级指南 diff + 定向 src diff。
- **坑②**：升级指南有「补写」形态——alpha.1/alpha.2 目录下的指南（job-id-identity、subagent-preset-composition、
  jsonl-persistence-plugin、sdk-minimal-image-compaction、stored-credentials-precedence 等）是后来补写的，
  文档晚于 tag。只按新版本目录过滤会漏。
- **坑③**：克隆基线不等于上游最新——本轮 fetch 前克隆停在 alpha.2，fetch 后 tag 列表尾部多了 4 个。
  调研第一步恒为 `git fetch origin --tags && git tag | sort -V`。

## 一、既有 P2 池三项的上游演化对账

| 池项 | 上游 0.2.1-rc.x 演化 | 判定 |
|---|---|---|
| per-call 模型路由 | **上游已给参考设计**（rc.1 subagent-control-tools + alpha.4 session-model-selection）：`modelSelection`/`modelDiscovery` 分离；发现工具 `list_subagent_models` 每 tool scope 恒一实例（多主注册 fail）；授权在 **Agent 创建时记录**（Host /model-selection-settings）——子会话继承、恢复保留、事后改设置**不重采样**既有会话；无记录政策时发现与显式选择只许父路由；启用设置+空允许清单=拒绝一切新顶层会话（UI 阻止保存该组合）。会话级选择**下一步界生效**（ACP 不再钉整回合），`agent/request` 废弃（不能改 provider/model/effort） | P2 池项**口径更新**：需求未现维持 demand-gated，但落地时照单收「创建时授权记录+不重采样+空清单全拒」纪律 |
| send_message | 注册面整合（send_message/interrupt_agent/list_agents 三件套一体注册，独立 /list-agents 导出删除；共享行禁用=三工具全禁，藏单个用 tool restrictions）；**结算通知 omitClosingText 去重**（子代理 finish 前经 send_message 发过话→结算通知不再带收尾全文，改「它刚给你发过消息，收尾省略」防同文双载） | 形态判定不变（大改 demand-gated）。去重纪律留档——gaea 无 send_message 无重复源，暂无可消费面 |
| 深度记账 | **depth.ts 零变更** | P2 维持原判 |

另：alpha.1 补写 job-id-identity——job id 从 `<kind>-N` 改 `<kind>-<16hex uuid>`（同一执行跨 registered/
settled 同 id；后起 registry 实例发不同 id；lifecycle 不再带 executionId）。gaea jobs.go:172=`<kind>-<seq>`
进程内单调、重启重置，但 gaea job 不跨恢复、单 manager 实例、模型面引用同进程内 id——**无实际碰撞面，
销号留档**。

## 二、新刀池候选（本轮增量）

### K1 切模型压缩门（alpha.4 model-switch-compaction）——真增量，待核后可立项

上游：compaction 自动模式下，切 provider/model **先查目标窗口压力预算再落新请求头**；容量无法解析、
预算非法、压缩压不进阈值 → `MODEL_SWITCH_COMPACTION_FAILED` 拒切并**保留旧请求头**（已完成修剪与既有
checkpoint 不动）；每路由可配 `headroomTokens`（默认 65536，须与该路由输出预留共存）。

gaea 现状（已核）：压缩窗口来自 profile 构造期 `ApplyModelProfile`（budget_alias.go:28，profile.ContextWindow
> 0 才覆写）；切引擎 `SetActiveEngine`（model_engine_handler.go:145）只改 client+持久化+emit model-changed，
**会话在途 runner 的压缩窗口是否随切重解析=待核（刀前必查）**。gaea 切引擎是模型中心常态操作，切到小窗
口模型后历史溢出的行为要有确定语义——与上游同判。

### K2 checkpoint 持久化门（rc.2 sdk-minimal-checkpoints）——小刀候选

上游：sdk-minimal 启用 session-checkpoint-policy——**会话历史 flush 完成前，模型请求/顶层工具执行/agent
步全部等待**；checkpoint 失败=停依赖操作（不再后台警告继续跑）；首用文件锁初始化计入首请求延迟；不改
会话格式。动机=持久化错误时操作照跑会留下「以为落盘了」的假账。

gaea 现状：断电线 fsync 三处+损坏隔离自愈已做（v4.517.0，2026-10-06 断电事件）；但「**写失败停回合**」
的闸语义（LogWriter 出错→本轮模型请求/工具步暂停）是否已具备=待核。若 gaea 现行为是「记日志失败只
slog.Warn 继续跑」，同判立项。

### K3 附件引用完整性校验（rc.3 attachment-read-access）——纪律收编候选

上游：`session/attachment` 读图从 `{sessionId, attachmentId}` 改为携带**完整 ImageAttachmentRef**
（attachmentId+mediaType+bytes+width+height），Host 逐项核与存储对象一致；**name/originalDimensions 是
调用方展示信息，明确不验**；缺失/损坏/不匹配显式失败。授权内不查图片归属（跨会话可读同 store 图片）。

gaea 对照面：识图/附件读路径（vision 线+附件链）是否按「完整引用逐项校验」取数——待核；纪律本身
（数据面字段全验、展示面字段明示不验）照单收。

### K4 工具取消错误码词汇（rc.3 ptc-caller-cancellation）——小纪律

上游：调用方取消的 run_code 结果 `CODE_RUN_FAILED` → **`ABORTED`**（AbortError）；agent 取消因由照记
turn/end（reason.kind=aborted）；运行时失败（超时/清理失败）与运行时本地取消仍 `CODE_RUN_FAILED`；
历史会话日志不重写，消费方须双认。

gaea 对照面：工具结果错误码族——SubagentStopReason 词汇（completed/interrupted/error，P1-C）已做同族
纪律；工具面取消 vs 失败是否可辨=待核，不可辨时照此词汇补。

## 三、纪律收编（随刀带走，不单独立项）

1. **preset 允许清单**（alpha.1 补写 subagent-preset-composition）：上游拒收 per-child persona/toolFilter，
   改 agentPreset 预设组合+`allowedAgentPresets` 模型可选清单——默认须在清单内、重复 id 构造期 fail-loud、
   fork 恒继承不可自改（要不同组合用 spawn 换预设）。gaea 现形态（task 人格走 Description/CompactDescriptor
   面+toolFilter 白名单 fail-loud）**销号维持**；将来 task 加 preset 参数时照单：模型可选预设必须带允许
   清单+构造期校验。
2. **条件工具 step admission**（rc.2 conditional-tools）：`when` 条件每步解析（provider/modelSelection 入参）、
   未 admit 调用回 UNKNOWN_TOOL、热重载/MCP 重连换注册=该步未启动调用失败（不静默换新实现）。
   **exit_plan_mode 上游改「进过 plan 模式才声明」——与 gaea v4.414 拍板（恒注册，「工具目录跨模式切换
   稳定」）方向相反，维持既有拍板不跟**。`when`+UNKNOWN_TOOL 纪律留档。
3. **信任分级词汇**（rc.2 auto-review-user-sources，上游实验特性）：`source.kind:user` 文本=任务指令可在
   其声明范围内授权中风险动作；子代理自建 prompt 与直父消息保持从属角色；文件/工具结果/checkpoint
   **永不能授权动作**；注入内容在过滤历史中呈 fact/constraint，用户文本呈 user-instruction。与 gaea
   Z3 防伪造同意条款+办公记忆写入确认红线同族，词汇更细，留档对照。
4. **指令链分链预算**（rc.1 agent-instructions-chain-budgets）：maxBytes 从「基线总量」改「用户全局链与
   项目链各算各」——双链同载消息可达两倍配置值；dsh-base 65536 保持手段=减半配置。gaea AGENTS 预算
   （37486B 漂移闸）单值口径——将来收紧时按链分，销号级留档。
5. **fork origin 判别**（rc.3 independent-session-forks）：普通 fork 不再记 parentSession（origin 独立），
   消费方判子代理须用 `origin === 'subagent'` 字段而非 parent 存在性；旧读器会把带继承送达记录的无父
   fork 判损坏日志拒开。gaea SubagentStore meta 侧车自带 runs/origin 面，等价；纪律=新消费方判别走
   origin 字段。
6. **结算通知三态措辞**（上游 rc.x 持续打磨）：stopReason→概要措辞（continuable 区分）+structured+diagnostic
   分离呈现——gaea P1-C 已对齐，无新增量。

## 四、辨伪与销号（一行判定）

- session-store-writes（rc.2，store 属主 publish 前/移除后拒写+失败创建留史可 resume）：gaea 会话文件
  本地 JSONL 直写生命周期不同，等价面已足，销号。
- session.follow activate 显式化（rc.3）：gaea 无跨进程 follow 面，销号。
- new-session-blank-reuse（rc.2，空白会话复用+draft 保全+预设流跳他预设草稿）：前端 UX 纪律，gaea 新会话
  行为待对照，入前端池非内核刀。
- stored-credentials-precedence（alpha.2 补写，存储键压过导出 env；headless 默认禁凭据库）：gaea 无
  .credentials store，销号。
- tool-executor-plugins（rc.2，native/PTC 执行器拆插件+maxParallelSubCalls 迁 PTC 执行器+loop 并行默认 10）：
  形态=cordis 插件面，gaea runParallel 硬帽 8 自有口径，销号。
- sdk-minimal-image-compaction（alpha.2 补写，read_image+自动压缩+超预算图片永久换占位符）：gaea 识图线
  自有管线，留档「超预算占位符」一手。
- approval-projection-dependency / headless-policy-composition / official-bundle-lists / jsonl-persistence-plugin /
  pi-ai-* / cua-* / web-* / desktop-* / 账号通知 / 插件引擎 / 打包基建（Windows runner supervision 等）：
  cordis 组合面/Web 客户端/桌面打包域，gaea 无对应消费面，一行销号。

## 五、刀池汇总（拍板池；2026-11-02 执行批后终态）

| # | 刀 | 性质 | 终态 |
|---|---|---|---|
| K1 | 切模型压缩门 | 真增量 | **已落地（2026-11-02，变体形态）**。刀前核实：主 executor 窗口=构建期 entry.ContextWindow（boot.go:519/127-155），wubigrok bridge 按活跃引擎请求期路由——切引擎后窗口失真，溢出梯子按旧窗压/截断。引擎登记表（modelengine.EngineConfig/ModelInfo）**无窗口元数据**，「切时重解析/拒切」无数据源→改走**溢出错误自学习**：provider.ContextOverflowLimit（三个实测措辞家族正则：OpenAI「maximum context length is N」/Anthropic「N tokens > M maximum 取 M」/Gemini「maximum number of tokens allowed (N)」）+ tryOverflowRecovery 救援前采纳（只缩不涨；0 窗口=禁压缩语义不自启）——切小窗引擎首溢即采纳真实窗口，本轮阶梯与后续轮次按真实值收敛。上游「拒切保旧头」的 gaea 等价面=不拒切、自学习收敛（偏离记录在案） |
| K2 | checkpoint 持久化门（写失败停回合） | 小刀 | **销号**。核实：EventLogSink 写失败=slog.Warn+return（sink.go:126），回合照跑=「事件日志尽力而为旁路」既有拍板语义；gaea 兜底已足（断电线 fsync 三处 v4.517.0+RepairLogFile torn-tail 修复+回合间关写器）。「写失败停回合」与既有语义冲突且无消费面（单机桌面非 headless API 契约）；上游 sdk-minimal 是 headless 部署形态，若将来 API 化再对齐 |
| K3 | 附件引用完整性校验 | 纪律收编 | **销号**。核实：GaeaAttachmentDataURL(path)=本地单进程读自己写的文件，无远程附件读 API；上游纪律防的是多客户端远程面的引用伪造，gaea 无此信任边界 |
| K4 | 工具取消 ABORTED/FAILED 双认词汇 | 小纪律 | **销号**。核实：gaea 已分流——StreamInterruptedError+interrupted bool（agent_run.go:154-166 流中断恢复「re-issue it if still needed」）+SubagentStopReason interrupted/error（P1-C）；取消 vs 失败在回合与子代理两层都可辨 |
| — | 子代理模型路由（P2 口径更新） | demand-gated | 需求现；落地照「创建时授权+不重采样+空清单全拒」 |
| — | send_message / session-reference / compactNow / plan 持久化二刀 | 维持在册 | 不变（深度记账维持销号） |

以上均为调研产出；动刀前逐项走「刀前核实」栏。
