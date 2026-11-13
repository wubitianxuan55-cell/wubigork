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

## 5. 续刀（完善批）：sheet 遗留清退 + 文案归一 + mock 图纸 fixture + mock 实机走查

- **刀A CadPanel 单形态收口**：variant="sheet"/inspectorHost/createPortal
  （§7.3 外置检查器宿主）三机制清退——右栏一级 Tab 后面板唯一形态=右栏纵向
  栈；portal.test 删除；cad-panel.css 重写（三态 --pane/--sheet/--inspector-rail
  并为单一纵向布局，空态卡/分区线/raster 适配收编；原 pane 形态竟是裸排无样式）。
- **刀B 文案归一**：三 locale 的 home.toolCad/welcome.toolCad 「CAD 看图→CAD
  图纸」（en 补真翻译 CAD Drawings），home.toolsSub「临时面板→右栏一级标签」，
  welcome.toolCadDesc 语义改写（右栏直达）；paneTabs 视图 id 注释枚举补
  cad/schedule。
- **刀C mock 完善**：mock PickFiles 对纯 CAD 过滤（dwg/dxf/pdf）回 fixture
  （空过滤/混合过滤仍空返，不污染附件流）——浏览器 mock 可全链演示 CAD 面板。
- **刀C2 mock 实机走查（vite 5199 + ?mock=1，browser-use）**：欢迎网格 6 卡
  （CAD 关）→ 管理 Popover 7 行（CAD 开关默认关）→ 开关启用 → 网格出 CAD 卡
  → 点卡开 Tab → 导入 → mock fixture 装载（JS-01-一层平面图/清晰度 1.00/
  分幅 raster/文本清单（26）/符号统计按钮）→ 问图纸 → mock 识图答案 → 发到
  对话 → 对话流收到【图纸问答：plan.dxf｜JS-01——一层平面图】消息；＋下拉
  菜单 7 项无未启用标注（cad 已启用）；进度计划 Tab 同开，横道图/质量体检/
  完整工作台渲染正常，与既有文件/交付 tab 并存。截图存档。
- **坑 6：浏览器 dev 不带 ?mock=1 时 bridge 走 HTTP RPC**（/api/rpc 404 假象
  ——「打开图纸失败：RPC 请求失败: 404」），mock 走查必须 ?mock=1。
- **坑 7：IAB locator 在高频重渲页面上 actionability 检查反复失败**（count()=1
  但 click 超时，force 也不达），改 playwright.evaluate 直驱点击+断言后走查
  顺利完成；会话 pane 快照回灌（关 tab 后自动还原）使 welcome 态需经快照
  清 tab 才能到达。
- **并发纪律新实践：locale 三文件混有 DSH 在途改动**，提交用 git diff 按
  hunk 过滤（marker=toolCad/toolsSub/右栏/right-panel tab）+ `git apply
  --cached --recount` 只暂存本刀 2 hunks/文件（MM 态），DSH hunks 原样留树。
- 门禁：tsc 绿；定向 9 文件 92 例绿（cadOpen/workspaceTabs/sidebarRegistry/
  paneTabs/App.palette/cad 全套/mock contract）；ESLint 0。
- 装机：提交 2f5aa8e3（10 文件 +86/−242）干净工作树重建（pnpm 20.6s/Vite
  38.24s/wails 1m），装机位 exe sha256 96C46474…（build 11/14 01:24:34）=
  工作树产物，smoke /api/health 200（PID 14820 已回收），工作树已回收。

## 6. 续刀二（完善批）：CAD 直达链收口——cadExt 叶模块 + 快照净化 + 树内走查实证

- **cadExt 叶模块**：isCadFilePath/CAD_FILE_EXTS 抽到零依赖叶（paneTabs 净化要
  用而 cadOpen 又依赖 paneTabs，反向导入成环；cadOpen 再导出保持 API 不变）。
- **paneTabs 快照净化**：dwg/dxf 文件 tab 是坏预览（打开链已统一落 CAD 面板），
  sanitize 丢弃旧快照残留的这类 tab，active 收敛末位；补 2 测试例。
