# 7. 轻语 · 人格陪伴（v4.11.0 基线复扫 · 2026-08-31）

> 调研方法说明：本节基于 2026-08-31 的公开网页检索（TechCrunch、网信办官网、维基百科、aicpb 榜单、厂商官网等）；中文搜索引擎结果受合规过滤与分词影响，部分国内产品细节标注「未核实」。

## 市场格局 · 最新动态

**中国：监管落锤是本周期最大变量。** 国家网信办等五部门 2026-04-10 发布《人工智能拟人化互动服务管理暂行办法》（令第 21 号），**2026-07-15 起施行**：禁止向未成年人提供「虚拟伴侣等虚拟亲密关系服务」；不得以诱导情感依赖、沉迷为服务目标；出现自残自杀迹象须干预并联络监护人/紧急联系人；连续使用每超 2 小时提醒；敏感个人信息用于训练须「单独同意」；注册超 100 万或月活超 10 万须安全评估并报省级网信，算法备案+年度核验，罚款上限 10–20 万元（[网信办全文](https://www.cac.gov.cn/2026-04/10/c_1777558395078289.htm)）。其征求意见稿于 2025-12-27 发布；据 21 世纪经济报道，2025 年 11 月国内头部月活为：星野 488 万、猫箱 472 万、X EVA 181 万、筑梦岛约 60 万，星野+Talkie 前 9 个月收入约 1.2 亿元；筑梦岛 2025 年 6 月因低俗内容被上海网信办约谈（[SFCCN](https://m.sfccn.com/2025/12-29/xNMDE0NDlfMjA5MjQxNw.html)）。施行后国内平台整改的公开报道**未核实**。

**海外：诉讼与未成年人禁入重塑产品形态。** Character.AI 于 2025-10-29 宣布自 11-25 起禁止未满 18 岁用户使用开放聊天，未成年人改用互动「Stories」（[TechCrunch](https://techcrunch.com/2025/11/25/character-ai-will-offer-interactive-stories-to-kids-instead-of-open-ended-chat/)、[Wikipedia](https://en.wikipedia.org/wiki/Character.AI)）；2026-01 Google 与 Character.AI 就青少年死亡案启动首批和解谈判（[TechCrunch](https://techcrunch.com/2026/01/07/google-and-character-ai-negotiate-first-major-settlements-in-teen-chatbot-death-cases/)）；2026-05 宾州医务委员会因聊天机器人冒充医生起诉（[TechCrunch](https://techcrunch.com/2026/05/05/pennsylvania-sues-character-ai-after-a-chatbot-allegedly-posed-as-a-doctor/)）。海外 web 流量榜（2026-06）：Character.AI 1.94 亿居首，JanitorAI 1.39 亿（+14.9%）快速逼近，成人向的 SpicyChat/polybuzz/Candy.ai 占据前五多数席位，Talkie 仅 652 万且环比下滑（[aicpb](https://aicpb.com/ai-rankings/products/ai-character-rankings)）——纯陪伴 web 端明显「灰度化、成人化」。

## 范式迁移（上轮调研以来的变化）

1. **从「沉浸陪伴」转向「内容+陪伴」：** Character.AI 2026 年上半年人均月使用超 950 分钟（Sensor Tower 数据，[TechCrunch](https://techcrunch.com/2026/07/09/character-ai-enters-the-microdrama-arena-with-its-own-productions-but-with-a-twist/)），7 月推出 AI 微短剧「c.ai Series」（18+ 观众可边看边与角色对话）、音频系列 c.ai FM、写作工具 c.ai Reads，并陆续上线 Lorebook/Books——头部平台在把自己变成「可对话的内容厂」而非纯聊天。
2. **资本面进入整合期：** MiniMax 2026-01-09 港股上市（SEHK:100），2025 年收入 7900 万美元、净亏 18.7 亿美元（[Wikipedia](https://en.wikipedia.org/wiki/MiniMax_(company))）；个性化陪伴应用 Dot 于 2025-09 关停（[TechCrunch](https://techcrunch.com/2025/09/05/personalized-ai-companion-app-dot-is-shutting-down/)）；Replika 2025 年换帅、用户超 4000 万，创始人 Kuyda 转做新公司（[Wikipedia](https://en.wikipedia.org/wiki/Replika)）。全球 AI 陪伴应用 2025 年内购收入约 1.2 亿美元（[TechCrunch](https://techcrunch.com/2025/08/12/ai-companion-apps-on-track-to-pull-in-120m-in-2025/)）；2026 上半年全球收入口径**未核实**。
3. **「长期记忆+主动关心+情感语音」在海外已成卖点组合：** Nomi 官网明确主打「Human-Level Memory（短/中/长期记忆）」「Proactive Nomi Messaging（用户离开一段时间后主动发消息）」「Emotive Voice Chats（情绪化语音通话，语调随情绪变化）」（[nomi.ai](https://nomi.ai/)）——这正是轻语规划中的三件事，海外独立厂商已产品化，但属厂商自述，实际效果口碑**未核实**。国内星野/猫箱 2026 年在记忆与主动关心上的功能更新**未核实**（其官网仍以移动端「沉浸式智能体社区」为主打，[xingyeai.com](https://www.xingyeai.com/)）。

## 对 gaea 的机会与威胁

**机会：**
- 《暂行办法》适用「面向境内公众提供」的服务，gaea 个人本地使用、不面向公众运营、无内置氪金，**天然落在适用范围之外**；而办法让云端陪伴产品背上防沉迷、单独同意、备案评估等成本，反向放大「数据不出机 + 无氪金 + 硬隔离」的差异化价值。
- 未成年人被整体排除出虚拟伴侣后，成年敏感用户对「不上传聊天记录、可随时彻底删除」的需求上升，本地记忆管线从卖点升级为刚需；SillyTavern（32.8k GitHub stars，纯本地、AGPL，「不提供任何在线服务」，[GitHub](https://github.com/SillyTavern/SillyTavern)）证明本地陪伴存在真实但偏极客的需求——**大众级桌面本地陪伴产品仍未发现**，「是否伪需求」无反证，但大众化证据也**未核实**。
- 头部转向内容化与灰度化后，「长期连续人格 + 看得见你的真实生活」的冷静陪伴是空位。

**威胁：**
- 若轻语未来做人格包社区共享、公开分发，可能触发办法中的应用商店责任与安全评估条款；「主动关心」若做成持续打扰，与立法要求的「便捷退出、不得以持续互动阻碍退出」精神冲突，需克制设计。
- 海外免费强产品（Character.AI 950 分钟/月黏性、Nomi 三件套）拉高了用户对记忆与语音的预期，轻语的情感语音若滞后太久会失去宣称空间。

## 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **0-3 月：** 补齐合规护栏与产品叙事——在轻语界面提供「非真人提示」「会话一键清空/导出删除」「连续时长提醒」开关（与办法条款对齐，即使不适用也构成卖点）；把「乐园/工位硬隔离 + 本地记忆」写进轻语首屏文案，对标 Nomi 的 memory/proactive/voice 三件套做差异化表。
- **下个 3-6 月：** 落地关系记忆图谱与克制的主动关心（仅限用户开启、频次封顶、一键静音，避开「诱导依赖」红线）；上线情绪化 TTS 语音（复用节奏引擎的 PAD 标尺驱动语调），参考 Nomi「语调随情绪变化」的表述做验收标准。
- **愿景 6-12 月：** 探索「陪伴×真实生活」的独占位（陪伴人格可感知你的工作文档完成度并温和追问，这是纯娱乐云产品做不到的）；暂缓人格包公开分享等可能触发平台责任的功能，观察《暂行办法》施行后首批执法与国内头部整改动作再定。

## 参考来源

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
