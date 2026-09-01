# AI 代理/助手工作台的 UI 简化模式调研（原始稿）

- 日期：2026-09-01
- 调研目的：为 gaea（Wails 桌面应用：左侧会话栏 + 主区对话四标签「对话/轨迹/上下文/概览」+ 右侧工作台六标签「文件/产物/变更/任务/分工/浏览器」，宽度可拖）的「UI 化繁为简」刀供弹药。
- 硬约束：**简化界面 ≠ 删除功能**。每条模式都标注「被隐藏的功能从哪里仍可达」。
- 方法与限制：本稿所有一手 URL 均于 2026-09-01 实际抓取核实（部分经 r.jina.ai 代理读取，内容为原文转述）；openai.com 主域直接抓取会 403/超时，改用其 help center 直连与 r.jina.ai 代理；DuckDuckGo/Mojeek/国内 Bing 搜索均被验证码或本地化结果污染，未采信。未能核实的项在文中显式标注「未核实」。
- 阅读方式：来源 URL 随句标注；「未核实」= 有传闻/记忆但本次未能拿到一手文本。

---

## 0. 结论速览：七种「不删功能」的收敛手法

| # | 手法 | 一句话定义 | 代表先例 |
|---|------|-----------|---------|
| 1 | 活动驱动的面板 | 面板内容只在有产出时自动出现，不占常驻位 | Claude Artifacts 自动右侧窗、Codex 自动预览、Devin Auto-open Agents Tab |
| 2 | 摘要代替原始流 | 用人话小结+要点列表顶替日志/终端原始输出 | Codex IDE 内联 review、Devin Progress tab |
| 3 | 默认折叠的过程细节 | 逐步执行默认是一行/一勾，点开才见 JSON/命令 | Cowork "Running command" 折叠、ChatGPT agent narration |
| 4 | 藏进 composer | 斜杠命令/模式开关把设置类功能从菜单里拿掉 | Codex 22 条 slash 命令、/side、/btw |
| 5 | 命令面板作总回收站 | 数百个动作收进 Cmd+K，界面只留高频入口 | Linear、Devin、Notion Cmd+K |
| 6 | 用户自定义显隐 | 任何 tab/侧栏项可隐藏但不删，藏起来的进 More | Linear personalized sidebar、Notion ••• Hide、Devin hideable sidebar |
| 7 | 空态不出现 | 区块在第一次使用前根本不渲染 | Notion Favorites 首次收藏才出现、Linear 空组默认隐藏 |

核心命题（被多个案例反复印证）：**这些产品不是把复杂度删掉，而是把复杂度从「常驻视觉空间」迁移到「按需检索空间」（命令面板/斜杠/悬停/折叠），并把「何时出现」的决定权从用户手里拿走一半（自动打开/自动收起/自动合并）。**

---

## 1. 多面板/多标签产品如何收敛视觉复杂度

### 1.1 渐进披露：面板「有货才开」，过程细节「点开才看」

**Claude Artifacts：产物面板自动开、自动有**
- Claude 生成合格产物（实质性、自包含、通常 15 行以上）时，「内容显示在主聊天右侧的专用窗口里」——面板是**生成行为的副产品**，不是常驻 tab。来源：https://support.claude.com/en/articles/9487310-what-are-artifacts-and-how-do-i-use-them
- 多个产物共存于一个会话时，用右上角 chat controls（滑杆图标）切换，而不是开 N 个面板。来源：同上
- 编辑不换面板：选中文字 → "Edit with Claude" 就地改；出错时错误旁有 "Try fixing with Claude" 按钮就近修。来源：同上
- Cowork 里的 live artifacts 在 Artifacts 视图中带 "Cowork" 标签区分归属。来源：https://support.claude.com/en/articles/14729249-use-live-artifacts-in-claude-cowork

