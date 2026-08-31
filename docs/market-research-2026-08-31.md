# gaea 市场调研复扫 · 模块制（2026-08-31）

> 基线：v4.11.0（2026-08-30 发布，绑定面 544）。方法：**8 个并行子代理分模块调研**（通用办公 / 造价 / 记忆知识底座 / 模型中心 / 编程 / 创作包 / 轻语陪伴 / 触点层），实际网络检索核实，重点窗口 2026 年（尤其 3–8 月）；未能核实的信息在文中如实标注「未核实」，未编造。
> 与既有文档的关系：本文是对 `docs/gaea-nextgen-roadmap-2026.md`（v3.7.0 基线调研）的**增量复扫**，不替代路线图；分模块原始稿保留在 `docs/research-2026-08-31/`，本文为合成版。

---

## 读前一分钟（TL;DR · 八模块八句话）

1. **通用办公**：「规划-审阅-执行」已成产品开关（WorkBuddy Ask/Plan/Craft、Claude Cowork 审批制），行业下半场卷权限围栏与安全沙箱；gaea 的「事后复核」（Verifier 引用级校验 + 证据链 + 回滚）仍是行业空白，「本地模型 + 本地执行」完整闭环仍独占。威胁 = 腾讯系双品牌夹击（WorkBuddy 免费积分 + Manus 并入腾讯系恢复运营）。
2. **造价**：广联达已把「AI 组价/AI 询价/AI 清标/无感建库」列成官方能力清单（2026-08 更新），gaea 未做项的差异化窗口在收窄；但龙头 AI 绑定企业流程与云账号——**个人从业者的单机、可审计、数据不出机组价/复核助手是空位**；GB/T 50500-2024（2025-09-01 实施）换代期是轻量工具快跑的切入机会。算量赛道别进。
3. **记忆底座**：BM25+语义混合检索已成行业标配（不再稀缺）；遗忘范式 =「失信标记 + 复核修正」而非删除；GraphRAG 进维护模式、agentic RAG 成默认；「记忆所有权/可移植」成为行业叙事——本地优先的 gaea 天然受益。MemOS MemCube 是「记忆分区」最强先例，但按「生活领域」分区且互不检索仍无先例，gaea 双空间隔离语义仍独特。
4. **模型中心**：计费已变三维（订阅积分 + 峰谷时段 + 缓存命中）；**GLM Coding Plan 改积分制 + 旧模型名自动切换，gaea 的 coding 端点静态适配面临失真**；Cherry Studio v2（8 月连发 5 版）直接覆盖模型中心核心体验；「一张引擎卡管全模态 + 成本仪表」在桌面端仍是空白位。
5. **编程**：「壳窗 + 外部 CLI」已从民间黑箱变成官方协议（claude attach/respawn、Remote Control、SessionStart staleness），**用户拍板的独立 DSH 窗口决策在 2026 格局下仍然成立且更稳**；独立壳窗商业脆弱（opcode/Crystal/Vibe Kanban 三个停更案例），gaea 只做体验层、不承担引擎维护，站位正确；「低焦虑管家层」（健康徽标/断连自愈/完成通知）是空位。
6. **创作包**：文图联动成头部标配（Sudowrite Visualize、蛙蛙「小说→剧本→漫剧视频」）；角色资产升级为「文图共用资产」——「创作间」角色/世界观一套数据共用正踩此方向；指令编辑 + 图层化成云端旗舰标配，gaea 绘梦 img2img 缺口在拉大；GLM-Image 升图像旗舰 + CogView-3-Flash 免费档可做「旗舰质量 + 免费兜底」分层。
7. **轻语**：**中国监管落锤——《人工智能拟人化互动服务管理暂行办法》2026-04-10 发布、2026-07-15 施行**（禁未成年人虚拟伴侣、不得诱导情感依赖、月活 10 万须安全评估）；gaea 个人本地使用不在适用范围，新规合规成本反向放大「数据不出机 + 无氪金 + 硬隔离」卖点；海外 Nomi 已把「长期记忆 + 主动关心 + 情感语音」三件套产品化；大众级桌面本地陪伴产品仍未发现（SillyTavern 证明极客需求存在）。
8. **触点层**：端到端实时语音已成红海且国产可直连、计费透明（GLM-Realtime 0.18 元/分钟、Qwen3.5-Omni-Realtime 支持 semantic_vad 语义打断、豆包 Seeduplex 全双工）；**gaea Realtime 代码与主流事件协议高度同构，换引擎边际成本低**；微信 hook 生态收缩（WeChatFerry 归档停更）、iLink 生态走强——gaea 押注 iLink 路线正确；wecom-cli（企微官方拥抱 Agent）是合规的第二 IM 入口候选。

## 跨模块综合结论（五个确定性）

1. **「可审计」是 gaea 最深的护城河，且行业正在逼近**。办公（ChatExcel 把「可复核」写进官网但停留口号层、Cowork/WorkBuddy 只有事前审批）、造价（广联达 AI 组价&审核）、记忆（OpenWiki 自纠错）都在往「可信」卷，但无人做到「事后引用级校验 + 操作回放 + 一键回滚」。gaea 的 Apply→Verify→Journal 先发半步，窗口不会永远开着——0-3 月应把它从内部能力**产品化为对外叙事与 UI**。
2. **「本地优先」从差异化卖点升级为合规红利**。陪伴新规、企业数据安全、记忆所有权叙事三股力量同时抬高「数据不出机」的价值；WorkBuddy 只做「本地文件 + 云端模型」半闭环，完整「本地模型 + 本地执行 + 可审计」仍无人做。
3. **计费与模型目录进入快变期，静态资产会失真**。GLM 积分制 + 模型名自动切换、Kimi V1 全平台下线（2026-08-31）、各家缓存专价——静态模型目录与 coding 端点适配需要热更新机制；模型中心欠账（成本仪表 + 自动路由）正好踩在 2026 计费红利上（缓存命中 0.1–0.19×、峰谷五折），**路由 v1 以「本地优先 → 缓存命中最大化 → 峰谷错峰」为目标函数，不做语义缓存**（GPTCache 教训）。
4. **实时语音换挡窗口已开**。拼接管线相对端到端已「可感落后一代」（打断自然度差距最直观），而 gaea Realtime 代码全就绪、国产 API 成熟且按分钟可预算——真机验证是最便宜的一跃（既有欠账①）。
5. **生态位警示：OpenClaw 现象**。OpenClaw（38.8 万星）+ AutoClaw（智谱官方本地客户端）+ iLink Bot SDK 形成「开源个人 AI 助理」事实标准，横跨模型中心/触点/编程壳窗三个模块与 gaea 相遇。gaea 的回答不是对抗，而是差异化：**中文办公纵深（造价/规范包）+ 可审计 + 双空间（工作/情感）隔离 + 单机零部署**，是通用框架不做的。

## 下一刀建议（研究增量 × 既有欠账 × 用户拍板，三重过滤）

| 优先 | 候选刀 | 依据（调研 → 行动） |
|---|---|---|
| ★★★ | **Realtime 真机验证收口**（欠账①） | 触点层调研：GLM-Realtime 0.18 元/分钟可预算、Qwen3.5-Omni-Realtime 的 semantic_vad 与 TurnControl 对齐；代码全就绪只差真机。收口方法不变：用户提供 Key + 麦克风走检查清单；可顺手把 provider 白名单从 "openai" 扩到 "zhipu"（按分钟计费更适合个人） |
| ★★★ | **指令内核 v1 + Ctrl+K 命令面板先行**（v4.5.2 倒序先做，桌面零外部依赖） | 指令中枢是既定主轴；桌面命令面板先行验证「意图解析 + 能力注册表」，语音/微信后接同一内核 |
| ★★☆ | **模型中心「计费快变」三件套**：静态目录热更新 + GLM coding 积分口径 + 成本仪表 v0（自动路由 v1 的前置，欠账④） | 模型中心调研：GLM Coding Plan 积分制使 coding 端点静态适配失真风险已落地；成本仪表在桌面端仍是空白位 |
| ★★☆ | **Verifier 产品化一刀**：通道 A 翻成 UI（声明↔实况 diff 可视化 / opsJson 操作回放 / 一键回滚入口） | 办公调研：事后复核是行业空白，先发占位 |
| ★☆☆ | **做梦 2.0 主动预取 MVP**（欠账③）：开机/空闲预装配高频工作记忆，纯本地「晨报」形态 | 记忆调研：ChatGPT Pulse / Gemini Spark 把「记得你 + 主动服务」变成默认预期 |
| ★☆☆ | GLM-Image 分层用法（glm-image 旗舰 + CogView-3-Flash 免费兜底）+ 角色卡字段一键注入生图提示词 | 创作调研：GLM-Image 已升旗舰；角色资产文图共用是「创作间」护城河 |
| 观察 | 造价 AI 组价最小闭环（强制溯源引用版）、轻语合规护栏开关（非真人提示/一键清空/时长提醒）、wecom-cli 第二 IM 入口 | 造价包优先级待用户拍板；合规护栏成本低、可顺手做；wecom-cli 待微信通道风险再评估 |

**不做清单（本轮调研确认）**：语义缓存（GPTCache 误命中 + 停止适配）、原生编程工作台（用户拍板 + 调研双重验证）、算量赛道（QuantifAI/Togal/Kreo 重度内卷且重资源）、人格包公开分享（新规平台责任风险，观察执法动向再定）。

---

## 1. 通用办公（v4.11.0 基线复扫 · 2026-08-31）

### 市场格局 · 最新动态

