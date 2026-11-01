# Gaea Voice V2 — M1 Runtime Spike 报告

> 日期：2026-11-01
> 执行方案：[V2 重新架构总方案](./gaea-voice-v2-rearchitecture-20261101.md) §0.4/§0.5 首个代码切片
> 关联调研：[社区蒸馏与 V2 架构](./gaea-voice-community-distill-20261101.md)

## 1. 结论

**sidecar 路线在 Windows 上全部硬门禁通过，M1 退出条件达成。**
`huggingface/speech-to-speech@6feb527`（v1.1.0，冻结调研 HEAD）作为独立进程在
Windows 11 + 无 CUDA 环境下完整跑通：

```
麦克风 PCM → VAD(silero) → Smart Turn(EOT) → STT(paraformer-zh, CPU)
  → LLM(herdsman 35B, chat-completions) → TTS(breeze SSE, pcm) → 下行 PCM16@16k
```

三场景协议断言全绿：**turn 8/8、cancel 8/8、bargein 10/10**。
打断代际取消语义（`cancelled/turn_detected` + 旧代际音频泄漏 0B）实证成立——
这正是 V2 方案 §3 的不可妥协约束 1/2/3/4，sidecar 已原生实现。

**意外收获**：v1.1.0 已内置 **Pipecat Smart Turn v3.2 ONNX**（实测推理 26~61ms CPU）、
VAD soft-end + 800ms speculative reopen grace、CancelScope 代际取消、
逐阶段延迟日志（`stt/llm/tts_ttfa/e2e/vad_decision/hold/smart_turn_status`）。
调研文档列为 M3 目标的 EOT 主体已在 sidecar 内，M3 只剩中文 A/B 与参数调优。

## 2. 工件

| 位置 | 说明 |
|---|---|
| `scripts/voice_v2/gaea-spike.yaml` | 管线配置（§0.4 冻结口径：paraformer / chat-completions@8080 / openai-tts@8012 pcm+stream） |
| `scripts/voice_v2/start_spike.bat` | sidecar 启动器（ASCII+CRLF，venv 指向 C:\AI\gaea-voice-v2） |
| `scripts/voice_v2/ws_probe/main.go` | CLI WebSocket 客户端：三场景 + 分阶段延迟 + 不变式断言 + 造料（breeze 合成测试语音） |
| `C:\AI\gaea-voice-v2\venv` | sidecar 虚拟环境（pip install -e <sts 克隆>[paraformer]） |
| `.tmp/voice_v2/` | 探针产物（报告 JSON / 每轮音频 WAV / 服务端日志） |

复现：

```bash
# 1) 依赖（一次性）：python -m venv C:\AI\gaea-voice-v2\venv
#    pip install -e "C:\AI\wubigrok\.tmp\research-hf-sts[paraformer]"
#    nltk punkt_tab 手动落 %APPDATA%\nltk_data（见 §5 坑）
# 2) 前置：herdsman 8080 / breeze 8012 在跑（gaea 按需拉起线或手动）
scripts/voice_v2/start_spike.bat                     # sidecar，127.0.0.1:8791
go run ./scripts/voice_v2/ws_probe -mkwav-text "你好，介绍一下你自己" -mkwav-out .tmp/voice_v2/a.wav
go run ./scripts/voice_v2/ws_probe -scenario turn    -speech .tmp/voice_v2/a.wav
go run ./scripts/voice_v2/ws_probe -scenario cancel  -speech .tmp/voice_v2/a.wav
go run ./scripts/voice_v2/ws_probe -scenario bargein -speech .tmp/voice_v2/a.wav -speech2 .tmp/voice_v2/b.wav
```

## 3. 三场景实证

测试语音由 breeze 合成（char_gaea 音色）→ 线性重采样 16k 当"用户说话"。
a.wav=「你好，请用两句话介绍一下你自己」(2.64s，句中有逗号停顿)，
b.wav=「等等，先别说了，我想问明天天气怎么样」(4.48s)。

| 场景 | 断言 | 结果 | 关键证据 |
|---|---|---|---|
| turn | 8/8 | **全 PASS** | 转写全句（逗号停顿 reopen 后合并）✓；response.done=completed ✓；8.4s 连续下行音频 |
| cancel | 8/8 | **全 PASS** | 首段音频到达即发 response.cancel → `cancelled/client_cancelled`；取消后新增音频 0B |
| bargein | 10/10 | **全 PASS** | AI 播报 1.7s 时插话 → `cancelled/turn_detected`；**旧代际泄漏 0B**；新轮 completed，第二轮转写完整 |

服务端权威延迟（herdsman 35B，仅管线验证、非最终延迟口径）：

```
Turn latency: stt=0.16s llm=2.87s tts_ttfa=0.36s e2e=5.79s vad_decision=0.10s
              hold=0.00s smart_turn_status=complete
```

- **speech_end → first audio 客户端口径 323~376ms**（breeze SSE 首段，达标线 ≤2.5s 的零头）。
- llm=2.87s 是 35B prefill 拖累（已知 Vulkan prefill 瓶颈，~13 tok/s），换 4B~8B 快模型是 M4 的事。
- Smart Turn 实测：`complete p=0.855~0.989, 26~61ms`（CPU ONNX），逗号停顿不切句靠它+reopen grace。

## 4. Windows 硬门禁（§0.3 逐项）

