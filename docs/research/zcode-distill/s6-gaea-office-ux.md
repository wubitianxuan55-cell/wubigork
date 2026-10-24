# S6 · gaea 办公板块前端现状审计（ZCode 交互蒸馏基线）

> 审计对象：`C:\AI\wubigrok`（gaea，React + TypeScript + Ant Design + Vite，办公前端在 `frontend/src/gaea`）
> 审计性质：**只读**。本文所有结论均来自实际读到的代码，行号对应当前工作区快照（2026-10-24）。
> 目的：作为「从 ZCode 蒸馏交互」的**现状基线**——先钉死"现在有什么壳/有什么数据"，再谈"还差什么"。

---

## 0. 覆盖度声明（读到哪里，哪里没读透）

| 文件 | 读法 | 覆盖 |
|---|---|---|
| `gaea/components/Transcript.tsx` | 全文 1070 行 | 100% |
| `gaea/components/Message.tsx` | 全文 463 行 | 100% |
| `gaea/components/ToolCard.tsx` / `ToolGroup.tsx` / `WorkHeader.tsx` / `ErrorCard.tsx` / `StreamingIndicator.tsx` / `TodoCard.tsx` / `AppStatus.tsx` | 全文 | 100% |
| `gaea/components/DeliverableCards.tsx` | 全文 269 行 | 100% |
| `gaea/components/DeliverablesPanel.tsx` | 全文 501 行 | 100% |
| `gaea/components/PreviewNavBar.tsx` / `ApprovalModal.tsx` / `TaskInboxPanel.tsx`（首 120 行） | 全文/节选 | 100% / 100% / ~26% |
| `gaea/components/FilePreview.tsx` | 首 140 + 440–599（渲染分支全览）+ 全文 hex/CJK 扫描 | ~60%（工具函数区未逐行读） |
| `gaea/components/DocxPreview.tsx` / `XlsxPreview.tsx` / `PptxEditPanel.tsx` | 头注 + 结构 grep + CJK/hex 扫描 | 头部契约 100%，正文细节 ~10% |
| `gaea/components/TaskCenter.tsx` | 首 120 + 关键 grep（进度/重试/取消） | ~35% |
| `gaea/components/TasksWorkbench.tsx` | 1–150 / 200–422 | ~85% |
| `gaea/components/SubagentThread.tsx` | 首 110（头注契约） | ~18% |
| `gaea/components/DagPanel.tsx` | 首 100 + CJK 扫描 | ~20% |
| `gaea/components/VersionTimeline.tsx` | 首 110（头注契约） | ~19% |
| `gaea/components/ContextView.tsx` | 首 130 | ~16% |
| `gaea/lib/store/controller_state.ts` | 全文 484 行 | 100% |
| `gaea/lib/store/controller.ts` | 全文 141 行 | 100% |
| `gaea/lib/store/controller_actions_turn.ts` / `controller_actions_memory.ts` / `store/preview.ts` / `paneFileOpen.ts` / `i18n.tsx` / `sidebarRegistry.ts` / `deliverablesTurn.ts` / `processStatus.ts` / `taskActivity.ts`（前段）/ `officeTurnProjection.ts`（前 90 行）/ `fileLinks.ts`（前 60 行） | 全文或大段 | ≥80% |
| `gaea/App.tsx`（54KB） | 1–120 / 355–474 / 620–1039 + 多点 grep | ~55%（中间布局/命令面板段按需检索） |
| `gaea/app/useDeliverables.ts` / `usePreviewAutoFront.ts`（前 120） / `useTaskCards.ts` / `useTasksAutoOpen.ts`（前 60） / `usePreviewPanel.ts`（前 80） | 全文或大段 | ≥70% |
| `gaea/styles.css` | 340–460 / 505–624 + 全文 grep | 关键类 100% |
| `gaea/redesign.css` / `src/components/module-launcher.css`（前 70） | 头部契约 | 设计语言来源 100% |
| `design-system/gaea/pages/gaea.md` / `chat.md` | 全文 | 100% |

**未纳入**：`XlsxPreview`/`DocxPreview`/`PptxEditPanel` 的内部编辑管线实现细节、`Sidebar`、`Composer`（25KB，仅从 App 调用面看）、`TrajectoryView`、`AskCard` 后半、`SubagentThread` 的消息流实现、`ContextView` 六分类细节。这些不是"办公对话流"主干，但若要做深度蒸馏需另开一刀。

---

## 1. 组件地图：办公对话页由什么拼成，数据从哪来

### 1.1 入口链（4 跳）

| 层 | 文件:行 | 内容 |
|---|---|---|
| 壳层板块清单 | `frontend/src/boards/manifests.ts:99-102` | `id:'gaea', label:'办公', page:'GaeaPage', lazy, keepAlive, layout:'full', shortcut:'ctrl+4', space:'work'` |
| 页面注册 | `frontend/src/main.tsx:19` | `registerPage('GaeaPage', lazy(() => import('./pages/GaeaPage')))` |
| 页面包装 | `frontend/src/pages/GaeaPage.tsx:12-22` | 只做三件事：`v3-zone` 容器、`LocaleProvider` 包 `GaeaApp`、import 三个 CSS |
| CSS 三件套 | `pages/GaeaPage.tsx:3-5` | `gaea/styles.css` + `gaea/tailwind.css` + `gaea/redesign.css` |
| 真正的工作台 | `gaea/App.tsx:91`（`export default function App()`） | 54KB 巨壳；已按域抽到 `gaea/app/*` 10 个 hook |

> 注：`pages/GaeaPage.tsx:7-11` 的注释把链路写明了——"办公板块：完整使用 gaeaW 原生版面（对话区 + 左侧 Sidebar + 右侧面板）"。

### 1.2 版面骨架（`gaea/App.tsx`）

```
<Layout class="gaea-app-layout">                       App.tsx:640
  <Sidebar/>                                            :655   左：会话栏（虚拟列表/项目分组）
  <section class="chat-pane">                           :685   中：对话列
    header（模型切换 / cwd / 导出 / 删除 / 面板开关）    :686-734
    <ChatTabs/>（chat | trajectory | context | memory + 子代理 tab） :742-771
    <main class="main">                                 :772
      chat       → <Transcript/> + <JumpBar/>            :790-797
                   + GenuiScope/GenuiAction Provider      :791-796
      trajectory → <TrajectoryView/>                     :798
      context    → <ContextView/>                        :799-817
      memory     → <MemoryPanel/>                        :818-837
      子代理 tab → <SubagentThread/>                     :778-787
    <footer>                                            :845
      <TodoCard/>（todo_write）                          :847-852
      <ContextPill/>（剩余上下文%）                       :855-866
      <RunStatus/>（耗时/tok/s/上下文水位）               :867-873
      <Composer/>                                        :875-890
  {previewFile && 主区大预览 <FilePreview/> + <PreviewNavBar/>}  :896-922
  {workspacePanelOpen && <WorkspacePane/>}              :924-945  右：面板注册表
  {state.approval && <ApprovalModal/>}                  :949-957  固定浮层
  {state.ask && <AskCard/>}                             :959-965  可拖拽浮卡
  <CommandPalette/> <ContextModal/> <SelectionToComposer/>       :987-1020
  <Modal>（task 卡多候选歧义选择器）                     :1025-1039
```

