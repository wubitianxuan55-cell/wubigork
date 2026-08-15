# 01 前端壳层调研报告（gaea 3.0 架构改造评审）

> 调研范围：frontend/src 下的壳层文件（main/App/MainLayout/appStore/hooks/api/utils/theme/wailsjsCompat/types）以及壳层直接耦合的 gaea/lib/bridge.ts。只读调研，未修改任何代码。所有结论均带文件路径 + 行号。

## 1. 概览（文件清单 + 行数 + 职责）

| 文件 | 行数 | 职责（一句话） |
|---|---|---|
| frontend/src/main.tsx | 154 | React 入口；StrictMode + ErrorBoundary + ToastProvider 装配；3 个 IIFE 环境修复（rAF 降级 / 前端诊断上报 gaea.log / 固定窗口缩放） |
| frontend/src/App.tsx | 163 | 主题壳：initBridge() + initRuntimePolyfill() 模块作用域最早调用（App.tsx:10-11）；M3 CSS 变量写入 :root（App.tsx:44-130）；antd ConfigProvider 包 MainLayout |
| frontend/src/App.css | 1 | 占位文件，仅注释（样式实际在 index.css 与 gaea/styles.css） |
| frontend/src/layouts/MainLayout.tsx | 470 | 壳层核心：导航菜单、页面装配与保活、快捷键、面包屑、底栏状态条、模型监控 |
| frontend/src/stores/appStore.ts | 375 | zustand 全局状态：登录/项目/主题外观/派生统计，localStorage 持久化 |
| frontend/src/stores/outlineStore.ts | 54 | zustand 大纲共享层（小说页面间共享） |
| frontend/src/wailsjsCompat.ts | 21 | 10 个后端门面的再导出合并层（S2-3 兼容） |
| frontend/src/hooks/useChatStream.ts | 177 | 聊天流式发送状态机（chat-stream:<runID> 订阅 + 30s 超时 + 模拟打字） |
| frontend/src/hooks/useChatTopics.ts | 189 | 聊天话题/模式/人格元数据状态机 |
| frontend/src/hooks/useChatVoice.ts | 59 | 语音对话集成（消息落库 + 首页语音入口兼容） |
| frontend/src/hooks/useVoiceChat.ts | 543 | 语音采集/识别/TTS 播放 + 8 个 voice:* 事件订阅 |
| frontend/src/hooks/useFeatureModel.ts | 48 | 功能级模型绑定（feature-model-changed 订阅） |
| frontend/src/hooks/useImageGenConfig.ts | 260 | 绘梦配置/引擎状态/后端切换状态机 |
| frontend/src/hooks/useImageGenQueue.ts | 289 | 生成队列/任务执行状态机 |
| frontend/src/hooks/useImageGenHistory.ts | 89 | 历史/灯箱状态与结果操作 |
| frontend/src/hooks/useCustomTemplates.ts | 83 | 自定义模板弹窗状态机 |
| frontend/src/api/settings.ts | 104 | 设置数据 API（config/TTS/语音/办公设置摘要） |
| frontend/src/api/engines.ts | 491 | 模型引擎 API（引擎 CRUD/模型中心/测评/分流统计，最大 api 模块） |
| frontend/src/api/image.ts | 223 | 图片生成 API（生成/ComfyUI/系统状态/剧照） |
| frontend/src/api/characterlib.ts | 113 | 全局角色库 API（24 个 Character* 方法） |
| frontend/src/api/httpToken.ts | 17 | HTTP 调试桥一次性 token 读取 |
| frontend/src/api/runtimePolyfill.ts | 318 | 浏览器环境 runtime.EventsOn 兼容层（SSE + 内存总线） |
| frontend/src/utils/ 13 个文件 | 4~78 | 纯函数工具：chatTopics/chatComposer/theme/loraFilter/outline/time/zIndex/scroll/text/readingProgress/mermaidPng/characterStatus |
| frontend/src/theme/m3-palette.ts | 245 | M3 Tonal Palette 轻量实现（appStore 主题 token 计算依赖） |
| frontend/src/types/index.ts | 127 | 共享领域类型（大纲/角色/世界观/TTS 等） |
| frontend/src/types/wails.d.ts | 264 | window.go/window.runtime 类型声明（AppFacade/AppAPI/RuntimeAPI） |
| frontend/src/gaea/lib/bridge.ts | ~900 | 办公 UI 的后端调用面：AppBindings 契约 + 门面路由代理 + 事件订阅（onEvent/onUpdaterProgress/onTaskEvent/onReady）+ initBridge 双环境注入 + 编译期绑定漂移检查 |
| frontend/src/gaea/lib/bindingNames.ts | 467 | gen_bindings -names 生成的 Go 绑定方法名全集（编译期校验用） |

