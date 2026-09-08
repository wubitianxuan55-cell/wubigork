# 通用 AI 助手 / Agent 产品如何呈现与交付「任务产出物」——调研原始稿

- 调研日期：2026-09-03
- 调研人：gaea 市场调研子代理（原始稿，未做任何代码改动）
- 调研目的：gaea 办公板块已有「交付文件卡（正文提及 + 后端权威登记表、缺失态徽标）+ 右栏会话产物面板（版本时间线/恢复、一键 zip、证据链、reveal）+ 预览面板（docx 框选即改+修订、xlsx Plan→Apply 直编+图表、md/text/PDF/OCR）」。本文只找**竞品已验证、我们还没有的跃升模式**，不重复验证现状。
- 方法与可信度标注：优先官方文档 / 官方博客 / 官方更新日志 / 官方帮助中心（标注〔官方〕）；第三方评测与社区帖仅作补充（标注〔第三方〕）。每条结论附来源 URL。查不到的一律写「未查到」，不编造。
- 说明：本文所有网络资料均为调研当日（2026-09-03）检索所得；功能与价格可能随时间变化，引用时请注意各来源的时效。

---

## 1. Claude（Artifacts / 文件创建 / Claude Code Artifacts）

### 1.1 产物如何呈现
- **对话内嵌卡 + 右侧专用窗口双形态**。Artifacts 在主聊天右侧的「dedicated window」中打开；一次对话可有多个 artifact，通过右上角 chat controls（滑块图标）切换。支持类型：Markdown/纯文本文档、代码、单页 HTML 网站、SVG、图表/流程图、可交互 React 组件。〔官方〕https://support.claude.com/en/articles/9487310-what-are-artifacts-and-how-do-i-use-them
- **文件类交付直接落在对话流里**：「Claude will generate the file, which you can then download directly from the conversation」，文件在整段对话内持续可下载；也可直接保存到 Google Drive。产物类型：.xlsx / .pptx / .docx / PDF / PNG 可视化 / Python 脚本。底层是 claude.ai 内的沙箱容器跑 Python/JS。〔官方〕https://support.claude.com/en/articles/12111783-create-and-edit-files-with-claude
- **侧边栏产物库**：侧边栏有独立「Artifacts」分区聚合所有作品、支持从零新建。注意：会话内 artifact **不会自动**进入该库，必须点「Publish」后才收录——官方把"库"定位为已发布物的集合。〔官方〕https://support.claude.com/en/articles/9487310-what-are-artifacts-and-how-do-i-use-them

### 1.2 版本与迭代
- **版本选择器（version selector）**：artifact 窗口内可直接切换不同版本；手动编辑不会改变 Claude 对原始内容的记忆；编辑历史消息会产生「不同对话版本 + 各自的 artifact 集合」。〔官方〕同上 9487310
- **Live Artifacts（Claude Cowork）版本历史**：可打开版本历史查看随时间的变化、**比较早版本与当前版本**、恢复到先前状态。〔官方〕https://support.claude.com/en/articles/14729249-use-live-artifacts-in-claude-cowork （经检索摘要转述，未逐字核对原文）
- **按引用修正**：官方帮助中可验证的形态是「高亮选中文本 → 点 **Edit with Claude** → 输入要求，Claude 只改所选处」；多 Markdown 文件草稿（如 skill/插件）支持**排队多个文件的编辑请求、一次性批量应用**；artifact 报错时附近有 **Try fixing with Claude** 按钮把错误详情带入新消息让 Claude 修。**任务书里提到的「Refs」这一官方功能名称未查到**（多个检索均无 Anthropic 官方 changelog 条目），最接近的官方能力即上述「选区即改」与错误修复按钮。〔官方〕https://support.claude.com/en/articles/9487310-what-are-artifacts-and-how-do-i-use-them
- **Claude Code Artifacts 的更新模型**：让 Claude 修改页面文件并**重新发布到同一 URL**，已打开的查看者原地看到变化；跨会话更新需给出 artifact URL 或用 `/artifacts` 附加，否则会新建。〔官方〕https://code.claude.com/docs/en/artifacts

### 1.3 交付动作
- artifact 窗口右下角：**View code / Copy / Download**（Download 为"下载文件到对话外使用"）。〔官方〕9487310
- **发布公开链接**：Publish → 复制公开链接；Free/Pro/Max 可公开发布，任何拿到链接者**无需登录**即可查看与操作；发布后还有 **Get embed code**（需在 Allowed domains 白名单域名）；**Unpublish 后该 artifact 永久不能再发布**，须新建 artifact 再发布。〔官方〕https://support.claude.com/en/articles/9547008-publish-and-share-artifacts
- **组织内分享**：Team/Enterprise 不能公开发布，只能 Share → Share & copy link，仅限本组织成员登录访问；**分享会连带暴露来源对话的所有附件与文件**（官方明确警示敏感文档风险）。〔官方〕同上
- **Remix 已下线**，替代流程是打开公开 artifact → Copy → 粘贴进新对话描述改动 → 生成独立新 artifact，原件不动。〔官方〕同上
- **文件保存到 Google Drive**（文件创建能力）。〔官方〕12111783

