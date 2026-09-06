# monday.com AI（AI Agents / monday magic）调研原始稿（补充样本）

> 调研日期：2026-09-06。定位：指定竞品清单之外的补充样本：其 agent 叙事（自动盯进度/写周报/行动项转化）正是刀5 之后「agent-native PM」的形态样本。信息以搜索摘要级为主，未逐页深挖。

## 来源 URL
- AI agents 用例（官方博客）：https://monday.com/blog/ai-agents/how-project-management-teams-use-monday-ai-agents
- AI agents PM 综述（Dust）：https://dust.tt/blog/ai-for-project-management
- 2026 趋势综述（Epicflow）：https://www.epicflow.com/blog/ai-agents-for-project-management/
- Atlassian 综述：https://www.atlassian.com/agile/project-management/ai-agents
- Agentic AI PM 观察（LinkedIn）：https://www.linkedin.com/pulse/agentic-ai-project-management-what-thrvc

## 能力清单（AI 能做什么）
- agents 自主处理项目执行：**生成状态报告、标记风险、把会议纪要转为被跟踪的行动项**。
- 综述口径（2026）：agent 盯项目→发现 deadline 滑动→发提醒/改派/更新计划——甚至「without human intervention」（LinkedIn 观察口径）。
- Epicflow 口径：持续跟踪进度、预测瓶颈、给建议。
- **未查到**：CPM 级排程调整能力；对日历/依赖的写操作细节。

## 交互形态
- 自治 agent（触发/定时）+ 报告产物回流；Atlassian 口径强调「跨会话/会议跟踪行动项」的后台协调角色。

## 数据流（读什么/写什么）
- 读：项目数据、会话/会议记录、跨工具数据。
- 写：状态报告、风险标记、行动项、（激进口径下）任务改派与计划更新。

## 可蒸馏点（对刀4/刀5）
1. **「周报/状态报告是 agent 的第一个大规模落地物」**（monday/Epicflow/ClickUp/Asana 四家口径一致）：gaea 的 schedule_analyze 第一个产品化输出应是「每周进度风险日报」（关键路径变化+时差消耗+延误预警），确定性数据+AI 叙事。
2. 「会议纪要→行动项→计划任务」链路提示 gaea 的 agent 可从办公域（gaea 本就是办公 AI 助手）把会议/文档内容写入进度计划——跨板块协同是 gaea 独有优势，竞品做不了。
3. 行业综述共识的分工边界：AI 盯+预警+建议，人决策+改关键计划；「without human intervention」口径仍属营销叙事。

## 不可取点
1. 自治改计划（reassign/update plan without human intervention）无确认门槛口径，与工程进度计划的严肃性冲突。
2. 本文件基于搜索摘要，功能细节未经官方文档逐条核实，引用时注意标注置信度。
