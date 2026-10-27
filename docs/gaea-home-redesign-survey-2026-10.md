# gaea 首页重设计 · 市场调研（2026-10）

> 状态：调研稿 v1 · 2026-10-26 · **方向一已拍板并落地（见 §8）**
> 缘起：用户判断现行首页「过于复杂、不符合潮流」，要求「简约大气」。本稿先做市场调研与方向候选，**重设计方向待拍板后再动工**。
> 现行基线：v4.476 全卡同权形态 + 2026-10-26 刀6（工具行前置+板块节分家，45e5ff91，未发版）。

---

## 0. 一句话结论

首页的复杂不在玄墨视觉语言（衬线大标题/描边退场/单色点睛恰好是 2025~2026 潮流本身），而在**信息结构**：一屏六区、12+2 张入口卡、四节状态仪表带，让首页同时当了「导航页＋仪表盘＋启动器＋创作辅助」四个角色。市场共识是把首页收敛为**一~三件事：开始（输入）＋继续（最近）＋需要你处理的（待办）**，导航归还常驻左栏，状态归还遥测轨与监测板块。

---

## 1. 现状：首页在承担四重职责

现行 classic 形态（`ModuleLauncher.tsx` DeskHome，一屏六区）：

| 区 | 区块 | 职责 |
|---|---|---|
| 顶 | HomeStrip（形态切换+日期） | 壳层开关 |
| ① | w-hero 指挥卡（印章「文」+「文书台」标题+lede+pill+命令条+气泡+语音状态行） | 问候＋工作启动 |
| ② | w-docs 最近文档卡（≤6 行）＋w-aside 写作进度环 | 继续＋创作辅助 |
| ③ | home-tools 工具节（CAD/进度 2 tile，2026-10-26 新增） | 工具入口 |
| ④ | home-boards 板块节（12 张全卡同权模块卡） | 导航 |
| ⑤ | w-stat-row 状态卡排（内核状态/最近会话/记忆脉搏/任务收件箱四节） | 仪表盘 |

与 2025~2026 潮流的四个差距：

1. **密度**：六区 vs 潮流「一屏三件事」（问候+输入+建议词）。AI 产品新会话空态几乎全部收敛到 1~3 个信息组。
2. **形态**：12 卡「能力矩阵宫格」属广场式首页——该形态 2026 年被市场集体证伪（豆包/千问/元宝下架智能体广场；Apple Photos 因杂乱反弹被迫回退；Facebook/Samsung 明确以 less cluttered 为改版目标）。且 12 卡与常驻左栏 rail 的导航职责完全重复。
3. **缺潮流第三层**：hero 命令条下方没有**场景建议词 chips**——这是 ChatGPT/Gemini/Claude/豆包/Kimi 共有的标准层（NN/g 有专门规范），既能消灭空白页焦虑，又是模块/工具能力的柔性入口。
4. **仪表盘职责越位**：四节状态里只有「任务收件箱」属于「需要你处理的」；内核/会话/记忆是自嗨型状态，与壳层恒显遥测轨、运行监测板块三重叠（v4.473 已为此退过一次 CPU 条）。

**明确不过时的部分**：玄墨 v11 视觉体系（三层层级/描边退场/金三处点睛/衬线大标题/暖 off-white 方向）与 calm UX、typography-first、60-30-10 用色趋势高度一致——本轮是**结构减法**，不是视觉重画。

---

## 2. 市场调研

### 2.1 国际 AI 产品：空态即首页，一屏三件事

| 产品 | 首屏形态 | 近年关键动作 |
|---|---|---|
| ChatGPT | 问候+居中输入+4 建议芯片，全屏仅三件事；历史/Projects/GPTs 全在左栏 | GPT-5 一度**砍掉模型选择器**（自动路由，后以三档折返）；2026 桌面超级应用化（合并 Codex/Work 标签）伴随大量复杂化抱怨 |
| Claude | 同为 input-first 空态；输入框带风格预设下拉 | 色板「回到」2023 暖米白+衬线展示字——暖纸面被验证为生产力取向；桌面端 Cowork/Code 加重，但网页首屏保持克制 |
| Gemini | "Hi,"+星标微动效；**输入框去掉外框容器更轻**；建议 pills 居中排在输入框正下方 | 2025-12 改版把文件管理收进侧栏 "My Stuff"；首页专属纯黑成品牌资产；模型切换隐退 |
| Perplexity | 输入框+Focus 模式钮+Discover 内容流（密度较高） | Discover 流证明内容型首页能拉打开频次，但桌面答案是浏览器侧栏而非更重的首页 |
| Copilot | **唯一内容型 hub**：Daily/Suggested/Recent 三区块卡片流 | 2025-09 Copilot Home、2026-09 Home+Code+Autopilot——工作台化最彻底，也是用户复杂化抱怨最集中的一家 |
| Grok | 类 ChatGPT 空态；品牌=暖白/深灰双档+第三档 Darkest | 无独立桌面端；能力入口即品牌标语，不铺宫格 |

