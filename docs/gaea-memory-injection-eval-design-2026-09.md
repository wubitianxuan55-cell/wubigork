# 记忆注入质量评测 设计（2026-09-12，市场调研候选2）

> 状态：🔄 本刀实现。上游=docs/gaea-market-survey-2026-09.md §5 候选2。
> 与检索评测分立：`GaeaRetrievalEvalRun`（v4.196，Recall@10≥0.8 统计门槛）测的是
> **检索面**（查询→四库命中）；本刀测 **注入面**——work 会话装配时真正进系统
> 提示词的两个记忆块（晨报预载 `BuildMorningPreloadBlock` + 项目本体
> `BuildProjectBrief`，v4.16/6.2）是否把「该进的进、不该进的挡在外」。
> LongMemEval 的 gaea 化：不追 LLM 答卷，追装配不变量——注入是确定性纯函数，
> 可体检、可门禁，零 LLM 零 embedding，跑用户**真实库**（与检索评测同哲学）。

## 口径：五条不变量 + 一个指标面

对真实库的 work 视图跑**真实构建器**（与 boot/sysprompt.go 装配点同函数、同预算、
同门控），对产物块断言：

| # | 不变量 | 判定 | 违规后果 |
|---|---|---|---|
| 1 | 预载块预算合规 | runes ≤ DefaultMorningPreloadBudget(600) | violation |
| 2 | 本体块预算合规 | runes ≤ projectBriefBudgetRunes(600) | violation |
| 3 | 归档/跨空间零泄漏（**精确路径**） | 预载条目名/本体引用键逐一核对：不得是归档名、不得是他空间名（块全文的子串兜底**不做**——work 条目正文合法提及归档名不是泄漏，子串匹配必误报） | violation |
| 4 | 引用零悬空 | brief 的 [MEM:x] 与预载行条目名必须在 work 视图内（悬空注入=模型会引用不存在的键） | violation |

指标面（信息展示，不计 Passed）：条目数 / 引用数 / 固化覆盖（PinnedTotal /
PinnedInBrief / MissingPinned）——固化未全收在预算紧张时是设计内行为
（「预算内按行完整收录」），只透出不判死。

**门槛**：Passed = 零 violation（确定性规则无统计门槛；与检索评测 0.8 分立，面板注明）。
**门控透明**：memory.enabled / morning_preload / project_brief 三开关随报告下发，
关态下 passed=true 但 Note 注明「注入关闭，体检仅覆盖结构不变量」。

## 形态

- Go 纯核 `memory.EvalInjectionBlocks`（internal/gaea/memory/injection_eval.go）：
  输入=work 视图/他空间名集/归档名集/两个真实块/预算，输出=报告；不读时钟不 IO。
- App 绑定 `GaeaMemoryEvalRun`（internal/app/gaea_memory_eval.go）：hubOfficeStore
  → 空间视图（SpaceModeIsOn ? ListInSpace("work") : List()，与装配点同口径）
  → 真实构建器 → 纯核。**绑定 625→626**（gen_bindings 再生）。
- 前端：办公记忆库面板工具栏「注入体检」钮 → Drawer 内 MemoryEvalSection
  （渐进披露，不加一级房间，UI 简化红线）；报告类型进 lib/types/memory.ts；
  前端绑定锁 447→448（spaceBindings.test）。

## 非目标

- 不做注入内容的语义质量评分（那需要 LLM 评审，另刀）；不追召回率统计
  （检索面已建制）；不改装配点任何行为（零 Go 行为变更，纯读路径）。
