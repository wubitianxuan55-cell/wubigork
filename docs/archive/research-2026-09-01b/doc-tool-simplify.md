# 办公文档 AI 工具与通用生产力软件的 UI 简化模式调研（gaea「UI 化繁为简」弹药稿）

- 日期：2026-09-01。调研员：市场调研子代理（原始稿）。
- 硬约束框架：**简化界面 ≠ 删除功能**。每个对象按「它们如何在不删功能的前提下降低界面复杂度」来分析。
- gaea 指代：主区 = 对话流 + 文档预览 pane（docx 框选即改 / xlsx 直编 / pptx 逐页预览），右侧 = 运行工作台。
- 方法与网络限制说明：本环境仅部分域可达（microsoft.com / notion.com / github.com / linear.app / feishu.cn / ai.wps.cn / kimi.com / cn.bing.com 等）；DuckDuckGo、Mojeek、Ecosia、百度、qbitai、9to5google、windowscentral 均被 CAPTCHA/403/超时拦截。凡未能从可达来源核实的内容，一律在文中明示「未核实」。VS Code 文档经官方仓库镜像（github.com/microsoft/vscode-docs）核实，官方站点 code.visualstudio.com 在本网络超时。

---

## 0. 总机制：四条「不删功能的简化」通用手法

1. **统一入口 + 渐进披露（progressive disclosure）**：单一入口先给 3-5 个情境化默认动作，复杂能力按需展开，而非一次性铺开全部路径。微软把这套逻辑写进了官方设计叙事：「Rather than presenting every path at once, this design organizes what matters first」（不是一次呈现所有路径，而是先组织最重要的事）。（来源：https://www.microsoft.com/en-us/microsoft-365/blog/2026/05/28/introducing-a-new-design-for-microsoft-365-copilot/ ，下称 MS 博客）
2. **就地化（in-place invocation）与常驻面板分工**：轻操作在画布内就地完成，重操作进侧栏；两者共享同一 AI，只是深度不同（详见 §1）。
3. **可达性保底**：被藏起来的功能必须有确定性的寻回路径（溢出菜单、命令面板、可重置布局），「藏」才敢藏得深（详见 §2，VS Code 为范本）。
4. **密度分层与状态弱化**：列表只保证主行信息常驻，次级信息用悬停/次行/展开承载；状态徽标只给异常态高对比（详见 §3）。Linear 的方法论背书：「A tool should be simple to get started with and grow more powerful as you scale」（工具应易上手、随规模变强，而不是一开始就全量暴露）。（来源：https://linear.app/method/introduction ，下称 Linear Method）

---

## 1. 文档编辑 AI 面板如何「少而全」：统一入口 + 渐进披露 + 就地/常驻分工

### 1.1 M365 Copilot 2026 新设计（一手源，官方博客 2026-05-28，Jon Friedman 撰写）

微软自述「stepped back, simplified, and reworked key parts of the experience」（退一步、做简化、重做了关键体验），核心原则即 **progressive disclosure**；并强调架构性转变「We're moving from individual features to connected experiences」（从孤立功能走向连接的体验）。（来源：MS 博客）

四个具体交互，均属「减法不减功能」：

- **Prompt line 升级为任务感知工作区**：Copilot 把输入框从静态盒子变成「a task-aware workspace」，「the prompt line gives you more space to express your needs」；且「the prompt surface can expand to fill the experience」——输入面可以展开为完整体验（支持粘贴内容、行内格式化）。即：**默认一行，需要时才变大**，而不是常驻一个大而全的输入面板。（来源：MS 博客）
- **左导航窗格可展开/收起**：「a left navigation pane that expands and contracts reveals a clearer space for agents, conversations, and history」——agent、会话、历史共用一个可伸缩容器；配合「a shared pinning system and more room for session recall」（共享的固定/置顶系统 + 会话召回空间），常驻项由用户主动 pin，而不是系统塞满。（来源：MS 博客）
- **应用内统一入口 + 侧栏与就地双通道**：Word/Excel/PowerPoint/Outlook 内是「a consistent entry point across apps that sits above your work」（位于工作内容上方的跨应用一致入口）；从入口进入后「Copilot opens a side pane that works directly with your document」（侧栏直接对文档工作）；同时 Copilot 可以「within a paragraph, cell, or slide」被就地唤起——**段落/单元格/幻灯片级别就地调用**。轻编辑不出画布，多步任务进侧栏，这是「少而全」的分工核心。（来源：MS 博客）
- **能力化 agent + 响应深度自适应**：agent 按能力命名（Designer、Researcher、Word、Excel、PowerPoint），底层「Work IQ」智能层让 Copilot「adapts response depth」——按任务需要调深浅，而不是给所有请求同样重的输出。界面简化的一部分是**让答案本身也有渐进披露**。（来源：MS 博客）

