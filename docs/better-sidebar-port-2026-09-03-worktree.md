# 工作树快照记录：办公右栏 dsh-better-sidebar pane 化移植（2026-09-03，未发版）

> 状态：已完成一版可运行的 pane 移植，前端全量门禁绿；**未 bump 版本、未发布**。
> 提交即快照，供下个会话继续。

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

- **左栏子代理会话入口（用户已选 2）**：
  1. 后端：新增 `GaeaSubagentSessions(parentSessionPath)`（或把 SessionMeta
     扩展为会话树）暴露 `<会话目录>/subagents/*`；
  2. 前端：Sidebar 虚拟滚动行模型加「子行」（父会话展开 → 子代理行），
     点击走已完成的独立子代理会话 tab；
  3. Go drift（绑定面）+ vitest + mock 实拍验收。
- 之后可考虑：完整发版（v4.60.0 流程：Go 全量 → 版本四处 → release note →
  提交/tag）。