**易混澄清（重要）**：`gaea/components/TaskInboxPanel.tsx` **不属于办公对话页**——它住在 gaea 目录下，但被壳层首页/启动器使用：`frontend/src/components/ModuleLauncher.tsx:46,458`（`<TaskInboxPanel open={inboxOpen} .../>`）与 `frontend/src/components/TasksFirstHome.tsx:19,85`（`<TaskInboxBoard .../>`）。它用 Ant Design 组件（`TaskInboxPanel.tsx:2` `import { Tabs, Tag, Typography ... } from "antd"`），与办公页的 Tailwind + 令牌语言不同源。审计办公对话流时**不要把它算进来**。

**右栏不是硬编码分支**，而是注册表：`lib/sidebarRegistry.ts:93-136` 的 `RENDERERS`。
- `deliverables` → `MergedPanel{ primary: DeliverablesPanel, secondary: ChangesPanel }`（`:104-119`）
- `tasks` → `TasksWorkbench`（`:120-125`，内含 会话待办 / 子代理树 / 本地模型工具 / DagPanel / TaskCenter）
- `files` / `browser` / `git` / `ui`(GenUI) / `cad`（`:96-135`）
- 元数据单一源：`lib/workspaceTabs.ts`（`design-system/gaea/pages/gaea.md:15-17` 确认）

### 1.3 数据从哪来（5 条独立数据通路）

| # | 通路 | 代码位置 | 承载内容 |
|---|---|---|---|
| 1 | **对话事件流**（唯一主干） | `lib/bridge.ts:14-28` 导出 `onEvent/onReady/app`；`lib/bridge/events.ts:18` `onEvent`；`lib/store/controller.ts:40-90` `ensureEventsBound` 恰好一次绑定 | `WireEvent` 14 种 kind（`lib/types/wire.ts:6-21`）：turn_started / reasoning / text / message / tool_dispatch / tool_result / usage / notice / phase / approval_request / ask_request / turn_done / compaction_* / preview_progress |
| | 传输层 | Wails 桌面 = `window.runtime.EventsOn`；HTTP 环境 = `api/runtimePolyfill.ts:319` 注释"连接 SSE 流（fetch 流式替代 EventSource）"；纯浏览器 dev = bridge mock（`lib/bridge.ts:1-6`） | 不是浏览器原生 SSE，而是桥接层模拟的事件通道 |
| 2 | **zustand controller store** | `lib/store/controller.ts:20` `create<ControllerState>`；reducer 纯函数在 `lib/store/controller_state.ts:417-473`；`applyEvent` 在 `:286-415` | `items`（对话项数组）、`running`、`approval`、`ask`、`context`、`jobs`、`usage`、`turnStartAt` |
| | 丢件补拉 | `controller.ts:51-54`（`noteEventSeq`）→ `controller_state.ts:439-467`（`resync` 整体替换 items，不动 running） | Wails 事件吞件时的序号防线 |
| | 运行态自愈 | `controller.ts:107-116` 30s 轮询 `GaeaRunning()`，不一致即 `localCancel` | 防止 turn_done 丢失后永久"执行中" |
| 3 | **会话派生**（纯前端 useMemo） | `app/useDeliverables.ts:24-57` `sessionDeliverables`（扫正文 + 写工具 args）；`:60` `sessionChanges`；`:67-107` `freshDeliverablePaths` 角标 | 产物清单 / 变更清单 / "新产物"红点 |
| 4 | **按需拉取的后端只读视图** | `DeliverableRegistry`（`lib/deliverablesTurn.ts:133-151`、`DeliverablesPanel.tsx:104-115`）；`GaeaJournalList(200)`（`DeliverablesPanel.tsx:222-230`）；`GaeaJournalList(15)` 证据链（`:254-262`）；`GaeaSubagentRuns` + `GaeaAgentNetwork`（`TasksWorkbench.tsx:93-121` 经 `subagentRunsStore`/`agentNetworkStore`）；`GaeaDagList` 2.5s 轮询（`DagPanel.tsx:27,99-100`）；`workApp.TaskList` + `gaea-task` 事件（`TaskCenter.tsx:77-103`）；`app.Preview`（`FilePreview.tsx:111-126`） | 交付物登记表、版本/证据链、子代理、DAG、后台任务队列、文件预览 |
| 5 | **模块级注入点**（组件解耦） | `lib/taskActivity.ts:51-71`（子代理活动 provider）、`lib/paneFileOpen.ts:11-26`（打开文件回调）、`lib/genuiHost.ts`、`lib/eventSync.ts` | 让"对话流组件"不直接依赖 App 的拉取逻辑 |

### 1.4 状态机关键事实（决定了后面所有"看不到"）

- `Item` 联合体共 6 种：`user / assistant / phase / notice / compaction / tool`——**没有任何时间戳字段**（`controller_state.ts:16-24`）。
- `tool_dispatch` 建项时 `status:"running"`、无 output（`controller_state.ts:349-355`）；`tool_result` 才一次性写 `output/error/recoverable/truncated`（`:356-368`）。
- `turn_done` 失败 = 追加一条 `notice{level:'warn'}`（`:407`），并清 approval/ask（`:410`）。
- `approval_request` 是**单槽覆盖**（`:402` `approval: e.approval`），不排队。
- `phase` 是自由文本事件（`:401`），无结构化阶段 id/进度。

---

## 2. 对话流的渲染分类：现在能渲染哪些"块"

### 2.1 分层结构（Transcript 的三段式）

1. **切轮**：`buildSegments` 按 `user` 切轮（`Transcript.tsx:129-165`）。
2. **轮内交替**：`alternatingSegments` 把同轮拆成"正文段 ↔ 过程段"交替（`:83-127`）——正文出现即 flush 当前过程段；`tool/compaction/phase` 进过程段，`notice` 特意留在外面独立成行（`:117-123` 注释：notice 曾被折进过程卡导致走查发现"看不到"）。
3. **段级 memo**：`TurnBlock`（`:308`）+ `segSig`/`segCache`（`:170-178,753-771`）保证流式只重渲染最后一段。

