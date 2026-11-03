# Gaea Voice V2 — M4 产品接入刀报告（voice.backend=v2）

> 日期：2026-11-01
> 承接：[M4 快模型报告](./gaea-voice-v2-m4-fastmodel-report-20261101.md)（验证层）
> 方案：[V2 总方案](./gaea-voice-v2-rearchitecture-20261101.md) §12/§15（feature flag 接入）

## 1. 结论

**V2 以 feature flag 进入 gaea 产品本体：设置面板可切档，聊天页语音台在 v2 档
渲染实时会话视图并连上 sidecar，轮完成写回聊天话题。V1 一行未动，回退=切回 v1。**

验证：单测 59 绿（前端 44 新 + Go 3 新 + 既有面回归）+ `tsc -b`/eslint/knip/
双 drift 零回归 + build.bat 冒烟过 + **真机 CDP 走查 10/10 PASS**。

## 2. 落地面

**Go（配置位五文件账 + 监护）**
- `voice_backend` 配置链：config_keys/types/save/load + getter（`voice.backend`
  位：v1=whisper 链〔默认〕| v2=实时 sidecar）；`voice.VoiceRuntimeConfig.VoiceBackend`
  + Validate（空/未知一律回 v1）；`VoiceGetSettings`/`VoiceApplySettings` 透传
  （既有绑定零新增改动面）。
- `internal/app/voice_v2_service.go`：**VoiceV2Status**（三依赖只读探测）/
  **VoiceV2Ensure**（幂等拉起 breeze→4B(8123)→sidecar(8791) 链，异步
  voice-v2-status 事件）/ **VoiceV2Stop**（按端口击杀，breeze 共享不动）。
  探测/spawn 全 seam 注入，测试绝不真拉 GPU 服务。
- 绑定四表同批 + gen_bindings 三件套（711 方法）。

**前端（frontend/src/voice/ 扩 runtime 层）**
- `runtime/useVoiceChatV2.ts`：会话运行时钩子（VoiceClientV2+双 worklet+播放泵+
  依赖链监护+mock 回放）；transport 双工厂（真连 WebSocket / 回放哑线缆分离——
  防哑线缆冒充真连接）。
- `components/chat/VoiceConsoleV2.tsx`：v2 语音台视图（相位徽章/实时转写/助手栏/
  开麦/打断/拉起依赖链）；**轮完成写回**：`ChatAppendMessages` 单事务双消息落库 +
  视图追加，话题绑开启时锁定（useChatVoice 同款防串台纪律）。
- ChatPage：`voiceBackend` 读取（一次性，切档下次开台生效）→ v2 档渲染
  VoiceConsoleV2；人格指令=当前人格 label+voiceGuide 折叠进 session instructions
  （完整人设/记忆接 gaea 内部端点=下一刀）。
- VoiceSettingsPanel 新「语音对话引擎」卡：v1/v2 切换 + 三依赖状态 + 拉起钮。
- Lab（?voicev2=1）重构为钩子消费方（与语音台同一实现，不再有平行线）。

## 3. 真机走查（CDP 10/10）

Lab 回归（钩子重构后 mock 回放全句转写+回执 144B）→ VoiceV2Status 三依赖就绪 →
voiceBackend=v2 落盘回显 → 聊天板块 → 语音台渲染 **VoiceConsoleV2** → 连 sidecar →
相位空闲 → 无未就绪警示 → 收尾切回 v1。走查配置零残留。

## 4. 坑（本轮新增）

1. **Git Bash `cmd /C` 被 MSYS 路径转换吃掉**（/C→C:\）→ cmd 进交互态空跑——
   一律 `cmd //C`（taskkill 同族）。
2. python 追加法改 ts 文件：rstrip 砍掉文件尾 `}` 再拼，把新方法拼到了 return
   对象外——**tsc -b 才抓到**（build.bat 比 `tsc --noEmit` 多查测试工程）；
   追加代码一律 Edit/Write 工具，禁止字符串拼接改源码。
