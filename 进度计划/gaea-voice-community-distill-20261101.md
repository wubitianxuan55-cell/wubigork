# Gaea 语音交互社区蒸馏与 V2 架构

> 日期：2026-11-01  
> 目标：停止继续微调单点阈值，先从成熟开源语音栈提取可复用架构、协议和评测方法。  
> 结论：Gaea 当前不是缺一个“更准的静音阈值”，而是缺少独立的轮次控制器、
> 推测生成、代际取消和分阶段延迟预算。

## 1. 社区基线

| 项目 | 核心方案 | 对 Gaea 的价值 |
|---|---|---|
| Pipecat | 可组合 frame pipeline；Smart Turn v3.2 ONNX；VAD、STT、LLM、TTS 分层 | 轮次策略、EOT 模型、评测集设计 |
| LiveKit Agents | 语义轮次检测、adaptive interruption、false interruption 恢复、preemptive generation | 防误打断、回退恢复、推测式生成 |
| TEN Framework | 10/16ms 流式 VAD；Qwen2.5 语义 turn detector；中英文测试集 | 中文 EOT 和低延迟 VAD 候选 |
| FireRedChat | pVAD + 语义 EoT + turn-taking controller；级联/半级联/端到端 | 全双工控制面与系统级指标 |
| Hugging Face speech-to-speech | VAD→STT→4B LLM→TTS，四线程队列；OpenAI Realtime 风格协议；推测式 reopen | 最接近本项目可借鉴的完整工程实现 |
| Moshi | 原生全双工 speech-to-speech，Mimi + Temporal Transformer | 长期方向；当前桌面/模型成本不适合先行 |

## 2. 蒸馏出的关键不变量

### 2.1 轮次开始与结束分离

- 用户开口只产生 `user_speech_started`，不等同于新回合已经提交。
- 用户停止发声只产生候选 `user_speech_stopped`。
- 语义 EOT、最终转写和最短等待共同决定 `user_turn_committed`。
- 用户重新开口可撤销 soft end，而不是直接丢弃上一轮再开新请求。

### 2.2 模型只负责对话，不负责抢麦

- VAD 负责声学活动。
- EOT 负责“是否说完”。
- interruption classifier 负责“是真插话还是嗯/啊等 backchannel”。
- LLM 不根据一句 interim 猜测是否应该打断。

### 2.3 所有输出必须带轮次代际

每个响应、文本 delta、TTS chunk 都携带：

```text
turn_id
turn_revision
response_id
generation
```

取消时递增 `generation` 并冲洗播放队列。旧代际的后续 token、音频 chunk 和完成回执
只能丢弃，不能污染新回合。

### 2.4 推测生成，提交后放行

- soft EOT 后立即启动 voice LLM，可选启动 TTS。
- 用户若继续说话，撤销推测响应。
- 用户若确认说完，转写一致则复用生成结果。
- 推测 TTS 默认应谨慎开启，因为它会增加取消浪费；先只推测 LLM。

### 2.5 固定静音只能做安全兜底

- 主路径使用语义 EOT 模型，不以固定 1 秒作为唯一判断。
- Pipecat Smart Turn v3.2 使用 16kHz、最近 8 秒音频，模型概率阈值 0.5，
  默认最多 3 秒静音兜底。
- LiveKit 推荐端点为 `min_delay=500ms`、`max_delay=3000ms`，并结合语义 turn detector。
- TEN 使用 10/16ms hop 的流式 VAD，再叠加语义 EOT。

### 2.6 误打断必须可恢复

- 真插话：停止播放、取消生成、清空未播音频、提交新回合。
- 短暂噪声/backchannel：不取消回答，或先 duck 音量再等待判定。
- LiveKit 默认在 2 秒无有效转写后视为 false interruption，并从原位置恢复回答。

### 2.7 延迟必须分阶段衡量

至少记录：

```text
vad_decision_ms
eot_inference_ms
stt_ms
llm_ttft_ms
llm_total_ms
tts_ttfa_ms
first_audio_ms
false_interruption_count
stale_audio_drop_count
```

禁止只用一个端到端秒数判断优化是否有效。

## 3. V2 目标架构

```text
WebView 麦克风
  -> 16k PCM16 + AEC/NS
  -> Streaming VAD
       -> SpeechStarted / SpeechStopped
  -> TurnController
       -> SoftEnd
       -> Semantic EOT
       -> Commit / Resume / Cancel
  -> Voice Dialogue Runtime
       -> Fast voice LLM
       -> Generation-tagged text stream
  -> Streaming TTS
       -> Clause chunks + generation
  -> Playback Router
       -> ordered playback queue
       -> generation gate
       -> barge-in flush
```

建议新增独立接口契约，不继续把逻辑塞进现有 `VoiceManager`：

