# gaea AI 工作台 UI 设计（2026-10）

> 状态：设计稿 v2 · 2026-10-25
> 范围：办公 / AI 工作板块的壳层与工作对象界面。
> 边界（用户 2026-10-25 拍板）：**CAD 看图与进度计划不单独做工作台**。它们只是办公板块里的工具，由 AI 或用户在对话里调用，打开为**临时工具面板**；不占导航、不占页签、不另造一套工作流。

---

## 0. 一句话结论

办公板块是**唯一工作台**：左边是工作对象，中间是对话/轨迹/上下文/记忆，右边是随当前对象变化的检查器。
CAD 与进度是**动词**——出现在输入框工具菜单、AI 工具卡、产物行里；点开后在对话上方浮出**临时工具面板**，关闭即回到对话，结果留在产物。

---

## 1. 市场调研（2026-10）

### 1.1 AI 工作台：容器变薄，输出物变实

| 产品 | 观察 | 对本设计的含义 |
|---|---|---|
| Cursor 3「Glass」（2026-04） | 拆掉聊天面板，改为 Agents Window：任务卡带 Planning/Executing/Reviewing/Done、触碰文件、diff 预览，可暂停/重启/分叉。 | 工作台第一信息是**任务状态**；工具是任务里的动作，不是并列页面。 |
| ChatGPT 桌面端（2026-07） | 合并通用对话与 Codex 后标签栏导致历史难找，官方承认「过渡版体验糟糕」；计划移除 Work 标签，把代理能力融回原生界面。 | **不要为每个能力加标签/工作台**。这是本次边界最直接的证据。 |
| Manus 1.6 | 首页是任务 hub；聊天内 artifact chip 打开文档查看器；文件/任务详情收在菜单；模型选择是顶栏 pill。 | 产物用 chip/行承载；低频动作收进菜单，不铺成一级入口。 |
| Claude 桌面端（2026-10 改版争议） | 正文对比度从 10.5:1 冲到 14.3:1 产生眩光，冷灰替代暖白，侧栏比正文区更亮；分段控件被下拉替代、说明文字被删。 | **安静的阅读面**是生产力；侧栏后退、暖 off-white、常用控件可见。 |
| Microsoft 365 Copilot / Office（2026） | 同一年里从「并排画布」摆回「聊天内嵌块/委派」，共同点是容器变薄、输出物变真（文件、可下载、带版本）。 | 工具面板应是**临时容器**；产物必须落成真实文件/版本，而不是一个常驻模块。 |

### 1.2 CAD 看图：画布优先，面板可停靠

| 产品 | 可借的形状 |
|---|---|
| Bluebeam Revu | 菜单栏 + 动态属性工具栏 + 左右/底部可停靠面板 + 导航栏 + 状态栏（坐标/比例）。 |
| DWG FastView | 模型/布局切换、图层冻结隔离、文字搜索跳转、测量记录、表格转 Excel、修订对比。 |
| Togal.AI | 上传后自动命名/整理图套；绿色按钮一键识别；AI 对话直接问图纸、核对数量；结果可导出。 |

**对 gaea 的含义**：图纸工具 = 图套/图层 + 画布 + 状态栏 + 一个 AI 动作（问这张图/算量）。它需要的是**一块临时画面**，不是办公里的第二个工作台。

### 1.3 进度计划：角色分层，不是更花的甘特图

2026 年 7 款进度软件横评的共同结论：选型先看计划复杂度与治理，不看功能数量；共同缺口是「更新成本」与「偏差解释」。
对 gaea：进度工具 = 甘特/网络图 + 基线/关键线路 + AI 体检/推演。它是**一次检查动作**，不是常驻排程台。

### 1.4 参考链接

