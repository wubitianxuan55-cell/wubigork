# gaea UI 市场对标与设计方向（2026-10）

> 状态：调研收官，方向提案待拍板。信源：四路网络调研（每条关键结论附 URL，区分官方文档/二手评测）+ gaea 前端只读摸底（路径级证据）。
> 红线继承：简化≠删功能；已退役方案（闲庭画廊版式/旗舰卡/首页晨报/GoalCard 系）不再提案；涉功能面新增一律进拍板池。

---

## 一、调研范围与方法

| 路 | 对象 | 回答的问题 |
|---|---|---|
| A. AI 办公/助手桌面端 | M365 Copilot、Notion Agent、WPS 灵犀、ChatGPT Desktop、Claude Desktop、Raycast、飞书/钉钉 | 聊天×文档共生范式、Agent 交互模式、记忆呈现 |
| B. AI 小说写作 | Novelcrafter、Sudowrite、NovelAI、Scrivener、Dabble、彩云小梦、笔灵、墨狐、阅文作家助手、蛙蛙写作 | 长篇结构管理、故事圣经、AI 辅助呈现、生成透明度 |
| C. 造价/进度专业软件 | 广联达 GCCP6.0、品茗、神机、未来、MS Project 2024、斑马进度、Linear（密度参照） | AEC 用户惯例、专业密度参数、品类 UX 弱点 |
| D. AI 图像 + 2026 趋势 | Midjourney、即梦、LibLib、Krea、Leonardo、ComfyUI 新前端；agentic UX/quiet software/令牌/排版趋势 | 生成节奏、资产管理、2026 标配与过气清单 |
| E. 自家摸底 | frontend/ 全板块 | 外壳结构、令牌真源、跨板块不一致清单 |

---

## 二、市场发现摘要

### 2.1 AI 办公/助手：容器在变薄，输出物在变实

**「聊天×文档共生」五范式**：

1. **停靠侧栏伴生型**（Word/Excel Copilot、WPS 灵犀、飞书多维表格）——文档为主、右侧固定 AI 窗格，改动直接落文档并适配格式；
2. **并排双栏画布型**（Office in Copilot 2026-10、Claude Artifacts、旧 ChatGPT canvas）——聊天与文档平权两栏，可全屏专注编辑；
3. **聊天内嵌块型**（ChatGPT writing blocks——canvas 侧栏已于 2026-05 移除、Notion 聊天内交互表格）——输出物是线程内可编辑块；
4. **端到端委派型**（Copilot Cowork、Notion Custom Agents、Raycast Automations）——后台分钟级执行，返回完整文档+活动日志；
5. **Spotlight 即时型**（Raycast Quick AI、钉钉魔法棒）——全局热键小窗。

2026 年两次方向相反的摆动（OpenAI 退回内嵌块、微软走向并排画布→委派），共同点：**容器在变薄，输出物在变实**（真实文件、可下载、带版本）。
来源：https://www.microsoft.com/en-us/copilot/blog/2026/05/28/introducing-a-new-design-for-microsoft-365-copilot ；https://techcommunity.microsoft.com/blog/microsoft-365-copilot-and-ai-blog/introducing-office-in-the-new-microsoft-365-copilot-app/4561230 ；https://felloai.com/chatgpt-canvas

**Agent 交互 2026 通用模式**（Excel Edit/Plan/Chat 三态、Raycast v2.3、WPS 拆解需求）：

- 模式三态：直接改 / 先出计划供审核再执行 / 只读问答；
- 实时推理展示 + 活改动 + Stop 随时中断（Excel "watch reasoning in the pane, see changes live"）；
- 分级审批：默认宽松、敏感操作先预览再应用（微软对共享/敏感文件）；
- 决策点暂停提问（Raycast 可折叠问题卡）；
- 活动日志 "visible and reversible"（Notion）；
- 错误三段式：发生了什么/为什么/下一步试什么（A/B 持续提升完成率，https://zylos.ai/research/2026-05-28-agentic-ux-frontend-design-patterns-ai-agents ）。
来源：https://support.microsoft.com/en-us/excel/copilot/frequently-asked-questions-about-copilot-in-excel ；https://www.raycast.com/changelog

**记忆/知识最佳实践**：