**Codex（桌面 app）：预览随任务完成自动打开**
- 「开启自动预览后，任务结束时应用可以打开生成的文件」；.html 生成物还能在「渲染预览/源码」间一键切换。来源：https://learn.chatgpt.com/codex/artifacts-viewer
- 聊天侧栏在任务运行中「可展示 agent 的计划、来源、生成文件与会话摘要」——侧栏内容随任务阶段变化，不是固定列表。来源：同上
- Annotations：点选文件局部即成为上下文，"refine without starting over or changing the parts you already like"——反馈入口就地化，避免开新面板。来源：同上

**Claude Cowork：过程默认一行，细节展开才见**
- 右侧栏 Progress 区「Steps will show as the task unfolds」，勾选随任务推进累积。来源：https://simonwillison.net/2026/Jan/12/claude-cowork/
- 每个执行步骤默认显示为一节 "Running command"，**点开才展开 Request JSON**（具体 shell 命令 + 一步人话解释，如 "Find draft files modified in the last 90 days"）。来源：同上
- Simon Willison 的直接评价：Cowork 与 Claude Code 引擎几乎相同，差别是「less intimidating default interface」（更不吓人的默认界面）。来源：同上
- 反面教材同样有价值：Simon 抱怨 Cowork 初版右侧栏**关不掉**，产物被挤在窄列里——「面板可收起」本身是简化的一部分，缺失即缺陷。来源：同上

**ChatGPT agent mode：narration + 可接管，代替双栏实时画面**
- 入口是 composer 的「tools dropdown → agent mode」，任何对话任意时刻可用；agent 跑在自己的虚拟电脑上（visual browser + text browser + terminal + API access）。来源：https://openai.com/index/introducing-chatgpt-agent/（经 r.jina.ai 读取）
- 屏幕上只有「on-screen narration」（文字旁白）说明正在做什么；用户可随时 interrupt、take over browser 或 stop——不需要常驻展示整个虚拟桌面。来源：同上
- "Watch Mode" 仅在关键动作（如发邮件）时要求主动监督；任务卡住可 pause 或要一份 progress summary。来源：同上
- 完成通知走手机 app 推送——等待期不在界面里占位。来源：同上

**gaea 启示（主区轨迹 tab + 右栏浏览器 tab）**：轨迹 tab 的默认态应该是「一行当前步骤 + 已完成计数」，JSON/命令级细节点开才渲染；agent 用浏览器时右栏浏览器 tab 才点亮，全程用旁白文字喂给对话流即可，不必常驻截图流。

### 1.2 智能默认：自动打开、自动合并、自动置前

**Devin：2026 年一整年的 changelog 就是一部「智能默认」教科书**（来源均为 https://docs.devin.ai/release-notes/overview ）：
- "Auto-open Agents Tab"（2026-04-01）：子会话出现时自动打开 Agents tab——面板**因事件而开**，而非因布局而开。
- "Checks Tab Always Visible"（2026-05-08）：高频审查信息反而反向地**固定常驻**——智能默认不等于全收起，高频项要顶到前面。
- "Progress tab brings these tools together in one unified view"（devin-session-tools 文档）：把 shell 命令、代码编辑、浏览器活动**合并进一条统一时间线**，三个原始工具面板仍存在但降级为「深入查看」入口。来源：https://docs.devin.ai/work-with-devin/devin-session-tools
- "Redesigned Changes Tab with persistent file tree"（2026-07-31）：变更审查的文件树持久化，避免反复展开。
- "Deduplicated File Tabs with Version Switcher"（2026-06-19）：同一文件多轮修改**去重成一个 tab** + 版本切换器——tab 数量不随任务时长线性增长。
- "Split Editor Groups in the Session Workspace"（2026-08-28，VS Code 式）：需要并排时才分屏，默认单栏。
- Slack 集成里 "Active Todo in Slack Plan Header"（2026-06-05）：任何界面只显示「当前那一个 todo」，不显示整张计划。

**Codex IDE 扩展：评审不新增导航面板**
- 卖点原话："Review a concise summary and the changed lines **without an extra navigation pane**"；每个文件一行 "Edited retry.ts +2−2" + 内联 Undo/Review；验证结果渲染成 "Validation passed" 要点列表而不是原始日志面板。来源：https://developers.openai.com/codex/ide（重定向至 https://learn.chatgpt.com/docs/codex/ide ）
- "Continue in" 开关在 Work locally / Cloud 间切换同一任务——环境切换是一枚开关，不是两个界面。来源：同上

