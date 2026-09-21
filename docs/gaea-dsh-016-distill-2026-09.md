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

1. **spill 主动泄洪**（`packages/spill/spill-policy`）：大文本工具结果进上下文**之前**落盘全文，模型只见 head/tail 预览+locator 取回指引；`read` 工具豁免防循环。gaea 现有 PruneStaleToolResults 是压缩时被动归档，spill 把经济性提前到执行时。需要配套一个取回工具。
2. **锚定式 token 计量**（`packages/llm/token-meter`）：压力=上次真实 usage 锚点+当前表面有符号差值，剪枝/压缩重写后估计不再漂移。gaea 现为裸字符估算，需 per-message 估价缓存。
3. **中断流结构化块保全**（agent-loop step catch）：abort 时已收到的 tool-call 块随 partial assistant 消息持久化（gaea v4.26 只存文本）。
4. **子代理控制面**（`tool-subagent-control`）：send_message/interrupt_agent/双向消息/cold-resume/委派深度记账——gaea 有 continue_from 底座，补控制面是「发射后只能等」→「可纠偏可叫停」的一步。上游 09-16/17 活跃大改，等其稳定。
5. **Fork 型子代理**（`subagent-fork-in-process`）：父会话平衡前缀做种子，子代理带完整上下文开跑（~百行机制）。
6. **子代理结构化输出**（outputSchema+两阶段捕获+terminal guard）：批量编排结果可靠性。
7. **Plan mode 审批门**（`plan-mode`）：log-only 模式+exit_plan_mode+审批流（非 GoalCard 卡片，不触红线）；需前端 intent 渲染配套。
8. **session-query 检索**（FTS5+live-preferred corpus+谱系追踪）：模型可检索本机过往会话；与 memory space 互补。
9. **timeout-policy TOOL_TIMEOUT 域隔离**：区分本插件定时器与外层嵌套 deadline，防误判为普通取消。
10. **技能目录 digest 重发布+热加载**（`tool-skill`）：目录变了才重发完整替换；skill-filesystem watch 热加载。
11. **session-projection-cache**：投影单元按 session 落盘冷启动加速（收益待评估）。
12. **contextBreakdown 投影**：system/tools/message 三分上下文构成，前端可直接白嫖思路。

## 侦察附注（不需要跟进的否定结论）

- dsh **无跨 provider fallback**（重试+压缩就是全部请求退化），gaea 无需补。
- dsh **无独立记忆子系统**（packages/memory 为空目录）；gaea 的 memory space+recall 领先，无需回蒸馏。
- Retry-After 优先：gaea `ParseRetryAfter`+`RetryPolicy.RetryAfterHeader` 已有等价物，不重做。
- maxConsecutiveWakes 自激励预算：gaea 无「任务完成自动唤醒开新回合」链路，暂无靶子；引入后台自动唤醒时必须配套。
- 工具中止合成结果配对：gaea executeBatch 对每个 call 必产出结果行（错误前缀/抑制占位），配对不变量已成立，不重做。
