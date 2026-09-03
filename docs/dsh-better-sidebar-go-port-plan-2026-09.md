# 调研与规划：DSH-better-sidebar 整个 Go 化移植到办公板块（替换现有右栏工作台）

> 日期：2026-09-03 | 基线：v4.59.0（绑定面 559）
> 状态：调研 + 方案完成，待用户拍板；**未改任何代码**
> 目标：① 全量盘点 [omdsh-dev/DSH-better-sidebar](https://github.com/omdsh-dev/DSH-better-sidebar)
> 的能力面（不止此前两轮蒸馏选的项）；② 把它的 host 半（文件/终端/Git/任务/浏览器/
> 会话作用域/信任围栏）用 **Go** 同构重写为 gaea 办公板块底座，React 只保留渲染壳；
> ③ **替换现在**办公板块右栏的 TS 蒸馏实现（4 tab 工作台 + 会话级外壳），让
> 「逻辑/状态/事件/读写/进程」的权威从前端 localStorage/组件内部逻辑迁到 Go；
> ④ 输出拍板清单与分刀排期。

---

## 0. 代码通读补记（2026-09-03 二轮，逐文件）

> 首版文档基于 README + 文件盘点 + 头部阅读；用户质疑「你读了代码了吗？」后，
> 对两侧相关源码做了逐文件补读。本节记录**读过的文件**与**由此修正的结论**，
> 作为首版映射表的勘误依据。未读透的部分如实列出（见 §0.3）。

### 0.1 dsh-better-sidebar 已通读（本地克隆 v0.14.1 src/）

- host 半：`index.ts`（全 1022 行：方法表/路由/终端 WS/设置 seam）、`git.ts`、
  `jobs-routes.ts`、`tools.ts`（8 个 terminal_* 工具全量）、`config.ts`、
  `prefs-shared.ts`、`fs-tree.ts`、`fs-operations.ts`、`fs-search.ts`、`wire.ts`、
  `trust-fence.ts`、`html-route.ts`、`pty-manager.ts`（头）、`agent-pty.ts`（头）。
- client 半：`service.ts`（注册表契约全量）、`api.ts`（客户端 API 全量）、
  `builtins/tabs.tsx`、`builtins/viewers.tsx`（通读）；`state.ts`/`Sidebar.tsx`/
  各视图组件仅头部，未逐行。

**由此修正的结论：**

1. **host API 面（不是 7 个概念路由，是一张方法表）**：`session.cwd`、
   `fs.tree/search/read/write`、`git.status/diff/stage/unstage/commit/branch/
   checkout/log/commit-diff/discard/revert/cherry-pick/show`、`pty.close`、
   `agent-pty.close`、`terminal.deps`、`jobs.output/kill`、`shell.get`、
   `settings.get/update`、`browser.probe`；另有 `/sidebar/upload`、
   `/sidebar/file`（媒体，**仅会话 cwd 内 + ≤20MB**）、`/sidebar/html`（路径编码
   路由，相对资源可解析，CSP sandbox）、`/sidebar/bundle`（懒加载 chunk）、
   `/sidebar/ws/terminal`、`/sidebar/ws/agent-terminals`。Go 移植应以方法表
   为蓝本（§4 已按此列包）。
2. **git.ts 无 push/pull/fetch（确认）**，还含 branch/checkout/log/show/
   commit-diff/discard/revert/cherry-pick；porcelain `-z` + `\x1f` 机器可读
   parse，`git -C <cwd>`，**从不设身份**。Go 侧可用 os/exec 1:1 同构。
3. **v0.14.1 的 `fs.write` 本身没有 isWithin 围栏**（只 requireAbsolute +
   原子 rename）；围栏只在上传/媒体/HTML 路由。v0.18 README 才把 workspace
   fence 做成设置项。→ Go 移植的 fs.write 应当**默认加围栏**（比插件更严，
   与 gaea GaeaWriteFile 的 WriteRoots 口径一致）。
4. **jobs.output 是「模型已读内容的回放」**（从 owner session 事件日志折叠
   tool/call + tool/result，绝不消费模型 job_output 游标），kill 走宿主 jobs
   注册表并限 owner session。gaea 的 TaskCenter（GaeaTaskList + 事件增量 +
   2s 输出轮询）语义近似但缺「回放/退出码/强杀」区分——映射时按此修正。
5. **客户端 API 只有 JSON 调用 + media/html URL + 上传**，无「订阅 API」——
   会话/任务/子代理数据经宿主推送镜像（session/jobs push）或轮询。gaea 已有
   /api/stream SSE + Wails runtime 事件，通道更完整。
6. settings 是**声明式 schema + revision-guarded update**（prefs 键如
   openByDefault/widthPercent/autoOpen*/agentTerminalTools/tabsEnabled/
   viewersEnabled/pluginSettings…），另有 `externalDisable`（与 aionui 右侧
   面板互斥）。gaea 现用 localStorage（workspaceTabs v2 记录 + 全局宽度键 +
   enabled 集），键语义同构但权威在浏览器。

### 0.2 gaea 办公板块已通读（v4.59 基线）

- 前端：`App.tsx`（1236 行全读，含布局/事件/面板上下文组装/快捷键）、
  `lib/workspaceTabs.ts`、`lib/sidebarRegistry.ts`、`components/WorkspaceTabs.tsx`；
  数据源抽查（`DeliverablesPanel`/`ChangesPanel`/`TaskCenter`/`SubagentsPanel`/
  `BrowserPanel`/`MergedPanel` 的绑定调用）；`FileTree.tsx`（头）、`FilePreview`/
  `EditorTabs` 等仅结构。右侧面板组件（DocxPreview/XlsxPreview/ContextView/
  ChatTabs）未逐行。
- Go：`internal/app/gaea_ui_extra.go`（ListDir/FileSearch/Materials/ReadFile/
  WriteFile/Open-RevealWorkspacePath 等，逐段读）、`gaea_browser.go`（全）、
  `gaea_ui_contextview.go`、`gaea_tasks.go`（接口面）、`gaea_journal.go`、
  `gaea_deliverable_registry.go`、`internal/gaea/tasks/tasks.go`（Cancel/Retry/
  Wait 语义段）、`internal/gaea/tool/builtin/sidebar_open.go`（全）、
  `internal/gaea/browser`（接口面）、`internal/gaea/contextview`、
  `internal/gaea/trajectory`（折叠函数面）、`internal/httpbridge/bridge.go`
  （SSE/RPC）、`main.go`（全）。

**由此修正的结论：**

1. **现状右栏 4 tab 与数据源确认**：files → `WorkspacePanel`（FileTree +
   EditorTabs，Go 侧 `GaeaListDir` 返回 name/isDir/size）；deliverables →
   `MergedPanel(DeliverablesPanel + ChangesPanel)`；tasks → `MergedPanel
   (TaskCenter + SubagentsPanel)`；browser → `BrowserPanel`（观察窗轮询
   `GaeaBrowserObserve`）。注册表 = `sidebarRegistry.ts` 前端 RENDERERS。
2. **产物是「双轨」而非纯启发式**：App.tsx 从 `state.items`（转录正文提及 +
   写类工具参数）现场折叠启发式清单；Go `GaeaDeliverableRegistry(sessionPath)`
   （trajectory.FoldDeliverables）为权威登记表（DeliverablesPanel 挂载自拉）；
   `GaeaJournalList/RollbackRecord/VerifyRecord/ZipDeliverables`（证据链）提供
   版本时间线/回滚/验证。→ F5「本轮文件视角」在 gaea 已有三套同源数据
   （启发式 + 登记表 + evidence），替换工作应**统一收敛到 Go 折叠 + 权威登记**
   并把启发式清单降为实时中间态，而不是新增第四套。
3. **gaea 文件绑定比插件粗**：ListDir 无 symlink/hidden/排序语义、无行级预算；
   ReadFile 返回整文件文本无字节上限/无 binary 探测；WriteFile 有扩展名白名单
   + WriteRoots 围栏 + 2MB 上限 + 原子写（比插件 fs.write 更严）。→ Go 工作台
   fs 层应新增媒体路由语义（字节/类型/上限/二进制探测）与上传，直接复用现有
   写路径守门。
4. **任务 Cancel = 协作式中断**（queued 原子置 cancelled / running context
   传播 + stopping 状态），**不是进程树强杀**；bash 工具另有 killProcessTree。
   → 「强杀/退出码」确为缺口；设计可复用 internal/gaea/proc + bash 工具的
   进程树杀法。
5. **事件通道**：桌面 = Wails runtime events + `GaeaResyncEvents` 防吞件；
   调试桥 = `/api/rpc`（JSON）+ `/api/stream`（SSE，fetch 流式 + Authorization）。
   **httpbridge 目前无 WebSocket 服务端**（gorilla/websocket 在 go.mod，为 CDP
   客户端使用）。→ 终端实时通道建议先走 SSE 下行 + RPC 上行（复用现有基建、
   无新增 WS 面），WS 作为二期选项；这比首版文档假设的「WS」更贴合 gaea。
6. **gaea 已有文件 watch**（internal/gaea/filewatch，fsnotify 防抖 + 合并），
   而插件明确无 watcher（手动刷新）。→ 移植文件树自动刷新是 gaea 相对插件的
   原生增益，应保留并接事件流。

### 0.3 仍未通读（如实声明）

- dsh client：`state.ts`（分栏/持久化/清理算法）、`Sidebar.tsx`（双工作台与
  浮窗布局）、`EditorHost/GitView/DiffView/SubagentView/TerminalView/BrowserView/
  upload/SideCard` 各视图细节（v0.18 的 sidechat/float windows 也仅 README）。
- gaea：DocxPreview/XlsxPreview/FilePreview 编辑实现、ChatTabs/ContextView
  内部、`lib/bridge.ts` 全表、useBridgeWatch/store 尾部、浏览器 manager 内部。
  这些与「是否动右栏/预览架构」直接相关，实施某刀前应再读对应文件。

---

## 1. 任务理解

仓库是 DSH（DeepSeek Harness Web）生态里的「服务化侧边栏工作台」插件：**host 半 =
Node 进程内路由/PTY/Git/fs/任务 API，client 半 = React 侧边栏**。gaea 是
Wails(Go) + React 桌面应用，办公板块（board id `gaea`，`frontend/src/pages/GaeaPage.tsx`
壳 + `frontend/src/gaea/App.tsx` 主视图）此前在 v4.23~v4.31 已把该插件**挑着蒸馏**
成了右栏 4 tab（文件/产物/任务/浏览器）+ ChatTabs + 会话级布局外壳。

本任务的「整个」= 不再挑项，把仓库能力面全量盘点后，按「host=Go、client=React 渲染」
同构落地；「替换现在的」= 办公板块右栏由新 Go 权威底座驱动的实现取代现有 TS
蒸馏实现。

**红线（沿 2026-08-31 用户拍板，不变）**：Word/Excel 等文件编辑能力全量保留，
换壳不换芯；蒸馏纪律沿用「学交互形状与键设计，不抄代码」（仓库 MIT 许可，但
gaea 代码库既有注释均以学习借鉴落笔，保持这一风格）。

---

## 2. 调研结论

### 2.1 「整个」功能清单（以 GitHub README v0.18.0 为主，本地克隆 v0.14.1 代码佐证）

| # | 功能 | 能力要点（v0.18 README / v0.14.1 src） | host 半（待 Go 化） | client 半（React 壳） |
|---|---|---|---|---|
| F1 | 文件工作台 | 懒加载目录树（软链接按目标类型、失效标红）、全局文件名搜索、上传/拖放、右键（新 Tab/复制路径）、`@文件` 引用、树即文件窗口 | `/sidebar/api` fs 路由（list/search/read/write/upload，原子写、预算封顶、跳过 .git/符号链接目录） | FileTree/TreePanel/EditorHost/UploadOverlay |
| F2 | 文件预览器族（6 viewer） | image/pdf/markdown(+Mermaid strict+HTML 消毒+TOC)/HTML/code/binary-download；office 预览由生态插件补齐 | `/sidebar/file` 媒体路由（Content-Type 按扩展名、二进制 NUL 探测、4KB 头）、`/sidebar/html` 预览路由（CSP sandbox） | PdfView/LazyTextEditor/MermaidView 等（xterm/编辑器/mermaid 懒加载 chunk） |
| F3 | 内嵌浏览器 | 多开网页 tab，back/forward/refresh、地址栏；不透明源沙箱 iframe（无 allow-same-origin）、外链协议分流、临时解锁、沙箱状态实时显示 | 无（纯 client iframe + 媒体/预览路由安全策略） | BrowserView/sandbox 状态条 |
| F4 | 真实终端 | xterm.js + node-pty 真实 shell；断线重连回放；shell/shellArgs 可配；**可选为模型注入 terminal_\* 工具**；UI 终端上限 3、agent 终端不限、固定终端（跨会话 pin） | `/sidebar/ws/terminal` WebSocket + PtyManager + AgentPtyRegistry + agent-pty | TerminalView（xterm 懒 chunk） |
| F5 | 文件变动统一 tab（v0.18 新增） | Git 视角（真 diff/历史/暂存·提交·还原/worktree·子仓库）+ 本轮文件视角（模型读/写/编辑实时追踪，按文件分组/类型筛选）；统一 diff（改蓝配对+行内字符级高亮+语法着色+上下文折叠）、底部可拖预览、可展开独立 diff tab | git CLI（status/diff/log/stage/commit/revert，不设身份）+ 会话事件折叠 | GitView/DiffView/DiffTab |
| F6 | 后台任务页 | 子代理树实时拓扑（批量实时预览）+ 后台任务（退出码/实时输出/强制终止）；新子代理/任务自动激活任务页 | jobs API（buildJobsApi）+ subagent 枚举 | SubagentView/subagent-jobs |
| F7 | 侧边对话（beta） | Codex 式侧线程：继承主会话完整上下文（含进行中回合，interrupted 冻结）独立运行；可追问、冷恢复、保存为新会话 | sidechat 路由/事件日志折叠 | sidechat tab |
| F8 | 双工作台 | 右侧栏 + 底部面板；拖 Tab 拆分/合并分栏（可跨面板）；**自由窗口**：tab 拖到主会话区成悬浮可缩放置顶窗口（390×780 默认，拖回停靠）；移动端全宽抽屉 | 无（纯 client 布局状态，按会话持久化） | Sidebar/split-pane/open-when-sized |
| F9 | 固定终端 | 右键终端 tab 固定到工作区/全局，跨会话不消失；PTY 按 home 会话直连 | pty-manager 支持 home session | TabBar 内联虚拟 tab |
| F10 | 会话隔离/状态持久化 | 布局/Tab/面板按会话持久化，陈旧状态净化；宽度为全局键（最后一次拖拽胜出） | 无（client localStorage + session store 快照） | state.ts/prefs.ts |
| F11 | 声明式设置 | 侧边卡片逐 tab/逐 viewer 开关 + 齿轮二级设置（开关/文本/数字/下拉） | prefs 命名空间（经 fenced 路由读写） | SideCardSection/prefs |
| F12 | 懒加载/性能 | 核心 ~325KB，终端/编辑器/mermaid 用到才拉 | bundle 路由 + chunk 重验证 | chunk-loader/lazy-chunk |
| F13 | i18n | zh/en 实时跟随宿主 | 无 | locales/lang |
| F14 | 服务化/安全 | `ctx.betterSidebar.registerTab/registerFileViewer`，内置 8 tab + 6 viewer 与生态插件同 API；Host 头信任围栏；HTML/浏览器沙箱；fs 原子写与 cwd 边界 | `/sidebar/*` 全路由 fenced；settings namespace | client-registry/service |

### 2.2 gaea 现状摸底（办公板块，v4.53 后右栏 4 tab）

**已具备且已在 Go 侧有底座的（直接复用/改造）：**

- **文件树/读/写/搜索**：`GaeaListDir`/`GaeaReadFile`/`GaeaWriteFile`/`GaeaPreview`/
  `GaeaFileSearch`/`GaeaFileSemanticSearch`/`GaeaFileIndexRebuild`（internal/app +
  internal/gaea/search、fileindex、largefile、filewatch）。FileTree 已懒加载、
  展开集 localStorage 持久化、树内搜索 300ms 防抖。
- **本轮文件视角（F5 后半）**：`trajectory.FoldDeliverables` + `GaeaDeliverableRegistry`
  + 变更面板（write 类工具前后快照 → 行级红绿 diff，docx 修订基线/xlsx Plan diff）；
  ChangesDiff/VersionTimeline 已具备统一 diff 的雏形。
- **后台任务/子代理（F6）**：`tasks.Manager`（队列/实时输出/Cancel/Retry/空间分账）+
  `GaeaTaskList/Cancel/Retry/Output` + 子代理树 `AgentTree`/活动流 `SubagentsPanel`
  （Trajectory/AgentNetwork）——与 F6 对齐度高，仅差「退出码展示/强杀语义/自动
  激活任务页角标」等收口（部分已有）。
- **模型主动打开（sidebar_open）**：`internal/gaea/tool/builtin/sidebar_open.go` +
  前端解析器 + 事件接线（F14 的一部分）。
- **浏览器观察（F3 的 gaea 变体）**：`browser.Manager`（CDP 控 Edge）+ `GaeaBrowserObserve`
  + BrowserPanel（截图步进流/操作时间线/自动弹出）——这是 agent 驱动形态，不是
  用户人工 iframe 浏览；两者的关系需拍板（见 §7）。
- **上下文洞察（dsh-context 移植成果）**：ContextView + contextview 折叠 + ContextBar
  体系已经用同款「Go 折叠 + React 看板」范式跑通，是本任务的**同构先例**。
- **会话级外壳（F10/F11 部分）**：workspaceTabs.ts 全局宽度键、会话记录 v2、
  per-tab 启用集、sanitize、WorkbenchValue 键；WorkspaceTabs 齿轮设置卡已落地。

**半具/缺口的精确定位：**

| 现 tab | 已有 | 相对 better-sidebar 缺 |
|---|---|---|
| 文件 | 树/搜索/编辑器 tab 化/FilePreview（docx/xlsx/md/图片/PDF） | CodeMirror 语法高亮编辑、HTML 内联预览（sandbox）、Mermaid strict 渲染、上传/拖放、软链接语义、媒体路由式大文件、树改动 watch 自动刷新 |
| 产物（=产物+变更合并） | 权威登记表、版本时间线/恢复、写入 diff、zip | Git 视角真 diff/暂存/提交/还原/历史/worktree；统一 diff 渲染（改蓝配对/字符级高亮/语法着色/上下文折叠）；模型读/写/编辑实时追踪的独立折叠层 |
| 任务（=任务+分工合并） | 任务中心实时输出/取消/重试、子代理树/活动流 | 退出码呈现、强杀语义、新子代理自动展开的任务页联动（部分已有 subagentPrefs）、固定终端类「跨会话 pin」概念暂无 |
| 浏览器 | 受控 Edge 观察窗（截图流/时间线） | 用户人工沙箱 iframe 多开浏览、外链协议分流接管、地址栏/进退刷新 |

**全缺（不在现 4 tab）：** F4 真实终端、F7 侧边对话、F8 底部面板/自由窗口/分栏、
F9 固定终端、F3 人工浏览器 tab、F5 Git 视角、F2 的 mermaid/html/code 编辑器面。

### 2.3 「整个 Go 化」可行性判断

- **host=Go 成立**：文件/搜索/任务/子代理/浏览器观察已 Go；Git CLI 封装、
  Windows ConPTY 终端、布局/注册表状态持久化、事件推送（现有 SSE + 绑定事件）
  都可新增 Go 实现。terminal 可复用 internal/gaea/sandbox、proc（进程树杀死）、
  filewatch 的经验；真 pty 需要新依赖/新实现（见 §4）。
- **client=React 壳不可去 JS**：xterm/CodeMirror/office 编辑器是浏览器渲染技术，
  纯 Go 无意义。因此「用 Go 重写整个」落地为**能力权威进 Go**，前端只做渲染与
  交互壳——与 dsh-context Go 化、与 DSH 插件自身 host/client 分半完全同构。
- **gaea 无插件系统**：`registerTab/registerFileViewer` 的服务化理念落为
  **代码级注册清单**（现 sidebarRegistry 前端版 → Go 权威清单 + 前端 RENDERERS），
  Manifest 化留缝；第三方插件生态不在移植范围。
- **替换即收敛**：现在实现散在 App.tsx（约 1300+ 行主视图）+ workspaceTabs/
  sidebarRegistry/editorTabs/localStorage。Go 化后布局状态与业务事件有单一
  事件源，前端组件可拆薄；这是本轮最实质的工程收益，也是「替换现在的」所指。

---

## 3. 目标形态（替换后）

办公板块右栏从「4 tab + 前端私有状态」演进为「工作台 manifest 驱动的 tab 集，
数据/状态由 Go workbench 服务供给」。建议的替换矩阵：

| 现在（前端实现） | 去向 |
|---|---|
| `workspaceTabs.ts` 4-tab 清单/设置/持久化 | Go `workbench.Registry` 权威清单 + 前端薄 RENDERERS（旧文件删除或降为渲染壳） |
| `sidebarRegistry.ts` | 由 Go manifest 驱动的匹配/启停（RENDERERS 保留） |
| 文件 tab（FileTree+EditorTabs+FilePreview） | F1/F2 工作台：数据源不变绑定升级 + 补齐 code 编辑器/HTML/mermaid/上传；**docx/xlsx 编辑原样保留** |
| 产物 tab（MergedPanel 产物+变更） | F5「文件变动」统一 tab：Git 视角（新）+ 本轮文件视角（现变更/产物保留，折叠层接 Go 事件） |
| 任务 tab（MergedPanel 任务+分工） | F6 任务页：补退出码/强杀/自动激活收口，架构基本不变（改事件源） |
| 浏览器 tab（观察窗） | 拍板决定：观察窗单列，或观察窗 + 人工沙箱浏览双形态 |
| （无） | 终端 tab（F4）、固定终端（F9，拍板项） |
| （无） | 底部面板/自由窗口/分栏（F8，拍板项，量级大） |
| （无） | 侧边对话（F7，接远期规划「办公侧边对话 tab」，拍板项） |
| ChatTabs（对话/轨迹/上下文/概览） | 不动（已 Go 化） |

---

## 4. Go 侧设计（草案）

新包建议全部落在 `internal/gaea/workbench/`（单一职责、纯函数可单测，沿用
contextview 先例），绑定层在 `internal/app/gaea_workbench*.go`：

```go
// registry.go — 工作台 manifest（取代前端 workspaceTabs/sidebarRegistry 的权威层）
type TabID string // files | files-changes | tasks | browser | terminal | sidechat
type TabManifest struct {
    ID          TabID     `json:"id"`
    LabelKey    string    `json:"labelKey"`     // i18n key
    DefaultOpen bool      `json:"defaultOpen"`
    SessionScoped bool    `json:"sessionScoped"` // 会话隔离 or 全局
}
// state.go — 会话级布局状态（右栏宽/激活 tab/启用集/分栏），替代 localStorage 权威
type LayoutState struct {
    V          int                `json:"v"`
    SessionKey string             `json:"sessionKey,omitempty"`
    Tab        TabID              `json:"tab"`
    Enabled    map[TabID]bool     `json:"enabled"`
    Width      int                `json:"width"`
    Splits     []Split            `json:"splits,omitempty"` // F8 拍板后
}
// fs.go — 树/读/写/上传（软链接分类、失效标红、原子写、预算封顶、cwd 围栏）
// git.go — Git CLI 封装（status/diff/log/stage/commit/revert/worktree；不设身份；
//          路径解析以 repoRoot 为锚）→ GitView 数据 + 绑定
// pty.go — ConPTY/pty 会话（Windows 主目标）：按会话 key 注册、resize/输出订阅、
//          断线回放、agent 终端工具开关（terminal_create/terminal_write/…）
// changes.go — 本轮文件折叠（读/写/编辑实时追踪，工具调用 → 文件活动分组）
// events.go — 事件推送（复用现有 SSE/binding 事件流：tree 变更、job 输出、
//             terminal 输出走同一管线）
```

关键决策点（实现期细化）：

- **绑定策略**：新增绑定走 `GaeaWorkbench*` 前缀；每加一个绑定同步
  `bindingNames.ts` + drift 契约 + mock。终端二进制流建议走现有本地 HTTP/WS
  （main.go 已有 httpbridge/流式服务器先例）而非 Wails 绑定。经 §0.2-5 代码
  确认 httpbridge 现无 WS 服务端：终端实时通道**先做 SSE 下行 + RPC 上行**
  （/api/stream + /api/rpc 复用，Authorization 鉴权已有），WS 列为二期。
- **权限**：所有 fs/git/pty 操作按会话 cwd + workspaceFence 校验（对标插件
  trust-fence 与内置 sidebar_open 的防穿越实现）；HTML/浏览器内容沙箱由前端
  iframe 属性 + Go 预览路由 CSP 双保险。注意插件 v0.14.1 的 `fs.write` 本身
  无围栏（仅上传/媒体/HTML 路由有），gaea 移植版默认全路由加围栏（写路径
  直接复用 `GaeaWriteFile` 的 WriteRoots + 白名单 + 原子写守门，§0.1-3）。
- **进程语义**：任务「取消」现为协作式中断（queued 原子置 cancelled / running
  context 传播）；「强杀/退出码」为缺口，复用 internal/gaea/proc 与 bash 工具
  killProcessTree（§0.2-4）。
- **状态存储**：布局状态 Go 侧按会话落盘（可随 gaea 数据备份迁移），偏好键与
  旧 localStorage 键兼容读（旧值可读、不报废，沿用 sanitize 精神）。
- **依赖**：Go 1.26 + `golang.org/x/sys/windows` ConPTY（终端如进范围）；git 走
  CLI（插件同款，不引入 go-git，保持「只调 CLI、绝不设身份」）。

---

## 5. 前端替换设计（草案，文件级）

- 保留：ChatTabs、Transcript、Composer、预览 pane 体系（FilePreview/DocxPreview/
  XlsxPreview 等）、产物/变更/任务核心组件（改数据源与事件，不推倒）。
- 改：`App.tsx` 主视图拆分（工作台部分移到 Workbench 容器）、`workspaceTabs.ts`/
  `sidebarRegistry.ts` 降为 Go manifest 的渲染接线、`EditorTabs` 数据源升级。
- 新（拍板后按刀引入）：`TerminalView`（@xterm/xterm）、`GitView`/`DiffView`
  （统一 diff 渲染）、CodeMirror `TextEditor`、Mermaid 预览、上传 overlay、
  （人工）`BrowserTabs`、`SideChat`（远期）。
- i18n：页面内容层保持 zh 单语（2026 i18n 决策），tab label 沿用中文即可。

---

## 6. 分期建议（对齐版本节奏；v4.60 起排）

| 版本 | 主题 | 内容 | 拍板/备注 |
|---|---|---|---|
| v4.60 | 工作台宿主框架 | `internal/gaea/workbench` 注册表 + 布局状态 Go 持久化（旧 localStorage 兼容读）+ `GaeaWorkbench*` 首批绑定 + 现有 4 tab 数据源收敛到新事件源；绑定面 559→N₁ | 建议先行：范围明确、无新交互、纯换底座，便于后续各刀骑在权威底座上 |
| v4.61 | 终端 tab | ConPTY + xterm + 会话回放 + 设置开关（shell/shellArgs）+ agent terminal 工具注入（开关默认关） | 依赖评估：ConPTY 工作量中等；风险在 Windows 进程树/编码 |
| v4.62 | Git 与统一文件变动 | Git 视角（status/diff/log/stage/commit/revert，无 push）+ 本轮文件折叠补全 + 统一 diff 渲染升级 | Git 面板是「替换现在的变更面板」的最直接补强 |
| v4.63 | 浏览器人工 tab + 预览补全 | 沙箱 iframe 多开（若拍板）+ HTML/mermaid/code 编辑器/上传 | 观察窗保留 |
| v4.64 | 收口删旧 | 删除旧 workspaceTabs/sidebarRegistry 私有状态路径、旧键迁移完成度审计、性能/懒加载/i18n 收口 | 每刀自带测试+drift PASS |
| 远期（另案） | F7 侧边对话、F8 双工作台/自由窗口、F9 固定终端 | 量大，建议作为独立规划 | 接既有远期清单 |

每刀沿用交付纪律：Go 全量绿 + vitest + tsc/eslint 0 + drift PASS + releases/vN.md +
README 日志表；只在前一刀验收后再动下一刀。

---

## 7. 拍板清单（本次需要用户裁决）

1. **范围**：「整个」是否含 F7/F8/F9（侧边对话、底部面板/自由窗口/分栏、固定
   终端）——它们量级大且部分与 gaea「主区 = 对话 + 预览」的既有布局哲学冲突；
   建议第一轮只做「右栏工作台全量」（F1-F6 补齐 + 替换），F7-F9 另案排期。
2. **迁移方式**：直接替换现有 4 tab（一刀切）还是过渡一版双轨（新旧 tab 并存
   一版、旧值收敛后再删旧）？建议过渡不超过一个版本，v4.60 装底座、v4.64 删旧。
3. **终端**：办公板块是否真要真实 shell 终端（含 Windows ConPTY 与 agent 终端
   工具）？还是只做「任务实时输出 + 强杀」（无交互 shell）？建议真终端，开关默认关。
4. **Git**：无 push/pull/fetch（与插件一致，防意外外发）；worktree/子仓库选择
   是否首轮就要？建议首轮单仓库 status/commit/revert。
5. **布局状态**：从 localStorage 迁到 Go 持久化（随 gaea 备份/重启一致）是否确认？
   建议迁，但保留旧键只读兼容。
6. **浏览器**：右栏浏览器是维持「受控 Edge 观察窗」单形态，还是再加「人工沙箱
   iframe 多开」（对标插件 F3，用户自己逛网页）？建议观察窗先不动，F3 放 v4.63。
7. **节奏**：是否按上表 v4.60→v4.64 逐刀推进（每刀一个发布），而不是一刀全做？

若按「推荐默认」执行，请直接回复「按推荐拍板」；否则逐条给出裁决。
