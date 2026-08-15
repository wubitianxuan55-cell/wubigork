# 审查报告：轻语与对话板块（review-whisper-chat）

## 高危
- H1 Orchestrator 会话状态机无并发保护（whisper_handler.go:44-46,109-111 只护 map 不护 orch；三个并发入口：chat_service.go:34 / whisper_state.go:58-72 微信 / voice_handler.go:110-121 语音）
- H2 WorkingMemory 无锁，异步 persist 与主流程并发读写 map → panic（working_memory.go:16-58；写方 whisper_handler.go:192，读方 :733/orchestrator.go:826；同族：AssociationIndex/HabitsStore/ActiveRecall）
- H3 跨会话持久化全表读-改-写，多会话并发互相覆盖（persistFactsToDB 764-795 + ReplaceFactsInDB 全表 DELETE+INSERT；SQLite 单连接只串行化单条语句）
- H4 最后一轮异步 LLM 记忆抽取几乎必然不落库（EnqueueMemoryWrite 与 persist 无先后约束；DrainAllMemoryWriteJobs 仅测试用）
- H5 WhisperClearSession 假清空（whisper_handler.go:452-458 只删内存；restoreWhisperState 全量恢复；DeleteCompanionStateFromDB/DeleteChatHistoryFromDB 存在未调用）
- H6 LLM 失败状态机已先行推进，重试双重计轮（whisper_handler.go:156 PreLLMTurn 提交，:180-184 失败无回滚）
- H7 语音播放无看门狗：PlaybackDone 丢失 → 永久卡 speaking（voice_manager.go:610-640 ackCh 无超时）
- H8 VAD+后端 ASR 下 thinking 阶段插话无效（voice_manager.go:228-230 StateThinking 丢弃；interruptTurn 仅浏览器识别可达）
- H9 跨会话包级节奏计数器串台（rhythm.go:26-29 包级全局；orchestrator.go:104-107 任一会话首轮 ResetRhythmState）

## 中危
- M1 ASR 失败/空识别后 VAD 不自动恢复监听（voice_manager.go:368-394）；M2 自动巩固空转（post_chat_turn.go:82-90 _ = pairs 丢弃）；
  M3 打断抢话内容丢弃；M4 TTS 慢合成不可取消（herdsman.go:63-69 180s）；M5 persona 双写两份存储；
  M6 WhisperDeleteFact/UpdateFact 只改内存；M7 每轮全表 FTS 重建+同步 trace 在热路径

## 低危
- L1 包级情绪涌现追踪死代码（emotional_emergence.go 无调用方）；L2 StreamASR 未接线含竞态；
  L3 GetState/GetFacts/SetEngine 无锁；L4 日志截断切非法 UTF-8（userMsg[:80]）；L5 shouldSearchWeb O(n²)；
  L6 毫秒时间戳话题 ID 碰撞；L7 记忆抽取 LLM 无 per-call 超时；L8 episodeEmotionMax 跨轮失效；L9 WriteErrors 无 UI 出口

## 亮点
- 异步错误回传链完整自洽（memory_write_job.go:75-89）；SQLite 单连接+WithTransaction 纪律；
  chat 存储事务化（store.go:205-310 单事务）；流式终态纪律（chat_service.go:97-158）；
  VoiceManager 状态机骨架成熟（打断累积/每句独立 ack/ctx 捕获）；情感→语音映射；Edge TTS WS 严谨

## 建议不做
1. 不做全局大锁/读写分离；2. 不启用 StreamASR；3. 不换 Edge TTS SDK；4. 不做跨会话定时批处理；
5. 暂不激活 MemoryConsolidator 全链路；6. 不升级 shouldSearchWeb

## 优先级
H1+H2（会话并发安全）> H3+H4（持久化正确性）> H5/H6 > H7/H8（语音活性）> M2/M3/M4
