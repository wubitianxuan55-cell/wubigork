# 08 · 模型与媒体引擎层评审报告（gaea 3.0 Step 3 Provider Seam 前置调研）

> 日期：2026-08-15 ｜ 调研人：架构评审子代理 ｜ 范围：只读调研
> 目标：为 gaea 3.0 Step 3「Provider Seam（定义/提供者/消费者三元组）」提供证据底稿。
> 覆盖包：internal/ai、internal/modelengine、internal/voice、internal/tts、internal/asr、
> internal/herdsman、internal/netclient、internal/auth、internal/channels/weixin、internal/config；
> 另含与 seam 直接相关的 internal/gaea/provider（已 seam 化的对照样本）与 internal/app 各 handler。

---

## 1. 概览（各包文件清单+职责）

### 1.1 internal/ai（17 个 .go，5080 行）—— 模型调用的统一门面
| 文件 | 行数 | 职责 |
|---|---|---|
| client.go | 1235 | *Client：OpenAI 兼容 chat/stream/images HTTP 客户端；引擎路由（engineMgr）+ token 管理 + 重试/SSE 解析 + 用量上报 |
| interface.go | 20 | LLMClient 接口（ChatStream/ChatSimpleStream/GenerateImage），办公引擎 bridge 依赖它 |
| types.go | 215 | ChatRequest/ChatMessage/ChatToolCall/SSEChunk/ImageGenerationRequest 等全部类型 |
| image_backend.go | 9 | ImageBackend 接口（生图 seam 的「定义」已存在） |
| image_openai.go | 164 | OpenAIImageBackend：通用 /v1/images/generations 后端（herdsman/ollama） |
| image_comfyui.go | 857 | ComfyUIBackend：ComfyUI REST 工作流（txt2img/img2img/t2v/LoRA/中断） |
| copilot.go | 368 | GhostComplete/CmdKEdit/OfficeEditText/XlsxEditOps/Beats/Prose 等写作辅助 |
| 其余 | — | 测试（client/copilot/image/stats/toolcall/regression） |

### 1.2 internal/modelengine（5 个 .go，2390 行）—— 引擎注册表 + 统计
| 文件 | 行数 | 职责 |
|---|---|---|
| engine.go | 683 | EngineType 枚举、EngineConfig/EngineStatus、Manager（7 引擎预置、模型清单刷新、engines.json 持久化、BuildChatURL 凭据解析） |
| stats.go | 529 | ModelCallUsage 上报、按引擎/模型累计统计、内置定价表、汇率折算、落盘节流 |
| 其余 | — | engine_test（589）/ engine_state_test / stats_test |

### 1.3 internal/voice（11 个 .go，2556 行）—— 语音管道状态机
voice_config.go（174）：VoiceMode/InputChannel/TTSEngine/ASRModel 枚举 + VoiceRuntimeConfig + 音频常量；voice_manager.go（861）：idle→listening→thinking→speaking 状态机、VAD/打断/PTT；tts_model_resolve.go（100）：Herdsman TTS 模型按已装列表动态解析；emotion_voice_map.go（170）：情绪→TTS 语音参数映射；wav.go（42）。

### 1.4 internal/tts（9 个 .go，1532 行）—— TTS 合成器集合
edge.go（368）：Edge TTS（WebSocket，含 Synthesizer 接口与 SynthesizerChain 回退链）；herdsman.go（376）：Herdsman /v1/audio/speech（customvoice/voicedesign/voiceclone 三构造器 + 音色解析）；sapi.go（77）：Windows SAPI；xai.go（121）：xAI Grok TTS /v1/tts；util.go（41）：SplitSentences。

### 1.5 internal/asr（2 个 .go，385 行）—— 语音识别客户端
herdsman_asr.go（164）：HerdsmanASR（/v1/audio/transcriptions 非流式 + TranscribeBytes）；herdsman_asr_stream.go（221）：WebSocket 流式识别。**只有 herdsman 一个实现**。

### 1.6 internal/herdsman（6 个 .go，1663 行）—— 本地模型服务探测
health.go（248）：一次性健康检查（端口/API/能力归类）；probe.go（324）：H0-1 环境与数据契约探测；lancheck.go（176）：H0-4 LAN 暴露检测与告警。

### 1.7 internal/netclient（6 个 .go，1194 行）—— 共享 HTTP/代理基础设施
netclient.go（298）：NewSimpleClient/NewHTTPClient/代理模式（auto/env/custom/off）；sysproxy/（Windows 系统代理读取）。

### 1.8 internal/auth（7 个 .go，1370 行）—— SuperGrok OAuth
oauth.go（368）：PKCE loopback 登录 + token 交换/刷新；token.go（195）：Token + TokenStore（DPAPI 加密落盘）；discovery.go（59）：OIDC Discovery；browser.go（22）。

### 1.9 internal/channels/weixin（3 个 .go，614 行）—— 微信助手 beta
clawbot.go（388）：iLink Bot 长轮询 Server（多助手实例）；qrlogin.go（119）：扫码登录。

### 1.10 internal/config（3 个 .go，1783 行）—— 应用配置层
config.go（1121）：Key 常量全集 + configFile(JSON) + Config + Load/Save。

### 1.11 与 seam 直接相关的对照/装配面（非本层但必读）
- internal/gaea/provider/provider.go（390）：**已实现的 Provider Seam**（接口/Factory/Register/New）——办公引擎的范例。
- internal/gaea/provider/bridge/bridge.go（218）：kind="wubigrok" 的 bridge provider，把办公引擎接到 ai.LLMClient。
- internal/gaea/config/config.go：办公引擎 TOML 配置（providers 段）——用户侧 TOML 的现状。
- internal/app 各 handler：tts_handler.go / voice_handler.go / voice_model_handler.go / model_engine_handler.go / image_handler.go / writing_state.go / characterlib_gen_handler.go / gaea_ocr.go / herdsman_lifecycle.go —— provider 分支的实际落点。