**gaea 启示（右栏 6 tab）**：
1. 「产物」tab 在产物落盘那一刻自动置前并高亮（学 Devin Auto-open）；
2. 「文件」tab 按文件去重 + 版本切换器，禁止每轮对话新开一个文件 tab；
3. 「任务」tab 默认只在头部显示一行 active todo（学 Devin Slack plan header）；
4. 「变更」tab 内联在对话流里给每文件 +N−N 摘要，tab 里只做深入 diff；
5. 反向规则：最高频的信息（如当前校验状态）要学 Checks Tab 常驻，而不是折叠。

### 1.3 空态隐藏：区块在首次使用前不渲染

- Notion 侧栏：**Favorites 区在第一次给页面打星之前根本不出现**；移除收藏后区块随之消失。来源：https://www.notion.com/help/sidebar
- Notion 侧栏：AI 关闭的工作区，「Chats with Notion AI」tab 消失、原位显示 New 按钮——功能不存在/未启用时入口也不存在。来源：同上
- Notion 侧栏：Favorites、Teamspaces、Shared、Private 各区块标题可点击折叠，记忆状态。来源：同上
- Linear display options：**"Show empty groups" 是开关，默认不显示空分组**；triage issues 默认隐藏。来源：https://linear.app/docs/display-options
- Linear 侧栏：Customers 链接**只在启用了 Customer requests 功能的工作区出现**——入口跟功能开通状态联动。来源：https://linear.app/changelog/2024-12-18-personalized-sidebar

**gaea 启示（右栏 + 左栏）**：右栏「任务/分工/产物」在没有内容时不应显示可点的空 tab（或至少 tab 上不渲染任何骨架屏）；左栏「收藏/置顶」分组学 Notion——第一次置顶会话之前不渲染该分组。空态时给一个「+ 新建」触发点即可。

### 1.4 上下文相关面板：不相关时消失/让位

- Devin 会话内工具面板按需出现：Interactive Browser 位于 Desktop tab，浏览器/IDE/终端三个工具面板不固定三分屏，而是 tab 化 + "add-tab menu" 按需添加。来源：https://docs.devin.ai/work-with-devin/devin-session-tools
- Devin "Agents Tab for Child Sessions"（2026-03-27）：只有存在子会话时才有 Agents tab；配合 Auto-open 形成完整闭环「出现→自动开→树状嵌套（2026-08-21 Nested Sub-Devin Sessions）」。来源：https://docs.devin.ai/release-notes/overview
- Claude Cowork：内置浏览器「opens a browser in the Cowork side panel instead of yours」，且 "doesn't touch your tabs or logins"——面板内容与用户自身环境隔离，用完即走。来源：https://claude.com/product/cowork （经 r.jina.ai 读取）；https://support.claude.com/en/articles/16607400-use-the-built-in-browser-in-claude-cowork
- Claude Cowork 右栏 Context 区固定分三段（Selected folders / Connectors / Working files）——上下文面板的内容**随任务挂载的上下文种类**变化，而不是一股脑列出全部能力。来源：https://simonwillison.net/2026/Jan/12/claude-cowork/
- Manus：Wide Research「only supports automatic triggering」——复杂任务自动裂解为最多 20 个并行子任务（"equivalent to having 20 Agents"），用户不配置任何并行 UI；四步流水线 Task Decomposition → Parallel Agent Deployment → Independent Processing → Result Synthesis 全部由主 agent 收口合成。来源：https://help.manus.im/en/articles/11960169-what-is-wide-research ；https://manus.im/docs/features/wide-research
- Manus Chat mode vs Agent mode：轻问答走 Chat（不耗 credits），复杂任务走 Agent（自主规划）——两种界面形态共用一个产品，模式由用户/系统选择。来源：https://help.manus.im/en/articles/11711128-what-are-the-differences-between-chat-mode-and-agent-mode （注：Adaptive mode 细节该帮助页未展开，未核实）