## 2. 装配与导航机制（证据）

### 2.1 组件树与启动顺序

1. createRoot 挂载：main.tsx:156-165 —— StrictMode > ErrorBoundary > ToastProvider > App。
2. 环境修复 IIFE 在模块求值时先于渲染执行：rAF 降级（main.tsx:20-73）、前端诊断上报（main.tsx:78-134，window error / unhandledrejection / confirm-alert 替换 / 心跳 / longtask）、缩放锁定（main.tsx:139-154）。
3. App.tsx:10-11 在模块作用域最早时机调用 initBridge() 与 initRuntimePolyfill()（幂等，见 bridge.ts:855-858、runtimePolyfill.ts:126-127）——保证组件树渲染前 window.go.app.App 与 window.runtime 可用。
4. App 组件从 appStore 读全部外观状态（App.tsx:22-28），计算 effTokens（App.tsx:36-41），useEffect 把 M3 token 写成 :root CSS 变量（App.tsx:44-130），再以 ConfigProvider 包住 MainLayout（App.tsx:137-157）。

### 2.2 导航机制完整清单

- **Page 类型**：MainLayout.tsx:27 —— `'home' | 'novel' | 'imagegen' | 'settings' | 'modelcenter' | 'characterlib' | 'chat' | 'gaea' | 'memoryhub'`（9 值字面量联合）。
- **allPageKeys**（导航白名单）：MainLayout.tsx:30 —— ['chat','novel','imagegen','gaea','memoryhub','modelcenter','characterlib']（7 个；home 与 settings 不在内）。注释明确"navigate 事件校验 + Ctrl+1~4 快捷键映射"。
- **menuItems**（顶栏菜单）：MainLayout.tsx:33-42 —— 8 项（首页 + 7 板块）。settings 不在菜单，只能由右上角按钮进入（MainLayout.tsx:383-387）。
- **pageComponents**：MainLayout.tsx:44-53 —— Record<Exclude<Page,'home'>, ReactNode>，8 页组件元素。
- **lazy 加载**：MainLayout.tsx:17-24 —— 8 个页面全部 React.lazy，Suspense fallback 在 MainLayout.tsx:434-442（"正在唤醒 AI 模块…"）。
- **页面切换途径（6 条）**：
  1. 菜单点击 onClick={() => setPage(key)}（MainLayout.tsx:336）；
  2. 右上角设置按钮 setPage('settings')（MainLayout.tsx:384）；
  3. ModuleLauncher onNavigate prop（MainLayout.tsx:446 → ModuleLauncher.tsx:81）；
  4. navigate 自定义事件：监听在 MainLayout.tsx:267-276（校验 detail.page ∈ allPageKeys 后 setPage）；分发点共 3 处——pages/chat/utils.ts:13-15（navigateToCharacterLib）、pages/CharacterPage.tsx:76-77、components/settings/ModelPanel.tsx:59；
  5. Ctrl+1~4 快捷键（MainLayout.tsx:284-287）：setPage(allPageKeys[Number(e.key)-1]) —— 顺序依赖 allPageKeys 前 4 项（chat/novel/imagegen/gaea）；
  6. Ctrl+N 新建项目跳 novel（MainLayout.tsx:289-292）。