**共性五条**：①空态即首页，输入框是唯一视觉重心；②模块入口全部左移侧栏，不进中央区；③建议词芯片是标准第三层；④桌面版首屏与网页同构，差异在 OS 集成（Alt+Space/Cowork），不在首页堆功能；⑤2026「内容 hub 回摆」（Copilot/ChatGPT 超级应用）全部伴随用户对复杂化的反弹——**内容流提频次的路线与「简约大气」目标相悖，不采**。

### 2.2 效率工具：最近优先、列表优先、面板做第二导航

- **「最近打开」是所有主页的第一公民**：Raycast 空态首列、VS Code Recent（引导完成后自动翻倍扩容）、Notion Recently Visited（唯一不可隐藏的 widget）、Figma Recents。
- **列表优于卡片墙**：Notion widget 只许显隐不许排序（受限定制防仪表盘失控）；Figma/Things 干脆只放列表。卡片只用于「新建/模板」类入口。
- **主页是可指派的槽位**：Linear 可把任何收藏视图设为默认主页；Notion/VS Code 提供启动页选项——gaea 的 homeLayout 双形态（classic/tasks）正是同一思路，应保留。
- **命令面板做第二导航而非唯一入口**：⌘K 已是事实标准，但 GitHub 弃用面板遭反弹的反例证明不能只靠面板。gaea「rail+命令面板」双轨齐备，结构健康。
- **收件箱只给「需要你处理的」**：Linear 2026 Priority 分离是最新实践；状态摘要进首页时只保留待办，其余沉入收件箱/监测。

### 2.3 中文 AI 产品：输入框+场景入口趋同，「广场」已死

- 首页高度趋同为**左侧会话列表+中央大输入框+输入框周边场景入口**（豆包/Kimi/千问/元宝同构）；**DeepSeek 最简**：输入框+两个模式开关+一句话，公认最「安静」。
- **智能体广场 2026 集体退场**：豆包（800 万自建智能体）、千问、元宝几乎同日下架——广场式首页被验证为高成本低价值形态。首页场景入口应做**官方精选**，不做能力宫格。
- **Kimi 2025 去气泡**（对话即文档）反映输出侧趋势，与首页无关，不采。
- **豆包桌面版**是桌面端最佳对标：「工作模式」强调**简单清爽+大屏工作空间+最近文件续做**，桌面端低运营位、重工作流。
- **WPS 灵犀是反面教材**：办公场景功能全铺，用户反馈「功能较多需要适应」——办公 AI 的认知负担问题是真实痛点，入口越少越好。

### 2.4 设计趋势（2025~2026）

- **Calm UX / 安静界面**是 2026 主旋律（柔和色彩/克制动效/降低认知负荷）；钉钉《2025 B 端设计趋势》同款结论（极简主导、留白韵律）。
- **进化版极简+typography-first**：大字号标题承担视觉层级，「用字代替图」——gaea 玄墨衬线大标题已是此路线。
- **Bento 网格**适合多模块组织，但共识是**格子要少**（首屏 4~7 格内），格多即背离大气。
- **去复杂化潮**：Facebook 向 Instagram 收敛、Apple Photos 杂乱反弹回退、One UI 10「更干净」——头部产品共同动作是做减法。
- **"Prompt is the product"**：2025 最热 UI 就是那个等待输入的框；NN/g 建议词规范=**紧贴输入框、一键可插入、具体优于笼统、人工策展**（覆盖面广的高影响力场景，可随机序评估）。
- **用色 60-30-10**：中性底+单一强调色点睛——玄墨「金色三处点睛」纪律同构，不动。

---

## 3. 对 gaea 的十条直接启示

