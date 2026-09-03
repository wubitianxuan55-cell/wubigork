# 工作树快照记录：办公右栏 dsh-better-sidebar pane 化移植（2026-09-03，未发版）

> 状态：已完成一版可运行的 pane 移植，前端全量门禁绿；**未 bump 版本、未发布**。
> 续会话：已再完成「左栏子代理会话入口」（快照二）与「产物行开 pane 文件
> tab」（快照三，见下）；仍未 bump/发布。
> 提交即快照，供下个会话继续。

## 〇〇、快照三：产物行打开语义统一（2026-09-03 续会话）

用户确认「文件打开统一开成 pane 文件 tab」：产物视图内产物行/权威登记行 +
正文交互卡 + 变更面板，全部从「收起工作台开大预览」改为 **在右栏 tab 条
新增/激活文件 tab**（与资源管理器同一套 tab，可多个并存、同路径去重）。

- 改动（第一步，4a0cae7a）：`lib/sidebarRegistry.ts` 新增 `openPaneFileTab`
  统一入口，产物视图 `DeliverablesPanel.onOpenFile` 改挂它（files 视图
  openFileTab 同步复用）。
- 改动（第二步，本提交）：新增 `lib/paneFileOpen.ts` 模块级注入
  `openPaneFileOrPreview`（App 注册 openPaneFile，未注册回落大预览，沿用
  setTaskCardActivityProvider 模式）——正文交付卡 DeliverableCards、行内附件
  Message/InlineAttachment、FileLinkText 文件 chip、Markdown/MemoMarkdown
  文件链接统一走它；sidebarRegistry 里变更面板 onOpenFile 也改挂
  `openPaneFileTab`（展开区「打开文件预览」同语义）。
- 边界：左栏不参与；无 App 注册的页面/单测自动回落旧预览通道，零行为回归。
- 验证：sidebarRegistry +2 用例（产物行/变更行 → pane file tab 且不回落大
  预览回调）；vitest 全量 **210 文件 / 1525 用例**、tsc/eslint 0、drift
  559 OK；`?mock=1` 实拍：正文卡点开 → 右栏 1 文件 tab；变更行 → 产物视图 +
  新文件 tab（docs/snapshots/2026-09-03-better-sidebar-port/ 下
  deliverable-click-file-tab.png / body-file-click-opens-tab.png /
  changes-open-file-tab.png）。

## 〇、快照二：左栏子代理会话入口（2026-09-03 续会话）

上一会话遗留的「下一刀」已完成（复用现有 `GaeaSubagentRuns`，绑定面保持
**559 不变**，不新增后端接口）：

1. **父会话行可展开**：Sidebar 虚拟滚动行模型加 `subagent` / `subagentNote`
   两种子行——会话行左侧新增展开钮（aria-expanded + data 锚点），首次展开
   懒加载该会话子代理，关闭再开走缓存；loading/空态/失败重试均有行内态。
2. **子代理子行**：层级缩进 + 状态点（运行脉冲/完成/失败）+ 任务摘要 +
   「状态 · 模型 · 相对时间」次行；点击子行 → 既有「独立子代理会话 tab」
   （`openSubagentThread`，不替换主会话）。
3. **当前会话运行中 5s 刷新**：与右栏分工面板同数据源同口径；历史会话子行
   按展开时快照展示。
4. **mock 按父会话归属**：c.jsonl（当前）保原两跑；a.jsonl / d.jsonl /
   annual-r1 各有演示子代理；其余会话空态——左栏各会话子代理不再串味。
5. **i18n**：三语各 +7 键（展开/收起/打开/空态/加载/失败/无题兜底）。

涉及文件：`components/Sidebar.tsx`（+行模型/懒加载/渲染）、
`components/Sidebar.test.tsx`（+2 用例）、`App.tsx`（透传 openSubagentThread）、
`lib/mock/office.ts`（SubagentRuns 按会话归属）。

验证：
- vitest 全量 **210 文件 / 1523 用例**通过（+2）；`tsc -b`、eslint 全量 0；
- 绑定面 drift：`bindingNames.ts` 与 Go 一致（559）；
- `?mock=1` 实拍：展开当前会话 → 2 个子代理子行 → 点子行 → 主区顶栏出现
  独立子代理会话 tab（标签=任务摘要），见
  docs/snapshots/2026-09-03-better-sidebar-port/sidebar-subagents-expanded.png
  与 .../sidebar-subagent-tab-open.png。

## 一、本快照完成内容

1. **pane 工作台语义（对标 dsh-better-sidebar）**
   - 右栏无 tab = 欢迎卡；点卡片开对应「视图 tab」；
   - 资源管理器是其中一种视图 tab；视图内点文件 → 新增「文件 tab」；
   - 同一 tab 条混排视图 tab 与文件 tab，可关、可 `＋` 新开、按会话持久化。
2. **视图补齐**
   - 文件 = ExplorerView（点文件 → pane 文件 tab）；
   - 浏览器 = 地址栏 + 沙箱 iframe（URL 随 pane tab 持久化）；
   - 产物/任务 = 复用既有面板能力。
3. **任务页重做**：紧凑「子代理拓扑 + 后台任务」连续单页，去掉轨迹式重复。
4. **点子代理 → 独立子代理会话 tab**：主区顶栏动态 tab（可关闭/并行切换）。
5. **清理旧两级组件**：删除 WorkspaceTabs / WorkspacePanel / EditorTabs /
   lib/editorTabs 及其测试。

## 二、主要文件

新增：
- frontend/src/gaea/lib/paneTabs.ts（+test）
- frontend/src/gaea/lib/browserPolicy.ts（+test）
- frontend/src/gaea/components/WorkspacePane.tsx（+test）
- frontend/src/gaea/components/ExplorerView.tsx
- frontend/src/gaea/components/BrowserPane.tsx
- frontend/src/gaea/components/TasksWorkbench.tsx

修改：frontend/src/gaea/App.tsx、components/ChatTabs.tsx、lib/sidebarRegistry.ts

删除：components/WorkspaceTabs(.tsx/.test)、WorkspacePanel(.tsx/.test)、
EditorTabs(.tsx/.test)、lib/editorTabs(.ts/.test)

## 三、验证

- 前端全量：vitest 210 文件 / 1521 用例通过；`tsc -b`、eslint 全量 0。
- `?mock=1` 实测链路：欢迎卡 → 文件视图 tab → 文件 tab；浏览器沙箱加载
  example.com；任务单页；点子代理开独立会话 tab。
- Go 侧零改动（未跑全量 Go）。

实拍：docs/snapshots/2026-09-03-better-sidebar-port/

## 四、下一步（下个会话）

- ~~左栏子代理会话入口（用户已选 2）~~：**已完成（快照二）**——复用
  `GaeaSubagentRuns` 保持绑定面 559，Sidebar 虚拟列表子行 + mock 实拍验收。
- ~~产物行打开语义~~：**已完成（快照三）**——产物行开 pane 文件 tab。
- 之后可考虑：完整发版（v4.60.0 流程：Go 全量 → 版本四处 → release note →
  提交/tag）。
