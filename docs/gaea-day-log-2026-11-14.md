# gaea Day Log · 2026-11-14 · CAD/进度计划回归右栏一级 Tab + 办公打开 CAD

> 用户拍板：把办公板块的 CAD 识图与进度计划改到右侧面板成为独立标签，办公板块
> 可以打开 CAD。ToolSheet 临时面板承载退役，纯前端刀（零 Go 改动、零新绑定）。

## 1. 刀路总览

- **清单**（`frontend/src/gaea/lib/workspaceTabs.ts`）：`WORKSPACE_TAB_IDS` 扩为
  7 项，新增 `cad`（「CAD 图纸」，`defaultEnabled:false` 可选使用红线沿用）与
  `schedule`（「进度计划」，默认启用）。命令面板 / 欢迎网格 / ＋下拉三入口由
  注册表 + paneEnabled 启用集自动派生，零入口改动。
- **注册表**（`lib/sidebarRegistry.ts`）：RENDERERS 以 `React.lazy + Suspense`
  接线 `CadPanel`（pane 变体即本源形态）/ `ScheduleTool`，保持既有代码切分；
  `WorkspacePanelContext` 恢复 `onAsk`（App 接 `send`）——「问图纸 / 交 AI 算量 /
  问这个计划」从右栏直发对话并收焦点。
- **App**（`gaea/App.tsx`）：退役三件套——`activeTool` 状态、`ToolSheet` 渲染
  分支、§7.3 检查器 rail 宿主（`cadInspectorHost` + 开合快照 effect，pane 变体
  依据面本就内联）。统一落点 `openToolTab(tool)`：停用先启用（与＋下拉「点击
  即启用」同语义）再 `openPaneView`。消费方重接：顶栏工具下拉、Composer 工具
  钮（经 `onOpenTool` 单源）、OPEN_TOOL 监听、首页 tile 冷启补开
  （OPEN_TOOL_FLAG）、命令面板（显式 `cmd-schedule`/`cmd-tool-cad` 退役——
  注册表循环派生的 `cmd-cad`/`cmd-schedule` 自带未启用标注）。
- **办公打开 CAD**（新 `lib/cadOpen.ts`）：dwg/dxf 打开链统一路由。
  `usePreviewPanel.openPaneFile`（工作区文件树/正文卡）与
  `sidebarRegistry.openPaneFileTab`（资源管理器行/交付卡/Git 变更行）双口拦截：
  `isCadFilePath` 判定（pdf 不在内，走文档预览链）→ 启用可选卡 → 请求总线
  `useCadOpenBus.requestOpen` → 开 `view:cad` tab。CadPanel 按 nonce 一次性消费
  （busy/ask 在途暂存总线不丢、忙态翻转重跑消费；消费即清，卸载重挂不重放——
  最近图纸列表仍可一键重开）。dwg/dxf 不再开 pane 文件 tab 坏预览。
- **退役/删除**：`ToolSheet.tsx` + `ToolSheet.test.tsx` 删除；行为注释同步改新
  语义（ScheduleFileCard/ToolCard/ScheduleTool/cadToolCache/CadPanel/events.ts）；
  `docs/gaea-ai-workbench-ui-2026-10.md` 头部加 2026-11-14 修订注（§7 ToolSheet
  叙述为史档）；`.gaea/progress.md` 立档。

## 2. 测试

- 新 `lib/cadOpen.test.ts` 5 例：扩展名契约/大小写、可选卡自动启用、视图 tab、
  总线投递 + nonce 递增、非 CAD 路径零副作用。
- 契约锁改口径：`App.palette.test.ts` 重写（注册表派生 + openToolTab +
  onAsk:send 源级断言）；`workspaceTabs.test.ts`（7 Tab 清单 + cad 可选卡默认关
  + 全禁兜底例外）；`sidebarRegistry.test.tsx`（cad/schedule 注册项 + 同源豁免）。
- 定向：cad 全套（含 portal/cache/race）+ schedule + WorkspacePane + Composer
  族 + ScheduleFileCard + ToolCard + app hooks = 15 文件 92 例绿；契约 4 文件
  50 例绿。
- 全量 vitest 4414 例（新增 5）两轮完整日志复核：本刀零交集定性——
  `WorkbenchPrefsPanel` 4 红实为 DSH 在途改动（同树并发把 locale
  `tasksAutoOpen` 文案改「新任务自动切**编队**视图」，其测试未跟上，非本刀）；
  `ContextView.test` 同因（DSH 正在改 context/fleet 族）；`DatabaseLibrary`
  首轮 5 红→复跑绿（与 knip 并发抢 CPU 的负载期假红，在册先例）。本刀触及面
  的定向 142 例全绿。

## 3. 坑册

1. **源级断言 split 锚点**：循环体提取不能按首个 `}` 切——模板字面量
   `${reg.label}` 的闭 `}` 在循环体前半段就出现，`split('}')[0]` 拿不到 run 块；
   锚定缩进闭合行 `\n    }\n` 才稳。
2. **负向源级断言勿对注释开火**：`not.toContain('ToolSheet')` 会被史注命中
   （退役记录本该写进注释）；改断 `not.toMatch(/import \{ ToolSheet/)` +
   `not.toMatch(/<ToolSheet/)`。
3. **全量套件勿与重活并发**：knip/eslint 与 vitest 全量并行抢 CPU 会产负载期
   假红（DatabaseLibrary 首轮 5 红复跑全绿）；定性纪律=零交集+复跑绿，且重跑
   时独占 CPU 并完整落盘日志。
4. **pane 视图快照向后兼容白吃**：`paneTabs.VIEW_IDS` 派生自 WORKSPACE_TABS——
   cad 回归清单后，v4.524 时代持久化的 `view:cad` 快照自动恢复合法，无需迁移。

## 4. 验证门禁与装机（干净树 fd9caa06）

- `tsc -b` 绿；ESLint 改动面 0；knip 无新增欠账（余项均为在册 wailsjs 生成件
  噪声）。
- 规约七执行：本刀 21 文件提交 `fd9caa06`（未 push）；`.worktrees/cadtab-fd9caa06`
  独立干净工作树构建——pnpm install 26.6s → `npm.cmd run build`（Vite 39.57s，
  dist 落仓根）→ `wails build -s -ldflags "-s -w" -trimpath`（1m0.5s，ucrt64
  gcc + CGO_ENABLED=1，bindings 未跳过）。
- **坑 5：`gaea_install.ps1` 的 `$SourceDir` 默认 `C:\AI\wubigrok\build\bin`**
  ——从工作树跑脚本装的是主树旧 exe（首轮 INSTALL OK 但 build time 23:17 与
  构建时刻 00:57 对不上，sha256 52D605F2… ≠ 3C0CDF4D…）；必须显式
  `-SourceDir <worktree>\build\bin`。重装后装机位 sha256 3C0CDF4D…=
  工作树产物，build time 11/14 00:57:28，DLL 三件同装。
- 干净树 exe 直跑冒烟遇策略拦截（在册），一律走装机位：`scripts\smoke.ps1
  -ExePath C:\AI\gaea-app\gaea.exe` → `/api/health` 200（PID 50232 已回收）。
- 二进制标记：装机 exe `grep -ac GaeaCadOpen` ×3 命中（绑定面在位）。
