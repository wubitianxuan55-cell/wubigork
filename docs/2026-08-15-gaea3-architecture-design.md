# gaea 3.0 架构改造规划设计

> 日期：2026-08-15 ｜ 状态：v1.0 定稿 ｜ 范围：架构主线改造（编程板块已按用户指示搁置）
> 证据基础：docs/gaea3-review/ 下 9 份分域评审报告（全部带 文件:行号 证据）+ .gaea/reviews/ 5 份阶段 7 审查。

---

## 0. 摘要

gaea 3.0 是一次**架构主线改造**，不是功能大版本：

- **方向**：原则级向 DeepSeek Harness（DSH）靠拢——事件日志作为会话事实源、板块 Manifest 化、Provider Seam 化、合成根统一装配。
- **边界**：技术栈留在 Go + React；不移植 Cordis 插件框架；不做全量运行期动态装配；不引入 TS 组件。
- **顺序**：Step 0 修债 → Step 1 事件日志 → Step 2 板块 Manifest → Step 3 Provider Seam → 发布 3.0.0。编程板块已搁置（用户指示），**不在当前范围内**；Manifest 验证试点由知识库与 chat 承担。
- **原则**：每步独立可验收、向后兼容、可回退；2.x 功能增量持续合入，直至 3.0.0 发布条件满足。
- **调研基础**：9 份分域评审报告（docs/gaea3-review/01-09，全部带文件:行号证据）+ .gaea/reviews/ 5 份阶段 7 既有审查。产。

---

## 1. 背景与动因

三个**实证缺陷**（均可复现，见论证记录）驱动本次改造：

1. **加板块 = 改壳代码（编译期耦合）**
   新增一个板块需要同时改动：MainLayout.tsx（Page 类型 / menuItems / pageComponents / allPageKeys / 快捷键）、ModuleLauncher.tsx（LauncherTarget / modules）、新增页面组件、后端新增绑定方法并重跑 gen_bindings、手工 Register 到模块注册表——约 6 处，全部编译期耦合。板块无法独立演进。
2. **模块注册表与派发规则不同步（活跃缺陷）**
   classifyMainBrainIntent 对"标书/招标/报价/proposal/tender"返回 ("office","create")，但 initModules（internal/app/module_bindings.go）只注册了 gaea/whisper/novel/imagegen 四个模块。office 仅在测试文件中注册过。主脑识别到办公意图 → Has("office") 为 false → **静默跳过，无任何日志**。
3. **会话存储不是事实源（技术债自认）**
   internal/gaea/agent/session/save.go 注释原文："The file is rewritten in full on every save… append-only would have to be reconciled with the compaction pass that mutates the middle of session.Messages."——整文件重写 JSONL，压缩与 append-only 冲突。会话无法回放、多前端无法同步、标题/统计/成本全靠即时内存计算。

**方向论证摘要**：DSH 的架构（Cordis 五判据：服务定位 / 依赖声明 / 可逆注册 / 事件通信 / 合成根）中，gaea 办公引擎内部已满足 1/3/5 的局部形态（tool.Registry、event.Sink、boot.Build 合成根、MCP plugin.Host），缺的是 2（依赖声明）与 4（拦截式事件语义）；板块层五项全缺。因此"向 DSH 靠拢"是**把引擎内已验证的架构思路推广到板块层**，不是重写。

---

## 2. 目标与非目标

### 2.1 目标

| 编号 | 目标 | 对应机制 |
|---|---|---|
| G1 | 加板块只写声明，不改壳代码 | 板块 Manifest（Step 2） |
| G2 | 会话可回放、可派生（标题/统计/成本/多前端） | 会话事件日志 + 投影（Step 1） |
| G3 | 换 provider 只改配置，不改调用方 | Provider Seam（Step 3） |
| G4 | 装配点唯一、前端与内核解耦 | 合成根升级（Step 2/3） |
| G5 | 注册表与派发规则一致性由机器保证 | 启动自检断言（Step 0/2） |

### 2.2 非目标（明确不做）

- ❌ 不移植 Cordis/任何 DI 框架（Go 侧用接口 + 配置表达，不引第三方容器依赖）
- ❌ 不引入运行期热装卸插件（板块在启动时装配，生命周期固定）——**用户已确认（2026-08-15）：个人产品完全插件化是过度设计**；工具级 MCP 热增删为保留例外
- ❌ 不做多形态分发（保持 Wails 桌面单形态；httpbridge 继续作为调试桥，不升级为产品级协议层）
- ❌ 编程板块已搁置（用户指示，见 §9 搁置说明）
- ❌ 不重写办公引擎（其架构已达标，改造只触及会话存储与装配）

---

## 3. 现状架构盘点（as-is）