---

## 2. ai 层调用面与 provider 分支

### 2.1 调用面：一个接口 + 一个巨型 Client
- 接口面极小：internal/ai/interface.go:8-19 定义 `LLMClient`（ChatStream/ChatSimpleStream/ChatSimpleStreamWithOptions/GenerateImage）。实现者是 `*Client`（internal/ai/client.go:25-58）。
- *Client 是「一切模型 I/O 的上帝对象」：同时持有 cfg/tokenStore/engineMgr/imageBackend/信号量/重试参数（client.go:30-57），并直接依赖 modelengine、auth、netclient 三个包（client.go:17-22）。
- 请求类型全部 OpenAI 兼容：ChatRequest（types.go:52-88）、SSEChunk（types.go:161-172）、ImageGenerationRequest（types.go:190-220）。

### 2.2 引擎路由（chat）：两分支的「伪 seam」
- client.go:180-196 `resolveChatEndpoint`：
  - 分支 1：engineID != "xai" 且有 engineMgr → `c.engineMgr.BuildChatURL(engineID)`（client.go:185-187）；
  - 分支 2：xAI → OAuth token + cfg.XaiAPIBaseURL/chat/completions（client.go:189-195）。
- client.go:202-232 `resolveModelName`：按 engineID=="xai" 分支决定默认模型来源（xAI→cfg.Model；其他→engineMgr.GetDefaultModel）。
- 401 刷新 token 只对 "xai" 引擎生效：非流式 client.go:413、流式 client.go:591（`reqEngine == "xai"`）。其他引擎 401 直接返回错误。
- 引擎「切换」接口：SetActiveEngine/ActiveEngineID（client.go:156-176），空串默认 "xai"（client.go:159/173）。
- 功能级覆盖：ChatRequest.EngineID 字段（types.go:54）非序列化透传，ChatSimpleOptions.EngineID 同理（types.go:115）。

### 2.3 引擎特性分支（硬编码在调用方）
- client.go:1036-1046：`reqEngine == "herdsman" || reqEngine == "ollama"` 时设置 EnableThinking + ChatTemplateKwargs。同样的分支重复出现在 gaea bridge（见 §5.6）。
- copilot.go 的 GhostComplete（20-96）/CmdKEdit（105-145）/OfficeEditText（150-186）/XlsxEditOps（190-223）/GenerateBeats（235-277）/GenerateProseFromBeat（282-367）全部经 ChatSimpleStream/WithOptions 或 ChatStream 走统一通道，**不感知引擎**——这是理想形态。

### 2.4 toolcall（工具调用）通道
- 类型：ChatToolCall/ChatToolFunction（types.go:18-34）、ChatToolSchema（types.go:40-50）、流式增量 ChatToolCallDelta（types.go:53-70）。
- 流式拼装：client.go:653-790 parseStreamEvents 按 index 拼装 tool_calls 增量，finish_reason=="tool_calls" 时输出完整 ToolCalls（client.go:771-777）。
- 消费方：gaea bridge 把 ai 的 tool call 转为 provider.ToolCall（bridge.go:88-99），办公引擎（internal/gaea agent + tool.Registry）执行；前端聊天板块直接消费 SSEChunk.ToolCalls。ai 层本身不执行工具。

### 2.5 生图接口：ImageBackend seam 已存在，但注册/选择是散的
- 定义：internal/ai/image_backend.go:5-9 `ImageBackend{ GenerateImage }`——seam 的「定义」已具备。
- 提供者三个：*ComfyUIBackend（image_comfyui.go:24-44，含 Interrupt/ResetCancel 等取消扩展）、*OpenAIImageBackend（image_openai.go:12-33，通用 /v1/images/generations，herdsman/ollama 复用）、内置 xAI 原生生成 generateImageXAI（client.go:1140-1206，私有方法非接口实现）。
- 消费点：client.go:1129-1137 `GenerateImage`：imageBackend != nil → 委托，否则回退 xAI。**选择逻辑不在 ai 层**，而在 app 层四处 switch（见 §7 清单 5）。
- GetImageBackend 返回裸接口供类型断言（client.go:1081-1089，调用方靠 `*ComfyUIBackend` 断言拿 Interrupt——注释自认这是 seam 缺陷）。

### 2.6 统计上报
- 每次调用经 recordUsage（client.go:893-909）或流式 defer（client.go:659-680）上报 modelengine.ModelCallUsage——统一通道，与引擎无关（好）。

---

## 3. modelengine 注册与生命周期

### 3.1 引擎枚举（编译期硬编码 7 个）
- engine.go:19-33：EngineType = xai | ollama | herdsman | deepseek | cosyvoice | opencode-go | opencode-zen。
- 预置注册在 NewManager：engine.go:98-191，逐个硬编码 EngineConfig（ID/Name/Type/Label/Color/Icon/IsLocal/BaseURL/Enabled/DefaultModel）；order 固定（engine.go:102）。**新增引擎必须改 NewManager + order + 全部 Type switch**。

### 3.2 数据结构
- EngineConfig（engine.go:45-60）：ID/Name/Type/Label/Color/Icon/IsLocal/BaseURL/APIKey/Enabled/DefaultModel/Models/Status。
- EngineStatus（engine.go:62-70）：Connected/ModelCount/Error/LastChecked/LatencyMs——最近连接状态缓存。
- Manager（engine.go:84-96）：engines map + order + statePath + 四个凭据字段（xaiKey/deepseekKey/opencodeKey/opencodeZenKey）+ httpClient + statsRecorder。