- 「它记得什么」可问可看（微软 "What do you know about me?"、Claude "view my current memory"）；
- 结构化管理：Claude 按 Topics 分组逐条编辑/删除；微软四源分层（显式保存/历史推断/自定义指令/工作洞察）；
- 逃生舱：微软 Temporary chat 一键不读不存；删除保留期明示；
- 记忆即文档：Notion 指令页=用户可直接编辑的持久记忆；
- 引用可信度可视化：Notion 已验证页带对勾、回答附可点击来源。
来源：https://support.microsoft.com/en-us/microsoft-365-copilot/personalize-what-microsoft-365-copilot-remembers ；https://support.claude.com/en/articles/11817273 ；https://www.notion.com/help/notion-agent

### 2.2 AI 小说写作：结构双向联动 + 生成透明度是分水岭

**2026 品类标配**：三栏骨架；幕/章/场景三级+卡片大纲且**结构视图与手稿双向联动**（Dabble Plot Grid 拖卡即改手稿）；故事圣经条目化+别名管理+自动/关键词注入（Novelcrafter Codex / Sudowrite Story Bible / NovelAI Lorebook，中外全部趋同）；划词浮动工具条；多候选生成+重掷；聊天侧栏；自动保存+版本恢复；暗色模式。

**Top5 亮点**：

1. NovelAI 生成透明三件套：Retry 候选分支列 + Token 概率表 + Context Viewer + AI/作者文本颜色编码（https://docs.novelai.net/text/editor.html ）；
2. Novelcrafter Codex 的 Progressions（设定随剧情演变）+ 别名自动追踪 + Mention Timeline（https://www.novelcrafter.com/features/codex ）；
3. Sudowrite 划词自适应工具条（单词→同义词库、多词→Rewrite/Expand）+ History 卡 "Looked at:" 上下文 chiclets（https://docs.sudowrite.com/... /selection-menu ）；
4. 阅文作家助手生态闭环（码字+书评月票+读者画像与稿酬数据同台）（https://www.yuewen.com/app/?type=appzj ）；
5. 彩云小梦三候选平行世界续写 + 世界广场（https://apps.apple.com/cn/app/彩云小梦/id1564619616 ）。

**反面教训**：Novelcrafter「得成为 AI 大师才能开始」劝退、Sudowrite Story Bible 强流程依赖——AI 能力要开箱即用，结构化设定渐进式引导而非前置必填。

### 2.3 造价/进度专业软件：惯例铁律 + 弹窗过载是机会

**AEC 用户惯例清单**（对标 gaea 造价/进度）：

- 「左树+中表+属性面板」三件套（GCCP6.0 官方 8 部分布局，属性默认在**下方**）；Ribbon/Office 心智；
- 清单录入三分法：编码直输（简写 1-1-1-1 自动带出）+ 名称列实时联想 + 查询库「插入/替换」双写回（https://aecore.glodon.com/doc/GCCP6/12c4f90d4ddc41ddb068c51d972e6c11 ）；
- 右键菜单功能族密度极高（临时删除/标记颜色/过滤/批注/下拉填充/局部汇总/复用组价…）；
- 表格↔横道图↔双代号网络图**三向实时联动**（斑马「一表双图」灵魂，https://www.zpert.com ）；
- 关键路径=红色是硬惯例（MS Project/ProjectLibre/国标双代号一致）；
- 双击偏差项**定位回源**+重新检查闭环（GCCP 云检查）；状态栏回显选中上下文；Excel 互操作刚需；批量操作决定效率口碑（未来软件跨单位工程批量换算）。

**品类弱点=机会点**：弹窗过载（查询/补充/检查全弹窗）、属性面板在下方浪费宽屏、拖拽暗加约束不透明（MS Project 拖甘特条暗设 FNLT，用户「拖不动条」困惑）、计划/实际对比原始（甘特条不动、手动加网格线）、无键盘优先/命令面板文化、AEC 工具普遍现代美学赤字（Procore 被评 old-school，用户确实会为界面现代性迁移）。

**「现代美学×专业密度」标杆=Linear**：降噪与增密并行——对齐优先于装饰、密度 A/B 试错、颜色系统收敛为三变量减少 chrome 色；键盘优先+命令面板全程不碰鼠标（https://linear.app/now/how-we-redesigned-the-linear-ui ）。密集 UI 惯例参数：**正文 ≥12–14px、行高 1.125–1.2、表格 condensed 字体**（https://news.ycombinator.com/item?id=40428386 ）。

### 2.4 AI 图像工具 + 2026 趋势

