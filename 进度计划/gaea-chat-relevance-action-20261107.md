# 聊天角色答非所问——众神殿裁决承接与 action-3 执行（2026-11-07）

> 承接众神殿 `pan_1794012727813_7dd108312e0d`（2026-11-07 上午，4 轮 4 席
> codex/zcode/grok/dsh，议题「如何解决聊天角色答非所问，胡乱说话」，
> 结论=**部分解决**）。裁决行动序：action-1 病例卡冻结 → action-2/3 并行 →
> action-4 冻结构建 → action-5 真实模型复跑 → action-6 仅在仍败时模型隔离。
> 禁令：不先调人格文案；代理例/假 LLM 探针只关技术行，不关用户主诉行。

## 一、三处病灶核实结果（裁决引用 → 今日树状态）

| # | 病灶 | 核实状态 |
|---|---|---|
| ① | VAD 前缀跨轮累积（语音输入污染，vadhandler） | ✅ **已修已装机**——2236cba1（11-07 01:07 语音线三批修刀批0）`ensureTurnForSpeechStart` 跨轮清前缀守卫。action-2 只剩真麦复测 |
| ② | 取消路径丢已生成正文 | ❌ 四处全在：`chat_tool_loop.go` chunk.Error 丢当前轮 sbContent；`whisper_handler.go:434` 连此前轮 res.Reply 一起抹；`chat_service.go` plain 工具/非工具两分支取消裸奔 `error:"请求已取消"` 且 partial 丢弃、不落库 |
| ③ | 前端超时只 `finish(false)` 不调 `ChatStreamCancel` | ❌ 在：后端继续生成+落库全文 ≠ 界面保留的 partial（两份权威）；B2 已把「超时清正文」修成「保留 partial」但没接撤销 |

## 二、action-3 执行记录（2026-11-07，本批）

### 后端（终稿闭合）

1. **chat_tool_loop.go 工具循环取消抢救**：chunk.Error 且 ctx 已取消时，当前轮
   已缓冲正文 `stripToolCallMarkup` 清洗后并入 `res.Reply`/`res.Reasoning` 并
   onDelta 外发，返回 nil 错误交上层按 ctx.Err() 走取消终态——取消不再丢当前轮残段。
2. **whisper_handler.go 取消不抹已累计正文**：`loopErr+ctxErr` 分支由
   `reply,reasoning="",""` 改为 `NormalizeReplySplitLines(res.Reply)`——此前轮
   已交付正文在取消终态中保留。
3. **chat_service.go plain 工具分支**：取消轮不再 emit `error:"请求已取消"`，
   回落 `res.Reply` canonical 走 cancelled 落库流程。
4. **chat_service.go plain 非工具分支**：chunk.Error+ctx 取消 → break 走取消落库
   （对齐 whisper 非工具分支 P1-5 语义）。
5. **chat_service.go 空轮取消**：cancelled 空终态（对齐 persona 空轮语义，不伪造
   成功）+ **用户输入单条落 user 行**（AppendMessage 单条口）——typed 消息不随取消湮灭。

### 前端（useChatStream.ts）

6. **超时撤销后端任务**：`timeoutMessage` 先调 `ChatStreamCancel(activeRunID)`
   再保留 partial 收尾——后端不再继续生成并落库完整正文形成第二份权威。
7. **cancelled 帧空 reply 回退本地 partial**：后端 canonical 优先，空（打断在
   缓冲段/空轮）时用本地已外发正文——可见正文不被「已停止生成，未收到内容」吞掉。

### 测试（新增 7 例，默认运行）

- Go：`TestChatStreamCancel_PlainNoTool_KeepsPartial`（非工具取消保留 partial+落库同文）、
  `TestChatStreamCancel_PlainToolLoop_EmptyRound`（空轮 cancelled+仅落 user 行）、
  `TestChatStreamCancel_PlainToolLoop_CrossRoundKeepsProse`（跨轮取消=此前轮散文+当前轮残段全保留）、
  `TestWhisperChatCancel_ToolLoop_KeepsPartial`（persona 工具轮取消保留残段）。
- 前端：超时撤销后端（ChatStreamCancel 被调+partial 保留）、cancelled 空 reply 回退本地
  partial、cancelled 带 canonical reply 以后端为准。