### 3.3 凭据分支（Manager 内部 switch）
- fetchModels（engine.go:383-474）：按 `engine.Type == EngineXAI/Deepseek/OpencodeGo/OpencodeZen` 四个 if-else 决定 Authorization 头（engine.go:392-401）；401 文案按类型分支（410-419）；OpenCode 目录按白名单函数过滤（engine.go:434-439，opencodeGoCompatible 513-519 / opencodeZenCompatible 525-533——**硬编码模型前缀**）；xAI 补 grok-tts 内置模型（459-471）。
- BuildChatURL（engine.go:655-682）：`switch engine.Type` 四分支取 apiKey（671-679），URL = BaseURL + /chat/completions（669）。这是 ai 层之外唯一的 chat 端点解析。

### 3.4 模型清单刷新机制
- TestConnection（engine.go:305-348）：fetchModels → 更新 Models + Status → saveState；失败也缓存失败状态（325-331）。
- RefreshModels（engine.go:350-379）：拉取并更新模型列表 + 状态。
- fetchModels 对非 200 直接错误；对本地引擎（ollama/herdsman/cosyvoice）不加 Authorization。
- 模型分类 classifyModelKind（engine.go:478-509）：按模型名关键词启发式分类 llm/tts/stt/ocr/rerank/embedding/image——**纯字符串匹配，与语音/OCR 侧的三处重复实现（见 §7 清单 7）**。

### 3.5 engine_state / 持久化
- 状态文件 engines.json：LoadState（engine.go:585-626）合并预置默认 + 磁盘覆盖（BaseURL/Enabled/DefaultModel/Models/Status）；saveState（engine.go:630-653）原子写、APIKey 清空不落盘。
- 装配：internal/app/app.go:283-289 —— NewManager("", deepseekKey) + UpdateOpencode* + LoadState(whisperDataRoot/engines.json) + SetStatsPath；app.go:293-296 EnsureModel("xai","grok-tts") / EnsureModel("cosyvoice","CosyVoice2-0.5B")（内置伪模型）；app.go:297-299 恢复 xAI token 注入 UpdateXAIKey。
- 统计：stats.go ModelCallUsage（19-32）、statsRecorder（191-218）、定价表 modelPricing（94-120+，前缀匹配）、estimatePrice 对本地引擎短路（stats.go:151-163 `switch engineID { case "ollama","herdsman": }`）。

### 3.6 「引擎启停」不在 modelengine
- Manager 只管配置/清单/状态；本地模型的实际启动/停止/下载/卸载走 herdsman.exe skill CLI：internal/app/herdsman_lifecycle.go:66 runHerdsmanCLI、180 HerdsmanModelStart、194 HerdsmanModelStop、208 HerdsmanModelDownload、222 HerdsmanModelUninstall；exe 定位 internal/app/herdsman_catalog.go:146-176。

### 3.7 前端模型中心如何驱动
- ModelB 门面（internal/app/bindings_model.go，34 个绑定方法）→ core/model_engine_handler.go：
  - GetEngines/SaveEngine/TestEngineConnection/RefreshEngineModels（17-53）
  - SetEngineDefaultModel（55-74）：**仅 xAI 的默认模型同步到全局 cfg.Model**（65-70，防污染逻辑）
  - SetActiveEngine（77-111）：校验存在/启用 → client.SetActiveEngine → 持久化 active_engine_id → 同步全局模型 → emit model-changed
  - GetActiveEngine/GetActiveModel（114-132）
  - DeepSeek/OpenCode 三组 Key setter + 脱敏状态（137-220，DPAPI 加密后存 config）
  - GetModelCallStats/ResetModelCallStats（239-253）、汇率（260-288）
- 语音模型三段选择（STT/LLM/TTS）独立绑定：internal/app/voice_model_handler.go（见 §4.4）。
- 引擎配置持久化经 config.Save 写 ~/.gaea_config.json，而模型清单/状态写 engines.json——**两份持久化面**。

---

## 4. 语音/TTS/ASR 引擎选择

### 4.1 voice 域：状态机 + 回调注入（引擎选择外包）
- 枚举：VoiceMode（vad/ptt/off，voice_config.go:14-20）、InputChannel（dual/voice-only/text-only，21-27）、TTSEngine（auto/herdsman/edge-tts/local-sapi/qwen3-tts/herdsman-edge-tts，26-34）、ASRModel（whisper-base/sherpa-onnx/funasr，35-40）。
- VoiceRuntimeConfig（voice_config.go:62-98）+ DefaultVoiceConfig（72-105）。
- VoiceState 状态机 idle→listening→thinking→speaking（voice_config.go:120-133）。
- voice.Manager（voice_manager.go:52-88）**不做引擎选择**：持 `asrClient *asr.HerdsmanASR`（74）、`whisperChatFn`（77）、`ttsSynthFn`（78），全部由 App 层注入（SetASRClient/SetWhisperChatFn/SetTTSSynthesizeFn，107-120）。这是回调式依赖注入——seam 雏形，但注入的是**函数**而非**接口/提供者**。
- 生命周期：Start（163-186）/Stop（189-217）；音频入口 PushAudioChunk（222-260，speaking 态做打断检测 233-256）；VAD processVAD（263-326）；对话管线 handleReply（406-508）：whisperChatFn → emotion → GetVoiceDescriptionWithPersonality（481）→ speak 流式 TTS（484-491）；PTT SetPTTActive（707-741）。

