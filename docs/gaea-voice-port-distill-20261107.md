# 语音 Go 移植蒸馏审计（上游 speech-to-speech v1.1.0 ↔ Go realtime 包）

日期：2026-11-07（凌晨批次，接 2026-11-06 夜实弹翻车主修）
上游基线：`clones/speech-to-speech` HEAD=8ebde04（release/v1.1.0 merge），`response.py:991` 锚点验证吻合。
方法：五路并行逐义务核对（vad_iterator / vad_handler / speculative_turns / response+history / audio+conversation+session），义务清单先行、只判不修；每项给上游锚 file:line + Go 判定（同位/漏抄/有意裁剪/改形）+ 风险。

## 已修（2026-11-06 夜，vad_prefix_test/transcript_join_test 钉死）

| 病灶 | 上游锚 | 根因 |
|---|---|---|
| 跨轮前缀不清→转写滚雪球（实弹字符数 2→54） | vad_handler.py:270-279 `_ensure_turn_for_speech_start` | 刀3 移植漏抄新轮清前缀守卫 |
| transcript done 空格撕裂（"风 把 白 日"） | response.py:605-610 句级 join 语义 | 句级 `" " join` 错挂到 delta 粒度 parts |

## 漏抄/缺口台账（按风险排序）

### 高（实弹可见）

| # | 病灶 | 锚 | 症状 |
|---|---|---|---|
| H1 | 后续 delta 不重过 gate：Go 仅首增量查 gate（pipeline.go:749-754），后续只拦 Hold 不查 Drop；上游 `_accept_turn_output`（response.py:1063-1079）每事件都查 | response.py:1063-1079 | 重开/补发路径开新轮后，旧响应剩余 delta 继续下行且按 completed 装配——字幕与语音脱节、回答与插话混流 |
| H2 | 被顶替/丢弃响应的 O3 回滚永不执行：finishResponse key 守卫早退（pipeline.go:968-973）跳过历史处置，provisional assistant 永久滞留 chat，污染后续每轮 LLM 上下文 | history.py:84-91 | 答非所问的隐性贡献源；chat 里堆尸体回复 |

### 中（真实症状窗口）

| # | 病灶 | 锚 | 症状 |
|---|---|---|---|
| M1 | OnSessionEnd 不复位 echoGuard/baseThreshold（vadhandler.go:385-393） | Go 新增态自欠清理（上游无此机制） | 播放中断线重连，首轮播放回声门静默失效→假 barge-in（2026-11-01/05 根修症状复发窗口）；今晚 t128 疑似回声复听 |
| M2 | audioBatch 全局缓冲 cancelled 不清零（dispatchAudio pipeline.go:769-785，仅 completed flush:998） | response.py:939-955 resampler per-response | 下一回答开头混入被打断的 ≤200ms 残音，挂在新 response_id 下 |
| M3 | 同轮重开"快照尾=user"假设错位（sttStage pipeline.go:516） | history.py:118-123 chat 身份守卫 | reopen 时 chat=[user(改), assistant_old(provisional)]→尾删错位，旧轮提问在 LLM 上下文重复出现 |
| M4 | processing_deadline 改形：delay≤0 合并"无死线"+ ProcessingDeadline bool 吞并"已到期"（turns.go:279-284/303-310，sttStage:440-444） | speculative_turns.py:149-159/169-174 | smart turn 慢时说完话响应莫名多压几秒 |
| M5 | STT 失败吞成空 completed 转写，transcription.failed 分支死代码（pipeline.go:451-456/824-825） | conversation.py:451-484 | 识别失败显示为空泡而非失败提示 |
| M6 | `response.output_audio.done` 构造器在（events.go:244）但零调用 | response.py:961-967 | 依赖该帧判定音频结束的客户端只能等 response.done |
| M7 | 零输出/早取消路径发 response.done 无前置 response.created | response.py:1121-1130 | 客户端收到未知 response_id 终态帧 |
| M8 | LLM 失败按 completed+兜底语收口、TTS 句失败仅日志：无 failed/incomplete 终态，item status 恒 completed | response.py:971-997 | 前端无法区分真完成与降级 |
| M9 | wait_for_gate 漏抄 commit 参数（turns.go:372-400）；ttsStage 等待结束后不补提交 | speculative_turns.py:221-247 | 边缘流下 TTS 出声而轮次未提交，可被重开翻转串轮 |
| M10 | commit 点窄窗 HOLD 置 dead 丢弃整条响应（pipeline.go:750-753），上游 TurnOutputHeld 停靠重试 | speculative_turns.py:34-35 | 重开恰在输出提交前一瞬，偶发丢一整段回答 |
| M11 | 客户端控制面空转：response.create 空实现（pipeline.go:341）、conversation.item.create 纯回声（server.go:321-327）、item.delete 静默吞、input_audio_buffer.commit 不校验不喂管线且回造随机 item_id（server.go:317-318） | conversation.py:118-201、audio.py:325-349 | **已核（2026-11-07）：gaea 前端发送面仅 session.update+麦克风帧+response.cancel（VoiceClientV2.ts:9/168），控制面路径不使用 → 记有意裁剪不修**；将来接手动注入/文本输入时先补 |
| M12 | 关闭清理序：槽位先释放+墓碑先清零（server.go:242-251、pipeline.go:313-315 closedKeys 重建） | router:290-385、service.py:407-443 | 快速重连竞态：新会话被旧 OnSessionEnd 抹配置/收旧残帧 |
| M13 | session.update 字段生效度：interrupt_response/create_response/语言/transcription 块不解析或硬编码（events.go:42-74、vadhandler.go:231/299、PipelineConfig 构造期定死） | session.py:36-104、runtime_config.py:59-88 | 客户端关 barge-in/关自动响应/设语言均被静默无视 |
| M14 | silence_duration_ms 运行时更新漏抄（events.go:26-30 协议层即未解析） | vad_handler.py:237-239 | 调静音断句窗无效，恒 64ms |
| M15 | 输入采样率协商/重采样整块漏抄（恒 16k 解释，server.go:20） | audio.py:283-306 | 客户端声明非 16k 即变速噪声（gaea 前端恒 16k，暂无症状） |

