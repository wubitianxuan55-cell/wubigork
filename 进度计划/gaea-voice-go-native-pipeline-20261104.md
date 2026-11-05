# 语音管线 Go 原生重写（sidecar 退役）主计划

- 状态：**进行中**（刀1~5 已提交；刀6 代码面 2026-11-05 落地未提交：sidecar/4B 拉起链退役〔工作树前批〕+启动连接卡死根修+**8792 网关退役**〔注入逻辑吸收进进程内 LLM 腿 voiceNativeLLMLeg，路由恒走聊天板块功能绑定，voice_chat_llm radio/8792 服务/qwen35 语音直代全退役；R3 零输出 status 瑕疵+6400B 尾批 flush 销项〕；余=发版；**bundle v2 已落地**（2026-11-05：模型路径可重定位 voiceNativeModelPath（BASE 优先/统一根兜底）+package_voice_bundle.py 重写（exe+三 DLL+models/voice+装机脚本+SHA256 清单，venv/4B/launcher 出局）+bundle 布局冷启实弹过：装配 3.49s+wire turn 全事件+兜底 delta 在线）
- 拍板：2026-11-04 用户拍板「sidecar 用 Go 重写，植入 gaea 语音系统，不要再搞这种了，严重影响使用」+「LLM 腿=刚刚优化的 llama 引擎里的 35B 模型，与聊天板块的功能绑定挂钩」+「直接动刀，不做过渡止血」
- 本文件为唯一事实源；语音 V2 前史见 `gaea-voice-v2-optimization-roadmap-20261103.md`

## 0. 背景与病根（2026-11-04 15:10 实测）

语音启动链 70.1s，其中 Python sidecar（huggingface/speech-to-speech @8791）占 67s：~48s 纯 Python import（torch/funasr/modelscope 全家桶，552MB torch）+ ~10s funasr/HF 每次启动联网检查 + 模型加载/warmup。`VoiceV2Stop` 杀进程导致每次重进全量重付（当日冷启 5 次）。运行态健康（单轮 e2e 1.97s、STT rtf 0.07~0.2、Smart Turn 22ms）——问题纯粹在冷启动。

## 1. 拍板方案

- **管线 Go 原生重写、进程内**：silero VAD → smart-turn EOT → paraformer STT → LLM 腿 → breeze TTS 腿
- **前端零改动**：8791 说标准 OpenAI Realtime 协议（契约 `frontend/src/voice/protocol/events.ts`）
- **LLM 腿（用户拍板）**：挂聊天板块功能绑定 → 自管 llama 35B（8124，直代整轮 0.79s 实测）；8792 网关注入逻辑吸收进进程内（system 设定+独立 user 上下文消息保 prefix cache）；8792 网关、语音专属 4B 腿（8123）、voice_chat_llm 三值 radio 全部退役
- **退役面**：venv（676MB+ 依赖）、sidecar 拉起线、monkeypatch（X-Gaea 头进程内直调消亡）
- 测试资产复用：`ws_probe`（turn/bargein/cancel 断言）、`eot_analyze.py`（91% 门）、`false_interrupt_bound.py`（≤3% 门）、`stt_compare.py`（A/B 语料）——保留 `Turn turn_N rev=R latency:` 日志格式无缝对接

## 2. 刀1 已落地（2026-11-04，本次提交）

### 2.1 交付物

- `internal/voice/native/`：
  - `logmel.go` — Whisper log-mel 纯 Go 移植（mel_filters.json 内嵌 80×201）
  - `native.go` / `engine_windows.go`（`//go:build windows && amd64 && cgo`）/ `engine_other.go`（`!windows || !amd64 || !cgo` stub）
  - `dump_test.go`（env 门控 DSP 调试探针）
  - testdata：两条 breeze 合成 utterance（16k wav）+ 金样特征 + golden.json
- 依赖：`k2-fsa/sherpa-onnx-go-windows/v2 v2.0.1`（STT/VAD 冒烟）、`yalue/onnxruntime_go v1.28.0`（EOT）、`gonum v0.17.0`（FFT）
- `build/voice-dlls/`：onnxruntime.dll（1.31.1，取自 gaea-voice-v2 venv capi）+ sherpa-onnx-c-api/cxx（2.0.1）——锁定已验证字节入仓
- `build.bat`：CGO_ENABLED=1 + CC=ucrt64 gcc + DLL 随 exe 派发