3. `npx tsc | head; echo $?` 的 `$?` 是 head 的退出码——判绿必须看 tsc 自身
   （本轮 `tsc -b` 直跑终验）。
4. WebView2 调试口被 user-data-folder 池化旧浏览器吞掉（旗标在命令行、端口不开）：
   杀掉该 WUDF 的全部 msedgewebview2 后冷启即恢复（M1 老坑的新形态）。
5. knip：懒挂载组件只准 default export（VoiceConsoleV2 重蹈 Lab 覆辙）。

## 5. 边界与遗留

- **未动**：whisperChatCore/V1 语音链/聊天数据库 schema/35B 文本路由（§0.5 红线
  全程遵守；DSH 在途面零碰撞）。
- **下一刀**：`/internal/voice/chat/completions`（gaea 人格/记忆/落库接管 V2 的
  LLM 位，方案 §11）——当前 v2 档的人格=instructions 折叠，记忆写回只经
  ChatAppendMessages 落库面。
- 挂实弹：真机开麦全链（v2 档语音台说话→打断→写回聊天）；sidecar 饿死 bug
  （M3 发现）上游跟踪；2.5s 灰区手感；sidecar 分发拍板（3GB 增量 vs 提取协议栈）。
- 运行态：走查壳（9333 可驱）+ 8791/8123/8012 常驻；`voice.backend` 当前=v1。

## 6. 实弹反馈刀（2026-11-01 晚，同日六续）

用户实测三反馈（慢/只输出前半段/声色不固定）。**取证推翻直觉：实测跑的是 V1**
——gaea 日志实锤（19:56~19:59 全为 `[whisper] WhisperChat start`）：
- 「只输出前半段」= **扬声器回声被麦克风拾音→每轮自打断**：日志每条都带
  「用户打断了你上一轮未说完的语音回复」，用户消息里出现「我是盖亚」「看你
  发呆的样子」——那是 AI 自己的播报被浏览器 ASR 听写进去（回声环实锤）；
- 「慢」= V1 35B 链（已知 Vulkan prefill 瓶颈，正是 V2 立项理由）；
- 「声色不固定」= V1 无预设回退（人格=plain）+ tts 音色查询 404 告警。
- 根因：走查收尾把档位切回 v1 + ChatPage 档位开页只读一次 → 用户在面板切了 v2
  落盘成功，但页面停在 v1——**切档不生效是产品级 UX bug**。

**四连修（全部落地，64/64 单测 + tsc -b + build + CDP 热切走查过）**：

| 修 | 落点 |
|---|---|
| 切档即时生效 | Go `VoiceApplySettings` 发 `voice-backend-changed` 事件；ChatPage 订阅热切语音台（v1 台自动停会话、v2 台挂载即连），零重载 |
| V2 人格音色 | 角色卡 `ttsVoiceId`（breeze 克隆 id）→ ChatPage 拉取 → `session.update.audio.output.voice`（空=服务端缺省）——声色从配置链固定 |
| 防回声假插话 | 采集阶梯**每一档保留 echoCancellation/noiseSuppression**（原裸档丢 EC=外放自打断同款雷）；VAD 阈值 0.6→0.75 |
| 端点提速 | `smart_turn_max_wait_ms 2000→1200`、`incomplete_delay 600→400`（社区初值偏保守；代价=密集停顿语速切句更激进，挂实弹校准） |

坑：wailsjs 模型字段是 `ttsVoiceId`（小写 d，非 TTSVoiceID）；python 拼接改源码
两连翻车（替换锚不匹配静默失败、return 对象外拼接）——**改源码一律 Edit/Write**。

走查：热切 v1→v2 当场换台零重载 ✓；sidecar 就绪 ✓；档位落盘回显 v2 ✓。
**交付态：voiceBackend 留在 v2、桌面副本 20:12 新鲜、8791/8123/8012 常驻——
用户开聊天页语音台即为 V2 实时档**（外放仍建议耳机，回声消除有限）。

## 7. 开麦无反应刀（同日七续）