效果数据（微软口径）：应用加载快 2 倍以上（load time 降 50%+），复杂 prompt 响应快约 10%；上线后 Copilot 使用量 Word +27%、Excel +33%、PowerPoint +43%、Outlook +30%——简化后使用率不降反升，可作为「简化≠功能衰减」的论据。（来源：MS 博客；数据为微软自述，未经第三方审计）

未核实：该博文的后续 rollout 文章（分应用上线节奏、Word 工具栏细节）在本网络未能检索到；Word「Ribbon 随 Copilot 简化为单行」一类说法未能核实，不作为论据。

### 1.2 WPS AI：把「功能入口」翻译成「意图入口」（一手源）

- 润色区提供固定短按钮组：「继续写」「缩写」「润色」「转换风格-古文风」，辅以「换一篇文档」换例——**一屏内 3-5 个高频动作 + 可替换上下文**，不铺全量写作功能。（来源：https://ai.wps.cn ）
- 表格场景用自然语言替代函数记忆：「无需记函数，你来说，WPS AI来做」，入口按钮「帮我写公式」「帮我筛选」——用两个意图按钮收纳整个函数/筛选功能族。（来源：https://ai.wps.cn ）
- 文档问答入口：「只需您输入想要了解的问题，WPS AI都能帮您在文档中进行精准搜索」——问答入口同时承担检索、摘要、写作多个功能族。（来源：https://ai.wps.cn ）
- 分级开放：页面标注「*网页版仅开放基础功能」并引导下载客户端体验更多 AI——**按端分级暴露功能**，也是不减功能的收纳策略（低配入口只展示基础集）。（来源：https://ai.wps.cn ）
- 页面未直接宣称侧边栏/悬浮工具条形态，入口在界面中的精确位置未核实。

### 1.3 Kimi（Agent 工作台）：按「产物类型」而非「功能菜单」组织入口（一手源）

- 主输入框一句话双轨：「尽管问，或做个 Agent 任务」——问答与任务共用一个入口，靠输入意图分流，不设两个功能区。（来源：https://www.kimi.com/ ）
- 侧栏按产物类型组织能力：PPT、文档、网站、表格、设计五个类型（另有 /slides、/docs、/websites、/sheets、/design、/deep-research 等直达页），即 **功能不删，但被归约为「用户能看懂的产物类型」**；「我的 Kimi」承载历史产物，「定时任务」承载周期任务。（来源：https://www.kimi.com/ ）
- Agent 任务页（Kimi Work）：定位「深度连接本地文件，支持浏览器自动化，全天候运行」；多智能体「自动协调多个专业智能体，拆分、解决多层次任务」；复杂执行细节（WebBridge 浏览、后台 Python）全部收在能力叙事里，UI 上只暴露「给一个目标」。营销页未展示进度时间线 UI 细节，任务运行中的界面形态未核实。（来源：https://www.kimi.com/products/kimi-work ）

### 1.4 未核实对象：Gemini 侧栏与飞书妙想