1. 首页只回答一个问题：「**现在开始做什么**」——开始新工作（输入）、继续旧工作（最近）、处理待办（收件箱），其余一律不进首页中央区。
2. 12 板块卡的导航职责**归还常驻左栏 rail**（rail 本就列出全部板块）＋命令面板；首页不出宫格。
3. hero 命令条**升格为页面的绝对主角**：放大居中，配 4~6 枚场景建议 chips（NN/g 规范）。
4. 建议词是模块/工具的柔性入口：「把这份预算表做成造价报告」「续写第 N 章」「问这张 CAD 图」——比 12 张卡更省空间且具体可用。
5. 最近文档+最近会话合并为**一个「继续」区**（列表行，非卡片）；理想态是跨板块续做（书/工程/文档统一），但需数据源支撑，可分期。
6. 状态四节只留「任务收件箱」摘要（需要你处理的）；内核/会话/记忆退役或收为一行文字摘要，细节归遥测轨与各板块。
7. 工具（CAD/进度，刀6 刚落）归宿=输入框「工具」菜单或建议词，而非独立卡区——OPEN_TOOL 双通道机制全部保留，只是入口呈现变（需拍板）。
8. 玄墨视觉语言、天玄主题、测试契约锁结构视觉可重画的纪律全部不动。
9. TasksFirstHome（任务优先形态）保留——「主页=可指派槽位」是市场共识。
10. 桌面端定位=**工作流首页**（豆包桌面式），不是内容门户（Copilot 式 hub 已被用户抱怨验证）。

---

## 4. 重设计候选方向（拍板池）

### 方向一「输入即首页」（最激进 · 对标 ChatGPT/DeepSeek）

```
│ (左 rail 常驻：全部板块导航)                │
│                                          │
│               晚上好                       │
│   ┌──────────────────────────────────┐   │
│   │  今天要做什么？        ⌘K ▦ 🎙 ➤  │   │
│   └──────────────────────────────────┘   │
│    [问这份合同] [续写第12章] [问这张图]     │
│    [进度体检] [画一张…] [+更多]            │
│                                          │
│   最近                                    │
│   · 施工合同审查.md           办公 · 10分钟前│
│   · 《长夜》第 12 章          小说 · 昨天   │
│   · 待处理 2 项（收件箱入口）               │
```

- 退役：12 板块卡（归 rail+面板）、工具节（归命令条工具菜单）、状态卡排（只留待办一行）、写作进度环（归小说板块）。
- 优点：最贴潮流，「大气」天花板；结构最简单，契约重写量小。
- 代价/风险：12 板块首页曝光归零——板块可发现性完全依赖 rail（rail 常驻，功能不丢）；命令条现为「发起聊天」语义，建议词直跳板块/工具需扩语义。

### 方向二「安静工作流台」（推荐 · 对标豆包桌面+VS Code Recent+Linear）

```
│  晚上好                                    │
│   ┌──────────────────────────────────┐   │
│   │  今天要做什么？        ⌘K ▦ 🎙 ➤  │   │
│   └──────────────────────────────────┘   │
│    [场景建议 chips ×4~6]                   │
│                                          │
│   继续                                    │
│   · 施工合同审查.md        办公  · 10分钟前  │
│   · 《长夜》第 12 章       小说  · 昨天      │
│   · 百锦路进度计划          进度  · 昨天      │
│   · 会话「造价对标」        聊天  · 前天      │
│                                          │
│   待处理 2 · 记忆 6 库 · 引擎 3/1    （单行摘要，点击进）
```

- 同方向一的输入区；「继续」区扩为**跨板块续做列表**（文档/会话起步，书/工程/造价分期接入）。
- 退役：12 板块卡、工具节、状态卡排（收为末尾单行摘要，全可点击进对应板块）。
- 优点：桌面工作流取向（续做优先），简约且不丢信息；保留「继续」这一效率工具验证过的第一公民；12 卡退役后板块心智由 rail+建议词双兜底。
- 代价/风险：跨板块「最近」需要统一 recent 数据源（现仅 recentFiles+ListSessions；小说书/进度工程/造价项目无统一源）——一期先文档+会话+工具产物，二期再聚齐，须接受分期。

### 方向三「精选 Bento」（最保守 · 保留卡片语言）

```
│  晚上好   ┌────────────────────┐          │
│  [输入条] │ 继续（最近对象列表）  │          │
│  [chips]  ├────────┬───────────┤          │
│           │ 收件箱  │ 工具·CAD  │          │
│           │ 2 待办 │ 工具·进度  │          │
│           └────────┴───────────┘          │
```

- hero 输入区不变；下方 4~6 格 bento：继续/收件箱/工具×2/1~2 个高频板块；其余 10 张板块卡退役。
- 优点：改动最小，卡片语言与玄墨层级无缝。
- 风险：仍是宫格心智，与「大气」有距离；「哪 4~6 个上桌」因人而异，选卡本身就是新的复杂度——**不推荐**，仅作保守备选。

---

## 5. 红线与约束（历史拍板不动摇）