### 1.4 Claude Code Artifacts（「会话输出即成品页」的交付闭环）
- 把会话中的输出发布为「claude.ai 私有 URL 上的 live 交互网页」，权限流经 Claude Code 的许可模式（Auto 模式由分类器审、Manual/Accept edits 模式弹窗确认，首次批准后自动重发布）。
- **每次发布即一个版本**；分享菜单里有**版本选择器**（"Sharing version 2"）和 **Always share latest version** 开关——即"分享指定快照 or 始终最新"两种语义。
- **评论与多人协作**：组织内分享（Team/Enterprise）可指定成员为 editor，editor 把 URL 给 Claude 即可拉取当前内容再发布其修改；org 内 artifact 支持评论线程，**线程须有人点 Send to Claude 或 @claude 才激活**，Claude 只回复/解决已激活线程；公开分享禁用评论。
- **硬约束**：纯静态单页（相对链接不解析）、仅 .html/.htm/.md、渲染后最大 16 MiB、CSP 白名单（cdnjs/Tailwind/jQuery/jsDelivr/Google Fonts）、托管于沙箱 `*.claudeusercontent.com` 域、带 MCP 连接器实时取数的 artifact **永远不能公开分享**。〔官方〕https://code.claude.com/docs/en/artifacts ；发布公告〔官方〕https://claude.com/blog/artifacts-in-claude-code

### 1.5 验收闭环
- 用户批注（选区 + Edit with Claude）、错误一键回修（Try fixing with Claude）、评论线程激活后 Claude 参与讨论（Claude Code Artifacts）、Cowork 版本对比后回滚——构成"人指出 → AI 改 → 版本留痕"的循环。〔官方〕见上

### 1.6 定价/限额相关
- 文件创建：全部套餐（Free–Enterprise）默认开启，开关在 Settings > Capabilities 的「Code execution and file creation」；**上传与下载单文件上限 30MB**；无文件数量上限；"creating files will use more of your limit compared to normal chats"（创建文件比普通对话更耗额度），任务时长与容器存活期有上限（数值未公开）。〔官方〕12111783
- 公开发布仅 Free/Pro/Max；Team/Enterprise 仅内部分享；含文件产物的对话 Free/Pro/Max 不能公开分享整段对话。〔官方〕9547008 / 12111783
- Free 套餐是否保留文件创建：官方当前页面写"所有套餐"，与 2025-09 上线初期"仅付费"的第三方报道（dev.to）不一致，以官方最新帮助页为准。〔第三方，历史信息〕https://dev.to/h1gbosn/inside-claudes-sandbox-what-happens-when-claudeai-creates-a-file-4gna

---

## 2. ChatGPT（Canvas / Code Interpreter / ChatGPT agent）

### 2.1 产物如何呈现
- **Canvas：独立侧板编辑器**（2024-10 发布）。"a new interface for working with ChatGPT on writing and coding projects that go beyond simple chat"——写作与代码在单独的画布面板中编辑，聊天退为旁白。〔官方〕https://openai.com/index/introducing-canvas/
- **Code Interpreter（现 Advanced Data Analysis）文件下载**：生成的文件放入沙箱 `/mnt/data`，以**可点击下载链接**直接出现在对话流中；多文件时常由模型自行打包 zip。〔第三方·官方论坛〕https://community.openai.com/t/failed-to-get-upload-status-for-mnt-data-filename/769410 ；〔第三方〕https://mitsloanedtech.mit.edu/ai/tools/how-to-use-chatgpts-advanced-data-analysis-feature/
- **ChatGPT agent**（2025-07-17 发布）：给 ChatGPT 一台虚拟计算机（浏览器+终端+文件），官方定位即"complete tasks like research, bookings, and **slideshows**"，产出幻灯片/表格等最终交付文件。〔官方〕https://openai.com/index/introducing-chatgpt-agent/ ；〔官方〕https://help.openai.com/en/articles/11794368-chatgpt-agent-release-notes

### 2.2 版本与迭代
- Canvas **自带后退按钮恢复历史版本**："You can also restore previous versions of your work by using the back button in canvas."〔官方〕https://openai.com/index/introducing-canvas/
- 2025-01 更新：Canvas 支持 o1（Pro/Plus/Team）、**HTML/React 代码实时渲染预览**（含 Free 用户）。〔官方·X 帖〕https://x.com/OpenAI/status/1882876172339757392 ；〔第三方〕https://the-decoder.com/openais-chatgpt-gets-a-major-canvas-upgrade-with-html-and-react-code-rendering/
- ChatGPT Release Notes 提到"Users can now share a Canvas asset such as rendered React/HTML code"（分享画布资产）。〔官方〕https://help.openai.com/en/articles/6825453-chatgpt-release-notes
- 无独立的 artifact 级"版本列表/回滚到指定版本/版本间 diff"面板（公开资料中未查到），版本能力弱于 Claude。此条为本文对比结论，非引用。

### 2.3 交付动作
- 对话内单文件下载链接（见上）；Canvas 可复制代码/文本。
- 分享：Canvas 资产可分享（渲染后的 React/HTML）。〔官方〕Release Notes 同上
- ChatGPT agent 产出文件可下载（如 PPTX/XLSX）。〔第三方〕https://www.entrepreneur.com/business-news/chatgpt-agent-creates-slide-decks-spreadsheets-from-prompts/494771 ；〔第三方·官方论坛〕https://community.openai.com/t/has-anyone-have-had-issues-with-chatgpt-being-able-to-deliver-excel-files-and-powerpoint-presentations/1116899
- **已知痛点（反面教材）**：沙箱下载链接会随会话过期失效，社区长期抱怨"download before X amount of time"；多文件输出需主动要求打包 zip。〔第三方〕https://www.linkedin.com/posts/kevingordonwpg_if-openai-could-fix-one-thing-about-chatgpt-activity-7361855126954807297-OGud