用户实测「点开麦无反应」。两连根因，逐一实锤：

1. **vite 把 AudioWorklet 内联成 data: URL，而 Chromium `audioWorklet.addModule`
   不接受 data: URL**——构建态必炸；M2 走查没炸是因为 vite dev 服务的是源文件
   （构建态从未点过开麦=验证盲区）。修：worklet 挪 `frontend/public/voice-worklets/`
   （原样拷贝进产物），引用改绝对路径 `/voice-worklets/*.js`。
2. **`!!AudioContext.prototype.audioWorklet` 特性检测在真浏览器抛
   `Illegal invocation`**（audioWorklet 是 prototype 上的 accessor getter，带
   接收者校验，prototype 上取值即抛；jsdom 桩/裸探针都测不出）。修：改
   `'audioWorklet' in AudioContext.prototype`（`in` 不触发 getter）。

配套：startMic/connect 全链异常上屏（此前 onClick `void` 吞 promise 异常=
按钮无声失败）；AudioContext 16k 档失败回落默认率。诊断手段：CDP 真点 +
`Runtime.exceptionThrown` 抓栈 + 逐步阶段标签 + sourcemap 缺失时 bundle
列号反查——异常上屏后 Illegal invocation 才暴露在 startMic 首行。

验证：构建态真机 CDP 点开麦——采集→VAD→response.done completed→播放回执
全链真实跑通（事件尾屏留证）。桌面副本 20:35 新鲜，档位 v2，依赖链常驻。

## 8. 稳定闭环 + Gaea 记忆网关（同日八续）

### 8.1 连接/轮次/播放三处产品级根修

- **连接不再假装成功**：`connected` 只在 transport `onopen` 后置真；连接失败、
  异常 close 会清 client/transport，可原地重连。断开同步停麦，不再留下向空
  client 继续送 PCM 的孤立采集。
- **response 与用户转写精确配对**：session ledger 在 `response.created` 消费
  pending transcript；用户文本和助手全文都记在对应 response 下。迟到的旧
  `response.done` 不再拿“全局最后一句”拼成新轮，重复 done 也不会双写聊天。
- **打断清两层音频**：旧实现只清 `PlaybackQueue`，但播放泵每 100ms 会把音频
  提前灌入 AudioWorklet ring，服务端 done 后 ring 仍可能继续播。现 worklet
  支持 `flush` 控制帧，`speech_started`/显式打断/新 response/断线会同时清
  调度队列与 ring。麦克风静音本身不再误杀正在播放的 AI 音频。
- **设备采样率不再冒充 16k**：个别设备拒绝 `AudioContext({sampleRate:16000})`
  时，采集 worklet 先抗混叠降到 16k 再切 320 样本；播放 worklet 把 16k 源线性
  升到设备实际采样率。开麦手势内恢复 suspended sink，避免“显示已连接但没声”。
- **依赖探测一致化**：`VoiceV2Status` 同一快照每端口只探一次，避免三项与 ready
  自相矛盾；设置页轮询改用每次 `refreshV2Status()` 的返回值，不再读闭包旧态。

### 8.2 Gaea 对话大脑接入第一刀

新增 `internal/app/voice_v2_gateway.go`，与 App 同生命周期运行于
`127.0.0.1:8792`：

```text
sidecar @8791 → Gaea 记忆网关 @8792 → 4B llama-server @8123
```

- OpenAI `/v1/chat/completions` 请求透明流式转发（包括 tool call/SSE；
  网关不拆模型 token）；
- 读取当前 `activePersonalityID`，从该人格 FactStore 先召回触发词命中，再补
  高权重/核心事实，严格限制 8 条、900 字；只读注入 system，短期对话仍是
  sidecar 原序 messages；
- `plain` 模式不注入人格记忆；网关固定转发到 qwen3-4b，拒绝 sidecar 模型名
  漂移；
- 四依赖状态新增 `gateway`，设置页显示“网关 ✓/✗”；
- App 退出关闭网关；`voice.backend=v2` 时同步停 sidecar+4B，不再留 3GB 模型
  无控制面常驻；breeze 共享引擎不动。
