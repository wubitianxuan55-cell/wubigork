# DSH (deepseek-harness) 0.1.6-alpha.2 蒸馏 —— 侦察与刀池（2026-09-21）

> **勘误（v4.374 同日）**：本档原题「Reasonix (dsh) 0.1.6 内核蒸馏」——**对象认定有误**。gaea 注释里的 Reasonix 真身是 GitHub **esengine/DeepSeek-Reasonix**（Go 单二进制终端 agent，V1.12/V1.15 端口出处），并非本仓 dsh（TS monorepo）。本档更名并保留：dsh 与 Reasonix 同属 DeepSeek harness 血统、机制互相借鉴，当轮五刀本身有效（v4.373.0 已发布），留池项仍按 dsh 口径有效。真身 Reasonix v1.15→v1.38 的蒸馏见 `docs/gaea-reasonix-v138-distill-2026-09.md`。**教训=蒸馏对象先经 GitHub 搜索核实真身，勿凭代码注释里的版本号猜本地克隆。**

- 源：`/c/AI/deepseek-harness`（dsh v0.1.6-alpha.2，上游同步 2026-09-19）
- 背景：gaea agent 内核曾蒸馏过 Reasonix 旧版（V1.12/V1.15 时代）的多组机制（repeatedSuccessBlock、malformed-args 回显、PrefixShape、cold_resume_prune、canonical todo、reasoning_language 等，见各源文件注释）。本次对 dsh 0.1.6 的 TS monorepo 做全量三域侦察（压缩/上下文、循环控制/护栏、子代理/计划/技能/记忆），对出真增量。
- 纪律：蒸馏=取道不取器；GoalCard 系已撤下勿再提；MCP/平台化/跨设备在册排除。

## 本轮已落地（v4.373.0，五刀，全 Go 内核，绑定面 707 零变更）

| 刀 | 源机制 | 落点 |
|---|---|---|
| 刀1 上下文溢出自愈 | compaction-basic `agent/request-error` CONTEXT_WINDOW_EXCEEDED → prune+强制压缩+有界重试，replaceGeneration 验证持久进展 | `provider.IsContextOverflow`（七措辞分类器，绝不误判输出 max_tokens）+ `AgentRunner.tryOverflowRecovery`（rewrite version 验进展；采样成功清零，连续上限 1；无进展如实放弃） |
| 刀2 缓存对齐摘要 | summarizer 复用会话 system+tools+被折叠区间消息原样重放为前缀，压缩指令作最后一条 user 消息 → 命中 provider 热 KV 缓存 | `compact.go summarize()` 重写（renderTranscript 平铺退役）；`currentSchemas()` 与 stream() 同源 |
| 刀3 shrink 硬校验 | region.ts：摘要定价必须 < 被替换区间，否则进恢复瀑布 | `compact()` 摘要后字符量校验，不达标退机械摘要——压缩永不膨胀 |
| 刀4 force 多轮压缩 | compaction-basic compactionRetries：压缩后重测，仍超再压一轮 | `compactIfOver` force 路径二轮（rewrite version 只计真实落地，空转不计入 compactStuck 熔断） |
| 刀5 repeat 分级提醒 | guard/repeat-tool-reminder：阈值 [3,5,8] 递进、deny 也计数、用户插话重置 | `repeat_detect.go` 三档（3 通用 / 5 点名工具+参数预览 500 上限 / 8 终止指令），4/6/7/9+ 沉默；`runDirect` 插话消费点重置链 |

测试：Go +6 文件级新例（`overflow_compact_test.go` ×5 + `TestRepeatNudgeTiers`）+ `TestIsContextOverflow`（provider）+ 既有快照断言迁移（summaryInstruction）。行为变更点：①摘要请求形状（测试 `TestSummarizeRequestCacheAlignedShape` 钉住前缀逐字稳定性）②repeat 4/6/7 不再逐步注入（原「≥3 每步注入」是上下文噪音）。

## 留池（下轮候选，按价值排序）

