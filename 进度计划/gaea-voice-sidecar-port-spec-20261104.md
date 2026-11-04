# sidecar 轮次机移植规格（Go port spec）

来源：.tmp/research-hf-sts（上游 6feb527）逐文件精读提取，2026-11-04。主计划见 gaea-voice-go-native-pipeline-20261104.md。本文是刀3/4/5 的实现依据，语义必须逐字对齐。

## 1. vad_iterator（silero 原始状态机）

常量：`min_silence_samples=int(16000*64/1000)=1024`、`speech_pad_samples=int(16000*500/1000)=8000`、`_model_reset_idle_samples=8000`（500ms 空闲仅重置推理态）。
状态：`triggered/temp_end/current_sample/last_speech_end_sample/buffer/prefix_buffer/_pre_speech_buffer(≤8000样本)/active_speech_samples/last_utterance_active_speech_samples/_model_idle_samples`。
window_size=512（v5@16k），current_sample 累加。

`__call__(chunk)` 分支：
- ONSET：`prob >= threshold` 且 !triggered → triggered=true；prefix_buffer=冻结 pre_speech_buffer 并清空；buffer=[chunk]；active=512；返回 None
- IDLE（!triggered 且 prob<th）：记入 pre_speech_buffer（按块裁剪至 8000 样本）；`prob < threshold-0.15` 时累计 _model_idle_samples，≥8000 → model.reset_states()（只重置推理态，pad/current_sample 保留）
- SEGMENT（triggered）：chunk 恒入 buffer；`prob >= threshold-0.15`（滞回带）→ active+=512 且 temp_end=0（间隙桥接）；`prob < threshold-0.15` → temp_end==0 时 temp_end=current_sample；`current_sample-temp_end < 1024` 返回 None；否则**段结束**：`last_speech_end_sample=temp_end-512`；triggered=false；utterance=prefix+buffer；冻结 active→last_utterance_active；清空 buffers；model.reset_states()；返回 utterance

## 2. vad_handler（gaea 配置）

生效值：thresh=0.6（前端 session.update 可改 0.75）、min_silence=64ms、min_speech=384ms、min_speech_continuation=192ms、speech_pad=500ms、smart_turn_max_wait=1200ms、incomplete_delay=400ms、speculative_reopen=800ms、unanswered_reopen=max(800,7000,1200)=7000ms（音频时钟！）、merge_window=0（禁用）、_SHORT_SEGMENT_MIN_FRAGMENT_MS=100。

每 chunk 步序：runtime turn_detection 应用（threshold 变更 log "VAD threshold updated to X"）→ int16→float32(/32768) → _total_samples 累加 → iterator() → 延迟 SpeechStarted 确认 → 段结束处理。

**延迟 SpeechStarted 确认**（is_triggered_now 且未发过）：segment_ms=buffer 总长；active_ms；start_ms=max(0,_audio_ms-speech_buffer_ms)；`speech_candidate_started(start_ms)`（HOLD≤2s 的重开候选）；active_min = has_candidate||can_reopen ? 192 : 384；`active_ms >= active_min` 才发 SpeechStartedEvent（audio_start_ms=start_ms, turn/rev 来自 tracker.speech_started(start_ms)→(turn_id,rev,reopened)），log `"Speech started (confirmed, active=%.0fms, min=%.0fms, segment=%.0fms, turn=%s rev=%s)"`。会话首个 turn（非重开）时清前缀缓存。

**段结束（utterance 返回）**：
- 空utterance=幻影触发：log "VAD: phantom trigger (empty buffer), closing speech pair"；已发 start 则补 speech_stopped；segment_discarded()/speech_candidate_cancelled()
- array=cat(utterance)；raw_active_ms=last_utterance_active/sr*1000；duration_ms；active<min（未发start时 384/192，已发时 0）→ **discard 路径**：log "VAD: discarding segment=%.0fms active=%.0fms (active_min=%sms, segment_max=%sms)"；补 speech_stopped；segment_discarded/candidate_cancelled
- **soft-end 接受**：未发 start 则补发 SpeechStartedEvent（**interrupt_response=False**！）；log "Speech soft-ended (segment=%.0fms, active=%.0fms, turn=%s rev=%s)"；`analysis_audio=前次 raw 前缀+本段`（喂 Smart Turn）；`_smart_turn_timing_ms` → (reopen_grace_ms, processing_delay_ms)：complete→(800,0) log "Smart Turn: complete (p=%.3f, %.1fms); using %dms speculative reopen grace"；incomplete→(1200,400) log "...incomplete (p=%.3f, %.1fms); using %dms speculative reopen grace and delaying processing by %dms"（宽限数值打 1200）；failed/disabled→(800,0)；`speech_end_at_s = received_at_s - max(0,_total_samples-last_speech_end_sample)/sr`；vad_decision_s=decision_at-speech_end_at；`output_array=前次前缀+本段`；发 SpeechStoppedEvent(duration_s=combined秒, audio_end_ms=_audio_ms, turn/rev)；存前缀；`segment_finalized(end_ms, output_hold_ms=grace, processing_delay_ms=delay)`；yield VADAudio(mode=final, processing_delay_s, speech_end_at_s)