### 2.4 验收闭环
- Canvas 写作快捷项：**Suggest edits**（行内建议，用户逐条接受/拒绝）、Adjust the length、Change reading level、Add polish、Add emojis；编码快捷项：**Review code、Add logs、Add comments、Fix bugs、Port to a language**。选中文字即可让 AI 只改所选。〔官方〕https://openai.com/index/introducing-canvas/ ；〔第三方〕https://sdtimes.com/ai/chatgpt-canvas-offers-a-new-visual-interface-for-working-with-chatgpt-in-a-more-collaborative-way/
- ChatGPT agent 运行中用户可随时接管/打断（take control mid-task）。〔官方〕introducing-chatgpt-agent

### 2.5 多文件/工程式交付
- ChatGPT 主对话**没有**工程树/manifest 式交付组织（未查到）；多文件工程交给 Codex 云任务（见第 5 节）。
- 代码场景的 Canvas HTML/React 预览是单页级，不是项目级。

### 2.6 定价/限额
- ChatGPT agent 上线时：Pro 每月 400 条 agent 消息、Plus 40 条，可购额外额度。〔第三方〕https://belitsoft.com/news/chatgpt-agent-openai-20250717 （官方 FAQ 有对应页面，本轮未能直接抓取 help.openai.com 原文，未能逐字核对，谨慎引用）
- Code Interpreter / Canvas：无单独交付限额的官方数值（未查到）。

---

## 3. Kimi（OK Computer / Kimi Work / Kimi 办公能力）

### 3.1 产物如何呈现
- Moonshot 官网口径：「建站、智能制表及 PPT 自主编辑，让复杂工作化繁为简」，产品矩阵含网站/文档/PPT/表格/深度研究/Kimi Claw。〔官方〕https://www.moonshot.cn/
- **OK Computer（Agent 模式，2025-09 内测）**：独立虚拟电脑，20+ 工具（文件系统、浏览器、终端、代码、图像/音频生成），从调研到成品交付全程执行；交付物包括网站、交互式仪表板、演示文稿。〔第三方·实测〕https://www.qbitai.com/2025/09/337099.html ；〔第三方·内测公告转载〕https://zhuanlan.zhihu.com/p/1955612000806766449
- **Kimi Work（桌面端 Beta）**：核心三件套=工作空间（Workspace）+ Goal Mode（目标拆解）+ Agent Swarm（最多约 300 并行智能体，据第三方转述）；产物（PPT/Excel/Word）可**存回本地原文件夹**，研究过程、参考文件与输出成果被聚合管理。〔第三方〕https://baike.baidu.com/item/Kimi%20Work/67899418 ；〔第三方·实测〕https://www.ithome.com/0/991/986.htm ；〔第三方〕https://zhuanlan.zhihu.com/p/2045607946923647034
- **任务书中的「文件袋」提法：未查到**（官方无此命名，检索命中的均为无关内容）。最接近的官方概念是 Kimi Work 的「工作空间」聚合。

### 3.2 版本与迭代
- 未查到 artifact 级版本历史/回滚的公开资料。
- 交付后的迭代形态：**在线编辑 + 对话继续改**（PPT 生成后可在线编辑，也可下载可编辑 PPTX 而非静态 PDF）。〔第三方·内测公告转载〕https://zhuanlan.zhihu.com/p/1955612000806766449 ；〔第三方〕https://blog.csdn.net 相关实测（PPT 在线修改/本地修改）

### 3.3 交付动作
- 演示文稿：**下载可编辑 .pptx**（不只是静态 PDF）。〔第三方·内测公告转载〕同上
- Kimi Work：成果文件直接写回本地指定目录（桌面端本地交付，云端产品少见）。〔第三方〕百度百科/IT之家 同上
- 打包下载：公开资料着墨少，未查到明确的"多产物一键 zip"官方说明。

### 3.4 验收闭环
- Goal Mode：自然语言目标 → 自动拆解执行；多 Agent 并行、过程可视。细节的"用户批注→AI 继续加工"产品化形态未查到公开文档。

### 3.5 多文件/工程式交付
- Kimi Work 工作空间=过程+参考+成果的聚合容器（第三方描述）；未见 manifest/目录树的官方说明。

### 3.6 定价/限额
- 未查到公开的交付相关限额数值（OK Computer/Kimi Work 均为内测/灰度阶段，公开计费资料缺失）。

---

## 4. Manus（任务回放 / 交付 / Slides / Website Builder / Projects）

### 4.1 产物如何呈现
- 定位："autonomous general AI agent designed to complete tasks and **deliver results**"，运行在带持久文件系统的虚拟计算机沙箱。〔官方〕https://manus.im/docs/introduction/welcome.md （经 docs 索引转述）
- 任务页 = 左侧对话 + 右侧 Manus 计算机视图（第三方广泛描述；官方文档本轮未见逐字页面布局说明）。〔第三方〕https://workos.com/blog/introducing-manus-the-general-ai-agent
- **任务回放（Replay）**：完成的任务可生成分享链接，格式 `manus.im/share/{taskID}?replay=1`，**逐帧回放 agent 的完整执行过程**——官方 Wide Research 文档直接给出多个 replay 示例链接；第三方评价其建立信任、便于调试。〔官方〕https://manus.im/docs/features/wide-research.md ；〔第三方〕https://workos.com/blog/introducing-manus-the-general-ai-agent ；〔第三方〕https://www.willo.ai/blog/manus-review