- Google Docs/Sheets 的 Gemini 侧栏：support.google.com 的相关帮助文章在本网络反复 404/超时，搜索源（cn.bing）被过滤。已知可核实的相邻一手信息仅一条：2026 年 Google Workspace 官方博客在售的「Sheets canvas」——「Sheets canvas turns data into interactive dashboards, custom study trackers, seating charts … all with a simple prompt」，即 AI 直接在画布内按一条简单 prompt 生成可交互元素（正文未能加载，细节未核实）。（来源：https://blog.google/products-and-platforms/products/workspace/sheets-canvas-for-google-sheets-spreadsheets/ ）
- 「侧栏常驻右侧 + 建议提示 chips + Insert/Copy/Refine 就地应用」等通行描述**未能核实**，仅作背景，不引为论据。
- 飞书妙想：feishu.cn 帮助中心与产品页均为 JS 渲染壳（抓取仅得页面标题），cn.bing 检索被字典词污染且「部分搜索结果未予显示」。可核实的仅有：官方 AI 产品页标题「飞书 AI｜真能用、真落地的企业 AI 助手」（来源：https://www.feishu.cn/product/ai ，仅标题）；中文维基条目称飞书「主打一站式无缝办公协作，近期則專攻人工智能加速」，条目（2026-07 修订）未描述妙想（来源：https://zh.wikipedia.org/wiki/飞书 ）。妙想交互细节全部未核实，不展开。

### 1.5 给 gaea 的启示（§1）

1. **右栏做成「一个」任务感知 pane，而不是每类文档一个面板**：pane 的默认 chips、占位文案、建议深度由「当前预览对象」决定（docx 段落 / xlsx 选区 / pptx 当前页各有 3-5 个 chips）——对齐 M365「task-aware workspace」与 WPS「意图按钮」。
2. **就地 vs 常驻的分工规则写死**：框选/选区/当前页触发的就地小工具条只放「一击即中」的动作（改写、格式化、求和、换图）；凡是需要多轮对话、上下文粘贴、长输出的，一键「在右侧继续」滑入常驻 pane。两条通道同一会话上下文（M365 side pane 与 in-canvas 同源）。
3. **Prompt 输入默认一行、可展开**：gaea 对话流卡片里的输入区做成「prompt line expands to fill」——默认一行，点开变多行 + 粘贴附件 + 格式工具，用完即收。
4. **响应深度自适应**：简单问题短答案、复杂任务自动产出带结构的方案卡（对齐 Work IQ「adapts response depth」），避免所有回答都用同一种重卡片轰炸对话流。
5. **产物类型化命名**：右栏产物列表用「文档 / 表格 / 幻灯片 / 报告」等用户词汇分组（对齐 Kimi sidebar），不暴露「导出 docx 引擎」「渲染器」这类实现词。

---

## 2. 预览/编辑面的 chrome 减法：工具栏收纳、上下文工具条与溢出可达

### 2.1 VS Code：面板「可藏、可移、可最大化、可重置」，但全部可达（一手源，官方文档镜像）

VS Code 是「藏而不删」的最完整范本：

- **任意区域可切换可见性**：Primary Side Bar、Secondary Side Bar、Panel（底部面板）、Zen Mode 均有 toggle 命令与菜单项（View > Appearance > …；如 "View: Toggle Secondary Side Bar Visibility"、"View: Toggle Zen Mode"、"View: Toggle Centered Layout"）——界面元素的存在与否是**用户状态，不是产品设计承诺**。（来源：https://github.com/microsoft/vscode-docs/blob/main/docs/configure/custom-layout.md ，官方 code.visualstudio.com/docs/configure/custom-layout 的仓库镜像；官网在本网络超时）
- **标题栏只留一个「Customize Layout」下拉**：最右侧按钮集中开/关全部区域与模式——顶栏本身保持极简，布局控制收进单一溢出点。（来源：同上）
- **视图可跨区搬移**：任意视图可拖拽在 Primary Side Bar / Secondary Side Bar / Panel 三区之间移动，落到已有视图上成组；键盘路径 "View: Move View"；每个视图有 "Reset Location" 单项重置，全局有 "View: Reset View Locations"——**布局个性化不被锁死，且永远可恢复**。（来源：同上）
- **面板对齐与最大化**：Panel Position 四向、Panel Alignment（Center/Justify/Left/Right）、"View: Toggle Maximized Panel"——同一内容在「半栏可见」与「占满」两档间切换，替代「开/关一个独立大页面」。（来源：同上）
- **布局持久化**：拖拽后的布局跨会话保留；Secondary Side Bar 的默认可见性可由设置 `workbench.secondarySideBar.defaultVisibility` 声明——**默认值声明式，用户改动持久化**。（来源：同上）

### 2.2 Linear：密度哲学 = 默认界面薄 + 提供额外语义（一手源）