```go
type TurnSignal struct {
    TurnID     string
    Revision   int
    Type       TurnSignalType
    SpeechStart time.Time
    SpeechEnd   time.Time
    Confidence float64
    Transcript string
}

type AssistantFrame struct {
    TurnID     string
    Revision   int
    ResponseID string
    Generation uint64
    Kind       AssistantFrameKind
    Audio      []byte
    Text       string
    MimeType   string
}
```

## 4. 推荐落地路线

### Step 0：冻结单点手调

不再直接修改现有静音阈值、句号切分和播放状态机。保留当前实现作为 fallback。

### Step 1：先做框架 Spike

以 `voice-v2` feature flag 启动 Hugging Face speech-to-speech 或 Pipecat 的本地 sidecar，
复用其：

- VAD + Smart Turn；
- turn controller；
- generation-tagged cancellation；
- OpenAI Realtime 风格事件协议；
- latency metrics。

LLM 先指向本地 4B～8B 快速模型，不接当前 35B 文本模型。

### Step 2：用中文语料做一次 A/B

至少覆盖：

- 已完成短句；
- 句中自然停顿；
- 思考后继续；
- 用户改口；
- 嗯/啊 backchannel；
- 环境噪声误触发；
- AI 播报中真插话；
- 长时间连续说话。

Pipecat 已提供 greeting、short_complete、continuation、cutoff、drift、long、reprompt、
tools 等 turn-completion 场景；TEN 提供中英文 EOT 测试集。

### Step 3：决定“接入还是提取”

若 sidecar 在安装体积、启动时间和 Windows 打包上可接受，直接保留框架。

否则只提取四项，不复制整套框架：

1. Smart Turn / TEN EOT 模型；
2. TurnController 状态机；
3. generation-tagged event protocol；
4. latency / false-interruption 评测与指标。

### Step 4：切换语音模型

文本聊天和语音对话分离路由：

- 文本：当前 35B；
- 语音：4B～14B 热模型；
- 固定短 system prompt；
- 2～4 轮原始短历史；
- 禁止把整份动态上下文快照写入历史。

### Step 5：流式 TTS 与播放器

- TTS 首段优先，不等待整句 WAV；
- clause chunk 携带 generation；
- 播放器按顺序排空，不按到达顺序乱播；
- barge-in 只 flush 当前及旧 generation；
- 不允许旧 chunk 触发新回合的 playback done。

## 5. 验收指标

第一阶段目标（本地 4B～8B 热模型）：

| 指标 | 目标 |
|---|---:|
| 语音结束到首音频 p50 | ≤ 2.5s |
| 语音结束到首音频 p95 | ≤ 4.0s |
| 中文 EOT 准确率 | ≥ 90% |
| 真插话成功率 | ≥ 95% |
| 误打断率 | ≤ 3% |
| 旧代际音频泄漏 | 0 |
| LLM 首 token p50 | ≤ 1.2s |
| TTS 首段 p50 | ≤ 0.5s |

所有指标按 VAD、EOT、STT、LLM、TTS、播放分段输出，不能只报总延迟。

## 6. 明确不做

- 不把 LiveKit WebRTC 服务端一并引入桌面产品。
- 不先上 Moshi 端到端模型；需要新的模型资产和推理预算。
- 不复制 Pipecat/TEN 的全部 provider 适配层。
- 不在 V2 稳定前继续重写当前 `VoiceManager`。
- 不把“用户可见文字相同”误当成“完整 Prompt 相同”。

## 7. 参考来源

- Pipecat: <https://github.com/pipecat-ai/pipecat>
- Pipecat Smart Turn:
  <https://github.com/pipecat-ai/pipecat/blob/main/src/pipecat/audio/turn/smart_turn/base_smart_turn.py>
- Pipecat turn strategies:
  <https://github.com/pipecat-ai/pipecat/blob/main/src/pipecat/turns/user_turn_strategies.py>
- LiveKit turns:
  <https://docs.livekit.io/agents/build/turns/>
- LiveKit turn tuning:
  <https://docs.livekit.io/agents/logic/turns/tuning/>
- TEN Turn Detection:
  <https://github.com/TEN-framework/ten-turn-detection>
- TEN VAD:
  <https://github.com/TEN-framework/ten-vad>
- FireRedChat:
  <https://github.com/FireRedTeam/FireRedChat>
- FireRedChat paper:
  <https://arxiv.org/abs/2509.06502>
- Hugging Face speech-to-speech:
  <https://github.com/huggingface/speech-to-speech>
- Hugging Face Realtime protocol design:
  <https://github.com/huggingface/speech-to-speech/tree/main/src/speech_to_speech/api/openai_realtime>
- Moshi:
  <https://github.com/kyutai-labs/moshi>