### 4.2 版本与迭代
- 无文件级版本历史/回滚的官方说明（未查到）。
- 迭代靠**对话续作**（继续发消息改结果）+ **Plan Mode**（任务中途暂停、与用户对齐计划后继续）。〔官方〕https://manus.im/blog/manus-plan-mode
- Projects 的更新语义明确：**指令更新**下一条消息即生效；**文件更新只对新任务生效**，旧任务保持创建时配置。〔官方〕https://manus.im/docs/features/projects.md

### 4.3 交付动作
- Wide Research 型任务交付：可排序表格、带筛选的电子表格、结构化数据库（如 250 条详细档案）、可视化、批量高清 PNG、报告。〔官方〕wide-research.md
- **Manus Slides 四通道导出**：PowerPoint (.pptx，PowerPoint/Google Slides 可编辑)、PDF（打印分发）、**Web Slides（浏览器交互式放映）**、Speaker Notes Document（独立讲稿文件）；支持 **Import template** 上传自有 .pptx 模板（版式/配色/字体套用）；生成后可对话式改（Ask Manus）或下载后在任意演示软件中编辑；生成耗时 5–15 分钟（10 页约 5 分钟、40 页深度研报约 15 分钟）。〔官方〕https://manus.im/docs/features/slides.md
- **Website Builder**：发布到托管 URL、自定义域名、GitHub 集成、Access Control、Make a Copy（复制整站再改）、App Publishing（移动端）。〔官方〕docs 索引 https://manus.im/docs/llms.txt
- 知乎一手实测：任务结束直接**索取 PPT 下载文件**交付。〔第三方〕https://zhuanlan.zhihu.com/p/28257983768

### 4.4 验收闭环
- Plan Mode 中途对齐；replay 可逐步观看并（据第三方）中途介入纠偏；代码块上「Ask Manus」按钮、复制按钮等就地续问入口。〔官方〕plan-mode 博客 / wide-research.md；〔第三方〕willo.ai
- **Projects 自学习**：把有用对话转成"经用户批准的"Project 指令与文件更新，让每个任务改进下一个任务。〔官方〕https://manus.im/blog/manus-projects-self-updating

### 4.5 多文件/工程式交付
- Projects = 持久工作区：**master instruction + 知识库**自动注入每个新任务；任务可移入 Project（"think of projects as folders"）；置顶/拖拽排序/按 All tasks、Non-project tasks、Favorites、Scheduled 过滤；Project 私有，受邀者共享指令与知识库但只见自己的任务；全套餐可用、无创建数限制。〔官方〕projects.md

### 4.6 定价/限额
- Free（有限月度积分）/Pro（充足积分）/Team（共享积分池）；积分消耗按**任务复杂度与资源占用**而非时长，任务开始前 dashboard 显示预估；月度积分不滚存，加购包永不过期。交付/发布的具体数量限额未查到。〔官方〕https://manus.im/docs/introduction/plans.md

---

## 5. Devin / Cursor Cloud Agents / Codex——「交付 = PR/diff」的验收流

### 5.1 Devin
- **呈现**：会话工作区多标签——**Editor**（原 Code files）、**Browser**（会话预览）、终端实时流；**Changes 标签 = 持久文件树侧栏（树/平铺切换）+ 全宽 diff（带语言图标）**；**Worklog** 时间线按组显示 +N/−M diff 统计，点击打开对应范围 diff；**Tasks 标签**显示 Devin 当前任务清单；测试录屏渲染为带通过/失败摘要、播放速度、循环播放的富卡片。〔官方〕https://docs.devin.ai/release-notes/overview
- **PR 交付与验收**：会话内嵌 PR 视图（**Commits 标签逐提交浏览、Checks 标签常驻、diff 设置菜单、词级 diff 高亮、行 permalink**）；评审评论锚定到行（退化到文件级）；线程上有 **Ask Devin** 按钮追问；**Devin Review** 平台把大 PR 拆成组织化 diff，GitHub 式 Cancel/Comment/Start a review、merge bar 显示批准进度（已收/所需）、安全发现一键 **Auto-fix with Devin**；会话有 **Approve session** 状态与待读红点。〔官方〕同上 + https://docs.devin.ai/work-with-devin/devin-review
- **验收前置**：Ask Mode 只读探索与规划不改代码；计划/任务列表透明可见；Pre-Approve Testing 让 Devin 先自测。〔官方〕https://docs.devin.ai/get-started/first-run
- 定价（ACU 等限额）：本轮未查到官方数值页。

### 5.2 Cursor（Cloud Agents，原名 Background Agents）
- 云端 VM 独立环境运行，构建功能/修 bug/写测试，**完成后自动开 Pull Request**；入口：Cursor 应用、cursor.com/agents、Cloud Agents API。〔官方〕https://cursor.com/docs/cloud-agent ；〔官方〕https://cursor.com/help/ai-features/background-agents
- 已知边界：经 API 拉起的 agent 不能回帖 PR 评论（论坛故障帖侧面印证）。〔第三方·官方论坛〕https://forum.cursor.com/t/failed-to-create-a-github-pull-request-from-cursor-web/147959