### 2.2 裸测结论（全绿）

| 门 | 结果 |
|---|---|
| EOT 金样 | p 与 venv（transformers5+torch STFT+ORT）逐位对齐：complete p=0.9740077 vs 0.9740077257，incomplete p=0.21420252 vs 0.21420252；推理 24~48ms（sidecar ~20ms 量级） |
| logmel DSP | 金样对拍 max diff < 2e-3 容差（实际 ~1e-4） |
| STT paraformer int8 | 「你好我想听一个很短的故事」逐字命中；去空格语义与 sidecar 一致 |
| VAD silero v5 | 两条 utterance 各切 1 段（WindowSize=512） |
| 构建 | `go build ./...` 无 cgo（CI 无 gcc 场景）与 cgo（CC=ucrt64）双模式全绿 |

### 2.3 踩坑实录（防复发）

1. **System32 全局 DLL 劫持**：本机 `C:\Windows\System32\onnxruntime.dll`（1.17.1，外来软件所丢）在搜索序中先于 PATH——任何不随 exe 派发 DLL 的进程都会拿到 1.17.1。铁律：**onnxruntime.dll 必须与 gaea.exe 同目录**（exe 目录是搜索序第一）。
2. **sherpa-onnx-go-windows v2.0.1 打包失配**：构建头要 ORT API 28，lib 目录自带的 onnxruntime.dll 却是 1.17.1（API 17）→ GetApi 空指针 0xc0000005。解法=统一用一只新版 ORT（1.31.1）：GetApi 向后兼容，yalue 要 28、sherpa 要 ≤17 双双满足。
3. **ORT 双消费者共享同一模块**：Windows 按模块名去重，先加载者赢——DLL 派发位置决定一切，路径参数救不了已加载的旧模块。
4. **silero v4/v5 语义**：sherpa release 的 643KB `silero_vad.onnx` 配 WindowSize=1536 建 VAD 会话 C++ throw；**v5（2.3MB，asr-models release 的 silero_vad_v5.onnx）+ WindowSize=512 才对**——且 sidecar 生产本来就用 torch.hub master=v5，语义一致。
5. **金样 fixture 纪律**：金样特征必须从**量化后**的 wav 样本计算（resample 过冲 >1.0 时 int16 截断翻转、非四舍五入都是坑）；生成脚本路径用 `REPO = dirname/../..`（相对路径写错过两次）。
6. **cgo 工具链**：MSYS2 ucrt64 gcc 16.1 可用但必须把 `/c/msys64/ucrt64/bin` 加入 PATH（其依赖 DLL 才能解析）；`CGO_ENABLED=1 + CC=gcc`。
7. **测试运行布局**：sherpa DLL 链接期解析须在进程启动前可达 → `go test -c -o .tmp/native.test/` + DLL 拷同目录 + 从包目录跑二进制（cwd 影响相对路径的 testdata，不影响 exe 目录的 DLL 解析）。

### 2.4 生产布局定案