**gaea 启示**：右栏「分工」tab 只有在多代理/子任务运行时才有内容——没有并行任务时直接收进「任务」tab 的一个分组即可，不必独立 tab；「浏览器」tab 只在 agent 发起网页操作期间置前，结束后回落。

---

## 2. 顶部/侧边 chrome 的减法：常驻入口数、图标 vs 文字、二级功能去向

### 2.1 常驻入口数：收敛到「一个输入框 + 一组模式开关」

**Claude Cowork（最激进）**：
- 「chat and Cowork share one home」——聊天与代理工作**共用同一个消息框**，靠消息框里的 "Cowork"/"Chat" 选择器切换；连接器靠输入框 "+" 菜单挂载；权限模式（Manual / Auto / Skip 三档）是消息框里的 mode selector。来源：https://support.claude.com/en/articles/13345190-get-started-with-claude-cowork
- 桌面 app 中 Cowork 是与 Chat、Code 并列的**第三个顶级 tab**——顶级常驻入口共 3 个，其余全在会话内。来源：https://simonwillison.net/2026/Jan/12/claude-cowork/
- 左侧栏只有 "+ New task" + 任务列表；计划任务单独一个 "Scheduled" 入口。来源：同上；https://support.claude.com/en/articles/13854387-schedule-recurring-tasks-in-claude-cowork

**Codex 桌面 app（次激进）**：
- composer 上方一个 "Chat or Work" toggle 定格局面；快速提问另有 "Quick chat icon"。来源：https://learn.chatgpt.com/docs/app
- 左栏常驻四项：New thread / Automations / Skills / Threads（Threads 按项目文件夹分组）。来源：https://simonwillison.net/2026/Feb/2/introducing-the-codex-app/
- 其余一切进 slash 命令（见 §4）。同一账号矩阵下客户端也只有四个：ChatGPT desktop app (Codex mode)、CLI、IDE extension、web（chatgpt.com/codex）。来源：https://help.openai.com/en/articles/11369540-using-chatgpt-agent

**Devin（工具型，稍多）**：
- 会话侧栏 2026-08-21 重设计："customizable nav tabs, grouping, richer filtering" + "Hideable Sidebar"（可整体藏起，悬停 peek）——入口数量本身交给用户调。来源：https://docs.devin.ai/release-notes/overview
- 会话内 tab 集：Desktop（由 Browser 更名）/ Progress / Changes / Agents / Checks，另有 "add-tab menu" 动态加面板。来源：https://docs.devin.ai/work-with-devin/devin-session-tools ；https://docs.devin.ai/release-notes/overview

**Notion / Linear（传统 SaaS 参照系）**：
- Notion 侧栏顶部功能入口固定五个（Search / Home / Chats with Notion AI / Meetings / Inbox）+ 底部五个（My Tasks / Library / Marketplace / Help / Trash），中间才是内容树；每项都能被用户 Hide。来源：https://www.notion.com/help/sidebar
- Linear 侧栏项可全部重排/隐藏，藏掉的进 "More menu"；未读显示可选 count 或 dot。来源：https://linear.app/changelog/2024-12-18-personalized-sidebar

### 2.2 图标 vs 文字 vs 分组的取舍（观察，非官方规范）

- Linear 命令菜单 2019 重设计：按功能**分组 + 组内再按类型细分 + 加图标**——图标是辅助识别，不是省空间的唯一手段；分组才是主要可扫读性来源。来源：https://linear.app/changelog/2019-12-18-new-command-menu
- Devin 侧栏未读用「status label + unread dot」双编码（2026-05-08），Linear 用 count or dot 二选一——两者都把「红点还是数字」降级为用户偏好而非设计强加。来源：https://docs.devin.ai/release-notes/overview ；https://linear.app/changelog/2024-12-18-personalized-sidebar
- Notion 顶部入口全部是**文字标签**（Search/Home/Meetings/Inbox），图标只用于折叠箭头和拖柄。来源：https://www.notion.com/help/sidebar
- 未核实：任何一家官方给出的「图标 vs 文字」可用性结论；以上为产品界面观察。

