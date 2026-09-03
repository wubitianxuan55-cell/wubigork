# 调研：办公文档场景中「AI 生成成品之后」的交付后链路

- 日期：2026-09-03
- 调研性质：原始稿（raw draft），仅供内部参考，未改任何代码
- 主题：竞品在「AI 成稿」到「用户真正能把成品用出去」之间的模式——格式保真导出、交付流转、模板化复用、交付确认、批量/工程化交付
- gaea 对照现状：md/Word/PDF 三出口、docx 修订制写入、xlsx 原生图表 + CrossEmbed 嵌 Word/PPT、产物版本时间线 + 恢复、zip 打包
- 方法说明：优先官方文档/博客/help center，第三方评测补充并标注。查不到的写「未查到」。飞书/钉钉部分页面为 JS 动态渲染无法直接抓取正文，结论基于检索摘要中引用的原文片段，已注明。

---

## 1. Microsoft 365 Copilot（Word / Loop）

### 1.1 成稿后的格式保真导出
- 「Edit with Copilot」（原名 Agent Mode）的核心承诺是**在原文档内就地创作、编辑、排版，使用 Word 内建样式与功能**，而不是另存副本。官方支持文档明确：Copilot "create, edit, refine, and format content in place—using Word's built-in styles and features"。这从产品机制上把「AI 写完→导出」的格式断层问题内消掉了：因为写的就是 Word 本体，导出链路沿用 Word 原生导出。来源：https://support.microsoft.com/en-us/word/edit-with-copilot-in-word
- 但「Copilot 聊天窗生成的稿子不是真正的 Word 文件」是高发差评：用户反馈从 chat 窗口复制进模板后格式全丢（第三方社区，r/Office365：https://www.reddit.com/r/Office365/comments/1o0lfuo/copilot_output_loses_formatting_when_copying_into/ ）；「为什么我们就是做不出带正确格式的 Word 文档」是近期集中吐槽（r/microsoft_365_copilot：https://www.reddit.com/r/microsoft_365_copilot/comments/1qvuef4/why_are_we_struggling_to_create_word_documents/ ）；也有用户反映 Copilot 会把标题样式等格式写丢（第三方博客 Gitbit，作者称已向微软报障：https://www.gitbit.org/docs/copilot-in-word-document-0kz9yhme ）。微软 Q&A 上还有专业用户抱怨「Edit with Copilot 只能改当前打开的文档，不能新建文件」（https://learn.microsoft.com/en-us/answers/questions/5698940/document-editing 与 https://learn.microsoft.com/en-us/answers/questions/5858106/copilot-platform-changes-no-longer-meet-profession ）。
- Copilot Pro（个人版）用户投诉「付费就是为了导出 PDF，结果导不了」，社区给的 workaround 是回到 Word 应用内用 Copilot 而不是用独立 Copilot chat（r/CopilotPro：https://www.reddit.com/r/CopilotPro/comments/1mmoz7r/i_paid_for_pro_so_i_could_export_to_pdf_but_its/ ）。

### 1.2 交付流转（审批/发送/归档/版本）
- 修订制闭环（2026 年重点更新）：面向法务/金融等高强度审阅场景，微软宣布 Copilot in Word 支持**词级精度（word-level precision）的 Track Changes**——Copilot 的改动默认可见，可从 Copilot 侧直接打开 Track Changes 获得完整审计轨迹；同时支持**锚定到具体文字的上下文批注**、目录（TOC）插入/更新、页眉页脚管理、长任务进度消息。来源：Microsoft 365 Copilot Blog《Copilot in Word: New Capabilities for Document Workflows》 https://techcommunity.microsoft.com/blog/microsoft365copilotblog/copilot-in-word-new-capabilities-for-document-workflows/4508974 （检索时该站维护中，要点转引自检索结果与转载：https://www.linkedin.com/posts/fsistrategies_copilot-in-word-new-capabilities-for-document-activity-7452726728847945729-xiS6 ）；支持文档确认："Edit with Copilot will respect Track Changes, and if Track Changes is enabled then Edit with Copilot changes will be tracked in the document"（https://support.microsoft.com/en-us/word/edit-with-copilot-in-word ）。Agentic 能力（Word/Excel/PPT agent）于 2026-04-22 GA：https://www.microsoft.com/en-us/microsoft-365/blog/2026/04/22/copilots-agentic-capabilities-in-word-excel-and-powerpoint-are-generally-available/
- 边界与诚实条款（官方明示的限制）：Copilot 既不能开关 Track Changes 也不能替你接受/拒绝修订；默认直接改文档，靠撤销/历史版本兜底；**共享文档中必须先在聊天里预览改动、人工确认后才落盘**；不能生成/插入图片；不能新增文件；改写锚定段落时可能连带删除其上批注。来源：https://support.microsoft.com/en-us/word/edit-with-copilot-in-word
- 版本沉淀：依托 Word/SharePoint 原生版本历史（"undo / view prior versions"），无独立产物时间线产品。

