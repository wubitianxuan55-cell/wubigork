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
