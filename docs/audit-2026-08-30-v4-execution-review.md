# v4.x 执行审计报告:规划承诺 vs 代码事实(2026-08-30)

> 触发:用户审查指出「规划被偷工减料」。本报告对路线图
> (`docs/gaea-nextgen-roadmap-2026.md`)对 v4.x 的**全部**承诺做逐项验证——
> 三路并行探查(阶段 0 地基 / 双空间内核+§15 补丁 / v4.1–v4.3 领域包),
> 每项验证到文件与行号,不采信发布说明的自述。
>
> **总裁决:不是「虚假宣传型」偷工——发布说明声称做了的基本都做了,且多处
> 主动标注缩水;真实形态是「最小版执行」:每一刀都立了骨架就奔下一个新面,
> 规划词汇在代码里的落点普遍是 1.0 最低档,纵深补课从未排期。**
> 已处置:路线图 §10.4a 插入 v4.5「指令中枢」重定跃升主轴(2026-08-30);
> 本文负责记录「已建内核的红线缺口」与「领域包纵深欠账」两张账。
>
> **补课状态（2026-08-30 v4.6.0）**：§B 红线缺口三条已全部接线（记忆注入
> InSpace 视图 / [tasks] 任务分账启用 / 事件订阅空间过滤推广，见
> releases/v4.6.0.md）；§C 纵深已落地 Mood→TTS 闭环、Verifier 通道 B 真视觉
> diff + 失败回 Plan、询价异常检测 + 价格预测 + OCR→询价库飞轮反向；§D 治理
> 收尾（keepAlive 8 处门控 + CSS 真硬编码 token 化）。未收欠账（规范包机制化 /
> 成本知识图谱+归因 / 生命库可写化评估）见 v4.6.0 欠账清单，排入下轮。

---

## A. 真做完的(含测试,经代码验证)

### 阶段 0 地基(S0.1–S0.6,五项全绿)

| 项 | 证据 |
|---|---|
| S0.1 AgentRunner 并发加固 | `internal/gaea/agent/agent.go:240` turnMu;execute_one.go:291-308/349-378 回合级 map 全部锁内读写(注释标 audit P0 race fix);`turnstate_race_test.go` 并发回归 |
| S0.5 `-race` CI | `.github/workflows/ci.yml:46-57` 独立 race job(agent/tool/control) |
| S0.2 Registry RWMutex + 幽灵名 | `internal/gaea/tool/tool.go:156-163` RWMutex;Add 早退防幽灵名(190-193);`registry_race_test.go` |
| S0.3 gate 撕裂换挡 | `agent.go:261` atomic.Pointer[gateWrapper],唯一读者 execute_one.go:67 Load;controller.go:709-739 mu 内换挡 |
| S0.4 retry_until 审批闸 | `agent/task.go:461-477` 统一 NewToolDispatcher.Check,拒绝即 errCheckBlocked 不执行;`task_retry_gate_test.go` 4 回归 |
| S0.6 edit_file 五件套 | `tool/builtin/{editfile,multiedit,editlines,movefile,grep}.go` 全部真实 Execute + 6 个测试文件;config.go:751-756 默认清单接入 |

### 双空间内核骨架(S1.1/S1.3/S1.5 + §15 大部分补丁)

| 项 | 证据 |
|---|---|
| S1.1 space 维度贯穿 | DB V14 迁移(facts/tasks space_id + 索引,`db/schema.go:338-345`);session 目录分区(`agent/session/space.go:20-25`);事件日志逐行带 space(`agent/session/log.go:94,399-406`) |
| S1.2 检索读端硬隔离 | `memory/sqlite.go:175,199` WHERE space_id 谓词;`app/brain_store.go:100-136` SearchInSpace(work 丢弃右脑);`app/gaea_unified_search.go:83-116` semantic 只做终过滤不动共享索引 |
| S1.3 模型/工具按空间 | `[space_profiles]` 7 功能域(config.go:139-146)装配期消费(boot.go:92-119);工具静态空间标签物理过滤(`tool/builtin/spacetags.go`,boot.go:403-405 装配期 AllowsSpace) |
| S1.5 审批策略按空间 | `config.go:201-235` PermissionsForSpace(play=allow+hard_ask 空=不弹卡);boot.go:205-208 接线;play 内容护栏 `app/play_guardrails.go` |
| ②子代理继承(fail-closed) | `agent/task.go:249/524/549-550` ctx 继承+会话空间≠ctx 即报错;`subagent_store.go:182-183` 续跑防穿越 |
| ③dream 空间化 | `app/gaea_dream.go:119,127` runDream(space);`controller_memory.go:282-333` 写侧空间落 facts |
| ④检索 scope | `GaeaUnifiedSearch(query, topN, scope...)`(变参保兼容);前端默认当前空间可显式「全部」 |
| ⑤[MEM:] 引用限定 | `memory/citations.go:48-64` ResolveCitations(text, space),跨空间=未知键静默 |
| ⑥产物路径分区 | `spaces/spaces.go:66-84`(play→.gaea/play/exports);四个消费点接线 |
| ⑦双首页 | ModuleLauncher 按 space 双翼装配(manifest.space 驱动,v4.3.2/v4.3.2b) |

