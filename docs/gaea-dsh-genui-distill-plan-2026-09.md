# gaea 蒸馏 dsh-genui 完整规划：办公 / 聊天板块「回答即 UI」+ 办公会话面板

> 状态：规划 v2 完整稿；执行进度 = **P0–P4 已完成，P5 收口并发布 v4.97.0**。
> 真机视觉/真模型端到端待用户复验（发布说明如实记录）。
> 执行记录：2026-09-05 落地 frontend/src/genui 内核 + 双板块 markdown 缝（P1/P2）；
> P3 新增 internal/gaea/genui（Handbook/ChatRule/OfficePointer embed + ValidateSpec）+
> genui_validate 工具 + genui inline 技能 + 办公/聊天/人格提示词接线；
> P4 新增办公右栏「UI」面板（genuiPanel store + GenuiPanel + sidebarRegistry/workspaceTabs
> 注册 + /panel 命令 + Markdown panel 投递/chip）。门禁：tsc/eslint 0，
> vitest 241 文件/1860 用例全绿；前端生产构建通过；Go genui/skill/tool/builtin/
> boot/app/whisper 全绿；办公验收走查（内嵌 UI→action→面板→重放）以 jsdom 自动化覆盖。
> 面板规格在 P4 起由 chip + 投递右栏承载（不再内联）。
> 日期：2026-09-05 · gaea 基线 v4.95.0（绑定面 579）
> 上游：github.com/omdsh-dev/dsh-genui（MIT）· 快照 commit `680693e`（2026-09-04）
> 上游调研方式：只读克隆至临时目录逐文件通读（SKILL.md / spec.ts / guard.ts /
> blocks / panel-store / plugin tools / docs plans 等）；gaea 侧现状按实际代码逐缝核对。

---

## 0. 摘要

把 dsh-genui 的**可搬运行为契约**（不是宿主插件机制）蒸馏进 gaea：

1. 模型可在回答正文中写 ```` ```genui ```` JSON 围栏，聊天与办公两个板块把它渲染成
   白名单可交互组件（stat/table/chart/card/表单/Quiz/…），文字照常穿插前后；
2. 交互**本地优先**：排序、判卷、重置、展开/收起零模型往返；只有带 `action` 的组件
   回传模型，无 action 的按钮诚实禁用；
3. 办公板块增加常驻「UI 面板」右栏工作台：模型用 `panel:true`/`append:true` 规格原地
   更新同一块会话级面板；
4. 状态按「会话 + 消息 + 内容指纹」持久化，历史重放原样恢复；安全按 gaea 自有策略
   （白名单渲染 + 有界规格 + 秘密禁令 + 既有外链/文件策略）收紧。

实现原则：**共享渲染内核 + 双板块三处 markdown 缝 + 模型侧技能/提示词**；不加宿主插件、
不加新 Wails 绑定面（P1–P3 绑定面保持 579），前端与 Go 分层各自可测。

---

## 1. 上游能力面（dsh-genui 行为契约快照）

### 1.1 是什么

dsh-genui 是 dsh 的浏览器侧插件：模型在回答里输出 ```` ```dsh-ui ```` 围栏（JSON 规格），
渲染器把围栏变成真实组件。它同时支持三条渲染通道：回答内围栏（answer-as-UI）、
`render_ui` 工具卡（工具行 UI）、会话面板（dock，模型 REPLACE/APPEND 原地更新）。

对 gaea 而言，宿主机制（fence-registry、DOM 观察、插件激活、资产懒加载路由、面板 dock 壳）
全部不需要；需要的是**语言、守卫、交互协议、状态持久化语义**这四件事。

### 1.2 组件词汇（上游全表）

规格根：`{"title"?, "gap"?, "panel"?, "append"?, "items":[节点]}`

| 族 | type | 行为要点 |
|---|---|---|
| 布局 | text / row / col / grid / card / divider / spacer | grid≤12 列；行/列可 wrap；递归渲染 |
| 展示 | stat / badge / progress / list / table / keyvalue / avatar / breadcrumb / timeline / callout / steps | stat 的 `-` 前缀红、`+` 前缀绿；table 表头本地排序、数值感知（千分位/k/m/b/万/亿/%/货币）；callout 有 info/success/warning/error；progress 0–100 |
| 图表 | chart（bars/line/donut）/ echart / plot | chart≤60 点、可多 series；echart 全功能 ECharts（option 深≤10、数组≤500、遍历预算 2000）；plot 数学函数图 + 参数滑块 + 动画，表达式走独立解析器 |
| 代码展示 | code / json / diff | code≤12k 字符；json 树查看器；diff 逐文件 old/new 文本 |
| 媒体 | image / audio / video | 只接受 http(s)/同源相对 URL；懒加载、失败态；无自动播放 |
| 内容图 | mermaid / diagram / scene3d | mermaid 自动修复降级；diagram 编辑级 27 类品牌图（节点≤9/边≤12/焦点≤2）；scene3d WebGL（mesh 1–5） |
| 交互 | button / input / select / checkbox / radio / switch / slider / textarea / tabs / accordion / copy / submit | 见 §1.3 |
| 教学 | quiz | 点选即判、可重试、带 action 可回传对错 |

### 1.3 交互语义（上游精确行为）

- **诚实交互**：按钮/开关/输入等必须带 `action` 才可点；无 action 渲染为禁用。带 action
  的点击后立即显示「已触发」本地反馈（证明事件已发，不代表模型已收到）。
- **防抖**：同名 action 300ms trailing 合并，快速连点只发最后一次（last value wins）。
- **输入**：input 回车即提交（submit 语义）、textarea Ctrl/Cmd+Enter 提交；blur 仅值有
  变化才发（聚焦又离开零往返）；带 `id` 的字段值跨刷新保留并进入 submit 的 `fields`。
