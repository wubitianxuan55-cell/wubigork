# 语音管线复刻审计台账（REA 纪律：义务须锚上游 file:line 原文）

- 状态：**已收口（批1~批4 全部完成，2026-11-05）**
- 方法：逐文件读上游原文（`.tmp/research-hf-sts/src/speech_to_speech/`，6feb527）→ 义务条目（ID/证据引用/引文/Go 现状/判罚）→ 差距修复（一处一刀：build+单测+ws_probe）
- 判罚：✅对齐 / ⚠️偏差（需修）/ ❌缺失（需补）/ 🔵上游死代码或 gaea 未用（不移植，记理由）
- 上游文件清单与进度：
  1. `api/openai_realtime/websocket_router.py` — ✅已读（发送循环/停靠/代际闸已移植）
  2. `api/openai_realtime/handlers/response.py` — ✅已读（finish_response 守卫/key-vs-current）
  3. `pipeline/speculative_turns.py` — ✅已读（9 单测）
  4. `pipeline/turn_latency.py` — ✅已读（日志行逐字）
  5. `STT/base_stt_handler.py` — ✅已读（输入门/2048 集）
  6. `VAD/vad_handler.py`+`vad_iterator.py`+`smart_turn.py` — ✅已读
  7. `api/openai_realtime/service.py` — ✅已读（2026-11-05 批1 逐行：dispatch 顺序/归属解析/终态收口）
  8. `pipeline/history.py` + `api/openai_realtime/handlers/history.py` — ✅已读（批2：hold/accept/reject/close/finalize 语义）
  9. `pipeline/messages.py` Chat 类 — ✅已读（批3：chat_size 裁剪/trim_if_needed/硬上限 2×size 逐出）
  10. `LLM/language_model.py` 历史写回 — ✅已读（批4：快照/anchor/capture；chat_completions_* 为 provider 适配器面 n/a）
  11. `api/openai_realtime/handlers/conversation.py`+`audio.py` — ✅已读（批4：全量逐行）

## 审计批次

- 批1 已完成（#7 service.py 逐行+S1~S20 判罚+F1~F6 修复，见下）
- 批2 已完成（#8 history 两文件，O1~O4；**O2/O3 修复为假绿——批1 复核证伪后补真修**）
- 批3 已完成（O7 修复+O8 记理由）
- 批4：#10/#11 补读收口（待续）
- 每批产出：义务表+判罚+修复提交

## 义务条目（批1：service.py 已逐行读完）

### 批1 证据（原文锚点）
- service.py:328-352 mark/clear/close response key（pending 集+128 墓碑+response_pending 旗）
- service.py:407-422 register：新连接重置 speculative turns+**新建 Chat（每连接独立历史）**
- service.py:424-443 unregister：history.close_session+chat.close()（抑制在途压缩/后续计费）+用量卷积+latency 清理
- service.py:562-593 close_pending_responses/close_response_key：逐 pending key 收口（用量保留+tracker discard+history.close+墓碑）
- service.py:610-669 dispatch 顺序：TokenUsage 永不判陈旧 → failed 响应抑制 assistant 输出（617-628）→ deleted turn 丢弃 → 陈旧 turn 丢弃（resolve future+discard pending turn）→ history.stage → HistoryCommitError→response.failed
- service.py:671-680 is_turn_output_held（发送循环复查不阻塞事件环）
- service.py:690-769 response_latency_context/_response_input：生成归属解析（工具延续归 call；**新轮已关闭→响应改挂当前轮** 761-769）
- service.py:771-794 record_input_turn（按保留 chat 修剪 item→turn 映射+anchors 128 上限）+ _input_turn_accounting（新轮清旧账 791-793）
- service.py:798-873 _on_transcription_completed：replace user 文本（826）/空转写移除 user（839-841）/不排队 prompt（844）/discard pending turn（861）/terminal 持发（866）
- service.py:875-900 _on_transcription_failed：should_listen 置位+移除 user+零计账+持发
- service.py:902-961 _on_audio_input_completed（audio 直入 LLM：gaea 无此腿）
- service.py:965-1044 token usage 暂存卷积（pending_token_usage）+_on_response_failed（keyed 路由 1022-1029+先 reject 后 failed 1038-1039+音尾哨兵关闭）

### 批1 义务与判罚

