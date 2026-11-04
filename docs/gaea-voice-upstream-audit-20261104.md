# 语音管线复刻审计台账（REA 纪律：义务须锚上游 file:line 原文）

- 状态：**进行中**
- 方法：逐文件读上游原文（`.tmp/research-hf-sts/src/speech_to_speech/`，6feb527）→ 义务条目（ID/证据引用/引文/Go 现状/判罚）→ 差距修复（一处一刀：build+单测+ws_probe）
- 判罚：✅对齐 / ⚠️偏差（需修）/ ❌缺失（需补）/ 🔵上游死代码或 gaea 未用（不移植，记理由）
- 上游文件清单与进度：
  1. `api/openai_realtime/websocket_router.py` — ✅已读（发送循环/停靠/代际闸已移植）
  2. `api/openai_realtime/handlers/response.py` — ✅已读（finish_response 守卫/key-vs-current）
  3. `pipeline/speculative_turns.py` — ✅已读（9 单测）
  4. `pipeline/turn_latency.py` — ✅已读（日志行逐字）
  5. `STT/base_stt_handler.py` — ✅已读（输入门/2048 集）
  6. `VAD/vad_handler.py`+`vad_iterator.py`+`smart_turn.py` — ✅已读
  7. **`api/openai_realtime/service.py` — 🔴未逐行**（只经子代理转述；_end_response/history staging/should_listen 细节）
  8. **`pipeline/history.py` + `api/openai_realtime/handlers/history.py` — 🔴完全未读**（spec 歧义#10：hold/accept/reject/close/finalize 语义——取消回滚聊天历史的机制）
  9. **`pipeline/messages.py` Chat 类 — 🔴未读**（chat_size 裁剪/trim_if_needed/硬上限 2×size 逐出/provisional generations）
  10. **`LLM/chat_completions_language_model.py` 历史写回 — 🟡部分**（ResponseHistory.capture/history_anchor_id 语义）
  11. `api/openai_realtime/handlers/conversation.py`+`audio.py` — 🟡经转述（input_item 复用/resolve 规则已移植；_ack_item/previous_item_id 链已移植）

## 审计批次

- 批3 已完成（O7 修复+O8 记理由）
- 批1（当前）：#7 service.py 逐行 → 义务条目
- 批2：#8 history 两文件（取消回滚聊天=用户可感知行为）
- 批3：#9 messages.py Chat 裁剪语义
- 批4：#10/#11 补读收口
- 每批产出：义务表+判罚+修复提交

### 批3 证据（chat.py:475-493 trim_if_needed；500-516 replace；已逐行）
- 裁剪按 **user_turn_count > size** 触发（不是消息数）；compactor=None（gaea 配置）→ 同步逐出最旧完整轮
- 2×size 硬上限在 _add_item_locked 内联逐出（admission control，语义同效不同粒度）
- replace_user_message_text 按 item_id 替换（O4 已对齐 ✓）

| ID | 义务 | Go 现状 | 判罚 |
|----|------|--------|------|
| O7 | 裁剪按轮计数（user+配对 assistant 整轮逐出，12 轮） | ✅已修：userTurns>size 逐出最旧轮（user+紧邻 assistant）；全套单测绿 | ✅对齐 |
| O8 | 2×size 硬上限 admission | FIFO 窗口天然有界 | 🔵近似可接受（同效不同粒度，记理由） |

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