**图像工具共性**：左窄栏+中央画廊/画布+右参数面板三区；参数用滑杆+推荐值而非裸输入（Leonardo 0.6/0.4 建议值）；**图片即会话**（点开即回填提示词/参数/参考图+一键复用，Midjourney 元数据随图走）；版本树以网格变体呈现（vary/upscale/pan/remix 从成品分叉）；蒙版笔刷与自然语言指令编辑**双轨并存**（即梦 SeedEdit、Krea 3 区域编辑）；专业复杂度「应用化」分层露出（ComfyUI Nodes 2.0/App Mode 双形态）。

**Top 亮点=ComfyUI Run Bar**：队列节奏三态（手动 Run / On Change / Instant）+ Shift 插队 + 可拖动停靠 + Active jobs 徽标——把「生成节奏」做成显式控件（https://docs.comfy.org/interface/overview ）。

**2026 趋势三档**：

- **已成标配**：流式+渐进披露三级（步骤→详情→原始数据）；暗色+三层设计令牌（primitives→semantic→component，语义层承载主题切换）；Bento 卡片为默认布局语法；diff/前后对比；可变字体（中文思源可变版单文件全字重）；
- **正在兴起**：Agent 步骤台账/审批门/后台胶囊标准化（**运行中 vs 等你输入 两态必须区分**，后者琥珀色+问号）；计划-执行-检查点六阶段；quiet software 定位语；密度回归与自适应密度；衬线回归；中文风格化开源字体（得意黑/霞鹜文楷）进标题场景；
- **已过气**：纯装饰玻璃拟态（退守为功能性 backdrop-filter 浮层）；「空洞极简」；**聊天框作为唯一 AI 界面**（长任务需画布/工作流视图）；无状态裸 spinner（"Agent is working..."）。
来源：https://zylos.ai/research/2026-05-28-agentic-ux-frontend-design-patterns-ai-agents ；https://www.setproduct.com/blog/ai-agent-ui-design-patterns ；https://www.digitalapplied.com/blog/design-systems-2026-scale-ui-without-chaos-methodology

---

## 三、gaea 现状对照

### 3.1 已领先市场的设计资产（不动的根基）

| 资产 | 市场对标 | 结论 |
|---|---|---|
| 玄墨 v11 视觉语言 + 7 预设×明暗 14 色板 + 语义令牌（appStore getThemeTokens，WCAG AA 机检） | 2026 三层令牌标配 | **令牌层已达市场标配且工程质量高**（摸底确认：颜色/圆角/明暗高度统一） |
| 上下文预算可视化（ContextWindowRing + 压缩水位四档状态梯，v4.485/v4.516） | NovelAI Context Viewer / Sudowrite chiclets | 理念同源且**水位档位呈现比多数竞品细** |
| 工具卡输出治理（排版单源/diff ±视图/组行截断/输出降噪，v4.503-507） | 2026 渐进披露三级 | 已到中间档（步骤→详情），缺第三级原始数据入口属可选 |
| 流式滚动手势化/回退菜单键盘可达/剪贴板诚实反馈（v4.510-516） | 各竞品零星 | **细节体验领先**（Claude/ChatGPT 桌面端均无手势化滚动跟随） |
| 多板块桌面壳（rail+38px strip+keepAlive 常驻+空间隐式切换） | 竞品均为单工作台 | 独有形态，是产品身份，不再动结构 |
| 安全纪律（写入确认红线/记忆建议制） | 微软 Temporary chat 同款思想 | 对齐最佳实践 |

### 3.2 系统性缺口（摸底路径级证据）

**缺口一：板块外壳四类不一致**（最大内部债）

- **页面头部 10 板块 10 种**：38px strip（foundation.css:243）之下各页自建第二条头——模型中心 34px（modelcenter.css:50）/设置 40px（settings-page.css:31）/造价 h-12=48px（CostLibraryPage.tsx:155）/小说 52px（novel-workspace.css:35）/画室 ≈54px（imagegen.css:25）/gaea·聊天·进度·原罪无定高；结构范式三种（图标章+标题+pill / 无标题只留状态 / 头部完全上交 strip）。
- **工具条按钮三套写法**：gaea 域 ToolbarButton（13px lucide）/聊天·绘梦·进度·监测 antd Button size=small 内联 /造价·办公 Tailwind 类。
- **空态至少 5 种形态**：统一 V3Empty 覆盖约 13 处但造价空库是自定义三步卡、读取失败另一套、监测纯文本、sin 残留 antd Empty 直用（SinCastPicker.tsx:8）。
- **字号两套档位定义**（index.css base 14 vs gaea/tailwind.css base 13）+ 微差泛滥（造价页内九档 10.5~34px；chat-board.css 七档）——首页 v11 复盘明说「层次靠 13↔14px 微差」是廉价根因，但其它板块仍是这个写法。
- **间距无全局令牌**：各板块 Tailwind 任意值/自写 CSS 混用。
- **节标三种语言**：首页 ml-eyebrow（11px/700/0.26em 渐隐尾线）是自觉设计；gaea 用 v3-panel-title；造价/监测只是 10.5px 灰字。
- rt-header 硬编码 radius 10px 脱离 8/12/16/28 档位（runtime.css:24）；边框语言三分（v3-panel 1px 实线 / 首页 4.5% 近无 / 造价 border-soft 各档透明度）。

