# Gaea Voice V2 重新架构总方案

> 状态：架构提案，先做技术 Spike，不直接替换现有语音链。  
> 日期：2026-11-01  
> 关联调研：
> [社区蒸馏与 V2 架构](./gaea-voice-community-distill-20261101.md)

## 0. 最终调研冻结

### 0.1 先验证 sidecar，不在第一版自建全部管线

选择 `huggingface/speech-to-speech` 作为第一个运行时 Spike，理由：

- 已实现 VAD → 流式 STT → LLM → 流式 TTS；
- 已实现 generation 取消、队列冲洗、推测 reopen、轮次延迟日志；
- 暴露 OpenAI Realtime 风格的 WebSocket/WebRTC 协议；
- LLM 可接 OpenAI-compatible `/v1/chat/completions`；
- TTS 可接 OpenAI-compatible `/v1/audio/speech`。

Pipecat 作为轮次策略和 Smart Turn 的源码参考；不先引入 Pipecat + LiveKit
双框架组合。

### 0.2 首轮模型选择

| 模型 | 实际体积 | 决策 |
|---|---:|---|
| Pipecat Smart Turn v3.2 CPU ONNX | 8.7 MB | V2 首选声学 EOT |
| FireRedChat pVAD ONNX | 3.9 MB + 83 MB speaker encoder | 第二阶段候选 |
| FireRedChat Chinese EOT q8 | 178 MB | 中文文本 EOT A/B |
| sherpa-onnx streaming zipformer 14M | 78 MB | 低延迟中文流式 ASR 候选 |
| Paraformer streaming | 848 MB | 中文质量 ASR 候选 |
| FireRedASR2-AED | 4.5 GB | 暂不作为桌面默认 |
| Easy Turn | 3.45 GB checkpoint | teacher/蒸馏，不作首版运行时 |
| TEN Turn Detection | 约 15 GB | 离线 teacher，不作桌面运行时 |

结论：

- V2 第一版只用 Smart Turn v3.2 CPU，保证体积和 CPU 延迟。
- 中文 EOT 先与 FireRed Chinese EOT 做 A/B，不直接上 Easy Turn/TEN。
- ASR 先用现有 sherpa streaming 与 Paraformer 比较中文延迟/WER。
- FireRed pVAD 在 VAD 误触发率不达标时再引入。

### 0.3 Windows 风险

`speech-to-speech` 的 `pyproject.toml` 已包含 Windows 条件依赖，但没有专门 Windows
部署文档。M1 Spike 必须把以下事项作为硬门禁：

- Windows 11 安装成功；
- 不安装 CUDA 也能启动；
- 启动后空闲内存；
- sidecar 停止/重启是否干净；
- Wails 退出后子进程是否残留；
- 端口冲突与随机端口；
- 安装包增量大小；
- 断网后是否仍能完全工作。

若 sidecar 失败，不推倒协议，只切换实现为 Go 控制面 + Python/ONNX turn service。

### 0.4 首轮 Spike 配置

目标是复用现有模型，而不是先下载新的 4B 模型：

```text
STT:
  stt=paraformer
  对照：当前已安装 sherpa streaming zipformer 14M

LLM:
  llm_backend=chat-completions
  base_url=http://127.0.0.1:8080/v1
  模型暂用当前模型验证管线；延迟门禁用下一轮 4B～8B 模型

TTS:
  tts=openai
  openai_tts_base_url=http://127.0.0.1:8012/v1
  openai_tts_model=breeze-tts-2
  openai_tts_response_format=pcm
  openai_tts_sample_rate=24000
  openai_tts_stream=true
```

首轮只验证管线正确性、取消和音频连续性；35B 模型不用于最终延迟验收。

### 0.5 下个会话的首个代码切片

按以下顺序开始，不直接改生产聊天页：

1. 新建独立 `voice-v2-spike` 启动脚本与配置，验证 Windows sidecar。
2. 写一个 CLI WebSocket 客户端，发送录制 PCM，接收转录/文本/PCM，验证打断和 generation。
3. 固化协议类型与事件回放测试。
4. 再实现 `VoiceClientV2` 和 AudioWorklet 最小页面。
5. 通过 M1 门禁后才接入 ChatPage。