```text
┌────────────────────────── 前端（React + antd） ──────────────────────────┐
│ MainLayout（静态导航 menuItems/pageComponents）                          │
│   首页 聊天 小说 绘梦 办公 记忆中枢 模型中心 角色库 设置                    │
│   ├─ pages/*：ChatPage / NovelPage / GaeaPage(办公) / ...                │
│   ├─ gaea/（办公 UI：Sidebar/Transcript/Composer/FileTree/ChangesPanel） │
│   └─ wailsjsCompat：10 个门面的再导出合并层                               │
└───────────────┬──────────────────────────────────────────────────────────┘
                │ window.go.main.App.*（Wails 绑定） + runtime.EventsOn 事件
┌───────────────┴──────────────────────────────────────────────────────────┐
│ 后端（Go 1.26 / Wails v2）                                               │
│ internal/app/                                                           │
│   ├─ core{ctx,cfg,client,engineMgr,chatStore,charLib} ← 共享基础依赖     │
│   │   └─ 指针嵌入 4 子状态（writingState/mediaState/whisperState/        │
│   │       officeState）→ App（Startup 33 步装配，app.go:249-374）        │
│   ├─ core.emit = 事件唯一出口（httpbridge.Publish + EventsEmit 双发，    │
│   │   app.go:214-226；66 发射点 / 21 类事件名）                          │
│   ├─ gaeaRuntime = 包级单例（gaea_handler.go:26，非 App 成员）            │
│   ├─ bindings_*.go（10 门面 462 方法，gen_bindings 反射生成，纯委托）    │
│   ├─ module_registry.go（Module{ID,Name,Intents,Handle}）← 半成品        │
│   ├─ main_brain.go（关键词 → intents 派发）← 与注册表不同步              │
│   └─ gaea_handler.go / gaea_ui*.go（办公引擎装配 + 绑定）                 │
│ internal/gaea/（办公引擎 —— 架构达标区）                                  │
│   ├─ boot.Build（合成根：config→模型→工具注册表→gate→Controller）        │
│   ├─ control.Controller（transport-agnostic 会话驱动）                   │
│   ├─ agent/（AgentRunner、session.Save/Load 整文件 JSONL）               │
│   ├─ event（类型化事件 Kind + Sink，单向流）                             │
│   ├─ tool（Registry + builtin init() 自注册 + MCP plugin.Host）          │
│   ├─ sandbox / command / skill / memory / search / jobs                  │
│ internal/modelengine/（引擎枚举 xai/ollama/herdsman/deepseek/...）        │
│ internal/httpbridge/（反射 RPC + SSE 调试桥，GAEA_HTTP_PORT）             │
└──────────────────────────────────────────────────────────────────────────┘
```

## 3.1 板块全景清单（五层对照 —— 早期草案曾遗漏，此处为准）

**结论先行：gaea 的"板块"在五个层面各有各的清单，互不一致。** 这是 Manifest 化要解决的核心问题，也是本设计最重要的输入。

### 层 1：前端导航（MainLayout 实际渲染，9 页）

| # | key | 名称 | 页面 | 入口 |
|---|---|---|---|---|
| 0 | home | 首页（启动器，非板块） | HomePage | 菜单 |
| 1 | chat | AI 聊天 | ChatPage | 菜单 / Ctrl+1 |
| 2 | novel | 小说 | NovelPage | 菜单 / Ctrl+2 |
| 3 | imagegen | AI 绘梦 | ImageGenPage | 菜单 / Ctrl+3 |
| 4 | gaea | 办公 | GaeaPage | 菜单 / Ctrl+4 |
| 5 | memoryhub | 记忆中枢 | MemoryHubPage | 菜单 |
| 6 | modelcenter | 模型引擎中心 | ModelCenterPage | 菜单 |
| 7 | characterlib | 角色库 | CharacterLibraryPage | 菜单 |
| 8 | settings | 设置 | SettingsPage | 右上按钮 |

另：**KnowledgePage（知识库）存在但未挂载任何导航**——孤儿页面，其能力已被 memoryhub 的 knowledge 库取代（MemoryHubPage 聚合 8 库：knowledge/cost/profile/office/materials/whisper/graph/digitallife）。

### 层 2：后端绑定门面（10 个，≠ 板块，是绑定面分组）

CoreB（壳：auth/project/stats/settings）｜ OfficeB（办公引擎主面，137 方法）｜ MemoryB（记忆域，31 方法，全部 Gaea* 前缀）｜ CostB（成本域，22 方法，全部 Gaea* 前缀）｜ ModelB（模型中心）｜ VoiceB（轻语+语音，45 方法：Whisper*/Voice*/TTS*）｜ ChatB（聊天）｜ NovelB（小说）｜ ImageB（绘梦）｜ CharlibB（角色库）

**关键事实**：MemoryB/CostB 的绑定方法全部是 Gaea* 前缀——它们服务的是办公引擎的记忆/成本域，被 MemoryHubPage 经 gaea bridge 调用。**门面数量（10）与板块数量（8）天然不对应**，门面是"按 Wails 绑定面拆分的 API 分组"，不是板块。

方法数全景（= TestBindingsCompleteness 的 462，精确吻合）：CoreB 48 / OfficeB 137 / MemoryB 31 / CostB 22 / ModelB 34 / VoiceB 45 / ChatB 25 / NovelB 67 / ImageB 38 / CharlibB 15；其中办公引擎域（Gaea* 前缀）194 个占 42%。

### 层 3：模块注册表（module_registry，仅 4 个）

gaea（主脑）｜ whisper（轻语）｜ novel（小说）｜ imagegen（绘梦）
—— 没有 chat/characterlib/modelcenter/memoryhub；主脑意图识别（classifyMainBrainIntent）却覆盖 office/novel/whisper/imagegen/gaea 五个域，office 未注册（§1 缺陷 2）。

### 层 4：README 宣称模块（7 个）

对话 ｜ 轻语 ｜ 小说 ｜ 绘梦 ｜ 模型引擎 ｜ 工程办公 ｜ 微信助手
—— 轻语 = 聊天板块内的人格模式（无独立页面）；**微信助手（channels/weixin，ClawBot beta）无任何前端页面**，前端仅在角色库编辑器出现过"微信"字样。

### 层 5：后端服务域（internal/app handler 域，29 个）

analysis, auth, chapter, character, characterlib, characterlib_gen, chat, context, copilot, create_chapter, feature_model, gaea, graph, image, lorebook, model_engine, ocr_model, office, outline, platform, plot_branch, project, stats, tts, visual, voice, voice_model, whisper, worldview