- Cursor 3 评测：https://www.openaitoolshub.org/en/blog/cursor-3-agent-first-review
- ChatGPT 界面争议：https://news.qq.com/rain/a/20260730A05AQY00
- Manus 交互地图：https://github.com/HomenShum/NodeBenchAI/blob/main/docs/chat-logs-unified/MANUS_INTERACTION_MAP.md
- Claude 改版实测：https://github.com/anthropics/claude-code/issues/104695
- Bluebeam 界面：https://support.bluebeam.com/it/online-help/revu20/Content/RevuHelp/Welcome/Getting-Around-the-Revu-Interface--MTV.htm
- DWG FastView：https://gstarcademy.com/tutorial-fastview
- Togal 功能：https://www.togal.ai/features
- 进度软件评测：https://worktile.com/solution-1/archives/230724

---

## 2. 现状诊断（办公板块）

以 1440×900、mock 数据走查：

1. **CAD 被做成了「一等公民」**：它在右栏 `WORKSPACE_TABS` 里与文件/产物/任务平级，占一个 Tab。用户要先找到这个 Tab，再在 340px 窄栏里完成导入→图框→看图→问答→统计，图纸被压到 38vh。
2. **进度被做成了独立板块**：`SchedulePage` 是完整页面，从办公进去就「离开对话」，回来时上下文已断。
3. **工具与对话是两条路**：工具在右栏/独立页，AI 在中间；工具结果要再发一次对话，「谁调用了什么、依据在哪」没有同屏链路。
4. **右侧是模块列表，不是检查器**：文件/产物/任务/浏览器/Git/UI/CAD 并列，和当前这一段对话没有关系。
5. **入口是能力矩阵**：首屏 12 张等权模块卡，用户先选「哪个工作台」再开始工作；CAD/进度被当成「房间」。

---

## 3. 设计原则

1. **一个工作台**：办公就是工作台；不新增 CAD/进度板块、页面、一级 Tab。
2. **工具是动作**：CAD 看图、进度计划只出现在输入框工具菜单、AI 工具卡、命令面板、产物行里。
3. **工具面板是临时的**：打开时浮在对话上方，带「工具」标识与关闭；Esc/遮罩/关闭钮都能退回对话，不改变当前会话。
4. **检查器随工具**：工具打开时右栏切到该工具的属性/依据；关闭后回对话上下文。
5. **产物归档**：工具结果落成真实文件（摘要 md、统计 csv、计划文件），可在「产物」重开工具或下载。
6. **安静的表面**：侧栏比画布暗；暖 off-white 正文；1px 发丝线；强调色只给运行态/主行动。

---

## 4. 信息架构

```
办公 / AI 工作台（唯一工作台）
├─ 左：工作列表（会话、置顶、运行中、最近产物）
├─ 中：办公视图
│   ├─ 对话（主视图：消息 + 工具卡 + 待确认 dock）
│   ├─ 轨迹（工具调用/产物/等待点）
│   ├─ 上下文（水位、载入文件、引用）
│   └─ 记忆（项目事实只读总览）
│   └─ 工具面板（临时覆盖在对话上方，不是页签）
│       ├─ CAD 看图：图套 rail + 画布 + 工具栏 + 状态栏
│       └─ 进度计划：视图切换 + 甘特/网络图 + 体检
├─ 右：检查器（对话上下文 / 当前工具）
└─ 底：输入岛（附件、工具、权限、模型、上下文水位）
```

关键交互：

- **调用**：输入框「工具」菜单选 CAD/进度；或 AI 工具卡上的「展开查看」；或命令面板；或产物行。
- **打开**：对话上方浮出工具面板，标题「工具 · CAD 看图」，副文「临时面板 · 关闭后解析结果留在产物」。
- **关闭**：Esc / 遮罩 / 关闭钮 → 回到原对话视图；不删除产物，不重置会话。
- **追问**：面板底部的「问这张图 / 问计划」把问题发回对话，回答在对话里继续。
- **归档**：工具结果自动进「产物」；重开工具不重跑解析（缓存/产物指针）。

---

## 5. 视觉规格

### 5.1 色彩（暗色默认）

