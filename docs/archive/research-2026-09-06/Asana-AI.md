# Asana AI（Asana Intelligence / AI Studio）调研原始稿

> 调研日期：2026-09-06。定位：从「copilot 功能集」升级为「agentic work management」平台：预置 AI 队友 + 无代码 AI Studio + 外部 LLM 连接器。

## 来源 URL
- 官方 AI 产品页：https://asana.com/product/ai
- Smart Status 官方帮助：https://help.asana.com/s/article/smart-status
- Asana AI 智能功能入门（官方）：https://help.asana.com/s/article/get-started-with-asana-ai
- 论坛讨论（Smart Status）：https://forum.asana.com/t/automate-project-status-updates-with-the-asana-intelligence-smart-status-feature/552005
- 第三方深度解读：https://cloudfresh.com/en/blog/asana-ai/ ；https://firebearstudio.com/blog/what-is-asana.html ；https://cirface.com/blog/asana-intelligence-project-management

## 能力清单（AI 能做什么）
- **AI Teammates**：30 个预置 AI 代理（市场/运营/IT 等），「no prompt engineering needed」，预授权嵌入工作流，在任务/工作流内完成复杂工作。
- **AI Studio**：无代码构建自动化——intake（请求收集）、routing（分派）、updates（更新）；把 AI 接进业务流程而非聊天窗口。
- **Asana Dash**：「AI Chief of Staff」，每天早晨从会议/邮件/任务中汇总优先级，给 next best actions。
- **Smart Status**：分析项目实时数据自动起草状态更新，标出 on-track/off-track、风险、盲点与开放问题（论坛称「project GPS」）。
- **AI Connectors & MCP**：让 ChatGPT/Claude/Gemini 直连 Work Graph，可「search, create, update, organize」Asana 工作项——外部 agent 成为操作端。
- **AI 项目计划模板**：组织任务/负责人/里程碑/截止日的模板资源；**AI 日历**：围绕团队优先级与产能排任务。
- 更早一代功能（Smart Goals / Smart Answers / AI 摘要 / 收件箱摘要）仍在，但页面叙事重心已转向 agents。

## 交互形态
- 嵌入式（任务/项目内 AI 按钮）、对话式（Dash 每日简报、外部 LLM 对话）、自动化（AI Studio 触发器+规则）三层并存。
- 明确口号「No prompt engineering needed」——交互被模板化/预置化，把 prompt 责任从用户移到产品。

## 数据流（读什么/写什么）
- 读：Work Graph 全量（任务/项目/目标/评论/会议/邮件），遵守 Asana 权限模型——「AI 只能访问用户已能看到的内容」。
- 写：路由分派、任务更新、状态报告草稿；外部连接器可 create/update 工作项。
- 信任条款：AI 合作方不用客户数据训练；每次查询后数据删除；合作方服务器限美国。

## 许可与定价
- Asana AI：所有付费版可用；AI Studio：Starter 及以上，需管理员在 admin console 启用。

## 可蒸馏点（对刀4/刀5）
1. **「AI 只能看用户能看的」权限继承原则**——gaea agent 工具三件套应绑定计划文件所在 workspace 的用户权限，schedule_get 返回范围与人类打开计划一致。
2. **Smart Status 的产出结构**（on-track/off-track/风险/盲点/开放问题）可作为 schedule_analyze 报告的段落模板——AI 解读关键线路时按「健康面/落后面/风险/待决问题」四段输出。
3. **MCP/连接器让外部 LLM 直接 create/update**：刀4 的 agent 工具三件套本质就是 gaea 版 MCP server；Asana 验证了「把图数据结构暴露给任意 agent」这条路。
4. 「No prompt engineering needed」的预置化交互（点按钮而非写提示词）适合建筑用户群：把「压 10 天」「导出周报」做成预置动作。

## 不可取点
1. 各功能分散（产品页与帮助文档口径不一致），用户认知成本高；gaea 应把三件套收拢在计划场景内。
2. AI Teammates 预授权直接写工作流，官方页未给确认门槛——对任务型数据可接受，对进度计划（牵一发动全身的重算）必须有 diff。
3. AI 日历「围绕产能排任务」没有公开任何排程算法/约束口径说明，黑箱。