首个 PR 不包含：

- 删除 V1；
- 修改现有聊天数据库；
- 修改 35B 文本路由；
- 替换现有语音设置页；
- 训练 Easy Turn 学生模型。

## 1. 结论先行

不再继续扩写 `internal/voice/voice_manager.go`。V2 应当是一个独立的实时语音运行时，
通过版本化协议连接桌面端和 Gaea 对话大脑：

```text
Gaea WebView / AudioWorklet
        ⇅ WebSocket: PCM + Realtime events
Gaea Voice Runtime
        ├─ Session / TurnController
        ├─ VAD + EOT + Barge-in
        ├─ ASR adapter
        ├─ Dialogue adapter → Gaea chat
        ├─ Voice LLM → 快速 4B～8B
        └─ TTS adapter → Breeze
```

第一优先级不是重写 Go 状态机，而是先做一个 **speech-to-speech 或 Pipecat sidecar Spike**：
复用成熟项目的轮次、取消、推测生成和音频管线。若 Windows 打包成本可接受，直接保留；
否则只提取其协议和控制器，模型始终使用外部进程隔离。

## 2. 本方案要解决的问题

当前 V1 的问题是结构性的：

- 固定静音阈值同时承担断句、打断和防噪。
- interim、final、response、audio 没有统一的 `turn_id + generation`。
- 句末、LLM、TTS、播放各自控制状态，难以证明延迟归属。
- “同句重复”、“迟到音频”、“误打断后恢复”都是后补条件。
- 语音和文本共用 35B 模型及长人格 Prompt，首 token 被上下文和预填充拖慢。
- 浏览器事件、Wails 事件和本地服务各自存在队列，缺少单一有序输出。

## 3. 不可妥协的设计约束

1. **一切输出带代际**：所有文本和音频必须带 `turn_id`、`revision`、`response_id`、`generation`。
2. **轮次开始、软结束、正式提交、打断是不同事件**。
3. **取消立即生效**：取消只使旧 generation 失效，不做“等一帧看看”的时序猜测。
4. **播放有序**：音频到达顺序不等于播放顺序，播放器维护唯一有序队列。
5. **模型不阻塞控制面**：ASR、LLM、TTS 全部通过工作线程或外部服务运行。
6. **固定静音只做兜底**：语义 EOT 优先，固定阈值用于超时保护。
7. **对话历史与动态上下文分离**：历史保存原始短对话，动态状态只进当前轮。
8. **语音模型独立路由**：语音默认使用低延迟 4B～8B，文本可继续使用 35B。
9. **所有优化由分阶段指标证明**，禁止只凭主观感受继续调常数。
10. **旧链路可随时回退**：V2 通过 feature flag 接入，V1 至少保留一个版本周期。

## 4. 社区结论

### 4.1 已确认可用的组件

| 组件 | 建议 | 理由 |
|---|---|---|
| Pipecat Smart Turn v3.2 | 首选声学 EOT | ONNX、CPU、8 秒音频、阈值 0.5、BSD-2、约 10～100ms |
| FireRed pVAD | 第二阶段候选 | 说话人 embedding，能抑制非主说话人和噪声 barge-in |
| FireRed Chinese EOT | 中文第二意见 | 文本输入、中英模型、Apache-2、可由 LiveKit 插件直接接 |
| TEN VAD | VAD A/B 候选 | 10/16ms hop、低 CPU；需与 Silero/FireRed pVAD 实测 |
| Easy Turn | 教师模型与评测基准 | 四态 complete/incomplete/backchannel/wait；1145 小时公开训练集 |
| Breeze SSE | 保留 TTS | 实测首段约 280ms，短句完整音频约 750ms |

TEN Turn Detection 使用约 7B 语言模型，推理成本偏高，适合作为离线教师或服务端能力，
不作为桌面端主路径。

### 4.2 可复用但不直接依赖的框架