| 角色 | 值 | 用法 |
|---|---|---|
| app bg | `#08090A` | 最底 |
| sidebar | `#050607` | 比画布更暗，后退 |
| canvas | `#0C0E10` | 对话 |
| surface | `#121517` | 卡片/输入岛/工具面板 |
| surface-2 | `#171B1E` | 悬浮/选中 |
| hairline | `rgba(255,255,255,.07)` | 1px 分隔 |
| text | `#E9E6DF` | 暖 off-white 正文 |
| text-2 | `#A7A39A` | 次要 |
| text-3 | `#6F6C66` | 弱 |
| accent | `#6EE7D2` | 运行/主行动 |
| gold | `#C9A86A` | 数据/编号点睛 |
| danger | `#F06A6A` | 关键线路/错误 |
| info | `#6FA8FF` | 链接/信息 |

亮色：`#F4F2ED / #FBFAF7 / #EDEAE3 / #1E1C18 / #0F766E / #8A6A2F`。

**对比度纪律**：正文 7–10:1，不追 14:1；次要 ≥4.5:1；不用纯白做长正文。

### 5.2 排印 / 间距 / 动效

- 正文 14px / 1.65；对话 14.5px / 1.7；工具面板 11–12.5px；数字 `tabular-nums` + 等宽栈。
- 4px 基准；面板内 12/16，区块 24；圆角：输入岛 16、卡片 12、按钮 8、工具面板 12。
- 动效 150–200ms，只做 opacity/transform；reduced-motion 全关。

### 5.3 工具面板规格

- 位置：对话区 `inset: 10px`，圆角 12，`shadow 0 24px 80px`；底下 52% 黑遮罩 + 1.5px blur。
- 头：40px；左「工具」徽标 + 工具名 + 当前文件；右「临时面板 · 关闭后结果留在产物」+ 关闭。
- 体：CAD 为图套 82px + 画布 + 状态栏；进度为工具栏 + 甘特 + 底部体检条。
- 检查器：右栏 316px，切到图层/图框/文字/统计或任务/逻辑/推演。
- 焦点：打开时焦点进面板，Tab 在面板内循环；Esc 关闭并归还焦点到触发元素。

---

## 6. 页面状态（原型覆盖）

1. **对话**：用户消息、助手结论、CAD 工具卡、设备统计、引用 chips、待确认 dock、输入岛工具菜单。
2. **CAD 工具面板**：图套 rail、平面图画布、工具栏、状态栏、右侧图层/质量检查器、问这张图。
3. **进度工具面板**：甘特/关键线路/今日线、AI 体检 3 条、右侧任务/偏差/what-if。
4. **空态工作台**：继续 + 工具 + 最近产物；工具 tile 打开的是工具面板，不是页面。
5. **轨迹 / 上下文 / 记忆**：证明办公主视图仍是完整工作记录，而不是工具页。

原型：`design/ai-workbench-2026-10/index.html`（`?tool=cad|schedule&theme=light`）。

---

## 7. 落到现有前端的接线方案（不改后端）

### 7.1 把 CAD 从「右栏 Tab」摘掉

- `workspaceTabs.ts`：从 `WORKSPACE_TAB_IDS` / `WORKSPACE_TABS` 移除 `cad`；`paneEnabled` 的 CAD 行、设置卡开关、欢迎网格卡同步删除。
- `sidebarRegistry.ts`：移除 `cad` 渲染项；`GaeaCad*` 四绑定保留，改由工具面板消费。
- 入口改三处：输入框工具菜单、AI 工具卡「展开查看」、产物行「重新打开」。

### 7.2 `ToolSheet` 壳层（新）

- `gaea/components/ToolSheet.tsx`：`{ open, kind, title, file, onClose, children }`；
  - 渲染在对话区内部（`position:absolute; inset:10px`），带遮罩、`role="dialog" aria-modal="true"`；
  - Esc/遮罩/关闭钮关闭；焦点陷阱；关闭后焦点归还触发元素；
  - 不改变 `chatTab`，所以关闭即回到原视图；不卸载会话。
- `gaea/lib/toolSheet.ts`：`{ active: null | 'cad' | 'schedule' | 'doc' | 'sheet', payload }`，按会话隔离；关闭只清 active。