### 2.2 块类型→组件→降级行为

| 块 | 组件（文件:行） | 视觉/行为 | 缺数据/异常时的降级 |
|---|---|---|---|
| 用户消息 | `UserMessage` `Message.tsx:74-225` | 右对齐气泡（`:134-159`）；>240 字折叠 3 行（`:128-130,143-158`）；`@.gaea/attachments/*` 解析成缩略图/文件 chip（`:25-70,113-125`）；`回退` 菜单 6 个 scope（`:209-216`），带 role=menu + 方向键漫游 | 附件 dataURL 失败 → 退化为 `[文件名]` 文本按钮（`:52-58`） |
| 助手正文 | `AssistantMessage` `Message.tsx:229-462` → `MemoMarkdown`（`:376`） | Markdown + 流式光标（`:378-380`）；子代理徽标（`:326-337`）；hover 操作：复制/沉淀技能/重新生成/👍👎（`:388-459`） | 复制失败 toast（`:398-406`）；反馈失败回滚按钮（`:284-294`） |
| 思考（reasoning） | 两条线：消息内 `Message.tsx:339-371`；过程卡内 `InlineReasoning` `Transcript.tsx:401-433` | 默认折叠；`displayReasoningText` 流式截断（`reasoningDisplay.ts`）；运行中"思考中…"+秒表（`:353-357`） | 无 reasoning 直接不渲染（`:412`）；纯思考段走 bare 模式不出卡（`Transcript.tsx:522-523,588-594`） |
| 过程卡 | `ProcessCard` `Transcript.tsx:465-628` | 一行头部：`已工作 Xs · N 个工具 · N 段思考`（`:525-531`）+ 四态徽标（`processStatus.ts:9-34`）；运行中自动展开（`:499-501`），完成自动收起（`:502-505`），用户手动操作后不再干预（`userOverridden`） | 无工具/仅思考 → bare；状态图标缺失 → 只显文字（`:536,612-617`） |
| 工具调用 | `ToolCard` `ToolCard.tsx:130-352` | 无边框行：图标 + 工具名 + 主体（bash 显 `$ 命令`，`:247-253`）+ diffstat `+N−M`（`:254-261`）+ 状态 glyph（`:45-50`）；展开后：行级 diff（`:282-291`）/ 嵌套子调用（`:293-299`）/ `$ 命令块`（`:302-306`）/ args JSON（`:308-312`）/ 有界输出 12 行 + 展开（`:313-336`） | `readOnly && 非错误` → 安静降级 `text-fg-faint/60`（`:175-176`）；output 截断标 `tool.truncated`（`:330-334`）；错误红字（`:338-342`）；`recoverable` 错误降灰（`:343-347`） |
| 重复工具折叠 | `ToolGroup` `ToolGroup.tsx:14-75`；判定 `scanGroups`（`:86-115`） | ≥2 连续同名 → 一行 `bash × 3` + 最多 3 个 subject（`:52-63`） | 运行中的调用不折叠（`skipRunning`，`:106`） |
| 阶段（phase） | 不再独立成行：`renderOutsideItems` 直接 `return null`（`Transcript.tsx:277-280`）；卡内历史 phase `:570-575`；最新 phase 上 `WorkHeader`（`WorkHeader.tsx:53-60,97,119`） | 运行中头部显示后端推的阶段文本（"正在启动引擎/正在解析 @引用/正在重试 (n/m)"，见 `WorkHeader.tsx:19-21` 注释） | 无 phase → 回退文案"思考中…"（`:119`） |
| 计划 / 待办 | **对话流内被显式剔除**：`todo_write` 在外部线 `return null`（`Transcript.tsx:271`）、在过程卡内过滤（`:344-350`）；渲染点在 (a) 输入框上方 `TodoCard`（`App.tsx:847-852`，`TodoCard.tsx:23-150`）(b) 右栏 `TasksWorkbench` 会话待办区（`TasksWorkbench.tsx:242-278`） | 折叠态摘要"进行中：xxx"、进度 n/m + 百分比条（`TodoCard.tsx:72-113`） | 关闭待办卡后**整轮不再出现**（`useTodoExtractor.ts:19-23` `dismissedTodo` 判定）；无 todo 则两处都不占位 |
| 审批 | `ApprovalModal` `ApprovalModal.tsx:48-171`（`App.tsx:949-957` 全局单例） | 五决策按钮（`:147-157`）+ 键盘 1..5（`:40-46`）+ Esc=deny（`:79-83`）；`request` 形态显示"申请规则 Tool(subject)"+模型理由原文（`:114-133`）；`schedule_apply` 换成结构化 diff 卡（`:135-145`）；硬确认工具隐藏会话/永久允许（`:63-71`） | 单槽覆盖：后到的审批直接顶掉前一个（`controller_state.ts:402`）；无超时提示；提交失败保留弹窗 + warn notice（`controller_actions_turn.ts:42-46`） |
| 提问 | `AskCard` `AskCard.tsx:10-275`（`App.tsx:959-965`） | 可拖拽居中浮卡（`:23-80`）；多问题/多选/自定义答案 | `onDismiss` 等价"空答案"（`App.tsx:963`） |
| 交付物 | `DeliverableCards` `DeliverableCards.tsx:57-269` | 回复尾部纵向列表：缩略图 + 文件名 + 相对路径；整行点击预览；hover 复制路径/外部打开/树中定位（`:162-262`）；徽标：已更新（`:172-183`）/ 草稿 / 就绪（`:186-213`）/ 未生成（`:214-225`） | `cards.length===0` 直接不渲染（`:95`）；登记表拉取失败静默降级为"仅正文卡"（`deliverablesTurn.ts:139-141`）；缺失探测失败按存在处理（`:102-114`） |
| 子代理 | 三处：徽标（`Message.tsx:326-337`）、task 卡 live 行（`ToolCard.tsx:82-128`，需 provider 注入）、完整会话（`SubagentThread`，`App.tsx:778-787` 独立 chat tab） | live 行 = 单行截断 `lastText` + `lastTool` + 已用时 + "查看分工"（`ToolCard.tsx:111-125`）；整卡可点跳转，多候选时弹选择器（`:188-196`，`App.tsx:1025-1039`） | 未注入 provider → 整行不渲染（`:100`）；ref 解析不到 → 卡片不可点（`taskActivity.ts:79-92`） |
| 错误 | 三类：`ErrorCard`（turn_done 失败，`ErrorCard.tsx:9-30`）、工具错误（`ToolCard.tsx:338-347`）、写路径失败 notice（`controller_actions_memory.ts:29-32` `failWrite`） | 红左边框卡 + 原文 + 关闭 | **全部只有"关闭"，没有"重试/换个方式"**（见 §4 G2） |
| 上下文压缩 | `CompactionCard` `Transcript.tsx:1043-1070` | pending 态 spinner"压缩中"（`:1046-1052`）；完成态可展开摘要 + `messages/trigger` meta（`:1054-1068`） | summary 空 → 展开为空 block |
| 连接兜底 | `StreamingIndicator` `StreamingIndicator.tsx:29-72` | 仅 >5s 无事件时出现顶条"仍在等待事件 + 可切轨迹面板"（`:21,53-55,67-69`） | ≤5s 完全隐藏（`:54`） |
| 工作态头部 | `WorkHeader` `WorkHeader.tsx:62-127` | 锚在最后一条用户消息后：spinner + 阶段 + 已用时 + 步数（`:113-124`）；完成后转"已完成 · 用时 · 步数"，有过程卡时让位（`:95`） | `steps===0` 的纯问答轮完成态不渲染（`:93-95`）；历史会话 `turnStartAt=0` 不渲染 |

