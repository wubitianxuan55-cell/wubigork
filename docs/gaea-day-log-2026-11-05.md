# 当日总账（2026-11-05）：语音 V2 唯一路径化大日——卡死根修 + 网关/V1 双退役 + 协议四修 + 交互重设计 + bundle v2

> 批末=桌面 exe build/bin/gaea.exe（C:\AI\gaea-app 装机同哈希），桌面入口收敛为
> 单个 gaea.lnk 快捷方式（散件 exe/3 DLL/gaea-go.bat 出清）。今日一笔主提交
> （语音生产代码全量）+ bundle v2 一笔 + 文档一笔 + release 一笔，均随本版列车。

## 提交索引（4 笔）

| 提交 | 线 | 一句话 |
|---|---|---|
| （C1）feat(voice)! | 语音全栈 | V2 唯一路径化：卡死根修（三环连环）+ 8792 网关退役 + V1 whisper 链退役 + 协议四修（409 退避/兜底转写/terminal 发布/task schema）+ 交互重设计 + 真实电平/哑麦检测 |
| （C2）feat(voice) | 分发 | bundle v2：可重定位净布局（exe+三 DLL+models/voice+装机脚本）+ gaea_install.ps1 一键装机 |
| （C3）docs(voice) | 台账 | 审计台账/主计划/本日总账 |
| （C4）chore(release) | 发版 | v4.532.0（wails.json+versioninfo.rc 对齐） |

## 卡死根修（用户报「语音系统启动连接卡死」三连重启同死局）

三环连环，CDP 沙箱实锤（9 次 WS/8.1s 全被弹）：
1. **并发 connect 竞态**：ensure 就绪瞬间事件/轮询/按钮三路同 tick 进入
   connect，引用 guard 在 ensureSink await 后才置位 → 三个 WebSocket 同建。
2. **孤儿会话锁死服务端**：第一个 WS 抢到会话槽（池=1）后被引用覆盖成
   孤儿（无人 close，浏览器 pong 保活 → 60s 读超时杀不掉），后续连接全被
   session_limit_reached 秒弹且服务端零日志。
3. **onOpen 清零重连计数**：每次被弹都重数 1/5 → 无限 1s 重连循环。

修：connect 并发去重（connectingRef 首个 await 前置位）+ 重连计数稳定
10s 窗复位（RECONNECT_STABLE_MS，翻转「open 即清零」固化测试）+ 服务端
60s 零客户端帧空闲接管（lastInbound 原子戳；接管等旧 epilogue 防配置抹除）
+ 拒绝留痕日志（incumbent_idle）。

## 网关退役（刀6 核心）

voice_v2_gateway.go -444 行；voiceNativeLLMLeg 进程内腿吸收注入逻辑
（prepare 台账/【本轮上下文】独立消息保 prefix cache/plain 长期记忆/失败
+取消双回滚逐字同位）；路由恒走聊天功能绑定；voice_chat_llm radio 退役；
status 缩 {realtime,breeze}。用户 voice_chat_llm=chat 实况=行为零变化。

## V1 whisper 链退役（用户拍板）

前端 V1 组件/hook/设置 V1 卡/模型中心 ASR 半边全删；后端 voice_handler
17 函数/voice_manager(1681 行)/internal/asr/云 realtime/config 三键出清；
朗读链与麦克风设备工具保留（micDevices util 迁出）；绑定 710→709。

## 协议四修（ws_probe 实弹驱动）

1. TTS 腿 breeze 409（共享引擎抢占）退避重试 400/800/1600ms。
2. F1 兜底语进转写面 + 终态前冲刷停靠 delta（原 done 转写空白）。
3. F2 hasResponse 响应期 user terminal 即时发布（speech_stopped 上线）。
4. TaskTool 全 Schema 补 engine/model（compact_drift 逮住的既有漂移）。

R3 零输出判定野外生效：TTS 全灭回合诚实判 cancelled/empty_output。