- `scripts/voice_v2/gaea-spike.yaml` 的 `responses_api_base_url` 已切到 8792。

边界：本轮接入的是**长期记忆读取 + 当前人格**；事实抽取/关系状态收尾仍由
文字轻语链负责，V2 语音轮的权威写回仍是聊天页单事务。若要让 V2 自己触发
FactStore 抽取，需要下一刀单独接 `PreLLMTurn/PostTurn`，不能仅靠网关代理。

### 8.3 验证

- 前端语音/语音台/设置面板定向 **53/53 绿**，其中覆盖 true-onopen、断线重连、
  迟到 done 配对、重复 done 去重、done 后仍有 PCM 的打断、worklet flush 帧、
  16k/48k 采集与播放重采样；
- Go `VoiceV2` 定向绿：四依赖探测、单次探测、网关 SSE 转发、system 注入顺序、
  plain 不注入、活跃人格 FactStore 召回；
- `tsc -b` + `vite build` 构建态绿；`go vet ./internal/app ./internal/voice` 绿；
  `go build` 临时产物成功。`-race` 本机因 cgo 未启用无法执行，不是测试失败。
- Wails 完整构建+启动冒烟过：`build/bin/gaea.exe` 56,851,968B，
  SHA256 `49D5B32117BC7106C0AF65911414FF07F806A29EE1FCF5750834F855A862F769`；
  桌面副本同哈希。构建后 12 个 Go 绑定门面由项目生成器复位，wailsjs 仅做
  CRLF 机械复位，无绑定签名变化（711 方法）。
- 当前 8791/8123/8012 仍是**上一版常驻进程**；8792 只由新 App 启动。实弹前
  必须启动包含本轮改动的新产物并重启 sidecar，使其重读 8792 配置。

## 9. V2 回合进入轻语记忆尾链（同日九续）

上一轮只做到“读取长期记忆 + 前端 ChatAppendMessages 落聊天”；V2 自己生成的
用户事实不会进入 FactStore。本轮新增后端绑定 `VoiceV2CommitTurn`：

```text
response.done(completed)
  → topic.mode 判定人格（前端不可伪造人格）
  → chat 单事务落 user+assistant
  → assistant.extra 写 voice_v2_response_id（持久判重）
  → 复用 whisper PreLLMTurn / WM / FinalizeTurn
  → TurnTrace 落库
  → FactStore 异步抽取 + companion_state 持久化
  → 返回 emotion/stage/totalTurns/warning
```

- 人格来源改为**话题 mode 单一事实源**；plain 模式只落聊天，不写入任何角色
  记忆，避免普通语音污染 persona。
- 同一 `response_id` 的 WebSocket 重试/重复回执只落一次聊天、一次状态推进、
  一次记忆抽取；判重读 assistant extra，进程内锁外仍有持久依据。
- V2 的模型生成仍由 sidecar 完成，Gaea 不再二次调用模型；只复用生成后的
  状态/记忆尾链，保持语音延迟目标。
- 聊天已保存但人格记忆尾链失败时返回 `warning`，语音台可见并提示，不把已成功
  的聊天错误回滚或报成整轮失败。
- 前端 `VoiceConsoleV2` 不再直调 `ChatAppendMessages`；改为等待后端权威提交后
  追加视图并刷新话题标题，避免聊天成功、状态/记忆失败时无提示。

验证：`go test ./internal/voice ./internal/app` 全量绿（App 169s）；覆盖单次提交、重复 response 零副作用、
plain 不写人格、空参数/未知话题拒绝、memory payload 的 user/assistant/turn。
前端语音/设置/契约定向 **58/58 绿**，`tsc -b` 绿；绑定漂移 712 方法一致；
Wails 完整构建+启动冒烟过，桌面副本与上方最终产物同哈希。边界：生成前仍未消费
`PreLLMTurn` 的完整动态提示，当前是在生成后补推进人格状态；下一刀若要把
关系/情绪也实时影响 V2 回答，应把“准备回合”和“提交回合”拆成同 `turn_id`
的两阶段协议。