### 层间不一致清单（Manifest 化必须弥合）

1. 导航 8 板块 vs 注册表 4 模块 vs README 7 模块 vs 门面 10 个 vs 服务域 29 个——五层对不齐；
2. KnowledgePage 孤儿页面（有实现、无挂载）；
3. 微信助手（weixin）有后端服务、无前端板块；
4. 轻语（whisper）是聊天板块内的模式，却在注册表/门面层独立成模块（VoiceB）；
5. 记忆中枢是"聚合板块"（8 库横跨办公记忆/轻语记忆/知识/成本/资料/图谱/数字生命），数据域横跨多个门面。

### Manifest 的板块清单（本设计提议的 canonical 集合，共 9 板块 + 1 壳 + 1 待决）

| canonical id | 名称 | 导航页 | 绑定门面 | 意图域 | 服务域 |
|---|---|---|---|---|---|
| chat | AI 聊天 | ChatPage | ChatB + VoiceB(whisper 部分) | whisper | chat, whisper, voice, tts |
| novel | 小说 | NovelPage + 子页 | NovelB | novel | chapter, character, outline, worldview, plot_branch, lorebook, graph, analysis |
| imagegen | 绘梦 | ImageGenPage | ImageB | imagegen | image |
| gaea | 办公 | GaeaPage | OfficeB + MemoryB + CostB | gaea | gaea, office, knowledge, memory, cost, skill, tasks |
| memoryhub | 记忆中枢 | MemoryHubPage | MemoryB + CostB | — | 8 库聚合 |
| modelcenter | 模型中心 | ModelCenterPage | ModelB | — | model_engine, feature_model, ocr_model, voice_model, herdsman |
| characterlib | 角色库 | CharacterLibraryPage | CharlibB | — | characterlib |
| settings | 设置 | SettingsPage | CoreB(部分) | — | auth, project, config, stats |
| weixin | 微信助手 | （无页面，beta） | — | — | channels/weixin |
| home | 首页 | HomePage | CoreB | — | （壳，不进 manifest 业务清单） |
| knowledge | 知识库 | KnowledgePage（孤儿） | MemoryB | — | **待决：恢复挂载 or 并入 memoryhub**（建议并入，见开放问题 D7） |

关键观察：
- **两级架构**：办公引擎（internal/gaea）已是"服务定位 + 合成根 + 事件流"；其余板块是"方法直调"。
- **前端桥接**：wailsjsCompat.ts 把 10 门面合并回旧命名空间形态，说明绑定面拆分对旧代码零影响——这是"门面式"改造成功的先例，Manifest 化应沿用同样策略。
- **事件链路**：Go 侧 runtime.EventsEmit(c.ctx, eventName, data)（app.go:225 的 emit 辅助）→ 前端 window.runtime.EventsOn（gaea/lib/bridge.ts 集中订阅）。

### 既有文档核查（docs/gaea2/ —— 仅作现状事实参考；其中 module-protocol.md §5 单窗口编排方向已废弃，不采纳）

1. **module-protocol.md（已删除）**：原 gaea2 时代的模块协议文档。其现状事实已核实并吸收进本设计：ModuleRegistry 协议（Module{ID,Name,Intents,Handle}/Register 拒重/Dispatch 显式报错/绑定 RunModule，bindings_chat.go）仍是现状代码；原文档声称 office(create→ProposalCreate) 已注册，**代码实况：office 未注册且 ProposalCreate 不存在**（grep 零命中）；三脑（brain.main/left/right + BrainStore，7 文件）仍是现状代码。其 §5「单窗口编排」方向已废弃，故删除该文档防干扰（备忘见 .gaea/AGENTS.md）。
2. **office-model-tool-call-chains.md**：办公板块两条模型入口的完整链路核对（聊天 Agent：controller→bridge provider（注册 kind 实为 "wubigrok"，bridge.go:159）→ai.Client→modelengine.Manager.BuildChatURL；功能直连：Gaea* handler→routeModel(feature)→model_router.go）——Provider Seam 改造的权威底图。
3. **2026-08-13-绘梦板块重构设计.md**：单个板块 UI 重构的市场调研范式（画布优先/渐进式披露/一个任务流/继承玻璃 HUD）——板块级设计已有先例方法论。

**对本设计的修正**：
- Step 2 的 Manifest 扩展现有 ModuleRegistry 协议层（Intents/Handle 是现状代码事实，与废弃愿景无关）：协议层 + UI 层（Page/Icon/Nav）+ 绑定层（Facades）+ 能力层（Tools）。
- Step 0 的 office 模块修复不是"补一行注册"：ProposalCreate 不存在——**D8 已决策 (b)**：office.create 路由到现成 GaeaSend（见 §10 决策记录与 §6 Step 0 细化清单）。

---

## 4. 目标架构（to-be）

```text
┌────────────────────────── 前端 ─────────────────────────────┐
│ MainLayout ← 由 BoardManifest 数据驱动（启动时拉取清单）     │
│ PageRegistry：page key → lazy 组件（main.tsx 注册表）        │
│ bridge.ts：统一事件订阅 + 门面调用（现状保持）                │
└──────────────┬───────────────────────────────────────────────┘
               │ Wails 绑定（门面仍静态生成，清单校验一致性）
┌──────────────┴───────────────────────────────────────────────┐
│ 后端                                                         │
│ board/ （新）：Board 接口 + 内置板块注册 + 启动自检            │
│   Manifest{id,name,icon,page,bindings,intents,tools,config}  │
│   ├─ 装配：NewBindings 由 manifest 生成（gen_bindings 升级）  │
│   ├─ 派发：module_registry 由 manifest 驱动 + 完整性断言      │
│   └─ 隔离：每板块独立 Controller；板块间经 App 协调（现状）   │
│ gaea/ 引擎：session 升级为事件日志 + 投影（Step 1）           │
│ providers/（新，逐步）：llm/image/ocr/voice seam 化（Step 3） │
│ httpbridge：保持调试桥定位                                    │
└──────────────────────────────────────────────────────────────┘
```

