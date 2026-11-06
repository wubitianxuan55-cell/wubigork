# gaea 聊天板块核心优化 · 主计划（人格聊天卫生修复第一批）

- 立项：2026-11-05（执行 11-05 深夜~11-06）
- 依据：众神殿两场连议收官文档
  - 前半场 `pan_1793883599669_1119e3ff555f`
  - 续议场 `pan_1793885604229_e142a2a8d43e`（4 轮，用户已确认小结，sha256:0e05d48d…）
  - 议题：如何解决 gaea 使用角色（带人格）聊天时，角色胡乱说话、不作为，导致无法正常聊天
- 基准 HEAD：`8bc5804295c98a940bef7d10c2ba731df247155a`（论证全部锚点在本 HEAD 工具核验，执行前抽查无漂移）

---

## 一、资料整理：论证结论摘要

### 1.1 四类病因（收官共识，全部有 HEAD 代码实证）

| 病因 | 机制 | 关键锚点 |
|---|---|---|
| ① 来源污染 | 搜索增广串作为唯一 userMsg 喂 TurnPlan/PreLLMTurn/WM/FinalizeTurn/记忆摄入；动态块+搜索结果经 PromptHistory 以 user 角色跨轮重放 4~8 轮 | whisper_handler.go:771-781/:445-462/:476-487、orchestrator.go:479-499/:155-167、chat_search_adapter.go:90-104 |
| ② 错误压制 | 六条压制文案×五个代码位×三套触发谓词（CalcSilence 概率 / gate aff<30 / <10字钳制）+ ≤5字 EvtCold 误分类构成 Trust/Aff/Stage 负反馈闭环 | orchestrator.go:797/:799/:377/:379/:395、psyche.go:178、strategy_gate.go:13、interpreter.go:394-396/:541-543 |
| ③ 假成功 | 空正文兜成 [""] 落库并发 done；取消轮在 cancelled 判定前落库；空轮照常推 WM/PromptHistory/共享事件 | whisper_handler.go:397-422/:493-514、chat_service.go:414-426、chat/store.go:241-283 |
| ④ 连续性缺口 | PromptHistory 只在内存（重启即失忆，HistoryInMessages 模式下模型只看它）；orchestrator 恢复完成前已发布（并发首轮拿半成品） | whisper_handler.go:138/:148-162/:992-1051、orchestrator.go:126-146/:496-499/:958-964、params.go:103-109 |

另有确定性假事实源：**生日**——`orchestrator.go:439-441` 对 nil 生日无条件写当前时间，首聊日第 2 轮起必注入「我的生日」（hermes.db 6/6 会话 ackemBirthday==firstMetDate 实证；`ackem_birthday.go` 的 06-11 是零引用死常量）。

### 1.2 首批裁决（收官批准）

顺序：**冻基线 → 空正文/取消守卫 → 生日先删默认再迁移 → 入口 raw/llm 双通道 → 重启干净预热+恢复后发布 → 来源字段契约立项 → 五桶探针后定压制臂**。四个代码切片互相独立、可单独回滚；验收只看「问题被回答/文本任务完成/原句被接住/动作有回执/无虚假完成」，**不用平均回复长度**。

### 1.3 保留分歧（不阻塞首批）

- **DSH 模型因子**：本地 3B-A3B Q4 指令遵循上限未排除；生日修复后固定同一状态快照，加 deepseek-v4-flash 云端同题对照臂。
- **A′ vs 意图/分类臂**：纯删/软化压制文案 vs interpreter 祈使识别+统一意图门控；六条文案里「话少」类两处、「别开话题」类两处、「≤15字」一处、psyche silent 一处，须拆开测。两臂验收都必须含「注入命中后下一轮中性长问题」。
- **issue-1 多轮办事续接**：旧天气四轮链（明天天气→四川成都→我要天气→成都 全转城市介绍）是结构缺口（shouldSearchWeb 只扫当前句），升 P0 门槛=当前 HEAD 同角色复现。

### 1.4 已淘汰方案（勿再提）

继续强化人格/VoiceGuide/「人格优先级最高」（加重出戏且解释不了 silent=false 短答）；用现有 TurnTrace 逐条归因（Silent=CalcSilence、TierBChars=psycheBlock 长度、FactsUsed=活跃总数，均不能承担归因）；平均回复长度作验收；全角色统一 06-11；双通道=「隔离完成」宣称；三件套通过=主诉解决。

### 1.5 探针设计要点（后续实弹基线用）