### 5.3 OpenAI Codex（cloud tasks）
- 任务在一次性云端沙箱执行 → 产出 **diff** → 用户审阅 → 可"request further revisions（继续要求修改）"→ 满意后 **open a GitHub pull request** 或本地集成。〔官方〕https://openai.com/index/introducing-codex/
- 工作可从 GitHub/GitLab/Linear/Slack 发起，在 PR/MR/issue/线程内原地交接；PR 更新后需再 @codex 才动作。〔官方〕https://learn.chatgpt.com/docs/cloud ；〔第三方〕https://www.linkedin.com/posts/dkundel_codex-just-received-a-whole-lot-of-new-updates-activity-7366605790159228929-w6wy

---

## 6. Genspark / Flowith / MiniMax Agent——「成品即交付」的页面型 Agent

### 6.1 Genspark
- **呈现**：AI Slides 在聊天旁的画布上逐页构建（research→structure→design）；建前可设 Professional vs Creative、Guide Mode（AI 先与你确认再动手）、Standard vs Ultra（0.5×/1.0× 积分），**项目创建后模式不可改**。〔官方〕https://www.genspark.ai/helpcenter/ai-slides
- **迭代（本批调研中最细的批注体系）**：**Select 模式**点选元素描述改动；**Draw 模式**（Pen/Marker/Box）圈画批注入队，**Send N edits 一次性批量提交**；一键 AI Edit（Fix Layout / Polish Content）；**Edit 工具栏**进入画布直接拖拽/改字/裁图/管页（仅 Professional 模式，**手动编辑不耗积分**）；**Verify content** 对内容做联网/上传文件核验并把**核验轨迹写进导出 PPTX 的演讲者备注**。〔官方〕同上
- **交付动作**：导出 PDF / PPTX / Google Slides（Google Slides 导出限 Plus/Pro/Team/Enterprise；PPTX/PDF 导出需付费计划；导出不耗积分）；浏览器 **Present**（从头/从当前页 + 现场圈画）与 **Presenter view**（演讲者控制台：当前页/下一页/备注/计时器 + 干净的观众窗口）；分享 = 邮件邀请或 General Access「Anyone with the link」，可选 **Presentation share**（仅成品）或 **Project share**（含创建过程）。模板体系叫 **Skills**：100+ 内置、可上传 PPT/PDF 或 zip 保存为自己的（Save my presentation as a template / ⋮ → Save as Skill）、可链接/邮件分享、Team Skills 经管理员审批全员可用。〔官方〕同上
- AI Workspace 4.0：Slides/Sheets/Docs Agent 以原生插件嵌入 PowerPoint/Excel/Word。〔官方〕https://www.genspark.ai/zh-cn/blog/genspark-ai-workspace-4
- 冲突备注：2025-08 第三方横评称 Genspark"不提供生成链接功能，需导出 PPTX"；与官方帮助中心的公开链接分享说明矛盾，应以官方为准（横评可能过时或针对特定入口）。〔第三方〕https://www.53ai.com/news/neirongchuangzuo/2025082202691.html

### 6.2 Flowith
- 2D 画布多线程 + Oracle 智能调度 + Agent NEO（无限步骤/10M 上下文）；官方文档将 Neo 定位为"执行任务、灵活应变并**交付成果**的动态智能体"；配套「知识花园」知识库支持协作与分享。〔官方〕https://flowith.io/docs/zh/agent-neo/use-cases ；〔官方〕https://flowith.io/docs/zh/knowledge-garden/overview ；〔第三方〕https://news.aibase.com/tw/news/18206 ；〔第三方〕https://m.36kr.com/p/3306079334425864
- **产物呈现与交付动作的官方细节（版本/下载/分享）未查到更多公开文档**——只有"交付成果"的定性描述。

### 6.3 MiniMax Agent
- 官方技能市场有「演示文稿生成专家」：创建专业多页 **HTML-PPT**，**支持导出 PDF/PPTX**。〔官方〕https://agent.minimaxi.com/skills
- MiniMax Code 桌面端：对话 + 项目工作区 + 文件操作 + 终端 + 浏览器 + 技能 + 记忆 + 自动化任务。〔官方〕https://agent.minimaxi.com/docs/code/welcome
- 第三方实测：交付网页交互细节丰富；免费初始 1000 积分（积分制）。〔第三方〕https://zhuanlan.zhihu.com/p/1921988813783302203 ；〔第三方〕https://www.sohu.com/a/905267989_122082871

---

## 7. Lovable / v0 / bolt.new——成品预览 + 一键部署/导出的交付闭环（可选样本）

### 7.1 Lovable
- **Publish = 发布快照到可分享 URL**（Lovable 托管），编辑继续在项目上进行、发布的是快照。〔官方〕https://docs.lovable.dev/features/publish
- **代码所有权**：代码编辑器中浏览/搜索/手动改、**在对话中引用精确行**（reference exact lines in chat）、**一键下载整个代码库（download your codebase）**；GitHub 双向同步（GitHub.com/企业版）。〔官方〕https://docs.lovable.dev/features/code-mode ；〔官方〕https://docs.lovable.dev/integrations/github
- **协作与再创作**：Share 弹窗支持工作区成员/外部协作者/任何拿到链接的人；**Remix**（以项目当前状态为副本新建，原件保留）+ 项目设置里 Public remixing 开关；Plan mode 先规划后写码。〔官方〕https://docs.lovable.dev/features/share-project ；https://docs.lovable.dev/features/projects/remix ；https://docs.lovable.dev/features/plan-mode
- Dev Mode（编辑任意代码）面向付费用户灰度。〔第三方·官方账号在 LinkedIn 的公告〕https://www.linkedin.com/posts/lovable-dev_introducing-lovables-dev-mode-to-early-access-activity-7303460125346820096-Xkft