设计原则：
1. **声明优先于代码**：板块能力（页面/绑定/意图/工具集）进 manifest，代码只实现能力本身。
2. **投影优于改写**：会话的任意视图（消息列表/标题/统计/成本）从事件日志派生，不落第二份真值。
3. **seam 优于 switch**：provider 切换走注册表 + 配置，调用方只见接口。
4. **兼容优于迁移**：所有格式/结构变更保留旧物读取路径，旧会话/旧配置可继续用。

---

## 5. 核心机制设计

### 5.1 会话事件日志（Step 1）

**现状**（05 报告核实）：agent/session.Session{ Messages []provider.Message }，Save() 整文件重写 JSONL，Load() 全量读回；压缩在回合中途 session.Replace（compact.go:214），原文只进 .gaea/archive；事件系统 16 种 Kind、12 处发射点，但**零持久化**——ToolDispatch/ToolResult 参数与输出、Usage、审批/提问及答复、Compaction、Steer/Retrying 全不入日志。

**设计**：

- **硬不变量**（09 报告采纳）：seq = 已写行数（写入器单点保证）；payload 必须无损 JSON（json.Valid 校验，非法拒绝）；"模型可见必入日志"是不变量而非约定。
- **日志文件**：<sessionDir>/<id>.gaea-log.jsonl，append-only，每行：
```json
{"seq": 1, "ts": 1780000000, "kind": "user_message", "payload": {...}}
{"seq": 2, "ts": 1780000001, "kind": "assistant_started", "payload": {"id": "m2"}}
{"seq": 3, "ts": 1780000001, "kind": "assistant_delta", "payload": {"id": "m2", "text": "你"}}
...
{"seq": 9, "ts": 1780000010, "kind": "tool_call", "payload": {"id":"t1","name":"readfile","args":"{...}"}}
{"seq": 10,"ts": 1780000011, "kind": "tool_result", "payload": {"id":"t1","output":"...","truncated":false}}
```
- **事件种类**：日志事件 = 现有 16 种 event.Kind（event.go:19-69：TurnStarted/Reasoning/Text/Message/ToolDispatch/ToolResult/Usage/Notice/Phase/ApprovalRequest/AskRequest/TurnDone/CompactionStarted/CompactionDone/Retrying/Steer）+ 会话性补充事件（user_message/assistant_delta/assistant_message 等消息级事件）。规则：**模型可见的输入必须已入日志**（沿用 DSH 不变量）。
- **检查点**：压缩发生时写 <id>.gaea-checkpoint.json（压缩后的消息投影 + 最后消费的 log seq）。恢复 = checkpoint + 从 seq 后重放。
- **投影层**：projectMessages 纯函数表（只有 user/assistant/tool 三类事件投影为消息），checkpoint + tail 重放；接口与现有 Session.Messages 完全兼容，Session 改为日志 + 投影的薄封装。
- **崩溃恢复语义（torn-tail，09 报告新增，成本半天）**：最后一个 turn_done 之后的不完整行截断；完整中断 turn 用合成 closers 补平衡；模型调用前 flush 检查点（fail-closed）——断电/崩溃后日志仍可回放。
- **派生能力**（日志带来的新增收益）：会话标题（首条 user_message）、token/成本统计（usage 事件累加）、变更面板（tool_call→writefile 等）、多前端同步（同一日志，两个 sink）。
- **迁移**：Load 检测旧格式（无 .gaea-log.jsonl 而有旧 <id>.jsonl）→ 旧格式读入 → 首次保存时写新日志，旧文件保留至 3.1 清理。

**挂钩点**（04 报告结论）：后端事件唯一出口 = core.emit（app.go:214-226，httpbridge.Publish + Wails EventsEmit 双发），全量 66 处发射点归组 21 类事件名——Step 1 日志写入在此单点挂钩，即可覆盖全部事件。

**引擎侧实现锚点**（05/06 报告结论）：持久化事件 sink 追加在 boot.go:114 的 Sync 包装之后（sink 链 Sync→watchdogSink 已存在，追加一环即可）；save.go 演进为"事件日志→派生 Messages"，压缩摘要落 checkpoint；agent/session/ 已拆子包，改动点 = save.go 重写 + Snapshot 触发点 + 持久化 sink + 子代理共用格式 + Resume 兼容。**最小改造面**（06）：AuditLogger/DecisionLogger 全链路代码就绪但离线（钩子 agent.go:288-290/execute_one.go:186-195 已通，boot 不传 AuditFunc、app 不调 NewAuditLogger）——Step 1 首日即可接线得审计 trail；session/archive/audit 三份 JSONL 共享 schema 前缀（turn/session/tool_call id）。事件发射端、Sink 包装链、Provider 注册表、Options 注入面全部直接复用，不推倒现有架构。

**兼容红线**（03 报告结论）：前端恢复/回退/兜底全链依赖 GaeaHistory 输出格式（gaea_ui.go:93-126 + rebuildHistoryItems store.ts:62-83），事件日志改造必须保持该输出逐字节不变，否则恢复后的过程卡/变更面板/工具卡全跑偏。