### 领域包核心链路(骨架真实)

- **证据链(Apply→Verify→Journal)**:`internal/gaea/evidence/journal.go` ChangeRecord(含 8KB 原文摘要/模型/时间戳/基线快照);六类写盘工具 RecordChange+StageBaseline;xlsx_apply 补卡;前端 DeliverablesPanel。
- **Verifier 通道 A**:xlsx_apply 公式重算+引用/摘要比对,verdict 三档落盘(`app/gaea_verify.go`)。
- **AI 组价全链**:`cost/priceband.go` P25-P75 分位/置信度/证据;`app/gaea_cost_compose.go` 语义检索(bge-m3+reranker)→价格带→LLM 人材机拆解→回写。
- **五算对比(静态)**:`coststage/coststage.go` 五阶段值/环比差/三档偏差+规则诊断;前端 FiveCalcPanel。
- **询价存取**:`costinquiry/` 四源归一数据点+ListExpiring 到期预警+SuggestAdjustments 调差建议;`pricefeed` 30min cron 抓源。
- **关系图谱(骨架)**:`whisper/memory_graph.go` 三元组+倒排+QuerySubgraph BFS 多跳;BuildBlock 注入 PreLLMTurn;三表持久化重建。
- **情绪状态机**:`whisper/emotion.go` EmotionStep(调制/惯性/Mood EWMA);9 情绪→TTSParams(`voice/emotion_voice_map.go:137`)。

---

## B. 红线缺口(内核已建但未接线/未启用——优先补课)

1. **逐轮记忆注入主链路跨空间(违反「互不可见」红线)**。
   `memory/recall.go:94-97` RecallBlock 用 `Store.List()` 全量取数(`control/input.go:62` 每轮调用);`memory.Load` 的空间视图(`Options.Space`→`InSpace`,memory.go:35-39/space_view.go)**已实现但两个生产调用点(`control/controller_memory.go:240`、`boot/sysprompt.go:55`)均不传 Space**。即 work 会话每轮自动记忆注入与 system prompt 记忆索引仍是跨空间全量。
2. **任务按空间分账:内核已备、生产未启用**。`tasks.Options{}` 空构造(`app/gaea_tasks.go:48`),PerSpace 每空间信号量与 Priority 出队(tasks.go:120-135,690-741)没被配置;`jobs` 包无 space 概念。实际运行=全局 FIFO 旧行为。
3. **事件订阅空间过滤:机制已建、覆盖面 1 处**。`frontend/src/events.ts:98-112` subscribeForSpace 已实现,生产仅 `WhisperGraphPanel.tsx:234` 接入;主事件流(MainLayout:462,546 等)与任务事件(后端已带 spaceId,tasks.go:65)未按空间消费。

---

## C. 领域包纵深欠账(骨架已立,纵深零代码)