- **visitedPages 保活**：MainLayout.tsx:223 初始 new Set(['home'])；MainLayout.tsx:226-233 每次 page 变化加入集合；渲染在 MainLayout.tsx:443-449 —— Array.from(visitedPages).map() 全部渲染、非当前页 display:none，实现"切换 tab 不销毁组件状态"（注释见 225 行）。Suspense/ErrorBoundary 包裹在 Content 内（MainLayout.tsx:433-451）。
- **home 特判**：MainLayout.tsx:445-447 —— p === 'home' 渲染 <ModuleLauncher onNavigate={...}/>，否则渲染 pageComponents[p]。
- **面包屑**：MainLayout.tsx:409-423 —— 条件 projectOpen && page !== 'novel' && page !== 'home'；第一级固定为 projectInfo.title 且点击回 novel（MainLayout.tsx:415-417）——"项目名→小说"语义硬编码在壳层。
- **Content 布局特判**：MainLayout.tsx:424-432 —— padding/背景/overflow 对 chat 与 gaea 两组页面做特殊值（页级布局策略散落在壳层）。
- **底栏**：MainLayout.tsx:459-462 常驻 StatusBar（组件定义 MainLayout.tsx:103-206，含 3s 轮询模型监控 141、超载告警 110-131、写作进度计算 152-154）。
- **孤儿页面**：pages/KnowledgePage.tsx 存在但未在任何导航/组件映射中注册（对照 menuItems/pageComponents 全表），与 gaea3 文档 §3.1 结论一致（能力已被 memoryhub 取代）。

## 3. 状态管理与后端调用面

### 3.1 状态管理

- **zustand**：useAppStore（appStore.ts:204）、useOutlineStore（outlineStore.ts:31）。无其他全局 store（grep zustand 仅此两处 create）。
- **Context**：只有组件级 Provider——ToastProvider（main.tsx:161）、LocaleProvider（GaeaPage.tsx:17，办公板块内部）。无全局业务 Context。
- **appStore 状态分组**：登录（loggedIn/login/checkLogin/logout，appStore.ts:264-301）、项目（projectOpen/projectPath/projects/novelsDir 及 load/delete/set，appStore.ts:303-362）、外观（baseTheme/mode/darkMode/density/motion/accentColor/fontFamily/fontSize，appStore.ts:211-262）、派生数据（projectInfo/stats，appStore.ts:309-321）。
- **持久化**：localStorage 键集中定义 appStore.ts:82-91；每个 setter 同步写 localStorage（如 setTheme 225、setMode 236）；系统暗色监听 matchMedia（appStore.ts:366-375）。
- **页面间状态共享现状**：页面级状态全部本地 useState/hook（ChatPage.tsx:36-58 持 messages，useChatTopics/useChatStream/useChatVoice 注入更新）；跨页面共享只有 appStore（壳层外观/项目）与 outlineStore（大纲）；跨页跳转靠 navigate 自定义事件广播（§2.2 第 4 条）或 prop 回调（ModuleLauncher onNavigate）。**没有会话级共享状态**——这是"会话事件日志作事实源"改造的直接落点。

### 3.2 后端调用面（三层并存）