### 4.2 TTS 引擎切换的真实落点：internal/app（不在 voice/tts 包）
- tts_handler.go:55-109 `TTSSpeakBase64` 四级回退：①用户选中 TTS 模型（57-67）→ ②扫描硬编码引擎序 `["herdsman","cosyvoice","ollama","xai","deepseek"]`（71-92，按模型名含 tts/voice/speech/voxcpm 判定）→ ③Edge TTS（95-99）→ ④WinTTS SAPI（102-106）。
- tts_handler.go:113-249 `TTSSpeakStreaming`：拼合成 `[]tts.Synthesizer` 列表——herdsman 四模型硬编码（129）、xAI 分支 `a.activeTTSEngine=="xai" && a.activeTTSModel=="grok-tts"`（146-159）、Edge（162-167）、SAPI（170-174）；失败后按 label switch 选活跃引擎（224-229）。
- Synthesizer 接口与回退链：internal/tts/edge.go:337-339（Synthesizer{Synthesize}）、341-367（SynthesizerChain，首个成功即返回）。
- voice 管道的 TTS：synthesizeVoiceTTS（voice_handler.go:175-193）——聊天语音绑定（chatVoiceEngine）→ 全局 TTS（activeTTSEngine/Model）→ TTSSpeakBase64 统一路由；tryEngineTTS（197-235）：cosyvoice 先 ensure 服务（206-208）→ xAI 分支 tryXaiTTS（209-211）→ 按模型名选 herdsman 构造器（voicedesign/voxcpm→WithDesc，voiceclone→WithClone，其余→With voice，213-226）。
- **注意**：voice.TTSEngine 枚举（voice_config.go:26-34）只在 VoiceApplySettings 里被写进 config.TTSEngine（voice_handler.go:402-404），合成路径（synthesizeVoiceTTS）读的却是 `activeTTSEngine`（模型中心选择），**TTSEngineAuto 等值实际未被消费**——枚举与真实路由脱节。

### 4.3 TTS 提供者（internal/tts）
- HerdsmanTTS（herdsman.go:19-27）：三构造器 NewHerdsmanTTS/WithDesc/WithClone（29-59）；buildBody 按 refAudio→voiceDescription→voice 三选一（131-148）；音色解析 resolveVoice + 服务端 supported_speakers 缓存（282-305、308-335）；defaultVoiceForModel 按模型名 switch（337-354）。
- EdgeTTS（edge.go:21-39）：WebSocket，voice 硬编码 zh-CN-XiaoxiaoNeural（37），EdgeVoices 列表（323-331）。
- XaiTTS（xai.go:26-64）：/v1/tts，token 由 `getToken func() (string, error)` 注入（复用 ai.Client.GetToken）；音色白名单 xaiVoices（8-17）+ IsXaiVoice。
- WinTTS（sapi.go:13-70）：PowerShell System.Speech 子进程。

### 4.4 模型中心对语音的驱动（三段模型独立绑定）
- SetActiveASRModel（voice_model_handler.go:22-64）：校验引擎+模型 → applyASRClient 重建 → 持久化 active_asr_engine/model。
- SetActiveTTSModel（77-103）：同构，写 active_tts_engine/model。
- SetChatVoiceModel（129-179）：功能绑定聊天语音（chatTts），空=回退全局。
- ttsVoiceForModel（190-207）：按模型名 switch 默认音色（edge→zh-CN-YunxiNeural 等）。
- GetVoicePipelineConfig（116-125）：stt/llm/tts/chatTts 四段。

### 4.5 ASR 引擎选择（伪多引擎）
- applyASRClient（voice_handler.go:128-163）三级：①用户选中（activeASREngine/Model，134-141）→ ②扫描 `["herdsman","ollama","xai","deepseek"]` 的 STT 模型（144-156，isSTTModel 按名匹配 166-171）→ ③默认 herdsman whisper-base（159-162）。
- 但 asr 包只有 `asr.NewHerdsmanASR`（asr/herdsman_asr.go:42-51）一个实现——所谓引擎选择只是换 baseURL+model，**全部落到 herdsman 协议**。
- 流式实现：herdsman_asr_stream.go（WebSocket /v1/audio/transcriptions/stream）。
- 浏览器端识别旁路：VoiceChatText（voice_handler.go:296-305）跳过后端 ASR，直进 HandleUserText。

### 4.6 Herdsman TTS 默认模型动态解析
- tts_model_resolve.go:15-100：ResolveHerdsmanTTSModel（configured vs installed 列表，优先级 herdsmanTTSPriority 21-26，voxcpm2 第一）；由 voiceSettingsMap 接入（voice_handler.go:445-461）。

---

## 5. herdsman / auth / weixin

### 5.1 herdsman：健康检查
- health.go:88-136 HealthCheck：①端口探测（net.DialTimeout 1s，98-105）②API 存活（GET {baseURL}/models，108-117）③按能力归类模型（120-131，ClassifyModelCapability 207-230 关键词匹配：rerank/bge/ocr/mineru/sherpa/tts/flux/mt/qwen 等）；Healthy = 端口+API+chat 模型（133）。
- 由 ModelB.HerdsmanHealth 暴露（bindings_model.go），能力键固定 10 个（health.go:51-54）。

### 5.2 herdsman：环境探测
- probe.go:34-62 Probe 结构（HomeDir/ConfigOK/CLI/APIReachable/DataFiles/Warnings）；NewProbe（103-124，默认 %USERPROFILE%.herdsman + localhost:8080/v1，HERDSMAN_PROBE_LIVE 控制是否真实 HTTP）；Run（139-147）→ probeConfig（151-188，复用 lancheck 的 YAML 逐行解析）+ probeCLI（192-208，HERDSMAN_EXE/PATH）+ probeDataFiles（212-235，launch_records/models/events.jsonl/skill-operations.json）+ probeAPI（299-323，非 2xx 告警契约漂移）。