| ID | 义务 | Go 现状 | 判罚 |
|----|------|--------|------|
| S1 | pending/closed key 双集+128 墓碑 | closedKeys 128 墓碑 ✓；pending 集无（单响应在途拓扑，无排队多响应面） | 🔵近似记理由 |
| S2 | register：新连接重置 turns | OnSessionEnd→turns.Reset ✓ | ✅对齐 |
| S3 | register：每连接新建 Chat | Pipeline 常驻单例，OnSessionEnd **不清 chat**——跨会话残留 | ⚠️需修（F6） |
| S4 | unregister：收口+用量卷积+latency 清理 | 队列/映射/turns/vad 清 ✓；用量卷积无消费方（gaea 每响应日志口径） | ✅/🔵记理由 |
| S5 | close_response_key：墓碑+tracker close+history.close | 墓碑+turns.Close ✓；**history.close 缺失（O2 回滚/O3 finalize 未实现）** | ❌需修（F2） |
| S6 | barge-in flush prompt 队列且标 dead | cancelActiveResponse **双接收错**：首载荷静默丢弃不标 dead+第二接收可阻塞发送循环 | ❌需修（F3） |
| S7 | failed 响应抑制 assistant 输出 | LLM 失败→兜底句走完 completed（有意 UX 偏差，前端无 error UI 面） | ⚠️记理由（有意偏差） |
| S8 | 陈旧 turn 丢弃+deleted turn 丢弃 | completedFinalRevs 去重+IsLatest 闸 ✓；conversation.item.delete 不支持（gaea 前端无此面） | ✅/🔵记理由 |
| S9 | HOLD 输出停靠+发送循环复查 | pendingText/parkedAudio/10ms 复查 ✓ | ✅对齐 |
| S10 | 新轮已关闭→响应改挂当前轮（761-769） | gate Drop→rs.dead 静默；gaea 全程 barge-in 已先行取消该响应，场景坍缩进 cancel | 🔵记理由 |
| S11 | record_input_turn 修剪+anchors 上限 | inputItemByTurnRev 无修剪（会话期有界+OnSessionEnd 清） | 🔵近似记理由 |
| S12 | 空转写：移除 user+不排队 prompt | text=="" 不写 chat/不排队 ✓；turn 收口走 turns 机 | ✅对齐 |
| S13 | transcription_failed 客户端帧+移除 user | STT 错误→空文本 completed 帧（无 failed 帧；前端对空文本兼容） | 🔵近似记理由 |
| S14 | token usage 暂存卷积 | Reply 同步返回 usage 入 done 帧，无跨线程时序面 | 🔵近似记理由 |
| S15 | _on_response_failed 先 reject 后 failed+begin announce | 失败→兜底句 completed（同 S7 有意偏差） | ⚠️记理由 |
| S16 | 失败响应不写历史 | `rs.ctx.Err()==nil` 即写——**err≠nil 也写半截**（上游 reject=全撤） | ⚠️需修（F4） |
| S17 | O2 取消回滚暂存条目 | **未实现**（批2 台账误记已修）；TestHistoryRollbackOnCancel 靠 ctx 断流不写入侥幸绿 | ❌需修（F2） |
| S18 | O3 完成 finalize（provisional=false） | **未实现**；测试等待循环超时后不校验 provisional=假绿 | ❌需修（F2） |
| S19 | O7 轮制裁剪同样约束 assistant 追加处 | llmStage 追加处仍按**消息数**裁剪——可拆散 user/assistant 配对 | ⚠️需修（F5） |
| S20 | chat 并发写串行化 | sttStage 写 chat **无锁**，与 llmStage goroutine（持 muHist）竞争=数据竞争 | ❌需修（F1） |

### 批1 修复（F1~F6，2026-11-05）

- F1 muHist 全覆盖：sttStage 的 O4 替换/O7 裁剪/append/快照全部持锁（S20）
- F2 finishResponse 终态处置：completed→finalize（provisional=false）；cancelled/failed→删除该 respKey 暂存条目（S17/S18，批2 假绿补真修）
- F3 cancelActiveResponse flush 改单接收+标 dead；llmStage 入口 dead 短路（S6）
- F4 O1 写入条件收紧为 err==nil && ctx 活（S16）
- F5 轮制裁剪抽 trimChatTurnsLocked 共用（sttStage+llmStage）（S19）
- F6 OnSessionEnd 清 chat/closedKeys（每连接新建 Chat 同位）（S3）
- 测试：TestHistoryWriteOnComplete 改硬校验 finalize；新增 TestHistoryRollbackAfterWrite（写完再取消必回滚/完成必 finalize）；TestCancelFlushQueuedPrompt（单请求 flush 不阻塞+dead 短路零生成）；回归 -race 全绿
- **ws_probe 实弹回归（Go 管线，8791 归属=gaea 进程验证后打）**：turn 8/8（speech_end→首音频 216ms，门 ≤2.5s）+bargein 10/10（cancelled/turn_detected+旧代际零泄漏+新响应 completed）+cancel 8/8（client_cancelled+零后置音频）——F2 取消回滚/F3 flush 全链实弹过门
- **坑新增（此批踩实）**：①无 BOM UTF-8 中文注释 .ps1 被 PS5.1 按 ANSI 乱码解析，**吞掉后续 $env: 赋值行**——ps1 必须 ASCII-only（build.bat 同款纪律）；②壳 env 失效时 ensure 走非 Go 分支拉起 sidecar，**ws_probe 会打在 sidecar 上假绿**——打门前必须验 8791 归属进程是 gaea（netstat+Get-Process），sidecar 启动的 8791 属 python

