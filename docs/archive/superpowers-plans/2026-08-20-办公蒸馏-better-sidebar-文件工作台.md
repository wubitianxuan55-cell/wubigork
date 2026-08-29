# 办公板块蒸馏 dsh-better-sidebar · 文件工作台（2026-08-20）

> 本轮目标：把 [omdsh-dev/DSH-better-sidebar](https://github.com/omdsh-dev/DSH-better-sidebar)
> （侧边栏完整工作台插件，已克隆到 `.tmp/dsh-better-sidebar`）的「文件工作台」交互范式
> 蒸馏进 gaea 办公板块右侧「工作区」文件树。
>
> 蒸馏原则（项目惯例）：**聚焦现有、入口按上下文收敛、不堆功能、零后端改动优先**。
> 插件 7 tab + 6 viewer 中，gaea 已具备：文件预览（多文件队列/前后导航）、最近文件快捷区、
> 资料分组面板、任务中心、子代理分工列表、Markdown Mermaid、布局持久化。本轮只补最痛的
> 文件树交互差距。

## 差距分析（vs 插件 FileTree / TreePanel）

| 能力 | 插件实现 | gaea 现状 | 本轮 |
|---|---|---|---|
| 行内 @ 引用 | 行悬浮「@文件」按钮 → 输入框 | 无（只有资料面板行有 @） | ✅ 加 |
| 右键菜单 | 预览 / 新 Tab / 侧边 / 下载 / 上传 / 复制相对/绝对路径 | 无 | ✅ 加（预览/外部打开/在文件夹中显示/复制相对路径） |
| 复制路径反馈 | 复制后按钮变「已复制」1.2s | 无 | ✅ 加 |
| 目录展开态 | 会话级状态持久化 | 组件内 useState，刷新即丢 | ✅ 按 cwd 持久化 localStorage |
| 树内搜索 | host 递归文件名搜索（预算封顶） | 无（跨库检索在搜索面板，语义向） | ✅ **C8 升级**：接已有 `GaeaFileSearch`（工作区递归文件名搜索、深度/数量封顶、跳过噪音目录），非空查询显示命中列表（插件 TreePanel 同款模式），空查询显示树 |
| 刷新 | TreePanel 刷新按钮清缓存 | WorkspacePanel 头部 refreshKey 重挂载 | 已有，不动 |
| 终端/Git/子代理拓扑/上传 | 插件特色 | 办公场景不适用 / 需后端数据 | 不做，记录到后续候选 |

> 实施中追加：**C3 会话隔离·右面板 Tab 记忆**（纯前端，对齐插件「布局按会话持久化」的轻量近似）——
> `rightTab` 记住上次选择，localStorage 单 key（`gaea.workspace.rightTab`，`isWorkspaceTabId` 校验、
> 非法值收敛回「文件」），见 `lib/workspaceTabs.ts` 的 `loadPersistedRightTab` / `savePersistedRightTab`
> 与 `App.tsx` 接线。本轮做全局版（办公单会话场景够用）；按 sessionPath 的细粒度版记入候选清单 C3。

## 实现设计

### 1. `frontend/src/gaea/components/FileTree.tsx`（核心改造）

新增 props（全部可选，向后兼容）：

```tsx
onReference?: (rel: string) => void;     // 行内 @ 引用
onOpenExternal?: (rel: string) => void;  // 右键：在外部程序中打开
onReveal?: (rel: string) => void;        // 右键：在文件夹中显示
```

- **行悬浮 @ 按钮**：文件/目录行（含根行）hover 显示 Paperclip 小按钮（aria-label「引用到输入框」），
  点击 `stopPropagation` 后调 `onReference(path)`。对齐插件 rowActions。
- **右键菜单**：antd `Dropdown trigger={["contextMenu"]}` 包行元素（项目已依赖 antd ^5.21.6，
  Sidebar 已用 Modal）。菜单项：
  - 文件行：预览（`onSelect(path)`）· 在外部程序中打开（有 `onOpenExternal` 才显示）·
    在文件夹中显示（有 `onReveal` 才显示）· 复制相对路径
  - 目录行：复制相对路径
  - 复制走 `navigator.clipboard?.writeText`（Sidebar 同款可选链），成功后行尾显示「已复制」
    标签 1.2s 替换 @ 按钮（对齐插件 copied 反馈）。
- **展开态持久化**：展开集提升到 FileTree 顶层 `expanded: Record<string, boolean>`，
  按 cwd 存取 localStorage key `gaea.fileTree.expanded.<cwd>`，条目数上限（如 500）防膨胀，
  损坏/越权静默回退；根行默认展开。刷新（refreshKey 重挂载）后恢复。
- **树内搜索（C8，接 `GaeaFileSearch`）**：树顶搜索框（300ms 防抖，非空时显示清空按钮）。
  非空查询 → 调 `app.FileSearch(needle, 50)`（工作区递归文件名搜索：名称子串、不区分大小写、
  深度 ≤6、跳过 .git/node_modules/dist 等噪音目录、数量封顶——对齐插件 fs.search 预算纪律），
  渲染命中列表取代树：文件命中点击 `onSelect`（预览）、目录命中展示不可点（title 提示），
  行尾 @ 按钮同样可用；无命中/失败显示提示；底部提示「搜索范围：整个工作区」。空查询 → 树。
  搜索命中上限 `FILE_SEARCH_LIMIT=50`（服务端钳制 + 前端二次限）。

### 2. `frontend/src/gaea/components/WorkspacePanel.tsx`（接线）

传三个新回调（内部实现）：

```tsx
onReference={(rel) => {
  useComposerInsertStore.getState().requestAt(rel);
  recordRecentFile(rel);                    // lib/recentFiles 去重置顶
  toast.show(`已引用 @${rel}`, "info");
}}
onOpenExternal={(rel) => void app.OpenWorkspacePath(rel).catch(() => {})}
onReveal={(rel) => void app.RevealWorkspacePath(rel).catch(() => {})}
```

## 验证

- `frontend`: `tsc -b` 0 errors；`eslint .` 0 errors；`vitest run` 全绿（FileTree.test.tsx
  新增：@ 引用点击 / 右键复制路径（clipboard mock）/ 展开态持久化 / 搜索过滤，旧加载三态用例不回归）。
- 可选 `vite build` 通过。

## 不做（记录为后续候选）

- 任务实时输出/退出码：需后端 TaskView 增输出流字段，另开。
- 子代理拓扑图：需 GaeaSubagentRuns 提供父子关系，另开。
- 终端 / Git 面板 / 文件上传：办公场景不适用或超出零后端边界。