### 1.3 模板化复用
- 官方文档未提供「把这份 AI 成稿结构存为模板」的一键功能（未查到对应官方能力）。
- 生态做法是用「/」引用既有文档作为素材生成新稿（"draft from a prompt, outline, notes, or referenced file"，https://support.microsoft.com/en-us/word/welcome-to-copilot-in-word ），本质是「拿上一份成品当 few-shot 素材」，比模板弱一档。

### 1.4 Loop 组件跨文档复用
- Loop components 是「可移植的内容片段，在所有被分享处保持同步」：在 Word 网页版、Teams、Outlook、Loop 应用里同一份表格/清单/段落实时同步，编辑一次处处更新（官方：https://support.microsoft.com/en-us/loop/get-started-with-microsoft-loop ；Outlook 场景：https://support.microsoft.com/en-us/loop/use-loop-components-in-outlook ）。从 Teams 分享到 Outlook 的方式是复制组件链接粘贴进邮件，仍是活的组件（微软 Q&A：https://learn.microsoft.com/en-us/answers/questions/5881900/how-do-i-use-loop-in-teams-and-share-a-component-f ）。
- 第三方治理分析指出组件仅在 Word for web/Teams/Outlook/Loop 内同步（桌面版 Word 不行），且有权限与合规治理成本（Rencore：https://rencore.com/en/blog/microsoft-loop-bridging-gap-seamless-team-collaboration/ ；Apps4.Pro 完整指南：https://blog.apps4.pro/microsoft-loop-features-use-cases-governance ）。社区对其真实用例存在疑问（r/microsoft365：https://www.reddit.com/r/microsoft365/comments/168llxe/loop_component_actual_use_case/ ）。

### 1.5 批量/工程化交付
- 应用内 Copilot 明确不连外部工具、不做定时批处理（"does not connect to external tools"，https://support.microsoft.com/en-us/word/welcome-to-copilot-in-word ）。批量交付靠 Microsoft 365 生态外的 Power Automate / Graph API（本次未逐条核查，记为待补）。

---

## 2. Gemini in Google Workspace（Docs / Sheets）

### 2.1 成稿后的格式保真导出
- 侧栏/底栏/行内三种入口，成稿后仍是标准 Google Doc，导出走 Docs 原生 File→Download（docx/pdf/odt 等）。官方能力总览：https://support.google.com/a/users/answer/15146419?hl=en ；Workspace Gemini for Docs 产品页：https://workspace.google.com/products/docs/ai/
- Gemini 应用侧新增「生成文件」能力：Canvas 里直接产出 PDF/Word/Excel/Docs/Sheets/Slides，可下载或导出到 Drive（Google 官方博客：https://blog.google/innovation-and-ai/products/gemini-app/generate-files-in-gemini/ ）。
- 格式保真口碑是老问题：社区共识「在 Google Docs、ODT、DOCX 之间搬运不可能保持完全一致的格式」（r/gsuite：https://www.reddit.com/r/gsuite/comments/13oem44/how_do_i_convert_google_docs_to_word_without/ ）；目录刷新后字体跑偏是长期 bug 类吐槽（Google 支持社区：https://support.google.com/docs/thread/251117694/table-of-contents-is-changing-fonts-when-it-s-refreshed?hl=en 与 https://support.google.com/docs/thread/10229737/font-style-on-table-of-contents-doesn-t-match-text-in-the-body?hl=en ）；反向（Word→Docs）同样破格式（Microsoft Learn Q&A：https://learn.microsoft.com/en-us/answers/questions/5339252/formatting-changes-between-ms-word-and-googledocs ）。

### 2.2 交付流转
- 流转依赖 Google 生态原生能力：评论/@ 提醒/Drive 共享/审批第三方补齐，Gemini 本身不做审批流。第三方分析明确指出「Gemini in Workspace 局限在文档内任务，不能跑自动化工作流、不能连外部工具、不能定时运行」（MindStudio：https://www.mindstudio.ai/blog/gemini-in-google-docs-sheets-slides-what-you-can-do ）。

### 2.3 模板化复用（本节亮点）
- **Match doc format**：官方支持「用一个既有 Google Doc 作为模板」，让新内容沿用该文档的 layout、style、structure；另有 **Match writing style** 沿用某 Drive 文档的文风；可用「个性化指令」固化角色/语气/输出格式（官方：https://support.google.com/a/users/answer/15146419?hl=en ）。这是「上次成稿=下次模板」模式最直接的官方实现。
- 个性化指令当前仅美区英语可用（同上官方页）。

### 2.4 交付确认
- AI 产出先在侧栏呈现，用户满意后插入/替换正文；确认机制即 Docs 原生协作（评论+建议模式），AI 不以修订身份写稿。官方页描述为通过侧栏、底栏或行内起草与润色（https://support.google.com/a/users/answer/15146419?hl=en ）。未查到「批注自动回流给 AI」的官方能力。

### 2.5 批量/工程化交付
- 应用内无定时/批量（见 2.2 第三方分析）；工程化走 Gemini API / Apps Script（本次未展开核查，记为待补）。
- 用户侧摩擦实例：把 Gemini 对话转成 PDF/Docs 的流程仍有阻力（Google 支持社区：https://support.google.com/gemini/thread/424706282/how-to-export-gemini-chat-to-pdf-google-docs?hl=en ）。

