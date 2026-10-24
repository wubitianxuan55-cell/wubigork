# Reasonix 2.x（studio 分支 2.0.0→2.50.0）真身蒸馏 —— 侦察与刀池（2026-10-24）

- 源：`clones/reasonix`（GitHub **esengine/DeepSeek-Reasonix** studio 分支，HEAD a26dacd PR #14337；Go 单二进制 agent，MIT）。前档 `docs/gaea-reasonix-v138-distill-2026-09.md`（v1.15→v1.38，四弹 16 刀落地 v4.374~4.378，留池 9 项全销号收官）——**本档覆盖其后的 2.x 增量：studio 2.0.0→2.50.0 共 59 版，PR 编号 #10613→#14337（≈3700 号增量）；main-v2 止于 1.39.0 维护线**。同期 gaea v4.379→v4.523（阶段八~十一+上游蒸馏 B0~B5），压缩/子代理/访问边界三条线都有自产增量，刀池已剔除已被 gaea 反超或对齐的面。
- 侦察方法：2.0.0 重写总纲 + SPEC/ROADMAP 全文 + 四路并行深潜（会话与上下文引擎 / 权限安全 / 子代理编排补漏 / 架构地图），落点全部到文件:行号。
- 纪律沿用前档：蒸馏=取道不取器；GoalCard 系红线不评估；MCP/平台化/跨设备在册排除（本轮新增排除：扩展 Runtime v2/插件包/主题市场/远程网关/ACP/计费钱包）。
- 2.x 架构一句话：平铺→分层（contract/runtime/safety/ext/assembly/state），权威转录不可变 + **压缩只写投影 sidecar**（ CoveredPrefixHash 五元 CAS 提交），委派五概念（profile/task/capability/capsule/scheduler）+ 完成裁决（citation 对 receipt 只降不升）。

## 一、上下文引擎刀池（gaea 主链路最痛的面）

