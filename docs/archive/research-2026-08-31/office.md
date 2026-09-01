# 市场调研复扫 · 通用办公

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