---

## 3. Notion AI（生成页 → 数据库/模板 → 分享发布）

### 3.1 成稿后的格式保真导出
- 官方导出支持 PDF/HTML/Markdown/CSV（整 workspace 也可导出）：https://www.notion.com/help/export-your-content
- 口碑差：PDF 导出被评「AWFUL」，社区用 Pandoc+LaTeX 自救（https://techresolve.blog/2025/12/23/notion-devs-do-you-plan-to-fix-the-awful-pdf-expo/ ）；Markdown 导出会丢 callout、数据库、嵌套页面、图片（第三方拆解：https://unmarkdown.com/blog/notion-export-broken ）；Notion 原生没有 docx 导出，Word 需经 Markdown/HTML 转换工具（BloggingX：https://bloggingx.com/export-notion-to-word/ ）；社区指导用第三方导出器保 PDF 排版（r/Notion：https://www.reddit.com/r/Notion/comments/mo8hm2/guide_how_to_properly_export_notion_to_pdf/ ）。官方在导入文档里也承认复杂样式会有问题、建议先简化（https://www.notion.com/help/import-data-into-notion ）。

### 3.2 交付流转
- 流转即协作：页面共享、评论、数据库视图、权限体系。审批流依赖第三方（未查到官方审批能力）。

### 3.3 模板化复用（本节亮点）
- **页面→数据库**：Notion AI 支持「Make this page a database」，把既有页面（乃至 Excel）转成数据库并自动配置属性（第三方教程：https://kurashi-notion.com/en/blogs/notion/notionai-database2 ；官方 autofill 文档同时确认「Notion AI 能新建数据库但不能编辑既有数据库」：https://www.notion.com/help/autofill ）。
- **数据库模板**：数据库内 New 下拉 → New template，把成稿结构固化为可重复生成的模板（官方：https://www.notion.com/help/database-templates ）；**Repeating database templates** 支持按周期自动生成新条目（官方指南：https://www.notion.com/help/guides/automate-work-repeating-database-templates ）；Button 块一键按模板建页（官方指南：https://www.notion.com/help/guides/automatically-generate-blocks-pages-with-buttons ）。
- **AI Autofill**：新条目入店后自动填充摘要/标签/翻译等属性，等于「AI 持续为复用结构补元数据」（官方：https://www.notion.com/help/autofill ；第三方解读：https://www.eesel.ai/blog/notion-ai-autofill ）。

### 3.4 交付确认
- 确认走评论区人工流转；未查到批注回流 AI 的官方闭环。

### 3.5 分享发布
- Share → Share to web 一键把页面发布为 notion.site 链接，任何有链接者可看（官方博客：https://www.notion.com/blog/personalize-public-pages ；移动端说明：https://www.notion.com/help/workspaces-on-mobile ；安全说明：https://www.notion.com/help/security-and-privacy ）。第三方生态把它当建站/帮组中心底座（HelpKit：https://www.helpkit.so/blog/how-to-use-notion-to-create-a-help-center-for-your-company ）。

### 3.6 批量/工程化交付
- Notion Agent/Skills（把重复指令存成技能）+ API + n8n/Make 集成是社区主流玩法（如「Turn Any Notion Page Into An AI Skill」教程：https://www.youtube.com/watch?v=jKs6HA9UlNg ）；官方 API 批量写入本次未展开核查，记为待补。

---

## 4. 飞书智能伙伴（Aily / 豆包工作伙伴）与钉钉 AI

> 注：飞书两篇官方手册页为 JS 渲染，直接抓取仅得标题；以下引文来自搜索引擎返回的原文片段，均已给出官方 URL 供复核。

### 4.1 飞书智能伙伴：成稿→交付→验收
- 官方功能手册把「豆包工作伙伴」定位为「一位 AI 同事」：在现有权限和工作流内接消息、文档、会议、任务与业务系统，支持「在飞书中发起互动任务、查看进展并**验收结果**」以及「**经验复用**」（https://aily.feishu.cn/hc/1u7kleqg/8u02e8ub ）。「专属工作伙伴」常驻联系人，可交付日报、总结、文档、PPT、数据报告，并支持拉群协作（https://aily.feishu.cn/hc/1u7kleqg/jsmwydi4 ）。
- 成稿能力：会中速览要点、会后**自动生成智能会议纪要文档，所有参会人可查看**（飞书官方说明书：https://www.feishu.cn/hc/zh-CN/articles/318505042260 ）。
- **Workflow 技能**可对大模型能力做多节点编排，实现「文档生成、审批流自动化」等复杂交付场景（https://www.feishu.cn/content/8u02e8ub ）；「工作流应用」面向 AI 审批、AI 质检等场景做图形化低代码流程（https://aily.feishu.cn/hc/1u7kleqg/owh6kvac ）；「Aily 之飞书消息」提供向用户/群聊发消息的节点（https://www.feishu.cn/content/mtb6n3ah ）；群组触发器支持机器人进群自动总结群消息等事件驱动（https://www.feishu.cn/content/sj7drxsj ）。
- 即：**「生成→审批→群推送→验收」全链路在同一 App 内闭环**，是中国办公语境下「交付即流转」的代表。
- 独立第三方深度横评较少（见 WPS 一节引用的官方选型文章仅为金山自家视角），差评情报本次未查到成规模的公开吐槽聚合。