- **sidebarRegistry 集成测试**：产物行 dwg → 断言不开文件 tab、开 cad 视图 tab
  + 可选卡自动启用 + 请求总线投递（此前只有单测锁路由函数，无接线级验证）。
- **mock 树 fixture**：根增 `图纸/`（结构平面图.dxf + 电气-自建房.dwg）；坑=树
  展开传给 ListDir 的 rel 无尾斜杠，分支须两形态都接。
- **mock 实机走查（确定性复跑）**：文件树点 结构平面图.dxf → 无坏文件 tab、
  「CAD 图纸」Tab 激活装载（JS-01 图框/分幅/最近图纸 chip=结构平面图.dxf）、
  零报错。此前两轮"失败"全是走查姿势竞态：CAD 视图激活后树被切走（后续找
  树行的点击全落空）+ mock 编辑触发 HMR 重载重置面板开合——非应用 bug。
- **FileTree 行为实证**：树行左键=onSelect（openFileTab→guard），「预览」主区
  入口只在右键菜单（FileTree:855），无第二分发通道要拦。
- 门禁：tsc 绿；定向 8 文件 64 例绿；ESLint 0。
- 装机：提交 e02e1d4f 干净工作树重建（pnpm 18.5s/Vite 40.31s/wails 54.8s），
  装机位 exe sha256 4256587E…（build 11/14 01:42:15）=工作树产物，smoke
  /api/health 200（PID 12608 已回收），工作树已回收。

## 7. 续刀三：顶栏「工具」下拉退役

- 用户拍板「删除顶栏上的工具按钮，cad与进度计划放到右侧面板了」：App 顶栏
  工具 Dropdown（office-tools-menu）整块删除——CAD 图纸/进度计划已是右栏一级
  Tab，入口由欢迎网格/＋下拉/命令面板/文件打开链四处派生；失效图标导入
  （Wrench/Layers/LineChart）与 antd Dropdown 导入、office-ui.css 高亮规则
  一并清退。输入框旁工具钮（ComposerToolbar onOpenTool）保留为快捷入口，
  注释改写；App.palette/Composer 族 42 例绿，tsc/ESLint 干净。
- 装机：提交 801cac31 干净工作树重建（pnpm 17.2s/Vite 40.60s/wails 51.8s），
  装机位 exe sha256 755BD2D6…（build 11/14 01:50:08）=工作树产物，smoke
  /api/health 200（PID 47784 已回收），工作树已回收。

## 8. 续刀四：CAD/进度计划视图激活自动加宽（窄栏适配）

- 窄栏度量（340px 默认宽）：甘特视图横向溢出 ~950px、工具条/图框条全靠横向
  滚动硬看——内容本质宽，滚动感差。
- 方案：激活沿（rightTab→cad/schedule）复用 v4.27 handleAutoWidenWorkspace
  自动加宽到 560（视口收敛封顶）。**每会话一次性**（autoWidenDoneRef 不重置）：
  此后用户拖窄不回弹，不抢「最后一次拖拽胜出」的宽度偏好语义。
- 实证：mock 重置宽度 340 → 激活进度计划 → grid 列 264/660/560、pane 560
  稳定（DOM 级）；状态层会话记录 width:560、CSS var 560 双证。
- 坑 8：dev 期编辑源文件触发 HMR 换 i18n 模块实例，懒 chunk 拿到新 context
  对象 → 整页「useI18n must be used within a LocaleProvider」假崩（重启 dev
  server 后不复现，60s 长窗零错误）——产物单 bundle 无此问题；走查遇整页
  i18n 崩先查 HMR 时序再怀疑代码。
- 坑 9：mock 环境会话/视口/面板开合被周期性重置（running 会话事件+更新横幅），
  截图式走查不稳，DOM 度量（单格内激活+采样）才是可靠姿势。
- 门禁：tsc/ESLint 0；App.palette/cadOpen/workspaceTabs 43 例绿。