**gaea 启示（左栏 + 顶栏）**：左栏常驻结构建议收敛为「New session 按钮 + 会话列表（可分文件夹）+ 一个 More」，其余（计划任务、知识、设置）进 More 或命令面板；tab/分组用文字短标签而非纯图标（Notion/Linear 先例），红点/数字做成用户偏好项。

---

## 3. 信息密度控制：密度切换、卡片 vs 行、留白

- **Linear Display properties**（最贴近 gaea 场景）：issue 行上 19 种属性（ID/状态/负责人/优先级/SLA/项目/截止/估算/标签/PR…）**逐项显隐**，文档原话强调「与过滤器不同，这只隐藏数据，不删除 issue」——就是「简化≠删功能」的官方表述。来源：https://linear.app/docs/display-options
- Linear 视图切换 Cmd+B（list/board/timeline）+ "Set as default" / "Reset to default"：密度形态是视图属性，可一键回滚。来源：同上
- Linear 计数即密度：点击分组头上的数字在「issue 数 / 总估算」间切换——同一个像素两种信息密度。来源：同上
- **Notion 数据库 peek view**：ctrl+shift+K/J 在预览卡内翻页——详情用「peek 覆盖层」而非新页面，保住列表上下文。来源：https://www.notion.com/help/keyboard-shortcuts
- Notion 侧栏 Private/Shared 区可设「显示条数 5 到全部」+ 排序方式，超出部分进 "More" pane（搜索/排序⇅/置顶>>）。来源：https://www.notion.com/help/sidebar
- **Codex IDE**：验证结果 "Validation passed" 要点列表**替代**原始日志视图——密度控制的上位手段是「换信息粒度」而非「缩字号」。来源：https://developers.openai.com/codex/ide
- Cowork 输出物本身分层："What matters most" 摘要 → 分级章节 → gap 段 → "Suggested next step"，同一份 memo 结构复用（"Memo 3 of 12"）——内容侧密度靠固定文档骨架。来源：https://claude.com/product/cowork
- Devin：sticky sidebar headers（2026-04-01）+ 会话通知的状态标签（2026-05-08）——滚动长列表用粘性分组头降低回扫成本。来源：https://docs.devin.ai/release-notes/overview
- 未核实：任何产品的「紧凑模式/密度切换」全局开关（Linear/Notion 本次抓取的文档均未见全局 density toggle；仅见上述视图级/属性级控制）。

**gaea 启示**：右栏每个 tab 学 Linear 做属性级显隐（如变更 tab 的「显示测试结果列」），不做全局紧凑模式；列表默认显示 5 条 + More（Notion）；会话列表加粘性日期/分组头（Devin）。

---

## 4. 「复杂度去哪了」：隐藏功能的可达入口（有出处实例）

### 4.1 斜杠命令（composer 内检索）
- Codex 官方口径："Slash commands let you run actions **without leaving the chat composer**"，输入 "/" 列出全部、继续输入过滤。实测命令表：/approve /cloud /cloud-environment /local /worktree /fork /side /task /project /compact /memories /ide-context /mcp /status /model /reasoning /personality /fast /plan /goal /init /review /pet /feedback，skills 还会进 slash 列表，自定义 prompt 成为 "/prompts:<name>"，"@" 引用上下文，"$" 显式调 skill。**连桌面宠物（/pet）、MCP 服务器状态（/mcp）、推理力度（/reasoning）都只从斜杠可达**。来源：https://learn.chatgpt.com/codex/reference/slash-commands
- 双向通道：/goal 启动的目标激活后出现进度行，「按钮让你管理 goal 而不用再打斜杠命令」——**slash 是入口，激活后回归可视按钮**，两条路都通。来源：同上
- Devin side chats 三入口：消息 hover 菜单、add-tab 菜单、"/btw 你的问题"，在 worklog 旁开临时面板——主对话不被污染。来源：https://docs.devin.ai/work-with-devin/devin-session-tools