- **删除史红线**：GoalCard 系（v3.6.0 撤下）勿再提；闲庭画廊版式（p-*/garden）已退役勿回；旗舰徽记/水印/设置横条特殊态已退役勿回；晨报卡已撤（组件保留，不主动回）。
- **v4.476 拍板形态**：12 卡全卡同权是拍板结果——本调研建议退役属**结构级反转，必须用户拍板**（本稿即拍板材料）。
- **刀6（2026-10-26 刚落地，未发版）**：工具节入口呈现若改（归命令条工具菜单/建议词），OPEN_TOOL 双通道机制全保留——入口位移需在拍板时一并确认。
- **TasksFirstHome 保留**；homeLayout 双形态机制不动。
- **测试契约随结构同批重写**（ModuleLauncher.test.tsx 契约锁结构；改结构=契约同批迁移）；perf-guards 源级守卫（recentFiles 订阅单源）随迁。
- **玄墨视觉语言/主题 token/金三处点睛纪律不动**——本轮只做结构减法与密度重排。
- 重叠扫描器（scripts/home-overlap-check.mjs）每断点必扫；AI 图像评审须 DOM 实测裁决。

## 6. 拍板点清单

1. 选方向：一（输入即首页）/ 二（安静工作流台，推荐）/ 三（精选 Bento）。
2. 12 板块卡是否退役（导航归 rail+命令面板）。
3. 工具节（刀6）归宿：归命令条「工具」菜单+建议词 vs 保留独立小行。
4. 状态卡排去留：全退役 vs 收为单行摘要（方向二默认）。
5. 「继续」区分期口径：一期=文档+会话（现有数据源零后端改动）；二期=书/工程/造价跨板块聚合（需新 recent 单源）。
6. 场景建议词首批内容（人工策展，4~6 条具体场景）。
7. 命令条语义是否扩展为全局动作（建议词带板块/工具直达目标）。

拍板后路径：原型（design/ 目录惯例）→ 契约先行（测试结构锁重写）→ CSS/JSX 重画（玄墨语言不动）→ 重叠扫描四档全零 → 全量门禁。

## 8. 拍板与落地记录（2026-10-26 同日）

**拍板**：用户选**方向一「输入即首页」**。拍板点随向裁定：12 板块卡退役（导航归 rail+命令面板）、工具节归宿=命令条「工具」下拉、状态排收为待办一行、「继续」一期=文档+会话（现有数据源零后端）、建议词人工策展 4 条、命令条语义不动（仍走语音助手文本通道 VoiceChatText，建议词只填入不代发）、TasksFirstHome 保留。

| 项 | 落点 |
|---|---|
| 中轴三段 | `ModuleLauncher.tsx` DeskHome 重写：`.w-zen`（880px 居中栏）= ①`.w-greet`（印章「文」+时辰问候大标〔5/12/18 点三档+夜深档〕+home.sub 题辞+56px 大命令条+建议词 2×2）②`.w-recent`（最近文档行 ≤4〔6→4〕+最近会话行〔SessionList 行改真按钮接 onOpen——旧列表态该参数本是死参〕）③`.w-todo`（待办一行：图标+等宽计数+打开胶囊） |
| 工具入口随迁 | 刀6 home-tools 独立节 → 命令条 `is-tools` 下拉（antd Dropdown v5 `menu={{items}}`，trigger click）；`openOfficeTool` 双通道（OPEN_TOOL 事件+OPEN_TOOL_FLAG 标记+导航 gaea）逐字保留；home-tool-cad/schedule testid 随迁到菜单项 |
| 退役面 | 12 板块卡+ModuleCard、home-boards/home-tools 节、w-stat-row 四节状态带、w-aside 写作进度环、hero pill；`launcher_parts` 瘦身：遥测 3s 轮询/记忆脉搏/TelemetryBody/KernelRow/WritingRing 全退役（LauncherData 收窄为 recentFiles+sessions）；`module-launcher.css` 1018→~560 行（.w-mod/.w-stat-row/.w-aside/.w-progress/.garden-chip/.ml-ring 等族删除；tasks 形态 .w-board/.w-hero/.ml-card/.t-* 结构样式全保留） |
| 词典 | 三语 +10 键（greetMorning/Afternoon/Evening/Night、recentTitle、sug1~4）+ placeholder 改「问点什么，或从下面选一个开始…」口径；退役键（capTitle/pill 等）按 statIdle 先例闲置保留防 parity 抖动 |
| 契约 | `ModuleLauncher.test.tsx` 重写：三段结构+**退役面零残留断言**（home-boards/home-tools/.w-mod/.w-stat-row/.w-aside/.w-progress/pill 全 null）+建议词填入不代发+工具下拉双通道（menu portal 取 label span testid）+最近行真打开+待办行 aria=任务；收件箱三例与形态分支/语音降级用例口径不变随迁 |

