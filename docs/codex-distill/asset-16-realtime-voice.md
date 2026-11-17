# 资产 16 · 实时语音会话（评估：不做双工，借三条交接机制）

> 源码：`core/src/realtime_conversation.rs`（**103 KB**）+ `realtime_conversation/{transcript_tail,sideband,existing_call}.rs`、
> `core/src/realtime_context.rs`（20 KB）、`core/src/context/realtime_delegation.rs`、
> `realtime-webrtc/`、`voice-host/`、`codex-api/src/endpoint/realtime_{call.rs,websocket/}`。
> 对照：gaea `internal/voice/`（TTS 向：`emotion_voice_map.go` 8 KB、`voice_config.go`、`native/`）+ `internal/tts/`。

---

## 一、结论先说

**不做云端实时双工**，理由三条：

1. 它的底座是 **OpenAI realtime WebSocket + WebRTC 音频帧**（`RealtimeAudioFrame` / `RealtimeWebsocketClient` /
   `RealtimeSessionMode`），gaea 是本地推理栈（sherpa-onnx），没有对应服务端；
2. gaea 现在只有 **TTS**（文本→语音），**没有 STT 回路**——实时双工要有「麦克风流 → 转写 → 回合」整条链路；
3. `realtime_conversation.rs` 单文件 103 KB，配套 webrtc + voice-host（GStreamer），移植成本远超收益。

**但有三条交接机制值得借**，它们与云端无关，纯粹是「旁路语音会话怎么和主线程对接」：

| 机制 | 价值 |
|---|---|
| 转写落地为「真回合」 | 语音内容进历史，可检索、可回放，不是旁路日志 |
| 委托片段（带标记与来源） | 主线程知道「这段是语音移交来的」，且可被识别/剥离 |
| XML 转义 + 有界保留 | 防伪标记注入，且不会把长语音全文塞进上下文 |

---

## 二、机制拆解（可借的三条）

### 2.1 转写落地为真回合（`transcript_tail.rs` 4 KB）

模块注释一句话点题：*"Saves terminal speech in model and client history before realtime closure, without inference."*
——在实时会话关闭**之前**，把结束语存进模型与客户端历史，**不触发推理**。

实现要点：
1. 生成新的 `sub_id`；
2. **若当前没有正在跑的回合**，先补一条合成 `TurnStarted` 事件并 `record_started_turn`——
   这样这段转写在历史里是「一个真回合」，而不是孤立消息；
3. 记录对话条目、持久化 rollout、发 `ItemCompleted`（`thread/items/list` 的数据源）。

**这句注释和这三个步骤是整篇最值钱的地方**：语音输入最容易做成「旁路通道」，
结果是用户说完一段话，历史里查不到、下一轮模型也看不到，只有语音面板自己的日志有。

### 2.2 委托片段（`realtime_delegation.rs` 3.2 KB）

```rust
content_kind = "realtime_conversation.delegation"
role         = "user"
type_markers = ("<realtime_delegation>", "</realtime_delegation>")
enum Source  = { Handoff, TranscriptTailFlush }
// TranscriptTailFlush 时正文里渲染 <source>transcript_tail_flush</source>
const MAX_REALTIME_DELEGATION_FIELD_BYTES = 4 * 1024;
const TRUNCATION_MARKER = "…";
```

它把「这段是移交来的」表达成**带标记的 user 片段**，而不是塞一句自然语言说明。
好处与 asset-12 一致：可被 `MatchesText` 识别、可在压缩时按 `Kind` 替换、可被 Guardian 按 `Prov` 判定。

**注意 `role = "user"`**：它故意用 user 角色，但带自己的 `<source>` 标记——
于是「用户真的说话」与「系统移交」在**内容分类**上区分，而不是靠角色区分。这正好印证 asset-12 那条纪律：
**归属与角色分离**。

### 2.3 有界与转义

`escape_xml_text_bounded(input, Retain::Start)`：XML 转义 + 只保留**开头** 4 KiB，超限加 `…`。
保留开头而不是结尾，是因为「用户说了什么」重点在开头；结尾的寒暄可以丢。

## 三、常量表（若要复刻，这些是全部硬数字）

```
AUDIO_IN_QUEUE_CAPACITY             = 256
TEXT_IN_QUEUE_CAPACITY              = 64
HANDOFF_OUT_QUEUE_CAPACITY          = 64
OUTPUT_EVENTS_QUEUE_CAPACITY        = 256
REALTIME_THREAD_SOURCE_MAX_BYTES    = 256
REALTIME_STARTUP_CONTEXT_TOKEN_BUDGET = 5_300
REALTIME_ASSISTANT_OUTPUT_TOKEN_BUDGET = 1_000
REALTIME_REASONING_STATUS_MAX_BYTES = 256
REALTIME_INITIAL_ITEMS_MAX_COUNT    = 128
REALTIME_INITIAL_ITEMS_MAX_TOKENS   = 8_192
REALTIME_MODE_INSTRUCTIONS_MAX_TOKENS = 8_192
HANDOFF_STREAM_FLUSH_INTERVAL       = 200 ms
```

**读法**：启动上下文 5.3k token / 助手输出 1k token —— 实时会话被明确限制成「小窗口高频」，
与主线程的长上下文分开算账；队列容量全是 2 的幂，说明是为了背压时有界丢弃。

---

## 四、若 gaea 将来接本地 STT，这是最小接入形态

1. 语音识别完成 → 得到文本 → **落成一条 user 消息**（走正常会话入口，不写旁路日志）；
2. 若当时没有活跃回合 → 补一条 `TurnStarted`（gaea 侧等价物是回合起始事件），保证历史可回放；
3. 用「委托片段」表达来源：`Kind = "voice.delegation"`、`Markers = <voice_delegation>…</voice_delegation>`、
   正文里带 `<source>stt</source>` 或 `<source>realtime_flush</source>`；
4. 有界：按 gaea 当前窗口折算（例如窗口的 3%），超出保留开头并加省略号；
5. 转义：正文里的 `<`/`>` 一律转义，避免伪造 `</voice_delegation>` 逃逸标记。

**验收**：语音输入后，历史里能查到该条消息且带来源标记；下一轮模型上下文里出现它；
伪造的 `</voice_delegation>` 文本被转义，不能提前闭合标记。

## 五、不做清单（明确排除）

| 不做 | 理由 |
|---|---|
| 云端 realtime WebSocket 双工 | 依赖 OpenAI 服务端，gaea 无对应 |
| WebRTC / `realtime-webrtc` | 音频传输栈，本地 go 应用无收益 |
| `voice-host`（GStreamer + Opus） | codex 自己的语音宿主进程，gaea 用 sherpa-onnx |
| 侧带（`existing_call` / `sideband`） | 与云端会话生命周期绑定 |

## 六、一句话

实时语音这一大块（103 KB + webrtc + GStreamer）对 gaea **不可移植**，
但「旁路会话如何把内容交回主线程」这三条（落成真回合、标记化委托、有界转义）是通用工程，
以后无论接 STT 还是接「第二个人格通道」都用得上。