**验收标准**：现有办公会话恢复/压缩/回退功能回归通过；GaeaHistory 输出与改造前逐字节一致（对同一会话）；新增派生 API（标题/统计）单元测试覆盖；Wails 事件丢包自愈三件套（看门狗 30s 校准 store.ts:434-443 / reconcileFinalAnswer store.ts:370-389 / localCancel store.ts:246-253）行为不变并补"事件丢失→自愈"测试。

### 5.2 板块 Manifest（Step 2）

**现状**：加板块改 6 处（§1 缺陷 1）；注册表半成品（缺陷 2）。

**设计**：

- **Manifest 定义**（JSON——前端零依赖可解析；Go 侧 encoding/json 直读）。TS 视角完整 schema（frontend/src/boards/types.ts）：
```ts
interface BoardManifest {
  id: string            // 板块稳定 id（导航白名单键，现 allPageKeys 的替代）
  label: string         // 菜单/面包屑/启动器显示名
  icon: string          // antd 图标名（图标注册表查表解析）
  page: string          // 页面组件 key，在 PageRegistry 中查找（替代 lazy import + pageComponents 两处登记）
  lazy: boolean         // PageRegistry 统一 lazy 包装
  keepAlive?: boolean   // visitedPages display:none 保活策略（默认 true）
  layout?: 'full' | 'padded'   // full = chat/gaea 全出血；padded = 默认 padding
  shortcut?: string     // ctrl+1~4 显式声明（不依赖数组顺序）
  menuOrder?: number    // 菜单顺序
  inMenu?: boolean      // 是否进菜单（settings 不进菜单，右上角按钮）
  breadcrumb?: { anchorTo?: string }  // 面包屑"项目名→novel"锚点语义
  isHome?: boolean      // 首页特判分支的替代
  nav?: { children: { id: string; label: string; page?: string; icon?: string }[] }  // 板块内子导航（NovelPage 6 tab/SettingsPage 9 分类/MemoryHubPage 8 库等）
  featureModel?: string // FeatureModelBar 的 feature 键（"事实上的板块注册表"，gaea/office 二义需消歧）
  bindings?: string[]   // 绑定门面归属（Go 侧同源声明）
  intents?: string[]    // 模块注册表意图（Go 侧同源声明）
  tools?: string[]      // 工具集过滤（板块级工具集声明）
}
```
示例：
```json
{
  "id": "novel",
  "name": "小说",
  "icon": "ReadOutlined",
  "page": "pages/novel/NovelPage",
  "lazy": true, "keepAlive": true, "layout": "padded",
  "nav": { "order": 3, "shortcut": "ctrl+3", "children": ["chapter", "character", "create", "export"] },
  "bindings": ["NovelB"],
  "intents": [{ "id": "create_chapter", "handler": "CreateChapter" }],
  "tools": [],
  "featureModel": "novel"
}
```
- **工具面板块化**（06 报告）：Tool 接口增加可选 BoardTags()/Category() 分类元数据；板块 manifest 声明工具集 = {tools: 启用子集, hide: 隐藏子集, extras: 注入工具, systemPrompt: 提示片段}；装配复用现有 FilterRegistry（task.go:256）+ FilteredSchemas（tool.go:281）+ HideUnlessOnly（tool.go:158）三件套；顺带消除"工具名字符串硬编码分类 5 处"（execute_one.go:176/336/442 等，收敛为接口能力）。
- **Go 侧**：新增 internal/app/board/ 包：Board 接口（ID/Name/Icon/Page()/Bindings()/Intents()/Tools() + Init(a *App) error）；builtins.go 注册 canonical 9 板块（chat/novel/imagegen/gaea/memoryhub/modelcenter/characterlib/settings/weixin，见 §3.1 清单）；module_registry.go 重构为由 manifest 填充，Dispatch 前先校验（缺 handler 的 intent 启动即报错，杜绝缺陷 2 复发）；gen_bindings 升级：NewBindings/initModules 双点改由 manifest 驱动（gen_bindings 的五级映射规则降级为默认推断，manifest 显式声明优先）+ 完整性测试（现有 bindings_completeness_test.go 反射兜底继续保留）。
- **层间覆盖**（09 报告借用 DSH patch 概念）：manifest 提供默认配置，用户/部署层按 id 整块覆盖（map[id]json.RawMessage，不合并字段），后写者胜。
- **前端侧**：新增绑定 GetBoardManifests() []BoardManifest（挂 CoreB）；MainLayout 改为清单驱动——菜单/快捷键/页面映射全部由数据生成，pageComponents 改为 PageRegistry: Record<pageKey, LazyComponent>（在 main.tsx 集中注册）；ModuleLauncher 同样从清单读取（LauncherTarget 类型放宽为 string）。
- **加板块的新流程**（目标态）：(1) 写 manifest（或 Go 侧 board.Register）；(2) 页面组件注册进 PageRegistry；(3) 新增绑定方法 → 重跑 gen_bindings；(4) 启动自检通过。——不再触碰 MainLayout/ModuleLauncher。

**验收标准**：无 manifest 信息遗漏时，导航/快捷键/启动器渲染与现状一致（像素级回归）；人为制造"intent 无 handler"时启动报错而非静默；新增一个空壳板块仅需 manifest + 一个页面文件。

### 5.3 Provider Seam（Step 3）

**现状**：internal/modelengine 是引擎枚举 + EngineConfig 的注册表（已有雏形）；图片后端在 gaea_tools.go/ai 里硬编码 switch；voice/ocr/tts 各自为政。

**设计**：seam 三元组（定义 / 提供者 / 消费者）：

