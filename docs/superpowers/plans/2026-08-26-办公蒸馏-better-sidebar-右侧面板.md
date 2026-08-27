# 办公板块蒸馏 dsh-better-sidebar · 右侧面板改造（2026-08-26）

> 本轮目标：把 [omdsh-dev/DSH-better-sidebar](https://github.com/omdsh-dev/DSH-better-sidebar)
> （已克隆 `.tmp/dsh-better-sidebar`）的右侧面板工作台范式，蒸馏进 gaea 办公板块右侧面板
> （4 主 Tab：文件 / 成果 / 运行 / 分析）。承接 2026-08-20 蒸馏候选清单 C1-C9——
> 本轮只做**纯前端、S 工作量、直接落在右侧面板本体**的三项，需后端字段的 C1/C2、
> 预览表面重活的 C4/C9、写文件权限的 C5 全部留后续轮次。
>
> 蒸馏原则（项目惯例）：聚焦现有、入口按上下文收敛、不堆功能、零后端改动优先。

## 范围

| # | 特性 | 插件参考 | gaea 现状 | 本轮改动 |
|---|---|---|---|---|
| C3 | 会话级右侧面板布局持久化 | `state.ts` 每会话一份 `SidebarState`，`localStorage "dsh-sidebar:v1:<id>"` + `sanitizeState` 校验回退 | `rightTab` 是**全局单 key**（`gaea.workspace.rightTab`），切会话不恢复各自的面板 Tab | 升级为按会话 key（`gaea.rightPanel.v1:<sessionKey>`）：会话切换恢复各自子面板，非法值回退默认；无会话 key 时回退全局 key（向后兼容） |
| C6 | 运行域活动角标 | `Sidebar.tsx` badge（99+ 封顶）+ `subagent-jobs.ts` detectNewJob | 4 主 Tab 无角标，新任务无前台提示 | 「运行」主 Tab 显示活跃任务计数角标（queued/running，99+ 封顶），点进运行组即清除；纯前端（`onTaskEvent` 已有） |
| C7 | 预览队列中键关闭 + 可点条 | `TabBar.tsx` 中键关闭（按下记录、弹起同目标才关、preventDefault autoscroll） | `PreviewNavBar` 只有 ← index/total → | 升级为文件 chip 列表：点击切换、× 关闭、中键关闭；store 增加 `closePreviewAt` |

## 不做（记录）

C1 任务实时输出（需后端 `GaeaTaskOutput`）、C2 子代理活动行/拓扑（需后端字段）、
C4 选区转对话（M，预览表面改造）、C5 工作区内联编辑（需 `GaeaWriteFile` 权限）、
C9 分栏对照（L）。终端 / Git / 内嵌浏览器 / HTML 沙箱延续既有排除结论。

## 实现设计

### 1. C3 会话级布局持久化

`frontend/src/gaea/lib/workspaceTabs.ts`：

- `loadPersistedRightTab(sessionKey?: string)`：key = `gaea.rightPanel.v1:<sessionKey>`
  （有 sessionKey 时）；无 sessionKey → 回退既有全局 key `gaea.workspace.rightTab`
  （读取旧值后写入新 key 属迁移，本轮不做，仅读取兼容）。
- `savePersistedRightTab(tab, sessionKey?)`：按上述 key 写入；非法值/配额失败静默。

`frontend/src/gaea/App.tsx`：

- `rightTab` 状态初始化与保存都改为感知 `currentSessionKey`：`currentSessionKey`
  变化（切会话/新建/恢复）时加载该会话的 rightTab；切换时保存到当前会话 key。
- 保持既有交互不变：点文件回「文件」面板等显式切换照常覆盖当前会话记忆。

### 2. C6 运行域活动角标

`frontend/src/gaea/components/WorkspaceTabs.tsx`：

- 新增可选 prop `badges?: Partial<Record<WorkspaceGroupId, number>>`；「运行」
  （running）组 Tab 按钮旁渲染小计数 pill（99+ 封顶，`>0` 才显示）。
- pill 样式走令牌（`--md-sys-color-primary`/`--gaea-glow`），不硬编码色值。

`frontend/src/gaea/App.tsx`（或独立 hook `useRunningBadge`）：

- 订阅 `onTaskEvent`（已有封装）：维护活跃任务计数（`queued`/`running` 增，
  `succeeded`/`failed`/`cancelled` 减），初始 `TaskList()` 拉一次基线。
- 仅当「运行」组未激活时显示角标（激活即视为已读，天然清除）。
- 子代理运行数本轮不纳入角标（避免 App 层新增轮询；SubagentsPanel 内已有 5s 轮询）。

### 3. C7 预览队列中键关闭 + 可点条

`frontend/src/gaea/lib/store.ts`：

- `PreviewState` 增加 `closePreviewAt(index: number): void`：从 `previewList` 移除该项，
  调整 `previewIndex`/`previewFile`（移除当前项时跳到相邻项；空队列关闭预览）。
- 纯函数逻辑放 `lib/previewQueue.ts` 便于单测（复用既有 `previewQueue.test.ts` 风格）。

`frontend/src/gaea/components/PreviewNavBar.tsx`：

- 升级为文件 chip 条：遍历 `previewList` 渲染 basename chip（活动项高亮），
  点击 `onJump(index)` 切换、× 按钮 `onClose(index)` 关闭。
- **中键关闭**：`onMouseDown` 记录目标 index（button===1 且 preventDefault 掉
  autoscroll），`onMouseUp` 仍在同一 chip 才 `onClose`（对齐插件 TabBar 语义）。
- 单文件队列不渲染（保持既有行为）。

`frontend/src/gaea/App.tsx`：`PreviewNavBar` 传 `files={previewList}` / `index` /
`onJump` / `onClose`。

## 验证

- 前端 `tsc -b` 0 errors；`vitest` 新增：C3 key 选择（含回退）、C6 角标渲染与
  99+ 封顶、C7 `closePreviewAt` 边界（首/中/尾/唯一/空）+ PreviewNavBar 中键语义。
- 既有 605+ 用例零回归；`workspaceTabs.test.ts` 更新为按会话 key 断言。