- **radio 聚合**：多个 radio 共用 `group` 时选择只本地记录；题带 `answer`+`explanation`
  时交卷**本地判卷**（得分、逐题 ✓/✗、解析、锁定），「重新作答」本地重置
  （`resetAction` 可选通知模型）；仅当题目都没带 answer 时，submit 才聚合成一次 action
  （payload 含 `answers/fields/total/answered`），`groups` 未答完不可提交。
- **卷子纪律**：多题用「每题 radio + 末尾一个 submit」，不要每题单独发 action。
- **秘密禁令**：password 输入蒙层显示、值不持久化、不进入 submit fields；模型侧禁止
  索取/生成密码、API Key、Token、恢复码。
- **action 事件**：`(action, payload)` 回传宿主；payload 依控件带值/选项/对错。模型
  根据 action 更新 UI 或走下一步。

### 1.4 守卫 / 修复 / 上限（上游权威值）

```
maxDepth 8 · maxNodes 200 · maxString 2000 · maxCode 12000 · maxMermaid 8000
maxGridCols 12 · maxTabs 12 · maxAccordionItems 24 · maxListItems 50 · maxOptions 50
maxTableRows 50 · maxTableCols 12 · maxChartPoints 60 · maxPlotSeries 8 · maxPlotParams 6
maxMeshes 5 · maxQuizOptions 8 · maxSteps 24 · maxTimelineItems 24
maxBreadcrumbItems 12 · maxKeyValuePairs 24 · maxTreeDepth 6
maxDiagramNodes 9 · maxDiagramEdges 12 · maxDiagramZones 3 · maxDiagramFocal 2
maxEChartOptionDepth 10 · maxEChartArrayLen 500 · maxEChartOptionNodes 2000
```

- 修复策略：已知 type 的字段类型错→整节点丢弃；数字 clamp 到合法域；整数域拒绝非整数；
  字符串截断；数组按 cap 截断；容器按 depth 递归；节点预算耗尽后同层其余兄弟丢弃。
- 上游对「未知 type」的策略是**透传**（给插件扩展留缝）；gaea 无插件生态，改采
  **白名单外一律丢弃**（更严，见 §7）。
- `validateGenuiSpec` 返回 `{ok, errors[]}`；另有 `validateGenuiChartSemantics`（chart 的
  kind/字段语义）与 `repairGenuiSpec`（幂等、前缀稳定：流式 chunk 已修复的组件在其后
  chunk 到达时位置不变）。
- 坏 JSON：前端只做「字符串内半角引号、尾随逗号」标点级修复；缺括号等结构性错误不猜，
  红横幅退化代码块（gaea 采用同策略）。

### 1.5 流式部分渲染

`parsePartialGenuiSpec`：先完整 parse（settled 常态）；失败后做**一次**左到右扫描收集
括号平衡前缀候选（预算 32 个、环缓冲只留最长、跳过字符串与转义），按最长优先试 parse；
单组件 root（裸 `{"type":"callout",…}`）自动包成 col。结果永远只是规格的**前缀**，
所以已渲染的组件都是完整合法的。

### 1.6 状态持久化

- `BlockInteractionState = {answers?, locked?, fields?}`；
- key = `f|p|t` 前缀 + 会话 id + 槽位（fence 序号 / panel publish / tool callId）+
  内容指纹（djb2）；同一内容重放恢复状态、新内容自动清零；
- localStorage LRU 200 条；写入 300ms 防抖；password 值在写盘前剥离；
- 同内容重渲染时组件**不重挂**（React key = durable/volatile 分区）。

### 1.7 会话面板与 render_ui 工具

- `panel:true` 的 fence/工具结果投递到会话 dock；`append:true` 合并增量（同名 tab 追加、
  新 tab 增加、普通 items 尾接）；**append 只在围栏完整时生效**（部分流不重复 merge）；
- panel-store 操作按 `sourceId + order`（消息 seq）排序去重：旧结果永不覆盖新结果；
  面板整体上限 200 节点/200 次 append，超限提示模型用 replace 重建；
- `/panel` 打开 dock；`/panel <指令>` 让模型定制；`/panel clear` 清空；
- `render_ui` 工具：模型把完整 spec 当参数调用，校验/修复后把 spec 放进 tool result 的
  `meta`（presentationMeta），工具卡用 callId 做状态 key，settle 后**同时**发布到面板；
  running 期间不发布（面板保留上次内容）。

### 1.8 安全模型

白名单节点 → React 元素直接渲染，无 raw HTML 路径；函数表达式走独立解析器不 eval；
媒体只允许 http(s)/同源相对；字符串转义由 React 完成；规格有界；持久化只存交互值。

### 1.9 上游测试面（可借鉴边界清单）

guard（超限/坏节点/自动修复/echart option 消毒）、partial parse 预算与环形缓冲、
dom-fence 挂载/卸载/RAF 清理、panel replace/append 幂等、交互防抖、本地判卷与锁定、
quiz、file-tree 折叠 key、mermaid 修复、错误边界降级、媒体失败态、图表契约。

---

## 2. gaea 现状地图（代码摸底）

### 2.1 板块与页面

| 板块 | manifest id | 页面 | 视觉栈 |
|---|---|---|---|
| 聊天 | chat | frontend/src/pages/ChatPage.tsx（plain + 轻语人格） | antd + `--md-sys-*`/`--color-*` token |
| 办公 | gaea | frontend/src/pages/GaeaPage.tsx → frontend/src/gaea/App.tsx | gaeaW 自绘组件 + tailwind 语义类 |

### 2.2 聊天板块关键链路

- 发送：ChatPage → `useChatStream.send` → plain 模式 `App.ChatStreamPlain(topicId,…)`
  订阅 `chat-stream:<runID>`（delta/reasoning/done/error，30s 无帧超时）；人格模式走
  `ChatCharacter`/`ChatSend(mode!=plain)` 整段返回 + 模拟打字。
