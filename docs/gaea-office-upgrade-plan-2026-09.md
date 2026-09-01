# Gaea 办公板块升级规划：右面板工作台化 / 子代理可视化 / 文件交互 / 成果交互

> 2026-08-31 市场调研 + 代码摸底。v2 修订（用户拍板）：**轨迹/上下文不收编入右面板**；右面板重造为
> DSH-better-sidebar / Codex 式**运行工作台**（子代理、浏览器、文件编辑器等），其现有「状态显示」类内容
> 迁到主区轨迹/上下文旁边。调研阶段未改任何代码。

## 一、市场调研结论（对标对象与可借鉴模式）

**头号对标：dsh-better-sidebar**（本地实装 v0.18.0-alpha.0，`~/.dsh/profiles/web/node_modules/`，README 为一手来源）。核心设计可整组借鉴：

- **形态**：VSCode 式右侧栏 + 底部面板双工作台；Tab 拆分/合并分栏；任意 tab 拖到主会话区成**自由窗口**（悬浮/缩放/置顶，拖回停靠）；宽度拖拽调节、按会话持久化、陈旧状态自动净化。
- **7 内置 tab**：文件工作台（懒加载目录树 + CodeMirror 编辑器 + 图片/MD+Mermaid/HTML/PDF viewer）、内嵌浏览器（多 tab 沙箱 iframe）、真实终端（xterm.js+node-pty，可选注入 terminal_* 工具）、Git 面板（行级红绿 diff + 历史 + 暂存/提交/还原）、**后台任务页（子代理树实时拓扑 + 批量实时预览 `subagents.live` 一次枚举整棵树 + 后台任务退出码/实时输出/强杀 + 新子代理自动展开侧栏）**、侧边对话（Codex 式侧线程，继承主会话完整上下文）、设置页。
- **服务化框架（核心理念）**：`ctx.betterSidebar` 开放 `registerTab` / `registerFileViewer`，内置 7 tab + 6 viewer 与第三方插件走同一套 API、能力完全对等 → 生态 28+ 插件（Excel 编辑页、Office 三件套预览、会话流程图实时可视化 Flowglass、文件改动审查页「行级 diff + 撤销 + chat 行深链」、本轮审查 turn-review、文件活动页等）。
- **模型主动打开**：设置开启后注入 `sidebar_open` 工具——模型可主动在侧边栏打开文件/文件夹（树以该目录为根）/网页；agent 终端由 agent 生命周期 reconcile。
- **其他**：懒加载（启动 ~325KB 核心，终端/编辑器/mermaid 重依赖按需拉）；声明式设置（每 tab 一张卡片独立开关 + 齿轮二级设置）；`@文件` 一键引用进输入框；产出文件 `revealInExplorer` 高亮定位。

**其他对标**：

