# gaea 3.0 愿景规划 · 调研报告 R5：中文办公 AI 市场（办公板块）

> **背景**：为 gaea 3.0 的「办公板块」收集公开信息。办公板块定位 = 本地 agent 工作台（42 工具），核心能力 = 中文办公文档生产（方案/标书/汇报/成本测算的草拟与拼装）、docx/xlsx/pdf 解析转换（纯本地）、知识库、成本库、交付物/任务管理、MCP 扩展、记忆中枢，走"本地优先 + 分层智能（云端统筹规划 + 本地执行）"路线。
> **调研时间**：2026-08（本报告所有数据截至检索日，产品功能/市场数字随时间变化，引用时注意核对年份）
> **方法**：中文 web 检索（每主题多轮）+ 关键页面精读（官方发布稿 / 官方帮助中心 / 新华网·央广网·证券时报·钛媒体·36氪·极客公园·智东西等权威/行业媒体 / 艾媒·智研·毕马威等机构报告 / GitHub 开源项目）。
> **范围**：① 中文办公 AI 市场格局（WPS AI / 飞书 / 钉钉 / 腾讯文档）；② 标书/方案/汇报自动生成细分工具；③ 办公 agent / 文档智能体与私有化部署方案；④ 中文用户办公 AI 心智（格式合规 / 数据顾虑 / 本地化接受度）；⑤ 本地 agent 文档工作台的空白与机会。

---

## 摘要：10 条核心结论（速览）