### 4.2 钉钉 AI：审批 + 定时群推送
- 审批 AI 助理：智能识别图片/文档关键信息→自动生成审批表单→自动分类（请假/报销）→一键批量审批（钉钉官方推介页：https://www.dingtalk.com/qidian/page-UkDqFcGj.html 、https://www.dingtalk.com/qidian/page-Q0MQ5YoW.html 、https://www.dingtalk.com/qidian/page-idbENeWS.html ）。
- 自动化流转：开放平台提供「定时触发（单次/周期/Cron）→ 机器人向群发消息」的官方流程模板（工作日定时收日报：https://open.dingtalk.com/document/connection/robot-weekdays-collect ；创建自动化流程：https://open.dingtalk.com/document/connection/automated-process-usage-guide ）；AI 助理可感知群内消息主动服务（https://open.dingtalk.com/document/ai-dev/group-message-aware-trigger ）；多维表更新后可自动把通知/文档链接推到群（钉钉 AI 表格介绍：https://www.dingtalk.com/qidian/page-xDECjiz8.html ）。
- 即：**「数据/表格变化→AI 生成→群机器人定时/事件推送」**是钉钉侧的交付主线，审批与 IM 天然同仓。
- 2026 年背景：阿里对全产品线下达 AI 改造考核，钉钉审批/会议/文档/日程全面接入 AI（新浪财经报道：https://finance.sina.cn/roll/2026-06-09/detail-iniavshm4879661.d.html ）。

---

## 5. WPS AI / 金山办公

### 5.1 成稿后的格式保真导出
- 定位差异：WPS AI 3.0（灵犀）自称「**原生 Office 智能体**」——AI 写的就是 WPS 文字/表格/演示本体，导出 docx/xlsx/pptx/pdf 走本地套件原生链路，格式断层天然最小（新华网 WAIC 2025 报道 WPS AI 3.0 发布：search snippet 见 https://lingxi.wps.cn/ 与新华网报道，转引自检索结果；官方入口 https://www.wps.ai/zh-hk/ 、https://lingxi.wps.cn/ ）。
- AI Writer：在 Word 内直接生成草稿、打磨语气，一键摘要长文档（官方：https://www.wps.ai/zh-hk/ ）。
- 官方选型文章（金山自家，需打折）对比 WPS AI/飞书/钉钉差异，称灵犀四个 AI 助手中 AI 写作助手支持零起草稿、PDF 对话、提取数据点、总结长报告（https://www.wps.cn/article/fdRFDBqM.html ）；Canvas 双屏协同（左侧文档右侧 AI 面板，划选后自然语言下指令）是 3.0 主打（知乎分析，转引自检索：https://zhuanlan.zhihu.com 题为《当 AI 办公过了概念验证期：WPS 灵犀的四个场景与技术纵深》）。
- 模板生态：10 万+ 免费 Word/PPT/Excel/PDF 模板，宣称「排版锁定保真度」与无水印导出（WPS 官方页：https://zh-hant.wps.com/feature/zh-CN-free-template-editing-app/ ）。
- Word→PPT 直转：官方教程称无需导出再导入，写完文档直接唤醒 AI 分析逻辑、梳理大纲、套模板出 PPT（https://www.wps.cn/article/ban-gong-xiao-lv-yan-shi-wen-gao-2026-IEMmSmwl.html 、https://www.wps.cn/article/6aLbA4io.html ）。
- 交付确认实例：官方「AI 写简历」教程给出「输入真实经历→选岗位方向→**逐段核验**→导出 PDF」流程，并区分免费/付费能力（https://www.wps.cn/article/ai-resume-write-how-to-use-steps.html ）——「逐段核验」是少见的官方强调人工验收的表述。
- 独立第三方深度横评少，公开差评聚合本次未查到（这与国内产品评测生态有关，属情报缺口而非无问题）。

### 5.2 本地交付语境
- WPS 是国内「本地办公文件+格式兼容」事实标准之一，在线编辑器完整支持 .doc/.docx（官方：https://www.wps.cn/article/ban-gong-xiao-lv-zai-xian-wen-dang-2026-LspojENp.html ）；对 gaea 这种本地优先工具的启示是「AI 结果直接落在本地原生格式上，交付即文件本身」。

---

## 6. Gamma / Tome（AI 演示文稿成品的交付后链路）