| 对标 | 关键交互模式 | 对 Gaea 的启示 |
|---|---|---|
| M365 Copilot 2026 新设计（[官方博客](https://www.microsoft.com/en-us/microsoft-365/blog/2026/05/28/introducing-a-new-design-for-microsoft-365-copilot/)） | 面板=能直接改文档的"编辑伙伴"；画布↔聊天连续循环；统一入口+渐进披露 | gaea 的框选即改/Plan→Apply 方向正确；差距在循环不顺（预览面板与右面板互斥） |
| Gemini in Workspace 侧栏（[官方](https://workspaceupdates.googleblog.com/2024/06/gemini-in-side-panel-of-google-docs-sheets-slides-drive.html)） | 常驻右侧面板 + 建议提示词 | 右面板常驻可调宽，不是互斥切换 |
| Claude Artifacts | 版本历史+任意回退+并排对比 | gaea 产物只有 vN 次数徽标，最大短板 |
| ChatGPT Canvas | 选区→浮动快捷菜单；对话与画布双向同步 | docx 框选即改已有；缺选区→对话通用化 |
| LangGraph Studio（[官方](https://docs.langchain.com/langgraph-platform/langgraph-studio)） | 图拓扑+执行路径+中间状态+时间旅行 | 子代理图要能下钻"这步干了什么" |
| Devin / Manus | 计划面板+实时活动流+回放 | 多代理活动合并时间线；回放远期 |
| Codex（IDE/侧边对话） | 侧边线程继承主上下文、改动审查 | 远期：办公侧边对话 tab |

通用 UX 共识（[uxdesign.cc](https://uxdesign.cc/where-should-ai-sit-in-your-ui-1710a258390e)、[uxforai](https://uxforai.com/p/ux-best-practices-copilot-design)）：任务越重越要空间→面板三态；AI 面板是有状态工作区。

## 二、现状摸底摘要（短板清单）

- **右面板错位**：现有 4 主 Tab×8 面板（文件/资料/成本库/产物/变更/任务/分工/统计，`workspaceTabs.ts:50-91`）是"资料+状态展示"集合，与 better-sidebar 式工作台（实时操作面）定位不同；宽度 CSS 写死；与主区预览 pane 互斥。
- **双体系割裂**：ChatTabs（对话/轨迹/上下文）与右面板并行，AgentNetworkCard 只在 context 标签里、分工面板在右侧，同源数据互不联动。
- **子代理可视化**：仅两层图不渲染嵌套子树（`AgentNode.Children` 已有数据）；无耗时/进度%；无合并活动流；transcript 只读无工具调用跳转（v4.21 欠账）；子代理富化靠任务摘要前缀匹配（`gaea_ui_contextview.go:55-94`）。
- **文件交互**：无日常编辑 diff（ChangesPanel 只有文件+次数）；无版本时间线/恢复；pptx 只出不进；docx 渲染失败无降级；xlsx/docx 选区不能转对话；无"模型主动打开"。
- **成果交互**：产物登记靠前端启发式（写工具参数+正文白名单），后端无权威登记表；vN 不可点；Verifier 通道 B 无逐页缩略图（v4.16 欠账）。
- **已领先项（保持）**：docx 框选即改+Word 修订制、xlsx Plan→Apply+原生图表+CrossEmbed、证据链声明↔实况+回滚、任务中心实时输出、@文件 引用、专注模式。

## 三、提升项

### A. 右面板重造：运行工作台（对标 dsh-better-sidebar）★本轮主轴

**定位反转**：右面板从「资料+状态展示」改为「**实时操作工作台**」；状态显示类内容（统计、运行概览）迁到**主区 ChatTabs**——放轨迹/上下文旁边（第四个「概览」tab，或并入轨迹/上下文顶部条）。

**A0 框架先行**（一切的地基）：
- Tab 注册制服务化：前端建 `sidebarRegistry`（registerTab / registerViewer），面板框架与 tab 内容解耦，内置 tab 与未来扩展能力对等（gaea 无插件系统，先做代码级注册点，为板块 Manifest 化留缝）。
- 工作台交互底座：右栏宽度拖拽+持久化（复用 preview-resizer）；Tab 拆分/合并分栏（先右栏内分栏，底部面板与自由窗口列为远期）；按会话持久化布局+陈旧状态净化；声明式设置卡片（每 tab 独立开关）。
- 懒加载：重内容（xlsx 渲染、docx-preview、图表）按需拉取，保持首屏轻。

**A1 分工/子代理拓扑 tab**（AgentNetworkCard+SubagentsPanel 合体进化）：
- 树形实时拓扑（渲染 Children 嵌套、展开/收起），运行中批量实时预览（学 `subagents.live`：一次枚举整棵树，单轮询，消灭 O(N²)）。
- 节点量化：耗时（running 显示已用时）+ 进度%；新子代理出现自动展开侧栏到本 tab（可关）。
- 下钻链：节点→活动行→transcript 定位到对应工具调用（收 v4.21 欠账）；合并活动流（Devin 式单列实时 feed）。
- 卡片点击联动：右面板内高亮对应子代理，不再跨体系跳转。

**A2 浏览器 tab**（gaea 已有 CDP 控制 Edge + 7 工具面，缺观察窗）：
- 受控页面观察窗：当前页截图流/URL/标题 + 操作时间线（导航/点击/填表/提取）+ 权限门状态；自动操作时自动弹出（可关）。
- 与 better-sidebar 的"用户浏览 iframe"不同：gaea 是 agent 驱动，先做观察窗；iframe 实时镜像与人工接管列远期。

**A3 文件工作台 tab**（吸收现文件面板+预览能力）：
- 资源管理器（FileTree 已有）+ **编辑器 tab 化**：文件树点开 → 右栏内多文件 tab（复用 FilePreview：docx 保真/xlsx 结构化/md/图片），@文件 引用已有。
- **红线（用户拍板 2026-08-31）：Word/Excel 等文件编辑能力全量保留，换壳不换芯。** 编辑器 tab 必须完整移植现有编辑面，一项不砍：
  - docx：框选即改（预设润色/精简/翻译/扩写+自定义指令）→ AI 替换双栏预览 → Word 修订模式写入 → 接受/拒绝全部修订（`DocxPreview.tsx`）；
  - xlsx：Excel 式直接编辑（双击/fx 栏）、行/列插入删除、AI 编辑审阅制 Plan→Apply（单元格级 diff）、原生图表一键生成+CrossEmbed 嵌 Word/PPT、LibreOffice 重算、>300 行虚拟滚动、多 sheet/合并单元格/NumFmt 还原（`XlsxPreview.tsx`）；
  - md/text 内联编辑 + Ctrl+S 保存状态机；PDF/OCR 逐页进度预览；导出三出口。
- **双入口保留**：主区可拖宽预览 pane 不删除——右栏编辑器 tab 用于「边跑边盯」，预览 pane（或工作台加宽/全屏态）用于大屏深度编辑；两态共享同一编辑组件与状态。
- 变更 tab（Git 面板式）：写类工具统一前后快照 → 行级红绿 diff + 撤销/恢复（docx 复用 docxedit 修订基线、xlsx 复用 Plan diff、md/text 行 diff）。
- **模型主动打开**：新增 `sidebar_open` 类工具（打开文件/目录/网页），agent 主动把关键产物/页面推到右面板，用户不找。
- 产物行 `revealInExplorer`：产出文件在树中高亮定位。

**A4 迁移方案（现 8 面板去向）**：
| 现面板 | 去向 |
|---|---|
| 文件 | → A3 文件工作台（**docx/xlsx 编辑能力原样随迁，换壳不换芯**） |
| 资料/成本库 | → 保留（A3 的侧栏页或独立 tab，低优先） |
| 产物 | → 文件工作台的「产出」区 + reveal 高亮；证据链入口保留 |
| 变更 | → A3 变更 tab（diff 化） |
| 任务 | → 后台任务 tab（并入子代理页或独立，对齐 better-sidebar 任务页） |
| 分工 | → A1 子代理拓扑 tab |
| 统计 | → **主区**：轨迹/上下文旁边的「概览」tab |

### B. 办公文件交互（并入 A3 执行，此处只列增量）
1. 文件版本时间线：产物 vN 徽标可点 → 逐版本列表+预览+恢复（Artifacts rewind 模式）。
2. pptx 最小交互：大纲+每页缩略图预览（python-pptx 管线已有），点页→"针对第 N 页修改"指令。
3. 选区联动补全：xlsx/docx 选区→SelectionToComposer 转对话；docx 渲染失败降级纯文本视图。

### C. 成果交互
1. 后端权威产物登记表：写类工具落盘时登记（路径/来源轮次/工具/时间），前端只读该表，修启发式漏登。
2. 对话↔产物双向引用：产物→来源消息已有，补对话内文件链接→打开右面板定位。
3. Verifier 通道 B 逐页缩略图（v4.16 欠账）。
4. 远期：办公侧边对话 tab（Codex 式侧线程继承主会话上下文）。

## 四、分期建议（对齐现有版本节奏；v4.22.0 已被「一次性收官」占用，故顺延）

| 版本 | 主题 | 内容 | 状态 |
|---|---|---|---|
| v4.23 | 工作台框架 | A0（注册制+宽度拖拽+设置卡片）+ A4 迁移（统计→主区概览）+ 现有面板挂进新框架 | ✅ 已发布 |
| v4.24 | 子代理工作台 | A1 全部（树拓扑/live/自动展开/下钻链/活动流）+ C1 产物登记表 | |
| v4.25 | 文件工作台 | A3（编辑器 tab 化+变更 diff+模型主动打开+reveal）+ B3 选区联动 | ✅ 已发布（releases/v4.25.0.md） |
| v4.26 | 对话流式重造（插刀，对齐 Codex：根因「发送后对话窗静默而轨迹在动」六连对账——WorkHeader 工作态头部/后端 phase 接线/子代理活动回投/seq+GaeaResyncEvents 吞件防线/重复工具折叠） | ✅ 已发布（releases/v4.26.0.md） |
| v4.27 | Codex 对齐批（用户连续要求「对齐 Codex 右侧面板与对话输出」）：右栏文件工作台全高预览+宽度放开+编辑器 tab 图标、标签扁平化（删资料/成本库、取消二级标签）、对话输出（去气泡/回合分隔/复制/diffstat 芯片）、子代理对话实时下钻（SubagentThread 3s 轮询）、上下文标签完善（水位分色头部/空态/文件活动可点/悬停构成详情） | ✅ 已发布（releases/v4.27.0.md） | |
| v4.28 | 浏览器与版本（原 v4.27 顺延） | A2 观察窗（截图步进流+操作时间线+自动弹出）+ B1 版本时间线（vN 徽标 popover+时间线+预览/恢复，证据链同源）+ B2/C3 pptx 大纲卡+逐页预览+页级指令 | ✅ 已发布（releases/v4.28.0.md） |
| v4.29 | UI 化繁为简（用户点名主轴；**简化≠删功能红线**） | 顶栏导出三钮收拢单钮下拉（md/Word/PDF 出口全保留）+ 右栏 tab 窄栏自适应图标化（Notion Icon only 式，6 tab 集合不变）+ 预览头部降噪（打开/定位图标化+去边框，编辑等状态语义保留）——弹药：docs/market-research-2026-09-01b.md | ✅ 已发布（releases/v4.29.0.md） |
| v4.30 | UI 化繁为简第二刀（用户点名「继续优化完善」；收 v4.29 欠账） | 产物生成自动置前/角标（Devin Auto-open 式：新产物 tab 角标+行「新」徽标）+ 面板行级降噪（Cowork 一行式：产物/变更/任务次级信息悬停次行化）+ 命令面板按当前视图重排（Linear 式：lib/paletteRank 纯函数）+ 预览「半幅↔最大化」两档（VS Code Toggle Maximized Panel） | ✅ 已发布（releases/v4.30.0.md） |
| 远期 | 终端 tab、底部面板/自由窗口、侧边对话、iframe 实时镜像 | — | |

每版交付纪律沿用 .gaea/AGENTS.md 约定：releases/vN.0.md + README 更新日志表 + 测试钉住（vitest、fold/golden、mock 契约 drift）。
