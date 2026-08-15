# 07 · 轻语域（internal/whisper）代码调研评审报告

> 调研对象：`C:\AI\wubigrok\internal\whisper`（232 文件 / 31,083 行，全项目最大单域）
> 调研方式：只读。先建文件全清单与职责归类，再精读核心文件（入口 handler、编排器、记忆管线、db 层、任务计划、语音链路、绑定层）。
> 改造背景：gaea（Wails v2 桌面应用，Go 后端）3.0 架构改造，原则级向 DeepSeek Harness 靠拢——会话事件日志作事实源、板块 Manifest 化、Provider Seam 化。
> 结论先行：whisper 在功能上已"缝合"进聊天板块（ChatSend 统一入口 + chat.db 话题存储），但在绑定层仍以 `VoiceB` 门面暴露全部 `Whisper*` 方法（命名与职责双重错位）；其记忆/会话存储（hermes.db）与聊天消息存储（chat.db）物理分离、且存在同一对话双写两份的事实冗余；LLM/ASR/TTS 三处切换点均为"可换"但未接口化收敛。

---

## 1. 概览（232 文件结构图 + 职责归类）

### 1.1 规模数据

| 维度 | 数值 | 证据 |
|---|---|---|
| Go 文件总数 | 232（全部为 .go，无资源文件） | glob `internal/whisper/**/*.go` |
| 总行数 | 31,083 | pwsh 逐文件行数统计 |
| 测试文件 | 22 个 / 4,733 行（占总行数 15%） | `*_test.go` 清单 |
| 非测试文件 | 210 个 / 26,350 行 | 同上 |
| 单文件最大 | orchestrator.go 894 行、types.go 531 行、interpreter.go 473 行 | 行数统计 Top |

子目录仅两层：`db/`（17 文件：database.go + 13 个 schema_vN.go + paths.go + 测试）与 `db/repos/`（13 文件：chat_history、companion_state、diary、episodes、fact_embeddings、fts、knowledge_triples、kv、memory_facts、openforu、procedural_habits、turn_traces + 测试）。**其余 ~200 个文件全部平铺在包根目录**，这是"232 个文件"的直接成因。

```
internal/whisper/                          # 包根：~200 文件平铺
├── orchestrator.go        (894)  # 回合编排：PreLLMTurn / TierA+B 组装
├── types.go               (531)  # FullState / Fact / Episode / Triple 等核心类型
├── interpreter.go         (473)  # L0 事件解释（event type 分类）
├── personality.go         (446)  # 30 预设 + 29 详细人格模板（PersonalityTemplates）
├── emotion_fusion.go      (384)  # 情绪融合
├── emotional_emergence.go (406)  # 情绪涌现
├── adult_mode.go          (253)  # 成人模式 FSM
├── desire.go / mirror.go / reunion.go / rhythm.go / canon.go ...  # 状态机子系统
├── memory_*.go            (~42)  # 记忆管线（ingest/write_job/consolidator/...）
├── agent_*.go / desktop_*.go (22)  # 桌面 agent 子系统（LLM 工具循环）
├── task_plan_*.go          (7)   # 任务计划
├── temporal_*.go           (7)   # 时间感知
├── wave_*.go               (3)   # 多波次对话
├── *prompt.go              (9)   # 提示词模板
├── llm_client.go / util.go / params.go / tracer.go  # 基础设施
└── *_test.go              (22)   # 测试
├── db/                           # SQLite 网关（hermes.db）
│   ├── database.go        (210)  # 单例连接池 + V1→V13 迁移 + whisper.db 旧库迁移
│   ├── paths.go                  # HermesDBFilename / DatabasePath
│   └── schema_v1.go .. schema_v13.go   # 13 步迁移（V11 FTS 独立化 / V12 追踪隔离 / V13 删死表）
└── db/repos/                     # 仓库层（每表一个文件）
    ├── fts.go             (322)  # FTS5 + LIKE 2-gram 中文降级
    ├── memory_facts.go    (313)  # 事实 CRUD（按 session 过滤）
    ├── companion_state.go / episodes.go / knowledge_triples.go / turn_traces.go
    ├── chat_history.go / diary.go / kv.go / fact_embeddings.go / procedural_habits.go / openforu.go
    └── *_test.go
```

### 1.2 为什么是 232 个文件（成因）