### 4.2 命令面板（全局检索）
- Linear：command menu 是「one of the core components」，包含 "hundreds of actions"（改 issue 属性到切 UI 主题），且**按当前视图重排优先级**（看 cycles 时 cycle 命令排前）。来源：https://linear.app/changelog/2019-12-18-new-command-menu
- Linear 连收起侧栏都能从命令菜单搜 "Collapse"/"Expand" 完成。来源：https://linear.app/changelog/unpublished-collapsible-sidebar（该 URL 实际载有 2023-01-12 Collapsible Sidebar 条目）
- Devin 命令面板 2026-05-29 上线（Cmd/Ctrl+K），随后把 pin/unpin（06-24）、archive/unarchive（06-26）、复制会话 URL（05-22）、切换组织（06-17）全部**只加进面板**而不加界面按钮。来源：https://docs.devin.ai/release-notes/overview
- Notion：cmd/ctrl+K（或 P）搜索并跳转最近页面，是页面级导航的主入口。来源：https://www.notion.com/help/keyboard-shortcuts
- Linear display options 也有命令面板路径："Show display options"；键盘 Shift+V 直达。来源：https://linear.app/docs/display-options

### 4.3 悬停与就地菜单
- Linear personalized sidebar：右键任何侧栏项显隐/重排，或 "Customize sidebar" 一次看全。来源：https://linear.app/changelog/2024-12-18-personalized-sidebar
- Devin：playbook 在消息框悬停出预览+复制（2026-07-24）；side chat 从消息 hover 菜单发起。来源：https://docs.devin.ai/release-notes/overview ；devin-session-tools
- Claude Artifacts："Edit with Claude" 选中即就地编辑；错误处就地 "Try fixing with Claude"。来源：https://support.claude.com/en/articles/9487310-what-are-artifacts-and-how-do-i-use-them
- Notion：侧栏项 ••• → Show / Move up/down / Hide；区块头点击折叠。来源：https://www.notion.com/help/sidebar

### 4.4 模式/权限选择器（把决策 UI 钉在输入框）
- Cowork 三档权限 Manual/Auto/Skip 就在 chat box 的 mode selector；连接器权限每档可配 Always allow / Needs approval / Blocked；永久删除文件强制弹批准。来源：https://support.claude.com/en/articles/13345190-get-started-with-claude-cowork
- ChatGPT agent 的 takeover mode：接管时用户键入（如密码）模型不可见——危险操作的 UI 是「临时移交」而不是永久开关。来源：https://openai.com/index/introducing-chatgpt-agent/
- Manus 接管：Manus 遇到困难会主动 prompt 用户接管 browser/VS Code，用户也可主动接管——控制权切换是事件驱动弹层，非常驻按钮。来源：https://help.manus.im/en/articles/11711218-how-can-i-take-over-manus-browser-or-vs-code

### 4.5 能力开关后置到设置
- Claude Artifacts 本身受 Settings > Capabilities 的 "Code execution and file creation" 开关控制——整个面板体系可以一键不存在（但不算删功能，是用户主动关闭）。来源：https://support.claude.com/en/articles/9487310-what-are-artifacts-and-how-do-i-use-them
- Codex 高危能力后置：Settings > Browser > "Enable full CDP access"（开发者模式）才开放浏览器底层控制。来源：https://help.openai.com/en/articles/11369540-using-chatgpt-agent

---

## 5. 对 gaea 的落地建议（汇总）

