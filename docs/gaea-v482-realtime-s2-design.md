# gaea 实时语音 Realtime S2 设计定稿（v4.8.2）

> 依据：S2 只读调研报告（2026-08-30，HEAD c726da2）+ S0/S1 落地现状。
> 分刀裁决：**方案 (a)——事件环骨架 + 协议测试 + 降级护栏**；真机体感
> （延迟/AEC/打断手感/真实音频格式怪癖）如实记账为真机欠账。

## 0. 核心结论

- `internal/voice` 目前**零 import** `internal/realtime`：S2 = 把对话环驱动源
  从「本地 VAD→ASR→chat→TTS」（voice_manager.go:372-520）换成 session 事件流。
- seam 接口够一半：`SendAudio` 已存在（provider.go:45）；**会话控制类客户端
  事件全缺**（commit/clear/create/cancel）、session.update 缺 `turn_detection`
  与 `output_audio_format`、事件常量缺 6-7 个。
- **协议级硬伤**：OpenAI Realtime pcm16 = 24kHz，gaea 全链路 16kHz（前端采集
  useVoiceChat.ts:69、后端 voice_config.go:144、openai.go 注释误标 16k）。
  16k 按 24k 解读=花栗鼠音+转写崩坏。必须新增 16k→24k 线性重采样纯函数。
- **前端死门**：useVoiceChat.ts:345-353 仅在 `!browserASRAvailable` 时推送 PCM
  ——WebView2 恒有 Web Speech API，桌面端 PCM 实际从未进 `VoicePushAudio`。
  realtime 模式必须强制推送分支。

## 1. 目标事件环（realtime 模式）

```
[前端] 麦克风 PCM16/16k/200ms (useVoiceChat.ts:351)
  → VoicePushAudio → Manager.PushAudioChunk
    【realtime 分支：旁路本地 VAD/状态门(:232-234)，16k→24k 重采样】
  → session.SendAudio → input_audio_buffer.append

[事件泵 goroutine ← session.Events()]
  input_audio_buffer.speech_started → StateSpeaking 时 barge-in：
      停播放(EmitVoiceTTSCancel) + CancelResponse() + ClearBuffer()
      → StateListening
  response.created【缺常量】          → setState(Thinking)
  response.audio.delta（24k PCM 已解码）→ 聚合器累积 → StateSpeaking
  response.audio_transcript.delta/done【缺常量】→ EmitVoiceReply（对话显示）
  conversation.item.input_audio_transcription.completed/failed【缺常量】
                                      → EmitVoiceTranscript（用户侧文本）
  response.done【缺常量】             → 冲洗聚合器：
      24k PCM → 包 WAV 头（24kHz）→ EmitVoiceTTSAudio
      （复用前端现播放环 useVoiceChat.ts:112-187，零播放侧改动）
      → setState(Idle) → 自动续听（voice_manager.go:511-519 模式）
  error → 降级：关会话 → 回拼接管线（宁降级不黑屏）
```

## 2. realtime 包改动（openai.go + provider.go）

- **事件常量 +7**：response.done、response.created、
  response.audio_transcript.delta / .done、
  conversation.item.input_audio_transcription.completed / .failed、
  input_audio_buffer.committed（建议）。机制=只加白名单映射
  （openai.go:275-284 knownServerEvents），解析骨架零改动。
- **TurnControl 可选接口**（不进 RealtimeSession 主接口，type-assert，
  失败=fail-closed 回拼接管线，与注册表 fail-closed 同纪律）：
  `Commit() / ClearBuffer() / CreateResponse() / CancelResponse()`
  → input_audio_buffer.commit / clear、response.create、response.cancel。
- **session.update 扩字段**：turn_detection
  `{type:"server_vad", create_response:true, interrupt_response:true}` +
  output_audio_format:"pcm16"（显式防漂移）。修正 16k 注释为 24k。
- **重采样纯函数** `Resample16kTo24k(pcm []byte) []byte`（线性插值，
  PCM16 LE mono；放 realtime 包，独立测试文件：含空帧/非整帧/正弦保真
  用例）。SendAudio realtime 路径在 voice 侧调用（voice 侧不引 import
  循环——realtime 不依赖 voice，方向安全）。

## 3. voice 包改动（voice_manager.go）

- 注入口：`SetRealtimeSession(s realtime.RealtimeSession)`（仿
  SetASRProvider :111-124 三口模式）+ 会话 nil/断开 = 自动回拼接管线。
- `PushAudioChunk` realtime 分支：旁路状态门与本地 VAD（:232-234,
  :267-327），重采样后 SendAudio；PTT 释放（:734-746）映射 Commit()。
- **事件泵**：Dial 成功后单 goroutine 消费 Events()，状态映射
  （listening/thinking/speaking/idle 对齐 :160-165 状态机）；聚合器
  按 response 聚 audio.delta，done 冲洗包 WAV（24kHz 头，wav.go 加
  24k 变体）发 EmitVoiceTTSAudio；transcript → EmitVoiceReply；
  input transcription → EmitVoiceTranscript。泵退出/错误 → 降级。
- 打断联动：本地 RMS barge-in（:236-260）在 realtime 模式旁路（双源
  冲突），以服务端 speech_started 为准；CancelTTS（:674-705）叠加
  CancelResponse + ClearBuffer 下发。

## 4. app 层（voice_handler.go）

- initVoice：provider 配置时经 registry 构造真实 session 并注入
  Manager（替换 probeRealtime 的纯探测）；构造失败保持现降级语义。
- VoiceStart 就绪门（:386-399）：realtime 会话在位时以 realtimeReady
  为门（不再强制 ASRReady）；否则维持原样。
- **零新增 Wails 绑定**（事件全走既有 voice:* 七事件 + 四动作口）。

## 5. 前端（最小集）

- useVoiceChat.ts：realtime 模式强制推送 PCM（旁路 :345-353 的
  browserASRAvailable 门）；realtime 模式抑制 VoiceChatText 本地识别
  路径（防双输入）；模式判定 = VoiceHealth/VoiceGetSettings 的
  realtimeProvider 非空 + realtimeReady。
- VoiceSettingsPanel：运行态指示（实时/拼接）可选，不阻塞本刀。
- 播放侧零改动（后端包 WAV，前端现 decodeAudioData 直接吃）。

## 6. 测试边界

**离线可做**（httptest 假 WS + 纯函数）：重采样保真、session.update
载荷（turn_detection/output_audio_format）、事件泵状态机映射、
barge-in 三联（cancel+clear+停播）、聚合器 done 冲洗 WAV、降级路径、
TurnControl 四动作协议帧。
**真机欠账**（如实记账）：端到端延迟、AEC 实效（echoCancellation 已开
useVoiceChat.ts:310）、打断手感、真实音频格式怪癖、新模型
gpt-realtime 的 instructions 落位差异。