**验证**：`tsc -b` 绿；eslint 0 error（launcher_parts react-refresh 警示系拆分前既有）；定向 36/36；**全量 vitest 4015/4015（458 文件）**；knip exit 0（唯一 unused export `resetPaneEnabledForTest` 属 CAD 线既有非本刀面）；无头 Edge（独立 user-data-dir+CDP 9344 避开用户壳 9333）四档视口 1600×1000/1440×900/1280×800/920×760 **重叠扫描全零 over=0**；DOM 实测（裁决走实测非图像评审）：问候=晚上好/衬线 30px（≤1060 高度档生效）、命令条 760×50 居中、建议词 2×2（首版 3+1 尾行孤悬→flex 收窄仍 3+1〔芯片实测 241/128/178px 前三恰好塞下〕→改 `grid repeat(2,auto)` 确定性 2×2）、满态 4 文档行渲染。

**留池**：①「继续」二期跨板块聚合（书/进度工程/造价项目需统一 recent 源）；②亮色档真机目检+用户真机走查；③tasks 形态与 classic 的视觉语言已分家（卡片台 vs 中轴），是否统一待用户体感；④发版推送待指示。

### 8.1 「继续」二期：跨板块工作对象聚合（2026-10-26 同日）

一期留池①当日清偿。范围拍板：**只收小说书+进度工程两类**——cost 判「不做」（其「选中项目」是 WorkcostBillView 组件本地筛选态，无恢复语义，硬造对象行是伪续做）。

| 项 | 落点 |
|---|---|
| recentObjects 单源 | `gaea/lib/recentObjects.ts`（仿 recentFiles：localStorage+写入广播+解析缓存；kind+key 去重置顶；每 kind 留 3 条、总量 12 防单一板块刷屏；读写 fail-safe 不影响打开主流程）+ 单源测试 4 例 |
| 打开链路共享件 | `components/novel/openNovelProject.ts` 抽自 HomePage.handleOpen（后端 OpenProject→store 落位→可选直达阅读 tab→记账）；HomePage 四处接线（handleOpen 改走共享件零编辑复绿；新建/文件导入/在线导入三处补一行记账）——书架与首页一份真相 |
| schedule 埋点 | `schedule/store.ts` switchProject/createThenSwitch 成功路径 record（失败 fail-closed 不记，防没打开的工程顶到续做行） |
| 首页对象组 | 「继续」区顶部 `.w-objects`（≤3 行：BookOutlined/BarChartOutlined+名称+板块标签+箭头），无记账整组不渲染；恢复链路：novel=切书闸门 guardNovelSwitch+共享件+navigate('novel')；schedule=`useScheduleStore.getState().openProject(rel)`（指针驱动，ScheduleTool/完整工作台自动跟随）→ OPEN_TOOL 双通道开工具面板——**store 走动态 import，schedule 域不进主包** |
| 词典 | 三语 +2 键（home.objKindNovel 小说/Novel/小說；home.objKindSchedule 进度/Schedule/進度） |
| 契约 | ModuleLauncher.test +3 mock（openNovelProject/guardNovelSwitch/schedule store——动态 import 同走 mock）+2 用例（对象组渲染+双恢复链路断言；空态整组不渲染）；recentObjects.test 4 例 |

**验证**：tsc 绿；定向 73/73（HomePage 11 例零编辑=抽取等价）；无头 Edge 暗色四档+亮色档（`gaea-display-mode` localStorage 配方）重叠扫描**全零 over=0**；对象组亮/暗两档渲染实测（objRows=2/docRows=2，亮色主文字 rgb(15,26,32) 深墨可读）。全量 vitest 见提交记录。

**沙箱走查（2026-10-26 深夜补，真实 Wails 绑定端到端）**：`build.bat skip-smoke` 构建 + `GAEA_WALKTHROUGH=1 GAEA_WEBVIEW_DEBUG=1`（沙箱隔离三行日志核验：书架/workspace/APPDATA 均指 Temp 走查沙箱）。两条恢复链路全过：①novel=CDP 真调 `CreateProject` 建书 → 种子 → 点首页续做行 → `OpenProject` 真调成功（记账 ts 回写为证）→ 落小说板块；②schedule=真调 `GaeaScheduleProjectCreate` 建工程 → 点行 → 指针真切+记账 → 办公 ToolSheet 打开且工具面板显示走查工程名。console 0 错。**断言辨误**：点行后 `.ml` 仍在 DOM——首页是 keepAlive 常驻隐藏（visitedPages）不卸载，判「已切走」须查可见性（offsetWidth=0）非存在性。