### 5.1 右栏六 tab（文件/产物/变更/任务/分工/浏览器）
1. **tab 按事件点亮**：产物生成→自动置前「产物」；agent 开浏览器→自动置前「浏览器」（先例：Devin Auto-open Agents Tab，2026-04-01；Claude Artifacts 自动右窗）。来源见 §1.2/§1.4。
2. **合并原始流**：「文件/变更/浏览器」三个 tab 的运行期活动合并为一条 Progress 时间线，原始视图降级为点击深入（先例：Devin Progress tab；devin-session-tools）。
3. **文件 tab 去重**：同文件多版本合成一个 tab + 版本切换器（先例：Devin 2026-06-19）。
4. **任务 tab 一行式**：头部永远只有一行 active todo，展开才见全表（先例：Devin Slack plan header，2026-06-05）。
5. **分工并入任务**：无并行子任务时「分工」不设独立 tab，收进任务分组；有子会话时才出现并可自动打开（先例：Devin Agents Tab 的存在条件）。
6. **每 tab 属性级显隐**：用户可隐藏列/区块但不删数据，隐藏项进 tab 的 More（先例：Linear Display properties 官方原话「只隐藏数据，不删除 issue」）。
7. **浏览器隔离**：内嵌浏览器与用户会话/登录态隔离，用完自动回落（先例：Cowork built-in browser "doesn't touch your tabs or logins"）。

### 5.2 主区四 tab（对话/轨迹/上下文/概览）
1. **轨迹默认一行**：每步一行人话摘要，点开才见 JSON/命令（先例：Cowork "Running command" 折叠 Request JSON；ChatGPT agent narration）。
2. **校验结果要点化**：用 "Validation passed" 式 bullet 代替原始日志面板；diff 评审内联在对话流，不另开导航面板（先例：Codex IDE "without an extra navigation pane"）。
3. **上下文 tab 学 Cowork 三段式**：Selected folders / Connectors / Working files；配合 @-引用 chips（先例：Codex IDE @-references）。
4. **概览 tab 固定文档骨架**：What matters most → 分级细节 → 建议下一步（先例：Cowork memo 结构）。
5. **上下文压缩命令化**：/compact、/status、/plan 等进 composer，主区不留设置按钮（先例：Codex slash 命令表）。
6. **权限模式常驻输入框**：三档（逐步批准/自动/跳过）做成输入框旁 selector，删除类操作强制批准（先例：Cowork mode selector；ChatGPT agent takeover）。

### 5.3 左侧会话栏
1. **整体可藏 + 悬停 peek**：一个键（Linear 用 `[`）收起，hover 边缘临时展开（先例：Devin Hideable Sidebar 2026-08-21；Linear Collapsible Sidebar 2023-01-12）。
2. **会话文件夹化**：可折叠分组 + 粘性头 + 未读点/数字可选 + 行内重命名 + 拖拽 + 批量归档（先例：Devin 2026 全年侧栏 changelog 群）。
3. **右键个性化**：任意项可隐藏进 More menu（先例：Linear personalized sidebar 2024-12-18）。
4. **空态不渲染**：置顶/收藏分组首次使用前不出现（先例：Notion Favorites）。
5. **命令面板做总回收站**：pin/归档/重命名/清空文件夹等全部进 Cmd+K 而不加按钮（先例：Devin 把 archive/unarchive/pin 只加进命令面板；Linear hundreds of actions）。
6. **顶部只留一个 + New**（先例：Cowork "+ New task"；Codex app "New thread" + Threads 按项目分组）。

### 5.4 一条总原则（供写进设计文档）
把界面分为三层：**常驻层**（输入框 + 当前活动面板 + 高频校验状态）、**点开层**（tab 内详情、折叠的 Request JSON）、**检索层**（命令面板 + slash + 悬停菜单）。任何新功能先放检索层，用出频率再逐层上浮——反向（删功能）永不发生（先例：Codex /goal → 按钮进度行的双向通道）。

---

## 6. 来源清单与未核实项