- 五桶：问句 / ≤5字祈使 / 6–9字祈使 / >10字文本任务 / 寒暄；另加搜索组（现行触发词/旧误触发句/打断前缀）、记忆纠正、首聊日分层、15 轮连续闲聊、重启差分两问（事实回忆题应过＋「继续刚才那个」应挂=机制定位）、并发首轮、压制命中后下一轮。
- 记录字段：rawUserMsg/llmUserMsg 全文长度+哈希、event.Type、gate.Level、strategy level、CalcSilence、clamp、stage(old/new)、Aff、Trust、PrimaryLabel、实际注入事实 ID+domain、searchTriggered、shouldSearchWeb、emptyReply、cancelled、finalType、promptHistoryLen、RecentExchangesLen、WriteErrors、TTFT、ackemBirthday/firstMetDate/isBirthdayToday。
- 环境纪律：隔离 GAEA_DATA_ROOT+测试角色，禁触碰生产会话；语音 V2（voice_v2_turn.go:390-405 用原文）分入口验收，双通道修复不得宣称为语音已覆盖。

---

## 二、第一批执行（本批）

| 刀 | 内容 | 状态 |
|---|---|---|
| 刀0 | 探针字段基建（TurnTrace L3 扩展+whisperChatCore 收尾结构化日志）+ `chat-persona-baseline` tag | ✅ 13b65f37 |
| 刀1 | 空正文/取消守卫（whisperChatCore 有界重试+终态诚实；chat_service 落库移到 cancelled 判定后；AppendExchange 空正文防线） | ✅ cfff7a0f |
| 刀2 | 生日显式化（删 orchestrator.go:439-441 默认；gaea 显式 06-11；存量 6 行等值自愈+文件备份） | ✅ f429cecd |
| 刀3 | 搜索入口 raw/llm 双通道（raw 进状态机/WM/FinalizeTurn/记忆；llmUserMsg 只进 messages+PromptHistory，缓存契约不动） | ✅ a931d216 |
| 刀4 | 重启预热+发布序（restore 完成再发布；chat_history 净行 ≤4 轮回灌 PromptHistory） | ✅ 16b52dda |

执行补记（2026-11-06 凌晨，详见 docs/gaea-day-log-2026-11-06.md）：当场逮到三处真问题——
建连期取消走 callErr 早退被误报故障（原逻辑该分支在 cancelled 判断之前，连带
ChatStreamMessages 错误路径 cancel=nil 的 panic 缝）；sin 导出测试历史空行种数据
改走 AppendMessage 单条口；heal 测试初版漏「先恢复后自愈」生产序。新增测试 12 例
全绿，internal/app 全包 + internal/whisper 全包回归绿。

范围外（挂池）：压制文案两臂实验、多轮续接、有界工具循环、来源字段契约、PromptHistory 同进程存储形态、plain 流/语音 V2/微信出口改动。

## 三、后续批次路线

1. ~~实弹基线~~ ✅ 2026-11-06（docs/gaea-chat-probe-baseline-20261106.md）：刀2/刀4 决定性实证；新证据=Stage 不晋级致 whisper 46/46 恒中=真主靶。
2. ~~压制定臂~~ ✅ 臂B 四刀全落+实弹验证（docs/gaea-chat-probe-armb-20261106.md）：①Stage 晋级（中性轮计正向）②EvtCold 祈使/短问豁免+问句词表补「谁」③钳制豁免祈使/任务句④压制文案拆分（限长不禁话题）。实弹：Stage STRANGER→FAMILIAR、whisper 46/46→连续段 0、五桶正文 23→45 字、Trust 38.5→50.5、零回归。A′ 纯文案臂不再需要（实弹双重证伪）。
3. ~~来源字段契约~~ ✅ 2026-11-06（a859524c）：chat_history 行级 source="raw" 标记（Exchange.Source），四个写入点盖章（chat 主链/语音 V2/微信 agent）；persist 只写有标记行——遗留旧行零改写零误标；未知形态不回灌。无 schema 迁移（rows_json 无模式扩展），论证要求的副本演练迁移面不存在（旧数据一字不动）。动态上下文/搜索结果不再持久化（刀3/刀4 后），无来源面可标。
4. 多轮续接：做「待补槽位+一次合并检索」最小续接时，**指代消解（最近未完成任务优先）一并入设计**（armB 重启差分暴露「继续刚才那个」多候选歧义）。
5. 有界工具循环：出现「用户明确要求办事仍被当纯聊天」坏例后才设计。