### 7.3 CAD：`CadPanel` → `CadTool` + `CadInspector`

- 现有 `CadPanel.tsx` 的 generation/竞态/`GaeaCadOpen/Raster/Texts/Count` 逻辑原样保留；
- 拆出：
  - `CadTool`：图套 rail + 画布 + 工具栏 + 状态栏 + 问图纸；
  - `CadInspector`：图层/图框/文字/统计/质量/告警（右栏，工具打开时替换对话检查器）；
- 关闭面板不销毁 `openRes`（缓存在 `cadToolStore`，按 path 键），重开不重跑解析；
- `GaeaCad*` 四绑定零改动；问图纸仍走 `RecognizeImage` 零新绑定。

### 7.4 进度：`SchedulePage` → `ScheduleTool`

- 从 `SchedulePage` 抽 `ScheduleTool`（工具栏 + 甘特/网络图 + 体检条）与 `ScheduleInspector`（任务/逻辑/推演）；
- `SchedulePage` 保留为薄壳/深链接（旧路由不 404），但办公菜单不再指向它；
- `schedule_*` 工具与 `GaeaSchedule*` 绑定零改动；`WhatIfPanel`/`DcmaView` 逻辑复用为检查器内容；
- 工具卡由 `schedule_*` 工具结果触发；`openTool('schedule')` 打开面板。

### 7.5 空态：能力矩阵 → 工作台

- `Welcome` 的 12 张模块卡改为：`继续`（最近两个工作对象）+ `工具`（CAD/进度/文档/表格，动作）+ `最近产物`（3 行）；
- 工具 tile 调 `openTool(...)`，不导航；
- 保留 `ml-space-*` testid；新文案三语同步。

### 7.6 令牌映射

- 原型令牌 → `appStore` `ThemeTokens`：`surface/surfaceContainer/surfaceContainerHigh/text/textSecondary/border/glow`；
- 暗色 `--canvas #0C0E10` 对应 `colorBgLayout`；`--sidebar` 新增 `shellSidebar` 派生（比 layout 暗一档）；
- 亮色暖纸面对应 `#F4F2ED / #FBFAF7`；`ensureLightContrast` 继续保障 accent 文本 AA；
- 不改 12 套主题表。

### 7.7 验收

- **入口**：CAD/进度不出现在左侧导航、办公页签、右栏一级 Tab、欢迎能力矩阵；只在工具菜单/工具卡/命令面板/产物行出现。
- **工具面板**：打开/关闭/Esc/遮罩/焦点归还；关闭后对话滚动位置不变；不重跑解析。
- **CAD**：三格式导入 → 图套切换 → 图层过滤 → 问图纸 → 发对话；右栏检查器正确切换。
- **进度**：体检 3 条可定位；what-if 不改原计划；关闭回对话。
- **视觉**：1440×900 暗/亮；正文 7–10:1；侧栏暗于画布；reduced-motion。
- **门禁**：`tsc`/`eslint`/`vitest`/`drift` 全绿；零新绑定；`WORKSPACE_TABS` 无 cad 的契约测试。

## 8. 原型

`design/ai-workbench-2026-10/index.html`（无依赖；`?tool=cad|schedule&theme=light`）。
截图见同目录 `screens-*.png`。

---

## 9. 首批落地（2026-10-25）

已按「第一刀」落生产前端（CAD 先走通，进度下一步）：