### 7.2 v0（Vercel）
- 导出路径：**Download ZIP**（项目整包，changelog 有其修复记录）、**Add to Codebase**（npx CLI 写入本地工程）、同步 GitHub、Deploy 到 Vercel；组件级 Copy Code。〔官方〕https://v0.app/changelog ；〔第三方·Vercel 官方社区〕https://community.vercel.com/t/become-a-v0-expert/5981 ；〔第三方〕https://www.reddit.com/r/nextjs/comments/1fzs4ji/how_can_i_import_the_project_i_made_from_v0dev_to/
- "Export to CodeSandbox" 为旧文档提法，现行文档未见，仅 Download ZIP / Add to Codebase / GitHub / Deploy。（本文对比结论，来源同上）

### 7.3 bolt.new
- 官方帮助中心：项目可 **Export → Download ZIP** 用本地编辑器继续；托管可选 **Bolt hosting 或接入 Netlify**。〔官方〕https://support.bolt.new/building/using-bolt/projects-files ；〔官方〕https://support.bolt.new/integrations/netlify
- 限额：免费档 token 约 150K/天（第三方转述，未查到官方数值页）。〔第三方〕https://opsily.com/blog/how-to-deploy-a-bolt-new-app

---

## 8. 可借鉴模式清单（对 gaea 办公板块）

> 筛选原则：剔除 gaea 已有能力（交付文件卡、版本时间线/恢复、一键 zip、证据链、reveal、docx 框选即改+修订、xlsx Plan→Apply）。量级：小=天级/纯前端+少量后端；中=1–2 周级/新链路；大=月级/新子系统。仅为估算。

| # | 模式名 | 出处（验证产品） | 对 gaea 办公板块的适配点 | 量级 |
|---|--------|------------------|--------------------------|------|
| 1 | 分享版本选择器 + 「始终分享最新」双语义 | Claude Code Artifacts 分享菜单（§1.4） | 右栏产物面板增加「生成分享包/分享链接」时可选：锁定某版本快照 or 跟随最新；gaea 本地优先可落地为"导出自包含 HTML/只读包 + 版本指纹" | 小 |
| 2 | 成品快照与工作副本分离 | Lovable Publish 快照（§7.1）、Manus Make a Copy（§4.3） | 交付/导出走"快照"，后续编辑不污染已交付版本；登记表为每个交付物记录"已交付快照 ID" | 小 |
| 3 | 跨会话产物库（发布制聚合） | Claude Artifacts 侧边栏（§1.1）、Kimi Work 工作空间（§3.1） | gaea 右栏是会话级；新增跨会话「产物库」页，按项目/任务聚合所有登记产物，支持筛选与搜索 | 中 |
| 4 | 版本间差异视图（compare，不只是回滚） | Claude Cowork 版本历史 compare（§1.2）、Devin Changes 词级 diff（§5.1） | 产物时间线两版本间提供 diff：md/text 文本 diff、xlsx 单元格级变更表、docx 修订汇总；恢复前先看差异 | 中 |
| 5 | 框选/圈选批注 → 批量修改队列（Send N edits） | Genspark Draw 模式批注入队（§6.1）、Claude 多文件排队一次性应用（§1.2） | docx/xlsx 预览内多次框选/圈选形成"修改清单"，一次提交给 agent 批量处理，逐条在修订记录中可追溯 | 中 |
| 6 | 批注线程 + @agent 激活回流 | Claude Code Artifacts 评论线程 Send to Claude/@claude（§1.4） | 预览面板内批注可"发送给助手"成为任务消息，处理结果回写为该产物新版本；未激活批注不进上下文 | 中 |
| 7 | 内容核验轨迹写入交付文件本体 | Genspark Verify content 写入 PPTX 备注（§6.1） | 与 gaea 证据链天然契合：导出 docx/xlsx 时把证据引用写入文档批注/页脚/xlsx 备注列，交付物自带出处 | 中 |
| 8 | 任务回放（可分享的步骤级 replay） | Manus `?replay=1` 分享回放（§4.1） | 把证据链升级为"可回放时间线"：工具调用/文件变更按步回放；本地优先可导出回放包给他人审阅 | 大 |
| 9 | 计划暂停确认（Plan Mode）与验收状态机 | Manus Plan Mode（§4.2）、Devin Approve session（§5.1）、Codex request revisions（§5.3） | 任务级「验收检查单 + 确认/要求修改」状态；确认后才算交付闭环，要求修改则回到 agent 继续加工 | 中 |
| 10 | PR 式 diff 验收（词级 diff + 行锚定评论 + 一键追问） | Devin PR 视图/Devin Review（§5.1）、Codex diff→PR（§5.3） | 办公文档非代码，但"逐行/逐格 diff + 在 diff 行上批注 + 批注即追问"可平移到 xlsx/docx 的 Plan→Apply 审阅 | 大 |
| 11 | 多通道导出矩阵 | Manus Slides 四通道 pptx/pdf/Web Slides/讲稿（§4.3）、Genspark 三格式（§6.1） | 交付卡增加「导出为」：md→docx/pdf、docx→pdf、xlsx 图表页→png/pdf；Web Slides 式 HTML 放映页 | 中 |
| 12 | 成品即演示（Present/演讲者视图） | Genspark Present + Presenter view（§6.1）、Manus Web Slides（§4.3） | md/html/PPT 类产物一键进入演示模式（全屏放映、备注、观众独立视图），交付即汇报 | 小-中 |
| 13 | Remix / 以产物为种子新建任务 | Claude Copy 流程（§1.3）、Lovable Remix（§7.1） | 产物右键「以此产物为输入新建任务」，预填上下文与说明；原产物冻结不动 | 小 |
| 14 | 交付目的地矩阵（不止下载） | Claude 存 Google Drive（§1.1）、Lovable GitHub 双向同步（§7.1）、bolt → Netlify（§7.3）、MiniMax HTML-PPT 导出（§6.3） | 交付动作扩展：另存到本地指定目录、用系统默认程序打开、导出到 WPS/Office 可编辑格式、复制为纯文本/Markdown | 小-中 |
| 15 | 产物失效检测徽标（时效反面教材） | ChatGPT /mnt/data 下载链接会话过期痛点（§2.3） | gaea 已有"未生成"徽标；补"路径失效/文件被移动"检测徽标与一键重生成，规避竞品已踩坑 | 小 |
| 16 | 工程式交付组织（文件树 + 入口 + manifest） | Devin Changes 文件树（§5.1）、Manus Projects 知识库+指令注入（§4.5）、Lovable 代码库下载（§7.1） | 多产物任务在右栏聚合为「产物包」：目录树视图 + 入口文件标识 + 包级 zip（已有 zip，补树与入口语义） | 中 |
| 17 | 模板/Skills 化交付（导入模板 + 存为模板） | Manus Import template（§4.3）、Genspark Save as Skill（§6.1） | 交付遵循用户自带模板（docx/xlsx/pptx 模板套用）；满意成品可"存为我的模板"复用 | 中 |
| 18 | 沙箱产物持久化承诺（对比项） | Claude「文件在整段对话内持续可下载」+30MB 明示限额（§1.6） | 在交付卡上明示保留策略（本地永久/可清理），把"什么时候会失效"写进 UI，建立交付信任 | 小 |

