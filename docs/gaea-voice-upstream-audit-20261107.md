# 语音实时管线全量对源审计 · 2026-11-07

> 触发：用户三轮真机复测连续翻车（回声自打断→续说断续→大段散文中段被掐），
> 指令「把上游代码自己再看了」。两路代理全文对读：服务端管线（s2s_pipeline+
> pipeline/*+handlers/response.py ↔ pipeline.go/turns.go/server.go）、客户端
> 音频链（audio_client.py ↔ pcmQueue/VoiceClientV2/useVoiceChatV2/worklets）。
> 全文锚点版报告在会话档；本文收录全部分歧与修复清单。**教训入册：移植审计
> 必须含发送循环/队列/状态机全行为面，禁「症状→补一层」。**

## A. 服务端管线分歧（上游=排队不取代、零内部取消）

| # | 严重度 | 上游 | 我们 | 后果 |
|---|---|---|---|---|
| A1 | 高 | 输出被轮次 gate HOLD 时放 `pending_output_item` 原地重试，不取新块，零丢失零乱序（websocket_router.py:938-957） | 单槽 parkedAudio：槽占时后续 held 块**直接丢弃**+乱序派发（pipeline.go:747-781）；每轮回复自己的 reopen 宽限 800/1200ms 内首块必然 HOLD | 句间丢块/缺字/断续 |
| A2 | 高 | done 哨兵与音频同队列严格有序，响应总有 completed 终态 | done 绕过 held（pipeline.go:803-805）→ 先于槽中块到达 → `audioEmitted==0` → **内部取消 cancelled/empty_output+历史回滚**（:769-771），无需任何用户输入 | 整段回复无声被撤 |
| A3 | 高 | 新响应**排队**不取代：旧响应保持 current 完整交付，response created 延迟到首个输出（response.py:80-99,1121-1130） | 转写完成即偷槽（STT goroutine 直写 inResponse/respState，pipeline.go:560,1034-1037）：旧响应 delta 被 H1 守卫静默拒附、旧 done 走 superseded 分支**零帧零日志**（:1075-1108） | 字幕定格、done 缺失卡状态机、混流；**日志零线索的最精确匹配** |
| A4 | 中 | cancel_scope.cancel() 递增代际+丢弃窗口=取消第一道闸（cancel_scope.py:24-33） | 代际机制**死代码**（gen 从不递增，discardable 恒 false）；ttsStage 不查 dead，Speak 失败日志被压（:687-689） | 取消 aftermath 全程静默 |
| A5 | 中 | ConnState 单线程读写 | STT goroutine 无锁写槽位 + `inputItemByTurnRev` **并发 map 写**（pipeline.go:516/822→1263） | 数据竞争，最坏 panic |
| A6 | 低-中 | session.update 深合并仅显式字段 | VAD 块整体替换，未带字段被静默复位（server.go:350-354） | 一次部分 update 重开 barge-in |

## B. 客户端音频链分歧（上游=全局 FIFO 无上限+声卡按需拉取）

| # | 严重度 | 上游 | 我们 | 后果 |
|---|---|---|---|---|
| B1 | 高 | 播放缓冲无上限，永不丢（audio_client.py:245,258-266）；输出按声卡回调拉取=天然背压 | sink ring 上限 4s，满**丢最旧**，泵 100ms 无背压全量灌（sinkProcessor.js:13-15,57-67; useVoiceChatV2.ts:236-245） | 长散文 TTS 产出快于播放 → 积压超 4s → **中段整段静默跳过**（tail 有 overflow 计数可验证） |
| B2 | 高 | response.created 对播放零操作（:511-515） | 新 activeResponseId 即 flushSink 清 ring（useVoiceChatV2.ts:200-204）——**刚修的 pcmQueue FIFO 被 ring 层抵消** | 多响应流上一响应尾部被掐 |
| B3 | 高 | speech_started 无条件 clear（:465-467）——因上游 interrupt 恒开 | 关打断后客户端仍冲 queue+ring（VoiceClientV2.ts:149-153） | 勿打断模式下回声/说话检测=跳段续播=断续 |
| B4 | 中-高 | done(cancelled) → clear 全清（:536-539） | 只 flush 队列，ring 不清 | 取消后尾音继续外放 ≤4s=回声燃料 |
| B5 | 中-高 | 「在播」由真实输出消费决定；is_active 含 150ms 尾 | 回执=队列拉空即发（≤100ms）；EchoGuardOff 挂生成终态，ring 还剩数秒 | 尾声处于监听态 → 回声触发 speech_started 掐尾段 |
| B6 | 中 | 采集无增益 | 自适应增益 ×4+浏览器 AGC 放大静默期回声 | 回声更易过 VAD |
| B7 | 低 | 起播缓冲/欠载迟滞旋钮（默认 0） | 无起播门，欠载补零 | 细碎爆音 |
| B8 | 低 | incomplete/failed 走 finish 继续播 | 非 completed 一律 flush | failed 时已生成部分全丢 |

## C. 症状映射

- 「大段散文中段被掐」：B1（ring 丢最旧，逐字吻合）+ A1（句间丢块）
- 「断续/跳段」：A3（偷槽 superseded）+ B2（created 清 ring）+ B3（勿打断仍冲）
- 「日志零线索的取消」：A2（empty_output 内部取消）+ A3（superseded 零帧）+ A4（日志被压）
- 回声自打断残余：B4/B5/B6

## D. 修复清单（全部=对齐上游，不自创）

**P0（高，一次修完）**：A1+A2（发送循环 HOLD=停止消费原地等待，done 严格后置；删单槽丢弃）→ A3（重建 response_pending 两态：转写完成只排队，新响应由发送循环在旧终态后创建）→ A5（并发写加锁）→ B1（泵改水位控制：ring 低于低水位才泵，同构声卡拉取）→ B2（created 不清 ring）→ B3（speech_started 冲刷加 interruptResponseOn 条件）→ B4（done(cancelled) 清 ring）
**P1**：A4（cancel_scope 激活或记偏离删除+ttsStage 查 dead+放开被压日志）→ B5（门关闭对齐真实播完：服务端 tail 与客户端 ring 余量协调，方案随 P0 落地评估）→ A6（session.update 深合并）→ B6（AGC 播放期冻结）
**P2**：B7 起播门旋钮、B8 incomplete/failed 继续播、A4 观测面补全

## E. 补充审计：VAD 段机 + 协议状态机（第二、三路，全文对读）

### E1. VAD 段机新增分歧

| # | 严重度 | 上游 | 我们 | 后果 |
|---|---|---|---|---|
| V1 | 中高 | block 模式门=`playback.is_active()`=**真实播放结束**才关 | EchoGuardOff 锚在**合成哨兵**（TTS done）+尾窗 600ms，而自记残响上界 0.5~2.4s | 播放末段仍出声时门已关，残回声 0.5+384ms 进段机；有下一响应在生成时→假 Started→**取消回复** |
| V2 | 中 | 默认模式播放期段机活着、回声 phantom/discard 会清陈旧候选 | 半双工丢帧→无 phantom→**陈旧 reopen 候选跨播放存活**（HasSpeechCandidate 不查 deadline） | 播放后首个猝发以 192ms（非 384ms）过确认→假 Started |
| V3 | 中 | block 模式同样冻结（帧不发送） | 半截段跨播放冻结拼接：TTFA 窗口内 onset 未确认→段机带半截 buffer 冻结播放+尾窗→播放后拼接定稿 | 播放前后的话缝成一条 STT 输入 |
| V4 | 低 | on_session_end 全量复位 | OnSessionEnd 不清 tailUntilNanos | 播放中重连，新会话首 600ms 被吞 |
| V5 | 低 | stitch（short-segment merge）机制存在，默认 merge=0 惰性 | 无对应物无开关 | 与上游默认配置等价；merge>0 部署不等价 |
| — | 正向 | — | 段机迭代器/确认窗/soft-end 三分支/commit 时序/turns.go 与上游逐锚同构（审计确认）；音频钟冻结与上游 block 模式同构；reopen 拉伸经 commit/CLOSED 相位裁定**不可达** | |

### E2. 协议状态机新增分歧

| # | 严重度 | 上游 | 我们 | 后果 |
|---|---|---|---|---|
| P1 | **高** | interrupt 关：speech_started 照发但**刻意不取消**，响应继续流式（audio.py:361-367, router:923-925） | 前端 speech_started **无条件** flush 队列+ring（VoiceClientV2.ts:149-154）——两路审计独立命中同一分歧 | 勿打断模式下每次 VAD 触发掐断在播回复→服务端续推→断续；直击勿打断语义 |
| P2 | **高** | 取消=close_pending_responses 全取消（含排队响应）；正常流单响应互斥 | pcmQueue flush() 全代际清空 vs FIFO 排队决策矛盾：任何一次 flush 丢掉排队中所有响应音频 | 掐断+半句续播+新回复接半句 |
| P3 | 中 | incomplete：不发尾包但已发音频有效（response.py:1001 flush=False） | status≠completed 一律 flush | incomplete 二次截断 |
| P4 | 中 | done(completed)=发送完成信号≠播完（router:903-908 注释） | completed→idle+回执即发；残播被下一个 speech_started flush 掐尾 | 回声门估时不足时尾段被掐 |
| P5 | 低-中 | reopen：cancel 后不发第二个 speech_started、同 item 复用 | 'interrupted' 相位只能被 speech_started 清走→reopen 时卡相位整轮 | UI 错态（不影响音频） |
| P6 | 中 | transcription.failed 事件存在 | 不在 known 集→partial 残句与下轮拼接 | 转写 UI |
| — | 正向 | — | barge-in 事件序/item 恰一 stop/transcript.done 权威替换/done 幂等/completed 尾包序/**reopen item 复用两侧咬合**全部同构 | |

### E3. 修订后 P0 修复清单（全部对齐上游）

1. **服务端发送循环**：A1 停靠原地等待零丢失 + A2 done 严格后置（消灭 empty_output 内部取消）
2. **A3 response_pending 两态**：转写完成只排队，新响应由发送循环在旧终态后创建（消灭无声取代）
3. **A5 并发写加锁**（STT goroutine 槽位/inputItemByTurnRev map）
4. **客户端 B1**：ring 水位控制替代丢最旧（泵按水位灌注，同构声卡按需拉取）
5. **客户端 P1（协议）**：speech_started 冲刷加 `interruptResponseOn()` 条件；created 不清 ring
6. **P2（协议）**：incomplete 不 flush（继续播已发部分）
**P1（次批）**：V1 回声门尾窗 600ms→2000ms（对齐自记残响上界；真实播完锚点需客户端上报，随两态重构评估）+ V2 候选 deadline + V4 重连清尾窗 + A4 cancel_scope 处置 + A6 session.update 深合并 + B6 AGC 播放期冻结 + P5 interrupted 相位
**P2**：V3 跨播放拼接观察、stitch、转写配对 item 作用域、text modality 事件
