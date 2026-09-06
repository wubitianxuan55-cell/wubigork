# Notion AI（Agent / Custom Agents / Autofill）调研原始稿

> 调研日期：2026-09-06。定位：AI 优先工作区，2026 年主页口号「where teams and agents create together」；没有排程引擎，但代表了「agent 直接操作数据库」的通用范式。

## 来源 URL
- 官方 AI 产品页：https://www.notion.com/product/ai
- 官网主页：https://www.notion.com/
- 数据库 AI Autofill 帮助：https://www.notion.com/help/autofill
- AI Meeting Notes 指南：https://www.notion.com/help/guides/preserve-perfect-meeting-memory-with-ai-meeting-notes
- 第三方实操：https://www.eesel.ai/blog/notion-ai-action-items-extraction ；https://aiweekly.co/learning-ai/ai-applications/how-to-use-notion-ai ；https://thecreatorsai.com/p/how-notion-ai-helps-you-automate

## 能力清单（AI 能做什么）
- **Notion Agent**：聊天式接受多步任务，用 Notion 内+连接应用+web 的上下文，直接创建/编辑页面与数据库。
- **Custom Agents**：触发器/定时驱动的自治代理，7x24 做 Slack 回答、任务分派、项目更新；管理员可禁用、可控制创建权限。
- **Enterprise Search**：跨应用秒级问答（只读）。
- **AI Connectors**：Slack/Google Drive/GitHub/Asana 等只读连接喂给搜索。
- **AI Meeting Notes**：无 bot 转写+摘要+行动项，可挂到数据库模板自动用于例会/站会。
- **Database Autofill**：AI 自动填充数据库属性（含 Custom Autofill，如从会议库抽取行动项）。
- **写作/AI Blocks**：页内改写/生成草稿。
- **未查到**：任何甘特/依赖/关键路径级 AI 能力（Notion 时间线视图是纯手动）。

## 交互形态
- 聊天（Agent）、自治（Custom Agents）、嵌入式（页内 AI blocks、数据库属性）、搜索框（Enterprise Search）。
- Custom Agents 按「trigger or schedule」运行——AI 被放进时间轴（定时任务），不只是等用户问。

## 数据流（读什么/写什么）
- 读：workspace 全内容+连接应用+web。
- 写：Agent 直接创建/编辑页面与数据库；Autofill 写属性值；Custom Agents 发更新、派任务。
- 官方产品页对**所有功能均未描述 preview/确认步骤**。

## 定价与计费
- Business/Enterprise 套餐含 AI；免费版试用。
- Custom Agents：免费至 2026-05-03，之后 $10 / 1000 credits；credit 耗尽代理暂停；用量过大可能被限流——「按动作计费+额度熔断」的 agent 商业化样板。

## 可蒸馏点（对刀4/刀5）
1. **Autofill 的「属性级 AI 写入」粒度**：AI 不整页重写，只填指定属性——对应 gaea 的 schedule_apply 粒度设计（按任务字段 apply，而非整计划重写）。
2. **trigger/schedule 型 agent**：gaea 刀5 之外可展望「每天早上 AI 检查计划文件→生成风险日报」，Notion 证明了定时 agent + credit 计费的产品形态。
3. 数据库模板挂 AI（例会自动记行动项）的「模板化 AI」分发方式，可映射到 gaea 的进度计划模板+AI 预检组合。
4. credit 熔断（额度耗尽代理自动暂停）是 agent 滥用的安全阀设计。

## 不可取点
1. 全线无 preview/确认流，Agent 直接改库——文档域可容忍，进度计划域不可。
2. 无结构化校验：AI 写入的属性值没有 schema/类型约束的产品叙事，脏数据风险转嫁用户。
3. 计费模型复杂（套餐+credits+限流三套并行），中小团队理解成本高。