### 5.3 herdsman：LAN 安全检查
- lancheck.go:43-130 CheckLanExposure：逐行 YAML 缩进解析 api 段（不引 YAML 依赖），提取 lan_accessible/port（102-117）；Exposed=true 时给中文处置指引（167-175：提示改 config.yaml，gaea 绝不代改）。probe 里同样给「lan_accessible=true」与「端口漂移」告警（probe.go:179-187）。

### 5.4 auth：SuperGrok OAuth PKCE
- 流程 DoLogin（oauth.go:59-193）：①OIDC Discovery（61-64，discovery.go:26-59，URL 可被 cfg.OIDCDiscoveryURL 覆盖）②newPKCE（66-70，32B verifier + S256 challenge，28-41）③state+nonce（72-83）④本机 loopback HTTP server（86-170，/callback 校验 state/error/code）⑤构建授权 URL 开浏览器（173-178）⑥等 token（180-192，5 分钟超时）。
- buildAuthURL（201-216）：client_id/scope(openid profile email offline_access grok-cli:access api:access)/S256/state/nonce/**plan=generic**/**referrer=wubigork**（211-212，注释明确：referrer 与品牌绑定，改会 500）。
- exchangeCodeForToken（223-273）：POST token_endpoint（code+verifier+challenge 双发，233-236）；403 → 明确「SuperGrok 订阅等级限制」文案（251-257）；安全校验 inference baseURL 仅允许 https://*.x.ai（269-270，324-353 validateInferenceBaseURL）。
- RefreshAccessToken（280-322）：OIDC Discovery 重新取端点 + refresh_token 交换。
- token 持久化：token.go Token 结构（23-43）、IsExpired 提前 1 小时刷新（45-60，对齐 hermes-agent）、TokenStore.Save/Load/Delete（62-134）：敏感字段经 secure（Windows DPAPI）加密，旧版明文自动迁移重写（Load 中 decryptToken 133-193）；legacy 路径 .wubigork_token.json 回退（NewTokenStore 69-76）。
- 消费方：ai.Client.GetToken（client.go:237-275，single-flight 刷新）与 tryRefreshToken（304-319）；grok-tts 经 XaiTTS getToken 复用。

### 5.5 weixin：ClawBot 桥接机制
- 独立 Server（clawbot.go:40-64）：Config{ILinkURL/BotToken/BotID/AssistantID/PersonalityID}（22-32）+ ChatFunc（36）+ OnSessionExpired 回调（56-57）。
- 生命周期 Start/Stop（81-119，幂等 + 重启支持）；长轮询 pollLoop（167-244）：get_updates_buf 增量游标、失败退避（maxFail=5/backoff 30s/retry 3s，131-135）、sessExp(-14) 触发 SessionExpired 回调并退出（203-211）；handle（246-266）→ ChatFunc → Send（270-291，context_token 上下文）；apiPost（326-350）iLink 私有头（AuthorizationType=ilink_bot_token、X-WECHAT-UIN、iLink-App-ClientVersion=132099）。
- 扫码登录：qrlogin.go GetQRCode（29-56）/PollQRStatus（58-91）/PollQRStatusWithCode（93-119）。
- **无前端板块**（3.0 §3.1 层间不一致 3）：app 层 initWeixin（app.go:316）注入 chatFn，仅此。

### 5.6 对照样本：办公引擎的 Provider Seam（已存在，须作为 Step 3 模板）
- internal/gaea/provider/provider.go：`Provider` 接口（282-289，Name/Stream）、`Config`（291-299，含 Engine 字段用于 bridge 类）、`Factory` + `registry` + `Register(kind, f)`（323-335）、`New(kind, cfg)`（338-351）、`Kinds()`（353-361）、StreamInterruptedError（363-389）。
- bridge：provider.Register("wubigrok", ...)（bridge.go:158-173），把 ai.LLMClient 适配成 provider.Provider（Stream 30-122），注入点 SetClient/SetFeature（144-156）。
- **反例**：bridge.go:41-51 仍硬编码 `p.engine == "herdsman" || p.engine == "ollama"` 开思考模式——特性分支泄漏到 seam 消费者。
- 办公引擎 TOML 配置已含 providers 段：internal/gaea/config/config.go:21-36（Providers []ProviderEntry）、257-278（ProviderEntry：Name/Kind/BaseURL/Model/Models/Default/APIKeyEnv/BalanceURL/ContextWindow/Price/Prices/Thinking/Effort）——**这正是 Step 3 想要的「换后端只改配置」形态**。

---

## 6. 配置层与 provider 选择

### 6.1 配置体系全景（三份，互不统一）
1. **应用配置** ~/.gaea_config.json（JSON，非 TOML）：internal/config/config.go:97-167 configFile + 168-289 Config。设计文档 §7 说「config.toml」，实际 app 层是 JSON；TOML 只存在于办公引擎（gaea/config，§5.6）。
2. **引擎状态** <dataRoot>/engines.json：modelengine engine.go:578-653（Manager 状态快照，APIKey 不落盘）。
3. **办公引擎 TOML** 项目 ./gaea.toml / ~/.config/gaea/config.toml：gaea/config/config.go:1-5（flag > 项目 > 用户 > 内置）。

