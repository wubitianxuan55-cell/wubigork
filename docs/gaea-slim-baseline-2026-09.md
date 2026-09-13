# gaea 瘦身 P0 基线快照（2026-09-09 落档）

> **状态：✅ 已落档（P0 基线，2026-09-09）** —— 本文是瘦身规划的**基线证据层**，
> 记录 P0 阶段七面基线四表 + 功能等价快照表 + IA 现状走查的**实测数字与代码层事实**。
>
> **文档关系**：`gaea-slim-masterplan-2026-09.md` = 瘦身执行层（权威/规划）· **本文 = 基线证据层（实测入档）** ·
> `webview2-shell-audit-2026-09.md` = 壳内审计证据层 · `gaea-convergence-plan-2026-09.md` = 治理层。
> masterplan §0 的 2026-09-08 立项数字为「基线」；本文全部更新为 **v4.166.0（P1 版，已发布）当前仓库实测**。
>
> **适用版本**：v4.166.0（commit b20f88ff，2026-09-08）。前端产物为当日 `npx vite build`（28.06s）全新构建，
> 非增量残留。
>
> **数据口径**：本文全部数字来源于**实际读码与实测**（过期 dist 不存在，已重构建）；「代码层存在」≠ 真机行为
> 验证——真机行为不在 P0 本表范围，待 P2「三路可达走查（壳内真机）」补齐（masterplan §2 P2 出口判据）。
>
> **结论速览**：P1 四刀轮子收口全部实证成立（b64/slug/novel diff/双门）；资产面数字与立项基本持平
> （dist 10.0MB/entry 1.19MB/MemoryHub 1.4MB 无恶化）；exe 46.22MB 与立项 46.2MB 持平（strip 属 P4）；
> npm 直依赖 26→29 为 codemirror 顶包移除（-1）+ 四子包显式化（+4）的净效应；**「13 板块平铺单一导航面、
> 双空间并列结构未表达」实证成立**（详见 §3.4），并新增勘查出 rail 编程板块代码层双入口疑似项（§3.1）。

---

## 1. 七面基线四表

四列：**现状实测**（2026-09-09 实测，括号内为立项 2026-09-08 基线与 diff）→ **目标态（6 周，抄录 masterplan §0）** → **长期态（masterplan）**。每面下方为更新注记。

### 面一 · 认知（信息架构）

| 现状实测（v4.166.0） | 目标态（6 周） | 长期态 |
|---|---|---|
| 13 一级板块平铺单一导航面：work5（gaea/cost/memoryhub/weixin/schedule）/ play4（chat/novel/imagegen/characterlib）/ shared3（home/modelcenter/settings）/ indep1（code）。**双空间并列结构未表达**（与立项一致）。 | 双空间切换一级可见、各自导航：工位 ≤6（schedule 并入办公文档面）、乐园 ≤4，shared/independent 归各自壳层 | 每空间板块数只减不增；新域先问「属于哪个空间、能否是文档」 |

- 证据：`frontend/src/boards/manifests.ts:74-143`（canonicalBoards 13 条，space 全齐）；
  后端权威清单 `internal/app/board/builtins.go:20-164`（12 业务 + knowledge D7，13 条）。
- 拍板红线复述：工位与乐园**并列平级**，瘦身≠「收乐园保办公」；双眼看板的实证走查见 §3。

### 面二 · 资产（前端体积）

| 现状实测（v4.166.0） | 目标态（6 周） | 长期态 |
|---|---|---|
| dist **10.0MB / 227 文件**（assets 225：208 JS + 17 CSS）（立项 10.03MB/224，≈持平）；**entry 1,192.79kB**（gz 375.36）；**MemoryHub chunk 1,436.11kB**（gz 388.51）；>500KB chunk ×5（entry/MemoryHubPage/GaeaPage 617/mermaid.core 622/cynefin 691） | entry 与 MemoryHub 双拆；locale 死键清零（**P1 已完成**，见注 3） | 首屏恒 ≤ 靶值；重库恒按需 |