### 6.1 导出保真口碑（重要差评样本）
- **PPTX 导出有损是共识差评**：第三方报告称 Gamma 的 .pptx 导出会「压平图层——动画消失、字体被替换、版式破裂」（SlideGMM：https://www.slidegmm.ai/en/blog/gamma-export-powerpoint-quality-guide ）；GetAlAI 评测列举「导出 PPT/Google Slides 后字体缺失、版式移位、文本框错位、非 16:9 页面」（https://getalai.com/blog/gamma-alternatives ）；用户实拆吐槽（r/powerpoint：https://www.reddit.com/r/powerpoint/comments/1j2zgpj/gamma_import_to_powerpoint/ ）。
- 官方缓解措施：导出时提示版式局限并提供**自定义字体下载直链**（先装字体再打开，减少字体替换）；官方博客宣布过「PPTX 导出改进：字号取整便于导出后再编辑」（24Slides 评测：https://24slides.com/presentbetter/gamma-app-review ；Gamma 官方 LinkedIn：https://www.linkedin.com/posts/gamma-app_exporting-your-gammas-to-powerpoint-got-major-activity-7250203474179616768-Ya__ ）。
- **PDF 导出保真更好但不可编辑**（每页整图式渲染）：MindStudio 教程指出 PDF 是保真与可编辑性的取舍（https://www.mindstudio.ai/blog/how-to-use-gamma-ai-build-presentations-tutorial ）；官方导出文档（PDF/PNG/PPT、单页下载、去水印）：https://help.gamma.app/en/articles/8022861-what-s-the-easiest-way-to-export-my-gamma
- 对 gaea 的镜鉴：gaea 用原生 pptx 直编，理论上避开「压平」问题——但**导出前字体/版式预警**是 Gamma 用血泪换来的标配体验。

### 6.2 分享发布 + 阅读分析（交付追踪）
- 分享设置：链接分享、权限（查看/编辑）、发布为网站（官方：https://help.gamma.app/en/articles/11047226-how-do-collaboration-and-sharing-settings-work-in-gamma ）。
- **Gamma Analytics**：对已分享内容给出浏览量、独立访客、每页停留时长、卡片级互动等参与度指标（官方：https://help.gamma.app/en/articles/11047329-how-do-i-track-my-gamma-s-performance-using-analytics ）；Changelog 显示 API 已可返回 engagement analytics 与 comments（https://meetgamma.canny.io/changelog ）——即「交付后的阅读行为数据」成为产品能力，且进了 API。
- 主题/重排：主题（theme）系统 + 一键重排版（Remix）是 Gamma 成稿后再加工的核心体验（官方帮助中心主题类条目本次未单独抓取，见上两条官方链接所在 help 域）。

### 6.3 模板化复用与工程化交付（本节亮点）
- **Gamma Generate API**：可编程生成 presentations/documents/webpages/social posts（官方：https://help.gamma.app/en/articles/11962420-does-gamma-have-an-api ）。
- **Zapier 官方集成**：表单新回复/CRM 成交/Slack 消息等事件自动触发生成成品（Zapier 官方博客「5 ways to automate Gamma」：https://zapier.com/blog/automate-gamma/ ；Gamma 集成页：https://gamma.app/integrations/zapier ；ChatGPT+Zapier MCP 教程：https://zapier.com/blog/gamma-with-zapier-mcp/ ）。

### 6.4 Tome：一个「交付后链路」的反面教材
- Tome 于 2024 年转向 AI 销售 CRM，**2025-04-30 正式关停演示文稿产品**，存量用户的成稿与再利用被迫迁移，竞品纷纷出「Tome 迁移」承接页（第三方复盘：https://deckary.com/blog/tome-review ）。教训：**云端暗格式（非标准文件）的 AI 成品，其生命周期绑定厂商存亡**——本地标准格式交付（gaea 路线）在「产物主权」上是对这一风险的直接回答。

---

## 7. Zapier / Make：AI 产出的自动投递与归档

- Zapier：9000+ 应用间编排 AI 工作流与 Agent（官方：https://zapier.com/ ）；「Email by Zapier + AI by Zapier」可直接把生成结果发邮件（https://zapier.com/apps/email/integrations/ai ）；AI 邮件分类/路由（https://zapier.com/automation/email-automation/ai-email-management ）；中文实战教程含会议纪要同步、内容分发等模板（aieii：https://aieii.com/posts/2026-03-23-zapier-ai-workflow-tutorial/ ）。
- Make：新一代 **AI Agent 可在画布上直接把 PDF/图片/CSV 作为输入与输出**，端到端完成「生成→投递→归档」（官方公告：https://www.make.com/en/blog/announcing-next-generation-make-ai-agents ；平台总览：https://www.make.com/en ；What is AI automation 指南：https://www.make.com/en/blog/what-is-ai-automation ）。
- 对 gaea 的意义：这两家证明了「AI 成品→自动投递（邮件/网盘/IM）→归档」是标准化需求，且全部由「触发器-动作」范式承载；gaea 本地版可用 webhook/CLI 复刻该范式的最小闭环。

---

## 可借鉴模式清单