- 「There is a lost art of building true quality software.」与「Keeping individuals productive is more important than generating perfect reports.」——界面为个体生产效率服务，不为展示全量状态服务。（来源：https://linear.app/method/introduction ）
- 「A tool should work for you, not the other way around.」；（来源：同上）
- 节制命名词汇本身也是 chrome 减法：「Don't invent terms if possible」「Projects should be called projects.」「Short specs are more likely to be read.」——**少造新词、少造新概念，界面文案负担直接下降**。（来源：同上）

### 2.3 未能核实的部分

- 「Word Ribbon 简化为单行」「上下文工具条/悬停工具条在 M365 2026 设计中的具体形态」未能从可达来源核实；WPS、飞书的工具栏收纳细节同样未核实。本节结论以 VS Code + Linear 一手内容为准。

### 2.4 给 gaea 的启示（§2，主区预览 pane 工具条）

1. **每类文档常驻工具 5-7 个 + 一个溢出桶**：docx（保存/撤销重做/缩放/目录/查找/导出/…）、xlsx、pptx 各自定义最小常驻集，其余进单一「⋯」溢出；溢出内支持搜索（对齐 VS Code Customize Layout 单点收纳）。
2. **工具条状态 = 用户状态**：预览 pane 的工具条可被用户整条隐藏（专注预览），隐藏状态持久化；提供「重置工具条」单项（对齐 Reset View Locations）。
3. **就地工具条走「浮现式」**：docx 框选浮现（改写/续写/格式刷）、xlsx 选区/列头浮现（求和/筛选/生成公式）、pptx 页缩略图悬停浮现（重写本页/换版式）——浮动条 3-4 个动作 + 「更多」进右栏 pane。
4. **预览内容两档占幅**：预览 pane 支持「半幅 ↔ 最大化」切换（对齐 Toggle Maximized Panel），而不是跳转独立全屏页面，保持对话流常在。
5. **文案即 chrome**：面板、按钮命名沿用用户已有词汇（「幻灯片」「筛选」「目录」），不造 gaea 自有术语（对齐 Linear「Projects should be called projects」）。

---

## 3. 文件/任务/产物列表的降噪：分层、克制徽标、空态与零值隐藏

### 3.1 Notion 数据库视图：同一数据的多副面孔（一手源）