1. **四大巨头都在做"AI 文档"，但都锚定云端协作平台，且"一键生成"质量不可用是行业公论**：WPS AI 3.0（灵犀，2025-07）、飞书 aily/多维表格 AI（2025-07 升级）、钉钉 AI 钉钉 1.0（2025-08）、腾讯文档接入 DeepSeek-R1（2025-02）均已把生成/解析/知识库/Agent 做成功能点，但均为"云上账号 + 在线文档"形态，且行业普遍承认生成内容"看似炫酷但不实用、甚至不能用"（金山办公田然原话）。来源：[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)、[36氪](https://m.36kr.com/p/3371623528452615)、[中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html)、[腾讯文档接入DeepSeek](https://www.fjtv.net/folder331/2025-02-17/6474905.html)
2. **标书/方案/汇报是真实存在的垂直市场，但产品碎片化**：华胜天成"投标大王 2.0"（2025-10）、开源易标书 OpenBidKit、AI 标书精灵、链企招投标文本生成算法等专做标书；Kimi AI PPT / Gamma / AhaSlides 主打"一键 PPT"；AI 周报/述职/汇报工具多为个人开源或插件，尚未出现"中文文档工作台"级别的统一产品。来源：[华胜天成](https://www.teamsun.com.cn/newsdetail.htm?id=2510090001)、[易标书GitHub](https://github.com/fb208/OpenBidKit_Yibiao)、[Kimi PPT助手](https://ai.cdsu.edu.cn/info/1055/1032.htm)
3. **标书 AI 的痛点不在"能不能写"，而在格式合规与风险**：AI 生成标书面临废标风险、串标嫌疑、格式无法被招标系统读取（"你写的标书，AI 可能根本读不到！"）、以及生成内容需人工逐项核验等四大法律风险；这正是"纯本地 + 格式保真 + 可溯源"的价值空间。来源：[知乎·合规自检5维度](https://zhuanlan.zhihu.com/p/2052792741961130963)、[阿里云开发者·合规全景](https://developer.aliyun.com/article/1754065)、[知乎·AI读不到标书](https://zhuanlan.zhihu.com/p/2062565575071643068)
4. **2026 年桌面办公 Agent 赛道爆发，巨头已集体入场**：腾讯 WorkBuddy（日活 1300 万+，2026-06 访问量 2097 万次居国内桌面办公智能体之首）、月之暗面 Kimi Work（2026-06-03 上线）、阿里 QwenWork（2026-08-03 公测）、字节 TRAE Work、华为云 OfficeClaw/OfficeAce（2026-04 邀测）；艾媒统计仅半年国内就有超 20 款国产桌面办公智能体发布。来源：[钛媒体·桌面办公Agent卡位战](https://www.tmtpost.com/8099969.html)、[艾媒白皮书](https://www.iimedia.cn/c400/113198.html)、[智东西·OfficeClaw](https://m.zhidx.com/p/549631.html)
5. **私有化/本地化是政企硬需求，"数据不出域"正倒逼 Agent 架构从云端转向全栈本地化**：银行（农发行 4 万员工全内网 RPA）、保险（渤海财险 ChatBI 数据零外流）、政务（甘肃武威"公文易办"私有化政务大模型，2026-07 投用）均要求数据不出内网；Gartner 预测到 2027 年超 60% 大型企业将在受监管环境采用私有化 AI 智能体。来源：[实在智能·数据不出域](https://www.ai-indeed.com/encyclopedia/27690.html)、[央广网·WPS AI私有化](https://tech.cnr.cn/techph/20250529/t20250529_527189297.shtml)、[国际在线·甘肃政务大模型](https://news.cri.cn/2026-07-07/1bb7f2f1-a854-460d-a31a-dd2306b7f135.html)
6. **中文用户心智：AI 使用率高但信任度低，"反复调整指令"是最大痛点**：智联招聘《职场"人机共生"调研报告》（2025-09）显示近八成职场人每周使用 AI 工具、56.1% 愿为 AI 付费，但"反复调整指令"成为最大痛点；毕马威《2026 全球科技报告》指出 88% 企业试点 Agentic AI、仅 24% 在多种场景实现 ROI，媒体直言"没有企业敢让 AI 独立完成一份关键文件的书写"。来源：[智联招聘](https://www.ceweekly.cn/zxfb/2025/0925/481469.html)、[钛媒体](https://www.tmtpost.com/8099969.html)
7. **文档格式保真与可编辑性是 AI 办公产品的公认短板**：华为云 OfficeClaw 总结现有 PPT 工具三大痛点——版式与内容遵从性差、模型幻觉导致信息失真、内容及图表不可编辑；WPS 灵犀把"格式保留、修改可控"作为核心卖点（识别解析数千种格式组合）。来源：[智东西](https://m.zhidx.com/p/549631.html)、[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)
8. **市场规模：中国 AI+办公软件 2025 年约 700 亿元，AI 智能体市场 804 亿元（+123.2%），2030 年预计 6968 亿元**；协同办公市场近 400 亿元。市场在增长，但办公智能体使用场景仍高度集中在 PPT 生成（51.8%）、表格数据分析（38.2%）、批量文件整理（37.1%）三件"轻活"上，重文档生产（标书/方案/公文/测算）尚未被有效覆盖。来源：[观研天下](https://www.gonyn.com/industry/1776655.html)、[艾媒](https://www.iimedia.cn/c400/113198.html)、[第一财经](https://www.yicai.com/news/102714109.html)
9. **微软 Copilot 在中国市场的受限状态反而让出空间**：Copilot 企业版 2024-03 进入中国但联网/搜索/文生图受限、私有化中文支持度不高，2024-10 更终止中国区部分个人服务；国产厂商因此获得"中文 + 私有化 + 信创"的错位窗口。来源：[钛媒体](https://www.tmtpost.com/6981843.html)、[湖南日报](https://www.hunantoday.cn/news/xhn/202410/20854422.html)
10. **"本地 agent 文档工作台"正处于形态验证期，尚未有产品同时覆盖"中文重文档生产 + 纯本地解析转换 + 知识库/成本库 + 交付物管理"**：主流玩家要么是云端协作平台的 AI 附加层（WPS/飞书/钉钉/腾讯文档），要么是通用桌面 Agent（WorkBuddy/Kimi Work/OfficeClaw），要么是专做单点的标书工具；gaea 办公板块的差异化组合（42 工具 + 本地 docx/xlsx/pdf 转换 + 成本库 + 交付物管理 + 分层智能）对应真实空白。来源：[钛媒体](https://www.tmtpost.com/8099969.html)、[智东西](https://m.zhidx.com/p/549631.html)

---

## 一、中文办公 AI 市场格局：四大巨头做到什么程度

### 1.1 WPS AI / WPS 灵犀（金山办公）——"原生 Office 智能体"

**形态演进（官方口径）**：
- **2023 年 WPS AI 1.0**：围绕 AIGC（内容创作）、Copilot（智慧助理）、Insight（知识洞察）三大方向推出系列 AI 功能，嵌入 WPS 各大组件（文字/表格/演示）。
- **2024 年 WPS AI 2.0**：聚焦企业特定场景，为组织构建"企业大脑"，用 AI 促进企业知识智能化应用。
- **2025 年 7 月 27 日 WPS AI 3.0「WPS 灵犀」**（WAIC 2025 发布，获"镇馆之宝"奖）：定位"原生 Office 智能体"，用户通过自然语言、多轮对话即可完成文档创作、演示文稿生成及语音助手，全程无需外部跳转；左 Office 套件、右灵犀的同屏交互形态，AI 直接修改左侧文档。来源：[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)、[WPS论坛](https://bbs.wps.cn/topic/59892)

**灵犀具体能力边界**：
- 文档：多轮对话修改、格式保留（识别解析数千种格式组合，保留图文混排/复杂表格/多级列表，无需手动二次排版，"出品即成品"）、合同风险提示、文档规范性审核。
- AI PPT：边聊边改大纲、二次精调模板/单页/版式，支持生成演讲稿、演讲视频；官方明确批评"一键生成 PPT 落地难"。
- 灵犀语音助手（移动端）：与文档"聊天"，从几百页财务报告提取关键信息等。
- WPS 知识库：把云文档升级为知识库，搜答案、筛数据，基于私域知识写方案/稿子/汇报——即"私人知识银行"。
- 规模：截至 2025-03 底 WPS Office 全球月度活跃设备 6.47 亿。来源：[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)

**私有化方案（2025-05-29 发布，西安 WPS 365 AI 办公中国行）**：
- WPS AI 主体能力完全支持私有化：AI 写作助手、AI 阅读助手、AI 数据助手、AI 设计助手、智能文档库、AI 会议助手、AI 邮箱助手、AI 公文助手。
- 适配 8 种国产操作系统、鲲鹏/海光 CPU、麒麟/统信 OS、达梦/人大金仓数据库；自研政务大模型适配国产 GPU。
- 部署方式：支持一站式全套私有化，也支持 AI 能力组件式集成。
- 文档洞察引擎：多模态解析 PDF/DOC/XLX/PPT（含图片表格对象），让大模型与企业高价值信息交互。
- 文档权限控制体系：只能检索权限范围内信息、问答可溯源。
- 落地案例：中国绿发（私有化 AI 智能办公平台"AI 同志"）；申万宏源（私域文档智能知识平台，财富经理知识获取效率 +80%、对客延时至少 -3 倍）；招商证券（数据助手/设计助手/知识库部署到企业服务器，覆盖投行尽调、研报撰写、合规审核，1 万+ 员工）。来源：[央广网](https://tech.cnr.cn/techph/20250529/t20250529_527189297.shtml)、[中国新闻网·广东](http://www.gd.chinanews.com.cn/2025/2025-06-02/442319.shtml)

**政务落地**：2026-07 甘肃省首个政务场景私有化智能公文大模型"公文易办"在武威市投用（WPS 365 助力），公文处理达"分钟级交付"（文稿生成/校对）；另有 WPS 365 政务版、金山政务 AI 一体机（金山云）。来源：[甘肃日报](https://gansu.gansudaily.com.cn/system/2026/07/09/031397643.shtml)、[新民周刊](https://m.xinminweekly.com.cn/content/47189.html)、[金山云](https://www.ksyun.com/cms/news/738.html)、[百度百科·WPS 365政务版](https://baike.baidu.com/item/WPS%20365%E6%94%BF%E5%8A%A1%E7%89%88/68029251)

**对 gaea 的意义**：WPS 灵犀证明了"多轮对话 + 格式保留 + 修改可控"才是中文文档 AI 的正确交互范式（而非一键生成）；但其能力绑定 WPS 生态与云端账号，本地 docx/xlsx/pdf 的独立转换、跨格式拼装与交付物管理不在其主线内。

### 1.2 飞书（字节跳动）——"智能伙伴 + aily 智能体平台 + 多维表格 AI"

**形态演进**：
- 2024 年发布"飞书智能伙伴"，逐步叠加技能/长期记忆/工作流 AI Agent 节点（智能体可通过飞书 wiki/知识库获得长期记忆、通过工作流 AI Agent 节点编排自动化）。来源：[飞书帮助中心·了解技能](https://www.feishu.cn/content/s855fpkr)、[飞书·智能体长期记忆](https://zhipu-ai.feishu.cn/wiki/Wt7OwHtO1irztnkkDztcae63nRf)、[飞书·工作流AI Agent节点](https://www.feishu.cn/hc/zh-CN/articles/643175485940)
- 2025-07-09 飞书发布会（CEO 谢欣）：发布并升级知识问答、AI 会议、飞书 Aily、飞书妙搭等多款 AI 产品；推出"AI 应用成熟度模型"（M1 概念验证 → M4 完全成熟），知识问答已达 M3、飞书妙记达 M4。来源：[36氪](https://m.36kr.com/p/3371623528452615)、[飞书帮助中心](https://www.feishu.cn/hc/zh-CN/articles/429896178269)

**aily 智能体平台**：
- 定位"企业版 Manus"：几步配置人设、接入企业知识、添加业务系统 MCP，即可打造专属 Agent；核心差异化是"拥有企业数据"，支持私域数据 + 数据权限与隔离。
- 2025-12 aily 工作助手上线"任务模式"：从"问答辅助"进化到"成果交付"，自主拆解复杂任务（深度调研/数据可视化/生成报告/建多维表格/生成网页），使用独立云端计算机，交付飞书云文档、可视化报告、多维表格、图片/播客；支持定时任务 7×24 后台运行（竞品追踪、财报分析、工作总结），完成后消息推送。来源：[中国日报](https://ex.chinadaily.com.cn/exchange/partners/82/rss/channel/cn/columns/sz8srm/stories/WS694108aaa310942cc4996e93.html)、[飞书帮助中心·快速了解aily](https://www.feishu.cn/hc/zh-CN/articles/790732948604)
- 落地案例：公牛集团"公牛智服"专家级客服 Agent，24 小时响应客户与上千家经销商，客服接待能力提升 30 倍。

**多维表格 AI（最被看好的 Agent 试验场）**：
- 月活超 1000 万；单表容量提升至 1000 万热行（较 2024 年翻 10 倍），2 万行加载 0.94 秒、5 万行 1 秒。
- "一句话生成业务系统"：对话式智能搭建助手 + 飞书妙搭（对话生成 AI 应用）+ 应用模式（一键"装修"成可交互系统）。
- 2025-07 宣布将在企微、钉钉平台上线多维表格，与钉钉 AI 表格正面竞争。来源：[36氪](https://m.36kr.com/p/3371623528452615)、[飞书官方文章](https://www.feishu.cn/content/article/7588081416752106693)、[南方+](https://static.nfnews.com/content/202507/12/c11502361.html)

**能力边界判断**：飞书把 AI 深度绑定"协同套件 + 私域数据"，Agent 能调用文档/会议纪要/多维表格；但交付物形态以飞书生态内产物为主（云文档/多维表格/网页），本地 docx/xlsx/pdf 的高保真生产与转换并非其重心。来源：[中国日报](https://ex.chinadaily.com.cn/exchange/partners/82/rss/channel/cn/columns/sz8srm/stories/WS694108aaa310942cc4996e93.html)

### 1.3 钉钉（阿里巴巴）——"AI 钉钉 1.0 / Agent OS"

**形态演进**：
- 2025-08-25 钉钉十周年发布会，陈航（无招）回归发布 8.0 暨 **AI 钉钉 1.0**："Agent 是 AI 钉钉的灵魂"，宣告全面进入 AI 原生阶段，推出超 10 款 AI 产品。来源：[中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html)、[极客公园](https://www.geekpark.net/news/353059)
- 2025-12-23 发布 AI 钉钉 1.1 版本"木兰"，定位"全球首发 AI 工作智能操作系统（Agent OS）"，从协同工具走向 Agent OS。来源：[东方财富](https://finance.eastmoney.com/a/202512233599866277.html)、[中国工业新闻网](https://www.cinn.cn/sz/2025/12-25/W1OP20mr.html)、[百度百科·木兰](https://baike.baidu.com/item/%E6%9C%A8%E5%85%B0/67117812)

**五大 AI 原生产品（8.0）**：
- **钉钉 One**：消息/审批/日程/会议四大 Agent 将散落信息整理为信息流卡片，"事找人"；语音指挥 Agent 发起会议、汇总数据、整理项目进展。
- **AI 搜问**：融合 AI 搜索 + 知识问答，基于权限范围自动汇总聊天/表格/CRM 生成结构化结果；AIFusion 引擎汇集全球 50 余种大模型，支持海外模型与**本地私有模型接入**（满足央国企/金融机构数据合规）。
- **AI 表格**：服务超 30 万家企业；AI 表格助理（自然语言生成表格/自动化工作流/仪表盘）+ 100 余款字段 Agent（多模态/OCR）+ O-Table 存算一体架构（百万行实时更新）。来源：[智东西·钉钉AI表格](https://zhidx.com/p/491079.html)
- **AI 听记**：1 亿小时音频训练的大模型，支持 30 余种方言、140+ 语言、200+ 行业术语，转写准确率 97%，36 类场景模板，纪要自动同步待办与 AI 表格。
- **DingTalkA1 硬件**：3.8mm 厚、6 麦克风阵列、6nm AI 芯片的"AI 硬件入口"。
- 客服 Agent 案例：100% 服务记录全量检测，客户满意度 30%→80%、成本下降 90%。来源：[中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html)

**能力边界判断**：钉钉强在"组织内的 Agent 化工作流"（审批/会议/表格/客服），文档生产偏轻（AI 表格是核心武器）；私有模型接入为政企数据合规预留了通道，但文档本体（docx/标书/方案）生产与本地解析转换不是钉钉的主线。来源：[中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html)、[智东西·钉钉AI表格](https://zhidx.com/p/491079.html)

### 1.4 腾讯文档 / 腾讯办公矩阵——"接入大模型 + WorkBuddy"

**腾讯文档 AI**：
- 2025-02 宣布接入 DeepSeek-R1（满血版）：直接生成文档、PPT；上线 PPT 直出、周报神器、文献速读等功能。来源：[环球网](https://3w.huanqiu.com/a/074633/4LWSkaVFIZi)、[驱动之家](https://m.mydrivers.com/newsview/1030689.html)、[AI Top 100](https://www.aitop100.cn/infomation/details/21553.html)
- 腾讯五大协同办公产品（文档/会议/企微/乐享/问卷等）AI 升级，"从单点提效迈向全流程智能"。来源：[腾讯云社区](https://cloud.tencent.cn/developer/article/2524521)
- 2025-09 极客公园梳理"腾讯 AI 的新叙事"：AI 能力全面注入办公矩阵（腾讯文档/腾讯会议/企业微信）。来源：[极客公园](https://w.geekpark.net/news/354095)
- 腾讯云 AI 办公套件在金融行业落地（协同效率与项目交付）。来源：[腾讯云](https://cloud.tencent.com.cn/developer/article/2677160)

**腾讯 WorkBuddy（桌面办公智能体，2026 年重点）**：
- 前身为 CodeBuddy；定位"生态整合型"桌面办公 Agent：完全兼容 OpenClaw 技能体系、内置 20+ Skills 技能包与 MCP 协议，支持混元/DeepSeek/GLM/Kimi 等大模型。
- 数据：第三方统计日活突破 1300 万；易观统计 2026-06 国内主流桌面端办公智能体访问量中 WorkBuddy 以 2097 万次居首（超 TRAE IDE 1279 万 + QoderWork 788 万之和）。
- 关键节点：2026-07-25 上架鸿蒙电脑应用市场（鸿蒙首个桌面办公智能体）；2026-07-30 V5.3.5 联合腾讯文档推出"人机双写"；2026-08 腾讯云发布 WorkBuddy 企业版与办公智能体套件；2026-08 更新支持零代码生成网页（HTML 编辑如 Word）。来源：[钛媒体·桌面办公Agent卡位战](https://www.tmtpost.com/8099969.html)、[极客公园](https://www.geekpark.net/news/365512)、[品玩](https://www.pingwest.com/a/314412)、[站长之家](https://www.chinaz.com/2026/0813/1770890.shtml)
- 政务落地："全国首个"腾讯版 WorkBuddy 入职政务系统，支持国产芯片。来源：[17173新闻](https://news.17173.com/content/06172026/070056000.shtml)

**能力边界判断**：腾讯把 AI 文档能力做进腾讯文档（在线），同时用 WorkBuddy 切入"桌面 + 本地技能 + 大模型自由接入"；WorkBuddy 是目前国内用户规模最大的桌面办公智能体，但其技能重心在通用文件/网页/表格处理，非中文重文档（标书/公文/测算）的垂直生产。

### 1.5 小结：四大巨头"AI 文档"能力矩阵对照

| 维度 | WPS AI 灵犀 | 飞书 aily/多维表格 | 钉钉 AI 钉钉 | 腾讯文档/WorkBuddy |
|---|---|---|---|---|
| 文档生成 | 强（原生 Office，格式保留） | 中（云文档产物） | 中（偏纪要/表格） | 中（DeepSeek 直出 + 人机双写） |
| 文档解析 | 强（文档洞察引擎，私有化） | 中（妙记/知识问答） | 中（AI 听记/搜问） | 中 |
| 知识库 | WPS 知识库（云文档升级） | 私域数据 + 权限隔离 | AI 搜问（AIFusion 接入私有模型） | 腾讯文档/乐享 |
| Agent 化 | 原生 Office 智能体（灵犀） | aily 智能体平台 + 任务模式 | Agent OS + 四大 Agent | WorkBuddy（桌面）+ 办公智能体套件 |
| 私有化/信创 | 强（WPS AI 私有化方案、政务版） | 弱（云原生） | 中（私有模型接入选项） | 中（WorkBuddy 政务版/国产芯片） |
| 本地 docx/xlsx/pdf 转换 | 中（客户端有本地能力） | 无（纯云端） | 无（纯云端） | WorkBuddy 桌面端部分本地 |

**共同边界**：四家都把 AI 当作"云上协作平台的附加能力"，文档生产绑定各自生态（在线文档/云盘），格式保真依赖各家渲染引擎；没有任何一家把"纯本地 docx/xlsx/pdf 解析转换 + 中文重文档拼装（标书/方案/成本测算）+ 独立交付物管理"作为产品主线。

---

## 二、"标书 / 方案 / 汇报自动生成"细分工具

### 2.1 一键 PPT 类（海外与国产）

**Kimi AI PPT（月之暗面）**：
- Kimi 内置 PPT 助手：一句话生成 PPT，支持大纲编辑、模板/风格切换，被高校/职场教程广泛使用。来源：[Kimi PPT 助手](https://ai.cdsu.edu.cn/info/1055/1032.htm)、[火山引擎社区](https://developer.volcengine.com/articles/7473796651687346213)
- 2026 年推出基于 Google Nano Banana Pro 的 AI 幻灯片生成器（48 小时限时免费试用）。来源：[站长之家](https://www.chinaz.com/ainews/23241.shtml)、[AI Top 100](https://www.aitop100.cn/infomation/details/32523.html)
- 2026-06-03 上线 Kimi Work 桌面办公智能体：依托 Kimi K2.6 长程能力（13 小时连续编码、300 子 Agent 并行、4000+ 次自主工具调用），Work/Chat 双模式，以"模型能力驱动"切入办公。来源：[钛媒体](https://www.tmtpost.com/8099969.html)

**Gamma（海外）**：
- AI 演示文稿生成：输入文本秒出 PPT，能写、能画、润色；中文流畅度尚可；社区反馈"效率高但价格贵"（免费额度有限）。来源：[Lilys AI 实测](https://lilys.ai/zh/notes/gamma-ai/gamma-ai-ppt-maker-review-tutorial)、[智东西](https://zhidx.com/p/502658.html)、[搜狐](https://www.sohu.com/a/895129969_99985415)
- 2025 年多篇中文横评把它与 Kimi/通义/讯飞智文/豆包并列为"AI PPT 七强/五强"。来源：[今日头条·7款工具横评](https://m.toutiao.com/article/7587068579779035700/)、[什么值得买·答辩PPT](https://post.smzdm.com/p/avvpz554/)

**AhaSlides（海外）**：
- AI 演示文稿生成器主打"互动式幻灯片"（投票/测验/问答），面向教学/会议场景；已提供 MCP 工具接口（AI 代理可读取演示内容）、ChatGPT 市场集成。来源：[AhaSlides 官网](https://ahaslides.com/zh-TW/features/mcp-tool/)、[AI 辅助功能博客](https://ahaslides.com/zh-CN/blog/ai-assisted-features-streamlined-tools/)、[AI 演示文稿生成器](https://ahaslides.com/zh-TW/features/ai-presentation-maker/)

**国产 AI PPT 横评（18 款实测）**：生成模式、内容结构、编辑体验与适用场景对比，结论是"一键生成质量普遍不可直接用、需人工大改"。来源：[技术栈实测](https://jishuzhan.net/article/2055137484833132546)、[1ai.net 10款实测](https://www.1ai.net/40400.html)

### 2.2 标书专用工具（垂直细分）

- **华胜天成"投标大王 2.0"**（2025-10 发布）：定位"AI 投标新标准"，两大颠覆升级（招标文件解析 + 标书内容生成），2025 WAIC 亮相；华胜天成 2025 年度十大事件之一。来源：[华胜天成官网](https://www.teamsun.com.cn/newsdetail.htm?id=2510090001)、[数据猿专访](http://www.teamsun.com.cn:8080/newsdetail.htm?id=2510090002)、[投资界](https://news.pedaily.cn/20260128/123051.shtml)
- **易标书 OpenBidKit（GitHub 开源）**：开箱即用的 AI 标书编写工具，含投标工具箱、知识库、标书查重、废标项检查，完全开源免费；是"标书场景 + 本地工具链"的直接样本。来源：[GitHub](https://github.com/fb208/OpenBidKit_Yibiao)、[AI铺子收录](https://www.aipuzi.cn/ai-tools/yibiaoai.html)
- **AI 标书精灵**（湖南百晓生，2024-11 上线）：服务政采与招投标行业。来源：[中国招标投标协会](http://www.ctba.org.cn/list_show.jsp?record_id=338223)、[经济网](https://www.ceweekly.cn/company/2024/1126/461598.html)
- **链企招投标文本生成算法**（浙江链企智能，2023 年成立的 AI 商业搜索公司）：招投标文本生成算法已过算法备案。来源：[百度百科](https://baike.baidu.com/item/%E9%93%BE%E4%BC%81%E6%8B%9B%E6%8A%95%E6%A0%87%E6%96%87%E6%9C%AC%E7%94%9F%E6%88%90%E7%AE%97%E6%B3%95)
- **tender-cli（GitHub aigc-open）**：标书生成智能体工具（CLI 形态）。来源：[GitHub](https://github.com/aigc-open/tender-cli)
- **政府侧采购智能辅助体系**：双鸭山市财政局打造"全国领先政府采购智能辅助体系"（2025-10 报道）；"生成 120 万字标书只要 10 分钟"（济南时报 2025 报道）——说明标杆案例已出现，但多为政府采购平台内部能力而非独立产品。来源：[双鸭山市政府](http://shuangyashan.gov.cn/sys/d83a395e787241e0a02a9528a12ed5ea/202510/c07_236348.shtml)、[济南时报](https://www.jinantimes.com.cn/news-100-9569715.html)
- **北京筑龙**：大模型助力采购供应链，招标文件/方案编制"效率神器"（2024-07）。来源：[中国网](http://business.china.com.cn/2024-07/02/content_42847987.html)
- **开源自研**：个人开发者"手搓自动生成标书的大模型工具"（2024 前后），GitHub 上 AI 投标文档生成器项目活跃，说明需求真实但供给碎片化。来源：[开源中国](https://www.whaleops.com/846839-846849_3111467.html)、[GitHub](https://github.com/Yuejunzhang1/AI-bid-document-generator-YNMT-)
- **注意**：搜索"青鸟云""标书狗"等传闻中的品牌未检索到有效产品信息，标书垂直市场实际由"华胜天成/链企/北京筑龙/易标书"等玩家构成，市场集中度低、信息不透明。

### 2.3 通用 AI 写作工具在标书/方案场景的现状

- **现状判断**：通用对话型 AI（豆包/文心/DeepSeek/Kimi 等）被大量用于"打草稿"，但标书/方案生产需要**招标文件逐条响应、格式排版、盖章签字版式、查重、废标项检查**，通用工具无法闭环，这正是垂直标书工具存在的原因。来源：[阿里云开发者·招投标垂直模型选型](https://developer.aliyun.com/article/1755021)、[知乎·AI写标书法律风险](https://zhuanlan.zhihu.com/p/2052792741961130963)
- **AI 周报/汇报生成**：多为个人开源/插件（GitReport-AI VS Code 插件、LazyLLM 述职 Agent、Aloudata 智能报告、帆软 AI 报表），或大厂功能点（腾讯文档"周报神器"、飞书 aily"工作总结报告"）。尚无独立的"中文汇报工作台"。来源：[GitHub·GitReport-AI](https://github.com/TinsanWoo/GitReport-AI)、[腾讯云·Aloudata](https://cloud.tencent.cn/developer/article/2597618)、[帆软](https://www.finereport.com/blog/article/68b01315d2527e0eb702d0c7)、[中国日报·aily任务模式](https://ex.chinadaily.com.cn/exchange/partners/82/rss/channel/cn/columns/sz8srm/stories/WS694108aaa310942cc4996e93.html)、[GitHub·日报周报工具](https://github.com/umlink/daily-report-skill)

### 2.4 标书 AI 的合规与格式问题（关键洞察）

- **格式问题**："你写的标书，AI 可能根本读不到！"——招标系统/评审系统对格式（PDF 转换、页眉页脚、签章、目录结构）有硬性要求，AI 生成的 markdown/纯文本无法直接进入投标流程。来源：[知乎](https://zhuanlan.zhihu.com/p/2062565575071643068)
- **法律风险**：用 AI 写标书的合规自检 5 个维度（真实性、技术响应、业绩证明、串标嫌疑、资质声明）；2026 年 AI 标书工具合规全景——四大法律风险（虚假业绩、雷同标书、数据泄露、格式废标）与三道技术防线；AI 生成投标文件的五类合规隐患；"用 AI 写标书会不会串标"。来源：[知乎](https://zhuanlan.zhihu.com/p/2052792741961130963)、[阿里云开发者](https://developer.aliyun.com/article/1754065)、[博客园](https://www.cnblogs.com/erhao-a-i-gong/p/22311135)、[搜狐](https://www.sohu.com/a/1062671148_121329020)
- **行业政策**：政府采购领域讨论"拥抱 AI 才能掌控 AI"（2025 年政府采购报），生成式 AI 在政采中的应用前景与法律风险成为期刊研究主题（《招标采购管理》2024 年第 2 期）。来源：[中国政府采购报](http://www.cgpnews.cn/epapers/69230)、[知网](https://wap.cnki.net/touch/web/Journal/Article/ZBCG202402025.html)
- **案例实证**：AI 标书系统客户案例显示，"从效率困局到中标利器"的智能化突围已被验证（树尚云案例），但依赖人工复核闭环。来源：[树尚云](https://m.shushangyun.com/article-23261.html)

---

## 三、办公 Agent / 文档智能体产品

### 3.1 微软 Copilot / Agent

- **全球形态**：Microsoft 365 Copilot（2023-11 商用）嵌入 Word/Excel/PowerPoint/Teams；2025-09 为 Office 应用新增**智能体模式（Agent Mode）与 Office 智能体**，微软称之为"Vibe Working（氛围办公）"——用户口述意图、AI 代为执行跨应用任务。来源：[InfoQ](https://www.infoq.cn/article/cy1zrc3jhjbvgtpb0mwi)、[Axios](https://www.axios.com/2025/09/29/vibe-working-microsoft-agent-mode)、[PCMag](https://ca.pcmag.com/ai/10881/microsoft-sets-the-tone-for-vibe-working-with-new-agent-mode-in-word-excel)
- **Ignite 2025**：AI Agents 成为大会中心，推出协作 Agent 与 Copilot 扩展体系。来源：[Counterpoint](https://counterpointresearch.com/en/insights/microsoft-ignite-2025-recap-ai-agents-take-centre-stage)、[Microsoft Mechanics](https://microsoftmechanics.libsyn.com/podcast/new-collaborative-agents-in-microsoft-365-copilot)
- **中国市场受限**：2024-03 Copilot 企业版进入中国、近 200 个国内客户部署，但 Bing 联网/文生图不可用、私有化中文支持度不高；2024-10 微软宣布终止中国区部分个人服务（含 Bing Chat/Copilot 个人版），OpenAI API 也退出中国。来源：[钛媒体](https://www.tmtpost.com/6981843.html)、[湖南日报](https://www.hunantoday.cn/news/xhn/202410/20854422.html)、[澎湃](https://m.thepaper.cn/newsDetail_forward_29076964)
- **判断**：微软 Copilot 在中文市场"能做但做不深"，且受数据合规约束，难以覆盖政企内网场景——这是国产办公 AI 的最大外部空窗。

### 3.2 Notion AI / Notion 3.0 Agent

- Notion AI：文档内问答、写作辅助、"第二大脑"知识管理，中文社区评测较多（"实不实用"两极）。来源：[少数派](https://sspai.com/post/102615)、[aitoolcn](https://aitoolcn.com/reviews/notion-ai)
- **Notion 3.0 Agent（2025-09）**：自主执行最长 20 分钟任务、跨平台数据整合，推出 Iris 数字队友；被视为"笔记工具向 agent 化跃迁"的标志。来源：[Skywork](https://skywork.ai/blog/notion-3-0-autonomous-tasks-cross-app/)、[53AI](https://www.53ai.com/news/LargeLanguageModel/2025092245761.html)
- 判断：Notion 是"个人知识工作台"标杆，但中文场景的文档生产（公文/标书格式）支持弱，且数据在海外云。

### 3.3 WPS AI 的 agent 化程度

- 灵犀=原生 Office 智能体（见 1.1），是国内"文档 agent 化"最彻底的产品；但其 agent 能力绑定 WPS 生态与云端账号，私有化版本面向政企而非个人本地。
- 配套：WPS 知识库 + 文档洞察引擎 = "企业知识 agent"底座（解析 PDF/DOC/XLX/PPT 含图片表格 + 权限管控 + 溯源）。
- 佐证：金山办公田然明确"办公正在走向人人都有 AI 助理的时代"，WPS 灵犀获 WAIC"镇馆之宝"，表明文档 agent 化是金山战略主线。来源：[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)

### 3.4 本地 / 私有化文档 AI 产品（RAG 私有化部署）

**开源本地 RAG 四件套（社区主流方案）**：
- **RAGFlow**（InfiniFlow，端到端 RAG，深度文档解析）来源：[百度云文章](https://cloud.baidu.com/article/3365409)
- **Langchain-Chatchat**（离线运行的大模型知识库，本地文档问答）来源：[腾讯云社区](https://cloud.tencent.cn/developer/article/2390871)
- **QAnything**（网易有道自研 RAG 引擎，2024-01 开源，PDF/Word 多格式）来源：[中国网·深圳](http://szjj.china.com.cn/m/2024-01/17/content_42673334.html)
- **AnythingLLM + Ollama + DeepSeek**（小白教程级本地知识库，桌面应用形态）来源：[腾讯云社区教程](https://developer.cloud.tencent.cn/article/2526582)、[Thoughtworks Radar](https://www.thoughtworks.com/zh-cn/radar/tools/anythingllm)
- 判断：本地 RAG 已成熟到"人人可搭"，但**停留在问答/检索层**，缺乏"生成→格式排版→交付"的文档生产能力——这正是本地文档工作台的补位点。

**商用私有化（政企交付）**：
- 实在智能"全栈本地化 Agent"：LLM/知识库/执行器全部客户侧部署；案例=农发行（4 万员工、1800 县级机构，全内网 RPA，反洗钱排查效率 +100%）、赣州银行、渤海财险（ChatBI NL2SQL 数据零外流）；其 IDP 文档审阅（合同/票据 OCR）已是本地化文档处理的成熟产品。来源：[实在智能·数据不出域](https://www.ai-indeed.com/encyclopedia/27690.html)
- 华为云 OfficeClaw → OfficeAce（2026-04 邀测，2026-08 升级）：基于 OpenClaw 的办公智能体，两大特色="思辨专家团"（多 Agent 辩论）+ "AI PPT 版式规划"（DeepResearch + 可编辑 PPT，5 Agent × 6 阶段流水线 × 6 项 QA × 3 轮修复）；AgentArts 平台提供安全沙箱、细粒度权限、层次化记忆、持续学习；面向企业运营工作台 + 员工工作台双端。来源：[智东西](https://m.zhidx.com/p/549631.html)、[华为云官方文档](https://support.huaweicloud.com/officeace-agentarts-pc/officeace-agentarts-pc-0090.html)
- 桌面办公 Agent 赛道全景（2026）：OpenClaw/Hermes 开源卡位"接管操作系统级任务调度"；国内 WorkBuddy（腾讯，日活 1300 万+）、Kimi Work（月之暗面）、QwenWork（阿里，2026-08-03 公测，整合 QoderWork/悟空/MuleRun，融入钉钉）、TRAE Work（字节，由 TRAE Solo 升级）；艾媒统计半年超 20 款国产发布；InfoQ 盘点 20 款桌面 Agent。来源：[钛媒体](https://www.tmtpost.com/8099969.html)、[艾媒](https://www.iimedia.cn/c400/113198.html)、[InfoQ](https://xie.infoq.cn/article/bb161038a4410ca327b8ba6b4)、[36氪](https://eu.36kr.com/zh/p/3868029236401414)
- 人民智擎可信办公智能体（2026-05 人民网发布）：主打政企办公"可信"范式。来源：[人民网](http://finance.people.com.cn/n1/2026/0510/c1004-40716908.html)
- OpenClaw 架构说明：一种针对个人自动化的开源技术架构，通过 MCP 打通大模型与外部软件，成为跨厂商轻量级智能体的通用形态（本地部署/云端调用/行业定制多版本）。来源：[百度百科·OpenClaw架构](https://baike.baidu.com/item/OpenClaw%E6%9E%B6%E6%9E%84)、[AI Native Landscape](https://landscape.jimmysong.io/projects/openclaw/)
- Claude Cowork（Anthropic，2026 年）：面向非编程用户的桌面 AI 代理，与 Claude Code 同引擎、两种工作形态，完成"工作"而非"对话"。来源：[DataCamp 对比](https://www.datacamp.com/zh/blog/claude-cowork-vs-claude-code)、[腾讯云社区](https://cloud.tencent.cn/developer/article/2666727)

---

## 四、中文用户的办公 AI 心智

### 4.1 使用渗透高，但"文档不安全感"普遍

- 智联招聘《职场"人机共生"演进情况调研报告》（2025-09）：近八成职场人每周使用 AI 工具；最常使用通用对话型 AI；**最多用于文档撰写编辑**；56.1% 愿意为 AI 服务付费；**"反复调整指令"成为最大痛点**（人机协作信任鸿沟）。来源：[经济网](https://www.ceweekly.cn/zxfb/2025/0925/481469.html)、[界面](https://www.jiemian.com/article/13383410.html)、[中新经纬](https://www.jwview.com/jingwei/html/09-22/634407.shtml)、[企业观察网](https://www.cneo.com.cn/detail89132.html)
- CNNIC：中国生成式 AI 用户规模 2.49 亿（2024-12），智能体成为生成式 AI 应用主流形态之一；文心一言使用率居首 11.5%。来源：[中国高新网](http://www.chinahightech.com/yaowen/2025-01/17/content_287543.html)、[新华网](http://www.xinhuanet.com/tech/20241203/ba5e48db66bc49a2a2dcc103a0f68ccd/c.html)
- 毕马威《2026 全球科技报告》：88% 企业已试点 Agentic AI，仅 24% 在多种应用场景实现投资回报；媒体判断"没有企业敢让 AI 独立完成一份关键文件的书写"——**质量稳定性与责任归属是核心顾虑**。来源：[钛媒体](https://www.tmtpost.com/8099969.html)
- 高校/职场高频使用：近两成高校师生频繁使用 AI 工具（读特新闻）。来源：[读特](https://m.dutenews.com/n/article/8592181)

### 4.2 格式 / 合规焦虑是中文文档场景特有的"硬约束"

- 党政机关公文格式有国家标准 GB/T 9704-2012（字体字号、版式都有规定），政务 AI 落地必须处理格式合规。来源：[安康市政府](https://www.ankang.gov.cn/WapContent-2060365.html)、[武汉大学党内法规研究中心](https://iplr.whu.edu.cn/info/1032/2875.htm)
- 标书场景：格式废标风险 + 评审系统读取问题 + 查重/串标合规（见 2.4）。
- 行业侧印证：WPS 灵犀把"格式保留、修改可控"列为第一卖点；华为云 OfficeClaw 把"版式遵从性、可编辑性"列为 PPT 生成三大痛点之一。来源：[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)、[智东西](https://m.zhidx.com/p/549631.html)

### 4.3 私有数据顾虑：从"不敢上传"到"数据不出域"

- 企业/个人对文档上云的核心顾虑=数据泄露与合规红线；钉钉陈航直言"AI 时代，企业最担心的是'数据裸奔'"，钉钉 AI 搜问的权限边界设计就是为了"让企业敢用 AI"。来源：[中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html)
- "大模型进不了内网之后，企业 AI 的生意才真正开始"（钛媒体）；大模型私有化部署浪潮的 A/B 面——警惕"信息孤岛"在 AI 时代复现（证券时报）；美团内网无法访问豆包/千问，不少员工从未使用（OFweek 2026-07 独家）。来源：[钛媒体](https://www.tmtpost.com/8070771.html)、[证券时报](https://stcn.com/article/detail/1582536.html)、[OFweek](https://www.ofweek.com/ai/2026-07/ART-201718-8460-30693093.html)
- 实在智能：金融/政务/医疗对数据安全"零容忍"，"数据不出域"倒逼 Agent 从云端推理转向全栈本地化（LLM/知识库/执行器全部客户侧部署）；Gartner 预测到 2027 年超 60% 大型企业将在受监管环境采用私有化 AI 智能体。来源：[实在智能](https://www.ai-indeed.com/encyclopedia/27690.html)
- 企业 AI 知识库落地需兼顾数据安全架构（八大行业落地实践，华为云社区）。来源：[阿里云](https://developer.aliyun.com/article/1750089)、[华为云社区](https://bbs.huaweicloud.com/blogs/482526)

### 4.4 银行 / 政务 / 企业私有化需求的实证

- **政务**：甘肃武威"公文易办"私有化政务大模型（2026-07，WPS 365 助力，公文分钟级交付）；金山 WPS AI 私有化方案适配 8 种国产 OS + 鲲鹏/海光 + 达梦/金仓；金山政务 AI 一体机（金山云）；海珠区接入 DeepSeek 智慧政务。来源：[国际在线](https://news.cri.cn/2026-07-07/1bb7f2f1-a854-460d-a31a-dd2306b7f135.html)、[央广网](https://tech.cnr.cn/techph/20250529/t20250529_527189297.shtml)、[金山云](https://www.ksyun.com/cms/news/738.html)、[广州政府](https://www.gz.gov.cn/xw/zwlb/gqdt/hzq/content/mpost_10158751.html)
- **银行/金融**：江苏农商联合银行累计落地 304 项智能应用场景（2026-07）；中电金信 AI 辅助工作平台赋能银行核心系统研发；南天信息金融网点业务知识库助手；大连银行 AI 智能体应用探索；农发行/渤海财险全本地化案例（见 3.4）。来源：[现代快报](http://xdkb.net/m1/rd/ja9qt/566414.html)、[搜狐](https://www.sohu.com/a/1027356975_120847886)、[南天信息](https://www.nantian.com.cn/case-detail/2942.html)、[大连银行](https://www.fddnet.cn/index.php?m=content&c=index&a=show&catid=336&id=145)
- **信创/国产化**：升腾"DeepSeek 赋能信创办公"；每日互动×中科可控×麒麟软件"个知·智能工作站信创版"；艾媒白皮书明确"用户对混合云部署、合规信创兼容、数据安全的要求持续提升"。来源：[飞儿科技](http://www.feig.com.cn/i_250220175509474940000000000000000001.html)、[DoNews](https://www.donews.com/news/detail/4/6633584.html)、[艾媒](https://www.iimedia.cn/c400/113198.html)
- **端侧 AI 接受度**：AI PC（联想"人工智能+"近 200 项成果、科大讯飞星火 AIPC、YOGA Air X）、AI 办公本（思必驰 X5）兴起，但端侧 AI "仍困于隐私与算力"。来源：[新华网](http://www.xinhuanet.com/tech/20250828/04d8ef9e29bd4b6991d0a0ac1d13efff/c.html)、[湖北广电](https://news.hbtv.com.cn/p/4561151.html)、[赛迪网](https://www.ccidnet.com/hlw/91227.jhtml)、[搜狐](https://m.sohu.com/a/1009576952_114986)

---

## 五、结论：本地 agent 文档工作台在中文市场的空白与机会

### 5.1 市场基础（有需求、有钱、在增长）

- 中国 AI+办公软件市场规模 2025 年约 700 亿元，需求迅速扩张（智研/观研）；协同办公市场近 400 亿元（第一财经）；2025 年中国 AI 智能体市场 804 亿元（同比 +123.2%），预计 2030 年达 6968 亿元，全球 2025 年 372 亿美元 → 2030 年 3122 亿美元（艾媒）。来源：[观研天下](https://www.gonyn.com/industry/1776655.html)、[智研咨询](https://www.chyxx.com/industry/1224479.html)、[第一财经](https://www.yicai.com/news/102714109.html)、[艾媒](https://www.iimedia.cn/c400/113198.html)
- 付费意愿：59.1% 的使用者愿为 AI 办公智能体付费（艾媒）；56.1% 职场人愿为 AI 服务付费（智联）。来源：[艾媒](https://www.iimedia.cn/c400/113198.html)、[中新经纬](https://www.jwview.com/jingwei/html/09-22/634407.shtml)

### 5.2 现有供给的四条路线与各自的"盲区"

| 路线 | 代表 | 覆盖 | 盲区 |
|---|---|---|---|
| 云端协作平台 + AI 附加 | WPS 灵犀 / 飞书 aily / 钉钉 AI 钉钉 / 腾讯文档 | 生成/知识库/Agent 化 | 绑定云端生态；文档绑定在线格式；非本地优先 |
| 桌面通用 Agent | WorkBuddy / Kimi Work / QwenWork / TRAE Work / OfficeClaw | 文件处理、网页、表格、跨应用 | 通用"轻活"为主；中文重文档（标书/公文/测算）无垂直深度 |
| 垂直标书工具 | 投标大王 / 易标书 / 标书精灵 / 链企 | 招标响应、查重 | 单点工具，无"工作台"闭环（缺知识库/成本库/交付物管理）；合规风险未解决 |
| 本地 RAG / 私有化 | RAGFlow / Chatchat / QAnything / AnythingLLM / 实在智能 | 文档解析、问答、私有化 | 停于"检索问答"，缺"生成→排版→交付"；或太重（政企级项目） |

### 5.3 gaea 办公板块的机会点（对照本调研）

1. **"本地优先 + 云端统筹"的形态正处于验证期**：艾媒判断"端云混合协同凭借灵活调用、成本优化优势成为主流部署方式"；微软 Copilot 在华受限、OpenClaw 类开源工具链成熟（MCP 成为通用标准），为"本地执行 + 云端规划"的分层架构提供了技术和心智窗口。来源：[艾媒](https://www.iimedia.cn/c400/113198.html)、[钛媒体](https://www.tmtpost.com/8099969.html)
2. **中文重文档生产（标书/方案/汇报/成本测算）是主流办公智能体的真空带**：艾媒数据显示办公智能体使用场景 51.8% 是 PPT 生成、38.2% 表格分析、37.1% 文件整理——"把一份正式文档从头写到尾并可交付"仍是行业公认难题（"没有企业敢让 AI 独立完成一份关键文件"）。来源：[艾媒](https://www.iimedia.cn/c400/113198.html)、[钛媒体](https://www.tmtpost.com/8099969.html)
3. **格式保真 + 可编辑 + 可溯源 = 差异化护城河**：华为云 OfficeClaw 与 WPS 灵犀都把"版式遵从/可编辑/格式保留"当作核心卖点，说明这是用户强感知的价值；gaea 的"纯本地 docx/xlsx/pdf 解析转换"可直接命中（招标系统读取、公文 GB/T 9704 排版、评审格式要求）。来源：[智东西](https://m.zhidx.com/p/549631.html)、[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)、[知乎](https://zhuanlan.zhihu.com/p/2062565575071643068)
4. **知识库 + 成本库 + 交付物/任务管理是"工作台"而非"聊天框"的护城河**：WPS 知识库/飞书私域数据/钉钉 AI 搜问都证明了"企业知识沉淀"是刚需，但都绑定各自云生态；独立、可导出、本地可控的知识库+成本库（如投标成本测算、报价单）在中文市场无对标产品。
5. **私有化/信创合规需求为"本地优先"提供企业级买单理由**：政务公文、银行内部文档、央国企数据合规均已形成成熟案例（甘肃武威、农发行、招商证券），个人专业用户（标书代理、咨询、造价）同样有"文档不能上传公网"的刚需。来源：[国际在线](https://news.cri.cn/2026-07-07/1bb7f2f1-a854-460d-a31a-dd2306b7f135.html)、[实在智能](https://www.ai-indeed.com/encyclopedia/27690.html)、[央广网](https://tech.cnr.cn/techph/20250529/t20250529_527189297.shtml)
6. **风险提示**：① 巨头（腾讯 WorkBuddy、阿里 QwenWork、华为 OfficeAce）正快速把"桌面 Agent"做成通用形态，窗口期估计 12–24 个月；② 艾媒/毕马威数据表明"价值兑现"是行业瓶颈，gaea 需在"可交付的中文文档成果"上做出可感知的质量差异；③ 办公智能体竞争正从"框架混战"转向"场景落地"，垂直深耕（标书/成本/公文）比通用更易建立壁垒。来源：[钛媒体](https://www.tmtpost.com/8099969.html)、[艾媒](https://www.iimedia.cn/c400/113198.html)

### 5.4 一句话总结

> 中文市场已确认"AI 办公"是千亿级赛道且正在 agent 化，但主流产品要么绑定云端协作生态、要么停留在"一键生成"的轻活层；**"本地优先 + 中文重文档生产（标书/方案/汇报/成本测算）+ 纯本地 docx/xlsx/pdf 解析转换 + 知识库/成本库/交付物管理"的组合在 2026 年 8 月仍无直接对标产品**，gaea 办公板块站在一个真实且有时间窗口的空白位上。


---

## 六、补充细节与深度用例（支撑前文结论）

### 6.1 WPS 灵犀的现场演示与场景细节（2025-07 发布会）

- 现场案例：用户通过与灵犀多轮对话完成装修合同"从新建、修改到优化"全过程——口语化的姓名/地址/金额自动补全进合同，项目信息按表格呈现，AI 提示合同条款潜在风险；新建文档时既能"写好"也能"审核规范性"。来源：[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)
- 场景样例：券商 FICC 分析师用灵犀处理政策文件/上市公司财报/评级报告，整理研报信息；求职者用"文档开口说话"模拟面试；考生用灵犀出测试题。来源：[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)
- 行业判断（田然）：AI 行业此前两大问题——① AI 能力藏在软件后面，用户看到软件变强大但 AI 能力无法完全发挥；② 用户通过聊天机器人调用 AI，单次生成内容"看似炫酷但并不实用，甚至不能用"。灵犀的思路是"针对场景做 AI 和软件的双向改造，软件为 AI 设计专有能力，同时教会 AI 跟软件深度交流"。来源：[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)
- 同屏交互：WPS Office 部分组件已形成"左 Office 套件、右 WPS 灵犀"的同屏形态，AI 识别意图后直接修改左侧文档，全程不跳转其他应用；多轮对话、修改可控、格式保留是对外宣称的三大优势。来源：[新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html)

### 6.2 飞书多维表格的 AI 化细节

- 定位：多维表格是依托 aPaaS 组建的产品，可替代许多小型业务系统（销售/客服/人力），"表格 + 大模型"成为真实业务场景试验场；中小团队无需再采买单独业务系统。来源：[36氪](https://m.36kr.com/p/3371623528452615)
- "用嘴搭系统"：用户对话式描述"搭建包含订单、物流、成交额类目的电商业务看板"，飞书妙搭调用 Agent 能力一键生成 AI 应用；应用模式内置列表/Tab/轮播等组件，让 AI 应用"看起来更像可交互的 AI 系统"。来源：[36氪](https://m.36kr.com/p/3371623528452615)
- AI 表格让"表格变成 AI 同事"：表格可加入群聊，不打开表就能用表（网易/品玩报道）。来源：[网易](https://m.163.com/dy/article/L0OMR6BT0511AQHO.html)、[品玩](https://www.pingwest.com/a/315155)
- 生态进攻：2025-07 飞书宣布多维表格将在企微、钉钉平台开放申请，正面进入腾讯/阿里生态腹地（南方+标题"办公产品集体卷 AI 表格"）。来源：[南方+](https://static.nfnews.com/content/202507/12/c11502361.html)、[第一财经](https://www.yicai.com/news/102714109.html)

### 6.3 钉钉 AI 的更多细节

- 陈航战略：钉钉 8.0=AI 钉钉 1.0，"过去十年解决'人连接人'，未来十年用 AI 解决'人连接事'，让每个企业拥有专属 AI 能力"；"传统办公软件是'被动响应'，Agent 要做到'主动服务'"。来源：[中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html)
- AIFusion 引擎：汇集全球 50 余种主流大模型，支持企业对比选型，提供海外模型、本地私有模型接入选项，满足央国企/金融机构数据合规——"权限边界"设计让企业"既不浪费数据价值，也不泄露核心信息"。来源：[中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html)
- AI 表格=“企业的 AI 应用工厂”："过去只有技术团队能做应用，现在一个 HR 用自然语言就能做薪资核算表，一个车间主任能做设备巡检流"。来源：[中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html)
- 行业场景：顾家家居 AI 销售助理（话术建议/客户意向分析/销售能力雷达图）、夸克联合作业批改（教师批改时间 2 小时/天→10 分钟）。来源：[中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html)
- 2025-12 AI 钉钉 1.1"木兰"：定位 AI 工作智能操作系统（Agent OS），连接物理与数字世界（东方财富/中国工业报）。来源：[东方财富](https://finance.eastmoney.com/a/202512233599866277.html)、[中国工业新闻网](https://www.cinn.cn/sz/2025/12-25/W1OP20mr.html)

### 6.4 腾讯 AI 办公矩阵补充

- 腾讯元宝电脑版（2025-02 前后）成为腾讯系 AI 入口，与办公矩阵联动。来源：[ZOL](https://ai.zol.com.cn/954/9545795.html)
- WorkBuddy 更新 2026-08：零代码生成网页，HTML 编辑"如同 Word"。来源：[站长之家](https://www.chinaz.com/2026/0813/1770890.shtml)
- 腾讯 WorkBuddy 企业版（2026-06-05 发布，腾讯云）：面向企业团队办公的智能体套件，从"超级个体"到"超级团队"。来源：[东方财富](https://finance.eastmoney.com/a/202606053762041582.html)、[品玩](https://www.pingwest.com/a/314412)、[极客公园](https://www.geekpark.net/news/365512)

### 6.5 Kimi Work / 桌面 Agent 细节

- Kimi Work 2026-06-03 上线，Beta 高频迭代：已新增文件预览、草稿自动保存、目标模式、插件中心；Kimi K2.6 长程任务能力（13 小时连续编码、300 子 Agent 并行、4000+ 次自主工具调用）。来源：[钛媒体](https://www.tmtpost.com/8099969.html)
- 阿里 QwenWork 2026-08-03 公测：将 QoderWork、悟空、MuleRun 三款产品整合为统一平台，是"业界首批将桌面智能体、云端智能体与企业协作智能体融于一体的平台之一"，将深度融入钉钉生态，未来推出移动 App 与国际版。来源：[钛媒体](https://www.tmtpost.com/8099969.html)
- 字节 2026-07-30 组织调整：飞书产品团队与豆包产品团队整合为豆包产品团队；飞书 GTM 与火山引擎整合为 ToB GTM"创造力服务平台"；新版飞书 aily 官宣主动工作、团队共享智能体、多智能体协同；TRAE Solo 升级为 TRAE Work。来源：[钛媒体](https://www.tmtpost.com/8099969.html)
- 行业阶段判断："2025 年的 Agent 框架混战已基本结束，产业注意力正式转向'怎么落地'"，下一阶段竞争聚焦场景落地能力与跨系统协同效率。来源：[钛媒体](https://www.tmtpost.com/8099969.html)

### 6.6 华为云 OfficeClaw / OfficeAce 深度

- 邀测时间：2026-04-16 启动邀测，华为云官网每天 10 点限量发放邀请码。来源：[智东西](https://m.zhidx.com/p/549631.html)、[同花顺](http://stock.10jqka.com.cn/20260416/c676039951.shtml)
- 两大特色："思辨专家团"（源自华为内部年轻工程师为提升代码质量开发的系统，多 Agent 接入不同模型、配置不同"灵魂属性"互相辩论，回应"主从式 Agent"单点思考偏见/单点故障问题）；"AI PPT 版式规划"（DeepResearch 深度研究 + 版式规划，专治版式遵从性差、模型幻觉、不可编辑三大痛点）。来源：[智东西](https://m.zhidx.com/p/549631.html)
- 五个 Agent 分工 + 6 阶段流水线编排 + 6 项 QA 质量检测 + 3 轮修复自动迭代，PPT 多轮迭代生成。来源：[智东西](https://m.zhidx.com/p/549631.html)
- AgentArts 平台能力：全链路观测审计、长任务资源优化、层次化记忆机制（复杂任务后期不遗忘前期提示）、安全沙箱、细粒度权限、企业环境持续学习。来源：[智东西](https://m.zhidx.com/p/549631.html)
- 2026-08 升级为 OfficeAce，瞄准"企业 Agent 入口"（钛媒体 TechPulse 报道）。来源：[华为云官方文档](https://support.huaweicloud.com/officeace-agentarts-pc/officeace-agentarts-pc-0090.html)
- 华为申请"华为云 OfficeClaw"商标（2026-05）。来源：[证券之星](https://bank.stockstar.com/IG2026050800028972.shtml)

### 6.7 本地 RAG 工具细节

- RAGFlow：端到端 RAG 解决方案，突出文档深度解析（表格/图片/版面），适合企业私有知识库。来源：[百度云文章](https://cloud.baidu.com/article/3365409)
- Langchain-Chatchat：可离线运行，支持本地文档问答，开源日报多次推荐。来源：[腾讯云社区](https://cloud.tencent.cn/developer/article/2390871)
- QAnything：网易有道自研，支持 PDF/Word/PPT 等多格式，宣称"任何格式都可问答"，2024-01 开源。来源：[中国网·深圳](http://szjj.china.com.cn/m/2024-01/17/content_42673334.html)
- AnythingLLM+Ollama+DeepSeek：社区教程规模庞大（本地知识库"保姆级"教程），说明"本地模型 + 本地文档"需求真实。来源：[腾讯云社区教程](https://developer.cloud.tencent.cn/article/2526582)、[腾讯云·AnythingLLM部署](https://cloud.tencent.cn/developer/article/2558277)
- 商用：实在智能 IDP 文档审阅平台（发票/合同/体检报告本地 OCR 拆箱-分拣-录入）、RPA+大模型组合的"全栈本地化数字员工"。来源：[实在智能](https://www.ai-indeed.com/encyclopedia/27690.html)

### 6.8 端侧 AI / AI PC 办公形态

- AI PC 生态：联想展示"人工智能+"近 200 项成果（2025-08）；科大讯飞发布星火 AIPC（2025）；YOGA Air X AI 元启版；思必驰 AI 办公本 X5（端侧大模型重塑办公体验）。来源：[新华网](http://www.xinhuanet.com/tech/20250828/04d8ef9e29bd4b6991d0a0ac1d13efff/c.html)、[湖北广电](https://news.hbtv.com.cn/p/4561151.html)、[赛迪网](https://www.ccidnet.com/hlw/91227.jhtml)、[百度百科](https://baike.baidu.com/item/YOGA%20Air%20X%20AI%E5%85%83%E5%90%AF%E7%89%88/65433404)
- 局限："端侧 AI 仍困于隐私与算力"（搜狐科技，2025-08），端侧大模型能力受限是当前瓶颈。来源：[搜狐](https://m.sohu.com/a/1009576952_114986)
- AI"搅动"PC 市场：端侧 AI 成为 2025 PC 换机叙事（经济参考报）。来源：[经济参考报](http://jjckb.xinhuanet.com/20250317/9c038afe7f0349b88cd4a3b7dfcd7758/c.html)

### 6.9 汇报 / 方案场景的补充证据

- Aloudata Agent"智能报告"：周报/月报生成（NL2SQL + 报告模板）。来源：[腾讯云](https://cloud.tencent.cn/developer/article/2597618)
- 帆软 FineReport AI 报表：面向企业报表自动化（"AI 报表如何自动生成报告"）。来源：[帆软](https://www.finereport.com/blog/article/68b01315d2527e0eb702d0c7)
- LazyLLM 述职 Agent：个人开发者用 LazyLLM 构建"打工人述职 Agent"。来源：[腾讯云](https://cloud.tencent.cn/developer/article/2566148)
- GitReport-AI：基于 Git 提交记录用 DeepSeek 生成周报/月报的 VS Code 插件。来源：[GitHub](https://github.com/TinsanWoo/GitReport-AI)
- 飞书 aily"基于我的飞书云文档、任务及会议纪要，生成周工作总结报告"是官方示例之一。来源：[中国日报](https://ex.chinadaily.com.cn/exchange/partners/82/rss/channel/cn/columns/sz8srm/stories/WS694108aaa310942cc4996e93.html)

---

## 七、市场数据与机构观点汇编

### 7.1 市场规模数据

| 指标 | 数值 | 时间 | 来源 |
|---|---|---|---|
| 中国 AI+办公软件市场规模 | 约 700 亿元 | 2025 | [观研天下](https://www.gonyn.com/industry/1776655.html) |
| 中国 AI 智能体市场规模 | 804 亿元（+123.2%） | 2025 | [艾媒白皮书](https://www.iimedia.cn/c400/113198.html) |
| 中国 AI 智能体市场规模（预测） | 6968 亿元 | 2030E | [艾媒白皮书](https://www.iimedia.cn/c400/113198.html) |
| 全球 AI 智能体市场规模 | 372 亿美元（+197.6%） | 2025 | [艾媒白皮书](https://www.iimedia.cn/c400/113198.html) |
| 全球 AI 智能体市场规模（预测） | 3122 亿美元 | 2030E | [艾媒白皮书](https://www.iimedia.cn/c400/113198.html) |
| 协同办公市场规模 | 近 400 亿元 | 2025 | [第一财经](https://www.yicai.com/news/102714109.html) |
| 中国生成式 AI 用户规模 | 2.49 亿人 | 2024-12 | [CNNIC/中国高新网](http://www.chinahightech.com/yaowen/2025-01/17/content_287543.html) |
| WPS Office 全球月活设备 | 6.47 亿 | 2025-03 | [新华网](https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html) |
| 飞书多维表格月活 | 超 1000 万 | 2025-07 | [36氪](https://m.36kr.com/p/3371623528452615) |
| 钉钉 AI 表格服务企业数 | 超 30 万家 | 2025-08 | [中国工业报](https://www.cinn.cn/2025/09-01/eryvd8gD.html) |

### 7.2 用户行为数据

- 近八成职场人每周使用 AI 工具；56.1% 愿为 AI 付费（智联招聘，2025-09）。来源：[经济网](https://www.ceweekly.cn/zxfb/2025/0925/481469.html)、[中新经纬](https://www.jwview.com/jingwei/html/09-22/634407.shtml)
- 59.1% 使用者愿为 AI 办公智能体付费；47.2% 优先考虑跨生态可兼容性、46.1% 考虑厂商服务能力（艾媒）。来源：[艾媒白皮书](https://www.iimedia.cn/c400/113198.html)
- 办公智能体使用场景集中度：自动生成 PPT 51.8%、表格数据分析 38.2%、批量文件整理 37.1%（艾媒）。来源：[艾媒白皮书](https://www.iimedia.cn/c400/113198.html)
- 88% 企业已试点 Agentic AI、仅 24% 在多种应用场景实现 ROI（毕马威 2026 全球科技报告）。来源：[钛媒体](https://www.tmtpost.com/8099969.html)
- 中国桌面办公智能体访问量（2026-06，易观）：WorkBuddy 2097 万次 > TRAE IDE 1279 万 > QoderWork 788 万。来源：[钛媒体](https://www.tmtpost.com/8099969.html)
- 2025 年中国 AI 智能体市场同比 +123.2%，"爆发式增长"（艾媒）。来源：[艾媒](https://www.iimedia.cn/c400/113198.html)

### 7.3 机构趋势判断

- Gartner：到 2027 年超 60% 大型企业将在受监管环境采用私有化部署 AI 智能体。来源：[实在智能](https://www.ai-indeed.com/encyclopedia/27690.html)
- 艾媒：未来行业将"多智能体分工协作重构办公流程；安全治理体系成为产品标配；端云混合协同成为主流部署方式；付费模式多元化"。来源：[艾媒](https://www.iimedia.cn/c400/113198.html)
- 产业共识："Agent 从聊天到干活的第一块试验田是桌面办公"；"2026 年作为智能体从'对话 AI'向'行动 AI'全面跃迁的元年"。来源：[钛媒体](https://www.tmtpost.com/8099969.html)
- 行业瓶颈：办公智能体竞争的核心瓶颈"并非技术，而是价值兑现"——免费工具已可覆盖 PPT/表格/文件整理等高频场景。来源：[钛媒体](https://www.tmtpost.com/8099969.html)

---

## 八、对 gaea 办公板块的产品启示（操作建议）

### 8.1 直接可复用的市场验证

1. "多轮对话 + 修改可控 + 格式保留"是中文文档 AI 被验证正确的交互范式（WPS 灵犀、华为 OfficeClaw 均以此为核心卖点）。
2. "一键生成"已被证伪为不可交付（金山田然批评、PPT 工具横评、华为"可编辑性"痛点），gaea 的"草拟 + 拼装 + 人工改稿闭环"定位正确。
3. "数据不出域 + 纯本地"在政企和敏感行业是刚需而非卖点（农发行/渤海财险/甘肃武威案例），gaea 本地优先架构符合趋势。
4. MCP 已是事实标准（AhaSlides/飞书 aily/WorkBuddy/OfficeClaw 全部支持），gaea 的 MCP 扩展方向正确。
5. 知识库+成本库+交付物管理的"工作台"组合无直接对标，是差异化空间；WPS 知识库/飞书私域数据证明了知识沉淀的价值，但均绑定生态。

### 8.2 需要警惕的竞争与风险

1. **窗口期 12–24 个月**：腾讯 WorkBuddy（日活 1300 万）、阿里 QwenWork、华为 OfficeAce 都在快速补齐桌面 Agent 能力；一旦他们把"中文重文档"纳入路线图，垂直空间会被压缩。
2. **价值兑现是行业通病**：毕马威 24% ROI 数据意味着产品必须做出"可交付成果"的质量差异，否则会沦为免费工具可替代的"轻活"。
3. **合规红线**：标书/公文场景的法律风险（虚假业绩、雷同、串标、格式废标）需要产品内置合规机制（查重、溯源、人工复核流程），这既是壁垒也是责任。
4. **端侧算力限制**：本地大模型能力仍有限（端侧 AI 困于隐私与算力），gaea 的"云端统筹规划 + 本地执行"分层架构恰好规避了这一矛盾。

### 8.3 建议的产品叙事（话术参考）

- 对个人专业用户（标书代理/咨询/造价/行政）："文档不出本机，格式成品即交付"——对标 WPS 灵犀的格式保留、华为的可编辑性、易标书的查重废标检查。
- 对企业/政企："本地优先的分层智能"——对标 WPS AI 私有化、实在智能全栈本地化、甘肃武威政务模型。
- 对开发者/极客："本地工作台 + MCP 扩展"——对标 WorkBuddy/OfficeClaw 的 OpenClaw 兼容与技能体系。

---

---

## 附：主要参考来源汇总

- 金山办公 / WPS AI：新华网发布稿 https://www.xinhuanet.com/finance/20250728/64d6e8901aa74e7ebc6942049d353b16/c.html ；央广网·私有化方案 https://tech.cnr.cn/techph/20250529/t20250529_527189297.shtml ；WPS 论坛 https://bbs.wps.cn/topic/59892 ；中国新闻网·广东 http://www.gd.chinanews.com.cn/2025/2025-06-02/442319.shtml
- 飞书：36氪发布会 https://m.36kr.com/p/3371623528452615 ；中国日报·aily 任务模式 https://ex.chinadaily.com.cn/exchange/partners/82/rss/channel/cn/columns/sz8srm/stories/WS694108aaa310942cc4996e93.html ；飞书帮助中心 https://www.feishu.cn/hc/zh-CN/articles/790732948604 、https://www.feishu.cn/hc/zh-CN/articles/429896178269
- 钉钉：中国工业报·8.0/AI 钉钉 1.0 https://www.cinn.cn/2025/09-01/eryvd8gD.html ；东方财富·AI 钉钉 1.1 https://finance.eastmoney.com/a/202512233599866277.html ；极客公园 https://www.geekpark.net/news/353059
- 腾讯：环球网·腾讯文档接入 DeepSeek https://3w.huanqiu.com/a/074633/4LWSkaVFIZi ；钛媒体·Copilot 入华 https://www.tmtpost.com/6981843.html ；钛媒体·桌面办公 Agent 卡位战 https://www.tmtpost.com/8099969.html
- 标书垂直：华胜天成 https://www.teamsun.com.cn/newsdetail.htm?id=2510090001 ；易标书 GitHub https://github.com/fb208/OpenBidKit_Yibiao ；知乎·合规 https://zhuanlan.zhihu.com/p/2052792741961130963 ；阿里云开发者·合规全景 https://developer.aliyun.com/article/1754065
- 桌面/办公智能体：艾媒白皮书 https://www.iimedia.cn/c400/113198.html ；智东西·OfficeClaw https://m.zhidx.com/p/549631.html ；InfoQ·20 款桌面 Agent https://xie.infoq.cn/article/bb161038a4410ca327b8ba6b4
- 私有化/数据不出域：实在智能 https://www.ai-indeed.com/encyclopedia/27690.html ；钛媒体·大模型进不了内网 https://www.tmtpost.com/8070771.html ；证券时报 https://stcn.com/article/detail/1582536.html
- 用户心智：智联招聘（经济网转载） https://www.ceweekly.cn/zxfb/2025/0925/481469.html ；CNNIC（中国高新网） http://www.chinahightech.com/yaowen/2025-01/17/content_287543.html
- 市场规模：观研天下 https://www.gonyn.com/industry/1776655.html ；智研咨询 https://www.chyxx.com/industry/1224479.html ；第一财经 https://www.yicai.com/news/102714109.html