| # | 模式名 | 出处 | 对 gaea 的适配点 | 实现量级 |
|---|--------|------|------------------|----------|
| 1 | AI 改动尊重原生修订：用户开着修订，AI 改动全部以修订呈现 | Copilot in Word Edit with Copilot（support.microsoft.com/en-us/word/edit-with-copilot-in-word） | gaea 已是 docx 修订制写入；补齐「修订粒度到词级 + AI 改动作者标记统一可辨」即可对标 | 小 |
| 2 | 共享/重要文档强制「预览 diff→确认→落盘」 | Copilot 在共享文档中必须先在聊天预览、确认后写入（同上） | gaea 产物写入前统一出 diff 预览（现已有版本时间线，可复用其对比视图） | 小 |
| 3 | AI 能力边界诚实条款（明示不能做什么：不能建新文件/不能插图/可能连带删批注） | Copilot 官方支持文档（同上） | gaea 在导出/写入入口列明当前能力边界与副作用提示（如修订写入对既有批注的影响） | 小 |
| 4 | 成稿结构一键固化为模板，「上次的报告=下次的模板」 | Gemini Match doc format（support.google.com/a/users/answer/15146419）；Notion database templates（notion.com/help/database-templates） | gaea 报告拼装产物加「存为模板」：抽取章节骨架+样式约定+数据源占位，下次换数据再生成 | 中 |
| 5 | 页面→数据库化 + AI 自动补属性 | Notion「Make this page a database」+ AI Autofill（notion.com/help/autofill） | gaea 把产物元数据（标题/数据源/版本/状态）结构化为索引页/清单，供检索与再拼装 | 中 |
| 6 | 周期性自动再生成（数据源绑定→定时出新版） | 钉钉定时触发+机器人群推（open.dingtalk.com/document/connection/robot-weekdays-collect）；Notion Repeating templates（notion.com/help/guides/automate-work-repeating-database-templates） | gaea 报告绑定 xlsx 数据源，支持「刷新重建」与本地定时任务（cron）产新版本入时间线 | 中-大 |
| 7 | 事件/API 触发生成（表单/CRM/消息→自动出成品） | Gamma Generate API（help.gamma.app/en/articles/11962420）；Zapier×Gamma（zapier.com/blog/automate-gamma/） | gaea 暴露 CLI/HTTP 触发一次拼装或转换，便于外部系统调用 | 中 |
| 8 | 产出自动投递：邮件/网盘/IM webhook，产完即送达 | Zapier Email+AI（zapier.com/apps/email/integrations/ai）；Make AI Agents 文件级 I/O（make.com/en/blog/announcing-next-generation-make-ai-agents）；飞书消息节点（feishu.cn/content/mtb6n3ah） | gaea 在导出后挂「投递动作」：本地目录/邮件/钉钉企微 webhook，作为 zip 打包的下一步 | 中 |
| 9 | 生成→审批→群推送→验收在同一工作流闭环 | 飞书 Aily Workflow/工作流应用（aily.feishu.cn/hc/1u7kleqg/owh6kvac）；钉钉审批 AI 助理（dingtalk.com/qidian/page-UkDqFcGj.html） | gaea 本地无审批流，但可导出「待审包」（修订版 docx+变更摘要）投递到用户既有审批/IM 流程 | 中 |
| 10 | 任务式交付验收：发起任务→看进展→验收结果 | 飞书豆包工作伙伴（aily.feishu.cn/hc/1u7kleqg/8u02e8ub） | gaea 产物时间线加「已验收」状态标记，验收动作=用户明确确认，可与模板模式（#4）联动 | 小 |
| 11 | 导出前保真预警：字体缺失提示+字体获取直链 | Gamma 导出提示+字体直链（24slides.com/presentbetter/gamma-app-review） | gaea docx/pptx 导出前检测字体可用性并提示替换/内嵌，防「打开就跑样」 | 小 |
| 12 | 保真与可编辑的取舍明示（PDF 高保真不可编辑 vs 可编辑格式有损） | Gamma PDF vs PPTX 取舍（mindstudio.ai/blog/how-to-use-gamma-ai-build-presentations-tutorial） | gaea 三出口处写清各格式保真承诺与已知损耗点，管理预期 | 小 |
| 13 | 活组件跨文档同步复用（一处改处处新） | Loop components（support.microsoft.com/en-us/loop/get-started-with-microsoft-loop） | gaea CrossEmbed（xlsx 嵌 Word/PPT）向「源文件更新→嵌入处提示刷新」演进 | 中 |
| 14 | 交付后阅读分析回流（谁看了/看多久/批注进 API） | Gamma Analytics（help.gamma.app/en/articles/11047329）；Gamma changelog（meetgamma.canny.io/changelog） | gaea 本地交付难以收集阅读数据，可做「批注/修订回流」：解析回收 docx 的批注与修订→生成下一轮 AI 修改指令 | 中 |
| 15 | 本地标准格式=产物主权（防云端暗格式关停风险） | Tome 2025-04-30 关停（deckary.com/blog/tome-review） | gaea 坚持落标准文件（docx/xlsx/pptx/pdf/md），宣传点可明写「产物不锁死在云端」 | 已有 |
| 16 | 成稿直转另一形态（Word→PPT 直出） | WPS AI 文档→PPT（wps.cn/article/ban-gong-xiao-lv-yan-shi-wen-gao-2026-IEMmSmwl.html） | gaea 报告→PPT 大纲卡已具雏形，补「同一数据源多形态一键再生成」入口 | 中 |
| 17 | 官方强调「逐段人工核验」的交付确认流程 | WPS AI 简历教程（wps.cn/article/ai-resume-write-how-to-use-steps.html） | gaea 长文产物提供「分段核验」视图（逐段确认/打回），降低整篇验收心理负担 | 中 |