## 10. 直接启动真机定位“拉起失败”（同日十续）

按用户反馈直接启动 `build/bin/gaea.exe`，用 Windows 电脑控制完成
聊天页 → V2 语音台 → 拉起依赖链 → 开麦 → 转写/回复/播放全链操作。
首轮实测暴露四个与模型服务无关的启动问题：

1. **sidecar HTTP `/` 探测错误**：sidecar 的 `/` 返回 404/偶发不完成，
   旧 `voiceV2Probe` 用 HTTP GET 会持续超时。现 8791 改 TCP 监听探测；
   仍打印 `404` 恰好证明服务已起，但不作为失败。
2. **冷启动窗口过短**：Paraformer + Smart Turn + Breeze 预热实测可超过
   90s，旧 UI 60s/设置页 30s 已先报失败。现后端 sidecar 硬超时 180s，
   语音台与设置页同口径。
3. **裸命令解析错误**：`detachedCmd("cmd")` 用 `os.Stat` 检查裸命令，
   在 Windows 不可靠；改为 `exec.LookPath`，sidecar 父/子进程现已稳定拉起。
4. **状态刷新不自愈**：后台最终已就绪，但语音台依赖按钮 Promise，未刷新。
   现增加 `voice-v2-status` 事件消费 + 未就绪期间每 3s 低频轮询，
   四项齐备后自动 `connect()`，无需用户二次点击。

真机结果：

- 单次点击“拉起依赖链”后，8012/8123/8791/8792 全部监听，界面自动变为
  可连接并建立 WebSocket；
- 开麦后出现真实转写、`response.done`、PCM 播报与播放回执，最终回到空闲；
- 语音轮已写回聊天页，关闭语音台后能看到成对用户/助手消息，证明
  `VoiceV2CommitTurn` 的权威落库生效；
- 测试结束后已关麦，避免继续采集环境声音。

## 11. 回合身份闭环：生成前快照不再靠文本猜配（同日十一续）

M4 的下一刀原计划是把“准备回合”和“提交回合”拆成同 `turn_id` 两阶段。
本轮先落了协议中最关键的一段身份链：

```text
sidecar input item_id
  → Chat Completions 私有请求头
  → Gaea 生成前 persona 快照绑定
  → 前端 response ledger
  → VoiceV2CommitTurn
  → 同一快照收尾
```

### 11.1 根因

此前网关已会在 4B 生成前执行 `PreLLMTurn`，但待提交快照只按
“人格 + 用户文本”匹配。用户连续两次说同一句、旧 `response.done` 迟到，
或两代请求重叠时，旧回答可能认领新准备的人格状态；这正是文本匹配无法
消除的串轮缝隙。

### 11.2 落地

- `scripts/voice_v2/gaea_sidecar_launcher.py` 在 sidecar CLI 启动前安装一个
  窄 monkeypatch：把已有 `turn_id / turn_revision / input item_id` 放进
  `X-Gaea-Voice-*` 私有请求头。补丁不改 prompt、不改助手文本，重复加载幂等。
- 8792 网关读取三个私有头，把 `item_id` 绑定到预推进快照；上游失败按快照
  指针精确回滚，不再靠用户文本猜测回滚目标。
- `VoiceSessionState` 的 response ledger 同时保存用户文本和 `item_id`；
  `response.done(completed)` 回执携带该 id，Wails
  `VoiceV2CommitTurn(..., responseID, itemID)` 消费对应快照。
- 前端提交链串行化：连续多轮即使 Wails 异步返回乱序，聊天问答对仍按语音
  发生顺序落库。`response_id` 持久判重与 `item_id` 快照关联继续双重防守。
- 旧 sidecar 未发头时保留“人格 + 文本”兼容路径；新 sidecar 一旦有 id，
  以协议身份为唯一匹配键，相同话术也绝不互换。