**腾讯系加速整合，成为个人办公 agent 最大变量。** 腾讯 WorkBuddy 于 2026-02-06 启动内测、03-09 正式上线（[百度百科](https://baike.baidu.com/item/WorkBuddy/67362053)、[知乎教程](https://zhuanlan.zhihu.com/p/2055276137723572615)），定位「全场景 AI 办公工作台」。官方文档明确「自主规划执行：自动拆解任务、规划步骤、执行操作」，支持「本地文件操作：可读取授权的电脑文件夹，进行批量处理」，并设「默认权限与安全沙箱」专章（[CodeBuddy 文档](https://www.codebuddy.cn/docs/workbuddy/Overview)）。Manus 经历剧变：Meta 约 20 亿美元收购于 2026-04-27 被国家发改委依外商投资安全审查叫停（AI 领域首例），2026-06 运营切割、停止数据共享，2026-07 以腾讯为首的中方财团按原价接盘，2026-08-11 官宣恢复独立运营；其年化收入由 2025-12 的 1 亿美元涨至 4-5 亿美元（[腾讯新闻](https://news.qq.com/rain/a/20260807A06TYJ00)、[知乎](https://www.zhihu.com/question/2070808774667921064)）。

**海外三线齐动。** Anthropic 将 Cowork 做成独立产品线：面向非编程知识工作（研究/分析/文档/多步任务），「Claude shows its plan and waits for your approval before anything significant」，文件夹与工具白名单、删除需审批，2026-02-24 上线私有插件市场、2026-08-26 上线内置浏览器；模型线 2026-06-30 Sonnet 5、2026-07-24 Opus 5（主打 long-running agents）（[Cowork 产品页](https://claude.com/product/cowork)、[Anthropic News](https://www.anthropic.com/news)）。Kimi（月之暗面）发布 K3：2.8 万亿参数、1M 上下文、权重 2026-07-27 前开源，官网主推「咨询级 PPT」「Swarm 智能体集群与 Goal 模式并行执行任务」；Kimi Agent 页标注「升级 Office 进阶能力：支持生成 Word/PDF 万字论文、构建 Excel 复杂公式」（[K3 文档](https://platform.kimi.com/docs/guide/kimi-k3-quickstart)、[Kimi 官网](https://www.kimi.com/)、[Kimi Agent](https://www.kimi.com/zh/agent)）。微软在做减法：2026-03 经 M365 Agent Store 引入合作伙伴内嵌体验（Canva/Figma 等），2026-08 宣布消费版 Copilot 与 M365 Copilot 合并并移除 Mico、Deep Research 等功能（[Wikipedia](https://en.wikipedia.org/wiki/Microsoft_365_Copilot)）。

**垂直工具 MCP 化。** ChatExcel（酷表，现属元空 AI）开放平台「支持 MCP 调用对接」（2026-03），官网主打生成「可复核、可汇报、可落地」的分析结果（[开放平台](https://open.chatexcel.com/)、[官网](https://www.chatexcel.com/homesite/home)）。

### 范式迁移（上轮调研以来的变化）

1. **「规划-审阅-执行」从共识变成产品开关，行业开始卷权限围栏与安全沙箱。** WorkBuddy 三模式 Ask/Plan/Craft，其中 Plan「先给出分步执行方案，你确认后才动手」（[菜鸟教程](https://www.runoob.com/ai-agent/workbuddy-usage.html)）；Cowork 把事前审批做成默认。竞争重心移向：最小授权（只授权必要目录）、敏感目录自动拦截、高危操作二次确认（WorkBuddy）、工具白名单与删除审批（Cowork）。
2. **本地执行权被巨头跟进，但只做一半。** WorkBuddy 宣称「你的文件处理默认在本地完成，原始数据不上传云端，服务端只处理数据片段、用后即弃」，同时要求「保持网络连接」调云端模型（[教程](https://www.runoob.com/ai-agent/workbuddy-usage.html)）。云端 agent 的本地文件执行已是标配，但「本地模型 + 本地执行」的完整数据不出机闭环仍无人做。
3. **成品交付 + 记忆/技能沉淀全面普及。** 金山灵犀官网以「记住/连接/行动/进化」四大能力为纲，直接交付 策略报告.docx、经营分析.xlsx、汇报演示.pptx，工作流可沉淀为可复用技能（[灵犀官网](https://lingxi.cn/)、[landing](https://lingxi.kdocs.cn/landing/)）；WorkBuddy 引导用户固定工作空间「长期沉淀工作上下文，AI 对你的背景越用越熟」。上轮锚点「灵犀 2026-07 升级为独立 AI 办公 Agent」：其 Agent 化定位与官网吻合，但具体升级时间点未核实；「WorkBuddy 人机双写」本轮未检索到公开信息（未核实）。
4. **「可复核」进入营销话术但停留在口号层。** ChatExcel 把「可复核」写进官网，但未见引用级校验、操作日志或回滚细节；长任务中断、产出不稳定仍是行业痛点（易观，见上轮基线）。Google Workspace Gemini（Deep Research）、Notion AI 2026 年 3-8 月的具体更新未检索到可靠信息（未核实）。

### 对 gaea 的机会与威胁

**机会**
- **事后复核是行业空白。** 竞品普遍只有「执行前确认」（Plan/Craft 开关、approval before acting），gaea Verifier 通道 A 的引用级校验（opsJson 操作日志 + 声明↔实况比对）+ 证据链 + 回滚，比行业深一层，正好卡位「下半场比交付与安全」（上轮 36氪结论延续）。
- **「本地模型 + 本地执行」完整闭环仍独占。** WorkBuddy 必须联网、Cowork 云订阅；gaea 断网可用的敏感文档场景（合同/财报/标书）云端产品进不来。
- 多跳因果链（≤2 跳）、多文件任务（读报表→计算→报告）等深校验能力，尚无竞品公开对标。

**威胁**
- WorkBuddy 免费积分 + 腾讯文档/微信/企微生态 + 本地文件授权读写 + 教程遍地，直接覆盖个人桌面办公主场景，用户心智教育已完成。
- Manus 并入腾讯系后，形成「通用 agent（Manus）+ 办公工作台（WorkBuddy）」双品牌夹击。
- Kimi K3 开源 + Agent「Office 进阶能力」拉高云端低价预期，放大本地小模型的能力差距感知。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **现在 0-3 月**：① 把 Verifier 复核产品化为对外叙事与 UI：声明↔实况 diff 可视化、opsJson 操作回放、一键回滚入口——行业只有事前审批，先发占位「事后可审计」；② 权限对齐行业底线：目录白名单、敏感目录拦截、高危操作二次确认（WorkBuddy/Cowork 已是标配）。
- **下个 3-6 月**：① 项目/工作空间记忆：固定工作目录 + 项目级事实基线长期沉淀（WorkBuddy 已引导用户如此使用）；② 办公技能沉淀闭环：把成功任务一键封装为可复用技能（对标灵犀「进化」与 Cowork 插件市场）；③ 多文件长任务的进度面板与中断恢复（行业持续痛点）。
- **愿景 6-12 月**：① 「本地模型 + 本地执行 + 可审计回滚」三合一作为主打叙事，抢占云端 agent 无法进入的敏感文档人群；② MCP 互操作：出站接入本地工具生态、可被其他 agent 调用（ChatExcel 已示范垂直工具 MCP 化）。

### 参考来源

- WorkBuddy 官方文档（定位/规划执行/本地文件/安全沙箱）：https://www.codebuddy.cn/docs/workbuddy/Overview
- WorkBuddy 上手教程（四阶段、Ask/Plan/Craft、本地执行与隐私、缺点）：https://www.runoob.com/ai-agent/workbuddy-usage.html
- WorkBuddy 百度百科（2026-02-06 内测）：https://baike.baidu.com/item/WorkBuddy/67362053
- WorkBuddy 知乎教程（2026-03-09 上线）：https://zhuanlan.zhihu.com/p/2055276137723572615
- Manus 收购-否决-接盘-独立运营时间线：https://news.qq.com/rain/a/20260807A06TYJ00
- Manus 恢复独立运营（知乎讨论，2026-08-11 信件）：https://www.zhihu.com/question/2070808774667921064
- Claude Cowork 产品页（审批机制/权限围栏/插件市场/内置浏览器）：https://claude.com/product/cowork
- Anthropic 新闻列表（Sonnet 5 2026-06-30、Opus 5 2026-07-24）：https://www.anthropic.com/news
- Kimi K3 技术文档（2.8 万亿参数/1M 上下文/开源时间）：https://platform.kimi.com/docs/guide/kimi-k3-quickstart
- Kimi 官网（K3 上线/咨询级 PPT/Swarm 与 Goal 模式）：https://www.kimi.com/
- Kimi Agent（Office 进阶能力）：https://www.kimi.com/zh/agent
- Microsoft 365 Copilot - Wikipedia（Agent Store 2026-03、2026-08 合并）：https://en.wikipedia.org/wiki/Microsoft_365_Copilot
- 金山灵犀官网（记住/连接/行动/进化、成品交付）：https://lingxi.cn/ ；https://lingxi.kdocs.cn/landing/
- ChatExcel 开放平台（MCP 对接）：https://open.chatexcel.com/ ；官网（可复核/可汇报/可落地）：https://www.chatexcel.com/homesite/home

## 2. 造价数据库（v4.11.0 基线复扫 · 2026-08-31）

> 方法说明：本环境无 WebSearch 工具，使用 WebFetch 抓取 cn.bing / 360 搜索结果页与厂商官网核实；部分新闻原始 URL 为搜索引擎跳转链接未能留存，已标注「出处+日期，原文链接未留存」。检索窗口：2026-08-31。

### 市场格局 · 最新动态

- **广联达已把「AI 组价」做成官方标配**。其《成本（造价）业务 DATA+AI 解决方案》页（页面资源更新于 2026-08-20）明确列出：建设方侧「AI 组价&审核、AI 询价比价、AI 清标、无感建库」，施工方侧「AI 量价双控」，数据底座挂指标网（gldzb.com）+广材网（gldjc.com），覆盖建设/施工/咨询/行业生态四端（https://compus.glodon.com/solution/175.html ）。即 gaea 待做的 AI 组价、询价飞轮已被龙头产品化。
- **广联达 QuantifAI：AI 算量走向全球市场**。2026-08-28 官网新闻：在 PAQS Congress 2026（8.21-25，科伦坡）发布面向全球的 AI 算量方案，覆盖图纸识别→三维建模→算量→模型检查→清单编制→成果输出；同期发表论文《AI 驱动的建筑成本管理与评标系统框架》，称多模态大模型+图神经网络+规则/AI 双引擎使整体效率提升 60%–80%、计算准确率 80%+，并强调「AI 造价不能只靠通用大模型，须与行业知识、业务流程、行业数据、专业软件深度融合」（https://www.glodon.com/news/1619.html ）。
- **收费范式：双轨制 + RaaS 叙事**。广联达 2026-01-15 机构调研确认「AI 赋能现有业务提质升级 + 新增 AI 原生产品单独收费」双轨模式、不影响原年费，并提出行业从「功能付费向 RaaS（按结果付费）演进」（新浪财经/21 智讯报道，2026-01-16，原文链接未留存；转载见 https://www.360kuai.com/pc/9be30ce9f4e0b24a0 ）。另：2026-06-01「2026 工程数智大会」（深圳）主打产业 AI×BIM2.0（第一财经报道，原文链接未留存）。
- **第二梯队跟进**：新点软件 2024-06 公告拟投 1.62 亿元建「行业大模型及数据要素运营平台」（3 年期、Agent 落地，凤凰网财经 2024-06-19，原文链接未留存）；新中大推「六和 AI 训推平台」并中标浙建集团训推一体平台的造价分析/智能问答模块（中国建设新闻网 2025-04-22；候选人公示 2025-12-02，原文链接未留存）；品茗有「茗智·企业大模型」及 AI+基建、AI+招标采购动态（https://www.pinming.cn/news_cont_7096.html ）；鲁班被报道「借大模型从工具商向智能解决方案商跃迁」（网易 2026-07-15，原文链接未留存）。**斯维尔 2026 年 AI 造价新动作：未核实**（官网 JS 渲染无法抓取正文）。
- **国际线：Agent 化+明码标价**。Kreo 推出自主 Agent「Caddie」（读图-算量-交付工程量、技能市场、语音交互），定价 Lite $35 / Plus $70 / Pro $175 每人每月，2026-08-25 仍在迭代，"AI Database" 标记 Coming Soon（https://kreo.net/ ）；Togal.AI 官网 2026 在售 Togal Assemblies、Togal.CHAT，宣称平面图识别 98% 准确、提效 5 倍（https://www.togal.ai/ ）；2026 年融资动态未核实。

### 范式迁移（上轮调研以来的变化）

1. **从「AI 讲 PPT」到「AI 进报价单」**：上轮基线中龙头 AI 多为战略叙事；本轮广联达官网已将 AI 组价&审核、AI 询价比价、AI 清标列为正式能力清单（同上 solution/175.html），落地深度首次可从产品页验证。
2. **政策落点已到**：GB/T 50500-2024 于 2024-11-26 发布、**2025-09-01 起实施**，由强制转推荐、清单准确性责任转向发包人；江西省等要求国有资金项目率先执行（江西省住建厅通知 2025-07-18，原文链接未留存）。至 2026 年中行业仍在消化期——服务新干线论坛 2026-07 仍在征集新标准应用指南资料——清单/定额规则库全量换代是全行业更新负担，也是切入点。
3. **数据护城河收窄但仍在**：龙头靠指标网/广材网/信息价闭环喂养 AI；而长尾已用通用大模型+公开定额踩平技术门槛——个人小站「造价HOME」已上线「造价 Ai 速答、套定额 AI 助手、定额自动套(2026)」并免费开放江苏 2026 消耗量定额（http://www.zaojiahome.com/ ）。「大模型+造价」不再是技术壁垒，数据与工作流才是。
4. **从业者叙事转变**：广联达系社区 2026-08-10 发文《AI 取代造价员？这五个误区该破了》，主流口径变为「会用 AI 的造价员替代不会用的」（服务新干线头条，原文链接未留存）；造价通社区 2026-03-10 亦称繁重算量将被自动化取代、从业者需上移（建识频道，原文链接未留存）。付费参照系：广联达 AI 原生产品单独收费、造价通数智询价/材价按期续购+询价按条计费、信息价 PDF 单份 45–60 元（https://www.zjtcn.com/ ）——个人对小额、按需、结果导向付费有真实习惯。

### 对 gaea 的机会与威胁

- **威胁**：①AI 组价/询价/清标已是龙头官方能力，gaea 这些「未做项」的差异化窗口在关闭，跟随时须换定位（个人版、本地版）；②龙头 RaaS 叙事可能抬升用户对「按结果付费」的期待，纯工具订阅被挤压；③数据墙：信息价/材价/指标依赖造价通、广材网等在线服务，gaea 单机无自有数据源，pricefeed 的数据获取是硬约束。
- **机会**：①龙头 AI 深度绑定云账号与企业流程，**个人从业者被产品体系忽视**——广联达云计价 GCCP7.0 虽「面向企业与个人」，但 AI 组价&审核等能力明显面向建设/施工方组织流程；单机、可审计、数据不出机的个人 AI 组价/复核助手存在空隙。②广联达自己承认 AI 造价须「行业知识+行业数据」——gaea 的成本数据带溯源字段，与造价「可复核」的职业习惯天然契合；个人多年积累的组价文件（Excel/计价软件导出）是龙头拿不到的私有语料。③新清单市场化计价弱化定额依赖、强化指标与历史价格依赖，利好「个人成本数据资产管理」这一 gaea 已有底座（costref/coststage）。④新标准 2025-09-01 实施后的规则换代期，轻量工具反而比重型软件更新更快。
- **不建议**：进入算量（CAD/BIM 图形识别）赛道——QuantifAI、Togal、Kreo 已重度内卷且重资源。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **0–3 月**：①清单/定额底座对齐 GB/T 50500-2024 新字段与责任条款（差异化卖点是「2026 新标准已适配」）；②「AI 组价建议」最小闭环：基于用户导入的 costproject/costref 历史项目给出组价建议，**每条建议强制附溯源引用**（引用哪条定额/哪个历史项目），与委托式可审计哲学对齐；③pricefeed 支持手动/文件导入信息价并留痕。
- **下个 3–6 月**：①对标广联达「无感建库」做个人版：从用户既有 Excel/计价文件自动沉淀个人指标库；②五算对比（估算→概算→预算→结算→决算）作为审计型报表；③询价飞轮 MVP：手动询价记录+多供应商比价，参考造价通按条付费但本地免费。
- **愿景 6–12 月**：①动态价格大脑：对接公开行情（造价通类走势/行情分析为合法公开源）做本地预测与价格预警；②成本问答 Agent（本地记忆+委托式执行：目标→计划→执行→复核全留痕）；③探索「按结果付费」心智下的个人订阅定价锚点。

### 参考来源

1. 广联达成本（造价）业务 DATA+AI 解决方案（AI 组价&审核/AI 询价比价/AI 清标/无感建库/AI 量价双控）：https://compus.glodon.com/solution/175.html （页面资源更新 2026-08-20）
2. 广联达官网新闻（PAQS/QuantifAI/CADCG2026 索引）：https://compus.glodon.com/ ；QuantifAI 详报：https://www.glodon.com/news/1619.html （2026-08-28）
3. 广联达双轨收费+RaaS（2026-01-15 机构调研）：https://www.360kuai.com/pc/9be30ce9f4e0b24a0 （新浪财经/21 智讯报道转载，2026-01-16）
4. Kreo 2026（Caddie Agent/定价/2026-08-25 更新）：https://kreo.net/
5. Togal.AI 官网（Togal Assemblies/Togal.CHAT）：https://www.togal.ai/
6. 造价通首页（AI 询价入口/数智询价/信息价/数聚超市定价）：https://www.zjtcn.com/ ；其 AI 询价子站 https://ds.zjtcn.com/ 为动态页，**功能细节未核实**
7. 造价HOME（造价 Ai 速答/套定额 AI 助手/定额自动套 2026/江苏 2026 定额）：http://www.zaojiahome.com/
8. 品茗科技新闻（AI+基建 2026-08-28、AI+招标采购研讨会 2026-07-17）：https://www.pinming.cn/news_cont_7096.html ；品茗云资料 V2026「资料AI助手」：https://www.pmgd.cn/?m=home&c=View&a=index&aid=456 （2026-06-30）
9. 新点软件 1.62 亿大模型项目：凤凰网财经/东方财富 2024-06-19 报道（原文链接未留存，经 360 搜索检索核实）
10. 新中大六和 AI 训推平台/浙建造价问答模块：中国建设新闻网 2025-04-22、剑鱼标讯候选人公示 2025-12-02（原文链接未留存）
11. 鲁班软件 AI 跃迁报道：网易 2026-07-15（原文链接未留存）
12. GB/T 50500-2024（2024-11-26 发布、2025-09-01 实施、强转推荐、责任转移）：江西省住建厅贯彻执行通知 2025-07-18（jxjst.gov.cn，原文链接未留存）；头条对比文 2026-03-30（经 360 搜索检索）
13. 从业者叙事：服务新干线头条《AI 取代造价员？这五个误区该破了》2026-08-10；造价通建识《AI 时代来临，造价员何去何从？》2026-03-10（原文链接未留存，经 360 搜索检索）
14. 斯维尔（清华系）2026 年 AI 造价新动作：**未核实**（官网 www.thsware.com 为 JS 渲染，正文抓取失败）

## 3. AI 底座 · 记忆与知识库（v4.11.0 基线复扫 · 2026-08-31）

### 市场格局 · 最新动态

开源记忆基础设施在 2026 年完成「从向量库到记忆 OS」的跃迁，梯队清晰：

- **Mem0**（64.4k stars）2026-04 发布新记忆算法：单次 ADD-only 提取（无 UPDATE/DELETE）、实体链接、多信号检索（语义 + BM25 + 实体）、时间推理；LoCoMo 92.5 / LongMemEval 94.4，迁移指南显示开源版进入 v3 [1]。BM25+语义混合检索已成行业标配。
- **MemOS 2.0「星尘」**（11.1k stars）：L1 轨迹 / L2 策略 / L3 世界模型三层记忆，MemCube 可组合记忆立方体（支持隔离与受控共享），MemScheduler 异步调度；2026-05 memos-local-plugin 2.0 实现 100% 本地 SQLite，2026-08-17 接入 DeepSeek Harness。README 未提及遗忘机制，最接近的是自然语言「记忆反馈与修正」[2]。
- **Letta（MemGPT）**大改版：主仓库转为落地页，活跃开发移至 letta-code（agent harness + 桌面应用），Letta Cloud 提供跨设备同步 agent 记忆/身份/会话；记忆固化为可被 agent 用工具编辑、可跨 agent 挂载的 memory blocks [3][4]。
- **Zep Graphiti**（30.4k stars）：双时态知识图谱 + 自动事实失效，语义+BM25+图遍历混合检索，v0.17 自定义图驱动、Kuzu 弃用、嵌入式 FalkorDB Lite 降低部署门槛，仅自托管 [5]。
- **大厂原生记忆**：ChatGPT Memory 为「显式记住指定信息 + 回忆过往对话」双开关（后者 2025-04 上线），2025-09 起 ChatGPT Pulse 依据聊天历史生成晨报 [7]；Gemini 2025-02 向 Advanced 订阅者开放回忆过往对话，2026-03 Pixel Drop 推出 Gemini 代办与 Magic Cue 等主动个性化功能，2026-05 发布 Gemini Spark 代理 [8][9]；Claude 走客户端文件型 memory tool（/memories 目录，view/create/str_replace/insert/delete/rename 六命令，存储归应用方，可跨会话并与 compaction 组合）[6]。ChatGPT/Gemini 记忆在 2026 年的专项升级细节：**未核实**（官方帮助页不可访问、搜索引擎反爬）。
- **本地知识库**：RAGFlow（89.7k stars）2025-12 给 agent 加 Memory 组件、2026-06 支持飞书/Discord/Telegram 等渠道 [11]；AnythingLLM（65.4k stars）上线智能技能选择（宣称省 80% token）、MCP 支持，桌面版内置 LanceDB+本地嵌入的零配置 RAG [12]。

### 范式迁移（上轮调研以来的变化）

1. **记忆架构化**：提取、固化、调度、检索成为独立工程层（Mem0 算法层、MemOS L1-L3+调度器、Graphiti 双时态图），「存下对话摘要」不再是记忆产品。
2. **遗忘被重新定义为「失信 + 复核」而非删除**：LangChain 的 OpenWiki 自纠错记忆（2026-08-25）用「主张 ↔ 代码证据版本」锚点做确定性过期检测，过期主张标记保留、验证后修正或刷新，形成自纠错回路 [13]；Graphiti 的自动事实失效同思路。
3. **GraphRAG 降温、agentic RAG 成为默认**：Microsoft GraphRAG 官宣进入维护模式、不再接受新功能 PR [10]；RAGFlow 以 agentic workflow 为主线并内置 Memory。重型批量图谱构建让位于轻量动态图 + agent 按需检索。
4. **记忆所有权/可移植性成为行业叙事**：Harrison Chase《Own your intelligence》（2026-07-25）明确提出「不拥有 context 与记忆，就不拥有积累的智能」，要求学习成果可跨模型、跨系统迁移 [14]；Mem0 已为 Claude Code/Cursor/Codex 提供「让 harness 记住你的项目」插件 [1][15]。
5. **主动式记忆从概念到产品**：ChatGPT Pulse、Gemini Spark 均以「记忆 + 历史」主动送达，记忆竞争从「记得准」转向「主动给」。

### 对 gaea 的机会与威胁

**威胁**：
- 大厂把「记得你 + 主动服务」变成默认体验，用户预期水涨船高；gaea 做梦 2.0 的主动预取尚未做，正好卡在这条新预期线上。
- MemOS 本地插件 2.0（100% 本地 + 三层记忆 + 智能去重 + Memory Viewer）与 gaea 三脑叙事正面重叠，且有论文与基准背书；MemCube 的「隔离 + 受控共享」是「记忆分区」最强公开先例——gaea「双空间」独创性被削弱。但注意：各家分区均按 user/session/agent 边界（Mem0 三级、Letta shared blocks、MemCube），未见按「生活领域」（工作 vs 角色人格）分区且互不检索的先例，gaea 的隔离语义仍独特。
- LoCoMo/LongMemEval 已成通用记分牌，gaea 无公开可复现指标，宣传缺第三方背书。

**机会**：
- 「记忆所有权」叙事正热，gaea 天然本地优先，可把「可查看、可删除、可导出迁移」做成对大厂云记忆的差异化卖点（个人版「遗忘权」）。
- 开源方案多需 Neo4j/Qdrant 等服务栈（MemOS docker compose、Graphiti），gaea 的单机 SQLite + BM25+语义零配置路线在 Windows 个人机上部署门槛最低。
- GraphRAG 进维护模式印证重型图谱不适合个人机；gaea 宜做实体链接 + 时态失效的「轻图谱」，而非全文档社区摘要。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

**0-3 月**：
- 做梦 2.0（DistillMerge）补齐冲突消解语义：借鉴 OpenWiki 的「主张-证据」思路，两条记忆冲突时并存并标记待复核，而非静默覆盖；蒸馏结果按 Mem0 新算法的 LoCoMo/LongMemEval 口径跑一组公开可复现评测。
- 上线「主动预取」MVP：开机/空闲时把高频工作记忆预装配进上下文预算（对标 Pulse 晨报形态，但纯本地）。
- 记忆管理面板补全：单条查看/编辑/删除 + 全量导出（JSON/Markdown），把「遗忘权」落成功能。

**下个 3-6 月**：
- 实体链接 + 轻时态失效（借鉴 Graphiti 双时态）：知识库与记忆分区实体互链，过期事实自动标记 stale。
- fileindex 与记忆检索合流：个人文件与对话记忆走统一混合检索，context 预算调度从雏形升级为显式「召回→评分→预算装配」管线。
- 双空间引入 MemCube 式「受控共享」白名单（如工作空间可引用知识分区、角色空间不可），把隔离从设计主张升级为可验证机制。

**愿景 6-12 月**：
- 记忆可移植性：gaea 记忆格式可导出（对齐 Mem0/JSON-LD）、跨机迁移、支持导入大厂商聊天导出，呼应「own your memory」叙事。
- 对齐 MemOS 的 L2 策略记忆（个人工作流偏好）与 L3 世界模型（个人知识图谱）；本地小模型驱动的后台做梦升级为「睡眠时固化」（sleep-time 式反思 + 主动预取联动）。

### 参考来源

1. Mem0 GitHub（新算法 2026-04、基准、v3）：https://github.com/mem0ai/mem0
2. MemOS GitHub（2.0 星尘、L1-L3、MemCube、本地插件）：https://github.com/MemTensor/MemOS
3. Letta GitHub（letta-code、Letta Cloud 同步）：https://github.com/letta-ai/letta
4. Letta 文档 · Memory Blocks：https://docs.letta.com/guides/agents/memory
5. Zep Graphiti GitHub（双时态图、事实失效、FalkorDB Lite）：https://github.com/getzep/graphiti
6. Anthropic memory tool 文档：https://platform.claude.com/docs/en/agents-and-tools/tool-use/memory-tool
7. Wikipedia · ChatGPT（Memory 双开关、Pulse 晨报）：https://en.wikipedia.org/wiki/ChatGPT
8. Wikipedia · Gemini（回忆过往对话、Spark 代理）：https://en.wikipedia.org/wiki/Gemini_(chatbot)
9. Google Blog · 2026-03 Pixel Drop（Gemini 代办/Magic Cue）：https://blog.google/products-and-platforms/devices/pixel/march-2026-pixel-drop/
10. Microsoft GraphRAG GitHub（维护模式声明）：https://github.com/microsoft/graphrag
11. RAGFlow GitHub（Memory 组件、渠道接入）：https://github.com/infiniflow/ragflow
12. AnythingLLM GitHub（技能选择、MCP、桌面本地 RAG）：https://github.com/Mintplex-Labs/anything-llm
13. LangChain Blog · Building Self-Correcting Memory in OpenWiki（2026-08-25）：https://www.langchain.com/blog/self-correcting-memory-openwiki
14. LangChain Blog · Own your intelligence（2026-07-25）：https://www.langchain.com/blog/own-your-intelligence
15. Mem0 文档（Claude Code/Cursor/Codex 插件、自托管）：https://docs.mem0.ai

## 4. 模型中心（v4.11.0 基线复扫 · 2026-08-31）

### 市场格局 · 最新动态

**国产 API 生态：旗舰变贵、廉价/免费档扩容、缓存与峰谷计费普及。** DeepSeek 定价页现以 V4 系列为主（V4-Flash-0731、V4-Pro-0813、V4-Flash-Vision-Exp），1M 上下文 / 384K 输出，输入缓存命中价约为未命中的 1/31，非高峰时段再五折，并提供 Anthropic 兼容端点（https://api-docs.deepseek.com/quick_start/pricing）。智谱 GLM 已迭代到 GLM-5.3/5.2/5.1，GLM-5.3-Flash 促销价 $0.075/$0.25（至 2026-09-09），GLM-4.7-Flash、GLM-4.5-Flash、GLM-4.6V-Flash 完全免费，缓存输入另有约 0.19× 的专价（https://docs.z.ai/guides/overview/pricing）；GLM-5.2 主打 1M 无损上下文与长程 Coding Agent（https://docs.bigmodel.cn/cn/guide/models/text/glm-5.2）。Kimi 旗舰 K3 为 1M 上下文、缓存命中 $0.30 / 未命中 $3.00 / 输出 $15.00，另有 K2.7 Code 与 K2.6（视觉），Moonshot V1 于 2026-08-31 全平台下线，域名迁至 platform.kimi.ai（https://platform.kimi.ai/docs/pricing/chat、https://platform.kimi.ai/docs/pricing/chat-k3）。阿里百炼自身已成多厂商全模态货架：qwen3.8-max/3.7-plus 之外还上架 deepseek-v4-pro、kimi-k3、glm-5.3，并统一提供生图（qwen-image-3.0-pro、wan2.7-image-pro）、视频、TTS/ASR/Realtime、embedding/rerank（https://help.aliyun.com/zh/model-studio/models）。

**编码套餐订阅化。** GLM Coding Plan 改为积分制：Lite/Pro/Max 三档按「5 小时积分 + 周积分」计（2,000/12,000/28,000 与 10,000/60,000/140,000），非高峰时段积分五折抵扣、宣称最高省 92%；套餐覆盖 GLM-5.3 与 GLM-5.3-Flash，调用旧名自动切换，支持 Claude Code、OpenClaw、OpenCode、TRAE、CodeBuddy 等工具（https://docs.bigmodel.cn/cn/coding-plan/overview）。

**桌面客户端全面 Agent 化。** Cherry Studio 8 月连发 v2.0.x 多版：三种 agent 会话运行时（pi、DeepSeek Harness「dsh」、Claude Agent SDK）、MCP prompts/resources 进入输入框、MCP-over-HTTP（/v1/mcps）、供应商目录免发版热更新、统一模型连通性检测、用量分析改用 ECharts、本地模型 DirectML/CoreML 加速（https://github.com/CherryHQ/cherry-studio/releases）。LobeChat 品牌升级为 LobeHub，定位「下一代 Agent harness」「7×24 小时 Agent 运营」（https://lobehub.com/zh）。AnythingLLM 主打 on-device、本地私密（https://anythingllm.com/）。LM Studio 搜索摘要显示其出现「Bionic 工作与代码智能体」及云模型、LM Link 等入口（官网抓取超时，细节未核实）。开源个人 AI 助理 OpenClaw 成为现象级产品（https://openclaw.ai），智谱推出「AutoClaw（澳龙）——国内首款一键安装的本地 OpenClaw 客户端」，内置 50+ Skills 与 AutoGLM 浏览器能力（https://www.zhipuai.cn/zh/about）。

### 范式迁移（上轮调研以来的变化）

1. **计费模型从「按量单价」变「订阅积分 + 峰谷时段 + 缓存命中」三维**：GLM Coding Plan 积分制与自动换模型、DeepSeek 峰谷五折、各家缓存专价（上轮基线均无）。
2. **prompt 前缀缓存计费成头部引擎标配**：Anthropic 读取 0.1×、写入 1.25×（5 分钟）/2×（1 小时），自动与显式断点并存（https://platform.claude.com/docs/en/build-with-claude/prompt-caching）；OpenAI 默认开启、GPT-5.6+ 读 0.1×/写 1.25×、门槛 1024 token（https://developers.openai.com/api/docs/guides/prompt-caching）；DeepSeek、Kimi、GLM 均有缓存价（见上）。
3. **通用语义缓存产品化退潮**：GPTCache 明示不再适配新 API/模型，且存在误命中问题（https://github.com/zilliztech/GPTCache）；主流路径转向引擎原生前缀缓存计费。
4. **智能路由产品化成熟**：OpenRouter Auto Router 按任务分类（约 30 类）+ 社区 7 天消费份额排名 + cost_tier 档位路由、不收附加费（https://openrouter.ai/docs/guides/routing/routers/auto-router），provider 路由默认按价格负载均衡（https://openrouter.ai/docs/features/provider-routing）；LiteLLM 内置 cost-based/latency-based 等策略（https://docs.litellm.ai/docs/routing）；聚合网关 new-api（46.8k stars）提供渠道加权、失败重试与含缓存命中的用量计费统计（https://github.com/QuantumNous/new-api）。
5. **多模态目录统一管理在平台/网关层已常态化**（new-api 聚合 chat/image/audio/video/embedding/rerank/realtime；百炼多厂商货架），桌面端以 Cherry「目录热更新 + 统一连通性检测」最接近，但「chat/生图/语音/embedding 一处管理」仍无桌面客户端标杆。

### 对 gaea 的机会与威胁

**机会**
- gaea 的 GLM coding=端点适配面临失真风险，也是升级契机：编码套餐已积分制化、旧模型名自动切到 GLM-5.3，静态模型目录若不热更新，模型名与用量口径都会过时（来源同上 bigmodel.cn）。
- 「成本仪表」在桌面端仍是空白位：Cherry 用量分析刚重设计、聚合网关才有缓存命中统计，gaea 做「按端点/属性的用量 + 缓存命中率 + 积分余量」即可差异化。
- 本地-云端自动路由 v1 不必做语义缓存：以「隐私/本地优先 → 缓存友好 → 峰谷错峰」为目标函数即可吃到 2026 计费红利（各家缓存价差 3–10 倍、DeepSeek 峰谷 2 倍）。

**威胁**
- Cherry Studio v2 的统一连通性检测、目录热更新、用量分析直接覆盖 gaea 模型中心核心体验，且 8 月连发 5 版迭代极快。
- OpenClaw/AutoClaw + 编码套餐把个人用户的「引擎选择」收编为订阅权益（且注明 OpenClaw 走次级调度/尽力交付），桌面助手的模型接入层被上游平台化。
- 旗舰单价高企（K3 输出 $15/M）+ 积分额度制增加用量统计与告警复杂度；coding 套餐「仅限指定工具」存在限制第三方客户端的政策风险（未核实是否长期收紧）。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

**0-3 月**
1. 成本仪表 v0：按引擎/云端属性/端点聚合 token、缓存命中率、估算成本；GLM coding 端点按「积分」口径展示并显示 5 小时/周额度余量与非高峰五折提示。
2. 静态模型目录改可热更新，建立 GLM 5.x 别名映射（5.2/5.1→5.3 自动切换的对端展示）。
3. 连接测试升级：chat ping 连发两次验证前缀缓存命中，把「缓存可用性」纳入云端属性。

**下个 3-6 月**
4. 本地-云端自动路由 v1：目标函数=本地优先 → 缓存命中最大化 → 峰谷价，采用「成本档位」设计（参考 OpenRouter cost_tier）而非复杂打分器。
5. prompt 前缀稳定性工程：固定系统提示词与记忆段前缀，吃满各家 0.1× 缓存价（注意各家最低缓存门槛 1024 token 起）。
6. 多模态目录统一 UI：chat/生图/TTS/ASR/embedding/rerank 一张引擎卡管理（对齐 new-api 类型聚合口径）。

**愿景 6-12 月**
7. 面向委托式任务的配额调度：编码套餐积分在多任务/子任务间的分配与熔断。
8. 本地语义缓存仅作实验特性评估（GPTCache 教训：误命中、停止适配新模型），默认不做。

### 参考来源

- DeepSeek API 定价（V4/缓存/峰谷/Anthropic 端点）：https://api-docs.deepseek.com/quick_start/pricing
- Z.ai GLM 按量定价与免费档：https://docs.z.ai/guides/overview/pricing
- GLM Coding Plan 套餐概览（积分/峰谷/工具支持）：https://docs.bigmodel.cn/cn/coding-plan/overview
- GLM-5.2 模型文档（1M 无损上下文）：https://docs.bigmodel.cn/cn/guide/models/text/glm-5.2
- 智谱关于页（AutoClaw 本地 OpenClaw 客户端）：https://www.zhipuai.cn/zh/about
- Kimi 定价索引（K3/K2.7 Code/K2.6、V1 下线）：https://platform.kimi.ai/docs/pricing/chat
- Kimi K3 定价（缓存命中价）：https://platform.kimi.ai/docs/pricing/chat-k3
- 阿里云百炼模型列表（多厂商全模态货架）：https://help.aliyun.com/zh/model-studio/models
- Cherry Studio Releases（v2.0.x，2026-08）：https://github.com/CherryHQ/cherry-studio/releases
- LobeHub 官网与文档（Agent harness 定位）：https://lobehub.com/zh 、https://lobehub.com/zh/docs/usage/start
- AnythingLLM（on-device 定位）：https://anythingllm.com/
- LM Studio（Bionic/云模型/LM Link，仅搜索摘要，未核实）：https://lmstudio.ai/login 、https://lm-studio.cn/
- OpenClaw 官网与文档：https://openclaw.ai 、https://docs.openclaw.ai/zh-CN
- OpenClaw 趋势参考（2026-02 指南）：https://zhuanlan.zhihu.com/p/2002485126714644013
- OpenRouter Provider Routing：https://openrouter.ai/docs/features/provider-routing
- OpenRouter Auto Router：https://openrouter.ai/docs/guides/routing/routers/auto-router
- LiteLLM Router 策略：https://docs.litellm.ai/docs/routing
- new-api 网关（聚合/计费/缓存命中统计）：https://github.com/QuantumNous/new-api
- GPTCache（维护状态与局限）：https://github.com/zilliztech/GPTCache
- Anthropic Prompt Caching 计费：https://platform.claude.com/docs/en/build-with-claude/prompt-caching
- OpenAI Prompt Caching 计费：https://developers.openai.com/api/docs/guides/prompt-caching

## 5. 编程板块（v4.11.0 基线复扫 · 2026-08-31）

> 调研方法说明：本轮通过 GitHub API 实测仓库活跃度（星标/最后推送日期）、直接抓取官网与 CHANGELOG 核实，中文搜索引擎结果受合规过滤较多，个别项标注「未核实」。调研日期 2026-08-31。

### 市场格局 · 最新动态

- **收敛为「一个底座、多个表面」**：终端 CLI agent（Claude Code、Codex CLI、OpenCode、Gemini CLI 等）成为事实底座，IDE（Cursor、Devin Desktop、Trae）与异步云端围绕其分层。GitHub 于 2026-02-04 起 Agent HQ public preview 直接接入 Claude 与 Codex，2026 年 6-7 月又追加 agent apps、Issues 内 agent 自动化控制——平台枢纽开始聚合第三方 agent（[1][2]）。
- **异步云端常态化**：OpenAI Codex 已五面一体（ChatGPT 桌面/Web、CLI、IDE 扩展、cloud、Remote），支持长任务、通知、由 Gmail/Slack/GitHub 事件触发的定时任务（2026-08），agent harness 开源（[7]）。Cognition 方面，windsurf.com 已 308 重定向至 devin.ai/desktop（2026-08-31 实测），Devin Desktop 定位「coding agent 之家」、可导入 VS Code/Cursor 配置；Windsurf 独立品牌是否完全退役**未核实**（[3][4]）。
- **中国侧**：Trae 拆分为 TraeCode + TraeWork（AI 办公平台），积分制四档约 ¥45-699/月，内置 Seed/GLM-5.2 模型，云端任务并行 2-20 个（[20][21]）；通义灵码免费、Lingma IDE 全面公测、未见 CLI 形态（[22]）；Kimi 推出 Kimi Code 订阅；围绕 Claude Code/Codex 的 API 中转站生态（约官方价 10-38%）成为壳窗软件的主要赞助来源（[9]）。

### 范式迁移（上轮调研以来的变化）

1. **「壳窗 + 外部 CLI」从民间黑箱走向官方协议**。Claude Code v2.1.x CHANGELOG（2026-08 在查）显示官方提供 `claude attach / logs / stop / respawn / rm` 后台会话管理、Remote Control 客户端实时流式工具调用、SessionStart 钩子返回会话 staleness 与重缓存成本、`/usage` 花费限额条、Claude Desktop 跨会话消息投递（[8]）。第三方壳窗的定位从「破解式包装」变为「官方接口的桌面客户端」，技术风险显著下降。
2. **壳窗供给繁荣但商业模型脆弱**。活跃样本：cc-switch（13.0 万星，Tauri，管理 8 种引擎）、claudecodeui/CloudCLI（1.35 万星，多引擎+手机/Web）、CodePilot（归藏，6.4 千星，17+ 供应商、手机遥控）、desktop-cc-gui、codeg、Nimbalyst（Crystal 后继，个人免费+iOS 伴侣）、Happy、Conductor（Mac 专属，$50/月 SaaS 化+企业版）（[9]-[17]）。收缩样本：opcode（2.24 万星）自 2025-10 停更；Crystal 停更；Vibe Kanban 公司运营关停、转社区开源（[15][16][23]）。存活路径共性：多引擎聚合、免费开源靠赞助、并入更大的个人助手（OpenClaw 38.8 万星，[18]），或 SaaS 化收云费用。
3. **「管家式」体验有直接对标**：会话恢复（Nimbalyst 原生 resume 全历史 [14]；claude-mem 9.3 万星做跨会话持久上下文 [19]）；完成通知/远程遥控（Happy [17]、Nimbalyst iOS、CodePilot 手机控制 [11]、Conductor mobile 官网标注 coming soon [16]）；健康总览（Conductor「一眼看到各 agent 在做什么」、Codex 通知与长任务 [7]）。

### 对 gaea 的机会与威胁

- **决策验证**：2026 年格局下「编程板块保持 DSH 壳窗、不并入工位、不共享工具面、不做原生工作台」仍然成立且更稳——官方协议化让壳窗更可靠；同时 opcode/Crystal/Vibe Kanban 三个停更案例证明独立壳窗的商业与维护风险真实存在。gaea 只做体验层、不承担引擎维护成本，符合收敛后的产业分工。
- **机会**：现有壳窗都在卷「多引擎并行与切换」，面向个人用户的「低焦虑管家层」——健康徽标、断连自愈、完成通知进入通知中心、与个人记忆弱联结——仍是空位；gaea 的中文办公场景与「工位/乐园」隔离可延伸出「编程会话不污染办公空间」的差异点。
- **威胁**：①官方客户端上移，Claude Desktop 已能跨会话投递/回复 Claude Code 消息、Codex 与 ChatGPT 桌面一体化，官方壳会挤压第三方壳的存在感（[7][8]）；②注意力分流，中文圈 10+ 免费壳窗与 OpenClaw 类个人助手直接竞争「个人助手管编程」的心智（[9][11][18]）；③中转站 + cc-switch 组合使用成本极低，编程板块若无管家价值难以留人（[9]）。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **现在 0-3 月（体验加固对标官方语义）**：断连自愈对齐 `claude attach/respawn` 与 `--resume` 行为；外链唤起打通 deeplink→终端/CLI；健康徽标读取进程存活 + SessionStart staleness 钩子；复述并冻结「不并轨、不共享工具面」决策（[8]）。
- **下个 3-6 月**：任务完成系统级通知（对标 Happy / Nimbalyst iOS 伴侣）；只读会话列表与一键恢复入口；适配清单扩展至 Codex CLI、OpenCode、Gemini CLI 等多引擎（[9][14][17]）。
- **愿景 6-12 月**：完成事件进入 gaea 通知中心并与个人记忆弱联结；观望 Remote Control 协议是否成为壳窗标准接口；仍不做原生编程工作台。

#### 未核实项
- GitHub Copilot Agent HQ 的 2026 年采用数据；Claude Code 是否有官方桌面 GUI（未搜到，但 CHANGELOG 已见 Claude Desktop 与其会话互通）；Windsurf 品牌退役程度；CodeBuddy 2026 现状（官网抓取为空）。

### 参考来源

1. https://github.blog/news-insights/company-news/pick-your-agent-use-claude-and-codex-on-agent-hq/ （2026-02-04）
2. https://github.blog/news-insights/company-news/welcome-home-agents/ （2025-10-28）
3. https://devin.ai/desktop （另：windsurf.com → devin.ai/desktop 308 重定向，2026-08-31 实测）
4. https://docs.devin.ai/zh/desktop/getting-started
5. https://claude.com/pricing （Pro $17-20/月、Max $100 起、Claude Code 全付费计划含、5 小时滚动+周限额、Managed Agents $0.08/session-hour）
6. https://cursor.com/pricing （Hobby 免费 / Pro $20 / Pro+ / Ultra 20x / Teams $40）
7. https://learn.chatgpt.com/docs （Codex 五表面、事件触发定时任务、开源 harness）
8. https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md （v2.1.251）
9. https://github.com/farion1231/cc-switch （13.0 万星，ccswitch.io，API 中转站赞助生态）
10. https://github.com/siteboon/claudecodeui （CloudCLI，多引擎，pushed 2026-08-27）
11. https://github.com/op7418/CodePilot （6.4 千星，17+ 供应商，BSL-1.1）
12. https://github.com/zhukunpenglinyutong/desktop-cc-gui （Tauri 多引擎桌面端）
13. https://github.com/xintaofei/codeg （多 agent 会话聚合工作区）
14. https://nimbalyst.com/crystal/ （Crystal 停更、Nimbalyst 后继，个人免费）
15. https://www.vibekanban.com/ （公司运营关停、转社区开源，2.8 万星）
16. https://conductor.build/ 及 https://conductor.build/pricing （Free/$50 Pro/$60 Teams/企业）
17. https://happy.engineering/ （Claude Code & Codex Remote Control）
18. https://github.com/openclaw/openclaw （38.8 万星，pushed 2026-08-31）
19. https://github.com/thedotmack/claude-mem （9.3 万星，跨会话持久上下文）
20. https://docs.trae.cn/ide_plans-and-billing （积分制 ¥45-699/月）
21. https://work.trae.cn/ （TraeWork AI 办公平台）
22. https://lingma.aliyun.com/ （通义灵码免费、Lingma IDE 公测）
23. https://github.com/winfunc/opcode （2.24 万星，pushed 2025-10-16，停更）

## 6. 创作包 · 小说 + 绘梦（v4.11.0 基线复扫 · 2026-08-31）

### 市场格局 · 最新动态

**小说创作**：海外 Sudowrite 已把「自研模型 + 资产库」做成完整闭环——Muse 1.5 专为小说散文训练、支持章节级输出与 agentic 工作流（[Muse 页，2026-08-29](https://sudowrite.com/muse)）；Story Bible/Story Engine 把角色、世界观、大纲作为每次生成的"记忆"，第三方评为"同类最扎实"；2026-08 上线 Ballad 1.1（批量生成省最高 70% 积分）与 Visualize 场景配图（2,500 积分/张）（[评测，2026-08-29 更新](https://www.varoo.cn/tools/sudowrite.html)；[定价页](https://sudowrite.com/pricing)）。定价 $10/$22/$44 三档，Max 档积分 12 个月滚动、年付约五折。国内蛙蛙写作（自研 Weaver 模型）主打角色卡 + 角色记忆模式（生成时自动校验初始设定防 OOC）与 Agent 工作流，并打通"小说→剧本→漫剧视频（自动角色绘制、镜头拆分、配音）"；按字数计费（约 ¥9.9/万字、¥68.9/50 万字包、3/7 天无限卡），无月卡（[评测指南，2026-01-21](https://aiproducthub.cn/sites/wawawriter-ai-smart-writing-assistant-tutorial.html)；[官网](https://wawawriter.com/app/)）。NovelAI 发布 Diffusion V5："更锐利细节、多角色、整页漫画一个模型"，多角色分离控制 + Vibe Transfer + 局部重绘（[官网](https://novelai.net/)）。基线报告中的阅文作家助手·妙笔（千万字级理解）仍为平台锁定型（内部基线：`docs/gaea-competitive-landscape-2026.md`）。彩云小梦旧域名 301 跳转至 [xiaomengai.com](https://www.xiaomengai.com/)（页面 JS 渲染，现状未核实）；笔灵 AI 2026 年动态未核实（多轮检索未获有效结果）。

**图像生成**：字节 Seedream 5.0 Pro（官网 © 2026；知乎测评 2026-07-09 与"GPT Image 2"对标）把生成+编辑统一架构推进到交互式编辑：响应空间标记与涂鸦线稿、支持图层拆分，可产出故事板/信息图等高密度版式（[官方页](https://seed.bytedance.com/zh/seedream5_0_pro)；Seedream 4.0 统一架构见 [官方页 2025-09](https://seed.bytedance.com/zh/seedream4_0)；[知乎测评（403，据检索摘要）](https://zhuanlan.zhihu.com/p/2058627990901241271)）。Gemini Nano Banana 已列名 Gemini API 官方文档（2026-08-24），Gemini 3.1 Pro 主打设计意图理解（[API 文档](https://ai.google.dev/gemini-api/docs?hl=zh-cn)；[DeepMind](https://deepmind.google/models/gemini/pro/)）；"GPT Image 2"仅见于上述知乎标题，未直接核实。智谱侧 GLM-Image 已升为图像旗舰（复杂指令遵循、文字渲染突出），CogView-4 为常规档、CogView-3-Flash 免费，模型概览未见独立图像编辑模型（[BigModel 文档](https://docs.bigmodel.cn/cn/guide/start/model-overview)）。开源侧 ComfyUI 官网主推 App Mode 简化工作流、降低新手门槛，仍是本地可控性代表（[comfy.org](https://comfy.org/)）；即梦海外版 Dreamina 开始聚合 Seedream/Seedance/GPT 等多家模型（[官网](https://dreamina.capcut.com/zh-tw/)）。可灵、通义万相 2026 动态未核实（检索受限）。

### 范式迁移（上轮调研以来的变化）

1. **编辑范式**：局部重绘/蒙版 →「指令编辑 + 图层化」成为云端旗舰标配（Seedream 5 Pro 图层拆分与批注改图、Gemini 对话式编辑）；gaea 对 img2img 的"诚实拒绝"对应的正是这一缺口。
2. **一致性**：从单图一致性走向「多角色同图 + 整页漫画」（NovelAI V5）与「角色记忆模式」（蛙蛙）；角色资产从写作工具附属品升级为文图共用资产。
3. **文图联动**：Sudowrite Visualize、蛙蛙"小说→剧本→漫剧视频"、Seedream 5 Pro 故事板，说明"叙事→分镜→成片"管线已进入头部产品；个人端"角色库 ↔ 一致性插画"闭环仍稀缺。
4. **计费与留存**：纯月订阅弱化，转向按量积分 + 滚动（Sudowrite 12 个月滚动、NovelAI Anlas 可纯按次购买）、年付大折（约五折）、字数包/无限日卡（蛙蛙，刻意无月卡）、"无限档 + 公平使用限额"（NovelAI V5 附加限额）；提示词/工作流共享社区成为留存抓手。行业系统性留存率数据未核实。

### 对 gaea 的机会与威胁

**机会**：①「创作间」把角色/世界观做成一套共用数据，恰好命中行业从"文本资产"走向"文图共用资产"的方向，是双模块合并的天然护城河；②本地优先 + URL 统一转 data URL 落盘，契合创作者版权敏感（蛙蛙/Sudowrite 均以"内容归用户、不用于训练"为核心卖点）；③智谱 GLM-Image 旗舰 + CogView-3-Flash 免费档便于分层接模型、控成本；④乐园空间 + 个人长期记忆可做出"越用越懂你的人物与世界设定"，是云端通用工具没有的粘性来源。
**威胁**：①云端旗舰（Seedream 5 Pro/Gemini）的指令编辑与一致性能力断层领先，gaea 绘梦除文生图外近乎空白，差距在拉大；②蛙蛙类产品已覆盖"文→图→视频"全链且定价透明，中文网文作者迁移成本低；③通用大模型（长上下文 + 免费/低价）持续挤压轻量续写、润色类功能的独立价值。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **0-3 月**：绘梦模型目录分层接入 GLM-Image（旗舰质量）与 CogView-3-Flash（免费兜底）；角色库/设定页字段一键注入生图提示词（角色卡→提示词模板，对齐 Story Bible 思路）；小说侧上线低成本高感知的「设定/伏笔一致性检查」。
- **3-6 月**：实验多角色同图与简单分镜排版（对标 NovelAI V5 多角色控制）；跟进 ComfyUI App Mode 趋势，评估本地引擎升级或工作流导入可行性；绘梦引入用量（张数/积分）统计与"创作消耗报告"，为将来计费与留存设计铺垫。
- **愿景 6-12 月**：创作间「叙事→插画→分镜漫画」管线（对标蛙蛙漫剧视频与 NovelAI 整页漫画）；世界观/角色资产跨模块复用（轻语角色扮演、小说封面生成）；若商业化，采用"积分 + 滚动 + 年付折扣"组合而非硬月费（2026 年观察到的主流范式）。

### 参考来源

- Sudowrite 定价：https://sudowrite.com/pricing （2026-08 核实）
- Sudowrite Muse 模型页（2026-08-29）：https://sudowrite.com/muse
- Sudowrite 中文评测（2026-08-29 更新，含 Muse 1.5/Story Bible/Visualize/Ballad 1.1）：https://www.varoo.cn/tools/sudowrite.html
- 蛙蛙写作评测指南（2026-01-21）：https://aiproducthub.cn/sites/wawawriter-ai-smart-writing-assistant-tutorial.html
- 蛙蛙写作官网：https://wawawriter.com/app/
- NovelAI 官网（Diffusion V5）：https://novelai.net/
- Seedream 5.0 Pro 官方页：https://seed.bytedance.com/zh/seedream5_0_pro
- Seedream 4.0 官方页（2025-09，生成+编辑统一架构）：https://seed.bytedance.com/zh/seedream4_0
- Seedream 5.0 Pro 知乎测评（2026-07-09，403 仅据检索摘要）：https://zhuanlan.zhihu.com/p/2058627990901241271
- Gemini API 文档（Nano Banana，2026-08-24）：https://ai.google.dev/gemini-api/docs?hl=zh-cn
- Gemini 3.1 Pro：https://deepmind.google/models/gemini/pro/
- 智谱 BigModel 模型概览（GLM-Image/CogView-4/CogView-3-Flash）：https://docs.bigmodel.cn/cn/guide/start/model-overview
- ComfyUI 官网（App Mode）：https://comfy.org/
- Dreamina（即梦海外版，多模型聚合）：https://dreamina.capcut.com/zh-tw/
- 彩云小梦跳转目标（现状未核实）：https://www.xiaomengai.com/
- gaea 内部基线报告：`docs/gaea-competitive-landscape-2026.md`（2026-08-29）

> 调研说明：本节基于 2026-08-31 的网络检索与页面核实；DuckDuckGo 被验证码拦截、部分 Bing 中文查询被污染或过滤、知乎/打开 AI 官网返回 403，故笔灵 AI、可灵、通义万相 2026 年动态与行业留存率数据均标注「未核实」，未编造任何产品或功能。

## 7. 轻语 · 人格陪伴（v4.11.0 基线复扫 · 2026-08-31）

> 调研方法说明：本节基于 2026-08-31 的公开网页检索（TechCrunch、网信办官网、维基百科、aicpb 榜单、厂商官网等）；中文搜索引擎结果受合规过滤与分词影响，部分国内产品细节标注「未核实」。

### 市场格局 · 最新动态

**中国：监管落锤是本周期最大变量。** 国家网信办等五部门 2026-04-10 发布《人工智能拟人化互动服务管理暂行办法》（令第 21 号），**2026-07-15 起施行**：禁止向未成年人提供「虚拟伴侣等虚拟亲密关系服务」；不得以诱导情感依赖、沉迷为服务目标；出现自残自杀迹象须干预并联络监护人/紧急联系人；连续使用每超 2 小时提醒；敏感个人信息用于训练须「单独同意」；注册超 100 万或月活超 10 万须安全评估并报省级网信，算法备案+年度核验，罚款上限 10–20 万元（[网信办全文](https://www.cac.gov.cn/2026-04/10/c_1777558395078289.htm)）。其征求意见稿于 2025-12-27 发布；据 21 世纪经济报道，2025 年 11 月国内头部月活为：星野 488 万、猫箱 472 万、X EVA 181 万、筑梦岛约 60 万，星野+Talkie 前 9 个月收入约 1.2 亿元；筑梦岛 2025 年 6 月因低俗内容被上海网信办约谈（[SFCCN](https://m.sfccn.com/2025/12-29/xNMDE0NDlfMjA5MjQxNw.html)）。施行后国内平台整改的公开报道**未核实**。

**海外：诉讼与未成年人禁入重塑产品形态。** Character.AI 于 2025-10-29 宣布自 11-25 起禁止未满 18 岁用户使用开放聊天，未成年人改用互动「Stories」（[TechCrunch](https://techcrunch.com/2025/11/25/character-ai-will-offer-interactive-stories-to-kids-instead-of-open-ended-chat/)、[Wikipedia](https://en.wikipedia.org/wiki/Character.AI)）；2026-01 Google 与 Character.AI 就青少年死亡案启动首批和解谈判（[TechCrunch](https://techcrunch.com/2026/01/07/google-and-character-ai-negotiate-first-major-settlements-in-teen-chatbot-death-cases/)）；2026-05 宾州医务委员会因聊天机器人冒充医生起诉（[TechCrunch](https://techcrunch.com/2026/05/05/pennsylvania-sues-character-ai-after-a-chatbot-allegedly-posed-as-a-doctor/)）。海外 web 流量榜（2026-06）：Character.AI 1.94 亿居首，JanitorAI 1.39 亿（+14.9%）快速逼近，成人向的 SpicyChat/polybuzz/Candy.ai 占据前五多数席位，Talkie 仅 652 万且环比下滑（[aicpb](https://aicpb.com/ai-rankings/products/ai-character-rankings)）——纯陪伴 web 端明显「灰度化、成人化」。

### 范式迁移（上轮调研以来的变化）

1. **从「沉浸陪伴」转向「内容+陪伴」：** Character.AI 2026 年上半年人均月使用超 950 分钟（Sensor Tower 数据，[TechCrunch](https://techcrunch.com/2026/07/09/character-ai-enters-the-microdrama-arena-with-its-own-productions-but-with-a-twist/)），7 月推出 AI 微短剧「c.ai Series」（18+ 观众可边看边与角色对话）、音频系列 c.ai FM、写作工具 c.ai Reads，并陆续上线 Lorebook/Books——头部平台在把自己变成「可对话的内容厂」而非纯聊天。
2. **资本面进入整合期：** MiniMax 2026-01-09 港股上市（SEHK:100），2025 年收入 7900 万美元、净亏 18.7 亿美元（[Wikipedia](https://en.wikipedia.org/wiki/MiniMax_(company))）；个性化陪伴应用 Dot 于 2025-09 关停（[TechCrunch](https://techcrunch.com/2025/09/05/personalized-ai-companion-app-dot-is-shutting-down/)）；Replika 2025 年换帅、用户超 4000 万，创始人 Kuyda 转做新公司（[Wikipedia](https://en.wikipedia.org/wiki/Replika)）。全球 AI 陪伴应用 2025 年内购收入约 1.2 亿美元（[TechCrunch](https://techcrunch.com/2025/08/12/ai-companion-apps-on-track-to-pull-in-120m-in-2025/)）；2026 上半年全球收入口径**未核实**。
3. **「长期记忆+主动关心+情感语音」在海外已成卖点组合：** Nomi 官网明确主打「Human-Level Memory（短/中/长期记忆）」「Proactive Nomi Messaging（用户离开一段时间后主动发消息）」「Emotive Voice Chats（情绪化语音通话，语调随情绪变化）」（[nomi.ai](https://nomi.ai/)）——这正是轻语规划中的三件事，海外独立厂商已产品化，但属厂商自述，实际效果口碑**未核实**。国内星野/猫箱 2026 年在记忆与主动关心上的功能更新**未核实**（其官网仍以移动端「沉浸式智能体社区」为主打，[xingyeai.com](https://www.xingyeai.com/)）。

### 对 gaea 的机会与威胁

**机会：**
- 《暂行办法》适用「面向境内公众提供」的服务，gaea 个人本地使用、不面向公众运营、无内置氪金，**天然落在适用范围之外**；而办法让云端陪伴产品背上防沉迷、单独同意、备案评估等成本，反向放大「数据不出机 + 无氪金 + 硬隔离」的差异化价值。
- 未成年人被整体排除出虚拟伴侣后，成年敏感用户对「不上传聊天记录、可随时彻底删除」的需求上升，本地记忆管线从卖点升级为刚需；SillyTavern（32.8k GitHub stars，纯本地、AGPL，「不提供任何在线服务」，[GitHub](https://github.com/SillyTavern/SillyTavern)）证明本地陪伴存在真实但偏极客的需求——**大众级桌面本地陪伴产品仍未发现**，「是否伪需求」无反证，但大众化证据也**未核实**。
- 头部转向内容化与灰度化后，「长期连续人格 + 看得见你的真实生活」的冷静陪伴是空位。

**威胁：**
- 若轻语未来做人格包社区共享、公开分发，可能触发办法中的应用商店责任与安全评估条款；「主动关心」若做成持续打扰，与立法要求的「便捷退出、不得以持续互动阻碍退出」精神冲突，需克制设计。
- 海外免费强产品（Character.AI 950 分钟/月黏性、Nomi 三件套）拉高了用户对记忆与语音的预期，轻语的情感语音若滞后太久会失去宣称空间。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **0-3 月：** 补齐合规护栏与产品叙事——在轻语界面提供「非真人提示」「会话一键清空/导出删除」「连续时长提醒」开关（与办法条款对齐，即使不适用也构成卖点）；把「乐园/工位硬隔离 + 本地记忆」写进轻语首屏文案，对标 Nomi 的 memory/proactive/voice 三件套做差异化表。
- **下个 3-6 月：** 落地关系记忆图谱与克制的主动关心（仅限用户开启、频次封顶、一键静音，避开「诱导依赖」红线）；上线情绪化 TTS 语音（复用节奏引擎的 PAD 标尺驱动语调），参考 Nomi「语调随情绪变化」的表述做验收标准。
- **愿景 6-12 月：** 探索「陪伴×真实生活」的独占位（陪伴人格可感知你的工作文档完成度并温和追问，这是纯娱乐云产品做不到的）；暂缓人格包公开分享等可能触发平台责任的功能，观察《暂行办法》施行后首批执法与国内头部整改动作再定。

### 参考来源

- 网信办：《人工智能拟人化互动服务管理暂行办法》全文（2026-04-10 发布，2026-07-15 施行）— https://www.cac.gov.cn/2026-04/10/c_1777558395078289.htm
- 21 世纪经济报道：AI 情感陪伴新规征求意见稿与国内产品月活（2025-12-29）— https://m.sfccn.com/2025/12-29/xNMDE0NDlfMjA5MjQxNw.html
- TechCrunch：Character.AI 专题（诉讼/儿童 Stories/微短剧时间线）— https://techcrunch.com/tag/character-ai/
- TechCrunch：宾州医务委员会起诉 Character.AI（2026-05-05）— https://techcrunch.com/2026/05/05/pennsylvania-sues-character-ai-after-a-chatbot-allegedly-posed-as-a-doctor/
- TechCrunch：Google/Character.AI 和解谈判（2026-01-07）— https://techcrunch.com/2026/01/07/google-and-character-ai-negotiate-first-major-settlements-in-teen-chatbot-death-cases/
- TechCrunch：c.ai Series 微短剧与 950 分钟/月（2026-07-09）— https://techcrunch.com/2026/07/09/character-ai-enters-the-microdrama-arena-with-its-own-productions-but-with-a-twist/
- Wikipedia：Character.AI — https://en.wikipedia.org/wiki/Character.AI
- Wikipedia：Replika — https://en.wikipedia.org/wiki/Replika
- Wikipedia：MiniMax（2026-01-09 港股上市、2025 财报）— https://en.wikipedia.org/wiki/MiniMax_(company)
- aicpb：AI Character Rankings（2026-06，网站访问量口径）— https://aicpb.com/ai-rankings/products/ai-character-rankings
- Nomi.ai 官网（记忆/主动消息/情感语音卖点）— https://nomi.ai/
- SillyTavern GitHub（32.8k stars，本地角色扮演前端）— https://github.com/SillyTavern/SillyTavern
- TechCrunch：Dot 关停（2025-09-05）— https://techcrunch.com/2025/09/05/personalized-ai-companion-app-dot-is-shutting-down/
- TechCrunch：AI 陪伴应用 2025 年收入约 1.2 亿美元（2025-08-12）— https://techcrunch.com/2025/08/12/ai-companion-apps-on-track-to-pull-in-120m-in-2025/
- 星野官网 — https://www.xingyeai.com/

> 调研方法：cn.bing / GitHub API / 官方文档站实际检索与抓取（2026-08-31）。国内搜索引擎对「微信机器人/封号」话题结果受合规过滤，相关结论已标注未核实。

## 8. 触点层 · 微信 + 语音 + 指令中枢（v4.11.0 基线复扫 · 2026-08-31）

### 市场格局 · 最新动态

**端到端实时语音 API 已成红海，国产可直连且计费透明。** OpenAI gpt-realtime 于 2025-08-28 GA：音频输入 $32/百万 tokens、输出 $64/百万 tokens，32k 上下文，支持 WebRTC/WebSocket/SIP（developers.openai.com/api/docs/models/gpt-realtime）。国内三家可直连：①阿里百炼 Qwen3.5-Omni-Realtime（plus/flash）：WebSocket/WebRTC/AOQ 三协议，支持 semantic_vad 语义打断、对话内语音控制（"语速快一些"）、113 语种识别与声音复刻；官方实测单轮总响应约 5.1 秒；音频 token 换算输入=秒×7、输出=秒×12.5，单价在控制台（help.aliyun.com/zh/model-studio/realtime）。②智谱 GLM-Realtime（flash-9B/air-32B）：wss 接入，server/client VAD、实时打断、Function Calling；按分钟计费，flash 音频 0.18 元/分钟、air 0.3 元/分钟（docs.bigmodel.cn/cn/guide/models/sound-and-video/glm-realtime.md）。③字节豆包实时语音模型 3.0「Seeduplex」原生全双工，wss duplex 协议 + response.cancel 打断，支持 8 种方言与复刻音色，2026-04 发布（volcengine.com/docs/6561/2549778；zhuanlan.zhihu.com/p/2025601158207521490）；API 单价未核实。MiniMax 实时语音 API 细节未核实，仅见 Music 模型 2026-08-20 停服公告（platform.minimaxi.com/subscribe/token-plan）。

**微信个人号 bot：hook 类灰色生态收缩，iLink 生态走强。** hook 类代表 WeChatFerry 已归档停更（6.8k★，最后推送 2026-07，github.com/lich0821/WeChatFerry），wechaty 更新停滞（2025-12 后无推送，github.com/wechaty/wechaty）；而面向 AI Agent 的「微信 iLink Bot SDK」持续活跃（622★，github.com/corespeed-io/wechatbot）。未发现微信官方「个人微信 AI 助理」合规通道（未核实，搜索结果受过滤）。企业微信官方转向拥抱 Agent：wecom-cli 2026-03-29 开源，「让人类和 AI Agent 都能在终端中操作企业微信」，长连接机器人可装 skills，覆盖消息/文档/会议/待办，官方明示幻觉与数据外泄风险（open.work.weixin.qq.com/help2/pc/21676；github.com/WecomTeam/wecom-cli，2.9k★）。

**「任何入口唤起同一个助理」已成默认玩法。** 开源侧 OpenClaw（38.8 万★，"Any OS. Any Platform."）与 hermes-agent（23.9 万★）把多入口个人助理做成基础设施（api.github.com，2026-08-31 检索）；产品侧腾讯元宝推电脑版「随时随地唤起、一键划词」，与手机 App、微信小程序构成多端（yuanbao.tencent.com/evt/promo/23c396e6dc400a72ee75384474bcb04a）；ChatGPT Windows 主打读取应用/文件/屏幕上下文（apps.microsoft.com/detail/9plm9xgg6vks）。

### 范式迁移（上轮调研以来的变化）

上轮基线（2025 末）：Realtime API 多为半双工 + 静音阈值 VAD，ASR+LLM+TTS 拼接管线仍是主流。2026 年四个迁移：①级联架构向端到端统一网络「全面转型」（k.sina.cn/article_7879923802_1d5ae185a06801g1ru.html，2026-08-03）；②全双工 + 语义打断成旗舰标配：Seeduplex（2026-04）、NVIDIA 开源 PersonaPlex 与 VoiceChat 11B（插话即让出话语权，cloud.tencent.com/developer/news/3809407；sohu.com/a/1061284680_122396381）、哈工大 Lychee-FD（2026-07，163.com/dy/article/L1VD13P30511AQHO.html）；③工程难点从「压延迟」转向 Voice Runtime——连续交互的回合状态管理（juejin.cn/post/7660896500881866804，2026-07-12）；④语义 VAD 普及：附和声/背景音不再误触发打断（Qwen realtime 文档）。

### 对 gaea 的机会与威胁

机会：gaea 的 Realtime 代码（openai provider、16k→24k 重采样、TurnControl）与主流事件协议高度同构（commit / cancel / speech_started），换引擎边际成本低；GLM 按分钟计费使个人成本可预算（0.18 元/分钟≈月重度使用十元级）；微信 iLink 通道与 OpenClaw 生态同构，且图片链路真机定稿是稀缺资产；wecom-cli 提供合规的「第二 IM 入口」。威胁：①Whisper+LLM+TTS 拼接管线相对端到端已可感落后一代，打断自然度差距最直观；②微信通道政策风险未除——hook 类项目归档说明通道可能随时失效，封号 2026 新政未核实，须有降级预案；③OpenClaw/hermes 在「多入口同一助理」上生态先发；④元宝把「桌面唤起 + 划词」做成免费国民级功能，抬高了桌面入口心智门槛。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **0-3 月**：真机验证 Realtime 全链路（先跑通 openai provider 与打断）；指令内核先落 Ctrl+K 命令面板；继续微信语音/视频消息抓包取样；成本口径入库（GLM 按分钟、OpenAI/Qwen 按 audio token）。
- **3-6 月**：接入第二个国产 provider——优先评估 Qwen3.5-Omni-Realtime（semantic_vad/语义打断与 TurnControl 对齐）或 GLM-Realtime（按分钟计费可预算）；打断体验升级到语义级；试点 wecom-cli 作为办公/团队入口；为 iLink 通道加风控（白名单、节流、人格分句播报的拟人化节流）与封号降级预案。
- **6-12 月**：全双工/免唤醒连续交互实验；「任何入口、任何模态唤起同一个 gaea」对标 OpenClaw 多平台与元宝多端完成体验闭环。

### 参考来源

1. OpenAI gpt-realtime 模型文档：https://developers.openai.com/api/docs/models/gpt-realtime
2. 阿里云百炼 Qwen-Omni 实时模型文档：https://help.aliyun.com/zh/model-studio/realtime
3. 智谱 GLM-Realtime 指南（含价格）：https://docs.bigmodel.cn/cn/guide/models/sound-and-video/glm-realtime.md
4. 智谱 GLM-Realtime AsyncAPI（wss 协议）：https://docs.bigmodel.cn/cn/asyncapi/realtime.md
5. 火山引擎豆包端到端实时语音（全双工版本）：https://www.volcengine.com/docs/6561/2549778
6. Seeduplex 官方页：https://seed.bytedance.com/zh/seeduplex ；发布文：https://zhuanlan.zhihu.com/p/2025601158207521490
7. MiniMax Token Plan（Music 停服公告）：https://platform.minimaxi.com/subscribe/token-plan
8. WeChatFerry（已归档）：https://github.com/lich0821/WeChatFerry ；wechaty：https://github.com/wechaty/wechaty
9. 微信 iLink Bot SDK for OpenClaw：https://github.com/corespeed-io/wechatbot
10. 企微 wecom-cli 帮助文档：https://open.work.weixin.qq.com/help2/pc/21676 ；仓库：https://github.com/WecomTeam/wecom-cli
11. 腾讯元宝电脑版推广页：https://yuanbao.tencent.com/evt/promo/23c396e6dc400a72ee75384474bcb04a
12. ChatGPT Windows（微软商店）：https://apps.microsoft.com/detail/9plm9xgg6vks
13. 级联→端到端转型报道：https://k.sina.cn/article_7879923802_1d5ae185a06801g1ru.html
14. GPT-Live 全双工语音 Agent 拆解：https://juejin.cn/post/7660896500881866804
15. NVIDIA PersonaPlex：https://cloud.tencent.com/developer/news/3809407 ；VoiceChat 11B：https://www.sohu.com/a/1061284680_122396381
16. Lychee-FD 全双工开源：https://www.163.com/dy/article/L1VD13P30511AQHO.html
17. 豆包/字节 API 价格汇总（2026-08-30 更新）：https://apirank.vip/zh/providers/bytedance/
