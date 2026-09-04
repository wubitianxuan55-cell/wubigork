# gaea × DSH 生态插件长期蒸馏规划（2026-09 起）

> 状态：**滚动蒸馏**（每版 1-2 刀）｜ gaea 基线：v4.78.0（2026-09-04）
> 源 A：[omdsh-dev/DSH-better-sidebar](https://github.com/omdsh-dev/DSH-better-sidebar)
> 源 A 快照：v0.18.0 · commit `e1b9b53`（2026-09-04）
> 源 B：[bowenliang123/dsh-context](https://github.com/bowenliang123/dsh-context)
> 源 B 快照：v0.41.3 · commit `fbf32ec`（2026-09-04）
> 性质：持续蒸馏规划——不追求一次 Go 化「替换」，而是按
> 「能力差距 → 分批蒸馏 → 版本验收 → 回源更新」的滚动循环推进。

## 1. 为什么需要长期规划

gaea 办公板块自 v4.23 起已把 DSH 生态两个代表性插件的能力**分批挑项蒸馏**：
pane 工作台/文件 tab（v4.25）、任务+分工同屏（v4.53）、子代理实时拓扑与后台
任务收口（v4.75-v4.77）、上下文页对齐 dsh-context（v4.67-v4.71）等。源仓库
本身仍在高速演进（better-sidebar v0.14.1 → v0.18.0 新增统一文件变动、侧边
对话、双工作台、固定终端；dsh-context 持续推进浏览器深读/真实用量/网络卡），
因此需要：

1. **持续追踪源侧**：固定两份源 commit，滚动对比能力面，避免「拍板一次后源
   又长出好东西」；
2. **保留已蒸馏资产**：docx/xlsx 编辑、证据链、上下文看板等 gaea 原生能力
   是差异优势，只借鉴源的交互形状与键设计，不整包替换；
3. **分批可验收**：每刀以版本发布收口，功能零删减红线只在用户明确要求时松动。

## 2. 蒸馏原则（红线）

- **学形状、不抄实现**：源为 MIT，但保持既有「学习借鉴」注释纪律；架构按
  gaea 的 Go 权威 + React 渲染壳演进，不引入 Node host；
- **Office 编辑全量保留**：docx/xlsx/pptx 相关编辑、Plan→Apply、证据链换壳
  不换芯；
- **能力以差距为准**：已有且同构的（任务/子代理/实时输出）只做收口，不重做；
- **安全默认更严**：所有新路由/绑定按会话 cwd + workspaceFence 校验；源侧
  `fs.write` 无围栏的旧行为不照搬；
- **交互拍板项**：真实终端、人工沙箱浏览、双工作台/自由窗口、侧边对话等
  大项必须逐项拍板后再动刀（见 §6 决策门）；
- **交付纪律**：每刀 = 前端 vitest + tsc/eslint 0 + Go 全量 + drift PASS +
  releases/vN.md + README 版本表；有交互变化附 `?mock=1` 实拍。

### 2.5 源文档读书记录（2026-09-04 补读）与关键结论

> 除 README 外，已通读源仓库 AGENTS.md、docs/external-plugin-guide.md，并补读
> 6 份核心设计文档（任务/后台、sidechat、自由窗口、声明式设置、懒加载、移动端
> 布局、模型主动打开）。记录结论，防止规划建立在二手摘要上。

- **AGENTS.md（仓库开发规则）**：host 零写入官方 DSH、只走插件自有路由/公开
  API；CI 有「npm 打包→真实挂载→无头渲染」冒烟。对 gaea 的意义：源的很多
  约束（如 jobs.output 只读回放）是「不修改宿主」造成的，gaea 是自研宿主，
  对应能力应做成**原生实现**（Go 权威 + 事件流），不必保留其绕行痕迹。
- **external-plugin-guide.md（接入 API 全参考）**：权威清单是 **7 内置 tab
  （editor/git/subagent/sidechat/terminal/browser/diff）+ 6 viewer
  （image/pdf/markdown/html/code/binary-download）**；TabDescriptor 有
  dedupe/available/createTab/badge/onOpen/声明式设置；`workspaceFence`
  默认开（fs.tree/read/write/媒体/HTML/upload 全部包含检查）；自由窗口、
  皮肤契约、portal 事件劫持等均为**实现契约**。gaea 没有插件系统，蒸馏落点 =
  「注册表驱动的 manifest + 代码级 RENDERERS」，不引入第三方生态。
- **2026-08-12-subagent-background-tasks-design.md**：任务输出 = 会话事件
  回放（不碰消费游标）；**终止是两击确认**（首击进入 3s 确认态，防误杀）；
  单共享 Dock + 隐藏停轮询。gaea TaskCenter 已具备实时输出 Dock；v4.77 把
  终止按钮直接命名为「强制终止」——应补「两击确认」再算完整蒸馏。
- **2026-08-20-sidechat-tab-design.md**：侧边对话的难点在种子算法：复制父会话
  全部事件、进行中回合用 `interrupted` 诚实闭合、悬挂 tool/call 回退结构化
  转储；前缀缓存复用；`origin:'subagent'` 零目录噪音；保存为新会话走 fork。
  gaea 的 SubagentThread「独立会话 tab + 保存为新会话」已具备 60% 交互底座，
  缺的是 Go 侧「种子继承 + 独立运行」语义。
- **2026-08-23-free-window-design.md / 2026-08-12-mobile-layout-design.md**：
  自由窗口是「拖到主会话区变悬浮窗、拖回停靠」的纯 client 状态机；窄屏
  （<768px）合并为全宽抽屉、桌面布局不挤压（`--dsh-sidebar-*` 归零）。
  gaea 桌面优先，办公布局 <1240 现为隐藏 workspace-pane；若做抽屉/自由窗口
  需先定义 gaea 自己的断点与「不挤压」语义。
- **2026-08-11-declarative-sidebar-settings-design.md**：设置卡由注册表驱动，
  逐 tab/viewer 开关 + 齿轮二级设置；保留式语义（禁用只影响新开）。gaea
  已把部分偏好转为设置卡（自动切任务视图等），可扩展为「面板清单卡」。
- **2026-08-12-lazy-chunks-design.md**：xterm/CodeMirror/mermaid 等重依赖
  拆独立 chunk 按需加载，核心 ~325KB。gaea 前端已是 Vite 构建，未来引入
  终端/编辑器时沿用同一原则（react 懒加载 + 重库独立 chunk）。
- **2026-08-23-agent-open-tools-design.md**：sidebar_open 的 delivered 语义
  （无订阅排队、连接后重放、消费即出队）。gaea 已有 sidebar_open 工具与
  reveal 接线，可在蒸馏收口时对照该语义。

### 2.6 第二源：dsh-context 读书记录（2026-09-04 补读）与差距结论

> 已通读 dsh-context v0.41.3 README、AGENTS.md、docs/compatibility.md，
> 并对齐 gaea 上下文页现状（v4.67-v4.71 已蒸馏 + 记忆/任务后续改版）。

- **源侧强项（gaea 上下文页已有对应但缺深水区）**：
  - Context Browser 的「对比上一轮变动」：每类 signed delta 徽标（+N 项 /
    ±Nk tokens）、压缩后步骤由归档重建并显式标注「近似」——gaea 上下文
    欠账清单里正是这一条（需 per-request surface 快照）；
  - 工具 schema 行的来源 chip（tool-* / dsh-* / mcp:<server> / 插件名）与
    size/name 排序、分类内可搜索字段——gaea 浏览器节点只有文本+token；
  - 工具结果展开整段调用（OK/error 状态、行数、Raw/Markdown 切换）与图片
    payload 缩略卡（含 dsh 图片 token 估算）——gaea 无；
  - File Activity 的 ±added/−removed 行级增量、搜索命中行、展开完整操作
    日志并跳转到对应工具结果——gaea 文件活动只有读写次数/时间；
  - Trend 的 step brief（User/In/Response 三行）点击跳转 Context Browser 的
    对应消息；hover 联动浏览器预览；provider 实际用量展示；
  - Agent Network 可点击跳到该会话自己的 Context tab；
  - `/context` 命令打开**居中弹层**（不离开对话），含当前构成 + 浏览器；
  - 设置中心持久化默认粒度/模式/文件活动排序（gaea 现为卡内瞬时 toggle）；
  - 成本估算 hover 展示 per-1M 费率。
- **源侧工程纪律值得吸收**：解析韧性（宿主 fold 全量、坏事件整条丢弃、
  客户端边界清洗）、100% 覆盖、兼容 seam 矩阵。gaea 有自己的 ContextView
  契约测试与 mock 走查，可把「抗坏数据」补充进上下文测试（现有部分有）。
- **gaea 优势不回退**：卡片化主区布局（v4.71）、记忆/概览的收纳决策、
  Go 侧 contextview 折叠权威 + 事件流——不因追 dsh-context 而回到纯前端
  自算形态。

## 3. 源能力面与 gaea 差距（2026-09-04 快照）

### 3.1 已基本对齐（只收口）

| 源能力 | gaea 现状 | 收口项 |
|---|---|---|
| 文件工作台（树/搜索/读/写/多文件 tab） | ExplorerView + FileTree + pane 文件 tab + FilePreview 族 | 上传/拖放、右键菜单、@文件 悬浮引用、树 watch 自动刷新（gaea 已有 filewatch，接事件流） |
| 后台任务页（子代理拓扑+任务） | AgentTree/TaskCenter 实时拓扑、退出码、实时输出、强制终止、自动激活 | 进程级强杀语义（Go 侧 killProcessTree）、会话固定任务概念 |
| 本轮文件视角（写文件追踪） | ChangesPanel + DeliverableRegistry + VersionTimeline | 模型读/写/编辑三态独立折叠层、按类型筛选 |
| 会话隔离/布局持久化 | paneTabs 会话记录 + sanitize + 宽度键 | 陈旧状态净化边界审计；必要时权威迁 Go（拍板项） |
| 模型主动打开 | sidebar_open 工具 + revealInExplorer | — |

### 3.2 半具（有底座、缺面）

| 源能力 | 差距 |
|---|---|
| Markdown/HTML/PDF/图片 预览族 | md 已有 ChatMarkdown/FilePreview；缺 Mermaid strict 安全渲染、README 级内嵌 HTML（DOMPurify 白名单）、浮动 TOC、HTML 沙箱预览路由 |
| 代码编辑器 | 文本/md 内联编辑已有；缺语法高亮编辑器与懒加载（CodeMirror 类） |
| Git 视角 | gaea 有产物/变更/回滚，但**无 Git 面板**：status/diff/stage/commit/revert/history/worktree；统一 diff 渲染（改蓝配对/行内高亮/语法着色/上下文折叠）是核心增量 |
| 内嵌浏览器 | 现有受控 Edge 观察窗（agent 驱动）保留；缺「用户人工沙箱 iframe 多开」+ 外链协议分流 + 沙箱状态条（需拍板是否做） |

### 3.3 全缺（大项，逐个拍板）

| 源能力 | 量级 | gaea 映射 |
|---|---|---|
| 真实终端（xterm + ConPTY + agent terminal 工具） | 中-高 | 建议第一轮只做「任务实时输出 + 强杀」，交互 shell 独立拍板 |
| 侧边对话（继承主会话上下文独立运行、可提升为顶层会话） | 高 | 与现有「子代理独立会话 tab」「记忆/上下文主区化」天然衔接，可分期做 |
| 双工作台/分栏/自由窗口 | 高 | 与 gaea「主区=对话+预览」哲学冲突，列为远期另案 |
| 固定终端/跨会话 pin | 依赖终端 | 终端拍板后再评估 |
| 声明式设置卡（逐 tab/viewer 开关+齿轮二级） | 中 | gaea 设置中心已有部分；可提炼为「面板偏好卡」 |

### 3.4 dsh-context 差距（上下文页，2026-09-04）

| 源能力 | gaea 现状 | 蒸馏优先级 |
|---|---|---|
| Context Browser「对比上一步」/ 压缩归档重建（标注近似） | 欠账（需 per-request surface 快照） | P1 |
| 工具来源 chip / 排序 / 分类内搜索 | 无（节点仅文本+token） | P2 |
| 工具结果展开整段调用 + Raw/Markdown + 图片缩略卡 | 无 | P2 |
| File Activity ±行增量 / 搜索命中行 / 操作日志展开跳转 | 无（仅读写次数/时间） | P2 |
| Trend step brief 点击跳转 / hover 联动浏览器 / 实际用量 | 跳转已有（v4.82 brief 锚点）；hover 联动有意不做；实际用量部分 | ✅（hover 边界成文） |
| Agent Network 点击进入对应会话 Context | 部分（AgentRadial 可开子代理对话 tab） | P3 |
| `/context` 居中弹层 | 斜杠命令切主区 tab（等价但不同交互） | P3（拍板） |
| 设置中心持久化默认（粒度/模式/排序） | 已落地（v4.82 gaea.context.prefs + 设置卡） | ✅ |
| 成本费率 hover | 已落地（v4.89 fold 透出 Rate 快照 + SummaryBar hover 三档明细） | ✅ |

## 4. 蒸馏节奏（建议循环）

每个版本（vN）为一次「源追踪 + 一至两刀」：

1. **回源**：拉源仓库最新 commit，仅更新本文档 §3 与差距清单（不每版做，可
   每月/每五版）；
2. **选刀**：按 §5 队列 + 用户拍板项，选 1-2 个差距；
3. **端口分析**：产出能力映射（源文件/行为 → gaea Go/React 落点），不足百行
   的改动在 release note 内说明，超过则先写 docs/plan；
4. **实现 + 验证**：按 §2 交付纪律；
5. **回填**：更新本规划的状态列与已蒸馏清单。

## 5. 分期路线（推荐默认；每刀可单独拍板）

### 阶段一（近 1-3 版，v4.78+）：收口与预览补全
- ~~1a 任务页收口：Go 侧进程级强杀（复用 internal/gaea/proc）+「强制终止」
  两击确认（对齐源文档防误杀）+ 任务事件自动激活的窄屏策略审计~~
  **已完成（v4.78.0）**：Progress.OnForceKill + Manager.Kill（queued 原子
  取消/running 先杀进程树再传播取消/无钩子诚实降级）；前端两击确认（首击
  3s 确认态）；窄屏审计通过（CSS 1239px ↔ JS <1240 一致）零改码；D1 采纳
  推荐默认；绑定面 +GaeaTaskKill（drift 570）；
- 1b 文件/浏览器小面：文件行右键菜单（新 tab/复制路径/在侧边打开）、@文件
  悬浮引用、上传/拖放（cwd 围栏复用写路径守门）；
- ~~1c HTML 沙箱预览（iframe 属性 + CSP）+ 外部链接协议分流~~ **已完成
  （v4.84.0，阶段一收口）**：GaeaPreview kind=html（原文截断读）；SandboxedHtml
  独立 iframe——`sandbox="allow-scripts"` 无同源 + Chromium csp 属性
  `default-src 'none'` 双保险 + 顶条如实标注；FilePreview/FilePreviewModal
  双消费点。外链分流=`classifyExternalLink` 纯函数（http/https 放行
  loopback 拒、mailto/tel 交系统、其余 blocked），接线 Markdown 链接与
  价格源两处点击点。

### 阶段二（中 2-4 版）：统一文件变动与 Git
- ~~2a 本轮文件折叠补全：模型读/写/编辑三态追踪 + 按文件分组 + 类型筛选~~
  **已完成（v4.85.0）**：ChangesPanel 重构为 写入/编辑/读取 三层独立折叠
  （同文件跨层独立出现；读取层轻量行默认收起；写/编辑层 diff+证据链回滚
  保真）+ 类型筛选 chips（文档/表格/图片/代码/其他，横贯三层）；读取白名单
  对齐后端 fileActionByTool；工具集参数化零破坏；
- ~~2b Git 面板最小集：status/diff/stage/commit/revert/history~~ **已完成
  （v4.86.0；D3 采纳推荐默认，v4.78 D1 先例）**：GaeaGit*7 绑定（git CLI
  exec 列表、仓库锚定 gaeaCwd、非仓库/git 缺失诚实错误）；Git 一级 Tab
  （声明式清单派生）——三分组状态、unified diff 复用 ChangesDiff（2c 的
  改蓝配对/字符高亮/语法着色仍留）、暂存/取消暂存/丢弃两击确认、提交仅
  暂存区不代 add、历史懒加载；无 push/pull/fetch，worktree/子仓库另案；
- ~~2c 统一 diff 渲染升级~~ **已完成（v4.87.0）**：diffRender 纯函数展示
  模型——改蓝配对（相邻删块+增块按行两两配对，蓝底替代红/绿，data-pair
  标记）+ 行内字符高亮（配对行字符级 LCS，变化片段独立着色，240 字上限）
  + 上下文折叠（ctx 中段收起可展开，fold 项携带被收起行）；ChangesDiff
  统一查看器承接 变更面板 LCS + Git 面板 unified diff 两数据源；**语法
  着色留阶段三 CodeMirror**（依赖高亮器）；docx/xlsx 对比迁移随
  VersionTimeline 另刀。

### 阶段二.5：上下文深水区（dsh-context 源，2-3 版）
- ~~2.5a 浏览器「对比上一步」~~ **已完成（v4.79.0）**：Go fold 请求组装点
  surface 快照（与 Category 同拍；system/tools 最新 header 整体估算防历史
  头重计）→ RequestDelta 逐类 signed delta + 跨压缩 Approx + 首请求 First；
  趋势卡请求详情 delta 条。**边界**：历史步仅聚合级 delta，dsh 式逐步节点
  回放有意不做（wire 载荷 + Go 权威折中，蓝图成文）；
- ~~2.5b 工具结果与图片深读~~ **已完成**：前半（v4.80.0：GaeaContextNodeDetail
  懒加载 + 来源 chip/error 点 + 时间序/大小序）+ 后半（v4.83.0：图片缩略卡
  + 官方 patch 口径 token 估算 ⌈w/28⌉×⌈h/28⌉ 先档位缩放再封顶——口径经
  2026-09-05 调研核实 platform.claude.com/docs/en/build-with-claude/vision，
  官方例 1000×1000→1296 单测锚定；弃用社区旧式 /750；缩略卡成对显示
  「原始尺寸→缩放尺寸」与「≈N tok · 标准档」；缺失/不可解码诚实降级；
  fold 修 stats.Images 死字段）；
- ~~2.5c File Activity 行级 ±增量、搜索命中行、操作日志展开跳转~~
  **已完成（v4.81.0）**：FileActivity.Added/Removed/Hits（写类工具参数
  确定性提取 + grep 结果行数下界）；前端 ±徽标 + 「N 次操作」展开 +
  操作行详情懒加载跳转（复用 v4.80 面板）。**1b 勘误**：右键菜单/@悬浮
  引用早已落地（v4.25-4.31），1b 销账（拖放入工作区树价值稀薄不做的
  理由见 release note）；
- ~~2.5d Trend step brief 跳转与 hover 联动、设置中心默认（粒度/模式/排序）~~
  **已完成（v4.82.0）**：Go fold 透出 briefUserSeq/briefRespSeq（user/
  assistant=消息节点，工具交换结果到达锚结果节点、未到退化 assistant 消息
  节点，三处同拍）；前端 brief 行跳转按钮→浏览器组展开（含分页全量）+滚动+
  高亮 3s（归档同款、锚点无节点诚实不跳）；趋势粒度/模式、浏览器排序、文件
  活动排序初值读 gaea.context.prefs 变更写回；设置中心办公分组「上下文页
  偏好」卡（选项复用 contextview.* 键）。**边界**：hover 联动预览不做
  （hover 驱动展开会反复抖动，点击是明确意图）；2.5e 仍待拍板；
- ~~2.5e `/context` 弹层~~ **已完成（v4.88.0，与调研回填「codex 式常驻
  剩余上下文徽标」合并为一刀）**：ContextPill 常驻徽标（迷你进度条三档
  配色，点击开弹层）+ ContextModal 居中弹层（复用 ContextView，打开挂载
  关闭卸载）；/context 命令改道弹层，主区 tab 保留。**剩余半项**：Agent
  Network 会话跳转需 GaeaContextView 按会话参数化，随中期候选另刀；

### 阶段三（拍板后）：预览与浏览增强
- ~~3a CodeMirror 语法高亮编辑器~~ **已完成（v4.91.0）**：codemirror@6 +
  六语言包（md/js·ts·jsx·tsx/py/json/css/html）懒加载 chunk；FilePreview
  编辑态双层回落（Suspense fallback + EditorBoundary→textarea）；**语法
  着色能力随依赖就位**（diff token 着色另小刀）；
- ~~3b Mermaid strict 渲染 + DOMPurify HTML 消毒~~ **已完成（v4.92.0）**：
  MemoMarkdown 流式尾部接 sanitizeHtml（DOMPurify 兜底转义回归）；
  **Mermaid SVG 外层再消毒经实验证伪后有意不做**（mermaid strict 已内置
  DOMPurify；外层 svg profile 必剥离 foreignObject 文字，ADD_TAGS 无法
  补救——决策成文 lib/sanitize.ts）；README 级内嵌 HTML 白名单渲染另刀
  拍板；
- 3c 人工沙箱浏览器多开（若拍板「观察窗 + 人工浏览双形态」）。

### 阶段四（拍板后）：真实终端（若做）
- 4a Windows ConPTY + xterm（懒加载）+ 会话回放；
- 4b shell/shellArgs 设置 + UI 终端上限；agent terminal 工具（默认关）；
- 4c 固定终端（拍板后）。

### 阶段五（拍板后）：侧边对话（若做）
- 5a 线程模型：继承主会话完整上下文、interrupted 冻结、独立运行不污染主会话
  （复用 SubagentThread 的独立会话 tab 思路）；
- 5b 追问/冷恢复/保存为新会话（gaea 已有「保存为新会话」先例）。

### 远期（另案，不建议进入当前版本队列）
- 双工作台（右侧栏+底部面板）、拖 Tab 拆分/合并、自由窗口；
- Go 侧权威布局状态迁移（若 localStorage 多端/备份出现分歧再触发）。

## 6. 决策门（逐项拍板后动刀）

| # | 决策 | 推荐默认 |
|---|---|---|
| D1 | 真实终端是否进入 2026 队列 | 先只做任务输出+强杀；交互终端独立排期 |
| D2 | 人工沙箱浏览器 | 观察窗保留，人工浏览另案 |
| D3 | Git 首轮范围 | 单仓库 status/commit/revert，无 push |
| D4 | 布局状态是否迁 Go | 暂缓；出现分歧再迁 |
| D5 | 双工作台/自由窗口 | 远期另案 |
| D6 | 每刀节奏 | 按 §4 每版 1-2 刀 |

## 7. 已蒸馏资产台账（防止重复蒸馏）

> 每完成一刀在此追加一行，并在源版本演进时回源核对。

| gaea 版本 | 蒸馏内容 | 源对应能力 | 备注 |
|---|---|---|---|
| v4.23-v4.31 | 右栏工作台框架、任务/分工/文件 tab、产物行 pane 化 | 文件工作台/后台任务 | 见 releases |
| v4.53-v4.63 | 任务+分工同屏、子代理会话 tab、产物自动置前 | 任务页/侧边对话前身 | — |
| v4.66-v4.67 | 双 store 轮询收敛、上下文页对齐 dsh-context | subagents.live 思想 | — |
| v4.72-v4.76 | 概览删除、记忆/上下文主区化、任务面板卡片化与整树折叠 | 任务页形态 | 参考图蒸馏 |
| v4.77 | 任务自动激活补全、强制终止、窄屏策略 | 任务页自动激活 | — |
| v4.78 | 任务进程树强杀（OnForceKill/Kill）+ 强制终止两击确认 | 任务页终止两击确认（2026-08-12 设计文档） | D1 采纳推荐默认；阶段一 1a 收口 |
| v4.79 | 上下文对比上一步（RequestDelta 聚合级 signed delta + 近似标注） | Context Browser 对比上一轮变动 | 节点级逐步回放有意不做（蓝图成文）；阶段二.5 2.5a 收口 |
| v4.80 | 工具结果深读（懒加载完整调用 + 原文/渲染 + 来源 chip + 排序） | Context Browser 工具行展开/来源/排序 | 懒加载单点回读代 dsh 全量回放；2.5b 前半收口 |
| v4.81 | 文件活动 ±行增量/命中数/操作日志展开跳转 | File Activity ±added/−removed + 操作日志 | 2.5c 收口；1b 勘误销账（前两项 v4.25-4.31 已落地） |
| v4.82 | Trend brief 跳转锚点 + 粒度/模式/排序偏好持久化 + 设置中心默认卡 | step brief 跳转 Context Browser + 设置持久化默认 | 2.5d 收口；hover 联动边界成文（有意不做） |
| v4.83 | 图片缩略卡 + 官方 patch 口径图片 token 估算 + 图片引用提取 | 图片 payload 缩略卡（含 token 估算） | 2.5b 后半收口，阶段二.5 全部完成；口径来自 2026-09-05 调研 |
| v4.84 | HTML 沙箱预览（sandbox+CSP iframe）+ classifyExternalLink 外链分流 | html viewer + 外链协议分流 | 1c 收口，阶段一全部完成 |
| v4.85 | 变更面板三态折叠（写入/编辑/读取）+ 类型筛选 chips | 统一文件变动（读写状态分层） | 2a 收口；读取白名单对齐 fileActionByTool |
| v4.86 | Git 面板最小集（status/diff/stage/unstage/discard/commit/log + 一级 Tab） | git tab（地址栏/暂存/提交/历史） | 2b 收口；D3 采纳推荐默认；2c 渲染升级另刀 |
| v4.87 | diff 渲染升级（改蓝配对/字符高亮/上下文折叠） | 统一 diff 查看器渲染 | 2c 收口（语法着色随 3a）；阶段二全部完成 |
| v4.89 | 成本费率 hover（fold Rate 快照 + 三档明细） | 成本 hover 展示 per-1M 费率 | 调研回填中期候选第一项（langfuse/ccusage 口径） |
| v4.91 | CodeMirror 语法高亮编辑器（懒加载 + 语言映射 + 双层回落） | 代码编辑器（语法高亮+懒加载） | 3a 收口；2c 遗留语法着色能力就位 |
| v4.92 | MemoMarkdown 消毒层 + Mermaid strict 决策成文 | （安全线对齐，非蒸馏能力） | 3b 收口；外层 SVG 再消毒实验证伪成文 |
| v4.88 | /context 居中弹层 + 常驻剩余上下文% 徽标 | /context 弹层 + 常驻 context 计数 | 2.5e 收口（Agent Network 跳转半项留后续） |

## 8. 成功标准（长期）

1. gaea 办公板块对源「任务/文件/产物/浏览器/对话」五大交互面各有明确落点，
   无长期空转的差距条目超过两个版本；
2. 每刀都有可回看的 release note + mock 实拍，不出现「做过但无法复述」；
3. gaea 原生优势（Office 编辑、证据链、上下文/记忆主区化）不被源同化掉；
4. 源侧每月更新一次差距清单，本文档状态列永远反映最新现状。
