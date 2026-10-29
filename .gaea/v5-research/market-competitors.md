# gaea V5 立项调研：2026 「个人 AI 助手 / 办公 Agent / 垂直 AI」市场事实底稿

> 调研时点：2026-10-29（Asia/Shanghai）｜对标产品：gaea v4.527（Windows 桌面、本地优先、造价 + 写作垂类）
> 证据分级：**A**＝官方/财报/官网定价页；**B**＝主流媒体、研究机构（QuestMobile、易观、IDC、Gartner）报道；**C**＝单一来源/自媒体/厂商口径/社区实测。
> 标注「报道口径」＝媒体转述、未见一手披露；标注「未取到」＝本次检索未取得可引用数值，**一律不填数字**。
> 方法与局限：本次检索环境的正文抓取通道被阻断（所有外链 fetch 返回 "resolves to a non-public IP"），因此事实主要来自**检索结果标题与摘要**。凡只有标题级证据的价格，均按「未取到具体月费」处理，不做推算。文末附「存疑与冲突清单」，逐条说明哪两个来源打架。

---

## 一、执行摘要（10 条）

1. **办公 Agent 已经打成「入口战争」，不再是产品战争。** 易观 2026-10-20 报告：**20 款桌面端 AI 原生办公 Agent 月访问量合计超 3 亿次**（B，[经济观察网](https://m.eeo.com.cn/2026/1020/1070979.shtml)）。同期 QuestMobile：9 月 **AI 原生 App 月活 5.46 亿，豆包 4.19 亿断层第一**（B）。
2. **三强格局初显且座次已定。** QuestMobile 9 月办公类 Agent 月活前三：**腾讯 WorkBuddy 3235.3 万（环比 +647.1%）、百度搭子 977 万、千问办公**（B）；千问办公 9 月初宣布**用户破 3000 万**（B，21 世纪经济报道 2026-09-04）。「办公 Agent 三强格局初显」已成报告标题级判断（B）。
3. **企业按席位收钱，仍是 2026 年唯一被验证的规模化变现。** 微软 **Microsoft 365 Copilot 付费席位 2026 年达 1500 万、同比 +160%**，$30/席/月，机构预测 2027 年中 6000 万席（B，199it / 36Kr）。
4. **消费侧价格天花板被压到「奶茶价」。** OpenAI **ChatGPT Go 全球 $8/月（含广告）**；阿里千问办公**高级套餐 19 元/月（2 倍额度）**；Google **AI Ultra 由 $250 降至 $100/月**、AI Plus 降价近四成（B/C）。个人订阅的 ARPU 上限已被钉死在 **￥19–￥60 / $8–$20** 量级。
5. **连 OpenAI 自己都不信「个人 flat 订阅」：** ChatGPT Work 改成**按用量计费**，媒体测算工作负载落在 **$100–200/月**（B，Business Model Analyst / GeekPark：「OpenAI starts pricing ChatGPT by the work it does」）。
6. **个人 Agent 创业是结构性难题，不是执行力问题。** 前豆包 PC 端负责人**齐俊元**创办 Today，**7 个月融资 13 亿元人民币（超 2 亿美元）**，腾讯、蚂蚁下注，媒体定调仍是「**做 AI 助理，但不好赚钱**」（B，2026-10-27）；同期评论直言「**个人 Agent 创业潮来了，但这门生意只适合大厂**」（B，证券之星 2026-10-26）。
7. **中间层（通用 Agent 应用）被两头挤：** Manus 估值 **40 亿美元**、收入半年翻倍（B），但被写成「**被 AI 大厂重新定价**」（B，钛媒体 2026-10-12）；Kuse **毛利微正**、Cursor **毛利率 -23%**（B，观点网）——应用层毛利结构性偏薄。
8. **垂直造价是「高集中 + 高付费 + 强替代叙事」的三高市场。** 广联达数字造价**市占率超 90%**（C，行业研究平台）；厂商称**土建提量准确率 98%**（C，厂商口径，2026-10-14）；从业者横评 6 款 AI 算量工具得出「**22 天压到 4 小时**」与「**免费的那款最贵**」（C，自媒体实测）。
9. **造价岗位收缩正在给「工具替代人力」提供付费理由，但买单方是公司不是个人。** 「AI 正在吃掉造价基础岗」「造价咨询公司批量裁员」成 2026 年行业高频标题（C）；同时从业者抱怨「**某联达的价格已经高得离谱，小公司基本没能力供一把软件锁**」（C）——个人从业者的支付能力与软件的定价模型错配。
10. **本地/隐私是政策与信任双驱动的真空位。** 网信办已公示**7 款手机端侧生成式 AI 服务完成备案**（B，2026-07）；IDC 预判**中国 AI 主机 2030 年市场规模 770 亿元**（B，2026-09）；而**豆包 2026-07-15 永久下线「AI 智能体」并 10-15 清除数据、引导迁往收费产品「猫箱」**（B/C），把「平台可单方面收回你的 AI 资产」变成了大众议题——这正是「本地优先」最锋利的一刀。

---

## 二、竞品矩阵表

### 2.1 通用 / 办公 Agent 阵营

| # | 产品（阵营） | 定位 | 价格（公开口径） | 分发入口 | 目标用户 | 证据 / 日期 |
|---|---|---|---|---|---|---|
| 1 | **Claude Cowork**（Anthropic） | 桌面通用 Agent，面向非程序员的「文件级干活」：整理文件夹、批量改文档、跨文件汇总 | 含在 **Claude Pro $20/月**（2026 年内由 Max 下放至 Pro）；Max **$200/月（$2,400/年）**；Team/Enterprise 席位制（单价未取到） | claude.ai 桌面端 + Claude Code 同源 | 白领、知识工作者、中小企业 | A/B：Anthropic Help「Use Cowork on Team and Enterprise」；极客网 2026-01-19 发布报道；BI 2026-06（$2,400/年） |
| 2 | **ChatGPT（Go / Plus）** | 通用个人助手 | **Go $8/月（含广告）**；Plus 长期锚点 **$20/月** | 全球 App/Web，分发能力最强 | 大众个人 | B：IT之家/新浪 2026；AWS Builder「$20/Month: The Price Ceiling」 |
| 3 | **ChatGPT Work + Workspace Agents**（OpenAI） | 把 agent 做成「企业工作流」，可直连 Slack、Salesforce；custom GPTs 的继任者 | **按用量计费**，媒体测算 **$100–200/月**；Business 席位另计 | ChatGPT 内 + 企业 SaaS 插件 | 团队/企业 | B：VentureBeat（Workspace Agents）；GeekPark「pricing by the work it does」 |
| 4 | **Microsoft 365 Copilot** | 办公套件内嵌 Copilot（Word/Excel/Teams/Outlook） | **$30/席/月**；**1500 万付费席位（2026，+160%）** | M365 装机量 + 企业协议 | 中大型企业 | B：EPC Group 定价指南 2026；199it（1500 万席）；36Kr（2027 中 6000 万席预测） |
| 5 | **Microsoft 365 E7**（微软 2026 新档） | 把 AI 塞进更高价套餐，「以 AI 提振销量」 | 比原档**高 50%+**（报道口径） | 企业 EA/CSP | 大型企业 | B：极客网 2026-03-10；Times of India |
| 6 | **Copilot Cowork**（微软 × Anthropic，报道） | 微软版「Cowork」，与 Anthropic 合作开发 | 未公布 | M365 | 企业 | B：Digital Today（「develops Copilot Cowork with Anthropic」） |
| 7 | **Google Gemini（AI Plus / AI Ultra）** | 个人订阅 + 全家桶捆绑（YouTube/存储） | **AI Ultra $100/月（原 $250）**；AI Plus 降价近四成、云存储翻倍至 400GB | Android/搜索/Workspace 默认入口 | 大众个人 | B：Digital Today 2026；电脑王阿达；⚠️另有来源称 Ultra 降至 $200，见存疑清单 |
| 8 | **Gemini for Workspace** | 企业办公套件内 AI，按席位/打包 | 未取到单价；2026 年出现「AI 重打包条款」争议 | Workspace 企业客户 | 企业 | B：Redress 采购白皮书；c114（取消按次计费、转算力计量） |
| 9 | **腾讯 WorkBuddy**（腾讯云 CodeBuddy 系） | 桌面 AI 办公 Agent（「敢拒单」的重执行型，全场景办公工作台） | 订阅制，官网有独立定价页（workbuddy.cn/pricing），**本次未取到具体月费** | 独立客户端 + 腾讯生态 | 个人白领 / 中小团队 | B：QuestMobile 9 月 **MAU 3235.3 万、环比 +647.1%**，榜单标注「腾讯 WorkBuddy」；雷科技五强横评。⚠️**任务书误记为「字节」，见附录 #1** |
| 10 | **字节 豆包**（含豆包电脑版） | 通用助手（含智能体）→ 办公 | 免费为主，2026 年「探索付费」 | 超级 App，**MAU 4.19 亿** | 大众个人 | B：QuestMobile 9 月；搜狐/新浪 2026-08-12「千问、豆包探索付费」。**字节未取得独立的办公 Agent 单品数据**（扣子空间/豆包办公未单列），故字节一侧以豆包（通用助手）代表，不单列办公 Agent 单品 |
| 11 | **阿里 千问办公（QwenWork）** | 个人版 + 企业版办公 Agent | **高级套餐 19 元/月（2 倍额度）** | 千问 App/PC + 钉钉/阿里云 | 个人白领 → 企业 | B：PChome 2026；千问办公定价页（个人版/企业版） |
| 12 | **阿里 夸克（PC 版）** | 浏览器即 AI 工作台 | 免费 + 会员（未取到单价） | 浏览器（装机入口） | 学生 / 年轻白领 | B：IT168 2026-07-31 实测「用夸克 PC 做完大模型调研」 |
| 13 | **腾讯 Buddy One + 腾讯网盘** | 跨应用 AI 协同体系，打通腾讯文档/会议/网盘 | 腾讯「AI 大会员」体系（未取到单价） | 微信/QQ/腾讯文档全家桶 | 个人 + 中小企业 | B：2026-10-29 上线（chinaz/iheima）；招商证券国际点评 |
| 14 | **百度 搭子 / GenFlow 4.0 / 库库 AI** | 办公智能体 + 通用智能体 + 独立端 | 免费为主，DuMate 走「积分扣减」计费 | 百度文库 / 百度网盘（亿级存量入口） | 个人白领 / 学生 | B：**MAU 977 万**（新浪 2026-10-20）；GenFlow4.0 2026-04-28；库库 AI 2026-08-14 |
| 15 | **金山办公 灵犀 / WPS Comate** | 「AI 原生办公产品」，WPS 内 AI 副驾 | 灵犀专业版（未取到单价） | **WPS 全球月活 6.78 亿** | WPS 存量个人 + 企业 | B：新华网 2026-07-17（灵犀专业版+Comate 同日发布）；东方财富 2026-10-27（Q3 净利同比降逾七成、月活 6.78 亿） |
| 16 | **钉钉 悟空 / 飞书智能伙伴** | 企业 IM 内 Agent（审批、报表、群内执行） | 随企业版订阅（未取到单价） | 钉钉/飞书（中国企业 IM 默认入口） | 企业 | B：36Kr 五款 Agent 实测「悟空接钉钉」；钉钉悟空竞品对比文档 |
| 17 | **Manus** | 通用任务 Agent（交付物导向） | 订阅 + 积分（未取到单价） | 海外独立站为主 | 海外专业用户 / 早期尝鲜者 | B：估值 **40 亿美元**（EEPW 2026-10）；「收入半年翻倍不止」；钛媒体「被 AI 大厂重新定价」2026-10-12 |
| 18 | **Genspark** | AI workspace / 多 agent 工作台 | 订阅 + 积分（Business 定价页，未取到单价） | 海外独立站 | 海外中小团队 | B：官网 Business Pricing；「Manus 与 Genspark 半年收入翻倍、跑通变现」（投资界） |
| 19 | **Kuse** | 画布式 AI workspace（知识 + 产出） | 订阅 + 积分 + Team 账单 | 海外独立站 | 海外知识工作者 | B：Kuse 官方计费文档；观点网「**Kuse 毛利微正、Cursor 毛利率 -23%**」 |
| 20 | **OpenHuman** | **开源、免费**桌面 Agent，可接 100+ 应用，可读记忆 | **$0（开源）** | GitHub（4 个月 **31K stars**） | 极客 / 隐私敏感 / 自建派 | C：腾讯云社区「4 个月 31K stars」；钛媒体「越用越后怕」（权限与信任） |

### 2.2 垂直阵营

| # | 产品 | 定位 | 价格（公开口径） | 分发 | 目标用户 | 证据 |
|---|---|---|---|---|---|---|
| 21 | **广联达 造价 AI**（GTJ / AI 提量 / 斑马） | 计价 + 算量 + 进度的全链条 | 软件锁 + 年费（从业者称「高得离谱，小公司供不起」） | 造价站/分公司/院校 + 加密锁 | 造价咨询、施工方、甲方 | C：厂商称**土建提量准确率 98%**（观点网 2026-10-14）；市占率**超 90%**（三个皮匠） |
| 22 | **鲁班软件 LTJ2026** | 智能土建算量平台（2026 新品） | 未取到 | 直销 + 渠道 | 施工/造价 | B：鲁班官网新品发布会 2026 |
| 23 | **品茗科技 / 新中大** | 造价与工程管理软件（上市公司/老牌） | 未取到 | 渠道 + 招标 | 中小施工企业 | B：同花顺调研 2026-09-08；八款工程管理软件横评 |
| 24 | **Togal.AI**（海外） | 图纸 AI 识图算量（takeoff），面向 GC estimator | 报价制（quote-based），未取到单价 | 海外直销 | 海外总包/estimator | B：PitchBook 档案；公开融资记录仅见 **Pre-A 500 万美元**（Owler 汇总） |
| 25 | **Exayard / CostFlowAI / FloorScan** | 新一代 AI 算量/造价 SaaS（订阅制） | 订阅套餐（官网定价页，未取到单价） | 独立站 + 试用 | 中小造价团队 | B：各自定价页（exayard.com / costflowai.com / floorscan.ai） |
| 26 | **利驰 图晓晓 AI 识图 + 报价工作室** | 电气成套「识图→报价」一体化，**年费制** | 年费（按「个」卖，签约快报可见） | 直销 + 续约 | 电气成套厂 | B：利驰官网签约快报 2026-09/10 多条（增购/续约年费服务） |
| 27 | **NovelCrafter** | 长篇小说 AI 写作（codex 设定库 + 按需模型） | **$4–$20/月 + 自付 AI 成本** | 独立站 | 英文长篇小说作者 | B：Novel Mage 2026 定价拆解；官网定价页 |
| 28 | **Sudowrite** | 小说写作 AI（Story Bible / 续写） | **4 档 $10–$59/月**（另有来源称 Hobby $19 起） | 独立站 + App Store | 英文小说作者 | B：CostBench 2026（$10–59）；CompareEdge（$19 起）⚠️口径不一 |
| 29 | **彩云小梦** | 中文 AI 续写/角色陪伴 | 会员制（未取到） | App | 中文网文/同人作者 | C：第三方社区 2026 年文章以《彩云小梦还能用吗？》讨论替代方案，指向服务已不可用或大幅受限；**官方停服公告未取得，待核实** |
| 30 | **阅文 Claw / 作家助手「小龙虾」/ Buddy 系列** | 平台侧 AI 创作工具（IP 全生命周期） | 随作家平台提供 | 阅文作家助手（**上半年新增 24 万作家**） | 网文作者 | B：人民网 2026-08-12；中国证券网（Claw、Buddy 系列） |
| 31 | **番茄小说 AI 治理** | 平台用 AI 反 AI（下架/封号）+ 作者 AI 工具 | — | 番茄 App | 网文作者 | C：**一个月下架 4 万本书、封 855 个号、67% 作者在用 AI**（今日头条） |
| 32 | **广联达 斑马进度** | 进度计划（P6 的国产替代）+ AI 生成总控计划 | 免费版 + 专业版（未取到单价） | 广联达渠道 | 项目经理/计划工程师 | B：2026-04-17 发布 AI 智能方案；获 2026 建造业议会 AI 大奖；C：「履约率只有 23%，30 分钟出总控计划」 |
| 33 | **Oracle Primavera P6** | 工程进度计划事实标准（EPPM / P6 Pro） | 按许可/订阅，报价制（未取到单价） | Oracle 直销 + 代理 | 大型工程/EPC | B：Oracle Shop 产品页；第三方 licensing 对比 2026 |
| 34 | **CAD 快速看图（广联达）/ 浩辰 CAD** | 图纸查看/测量/批注 | VIP 会员 + 批量授权码 | 广联达/浩辰渠道 | 造价、施工、设计 | B：官网 VIP 购买页、批量授权页；浩辰 2026 调研 |

---

## 三、市场规模、价格带与结构

### 3.1 规模（只列可引用口径）

| 口径 | 数值 | 来源 | 日期 | 等级 |
|---|---|---|---|---|
| 中国 AI 办公智能体市场 | **2026 年预计 449 亿元** | 观点网（引报告） | 2026-08-25 | B |
| 桌面端 AI 原生办公 Agent | **20 款产品月访问量合计超 3 亿次** | 易观分析 | 2026-10-20 | B |
| 中国 AI 原生 App | **9 月月活 5.46 亿；豆包 4.19 亿** | QuestMobile | 2026-09 | B |
| 办公类 Agent 月活（单产品） | WorkBuddy **3235.3 万**（+647.1% QoQ）；百度搭子 **977 万**；千问办公前三 | QuestMobile | 2026-09 | B |
| 千问办公用户 | **破 3000 万** | 21 世纪经济报道 | 2026-09-04 | B |
| 企业级 Agent（全球） | **超 220 亿美元** | 巴西/葡语媒体报道行业报告 | 2026-09-16 | C（转述） |
| 中国 AI 主机（端侧/本地硬件） | **2030 年 770 亿元** | IDC 预判（白皮书） | 2026-09 | B |
| 工程造价软件（全球/中国） | **未取到具体规模数值** | Research and Markets / 共研 / 168 报告均有付费报告 | 2026 | —（不填） |
| 个人 AI 助理 / AI 知识库（全球） | **未取到具体规模数值** | Research and Markets「Personal AI Assistant Market Report 2026」「Personal Knowledge Base AI Market Report 2026」 | 2026 | —（不填） |

> 说明：Research and Markets 的《Personal AI Assistant Market Report 2026》《Personal Knowledge Base AI Market Report 2026》确实存在（可检索到条目），但正文付费墙 + 本次抓取通道被阻断，**未取得数值**。任何「XX 亿美元」的引用都需二次核验，本底稿不填。

### 3.2 价格带（2026 年 10 月的五个台阶）

| 台阶 | 代表价 | 代表产品 | 含义 |
|---|---|---|---|
| 免费/开源 | ￥0 | 豆包、OpenHuman、斑马免费版、CAD 快速看图基础版 | 获客与防御性定价，挤压独立工具生存空间 |
| 奶茶价 | **$8/月、￥19/月** | ChatGPT Go、千问办公高级套餐 | 个人付费意愿的真实天花板 |
| 主流订阅 | **$20/月** | Claude Pro（含 Cowork）、ChatGPT Plus、Google AI Plus | 「$20 锚点」被全行业复制 |
| 重度个人 | **$100–$200/月** | Claude Max（$200）、ChatGPT Work（按量，$100–200）、Google AI Ultra（$100） | 只有重度专业用户/团队能承受 |
| 企业席位 | **$30/席/月** 起 | M365 Copilot | 唯一被验证的规模化收入模型；但按席位卖会被「按结果卖」挑战 |

**价格结论：**
- 个人侧「$8–$20」是硬顶；**任何面向个人的通用助手，单靠订阅很难覆盖推理成本**（Kuse 毛利微正、Cursor 毛利率 -23% 是直接证据）。
- 企业侧「$30/席」仍在增长（1500 万席、+160%），但微软已开始用 **E7 高价套餐 + 计量（Copilot Credits）** 双层收割，说明**纯席位制也在被成本逼着改**。
- 中国侧价格战更狠：**19 元/月** 已经能买到「2 倍额度」的办公 Agent；电信甚至出现 **9.9 元/1000 万 Token** 的算力促销（C）。

### 3.3 分发：2026 年真正的护城河是「入口」，不是模型

- **微信/QQ 系**：腾讯 Buddy One 一天内打通文档、会议、网盘（2026-10-29 上线），把 Agent 塞进已有关系链；底座之上再叠一层桌面执行型 Agent——**WorkBuddy（腾讯云 CodeBuddy 系）MAU 3235.3 万、单月环比 +647.1%**，是「入口 + 动作」双层结构的样本。
- **WPS 系**：金山办公全球月活 **6.78 亿**，灵犀/Comate 直接长在文档里。
- **百度文库/网盘**：亿级存量 + 「找资料」原生场景，搭子 977 万月活主要靠此。
- **浏览器/操作系统**：夸克 PC、Edge/Copilot、Apple 端侧——**入口在别人手里，独立 App 的获客成本被结构性抬高**。
- **开源社区**：OpenHuman 4 个月 31K stars，是唯一「不依赖巨头入口」的分发路径，但变现与信任（权限）双弱。

### 3.4 个人 vs 企业：分化已经写进产品定义

钛媒体 2026 年「AI 进展观察」系列把这件事讲透了：**个人 Agent 与企业 Agent 在定位、功能、商业模式上已分道**（B，《个人 Agent 与企业 Agent：定位分化、全球格局与演进路径》，以及《个人与企业研发 Agent，功能到底差在哪？》）。核心差异：

| 维度 | 个人 Agent | 企业 Agent |
|---|---|---|
| 付费主体 | 个人（价格敏感，￥19/$8 锚点） | 公司（预算制，$30/席） |
| 成功标准 | 「省我时间」、好玩、情绪价值 | 可审计、可管控、ROI 可算 |
| 数据 | 个人文件/聊天记录（信任敏感） | 内网/业务系统（不出域） |
| 分发 | 应用商店/社交裂变 | 渠道/集成商/企业协议 |
| 变现难度 | **极高**（ARPU 低、流失高、推理成本刚性） | 中（销售周期长但 LTV 高） |

**变现难的三条硬证据：**
1. **齐俊元 Today**：7 个月 13 亿元融资、腾讯+蚂蚁下注，媒体标题仍是「**做 AI 助理，但不好赚钱**」（B，2026-10-27）。
2. **「个人 Agent 创业潮来了，但这门生意只适合大厂」**（B，证券之星 2026-10-26）——因为获客要靠巨头入口、成本要靠巨头算力。
3. **「有用户，有收入，AI 应用却不是好生意」**（B，老虎社区 2026）——Kuse 毛利微正、Cursor 毛利率 -23%。

### 3.5 失败 / 降温 / 信任事故（2026）

| 事件 | 事实 | 教训 | 来源 |
|---|---|---|---|
| **豆包「AI 智能体」永久下线** | 2026-07-03 通知：**7-15 永久下线，10-15 清除用户数据**，引导迁往收费产品「猫箱」；用户称记忆/角色/风格无法继承 | 用户自建的 Agent 资产**不属于用户**；平台可单方面收回 → 本地化/可导出的价值主张 | B/C：虎嗅《豆包、千问于〈人工智能拟人化互动服务管理暂行办法〉生效前下线智能体功能，引发用户权益争议》；钛媒体《豆包、千问，挥刀砍向数字生命》 |
| **企业 Agent 上线 7 天关停** | 老板花 **50 万元**做智能客服 Agent，上线一周关停，日均 2000 单跌至不足 50 单 | 「Demo 惊艳 ≠ 生产可用」；缺少兜底与运营即死 | C：dbaplus《老板花 50 万搞了个 AI Agent，上线一周就关了》 |
| **AI 陪伴赛道集体退潮** | 「赛道顶流陨落，国内玩家折戟」；Moemate、Moxie 等关停记录被整理 | 情绪价值产品**留存与合规双杀** | B/C：界面/投中网 2026-10；PocketAnimus 关停记录 |
| **AI 应用层不赚钱** | 高增长、有收入，但毛利极薄 | 通用 Agent 无壁垒，价格战即利润战 | B：观点网、老虎社区 |
| **平台反 AI 内容** | 番茄一个月**下架 4 万本书、封 855 个号**，同时 **67% 作者在用 AI** | 平台既卖 AI 工具又打击 AI 内容，工具方需选边 | C：今日头条 |

---

## 四、垂直赛道深挖

### 4.1 垂直造价（gaea 主战场之一）

**市场结构：一家独大 + 高切换成本 + 加密锁经济。**

- **集中度**：广联达数字造价**市占率超 90%**（C，三个皮匠行业研究）；2025 年净利润**同比 +61.77%**（B，证券时报），说明主业仍在赚钱。
- **能力宣称**：2026-10-14 发布造价全场景 AI 产品，称**土建提量准确率 98%**（C，厂商口径，观点网/中国建设报）。**这是厂商自测口径，不等于第三方复现。**
- **实测反证**：从业者用同一套图纸横评 **6 款 AI 算量工具**，结论是「**22 天变 4 小时**」且「**免费的那款最贵**」（C，自媒体）。「免费最贵」的具体机制（导出受限/人工返工/精度不足）未取得原文细节，**仅标题级证据**。
- **竞品动作**：鲁班 2026 年发 LTJ2026 智能土建算量平台（B）；品茗、新中大在工程管理软件横评中常年同框（B）；海外 **Togal.AI** 用「图纸识别 → takeoff」切总包预算员，公开融资仅见 Pre-A 500 万美元（B/C）。
- **新势力**：Exayard、CostFlowAI、FloorScan 等订阅制 AI 算量 SaaS 已上线定价页（B），说明「轻量、按订阅、面向小团队」的缝隙**已经有人在试**。
- **岗位收缩**：2026 年行业高频标题——「AI 正在吃掉造价基础岗」「造价咨询公司批量裁员」「造价行业进入深水区：缩水的不只是工资，更是职业预期」（C）。这既是**付费理由**（老板买工具替代人力），也是**付费阻力**（从业者收入预期下降）。
- **付费习惯**：造价软件长期是「**加密锁 + 年费 + 按节点**」的重资产采购，决策权在公司/所长，个人从业者**没有独立付费习惯**；且价格被普遍认为过高（「小公司基本没能力供一把软件锁」，C）。**这恰好是「个人版/轻量版」的空位，也是 gaea 这类桌面工具的机会窗口。**

**结论**：造价是「**高价值、高集中、低个人付费**」市场。要吃到它，必须绕开「替代广联达」的正面战场，切「**广联达不做/做不好的个人环节**」：读图、提量核对、计价复核、跨文件汇总、离线可用。

### 4.2 垂直写作

- **海外工具价格带**：NovelCrafter **$4–$20/月 + 自付 AI 成本**（B）；Sudowrite **4 档 $10–$59/月**（B，另有 $19 起口径，见存疑）。两者都证明：**长篇作者愿意为「设定一致性/上下文管理」付费**，而不是为「生成一段文字」付费。
- **中文侧**：彩云小梦服务状态存疑（C，待核实）；阅文走「平台内工具」路线——Claw、作家助手「小龙虾」、Buddy 系列（B），**上半年新增 24 万作家、收入 +10.7%**（B）。平台把 AI 变成作者留存工具，独立工具很难切入。
- **付费意愿的真实分布**：番茄 **67% 作者在用 AI**，但 **85% 新人一年稿费不够买手机**（C）、「3000 万网文作者」基数（C）。**用 AI 的人多，能付钱的人少**——付费意愿集中在中腰部以上、且已经在用 NovelCrafter 类工具的那批人。
- **风险**：平台反 AI（番茄下架 4 万本/封 855 号），**工具方容易被当成「作弊器」**；产品叙事必须站在「辅助创作/一致性管理」而非「批量生成」。

### 4.3 进度计划 / CAD 读图

- **P6**：工程进度的事实标准，EPPM/P6 Pro 按许可报价（B，Oracle Shop）；价格不透明、实施重，**中小企业与个人计划工程师被排除在外**。
- **斑马进度**：广联达的国产替代 + AI 生成总控计划（2026-04-17 发布 AI 智能方案；获 2026 建造业议会 AI 大奖）。自媒体称「**履约率只有 23%，AI 斑马 30 分钟出总控计划**」（C，标题级）。**「计划生成」已被验证有 demand，但「计划执行/纠偏」仍是空白。**
- **CAD 读图**：广联达 CAD 快速看图（VIP + 批量授权）、浩辰 CAD（AI 赋能，2026 调研）、利驰「图晓晓 AI 识图 + 报价工作室」**按年费卖**（B，签约快报可见持续续约/增购）。**「识图 → 工程量 → 报价」的链条在国内已被拆成多个年费产品，但没有一个「本地、离线、个人可买」的整合体。**

### 4.4 本地 / 隐私赛道

- **合规底座**：网信办已公示 **7 款手机端侧生成式 AI 服务完成备案**（OPPO、苹果、华为、小米、vivo 等在列，B，2026-07）；《人工智能拟人化互动服务管理暂行办法》2026 年生效，直接导致豆包/千问下线智能体功能（B）。
- **企业需求**：「**数据不出域，AI 才能真正落地**」成为服务商标准话术（C）；钛媒体《大模型进不了内网之后，企业 AI 的生意才真正开始》（B）——**私有化/本地部署是 2026 年企业侧的确定性需求**。
- **硬件信号**：IDC 预判 **中国 AI 主机 2030 年 770 亿元**（B，2026-09）；「AI 手机之后，AI 主机也出现了」（B，2026-09-19）；Apple Mac 营收超预期被归因于**本地 AI 需求**（B）。
- **对 gaea 的含义**：**「本地优先 + 数据不出域 + 可导出」不是情怀，是 2026 年唯一能同时对抗「平台收回资产」和「企业合规」的叙事**。但要注意：本地模型能力弱于云端，**必须做成「本地优先 + 可选云增强」的混合架构**，否则会在效果上被豆包/千问碾压。

---

## 五、空位与进入壁垒

### 5.1 四个可验证的空位

1. **「本地优先」的个人知识工作台**
   - 需求证据：豆包智能体下线 + 清数据（B）、企业「数据不出域」（B）、端侧备案开闸（B）。
   - 竞争现状：OpenHuman 开源但权限吓人（C「越用越后怕」）；巨头产品全部云优先。
   - 风险：**用户嘴上在乎隐私，手上选方便**；必须把「本地」翻译成可感知的好处（离线可用、不怕封号、文件不上传、可带走）。

2. **造价的「个人版/轻量版」**
   - 需求证据：广联达锁贵、小公司供不起（C）；6 款工具实测「免费最贵」（C）；岗位收缩带来的「一个人干三个人的活」（C）。
   - 竞争现状：广联达全链条但重；鲁班/品茗/新中大同样重；新势力 SaaS 尚早期。
   - 风险：**造价数据（定额库、图集、地区规则）是真正的护城河**，没有数据就只能做「读图/整理」的浅层工具。

3. **中文长篇小说「一致性引擎」**
   - 需求证据：NovelCrafter 用 codex 验证了付费意愿（B）；中文侧只有平台内工具（阅文）与状态存疑的彩云小梦（C）。
   - 风险：**平台反 AI**；且中文作者付费能力弱于英文作者（85% 新人稿费不够买手机，C）。

4. **「读图 → 算量 → 计划」的本地闭环**
   - 需求证据：利驰按年费卖识图（B）、斑马卖 AI 计划（B）、CAD 看图卖 VIP（B）——**三个年费产品拼起来才是用户的一天**。
   - 风险：每一段都有强势 incumbent；闭环需要跨格式解析能力（DWG/PDF/Excel/Project）。

### 5.2 五道进入壁垒（按硬度排序）

| 壁垒 | 说明 | gaea 的可达性 |
|---|---|---|
| **分发入口** | 微信/QQ、WPS 6.78 亿、百度文库网盘、浏览器 | 低——只能靠桌面端 + 口碑 + 垂类社区 |
| **价格** | ￥19/月、$8/月、免费 | 中——本地推理边际成本低，可打「一次性/买断」差异 |
| **模型成本** | 按量计费下，通用 Agent 毛利被压穿（Kuse 微正、Cursor -23%） | 高——本地推理把边际成本转给用户硬件 |
| **合规** | 拟人化办法、端侧备案、内容安全 | 中——本地工具责任边界更清晰，但不能碰「情感陪伴」 |
| **数据资产** | 定额库/图集/规范/模板；写作的设定库 | 低→中——需长期积累，但一旦形成即高粘性 |

### 5.3 对 gaea V5 的六条判断

1. **不要做「又一个通用办公 Agent」**：入口、价格、算力三样都不在 gaea 手里，WorkBuddy 一个月 +647% 的增长不是产品力，是入口力。
2. **「本地优先」要作为默认架构，不是营销标签**：豆包下线事件证明了「云端资产可被收回」，这是唯一能讲给个人用户听的、带情绪的差异点。
3. **造价必须做「个人可买」的定价**：年费/买断 + 免费试用，绕开加密锁与公司采购；先做「读图/提量核对/计价复核」这些**广联达不做或做得很重的环节**。
4. **写作要卖「一致性」而不是「生成」**：codex/设定库/上下文管理是已验证的付费点；避开「批量生成」的伦理与平台风险。
5. **混合推理是必选项**：本地小模型做隐私敏感与离线场景，云端大模型做质量兜底；按任务路由，而不是二选一。
6. **警惕「个人 Agent 不赚钱」的结构性陷阱**：如果 V5 的收入模型仍是「个人月订阅」，就要正面回答「凭什么比 19 元的千问办公更好」；**买断/一次性 + 垂类数据订阅**可能是更现实的路。

### 5.4 仍需验证的 8 个问题（下一轮调研）

1. WorkBuddy、灵犀专业版、Buddy One、百度搭子的**具体月费**（官网定价页正文）。
2. 广联达造价 AI 的**实际客单价/续费率**（财报电话会、调研纪要）。
3. 6 款造价 AI 实测的**完整方法与打分**（原文正文，而非标题）。
4. Togal.AI 的**当前融资与 ARR**（PitchBook 需订阅）。
5. Research and Markets 两份报告的**具体市场规模数值**（付费墙）。
6. 彩云小梦**是否已停服**（官方公告）。
7. 豆包智能体下线后的**用户迁移数据**（猫箱 DAU 变化）。
8. 造价基础岗**招聘量的量化下降**（招聘平台数据，而非自媒体标题）。

---

## 六、参考链接

> 按主题分组；日期为报道/发布日；标注「标题级」表示仅取得标题与摘要，未读正文。

**通用 / 办公 Agent**

- Claude Cowork 发布（2026-01-19）— https://ai.zhiding.cn/2026/0119/3177152.shtml
- Anthropic Help：在 Team / Enterprise 方案上使用 Cowork — https://support.claude.com/en/articles/13455879-use-claude-cowork-on-team-and-enterprise-plans
- Anthropic Help：开始使用 Claude Cowork — https://support.claude.com/zh-CN/articles/13345190
- Claude Cowork 扩展至 $20 Pro 订阅用户（网易）— https://m.163.com/dy/article/KJG5A05V0511BLFD.html
- Claude Max $2,400/年用户报道（Business Insider, 2026-06）— https://www.businessinsider.com/claude-max-20x-ai-plan-subscribers-2026-6
- Claude 定价反复横跳（华尔街见闻）— https://awtmt.com/articles/3771400
- M365 Copilot $30/席/月（EPC Group, 2026）— https://www.epcgroup.net/microsoft-365-copilot-pricing-licensing-enterprise-guide-2026
- Copilot 付费席位 1500 万、+160%（199it）— http://www.199it.com/archives/1810330.html
- Copilot 席位 2027 年中达 6000 万（36Kr）— https://36kr.com/p/4005263791511176
- 微软 M365 E7 订阅计划（2026-03-10）— https://ai.zhiding.cn/2026/0310/3180740.shtml
- Copilot 拆分与计量（Aragon Research）— https://aragonresearch.com/copilot-unbundled-prepare-for-metered-ai/
- 微软开发 Copilot Cowork（Digital Today）— https://www.digitaltoday.co.kr/en/view/33710
- ChatGPT Go $8/月（IT之家）— https://m.ithome.com/html/0913989.htm
- OpenAI Workspace Agents（VentureBeat）— https://venturebeat.com/ai/openai-unveils-workspace-agents-a-successor-to-custom-gpts-for-enterprises-that-can-plug-directly-into-slack-salesforce-and-more
- OpenAI 按工作量计费（GeekPark）— https://about.geekpark.net/at-devday-openai-starts-pricing-chatgpt-by-the-work-it-does/
- ChatGPT Work $100–200/月测算 — https://businessmodelanalyst.com/openai-chatgpt-work-token-meter/
- OpenAI 年化营收约 500 亿美元（2026-09 底）— https://d.drcnet.com.cn/?docid=8352020
- Google AI Ultra 降至 $100（Digital Today）— https://www.digitaltoday.co.kr/cn/view/56986
- Google AI Plus 降价、存储翻倍 — https://m.163.com/dy/article/KV2Q8BIT0511BLFD.html
- Gemini 取消按次计费转算力计量（c114）— https://www.c114.net.cn/industry/83059.html

**中国办公 Agent**

- 易观：20 款桌面端 AI 原生办公 Agent 月访问超 3 亿次（2026-10-20）— https://m.eeo.com.cn/2026/1020/1070979.shtml
- QuestMobile 9 月办公 Agent 前三（2026-10-20）— https://www.techweb.com.cn/it/2026-10-20/2979859.shtml
- WorkBuddy 月活 3235.3 万、+647.1%（199it）— http://www.199it.com/archives/1858394.html
- 百度搭子 9 月月活 977 万（新浪 2026-10-20）— https://finance.sina.com.cn/roll/2026-10-20/doc-inivwitm5792786.shtml
- 千问办公用户破 3000 万（21 世纪经济报道 2026-09-04）— https://www.21jingji.com/article/20260904/herald/6a0e86cb5466d58ac88b65c6a0ab3ad1.html
- 千问办公高级套餐 19 元/月（PChome）— https://article.pchome.net/news/15475.html
- 千问办公定价页 — https://qwenwork.cn/pricing
- 千问、豆包探索付费（2026-08-12）— https://cj.sina.cn/article/norm_detail?url=http%3A%2F%2Ffinance.sina.com.cn%2Fjjxw%2F2026-08-12%2Fdoc-ininaezi3595434.shtml
- 腾讯 Buddy One 上线（2026-10-29）— https://www.chinaz.com/2026/1029/1785065.shtml
- 腾讯办公套件 Buddy One + 腾讯网盘 — https://www.iheima.com/article-403588.html
- 百度 GenFlow 4.0（2026-04-28）— https://www.stdaily.com/web/gdxw/2026-04/28/content_509368.html
- 百度「库库 AI」独立端（2026-08-14）— http://imgs.xinhuanet.com/tech/20260814/b957763bbe184d11a3dbc9261e599f86/c.html
- 金山办公灵犀专业版 + WPS Comate（2026-07-17）— http://www.xinhuanet.com/finance/20260717/006e0852cda042b880fd029554947189/c.html
- 金山办公 Q3 净利降逾七成、月活 6.78 亿（2026-10-27）— https://finance.eastmoney.com/a/202610273910248927.html
- 办公 Agent 五强横评（雷科技）— https://www.leikeji.com/article/78887
- 五款 Agent 办公实测（36Kr）— https://www.36kr.com/p/3868529576655491
- 大厂抢滩 AI 办公（钛媒体）— https://m.tmtpost.com/8091801.html
- AI 办公「三国杀」（DoNews）— https://www.donews.com/article/detail/5331/108944.html
- QuestMobile 三强格局报告 — https://www.questmobile.cn/research/report/2094686886299705346

**个人 vs 企业 / 变现**

- 钛媒体：个人 Agent 与企业 Agent 定位分化 — https://www.tmtpost.com/8157089.html
- 钛媒体：个人与企业研发 Agent，功能差在哪 — https://m.tmtpost.com/8167899.html
- 融资 13 亿，前豆包负责人做 AI 助理但不好赚钱（2026-10-27）— https://view.inews.qq.com/a/20261027A0D7Y600
- 个人 Agent 创业潮只适合大厂（证券之星 2026-10-26）— https://wap.stockstar.com/detail/IG2026102600023186
- 有用户、有收入，AI 应用却不是好生意 — https://www.itiger.com/hans/news/2664325886
- Kuse 毛利微正、Cursor 毛利率 -23%（观点网）— https://m.guandian.cn/m/show/596628
- Manus 被 AI 大厂重新定价（钛媒体 2026-10-12）— https://m.tmtpost.com/8164195.html
- Manus 40 亿美元估值（EEPW 2026-10）— https://m.eepw.com.cn/article/202610/483950.html
- Manus / Genspark 收入半年翻倍（投资界）— https://m.pedaily.cn/news/565316
- 中国 AI 办公智能体 2026 年 449 亿元（观点网 2026-08-25）— https://www.guandian.cn/m/show/590648

**造价 / 工程**

- 广联达造价全场景 AI，土建提量 98%（2026-10-14）— https://www.guandian.cn/article/20261014/613362.html
- 广联达造价 AI 新品（中国建设报 2026-10-20）— https://www.chinajsb.cn/html/202610/20/60437.html
- 广联达数字造价市占率超 90%（三个皮匠）— https://www.sgpjbg.com/labelsyh/guangliandaxingyediwei/1/7075991.html
- 广联达 2025 年净利 +61.77% — https://www.stcn.com/article/detail/3692943.html
- 6 款造价 AI 工具实测：免费的那款最贵 — https://m.toutiao.com/article/7680709754436583988/
- 22 天变 4 小时：6 款 AI 算量横评 — http://mp.weixin.qq.com/s?__biz=MzIyMTcwNzA5Ng==&mid=2247485061
- AI 正在吃掉造价基础岗 — https://www.fwxgx.com/articles/247892
- 造价行业分水岭：手工算量淘汰 — https://m.fwxgx.com/articles/247998
- 鲁班 LTJ2026 智能土建算量平台 — https://www.luban.cn/LTJ
- Togal.AI PitchBook 档案 — https://pitchbook.com/profiles/company/433187-11
- Togal.AI 定价评测（Struvia）— https://struvia.co/blog/togal-ai-pricing
- Exayard 订阅套餐 — https://exayard.com/help/zh-CN/articles/14033075
- CostFlowAI Pro 定价 — https://costflowai.com/pricing/
- 利驰 AI 识图报价一体化（签约快报 2026-10-28）— http://www.leadsoft.com.cn/index.php?id=4773
- 斑马进度 AI 智能解决方案（2026-04-17）— http://stock.10jqka.com.cn/20260417/c676067869.shtml
- 斑马进度产品页 — https://www.glodon.cn/product/204.html
- Primavera P6 定价（2026）— https://greenwolftechlabs.com/primavera-p6-pricing/
- CAD 快速看图 VIP — https://cad.glodon.com/payment/vip/privatetraffic

**写作**

- NovelCrafter 定价 — https://www.novelcrafter.com/pricing
- NovelCrafter 2026 实际花费拆解 — https://novelmage.com/blog/novelcrafter-pricing-in-2026-what-you-actually-pay
- Sudowrite 定价（CostBench 2026）— https://costbench.com/software/ai-writing-tools/sudowrite/
- Sudowrite 定价（CompareEdge）— https://comparedge.com/tools/sudowrite/pricing
- 阅文上半年 24 万新作家、收入 +10.7%（2026-08-12）— http://sh.people.com.cn/BIG5/n2/2026/0812/c350122-41666308.html
- 阅文作家专属 AI「Claw」（2026-03）— https://kw.beijing.gov.cn/xwdt/kcyx/xwdtyqqy/202603/t20260316_4557563.html
- 阅文 Buddy 系列智能体 — https://www.cnstock.com/commonDetail/749807
- 番茄一个月下架 4 万本书、封 855 个号、67% 作者用 AI — https://m.toutiao.com/article/7655282662768116234/
- 85% 新人一年稿费不够买手机 — https://m.toutiao.com/article/7624761751334175259/
- 彩云小梦还能用吗（第三方，2026）— https://foreverse.cn/zh/blog/after-caiyun-xiaomeng

**本地 / 隐私 / 硬件**

- 7 款手机端侧生成式 AI 完成备案（2026-07）— https://www.stcn.com/article/detail/4021861.html
- 手机端侧 AI 备案公示解读 — https://paper.cnstock.com/html/2026-07/16/content_2245321.htm
- 《人工智能拟人化互动服务管理暂行办法》— https://policy.mofcom.gov.cn/claw/clawContent.shtml?id=105540
- IDC：AI 主机 2030 年 770 亿元（2026-09）— https://m.cnfin.com/hg-lb//zixun/20260910/4468311_1.html
- AI 手机之后，AI 主机也出现了（2026-09-19）— https://www.sfccn.com/2026/9-19/zMMDE0NDlfMjI0NjMzMg.html
- 大模型进不了内网之后（钛媒体）— https://www.tmtpost.com/8070771.html
- Apple Mac 营收与本地 AI 需求 — https://www.digitaltoday.co.kr/cn/view/52416
- OpenHuman 4 个月 31K stars — https://cloud.tencent.com.cn/developer/article/2686137
- OpenHuman 越用越后怕（钛媒体）— https://m.tmtpost.com/7992192.html

**失败 / 降温**

- 豆包、千问下线智能体功能引发争议（虎嗅）— https://www.huxiu.com/article/4874364.html
- 豆包、千问挥刀砍向数字生命（钛媒体）— https://m.tmtpost.com/8060421.html
- 老板花 50 万搞 AI Agent，上线一周就关了 — https://dbaplus.cn/news-73-7450-1.html
- AI 陪伴赛道顶流陨落（界面 2026-10）— https://www.jiemian.com/article/15199166.html
- 倒闭的 AI 伴侣 App 记录 — https://pocketanimus.com/zh/insights/dead-ai-companion-apps/

---

## 附录：存疑与冲突清单（引用前必须处理）

| # | 冲突点 | 说法 A | 说法 B | 处理建议 |
|---|---|---|---|---|
| 1 | **两处归属纠偏（最需注意）** | 任务书写「阿里 灵犀/夸克」与「字节 WorkBuddy」 | 公开信息 + gaea 项目记忆一致指向：**灵犀 = 金山办公（WPS）的 AI 原生办公产品**（2026-07-17 灵犀专业版、2026-10-27 新版），阿里侧对应「千问办公 / 夸克」；**WorkBuddy = 腾讯云 CodeBuddy 系的全场景 AI 办公工作台**，QuestMobile 榜单亦标注「腾讯 WorkBuddy」（3235.3 万 MAU） | 本底稿已按纠正值写入矩阵（第 9 行、第 15 行）；「字节」一侧只有豆包/扣子，**无独立办公 Agent 单品数据**；**V5 立项文档必须沿用纠正后的归属** |
| 2 | Google AI Ultra 价格 | Digital Today：**$250 → $100** | c114：**最高阶 Ultra 降至 $200** | 两个数字并存；引用时写「$100–$200 区间（口径不一）」，或标注具体来源 |
| 3 | Sudowrite 价格 | CostBench：4 档 **$10–$59/月** | CompareEdge：Hobby/Student **$19 起** | 写「$10–$59（4 档）」，并注明起价口径不一 |
| 4 | 彩云小梦状态 | 第三方文章讨论「还能用吗」 | 无官方停服公告可查 | 标 **C / 待核实**，不写「已停服」 |
| 5 | 造价 98% 准确率 | 厂商发布口径 | 无第三方复现 | 必须写「厂商口径」，不可当作行业事实 |
| 6 | 「22 天 → 4 小时」 | 自媒体实测 | 样本仅 1 套图纸、6 款工具 | 写「单一实测、非统计结论」 |
| 7 | 企业级 Agent「220 亿美元」 | 葡语媒体转述行业报告 | 未见原报告 | 标 C，或直接删除 |
| 8 | 斑马「履约率 23%」 | 自媒体标题 | 未说明统计口径 | 引用时注明「标题级、口径不明」 |
| 9 | 广联达「3000 亿蓝海」 | 媒体标题（建筑信息化） | 非造价软件规模 | 不可当作造价软件 TAM |
| 10 | 微软 Copilot Cowork | Digital Today 报道「与 Anthropic 合作开发」 | 微软/Anthropic 官方未确认 | 标「报道口径」 |

---

*本文件由调研子代理生成于 2026-10-29，UTF-8 无 BOM。数字均可回溯至第六节链接；未取到数值处一律留空并标注，未做任何推算或补齐。*