**走查揭出隔离洞并当场根修（v4.428.1 同类）**：上述 schedule 链路当时实际把 `走查工程` 写进了**真实仓库**（`.gaea/schedule/index.json` 指针被改+`进度计划/` 新增文件）——污染已即时复原（checkout 索引+删文件，当前计划本体未动）。根因=**未登录走查壳 `ga.cfg==nil` → `gaeaCwd()` 回退 `os.Getwd()`=仓库根**，schedule/CAD/dag/crosslink 等办公面绑定全部静默落真实目录（昨日会话登录过沙箱故未漏）。根修=`gaeaCwd()` 走查态 fail-closed 到 `WalkthroughWorkspaceDir()`（同 userHome 纪律：宁可落沙箱绝不回落）+ 回归测试两例；重建后**同事故形态复现验证**：复现工程落 Temp 沙箱、真实索引零触碰。另：9333 被昨日会话遗留的无窗口 msedge+msedgewebview2 双占（dev-mock/原型页签=自动化残留特征），取证（StartTime 前日+MainWindowTitle 空）后清除再启壳——CDP 驱动前必查 9333 属主配方再次应验。

**二期后留池**：①tasks 形态是否补对象组（当前 classic 独有）；②发版推送待指示。（真机实弹走查已于沙箱真绑定链路代偿：见上「沙箱走查」段。）

### 8.2 一级导航合一：rail 不再按空间分页（2026-10-26 用户拍板）

用户指正「一级导航还在分页，没有合并显示全部一级标题」——rail 此前按 书斋/闲庭 双空间分域（v4.475 首页合一时保留的口径，work 侧竟看不到 聊天/小说/绘梦/角色库/原罪）。合一落地：

| 项 | 落点 |
|---|---|
| rail 全量 | `CommandRail` 数据源 `getActiveMenuBoardsForSpace(space)` → `getActiveMenuBoards()`：首页+全部 inMenu 板块同栏（12 项，menuOrder 序）；空间指示徽标（v3-rail-space 竖排微章）随分域退役 |
| 随迁退役 | 每空间最后页面持久化（SHELL_PAGE_PREFIX 三件+save effect）、keepAlive 空间剪枝（pruneVisitedForSpace——访问过的板块全部常驻，跨板块往返不丢现场）、NAVIGATE 空间可达性门禁+v4.51 crossSpace 双订阅（统一走 navigateBoard 自动换空间）、CommandRail 的 space/onSwitchSpace props、Ctrl+1~9 改全量序 |
| 数据面保留 | 壳层空间降为纯数据概念：会话空间/事件分域（subscribeForSpace）/任务标签仍用 ShellSpace+boardSpace；navigateBoard 仍按目标板块归属隐式切空间 |
| 死码清理 | `getActiveMenuBoardsForSpace`/`filterBoardsForSpace`/`isBoardReachableInSpace`/`pruneVisitedForSpace` 退役；space.test 归属表改 boardSpace 纯归属口径（shared=home/模型中心/设置/运行监测）；CSS `.v3-rail-space` 族删除 |
| 契约 | CommandRail.test 重写（全量 12 项同栏+徽标零残留+跨域点击直达+激活态）；manifests.test getActiveMenuBoards 全量序锁 |