| 项目 | 借鉴内容 |
|---|---|
| Hugging Face speech-to-speech | 线程+队列管线、Realtime 协议、generation cancel、speculative reopen |
| Pipecat | turn start/stop 策略、Smart Turn、评测场景 |
| LiveKit Agents | endpointing 默认值、adaptive interruption、false interruption 恢复 |
| FireRedChat | pVAD + EOT + turn controller、全双工系统指标 |
| backchannel | 单一有序事件流、微模型隔离线程、约 200ms barge-in 设计 |
| avincodes/voice-agent | 明确的分阶段延迟预算与任务取消 |
| Qwen Audio Agent | ACP/Agent 后端、语音前端与任务结果回流 |

### 4.3 社区采用的初始参数

这些只是 A/B 初始值，不是最终产品常数：

```text
VAD min speech:          160～384 ms
Endpoint min delay:      300～500 ms
Endpoint max delay:      2500～3000 ms
Soft-end reopen grace:   800 ms
Barge-in confirmation:   200～500 ms
False interruption wait: 2000 ms
Backchannel boundary:    1000 ms
Preemptive LLM:          ON
Preemptive TTS:          OFF
```

## 5. V2 总体架构

### 5.1 进程边界

```text
┌──────────────────────────────────────────────────────────┐
│ Gaea Desktop (Wails / WebView2)                          │
│ VoiceClientV2                                            │
│  ├─ AudioWorklet microphone capture                      │
│  ├─ PCM playback ring buffer                             │
│  ├─ transcript / state renderer                          │
│  └─ WebSocket realtime client                            │
└──────────────────────┬───────────────────────────────────┘
                       │ ordered events
┌──────────────────────▼───────────────────────────────────┐
│ Gaea Voice Runtime V2                                    │
│  SessionManager                                          │
│  ├─ TurnController                                       │
│  ├─ InterruptionController                               │
│  ├─ SpeechPipeline                                       │
│  │   ├─ VAD/EOT adapter                                  │
│  │   ├─ ASR adapter                                      │
│  │   ├─ Dialogue adapter                                 │
│  │   └─ TTS adapter                                      │
│  ├─ Response generation ledger                           │
│  └─ Metrics exporter                                     │
└──────────────────────┬───────────────────────────────────┘
                       │ OpenAI-compatible HTTP / child process
┌──────────────────────▼───────────────────────────────────┐
│ Existing Gaea Brains                                     │
│  ├─ Herdsman: fast voice model / text model              │
│  ├─ Whisper/FireRed ASR                                  │
│  ├─ Breeze TTS                                           │
│  └─ Gaea persona + memory                                │
└──────────────────────────────────────────────────────────┘
```

运行时可先采用 sidecar：

- `speech-to-speech` Spike：最快验证完整轮次/取消协议；
- Pipecat Spike：模块更清晰，但需要自己组装 local transport；
- 两者都必须通过同一 `VoiceClientV2` 协议，不允许前端感知实现差异。

### 5.2 代码边界

建议新增：

```text
cmd/gaea-voice-runtime/
internal/voicecore/
  protocol/
  session/
  turn/
  interruption/
  speech/
  dialogue/
  metrics/
frontend/src/voice/
  audio-worklet/
  realtime-client/
  session-store/
  playback-queue/
```

不把 V2 逻辑继续放进 `internal/voice`；V1 只保留为 fallback adapter。

## 6. Realtime 协议

协议以 OpenAI Realtime 事件形状为基线，避免重新发明 transport：

### 6.1 Client → Runtime

```text
session.update
input_audio_buffer.append
input_audio_buffer.commit
input_audio_buffer.clear
conversation.item.create
response.create
response.cancel
```

### 6.2 Runtime → Client

```text
session.created / session.updated
input_audio_buffer.speech_started / speech_stopped
input_audio_buffer.committed
conversation.item.input_audio_transcription.delta / completed
response.created
response.text.delta / done
response.audio.delta / done
response.done
error
metrics
```

### 6.3 Gaea 扩展

```text
input_audio_buffer.soft_ended
input_audio_buffer.reopened
response.superseded
session.latency_trace
```

每个事件至少带：

```json
{
  "turn_id": "turn_42",
  "revision": 1,
  "response_id": "resp_42_1",
  "generation": 17
}
```