| # | 机制（上游落点） | gaea 现状 | 刀深判断 |
|---|---|---|---|
| A1 | **usage anchor 压力校准**（`agent/usage_anchor.go`：压力=provider 上次真实 prompt_tokens+本地增量同比相消；tokPerChar 由真实 usage 校准限 0.05–2 兜底 0.25；无窗口端点用 contextWindowProbe 从被拒请求推断真实窗口） | **辨伪销号（2026-10-24 对源复核）**：gaea `tokenmeter.go`（v4.381 dsh 蒸馏）即同构等价物——锚点+有符号差值+per-message memo 相消，tokPerChar 钳 0.05–2 兜底 0.25 全同；报告初稿「水位线全靠本地估算」系误判（v4.523 上屏的是三阈值**显示**，计量 v4.381 已锚定）。窗口探测（contextWindowProbe）gaea 无对应面，留观察 | ~~高~~ **已存在** |
| A2 | **压缩内容级保留策略**（`compact.go:353-460`+`compact_user_turns.go`：digest 后用户轮逐字保留预算=窗口 5% 下限 1024 tok、最老优先、预算唯一闸〔单轮上限是被刻意移除的形态〕、固定比率估价防漂、丢弃显形+`[[keep]]` 逃生通道、失败 tool 结果优先保留+保留顶过上限降 KeepErrors 再试） | 有 KeepErrors（前缀兜底）+KeepUserMarked+KeepProtected；**digest 后用户轮零保留**——region 内短轮走 pinnableUserTurn 无预算封顶，长轮直接落 fold | **已落地 2026-10-24（刀 A2 v4.524）**：keepUserTurns 进 partitionFold（预算唯一闸对齐上游、丢弃 Notice 上屏、固定 0.25 估价）；pinnableUserTurn 收缩回头部 pin 专职。未取：MarkedDropped 独立预算（gaea [[keep]] 轮与 userTurns 共用 policy 面即可）、单轮溢出的结构化 ToolFailure 判定（gaea provider.Message 无该字段，前缀兜底已是诚实边界） |
| A3 | **压缩验收经济学**（单档 digest：候选必须更小/≤checkpointCeilingRatio 0.50/不得≥trigger 0.85/固定前缀已超 50% 走 ≥25% 节省例外/折叠区必须在 closedPrefixEnd 闭合边界结束/活动回合用户消息永不折叠） | compact.go 有 shrink 硬校验（摘要<fold 区）+consecutiveCompacts 重试+compactStuck 熔断，但**候选验收无经济学** | **已落地 2026-10-24（刀 A3 v4.524，适配版）**：验收闸进 compact——候选下限（机械摘要保底估算，shrink 校验保证可达）≥ trigger 一律跳过（落回 trigger 的压缩买来下一次折叠；force 不豁免；manual 且低于 trigger 例外）+固定前缀超 ceiling 走 ≥25% 节省例外。**两处刻意差异**：①下限用机械摘要保底而非真实候选（gaea 摘要在未来时点）；②通用 above-ceiling 拒绝不取——gaea tail 预算有 2000 tok 下限（v10.11），小窗口下 tail 单独顶破 50%，照搬会关死小窗口自动压缩（探针实证：例外分支在 gaea 常数下由 trigger 闸覆盖）。closedPrefixEnd 不变量以回归测试钉死（tailStart 对齐+配对完整性） |
| A4 | **中断交接块 interrupted-turn-recovery**（`agent/interrupted_recovery.go`：流失败/取消时持久化 bounded 记录〔≤24 工具对/≤8 文件/240 字符〕，下一条真实用户消息一次性折入「不要盲目重复已完成的写入」前缀） | 零命中；中断后模型凭记忆重做已落盘写入是真实痛点（办公/小说长生成中断场景多） | **已落地 2026-10-24（刀 A4 v4.524）**：capture=流终局+预算阻断两 return（回扫真实轮起点跳过 steer/流恢复/readiness 三类合成用户消息+压缩摘要）；注入=runDirect 入口 one-shot；判据="stream interrupted" 信封文案或无结果回执=被打断。刻意差异：gaea 部分输出本就落库（v4.379 刀2）无 unsafe 条款；记录内存驻留不持久（进程重启丢失=诚实边界，上游 LocalOnly 持久） |
| A5 | **折叠区召回 compact_search**（折叠内容建四类文档 BM25 召回，噪声比 0.25 限 8，搜索永远走 canonical 穿越折叠代次） | 零命中；压缩后旧内容只剩救援 archive，无检索通路 | **已落地 2026-10-24（刀 A5 v4.524，形态适配）**：`recall_archived` 新内置工具（只读，ctx 注入 archive 目录）——检索对象=archive JSONL（gaea 的折叠区在文件里，上游在 DB canonical）；确定性计分（查询词+CJK 双元命中计数）、tool_output 由所属调用寻址（召回连来路）、噪声比 0.25、默认 8 上限 20、片段 240 rune、输出 4KiB 封顶；坏档跳过不拖垮其余（per-source 口径）；合成消息/摘要不入检索面。上游「搜索走 canonical 穿越代次」在 gaea 由 archive 文件天然承载（跨代原文都在盘上） |

## 二、安全/权限刀池

| # | 机制（上游落点） | gaea 现状 | 刀深判断 |
|---|---|---|---|
| A6 | **递归删除硬拒绝 guard**（`tools/builtin/bash_delete*`+`safety/shellsafe/delete*`：home/文件系统根/工作区祖先/工作区外递归删除**全审批模式硬拒绝**，字面解释器负载/find -delete/xargs/cmd /c 递归解析，错误码族 shell.delete_*） | **零命中**——bash 面无递归删除防护；memory 在册事故：taskkill //IM node.exe 全杀 | **高**——纯增量零行为风险，防事故刀 |
| A7 | **诚实读拒绝**（`tools/builtin/target_access.go:86-111`：结构化 Refusal——越界列出具体允许读根+两条出路、禁读列表只说「在禁止列表上」不指认规则防探测、覆盖写保留读因改名） | `ErrAccessDenied`="path access denied" 一句话，无路径无根无出路 | **高**——B1 AccessPolicy 的拒绝面升级，模板化纯函数 |
| A8 | **bash 审批六分类**（`bash_approval.go`：Unparsable/NestedExecution `$()`/DynamicName/InlineCode〔python -c、heredoc 喂解释器〕/IndirectExecution〔eval/source/xargs/find -exec/sudo〕/HereDocBody → reusable/exactOnly/requireHuman 三级） | gaea 无 bash 审批 UX（桌面单用户自动执行为主） | **中**——先盘 gaea bash 审批面是否存在再定；只读表 9 月已对齐 |
| A9 | **fresh-human 决策类**（`approval.go:653-660`：永不代答清单〔sandbox_escape/memory 写/config_write/network_egress/paid_search…〕+ grants 契约 fresh 类永不持久化 + 无审批人 unattendedAsk 直接拒） | 「记忆写入确认红线」是产品原则但未机制化为决策类清单 | **中**——把红线显式化成「哪些决策只能回到人」单一列表+持久化契约 |