- 渲染：ChatRow 内——流式与人格用 `MarkdownContent`（frontend/src/components/MarkdownContent.tsx，
  纯 react-markdown、无 components prop、无代码块语言处理）；plain 终态用
  `ChatMarkdown`（components/ChatMarkdown.tsx，自带 code 块头 + 复制按钮）。
- 后端：`internal/app/chat_service.go`：`chatPlainSystemPrompt` 常量；`ChatStreamPlain`
  （runID 异步流）；人格经 `chatSendPersona → WhisperChat(WithSearch) → whisper
  Orchestrator`。人格系统提示词装配链入口可锚定 `internal/whisper/main_chat_prompt.go`
  `BuildMainChatSystemPrompt`（另有人格 preset / psyche / 记忆/情绪注入，实施时需确认
  该函数是否为所有人格的必经点；若是 custom 人格自带 system prompt 的路径，需另做分支）。
- 持久化：Topic/Message 存原始 markdown 文本（assistant 回复含围栏也会原文落库）；
  `ChatTopicExportMarkdown` 原样导出。
- action 限制：聊天**没有运行中 steer**；assistant 流式期间不可注入，带 action 组件应
  在回合结束后触发一次新话题内发送（复用 send，sending 锁保证不并发）。

### 2.3 办公板块关键链路

- 布局：左 Sidebar（会话）+ 中间 Transcript（消息流）/ChatTabs（对话/轨迹/上下文/记忆）+
  右侧工作台（一级 Tab：文件/产物/任务/浏览器/Git）+ 底部 Composer；右栏 Tab 清单
  `frontend/src/gaea/lib/workspaceTabs.ts` 单一数据源、渲染接线 `lib/sidebarRegistry.ts`
  （v4.23 起注册制，v4.86 加 Git 是现成范本）。
- 消息流：store（`gaea/lib/store.ts` zustand）消费后端事件（text/message/reasoning/
  tool_dispatch/tool_result/…）组装 `Item`（user/assistant/phase/notice/tool/compaction）；
  Transcript 按轮分段，assistant 走 `AssistantMessage` → `MemoMarkdown`；
  `MemoMarkdown`（gaea/components/MemoMarkdown.tsx）流式用 `findStableCut` 把未闭合
  ``` 围栏留在 pending 段，稳定前缀交给 `Markdown.tsx` 全量渲染；`Markdown.tsx` 的
  `code` 组件已按 language 分派 mermaid（≈413–429 行）——genui 加同缝分支即可。
- 工具卡：`ToolCard.tsx` 按 `item.name` 分派图标/摘要（lib/tools.ts、tool_icons.ts），
  工具行位于对话流中；`DeliverableCards` 在轮尾按正文路径登记「产物」。
- 发送：store.send(displayText, submitText?) → `GaeaSend`/`GaeaSendDisplay`（空闲开新回合）；
  store.steer → `GaeaSteer`（运行中注入当前回合，后端对空闲态有 Submit 兜底语义）；
  approve/ask/answerQuestion 为既有结构化交互通道。
- 后端：办公 Agent 工具注册 `internal/gaea/tool/builtin/*.go`（init() 自注册，
  `tool.RegisterBuiltin`）；技能 `internal/gaea/skill`：`builtinSkills()` 注册内置技能，
  `Store.List/Read` 按 project/custom/global/builtin 优先级合并，`skill.ApplyIndex`
  把「名称+描述」索引折进系统提示词（≤4000 字，body 按需 run_skill 读取）；
  办公系统提示词装配链：config.ResolveSystemPrompt → outputstyle → LanguagePolicy →
  记忆/晨报/pins → `skill.ApplyIndex`（internal/gaea/boot/sysprompt.go buildSystemPrompt）。
- 会话面板候选宿主：右栏「UI 面板」新增一级 Tab 只改两处（workspaceTabs.ts +
  sidebarRegistry.ts RENDERERS），按钮条/命令面板/设置卡自动派生；面板数据可自取
  （对照 GitPanel/BrowserPanel 模式），会话隔离照 `gaea.rightPanel.v1:<sessionKey>` /
  paneTabs setSessionKey 模式。

### 2.4 可复用资产

- 办公 Markdown 已有 mermaid 渲染/导出、KaTeX、文件链接/预览、sanitize 消毒层；
- 办公已有轻量 SVG 图表先例（TrendChart.tsx），genui chart 可同口径手写；
- 办公 store 已具备运行中 steer 与空闲 send 双通道，action 无需新绑定；
- 办公右栏已有注册制面板体系；聊天无面板需求，不做。

### 2.5 需要新做的（gaea 空白）

1. genui 规格语言 TS 类型 + guard（全量，比上游更严的未知 type 策略）；
2. GenUI 组件库（约 24 个 type 的 v1 子集）与样式（只依赖全局 CSS 变量）；
3. 双板块 markdown 渲染缝 + 流式/终态/history 状态 key 通路；
4. action 宿主适配（办公 steer/send、聊天回合后发送；本地优先全部可用）；
5. 办公面板 store + UI 面板 Tab + /panel 命令；
6. 模型侧：genui inline 技能 + genui_validate 工具 + 双板块提示词规则。

---

## 3. 蒸馏范围裁定

### 3.1 v1 必收（本规划 P1–P5 全量）

| 上游能力 | gaea v1 落法 |
|---|---|
| 回答内围栏 UI | ```genui 围栏，聊天/办公均渲染 |
| 组件语言（布局/展示/轻图表/交互/Quiz） | 共享模块实现 |
| 本地优先 + action 协议 | 见 §5.4 |
| 守卫与修复 | 白名单丢弃 + 预算 + 标点级修复 + 红横幅降级 |
| 状态持久化 | 会话+消息+指纹，LRU 200 |
| 会话面板 | 办公右栏「UI 面板」Tab（REPLACE/APPEND、/panel） |
| render_ui 工具卡 | **v1 不做工具卡通道**，统一由回答围栏承载（§3.3 决策 3） |
| validate 工具 | genui_validate（结构化校验版，§5.8） |
| mermaid | 不进 genui 词汇，直接用 gaea 既有 mermaid 围栏 |

### 3.2 v2 后置候选（不进本规划执行表）

plot（函数图+滑块/动画）、echart 全功能、diagram 27 类、scene3d、image/audio/video
媒体组件、render_ui 工具卡 + callId 持久化、genui 输出模式记忆偏好（开关）。

### 3.3 决策默认值（待用户拍板，见 §9）

1. 围栏标签主用 `genui`，另兼容别名 `dsh-ui`（若拍板不兼容则只认 genui）；
2. 聊天范围默认 plain + 人格都接（统一注入点），纯闲聊门控不发 UI；
3. 办公 v1 只做回答围栏 + 面板围栏（零绑定面变更），render_ui 工具卡后置；
4. 媒体/本地产物图 v1 不含，P5 后按 gaea 本地文件体系另案；
5. genui 组件文案跟随内容语言（模型/用户输入），不铺三语字典；新增 chrome 标签按
   workspaceTabs 惯例中文即可。

---

## 4. 目标架构

### 4.1 模块划分

```
frontend/src/genui/            ← 共享渲染内核（两板块 import，零 antd/tailwind 依赖）
  spec.ts                      ← 类型 + 白名单 + GENUI_LIMITS（单一权威常量）
  guard.ts                     ← 校验/修复/计数（纯函数）
  parse.ts                     ← 围栏拆分 + 部分解析（预算 32、单次扫描）
  fingerprint.ts               ← djb2 指纹 + 状态 key 构造
  interaction-store.ts         ← 本地交互状态持久化（LRU 200）
  panel-store.ts               ← 办公面板内容 store（抽象，宿主注入实现）
  GenuiActionContext.tsx       ← action 提供者（缺省：纯展示，禁用 action）
  GenuiBlock.tsx               ← 渲染壳（banner + items 列 + reveal）
  blocks/                      ← layout / display / chart / code / forms / quiz
  markdownFence.tsx            ← react-markdown code 适配（供三处缝复用）
  GenuiTextSegments.tsx        ← 文本/围栏分段渲染（办公使用；聊天可选）
  styles.css                   ← 只用 --md-sys-*/--color-*/--gaea-glow/--v3-* 令牌