1. ~~**spill 主动泄洪**~~ → **已落地 v4.379.0（刀1）**：`internal/gaea/spill` 内存泄洪库+`executeOne` 压缩前捕获+`read_spill` 取回工具+`[agent] tool_spill` 配置门（子代理随父）。取道不取器：泄洪必须在按工具压缩/全局截断**之前**保原文（上游泄洪对象是最终文本，gaea 压缩管线更激进）；locator 前置是截断生存性结论（单行 JSON 信封头部裁切，尾缀必死）；存储取 runner 内存+64MB FIFO 驱逐（磁盘版唯一收益是重启幸存，不值无主痕迹文件卫生面）；豁免 read_file/read_spill/task；best-effort 纪律照单全收。
2. ~~**锚定式 token 计量**~~ → **已落地 v4.381.0**：`agent/tokenmeter.go`——压力=上次真实 usage 锚点+表面有符号差值；per-message 估价 memo（FNV-64，与 msgChars 同记账面）让差值只统计新增/删除/改写，未变消息锚点估价逐项相消（剪枝/压缩重写后不漂移）；落锚点=maybeCompact 收到真实 usage 时（表面恰=刚被 answered 的请求，无系统偏差）；负差值钳非负，无锚回退裸字符估算旧口径。EstimateContextTokens 委托计量表，mid-turn/compactIfOver/force 检查全过同一口径。
3. ~~**中断流结构化块保全**~~ → **已落地 v4.379.0（刀2）**：终态流错误/预算阻断路径 assistant（含 calls）+成对结果行落库（预执行只读用真实结果，其余合成「received but not executed」占位，信封 JSON 同约定）；恢复路径（重试采样）刻意不落 calls——落了就是悬空 assistant(tool_calls) 非法历史形态，两路径分流是关键辨析。
4. **子代理控制面**（`tool-subagent-control`）：send_message/interrupt_agent/双向消息/cold-resume/委派深度记账——gaea 有 continue_from 底座，补控制面是「发射后只能等」→「可纠偏可叫停」的一步。上游 09-16/17 活跃大改，等其稳定。
5. ~~**Fork 型子代理**~~ → **已落地 v4.380.0（刀1）**：task 新增 `fork` 参数，TaskTool 实现 ContextualTool 取父历史做种子（字节同源含父 system，不注模板提示——子请求前缀与父请求逐字节一致=KV 缓存继承）；组合禁忌 background/continue_from/retry_until；runSubSession 会话持有上移。顺带根修：V10.36 父对齐 schema 此前被 P0-1 回合重置抹成死代码（New 存 baseSchemas、重置恢复基线修复）。
6. ~~**子代理结构化输出**~~ → **已落地 v4.380.0（刀2 补完）**：原 output_schema 只在父级事后验（子代理不知道要产 JSON）——补 schema 注入子代理 prompt + terminal guard（非 JSON 同会话纠偏重入，热缓存成本极低，有界 2 次，耗尽诚实降级诊断前缀+原样）+ stripJSONFences 剥围栏。
7. **Plan mode 审批门**（`plan-mode`）：log-only 模式+exit_plan_mode+审批流（非 GoalCard 卡片，不触红线）；需前端 intent 渲染配套。
8. ~~**session-query 检索**~~ → **首刀已落地 v4.384.0**：`internal/gaea/sessionquery`+SchemaV24 FTS5 虚表——过往会话 user/assistant 正文全文索引（装配空间会话目录，空间隔离由目录构造），`session_search` 只读工具 newest-first+snippet；增量按 (size,mtime) 新鲜度惰性刷新零 boot 成本；中文 MATCH 零命中回退 LIKE 子串扫描+手工裁窗（whisper 先例）。未取：live-preferred 排序与谱系追踪（gaea 无多消费者语义；跨会话语谱系场景待真实需求）。
9. **session-projection-cache**：投影单元按 session 落盘冷启动加速（收益待评估）。

已销号（2026-09-21，v4.378.0 同日对账）：

- ~~⑨ timeout-policy TOOL_TIMEOUT 域隔离~~ → **辨伪销号**：gaea 工具超时是工具内自管（bash 300s 倒计时到期后以错误字符串「command timed out」返回模型，不取消外层 ctx），不存在「嵌套 deadline 被外层误判为用户取消」的共享通道——上游的 TOOL_TIMEOUT 哨兵错误域在 gaea 无靶子。
- ~~⑩ 技能目录 digest 重发布+热加载~~ → **辨伪销号**：与 reasonix watcher 热重载同判——技能目录在 system prompt 缓存稳定前缀内，mid-session 重发布必破前缀；新会话即得新目录，单用户桌面够用。
- ~~⑫ contextBreakdown 三分上下文构成~~ → **辨伪销号（已有等价物）**：gaea ContextView 折叠器按 catSystem/catTools/消息类目建节点（contextview/fold.go applyHeader：system prompt 与工具 schema 估算+变化才新增节点），构成视图已在面板可见——「前端可白嫖」的部分早就是现房。

## 侦察附注（不需要跟进的否定结论）

- dsh **无跨 provider fallback**（重试+压缩就是全部请求退化），gaea 无需补。
- dsh **无独立记忆子系统**（packages/memory 为空目录）；gaea 的 memory space+recall 领先，无需回蒸馏。
- Retry-After 优先：gaea `ParseRetryAfter`+`RetryPolicy.RetryAfterHeader` 已有等价物，不重做。
- maxConsecutiveWakes 自激励预算：gaea 无「任务完成自动唤醒开新回合」链路，暂无靶子；引入后台自动唤醒时必须配套。
- 工具中止合成结果配对：gaea executeBatch 对每个 call 必产出结果行（错误前缀/抑制占位），配对不变量已成立，不重做。