## 三、子代理刀池（子代理强化线程 P2 同域）

| # | 机制（上游落点） | gaea 现状 | 刀深判断 |
|---|---|---|---|
| A10 | **write_paths 写范围声明+预检不相交**（`delegation/task.go:404` 声明写路径 glob；`fleet.go:43` 并发 writer 预检路径不相交，重叠串行化；两个整工作区声明 preflight 即败；越界写走 extend_write_paths 能力卡必答于人；乐观/严格两档出厂默认 optimistic） | AccessPolicy 管读写边界但**无子代理间写范围互斥**——双会话同树互踩（stale-Edit/TS2300 撞车坑）的 agent 内同型问题无防护 | **已落地 2026-10-24（刀 A10 v4.524）**：task 工具增 write_paths（glob/逃逸/卷根拒，cwd 归一去重）；冲突键按声明集计键（同集同键必串行、异集共存并行）+ 执行期 claim 注册表两两重叠检查（目录包含语义，重叠 fail-fast 如实指路）；真后台任务声明随 job 存活跨回合。**刻意偏差**：声明是协调元数据非安全边界（上游 extend_write_paths 能力卡需审批 UX，gaea 无；越界执行期拦截不做，AccessPolicy 仍是安全边界）；未声明调用零变化（上游 optimistic 的 gaea 对应面=文件观察 stale 检测执行期兜底） |
| A11 | **complete_subtask 完成裁决**（`agent/complete_subtask.go`+`safety/evidence/completion_report.go`：验收判据 citation 对 receipt **只降不升**——verification→HasSuccessfulCommand、diff→HasSuccessfulWrite、files→HasSuccessfulReadOrWrite、manual 永不自证；无回执支撑降 unsatisfied，任一降级 complete 整体降 partial；提交永远成功只分 closed/needs_work 文案） | 子代理完成=模型自由陈述，无回执对照 | **已落地 2026-10-24（刀 A11 v4.524）**：complete_subtask 常驻子代理注册表（不占白名单名额）+ 报告形状校验 + 回执账本=子会话消息（assistant 调用↔tool 结果按 ID 配对，无结果=被中断不构成支撑；失败=isErrorMessage 前缀族）+ runSubSession 收尾裁决（降级结论+逐条理由附结果尾部给父模型；partial/blocked/failed 不是完成主张不裁决）。**刻意偏差**：子代理侧只收确认文案不预判 needs_work（上游宿主侧账本在工具执行期可见，gaea 账本在收尾才齐）；review 证据永不自证（gaea 子会话无 review 回执面）；注册面守卫白名单按其协议登记枚举词 |
| B1 | **ContextCapsule 出发上下文清单**（`sessionstore/subagent_records.go:177-189`：不可变清单+sha256 hash——WorkspaceRoot/SystemPromptSource/ToolScope/Model/Inherited 逐项布尔） | 有 CompactDescriptor（模型面描述），无出发上下文快照 | **已落地 2026-10-24（v4.524）**：SubagentMeta 打平快照字段（ToolScope/Model 死字段写实+workspaceRoot/systemPromptHash/effort/resumedFrom），applySnapshot 一次写定（续跑/追问不篡改）；ToolScope 口径=执行面白名单；模型身份 boot 按实际选中条目注入；不做整体 hash 与续跑一致性闸（gaea 续跑既有闸=status/kind/space，刻意偏差留档） |
| B2 | **list_subagents/read_subagent_result 血统取址**（`delegation/subagent_list.go`：重启打断的聚合没送达时完成态 child 留盘可按 ref 取回，避免重复付费重跑；owner 血统校验 spawn 会话或其后代+同 workspace） | gaea 子代理结果重启后？待盘 | **已落地 2026-10-24（v4.524）**：read_subagent_result 模型面按 ref 分页取回（12KB/24KB rune 对齐；completed 才读、failed=PARTIAL 标注、假 running 判活不改写 meta——多窗无租约语义）；血统校验不引入（store 目录即作用域+ref 不可枚举，与 interrupt_agent 同威胁模型，刻意偏差留档） |

## 四、可靠性刀池（工程面大，列拍板池）