- 注 1：构建路径 `frontend/vite.config.ts:59-62` outDir=`../dist`（仓库根 dist），emptyOutDir 全新产物，28.06s。
- 注 2：MemoryHub 1.4MB 解剖（立项怀疑「3d-force-graph/cytoscape/cynefin 同包」）——本构建实测：
  `MemoryHubPage-Ap5_JR8G.js` 1,436.11kB 含 GraphView 静态引入的 **3d-force-graph**（`GraphView.tsx:2`）；
  **cytoscape（443.72kB）与 cynefin（690.83kB）已独立成 chunk**，不在 MemoryHubPage 内。立项「按 tab 动态 import」对应
  P4 目标，现状 3d-force-graph 仍随页首载。
- 注 3：**mermaid(621.80kB)/katex(261.33kB)/CodeEditor(317.84kB) 仅宿主视图加载** 立项悬项，本构建实证成立：
  mermaid 仅 `utils/mermaidPng.ts:5` 与 `gaea/components/Markdown.tsx:27` 引入；katex 仅 `Markdown.tsx:25`；
  CodeEditor 经 `gaea/components/FilePreview.tsx:30-31` React.lazy 懒载。
- 注 4：locale 死键 P1 已清零（commit b20f88ff：1445 键 0 死，静态 1438 + 动态 7，三语同步删 669 行）。

### 面三 · 结构（架构止血）

| 现状实测（v4.166.0） | 目标态（6 周） | 长期态 |
|---|---|---|
| office 包枢纽 **272 入/70 出**（立项数字，未经工具复测，标注待 P3 前复测）；>50KB 源文件**实测 16 个**（立项列举 9 个全数仍在，新增 7 个，见注） | 抽公共内核；巨文件首批 4 个拆掉（App/bridge/types/GanttView） | >50KB 源文件 ≤3；枢纽入度 <100 |

- 注：实测 >50KB（排除 clones/.tmp/dist/node_modules/wailsjs/测试）：App.tsx 84.6 / bridge.ts 85.1 / **locales en.ts 84.7、zh.ts 82.4、zh-TW.ts 82.5（新增）** / types.ts 66.4 / GanttView.tsx 64.8 / KnowledgePanel.tsx 62.2 / config.go 58.7 / DeliverablesPanel.tsx 57.4 / XlsxPreview.tsx 56.4 / ChapterPage.tsx 55.9 / **herdsmanTemplates.ts 55.7、engine.go 55.2、store.ts 54.3、SchedulePage.tsx 52.7（新增）**。
- 立项 9 个巨文件尺寸逐一与当前一致（85/85/66/65/59/62/57/56/56 KB），说明 P1 未触结构面（符合阶段顺序）。

### 面四 · 轮子（重复收敛）—— P1 四刀已清零/已收口 ✅

| 类 | 立项基线（2026-09-08） | v4.166.0 实测（2026-09-09） |
|---|---|---|
| b64 解码 | ×18 处 | **已收口**：全仓 `atob(` 仅 2 处——canonical `bytes.ts:15`（b64ToBytes）+ `mock/office.ts:534`（mock 后端面，保留清单），约 20 处手搓解码收口 |
| title-slug | Go×5 | **已收口**：`internal/gaea/strutil/title_slug.go:33` TitleSlug 唯一规范；cost.SlugName（`cost.go:683-684` 保留 cost 兜底）、knowledgeimport.slugName（`knowledgeimport.go:487`）、app.slugFromTitle（`gaea_knowledge_import.go:243`）、modelengine.customSlug（`engine.go:608` 保留 ASCII 白名单前置）、controller_memory.slugifyName（`controller_memory.go:425`）全部并入；legacy golden matrix 锁死五旧实现（`title_slug_test.go:44-122`）；schedule.SafeSlugName（`schedule/index.go:170`）/ memory.slugify **语义不同不并（保留）** |
| novel 行 diff | 手搓弱贪心 ×1 | **已收口**：迁 `frontend/src/gaea/lib/diff.ts` LCS（3 行窗口漂移边界用例 + 空串 0 行口径锁死） |
| 下载出口 | ×5 | **已收口**：`gaea/lib/saveFile.ts`（downloadBlob/dataUrlToBlob/saveExportBlob）统一；审计刀 B 下载类 ×4（ImageGen×2/NovelSetting/办公 md）并入 exportConversation 管线 |
| 选取点 | ×8 | **已收口**：`gaea/lib/pickFile.ts` + `inShellEnv` 唯一判定；审计刀 C 上传类 ×3（ControlPanel/VisionTrial/SkillModal）壳内 GaeaPickFiles |
| 双轨 bridge | 手工路由 vs wailsjsCompat | **未动（P3 渐进退役）**：`bridge.ts` 手工路由 602 行仍在 |