### 2.3 一句话总结渲染分类

**能渲染**：文本、思考、工具（含 diff/命令/嵌套/有界输出/折叠组）、过程卡（四态）、阶段（仅头部一行）、审批、提问、交付物卡（带 4 种状态徽标）、子代理（徽标+live 行+独立会话）、错误（仅提示）、压缩、待办（在输入框/右栏，不在流内）。
**结构性缺口**：**没有"计划块"**（plan 只在 Composer 有开关，`App.tsx:885-886`；状态靠中文正则反推，见 §4 G4）、**没有"步骤级进度"**（无 percent 字段）、**没有"重试动作"**、**工具执行中无任何增量输出**。

---

## 3. 交付物链路：生成 → 登记 → 面板 → 预览 → 打开

### 3.1 五段链路（含每一段的失败语义）

```
① 工具写盘
   WRITE_TOOL_NAMES           lib/changes.ts:16
   + 生成/导出类 3 种          lib/officeTurnProjection.ts:29-35（format_convert / chart_gen / diagram_gen）
   + xlsx_apply              同上
        │
② 登记（三路并行，互相去重）
   (a) 正文启发式：deliverableMentions(text)          lib/fileLinks.ts（正则，含中文文件名特化 :27-42）
       → deliverablePathKey 归一（\ → /、小写）        lib/deliverablesTurn.ts:45-47
   (b) 后端权威登记表：app.DeliverableRegistry(sessionPath)
       → 只取"本轮"条目（前端 turnNo+1 = 后端 Turn）   lib/deliverablesTurn.ts:28-30,51-58
       → mergeDeliverableCards 正文在前/登记在后去重    :68-88
   (c) 会话派生清单：useDeliverables 扫 items         app/useDeliverables.ts:24-57
        │
③ 面板（右栏 "产物" tab = MergedPanel 上区）
   DeliverablesPanel                 components/DeliverablesPanel.tsx:60-501
     ├ 行：DeliverableRow             components/deliverables/DeliverableRow.tsx
     ├ 验收徽标/汇总                 :118-132 / :368-377
     ├ 自动弹出胶囊（默认关）         :381-408（偏好 gaea.deliverableAutoOpen）
     ├ 打包 zip / 复制全部路径        :334-346 / :207-215
     ├ 权威登记表折叠区               :475-483（RegistrySection）
     └ 证据链折叠区                   :486-498（EvidenceSection，含 VerifyRecord/Rollback/回plan）
   版本时间线：GaeaJournalList(200) → groupVersionsByPath → VersionTimeline
                                                    :217-234 / VersionTimeline.tsx:56-65
        │
④ 预览（两条出口）
   openPaneFileOrPreview            lib/paneFileOpen.ts:19-26
     ├ 已注册 pane 打开器 → 右栏文件 tab（App 注入）
     └ 未注册 → usePreviewStore 大预览队列  lib/store/preview.ts:31-47（队列上限 50）
   FilePreview 按 kind 分发           components/FilePreview.tsx:456-593
        │
⑤ 打开/联动
   写后自动置前（写弹读不弹）        app/usePreviewAutoFront.ts:48-58（<1240px / 专注模式不弹，:54）
   写后静默刷新（800ms 防抖）        :60-95 → requestReload → FilePreview reloadSignal（FilePreview.tsx:138-160）
   写后刷新徽标                     FilePreview.tsx:278（data-testid="office-auto-refreshed"）
```

### 3.2 用户实际看到什么（按时间顺序）

1. **生成中**：对话流里一张 `write_file` / `docx_*` 工具卡，`running` spinner，**没有输出、没有进度、没有耗时**（见 §4 G1）。
2. **生成完成**：该轮尾部（轮尾段）出现**交付物卡列表**（`DeliverableCards`）——缩略图 + 文件名 + 相对路径 + 徽标（`已更新`/`草稿`/`就绪`/`未生成`）。同时若开启自动弹出，右栏切到"产物"tab 并给该行"新"高亮（`useDeliverables.ts:92-100`）。
3. **点一下卡片**：右栏开**文件 tab**（或大预览），按扩展名渲染：
   - docx → `DocxPreview`（docx-preview 保真渲染 + 框选即改 + 修改队列 + 大纲；`DocxPreview.tsx:56-78` 契约）
   - xlsx → `XlsxPreview`（含"待应用变更"计划预览，`XlsxPreview.tsx:728-760`）
   - pptx/pdf → 逐页缩略图 + `PptxOutline` + `PptxEditPanel`（`FilePreview.tsx:467-527`）
   - md/txt/html/image → Markdown / 进度计划摘要卡 / 沙箱 iframe / img（`:528-580`）
   - 不支持 → "在外部程序中打开"按钮（`:581-593`）
4. **多文件**：底部 `PreviewNavBar` chip 条（队列 >1 才出现，`PreviewNavBar.tsx:33`），支持中键关闭。
5. **回看**：产物面板每行可展开**版本时间线**（`vN` 徽标 → 逐版本：预览/与当前对比/恢复；`VersionTimeline.tsx:56-65,110+`），可做**验收标记**（已验收/要求修改），可看**证据卡**（声明↔实况 diff、复核 verdict、回滚）。

### 3.3 链路上的"不诚实点"（已自证）