1. **ackem 直译移植**：大量文件头部注释为 `100% 对齐 ackem src/...ts`（如 `orchestrator.go:1`、`memory_write_job.go:2`、`memory_ingest.go:2`、`task_plan_loop.go:2`、`voice_manager.go:3`）。ackem 是 TypeScript 单体，模块即文件；Go 移植保持了一文件一模块的习惯，未做合并。
2. **状态机/子系统极细拆分**：情绪融合（emotion_fusion.go）、涌现（emotional_emergence.go）、成人模式（adult_mode.go）、欲望栈（desire.go）、镜像（mirror.go）、重逢（reunion.go）、节奏（rhythm.go）、时间感知（temporal_*.go ×7）、策略（strategy_*.go ×4）……每个可独立演进的机制一个文件。
3. **提示词单独成文件**：main_chat_prompt.go、memory_fact_extract_prompt.go、memory_consolidation_prompt.go、memory_contradiction_prompt.go、turn_plan_prompt.go、openforu_*_prompt.go ×4、diary_prompt.go 等 9 个 `*_prompt.go`（另有零散常量内嵌提示词，见 memory_ingest.go:18、memory_consolidator.go:161）。
4. **桌面 agent 子系统内嵌**：agent_*.go ×8 + desktop_*.go ×14 构成一个完整的"桌面助手 LLM agent 循环"（工具调用、任务计划、确认服务、机器映射），体量相当于一个独立域却寄生在 whisper 内。

### 1.3 职责归类（按文件名聚类，含代表性文件）