| 项 | 落点 |
|---|---|
| ToolSheet 壳层 | `gaea/components/ToolSheet.tsx` + `ToolSheet.test.tsx`：临时面板、Esc/遮罩/关闭、焦点归还、焦点陷阱 |
| CAD 工具化 | `CadPanel` 增加 `variant="sheet"`；`cad-panel.css` 三栏（图套/画布/文字统计）；空态改为「打开一张图纸」卡 |
| 右栏摘除 CAD | `workspaceTabs.ts` 移除 `cad`（6 项）；`sidebarRegistry.ts` 移除 CAD 渲染；欢迎网格/管理/命令面板不再出现 CAD 一级 Tab |
| 入口 | 办公顶栏「工具」下拉 + Composer 工具栏工具钮 + 命令面板「CAD 看图」；三处都打开同一 ToolSheet |
| 关闭语义 | `activeTool` 只清面板；对话/会话不变；`CadPanel` 的解析缓存/竞态逻辑原样保留 |
| 未做（下一步） | 进度 `ScheduleTool` 抽取；`SchedulePage` 仍保留深链接壳；工具面板与右栏检查器合并；自动从工具卡打开 |

**验证**：`tsc -b` 绿；ESLint 0；定向 vitest 50 例绿；全量 vitest 3973/3974（1 例 `FilePreviewModal` 超时，隔离复跑 15/15 绿=并发负载 flaky）；`pnpm run build` 绿（CadPanel 独立 chunk 14.8KB）。

---

## 10. 第二刀：进度工具化（2026-10-25）

§9「未做」三件中的两件已落（工具卡自动打开 + 进度抽取）；「工具面板与右栏检查器合并」留池。

| 项 | 落点 |
|---|---|
| ScheduleTool | `gaea/components/schedule/ScheduleTool.tsx`：紧凑面=工程名 + 视图四档（横道/单代号/双代号/质量体检，编制档 usage/monte/evm 留完整工作台）+ 撤销/重做/添加任务 + 状态栏（工作数/总工期/关键数/基线漂移/倒排超期/保存态）+「问计划」回对话；TaskInspector+WhatIfPanel 同款复用；空态出「载入示例工程」。懒加载独立 chunk |
| 入口扩双工具 | `activeTool: "cad" \| "schedule"`；顶栏「工具」下拉两档；Composer 工具钮改下拉（两档同源）；命令面板 `cmd-schedule` 从跳板块改开工具面板；ScheduleFileCard「在进度计划中打开」改发 OPEN_TOOL（切指针语义原样保留） |
| 工具卡自动入口 | `ToolCard` schedule_apply done 回执动作行：「展开查看」开进度工具面板（FRONTEND_EVENTS.OPEN_TOOL 第 6 个前端事件）；回滚行（ScheduleApplyRollback 改片段渲染）同排共存，行容器上移 ToolCard |
| 完整工作台 | ScheduleTool 头部「完整工作台」深链接（NAVIGATE 板块路由保留不 404）；SchedulePage 原样保留（导入导出/工程管理/资源/模板/挣值模拟仍在） |
| 数据单源 | 与 SchedulePage 共享 useScheduleStore（initScheduleSync 模块级幂等）；agent schedule_apply 写入经 usePreviewAutoFront 即时回读两条路都可见 |

**验证**：`tsc -b` 绿；ESLint 0；全量 vitest 3989/3989（含 ScheduleTool 7 新例、App.palette/events 两个源级守卫同批改口径——守卫随代码迁移纪律）；`pnpm run build` 绿（ScheduleTool 独立 chunk 7.21KB、CadPanel 14.81KB，懒加载）。

---

## 11. 第三刀：CAD 重开不重跑 + 第四刀：空态工作台工具区（2026-10-25）

| 项 | 落点 |
|---|---|
| cadToolCache（§7.3 尾） | `gaea/components/cad/cadToolCache.ts`：解析产物按 path 缓存（模块级单例，cadRecent 同款）。ToolSheet 关闭卸载 CadPanel 后 openRes/raster/texts/count 留存，重开（最近图纸/重复导入）零 CadOpen/CadRaster/CadTexts/CadCount 调用。图纸只读红线→无失效需求；内存节制：raster 仅缓存默认宽度渲图（ASK_WIDTH 加宽图不缓存）、每文件 3 张 LRU、8 文件 LRU。CadPanel 六处接线（load 命中短路/open/raster/texts/count 落/handoff 内联落），代际竞态守卫原样生效 |
| 空态工具区（§6.4） | `Welcome` 新增工具节（能力卡后）：CAD 看图/进度计划两 tile，点击开 ToolSheet 不导航；`onOpenTool` 缺省整节不渲染（宿主可选接线）；Transcript/App 穿线；三语 5 键（welcome.tools*） |
| 挂拍板池 | §7.5 首页 12 模块卡改造——首页形态是 v4.476 刚拍板的极简化（模块卡极简化保留全卡），且 doc/sheet 工具 tile 无 ToolSheet kind 支撑；改首页与既有拍板有张力，提请拍板后再动（**2026-10-26 用户拍板执行，落地见 §13**） |
| 留池 | ~~工具面板与右栏检查器合并~~ → **2026-10-28 已落（CAD 侧，见 §14）**；§7.6 令牌映射暖纸面（需单独拍板） |