- **派发即登记、失败不回剔**（`lib/deliverablesTurn.ts:98-114` 注释明说）→ 写失败也会留一张卡。
- 缺失判定**只对登记-only 卡**做，且"探测失败按存在处理"（`:102-114`；`DeliverableCards.tsx:133-135`）→ 正文提及的卡永远不标"未生成"。
- 会话产物面板（`DeliverablesPanel`）**不显示缺失态**，只显示验收/版本。

---

## 4. 交互缺口：用户看不到 / 看不懂 / 要手动做什么

> 每条给"现象 → 代码证据 → 谁该补数据"。

### G1 长任务执行期：只有 spinner，没有耗时、没有增量输出

- **现象**：跑一个 3 分钟的 bash 或生成 60 页 docx，工具卡从头到尾一行 spinner，直到 `tool_result` 才一次性出现输出。
- **证据**：
  - `Item.tool` 无任何时间字段：`lib/store/controller_state.ts:16-24`。
  - ToolCard 作者自己承认计时只能靠挂载时刻近似，且只给 `task` 卡做了：`components/ToolCard.tsx:85-87`「Item 契约无时间戳，以挂载时刻近似计时」。
  - 事件契约里没有"工具输出增量"：`lib/types/wire.ts:38-49`（`partial` 仅表示"只带 name 的早期 dispatch"，不是流式输出）。
  - reducer 一次性落 output：`controller_state.ts:356-368`。
  - 唯一的整体计时在轮级：`WorkHeader.tsx:99-124`（已用时 + 步数）。
- **要补**：后端在 `tool_result` 前增发 `tool_progress`/`tool_output_delta` + `startedAt`；前端 ToolCard 加"已用时 / 当前动作"行（模板已存在于 `AgentTree.tsx:252-267` 的指标行）。

### G2 失败无重试：错误只能"关掉"

- **现象**：turn 失败/工具失败后，用户只能自己重新打字；`重新生成` 只对**最后一条** assistant 出现。
- **证据**：
  - `ErrorCard` 只有 X：`components/ErrorCard.tsx:18-28`。
  - 工具错误只有红字：`ToolCard.tsx:338-347`。
  - 重生成资格 = 最后一条非流式 assistant：`Transcript.tsx:914-923`。
  - 写路径失败 notice 文案只说"请重试"，没有动作：`controller_actions_memory.ts:31`。
  - **对照**：`TaskCenter` 有完整 cancel/kill/retry（`TaskCenter.tsx:145-180,518-532`）；`PptxEditPanel` 有 `pptx-edit-retry`；`DocxPreview` 队列"失败可单独重试"（`DocxPreview.tsx:66-71`）。→ 重试原语**已存在，只是没接到对话流**。
- **要补**：错误块上加"重试该工具/重跑该轮"（后端需要 `RetryTool(callId)` 或复用 `regenerate` 语义）。

### G3 审批信息不足：一个 tool 名 + 一段 subject，其他都没有

- **现象**：模型要写 8 个文件，审批卡只说 `write_file` 和一句话；用户不知道要写哪些文件、改什么、有没有超时、后面还有几个审批在排队。
- **证据**：
  - 契约只有 4 个字段：`lib/types/wire.ts:70-79`（id/tool/subject/request/reason）——无参数快照、无超时、无队列 id。
  - 单槽覆盖、后到顶掉先到：`controller_state.ts:402`；turn_done 直接清空：`:410`。
  - 除 `schedule_apply` 外不渲染任何结构化预览：`ApprovalModal.tsx:135-145`（其他工具只有 `<pre>{subject}</pre>`，`:140-144`）。
  - 审批期间输入框被禁：`App.tsx:888`（`disabled={... || state.approval != null}`）→ 用户唯一的动作就是拍板。
  - 位置固定悬浮：`styles.css:539-547`（`position:fixed; bottom:176px`），窄屏再挪位（`:549-581`），与待办卡/输入区同区竞争。
- **要补**：审批负载带 args 结构化摘要（写类=路径列表 + diffstat；命令类=命令块）；超时倒计时；审批队列（`approvals: []` 而非单槽）。

### G4 计划模式/计划内容不可见，且状态靠中文正则反推

- **现象**：用户按了"计划"开关后，界面上没有任何"当前处于计划模式"的持久标识，也没有"计划内容"块；只有一条 notice 文本。
- **证据**：
  - `planActive` 是从 notice 文本里正则匹配出来的：`App.tsx:372-381`（`/计划模式已开启|已处于开启状态/`、`/已退出计划模式|计划模式未开启/`）——**换语言/换文案即失效**。
  - 计划开关只是发一句 `/plan on`：`App.tsx:886`。
  - `todo_write` 不进对话流：`Transcript.tsx:271,344-350`；待办卡可被永久关闭：`useTodoExtractor.ts:19-23`。
- **要补**：结构化 `plan` item（steps + 状态），或把 `todo_write` 提升为一等"计划块"渲染在流内（数据已在 `parseTodos(items)`）。

### G5 上下文 / 子代理 / 任务队列：都要"切一个面"才看得到

- **现象**：右栏默认关；上下文是主区一个 tab；子代理要开右栏任务页；后台任务队列在右栏最底部。
- **证据**：
  - 右栏默认关 + 自动展开默认关（用户 2026-09-27 拍板）：`app/useTasksAutoOpen.ts:5-7,27-31`。
  - 上下文视图是 `chatTab === "context"` 的主区 tab：`App.tsx:799-817`；流内只有输入框上方一个百分比 `ContextPill`：`App.tsx:855-866`。
  - 任务面板需 `workspacePanelOpen` 才挂载：`App.tsx:924-945`。
  - TaskCenter 未接 `sessionPath`，"本会话"过滤 chip 恒不显形：`TasksWorkbench.tsx:412-417`（注释自认欠账）。
- **要补**：在对话流内提供"一行式"上下文/队列摘要（Codex 的 footer status line 语义），而不是要求用户切面板。

### G6 长任务无进度百分比（子代理与 DAG 有数据但不在主流里）

- **现象**：子代理跑 10 分钟，主窗只有一行被截断的"最后一段文本"。
- **证据**：
  - 数据结构只有 `lastText/lastTool/toolCalls`，没有 percent/step/total：`lib/types/subagent.ts:13-23`。
  - 主窗 live 行单行截断：`ToolCard.tsx:111-125`；未注入 provider 时**整行不渲染**：`:100`。
  - 但 `AgentNode` 已有 `firstTs/lastTs/toolCalls/tokens/errors`：`lib/types/agent.ts:9-22`；`AgentTree` 已经把"工具×N / 耗时 / ≈tokens / err N"排成指标行：`AgentTree.tsx:252-267`。
  - DAG 有节点级状态（含"待审批"）：`DagPanel.tsx:29-41`。