**验证**：tsc 绿/eslint 0err/knip exit0/**全量 vitest 4016/4016**；无头 Edge 实测：rail 12 项同栏无溢出、指示徽标零残留、work 态点「小说」直达（novel-hub 挂载+激活态跟随）、回首页三段结构完好。

### 8.3 DeepSeek 化：中轴两段+板块胶囊行（2026-10-27 用户拍板）

用户携 DeepSeek 网页版截图再拍板「要简洁，删除最近文档、待处理项，删除任务优先版面，模仿 DeepSeek 的网页版，把各个功能板块置于对话框下端」。方向一三段（开始/继续/待办）收为中轴两段：

| 项 | 落点 |
|---|---|
| 形态 | `ModuleLauncher.tsx` DeskHome 重写：①`.w-greet` 开始段=印章+时辰问候+题辞+大命令条（语音/工具/发送全保留）+**板块胶囊行**（`.w-boards`，命令条正下方=DeepSeek 落地页功能入口位；`deriveLauncherModules` 全量 12 模块=方向一退役的 12 板块卡以胶囊形回归，点击直达，manifest 订阅实时更新）②`.w-recent` 继续段=工作对象组（书/工程）+最近会话行——**对象组与会话皆空时整段不渲染**（冷启首页=纯落地页，工作有现场才长出「最近」） |
| 用户点名删除 | 最近文档行（`desk-recent-docs` 块+空态 CTA；办公板块自有最近面承接）、待办一行（`.w-todo`/`desk-task-inbox`）、任务优先版面（**TasksFirstHome.tsx+测试整删**、homeLayout 分发/`setHomeLayout`/`HomeLayoutPanel` 设置面板/appStore 三件全撤） |
| 随形态退役 | 建议词 chips（DeepSeek 形态输入框下只有功能胶囊；`home.sug1~4` 键按 statIdle 先例闲置）、顶栏仪表条 HomeStrip（形态切换既亡，孤 date 无撑——`.ml-strip`/`.ml-space-btn` 族删）、placeholder 改「问点什么，或从下面进入一个工作台…」口径 |
| 收件箱入口归位 | 待办行+任务优先形态退役后面板几成孤岛（Ctrl+K/语音/微信只「存」不「看」）——**命令面板空态新增「打开任务收件箱」**：`OPEN_TASK_INBOX` 事件（`events.ts`）→ 首页 ModuleLauncher 单例面板；首页 keepAlive 常驻（visitedPages 自 boot 含 home）恒有听者，无需 OPEN_TOOL 式 sessionStorage 补开标记 |
| launcher_parts 瘦身 | `TaskInboxEntry`（待办行挂点内容）+recentFiles 订阅源退役——LauncherData 收窄为 `{sessions}` 单源；CSS 809→~470 行（.ml-strip/.ml-space-btn/.ml-card 族/.w-board/.w-hero 卡/.t-*/.w-chips/.w-empty/.w-todo/.ml-krow-strong/.ml-memory/.ml-inline-cta 全删，任务形态卡片语言随形态整退） |
| 词典 | 三语 +2 键（home.boardsAria 功能板块、shell.search.openInbox 打开任务收件箱）+placeholder 三语改口径；tasksFirst*/homeLayout*/sug1~4/recentDocs* 等退役键按 statIdle 先例闲置保留防 parity 抖动 |
| 契约 | ModuleLauncher.test 重写：两段结构+胶囊行全量 12+直达导航+退役面零残留（desk-recent-docs/desk-task-inbox/w-chip/tasks-first-home/home-layout-* 全 null）+对象组双恢复链路口径不变+收件箱改事件入口（挂载零拉取、OPEN_TASK_INBOX 开面板）；SearchModal.test +空态入口例；appStore.test homeLayout describe 删 |

**验证**：tsc -b 绿；eslint 0 error（launcher_parts react-refresh 警示系拆分前既有）；knip exit 0（唯一 unused export 仍为 CAD 线既有）；**全量 vitest 4004/4004（458 文件，含两枚源级守卫随迁：events.test 事件数 6→7、perf-guards 最近文件守卫改防回潮口径）**；无头 Edge（vite dev + 独立 user-data-dir，1440×900）截图+DOM 实测：问候居中、命令条 760×50、胶囊行 12 枚全渲染（chat/novel/imagegen/gaea/cost/memoryhub/modelcenter/characterlib/weixin/sin/runtime/settings——识图初审漏数 runtime，DOM dump-dom 实证 12/12）两行排布无溢出无重叠、退役面（最近文档/待办/建议词/仪表条）零残留、左侧 rail 完好。

**留池**：①亮/暗两档真机目检+用户真机走查；②发版推送待指示（与 §8/8.1/8.2 四提交同批）。

### 8.4 身份位换 gaea 字标 + 命令条大气化（2026-10-27 用户复检拍板）

§8.3 落地后用户复检两刀：「印章『文』+ 时辰问候删除改成其它的」「输入框不够大气」；身份位续拍「不要用文书台，用 gaea 软件的名字」。

| 项 | 落点 |
|---|---|
| 身份位 | 印章 `.w-seal`（印泥朱 #d06055=CSS 末枚 hex 豁免，随印销号）+ 时辰问候（greetKeyOf 三段档）退役——**gaea 软件名字标**承接（`home.brand` 三语同值，与 rail/favicon 同一拼写；44px 衬线+0.06em 字距，静态无时间态=DeepSeek 品牌名同构）；greet* 四键与 home.title 按先例闲置留典 |
| 命令条大气化 | `.w-cmd` 56→**72px** 高、max-width 760→**880**（=中轴柱通栏）、圆角 12→**18px**、输入 15→**16.5px**、按钮 38→46px/13.5px、orb 22→24px、kbd 升档；`.w-boards` 胶囊行随命令条同步 880 对齐；高度档 ≤1060 72→62/字标 38；宽度档 900/640 随迁 |
| 随迁 | appStore.contrast.test 印章豁免判据注释销号留档（hex 已零存在）；ModuleLauncher.test 开始段改锁 gaea 字标+印章/四问候零残留 |

**验证**：tsc -b 绿；定向 46/46（ModuleLauncher 11+contrast 24+…）；无头 Edge（vite dev+独立 user-data-dir，1440×900）截图+DOM 实测：h1=gaea、印章/问候/仪表条零残留、命令条 72px 通栏成视觉主角、12 胶囊 8+4 无溢出无重叠。