### 已核实一手来源（2026-09-01 抓取）
- ChatGPT agent mode 官宣（经 r.jina.ai）：https://openai.com/index/introducing-chatgpt-agent/
- Codex IDE 扩展文档：https://developers.openai.com/codex/ide（重定向 https://learn.chatgpt.com/docs/codex/ide ）
- Codex 斜杠命令参考：https://learn.chatgpt.com/codex/reference/slash-commands
- Codex 产物预览（Work with files）：https://learn.chatgpt.com/codex/artifacts-viewer
- Codex 客户端矩阵 / Codex mode / Record & Replay：https://help.openai.com/en/articles/11369540-using-chatgpt-agent
- Codex app 上手（Simon Willison，2026-02-02）：https://simonwillison.net/2026/Feb/2/introducing-the-codex-app/
- Claude Cowork 产品页（经 r.jina.ai）：https://claude.com/product/cowork
- Cowork 入门/权限模式：https://support.claude.com/en/articles/13345190-get-started-with-claude-cowork
- Cowork 首晒 UI 布局（Simon Willison，2026-01-12）：https://simonwillison.net/2026/Jan/12/claude-cowork/
- Artifacts 行为：https://support.claude.com/en/articles/9487310-what-are-artifacts-and-how-do-i-use-them
- Cowork live artifacts：https://support.claude.com/en/articles/14729249-use-live-artifacts-in-claude-cowork
- Cowork 内置浏览器：https://support.claude.com/en/articles/16607400-use-the-built-in-browser-in-claude-cowork
- Devin release notes（2026-03~08 全量 UI 变更）：https://docs.devin.ai/release-notes/overview
- Devin 会话工具（Desktop/Progress tab、side chats、/btw）：https://docs.devin.ai/work-with-devin/devin-session-tools
- Devin 2.0（Interactive Planning、并行 Devin）：https://cognition.com/blog/devin-2
- Devin Desktop（Agent Command Center、Spaces）：https://cognition.com/blog/introducing-devin-desktop
- Manus Chat/Agent mode：https://help.manus.im/en/articles/11711128-what-are-the-differences-between-chat-mode-and-agent-mode
- Manus Wide Research：https://help.manus.im/en/articles/11960169-what-is-wide-research ；https://manus.im/docs/features/wide-research
- Manus 接管 browser/VS Code：https://help.manus.im/en/articles/11711218-how-can-i-take-over-manus-browser-or-vs-code
- Manus Desktop "My Computer"：https://manus.im/docs/features/desktop.md
- Notion 侧栏（Navigate with the sidebar）：https://www.notion.com/help/sidebar
- Notion 键盘快捷键：https://www.notion.com/help/keyboard-shortcuts
- Linear Display options：https://linear.app/docs/display-options
- Linear Inbox：https://linear.app/docs/inbox
- Linear Personalized sidebar（2024-12-18）：https://linear.app/changelog/2024-12-18-personalized-sidebar
- Linear Collapsible sidebar（2023-01-12）：https://linear.app/changelog/unpublished-collapsible-sidebar
- Linear Command menu（2019-12-18）：https://linear.app/changelog/2019-12-18-new-command-menu
- Linear Desktop navigation（2021-09-23）：https://linear.app/changelog/2021-09-23-desktop-navigation

### 未核实项（明示）
1. **ChatGPT 消费级桌面应用的侧栏收起、多 tab、Pulse、Canvas 自动展开**的具体行为：help.openai.com 站内搜索入口 403，release notes 文章 ID 未能定位；本稿仅采信 agent mode 官宣页。若需补课，可后续抓 help.openai.com 站内文章或 Wait but Why/TC 报道。
2. **Manus 右侧 "computer" 面板的展开/全屏/收起交互细节**：帮助文档以截图呈现，正文无文字描述；本稿只采信「接管 prompt」相关文字。
3. **Codex app 的 tab/diff review 交互**：Simon Willison 文章明确未覆盖；learn.chatgpt.com 对应页（/codex/reference/settings、/docs/codex/changelog）404。
4. **任何产品的官方「图标 vs 文字」可用性结论**：未见一手文档，§2.2 仅为界面观察。
5. NN/g《Progressive Disclosure》理论框架（https://www.nngroup.com/articles/progressive-disclosure/）：未抓取原文，仅作背景概念引用。
6. Manus "Adaptive mode" 的具体工作机制：帮助页 FAQ 仅有问题无答案。
7. Linear 移动端自定义导航（2026-01-22 changelog）：条目标题存在（sitemap 可见），正文未抓取。