- **要补**：把 AgentTree 的指标行语义搬进过程卡/task 卡；`WireTool` 增 `parentId` 已有（`wire.ts:48`），可做"父调用聚合进度"。

### G7 静默窗口与"无阶段"时的空转感

- **现象**：首 token 之前只有一行"思考中…"，>5s 才出现等待条；用户无法判断"是在排队、在压缩、还是在重试"。
- **证据**：
  - 无 phase → 固定文案：`WorkHeader.tsx:119`。
  - 等待条 5s 阈值 + 只有一句话：`StreamingIndicator.tsx:21,53-55,67-69`。
  - 压缩中只有一个小 spinner：`Transcript.tsx:1046-1052`。
- **要补**：阶段词表结构化（排队/思考/工具/压缩/重试 n/m）+ 阶段耗时。

### G8 交付物"登记成功 ≠ 文件存在"对用户不透明

- **现象**：卡上有个文件，点开报"读取文件失败"，但卡片仍正常显示（正文提及路径的卡不标缺失）。
- **证据**：`deliverablesTurn.ts:98-114`（派发即登记、失败不回剔、探测失败按存在处理）；`DeliverableCards.tsx:133-135`（只对 registry-only 标缺失）；`FilePreview.tsx:121-124`（读失败 → kind:"error"，`:581-593` 才显示）。
- **要补**：统一"产物状态"字段（registered/written/verified/missing），面板与卡片同源展示。

### G9 i18n：对话流干净，**预览/办公文档面大量中文直出**

- 统计（脚本剔除注释后含 CJK 的行数）：`XlsxPreview.tsx` **58**、`DagPanel.tsx` **53**、`DocxPreview.tsx` **45**、`FilePreview.tsx` **38**、`PptxEditPanel.tsx` **31**、`VersionTimeline.tsx` **20**、`TasksWorkbench.tsx` **8**、`ContextView.tsx` **4**；对照对话流主干 `Transcript.tsx` **0**、`WorkHeader.tsx` **0**、`ErrorCard.tsx` **0**、`DeliverableCards.tsx` **0**。
- 具体样例：
  - `Message.tsx:356`：`${reasoningLines} 行 · ${elapsedStr}`（思考行 meta 硬编码中文）。
  - `ToolCard.tsx:68`：`… 已折叠 ${folded} 行`。
  - `controller_actions_memory.ts:31`：`${what}失败：…，请重试`（what 是中文常量，见 `controller_actions_turn.ts:20,29,34,45,50`）。
  - `AppStatus.tsx:49`（后台任务完成 toast）、`:105`（即将强制压缩）、`:137`（执行中）。
  - `TodoCard.tsx:66,75,83-88,92`（待办/全部完成/进行中：/收起待办）。
  - `FilePreview.tsx:445,451,495,503,532,561,584,590`（OCR/加载中/第N页/截断提示/在外部程序中打开）。
  - `DagPanel.tsx:34-41`（待跑/运行中/完成/失败/跳过/已验收/待审批）。
  - `VersionTimeline.tsx:37-38` 注释自认"页级文案与本组件既有中文直出口径一致，未新增 i18n 键"。
- **i18n 机制本身是好的**：三语 zh/en/zh-TW，zh 静态、其余懒加载 chunk（`lib/i18n.tsx:18-43`），回退链 zh→en→裸键（`:73-80`），无 Provider 场景有 `useTOptional`（`:149-153`）。缺口在**覆盖率**不在机制。

### G10 其它（次要但会咬人）

- 审批卡固定 `bottom:176px`，与待办卡/输入框叠区风险：`styles.css:539-547` + `App.tsx:845-890`。
- `ProcessCard` 四态里 `stopped` 的语义对用户是"已停止"，但 `turn_done` 也会把仍在 running 的工具统一置 `stopped`（`controller_state.ts:406`）→ 用户分不清"我取消的"和"后端没回结果的"。
- 会话恢复时 running 工具一律还原 `stopped`（`controller_state.ts:247-256`），无"这一轮被中断"的显式提示（只有工具行 Ban 图标）。
- `DeliverablesPanel.tsx:318` 注释保留历史遗留：`NAVIGATE page:"gaea"`（旧 id "office" 已被壳层白名单丢弃，`DeliverablesPanel.test.tsx:522,574` 钉死）。

---

## 5. 视觉与工程约束

### 5.1 「玄墨 v11」与本板块的真实关系（重要澄清）

- **玄墨 v11 的原产地是首页**：`frontend/src/components/module-launcher.css:1-34` 定义了"五条对策"——① 三层层级（L1 hero/L2 工作层/L3 仪表带）② 描边退场（边线 4.5%/6% + 表面色阶 + 阴影令牌）③ 金色三处点睛（hero 顶线 / 命令条 focus / 等宽数据值）④ 排印三档（display/名称/标签，消灭 11-12.5px 满屏）⑤ 栅格对缝 gap 14px。
- **推广范围**：v4.521（角色库 + 青鸟）、v4.522（记忆中枢 + 设置中心）"方言板块清零"（`CHANGELOG.md:7-11`）；`releases/v4.521.0.md:7` 明确"办公/首页/小说/轻语近两月已相继收官"。
- **办公板块当前的语言不是玄墨，而是三层混合**：
  1) 对话流是 **Codex 式**（去边框工具行、过程条、完成即收起）：`Transcript.tsx:73-77,481-484,596-600`；
  2) 右栏面板是 **v3「星枢」细条头**：`DeliverablesPanel.tsx:53`（"v3 星枢面板语言：v3-panel-head 细条头部 + 低边框 hover 高亮行"）；
  3) 壳层是 **Luminous Glass 2.0 / Constellation OS v3**（玻璃只做容器、顶部 1px 高光 `--v3-edge`、柔光 `--v3-glow-soft`）：`redesign.css:1-9`。
- **结论（含推断，标注如下）**：**事实**是——办公板块自己的 CSS 声明的语言是上面三套，且 v4.521/v4.522 两轮玄墨推广的 release note 只列了角色库/青鸟（`releases/v4.521.0.md:12,33`）与记忆中枢/设置（`releases/v4.522.0.md:4,11`），没有办公；**推断**是——办公板块未被玄墨化。若"从 ZCode 蒸馏交互"要顺带统一视觉，办公板块属于**需要被玄墨化的一方**，不能反过来说办公已遵循玄墨。玄墨的"描边退场/金色三处/排印三档"与办公现状（大量 9~11px 微字、accent 光条、hover 高亮）存在系统性冲突，需单独立项。