### 6.2 应用配置项全集（Key 常量，config.go:18-86）
- 写作/推理：novels_dir、default_temperature、analysis_temperature、reasoning_effort、quality_threshold/max_retries、model（19-40 段）。
- **provider 选择类**（直接决定路由）：
  - image_backend（32，值域 "xai"/"comfyui"/"herdsman"/"ollama"，config.go:109/212/481）
  - portrait_backend/portrait_model（36-37，角色库剧照独立后端，空=跟随绘梦）
  - active_engine_id（40）+ model（41）：全局 LLM 引擎
  - active_asr_engine/active_asr_model（42-43）、active_tts_engine/active_tts_model（44-45）、tts_voice（46）、active_ocr_engine/active_ocr_model（47-48）：模型中心三段选择
  - func_chat/novel/office/gaea/characterlib/routine 的 engine+model（53-64）+ enabled（66-71）：功能级绑定与启停
  - func_chat_voice_engine/model（50-51）：聊天语音绑定
  - sensitive_local（74）：敏感域本地化
  - keep_warm_enabled / auto_preload（76-77）：本地模型调度
  - deepseek/opencode_go/opencode_zen api_key（78-80）
  - usd_cny_rate（82）、cosyvoice_dir/port（84-85）
- 加载：Load（config.go:430-786），优先级 默认值(438-488) < 环境变量(491-566) < 文件(568-780)；损坏备份 .corrupt-时间戳（770-779）；旧品牌 .wubigork_config.json 回退（570-580）。
- 保存：Save(key,value)（857-885）+ saveSetters 注册表（926-1111）+ 原子写（890-923）。
- 功能级模型读写：GetFeatureModel/SetFeatureModel/GetFeatureModelEnabled（295-384）——按 feature switch 七分支（chat/whisper 别名/novel/office/gaea/characterlib/routine）。

### 6.3 provider 选择的总体格局（as-is）
- **全局 LLM**：active_engine_id → modelengine.Manager（引擎清单+凭据）→ ai.Client 两分支路由（§2.2）。
- **功能 LLM**：func_*_engine/model → ChatRequest.EngineID 覆盖 → 同样落到 ai.Client 路由。
- **生图**：image_backend + portrait_backend → app 层 switch 构造 ImageBackend（§2.5，四处重复）。
- **TTS**：active_tts_engine/model + func_chat_voice_* → app 层 tryEngineTTS/TTSSpeakBase64 回退链（§4.2）。
- **ASR**：active_asr_engine/model → 但实现只有 herdsman 协议（§4.5）。
- **OCR**：active_ocr_engine/model → gaea_ocr.go 回退链（§7 O1-O3，4 级回退：选中→/v1/ocr→/v1/parse→本地 docmd）。
- **办公引擎**：gaea.toml providers 段（已 seam 化，§5.6）。

---

## 7. Step 3 seam 目标清单（switch 逐处 + 难度评估）

> 难度：低 = 现成接口 + 单一装配点；中 = 多装配点或行为差异需测试固化；高 = 跨包重构 + 行为矩阵。

### LLM seam（定义 LLMProvider{Chat,Stream}，正式化 wubigrok bridge）
| # | 位置 | 现状形态 | 改造 | 难度 |
|---|---|---|---|---|
| L1 | internal/ai/client.go:180-196 resolveChatEndpoint | 两分支：xai vs engineMgr.BuildChatURL | chat 端点解析下沉到 provider（wubigrok 注册表项），Client 只持接口 | 低 |
| L2 | internal/ai/client.go:202-232 resolveModelName | engineID=="xai" 分支决定默认模型 | 默认模型进 EngineConfig/ProviderConfig（已有 DefaultModel） | 低 |
| L3 | internal/ai/client.go:413 与 :591 | 401 刷新仅 reqEngine=="xai" | 认证策略进 provider 定义（xAI 特有 OAuth） | 中 |
| L4 | internal/ai/client.go:1036-1046 | herdsman/ollama 开思考模式 | 能力标志（supports_thinking）进 EngineConfig/ProviderEntry | 低 |
| L5 | internal/gaea/provider/bridge/bridge.go:41-51 | 同 L4 的重复分支 | 同上，bridge 只读能力标志 | 低 |
| L6 | internal/modelengine/engine.go:392-401 与 :671-679 | 按 Type if-else/switch 取凭据 | 凭据解析进 provider（api_key_env 模式，见 gaea/config ProviderEntry） | 低 |
| L7 | internal/modelengine/engine.go:102/109-188 NewManager | 7 引擎硬编码预置 | 预置表数据化（或配置驱动），注册表可追加 | 中 |
| L8 | internal/modelengine/engine.go:513-533 | opencode 兼容白名单硬编码 | 端点形态（chat/responses/messages）进 provider 元数据 | 中 |

### Image seam（定义 ImageProvider{Generate,Interrupt?,ListLoras?}，注册表 + image.backend 驱动）
| # | 位置 | 现状形态 | 改造 | 难度 |
|---|---|---|---|---|
| I1 | internal/app/app.go:410-436 initImageBackend | switch image_backend（comfyui/herdsman/ollama/xai） | 注册表 image.Register("comfyui",...) 等 + GetImageProvider(cfg) | 低 |
| I2 | internal/app/writing_state.go:59-84 restoreImageBackend | **同 I1 的第二份 switch（含 comfyui 分支还顺带装 herdsman 的 bug 状逻辑）** | 删除，复用注册表 | 低 |
| I3 | internal/app/characterlib_gen_handler.go:396-421 buildPortraitClient | 第三份 switch（portrait_backend 或 image_backend） | 同上（portrait 只改 provider 配置） | 低 |
| I4 | internal/app/image_handler.go:561-608 SetImageBackend | 第四份 switch（写配置 + 构造后端） | 只写配置，装配走注册表 | 低 |
| I5 | internal/app/image_handler.go:683-703 | 模型列表按 backend switch 补充默认 | 默认模型进 provider 定义 | 低 |
| I6 | internal/ai/client.go:1081-1089 GetImageBackend | 裸接口 + 调用方类型断言拿 Interrupt | 扩展接口（可中断能力）入 seam 定义 | 中 |
| I7 | internal/ai/client.go:1129-1137 GenerateImage | backend!=nil 委托否则 xAI | 无默认后端时从注册表取 | 低 |