---

## 参考 URL 全表

### Microsoft 365 Copilot
1. https://support.microsoft.com/en-us/word/welcome-to-copilot-in-word
2. https://support.microsoft.com/en-us/word/edit-with-copilot-in-word
3. https://support.microsoft.com/en-us/word/copilot/rewrite-text-with-copilot-in-word
4. https://techcommunity.microsoft.com/blog/microsoft365copilotblog/copilot-in-word-new-capabilities-for-document-workflows/4508974
5. https://techcommunity.microsoft.com/blog/microsoft365copilotblog/introducing-word-excel-and-powerpoint-agents-in-microsoft-365-copilot/4470604
6. https://www.microsoft.com/en-us/microsoft-365/blog/2026/04/22/copilots-agentic-capabilities-in-word-excel-and-powerpoint-are-generally-available/
7. https://www.microsoft.com/en-us/microsoft-365/word/word-ai
8. https://support.microsoft.com/en-us/loop/get-started-with-microsoft-loop
9. https://support.microsoft.com/en-us/loop/use-loop-components-in-outlook
10. https://learn.microsoft.com/en-us/answers/questions/5881900/how-do-i-use-loop-in-teams-and-share-a-component-f
11. https://learn.microsoft.com/en-us/answers/questions/5698940/document-editing
12. https://learn.microsoft.com/en-us/answers/questions/5858106/copilot-platform-changes-no-longer-meet-profession
13. https://rencore.com/en/blog/microsoft-loop-bridging-gap-seamless-team-collaboration/
14. https://blog.apps4.pro/microsoft-loop-features-use-cases-governance
15. https://www.linkedin.com/posts/fsistrategies_copilot-in-word-new-capabilities-for-document-activity-7452726728847945729-xiS6
16. https://thewincentral.com/copilot-word-new-capabilities-document-workflows/
17. https://windowsforum.com/windows-news.4/copilot-in-word-gets-word-level-track-changes-comments-and-better-structure.413242/ （403，仅存目）
18. https://www.reddit.com/r/Office365/comments/1o0lfuo/copilot_output_loses_formatting_when_copying_into/
19. https://www.reddit.com/r/microsoft_365_copilot/comments/1qvuef4/why_are_we_struggling_to_create_word_documents/
20. https://www.reddit.com/r/CopilotPro/comments/1mmoz7r/i_paid_for_pro_so_i_could_export_to_pdf_but_its/
21. https://www.reddit.com/r/microsoft/comments/1l07r4q/copilot_in_word_is_such_a_mess/
22. https://www.gitbit.org/docs/copilot-in-word-document-0kz9yhme
23. https://www.remio.ai/post/microsoft-copilot-won-t-create-documents-3-hacks-to-fix-formatting-and-template-issues

### Gemini in Google Workspace
24. https://support.google.com/a/users/answer/15146419?hl=en
25. https://workspace.google.com/products/docs/ai/
26. https://blog.google/innovation-and-ai/products/gemini-app/generate-files-in-gemini/
27. https://support.google.com/gemini/community-video/430728918/how-to-export-word-excel-pdf-files-from-google-gemini-keshavtechy-google-gemini-update?hl=en
28. https://support.google.com/gemini/thread/424706282/how-to-export-gemini-chat-to-pdf-google-docs?hl=en
29. https://www.mindstudio.ai/blog/gemini-in-google-docs-sheets-slides-what-you-can-do
30. https://www.reddit.com/r/GeminiAI/comments/1qmbwoj/gemini_not_able_to_create_google_docs_why/
31. https://www.reddit.com/r/gsuite/comments/13oem44/how_do_i_convert_google_docs_to_word_without/
32. https://support.google.com/docs/thread/251117694/table-of-contents-is-changing-fonts-when-it-s-refreshed?hl=en
33. https://support.google.com/docs/thread/10229737/font-style-on-table-of-contents-doesn-t-match-text-in-the-body?hl=en
34. https://learn.microsoft.com/en-us/answers/questions/5339252/formatting-changes-between-ms-word-and-googledocs
35. https://employernews.co.uk/news/converting-google-docs-to-word-preserving-formatting-and-comments/

### Notion AI
36. https://www.notion.com/help/export-your-content
37. https://www.notion.com/help/import-data-into-notion
38. https://www.notion.com/help/autofill
39. https://www.notion.com/help/database-templates
40. https://www.notion.com/help/guides/automate-work-repeating-database-templates
41. https://www.notion.com/help/guides/automatically-generate-blocks-pages-with-buttons
42. https://www.notion.com/help/security-and-privacy
43. https://www.notion.com/help/workspaces-on-mobile
44. https://www.notion.com/blog/personalize-public-pages
45. https://kurashi-notion.com/en/blogs/notion/notionai-database2
46. https://www.eesel.ai/blog/notion-ai-autofill
47. https://techresolve.blog/2025/12/23/notion-devs-do-you-plan-to-fix-the-awful-pdf-expo/
48. https://unmarkdown.com/blog/notion-export-broken
49. https://www.reddit.com/r/Notion/comments/mo8hm2/guide_how_to_properly_export_notion_to_pdf/
50. https://bloggingx.com/export-notion-to-word/
51. https://www.helpkit.so/blog/how-to-use-notion-to-create-a-help-center-for-your-company
52. https://www.youtube.com/watch?v=jKs6HA9UlNg