| 领域包 | 已落地的 1.0 | 欠账(规划承诺、代码零实现/严重缩水) |
|---|---|---|
| 办公信任链 v4.1 | 证据卡+基线+六类工具落卡;Verifier 通道 A | ①通道 B 是**页数对比非视觉 diff**(gaea_verify.go:107 仅比 /Type /Page 计数),且只对 xlsx_apply;②「失败回 Plan」未实现(fail 仅提示语);③中文规范包=单个硬编码红头 lint(`office/standard/redhead.go`,字符串启发式),无规范包机制/模板/造价工程表式;④证据卡缺「检索片段」维度;⑤无独立审计导出绑定(仅 markdown 投影) |
| 造价 AI 化 v4.2 | 组价全链;五算静态对比;询价存取+预警+调差建议;价格源 cron | ①**异常检测/价格预测零代码**;②OCR 报价单不自动入询价库(飞轮反向未接,gaea_cost_import_vision.go 无 inquiry 关联);③供应商比价仅是来源标签;④成本知识图谱+归因对标零实现(costref 仅 P25/P75 树状聚合);⑤「分类定位」缩水为两分类;⑥全过程成本动态(已发生+完工预测)无 |
| 乐园做深 v4.3 | 图谱骨架(三元组+BFS);情绪状态机+9 情绪 TTS 映射;参考图/img2img | ①图谱上无**情绪/因果/时空**维度(三元组主语几乎全为"用户",情绪活在图外 EmotionState);推理仅邻接遍历;「记忆回放」零代码;②长期心境 Mood **只存不用**——无任何 TTS 路径读 Mood,「听得出她今天低落」是原料非成品;映射为离散标签→静态预设,非连续韵律;③数字生命库可写化未实现(herdsman_digitallife.go 明确只读) |
| §8 革命性跃升 | 微信离线代办(v4.4.0);指令中枢一档(v4.5.0) | 端到端实时语音 0;语音控制桌面(JARVIS)能力面仅四类;微信任务入口 ~20%(无图片识别/文件卡片/通用能力触发) |

---

## D. 前端治理收尾(S0.7 剩余)

- 聊天虚拟化:等效达成(memo+尾部窗口方案,MessageList.tsx,有测试)。
- **hex token 化未收尾**:token 基建在(index.css 147 自定义属性/ThemeTokens),但中央定义外仍散落 ~300 处硬编码 hex(非测试 ts/tsx ~237 处/~40 文件,css 79 处;重灾区 WhisperTracePanel 23、WhisperEmotionPanel 22、RelationGraph 18、XlsxPreview 17)。
- **keepAlive 轮询门控覆盖 5/13**:已门控 MainLayout/ModuleLauncher/ResourceMonitor/useImageGenConfig/WeixinPage;仍裸奔 ~8 处(gaea 壳层 TaskCenter.tsx:107 2s、SubagentsPanel.tsx:67 5s、FeatureModelBar.tsx:32 8s、ProgrammingPage.tsx:160 3s、BenchmarkSection.tsx:105 8s、useStatsState.ts:45、useImageGenQueue.ts:286、useBridgeWatch.ts:44)。

---

## E. 补课刀序(与 §10.4a v4.5 刀序合并)

1. **v4.5.1a 双空间收尾补课刀**(红线+B 类,均为已建内核的接线):记忆注入接 `InSpace` 视图(两个调用点传 SessionSpace)→ 任务分账启用(PerSpace/Priority 配置)→ subscribeForSpace 推广到主事件流与任务事件。
2. **v4.5.1b**(既定):微信消息接统一路由 + iLink 图片/文件卡片协议。
3. **v4.5.2**(既定):Ctrl+K 命令面板接内核;顺带前端治理收尾(hex/裸轮询,机械化)。
4. **v4.6 及纵深刀序**(C 类按价值排序):Mood 进 TTS 闭环(原料已备,读路径一行链)→ Verifier 通道 B 真 diff + 失败回 Plan → 询价异常检测+OCR→询价库飞轮 → 规范包机制化 → 成本图谱+归因 → 生命库可写化评估。
5. **执行纪律修正**:每刀验收新增「纵深检查」——不许只立骨架;发布说明必须列「欠账清单」条目(沿用 v4.3.0 先例)。

---

## 附:审计方法与局限

- 方法:三路并行探查(阶段 0 六项 / 双空间 5 组 / 领域包 8 项),每项 grep 到实现文件与行号,不采信 CHANGELOG/releases 自述;i18n(zh-only)与编程板块独立窗口为用户拍板决策,不计入欠账。
- 局限:覆盖路线图 §14/§15 与 §1/§2/§7 的执行承诺;§3(AI 底座)/§5(小说)/§6(绘梦)的部分远期项(Story Graph 全书脑图、导演 Agent、上下文编译引擎、MCP host 等)未逐一验证——其中上下文编译/MCP host 属 T0 内核层排期,状态待下一轮审计补充。