**缺口二：专业模块未达 AEC 惯例密度**（对标 2.3）

造价：定额查询/挂接走弹窗（品类通病同款）、无右键菜单族、无编码直输+名称联想三分法录入、批量操作入口弱；进度：表格↔三图已有联动（v4.139+），缺键盘流（回车带出/下拉填充）与命令面板承载、拖拽副作用行内提示。

**缺口三：AI-native 增量模式**（对标 2.1/2.4）

- 后台任务**两态胶囊**（运行中 vs 等你输入，琥珀+问号）未统一——gaea 有 Notice/唤醒（Z6）与各板块任务中心，但无全局常驻胶囊；
- **引用 chips 回链**（回答中的文档/知识引用可点击跳原文位置）缺位；
- 记忆中枢有库管理+建议制，但缺 Claude Topics 式「它记得我什么」一页总览（只读展示，不动建议制红线）。

**缺口四：模块专项对标差**（对标 2.2/2.4）

小说：无角色别名自动检测/提及跳转（活 Codex）、无 AI 文本颜色编码、生成卡无「本次上下文构成」chiclets；画室：队列无节奏三态（Run/OnChange/Instant）、版本对比无网格分叉操作。

---

## 四、设计方向提案（拍板池）

> 六个方向按杠杆排序；D1 是纯外壳统一（无功能删除、测试契约锁结构视觉可重画范式保证零测试编辑）；D2~D5 含功能面新增须拍板；建议按 阶段一基建→阶段二专业→阶段三增量 推进。

### D1 板块外壳统一（Board Shell Kit）——P0，杠杆最大

**目标**：消灭「10 板块 10 种头部/三套工具条/5 种空态/两套字号档」，让玄墨语言从首页推广到全站（接续玄墨推广线程的收尾刀）。

**套件内容**：

1. **PageHeader 统一组件**：两档高度（compact 40px / 标准 48px），三段式（左=图标章+工作区名+上下文 pill；中=子导航槽；右=动作区）；「板块名已在 strip 面包屑」原则下标题降级为工作区名（模型中心/画室现行做法吸收为规范）；full 布局三板块（gaea/chat/sin）维持上交 strip 模式不动。
2. **ToolbarButton 单源收编**：三套写法归一（保留 gaea 域 13px lucide 规范为基准），进度/监测/聊天/造价全部改接。
3. **字号档收敛**：全局 UI base 定 13px（桌面密度基准），阅读区/正文 14px+，档位锁 11/12/13/14/16/20 六档，禁 10.5/11.5/12.5 微差档（首页 v11 纪律推广）；数字值一律 mono+tabular-nums（玄墨第三点睛位全站化）。
4. **间距令牌**：--space-1..8（4/8 基数，对齐 v11 栅格 8+4），先立档再逐板块收敛。
5. **节标单源**：SectionLabel（eyebrow 款）全站推广，造价/监测的灰字小标改接。
6. **空态三档**：首次引导 / 无数据 / 失败重试——V3Empty 扩三档变体全覆盖，退役各板块自写空态（简化≠删功能：引导内容保留）。
7. **radius/边框归档**：rt-header 10px 等游离值归 8/12/16/28 档；边框语言按板块属性二分（数据面板=1px 实线 v3-panel；画廊/卡片面=描边退场近无边线）。

**范围**：首批换壳=运行监测+模型中心+设置（最小三块）；造价/画室/小说随各自专项刀走。
**依据**：Linear 对齐纪律（2.3）；2026 令牌标配（2.4）；摸底缺口一（3.2）。

### D2 专业密度与键盘流（造价+进度）——P1，须拍板