| Seam | 接口（定义） | 提供者（现状 → 目标） | 消费者 |
|---|---|---|---|
| LLM | LLMProvider{ Chat, Stream } | bridge provider（注册 kind="wubigrok"，bridge.go:159）已存在 → 正式化注册表 | 各板块 chat、办公 agent |
| Image | ImageProvider{ Generate } | ai/image_openai.go、image_comfyui.go（已按后端分文件）→ 注册表 + 配置选择 | 绘梦板块、image_gen 工具 |
| OCR | OCRProvider{ Extract } | herdsman/paddle/mineru switch → 注册表 | 办公 docmd |
| Voice | TTSProvider/ASRProvider | voice/tts/whisper → 注册表 | 轻语、语音设置 |

- 注册表统一形态：providers.Register(kind string, p Provider) + config 字段（如 image.backend: "comfyui"）驱动选择；调用方只依赖接口。
- **目标清单扩展**（08 报告 app 层 31 处 + 06 报告引擎层 8 处，详见评审报告）：08 已列 LLM/Image/Voice·TTS·ASR/OCR/分类；06 补：websearch 6 引擎硬编码扇出（websearch.go:169-198）、embed/rerank 环境变量绑死（cost_tools.go:190-251）、vision 端点硬编码（vision.go:21-35）、markitdown 两级回退（readfile.go:246-282）、余额查询只认 DeepSeek 形状（billing/balance.go:34-43）；LLM provider 注册表（provider.go:326-361）是现成正向先例。
- **seam 三纪律（09 报告照抄 DSH）**：定义含事件词汇；提供者互斥注册；不可用即 fail-closed（拒绝运行而非静默降级）。
- **会话级策略作为日志事件折叠**（09 新增）：sandbox 模式/审批策略等会话级配置入日志（header 事件），恢复时从日志折叠——与 §5.1"回放即真相"衔接。
- 与模型中心的关系：modelengine.Manager 保留为"引擎清单 + 凭据"层，seam 在其上做"能力路由"。

**验收标准**：切换生图后端/OCR 引擎/TTS 引擎只改配置项，代码零改动；现有按引擎分支的行为以测试固化。

### 5.4 合成根统一（Step 2 配套）

- boot.Build（办公引擎合成根）保持不动；新增**板块装配阶段**（app.Startup 内）：loadManifests → 校验 → init each board → 组装 bindings。
- 装配顺序写入 app.go Startup 注释（对齐 DSH "layers apply in order" 的理念），保证可审计。
- 板块间通信维持现状（App 协调层），**不**在 3.0 引入板块级事件总线（列入远期 Step 4，见 §10 开放问题 D4）。

---

## 6. 分阶段实施计划

### Step 0 —— 修债（0.5 天，可搭阶段 7 任一刀发布）

- [ ] initModules 补注册 office 模块（按 D8 决策实现；协议文档指明 office.create→ProposalCreate，但该方法不存在）
- [ ] main_brain_chat_test 补断言：办公意图不再静默跳过
- [ ] 版本常量对齐（app_info.go AppVersion 与 wails.json/versioninfo.rc 一致）
- 验收：主脑输入"写一份标书"返回 office 模块输出；回归通过。
- 风险：无（纯补注册 + 测试）。
- 细化清单（D8 已决策 b）：(1) office 模块 Intents: ["create"]，Handle 路由现成 GaeaSend（gaea_handler.go:283），不实现不存在的 ProposalCreate；(2) 补 MainBrainChat 全链路测试（现状只测 classify 不测派发）；(3) Dispatch unknown module 分支补 slog.Error；(4) 版本源三处（app_info.go:11 AppVersion / wails.json productVersion / versioninfo.rc）同步并进闸门脚本。

### Step 1 —— 会话事件日志（3-5 天）

- [ ] agent/session：日志写入器（append-only、原子追加、seq/ts）
- [ ] 事件种类与 event.Kind 映射；projectMessages 投影层
- [ ] 检查点 + 压缩协议（压缩后 checkpoint，不再改写日志）
- [ ] 旧格式加载迁移路径 + 保留旧文件
- [ ] 派生 API：标题/统计/成本（日志重放）
- [ ] 回归：恢复/压缩/回退/归档全流程测试
- [ ] **GaeaHistory 黄金测试**（现状缺口：仅 gaea_projects_test.go 有引用性断言，无输出钉死测试）——改造前先录 golden fixture，改造后逐字节比对
- 验收：见 §5.1。风险：压缩与回放一致性（用现有 session_test 全套兜底）；高。
- 前端配合面（03 报告）：GaeaHistory 输出逐字节不变（红线）+ 黄金测试；统计/需求/变更 key 从会话文件路径改稳定会话 id（App.tsx:506-514）；死绑定 GaeaWorkspaceChanges 删除或改日志派生；事件丢包自愈三件套（看门狗 30s/reconcileFinalAnswer/localCancel）保留 + 补测试。

### Step 2 —— 板块 Manifest（3-5 天）

- [ ] board/ 包 + canonical 9 板块 manifest 落地（含 weixin 服务板块、knowledge 归属决策 D7）
- [ ] module_registry manifest 驱动 + 完整性启动断言
- [ ] gen_bindings 升级（manifest → 门面清单）
- [ ] GetBoardManifests 绑定；MainLayout/ModuleLauncher 清单化
- [ ] 前端 PageRegistry 注册表
- 验收：见 §5.2。风险：前端导航回归（用现有页面快照测试兜底）；中。
- Go 侧细化（04 报告）：board/ 包 + 9 板块 manifest；启动自检（intent handler 存在性/page 键唯一性/绑定门面存在性，失败显式报错退出）；每板块 Init(a *App) 只做装配不写业务。
- 前端细化（01/02 报告）：MainLayout 12 硬编码点全部收敛（映射表见本文附 B）；PageRegistry 在 main.tsx 集中注册；ModuleLauncher 同步清单化（顺带补 memoryhub/characterlib 缺失入口）。