- 核心句：「You can view the same database in multiple ways, and switch back and forth between them depending on your needs.」——**数据只有一份，视图是廉价可切换的**；List 视图官方自述即「a very clean, minimal layout of your database items」（列表视图 = 干净极简的默认）。（来源：https://www.notion.com/help/views-filters-and-sorts ）
- 视图 tabs 收纳：「You can click through your views to see your data in different ways」；tabs 可设 Icon only / Text only / both（个人级、按数据库记忆）；视图多时收进「{#} more...」——**tab 溢出有确定去处**。（来源：同上）
- 每视图独立设置单层菜单（View settings）：Layout、属性可见性（「Show or hide database properties for each view」）、Filter（「Add criteria based on property values to show or hide data」）、Sort、Group——**降噪的主手段是每视图的过滤与属性隐藏，而不是删数据**。（来源：同上）
- 视图设置互不串扰：「Settings applied to one database view won't be applied across all other database views automatically」；改过滤时可选「Save for everyone」或仅自己——**降噪偏好默认私有，不污染他人**。（来源：同上）

### 3.2 Kimi：列表即产物抽屉（一手源）

- 侧栏以产物类型分区（PPT/文档/网站/表格/设计）+「我的 Kimi」（历史产物）+「定时任务」（周期任务）——列表顶层只有三类：**类型、我的、定时**，没有混杂的全量时间线。（来源：https://www.kimi.com/ ）
- 任务后台化：定时任务「在后台悄然运行，准时完成」；夜间运行的前提是用户在设置里开启「保持电脑唤醒」——**任务列表只收「完成/待运行」的确定性条目，运行中细节不占列表**。（来源：https://www.kimi.com/products/kimi-work ）

### 3.3 空态/零值隐藏的通用佐证（间接）

- Notion 的 Filter 语义「show or hide data」即官方机制级的零值隐藏——过滤后为空的分组不必显示。（来源：https://www.notion.com/help/views-filters-and-sorts ）
- Linear 的 backlog 哲学是数据层降噪：「You don't need to save every feature request or piece of feedback indefinitely. Important ones will resurface, low priority ones will never get done.」——**列表里不留「永远不会被做」的死条目**，列表长度本身是设计出来的。（来源：https://linear.app/method/introduction ）

### 3.4 给 gaea 的启示（§3，产物/变更/任务列表）

1. **主行四件套**：名称 + 一个状态点 + 相对时间 + 唯一主操作；次级信息（变更摘要、影响范围、耗时）收进悬停展开或次行小字，不与主行并列堆叠。
2. **状态徽标克制规则**：仅异常态（失败/需确认/冲突）用高对比徽标；成功态用中性小圆点或仅颜色，不带文字徽标；进行中给进度但不发通知。
3. **列表顶部用 Notion 式视图 tabs**：「全部 / 变更 / 任务」三个默认视图 + 「+ 添加视图」，每个视图自带独立的 filter/sort/显示属性，互不串扰；tab 多了收进「更多 ⌄」。
4. **空态与零值**：过滤后为空的分组直接不渲染（仅剩一个视图时隐藏 tab 行）；「0 条失败」不渲染失败分组——只在异常出现时才让该类别进入视野。
5. **任务后台化**：长任务（批量转换、整文档改写）进「定时任务/后台」列表，运行中不占对话流；完成时以产物条目回到「我的」列表（对齐 Kimi 定时任务 + 产物抽屉）。

---

## 4. 设置与偏好收纳：声明式卡片与每面板独立开关

### 4.1 已核实的先例

- **VS Code：声明式默认 + 持久化覆盖**。Secondary Side Bar 的默认可见性是一个设置项（`workbench.secondarySideBar.defaultVisibility`），用户手动开合后又跨会话持久化——「默认值是声明式的，覆盖是持久的」，两者不冲突；每个视图/面板有自己的上下文菜单（含单项 Reset Location），**开关粒度到每个视图**。（来源：https://github.com/microsoft/vscode-docs/blob/main/docs/configure/custom-layout.md ）
- **Notion：每视图一层设置菜单**。View settings 是单层卡片式菜单（Layout / 属性可见性 / Filter / Sort / Group），且「每视图独立、默认私有、可选 Save for everyone」——设置范围（个人 vs 全体）作为显式选项而不是隐藏规则。（来源：https://www.notion.com/help/views-filters-and-sorts ）
- **Kimi Work：每特性一个开关 + 危险操作前置确认**。「保持电脑唤醒」是定时任务专属开关（「只需在设置中开启保持电脑唤醒选项」）；Agent 执行遵循「在修改、覆盖或在本地目录中运行代码之前，会提示你明确授权。未经你的同意，任何操作都不会执行」——**设置跟着特性走（出现在特性旁边），不集中进大设置页**；破坏性动作在动作前确认，而不是靠事后撤销承担复杂度。（来源：https://www.kimi.com/products/kimi-work ）
- **WPS：按端分级的功能暴露**。「网页版仅开放基础功能」——低配端不展示高级设置，而不是用灰置选项占位。（来源：https://ai.wps.cn ）

### 4.2 未核实

- 「声明式设置卡片（declarative settings card）」作为成体系的设计规范（如某家设计系统文档）未能核实；上文以四个具体交互先例替代。

### 4.3 给 gaea 的启示（§4，右栏面板与各 pane 的设置）

1. **每面板独立开关 + 声明式默认**：右栏工作台、对话流、预览工具条各自有「显示/隐藏/默认收起」开关；默认值在配置中声明，用户改动持久化并给单项「恢复默认」。
2. **设置跟特性走**：任务自动刷新、通知、唤醒/后台运行等开关放在特性卡片的角落（对齐 Kimi「保持电脑唤醒」），集中设置页只留全局项。
3. **视图/列表设置做成单层卡片**：产物列表每个视图一个 settings 卡（过滤/排序/显示列），不嵌套超过一层；改动范围显式标「仅对我 / 对空间所有人」（对齐 Notion Save for everyone）。
4. **危险动作前置确认**：应用变更、覆盖文件、批量删除前给明确授权弹层，文案写清「未经同意不执行」（对齐 Kimi Work 授权句式），替代事后复杂撤销链。

---

## 5. 模式 → gaea 落点速查表

| 模式 | 来源先例 | gaea 落点 |
|---|---|---|
| 统一入口 + 渐进披露（3-5 个情境 chips，输入面可展开） | M365 新设计「task-aware workspace / prompt line expands」 | 右栏 pane 顶部 chips 随预览对象变化；对话流输入默认一行可展开 |
| 就地唤起 vs 面板常驻分工 | M365「within a paragraph, cell, or slide」+ side pane | docx 框选/xlsx 选区/pptx 当前页浮出 3-4 动作迷你条；多步任务「在右侧继续」 |
| 响应深度自适应 | M365 Work IQ「adapts response depth」 | 快问答短卡，重任务方案卡；避免统一重卡片 |
| chrome 单点收纳 + 万物可 toggle + 可重置 | VS Code Customize Layout / Toggle / Reset View Locations | 预览工具条 5-7 常驻 + 溢出桶；工具条隐藏持久化；一键恢复默认布局 |
| 半幅 ↔ 最大化切换 | VS Code Toggle Maximized Panel | 预览 pane 两档占幅，不跳独立全屏页 |
| 同数据多视图、每视图独立过滤、tab 溢出收进 more | Notion views / {#} more / per-view settings | 产物/变更/任务列表视图化；空分组不渲染；徽标只给异常态 |
| 列表按产物类型组织、任务后台化 | Kimi sidebar 类型分区 / 定时任务 | 产物列表按「文档/表格/幻灯片/报告」分组；长任务后台跑、完成后入列表 |
| 设置跟特性走 + 前置授权 | Kimi 保持唤醒开关 / 执行前明确授权 | 每面板设置卡就地放；应用变更/覆盖文件前确认 |
| 分级暴露功能 | WPS「网页版仅开放基础功能」 | 窄窗口/低配模式自动降级工具条与右栏，不减后端能力 |
| 文案零发明 | Linear「Projects should be called projects」 | 全界面用用户词汇命名功能 |

---

## 6. 未核实项清单（明示）

1. M365 Copilot 2026 新设计的后续 rollout 文章（分应用节奏、Word/Excel 工具栏变化）——未能检索到，仅主博文（2026-05-28）已核实。
2. 「Ribbon → 单行简化」及 M365 悬停/上下文工具条细节——未核实。
3. Google Docs/Sheets Gemini 侧栏交互（侧栏位置、建议 chips、Insert/Copy/Refine 按钮）——官方帮助页在本网络不可达，仅「Sheets canvas」标题+摘要核实。
4. 飞书妙想的任何交互细节——官方页 JS 壳、搜索引擎被过滤，全部未核实；仅有官方产品页标题与维基概述性语句。
5. WPS AI 入口在客户端界面中的精确位置（侧边栏 vs 浮动）——官方产品页未明示，未核实。
6. 微软使用量增长数据（+27%/33%/43%/30%）为官方自述口径，未经第三方审计。

## 附：已核实一手来源索引

- MS 博客（M365 Copilot 新设计）：https://www.microsoft.com/en-us/microsoft-365/blog/2026/05/28/introducing-a-new-design-for-microsoft-365-copilot/
- VS Code 自定义布局（官方仓库镜像）：https://github.com/microsoft/vscode-docs/blob/main/docs/configure/custom-layout.md
- Notion 数据库视图：https://www.notion.com/help/views-filters-and-sorts
- Linear Method（索引与 Principles & Practices）：https://linear.app/method 、 https://linear.app/method/introduction
- WPS AI 产品页：https://ai.wps.cn
- Kimi 首页与 Kimi Work：https://www.kimi.com/ 、 https://www.kimi.com/products/kimi-work
- Google Workspace 官方博客（Sheets canvas 条目页）：https://blog.google/products-and-platforms/products/workspace/sheets-canvas-for-google-sheets-spreadsheets/
- 飞书 AI 产品页（仅标题）：https://www.feishu.cn/product/ai
- 中文维基飞书条目（AI 概述句）：https://zh.wikipedia.org/wiki/飞书