音频优先使用 binary frame；兼容期可以先 base64，但不能长期依赖。

## 7. TurnController 状态机

```text
IDLE
  └─ speech_started → LISTENING
LISTENING
  ├─ silence_candidate → SOFT_ENDED
  └─ hard_timeout → COMMITTING
SOFT_ENDED
  ├─ speech_resumed → LISTENING（revision + 1）
  ├─ EOT incomplete → LISTENING / WAIT
  └─ EOT complete + final text → COMMITTING
COMMITTING
  └─ response accepted → THINKING
THINKING
  ├─ first text → DISPLAYING
  └─ first audio → SPEAKING
SPEAKING
  ├─ barge-in confirmed → CANCELLING
  ├─ false interruption → resume SPEAKING
  └─ playback done → CLOSED
```

规则：

- 用户再次开口不总是新 turn；在 reopen grace 内可以提升同一 turn 的 revision。
- barge-in 只取消当前 response generation，不取消 session、麦克风和 ASR。
- 工具调用默认不因语音打断回滚；只取消用户可见的 TTS/LLM 生成。
- 被取消的 assistant 文本按“用户实际听到的位置”截断后写入历史。

## 8. 四态轮次判定与蒸馏

目标模型输出四态：

```text
complete     用户说完了，可以回答
incomplete   用户只是句中停顿，应继续听
backchannel  嗯、啊、对、好等反馈，不抢占发言权
wait         用户要求暂停、思考或不要立即回答
```

### 8.1 第一阶段：直接使用开源模型

- 声学：Pipecat Smart Turn v3.2 ONNX；
- 文本：FireRed Chinese/Multilingual EOT；
- 噪声/主说话人：FireRed pVAD；
- 低延迟 VAD：TEN VAD 与 Silero 做 A/B。

判定建议：

```text
VAD speech end
  ├─ Smart Turn != complete → 保持 LISTENING
  ├─ FireRed EOT = incomplete → 保持 LISTENING
  ├─ 两者冲突 → 低成本文本二次判定
  └─ 两者均 complete → COMMIT
```

低置信度区间不做硬猜，延长到 max endpointing 后兜底提交。

### 8.2 第二阶段：小模型蒸馏

Easy Turn 同时使用声学和语言学信息，四态准确率和数据规模适合作为 teacher。
TEN 7B 和 FireRed EOT 可作为第二教师。

蒸馏目标：

- Student 输入：最近 6～8 秒 16k PCM + ASR 稳定文本 + 最近一句 assistant 文本；
- Student 输出：四态 softmax；
- Teacher：Easy Turn / TEN / FireRed / Smart Turn 集成；
- 蒸馏损失：teacher KL + hard-label CE + backchannel/wait 加权；
- 部署：ONNX INT8，CPU 可跑，目标 p95 ≤ 80ms；
- 语料：Easy Turn 公开训练/测试集 + TEN 中英文测试集 + Gaea 自有脱敏样本。

只收集用户明确开启“帮助改进离线语音”的样本；默认不上传。

## 9. 打断与恢复策略

### 9.1 两阶段插话

1. 检测到用户开口：立即 duck TTS 音量 30%～50%，暂不取消。
2. 满足以下任一条件才正式 barge-in：
   - 连续语音达到 200～500ms；
   - interim 出现至少 2 个有效词/字符；
   - 主说话人 VAD 确认；
   - TurnCut/pVAD 判定为真实插话。

### 9.2 False interruption

- 真插话：generation + 1，flush 播放和 TTS 请求，提交新 turn。
- 误触发：2 秒内没有有效转写，恢复原 generation 的未播放音频。
- backchannel：不打断，只记录到对话上下文。

## 10. 前端重设计

### 10.1 麦克风

- `AudioWorklet` 固定 20ms/320 sample @16k 输出。
- getUserMedia 开启 `echoCancellation / noiseSuppression / autoGainControl` 并可配置。
- 保留预录音缓冲区，确保第一个音素不被 VAD 裁掉。
- PCM 不再经 ScriptProcessor，也不每块 base64 化。

### 10.2 播放器