- **测试基建坑（新录）**：服务端「写完即信号」在客户端消费前存在竞速——工具轮正文
  先完整缓冲、mid-round 无客户端可见信号，`ssePartialThenHang` 信号前留 500ms 本地
  回环消费窗；非工具路径 delta 边流边外发，可在首个 delta 事件确定性取消。

### 回归

- `go test ./internal/app/ -count=1` 全包通过（168.9s）；
- `tsc -b` 0 错；ChatPage 20/20、SearchCard+MessageList 20/20。

## 三、action-1 病例卡（草稿冻结，待用户确认项标 ❓）

### 卡 A · 语音转写格（whisper_gaea，2026-11-07）

| 字段 | 值 |
|---|---|
| 入口 | 语音（whisper VAD→ASR→whisperChatCore） |
| 会话/角色 | chat_history.session_id=`whisper_gaea`；角色 ❓（用户确认哪位） |
| 模型/构建 | 10:48 装机壳 PID 31888 sha256=94785E23（含回声门 v2）；上轮 10:32 壳 C0E41CB2 |
| 症状（第一轮复测 10:36~10:38） | ①重复输入 ②自己的话当输入 ③杂音当输入 |
| 复测取证（10:36~10:38 日志+DB） | **滚雪球已消失**（转写 3/3/8/4 字，2236cba1 前缀清零生效）；真凶=TTS 播放期扬声器回声被 VAD 确认成假插话——回声猝发 active=192~384ms 全过 0.75 阈值门（silero 对 TTS 回声=真语音给高分），打断级联 `tts=cut` 掐碎回复（turn_1 rev=0..4、turn_3 rev=0..2），回声再被转写成「你不能/其有我」类杂串进对话。**三症状同源=回声门缺时长闸** |
| 输入真值 | ❌ 缺原始分轮音频（隐私）——用户确认当轮实体即可升 L1 |
| 期望锚点 | ①本轮转写=本轮实说（无历史前缀）②TTS 播放期间无假插话/假打断 ③回答针对本轮问题 |
| 修复状态 | 前缀清零已验证 ✅；回声门 v2（确认窗 600ms）二轮实弹证伪——**长句回声 active=608ms 过门**，任意有限确认窗输给连续回声；**回声门 v3 已装机**（2d63a752：上游 `block_mic_during_playback` 同位，播放期+尾窗 600ms 捕获侧整帧丢弃，VAD 零摄入；对源锚 audio_client.py:1046-1048）；**待三轮真麦复测**——日志应有 `echo gate on (capture-drop half-duplex…)`/`echo gate off (…droppedMs=N)` |
| 残余风险 | 播放期语音打断半双工让位（打断走 UI/协议 interrupt 通道）；真语音打断恢复=未来 AEC/按住说话 |

### 卡 B · 文本 persona 格（bokke/tsundere，2026-11-06）

| 字段 | 值 |
|---|---|
| 入口 | 文本角色对话（persona 流式） |
| 会话/角色 | bokke / tsundere（whisper 探针会话）；模型 8124 qwen35 |
| 症状 | ①诗/作品进 GenUI 围栏而非正文 ②「重新写」原样重发 ③口诉「说一段散文」多轮拖延 |
| 输入真值 | 探针会话逐轮留档（docs/gaea-chat-probe-delivery-contract-20261106.md） |
| 期望锚点 | ①作品直出正文非围栏 ②重写=新作品 ③口诉交付一轮完成 |
| 修复状态 | B3/B4/A3 残刀已修已装机（四病灶探针治愈）；**待用户实弹复测确认主诉可关** |
| ❓ 用户确认 | 是否仍有新病例？有则补「入口/角色/一轮原话/期望」四项再立卡 |

## 四、下一步

| 项 | 谁 | 内容 |
|---|---|---|
| action-2 | **用户** | 装机态真麦连续 5 轮语音，确认每轮原始输入、验证无跨轮前缀累积 |
| action-4 | 执行体 | 语音修复（已装机）+ 本批 chat 取消/超时修复汇入同一发布候选，冻结构建身份（SHA256） |
| action-5 | **用户**+执行体 | 冻结构建上复跑卡 A/卡 B，按预登记锚点判定；通过才关对应入口的主诉行 |
| 观察项 | 执行体 | codex/zcode 席位 usage 上报（缓存命中率证据）另开一刀；B3 残余=三态搜索控件+forceSearch 兜底（前一日搜索计划）不与本线混车 |