### Voice/TTS/ASR seam（定义 TTSProvider/ASRProvider，注册表 + 模型中心选择驱动）
| # | 位置 | 现状形态 | 改造 | 难度 |
|---|---|---|---|---|
| V1 | internal/app/tts_handler.go:55-109 TTSSpeakBase64 | 4 级硬编码回退（选中→扫描引擎序→Edge→SAPI） | 注册表 chain：provider 列表来自注册表 + 配置顺序 | 中 |
| V2 | internal/app/tts_handler.go:113-249 TTSSpeakStreaming | 硬编码引擎列表 + label switch（224-229） | 同 V1；Synthesizer 已是接口（tts/edge.go:337-339） | 中 |
| V3 | internal/app/voice_handler.go:197-235 tryEngineTTS | xai/cosyvoice/herdsman 三分支 + 模型名选构造器 | 按 provider 能力（tts_mode: customvoice/voicedesign/voiceclone）路由 | 中 |
| V4 | internal/app/voice_handler.go:128-163 applyASRClient | 三级扫描 + isSTTModel 名匹配 | ASR 注册表 + 引擎能力分类（复用 classifyModelKind） | 中 |
| V5 | internal/app/voice_handler.go:175-193 synthesizeVoiceTTS | chatVoice → 全局 → base64 回退 三链 | 同一 TTS 注册表，仅配置不同 | 低 |
| V6 | internal/app/voice_model_handler.go:190-207 ttsVoiceForModel | 模型名 switch 默认音色 | 默认音色进 provider 定义 | 低 |
| V7 | internal/tts/herdsman.go:337-354 defaultVoiceForModel | 模型名 switch | 同上（与 V6 重复逻辑） | 低 |
| V8 | internal/voice/voice_config.go:26-34 TTSEngine 枚举 | 与 active_tts_engine 脱节、TTSEngineAuto 未消费 | 统一为模型中心引擎 ID，删除伪枚举或改为文档性别名 | 中 |
| V9 | internal/asr/herdsman_asr.go（整包） | 仅 herdsman 协议实现 | ASRProvider 接口 + herdsman 提供者（首个）；换引擎不再改 voice_manager | 低（接口化） |
| V10 | internal/voice/voice_manager.go:74/107-120 | asrClient/回调函数注入 | 注入 provider 接口（可测性不变） | 低 |

### OCR seam（定义 OCRProvider{Extract}）
| # | 位置 | 现状形态 | 改造 | 难度 |
|---|---|---|---|---|
| O1 | internal/app/gaea_ocr.go:16-36 GaeaOCRText | 4 级回退（选中→/v1/ocr→/v1/parse→本地 docmd） | 注册表 + active_ocr_engine/model 驱动 | 中 |
| O2 | internal/app/gaea_ocr.go:40-72 herdsmanOCRWith | 模型名含 mineru 走 parse 否则 ocr | 端点形态进 provider 定义 | 低 |
| O3 | internal/app/gaea_ocr.go:145-161 pickHerdsmanModel | capability→模型名关键词 | 复用 classifyModelKind（engine.go:478-509） | 低 |

### 分类/枚举统一（跨 seam 公共债）
| # | 位置 | 现状形态 | 改造 | 难度 |
|---|---|---|---|---|
| C1 | engine.go:478-509 vs app/voice_handler.go:166-171 vs app/gaea_ocr.go:145-161 vs app/voice_model_handler.go:159 | classifyModelKind/isSTTModel/pickHerdsmanModel/语音校验 四处关键词分类 | 单一分类器（含 tts/stt/ocr/rerank/embedding/image/llm） | 低 |
| C2 | internal/modelengine/stats.go:151-163 | estimatePrice 对本地引擎短路 | 引擎元数据（is_local）驱动 | 低 |
| C3 | internal/config/config.go:295-384 | GetFeatureModel 按 feature switch | feature 进 Manifest（Step 2 联动），配置映射数据化 | 中 |

---

## 8. 缺陷与风险