### 批3 证据（chat.py:475-493 trim_if_needed；500-516 replace；已逐行）
- 裁剪按 **user_turn_count > size** 触发（不是消息数）；compactor=None（gaea 配置）→ 同步逐出最旧完整轮
- 2×size 硬上限在 _add_item_locked 内联逐出（admission control，语义同效不同粒度）
- replace_user_message_text 按 item_id 替换（O4 已对齐 ✓）

| ID | 义务 | Go 现状 | 判罚 |
|----|------|--------|------|
| O7 | 裁剪按轮计数（user+配对 assistant 整轮逐出，12 轮） | ✅已修：userTurns>size 逐出最旧轮（user+紧邻 assistant）；全套单测绿 | ✅对齐 |
| O8 | 2×size 硬上限 admission | FIFO 窗口天然有界 | 🔵近似可接受（同效不同粒度，记理由） |

## 义务条目（批4：#10 LLM 历史写回 + #11 conversation/audio 已逐行读完）

### 批4 证据（原文锚点）
- language_model.py:624-631 快照隔离生成：`active_chat = original_chat.copy(deep=True)` + `history_anchor_id = active_chat.history_anchor_id()`
- language_model.py:679/700-703 ResponseHistory.capture(original_chat, items, after_item_id=history_anchor_id, complete=True)——写回钉在快照锚点之后
- language_model.py:612-620 LLM 阶段陈旧请求前置跳过（_turn_is_latest→EndOfResponse）
- language_model.py:693-696 turn_output_allowed/commit_allowed 门
- conversation.py:118-159 item.create/delete 响应期 deferred（工具面）+ :321-389 partial delta 稳定词前缀（双假设锚定）
- conversation.py:417-449 on_transcription_completed 终态化+时长 usage
- audio.py:177-231 _input_terminal_disposition 规则表（Go resolveInputTerminals 逐规则同构：超龄同轮 discard/跨轮 publish+input_closed/started 守卫/unanswered||failed→gate(commit)/无响应收口 turns.close/is_committed→publish/else hold）
- audio.py:232-271 publish 顺序 speech_stopped→committed→item.created→transcription（Go publishTerminal 同构）

### 批4 义务与判罚

| ID | 义务 | Go 现状 | 判罚 |
|----|------|--------|------|
| R1 | LLM 快照隔离生成（deep copy） | promptRequest.history 在 sttStage 时点快照 | ✅对齐 |
| R2 | **anchor 写回：assistant 钉在快照锚点之后**——生成期 soft-end 非打断语音（vadhandler.go:255 InterruptResponse=false）追加的 user 不被越过 | 尾插 append——非打断路径会错序 | ❌需修（F7：writeAssistantAnchor，按 rs.turnID 锚插+越 assistant+锚缺失尾插兜底） |
| R3 | LLM 阶段陈旧请求前置跳过 | dead/closedKeys 短路+首 delta gate Drop；gate Drop 后响应以 completed+空输出收口（status 语义瑕疵） | 🔵近似记理由（LLM 调用浪费面=gaea 本地 35B 零成本；status 瑕疵挂刀6 复核池） |
| R4 | 工具生成期逐 chunk history capture | 无工具面 | 🔵不移植（gaea 无工具） |
| R5 | compactor 历史压缩 | 无 | 🔵不移植（O6 同判） |
| R6 | item.create 注入 chat+响应期 deferred | server.go 仅回声不入 chat（gaea 前端无 item.create 注入面，文本走 commit 路径） | 🔵不移植记理由 |
| R7 | partial transcription delta 稳定词前缀 | gaea STT=paraformer 全句转写，无部分假设流 | 🔵不移植记理由 |
| R8 | conversation item.delete 全链 | 前端无删除面 | 🔵不移植（S8 同判） |
| R9 | 512 块 framing+跨帧余数 | OnAudio pendingSamples | ✅对齐 |
| R10 | 输入重采样（session.update 采样协商） | 协议固定 16k PCM16 | 🔵不移植记理由 |
| R11 | conversation.item.truncate 输出截断 | barge-in=cancel 语义，前端不发 truncate | 🔵不移植记理由 |
| R12 | terminal disposition 规则表+publish 顺序 | resolveInputTerminals/publishTerminal 逐规则同构 | ✅对齐 |