### 5.2 令牌与零硬编码 hex 纪律

| 检查 | 结果 | 证据 |
|---|---|---|
| 主题变量注入 | 壳层统一注入 `--md-sys-color-*` / `--gaea-glow` / `--accent-rgb` / `--v3-fg-soft` 等 | `frontend/src/App.tsx:106-152`（`:147` 注入 `--gaea-glow`；`:126` 补 `-error` 别名） |
| 办公 TSX 硬编码 hex | **基本为零**：Transcript/Message/ToolCard/ToolGroup/WorkHeader/ErrorCard/DeliverableCards/DeliverablesPanel/DeliverableRow/EvidenceSection/FilePreview/XlsxPreview/PptxEditPanel/PreviewNavBar/ApprovalModal/TaskCenter/TasksWorkbench/SubagentThread/DagPanel/VersionTimeline/TodoCard/AppStatus **全部 0 处**；`DocxPreview.tsx` 1 处、`ContextView.tsx` 8 处（其中 3 处标 `hex-exempt`，其余为 `var(--md-sys-color-warning, #f59e0b)` 形式的 fallback） | 本次扫描（脚本：去注释后 `#[0-9a-f]{3,8}` 计数） |
| `styles.css` 的 17 处 hex | **全部是 `var(..., #hex)` 的兜底值**（`:17-41`），不是直出颜色 | `gaea/styles.css:17-41` |
| `redesign.css` | 0 处 hex | 同上扫描 |
| 回归锁 | `src/contrast-hex.test.ts:5-43`（具体文件不得出现 `#555/#ddd/#374151` 等硬编码深灰；tertiary 必须走 `ThemeTokens`） | — |
| 已知欠账 | `design-system/gaea/pages/gaea.md:40` 落地项「清组件内硬编码色值（存量核查）」**未勾选** | — |

### 5.3 测试契约：改结构/改文案会红

- **测试量级**：仅 `gaea/components` 下就有 100+ `*.test.tsx`；`data-testid` 是结构契约（279 处命中，办公对话面关键的有）：
  `task-live`（ToolCard:105）、`work-header`（WorkHeader:105）、`subagent-badge`（Message:329）、`office-turn-phase-draft/ready`（DeliverableCards:188,202）、`deliverable-acceptance-summary` / `deliverable-auto-open-toggle`（DeliverablesPanel:370,383）、`deliverable-registry(-row)`（RegistrySection）、`session-todos` / `subagent-section-toggle` / `tasks-workbench-retry`（TasksWorkbench:243,282,231/318）、`agent-thread*`（SubagentThread:91,349,485…）、`office-auto-refreshed` / `pptx-pages`（FilePreview:278,477）、`docx-queue*`（DocxPreview:882-1008）、`pptx-edit-*`（PptxEditPanel:258-460）、`dag-section(-toggle)` / `dag-retry`（DagPanel:393,396,498）、`vcompare-*`（VersionTimeline:94,288,307）。
- **源级断言**（不渲染，直接读 `App.tsx` 源码字符串）：`App.palette.test.ts:12-35`——命令面板项的字面量、图标、关键词、`NAVIGATE(page="schedule")` 都被钉死。
- **语义级断言**（行为契约）：
  - 过程卡折叠语义：`ProcessCard.test.tsx:21-41`（完成态 `aria-expanded=false`；运行中 `true`）、`:56-69`（纯思考走 bare，只有 1 个折叠头）。
  - 审批卡中文文案 + 五决策顺序：`ApprovalModal.test.tsx:17-73`（"允许这次工具调用吗？"、"申请理由（模型提供）"、`[deny, allow_once, allow_session, persist_allow, abort]`）。
  - 交付物导航 id：`DeliverablesPanel.test.tsx:522,574`。
- **对蒸馏的含义**：任何"把中文文案换成 i18n 键""把审批卡加字段""把工具卡加耗时行"的动作，都需要同步改测试；反过来，这些测试就是**现有交互形状的冻结层**。

### 5.4 i18n 三语

- 语言集：`en | zh | zh-TW`（`lib/i18n.tsx:18`）；字典体量 zh 91.9KB / en 94.5KB / zh-TW 92KB（`locales/*.ts`）。
- 加载策略：zh 静态内置，en/zh-TW 首次切换时 `import()` 独立 chunk（`:26-43`）；未加载时 `translate` 逐级回退（`:73-80`）。
- 非 React 场景：模块级 `currentLocale` 镜像 + `t()`（`:47,84-86`）；无 Provider 用 `useTOptional`（`:149-153`）。
- 测试均显式 `localStorage.setItem("gaea-lang","zh")` 后再断言中文（`ProcessCard.test.tsx:12`、`ApprovalModal.test.tsx:9-11`）。
- **覆盖缺口见 §4 G9**。

---

## 6. 可复用性判断：哪些壳已就绪、只差数据

> 判据：组件本身是否已具备"呈现 + 交互 + 状态语义"，只缺后端字段/接线。