| 类别 | 文件数 | 代表性文件（均为行内证据源） |
|---|---|---|
| 编排/回合/状态机 | ~15 | orchestrator.go(894)、post_chat_turn.go、finalize_companion_reply.go、dispatch_router.go、paced_stream.go、wave_chat.go/wave_endpoint.go/wave_messages.go、turn_bubble_queue.go、deferred_context.go |
| 记忆管线（抽取/存储/检索/整合） | ~42 | memory_ingest.go、memory_write_job.go、memory_fact.go、memory_episode.go、memory_retrieve.go、memory_consolidator.go、memory_contradiction.go、memory_self_editor.go、memory_graph.go、vector_store.go、vector_search.go、active_recall.go、working_memory.go、fact_landing.go、findings_merge.go |
| 人格/情绪/关系/时间感知 | ~42 | personality.go(446)、emotion.go、emotion_fusion.go、emotional_emergence.go、relationship.go、desire.go、psyche.go、mirror.go、reunion.go、rhythm.go、temporal_*.go ×7、special_date_detector.go、holiday_detector.go、canon.go、adult_mode.go |
| 桌面 agent 子系统 | 22 | agent_loop_runner.go、agent_loop.go、agent_job_manager.go、agent_tool_batch.go、agent_tool_round.go、desktop_router.go、desktop_executor.go、desktop_capability_routing.go、desktop_actions.go、desktop_investigation.go、desktop_machine_map.go、machine_map_store.go |
| db 层（含 repos） | 31 | db/database.go、db/paths.go、db/schema_v1..v13.go、db/repos/*.go（fts、memory_facts、companion_state、episodes、turn_traces、chat_history 等） |
| 任务计划 | 7 | task_plan_store.go、task_plan_loop.go、task_plan_parse.go、task_plan_progress.go、task_plan_prompt.go、task_plan_resolve.go、task_plan_inject.go |
| 用户画像/档案 | ~8 | user_dossier.go、user_fact_guard.go、user_activity.go、user_presence.go、user_name.go、profiling_user.go、profiling_dimension.go |
| 提示词模板 | 9 | main_chat_prompt.go、memory_fact_extract_prompt.go、openforu_plan_prompt.go 等 |
| 检索/搜索 | 4 | document_search.go、search_query_resolver.go、web_search.go、semantic_reranker.go（另有 repos/fts.go） |
| 基础设施/工具 | ~10 | types.go(531)、llm_client.go、params.go、util.go、tracer.go、tool_def.go、tool_followup.go、sync_light_write.go、context.go、runtime_context.go |
| 测试 | 22 | whisper_core_test.go、whisper_memory_test.go、whisper_pipeline_test.go、memory_consolidator_test.go、agent_loop_runner_test.go 等 |

### 1.4 核心入口与关键文件（精读清单）

| 文件 | 行数 | 角色 |
|---|---|---|
| internal/app/whisper_handler.go | ~790 | 绑定层门面实现：WhisperChat / WhisperChatWithSearch / getOrCreateOrch / persistStateAsync / restoreWhisperState |
| internal/whisper/orchestrator.go | 894 | 回合编排：PreLLMTurn（L0 事件解释→L1 关系→L2 情绪→L3 表达→成人 FSM→涌现→Tier A/B 组装） |
| internal/whisper/memory_write_job.go | 189 | 异步记忆写入队列（每会话串行化） |
| internal/whisper/memory_ingest.go | ~289 | 记忆摄入管线（LLM 事实抽取→三元组→情节） |
| internal/whisper/memory_consolidator.go | ~180 | LLM 记忆整合（raw → consolidated 洞察） |
| internal/whisper/db/database.go | 210 | hermes.db 单例 + V1→V13 迁移链 + whisper.db 旧库迁移 |
| internal/whisper/db/repos/fts.go | ~322 | FTS5 全文检索 + 中文 LIKE 2-gram 降级 |
| internal/whisper/task_plan_store.go | ~260 | 任务计划存储（task_plan.json 原子落盘） |
| internal/voice/voice_manager.go | ~510 | 语音管道状态机（VAD/打断/ASR/TTS 编排） |
| internal/app/voice_handler.go | ~560 | App 层语音接线（ASR/TTS 引擎路由） |

---

## 2. 与聊天板块的边界

### 2.1 前端：ChatPage 对 whisper 的调用面（极小）

ChatPage.tsx 全文仅 3 处 `Whisper*` 引用（grep 结果）：

- `ChatPage.tsx:171` `await App.WhisperClearSession(activePersonality)`（切换人格时清内存会话）
- `ChatPage.tsx:257` `await App.WhisperClearSession(modeRef.current)`（清空对话时）
- `ChatPage.tsx:391` 注释："绑定模型条（聊天板块统一入口；whisper 为 chat 别名）"，模型条用 `FeatureModelBar feature="chat"`

**对话发送不直接走 Whisper 绑定**：plain 模式走 `App.ChatStreamPlain`（useChatStream.ts:120，事件流 `chat-stream:<runID>`）；人格模式走 `App.ChatSend`（useChatStream.ts:164）。语音入口 `useChatVoice.ts:47` 先 `App.VoiceApplySettings({personalityPresetId: getMode()})` 再启动；语音消息落库走 `App.ChatAppendMessages`（useChatVoice.ts:25）。

### 2.2 后端：ChatSend 是统一路由，whisper 是"模式"

`internal/app/chat_service.go:27-35`：

```go
func (a *App) ChatSend(topicID, message, mode string, ...) {
	if mode == "" || mode == "plain" { return a.chatSendPlain(...) }
	return a.chatSendPersona(topicID, message, mode, ...)   // 人格模式 = 轻语
}
```

`chatSendPersona`（chat_service.go:160-185）内部委托 `a.WhisperChatWithSearch` / `a.WhisperChat`，然后把 reply + 情绪元数据经 `appendChatExchange` 落库到 **chat.db**。结论：**在服务层，whisper 已经是"聊天板块内的一种人格模式"**，与前端心智一致（话题 `mode` 字段即 personaID，chat/store.go:13）。

### 2.3 绑定层：whisper 没有自己的门面，寄生在 VoiceB 里

`internal/app/bindings_manifest.go:7-23` 的 `NewBindings` 注册了 10 个门面：CoreB、OfficeB、MemoryB、CostB、ModelB、**VoiceB**、ChatB、NovelB、ImageB、CharlibB。其中：

- `bindings_voice.go:11-13` 注释自称"语音（TTS/ASR/轻语）绑定门面"，方法清单中 **30 个 `Whisper*` 方法**（WhisperChat、WhisperChatWithSearch、WhisperGetPersonalities、WhisperGetFacts、WhisperGetTraces、WhisperTaskPlanStatus、WhisperWeixin* 等）与 16 个 Voice/TTS 方法同驻一个门面。
- ChatB（bindings_chat.go:11-37）反而只暴露 `Chat*`/话题/主脑方法，**不含任何 Whisper 方法**。

即：前端"聊天板块"调用的语音+人格能力，前端绑定生成物来自 `VoiceB`——这是任务描述中"独立 VoiceB 门面"的具体形态。

### 2.4 存储边界：chat.db 与 hermes.db 物理分离

| 存储 | 文件 | 内容 | 证据 |
|---|---|---|---|
| 聊天话题/消息 | `{dataRoot}/whisper_data/chat.db` | chat_topics + chat_messages（role/content/extra/seq） | chat/db.go:34-35、chat/db.go:70-87 |
| 轻语记忆/状态 | `{dataRoot}/whisper_data/hermes.db` | companion_state、memory_facts、episodes、knowledge_triples、turn_traces、chat_history、fts 等 19+ 表 | db/paths.go:11-19、db/database.go:202-212 |
| 数据根 | `{DataRoot}/whisper_data` | app.go:243 `whisperDataRoot: filepath.Join(config.DataRoot(), "whisper_data")` | internal/app/app.go:243 |

两个连接池各自 `SetMaxOpenConns(1)`（chat/db.go:40、whisper/db/database.go:53）。**同一段对话被双写两份**：`chat.db.chat_messages`（appendChatExchange，chat_service.go:189-198）与 `hermes.db.chat_history`（persistWhisperStateWithSnapshot 里 `SaveChatHistoryToDB`，whisper_handler.go:767-779）同时落库，且 hermes.db 侧还有 `turn_traces` 留存每轮 L0-L4 快照（orchestrator.go:433-442）。

### 2.5 第三个入口：微信通道

whisper 除了 GUI 聊天板块与语音管道，还有**微信通道**入口（whisper_state.go:14-95）：`initWeixin` 加载 `assistant` 管理器（assistants.json），为每个启用助手启动 `weixin.Server`，回调内先注入自定义名（`orch.AssistantName`）再调 `WhisperChatWithSearch`（whisper_state.go:58-72）。会话过期经 `OnSessionExpired` 以 `gaea-event` notice 通知前端（whisper_state.go:76-87）。

**三个入口（GUI / 微信 / 语音）都汇聚到 `WhisperChat`**，因此 orchestrator 提供 `LockTurn/UnlockTurn` 串行化同一角色会话（orchestrator.go:86-87），app 层在回合前后持锁（whisper_handler.go:155-156），异步持久化用快照不持锁（whisper_handler.go:733-738）。

### 2.6 共享组件

- **LLM 客户端**：whisper 与 chat plain 共用 `a.client`（ai.ChatSimpleStreamDetailed），仅 systemPrompt 不同（chatPlainSystemPrompt，chat_service.go:18 vs orchestrator PreLLMTurn 组装）。
- **模型绑定**：`featureModel("chat")` 同时服务两条路（whisper_handler.go:174、chat_service.go:54/91）——"聊天/轻语合并后统一用 chat 绑定"。
- **联网搜索**：chat 与 whisper 共用 `whisper.WebSearch`（chat_service.go:21 别名注入；whisper_handler.go:493）。
- **微信通道**：`assistant` 管理器 + weixin.Server 由 whisperState 持有（whisper_state.go:14-95），回调仍走 WhisperChatWithSearch。

---

## 3. 记忆管线机制

### 3.1 管线全景（每回合异步触发）

`WhisperChat`（whisper_handler.go:136-260）回合结束后：

1. `whisper.EnqueueMemoryWrite(...)` 异步入队（whisper_handler.go:227-236）→ `memory_write_job.go:73-90` 按 **sessionID 串行化**（chan 令牌队列），不阻塞聊天。
2. `runMemoryWriteJob`（memory_write_job.go:92-130）→ `MemoryIngestPipeline.AfterTurn`（memory_ingest.go:74-98）四步：
   - **LLM 事实抽取**（memory_ingest.go:102-157）：`factExtractionPrompt`（memory_ingest.go:18-29）抽取 domain/subcategory/subject/summary/weight/confidence/selfRelevance → 经 `vetCreatorContradicting`（创造者矛盾过滤）与 `FilterExtractedUserFacts`（只从用户自述写入）→ `FactStore.Add`（FactLayer:"raw"）。
   - **自动退役**（memory_ingest.go:83-85）：`AutoRetire()` 按回合间隔执行。
   - **三元组提取**（memory_ingest.go:172-196）：按 subcategory 映射为 `(用户, 赞赏/表达脆弱/关系, ...)` 写 KnowledgeGraph。
   - **情节生成**（memory_ingest.go:200-242）：情绪强度超阈值时按间隔调用 LLM 生成 Episode 摘要（带 PrevEpisodeID 链）。
3. 主动遗忘（memory_write_job.go:16-20, 142-168）：命中"别提了/翻篇了/忘了这件事…"触发词 → 命中事实 `Sensitivity="avoid"`。
4. 持久化（whisper_handler.go:238 `go a.persistStateAsync(orch)`，726-790）：回合锁外快照（CloneFullState）→ companion_state / chat_history / facts / episodes / triples 分批落 hermes.db。

### 3.2 LLM 记忆整合（consolidated 层）

`memory_consolidator.go:25-125`：`Consolidate` 收集 `FactLayer==raw` 的最近事实（≤30 条，≥8 条才触发），LLM 合成高层洞察（`consolidationSystemZH`，memory_consolidator.go:161-184），写入 `FactLayer:"consolidated"`、weight 2.5、`DerivedFrom` 溯源。另有矛盾检测（memory_contradiction.go）、自我编辑（memory_self_editor.go）、关联索引（memory_assoc.go）等，构成完整记忆生命周期。

### 3.3 FTS 全文检索与中文降级

- 独立 FTS5 表（`schema_v11.go:11-23`：memory_facts_fts、episodes_fts，非 external-content 表，由 `RebuildFactsFTS`/增量 Insert/Delete 显式同步，fts.go:14-149）。
- 查询：`SearchFactIDsFTS`（fts.go:154-193）按词 `"word" OR ...` MATCH；**MATCH 失败或空结果时降级 LIKE 2-gram 子串匹配**（`buildLikePatterns`，fts.go:256-271），解决中文口语整词不命中问题。
- 接入点：whisper_handler.go:101-107 把 `repos.SearchFactIDsFTS` 注入 `orch.FTSSearch`（回调注入避免 db/repos 循环依赖）；orchestrator.go:856-863 在 Tier B 记忆块中 FTS 召回，命中事实权重 ×1.3。

### 3.4 角色记忆隔离

- 存储侧：facts/episodes/triples 均带 `SourceSessionID`（memory_ingest.go:150、memory_facts repo 按 session 过滤）。
- 恢复侧：`restoreWhisperState`（whisper_handler.go:657-706）只加载本会话 facts/episodes，KG 按 `ownFactIDs` 过滤三元组；`whisper_memory_isolation_test.go` 三组测试（TestWhisperMemoryRestore_IsolatedBySession / Persist_PreservesOtherSessions / Traces_IsolatedBySession）验证隔离与不覆盖。
- 会话标识：`sessionID = "whisper_" + personalityID`（whisper_handler.go:49），一个角色一个 Orchestrator 实例，全局 map `whisperSessions`（whisper_handler.go:44-46）。

### 3.5 与办公引擎 memory（internal/gaea/memory）的区别

| 维度 | whisper 记忆 | gaea/memory（办公引擎） |
|---|---|---|
| 形态 | SQLite 结构化事实（memory_facts/episodes/triples） | 文档记忆（AGENTS.md 等层级 doc）+ 自动记忆 store（memory.go:18-26） |
| 存储 | hermes.db（modernc.org/sqlite，MaxOpenConns(1)） | Hephaestus.db 或文件后端（Options.DB，memory.go:31-35） |
| 检索 | FTS5 + LIKE 2-gram + 触发词 + 向量 | 内存倒排索引 + 语义检索（Search/BuildSearchIndex） |
| 语义 | 记忆"用户这个人"（画像/关系/情绪） | 记忆"项目/会话约定"（约定/文档/快速记忆） |
| LLM 消费 | Tier B 上下文块注入（orchestrator.go:838+） | Compose 折叠进系统提示词 |
| 隔离 | 按 sessionID（角色隔离） | 按 scope（user-global / project / local） |

两者无代码复用，是两套并行的记忆体系（whisper 域自建 db 层，gaea/memory 域自建 store 层），这是 3.0 合并/收敛的候选点。

---

## 4. 语音链路

### 4.1 管道与状态机

`internal/voice/voice_manager.go`：状态机 `idle → listening → thinking → speaking → idle`（voice_manager.go:6-9），管道 `音频输入 → VAD → ASR → Whisper 引擎 → TTS → 音频输出`（voice_manager.go:10）。

- **VAD**：RMS 能量检测 + 自适应噪声底噪（`updateNoiseFloor`，voice_manager.go:344-354）、连续帧确认（MinSpeechFrames）、静音超阈值/单轮超时触发识别（voice_manager.go:305-315）。
- **打断（barge-in）**：speaking 态累积 `interruptSpeechMs` ≥ 阈值（默认 500ms）触发 `handleInterrupt`（voice_manager.go:232-256）；thinking 阶段插话经 `turnCancelCh` 跳过过期回复（voice_manager.go:464-473）。
- **回合串行**：`turnMu` 串行化所有输入入口（voice_manager.go:398-402）。

### 4.2 引擎接入方式

| 环节 | 接入形态 | 证据 |
|---|---|---|
| ASR | `asr.NewHerdsmanASR(eng.BaseURL, model)` 单实现（herdsman whisper-base/funasr/zipformer），经模型中心引擎路由选择 | voice_handler.go:128-171（applyASRClient 三级优先级：用户 STT 模型→扫描引擎 STT 模型→默认 whisper-base）；`isSTTModel`（voice_handler.go:166-171） |
| 对话 | `SetWhisperChatFn` 注入 `WhisperChatWithSearch`（voice_handler.go:109-122），语音也走人格+搜索增强 | voice_manager.go:113-115、voice_handler.go:110-121 |
| TTS | `SetTTSSynthesizeFn` 注入 `synthesizeVoiceTTS`（voice_handler.go:101-103），内部三级路由：功能绑定"聊天语音"模型 → 全局 TTS 模型 → TTSSpeakBase64 统一路由（Edge/SAPI/Herdsman/xAI/cosyvoice） | voice_handler.go:175-193、internal/tts/{edge,sapi,herdsman,xai}.go |
| 浏览器 ASR | `VoiceChatText` 直接入管道跳过后端 ASR | voice_handler.go:296-305 |

### 4.3 事件发射

App 层 `voiceEmitter` 把状态机事件桥接到 Wails 前端（voice_handler.go:44-78）：`voice:state`、`voice:transcript`、`voice:tts-audio`、`voice:tts-speak-text`、`voice:tts-speak-cancel`、`voice:listening`、`voice:thinking`、`voice:error`、`voice:reply`。前端 useVoiceChat hook 消费；语音消息经 `ChatAppendMessages` 落 chat.db（useChatVoice.ts:25）。

### 4.4 语音状态与 App 生命周期

`mediaState`（含 voiceManager）在 Startup 时 `initVoice()`（voice_handler.go:83-106）；`VoiceRestartService` 可停/重启管道重检引擎可用性（voice_handler.go:519-525）。

---

## 5. 特色机制（taskplan / 可观测性）

### 5.1 whisper_taskplan（任务计划）

- 存储：`task_plan_store.go`——`TaskPlanStore` 内存 map + 原子落盘 `{whisperDataRoot}/task_plan.json`（`fileutil.AtomicWrite`，task_plan_store.go:236）；仅保存 status=active 计划（Save 完成态不落盘，task_plan_store.go:115-129）；重启经 `ReloadFromDisk` 恢复（task_plan_store.go:176-205）。
- 装配：app 层进程级惰性单例 `taskPlanStore(dataRoot)`（whisper_taskplan.go:22-32）；绑定 `WhisperTaskPlanStatus`（whisper_taskplan.go:36-62）供前端面板轮询；`WhisperTaskPlanResume`（whisper_taskplan.go:68-79）恢复入口（注释自认"尚无 orchestrator 级继续机制接线，仅标记 active + 落盘"）。
- 循环门控：`task_plan_loop.go` `CheckTaskPlanLoop`（task_plan_loop.go:17-62）判定 allPassed / 超轮数 / 无待执行步骤；`BuildTaskPlanNudge` 生成继续执行提示注入（task_plan_loop.go:89-107）。
- 继续意图：`IsContinueTaskPlanIntent` 正则（"继续/接着/完成剩余/做完…"，task_plan_store.go:244-253）。
- 执行器：desktop agent 子系统 `AgentLoopRunner.RunAgentLoop`（agent_loop_runner.go:42-80）——LLM 驱动多轮循环 + 工具批次执行 + 结果反馈 + 验收（verify 规则，task_plan_store.go:26-33）。

### 5.2 whisper_write_observability（写作/异步写可观测性）

`internal/app/whisper_write_observability.go:18-54`：`whisperWriteErrors` 原子计数 + 最近错误摘要，经 `MemoryWriteErrorSink`（memory_write_job.go:67）回传 LLM 抽取失败 / JSON 解析失败 / 落库失败 / panic（whisper_handler.go:227-235 注入 `a.recordMemoryWriteError`）。**刻意不新增 Wails 绑定**（whisper_write_observability.go:15、53），仅内部诊断。

### 5.3 桌面 agent 子系统（内嵌的"第二引擎"）

whisper 内嵌了一套完整 LLM agent 循环（agent_*.go + desktop_*.go ×22），与聊天记忆无关，职责是桌面任务执行：

- `AgentLoopRunner.RunAgentLoop`（agent_loop_runner.go:42-80）：LLM 驱动多轮循环，每轮 `callLLM` → 无工具调用即完成；有工具调用则 `executeToolBatch` 批执行（agent_tool_batch.go）→ 结果反馈下一轮。
- 工具面：`desktop_actions.go` / `desktop_executor.go`（372 行）/ `desktop_router.go` / `desktop_capability_routing.go`，配合 `machine_map_*`（机器地图：动态发现可用桌面能力）。
- 配套：`ConfirmService`（确认服务，桌面_confirm_bypass.go 可绕过）、`DeliveryCoordinator`（消息分发）、`desktop_audit_log.go`（审计日志）。
- 任务计划是这套 agent 的产物之一（`AgentLoopResult.TaskPlan`，agent_loop_runner.go:29），经 TaskPlanStore 落盘（第 5.1 节）。

该子系统与聊天/记忆无强耦合，是 3.0 拆分为独立板块（或并入办公/工具板块）的首选候选。

### 5.4 其他特色机制（简列）

- **TurnTrace 轮次追踪**：每回合 L0-L4 结构化快照，落 `turn_traces`（SchemaV12 后带 session_id 隔离，schema_v12.go:5-7），角色库"追踪"页数据源（whisper_handler.go:215-217, 318-328）。
- **wave 多波次对话**：wave_chat.go 按情绪强度决定 1-4 波回复（WavePlan/WaveCount），对齐 ackem waveChat.ts。
- **dispatch_router**：话题/意图路由（dispatch_router.go:256 行）。
- **desktop audit log / machine map**：桌面 agent 审计日志（desktop_audit_log.go）与"机器地图"（machine_map_collector/indexer/store）用于桌面操作能力的动态发现。

---

## 6. 与 3.0 目标相关的关键发现（Manifest 归属评估）

### 6.1 板块 Manifest 化：whisper 应归入 chat 板块（高置信）

现状评估（证据链）：

1. **前端**：ChatPage 是唯一宿主，人格模式即话题 `mode=personaID`，模型条 feature="chat"（ChatPage.tsx:391）。
2. **服务层**：ChatSend 统一路由，persona 模式内部调 WhisperChat（chat_service.go:27-35, 160-185）。
3. **绑定层**：whisper 全部方法却暴露在 **VoiceB** 下（bindings_voice.go:33-59），与 ChatB 无关（bindings_chat.go）。

结论：**功能上 whisper 已是"聊天板块内的模式"，绑定层归属是历史遗留的错位**。3.0 板块 Manifest 化时应把 `Whisper*` 方法从 VoiceB 迁出：语音管道方法（Voice*）留在语音板块（或归入 chat 板块的"语音输入"能力），whisper 状态/记忆/任务计划方法并入 ChatB（或新建 `WhisperB` 注册进 `NewBindings`，bindings_manifest.go:11-22）。这是纯绑定面重组，方法体零改动（现门面即"纯委托"模式，bindings_voice.go:11 注释），迁移成本低、风险小。

### 6.2 会话/记忆存储是否受 Step 1 事件日志影响

Step 1（会话事件日志作事实源）对 whisper 的影响评估：

- whisper **当前没有事件日志**：不 import gaea/event（grep 证实）；`shared_events` 表存在（schema_v2.go:51-56）但仅建表、无写入代码；每回合的"事实源"是 `turn_traces`（L0-L4 快照，orchestrator.go:433-442）与 `chat_history`。
- `FullState`（companion_state 持久化）是**大 blob 快照**（关系/情绪/计数器/画像/欲望栈，state_persistence.go:57-78），状态机（成人 FSM、情绪涌现、性格漂移）依赖进程内连续计数（orchestrator.go 中 adultBudget/recentEventTypes 等私有字段**不进快照**，orchestrator.go:57-66）。
- 结论：**Step 1 事件日志可作为新的"事实源"并行建立（替代/补充 turn_traces），但 whisper 的状态机无法简单从事件重放**——重启恢复依赖 companion_state 快照而非事件。建议：事件日志先做可观测性与审计用途（对齐 turn_traces 的 L0-L4 语义），状态恢复继续走快照；**不要在 Step 1 就强制 whisper 改为事件溯源**（其 L0-L4 事件类型已定义在 interpreter.go，未来可渐进映射）。
- 存储保持独立是低风险的：hermes.db 与 chat.db 均物理独立（第 2.4 节），事件日志若建在 office 引擎侧，需要明确 whisper 回合是否也要写事件（建议写，经 gaea/event 总线发布 whisper 事件，如 voice:state 已是先例——但那是 app 层 emit 而非事件存储）。

### 6.3 Provider Seam 化目标清单（切换点盘点）

| 切换点 | 现状 | 已存在的 seam | 缺口 |
|---|---|---|---|
| LLM 引擎 | 模型中心 engine/model 路由 | `LlmClient` 接口（llm_client.go:10-13，仅 `Chat` 方法）；app 层 `featureModel("chat")` + `routeModel`（whisper_handler.go:23, 174-182）；orch 持有 EngineID/ModelName（orchestrator.go:46-47） | 接口过薄（无流式/思考参数，语音管道与聊天主链路不走 LlmClient 而是直连 a.client）；主对话 LLM 调用在 app 层硬编码 `a.client.ChatSimpleStreamDetailed`（whisper_handler.go:189），未接口化 |
| ASR 引擎 | 仅 herdsman 单实现 | `asr.HerdsmanASR`（asr/herdsman_asr.go + _stream）经 BaseURL+model 适配任何 OpenAI 兼容端点；模型中心 STT 路由（voice_handler.go:128-163） | 无 `ASREngine` 接口，绑定死 `*asr.HerdsmanASR`（voice_manager.go:74）；浏览器 Web Speech 作为第二实现绕过后端（VoiceChatText） |
| TTS 引擎 | 多实现：Herdsman / Edge / SAPI / xAI / cosyvoice | `tts` 包内每引擎独立类型 + `SynthesizeWithMime` 同签名；app 层 `tryEngineTTS`（voice_handler.go:197-235）按 engineID 分派 | 无统一 `Synthesizer` 接口，分派逻辑散落在 voice_handler.go 与 tts_service.go（TTSSpeakBase64 路由）；cosyvoice 需进程拉起（voice_handler.go:206-208） |
| 语音管道 | 状态机固定 | `voice.Manager` 注入式回调（WhisperChatFn/TTSSynthesizeFn/ASRClient，voice_manager.go:107-120） | 回调是函数签名而非引擎接口，跨引擎能力协商（如音色描述/克隆）缺失 |

改造建议：以 `LlmClient`、`ASREngine`、`TTSSynthesizer` 三个接口收敛切换点，app 层只做装配；`voice.Manager` 的回调改为接口注入。

---

## 7. 缺陷与风险

1. **绑定层命名与职责错位**（P1）：Whisper 方法挂在 VoiceB 下（bindings_voice.go:33-59），前端生成物与板块心智不一致，3.0 Manifest 化必须处理（见 6.1）。
2. **同一对话双写冗余**（P1）：chat.db.chat_messages 与 hermes.db.chat_history 同存一份交换（chat_service.go:189 vs whisper_handler.go:767-779），数据不一致风险（如"清空话题"只删 chat.db，hermes.db 历史仍在——ChatTopicClear 走 chatStore.ClearMessages，chat/store.go:155-165，不动 hermes.db）。
3. **会话生命周期不收敛**（P2）：`whisperSessions` 全局 map 按角色常驻，仅 WhisperClearSession 删除（whisper_handler.go:461-467）；每回合 2 个后台协程（记忆写入 + 持久化，whisper_handler.go:219-238）无上限控制，多角色长会话下内存/DB 写放大。
4. **状态机不可重放**（P2）：FullState 快照丢失进程内连续状态（成人预算/涌现计数/recentEventTypes 在 orchestrator.go:57-66 不进 DB），崩溃恢复后状态机语义偏移；无事件溯源能力（见 6.2）。
5. **单包平铺 232 文件**（P2）：除 db/ 外无子包边界，desktop agent、temporal、memory 等子系统均混在包根；同包内互相可见，无法强制分层（如 memory 管线可直接调用 desktop 工具）。
6. **VAD/打断调参面大**：语音状态机阈值（MinSpeechFrames、SilenceThresholdMs、InterruptThresholdMs）散在 voice 包常量与 config，无统一校准入口（voice_manager.go:291-315）。
7. **记忆 LLM 抽取为尽力而为**：事实抽取失败仅记日志 + 计数（memory_ingest.go:110-113），用户关键信息可能静默丢失；无重试/补偿机制。
8. **两套记忆体系并存**：whisper 记忆（hermes.db）与办公引擎记忆（Hephaestus.db + doc）零复用（第 3.5 节），语义检索/向量能力重复建设（vector_store.go vs gaea/retrieval）。
9. **db 层连接数**：chat.db 与 hermes.db 各自 MaxOpenConns(1)，两个文件同一数据根下并行打开，备份/迁移需分别处理（gaea_data_backup.go 把 whisper_data 整体打包）。
10. **成人模式默认开启**：`AdultMode: true` 为产品决策（orchestrator.go:106），内容策略（敏感度/隐私等级）内嵌在记忆写入路径（memory_write_job.go:132-138），3.0 需明确内容治理边界。

---

## 8. 改造建议（按 3.0 三原则映射）

### 8.1 板块 Manifest 化（chat 板块吸纳 whisper）

1. **绑定面重组**（Step A，纯搬移零逻辑改动）：把 bindings_voice.go 中 30 个 `Whisper*` 方法迁入 ChatB（bindings_chat.go）或新建 WhisperB 并注册进 NewBindings（bindings_manifest.go:11-22）；VoiceB 只留语音管道方法。同步重生成 wailsjs 绑定。
2. **Manifest 声明**：chat 板块 manifest 声明能力列表：plain 对话（流式）、persona 对话（whisper 引擎，含记忆/情绪/任务计划）、语音输入输出（voice）、角色库联动；whisper 的 29 人格模板（personality.go:131）与角色库（characterlib）作为数据资产注入。
3. **服务层保持**：ChatSend 统一路由已是正确形态（chat_service.go:27-35），无需改动；建议把 `WhisperChatWithSearch` 显式暴露为 `chat.PersonaChat` 别名以便追踪。

### 8.2 会话事件日志（Step 1）的落地方式

1. **并行建立事件日志，不推倒状态机**：新增 whisper 回合事件（chat/whisper 域）写入事件存储，语义对齐 turn_traces 的 L0-L4（interpreter.go 已有事件类型定义）；hermes.db companion_state 快照继续作为恢复源。
2. **消除双写**：以 chat.db 为消息事实源，hermes.db.chat_history 改为派生/删除（或事件日志替代），"清空话题/导出/重放"统一走 chat 存储。
3. **可观测性增强**：把 whisper_write_observability 的 WriteErrors 升级为事件（whisper_write_observability.go:18-54 目前仅内存计数）。

### 8.3 Provider Seam 化

1. 定义 `LlmClient` 增强接口（流式 + 思考参数），让 orchestrator 主链路与记忆后台共用同一 seam；引擎/模型解析收敛到 `featureModel`（whisper_handler.go:174）。
2. 新增 `ASREngine` 接口（asr 包），voice.Manager 依赖接口而非 `*HerdsmanASR`（voice_manager.go:74）；浏览器 Web Speech 实现同一接口。
3. 新增 `TTSSynthesizer` 统一接口（tts 包），Herdsman/Edge/SAPI/xAI/cosyvoice 各自实现，删除 voice_handler.go:197-235 的 if-else 分派。
4. 语音管道保持注入式（voice_manager.go:107-120）但注入对象改为接口。

### 8.4 结构性优化（中远期）

- whisper 包按子系统拆子包（memory/、temporal/、desktopagent/、db/ 已有），打破 232 文件平铺；先拆无依赖环的 memory 与 db（db 已是子包）。
- 记忆体系收敛：评估 whisper 记忆与 gaea/memory 共用检索/向量基础设施（vector_store.go 与 gaea/retrieval）。
- 会话生命周期治理：Orchestrator 池化/闲置回收，后台协程限流（memory write queue 已串行化，持久化协程可合并为单飞）。

---

*（本报告为只读调研产物；所有路径相对仓库根 C:\AI\wubigrok，行号以调研当日文件为准。）*