- TTS 输出统一转入 PCM ring buffer。
- 不把每个 SSE chunk 包成独立 WAV 再交给 `<audio>`；这会引入块间空隙。
- 播放器只接收：

```text
response_id, generation, PCM16, sample_rate, sequence
```

- barge-in 时只 flush 当前 generation 的未播放 chunk。
- 播放完成回执必须带 response/generation，不能使用全局布尔状态。

### 10.3 UI 状态

UI 由协议事件直接驱动：

```text
LISTENING / PAUSED / THINKING / SPEAKING / INTERRUPTED / RECONNECTING / ERROR
```

禁止再用 volume 阈值猜测 AI 是否在说话。

## 11. Gaea 对话大脑接入

新增一个内部 OpenAI-compatible Stream endpoint：

```text
POST /internal/voice/chat/completions
```

Voice Runtime 只提交本轮稳定转写和必要上下文；Gaea 负责：

- 人格、记忆、关系状态；
- 工具调用和权限；
- 文本落库；
- 快速模型路由；
- 最终消息写回。

语音 Prompt 预算：

```text
稳定人格段：600～1000 字
当前动态状态：400～800 字
历史：2 轮原始短对话
输出上限：160～240 tokens
```

动态上下文不能整段复制到每一轮历史；历史必须是不可变、短、可缓存的消息。

## 12. 模型路由

新增配置而不是继续复用文本功能绑定：

```text
voice_chat_engine
voice_chat_model
voice_asr_engine
voice_asr_model
voice_tts_engine
voice_tts_model
voice_eot_model
```

默认建议：

- text chat：35B；
- voice chat：4B～8B，高优先级常驻；
- ASR：FireRedASR / Whisper-small / sherpa streaming；
- TTS：Breeze SSE；
- EOT：Smart Turn v3.2 + FireRed Chinese EOT A/B；
- pVAD：FireRed Chat 第二阶段接入。

## 13. 观测与评测

### 13.1 每轮 trace

```text
turn_id / revision / response_id / generation
speech_started_at
speech_ended_at
vad_decision_ms
eot_model_ms
eot_label / eot_confidence
stt_first_partial_ms / stt_final_ms
llm_ttft_ms / llm_total_ms / prompt_tokens / cache_hit_tokens
tts_ttfa_ms / first_pcm_at
playback_start_ms
barge_in_detected_ms / barge_in_to_silence_ms
false_interruption / reopen_count
```

### 13.2 语料

- Pipecat turn-completion：greeting、short_complete、continuation、cutoff、drift、long、reprompt、tools；
- TEN 中英文 finished/unfinished/wait 测试集；
- Easy Turn 四态测试集；
- Gaea 自有测试：中文口语、方言口音、情绪语音、AI 自播回声、儿童声、多人环境。

### 13.3 门禁

| 指标 | V2 门槛 |
|---|---:|
| speech_end → first audio p50 | ≤ 2.5s |
| speech_end → first audio p95 | ≤ 4.0s |
| 中文 EOT accuracy | ≥ 90% |
| barge-in → 静音 p95 | ≤ 500ms |
| 误打断率 | ≤ 3% |
| 迟到/串台音频 | 0 |
| 同会话缓存命中 | ≥ 70% |
| 语音首轮输入 token | ≤ 2500 |

## 14. 分阶段实施

### M0：调研冻结（完成）

输出：社区调研、组件选型、架构约束、测试语料清单。

### M1：Runtime Spike

分别用 `speech-to-speech` 和 Pipecat 做最小可运行 sidecar：

- 16k PCM 输入；
- 流式转写；
- 固定短 system prompt；
- Herdsman 4B/8B；
- Breeze PCM 输出；
- cancel + generation。

退出条件：能测出真实 p50/p95、Windows 安装体积、启动时间和空闲内存。

### M2：Protocol + Mock Runtime

- 确定 Realtime 协议；
- 做无模型回放测试；
- 前端 AudioWorklet 和 PCM ring buffer；
- 用脚本 PCM 验证轮次、打断、恢复、迟到帧。

退出条件：不需要真实模型即可完整回放并断言事件有序性。

### M3：VAD + EOT