## 3. SpeculativeTurnTracker（轮次机）

类型：TurnPhase=LISTENING/SOFT_ENDED/ANSWERING/CLOSED；gate=ACCEPT/DROP/HOLD(deadline)；_PENDING_REOPEN_HOLD_S=2.0；turn id=`turn_{sequence}`（seq 从 1 起）。

核心方法：
- `start_turn()`：seq+=1，current=(seq,turn_N,rev=0)，phase=LISTENING，清全部悬挂态
- `speech_started(audio_ms)`：current 且 LISTENING → 原样继续（False）；否则可重开则起候选→confirm（rev+1，True）；否则 start_turn（新 turn，False）
- `can_reopen(audio_ms)`：current 存在 且 phase==SOFT_ENDED 且 未提交未关闭 且 `max(0,audio_ms-_last_final_audio_ms) <= 7000`（**音频时钟**，audio_ms=管线累计输入音频 ms）
- `begin_reopen_candidate`：幂等（同 turn+base_rev 返回同 candidate）；候选=(turn,base,cand=base+1,deadline=now+2.0)
- `confirm_reopen_candidate`：校验 pending 匹配+current 未阻塞 → current=（同seq,turn,cand_rev），phase=LISTENING，清 grace/pending/deadline
- `segment_finalized(audio_ms, output_hold_ms, processing_delay_ms)`：current 有效时 phase=SOFT_ENDED；_last_final_audio_ms=audio_ms；_processing_deadline=now+delay/1000；start_reopen_grace(turn,rev,hold/1000)（deadline 仅延长不缩短）
- `segment_discarded()`：cancel 候选；LISTENING→SOFT_ENDED
- `gate(turn,rev,hold_for_grace,hold_until,commit)`：非 current 非 committed→DROP；未提交时 deadlines=[hold_until(若>now), pending_reopen(deadline>now 且 base_rev 匹配), grace(若 hold_for_grace 且匹配 current)]→非空返回 HOLD(min)；commit=true 时提交（committed 集合+phase=ANSWERING+清 grace）；返回 ACCEPT
- `wait_for_gate(...)`：阻塞至非 HOLD；hold_until==None 且非 pending-hold 的等待计入 wait_observer（smart_turn_wait 区间合并）
- `processing_deadline(turn,rev)`：仅 current 返回 _processing_deadline
- `is_latest(turn,rev)`：None→True；==current 或 ∈committed
- `close(turn,rev)`：从 committed 移除；若==current → _closed_current 记录+phase=CLOSED+清 processing_deadline（仍 current，回答有效，不可重开，下轮 start 后自然 stale）
- `is_committed/is_current_turn`；`reset()` 清全部

提交 vs 关闭：提交=首个被接受的 assistant 输出（gate commit=true）→ANSWERING，不可再重开（_blocks_reopen）；关闭=response 终态→从 committed 移除，若 current 则 CLOSED。

## 4. STT 输入门（base_stt_handler）

- completed-final 集合：OrderedDict（turn,rev）→ 上限 2048，发射 Transcription 时写入；命中→丢输入（input-after-final）
- final 稳定等待：`hold_until = monotonic + max(final_revision_settle_s=0, max(0, processing_delay_s - age))`；若 tracker 有 processing_deadline（仅 current）则用 `deadline - now` 替代；`wait_for_gate(hold_for_grace=False, hold_until=...)`
- 等待 ≥50ms：log `"%s: STT input gate waited %.3fs for turn=%s rev=%s mode=%s latest=%s age=%.3fs queue=%s"`（前缀=类名）；且 vad_settle_s=max(0,gate_wait)（仅 final+latest）
- 非最新→丢弃（log "dropping stale STT %s for turn=%s rev=%s age=%.3fs" 首次 INFO 后 DEBUG 去重）+ discard_pending_turn + 扫输入队列丢 stale
- 输出侧：发射前 is_latest 校验（hold_for_pending_reopen=True）

## 5. 协议发射（service 层可观察行为）

ID：`{prefix}_{uuidhex}`；resp/item/event/msg/conv/session；turn_{n}；response_key=uuid4().hex。

**输入项生命周期**：input_item_by_turn_revision[(turn,rev)]→item_<id>；重开复用同一 item（不重发 speech_started）；SpeechStarted 时若 in_response 且 interrupt（默认 True）→ **barge-in**：finish_response(cancelled, reason="turn_detected"——精确拼写) 先发（output_audio.done 若有音频→各 done 帧（item status=incomplete）→response.done{cancelled,turn_detected}），然后 generation bump+三队列 flush+丢弃在途音频；turns+=1（非重开）。