| 目标态（6 周） | 长期态 |
|---|---|
| 重复清单清零或归入保留清单（**P1 已达成：四刀全收**；剩余=双轨 bridge 跨版本退役 + XlsxPreview 虚拟滚动评估池） | 防复发规约 7 条（masterplan §4，AGENTS 已回写）+ eslint 门禁（P2 起） |

### 面五 · 运行（启动与内存）

| 现状实测（v4.166.0） | 目标态（6 周） | 长期态 |
|---|---|---|
| 冷启动（首帧/可交互）/ 常驻内存**仍无基线**（与立项一致，P4 建基线）；`app.New()` 全模块初始化未改；此次实测未采集运行指标（属真机走查，非本表范围） | 基线 + 懒初始化改造；可交互 -30% | 季度复测防回归 |

### 面六 · 产物（二进制与依赖）

| 现状实测（v4.166.0） | 目标态（6 周） | 长期态 |
|---|---|---|
| **exe 46.22MB**（`releases/gaea-v4.166.0.exe`，2026-09-08 22:21:58；立项 46.2MB ≈ 持平，strip 未做，属 P4）。**npm 直依赖 29**（立项 26 → codemirror 顶包移除 -1 → @codemirror/state|view|commands|language 四子包显式化 +4）；**lock 699 包**（立项 701，净 -2）。存疑名单 gsap/docx-preview/unist-util-visit 验活全部洗清（commit b20f88ff） | 发布版 strip 实测；依赖逐个验活（已验，P1 完成） | exe ≤40MB；直依赖零死重 |