### Step 3 —— Provider Seam（2-4 天，按 08 报告子步排序）

- [ ] 3a Image seam（xai/herdsman/ollama/comfyui 注册化——消灭 4 处重复 switch，纯重构收益最大）
- [ ] 3b LLM seam（桥接 provider 正式化：chat 路由/凭据/默认模型/能力标志进注册表；认证策略按 provider 隔离）
- [ ] 3c OCR/ASR/TTS seam（先接口化 asr/tts，行为用现有 *_test 固化）
- [ ] 3d 分类统一（classifyModelKind 等 4 处关键词分类收敛）+ 配置消费面统一（三份配置同一语义）+ 06 引擎层 8 处（websearch/embed/rerank/vision/ocr/markitdown/billing）
- 验收：见 §5.3。风险：各后端行为差异（用现有 *_test 固化）；中。

### 6.6 回退保障设计（用户要求：现行版本随时可回退）

**目标**：任何 3.0 改动落地后若出问题，可一键回到改动前的可运行状态——代码、数据、运行行为三层不丢不坏。

**L1 代码层（git）**：
- 每 Step 一个独立提交（不混改动，提交信息带 Step 编号）；阶段 7 每刀照旧 tag；main 单线，3.0 不开长分支（避免合并漂移）；
- 回退路径 = git revert <step-commit>；Step 2 前端大改保留旧实现一个版本（pageComponents 并行存在），revert 无需数据迁移。

**L2 数据层（用户数据）**：
- Step 1 事件日志：旧会话 *.jsonl 一律只读（新日志写 .gaea-log.jsonl 新文件）；legacy 读取路径保留至 3.1；切回 legacy = 行为等同改造前；
- 配置：3.0 新增配置键全部可选、缺省即旧行为，不迁移用户现有配置；
- 数据库：Step 0-3 不涉及任何 schema 迁移；未来若必须迁移，先备份 + 提供降级路径（对齐既有 backup.ps1 / GaeaDataBackup）。

**L3 发布层（二进制）**：
- 沿用阶段 7 纪律：每刀/每 Step 独立发布（exe + SHA256SUMS + 冒烟 + 发布说明）；releases/ 保留最近 5 版；
- 回退 = 覆盖安装上一版 exe（数据兼容由 L2 保证）；版本源三处（versioninfo.rc/wails.json/app_info.go）随发布同步。

**L4 运行时层（feature flags）**：
- 现有：session.log_format（Step 1）；
- 计划新增：manifest 导航开关（Step 2，legacy 导航模式保留到验收通过）、provider seam fallback（Step 3，配置缺失回退旧 switch 路径一个版本）；
- 原则：每个"新机制替换旧机制"的 Step 都带配置级回退开关——默认新机制、可切旧机制。

**每 Step 验收新增「回退演练」**（Step 0 建立脚本，后续复用）：
- 装新版 → 跑关键路径 → 切回旧版/旧开关 → 验证数据完好（会话可恢复、配置不丢、导航正常）；
- Step 1 专项：新日志写过的会话切回 legacy 后 GaeaHistory 逐字节一致（复用黄金测试）。

### 3.0.0 发布条件

- Step 0-3 全部落地且验收通过；
- 旧会话/旧配置迁移路径实测通过；
- 回归套件（go test ./... + 前端 vitest）全绿；
- 一键回退开关验证通过（§7）。

---

## 7. 兼容与迁移策略

| 对象 | 策略 |
|---|---|
| 旧会话文件（*.jsonl） | 读取兼容，首次保存转新日志，旧文件保留至 3.1 |
| 用户配置（三份割裂：~/.gaea_config.json（应用层 JSON）+ engines.json（模型中心）+ gaea.toml（办公引擎 TOML）） | 格式不变；新增字段全可选；seam 层统一消费面（三份表达同一"引擎选择"语义的现状记入 Step 3d 收敛） |
| Wails 绑定面 | 门面名与方法名不变（wailsjsCompat 兼容层继续有效） |
| 前端页面 | 组件不变，仅导航装配改数据驱动 |
| httpbridge | 不变（调试桥） |
| 回退 | 每个 Step 独立提交 + 发布说明；3.0.0 前任意 Step 可 revert |

---

## 8. 版本与发布计划

- **前置：阶段 7 正确性纵深（v2.34.0–v2.37.0）**：2026-08-14 定稿的阶段 7 计划（docs/superpowers/plans/2026-08-14-gaea长期规划-阶段7-正确性纵深.md）四刀（T7-1 并发正确性 → T7-2 可见性收口 → T7-3 名实相符 → T7-4 前端性能收尾）先行，不改其节奏；其依据是 .gaea/reviews/ 下 5 份既有审查报告（backend-core/frontend/model-cost/office-memory/whisper-chat），本设计的 9 份分域评审报告与之互补（阶段 7 报告定位"正确性短板"，本设计报告定位"架构改造面"）。
- **3.0 架构主线启动时机**：v2.37.0 发布后启动 Step 0-3。理由（文件级冲突分析）：Step 3（Provider Seam）与 T7-1.4（internal/ai/client.go 加锁/重试）同文件，Step 2（app 层 manifest）与 T7-2（gaea_*.go 吞错收口）同文件——先正确性后架构，避免返工；Step 1（internal/gaea/agent/session 事件日志）与 T7 无文件冲突，若带宽允许可与阶段 7 并行开发（独立分支）。
- **Step 0 例外**：office 模块补注册 + 版本常量对齐是小改，可搭阶段 7 任一刀的车（piggyback），无需等 3.0。
- **2.x 线**：阶段 7 四刀 v2.34.0–v2.37.0 照常独立发布；3.0.0 = Step 0-3 全部落地后的首个发布（预计阶段 7 收官后 2-3 周）。
- **3.0.0**：Step 0-3 全部落地后的首个发布；CHANGELOG 以「架构主线：事件日志 + 板块 Manifest + Provider Seam」为主标题。
- 版本号维护点：versioninfo.rc（FILEVERSION/PRODUCTVERSION）、wails.json productVersion、internal/app/app_info.go AppVersion —— 三处同步（Step 0 顺手脚本化）。