**留池**：①亮/暗两档真机目检+用户真机走查；②发版推送待指示（与 §8~8.3 五提交同批）。

### 8.5 命令条多行化（2026-10-27 用户三检拍板）

§8.4 放大到 72px 后用户三检「输入框还是小，是因为是单行吗？改成多行，加高」——诊断正确：单行横排 72px 天花板就在那里。落点：

| 项 | 落点 |
|---|---|
| 多行主体 | `Input` → `Input.TextArea`（borderless，autoSize 2 行起/10 行封顶）——容器改纵向 flex：textarea 占满剩余高 + **框内底部动作行** `.w-cmd-row`（orb 存在位+spacer+工具/语音+⌘K+圆形发送 44px）；min-height 132px 托底（≤1060 高度档 116 / ≤640 档 108），长文随内容撑高 |
| 键盘语义 | **Enter=发送**（走语音文本通道 VoiceChatText，发送后清空）、**Shift+Enter=换行**（onPressEnter 判 shiftKey 放行默认行为）——DeepSeek/ChatGPT 同构 |
| 动作钮 | 动作行内回归 40px/13px（框内嵌钮不做 46 全尺寸防行高失控）；发送钮改 **44px 正圆** |
| 契约 | ModuleLauncher.test +1 例：textarea 主体+框内动作行结构锁、Shift+Enter 不发送、Enter 发送且清空 |

**验证**：tsc -b 绿；定向 12/12；无头 Edge（1440×900）截图+DOM 实测：输入区 ~130-140px 高、上文本下动作行结构成立、880 通栏为页面视觉主角（较 DeepSeek 官网 56-64px 单行更高）、placeholder 左上角起排（多行典型式）；胶囊行/整体无溢出无重叠。

**留池**：①亮/暗两档真机目检+用户真机走查；②发版推送待指示（与 §8~8.4 六提交同批）。

## 7. 参考来源（节选）

**国际 AI**
- ChatGPT 空态对比（2025-06）：intuitionlabs.ai/articles/conversational-ai-ui-comparison-2025
- GPT-5 模型选择器：techcrunch.com/2025/08/12/chatgpts-model-picker-is-back-and-its-complicated
- Claude 色板回归暖米白：medium.com/design-bootcamp/claudes-ui-came-full-circle…
- Gemini 2025-12 改版：9to5google.com/2025/12/03/gemini-web-redesign-dec-2025
- Copilot Home（Daily/Suggested/Recent）：support.microsoft.com/en-us/copilot/try-out-copilot-home-79451b65…
- 2026-09 Copilot 新设计：blogs.microsoft.com/blog/2026/09/25/introducing-the-new-copilot-with-home-code-and-autopilot
- ChatGPT 桌面超级应用争议：news.qq.com/rain/a/20260730A05AQY00

**效率工具**
- Raycast Root Search：manual.raycast.com/search-bar
- Notion 侧栏/Home：notion.com/help/navigate-with-the-sidebar；thomasjfrank.com/notion-home-everything-you-need-to-know
- Linear changelog（Priority/默认主页）：linear.app/changelog
- VS Code Welcome 渐进退场：code.visualstudio.com/updates/v1_57
- GitHub 命令面板弃用反弹：neowin.net/news/github-customers-rejoice-as-command-palette-deprecation-is-delayed
- 命令面板趋势：ux.stackexchange.com/questions/153299…；21st.dev/blog/react-command-palette-components

**中文 AI**
- 豆包新版网页版：developer.volcengine.com/articles/7473796661194522650；桌面版 doubao.com/download/desktop
- 智能体广场集体下架（豆包/千问/元宝）：yage.ai/share/uga-sunset-20260704.html
- Kimi 去气泡改版：finance.sina.com.cn/tech/shenji/2025-05-13/doc-inewkrtz4521823.shtml
- WPS 灵犀：news.cn/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html；评测 bitai.com/2025/07/315752.html

**趋势**
- Prompt is the product：uxdesign.cc/prompt-is-the-product-74b7acde763d
- NN/g 建议词规范：nngroup.com/articles/designing-use-case-prompt-suggestions
- NN/g 空态/聊天机器人规范：nngroup.com/articles/ai-chatbots-design-guidelines
- Calm UX / 2026 趋势：elements.envato.com/learn/ux-ui-design-trends；优设·钉钉 2025 B 端趋势 uisdc.com
- 60-30-10：nngroup.com/articles/color-enhance-design；updivision.com/blog/post/ui-color-trends-to-watch-in-2026
- Apple Photos 回退：pcmag.com（iOS 19 简化回退报道）；One UI 10：sammobile.com