## 交互重设计 + 真实电平（ui-ux-pro-max 检索落地）

- bar 相位电平条（listening/speaking 起伏·thinking 呼吸·error/interrupted
  语义色）+ preflight「装配中」态 + 错误重试标签 + 过时文案修正。
- captureProcessor RMS 240ms 上报 → micLevel/micSilent（开麦 3s 无信号=
  哑麦警示）——**实弹当场抓真问题：本机默认录音设备=零信号虚拟设备（ToDesk）**。

## 时序坑（记档防复发）

- bargein 插话语料必须全长（短语料 384ms 阈值丢弃，sidecar 同律）。
- TTS 秒发音频：done 与插话确认只差 27ms——长首句造料恢复原 PASS 口径。
- 沙箱 LLM「登录已过期」=环境噪声非管线病（兜底路径正好被实弹验证）。
- tsc 必须 -b 口径；vitest 必须 frontend/ cwd 净跑（仓库根扫 clones/.tmp 假红）；
  管道 rc 谎报（`cmd | head` 的 rc 是 head 的）。

## 真麦「无法识别语音」根修（2026-11-05 晨，用户实测驱动，待真麦复测）

**现象**：v4.532.0 装机后用户真麦说话全丢弃——`VAD: discarding segment=820ms
active=160ms`，四段全过不了 384ms 语音门，STT 零触发。

**诊断弯路（记档）**：①先怪阈值（0.75→0.5 播放期双门：监听 silero 默认 0.5，
TTS 首音频 EchoGuardOn 0.75 防扬声器回声自打断=保住 11-01 实弹成果，finishResponse
关门覆盖 barge-in cancelled）——降阈后 active 不变，阈值无罪；②再怪电平——合成弱
语料（×0.25，RMS 0.0099）@0.5 也全 discard，电平假设也无法解释。

**真凶（决定性实验，vadcheck 工具对照）**：同一段弱语料——torch.hub silero
（sidecar 时代）mean=0.719；原版 silero_vad_v5.onnx（python ORT）mean=0.708；
**Go 手术版按 (1,512) 喂=mean 0.100**。根因=**silero v5 ONNX 输入契约
(1,576)=上块末尾 64 样本 context+本块 512；torch JIT 内部自维护 context，
ONNX 导出后挪到输入侧，Go 按 512 直喂=context 恒错位**。强电平信噪比高错位
不致命（假麦/合成语料全链实弹全绿的盲区），弱电平（用户麦 RMS≈0.01~0.02）
判性崩——「昨天还行今天不行」=sidecar(torch)→Go(ONNX) 重写丢契约。
fix_silero_sr 图手术无辜（onnx 库 load+save 往返字节级一致；手术「EQUIVALENT」
验证用 randn(1,512) 在错误形状下做，两边一起错当然「等价」——教训：等价性
验证的输入必须在正确契约下取，且须含弱电平真实语料）。

**修**：SileroVAD 自维护 ctx64（Prob 拼 [ctx64,x]→(1,576)，推进 ctx=x 尾 64，
Reset 清零；非 512 块 fail-loud）。修复后弱语料 mean=0.708 与 torch 同档、
连 0.75 都过门；ws_probe 弱料全链逐字转录 PASS（e2e 4.6s）+cancel/bargein 零泄漏。

**随刀**：①captureProcessor 自适应增益（1s 窗原始 RMS<0.012 升 ×2 上限 4，
原始 peak>0.22 回 1 防削波；包络统计恒用增益前防自激励；正常设备恒 1）——
弱麦保险+STT 受益，worklet 3 单测；②前端 vadThreshold 0.75→0.5+探针同步+探针
零响应回合 panic 修（b4f2d8f7 同族）；③discard 行加临时 rms 字段（验真麦后撤）；
④vadcheck 概率曲线探针（scripts/voice_v2/vadcheck，exe 目录须带 DLL——System32
ORT 劫持旧坑）。**M5 §7 四项拍板固化**（T0 免确认采纳/确认词立项默认关/首批
status+save_task/读屏挂未来重评），M5 解冻待排刀。