- gaea.exe 同目录：onnxruntime.dll + sherpa-onnx-c-api.dll（+cxx）
- `C:\AI\models\voice\`：model.int8.onnx（paraformer 214MB）+ tokens.txt + smart-turn-v3.2-cpu.onnx（8.7MB，HF 缓存拷入）+ silero_vad_v5.onnx（2.3MB）
- ORT 版本配对：onnxruntime.dll 1.31.1 ↔ yalue v1.28.0（API 28）↔ sherpa 2.0.1（API ≤28）；升任一方须三对齐复验

## 2.5 刀3 已落地（2026-11-04）

- `internal/voice/realtime`：turns.go（SpeculativeTurnTracker 全语义：grace 仅延长/候选 HOLD 2s/音频时钟 7000ms/提交 vs 关闭——9 单测固化）+ vad.go（silero 状态机：滞回带 threshold-0.15/样本钟 64ms 静音/pre-speech pad 500ms/500ms 空闲模型复位）+ vadhandler.go（延迟确认 384/192+候选先行/幻影/discard/soft-end+Smart Turn 宽限表+前缀拼接；日志行逐字保 sidecar 格式）+ pipeline.go（输入项生命周期+持发规则+STT 门+item 复用重开+50ms 持发重试泵）+ native SileroVAD 直跑（ORT）
- **实弹验证（breeze 合成句回放）**：384ms fragment → soft-end → Smart Turn complete → 续说 192ms 判据 → **同轮 rev=1 重开**（item 复用，仅一次 speech_started）→ 12 字全句转写（前缀拼接生效）——speculative reopen 全链实证
- **语义保真要点**：转写完不提前 commit+close（会杀死 reopen）；terminal 持发至「响应提交/下一轮开始/会话结束」；无响应腿（刀5 前）下 stopped/committed/transcription 持而不发=正确行为
- **坑新增**：①silero LSTM 态须 stateN→state 回拷（漏拷=概率恒近随机）；②yalue 不支持 0 维标量（sr 输入）→ 图手术常量化（fix_silero_sr.py 产 silero_vad_v5_16k.onnx，等价性验证过）；③OnAudio 512 块化须跨帧累积余数（前端 320 样本帧粒度，否则全丢）；④app 包测试二进制链接 sherpa DLL 后，go test 须 PATH 带 DLL 目录
- 余：ws_probe turn/bargein/cancel 三场景门挂刀5（响应腿是前置）

## 2.6 刀5 已落地（2026-11-04，代码面；实弹门挂下步）

- `realtime/response.go`：response 装配（created 顺序守卫/assistant item 链/6400B 音频批/finishResponse{usage+turn_latency metadata}/tracker close/128 墓碑/chat 历史 12 窗）+ ResponseLegs 接口（Reply 带 turn/rev/itemID 供关联头）
- `realtime/latency.go`：延迟日志行逐字（stt/llm/tts_ttfa/e2e/vad_decision/hold/smart_turn_status/status[ tts=cut]/source=speech/response_key=...）+ spokenSplit 口语分句（CJK 句读+西文句读边界守卫+60 rune 强切）
- `internal/app/voice_v2_go_legs.go`：LLM 腿=直连现役 8792 网关 SSE（persona 注入/prepared 台账/功能绑定路由**零漂移复用**，X-Gaea 三头照旧骑行；用户拍板「跟随聊天绑定→35B」由 voice_chat_llm=chat 配置兑现）+ TTS 腿=breeze 8012 流式 PCM（16k 直出/24k 线性重采样跨块 carry）+ voiceResponseLegs 组合
- pipeline.generateResponse：转写非空 → chat user → beginResponse → created → LLM 流（增量开输出+transcript.delta+分句下发）→ 逐句 speakSentence（gate commit=true 首音频提交轮+TTFA/e2e 记账）→ finishResponse(completed)；LLM 失败兜底语；barge-in=cancel ctx+finishResponse(cancelled/turn_detected)
- GAEA_VOICE_REALTIME_GO 模式接线组合腿（网关 8792 启动期已在跑，依赖零新增）
- **余（下会话实弹）**：ws_probe turn/bargein/cancel 三场景打 Go 版（需 gaea 壳+网关+breeze+35B 在跑）；真机全链路+e2e 记分卡（门 ≤2.5s，对照 0.79s）；前端联通目检

## 2.7 管线架构重写为源码同构拓扑（2026-11-04 晚，用户指令"不要手搓，看复刻源码"后）



- **推倒重来**：pipeline.go 按 s2s_pipeline.py 队列图+websocket_router._send_loop_for 重写——
  audioCh/sttCh/promptCh/ttsCh/audioOut/textCh 六队列 + VAD/STT/LLM/TTS 四阶段线程 + 信号驱动发送循环；
  协议翻译（resolve terminals/barge-in/response 装配）只在发送循环单线程（service.dispatch_pipeline_event 同位）
- **代际取消（用户问"上游如何解决"后照搬）**：cancelScope（cancel_scope.py 同位：gen 计数+discarding 窗口）
  + audioOut 带代际、发送循环 discardable 闸（router:1134 同位，含 done 哨兵）+ finishResponse key-vs-current
  守卫（response.py:991 同位）——撤掉手搓 finished 标志
- **ws_probe 三场景全绿（同一二进制）**：turn 8/8（e2e 304~529ms）、bargein 9/9（cancelled/turn_detected+
  零泄漏+新响应 completed+两轮转写）、cancel 8/8（client_cancelled+零后置音频）
- **自测基建定型**：CDP 直调 `window.go.app.VoiceB.VoiceV2Ensure` 免点击 + ws_probe + Start-Process 单实例
  （bash 后台 & 会双开竞态）；测试壳日志在 %TEMP%\gaea-voice-v2 与 gaea 日志双处
- **教训固化**：①上游答疑优先读源码原文（cancel_scope.py 的 generation 设计一句话顶半天调试）
  ②每改一处→build+单测+ws_probe 单场景，一次一刀 ③批量字符串补丁是错漏之源，改用整文件重写

## 2.8 真麦「无法识别」根修（2026-11-05 晨，v4.532.0 实弹翻车；待用户真麦复测后提交）

- **现象**：真麦说话四段全 `discarding segment=660~820ms active=64~192ms`，STT 零触发
- **真凶**：silero v5 ONNX 输入契约 **(1,576)=64 context+512 chunk**；Go 按 (1,512) 直喂=context 恒错位——强电平不致命（合成语料实弹全绿掩盖），弱电平（用户麦 RMS≈0.01）判性崩（mean 0.10 vs torch 0.72）。「昨天还行」=sidecar torch JIT 内部自维护 context
- **修**：SileroVAD 自维护 ctx64（engine_windows.go）；vadcheck 探针定位（scripts/voice_v2/vadcheck）；修复后 weak mean=0.708=与 torch 同档，ws_probe 弱料全链逐字 PASS+cancel/bargein 零泄漏
- **随刀**：播放期回声门（监听 0.5/TTS 期 0.75，EchoGuardOn/Off）+captureProcessor 自适应增益（弱麦 ×4 上限，peak>0.22 回 1）+前端/探针 0.5 同步+探针零响应 panic 修+discard 行临时 rms 字段（验后撤）
- **教训**：等价性验证输入须在正确契约下取且含弱电平真实语料（fix_silero_sr 的 EQUIVALENT 在 (1,512) 错误形状下两边一起错）；假麦/合成语料绿≠真机可用，发版门须真麦烟测

## 3. 刀2~6 计划

- **刀2 WS 服务器+协议层**：8791 `/v1/realtime`（gorilla/websocket），session pool=1（第二条 `session_limit_reached`+close 1008）、session.created/updated、events.ts 全事件、response.created 顺序守卫、response_key 墓碑（≤128）、下行 6400B 分块；门=前端可连 + orderingViolations() 干净
- **刀3 VAD+EOT+轮次机**：ORT 直跑 silero v5 原始概率（不用 sherpa 段式 VAD）+ 移植 vad_iterator 迟滞状态机（min_speech 384/192ms、min_silence 64ms、speech_pad 500ms、延迟 SpeechStarted、前端 threshold=0.75 接 session.update）+ smart-turn 判决宽限（complete 800ms / incomplete 1200ms+STT delay 400ms / 失败 800ms）+ SpeculativeTurnTracker（turn_{n}/rev、LISTENING→SOFT_ENDED→ANSWERING→CLOSED、续语音同轮 rev+1 重开、未答上限 7s、HOLD ≤2s、陈旧丢弃、跨 revision 前缀拼接）+ STT 输入门；门=ws_probe 三场景全绿 + eot_analyze 对齐 91% 基线
- **刀4 STT**：paraformer 全句转写 + 前缀拼接 + transcription.delta/completed；门=stt_compare 语料 A/B 对齐 funasr 基线
- **刀5 LLM+TTS 腿+response 生命周期**：功能绑定→35B SSE（persona 注入进程内组装，prepareVoiceV2Turn 调用点从网关迁入管线）+ 分句批切（复用 splitSentences 模式）+ breeze 流式 PCM（优先 breeze 直出 16k 免 FIR）+ barge-in 取消 + turn_latency 元数据；门=整轮 e2e ≤2.5s（35B 直代基线 0.79s）、barge-in 无陈旧音频
- **刀6 收口+退役**：VoiceV2Ensure 简化（breeze→[35B ensure]→管线 lazy init，去 sidecar/4B/网关三段）、VoiceV2Stop 语义改、voice_chat_llm 设置收敛、bundle v2（venv 出局）、voice_v2_gateway.go 退役、scripts/voice_v2 手动回滚物保留一版期、发版

## 4. 全局验收门（刀6 后）

真机：冷启按钮→就绪 ≤5s、完整对话、打断、P2 预览确认链、控制台关闭 commit/abort 时序（roadmap:94 铁律）、人格记忆注入可见、功能绑定换绑语音跟随；离线工具：eot_analyze 91% 门、false_interrupt_bound ≤3%、stt_compare A/B；记分卡 35B 直代整轮 0.79s 对照门 ≤2.5s。