frontend/src/pages/chat/...    ← 聊天接入（不反向依赖 gaea/）
frontend/src/gaea/components/  ← 办公接入（Markdown/MemoMarkdown/新 GenuiPanel）
internal/gaea/skill/           ← genui 内置 inline 技能
internal/gaea/tool/builtin/    ← genui_validate 只读工具
internal/app/                  ← chatPlainSystemPrompt / whisper 提示词注入点
internal/gaea/genui/           ← （可选）Go 侧结构校验/上限常量（与 TS 同源注释）
```

### 4.2 数据流

```
模型输出 text(含 ```genui 围栏)
  → 后端照常事件流/落库（原文 markdown，零改动）
  → 前端 ChatRow / AssistantMessage 拿到 text
  → markdown 缝识别 code lang=genui
  → parse.ts 提取围栏体 → guard.ts 修复/校验
  → panel:true ? 发布 office panel-store（消息流内仅占位 chip）
              : GenuiBlock 原地渲染（stateKey = 板块:会话:消息:指纹）
  → 用户交互：本地状态 → interaction-store；action → GenuiActionContext
  → 宿主 action 通道：
     办公：running → steer(短文案) / 空闲 → send(展示文案, 信封原文)
     聊天：回合结束 → 新话题发送（[UI 操作] 文案）
  → 模型读到 action 描述 → 输出更新后的回答/面板规格
```

### 4.3 状态与作用域

- 交互状态 key：`genui:<chat|office>:<sessionId/topicId>:<msgKey/assistantId>#<fenceNo>:<fp>`；
- 面板内容 key：`genui:office:panel:<sessionKey>`（内容与交互分离存储）；
- 恢复会话：历史消息按序重放 → 围栏按序重渲染/重发布 → 终态与中断前一致；
  同 sourceId+指纹的重复发布幂等跳过。

---

## 5. 详细设计

### 5.1 spec 语言（gaea v1 白名单）

根：`{ "title"?, "gap"?, "panel"?, "append"?, "items": [...] }`

#### 布局

| type | 字段 | 约束 |
|---|---|---|
| text | content 必填；size=h1/h2/h3/body/muted/caption；center? | 纯文本渲染，无 markdown |
| row / col | items；wrap?；gap? | 递归 |
| grid | cols(1–12)；items | |
| card | title?；items | 实底卡 + 卡头 |
| divider / spacer | – | |

#### 展示

| type | 字段 | 行为 |
|---|---|---|
| stat | label；value；delta? | delta 前缀自动配色（+绿/−红） |
| badge | label；tone=success/warn/danger/accent | |
| progress | label?；value 0–100；valueLabel? | |
| keyvalue | pairs[{key,value}] ≤24 | |
| list | items ≤50（字符串或节点） | 行内可嵌节点 |
| table | columns；rows ≤50×12 | 表头点击三态本地排序；数值感知 |
| timeline | items ≤24（title/desc/time?） | |
| callout | tone=info/success/warning/error；title?；content | |
| steps | current?；steps ≤24（title/desc?） | |
| avatar | name；color?（hex 或预置色） | 首字头像 |

#### 轻图表（手写 SVG）

| type | 字段 | 说明 |
|---|---|---|
| chart | kind=bars/line/donut；data≤60；series? | 复用 TrendChart 口径实现；tooltip 用 <title> |

#### 代码展示

code / json / diff（json 树查看器 v1 可退化为 code 高亮展示；diff 复用 ChangesDiff
的纯 diff 数据形状但做独立只读展示，不接文件路径，杜绝路径副作用）。

#### 交互

统一规则：**无 action 的交互控件渲染为禁用/纯展示**；控件 action 触发交给
GenuiActionContext；同一 action 300ms trailing 防抖。

| type | 关键字段 | 本地行为 |
|---|---|---|
| button | label；tone；action? | 点击“已触发”反馈 |
| input | label/placeholder/value/inputType/text/email/password/id?；action? | Enter 提交、blur 仅值变才发；password 不持久化不进收集 |
| textarea | rows?；id?；action? | Ctrl/Cmd+Enter 提交 |
| select | options；selected?；id?；action? | |
| checkbox / switch | checked?；label；action? | |
| radio | options；selected?；group?；answer?；explanation?；action? | 有 group 本地聚合；有 answer 本地判卷 |
| slider | min/max/step/value；id?；action? | 数值实时显示，拖拽合并发送 |
| submit | label；action?；groups?；resetAction? | 全带 answer→本地判卷锁定；全无→聚合一次 action |
| tabs | tabs[{label,items}] ≤12 | 键盘方向键切换、Home/End |
| accordion | items ≤24 | |
| copy | label/text | |
| quiz | question/options(correct?+feedback?)/explanation/id?/action? | 点选即判、重试、可选回传 |

#### 媒体策略（v1 不含组件；若 P5 加 image 则）

图片源仅允许 gaea 外链策略放行项（http(s)/mailto/data:image 走既有 classifyExternalLink
白名单）与「本地产物绝对路径」——后者经 `@.gaea/attachments`/文件预览通道打开，**不**
使用 file:// 协议。

### 5.2 guard 与修复（gaea 版）

- 常量 `GENUI_LIMITS` 与 §1.4 对齐，减去 diagram/echart/plot 相关条目；
- 未知 type：**丢弃整节点**（与上游透传不同，gaea 无扩展生态；保留容器内其余兄弟）；
- 数字 clamp/整数拒绝、字符串截断、数组截断、深度预算、节点预算；
- 空 items 视为非法（退化代码块）；坏 JSON 只做标点级修复（半角引号/尾随逗号），
  结构性错误红横幅 + 原样代码块；
- `validateGenuiSpec` 输出 `{ok, errors[]}`，供工具与前端共用同一形状；
- 幂等 + 前缀稳定（流式 chunk 已存活组件位置不变）。

### 5.3 流式

- 聊天：plain/persona 流式期间维持现状渲染（不抢跑）；回复结束/最终帧后整段稳定，
  围栏一次性渲染。MemoMarkdown 的 findStableCut 天然把未闭合围栏留在 pending，
  **办公稳定前缀语义继续可用**：一条长回答中先闭合的围栏随后续正文 \n\n 切分逐段
  上屏（接近上游“边写边现”），只有最后一个未闭合围栏等收尾；
- 部分解析器（parse.ts）供 P5 可选增强：若需要“围栏还没闭合、内部组件先现”，对
  pending 尾部用 parsePartialGenuiSpec 渲染前缀组件，闭合后以同一 volatile 实例接续
  （React key 不换），避免闪烁；默认不启用，防过度设计。

### 5.4 交互与 action 协议

#### 本地优先

排序、判卷、重做、折叠、tabs/accordion 选择、radio 聚合、copy、重置：组件内部 state
完成，零网络。

#### action 信封（发送给模型的文本）

信封用「行首标记 + JSON body」，便于模型解析、便于历史可读：

```text
[genui-action]
{"type":"button","action":"refresh","source":"a3#0","panelKey":"main","payload":{"id":"order-panel"}}
```

约束：
- 长度上限 1200 字符、payload 逐字段类型/长度清洗、控制字符剔除；
- 不含秘密值（password 的 action 也只在用户显式提交时传值，且不落库）；
- 办公展示层用短文案（如 `（UI 操作）刷新「订单看板」`），信封原文走
  SubmitDisplay 的 raw 或运行中 steer；聊天无 raw/display 分离，直接把
  `[UI 操作] 组件「订单看板」的「刷新」被点击` 作为话题内消息发出（信封/文案二选一，
  建议聊天用可读文案 + 显式字段清单，提示词定义解析规则）。

#### 宿主接线

- 办公：运行中 `steer(actionText)`；空闲 `send(displayText, actionText)`；
- 聊天：仅在本回合结束后允许 action（sending 锁外），复用 send；
- 双端都提供 `GenuiActionProvider`（办公 App / ChatPage 消息区包一层），缺省无提供者
  时组件自动进入纯展示态（action 按钮禁用）——离线/旧消息预览永不假装可点。

### 5.5 渲染接入（精确改动面）

#### 办公

1. `gaea/components/Markdown.tsx` `code` 组件（413–429 区域）增加
   `language-genui`/`language-dsh-ui` 分支：文本经 parse+guard 后
   `GenuiBlock` 渲染；spec.panel=true 时渲染轻量占位 chip「已更新 UI 面板」（并发布
   面板，见 5.7）；
2. `MemoMarkdown.tsx` 增加可选 props `sourceKey?: string`、`onPanel?: (spec)=>void`，
  透传给 Markdown code 分支，保证 state key 与 panel 发布源 id 可用；流式稳定前缀
  每 chunk 渲染时 panel 发布按 sourceId+指纹幂等；
3. `AssistantMessage`（Message.tsx）把 `item.id` 作为 sourceKey 传入。

#### 聊天

1. `ChatMarkdown.tsx` code 组件加 genui 分支（保留现有代码块头）；
2. `MarkdownContent.tsx` 增加可选 `components?: Components`（默认不传 = 行为不变），
  或由 ChatRow 传入仅含 genui code 的覆盖组件；
3. `ChatRow` 消息区包 `GenuiActionProvider`（由 ChatPage/MessageList 注入 handler）；
  流式行不发 action（handler 内部判断 sending）。

#### 通用

- 围栏 stateKey 生成：聊天 = `genui:chat:<topicId>:<msgKey>#<fenceNo>:<fp>`；
  办公 = `genui:office:<sessionKey>:<assistantId>#<fenceNo>:<fp>`；
- fenceNo 为该条 assistant 消息内 genui 围栏序号（解析时确定，稳定）。

### 5.6 历史重放 / 导出 / 记忆

- 重放：消息原文含围栏 → 渲染/发布自动恢复；同指纹状态恢复；
- 导出：聊天 Markdown 导出原样保留围栏（在别处自然退化为代码块）；办公 Markdown/
  docx 导出走既有流水线，不做特殊处理，但 release notes 说明 UI 是交互态产物；
- 记忆/压缩：**待确认影响点**——办公“做梦”、上下文压缩读取 assistant 正文时会看到
  JSON 围栏原文；建议正文保留（模型需要回看），归档/记忆摘要抽取侧可跳过 genui 围栏
  体（P5 落地：在抽取函数前用 parse.ts 剥离，不进入 fact 文本）。

### 5.7 办公「UI 面板」

#### 面板 store（抽象 + 宿主实现）

```
状态：{ sourceId→{order,spec} 有向历史, content: 合并后的 Spec }
操作：
  publish(sessionKey, source:{msgId,fenceNo,seq?}, spec)   // panel:true，REPLACE
  append(sessionKey, source, spec)                          // append:true，仅完整围栏
  clear(sessionKey)
派生：面板总节点 ≤200；append 累计 ≤200 次后提示模型 replace；
隔离：按 currentSessionKey；切换会话读档恢复（localStorage，参照 paneTabs）。
```

- 顺序权威：后端没有新事件，面板按「消息重放顺序」重建即可收敛；同一条消息重复渲染
  （chunk 流式、React 重挂）用 sourceId+指纹幂等去抖；
- 交互状态：面板内组件 state key 用 `genui:office:panel:<sessionKey>:<contentFp>`
  （内容变则重置，对齐上游 panel publish key 语义；如需细粒度保留列为 P5 候选）。

#### UI 面板 Tab

1. `workspaceTabs.ts`：`WORKSPACE_TAB_IDS` 追加 `"ui"`，清单加 `{id:"ui", label:"UI 面板",
   icon:…, keywords:[…], defaultEnabled:true}`（Git 先例 v4.86）；
2. `sidebarRegistry.ts`：RENDERERS 加 `ui: () => <GenuiPanel/>`；
3. `GenuiPanel.tsx`：读 panel store + 空态（“模型生成的交互面板会显示在这里”）+ 头部
   清空按钮（两击确认）+ 内容即 `GenuiBlock`（复用同一渲染壳）；
4. badge：新面板内容更新时可在 Tab 上出角标（沿用 workspace 角标机制，P5 可选）。

#### /panel 命令

- `lib/command.ts` / CommandPalette 注册：`/panel` 打开右栏 UI 面板；
  `/panel clear` 清空；`/panel <指令>` 按既有“命令即发送文本”语义把指令发给模型
  （模型输出新面板规格）；
- 聊天板块不提供 /panel（无面板）。

### 5.8 模型侧接线

#### 办公

1. **内置 inline 技能 `genui`**（internal/gaea/skill/builtins.go 追加，Scope=builtin、
   RunAs=RunInline、AllowedTools 空）：Body = 裁切版词汇手册（§5.1 + JSON 自检四步 +
   数量纪律 3–8 组件 + 场景映射表 + action 语义 + secrets 禁令 + panel 用法），
   Name/Description 进入现有 Skills 索引（skill.ApplyIndex ≤4000 字自动截断）；
2. **系统提示词指针**（boot/sysprompt.go buildSystemPrompt 尾段或 skill index 之前追加
   ~300 字）：回答命中数据/对比/选项/流程等结构化呈现时主动使用 genui 围栏；先用
   `run_skill({name:"genui"})` 取词汇；复杂规格先 `genui_validate`；纯文字不硬塞；
3. **genui_validate 只读工具**（internal/gaea/tool/builtin/genui_validate.go，
   init() 注册、ReadOnly=true、CompactDescriptor 实现）：
   - schema：`{"spec":{"type":"string"}}`（围栏内 JSON 文本）；
   - Execute：encoding/json 语法解析 → 结构校验（根形状、type 白名单、预算/深度、
     明显类型错误）→ 返回 `✅ OK（N 节点）` 或 `❌ …（行/字段定位）`；
   - 声明“最终以渲染器为准”（Go 侧只做结构校验，不承诺与 TS guard 逐字节一致的
     修复）；上限常量以 §1.4 为准，Go/TS 两处注释同一来源；
4. **能力可见**：genui_validate 经工具注册自动进模型工具清单与 CapabilitiesPanel；
   前端 `tool_icons.ts` 加图标、`lib/tools.ts` summarize 可选加摘要分支（未加时走通用）。

#### 聊天

1. plain：`internal/app/chat_service.go` `chatPlainSystemPrompt` 常量追加「结构化呈现
   规则」块（目标 ≤1.2k 字，只给高价值组件 + 判据 + 围栏写法 + 自检四步 + 不发 UI
   的例外）；
2. 人格：在 `internal/whisper` 统一装配点追加同款规则（候选锚点
   `main_chat_prompt.go: BuildMainChatSystemPrompt` 结尾追加一段；实施首步验证 custom
   人格/psyche 是否会覆盖该 builder，若覆盖则在 whisper 编排器最终 SystemPrompt 组装
   处做末尾追加，保证所有模式必经）；
3. 门控条款：纯闲聊、一句话能说清、用户未要求 UI 时不发围栏；组件数量纪律 3–8；
4. 人格回复落库原文含围栏，聊天历史重放即可渲染，无额外后端工作。

#### 模型输出样例（skill Body 内置示范）

````text
回答正文…
```genui
{"title":"本月订单","items":[
  {"type":"stat","label":"营收","value":"¥128,430","delta":"+12.4%"},
  {"type":"chart","kind":"bars","data":[{"label":"1月","value":98},{"label":"2月","value":112}]}
]}
```
后续正文…
````

### 5.9 样式 / 主题 / 文案

- GenUI 组件零硬编码色值：只消费全局变量（`--color-primary/-container/-text/
  -border/-success/-warning/-destructive/-surface*`、`--gaea-glow`、`--v3-*`、
  `--md-sys-radius-*`），两板块同一套 tokens（App.tsx 顶层已统一注入）；
- 尺寸基线对齐 gaea 3.0 蓝图：卡内 16px、圆角 md(12)/lg(16)、正文 13–14px、
  `tabular-nums` 数字；reduced-motion 禁用 reveal 动画；
- 组件间文案尽量由模型内容决定；gaea 侧新增的固定 chrome 文案（占位 chip、
  面板空态、/panel 提示）进 gaea/locales（zh/zh-TW/en）既有键体系；
- 聊天与办公外观差异只通过 CSS 变量与上下文微调（如气泡内 max-width），共享组件
  不引入任何一方的私有类名。

### 5.10 安全模型（gaea 收紧版）

1. 白名单渲染：type 表之外一律丢弃；渲染走 React 元素，无 dangerouslySetInnerHTML
   （code/json 展示用受控 token 着色或纯文本）；
2. 无 eval/Function；无远程组件加载；
3. 预算硬上限（节点/深度/字符串/表/列表/选项/chart 点数）+ 解析前围栏体 64KB 上限；
4. 媒体/链接沿用 classifyExternalLink 与本地文件预览策略，genui 不开新协议例外；
5. 持久化只存交互值（answers/fields/locked），不存正文/spec；password 值不落库、
   不收集；LRU 200 防膨胀；
6. 对话内 action 只允许「steer/send 文本」两种出口，无新权限面；payload 清洗；
7. markdown 正文仍走既有 sanitize 消毒层；genui 组件渲染不与 HTML 白名单交错。

---

## 6. 后端变更清单（预估）

| 位置 | 变更 | 绑定面 |
|---|---|---|
| internal/gaea/skill/builtins.go | +genui inline 技能（Body 词汇手册） | 0 |
| internal/gaea/tool/builtin/genui_validate.go（新） | +只读校验工具 init 注册 | 0 |
| internal/gaea/boot/sysprompt.go | +办公输出纪律指针块 | 0 |
| internal/app/chat_service.go | +plain 聊天 UI 规则块 | 0 |
| internal/whisper/*（确认后单点） | +人格统一 UI 规则块 | 0 |
| internal/gaea/genui/（可选新包） | Go 侧结构校验函数 + 单测 | 0 |
| Go 测试 | genui_validate / skill 索引 / 提示词常量 | – |

> 说明：前端渲染、状态、面板全部走既有文本通道；不需要新增 WireEvent 类型、不需要
> wails 绑定方法，因此 P1–P3 绑定面保持 579 不变（drift PASS 继续成立）。

---

## 7. 分阶段执行计划（版本建议，最终以发布节奏为准）

### P0 · 词汇与接口定稿（可并入 P1 首日）

- 拍板 §9 决策点；冻结 v1 白名单与 GENUI_LIMITS；
- 产出：spec.ts 类型 + limits 常量 + 契约测试列表。

### P1 · 共享渲染内核（建议 v4.96.0）

文件：frontend/src/genui/*（新目录，全部新文件）

- spec/guard/parse/fingerprint/interaction-store（纯函数 + 单测）；
- GenuiBlock + blocks（layout/display/chart/code/forms/quiz）+ styles.css；
- GenuiActionContext + 防抖 + “已触发”反馈 + password 禁令；
- GenuiTextSegments / markdownFence 适配器（供 P2 用）。

验收：vitest 新套件（guard 边界、部分解析预算、每族渲染、持久化/重置/secret、
action 防抖、quiz 判卷）；tsc/eslint 0；与既有模块零耦合（无 gaea/antd import）。

### P2 · 双板块接入（建议 v4.97.0，可拆 2a 办公 / 2b 聊天两刀）

- 2a 办公：Markdown.tsx code 缝 + MemoMarkdown sourceKey/onPanel + AssistantMessage
  接线；genui 占位 chip；面板发布走 panel-store（先落地 store，Tab 在 P4）；
- 2b 聊天：ChatMarkdown + MarkdownContent + ChatRow/Page 的 ActionProvider；
  plain 终态与人格终态渲染；流式中不抢跑。

验收：?mock=1 DOM 走查 stat+table+chart 渲染、无 action 按钮禁用、坏围栏红横幅退化；
办公流式一条多围栏回答逐段上屏；聊天历史重放恢复答案。

### P3 · 模型侧接线（建议 v4.98.0）

- genui inline 技能 + 办公系统提示词指针 + genui_validate 工具（Go+注册+测试）；
- chatPlainSystemPrompt UI 规则块；whisper 统一注入点（先确认 custom 人格覆盖语义）；
- 真模型验收：办公让模型生成“本月订单看板/对比表/流程步骤”，聊天 plain 与一个
  人格各验一轮；JSON 错误率回归（先验后发纪律）。

验收：Go test-all 0 FAIL；vitest 计数续增；drift PASS（579）；真模型对话走查。

### P4 · 办公 UI 面板（建议 v4.99.0）

- panel-store + workspaceTabs "ui" + sidebarRegistry renderer + GenuiPanel 组件；
- /panel 命令（open/clear/指令）接 CommandPalette；
- 多回合 REPLACE/APPEND 更新、会话切换隔离、恢复重放收敛、角标（可选）。

验收：多回合“更新同一面板/追加 tab” mock + 真模型走查；清空两击确认；切换会话
互不串；历史恢复终态一致；vitest/tsc/eslint 0。

### P5 · 加固收口（建议 v4.100.0）

- 上限/降级审计、64KB 围栏上限、导出说明、记忆/压缩侧围栏剥离（若影响确认）；
- README/CHANGELOG/版本表/releases/vN.md、设计蓝图 design-system 页签补记；
- 视觉验收（judge 或 DOM 断言）、release exe 冒烟、.gaea/AGENTS.md + progress.md 回写。

### 并行线建议（沿用仓库默认并发纪律）

- P1：guard/parse 线、组件线、状态线可拆 3 条互不相交子线，主代理集成 GenuiBlock；
- P2：办公缝与聊天缝足迹互斥可并行；共享 genui 模块单一负责人（契约先行）；
- P3：Go 线（技能+工具+提示词）与前端能力面（图标/说明）互斥可并行；
- P4：单线为主。

---

## 8. 验收与测试门禁（每刀）

1. 前端：`cd frontend && npm test`（vitest 计数续增并更新 README/AGENTS 记录）、
   `tsc -b` 与 `eslint` 0；
2. Go：`scripts/test-all.ps1` 全量 0 FAIL；新增 genui_validate/skill 单测；
3. 契约：`scripts/check-types-drift.mjs` PASS（P1–P3 绑定面 579 不变）；
4. 交互走查：`?mock=1` DOM 断言（渲染/禁用/降级/持久化）；需真模型的项以单测 + 真机
   对话为验收面并在 release notes 如实标注；
5. 文档：releases/vN.md + README 版本表 + 更新本规划状态；.gaea/progress.md 回写。

---

## 9. 决策拍板清单

| # | 决策 | 推荐 | 影响 |
|---|---|---|---|
| 1 | 围栏标签 | `genui` 主用 + `dsh-ui` 兼容别名（成本≈1 行） | 模型先验可用性 vs 品牌纯净 |
| 2 | 聊天范围 | plain + 人格都接（统一注入点） | 需先验证 custom 人格覆盖链 |
| 3 | render_ui 工具卡 | v1 不做，围栏统一承载 | 零绑定面；办公面板仍可用 panel 围栏 |
| 4 | 媒体/本地产物图 | v1 不含，另案 | 避免本地文件 URL 语义仓促设计 |
| 5 | chat 导出含围栏 | 保留原文（自然退化） | 导出产物含代码块 |
| 6 | 未知 type 策略 | 白名单外丢弃（比上游更严） | 无扩展生态，杜绝注入面 |
| 7 | 状态粒度 | 消息级 key + 内容指纹 | 换题自动清零；同内容恢复 |
| 8 | 组件文案 i18n | 内容语言跟随；chrome 中文 | 对齐“页面内容 zh 单语”决策 |
| 9 | 版本编排 | P1→P5 五刀 | 可压缩合并 |

---

## 10. 风险与对策

| 风险 | 对策 |
|---|---|
| LLM JSON 输出不稳定 | 自检四步提示 + genui_validate + 标点级修复；结构性错误红横幅不猜 |
| 围栏污染记忆/压缩/成本 | 正文保留；记忆抽取侧 P5 剥离；token 成本如实可见（上下文页已统计） |
| 人格模式提示词注入破坏记忆管线 | 实施先验证装配链必经点与 custom 人格覆盖；规则块短小 ≤1.2k |
| 双板块样式漂移 | 共享组件只依赖全局 tokens；两板块接入仅传 provider/key |
| 流式体验折损 | 办公稳定前缀天然逐段上屏；聊天终态渲染；P5 才评估部分解析增强 |
| 历史会话反复发布面板 | sourceId+指纹幂等 + 重放顺序收敛，天然终态一致 |
| Go/TS 校验漂移 | Go 工具声明“结构校验、渲染器为最终权威”；上限常量同注释源 |
| action 在流式中误触发 | 聊天 sending 锁外放行；办公 running 走 steer；缺省无 provider 即禁用 |

---

## 11. 明确不做（反模式清单）

- 不复制 dsh 插件宿主、DOM 观察、fence-registry、双通道激活；
- 不做 ECharts/3D/diagram 编辑级引擎（重资产，价值密度低）；
- 不把 genui 做成“可以渲染任意 HTML/React”的万能口（白名单是安全边界）；
- 不在聊天加 dock/panel（chat 是会话不是工作台）；
- 不新增 Wails 绑定/事件类型绕开文本通道（P1–P3 绑定面 579 不动）；
- 不把渲染器与 antd/tailwind 耦合（聊天与办公必须同一套内核）；
- 不为 genui 引入 localStorage 敏感数据（交互值以外一律不存）。

---

## 12. 待实施首日确认的空白点

1. whisper 人格链路最终 SystemPrompt 的唯一必经点（custom 人格是否绕过
   BuildMainChatSystemPrompt）；
2. 办公“做梦/归档”摘要读取 assistant 正文的位置（确认是否需要剥离围栏）；
3. 办公运行中 steer 的用户可见形态（是否显示“插话”行）与 action 展示文案长度上限；
4. ChatPage 历史话题重载的 msg.key 稳定性（决定状态 key 是否含消息序号）；
5. 现有 vitest 计数基线（v4.95：1786/1786）与测试命令别名。