1. **生图后端 switch 重复 4 处**（app.go:410-436 / writing_state.go:59-84 / characterlib_gen_handler.go:396-421 / image_handler.go:561-608），且 writing_state.go 的 comfyui 分支同时装配 herdsman 后端（66-73）——行为不一致的高危重复。Step 3 Image seam 收益最大、成本最低。
2. **引擎枚举与分支编译期硬编码**：加一个引擎要改 NewManager（engine.go:98-191）、order（102）、fetchModels 凭据（392-401）、BuildChatURL（671-679）、401 文案（410-419）、可能还有 opencode 白名单（513-533）与 classifyModelKind。与「换后端只改配置」的目标直接冲突。
3. **401/token 策略只对 xAI 生效**（client.go:413/591）：deepseek/opencode 等引擎 401 时无刷新重试；而 OAuth token 刷新逻辑（GetToken/tryRefreshToken）与 API key 体系并存于同一 Client（client.go:237-319）——认证策略应按 provider 隔离。
4. **特性分支泄漏到 seam 消费者**：herdsman/ollama 思考模式开关同时出现在 ai/client.go:1036 与 bridge.go:41-51，两处魔数 4096（max_tokens 抬升）重复。能力标志应进 EngineConfig/ProviderEntry。
5. **语音配置双体系脱节**：voice.TTSEngine 枚举（voice_config.go:26-34）与实际路由键 activeTTSEngine 脱节；TTSEngineAuto 等值写进配置后无人消费（voice_handler.go:402-404 vs 175-193）。ASR 同理——枚举声称 3 模型，实现只有 herdsman 协议（asr 包）。
6. **ASR/OCR 是「伪多引擎」**：引擎选择只是换 baseURL/model，协议仍是 herdsman；换真后端（如 whisper.cpp 本地进程）需要新客户端，当前无 seam 承载。
7. **配置三份割裂**：~/.gaea_config.json（JSON）+ engines.json + gaea.toml；同一种「引擎选择」语义（active_engine_id vs 办公 providers 段 vs func_* 绑定）三种表达。Step 3 应至少在 seams 层统一消费面。
8. **模型名启发式分类 4 处重复**（§7 C1），且关键词匹配会随模型命名漂移静默错分（如 "voice" 命中 tts 却可能是 ASR 模型名），影响模型中心/语音/OCR 的路由正确性。
9. **herdsman 契约脆弱性被显式承认**：probe.go 探测「非公开契约」（config.yaml 格式、events.jsonl 等）并给出契约漂移告警（probe.go:319-322）；lancheck 用逐行 YAML 解析（不引库）——herdsman 升级会破坏 gaea 的本地能力链，当前靠告警而非隔离层兜底。
10. **grok-tts / CosyVoice2-0.5B 伪模型注入**（app.go:293-296 EnsureModel）依赖引擎 ID 字面量；cosyvoice 服务拉起逻辑（tryEngineTTS 206-208 / TestEngineConnection 38-43）也按 engineID 字符串分支——本地 TTS 服务的生命周期管理游离在注册表之外。

---

## 9. 改造建议

1. **以 internal/gaea/provider 为模板，把 Provider Seam 推广到 app 层**：复制 Register/Factory/New 模式（provider.go:323-351），新建 internal/providers（或 app/providers）承载 llm/image/tts/asr/ocr 五个注册表；office 引擎的 wubigrok bridge 保留，作为第一个已注册的 LLM 提供者。
2. **Step 3 子步排序（按收益/成本）**：
   - 3a Image seam（清理 §7 I1-I7，消灭 4 处重复 switch，纯重构、可回归）；
   - 3b LLM seam 正式化（L1-L8：chat 路由/凭据/默认模型/能力标志进注册表；认证策略隔离）；
   - 3c OCR/ASR/TTS seam（O1-O3、V1-V10；先接口化 asr 与 tts，行为用现有 *_test 固化）；
   - 3d 分类统一（C1-C3）+ 配置消费面统一。
3. **能力标志优先于字符串分支**：在 EngineConfig/ProviderEntry 增加 supports_thinking / auth（oauth|apikey）/ tts_mode / endpoint_shape（chat|responses|messages）等字段，消灭 client.go:1036、bridge.go:41、gaea_ocr.go:62 一类的模型名/引擎名判断。
4. **Manager 定位收敛**：modelengine.Manager 保留为「引擎清单 + 凭据 + 状态 + 统计」层（§3），**移除** URL/凭据解析（BuildChatURL）——该职责归 provider；stats.go 的 estimatePrice 本地短路改由 IsLocal 元数据驱动。
5. **认证策略进 provider 定义**：xAI OAuth 刷新只发生在 xai provider 内；deepseek/opencode 的 API key 解析走 api_key_env（对齐 gaea/config ProviderEntry.APIKeyEnv），删除 ai.Client 里针对引擎名的 401 分支。
6. **语音配置单源化**：删除 voice.TTSEngine 伪枚举或降级为兼容别名，统一读 active_tts_engine/active_asr_engine（模型中心），voice.Manager 的注入面从「函数」升级为「provider 接口」。
7. **统一模型分类器**：classifyModelKind（engine.go:478-509）作为唯一分类入口，isSTTModel/pickHerdsmanModel/语音校验改为调用它或共享关键词表。
8. **配置迁移零破坏**：新增 seam 字段全可选（对齐 3.0 §7「格式不变；新增字段全可选」）；image_backend/active_engine_id 等旧键继续生效，测试固化各后端行为差异（验收标准见设计 §5.3）。

---

## 附：关键文件行号速查
- ai 统一调用面：internal/ai/interface.go:8-19；internal/ai/client.go:25-58, 180-196, 202-232, 335-465, 470-545, 987-1055, 1057-1137
- 引擎注册表：internal/modelengine/engine.go:19-33, 45-70, 84-96, 98-191, 305-379, 383-474, 478-533, 578-653, 655-682；stats.go:19-32, 151-163
- 语音状态机：internal/voice/voice_manager.go:52-88, 163-217, 222-260, 406-508, 644-657, 707-741；voice_config.go:14-133
- TTS 引擎选择：internal/app/tts_handler.go:55-109, 113-249；voice_handler.go:175-235；voice_model_handler.go:77-207；internal/tts/edge.go:337-367, herdsman.go:29-59, 131-148, 337-354, xai.go:8-64
- ASR：internal/app/voice_handler.go:128-171；internal/asr/herdsman_asr.go:26-51
- herdsman：internal/herdsman/health.go:88-136, 207-230；probe.go:34-62, 139-188, 299-323；lancheck.go:43-130
- auth：internal/auth/oauth.go:59-216, 223-353；token.go:23-193；discovery.go:26-59
- weixin：internal/channels/weixin/clawbot.go:22-64, 81-119, 167-291, 326-350；qrlogin.go:29-119
- config：internal/config/config.go:18-86, 97-289, 295-384, 430-786, 857-1111
- 对照样本（seam 模板）：internal/gaea/provider/provider.go:281-361；bridge/bridge.go:18-173；gaea/config/config.go:21-36, 257-278
- 引擎启停：internal/app/herdsman_lifecycle.go:66, 180-222