## §14 检查器随工具（2026-10-28，CAD 侧）

用户「继续」拍板续池。落法与 §7.3 假设不同：**不做 CadTool/CadInspector 状态上移**
（15+ useState+6 ref 代际竞态守卫是大手术），改走 **portal**——CadPanel 状态与回调
零上移，检查器节点（文本清单+符号统计）经 `createPortal` 渲染进右栏宿主。

- `CadPanel` 新 prop `inspectorHost?: HTMLElement | null`：宿主在挂 → 检查器 portal
  进右栏，sheet 画布区 3 栏变 2 栏（`data-inspector="external"`）；宿主不在（右栏
  收起/pane 变体）回落内联=与合并前 DOM 完全一致。右栏收放实时跟随（宿主 ref
  callback → App state → prop）。
- `App.tsx`：右栏在 `activeTool==='cad'` 期间整体切「图纸检查器」宿主（设计原则 4
  「右栏随当前对象变化」的第一次真实发生）；开 CAD 自动亮右栏、关 CAD 还原打开前
  开合态（快照只在工具切换沿动作一次，打开期间用户手动收放不被抢）。
- **进度工具不参与**：TaskInspector 抽屉（右键行→抽屉+what-if）交互已自洽，硬搬
  右栏是负收益——§7.4 尾的「ScheduleInspector」判定不做，销号。
- 行为锁：`CadPanel.portal.test.tsx` 2 例（宿主在挂=检查器住右栏+sheet external；
  宿主撤走=回落内联 inline）；既有 CadPanel.test/cache/race 30 例零编辑复绿
  （无宿主=内联=合并前）。空态 CTA 补 `cad-import-empty` testid（sheet 空态无
  cad-import 钮；测试按 testid 不按文案——测试里 LocaleProvider 落过英文）。
- 已知边界：ToolSheet 焦点陷阱只圈 sheet 本体，右栏检查器的钮鼠标可达、键盘 Tab
  不可达（辅助面，接受；如需闭合再把宿主纳入 trap 语境）。

**验证**：`tsc -b` 绿；ESLint 0；全量 vitest 3997/3997（cad 20 例含缓存 6 新例；既有「卸载重挂一键重开」用例断言随新语义同批迁移=重开后 CadOpen 不再二次调用；Welcome 6 例含工具节 2 新例）。

---

## 12. 第五刀：shellSidebar 派生——侧栏后退（§7.6，2026-10-25）

| 项 | 落点 |
|---|---|
| 令牌派生 | `appStore.ts`：ThemeTokens 新增 `shellSidebar`；`getThemeTokens` 中央派生 hex（mixHex 线性混合）——暗态 colorBgLayout 掺黑 22%（后退一档，文本对比只增），亮态掺 5% 暖灰 `#8a857c`（压灰微暖=暖纸方向）。**不改 7×2 主题表** |
| 下发与消费 | App.tsx 下发 `--md-sys-color-shell-sidebar`；Sidebar aside 与 redesign.css rail 的背景链在 `--v3-rail-bg`（保留覆盖位）之后、glassBg 之前插入；styles.css `--sidebar-bg` 同步改链。此前侧栏=画布同调（glassBg→surface），无后退 |
| 亮色暖纸 | 本刀只落在派生层（侧栏暖灰）；全量亮色暖纸改版=原型令牌映射文档（§7.6），动主题表面需单独拍板 |
| 对比度契约 | appStore.contrast.test 新增 14 例：全预设×明暗 shellSidebar 为 hex、侧栏正文四令牌（text/secondary/tertiary/primary）≥4.5:1（会话列表文本直接压壳侧栏）；暗态加亮度严格低于画布的后退不变式 |