- 接入候选 VAD；
- 接入 Smart Turn v3.2；
- 可选 FireRed Chinese EOT；
- 建立四态测试集和实时指标。

退出条件：中文 EOT ≥90%，误打断 ≤3%，EOT 推理 p95 ≤80ms。

### M4：Gaea Dialogue + TTS

- 对话 endpoint；
- 快速语音模型；
- Breeze SSE；
- PCM 播放调度；
- 人格/记忆写回。

退出条件：首音频门禁达标，文本与语音消息一致。

### M5：推测生成与恢复

- soft end 后预生成；
- revision/reopen；
- false interruption resume；
- 工具调用保护。

退出条件：自然停顿不切句，误打断可恢复，旧 generation 无泄漏。

### M6：蒸馏

- 收集四态数据；
- teacher 集成伪标；
- 训练小型双模态 student；
- ONNX INT8；
- 离线 A/B。

退出条件：学生模型质量接近 teacher，CPU p95 ≤80ms，模型体积可控。

## 15. 回退与迁移

- `voice.backend=v2` feature flag 控制整体切换。
- UI 同时提供 V1/V2 选择仅用于开发。
- 数据层继续复用 ChatAppendMessages 和 VoiceConsole 外层布局。
- V2 稳定后才删除旧 `useVoiceChat` / `VoiceManager`。
- 失败时自动回 V1，但不能在 V2 内悄悄降级成套壳 V1。

## 16. 当前技术决策

推荐顺序：

1. 先做 `speech-to-speech` sidecar Spike，因为它已经包含 Realtime 协议、generation cancel、
   speculative reopen 和分阶段延迟日志。
2. 同时用 Pipecat Smart Turn 源码/模型做 EOT 对照。
3. Sidecar 打包不合格时，保留相同协议，改用 Go 控制面 + Python/ONNX turn service。
4. 不直接引入 LiveKit Server。
5. 不先做端到端 Moshi；它是后续独立实验，不是 V2 前置条件。

## 17. 调研快照

本次读取的 GitHub HEAD：

```text
Pipecat             6f777a0ef3bb98163d10d708525a0db21cda5fbf
speech-to-speech    6feb527838bfd7e4f6e911b1808d76c722dbbe56
LiveKit Agents      c168b713e281d6ad2cd4ceef8f0f107802499b54
FireRedChat         9d8c4b99fbd27380f6bf44f9b2f78ae615229a55
```

许可证初查：

- Pipecat：BSD-2-Clause；
- speech-to-speech：Apache-2.0；
- LiveKit Agents：Apache-2.0，模型另看 model license；
- FireRedChat / pVAD / turn detector：Apache-2.0；
- Smart Turn v3：BSD-2-Clause；
- Easy Turn：Apache-2.0；
- TEN：Apache-2.0 加附加条件；
- TurnCut：MIT；
- Moshi：代码与模型分别核对。

## 18. 参考链接

- [Pipecat](https://github.com/pipecat-ai/pipecat)
- [Pipecat Smart Turn](https://github.com/pipecat-ai/smart-turn)
- [Hugging Face speech-to-speech](https://github.com/huggingface/speech-to-speech)
- [LiveKit turn handling](https://docs.livekit.io/agents/build/turns/)
- [FireRedChat](https://github.com/FireRedTeam/FireRedChat)
- [FireRed pVAD](https://github.com/FireRedTeam/livekit-plugins-fireredchat-pvad)
- [FireRed Turn Detector](https://github.com/FireRedTeam/livekit-plugins-fireredchat-turn-detector)
- [TEN VAD](https://github.com/TEN-framework/ten-vad)
- [TEN Turn Detection](https://github.com/TEN-framework/ten-turn-detection)
- [Easy Turn](https://github.com/ASLP-lab/Easy-Turn)
- [TurnCut](https://github.com/ZECTBynmo/turncut)
- [backchannel](https://github.com/nabaruns/backchannel)
- [avincodes/voice-agent](https://github.com/avincodes/voice-agent)
- [Qwen Audio Agent](https://github.com/GHJU20001017/Full-Duplex-Model/tree/main/qwen-audio-agent)
- [Moshi](https://github.com/kyutai-labs/moshi)