**speech_stopped 被持**（PendingInputTerminal per item）：不立即发。resolve_input_terminals 处置规则：非 latest→（同 turn 超龄 rev：discard；旧 turn：publish+释放）；started_turn_id 不同→publish；未答（input_closed 且无 response）或 transcription failed→gate(commit=true)：HOLD→继续持；ACCEPT→committed 后（若无 response）tracker.close+publish；is_committed→publish；否则 hold。
**publish 顺序**：speech_stopped → input_audio_buffer.committed{item_id,previous_item_id} → conversation.item.created（user message {id,type:"message",role:"user",status:"completed",content:[{type:"input_audio"}]}）→ transcription.completed{item_id,transcript,usage:{seconds,type:"duration"}}。

**转写完成→响应**：chat 加 user item（.id=item_id；同 turn 已有则 replace_user_message_text）；create_response（默认 True）→ GenerateResponseRequest{response_key, turn/rev, speech_stopped_at_s} 入 LLM 队列；terminal 持到 commit。

**响应链（隐式，语音路径）**：response.created（首音频前惰性发射，in_progress）→ output_item.added（assistant message in_progress, content:[]）→ content_part.added → output_audio_transcript.delta（文本段间加前导空格）→ [output_audio.delta（≤6400B 批）→ output_audio.done] → output_audio_transcript.done（transcript=" ".join(parts)）→ content_part.done → output_item.done（status=completed|incomplete）→ response.done{status, status_details{reason?}, usage{input_tokens,output_tokens,total_tokens, input_token_details{audio_tokens:0,cached_tokens:0,image_tokens:0,text_tokens:input}, output_token_details{...}}, metadata{...}}。
- reason 值：turn_detected / client_cancelled / max_output_tokens / content_filter；failed 时 status_details.error.type="response_failed"
- 首音频出站时 response_playing.set()；response 完成后 "Pipeline %d: response complete, listening re-enabled"

**客户端 response.cancel**：had_response → cancel_scope.cancel+flush 三队列；finish_response(cancelled, "client_cancelled"——精确拼写)；无在途响应时静默关 pending（不伪造 done）。closed_response_keys 墓碑（≤128，FIFO 驱逐）：墓碑后同 key 的 AudioOutput/assistant 事件全丢。

**usage 路由**：closed key→全局；pending key→缓冲（响应可见时附上）；当前→response_usage（log "Token usage (response): input=%d, output=%d"）。

**should_listen**：set-only（永不清）；VAD 不静音，靠 barge-in 取消。

## 6. 延迟账（turn_latency）

tracker 字段：stt_s（推理耗时）、llm_ttft_s（首个非空白 delta，first-write-wins，不进 payload）、llm_s（整代）、tts_ttfa_s（首模型音频块，first-write-wins）、e2e_s（首 yielding 音频 − speech_stopped_at_s，SpeechLatencyAnchor 单发）、vad_settle_s（STT 输入门整等待，≥0.05 才记）、vad_decision_s、smart_turn_analysis_s/status/grace/processing_delay、smart_turn_wait_s（区间合并）、status。
pending slot（(turn,rev)）→ 首 response tracker 创建时 absorb。pop 后复用禁止（取消不可复活）。

**日志行（eot_analyze.py/false_interrupt_bound.py 解析此格式，逐字保留）**：
`Turn {turn_id} rev={rev} latency: stt={.2f}s llm={.2f}s tts_ttfa={.2f}s e2e={.2f}s [vad_settle={.2f}s ]vad_decision={.2f}s hold={.2f}s smart_turn_status={...} status={...}[ tts=cut][ source={speech|client|tool}] response_key={hex}`
（`_fmt=None→"n/a"`；rev None→0；mlx 段 darwin-only 跳过；status==cancelled 且有 tts_ttfa → " tts=cut"）

**response.done metadata**：key=`speech_to_speech.turn_latency`，JSON sort_keys 紧凑，仅当 metadata<16 条且 ≤512 字符：
`{version:2, source, turn_id, turn_revision, response_key, stt_s, llm_s, tts_ttfa_s, e2e_s, vad_decision_s(9dp), smart_status, hold_s(9dp), status}`

## 7. Go 移植清单（对照检查）

1. silero 直跑状态机（§1）+ L STM 态经 ORT（state [2,1,128]/stateN，input [N,512]，sr int64）
2. 延迟 SpeechStarted（384/192+候选先于阈值判定）✦log 行
3. 段结束三分支（幻影/discard/soft-end）+ Smart Turn 宽限表 + 前缀拼接 ✦log 行
4. Tracker 全方法（§3）含音频时钟 7000ms 上限、HOLD≤2s、grace 仅延长
5. STT 输入门（§4）+ completed-final 集（2048）
6. 输入项生命周期+持发规则+publish 顺序（§5）
7. 响应链全序+batch 6400B+reason 精确拼写（§5）
8. cancel 双路（turn_detected/client_cancelled）+墓碑 128
9. usage 路由+done metadata（§6）
10. 延迟日志行逐字（eot_analyze 兼容）