- **B3 durable inbox 幂等收据**（`state/sessioninbox`：幂等键/uncertain 状态/RetrySafe 编码〔唯一值 steer_unapplied〕/快照收据边界——崩溃恢复绝不静默丢失或重放）：**盘底结论（2026-10-24）=完整收据面不移植**（gaea 无多进程/多宿主场景，gaea-log+checkpoint+state running 位已是粗收据）；**最小刀已落地（v4.524）**：steer 收据——running 检查与退出清队列竞态窗（不崩溃也丢+UI 已回执）经 intake 关门闸关死（SteerAccept 拒绝→controller 降级 Send），未消费 steer 落会话记录（[unapplied steer] 前缀进合成轮白名单）+Notice 显形，不自动重放（上游 #7045）；jobs 完成注记持久化=独立小刀留池（幂等键须会话路径+启动时间戳+seq，job 自增 seq 重启撞号）。
- **B4 checkpoint/rewind v2**（Coverage/Gap 声明+MutationObserver+AfterSHA256 所有权指纹+Prepare/Commit/Undo 补偿事务）：全新子系统，等实弹痛点。
- **B5 readiness 续跑预算化**（`control/readiness_continuation.go`：停滞签名 stall=2/max=8 上限+合成轮身份标识与真人区分+预算暂停 grace 回合「下一条消息继续」）：**已落地 2026-10-24（v4.524）**：盘底发现 readiness 闸死闸（finalReadinessCheck 传 nil→missing 恒空从未 block）——新原语 evidence.UnverifiedCompletedTransitions（回合内写间跃迁口径：首写=基线防多轮携带误报，之后新转 completed 须本回合 complete_step 收据）+停滞早停（reason 全文=签名，连续两轮同签名即停）+readiness 耗尽 return 挂 A4 交接块（第三挂点）；合成轮白名单补 [system] 门 nudge（A4 回扫此前把门 nudge 误当真人轮起点）；cap=3 保留（与上游 max=8 非同一对象）；上游式 grace 暂停/跨轮台账保全不做（gaea 会话全文落库+交接块=低配同构）。
- **B4 checkpoint/rewind v2**：维持等实弹痛点，不做。

## 五、观察池（C 档）

历史搜索目录 FTS5（派生数据可重建）；能力自检 doctor（静态只读诊断+稳定 issue 码+缺失降级为 info——运行监测板块延伸）；定时任务 scheduler（Trigger at/every+per-run 预算+信任门「信任是人的可撤回物」+固定只读姿态子进程隔离——设计精彩，gaea 需求待证）；redirectguard 统一重定向策略（HTTPS-only/不带凭据/host 白名单/maxHops=10）；session worktrees（git 项目会话隔离，gaea 非 coding 主场）；双模型 Coordinator（planner/executor 独立会话保前缀稳定）。

## 六、否定结论（不跟进）

- **MCP 全家/扩展 Runtime v2/插件包/主题市场**：平台化在册排除。
- **远程网关/手机网页/ACP**：跨设备在册排除。
- **计费/钱包/订阅估值**：gaea 无此产品面。
- **compositions/agent presets/persona**：Studio 前端预设系统（平台化）；gaea 人设=技能+绑定自有体系。
- **OS 级沙箱三平台**（seatbelt/bwrap/win_launch）：Windows 无靶子，gaea JobObject+boundedOutput+硬杀体系自洽。
- **SessionWriteAuthority/DAG 会话日志**：1.x 已否定沿用；2.x lineage 不改变单运行时判定。
- **memory v2 主体**：gaea 三态+DecayScore+subject keys 已领先（v4.378）；「召回只附加当前用户轮永不改前缀」已是 gaea pinned/relevant 二维现状。
- **egress loopback 代理**：netclient 已是完全体（1.x 结论沿用）；redirectguard 小件入观察池。

## 七、辨伪（2.x 报告中出现但 gaea 已有等价物，不立项）

PrefixShape/BodyChain 缓存归因（gaea `cache_shape.go` 9 月批落地，BodyChain 对照待一次复核）；live file observations/stale version（v4.375 刀2）；采样恢复冻结重放（v4.374 刀5）；pinned 正文注入/subject keys（v4.378）；skill 目录预算化（v4.376）。

## 八、拍板池建议序（若只做三批）

1. **安全两刀 A6+A7**（递归删除 guard+诚实拒绝）——纯增量零风险，一日双刀。
2. **压缩两刀 A1+A2**（usage anchor+内容级保留）——压缩线深化，接 v4.523 水位线；A3 验收经济学同批预核。
3. **子代理两刀 A10+A11**（write_paths 预检+完成裁决）——子代理强化线程 P2 收口。