### 低（可观测性/配置门禁/边角，暂不动）

vad_iterator：采样率校验缺失；onset 阈值 float32 化（0.5/0.75 无差）；打分失败降级 prob=0（sidecar 抛错）。
vad_handler：_resolve_min_speech_continuation_ms 缺失（配 0 时 discard 门失效）；Smart Turn 阈值硬编码 0.5；构造校验缺失；lastHold/SetWaitObserver 未接线（延迟账 hold 恒 0）；port-spec 文档默认值表滞后（0.6/30ms/2000/600 vs 实弹 0.5/500/1200/400，需回写）。
turns：cancel_reopen_candidate revision 过滤参数（上游也无调用者）；墙钟替代单调钟（NTP/休眠跳变窗口）。
response：cumulative usage/responses_cancelled 计数；voice/conversation_id/output_modalities 字段；cancelScope 代际闸死代码（gen 不递增，注释声称的防线不存在——**激活 scope 前必须先修，否则 done 哨兵 gen=0 被误丢**）；会话切换 proposal.chat 身份守卫。
audio/conversation：partial 转写 delta 缺失（前端契约列有该事件）；session.updated 缺 id；帧缺 output_index/content_index；重开旧 revision 映射不清理（表缓增）。

## 同位确认（抽主干）

相位机全迁移/can_reopen 五条件/2s HOLD/7000ms/grace 仅延长/commit-close 语义（turns.go）；延迟确认 384/192/候选先行/幻影/discard 三条件/Smart Turn 宽限表/前缀三读写点/OnSessionEnd 清理序（vadhandler.go）；滞回带 0.15/500ms 复位/lastUtterActive 三段生命周期/pre-pad 修剪/结束复位包 9 项（vad.go）；terminal 持发规则表/重开 item 复用 §5.1/首增量提交序/终态装配序/usage 记账（pipeline.go）；持发重试泵/单槽拒绝（server.go）。

## 建议刀序（一处一刀，待拍板）

1. **批1（同族状态泄漏+今晚症状邻接）✅ 已落（2026-11-07 00:40 装机）**：H1+H2（dispatchText 跨 key+DROP 双守卫 / finishResponse 被顶替路径补 O3 回滚）→ M1（echoGuard/baseThreshold 复位）→ M2（audioBatch 属主键清理）→ M3（llmStage 切史按本轮 user 为界）。测试 port_distill_batch1_test.go 6 例。
2. **批2（体验慢/错提示）✅ 已落（2026-11-07 00:50 装机）**：M4（sttGateWait：死线已到期=holdUntil 不再持门立即放行，不回退 item delay 再等一整段）→ M5（STT 失败发 transcription.failed 终态，failed 分支激活，空泡病封死）→ M6（audio.done 收口帧，audioEmitted>0 恰一次）→ M7（终态前 created 补发，孤 done 封死）。测试 port_distill_batch2_test.go 4 例；前端 voice 53/53 绿（failed 帧走 UnknownServerEvent 放行）。**M8（failed/incomplete 终态）挂起**——动前须核 voiceSession 对 done status failed/incomplete 的状态机反应，避免兜底语路径 UX 回归。
3. **批3（协议面）✅ 已落（2026-11-07 01:04 装机）**：M12（槽位释放移至 OnSessionEnd 后+closedKeys 墓碑替代清零）→ M13（interrupt_response/create_response 会话级开关生效，*bool sticky+会话复位；前端哑旋钮激活）→ M14（silence_duration_ms 运行时可调+会话结束回落）。M15 记有意裁剪（gaea 前端恒 16k）。测试 port_distill_batch3_test.go 5 例。
4. **M8 挂拍板**：前端核对完毕——failed→error 态+冲刷未播块、incomplete→idle+冲刷（voiceSession.ts:246-258 + VoiceClientV2.ts:130-133）。gaea 的 F1 兜底语/TTS 续播设计会被 failed/incomplete→flush 截断（LLM 失败说兜底语被打断、TTS 单句失败后余句被砍）。两选一：①接受截断换诚实 failed/incomplete 终态；②维持现状（记有意偏离，协议台账注记）。默认②不动。