| 组件 | 现状能力 | 只差什么 | 可承载的 ZCode 交互 |
|---|---|---|---|
| **ProcessCard** `Transcript.tsx:465-628` | 四态、自动展开/收起、耗时+工具数+思考段数、子调用分组、bare 模式 | ① per-item 时间戳 ② percent/step-total 字段 | 「步骤流 / 计划执行面板」——把每个计划步骤画成一条过程卡行 |
| **ToolCard** `ToolCard.tsx:130-352` | 命令块、行级 diff、diffstat、嵌套子调用、有界输出+展开、错误分级、只读降噪、整卡可点跳转 | ① 增量输出事件 ② 工具开始/结束时间 | 「可观测的工具执行」——实时 stdout、耗时、重试按钮 |
| **AgentTree 指标行** `AgentTree.tsx:252-267` | 模型/工具×N/耗时(live)/≈tokens/错误数 | 无（模板已完整） | 把这段指标行搬进 ToolCard/ProcessCard 即得"子代理进度条" |
| **DagPanel** `DagPanel.tsx:29-53,393-498` | 节点级状态（含"待审批"）、起跑/单跑/重跑/改向/终止/验收/存模板 | 只缺"由 agent 计划驱动"的接线 | 「计划步骤可干预」——每步重跑/跳过/改向 |
| **TaskCenter** `TaskCenter.tsx:105-180,369-538` | **唯一完整闭环**：进度条 percent + 取消 + 强制终止 + 重试 + 输出尾(dock) + 事件即推/轮询兜底 + 状态三态（queued/running/stopping） | 数据源是 `workApp.TaskList`（非会话 store）；未接 sessionPath 过滤 | **最成熟的"长任务"范式**——抽象成 agent 任务模型可直接复用 |
| **DeliverableCards** `DeliverableCards.tsx:57-269` | 双路合并（正文+登记）、4 种徽标、路径归一去重、跨段防重复 | ① 产物内进度/分片 ② 统一状态字段 | 「产物生命周期卡」——生成中→草稿→就绪→已验收 |
| **VersionTimeline** `VersionTimeline.tsx:56-65,110+` | 逐版本列表、与当前对比（text/docx/xlsx/pptx 四类 diff）、恢复、diffstat | 无（能力过剩） | 「产物版本审阅/回滚」 |
| **ApprovalModal** `ApprovalModal.tsx:34-171` | 五决策 + 快捷键 + hardAsk 分级 + reason 原文 + schedule diff 特化 | ① args 结构化摘要 ② 超时 ③ 队列（后端单槽） | 「审批中心」——多待办审批 + 详情面 |
| **SubagentThread** `SubagentThread.tsx:70-82` | 直接复用 `AssistantMessage` + `ToolCard`（证明主对话渲染件可跨场景复用）+ 流式 subagent_text + 追问 + 提升为主会话 | 无 | 「子会话详情」——现成的第二对话流 |
| **ContextView / ContextModal** | 六分类构成、水位线、趋势、AgentRadial 拓扑、请求时间线、文件活动 | 无 | 「上下文可视化」——只需入口前移（现在藏在 tab 里） |
| **TodoCard** `TodoCard.tsx:23-207` | 进度条 + 三态 + 分组 + 当前任务高亮 + 折叠摘要 | 位置（应在流内）+ 结构化计划语义 | 「计划块」（轻量版，零后端改动即可前移） |

**一句话结论**：办公板块**呈现件成熟度 ≫ 事件契约成熟度**。几乎所有"更丰富的 agent 交互"都卡在同一个地方——`WireEvent`/`Item` 缺三样东西：**时间戳、步骤/进度、增量输出**。补上这三样，§4 的 G1/G2/G6/G7 大部分可由现成组件承载；G3/G4/G5 则主要是"接线与布局"问题（数据已有，位置不对）。

---

## 7. 给"从 ZCode 蒸馏"的落地建议（按投入产出排序）

1. **先扩契约**（后端）：`WireTool.startedAt/finishedAt`、`tool_progress{percent,label}`、`tool_output_delta{chunk}`、`plan{steps[]}`、`approvals[]`（队列化）。零 UI 改动即可解锁 4 个缺口。
2. **复用 TaskCenter 范式**：把它抽象为 `<LongTaskCard>`，让 agent 的 `bash/docx_*` 长任务走同一套"进度+取消+重试+输出尾"。
3. **把待办前移**：`TodoCard` 已有全部视觉，移进 `renderOutsideItems` 的 `notice` 同级即可，不必等新契约。
4. **审批升级**：先用现有 `args` 在卡内做结构化渲染（写类=路径列表+diffstat，命令类=命令块）——纯前端可做，参考 `schedule_apply` 的 `ScheduleDiffCard` 特化路径。
5. **i18n 补齐**：按 §4 G9 的文件清单逐个体化，**必须同步改测试**（`ApprovalModal.test.tsx` 等钉死中文）。
6. **视觉统一**：玄墨化办公 = 独立项目（三套语言并存：Codex 对话 / 星枢面板 / Luminous Glass 壳），不建议与交互蒸馏同一刀。

---

## 附录 A：本报告引用的文件清单（按路径）

- 壳/入口：`frontend/src/boards/manifests.ts`、`frontend/src/main.tsx`、`frontend/src/pages/GaeaPage.tsx`、`frontend/src/App.tsx`
- 办公壳：`frontend/src/gaea/App.tsx`、`frontend/src/gaea/app/*.ts`
- 对话流：`gaea/components/{Transcript,Message,ToolCard,ToolGroup,WorkHeader,ErrorCard,StreamingIndicator,TodoCard,AppStatus,AskCard,SubagentThread}.tsx`
- 交付物：`gaea/components/{DeliverableCards,DeliverablesPanel,VersionTimeline,PreviewNavBar}.tsx`、`gaea/components/deliverables/*`、`gaea/lib/{deliverablesTurn,deliverableStatus,verifyDiff,versionTimeline}.ts*`
- 预览：`gaea/components/{FilePreview,DocxPreview,XlsxPreview,PptxEditPanel,DocxOutline,PptxOutline,FileVersionStrip}.tsx`、`gaea/hooks/usePreviewProgress.ts`
- 任务：`gaea/components/{TasksWorkbench,TaskCenter,TaskInboxPanel,DagPanel,AgentTree}.tsx`
- 状态/数据：`gaea/lib/store/{controller,controller_state,controller_actions_turn,controller_actions_memory,preview}.ts`、`gaea/lib/{bridge.ts,bridge/events.ts,i18n.tsx,sidebarRegistry.ts,taskActivity.ts,paneFileOpen.ts,officeTurnProjection.ts,fileLinks.ts,processStatus.ts,changes.ts}`
- 样式/设计：`gaea/styles.css`、`gaea/redesign.css`、`components/module-launcher.css`、`design-system/gaea/pages/{gaea,chat}.md`
- 测试：`gaea/components/{ProcessCard,ApprovalModal,DeliverablesPanel,Message,WorkHeader,TasksWorkbench,ContextView}.test.tsx`、`gaea/App.palette.test.ts`、`frontend/src/contrast-hex.test.ts`

## 附录 B：本次审计的"未证实项"（不要当结论用）

1. `AgentTree` 的耗时 `dur.label` 具体计算方式（只看渲染行 `:252-267`，未读上半文件）。
2. `ContextView` 六分类各自的准确口径（只读首 130 行 + `context/cats.ts` 引用）。
3. `DocxPreview` / `XlsxPreview` / `PptxEditPanel` 的编辑写回具体绑定（只看头注与 testid）。
4. `paneEnabled` 开关对"点文件开 pane vs 大预览"的实际分流条件（未读 `lib/paneEnabled.ts`）。
5. `SubagentThread` 的追问/提升为主会话完整交互（只看头注与 testid 列表）。
6. `XlsxPreview` 的 `plan → XlsxApplyEdit` 审批语义是否走 `ApprovalModal`（只看到预览侧计划区）。