| 门禁 | 实测 | 判定 |
|---|---|---|
| Windows 11 安装成功 | py3.13 + venv 一次装通（torch 2.15.0+cpu） | ✅ |
| 不装 CUDA 也能启动 | `cuda_available=False`，全程 CPU 推理 | ✅ |
| 启动后空闲内存 | WS 1.77~1.91 GB / Private 3.40~3.66 GB（重，见 §6） | ✅（偏重） |
| sidecar 停止/重启干净 | taskkill 后零 python 残留、仅客户端 TIME_WAIT；热重启 15s 就绪即恢复服务 | ✅ |
| Wails 退出后子进程残留 | python 无子进程（LLM/TTS 走 HTTP 到既有服务），单进程单杀点 | ✅ |
| 端口冲突与随机端口 | 8791 显式配置，可换；WS 路径 `/v1/realtime` | ✅ |
| 安装包增量 | venv 2.0 GB + 模型缓存 ~1.0 GB（paraformer 990MB + silero/SmartTurn ~15MB） | ⚠️ 3GB 级，见 §6 |
| 断网仍能完全工作 | 全链 loopback（8080/8012/8791）+ 模型缓存命中；运行期零活动外联（仅模型下载期 CLOSE_WAIT 残留） | ✅（论证+连接实证，未拔线实测） |

冷启动（含 990MB 模型下载）110s；**热启动 15s 到 uvicorn ready**（模型已缓存）。

## 5. 坑（复现必读）

1. **nltk punkt_tab 缺失** → LLM 生成流 `LookupError`、response failed（serve 路径不跑
   s2s_pipeline 的 nltk 自举）。且 nltk 下载器在本机代理 fake-IP 下不通：用 curl 拉
   `raw.githubusercontent.com/nltk/nltk_data/gh-pages/packages/tokenizers/punkt_tab.zip`
   手动解压到 `%APPDATA%\nltk_data\tokenizers\`。
2. **cmd.exe 批处理必须 ASCII+CRLF**：.bat 里写 UTF-8 中文注释会被按 GBK 解码撕碎命令行
   （`'ea' 不是内部或外部命令` 家族报错）。
3. breeze TTS 侧 **voice 必须是已注册音色 id**（本机实例为 char_gaea 等；未知名 404）。
   `pcm + stream_format=audio + stream=true` 兼容（audio/pcm 流，无 WAV 头）。
4. gorilla/websocket 禁止并发 WriteMessage：探针音频流与 response.cancel 并发发送需互斥锁。
5. wire 层 `input_audio_buffer.speech_stopped` 是**轮次提交期**事件，不是 VAD 内部软结束时刻；
   客户端拿到的 vad_end ≈ 提交延迟（含 reopen grace + endpoint hold），真实分段延迟以服务端
   `Turn latency` 日志为准（M2 协议固化时考虑暴露 `session.latency_trace` 扩展事件）。
6. Go `printf` 写 .bat 时 `\v` 是垂直制表符转义——用 python 写文件。

## 6. 对「接入还是提取」的初步数据（调研文档 Step 3）

- **接入（保留 sidecar）成本**：venv 2.0GB + 模型 ~1GB + 常驻内存 1.8GB WS + 热启动 15s。
  Windows 打包需捆 Python 运行时（onnx/torch CPU 轮子），安装器增量 3GB 级。
- **已白得的能力**（自研要重写）：Smart Turn EOT、soft-end/reopen 轮次状态机、CancelScope
  代际取消、Realtime 协议路由（WS+WebRTC）、工具调用链、逐阶段延迟日志、浏览器 demo 参考。
- 结论倾向：**协议照抄、控制面按 M2 固化成 gaea 自有 `voicecore`**；sidecar 在开发期继续当
  参照实现与回放对照。是否随产品分发 sidecar，挂 M2 结束时拍板（打包体积是主要反对票）。

## 7. 遗留与下一步

- **M2（下一刀）**：Realtime 协议类型固化（TS/Go 双侧）+ 无模型 mock runtime 回放测试 +
  前端 AudioWorklet 采集（20ms/320 sample）+ PCM ring buffer 播放；探针升级为回放断言器。
- **M3 收窄**：只做 Smart Turn 中文 A/B（vs FireRed Chinese EOT）+ 参数区间调优（社区初值在档）。
- **M4**：语音专属快模型路由（`voice_chat_engine/model` 独立于 35B 文本路由）——本轮 35B 只验管线。
- STT 候选对照（paraformer-zh vs sherpa streaming 14M）延迟/WER 实测挂 M3/M4。
- 探针增强池：`response.output_audio_transcript.delta` 文本一致性断言、多轮连续对话、
  `conversation.item.create` 注入、工具调用路径。
- 当前 sidecar 实例：PID 见运行时（8791），`taskkill /PID <pid> /F` 停止。

## 8. 本轮产物清单

- 仓库新增：`scripts/voice_v2/gaea-spike.yaml`、`scripts/voice_v2/start_spike.bat`、
  `scripts/voice_v2/ws_probe/main.go`（gofmt/vet 干净）。
- 机器侧：`C:\AI\gaea-voice-v2\venv`、`%APPDATA%\nltk_data`、modelscope 缓存
  （`~\.cache\modelscope`，paraformer-zh 990MB）。
- 未动 V1 任何代码；未改聊天数据库/文本路由/语音设置页（§0.5 首个 PR 边界遵守）。