### 11.3 验证

- sidecar 启动器真实 CLI `--help` 加载通过；3 个 Python 断言覆盖请求头注入、
  最新用户 item 识别、已有供应商头保留；
- Go `go test ./internal/whisper ./internal/app` 全绿（app 250.131s），新增覆盖
  网关绑定 item id、相同文本不同 item 拒绝误领、失败按快照回滚；
- 前端语音/语音台/契约定向 **54/54 绿**；`tsc -b`、Vite 生产构建、
  本次改动文件 ESLint、`go vet ./internal/app` 全绿；
- Wails 绑定已重生为 5 参数，712 方法签名清单同步；
- `go build -o .tmp/gaea-voice-v2-turn-id.exe .` 完整编译通过：
  70,928,896B，SHA256
  `E71B0D81AC56D648FD827B20CE9F0FA73EB5A2E7D4DCF9054862FAE92AA0ADC9`。

全量前端回归为 4170/4172：运行监测用例单跑已转绿；小说设置页一致性检查仍
失败，来自当前工作区既有非语音改动。全仓 ESLint 另有两条既有错误
（`useVoiceChat.ts` 的 `prefer-const`、`ModelSwitcher.i18n.test.tsx` 的
`no-explicit-any`），均不在本次改动面。

### 11.4 生效边界

当前桌面实例仍锁着 `build/bin/gaea.exe`，且 8791 sidecar 在改动前启动；
为避免打断正在进行的产品会话，本轮未强杀重启。下一次关闭/重开桌面应用并
从“拉起依赖链”启动 sidecar 后，新启动器才会加载请求头补丁。未重启时网关
自动走旧文本兼容路径，功能不回归，但没有精确 item 防串轮。

## 12. “启动本地语音”卡死根修（同日十二续）

### 12.1 根因

桌面副本 `C:\Users\wubi\Desktop\gaea.exe` 启动时工作目录不是仓库，
`repoRootFromExe()` 会从 exe 所在目录向卷根查找脚本。Windows 上
`filepath.Dir("C:\\")` 仍返回 `C:\\`，旧循环却将它和 `VolumeName` 返回的
`C:` 比较，条件永远为真；goroutine 在根目录无限 `GetFileAttributesEx`，因此
8791 sidecar 永远不会进入拉起阶段，前端一直显示“准备本地语音”。

诊断端口 `/stack` 实锤 goroutine 卡在 `voice_v2_service.go` 的仓库根查找。

### 12.2 修复

- 祖先目录遍历改为 `parent == dir` 终止，并对重复候选去重；支持
  `GAEA_VOICE_V2_ROOT`，最后回退当前 spike 装机根 `C:\AI\wubigrok`。
- sidecar 不再经 `cmd.exe /C start_spike.bat`，改为直接执行 venv Python +
  `gaea_sidecar_launcher.py`，消除批处理路径二次解析和旧 `cmd.exe` 错误。
- 后端状态新增 `stage/starting` 与阶段日志；前端启动中显示当前阶段、禁用
  重复点击，并在后端报告终止错误时立即停止轮询，不再用固定 180s 掩盖失败。
- 语音设置页同步消费 `error/stage`，超时或失败可见。

### 12.3 验证

- 新增卷根终止、祖先遍历去重、环境变量根覆盖测试；定向 Go 测试与
  `go vet ./internal/app` 绿。
- 前端语音运行时/语音台/设置 **22 例**绿，`tsc -b`、相关 ESLint 0 error。
- 从桌面目录启动新产物并点击“启动本地语音”：按钮立即进入禁用进度态；
  8012/8123/8791/8792 全部监听，26s 内自动进入“已连接 · 待机”，
  sidecar 日志确认 WebSocket session 建立。
- Wails 完整构建通过；桌面副本与 `build/bin/gaea.exe` 同哈希：
  56,892,928B，
  `FA982E8C7B630225376CBD7D9DC8A43DCB4D8EDCD3C907A919E0C589D6BE97CF`。