---

## 9. 参考 URL 全表

### 官方来源
| # | URL | 用于 |
|---|-----|------|
| 1 | https://support.claude.com/en/articles/9487310-what-are-artifacts-and-how-do-i-use-them | Claude Artifacts 呈现/版本/下载/库 |
| 2 | https://support.claude.com/en/articles/12111783-create-and-edit-files-with-claude | Claude 文件创建/下载/30MB/套餐 |
| 3 | https://support.claude.com/en/articles/9547008-publish-and-share-artifacts | Claude 发布/分享/Remix/组织内分享 |
| 4 | https://support.claude.com/en/articles/14729249-use-live-artifacts-in-claude-cowork | Live Artifacts 版本历史（检索摘要转述） |
| 5 | https://code.claude.com/docs/en/artifacts | Claude Code Artifacts 版本/分享/评论/约束 |
| 6 | https://claude.com/blog/artifacts-in-claude-code | Claude Code Artifacts 公告 |
| 7 | https://openai.com/index/introducing-canvas/ | Canvas 发布/快捷项/后退恢复版本 |
| 8 | https://x.com/OpenAI/status/1882876172339757392 | Canvas 更新（o1、HTML/React 渲染） |
| 9 | https://help.openai.com/en/articles/6825453-chatgpt-release-notes | Canvas 资产分享等 |
| 10 | https://openai.com/index/introducing-chatgpt-agent/ | ChatGPT agent 发布 |
| 11 | https://help.openai.com/en/articles/11794368-chatgpt-agent-release-notes | ChatGPT agent 发布节奏 |
| 12 | https://openai.com/index/introducing-codex/ | Codex diff→PR 交付流 |
| 13 | https://learn.chatgpt.com/docs/cloud | Codex cloud 集成 |
| 14 | https://manus.im/docs/llms.txt | Manus 文档索引 |
| 15 | https://manus.im/docs/features/wide-research.md | Manus 交付形态/replay 链接 |
| 16 | https://manus.im/docs/features/slides.md | Manus Slides 导出/模板 |
| 17 | https://manus.im/docs/features/projects.md | Manus Projects 组织/更新语义 |
| 18 | https://manus.im/docs/introduction/plans.md | Manus 套餐/积分 |
| 19 | https://manus.im/blog/manus-plan-mode | Manus Plan Mode |
| 20 | https://manus.im/blog/manus-projects-self-updating | Manus 项目自学习 |
| 21 | https://docs.devin.ai/release-notes/overview | Devin 会话/PR 视图/Worklog/Changes |
| 22 | https://docs.devin.ai/work-with-devin/devin-review | Devin Review |
| 23 | https://docs.devin.ai/get-started/first-run | Devin Ask Mode/首次会话 |
| 24 | https://cursor.com/docs/cloud-agent | Cursor Cloud Agents |
| 25 | https://cursor.com/help/ai-features/background-agents | Cursor 自动开 PR |
| 26 | https://www.genspark.ai/helpcenter/ai-slides | Genspark AI Slides 全流程 |
| 27 | https://www.genspark.ai/zh-cn/blog/genspark-ai-workspace-4 | Genspark Office 插件 |
| 28 | https://flowith.io/docs/zh/agent-neo/use-cases | Flowith Neo 交付定位 |
| 29 | https://flowith.io/docs/zh/knowledge-garden/overview | Flowith 知识花园 |
| 30 | https://agent.minimaxi.com/skills | MiniMax HTML-PPT 技能/导出 |
| 31 | https://agent.minimaxi.com/docs/code/welcome | MiniMax Code 桌面端 |
| 32 | https://docs.lovable.dev/features/publish | Lovable 发布快照 |
| 33 | https://docs.lovable.dev/features/code-mode | Lovable 代码编辑/整库下载 |
| 34 | https://docs.lovable.dev/integrations/github | Lovable GitHub 同步 |
| 35 | https://docs.lovable.dev/features/share-project | Lovable 协作分享 |
| 36 | https://docs.lovable.dev/features/projects/remix | Lovable Remix |
| 37 | https://docs.lovable.dev/features/plan-mode | Lovable Plan mode |
| 38 | https://v0.app/changelog | v0 Download ZIP 等 |
| 39 | https://support.bolt.new/building/using-bolt/projects-files | bolt ZIP 导出 |
| 40 | https://support.bolt.new/integrations/netlify | bolt 托管/Netlify |
| 41 | https://www.moonshot.cn/ | Kimi 官方产品口径 |