1. 造价录入三分法：编码直输（带简写）+名称列实时联想+**库查询从弹窗改右侧停靠栏**（插入/替换双写回）——直击品类「弹窗过载」痛点；
2. 右键菜单族按对象类型分层+常用置顶（对标 GCCP 右键族）；
3. 键盘流：回车确认带出、下拉填充、Ctrl 多选批量换算/改费率；Ctrl+K 命令面板承载功能区命令（壳层 SearchModal 与 gaea CommandPalette 已有底座）；
4. 数据健康页对标升级：偏差项**双击定位回源**+重新检查闭环（GCCP 云检查范式）；
5. 密度参数：正文 ≥12px、表格行高紧凑档 1.125–1.2、关键路径红为唯一高饱和语义色（其余降 chrome 色占比）；
6. 进度：拖拽改期如有约束副作用，光标旁行内提示+一键撤销（MS Project FNLT 反面教材）；属性/检查面板右侧可停靠（现有 TaskInspector 抽屉升级）。

### D3 Agent 交互 2026 对齐（办公台）——P1，须拍板

1. **全局后台任务胶囊两态**：运行中（常驻、进度）vs 等你输入（琥珀+问号，点击直达）——收编 Z6 唤醒 Notice 与各板块任务中心为统一呈现（底层不动，只加壳层胶囊）；
2. **引用 chips 回链**：回答中的文档/知识引用做成可点击 chip 跳原文位置；
3. 记忆中枢加「它记得我什么」只读总览页（Topics 分组+来源标签+删除走既有建议制——不碰写入确认红线）；
4. 错误三段式文案规范（发生了什么/为什么/下一步试什么）入 ErrorCard。

### D4 小说工坊 Codex 活化——P2，须拍板

1. 角色别名自动检测+正文提及可点击跳转（活 Codex，Novelcrafter 范式）；
2. 生成卡加「本次上下文构成」chiclets（Sudowrite "Looked at:" 范式，与评测面板同源数据）；
3. AI 文本与作者文本颜色编码（NovelAI 范式，低成本的信任感）；
4. 划词自适应工具条（词→同义/描写；句→扩写/润色；段→重写/缩写）——现有场景章局部重写菜单升级；
5. Extract from Chat：聊天结果一键提取为场景节拍/角色条目/大纲节点。

### D5 画室生成节奏与版本树——P2，须拍板

1. 队列节奏三态（单次/参数变更自动跑/连续出图）+队列徽标+插队（ComfyUI Run Bar 范式，gaea 本就对接 ComfyUI）；
2. 版本树网格：生成结果网格 hover 出 vary/upscale/重绘分叉操作；
3. 参数滑杆化+推荐值标注（替代裸输入框）。

### D6 令牌三层化基建——并入 D1 前置步

现有 --md-sys-* 即 semantic 层，补 primitives 命名与 spacing/size 令牌即达标，无需推倒；语义令牌按角色命名原则对齐（color.text.warning 式）。

---

## 五、实施建议

- **阶段一（基建+换壳）**：D6+D1 全量——纯视觉/结构统一，零功能删除，走「测试契约锁结构视觉可重画」范式（v4.474 先例），首批监测/模型中心/设置三块；
- **阶段二（专业纵深）**：D2 造价/进度专项；
- **阶段三（AI 增量+模块专项）**：D3 办公台，D4/D5 随用户实弹反馈排期；
- 每刀走查沿用既有配方（无头 Edge+CDP+亮暗双档+overlap 扫描）；D1 换壳批必须逐板块目检截图。

## 附录：主要信源

微软 Copilot 系（microsoft.com/copilot/blog 2026-05-28；blogs.microsoft.com 2026-09-25；techcommunity 4561230）；Excel Copilot FAQ；Copilot Memory 支持文档；Notion Agent 帮助（notion.com/help/notion-agent）；WPS（bbs.wps.cn/topic/73559；365.wps.cn）；ChatGPT canvas 变迁（felloai.com/chatgpt-canvas）；Claude Artifacts/Memory（support.claude.com 17153992/11817273）；Raycast changelog；Novelcrafter（novelcrafter.com/features/codex 等）；Sudowrite docs（selection-menu 等）；NovelAI docs（editor/lorebook）；Dabble（dabblewriter.com/docs）；阅文作家助手；彩云小梦 App Store；GCCP6.0 帮助（aecore.glodon.com 三页）；斑马（zpert.com）；MS Project 支持文档（甘特/拆分/关键路径）；Linear 重设计（linear.app/now）；ComfyUI docs（interface/overview、nodes-2）；Midjourney/Krea/Leonardo 二手评测；agentic UX 综述（zylos.ai；setproduct.com）；设计令牌（digitalapplied.com；figma.com/resource-library/design-tokens）；中文可变字体（github.com/adobe-fonts/source-han-sans）。