### 批4 修复（F7，2026-11-05）
- writeAssistantAnchor：assistant 插入本轮 user（turnKey==rs.turnID）之后、越过后排 assistant；锚缺失尾插兜底
- 测试：TestHistoryAnchorWriteBack（锚点中插/新 user 不被越过/锚缺失兜底三断言）

## 义务条目（批2：history 两文件已逐行读完）

### 批2 证据（原文锚点）
- pipeline/history.py:74-80 `write(): return self.chat.add_provisional_generation_items(response_key, ...)` — 助手输出以 response_key 为键的**可回滚暂存代际**写入
- pipeline/history.py:82-93 `clean_up(): snapshot_history_cleanup → ... trim_if_needed → 异常 restore_history_cleanup` — 终态清理**事务化**（成功才把 input item 转永久）
- handlers/history.py:38-42 类注释：`close undoes whatever a response wrote unless finalize made it permanent first`
- handlers/history.py:55-58 `reject(): chat.rollback_provisional_generation(key)` — 失败/不完整响应回滚
- handlers/history.py:63-69 `close(): rollback_provisional_generation(key)` unless finalized — **取消=从聊天历史撤销该响应写过的内容**
- handlers/history.py:97-101 `_keep`：generation-done 失败/assistant done incomplete/response-failed → reject
- handlers/history.py:104-106：proposal.chat 非当前会话 chat（迟到输出）→ 丢弃
- handlers/history.py:118-121：keep-newest-wins（(len(items), complete) 元组比较）

### 义务与判罚

| ID | 义务 | Go 现状 | 判罚 |
|----|------|--------|------|
| O1 | 助手回复写入聊天历史（暂存代际制，键=response key） | **llmStage 从不写回 assistant 历史**——每轮 LLM 看到的都是无回复历史 | ❌缺失（多轮上下文断裂） |
| O2 | 取消/失败 → `rollback_provisional_generation(key)`：被打断的回复从历史撤销 | 无任何回滚机制 | ❌缺失（打断的半截回复会以任何形式残留——现因 O1 反而"不残留"，修复 O1 时必须同批实现 O2） |
| O3 | 完成 → `finalize_provisional_generation(key)` 转永久 | 无 | ❌缺失（随 O1/O2 同批） |
| O4 | 同轮重开的 user 文本：`replace_user_message_text`（不新增条目） | chatAppend 无条件 append——重开会产出重复 user 条目 | ⚠️偏差（需修：按 turn 键替换） |
| O5 | 提交时机：首 delta gate 提交后才 history.accept 开始写 | （随 O1 设计）首 delta 提交后暂存写入 | 随 O1 |
| O6 | 终态清理事务化（snapshot/restore）+ trim_if_needed | chat FIFO 窗口近似（无 2×size 硬上限语义） | 🔵近似可接受（gaea 无工具链/图片/音频历史压缩，trim_if_needed 主干=FIFO；记理由） |

### 批2 修复（已落地，2026-11-04）

- ChatMessage 增 respKey/provisional/turnKey（暂存代际数据面）
- O1：llmStage 完整回复以 `assistant{respKey, provisional:true}` 暂存写入（ctx 断流不写）
- O2：finishResponse cancelled/failed → 删除该 key 暂存条目（rollback_provisional_generation 同位）
- O3：completed → provisional=false（finalize 同位）
- O4：sttStage user 写入按 turnID 替换（replace_user_message_text 同位）
- 测试：TestHistoryWriteOnComplete / TestHistoryRollbackOnCancel / TestHistoryUserReplacedOnReopen
- 回归：ws_probe turn 8/8 + bargein 10/10 零失败
- O6 判罚维持 🔵近似（trim 主干=FIFO 窗口；2×size 硬上限语义挂刀6 复核池）

### 批2 修复设计（faithful minimum，gaea 无工具/图片面）
- ChatMessage 加 `respKey string` + `provisional bool`
- llmStage 首 delta（发送循环已 gate commit 处）：写入/更新 assistant 暂存条目（随流累积文本，滚动更新同一条目）
- finishResponse completed → finalize（provisional=false）；cancelled/failed → rollback（删除该 key 暂存条目）
- sttStage：user 写入按 turnKey 替换（同 key 已有则改文本）