- 注：直依赖 29 明细见 `frontend/package.json:13-43`（11 个 @codemirror/* + antd/3d-force-graph/d3-force/docx-preview/dompurify/gsap/mermaid/react/react-dom/react-markdown/react-window/rehype-katex/rehype-raw/rehype-sanitize/remark-gfm/remark-math/unist-util-visit/zustand/…）。

### 面七 · 数据/知识

| 现状实测（v4.166.0） | 目标态（6 周） | 长期态 |
|---|---|---|
| `.gaea/progress.md` **242.7KB**（立项 244KB，-1.3KB）；持久化四范式（SQLite/JSON/JSONL/index 缓存）并存未变；**clones/ 实测 212.5MB** 占仓目录（立项确认，移出属 P5） | 知识外化三件套（30 分钟上手文档 / progress.md 瘦身归档 / 个人路径清洗） | 文档/数据总量与信噪比纳入季度审计 |

---

## 2. 功能等价快照表（13 板块 × 核心动作可达性）

> **板块清单口径**：`manifests.ts:74-143` canonicalBoards 13 条（home + 12 业务）即一级导航全集；
> 后端另有 knowledge（D7）独立板块，前端归一过滤不进一级导航（`manifests.ts:248-249`），单独注记。
> **可达性口径**：rail=左侧指挥轨道（inMenu 全量）；首页=ModuleLauncher Bento；快捷键=运行时 Ctrl+1~9
> （当前空间菜单位序，`MainLayout.tsx:496-503`）+ manifest 声明 ctrl+1~4；隐式=右上角/模型 pill 等。
> **实现状态**：代码层抽查（页面文件注册 = `main.tsx:16-31` 全部存在；动作关键词抽查见下）；真机行为不在本表范围。

| 板块 id | label | space | 核心动作（用户视角） | 可达性（入口路径） | 实现状态（代码层，证据） |
|---|---|---|---|---|---|
| home | 首页 | shared | 从启动器进入各板块；命令条打字/语音对话；遥测/写作进度/晨报查看 | 默认落地页（每次启动从 home 开始，`MainLayout.tsx:389`）；rail 首位；无快捷键 | ✅ ModuleLauncher（`MainLayout.tsx:746-751`，`ModuleLauncher.tsx:289+`） |
| chat | 聊天 | play | 发消息（流式）；切话题/历史；人格切换；语音 | rail/首页瓦片/manifest ctrl+1（运行时 play 空间 Ctrl+1 亦为 chat） | ✅ ChatPage：发送管道 `ChatPage.tsx:218-234`、话题状态机 `:46` 起、人格状态条 `:374-410` |
| novel | 小说 | play | 书架浏览与「继续阅读」；设定/角色/创作/阅读；导出 | rail/首页瓦片/manifest ctrl+2 | ✅ NovelPage 5 tab（`NovelPage.tsx:45-49`）+ ChapterPage(55.9KB)；后端 nav 含「导出」（`builtins.go:41`） |
| imagegen | 绘梦 | play | 生成（文生图/img2img/t2v）；历史/灯箱；素材库；模板 | rail/首页瓦片/manifest ctrl+3（play 空间 Ctrl+3） | ✅ ImageGenPage 3 分区工作台 + 生成队列/历史/模板 hook（`ImageGenPage.tsx:2-7,247-426`） |
| gaea | 办公 | work | 任务制工作台；文档/造价/进度计划 AI 编排；命令面板 | rail/首页旗舰大卡（4×2）/manifest ctrl+4 / Ctrl+K 命令面板 | ✅ GaeaPage→gaea/App.tsx：CommandPalette（`gaea/App.tsx:44,1727`）；chunk 617.22kB |
| cost | 造价数据库 | work | 条目增删改；搜索/筛选；组价/测算；价格源订阅；价格仓库 | rail/首页瓦片 | ✅ CostLibraryPage：新建条目 `:218,426`、价格源订阅 `:571`；后端 nav 7 子面（`builtins.go:76-81`） |
| code | 编程 | independent | 启停 Harness Web；桌面内嵌工作台；重载；打开 Harness 目录 | rail 主体 + foot 独立单列（代码层双入口，见 §3.1）；首页宽瓦片（独立窗口徽标，`ModuleLauncher.tsx:519-529`） | ✅ ProgrammingPage：启动/内嵌工作台（`ProgrammingPage.tsx:294-363,406-425`） |
| memoryhub | 记忆中枢 | work | 知识库/用户画像/办公记忆/项目资料/聊天记忆/记忆图谱/数字生命 | rail/首页瓦片 | ✅ MemoryHubPage 7 子面板（`MemoryHubPage.tsx:18-24`）；chunk 1,436.11kB |
| modelcenter | 模型中心 | shared | 模型配置/引擎管理/功能绑定/调用统计/故障转移 | rail/首页瓦片/**顶栏模型 pill 点击**（`MainLayout.tsx:660-665`） | ✅ ModelCenterPage 引擎控制台 5 分类（`ModelCenterPage.tsx:187-195`） |
| characterlib | 角色库 | play | 角色档案 CRUD；检索/筛选；导入项目；一键补齐；设为聊天人格 | rail/首页瓦片 | ✅ CharacterLibraryPage 3 分区（`CharacterLibraryPage.tsx:274-335`，新建/删除/导入/补齐 `:178-242`） |
| settings | 设置 | shared | 通用/聊天/小说/绘梦/办公/模型/安全/数据/关于 9 面 | **隐式入口**：右上角设置按钮（`inMenu:false`，`MainLayout.tsx:701-703`）+ 首页瓦片（`ModuleLauncher.tsx:530-537`）；不在 rail | ✅ SettingsPage + SETTINGS_NAV 9 子页（`manifests.ts:49-55`） |
| weixin | 青鸟 | work | 扫码绑定微信；离线代办提醒；多助手管理（新增/启停/删除/改人格） | rail/首页瓦片 | ✅ WeixinPage：扫码三步流 `:137-348`、离线提醒 `:534-583`、多助手管理 `:29-31` |
| schedule | 进度计划 | work | 横道图/单代号/双代号三视图；倒排工期；基线对比；导出 | rail/首页瓦片 | ✅ SchedulePage：三视图 tab（`SchedulePage.tsx:703-705`）、EXPORT_KIND（`:129-131`）、倒排/竣工线（`:926-971`） |
| *knowledge*（非一级导航） | 知识库 | work | 显式知识条目管理与检索（knowledge_search/add 工具） | **无一级入口**：`normalizeManifests` 过滤（`manifests.ts:249`）；等价可达=记忆中枢「知识库」子面（KnowledgePanel）；KnowledgePage 组件已注册但**无任何引用（孤儿页）**（`main.tsx:29`，后端注 `builtins.go:149`） | ⚠️ 组件在（`KnowledgePage.tsx:13-22`）但一级导航隐藏（3.0 定制），功能等价走记忆中枢子面账 |

**汇总**：13 个一遍板块全部有代码层实现且页面文件存在；12 个一级可达（11 经 rail + settings 隐式）；
knowledge 为受控隐藏的维度（功能并入记忆中枢，等价性成立，但**孤儿页组件与 knowledge 工具链真机通路**列入 P2 走查项）。

---

## 3. IA 现状走查（rail 实际显示 / 默认空间 / 首页陈列 / 双空间并列实证）

> 目的：P2「双空间并列落地」「home 空间感知化」改造前的现状基线。全部为代码层事实（v4.166.0）。
>
> **v4.169.0 重走查（2026-09-09 更新）**：本版 P2 主干落地后，§3.1 rail 双入口、§3.3 首页平铺、
> §3.4 并列缺口三处已闭合（详见 docs/gaea-slim-p2-dualspace-2026-09.md）——rail 顶部新增工位/乐园切换器、
> rail 主体按空间分域（code 独立仅 foot 单列=唯一入口）、首页 Bento 按空间过滤+每空间旗舰+工位最近文档；
> 空间感知点由 3 处扩至 6+（navigateBoard 自动切空间 / Ctrl+N / 每空间页面持久化 / keepAlive 剪枝 / **rail 切换器显式切换** /
> **Bento 分域** / **home 空间 chip+最近文档**）。真机渲染实态仍待壳内走查（见审计文档 §6 待真机清单）。

### 3.1 rail（左侧指挥轨道）实际显示

- **结构**：`v3-rail-dock` 自动隐藏 OS 式 dock（默认收进左缘，滑过展开，`MainLayout.tsx:599-607`）；CommandRail 固定窄栏纯图标
  （`MainLayout.tsx:288-379`）。
- **显示内容**：`getActiveMenuBoards()` = **全部 inMenu 板块 12 个**（home/chat/novel/imagegen/gaea/cost/code/memoryhub/
  modelcenter/characterlib/weixin/schedule），按 menuOrder 平铺（`MainLayout.tsx:296`；派生 `manifests.ts:156-160`）。
  纯图标、无文字标签（title/aria-label 有），**无分组**——不分空间、无分隔线（仅 foot 一条）。
- **foot 独立区**：独立板块（code）divider 后单列 + 深色模式切换（`MainLayout.tsx:342-376`）。
- ⚠️ **勘查发现【代码层双入口疑似项】**：code 板块 `inMenu:true`（`manifests.ts:110`、后端 `builtins.go:89`），
  `getActiveMenuBoards()` 不过滤 independent → code **同时出现在主体列表第 7 位与 foot 独立区**。
  注释语义为「独立窗口单列」（`MainLayout.tsx:294-295`），与代码实为双渲染相悖；真机最终渲染待 P2 壳内走查确认。
- **空间切换入口：已移除**（v4.3.2c 注释 `MainLayout.tsx:294-295`「空间切换入口已移除，首页三栏即空间入口」）；
  `SHELL_SPACES` 词条（`space.ts:27-36`）**无任何组件消费**（仅测试引用 `space.test.ts:13-14`）——现无显式空间切换器 UI。

### 3.2 默认空间与空间切换表达

- **默认空间 = work（工位）**：`loadShellSpace()` 读 localStorage `gaea.shell.space`，非法/缺失回退 `'work'`
  （`appStore.ts:168-174`；键定义 `:125`）。
- **空间切换表达（现状）**：
  1. **隐式自动切换**（主路径）：`navigateBoard` 按目标板块 `manifest.space` 自动 setSpace + 落地目标页
     （`MainLayout.tsx:537-554`；work→work/play→play 不切换，shared/independent 保持当前空间）；
  2. **Ctrl+N**：显式切到乐园并落 novel（新建项目流，`MainLayout.tsx:505-513`）；
  3. **每空间最后页面持久化**：`gaea.shell.page.<space>`，切空间按空间恢复（`MainLayout.tsx:33-50,404-417`）；
  4. **性能门控**：切空间剪枝 keepAlive 页面（`pruneVisitedForSpace`，`space.ts:72-77`）。
- 无显式的「工位/乐园」一级切换控件（见 3.1 末项）——空间切换只发生在**点击/快捷键导航**的副作用中。

### 3.3 首页陈列（home 板块 = ModuleLauncher）

- **左舷**：Hero 命令条（打字 + 语音 + 发送 + ⌘K 徽标）+ 能力矩阵 Bento：
  **gaea 旗舰 4×2 大卡** → 9 块普通瓦片（chat/novel/imagegen/cost/memoryhub/modelcenter/characterlib/weixin/schedule）
  → **code 宽瓦片**（span 4×1，独立窗口徽标）→ **settings 瓦片**收尾（`ModuleLauncher.tsx:294-298,512-541`）。
- **右舷**：内核遥测（模型/引擎/CPU·MEM·GPU）+ 写作进度环 + 最近会话 + 记忆脉搏；**晨报仅 work 空间渲染**
  （`ModuleLauncher.tsx:628`）。
- ⚠️ **空间过滤：无**。`allModules = deriveLauncherModules(activeBoards, LAUNCHER_DESC)` **不传 space**（`ModuleLauncher.tsx:294`）
  → 首页 Bento 全量 12 板块**平铺**，无「左翼书房/右翼庭院」分区；`space` prop 仅用于晨报 work-only 条件。
  注：`navigateBoard` 注释与 manifest 注释中「首页三栏/左翼书房板块→work、右翼庭院板块→play」
  （`MainLayout.tsx:534-536`、`builtins.go:144`）为 v4 重构前的旧语义，**与当前单 Bento 平铺实现不符，注释未同步**。

### 3.4 双空间并列结构现状（P2 改造前基线 · 实证）

- **manifest 空间标注**：13 板块全齐（work5/play4/shared3/indep1；`manifests.ts:74-143`、`builtins.go:20-164`）——
  数据层双空间模型（`space.ts:19-64`，含 `filterBoardsForSpace`/`isBoardReachableInSpace`/`getActiveMenuBoardsForSpace`）**已就绪**。
- **UI 表达实证**：❌ 未落地——①rail 平铺 12 板块不分空间（3.1）；②首页 Bento 平铺 12 板块不分空间（3.3）；③无空间切换器 UI（3.1 末）。
  当前**仅 3 处空间感知**：
  1. `navigateBoard` 按目标板块自动切换壳层空间（导航语义按空间，`MainLayout.tsx:537-554`）；
  2. Ctrl+1~9 按**当前空间**菜单位序定位（`MainLayout.tsx:497` `getActiveMenuBoardsForSpace(space)`）；
  3. 晨报 work-only + 跨空间 keepAlive 剪枝。
- **结论**：masterplan 拍板「13 板块平铺单一导航面，双空间并列结构未表达」（`masterplan §0` 行一、`§1 轨道一·2`）
  在 v4.166.0 代码层**实证成立**。并列缺失点 = 空间切换器、分域导航（rail/Bento 双处）、home 空间感知
  （现状是单一首页平铺，非「工位=最近工程文档待办 / 乐园=最近会话小说创作」）。
- **P2 走查待证项**（壳内真机）：rail code 双入口渲染实态；后端清单合并后 nav 子项差异
  （cost/novel 子导航后端 7/6 项 vs 前端静态 4/5 项——运行时以后端为准，`manifests.ts:263-266`）；knowledge 孤儿页确无入口。

### 3.5 三路可达性汇总

| 入口 | 覆盖板块 | 备注 |
|---|---|---|
| rail（左侧轨道） | 12（全部 inMenu；settings 除外） | 平铺无分组；code 主体+foot 双入口（疑似） |
| 首页 Bento | 13（12 瓦片 + settings） | 全量平铺；gaea 旗舰/code 宽瓦/settings 收尾 |
| 快捷键 | manifest 声明 ctrl+1~4（chat/novel/imagegen/gaea，`manifests.ts:83,88,94,100,195`；测试 `manifests.test.ts:88-93`）；运行时 **Ctrl+1~9=当前空间菜单位序**（`MainLayout.tsx:496-503`）；Ctrl+K 全局搜索（`MainLayout.tsx:516-523`，gaea 板块让位）；Ctrl+N 新建项目→乐园 novel | manifest shortcut 字段为声明数据，运行时实际按当前空间位置取数 |
| 隐式入口 | settings（右上角按钮）、modelcenter（顶栏模型 pill）、memoryhub 知识库子面（knowledge 等价可达） | settings `inMenu:false` |

---

## 4. 后续更新时机（靶值对拍）

| 版本/阶段 | 更新内容 | 对拍项 |
|---|---|---|
| **P2 形态版（v4.169.0）** | §3 全章重走查（✅ 已落：双空间 rail 分域+切换器 / home 空间感知 / code 双入口闭合 / knowledge 孤儿页删除 / G-2 基线数；剩壳内真机渲染实态）；§2 等价快照表 13 板块可达性不变（settings/模型中心 隐式与 pill 入口保持） | IA 三节 + 等价表；rail code 双入口✅确认；knowledge 孤儿页✅处置 |
| **P4 性能版** | §1 面二（entry/MemoryHub 拆解后体积）/ 面五（冷启动基线 + 可交互 -30% 对靶）/ 面六（exe strip ≤40MB 对靶，以本文 46.22MB 为实测定靶） | exe≤40MB、首屏靶值、entry/MemoryHub gz |
| **每版度量看板** | masterplan §3 六行（每空间一级板块数 / entry gz / 冷启动可交互 / 常驻内存 / exe / >50KB 源文件数）按本文件口径每版落账 | 防回归 |
| **P5 维持/季度** | §1 面七（progress.md/clones/持久化范式）季度复测；防复发规约 eslint 门禁执行情况 | 度量看板不劣化 |

> 落档记录：2026-09-09，基于 v4.166.0（commit b20f88ff）实测；前端 dist 为当日 28.06s 全新构建
> （`npx vite build`，outDir=仓库根 `dist/`）。

---

## 5. 冷启动基线首采（2026-09-12，维持轨「冷启动基线采集」清账）

> 采集脚本 `.tmp/coldstart.mjs`（spawn→CDP 端口→Wails 绑定注入→DOM 文本就绪 四段计时，taskkill 后冷起，连测 3 轮）；对象=v4.253.0 exe（49,038,848B）。**口径注意**：`DOM 文本就绪`（body.textContent>500 字符）≠ 首帧像素/数据满载渲染，仅为本脚本的稳定可复现指标。

| 轮 | spawn→CDP | CDP→绑定 | →DOM 就绪 | spawn→DOM 总计 |
|---|---|---|---|---|
| 1 | 470ms | 252ms | 24ms | **1054ms** |
| 2 | 695ms | 246ms | 17ms | **1070ms** |
| 3 | 813ms | 252ms | 18ms | **1198ms** |

> **结论**：冷启动 DOM 就绪稳定在 **1.05~1.2s**（绑定注入≈0.25s）。背景：此前的启动链同步段（MCP 串行无 deadline/微信同步 notifyStart/远程剧照串行下载/逐文件拷贝迁移）已在 v4.251/252 清偿（普查档二遍扫描节）——本次为首采基线，供后续「可交互 -30%」对靶与季度防回归复测。复测口径=同脚本同轮数。
## 6. 冷启动基线复测（2026-09-13，维持轨二次清账）：代码零回归，环境态漂移

> 起因：原罪板块七刀（v4.262~v4.267）后例行复测，v4.267.0 exe 三组×三轮同脚本测得
> **绑定→DOM 就绪 409~442ms（基线 17~24ms）、总耗时 1.36~1.67s（基线 1.05~1.2s）**，
> 疑似回归。

| 组 | exe | spawn→CDP | CDP→绑定 | 绑定→DOM | 总耗时 |
|---|---|---|---|---|---|
| 基线（09-12 夜，§5） | v4.253.0 | 470~813 | 246~252 | **17~24** | 1.05~1.20s |
| 复测×3组（09-13 凌晨） | v4.267.0 | 331~900 | 221~383 | **409~442** | 1.36~2.10s |
| **对照实验**（v4.253 原版代码今晨重建同测） | v4.253.0 | 520~1341 | 247~394 | **414~423** | 1.37~2.12s |

> **结论：代码零回归。** v4.253 原版代码在同一时刻、同一机器、同口径测得的
> 绑定→DOM 与 v4.267 完全一致（~420ms）——+400ms 是**环境态漂移**（基线首采后
> 4 小时内经历数十次构建/走查，WebView2 profile 缓存与 AV 扫描状态已变），非代码
> 退化。插桩旁证（.tmp/coldstart-probe.mjs）：entry 求值仅 15ms、无长任务、无慢
> 资源，HTML→load=117ms，FCP=636ms——空窗是异步等待而非 JS 求值；且 entry 体积
> 自基线以来不增反降（1,192.79kB→752kB，MemoryHub/GaeaPage 拆包生效）。
> **后续口径**：①基线数字与复测数字都真实，但跨环境不可直接对比——「可交互
> -30%」对靶与季度防回归**必须同环境对测**（同一次开机窗口内新旧 exe 各测一组）；
> ②环境稳态后（下次正常开机）可再采一轮，若仍 ~420ms 则把 §5 基线修订为含环境
> 噪声的复现区间；③本次对照实验脚本留存 .tmp/coldstart-probe.mjs（起壳后读
> Navigation/Paint/Resource/长任务四类计时）。

## 7. 每版度量看板（§面五规矩「每版六行」恢复落账，自 v4.267.0 起）

> 口径=masterplan §3 六行；基线面五之后 v4.254~v4.266 未逐版落账（流程滑步，本节起
> 恢复逐版执行）。「每空间一级板块数」按 manifests 统计（含未进菜单板块，shared/
> independent 另计）。

| 版本 | 板块数(工位/乐园) | entry chunk(gz) | 冷启动可交互 | 常驻内存 | exe | >50KB 源文件 |
|---|---|---|---|---|---|---|
| **v4.273.0**（2026-09-13） | 5 / 5（同 v4.267，manifests 未动） | **243.6 KB**（v4.267=243.0；+0.6KB≈UX 三刀纯 CSS，零回归） | 1.39~1.70s 六轮（bind→UI 405~424ms；与 v4.267 同区间） | 139.1 MB 工作集 / 152.9 私有（采样时办公工作台开着用户真实工程，环境态漂移跨环境不可比，口径同 v4.267 注） | 47.0 MB（49,292,288B） | **6**（同 v4.267：locales×3+control/controller.go+lib/store/controller.ts+novel-workspace.css，零变化） |
| ↳ 注（2026-09-13） | **v4.268~v4.272 未逐版采集（流程滑步再现，CHANGELOG v4.252/253 漏记同型）**：期间 v4.268=knip 零行为、v4.269=Go 侧 wire 修复、v4.270~272=sin 工具/UX CSS 刀。自本行起恢复逐版落账。 |
| **v4.267.0**（2026-09-13） | 5 / 5（工位含 schedule 未进菜单；shared 3、independent 1） | **243.0 KB**（基线 375.36，-35%，MemoryHub/GaeaPage 拆包生效） | 1.36~1.67s（bind→UI ~420ms；环境态漂移见 §6，代码零回归，跨环境不可比） | **125.3 MB 工作集** / 139.5 私有（首采=P4 建内存基线） | 47.0 MB（49,281,024B，strip 未做属 P4） | **6**（基线 16，巨文件拆解生效；余=en/zh/zh-TW locales+control/controller.go+lib/store/controller.ts+novel-workspace.css） |
| 基线参照（§5/面三，v4.253 期） | — | 375.36 KB | 1.05~1.20s（§5 口径注：DOM 文本就绪≠首帧） | 无基线（本节首采） | 46.22 MB | 16 |