---

## 9. 搁置项：编程板块

> **用户指示（2026-08-15）：编程板块以后再说。** 原"二期试点"规划整体移出当前范围，不做设计预留；Manifest 机制的验证试点改由知识库板块（KnowledgePage 现成页面）与 chat 板块迁移承担（见 §6 Step 2 与愿景规划 §5.3）。若未来重启，所需地基（事件日志/Manifest/工具集过滤三件套）已建成，可直接复用。

## 10. 开放问题与决策记录（定稿状态）

| 编号 | 问题 | 状态 | 说明 |
|---|---|---|---|
| D1 | 3.0.0 发布时机 | **✅ 已确认** | 随 V3：3.0.0 地基 → 3.1.0 板块生态 → 3.2.0 受控自主 → 3.3.0+ 身份 |
| D2 | manifest 格式 | **✅ 已确认** | JSON（前端零依赖）；用户侧配置继续 TOML |
| D3 | 事件日志文件位置 | **✅ 已确认** | <workspace>/.gaea/sessions/<id>.gaea-log.jsonl |
| D4 | 板块间事件总线 | **✅ 已定：不做** | 单窗口编排方向已废弃；3.0 不引入板块级事件总线，板块间维持 App 协调层 |
| D5 | 旧会话文件清理时机 | **✅ 已确认** | 3.1 发布时删 3.0 遗留旧文件（此前一律只读保留） |
| D6 | 事件日志回退开关 | **✅ 已确认** | 默认开启 + session.log_format="legacy|event" 回退（见 §6.6 回退保障设计 L4） |
| D7 | KnowledgePage 孤儿页归属 | **✅ 已确认** | 恢复挂载为独立板块 = 3.0.0 最小 manifest 试点（随 V2） |
| D8 | office 模块修复方案 | **✅ 已决策 (b)** | office.create 路由 GaeaSend + MainBrainChat 全链路测试 |
| D9 | 5 个死 seam + 3 处死代码 | **✅ 已确认** | Step 1 接线 AuditFunc（日志即审计）与 CtxMgr 收敛；其余删除 |

**全部决策已随 V1-V8 确认（2026-08-15）**：与愿景文档 §7 的 V1-V8 为同一批决策，两表互相印证。
## 附：与 DSH 架构的对应关系

| DSH 概念 | gaea 3.0 对应 |
|---|---|
| SessionEventMap + deriveMessages（日志=事实源） | §5.1 事件日志 + 投影 |
| cordis.patch.yml 声明式装配 | §5.2 Board Manifest |
| capability seam（定义/提供者/消费者） | §5.3 Provider Seam |
| app-boot 合成根 | §5.4 板块装配阶段 + 既有 boot.Build |
| 无特权核心（"There is no privileged core to patch"） | 板块能力全部走 manifest + 接口，核心只做装配与协调 |
| client-connection（UI/内核分离） | 保持 httpbridge 调试桥定位（3.0 非目标） |

## 附 B：MainLayout 12 个硬编码点的收敛映射（01 报告 §5.1）

| # | 现状硬编码点 | 收敛后 | 备注 |
|---|---|---|---|
| 1 | Page 类型字面量联合（MainLayout.tsx:27） | manifest.id 派生（类型级断言锁死） | 复用 bridge.ts:880-939 AssertNever 模式 |
| 2 | allPageKeys（:30） | manifest 派生导航白名单 | navigate 校验改查 manifest |
| 3 | React.lazy 导入（:17-24） | PageRegistry 统一 lazy 包装 | 页面组件在 main.tsx 集中注册 |
| 4 | menuItems（:33-42） | manifest.filter(inMenu).sort(menuOrder) 派生 | 文案/图标/顺序全进 manifest |
| 5 | pageComponents（:44-53） | PageRegistry[id] 查找 | 与 lazy 注册合一 |
| 6 | Ctrl+1~4（:284-287） | manifest.shortcut 显式声明 | 修掉"菜单重排悄悄改快捷键"缺陷 |
| 7 | pageLabels（:208-210） | manifest.label | 面包屑直接取 |
| 8 | 面包屑"项目名→novel"（:415-417） | manifest.breadcrumb.anchorTo | novel 声明自己是项目锚点 |
| 9 | Content 布局特判（:424-432） | manifest.layout | chat/gaea=full，其余 padded |
| 10 | visitedPages 初始 home（:223） | manifest.isHome | 启动页语义进声明 |
| 11 | home 特判分支（:445-447） | manifest.isHome 渲染 ModuleLauncher | 启动器仍属壳层但由声明驱动 |
| 12 | settings 隐式入口（:383-387） | manifest.inMenu=false + 壳层保留按钮 | 入口收敛到 manifest 语义 |

迁移策略：PageRegistry 与旧 pageComponents 并行一个版本（过渡期）；main.tsx 按现 8 页逐一 registerPage；回归以"导航像素级不变"验收。事件常量表（21 后端 + 4 前端事件名）另建 src/events.ts 常量 + subscribe() 统一封装（细节见 01 报告 §4）。

