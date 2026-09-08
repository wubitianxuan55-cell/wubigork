# ClickUp Brain（Brain²）调研原始稿

> 调研日期：2026-09-06。定位：ClickUp 全家桶 AI 层，2026 年升级为「Brain²」，宣传「one AI to replace them all」，把任务/文档/聊天/日历/连接应用全部作为其上下文。

## 来源 URL
- 官方产品页：https://clickup.com/brain
- 官方使用指南：https://clickup.com/blog/how-to-use-clickup-ai/
- AI 站会状态更新：https://clickup.com/blog/ai-powered-status-updates-standups/
- 数据→洞察：https://clickup.com/blog/clickup-brain-actionable-insights/
- 第三方评测：https://agent-finder.co/reviews/clickup-brain ；https://www.upsys-consulting.com/en/blog/maximize-productivity-with-clickup-ai ；https://www.morgen.so/blog-posts/clickup-review
- 对比：https://tech-insider.org/notion-vs-clickup-2026/

## 能力清单（AI 能做什么）
- 生成类：AI Tasks（结构化工作单元，携带状态/负责人/依赖）；一键生成任务+子任务+描述；生成项目更新、周报、报告/幻灯/仪表盘/页面/图表/表格甚至代码。
- 分析/问答类：Deep Search / Enterprise Search 跨 workspace+应用+web 找答案；Recall 混合向量+图检索全历史；跨任务/文档/评论的 AI 摘要（评测称每次站会省 15-20 分钟）。
- 调整类：Brain 可创建任务、评论、指派、改状态、更新数据库条目（活动流演示）；角色化 agents（Project/Ops/Finance 等）执行端到端事项。
- 陪伴类：Super Agents / Ambient Agents（环境感知代理，主动给上下文与建议）；Notetaker 入会记笔记；「knows your schedule」日历感知，偏好设定如「别把会议排到 10 点前」。
- **未查到**：对甘特图/关键路径/工期的对话式重排能力（ClickUp 有甘特视图但 AI 层未见 CPM 级操作）。

## 交互形态
- AI Chats（线程化对话）、AI Channels（话题频道）、@Brain 在任意位置提及唤起、语音输入、桌面/移动/浏览器插件。
- 嵌入式：任务/文档/数据库内直接 AI 生成与改写。
- Ambient：无需提问，主动浮出相关上下文与建议。

## 数据流（读什么/写什么）
- 读：任务、文档、聊天、日历、邮件、连接应用（Google Drive/GitHub/Salesforce/Figma/Slack/Gmail/Outlook/Linear/Notion）+ 任意 MCP。
- 写：创建/评论/指派任务、改状态、更新数据库、生成报告/仪表盘/页面等。
- 计费：Brain² $9/用户/月（1500 AI Super Credits，对话不耗 credit）；Everything AI $28/用户/月。模型层用 ChatGPT/Claude/Gemini 多模型。

## 可蒸馏点（对刀4/刀5）
1. **「AI Tasks 携带状态/负责人/依赖」**的字段化生成思路：AI 产物不是一段文本而是结构化对象——与 gaea schedule_apply 接收结构化任务/搭接字段完全同构。
2. **@Brain 任意位置唤起 + Ambient 主动建议**：对话不必只在一个聊天面板；在甘特图上选中任务可直接唤起 AI 调整，环境级建议（如「检测到关键路径变化」）作为低打扰入口。
3. **多模型+credit 计费**证明「对话免费/动作计费」的产品化路径可行（对 gaea 定价有参考）。
4. Deep Search 的「workspace+应用+web」三级检索范围划分，可映射 gaea 的「计划文件/workspace/用户上传」三级上下文。

## 不可取点
1. 能力广而浅：宣传覆盖 12 类产物（报告到代码），但**没有排程正确性背书**——生成依赖/工期无引擎校验。gaea 反而应强调「AI 产出必过 CPM 引擎」。
2. Ambient Agents 主动改任务状态/数据库，页面上未见确认门槛描述——自主写操作的信任风险。
3. 品牌改名频繁（ClickUp Brain→Brain²），功能口径漂移，第三方评测多依赖官方口径。