## 遗留

- 真机一句话验收（用户侧）；M5 排刀；误打断率语料门（venv 出局后
  A/B 基线改 Go 自证口径待定）。

## M5 三刀落地（2026-11-05 午后，§7 拍板后解冻；未实弹待用户真机验证）

- **刀一（cb885709）T0 免确认+回执朗读**：PreviewIntent 对 navigate/status
  直接执行返回 executed（不入台账）；前端出回执不出确认卡；回执 TTS 朗读
  （TTSSpeakStreaming 后端播+前端拉阈值 0.9 防回声假插话+估时恢复 0.5）。
  执行层抽 execVoiceV2Action 包级桩点（单测禁连真引擎——GetModelMonitor
  挂死教训）。契约四表同批（bridge/spaceBindings/legacy 摘除/mock 桩）；
  顺手修 bindingSignatureCount 717→709 既有漂移（HEAD 态即红）。
- **刀二（967b1fc4）模型工具通道**：语音 LLM 腿注入首批工具 schema
  （get_status/save_task）；流式截获 tool_calls→T0 直执行回执（ack 追加进
  回复流——前缀半句已播不可撤回）；T1 登记 P2 台账+gaea-voice-tool-preview
  事件下发凭据→前端直出确认卡（复用 P2 执行钮）。频控=同 action 60s≤3
  （防模型循环发起）；工具面即白名单（T2/T3 不在清单）。
- **刀三（3c21f0e5）语音确认词（默认关）**：voice_confirm_word 键四登记+
  handler 透传+面板 Switch；onTurnComplete 去标点精确匹配「确认执行」→
  复用执行钮通道（幂等早退/每预览一次/非播放期）。确认词轮照常走 LLM
  （管线面拦截挂后续刀）。
- 测试：后端 config+app 全量绿；前端 1056 全绿；tsc 0。装机。
- 待办：真机实弹三刀（T0 说「打开运行监测」直接切+朗读；工具轮说「查一下
  现在用什么模型」走 get_status；「帮我记个任务 xx」出确认卡+「确认执行」
  语音确认）→发版。

## 网络搜索刀1（众神殿共识落地，晚间）

- 依据：众神殿 `pan_1793872913023`（议题「网络搜索能力弱」，3 轮四席全签三刀刀谱）；
  论证台账 docs/gaea-web-search-pantheon-digest-20261105.md，实施计划
  进度计划/gaea-web-search-optimization-20261105.md。
- 刀1 落地：统一检索核 RunSearch/SearchAttempt/ClassifySearchState（builtin/websearch_core.go，
  策略滤空≠ALL_FAILED 拆分；办公 Execute 输出契约零变化）+聊天 adapter
  （app/chat_search_adapter.go：清洗下沉+移除「在哪里/多少钱」剥尾+8s 硬超时+注入头
  按状态选文案）+plain 提示词删「不要说自己无法联网搜索」+四入口收敛（plain/人格/
  WhisperWebSearch 绑定/agent loop 委托缝 whisper.WebSearchToolExec，boot 注入）；
  whisper 自研 Bing/DDG 栈整删（web_search.go −253 行，符号删除=源级守卫）；人格路径
  失败不再静默；WhisperWebSearch 死 error 字段修活。
- 测试：桩引擎注册带清理（不污染 AllKinds 全量断言）；core 测试 count=5 无抖动
  （修两处竞态：胜者先发 result 后发 attempt+收集循环按尝试计数）；builtin/whisper/app
  三包全量绿，build/vet 0。
- 余：主源 A/B 待用户拍板（刀3+A 臂基线前置）；30 query keyless 基线待跑（保留窗/
  覆盖门阈值定数）；刀2 短窗合并+主源保留窗+覆盖门 v0 按基线数值落。