**第 1 层：wailsjsCompat 合并层**（wailsjsCompat.ts:16-25，re-export 10 门面 CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB）。生成物 frontend/wailsjs/go/app/*.js 已入库，调用形态是 window['go']['app']['CoreB']['Xxx']()（CoreB.js:5-15 示例）。全仓 41 个文件 import 它（grep 统计），是业务页面最主要调用入口。

**第 2 层：api/ 模块封装**（消除 (window as any) 的薄封装）：
- api/settings.ts —— GetConfig/SaveConfig/GetImageBackendInfo/SetImageBackend/TTS 系列（settings.ts:21-104）；
- api/image.ts —— 图片生成 5 模式/ComfyUI 状态与 LoRA/系统状态/角色剧照（image.ts:47-225）；
- api/engines.ts —— 引擎 CRUD、Key 管理、模型统计、Herdsman 目录/启停/测评/分流（engines.ts:307-493，491 行为 api 最大模块，其中类型定义占约 300 行）；
- api/characterlib.ts —— 全局角色库 24 个方法（characterlib.ts:22-116）；
- api/httpToken.ts —— 读取 __GAEA_HTTP_TOKEN / localStorage（httpToken.ts:14-23）。

**第 3 层：window.go.app.App 直连**（绕过两层封装，约 12 个文件）：appStore.ts（13 处，appStore.ts:267/272/288/296/311/318/325/334/343/353/357）、MainLayout.tsx:241（GetActiveModel）、HomePage.tsx:65-86、SearchModal.tsx:56、SkillModal.tsx:35、TTSPlayer.tsx:165、SecurityBanner.tsx:44、components/novel/*（outlines.ts:17、CommandBar.tsx:70、usePlotBranch.ts:36/48、AIConsole 等）、SecurityPanel.tsx:26、MemoryHubPage.tsx:86、ModelCenterPage 系列（ModelCenterPage.tsx:34、BenchmarkSection.tsx:154、useBindState.ts:51）。

**第 4 层（办公板块专属）：gaea/lib/bridge.ts 的 app 代理**：
- app 代理（bridge.ts:742-756）：调用时 realApp() ?? getMock()，经 gaeaToGaea 短名→Gaea* 前缀映射（bridge.ts:503+，如 Submit→GaeaSend）后按方法名在门面对象上查找；统一 invoke 错误归一化（bridge.ts:720-730，上报 gaea.log）。
- realApp()（bridge.ts:438-454）：Proxy 遍历 window.go.app 下所有门面按 key 找方法。
- initBridge（bridge.ts:855-871）：Wails 环境 ensureLegacyAppProxy 补 window.go.app.App 旧形态代理（bridge.ts:793-811，第 3 层直连点因此仍可用）；HTTP 环境 createAppProxy → POST /api/rpc（bridge.ts:814-848，Bearer token 鉴权）。
- 编译期漂移检查：bridge.ts:880-939 用 bindingNames（gen_bindings 生成）对 AppBindings/gaeaToGaea 做类型级双向断言；LegacySurfaceNames（bridge.ts:921+）显式列出"Go 侧存在但不经 AppBindings 消费"的 legacy 方法。

### 3.3 runtimePolyfill 的作用（浏览器调试）

- 初始化：App.tsx:11 模块作用域调用；仅当无原生 window.runtime.EventsOn 时接管（runtimePolyfill.ts:130-133），先探测 /api/health 确认 Go 桥接存在（runtimePolyfill.ts:229-243），订阅事件经 fetch SSE 连接 /api/stream?id=<事件名>（runtimePolyfill.ts:311，token 走 Authorization: Bearer，runtimePolyfill.ts:307-310）。
- 内存事件总线（runtimePolyfill.ts:39）支持 EventsEmit 本地发射；断线 5s 自动重连（runtimePolyfill.ts:346-350）。
- 帧解析为纯函数可单测：parseSSEFrame（runtimePolyfill.ts:58-76）、parseSSEStream（runtimePolyfill.ts:82-114）。
- **局限**：polyfill 只补 window.runtime，不补 wailsjs 生成模块（wailsjs/go/app/*.js 是已入库的真实模块，浏览器 dev 下 import 不报错但 window.go.app 为空——靠 gaea/lib/bridge.ts 的 mock 兜底，bridge.ts:1-6 注释说明"pnpm dev 外壳外回退 mock"）。

## 4. 事件订阅清单（window.runtime.EventsOn 全量）

### 4.1 后端事件（订阅点 → 事件名 → 文件:行）

| # | 事件名 | 订阅点 | 卸载方式 |
|---|---|---|---|
| 1 | model-changed | MainLayout.tsx:251（顶栏模型标签刷新） | EventsOff('model-changed') MainLayout.tsx:256 |
| 2 | feature-model-changed | hooks/useFeatureModel.ts:34（按 d.feature 过滤） | EventsOn 返回的卸载函数 |
| 3 | chat-stream:<runID> | hooks/useChatStream.ts:123（动态频道，runID 由 ChatStreamPlain 返回） | streamCleanupRef（useChatStream.ts:96） |
| 4 | create-chapter-stream | components/novel/create/useChapterStream.ts:46（CREATE_CHAPTER_STREAM_CHANNEL） | useChapterStream.ts:52 |
| 5 | ghost-stream | components/novel/editor/GhostText.tsx:65 | EventsOff('ghost-stream', handler) GhostText.tsx:70 |
| 6 | xai-output | components/novel/AIConsole.tsx:62 | EventsOff('xai-output') AIConsole.tsx:65 |
| 7 | tts-stream | components/TTSPlayer.tsx:109 | EventsOff('tts-stream') TTSPlayer.tsx:142 |
| 8 | new-characters-discovered | components/novel/create/NewCharactersModal.tsx:60 | EventsOff 同文件:61 |
| 9 | character-fill-progress | pages/CharacterLibraryPage.tsx:189 | EventsOff('character-fill-progress') CharacterLibraryPage.tsx:207 |
| 10-17 | voice:state / voice:transcript / voice:reply / voice:tts-audio / voice:tts-speak-text / voice:tts-speak-cancel / voice:listening / voice:thinking | hooks/useVoiceChat.ts:116/127/143/150/164/189/194/200 | unsubs 数组统一退订（useVoiceChat.ts:205-207） |
| 18 | gaea-event（EVENT_CHANNEL 常量） | gaea/lib/bridge.ts:420（onEvent） | 返回卸载函数（bridge.ts:421） |
| 19 | updater:progress | gaea/lib/bridge.ts:464（onUpdaterProgress） | 同上 |
| 20 | gaea-task | gaea/lib/bridge.ts:478（onTaskEvent） | 同上 |
| 21 | gaea-ready | gaea/lib/bridge.ts:488（onReady） | 同上 |

### 4.2 前端自定义事件（dispatchEvent，非 runtime）

| 事件名 | 分发点 | 消费点 |
|---|---|---|
| navigate | pages/chat/utils.ts:13-15、pages/CharacterPage.tsx:76-77、components/settings/ModelPanel.tsx:59 | MainLayout.tsx:267-276（allPageKeys 白名单校验） |
| gaea-persona-changed | pages/CharacterLibraryPage.tsx:34 | CharacterLibraryPage.tsx:85 |
| gaea-project-chars-changed | pages/CharacterLibraryPage.tsx:140、components/novel/create/NewCharactersModal.tsx:100 | CharacterLibraryPage.tsx 等 |
| ai-assist-send | components/novel/ChapterEditor.tsx:69 | （章节编辑器内部） |

### 4.3 订阅方式两套并存

- 规范路径：import { EventsOn } from '../../wailsjs/runtime/runtime'（useChatStream.ts:6、useVoiceChat.ts:3、useFeatureModel.ts:3）——返回卸载函数。
- 手动路径：window.runtime?.EventsOn?.('xxx', h) 裸调（MainLayout/CharacterLibraryPage/AIConsole/TTSPlayer/GhostText/NewCharactersModal/useChapterStream/bridge.ts）——其中 CharacterLibraryPage.tsx:207 的 EventsOff 不带回调（按 wailsjs 语义会移除该事件全部监听，存在误删风险）。

## 5. 与 3.0 目标相关的关键发现（Manifest 化硬编码点清单）

### 5.1 MainLayout 内"加板块必须改"的硬编码点（逐一）

| # | 硬编码点 | 位置 | 改动成本说明 |
|---|---|---|---|
| 1 | Page 类型字面量联合 | MainLayout.tsx:27 | 新增板块必须扩展联合类型 |
| 2 | allPageKeys 数组 | MainLayout.tsx:30 | 决定 navigate 白名单与 Ctrl+N 之外所有快捷入口 |
| 3 | React.lazy 导入 | MainLayout.tsx:17-24 | 每个页面一行 import |
| 4 | menuItems 菜单项 | MainLayout.tsx:33-42 | 菜单文案/图标/顺序 |
| 5 | pageComponents 映射 | MainLayout.tsx:44-53 | 页面组件实例注册 |
| 6 | 快捷键 Ctrl+1~4 | MainLayout.tsx:284-287 | 顺序绑定 allPageKeys 前 4 项，新增快捷入口需重排 |
| 7 | pageLabels 映射 | MainLayout.tsx:208-210 | 面包屑标题 |
| 8 | 面包屑"项目名→novel"语义 | MainLayout.tsx:415-417 | 项目页语义硬编码为 novel |
| 9 | Content 布局特判 | MainLayout.tsx:424-432 | chat/gaea 专属 padding/背景/overflow 特判 |
| 10 | visitedPages 初始 home | MainLayout.tsx:223 | 启动页固定 home |
| 11 | home 特判分支 | MainLayout.tsx:445-447 | 启动器/普通页双分支渲染 |
| 12 | settings 隐式入口 | MainLayout.tsx:383-387 | 不进菜单，只有右上角按钮——入口分散 |

### 5.2 壳层之外的重复清单（同一板块信息在多处维护）

- ModuleLauncher.tsx:17-18 LauncherTarget（6 值）+ modules 数组（ModuleLauncher.tsx:31-38）——与 MainLayout Page 类型/菜单**部分重复维护**，且 LauncherTarget 无 memoryhub/characterlib（首页启动器缺两块）。
- 后端侧 module_registry.go（4 模块）、main_brain 意图表、README 7 模块、前端 9 页——五层清单不一致（gaea3 文档 §3.1 已列，前端证据见 §2.2）。
- 页面组件与 lazy 注册分散在 MainLayout 一处，但每页的路由/保活/布局策略（§5.1 #1/#2/#5/#9/#11）没有集中描述物。

### 5.3 壳层与业务页面的耦合度

- **壳层直连后端**：MainLayout 直接 import * as App from wailsjsCompat（MainLayout.tsx:16），直调 GetActiveModel（241）与 GetModelMonitor（134）——壳层承担模型监控展示逻辑（StatusBar 103-206 内联 103 行业务逻辑：超载告警、写作进度、引擎清单）。
- **登录状态耦合**：登录 xAI 按钮在壳层顶栏（MainLayout.tsx:392-399），逻辑在 appStore.login（appStore.ts:264-284，75 次×4s 轮询最长 5 分钟）。
- **页面编排耦合**：ChatPage 是编排层（ChatPage.tsx:34-49 组装三个 hook + 展示组件），页面内状态不跨页共享——"换板块"即状态丢失，visitedPages 保活只保组件实例不保业务状态。
- **办公板块相对自治**：GaeaPage 只是薄壳（GaeaPage.tsx:13-23，LocaleProvider 包 GaeaApp），其内部全部经 bridge.ts 调用后端——这是三块业务中最接近"板块独立"的形态，也是 Manifest 化的现成参照。

### 5.4 适合下沉为清单驱动的逻辑

1. **导航清单**（§5.1 #1-7/#10-12）：menuItems/pageComponents/allPageKeys/pageLabels/快捷键/首页入口 → 单一板块 Manifest（id/label/icon/component/lazy/keepAlive/layout/shortcut/menuOrder）。
2. **布局策略**（§5.1 #9）：Content padding/背景/overflow 特判 → Manifest 的 layout 属性（'full' | 'padded'）。
3. **壳层监控展示**（§5.3）：StatusBar 的模型监控轮询/超载告警 → 独立 hook + 数据源，壳层只做装配。
4. **事件名**：21 个后端事件名 + 4 个前端事件名散落硬编码 → 事件常量表（eventNames.ts）。
5. **后端调用入口**：4 层并存（§3.2）→ 收敛为单一 Seam（wailsjsCompat 或 bridge app 代理二选一），与"Provider Seam 化"目标同构。

## 6. 缺陷与风险

1. **后端调用 4 层并存**（§3.2）：同一方法存在 3 种以上调用路径（例：GetActiveModel —— MainLayout.tsx:241 直连 / api/settings.ts:78-81 经 wailsjsCompat / bridge.ts 代理可按名路由）。改造时"换 provider 只改配置"的 Seam 需要先统一入口，否则漂移面大。
2. **事件订阅两套 API 并存且语义不一致**（§4.3）：wailsjs runtime EventsOn 返回卸载函数；window.runtime?.EventsOn?. 裸调路径里 CharacterLibraryPage.tsx:207 的 EventsOff 无回调参数（会移除该事件全部监听，若多订阅点共存会互相误删）；types/wails.d.ts:21 声明的 EventsOn 返回 void，与真实 wailsjs 语义不符。
3. **壳层膨胀**：MainLayout.tsx 470 行内含 StatusBar 103 行（103-206）+ 主题色块 UI（342-362）+ 登录按钮（392-399）+ 模型监控轮询（107-143）——壳层同时承担装配、展示、轮询、告警四类职责。
4. **导航白名单与快捷键顺序耦合**（MainLayout.tsx:284-287）：Ctrl+1~4 依赖 allPageKeys 前 4 项顺序，任何菜单重排都会悄悄改快捷键语义，无显式校验。
5. **页面状态无跨页共享**：ChatPage/ImageGenPage 等页面级状态在 visitedPages 保活下幸存，但刷新/重建即丢；会话事实源缺失（与 gaea3 缺陷 3 呼应——前端只能拿后端投影结果，无法重放）。
6. **login 轮询阻塞式 5 分钟**（appStore.ts:269-279）：Login() 后 75×4s 轮询 GetLoginStatus，无事件驱动替代（后端无 login 事件）。
7. **孤儿页 KnowledgePage**：未注册任何导航（§2.2），长期无人认领，容易误导后续改造者（"看起来有知识库板块"）。
8. **runtimePolyfill 覆盖有限**：SSE 按事件名单连接（runtimePolyfill.ts:311），多订阅点共享一个连接；浏览器 dev 下 wailsjs 生成模块可 import 但 window.go.app 为空，页面错误被 mock 静默吞掉的风险（bridge.ts:1-6 只保证办公板块 UI 可开发，不保证聊天/小说/绘梦板块的浏览器调试）。
9. **bridge.ts 的 gaeaToGaea 手工映射表**（bridge.ts:503+）虽经编译期断言保护（bridge.ts:880-939），但映射本身是办公板块专用的"短名→Gaea* 前缀"约定，属于办公引擎内部命名规则，对 Manifest 化后的通用板块协议不具备可迁移性。
10. **事件名无集中常量表**：21+4 个事件名全部字符串硬编码（§4），改名需全局 grep，无编译期保护。

## 7. 改造建议

1. **建立板块 Manifest（Step 2 前端落点）**：定义 `BoardManifest { id, label, icon, component, lazy, keepAlive, layout: 'full'|'padded', shortcut?, menuOrder?, breadcrumb? }`，MainLayout 从单一 manifest 派生 menuItems/pageComponents/allPageKeys/pageLabels/快捷键/Content 布局/首页入口；ModuleLauncher 从同一 manifest 过滤生成卡片（顺带补上 memoryhub/characterlib 两块缺失入口）。「加板块只写声明」在壳层的 12 个硬编码点（§5.1）全部收敛。
2. **导航事件契约化**：navigate 事件载荷校验从 allPageKeys 数组改为 manifest.keys（MainLayout.tsx:270-271），并给事件名加常量。
3. **统一事件订阅层**：新建 src/events.ts 常量表 + subscribe() 封装（返回卸载函数），把 12 处 window.runtime?.EventsOn?. 裸调全部迁移；修掉 CharacterLibraryPage.tsx:207 的无回调 EventsOff。
4. **后端调用面收敛为单 Seam**：以 wailsjsCompat 为唯一业务入口（api/ 模块内统一），将 12 个 window.go.app.App 直连文件（§3.2 第 3 层）迁移；bridge.ts 的 app 代理保留给办公板块但明确为 legacy 面；为 Step 3 Provider Seam 预留"调用方不感知 provider"的接口层。
5. **壳层瘦身**：StatusBar 拆为独立组件 + useModelMonitor hook（轮询/告警下沉）；登录交互保留但轮询改为事件驱动（后端补 login 事件或 SSE 推送）。
6. **会话状态事实源化（Step 1 前端配合）**：ChatPage 消息流改为从会话事件日志投影读取/订阅（替代 ChatMessagesList 全量重读 + 本地 useState），跨页/多前端同步自然获得。
7. **孤儿页决策**：KnowledgePage 挂载到 manifest 或删除（防误导）。
8. **编译期护栏延续**：把 manifest 的 id 与 allPageKeys/Page 类型用类型级断言绑定（复用 bridge.ts:880-939 的 AssertNever 模式），menu 重排不改变快捷键语义。

> 结论：壳层是 3.0「板块 Manifest 化」前端改造的主战场——MainLayout.tsx 一处集中了 12 个硬编码点，且与 ModuleLauncher 的 LauncherTarget、后端 module_registry 存在三层清单漂移；后端调用面 4 层并存是 Provider Seam 化的前置障碍；事件订阅分散且两套 API 并存是事件日志化的直接风险面。办公板块（bridge.ts + GaeaPage 薄壳）已是相对自治形态，可作 Manifest 化的样板。