**验证**：`tsc -b` 绿；ESLint 0；定向 contrast 35/35（含 14 新例）+Sidebar/appStore/perf-guards 32/32；全量 vitest 4010/4011（1 例 `FilePreviewModal` Esc 关闭为已知并发负载 flaky，隔离复跑 15/15 绿——与 §9 首批同签名，非本刀面）。

---

## 13. 第六刀：首页 12 模块卡工作台化——工具行前置（§7.5 销项，2026-10-26）

2026-10-26 用户拍板执行 §7.5 挂拍板项，并明确**不采纳本文档 §7.5/§8 原型的三段方案**（继续+最近产物无跨板块数据源、doc/sheet tile 无 ToolSheet kind 支撑——老方案基于旧首页认知，已过时）。落地以刀1~5 之后的现状为基准做增量：

| 项 | 落点 |
|---|---|
| 对象修正 | §7.5 原文「Welcome 的 12 张模块卡」实为**首页 `DeskHome`（ModuleLauncher）能力矩阵节**的 12 张板块卡（chat/novel/imagegen/gaea/cost/memoryhub/modelcenter/characterlib/settings 特判/weixin/sin/runtime；schedule v4.168 起 inMenu:false 不在网格）。办公空态 Welcome 无 12 卡（刀4 已加工具节） |
| 工具行前置 | `DeskHome` 能力矩阵节前新增**工具节**（`home-tools`）：CAD 看图/进度计划两 tile——工具=动作不是房间；与板块节（`home-boards`）分节，两类心智不再混排。tile 复用 `.w-mod` 全卡同权形态（v4.476 极简语言零新 CSS），`HOME_TOOLS` 清单只收已落地 ToolSheet kind（cad/schedule），doc/sheet 无 kind 支撑不冒充工具 |
| 跨板打开链路 | 首页 tile 点击=**OPEN_TOOL 双通道**：`emitFrontendEvent`（办公 App keepAlive 常驻直达 `setActiveTool`）+ sessionStorage 标记 `OPEN_TOOL_FLAG`（首访未挂载、事件无人听时由办公 App 挂载效应补开，读后即清——`OPEN_STATS_FLAG` 同款模式）+ `onNavigate('gaea')`。比 OPEN_STATS 模式多一手：**监听侧顺带清标记**，杜绝已挂载场景写入的标记残留到下次冷启误开 |
| 板块节口径 | `home.capTitle` 三语「能力矩阵」→「板块」（Boards/板塊）——本节就是板块入口网格，「能力矩阵」是 §2 诊断用语；sub「全部工作台，一屏尽收」保留。12 卡全卡同权**零变化**（v4.476 拍板形态不动） |
| 事件面 | `events.ts` 新增 `OPEN_TOOL_FLAG = 'gaea.tool-open'`（标记值为工具名，区别于 OPEN_STATS_FLAG 的 '1'） |
| 消费侧 | `gaea/App.tsx` OPEN_TOOL 监听效应扩为双通道：挂载效应读标记补开（privacy try/catch 静默）+ 监听内清标记 |
| TasksFirstHome | 不动——任务优先首页心智是「先处理收件箱」，能力 chips 是紧凑文本视图，工具直达不属其核心路径 |

**验证**：`tsc -b` 绿；ESLint 0；定向 ModuleLauncher 13 例（含工具节新例：双通道事件+标记+导航断言、tile 三件套形态、板块断言 scope 化迁移）+ App.palette 源级守卫 5 例（含冷启补开双通道新例）= 18/18 绿；全量 vitest 见提交记录。