### 第三方来源（补充/评测）
| # | URL | 用于 |
|---|-----|------|
| 42 | https://mitsloanedtech.mit.edu/ai/tools/how-to-use-chatgpts-advanced-data-analysis-feature/ | ADA 文件下载 |
| 43 | https://community.openai.com/t/failed-to-get-upload-status-for-mnt-data-filename/769410 | /mnt/data 与下载链接 |
| 44 | https://www.linkedin.com/posts/kevingordonwpg_if-openai-could-fix-one-thing-about-chatgpt-activity-7361855126954807297-OGud | 下载链接过期痛点 |
| 45 | https://www.entrepreneur.com/business-news/chatgpt-agent-creates-slide-decks-spreadsheets-from-prompts/494771 | agent 产出 PPT/表格 |
| 46 | https://community.openai.com/t/has-anyone-have-had-issues-with-chatgpt-being-able-to-deliver-excel-files-and-powerpoint-presentations/1116899 | agent 文件交付问题 |
| 47 | https://belitsoft.com/news/chatgpt-agent-openai-20250717 | agent 限额 40/400 |
| 48 | https://the-decoder.com/openais-chatgpt-gets-a-major-canvas-upgrade-with-html-and-react-code-rendering/ | Canvas 渲染更新 |
| 49 | https://sdtimes.com/ai/chatgpt-canvas-offers-a-new-visual-interface-for-working-with-chatgpt-in-a-more-collaborative-way/ | Canvas 快捷项清单 |
| 50 | https://www.qbitai.com/2025/09/337099.html | Kimi OK Computer 实测 |
| 51 | https://zhuanlan.zhihu.com/p/1955612000806766449 | OK Computer 内测公告（转引） |
| 52 | https://baike.baidu.com/item/Kimi%20Work/67899418 | Kimi Work 工作空间 |
| 53 | https://www.ithome.com/0/991/986.htm | Kimi Work 实测 |
| 54 | https://zhuanlan.zhihu.com/p/2045607946923647034 | Kimi Work Beta 介绍 |
| 55 | https://workos.com/blog/introducing-manus-the-general-ai-agent | Manus replay 会话 |
| 56 | https://www.willo.ai/blog/manus-review | Manus 回放/介入 |
| 57 | https://zhuanlan.zhihu.com/p/28257983768 | Manus 一手实测（索取 PPT 文件） |
| 58 | https://www.53ai.com/news/neirongchuangzuo/2025082202691.html | 11 款 PPT 横评（冲突备注） |
| 59 | https://news.aibase.com/tw/news/18206 | Flowith NEO |
| 60 | https://m.36kr.com/p/3306079334425864 | Flowith 知识花园 |
| 61 | https://zhuanlan.zhihu.com/p/1921988813783302203 | MiniMax 网页交付评测 |
| 62 | https://www.sohu.com/a/905267989_122082871 | MiniMax 积分体验 |
| 63 | https://community.vercel.com/t/become-a-v0-expert/5981 | v0 导出路径 |
| 64 | https://www.reddit.com/r/nextjs/comments/1fzs4ji/how_can_i_import_the_project_i_made_from_v0dev_to/ | v0 迁移 |
| 65 | https://www.linkedin.com/posts/lovable-dev_introducing-lovables-dev-mode-to-early-access-activity-7303460125346820096-Xkft | Lovable Dev Mode |
| 66 | https://opsily.com/blog/how-to-deploy-a-bolt-new-app | bolt token 限额（未官方证实） |
| 67 | https://dev.to/h1gbosn/inside-claudes-sandbox-what-happens-when-claudeai-creates-a-file-4gna | Claude 沙箱历史信息 |
| 68 | https://forum.cursor.com/t/failed-to-create-a-github-pull-request-from-cursor-web/147959 | Cursor Cloud Agent 边界 |
| 69 | https://www.linkedin.com/posts/dkundel_codex-just-received-a-whole-lot-of-new-updates-activity-7366605790159228929-w6wy | Codex 更新/@codex |

### 检索但未证实/未查到清单（防编造声明）
- Claude「Refs」作为官方功能名：未查到官方条目（已用官方可验证的 Edit with Claude / Try fixing with Claude 替代描述）。
- Kimi「文件袋」：未查到该官方命名。
- Kimi 交付限额、ChatGPT Canvas 官方 FAQ 原文（help.openai.com 拒绝抓取 403）、Devin ACU 定价数值、Manus 交付数量限额、bolt token 限额官方数值：均未查到。