### 飞书 / 钉钉
53. https://aily.feishu.cn/hc/1u7kleqg/8u02e8ub
54. https://aily.feishu.cn/hc/1u7kleqg/jsmwydi4
55. https://aily.feishu.cn/hc/1u7kleqg/owh6kvac
56. https://www.feishu.cn/hc/zh-CN/articles/318505042260
57. https://www.feishu.cn/content/8u02e8ub
58. https://www.feishu.cn/content/mtb6n3ah
59. https://www.feishu.cn/content/sj7drxsj
60. https://www.dingtalk.com/qidian/page-UkDqFcGj.html
61. https://www.dingtalk.com/qidian/page-Q0MQ5YoW.html
62. https://www.dingtalk.com/qidian/page-idbENeWS.html
63. https://www.dingtalk.com/qidian/page-xDECjiz8.html
64. https://open.dingtalk.com/document/connection/robot-weekdays-collect
65. https://open.dingtalk.com/document/connection/automated-process-usage-guide
66. https://open.dingtalk.com/document/ai-dev/group-message-aware-trigger
67. https://open.dingtalk.com/document/connection/commerce-synchronizedl-time
68. https://finance.sina.cn/roll/2026-06-09/detail-iniavshm4879661.d.html
69. https://help.aliyun.com/zh/jvs/user-guide/dingtalk-ai-form-and-document-access

### WPS AI / 金山
70. https://www.wps.ai/zh-hk/
71. https://lingxi.wps.cn/
72. https://zh-hant.wps.com/feature/zh-CN-free-template-editing-app/
73. https://www.wps.cn/article/fdRFDBqM.html
74. https://www.wps.cn/article/ban-gong-xiao-lv-yan-shi-wen-gao-2026-IEMmSmwl.html
75. https://www.wps.cn/article/6aLbA4io.html
76. https://www.wps.cn/article/ai-resume-write-how-to-use-steps.html
77. https://www.wps.cn/article/ban-gong-xiao-lv-zai-xian-wen-dang-2026-LspojENp.html
78. https://ai-bot.cn/（WPS 灵犀条目，转引自检索）

### Gamma / Tome
79. https://help.gamma.app/en/articles/8022861-what-s-the-easiest-way-to-export-my-gamma
80. https://help.gamma.app/en/articles/11962420-does-gamma-have-an-api
81. https://help.gamma.app/en/articles/11047329-how-do-i-track-my-gamma-s-performance-using-analytics
82. https://help.gamma.app/en/articles/11047226-how-do-collaboration-and-sharing-settings-work-in-gamma
83. https://meetgamma.canny.io/changelog
84. https://gamma.app/integrations/zapier
85. https://www.slidegmm.ai/en/blog/gamma-export-powerpoint-quality-guide
86. https://getalai.com/blog/gamma-alternatives
87. https://www.mindstudio.ai/blog/how-to-use-gamma-ai-build-presentations-tutorial
88. https://24slides.com/presentbetter/gamma-app-review
89. https://www.reddit.com/r/powerpoint/comments/1j2zgpj/gamma_import_to_powerpoint/
90. https://www.linkedin.com/posts/gamma-app_exporting-your-gammas-to-powerpoint-got-major-activity-7250203474179616768-Ya__
91. https://deckary.com/blog/tome-review

### Zapier / Make
92. https://zapier.com/
93. https://zapier.com/apps/email/integrations/ai
94. https://zapier.com/automation/email-automation/ai-email-management
95. https://zapier.com/blog/automate-gamma/
96. https://zapier.com/blog/gamma-with-zapier-mcp/
97. https://zapier.com/apps/google-slides/integrations/gamma
98. https://www.make.com/en
99. https://www.make.com/en/blog/announcing-next-generation-make-ai-agents
100. https://www.make.com/en/blog/what-is-ai-automation
101. https://aieii.com/posts/2026-03-23-zapier-ai-workflow-tutorial/

---

## 附：本次未查到/待补清单（诚实声明）
- Copilot in Word 的「样式继承」官方细节：官方只说用内建样式写入，未见「继承现有文档自定义样式」的逐条说明。
- Copilot/Power Automate 批量文档交付的官方现成模板：本次未逐条核查。
- Gemini in Workspace 的 Apps Script/Gemini API 工程化交付细节：未展开。
- Notion 官方 API 批量写入性能/限制：未展开。
- 飞书 Aily 的公开差评聚合、WPS 灵犀的独立第三方深度横评与差评聚合：未查到成规模来源。
- 飞书两篇官方手册正文因 JS 渲染未能直接抓取，相关结论均来自检索摘要中的原文片段，建议复核时人工打开对应 URL。
- techcommunity.microsoft.com 两条关键博客在调研时处于维护状态，要点经二手转载交叉确认（LinkedIn/thewincentral/Windows Forum 检索摘要）。
